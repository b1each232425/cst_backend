package grade

//annotation:grade-service
//author:{"name":"txl","tel":"19832706790", "email":"188306257@qq.com"}

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
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

	// // 测试使用强行触发错误
	// forceErr := ""
	// if val := ctx.Value("force-error"); val != nil {
	// 	forceErr = val.(string)
	// }

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
			q.Err = fmt.Errorf("不支持的类型: %s", category)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		req.Category = category

		if page = queryParams.Get("page"); page == "" {
			q.Err = fmt.Errorf("页码为空: %s", page)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		if p, err = strconv.Atoi(page); err != nil {
			q.Err = fmt.Errorf("无效页码: %s", page)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		req.Page = p

		if pageSize = queryParams.Get("pageSize"); pageSize == "" {
			q.Err = fmt.Errorf("每页数量为空: %s", pageSize)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		if p, err = strconv.Atoi(pageSize); err != nil {
			q.Err = fmt.Errorf("无效每页数量: %s", pageSize)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		req.PageSize = p

		// 用户身份
		// if q.SysUser == nil || !q.SysUser.ID.Valid || forceErr == "q.SysUser nil" {
		// 	q.Err = fmt.Errorf("非法请求，鉴权用户失败")
		// 	z.Error(q.Err.Error())
		// 	q.RespErr()
		// 	return
		// }
		// req.TeacherID = q.SysUser.ID.Int64
		req.TeacherID = 1681

		if name := queryParams.Get("name"); name != "" {
			req.Filter.Name = name
		}

		switch req.Category {
		case "exam":
			if examID := queryParams.Get("examID"); examID != "" {
				p, err := strconv.Atoi(examID)
				if err != nil {
					q.Err = fmt.Errorf("无效考试ID: %s", examID)
					z.Error(q.Err.Error())
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
				z.Error(q.Err.Error())
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
				q.RespErr()
				return
			}

			if result != nil {
				data, _ = json.Marshal(result)
			}

		case "practice":
			// 练习ID
			if practiceID := queryParams.Get("practiceID"); practiceID != "" {
				p, err := strconv.Atoi(practiceID)
				if err != nil {
					q.Err = fmt.Errorf("无效练习ID: %s", practiceID)
					z.Error(q.Err.Error())
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
				q.RespErr()
				return
			}

			if result != nil {
				data, _ = json.Marshal(result)
			}

		default:
			q.Err = fmt.Errorf("不支持的类型: %s", req.Category)
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
		var args GradeSubmitArgs

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

		// // 用户身份校验
		// if q.SysUser == nil || !q.SysUser.ID.Valid || forceErr == "q.SysUser nil" {
		// 	q.Err = fmt.Errorf("非法请求，鉴权用户失败")
		// 	z.Error(q.Err.Error())
		// 	q.RespErr()
		// 	return
		// }
		// args.TeacherID = q.SysUser.ID.Int64
		args.TeacherID = 1681

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
			err = errors.New(forceErr)
		}
		if err != nil {
			q.Err = err
			q.RespErr()
			return
		}
		q.Err = nil
		q.Msg.Status = 0
		q.Msg.Msg = fmt.Sprintf("success rowsAffected:%v", rowsAffected)

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
			category      string // 类别：exam practice
			req           GradeDistributionArgs
			columnNumStr  string
			columnNum     int
			examIDStr     string
			examID        int
			practiceIDStr string
			practiceID    int
		)
		queryParams := q.R.URL.Query()

		if category = queryParams.Get("category"); category == "" {
			q.Err = fmt.Errorf("类型为空: %s", category)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if columnNumStr = queryParams.Get("columnNum"); columnNumStr == "" {
			q.Err = fmt.Errorf("列数为空: %s", columnNumStr)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		if columnNum, err = strconv.Atoi(columnNumStr); err != nil {
			q.Err = fmt.Errorf("列数无效: %d", columnNum)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		req.ColumnNum = columnNum

		// 用户身份
		// if q.SysUser == nil || !q.SysUser.ID.Valid || forceErr == "q.SysUser nil" {
		// 	q.Err = fmt.Errorf("非法请求，鉴权用户失败")
		// 	z.Error(q.Err.Error())
		// 	q.RespErr()
		// 	return
		// }
		// req.TeacherID = q.SysUser.ID.Int64

		dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
		defer cancel()

		switch category {
		case "exam":
			if examIDStr = queryParams.Get("examID"); examIDStr == "" {
				q.Err = fmt.Errorf("考试ID为空: %s", examIDStr)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			if examID, err = strconv.Atoi(examIDStr); err != nil {
				q.Err = fmt.Errorf("无效考试ID: %d", examID)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			req.ExamID = examID

			var result ExamGradeDistribution

			result, err = GradeDistributionExam(dmlCtx, req)
			if err != nil {
				q.Err = err
				q.RespErr()
				return
			}

			// z.Debug("gradeDistributionExam", zap.Any("result", result))

			data, err = json.Marshal(result)
			if err != nil {
				q.Err = fmt.Errorf("json序列化失败: %v", err)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

		case "practice":
			if practiceIDStr = queryParams.Get("practiceID"); practiceIDStr == "" {
				q.Err = fmt.Errorf("练习ID为空: %s", practiceIDStr)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			if practiceID, err = strconv.Atoi(practiceIDStr); err != nil {
				q.Err = fmt.Errorf("无效练习ID: %d", practiceID)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			req.PracticeID = practiceID

			result, err := GradeDistributionPractice(dmlCtx, req)
			if err != nil {
				q.Err = err
				q.RespErr()
				return
			}

			// z.Debug("gradeDistributionPractice", zap.Any("result", result))

			data, err = json.Marshal(result)
			if err != nil {
				q.Err = fmt.Errorf("json序列化失败: %v", err)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

		default:
			q.Err = fmt.Errorf("不支持的类型: %s", req.Category)
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

			req GradeExamineeListArgs

			// 必需参数集
			category   string // 类别：exam practice
			page       string
			pageSize   string
			keyword    string
			examID     []int
			practiceID []int
		)
		var p int
		queryParams := q.R.URL.Query()

		if category = queryParams.Get("category"); category == "" {
			q.Err = fmt.Errorf("不支持的类型: %s", category)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		req.Category = category

		if page = queryParams.Get("page"); page == "" {
			q.Err = fmt.Errorf("页码为空: %s", page)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		if p, err = strconv.Atoi(page); err != nil {
			q.Err = fmt.Errorf("无效页码: %s", page)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		req.Page = p

		if pageSize = queryParams.Get("pageSize"); pageSize == "" {
			q.Err = fmt.Errorf("每页数量为空: %s", pageSize)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		if p, err = strconv.Atoi(pageSize); err != nil {
			q.Err = fmt.Errorf("无效每页数量: %s", pageSize)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		req.PageSize = p

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
		// req.TeacherID = q.SysUser.ID.Int64
		req.TeacherID = 1681

		dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
		defer cancel()

		switch category {
		case "exam":
			if examIDs := queryParams.Get("examID"); examIDs != "" {
				var intSlice []int
				for _, examID := range bytes.Split([]byte(examIDs), []byte(",")) {
					intValue, err := strconv.Atoi(string(examID))
					if err != nil {
						q.Err = err
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					intSlice = append(intSlice, intValue)
				}
				examID = intSlice
			}
			req.ExamID = examID

			var result []ExamExamineeScoreInfo

			result, rowCount, err = GradeExamineeListExam(dmlCtx, req)
			if err != nil {
				q.Err = err
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			z.Debug("gradeListExamineeExam", zap.Any("result", result))

			if result != nil {
				data, _ = json.Marshal(result)
			}

		case "practice":
			if practiceIDs := queryParams.Get("practiceID"); practiceIDs != "" {
				var intSlice []int
				for _, practiceID := range bytes.Split([]byte(practiceIDs), []byte(",")) {
					intValue, err := strconv.Atoi(string(practiceID))
					if err != nil {
						q.Err = err
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
					intSlice = append(intSlice, intValue)
				}
				practiceID = intSlice
			}
			req.PracticeID = practiceID

			var result []PracticeExamineeScoreInfo

			result, rowCount, err = GradeExamineeListPractice(dmlCtx, req)
			if err != nil {
				q.Err = err
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			z.Debug("gradeListExamineePractice", zap.Any("result", result))

			if result != nil {
				data, _ = json.Marshal(result)
			}

		default:
			q.Err = fmt.Errorf("不支持的类型: %s", req.Category)
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
			req              GradeArgs
			category         string // 类别：exam practice
			examSessionIDStr string
			examSessionID    int
			practiceIDStr    string
			practiceID       int
		)
		queryParams := q.R.URL.Query()

		if category = queryParams.Get("category"); category == "" {
			q.Err = fmt.Errorf("不支持的类型: %s", category)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		req.Category = category

		// // 用户身份
		// if q.SysUser == nil || !q.SysUser.ID.Valid || forceErr == "q.SysUser nil" {
		// 	q.Err = fmt.Errorf("非法请求，鉴权用户失败")
		// 	z.Error(q.Err.Error())
		// 	q.RespErr()
		// 	return
		// }
		// req.TeacherID = q.SysUser.ID.Int64
		req.TeacherID = 1681

		switch category {
		case "exam":
			if examSessionIDStr = queryParams.Get("examSessionID"); examSessionIDStr == "" {
				q.Err = fmt.Errorf("examSessionID为空: %s", examSessionIDStr)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			if examSessionID, err = strconv.Atoi(examSessionIDStr); err != nil {
				q.Err = fmt.Errorf("examSessionID无效: %d", examSessionID)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			req.ExamSessionID = examSessionID

			dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
			defer cancel()

			result, err := GradeAnalysisExam(dmlCtx, req)
			if err != nil {
				q.Err = err
				q.RespErr()
				return
			}

			// z.Debug("gradeAnalysisExam", zap.Any("result", result))

			data, err = json.Marshal(result)
			if err != nil {
				q.Err = fmt.Errorf("json序列化失败: %v", err)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

		case "practice":
			if practiceIDStr = queryParams.Get("practiceID"); practiceIDStr == "" {
				q.Err = fmt.Errorf("practiceID为空: %s", practiceIDStr)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			if practiceID, err = strconv.Atoi(practiceIDStr); err != nil {
				q.Err = fmt.Errorf("practiceID无效: %d", practiceID)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			req.PracticeID = practiceID

			dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
			defer cancel()

			result, err := GetAnalysisPractice(dmlCtx, req)
			if err != nil {
				q.Err = err
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			// z.Debug("gradeAnalysisPractice", zap.Any("result", result))

			data, err = json.Marshal(result)
			if err != nil {
				q.Err = fmt.Errorf("json序列化失败: %v", err)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

		default:
			q.Err = fmt.Errorf("不支持的类型: %s", req.Category)
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
			req              GradeArgs
			category         string // 类别：exam practice
			studentIDStr     string
			studentID        int
			examSessionIDStr string
			examSessionID    int
			practiceIDStr    string
			practiceID       int
		)
		queryParams := q.R.URL.Query()

		if category = queryParams.Get("category"); category == "" {
			q.Err = fmt.Errorf("不支持的类型: %s", category)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		req.Category = category

		// // 用户身份
		// if q.SysUser == nil || !q.SysUser.ID.Valid || forceErr == "q.SysUser nil" {
		// 	q.Err = fmt.Errorf("非法请求，鉴权用户失败")
		// 	z.Error(q.Err.Error())
		// 	q.RespErr()
		// 	return
		// }
		// req.TeacherID = q.SysUser.ID.Int64
		req.TeacherID = 1681

		if studentIDStr = queryParams.Get("studentID"); studentIDStr == "" {
			q.Err = fmt.Errorf("学生ID为空：: %s", studentIDStr)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		if studentID, err = strconv.Atoi(studentIDStr); err != nil {
			q.Err = fmt.Errorf("无效学生ID: %d", studentID)
			z.Error(q.Err.Error())
			q.RespErr()
			return
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

		switch category {
		case "exam":
			if examSessionIDStr = queryParams.Get("examSessionID"); examSessionIDStr == "" {
				q.Err = fmt.Errorf("examSessionID为空: %s", examSessionIDStr)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			if examSessionID, err = strconv.Atoi(examSessionIDStr); err != nil {
				q.Err = fmt.Errorf("examSessionID无效: %d", examSessionID)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			result, err := getScoreS(dmlCtx, tx, studentID, 0, examSessionID)
			if err != nil {
				q.Err = err
				q.RespErr()
				return
			}

			z.Debug("gradeAnalysisExam", zap.Any("result", result))

			data, err = json.Marshal(result)
			if err != nil {
				q.Err = fmt.Errorf("json序列化失败: %v", err)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

		case "practice":
			if practiceIDStr = queryParams.Get("practiceID"); practiceIDStr == "" {
				q.Err = fmt.Errorf("practiceID为空: %s", practiceIDStr)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			if practiceID, err = strconv.Atoi(practiceIDStr); err != nil {
				q.Err = fmt.Errorf("practiceID无效: %d", practiceID)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			result, err := getScoreS(dmlCtx, tx, studentID, 0, examSessionID)
			if err != nil {
				q.Err = err
				q.RespErr()
				return
			}

			dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
			defer cancel()

			result, err = getScoreS(dmlCtx, tx, studentID, examSessionID, practiceID)
			if err != nil {
				q.Err = err
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			// z.Debug("gradeAnalysisPractice", zap.Any("result", result))

			data, err = json.Marshal(result)
			if err != nil {
				q.Err = fmt.Errorf("json序列化失败: %v", err)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

		default:
			q.Err = fmt.Errorf("不支持的类型: %s", req.Category)
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
