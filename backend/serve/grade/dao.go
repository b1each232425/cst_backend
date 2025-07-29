package grade

import (
	"context"
	"fmt"
	"strings"
	"time"

	// "gopkg.in/guregu/null.v4"

	"github.com/jackc/pgx/v5"
	"w2w.io/cmn"
)

/*
* GetExamSessionCount
 */
func GetExamSessionCount(examID []int) (int, error) {

	var err error

	z.Info("----->" + cmn.FncName())

	conn := cmn.GetDbConn()
	if conn == nil {
		z.Error("PostgreSQL connection pool is nil")
		return 0, ErrNilDBConn
	}

	placeholders := make([]string, len(examID))
	params := []any{}
	for i, id := range examID {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		params = append(params, id)
	}
	placeholderStr := strings.Join(placeholders, ", ")

	var count int

	sql := fmt.Sprintf(`SELECT COUNT(id) FROM t_exam_session WHERE exam_id IN (%s)`, placeholderStr)

	err = conn.QueryRow(sql, params...).Scan(&count)
	if err != nil {
		err = fmt.Errorf("scan exam session count(examID:%v) occurred error: %s", examID, err.Error())
		z.Error(err.Error())
		return 0, err
	}

	return count, nil
}

func GetExamSessionInfo(ctx context.Context, examID int, queryArgs ...string) ([]cmn.TExamSession, error) {

	z.Info("---->" + cmn.FncName())

	result := []cmn.TExamSession{}

	if examID < 0 {
		z.Error(ErrInvalidID.Error())
		return result, ErrInvalidID
	}

	conn := cmn.GetPgxConn()
	if conn == nil {
		z.Error("PostgreSQL connection pool is nil")
		return result, ErrNilDBConn
	}

	sql := `
			SELECT
				es.id, es.exam_id, es.paper_id, es.mark_method, es.period_mode,
				es.start_time, es.end_time, es.duration, es.question_shuffled_mode,
				es.mark_mode, es.name_visibility_in, es.session_num, es.late_entry_time,
				es.early_submission_time, es.reviewer_ids
			FROM t_exam_session es
			WHERE exam_id=$1 AND status = '06'
			ORDER BY start_time ASC`

	var es_rows pgx.Rows
	es_rows, err := conn.Query(context.Background(), sql, examID)
	if err != nil {
		z.Error(err.Error())
		return nil, err
	}
	defer es_rows.Close()
	for es_rows.Next() {
		var es cmn.TExamSession
		err := es_rows.Scan(
			&es.ID,
			&es.ExamID,
			&es.PaperID,
			&es.MarkMethod,
			&es.PeriodMode,
			&es.StartTime,
			&es.EndTime,
			&es.Duration,
			&es.QuestionShuffledMode,
			&es.MarkMode,
			&es.NameVisibilityIn,
			&es.SessionNum,
			&es.LateEntryTime,
			&es.EarlySubmissionTime,
			&es.ReviewerIds,
			&es.PaperName,
			&es.PaperCategory,
		)
		if err != nil {
			z.Error(err.Error())
			return nil, err
		}
		result = append(result, es)
	}

	return result, nil
}

/*
* GetRowCount 用于获取指定 SQL 查询结果的行数。
* 该函数构建一个嵌套查询，通过 COUNT(1) 统计传入查询的结果行数。
* 参数 ctx 是上下文，用于控制请求的生命周期。
* 参数 query 是包含 SQL 语句和查询参数的结构体指针。
* 返回值为查询结果的行数和可能出现的错误。
 */
func GetRowCount(ctx context.Context, sql string, params []any) (int64, error) {

	z.Info("---->" + cmn.FncName())

	var err error

	var result int64

	if sql == "" {
		err = fmt.Errorf("%w: query SQL is empty, please check if the query was initialized correctly", ErrEmptyQuery)
		z.Error(err.Error())
		return result, err
	}

	countSql := fmt.Sprintf(`SELECT COUNT(1) FROM (%s) AS exam_grade_list_count`, sql)

	conn := cmn.GetPgxConn()
	if conn == nil {
		err = fmt.Errorf("get exam grade failed: %w", ErrNilDBConn)
		z.Error(err.Error())
		return result, err
	}

	err = conn.QueryRow(ctx, countSql, params...).Scan(&result)
	if err != nil {
		err = fmt.Errorf("get exam grade row count error: %w", err)
		z.Error(err.Error())
		return result, err
	}

	return result, err
}

func GetManualPaperDetailByPaperID(ctx context.Context, paperID int64) (*cmn.TVPaper, error) {
	if paperID <= 0 {
		z.Error(ErrInvalidPaperID.Error())
		return nil, ErrInvalidPaperID
	}

	query := `SELECT 
    id,name,assembly_type,category,level,suggested_duration,description,tags,creator,create_time,update_time,status,total_score,question_count,groups_data
	FROM v_paper
	WHERE id = $1
	LIMIT 1`

	db := cmn.GetDbConn()
	row := db.QueryRowContext(ctx, query, paperID)
	var paper cmn.TVPaper
	err := row.Scan(
		&paper.ID,
		&paper.Name,
		&paper.AssemblyType,
		&paper.Category,
		&paper.Level,
		&paper.SuggestedDuration,
		&paper.Description,
		&paper.Tags,
		&paper.Creator,
		&paper.CreateTime,
		&paper.UpdateTime,
		&paper.Status,
		&paper.TotalScore,
		&paper.QuestionCount,
		&paper.GroupsData,
	)
	if err != nil {
		z.Error(err.Error())
		return nil, err
	}
	return &paper, nil
}

