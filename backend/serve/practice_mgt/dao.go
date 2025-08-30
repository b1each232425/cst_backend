package practice_mgt

//annotation:practice_mgt-service
//author:{"name":"ZouDeLun","tel":"15920422045", "email":"1311866870@qq.com"}
import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"
	"strings"
	"time"
	"w2w.io/cmn"
	"w2w.io/null"
	"w2w.io/serve/examPaper"
	"w2w.io/serve/mark"
)

// UpsertPractice 新增/修改练习信息 根据用户传输的信息动态构建SQL语句
/*
关键参数说明：
	p 要插入/更新的练习信息
	ps 参与练习的学生ID数组
	uid 操作人
*/
func UpsertPractice(ctx context.Context, p *cmn.TPractice, ps []int64, uid int64, isClearStudent bool) error {
	if uid <= 0 {
		err := fmt.Errorf("invalid updator ID param")
		z.Error(err.Error())
		return err
	}
	if !p.ID.Valid {
		return AddPractice(ctx, p, ps, uid)
	}
	p2, _, _, err := LoadPracticeById(ctx, p.ID.Int64)
	if err != nil {
		return err
	}
	if p2.Status.String == PracticeStatus.Released {
		return fmt.Errorf("练习已经发布，不可修改练习信息")
	}
	return UpdatePractice(ctx, p, ps, uid, false, isClearStudent)

}

// UpdatePractice 更新练习本身信息
/*
关键参数说明：
	p 要插入/更新的练习信息
	ps 参与练习的学生ID数组 为空或者长度为0则不更新学生名单 否则就更新
	uid 操作人
	isOperate 是否通过operate操作函数调用的：是则允许更新status字段，否则不允许更新status字段
	isClear 是否清除学生名单
*/
func UpdatePractice(ctx context.Context, p *cmn.TPractice, ps []int64, uid int64, isOperate, isClear bool) error {
	if uid <= 0 {
		err := fmt.Errorf("invalid updator ID param")
		z.Error(err.Error())
		return err
	}
	// 用于测试，强制执行某些错误分支
	forceErr, _ := ctx.Value("force-error").(string)
	now := time.Now().UnixMilli()
	p.UpdatedBy = null.IntFrom(uid)
	p.UpdateTime = null.IntFrom(now)
	update := S2Map(p)
	notUpdate := []string{
		"id",
		"creator",
		"create_time",
	}
	// 不允许随意的更改练习的发布状态
	if !isOperate {
		notUpdate = append(notUpdate, "status")
	}

	RemoveFields(update, notUpdate...)
	z.Sugar().Infof("update:%v", Json(update))
	tableName := p.GetTableName()
	var clauses []string
	var args []interface{}
	idx := 1
	for field, value := range update {
		clauses = append(clauses, fmt.Sprintf("%s = $%d", field, idx))
		args = append(args, value)
		idx++
	}
	args = append(args, p.ID)
	// 更新练习本身与更新练习学生是两个事情，可以不同步进行
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = $%d", tableName, strings.Join(clauses, ", "), idx)
	z.Sugar().Debugf("update sql:%v", query)
	z.Sugar().Debugf("update args:%v", args)
	sqlxDB := cmn.GetDbConn()
	_, err := sqlxDB.ExecContext(ctx, query, args...)
	if err != nil || forceErr == "query" {
		err = fmt.Errorf("updatePractice call failed:%v", err)
		z.Error(err.Error())
		return err
	}
	if !isClear && len(ps) == 0 {
		return nil
	}
	err = UpsertPracticeStudentV2(ctx, p.ID.Int64, uid, ps)
	if err != nil {
		return err
	}
	return nil

}

// AddPractice 添加一场练习 包括插入成功导入的学生
/*
关键参数说明：
	p 要插入/更新的练习信息
	ps 参与练习的学生ID数组
	uid 操作人
*/
func AddPractice(ctx context.Context, p *cmn.TPractice, ps []int64, uid int64) error {
	var id int64
	now := time.Now().UnixMilli()
	sqlxDB := cmn.GetDbConn()
	// 用于测试，强制执行某些错误分支
	forceErr, _ := ctx.Value("force-error").(string)
	s := `
	INSERT INTO assessuser.t_practice (name,correct_mode,creator,create_time, update_time, addi,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`
	err := sqlxDB.QueryRowxContext(ctx, s, p.Name, p.CorrectMode, uid, now, now, p.Addi, p.AllowedAttempts, p.Type, p.PaperID).Scan(&id)
	if err != nil || forceErr == "query" {
		err = fmt.Errorf("addPractice call failed:%v", err)
		z.Error(err.Error())
		return err
	}
	p.ID = null.IntFrom(id)
	if ps == nil || len(ps) == 0 {
		return nil
	}
	err = UpsertPracticeStudentV2(ctx, id, uid, ps)
	if err != nil {
		return err
	}
	return nil
}

// UpsertPracticeStudent 更新一次练习参与的学生名单
/*
关键参数说明：
	pid 要绑定关联的练习ID
	uid 操作人
	ps 参与练习的学生ID数组
*/
func UpsertPracticeStudent(ctx context.Context, pid, uid int64, ps []int64) error {
	if ps == nil || len(ps) == 0 {
		return nil
	}
	if pid <= 0 {
		err := fmt.Errorf("invalid practiceId  param")
		z.Error(err.Error())
		return err
	}
	if uid <= 0 {
		err := fmt.Errorf("invalid uid param")
		z.Error(err.Error())
		return err
	}
	// 用于测试，强制执行某些错误分支
	forceErr, _ := ctx.Value("force-error").(string)
	//添加学生
	addPStr := strings.Repeat("(?,?,?,?,?,?,?),", len(ps)-1) + "(?,?,?,?,?,?,?)"
	addPArgs := make([]interface{}, 0, len(ps)*7+1)

	// 软删除学生
	var valueExpr []string
	var delPArgs []interface{}
	now := time.Now().UnixMilli()
	sqlxDB := cmn.GetDbConn()
	tx, err := sqlxDB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil || forceErr == "beginTx" {
		err = fmt.Errorf("beginTx called failed:%v", err)
		z.Error(err.Error())
		return err
	}
	defer func() {
		if forceErr == "rollback" {
			err = fmt.Errorf("触发回滚")
		}
		if err != nil {
			// 操作失败回滚
			err = tx.Rollback()
			if forceErr == "rollback" {
				err = fmt.Errorf("触发回滚")
			}
			if err != nil {
				z.Error(err.Error())
				return
			}
		}
		// 无错误则提交
		err = tx.Commit()
		if forceErr == "commit" {
			err = fmt.Errorf("commit failed")
		}
		if err != nil {
			z.Error(err.Error())
		}
	}()

	t := `
	INSERT INTO assessuser.t_practice_student 
    	(student_id, practice_id, creator, create_time, updated_by, update_time, status) 
	VALUES %s
	ON CONFLICT (student_id, practice_id)
	DO UPDATE SET
    	status = EXCLUDED.status,
    	updated_by = EXCLUDED.updated_by,
    	update_time = EXCLUDED.update_time
	WHERE assessuser.t_practice_student.status IS DISTINCT FROM ? 
	`
	s1 := fmt.Sprintf(t, addPStr)

	for _, sid := range ps {
		addPArgs = append(addPArgs,
			sid, pid, uid, now, uid, now, PracticeStudentStatus.Normal,
		)
	}
	addPArgs = append(addPArgs, PracticeStudentStatus.Normal)

	// 使用sqlx.In 构建批量操作的SQL语句
	addPQuery, args, _ := sqlx.In(s1, addPArgs...)
	addPQuery = sqlx.Rebind(sqlx.DOLLAR, addPQuery)

	z.Sugar().Debugf("打印输出一下增加SQL语句:%v", addPQuery)
	z.Sugar().Debugf("打印输出一下增加SQL参数:%v", args...)
	_, err = tx.ExecContext(ctx, addPQuery, args...)
	if err != nil || forceErr == "query1" {
		err = fmt.Errorf("add PracticeStudent call failed:%v", err)
		z.Error(err.Error())
		return err
	}

	// 然后继续进行删除不在名单上学生的操作
	t2 := `
	UPDATE assessuser.t_practice_student t
	SET status = $1 , update_time = $2 , updated_by = $3
	WHERE t.practice_id = $4
		AND NOT EXISTS (
			SELECT 1 
			FROM (VALUES %s) AS excluded(sid)
			WHERE t.student_id = excluded.sid
		)
	`
	delPArgs = append(delPArgs, PracticeStudentStatus.Deleted, now, uid, pid)
	for _, sid := range ps {
		valueExpr = append(valueExpr, fmt.Sprintf("($%d::bigint)", len(delPArgs)+1))
		delPArgs = append(delPArgs, sid)
	}

	s2 := fmt.Sprintf(t2, strings.Join(valueExpr, ", "))

	z.Sugar().Debugf("打印输出一下删除SQL语句:%v", s2)
	z.Sugar().Debugf("打印输出一下删除SQL参数:%v", delPArgs...)
	_, err = tx.ExecContext(ctx, s2, delPArgs...)
	if err != nil || forceErr == "query2" {
		err = fmt.Errorf("delete PracticeStudent call failed:%v", err)
		z.Error(err.Error())
		return err
	}
	return nil
}

