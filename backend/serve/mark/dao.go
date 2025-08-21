package mark

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jmoiron/sqlx/types"
	"github.com/lib/pq"
	"strings"
	"time"
	"w2w.io/cmn"
	"w2w.io/null"
)

type Exam struct {
	Id           int64         `json:"id"`
	Name         string        `json:"name"`
	Type         string        `json:"type"`
	Class        string        `json:"class"`
	ExamSessions []ExamSession `json:"exam_sessions"`
}

type ExamSession struct {
	Id                   int64  `json:"id"`
	Name                 string `json:"name"`
	PaperName            string `json:"paper_name"`
	ExamSessionType      string `json:"session_type"`
	MarkMethod           string `json:"mark_method"`
	MarkMode             string `json:"mark_mode"`
	RespondentCount      int    `json:"respondent_count"`
	UnMarkedStudentCount int    `json:"unmarked_student_count"`
	StartTime            int64  `json:"start_time"`
	EndTime              int64  `json:"end_time"`
	Status               string `json:"status"`
	MarkStatus           string `json:"mark_status"`
}

type Practice struct {
	ID                   int64  `json:"id"`                     //练习id
	Name                 string `json:"name"`                   // 练习名称
	MarkMode             string `json:"mark_mode"`              // 批阅方式
	Type                 string `json:"type"`                   // 练习类型
	RespondentCount      int    `json:"respondent_count"`       // 作答人数
	UnMarkedStudentCount int    `json:"unmarked_student_count"` // 未批阅人数
}

type QueryCondition struct {
	TeacherID            int64 `json:"teacher_id" validate:"required,gt=0"`
	ExamSessionID        int64 `json:"exam_session_id" validate:"gte=0"`
	ExamineeID           int64 `json:"examinee_id" validate:"gte=0"`
	PracticeID           int64 `json:"practice_id" validate:"gte=0"`
	PracticeSubmissionID int64 `json:"practice_submission_id" validate:"gte=0"`
}

type MarkerInfo struct {
	MarkerID           int64           `json:"marker_id"`
	MarkInfos          []cmn.TMarkInfo `json:"mark_infos"`
	MarkMode           string          `json:"mark_mode"` // 00：不需要手动批改  02：全卷多评 04：试卷分配 06：题组专评 08：题目分配 10：单人（人工）批改 12：批改单个练习学生
	MarkMethod         string          `json:"mark_method"`
	MarkedStudentNames []string        `json:"marked_student_names"`
}

type QuestionSet struct {
	cmn.TExamPaperGroup
	Score     float64                   `json:"Score"`     // 题组分数
	Questions []*cmn.TExamPaperQuestion `json:"Questions"` // 题目
}

// Question 题目
//type Question struct {
//	ID               int64              `json:"id"`
//	Analyze          string             `json:"analyze"`
//	MarkRole         string             `json:"mark_role"`
//	StandardAnswers  []*StandardAnswer  `json:"standard_answers"`
//	ObjectiveAnswers []string           `json:"objective_answers"`
//	Details          []*QuestionDetails `json:"details"`
//	OrderNum         int                `json:"order_num"`
//	Type             string             `json:"type"` // 00:单选题  02:多选题 04:判断题 06:填空题 08:简答题 10:编程题
//	Score            float64            `json:"score"`
//}

// SubjectiveAnswer 主观题答案结构
type SubjectiveAnswer struct {
	Index             int      `json:"index"`              // 答案索引
	Answer            string   `json:"answer"`             // 答案
	AlternativeAnswer []string `json:"alternative_answer"` // 备选答案
	Score             float64  `json:"score"`              // 分数
	GradingRule       string   `json:"grading_rule"`       // 评分规则
}

type StudentInfo struct {
	OfficialName         null.String `json:"OfficialName"`
	SerialNumber         null.Int    `json:"SerialNumber"`
	ExamineeID           null.Int    `json:"ExamineeID"`
	PracticeSubmissionID null.Int    `json:"PracticeSubmissionID"`
}

