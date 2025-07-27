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

type QueryCondition struct {
	TeacherID     int64 `json:"teacher_id" validate:"required,gt=0"`
	ExamSessionID int64 `json:"exam_session_id" validate:"gte=0"`
	ExamineeID    int64 `json:"examinee_id" validate:"gte=0"`
	PracticeID    int64 `json:"practice_id" validate:"gte=0"`
}

type MarkerInfo struct {
	MarkerID           int64           `json:"marker_id"`
	MarkInfos          []cmn.TMarkInfo `json:"mark_infos"`
	MarkMode           string          `json:"mark_mode"`
	MarkMethod         string          `json:"mark_method"`
	MarkedStudentNames []string        `json:"marked_student_names"`
}

// Answer 作答
type Answer struct {
	Details        []*AnswerDetails `json:"details"`
	QuestionNumber int              `json:"question_number"`
	QuestionID     int64            `json:"question_id"`
	QuestionScore  float64          `json:"question_score"`
	AnswersJson    types.JSONText   `json:"answers_json"`
	ActualOptions  []string         `json:"actual_options"`
	ActualAnswers  []string         `json:"actual_answers"`
}

type RawAnswer struct {
	QuestionID int      `json:"question_id"`
	SubAnswers []string `json:"answer"`
	Answer     []struct {
		Answer string `json:"answer"`
	} `json:"sub_answers"`
}

type AnswerDetails struct {
	Index   int    `json:"index"`
	Type    string `json:"type"` // 回答的类型 如 链接、文本
	Content string `json:"content"`
}

type QuestionDetails struct {
	Content string  `json:"content"`
	Score   float64 `json:"score"`
	Index   int     `json:"index"`
}

type QuestionSet struct {
	ID        int64       `json:"id"`        // 题组id
	Name      string      `json:"name"`      // 题组名称
	Order     int         `json:"order"`     // 题组序号
	Questions []*Question `json:"questions"` // 题目
	Score     float64     `json:"score"`
}

// Question 题目
type Question struct {
	ID               int64              `json:"id"`
	Analyze          string             `json:"analyze"`
	MarkRole         string             `json:"mark_role"`
	StandardAnswers  []*StandardAnswer  `json:"standard_answers"`
	ObjectiveAnswers []string           `json:"objective_answers"`
	Details          []*QuestionDetails `json:"details"`
	OrderNum         int                `json:"order_num"`
	Type             string             `json:"type"` // 00:单选题  02:多选题 04:判断题 06:填空题 08:简答题 10:编程题
	Score            float64            `json:"score"`
}

// 简答题、填空题答案
type StandardAnswer struct {
	// 追加答案
	AlternativeAnswer []string `json:"alternative_answer"`
	// 答案
	Answer string `json:"answer"`
	// 批改规则/提示词
	GradingRule string `json:"grading_rule"`
	// 答案索引
	Index int64 `json:"index"`
	// 该答案的分数
	Score int64 `json:"score"`
}

type Student struct {
	Name                string  `json:"name"`
	Nickname            string  `json:"nickname"`
	Number              int     `json:"number"` // 考生序号
	Phone               string  `json:"phone"`
	Score               float64 `json:"score"`
	Status              string  `json:"status"`
	StudentID           int64   `json:"student_id"`
	ExamineeID          int64   `json:"examinee_id"`
	IdType              string  `json:"id_type"`
	IdNumber            string  `json:"id_number"`
	ExamAdmissionNumber string  `json:"exam_admission_number"`
}

