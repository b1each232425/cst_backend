package practice

//annotation:practice-service
//author:{"name":"zoudelun","tel":"15920422045", "email":"1311866870@qq.com"}

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx/types"
	"io"
	"strconv"
	"strings"
	"time"
	"w2w.io/null"

	"go.uber.org/zap"
	"w2w.io/cmn"
)

var z *zap.Logger

func init() {
	//Setup package scope variables, just like logger, db connector, configure parameters, etc.
	cmn.PackageStarters = append(cmn.PackageStarters, func() {
		z = cmn.GetLogger()
		z.Info("message zLogger settled")
	})
}

func Enroll(author string) {
	z.Info("message.Enroll called")
	var developer *cmn.ModuleAuthor
	if author != "" {
		var d cmn.ModuleAuthor
		err := json.Unmarshal([]byte(author), &d)
		if err != nil {
			z.Error(err.Error())
			return
		}
		developer = &d
	}

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: practiceH,

		Path: "/practice",
		Name: "practice",

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: practiceStudentH,

		Path: "/practiceStudent",
		Name: "practiceStudent",

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})

}

var PracticeStatus = struct {
	PendingRelease string // 未发布 00
	Released       string // 已发布 02
	Deleted        string // 已删除 04
}{
	PendingRelease: "00",
	Released:       "02",
	Deleted:        "04",
}
var PracticeStudentStatus = struct {
	Normal  string // 正常 00
	Deleted string // 被删除 02
}{
	Normal:  "00",
	Deleted: "02",
}

type practiceInfo struct {
	Practice cmn.TPractice          `json:"practice"`
	Student  []cmn.TPracticeStudent `json:"student"`
}

func practiceH(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	method := strings.ToLower(q.R.Method)
	ctx, cancel := context.WithTimeout(q.R.Context(), 5*time.Second)
	defer cancel()
	switch method {
	case "post":
		{
			var buf []byte
			buf, q.Err = io.ReadAll(q.R.Body)
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			defer func() {
				q.Err = q.R.Body.Close()
				if q.Err != nil {
					z.Error(q.Err.Error())
					q.RespErr()
					return
				}
			}()
			if len(buf) == 0 {
				q.Err = fmt.Errorf("Call /api/upLogin with  empty body")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			var p practiceInfo
			q.Err = json.Unmarshal(buf, &p)
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			err := UpsertPractice(ctx, &p.Practice, p.Student, 1)
			if err != nil {
				q.Err = err
				q.RespErr()
				return
			}
			data, err := json.Marshal(p.Practice)
			if err != nil {
				z.Error(err.Error())
				q.Err = err
				q.RespErr()
				return
			}
			q.Msg.Data = data
			z.Info("---->" + cmn.FncName())
			q.Msg.Msg = "OK"
			q.Resp()
			return
		}
	case "get":
		{
			name := q.R.URL.Query().Get("name")
			pType := q.R.URL.Query().Get("type")
			status := q.R.URL.Query().Get("status")

			pageStr := q.R.URL.Query().Get("page")
			if pageStr == "" {
				q.Err = fmt.Errorf("缺失分页查询页号")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			var page int
			page, q.Err = strconv.Atoi(pageStr)
			if q.Err != nil {
				q.Err = fmt.Errorf("分页查询的页号解析失败：%v", q.Err.Error())
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			pageSizeStr := q.R.URL.Query().Get("page_size")
			if pageSizeStr == "" {
				q.Err = fmt.Errorf("缺失分页查询页号")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			var pageSize int
			pageSize, q.Err = strconv.Atoi(pageSizeStr)
			if q.Err != nil {
				q.Err = fmt.Errorf("分页页大小解析失败：%v", q.Err.Error())
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			// 排序字段
			orderBy := []string{"create_time desc"}
			p, total, err := ListPractice(ctx, name, pType, status, orderBy, page, pageSize, 1)
			if err != nil {
				q.Err = err
				q.RespErr()
				return
			}
			result := Map{
				"total":     total,
				"practices": p,
			}
			data, err := json.Marshal(result)
			if err != nil {
				z.Error(err.Error())
				q.Err = err
				q.RespErr()
				return
			}
			q.Msg.Data = data
			z.Info("---->" + cmn.FncName())
			q.Msg.Msg = "OK"
			q.Resp()
			return
		}
	case "patch":
		{
			// 更新练习状态，对练习进行发布、取消操作
			idStr := q.R.URL.Query().Get("id")
			status := q.R.URL.Query().Get("status")

			var id int64
			id, q.Err = strconv.ParseInt(idStr, 10, 64)
			if q.Err != nil {
				q.Err = fmt.Errorf("分页页大小解析失败：%v", q.Err.Error())
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			// TODO 处理练习的发布状态 生成练习考卷
			q.Err = OperatePracticeStatus(ctx, id, status, 1)
			if q.Err != nil {
				q.RespErr()
				return
			}
			z.Info("---->" + cmn.FncName())
			q.Msg.Status = 0
			q.Msg.Msg = "OK"
			q.Resp()
			return

		}
	default:
		q.Err = fmt.Errorf("please call /api/Practice with  http POST/GET/PATCH method")
		q.RespErr()
		return
	}

}

// 处理新增/编辑学生练习名单
func practiceStudentH(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	testData := []cmn.TPracticeStudent{
		{
			PracticeID: null.IntFrom(1),
			StudentID:  null.IntFrom(10001),
			Status:     null.StringFrom("00"),
			Creator:    null.IntFrom(1001),
			UpdatedBy:  null.IntFrom(1001),
			CreateTime: null.IntFrom(12),
			UpdateTime: null.IntFrom(13),
			Addi:       types.JSONText(`{"note":"测试学生1"}`),
		},
		{
			PracticeID: null.IntFrom(2),
			StudentID:  null.IntFrom(10002),
			Status:     null.StringFrom("00"),
			Creator:    null.IntFrom(1001),
			UpdatedBy:  null.IntFrom(1001),
			CreateTime: null.IntFrom(12),
			UpdateTime: null.IntFrom(14),
			Addi:       types.JSONText(`{"note":"测试学生2"}`),
		},
	}
	err := UpsertPracticeStudent(context.Background(), 1, 1, testData)
	if err != nil {
		q.Err = err
		q.RespErr()
		return
	}
	q.Resp()
}
