package practice_mgt

//annotation:practice-service
//author:{"name":"ZouDeLun","tel":"15920422045", "email":"1311866870@qq.com"}

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"io"
	"strconv"
	"strings"
	"time"
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
		Fn: practiceTH,

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

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: practiceStudentListH,

		Path: "/practiceStudentList",
		Name: "practiceStudentList",

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: practiceSH,

		Path: "/practiceS",
		Name: "practiceS",

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})

}

func practiceTH(ctx context.Context) {
	TEMPUID := int64(1)
	q := cmn.GetCtxValue(ctx)
	// TODO 对接用户管理的令牌功能
	//userID, _ := q.Session.Values["ID"].(int64)
	//if userID <= 0 {
	//	q.Err = fmt.Errorf("invalid session")
	//	z.Error(q.Err.Error())
	//	return
	//}
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
			err := UpsertPractice(ctx, &p.Practice, p.Student, TEMPUID)
			if err != nil {
				q.Err = err
				q.RespErr()
				return
			}
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
			id := q.R.URL.Query().Get("id")
			// 代表此时进入到编辑页面了
			if id != "" {
				var pid int64
				pid, q.Err = strconv.ParseInt(id, 10, 64)
				if q.Err != nil {
					q.Err = fmt.Errorf("练习ID获取失败：%v", q.Err.Error())
					z.Error(q.Err.Error())
					q.RespErr()
					return
				}
				p, pName, sCount, err := LoadPracticeById(ctx, pid)
				if err != nil {
					q.Err = err
					q.RespErr()
					return
				}
				result := Map{
					"practice":      p,
					"student_count": sCount,
					"paper_name":    pName,
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
				q.Err = fmt.Errorf("缺失分页查询页大小")
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
			p, total, err := ListPracticeT(ctx, name, pType, status, orderBy, page, pageSize, TEMPUID)
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
			q.Err = OperatePracticeStatus(ctx, id, status, TEMPUID)
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

// 处理获取学生名单列表
func practiceStudentH(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	testData := []int64{
		123, 123, 456,
	}
	err := UpsertPracticeStudent(context.Background(), 1, 1, testData)
	if err != nil {
		q.Err = err
		q.RespErr()
		return
	}
	q.Resp()
}

func practiceSH(ctx context.Context) {
	TEMPUID := int64(1)
	q := cmn.GetCtxValue(ctx)
	// TODO 对接用户管理的令牌功能
	//userID, _ := q.Session.Values["ID"].(int64)
	//if userID <= 0 {
	//	q.Err = fmt.Errorf("invalid session")
	//	z.Error(q.Err.Error())
	//	return
	//}
	method := strings.ToLower(q.R.Method)
	ctx, cancel := context.WithTimeout(q.R.Context(), 5*time.Second)
	defer cancel()
	if method != "get" {
		q.Err = fmt.Errorf("please call /api/practiceS with  http GET method")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	name := q.R.URL.Query().Get("name")
	d := q.R.URL.Query().Get("difficulty")
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
		q.Err = fmt.Errorf("缺失分页查询页大小")
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
	p, total, err := ListPracticeS(ctx, name, d, orderBy, page, pageSize, TEMPUID)
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
}

func practiceStudentListH(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	ctx, cancel := context.WithTimeout(q.R.Context(), 5*time.Second)
	defer cancel()
	id := q.R.URL.Query().Get("id")
	if id == "" {
		q.Err = fmt.Errorf("缺失练习ID")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	var pid int64
	tResult := make([]int64, 0)
	result := make([]StudentInfo, 0)

	pid, q.Err = strconv.ParseInt(id, 10, 64)

	s := `SELECT student_id FROM t_practice_student WHERE practice_id = $1 AND status = $2`
	sqlxDB := cmn.GetDbConn()
	rows, err := sqlxDB.QueryContext(ctx, s, pid, PracticeStudentStatus.Normal)
	if err != nil {
		err = fmt.Errorf("query studentList in practice failed:%v", err)
		z.Error(err.Error())
		q.Err = err
		q.RespErr()
		return
	}
	defer func() {
		err = rows.Close()
		if err != nil {
			q.Err = err
			z.Error(err.Error())
			q.RespErr()
			return
		}
	}()
	for rows.Next() {
		var id int64
		q.Err = rows.Scan(&id)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		tResult = append(tResult, id)
	}
	// 这里已经有这个student的id数组之后，就需要获取这个数据
	if len(tResult) == 0 {
		data, _ := json.Marshal(tResult)
		q.Msg.Data = data
		z.Info("---->" + cmn.FncName())
		q.Msg.Msg = "OK but empty result"
		q.Resp()
		return
	}
	s1 := `SELECT id, account, official_name, id_card_no, mobile_phone,password FROM t_user WHERE id IN (?)"`
	query, args, err := sqlx.In(s1, result)
	if err != nil {
		q.Err = fmt.Errorf("prepare sqlx.In sql query failed:%v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	query = sqlx.Rebind(sqlx.DOLLAR, query)
	z.Sugar().Debugf("打印一下构建的查询用户信息语句：%v", query)
	z.Sugar().Debugf("打印一下构建的查询用户信息注入参数：%v", args)
	rows, err = sqlxDB.QueryContext(ctx, s, args...)
	if err != nil {
		err = fmt.Errorf("query studentList in practice failed:%v", err)
		q.Err = err
		z.Error(err.Error())
		q.RespErr()
		return
	}
	defer func() {
		err = rows.Close()
		if err != nil {
			q.Err = err
			z.Error(err.Error())
			q.RespErr()
			return
		}
	}()
	for rows.Next() {
		var s StudentInfo
		q.Err = rows.Scan(&s.ID, &s.Account, &s.OfficialName, &s.IdCardNo, &s.Phone, &s.Password)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		result = append(result, s)
	}
	data, err := json.Marshal(result)
	if err != nil {
		q.Err = err
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	q.Msg.Data = data
	z.Info("---->" + cmn.FncName())
	q.Msg.Msg = "OK"
	q.Resp()
}