// QueryExamList 获取考试列表 管理员查询全部列表，教师查询与其有关的考试列表
func QueryExamList(ctx context.Context, req GetExamListReq) (examList []Exam, rowCount int, err error) {
	err = cmn.Validate(req)
	if err != nil {
		err = fmt.Errorf("invalid query condition: %s", err.Error())
		z.Error(err.Error())
		return nil, -1, err
	}

	teacherID := req.User.ID
	z.Sugar().Infof("teacherID: %d", teacherID)

	examListQuery := `	SELECT
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
						COALESCE(mc.marked_count, 0) AS marked_student_count 
					FROM
						 t_exam_session es 
					JOIN t_exam_info e ON e.id = es.exam_id AND e.status !='12'
					JOIN t_exam_paper ep 
						ON ep.exam_session_id = es.id 
						AND ep.status IS NOT NULL 
						AND ep.status != '04' 
					LEFT JOIN v_exam_respondent_count rc ON rc.exam_session_id = es.id 
					LEFT JOIN v_exam_unmarked_student_count umc ON umc.exam_session_id = es.id 
					LEFT JOIN v_exam_teacher_marked_count mc ON mc.exam_session_id = es.id 
					LEFT JOIN t_mark_info mi ON es.id = mi.exam_session_id AND mi.status != '04' 
					WHERE es.status IS NOT NULL AND es.status != '06' %s -- 动态插入where条件
					ORDER BY
						e.create_time DESC, e.id DESC 
					LIMIT $1 OFFSET $2;`

	getExamCountQuery := `	SELECT COUNT(DISTINCT es.exam_id) AS total_exams
							FROM t_exam_session es
							JOIN t_exam_info ei ON es.exam_id = ei.id AND ei.status != '12'
							JOIN t_exam_paper ep ON es.id = ep.exam_session_id AND ep.status != '04'
							LEFT JOIN t_mark_info mi ON es.id = mi.exam_session_id AND mi.status != '04'
							WHERE es.status IS NOT NULL
							  AND es.status != '06' %s -- 动态关联where条件`

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
	if err != nil {
		err = fmt.Errorf("getExamList count SQL error: %s", err.Error())
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

	if req.ExamName != "" {
		whereClause = append(whereClause, fmt.Sprintf(" AND e.name ILIKE $%d ", argIdx))
		args = append(args, "%"+req.ExamName+"%")
		argIdx++
	}

	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		whereClause = append(whereClause, fmt.Sprintf(" AND es.start_time >= $%d AND es.start_time <= $%d ", argIdx, argIdx+1))
		args = append(args, req.StartTime.UnixMilli(), req.EndTime.UnixMilli())
		argIdx += 2
	}

	if !req.User.IsAdmin {
		whereClause = append(whereClause, fmt.Sprintf(" AND mi.mark_teacher_id = $%d ", argIdx))
		args = append(args, teacherID)
		argIdx++
	}

	examListQuery = fmt.Sprintf(examListQuery, strings.Join(whereClause, " "))

	rows, err := pgxConn.Query(ctx, examListQuery, args...)
	if err != nil {
		err = fmt.Errorf("getExamList SQL error: %s", err.Error())
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
		if err != nil {
			err = fmt.Errorf("scan getExamList SQL result error: %s", err.Error())
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
		if !req.User.IsAdmin && session.MarkMode.String == "04" {
			// 非管理员下的试卷分配模式
			var examineeIDs []int64
			if err = json.Unmarshal(markExamineeIds, &examineeIDs); err != nil {
				//m.l.Errorf("Unable to unmarshal markExamineeIDs:%s", err)
				err = fmt.Errorf("unable to unmarshal markExamineeIDs: %s", err.Error())
				z.Error(err.Error())
				return nil, -1, err
			}

			unMarkedStudentCount = len(examineeIDs) - int(markedStudentCount.Int64)
		} else if req.User.IsAdmin {
			unMarkedStudentCount = int(totalUnmarkedStudentCount.Int64)
		} else {
			unMarkedStudentCount = int(respondentCount.Int64 - markedStudentCount.Int64)
		}

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

	if err = rows.Err(); err != nil {
		err = fmt.Errorf("error while scanning rows: %s", err.Error())
		z.Error(err.Error())
		return nil, -1, err
	}

	for _, key := range examsMapKeys {
		examList = append(examList, *examsMap[key])
	}

	return
}

func QueryMarkingResults(ctx context.Context, cond QueryCondition) (markingResults []*cmn.TMark, err error) {
	err = cmn.Validate(cond)
	if err != nil {
		err = fmt.Errorf("invalid query condition: %s", err.Error())
		z.Error(err.Error())
		return
	}

	if cond.ExamSessionID <= 0 {
		err = fmt.Errorf("invalid exam_session_id")
		z.Error(err.Error())
		return
	}

	var (
		whereClause []string
		args        []interface{}
	)
	var argIndex = 2

	args = append(args, cond.ExamSessionID)
	if cond.ExamineeID > 0 {
		// 获取单个考生的阅卷结果
		whereClause = append(whereClause, fmt.Sprintf(" AND m.examinee_id = $%d ", argIndex))
		args = append(args, cond.ExamineeID)
		argIndex++
	} else {
		// 获取当前教师批阅的所有阅卷结果
		whereClause = append(whereClause, fmt.Sprintf(" AND m.teacher_id = $%d ", argIndex))
		args = append(args, cond.TeacherID)
		argIndex++
	}

	getMarkingResultQuery := `	SELECT m.teacher_id, m.examinee_id, m.question_id, m.mark_details, m.score
								FROM t_mark m
								JOIN t_exam_paper_question q ON q.status != '04' AND q.id = m.question_id
								WHERE m.exam_session_id = $1 AND q.type IN ('06', '08') %s -- 动态拼接where条件
								ORDER BY m.examinee_id, q.order`
	pgxConn := cmn.GetPgxConn()

	getMarkingResultQuery = fmt.Sprintf(getMarkingResultQuery, strings.Join(whereClause, " "))

	rows, err := pgxConn.Query(ctx, getMarkingResultQuery, args...)
	if err != nil {
		err = fmt.Errorf("exec getMarkingResults SQL error: %s", err.Error())
		z.Error(err.Error())
		return nil, err
	}

	for rows.Next() {
		var row cmn.TMark
		err = rows.Scan(&row.TeacherID, &row.ExamineeID, &row.QuestionID, &row.MarkDetails, &row.Score)
		if err != nil {
			err = fmt.Errorf("scan rows error: %s", err.Error())
			z.Error(err.Error())
			return nil, err
		}

		markingResults = append(markingResults, &row)
	}

	if rows.Err() != nil {
		err = fmt.Errorf("error while scanning rows: %s", rows.Err())
		z.Error(err.Error())
		return nil, err
	}

	return
}

func QueryExamMarkerInfo(ctx context.Context, cond QueryCondition) (markerInfo MarkerInfo, err error) {
	if cond.ExamSessionID == 0 {
		err = fmt.Errorf("exam session id is required")
		z.Error(err.Error())
		return
	}

	getExamMarkerInfoSQL := `	SELECT mi.mark_teacher_id, mi.mark_count, 
	       							mi.mark_question_groups, mi.mark_examinee_ids, es.mark_mode, es.mark_method, u.official_name
								FROM t_mark_info mi
								JOIN t_exam_session es ON es.id = mi.exam_session_id
								JOIN t_user u ON u.id = mi.mark_teacher_id
								WHERE mi.exam_session_id = $1 AND mi.status != '04'`

	pgxConn := cmn.GetPgxConn()
	rows, err := pgxConn.Query(ctx, getExamMarkerInfoSQL, cond.ExamSessionID)
	if err != nil {
		err = fmt.Errorf("exec getExamMarkerInfo SQL error: %s", err.Error())
		z.Error(err.Error())
		return
	}

	var markMode null.String
	var markMethod null.String
	for rows.Next() {
		var markInfo cmn.TMarkInfo
		var name null.String

		err = rows.Scan(&markInfo.MarkTeacherID, &markInfo.MarkCount, &markInfo.MarkQuestionGroups, &markInfo.MarkExamineeIds, &markMode, &markMethod, &name)
		if err != nil {
			err = fmt.Errorf("unable to scan row data:%s", err)
			z.Error(err.Error())
			return
		}

		markerInfo.MarkInfos = append(markerInfo.MarkInfos, markInfo)
		markerInfo.MarkedStudentNames = append(markerInfo.MarkedStudentNames, name.String)
	}

	markerInfo.MarkMethod = markMethod.String
	markerInfo.MarkMode = markMode.String

	return
}

func QueryStudentAnswersByMarkMode(ctx context.Context, answerType string, cond QueryCondition, markerInfo MarkerInfo) (studentAnswers []*cmn.TVStudentAnswerQuestion, err error) {
	err = cmn.Validate(cond)
	if err != nil {
		err = fmt.Errorf("invalid query condition: %s", err.Error())
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
	case "04": // 试卷分配 以及批改单个考生
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

		var questionIDs []int64
		err = json.Unmarshal(markerInfo.MarkInfos[0].QuestionIds, &questionIDs)
		if err != nil {
			err = fmt.Errorf("unable to unmarshal question_ids: %v", err)
			z.Error(err.Error())
			return
		}

		args = append(args, pq.Array(questionIDs))
		break
	default:
		break
	}

	query = fmt.Sprintf(query, strings.Join(whereClause, " "))

	pgxConn := cmn.GetPgxConn()
	rows, err := pgxConn.Query(ctx, query, args...)
	if err != nil {
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
		if err != nil {
			err = fmt.Errorf("unable to scan row data:%s", err)
			z.Error(err.Error())
			return nil, err
		}

		studentAnswers = append(studentAnswers, row)

	}

	if rows.Err() != nil {
		err = fmt.Errorf("error while scanning rows: %v", rows.Err())
		z.Error(err.Error())
		return
	}

	return

}

func QueryExamQuestionsByMarkMode(ctx context.Context, cond QueryCondition, markerInfo MarkerInfo) (questionSets []*QuestionSet, err error) {
	err = cmn.Validate(cond)
	if err != nil {
		err = fmt.Errorf("invalid query condition: %s", err.Error())
		z.Error(err.Error())
		return
	}

	markMode := markerInfo.MarkMode
	markMethod := markerInfo.MarkMethod

	if markMode == "" || markMethod == "" {
		err = fmt.Errorf("invalid mark mode or mark method")
		z.Error(err.Error())
		return
	}

	if !validateMarkMode(markMode) {
		err = fmt.Errorf("invalid mark mode")
		z.Error(err.Error())
		return
	}

	getQuestionsQuery := `	SELECT g.id, g.name, g."order", q.id, q.score, q.type, q."order", q.group_id, q.content, q.options, 
								   q.answers, q.analysis, q.title, 
								   q.input, q.output, q.example, q.repo, q.commit_id 
							FROM t_exam_paper p
							JOIN t_exam_paper_question q ON p.id = q.exam_paper_id
							JOIN t_paper_group g ON q.group_id = g.id 
							WHERE p.status != '04' AND p.status IS NOT NULL 
							  AND p.exam_session_id = $1 
							  AND q.status != '04' AND q.status IS NOT NULL 
							  AND q.type IN ('06', '08', '10') 
							  %s -- 动态拼接where条件`

	var whereClause []string
	var args []interface{}
	var argIndex int

	args = append(args, cond.ExamSessionID)
	argIndex = 2

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
			err = fmt.Errorf("unable to unmarshal question groups: %v", err)
			z.Error(err.Error())
			return nil, err
		}

		args = append(args, pq.Array(questionIDs))
		argIndex++
	}

	getQuestionsQuery = fmt.Sprintf(getQuestionsQuery, strings.Join(whereClause, " "))

	pgxConn := cmn.GetPgxConn()
	rows, err := pgxConn.Query(ctx, getQuestionsQuery, args...)
	if err != nil {
		err = fmt.Errorf("exec getQuestionsQuery SQL error: %s", err.Error())
		z.Error(err.Error())
		return
	}

	var questionSetsMap map[int64]*QuestionSet

	for rows.Next() {
		var question cmn.TExamPaperQuestion
		var questionGroup cmn.TPaperGroup
		err = rows.Scan(&questionGroup.ID, &questionGroup.Name, &questionGroup.Order, &question.ID, &question.Score, &question.Type, &question.Order, &question.GroupID, &question.Content, &question.Options, &question.Answers, &question.Analysis, &question.Title, &question.Input, &question.Output, &question.Example, &question.Repo, &question.CommitID)
		if err != nil {
			err = fmt.Errorf("unable to scan row data: %s", err)
			z.Error(err.Error())
			return
		}

		set, exist := questionSetsMap[question.GroupID.Int64]
		if !exist {
			set = &QuestionSet{
				ID:        question.GroupID.Int64,
				Name:      questionGroup.Name.String,
				Order:     int(questionGroup.Order.Int64),
				Questions: []*Question{},
			}
			questionSetsMap[question.GroupID.Int64] = set
		}

		var standardAnswers []*StandardAnswer
		standardAnswers, _, err = ConvertRawStandardAnswerData(question.Answers, question.Type.String)
		if err != nil {
			err = fmt.Errorf("unable to convert raw standard answer data: %v", err)
			z.Error(err.Error())
			standardAnswers = []*StandardAnswer{}
		}

		var analyze string

		if len(standardAnswers) > 1 {
			for i, standardAnswer := range standardAnswers {
				analyze += fmt.Sprintf("（%d）%s\n", i+1, standardAnswer.GradingRule)
			}
		} else if len(standardAnswers) == 1 {
			analyze = standardAnswers[0].GradingRule
		} else {
			analyze = ""
		}

		set.Score += question.Score.Float64
		set.Questions = append(set.Questions, &Question{
			ID:      question.ID.Int64,
			Analyze: analyze,
			Details: []*QuestionDetails{
				{Content: question.Content.String, Score: question.Score.Float64},
			},
			MarkRole:        analyze,
			OrderNum:        int(question.Order.Int64),
			Score:           question.Score.Float64,
			StandardAnswers: standardAnswers,
			Type:            question.Type.String,
		})

	}

	for _, result := range questionSetsMap {
		questionSets = append(questionSets, result)
	}

	return

}

