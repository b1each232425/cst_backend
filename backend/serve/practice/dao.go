package practice

//annotation:practice-service
//author:{"name":"ZouDeLun","tel":"15920422045", "email":"1311866870@qq.com"}
import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/jmoiron/sqlx"
	"strings"
	"time"
	"w2w.io/cmn"
	"w2w.io/null"
	"w2w.io/serve/examPaper"
)

// UpsertPractice 新增/修改练习信息 根据用户传输的信息动态构建SQL语句
func UpsertPractice(ctx context.Context, p *cmn.TPractice, ps []cmn.TPracticeStudent, uid int64) error {
	if uid <= 0 {
		err := fmt.Errorf("invalid updator ID param")
		z.Error(err.Error())
		return err
	}
	if p.ID.ValueOrZero() <= 0 {
		return AddPractice(ctx, p, ps, uid)
	}
	p2, err := LoadPracticeById(ctx, p.ID.ValueOrZero())
	if err != nil {
		z.Error(err.Error())
		return err
	}
	if p2.Status.String == PracticeStatus.Released {
		return fmt.Errorf("练习已经发布，不可修改练习信息")
	}
	return UpdatePractice(ctx, p, ps, uid)

}

// UpdatePractice 更新练习本身信息
func UpdatePractice(ctx context.Context, p *cmn.TPractice, ps []cmn.TPracticeStudent, uid int64) error {
	if uid <= 0 {
		err := fmt.Errorf("invalid updator ID param")
		z.Error(err.Error())
		return err
	}
	p.UpdatedBy = null.IntFrom(uid)
	p.UpdateTime = null.IntFrom(Timestamp(time.Now()))
	update, err := S2Map(p)
	if err != nil {
		err = fmt.Errorf("invalid practice params:%v", err)
		z.Error(err.Error())
		return err
	}
	notUpdate := []string{
		"id",
		"creator",
		"create_time",
	}

	RemoveFields(update, notUpdate...)
	z.Sugar().Debugf("update:%v", Json(update))
	tableName := p.GetTableName()
	var clauses []string
	var args []interface{}
	idx := 1
	for field, value := range update {
		clauses = append(clauses, fmt.Sprintf("%s = $%d", field, idx))
		args = append(args, value)
		idx++
	}
	args = append(args, uid)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = $%d", tableName, strings.Join(clauses, ", "), idx)
	// 这里执行他同一个事务，但是不会进行回滚的
	sqlxDB := cmn.GetDbConn()
	_, err = sqlxDB.ExecContext(ctx, query, args...)
	if err != nil {
		err = fmt.Errorf("updatePractice call failed:%v", err)
		z.Error(err.Error())
		return err
	}
	err = UpsertPracticeStudent(ctx, p.ID.Int64, uid, ps)
	if err != nil {
		err = fmt.Errorf("UpsertPracticeStudent call failed:%v", err)
		z.Error(err.Error())
		return err
	}

	return nil

}

// AddPractice 添加一场练习 包括插入成功导入的学生
// TODO 需对接学生管理接口
func AddPractice(ctx context.Context, p *cmn.TPractice, ps []cmn.TPracticeStudent, uid int64) error {
	var id int64

	p.Creator = null.IntFrom(uid)
	p.CreateTime = null.IntFrom(Timestamp(time.Now()))
	if !p.Status.Valid {
		p.Status = null.StringFrom(PracticeStatus.PendingRelease)
	}
	sqlxDB := cmn.GetDbConn()
	s := `
	INSERT INTO assessuser.t_practice (name , correct_mode,creator,create_time,updated_by, update_time, addi, status,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING id`
	err := sqlxDB.QueryRowxContext(ctx, s, p.Name, p.CorrectMode, p.Creator, p.CreateTime, p.UpdatedBy, p.UpdateTime, p.Addi, p.Status, p.AllowedAttempts, p.Type, p.PaperID).Scan(&id)
	if err != nil {
		err = fmt.Errorf("addPractice called failed:%v", err)
		z.Error(err.Error())
		return err
	}
	p.ID = null.IntFrom(id)
	err = UpsertPracticeStudent(ctx, id, uid, ps)
	if err != nil {
		err = fmt.Errorf("UpsertPracticeStudent called failed:%v", err)
		z.Error(err.Error())
		return err
	}
	return nil
}

