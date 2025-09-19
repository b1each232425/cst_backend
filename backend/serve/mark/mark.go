package mark

//annotation:mark-service
//author:{"name":"Lin ZiXin","tel":"13430362233","email":"linzixinzz1@gmail.com"}

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"w2w.io/serve/auth_mgt"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"github.com/sony/gobreaker/v2"
	"github.com/spf13/viper"
	"w2w.io/null"
	"w2w.io/serve/ai_mark"
	"w2w.io/serve/examPaper"

	"go.uber.org/zap"
	"w2w.io/cmn"
)

var z *zap.Logger
var aiMarkTaskLimiter *cmn.RateLimiterTaskRunner
var aiMarkTaskCB *gobreaker.CircuitBreaker[any]

type forceErrKey string

const ForceErrKey = forceErrKey("force-err")

const (
	TaskTypeAIMarkRequest = "ai_mark:grade"
)

func init() {
	//Setup package scope variables, just like logger, db connector, configure parameters, etc.
	cmn.PackageStarters = append(cmn.PackageStarters, func() {
		z = cmn.GetLogger()
		z.Info("user zLogger settled")
	})
}

type User struct {
	ID      int64  `json:"id" validate:"required,gt=0"`
	Level   string `json:"level"`
	IsAdmin bool   `json:"is_admin"`
}