func QueryExamineeInfo(ctx context.Context, cond QueryCondition) (students []Student, err error) {
	err = cmn.Validate(cond)
	if err != nil {
		err = fmt.Errorf("invalid query condition: %s", err.Error())
		z.Error(err.Error())
		return
	}

	query := `SELECT u.official_name, u.nickname, u.id, e.id, e.serial_number
							FROM t_examinee e
							LEFT JOIN t_user u ON e.student_id = u.id
							WHERE e.exam_session_id = $1 AND e.status = '10' %s -- 动态插入s
							ORDER BY e.serial_number`

	var args []interface{}
	args = append(args, cond.ExamSessionID)

	if cond.ExamineeID > 0 {
		query = fmt.Sprintf(query, " AND e.id = $2")
		args = append(args, cond.ExamineeID)
	} else {
		query = fmt.Sprintf(query, "")
	}

	pgxConn := cmn.GetPgxConn()
	rows, err := pgxConn.Query(ctx, query, args...)
	if err != nil {
		err = fmt.Errorf("exec query examinee info SQL error: %s", err.Error())
		z.Error(err.Error())
		return
	}

	for rows.Next() {
		var officialName null.String
		var nickname null.String
		var studentID null.Int
		var orderNumber null.Int
		var examineeID null.Int
		err = rows.Scan(&officialName, &nickname, &studentID, &examineeID, &orderNumber)
		if err != nil {
			err = fmt.Errorf("unable to scan row data:%s", err)
			z.Error(err.Error())
			return
		}
		students = append(students, Student{
			Name:       officialName.String,
			Nickname:   nickname.String,
			Number:     int(orderNumber.Int64),
			StudentID:  studentID.Int64,
			ExamineeID: examineeID.Int64,
		})
	}

	return
}