// UpsertPracticeStudentV2 手动批量导入学生 用于教师创建练习时，手动导入的学生
func UpsertPracticeStudentV2(ctx context.Context, pid, uid int64, ps []int64) error {

	if pid <= 0 {
		err := fmt.Errorf("invalid practiceId param")
		z.Error(err.Error())
		return err
	}
	if uid <= 0 {
		err := fmt.Errorf("invalid uid param")
		z.Error(err.Error())
		return err
	}
	forceErr, _ := ctx.Value("force-error").(string)
	now := time.Now().UnixMilli()
	sqlxDB := cmn.GetDbConn()
	tx, err := sqlxDB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil || forceErr == "beginTx" {
		err = fmt.Errorf("beginTx called failed:%v", err)
		z.Error(err.Error())
		return err
	}
	defer func() {
		if forceErr == "rollback" {
			err = fmt.Errorf("触发回滚")
		}
		if err != nil {
			_ = tx.Rollback()
			if forceErr == "rollback" {
				err = fmt.Errorf("触发回滚")
			}
			if err != nil {
				z.Error(err.Error())
				return
			}
		}
		_ = tx.Commit()
		if forceErr == "commit" {
			err = fmt.Errorf("commit failed")
		}
		if err != nil {
			z.Error(err.Error())
		}
	}()

	if ps == nil || len(ps) == 0 {
		// 清空该练习下所有学生
		delSQL := `
            UPDATE assessuser.t_practice_student
            SET status = $1, update_time = $2, updated_by = $3
            WHERE practice_id = $4 AND status != $1
        `
		_, err = tx.ExecContext(ctx, delSQL, PracticeStudentStatus.Deleted, now, uid, pid)
		if err != nil || forceErr == "query1" {
			err = fmt.Errorf("clear PracticeStudent call failed:%v", err)
			z.Error(err.Error())
			return err
		}
		return nil
	}

	// upsert名单
	addPStr := strings.Repeat("(?,?,?,?,?,?,?),", len(ps)-1) + "(?,?,?,?,?,?,?)"
	addPArgs := make([]interface{}, 0, len(ps)*7+1)
	for _, sid := range ps {
		addPArgs = append(addPArgs,
			sid, pid, uid, now, uid, now, PracticeStudentStatus.Normal,
		)
	}
	addPArgs = append(addPArgs, PracticeStudentStatus.Normal)
	t := `
        INSERT INTO assessuser.t_practice_student 
            (student_id, practice_id, creator, create_time, updated_by, update_time, status) 
        VALUES %s
        ON CONFLICT (student_id, practice_id)
        DO UPDATE SET
            status = EXCLUDED.status,
            updated_by = EXCLUDED.updated_by,
            update_time = EXCLUDED.update_time
        WHERE assessuser.t_practice_student.status IS DISTINCT FROM ?
    `
	s1 := fmt.Sprintf(t, addPStr)
	addPQuery, args, _ := sqlx.In(s1, addPArgs...)
	addPQuery = sqlx.Rebind(sqlx.DOLLAR, addPQuery)
	z.Sugar().Debugf("打印输出一下增加SQL语句:%v", addPQuery)
	z.Sugar().Debugf("打印输出一下增加SQL参数:%v", args...)
	_, err = tx.ExecContext(ctx, addPQuery, args...)
	if err != nil || forceErr == "query2" {
		err = fmt.Errorf("add PracticeStudent call failed:%v", err)
		z.Error(err.Error())
		return err
	}

	// 删除不在名单上的学生
	var valueExpr []string
	var delPArgs []interface{}
	delPArgs = append(delPArgs, PracticeStudentStatus.Deleted, now, uid, pid)
	for _, sid := range ps {
		valueExpr = append(valueExpr, fmt.Sprintf("($%d::bigint)", len(delPArgs)+1))
		delPArgs = append(delPArgs, sid)
	}
	t2 := `
        UPDATE assessuser.t_practice_student t
        SET status = $1, update_time = $2, updated_by = $3
        WHERE t.practice_id = $4
            AND NOT EXISTS (
                SELECT 1 
                FROM (VALUES %s) AS excluded(sid)
                WHERE t.student_id = excluded.sid
            )
    `
	s2 := fmt.Sprintf(t2, strings.Join(valueExpr, ", "))
	z.Sugar().Debugf("打印输出一下删除SQL语句:%v", s2)
	z.Sugar().Debugf("打印输出一下删除SQL参数:%v", delPArgs...)
	_, err = tx.ExecContext(ctx, s2, delPArgs...)
	if err != nil || forceErr == "query3" {
		err = fmt.Errorf("delete PracticeStudent call failed:%v", err)
		z.Error(err.Error())
		return err
	}
	return nil
}

// LoadPracticeById 获取单个练习详情 其中不需要查询学生具体信息
/*
关键参数说明：
	pid 要查询的练习ID

返回参数说明：
	1、练习信息
	2、练习绑定的试卷名
	3、练习参与学生数量
	4、可能出现的错误
*/
func LoadPracticeById(ctx context.Context, pid int64) (*cmn.TPractice, string, int, error) {
	if pid <= 0 {
		err := fmt.Errorf("非法practiceID:%v", pid)
		z.Error(err.Error())
		return &cmn.TPractice{}, "", 0, err
	}
	// 用于测试，强制执行某些错误分支
	forceErr, _ := ctx.Value("force-error").(string)
	s := `
	select p.id, p.name, p.correct_mode,p.addi,p.status,p.type,
			COALESCE(tp.name, '') as paper_name,p.allowed_attempts,p.paper_id,p.exam_paper_id,
			COALESCE((SELECT COUNT(*) FROM assessuser.t_practice_student tps WHERE tps.practice_id=p.id AND tps.status=$1),0) as student_cnt
	from assessuser.t_practice p
	left join assessuser.t_paper tp on tp.id = p.paper_id AND tp.status != $2 AND tp.status != $3
	where p.id = $4 AND p.status != $5
	limit 1`
	sqlxDB := cmn.GetDbConn()
	var stmt *sqlx.Stmt
	if forceErr == "prepare" {
		s = `
	select p.id, p.name, p.correct_mode,p.addi,p.status,p.type,
			COALESCE(tp.name, '') as paper_name,p.allowed_attempts,p.paper_id,p.exam_paper_id,
			COALESCE((SELECT COUNT(*) FROM assessuser.t_practice_student tps WHERE tps.practice_id=p.id AND tps.status=$1),0) as student_cnt
	from assessuser.t_practice p,
	left join assessuser.t_paper tp on tp.id = p.paper_id AND tp.status != $2 AND tp.status != $3
	where p.id = $4 AND `
	}
	stmt, err := sqlxDB.Preparex(s)
	if err != nil {
		err = fmt.Errorf("prepare sql err:%v", err)
		z.Error(err.Error())
		return &cmn.TPractice{}, "", 0, err
	}

	defer func() {
		err = stmt.Close()
		if err != nil || forceErr == "close" {
			err = fmt.Errorf("*sqlx.Stmt close failed:%v", err)
			z.Error(err.Error())
			return
		}
	}()
	var p cmn.TPractice
	var paperName string
	var studentCount int
	err = stmt.QueryRowxContext(ctx, PracticeStudentStatus.Normal, examPaper.PaperStatus.Deleted, examPaper.PaperStatus.Disabled, pid, PracticeStatus.Deleted).
		Scan(&p.ID, &p.Name, &p.CorrectMode,
			&p.Addi, &p.Status, &p.Type, &paperName, &p.AllowedAttempts, &p.PaperID, &p.ExamPaperID, &studentCount)
	if errors.Is(err, sql.ErrNoRows) {
		err = fmt.Errorf("非法practiceID , 无该练习记录:%v", err)
		z.Error(err.Error())
		return &cmn.TPractice{}, "", 0, err
	} else if err != nil || forceErr == "lQuery" {
		err = fmt.Errorf("LoadPracticeById call failed：%v", err)
		z.Error(err.Error())
		return &cmn.TPractice{}, "", 0, err
	} else {
		return &p, paperName, studentCount, nil
	}
}

// LoadPracticeByIDs 批量获取练习详情
/*
关键参数说明：
	ids 要查询的练习ID数组
返回参数说明：
	1、以练习ID为key，练习信息为value的map
	4、可能出现的错误
*/
func LoadPracticeByIDs(ctx context.Context, ids []int64) (map[int64]*cmn.TPractice, error) {
	if len(ids) == 0 {
		err := fmt.Errorf("非法practiceIDs:%v", ids)
		z.Error(err.Error())
		return nil, err
	}
	conn := cmn.GetPgxConn()
	result := make(map[int64]*cmn.TPractice)
	// 用于测试，强制执行某些错误分支
	forceErr, _ := ctx.Value("force-error").(string)
	s := `
	select p.id, p.name, p.correct_mode,p.addi,p.status,p.type,
			p.allowed_attempts,p.paper_id,p.exam_paper_id
	from assessuser.t_practice p
	where p.id = ANY($1) AND p.status != $2`
	rows, err := conn.Query(ctx, s, ids, PracticeStatus.Deleted)
	if err != nil || forceErr == "lQuery" {
		err = fmt.Errorf("批量查询练习数据失败：%v", err)
		z.Error(err.Error())
		return nil, err
	}

	defer func() {
		rows.Close()
	}()

	for rows.Next() {
		// 遍历，就需要创建了
		p := &cmn.TPractice{}
		err = rows.Scan(&p.ID, &p.Name, &p.CorrectMode, &p.Addi, &p.Status, &p.Type, &p.AllowedAttempts, &p.PaperID, &p.ExamPaperID)
		if err != nil || forceErr == "lScan" {
			err = fmt.Errorf("批量解析练习数据失败：%v", err)
			z.Error(err.Error())
			return nil, err
		}
		result[p.ID.Int64] = p
	}
	if len(result) == 0 {
		// 这里就直接报错
		err = fmt.Errorf("批量查询练习记录失败，记录为空")
		z.Error(err.Error())
		return nil, err
	}

	return result, nil
}

