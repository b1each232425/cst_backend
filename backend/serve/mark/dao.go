package mark

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jmoiron/sqlx/types"
	"github.com/lib/pq"
	"math"
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
	TeacherID                 int64 `json:"teacher_id" validate:"required,gt=0"`
	ExamSessionID             int64 `json:"exam_session_id" validate:"gte=0"`
	ExamineeID                int64 `json:"examinee_id" validate:"gte=0"`
	PracticeID                int64 `json:"practice_id" validate:"gte=0"`
	PracticeSubmissionID      int64 `json:"practice_submission_id" validate:"gte=0"`
	PracticeWrongSubmissionID int64 `json:"practice_wrong_submission_id"`
	QuestionID                int64 `json:"question_id"`
}

type MarkerInfo struct {
	MarkerID           int64           `json:"marker_id"`
	MarkInfos          []cmn.TMarkInfo `json:"mark_infos"`
	MarkMode           string          `json:"mark_mode"` // 00：不需要手动批改  02：全卷多评 04：试卷分配 06：题组专评 08：题目分配 10：单人（人工）批改 12：批改单个练习学生 14：批改练习错题
	MarkMethod         string          `json:"mark_method"`
	MarkedStudentNames []string        `json:"marked_student_names"`
}

type QuestionSet struct {
	cmn.TExamPaperGroup
	Score     float64                   `json:"Score"`     // 题组分数
	Questions []*cmn.TExamPaperQuestion `json:"Questions"` // 题目
}

// QuestionDetails 题目
//type QuestionDetails struct {
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

// 获取考试列表 管理员查询全部列表，教师查询与其有关的考试列表
func QueryExamList(ctx context.Context, req QueryMarkingListReq) ([]Exam, int, error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	if err := cmn.Validate(req); err != nil {
		return nil, -1, fmt.Errorf("无效的 query cond: %v", err)
	}

	teacherID := req.User.ID

	getExamCountQuery := `	SELECT COUNT(DISTINCT es.id) AS total_exams
							FROM t_exam_session es
							JOIN t_exam_info ei ON es.exam_id = ei.id AND ei.status != '12' AND ei.status != '14' AND ei.status != '16' 
							JOIN t_paper p ON p.id = es.paper_id 							
							JOIN t_exam_paper ep ON ep.id = p.exampaper_id AND ep.status != '04'
							LEFT JOIN t_mark_info mi ON mi.exam_session_id = es.id AND mi.status != '04'
							WHERE es.status IS NOT NULL
							  AND es.status != '14' %s -- 动态关联where条件`

	var (
		countArgs           []interface{}
		rowCountWhereClause []string
		countArgIdx         = 1
		rowCount            int
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
	err := pgxConn.QueryRow(ctx, getExamCountQuery, countArgs...).Scan(&rowCount)
	if err != nil || forceErr == "QueryExamList-pgxConn.QueryRow" {
		return nil, -1, fmt.Errorf("获取考试列表总数失败: %v", err)
	}

	examListQuery := ` 
					SELECT 
						ei.id AS exam_id, 
						ei.name, 
						ei.type, 
						es.id AS session_id, 
						es.session_num, 
						es.start_time, 
						es.end_time, 
						es.status, 
						es.mark_method, 
						es.mark_mode, 
						ep.name, 
						mi.mark_examinee_ids, 
						mi.question_ids,
						mi.status, 
						COALESCE(erc.respondent_count, 0) AS respondent_count, 
					--	COALESCE(eumc.unmarked_student_count, 0) AS unmarked_student_count, 
						COALESCE(mc.marked_count, -1) AS marked_student_count 
					FROM
						 t_exam_session es 
					JOIN t_exam_info ei ON ei.id = es.exam_id AND ei.status !='12' AND ei.status != '14' AND ei.status != '16' %s
					JOIN t_paper p ON p.id = es.paper_id 
					JOIN t_exam_paper ep 
						ON ep.id = p.exampaper_id 
						AND ep.status IS NOT NULL 
						AND ep.status != '04' 
					LEFT JOIN v_exam_respondent_count erc ON erc.exam_session_id = es.id 
				--  LEFT JOIN v_exam_unmarked_student_count eumc ON eumc.exam_session_id = es.id 
					LEFT JOIN v_exam_teacher_marked_count mc ON mc.exam_session_id = es.id 
					LEFT JOIN t_mark_info mi ON es.id = mi.exam_session_id AND mi.status != '04' 
					WHERE es.status IS NOT NULL AND es.status != '14' %s -- 动态插入where条件
					ORDER BY
						ei.create_time DESC, ei.id DESC, es.start_time ASC 
					LIMIT $1 OFFSET $2
					`

	var (
		examWhereClause string
		whereClause     []string
		args            []interface{}
		argIdx          = 1
	)
	args = append(args, req.Limit, req.Offset)
	argIdx += 2

	if req.ExamName != "" {
		examWhereClause = fmt.Sprintf(" AND ei.name ILIKE $%d ", argIdx)
		//whereClause = append(whereClause, fmt.Sprintf(" AND e.name ILIKE $%d ", argIdx))
		args = append(args, "%"+req.ExamName+"%")
		argIdx++
	}

	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		whereClause = append(whereClause, fmt.Sprintf(" AND es.start_time >= $%d AND es.start_time <= $%d ", argIdx, argIdx+1))
		args = append(args, req.StartTime.UnixMilli(), req.EndTime.UnixMilli())
		argIdx += 2
	}

	// TODO 需要处理，传进来的？
	if !req.User.IsAdmin {
		// 普通批阅员，非管理员
		whereClause = append(whereClause, fmt.Sprintf(" AND mi.mark_teacher_id = $%d AND mc.teacher_id = $%d ", argIdx, argIdx))
		args = append(args, teacherID)
		argIdx += 2
	}

	examListQuery = fmt.Sprintf(examListQuery, examWhereClause, strings.Join(whereClause, " "))

	rows, err := pgxConn.Query(ctx, examListQuery, args...)
	if err != nil || forceErr == "QueryExamList-pgxConn.Query" {
		return nil, -1, fmt.Errorf("获取考试列表 sql 失败: %v", err)
	}
	defer rows.Close()

	// key 是 exam_id
	examsMap := make(map[int64]*Exam, req.Limit)
	examsMapKeys := make([]int64, 0, req.Limit)

	for rows.Next() {
		var (
			exam               cmn.TExamInfo
			session            cmn.TExamSession
			markedStudentCount null.Int
			respondentCount    null.Int
			paperName          null.String
			//unmarkedStudentCount      cmn.NullInt
			//totalUnmarkedStudentCount null.Int
			markExamineeIdsJson types.JSONText
			markStatus          null.String
			questionIDsJson     types.JSONText
		)

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
			&markExamineeIdsJson,
			&questionIDsJson,
			&markStatus,
			&respondentCount,
			//&totalUnmarkedStudentCount,
			&markedStudentCount,
		)
		if err != nil || forceErr == "QueryExamList-rows.Scan" {
			return nil, -1, fmt.Errorf("从 row 中获取值失败: %v", err)
		}

		currentExamID := exam.ID.Int64

		e, exists := examsMap[currentExamID]
		if !exists {
			e = &Exam{
				Id:   exam.ID.Int64,
				Name: exam.Name.String,
				Type: exam.Type.String,
			}
			examsMap[currentExamID] = e
			examsMapKeys = append(examsMapKeys, currentExamID)
		}

		// 待批改数的计算
		// TODO
		unMarkedStudentCount := 0

		if markedStudentCount.Int64 == -1 { // TODO???
			// 说明无主观题目需要批改
			unMarkedStudentCount = 0
		} else {
			totalCount := respondentCount.Int64

			switch session.MarkMode.String {
			// 试卷分配
			case "04":
				var markExamineeIds []int64
				if err = json.Unmarshal(markExamineeIdsJson, &markExamineeIds); err != nil {
					return nil, -1, fmt.Errorf("解析 markExamineeIdsJson 失败: %v", err)
				}

				totalCount = int64(len(markExamineeIds))

			// 题组分配 // 题目分配
			case "06", "08":
				var questionIDs []int64
				if err = json.Unmarshal(questionIDsJson, &questionIDs); err != nil {
					return nil, -1, fmt.Errorf("解析 questionIDsJson 失败: %v", err)
				}

				// 没有需要批改的题目,待批改数数置 0
				if len(questionIDs) == 0 {
					totalCount = 0
				}
			}

			// 需要相减的原因: v_exam_teacher_marked_count 是根据 t_mark 的, 符合实际
			unMarkedStudentCount = int(math.Max(float64(totalCount-markedStudentCount.Int64), 0))
		}

		//if !req.User.IsAdmin && session.MarkMode.String == "04" {
		//	// 非管理员下的试卷分配模式
		//	var examineeIDs []int64
		//	if err = json.Unmarshal(markExamineeIdsJson, &examineeIDs); err != nil {
		//		return nil, -1, fmt.Errorf("解析 markExamineeIdsJson 失败: %v", err)
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
	if err = rows.Err(); err != nil {
		return nil, -1, fmt.Errorf("行遍历失败: %v", err)
	}

	examList := make([]Exam, 0, req.Limit)

	for _, key := range examsMapKeys {
		examList = append(examList, *examsMap[key])
	}

	return examList, rowCount, nil
}