// QueryExamList 获取考试列表 管理员查询全部列表，教师查询与其有关的考试列表
func QueryExamList(ctx context.Context, req QueryMarkingListReq) (examList []Exam, rowCount int, err error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	err = cmn.Validate(req)
	if err != nil {
		err = fmt.Errorf("invalid query condition: %v", err)
		z.Error(err.Error())
		return nil, -1, err
	}

	teacherID := req.User.ID
	//z.Sugar().Infof("teacherID: %d", teacherID)

	examListQuery := `	
					-- 第一步：先选出你想要的 exam_info（比如最新的 10 个考试）
					WITH target_exam_infos AS ( 
						SELECT id 
						FROM t_exam_info 
						WHERE status != '12' AND status != '14' AND status != '16' %s -- 拼接对exam_info的where条件
						ORDER BY create_time DESC, id DESC
						LIMIT $1 OFFSET $2  
					) 
					SELECT 
						e.id AS exam_id, 
						e.name, 
						e.type, 
						es.id AS session_id, 
						es.session_num, 
						es.start_time, 
						es.end_time, 
						es.status, 
						es.mark_method, 
						es.mark_mode, 
						ep.name, 
						mi.mark_examinee_ids, 
						mi.status, 
						COALESCE(rc.respondent_count, 0) AS respondent_count, 
						COALESCE(umc.unmarked_student_count, 0) AS unmarked_student_count, 
						COALESCE(mc.marked_count, -1) AS marked_student_count 
					FROM
						 t_exam_session es 
					JOIN t_exam_info e ON e.id = es.exam_id AND e.status !='12' AND e.status != '14' AND e.status != '16' 
					JOIN target_exam_infos tei ON tei.id = e.id
					JOIN t_exam_paper ep 
						ON ep.exam_session_id = es.id 
						AND ep.status IS NOT NULL 
						AND ep.status != '04' 
					LEFT JOIN v_exam_respondent_count rc ON rc.exam_session_id = es.id 
					LEFT JOIN v_exam_unmarked_student_count umc ON umc.exam_session_id = es.id 
					LEFT JOIN v_exam_teacher_marked_count mc ON mc.exam_session_id = es.id 
					LEFT JOIN t_mark_info mi ON es.id = mi.exam_session_id AND mi.status != '04' 
					WHERE es.status IS NOT NULL AND es.status != '14' %s -- 动态插入where条件
					ORDER BY
						e.create_time DESC, e.id DESC, es.start_time ASC 
					`

	getExamCountQuery := `	SELECT COUNT(DISTINCT es.exam_id) AS total_exams
							FROM t_exam_session es
							JOIN t_exam_info ei ON es.exam_id = ei.id AND ei.status != '12' AND ei.status != '14' AND ei.status != '16' 
							JOIN t_exam_paper ep ON es.id = ep.exam_session_id AND ep.status != '04'
							LEFT JOIN t_mark_info mi ON es.id = mi.exam_session_id AND mi.status != '04'
							WHERE es.status IS NOT NULL
							  AND es.status != '14' %s -- 动态关联where条件`

	var (
		countArgs           []interface{}
		rowCountWhereClause []string
		countArgIdx         = 1
	)

	if req.ExamName != "" {
		rowCountWhereClause = append(rowCountWhereClause, fmt.Sprintf(" AND ei.name ILIKE $%d ", countArgIdx))
		countArgs = append(countArgs, "%"+req.ExamName+"%")
		countArgIdx++
	}

	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		rowCountWhereClause = append(rowCountWhereClause, fmt.Sprintf(" AND es.start_time >= $%d AND es.start_time <= $%d ", countArgIdx, countArgIdx+1))
		countArgs = append(countArgs, req.StartTime.UnixMilli(), req.EndTime.UnixMilli())
		countArgIdx += 2
	}

	// 普通批阅员
	if !req.User.IsAdmin {
		rowCountWhereClause = append(rowCountWhereClause, fmt.Sprintf(" AND mi.mark_teacher_id = $%d ", countArgIdx))
		countArgs = append(countArgs, teacherID)
		countArgIdx++
	}

	getExamCountQuery = fmt.Sprintf(getExamCountQuery, strings.Join(rowCountWhereClause, " "))

	pgxConn := cmn.GetPgxConn()
	err = pgxConn.QueryRow(ctx, getExamCountQuery, countArgs...).Scan(&rowCount)
	if err != nil || forceErr == "QueryExamList-pgxConn.QueryRow" {
		err = fmt.Errorf("getExamList count SQL error: %v", err)
		z.Error(err.Error())
		return nil, -1, err
	}

	// 条件动态拼接
	var (
		examWhereClause string
		whereClause     []string
		args            []interface{}
		argIdx          = 1
	)
	args = append(args, req.Limit, req.Offset)
	argIdx = 3

	if req.ExamName != "" {
		examWhereClause = fmt.Sprintf(" AND name ILIKE $%d ", argIdx)
		//whereClause = append(whereClause, fmt.Sprintf(" AND e.name ILIKE $%d ", argIdx))
		args = append(args, "%"+req.ExamName+"%")
		argIdx++
	}

	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		whereClause = append(whereClause, fmt.Sprintf(" AND es.start_time >= $%d AND es.start_time <= $%d ", argIdx, argIdx+1))
		args = append(args, req.StartTime.UnixMilli(), req.EndTime.UnixMilli())
		argIdx += 2
	}

	if !req.User.IsAdmin {
		// 普通批阅员，非管理员
		whereClause = append(whereClause, fmt.Sprintf(" AND mi.mark_teacher_id = $%d ", argIdx))
		args = append(args, teacherID)
		argIdx++
	}

	examListQuery = fmt.Sprintf(examListQuery, examWhereClause, strings.Join(whereClause, " "))

	rows, err := pgxConn.Query(ctx, examListQuery, args...)
	defer rows.Close()
	if err != nil || forceErr == "QueryExamList-pgxConn.Query" {
		err = fmt.Errorf("getExamList SQL error: %v", err)
		z.Error(err.Error())
		return nil, -1, err
	}

	examsMap := make(map[int64]*Exam)
	examsMapKeys := make([]int64, 0)

	for rows.Next() {
		var exam cmn.TExamInfo
		var session cmn.TExamSession
		//var unmarkedStudentCount cmn.NullInt
		var totalUnmarkedStudentCount null.Int
		var markedStudentCount null.Int
		var respondentCount null.Int
		var paperName null.String
		var markExamineeIds types.JSONText
		var markStatus null.String

		err = rows.Scan(
			&exam.ID,
			&exam.Name,
			&exam.Type,
			&session.ID,
			&session.SessionNum,
			&session.StartTime,
			&session.EndTime,
			&session.Status,
			&session.MarkMethod,
			&session.MarkMode,
			&paperName,
			&markExamineeIds,
			&markStatus,
			&respondentCount,
			&totalUnmarkedStudentCount,
			&markedStudentCount,
		)
		if err != nil || forceErr == "QueryExamList-rows.Scan" {
			err = fmt.Errorf("scan getExamList SQL result error: %v", err)
			z.Error(err.Error())
			return nil, -1, err
		}

		e, exists := examsMap[exam.ID.Int64]
		if !exists {
			e = &Exam{
				Id:   exam.ID.Int64,
				Name: exam.Name.String,
				Type: exam.Type.String,
			}
			examsMap[exam.ID.Int64] = e
			examsMapKeys = append(examsMapKeys, exam.ID.Int64)
		}

		// TODO
		unMarkedStudentCount := 0

		if markedStudentCount.Int64 == -1 {
			// 说明无主观题目需要批改
			unMarkedStudentCount = 0
		} else {
			unMarkedStudentCount = int(respondentCount.Int64 - markedStudentCount.Int64)
		}
		//if !req.User.IsAdmin && session.MarkMode.String == "04" {
		//	// 非管理员下的试卷分配模式
		//	var examineeIDs []int64
		//	if err = json.Unmarshal(markExamineeIds, &examineeIDs); err != nil {
		//		err = fmt.Errorf("unable to unmarshal markExamineeIDs: %v", err)
		//		z.Error(err.Error())
		//		return nil, -1, err
		//	}
		//
		//	unMarkedStudentCount = len(examineeIDs) - int(markedStudentCount.Int64)
		//} else if req.User.IsAdmin {
		//	unMarkedStudentCount = int(totalUnmarkedStudentCount.Int64)
		//} else {
		//	unMarkedStudentCount = int(respondentCount.Int64 - markedStudentCount.Int64)
		//}

		//m.l.Info("----------->..", session.ID.Int64, respondentCount.Int64, markedStudentCount.Int64, unMarkedStudentCount)

		if session.ID.Valid {
			s := &ExamSession{
				Id:                   session.ID.Int64,
				Name:                 fmt.Sprint(session.SessionNum.Int64),
				PaperName:            paperName.String,
				RespondentCount:      int(respondentCount.Int64),
				UnMarkedStudentCount: unMarkedStudentCount,
				Status:               session.Status.String,
				MarkStatus:           markStatus.String,
				MarkMethod:           session.MarkMethod,
				MarkMode:             session.MarkMode.String,
				StartTime:            session.StartTime.Int64,
				EndTime:              session.EndTime.Int64,
			}
			e.ExamSessions = append(e.ExamSessions, *s)
		}

		//examList = append(examList, exam)

	}

	for _, key := range examsMapKeys {
		examList = append(examList, *examsMap[key])
	}

	return
}