// ListPracticeS 学生权限及以下获取练习列表
/*
关键参数说明：条件查询
	pType 练习类型
	name 练习名称（模糊）
	difficulty 练习难度
	orderBy 排序顺序
	page 页号
	pageSize 页大小
	uid 操作人ID
*/
func ListPracticeS(ctx context.Context, pType, name, difficulty string, orderBy []string, page, pageSize int, uid int64) ([]*cmn.TVPracticeSummary, int, error) {
	result := make([]*cmn.TVPracticeSummary, 0)
	// 查询条件
	var clauses []string
	// 占位符
	var args []interface{}
	// 用于测试，强制执行某些错误分支
	forceErr, _ := ctx.Value("force-error").(string)
	if pType == "" {
		err := fmt.Errorf("invalid practice type param")
		z.Error(err.Error())
		return nil, 0, err
	}
	if name != "" {
		clauses = append(clauses, fmt.Sprintf("%s LIKE $%d", "name", len(args)+1))
		args = append(args, "%"+name+"%")
	}
	if difficulty != "" {
		clauses = append(clauses, fmt.Sprintf("%s = $%d", "difficulty", len(args)+1))
		args = append(args, difficulty)
	}
	clauses = append(clauses, fmt.Sprintf("type = $%d", len(args)+1))
	args = append(args, pType)
	clauses = append(clauses, fmt.Sprintf("practice_status = $%d", len(args)+1))
	args = append(args, PracticeStatus.Released)
	clauses = append(clauses, fmt.Sprintf("practice_student_status = $%d", len(args)+1))
	args = append(args, PracticeStudentStatus.Normal)
	clauses = append(clauses, fmt.Sprintf("student_id = $%d", len(args)+1))
	args = append(args, uid)

	s := `SELECT
		id,name,type,attempt_count,difficulty,allowed_attempts,question_count,wrong_count,
		total_score,highest_score,paper_total_score,paper_id,latest_unsubmitted_id,latest_submitted_id,pending_mark_id
		FROM assessuser.v_practice_summary`

	if len(clauses) > 0 {
		s += " WHERE " + strings.Join(clauses, " AND ")
	}
	// 添加ORDER BY子句
	if len(orderBy) > 0 {
		s += " ORDER BY " + strings.Join(orderBy, ", ")
	}
	// 添加分页参数
	if pageSize > 0 && pageSize <= 100 {
		s += fmt.Sprintf(" LIMIT $%d", len(args)+1)
		args = append(args, pageSize)
	}

	if page > 0 {
		offset := (page - 1) * pageSize
		s += fmt.Sprintf(" OFFSET $%d", len(args)+1)
		args = append(args, offset)
	}

	z.Sugar().Debugf("打印输出一下获取学生权限练习列表操作语句：%v", s)
	z.Sugar().Debugf("打印输出一下学生权限获取练习列表参数表：%v", args)
	sqlxDB := cmn.GetDbConn()
	rows, err := sqlxDB.QueryxContext(ctx, s, args...)
	if err != nil || forceErr == "sQuery1" {
		err = fmt.Errorf("查询学生权限练习列表失败：%v", err)
		z.Error(err.Error())
		return nil, 0, err
	}
	defer func() {
		err = rows.Close()
		if forceErr == "row close" {
			err = fmt.Errorf("关闭数据库连接数据失败")
		}
		if err != nil {
			z.Error(err.Error())
			return
		}
	}()
	// 遍历行数据
	for rows.Next() {
		var p cmn.TVPracticeSummary
		err = rows.Scan(&p.ID, &p.Name, &p.Type, &p.AttemptCount, &p.Difficulty, &p.AllowedAttempts,
			&p.QuestionCount, &p.WrongCount, &p.TotalScore, &p.HighestScore, &p.PaperTotalScore,
			&p.PaperID, &p.LatestUnsubmittedID, &p.LatestSubmittedID, &p.PendingMarkID)
		if err != nil || forceErr == "sQuery2" {
			err = fmt.Errorf("解析练习数据失败:%v", err)
			z.Error(err.Error())
			return nil, 0, err
		}
		result = append(result, &p)
	}
	return result, len(result), nil
}

// ListPracticeT 教师权限及以上获取练习列表
/*
关键参数说明：条件查询
	name 练习名称（模糊）
	pType 练习类型
	status 练习发布状态
	orderBy 排序顺序
	page 页号
	pageSize 页大小
	uid 操作人ID
*/
func ListPracticeT(ctx context.Context, name, pType, status string, orderBy []string, page, pageSize int, uid int64) ([]Map, int, error) {
	result := make([]Map, 0)
	// 用于测试，强制执行某些错误分支
	forceErr, _ := ctx.Value("force-error").(string)
	// 查询条件
	var clauses []string
	// 占位符
	var args []interface{}
	args = append(args, PracticeStudentStatus.Normal)
	// 占位符数值
	if name != "" {
		clauses = append(clauses, fmt.Sprintf("%s LIKE $%d", "tp.name", len(args)+1))
		args = append(args, "%"+name+"%")
	}
	if status != "" {
		clauses = append(clauses, fmt.Sprintf("%s = $%d", "tp.status", len(args)+1))
		args = append(args, status)
	}
	if pType != "" {
		clauses = append(clauses, fmt.Sprintf("%s = $%d", "tp.type", len(args)+1))
		args = append(args, pType)
	}
	clauses = append(clauses, fmt.Sprintf("tp.status != $%d", len(args)+1))
	args = append(args, PracticeStatus.Deleted)
	clauses = append(clauses, fmt.Sprintf("tp.creator = $%d", len(args)+1))
	args = append(args, uid)
	s := `
 	SELECT 
		tp.id, tp.name,tp.correct_mode,
		tp.type, tp.creator, tp.create_time, tp.updated_by, tp.update_time, tp.addi, tp.status ,tp.allowed_attempts,
		COALESCE((SELECT COUNT(*) FROM assessuser.t_practice_student tps WHERE tps.practice_id=tp.id AND status=$1),0) as student_cnt
		FROM assessuser.t_practice tp
	`
	if len(clauses) > 0 {
		s += " WHERE " + strings.Join(clauses, " AND ")
	}
	// 添加ORDER BY子句
	if len(orderBy) > 0 {
		s += " ORDER BY " + strings.Join(orderBy, ", ")
	}
	// 添加分页参数
	s += fmt.Sprintf(" LIMIT $%d", len(args)+1)
	args = append(args, pageSize)

	offset := (page - 1) * pageSize
	s += fmt.Sprintf(" OFFSET $%d", len(args)+1)
	args = append(args, offset)

	z.Sugar().Debugf("打印输出一下这个操作语句：%v", s)
	z.Sugar().Debugf("打印输出一下参数表：%v", args)

	// 这些实体查询的每个函数之间作用都不一样，需要花时间去了解这个函数的具体用处了
	sqlxDB := cmn.GetDbConn()
	rows, err := sqlxDB.QueryxContext(ctx, s, args...)
	if err != nil || forceErr == "query" {
		err = fmt.Errorf("search practice failed:%v", err)
		z.Error(err.Error())
		return nil, 0, err
	}
	defer func() {
		err = rows.Close()
		if err != nil || forceErr == "row close" {
			err = fmt.Errorf("row failed to close:%v", err)
			z.Error(err.Error())
			return
		}
	}()
	// 查询数据在这里
	for rows.Next() {
		M := Map{}
		var p cmn.TPractice
		var studentCount int64
		err = rows.Scan(&p.ID, &p.Name, &p.CorrectMode, &p.Type, &p.Creator,
			&p.CreateTime, &p.UpdatedBy, &p.UpdateTime, &p.Addi, &p.Status, &p.AllowedAttempts, &studentCount,
		)
		if err != nil || forceErr == "scan" {
			err = fmt.Errorf("解析练习数据失败:%v", err)
			z.Error(err.Error())
			return nil, 0, err
		}
		M["practice"] = p
		M["student_count"] = studentCount
		result = append(result, M)
	}
	return result, len(result), nil
}

// GetPracticeListByRegisterPlan 创建报名计划的账号 查看所有已发布的练习列表。要求是教师及以上
func GetPracticeListByRegisterPlan(ctx context.Context, practiceName, teacherName string, page, pageSize int, orderBy []string) ([]RegisterPractice, error) {
	// 定义的实际上就是一个简单的练习结构体的列表就可以了
	sqlxDB := cmn.GetDbConn()
	var result []RegisterPractice
	forceErr, _ := ctx.Value("force-error").(string)
	// 查询条件
	var clauses []string
	// 占位符
	var args []interface{}
	clauses = append(clauses, fmt.Sprintf("tp.status = $%d", len(args)+1))
	args = append(args, PracticeStatus.Released)
	// 占位符数值
	if practiceName != "" {
		clauses = append(clauses, fmt.Sprintf("%s LIKE $%d", "tp.name", len(args)+1))
		args = append(args, "%"+practiceName+"%")
	}
	if teacherName != "" {
		clauses = append(clauses, fmt.Sprintf("%s LIKE $%d", "u.official_name", len(args)+1))
		args = append(args, "%"+teacherName+"%")
	}

	s := `
 	SELECT 
		tp.id, tp.name,tp.correct_mode,
		tp.type, tp.creator, tp.create_time, tp.updated_by, tp.update_time, tp.addi, tp.status ,tp.allowed_attempts,u.official_name
		FROM assessuser.t_practice tp
		LEFT JOIN assessuser.t_user u ON u.id = tp.creator
	`
	if len(clauses) > 0 {
		s += " WHERE " + strings.Join(clauses, " AND ")
	}
	// 添加ORDER BY子句
	if len(orderBy) > 0 {
		s += " ORDER BY " + strings.Join(orderBy, ", ")
	}
	// 添加分页参数
	s += fmt.Sprintf(" LIMIT $%d", len(args)+1)
	args = append(args, pageSize)

	offset := (page - 1) * pageSize
	s += fmt.Sprintf(" OFFSET $%d", len(args)+1)
	args = append(args, offset)

	z.Sugar().Debugf("打印输出一下这个操作语句：%v", s)
	z.Sugar().Debugf("打印输出一下参数表：%v", args)
	rows, err := sqlxDB.QueryxContext(ctx, s, args...)
	if err != nil || forceErr == "query" {
		err = fmt.Errorf("查询可用于报名计划绑定的练习失败:%v", err)
		z.Error(err.Error())
		return nil, err
	}
	defer func() {
		err = rows.Close()
		if err != nil || forceErr == "Close" {
			err = fmt.Errorf("row failed to close:%v", err)
			z.Error(err.Error())
			return
		}
	}()
	// 查询数据在这里
	for rows.Next() {
		var p RegisterPractice
		err = rows.Scan(&p.ID, &p.Name, &p.CorrectMode, &p.Type, &p.Creator,
			&p.CreateTime, &p.UpdatedBy, &p.UpdateTime, &p.Addi, &p.Status, &p.AllowedAttempts, &p.TeacherName,
		)
		if err != nil || forceErr == "scan" {
			err = fmt.Errorf("解析练习数据失败:%v", err)
			z.Error(err.Error())
			return nil, err
		}
		result = append(result, p)
	}
	return result, nil
}

