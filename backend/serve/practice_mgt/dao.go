package practice_mgt

//annotation:practice_mgt-service
//author:{"name":"ZouDeLun","tel":"15920422045", "email":"1311866870@qq.com"}
import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jmoiron/sqlx"
	"strings"
	"time"
	"w2w.io/cmn"
	"w2w.io/null"
	"w2w.io/serve/examPaper"
	"w2w.io/serve/mark"
)

// UpsertPractice 新增/修改练习信息 根据用户传输的信息动态构建SQL语句
func UpsertPractice(ctx context.Context, p *cmn.TPractice, ps []int64, uid int64) error {
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
	return UpdatePractice(ctx, p, ps, uid, false)

}

// UpdatePractice 更新练习本身信息
func UpdatePractice(ctx context.Context, p *cmn.TPractice, ps []int64, uid int64, isOperate bool) error {
	if uid <= 0 {
		err := fmt.Errorf("invalid updator ID param")
		z.Error(err.Error())
		return err
	}
	now := time.Now().UnixMilli()
	p.UpdatedBy = null.IntFrom(uid)
	p.UpdateTime = null.IntFrom(now)
	update, _ := S2Map(p)
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
	if err != nil {
		err = fmt.Errorf("updatePractice call failed:%v", err)
		z.Error(err.Error())
		return err
	}
	err = UpsertPracticeStudent(ctx, p.ID.Int64, uid, ps)
	if err != nil {
		return err
	}
	return nil

}

// AddPractice 添加一场练习 包括插入成功导入的学生
// TODO 需对接学生管理接口
func AddPractice(ctx context.Context, p *cmn.TPractice, ps []int64, uid int64) error {
	var id int64
	now := time.Now().UnixMilli()
	sqlxDB := cmn.GetDbConn()
	s := `
	INSERT INTO assessuser.t_practice (name,correct_mode,creator,create_time, update_time, addi,allowed_attempts,type,paper_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`
	err := sqlxDB.QueryRowxContext(ctx, s, p.Name, p.CorrectMode, uid, now, now, p.Addi, p.AllowedAttempts, p.Type, p.PaperID).Scan(&id)
	if err != nil {
		err = fmt.Errorf("addPractice call failed:%v", err)
		z.Error(err.Error())
		return err
	}
	p.ID = null.IntFrom(id)
	err = UpsertPracticeStudent(ctx, id, uid, ps)
	if err != nil {
		return err
	}
	return nil
}