// QueryPracticeList 获取练习列表 管理员查询全部列表，教师查询与其有关的练习列表
func QueryPracticeList(ctx context.Context, req QueryMarkingListReq) (practices []Practice, rowCount int, err error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	err = cmn.Validate(req)
	if err != nil {
		err = fmt.Errorf("invalid query condition: %v", err)
		z.Error(err.Error())
		return nil, -1, err
	}

	teacherID := req.User.ID

	listQuery := `	SELECT p.id, 
						   p.name, 
						   p.correct_mode, 
						   p.type, 
						   COALESCE(rc.respondent_count, 0) AS respondent_count, 
						   COALESCE(usc.unmarked_count, 0)   AS unmarked_count 
					FROM t_practice p 
							 LEFT JOIN (SELECT ps.practice_id, 
										   COUNT(DISTINCT ps.student_id) AS respondent_count 
										FROM t_practice_submissions ps 
										WHERE ps.status != '04' 
										GROUP BY ps.practice_id) rc ON rc.practice_id = p.id 
							 LEFT JOIN v_practice_unmarked_student_cnt usc ON usc.practice_id = p.id 
							 LEFT JOIN t_mark_info mi ON mi.practice_id = p.id AND mi.status != '04'
					WHERE p.status != '04' %s -- 动态拼接
					ORDER BY p.create_time DESC, p.id DESC 
					LIMIT $1 OFFSET $2;`

	rowCountQuery := `	SELECT COUNT(*) AS practice_count
								FROM t_practice p
								LEFT JOIN t_mark_info mi ON mi.practice_id = p.id AND mi.status != '04' 
								WHERE p.status != '04' %s -- 动态拼接`

	var (
		countArgs           []interface{}
		rowCountWhereClause []string
		countArgIdx         = 1
	)

	if req.PracticeName != "" {
		rowCountWhereClause = append(rowCountWhereClause, fmt.Sprintf(" AND p.name ILIKE $%d ", countArgIdx))
		countArgs = append(countArgs, "%"+req.PracticeName+"%")
		countArgIdx++
	}

	// 普通批阅员
	if !req.User.IsAdmin {
		rowCountWhereClause = append(rowCountWhereClause, fmt.Sprintf(" AND mi.mark_teacher_id = $%d ", countArgIdx))
		countArgs = append(countArgs, teacherID)
		countArgIdx++
	}

	rowCountQuery = fmt.Sprintf(rowCountQuery, strings.Join(rowCountWhereClause, " "))

	pgxConn := cmn.GetPgxConn()
	err = pgxConn.QueryRow(ctx, rowCountQuery, countArgs...).Scan(&rowCount)
	if err != nil || forceErr == "QueryPracticeList-pgxConn.QueryRow" {
		err = fmt.Errorf("QueryPracticeList row count SQL error: %v", err)
		z.Error(err.Error())
		return nil, -1, err
	}

	// 条件动态拼接
	var (
		whereClause []string
		args        []interface{}
		argIdx      = 1
	)
	args = append(args, req.Limit, req.Offset)
	argIdx = 3

	if req.PracticeName != "" {
		whereClause = append(whereClause, fmt.Sprintf(" AND p.name ILIKE $%d ", argIdx))
		args = append(args, "%"+req.PracticeName+"%")
		argIdx++
	}

	if !req.User.IsAdmin {
		// 普通批阅员，非管理员
		whereClause = append(whereClause, fmt.Sprintf(" AND mi.mark_teacher_id = $%d ", argIdx))
		args = append(args, teacherID)
		argIdx++
	}

	listQuery = fmt.Sprintf(listQuery, strings.Join(whereClause, " "))

	rows, err := pgxConn.Query(ctx, listQuery, args...)
	defer rows.Close()
	if err != nil || forceErr == "QueryPracticeList-pgxConn.Query" {
		err = fmt.Errorf("QueryPracticeList SQL error: %v", err)
		z.Error(err.Error())
		return nil, -1, err
	}

	for rows.Next() {
		var practice Practice
		var row cmn.TPractice
		err = rows.Scan(&row.ID, &row.Name, &row.CorrectMode, &row.Type, &practice.RespondentCount, &practice.UnMarkedStudentCount)
		if err != nil || forceErr == "QueryPracticeList-rows.Scan" {
			err = fmt.Errorf("扫描解析练习数据失败:%v", err)
			return nil, -1, err
		}

		practice.MarkMode = row.CorrectMode.String
		practice.Type = row.Type.String
		practice.Name = row.Name.String
		practice.ID = row.ID.Int64

		practices = append(practices, practice)

	}

	return
}