// OperatePracticeStatus 教师及以上权限操作练习发布状态 控制学生能否作答、能否在列表中查看到该练习 并配置批改信息
/*
关键参数：
	pid 练习ID
	status 想要切换的状态
	uid 操作者
*/
// OperatePracticeStatus 操作练习的发布状态 取消/发布/删除 练习
func OperatePracticeStatus(ctx context.Context, pid int64, status string, uid int64) error {
	var err error
	if pid <= 0 {
		err = fmt.Errorf("invalid practice ID param")
		z.Error(err.Error())
		return err
	}
	// 用于测试，强制执行某些错误分支
	forceErr, _ := ctx.Value("force-error").(string)
	conn := cmn.GetPgxConn()
	now := time.Now().UnixMilli()
	p, _, _, err := LoadPracticeById(ctx, pid)
	if err != nil {
		return err
	}
	tx, err := conn.Begin(ctx)
	if err != nil || forceErr == "beginTx" {
		err = fmt.Errorf("beginTx called failed:%v", err)
		z.Error(err.Error())
		return err
	}
	defer func() {
		if forceErr == "rollback" {
			err = fmt.Errorf("触发回滚")
		}
		if err != nil {
			// 操作失败回滚
			err = tx.Rollback(ctx)
			if forceErr == "rollback" {
				err = fmt.Errorf("触发回滚")
			}
			if err != nil {
				z.Error(err.Error())
				return
			}
		}
		// 无错误则提交
		err = tx.Commit(ctx)
		if forceErr == "commit" {
			err = fmt.Errorf("commit failed")
		}
		if err != nil {
			z.Error(err.Error())
		}

	}()
	if status == PracticeStatus.Released {
		// 这里需要重构一下这个东西
		var examPaperId int64
		s := `SELECT pa.exam_paper_id FROM t_practice p JOIN t_paper pa ON pa.id = p.paper_id WHERE p.id = $1 AND pa.status = $2`
		err = tx.QueryRow(ctx, s, pid, examPaper.PaperStatus.Published).Scan(&examPaperId)
		if err != nil {
			err = fmt.Errorf("查看练习绑定的试卷中已发布的考卷ID失败:%v", err)
			z.Error(err.Error())
			return err
		}

		s = `UPDATE assessuser.t_practice SET status = $1,update_time = $2, updated_by = $3 ,exam_paper_id = $4 WHERE id = $5`
		_, err = tx.Exec(ctx, s, status, now, uid, examPaperId, pid)
		if err != nil || forceErr == "pQuery1" {
			err = fmt.Errorf("更新练习状态 发布->未发布 失败:%v", err)
			z.Error(err.Error())
			return err
		}
		// 生成批改配置信息
		req := mark.HandleMarkerInfoReq{
			PracticeID: p.ID.Int64,
			MarkMode:   p.CorrectMode.String,
			Markers:    []int64{uid},
			Status:     "00",
		}

		err = mark.HandleMarkerInfo(ctx, &tx, uid, req)
		if err != nil || forceErr == "mark" {
			return err
		}
		return nil
	} else if status == PracticeStatus.Deleted {
		isAnswer := false
		s := `SELECT EXISTS(SELECT 1 FROM assessuser.t_practice_submissions WHERE practice_id = $1)`
		err = tx.QueryRow(ctx, s, p.ID.Int64).Scan(&isAnswer)
		if err != nil || forceErr == "pQuery2" {
			err = fmt.Errorf("遍历查询是否有学生作答记录失败：%v", err)
			z.Error(err.Error())
			return err
		}
		if isAnswer {
			err = fmt.Errorf("此时练习名称为：%v的练习已有学生参与作答，不能删除", p.Name.String)
			z.Error(err.Error())
			return err
		}

		// 若练习已经发布了，无法被删除，必须先回退为待发布状态后才能被删除 但是此时你无法通过LoadPracticeById这个函数去查询到已被删除的
		s = `UPDATE assessuser.t_practice SET status = $1,update_time = $2, updated_by = $3  WHERE id = $4`
		_, err = tx.Exec(ctx, s, status, now, uid, pid)
		if err != nil || forceErr == "pQuery3" {
			err = fmt.Errorf("更新练习状态 发布-> 未发布 或 未发布-> 删除 失败:%v", err)
			z.Error(err.Error())
			return err
		}
		// 清除批改配置信息
		req := mark.HandleMarkerInfoReq{
			Status:      "02",
			PracticeIDs: []int64{p.ID.Int64},
		}

		err = mark.HandleMarkerInfo(ctx, &tx, uid, req)
		if err != nil || forceErr == "mark1" {
			return err
		}
		return nil
	} else if status == PracticeStatus.Disabled {
		s := `UPDATE assessuser.t_practice SET status = $1,update_time = $2, updated_by = $3  WHERE id = $4`
		_, err = tx.Exec(ctx, s, status, now, uid, pid)
		if err != nil || forceErr == "pQuery5" {
			err = fmt.Errorf("更新练习状态 发布-> 作废失败:%v", err)
			z.Error(err.Error())
			return err
		}

		// 更改practice_submission练习学生的提交状态及其练习次数，将本次练习附带的所有次数均变为无效
		s = `UPDATE assessuser.t_practice_submissions SET status = $1,update_time = $2,updated_by = $3 WHERE practice_id = $4`
		_, err = tx.Exec(ctx, s, PracticeSubmissionStatus.Disabled, now, uid, pid)
		if err != nil || forceErr == "pQuery6" {
			err = fmt.Errorf("重置学生练习提交记录信息失败：%v", err)
			z.Error(err.Error())
			return err
		}
		s = `UPDATE assessuser.t_practice_wrong_submissions w
			SET 
			  status = $1,        
			  update_time = $2,  
			  updated_by = $3    
			FROM assessuser.t_practice_submissions ps
			WHERE 
			  w.practice_submission_id = ps.id 
			  AND ps.practice_id = $4;`
		_, err = tx.Exec(ctx, s, WrongSubmissionStatus.Disabled, now, uid, pid)
		if err != nil || forceErr == "pQuery7" {
			err = fmt.Errorf("批量作废学生错题练习提交记录信息失败：%v", err)
			z.Error(err.Error())
			return err
		}

		// 清除批改配置信息
		req := mark.HandleMarkerInfoReq{
			Status:      "02",
			PracticeIDs: []int64{p.ID.Int64},
		}

		err = mark.HandleMarkerInfo(ctx, &tx, uid, req)
		if err != nil || forceErr == "mark2" {
			return err
		}
		return nil
	} else {
		err = fmt.Errorf("传入要更换的练习status:%v 非法,请传入合法的练习状态", status)
		z.Error(err.Error())
		return err
	}
}