// UpsertPracticeStudent 更新一次练习参与的学生名单
func UpsertPracticeStudent(ctx context.Context, practiceId, uid int64, students []cmn.TPracticeStudent) error {
	if len(students) == 0 {
		return nil
	}
	if practiceId <= 0 {
		err := fmt.Errorf("invalid practiceId  param")
		z.Error(err.Error())
		return err
	}
	if uid <= 0 {
		err := fmt.Errorf("invalid uid param")
		z.Error(err.Error())
		return err
	}
	//添加学生
	addPStr := strings.Repeat("(?,?,?,?,?,?,?),", len(students)-1) + "(?,?,?,?,?,?,?)"
	addPArgs := make([]interface{}, 0, len(students)*7+1)

	// 软删除学生
	var delPArgs []interface{}
	var valueExpr []string

	sqlxDB := cmn.GetDbConn()
	tx, err := sqlxDB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		err = fmt.Errorf("beginTx called failed:%v", err)
		z.Error(err.Error())
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			// 发生 panic 回滚
			err = tx.Rollback()
			panic(p)
		} else if err != nil {
			// 操作失败回滚
			err = tx.Rollback()
		} else {
			// 无错误则提交
			err = tx.Commit()
		}
	}()

	t := `
	INSERT INTO assessuser.t_practice_student 
    	(student_id, practice_id, creator, create_time, updated_by, update_time, status) 
	VALUES %s
	ON CONFLICT (student_id, practice_id)
	DO UPDATE SET
    	status = EXCLUDED.stat	us,
    	updated_by = EXCLUDED.updated_by,
    	update_time = EXCLUDED.update_time
	WHERE assessuser.t_practice_student.status IS DISTINCT FROM ? 
	`
	s1 := fmt.Sprintf(t, addPStr)

	for _, stu := range students {
		addPArgs = append(addPArgs,
			stu.StudentID, stu.PracticeID, stu.Creator, stu.CreateTime,
			stu.UpdatedBy, stu.UpdateTime, stu.Status,
		)
	}
	addPArgs = append(addPArgs, PracticeStudentStatus.Normal)

	// 使用sqlx.In 构建批量操作的SQL语句
	addPQuery, args, _ := sqlx.In(s1, addPArgs...)
	addPQuery = sqlx.Rebind(sqlx.DOLLAR, addPQuery)

	z.Sugar().Debugf("打印输出一下增加SQL语句:%v", addPQuery)
	z.Sugar().Debugf("打印输出一下增加SQL参数:%v", args)
	_, err = tx.ExecContext(ctx, addPQuery, args...)
	if err != nil {
		z.Error(err.Error())
		return err
	}

	// 然后继续进行删除不在名单上学生的操作
	t2 := `
	UPDATE assessuser.t_practice_student t
	SET status = $1 , update_time = $2 , updated_by = $3
	WHERE t.practice_id = $4
		AND NOT EXIST (
			SELECT 1 FROM (VALUES %s) AS excluded(sid)
			WHERE t.student_id = excluded.sid
	)	
	`
	delPArgs = append(delPArgs, PracticeStudentStatus.Normal, Timestamp(time.Now().UTC()), uid, practiceId)
	for _, s := range students {
		valueExpr = append(valueExpr, fmt.Sprintf("($%d::bigint)", len(delPArgs)+1))
		delPArgs = append(delPArgs, s.StudentID.Int64)
	}

	s2 := fmt.Sprintf(t2, strings.Join(valueExpr, ", "))

	z.Sugar().Debugf("打印输出一下删除SQL语句:%v", s2)
	z.Sugar().Debugf("打印输出一下删除SQL参数:%v", delPArgs)
	_, err = tx.ExecContext(ctx, s2, delPArgs)
	if err != nil {
		z.Error(err.Error())
		return err
	}
	return nil
}

