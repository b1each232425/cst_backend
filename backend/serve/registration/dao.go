package registration

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
	"w2w.io/serve/practice_mgt"
)

// 教师查看报名计划列表
func ListRegisterT(ctx context.Context, name string, course string, status string, orderBy []string, page int, pageSize int, userID int64) ([]Map, int, error) {
	result := make([]Map, 0)
	// 用于测试，强制执行某些错误分支
	forceErr, _ := ctx.Value("force-error").(string)
	//构建查询条件
	var clauses []string
	//构建占位符
	var args []interface{}
	args = append(args, RegisterStudentStatus.Apply)
	if name != "" {
		clauses = append(clauses, fmt.Sprintf("%s LIKE $%d", "r.name", len(args)+1))
		args = append(args, "%"+name+"%")
	}
	if course != "" {
		clauses = append(clauses, fmt.Sprintf("%s  =$%d", "r.course", len(args)+1))
		args = append(args, "%"+course+"%")
	}
	if status != "" {
		clauses = append(clauses, fmt.Sprintf("%s  =$%d", "r.status", len(args)+1))
		args = append(args, status)
	}
	clauses = append(clauses, fmt.Sprintf("r.status != $%d", len(args)+1))
	args = append(args, RegisterStatus.Deleted)
	clauses = append(clauses, fmt.Sprintf("r.creator = $%d", len(args)+1))
	args = append(args, userID)
	s := `
	SELECT r.id, r.name , r.course , COALESCE((SELECT COUNT(*) FROM assessuser.t_exam_plan_student eps WHERE eps.register_id=r.id AND eps.status!=$1),0) , r.max_number , r.review_end_time , r.start_time , r.end_time , COALESCE(STRING_AGG(p.name, '、'),'') , r.status ,r.exam_plan_location
	FROM assessuser.t_register_plan r  LEFT JOIN assessuser.t_register_practice rp ON rp.register_id=r.id 
	LEFT JOIN  assessuser.t_practice p ON p.id=rp.practice_id
		`
	if len(clauses) > 0 {
		s += "WHERE " + strings.Join(clauses, " AND ")
	}
	s += " GROUP BY r.id  , r.name, r.course, r.max_number, r.review_end_time, r.start_time, r.end_time, r.status , r.exam_plan_location"
	//添加orderBy语句
	if len(orderBy) > 0 {
		s += " ORDER BY " + strings.Join(orderBy, ", ")
	}
	//添加分页信息
	offSet := (page - 1) * pageSize
	s += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, pageSize, offSet)

	z.Sugar().Debugf("打印输出一下这个操作语句：%v", s)
	z.Sugar().Debugf("打印输出一下参数表：%v", args)
	// 这些实体查询的每个函数之间作用都不一样，需要花时间去了解这个函数的具体用处了
	sqlxDB := cmn.GetDbConn()
	rows, err := sqlxDB.QueryxContext(ctx, s, args...)
	if err != nil || forceErr == "query" {
		err = fmt.Errorf("search register failed:%v", err)
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
	for rows.Next() {
		M := Map{}
		var r cmn.TRegisterPlan
		var practiceName string
		var studentCount int64
		err = rows.Scan(&r.ID, &r.Name, &r.Course, &studentCount, &r.MaxNumber, &r.ReviewEndTime, &r.StartTime, &r.EndTime, &practiceName, &r.Status, &r.ExamPlanLocation)
		if err != nil || forceErr == "scan" {
			err = fmt.Errorf("解析数据失败:%v", err)
			z.Error(err.Error())
			return nil, 0, err
		}
		M["register"] = r
		M["studentCount"] = studentCount
		M["practiceName"] = practiceName
		result = append(result, M)
	}
	return result, len(result), nil
}

// 学生查看报名计划
func ListRegisterS(ctx context.Context, name string, course string, status string, orderBy []string, page int, pageSize int, userID int64) ([]Map, int, error) {
	result := make([]Map, 0)
	// 用于测试，强制执行某些错误分支
	forceErr, _ := ctx.Value("force-error").(string)
	//构建查询条件
	var clauses []string
	//构建占位符
	var args []interface{}
	args = append(args, userID)
	if name != "" {
		clauses = append(clauses, fmt.Sprintf("%s LIKE $%d", "r.name", len(args)+1))
		args = append(args, "%"+name+"%")
	}
	if course != "" {
		clauses = append(clauses, fmt.Sprintf("%s  =$%d", "r.course", len(args)+1))
		args = append(args, "%"+course+"%")
	}
	if status != "" {
		clauses = append(clauses, fmt.Sprintf("%s  =$%d", "eps.status", len(args)+1))
		args = append(args, status)
	}
	clauses = append(clauses, fmt.Sprintf("r.status = $%d ", len(args)+1))
	args = append(args, RegisterStatus.Released)
	clauses = append(clauses, fmt.Sprintf("r.start_time < $%d", len(args)+1))
	args = append(args, time.Now().UnixMilli())
	s := `
	SELECT r.id, r.name ,r.review_end_time, r.start_time , r.end_time , r.course , r.exam_plan_location,eps.exam_type , eps.status
	FROM assessuser.t_register_plan r LEFT JOIN  assessuser.t_exam_plan_student eps ON eps.register_id = r.id AND eps.student_id = $1 
		`
	if len(clauses) > 0 {
		s += "WHERE " + strings.Join(clauses, " AND ")
	}
	//添加orderBy语句
	if len(orderBy) > 0 {
		s += " ORDER BY " + strings.Join(orderBy, ", ")
	}
	//添加分页信息
	offSet := (page - 1) * pageSize
	s += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, pageSize, offSet)

	z.Sugar().Debugf("打印输出一下这个操作语句：%v", s)
	z.Sugar().Debugf("打印输出一下参数表：%v", args)
	// 这些实体查询的每个函数之间作用都不一样，需要花时间去了解这个函数的具体用处了
	sqlxDB := cmn.GetDbConn()
	rows, err := sqlxDB.QueryxContext(ctx, s, args...)
	if err != nil || forceErr == "query" {
		err = fmt.Errorf("search register failed:%v", err)
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
	for rows.Next() {
		M := Map{}
		var r cmn.TRegisterPlan
		var student cmn.TExamPlanStudent
		err = rows.Scan(&r.ID, &r.Name, &r.ReviewEndTime, &r.StartTime, &r.EndTime, &r.Course, &r.ExamPlanLocation, &student.ExamType, &student.Status)
		if err != nil || forceErr == "scan" {
			err = fmt.Errorf("解析数据失败:%v", err)
			z.Error(err.Error())
			return nil, 0, err
		}
		M["register"] = r
		M["student"] = student
		result = append(result, M)
	}
	return result, len(result), nil
}