// OperatePracticeStatusV2 教师及以上权限批量操作练习发布状态 控制学生能否作答、能否在列表中查看到该练习 并配置批改信息
/*
关键参数：
	ids 练习ID数组
	status 想要切换的状态
	uid 操作者
*/
// OperatePracticeStatus 操作练习的发布状态 取消/发布/删除 练习
func OperatePracticeStatusV2(ctx context.Context, ids []int64, status string, uid int64) error {
	var err error
	// 用于测试，强制执行某些错误分支
	forceErr, _ := ctx.Value("force-error").(string)
	conn := cmn.GetPgxConn()
	now := time.Now().UnixMilli()
	ps, err := LoadPracticeByIDs(ctx, ids)
	if err != nil {
		return err
	}
	tx, err := conn.Begin(ctx)
	if err != nil || forceErr == "beginTx" {
		err = fmt.Errorf("beginTx called failed:%v", err)
		z.Error(err.Error())
		return err
	}
	defer func() {
		if forceErr == "rollback" {
			err = fmt.Errorf("触发回滚")
		}
		if err != nil {
			// 操作失败回滚
			err = tx.Rollback(ctx)
			if forceErr == "rollback" {
				err = fmt.Errorf("触发回滚")
			}
			if err != nil {
				z.Error(err.Error())
				return
			}
		}
		// 无错误则提交
		err = tx.Commit(ctx)
		if forceErr == "commit" {
			err = fmt.Errorf("commit failed")
		}
		if err != nil {
			z.Error(err.Error())
		}

	}()
	signStatus := ""
	for _, p := range ps {
		if signStatus == "" {
			signStatus = p.Status.String
		}
		if p.Status.String != signStatus {
			err = fmt.Errorf("此时要批量操作的练习状态不一，无法进行批量操作")
			z.Error(err.Error())
			return err
		}
		if p.Status.String == PracticeStatus.Disabled {
			err = fmt.Errorf("不能操作已作废的练习")
			z.Error(err.Error())
			return err
		}
	}
	if status == PracticeStatus.Released {
		// 批量操作
		for pid, p := range ps {
			// 这里都需要进行查询，需要找到这个practice对应的试卷里面的考卷ID
			var examPaperId int64
			s := `SELECT pa.exam_paper_id FROM t_practice p JOIN t_paper pa ON pa.id = p.paper_id WHERE p.id = $1 AND pa.status = $2`
			err = tx.QueryRow(ctx, s, pid, examPaper.PaperStatus.Published).Scan(&examPaperId)
			if err != nil {
				err = fmt.Errorf("查看练习绑定的试卷中已发布的考卷ID失败:%v", err)
				z.Error(err.Error())
				return err
			}

			s = `UPDATE assessuser.t_practice SET status = $1,update_time = $2, updated_by = $3 ,exam_paper_id = $4 WHERE id = $5`
			_, err = tx.Exec(ctx, s, status, now, uid, examPaperId, pid)
			if err != nil || forceErr == "pQuery1" {
				err = fmt.Errorf("更新练习状态 未发布->发布 失败:%v", err)
				z.Error(err.Error())
				return err
			}
			// 生成批改配置信息
			req := mark.HandleMarkerInfoReq{
				PracticeID: pid,
				MarkMode:   p.CorrectMode.String,
				Markers:    []int64{uid},
				Status:     "00",
			}

			err = mark.HandleMarkerInfo(ctx, &tx, uid, req)
			if forceErr == "mark" {
				err = fmt.Errorf("新增练习批改配置失败")
			}
			if err != nil {
				return err
			}
		}
		return nil
	} else if status == PracticeStatus.Deleted {
		// 进行批量操作
		tempIsAnswer := false
		var invalidName []string
		for _, p := range ps {
			s := `SELECT EXISTS(SELECT 1 FROM assessuser.t_practice_submissions WHERE practice_id = $1)`
			err = tx.QueryRow(ctx, s, p.ID.Int64).Scan(&tempIsAnswer)
			if err != nil || forceErr == "pQuery2" {
				err = fmt.Errorf("遍历查询是否有学生作答记录失败：%v", err)
				z.Error(err.Error())
				return err
			}
			// 就代表此时有学生作答过，就不能进行删除操作（包括批量删除）
			if tempIsAnswer {
				invalidName = append(invalidName, p.Name.String)
			}
		}

		if len(invalidName) > 0 {
			err = fmt.Errorf("此时练习名称为：%v的练习已有学生参与作答，不能删除", invalidName)
			z.Error(err.Error())
			return err
		}

		s := `UPDATE assessuser.t_practice SET status = $1,update_time = $2, updated_by = $3  WHERE id = ANY($4)`
		_, err = tx.Exec(ctx, s, status, now, uid, ids)
		if err != nil || forceErr == "pQuery3" {
			err = fmt.Errorf("更新练习状态 发布-> 删除 失败:%v", err)
			z.Error(err.Error())
			return err
		}
		// 清除批改配置信息
		req := mark.HandleMarkerInfoReq{
			Status:      "02",
			PracticeIDs: ids,
		}

		err = mark.HandleMarkerInfo(ctx, &tx, uid, req)
		if forceErr == "mark1" {
			err = fmt.Errorf("清除批改配置失败")
		}
		if err != nil {
			return err
		}
		return nil
	} else if status == PracticeStatus.Disabled {
		s := `UPDATE assessuser.t_practice SET status = $1,update_time = $2, updated_by = $3  WHERE id = ANY($4)`
		_, err = tx.Exec(ctx, s, status, now, uid, ids)
		if err != nil || forceErr == "pQuery5" {
			err = fmt.Errorf("更新练习状态 发布->作废 失败:%v", err)
			z.Error(err.Error())
			return err
		}
		// 更改practice_submission练习学生的提交状态及其练习次数，将本次练习附带的所有次数均变为无效
		s = `UPDATE assessuser.t_practice_submissions SET status = $1,update_time = $2,updated_by = $3  WHERE practice_id = ANY($4)`
		_, err = tx.Exec(ctx, s, PracticeSubmissionStatus.Disabled, now, uid, ids)
		if err != nil || forceErr == "pQuery6" {
			err = fmt.Errorf("批量重置学生练习提交记录信息失败：%v", err)
			z.Error(err.Error())
			return err
		}
		s = `UPDATE assessuser.t_practice_wrong_submissions w
			SET 
			  status = $1,        
			  update_time = $2,  
			  updated_by = $3    
			FROM assessuser.t_practice_submissions ps
			WHERE 
			  w.practice_submission_id = ps.id 
			  AND ps.practice_id = ANY($4);`
		_, err = tx.Exec(ctx, s, WrongSubmissionStatus.Disabled, now, uid, ids)
		if err != nil || forceErr == "pQuery7" {
			err = fmt.Errorf("批量作废学生错题练习提交记录信息失败：%v", err)
			z.Error(err.Error())
			return err
		}
		// 清除批改配置信息
		req := mark.HandleMarkerInfoReq{
			Status:      "02",
			PracticeIDs: ids,
		}

		err = mark.HandleMarkerInfo(ctx, &tx, uid, req)
		if forceErr == "mark2" {
			err = fmt.Errorf("清除批改配置失败")
		}
		if err != nil {
			return err
		}
		return nil

	} else {
		err = fmt.Errorf("传入要更换的练习status:%v 非法,请传入合法的练习状态", status)
		z.Error(err.Error())
		return err
	}
}