// QueryMarkingResults 查询某教师的阅卷结果
func QueryMarkingResults(ctx context.Context, cond QueryCondition) (markingResults []*cmn.TMark, err error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	if cond.TeacherID <= 0 {
		err = fmt.Errorf("invalid params: 请求参数必须包含教师ID")
		z.Error(err.Error())
		return
	}

	if cond.ExamSessionID <= 0 && cond.PracticeID <= 0 {
		err = fmt.Errorf("invalid params: 请求参数必须包含考试场次ID或者练习ID中的一个")
		z.Error(err.Error())
		return
	}

	if cond.PracticeID > 0 && cond.ExamSessionID > 0 {
		err = fmt.Errorf("invalid params: 请求参数不能同时包含练习ID和考试场次ID")
		z.Error(err.Error())
		return
	}

	var (
		whereClause []string
		args        []interface{}
	)
	var argIndex = 2

	// 获取当前教师批阅的所有阅卷结果
	args = append(args, cond.TeacherID)

	if cond.ExamSessionID > 0 {
		// 获取考试场次下的阅卷结果
		whereClause = append(whereClause, fmt.Sprintf(" AND m.exam_session_id = $%d ", argIndex))
		args = append(args, cond.ExamSessionID)
		argIndex++

		if cond.ExamineeID > 0 {
			// 获取单个考生的阅卷结果
			whereClause = append(whereClause, fmt.Sprintf(" AND m.examinee_id = $%d ", argIndex))
			args = append(args, cond.ExamineeID)
			argIndex++
		}
	}

	if cond.PracticeID > 0 {
		// 获取练习下的阅卷结果
		whereClause = append(whereClause, fmt.Sprintf(" AND m.practice_id = $%d ", argIndex))
		args = append(args, cond.PracticeID)
		argIndex++

		if cond.PracticeSubmissionID > 0 {
			// 获取单个提交下的阅卷结果
			whereClause = append(whereClause, fmt.Sprintf(" AND m.practice_submission_id = $%d ", argIndex))
			args = append(args, cond.PracticeSubmissionID)
			argIndex++
		} else {
			// 在不指定提交id的情况下保证获取的是最新的提交
			whereClause = append(whereClause, " AND mp.submission_id IS NOT NULL ")
		}
	}

	getMarkingResultQuery := `	SELECT m.teacher_id, m.examinee_id, m.practice_submission_id, m.question_id, m.mark_details, m.score
								FROM t_mark m
								JOIN t_exam_paper_question q ON q.status != '04' AND q.id = m.question_id
								LEFT JOIN v_latest_pending_mark_practice mp ON mp.practice_id = m.practice_id AND mp.submission_id = m.practice_submission_id 
								WHERE m.teacher_id = $1 AND q.type IN ('06', '08') %s -- 动态拼接where条件
								ORDER BY m.examinee_id, m.practice_submission_id, q.order`

	pgxConn := cmn.GetPgxConn()

	getMarkingResultQuery = fmt.Sprintf(getMarkingResultQuery, strings.Join(whereClause, " "))

	rows, err := pgxConn.Query(ctx, getMarkingResultQuery, args...)
	defer rows.Close()
	if err != nil || forceErr == "QueryMarkingResults-pgxConn.Query" {
		err = fmt.Errorf("exec getMarkingResults SQL error: %v", err)
		z.Error(err.Error())
		return nil, err
	}

	for rows.Next() {
		var row cmn.TMark
		err = rows.Scan(&row.TeacherID, &row.ExamineeID, &row.PracticeSubmissionID, &row.QuestionID, &row.MarkDetails, &row.Score)
		if err != nil || forceErr == "QueryMarkingResults-rows.Scan" {
			err = fmt.Errorf("unable to scan row data: %v", err)
			z.Error(err.Error())
			return nil, err
		}

		if cond.ExamSessionID > 0 {
			row.ExamSessionID = null.IntFrom(cond.ExamSessionID)
		} else {
			row.PracticeID = null.IntFrom(cond.PracticeID)
		}

		markingResults = append(markingResults, &row)
	}
	return
}

func QueryMarkerInfo(ctx context.Context, cond QueryCondition) (markerInfo MarkerInfo, err error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	// 参数校验
	if cond.ExamSessionID == 0 && cond.PracticeID == 0 {
		err = fmt.Errorf("invalid params: either exam session id or practice id is required")
		z.Error(err.Error())
		return
	}

	var (
		query       string
		queryParams []interface{}
	)

	// 构建不同的SQL查询
	if cond.ExamSessionID != 0 {
		query = `SELECT mi.mark_teacher_id, mi.mark_count, 
                        mi.mark_question_groups, mi.mark_examinee_ids, 
                        es.mark_mode, es.mark_method, u.official_name
                 FROM t_mark_info mi
                 JOIN t_exam_session es ON es.id = mi.exam_session_id
                 JOIN t_user u ON u.id = mi.mark_teacher_id
                 WHERE mi.exam_session_id = $1 AND mi.status != '04'`
		queryParams = append(queryParams, cond.ExamSessionID)
	} else {
		query = `SELECT mi.mark_teacher_id, mi.mark_count, 
                        mi.mark_question_groups, mi.mark_examinee_ids, p.correct_mode 
                 FROM t_mark_info mi
                 JOIN t_practice p ON p.id = mi.practice_id
                 WHERE mi.practice_id = $1 AND mi.status != '04'`
		queryParams = append(queryParams, cond.PracticeID)
	}

	pgxConn := cmn.GetPgxConn()
	rows, err := pgxConn.Query(ctx, query, queryParams...)
	defer rows.Close()
	if err != nil || forceErr == "QueryMarkerInfo-pgxConn.Query" {
		err = fmt.Errorf("exec QueryMarkerInfo SQL error: %v", err)
		z.Error(err.Error())
		return
	}

	var (
		markMode   null.String
		markMethod null.String
		scanVars   []interface{}
	)

	for rows.Next() {
		var markInfo cmn.TMarkInfo
		var name null.String

		scanVars = []interface{}{&markInfo.MarkTeacherID, &markInfo.MarkCount,
			&markInfo.MarkQuestionGroups, &markInfo.MarkExamineeIds}

		if cond.ExamSessionID != 0 {
			scanVars = append(scanVars, &markMode, &markMethod, &name)
		} else {
			scanVars = append(scanVars, &markMode)
		}

		err = rows.Scan(scanVars...)
		if err != nil || forceErr == "QueryMarkerInfo-rows.Scan" {
			err = fmt.Errorf("unable to scan row data: %v", err)
			z.Error(err.Error())
			return
		}

		markerInfo.MarkInfos = append(markerInfo.MarkInfos, markInfo)
		if cond.ExamSessionID != 0 {
			markerInfo.MarkedStudentNames = append(markerInfo.MarkedStudentNames, name.String)
		}
	}

	markerInfo.MarkMode = markMode.String

	return markerInfo, nil
}