// UpsertPracticeStudent 更新一次练习参与的学生名单
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
	// 这里添加这个rollback的错误
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
		if err != nil || forceErr == "Rollback" {
			// 操作失败回滚
			err = tx.Rollback()
			if err != nil {
				z.Error(err.Error())
			}
		} else {
			// 无错误则提交
			err = tx.Commit()
			if err != nil {
				z.Error(err.Error())
			}
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
	if err != nil {
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
	if err != nil {
		z.Error(err.Error())
		return err
	}
	return nil
}

// LoadPracticeById 获取练习详情 其中不需要查询学生具体信息
func LoadPracticeById(ctx context.Context, practiceId int64) (*cmn.TPractice, string, int, error) {
	if practiceId <= 0 {
		err := fmt.Errorf("非法practiceID:%v", practiceId)
		z.Error(err.Error())
		return &cmn.TPractice{}, "", 0, err
	}
	s := `
	select p.id, p.name, p.correct_mode,p.addi,p.status,p.type,
			COALESCE(tp.name, '') as paper_name,p.allowed_attempts,p.paper_id,p.exam_paper_id,
			COALESCE((SELECT COUNT(*) FROM assessuser.t_practice_student tps WHERE tps.practice_id=tp.id AND status=$1),0) as student_cnt
	from assessuser.t_practice p
	left join assessuser.t_paper tp on tp.id = p.paper_id AND tp.status = $2
	where p.id = $3 AND p.status != $4
	limit 1`
	sqlxDB := cmn.GetDbConn()
	var stmt *sqlx.Stmt
	stmt, err := sqlxDB.Preparex(s)
	if err != nil {
		z.Error(err.Error())
		return &cmn.TPractice{}, "", 0, err
	}

	defer func() {
		err = stmt.Close()
		if err != nil {
			z.Error(err.Error())
			return
		}
	}()
	var p cmn.TPractice
	var paperName string
	var studentCount int
	err = stmt.QueryRowxContext(ctx, PracticeStudentStatus.Normal, examPaper.PaperStatus.Normal, practiceId, PracticeStatus.Deleted).
		Scan(&p.ID, &p.Name, &p.CorrectMode,
			&p.Addi, &p.Status, &p.Type, &paperName, &p.AllowedAttempts, &p.PaperID, &p.ExamPaperID, &studentCount)
	if errors.Is(err, sql.ErrNoRows) {
		err = fmt.Errorf("非法practiceID ， 无该练习记录:%v", err)
		z.Error(err.Error())
		return &cmn.TPractice{}, "", 0, err
	} else if err != nil {
		err = fmt.Errorf("LoadPracticeById call failed：%v", err)
		z.Error(err.Error())
		return &cmn.TPractice{}, "", 0, err
	} else {
		return &p, paperName, studentCount, nil
	}
}

// ListPracticeS 学生权限及以下获取练习列表
// TODO 添加上权限设计 可能会整合成一个接口

func ListPracticeS(ctx context.Context, pType, name, difficulty string, orderBy []string, page, pageSize int, uid int64) ([]*cmn.TVPracticeSummary, int, error) {
	result := make([]*cmn.TVPracticeSummary, 0)
	// 查询条件
	var clauses []string
	// 占位符
	var args []interface{}
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
		total_score,highest_score,paper_total_score,paper_id,latest_unsubmitted_id,latest_submitted_id
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
	// 遍历行数据
	for rows.Next() {
		var p cmn.TVPracticeSummary
		err = rows.Scan(&p.ID, &p.Name, &p.Type, &p.AttemptCount, p.Difficulty, &p.AllowedAttempts,
			&p.QuestionCount, &p.WrongCount, &p.TotalScore, &p.HighestScore, &p.PaperTotalScore,
			&p.PaperID, &p.LatestUnsubmittedID, &p.LatestSubmittedID)
		if err != nil {
			err = fmt.Errorf("解析练习数据失败:%v", err)
			z.Error(err.Error())
			return nil, 0, err
		}
		result = append(result, &p)
	}
	return result, len(result), nil
}

// ListPracticeT 教师权限及以上获取练习列表
func ListPracticeT(ctx context.Context, name, pType, status string, orderBy []string, page, pageSize int, uid int64) ([]Map, int, error) {
	result := make([]Map, 0)
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
		M["student_count"] = studentCount
		result = append(result, M)
	}
	return result, len(result), nil
}