// EnterPracticeGetPaperDetails 学生进入练习作答所需试卷信息、练习基本信息
/*
关键参数说明：
	pid 练习唯一ID
	uid 用户唯一ID（学生唯一ID）
处理情况如下：
1、学生上次有作答，但无提交：
	返回携带学生作答信息的试卷题组、试卷题目、练习基本信息
2、学生上次作答已提交：
	生成新的提交记录，生成新的学生答卷，返回基本试卷题组、试卷题目、练习基本信息

返回参数说明：
	1、练习基本信息、试卷题目题组基本信息
	2、考卷题组信息 以题组ID分组（利用哈希表快速查询题目所在题组）
	3、根据题组ID分组的题目数组
*/
func EnterPracticeGetPaperDetails(ctx context.Context, tx pgx.Tx, pid int64, uid int64) (*EnterPracticeInfo, map[int64]*cmn.TExamPaperGroup, map[int64][]*examPaper.ExamQuestion, error) {
	// 去判断多种状态学生进入作答的状态
	if pid <= 0 || uid <= 0 {
		err := fmt.Errorf("invalid practiceID | uid param")
		z.Error(err.Error())
		return nil, nil, nil, err
	}
	TEST := "test"
	//查看是否需要返回mock的数据
	test, ok := ctx.Value(TEST).(string)
	if ok || test != "" {
		switch test {
		case "normal-withStudentAnswer-resp":
			{
				now := time.Now().UnixMilli()
				p := &cmn.TVExamPaper{}
				epInfo := &EnterPracticeInfo{}
				p.ID = null.IntFrom(101)
				p.ExamSessionID = null.IntFrom(201)
				p.PracticeID = null.IntFrom(201)
				p.Name = null.StringFrom("英语期末试卷")
				p.Creator = null.IntFrom(1)
				p.CreateTime = null.IntFrom(now)
				p.UpdateTime = null.IntFrom(now)
				p.Status = null.StringFrom("00")
				p.TotalScore = null.FloatFrom(6)
				p.QuestionCount = null.IntFrom(2)

				groupMap := make(map[int64]*cmn.TExamPaperGroup)
				groupMap[int64(200)] = &cmn.TExamPaperGroup{
					ID:    null.IntFrom(200),
					Name:  null.StringFrom("一、单选题（共1题，共3分）"),
					Order: null.IntFrom(1),
				}
				groupMap[int64(201)] = &cmn.TExamPaperGroup{
					ID:    null.IntFrom(201),
					Name:  null.StringFrom("二、填空题（共1题，共3分）"),
					Order: null.IntFrom(2),
				}

				questionMap := make(map[int64][]*examPaper.ExamQuestion)
				qList1 := make([]*examPaper.ExamQuestion, 0)
				q1 := &examPaper.ExamQuestion{}
				q1.ID = null.IntFrom(2042)
				q1.Type = null.StringFrom("00")
				q1.Title = null.StringFrom("")
				q1.Content = null.StringFrom("<p><span style=\"font-family: 等线; font-size: 12pt\">具有风险分析的软件生命周期模型是</span><span style=\"font-family: Aptos, sans-serif; font-size: 12pt\">()</span></p>")
				q1.Options = JSONText(`
                    [
                        {
                            "label": "A",
                            "value": "<p><span style=\"font-family: 等线; font-size: 12pt\">瀑布模型</span></p>"
                        },
                        {
                            "label": "B",
                            "value": "<p><span style=\"font-family: 等线; font-size: 12pt\">喷泉模型</span></p>"
                        },
                        {
                            "label": "C",
                            "value": "<p><span style=\"font-family: 等线; font-size: 12pt\">螺旋模型</span></p>"
                        },
                        {
                            "label": "D",
                            "value": "<p><span style=\"font-family: 等线; font-size: 12pt\">增量模型</span></p>"
                        }
                    ]
                `)
				q1.StudentAnswer = JSONText(`{
    			"type": "00",
    			"answer": [
        			"A"
    			],
    			"question_id": 2042
				}`)
				q1.Score = null.FloatFrom(3)
				q1.Order = null.IntFrom(1)
				q1.GroupID = null.IntFrom(200)
				qList1 = append(qList1, q1)

				qList2 := make([]*examPaper.ExamQuestion, 0)
				q2 := &examPaper.ExamQuestion{}
				q2.ID = null.IntFrom(2045)
				q2.Type = null.StringFrom("06")
				q2.Title = null.StringFrom("")
				q2.Content = null.StringFrom("<p><span style=\"font-family: 等线; font-size: 12pt\">用例之间的关系主要有三种：</span><span style=\"font-family: Aptos, sans-serif; font-size: 12pt\">(1)</span><span style=\"font-family: 等线; font-size: 12pt\">、</span><span style=\"font-family: Aptos, sans-serif; font-size: 12pt\">(2) </span><span style=\"font-family: 等线; font-size: 12pt\">和</span><span style=\"font-family: Aptos, sans-serif; font-size: 12pt\"> (3)</span></p>")
				q2.Options = JSONText(`[]`)
				q2.Score = null.FloatFrom(3)
				// 这里是给主观题使用的小题个数
				q2.AnswerNum = 3
				q1.StudentAnswer = JSONText(`{
    				"type": "06",
    				"answer": [
        				"<p><span style=\"font-size: 12pt\">包含</span></p>",
        				"<p><span style=\"font-size: 12pt\">继承</span></p>",
        				"<p><span style=\"font-size: 12pt\">1</span></p>"
    				],
    				"question_id": 2045
				}`)
				q2.Order = null.IntFrom(1)
				q2.GroupID = null.IntFrom(201)
				qList2 = append(qList2, q2)

				questionMap[int64(200)] = qList1
				questionMap[int64(201)] = qList2

				epInfo.PracticeSubmissionID = 159
				epInfo.PaperName = "英语期末试卷"
				epInfo.QuestionCount = 2
				epInfo.TotalScore = 6
				epInfo.GroupCount = 2

				return epInfo, groupMap, questionMap, nil

			}
		case "normal-resp":
			{
				now := time.Now().UnixMilli()
				p := &cmn.TVExamPaper{}
				epInfo := &EnterPracticeInfo{}
				p.ID = null.IntFrom(101)
				p.ExamSessionID = null.IntFrom(201)
				p.PracticeID = null.IntFrom(201)
				p.Name = null.StringFrom("英语期末试卷")
				p.Creator = null.IntFrom(1)
				p.CreateTime = null.IntFrom(now)
				p.UpdateTime = null.IntFrom(now)
				p.Status = null.StringFrom("00")
				p.TotalScore = null.FloatFrom(6)
				p.QuestionCount = null.IntFrom(2)

				groupMap := make(map[int64]*cmn.TExamPaperGroup)
				groupMap[int64(200)] = &cmn.TExamPaperGroup{
					ID:    null.IntFrom(200),
					Name:  null.StringFrom("一、单选题（共1题，共3分）"),
					Order: null.IntFrom(1),
				}
				groupMap[int64(201)] = &cmn.TExamPaperGroup{
					ID:    null.IntFrom(201),
					Name:  null.StringFrom("二、填空题（共1题，共3分）"),
					Order: null.IntFrom(2),
				}

				questionMap := make(map[int64][]*examPaper.ExamQuestion)
				qList1 := make([]*examPaper.ExamQuestion, 0)
				q1 := &examPaper.ExamQuestion{}
				q1.ID = null.IntFrom(2042)
				q1.Type = null.StringFrom("00")
				q1.Title = null.StringFrom("")
				q1.Content = null.StringFrom("<p><span style=\"font-family: 等线; font-size: 12pt\">具有风险分析的软件生命周期模型是</span><span style=\"font-family: Aptos, sans-serif; font-size: 12pt\">()</span></p>")
				q1.Options = JSONText(`
                    [
                        {
                            "label": "A",
                            "value": "<p><span style=\"font-family: 等线; font-size: 12pt\">瀑布模型</span></p>"
                        },
                        {
                            "label": "B",
                            "value": "<p><span style=\"font-family: 等线; font-size: 12pt\">喷泉模型</span></p>"
                        },
                        {
                            "label": "C",
                            "value": "<p><span style=\"font-family: 等线; font-size: 12pt\">螺旋模型</span></p>"
                        },
                        {
                            "label": "D",
                            "value": "<p><span style=\"font-family: 等线; font-size: 12pt\">增量模型</span></p>"
                        }
                    ]
                `)
				q1.Score = null.FloatFrom(3)
				q1.Order = null.IntFrom(1)
				qList1 = append(qList1, q1)

				qList2 := make([]*examPaper.ExamQuestion, 0)
				q2 := &examPaper.ExamQuestion{}
				q2.ID = null.IntFrom(2045)
				q2.Type = null.StringFrom("06")
				q2.Title = null.StringFrom("")
				q2.Content = null.StringFrom("<p><span style=\"font-family: 等线; font-size: 12pt\">用例之间的关系主要有三种：</span><span style=\"font-family: Aptos, sans-serif; font-size: 12pt\">(1)</span><span style=\"font-family: 等线; font-size: 12pt\">、</span><span style=\"font-family: Aptos, sans-serif; font-size: 12pt\">(2) </span><span style=\"font-family: 等线; font-size: 12pt\">和</span><span style=\"font-family: Aptos, sans-serif; font-size: 12pt\"> (3)</span></p>")
				q2.Options = JSONText(`[]`)
				q2.Score = null.FloatFrom(3)
				// 这里是给主观题使用的小题个数
				q2.AnswerNum = 3
				q2.Order = null.IntFrom(1)
				qList2 = append(qList2, q2)

				questionMap[int64(200)] = qList1
				questionMap[int64(201)] = qList2

				epInfo.PracticeSubmissionID = 159
				epInfo.PaperName = "英语期末试卷"
				epInfo.QuestionCount = 2
				epInfo.TotalScore = 6
				epInfo.GroupCount = 2

				return epInfo, groupMap, questionMap, nil
			}
		}
	}
	var ps cmn.TVPracticeSummary
	sqlxDB := cmn.GetDbConn()
	now := time.Now().UnixMilli()
	// 用于测试，强制执行某些错误分支
	forceErr, _ := ctx.Value("force-error").(string)
	// 判断学生作答练习的情况
	var submissionStatus string

	// 这里先根据练习ID跟userId去获取一下这个练习状态 去查这个last_unSubmitted_id然后能根据这个ID去拿出这个 这里有可能是学生根据没有一次提交
	// 也就是说此时是第一次进入，那就需要创建新的submissions的，同理如果查询出有记录，但是last为空的话，仍然需要创建，否则就不需要创建
	s := `SELECT allowed_attempts,attempt_count,latest_unsubmitted_id, latest_submitted_id,pending_mark_id, exam_paper_id,paper_name,suggested_duration
	 FROM assessuser.v_practice_summary 
	 WHERE id = $1 AND student_id = $2 AND practice_status = $3 
	 AND practice_student_status != $4`
	err := sqlxDB.QueryRowxContext(ctx, s, pid, uid, PracticeStatus.Released, PracticeStudentStatus.Deleted).Scan(&ps.AllowedAttempts, &ps.AttemptCount, &ps.LatestUnsubmittedID,
		&ps.LatestSubmittedID, &ps.PendingMarkID, &ps.ExamPaperID,
		&ps.PaperName, &ps.SuggestedDuration)
	if err != nil {
		err = fmt.Errorf("查询学生练习视图 v_practice_summary失败:%v", err)
		z.Error(err.Error())
		return nil, nil, nil, err
	}
	// 证明此时有练习需要等待批改，所以不能进行再一次的作答的
	if ps.PendingMarkID.Valid && ps.PendingMarkID.Int64 > 0 {
		err = fmt.Errorf("目前处于待批改的状态，无法重新进入作答")
		z.Error(err.Error())
		return nil, nil, nil, err
	}

	z.Sugar().Debugf("打印输出一下查询出来的数据:%v,%v,%v", ps.AttemptCount, ps.LatestSubmittedID, ps.LatestUnsubmittedID)
	if ps.LatestUnsubmittedID.Int64 == 0 && ps.LatestSubmittedID.Int64 == 0 {
		// 如果两个值均等于0的话，那就没有过练习记录，
		submissionStatus = StudentSubmissionStatus.NeverAnswer
	} else if ps.LatestUnsubmittedID.Int64 != 0 {
		// 如果上一次练习提交ID不等于0且上一次记录已提交 那就有未提交的练习记录，
		submissionStatus = StudentSubmissionStatus.UnSubmitted
	} else {
		// 否则就都是已经提交的状态
		submissionStatus = StudentSubmissionStatus.Submitted
	}

	var pSubmissionID int64
	epInfo := EnterPracticeInfo{}
	withStudentAnswer := false
	switch submissionStatus {
	//以前所有的记录均已提交，现在重新练习
	case StudentSubmissionStatus.Submitted:
		{
			if ps.AllowedAttempts.Int64 != 0 && ps.AllowedAttempts.Int64 <= ps.AttemptCount.Int64 {
				// 学生进入练习次数已经满了，无法再继续获取
				err = fmt.Errorf("已达练习最大次数:%v，无法再次进入练习", ps.AllowedAttempts.Int64)
				z.Error(err.Error())
				return nil, nil, nil, err
			}
			newAttempt := ps.AttemptCount.Int64 + 1
			s := `INSERT INTO assessuser.t_practice_submissions (practice_id,student_id,exam_paper_id,creator,create_time,update_time,attempt) VALUES (
			$1,$2,$3,$4,$5,$6,$7
		) RETURNING id`
			err = tx.QueryRow(ctx, s, pid, uid, ps.ExamPaperID, uid, now, now, newAttempt).Scan(&pSubmissionID)
			if err != nil || forceErr == "pQuery1" {
				err = fmt.Errorf("新增一个学生二次练习作答记录失败:%v", err)
				z.Error(err.Error())
				return nil, nil, nil, err
			}
			r := examPaper.GenerateAnswerQuestionsRequest{
				ExamPaperID:          ps.ExamPaperID.Int64,
				Category:             examPaper.PaperCategory.Practice,
				PracticeSubmissionID: []int64{pSubmissionID},
				IsOptionRandom:       true,
				IsQuestionRandom:     true,
				Attempt:              newAttempt,
			}
			// 生成学生答卷
			err = examPaper.GenerateAnswerQuestion(ctx, tx, r, uid)
			if err != nil {
				return nil, nil, nil, err
			}
			withStudentAnswer = false
		}
		//第一次进入练习 没有任何练习提交记录  第一次进入就可以拿到
	case StudentSubmissionStatus.NeverAnswer:
		{
			//在创建记录之前，需要先加载一下练习的基本信息
			p, _, _, err := LoadPracticeById(ctx, pid)
			if forceErr == "LoadPracticeById" {
				err = fmt.Errorf("LoadPracticeById call faild")
			}
			if err != nil {
				return nil, nil, nil, err
			}
			s := `INSERT INTO assessuser.t_practice_submissions (practice_id,student_id,exam_paper_id,creator,create_time,update_time,attempt) VALUES (
					$1,$2,$3,$4,$5,$6,$7
				  ) RETURNING id`
			err = tx.QueryRow(ctx, s, pid, uid, p.ExamPaperID, uid, now, now, 1).Scan(&pSubmissionID)
			if err != nil || forceErr == "pQuery2" {
				err = fmt.Errorf("初始化一个学生练习作答记录失败:%v", err)
				z.Error(err.Error())
				return nil, nil, nil, err
			}
			r := examPaper.GenerateAnswerQuestionsRequest{
				ExamPaperID:          p.ExamPaperID.Int64,
				Category:             examPaper.PaperCategory.Practice,
				PracticeSubmissionID: []int64{pSubmissionID},
				IsOptionRandom:       true,
				IsQuestionRandom:     true,
				Attempt:              1,
			}
			// 生成学生答卷
			err = examPaper.GenerateAnswerQuestion(ctx, tx, r, uid)
			if err != nil {
				return nil, nil, nil, err
			}
			withStudentAnswer = false
		}
	case StudentSubmissionStatus.UnSubmitted:
		{
			withStudentAnswer = true
			pSubmissionID = ps.LatestUnsubmittedID.Int64
		}
	}

	// 获取以上三种情况之后，根据参数传入加载此时学生作答应该查看的试卷
	p, pg, pq, err := examPaper.LoadExamPaperDetailByUserId(ctx, tx, ps.ExamPaperID.Int64, pSubmissionID, 0, withStudentAnswer, false, false)
	if err != nil {
		return nil, nil, nil, err
	}
	epInfo.PracticeSubmissionID = pSubmissionID
	epInfo.PaperName = ps.PaperName.String
	epInfo.QuestionCount = p.QuestionCount.Int64
	epInfo.TotalScore = p.TotalScore.Float64
	epInfo.GroupCount = p.GroupCount.Int64
	epInfo.Duration = ps.SuggestedDuration.Int64

	return &epInfo, pg, pq, nil

}