// LoadPracticeById 获取练习详情
func LoadPracticeById(ctx context.Context, practiceId int64) (p cmn.TPractice, err error) {
	if practiceId <= 0 {
		err = fmt.Errorf("invalid practice ID param")
		z.Error(err.Error())
		return cmn.TPractice{}, err
	}
	s := `
	select (id,name,correct_mode,type,creator,create_time,updated_by,update_time,addi,status,allow_attempts,paper_id,exam_paper_id) from t_practice 
	where id = $1 AND status != $2
	limit 1`

	sqlxDB := cmn.GetDbConn()
	var stmt *sqlx.Stmt
	stmt, err = sqlxDB.Preparex(s)
	if err != nil {
		z.Error(err.Error())
		return cmn.TPractice{}, err
	}

	defer func() {
		err = stmt.Close()
		if err != nil {
			z.Error(err.Error())
			return
		}
	}()

	row := stmt.QueryRowxContext(ctx, practiceId, PracticeStatus.Deleted)
	err = row.StructScan(&p)
	if errors.Is(err, sql.ErrNoRows) {
		err = fmt.Errorf("无该练习记录:%v", err)
		z.Error(err.Error())
		return cmn.TPractice{}, err
	} else if err != nil {
		err = fmt.Errorf("LoadPracticeById call failed：%v", err)
		z.Error(err.Error())
		return cmn.TPractice{}, err
	}
	return p, nil
}

