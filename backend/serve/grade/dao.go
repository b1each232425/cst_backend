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

// gradeListExam 获取考试成绩列表
/*
* 获取考试成绩列表
* 关键参数说明:
* userID 教师ID
* req 成绩列表请求参数
*  page 页码
*  pageSize 每页数量
*  examID 考试ID
*  practiceID 练习ID
*  filter 筛选条件
*   filter.Name 姓名
*   filter.Type 类型
*   filter.Submitted 是否已提交成绩
* return 考试成绩列表 考试成绩总数 错误信息
 */
func gradeListExam(ctx context.Context, userID int64, req *GradeListReq) ([]GradeExam, int64, error) {
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

	if req.Page <= 0 {
		err = fmt.Errorf("无效页码: 页码必须为正整数")
		z.Error(err.Error())
		return result, rowCount, err
	}
	page = req.Page

	if req.PageSize <= 0 {
		err = fmt.Errorf("无效每页数量: 每页数量必须为正整数")
		z.Error(err.Error())
		return result, rowCount, err
	}
	pageSize = req.PageSize

	if userID <= 0 && userID != -1 {
		err = fmt.Errorf("无效教师ID")
		z.Error(err.Error())
		return result, rowCount, err
	}

	if req.ExamID < 0 {
		err = fmt.Errorf("无效考试ID: 考试ID必须为正整数")
		z.Error(err.Error())
		return result, rowCount, err
	}

	// 查询条件
	whereClause := " WHERE 1=1 "

	// 参数
	params := []any{}

	// 筛选教师：-1 表示所有教师
	if userID > 0 {
		whereClause += fmt.Sprintf(" AND ei.creator=$%d ", len(params)+1)
		params = append(params, userID)
	}

	// 筛选考试：0 表示所有考试
	if req.ExamID > 0 {
		whereClause += fmt.Sprintf(" AND ei.id=$%d ", len(params)+1)
		params = append(params, req.ExamID)
	}

	filter := req.Filter

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

// gradeListPractice 获取练习成绩列表
/*
* 获取练习成绩列表
* 关键参数说明：
*  userID 教师ID
*  req 成绩列表请求参数
*  page 页码
*  pageSize 每页数量
*  practiceID 练习ID
*  filter 筛选条件
*   filter.Name 姓名
* return 练习成绩列表 练习成绩总数 错误信息
 */
func gradeListPractice(ctx context.Context, userID int64, req *GradeListReq) ([]GradePractice, int64, error) {
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

	if req.Page <= 0 {
		err = fmt.Errorf("无效页码: 页码必须为正整数")
		z.Error(err.Error())
		return result, 0, err
	}
	page = req.Page

	if req.PageSize <= 0 {
		err = fmt.Errorf("无效每页数量: 每页数量必须为正整数")
		z.Error(err.Error())
		return result, 0, err
	}
	pageSize = req.PageSize

	if userID <= 0 && userID != -1 {
		err = fmt.Errorf("无效教师ID")
		z.Error(err.Error())
		return result, 0, err
	}

	if req.PracticeID < 0 {
		err = fmt.Errorf("无效练习ID: 练习ID必须为正整数")
		z.Error(err.Error())
		return result, 0, err
	}

	whereClause := " WHERE 1=1 "

	params := []any{}

	if userID > 0 {
		whereClause += fmt.Sprintf(" AND p.creator=$%d ", len(params)+1)
		params = append(params, userID)
	}

	if req.PracticeID > 0 {
		whereClause += fmt.Sprintf(" AND p.practice_id=$%d ", len(params)+1)
		params = append(params, req.PracticeID)
	}

	filter := req.Filter

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

// setExamGradeSubmitted 提交考试成绩
/*
* 提交考试成绩
* 关键参数说明：
*  userID 教师ID
*  examIDs 考试ID列表
* return 影响行数 错误信息
 */
func setExamGradeSubmitted(ctx context.Context, userID int64, examIDs []int) (int64, error) {
	z.Info("---->" + cmn.FncName())

	var err error
	var rowsAffected int64

	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}

	if userID <= 0 {
		err = fmt.Errorf("无效教师ID")
		z.Error(err.Error())
		return 0, err
	}

	if len(examIDs) <= 0 {
		err = fmt.Errorf("无效考试ID，长度为0")
		z.Error(err.Error())
		return 0, err
	}

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

		commandTag, err := tx.Exec(ctx, updateSql, examID, userID, curr)
		if forceErr == "tx exec fail" {
			err = errors.New(forceErr)
		}
		if err != nil {
			err = fmt.Errorf("提交考试成绩失败(examID=%v userID=%v) error: %s", examID, userID, err.Error())
			z.Error(err.Error())
			return 0, err
		}
		rowsAffected = commandTag.RowsAffected()
	}

	txSuccess = true

	return rowsAffected, nil
}

