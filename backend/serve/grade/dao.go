package grade

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
	"w2w.io/cmn"
	"w2w.io/null"
	"w2w.io/serve/examPaper"
)

/*
* 获取考试成绩列表
 */
func gradeListExam(ctx context.Context, args *GradeListArgs) ([]GradeExam, int64, error) {
	z.Info("---->" + cmn.FncName())

	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}
	z.Debug(forceErr)

	var (
		page     int
		pageSize int

		result   []GradeExam
		rowCount int64
		err      error
	)

	// 分页参数: Page PageSize
	// 数据库查询必需参数: TeacherID ExamID
	// 可选筛选条件参数: Name Type Submitted

	if args.Page <= 0 {
		err = fmt.Errorf("无效页码: 页码必须为正整数")
		z.Error(err.Error())
		return result, rowCount, err
	}
	page = args.Page

	if args.PageSize <= 0 {
		err = fmt.Errorf("无效每页数量: 每页数量必须为正整数")
		z.Error(err.Error())
		return result, rowCount, err
	}
	pageSize = args.PageSize

	if args.TeacherID <= 0 && args.TeacherID != -1 {
		err = fmt.Errorf("无效教师ID")
		z.Error(err.Error())
		return result, rowCount, err
	}

	if args.ExamID < 0 {
		err = fmt.Errorf("无效考试ID: 考试ID必须为正整数")
		z.Error(err.Error())
		return result, rowCount, err
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

	// -1 表示全选
	if filter.Submitted != -1 {
		whereClause += fmt.Sprintf(" AND ei.submitted = $%d ", len(params)+1)
		params = append(params, filter.Submitted == 1)
	}

	// 视图SQL
	sql := fmt.Sprintf(`
	SELECT 
		ei.id AS id,
		ei.name AS exam_name,
		ei.type AS exam_type,
		jsonb_agg(esi) AS exam_session_info,
		ei.submitted AS submitted
	FROM t_exam_info ei     
		LEFT JOIN v_z_grade_exam_session_info esi ON esi.exam_id = ei.id
	%s
	GROUP BY ei.id
	`, whereClause)

	conn := cmn.GetPgxConn()
	if conn == nil || forceErr == "conn nil" {
		err = fmt.Errorf("获取考试成绩列表查询数据库连接失败")
		z.Error(err.Error())
		return result, rowCount, err
	}

	// 统计SQL
	countSql := fmt.Sprintf(`SELECT COUNT(1) FROM (%s) AS exam_grade_list_count`, sql)
	err = conn.QueryRow(ctx, countSql, params...).Scan(&rowCount)
	if forceErr == "conn.QueryRow fail" {
		err = errors.New(forceErr)
	}
	if err != nil {
		err = fmt.Errorf("执行查询语句失败: %w", err)
		z.Error(err.Error())
		return result, 0, err
	}

	// 分页SQL
	listSQL := fmt.Sprintf(`%s
	ORDER BY ei.id DESC
	LIMIT $%d OFFSET $%d`,
		sql, len(params)+1, len(params)+2)
	listParams := append(params, pageSize, (page-1)*pageSize)

	rows, err := conn.Query(ctx, listSQL, listParams...)
	if err != nil || forceErr == "conn query fail" {
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
		if forceErr == "rows scan fail" {
			err = errors.New(forceErr)
		}
		if err != nil {
			err = fmt.Errorf("扫描考试成绩列表失败: %w", err)
			z.Error(err.Error())
			return result, rowCount, err
		}
		result = append(result, grade)
	}
	err = rows.Err()

	return result, rowCount, nil
}

/*
* 获取练习成绩列表
 */
func gradeListPractice(ctx context.Context, args *GradeListArgs) ([]GradePractice, int64, error) {
	z.Info("---->" + cmn.FncName())

	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}
	z.Debug(forceErr)

	var err error
	var page int
	var pageSize int
	var result []GradePractice
	var rowCount int64

	// 分页参数: Page PageSize
	// 数据库查询必需参数: TeacherID PracticeID
	// 可选筛选参数: Name

	if args.Page <= 0 {
		err = fmt.Errorf("无效页码: 页码必须为正整数")
		z.Error(err.Error())
		return result, 0, err
	}
	page = args.Page

	if args.PageSize <= 0 {
		err = fmt.Errorf("无效每页数量: 每页数量必须为正整数")
		z.Error(err.Error())
		return result, 0, err
	}
	pageSize = args.PageSize

	if args.TeacherID <= 0 && args.TeacherID != -1 {
		err = fmt.Errorf("无效教师ID")
		z.Error(err.Error())
		return result, 0, err
	}

	if args.PracticeID < 0 {
		err = fmt.Errorf("无效练习ID: 练习ID必须为正整数")
		z.Error(err.Error())
		return result, 0, err
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

	// 视图查询SQL
	sql := fmt.Sprintf(`
	SELECT
		p.practice_id,
		p.practice_name,
		p.total_score,
		p.averge_score,
		p.actual_completer,
		p.pass_student
	FROM v_z_grade_practice_statistics p 
	%s
	GROUP BY
		p.practice_id, p.practice_name, p.total_score, p.averge_score, p.actual_completer, p.pass_student
		`, whereClause)

	conn := cmn.GetPgxConn()
	if conn == nil || forceErr == "conn nil" {
		err = fmt.Errorf("获取考试成绩列表查询数据库连接失败")
		z.Error(err.Error())
		return result, rowCount, err
	}

	// 统计SQL
	countSql := fmt.Sprintf(`SELECT COUNT(1) FROM (%s) AS practice_grade_list_count`, sql)
	err = conn.QueryRow(ctx, countSql, params...).Scan(&rowCount)
	if forceErr == "conn.QueryRow fail" {
		err = errors.New(forceErr)
	}
	if err != nil {
		err = fmt.Errorf("执行查询语句失败: %w", err)
		z.Error(err.Error())
		return result, 0, err
	}

	// 分页SQL
	listSQL := fmt.Sprintf(`%s
	ORDER BY p.practice_id DESC
	LIMIT $%d OFFSET $%d`,
		sql, len(params)+1, len(params)+2)
	listParams := append(params, pageSize, (page-1)*pageSize)

	rows, err := conn.Query(ctx, listSQL, listParams...)
	if err != nil || forceErr == "conn query fail" {
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
		if forceErr == "rows scan fail" {
			err = errors.New(forceErr)
		}
		if err != nil {
			err = fmt.Errorf("扫描练习成绩列表出现错误: %w", err)
			z.Error(err.Error())
			return result, rowCount, err
		}

		result = append(result, grade)
	}

	return result, rowCount, nil
}

/*
* 提交考试成绩
 */