func GradeListExam(ctx context.Context, args *GradeListArgs) ([]GradeExam, int64, error) {
	var err error
	var page int
	var pageSize int
	var result []GradeExam
	var rowCount int64

	z.Info("---->" + cmn.FncName())

	// 分页参数: Page PageSize
	// 数据库查询必需参数: TeacherID ExamID PassScoreRate
	// 可选参数: Name Type Submitted

	if args.Page <= 0 {
		err = fmt.Errorf("%w: 页码必须为正整数", ErrInvalidPage)
		z.Error(err.Error())
		return nil, 0, err
	}
	page = args.Page

	if args.PageSize <= 0 {
		err = fmt.Errorf("%w: 每页数量必须为正整数", ErrInvalidPageSize)
		z.Error(err.Error())
		return nil, 0, err
	}
	pageSize = args.PageSize

	if args.TeacherID <= 0 && args.TeacherID != -1 {
		err = fmt.Errorf("%w: 教师ID必须为正整数或-1(表示所有教师)", ErrInvalidID)
		z.Error(err.Error())
		return nil, 0, err
	}

	if args.ExamID < 0 {
		err = fmt.Errorf("%w: 考试ID必须为正整数", ErrInvalidID)
		z.Error(err.Error())
		return nil, 0, err
	}

	if args.PassScoreRate < 0 || args.PassScoreRate > 1 {
		err = fmt.Errorf("%w: 及格率必须在0到1之间", ErrInvalidRate)
		z.Error(err.Error())
		return nil, 0, err
	}

	// 查询条件
	whereClause := " WHERE 1=1 "

	// 参数
	params := []any{}

	// 筛选教师：-1 表示所有教师
	if args.TeacherID > 0 {
		whereClause += fmt.Sprintf(" AND ei.creator=$%d ", len(params)+1)
		params = append(params, args.TeacherID)
	}

	// 筛选考试：0 表示所有考试
	if args.ExamID > 0 {
		whereClause += fmt.Sprintf(" AND ei.id=$%d ", len(params)+1)
		params = append(params, args.ExamID)
	}

	filter := args.Filter

	if filter.Name != "" {
		whereClause += fmt.Sprintf(" AND ei.name::text ILIKE $%d ", len(params)+1)
		params = append(params, fmt.Sprintf("%%%s%%", filter.Name))
	}

	if filter.Type != "" {
		whereClause += fmt.Sprintf(" AND ei.type = $%d ", len(params)+1)
		params = append(params, filter.Type)
	}

	if filter.Submitted != -1 {
		whereClause += fmt.Sprintf(" AND ei.submitted = $%d ", len(params)+1)
		params = append(params, filter.Submitted == 1)
	}

	// // 查询SQL
	// sql := fmt.Sprintf(`WITH pass_students AS (
	// 	SELECT
	// 		ets.exam_session_id,
	// 		COUNT(DISTINCT ets.student_id) AS pass_num,
	// 		vep.total_score
	// 	FROM v_student_exam_total_score ets                                                           -- 展示学生考试成绩视图：考试场次ID 学生ID
	// 		JOIN v_exam_paper vep ON vep.exam_session_id = ets.exam_session_id                        -- 该考试场次的成绩 v_exam_paper
	// 	WHERE ets.total_score >= %f * vep.total_score
	// 	GROUP BY
	// 		ets.exam_session_id,
	// 		vep.id,
	// 		vep.total_score
	// ),
	// examinees AS (
	// 	SELECT
	// 		exam_session_id,                                                                          -- 考试场次ID
	// 		COUNT(DISTINCT student_id) AS scheduled_examinees,                                        -- 计划参加考试的考生人数
	// 		COUNT(
	// 			CASE
	// 				WHEN status != '02' AND student_id IS NOT NULL THEN 1
	// 				ELSE NULL::integer
	// 			END) AS actual_examinees                                                              -- 实际参加考试的考生人数
	// 	FROM t_examinee                                                                               -- 从考生表：考试场次ID 计划参加考试的考生人数 实际参加考试的考生人数
	// 	WHERE status <> '08'
	// 	GROUP BY
	// 		exam_session_id
	// 	ORDER BY
	// 		exam_session_id DESC
	// ),
	// exam_session_info AS (
	// 	SELECT
	// 		ets.exam_id,
	// 		ets.exam_session_id,
	// 		AVG(ets.total_score) AS average_score,
	// 		es.start_time AS start_time,
	// 		es.end_time AS end_time,
	// 		es.mark_mode AS mark_mode,
	// 		vep.id AS exam_paper_id,
	// 		vep.name AS paper_name,
	// 		vep.total_score AS total_score,
	// 		ps.pass_num AS pass_examinees,
	// 		ee.actual_examinees,
	// 		ee.scheduled_examinees
	// 	FROM v_student_exam_total_score ets                                                     -- 展示学生考试成绩视图：考试场次ID 学生ID 考试成绩
	// 		LEFT JOIN t_exam_session es ON es.id = ets.exam_session_id                          -- 考试场次表：开始时间 结束时间 批改模式
	// 		LEFT JOIN v_exam_paper vep ON vep.exam_session_id = ets.exam_session_id             -- 考试考卷表：试卷ID 试卷名称 试卷总分
	// 		LEFT JOIN pass_students ps ON ps.exam_session_id = ets.exam_session_id              -- 该考试场次的及格人数
	// 		JOIN examinees ee ON ee.exam_session_id = ets.exam_session_id                       -- 该考试场次的计划参加考试的考生人数 实际参加考试的考生人数
	// 	GROUP BY
	// 		ets.exam_id,
	// 		ets.exam_session_id,
	// 		es.id,
	// 		vep.id,
	// 		vep.name,
	// 		vep.total_score,
	// 		ps.pass_num,
	// 		ee.actual_examinees,
	// 		ee.scheduled_examinees
	// 	ORDER BY
	// 		es.start_time DESC
	// )
	// SELECT
	// 	ei.id AS id,
	// 	ei.name AS exam_name,
	// 	ei.type AS exam_type,
	// 	jsonb_agg(exam_session_info) AS exam_session_info,
	// 	ei.submitted AS submitted
	// FROM t_exam_info ei                                                                -- 考试信息表：考试ID 考试名称 考试类型 提交状态 examInfoTableName
	// 	JOIN exam_session_info ON exam_session_info.exam_id = ei.id                    -- exam_session_info
	// %s
	// GROUP BY
	// 	ei.id
	// `, args.PassScoreRate, whereClause)

	// 视图SQL
	sql := fmt.Sprintf(`
	SELECT 
		ei.id AS id,
		ei.name AS exam_name,
		ei.type AS exam_type,
		jsonb_agg(esi) AS exam_session_info,
		ei.submitted AS submitted
	FROM t_exam_info ei     
	    LEFT JOIN v_grade_exam_session_info esi ON esi.exam_id = ei.id
	%s
	GROUP BY ei.id
	`, whereClause)

	// 分页SQL
	listSQL := fmt.Sprintf(`%s
	ORDER BY ei.id DESC
	LIMIT $%d OFFSET $%d`,
		sql, len(params)+1, len(params)+2)
	listParams := append(params, pageSize, (page-1)*pageSize)

	// 执行查询
	conn := cmn.GetPgxConn()
	if conn == nil {
		err = fmt.Errorf("获取考试成绩列表查询数据库连接失败: %w", ErrNilDBConn)
		z.Error(err.Error())
		return result, rowCount, err
	}

	rows, err := conn.Query(ctx, listSQL, listParams...)
	if err != nil {
		err = fmt.Errorf("执行获取考试成绩列表分页查询失败: %w", err)
		z.Error(err.Error())
		return result, rowCount, err
	}

	// 解析结果
	defer rows.Close()
	for rows.Next() {
		var grade GradeExam
		err = rows.Scan(
			&grade.ID,
			&grade.Name,
			&grade.Type,
			&grade.Sessions,
			&grade.Submitted,
		)
		z.Sugar().Debug("grade: %#v", grade)
		if err != nil {
			err = fmt.Errorf("扫描考试成绩列表失败: %w", err)
			z.Error(err.Error())
			return result, rowCount, err
		}
		result = append(result, grade)
	}

	// 统计总数
	rowCount, err = GetRowCount(ctx, sql, params)
	if err != nil {
		z.Error(err.Error())
		return nil, 0, err
	}

	return result, rowCount, nil
}