// 学生进行报名
func StudentRegister(ctx context.Context, registerID int64, status string, RegisterType string, students []registerStudentType, userID int64) error {
	var err error
	forceErr, _ := ctx.Value("force-error").(string)

	conn := cmn.GetPgxConn()
	now := time.Now().UnixMilli()
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
	//判断当前报名计划是否处于发布状态
	var registerStatus string
	s := `SELECT status FROM assessuser.t_register_plan WHERE id = $1`
	err = tx.QueryRow(ctx, s, registerID).Scan(&registerStatus)
	if err != nil || forceErr == "query" {
		err = fmt.Errorf("query register failed:%v", err)
		z.Error(err.Error())
		return err
	}
	if registerStatus != RegisterStatus.Released {
		err = fmt.Errorf("报名计划状态不处于已发布，无法报名 %v ", status)
		z.Error(err.Error())
		return err
	}
	//获取当前学生的报名状态
	s = `
	SELECT status FROM assessuser.t_exam_plan_student WHERE register_id = $1 AND student_id = $2
`
	var studentStatus string
	err = tx.QueryRow(ctx, s, registerID, userID).Scan(&studentStatus)
	var noExist bool
	noExist = errors.Is(err, sql.ErrNoRows)
	if (err != nil && !noExist) || forceErr == "query" {
		err = fmt.Errorf("query student failed:%v", err)
		z.Error(err.Error())
		return err
	}
	if (studentStatus == RegisterStudentStatus.Apply && status == RegisterStudentStatus.Pending) || (status == RegisterStudentStatus.Pending && noExist) {
		//检验当前的人数是否超过限制
		registerInfo, _, _, current, err := LoadRegisterById(ctx, registerID)
		if err != nil {
			z.Error(err.Error())
			return err
		}
		if registerInfo.MaxNumber.Int64 < current+1 {
			err = fmt.Errorf("报名计划人数已满")
			z.Error(err.Error())
			return err
		}
	}
	var registerTime int64
	if status == RegisterStudentStatus.Pending && noExist {
		registerTime = now
	}
	//若报名计划已经报名
	s = `
	INSERT INTO assessuser.t_exam_plan_student (student_id , register_id , type  , exam_type , register_time  , creator , updated_by , create_time , update_time , status )
	VALUES ($1 , $2 , $3 , $4 , $5 , $6 , $7 , $8 ,$9 , $10 )
	ON CONFLICT (student_id,register_id) 
	DO UPDATE  SET
	    status = EXCLUDED.status,
            updated_by = EXCLUDED.updated_by,
            update_time = EXCLUDED.update_time
	   WHERE  assessuser.t_exam_plan_student.status IS DISTINCT FROM $11
`
	z.Sugar().Debugf("打印输出一下这个操作语句：%v", s)
	for _, student := range students {
		_, err = tx.Exec(ctx, s, student.StudentID, registerID, RegisterType, student.ExamType, registerTime, userID, userID, now, now, status, RegisterStudentStatus.Approved)
		if err != nil || forceErr == "exec" {
			err = fmt.Errorf("exec failed:%v", err)
			z.Error(err.Error())
			return err
		}
	}
	//把学生关联到练习计划
	if studentStatus == RegisterStudentStatus.Apply && status == RegisterStudentStatus.Pending {
		err = practice_mgt.BoundPracticeEnterRegisterPlan(ctx, tx, userID, registerID)
		if err != nil {
			z.Error(err.Error())
			return err
		}
	}
	return nil
}