// gradeDistributionExam 返回指定考试ID的成绩分布信息
/*
* 成绩分布接口
* 关键参数说明：
*  examID 考试ID
*  columnNum 列数
* return 考试成绩分布 错误信息
 */
func gradeDistributionExam(ctx context.Context, examID int, columnNum int) (ExamGradeDistribution, error) {

	z.Info("---->" + cmn.FncName())

	var err error
	var result ExamGradeDistribution

	if examID < 0 {
		err = fmt.Errorf("examID无效")
		z.Error(err.Error())
		return result, err
	}

	if columnNum < 1 {
		err = fmt.Errorf("列数无效")
		z.Error(err.Error())
		return result, err
	}

	conn := cmn.GetPgxConn()
	if conn == nil {
		err = fmt.Errorf("获取数据库连接失败")
		z.Error(err.Error())
		return result, err
	}

	whereClause := " WHERE ei.id = $1 "

	distributions := []string{}
	scoreInterval := 1 / float64(columnNum)

	initAddiCondition := ""
	if columnNum == 1 {
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
		)`, scoreInterval*float64(columnNum-1), 1, initAddiCondition)
	distributions = append(distributions, initDistribution)

	for i := columnNum - 1; i > 0; i-- {
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

	sql := fmt.Sprintf(`
	WITH exam_session_grades AS ( SELECT
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
		GROUP BY sets.exam_id, sets.exam_session_id, ep.id, ep.name, ep.total_score
	)
	SELECT
		ei.id,
		ei.name,
		jsonb_agg(exam_session_grades) AS exam_session_score_distribution
	FROM t_exam_info ei
		JOIN exam_session_grades ON exam_session_grades.exam_id = ei.id
	%s
	GROUP BY ei.id
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

// gradeDistributionPractice 返回指定练习ID的成绩分布信息
/*
* 关键参数说明：
*  practiceID 练习ID
*  columnNum 列数
* return 练习成绩分布 错误信息
 */
func gradeDistributionPractice(ctx context.Context, practiceID int, columnNum int) (PracticeGradeDistribution, error) {

	z.Info("---->" + cmn.FncName())

	var err error

	result := PracticeGradeDistribution{}

	if practiceID <= 0 {
		err = fmt.Errorf("练习ID无效")
		z.Error(err.Error())
		return result, err
	}

	if columnNum < 1 {
		err = fmt.Errorf("列数无效")
		z.Error(err.Error())
		return result, err
	}

	conn := cmn.GetPgxConn()
	if conn == nil {
		err = fmt.Errorf("获取数据库连接失败")
		z.Error(err.Error())
		return result, err
	}

	whereClause := " WHERE practices.id = $1 "

	distributions := []string{}

	scoreInterval := 1 / float64(columnNum)

	initAddiCondition := ""

	if columnNum == 1 {
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
		)`, scoreInterval*float64(columnNum-1), 1, initAddiCondition)

	distributions = append(distributions, initDistribution)

	for i := columnNum - 1; i > 0; i-- {

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

// gradeExamineeListExam 按考试ID分类返回考生成绩列表，支持导出功能
/*
* 关键参数说明：
*  req 成绩列表请求参数
	examID 考试ID列表
	practiceID 练习ID列表
	page 页码
	pageSize 每页数量
	filter 过滤条件
	filter.Keyword 关键词
* return 考试成绩导出响应 错误信息
*/
func gradeExamineeListExam(ctx context.Context, req GradeExamineeListReq) (ExamScoreExportResponse, int64, error) {
	z.Info("---->" + cmn.FncName())

	// forceErr := ""
	// if val := ctx.Value("force-error"); val != nil {
	// 	forceErr = val.(string)
	// }

	var err error
	var response ExamScoreExportResponse

	examID := req.ExamID
	page := req.Page
	pageSize := req.PageSize
	filter := req.Filter
	params := []any{}
	totalCount := int64(0)

	if len(examID) <= 0 {
		err = fmt.Errorf("考试ID长度小于等于0")
		z.Error(err.Error())
		return response, totalCount, err
	}
	if page <= 0 {
		err = fmt.Errorf("页码小于等于0")
		z.Error(err.Error())
		return response, totalCount, err
	}
	if pageSize <= 0 {
		err = fmt.Errorf("每页数量小于等于0")
		z.Error(err.Error())
		return response, totalCount, err
	}

	// 多个考试ID
	placeholders := make([]string, len(examID))
	for i, id := range examID {
		placeholders[i] = fmt.Sprintf("$%d", len(params)+1)
		params = append(params, id)
	}
	placeholderStr := strings.Join(placeholders, ", ")
	whereClause := fmt.Sprintf("WHERE ets.exam_id IN (%s) ", placeholderStr)

	// 姓名,昵称,电话模糊搜索
	if filter.Keyword != "" {
		whereClause += fmt.Sprintf(" AND (e.official_name::text ILIKE $%d ", len(params)+1)
		params = append(params, fmt.Sprintf("%%%s%%", filter.Keyword))

		whereClause += fmt.Sprintf(" OR e.mobile_phone::text ILIKE $%d ", len(params)+1)
		params = append(params, fmt.Sprintf("%%%s%%", filter.Keyword))

		whereClause += fmt.Sprintf(" OR e.account::text ILIKE $%d)", len(params)+1)
		params = append(params, fmt.Sprintf("%%%s%%", filter.Keyword))
	}

	sql := fmt.Sprintf(`
	SELECT
		ets.exam_id,
		ei.name AS exam_name,
		e.student_id,
		e.mobile_phone AS phone,
		e.official_name AS name,
		e.account AS nickname,
		STRING_AGG(COALESCE(e.remark, ''), '') AS remark,
		jsonb_agg(jsonb_build_object(
				'exam_id', ets.exam_id,
				'exam_session_id', ets.exam_session_id,
				'score', ets.total_score
			) ORDER BY ets.exam_session_id) AS exam_sessions
	FROM v_student_exam_total_score ets
		JOIN t_exam_info ei ON ei.id = ets.exam_id
		JOIN v_examinee_info e ON e.student_id = ets.student_id
	%s
	GROUP BY
		ets.exam_id, ei.name, e.student_id, e.mobile_phone, e.official_name, e.account
	ORDER BY
		ets.exam_id, e.official_name
	`, whereClause)

	var listSQL string
	var listParams []any

	if page == -1 && pageSize == -1 {
		listSQL = sql
		listParams = params
	} else {
		listParams = params
		listSQL = fmt.Sprintf(`%s
		LIMIT $%d OFFSET $%d`,
			sql, len(listParams)+1, len(listParams)+2)
		listParams = append(listParams, int32(pageSize), int32((page-1)*pageSize))
	}

	conn := cmn.GetPgxConn()
	if conn == nil {
		err = fmt.Errorf("获取数据库连接为空")
		z.Error(err.Error())
		return response, totalCount, err
	}

	rows, err := conn.Query(ctx, listSQL, listParams...)
	if err != nil {
		err = fmt.Errorf("查询考试考生成绩列表失败 错误: %s", err.Error())
		z.Error(err.Error())
		return response, totalCount, err
	}
	defer rows.Close()

	// 按examID分组
	examMap := make(map[int64]*ExamScoreExportData)

	for rows.Next() {
		var examID int64
		var examName null.String
		var student StudentExamScoreInfo

		err = rows.Scan(
			&examID,
			&examName,
			&student.StudentID,
			&student.Phone,
			&student.Name,
			&student.Nickname,
			&student.Remark,
			&student.ExamSessions,
		)
		if err != nil {
			err = fmt.Errorf("扫描考试考生成绩列表失败 错误: %w", err)
			z.Error(err.Error())
			return response, totalCount, err
		}

		// 按examID分组
		if examData, exists := examMap[examID]; exists {
			examData.StudentScores = append(examData.StudentScores, student)
		} else {
			examMap[examID] = &ExamScoreExportData{
				ExamID:        examID,
				ExamName:      examName,
				StudentScores: []StudentExamScoreInfo{student},
			}
		}
		totalCount++
	}

	for _, examData := range examMap {
		response.Exams = append(response.Exams, *examData)
	}
	response.Total = totalCount

	return response, totalCount, nil
}

// gradeExamineeListPractice 按练习ID分类返回考生练习成绩列表，支持导出功能
/*
关键参数说明：
	req 成绩列表请求参数
	practiceID 练习ID列表
	page 页码
	pageSize 每页数量
	filter 过滤条件
	filter.Keyword 关键词
return 练习成绩导出响应 错误信息
*/
func gradeExamineeListPractice(ctx context.Context, req GradeExamineeListReq) (PracticeScoreExportResponse, int64, error) {
	z.Info("---->" + cmn.FncName())

	var err error
	var response PracticeScoreExportResponse

	practiceID := req.PracticeID
	page := req.Page
	pageSize := req.PageSize
	filter := req.Filter
	params := []any{}
	totalCount := int64(0)

	// forceErr := ""
	// if val := ctx.Value("force-error"); val != nil {
	// 	forceErr = val.(string)
	// }

	if len(practiceID) <= 0 {
		err = fmt.Errorf("练习ID列表为空")
		z.Error(err.Error())
		return response, totalCount, err
	}
	if page <= 0 {
		err = fmt.Errorf("页码小于等于0")
		z.Error(err.Error())
		return response, totalCount, err
	}
	if pageSize <= 0 {
		err = fmt.Errorf("每页数量小于等于0")
		z.Error(err.Error())
		return response, totalCount, err
	}

	// 多个练习ID
	placeholders := make([]string, len(practiceID))
	for i, id := range practiceID {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		params = append(params, id)
	}
	placeholderStr := strings.Join(placeholders, ", ")
	whereClause := fmt.Sprintf("WHERE pts.practice_id IN (%s) ", placeholderStr)

	// 关键词过滤
	if filter.Keyword != "" {
		whereClause += fmt.Sprintf(" AND (ei.official_name::text ILIKE $%d ", len(params)+1)
		params = append(params, fmt.Sprintf("%%%s%%", filter.Keyword))

		whereClause += fmt.Sprintf(" OR ei.mobile_phone::text ILIKE $%d ", len(params)+1)
		params = append(params, fmt.Sprintf("%%%s%%", filter.Keyword))

		whereClause += fmt.Sprintf(" OR ei.account::text ILIKE $%d)", len(params)+1)
		params = append(params, fmt.Sprintf("%%%s%%", filter.Keyword))
	}

	sql := fmt.Sprintf(`
	SELECT
		pts.practice_id,
		p.name AS practice_name,
		pts.student_id,
		ei.mobile_phone AS phone,
		ei.official_name AS name,
		ei.account AS nickname,
		MAX(pts.total_score) AS highest_score,
		COUNT(DISTINCT pts.id) AS submitted_cnt
	FROM v_student_practice_total_score pts
		LEFT JOIN v_examinee_info ei ON ei.student_id = pts.student_id
		LEFT JOIN t_practice p ON p.id = pts.practice_id
	%s
	GROUP BY pts.practice_id, p.name, pts.student_id, ei.mobile_phone, ei.official_name, ei.account
	ORDER BY pts.practice_id, pts.student_id
		`, whereClause)

	var listSQL string
	var listParams []any

	if req.Page == -1 && req.PageSize == -1 {
		listSQL = sql
		listParams = params
	} else {
		listParams = params
		listSQL = fmt.Sprintf(`%s
	LIMIT $%d OFFSET $%d`,
			sql, len(listParams)+1, len(listParams)+2)
		listParams = append(listParams, int32(req.PageSize), int32((req.Page-1)*req.PageSize))
	}

	conn := cmn.GetPgxConn()
	if conn == nil {
		err = fmt.Errorf("获取数据库连接为空")
		z.Error(err.Error())
		return response, totalCount, err
	}

	rows, err := conn.Query(ctx, listSQL, listParams...)
	if err != nil {
		err = fmt.Errorf("查询练习考生成绩列表失败 错误: %w", err)
		z.Error(err.Error())
		return response, totalCount, err
	}

	defer rows.Close()
	practiceMap := make(map[int64]*PracticeScoreExportData)

	for rows.Next() {
		var practiceID int64
		var practiceName null.String
		var scoreInfo PracticeExamineeScoreInfo
		err = rows.Scan(
			&practiceID,
			&practiceName,
			&scoreInfo.StuID,
			&scoreInfo.Phone,
			&scoreInfo.Name,
			&scoreInfo.Nickname,
			&scoreInfo.HighestScore,
			&scoreInfo.SubmittedCnt,
		)
		if err != nil {
			err = fmt.Errorf("查询练习考生成绩列表失败 错误: %w", err)
			z.Error(err.Error())
			return response, totalCount, err
		}

		scoreInfo.PracticeID = null.IntFrom(practiceID)

		if practiceData, exists := practiceMap[practiceID]; exists {
			practiceData.StudentScores = append(practiceData.StudentScores, scoreInfo)
		} else {
			practiceMap[practiceID] = &PracticeScoreExportData{
				PracticeID:    practiceID,
				PracticeName:  practiceName,
				StudentScores: []PracticeExamineeScoreInfo{scoreInfo},
			}
		}
		totalCount++
	}

	for _, practiceData := range practiceMap {
		response.Practices = append(response.Practices, *practiceData)
	}

	return response, totalCount, err
}

// gradeAnalysis 获取考卷分析
/*
	关键参数说明：
		esid 考试场次ID
		pid 练习ID
	返回参数说明: 试卷分析 错误信息
*/
func gradeAnalysisByID(ctx context.Context, esid int64, pid int64) (Analysis, error) {
	z.Info("---->" + cmn.FncName())

	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}

	var analysis Analysis
	var err error

	if esid <= 0 && pid <= 0 {
		err = fmt.Errorf("考试场次ID 和 练习ID 不能同时为非正整数, (examSessionID=%v),(practiceID=%v)", esid, pid)
		z.Error(err.Error())
		return analysis, err
	}

	conn := cmn.GetPgxConn()
	if conn == nil {
		err = fmt.Errorf("获取数据库连接为空")
		z.Error(err.Error())
		return analysis, err
	}

	var tx pgx.Tx
	tx, err = conn.Begin(context.Background())
	if err != nil || forceErr == "conn begin tx fail" {
		err = fmt.Errorf("开启事务失败: %w", err)
		z.Error(err.Error())
		return analysis, err
	}

	// 第一步：获取要分析的考卷ID，exam_paper_id
	var id int64
	var epSql string
	if pid > 0 {
		epSql = `SELECT id AS exam_paper_id FROM t_exam_paper WHERE practice_id = $1`
		id = pid
	}
	if esid > 0 {
		epSql = `SELECT id AS exam_paper_id FROM t_exam_paper WHERE exam_session_id = $1`
		id = esid
	}

	err = conn.QueryRow(ctx, epSql, id).Scan(&analysis.ExamPaperID)
	if err != nil {
		err = fmt.Errorf("获取考卷ID失败: %w,(examSessionID=%v),(practiceID=%v)", err, esid, pid)
		z.Error(err.Error())
		return analysis, err
	}

	// 第二步：通过考卷ID获取考卷内容，考卷信息，考卷题组，考卷题目
	analysis.ExamPaper, analysis.ExamPaperGroup, analysis.ExamPaperQuestion, err = examPaper.LoadExamPaperDetailsById(ctx, tx, analysis.ExamPaperID, true, true)
	if err != nil {
		err = fmt.Errorf("获取考卷内容失败: %w,(examSessionID=%v),(practiceID=%v),(examPaperID=%v)", err, esid, pid, analysis.ExamPaperID)
		z.Error(err.Error())
		return analysis, err
	}

	// 第三步：获取考卷所有的题目序号，用于获取考生作答答案
	var qids []null.Int
	for _, v := range analysis.ExamPaperQuestion {
		for _, q := range v {
			qids = append(qids, q.ID)
		}
	}
	qidsStr := make([]string, len(qids))
	for i, id := range qids {
		qidsStr[i] = fmt.Sprintf("%d", id.Int64)
	}
	qidsSql := strings.Join(qidsStr, ", ")

	// 获取学生答案
	studentAnswerSql := fmt.Sprintf(`
	SELECT
		type, answer, question_id
	FROM t_student_answers tsa
	WHERE question_id IN (%s)
	`, qidsSql)

	rows, err := conn.Query(ctx, studentAnswerSql)
	if err != nil {
		err = fmt.Errorf("查询学生答案失败: %w,(examSessionID=%v),(practiceID=%v),(examPaperID=%v),(qids=%s)", err, esid, pid, analysis.ExamPaperID, qidsStr)
		z.Error(err.Error())
		return analysis, err
	}
	defer rows.Close()

	// 第四步:从学生选择题答案中进行统计
	type AnswerData struct {
		Answer []string `json:"answer"`
	}
	questionAnswersStats := make(map[null.Int]map[string]int)
	for rows.Next() {
		var ans struct {
			Type       string   `json:"type"`
			AnsJson    JSONText `json:"answer"`
			QuestionID null.Int `json:"question_id"`
		}

		err = rows.Scan(&ans.Type, &ans.AnsJson, &ans.QuestionID)
		if err != nil {
			err = fmt.Errorf("查询学生答案失败: %w,(examSessionID=%v),(practiceID=%v),(examPaperID=%v),(qids=%s)", err, esid, pid, analysis.ExamPaperID, qidsStr)
			z.Error(err.Error())
			return analysis, err
		}

		// 不是单选、多选、判断题型
		if ans.Type != "00" && ans.Type != "02" && ans.Type != "04" {
			continue
		}

		// 数据库存储学生答案为空
		if ans.AnsJson == nil {
			z.Warn("数据库存储学生答案为空, 请检查")
			continue
		}

		var answerData AnswerData
		err = json.Unmarshal(ans.AnsJson, &answerData)
		if err != nil {
			return analysis, fmt.Errorf("unmarshal student answer: %w", err)
		}

		if err != nil {
			err = fmt.Errorf("反序列化学生答案失败: %w,(examSessionID=%v),(practiceID=%v),(examPaperID=%v),(qids=%s),(ans=%s)", err, esid, pid, analysis.ExamPaperID, qidsStr, ans.AnsJson)
			z.Error(err.Error())
			return analysis, err
		}

		// 该题目答案还没有出现过
		if questionAnswersStats[ans.QuestionID] == nil {
			questionAnswersStats[ans.QuestionID] = make(map[string]int)
		}

		// 统计答案
		// 说明:选择题答案是数组形式的选项
		for _, answer := range answerData.Answer {
			if _, ok := questionAnswersStats[ans.QuestionID][answer]; !ok {
				questionAnswersStats[ans.QuestionID][answer] = 0
			}
			questionAnswersStats[ans.QuestionID][answer]++
		}
	}
	analysis.QuestionAnswersStats = questionAnswersStats

	// 获取考生得分统计
	getSubjectiveScoreSql := fmt.Sprintf(`
	SELECT
		tm.question_id, AVG(tm.score) AS total_score
	FROM t_mark tm
	WHERE tm.question_id IN (%s)
	GROUP BY tm.question_id
	`, qidsSql)

	rows, err = conn.Query(ctx, getSubjectiveScoreSql)
	if err != nil {
		err = fmt.Errorf("获取考卷平均分失败: %w,(examSessionID=%v),(practiceID=%v),(examPaperID=%v),(qids=%s)", err, esid, pid, analysis.ExamPaperID, qidsStr)
		z.Error(err.Error())
		return analysis, err
	}
	defer rows.Close()

	// 第五步:从学生题目得分中统计平均分
	subjectiveScores := make(map[null.Int]float64)
	for rows.Next() {
		var qid null.Int
		var avgScore float64
		err := rows.Scan(&qid, &avgScore)
		if err != nil {
			err = fmt.Errorf("获取考卷平均分失败: %w,(examSessionID=%v),(practiceID=%v),(examPaperID=%v),(qids=%s),(qid=%v)", err, esid, pid, analysis.ExamPaperID, qidsStr, qid)
			z.Error(err.Error())
			return analysis, err
		}
		subjectiveScores[qid] = avgScore
	}
	analysis.SubjectiveScores = subjectiveScores

	return analysis, nil
}

// getScoreS
/*
	关键参数说明：
		ctx 上下文
		tx 数据库事务
		studentID 学生ID
		examSessionID 考试场次ID
		practiceID 练习ID
	return 成绩响应 错误信息

*/
func getScoreS(ctx context.Context, tx pgx.Tx, studentID int64, examSessionID int, practiceID int) (Map, error) {
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
		return result, fmt.Errorf("%w: ", "获取数据库连接为空")
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
	result = Map{}
	result["exam_paper"] = vep
	result["exam_paper_group"] = tepg
	result["exam_question"] = eq

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
		return result, fmt.Errorf("%w: ", "获取数据库连接为空")
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