func GradeListPractice(ctx context.Context, args *GradeListArgs) ([]GradePractice, int64, error) {
	var err error
	var page int
	var pageSize int
	var result []GradePractice
	var rowCount int64

	z.Info("---->" + cmn.FncName())

	// 分页参数: Page PageSize
	// 数据库查询必需参数: TeacherID PracticeID PassScoreRate
	// 可选参数: Name

	if args.Page <= 0 {
		err = fmt.Errorf("%w: 页码必须为正整数", ErrInvalidPage)
		z.Error(err.Error())
		return nil, 0, err
	}
	page = args.Page

	if args.PageSize <= 0 {
		err = fmt.Errorf("%w: 每页数量必须为正整数", ErrInvalidPageSize)
		z.Error(err.Error())
		return nil, 0, err
	}
	pageSize = args.PageSize

	if args.TeacherID <= 0 && args.TeacherID != -1 {
		err = fmt.Errorf("%w: 教师ID必须为正整数或-1(表示所有教师)", ErrInvalidID)
		z.Error(err.Error())
		return nil, 0, err
	}

	if args.PracticeID < 0 {
		err = fmt.Errorf("%w: 练习ID必须为正整数", ErrInvalidID)
		z.Error(err.Error())
		return nil, 0, err
	}

	if args.PassScoreRate < 0 || args.PassScoreRate > 1 {
		err = fmt.Errorf("%w: 及格率必须在0到1之间", ErrInvalidRate)
		z.Error(err.Error())
		return nil, 0, err
	}

	whereClause := " WHERE 1=1 "

	params := []any{}

	if args.TeacherID > 0 {
		whereClause += fmt.Sprintf(" AND p.creator=$%d ", len(params)+1)
		params = append(params, args.TeacherID)
	}

	if args.PracticeID > 0 {
		whereClause += fmt.Sprintf(" AND p.practice_id=$%d ", len(params)+1)
		params = append(params, args.PracticeID)
	}

	filter := args.Filter

	if filter.Name != "" {
		whereClause += fmt.Sprintf(" AND p.practice_name::text ILIKE $%d ", len(params)+1)
		params = append(params, fmt.Sprintf("%%%s%%", filter.Name))
	}

	// // 查询SQL
	// sql := fmt.Sprintf(`
	// WITH pass_students AS (                                            -- pass_num 及格人数
	// 	SELECT
	// 		pts.practice_id,
	// 		COUNT(DISTINCT pts.student_id) AS pass_num
	// 	FROM v_student_practice_total_score pts                                 -- 展示学生成绩：练习ID 通过人数
	// 		LEFT JOIN v_exam_paper vep ON vep.practice_id = pts.practice_id     -- 该练习的总分
	// 	WHERE pts.total_score >= %f * vep.total_score::integer
	// 	GROUP BY
	// 		pts.practice_id,
	// 		vep.total_score
	// )
	// SELECT
	// 	p.id AS practice_id,
	// 	p.name AS practice_name,
	// 	vep.total_score AS total_score,
	// 	AVG(pts.total_score) AS averge_score,
	// 	COUNT(DISTINCT pts.student_id) AS actual_completer,
	// 	ps.pass_num AS pass_student
	// FROM t_practice p                                                            -- 展示练习列表：练习ID 名字
	// 	LEFT JOIN v_student_practice_total_score pts ON pts.practice_id = p.id   -- 该练习的平均成绩、完成人数
	// 	LEFT JOIN v_exam_paper vep ON vep.practice_id = p.id                     -- 该练习的总分
	// 	LEFT JOIN pass_students ps ON ps.practice_id = p.id                      -- 该练习的通过人数
	// %s
	// GROUP BY
	// 	p.id,
	// 	p.name,
	// 	vep.total_score,
	// 	ps.pass_num`,
	// 	args.PassScoreRate, whereClause)

	// 视图查询SQL
	sql := fmt.Sprintf(`
	SELECT
		p.practice_id,
		p.practice_name,
		p.total_score,
		p.averge_score,
		p.actual_completer,
		p.pass_student
	FROM v_grade_practice_statistics p                                                            -- 展示练习列表：练习ID 名字
	%s
	GROUP BY
		p.practice_id, p.practice_name, p.total_score, p.averge_score, p.actual_completer, p.pass_student
		`, whereClause)

	// 分页SQL
	listSQL := fmt.Sprintf(`%s
	ORDER BY p.practice_id DESC
	LIMIT $%d OFFSET $%d`,
		sql, len(params)+1, len(params)+2)
	listParams := append(params, pageSize, (page-1)*pageSize)

	// 分页查询
	conn := cmn.GetPgxConn()
	if conn == nil {
		err = fmt.Errorf("查询练习成绩列表获取数据库连接失败: %w", ErrNilDBConn)
		z.Error(err.Error())
		return result, rowCount, err
	}
	rows, err := conn.Query(ctx, listSQL, listParams...)
	if err != nil {
		err = fmt.Errorf("执行练习成绩列表分页查询出现错误: %w", err)
		z.Error(err.Error())
		return result, rowCount, err
	}

	defer rows.Close()
	// 查询结果
	for rows.Next() {
		var grade GradePractice

		err := rows.Scan(
			&grade.ID,
			&grade.Name,
			&grade.TotalScore,
			&grade.AverageScore,
			&grade.CompletedStudents,
			&grade.PassedStudents,
		)
		if err != nil {
			err = fmt.Errorf("扫描练习成绩列表出现错误: %s", err.Error())
			z.Error(err.Error())
			return result, rowCount, err
		}

		result = append(result, grade)
	}

	rowCount, err = GetRowCount(ctx, sql, params)
	if err != nil {
		z.Error(err.Error())
		return nil, 0, err
	}

	return result, rowCount, nil
}

func SetExamGradeSubmitted(ctx context.Context, args *GradeSubmitArgs) error {
	var err error

	z.Info("---->" + cmn.FncName())

	if args.TeacherID <= 0 {
		err := fmt.Errorf("%w: 教师ID无效，必须为正整数", ErrInvalidID)
		z.Error(err.Error())
		return err
	}
	teacherID := args.TeacherID

	if len(args.ExamIDs) <= 0 {
		err = fmt.Errorf("%w: examID is invalid, expected a positive number", ErrInvalidID)
		z.Error(err.Error())
		return err
	}
	examIDs := args.ExamIDs

	conn := cmn.GetPgxConn()
	if conn == nil {
		err = fmt.Errorf("查询练习成绩列表获取数据库连接失败: %w", ErrNilDBConn)
		z.Error(err.Error())
		return err
	}

	var tx pgx.Tx
	tx, err = conn.Begin(context.Background())
	if err != nil {
		err = fmt.Errorf("begin transaction error: %w", err)
		z.Error(err.Error())
		return err
	}

	// 标记事务是否成功，默认失败
	txSuccess := false
	defer func() {
		if !txSuccess {
			rollbackErr := tx.Rollback(context.Background())
			if rollbackErr != nil {
				z.Sugar().Error(fmt.Errorf("rollback transaction error: %w", rollbackErr))
			}
		}
	}()

	for _, examID := range examIDs {
		examSessions, err := GetExamSessionInfo(ctx, examID, "id", "start_time", "end_time")
		if err != nil {
			err = fmt.Errorf("get exam session info(examID=%v) error: %w", examID, err)
			z.Error(err.Error())
			return err
		}

		currTime := time.Now()

		for _, examSession := range examSessions {
			// 检查 EndTime 是否有效
			if examSession.EndTime.Valid {
				// 毫秒级时间戳
				endTime := time.UnixMilli(examSession.EndTime.Int64)
				if endTime.Before(currTime) {
					continue
				}
			} else {
				// 如果 EndTime 无效，则认为考试未结束
				err = fmt.Errorf("submit exam grade error: %w: exam session(examSession=%v) has no valid end time", ErrExamIsNotOver, examSession.ID.Int64)
				z.Error(err.Error())
				return err
			}

			err = fmt.Errorf("submit exam grade error: %w: exam session(examSession=%v) has not ended yet", ErrExamIsNotOver, examSession.ID.Int64)
			z.Error(err.Error())
			return err
		}

		sql := `UPDATE t_exam_info SET submitted=true, updated_by=$2, update_time=$3 WHERE id=$1`

		// 获取当前时间戳，这里以毫秒级为例
		curr := currTime.UnixMilli()

		z.Sugar().Debug("set exam grade submitted(examID=%v teacherID=%v) currTime=%v", examID, teacherID, curr)
		commandTag, err := tx.Exec(ctx, sql, examID, teacherID, curr)
		if err != nil {
			err = fmt.Errorf("set exam grade submitted(examID=%v teacherID=%v) error: %s", examID, teacherID, err.Error())
			z.Error(err.Error())
			return err
		}
		// 获取受影响的行数
		rowsAffected := commandTag.RowsAffected()
		z.Sugar().Debug("set exam grade submitted(examID=%v) affected rows: %d", examID, rowsAffected)
	}

	// 提交事务
	err = tx.Commit(context.Background())
	if err != nil {
		err = fmt.Errorf("commit transaction error: %w", err)
		z.Error(err.Error())
		return err
	}
	txSuccess = true

	return nil
}