func setExamGradeSubmitted(ctx context.Context, args *GradeSubmitArgs) (int64, error) {
	z.Info("---->" + cmn.FncName())

	var err error
	var rowsAffected int64

	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}

	if args.TeacherID <= 0 {
		err = fmt.Errorf("无效教师ID")
		z.Error(err.Error())
		return 0, err
	}
	teacherID := args.TeacherID

	if len(args.ExamIDs) <= 0 {
		err = fmt.Errorf("无效考试ID，长度为0")
		z.Error(err.Error())
		return 0, err
	}
	examIDs := args.ExamIDs

	conn := cmn.GetPgxConn()
	if conn == nil || forceErr == "conn nil" {
		err = fmt.Errorf("查询练习成绩列表获取数据库连接失败")
		z.Error(err.Error())
		return 0, err
	}

	var tx pgx.Tx
	tx, err = conn.Begin(context.Background())
	if err != nil || forceErr == "conn begin tx fail" {
		err = fmt.Errorf("开启事务失败: %w", err)
		z.Error(err.Error())
		return 0, err
	}

	// 标记事务是否成功，默认失败
	txSuccess := false
	defer func() {
		// 事务回滚
		if !txSuccess || forceErr == "txSuccess must fail" {
			rollbackErr := tx.Rollback(context.Background())
			if rollbackErr != nil {
				err = fmt.Errorf("提交考试成绩失败: 回滚事务失败: %w", rollbackErr)
				z.Error(err.Error())
			}
			return
		}
		// 提交事务
		err = tx.Commit(context.Background())
		if err != nil || forceErr == "tx commit fail" {
			err = fmt.Errorf("提交考试成绩失败: %w", err)
			z.Error(err.Error())
			return
		}
	}()

	examSessions := []cmn.TExamSession{}

	for _, examID := range examIDs {

		querySql := `
			SELECT
				id, end_time
			FROM t_exam_session
			WHERE exam_id=$1 AND status = '06'
			ORDER BY start_time ASC`

		var es_rows pgx.Rows
		es_rows, err := conn.Query(context.Background(), querySql, examID)
		if forceErr == "conn query fail" {
			err = errors.New(forceErr)
		}
		if err != nil {
			z.Error(err.Error())
			return 0, err
		}
		defer es_rows.Close()
		for es_rows.Next() {
			var es cmn.TExamSession
			err = es_rows.Scan(
				&es.ID,
				&es.EndTime,
			)
			if forceErr == "rows scan fail" {
				err = errors.New(forceErr)
			}
			if err != nil {
				z.Error(err.Error())
				return 0, err
			}
			examSessions = append(examSessions, es)
		}

		z.Sugar().Debug("examSessions", zap.Any("examSessions", examSessions))

		currTime := time.Now()

		for _, examSession := range examSessions {
			// 检查 EndTime 是否有效
			if !examSession.EndTime.Valid || forceErr == "examSession endTime invalid" {
				// 如果 EndTime 无效，则认为考试未结束
				err = fmt.Errorf("提交考试成绩失败: %w: 考试场次(examSession=%v)没有有效结束时间", ErrExamIsNotOver, examSession.ID.Int64)
				z.Error(err.Error())
				return 0, err
			}

			endTime := time.UnixMilli(examSession.EndTime.Int64)
			if endTime.After(currTime) || forceErr == "endTime after currTime" {
				// 考试结束时间未过，不可提交
				err = fmt.Errorf("提交考试成绩失败: %w: 考试场次(examSession=%v)还未结束", ErrExamIsNotOver, examSession.ID.Int64)
				z.Error(err.Error())
				return 0, err
			}
		}

		updateSql := `UPDATE t_exam_info SET submitted=true, updated_by=$2, update_time=$3 WHERE id=$1`

		// 获取当前毫秒级时间戳
		curr := currTime.UnixMilli()

		commandTag, err := tx.Exec(ctx, updateSql, examID, teacherID, curr)
		if forceErr == "tx exec fail" {
			err = errors.New(forceErr)
		}
		if err != nil {
			err = fmt.Errorf("提交考试成绩失败(examID=%v teacherID=%v) error: %s", examID, teacherID, err.Error())
			z.Error(err.Error())
			return 0, err
		}
		rowsAffected = commandTag.RowsAffected()
	}

	txSuccess = true

	return rowsAffected, nil
}

func GradeDistributionExam(ctx context.Context, args GradeDistributionArgs) (ExamGradeDistribution, error) {

	z.Info("---->" + cmn.FncName())

	var err error

	result := ExamGradeDistribution{}

	if args.ExamID < 0 {
		err = fmt.Errorf("examID无效")
		z.Error(err.Error())
		return result, err
	}

	if args.ColumnNum < 1 {
		err = fmt.Errorf("列数无效")
		z.Error(err.Error())
		return result, err
	}

	examID := args.ExamID

	conn := cmn.GetPgxConn()
	if conn == nil {
		err = fmt.Errorf("获取数据库连接失败")
		z.Error(err.Error())
		return result, err
	}

	whereClause := " WHERE ei.id = $1 "

	distributions := []string{}

	scoreInterval := 1 / float64(args.ColumnNum)

	initAddiCondition := ""

	if args.ColumnNum == 1 {
		initAddiCondition = " OR sets.total_score IS NULL "
	}

	initDistribution := fmt.Sprintf(
		`SUM (
			CASE
				WHEN (sets.total_score >= %f * ep.total_score
					AND sets.total_score <= %d * ep.total_score)
					%s
					THEN 1
				ELSE 0
			END
		)`, scoreInterval*float64(args.ColumnNum-1), 1, initAddiCondition)

	distributions = append(distributions, initDistribution)

	for i := args.ColumnNum - 1; i > 0; i-- {

		addiCondition := ""

		if i == 1 {
			addiCondition = " OR sets.total_score IS NULL "
		}

		sqlStr := fmt.Sprintf(
			`SUM (
				CASE
					WHEN (sets.total_score >= %f * ep.total_score
						AND sets.total_score < %f * ep.total_score)
						%s
						THEN 1
					ELSE 0
				END
			)`, scoreInterval*float64(i-1), scoreInterval*float64(i), addiCondition)

		distributions = append(distributions, sqlStr)

	}

	sql := fmt.Sprintf(`WITH exam_session_grades AS (
		SELECT
			sets.exam_id AS exam_id,
			sets.exam_session_id,
			ep.id AS exam_paper_id,
			ep.name AS exam_paper_name,
			ep.total_score AS total_score,
			COUNT(DISTINCT sets.student_id) AS total,
			ARRAY[
				%s
			] AS score_distribution
		FROM v_student_exam_total_score sets
			JOIN v_exam_paper ep ON ep.exam_session_id =  sets.exam_session_id
		GROUP BY
			sets.exam_id,
			sets.exam_session_id,
			ep.id,
			ep.name,
			ep.total_score
	)
	SELECT
		ei.id,
		ei.name,
		jsonb_agg(exam_session_grades) AS exam_session_score_distribution
	FROM t_exam_info ei
		JOIN exam_session_grades ON exam_session_grades.exam_id = ei.id
	%s
	GROUP BY
		ei.id
	`, strings.Join(distributions, ", "), whereClause)

	z.Sugar().Debug("sql:%#v", sql)

	err = conn.QueryRow(ctx, sql, examID).Scan(
		&result.ExamID,
		&result.ExamName,
		&result.GradeDistribution,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = fmt.Errorf("%w: no exam found with ID %d", err, examID)
			z.Error(err.Error())
			return result, err
		}

		err = fmt.Errorf("获取考试成绩分布(examID=%d) 错误: %s", examID, err.Error())

		z.Error(err.Error())
		return result, err
	}

	return result, err
}