// 根据报名计划查询学生列表
func GetRegisterStudentById(cxt context.Context, registerID int64, message string, registerType string, status string, orderBy []string, page int, pageSize int, userID int64) ([]Map, int, error) {
	result := make([]Map, 0)
	forceErr, _ := cxt.Value("force-error").(string)

	//构建查询条件
	var clauses []string
	//构建占位符
	var args []interface{}

	if message != "" {
		clauses = append(clauses, fmt.Sprintf("%s LIKE $%d OR %s LIKE $%d OR %s LIKE $%d OR %s LIKE $%d", "u.name", len(args)+1,
			"u.email", len(args)+2, "u.id_card_no", len(args)+3, "u.mobile_phone", len(args)+4))
		args = append(args, "%"+message+"%")
	}
	if registerType != "" {
		clauses = append(clauses, fmt.Sprintf("%s = $%d", "eps.type", len(args)+1))
		args = append(args, registerType)
	}
	if status != "" {
		clauses = append(clauses, fmt.Sprintf("%s = $%d", "eps.status", len(args)+1))
		args = append(args, status)
	}
	clauses = append(clauses, fmt.Sprintf("eps.register_id = $%d", len(args)+1))
	args = append(args, registerID)
	clauses = append(clauses, fmt.Sprintf("eps.status != $%d", len(args)+1))
	args = append(args, RegisterStudentStatus.Apply)
	s := `
	SELECT u.id ,  u.official_name , u.mobile_phone , u.email , u.gender , u.id_card_no , u.id_card_type , eps.register_time , eps.type , eps.exam_type , COALESCE((SELECT official_name FROM assessuser.t_user WHERE id =eps.reviewer),'') AS reviewer , eps.status
	FROM assessuser.t_user u JOIN assessuser.t_exam_plan_student eps ON eps.student_id =u.id  
`
	if len(clauses) > 0 {
		s += "WHERE " + strings.Join(clauses, " AND ")
	}
	//添加orderBy语句
	if len(orderBy) > 0 {
		s += " ORDER BY " + strings.Join(orderBy, ", ")
	}
	//添加分页参数
	offset := (page - 1) * pageSize
	s += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, pageSize, offset)
	z.Sugar().Debugf("打印输出一下这个操作语句：%v", s)
	z.Sugar().Debugf("打印输出一下参数表：%v", args)
	sqlxDB := cmn.GetDbConn()
	rows, err := sqlxDB.QueryxContext(cxt, s, args...)
	if err != nil || forceErr == "query" {
		err = fmt.Errorf("query register failed:%v", err)
		z.Error(err.Error())
		return nil, 0, err
	}
	defer func() {
		err = rows.Close()
		if err != nil || forceErr == "row close" {
			z.Error(err.Error())
			return
		}
	}()
	for rows.Next() {
		M := Map{}
		var student cmn.TUser
		var planStudent cmn.TExamPlanStudent
		var reviewer string
		err = rows.Scan(&student.ID, &student.OfficialName, &student.MobilePhone, &student.Email, &student.Gender, &student.IDCardNo, &student.IDCardType, &planStudent.RegisterTime, &planStudent.Type, &planStudent.ExamType, &reviewer, &planStudent.Status)
		if err != nil || forceErr == "scan" {
			err = fmt.Errorf("解析数据失败:%v", err)
			z.Error(err.Error())
			return nil, 0, err
		}
		M["student"] = student
		M["detail"] = planStudent
		M["reviewer"] = reviewer
		result = append(result, M)
	}
	return result, len(result), nil
}
func UpsertRegister(ctx context.Context, registration *cmn.TRegisterPlan, practiceIds []int64, userID int64, action string) error {
	if userID <= 0 {
		err := fmt.Errorf("用户ID不能小于等于0")
		z.Error(err.Error())
		return err
	}
	if !registration.ID.Valid {
		return AddRegister(ctx, registration, practiceIds, userID)
	}
	//获取当前的报名计划详细内容
	register, _, _, _, err := LoadRegisterById(ctx, registration.ID.Int64)
	if err != nil {
		return err
	}
	if register.Status == null.NewString(RegisterStatus.Deleted, true) || register.Status == null.NewString(RegisterStatus.Disabled, true) || register.Status == null.NewString(RegisterStatus.Cancel, true) || register.Status == null.NewString(RegisterStatus.Ending, true) || register.Status == null.NewString(RegisterStatus.ReviewEnding, true) {
		err := fmt.Errorf("报名计划状态为%v，无法修改", register.Status)
		z.Error(err.Error())
		return err
	}
	return UpdateRegister(ctx, registration, practiceIds, userID, action)
}
func UpdateRegister(ctx context.Context, registration *cmn.TRegisterPlan, practiceIds []int64, userID int64, action string) error {

	if userID <= 0 {
		err := fmt.Errorf("用户ID不能小于等于0")
		z.Error(err.Error())
		return err
	}
	if !registration.ID.Valid {
		err := fmt.Errorf("报名计划ID不存在")
		if err != nil {
			z.Error(err.Error())
			return err
		}
	}
	forceErr, _ := ctx.Value("force-error").(string)
	now := time.Now().UnixMilli()
	registration.UpdatedBy = null.NewInt(userID, true)
	registration.UpdateTime = null.NewInt(now, true)
	action = strings.ToLower(action)
	update := s2Map(registration)
	notUpdate := []string{
		"id",
		"creator",
		"create_time",
		"status",
		"reviewer_ids",
	}
	RemoveFields(update, notUpdate...)
	var clauses []string
	var args []interface{}
	idx := 1
	for field, value := range update {
		clauses = append(clauses, fmt.Sprintf("%s = $%d", field, idx))
		args = append(args, value)
		idx++
	}
	args = append(args, registration.ID)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = $%d", "assessuser.t_register_plan", strings.Join(clauses, ", "), idx)
	z.Sugar().Debugf("update sql:%v", query)
	z.Sugar().Debugf("update args:%v", args)
	sqlxDB := cmn.GetPgxConn()
	tx, err := sqlxDB.Begin(ctx)
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
			err = tx.Rollback(ctx)
			if forceErr == "rollback" {
				err = fmt.Errorf("触发回滚")
			}
			if err != nil {
				z.Error(err.Error())
				return
			}
		}
		_ = tx.Commit(ctx)
		if forceErr == "commit" {
			err = fmt.Errorf("commit failed")
		}
		if err != nil {
			z.Error(err.Error())
		}
	}()

	_, err = tx.Exec(ctx, query, args...)
	if err != nil || forceErr == "query" {
		err = fmt.Errorf("updateRegister call failed:%v", err)
		z.Error(err.Error())
		return err
	}
	if action == "clear" && len(practiceIds) == 0 {
		//清空与报名计划绑定的练习
	}
	err = UpsertRegisterPractice(ctx, tx, registration.ID.Int64, practiceIds, userID)
	if err != nil {
		return err
	}
	return nil
}
func AddRegister(ctx context.Context, registration *cmn.TRegisterPlan, practiceIds []int64, userID int64) error {
	var id int64
	now := time.Now().UnixMilli()
	//用于测试，强制执行某些错误分支
	forceErr, _ := ctx.Value("force-error").(string)
	sqlxDB := cmn.GetPgxConn()
	tx, err := sqlxDB.Begin(ctx)
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
			if err != nil {
				z.Error(err.Error())
			}
		} else {
			// 无错误则提交
			err = tx.Commit(ctx)
			if forceErr == "commit" {
				err = fmt.Errorf("commit failed")
			}
			if err != nil {
				z.Error(err.Error())
			}
		}
	}()
	s := `
	INSERT INTO assessuser.t_register_plan (name , start_time , end_time , review_end_time , reviewer_ids , max_number , course , exam_plan_location , creator , updated_by , create_time , update_time , status)
	VALUES ($1,$2 ,$3 ,$4 , $5 ,$6 ,$7 ,$8 ,$9 ,$10 ,$11 ,$12 ,$13) RETURNING id
`
	err = tx.QueryRow(ctx, s, registration.Name, registration.StartTime, registration.EndTime, registration.ReviewEndTime, registration.ReviewerIds, registration.MaxNumber, registration.Course, registration.ExamPlanLocation, userID, userID, now, now, RegisterStatus.PendingRelease).Scan(&id)
	if forceErr == "QueryRow" {
		err = fmt.Errorf("QueryRow failed")
	}
	if err != nil {
		err = fmt.Errorf("添加报名计划失败:%v", err)
		z.Error(err.Error())
		return err
	}
	registration.ID = null.IntFrom(id)
	if practiceIds == nil || len(practiceIds) == 0 {
		z.Sugar().Debugf("打印输出一下增加SQL语句:%v", practiceIds)
		return nil
	}
	err = UpsertRegisterPractice(ctx, tx, registration.ID.Int64, practiceIds, userID)
	if err != nil {
		return err
	}
	return nil
}
func UpsertRegisterPractice(ctx context.Context, tx pgx.Tx, registerID int64, practiceIds []int64, userID int64) error {
	forceErr, _ := ctx.Value("force-error").(string)
	if registerID <= 0 {
		err := fmt.Errorf("registerID不能小于等于0")
		z.Error(err.Error())
		return err
	}
	if userID <= 0 {
		err := fmt.Errorf("userID不能小于等于0")
		z.Error(err.Error())
		return err
	}
	now := time.Now().UnixMilli()
	if practiceIds == nil || len(practiceIds) == 0 {
		//删除当前报名列表下面的所有练习
		delSQL := `
		UPDATE assessuser.t_register_practice 
		SET status =$1 , update_time= $2 , updated_by = $3
		WHERE register_id =$4 AND status != $1
`
		_, err := tx.Exec(ctx, delSQL, RegisterPracticeStatus.Delete, now, userID, registerID)
		z.Sugar().Debugf("打印输出一下增加SQL语句:%v", delSQL)
		if err != nil || forceErr == "del" {
			err = fmt.Errorf("删除报名计划下的所有练习失败:%v", err)
			z.Error(err.Error())
			return err
		}
		return nil
	}
	//upsert名单
	addRpStr := strings.Repeat("(?,?,?,?,?,?,?),", len(practiceIds)-1) + "(?,?,?,?,?,?,?)"
	addRpArgs := make([]interface{}, 0, len(practiceIds)*7+1)
	for _, practiceId := range practiceIds {
		addRpArgs = append(addRpArgs,
			registerID, practiceId, userID, userID, now, now, RegisterPracticeStatus.Normal,
		)
	}
	addRpArgs = append(addRpArgs, RegisterPracticeStatus.Normal)
	t := `
		INSERT INTO assessuser.t_register_practice 
			(register_id , practice_id , creator , updated_by , create_time , update_time , status)
		VALUES  %s
		ON CONFLICT (register_id , practice_id)
		DO UPDATE SET
		   status =EXCLUDED.status,
		   updated_by = EXCLUDED.updated_by,
		   update_time = EXCLUDED.update_time
		WHERE assessuser.t_register_practice.status IS DISTINCT FROM ?
		   
`
	s1 := fmt.Sprintf(t, addRpStr)
	//修正格式
	addRQuery, args, err := sqlx.In(s1, addRpArgs...)
	if err != nil {
		err = fmt.Errorf("添加名单参数错误:%v", err)
		z.Error(err.Error())
		return err
	}
	addRQuery = sqlx.Rebind(sqlx.DOLLAR, addRQuery)
	z.Sugar().Debugf("打印输出一下增加SQL语句:%v", addRQuery)
	z.Sugar().Debugf("打印输出一下增加SQL参数:%v", args...)
	_, err = tx.Exec(ctx, addRQuery, args...)
	if err != nil || forceErr == "query2" {
		err = fmt.Errorf("添加名单失败:%v", err)
		z.Error(err.Error())
		return err
	}
	//删除没选择的练习
	var valueExpr []string
	var delRArgs []interface{}
	delRArgs = append(delRArgs, RegisterPracticeStatus.Delete, now, userID, registerID)
	for _, practiceId := range practiceIds {
		valueExpr = append(valueExpr, fmt.Sprintf("($%d::bigint)", len(delRArgs)+1))
		delRArgs = append(delRArgs, practiceId)
	}
	t2 := `
		UPDATE assessuser.t_register_practice t
		SET status = $1, update_time = $2, updated_by = $3
		WHERE t.register_id = $4
			AND NOT EXISTS (
				SELECT 1 
				FROM (VALUES %s) AS excluded(practice_id)
				WHERE t.practice_id = excluded.practice_id
			)
	`
	s2 := fmt.Sprintf(t2, strings.Join(valueExpr, ", "))
	z.Sugar().Debugf("打印输出一下删除SQL语句:%v", s2)
	z.Sugar().Debugf("打印输出一下删除SQL参数:%v", delRArgs...)
	_, err = tx.Exec(ctx, s2, delRArgs...)
	if err != nil || forceErr == "query3" {
		err = fmt.Errorf("删除名单失败:%v", err)
		z.Error(err.Error())
		return err
	}
	return nil
}
func LoadRegisterById(ctx context.Context, registerID int64) (*cmn.TRegisterPlan, []cmn.TPractice, []Reviewer, int64, error) {
	forceErr := ctx.Value("force-error")

	s := `
	SELECT r.id, r.name , r.course , r.review_end_time , r. max_number , r.start_time , r.end_time , r.reviewer_ids , r.exam_plan_location ,r.status ,COALESCE((SELECT count(*) FROM assessuser.t_exam_plan_student WHERE register_id = r.id AND status NOT IN ($1 ,$2)),0)
	FROM assessuser.t_register_plan r WHERE r.id = $3
`
	sqlxDB := cmn.GetDbConn()
	row := sqlxDB.QueryRowContext(ctx, s, RegisterStudentStatus.Apply, RegisterStudentStatus.Moved, registerID)
	var register cmn.TRegisterPlan
	var currentNumber int64
	err := row.Scan(&register.ID, &register.Name, &register.Course, &register.ReviewEndTime, &register.MaxNumber, &register.StartTime, &register.EndTime, &register.ReviewerIds, &register.ExamPlanLocation, &register.Status, &currentNumber)
	if err != nil || forceErr == "query" {
		err = fmt.Errorf("查询报名计划失败:%v", err)
		z.Error(err.Error())
		return nil, nil, nil, -1, err
	}
	//查询报名计划相关练习
	s = `SELECT r.practice_id ,p.name,p.type FROM assessuser.t_register_practice r JOIN assessuser.t_practice p ON p.id=r.practice_id WHERE r.register_id =$1`
	rows, err := sqlxDB.QueryxContext(ctx, s, registerID)
	if err != nil || forceErr == "query2" {
		err = fmt.Errorf("查询报名计划下的练习失败:%v", err)
		z.Error(err.Error())
		return nil, nil, nil, -1, err
	}
	var practices []cmn.TPractice
	for rows.Next() {
		var practice cmn.TPractice
		err := rows.Scan(&practice.ID, &practice.Name, &practice.Type)
		if err != nil {
			z.Error(err.Error())
			return nil, nil, nil, -1, err
		}
		practices = append(practices, practice)
	}
	//查询报名计划审查人名字
	s = `SELECT id,official_name FROM assessuser.t_user WHERE id IN (SELECT unnest(r.reviewer_ids) FROM assessuser.t_register_plan r 
    WHERE r.id = $1)`
	rows, err = sqlxDB.QueryxContext(ctx, s, registerID)
	if err != nil || forceErr == "query3" {
		err = fmt.Errorf("查询报名计划下的审查人失败:%v", err)
		z.Error(err.Error())
		return nil, nil, nil, -1, err
	}
	var reviewers []Reviewer
	for rows.Next() {
		var reviewer Reviewer
		err := rows.Scan(&reviewer.ID, &reviewer.OfficialName)
		if err != nil {
			z.Error(err.Error())
			return nil, nil, nil, -1, err
		}
		reviewers = append(reviewers, reviewer)
	}

	return &register, practices, reviewers, currentNumber, nil

}