func QueryStudentAnswersByMarkMode(ctx context.Context, answerType string, cond QueryCondition, markerInfo MarkerInfo) (studentAnswers []*cmn.TVStudentAnswerQuestion, err error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	err = cmn.Validate(cond)
	if err != nil {
		err = fmt.Errorf("invalid query condition: %v", err)
		z.Error(err.Error())
		return
	}

	if answerType == "" {
		err = fmt.Errorf("invalid answer_type")
		z.Error(err.Error())
		return
	}

	query := `	SELECT exam_session_id, practice_id, question_id, "order", 
					type, question_score, answer, answer_score, actual_answers, 
					actual_options, examinee_id, practice_submission_id 
				FROM v_student_answer_question
				%s -- where条件
				ORDER BY "order" ASC
			`

	var args []interface{}
	var whereClause []string
	argIdx := 1
	whereClause = append(whereClause, " WHERE type IS NOT NULL ")

	if cond.PracticeID > 0 {
		whereClause = append(whereClause, fmt.Sprintf(" AND practice_id = $%d ", argIdx))
		args = append(args, cond.PracticeID)
		argIdx++
	} else if cond.ExamSessionID > 0 {
		whereClause = append(whereClause, fmt.Sprintf(" AND exam_session_id = $%d ", argIdx))
		args = append(args, cond.ExamSessionID)
		argIdx++
	}

	if answerType == "02" {
		// 单选、多选、判断题
		whereClause = append(whereClause, fmt.Sprintf(" AND type IN ('00', '02', '04') "))
	} else {
		// 填空、简答题
		whereClause = append(whereClause, fmt.Sprintf(" AND type IN ('06', '08') "))
	}

	switch markerInfo.MarkMode {
	case "04": // 试卷分配 查询单个考试考生的数据
		whereClause = append(whereClause, fmt.Sprintf(" AND examinee_id = ANY($%d) ", argIdx))
		argIdx++

		var markExamineeIDs []int64

		if cond.ExamineeID > 0 { // 获取单个考生的数据
			markExamineeIDs = append(markExamineeIDs, cond.ExamineeID)
		} else {
			err = json.Unmarshal(markerInfo.MarkInfos[0].MarkExamineeIds, &markExamineeIDs)
			if err != nil {
				err = fmt.Errorf("unmarshal mark examinee ids error: %v", err)
				z.Error(err.Error())
				return
			}
		}

		args = append(args, pq.Array(markExamineeIDs))
		break
	case "06":
		whereClause = append(whereClause, fmt.Sprintf(" AND question_id = ANY($%d) ", argIdx))
		argIdx++

		var questionIDs []int64
		err = json.Unmarshal(markerInfo.MarkInfos[0].QuestionIds, &questionIDs)
		if err != nil {
			err = fmt.Errorf("unable to unmarshal question_ids: %v", err)
			z.Error(err.Error())
			return
		}

		args = append(args, pq.Array(questionIDs))
		break
	case "12": // 查询单个练习学生的数据
		whereClause = append(whereClause, fmt.Sprintf(" AND practice_submission_id = $%d ", argIdx))
		argIdx++

		if cond.PracticeSubmissionID <= 0 {
			err = fmt.Errorf("invalid practice_submission_id")
			z.Error(err.Error())
			return
		}

		args = append(args, cond.PracticeSubmissionID)
		break
	default:
		break
	}

	query = fmt.Sprintf(query, strings.Join(whereClause, " "))

	pgxConn := cmn.GetPgxConn()
	rows, err := pgxConn.Query(ctx, query, args...)
	defer rows.Close()
	if err != nil || forceErr == "QueryStudentAnswersByMarkMode-pgxConn.Query" {
		err = fmt.Errorf("exec getStudentAnswersByMarkMode SQL error: %v", err)
		z.Error(err.Error())
		return
	}

	for rows.Next() {
		row := new(cmn.TVStudentAnswerQuestion)
		err = rows.Scan(
			&row.ExamSessionID,
			&row.PracticeID,
			&row.QuestionID,
			&row.Order,
			&row.Type,
			&row.QuestionScore,
			&row.Answer,
			&row.AnswerScore,
			&row.ActualAnswers,
			&row.ActualOptions,
			&row.ExamineeID,
			&row.PracticeSubmissionID,
		)
		if err != nil || forceErr == "rows.Scan" {
			err = fmt.Errorf("unable to scan row data: %v", err)
			z.Error(err.Error())
			return nil, err
		}

		studentAnswers = append(studentAnswers, row)

	}

	return
}

