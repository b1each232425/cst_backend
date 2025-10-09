package mark

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jmoiron/sqlx/types"
	"github.com/lib/pq"
	"go.uber.org/zap"

	"math"
	"time"
	"w2w.io/cmn"
	"w2w.io/null"
)

// 获取考试 管理员查询全部，教师查询与其有关的考试 // TODO 简化 // TODO 是否有综合演练
func QueryExams(ctx context.Context, req QueryMarkingListReq) ([]Exam, int, error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	if forceErr == "QueryExams" {
		return nil, -1, ForceErr
	}

	if err := cmn.Validate(req); err != nil {
		return nil, -1, fmt.Errorf("无效的 req: %v", err)
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
		rowCount            int
	)

	if req.ExamName != "" {
		rowCountWhereClause = append(rowCountWhereClause, fmt.Sprintf("ei.name ILIKE $%d", len(countArgs)+1))
		countArgs = append(countArgs, "%"+req.ExamName+"%")
	}

	if !req.StartTime.IsZero() {
		rowCountWhereClause = append(rowCountWhereClause, fmt.Sprintf("es.start_time >= $%d", len(countArgs)+1))
		countArgs = append(countArgs, req.StartTime.UnixMilli())
	}

	if !req.EndTime.IsZero() {
		rowCountWhereClause = append(rowCountWhereClause, fmt.Sprintf("es.start_time <= $%d", len(countArgs)+1))
		countArgs = append(countArgs, req.EndTime.UnixMilli())
	}

	if req.ExamSessionID > 0 {
		rowCountWhereClause = append(rowCountWhereClause, fmt.Sprintf("es.id = $%d", len(countArgs)+1))
		countArgs = append(countArgs, req.ExamSessionID)
	}

	// 普通批阅员
	if !req.User.IsAdmin { // TODO 核分员 // 只能是获取要核分的列表
		rowCountWhereClause = append(rowCountWhereClause, fmt.Sprintf("mi.mark_teacher_id = $%d", len(countArgs)+1))
		countArgs = append(countArgs, teacherID)
	}

	getExamCountQuery = fmt.Sprintf(getExamCountQuery, joinWhereClause(rowCountWhereClause))

	pgxConn := cmn.GetPgxConn()

	err := pgxConn.QueryRow(ctx, getExamCountQuery, countArgs...).Scan(&rowCount)
	if forceErr == "getExamCountQuery" {
		err = ForceErr
	}
	if err != nil {
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
						COALESCE(eec.examiniee_count, 0) AS examinee_count, 
						COALESCE(erc.respondent_count, 0) AS respondent_count, 
					--	COALESCE(eumc.unmarked_student_count, 0) AS unmarked_student_count, 
						COALESCE(mc.marked_count, -1) AS marked_student_count  -- 只有学生作答才能被批改并记录在视图mc
					FROM
						 t_exam_session es 
					JOIN t_exam_info ei ON ei.id = es.exam_id AND ei.status !='12' AND ei.status != '14' AND ei.status != '16' %s
					JOIN t_paper p ON p.id = es.paper_id 
					JOIN t_exam_paper ep ON ep.id = p.exampaper_id AND ep.status IS NOT NULL AND ep.status != '04' 
					LEFT JOIN v_exam_examinee_count eec ON eec.exam_session_id = es.id 
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
		examWhereClause []string
		whereClause     []string
		args            []interface{}
	)

	// 默认查询一行
	limit, offset := 1, 0
	if req.Limit > 0 {
		limit, offset = req.Limit, req.Offset
	}
	args = append(args, limit, offset)

	if req.ExamName != "" {
		examWhereClause = append(examWhereClause, fmt.Sprintf("ei.name ILIKE $%d", len(args)+1))
		args = append(args, "%"+req.ExamName+"%")
	}

	if !req.StartTime.IsZero() {
		whereClause = append(whereClause, fmt.Sprintf("es.start_time >= $%d", len(args)+1))
		args = append(args, req.StartTime.UnixMilli())
	}

	if !req.EndTime.IsZero() {
		whereClause = append(whereClause, fmt.Sprintf("es.start_time <= $%d", len(args)+1))
		args = append(args, req.EndTime.UnixMilli())
	}

	if req.ExamSessionID > 0 {
		whereClause = append(whereClause, fmt.Sprintf("es.id = $%d", len(args)+1))
		args = append(args, req.ExamSessionID)
	}

	// TODO 需要处理，传进来的？
	if !req.User.IsAdmin {
		// 普通批阅员，非管理员
		whereClause = append(whereClause, fmt.Sprintf("mi.mark_teacher_id = $%d AND ( mc.teacher_id = $%d OR mc.teacher_id IS NULL )", len(args)+1, len(args)+1)) // 因为考试参与人数为0的也需要计算进去
		args = append(args, teacherID)
	}

	examListQuery = fmt.Sprintf(examListQuery, joinWhereClause(examWhereClause), joinWhereClause(whereClause))

	rows, err := pgxConn.Query(ctx, examListQuery, args...)
	if forceErr == "examListQuery" {
		err = ForceErr
	}
	if err != nil {
		return nil, -1, fmt.Errorf("获取考试列表 sql 失败: %v", err)
	}
	defer rows.Close()

	// key 是 exam_id
	examsMap := make(map[int64]*Exam, req.Limit)
	examsMapKeys := make([]int64, 0, req.Limit)

	for rows.Next() {
		var (
			exam               cmn.TExamInfo
			examSession        cmn.TExamSession
			markedStudentCount null.Int
			examineeCount      null.Int // 应考人数
			respondentCount    null.Int // 参与作答的总人数
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
			&examSession.ID,
			&examSession.SessionNum,
			&examSession.StartTime,
			&examSession.EndTime,
			&examSession.Status,
			&examSession.MarkMethod,
			&examSession.MarkMode,
			&paperName,
			&markExamineeIdsJson,
			&questionIDsJson,
			&markStatus,
			&examineeCount,
			&respondentCount,
			//&totalUnmarkedStudentCount,
			&markedStudentCount,
		)
		if forceErr == "rows.Scan" {
			err = ForceErr
		}
		if err != nil {
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

		// 参加这场考试的总人数
		totalCount := respondentCount.Int64

		// 待批改数
		unMarkedStudentCount := 0

		// 如果存在一个同学作答
		if markedStudentCount.Int64 != -1 {
			switch examSession.MarkMode.String {
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

			// 需要相减得到未批改人数的原因: v_exam_teacher_marked_count 是根据 t_mark 的, 符合实际
			unMarkedStudentCount = int(math.Max(float64(totalCount-markedStudentCount.Int64), 0))
		}

		z.Info("", zap.Any("unMarkedStudentCount", unMarkedStudentCount), zap.Any("markedStudentCount", markedStudentCount.Int64))

		if examSession.ID.Valid {
			es := &ExamSession{
				Id:                   examSession.ID.Int64,
				Name:                 fmt.Sprint(examSession.SessionNum.Int64), // TODO 暂时使用场次编号作为名字
				PaperName:            paperName.String,
				ExamineeCount:        int(examineeCount.Int64),
				RespondentCount:      int(totalCount),
				UnMarkedStudentCount: unMarkedStudentCount,
				Status:               examSession.Status.String,
				MarkStatus:           markStatus.String,
				MarkMethod:           examSession.MarkMethod,
				MarkMode:             examSession.MarkMode.String,
				StartTime:            examSession.StartTime.Int64,
				EndTime:              examSession.EndTime.Int64,
			}
			e.ExamSessions = append(e.ExamSessions, *es)
		}
	}
	err = rows.Err()
	if forceErr == "rows.Err" {
		err = ForceErr
	}
	if err != nil {
		return nil, -1, fmt.Errorf("行遍历失败: %v", err)
	}

	examList := make([]Exam, 0, req.Limit)

	for _, key := range examsMapKeys {
		examList = append(examList, *examsMap[key])
	}

	return examList, rowCount, nil
}

// 获取练习列表 管理员查询全部列表，教师查询与其有关的练习列表
func QueryPractices(ctx context.Context, req QueryMarkingListReq) ([]Practice, int, error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	if forceErr == "QueryPractices" {
		return nil, -1, ForceErr
	}

	if err := cmn.Validate(req); err != nil {
		return nil, -1, fmt.Errorf("无效的 req: %v", err)
	}

	teacherID := req.User.ID

	rowCountQuery := `	SELECT COUNT(*) AS practice_count
								FROM t_practice p
								LEFT JOIN t_mark_info mi ON mi.practice_id = p.id AND mi.status != '04' 
								WHERE p.status != '04' %s -- 动态拼接`

	var (
		countArgs           []interface{}
		rowCountWhereClause []string
		rowCount            int
	)

	if req.PracticeName != "" {
		rowCountWhereClause = append(rowCountWhereClause, fmt.Sprintf("p.name ILIKE $%d", len(countArgs)+1))
		countArgs = append(countArgs, "%"+req.PracticeName+"%")
	}

	// 普通批阅员
	if !req.User.IsAdmin {
		rowCountWhereClause = append(rowCountWhereClause, fmt.Sprintf("mi.mark_teacher_id = $%d", len(countArgs)+1))
		countArgs = append(countArgs, teacherID)
	}

	rowCountQuery = fmt.Sprintf(rowCountQuery, joinWhereClause(rowCountWhereClause))

	pgxConn := cmn.GetPgxConn()
	err := pgxConn.QueryRow(ctx, rowCountQuery, countArgs...).Scan(&rowCount)
	if forceErr == "rowCountQuery" {
		err = ForceErr
	}
	if err != nil {
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

	var (
		whereClause []string
		args        []interface{}
	)

	limit, offset := 1, 0
	if req.Limit > 0 {
		limit, offset = req.Limit, req.Offset
	}
	args = append(args, limit, offset)

	if req.PracticeName != "" {
		whereClause = append(whereClause, fmt.Sprintf("p.name ILIKE $%d", len(args)+1))
		args = append(args, "%"+req.PracticeName+"%")
	}

	if !req.User.IsAdmin {
		// 普通批阅员，非管理员
		whereClause = append(whereClause, fmt.Sprintf("mi.mark_teacher_id = $%d", len(args)+1))
		args = append(args, teacherID)
	}

	listQuery = fmt.Sprintf(listQuery, joinWhereClause(whereClause))

	rows, err := pgxConn.Query(ctx, listQuery, args...)
	if forceErr == "listQuery" {
		err = ForceErr
	}
	if err != nil {
		return nil, -1, fmt.Errorf("获取所有练习数目 sql 失败: %v", err)
	}
	defer rows.Close()

	practices := make([]Practice, 0, req.Limit)

	for rows.Next() {
		var (
			p   Practice
			row cmn.TPractice
		)
		err = rows.Scan(&row.ID, &row.Name, &row.CorrectMode, &row.Type, &p.RespondentCount, &p.UnMarkedStudentCount)
		if forceErr == "rows.Scan" {
			err = ForceErr
		}
		if err != nil {
			return nil, -1, fmt.Errorf("扫描解析练习数据失败: %v", err)
		}

		p.MarkMode = row.CorrectMode.String
		p.Type = row.Type.String
		p.Name = row.Name.String
		p.ID = row.ID.Int64

		practices = append(practices, p)
	}
	err = rows.Err()
	if forceErr == "rows.Err" {
		err = ForceErr
	}
	if err != nil {
		return nil, -1, fmt.Errorf("行遍历错误: %v", err)
	}

	return practices, rowCount, nil
}

// 获取一个批改员的阅卷结果（也可以获得单个学生的, 可以是单个问题的）
// teacherID <= 0 ： 查询所有老师的批改结果
func QueryMarkingResults(ctx context.Context, questionType []string, cond QueryCondition) ([]cmn.TMark, error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	if forceErr == "QueryMarkingResults" {
		return nil, ForceErr
	}

	if len(questionType) == 0 {
		return nil, errors.New("缺少 问题类型")
	}

	getMarkingResultQuery := `	SELECT m.teacher_id, m.examinee_id, m.practice_submission_id, m.question_id, m.mark_details, m.score, m.review_comment, m.remark
								FROM t_mark m
								JOIN t_exam_paper_question q ON q.status != '04' AND q.id = m.question_id
								LEFT JOIN v_latest_pending_mark_practice mp ON mp.practice_id = m.practice_id AND mp.submission_id = m.practice_submission_id 
								WHERE q.type = ANY($1) %s -- 动态拼接where条件
								ORDER BY m.examinee_id, m.practice_submission_id, q.order`

	var (
		whereClause []string
		args        []interface{}
	)

	args = append(args, pq.Array(questionType))

	if cond.ExamSessionID > 0 {
		// 获取考试场次下的阅卷结果（提交的时候需要全部的）
		whereClause = append(whereClause, fmt.Sprintf("m.exam_session_id = $%d", len(args)+1))
		args = append(args, cond.ExamSessionID)

		if cond.ExamineeID > 0 {
			// 获取单个考生的阅卷结果
			whereClause = append(whereClause, fmt.Sprintf("m.examinee_id = $%d", len(args)+1))
			args = append(args, cond.ExamineeID)
		}

		if cond.QuestionID > 0 {
			// 获取单个问题的阅卷结果
			whereClause = append(whereClause, fmt.Sprintf("m.question_id = $%d", len(args)+1))
			args = append(args, cond.QuestionID)
		}
	} else {
		// 获取练习下的阅卷结果
		whereClause = append(whereClause, fmt.Sprintf("m.practice_id = $%d", len(args)+1))
		args = append(args, cond.PracticeID)

		if cond.PracticeSubmissionID > 0 {
			// 获取单个提交下的阅卷结果
			whereClause = append(whereClause, fmt.Sprintf("m.practice_submission_id = $%d", len(args)+1))
			args = append(args, cond.PracticeSubmissionID)
		} else {
			// 在不指定提交id的情况下保证获取的是最新的提交
			whereClause = append(whereClause, "mp.submission_id IS NOT NULL")
		}
	}

	// 约束老师
	if cond.TeacherID > 0 {
		whereClause = append(whereClause, fmt.Sprintf("m.teacher_id = $%d", len(args)+1))
		args = append(args, cond.TeacherID)
	}

	pgxConn := cmn.GetPgxConn()

	getMarkingResultQuery = fmt.Sprintf(getMarkingResultQuery, joinWhereClause(whereClause))

	rows, err := pgxConn.Query(ctx, getMarkingResultQuery, args...)
	if forceErr == "getMarkingResultQuery" {
		err = ForceErr
	}
	if err != nil {
		return nil, fmt.Errorf("获取批改结果 sql 失败: %v", err)
	}
	defer rows.Close()

	markingResults := make([]cmn.TMark, 0, 10) // 假设 10

	for rows.Next() {
		var mr cmn.TMark
		err = rows.Scan(&mr.TeacherID, &mr.ExamineeID, &mr.PracticeSubmissionID, &mr.QuestionID, &mr.MarkDetails, &mr.Score, &mr.ReviewComment, &mr.Remark)
		if forceErr == "rows.Scan" {
			err = ForceErr
		}
		if err != nil {
			return nil, fmt.Errorf("从 row 读取结果失败: %v", err)
		}

		if cond.ExamSessionID > 0 {
			mr.ExamSessionID = null.IntFrom(cond.ExamSessionID)
		} else {
			mr.PracticeID = null.IntFrom(cond.PracticeID)
		}

		markingResults = append(markingResults, mr)
	}
	err = rows.Err()
	if forceErr == "rows.Err" {
		err = ForceErr
	}
	if err != nil {
		return nil, fmt.Errorf("行遍历错误: %v", err)
	}

	return markingResults, nil
}

// 查询阅卷配置信息
func QueryMarkerInfo(ctx context.Context, cond QueryCondition) (MarkerInfo, error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	if forceErr == "QueryMarkerInfo" {
		return MarkerInfo{}, ForceErr
	}

	var (
		query       string
		queryParams []interface{}
	)

	// 构建不同的SQL查询
	if cond.ExamSessionID > 0 {
		query = `SELECT mi.mark_teacher_id, mi.mark_question_groups, 
       					mi.mark_examinee_ids, mi.question_ids,
                        es.mark_mode, es.mark_method, u.official_name
                 FROM t_mark_info mi
                 JOIN t_exam_session es ON es.id = mi.exam_session_id
                 JOIN t_user u ON u.id = mi.mark_teacher_id
                 WHERE mi.exam_session_id = $1 AND mi.status != '04'`
		queryParams = append(queryParams, cond.ExamSessionID)
	} else {
		query = `SELECT mi.mark_teacher_id, mi.mark_question_groups,
       					mi.mark_examinee_ids, mi.question_ids,
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
	if forceErr == "query" {
		err = ForceErr
	}
	if err != nil {
		return MarkerInfo{}, fmt.Errorf("获取批改配置信息 sql 失败: %v", err)
	}
	defer rows.Close()

	var (
		markerInfo MarkerInfo
		markMode   null.String
		markMethod null.String
		scanVars   []interface{}
	)

	// 循环获取 markerInfo.MarkInfos，表中一场考试的每一行的markMode、markMethod其实都是一样的
	for rows.Next() {
		var mi cmn.TMarkInfo
		var name null.String

		scanVars = []interface{}{&mi.MarkTeacherID, &mi.MarkQuestionGroups, &mi.MarkExamineeIds, &mi.QuestionIds}

		if cond.ExamSessionID > 0 { // 考试
			scanVars = append(scanVars, &markMode, &markMethod, &name)
		} else { // 练习
			scanVars = append(scanVars, &markMode)
		}

		err = rows.Scan(scanVars...)
		if forceErr == "rows.Scan" {
			err = ForceErr
		}
		if err != nil {
			return MarkerInfo{}, fmt.Errorf("从 row 读取结果失败: %v", err)
		}

		markerInfo.MarkInfos = append(markerInfo.MarkInfos, mi) // 每个老师一个值。数组：多人批改
		//if cond.ExamSessionID > 0 {
		//	markerInfo.MarkedStudentNames = append(markerInfo.MarkedStudentNames, name.String) // TODO 错误？学生的名字不是老师的名字
		//}
	}
	err = rows.Err()
	if forceErr == "rows.Err" {
		err = ForceErr
	}
	if err != nil {
		return MarkerInfo{}, fmt.Errorf("行遍历失败: %v", err)
	}

	markerInfo.MarkMode = markMode.String
	markerInfo.MarkMethod = markMethod.String

	return markerInfo, nil
}

// 查询一个学生的答案  questionType: "00": 客观题， "02"：主观题
func QueryStudentAnswers(ctx context.Context, tx pgx.Tx, questionType string, cond QueryCondition, markerInfo MarkerInfo) ([]cmn.TVStudentAnswerQuestion, error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	if forceErr == "QueryStudentAnswers" {
		return nil, ForceErr
	}

	if questionType != "00" && questionType != "02" {
		return nil, fmt.Errorf("无效的题目类型: %v", questionType)
	}

	// 考试，练习，练习错题，答案都是存储在 t_student_answers 表的，后两个都是使用 practice_submission_id 区别的， 意味着，同一时刻，同个学生的一个练习及其练习错题只存在一种
	query := `	SELECT exam_session_id, practice_id, question_id, "order", 
					type, question_score, answer, answer_score, actual_answers, 
					actual_options, examinee_id, practice_submission_id 
				FROM v_student_answer_question
				WHERE type IS NOT NULL %s -- where条件
				ORDER BY "order" ASC
			`

	var (
		whereClause []string
		args        []interface{}
	)

	if cond.ExamSessionID > 0 {
		if cond.ExamineeID <= 0 {
			return nil, errors.New("无效的 考生ID")
		}

		whereClause = append(whereClause, fmt.Sprintf("examinee_id = $%d", len(args)+1))
		args = append(args, cond.ExamineeID)
	} else {
		if cond.PracticeSubmissionID <= 0 {
			return nil, errors.New("无效的 练习提交ID")
		}

		whereClause = append(whereClause, fmt.Sprintf("practice_submission_id = $%d", len(args)+1))
		args = append(args, cond.PracticeSubmissionID)

		// 如果是错题，由于是自动批改，只需要错题，减少 token 消耗
		if cond.PracticeWrongSubmissionID > 0 {
			whereClause = append(whereClause, "question_score != answer_score ")
		}
	}

	// 单选、多选、判断题
	if questionType == "00" {
		whereClause = append(whereClause, "type IN ('00', '02', '04')")
	} else { // 填空、简答题 // TODO 综合演练
		whereClause = append(whereClause, "type IN ('06', '08')")
	}

	switch markerInfo.MarkMode {
	case "06", "08": // 题组专评、题目专评
		if len(markerInfo.MarkInfos) < 1 {
			return nil, errors.New("批改配置信息错误")
		}

		var questionIDs []int64
		err := markerInfo.MarkInfos[0].QuestionIds.Unmarshal(&questionIDs)
		if err != nil {
			return nil, fmt.Errorf("解析 markerInfo.MarkInfos[0].QuestionIds 失败: %v", err)
		}

		whereClause = append(whereClause, fmt.Sprintf("question_id = ANY($%d)", len(args)+1))
		args = append(args, pq.Array(questionIDs))
	}

	query = fmt.Sprintf(query, joinWhereClause(whereClause))

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

	if forceErr == "query" {
		err = ForceErr
	}
	if err != nil {
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
		if forceErr == "rows.Scan" {
			err = ForceErr
		}
		if err != nil {
			return nil, fmt.Errorf("从 row 中读取结果失败: %v", err)
		}

		studentAnswers = append(studentAnswers, sa)
	}
	err = rows.Err()
	if forceErr == "rows.Err" {
		err = ForceErr
	}
	if err != nil {
		return nil, fmt.Errorf("行遍历失败: %v", err)
	}

	return studentAnswers, nil
}

// 查询主观题 （可查询单道题目）
func QuerySubjectiveQuestions(ctx context.Context, tx pgx.Tx, types []string, cond QueryCondition, markerInfo MarkerInfo) ([]QuestionSet, error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)

	if len(types) == 0 {
		return nil, errors.New("缺少题目类型")
	}

	getQuestionsQuery := `	SELECT DISTINCT pg.id, pg.name, pg."order", q.id, 
							q.score, q.type, q."order", q.group_id,
							q.content, q.options, q.answers, q.analysis, q.title 
							FROM t_exam_paper p
							JOIN t_exam_paper_group pg ON pg.exam_paper_id = p.id 
							JOIN t_exam_paper_question q ON q.group_id = pg.id
							JOIN t_paper tp ON tp.exampaper_id = p.id 
							LEFT JOIN t_exam_session es ON es.paper_id = tp.id 
							LEFT JOIN t_practice pra ON pra.paper_id = tp.id
							WHERE p.status != '04' AND p.status IS NOT NULL 
							  AND q.status != '04' AND q.status IS NOT NULL 
							  AND q.type = ANY($1) 
							  %s -- 动态拼接where条件
							ORDER BY pg.id, q.id`

	var (
		whereClause []string
		args        []interface{}
		err         error
	)

	args = append(args, pq.Array(types))

	switch {
	case cond.QuestionID > 0: // 存在 questionID，直接使用这个查询即可，sql已去重，无需担心考试练习重复使用一道题目而查出重复的题目
		whereClause = append(whereClause, fmt.Sprintf("q.id = $%d", len(args)+1))
		args = append(args, cond.QuestionID)
	case cond.ExamSessionID > 0:
		whereClause = append(whereClause, fmt.Sprintf("es.id = $%d", len(args)+1))
		args = append(args, cond.ExamSessionID)

		// 考试才有
		switch markerInfo.MarkMode {
		case "06", "08": // 题组专评、题目专评
			if len(markerInfo.MarkInfos) < 1 {
				return nil, errors.New("批改配置信息错误")
			}

			var questionIDs []int64
			err = markerInfo.MarkInfos[0].QuestionIds.Unmarshal(&questionIDs)
			if err != nil {
				return nil, fmt.Errorf("解析 markerInfo.MarkInfos[0].QuestionIds 失败: %v", err)
			}

			whereClause = append(whereClause, fmt.Sprintf("q.id = ANY($%d)", len(args)+1))
			args = append(args, pq.Array(questionIDs))
		}
	case cond.PracticeID > 0:
		whereClause = append(whereClause, fmt.Sprintf("pra.id = $%d", len(args)+1))
		args = append(args, cond.PracticeID)
	}

	getQuestionsQuery = fmt.Sprintf(getQuestionsQuery, joinWhereClause(whereClause))

	var rows pgx.Rows

	if tx != nil {
		rows, err = tx.Query(ctx, getQuestionsQuery, args...)
	} else {
		pgxConn := cmn.GetPgxConn()
		rows, err = pgxConn.Query(ctx, getQuestionsQuery, args...)
	}

	if forceErr == "getQuestionsQuery" {
		err = ForceErr
	}
	if err != nil {
		return nil, fmt.Errorf("获取主观题 sql 失败: %v", err)
	}
	defer rows.Close()

	// question_set_id -> question_set
	questionSetsMap := make(map[int64]*QuestionSet, 10)

	for rows.Next() {
		var (
			qg cmn.TExamPaperGroup
			q  cmn.TExamPaperQuestion
		)

		err = rows.Scan(&qg.ID, &qg.Name, &qg.Order, &q.ID, &q.Score, &q.Type, &q.Order, &q.GroupID, &q.Content, &q.Options, &q.Answers, &q.Analysis, &q.Title)
		if forceErr == "rows.Scan" {
			err = ForceErr
		}
		if err != nil {
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
	err = rows.Err()
	if forceErr == "rows.Err" {
		err = ForceErr
	}
	if err != nil {
		return nil, fmt.Errorf("行遍历: %v", err)
	}

	questionSets := make([]QuestionSet, 0, 10)

	for _, result := range questionSetsMap {
		questionSets = append(questionSets, *result)
	}

	return questionSets, nil
}

// 查询一个老师对于一个考试场次的一道题目的已批改人数
func QueryMarkedStudentCountForQuestion(ctx context.Context, cond QueryCondition) (int64, error) {
	if cond.TeacherID <= 0 {
		return -1, errors.New("缺少 老师ID")
	}

	if cond.QuestionID <= 0 {
		return -1, errors.New("缺少 问题ID")
	}

	if cond.ExamSessionID <= 0 {
		return -1, errors.New("缺少 考试场次ID")
	}

	query := `SELECT COUNT(DISTINCT examinee_id) 
			  FROM t_mark 
			  WHERE exam_session_id = $1 AND teacher_id = $2 AND question_id = $3 
			 `

	var count int64

	pgxConn := cmn.GetPgxConn()
	err := pgxConn.QueryRow(ctx, query, cond.ExamSessionID, cond.TeacherID, cond.QuestionID).Scan(&count)
	if err != nil {
		return -1, fmt.Errorf("执行 query 失败: %v", err)
	}

	return count, nil
}

// 查询考试场次的状态
func QueryExamSessionStatus(ctx context.Context, tx pgx.Tx, examSessionID int64) (string, error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	if forceErr == "QueryExamSessionStatus" {
		return "", ForceErr
	}

	query := `SELECT status FROM t_exam_session WHERE id = $1`

	var (
		examSessionStatus string
		err               error
	)

	if tx == nil {
		pgxConn := cmn.GetPgxConn()
		err = pgxConn.QueryRow(ctx, query, examSessionID).Scan(&examSessionStatus)
	} else {
		err = tx.QueryRow(ctx, query, examSessionID).Scan(&examSessionStatus)
	}

	if forceErr == "query" {
		err = ForceErr
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("查询不到考试场次 %d : %v", examSessionID, err)
		} else {
			return "", fmt.Errorf("执行 query 失败: %v", err)
		}
	}

	return examSessionStatus, nil
}

// 分页查询考生信息（可查询单个学生，单个题目）（考试可以选择考生状态，练习不能） // TODO对于缺考的学生，直接判0分
func QueryStudents(ctx context.Context, cond QueryCondition, markerInfo MarkerInfo, examineeStatuses []string, req QueryStudentsReq) ([]StudentInfo, int, error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	if forceErr == "QueryStudents" {
		return nil, -1, ForceErr
	}

	var (
		queryCount  string // 获取总数
		query       string // 获取这页的信息
		whereClause []string
		args        []interface{}
	)

	if cond.ExamSessionID > 0 {
		queryCount = `  SELECT COUNT(DISTINCT ei.id)
						FROM v_examinee_info ei
						LEFT JOIN t_mark m ON m.examinee_id = ei.id 
						WHERE ei.exam_session_id = $1 %s -- 动态拼接where条件`

		query = `	SELECT DISTINCT ei.id, ei.official_name, ei.serial_number, ei.examinee_number, ei.id_card_no
					FROM v_examinee_info ei
					LEFT JOIN t_mark m ON m.examinee_id = ei.id 
					WHERE ei.exam_session_id = $1 %s -- 动态拼接where条件
					ORDER BY ei.serial_number, ei.examinee_number
					LIMIT $%d OFFSET $%d`

		args = append(args, cond.ExamSessionID)

		// 约束考生个数
		switch {
		case cond.ExamineeID > 0 && markerInfo.MarkMode == "04":
			return nil, -1, errors.New("考生ID 和 试卷分配模式 不能同时存在")

		// 查询单个学生
		case cond.ExamineeID > 0:
			whereClause = append(whereClause, fmt.Sprintf(" AND id = $%d ", len(args)+1))
			args = append(args, cond.ExamineeID)

		// 试卷分配（考试）
		case cond.ExamSessionID > 0 && markerInfo.MarkMode == "04":
			if len(markerInfo.MarkInfos) < 1 {
				return nil, -1, errors.New("批改配置信息错误")
			}

			var examineeIds []int64
			err := json.Unmarshal(markerInfo.MarkInfos[0].MarkExamineeIds, &examineeIds)
			if err != nil {
				return nil, -1, fmt.Errorf("解析 markerInfo.MarkInfos[0].MarkExamineeIds 失败: %v", err)
			}

			whereClause = append(whereClause, fmt.Sprintf(" AND id = ANY($%d) ", len(args)+1))
			args = append(args, pq.Array(examineeIds))
		}

		// 约束考生状态
		if len(examineeStatuses) > 0 {
			whereClause = append(whereClause, fmt.Sprintf("examinee_status = ANY($%d)", len(args)+1))
			args = append(args, pq.Array(examineeStatuses))
		}

		// 约束考生准考证号/身份证号
		if req.KeyWord != "" {
			whereClause = append(whereClause, fmt.Sprintf("ei.examinee_number ILIKE $%d AND ei.id_card_no ILIKE $%d", len(args)+1, len(args)+1))
			args = append(args, "%"+req.KeyWord+"%")
		}

		// 约束点评状态（针对综合演练）
		switch req.CommentStatus {
		// 未点评
		case "00":
			whereClause = append(whereClause, "m.question_id IS NULL") // t_mark 表在批改考试结束后才填入非综合演练的数据，点评前，t_mark表 一定没有这场考试的数据

		// 已点评
		case "02":
			if cond.QuestionID < 0 {
				return nil, -1, errors.New("缺少 问题ID")
			}

			whereClause = append(whereClause, fmt.Sprintf("m.question_id = $%d", len(args)+1))
			args = append(args, cond.QuestionID)
		}

	} else {
		queryCount = `SELECT COUNT(DISTINCT mp.submission_id)
					FROM v_latest_pending_mark_practice mp 
					JOIN t_user u ON mp.student_id = u.id 
					WHERE mp.practice_id = $1 %s -- 动态插入s`

		query = `	SELECT DISTINCT mp.submission_id, u.official_name
					FROM v_latest_pending_mark_practice mp 
					JOIN t_user u ON mp.student_id = u.id 
					WHERE mp.practice_id = $1 %s -- 动态插入s
					ORDER BY mp.submission_id
					LIMIT $%d OFFSET $%d`
		args = append(args, cond.PracticeID)

		// 查询单个学生
		if cond.PracticeSubmissionID > 0 {
			whereClause = append(whereClause, fmt.Sprintf("mp.submission_id = $%d", len(args)+1))
			args = append(args, cond.PracticeSubmissionID)
		}
	}

	filter := joinWhereClause(whereClause)

	pgxConn := cmn.GetPgxConn()

	queryCount = fmt.Sprintf(queryCount, filter)

	var count int
	err := pgxConn.QueryRow(ctx, queryCount, args...).Scan(&count) // 除去两个分页的参数
	if err != nil {
		return nil, -1, fmt.Errorf("查询学生信息总数失败: %v", err)
	}

	query = fmt.Sprintf(query, filter, len(args)+1, len(args)+2)
	args = append(args, req.Limit, req.Offset)

	rows, err := pgxConn.Query(ctx, query, args...)
	if forceErr == "query" {
		err = ForceErr
	}
	if err != nil {
		return nil, -1, fmt.Errorf("查询学生信息 sql 失败: %v", err)
	}
	defer rows.Close()

	studentInfos := make([]StudentInfo, 0, 50) // 假设每次 50 个学生考试

	for rows.Next() {
		var (
			si       StudentInfo
			scanVars []interface{}
		)

		if cond.ExamSessionID > 0 {
			scanVars = []interface{}{&si.ExamineeID, &si.OfficialName, &si.SerialNumber, &si.ExamineeNumber, &si.IDCardNo}
		} else {
			scanVars = []interface{}{&si.PracticeSubmissionID, &si.OfficialName}
		}

		err = rows.Scan(scanVars...)
		if forceErr == "rows.Scan" {
			err = ForceErr
		}
		if err != nil {
			return nil, -1, fmt.Errorf("从 row 中读取结果失败: %v", err)
		}

		studentInfos = append(studentInfos, si)
	}
	err = rows.Err()
	if forceErr == "rows.Err" {
		err = ForceErr
	}
	if err != nil {
		return nil, -1, fmt.Errorf("行遍历错误: %v", err)
	}

	return studentInfos, count, nil
}

// 更新批改配置状态
func UpdateMarkerInfoStatus(ctx context.Context, tx pgx.Tx, teacherID int64, ids []int64, mode, status string) ([]int64, error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	if forceErr == "UpdateMarkerInfoStatus" {
		return nil, ForceErr
	}

	if teacherID <= 0 {
		return nil, errors.New("无效的 teacher ID")
	}

	if ids == nil || len(ids) == 0 {
		return nil, errors.New("没有 exam_session_ids 或者 practice_ids 可以用于更新")
	}

	if status != "00" && status != "02" && status != "04" {
		return nil, errors.New("status 错误")
	}

	updateQuery := `UPDATE t_mark_info 
					SET
						status = $2,
						updated_by = $3, 
						update_time = $4
					WHERE %s 
					RETURNING id`

	var whereClause []string

	switch mode {
	case "00":
		whereClause = append(whereClause, "exam_session_id = ANY($1)")
	case "02":
		whereClause = append(whereClause, "practice_id = ANY($1)")
	default:
		return nil, fmt.Errorf("无效的模式: %s，不是练习也不是考试", mode)
	}

	updateQuery = fmt.Sprintf(updateQuery, joinWhereClause(whereClause))

	rows, err := tx.Query(ctx, updateQuery, pq.Array(ids), status, teacherID, time.Now().UnixMilli())
	if forceErr == "updateQuery" {
		err = ForceErr
	}
	if err != nil {
		return nil, fmt.Errorf("更新批改配置信息 sql 失败: %v", err)
	}
	defer rows.Close()

	targetIDs := make([]int64, 0, len(ids))

	for rows.Next() {
		var id null.Int

		err = rows.Scan(&id)
		if forceErr == "rows.Scan" {
			err = ForceErr
		}
		if err != nil {
			return nil, fmt.Errorf("从 row 中读取结果失败: %v", err)
		}

		targetIDs = append(targetIDs, id.Int64)
	}
	err = rows.Err()
	if forceErr == "rows.Err" {
		err = ForceErr
	}
	if err != nil {
		return nil, fmt.Errorf("行遍历错误: %v", err)
	}

	return targetIDs, nil
}

// 更新对学生的批改过程
func UpsertMarkingResults(ctx context.Context, tx pgx.Tx, markingResults []cmn.TMark) ([]int64, error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string) // 用于强制执行错误处理代码
	if forceErr == "UpsertMarkingResults" {
		return nil, ForceErr
	}

	// 不存在则插入，否则更新，需要索引
	upsertMarkQuery := `
						INSERT INTO t_mark 
							(examinee_id, question_id, teacher_id, exam_session_id, mark_details, score, create_time, creator, status, practice_submission_id, practice_id)
						VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
						ON CONFLICT %s -- 约束条件 
						DO UPDATE SET
							mark_details = EXCLUDED.mark_details,
							score = EXCLUDED.score,
							update_time = EXCLUDED.create_time,
							updated_by = EXCLUDED.creator
						RETURNING id
						`

	upsertExamMarkQuery := fmt.Sprintf(upsertMarkQuery, `(examinee_id, question_id, teacher_id, exam_session_id)`)
	upsertPracticeMarkQuery := fmt.Sprintf(upsertMarkQuery, `(question_id, teacher_id, practice_id, practice_submission_id)`)

	targetIDs := make([]int64, 0, len(markingResults))
	var query string

	if markingResults[0].ExamSessionID.Int64 > 0 {
		query = upsertExamMarkQuery
	} else {
		query = upsertPracticeMarkQuery
	}

	for _, markingResult := range markingResults {
		if markingResult.Status.String == "" {
			markingResult.Status = null.StringFrom("00")
		}

		var (
			id  null.Int
			err error
		)

		args := []interface{}{markingResult.ExamineeID, markingResult.QuestionID, markingResult.TeacherID, markingResult.ExamSessionID, markingResult.MarkDetails, markingResult.Score, time.Now().UnixMilli(), markingResult.Creator, markingResult.Status, markingResult.PracticeSubmissionID, markingResult.PracticeID}

		if tx != nil {
			err = tx.QueryRow(ctx, query, args...).Scan(&id)
		} else {
			pgConn := cmn.GetPgxConn()
			err = pgConn.QueryRow(ctx, query, args...).Scan(&id)
		}

		if forceErr == "query" {
			err = ForceErr
		}
		if err != nil {
			return nil, fmt.Errorf("记录批改过程 sql 失败: %v", err)
		}
		targetIDs = append(targetIDs, id.Int64)
	}

	return targetIDs, nil
}

// 查询当前老师需要批改的考试的已批改人数和已批改总题数
// 只获取主观题（不包括综合演练）
// 练习不需要teacherID，因为是单人批改（练习需要提交才能“未提交人数-1”）
// 只读事务？
func QueryMarkedPersonAndQuestionCount(ctx context.Context, tx pgx.Tx, cond QueryCondition) (int64, int64, error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string) // 用于强制执行错误处理代码
	if forceErr == "QueryMarkedPersonAndQuestionCount" {
		return -1, -1, ForceErr
	}

	var (
		queryMarkedPerson  string
		queryQuestionCount string
		args               []interface{}
	)

	if cond.ExamSessionID > 0 {
		queryMarkedPerson = `SELECT etmc.marked_count 
			     		     FROM v_exam_teacher_marked_count etmc 
				             WHERE etmc.teacher_id = $1 AND etmc.exam_session_id = $2`

		queryQuestionCount = `SELECT COUNT(*) 
							  FROM t_mark m
							  JOIN t_exam_paper_question q ON m.question_id = q.id
							  WHERE teacher_id = $1 AND exam_session_id = $2 AND q.type IN ('06', '08', '10')`

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

	if forceErr == "queryMarkedPerson" {
		err = ForceErr
	}
	if err != nil {
		return -1, -1, fmt.Errorf("获取已批改人数 sql 失败: %v", err)
	}

	if tx == nil {
		pgxConn := cmn.GetPgxConn()
		err = pgxConn.QueryRow(ctx, queryQuestionCount, args...).Scan(&markedQuestionCount)
	} else {
		err = tx.QueryRow(ctx, queryQuestionCount, args...).Scan(&markedQuestionCount)
	}

	if forceErr == "queryQuestionCount" {
		err = ForceErr
	}
	if err != nil {
		return -1, -1, fmt.Errorf("获取已批改问题数 sql 失败: %v", err)
	}

	return markedPerson, markedQuestionCount, nil
}

// 保存学生问题最终的得分
func UpdateStudentAnswerScore(ctx context.Context, tx pgx.Tx, markingResults []cmn.TMark, cond QueryCondition) ([]int64, error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	if forceErr == "UpdateStudentAnswerScore" {
		return nil, ForceErr
	}

	if len(markingResults) == 0 {
		return nil, errors.New("没有批改结果可用于更新")
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
		whereClause = append(whereClause, fmt.Sprintf("examinee_id = $%d", argIdx))
	} else {
		whereClause = append(whereClause, fmt.Sprintf("practice_submission_id = $%d", argIdx))
	}
	argIdx++

	updateQuery = fmt.Sprintf(updateQuery, joinWhereClause(whereClause))

	// studentID --> questionID --> 不同老师分数 用以计算平均分
	studentQuestionScoresMap := make(map[int64]map[int64][]float64, 50) // 假设50个人

	// 构造 studentQuestionScoresMap
	for _, mr := range markingResults {
		var currentStudentID int64

		if cond.ExamSessionID > 0 {
			if mr.ExamineeID.Int64 <= 0 {
				return nil, fmt.Errorf("缺少 考生ID")
			}

			currentStudentID = mr.ExamineeID.Int64
		} else {
			if mr.PracticeSubmissionID.Int64 <= 0 {
				return nil, fmt.Errorf("缺少 练习提交ID")
			}

			currentStudentID = mr.PracticeSubmissionID.Int64
		}

		currentQuestionID := mr.QuestionID.Int64

		if _, ok := studentQuestionScoresMap[currentStudentID]; !ok {
			studentQuestionScoresMap[currentStudentID] = make(map[int64][]float64, 50) // 50道题目
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

	// 计算平均分，写入 t_student_answer
	for studentID, questionScores := range studentQuestionScoresMap {
		for questionID, scores := range questionScores {
			var finalScore float64
			for _, score := range scores {
				finalScore += score
			}
			finalScore = math.Round(finalScore/float64(len(scores))*10) / 10 // 四舍五入保留一位小数

			var targetID int64

			err = tx.QueryRow(ctx, updateQuery,
				finalScore,
				time.Now().UnixMilli(),
				cond.TeacherID,
				questionID,
				studentID,
			).Scan(&targetID)
			if forceErr == "updateQuery" {
				err = ForceErr
			}
			if err != nil {
				return nil, fmt.Errorf("更新 学生答案分数 sql 失败: %v", err)
			}

			targetIDs = append(targetIDs, targetID)
		}
	}

	return targetIDs, nil
}

// 更新 考试/练习提交/练习错题 的状态
func UpdateExamSessionOrPracticeSubmissionStatus(ctx context.Context, tx pgx.Tx, cond QueryCondition, status string) (int64, error) {
	forceErr, _ := ctx.Value(ForceErrKey).(string)
	if forceErr == "UpdateExamSessionOrPracticeSubmissionStatus" {
		return -1, ForceErr
	}

	// TODO 加强
	if status == "" {
		return -1, errors.New("无效的 考试/练习 状态")
	}

	updateQuery := `UPDATE %s -- 拼接表名 
					SET
						status = $2,
						updated_by = $3, 
						update_time = $4
					WHERE id = $1
					RETURNING id`

	var (
		tableName string
		id        int64
	)

	switch {
	case cond.PracticeWrongSubmissionID > 0:
		tableName = "t_practice_wrong_submissions"
		id = cond.PracticeWrongSubmissionID

	case cond.ExamSessionID > 0:
		tableName = "t_exam_session"
		id = cond.ExamSessionID

	case cond.PracticeSubmissionID > 0:
		tableName = "t_practice_submissions"
		id = cond.PracticeSubmissionID

	default:
		return -1, errors.New("缺少 考试场次ID/练习提交ID/练习错题提交ID")
	}

	updateQuery = fmt.Sprintf(updateQuery, tableName)
	args := []interface{}{id, status, cond.TeacherID, time.Now().UnixMilli()}

	var (
		targetID int64
		err      error
	)

	if tx == nil {
		pgConn := cmn.GetPgxConn()
		err = pgConn.QueryRow(ctx, updateQuery, args...).Scan(&targetID)
	} else {
		err = tx.QueryRow(ctx, updateQuery, args...).Scan(&targetID)
	}

	if forceErr == "updateQuery" {
		err = ForceErr
	}
	if err != nil {
		return -1, fmt.Errorf("更新 考试/练习 状态 sql 失败: %v", err)
	}

	return targetID, nil
}