// 批量操作报名计划状态
func OperateRegisterStatus(ctx context.Context, registerIDs []int64, status string, userID int64) error {
	if len(registerIDs) == 0 {
		err := fmt.Errorf("registerIDs不能为空")
		z.Error(err.Error())
		return err
	}
	if userID <= 0 {
		err := fmt.Errorf("userID不能小于等于0")
		z.Error(err.Error())
		return err
	}
	forceErr := ctx.Value("force-error")
	now := time.Now().UnixMilli()
	//获取每个报名计划的信息
	Rs, err := LoadRegisterByIds(ctx, registerIDs)
	if err != nil {
		return err
	}
	sqlxDB := cmn.GetPgxConn()
	tx, err := sqlxDB.Begin(ctx)
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
	for _, register := range Rs {
		if signStatus == "" {
			signStatus = register.Status.String
		}
		if register.Status.String != signStatus {
			err = fmt.Errorf("此时要批量操作的报名计划状态不一，无法进行批量操作")
			z.Error(err.Error())
			return err
		}
		if register.Status.String == RegisterStatus.Disabled {
			err = fmt.Errorf("报名计划状态为作废，无法进行操作")
			z.Error(err.Error())
			return err
		}

	}
	if status == RegisterStatus.Released {
		for _, register := range Rs {
			s := `
			UPDATE assessuser.t_register_plan r  SET status = $1,update_time = $2, updated_by = $3  WHERE id = $4
`
			_, err = tx.Exec(ctx, s, RegisterStatus.Released, now, userID, register.ID)
			if err != nil || forceErr == "operate" {
				err = fmt.Errorf("更新报名计划状态失败:%v", err)
				z.Error(err.Error())
				return err
			}
		}
		return nil

	} else if status == RegisterStatus.Disabled {
		for _, register := range Rs {
			s := `
			UPDATE assessuser.t_register_plan  SET status = $1,update_time = $2, updated_by = $3  WHERE id = $4
`
			_, err = tx.Exec(ctx, s, RegisterStatus.Disabled, now, userID, register.ID)
			if err != nil || forceErr == "operate" {
				err = fmt.Errorf("更新报名计划状态失败:%v", err)
				z.Error(err.Error())
			}
			//让与其相关联的练习关系失效
			s = `
			UPDATE assessuser.t_register_practice  SET status = $1,update_time = $2, updated_by = $3  WHERE register_id = $4
`
			_, err = tx.Exec(ctx, s, RegisterPracticeStatus.Delete, now, userID, register.ID)
			if err != nil || forceErr == "operate1" {
				err = fmt.Errorf("更新报名计划状态失败:%v", err)
				z.Error(err.Error())
			}
			//让与其相关联的学生也失效
			s = `
			UPDATE assessuser.t_exam_plan_student   SET status = $1,update_time = $2, updated_by = $3  WHERE register_id = $4
`
			_, err = tx.Exec(ctx, s, RegisterStudentStatus.Deleted, now, userID, register.ID)
			if err != nil || forceErr == "operate2" {
				err = fmt.Errorf("更新报名计划状态失败:%v", err)
				z.Error(err.Error())
			}

		}
		return nil
	} else if status == RegisterStatus.Deleted {
		registerIsUsed := false
		var invalidName []string
		for _, register := range Rs {
			s := `
			SELECT EXISTS(SELECT 1 FROM assessuser.t_exam_plan_student eps WHERE eps.register_id =$1 )
`
			err = tx.QueryRow(ctx, s, register.ID).Scan(&registerIsUsed)
			if err != nil || forceErr == "query1" {
				err = fmt.Errorf("查询报名计划是否被使用失败:%v", err)
				z.Error(err.Error())
				return err
			}
			if registerIsUsed {
				invalidName = append(invalidName, register.Name.String)
			}
		}
		if len(invalidName) > 0 {
			err = fmt.Errorf("此时报名计划名称为：%v的报名计划已有学生报名，不能删除", invalidName)
			z.Error(err.Error())
			return err
		}
		s := `
		UPDATE assessuser.t_register_plan SET status = $1,update_time = $2, updated_by = $3  WHERE id = ANY($4)
`
		_, err = tx.Exec(ctx, s, RegisterStatus.Deleted, now, userID, registerIDs)
		if err != nil || forceErr == "operate" {
			err = fmt.Errorf("删除报名计划失败:%v", err)
			z.Error(err.Error())
			return err
		}
		//删除与其关联的练习
		s = `
		UPDATE  assessuser.t_register_practice SET status = $1,update_time = $2, updated_by = $3  WHERE register_id = ANY($4)
`
		_, err = tx.Exec(ctx, s, RegisterPracticeStatus.Delete, now, userID, registerIDs)
	}
	return nil
	//

}
func LoadRegisterByIds(ctx context.Context, registerIDs []int64) (registers []*cmn.TRegisterPlan, err error) {
	forceErr := ctx.Value("force-error")
	s := `
	SELECT r.id, r.name , r.course , r.review_end_time , r. max_number , r.start_time , r.end_time , r.reviewer_ids , r.exam_plan_location ,r.status
	FROM t_register_plan r WHERE r.id = ANY($1) AND r.status != $2
`
	sqlxDB := cmn.GetPgxConn()
	rows, err := sqlxDB.Query(ctx, s, registerIDs, RegisterStatus.Deleted)
	if err != nil || forceErr == "query" {
		err = fmt.Errorf("查询报名计划失败:%v", err)
		z.Error(err.Error())
		return nil, err

	}
	for rows.Next() {
		var register cmn.TRegisterPlan
		err = rows.Scan(&register.ID, &register.Name, &register.Course, &register.ReviewEndTime, &register.MaxNumber, &register.StartTime, &register.EndTime, &register.ReviewerIds, &register.ExamPlanLocation, &register.Status)
		if err != nil || forceErr == "lScan" {
			err = fmt.Errorf("获取报名计划信息失败:%v", err)
			z.Error(err.Error())
			return nil, err
		}
		registers = append(registers, &register)
	}
	return registers, nil
}