// func GradeDistributionExam(ctx context.Context, args GradeDistributionExamArgs) (ExamGradeDistribution, error) {

// 	z.Info("---->" + cmn.FncName())

// 	var err error

// 	result := ExamGradeDistribution{}

// 	if args.ExamID < 0 {
// 		err = fmt.Errorf("%w: examID is invalid, expected a positive number", ErrInvalidID)
// 		z.Error(err.Error())
// 		return result, err
// 	}

// 	if args.ColumnNum < 1 {
// 		err = fmt.Errorf("%w: columnNum is invalid, expected a positive number", ErrInvalidArg)
// 		z.Error(err.Error())
// 		return result, err
// 	}

// 	examID := args.ExamID

// 	conn := cmn.GetPgxConn()
// 	if conn == nil {
// 		err = fmt.Errorf("get exam grade failed: %w", ErrNilDBConn)
// 		z.Error(err.Error())
// 		return result, err
// 	}

// 	whereClause := " WHERE exam_infos.id = $1 "

// 	distributions := []string{}

// 	scoreInterval := 1 / float64(args.ColumnNum)

// 	initAddiCondition := ""

// 	if args.ColumnNum == 1 {
// 		initAddiCondition = " OR v_stu_score.total_score IS NULL "
// 	}

// 	initDistribution := fmt.Sprintf(
// 		`SUM (
// 			CASE
// 				WHEN (v_stu_score.total_score >= %f * exam_papers.total_score
// 					AND v_stu_score.total_score <= %d * exam_papers.total_score)
// 					%s
// 					THEN 1
// 				ELSE 0
// 			END
// 		)`, scoreInterval*float64(args.ColumnNum-1), 1, initAddiCondition)

// 	distributions = append(distributions, initDistribution)

// 	for i := args.ColumnNum - 1; i > 0; i-- {

// 		addiCondition := ""

// 		if i == 1 {
// 			addiCondition = " OR v_stu_score.total_score IS NULL "
// 		}

// 		sqlStr := fmt.Sprintf(
// 			`SUM (
// 				CASE
// 					WHEN (v_stu_score.total_score >= %f * exam_papers.total_score
// 						AND v_stu_score.total_score < %f * exam_papers.total_score)
// 						%s
// 						THEN 1
// 					ELSE 0
// 				END
// 			)`, scoreInterval*float64(i-1), scoreInterval*float64(i), addiCondition)

// 		distributions = append(distributions, sqlStr)

// 	}

// 	sql := fmt.Sprintf(`WITH exam_session_grades AS (
// 		SELECT
// 			v_stu_score.exam_id AS exam_id,
// 			v_stu_score.exam_session_id,
// 			exam_papers.id AS exam_paper_id,
// 			exam_papers.name AS exam_paper_name,
// 			exam_papers.total_score AS exam_paper_total_score,
// 			COUNT(DISTINCT v_stu_score.student_id) AS total,
// 			ARRAY[
// 				%s
// 			] AS score_distribution

// 		FROM v_student_exam_total_score v_stu_score
// 			JOIN t_exam_paper exam_papers ON exam_papers.exam_session_id =  v_stu_score.exam_session_id

// 		GROUP BY
// 			v_stu_score.exam_id,
// 			v_stu_score.exam_session_id,
// 			exam_papers.id,
// 			exam_papers.name

// 	)
// 	SELECT
// 		exam_infos.id,
// 		exam_infos.name,
// 		jsonb_agg(exam_session_grades) AS exam_session_score_distribution
// 	FROM t_exam_info exam_infos
// 		JOIN exam_session_grades ON exam_session_grades.exam_id = exam_infos.id
// 	%s
// 	GROUP BY
// 		exam_infos.id
// 	`, strings.Join(distributions, ", "), whereClause)

// 	err = conn.QueryRow(ctx, sql, examID).Scan(
// 		&result.ExamID,
// 		&result.ExamName,
// 		&result.GradeDistribution,
// 	)
// 	if err != nil {
// 		// if errors.Is(err, pgx.ErrNoRows) {
// 		// 	err = fmt.Errorf("%w: no exam found with ID %d", err, examID)
// 		// 	z.Error(err.Error())
// 		// 	return result, err
// 		// }

// 		err = fmt.Errorf("get exam grade distribution(examID=%d) error: %s", examID, err.Error())
// 		z.Error(err.Error())
// 		return result, err
// 	}

// 	return result, err
// }

// // GetPracticeGradeDistribution 将返回指定练习ID的成绩分布信息
// func GradeDistributionPractice(ctx context.Context, args GradeDistributionPracticeArgs) (PracticeGradeDistribution, error) {

// 	z.Info("---->" + cmn.FncName())

// 	var err error

// 	result := PracticeGradeDistribution{}

// 	if args.PracticeID <= 0 {
// 		err = fmt.Errorf("%w: practiceID is invalid, expected a positive number", ErrInvalidID)
// 		z.Error(err.Error())
// 		return result, err
// 	}

// 	if args.ColumnNum < 1 {
// 		err = fmt.Errorf("%w: columnNum is invalid, expected a positive number", ErrInvalidArg)
// 		z.Error(err.Error())
// 		return result, err
// 	}

// 	practiceID := args.PracticeID

// 	conn := cmn.GetPgxConn()
// 	if conn == nil {
// 		err = fmt.Errorf("get exam grade failed: %w", ErrNilDBConn)
// 		z.Error(err.Error())
// 		return result, err
// 	}

// 	whereClause := " WHERE practices.id = $1 "

// 	distributions := []string{}

// 	scoreInterval := 1 / float64(args.ColumnNum)

// 	initAddiCondition := ""

// 	if args.ColumnNum == 1 {
// 		initAddiCondition = " OR stu_practice_data.avg_score IS NULL "
// 	}

// 	initDistribution := fmt.Sprintf(
// 		`SUM (
// 			CASE
// 				WHEN (stu_practice_data.avg_score >= %f * exam_papers.total_score
// 					AND stu_practice_data.avg_score <= %d * exam_papers.total_score)
// 					%s
// 					THEN 1
// 				ELSE 0
// 			END
// 		)`, scoreInterval*float64(args.ColumnNum-1), 1, initAddiCondition)

// 	distributions = append(distributions, initDistribution)

// 	for i := args.ColumnNum - 1; i > 0; i-- {

// 		addiCondition := ""

// 		if i == 1 {
// 			addiCondition = " OR stu_practice_data.avg_score IS NULL "
// 		}

// 		sqlStr := fmt.Sprintf(
// 			`SUM (
// 				CASE
// 					WHEN (stu_practice_data.avg_score >= %f * exam_papers.total_score
// 						AND stu_practice_data.avg_score < %f * exam_papers.total_score)
// 						%s
// 						THEN 1
// 					ELSE 0
// 				END
// 			)`, scoreInterval*float64(i-1), scoreInterval*float64(i), addiCondition)

// 		distributions = append(distributions, sqlStr)

// 	}

// 	sql := fmt.Sprintf(`WITH stu_practice_data AS (
// 		SELECT
// 			student_id,
// 			practice_id,
// 			name,
// 			exam_paper_id,
// 			COUNT(1) AS attempt,
// 			AVG(total_score) AS avg_score,
// 			AVG(wrong_count)::integer AS avg_wrong_count,
// 			AVG(used_time) AS avg_used_time
// 		FROM v_student_practice_total_score
// 		GROUP BY
// 			student_id,
// 			practice_id,
// 			name,
// 			exam_paper_id
// 	)
// 	SELECT
// 		practices.id AS practice_id,
// 		practices.name AS practice_name,
// 		COUNT( stu_practice_data.student_id) AS total_stu,
// 		exam_papers.total_score AS total_score,
// 		ARRAY[
// 			%s
// 		] AS score_distribution