// EnterPracticeWrongCollection 学生进入错题集详情 练习最近的一次练习提交做错的题目
/*
关键参数说明：
	pid 练习唯一ID
	uid 用户唯一ID（学生唯一ID）

调用时机说明：
	只要是进入练习错题 就调用这个函数 包括第一次作答错题、继续作答错题、重新作答错题

返回参数说明：
	1、练习基本信息、试卷题目题组基本信息 （其中的practice_submission_id用于更新学生作答；wrong_submission_id用于控制作答错题的时间与状态）
	2、考卷题组信息 以题组ID分组（利用哈希表快速查询题目所在题组）
	3、根据题组ID分组的题目数组
*/

func EnterPracticeWrongCollection(ctx context.Context, tx pgx.Tx, pid, uid int64) (*EnterPracticeInfo, map[int64]*cmn.TExamPaperGroup, map[int64][]*examPaper.ExamQuestion, error) {
	// 这里是进入错题集的函数了 这里还是需要先判断一下这个summary的
	if pid <= 0 || uid <= 0 {
		err := fmt.Errorf("invalid practiceID | uid param")
		z.Error(err.Error())
		return nil, nil, nil, err
	}
	// 用于测试，强制执行某些错误分支
	forceErr, _ := ctx.Value("force-error").(string)
	// 这里要先获取 ，并不需要获取了，直接查这个视图，获取里面的错题 但是需要进行参数检测，假设他没有错题的话，或者是此时都没有进行作答的话呢？那就不需要了
	//所以需要先查询一下 practice_summary了，去看看是否已经提交了
	s := `SELECT latest_submitted_id,latest_unsubmitted_id,pending_mark_id,wrong_count
	 FROM assessuser.v_practice_summary 
	 WHERE id = $1 AND student_id = $2`

	var ps cmn.TVPracticeSummary
	now := time.Now().UnixMilli()
	err := tx.QueryRow(ctx, s, pid, uid).Scan(&ps.LatestSubmittedID, &ps.LatestUnsubmittedID, &ps.PendingMarkID, &ps.WrongCount)
	if err != nil || forceErr == "cQuery1" {
		err = fmt.Errorf("查询学生练习记录信息失败：%v", err)
		z.Error(err.Error())
		return nil, nil, nil, err
	}
	// 进入这个分支就代表此时学生已经开启了一次新的练习提交记录了 因此不允许再次进入错题集
	if ps.LatestSubmittedID.Int64 == 0 {
		err = fmt.Errorf("请求学生错题集失败 , 此时学生没有提交过练习，无法进入错题练习")
		z.Error(err.Error())
		return nil, nil, nil, err
	}
	if ps.LatestUnsubmittedID.Int64 > 0 {
		err = fmt.Errorf("请求学生错题集失败 , 此时学生已经进入一次练习，但是未提交，无法进入错题练习")
		z.Error(err.Error())
		return nil, nil, nil, err
	}
	if ps.PendingMarkID.Int64 > 0 {
		err = fmt.Errorf("请求学生错题集失败 , 此时学生已经进入一次练习，已提交但是待教师批改出成绩，无法进入错题练习")
		z.Error(err.Error())
		return nil, nil, nil, err
	}
	if ps.WrongCount.Int64 == 0 {
		err = fmt.Errorf("请求学生错题集失败 , 学生最新练习提交记录中无错题，无需进入错题练习")
		z.Error(err.Error())
		return nil, nil, nil, err
	}

	s = `SELECT latest_unsubmitted_id,latest_submitted_id,max_attempt
	 FROM assessuser.v_w_practice_summary 
	 WHERE practice_id = $1 AND student_id = $2`
	var wps cmn.TVWPracticeSummary
	err = tx.QueryRow(ctx, s, pid, uid).Scan(&wps.LatestUnsubmittedID, &wps.LatestSubmittedID, &wps.MaxAttempt)
	if err != nil || forceErr == "cQuery2" {
		err = fmt.Errorf("查询学生错题练习记录信息失败：%v", err)
		z.Error(err.Error())
		return nil, nil, nil, err
	}
	// 这里开始分流 到底是查询新的一次错题练习 还是需要携带此时学生作答的呢
	// 判断学生作答错题的情况 这里的practice_submission_id是用于找到对应学生答题情况的 而wrong_submission_id是用于处理错题提交作答途中的时间的
	epInfo := &EnterPracticeInfo{}
	var wpSubmissionID int64
	var wrongSubmissionStatus string
	var withStudentAnswer bool
	epInfo.PracticeSubmissionID = ps.LatestSubmittedID.Int64
	if wps.LatestSubmittedID.Int64 == 0 && wps.LatestUnsubmittedID.Int64 == 0 {
		// 此时就是从来都没有进入过本次的错题集的
		wrongSubmissionStatus = StudentSubmissionStatus.NeverAnswer
	} else if wps.LatestUnsubmittedID.Int64 > 0 {
		wrongSubmissionStatus = StudentSubmissionStatus.UnSubmitted
	} else {
		wrongSubmissionStatus = StudentSubmissionStatus.Submitted
	}
	// 根据三种情况进行获取不同的信息并且需要创建对应的记录才行
	switch wrongSubmissionStatus {
	case StudentSubmissionStatus.UnSubmitted:
		{
			withStudentAnswer = true
			wpSubmissionID = wps.LatestUnsubmittedID.Int64
		}
	case StudentSubmissionStatus.Submitted:
		{
			withStudentAnswer = false
			s = `INSERT INTO t_practice_wrong_submissions (practice_submission_id,attempt,creator,create_time,update_time,status)VALUES(
				$1,$2,$3,$4,$5,$6
			) RETURNING id`
			err = tx.QueryRow(ctx, s, ps.LatestSubmittedID, wps.MaxAttempt.Int64+1, uid, now, now, "00").Scan(&wpSubmissionID)
			if err != nil || forceErr == "cQuery3" {
				err = fmt.Errorf("创建错题练习提交记录失败：%v", err)
				z.Error(err.Error())
				return nil, nil, nil, err
			}
		}
	case StudentSubmissionStatus.NeverAnswer:
		{
			s = `INSERT  INTO t_practice_wrong_submissions (practice_submission_id,attempt,creator,create_time,update_time,status)VALUES(
				$1,$2,$3,$4,$5,$6
			) RETURNING id`
			err = tx.QueryRow(ctx, s, ps.LatestSubmittedID, 1, uid, now, now, "00").Scan(&wpSubmissionID)
			if err != nil || forceErr == "cQuery4" {
				err = fmt.Errorf("创建错题练习提交记录失败：%v", err)
				z.Error(err.Error())
				return nil, nil, nil, err
			}
			withStudentAnswer = false
		}
	}
	epInfo, pg, pq, err := LoadErrorCollectionDetailsById(ctx, tx, pid, uid, false, false)
	if err != nil {
		return nil, nil, nil, err
	}
	// 构建前端需要的题组结构体
	groupMap := make(map[int64]*cmn.TExamPaperGroup)
	for _, g := range pg {
		groupMap[g.ID.Int64] = g
	}
	epInfo.WrongSubmissionID = wpSubmissionID
	epInfo.PaperName = ps.PaperName.String + "（错题）"
	epInfo.PracticeSubmissionID = ps.LatestSubmittedID.Int64

	// 更新错题为可作答状态 如果是需要重新组装错题，需要将原
	if wrongSubmissionStatus != StudentSubmissionStatus.UnSubmitted {
		s = `UPDATE t_student_answers sa  
			SET status = $1 
			FROM t_exam_paper_question epq 
			WHERE epq.id = sa.question_id       
  			AND sa.practice_submission_id = $2 
			AND sa.answer_score < epq.score;`
		_, err = tx.Exec(ctx, s, "00", ps.LatestSubmittedID)
		if err != nil || forceErr == "cQuery6" {
			err = fmt.Errorf("更新学生答卷为可作答状态失败：%v", err)
			z.Error(err.Error())
			return nil, nil, nil, err
		}
	}

	if !withStudentAnswer {
		return epInfo, groupMap, pq, nil
	}
	// 否则的话，那就需要去查询一下学生的作答
	var sAnswer []*cmn.TStudentAnswers
	s = `SELECT 
			sa.question_id,
			sa."order",
			sa.answer
		FROM assessuser.t_student_answers sa
		JOIN assessuser.t_exam_paper_question q ON sa.question_id = q.id
		WHERE q.score > sa.answer_score AND sa.practice_submission_id = $1
		ORDER BY sa."order"
		`
	rows, err := tx.Query(ctx, s, ps.LatestSubmittedID)
	if err != nil || forceErr == "cQuery5" {
		err = fmt.Errorf("query statement failed:%v", err)
		z.Error(err.Error())
		return nil, nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var a cmn.TStudentAnswers
		// 将属于这个学生的真正题目、学生作答等信息获取出来，解析在答卷结构体中，最后经过循环遍历，嵌入成一个完整的题目
		err = rows.Scan(&a.QuestionID, &a.Order, &a.Answer)
		if err != nil || forceErr == "scan" {
			err = fmt.Errorf("scan student answer failed:%v", err)
			z.Error(err.Error())
			return nil, nil, nil, err
		}
		sAnswer = append(sAnswer, &a)
	}
	if len(sAnswer) == 0 || forceErr == "emptyAnswer" {
		err = fmt.Errorf("query empty student answer, please checkout generate exam paper or student answer")
		z.Error(err.Error())
		return nil, nil, nil, err
	}

	questionIndex := make(map[int64]*examPaper.ExamQuestion)
	for _, questions := range pq {
		for _, q := range questions {
			questionIndex[q.ID.Int64] = q
		}
	}
	for _, sa := range sAnswer {
		tq, exists := questionIndex[sa.QuestionID.Int64]
		if forceErr == "exist" {
			exists = false
		}
		if !exists {
			err = fmt.Errorf("exam question id:%v not found in exam paper", sa.QuestionID.Int64)
			z.Error(err.Error())
			return nil, nil, nil, err
		}
		// 赋值学生作答情况
		tq.StudentAnswer = sa.Answer
		// 赋值真实记录乱序之后的学生答卷
		tq.Order = sa.Order
	}

	return epInfo, groupMap, pq, nil

}

