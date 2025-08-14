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
	// 记录函数调用信息，便于日志追踪
	z.Info("---->" + cmn.FncName())

	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}

	var (
		page     int
		pageSize int
		examID   int64

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
	examID = req.ExamID

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
	if examID > 0 {
		whereClause += fmt.Sprintf(" AND ei.id=$%d ", len(params)+1)
		params = append(params, examID)
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
	// 拼接完整SQL语句
	sql := fmt.Sprintf(`
	SELECT ei.id AS id, ei.name AS exam_name, ei.type AS exam_type, jsonb_agg(esi) AS exam_session_info, ei.submitted AS submitted
	FROM t_exam_info ei     
		LEFT JOIN v_z_grade_exam_session_info esi ON esi.exam_id = ei.id
	%s
	GROUP BY ei.id
	`, whereClause)

	// 获取数据库连接
	// 获取数据库连接
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

	// 分页查询SQL
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

	var (
		page       int
		pageSize   int
		practiceID int64

		result   []GradePractice
		rowCount int64
		err      error
	)

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
	practiceID = req.PracticeID

	whereClause := " WHERE 1=1 "

	params := []any{}

	if userID > 0 {
		whereClause += fmt.Sprintf(" AND p.creator=$%d ", len(params)+1)
		params = append(params, userID)
	}

	if practiceID > 0 {
		whereClause += fmt.Sprintf(" AND p.practice_id=$%d ", len(params)+1)
		params = append(params, practiceID)
	}

	filter := req.Filter

	if filter.Name != "" {
		whereClause += fmt.Sprintf(" AND p.practice_name::text ILIKE $%d ", len(params)+1)
		params = append(params, fmt.Sprintf("%%%s%%", filter.Name))
	}

	// 视图查询SQL
	sql := fmt.Sprintf(`
	SELECT practice_id, practice_name, total_score, averge_score, actual_completer, pass_student
	FROM v_z_grade_practice_statistics p 
	%s
	-- GROUP BY p.practice_id, p.practice_name, p.total_score, p.averge_score, p.actual_completer, p.pass_student
	GROUP BY practice_id
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

	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}

	var (
		err          error
		rowsAffected int64
	)

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
// 获取指定考试的成绩分布信息
func gradeDistributionExam(ctx context.Context, examID int, columnNum int) (ExamGradeDistribution, error) {
	z.Info("---->" + cmn.FncName())

	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}

	var (
		result ExamGradeDistribution
		err    error
	)

	// 检查考试ID是否合法
	if examID < 0 {
		err = fmt.Errorf("examID无效(examID=%v)", examID)
		z.Error(err.Error())
		return result, err
	}

	// 检查分布列数是否合法
	if columnNum <= 0 {
		err = fmt.Errorf("列数无效(columnNum=%v)", columnNum)
		z.Error(err.Error())
		return result, err
	}

	conn := cmn.GetPgxConn()
	if conn == nil || forceErr == "conn nil" {
		err = fmt.Errorf("获取数据库连接失败")
		z.Error(err.Error())
		return result, err
	}

	// 筛选考试ID
	whereClause := " WHERE ei.id = $1 "

	// 用于存放各分数段统计SQL
	distributionsSql := []string{}
	// 计算分数区间比例：多少个区间
	scoreInterval := 1 / float64(columnNum)

	// 初始条件
	var initSql string
	// 如果只有一列，允许学生总分为空计入统计，视作成绩为0
	if columnNum == 1 {
		initSql = " OR ets.total_score IS NULL "
	}

	// 构造最后一列的分布统计SQL
	initDistribution := fmt.Sprintf(
		`SUM ( CASE
					WHEN (ets.total_score >= %f * ep.total_score
						AND ets.total_score <= ep.total_score)
						%s
						THEN 1
					ELSE 0
				END )`, scoreInterval*float64(columnNum-1), initSql)
	distributionsSql = append(distributionsSql, initDistribution) // 添加到分布统计数组

	// 构造其他列的分布统计SQL
	for i := columnNum - 1; i > 0; i-- {
		addi := ""
		// 当最后一列将成绩为空列入统计
		if i == 1 {
			addi = " OR ets.total_score IS NULL "
		}

		distribution := fmt.Sprintf(
			`SUM ( CASE
					WHEN (ets.total_score >= %f * ep.total_score
						AND ets.total_score < %f * ep.total_score)
						%s
						THEN 1
					ELSE 0
				END )`, scoreInterval*float64(i-1), scoreInterval*float64(i), addi)

		// 添加到分布统计数组
		distributionsSql = append(distributionsSql, distribution)

	}

	sql := fmt.Sprintf(`
	WITH session_grades AS ( SELECT
		ets.exam_id AS exam_id, ets.exam_session_id, ep.id AS exam_paper_id, ep.name AS exam_paper_name, ep.total_score AS total_score, COUNT(DISTINCT ets.student_id) AS total, ARRAY[ %s ] AS score_distribution
	FROM v_student_exam_total_score ets
		JOIN v_exam_paper ep ON ep.exam_session_id =  ets.exam_session_id
	GROUP BY ets.exam_id, ets.exam_session_id, ep.id, ep.name, ep.total_score )
	SELECT ei.id, ei.name, jsonb_agg(session_grades) AS grade_distribution
	FROM t_exam_info ei
		JOIN session_grades ON session_grades.exam_id = ei.id
	%s
	GROUP BY ei.id
	`, strings.Join(distributionsSql, ", "), whereClause)

	// 执行SQL查询并扫描结果
	err = conn.QueryRow(ctx, sql, examID).Scan(
		&result.ExamID,
		&result.ExamName,
		&result.GradeDistribution,
	)
	if err != nil {
		// 未查到考试ID
		if errors.Is(err, pgx.ErrNoRows) {
			err = fmt.Errorf("该考试还不能统计分布(examID=%v):err:%w", examID, err)
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

	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}

	var (
		err    error
		result PracticeGradeDistribution
	)

	// 检查练习ID是否合法
	if practiceID <= 0 {
		err = fmt.Errorf("练习ID无效(practiceID=%v)", practiceID)
		z.Error(err.Error())
		return result, err
	}

	// 检查分布列数是否合法
	if columnNum < 1 {
		err = fmt.Errorf("列数无效(columnNum=%v)", columnNum)
		z.Error(err.Error())
		return result, err
	}

	conn := cmn.GetPgxConn()
	if conn == nil || forceErr == "conn nil" {
		err = fmt.Errorf("获取数据库连接失败")
		z.Error(err.Error())
		return result, err
	}

	// 筛选练习ID
	whereClause := " WHERE p.id = $1 "

	// 用于存放各分数段统计SQL
	distributionsSql := []string{}
	// 计算分数区间比例：多少个区间
	scoreInterval := 1 / float64(columnNum)

	// 初始条件
	var initSql string
	// 如果只有一列，允许学生总分为空计入统计，视作成绩为0
	if columnNum == 1 {
		initSql = " OR sp.avg_score IS NULL "
	}

	// 构造最后一列的分布统计SQL
	initDistribution := fmt.Sprintf(
		`SUM ( CASE
					WHEN (sp.avg_score >= %f * ep.total_score
						AND sp.avg_score <= %d * ep.total_score)
						%s
						THEN 1
					ELSE 0
				END )`, scoreInterval*float64(columnNum-1), 1, initSql)

	distributionsSql = append(distributionsSql, initDistribution) // 添加到分布统计数组

	// 构造其他列的分布统计SQL
	for i := columnNum - 1; i > 0; i-- {
		addi := ""
		// 当最后一列将成绩为空列入统计
		if i == 1 {
			addi = " OR sp.avg_score IS NULL "
		}

		distribution := fmt.Sprintf(
			`SUM ( CASE
					WHEN (sp.avg_score >= %f * ep.total_score
						AND sp.avg_score < %f * ep.total_score)
						%s
						THEN 1
					ELSE 0
				END )`, scoreInterval*float64(i-1), scoreInterval*float64(i), addi)

		distributionsSql = append(distributionsSql, distribution) // 添加到分布统计数组

	}

	// 拼接完整SQL语句
	sql := fmt.Sprintf(`WITH stu_practice AS (
		SELECT student_id, practice_id, name, exam_paper_id, COUNT(1) AS attempt, AVG(total_score) AS avg_score, AVG(wrong_count)::integer AS avg_wrong_count, AVG(used_time) AS avg_used_time
		FROM v_student_practice_total_score
		GROUP BY student_id, practice_id, name, exam_paper_id )
	SELECT p.id AS practice_id, p.name AS practice_name, COUNT(sp.student_id) AS total_stu, ep.total_score AS total_score, ARRAY[ %s ] AS score_distribution
	FROM t_practice p
		JOIN stu_practice sp ON sp.practice_id = p.id
		JOIN v_exam_paper ep ON ep.id = sp.exam_paper_id
	%s
	GROUP BY p.id, ep.total_score
	`, strings.Join(distributionsSql, ", "), whereClause)

	// 执行SQL查询并扫描结果
	err = conn.QueryRow(ctx, sql, practiceID).Scan(
		&result.PracticeID,
		&result.PracticeName,
		&result.TotalStudents,
		&result.TotalScore,
		&result.GradeDistribution,
	)
	if err != nil {
		// 未查到练习ID
		if errors.Is(err, pgx.ErrNoRows) {
			err = fmt.Errorf("该练习还不能统计分布(practiceID=%d):err:%w", practiceID, err)
			z.Error(err.Error())
			return result, err
		}

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
func gradeExamineeListExam(ctx context.Context, req GradeExamineeListReq) ([]ExamineeScoreList, int64, error) {
	z.Info("---->" + cmn.FncName())

	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}

	var (
		examID   []int64
		page     int
		pageSize int

		err        error
		totalCount int64
		result     []ExamineeScoreList
	)

	if len(req.ExamID) <= 0 {
		err = fmt.Errorf("考试ID长度小于等于0")
		z.Error(err.Error())
		return result, totalCount, err
	}
	examID = req.ExamID

	if req.Page <= 0 {
		err = fmt.Errorf("页码小于等于0(page=%v)", req.Page)
		z.Error(err.Error())
		return result, totalCount, err
	}
	page = req.Page

	if req.PageSize <= 0 {
		err = fmt.Errorf("每页数量小于等于0(pageSize=%v)", req.PageSize)
		z.Error(err.Error())
		return result, totalCount, err
	}
	pageSize = req.PageSize

	var sql string
	var params []any

	// 多个考试ID
	placeholders := make([]string, len(examID))
	for i, id := range examID {
		placeholders[i] = fmt.Sprintf("$%d", len(params)+1)
		params = append(params, id)
	}
	placeholderStr := strings.Join(placeholders, ", ")
	whereClause := fmt.Sprintf("WHERE ets.exam_id IN (%s) ", placeholderStr)

	filter := req.Filter
	// 姓名,昵称,电话模糊搜索
	if filter.Keyword != "" {
		whereClause += fmt.Sprintf(" AND (e.official_name::text ILIKE $%d ", len(params)+1)
		params = append(params, fmt.Sprintf("%%%s%%", filter.Keyword))

		whereClause += fmt.Sprintf(" OR e.mobile_phone::text ILIKE $%d ", len(params)+1)
		params = append(params, fmt.Sprintf("%%%s%%", filter.Keyword))

		whereClause += fmt.Sprintf(" OR e.account::text ILIKE $%d)", len(params)+1)
		params = append(params, fmt.Sprintf("%%%s%%", filter.Keyword))
	}

	sql = fmt.Sprintf(`
	SELECT
		ets.exam_id, ei.name AS exam_name, e.student_id, e.mobile_phone AS phone, e.official_name AS name, e.account AS nickname, STRING_AGG(COALESCE(e.remark, ''), '') AS remark, 
		jsonb_agg(jsonb_build_object('exam_id', ets.exam_id, 'exam_session_id', ets.exam_session_id, 'score', ets.total_score) ORDER BY ets.exam_session_id) AS exam_sessions
	FROM v_student_exam_total_score ets
		JOIN t_exam_info ei ON ei.id = ets.exam_id
		JOIN v_examinee_info e ON e.student_id = ets.student_id
	%s
	GROUP BY ets.exam_id, ei.name, e.student_id, e.mobile_phone, e.official_name, e.account
	ORDER BY ets.exam_id, e.official_name
	`, whereClause)

	conn := cmn.GetPgxConn()
	if conn == nil || forceErr == "conn nil" {
		err = fmt.Errorf("获取数据库连接为空")
		z.Error(err.Error())
		return result, totalCount, err
	}

	// 查询列表
	var listSQL string
	var listParams []any

	// 不进行筛选，用于导出成绩
	if page == -1 && pageSize == -1 {
		listSQL = sql
		listParams = params
	} else {
		// 分页
		listParams = params
		listSQL = fmt.Sprintf(`%s
		LIMIT $%d OFFSET $%d`,
			sql, len(listParams)+1, len(listParams)+2)
		listParams = append(listParams, int32(pageSize), int32((page-1)*pageSize))
	}

	rows, err := conn.Query(ctx, listSQL, listParams...)
	if err != nil {
		err = fmt.Errorf("查询考试考生成绩列表失败 错误: %s", err.Error())
		z.Error(err.Error())
		return result, totalCount, err
	}
	defer rows.Close()

	// 按examID分组
	examMap := make(map[int64]*ExamineeScoreList)

	for rows.Next() {
		var examID int64
		var examName null.String
		var student StudentExamScore

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
			return result, totalCount, err
		}

		// 按examID分组，将学生成绩归类到对应考试
		if examData, exists := examMap[examID]; exists {
			// 如果该examID已存在，则将当前学生成绩添加到该考试下
			examData.StudentScores = append(examData.StudentScores, student)
		} else {
			// 如果该examID不存在，则新建一个考试分组，并添加当前学生成绩
			examMap[examID] = &ExamineeScoreList{
				ExamID:        examID,                      // 考试ID
				ExamName:      examName,                    // 考试名称
				StudentScores: []StudentExamScore{student}, // 当前学生成绩
			}
		}
		totalCount++ // 累加总人数计数
	}

	for _, examData := range examMap {
		result = append(result, *examData)
	}

	return result, totalCount, nil
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
func gradeExamineeListPractice(ctx context.Context, req GradeExamineeListReq) ([]PracticeScoreList, int64, error) {
	z.Info("---->" + cmn.FncName())

	forceErr := ""
	if val := ctx.Value("force-error"); val != nil {
		forceErr = val.(string)
	}

	var (
		page       int
		pageSize   int
		practiceID []int64

		err        error
		result     []PracticeScoreList
		totalCount int64
	)

	if len(req.PracticeID) <= 0 {
		err = fmt.Errorf("练习ID列表为空")
		z.Error(err.Error())
		return result, totalCount, err
	}
	practiceID = req.PracticeID
	if req.Page <= 0 {
		err = fmt.Errorf("页码小于等于0(page=%v)", req.Page)
		z.Error(err.Error())
		return result, totalCount, err
	}
	page = req.Page

	if req.PageSize <= 0 {
		err = fmt.Errorf("每页数量小于等于0(pageSize=%v)", req.PageSize)
		z.Error(err.Error())
		return result, totalCount, err
	}
	pageSize = req.PageSize

	var sql string
	var params []any

	// 多个练习ID
	placeholders := make([]string, len(practiceID))
	for i, id := range practiceID {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		params = append(params, id)
	}
	placeholderStr := strings.Join(placeholders, ", ")
	whereClause := fmt.Sprintf("WHERE pts.practice_id IN (%s) ", placeholderStr)

	filter := req.Filter
	// 关键词过滤
	if filter.Keyword != "" {
		whereClause += fmt.Sprintf(" AND (ei.official_name::text ILIKE $%d ", len(params)+1)
		params = append(params, fmt.Sprintf("%%%s%%", filter.Keyword))

		whereClause += fmt.Sprintf(" OR ei.mobile_phone::text ILIKE $%d ", len(params)+1)
		params = append(params, fmt.Sprintf("%%%s%%", filter.Keyword))

		whereClause += fmt.Sprintf(" OR ei.account::text ILIKE $%d)", len(params)+1)
		params = append(params, fmt.Sprintf("%%%s%%", filter.Keyword))
	}

	sql = fmt.Sprintf(`
	SELECT
		pts.practice_id, p.name AS practice_name, pts.student_id, ei.mobile_phone AS phone, ei.official_name AS name, ei.account AS nickname, MAX(pts.total_score) AS highest_score, COUNT(DISTINCT pts.id) AS submitted_cnt
	FROM v_student_practice_total_score pts
		LEFT JOIN v_examinee_info ei ON ei.student_id = pts.student_id
		LEFT JOIN t_practice p ON p.id = pts.practice_id
	%s
	GROUP BY pts.practice_id, p.name, pts.student_id, ei.mobile_phone, ei.official_name, ei.account
	ORDER BY pts.practice_id, pts.student_id
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
		listParams = append(listParams, pageSize, int32((page-1)*pageSize))
	}

	conn := cmn.GetPgxConn()
	if conn == nil || forceErr == "conn nil" {
		err = fmt.Errorf("获取数据库连接为空")
		z.Error(err.Error())
		return result, totalCount, err
	}

	rows, err := conn.Query(ctx, listSQL, listParams...)
	if err != nil {
		err = fmt.Errorf("查询练习考生成绩列表失败 错误: %w", err)
		z.Error(err.Error())
		return result, totalCount, err
	}

	defer rows.Close()
	practiceMap := make(map[int64]*PracticeScoreList)

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
			return result, totalCount, err
		}

		if practiceData, exists := practiceMap[practiceID]; exists {
			practiceData.StudentScores = append(practiceData.StudentScores, scoreInfo)
		} else {
			practiceMap[practiceID] = &PracticeScoreList{
				PracticeID:    practiceID,
				PracticeName:  practiceName,
				StudentScores: []PracticeExamineeScoreInfo{scoreInfo},
			}
		}
		totalCount++
	}

	for _, practiceData := range practiceMap {
		result = append(result, *practiceData)
	}

	return result, totalCount, err
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

	var (
		analysis Analysis
		err      error
	)

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

	// 第一步：从考试场次或练习ID获取要分析的考卷ID，exam_paper_id
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