// 批量通过或不通过学生审核
func OperateRegisterStudentStatus(ctx context.Context, tx pgx.Tx, ids []int64, status string, userID int64, RegisterID int64, failReason string) error {
	if status == "" {
		return fmt.Errorf("请选择操作")
	}
	if userID <= 0 {
		err := fmt.Errorf("用户ID错误:%v", userID)
		z.Error(err.Error())
		return err
	}
	if RegisterID <= 0 {
		err := fmt.Errorf("报名计划ID错误:%v", RegisterID)
		z.Error(err.Error())
		return err
	}
	now := time.Now().UnixMilli()
	forceErr, _ := ctx.Value("force-error").(string)
	//获取报名计划信息
	register, _, _, _, err := LoadRegisterById(ctx, RegisterID)
	if err != nil {
		return err
	}
	if register.Status.String != RegisterStatus.Released {
		err := fmt.Errorf("当前报名计划状态为：%v，不能进行操作", register.Status.String)
		z.Error(err.Error())
		return err
	}
	//获取学生状态
	var studentStatus string
	students, err := LoadRegisterStudentStatusByIds(ctx, ids)
	for _, student := range students {
		if studentStatus == "" {
			studentStatus = student.Status.String
		}
		if student.Status.String != studentStatus {
			err := fmt.Errorf("此时要批量操作的学生状态不一，无法进行批量操作")
			z.Error(err.Error())
			return err
		}
	}
	if studentStatus == "" {
		err := fmt.Errorf("请选择学生")
		z.Error(err.Error())
		return err
	}
	//对学生状态进行操作
	var s string
	if tx != nil {
		//使用传进来的tx
		s = `
	UPDATE assessuser.t_exam_plan_student SET status = $1,update_time = $2, updated_by = $3   WHERE student_id = ANY($4) AND register_id =$5`
		_, err = tx.Exec(ctx, s, status, now, userID, ids, RegisterID)
		if err != nil || forceErr == "pQuery" {
			err = fmt.Errorf("设置迁移学生状态失败:%v", err)
			z.Error(err.Error())
			return err
		}

	} else {
		//批量通过或不通过学生审核
		sqlxDB := cmn.GetPgxConn()
		tx, err = sqlxDB.Begin(ctx)
		if err != nil || forceErr == "begin" {
			err = fmt.Errorf("开启事务失败:%v", err)
			z.Error(err.Error())
			return err
		}
		defer func() {
			if forceErr == "rollback" {
				err = fmt.Errorf("触发回滚")
			}
			if err != nil {
				_ = tx.Rollback(ctx)
				if forceErr == "rollback" {
					err = fmt.Errorf("回滚失败")
				}
				if err != nil {
					z.Error(err.Error())
					return
				}
			}
			err = tx.Commit(ctx)
			if forceErr == "commit" {
				err = fmt.Errorf("触发commit")
			}
			if err != nil {
				z.Error(err.Error())
				return
			}
		}()
		s = `
	UPDATE assessuser.t_exam_plan_student SET status = $1,update_time = $2, updated_by = $3 , fail_reason =$4 ,reviewer=$5  WHERE student_id = ANY($6) AND register_id =$7`
		_, err = tx.Exec(ctx, s, status, now, userID, failReason, userID, ids, RegisterID)
		if err != nil || forceErr == "pQuery" {
			err = fmt.Errorf("更新学生状态失败:%v", err)
			z.Error(err.Error())
			return err
		}
	}
	return nil
}
func LoadRegisterStudentStatusByIds(ctx context.Context, ids []int64) (students []cmn.TExamPlanStudent, err error) {
	forceErr, _ := ctx.Value("force-error").(string)
	s := `
	SELECT  eps.id ,eps.student_id , eps.status
	FROM t_exam_plan_student eps WHERE eps.student_id = ANY($1)
`
	sqlxDB := cmn.GetPgxConn()
	rows, err := sqlxDB.Query(ctx, s, ids)
	if err != nil || forceErr == "query" {
		err = fmt.Errorf("查询学生状态失败:%v", err)
		z.Error(err.Error())
		return nil, err
	}
	for rows.Next() {
		var student cmn.TExamPlanStudent
		err = rows.Scan(&student.ID, &student.StudentID, &student.Status)
		if err != nil || forceErr == "lScan" {
			err = fmt.Errorf("获取学生状态失败:%v", err)
			z.Error(err.Error())
			return nil, err
		}
		students = append(students, student)
	}
	return students, nil
}