func UpdateMarkerInfoState(ctx context.Context, tx *pgx.Tx, teacherID int64, ids []int64, mode string) (targetIDs []int64, err error) {
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
	if err != nil {
		err = fmt.Errorf("exec update query error: %v", err)
		z.Error(err.Error())
		return
	}

	for rows.Next() {
		var id null.Int
		err = rows.Scan(&id)
		if err != nil {
			err = fmt.Errorf("unable to scan row data: %v", err)
			z.Error(err.Error())
			return
		}
		targetIDs = append(targetIDs, id.Int64)
	}

	return
}

func InsertOrUpdateMarkingResults(ctx context.Context, markingResults []*cmn.TMark) (targetIDs []int64, err error) {
	// 不存在则插入，否则更新，需要索引
	upsertMarkQuery := `
						INSERT INTO t_mark 
							(examinee_id, question_id, teacher_id, exam_session_id, mark_details, score, create_time, creator, status, updated_by, update_time, practice_submission_id, practice_id)
						VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
						ON CONFLICT (examinee_id, question_id, teacher_id, exam_session_id, practice_id, practice_submission_id) 
						DO UPDATE SET
							mark_details = EXCLUDED.mark_details,
							score = EXCLUDED.score,
							update_time = EXCLUDED.update_time,
							updated_by = EXCLUDED.updated_by
						RETURNING id
						`
	pgxConn := cmn.GetPgxConn()

	tx, err := pgxConn.Begin(ctx)
	if err != nil {
		err = fmt.Errorf("begin transaction error: %s", err.Error())
		z.Error(err.Error())
		return
	}

	defer func() {
		if err != nil {
			err_ := tx.Rollback(ctx)
			if err_ != nil {
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
			err = fmt.Errorf("teacherID is required for mark")
			z.Error(err.Error())
			return
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

		var id null.Int

		err = tx.QueryRow(ctx, upsertMarkQuery, mark.ExamineeID, mark.QuestionID, mark.TeacherID, mark.ExamSessionID, mark.MarkDetails, mark.Score, time.Now().UnixMilli(), mark.Creator, mark.Status, mark.UpdatedBy, mark.UpdateTime, mark.PracticeSubmissionID, mark.PracticeID).Scan(&id)
		if err != nil {
			err = fmt.Errorf("exec upsertMark query error: %s", err.Error())
			z.Error(err.Error())
			return
		}
		targetIDs = append(targetIDs, id.Int64)
	}

	err = tx.Commit(ctx)
	if err != nil {
		err = fmt.Errorf("commit transaction error: %s", err.Error())
		z.Error(err.Error())
		return
	}

	return
}

func updateStudentAnswerScore(ctx context.Context, markingResults []*cmn.TMark, cond QueryCondition) (targetIDs []int64, err error) {
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
	if err != nil {
		err = fmt.Errorf("begin transaction error: %v", err)
		z.Error(err.Error())
		return
	}

	defer func() {
		if err != nil {
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

		z.Sugar().Infof("--> %v", args)

		err = tx.QueryRow(ctx, updateQuery, args...).Scan(&targetID)
		if err != nil {
			err = fmt.Errorf("exec updateStudentAnswerScore sql error: %v", err)
			z.Error(err.Error())
			return
		}

		targetIDs = append(targetIDs, targetID)
	}

	err = tx.Commit(ctx)
	if err != nil {
		err = fmt.Errorf("commit tx error: %v", err)
		z.Error(err.Error())
		return
	}
	return
}

func updateExamSessionState(ctx context.Context, tx *pgx.Tx, teacherID int64, examSessionIDs []int64, status string) (targetIDs []int64, err error) {
	if teacherID <= 0 {
		err = fmt.Errorf("invalid params: teacher_id is required")
		z.Error(err.Error())
		return
	}

	if examSessionIDs == nil || len(examSessionIDs) == 0 {
		err = fmt.Errorf("no examSessionIDs to update")
		z.Error(err.Error())
		return
	}

	if status == "" {
		err = fmt.Errorf("invalid params: status is required")
		z.Error(err.Error())
		return
	}

	updateQuery := `UPDATE t_exam_session 
					SET
						status = $2,
						updated_by = $3, 
						update_time = $4
					WHERE id = ANY($1)
					RETURNING id`

	rows, err := (*tx).Query(ctx, updateQuery, pq.Array(examSessionIDs), status, teacherID, time.Now().UnixMilli())
	if err != nil {
		err = fmt.Errorf("exec updateExamSessionState sql error: %v", err)
		z.Error(err.Error())
		return
	}

	for rows.Next() {
		var id int64
		err = rows.Scan(&id)
		if err != nil {
			err = fmt.Errorf("unable to scan row data: %v", err)
			z.Error(err.Error())
			return
		}
		targetIDs = append(targetIDs, id)
	}

	return
}