type QueryMarkingListReq struct {
	User         *User     `json:"user" validate:"required"`
	ExamName     string    `json:"exam_name" validate:"max=999"`
	PracticeName string    `json:"practice_name" validate:"max=999"`
	Limit        int       `json:"limit" validate:"required,gt=0,lte=1000"`
	Offset       int       `json:"offset" validate:"gte=0"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	Status       string    `json:"status"`
}

type MarkDetails struct {
	Analyze string  `json:"analyze"`
	Index   int     `json:"index"`
	Score   float64 `json:"score"`
}

type HandleMarkerInfoReq struct {
	Markers        []int64                             `json:"markers"`          // *批改员id数组
	QuestionGroups []examPaper.SubjectiveQuestionGroup `json:"question_groups"`  // TODO 题组（已废弃，不再外部传入，而是内部自己查）
	QuestionIDs    []int64                             `json:"question_ids"`     // 题目id数组
	ExamineeIDs    []int64                             `json:"examinee_ids"`     // 考生id数组
	MarkMode       string                              `json:"mark_mode"`        // *批卷模式 00：不需要手动批改  02：全卷多评 04：试卷分配 06：题组专评 08：题目分配 10：单人（人工）批改
	ExamSessionID  int64                               `json:"exam_session_id"`  // 考试场次id
	PracticeID     int64                               `json:"practice_id"`      // 练习id
	Status         string                              `json:"status"`           // *00 插入批改配置 02 删除批改配置
	ExamSessionIDs []int64                             `json:"exam_session_ids"` // 要删除的考试场次id数组
	PracticeIDs    []int64                             `json:"practice_ids"`     // 要删除的练习id数组
}

type AIMarkRequest struct {
	Question       *ai_mark.QuestionDetails
	StudentAnswers []*ai_mark.StudentAnswer
}

type AIMarkTaskPayLoad struct {
	AIMarkRequest      AIMarkRequest  `json:"ai_mark_request"`
	QueryCondition     QueryCondition `json:"query_condition"`
	UniqueTaskCountKey string         `json:"unique_task_count_key"`
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

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: GetExamList,

		Path: "/mark/exam",
		Name: "获取考试批改列表",

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "试卷批改.获取考试列表",
				AccessAction: auth_mgt.CAPIAccessActionRead,
				Configurable: false,
			},
		},

		Developer: developer,
		WhiteList: true,

		DomainID: int64(cmn.CDomainSys),

		DefaultDomain: int64(cmn.CDomainSys),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: GetPracticeList,

		Path: "/mark/practice",
		Name: "获取练习批改列表",

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "试卷批改.获取练习列表",
				AccessAction: auth_mgt.CAPIAccessActionRead,
				Configurable: false,
			},
		},

		Developer: developer,
		WhiteList: true,

		DomainID: int64(cmn.CDomainSys),

		DefaultDomain: int64(cmn.CDomainSys),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: GetQuestionsAndStudentInfos,

		Path: "/mark/questions-and-student-infos",
		Name: "获取批改所需问题和学生信息",

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "试卷批改.获取批改所需问题和学生信息",
				AccessAction: auth_mgt.CAPIAccessActionRead,
				Configurable: false,
			},
		},

		Developer: developer,
		WhiteList: true,

		DomainID: int64(cmn.CDomainSys),

		DefaultDomain: int64(cmn.CDomainSys),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: GetStudentAnswersAndMarkResults,

		Path: "/mark/student-answers-and-mark-results",
		Name: "获取批改所需学生答案和批改记录",

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "试卷批改.获取批改所需学生答案和批改记录",
				AccessAction: auth_mgt.CAPIAccessActionRead,
				Configurable: false,
			},
		},

		Developer: developer,
		WhiteList: true,

		DomainID: int64(cmn.CDomainSys),

		DefaultDomain: int64(cmn.CDomainSys),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: SaveMarkingResults,

		Path: "/mark/marking-results",
		Name: "保存批改结果",

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "试卷批改.保存批改结果",
				AccessAction: auth_mgt.CAPIAccessActionRead,
				Configurable: false,
			},
		},

		Developer: developer,
		WhiteList: true,

		DomainID: int64(cmn.CDomainSys),

		DefaultDomain: int64(cmn.CDomainSys),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: SubmitResult,

		Path: "/mark/results-submission",
		Name: "提交批改",

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "试卷批改.提交批改",
				AccessAction: auth_mgt.CAPIAccessActionRead,
				Configurable: false,
			},
		},

		Developer: developer,
		WhiteList: true,

		DomainID: int64(cmn.CDomainSys),

		DefaultDomain: int64(cmn.CDomainSys),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: UpdateMarkingState,

		Path: "/mark/state",
		Name: "更新批改状态",

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "试卷批改.更新批改状态",
				AccessAction: auth_mgt.CAPIAccessActionRead,
				Configurable: false,
			},
		},

		Developer: developer,
		WhiteList: true,

		DomainID: int64(cmn.CDomainSys),

		DefaultDomain: int64(cmn.CDomainSys),
	})

	InitAIMarkTaskLimiterAndBreaker()

	cmn.RegisterTaskHandler(TaskTypeAIMarkRequest, TaskMiddleware(HandleAIMarkTask))
}

// 获取考试批改列表
func GetExamList(ctx context.Context) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)

	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	method := strings.ToLower(q.R.Method)
	if method != "get" {
		q.Err = errors.New("please call /api/mark/exam with http GET method")
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
		q.Err = fmt.Errorf("error parsing page index: %v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	if pageIndex < 1 {
		q.Err = errors.New("page index must be greater than 0")
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
		q.Err = fmt.Errorf("error parsing page size: %v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	if pageSize < 1 || pageSize > 1000 {
		q.Err = errors.New("page size must be between 1 and 1000")
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
			q.Err = fmt.Errorf("error parsing start time: %v", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		endTimeInt64, err := strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			q.Err = fmt.Errorf("error parsing end time: %v", err)
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
		q.Err = err
		q.RespErr()
		return
	}

	jsonData, err := json.Marshal(map[string]interface{}{
		"exam_list": exams,
	})

	if err != nil || forceErr == "GetExamList-json.Marshal" {
		q.Err = fmt.Errorf("unable to marshal response data: %v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	q.Msg.Data = jsonData
	q.Msg.RowCount = int64(rowCount)
	q.Resp()
}

// 获取练习批改列表
func GetPracticeList(ctx context.Context) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)

	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	method := strings.ToLower(q.R.Method)
	if method != "get" {
		q.Err = errors.New("please call /api/mark/practice with http GET method")
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
		q.Err = fmt.Errorf("error parsing page index: %v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	if pageIndex < 1 {
		q.Err = errors.New("page index must be greater than 0")
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
		q.Err = fmt.Errorf("error parsing page size: %v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	if pageSize < 1 || pageSize > 1000 {
		q.Err = errors.New("page size must be between 1 and 1000")
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
		q.Err = err
		q.RespErr()
		return
	}

	jsonData, err := json.Marshal(map[string]interface{}{
		"practice_list": practices,
	})
	if err != nil || forceErr == "GetPracticeList-json.Marshal" {
		q.Err = fmt.Errorf("unable to marshal response data: %v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	q.Msg.Data = jsonData
	q.Msg.RowCount = int64(rowCount)
	q.Resp()
}

// 获取本次考试/练习的需要批改的题目和学生信息
func GetQuestionsAndStudentInfos(ctx context.Context) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)

	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	method := strings.ToLower(q.R.Method)
	if method != "get" {
		q.Err = errors.New("please call /api/mark/questions-and-student-infos with http GET method")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	queryParams := q.R.URL.Query()

	examSessionIDStr := queryParams.Get("exam_session_id")
	practiceIDStr := queryParams.Get("practice_id")
	if examSessionIDStr == "" && practiceIDStr == "" {
		q.Err = errors.New("请求参数必须包含练习ID或者考试场次ID中的一个")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	if practiceIDStr != "" && examSessionIDStr != "" {
		q.Err = errors.New("请求参数不能同时包含练习ID和考试场次ID")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	var examSessionID int64
	var practiceID int64
	var err error
	if examSessionIDStr != "" {
		examSessionID, err = strconv.ParseInt(examSessionIDStr, 10, 64)
		if err != nil {
			q.Err = fmt.Errorf("error parsing exam_session_id: %v", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
	}

	if practiceIDStr != "" {
		practiceID, err = strconv.ParseInt(practiceIDStr, 10, 64)
		if err != nil {
			q.Err = fmt.Errorf("error parsing practice_id: %v", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
	}

	cond := QueryCondition{
		TeacherID:     q.SysUser.ID.Int64,
		ExamSessionID: examSessionID,
		PracticeID:    practiceID,
	}

	// 1. 获取批改信息，得到mark_mode，据此判断需要获取哪一部分数据
	markerInfo, err := QueryMarkerInfo(ctx, cond)
	if err != nil {
		q.Err = err
		q.RespErr()
		return
	}

	// 批改模式必须正确
	switch markerInfo.MarkMode {
	case "00", "02", "04", "06", "08", "10", "12", "14": // TODO
	default:
		q.Err = errors.New("mark_mode error")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	// 必定会有批改配置
	if markerInfo.MarkInfos == nil || len(markerInfo.MarkInfos) == 0 {
		q.Err = errors.New("no mark_info")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	z.Info("", zap.Any("----------------->", markerInfo))

	// 2. 获取该老师需要批改的问题（t_mark_info 一条记录）
	questionSets, err := QuerySubjectiveQuestions(ctx, cond, markerInfo)
	if err != nil {
		q.Err = err
		q.RespErr()
		return
	}

	// 3. 获取需要批改的学生的信息
	studentInfos, err := QueryStudentInfos(ctx, cond, markerInfo)
	if err != nil {
		q.Err = err
		q.RespErr()
		return
	}

	// 4. 获取已批改人数，已批改总问题数
	markedPerson, markedQuestions, err := QueryMarkedPersonAndQuestionCount(ctx, cond)
	if err != nil {
		q.Err = err
		q.RespErr()
		return
	}

	jsonData, err := json.Marshal(map[string]interface{}{
		"teacher_id":       q.SysUser.ID.Int64, // 用于区别其他批改员
		"question_sets":    questionSets,
		"student_infos":    studentInfos,
		"marked_person":    markedPerson,
		"marked_questions": markedQuestions,
	})
	if err != nil || forceErr == "HandleMarkingDetails-json.Marshal" {
		q.Err = fmt.Errorf("failed to marshal response data: %v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	q.Msg.Data = jsonData
	q.Resp()
}

// 根据场次/练习+考生id获取学生的答案和批改结果
func GetStudentAnswersAndMarkResults(ctx context.Context) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)

	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	method := strings.ToLower(q.R.Method)
	if method != "get" {
		q.Err = errors.New("please call /api/mark/student-answers-and-mark-results with http GET method")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	queryParams := q.R.URL.Query()

	examSessionIDStr := queryParams.Get("exam_session_id")
	practiceIDStr := queryParams.Get("practice_id")
	if examSessionIDStr == "" && practiceIDStr == "" {
		q.Err = errors.New("请求参数必须包含练习ID或者考试场次ID中的一个")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	if practiceIDStr != "" && examSessionIDStr != "" {
		q.Err = errors.New("请求参数不能同时包含练习ID和考试场次ID")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	var examSessionID int64
	var examineeID int64
	var practiceID int64
	var practiceSubmissionID int64
	var err error

	if examSessionIDStr != "" {
		examSessionID, err = strconv.ParseInt(examSessionIDStr, 10, 64)
		if err != nil {
			q.Err = fmt.Errorf("error parsing exam_session_id: %v", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
	}

	if practiceIDStr != "" {
		practiceID, err = strconv.ParseInt(practiceIDStr, 10, 64)
		if err != nil {
			q.Err = fmt.Errorf("error parsing practice_id: %v", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
	}

	examineeIDStr := queryParams.Get("examinee_id")
	if examineeIDStr != "" {
		examineeID, err = strconv.ParseInt(examineeIDStr, 10, 64)
		if err != nil {
			q.Err = fmt.Errorf("error parsing examinee_id: %v", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
	}

	practiceSubmissionIDStr := queryParams.Get("practice_submission_id")
	if practiceSubmissionIDStr != "" {
		practiceSubmissionID, err = strconv.ParseInt(practiceSubmissionIDStr, 10, 64)
		if err != nil {
			q.Err = fmt.Errorf("error parsing practice_submission_id: %v", err)
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
		q.Err = err
		q.RespErr()
		return
	}

	// 自定义，用于区别考试（数据库表对应字段没有这个）
	switch {
	case cond.PracticeSubmissionID > 0:
		// 查询单个练习学生
		markerInfo.MarkMode = "12" // 批改单个练习学生
	case cond.PracticeWrongSubmissionID > 0:
		// 批改错题集练习的提交
		markerInfo.MarkMode = "14" // 批改练习错题
	}

	// TODO 检查

	// 2. 获取学生主观题答案
	studentAnswers, err := QueryStudentAnswers(ctx, "02", cond, markerInfo)
	if err != nil {
		q.Err = err
		q.RespErr()
		return
	}

	// 3. 获取老师对该学生的批改记录
	markingResults, err := QueryMarkingResults(ctx, cond)
	if err != nil {
		q.Err = err
		q.RespErr()
		return
	}

	jsonData, err := json.Marshal(map[string]interface{}{
		"student_answers": studentAnswers,
		"marking_results": markingResults,
	})
	if err != nil || forceErr == "HandleMarkingDetails-json.Marshal" {
		q.Err = fmt.Errorf("failed to marshal response data: %v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	q.Msg.Data = jsonData
	q.Resp()
}

// 保存批改结果，并返回已批改数和总批改题目
func SaveMarkingResults(ctx context.Context) {
	forceErr, _ := ctx.Value(ForceErrKey).(string) // 用于强制执行错误处理代码

	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	method := strings.ToLower(q.R.Method)
	if method != "post" {
		q.Err = errors.New("please call /api/mark/marking-results with http post method")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	var body cmn.ReqProto
	// 直接从 Body 流中解码
	if err := json.NewDecoder(q.R.Body).Decode(&body); err != nil {
		q.Err = fmt.Errorf("failed to decode request body: %v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	var markingResult cmn.TMark
	q.Err = json.Unmarshal(body.Data, &markingResult)
	if q.Err != nil {
		q.Err = fmt.Errorf("failed to unmarshal marking results: %v", q.Err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	// 1. 保存批改记录
	_, err := InsertOrUpdateMarkingResults(ctx, nil, []cmn.TMark{markingResult})
	if err != nil || forceErr == "insertOrUpdateMarkingResults" {
		q.Err = fmt.Errorf("failed to insert or update marking results: %v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	cond := QueryCondition{
		TeacherID:     q.SysUser.ID.Int64,
		PracticeID:    markingResult.PracticeID.Int64,
		ExamSessionID: markingResult.ExamSessionID.Int64,
	}

	if cond.PracticeID > 0 && cond.ExamSessionID > 0 {
		//TODO
	}

	// 2. 查询已批改学生人数和问题数
	markedPerson, markedQuestions, err := QueryMarkedPersonAndQuestionCount(ctx, cond)
	if err != nil || forceErr == "queryMarkedPersonAndQuestionCount" {
		q.Err = err
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	jsonData, err := json.Marshal(map[string]interface{}{
		"marked_person":    markedPerson,
		"marked_questions": markedQuestions,
	})
	if err != nil || forceErr == "failed to marshal data" {
		q.Err = err
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	q.Msg.Data = jsonData
	q.Resp()
}

// 更新考试的状态为“批改中”
func UpdateMarkingState(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	method := strings.ToLower(q.R.Method)
	if method != "patch" {
		q.Err = errors.New("please call /api/mark/state with http patch method")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	queryParams := q.R.URL.Query()

	examSessionIDStr := queryParams.Get("exam_session_id")
	if examSessionIDStr == "" {
		q.Err = errors.New("exam_session_id is required")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	examSessionID, err := strconv.ParseInt(examSessionIDStr, 10, 64)
	if err != nil {
		q.Err = fmt.Errorf("error parsing exam_session_id: %v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	_, err = updateExamSessionOrPracticeSubmissionState(ctx, nil, q.SysUser.ID.Int64, []int64{examSessionID}, nil, "08")
	if err != nil {
		q.Err = err
		z.Error(err.Error())
		q.RespErr()
		return
	}

	q.Resp()
}

// 提交考试/练习
func SubmitResult(ctx context.Context) {
	forceErr, _ := ctx.Value(ForceErrKey).(string) // 用于强制执行错误处理代码

	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	method := strings.ToLower(q.R.Method)
	if method != "patch" {
		q.Err = errors.New("please call /api/mark/results-submission with http patch method")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	queryParams := q.R.URL.Query()

	examSessionIDStr := queryParams.Get("exam_session_id")
	practiceIDStr := queryParams.Get("practice_id")
	practiceSubmissionIDStr := queryParams.Get("practice_submission_id")

	if examSessionIDStr == "" && practiceIDStr == "" {
		q.Err = fmt.Errorf("请求参数必须包含练习ID或者考试场次ID中的一个")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	if examSessionIDStr != "" && practiceIDStr != "" {
		q.Err = fmt.Errorf("请求参数不能同时包含练习ID和考试场次ID")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	var examSessionID int64
	var practiceID int64
	var practiceSubmissionID int64
	var err error

	if examSessionIDStr != "" {
		examSessionID, err = strconv.ParseInt(examSessionIDStr, 10, 64)
		if err != nil {
			q.Err = fmt.Errorf("error parsing exam_session_id: %v", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
	}

	if practiceIDStr != "" {
		practiceID, err = strconv.ParseInt(practiceIDStr, 10, 64)
		if err != nil {
			q.Err = fmt.Errorf("error parsing practice_id: %v", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if practiceSubmissionIDStr != "" {
			practiceSubmissionID, err = strconv.ParseInt(practiceSubmissionIDStr, 10, 64)
			if err != nil {
				q.Err = fmt.Errorf("error parsing practice_submission_id: %v", err)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}
	}

	cond := QueryCondition{
		ExamSessionID:        examSessionID,
		PracticeID:           practiceID,
		PracticeSubmissionID: practiceSubmissionID,
		TeacherID:            q.SysUser.ID.Int64,
	}

	pgxConn := cmn.GetPgxConn()
	tx, err := pgxConn.Begin(ctx)
	if err != nil || forceErr == "pgxConn.Begin" {
		q.Err = fmt.Errorf("begin transaction error: %v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	defer func() {
		if err != nil {
			if err := tx.Rollback(ctx); err != nil || forceErr == "tx.Rollback" {
				z.Error(err.Error())
			}
		}
	}()

	// 1. 获取最终批改的结果
	results, err := QueryMarkingResults(ctx, cond)
	if err != nil || forceErr == "queryMarkingResults" {
		q.Err = err
		q.RespErr()
		return
	}

	// 2. 写入sa表存储学生最后的得分
	_, err = updateStudentAnswerScore(ctx, tx, results, cond)
	if err != nil || forceErr == "updateStudentAnswerScore" {
		q.Err = err
		q.RespErr()
		return
	}

	var status string
	var examSessionIDs []int64
	var practiceSubmissionIDs []int64

	// 考试：10， 练习提交：08
	if cond.ExamSessionID > 0 {
		status = "10"
		examSessionIDs = []int64{cond.ExamSessionID}
	} else if cond.PracticeSubmissionID > 0 {
		status = "08"
		practiceSubmissionIDs = []int64{cond.PracticeSubmissionID}
	}

	// 3. 更改考试/练习的状态为“已批改”
	_, err = updateExamSessionOrPracticeSubmissionState(ctx, tx, cond.TeacherID, examSessionIDs, practiceSubmissionIDs, status)
	if err != nil || forceErr == "updateExamSessionOrPracticeSubmissionState" {
		q.Err = err
		q.RespErr()
		return
	}

	err = tx.Commit(ctx)
	if err != nil || forceErr == "tx.Commit" {
		q.Err = fmt.Errorf("tx commit error: %v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	q.Resp()
}

// 用于保存批改配置信息
func HandleMarkerInfo(ctx context.Context, tx *pgx.Tx, teacherID int64, req HandleMarkerInfoReq) error {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	if teacherID <= 0 {
		err := errors.New("invalid teacherID")
		z.Error(err.Error())
		return err
	}

	// 00 插入批改配置 02 删除批改配置
	if req.Status != "00" && req.Status != "02" {
		err := errors.New("invalid status")
		z.Error(err.Error())
		return err
	}

	// 删除批改配置
	if req.Status == "02" {
		var mode string
		var ids []int64

		if len(req.PracticeIDs) > 0 {
			// 删除练习批改配置
			ids = req.PracticeIDs
			mode = "02" // 练习
		} else {
			if req.ExamSessionIDs == nil || len(req.ExamSessionIDs) == 0 {
				// 删除考试批改配置
				err := errors.New("no examSessionIDs to update mark info state")
				z.Error(err.Error())
				return err
			}

			ids = req.ExamSessionIDs
			mode = "00" // 考试
		}

		_, err := UpdateMarkerInfoState(ctx, tx, teacherID, ids, mode)
		if err != nil {
			return err
		}
		return nil
	}

	// req.Status == "00" 插入批改配置
	if req.ExamSessionID <= 0 && req.PracticeID <= 0 {
		err := errors.New("invalid exam_session_id && practice_id")
		z.Error(err.Error())
		return err
	}

	var markInfos []cmn.TMarkInfo

	switch req.MarkMode {
	// 自动批改 // 单人批改
	case "00", "10":
		if req.MarkMode == "10" && (req.Markers == nil || len(req.Markers) == 0 || len(req.Markers) > 1) {
			err := errors.New("invalid markers")
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
			err := errors.New("markers is empty")
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
			err := errors.New("examinee_ids is empty")
			z.Error(err.Error())
			return err
		}

		// 将考生数组打乱平均分成n组
		splitIDs := randomSplit(req.ExamineeIDs, len(req.Markers))
		for i, ids := range splitIDs {
			markExamineeIdsBytes, err := json.Marshal(ids)
			if err != nil || forceErr == "json.Marshal-1" {
				err = fmt.Errorf("failed to marshal MarkExamineeIds: %v", err)
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
		//if req.QuestionGroups == nil || len(req.QuestionGroups) == 0 {
		//	err := errors.New("invalid question groups")
		//	z.Error(err.Error())
		//	return err
		//}

		// 查询本次exam_session所使用的试卷的所有主观题组
		cond := QueryCondition{
			ExamSessionID: req.ExamSessionID,
			TeacherID:     teacherID,
		}

		markerInfo := MarkerInfo{}

		// 查询这张试卷的所有主观题题组
		questionSets, err := QuerySubjectiveQuestions(ctx, cond, markerInfo)
		if err != nil {
			return err
		}

		z.Debug("questionSets的值", zap.Any("", questionSets))

		// 将题组平均打乱成n份 [[3, 4], [1， 2], [5]]
		splitGroups := randomSplit(questionSets, len(req.Markers))
		for i, groups := range splitGroups {
			questionIDs := make([]int64, 0, 10) // 假设 10 小题

			for _, questionGroup := range groups {
				for _, question := range questionGroup.Questions {
					questionIDs = append(questionIDs, question.ID.Int64)
				}
			}

			z.Debug("questionIDs的值", zap.Any("", questionIDs))

			var markQuestionIDsBytes []byte
			markQuestionIDsBytes, err := json.Marshal(questionIDs)
			if err != nil || forceErr == "json.Marshal-2" {
				err = fmt.Errorf("failed to marshal markQuestionIDs: %v", err)
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

	// TODO 其他情况的判断

	// 分题目
	case "08":
	}

	// 信息判断，其实这里已经做到了上面的TODO
	if len(markInfos) <= 0 {
		err := errors.New("no markInfos to insert")
		z.Error(err.Error())
		return err
	}

	// TODO 可以先预编译，加快效率
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
			err = fmt.Errorf("exec insert query error: %v", err)
			z.Error(err.Error())
			return err
		}
		targetIDs = append(targetIDs, id.Int64)
	}

	return nil
}

// 自动批改客观题
func markObjectiveQuestionAnswers(ctx context.Context, tx pgx.Tx, cond QueryCondition) error {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	if err := cmn.Validate(cond); err != nil {
		err = fmt.Errorf("invalid query condition: %v", err)
		z.Error(err.Error())
		return err
	}

	if cond.ExamSessionID <= 0 && cond.PracticeID <= 0 {
		err := errors.New("invalid params: exam session id or practice id must be greater than zero")
		z.Error(err.Error())
		return err
	}

	if cond.ExamSessionID > 0 && cond.PracticeID > 0 {
		err := errors.New("invalid params: exam session id && practice id cannot be both greater than zero")
		z.Error(err.Error())
		return err
	}

	markerInfo := MarkerInfo{
		MarkerID: cond.TeacherID,
	}

	if cond.ExamineeID > 0 {
		// 查询单个考试考生
		markerInfo.MarkMode = "04"
	}

	if cond.PracticeSubmissionID > 0 && cond.PracticeWrongSubmissionID <= 0 {
		// 查询单个练习学生
		markerInfo.MarkMode = "12" // 规定
	}

	if cond.PracticeWrongSubmissionID > 0 {
		// 批改错题集练习的提交
		markerInfo.MarkMode = "14" // 规定
	}

	studentAnswers, err := QueryStudentAnswers(ctx, "00", cond, markerInfo)
	if err != nil {
		err = fmt.Errorf("failed to query student answers: %v", err)
		z.Error(err.Error())
		return err
	}

	// 没有作答直接返回
	if len(studentAnswers) <= 0 {
		z.Info("student no answers")
		return nil
	}

	marks := make([]cmn.TMark, len(studentAnswers))
	for i, studentAnswer := range studentAnswers {
		mark := cmn.TMark{
			TeacherID:            null.IntFrom(cond.TeacherID),
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
			err = fmt.Errorf("failed to unmarshal actual answers: %v", err)
			z.Error(err.Error())
			return err
		}

		if len(standardAnswers) <= 0 {
			err = fmt.Errorf("invalid actual answers")
			z.Error(err.Error())
			return err
		}

		var answers struct {
			Answer []string `json:"answer"`
		}

		if len(studentAnswer.Answer) != 0 {
			if err = json.Unmarshal(studentAnswer.Answer, &answers); err != nil {
				err = fmt.Errorf("failed to unmarshal answer: %v", err)
				z.Error(err.Error())
				return err
			}

			// 对比答案，正确给满分
			if CompareSlices(answers.Answer, standardAnswers) {
				mark.Score = null.FloatFrom(studentAnswer.QuestionScore.Float64)
			}
		} else { // 没有答案直接0分
			mark.Score = null.FloatFrom(0) // 应对空值
		}

		markDetails := make([]MarkDetails, 0, 10)

		markDetails = append(markDetails, MarkDetails{
			Index:   0,
			Score:   mark.Score.Float64,
			Analyze: "",
		})

		mark.MarkDetails, err = json.Marshal(markDetails)
		if err != nil || forceErr == "markObjectiveQuestionAnswers-json.Marshal" {
			err = fmt.Errorf("failed to marshal mark details: %v", err)
			z.Error(err.Error())
			return err
		}

		marks[i] = mark
	}

	_, err = updateStudentAnswerScore(ctx, tx, marks, cond)
	if err != nil {
		err = fmt.Errorf("failed to update student answer score: %v", err)
		z.Error(err.Error())
		return err
	}

	var subjectiveQuestionCounts int
	var whereClause []string

	// 查询该考试/练习的主观题数量
	querySubjectiveQuestionCounts := `
	SELECT COALESCE(count(epq.id), 0)
	FROM t_exam_paper ep
			 JOIN t_paper p ON p.exampaper_id = ep.id
			 LEFT JOIN t_exam_session es ON es.paper_id = p.id 
			 LEFT JOIN t_practice pra ON pra.paper_id = p.id 
			 JOIN t_exam_paper_group epg ON epg.exam_paper_id = ep.id 
			 LEFT JOIN t_exam_paper_question epq ON epq.group_id = epg.id AND epq.type IN ('06', '08') AND epq.status != '04'
	WHERE %s --动态拼接where条件
	`

	if cond.ExamSessionID > 0 {
		whereClause = append(whereClause, fmt.Sprintf(" AND es.id = %d", cond.ExamSessionID))
	} else if cond.PracticeID > 0 {
		whereClause = append(whereClause, fmt.Sprintf(" AND pra.id = %d", cond.PracticeID))
	}

	querySubjectiveQuestionCounts = fmt.Sprintf(querySubjectiveQuestionCounts, strings.Join(whereClause, " "))

	err = tx.QueryRow(ctx, querySubjectiveQuestionCounts).Scan(&subjectiveQuestionCounts)
	if err != nil || forceErr == "markObjectiveQuestionAnswers-pgxConn.QueryRow" {
		err = fmt.Errorf("failed to query subjective question counts: %v", err)
		z.Error(err.Error())
		return err
	}

	if subjectiveQuestionCounts != 0 {
		// 主观题数量大于0，不可以直接提交，所以退出
		return nil
	}

	// 考试/练习仅有客观题，改完自动提交
	z.Info("---------->no subjective question found, auto submit")

	if cond.PracticeWrongSubmissionID > 0 {
		// 更新批改错题集练习的提交状态
		_, err = updatePracticeWrongSubmissionState(ctx, tx, cond.TeacherID, []int64{cond.PracticeWrongSubmissionID}, "08")
		if err != nil {
			return err
		}
	}

	var status string
	var examSessionIDs []int64
	var practiceSubmissionIDs []int64
	// 考试：10， 练习提交：08
	if cond.ExamSessionID > 0 {
		status = "10"
		examSessionIDs = []int64{cond.ExamSessionID}
	} else if cond.PracticeSubmissionID > 0 {
		status = "08"
		practiceSubmissionIDs = []int64{cond.PracticeSubmissionID}
	}

	_, err = updateExamSessionOrPracticeSubmissionState(ctx, tx, cond.TeacherID, examSessionIDs, practiceSubmissionIDs, status)
	if err != nil {
		err = fmt.Errorf("update exam session state error: %v", err)
		z.Error(err.Error())
		return err
	}

	return nil
}

// --------------------- AI批改 -----------------------

// 自动批改处理函数，学生点击提交就会触发
func AutoMark(ctx context.Context, cond QueryCondition) error {
	forceErr, _ := ctx.Value(ForceErrKey).(string)

	if cond.ExamSessionID <= 0 && cond.PracticeID <= 0 {
		err := errors.New("invalid params: exam session id or practice id must be greater than zero")
		z.Error(err.Error())
		return err
	}

	if cond.ExamSessionID > 0 && cond.PracticeID > 0 {
		err := errors.New("invalid params: exam session id && practice id cannot be both greater than zero")
		z.Error(err.Error())
		return err
	}

	markerInfo, err := QueryMarkerInfo(ctx, cond)
	if err != nil {
		return err
	}

	if len(markerInfo.MarkInfos) == 0 {
		err = errors.New("no marker info found")
		z.Error(err.Error())
		return err
	}

	if markerInfo.MarkMode == "" {
		err = errors.New("invalid mark mode in marker info")
		z.Error(err.Error())
		return err
	}

	cond.TeacherID = markerInfo.MarkInfos[0].MarkTeacherID.Int64

	pgxConn := cmn.GetPgxConn()
	tx, err := pgxConn.Begin(ctx)
	if err != nil || forceErr == "AutoMark-pgxConn.Begin" {
		err = fmt.Errorf("begin transaction error: %v", err)
		z.Error(err.Error())
		return err
	}

	defer func() {
		if err != nil {
			if err := tx.Rollback(ctx); err != nil || forceErr == "AutoMark-tx.Rollback" {
				z.Error(err.Error())
			}
		}
	}()

	// 自动批改客观题，可以自己一个事务
	err = markObjectiveQuestionAnswers(ctx, tx, cond)
	if err != nil {
		return err
	}

	if markerInfo.MarkMode != "00" {
		// 不需要自动批改
		return nil
	}

	//TODO 自动(AI)批改主观题

	questionSets, err := QuerySubjectiveQuestions(ctx, cond, markerInfo)
	if err != nil {
		return err
	}

	var questions []*cmn.TExamPaperQuestion
	for _, questionSet := range questionSets {
		questions = append(questions, questionSet.Questions...)
	}

	if cond.PracticeWrongSubmissionID > 0 {
		// 练习学生作答错题
		markerInfo.MarkMode = "14"
	}

	studentAnswers, err := QueryStudentAnswers(ctx, "02", cond, markerInfo)
	if err != nil {
		return err
	}

	// 开启ai批改
	err = GenerateAIMarkTask(ctx, cond, questions, studentAnswers)
	if err != nil {
		return err
	}

	return nil
}

func InitAIMarkTaskLimiterAndBreaker() {
	maxConcurrency := viper.GetInt("chatModel.maxConcurrency")
	if maxConcurrency <= 0 {
		z.Error("从配置文件获取chatModel.maxConcurrency失败")
		maxConcurrency = 50
	}
	aiMarkTaskLimiter = cmn.NewRateLimiterTaskRunner(int64(maxConcurrency), 20, 40)

	st := gobreaker.Settings{
		Name:         "AI Mark breaker",
		MaxRequests:  10,
		Timeout:      15 * time.Second,
		Interval:     24 * time.Hour,
		BucketPeriod: 1 * time.Hour,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			// 熔断条件：请求数 >= 3 && (失败率 >= 50% || 连续失败次数 >= 3)
			return counts.Requests >= 5 && (failureRatio >= 0.5 || counts.ConsecutiveFailures >= 3)
		},
		// 监控熔断状态变化。
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			z.Sugar().Warnf("AI Mark Task Circuit Breaker %s changed from %s to %s\n", name, from, to)

			if to == gobreaker.StateOpen {
				z.Sugar().Warnf("AI Mark Task Circuit Breaker %s is open\n", name)
			}
		},
	}

	aiMarkTaskCB = gobreaker.NewCircuitBreaker[any](st)
}

func GetAIMarkTaskLimiter() *cmn.RateLimiterTaskRunner {
	if aiMarkTaskLimiter == nil {
		InitAIMarkTaskLimiterAndBreaker()
	}
	return aiMarkTaskLimiter
}

func GetAIMarkTaskCB() *gobreaker.CircuitBreaker[any] {
	if aiMarkTaskCB == nil {
		InitAIMarkTaskLimiterAndBreaker()
	}
	return aiMarkTaskCB
}

func GenerateAIMarkTask(ctx context.Context, cond QueryCondition, questions []*cmn.TExamPaperQuestion, studentAnswers []cmn.TVStudentAnswerQuestion) (err error) {
	if cond.TeacherID <= 0 {
		err = fmt.Errorf("用户ID无效")
		z.Error(err.Error())
		return
	}

	if cond.ExamSessionID <= 0 && cond.PracticeID <= 0 {
		err = fmt.Errorf("请求参数需包含场次id或者练习id")
		z.Error(err.Error())
		return
	}

	if cond.ExamSessionID > 0 && cond.PracticeID > 0 {
		err = fmt.Errorf("请求参数不能同时包含练习ID和考试场次ID")
		z.Error(err.Error())
		return
	}

	//chatModel, err := ai_mark.GetChatModel()
	//if err != nil {
	//	return
	//}

	questionAnswersMap := make(map[int64][]*ai_mark.StudentAnswer)

	for _, studentAnswer := range studentAnswers {
		if studentAnswer.QuestionID.Int64 <= 0 {
			err = fmt.Errorf("学生答案题目ID无效")
			z.Error(err.Error())
			return
		}

		if studentAnswer.ExamineeID.Int64 <= 0 && studentAnswer.PracticeSubmissionID.Int64 <= 0 {
			err = fmt.Errorf("学生答案考生ID和提交ID不能同时为空")
			z.Error(err.Error())
			return
		}

		if cond.ExamSessionID > 0 && studentAnswer.ExamineeID.Int64 <= 0 {
			err = fmt.Errorf("学生答案考生ID无效")
			z.Error(err.Error())
			return
		}

		if cond.PracticeID > 0 && studentAnswer.PracticeSubmissionID.Int64 <= 0 {
			err = fmt.Errorf("学生答案提交ID无效")
			z.Error(err.Error())
			return
		}

		if questionAnswersMap[studentAnswer.QuestionID.Int64] == nil {
			questionAnswersMap[studentAnswer.QuestionID.Int64] = []*ai_mark.StudentAnswer{}
		}

		var studentID int64
		if cond.ExamSessionID > 0 {
			studentID = studentAnswer.ExamineeID.Int64
		} else {
			studentID = studentAnswer.PracticeSubmissionID.Int64
		}

		if studentAnswer.Answer == nil || len(studentAnswer.Answer) == 0 {
			studentAnswer.Answer = []byte(`{"answer": []}`)
		}

		var answers struct {
			Answer []string `json:"answer"`
		}
		err = json.Unmarshal(studentAnswer.Answer, &answers)
		if err != nil {
			err = fmt.Errorf("解析学生答案失败: %v 原始输入: %v", err, string(studentAnswer.Answer))
			z.Error(err.Error())
			return
		}

		var studentAnswersBuilder strings.Builder
		for i, answer := range answers.Answer {
			indexStr := fmt.Sprintf("（%d）", i+1)
			studentAnswersBuilder.WriteString(indexStr)
			studentAnswersBuilder.WriteString(answer)
			studentAnswersBuilder.WriteString("\n")
		}

		a := &ai_mark.StudentAnswer{
			StudentID: studentID,
			Answer:    studentAnswersBuilder.String(),
		}

		questionAnswersMap[studentAnswer.QuestionID.Int64] = append(questionAnswersMap[studentAnswer.QuestionID.Int64], a)

	}

	var aiMarkRequests []AIMarkRequest

	var shouldSplit func(slice []*ai_mark.StudentAnswer) bool
	shouldSplit = func(slice []*ai_mark.StudentAnswer) bool {
		return len(slice) > 50
	}

	for _, q := range questions {
		if q.ID.Int64 <= 0 {
			err = fmt.Errorf("题目ID无效")
			z.Error(err.Error())
			return
		}

		var standardAnswers []*SubjectiveAnswer
		err = json.Unmarshal(q.Answers, &standardAnswers)
		if err != nil {
			err = fmt.Errorf("unable to convert raw standard answer data: %v", err)
			z.Error(err.Error())
			return
		}

		if len(standardAnswers) == 0 {
			err = fmt.Errorf("no standard answer found in question (%d)", q.ID.Int64)
			z.Error(err.Error())
			return
		}

		var answerBuilder strings.Builder
		var ruleBuilder strings.Builder
		for _, a := range standardAnswers {
			indexStr := fmt.Sprintf("（%d）", a.Index)
			answerBuilder.WriteString(indexStr)
			answerBuilder.WriteString(a.Answer)
			answerBuilder.WriteString("\n")
			ruleBuilder.WriteString(indexStr)
			ruleBuilder.WriteString(a.GradingRule)
			ruleBuilder.WriteString("\n")
		}

		// 50个作答为一组
		splitAnswers := splitSlice(questionAnswersMap[q.ID.Int64], shouldSplit)
		for _, answers := range splitAnswers {
			var aiMarkRequest = AIMarkRequest{
				Question: &ai_mark.QuestionDetails{
					QuestionID: q.ID.Int64,
					Answer:     answerBuilder.String(),
					Rule:       ruleBuilder.String(),
					Score:      q.Score.Float64,
				},
				StudentAnswers: answers,
			}

			aiMarkRequests = append(aiMarkRequests, aiMarkRequest)
		}
	}

	redisClient := cmn.GetRedisConn()

	var uniqueKey = TaskTypeAIMarkRequest

	if cond.ExamSessionID > 0 {
		uniqueKey += ":exam_session:" + strconv.FormatInt(cond.ExamSessionID, 10)
		if cond.ExamineeID > 0 {
			uniqueKey += ":examinee:" + strconv.FormatInt(cond.ExamineeID, 10)
		}
	} else {
		uniqueKey += ":practice:" + strconv.FormatInt(cond.PracticeID, 10)
		if cond.PracticeSubmissionID > 0 {
			uniqueKey += ":practice_submission:" + strconv.FormatInt(cond.PracticeSubmissionID, 10)
		}
	}

	uniqueTaskCountKey := uniqueKey + ":count"

	err = redisClient.Set(ctx, uniqueTaskCountKey, len(aiMarkRequests), 7*24*time.Hour).Err()

	// 设置唯一性约束：24小时内，不允许重复入队
	opts := []asynq.Option{
		asynq.MaxRetry(5),
		asynq.Unique(24 * time.Hour),
		asynq.Timeout(5 * time.Minute),
	}

	for _, aiMarkRequest := range aiMarkRequests {
		//	taskPayload := cmn.TaskPayload{
		//		Type: TaskTypeAIMarkRequest,
		//		Data: AIMarkTaskPayLoad{
		//			AIMarkRequest:      aiMarkRequest,
		//			QueryCondition:     cond,
		//			UniqueTaskCountKey: uniqueTaskCountKey,
		//		},
		//		Timestamp: time.Now().Unix(),
		//	}
		//
		//	var payloadBytes []byte
		//	payloadBytes, err = json.Marshal(taskPayload)
		//	if err != nil {
		//		err = fmt.Errorf("Failed to marshal task payload: %v", err)
		//		z.Sugar().Error(err.Error())
		//		return err
		//	}
		//
		//	task := asynq.NewTask(string(TaskTypeAIMarkRequest), payloadBytes, opts...)
		//
		//	h := TaskMiddleware(HandleAIMarkTask)
		//	err = h(ctx, task)
		//	if err != nil {
		//		err = fmt.Errorf("failed to handle ai mark task: %v", err)
		//		z.Sugar().Error(err.Error())
		//		return
		//	}

		// TODO 原实现，不够完善
		err = cmn.SendTask(TaskTypeAIMarkRequest, AIMarkTaskPayLoad{
			AIMarkRequest:      aiMarkRequest,
			QueryCondition:     cond,
			UniqueTaskCountKey: uniqueTaskCountKey,
		}, opts...)
		if err != nil {
			return
		}
	}

	return

}

func TaskMiddleware(handler func(ctx context.Context, task *asynq.Task) error) func(ctx context.Context, task *asynq.Task) error {
	return func(ctx context.Context, task *asynq.Task) error {
		limiter := GetAIMarkTaskLimiter()

		z.Info("waiting limiter")

		err, releaseFunc := limiter.Wait(ctx)
		if releaseFunc != nil {
			defer releaseFunc()
		}
		if err != nil {
			// 获取信号量失败，直接返回错误
			return err
		}
		z.Info("limiter waiting over")
		// 释放信号量

		cb := GetAIMarkTaskCB()

		_, err = cb.Execute(func() (any, error) {
			return nil, handler(ctx, task)
		})

		return err
	}
}

func HandleAIMarkTask(ctx context.Context, task *asynq.Task) error {
	forceErr, _ := ctx.Value(ForceErrKey).(string)

	var err error
	var taskPayLoad cmn.TaskPayload

	err = json.Unmarshal(task.Payload(), &taskPayLoad)
	if err != nil {
		err = fmt.Errorf("failed to unmarshal task payload: %v", err)
		z.Error(err.Error())
		return err
	}

	//z.Sugar().Infof("HandleAIMarkTask: %+v", taskPayLoad)

	jsonData, err := json.Marshal(taskPayLoad.Data)
	if err != nil || forceErr == "HandleAIMarkTask-json.Marshal-payload-data" {
		err = fmt.Errorf("failed to marshal task payload data: %v", err)
		z.Error(err.Error())
		return err
	}

	var payloadData AIMarkTaskPayLoad
	err = json.Unmarshal(jsonData, &payloadData)
	if err != nil {
		err = fmt.Errorf("failed to unmarshal task payload data: %v, raw data: %+v", err, string(jsonData))
		z.Error(err.Error())
		return err
	}

	z.Sugar().Infof("HandleAIMarkTask: %+v", payloadData.UniqueTaskCountKey)

	if payloadData.UniqueTaskCountKey == "" {
		err = fmt.Errorf("任务payload中的任务计数键为空")
		z.Error(err.Error())
		return err
	}

	if payloadData.QueryCondition.TeacherID <= 0 {
		err = fmt.Errorf("任务payload中的教师ID无效")
		z.Error(err.Error())
		return err
	}

	if payloadData.QueryCondition.ExamSessionID <= 0 && payloadData.QueryCondition.PracticeID <= 0 {
		err = fmt.Errorf("任务payload中的考试场次ID与练习ID都无效")
		z.Error(err.Error())
		return err
	}

	if payloadData.QueryCondition.ExamSessionID > 0 && payloadData.QueryCondition.PracticeID > 0 {
		err = fmt.Errorf("任务payload中的考试场次ID与练习ID不能同时有效")
		z.Error(err.Error())
		return err
	}

	chatModel, err := ai_mark.GetChatModel()
	if err != nil {
		return err
	}

	respContent, err := chatModel.AIMark(ctx, payloadData.AIMarkRequest.Question, payloadData.AIMarkRequest.StudentAnswers)
	if err != nil {
		return err
	}
	//z.Sugar().Infof("chatResp: %+v", respContent)
	//
	//z.Sugar().Infof("HandleAIMarkTask data->: %+v", payloadData)

	var marks []cmn.TMark

	for _, result := range respContent.MarkResult {
		var markDetails = MarkDetails{
			Index:   1,
			Score:   result.Score,
			Analyze: result.Analyze,
		}

		var markDetailsJSON []byte
		markDetailsJSON, err = json.Marshal(markDetails)
		if err != nil || forceErr == "HandleAIMarkTask-json.Marshal-mark-details" {
			err = fmt.Errorf("failed to marshal mark details: %v", err)
			z.Error(err.Error())
			return err
		}

		var mark = cmn.TMark{
			TeacherID:   null.IntFrom(payloadData.QueryCondition.TeacherID),
			QuestionID:  null.IntFrom(payloadData.AIMarkRequest.Question.QuestionID),
			Score:       null.FloatFrom(result.Score),
			MarkDetails: markDetailsJSON,
			Creator:     null.IntFrom(payloadData.QueryCondition.TeacherID),
			CreateTime:  null.IntFrom(time.Now().UnixMilli()),
		}

		if payloadData.QueryCondition.ExamSessionID > 0 {
			// 考试会话
			mark.ExamSessionID = null.IntFrom(payloadData.QueryCondition.ExamSessionID)
			mark.ExamineeID = null.IntFrom(result.StudentID)
		} else {
			// 练习会话
			mark.PracticeID = null.IntFrom(payloadData.QueryCondition.PracticeID)
			mark.PracticeSubmissionID = null.IntFrom(result.StudentID)
		}

		marks = append(marks, mark)
	}

	pgxConn := cmn.GetPgxConn()
	tx, err := pgxConn.Begin(ctx)
	if err != nil || forceErr == "pgxConn.Begin" {
		err = fmt.Errorf("begin transaction error: %w", err)
		z.Error(err.Error())
		return err
	}

	defer func() {
		if err != nil || forceErr == "tx.Rollback" {
			if err = tx.Rollback(ctx); err != nil {
				err = fmt.Errorf("rollback tx error: %w", err)
				z.Error(err.Error())
			}
		}
	}()

	_, err = InsertOrUpdateMarkingResults(ctx, tx, marks)
	if err != nil {
		return err
	}

	_, err = updateStudentAnswerScore(ctx, tx, marks, payloadData.QueryCondition)
	if err != nil {
		return err
	}

	redisClient := cmn.GetRedisConn()

	// 定义 Lua 脚本
	luaScript := `
		local current = redis.call('GET', KEYS[1])
		if not current then
			return "04"
		end
		local num = tonumber(current)
		if num == 1 then
			return "02"
		elseif num <= 0 then 
			return "06"
		else
			redis.call('DECR', KEYS[1])
			return "00"
		end
	`

	scriptRunCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	script := redis.NewScript(luaScript)

	result, err := script.Run(scriptRunCtx, redisClient, []string{payloadData.UniqueTaskCountKey}).Result()
	if err != nil || forceErr == "HandleAIMarkTask-script.Run().Result" {
		err = fmt.Errorf("lua脚本执行出错: %v", err)
		z.Error(err.Error())
		return err
	}

	status, ok := result.(string)
	if !ok || forceErr == "HandleAIMarkTask-script-result.(string)" {
		err = fmt.Errorf("lua脚本返回结果类型错误")
		z.Error(err.Error())
		return err
	}

	switch status {
	case "00":
		// 减库存成功
		break
	case "02":
		// 任务已全部完成
		z.Sugar().Infof("本批次批改任务已全部完成")

		// 删除该key
		err = redisClient.Del(scriptRunCtx, payloadData.UniqueTaskCountKey).Err()
		if err != nil || forceErr == "HandleAIMarkTask-redisClient.Del" {
			err = fmt.Errorf("删除key失败: %w", err)
			z.Error(err.Error())
			return err
		}

		if payloadData.QueryCondition.PracticeSubmissionID < 0 {
			return nil
		}

		if payloadData.QueryCondition.PracticeWrongSubmissionID > 0 {
			_, err = updatePracticeWrongSubmissionState(ctx, tx, payloadData.QueryCondition.TeacherID, []int64{payloadData.QueryCondition.PracticeWrongSubmissionID}, "08")
			if err != nil {
				return err
			}
		}

		_, err = updateExamSessionOrPracticeSubmissionState(ctx, tx, payloadData.QueryCondition.TeacherID, nil, []int64{payloadData.QueryCondition.PracticeSubmissionID}, "08")
		if err != nil {
			return err
		}

	case "04":
		// 找不到该计数键值
		err = fmt.Errorf("找不到任务计数键值：%s", payloadData.UniqueTaskCountKey)
		z.Error(err.Error())
		return err

	case "06":
		// 超减计数键值
		err = fmt.Errorf("超减任务计数键值：%s", payloadData.UniqueTaskCountKey)
		z.Error(err.Error())
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		z.Error(fmt.Errorf("tx commit error: %v", err).Error())
		return err
	}

	return nil
}