func QueryQuestionsByMarkMode(ctx context.Context, cond QueryCondition, markerInfo MarkerInfo) (questionSets []*QuestionSet, err error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	err = cmn.Validate(cond)
	if err != nil {
		err = fmt.Errorf("invalid query condition: %v", err)
		z.Error(err.Error())
		return
	}

	if cond.PracticeID <= 0 && cond.ExamSessionID <= 0 {
		err = fmt.Errorf("invalid params: 请求参数必须包含练习ID和考试ID中的一个")
		z.Error(err.Error())
		return
	}

	if cond.PracticeID > 0 && cond.ExamSessionID > 0 {
		err = fmt.Errorf("invalid params: 请求参数不能同时包含练习ID和考试ID")
		z.Error(err.Error())
		return
	}

	markMode := markerInfo.MarkMode

	if markMode == "" {
		err = fmt.Errorf("invalid mark mode")
		z.Error(err.Error())
		return
	}

	if !validateMarkMode(markMode) {
		err = fmt.Errorf("invalid mark mode")
		z.Error(err.Error())
		return
	}

	getQuestionsQuery := `	SELECT pg.id, pg.name, pg."order", q.id, q.score, q.type, q."order", q.group_id, q.content, q.options, 
								   q.answers, q.analysis, q.title 
							FROM t_exam_paper p
							JOIN t_exam_paper_group pg ON pg.exam_paper_id = p.id 
							JOIN t_exam_paper_question q ON q.group_id = pg.id
							WHERE p.status != '04' AND p.status IS NOT NULL 
							  AND q.status != '04' AND q.status IS NOT NULL 
							  AND q.type IN ('06', '08', '10') 
							  %s -- 动态拼接where条件`

	var whereClause []string
	var args []interface{}
	var argIndex = 1

	if cond.ExamSessionID > 0 {
		whereClause = append(whereClause, fmt.Sprintf(" AND p.exam_session_id = $%d ", argIndex))
		args = append(args, cond.ExamSessionID)
	} else {
		whereClause = append(whereClause, fmt.Sprintf(" AND p.practice_id = $%d ", argIndex))
		args = append(args, cond.PracticeID)
	}
	argIndex++

	if markMode == "06" {
		if markerInfo.MarkInfos == nil || len(markerInfo.MarkInfos) == 0 {
			err = fmt.Errorf("invalid markInfos")
			z.Error(err.Error())
			return
		}

		whereClause = append(whereClause, fmt.Sprintf(" AND q.id = ANY($%d)", argIndex))

		var questionIDs []int64
		err = markerInfo.MarkInfos[0].QuestionIds.Unmarshal(&questionIDs)
		if err != nil {
			err = fmt.Errorf("unable to unmarshal question ids: %v", err)
			z.Error(err.Error())
			return nil, err
		}

		args = append(args, pq.Array(questionIDs))
		argIndex++
	}

	getQuestionsQuery = fmt.Sprintf(getQuestionsQuery, strings.Join(whereClause, " "))

	pgxConn := cmn.GetPgxConn()
	rows, err := pgxConn.Query(ctx, getQuestionsQuery, args...)
	defer rows.Close()
	if err != nil || forceErr == "QueryQuestionsByMarkMode-pgxConn.Query" {
		err = fmt.Errorf("exec getQuestionsQuery SQL error: %v", err)
		z.Error(err.Error())
		return
	}

	questionSetsMap := make(map[int64]*QuestionSet)

	for rows.Next() {
		var question cmn.TExamPaperQuestion
		var questionGroup cmn.TExamPaperGroup
		err = rows.Scan(&questionGroup.ID, &questionGroup.Name, &questionGroup.Order, &question.ID, &question.Score, &question.Type, &question.Order, &question.GroupID, &question.Content, &question.Options, &question.Answers, &question.Analysis, &question.Title)
		if err != nil || forceErr == "QueryQuestionsByMarkMode-rows.Scan" {
			err = fmt.Errorf("unable to scan row data: %v", err)
			z.Error(err.Error())
			return
		}

		set, exist := questionSetsMap[question.GroupID.Int64]
		if !exist {
			set = &QuestionSet{
				TExamPaperGroup: questionGroup,
				Score:           0.0,
				Questions:       []*cmn.TExamPaperQuestion{},
			}
			questionSetsMap[question.GroupID.Int64] = set
		}

		var standardAnswers []*SubjectiveAnswer
		//standardAnswers, _, err = ConvertRawStandardAnswerData(question.Answers, question.Type.String)
		err = json.Unmarshal(question.Answers, &standardAnswers)
		if err != nil {
			err = fmt.Errorf("unable to convert raw standard answer data: %v", err)
			z.Error(err.Error())
			standardAnswers = []*SubjectiveAnswer{}
		}

		set.Score += question.Score.Float64
		set.Questions = append(set.Questions, &question)

	}

	for _, result := range questionSetsMap {
		questionSets = append(questionSets, result)
	}

	return

}

func QueryExamineeInfo(ctx context.Context, cond QueryCondition) (examineeInfos []*cmn.TVExamineeInfo, err error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	err = cmn.Validate(cond)
	if err != nil {
		err = fmt.Errorf("invalid query condition: %v", err)
		z.Error(err.Error())
		return
	}

	query := `	SELECT official_name, id, serial_number
				FROM v_examinee_info
				WHERE exam_session_id = $1 AND examinee_status = '10' %s -- 动态拼接where条件
				ORDER BY serial_number`

	var args []interface{}
	args = append(args, cond.ExamSessionID)

	if cond.ExamineeID > 0 {
		query = fmt.Sprintf(query, " AND id = $2 ")
		args = append(args, cond.ExamineeID)
	} else {
		query = fmt.Sprintf(query, "")
	}

	pgxConn := cmn.GetPgxConn()
	rows, err := pgxConn.Query(ctx, query, args...)
	defer rows.Close()
	if err != nil || forceErr == "QueryExamineeInfo-pgxConn.Query" {
		err = fmt.Errorf("exec query examinee info SQL error: %v", err)
		z.Error(err.Error())
		return
	}

	for rows.Next() {
		var examineeInfo cmn.TVExamineeInfo
		err = rows.Scan(&examineeInfo.OfficialName, &examineeInfo.ID, &examineeInfo.SerialNumber)
		if err != nil || forceErr == "QueryExamineeInfo-rows.Scan" {
			err = fmt.Errorf("unable to scan row data: %v", err)
			z.Error(err.Error())
			return
		}
		examineeInfos = append(examineeInfos, &examineeInfo)
	}

	return
}

func QueryStudentsInfo(ctx context.Context, cond QueryCondition) (studentInfos []*StudentInfo, err error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	if cond.TeacherID <= 0 {
		err = fmt.Errorf("invalid params: 请求参数必须包含教师ID")
		z.Error(err.Error())
		return
	}

	if cond.PracticeID <= 0 && cond.ExamSessionID <= 0 {
		err = fmt.Errorf("invalid params: 请求参数必须包含考试场次ID或者练习ID中的一个")
		z.Error(err.Error())
		return
	}

	if cond.PracticeID > 0 && cond.ExamSessionID > 0 {
		err = fmt.Errorf("invalid params: 请求参数不能同时包含练习ID和考试场次ID")
		z.Error(err.Error())
		return
	}

	var query string
	var args []interface{}
	var whereClause = ""
	if cond.ExamSessionID > 0 {
		query = `	SELECT official_name, id, serial_number
					FROM v_examinee_info
					WHERE exam_session_id = $1 AND examinee_status = '10' %s -- 动态拼接where条件
					ORDER BY serial_number`
		args = append(args, cond.ExamSessionID)
		if cond.ExamineeID > 0 {
			whereClause = " AND id = $2 "
			args = append(args, cond.ExamineeID)
		}
	} else {
		// 获取需要批改的练习人员信息
		query = `	SELECT u.official_name, mp.submission_id 
					FROM v_latest_pending_mark_practice mp 
					JOIN t_user u ON mp.student_id = u.id 
					WHERE mp.practice_id = $1 %s -- 动态插入s
					ORDER BY mp.submission_id`
		args = append(args, cond.PracticeID)
		if cond.PracticeSubmissionID > 0 {
			whereClause = " AND mp.submission_id = $2 "
			args = append(args, cond.PracticeSubmissionID)
		}
	}

	query = fmt.Sprintf(query, whereClause)

	pgxConn := cmn.GetPgxConn()
	rows, err := pgxConn.Query(ctx, query, args...)
	defer rows.Close()
	if err != nil || forceErr == "QueryStudentInfo-pgxConn.Query" {
		err = fmt.Errorf("exec query student info SQL error: %v", err)
		z.Error(err.Error())
		return
	}

	for rows.Next() {
		var info StudentInfo
		var scanVars []interface{}
		if cond.ExamSessionID > 0 {
			scanVars = []interface{}{&info.OfficialName, &info.ExamineeID, &info.SerialNumber}
		} else {
			scanVars = []interface{}{&info.OfficialName, &info.PracticeSubmissionID}
		}

		err = rows.Scan(scanVars...)
		if err != nil || forceErr == "QueryStudentsInfo-rows.Scan" {
			err = fmt.Errorf("unable to scan row data: %v", err)
			z.Error(err.Error())
			return
		}
		studentInfos = append(studentInfos, &info)
	}

	return
}