// ListPractice 获取练习列表
func ListPractice(ctx context.Context, name, pType, status string, orderBy []string, page, pageSize int, uid int64) ([]Map, int, error) {
	result := make([]Map, 0)
	// 查询条件
	var clauses []string
	// 占位符
	var args []interface{}
	args = append(args, PracticeStudentStatus.Normal)
	// 占位符数值
	if name != "" {
		clauses = append(clauses, fmt.Sprintf("%s LIKE $%d", "name", len(args)+1))
		args = append(args, "%"+name+"%")
	}
	if status != "" {
		clauses = append(clauses, fmt.Sprintf("%s = $%d", "status", len(args)+1))
		args = append(args, status)
	}
	if pType != "" {
		clauses = append(clauses, fmt.Sprintf("%s = $%d", "type", len(args)+1))
		args = append(args, pType)
	}
	clauses = append(clauses, fmt.Sprintf("status != $%d", len(args)+1))
	args = append(args, PracticeStatus.Deleted)
	clauses = append(clauses, fmt.Sprintf("creator = $%d", len(args)+1))
	args = append(args, uid)
	s := `
 	SELECT 
		tp.id, tp.name,tp.correct_mode,
		tp.type, tp.creator, tp.create_time, tp.updated_by, tp.update_time, tp.addi, tp.status ,tp.allowed_attempts,
		COALESCE((SELECT COUNT(*) FROM t_practice_student tps WHERE tps.practice_id=tp.id AND status=$1),0) as student_cnt
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
	if pageSize > 0 && pageSize <= 100 {
		s += fmt.Sprintf(" LIMIT $%d", len(args)+1)
		args = append(args, pageSize)
	}

	if page > 0 {
		offset := (page - 1) * pageSize
		s += fmt.Sprintf(" OFFSET $%d", len(args)+1)
		args = append(args, offset)
	}

	z.Sugar().Debugf("打印输出一下这个操作语句：%v", s)
	z.Sugar().Debugf("打印输出一下参数表：%v", args)

	// 这些实体查询的每个函数之间作用都不一样，需要花时间去了解这个函数的具体用处了
	sqlxDB := cmn.GetDbConn()
	rows, err := sqlxDB.QueryxContext(ctx, s, args...)
	if err != nil {
		z.Error(err.Error())
		return nil, 0, err
	}
	defer func() {
		err = rows.Close()
		if err != nil {
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
		if err != nil {
			err = fmt.Errorf("解析练习数据失败:%v", err)
			z.Error(err.Error())
			return nil, 0, err
		}
		M["practice"] = p
		M["student"] = studentCount
		result = append(result, M)
	}
	return result, len(result), nil
}

// ListPracticeStudentIds 获取参与某次练习的所有考生Id
func ListPracticeStudentIds(ctx context.Context, practiceId int64) ([]int64, error) {
	if practiceId <= 0 {
		err := fmt.Errorf("invalid practice ID param")
		z.Error(err.Error())
		return nil, err
	}
	ids := make([]int64, 0)
	s := `SELECT student_id FROM assessuser.t_practice_student WHERE practice_id = $1 AND status = $2`
	sqlxDB := cmn.GetDbConn()
	rows, err := sqlxDB.QueryxContext(ctx, s, practiceId, PracticeStudentStatus.Normal)
	if err != nil {
		err = fmt.Errorf("ListPracticeStudentIds call failed:%v", err)
		z.Error(err.Error())
		return nil, err
	}
	defer func() {
		err = rows.Close()
		if err != nil {
			z.Error(err.Error())
			return
		}
	}()
	for rows.Next() {
		var studentId int64
		err = rows.Scan(&studentId)
		if err != nil {
			err = fmt.Errorf("scan student id failed:%v", err)
			z.Error(err.Error())
			return nil, err
		}
		ids = append(ids, studentId)
	}

	return ids, nil
}

// OperatePracticeStatus 操作练习的发布状态
func OperatePracticeStatus(ctx context.Context, practiceId int64, status string, uid int64) error {

	if practiceId <= 0 {
		err := fmt.Errorf("invalid practice ID param")
		z.Error(err.Error())
		return err
	}
	sqlxDB := cmn.GetDbConn()
	p, err := LoadPracticeById(ctx, practiceId)
	if err != nil {
		return err
	}
	tx, err := sqlxDB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		err = fmt.Errorf("beginTx called failed:%v", err)
		z.Error(err.Error())
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			// 发生 panic 回滚
			err = tx.Rollback()
			panic(p)
		} else if err != nil {
			// 操作失败回滚
			err = tx.Rollback()
		} else {
			// 无错误则提交
			err = tx.Commit()
		}
	}()
	switch status {
	case PracticeStatus.Released:
		{
			if p.Status.String != PracticeStatus.PendingRelease {
				err = fmt.Errorf("获取练习状态出现数据错误:原练习状态不为待发布状态")
				z.Error(err.Error())
				return err
			}
			if !p.PaperID.Valid || p.PaperID.Int64 <= 0 {
				err = fmt.Errorf("获取练习试卷信息出现数据错误：绑定的练习试卷为空或非法")
				z.Error(err.Error())
				return err
			}
			examPaperId, _, err := examPaper.GenerateExamPaper(ctx, tx, examPaper.PaperCategory.Practice, p.PaperID.Int64, practiceId, 0, uid, false)
			if err != nil {
				return err
			}
			if examPaperId == nil {
				err = fmt.Errorf("生成练习考卷返回的考卷ID为空")
				z.Error(err.Error())
				return err
			}
			// 更新练习状态信息
			p.ExamPaperID = null.IntFrom(*examPaperId)
			p.Status = null.StringFrom(PracticeStatus.Released)
			p.UpdatedBy = null.IntFrom(uid)
			p.UpdateTime = null.IntFrom(Timestamp(time.Now().UTC()))

			err = UpdatePractice(ctx, &p, nil, uid)
			if err != nil {
				return err
			}
			return nil
		}

	case PracticeStatus.PendingRelease:
		{
			if p.Status.String != PracticeStatus.Released {
				err = fmt.Errorf("获取练习状态出现数据错误")
				z.Error(err.Error())
				return err
			}
			s := `UPDATE assessuser.t_practice SET status = $1,update_time = $2, updated_by = $3  WHERE id = $4`
			_, err = tx.ExecContext(ctx, s, PracticeStatus.PendingRelease, Timestamp(time.Now().UTC()), uid, practiceId)
			if err != nil {
				err = fmt.Errorf("OperatePracticeStatus to pendingRelease failed:%v", err)
				z.Error(err.Error())
				return err
			}
			return nil
		}
	default:
		err = fmt.Errorf("please call OperatePracticeStatus with valid param:status ")
		z.Error(err.Error())
		return err
	}
}