// 	FROM t_practice practices
// 		JOIN stu_practice_data ON stu_practice_data.practice_id = practices.id
// 		JOIN t_exam_paper exam_papers ON exam_papers.id = stu_practice_data.exam_paper_id
// 	%s
// 	GROUP BY
// 		practices.id,
// 		exam_papers.total_score
// 	`, strings.Join(distributions, ", "), whereClause)

// 	err = conn.QueryRow(ctx, sql, practiceID).Scan(
// 		&result.PracticeID,
// 		&result.PracticeName,
// 		&result.TotalStudents,
// 		&result.TotalScore,
// 		&result.GradeDistribution,
// 	)
// 	if err != nil {
// 		// if errors.Is(err, pgx.ErrNoRows) {
// 		// 	err = fmt.Errorf("%w: no practice found with ID %d", err, practiceID)
// 		// 	z.Error(err.Error())
// 		// 	return result, err
// 		// }

// 		err = fmt.Errorf("get practice grade distribution(practiceID=%d) error: %s", practiceID, err.Error())
// 		z.Error(err.Error())
// 		return result, err
// 	}

// 	return result, err

// }

// func GradeListExamineeExam(ctx context.Context, args GradeListExamineeExamArgs) ([]ExamExamineeScoreInfo, int64, error) {
// 	var err error
// 	z.Info("---->" + cmn.FncName())

// 	if len(args.ExamID) <= 0 {
// 		err = fmt.Errorf("%w: invalid exam ID, expected a positive number, got %d", ErrInvalidID, args.ExamID)
// 		z.Error(err.Error())
// 		return nil, 0, err
// 	}

// 	filter := args.Filter

// 	params := []any{}

// 	placeholders := make([]string, len(args.ExamID))
// 	for i, id := range args.ExamID {
// 		placeholders[i] = fmt.Sprintf("$%d", i+1)
// 		params = append(params, id)
// 	}
// 	placeholderStr := strings.Join(placeholders, ", ")
// 	whereClause := fmt.Sprintf("WHERE v_stu_score.exam_id IN (%s) ", placeholderStr)

// 	if filter.Keyword != "" {
// 		whereClause += fmt.Sprintf(" AND (v_examinee_info.official_name::text ILIKE $%d ", len(params)+1)
// 		params = append(params, fmt.Sprintf("%%%s%%", filter.Keyword))

// 		whereClause += fmt.Sprintf(" OR v_examinee_info.mobile_phone::text ILIKE $%d ", len(params)+1)
// 		params = append(params, fmt.Sprintf("%%%s%%", filter.Keyword))

// 		whereClause += fmt.Sprintf(" OR v_examinee_info.account::text ILIKE $%d)", len(params)+1)
// 		params = append(params, fmt.Sprintf("%%%s%%", filter.Keyword))
// 	}

// 	tExamInfo := cmn.TExamInfo{}

// 	examInfoTableName := tExamInfo.GetTableName()

// 	sql := fmt.Sprintf(`
// 	SELECT
// 		v_stu_score.exam_id,
// 		v_stu_score.exam_session_id,
// 		v_stu_score.student_id AS stu_id,
// 		v_examinee_info.mobile_phone AS phone,
// 		v_examinee_info.official_name AS name,
// 		v_examinee_info.account AS nickname,
// 		v_stu_score.total_score AS score,
// 		v_stu_score.total_score AS total_score,
// 		STRING_AGG(COALESCE(v_examinee_info.remark, ''), '') AS remark
// 	FROM v_student_exam_total_score v_stu_score
// 		JOIN %s exam_infos ON exam_infos.id = v_stu_score.exam_id
// 		JOIN v_examinee_info ON v_examinee_info.student_id = v_stu_score.student_id
// 	%s
// 	GROUP BY
// 		v_stu_score.exam_id,
// 		v_stu_score.exam_session_id,
// 		v_stu_score.student_id,
// 		v_examinee_info.mobile_phone,
// 		v_examinee_info.official_name,
// 		v_examinee_info.account,
// 		v_stu_score.total_score,
// 		exam_infos.id
// 	`, examInfoTableName, whereClause)

// 	var result []ExamExamineeScoreInfo
// 	var rowCount int64

// 	var listSQL string
// 	var listParams []any

// 	if args.Page == -1 && args.PageSize == -1 {
// 		listSQL = sql
// 		listParams = params
// 	} else {
// 		if args.Page <= 0 {
// 			err = fmt.Errorf("%w: page is invalid, expected a positive number", ErrInvalidPage)
// 			z.Error(err.Error())
// 			return nil, 0, err
// 		}
// 		if args.PageSize <= 0 {
// 			err = fmt.Errorf("%w: pageSize is invalid, expected a positive number", ErrInvalidPageSize)
// 			z.Error(err.Error())
// 			return nil, 0, err
// 		}

// 		examSessionCount, err := GetExamSessionCount(args.ExamID)
// 		if err != nil {
// 			z.Error(err.Error())
// 			return nil, 0, err
// 		}

// 		listSQL = fmt.Sprintf(`%s
// 	ORDER BY exam_infos.id DESC
// 	LIMIT $%d OFFSET $%d`,
// 			sql, len(params)+1, len(params)+2)

// 		listParams = append(listParams, args.PageSize*examSessionCount, (args.Page-1)*args.PageSize*examSessionCount)
// 	}

// 	conn := cmn.GetDbConn()
// 	if conn == nil {
// 		err = fmt.Errorf("get exam examinee score failed: %w", ErrNilDBConn)
// 		z.Error(err.Error())
// 		return nil, 0, err
// 	}

// 	rows, err := conn.Query(listSQL, listParams...)
// 	if err != nil {
// 		err = fmt.Errorf("get exam examinee score list error: %w", err)
// 		z.Error(err.Error())
// 		return nil, 0, err
// 	}

// 	defer rows.Close()
// 	for rows.Next() {
// 		var scoreInfo ExamExamineeScoreInfo
// 		err := rows.Scan(
// 			&scoreInfo.ExamID,
// 			&scoreInfo.ExamSessionID,
// 			&scoreInfo.StuID,
// 			&scoreInfo.Phone,
// 			&scoreInfo.Name,
// 			&scoreInfo.Nickname,
// 			&scoreInfo.Score,
// 			&scoreInfo.TotalScore,
// 			&scoreInfo.Remark,
// 		)
// 		if err != nil {
// 			err = fmt.Errorf("scan exam examinee score list error: %w", err)
// 			z.Error(err.Error())
// 			return nil, 0, err
// 		}
// 		result = append(result, scoreInfo)

// 	}

// 	rowCount, err = GetRowCount(ctx, sql, params)
// 	if err != nil {
// 		z.Error(err.Error())
// 		return nil, 0, err
// 	}

// 	return result, rowCount, err
// }

// func GradeListExamineePractice(ctx context.Context, args GradeListExamineePracticeArgs) ([]PracticeExamineeScoreInfo, int64, error) {
// 	var err error
// 	z.Info("---->" + cmn.FncName())

// 	if len(args.PracticeID) <= 0 {
// 		err = fmt.Errorf("%w: invalid exam ID, expected a positive number, got %d", ErrInvalidID, args.PracticeID)
// 		z.Error(err.Error())
// 		return nil, 0, err
// 	}

