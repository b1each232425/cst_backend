package mark

//annotation:mark-service
//author:{"name":"Lin WeiJia","tel":"18475124988","email":"wj2144632819@qq.com"}

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"
	"w2w.io/serve/auth_mgt"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
	"github.com/sony/gobreaker/v2"
	"github.com/spf13/viper"
	"w2w.io/null"
	"w2w.io/serve/ai_mark"

	"go.uber.org/zap"
	"w2w.io/cmn"
)

var (
	aiMarkTaskLimiter *cmn.RateLimiterTaskRunner     // 控制请求速率/并发，防止过载
	aiMarkTaskCB      *gobreaker.CircuitBreaker[any] // 保护系统免受下游服务失败影响
)

func init() {
	//Setup package scope variables, just like logger, db connector, configure parameters, etc.
	cmn.PackageStarters = append(cmn.PackageStarters, func() {
		z = cmn.GetLogger()
		z.Info("user zLogger settled")
	})
}

func Enroll(author string) {
	z.Info("mark.Enroll called")

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

	// 获取考试列表
	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: getExamList,

		Path: "/mark/exam",
		Name: "/mark/exam",

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "试卷批改.考试批改.查看考试列表",
				AccessAction: auth_mgt.CAPIAccessActionRead,
				Configurable: true,
			},
		},

		Developer: developer,
		WhiteList: true,

		DomainID: int64(cmn.CDomainSys),

		DefaultDomain: int64(cmn.CDomainSys),
	})

	// 获取练习列表
	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: getPracticeList,

		Path: "/mark/practice",
		Name: "/mark/practice",

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "试卷批改.练习批改.查看练习列表",
				AccessAction: auth_mgt.CAPIAccessActionRead,
				Configurable: true,
			},
		},

		Developer: developer,
		WhiteList: true,

		DomainID: int64(cmn.CDomainSys),

		DefaultDomain: int64(cmn.CDomainSys),
	})

	// 考试详情/批改
	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: examSessionDetail,

		Path: "/mark/exam-session/detail",
		Name: "/mark/exam-session/detail",

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "试卷批改.考试批改.查看考试详情",
				AccessAction: auth_mgt.CAPIAccessActionRead,
				Configurable: true,
			},
			{
				Name:         "试卷批改.考试批改.批改考试",
				AccessAction: auth_mgt.CAPIAccessActionUpdate,
				Configurable: true,
			},
		},

		Developer: developer,
		WhiteList: true,

		DomainID: int64(cmn.CDomainSys),

		DefaultDomain: int64(cmn.CDomainSys),
	})

	// 练习详情/批改
	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: practiceDetail,

		Path: "/mark/practice/detail",
		Name: "/mark/practice/detail",

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "试卷批改.练习批改.查看练习详情",
				AccessAction: auth_mgt.CAPIAccessActionRead,
				Configurable: true,
			},
			{
				Name:         "试卷批改.练习批改.批改练习",
				AccessAction: auth_mgt.CAPIAccessActionUpdate,
				Configurable: true,
			},
		},

		Developer: developer,
		WhiteList: true,

		DomainID: int64(cmn.CDomainSys),

		DefaultDomain: int64(cmn.CDomainSys),
	})

	// 获取练习/考试学生的答案和批改结果
	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: getStudentAnswersAndMarkResults,

		Path: "/mark/student-answers-and-mark-results",
		Name: "/mark/student-answers-and-mark-results",

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "试卷批改.获取学生答案与批改结果",
				AccessAction: auth_mgt.CAPIAccessActionRead,
				Configurable: false, // 不可配置
			},
		},

		Developer: developer,
		WhiteList: true,

		DomainID: int64(cmn.CDomainSys),

		DefaultDomain: int64(cmn.CDomainSys),
	})

	// 考试提交
	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: examSessionSubmission,

		Path: "/mark/exam-session/results-submission",
		Name: "/mark/exam-session/results-submission",

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "试卷批改.考试批改.提交考试",
				AccessAction: auth_mgt.CAPIAccessActionUpdate,
				Configurable: true,
			},
		},

		Developer: developer,
		WhiteList: true,

		DomainID: int64(cmn.CDomainSys),

		DefaultDomain: int64(cmn.CDomainSys),
	})

	// 练习学生提交
	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: practiceStudentSubmission,

		Path: "/mark/practice-student/results-submission",
		Name: "/mark/practice-student/results-submission",

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "试卷批改.练习批改.提交练习",
				AccessAction: auth_mgt.CAPIAccessActionUpdate,
				Configurable: true,
			},
		},

		Developer: developer,
		WhiteList: true,

		DomainID: int64(cmn.CDomainSys),

		DefaultDomain: int64(cmn.CDomainSys),
	})

	initAIMarkTaskLimiterAndBreaker()

	cmn.RegisterTaskHandler(TaskTypeAIMarkRequest, taskMiddleware(handleAIMarkTask))
}

// --------------------- handler -----------------------

