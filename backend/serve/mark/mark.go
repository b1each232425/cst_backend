package mark

//annotation:mark-service
//author:{"name":"Lin ZiXin","tel":"13430362233","email":"linzixinzz1@gmail.com"}

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jmoiron/sqlx/types"
	"io"
	"strconv"
	"strings"
	"time"
	"w2w.io/null"
	"w2w.io/serve/examPaper"

	"go.uber.org/zap"
	"w2w.io/cmn"
)

var z *zap.Logger

const ForceErrKey = "force-err"

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

type MarkingResult struct {
	ExamineeID           int64  `json:"examinee_id"`
	PracticeSubmissionID int64  `json:"practice_submission_id"`
	Marks                []Mark `json:"marks"`
}

type Mark struct {
	Details        []Details      `json:"details"`
	DetailsJSON    types.JSONText `json:"details_json"`
	Score          int            `json:"score"`
	QuestionNumber int            `json:"question_number"`
	QuestionID     int64          `json:"question_id"`
	Marker         int64          `json:"marker"`
}

type Details struct {
	Analyze string `json:"analyze"`
	Index   int    `json:"index"`
	Score   int    `json:"score"`
}

type HandleMarkerInfoReq struct {
	Markers        []int64                             `json:"markers"`          // *批改员id数组
	QuestionGroups []examPaper.SubjectiveQuestionGroup `json:"question_groups"`  // *题组（配置时传入）
	QuestionIDs    []int64                             `json:"question_ids"`     // 题目id数组
	ExamineeIDs    []int64                             `json:"examinee_ids"`     // 考生id数组
	MarkMode       string                              `json:"mark_mode"`        // *批卷模式 00：不需要手动批改  02：全卷多评 04：试卷分配 06：题组专评 08：题目分配 10：单人（人工）批改
	ExamSessionID  int64                               `json:"exam_session_id"`  // 考试场次id
	PracticeID     int64                               `json:"practice_id"`      // 练习id
	Status         string                              `json:"status"`           // *00 插入批改配置 02 删除批改配置
	ExamSessionIDs []int64                             `json:"exam_session_ids"` // 要删除的考试场次id数组
	PracticeIDs    []int64                             `json:"practice_ids"`     // 要删除的练习id数组
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
		Fn: HandleExamList,

		Path: "/mark/exam",
		Name: "mark.get-exam-list",

		Developer: developer,
		WhiteList: true,

		DomainID: int64(cmn.CDomainSys),

		DefaultDomain: int64(cmn.CDomainSys),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: HandleExamList,

		Path: "/mark/practice",
		Name: "mark.get-practice-list",

		Developer: developer,
		WhiteList: true,

		DomainID: int64(cmn.CDomainSys),

		DefaultDomain: int64(cmn.CDomainSys),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: HandleMarkingDetails,

		Path: "/mark/details",
		Name: "/mark/details",

		Developer: developer,
		WhiteList: true,

		DomainID: int64(cmn.CDomainSys),

		DefaultDomain: int64(cmn.CDomainSys),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: HandleMarkingResults,

		Path: "/mark/marking-results",
		Name: "/mark/marking-results",

		Developer: developer,
		WhiteList: true,

		DomainID: int64(cmn.CDomainSys),

		DefaultDomain: int64(cmn.CDomainSys),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: HandleResultsSubmission,

		Path: "/mark/results-submission",
		Name: "/mark/results-submission",

		Developer: developer,
		WhiteList: true,

		DomainID: int64(cmn.CDomainSys),

		DefaultDomain: int64(cmn.CDomainSys),
	})

}

