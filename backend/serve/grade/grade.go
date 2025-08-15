package grade

//annotation:grade-service
//author:{"name":"txl","tel":"19832706790", "email":"188306257@qq.com"}

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/tidwall/gjson"
	"go.uber.org/zap"
	"w2w.io/cmn"
)

var z *zap.Logger

const (
	TIMEOUT = 60 * time.Second
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

	//  ********** 成绩分布接口 **********
	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: gradeDistributionH,

		Path: "/grade/distribution",
		Name: "grade/distribution",

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})

	//  ********** 考生成绩列表接口 **********
	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: gradeExamineeListH,

		Path: "/grade/examinee/list",
		Name: "grade/examinee/list",

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})

	//  ********** 成绩接口 **********
	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: gradeH,

		Path: "/grade",
		Name: "grade",

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: gradeSH,

		Path: "/grades",
		Name: "grades",

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
			req       GradeListReq
			category  string // 类别：exam practice
			userID    int64
			page      int
			pageSize  int
			submitted int
		)
		var p string

		if p = queryParams.Get("page"); p == "" {
			q.Err = fmt.Errorf("页码为空: %s", p)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		if page, err = strconv.Atoi(p); err != nil {
			q.Err = fmt.Errorf("无效页码: %s", p)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		req.Page = page

		if p = queryParams.Get("pageSize"); p == "" {
			q.Err = fmt.Errorf("每页数量为空: %s", p)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		if pageSize, err = strconv.Atoi(p); err != nil {
			q.Err = fmt.Errorf("无效每页数量: %s", p)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		req.PageSize = pageSize

		// 用户身份
		if q.SysUser == nil || !q.SysUser.ID.Valid || forceErr == "q.SysUser nil" {
			q.Err = fmt.Errorf("非法请求，鉴权用户失败")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		userID = q.SysUser.ID.Int64

		if name := queryParams.Get("name"); name != "" {
			req.Filter.Name = name
		}

		if category = queryParams.Get("category"); category == "" {
			q.Err = fmt.Errorf("不支持的类型: %s", category)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		switch category {
		case "exam":
			if p = queryParams.Get("examID"); p != "" {
				examID, err := strconv.ParseInt(p, 10, 64)
				if err != nil {
					q.Err = fmt.Errorf("无效考试ID: %s", p)
					z.Error(q.Err.Error())
					q.RespErr()
					return
				}
				req.ExamID = examID
			}

			if examType := queryParams.Get("type"); examType != "" {
				req.Filter.Type = examType
			}

			submitted, err = strconv.Atoi(queryParams.Get("submitted"))
			if err != nil || submitted != 0 && submitted != 1 && submitted != -1 {
				q.Err = fmt.Errorf("无效提交状态: %d", submitted)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			req.Filter.Submitted = submitted

			dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
			defer cancel()

			// 调用数据库层处理
			var result []GradeExam
			result, rowCount, err = gradeListExam(dmlCtx, userID, &req)
			if err != nil {
				q.Err = fmt.Errorf("获取考试成绩列表失败 错误信息:%w", err)
				q.RespErr()
				return
			}

			if result != nil {
				data, _ = json.Marshal(result)
			}

		case "practice":
			// 练习ID
			if p := queryParams.Get("practiceID"); p != "" {
				practiceID, err := strconv.ParseInt(p, 10, 64)
				if err != nil {
					q.Err = fmt.Errorf("无效练习ID: %s", p)
					z.Error(q.Err.Error())
					q.RespErr()
					return
				}
				req.PracticeID = practiceID
			}

			dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
			defer cancel()

			// 调用数据库层处理
			var result []GradePractice
			result, rowCount, err = gradeListPractice(dmlCtx, userID, &req)
			if err != nil {
				q.Err = fmt.Errorf("获取练习成绩列表失败 错误信息:%w", err)
				q.RespErr()
				return
			}

			if result != nil {
				data, _ = json.Marshal(result)
			}

		default:
			q.Err = fmt.Errorf("不支持的类型: %s", category)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Msg.Data = data
		q.Msg.RowCount = rowCount

		q.Err = nil
		q.Msg.Status = 0
		q.Msg.Msg = "success"

	default:
		q.Err = fmt.Errorf("不支持的请求方法: %s", method)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	q.Resp()
}

func gradeSubmissionH(ctx context.Context) {
	z.Info("---->" + cmn.FncName())

	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}

	q := cmn.GetCtxValue(ctx)

	method := strings.ToLower(q.R.Method)
	switch method {
	case "patch":

		var err error

		var buf []byte
		buf, err = io.ReadAll(q.R.Body)
		if forceErr == "io.ReadAll fail" {
			err = errors.New(forceErr)
		}
		if err != nil {
			q.Err = err
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		defer func() {
			err = q.R.Body.Close()
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

		var userID int64
		var examIDs []int

		// 用户身份校验
		if q.SysUser == nil || !q.SysUser.ID.Valid || forceErr == "q.SysUser nil" {
			q.Err = fmt.Errorf("非法请求，鉴权用户失败")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		userID = q.SysUser.ID.Int64

		examIDQuerys := gjson.GetBytes(buf, "data.exam_ids").Array()
		if len(examIDQuerys) <= 0 {
			q.Err = fmt.Errorf("examIDs为空")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
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

		dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
		defer cancel()

		rowsAffected, err := setExamGradeSubmitted(dmlCtx, userID, examIDs)
		if forceErr == "setExamGradeSubmitted fail" {
			err = errors.New(forceErr)
		}
		if err != nil {
			q.Err = err
			q.RespErr()
			return
		}
		q.Err = nil
		q.Msg.Status = 0
		q.Msg.Msg = fmt.Sprintf("成功提交%v条数据", rowsAffected)

	default:
		q.Err = fmt.Errorf("不支持的请求方法: %s", method)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	q.Resp()
}

func gradeDistributionH(ctx context.Context) {
	z.Info("---->" + cmn.FncName())

	// // 测试使用强行触发错误
	// forceErr := ""
	// if val := ctx.Value("force-error"); val != nil {
	// 	forceErr = val.(string)
	// }

	q := cmn.GetCtxValue(ctx)

	method := strings.ToLower(q.R.Method)
	switch method {
	case "get":
		var (
			// 结果集
			err  error
			data []byte

			// 必需参数集
			category   string // 类别：exam practice
			columnNum  int
			examID     int
			practiceID int
		)
		var p string

		queryParams := q.R.URL.Query()

		// 列数
		if p = queryParams.Get("columnNum"); p == "" {
			q.Err = fmt.Errorf("列数为空: %s", p)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		if columnNum, err = strconv.Atoi(p); err != nil {
			q.Err = fmt.Errorf("列数无效: %d", columnNum)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// // 用户身份
		// if q.SysUser == nil || !q.SysUser.ID.Valid || forceErr == "q.SysUser nil" {
		// 	q.Err = fmt.Errorf("非法请求，鉴权用户失败")
		// 	z.Error(q.Err.Error())
		// 	q.RespErr()
		// 	return
		// }

		dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
		defer cancel()

		if category = queryParams.Get("category"); category == "" {
			q.Err = fmt.Errorf("类型为空: %s", category)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		switch category {
		case "exam":
			// 考试ID - 涉及多场次
			if p = queryParams.Get("examID"); p == "" {
				q.Err = fmt.Errorf("考试ID为空: %s", p)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			if examID, err = strconv.Atoi(p); err != nil {
				q.Err = fmt.Errorf("无效考试ID: %d", examID)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			var result ExamGradeDistribution

			result, err = gradeDistributionExam(dmlCtx, examID, columnNum)
			if err != nil {
				q.Err = err
				q.RespErr()
				return
			}

			data, err = json.Marshal(result)
			if err != nil {
				q.Err = fmt.Errorf("json序列化失败: %v", err)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

		case "practice":
			// 练习ID
			if p = queryParams.Get("practiceID"); p == "" {
				q.Err = fmt.Errorf("练习ID为空: %s", p)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			if practiceID, err = strconv.Atoi(p); err != nil {
				q.Err = fmt.Errorf("无效练习ID: %d", practiceID)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			result, err := gradeDistributionPractice(dmlCtx, practiceID, columnNum)
			if err != nil {
				q.Err = err
				q.RespErr()
				return
			}

			data, err = json.Marshal(result)
			if err != nil {
				q.Err = fmt.Errorf("json序列化失败: %v", err)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

		default:
			q.Err = fmt.Errorf("不支持的类型: %s", category)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Msg.Data = data

		q.Err = nil
		q.Msg.Status = 0
		q.Msg.Msg = "success"

	default:
		q.Err = fmt.Errorf("unsupported method: %s", method)
		z.Warn(q.Err.Error())
		q.RespErr()
		return
	}
	q.Resp()
}

func gradeExamineeListH(ctx context.Context) {
	z.Info("---->" + cmn.FncName())

	// 测试使用强行触发错误
	// forceErr := ""
	// if val := ctx.Value("force-error"); val != nil {
	// 	forceErr = val.(string)
	// }

	q := cmn.GetCtxValue(ctx)

	method := strings.ToLower(q.R.Method)
	switch method {
	case "get":
		var (
			// 结果集
			err      error
			data     []byte
			rowCount int64

			req GradeExamineeListReq

			// 必需参数集
			category   string // 类别：exam practice
			page       int
			pageSize   int
			keyword    string
			examID     []int64
			practiceID []int64
		)
		var p string
		queryParams := q.R.URL.Query()

		if p = queryParams.Get("page"); p == "" {
			q.Err = fmt.Errorf("页码为空: %s", p)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		if page, err = strconv.Atoi(p); err != nil {
			q.Err = fmt.Errorf("传入无效页码: %s", p)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		req.Page = page

		if p = queryParams.Get("pageSize"); p == "" {
			q.Err = fmt.Errorf("每页数量为空: %s", p)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		if pageSize, err = strconv.Atoi(p); err != nil {
			q.Err = fmt.Errorf("传入无效每页数量: %s", p)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		req.PageSize = pageSize

		if keyword = queryParams.Get("keyword"); keyword != "" {
			req.Filter.Keyword = keyword
		}

		// 用户身份
		// if q.SysUser == nil || !q.SysUser.ID.Valid || forceErr == "q.SysUser nil" {
		// 	q.Err = fmt.Errorf("非法请求，鉴权用户失败")
		// 	z.Error(q.Err.Error())
		// 	q.RespErr()
		// 	return
		// }

		if category = queryParams.Get("category"); category == "" {
			q.Err = fmt.Errorf("类别为空")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
		defer cancel()

		switch category {
		case "exam":
			if examIDs := queryParams.Get("examID"); examIDs != "" {
				var eids []int64
				for _, eid := range bytes.Split([]byte(examIDs), []byte(",")) {
					e, err := strconv.ParseInt(string(eid), 10, 64)
					if err != nil {
						q.Err = err
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					if e <= 0 {
						q.Err = fmt.Errorf("传入考试ID存在非正整数: %d", e)
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					eids = append(eids, e)
				}
				examID = eids
			}
			req.ExamID = examID

			var result []ExamineeScoreList
			result, rowCount, err = gradeExamineeListExam(dmlCtx, req)
			if err != nil {
				q.Err = err
				q.RespErr()
				return
			}

			data, _ = json.Marshal(result)

		case "practice":
			if practiceIDs := queryParams.Get("practiceID"); practiceIDs != "" {
				var pids []int64
				for _, pid := range bytes.Split([]byte(practiceIDs), []byte(",")) {
					p, err := strconv.ParseInt(string(pid), 10, 64)
					if err != nil {
						q.Err = err
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					pids = append(pids, p)
				}
				practiceID = pids
			}
			req.PracticeID = practiceID

			var result []PracticeScoreList

			result, rowCount, err = gradeExamineeListPractice(dmlCtx, req)
			if err != nil {
				q.Err = err
				q.RespErr()
				return
			}

			data, _ = json.Marshal(result)
		default:
			q.Err = fmt.Errorf("不支持的类型: %s", category)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Msg.Data = data
		q.Msg.RowCount = rowCount

		q.Err = nil
		q.Msg.Status = 0
		q.Msg.Msg = "success"

	default:
		q.Err = fmt.Errorf("unsupported method: %s", method)
		z.Warn(q.Err.Error())
		q.RespErr()
		return
	}
	q.Resp()
}

func gradeH(ctx context.Context) {
	z.Info("---->" + cmn.FncName())

	// // 测试使用强行触发错误
	// forceErr := ""
	// if val := ctx.Value("force-error"); val != nil {
	// 	forceErr = val.(string)
	// }

	q := cmn.GetCtxValue(ctx)

	method := strings.ToLower(q.R.Method)
	switch method {
	case "get":
		var (
			// 结果集
			err  error
			data []byte

			// 必需参数集
			category      string // 类别：exam practice
			examSessionID int64  // 考试场次ID
			practiceID    int64  // 练习ID
		)
		var p string
		queryParams := q.R.URL.Query()

		if category = queryParams.Get("category"); category == "" {
			q.Err = errors.New("传入类别参数为空")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 用户身份
		// if q.SysUser == nil || !q.SysUser.ID.Valid || forceErr == "q.SysUser nil" {
		// 	q.Err = fmt.Errorf("非法请求，鉴权用户失败")
		// 	z.Error(q.Err.Error())
		// 	q.RespErr()
		// 	return
		// }
		// userID = q.SysUser.ID.Int64

		dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
		defer cancel()

		switch category {
		case "exam":
			p = queryParams.Get("examSessionID")
			if p == "" {
				q.Err = fmt.Errorf("examSessionID为空: %s", p)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			examSessionID, err = strconv.ParseInt(p, 10, 64)
			if err != nil {
				q.Err = fmt.Errorf("examSessionID无效, 传入:%s", p)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			var result Analysis

			result, err = gradeAnalysisByID(dmlCtx, examSessionID, 0)
			if err != nil {
				q.Err = err
				q.RespErr()
				return
			}

			data, err = json.Marshal(result)
			if err != nil {
				q.Err = fmt.Errorf("结果json序列化失败: %v", err)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

		case "practice":
			if p = queryParams.Get("practiceID"); p == "" {
				q.Err = fmt.Errorf("practiceID为空: %s", p)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			if practiceID, err = strconv.ParseInt(p, 10, 64); err != nil {
				q.Err = fmt.Errorf("传入practiceID无效: %s", p)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			var result Analysis

			result, err = gradeAnalysisByID(dmlCtx, 0, practiceID)
			if err != nil {
				q.Err = err
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			data, err = json.Marshal(result)
			if err != nil {
				q.Err = fmt.Errorf("json序列化失败: %v", err)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

		default:
			q.Err = fmt.Errorf("不支持的类型: %s", category)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		q.Msg.Data = data

		q.Err = nil
		q.Msg.Status = 0
		q.Msg.Msg = "success"

	default:
		q.Err = fmt.Errorf("unsupported method: %s", method)
		z.Warn(q.Err.Error())
		q.RespErr()
		return
	}
	q.Resp()
}

func gradeSH(ctx context.Context) {
	z.Info("---->" + cmn.FncName())

	// 测试使用强行触发错误
	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}

	q := cmn.GetCtxValue(ctx)

	method := strings.ToLower(q.R.Method)
	switch method {
	case "get":
		var (
			// 结果集
			err  error
			data []byte

			// 必需参数集
			category      string // 类别：exam practice
			userID        int64
			studentID     int64
			examSessionID int64
			practiceID    int64
		)
		var p string
		queryParams := q.R.URL.Query()

		// TODO：不支持传入ID
		if p = queryParams.Get("studentID"); p != "" {
			if studentID, err = strconv.ParseInt(p, 10, 64); err != nil {
				q.Err = fmt.Errorf("无效学生ID: %s", p)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		} else {
			if q.SysUser == nil || !q.SysUser.ID.Valid || forceErr == "q.SysUser nil" {
				q.Err = fmt.Errorf("非法请求，鉴权用户失败")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			userID = q.SysUser.ID.Int64
			studentID = userID
		}
		dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
		defer cancel()

		conn := cmn.GetPgxConn()
		if conn == nil || forceErr == "conn nil" {
			err = fmt.Errorf("查询练习成绩列表获取数据库连接失败")
			z.Error(err.Error())
			return
		}

		var tx pgx.Tx
		tx, err = conn.Begin(context.Background())
		if err != nil || forceErr == "conn begin tx fail" {
			err = fmt.Errorf("开启事务失败: %w", err)
			z.Error(err.Error())
			return
		}

		if category = queryParams.Get("category"); category == "" {
			q.Err = fmt.Errorf("不支持的类型: %s", category)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		switch category {
		case "exam":
			if p = queryParams.Get("examSessionID"); p == "" {
				q.Err = fmt.Errorf("examSessionID为空: %s", p)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			if examSessionID, err = strconv.ParseInt(p, 10, 64); err != nil {
				q.Err = fmt.Errorf("examSessionID无效: %d", examSessionID)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			var result Map

			result, err = getScoreExam(dmlCtx, tx, studentID, examSessionID)
			if err != nil {
				q.Err = err
				q.RespErr()
				return
			}

			data, err = json.Marshal(result)
			if err != nil {
				q.Err = fmt.Errorf("json序列化失败: %v", err)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

		case "practice":
			if p = queryParams.Get("practiceID"); p == "" {
				q.Err = fmt.Errorf("practiceID为空: %s", p)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			if practiceID, err = strconv.ParseInt(p, 10, 64); err != nil {
				q.Err = fmt.Errorf("practiceID无效: %s", p)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			var result Map

			result, err = getScorePractice(dmlCtx, tx, studentID, practiceID)
			if err != nil {
				q.Err = err
				q.RespErr()
				return
			}

			data, err = json.Marshal(result)
			if err != nil {
				q.Err = fmt.Errorf("json序列化失败: %v", err)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

		default:
			q.Err = fmt.Errorf("不支持的类型: %s", category)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		q.Msg.Data = data

		q.Err = nil
		q.Msg.Status = 0
		q.Msg.Msg = "success"

	default:
		q.Err = fmt.Errorf("unsupported method: %s", method)
		z.Warn(q.Err.Error())
		q.RespErr()
		return
	}
	q.Resp()
}