// 批量导入学生
func UpsertRegisterStudent(ctx context.Context, registerID int64, studentIDs []registerStudentType, userID int64) error {
	if registerID <= 0 {
		err := fmt.Errorf("报名计划ID错误:%v", registerID)
		z.Error(err.Error())
		return err
	}
	if userID <= 0 {
		err := fmt.Errorf("用户ID错误:%v", userID)
		z.Error(err.Error())
		return err
	}
	now := time.Now().UnixMilli()
	forceErr, _ := ctx.Value("force-error").(string)
	//要检验报名计划正在发布中才能导入
	register, _, _, currentNumber, err := LoadRegisterById(ctx, registerID)
	if err != nil {
		z.Error(err.Error())
		return err
	}
	if currentNumber+int64(len(studentIDs)) > register.MaxNumber.Int64 {
		err := fmt.Errorf("导入学生人数超出报名计划可容纳人数")
		z.Error(err.Error())
		return err
	}
	if register.Status.String != RegisterStatus.Released {
		err := fmt.Errorf("当前报名计划状态为：%v，不能导入学生", register.Status.String)
		z.Error(err.Error())
		return err
	}

	sqlxDB := cmn.GetPgxConn()
	tx, err := sqlxDB.Begin(ctx)
	if err != nil || forceErr == "begin" {
		err = fmt.Errorf("开启事务失败:%v", err)
		z.Error(err.Error())
		return err
	}
	defer func() {
		if forceErr == "rollback" {
			err = fmt.Errorf("触发回滚")
		}
		if err != nil {
			_ = tx.Rollback(ctx)
			if forceErr == "rollback" {
				err = fmt.Errorf("回滚失败")
			}
			if err != nil {
				z.Error(err.Error())
				return
			}
		}
		err = tx.Commit(ctx)
		if forceErr == "commit" {
			err = fmt.Errorf("触发commit")
		}
		if err != nil {
			z.Error(err.Error())
			return
		}
	}()
	//upsert名单
	addRStr := strings.Repeat("(?,?,?,?,?,?,?,?,?),", len(studentIDs)-1) + "(?,?,?,?,?,?,?,?,?)"
	addRArgs := make([]interface{}, 0, len(studentIDs)*9+1)

	for _, student := range studentIDs {
		addRArgs = append(addRArgs,
			registerID, student.StudentID, RegisterType.Import, student.ExamType, now, userID, userID, now, RegisterStudentStatus.Approved,
		)
	}
	addRArgs = append(addRArgs, RegisterStudentStatus.Approved)
	s := `
	 INSERT INTO assessuser.t_exam_plan_student(register_id, student_id,type,exam_type, create_time, creator, updated_by , update_time ,status) VALUES %s
	 ON CONFLICT (register_id, student_id) DO UPDATE SET 
	  status = EXCLUDED.status,
            updated_by = EXCLUDED.updated_by,
            update_time = EXCLUDED.update_time
        WHERE assessuser.t_exam_plan_student.status IS DISTINCT FROM ?
 `
	s1 := fmt.Sprintf(s, addRStr)
	addRQuery, args, err := sqlx.In(s1, addRArgs...)
	if err != nil || forceErr == "sqlxIn" {
		err = fmt.Errorf("批量导入学生参数处理失败:%v", err)
		z.Error(err.Error())
		return err
	}
	addRQuery = sqlx.Rebind(sqlx.DOLLAR, addRQuery)
	z.Sugar().Debugf("打印输出一下增加SQL语句:%v", addRQuery)
	z.Sugar().Debugf("打印输出一下增加SQL参数:%v", args...)
	_, err = tx.Exec(ctx, addRQuery, args...)
	if err != nil || forceErr == "pQuery" {
		err = fmt.Errorf("批量导入学生失败:%v", err)
		z.Error(err.Error())
		return err
	}
	return nil
}