// ListPracticeStudentIds 获取参与某次练习的所有考生Id
func ListPracticeStudentIds(ctx context.Context, pid int64) ([]int64, error) {
	if pid <= 0 {
		err := fmt.Errorf("invalid practice ID param")
		z.Error(err.Error())
		return nil, err
	}
	ids := make([]int64, 0)
	s := `SELECT student_id FROM assessuser.t_practice_student WHERE practice_id = $1 AND status = $2`
	sqlxDB := cmn.GetDbConn()
	rows, err := sqlxDB.QueryxContext(ctx, s, pid, PracticeStudentStatus.Normal)
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
		if err != nil || forceErr == "Rollback" {
			// 操作失败回滚
			err = tx.Rollback(ctx)
			if err != nil {
				z.Error(err.Error())
			}
		} else {
			// 无错误则提交
			err = tx.Commit(ctx)
			if err != nil {
				z.Error(err.Error())
			}
		}
	}()
	if status == PracticeStatus.Released {
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
		//只有当第一次创建发布练习时，才会创建新的考卷 但是这样有个问题，那就是可能会出现老师选择发布了之后，但是又取消，
		//重新编辑了一张新的试卷 ，再继续发布，但实际上是不会影响你exam_paper_id的值的 只是会影响你paperID的值，因此不能通过这个是否存在来判断他是否应该生成考卷
		examPaperId, _, err := examPaper.GenerateExamPaper(ctx, tx, examPaper.PaperCategory.Practice, p.PaperID.Int64, pid, 0, uid, false)
		if err != nil {
			return err
		}
		if examPaperId == nil {
			err = fmt.Errorf("生成练习考卷返回的考卷ID为空")
			z.Error(err.Error())
			return err
		}
		p.ExamPaperID = null.IntFrom(*examPaperId)
		// 更新练习状态信息
		p.Status = null.StringFrom(PracticeStatus.Released)
		p.UpdatedBy = null.IntFrom(uid)
		p.UpdateTime = null.IntFrom(now)

		err = UpdatePractice(ctx, p, nil, uid, true)
		if err != nil {
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
		if err != nil {
			return err
		}
		return nil
	} else if status == PracticeStatus.PendingRelease || status == PracticeStatus.Deleted {
		if p.Status.String != PracticeStatus.Released {
			err = fmt.Errorf("获取练习状态出现数据错误")
			z.Error(err.Error())
			return err
		}
		s := `UPDATE assessuser.t_practice SET status = $1,update_time = $2, updated_by = $3  WHERE id = $4`
		_, err = tx.Exec(ctx, s, PracticeStatus.PendingRelease, now, uid, pid)
		if err != nil {
			err = fmt.Errorf("OperatePracticeStatus to pendingRelease failed:%v", err)
			z.Error(err.Error())
			return err
		}

		// 清除批改配置信息
		req := mark.HandleMarkerInfoReq{
			Status:      "02",
			PracticeIDs: []int64{p.ID.Int64},
		}

		err = mark.HandleMarkerInfo(ctx, &tx, uid, req)
		if err != nil {
			return err
		}
		return nil
	} else {
		err = fmt.Errorf("please call OperatePracticeStatus with valid param:status ")
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
				var p *cmn.TVExamPaper
				var epInfo *EnterPracticeInfo
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
				var q1 *examPaper.ExamQuestion
				q1.ID = null.IntFrom(2042)
				q1.Type = null.StringFrom("00")
				q1.Title = null.StringFrom("")
				q1.Content = null.StringFrom("<p><span style=\"font-family: 等线; font-size: 12pt\">具有风险分析的软件生命周期模型是</span><span style=\"font-family: Aptos, sans-serif; font-size: 12pt\">()</span></p>")
				q1.Options = JSONText(`[
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
                ]`)
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
				var q2 *examPaper.ExamQuestion
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
				var p *cmn.TVExamPaper
				var epInfo *EnterPracticeInfo
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
				var q1 *examPaper.ExamQuestion
				q1.ID = null.IntFrom(2042)
				q1.Type = null.StringFrom("00")
				q1.Title = null.StringFrom("")
				q1.Content = null.StringFrom("<p><span style=\"font-family: 等线; font-size: 12pt\">具有风险分析的软件生命周期模型是</span><span style=\"font-family: Aptos, sans-serif; font-size: 12pt\">()</span></p>")
				q1.Options = JSONText(`[
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
                ]`)
				q1.Score = null.FloatFrom(3)
				q1.Order = null.IntFrom(1)
				qList1 = append(qList1, q1)

				qList2 := make([]*examPaper.ExamQuestion, 0)
				var q2 *examPaper.ExamQuestion
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
		default:
			{
				err := fmt.Errorf("不成功都是失败")
				return nil, nil, nil, err
			}
		}
	}
	var ps cmn.TVPracticeSummary
	sqlxDB := cmn.GetDbConn()
	now := time.Now().UnixMilli()
	// 判断学生作答练习的情况
	var submissionStatus string

	// 这里先根据练习ID跟userId去获取一下这个练习状态 去查这个last_unSubmitted_id然后能根据这个ID去拿出这个 这里有可能是学生根据没有一次提交
	// 也就是说此时是第一次进入，那就需要创建新的submissions的，同理如果查询出有记录，但是last为空的话，仍然需要创建，否则就不需要创建
	s := `SELECT allowed_attempts,attempt_count,latest_unsubmitted_id, latest_submitted_id, exam_paper_id,paper_name,suggested_duration
	 FROM assessuser.v_practice_summary 
	 WHERE id = $1 AND student_id = $2 AND practice_status != $3 
	 AND practice_student_status != $4`
	err := sqlxDB.QueryRowxContext(ctx, s, pid, uid, PracticeStatus.Deleted, PracticeStudentStatus.Deleted).Scan(&ps.AllowedAttempts, &ps.AttemptCount, &ps.LatestUnsubmittedID,
		&ps.LatestSubmittedID, &ps.ExamPaperID,
		&ps.PaperName, &ps.SuggestedDuration)
	if err != nil {
		err = fmt.Errorf("select student practice submission failed:%v", err)
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
			if ps.AllowedAttempts.Int64 != 0 && ps.AllowedAttempts.Int64 == ps.AttemptCount.Int64 {
				// 学生进入练习次数已经满了，无法再继续获取
				err = fmt.Errorf("已达练习最大次数:%v，无法再次进入练习", ps.AttemptCount.Int64)
				z.Error(err.Error())
				return nil, nil, nil, err
			}
			if !ps.ExamPaperID.Valid || ps.ExamPaperID.Int64 <= 0 {
				err = fmt.Errorf("练习所属考卷ID丢失，请检查练习视图或操作发布练习逻辑")
				z.Error(err.Error())
				return nil, nil, nil, err
			}
			newAttempt := ps.AttemptCount.Int64 + 1
			s := `INSERT INTO assessuser.t_practice_submissions (practice_id,student_id,exam_paper_id,creator,create_time,update_time,attempt) VALUES (
			$1,$2,$3,$4,$5,$6,$7	
		) RETURNING id`
			err = tx.QueryRow(ctx, s, pid, uid, ps.ExamPaperID, uid, now, now, newAttempt).Scan(&pSubmissionID)
			if err != nil {
				err = fmt.Errorf("insert practice submission failed:%v", err)
				z.Error(err.Error())
				return nil, nil, nil, err
			}
			r := examPaper.GenerateAnswerQuestionsRequest{
				ExamPaperID:          ps.ExamPaperID.Int64,
				Category:             examPaper.PaperCategory.Practice,
				PracticeSubmissionID: []int64{pSubmissionID},
				IsOptionRandom:       false,
				IsQuestionRandom:     false,
				Attempt:              newAttempt,
			}
			// 生成学生答卷
			err = examPaper.GenerateAnswerQuestion(ctx, tx, r, uid)
			if err != nil {
				return nil, nil, nil, err
			}
			withStudentAnswer = false
		}
		//第一次进入练习 没有任何练习提交记录
	case StudentSubmissionStatus.NeverAnswer:
		{
			//在创建记录之前，需要先加载一下练习的基本信息
			p, _, _, err := LoadPracticeById(ctx, pid)
			if err != nil {
				return nil, nil, nil, err
			}
			s := `INSERT INTO assessuser.t_practice_submissions (practice_id,student_id,exam_paper_id,creator,create_time,update_time,attempt) VALUES (
					$1,$2,$3,$4,$5,$6,$7	
				  ) RETURNING id`
			err = tx.QueryRow(ctx, s, pid, uid, p.ExamPaperID, uid, now, now, 1).Scan(&pSubmissionID)
			if err != nil {
				err = fmt.Errorf("insert practice submission failed:%v", err)
				z.Error(err.Error())
				return nil, nil, nil, err
			}
			r := examPaper.GenerateAnswerQuestionsRequest{
				ExamPaperID:          p.ExamPaperID.Int64,
				Category:             examPaper.PaperCategory.Practice,
				PracticeSubmissionID: []int64{pSubmissionID},
				IsOptionRandom:       false,
				IsQuestionRandom:     false,
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
	default:
		{
			err = fmt.Errorf("invalid practice submissions status")
			z.Error(err.Error())
			return nil, nil, nil, err
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

	return &epInfo, pg, pq, nil

}