func UpdateMarkerInfoState(ctx context.Context, tx *pgx.Tx, teacherID int64, ids []int64, mode string) (targetIDs []int64, err error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	if teacherID <= 0 {
		err = fmt.Errorf("invalid teacher ID param")
		z.Error(err.Error())
		return
	}

	if ids == nil || len(ids) == 0 {
		err = fmt.Errorf("no exam_session_ids or practice_ids to update")
		z.Error(err.Error())
		return
	}

	var whereClause []string

	updateQuery := `UPDATE t_mark_info 
					SET
						status = $2,
						updated_by = $3, 
						update_time = $4
					WHERE %s 
					RETURNING id`

	if mode == "00" {
		whereClause = append(whereClause, "exam_session_id = ANY($1)")
	} else if mode == "02" {
		whereClause = append(whereClause, "practice_id = ANY($1)")
	} else {
		err = fmt.Errorf("invalid update mode")
		z.Error(err.Error())
		return
	}

	updateQuery = fmt.Sprintf(updateQuery, strings.Join(whereClause, " "))

	rows, err := (*tx).Query(ctx, updateQuery, pq.Array(ids), "04", teacherID, time.Now().UnixMilli())
	defer rows.Close()
	if err != nil || forceErr == "tx.Query" {
		err = fmt.Errorf("exec update query error: %v", err)
		z.Error(err.Error())
		return
	}

	for rows.Next() {
		var id null.Int
		err = rows.Scan(&id)
		if err != nil || forceErr == "rows.Scan" {
			err = fmt.Errorf("unable to scan row data: %v", err)
			z.Error(err.Error())
			return
		}
		targetIDs = append(targetIDs, id.Int64)
	}

	return
}

func InsertOrUpdateMarkingResults(ctx context.Context, markingResults []*cmn.TMark) (targetIDs []int64, err error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string) // 用于强制执行错误处理代码
	// 不存在则插入，否则更新，需要索引
	upsertMarkQuery := `
						INSERT INTO t_mark 
							(examinee_id, question_id, teacher_id, exam_session_id, mark_details, score, create_time, creator, status, updated_by, update_time, practice_submission_id, practice_id)
						VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
						ON CONFLICT %s -- 约束条件 
						DO UPDATE SET
							mark_details = EXCLUDED.mark_details,
							score = EXCLUDED.score,
							update_time = EXCLUDED.update_time,
							updated_by = EXCLUDED.updated_by
						RETURNING id
						`

	upsertExamMarkQuery := fmt.Sprintf(upsertMarkQuery, `(examinee_id, question_id, teacher_id, exam_session_id)`)
	upsertPracticeMarkQuery := fmt.Sprintf(upsertMarkQuery, `(question_id, teacher_id, practice_id, practice_submission_id)`)

	pgxConn := cmn.GetPgxConn()

	tx, err := pgxConn.Begin(ctx)
	if err != nil || forceErr == "pgxConn.Begin" {
		err = fmt.Errorf("begin transaction error: %w", err)
		z.Error(err.Error())
		return
	}

	defer func() {
		if err != nil || forceErr == "tx.Rollback" {
			err_ := tx.Rollback(ctx)
			if err_ != nil {
				err_ = fmt.Errorf("rollback tx error: %w", err_)
				z.Error(err_.Error())
				return
			}
			return
		}
	}()

	for _, mark := range markingResults {
		if mark.Status.String == "" {
			mark.Status = null.StringFrom("00")
		}

		if mark.TeacherID.Int64 == 0 {
			q := cmn.GetCtxValue(ctx)
			mark.TeacherID = null.IntFrom(q.SysUser.ID.Int64)
		}

		if mark.Creator.Int64 <= 0 {
			mark.Creator = null.IntFrom(mark.TeacherID.Int64)
		}

		if mark.QuestionID.Int64 <= 0 {
			err = fmt.Errorf("questionID is required for mark")
			z.Error(err.Error())
			return
		}

		validExam := mark.ExamSessionID.Int64 > 0 && mark.ExamineeID.Int64 > 0 &&
			mark.PracticeID.Int64 <= 0 && mark.PracticeSubmissionID.Int64 <= 0

		validPractice := mark.PracticeID.Int64 > 0 && mark.PracticeSubmissionID.Int64 > 0 &&
			mark.ExamSessionID.Int64 <= 0 && mark.ExamineeID.Int64 <= 0

		if !(validExam || validPractice) {
			err = fmt.Errorf("invalid insert params: must have either (exam session id and examinee id) both > 0 with others <= 0, or (practice id and practice submission id) both > 0 with others <= 0")
			z.Error(err.Error())
			return
		}

		var query string
		if validExam {
			query = upsertExamMarkQuery
		} else {
			query = upsertPracticeMarkQuery
		}

		var id null.Int

		err = tx.QueryRow(ctx, query, mark.ExamineeID, mark.QuestionID, mark.TeacherID, mark.ExamSessionID, mark.MarkDetails, mark.Score, time.Now().UnixMilli(), mark.Creator, mark.Status, mark.UpdatedBy, mark.UpdateTime, mark.PracticeSubmissionID, mark.PracticeID).Scan(&id)
		if err != nil || forceErr == "tx.QueryRow" {
			err = fmt.Errorf("exec upsertMark query error: %v", err)
			z.Error(err.Error())
			return
		}
		targetIDs = append(targetIDs, id.Int64)
	}

	err = tx.Commit(ctx)
	if err != nil || forceErr == "tx.Commit" {
		err = fmt.Errorf("commit transaction error: %w", err)
		z.Error(err.Error())
		return
	}

	return
}

