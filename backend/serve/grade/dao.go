package grade

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
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
	analysis.ExamPaper, analysis.ExamPaperGroup, analysis.ExamPaperQuestion, err = examPaper.LoadExamPaperDetailsById(ctx, tx, analysis.ExamPaperID, true, true, true)
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
func getScoreS(ctx context.Context, tx pgx.Tx, studentID, examSessionID, practiceID int64) (Map, error) {
	var (
		epid               int64 // 考卷Id
		psid               int64 // 练习生提交Id 大于0则查询练习学生试卷
		eid                int64 // 考生Id 大于0则查询考试学生试卷
		err                error
		vep                *cmn.TVExamPaper
		tepg               map[int64]*cmn.TExamPaperGroup
		eq                 map[int64][]*examPaper.ExamQuestion
		result             Map
		examInfoMap        Map
		examSessionInfoMap []Map
	)
	result = Map{}
	examInfoMap = Map{}

	conn := cmn.GetPgxConn()
	if conn == nil {
		err = fmt.Errorf("获取数据库连接为空")
		z.Error(err.Error())
		return result, err
	}

	// 校验
	if examSessionID <= 0 && practiceID <= 0 {
		err = fmt.Errorf("考试场次ID或练习ID不能为空(examSessionID=%v, practiceID=%v)", examSessionID, practiceID)
		z.Error(err.Error())
		return result, err
	}
	if studentID <= 0 {
		err = fmt.Errorf("学生ID不能为空(studentID=%v)", studentID)
		z.Error(err.Error())
		return result, err
	}

	// 第一步：获取考卷ID和考试ID/练习ID
	var sql string
	if examSessionID > 0 {
		sql = `
	SELECT id, exam_paper_id
	FROM t_examinee
    WHERE student_id = $1 AND exam_session_id = $2
	`
		err = conn.QueryRow(ctx, sql, studentID, examSessionID).Scan(&eid, &epid)
		if err != nil {
			err = fmt.Errorf("查询考试学生试卷ID失败: (examSessionID=%v, studentID=%v) %w", examSessionID, studentID, err)
			z.Error(err.Error())
			return result, err
		}
	}
	if practiceID > 0 {
		sql = `
	SELECT id,exam_paper_id
	FROM t_practice_submissions
	WHERE student_id = $1 AND practice_id = $2
	`
		err = conn.QueryRow(ctx, sql, studentID, practiceID).Scan(&psid, &epid)
		if err != nil {
			err = fmt.Errorf("查询练习学生试卷ID失败: %w", err)
			z.Error(err.Error())
			return result, err
		}
	}
	z.Sugar().Debugf("eid: %d, psid: %d, epid: %d", eid, psid, epid)

	// 第二步：获取考卷信息：考卷、题组、题目
	vep, tepg, eq, err = examPaper.LoadExamPaperDetailByUserId(ctx, tx, epid, psid, eid, true, true, true)
	if err != nil {
		err = fmt.Errorf("调用LoadExamPaperDetailByUserId失败:%w", err)
		z.Error(err.Error())
		return result, err
	}

	result["exam_paper"] = vep
	result["exam_paper_group"] = tepg
	result["exam_question"] = eq

	// 第三步：获取考试场次成绩排行榜
	rank, err := getSessionScoreRank(ctx, examSessionID)
	if err != nil {
		err = fmt.Errorf("getSessionScoreRank 失败:%w", err)
		z.Error(err.Error())
		return result, err
	}
	result["rank"] = rank
	z.Sugar().Debug("rank: %v", rank)

	// 第五步：获取学生ID
	for _, v := range rank {
		if v.StudentID == studentID {
			//  存储考试基本信息：学生总分
			examInfoMap["StudentScore"] = v.TotalScore.ValueOrZero()
			result["student_id"] = v.StudentID
			z.Sugar().Debug("result[student_id]: %v", result["student_id"])
			break
		}
	}

	examSessionInfo, err := getExamSessionInfo(ctx, examSessionID, studentID)
	if err != nil {
		err = fmt.Errorf("getExamSessionInfo 失败:%w", err)
		z.Error(err.Error())
		return result, err
	}
	z.Sugar().Debug("examSessionInfo: %v", examSessionInfo)

	// 第五步：获取考试场次信息
	for _, v := range examSessionInfo {
		examInfo := Map{
			"ID":         v.ID.ValueOrZero(),
			"PaperID":    v.PaperID.ValueOrZero(),
			"ExamTime":   v.Duration.ValueOrZero(),
			"ExamineeID": v.ExamineeID.ValueOrZero(),
			"SessionNum": v.SessionNum.ValueOrZero(),
		}
		examSessionInfoMap = append(examSessionInfoMap, examInfo)
		z.Sugar().Debug("examInfo: %v", examInfo)
	}
	// 保存考试场次信息：用于学生二次查询试卷作答详情
	result["examSessionInfo"] = examSessionInfoMap

	// 第四步：获取考试答题信息
	answerNum, err := getStudentExamDoneAnswer(ctx, eid)
	if err != nil {
		err = fmt.Errorf("getStudentExamDoneAnswer 失败:%w", err)
		z.Error(err.Error())
		return result, err
	}
	z.Sugar().Debug("answerNum: %v", answerNum)
	//  存储考试基本信息：学生作答总数
	examInfoMap["AnswerNum"] = len(answerNum)
	answerTime, err := getExamineeAnswerTime(ctx, eid)
	if err != nil {
		err = fmt.Errorf("getExamineeAnswerTime 失败:%w", err)
		z.Error(err.Error())
		return result, err

	}
	z.Sugar().Debug("answerTime: %v", answerTime)
	examInfoMap["AnswerTime"] = answerTime

	result["examInfo"] = examInfoMap
	result["examSessionInfo"] = examSessionInfoMap
	z.Sugar().Debug("result: %v", result)
	z.Sugar().Debug("examInfoMap: %v", examInfoMap)
	z.Sugar().Debug("examSessionInfoMap: %v", examSessionInfoMap)

	return result, err
}

