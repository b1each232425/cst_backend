package grade

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"w2w.io/cmn"
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
	z.Debug(forceErr)

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

	for _, examID := range examIDs {
		examSessions := []cmn.TExamSession{}

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
			err := es_rows.Scan(
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