func HandleExamList(ctx context.Context) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())
	method := strings.ToLower(q.R.Method)
	if method != "get" {
		q.Err = fmt.Errorf("please call /api/mark/exam with http GET method")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	queryParams := q.R.URL.Query()

	pageIndexStr := queryParams.Get("page_index")
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
		q.Err = fmt.Errorf("page index must be greater than 0")
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
		q.Err = fmt.Errorf("page size must be between 1 and 1000")
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
		//z.Sugar().Infof("start_time: %s, end_time: %s", startTimeStr, endTimeStr)

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

	//z.Sugar().Infof("======(%v)===>>: %+v", rowCount, exams)

	if err != nil || forceErr == "HandleExamList-json.Marshal" {
		q.Err = fmt.Errorf("unable to marshal response data: %v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	q.Msg.Data = jsonData
	q.Msg.RowCount = int64(rowCount)
	q.Err = nil
	q.Resp()
	return
}

func HandlePracticeList(ctx context.Context) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())
	method := strings.ToLower(q.R.Method)
	if method != "get" {
		q.Err = fmt.Errorf("please call /api/mark/practice with http GET method")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	queryParams := q.R.URL.Query()

	pageIndexStr := queryParams.Get("page_index")
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
		q.Err = fmt.Errorf("page index must be greater than 0")
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
		q.Err = fmt.Errorf("page size must be between 1 and 1000")
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
		//z.Sugar().Infof("start_time: %s, end_time: %s", startTimeStr, endTimeStr)

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

	//z.Sugar().Infof("======(%v)===>>: %+v", rowCount, exams)

	if err != nil || forceErr == "HandleExamList-json.Marshal" {
		q.Err = fmt.Errorf("unable to marshal response data: %v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	q.Msg.Data = jsonData
	q.Msg.RowCount = int64(rowCount)
	q.Err = nil
	q.Resp()
	return
}

func HandleMarkingDetails(ctx context.Context) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())
	method := strings.ToLower(q.R.Method)
	if method != "get" {
		q.Err = fmt.Errorf("please call /api/mark/details with http GET method")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	queryParams := q.R.URL.Query()

	examSessionIDStr := queryParams.Get("exam_session_id")
	practiceIDStr := queryParams.Get("practice_id")
	if examSessionIDStr == "" && practiceIDStr == "" {
		q.Err = fmt.Errorf("请求参数必须包含练习ID或者考试场次ID中的一个")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	if practiceIDStr != "" && examSessionIDStr != "" {
		q.Err = fmt.Errorf("请求参数不能同时包含练习ID和考试场次ID")
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

	examineeIDStr := queryParams.Get("examinee_id")
	var examineeID int64
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
	var practiceSubmissionID int64
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

	studentInfos, err := QueryStudentsInfo(ctx, cond)
	if err != nil {
		q.Err = err
		q.RespErr()
		return
	}

	markingResults, err := QueryMarkingResults(ctx, cond)
	if err != nil {
		q.Err = err
		q.RespErr()
		return
	}

	markerInfo, err := QueryMarkerInfo(ctx, cond)
	if err != nil {
		q.Err = err
		q.RespErr()
		return
	}

	if markerInfo.MarkInfos == nil || len(markerInfo.MarkInfos) == 0 {
		q.Err = fmt.Errorf("no marker info")
		q.RespErr()
		return
	}

	questionSets, err := QueryQuestionsByMarkMode(ctx, cond, markerInfo)
	if err != nil {
		q.Err = err
		q.RespErr()
		return
	}

	studentAnswers, err := QueryStudentAnswersByMarkMode(ctx, "00", cond, markerInfo)
	if err != nil {
		q.Err = err
		q.RespErr()
		return
	}

	jsonData, err := json.Marshal(map[string]interface{}{
		"student_infos":   studentInfos,
		"student_answers": studentAnswers,
		"question_sets":   questionSets,
		"marking_results": markingResults,
	})
	if err != nil || forceErr == "HandleMarkingDetails-json.Marshal" {
		q.Err = fmt.Errorf("failed to marshal response data: %v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	//z.Sugar().Infof("getExamList response: %s", string(jsonData))

	q.Msg.Data = jsonData
	q.Err = nil
	q.Resp()
	return

}

func HandleMarkerInfo(ctx context.Context, tx *pgx.Tx, teacherID int64, req HandleMarkerInfoReq) (err error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	if teacherID <= 0 {
		err = fmt.Errorf("invalid teacherID")
		z.Error(err.Error())
		return
	}

	if req.Status != "00" && req.Status != "02" {
		err = fmt.Errorf("invalid status")
		z.Error(err.Error())
		return
	}

	if req.Status == "02" {
		var mode string
		var ids []int64

		if len(req.PracticeIDs) > 0 {
			// 删除练习批改配置
			ids = req.PracticeIDs
			mode = "02"
		} else {
			if req.ExamSessionIDs == nil || len(req.ExamSessionIDs) == 0 {
				// 删除考试批改配置
				err = fmt.Errorf("no examSessionIDs to update mark info state")
				z.Error(err.Error())
				return
			}

			ids = req.ExamSessionIDs
			mode = "00"
		}

		_, err = UpdateMarkerInfoState(ctx, tx, teacherID, ids, mode)
		if err != nil {
			return
		}
		return
	}

	if req.ExamSessionID <= 0 && req.PracticeID <= 0 {
		err = fmt.Errorf("invalid exam_session_id && practice_id")
		z.Error(err.Error())
		return
	}

	var markInfos []cmn.TMarkInfo

	switch req.MarkMode {
	case "00":
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
		break
	case "02": // 全卷多评，多位批阅员
		if req.Markers == nil || len(req.Markers) == 0 {
			err = fmt.Errorf("markers is empty")
			z.Error(err.Error())
			return
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

		break
	case "04": // 试卷分配
		if req.ExamineeIDs == nil || len(req.ExamineeIDs) == 0 {
			err = fmt.Errorf("examinee_ids is empty")
			z.Error(err.Error())
			return
		}

		// 将id数组打乱平均分成n组
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
		break
	case "06": // 分题组
		if req.QuestionGroups == nil || len(req.QuestionGroups) == 0 {
			err = fmt.Errorf("invalid question groups")
			z.Error(err.Error())
			return
		}

		// 分题组
		splitGroups := randomSplit(req.QuestionGroups, len(req.Markers))
		for i, groups := range splitGroups {
			var questionIDs []int64
			for _, group := range groups {
				questionIDs = append(questionIDs, group.QuestionIDs...)
			}

			var markQuestionIDsBytes []byte
			markQuestionIDsBytes, err = json.Marshal(questionIDs)
			if err != nil || forceErr == "json.Marshal-2" {
				err = fmt.Errorf("failed to marshal markQuestionIDs: %v", err)
				return
			}

			markInfos = append(markInfos, cmn.TMarkInfo{
				ExamSessionID:   null.IntFrom(req.ExamSessionID),
				MarkTeacherID:   null.IntFrom(req.Markers[i]),
				MarkCount:       null.IntFrom(0),
				MarkExamineeIds: nil,
				QuestionIds:     markQuestionIDsBytes,
				Creator:         null.IntFrom(teacherID),
				UpdatedBy:       null.IntFrom(teacherID),
				CreateTime:      null.IntFrom(time.Now().UnixMilli()),
				Status:          null.StringFrom("00"),
			})
		}
		break
	case "10": // 单人批改
		if req.Markers == nil || len(req.Markers) == 0 || len(req.Markers) > 1 {
			err = fmt.Errorf("invalid markers")
			z.Error(err.Error())
			return
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
		break
	}

	//z.Sugar().Infof("markInfos: %+v", markInfos)

	if len(markInfos) <= 0 {
		err = fmt.Errorf("no markInfos to insert")
		z.Error(err.Error())
		return
	}

	insertQuery := `INSERT INTO t_mark_info 
						(exam_session_id, practice_id, mark_teacher_id, mark_count, question_ids, mark_examinee_ids, creator, create_time, updated_by, update_time, addi, status) 
					VALUES 
						($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) 
					RETURNING id`

	var targetIDs []int64
	for _, info := range markInfos {
		var id null.Int
		err = (*tx).QueryRow(ctx, insertQuery, info.ExamSessionID, info.PracticeID, info.MarkTeacherID, info.MarkCount, info.QuestionIds, info.MarkExamineeIds, info.Creator, info.CreateTime, info.UpdatedBy, info.UpdateTime, info.Addi, info.Status).Scan(&id)
		if err != nil || forceErr == "HandleMarkerInfo-tx.QueryRow" {
			err = fmt.Errorf("exec insert query error: %v", err)
			z.Error(err.Error())
			return
		}
		targetIDs = append(targetIDs, id.Int64)
	}

	return
}

func HandleMarkingResults(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())
	forceErr, _ := ctx.Value(ForceErrKey).(string) // 用于强制执行错误处理代码
	method := strings.ToLower(q.R.Method)
	if method != "post" {
		q.Err = fmt.Errorf("please call /api/mark/marking-results with http post method")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	// 从body中获取
	var buf []byte
	buf, q.Err = io.ReadAll(q.R.Body)
	if q.Err != nil || forceErr == "io.ReadAll" {
		q.Err = fmt.Errorf("failed to read request body: %w", q.Err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	defer func() {
		err := q.R.Body.Close()
		if err != nil || forceErr == "io.Close" {
			err = fmt.Errorf("failed to close request body: %w", err)
			z.Error(err.Error())
			return
		}
	}()

	if len(buf) == 0 {
		q.Err = fmt.Errorf("call /mark/marking-results with empty body")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	var body cmn.ReqProto
	q.Err = json.Unmarshal(buf, &body)
	if q.Err != nil {
		q.Err = fmt.Errorf("failed to unmarshal request body: %v", q.Err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	var markingResults []*cmn.TMark
	q.Err = json.Unmarshal(body.Data, &markingResults)
	if q.Err != nil {
		q.Err = fmt.Errorf("failed to unmarshal marking results: %v", q.Err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	if len(markingResults) == 0 {
		q.Err = fmt.Errorf("invalid request body: no marking results provided")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	_, q.Err = InsertOrUpdateMarkingResults(ctx, markingResults)
	if q.Err != nil {
		q.Err = fmt.Errorf("failed to insert or update marking results: %v", q.Err)
		z.Error(q.Err.Error())
		q.RespErr()

		z.Sugar().Infof("---------> failed called %v", q.Err)
		return
	}

	z.Sugar().Infof("---------> success called")
	q.Err = nil
	q.Resp()
	return

}

func HandleResultsSubmission(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())
	forceErr, _ := ctx.Value(ForceErrKey).(string) // 用于强制执行错误处理代码
	method := strings.ToLower(q.R.Method)
	if method != "patch" {
		q.Err = fmt.Errorf("please call /api/mark/results-submission with http patch method")
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

	results, err := QueryMarkingResults(ctx, cond)
	if err != nil {
		q.Err = err
		q.RespErr()
		return
	}

	_, err = updateStudentAnswerScore(ctx, results, cond)
	if err != nil {
		q.Err = err
		q.RespErr()
		return
	}

	var status string
	var examSessionIDs []int64
	var practiceSubmissionIDs []int64
	if cond.ExamSessionID > 0 {
		status = "10"
		examSessionIDs = []int64{cond.ExamSessionID}
	} else if cond.PracticeSubmissionID > 0 {
		status = "08"
		practiceSubmissionIDs = []int64{cond.PracticeSubmissionID}
	}

	pgxConn := cmn.GetPgxConn()

	tx, err := pgxConn.Begin(ctx)
	defer func() {
		if err != nil {
			err_ := tx.Rollback(ctx)
			if err_ != nil || forceErr == "HandleResultsSubmission-tx.Rollback" {
				z.Sugar().Error(err_)
			}
		} else {
			err_ := tx.Commit(ctx)
			if err_ != nil || forceErr == "HandleResultsSubmission-tx.Commit" {
				z.Sugar().Error(err_)
			}
		}
	}()

	_, err = updateExamSessionOrPracticeSubmissionState(ctx, &tx, cond.TeacherID, examSessionIDs, practiceSubmissionIDs, status)
	if err != nil {
		q.Err = err
		q.RespErr()
		return
	}
}

func MarkObjectiveQuestionAnswers(ctx context.Context, cond QueryCondition) (err error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	err = cmn.Validate(cond)
	if err != nil {
		err = fmt.Errorf("invalid query condition: %v", err)
		z.Error(err.Error())
		return err
	}

	if cond.ExamSessionID <= 0 && cond.PracticeID <= 0 {
		err = fmt.Errorf("invalid params: exam session id or practice id must be greater than zero")
		z.Error(err.Error())
		return
	}

	if cond.ExamSessionID > 0 && cond.PracticeID > 0 {
		err = fmt.Errorf("invalid params: exam session id && practice id cannot be both greater than zero")
		z.Error(err.Error())
		return
	}

	markerInfo := MarkerInfo{
		MarkerID: cond.TeacherID,
	}

	if cond.ExamineeID > 0 {
		// 查询单个考试考生
		markerInfo.MarkMode = "04"
	}

	if cond.PracticeSubmissionID > 0 {
		// 查询单个练习学生
		markerInfo.MarkMode = "12"
	}

	studentAnswers, err := QueryStudentAnswersByMarkMode(ctx, "02", cond, markerInfo)
	if err != nil {
		err = fmt.Errorf("failed to query student answers: %v", err)
		z.Error(err.Error())
		return err
	}

	marks := make([]*cmn.TMark, len(studentAnswers))
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
			return
		}

		if len(standardAnswers) <= 0 {
			err = fmt.Errorf("invalid actual answers")
			z.Error(err.Error())
			return
		}

		var answers struct {
			Answer []string `json:"answer"`
		}
		err = json.Unmarshal(studentAnswer.Answer, &answers)
		if err != nil {
			err = fmt.Errorf("failed to unmarshal answer: %v", err)
			z.Error(err.Error())
			return
		}

		if CompareSlices(answers.Answer, standardAnswers) {
			mark.Score = null.FloatFrom(studentAnswer.QuestionScore.Float64)
		}

		var markDetails []Details

		markDetails = append(markDetails, Details{
			Index:   0,
			Score:   int(mark.Score.Float64),
			Analyze: "",
		})

		mark.MarkDetails, err = json.Marshal(markDetails)
		if err != nil || forceErr == "MarkObjectiveQuestionAnswers-json.Marshal" {
			err = fmt.Errorf("failed to marshal mark details: %v", err)
			z.Error(err.Error())
			return
		}

		marks[i] = &mark
	}

	_, err = updateStudentAnswerScore(ctx, marks, cond)
	if err != nil {
		err = fmt.Errorf("failed to update student answer score: %v", err)
		z.Error(err.Error())
		return
	}

	var subjectiveQustionCounts int
	var whereClause []string

	// 查询该考试/练习的主观题数量
	querySubjectiveQuestionCounts := `
	SELECT COALESCE(count(q.id), 0)
	FROM t_exam_paper p
			 LEFT JOIN t_exam_session es ON p.exam_session_id = es.id
			 LEFT JOIN t_practice pra ON p.practice_id = pra.id
			 JOIN t_exam_paper_group pg ON pg.exam_paper_id = p.id 
			 LEFT JOIN t_exam_paper_question q ON q.group_id = pg.id AND q.type IN ('06', '08') 
	WHERE q.status != '04' %s --动态拼接where条件
	`

	if cond.ExamSessionID > 0 {
		whereClause = append(whereClause, fmt.Sprintf(" AND es.id = %d", cond.ExamSessionID))
	} else if cond.PracticeID > 0 {
		whereClause = append(whereClause, fmt.Sprintf(" AND p.id = %d", cond.PracticeID))
	}

	querySubjectiveQuestionCounts = fmt.Sprintf(querySubjectiveQuestionCounts, strings.Join(whereClause, " "))

	pgxConn := cmn.GetPgxConn()

	err = pgxConn.QueryRow(ctx, querySubjectiveQuestionCounts).Scan(&subjectiveQustionCounts)
	if err != nil || forceErr == "MarkObjectiveQuestionAnswers-pgxConn.QueryRow" {
		err = fmt.Errorf("failed to query subjective question counts: %v", err)
		z.Error(err.Error())
		return
	}

	if subjectiveQustionCounts != 0 {
		// 主观题数量大于0，直接结束
		return
	}

	// 考试/练习仅有客观题，改完自动提交
	z.Info("---------->no subjective question found, auto submit")
	var tx pgx.Tx
	tx, err = pgxConn.Begin(ctx)
	if err != nil || forceErr == "MarkObjectiveQuestionAnswers-pgxConn.Begin" {
		err = fmt.Errorf("begin transaction error: %v", err)
		z.Error(err.Error())
		return err
	}

	defer func() {
		if err != nil || forceErr == "MarkObjectiveQuestionAnswers-tx.Rollback" {
			err_ := tx.Rollback(ctx)
			if err_ != nil {
				z.Error(err_.Error())
				return
			}
		}
	}()

	var status string
	var examSessionIDs []int64
	var practiceSubmissionIDs []int64
	if cond.ExamSessionID > 0 {
		status = "10"
		examSessionIDs = []int64{cond.ExamSessionID}
	} else if cond.PracticeSubmissionID > 0 {
		status = "08"
		practiceSubmissionIDs = []int64{cond.PracticeSubmissionID}
	}

	_, err = updateExamSessionOrPracticeSubmissionState(ctx, &tx, cond.TeacherID, examSessionIDs, practiceSubmissionIDs, status)
	if err != nil {
		err = fmt.Errorf("update exam session state error: %v", err)
		z.Error(err.Error())
		return err
	}

	err = tx.Commit(ctx)
	if err != nil || forceErr == "MarkObjectiveQuestionAnswers-tx.Commit" {
		err = fmt.Errorf("commit tx error: %v", err)
		z.Error(err.Error())
		return
	}

	return
}

// AutoMark 自动批改处理函数
func AutoMark(ctx context.Context, cond QueryCondition) (err error) {
	if cond.ExamSessionID <= 0 && cond.PracticeID <= 0 {
		err = fmt.Errorf("invalid params: exam session id or practice id must be greater than zero")
		z.Error(err.Error())
		return
	}

	if cond.ExamSessionID > 0 && cond.PracticeID > 0 {
		err = fmt.Errorf("invalid params: exam session id && practice id cannot be both greater than zero")
		z.Error(err.Error())
		return
	}

	markerInfo, err := QueryMarkerInfo(ctx, cond)
	if err != nil {
		return
	}

	if len(markerInfo.MarkInfos) == 0 {
		err = fmt.Errorf("no marker info found")
		z.Error(err.Error())
		return
	}

	if markerInfo.MarkMode == "" {
		err = fmt.Errorf("invalid mark mode in marker info")
		z.Error(err.Error())
		return
	}

	if markerInfo.MarkMode != "00" {
		// 不需要自动批改
		return nil
	}

	cond.TeacherID = markerInfo.MarkInfos[0].MarkTeacherID.Int64

	err = MarkObjectiveQuestionAnswers(ctx, cond)
	if err != nil {
		return
	}

	return
}