// 	filter := args.Filter
// 	params := []any{}

// 	placeholders := make([]string, len(args.PracticeID))
// 	for i, id := range args.PracticeID {
// 		placeholders[i] = fmt.Sprintf("$%d", i+1)
// 		params = append(params, id)
// 	}
// 	placeholderStr := strings.Join(placeholders, ", ")
// 	whereClause := fmt.Sprintf("WHERE v_stu_score.practice_id IN (%s) ", placeholderStr)

// 	if filter.Keyword != "" {
// 		whereClause += fmt.Sprintf(" AND (v_examinee_info.official_name::text ILIKE $%d ", len(params)+1)
// 		params = append(params, fmt.Sprintf("%%%s%%", filter.Keyword))

// 		whereClause += fmt.Sprintf(" OR v_examinee_info.mobile_phone::text ILIKE $%d ", len(params)+1)
// 		params = append(params, fmt.Sprintf("%%%s%%", filter.Keyword))

// 		whereClause += fmt.Sprintf(" OR v_examinee_info.account::text ILIKE $%d)", len(params)+1)
// 		params = append(params, fmt.Sprintf("%%%s%%", filter.Keyword))
// 	}

// 	tPractice := cmn.TPractice{}
// 	practiceTableName := tPractice.GetTableName()

// 	sql := fmt.Sprintf(`
// 	SELECT
// 		v_stu_score.practice_id,
// 		v_stu_score.student_id,
// 		v_examinee_info.mobile_phone AS phone,
// 		v_examinee_info.official_name AS name,
// 		v_examinee_info.account AS nickname,
// 		MAX(v_stu_score.total_score) AS highest_score,
// 		COUNT(DISTINCT v_stu_score.id) AS submitted_cnt,
// 		STRING_AGG(COALESCE(v_examinee_info.remark, ''), '') AS remark
// 	FROM v_student_practice_total_score v_stu_score
// 		JOIN %s exam_infos ON exam_infos.id = v_stu_score.practice_id
// 		JOIN v_examinee_info ON v_examinee_info.student_id = v_stu_score.student_id
// 	%s
// 	GROUP BY v_stu_score.practice_id, v_stu_score.student_id, v_examinee_info.mobile_phone, v_examinee_info.official_name, v_examinee_info.account, exam_infos.id
// 		`, practiceTableName, whereClause)

// 	var result []PracticeExamineeScoreInfo
// 	var rowCount int64

// 	var listSQL string
// 	var listParams []any

// 	if args.Page == -1 && args.PageSize == -1 {
// 		sql = sql
// 		params = params
// 	} else {
// 		if args.Page <= 0 {
// 			err = fmt.Errorf("%w: page is invalid, expected a positive number", ErrInvalidPage)
// 			z.Error(err.Error())
// 			return nil, 0, err
// 		}
// 		if args.PageSize <= 0 {
// 			err = fmt.Errorf("%w: pageSize is invalid, expected a positive number", ErrInvalidPageSize)
// 			z.Error(err.Error())
// 			return nil, 0, err
// 		}

// 		listSQL = fmt.Sprintf(`%s
// 	ORDER BY exam_infos.id DESC
// 	LIMIT $%d OFFSET $%d`,
// 			sql, len(listParams)+1, len(listParams)+2)

// 		listParams = append(listParams, args.PageSize, (args.Page-1)*args.PageSize)
// 	}

// 	conn := cmn.GetDbConn()
// 	if conn == nil {
// 		err = fmt.Errorf("get exam examinee score failed: %w", ErrNilDBConn)
// 		z.Error(err.Error())
// 		return nil, 0, err
// 	}

// 	rows, err := conn.Query(listSQL, listParams...)
// 	if err != nil {
// 		err = fmt.Errorf("get exam examinee score list error: %w", err)
// 		z.Error(err.Error())
// 		return nil, 0, err
// 	}

// 	defer rows.Close()
// 	for rows.Next() {
// 		var scoreInfo PracticeExamineeScoreInfo
// 		err := rows.Scan(
// 			&scoreInfo.PracticeID,
// 			&scoreInfo.StuID,
// 			&scoreInfo.Phone,
// 			&scoreInfo.Name,
// 			&scoreInfo.Nickname,
// 			&scoreInfo.HighestScore,
// 			&scoreInfo.SubmittedCnt,
// 			&scoreInfo.Remark,
// 		)
// 		z.Debug("exam examinee score info", zap.Any("scoreInfo", scoreInfo))
// 		if err != nil {
// 			err = fmt.Errorf("scan exam examinee score list error: %w", err)
// 			z.Error(err.Error())
// 			return nil, 0, err
// 		}
// 		result = append(result, scoreInfo)

// 	}

// 	rowCount, err = GetRowCount(ctx, sql, params)
// 	if err != nil {
// 		z.Error(err.Error())
// 		return nil, 0, err
// 	}

// 	return result, rowCount, err

// }

// func GradeAnalysisExam(ctx context.Context, args GradeAnalysisExamArgs) (ExamAnalysis, error) {
// 	var analysis ExamAnalysis

// 	z.Info("---->" + cmn.FncName())

// 	conn := cmn.GetDbConn()
// 	if conn == nil {
// 		return analysis, fmt.Errorf("%w: ", ErrNilDBConn)
// 	}

// 	// 从exam_paper表获取paper_id
// 	examPaperSql := `
// 		SELECT
// 			exam_session_id, id AS exam_paper_id, question_groups
// 		FROM t_exam_paper
// 		WHERE exam_session_id = $1
// 	`
// 	examPaperParams := []any{args.ExamSessionID}

// 	rows, err := conn.Query(examPaperSql, examPaperParams...)
// 	if err != nil {
// 		err = fmt.Errorf("get created exam info(examSessionID=%v) occurred error: %s", analysis.ExamSessionID, err.Error())
// 		z.Error(err.Error())
// 		return analysis, err
// 	}
// 	defer rows.Close()

// 	for rows.Next() {
// 		err := rows.Scan(
// 			&analysis.ExamSessionID,
// 			&analysis.ExamPaperID,
// 			&analysis.QuestionGroups,
// 		)
// 		// z.Debug("exam analysis info", zap.Any("analysis", analysis))
// 		if err != nil {
// 			err = fmt.Errorf("scan created exam info(examSessionID=%v) occurred error: %s", analysis.ExamSessionID, err.Error())
// 			z.Error(err.Error())
// 			return analysis, err
// 		}
// 	}

// 	var examPaper *cmn.TVPaper
// 	examPaper, err = GetManualPaperDetailByPaperID(ctx, analysis.ExamPaperID)
// 	if err != nil {
// 		err = fmt.Errorf("get exam paper detail for exam paper ID %d: %w", analysis.ExamPaperID, err)
// 		z.Error(err.Error())
// 		return analysis, err
// 	}
// 	analysis.ExamPaper = examPaper
// 	// 从exam_paper_question表获取exam_paper_id对应的题目信息
// 	// examPaperQuestionSql := `
// 	// 	SELECT id, exam_paper_id, score, type, content, options, answers,
// 	// 	       analysis, title, answer_path, test_path, input, output,
// 	// 	       example, repo, commit_id, creator, create_time, updated_by,
// 	// 	       update_time, addi, status, "order", group_id, question_attachments_path
// 	// 	FROM t_exam_paper_question
// 	// 	WHERE exam_paper_id = $1`
// 	// examPaperQuestionSql := `
// 	// 	SELECT id, score, type, content, options, answers,
// 	// 	       analysis, title, answer_path, test_path, input, output,
// 	// 	       example, repo, commit_id, creator, create_time, updated_by,
// 	// 	       update_time, addi, status, "order", group_id, question_attachments_path
// 	// 	FROM t_exam_paper_question
// 	// 	WHERE exam_paper_id = $1`
// 	// rows, err = conn.Query(examPaperQuestionSql, analysis.ExamPaperID)
// 	// if err != nil {
// 	// 	z.Error("get exam paper questions for exam paper ID", zap.Error(err))
// 	// 	return analysis, fmt.Errorf("get exam paper questions for exam paper ID: %w", err)
// 	// }
// 	// defer rows.Close()