// // GetPracticeGradeDistribution 将返回指定练习ID的成绩分布信息
func GradeDistributionPractice(ctx context.Context, args GradeDistributionArgs) (PracticeGradeDistribution, error) {

	z.Info("---->" + cmn.FncName())

	var err error

	result := PracticeGradeDistribution{}

	if args.PracticeID <= 0 {
		err = fmt.Errorf("练习ID无效")
		z.Error(err.Error())
		return result, err
	}

	if args.ColumnNum < 1 {
		err = fmt.Errorf("列数无效")
		z.Error(err.Error())
		return result, err
	}

	practiceID := args.PracticeID

	conn := cmn.GetPgxConn()
	if conn == nil {
		err = fmt.Errorf("获取数据库连接失败")
		z.Error(err.Error())
		return result, err
	}

	whereClause := " WHERE practices.id = $1 "

	distributions := []string{}

	scoreInterval := 1 / float64(args.ColumnNum)

	initAddiCondition := ""

	if args.ColumnNum == 1 {
		initAddiCondition = " OR stu_practice_data.avg_score IS NULL "
	}

	initDistribution := fmt.Sprintf(
		`SUM (
			CASE
				WHEN (stu_practice_data.avg_score >= %f * exam_papers.total_score
					AND stu_practice_data.avg_score <= %d * exam_papers.total_score)
					%s
					THEN 1
				ELSE 0
			END
		)`, scoreInterval*float64(args.ColumnNum-1), 1, initAddiCondition)

	distributions = append(distributions, initDistribution)

	for i := args.ColumnNum - 1; i > 0; i-- {

		addiCondition := ""

		if i == 1 {
			addiCondition = " OR stu_practice_data.avg_score IS NULL "
		}

		sqlStr := fmt.Sprintf(
			`SUM (
				CASE
					WHEN (stu_practice_data.avg_score >= %f * exam_papers.total_score
						AND stu_practice_data.avg_score < %f * exam_papers.total_score)
						%s
						THEN 1
					ELSE 0
				END
			)`, scoreInterval*float64(i-1), scoreInterval*float64(i), addiCondition)

		distributions = append(distributions, sqlStr)

	}

	sql := fmt.Sprintf(`WITH stu_practice_data AS (
		SELECT
			student_id,
			practice_id,
			name,
			exam_paper_id,
			COUNT(1) AS attempt,
			AVG(total_score) AS avg_score,
			AVG(wrong_count)::integer AS avg_wrong_count,
			AVG(used_time) AS avg_used_time
		FROM v_student_practice_total_score
		GROUP BY
			student_id,
			practice_id,
			name,
			exam_paper_id
	)
	SELECT
		practices.id AS practice_id,
		practices.name AS practice_name,
		COUNT( stu_practice_data.student_id) AS total_stu,
		exam_papers.total_score AS total_score,
		ARRAY[
			%s
		] AS score_distribution

	FROM t_practice practices
		JOIN stu_practice_data ON stu_practice_data.practice_id = practices.id
		JOIN v_exam_paper exam_papers ON exam_papers.id = stu_practice_data.exam_paper_id
	%s
	GROUP BY
		practices.id,
		exam_papers.total_score
	`, strings.Join(distributions, ", "), whereClause)

	err = conn.QueryRow(ctx, sql, practiceID).Scan(
		&result.PracticeID,
		&result.PracticeName,
		&result.TotalStudents,
		&result.TotalScore,
		&result.GradeDistribution,
	)
	if err != nil {
		// if errors.Is(err, pgx.ErrNoRows) {
		// 	err = fmt.Errorf("%w: no practice found with ID %d", err, practiceID)
		// 	z.Error(err.Error())
		// 	return result, err
		// }

		err = fmt.Errorf("获取练习成绩分布失败(practiceID=%d) 错误: %s", practiceID, err.Error())
		z.Error(err.Error())
		return result, err
	}

	return result, err

}

func GradeExamineeListExam(ctx context.Context, args GradeExamineeListArgs) ([]StudentExamScoreInfo, int64, error) {
	z.Info("---->" + cmn.FncName())
	var err error

	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}

	if len(args.ExamID) <= 0 {
		err = fmt.Errorf("考试ID为空")
		z.Error(err.Error())
		return nil, 0, err
	}

	filter := args.Filter

	params := []any{}

	placeholders := make([]string, len(args.ExamID))
	for i, id := range args.ExamID {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		params = append(params, id)
	}
	placeholderStr := strings.Join(placeholders, ", ")
	whereClause := fmt.Sprintf("WHERE v_stu_score.exam_id IN (%s) ", placeholderStr)

	if filter.Keyword != "" {
		whereClause += fmt.Sprintf(" AND (v_examinee_info.official_name::text ILIKE $%d ", len(params)+1)
		params = append(params, fmt.Sprintf("%%%s%%", filter.Keyword))

		whereClause += fmt.Sprintf(" OR v_examinee_info.mobile_phone::text ILIKE $%d ", len(params)+1)
		params = append(params, fmt.Sprintf("%%%s%%", filter.Keyword))

		whereClause += fmt.Sprintf(" OR v_examinee_info.account::text ILIKE $%d)", len(params)+1)
		params = append(params, fmt.Sprintf("%%%s%%", filter.Keyword))
	}

	sql := fmt.Sprintf(`
SELECT
    v_examinee_info.student_id,
    v_examinee_info.mobile_phone AS phone,
    v_examinee_info.official_name AS name,
    v_examinee_info.account AS nickname,
    STRING_AGG(COALESCE(v_examinee_info.remark, ''), '') AS remark,
    jsonb_agg(
        jsonb_build_object(
            'exam_id', v_stu_score.exam_id,
            'exam_session_id', v_stu_score.exam_session_id,
            'score', v_stu_score.total_score
        )
    ) AS exam_sessions
FROM v_student_exam_total_score v_stu_score
	JOIN t_exam_info exam_infos ON exam_infos.id = v_stu_score.exam_id
	JOIN v_examinee_info ON v_examinee_info.student_id = v_stu_score.student_id
%s
GROUP BY
    v_examinee_info.student_id,
    v_examinee_info.mobile_phone,
    v_examinee_info.official_name,
    v_examinee_info.account,
	exam_infos.id
	`, whereClause)

	// var result []ExamExamineeScoreInfo
	var result []StudentExamScoreInfo
	var rowCount int64

	var listSQL string
	var listParams []any

	conn := cmn.GetPgxConn()
	if conn == nil {
		err = fmt.Errorf("get exam examinee score failed: %w", ErrNilDBConn)
		z.Error(err.Error())
		return nil, 0, err
	}

	if args.Page == -1 && args.PageSize == -1 {
		listSQL = sql
		listParams = params
	} else {
		if args.Page <= 0 {
			err = fmt.Errorf("页码无效，期望一个正整数")
			z.Error(err.Error())
			return nil, 0, err
		}
		if args.PageSize <= 0 {
			err = fmt.Errorf("每页数量无效，期望一个正整数")
			z.Error(err.Error())
			return nil, 0, err
		}

		var examSessionCount int
		{
			examID := args.ExamID
			placeholders := make([]string, len(examID))
			examSessionParams := []any{}
			for i, id := range examID {
				placeholders[i] = fmt.Sprintf("$%d", i+1)
				examSessionParams = append(examSessionParams, id)
			}
			examSessionPlaceholderStr := strings.Join(placeholders, ", ")

			examSessionSql := fmt.Sprintf(`SELECT COUNT(id) FROM t_exam_session WHERE exam_id IN (%s)`, examSessionPlaceholderStr)

			err = conn.QueryRow(ctx, examSessionSql, examSessionParams...).Scan(&examSessionCount)
			if err != nil {
				err = fmt.Errorf("scan exam session count(examID:%v) occurred error: %s", args.ExamID, err.Error())
				z.Error(err.Error())
				return result, 0, err
			}
		}

		listParams = params
		z.Sugar().Debug("打印examSessionCount是什么：", examSessionCount)
		z.Sugar().Debug("listParams", listParams)
		listSQL = fmt.Sprintf(`%s
	ORDER BY exam_infos.id DESC
	LIMIT $%d OFFSET $%d`,
			sql, len(listParams)+1, len(listParams)+2)
		z.Sugar().Debug("listSQL", listSQL)
		listParams = append(listParams, args.PageSize*examSessionCount, (args.Page-1)*args.PageSize*examSessionCount)
		z.Sugar().Debug("listParams", listParams)
	}

	rows, err := conn.Query(ctx, listSQL, listParams...)
	if err != nil {
		err = fmt.Errorf("查询考试考生成绩列表失败 错误: %s", err.Error())
		z.Error(err.Error())
		return nil, 0, err
	}

	defer rows.Close()
	for rows.Next() {

		var s2 StudentExamScoreInfo
		err = rows.Scan(
			&s2.StudentID,
			&s2.Phone,
			&s2.Name,
			&s2.Nickname,
			&s2.Remark,
			&s2.ExamSessions,
		)
		if err != nil {
			err = fmt.Errorf("scan exam examinee score list error: %w", err)
			z.Error(err.Error())
			return nil, 0, err
		}
		result = append(result, s2)

	}

	// 统计SQL
	countSql := fmt.Sprintf(`SELECT COUNT(1) FROM (%s) AS practice_grade_list_count`, sql)
	err = conn.QueryRow(ctx, countSql, params...).Scan(&rowCount)
	if forceErr == "conn.QueryRow fail" {
		err = errors.New(forceErr)
	}
	if err != nil {
		err = fmt.Errorf("执行查询语句失败: %w", err)
		z.Error(err.Error())
		return result, 0, err
	}

	return result, rowCount, err
}