func updateStudentAnswerScore(ctx context.Context, markingResults []*cmn.TMark, cond QueryCondition) (targetIDs []int64, err error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	if len(markingResults) == 0 {
		err = fmt.Errorf("no marking results for update")
		z.Error(err.Error())
		return
	}

	updateQuery := `UPDATE t_student_answers
					SET answer_score = $1,
						update_time = $2,
						updated_by = $3
					WHERE question_id = $4 %s -- 动态拼接where
					RETURNING id`

	var argIdx int
	var whereClause []string

	argIdx = 5
	if cond.ExamSessionID > 0 {
		whereClause = append(whereClause, fmt.Sprintf(" AND examinee_id = $%d", argIdx))
		argIdx += 1
	} else if cond.PracticeID > 0 {
		whereClause = append(whereClause, fmt.Sprintf(" AND practice_submission_id = $%d", argIdx))
		argIdx += 1
	} else {
		err = fmt.Errorf("invalid params: exam_session_id or practice_id is required")
		z.Error(err.Error())
		return
	}

	updateQuery = fmt.Sprintf(updateQuery, strings.Join(whereClause, " "))

	pgxConn := cmn.GetPgxConn()
	tx, err := pgxConn.Begin(ctx)
	if err != nil || forceErr == "updateStudentAnswerScore-pgxConn.Begin" {
		err = fmt.Errorf("begin transaction error: %v", err)
		z.Error(err.Error())
		return
	}

	defer func() {
		if err != nil || forceErr == "tx.Rollback" {
			err_ := tx.Rollback(ctx)
			if err_ != nil {
				z.Error(err_.Error())
				return
			}
		}
	}()

	for _, mark := range markingResults {
		if mark.QuestionID.Int64 <= 0 {
			err = fmt.Errorf("question_id is required in mark")
			z.Error(err.Error())
			return
		}

		if mark.TeacherID.Int64 <= 0 {
			err = fmt.Errorf("teacher id is required in mark")
			z.Error(err.Error())
			return
		}

		var targetID int64
		args := []interface{}{
			mark.Score.Float64,
			time.Now().UnixMilli(),
			mark.TeacherID.Int64,
			mark.QuestionID.Int64,
		}

		if cond.ExamSessionID > 0 {
			if mark.ExamineeID.Int64 <= 0 {
				err = fmt.Errorf("examinee_id is required where exam_session_id is specified")
				z.Error(err.Error())
				return
			}
			args = append(args, mark.ExamineeID.Int64)
		} else if cond.PracticeID > 0 {
			if mark.PracticeSubmissionID.Int64 <= 0 {
				err = fmt.Errorf("practice_submission_id is required where practice_id is specified")
				z.Error(err.Error())
				return
			}
			args = append(args, mark.PracticeSubmissionID.Int64)
		}

		//z.Sugar().Infof("--> %v", args)

		err = tx.QueryRow(ctx, updateQuery, args...).Scan(&targetID)
		if err != nil || forceErr == "tx.QueryRow" {
			err = fmt.Errorf("exec updateStudentAnswerScore sql error: %v", err)
			z.Error(err.Error())
			return
		}

		targetIDs = append(targetIDs, targetID)
	}

	err = tx.Commit(ctx)
	if err != nil || forceErr == "tx.Commit" {
		err = fmt.Errorf("commit tx error: %v", err)
		z.Error(err.Error())
		return
	}
	return
}

func updateExamSessionOrPracticeSubmissionState(ctx context.Context, tx *pgx.Tx, teacherID int64, examSessionIDs []int64, practiceSubmissionIDs []int64, status string) (targetIDs []int64, err error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	if teacherID <= 0 {
		err = fmt.Errorf("invalid params: teacher_id is required")
		z.Error(err.Error())
		return
	}

	if len(examSessionIDs) == 0 && len(practiceSubmissionIDs) == 0 {
		err = fmt.Errorf("invalid params: exam_session_ids or practice_submission_ids is required")
		z.Error(err.Error())
		return
	}

	if len(examSessionIDs) > 0 && len(practiceSubmissionIDs) > 0 {
		err = fmt.Errorf("invalid params: exam_session_ids and practice_submission_ids cannot be both specified")
		z.Error(err.Error())
		return
	}

	if status == "" {
		err = fmt.Errorf("invalid params: status is required")
		z.Error(err.Error())
		return
	}

	updateQuery := `UPDATE %s -- 拼接表名 
					SET
						status = $2,
						updated_by = $3, 
						update_time = $4
					WHERE id = ANY($1)
					RETURNING id`

	var ids []int64
	if len(examSessionIDs) > 0 {
		updateQuery = fmt.Sprintf(updateQuery, "t_exam_session")
		ids = examSessionIDs
	} else {
		updateQuery = fmt.Sprintf(updateQuery, "t_practice_submissions")
		ids = practiceSubmissionIDs
	}

	rows, err := (*tx).Query(ctx, updateQuery, pq.Array(ids), status, teacherID, time.Now().UnixMilli())
	if err != nil || forceErr == "updateExamSessionOrPracticeSubmissionState-tx.Query" {
		err = fmt.Errorf("exec updateExamSessionState sql error: %v", err)
		z.Error(err.Error())
		return
	}

	defer rows.Close()

	for rows.Next() {
		var id int64
		err = rows.Scan(&id)
		if err != nil || forceErr == "rows.Scan" {
			err = fmt.Errorf("unable to scan row data: %v", err)
			z.Error(err.Error())
			return
		}
		targetIDs = append(targetIDs, id)
	}

	return
}