// getScoreExam
/*
	关键参数说明：
		ctx 上下文
		tx 数据库事务
		studentID 学生ID
		examSessionID 考试场次ID
		practiceID 练习ID
	return 成绩响应 错误信息

*/
func getScoreExam(ctx context.Context, tx pgx.Tx, studentID, examSessionID int64) (Map, error) {
	z.Info("---->" + cmn.FncName())

	var (
		epid int64 // 考卷Id
		psid int64 // 练习生提交Id 大于0则查询练习学生试卷
		eid  int64 // 考生Id 大于0则查询考试学生试卷

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
	if examSessionID <= 0 {
		err = fmt.Errorf("考试场次ID不能为空(examSessionID=%v)", examSessionID)
		z.Error(err.Error())
		return result, err
	}
	if studentID <= 0 {
		err = fmt.Errorf("学生ID不能为空(studentID=%v)", studentID)
		z.Error(err.Error())
		return result, err
	}

	// 第一步：获取考生ID和考卷ID
	epSql := `
	SELECT id, exam_paper_id
	FROM t_examinee
    WHERE student_id = $1 AND exam_session_id = $2
	`
	err = conn.QueryRow(ctx, epSql, studentID, examSessionID).Scan(&eid, &epid)
	if err != nil {
		err = fmt.Errorf("查询考试学生试卷ID失败: (examSessionID=%v, studentID=%v) %w", examSessionID, studentID, err)
		z.Error(err.Error())
		return result, err
	}

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
	rank := make([]ExamSessionScoreRank, 0)

	rankSql := `SELECT
	ets.student_id, u.official_name, COALESCE(ets.total_score, 0) AS total_score,  RANK() OVER (ORDER BY COALESCE(ets.total_score, 0) DESC) AS rank  
	FROM v_student_exam_total_score ets
	JOIN t_user u ON ets.student_id = u.id
	WHERE ets.exam_session_id = $1  
	ORDER BY COALESCE(ets.total_score, 0) DESC; `

	rows, err := conn.Query(ctx, rankSql, examSessionID)
	if err != nil {
		err = fmt.Errorf("查询考试场次成绩失败: %w", err)
		z.Error(err.Error())
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var r ExamSessionScoreRank
		err = rows.Scan(&r.StudentID, &r.OfficialName, &r.TotalScore, &r.Rank)
		if err != nil {
			err = fmt.Errorf("扫描考试场次成绩失败: %w", err)
			z.Error(err.Error())
			return result, err
		}
		rank = append(rank, r)
	}
	if err = rows.Err(); err != nil {
		err = fmt.Errorf("遍历考试场次成绩失败: %w", err)
		z.Error(err.Error())
		return result, err
	}
	result["rank"] = rank

	// 第四步：获取学生ID，学生考试总分
	for _, v := range rank {
		if v.StudentID == studentID {
			//  存储考试基本信息：学生总分
			examInfoMap["StudentScore"] = v.TotalScore
			result["student_id"] = v.StudentID
			break
		}
	}

	// 第五步：获取考试场次信息，根据考试ID与学生ID，获取本次考试的信息；包括试卷id，考生id，场次顺序；筛选已批改的状态
	// 取出examID
	eiSql := `
	SELECT es.exam_id
	FROM t_exam_session es
	WHERE es.id = $1
	`

	var examID int64
	err = conn.QueryRow(ctx, eiSql, examSessionID).Scan(&examID)
	if err != nil {
		err = fmt.Errorf("查询考试场次ID失败: %w", err)
		z.Error(err.Error())
		return nil, err
	}

	// TODO: 筛选状态调整
	esiSql := `
		SELECT 
			es.id, e.exam_paper_id, es.duration, e.id as examinee_id, es.session_num
		FROM t_exam_session es
		LEFT JOIN t_examinee e ON es.id = e.exam_session_id AND e.student_id = $1 AND e.status != $2
		WHERE es.exam_id = $3 AND es.status != $4
		ORDER BY es.session_num
	`
	rows, err = conn.Query(ctx, esiSql, studentID, "08", examID, "06")
	if err != nil {
		err = fmt.Errorf("查询考试场次失败: %w", err)
		z.Error(err.Error())
		return nil, err
	}
	defer rows.Close()

	var sessionInfo []SessionInfo
	for rows.Next() {
		var session SessionInfo
		err = rows.Scan(
			&session.ID,
			&session.PaperID,
			&session.Duration,
			&session.ExamineeID,
			&session.SessionNum,
		)
		if err != nil {
			err = fmt.Errorf("扫描考试场次失败: %w", err)
			z.Error(err.Error())
			return nil, err
		}
		sessionInfo = append(sessionInfo, session)
	}
	if rows.Err() != nil {
		err = fmt.Errorf("遍历考试场次失败: %w", err)
		z.Error(err.Error())
		return nil, err
	}
	for _, v := range sessionInfo {
		examInfo := Map{
			"ID":         v.ID.ValueOrZero(),
			"PaperID":    v.PaperID.ValueOrZero(),
			"ExamTime":   v.Duration.ValueOrZero(),
			"ExamineeID": v.ExamineeID.ValueOrZero(),
			"SessionNum": v.SessionNum.ValueOrZero(),
		}
		examSessionInfoMap = append(examSessionInfoMap, examInfo)
	}
	result["examSessionInfo"] = examSessionInfoMap

	// 第六步：获取考试答题信息
	ansNumSql := `SELECT COUNT(*) FROM t_student_answers WHERE examinee_id = $1`
	var answerNum int
	err = conn.QueryRow(ctx, ansNumSql, eid).Scan(&answerNum)
	if err != nil {
		err = fmt.Errorf("学生ID不能为空(studentID=%v)", studentID)
		z.Error(err.Error())
		return result, err
	}
	examInfoMap["AnswerNum"] = answerNum

	ansTimeSql := `SELECT
	start_time,end_time
	FROM t_examinee
	WHERE id = $1
	`

	var start, end null.Int
	err = conn.QueryRow(ctx, ansTimeSql, eid).Scan(&start, &end)
	if err != nil {
		err = fmt.Errorf("查询考试场次时间失败: %w", err)
		z.Error(err.Error())
		return result, err
	}
	if !start.Valid || !end.Valid {
		err = fmt.Errorf("考试场次时间为空")
		z.Warn(err.Error())
		return result, err
	}
	answerTime := end.Int64 - start.Int64
	examInfoMap["AnswerTime"] = answerTime

	result["examInfo"] = examInfoMap
	result["examSessionInfo"] = examSessionInfoMap

	return result, err
}

func getScorePractice(ctx context.Context, tx pgx.Tx, studentID int64, practiceID int64) (Map, error) {
	var (
		epid int64 // 考卷Id
		psid int64 // 练习生提交Id 大于0则查询练习学生试卷
		eid  int64 // 考生Id 大于0则查询考试学生试卷

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
	if practiceID <= 0 {
		err = fmt.Errorf("练习ID不能为空(practiceID=%v)", practiceID)
		z.Error(err.Error())
		return result, err
	}
	if studentID <= 0 {
		err = fmt.Errorf("学生ID不能为空(studentID=%v)", studentID)
		z.Error(err.Error())
		return result, err
	}

	// 第一步：获取考卷ID和练习ID
	epSql := `
	SELECT id,exam_paper_id
	FROM t_practice_submissions
	WHERE student_id = $1 AND practice_id = $2
	`
	err = conn.QueryRow(ctx, epSql, studentID, practiceID).Scan(&psid, &epid)
	if err != nil {
		err = fmt.Errorf("查询练习学生试卷ID失败: %w", err)
		z.Error(err.Error())
		return result, err
	}

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

	// 第三步：获取学生总分
	var totalScore float64
	for _, q := range eq {
		for _, q2 := range q {
			totalScore += q2.StudentScore.Float64
		}
	}
	practiceInfoMap["StudentScore"] = totalScore

	selectSql := `SELECT COUNT(*) FROM t_student_answers WHERE practice_submission_id = $1`
	var answerNum int
	err = conn.QueryRow(ctx, selectSql, psid).Scan(&answerNum)
	if err != nil {
		err = fmt.Errorf("获取失败")
		z.Error(err.Error())
		return nil, err
	}
	practiceInfoMap["AnswerNum"] = answerNum

	// 练习建议时长
	durationSql := `SELECT 
			tp.suggested_duration
		FROM t_practice p
		JOIN t_practice_submissions ps ON p.id = ps.practice_id
		LEFT JOIN t_paper tp ON tp.id = p.paper_id        
		    -- WHERE ps.id=$1 AND p.status = $2 AND ps.status = $3 AND ps.student_id = $4 
		WHERE ps.id=$1 AND p.status = $2 AND ps.student_id = $3`
	var suggestDuration null.Int
	err = conn.QueryRow(ctx, durationSql, psid, "02", studentID).Scan(&suggestDuration)
	if err != nil {
		z.Error("query row err", zap.Error(err))
		return result, err
	}
	practiceInfoMap["SuggestTime"] = suggestDuration

	// 获取学生练习的作答信息
	practiceAnswerInfo, err := getPracticeAnswerInfoByPracticeIDOrderAttempt(ctx, practiceID, studentID)
	if err != nil {
		z.Sugar().Errorf("getPracticeAnswerInfoByPracticeIDOrderAttempt call failed:%v", err)
		return nil, fmt.Errorf("getPracticeAnswerInfoByPracticeIDOrderAttempt call failed:%v", err)
	}
	// 学生本次练习时长
	practiceInfoMap["AnswerTime"] = math.Ceil(practiceAnswerInfo.UsedTime.Float64)

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
