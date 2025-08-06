package grade

//annotation:grade-service
//author:{"name":"txl","tel":"19832706790", "email":"188306257@qq.com"}

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/gjson"
	"go.uber.org/zap"
	"w2w.io/cmn"
)

var z *zap.Logger

const (
	TIMEOUT = 10 * time.Second
)

func init() {
	//Setup package scope variables, just like z, db connector, configure parameters, etc.
	cmn.PackageStarters = append(cmn.PackageStarters, func() {
		z = cmn.GetLogger()
		z.Info("message zz settled")
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

	//  ********** 成绩列表接口 **********
	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: gradeListH,

		Path: "/grade/list",
		Name: "grade/list",

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})

	//  ********** 成绩提交接口 **********
	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: gradeSubmissionH,

		Path: "/grade/submission",
		Name: "grade/submission",

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})

}

func gradeListH(ctx context.Context) {
	z.Info("---->" + cmn.FncName())

	// 测试使用强行触发错误
	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}
	z.Debug(forceErr)

	q := cmn.GetCtxValue(ctx)

	method := strings.ToLower(q.R.Method)
	switch method {
	case "get":
		queryParams := q.R.URL.Query()

		var (
			// 结果集
			err      error
			data     []byte
			rowCount int64

			// 必需参数集
			req      GradeListArgs
			category string // 类别：exam practice
			page     string
			pageSize string
		)
		var p int

		if category = queryParams.Get("category"); category == "" {
			q.Err = fmt.Errorf("不支持的类型: %s", req.Category)
			z.Warn(q.Err.Error())
			q.RespErr()
			return
		}
		req.Category = category

		if page = queryParams.Get("page"); page == "" {
			q.Err = fmt.Errorf("页码为空: %s", page)
			z.Warn(q.Err.Error())
			q.RespErr()
			return
		}
		if p, err = strconv.Atoi(page); err != nil {
			q.Err = fmt.Errorf("无效页码: %s", page)
			z.Warn(q.Err.Error())
			q.RespErr()
			return
		}
		req.Page = p

		if pageSize = queryParams.Get("pageSize"); pageSize == "" {
			q.Err = fmt.Errorf("每页数量为空: %s", pageSize)
			z.Warn(q.Err.Error())
			q.RespErr()
			return
		}
		if p, err = strconv.Atoi(pageSize); err != nil {
			q.Err = fmt.Errorf("无效每页数量: %s", pageSize)
			z.Warn(q.Err.Error())
			q.RespErr()
			return
		}
		req.PageSize = p

		// 用户身份
		if q.SysUser == nil || forceErr == "q.SysUser nil" {
			q.Err = fmt.Errorf("非法请求，鉴权用户失败")
			z.Warn(q.Err.Error())
			q.RespErr()
			return
		}
		req.TeacherID = q.SysUser.ID.Int64

		// 管理员实现全部教师数据展示
		teacherID := queryParams.Get("teacherID")
		if teacherID != "" {
			p, err := strconv.ParseInt(teacherID, 10, 64)
			if err != nil {
				q.Err = fmt.Errorf("无效教师ID: %s", teacherID)
				z.Warn(q.Err.Error())
				q.RespErr()
				return
			}
			req.TeacherID = p
		}

		if name := queryParams.Get("name"); name != "" {
			req.Filter.Name = name
		}

		switch req.Category {
		case "exam":
			if examID := queryParams.Get("examID"); examID != "" {
				p, err := strconv.Atoi(examID)
				if err != nil {
					q.Err = fmt.Errorf("无效考试ID: %s", examID)
					z.Warn(q.Err.Error())
					q.RespErr()
					return
				}
				req.ExamID = p
			}

			if examType := queryParams.Get("type"); examType != "" {
				req.Filter.Type = examType
			}

			submitted := queryParams.Get("submitted")
			switch submitted {
			case "0":
				req.Filter.Submitted = 0
			case "1":
				req.Filter.Submitted = 1
			case "-1":
				req.Filter.Submitted = -1
			default:
				q.Err = fmt.Errorf("无效提交状态: %s", submitted)
				z.Warn(q.Err.Error())
				q.RespErr()
				return
			}

			dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
			defer cancel()

			// 调用数据库层处理
			var result []GradeExam
			result, rowCount, err = gradeListExam(dmlCtx, &req)
			if err != nil {
				q.Err = fmt.Errorf("获取考试成绩列表失败 错误信息:%w", err)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			if result != nil {
				data, _ = json.Marshal(result)
			}

			// z.Debug("gradeListExam", zap.Any("result", result))

		case "practice":
			// 练习ID
			if practiceID := queryParams.Get("practiceID"); practiceID != "" {
				p, err := strconv.Atoi(practiceID)
				if err != nil {
					q.Err = fmt.Errorf("无效练习ID: %s", practiceID)
					z.Warn(q.Err.Error())
					q.RespErr()
					return
				}
				req.PracticeID = p
			}

			dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
			defer cancel()

			// 调用数据库层处理
			var result []GradePractice
			result, rowCount, err = gradeListPractice(dmlCtx, &req)
			if err != nil {
				q.Err = fmt.Errorf("获取练习成绩列表失败 错误信息:%w", err)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			if result != nil {
				data, _ = json.Marshal(result)
			}

		default:
			q.Err = fmt.Errorf("不支持的类型: %s", req.Category)
			z.Warn(q.Err.Error())
			q.RespErr()
			return
		}

		q.Msg.Data = data
		q.Msg.RowCount = rowCount

		q.Err = nil
		q.Msg.Status = 0
		q.Msg.Msg = "success"
		q.Resp()

	default:
		q.Err = fmt.Errorf("不支持的请求方法: %s", method)
		z.Warn(q.Err.Error())
		q.RespErr()
		return
	}
}

func gradeSubmissionH(ctx context.Context) {
	z.Info("---->" + cmn.FncName())

	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}
	z.Debug(forceErr)

	q := cmn.GetCtxValue(ctx)

	method := strings.ToLower(q.R.Method)
	switch method {
	case "patch":

		var err error
		var args GradeSubmitArgs

		var buf []byte
		buf, err = io.ReadAll(q.R.Body)
		if forceErr == "io.ReadAll fail" {
			err = fmt.Errorf(forceErr)
		}
		if err != nil {
			q.Err = err
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		defer func() {
			err := q.R.Body.Close()
			if forceErr == "q.R.Body.Close-fail" {
				err = errors.New(forceErr)
			}
			if err != nil {
				z.Error(err.Error())
			}
		}()

		if len(buf) == 0 || forceErr == "io.ReadAll len(buf)==0" {
			q.Err = fmt.Errorf("调用/api/course接口时，请求体为空")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 用户身份校验
		if q.SysUser == nil || forceErr == "q.SysUser nil" {
			q.Err = fmt.Errorf("非法请求，鉴权用户失败")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		args.TeacherID = q.SysUser.ID.Int64

		examIDQuerys := gjson.GetBytes(buf, "data.exam_ids").Array()
		if len(examIDQuerys) <= 0 {
			q.Err = fmt.Errorf("examIDs为空")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		var examIDs []int
		for _, examIDQuery := range examIDQuerys {
			id := int(examIDQuery.Num)
			if id <= 0 {
				q.Err = fmt.Errorf("请求存在非法examID")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			examIDs = append(examIDs, id)
		}
		args.ExamIDs = examIDs

		dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
		defer cancel()

		rowsAffected, err := setExamGradeSubmitted(dmlCtx, &args)
		if forceErr == "setExamGradeSubmitted fail" {
			err = fmt.Errorf(forceErr)
		}
		if err != nil {
			q.Err = err
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		q.Err = nil
		q.Msg.Status = 0
		q.Msg.Msg = fmt.Sprintf("success rowsAffected:%v", rowsAffected)
		q.Resp()

	default:
		q.Err = fmt.Errorf("不支持的请求方法: %s", method)
		z.Warn(q.Err.Error())
		q.RespErr()
		return
	}
}