func GradeExamineeListPractice(ctx context.Context, args GradeExamineeListArgs) ([]PracticeExamineeScoreInfo, int64, error) {
	z.Info("---->" + cmn.FncName())
	var err error

	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}

	if len(args.PracticeID) <= 0 {
		err = fmt.Errorf("%w: invalid exam ID, expected a positive number, got %d", ErrInvalidID, args.PracticeID)
		z.Error(err.Error())
		return nil, 0, err
	}

	filter := args.Filter
	params := []any{}

	z.Sugar().Debug("PracticeID", zap.Any("PracticeID", args.PracticeID))

	placeholders := make([]string, len(args.PracticeID))
	for i, id := range args.PracticeID {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		params = append(params, id)
	}
	placeholderStr := strings.Join(placeholders, ", ")
	z.Sugar().Debug("placeholderStr:", placeholderStr)
	whereClause := fmt.Sprintf("WHERE v_stu_score.practice_id IN (%s) ", placeholderStr)

	if filter.Keyword != "" {
		whereClause += fmt.Sprintf(" AND (v_examinee_info.official_name::text ILIKE $%d ", len(params)+1)
		params = append(params, fmt.Sprintf("%%%s%%", filter.Keyword))

		whereClause += fmt.Sprintf(" OR v_examinee_info.mobile_phone::text ILIKE $%d ", len(params)+1)
		params = append(params, fmt.Sprintf("%%%s%%", filter.Keyword))

		whereClause += fmt.Sprintf(" OR v_examinee_info.account::text ILIKE $%d)", len(params)+1)
		params = append(params, fmt.Sprintf("%%%s%%", filter.Keyword))
	}

	sql := fmt.Sprintf(`
	SELECT
		v_stu_score.practice_id,
		v_stu_score.student_id,
		v_examinee_info.mobile_phone AS phone,
		v_examinee_info.official_name AS name,
		v_examinee_info.account AS nickname,
		MAX(v_stu_score.total_score) AS highest_score,
		COUNT(DISTINCT v_stu_score.id) AS submitted_cnt
	FROM v_student_practice_total_score v_stu_score
		LEFT JOIN v_examinee_info ON v_examinee_info.student_id = v_stu_score.student_id
	%s
	GROUP BY 
	v_stu_score.practice_id, 
	v_stu_score.student_id, 
	v_examinee_info.mobile_phone, 
	v_examinee_info.official_name, 
	v_examinee_info.account
		`, whereClause)

	var result []PracticeExamineeScoreInfo
	var rowCount int64

	var listSQL string
	var listParams []any

	if args.Page == -1 && args.PageSize == -1 {
		listSQL = sql
		listParams = params
	} else {
		if args.Page <= 0 {
			err = fmt.Errorf("%w: page is invalid, expected a positive number", ErrInvalidPage)
			z.Error(err.Error())
			return nil, 0, err
		}
		if args.PageSize <= 0 {
			err = fmt.Errorf("%w: pageSize is invalid, expected a positive number", ErrInvalidPageSize)
			z.Error(err.Error())
			return nil, 0, err
		}

		listParams = params
		listSQL = fmt.Sprintf(`%s
	LIMIT $%d OFFSET $%d`,
			sql, len(listParams)+1, len(listParams)+2)

		listParams = append(listParams, args.PageSize, (args.Page-1)*args.PageSize)
	}

	z.Sugar().Debug("listSQL", zap.Any("listSQL", listSQL))
	z.Sugar().Debug("listParams", zap.Any("listParams", listParams))

	conn := cmn.GetPgxConn()
	if conn == nil {
		err = fmt.Errorf("get exam examinee score failed: %w", ErrNilDBConn)
		z.Error(err.Error())
		return nil, 0, err
	}

	rows, err := conn.Query(ctx, listSQL, listParams...)
	if err != nil {
		err = fmt.Errorf("get exam examinee score list error: %w", err)
		z.Error(err.Error())
		return nil, 0, err
	}

	defer rows.Close()
	for rows.Next() {
		var scoreInfo PracticeExamineeScoreInfo
		err = rows.Scan(
			&scoreInfo.PracticeID,
			&scoreInfo.StuID,
			&scoreInfo.Phone,
			&scoreInfo.Name,
			&scoreInfo.Nickname,
			&scoreInfo.HighestScore,
			&scoreInfo.SubmittedCnt,
		)
		z.Debug("exam examinee score info", zap.Any("scoreInfo", scoreInfo))
		if err != nil {
			err = fmt.Errorf("scan exam examinee score list error: %w", err)
			z.Error(err.Error())
			return nil, 0, err
		}
		result = append(result, scoreInfo)

	}

	z.Sugar().Debug("listSQL", zap.Any("listSQL", listSQL))
	z.Sugar().Debug("listParams", zap.Any("listParams", listParams))

	// 统计SQL
	countSql := fmt.Sprintf(`SELECT COUNT(1) FROM (%s) AS practice_grade_list_count`, sql)
	err = conn.QueryRow(ctx, countSql, params...).Scan(&rowCount)
	if forceErr == "conn.QueryRow fail" {
		err = errors.New(forceErr)
	}
	if err != nil {
		err = fmt.Errorf("执行查询语句失败: %w", err)
		z.Error(err.Error())
		return result, 0, err
	}

	return result, rowCount, err

}