// getSessionScoreRank 获取某一场次的考生成绩排行榜 完成
func getSessionScoreRank(ctx context.Context, examSessionID int64) ([]ExamSessionScoreRank, error) {

	rank := make([]ExamSessionScoreRank, 0)
	var err error

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
		err = fmt.Errorf("获取数据库连接为空")
		z.Error(err.Error())
		return rank, err
	}

	rows, err := conn.Query(ctx, selectSql, examSessionID)
	if err != nil {
		err = fmt.Errorf("查询考试场次成绩失败: %w", err)
		z.Error(err.Error())
		return rank, err
	}
	defer rows.Close()
	// 这里要返回这个自己定义的结构体

	// 这里会一直遍历，直到取出所有结果集
	for rows.Next() {
		var r ExamSessionScoreRank
		err = rows.Scan(&r.StudentID, &r.OfficialName, &r.TotalScore, &r.Rank)
		if err != nil {
			err = fmt.Errorf("扫描考试场次成绩失败: %w", err)
			z.Error(err.Error())
			return rank, err
		}
		rank = append(rank, r)
	}
	if err = rows.Err(); err != nil {
		err = fmt.Errorf("遍历考试场次成绩失败: %w", err)
		z.Error(err.Error())
		return rank, err
	}

	return rank, nil
}