// 获取练习列表 管理员查询全部列表，教师查询与其有关的练习列表
func QueryPracticeList(ctx context.Context, req QueryMarkingListReq) ([]Practice, int, error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	if err := cmn.Validate(req); err != nil {
		return nil, -1, fmt.Errorf("无效的 query condition: %v", err)
	}

	teacherID := req.User.ID

	rowCountQuery := `	SELECT COUNT(*) AS practice_count
								FROM t_practice p
								LEFT JOIN t_mark_info mi ON mi.practice_id = p.id AND mi.status != '04' 
								WHERE p.status != '04' %s -- 动态拼接`

	var (
		countArgs           []interface{}
		rowCountWhereClause []string
		countArgIdx         = 1
		rowCount            int
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
	err := pgxConn.QueryRow(ctx, rowCountQuery, countArgs...).Scan(&rowCount)
	if err != nil || forceErr == "QueryPracticeList-pgxConn.QueryRow" {
		return nil, -1, fmt.Errorf("查询练习列表失败: %v", err)
	}

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
	if err != nil || forceErr == "QueryPracticeList-pgxConn.Query" {
		return nil, -1, fmt.Errorf("获取所有练习数目 sql 失败: %v", err)
	}
	defer rows.Close()

	practices := make([]Practice, 0, req.Limit)

	for rows.Next() {
		var p Practice
		var row cmn.TPractice
		err = rows.Scan(&row.ID, &row.Name, &row.CorrectMode, &row.Type, &p.RespondentCount, &p.UnMarkedStudentCount)
		if err != nil || forceErr == "QueryPracticeList-rows.Scan" {
			return nil, -1, fmt.Errorf("扫描解析练习数据失败: %v", err)
		}

		p.MarkMode = row.CorrectMode.String
		p.Type = row.Type.String
		p.Name = row.Name.String
		p.ID = row.ID.Int64

		practices = append(practices, p)
	}
	if err = rows.Err(); err != nil {
		return nil, -1, fmt.Errorf("行遍历错误: %v", err)
	}

	return practices, rowCount, nil
}

// 获取所有批改员的阅卷结果
func QueryMarkingResults(ctx context.Context, cond QueryCondition) ([]cmn.TMark, error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	if cond.TeacherID <= 0 {
		return nil, errors.New("无效的请求参数，必须包含 教师ID")
	}

	if cond.ExamSessionID <= 0 && cond.PracticeID <= 0 {
		return nil, fmt.Errorf("无效的请求参数，必须包含 考试场次ID 或者 练习ID 中的一个")
	}

	if cond.PracticeID > 0 && cond.ExamSessionID > 0 {
		return nil, fmt.Errorf("无效的请求参数，不能同时包含练习ID和考试场次ID")
	}

	var (
		whereClause []string
		args        []interface{}
	)
	//var argIndex = 2
	var argIndex = 1

	// 获取当前教师批阅的所有阅卷结果
	//args = append(args, cond.TeacherID)

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
								WHERE q.type IN ('06', '08') %s -- 动态拼接where条件
								ORDER BY m.examinee_id, m.practice_submission_id, q.order`

	pgxConn := cmn.GetPgxConn()

	getMarkingResultQuery = fmt.Sprintf(getMarkingResultQuery, strings.Join(whereClause, " "))

	rows, err := pgxConn.Query(ctx, getMarkingResultQuery, args...)
	if err != nil || forceErr == "QueryMarkingResults-pgxConn.Query" {
		return nil, fmt.Errorf("获取批改结果 sql 失败: %v", err)
	}
	defer rows.Close()

	markingResults := make([]cmn.TMark, 0, 10) // 假设 10

	for rows.Next() {
		var mr cmn.TMark
		err = rows.Scan(&mr.TeacherID, &mr.ExamineeID, &mr.PracticeSubmissionID, &mr.QuestionID, &mr.MarkDetails, &mr.Score)
		if err != nil || forceErr == "QueryMarkingResults-rows.Scan" {
			return nil, fmt.Errorf("从 row 读取结果失败: %v", err)
		}

		if cond.ExamSessionID > 0 {
			mr.ExamSessionID = null.IntFrom(cond.ExamSessionID)
		} else {
			mr.PracticeID = null.IntFrom(cond.PracticeID)
		}

		markingResults = append(markingResults, mr)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("行遍历错误: %v", err)
	}

	return markingResults, nil
}

// 查询阅卷配置信息
func QueryMarkerInfo(ctx context.Context, cond QueryCondition) (MarkerInfo, error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)

	// 参数校验
	if cond.ExamSessionID <= 0 && cond.PracticeID <= 0 {
		return MarkerInfo{}, errors.New("无效的请求参数，请求参数必须包含 考试场次ID 或者 练习ID 中的一个")
	}

	var (
		query       string
		queryParams []interface{}
	)

	// 构建不同的SQL查询
	if cond.ExamSessionID > 0 {
		query = `SELECT mi.mark_teacher_id, mi.mark_count, 
                        mi.mark_question_groups, mi.mark_examinee_ids, mi.question_ids,
                        es.mark_mode, es.mark_method, u.official_name
                 FROM t_mark_info mi
                 JOIN t_exam_session es ON es.id = mi.exam_session_id
                 JOIN t_user u ON u.id = mi.mark_teacher_id
                 WHERE mi.exam_session_id = $1 AND mi.status != '04'`
		queryParams = append(queryParams, cond.ExamSessionID)
	} else {
		query = `SELECT mi.mark_teacher_id, mi.mark_count, 
                        mi.mark_question_groups, mi.mark_examinee_ids, mi.question_ids,
                        p.correct_mode 
                 FROM t_mark_info mi
                 JOIN t_practice p ON p.id = mi.practice_id
                 WHERE mi.practice_id = $1 AND mi.status != '04'`
		queryParams = append(queryParams, cond.PracticeID)
	}

	// 做这个判断，是因为自动批改时，查询是没有 teacher_id 的，提交时自动批改，主体是学生，学生的 id
	if cond.TeacherID > 0 {
		query = query + " AND mi.mark_teacher_id = $2 "
		queryParams = append(queryParams, cond.TeacherID)
	}

	pgxConn := cmn.GetPgxConn()
	rows, err := pgxConn.Query(ctx, query, queryParams...)
	if err != nil || forceErr == "QueryMarkerInfo-pgxConn.Query" {
		return MarkerInfo{}, fmt.Errorf("获取批改配置信息 sql 失败: %v", err)
	}
	defer rows.Close()

	var (
		markerInfo MarkerInfo
		markMode   null.String
		markMethod null.String
		scanVars   []interface{}
	)

	for rows.Next() {
		var mi cmn.TMarkInfo
		var name null.String

		scanVars = []interface{}{&mi.MarkTeacherID, &mi.MarkCount,
			&mi.MarkQuestionGroups, &mi.MarkExamineeIds, &mi.QuestionIds}

		if cond.ExamSessionID != 0 {
			scanVars = append(scanVars, &markMode, &markMethod, &name)
		} else {
			scanVars = append(scanVars, &markMode)
		}

		err = rows.Scan(scanVars...)
		if err != nil || forceErr == "QueryMarkerInfo-rows.Scan" {
			return MarkerInfo{}, fmt.Errorf("从 row 读取结果失败: %v", err)
		}

		markerInfo.MarkInfos = append(markerInfo.MarkInfos, mi) // 每个老师一个值。数组：多人批改
		if cond.ExamSessionID != 0 {
			markerInfo.MarkedStudentNames = append(markerInfo.MarkedStudentNames, name.String) // TODO 错误？学生的名字不是老师的名字
		}
	}
	if err = rows.Err(); err != nil {
		return MarkerInfo{}, fmt.Errorf("行遍历失败: %v", err)
	}

	markerInfo.MarkMode = markMode.String
	markerInfo.MarkMethod = markMethod.String
	// TODO  MarkerID

	return markerInfo, nil
}

// 查询学生的答案  questionType: "00": 客观题， "02"：主观题
func QueryStudentAnswers(ctx context.Context, tx pgx.Tx, questionType string, cond QueryCondition, markerInfo MarkerInfo) ([]cmn.TVStudentAnswerQuestion, error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)

	if questionType != "00" && questionType != "02" {
		return nil, fmt.Errorf("无效的题目类型: %v", questionType)
	}

	// 考试，练习，练习错题，答案都是存储在 t_student_answer 表的，后两个都是使用 practice_submission_id 区别的， 意味着，同一时刻，同个学生的一个练习及其练习错题只存在一种
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

	if questionType == "00" {
		// 单选、多选、判断题
		whereClause = append(whereClause, " AND type IN ('00', '02', '04') ")
	} else {
		// 填空、简答题
		whereClause = append(whereClause, " AND type IN ('06', '08') ")
	}

	switch markerInfo.MarkMode {
	case "00", "04", "10": // 自动批改 试卷分配 单人批改 // 查询单个考试考生的数据 // 重构，一次只获取一个考生的答案
		var markExamineeIDs []int64

		if cond.ExamineeID > 0 { // 获取单个考生的数据
			markExamineeIDs = append(markExamineeIDs, cond.ExamineeID)
		} else {
			// 第一位老师是否总是自己？
			if err := json.Unmarshal(markerInfo.MarkInfos[0].MarkExamineeIds, &markExamineeIDs); err != nil {
				return nil, fmt.Errorf("解析 markerInfo.MarkInfos[0].MarkExamineeIds 失败: %v", err)
			}
		}

		whereClause = append(whereClause, fmt.Sprintf(" AND examinee_id = ANY($%d) ", argIdx))
		argIdx++
		args = append(args, pq.Array(markExamineeIDs))

	case "06": // 题组专评 // TODO 题目专评 “08”
		var questionIDs []int64
		if err := json.Unmarshal(markerInfo.MarkInfos[0].QuestionIds, &questionIDs); err != nil {
			return nil, fmt.Errorf("解析 markerInfo.MarkInfos[0].QuestionIds 失败: %v", err)
		}

		whereClause = append(whereClause, fmt.Sprintf(" AND question_id = ANY($%d) ", argIdx))
		argIdx++
		args = append(args, pq.Array(questionIDs))

	case "12": // 查询单个练习学生的数据，包括错题
		if cond.PracticeSubmissionID <= 0 {
			return nil, errors.New("无效的 practice_submission_id")
		}

		whereClause = append(whereClause, fmt.Sprintf(" AND practice_submission_id = $%d ", argIdx))
		argIdx++
		args = append(args, cond.PracticeSubmissionID)

		// 如果是错题，由于是自动批改，只需要错题，减少 token 消耗
		if cond.PracticeWrongSubmissionID > 0 {
			whereClause = append(whereClause, " AND question_score != answer_score ")
		}
	}

	query = fmt.Sprintf(query, strings.Join(whereClause, " "))

	var (
		rows pgx.Rows
		err  error
	)

	if tx != nil {
		rows, err = tx.Query(ctx, query, args...)
	} else {
		pgxConn := cmn.GetPgxConn()
		rows, err = pgxConn.Query(ctx, query, args...)
	}

	if err != nil || forceErr == "QueryStudentAnswers-pgxConn.Query" {
		return nil, fmt.Errorf("获取学生答案 sql 错误: %v", err)
	}
	defer rows.Close()

	studentAnswers := make([]cmn.TVStudentAnswerQuestion, 0, 50) // 假设 50

	for rows.Next() {
		var sa cmn.TVStudentAnswerQuestion
		err = rows.Scan(
			&sa.ExamSessionID,
			&sa.PracticeID,
			&sa.QuestionID,
			&sa.Order,
			&sa.Type,
			&sa.QuestionScore,
			&sa.Answer,
			&sa.AnswerScore,
			&sa.ActualAnswers,
			&sa.ActualOptions,
			&sa.ExamineeID,
			&sa.PracticeSubmissionID,
		)
		if err != nil || forceErr == "rows.Scan" {
			return nil, fmt.Errorf("从 row 中读取结果失败: %v", err)
		}

		studentAnswers = append(studentAnswers, sa)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("行遍历失败: %v", err)
	}

	return studentAnswers, nil
}

// 查询主观题 （可查询单道题目）
func QuerySubjectiveQuestions(ctx context.Context, tx pgx.Tx, cond QueryCondition, markerInfo MarkerInfo) ([]QuestionSet, error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)

	getQuestionsQuery := `	SELECT pg.id, pg.name, pg."order", q.id, q.score, q.type, q."order", q.group_id, q.content, q.options, 
								   q.answers, q.analysis, q.title 
							FROM t_exam_paper p
							JOIN t_exam_paper_group pg ON pg.exam_paper_id = p.id 
							JOIN t_exam_paper_question q ON q.group_id = pg.id
							JOIN t_paper tp ON tp.exampaper_id = p.id 
							LEFT JOIN t_exam_session es ON es.paper_id = tp.id 
							LEFT JOIN t_practice pra ON pra.paper_id = tp.id
							WHERE p.status != '04' AND p.status IS NOT NULL 
							  AND q.status != '04' AND q.status IS NOT NULL 
							  AND q.type IN ('06', '08', '10') 
							  %s -- 动态拼接where条件`

	var (
		whereClause []string
		args        []interface{}
		argIndex    = 1
	)

	if cond.ExamSessionID > 0 {
		whereClause = append(whereClause, fmt.Sprintf(" AND es.id = $%d ", argIndex))
		args = append(args, cond.ExamSessionID)
	} else {
		whereClause = append(whereClause, fmt.Sprintf(" AND pra.id = $%d ", argIndex))
		args = append(args, cond.PracticeID)
	}
	argIndex++

	if cond.QuestionID > 0 {
		whereClause = append(whereClause, fmt.Sprintf(" AND q.id = $%d ", argIndex))
		args = append(args, cond.QuestionID)
		argIndex++
	}

	var err error

	switch markerInfo.MarkMode {
	case "06": // 题组专评
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
	case "08": // 题目分配
	}

	getQuestionsQuery = fmt.Sprintf(getQuestionsQuery, strings.Join(whereClause, " "))

	var rows pgx.Rows

	if tx != nil {
		rows, err = tx.Query(ctx, getQuestionsQuery, args...)
	} else {
		pgxConn := cmn.GetPgxConn()
		rows, err = pgxConn.Query(ctx, getQuestionsQuery, args...)
	}

	if err != nil || forceErr == "QuerySubjectiveQuestions-pgxConn.Query" {
		return nil, fmt.Errorf("获取主观题 sql 失败: %v", err)
	}
	defer rows.Close()

	questionSetsMap := make(map[int64]*QuestionSet, 10)

	for rows.Next() {
		var q cmn.TExamPaperQuestion
		var qg cmn.TExamPaperGroup
		err = rows.Scan(&qg.ID, &qg.Name, &qg.Order, &q.ID, &q.Score, &q.Type, &q.Order, &q.GroupID, &q.Content, &q.Options, &q.Answers, &q.Analysis, &q.Title)
		if err != nil || forceErr == "QuerySubjectiveQuestions-rows.Scan" {
			return nil, fmt.Errorf("从 row 中读取结果失败: %v", err)
		}

		currentQuestionGroupID := q.GroupID.Int64

		set, exist := questionSetsMap[currentQuestionGroupID]
		if !exist {
			set = &QuestionSet{
				TExamPaperGroup: qg,
				Score:           0.0,
				Questions:       []*cmn.TExamPaperQuestion{},
			}
			questionSetsMap[currentQuestionGroupID] = set
		}

		var standardAnswers []*SubjectiveAnswer
		err = json.Unmarshal(q.Answers, &standardAnswers)
		if err != nil {
			return nil, fmt.Errorf("解析 q.Answers 失败: %v", err)
		}

		set.Score += q.Score.Float64
		set.Questions = append(set.Questions, &q)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("行遍历: %v", err)
	}

	questionSets := make([]QuestionSet, 0, 10)

	for _, result := range questionSetsMap {
		questionSets = append(questionSets, *result)
	}

	return questionSets, nil
}

func QueryExamineeInfo(ctx context.Context, cond QueryCondition) ([]cmn.TVExamineeInfo, error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	if err := cmn.Validate(cond); err != nil {
		return nil, fmt.Errorf("无效的 query condition: %v", err)
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
	if err != nil || forceErr == "QueryExamineeInfo-pgxConn.Query" {
		return nil, fmt.Errorf("查询考生信息 sql 失败: %v", err)
	}
	defer rows.Close()

	examineeInfos := make([]cmn.TVExamineeInfo, 0, 10)

	for rows.Next() {
		var ei cmn.TVExamineeInfo
		err = rows.Scan(&ei.OfficialName, &ei.ID, &ei.SerialNumber)
		if err != nil || forceErr == "QueryExamineeInfo-rows.Scan" {
			return nil, fmt.Errorf("从 row 中读取结果失败: %v", err)
		}
		examineeInfos = append(examineeInfos, ei)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("行迭代错误: %v", err)
	}

	return examineeInfos, nil
}

func QueryExamSessionStatus(ctx context.Context, tx pgx.Tx, examSessionID int64) (string, error) {
	queryExamSessionSql := `SELECT status FROM t_exam_session WHERE id = $1`

	var (
		examSessionStatus string
		err               error
	)

	if tx == nil {
		pgxConn := cmn.GetPgxConn()
		err = pgxConn.QueryRow(ctx, queryExamSessionSql, examSessionID).Scan(&examSessionStatus)
	} else {
		err = tx.QueryRow(ctx, queryExamSessionSql, examSessionID).Scan(&examSessionStatus)
	}

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("查询不到考试场次 %d : %v", examSessionID, err)
		} else {
			return "", fmt.Errorf("执行 queryExamSessionSql 失败: %v", err)
		}
	}

	return examSessionStatus, nil
}

func QueryStudentInfos(ctx context.Context, cond QueryCondition, markerInfo MarkerInfo) ([]StudentInfo, error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	if cond.TeacherID <= 0 {
		err := errors.New("无效的请求参数，必须包含 教师ID")
		z.Error(err.Error())
		return nil, err
	}

	if cond.PracticeID <= 0 && cond.ExamSessionID <= 0 {
		return nil, errors.New("无效的请求参数，必须包含 考试场次ID 或者 练习ID 中的一个")
	}

	if cond.PracticeID > 0 && cond.ExamSessionID > 0 {
		return nil, errors.New("无效的请求参数，不能同时包含 练习ID 和 考试场次ID")
	}

	var query string
	var args []interface{}
	var whereClause []string
	argIdx := 2
	if cond.ExamSessionID > 0 {
		query = `	SELECT official_name, id, serial_number
					FROM v_examinee_info
					WHERE exam_session_id = $1 AND examinee_status = '10' %s -- 动态拼接where条件
					ORDER BY serial_number`
		args = append(args, cond.ExamSessionID)
		if cond.ExamineeID > 0 {
			whereClause = append(whereClause, fmt.Sprintf(" AND id = $%d ", argIdx))
			args = append(args, cond.ExamineeID)
			argIdx++
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
			whereClause = append(whereClause, fmt.Sprintf(" AND mp.submission_id = $%d ", argIdx))
			args = append(args, cond.PracticeSubmissionID)
			argIdx++
		}
	}

	// 试卷分配（考试）
	if markerInfo.MarkMode == "04" {
		var examineeIds []int64
		err := json.Unmarshal(markerInfo.MarkInfos[0].MarkExamineeIds, &examineeIds)
		if err != nil {
			return nil, fmt.Errorf("解析 markerInfo.MarkInfos[0].MarkExamineeIds 失败: %v", err)
		}

		whereClause = append(whereClause, fmt.Sprintf(" AND id = ANY($%d) ", argIdx))
		args = append(args, pq.Array(examineeIds))
		argIdx++
	}

	query = fmt.Sprintf(query, strings.Join(whereClause, " "))

	pgxConn := cmn.GetPgxConn()
	rows, err := pgxConn.Query(ctx, query, args...)
	if err != nil || forceErr == "QueryStudentInfo-pgxConn.Query" {
		return nil, fmt.Errorf("查询学生信息 sql 失败: %v", err)
	}
	defer rows.Close()

	studentInfos := make([]StudentInfo, 0, 50) // 假设每次 50 个学生考试

	for rows.Next() {
		var si StudentInfo

		var scanVars []interface{}
		if cond.ExamSessionID > 0 {
			scanVars = []interface{}{&si.OfficialName, &si.ExamineeID, &si.SerialNumber}
		} else {
			scanVars = []interface{}{&si.OfficialName, &si.PracticeSubmissionID}
		}

		err = rows.Scan(scanVars...)
		if err != nil || forceErr == "QueryStudentInfos-rows.Scan" {
			return nil, fmt.Errorf("从 row 中读取结果失败: %v", err)
		}
		studentInfos = append(studentInfos, si)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("行遍历错误: %v", err)
	}

	return studentInfos, nil
}

func UpdateMarkerInfoState(ctx context.Context, tx *pgx.Tx, teacherID int64, ids []int64, mode string) ([]int64, error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	if teacherID <= 0 {
		return nil, errors.New("无效的 teacher ID")
	}

	if ids == nil || len(ids) == 0 {
		return nil, errors.New("没有 exam_session_ids 或者 practice_ids 可以用于更新")
	}

	var whereClause []string

	updateQuery := `UPDATE t_mark_info 
					SET
						status = $2,
						updated_by = $3, 
						update_time = $4
					WHERE %s 
					RETURNING id`

	switch mode {
	case "00":
		whereClause = append(whereClause, "exam_session_id = ANY($1)")
	case "02":
		whereClause = append(whereClause, "practice_id = ANY($1)")
	default:
		return nil, fmt.Errorf("无效的更新模式: %s", mode)
	}

	updateQuery = fmt.Sprintf(updateQuery, strings.Join(whereClause, " "))

	rows, err := (*tx).Query(ctx, updateQuery, pq.Array(ids), "04", teacherID, time.Now().UnixMilli())
	if err != nil || forceErr == "tx.Query" {
		return nil, fmt.Errorf("更新批改配置信息 sql 失败: %v", err)
	}
	defer rows.Close()

	targetIDs := make([]int64, 0, len(ids))

	for rows.Next() {
		var id null.Int
		err = rows.Scan(&id)
		if err != nil || forceErr == "rows.Scan" {
			return nil, fmt.Errorf("从 row 中读取结果失败: %v", err)
		}
		targetIDs = append(targetIDs, id.Int64)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("行遍历错误: %v", err)
	}

	return targetIDs, nil
}

func UpsertMarkingResults(ctx context.Context, tx pgx.Tx, markingResults []cmn.TMark) ([]int64, error) {
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

	targetIDs := make([]int64, 0, len(markingResults))

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
			return nil, errors.New("questionID 用于记录批改过程是必需的")
		}

		validExam := mark.ExamSessionID.Int64 > 0 && mark.ExamineeID.Int64 > 0 &&
			mark.PracticeID.Int64 <= 0 && mark.PracticeSubmissionID.Int64 <= 0

		validPractice := mark.PracticeID.Int64 > 0 && mark.PracticeSubmissionID.Int64 > 0 &&
			mark.ExamSessionID.Int64 <= 0 && mark.ExamineeID.Int64 <= 0

		if !(validExam || validPractice) {
			return nil, errors.New("无效的插入参数：必须满足以下条件之一：(exam session id 和 examinee id) 同时大于0且其他参数小于等于0，或者 (practice id 和 practice submission id) 同时大于0且其他参数小于等于0")
		}

		var query string
		if validExam {
			query = upsertExamMarkQuery
		} else {
			query = upsertPracticeMarkQuery
		}

		var (
			id  null.Int
			err error
		)
		args := make([]interface{}, 0, 13)

		args = append(args, mark.ExamineeID, mark.QuestionID, mark.TeacherID, mark.ExamSessionID, mark.MarkDetails, mark.Score, time.Now().UnixMilli(), mark.Creator, mark.Status, mark.UpdatedBy, mark.UpdateTime, mark.PracticeSubmissionID, mark.PracticeID)

		if tx != nil {
			err = tx.QueryRow(ctx, query, args...).Scan(&id)
		} else {
			pgConn := cmn.GetPgxConn()
			err = pgConn.QueryRow(ctx, query, args...).Scan(&id)
		}

		if err != nil || forceErr == "tx.QueryRow" {
			return nil, fmt.Errorf("记录批改过程 sql 失败: %v", err)
		}
		targetIDs = append(targetIDs, id.Int64)
	}

	return targetIDs, nil
}

// 查询当前老师需要批改的考试的已批改人数和已批改总题数
// 没必要开启只读事务
func QueryMarkedPersonAndQuestionCount(ctx context.Context, tx pgx.Tx, cond QueryCondition) (int64, int64, error) {
	var queryMarkedPerson string
	var queryQuestionCount string
	var args []interface{}

	if cond.ExamSessionID > 0 {
		queryMarkedPerson = `SELECT etmc.marked_count 
			     		     FROM v_exam_teacher_marked_count etmc 
				             WHERE etmc.teacher_id = $1 AND etmc.exam_session_id = $2`

		queryQuestionCount = `SELECT COUNT(*) 
							  FROM t_mark
							  WHERE teacher_id = $1 AND exam_session_id = $2`

		args = append(args, cond.TeacherID, cond.ExamSessionID)
	} else {
		queryMarkedPerson = `SELECT (
							 SELECT COUNT(Distinct ps.student_id) FROM t_practice_submissions ps WHERE ps.practice_id = $1)-pusc.unmarked_count
       						 FROM v_practice_unmarked_student_cnt pusc 
							 WHERE pusc.practice_id = $1`

		queryQuestionCount = `SELECT COUNT(*)
							  FROM t_mark m
							  JOIN t_practice_submissions ps ON m.practice_submission_id = ps.id and ps.status = '06' -- 因为练习能多次提交的特殊，所以需要status判断
							  WHERE m.practice_id = $1`

		args = append(args, cond.PracticeID)
	}

	var (
		markedPerson, markedQuestionCount int64
		err                               error
	)

	if tx == nil {
		pgxConn := cmn.GetPgxConn()
		err = pgxConn.QueryRow(ctx, queryMarkedPerson, args...).Scan(&markedPerson)
	} else {
		err = tx.QueryRow(ctx, queryMarkedPerson, args...).Scan(&markedPerson)
	}

	if err != nil {
		return 0, 0, fmt.Errorf("获取已批改人数 sql 失败: %v", err)
	}

	if tx == nil {
		pgxConn := cmn.GetPgxConn()
		err = pgxConn.QueryRow(ctx, queryQuestionCount, args...).Scan(&markedQuestionCount)
	} else {
		err = tx.QueryRow(ctx, queryQuestionCount, args...).Scan(&markedQuestionCount)
	}

	if err != nil {
		return 0, 0, fmt.Errorf("获取已批改问题数 sql 失败: %v", err)
	}

	return markedPerson, markedQuestionCount, nil
}

// 用于主观题保存，客观题保存，ai批改分数保存
func UpdateStudentAnswerScore(ctx context.Context, tx pgx.Tx, markingResults []cmn.TMark, cond QueryCondition) ([]int64, error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)

	if len(markingResults) == 0 {
		return nil, fmt.Errorf("没有批改结果可用于更新")
	}

	updateQuery := `UPDATE t_student_answers
					SET answer_score = $1, 
						update_time = $2, 
						updated_by = $3  -- 被谁提交
					WHERE question_id = $4 %s -- 动态拼接where  -- 考生id 与 问题id 可以确定一行
					RETURNING id`

	// 批阅具体细节存放在 t_mark

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
		return nil, fmt.Errorf("无效的 exam_session_id 和 practice_id")
	}

	updateQuery = fmt.Sprintf(updateQuery, strings.Join(whereClause, " "))

	// studentID --> questionID --> 不同老师分数 用以计算平均分
	studentQuestionScoresMap := make(map[int64]map[int64][]float64, 100) // 假设100个人

	for _, mr := range markingResults {
		var currentStudentID int64

		if cond.ExamSessionID > 0 {
			if mr.ExamineeID.Int64 <= 0 {
				return nil, fmt.Errorf("exam_session_id 存在时， examinee_id 是必要的")
			} else {
				currentStudentID = mr.ExamineeID.Int64
			}
		} else if cond.PracticeID > 0 {
			if mr.PracticeSubmissionID.Int64 <= 0 {
				return nil, fmt.Errorf("practice_id 存在时，practice_submission_id 是必要的")
			} else {
				currentStudentID = mr.PracticeSubmissionID.Int64
			}
		}

		currentQuestionID := mr.QuestionID.Int64

		if _, ok := studentQuestionScoresMap[currentStudentID]; !ok {
			studentQuestionScoresMap[currentStudentID] = make(map[int64][]float64, 100) // 100道题目
		}

		if _, ok := studentQuestionScoresMap[currentStudentID][currentQuestionID]; !ok {
			studentQuestionScoresMap[currentStudentID][currentQuestionID] = make([]float64, 0, 5) // 假设一道题目最多5小题
		}

		studentQuestionScoresMap[currentStudentID][currentQuestionID] = append(studentQuestionScoresMap[currentStudentID][currentQuestionID], mr.Score.Float64)
	}

	var (
		targetIDs []int64
		err       error
	)

	for studentID, questionScores := range studentQuestionScoresMap {
		for questionID, scores := range questionScores {
			var avgScore float64
			for _, score := range scores {
				avgScore += score
			}
			avgScore /= float64(len(scores))

			var targetID int64
			args := []interface{}{
				avgScore,
				time.Now().UnixMilli(),
				cond.TeacherID,
				questionID,
			}

			if cond.ExamSessionID > 0 {
				args = append(args, studentID)
			} else if cond.PracticeID > 0 {
				args = append(args, cond.PracticeSubmissionID)
			}

			err = tx.QueryRow(ctx, updateQuery, args...).Scan(&targetID)
			if err != nil || forceErr == "tx.QueryRow" {
				return nil, fmt.Errorf("更新 学生答案分数 sql 失败: %v", err)
			}

			targetIDs = append(targetIDs, targetID)
		}
	}

	return targetIDs, nil
}

// UpdateExamSessionOrPracticeSubmissionState 更新考试场次或练习提交的状态
//
// 参数:
//   - ctx: 上下文，用于控制请求生命周期和传递上下文信息
//   - tx: 数据库事务对象，用于执行更新操作
//   - teacherID: 教师ID，表示执行更新操作的教师
//   - examSessionIDs: 考试场次ID列表，用于指定要更新的考试场次
//   - practiceSubmissionIDs: 练习提交ID列表，用于指定要更新的练习提交
//   - status: 目标状态值，将被设置为记录的新状态(10: 考试已提交 08: 练习已提交)
//
// 返回值:
//   - targetIDs: 成功更新的记录ID列表
//   - err: 操作过程中发生的错误信息
func UpdateExamSessionOrPracticeSubmissionState(ctx context.Context, tx pgx.Tx, teacherID int64, examSessionIDs []int64, practiceSubmissionIDs []int64, status string) ([]int64, error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	if teacherID <= 0 {
		return nil, fmt.Errorf("无效的 teacher_id")
	}

	if len(examSessionIDs) == 0 && len(practiceSubmissionIDs) == 0 {
		return nil, fmt.Errorf("无效的 exam_session_ids 和 practice_submission_ids")
	}

	if len(examSessionIDs) > 0 && len(practiceSubmissionIDs) > 0 {
		return nil, fmt.Errorf("exam_session_ids 和 practice_submission_ids 不能同时存在")
	}

	if status == "" {
		return nil, fmt.Errorf("无效的 考试/练习 状态")
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

	var (
		rows pgx.Rows
		err  error
	)

	if tx == nil {
		pgConn := cmn.GetPgxConn()
		rows, err = pgConn.Query(ctx, updateQuery, pq.Array(ids), status, teacherID, time.Now().UnixMilli())
	} else {
		rows, err = tx.Query(ctx, updateQuery, pq.Array(ids), status, teacherID, time.Now().UnixMilli())
	}

	if err != nil || forceErr == "UpdateExamSessionOrPracticeSubmissionState-tx.Query" {
		return nil, fmt.Errorf("更新 考试/练习 状态 sql 失败: %v", err)
	}
	defer rows.Close()

	targetIDs := make([]int64, 0, 10)

	for rows.Next() {
		var id int64
		err = rows.Scan(&id)
		if err != nil || forceErr == "rows.Scan" {
			return nil, fmt.Errorf("从 row 中读取结果失败: %v", err)
		}
		targetIDs = append(targetIDs, id)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("行遍历错误: %v", err)
	}

	return targetIDs, nil
}

func UpdatePracticeWrongSubmissionState(ctx context.Context, tx pgx.Tx, teacherID int64, practiceWrongSubmissionIDs []int64, status string) ([]int64, error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	if teacherID <= 0 {
		return nil, fmt.Errorf("无效的 teacher_id")
	}

	if len(practiceWrongSubmissionIDs) <= 0 {
		return nil, fmt.Errorf("无效的 practice_wrong_submission_ids")
	}

	if status == "" {
		return nil, fmt.Errorf("无效的练习状态")
	}

	updateQuery := `UPDATE t_practice_wrong_submissions
					SET
						status = $2,
						updated_by = $3, 
						update_time = $4
					WHERE id = ANY($1)
					RETURNING id`

	rows, err := tx.Query(ctx, updateQuery, pq.Array(practiceWrongSubmissionIDs), status, teacherID, time.Now().UnixMilli())
	if err != nil || forceErr == "UpdatePracticeWrongSubmissionState-tx.Query" {
		return nil, fmt.Errorf("更新错题练习 sql 失败: %v", err)
	}
	defer rows.Close()

	targetIDs := make([]int64, 0, 10)

	for rows.Next() {
		var id int64
		err = rows.Scan(&id)
		if err != nil || forceErr == "UpdatePracticeWrongSubmissionState-rows.Scan" {
			return nil, fmt.Errorf("从 row 中读取结果失败: %v", err)
		}
		targetIDs = append(targetIDs, id)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("行遍历失败: %v", err)
	}

	return targetIDs, nil
}