func GradeAnalysisExam(ctx context.Context, args GradeArgs) (ExamAnalysis, error) {
	z.Info("---->" + cmn.FncName())

	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}

	var analysis ExamAnalysis

	var err error

	conn := cmn.GetPgxConn()
	if conn == nil {
		return analysis, fmt.Errorf("%w: ", ErrNilDBConn)
	}

	// 从exam_paper表获取paper_id
	examPaperSql := `
		SELECT
			exam_session_id, id AS exam_paper_id
		FROM t_exam_paper
		WHERE exam_session_id = $1
	`
	examPaperParams := []any{args.ExamSessionID}

	err = conn.QueryRow(ctx, examPaperSql, examPaperParams...).Scan(&analysis.ExamSessionID, &analysis.ExamPaperID)
	if err != nil {
		err = fmt.Errorf("get created exam info(examSessionID=%v) occurred error: %s", analysis.ExamSessionID, err.Error())
		z.Error(err.Error())
		return analysis, err
	}

	// for rows.Next() {
	// 	err := rows.Scan(

	// 	)
	// 	z.Debug("exam analysis info", zap.Any("analysis", analysis))
	// 	if err != nil {
	// 		err = fmt.Errorf("scan created exam info(examSessionID=%v) occurred error: %s", analysis.ExamSessionID, err.Error())
	// 		z.Error(err.Error())
	// 		return analysis, err
	// 	}
	// }

	var tx pgx.Tx
	tx, err = conn.Begin(context.Background())
	if err != nil || forceErr == "conn begin tx fail" {
		err = fmt.Errorf("开启事务失败: %w", err)
		z.Error(err.Error())
		return analysis, err
	}

	p, pg, pq, err := examPaper.LoadExamPaperDetailsById(ctx, tx, analysis.ExamPaperID, true, true)
	z.Sugar().Debugf("打印p是什么：%+v", p)
	z.Sugar().Debugf("打印p是什么：%#v", p)
	z.Sugar().Debugf("打印pg是什么：%+v", pg)
	z.Sugar().Debugf("打印pg是什么：%#v", pg)
	z.Sugar().Debugf("打印pq是什么：%+v", pq)
	z.Sugar().Debugf("打印pq是什么：%#v", pq)
	// var examPaper *cmn.TVPaper
	// examPaper, err = GetManualPaperDetailByPaperID(ctx, analysis.ExamPaperID)
	// if err != nil {
	// 	err = fmt.Errorf("get exam paper detail for exam paper ID %d: %w", analysis.ExamPaperID, err)
	// 	z.Error(err.Error())
	// 	return analysis, err
	// }
	// analysis.ExamPaper = examPaper

	// 从exam_paper_question表获取exam_paper_id对应的题目信息
	// examPaperQuestionSql := `
	// 	SELECT id, exam_paper_id, score, type, content, options, answers,
	// 	       analysis, title, answer_path, test_path, input, output,
	// 	       example, repo, commit_id, creator, create_time, updated_by,
	// 	       update_time, addi, status, "order", group_id, question_attachments_path
	// 	FROM t_exam_paper_question
	// 	WHERE exam_paper_id = $1`
	// examPaperQuestionSql := `
	// 	SELECT id, score, type, content, options, answers,
	// 	       analysis, title, answer_path, test_path, input, output,
	// 	       example, repo, commit_id, creator, create_time, updated_by,
	// 	       update_time, addi, status, "order", group_id, question_attachments_path
	// 	FROM t_exam_paper_question
	// 	WHERE exam_paper_id = $1`
	// rows, err = conn.Query(examPaperQuestionSql, analysis.ExamPaperID)
	// if err != nil {
	// 	z.Error("get exam paper questions for exam paper ID", zap.Error(err))
	// 	return analysis, fmt.Errorf("get exam paper questions for exam paper ID: %w", err)
	// }
	// defer rows.Close()

	// // 返回题目信息
	// var questions []cmn.TExamPaperQuestion
	// 收集题目ID，用于获取学生答案
	// var questionIDs []cmn.NullInt
	var questionIDs []null.Int

	// for rows.Next() {
	// 	var q cmn.TExamPaperQuestion
	// 	err := rows.Scan(
	// 		&q.ID, &q.Score, &q.Type, &q.Content,
	// 		&q.Options, &q.Answers, &q.Analysis, &q.Title, &q.AnswerPath,
	// 		&q.TestPath, &q.Input, &q.Output, &q.Example, &q.Repo,
	// 		&q.CommitID, &q.Creator, &q.CreateTime, &q.UpdatedBy,
	// 		&q.UpdateTime, &q.Addi, &q.Status, &q.Order, &q.GroupID,
	// 		&q.QuestionAttachmentsPath)
	// 	if err != nil {
	// 		return analysis, fmt.Errorf("scan exam paper question for exam paper ID")
	// 	}
	// 	questionIDs = append(questionIDs, q.ID)
	// 	questions = append(questions, q)
	// }
	// if err = rows.Err(); err != nil {
	// 	return analysis, fmt.Errorf("error occurred while iterating exam paper questions for exam paper ID: %w", err)
	// }

	// // z.Debug("exam paper questions", zap.Any("questions", questions))
	// analysis.Questions = questions

	// 拼接 question IDs
	questionIDsStr := make([]string, len(questionIDs))
	for i, id := range questionIDs {
		questionIDsStr[i] = fmt.Sprintf("%d", id.Int64)
	}
	questionIDsSql := strings.Join(questionIDsStr, ", ")

	// 获取学生答案统计
	getStudentAnswerSql := fmt.Sprintf(`
	SELECT
		tsa.answer
	FROM t_student_answers tsa
	WHERE tsa.question_id IN (%s)
	`, questionIDsSql)

	rows, err := conn.Query(ctx, getStudentAnswerSql)
	if err != nil {
		z.Error("get student answers for question ID", zap.Error(err))
		return analysis, fmt.Errorf("get student answers for exam paper ID: %w", err)
	}
	defer rows.Close()

	// 获取学生答案统计
	questionAnswersStats := make(map[null.Int]map[string]int)

	for rows.Next() {
		var Answer JSONText
		err := rows.Scan(&Answer)
		if err != nil {
			z.Error("scan student answer row", zap.Error(err))
			return analysis, fmt.Errorf("scan student answer row: %w", err)
		}
		raw := json.RawMessage(Answer)
		var ans struct {
			Type       string   `json:"type"`
			Ans        []string `json:"answer"`
			QuestionID null.Int `json:"question_id"`
		}
		if err := json.Unmarshal(raw, &ans); err != nil {
			z.Error("unmarshal group failed", zap.Error(err))
			return analysis, fmt.Errorf("unmarshal student answer row: %w", err)
		}
		switch ans.Type {
		case "00":
			if questionAnswersStats[ans.QuestionID] == nil {
				questionAnswersStats[ans.QuestionID] = make(map[string]int)
			}
			for _, answer := range ans.Ans {
				if _, ok := questionAnswersStats[ans.QuestionID][answer]; !ok {
					questionAnswersStats[ans.QuestionID][answer] = 0
				}
				questionAnswersStats[ans.QuestionID][answer]++
			}
		case "02":
			if questionAnswersStats[ans.QuestionID] == nil {
				questionAnswersStats[ans.QuestionID] = make(map[string]int)
			}
			for _, answer := range ans.Ans {
				if _, ok := questionAnswersStats[ans.QuestionID][answer]; !ok {
					questionAnswersStats[ans.QuestionID][answer] = 0
				}
				questionAnswersStats[ans.QuestionID][answer]++
			}
		case "04":
			if questionAnswersStats[ans.QuestionID] == nil {
				questionAnswersStats[ans.QuestionID] = make(map[string]int)
			}
			for _, answer := range ans.Ans {
				if _, ok := questionAnswersStats[ans.QuestionID][answer]; !ok {
					questionAnswersStats[ans.QuestionID][answer] = 0
				}
				questionAnswersStats[ans.QuestionID][answer]++
			}
		}
	}
	// z.Debug("question answers", zap.Any("questionAnswersStats", questionAnswersStats))

	analysis.QuestionAnswersStats = questionAnswersStats

	// 获取学生主观题得分统计
	getSubjectiveScoreSql := fmt.Sprintf(`
	SELECT
		tm.question_id, AVG(tm.score) AS total_score
	FROM t_mark tm
	WHERE tm.question_id IN (%s)
	GROUP BY tm.question_id
	`, questionIDsSql)

	rows, err = conn.Query(ctx, getSubjectiveScoreSql)
	if err != nil {
		z.Error("get student subjective scores for question ID", zap.Error(err))
		return analysis, fmt.Errorf("get student subjective scores for exam paper ID: %w", err)
	}
	defer rows.Close()

	// 获取学生主观题得分统计
	subjectiveScores := make(map[null.Int]float64)

	for rows.Next() {
		var questionID null.Int
		var totalScore float64
		err := rows.Scan(&questionID, &totalScore)
		if err != nil {
			z.Error("scan student subjective score row", zap.Error(err))
			return analysis, fmt.Errorf("scan student subjective score row: %w", err)
		}
		subjectiveScores[questionID] = totalScore
	}
	// z.Debug("subjective scores", zap.Any("subjectiveScores", subjectiveScores))

	analysis.SubjectiveScores = subjectiveScores

	return analysis, nil
}