// 获取学生本张试卷的学生的作答情况答案及其分数
func getStudentExamDoneAnswer(ctx context.Context, examineeID int64) ([]cmn.TStudentAnswers, error) {
	var AllAnswers []cmn.TStudentAnswers
	var err error

	selectSql := `SELECT id, type ,question_id,answer,answer_score,addi,status FROM t_student_answers WHERE examinee_id = $1`

	conn := cmn.GetPgxConn()
	if conn == nil {
		err = fmt.Errorf("获取数据库连接为空")
		z.Error(err.Error())
		return AllAnswers, err
	}

	//根据考生ID去查询此时的数据
	rows, err := conn.Query(ctx, selectSql, examineeID)
	if err != nil {
		z.Error("student_exam_answer/service SaveStudentExamAnswer", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	// 一个答案算一次,插入到这个结构体数组中
	for rows.Next() {
		var r cmn.TStudentAnswers
		err := rows.Scan(&r.ID, &r.Type, &r.QuestionID, &r.Answer, &r.AnswerScore, &r.Addi, &r.Status)
		if err != nil {
			z.Error("student_exam_answer/service SaveStudentExamAnswer getAnswerByExamineeID error", zap.Error(err))
			return nil, err
		}
		AllAnswers = append(AllAnswers, r)
	}
	return AllAnswers, nil
}

func getExamineeAnswerTime(ctx context.Context, examineeID int64) (int64, error) {
	var err error

	if examineeID <= 0 {
		z.Warn("invalid input params examineeID")
		return -1, errors.New("invalid input params examineeID")
	}
	selectSql := `SELECT
	start_time,end_time
	FROM t_examinee
	WHERE id = $1
	`
	conn := cmn.GetPgxConn()
	if conn == nil {
		err = fmt.Errorf("获取数据库连接为空")
		z.Error(err.Error())
		return -1, err
	}

	var start, end null.Int
	err = conn.QueryRow(ctx, selectSql, examineeID).Scan(&start, &end)
	if err != nil {
		z.Sugar().Errorf("query examinee answer time failed:%v", err)
		return -1, fmt.Errorf("query examinee answer time failed:%v", err)
	}
	if !start.Valid || !end.Valid {
		z.Warn("student answer time is NULL ;data is invalid")
		return 0, nil
	}
	answerTime := end.Int64 - start.Int64
	return answerTime, nil

}

type ExamSessionReflect struct {
	ID         null.Int    `json:"ID,omitempty" `
	PaperID    null.Int    `json:"PaperID,omitempty" `
	StartTime  null.Int    `json:"StartTime,omitempty" `
	EndTime    null.Int    `json:"EndTime,omitempty" `
	Duration   null.Int    `json:"Duration,omitempty" `
	Status     null.String `json:"Status,omitempty" `
	SessionNum null.Int    `json:"SessionNum,omitempty" `
	ExamineeID null.Int    `json:"ExamineeID,omitempty"`
}

// 根据考试ID与学生ID，获取本次考试的很多信息了；包括试卷id，考生id，场次顺序等等；并且这些都应该是已批改的状态;这里面包含的信息，取出其中一个
func getExamSessionInfo(ctx context.Context, examSessionID int64, studentID int64) ([]ExamSessionReflect, error) {

	var err error
	if examSessionID <= 0 {
		z.Error("examID is nil")
		return nil, errors.New("examID is nil")
	}

	if studentID <= 0 {
		z.Error("studentID is nil")
		return nil, errors.New("studentID is nil")
	}

	conn := cmn.GetPgxConn()
	if conn == nil {
		err = fmt.Errorf("获取数据库连接为空")
		z.Error(err.Error())
		return []ExamSessionReflect{}, err
	}

	// 取出examID
	examSql := `
	SELECT es.exam_id
	FROM t_exam_session es
	WHERE es.id = $1
	`

	var examID int64
	err = conn.QueryRow(ctx, examSql, examSessionID).Scan(&examID)
	if err != nil {
		z.Sugar().Errorf("failed to query exam ID: %s", err.Error())
		return nil, fmt.Errorf("failed to query exam ID: %s", err.Error())
	}

	query := `
		SELECT 
			es.id, e.exam_paper_id ,es.duration,
			es.status,
			e.id as examinee_id,
			e.start_time,
			e.end_time,
			es.session_num
		FROM t_exam_session es
		LEFT JOIN t_examinee e ON es.id = e.exam_session_id AND e.student_id = $1 AND e.status != $2
		WHERE es.exam_id = $3 AND es.status != $4
		ORDER BY es.session_num
	`

	rows, err := conn.Query(ctx, query, studentID, "08", examID, "06")
	if err != nil {
		z.Sugar().Errorf("failed to query exam sessions: %s", err.Error())
		return nil, fmt.Errorf("failed to query exam sessions: %s", err.Error())
	}
	defer rows.Close()

	var sessions []ExamSessionReflect

	// 由于本身就已经带有排序了的，所以第一次直接选取第一个paperID，examineeID即可
	for rows.Next() {
		var session ExamSessionReflect
		err := rows.Scan(
			&session.ID,
			&session.PaperID,
			&session.Duration,
			&session.Status,
			&session.ExamineeID,
			&session.StartTime,
			&session.EndTime,
			&session.SessionNum,
		)
		if err != nil {
			z.Sugar().Errorf("failed to scan exam session row: %s", err.Error())
			return nil, fmt.Errorf("failed to scan exam session row: %s", err.Error())
		}
		sessions = append(sessions, session)
	}
	if rows.Err() != nil {
		z.Sugar().Errorf("row iteration error: %s", rows.Err().Error())
		return nil, fmt.Errorf("row iteration error: %s", rows.Err().Error())
	}

	return sessions, nil
}

func getScoreSPractice(ctx context.Context, tx pgx.Tx, studentID int64, examSessionID int64, practiceID int64) (Map, error) {
	var (
		epid            int64 // 考卷Id
		psid            int64 // 练习生提交Id 大于0则查询练习学生试卷
		eid             int64 // 考生Id 大于0则查询考试学生试卷
		err             error
		vep             *cmn.TVExamPaper
		tepg            map[int64]*cmn.TExamPaperGroup
		eq              map[int64][]*examPaper.ExamQuestion
		result          Map
		practiceInfoMap Map
	)
	result = Map{}
	practiceInfoMap = Map{}

	conn := cmn.GetPgxConn()
	if conn == nil {
		err = fmt.Errorf("获取数据库连接为空")
		z.Error(err.Error())
		return result, err
	}

	// 校验
	if examSessionID <= 0 && practiceID <= 0 {
		err = fmt.Errorf("考试场次ID或练习ID不能为空(examSessionID=%v, practiceID=%v)", examSessionID, practiceID)
		z.Error(err.Error())
		return result, err
	}
	if studentID <= 0 {
		err = fmt.Errorf("学生ID不能为空(studentID=%v)", studentID)
		z.Error(err.Error())
		return result, err
	}

	// 第一步：获取考卷ID和考试ID/练习ID
	var sql string
	if examSessionID > 0 {
		sql = `
	SELECT id, exam_paper_id
	FROM t_examinee
    WHERE student_id = $1 AND exam_session_id = $2
	`
		err = conn.QueryRow(ctx, sql, studentID, examSessionID).Scan(&eid, &epid)
		if err != nil {
			err = fmt.Errorf("查询考试学生试卷ID失败: (examSessionID=%v, studentID=%v) %w", examSessionID, studentID, err)
			z.Error(err.Error())
			return result, err
		}
	}
	if practiceID > 0 {
		sql = `
	SELECT id,exam_paper_id
	FROM t_practice_submissions
	WHERE student_id = $1 AND practice_id = $2
	`
		err = conn.QueryRow(ctx, sql, studentID, practiceID).Scan(&psid, &epid)
		if err != nil {
			err = fmt.Errorf("查询练习学生试卷ID失败: %w", err)
			z.Error(err.Error())
			return result, err
		}
	}
	z.Sugar().Debugf("eid: %d, psid: %d, epid: %d", eid, psid, epid)

	// 第二步：获取考卷信息：考卷、题组、题目
	vep, tepg, eq, err = examPaper.LoadExamPaperDetailByUserId(ctx, tx, epid, psid, eid, true, true, true)
	if err != nil {
		err = fmt.Errorf("调用LoadExamPaperDetailByUserId失败:%w", err)
		z.Error(err.Error())
		return result, err
	}

	result["exam_paper"] = vep
	result["exam_paper_group"] = tepg
	result["exam_question"] = eq

	answerNum, err := getStudentPracticeAnswerByPracticeID(ctx, psid, studentID)
	if err != nil {
		z.Sugar().Errorf("getStudentPracticeAnswerByPracticeID call failed:%v", err)
		return nil, fmt.Errorf("getStudentPracticeAnswerByPracticeID call failed:%v", err)
	}
	// 学生作答题目总数
	practiceInfoMap["AnswerNum"] = len(answerNum)

	// 获取练习本身的试卷信息
	practiceInfo, err := getPracticeRecord(ctx, psid, studentID)
	if err != nil {
		z.Sugar().Errorf("getPracticeRecord call failed:%v", err)
		return nil, fmt.Errorf("getPracticeRecord call failed:%v", err)
	}
	// 练习建议时长
	practiceInfoMap["SuggestTime"] = practiceInfo.Duration.ValueOrZero()

	// 获取学生练习的作答信息
	practiceAnswerInfo, err := getPracticeAnswerInfoByPracticeIDOrderAttempt(ctx, practiceID, studentID)
	if err != nil {
		z.Sugar().Errorf("getPracticeAnswerInfoByPracticeIDOrderAttempt call failed:%v", err)
		return nil, fmt.Errorf("getPracticeAnswerInfoByPracticeIDOrderAttempt call failed:%v", err)
	}
	// 学生本次练习时长
	practiceInfoMap["AnswerTime"] = math.Ceil(practiceAnswerInfo.UsedTime.Float64)

	var totalScore float64
	// 学生本次练习得分
	for _, v := range eq {
		for _, v2 := range v {
			totalScore += v2.StudentScore.Float64
		}

	}
	practiceInfoMap["StudentScore"] = totalScore

	result["practiceInfo"] = practiceInfoMap

	return result, err

}

type PracticeInfo struct {
	ID          null.Int
	ExamPaperID null.Int
	TotalScore  null.Float
	UsedTime    null.Float
}

func getPracticeAnswerInfoByPracticeIDOrderAttempt(ctx context.Context, practiceID, studentID int64) (PracticeInfo, error) {
	var err error

	// 这里搜索视图中的数据 并且需要根据尝试次数进行排序
	selectSql := `WITH filtered_data AS (
    SELECT DISTINCT ps.id,
        ps.practice_id,
        p.name,
        ps.student_id, 
        ps.exam_paper_id,
        CASE
            WHEN bool_or(a.answer_score IS NULL) THEN NULL::double precision
            ELSE sum(a.answer_score)
        END AS total_score,
        p.type,
        ps.attempt,
        ps.status,
        count(
            CASE
                WHEN a.answer_score <> epq.score THEN 1
                ELSE NULL::integer
            END) AS wrong_count,
        ps.end_time - ps.start_time AS used_time  -- 修正这里
    FROM t_practice_submissions ps
    JOIN t_student_answers a ON ps.id = a.practice_submission_id
    JOIN t_exam_paper_question epq ON a.question_id = epq.id
    JOIN t_practice p ON ps.practice_id = p.id
    WHERE ps.practice_id = $1
        AND ps.student_id = $2
    GROUP BY ps.id, p.id, ps.attempt, ps.student_id
)
SELECT 
    id,
    exam_paper_id,
    total_score,
	used_time
FROM (
    SELECT 
        *,
        ROW_NUMBER() OVER (
            ORDER BY 
                CASE 
                    WHEN status = '06' THEN 1  -- 优先选择状态为'06'的记录
                    ELSE 2 
                END,
                attempt DESC  -- 相同状态下按提交次数倒序
        ) AS rn
    FROM filtered_data
) ranked
WHERE rn = 1;`

	conn := cmn.GetPgxConn()
	if conn == nil {
		err = fmt.Errorf("获取数据库连接为空")
		z.Error(err.Error())
		return PracticeInfo{}, err
	}

	var practiceInfo PracticeInfo
	err = conn.QueryRow(ctx, selectSql, practiceID, studentID).Scan(
		&practiceInfo.ID,
		&practiceInfo.ExamPaperID,
		&practiceInfo.TotalScore,
		&practiceInfo.UsedTime,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// 没有找到记录
			z.Error("getPracticeInfoByPracticeIDOrderAttempt invalid data request", zap.Error(err))
			return PracticeInfo{}, nil
		}
		z.Error("getPracticeInfoByPracticeIDOrderAttempt error", zap.Error(err))
		return PracticeInfo{}, err
	}
	return practiceInfo, nil
}

func getStudentPracticeAnswerByPracticeID(ctx context.Context, practiceSubmissionID, studentID int64) ([]cmn.TStudentAnswers, error) {
	if practiceSubmissionID <= 0 {
		z.Warn("invalid input params practiceSubmissionID")
		return nil, errors.New("invalid input params practiceSubmissionID")
	}
	if studentID <= 0 {
		z.Warn("invalid input params studentId")
		return nil, errors.New("invalid input params studentId")
	}

	var err error
	var AllAnswers []cmn.TStudentAnswers

	conn := cmn.GetPgxConn()
	if conn == nil {
		err = fmt.Errorf("获取数据库连接为空")
		z.Error(err.Error())
		return nil, err
	}
	selectSql := `SELECT id, type ,question_id,answer,answer_score,addi,status FROM t_student_answers WHERE practice_submission_id = $1`
	//根据考生ID去查询此时的数据
	rows, err := conn.Query(ctx, selectSql, practiceSubmissionID)
	if err != nil {
		err = fmt.Errorf("获取失败")
		z.Error(err.Error())
		return nil, err
	}
	defer rows.Close()

	// 一个答案算一次,插入到这个结构体数组中
	for rows.Next() {
		var r cmn.TStudentAnswers
		err := rows.Scan(&r.ID, &r.Type, &r.QuestionID, &r.Answer, &r.AnswerScore, &r.Addi, &r.Status)
		if err != nil {
			z.Error("student_exam_answer/service GetStudentPracticeAnswer getAnswerByExamineeID error", zap.Error(err))
			return nil, err
		}
		AllAnswers = append(AllAnswers, r)
	}
	return AllAnswers, nil
}

type PracticeRecord struct {
	ID                null.Int    `json:"id,omitempty"`                   // 练习 ID
	Type              null.String `json:"type,omitempty"`                 // 练习类型
	Name              null.String `json:"name,omitempty"`                 // 练习名称
	AttemptCount      null.Int    `json:"attempt_count,omitempty"`        // 尝试次数
	Difficulty        null.String `json:"difficulty,omitempty"`           // 难度
	QuestionCount     null.Int    `json:"question_count,omitempty"`       // 题目数量
	WrongCount        null.Int    `json:"wrong_count,omitempty"`          // 错题数量
	TotalScore        null.Float  `json:"total_score,omitempty"`          // 学生得分
	HighestScore      null.Float  `json:"highest_score,omitempty"`        // 最高分
	Creator           null.Int    `json:"creator,omitempty"`              // 创建者ID
	CreateTime        null.Int    `json:"create_time,omitempty"`          // 创建时间
	UpdatedBy         null.Int    `json:"updated_by,omitempty"`           // 更新者ID
	UpdateTime        null.Int    `json:"update_time,omitempty"`          // 更新时间
	Status            null.String `json:"status,omitempty"`               // 状态
	AllowedAttempts   null.Int    `json:"allowed_attempts,omitempty"`     //可作答次数，如果为0，则说明是无限次数
	Duration          null.Int    `json:"duration,omitempty"`             //建议时长
	LastUnSubmittedId null.Int    `json:"last_un_submitted_id,omitempty"` //最近一次的未提交的练习id
	StartTime         null.Int    `json:"start_time,omitempty"`
	EndTime           null.Int    `json:"end_time,omitempty"`
}

func getPracticeRecord(ctx context.Context, practiceSubmissionID, studentId int64) (PracticeRecord, error) {
	var err error

	selectSql := `SELECT 
			p.type,
			p.name,
			COALESCE(tp.level, '00') AS difficulty,
			ps.creator,
			ps.create_time,
			ps.updated_by,
			ps.update_time,
			ps.status,
			p.allowed_attempts,
			tp.suggested_duration,
			ps.start_time,
			ps.end_time
		FROM assessuser.t_practice p
		JOIN assessuser.t_practice_submissions ps ON p.id = ps.practice_id
		LEFT JOIN assessuser.t_paper tp ON tp.id = p.paper_id
		-- WHERE ps.id=$1 AND p.status = $2 AND ps.status = $3 AND ps.student_id = $4 
		WHERE ps.id=$1 AND p.status = $2 AND ps.student_id = $3`

	conn := cmn.GetPgxConn()
	if conn == nil {
		err = fmt.Errorf("获取数据库连接为空")
		z.Error(err.Error())
		return PracticeRecord{}, err
	}

	var practice PracticeRecord
	var suggestDuration null.Int
	err = conn.QueryRow(ctx, selectSql, practiceSubmissionID, "02", studentId).Scan(
		&practice.Type,
		&practice.Name,
		&practice.Difficulty,
		&practice.Creator,
		&practice.CreateTime,
		&practice.UpdatedBy,
		&practice.UpdateTime,
		&practice.Status,
		&practice.AllowedAttempts,
		&suggestDuration,
		&practice.StartTime,
		&practice.EndTime)
	if err != nil {
		z.Error("query row err", zap.Error(err))
		return PracticeRecord{}, err
	}

	practice.Duration = suggestDuration

	return practice, nil
}