// 	// // 返回题目信息
// 	// var questions []cmn.TExamPaperQuestion
// 	// 收集题目ID，用于获取学生答案
// 	// var questionIDs []cmn.NullInt
// 	var questionIDs []null.Int

// 	// for rows.Next() {
// 	// 	var q cmn.TExamPaperQuestion
// 	// 	err := rows.Scan(
// 	// 		&q.ID, &q.Score, &q.Type, &q.Content,
// 	// 		&q.Options, &q.Answers, &q.Analysis, &q.Title, &q.AnswerPath,
// 	// 		&q.TestPath, &q.Input, &q.Output, &q.Example, &q.Repo,
// 	// 		&q.CommitID, &q.Creator, &q.CreateTime, &q.UpdatedBy,
// 	// 		&q.UpdateTime, &q.Addi, &q.Status, &q.Order, &q.GroupID,
// 	// 		&q.QuestionAttachmentsPath)
// 	// 	if err != nil {
// 	// 		return analysis, fmt.Errorf("scan exam paper question for exam paper ID")
// 	// 	}
// 	// 	questionIDs = append(questionIDs, q.ID)
// 	// 	questions = append(questions, q)
// 	// }
// 	// if err = rows.Err(); err != nil {
// 	// 	return analysis, fmt.Errorf("error occurred while iterating exam paper questions for exam paper ID: %w", err)
// 	// }

// 	// // z.Debug("exam paper questions", zap.Any("questions", questions))
// 	// analysis.Questions = questions

// 	// 拼接 question IDs
// 	questionIDsStr := make([]string, len(questionIDs))
// 	for i, id := range questionIDs {
// 		questionIDsStr[i] = fmt.Sprintf("%d", id.Int64)
// 	}
// 	questionIDsSql := strings.Join(questionIDsStr, ", ")

// 	// 获取学生答案统计
// 	getStudentAnswerSql := fmt.Sprintf(`
// 	SELECT
// 		tsa.answer
// 	FROM t_student_answers tsa
// 	WHERE tsa.question_id IN (%s)
// 	`, questionIDsSql)

// 	rows, err = conn.Query(getStudentAnswerSql)
// 	if err != nil {
// 		z.Error("get student answers for question ID", zap.Error(err))
// 		return analysis, fmt.Errorf("get student answers for exam paper ID: %w", err)
// 	}
// 	defer rows.Close()

// 	// 获取学生答案统计
// 	questionAnswersStats := make(map[null.Int]map[string]int)

// 	for rows.Next() {
// 		var Answer JSONText
// 		err := rows.Scan(&Answer)
// 		if err != nil {
// 			z.Error("scan student answer row", zap.Error(err))
// 			return analysis, fmt.Errorf("scan student answer row: %w", err)
// 		}
// 		raw := json.RawMessage(Answer)
// 		var ans struct {
// 			Type       string   `json:"type"`
// 			Ans        []string `json:"answer"`
// 			QuestionID null.Int `json:"question_id"`
// 		}
// 		if err := json.Unmarshal(raw, &ans); err != nil {
// 			z.Error("unmarshal group failed", zap.Error(err))
// 			return analysis, fmt.Errorf("unmarshal student answer row: %w", err)
// 		}
// 		switch ans.Type {
// 		case "00":
// 			if questionAnswersStats[ans.QuestionID] == nil {
// 				questionAnswersStats[ans.QuestionID] = make(map[string]int)
// 			}
// 			for _, answer := range ans.Ans {
// 				if _, ok := questionAnswersStats[ans.QuestionID][answer]; !ok {
// 					questionAnswersStats[ans.QuestionID][answer] = 0
// 				}
// 				questionAnswersStats[ans.QuestionID][answer]++
// 			}
// 		case "02":
// 			if questionAnswersStats[ans.QuestionID] == nil {
// 				questionAnswersStats[ans.QuestionID] = make(map[string]int)
// 			}
// 			for _, answer := range ans.Ans {
// 				if _, ok := questionAnswersStats[ans.QuestionID][answer]; !ok {
// 					questionAnswersStats[ans.QuestionID][answer] = 0
// 				}
// 				questionAnswersStats[ans.QuestionID][answer]++
// 			}
// 		case "04":
// 			if questionAnswersStats[ans.QuestionID] == nil {
// 				questionAnswersStats[ans.QuestionID] = make(map[string]int)
// 			}
// 			for _, answer := range ans.Ans {
// 				if _, ok := questionAnswersStats[ans.QuestionID][answer]; !ok {
// 					questionAnswersStats[ans.QuestionID][answer] = 0
// 				}
// 				questionAnswersStats[ans.QuestionID][answer]++
// 			}
// 		}
// 	}
// 	// z.Debug("question answers", zap.Any("questionAnswersStats", questionAnswersStats))

// 	analysis.QuestionAnswersStats = questionAnswersStats

// 	// 获取学生主观题得分统计
// 	getSubjectiveScoreSql := fmt.Sprintf(`
// 	SELECT
// 		tm.question_id, AVG(tm.score) AS total_score
// 	FROM t_mark tm
// 	WHERE tm.question_id IN (%s)
// 	GROUP BY tm.question_id
// 	`, questionIDsSql)

// 	rows, err = conn.Query(getSubjectiveScoreSql)
// 	if err != nil {
// 		z.Error("get student subjective scores for question ID", zap.Error(err))
// 		return analysis, fmt.Errorf("get student subjective scores for exam paper ID: %w", err)
// 	}
// 	defer rows.Close()

// 	// 获取学生主观题得分统计
// 	subjectiveScores := make(map[null.Int]float64)

// 	for rows.Next() {
// 		var questionID null.Int
// 		var totalScore float64
// 		err := rows.Scan(&questionID, &totalScore)
// 		if err != nil {
// 			z.Error("scan student subjective score row", zap.Error(err))
// 			return analysis, fmt.Errorf("scan student subjective score row: %w", err)
// 		}
// 		subjectiveScores[questionID] = totalScore
// 	}
// 	// z.Debug("subjective scores", zap.Any("subjectiveScores", subjectiveScores))

// 	analysis.SubjectiveScores = subjectiveScores

// 	return analysis, nil
// }

// func GetAnalysisPractice(ctx context.Context, args GradeAnalysisPracticeArgs) (PracticeAnalysis, error) {
// 	var analysis PracticeAnalysis

// 	z.Info("---->" + cmn.FncName())

// 	conn := cmn.GetDbConn()
// 	if conn == nil {
// 		return analysis, fmt.Errorf("%w: ", ErrNilDBConn)
// 	}

// 	// 通过paperID获取题目
// 	examPaperSql := `
// 		SELECT
// 			id AS exam_paper_id, question_groups
// 		FROM t_exam_paper
// 		WHERE practice_id = $1
// 	`
// 	examPaperParams := []any{args.PracticeID}

// 	rows, err := conn.Query(examPaperSql, examPaperParams...)
// 	if err != nil {
// 		err = fmt.Errorf("get created exam info(practiceID=%v) occurred error: %s", args.PracticeID, err.Error())
// 		z.Error(err.Error())
// 		return analysis, err
// 	}
// 	defer rows.Close()