func GetAnalysisPractice(ctx context.Context, args GradeArgs) (PracticeAnalysis, error) {
	var analysis PracticeAnalysis

	z.Info("---->" + cmn.FncName())

	conn := cmn.GetPgxConn()
	if conn == nil {
		return analysis, fmt.Errorf("%w: ", ErrNilDBConn)
	}

	// 通过paperID获取题目
	examPaperSql := `
		SELECT
			id AS exam_paper_id, question_groups
		FROM t_exam_paper
		WHERE practice_id = $1
	`
	examPaperParams := []any{args.PracticeID}

	rows, err := conn.Query(ctx, examPaperSql, examPaperParams...)
	if err != nil {
		err = fmt.Errorf("get created exam info(practiceID=%v) occurred error: %s", args.PracticeID, err.Error())
		z.Error(err.Error())
		return analysis, err
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(
			&analysis.ExamPaperID,
			&analysis.QuestionGroups,
		)
		// z.Debug("exam analysis info", zap.Any("analysis", analysis))
		if err != nil {
			err = fmt.Errorf("scan created exam info(practiceID=%v) occurred error: %s", args.PracticeID, err.Error())
			z.Error(err.Error())
			return analysis, err
		}
	}
	// examPaperQuestionSql := `
	// 	SELECT id, exam_paper_id, score, type, content, options, answers,
	// 	       analysis, title, answer_path, test_path, input, output,
	// 	       example, repo, commit_id, creator, create_time, updated_by,
	// 	       update_time, addi, status, "order", group_id, question_attachments_path
	// 	FROM t_exam_paper_question
	// 	WHERE exam_paper_id = $1`
	examPaperQuestionSql := `
		SELECT id, score, type, content, options, answers,
		       analysis, title,  input, output,
		       example, repo, commit_id, creator, create_time, updated_by,
		       update_time, addi, status, "order", group_id, question_attachments_path
		FROM t_exam_paper_question
		WHERE exam_paper_id = $1`

	rows, err = conn.Query(ctx, examPaperQuestionSql, analysis.ExamPaperID)
	if err != nil {
		z.Error("get exam paper questions for exam paper ID", zap.Error(err))
		return analysis, fmt.Errorf("get exam paper questions for exam paper ID: %w", err)
	}
	defer rows.Close()

	// 返回题目信息
	var questions []cmn.TExamPaperQuestion
	// 收集题目ID，用于获取学生答案
	// var questionIDs []cmn.NullInt
	var questionIDs []null.Int

	for rows.Next() {
		var q cmn.TExamPaperQuestion
		err := rows.Scan(
			&q.ID, &q.Score, &q.Type, &q.Content,
			&q.Options, &q.Answers, &q.Analysis, &q.Title, &q.Input, &q.Output, &q.Example, &q.Repo,
			&q.CommitID, &q.Creator, &q.CreateTime, &q.UpdatedBy,
			&q.UpdateTime, &q.Addi, &q.Status, &q.Order, &q.GroupID,
			&q.QuestionAttachmentsPath)
		if err != nil {
			return analysis, fmt.Errorf("scan exam paper question for exam paper ID")
		}
		questionIDs = append(questionIDs, q.ID)
		questions = append(questions, q)
	}
	if err = rows.Err(); err != nil {
		return analysis, fmt.Errorf("error occurred while iterating exam paper questions for exam paper ID: %w", err)
	}

	// z.Debug("exam paper questions", zap.Any("questions", questions))
	analysis.Questions = questions

	// 拼接 question IDs
	questionIDsStr := make([]string, len(questionIDs))
	for i, id := range questionIDs {
		questionIDsStr[i] = fmt.Sprintf("%d", id.Int64)
	}
	questionIDsSql := strings.Join(questionIDsStr, ", ")

	// 获取学生答案统计
	getStudentAnswerSql := fmt.Sprintf(`
	SELECT
		tsa.answer
	FROM t_student_answers tsa
	WHERE tsa.question_id IN (%s)
	`, questionIDsSql)

	rows, err = conn.Query(ctx, getStudentAnswerSql)
	if err != nil {
		z.Error("get student answers for question ID", zap.Error(err))
		return analysis, fmt.Errorf("get student answers for exam paper ID: %w", err)
	}
	defer rows.Close()

	// 获取学生答案统计
	questionAnswersStats := make(map[null.Int]map[string]int)

	for rows.Next() {
		var Answer JSONText
		err := rows.Scan(&Answer)
		if err != nil {
			z.Error("scan student answer row", zap.Error(err))
			return analysis, fmt.Errorf("scan student answer row: %w", err)
		}
		raw := json.RawMessage(Answer)
		var ans struct {
			Type       string   `json:"type"`
			Ans        []string `json:"answer"`
			QuestionID null.Int `json:"question_id"`
		}
		if err := json.Unmarshal(raw, &ans); err != nil {
			z.Error("unmarshal group failed", zap.Error(err))
			return analysis, fmt.Errorf("unmarshal student answer row: %w", err)
		}
		switch ans.Type {
		case "00":
			if questionAnswersStats[ans.QuestionID] == nil {
				questionAnswersStats[ans.QuestionID] = make(map[string]int)
			}
			for _, answer := range ans.Ans {
				if _, ok := questionAnswersStats[ans.QuestionID][answer]; !ok {
					questionAnswersStats[ans.QuestionID][answer] = 0
				}
				questionAnswersStats[ans.QuestionID][answer]++
			}
		case "02":
			if questionAnswersStats[ans.QuestionID] == nil {
				questionAnswersStats[ans.QuestionID] = make(map[string]int)
			}
			for _, answer := range ans.Ans {
				if _, ok := questionAnswersStats[ans.QuestionID][answer]; !ok {
					questionAnswersStats[ans.QuestionID][answer] = 0
				}
				questionAnswersStats[ans.QuestionID][answer]++
			}
		case "04":
			if questionAnswersStats[ans.QuestionID] == nil {
				questionAnswersStats[ans.QuestionID] = make(map[string]int)
			}
			for _, answer := range ans.Ans {
				if _, ok := questionAnswersStats[ans.QuestionID][answer]; !ok {
					questionAnswersStats[ans.QuestionID][answer] = 0
				}
				questionAnswersStats[ans.QuestionID][answer]++
			}
		}
	}
	// z.Debug("question answers", zap.Any("questionAnswersStats", questionAnswersStats))

	analysis.QuestionAnswersStats = questionAnswersStats

	// 获取学生主观题得分统计
	getSubjectiveScoreSql := fmt.Sprintf(`
	SELECT
		tm.question_id, AVG(tm.score) AS total_score
	FROM t_mark tm
	WHERE tm.question_id IN (%s)
	GROUP BY tm.question_id
	`, questionIDsSql)

	rows, err = conn.Query(ctx, getSubjectiveScoreSql)
	if err != nil {
		z.Error("get student subjective scores for question ID", zap.Error(err))
		return analysis, fmt.Errorf("get student subjective scores for exam paper ID: %w", err)
	}
	defer rows.Close()

	// 获取学生主观题得分统计
	subjectiveScores := make(map[null.Int]float64)

	for rows.Next() {
		var questionID null.Int
		var totalScore float64
		err := rows.Scan(&questionID, &totalScore)
		if err != nil {
			z.Error("scan student subjective score row", zap.Error(err))
			return analysis, fmt.Errorf("scan student subjective score row: %w", err)
		}
		subjectiveScores[questionID] = totalScore
	}
	// z.Debug("subjective scores", zap.Any("subjectiveScores", subjectiveScores))

	analysis.SubjectiveScores = subjectiveScores

	return analysis, nil
}