// LoadErrorCollectionDetailsById 从最新一次练习提交记录的错题集作答中提取获取错题集的试题 包括是否包含答案、解析等 这里的答案与解析，应该不能进行删除或者是获取，必须
// 参数是一样的，只不过是需要判断到底需要的是哪个？那就直接合成到一个函数中 也就是说此时到底是需要从哪个视图进行查询的话
func LoadErrorCollectionDetailsById(ctx context.Context, tx pgx.Tx, pid, uid int64, withAnswers, withAnalysis bool) (*EnterPracticeInfo, []*cmn.TExamPaperGroup, map[int64][]*examPaper.ExamQuestion, error) {
	var err error
	if pid <= 0 || uid <= 0 {
		err = fmt.Errorf("invalid pid or uid param")
		z.Error(err.Error())
		return nil, nil, nil, err
	}
	//查看是否需要返回mock的数据
	forceErr, _ := ctx.Value("force-error").(string)
	// 一张考卷卷拥有的题组map
	var examGroups []*cmn.TExamPaperGroup
	// 一个题组下拥有的题目数组
	examQuestions := make(map[int64][]*examPaper.ExamQuestion)
	var groupData []examPaper.ExamGroup
	p := &EnterPracticeInfo{}

	var pwc cmn.TVZSubmissionWrongCollection
	s := `SELECT id,name,total_score,question_count,group_count,groups_data  FROM assessuser.v_z_submission_wrong_collection WHERE practice_id = $1 AND student_id = $2`
	err = tx.QueryRow(ctx, s, pid, uid).Scan(&pwc.ID, &pwc.Name, &pwc.TotalScore, &pwc.QuestionCount, &pwc.GroupCount, &pwc.GroupsData)
	if err != nil || forceErr == "ecQuery1" {
		err = fmt.Errorf("查询学生错题集信息失败：%v", err)
		z.Error(err.Error())
		return nil, nil, nil, err
	}
	if forceErr == "json" {
		pwc.GroupsData = types.JSONText(`invalid json: missing closing brace`)
	}
	if forceErr == "jsonA" {
		pwc.GroupsData = types.JSONText(`[]`) // 空数组
	}
	err = json.Unmarshal(pwc.GroupsData, &groupData)
	if err != nil {
		err = fmt.Errorf("unmarshal group data failed:%v", err)
		z.Error(err.Error())
		return nil, nil, nil, err
	}
	pwc.GroupsData = nil
	p.PaperName = pwc.Name.String
	p.TotalScore = pwc.TotalScore.Float64
	p.GroupCount = pwc.GroupCount.Int64
	p.QuestionCount = int64(pwc.QuestionCount.Float64)

	if len(groupData) == 0 {
		err = fmt.Errorf("数据错误：此时返回的错题数组为空！")
		z.Error(err.Error())
		return nil, nil, nil, err
	}
	for _, v := range groupData {
		examGroups = append(examGroups, &cmn.TExamPaperGroup{
			ID:          v.ID,
			ExamPaperID: v.ExamPaperID,
			Name:        v.Name,
			Order:       v.Order,
			Creator:     v.Creator,
			CreateTime:  v.CreateTime,
			UpdatedBy:   v.UpdatedBy,
			UpdateTime:  v.UpdateTime,
			Addi:        v.Addi,
			Status:      v.Status,
		})
		if _, exists := examQuestions[v.ID.Int64]; !exists {
			examQuestions[v.ID.Int64] = make([]*examPaper.ExamQuestion, 0)
		}
		// 保留答案与解析
		for idx := range v.Questions {
			q := v.Questions[idx]
			var answersSlice []interface{}
			if len(q.Answers) > 0 {
				if forceErr == "jsonB" {
					q.Answers = types.JSONText(`invalid json: missing closing brace`)
				}
				if err = json.Unmarshal(q.Answers, &answersSlice); err != nil {
					err = fmt.Errorf("failed to unmarshal Answers questionId:%v for:%v", q.ID.Int64, err)
					z.Error(err.Error())
					return nil, nil, nil, err
				}
			}
			q.AnswerNum = len(answersSlice)
			if !withAnalysis {
				q.Analysis = null.String{}
			}
			if !withAnswers {
				q.Answers = nil
			}
			examQuestions[v.ID.Int64] = append(examQuestions[v.ID.Int64], &q)
		}
	}

	return p, examGroups, examQuestions, nil
}

// BoundPracticeEnterRegisterPlan 学生成功报名一个报名计划后 绑定计划中携带的练习
/*
调用时机：
	无论此时报名计划中有没有附带绑定配套练习 都调用这个函数
传参：
	学生ID
	报名计划ID
*/
func BoundPracticeEnterRegisterPlan(ctx context.Context, tx pgx.Tx, uid, rpid int64) error {
	var err error
	if uid <= 0 || rpid <= 0 {
		err = fmt.Errorf("invalid uid or rpid param")
		z.Error(err.Error())
		return err
	}
	//查看是否需要返回mock的数据
	forceErr, _ := ctx.Value("force-error").(string)
	now := time.Now().UnixMilli()
	// 因此这需要查询上所有这些的练习id，并且查看这些练习是否已经作废了等等，这里仅仅只给已发布或者待发布的练习进行新增这个名额
	s := `SELECT practice_id FROM t_register_practice rp 
			LEFT JOIN t_practice p ON p.id = rp.practice_id
			WHERE rp.register_id = $1 AND p.status = $2	`
	rows, err := tx.Query(ctx, s, rpid, PracticeStatus.Released)
	if err != nil || forceErr == "query" {
		err = fmt.Errorf("查询报名计划所绑定的练习失败:%v", err)
		z.Error(err.Error())
		return err
	}

	defer rows.Close()

	var pids []int64
	for rows.Next() {
		var pid int64
		err = rows.Scan(&pid)
		if err != nil || forceErr == "scan" {
			err = fmt.Errorf("扫描解析数据库数据失败:%v", err)
			z.Error(err.Error())
			return err
		}
		pids = append(pids, pid)
	}
	if len(pids) == 0 {
		// 此时查询不到当前这个报名计划中所拥有的练习 ，那就直接返回，表示成功
		return nil
	}
	s = `
    INSERT INTO assessuser.t_practice_student 
        (student_id, practice_id, creator, create_time, status)
    SELECT $1, UNNEST($2::bigint[]), $3, $4, $5
    ON CONFLICT (student_id, practice_id) 
    DO NOTHING`
	_, err = tx.Exec(ctx, s, uid, pids, uid, now, PracticeStudentStatus.Normal)
	if err != nil || forceErr == "query1" {
		err = fmt.Errorf("执行插入学生练习名单数据库失败:%v", err)
		z.Error(err.Error())
		return err
	}
	return nil
}