// 获取考试批改列表
func getExamList(ctx context.Context) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)

	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	method := strings.ToLower(q.R.Method)
	if method != "get" {
		q.Err = fmt.Errorf("不支持的 HTTP 方法: %s", q.R.Method)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	//var readable bool
	//
	//if readable, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, nil, q.Ep.Path, auth_mgt.CAPIAccessActionRead); q.Err != nil {
	//	q.Err = fmt.Errorf("获取访问权限失败: %v", q.Err)
	//	z.Error(q.Err.Error())
	//	q.RespErr()
	//	return
	//}
	//
	//if !readable {
	//	q.Err = errors.New("无权访问")
	//	z.Error(q.Err.Error())
	//	q.RespErr()
	//	return
	//}

	queryParams := q.R.URL.Query()

	pageIndexStr := queryParams.Get("page")
	if pageIndexStr == "" {
		pageIndexStr = "1"
	}
	pageIndex, err := strconv.Atoi(pageIndexStr)
	if err != nil {
		q.Err = fmt.Errorf("pageIndexStr 类型转化失败: %v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	if pageIndex < 1 {
		q.Err = errors.New("page 必须大于 0")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	pageSizeStr := queryParams.Get("page_size")
	if pageSizeStr == "" {
		pageSizeStr = "10"
	}
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil {
		q.Err = fmt.Errorf("pageSizeStr 类型转化失败: %v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	if pageSize < 1 || pageSize > 1000 {
		q.Err = errors.New("pageSize 必须在 1 和 1000 之间")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	examName := queryParams.Get("exam_name")
	examStatus := queryParams.Get("status")
	startTimeStr := queryParams.Get("start_time")
	endTimeStr := queryParams.Get("end_time")

	var startTime, endTime time.Time
	if startTimeStr != "" && endTimeStr != "" {
		startTimeInt64, err := strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			q.Err = fmt.Errorf("startTimeStr 类型转化失败: %v", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		endTimeInt64, err := strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			q.Err = fmt.Errorf("endTimeStr 类型转化失败: %v", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		startTime = time.UnixMilli(startTimeInt64)
		endTime = time.UnixMilli(endTimeInt64)
	}

	req := QueryMarkingListReq{
		User: &User{
			ID: q.SysUser.ID.Int64,
		},
		ExamName:  examName,
		Limit:     pageSize,
		Offset:    pageSize * (pageIndex - 1),
		StartTime: startTime,
		EndTime:   endTime,
		Status:    examStatus,
	}

	exams, rowCount, err := QueryExamList(ctx, req)
	if err != nil {
		q.Err = fmt.Errorf("查询考试列表失败: %v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	examsJson, err := json.Marshal(map[string]interface{}{"exam_list": exams})
	if forceErr == "json.Marshal-exams" {
		err = ForceErr
	}
	if err != nil {
		q.Err = fmt.Errorf("构造 examsJson 失败: %v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	q.Msg.Data = examsJson
	q.Msg.RowCount = int64(rowCount)
	q.Resp()
}

// 获取练习批改列表
func getPracticeList(ctx context.Context) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)

	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	method := strings.ToLower(q.R.Method)
	if method != "get" {
		q.Err = fmt.Errorf("不支持的 HTTP 方法: %s", q.R.Method)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	var readable bool

	if readable, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, nil, q.Ep.Path, auth_mgt.CAPIAccessActionRead); q.Err != nil {
		q.Err = fmt.Errorf("获取访问权限失败: %v", q.Err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	if !readable {
		q.Err = errors.New("无权访问")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	queryParams := q.R.URL.Query()

	pageIndexStr := queryParams.Get("page")
	if pageIndexStr == "" {
		pageIndexStr = "1"
	}
	pageIndex, err := strconv.Atoi(pageIndexStr)
	if err != nil {
		q.Err = fmt.Errorf("pageIndexStr 类型转化失败: %v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	if pageIndex < 1 {
		q.Err = errors.New("page 必须大于 0")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	pageSizeStr := queryParams.Get("page_size")
	if pageSizeStr == "" {
		pageSizeStr = "10"
	}
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil {
		q.Err = fmt.Errorf("pageSizeStr 类型转化失败: %v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	if pageSize < 1 || pageSize > 1000 {
		q.Err = errors.New("pageSize 必须在 1 和 1000 之间")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	practiceName := queryParams.Get("practice_name")

	req := QueryMarkingListReq{
		User: &User{
			ID: q.SysUser.ID.Int64,
		},
		PracticeName: practiceName,
		Limit:        pageSize,
		Offset:       pageSize * (pageIndex - 1),
	}

	practices, rowCount, err := QueryPracticeList(ctx, req)
	if err != nil {
		q.Err = fmt.Errorf("查询练习列表失败: %v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	practicesJson, err := json.Marshal(map[string]interface{}{"practice_list": practices})
	if forceErr == "json.Marshal-practices" {
		err = errors.New("")
	}
	if err != nil {
		q.Err = fmt.Errorf("构造 practicesJson 失败: %v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	q.Msg.Data = practicesJson
	q.Msg.RowCount = int64(rowCount)
	q.Resp()
}

// 考试查看/批改
func examSessionDetail(ctx context.Context) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)

	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	method := strings.ToLower(q.R.Method)

	switch method {
	case "get":
		var readable bool

		if readable, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, nil, q.Ep.Path, auth_mgt.CAPIAccessActionRead); q.Err != nil {
			q.Err = fmt.Errorf("获取访问权限失败: %v", q.Err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if !readable {
			q.Err = errors.New("无权访问")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		examSessionIDStr := q.R.URL.Query().Get("exam_session_id")

		if examSessionIDStr == "" {
			q.Err = errors.New("请求参数必须包含 考试场次ID")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		examSessionID, err := strconv.ParseInt(examSessionIDStr, 10, 64)
		if err != nil {
			q.Err = fmt.Errorf("examSessionIDStr 类型转化失败: %v", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		detail, err := getQuestionsAndStudentInfos(ctx, QueryCondition{
			TeacherID:     q.SysUser.ID.Int64,
			ExamSessionID: examSessionID,
		})
		if err != nil {
			q.Err = fmt.Errorf("获取问题和学生信息失败: %v", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		detailJson, err := json.Marshal(detail)
		if forceErr == "json.Marshal-detail" {
			err = ForceErr
		}
		if err != nil {
			q.Err = fmt.Errorf("构造 detailJson 失败 : %v", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Msg.Data = detailJson

	case "post":
		var editable bool

		if editable, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, nil, q.Ep.Path, auth_mgt.CAPIAccessActionUpdate); q.Err != nil {
			q.Err = fmt.Errorf("获取访问权限失败: %v", q.Err)
			q.RespErr()
			return
		}

		if !editable {
			q.Err = errors.New("无权访问")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		markedInfoJson, err := saveMarkingResults(ctx, q.SysUser.ID.Int64, q.R.Body)
		if err != nil {
			q.Err = fmt.Errorf("保存考试批改结果失败: %v", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Msg.Data = markedInfoJson

	default:
		q.Err = fmt.Errorf("不支持的 HTTP 方法: %s", q.R.Method)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	q.Resp()
}

// 练习查看/批改
func practiceDetail(ctx context.Context) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)

	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	method := strings.ToLower(q.R.Method)
	switch method {
	case "get":
		var readable bool

		if readable, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, nil, q.Ep.Path, auth_mgt.CAPIAccessActionRead); q.Err != nil {
			q.Err = fmt.Errorf("获取访问权限失败: %v", q.Err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if !readable {
			q.Err = errors.New("无权访问")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		queryParams := q.R.URL.Query()
		practiceIDStr := queryParams.Get("practice_id")

		if practiceIDStr == "" {
			q.Err = errors.New("请求参数缺少 练习ID")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		practiceID, err := strconv.ParseInt(practiceIDStr, 10, 64)
		if err != nil {
			q.Err = fmt.Errorf("practiceIDStr 类型转化失败: %v", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		detail, err := getQuestionsAndStudentInfos(ctx, QueryCondition{
			TeacherID:  q.SysUser.ID.Int64,
			PracticeID: practiceID,
		})
		if err != nil {
			q.Err = fmt.Errorf("获取问题和学生信息失败: %v", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		detailJson, err := json.Marshal(detail)
		if forceErr == "json.Marshal-detail" {
			err = ForceErr
		}
		if err != nil {
			q.Err = fmt.Errorf("构造 detailJson 失败: %v", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Msg.Data = detailJson

	case "post":
		var editable bool

		if editable, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, nil, q.Ep.Path, auth_mgt.CAPIAccessActionUpdate); q.Err != nil {
			q.Err = fmt.Errorf("获取访问权限失败: %v", q.Err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if !editable {
			q.Err = errors.New("无权访问")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		markedInfoJson, err := saveMarkingResults(ctx, q.SysUser.ID.Int64, q.R.Body)
		if err != nil {
			q.Err = fmt.Errorf("保存练习批改结果失败: %v", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Msg.Data = markedInfoJson

	default:
		q.Err = fmt.Errorf("不支持的 HTTP 方法: %s", q.R.Method)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	q.Resp()
}

// 考试场次提交
func examSessionSubmission(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	method := strings.ToLower(q.R.Method)
	if method != "patch" {
		q.Err = fmt.Errorf("不支持的 HTTP 方法: %s", method)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	// 权限校验（核分员才能提交）
	var editable bool

	if editable, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, nil, q.Ep.Path, auth_mgt.CAPIAccessActionUpdate); q.Err != nil {
		q.Err = fmt.Errorf("获取访问权限失败: %v", q.Err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	if !editable {
		q.Err = errors.New("无权访问")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	examSessionIDStr := q.R.URL.Query().Get("exam_session_id")

	if examSessionIDStr == "" {
		q.Err = fmt.Errorf("请求参数必须包含 考试场次ID")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	examSessionID, err := strconv.ParseInt(examSessionIDStr, 10, 64)
	if err != nil {
		q.Err = fmt.Errorf("examSessionIDStr 类型转化失败: %v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	err = submitResult(ctx, QueryCondition{
		TeacherID:     q.SysUser.ID.Int64,
		ExamSessionID: examSessionID,
	})
	if err != nil {
		q.Err = fmt.Errorf("提交考试失败: %v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	q.Resp()
}

// 练习学生提交
func practiceStudentSubmission(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	method := strings.ToLower(q.R.Method)
	if method != "patch" {
		q.Err = fmt.Errorf("不支持的 HTTP 方法: %s", method)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	// 权限校验（核分员才能提交）
	var editable bool

	if editable, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, nil, q.Ep.Path, auth_mgt.CAPIAccessActionUpdate); q.Err != nil {
		q.Err = fmt.Errorf("获取访问权限失败: %v", q.Err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	if !editable {
		q.Err = errors.New("无权访问")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	queryParams := q.R.URL.Query()

	practiceIDStr := queryParams.Get("practice_id")
	practiceSubmissionIDStr := queryParams.Get("practice_submission_id")

	if practiceIDStr == "" {
		q.Err = fmt.Errorf("请求参数必须包含 练习ID")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	practiceID, err := strconv.ParseInt(practiceIDStr, 10, 64)
	if err != nil {
		q.Err = fmt.Errorf("practiceIDStr 类型转化失败: %v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	if practiceSubmissionIDStr == "" {
		q.Err = fmt.Errorf("请求参数必须包含 练习提交ID")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	practiceSubmissionID, err := strconv.ParseInt(practiceSubmissionIDStr, 10, 64)
	if err != nil {
		q.Err = fmt.Errorf("practiceSubmissionIDStr 类型转化失败: %v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	err = submitResult(ctx, QueryCondition{
		TeacherID:            q.SysUser.ID.Int64,
		PracticeID:           practiceID,
		PracticeSubmissionID: practiceSubmissionID,
	})
	if err != nil {
		q.Err = fmt.Errorf("提交练习学生失败: %v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	q.Resp()
}

// 根据场次/练习+考生id获取学生的答案和批改结果
func getStudentAnswersAndMarkResults(ctx context.Context) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)

	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	method := strings.ToLower(q.R.Method)
	if method != "get" {
		q.Err = fmt.Errorf("不支持的 HTTP 方法: %s", method)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	var readable bool

	if readable, q.Err = auth_mgt.CheckUserAPIAccessible(ctx, nil, q.Ep.Path, auth_mgt.CAPIAccessActionRead); q.Err != nil {
		q.Err = fmt.Errorf("获取权限失败: %v", q.Err)
		q.RespErr()
		return
	}

	if !readable {
		q.Err = errors.New("无权访问")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	queryParams := q.R.URL.Query()

	examSessionIDStr := queryParams.Get("exam_session_id")
	practiceIDStr := queryParams.Get("practice_id")

	if examSessionIDStr == "" && practiceIDStr == "" {
		q.Err = errors.New("请求参数必须包含 练习ID 或者 考试场次ID 中的一个")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	if practiceIDStr != "" && examSessionIDStr != "" {
		q.Err = errors.New("请求参数不能同时包含 练习ID 和 考试场次ID")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	var (
		examSessionID, examineeID, practiceID, practiceSubmissionID int64
		err                                                         error
	)

	if examSessionIDStr != "" {
		examSessionID, err = strconv.ParseInt(examSessionIDStr, 10, 64)
		if err != nil {
			q.Err = fmt.Errorf("examSessionIDStr 类型转化失败: %v", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		examineeIDStr := queryParams.Get("examinee_id")
		if examineeIDStr == "" {
			q.Err = errors.New("缺少 考生ID")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		examineeID, err = strconv.ParseInt(examineeIDStr, 10, 64)
		if err != nil {
			q.Err = fmt.Errorf("examineeIDStr 类型转化失败: %v", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
	}

	if practiceIDStr != "" {
		practiceID, err = strconv.ParseInt(practiceIDStr, 10, 64)
		if err != nil {
			q.Err = fmt.Errorf("practiceIDStr 类型转化失败: %v", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		practiceSubmissionIDStr := queryParams.Get("practice_submission_id")
		if practiceSubmissionIDStr == "" {
			q.Err = errors.New("缺少 练习提交ID")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		practiceSubmissionID, err = strconv.ParseInt(practiceSubmissionIDStr, 10, 64)
		if err != nil {
			q.Err = fmt.Errorf("practiceSubmissionIDStr 类型转化失败: %v", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
	}

	cond := QueryCondition{
		TeacherID:            q.SysUser.ID.Int64,
		ExamSessionID:        examSessionID,
		ExamineeID:           examineeID,
		PracticeID:           practiceID,
		PracticeSubmissionID: practiceSubmissionID,
	}

	// 1. 获取批改信息，得到mark_mode，据此判断需要获取哪一部分数据
	markerInfo, err := QueryMarkerInfo(ctx, cond)
	if err != nil {
		q.Err = fmt.Errorf("查询批改信息失败: %v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	markInfoLen := len(markerInfo.MarkInfos)
	if markInfoLen != 1 {
		q.Err = fmt.Errorf("批改配置信息错误, 正常情况应为一条配置信息, 实际为 %d", markInfoLen)
		z.Error(q.Err.Error(), zap.Any("markerInfo.MarkInfos", markerInfo.MarkInfos))
		q.RespErr()
		return
	}

	// 自定义，用于区别考试（数据库表对应字段没有这个）
	if cond.PracticeSubmissionID > 0 {
		// 查询单个练习学生
		markerInfo.MarkMode = "12" // 批改单个练习学生
	}

	// TODO 检查

	// 2. 获取学生主观题答案
	studentAnswers, err := QueryStudentAnswers(ctx, nil, "02", cond, markerInfo)
	if err != nil {
		q.Err = fmt.Errorf("获取学生主观题答案失败: %v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	// 3. 获取老师对该学生的批改记录
	markingResults, err := QuerySubjectiveQuestionsMarkingResults(ctx, cond)
	if err != nil {
		q.Err = fmt.Errorf("获取老师对该学生的批改记录失败: %v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	jsonData, err := json.Marshal(map[string]interface{}{
		"student_answers": studentAnswers,
		"marking_results": markingResults,
	})
	if forceErr == "json.Marshal-map" {
		err = ForceErr
	}
	if err != nil {
		q.Err = fmt.Errorf("构造 jsonData 失败: %v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	q.Msg.Data = jsonData
	q.Resp()
}

// --------------------- AI批改 -----------------------

// 自动批改处理函数，学生点击提交就会触发，导致有的学生的考卷会立即得到分数，这时候体现出“考试成绩管理‘提交’”的作用（考试必须全部批改之后才能看到成绩，练习是无所谓的） // TODO 错误需要接收然后将批改状态变为“异常”
func AutoMark(ctx context.Context, cond QueryCondition) (err error) { // TODO 缺考的人直接批阅为0
	forceErr, _ := ctx.Value(ForceErrKey).(string)

	if cond.ExamSessionID <= 0 && cond.PracticeID <= 0 {
		err = errors.New("exam session id / practice id 必须存在一个")
		z.Error(err.Error())
		return
	}

	if cond.ExamSessionID > 0 && cond.PracticeID > 0 {
		err = errors.New("exam session id 和 practice id 不能同时存在")
		z.Error(err.Error())
		return
	}

	pgxConn := cmn.GetPgxConn()
	tx, err := pgxConn.Begin(ctx)
	if err != nil || forceErr == "AutoMark-pgxConn.Begin" {
		err = fmt.Errorf("开启事务失败: %v", err)
		z.Error(err.Error())
		return
	}

	defer func() {
		if err == nil {
			if err = tx.Commit(ctx); err != nil {
				err = fmt.Errorf("事务提交失败: %v", err)
				z.Error(err.Error())
			}
			return
		}

		if rerr := tx.Rollback(ctx); rerr != nil {
			err = fmt.Errorf("%v; 回滚失败: %w", err, rerr)
			z.Error(err.Error())
		}
	}()

	var (
		questionStrKeyPrefix string
		suffixId             int64
	)

	if cond.ExamSessionID > 0 {
		questionStrKeyPrefix = "exam_session"
		suffixId = cond.ExamSessionID
	} else {
		questionStrKeyPrefix = "practice"
		suffixId = cond.PracticeID
	}

	// 组合成最终 key
	questionStrKey := fmt.Sprintf("%s:%d", questionStrKeyPrefix, suffixId)

	// 自动批改客观题，可以自己一个事务
	// 不准备 批改教师id
	err = markObjectiveQuestionAnswers(ctx, tx, cond, questionStrKey)
	if err != nil {
		err = fmt.Errorf("自动批改客观题失败: %v", err)
		z.Error(err.Error())
		return
	}

	markerInfo, err := QueryMarkerInfo(ctx, cond)
	if err != nil {
		err = fmt.Errorf("查询批改配置失败: %v", err)
		z.Error(err.Error())
		return
	}

	// TODO 目前，实现的是，考试、练习的自动批改和练习错题的批改，都是学生点击“提交”后触发的；
	// TODO 先实现这种，后面实现“考完试之后统一进行ai批改，一次一题多学生，保证准确率”
	// 自动批改 / 错题批改
	if markerInfo.MarkMode != "00" && cond.PracticeWrongSubmissionID <= 0 {
		z.Info(fmt.Sprintf("本次批改不是自动批改模式，不会直接提交，mark_mode 为 %s, practice_wrong_submission_id 为 %d", markerInfo.MarkMode, cond.PracticeWrongSubmissionID))
		return nil
	}

	z.Info("开始进行主观题 ai批改")

	// 等于1是因为，配置批改信息的时候，就只有一行记录（HandleMarkerInfo）
	if len(markerInfo.MarkInfos) != 1 {
		err = errors.New("批改配置信息错误")
		z.Error(err.Error(), zap.Any("markerInfo.MarkInfos", markerInfo.MarkInfos))
		return
	}

	// 认为，考试 / 练习的发布者是批改人
	cond.TeacherID = markerInfo.MarkInfos[0].MarkTeacherID.Int64

	// 2. 查询出所有主观题
	questionsStr, err := getSubjectQuestionsFromRedis(ctx, tx, cond, questionStrKey)
	if err != nil {
		err = fmt.Errorf("从 redis 中获取主观题失败: %v", err)
		z.Error(err.Error())
		return
	}

	var questionSets []QuestionSet
	if err = json.Unmarshal([]byte(questionsStr), &questionSets); err != nil {
		err = fmt.Errorf("解析 questionsStr json 失败: %v", err)
		z.Error(err.Error())
		return
	}

	var questions []*cmn.TExamPaperQuestion
	for _, questionSet := range questionSets {
		questions = append(questions, questionSet.Questions...)
	}

	// 练习、练习错题都是使用这个 id，来查询学生的答案
	if cond.PracticeSubmissionID > 0 {
		markerInfo.MarkMode = "12"
	}

	// 3. 查出当前学生的所有答案（练习错题下，只需要获取错题）
	studentAnswers, err := QueryStudentAnswers(ctx, nil, "02", cond, markerInfo)
	if err != nil {
		err = fmt.Errorf("查询当前考生所有的答案失败: %v", err)
		z.Error(err.Error())
		return
	}

	// TODO 如果学生答案为空，这道作答直接给0分

	// TODO 事务怎么决定？

	// 4. 开启 ai 批改（自己一个事务，错误重试，超出重试次数标记异常）
	// 给出所有问题，以及该学生的所有的答案
	// 每一位学生都会开启 ai批改
	if err = generateAIMarkTask(ctx, cond, questions, studentAnswers); err != nil {
		err = fmt.Errorf("启用 ai 批改 失败: %v", err)
		z.Error(err.Error())
		return
	}

	return nil
}

// 自动批改客观题 （不记录批改教师，因为存在多个教师的情况，多人批改）
func markObjectiveQuestionAnswers(ctx context.Context, tx pgx.Tx, cond QueryCondition, questionStrKey string) error {
	z.Info("开始进行客观题批改")

	forceErr, _ := ctx.Value(ForceErrKey).(string)

	markerInfo := MarkerInfo{
		//MarkerID: cond.TeacherID,
	}

	if cond.ExamineeID > 0 {
		// 查询单个考试考生
		markerInfo.MarkMode = "04"
	}

	if cond.PracticeSubmissionID > 0 {
		// 查询单个练习学生
		markerInfo.MarkMode = "12" // 规定
	}

	studentAnswers, err := QueryStudentAnswers(ctx, tx, "00", cond, markerInfo)
	if err != nil {
		return fmt.Errorf("查询学生答案失败: %v", err)
	}

	// sa表中没有存在学生答案的记录（这种情况不存在，正常情况必定会有记录，即使没有作答）
	if len(studentAnswers) <= 0 {
		z.Info("该学生没有作答")
		return nil
	}

	marks := make([]cmn.TMark, len(studentAnswers))
	for i, studentAnswer := range studentAnswers {
		mark := cmn.TMark{
			//TeacherID:            null.IntFrom(cond.TeacherID), // 客观题不标注批改者
			QuestionID:           studentAnswer.QuestionID,
			ExamineeID:           studentAnswer.ExamineeID,
			ExamSessionID:        studentAnswer.ExamSessionID,
			PracticeID:           studentAnswer.PracticeID,
			PracticeSubmissionID: studentAnswer.PracticeSubmissionID,
			Status:               null.StringFrom("00"),
			Creator:              null.IntFrom(cond.TeacherID),
			UpdatedBy:            null.IntFrom(cond.TeacherID),
			CreateTime:           null.IntFrom(time.Now().UnixMilli()),
			UpdateTime:           null.IntFrom(time.Now().UnixMilli()),
			Score:                null.FloatFrom(0),
		}

		var standardAnswers []string

		err = json.Unmarshal(studentAnswer.ActualAnswers, &standardAnswers)
		if err != nil {
			return fmt.Errorf("解析 studentAnswer.ActualAnswers 失败: %v", err)
		}

		if len(standardAnswers) <= 0 {
			return fmt.Errorf("无效的标准答案")
		}

		var answers struct {
			Answer []string `json:"answer"`
		}

		if len(studentAnswer.Answer) != 0 {
			if err = json.Unmarshal(studentAnswer.Answer, &answers); err != nil {
				return fmt.Errorf("解析 studentAnswer.Answer 失败: %v", err)
			}

			// 对比答案，正确给满分
			if compareSlices(answers.Answer, standardAnswers) {
				mark.Score = null.FloatFrom(studentAnswer.QuestionScore.Float64)
			}
		} else { // 没有答案直接0分
			mark.Score = null.FloatFrom(0) // 应对空值
		}

		markDetails := make([]MarkDetail, 0, 10)

		// 客观题只有一小题，只需要赋值 score
		markDetails = append(markDetails, MarkDetail{Score: mark.Score.Float64})

		mark.MarkDetails, err = json.Marshal(markDetails)
		if err != nil || forceErr == "markObjectiveQuestionAnswers-json.Marshal" {
			return fmt.Errorf("构造 mark.MarkDetail Json 失败 : %v", err)
		}

		marks[i] = mark
	}

	_, err = UpdateStudentAnswerScore(ctx, tx, marks, cond)
	if err != nil {
		return fmt.Errorf("保存学生批改的分数失败: %v", err)
	}

	questionsStr, err := getSubjectQuestionsFromRedis(ctx, tx, cond, questionStrKey)
	if err != nil {
		return fmt.Errorf("从 redis 中获取主观题失败: %v", err)
	}

	var questions []QuestionSet
	if err = json.Unmarshal([]byte(questionsStr), &questions); err != nil {
		return fmt.Errorf("解析 questionsStr json 失败: %v", err)
	}

	if len(questions) != 0 {
		z.Info("试卷存在主观题，不可直接提交")
		return nil
	}

	// 没有主观题，可继续自动提交
	z.Info("试卷没有主观题，开始执行提交操作")

	if cond.PracticeWrongSubmissionID > 0 {
		// 更新批改错题集练习的提交状态
		_, err = UpdatePracticeWrongSubmissionStatus(ctx, tx, cond.TeacherID, []int64{cond.PracticeWrongSubmissionID}, PracticeWrongSubmissionStatusSubmitted)
		if err != nil {
			return fmt.Errorf("更新批改错题集练习的提交状态失败： %v", err)
		}
	} else {
		err = updateToSubmissionStatus(ctx, tx, cond)
		if err != nil {
			return fmt.Errorf("更新 考试/练习 提交状态失败: %v", err)
		}
	}

	return nil
}

// 从 redis 中获取试卷主观题
func getSubjectQuestionsFromRedis(ctx context.Context, tx pgx.Tx, cond QueryCondition, questionStrKey string) (string, error) {
	redisClient := cmn.GetRedisConn()

	questionsStr, err := redisClient.Get(ctx, questionStrKey).Result()
	if errors.Is(err, redis.Nil) {
		z.Sugar().Infof("%s key 不存在，需要从数据库中查询", questionStrKey)

		questions, err := QuerySubjectiveQuestions(ctx, tx, cond, MarkerInfo{})
		if err != nil {
			return "", fmt.Errorf("查询试卷主观题失败: %v", err)
		}

		questionsJson, err := json.Marshal(questions)
		if err != nil {
			return "", fmt.Errorf("构造 questionsJson 失败: %v", err)
		}

		questionsStr = string(questionsJson)

		// 空数组照样存储
		err = redisClient.Set(ctx, questionStrKey, questionsStr, 5*time.Minute).Err()
		if err != nil {
			return "", fmt.Errorf("在 redis 中设置 %s key 失败: %v", questionStrKey, err)
		}

	} else if err != nil {
		return "", fmt.Errorf("从 redis 中获取 %s key 失败: %v", questionStrKey, err)
	} else {
		// key 存在，每次访问延长 TTL
		if err = redisClient.Expire(ctx, questionStrKey, 2*time.Minute).Err(); err != nil {
			return "", fmt.Errorf("刷新 %s key TTL 失败: %v", questionStrKey, err)
		}
	}

	return questionsStr, nil
}

// 派发任务
func generateAIMarkTask(ctx context.Context, cond QueryCondition, questions []*cmn.TExamPaperQuestion, studentAnswers []cmn.TVStudentAnswerQuestion) error {
	var err error

	answersTaskNeed := make([]*ai_mark.StudentAnswer, len(studentAnswers))

	// 构造 answersTaskNeed 数组
	for i, studentAnswer := range studentAnswers {
		questionID := studentAnswer.QuestionID.Int64

		if questionID <= 0 {
			return errors.New("学生答案 题目ID 无效")
		}

		if studentAnswer.ExamineeID.Int64 <= 0 && studentAnswer.PracticeSubmissionID.Int64 <= 0 {
			return errors.New("学生答案 考生ID 和 提交ID 不能同时为空")
		}

		if cond.ExamSessionID > 0 && studentAnswer.ExamineeID.Int64 <= 0 {
			return errors.New("学生答案 考生ID 无效")
		}

		if cond.PracticeID > 0 && studentAnswer.PracticeSubmissionID.Int64 <= 0 {
			return errors.New("学生答案 提交ID 无效")
		}

		// 学生没有作答该题目
		if studentAnswer.Answer == nil || len(studentAnswer.Answer) == 0 {
			studentAnswer.Answer = []byte(`{"answer": []}`)
		}

		var answers struct {
			Answer []string `json:"answer"`
		}

		err = json.Unmarshal(studentAnswer.Answer, &answers)
		if err != nil {
			return fmt.Errorf("解析学生答案失败: %v， 原始 json: %v", err, string(studentAnswer.Answer))
		}

		// 构造答案文本
		// Answers: { answer: ["A", "C", "光合作用"] }
		/*
		  （1）A
		  （2）C
		  （3）光合作用
		*/
		var studentAnswersBuilder strings.Builder
		for i, answer := range answers.Answer {
			studentAnswersBuilder.WriteString(fmt.Sprintf("（%d）", i+1))
			studentAnswersBuilder.WriteString(answer)
			studentAnswersBuilder.WriteString("\n")
		}

		answersTaskNeed[i] = &ai_mark.StudentAnswer{
			QuestionID: questionID,
			Answer:     studentAnswersBuilder.String(),
		}
	}

	questionsTaskNeed := make([]*ai_mark.QuestionDetail, len(questions))

	// 构造 questionsTaskNeed  数组
	for i, q := range questions {
		questionID := q.ID.Int64
		if questionID <= 0 {
			return errors.New("题目ID 无效")
		}

		var standardAnswers []*SubjectiveAnswer
		err = json.Unmarshal(q.Answers, &standardAnswers)
		if err != nil {
			return fmt.Errorf("解析 q.Answers 失败: %v", err)
		}

		if len(standardAnswers) == 0 {
			return fmt.Errorf("没有找到问题的标准答案， 题目ID 为 %d", q.ID.Int64)
		}

		// 构造标准答案与批改规则
		//[
		//	{
		//		Index: 1,
		//		Answer: "事件循环是JavaScript处理异步操作的机制。",
		//		GradingRule: "出现“事件循环”得2分",
		//	},
		//	{
		//		...
		//	}
		//]
		/*
			（1）事件循环是JavaScript处理异步操作的机制

			（1）出现“事件循环”得2分
		*/
		var answerBuilder, ruleBuilder strings.Builder
		for _, a := range standardAnswers {
			indexStr := fmt.Sprintf("（%d）", a.Index)

			answerBuilder.WriteString(indexStr)
			answerBuilder.WriteString(a.Answer) // TODO 备选答案没有加上去
			answerBuilder.WriteString("\n")

			ruleBuilder.WriteString(indexStr)
			ruleBuilder.WriteString(a.GradingRule)
			ruleBuilder.WriteString("\n")
		}

		questionsTaskNeed[i] = &ai_mark.QuestionDetail{
			ID:     questionID,
			Score:  q.Score.Float64,
			Answer: answerBuilder.String(),
			Rule:   ruleBuilder.String(),
		}
	}

	// TODO 如果超出 token 数,分批
	var aiMarkRequests []AIMarkRequest

	aiMarkRequest := AIMarkRequest{
		QuestionDetails: questionsTaskNeed,
		StudentAnswers:  answersTaskNeed,
	}

	aiMarkRequests = append(aiMarkRequests, aiMarkRequest)

	asynqOpt := []asynq.Option{
		asynq.MaxRetry(5),              // 任务首次执行失败后，会自动重试 5 次（总计执行 6 次：1 次初始 + 5 次重试）。
		asynq.Unique(24 * time.Hour),   // 设置唯一性约束：24小时内，不允许重复入队，基于任务的 Type（任务类型）和 Payload（任务数据）的哈希值。如果两个任务类型相同且 payload 完全一致，则视为 “相同任务”
		asynq.Timeout(5 * time.Minute), //若任务处理器（Handler）在 5 分钟内未完成执行（未返回 nil 或错误），Asynq 会强制将任务标记为 “失败”，并触发重试（受 MaxRetry 限制）
		asynq.Queue("critical"),        // 指定要进入的队列，默认 default 队列
	}

	//uniqueTaskCountKey := TaskTypeAIMarkRequest // 存放的是这次任务的总批数 // 该考生被分为多少次任务（由于 token 有限）
	//if cond.ExamSessionID > 0 {
	//	uniqueTaskCountKey += fmt.Sprintf(":exam_session:%d:examinee:%d:count", cond.ExamSessionID, cond.ExamineeID)
	//} else {
	//	uniqueTaskCountKey += fmt.Sprintf(":practice:%d:practice_submission:%d:count", cond.PracticeID, cond.PracticeSubmissionID)
	//}
	//
	//redisClient := cmn.GetRedisConn()
	//err = redisClient.Set(ctx, uniqueTaskCountKey, len(aiMarkRequests), 7*24*time.Hour).Err() // 一星期

	// TODO 需要判断这次消耗的 token 是否超出限制；因为超出的token会被截断丢弃，发送错乱

	// 分批添加任务
	taskTotalCount := len(aiMarkRequests)

	for _, aiMarkRequest := range aiMarkRequests {
		if err = cmn.SendTask(TaskTypeAIMarkRequest, AIMarkTaskPayLoad{
			AIMarkRequest:  aiMarkRequest,
			QueryCondition: cond,
			TaskTotalCount: &taskTotalCount,
			CountMu:        &sync.Mutex{},
		}, asynqOpt...); err != nil {
			return fmt.Errorf("向 asyncq 发送任务失败: %v", err)
		}
	}

	return nil
}

// 处理任务
func handleAIMarkTask(ctx context.Context, task *asynq.Task) (err error) {
	z.Sugar().Infof("handleAIMarkTask func 开始执行")

	forceErr, _ := ctx.Value(ForceErrKey).(string)

	var taskPayLoad cmn.TaskPayload

	err = json.Unmarshal(task.Payload(), &taskPayLoad)
	if err != nil {
		return fmt.Errorf("解析 task.Payload() 失败: %v", err)
	}

	taskPayLoadDataJson, err := json.Marshal(taskPayLoad.Data)
	if err != nil || forceErr == "handleAIMarkTask-json.Marshal-payload-data" {
		return fmt.Errorf("构造 taskPayLoadDataJson 失败: %v", err)
	}

	var payloadData AIMarkTaskPayLoad
	err = json.Unmarshal(taskPayLoadDataJson, &payloadData)
	if err != nil {
		return fmt.Errorf("解析 taskPayLoadDataJson 失败: %v, 原始 json: %+v", err, string(taskPayLoadDataJson))
	}

	z.Sugar().Infof("handleAIMarkTask data->: %+v", payloadData)

	if *payloadData.TaskTotalCount <= 0 {
		return fmt.Errorf("任务 payload 中的任务总数错误, 至少为 1")
	}

	cond := payloadData.QueryCondition

	if cond.TeacherID <= 0 {
		return fmt.Errorf("任务 payload 中 的教师ID 无效")
	}

	if cond.ExamSessionID <= 0 && cond.PracticeID <= 0 {
		return fmt.Errorf("任务 payload 中的 考试场次ID 与 练习ID 都无效")
	}

	if cond.ExamSessionID > 0 && cond.PracticeID > 0 {
		return fmt.Errorf("任务 payload 中的 考试场次ID 与 练习ID 不能同时有效")
	}

	chatModel, err := ai_mark.GetChatModel()
	if err != nil {
		return fmt.Errorf("获取 chatModel 失败:%v", err)
	}

	respContent, err := chatModel.AIMark(ctx, payloadData.AIMarkRequest.QuestionDetails, payloadData.AIMarkRequest.StudentAnswers, chatModel.SendChatCompletions)
	if err != nil {
		return fmt.Errorf("大模型批改失败: %v", err)
	}

	z.Sugar().Infof("chatResp: %+v", respContent)

	markLen := len(respContent.MarkResults)

	pgxConn := cmn.GetPgxConn()
	tx, err := pgxConn.Begin(ctx)
	if err != nil || forceErr == "pgxConn.Begin" {
		return fmt.Errorf("事务开启错误: %w", err)
	}

	defer func() {
		if err == nil {
			if err = tx.Commit(ctx); err != nil {
				err = fmt.Errorf("事务提交失败: %v", err)
				z.Error(err.Error())
			}
			return
		}

		if rerr := tx.Rollback(ctx); rerr != nil {
			err = fmt.Errorf("%v; 回滚失败: %w", err, rerr)
			z.Error(err.Error())
		}
	}()

	// 从负载中获取问题即可
	questions := payloadData.AIMarkRequest.QuestionDetails

	// question_id --> score
	questionScoreMap := make(map[int64]float64, markLen)
	for _, question := range questions {
		questionScoreMap[question.ID] = question.Score
	}

	marks := make([]cmn.TMark, 0, markLen)

	// 遍历 ai 返回的批改结果，检验分数并构造 marks 数组
	for _, result := range respContent.MarkResults {
		aiMarkScore := result.Score
		questionScore := questionScoreMap[result.QuestionID]

		// 分数校验
		if aiMarkScore > questionScore {
			return fmt.Errorf("ai 所批分数超出题目的分数，ai所给分数: %f，题目分数: %f", aiMarkScore, questionScore)
		}

		markDetail := MarkDetail{
			Index:   1, // TODO 不管多少小问都是写在一个
			Score:   aiMarkScore,
			Analyze: result.Analyze,
		}

		var markDetailsJSON []byte
		markDetailsJSON, err = json.Marshal([]MarkDetail{markDetail}) // 构造数组
		if err != nil || forceErr == "handleAIMarkTask-json.Marshal-mark-details" {
			return fmt.Errorf("构造 markDetailsJSON 失败: %v", err)
		}

		mark := cmn.TMark{
			TeacherID:   null.IntFrom(cond.TeacherID), // 发布考试的老师
			QuestionID:  null.IntFrom(result.QuestionID),
			Score:       null.FloatFrom(result.Score),
			MarkDetails: markDetailsJSON,
			Creator:     null.IntFrom(cond.TeacherID),
			CreateTime:  null.IntFrom(time.Now().UnixMilli()),
		}

		if cond.ExamSessionID > 0 {
			// 考试会话
			mark.ExamSessionID = null.IntFrom(cond.ExamSessionID)
			mark.ExamineeID = null.IntFrom(cond.ExamineeID)
		} else {
			// 练习会话
			mark.PracticeID = null.IntFrom(cond.PracticeID)
			mark.PracticeSubmissionID = null.IntFrom(cond.PracticeSubmissionID)
		}

		marks = append(marks, mark)
	}

	// 记录 ai批改 的过程
	_, err = UpsertMarkingResults(ctx, tx, marks)
	if err != nil {
		return fmt.Errorf("ai批改，记录批改过程失败: %v", err)
	}

	// 将 ai批改 得分结果写入 sa表
	_, err = UpdateStudentAnswerScore(ctx, tx, marks, cond)
	if err != nil {
		return fmt.Errorf("ai批改，保存批改后的分数失败: %v", err)
	}

	// 如果是最后一个任务, 更新为“已批改”

	// 将计数器锁住
	payloadData.CountMu.Lock()
	defer payloadData.CountMu.Unlock()

	if *payloadData.TaskTotalCount == 1 {
		switch {
		// 如果是错题
		case cond.PracticeWrongSubmissionID > 0:
			_, err = UpdatePracticeWrongSubmissionStatus(ctx, tx, payloadData.QueryCondition.TeacherID, []int64{payloadData.QueryCondition.PracticeWrongSubmissionID}, PracticeWrongSubmissionStatusSubmitted)
			if err != nil {
				err = fmt.Errorf("ai 批改完成之后，练习错题状态无法更改: %v", err)
				z.Error(err.Error())
				return
			}
			z.Info("更新练习错题状态成功 update state success")

		// 如果是考试
		case cond.ExamSessionID > 0:
			redisClient := cmn.GetRedisConn()

			examSessionMStatusUpdateKey := fmt.Sprintf("exam_session:%dstate_update", cond.ExamSessionID)
			exists, rerr := redisClient.Exists(ctx, examSessionMStatusUpdateKey).Result()
			if rerr != nil {
				err = fmt.Errorf("检查定时任务状态失败: %v", err)
				return
			}

			// 如果不存在
			if exists == 0 {
				// 先查询距离结束的时间x,然后开启定时任务(x时间之后开启),内容是,定时检查考试状态,合理后,查询待批改数,为0时可以进行更新考试状态

				// 1. 查询考试结束时间 x
				var endTime int64
				queryExamSessionSql := `SELECT end_time FROM t_exam_session WHERE id = $1`
				err = tx.QueryRow(ctx, queryExamSessionSql, cond.ExamSessionID).Scan(&endTime)
				if err != nil {
					if errors.Is(err, pgx.ErrNoRows) {
						return fmt.Errorf("查询不到考试场次 %d : %v", cond.ExamSessionID, err)
					} else {
						return fmt.Errorf("获取考试场次 %d 的 结束时间失败: %v", cond.ExamSessionID, err)
					}
				}

				// 2.开启一个 x 时间之后的周期性任务
				offsetTime := int64(5000) // 先让考试变为"批改中"
				delay := time.Duration(endTime+offsetTime-time.Now().UnixMilli()) * time.Millisecond
				startExamSessionStatusCheck(context.Background(), cond.ExamSessionID, cond.TeacherID, delay)

				// 生成一个 kv
				if err = redisClient.Set(ctx, examSessionMStatusUpdateKey, 1, 7*24*time.Hour).Err(); err != nil {
					return fmt.Errorf("设置 examSessionMStatusUpdateKey 失败: %v", err)
				}
			} else {
				z.Info("定时任务已存在, 无需重复调度")
			}

		// 如果是练习
		case cond.PracticeID > 0:
			_, err = UpdateExamSessionOrPracticeSubmissionStatus(ctx, tx, cond.TeacherID, nil, []int64{cond.PracticeSubmissionID}, PracticeSubmissionStatusSubmitted)
			if err != nil {
				return fmt.Errorf("ai 批改完成之后，练习状态无法更改: %v", err)
			}
			z.Info("更新练习状态成功 update state success")
		}

	} else {
		*payloadData.TaskTotalCount--
		z.Info("有一位同学批改完毕")
	}

	return nil
}

// 每次重试都是从这里开始
func taskMiddleware(handler func(ctx context.Context, task *asynq.Task) error) func(ctx context.Context, task *asynq.Task) error {
	return func(ctx context.Context, task *asynq.Task) error {
		// 限流器
		limiter := getAIMarkTaskLimiter()
		z.Info("等待从限流器中获取信号量")
		err, releaseFunc := limiter.Wait(ctx)
		if err != nil {
			return fmt.Errorf("获取信号量失败: %v", err)
		}
		defer releaseFunc() // 确保函数退出时释放信号量
		z.Info("获取信号量成功，限流等待结束")

		// 熔断器
		cb := getAIMarkTaskCB()
		_, err = cb.Execute(func() (any, error) {
			return nil, handler(ctx, task) // 这里执行 handler，处理任务。返回结果时会记录在熔断器中，用于更改熔断器的状态
		})
		if err != nil {
			err = fmt.Errorf("执行任务失败: %v", err)
			z.Error(err.Error())
			return err
		}

		return nil
	}
}

// 初始化限制器和熔断器
func initAIMarkTaskLimiterAndBreaker() {
	maxConcurrency := viper.GetInt("chatModel.maxConcurrency")
	if maxConcurrency <= 0 {
		z.Warn("从配置文件获取 chatModel.maxConcurrency 失败")
		maxConcurrency = 50 // 默认 50
	}
	aiMarkTaskLimiter = cmn.NewRateLimiterTaskRunner(int64(maxConcurrency), 20, 40)

	st := gobreaker.Settings{
		Name:         "AI Mark breaker",
		MaxRequests:  10,               // 半开状态允许的最大请求数，根据尝试结果决定是否恢复闭合，或者继续打开
		Timeout:      15 * time.Second, // 打开状态的持续时间，超出变为“半开”
		Interval:     24 * time.Hour,   // 连续失败计数的重置周期
		BucketPeriod: 1 * time.Hour,    // 统计滑动窗口的时间间隔

		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			// 熔断条件：请求数 >= 5 && (失败率 >= 50% || 连续失败次数 >= 3)
			return counts.Requests >= 5 && (failureRatio >= 0.5 || counts.ConsecutiveFailures >= 3)
		},

		// 监控熔断状态变化
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			z.Sugar().Warnf(" 熔断器 %s 的状态从 %s 变为 %s\n", name, from, to)

			if to == gobreaker.StateOpen {
				z.Sugar().Warnf(" 熔断器 %s 已经打开\n", name)
			}
		},
	}

	aiMarkTaskCB = gobreaker.NewCircuitBreaker[any](st)
}

// 获取全局限流器
func getAIMarkTaskLimiter() *cmn.RateLimiterTaskRunner {
	if aiMarkTaskLimiter == nil {
		initAIMarkTaskLimiterAndBreaker()
	}
	return aiMarkTaskLimiter
}

// 获取全局熔断器
func getAIMarkTaskCB() *gobreaker.CircuitBreaker[any] {
	if aiMarkTaskCB == nil {
		initAIMarkTaskLimiterAndBreaker()
	}
	return aiMarkTaskCB
}

// 定时任务用以更新考试状态"已批改"
func startExamSessionStatusCheck(ctx context.Context, examSessionID, teacherID int64, initialDelay time.Duration) {
	// 在 Goroutine 中运行定时任务，避免阻塞
	go func() {
		// 初始延迟
		z.Info("等待初始延迟",
			zap.Int64("exam_session_id", examSessionID),
			zap.Duration("initial_delay", initialDelay))

		if initialDelay < 0 {
			z.Warn("延迟时长为负数，将跳过休眠")
			initialDelay = 0 // 显式置为 0
		}

		time.Sleep(initialDelay)

		pgxConn := cmn.GetPgxConn()
		redisClient := cmn.GetRedisConn()

		// 定义检查逻辑为可复用函数
		checkStatus := func() bool {
			// 1. 查询考试状态
			examSessionStatus, err := QueryExamSessionStatus(ctx, nil, examSessionID)
			if err != nil {
				z.Error(fmt.Errorf("查询考试场次 %d 的状态失败: %v", examSessionID, err).Error())
				return false
			}

			switch examSessionStatus {
			// "批改中"
			case "08":
				var examSessionUnmarkedCount int64

				// 2. 查询未批改数
				queryExamSessionUnmarkedCountSql := `SELECT unmarked_student_count FROM v_exam_unmarked_student_count WHERE exam_session_id = $1`
				err = pgxConn.QueryRow(ctx, queryExamSessionUnmarkedCountSql, examSessionID).Scan(&examSessionUnmarkedCount)
				if err != nil {
					if errors.Is(err, pgx.ErrNoRows) {
						z.Error(fmt.Errorf("查询不到考试场次 %d : %v", examSessionID, err).Error())
					} else {
						z.Error(fmt.Errorf("获取考试场次 %d 的 未批改人数失败: %v", examSessionID, err).Error())
					}
					return false
				}

				// 3. 所有人都已批改, 直接更新
				if examSessionUnmarkedCount == 0 {
					_, err = UpdateExamSessionOrPracticeSubmissionStatus(ctx, nil, teacherID, []int64{examSessionID}, nil, ExamSessionStatusSubmitted)
					if err != nil {
						z.Error(fmt.Errorf("ai 批改完成之后，考试状态无法更改: %v", err).Error())
						return false
					}

					z.Info("更新考试状态成功 update state success")

					examSessionMStatusUpdateKey := fmt.Sprintf("exam_session:%dstate_update", examSessionID)

					// 所有人批改完了,可以删除
					if err = redisClient.Del(ctx, examSessionMStatusUpdateKey).Err(); err != nil {
						z.Error(fmt.Errorf("删除 %s key 失败: %v", examSessionMStatusUpdateKey, err).Error())
						return false
					}

					return true // 结束任务
				}

			// “已批改"
			case "10":
				return true

			default:
				z.Info("考试未结束, 无法更新为\"已批改\"")
			}
			return false
		}

		// 在 0s 立即执行一次检查
		if checkStatus() {
			return // 如果检查导致任务结束，直接退出
		}

		// 定时两分钟 （第 0s 不会执行）
		ticker := time.NewTicker(2 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				z.Error("定时任务被取消",
					zap.Int64("exam_session_id", examSessionID),
					zap.Error(ctx.Err()),
				)
				return
			case <-ticker.C:
				if checkStatus() {
					return // 如果检查导致任务结束，直接退出
				}
			}
		}
	}()
}

// --------------------- 抽象方法 -----------------------

// 提交考试（核分员）/练习
func submitResult(ctx context.Context, cond QueryCondition) (err error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string) // 用于强制执行错误处理代码
	if forceErr == "submitResult" {
		return ForceErr
	}

	pgxConn := cmn.GetPgxConn()
	tx, err := pgxConn.Begin(ctx)
	if err != nil || forceErr == "pgxConn.Begin" {
		return fmt.Errorf("事务开启失败: %v", err)
	}

	defer func() {
		if err == nil {
			if cerr := tx.Commit(ctx); cerr != nil {
				err = fmt.Errorf("事务提交失败: %v", cerr)
			}
			return
		}

		if rerr := tx.Rollback(ctx); rerr != nil {
			err = fmt.Errorf("%v; 回滚失败: %w", err, rerr)
		}
	}()

	if cond.ExamSessionID > 0 {
		// TODO 验证人数是否为 0
	}

	// 1. 获取最终批改的结果
	results, err := QuerySubjectiveQuestionsMarkingResults(ctx, cond)
	if err != nil || forceErr == "queryMarkingResults" {
		err = fmt.Errorf("获取最终批改的结果: %v", err)
		return
	}

	// 2. 写入sa表存储学生最后的得分
	_, err = UpdateStudentAnswerScore(ctx, tx, results, cond)
	if err != nil || forceErr == "UpdateStudentAnswerScore" {
		err = fmt.Errorf("保存批改分数失败: %v", err)
		return
	}

	// 3. 更新考试 / 练习的状态
	err = updateToSubmissionStatus(ctx, tx, cond)
	if err != nil {
		return fmt.Errorf("更新 考试/练习 提交状态失败: %v", err)
	}

	return nil
}

// 获取本次考试/练习的需要批改的题目和学生信息
func getQuestionsAndStudentInfos(ctx context.Context, cond QueryCondition) (Detail, error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	if forceErr == "getQuestionsAndStudentInfos" {
		return Detail{}, errors.New("")
	}

	// 1. 获取批改信息，得到mark_mode，据此判断需要获取哪一部分数据
	markerInfo, err := QueryMarkerInfo(ctx, cond) // 根据自己的 teacher_id 查询出来的，所以 len(markerInfo.MarkInfos) == 1
	if err != nil {
		return Detail{}, fmt.Errorf("获取批改信息失败: %v", err)
	}

	// 查不到的情况（正常情况，必定会有批改配置）
	if len(markerInfo.MarkInfos) != 1 {
		return Detail{}, fmt.Errorf("批改配置信息错误: %v", err)
	}

	// 2. 获取该老师需要批改的问题（t_mark_info 一条记录）
	questionSets, err := QuerySubjectiveQuestions(ctx, nil, cond, markerInfo)
	if err != nil {
		return Detail{}, fmt.Errorf("获取该老师需要批改的问题: %v", err)
	}

	// 3. 获取需要批改的学生的信息
	studentInfos, err := QueryStudentInfos(ctx, cond, markerInfo)
	if err != nil {
		return Detail{}, fmt.Errorf("获取需要批改的学生的信息: %v", err)
	}

	// 4. 获取已批改人数，已批改总问题数
	markedPerson, markedQuestions, err := QueryMarkedPersonAndQuestionCount(ctx, nil, cond)
	if err != nil {
		return Detail{}, fmt.Errorf("获取已批改人数: %v", err)
	}

	return Detail{
		TeacherID:       cond.TeacherID,
		QuestionSets:    questionSets,
		StudentInfos:    studentInfos,
		MarkedPerson:    markedPerson,
		MarkedQuestions: markedQuestions,
	}, nil
}

// 保存批改结果，并返回已批改数和总批改题目
func saveMarkingResults(ctx context.Context, teacherID int64, reqBody io.Reader) (markedInfoJson []byte, err error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	if forceErr == "saveMarkingResults" {
		return nil, ForceErr
	}

	// 直接从 Body 流中解码
	var body cmn.ReqProto
	err = json.NewDecoder(reqBody).Decode(&body)
	if err != nil {
		err = fmt.Errorf("解析请求体失败: %v", err)
		return
	}

	var markingResult cmn.TMark
	err = json.Unmarshal(body.Data, &markingResult)
	if err != nil {
		err = fmt.Errorf("解析 body.Data 失败: %v", err)
		return
	}

	QuestionID := markingResult.QuestionID.Int64

	if QuestionID <= 0 {
		err = errors.New("缺少 问题ID")
		return
	}

	examSessionID := markingResult.ExamSessionID.Int64
	practiceID := markingResult.PracticeID.Int64

	if practiceID > 0 && examSessionID > 0 {
		err = errors.New("practiceID 和 examSessionID 不能同时存在")
		return
	}

	if practiceID <= 0 && examSessionID <= 0 {
		err = errors.New("practiceID 和 examSessionID 必须存在一个")
		return
	}

	if markingResult.ExamineeID.Int64 <= 0 {
		err = errors.New("缺少 学生ID")
		return
	}

	pgxConn := cmn.GetPgxConn()
	tx, err := pgxConn.Begin(ctx)
	if err != nil || forceErr == "pgxConn.Begin" {
		err = fmt.Errorf("事务开启错误: %v", err)
		return
	}

	defer func() {
		if err == nil {
			if cerr := tx.Commit(ctx); cerr != nil {
				err = fmt.Errorf("事务提交失败: %v", cerr)
			}
			return
		}

		if rerr := tx.Rollback(ctx); rerr != nil {
			err = fmt.Errorf("%v; 回滚失败: %w", err, rerr)
		}
	}()

	cond := QueryCondition{
		TeacherID:     teacherID,
		PracticeID:    practiceID,
		ExamSessionID: examSessionID,
		QuestionID:    QuestionID,
	}

	// 考试特判
	if cond.ExamSessionID > 0 {
		examSessionStatus, qerr := QueryExamSessionStatus(ctx, nil, cond.ExamSessionID)
		if qerr != nil {
			err = fmt.Errorf("查询考试场次 %d 的状态失败: %v", cond.ExamSessionID, qerr)
			return
		}

		z.Info("examSessionStatus", zap.Any("examSessionStatus", examSessionStatus))
		// 1. 检查是否考试状态是"已结束"
		if examSessionStatus == "06" {
			// 更新为"批改中"
			_, err = UpdateExamSessionOrPracticeSubmissionStatus(ctx, tx, teacherID, []int64{cond.ExamSessionID}, nil, PracticeSubmissionStatusSubmitted)
			if err != nil {
				err = fmt.Errorf("更新考试状态失败: %v", err)
				return
			}
		}
	}

	// 2. 分数校验
	// 查询该问题
	questionSets, err := QuerySubjectiveQuestions(ctx, tx, cond, MarkerInfo{})
	if err != nil {
		err = fmt.Errorf("查询问题失败: %v", err)
		return
	}

	if len(questionSets) != 1 {
		err = fmt.Errorf("查询得到的题组数目应为 1，实际为 %d", len(questionSets))
		return
	}

	if len(questionSets[0].Questions) != 1 {
		err = fmt.Errorf("查询得到的题目数目应为 1，实际为 %d", len(questionSets[0].Questions))
		return
	}

	// 得到该小题的分数
	var givenScores []struct {
		Index int64
		Score float64
	}
	err = json.Unmarshal(questionSets[0].Questions[0].Answers, &givenScores)
	if err != nil {
		err = fmt.Errorf("解析 questionSets[0].Questions[0].Answers 失败：%v", err)
		return
	}

	// 获取前端返回的批改结果
	var markDetails []MarkDetail
	err = json.Unmarshal(markingResult.MarkDetails, &markDetails)
	if err != nil {
		err = fmt.Errorf("解析 markingResult.MarkDetails 失败: %v", err)
		return
	}

	questionLen := int64(len(givenScores))

	// index -> score
	givenScoresMap := make(map[int64]float64, questionLen)
	for _, givenScore := range givenScores {
		givenScoresMap[givenScore.Index] = givenScore.Score
	}

	// 批改时计算的小题批改后的总分
	var markingTotalScore float64

	for _, markDetail := range markDetails {
		// 1. 校验题目序号
		index := int64(markDetail.Index)
		if index > questionLen {
			err = fmt.Errorf("所批分数的题目小题的序号超出题目范围，超出的序号是 %d，实际最多为 %d", index, questionLen)
			return
		}

		// 2. 检验小题分数大小
		givenScore := givenScoresMap[index]
		if markDetail.Score > givenScore {
			err = fmt.Errorf("所批分数超出题目小题的分数，小题为 %d，超出的分数是 %f，实际最多为 %f", markDetail.Index, markDetail.Score, givenScore)
			return
		} else {
			markingTotalScore += markDetail.Score
		}
	}

	// 3. 验证所给总分大小（因为总分是单独一个字段）
	givenTotalScore := questionSets[0].Score
	if markingTotalScore > givenTotalScore {
		err = fmt.Errorf("所批总分数超出题目总分数，超出的分数是 %f，实际最多为 %f", markingTotalScore, givenTotalScore)
		return
	}

	// 4. 保存该批改记录
	_, err = UpsertMarkingResults(ctx, tx, []cmn.TMark{markingResult})
	if err != nil || forceErr == "insertOrUpdateMarkingResults" {
		err = fmt.Errorf("保存批改记录失败: %v", err)
		return
	}

	// 5. 查询已批改学生人数和问题数
	markedPerson, markedQuestions, err := QueryMarkedPersonAndQuestionCount(ctx, tx, cond)
	if err != nil || forceErr == "queryMarkedPersonAndQuestionCount" {
		err = fmt.Errorf("查询已批改学生人数和问题数失败: %v", err)
		return
	}

	markedInfoJson, err = json.Marshal(map[string]interface{}{
		"marked_person":    markedPerson,
		"marked_questions": markedQuestions,
	})
	if err != nil || forceErr == "failed to marshal data" {
		err = fmt.Errorf("构造 jsonData 失败: %v", err)
		return
	}

	return markedInfoJson, nil
}

// 用于保存批改配置信息
func HandleMarkerInfo(ctx context.Context, tx *pgx.Tx, teacherID int64, req HandleMarkerInfoReq) error {
	forceErr, _ := ctx.Value(ForceErrKey).(string)

	if teacherID <= 0 {
		err := errors.New("无效的 teacherID")
		z.Error(err.Error())
		return err
	}

	switch req.Status {
	// 保存批改配置
	case "00":
		if req.ExamSessionID <= 0 && req.PracticeID <= 0 {
			err := fmt.Errorf("无效的 exam_session_id：%d 或 practice_id：%d", req.ExamSessionID, req.PracticeID)
			z.Error(err.Error())
			return err
		}

		var markInfos []cmn.TMarkInfo

		switch req.MarkMode {
		// 自动批改 // 单人批改
		case "00", "10": // 正好的情况：练习仅有这两种批改模式
			if req.MarkMode == "10" && (req.Markers == nil || len(req.Markers) == 0 || len(req.Markers) > 1) {
				err := errors.New("无效的 markers")
				z.Error(err.Error())
				return err
			}

			markInfo := cmn.TMarkInfo{
				MarkTeacherID:      null.IntFrom(teacherID),
				MarkCount:          null.IntFrom(0),
				MarkExamineeIds:    nil,
				MarkQuestionGroups: nil,
				Creator:            null.IntFrom(teacherID),
				UpdatedBy:          null.IntFrom(teacherID),
				CreateTime:         null.IntFrom(time.Now().UnixMilli()),
				Status:             null.StringFrom("00"),
			}

			if req.PracticeID > 0 {
				markInfo.PracticeID = null.IntFrom(req.PracticeID)
			} else {
				markInfo.ExamSessionID = null.IntFrom(req.ExamSessionID)
			}
			markInfos = append(markInfos, markInfo)

		// 全卷多评，多位批阅员同时所有的卷子
		case "02":
			if req.Markers == nil || len(req.Markers) == 0 {
				err := errors.New("markers 为空，无法配置批改信息")
				z.Error(err.Error())
				return err
			}

			for _, marker := range req.Markers {
				markInfos = append(markInfos, cmn.TMarkInfo{
					ExamSessionID:      null.IntFrom(req.ExamSessionID),
					MarkTeacherID:      null.IntFrom(marker),
					MarkCount:          null.IntFrom(0),
					MarkExamineeIds:    nil,
					MarkQuestionGroups: nil,
					Creator:            null.IntFrom(teacherID),
					UpdatedBy:          null.IntFrom(teacherID),
					CreateTime:         null.IntFrom(time.Now().UnixMilli()),
					Status:             null.StringFrom("00"),
				})
			}

		// 试卷分配，每位老师批改不同的考生
		case "04":
			if req.ExamineeIDs == nil || len(req.ExamineeIDs) == 0 {
				err := errors.New("examinee_ids 为空，无法分配考生")
				z.Error(err.Error())
				return err
			}

			// 将考生数组打乱平均分成n组
			// 忽略错误是因为上一步已经判断非0
			splitIDs, _ := randomSplit(req.ExamineeIDs, len(req.Markers))
			for i, ids := range splitIDs {
				markExamineeIdsBytes, err := json.Marshal(ids)
				if err != nil || forceErr == "json.Marshal-1" {
					err = fmt.Errorf("构造 markExamineeIds Json 失败: %v", err)
					z.Error(err.Error())
					return err
				}

				markInfos = append(markInfos, cmn.TMarkInfo{
					ExamSessionID:      null.IntFrom(req.ExamSessionID),
					MarkTeacherID:      null.IntFrom(req.Markers[i]),
					MarkCount:          null.IntFrom(0),
					MarkExamineeIds:    markExamineeIdsBytes,
					MarkQuestionGroups: nil,
					Creator:            null.IntFrom(teacherID),
					UpdatedBy:          null.IntFrom(teacherID),
					CreateTime:         null.IntFrom(time.Now().UnixMilli()),
					Status:             null.StringFrom("00"),
				})
			}

		// 分题组
		case "06":
			// 查询本次exam_session所使用的试卷的所有主观题组
			cond := QueryCondition{
				ExamSessionID: req.ExamSessionID,
				TeacherID:     teacherID,
			}

			markerInfo := MarkerInfo{}

			// 查询这张试卷的所有主观题题组
			questionSets, err := QuerySubjectiveQuestions(ctx, nil, cond, markerInfo)
			if err != nil {
				err = fmt.Errorf("查询试卷的主观题目失败: %v", err)
				z.Error(err.Error())
				return err
			}

			// 将题组平均打乱成n份 [[3, 4], [1， 2], [5]]
			splitGroups, err := randomSplit(questionSets, len(req.Markers))
			if err != nil {
				err = fmt.Errorf("将题组平均打乱成n份失败: %v", err)
				z.Error(err.Error())
				return err
			}
			for i, groups := range splitGroups {
				questionIDs := make([]int64, 0, 10) // 假设 10 小题

				for _, questionGroup := range groups {
					for _, question := range questionGroup.Questions {
						questionIDs = append(questionIDs, question.ID.Int64)
					}
				}

				markQuestionIDsBytes, err := json.Marshal(questionIDs)
				if err != nil || forceErr == "json.Marshal-2" {
					err = fmt.Errorf("构造 markQuestionIDsBytes Json 失败: %v", err)
					z.Error(err.Error())
					return err
				}

				markInfos = append(markInfos, cmn.TMarkInfo{
					ExamSessionID:   null.IntFrom(req.ExamSessionID),
					MarkTeacherID:   null.IntFrom(req.Markers[i]),
					MarkCount:       null.IntFrom(0),
					MarkExamineeIds: nil,
					QuestionIds:     markQuestionIDsBytes, // 存放题组的所有的问题的id
					Creator:         null.IntFrom(teacherID),
					UpdatedBy:       null.IntFrom(teacherID),
					CreateTime:      null.IntFrom(time.Now().UnixMilli()),
					Status:          null.StringFrom("00"),
				})
			}

		// 分题目
		case "08":

		default:
			err := fmt.Errorf("无效的批改模式: %s", req.MarkMode)
			z.Error(err.Error())
			return err
		}

		// 信息判断，其实这里已经做到了上面的TODO
		if len(markInfos) <= 0 {
			err := errors.New("没有批改信息可以用于保存")
			z.Error(err.Error())
			return err
		}

		insertQuery := `INSERT INTO t_mark_info 
						(exam_session_id, practice_id, mark_teacher_id, mark_count, question_ids, mark_examinee_ids, creator, create_time, updated_by, update_time, addi, status) 
					VALUES 
						($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) 
					RETURNING id`

		var targetIDs []int64
		for _, info := range markInfos {
			var id null.Int
			err := (*tx).QueryRow(ctx, insertQuery, info.ExamSessionID, info.PracticeID, info.MarkTeacherID, info.MarkCount, info.QuestionIds, info.MarkExamineeIds, info.Creator, info.CreateTime, info.UpdatedBy, info.UpdateTime, info.Addi, info.Status).Scan(&id)
			if err != nil || forceErr == "HandleMarkerInfo-tx.QueryRow" {
				err = fmt.Errorf("执行 QueryRow 失败: %v", err)
				z.Error(err.Error())
				return err
			}
			targetIDs = append(targetIDs, id.Int64)
		}

	// 删除批改配置
	case "02":
		var mode string
		var ids []int64

		if len(req.PracticeIDs) > 0 {
			// 删除练习批改配置
			ids = req.PracticeIDs
			mode = "02" // 练习
		} else {
			if req.ExamSessionIDs == nil || len(req.ExamSessionIDs) == 0 {
				// 删除考试批改配置
				err := errors.New("缺少 examSessionIDs，无法修改考试批改配置")
				z.Error(err.Error())
				return err
			}

			ids = req.ExamSessionIDs
			mode = "00" // 考试
		}

		_, err := UpdateMarkerInfoStatus(ctx, *tx, teacherID, ids, mode, "04")
		if err != nil {
			err = fmt.Errorf("更新批改配置的状态失败: %v", err)
			z.Error(err.Error())
			return err
		}

	default:
		err := fmt.Errorf("无效的 status: %s", req.Status)
		z.Error(err.Error())
		return err
	}

	return nil
}

// 根据 cond 判断更新考试 / 练习的状态
func updateToSubmissionStatus(ctx context.Context, tx pgx.Tx, cond QueryCondition) error {
	var status string
	var examSessionIDs []int64
	var practiceSubmissionIDs []int64

	if cond.ExamSessionID > 0 {
		status = ExamSessionStatusSubmitted
		examSessionIDs = []int64{cond.ExamSessionID}
	} else if cond.PracticeSubmissionID > 0 {
		status = PracticeSubmissionStatusSubmitted
		practiceSubmissionIDs = []int64{cond.PracticeSubmissionID}
	}

	_, err := UpdateExamSessionOrPracticeSubmissionStatus(ctx, tx, cond.TeacherID, examSessionIDs, practiceSubmissionIDs, status)
	if err != nil {
		return err
	}

	return nil
}