func getScoreS(ctx context.Context, tx pgx.Tx, studentID int, examSessionID int, practiceID int) (Map, error) {
	var (
		epid   int64 // 考卷Id
		psid   int64 // 练习生提交Id 大于0则查询练习学生试卷
		eid    int64 // 考生Id 大于0则查询考试学生试卷
		err    error
		vep    *cmn.TVExamPaper
		tepg   map[int64]*cmn.TExamPaperGroup
		eq     map[int64][]*examPaper.ExamQuestion
		result Map
	)

	conn := cmn.GetPgxConn()
	if conn == nil {
		return result, fmt.Errorf("%w: ", ErrNilDBConn)
	}

	if examSessionID <= 0 && practiceID <= 0 {
		err = fmt.Errorf("exam session id or practice id should be greater than 0")
		return result, err
	}
	var sql string

	if examSessionID > 0 {
		sql = `
	SELECT id, exam_paper_id
	FROM t_examinee
    WHERE student_id = $1 AND exam_session_id = $2
	`
		z.Sugar().Debug("this exam")
		err = conn.QueryRow(ctx, sql, studentID, examSessionID).Scan(&eid, &epid)
		if err != nil {
			return result, fmt.Errorf("query student exam paper ID: %w", err)
		}
	}

	if practiceID > 0 {
		sql = `
	SELECT id,exam_paper_id
	FROM t_practice_submissions
	WHERE student_id = $1 AND practice_id = $2
	`
		z.Sugar().Debug("this practice")
		err = conn.QueryRow(ctx, sql, studentID, practiceID).Scan(&psid, &epid)
		if err != nil {
			return result, fmt.Errorf("query student exam paper ID: %w", err)
		}
	}

	z.Sugar().Debug("epid:", epid, "psid:", psid, "eid:", eid)

	vep, tepg, eq, err = examPaper.LoadExamPaperDetailByUserId(ctx, tx, epid, psid, eid, true, true, true)
	z.Sugar().Debug("vep:", vep)
	z.Sugar().Debug("tepg:", tepg)
	z.Sugar().Debug("eq:", eq)
	//result["vep"] = vep
	//result["tepg"] = tepg
	//result["eq"] = eq
	result = Map{}
	result["exam_paper"] = vep
	result["exam_paper_group"] = tepg
	result["exam_question"] = eq
	//rank, err := getRankList(examSessionID)

	rank, err := getSessionScoreRank(ctx, int64(examSessionID))
	result["rank"] = rank
	z.Sugar().Debug("rank:", result["rank"])

	//examInfoMap := Map{}
	//var examSessionInfoMap []Map

	//examSessionInfo, err := getExamSessionInfo(ctx, examID, studentID)
	//if err != nil {
	//	z.Sugar().Errorf("getExamSessionInfo call failed:%v", err)
	//	return nil, fmt.Errorf("getExamSessionInfo call failed:%v", err)
	//}
	//// 这里是已经获取到第一个场次了，必须围绕着第一个场次进行
	//if len(examSessionInfo) > 0 {
	//	// 取出一场考试中第一个场次的信息
	//	var firstExamSession ExamSessionReflect
	//	firstExamSession = examSessionInfo[0]
	//	// 此时已经有第一个场次的信息 包括试卷与场次的关系等等
	//	paper, err := srv.Repo.getExamPaperByExamineeID(ctx, firstExamSession.ExamineeID.Int64, firstExamSession.PaperID.Int64)
	//	if err != nil {
	//		z.Sugar().Errorf("getExamPaperByExamineeID call failed:%v", err)
	//		return nil, fmt.Errorf("getExamPaperByExamineeID call failed:%v", err)
	//	}
	//	// 存储考试基本信息：当前的试卷名
	//	examInfoMap["paperName"] = paper.Name
	//	// 这里能获取到paper的名字了； 但也是仅获取一场
	//	for _, v := range examSessionInfo {
	//		examInfo := Map{
	//			"ID":         v.ID.ValueOrZero(),
	//			"PaperID":    v.PaperID.ValueOrZero(),
	//			"ExamTime":   v.Duration.ValueOrZero(),
	//			"ExamineeID": v.ExamineeID.ValueOrZero(),
	//			"SessionNum": v.SessionNum.ValueOrZero(),
	//		}
	//		examSessionInfoMap = append(examSessionInfoMap, examInfo)
	//	}
	//	// 保存考试场次信息：用于学生二次查询试卷作答详情
	//	result["examSessionInfo"] = examSessionInfoMap
	//
	//	// 这里拿到原题目 需要构建一个关于题目ID的map,用于遍历时取出
	//	questions := paper.Questions
	//	for _, v := range questions {
	//		questionMap[v.ID] = v
	//	}
	//	// 存储考试基本信息：题目总数
	//	examInfoMap["questionNum"] = len(questions)
	//
	//	answerTime, err := srv.Repo.getExamineeAnswerTime(ctx, firstExamSession.ExamineeID.Int64)
	//	if err != nil {
	//		z.Sugar().Errorf("getExamineeAnswerTime call failed:%v", err)
	//		return nil, fmt.Errorf("getExamineeAnswerTime call failed:%v", err)
	//	}
	//	examInfoMap["answerTime"] = answerTime
	//	// 这里获取学生的作答,合并学生作答详情于题目中
	//	answer, err := srv.Repo.getStudentExamDoneAnswer(ctx, firstExamSession.ExamineeID.Int64)
	//	if err != nil {
	//		z.Sugar().Errorf("getStudentExamDoneAnswer call failed:%v", err)
	//		return nil, fmt.Errorf("getStudentExamDoneAnswer call failed:%v", err)
	//	}
	//	//  存储考试基本信息：学生作答总数
	//	examInfoMap["answerNum"] = len(answer)
	//	for _, v := range answer {
	//		question := questionMap[v.QuestionID.Int64]
	//		n := tranformQuestionAnswer(v, question)
	//		newQuestionList = append(newQuestionList, n)
	//	}
	//	// 存储新题目：包括原题目与学生作答
	//	result["questions"] = newQuestionList
	//	// 之后需要获取这个排行榜的信息‘
	//	rank, err := srv.Repo.getSessionScoreRank(ctx, firstExamSession.ID.Int64)
	//	if err != nil {
	//		z.Sugar().Errorf("getSessionScoreRank call failed:%v", err)
	//		return nil, fmt.Errorf("getSessionScoreRank call failed:%v", err)
	//	}
	//	// 这里多获取一个学生ID
	//	for _, v := range rank {
	//		if v.StudentID == studentID {
	//			//  存储考试基本信息：学生总分
	//			examInfoMap["studentScore"] = v.TotalScore.ValueOrZero()
	//			result["student_id"] = v.StudentID
	//			break
	//		}
	//	}
	//	result["examInfo"] = examInfoMap
	//	result["rank"] = rank
	//	// 这里处理这个考试试卷的基本信息：必须是考卷名字，考试的时长
	//	return result, nil
	//} else {
	//	// 若此时根据examID跟student_id都无法获取到场次信息的话，那就直接报错
	//	z.Error("getExamSessionInfo cannot get valid session")
	//	return nil, errors.New("getExamSessionInfo cannot get valid session")
	//}

	z.Sugar().Debug("result:", zap.Any("result", result))

	return result, err
}