// 移动学生
func MoveStudent(ctx context.Context, fromRegisterID int64, toRegisterID int64, students []registerStudentType, status string, userID int64) error {
	if fromRegisterID <= 0 || toRegisterID <= 0 {
		err := fmt.Errorf("报名计划ID错误 ")
		z.Error(err.Error())
		return err
	}
	if userID <= 0 {
		err := fmt.Errorf("用户ID错误:%v", userID)
		z.Error(err.Error())
		return err
	}
	now := time.Now().UnixMilli()
	forceErr, _ := ctx.Value("force-error").(string)
	//检验目标报名计划是否处于发布状态
	register, _, _, currentNumber, err := LoadRegisterById(ctx, toRegisterID)
	if err != nil {
		z.Error(err.Error())
		return err
	}
	var registerStatus string
	registerStatus = register.Status.String
	if registerStatus != RegisterStatus.Released {
		err := fmt.Errorf("目标报名计划状态为：%v，不能移动学生", registerStatus)
		z.Error(err.Error())
		return err
	}
	if register.MaxNumber.Int64 < int64(len(students))+currentNumber {
		err := fmt.Errorf("目标报名计划可容纳人数不足，无法进行移动")
		z.Error(err.Error())
		return err
	}
	sqlxDB := cmn.GetPgxConn()
	tx, err := sqlxDB.Begin(ctx)
	if err != nil || forceErr == "begin" {
		err = fmt.Errorf("开启事务失败:%v", err)
		z.Error(err.Error())
		return err
	}
	defer func() {
		if forceErr == "rollback" {
			err = fmt.Errorf("触发回滚")
		}
		if err != nil {
			_ = tx.Rollback(ctx)
			if forceErr == "rollback" {
				err = fmt.Errorf("回滚失败")
			}
			if err != nil {
				z.Error(err.Error())
				return
			}
		}
		err = tx.Commit(ctx)
		if forceErr == "commit" {
			err = fmt.Errorf("触发commit")
		}
		if err != nil {
			z.Error(err.Error())
			return
		}
	}()
	var planStudents []*cmn.TExamPlanStudent
	//获取原本学生的状态
	s := `SELECT * FROM  assessuser.t_exam_plan_student WHERE register_id =$1 AND student_id IN ANY ($2) `
	rows, err := tx.Query(ctx, s, fromRegisterID, students)
	if err != nil {
		err = fmt.Errorf("获取学生状态失败:%v", err)
		z.Error(err.Error())
		return err
	}
	for rows.Next() {
		var student *cmn.TExamPlanStudent
		err = rows.Scan(
			&student.ID,
			&student.StudentID,
			&student.RegisterID,
			&student.Type,
			&student.FailReason,
			&student.ExamType,
			&student.RegisterTime,
			&student.Reviewer,
			&student.Addi,
			&student.Creator,
			&student.UpdatedBy,
			&student.CreateTime,
			&student.UpdateTime,
			&student.Status,
		)
		if err != nil {
			err = fmt.Errorf("扫描学生状态失败 错误: %w", err)
			z.Error(err.Error())
			return err
		}
		planStudents = append(planStudents, student)
	}
	//导入新的计划当中

	//upsert名单
	addRStr := strings.Repeat("(?,?,?,?,?,?,?,?,?,?,?,?),", len(planStudents)-1) + "(?,?,?,?,?,?,?,?,?,?,?,?)"
	addRArgs := make([]interface{}, 0, len(planStudents)*12+1)

	for _, student := range planStudents {
		addRArgs = append(addRArgs,
			toRegisterID, student.StudentID, RegisterType.Import, student.ExamType, now, userID, userID, now, student.Status, student.FailReason, student.Reviewer, student.RegisterTime,
		)
	}
	addRArgs = append(addRArgs, RegisterStudentStatus.Approved)
	s = `
	 INSERT INTO assessuser.t_exam_plan_student(register_id, student_id,type,exam_type, create_time, creator, updated_by , update_time ,status, fail_reason,reviewer,register_time) VALUES %s
	 ON CONFLICT (register_id, student_id) DO UPDATE SET 
	  status = EXCLUDED.status,
            updated_by = EXCLUDED.updated_by,
            update_time = EXCLUDED.update_time
        WHERE assessuser.t_exam_plan_student.status IS DISTINCT FROM ?
 `
	s1 := fmt.Sprintf(s, addRStr)
	addRQuery, args, err := sqlx.In(s1, addRArgs...)
	if err != nil || forceErr == "sqlxIn" {
		err = fmt.Errorf("批量移动学生参数处理失败:%v", err)
		z.Error(err.Error())
		return err
	}
	addRQuery = sqlx.Rebind(sqlx.DOLLAR, addRQuery)
	z.Sugar().Debugf("打印输出一下增加SQL语句:%v", addRQuery)
	z.Sugar().Debugf("打印输出一下增加SQL参数:%v", args...)
	_, err = tx.Exec(ctx, addRQuery, args...)
	if err != nil || forceErr == "pQuery" {
		err = fmt.Errorf("批量移动学生失败:%v", err)
		z.Error(err.Error())
		return err
	}
	//修改原来的报名计划的学生状态为已迁移
	var studentIDs []int64
	for _, student := range students {
		studentIDs = append(studentIDs, student.StudentID)
	}
	//调用更新原本学生状态方法
	err = OperateRegisterStudentStatus(ctx, tx, studentIDs, status, userID, fromRegisterID, "")
	if err != nil {
		z.Error(err.Error())
		return err
	}

	return nil
}
func ListReviewers(ctx context.Context, userID int64, registerID int64, name string, page int, pageSize int, orderBy []string) ([]Map, int, error) {
	if registerID <= 0 {
		err := fmt.Errorf("报名计划ID错误 ")
		z.Error(err.Error())
		return nil, 0, err
	}
	if userID <= 0 {
		err := fmt.Errorf("用户ID错误:%v", userID)
		z.Error(err.Error())
		return nil, 0, err
	}
	forceErr, _ := ctx.Value("force-error").(string)
	//获取报名计划的审查者id
	register, _, _, _, err := LoadRegisterById(ctx, registerID)
	if err != nil {
		z.Error(err.Error())
		return nil, 0, err
	}
	result := make([]Map, 0)
	var clauses []string
	var args []interface{}
	if name != "" {
		clauses = append(clauses, fmt.Sprintf("%s LIKE $%d", "u.official_name", len(args)+1))
		args = append(args, "%"+name+"%")
	}
	clauses = append(clauses, fmt.Sprintf("%s =ANY ($%d)", "u.id", len(args)+1))
	args = append(args, register.ReviewerIds)
	s := `SELECT u.id , COALESCE( u.official_name,'') ,COALESCE(u.gender,''), COALESCE(u.mobile_phone,''), COALESCE(u.id_card_no,'') , COALESCE(u.id_card_type,'') FROM t_user u `
	//构建查询顺序
	//构建查询条件
	if len(clauses) > 0 {
		s += "WHERE " + strings.Join(clauses, " AND ")
	}
	//添加orderBy语句
	if len(orderBy) > 0 {
		s += " ORDER BY " + strings.Join(orderBy, ", ")
	}
	//添加分页信息
	offSet := (page - 1) * pageSize
	s += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, pageSize, offSet)

	z.Sugar().Debugf("打印输出一下这个操作语句：%v", s)
	z.Sugar().Debugf("打印输出一下参数表：%v", args)
	// 这些实体查询的每个函数之间作用都不一样，需要花时间去了解这个函数的具体用处了
	sqlxDB := cmn.GetDbConn()
	rows, err := sqlxDB.QueryxContext(ctx, s, args...)
	if err != nil || forceErr == "query" {
		err = fmt.Errorf("search register failed:%v", err)
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

	for rows.Next() {
		M := Map{}
		var reviewer Reviewer
		err = rows.Scan(&reviewer.ID, &reviewer.OfficialName, &reviewer.Gender, &reviewer.MobilePhone, &reviewer.IDCardNo, &reviewer.IDCardType)
		if err != nil || forceErr == "row scan" {
			err = fmt.Errorf("row scan failed:%v", err)
			z.Error(err.Error())
			return nil, 0, err

		}
		M["reviewer"] = reviewer
		result = append(result, M)
	}
	return result, len(result), nil

}