// 	for rows.Next() {
// 		err := rows.Scan(
// 			&analysis.ExamPaperID,
// 			&analysis.QuestionGroups,
// 		)
// 		// z.Debug("exam analysis info", zap.Any("analysis", analysis))
// 		if err != nil {
// 			err = fmt.Errorf("scan created exam info(practiceID=%v) occurred error: %s", args.PracticeID, err.Error())
// 			z.Error(err.Error())
// 			return analysis, err
// 		}
// 	}
// 	// examPaperQuestionSql := `
// 	// 	SELECT id, exam_paper_id, score, type, content, options, answers,
// 	// 	       analysis, title, answer_path, test_path, input, output,
// 	// 	       example, repo, commit_id, creator, create_time, updated_by,
// 	// 	       update_time, addi, status, "order", group_id, question_attachments_path
// 	// 	FROM t_exam_paper_question
// 	// 	WHERE exam_paper_id = $1`
// 	examPaperQuestionSql := `
// 		SELECT id, score, type, content, options, answers,
// 		       analysis, title, answer_path, test_path, input, output,
// 		       example, repo, commit_id, creator, create_time, updated_by,
// 		       update_time, addi, status, "order", group_id, question_attachments_path
// 		FROM t_exam_paper_question
// 		WHERE exam_paper_id = $1`

// 	rows, err = conn.Query(examPaperQuestionSql, analysis.ExamPaperID)
// 	if err != nil {
// 		z.Error("get exam paper questions for exam paper ID", zap.Error(err))
// 		return analysis, fmt.Errorf("get exam paper questions for exam paper ID: %w", err)
// 	}
// 	defer rows.Close()

// 	// 返回题目信息
// 	var questions []cmn.TExamPaperQuestion
// 	// 收集题目ID，用于获取学生答案
// 	// var questionIDs []cmn.NullInt
// 	var questionIDs []null.Int

// 	for rows.Next() {
// 		var q cmn.TExamPaperQuestion
// 		err := rows.Scan(
// 			&q.ID, &q.Score, &q.Type, &q.Content,
// 			&q.Options, &q.Answers, &q.Analysis, &q.Title, &q.AnswerPath,
// 			&q.TestPath, &q.Input, &q.Output, &q.Example, &q.Repo,
// 			&q.CommitID, &q.Creator, &q.CreateTime, &q.UpdatedBy,
// 			&q.UpdateTime, &q.Addi, &q.Status, &q.Order, &q.GroupID,
// 			&q.QuestionAttachmentsPath)
// 		if err != nil {
// 			return analysis, fmt.Errorf("scan exam paper question for exam paper ID")
// 		}
// 		questionIDs = append(questionIDs, q.ID)
// 		questions = append(questions, q)
// 	}
// 	if err = rows.Err(); err != nil {
// 		return analysis, fmt.Errorf("error occurred while iterating exam paper questions for exam paper ID: %w", err)
// 	}

// 	// z.Debug("exam paper questions", zap.Any("questions", questions))
// 	analysis.Questions = questions

// 	// 拼接 question IDs
// 	questionIDsStr := make([]string, len(questionIDs))
// 	for i, id := range questionIDs {
// 		questionIDsStr[i] = fmt.Sprintf("%d", id.Int64)
// 	}
// 	questionIDsSql := strings.Join(questionIDsStr, ", ")

// 	// 获取学生答案统计
// 	getStudentAnswerSql := fmt.Sprintf(`
// 	SELECT
// 		tsa.answer
// 	FROM t_student_answers tsa
// 	WHERE tsa.question_id IN (%s)
// 	`, questionIDsSql)

// 	rows, err = conn.Query(getStudentAnswerSql)
// 	if err != nil {
// 		z.Error("get student answers for question ID", zap.Error(err))
// 		return analysis, fmt.Errorf("get student answers for exam paper ID: %w", err)
// 	}
// 	defer rows.Close()

// 	// 获取学生答案统计
// 	questionAnswersStats := make(map[null.Int]map[string]int)

// 	for rows.Next() {
// 		var Answer JSONText
// 		err := rows.Scan(&Answer)
// 		if err != nil {
// 			z.Error("scan student answer row", zap.Error(err))
// 			return analysis, fmt.Errorf("scan student answer row: %w", err)
// 		}
// 		raw := json.RawMessage(Answer)
// 		var ans struct {
// 			Type       string   `json:"type"`
// 			Ans        []string `json:"answer"`
// 			QuestionID null.Int `json:"question_id"`
// 		}
// 		if err := json.Unmarshal(raw, &ans); err != nil {
// 			z.Error("unmarshal group failed", zap.Error(err))
// 			return analysis, fmt.Errorf("unmarshal student answer row: %w", err)
// 		}
// 		switch ans.Type {
// 		case "00":
// 			if questionAnswersStats[ans.QuestionID] == nil {
// 				questionAnswersStats[ans.QuestionID] = make(map[string]int)
// 			}
// 			for _, answer := range ans.Ans {
// 				if _, ok := questionAnswersStats[ans.QuestionID][answer]; !ok {
// 					questionAnswersStats[ans.QuestionID][answer] = 0
// 				}
// 				questionAnswersStats[ans.QuestionID][answer]++
// 			}
// 		case "02":
// 			if questionAnswersStats[ans.QuestionID] == nil {
// 				questionAnswersStats[ans.QuestionID] = make(map[string]int)
// 			}
// 			for _, answer := range ans.Ans {
// 				if _, ok := questionAnswersStats[ans.QuestionID][answer]; !ok {
// 					questionAnswersStats[ans.QuestionID][answer] = 0
// 				}
// 				questionAnswersStats[ans.QuestionID][answer]++
// 			}
// 		case "04":
// 			if questionAnswersStats[ans.QuestionID] == nil {
// 				questionAnswersStats[ans.QuestionID] = make(map[string]int)
// 			}
// 			for _, answer := range ans.Ans {
// 				if _, ok := questionAnswersStats[ans.QuestionID][answer]; !ok {
// 					questionAnswersStats[ans.QuestionID][answer] = 0
// 				}
// 				questionAnswersStats[ans.QuestionID][answer]++
// 			}
// 		}
// 	}
// 	// z.Debug("question answers", zap.Any("questionAnswersStats", questionAnswersStats))

// 	analysis.QuestionAnswersStats = questionAnswersStats

// 	// 获取学生主观题得分统计
// 	getSubjectiveScoreSql := fmt.Sprintf(`
// 	SELECT
// 		tm.question_id, AVG(tm.score) AS total_score
// 	FROM t_mark tm
// 	WHERE tm.question_id IN (%s)
// 	GROUP BY tm.question_id
// 	`, questionIDsSql)

// 	rows, err = conn.Query(getSubjectiveScoreSql)
// 	if err != nil {
// 		z.Error("get student subjective scores for question ID", zap.Error(err))
// 		return analysis, fmt.Errorf("get student subjective scores for exam paper ID: %w", err)
// 	}
// 	defer rows.Close()

// 	// 获取学生主观题得分统计
// 	subjectiveScores := make(map[null.Int]float64)

// 	for rows.Next() {
// 		var questionID null.Int
// 		var totalScore float64
// 		err := rows.Scan(&questionID, &totalScore)
// 		if err != nil {
// 			z.Error("scan student subjective score row", zap.Error(err))
// 			return analysis, fmt.Errorf("scan student subjective score row: %w", err)
// 		}
// 		subjectiveScores[questionID] = totalScore
// 	}
// 	// z.Debug("subjective scores", zap.Any("subjectiveScores", subjectiveScores))

// 	analysis.SubjectiveScores = subjectiveScores

// 	return analysis, nil
// }