// getSessionScoreRank 获取某一场次的考生成绩排行榜 完成
func getSessionScoreRank(ctx context.Context, examSessionID int64) ([]ExamSessionScoreRank, error) {

	var result []ExamSessionScoreRank
	var err error

	// TODO 根据班级条件筛选
	selectSql := `SELECT
	vs.student_id,
    u.official_name, 
    COALESCE(vs.total_score, 0) AS total_score,  
    RANK() OVER (ORDER BY COALESCE(vs.total_score, 0) DESC) AS rank  
FROM v_student_exam_total_score vs
JOIN t_user u ON vs.student_id = u.id
WHERE vs.exam_session_id = $1  
ORDER BY
    COALESCE(vs.total_score, 0) DESC; `

	conn := cmn.GetPgxConn()
	if conn == nil {
		return result, fmt.Errorf("%w: ", ErrNilDBConn)
	}

	rows, err := conn.Query(ctx, selectSql, examSessionID)
	if err != nil {
		z.Sugar().Error("get session score rank failed: %s", err.Error())
		return nil, fmt.Errorf("get session score rank failed: %s", err.Error())
	}
	defer rows.Close()
	// 这里要返回这个自己定义的结构体
	rank := make([]ExamSessionScoreRank, 0)

	// 这里会一直遍历，直到取出所有结果集
	for rows.Next() {
		var r ExamSessionScoreRank
		err := rows.Scan(&r.StudentID, &r.OfficialName, &r.TotalScore, &r.Rank)
		if err != nil {
			return nil, fmt.Errorf("row Scan error: %s", err.Error())
		}
		rank = append(rank, r)
	}
	if err = rows.Err(); err != nil {
		z.Error("row iteration error: %w", zap.Error(err))
		return nil, fmt.Errorf("row iteration error: %s", err.Error())
	}

	return rank, nil
}

//
//func getExamSessionInfo(ctx context.Context, examID int64, studentID int64) ([]ExamSessionReflect, error) {
//	Z.Info("----->" + utils.GetFunctionName())
//	if examID <= 0 {
//		z.Error("examID is nil")
//		return nil, errors.New("examID is nil")
//	}
//
//	if studentID <= 0 {
//		z.Error("studentID is nil")
//		return nil, errors.New("studentID is nil")
//	}
//
//	query := `
//		SELECT
//			es.id, e.exam_paper_id ,es.duration,
//			es.status,
//			e.id as examinee_id,
//			e.start_time,
//			e.end_time,
//			es.session_num
//		FROM t_exam_session es
//		LEFT JOIN t_examinee e ON es.id = e.exam_session_id AND e.student_id = $1 AND e.status != $2
//		WHERE es.exam_id = $3 AND es.status != $4
//		ORDER BY es.session_num
//	`
//
//	rows, err := repo.db.Query(ctx, query, studentID, ExamStatus.Deleted, examID, SessionStatus.Disabled)
//	if err != nil {
//		z.Sugar().Errorf("failed to query exam sessions: %s", err.Error())
//		return nil, fmt.Errorf("failed to query exam sessions: %s", err.Error())
//	}
//	defer rows.Close()
//
//	var sessions []ExamSessionReflect
//
//	// 由于本身就已经带有排序了的，所以第一次直接选取第一个paperID，examineeID即可
//	for rows.Next() {
//		var session ExamSessionReflect
//		err := rows.Scan(
//			&session.ID,
//			&session.PaperID,
//			&session.Duration,
//			&session.Status,
//			&session.ExamineeID,
//			&session.StartTime,
//			&session.EndTime,
//			&session.SessionNum,
//		)
//		if err != nil {
//			z.Sugar().Errorf("failed to scan exam session row: %s", err.Error())
//			return nil, fmt.Errorf("failed to scan exam session row: %s", err.Error())
//		}
//		sessions = append(sessions, session)
//		//z.Info("考生ID" + strconv.Itoa(int(session.ExamineeID.Int64)) + " 考试场次ID" + strconv.Itoa(int(session.ID.Int64)))
//	}
//	if rows.Err() != nil {
//		z.Sugar().Errorf("row iteration error: %s", rows.Err().Error())
//		return nil, fmt.Errorf("row iteration error: %s", rows.Err().Error())
//	}
//
//	return sessions, nil
//}
