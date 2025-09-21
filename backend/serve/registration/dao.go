package registration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"strings"
	"time"
	"w2w.io/cmn"
	"w2w.io/null"
	"w2w.io/serve/auth_mgt"
	"w2w.io/serve/practice_mgt"
)

// 教师查看报名计划列表
func ListRegisterT(ctx context.Context, name string, course string, status string, orderBy []string, page int, pageSize int, userID int64, searchType string) ([]Map, int, error) {
	result := make([]Map, 0)
	// 用于测试，强制执行某些错误分支
	forceErr, _ := ctx.Value("force-error").(string)
	//从上下文中获取权限信息
	authority, _ := ctx.Value("authority").(auth_mgt.Authority)

	//构建查询条件
	var clauses []string
	//构建占位符
	var args []interface{}
	if searchType == "02" {
		args = append(args, RegisterStudentStatus.Apply)
		args = append(args, RegisterStudentStatus.Rejected)
		args = append(args, RegisterStudentStatus.Pending)
		args = append(args, RegisterStudentStatus.Moved)
		args = append(args, RegisterPracticeStatus.Normal)
		if name != "" {
			clauses = append(clauses, fmt.Sprintf("%s LIKE $%d", "r.name", len(args)+1))
			args = append(args, "%"+name+"%")
		}
		if course != "" {
			clauses = append(clauses, fmt.Sprintf("%s  =$%d", "r.course", len(args)+1))
			args = append(args, course)
		}

		clauses = append(clauses, fmt.Sprintf("r.status = $%d", len(args)+1))
		args = append(args, RegisterStatus.ReviewEnding)
	} else if searchType == "04" {
		args = append(args, RegisterStudentStatus.Apply)
		args = append(args, RegisterStudentStatus.Apply)
		args = append(args, RegisterStudentStatus.Apply)
		args = append(args, RegisterStudentStatus.Apply)
		args = append(args, RegisterPracticeStatus.Normal)
		if name != "" {
			clauses = append(clauses, fmt.Sprintf("%s LIKE $%d", "r.name", len(args)+1))
			args = append(args, "%"+name+"%")
		}
		if course != "" {
			clauses = append(clauses, fmt.Sprintf("%s  =$%d", "r.course", len(args)+1))
			args = append(args, course)
		}

		clauses = append(clauses, fmt.Sprintf("r.status NOT IN($%d , $%d , $%d , $%d)", len(args)+1, len(args)+2, len(args)+3, len(args)+4))
		args = append(args, RegisterStatus.ReviewEnding, RegisterStatus.Deleted, RegisterStatus.Disabled, RegisterStatus.Cancel)
	} else {
		args = append(args, RegisterStudentStatus.Apply)
		args = append(args, RegisterStudentStatus.Apply)
		args = append(args, RegisterStudentStatus.Apply)
		args = append(args, RegisterStudentStatus.Apply)
		args = append(args, RegisterPracticeStatus.Normal)
		if name != "" {
			clauses = append(clauses, fmt.Sprintf("%s LIKE $%d", "r.name", len(args)+1))
			args = append(args, "%"+name+"%")
		}
		if course != "" {
			clauses = append(clauses, fmt.Sprintf("%s  =$%d", "r.course", len(args)+1))
			args = append(args, course)
		}

		if status != "" {
			clauses = append(clauses, fmt.Sprintf("%s  =$%d", "r.status", len(args)+1))
			args = append(args, status)
		}
		clauses = append(clauses, fmt.Sprintf("r.status != $%d", len(args)+1))
		args = append(args, RegisterStatus.Deleted)
	}
	clauses = append(clauses, fmt.Sprintf("(r.creator = $%d OR r.domain_id = ANY($%d))", len(args)+1, len(args)+2))
	args = append(args, userID, authority.AccessibleDomains)

	s := `
	SELECT r.id, r.name , r.course , COALESCE((SELECT COUNT(*) FROM assessuser.t_exam_plan_student eps WHERE eps.register_id=r.id AND eps.status NOT IN ($1 ,$2 ,$3 ,$4)),0) , r.max_number , r.review_end_time , r.start_time , r.end_time , COALESCE(STRING_AGG(p.name, '、'),'') , r.status ,r.exam_plan_location
	FROM assessuser.t_register_plan r  LEFT JOIN assessuser.t_register_practice rp ON rp.register_id=r.id AND rp.status=$5
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
	sqlxDB := cmn.GetPgxConn()
	rows1, err := sqlxDB.Query(ctx, s, args...)
	defer rows1.Close()
	if err != nil || forceErr == "query" {
		err = fmt.Errorf("search register failed:%v", err)
		z.Error(err.Error())
		return nil, 0, err
	}

	for rows1.Next() {
		M := Map{}
		var r cmn.TRegisterPlan
		var practiceName string
		var studentCount int64
		err = rows1.Scan(&r.ID, &r.Name, &r.Course, &studentCount, &r.MaxNumber, &r.ReviewEndTime, &r.StartTime, &r.EndTime, &practiceName, &r.Status, &r.ExamPlanLocation)
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
	clauses = []string{}
	args = []interface{}{}
	if name != "" {
		clauses = append(clauses, fmt.Sprintf("%s LIKE $%d", "r.name", len(args)+1))
		args = append(args, "%"+name+"%")
	}
	if course != "" {
		clauses = append(clauses, fmt.Sprintf("%s  =$%d", "r.course", len(args)+1))
		args = append(args, course)
	}
	if status != "" {
		clauses = append(clauses, fmt.Sprintf("%s  =$%d", "r.status", len(args)+1))
		args = append(args, status)
	}
	clauses = append(clauses, fmt.Sprintf("r.status != $%d", len(args)+1))
	args = append(args, RegisterStatus.Deleted)
	clauses = append(clauses, fmt.Sprintf("(r.creator = $%d OR r.domain_id = ANY($%d))", len(args)+1, len(args)+2))
	args = append(args, userID, authority.AccessibleDomains)
	if searchType == "02" {
		clauses = append(clauses, fmt.Sprintf("r.status = $%d", len(args)+1))
		args = append(args, RegisterStatus.ReviewEnding)
	} else if searchType == "04" {
		clauses = append(clauses, fmt.Sprintf("r.status NOT IN($%d , $%d , $%d , $%d)", len(args)+1, len(args)+2, len(args)+3, len(args)+4))
		args = append(args, RegisterStatus.ReviewEnding, RegisterStatus.Deleted, RegisterStatus.Disabled, RegisterStatus.Cancel)
	}
	//查询总数
	s = ` SELECT COUNT(*) FROM assessuser.t_register_plan r `
	if len(clauses) > 0 {
		s += " WHERE " + strings.Join(clauses, " AND ")
	}

	z.Sugar().Debugf("打印输出一下这个操作语句：%v", s)
	z.Sugar().Debugf("打印输出一下参数表：%v", args)
	rows2, err := sqlxDB.Query(ctx, s, args...)
	defer rows2.Close()
	if err != nil || forceErr == "queryx" {
		err = fmt.Errorf("查询数据失败:%v", err)
		z.Error(err.Error())
		return nil, 0, err
	}
	var total int

	for rows2.Next() {
		err = rows2.Scan(&total)
		if err != nil || forceErr == "lScan" {
			err = fmt.Errorf("解析数据失败:%v", err)
			z.Error(err.Error())
			return nil, 0, err
		}
	}

	return result, total, nil
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
		args = append(args, course)
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
	sqlxDB := cmn.GetPgxConn()
	rows1, err := sqlxDB.Query(ctx, s, args...)
	defer rows1.Close()
	if err != nil || forceErr == "query" {
		err = fmt.Errorf("search register failed:%v", err)
		z.Error(err.Error())
		return nil, 0, err
	}

	for rows1.Next() {
		M := Map{}
		var r cmn.TRegisterPlan
		var student cmn.TExamPlanStudent
		err = rows1.Scan(&r.ID, &r.Name, &r.ReviewEndTime, &r.StartTime, &r.EndTime, &r.Course, &r.ExamPlanLocation, &student.ExamType, &student.Status)
		if err != nil || forceErr == "scan" {
			err = fmt.Errorf("解析数据失败:%v", err)
			z.Error(err.Error())
			return nil, 0, err
		}
		M["register"] = r
		M["student"] = student
		result = append(result, M)
	}
	clauses = []string{}
	args = []interface{}{}
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
	//查询总数
	s = ` SELECT COUNT(*) FROM assessuser.t_register_plan r LEFT JOIN assessuser.t_exam_plan_student eps ON eps.register_id = r.id AND eps.student_id =$1`
	if len(clauses) > 0 {
		s += " WHERE " + strings.Join(clauses, " AND ")
	}
	rows2, err := sqlxDB.Query(ctx, s, args...)
	defer rows2.Close()
	if err != nil || forceErr == "queryx" {
		err = fmt.Errorf("查询数据失败:%v", err)
		z.Error(err.Error())
		return nil, 0, err
	}

	var total int

	for rows2.Next() {
		err = rows2.Scan(&total)
		if err != nil || forceErr == "lScan" {
			err = fmt.Errorf("解析数据失败:%v", err)
			z.Error(err.Error())
			return nil, 0, err
		}
	}

	return result, total, nil
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
	//判断当前报名计划是否处于发布状态
	var registerStatus string
	s := `SELECT status FROM assessuser.t_register_plan WHERE id = $1`
	err = tx.QueryRow(ctx, s, registerID).Scan(&registerStatus)
	if err != nil || forceErr == "querysr" {
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
	if (err != nil && !noExist) || forceErr == "queryss" {
		err = fmt.Errorf("query student failed:%v", err)
		z.Error(err.Error())
		return err
	}
	if (studentStatus == RegisterStudentStatus.Apply && status == RegisterStudentStatus.Pending) || (status == RegisterStudentStatus.Pending && noExist) {
		//检验当前的人数是否超过限制
		registerInfo, _, _, current, err := LoadRegisterById(ctx, registerID)
		if err != nil {
			return err
		}
		if (registerInfo.MaxNumber.Int64 < current+int64(len(students))) && registerInfo.MaxNumber.Int64 != 0 {
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
			err = fmt.Errorf("更新当前报名信息失败:%v", err)
			z.Error(err.Error())
			return err
		}
	}
	return nil
}

// 根据报名计划查询学生列表
func GetRegisterStudentById(ctx context.Context, registerID int64, message string, registerType string, status string, orderBy []string, page int, pageSize int, userID int64, searchType string) ([]Map, int, error) {
	result := make([]Map, 0)
	forceErr, _ := ctx.Value("force-error").(string)

	//构建查询条件
	var clauses []string
	//构建占位符
	var args []interface{}
	var s string

	if searchType == "00" {
		if message != "" {
			clauses = append(clauses, fmt.Sprintf("(%s LIKE $%d OR %s LIKE $%d OR %s LIKE $%d OR %s LIKE $%d )", "u.official_name", len(args)+1,
				"u.email", len(args)+2, "u.id_card_no", len(args)+3, "u.mobile_phone", len(args)+4))
			args = append(args, "%"+message+"%", "%"+message+"%", "%"+message+"%", "%"+message+"%")
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
		s = `
	SELECT u.id ,  u.official_name , u.mobile_phone , u.email , u.gender , u.id_card_no , u.id_card_type , eps.register_time , eps.type , eps.exam_type , COALESCE((SELECT official_name FROM assessuser.t_user WHERE id =eps.reviewer),'') AS reviewer , eps.status, eps.register_id, eps.id
	FROM assessuser.t_user u JOIN assessuser.t_exam_plan_student eps ON eps.student_id =u.id  
`
	} else if searchType == "02" {
		if message != "" {
			clauses = append(clauses, fmt.Sprintf("(%s LIKE $%d OR %s LIKE $%d OR %s LIKE $%d OR %s LIKE $%d )", "u.official_name", len(args)+1,
				"u.email", len(args)+2, "u.id_card_no", len(args)+3, "u.mobile_phone", len(args)+4))
			args = append(args, "%"+message+"%", "%"+message+"%", "%"+message+"%", "%"+message+"%")
		}
		if registerType != "" {
			clauses = append(clauses, fmt.Sprintf("%s = $%d", "eps.type", len(args)+1))
			args = append(args, registerType)
		}
		clauses = append(clauses, fmt.Sprintf("%s = $%d", "eps.status", len(args)+1))
		args = append(args, RegisterStudentStatus.Approved)
		clauses = append(clauses, fmt.Sprintf("eps.register_id = $%d", len(args)+1))
		args = append(args, registerID)
		clauses = append(clauses, fmt.Sprintf("NOT EXISTS (SELECT 1 FROM assessuser.t_exam_student es WHERE es.exam_plan_student_id = eps.id)"))
		s = `SELECT u.id ,  u.official_name , u.mobile_phone , u.email , u.gender , u.id_card_no , u.id_card_type , eps.register_time , eps.type , eps.exam_type , COALESCE((SELECT official_name FROM assessuser.t_user WHERE id =eps.reviewer),'') AS reviewer , eps.status ,eps.register_id, eps.id
	FROM assessuser.t_user u JOIN assessuser.t_exam_plan_student eps ON eps.student_id =u.id  `
	}
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

	sqlxDB := cmn.GetPgxConn()
	rows1, err := sqlxDB.Query(ctx, s, args...)
	defer rows1.Close()
	if err != nil || forceErr == "query" {
		err = fmt.Errorf("query register failed:%v", err)
		z.Error(err.Error())
		return nil, 0, err
	}
	for rows1.Next() {
		M := Map{}
		var student cmn.TUser
		var planStudent cmn.TExamPlanStudent
		var reviewer string
		err = rows1.Scan(&student.ID, &student.OfficialName, &student.MobilePhone, &student.Email, &student.Gender, &student.IDCardNo, &student.IDCardType, &planStudent.RegisterTime, &planStudent.Type, &planStudent.ExamType, &reviewer, &planStudent.Status, &planStudent.RegisterID, &planStudent.ID)
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
	clauses = []string{}
	args = []interface{}{}
	if message != "" {
		clauses = append(clauses, fmt.Sprintf("%s LIKE $%d OR %s LIKE $%d OR %s LIKE $%d OR %s LIKE $%d", "u.official_name", len(args)+1,
			"u.email", len(args)+2, "u.id_card_no", len(args)+3, "u.mobile_phone", len(args)+4))
		args = append(args, "%"+message+"%", "%"+message+"%", "%"+message+"%", "%"+message+"%")
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
	if searchType == "02" {
		clauses = append(clauses, fmt.Sprintf("%s = $%d", "eps.status  ", len(args)+1))
		args = append(args, RegisterStudentStatus.Approved)
		clauses = append(clauses, fmt.Sprintf("NOT EXISTS (SELECT 1 FROM assessuser.t_exam_student es WHERE es.exam_plan_student_id = eps.id)"))
	}
	//获取总数
	s = `SELECT COUNT(*) FROM assessuser.t_user u JOIN assessuser.t_exam_plan_student eps ON eps.student_id =u.id  `
	if len(clauses) > 0 {
		s += "WHERE " + strings.Join(clauses, " AND ")
	}
	rows, err := sqlxDB.Query(ctx, s, args...)
	defer rows.Close()
	if err != nil || forceErr == "query2" {
		err = fmt.Errorf("查询总数失败:%v", err)
		z.Error(err.Error())
		return nil, 0, err
	}

	var total int

	for rows.Next() {
		err = rows.Scan(&total)
		if err != nil || forceErr == "scan2" {
			err = fmt.Errorf("查询总数失败:%v", err)
			z.Error(err.Error())
			return nil, 0, err
		}
	}

	return result, total, nil
}
func UpsertRegister(ctx context.Context, registration *cmn.TRegisterPlan, practiceIds []int64, userID int64, action string, reviewers []int64) error {
	if userID <= 0 {
		err := fmt.Errorf("用户ID不能小于等于0")
		z.Error(err.Error())
		return err
	}
	if !registration.ID.Valid {
		err := AddRegister(ctx, registration, practiceIds, userID)
		if err != nil {
			return err
		}
		return nil
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
	err = UpdateRegister(ctx, registration, practiceIds, userID, action, reviewers)
	if err != nil {
		z.Error(err.Error())
		return err
	}
	//设置新的报名计划定时器
	err = SetRegisterTimers(ctx, registration.ID.Int64)
	if err != nil {
		z.Error(err.Error())
		return err
	}
	return nil
}
func UpdateRegister(ctx context.Context, registration *cmn.TRegisterPlan, practiceIds []int64, userID int64, action string, reviewers []int64) error {

	if userID <= 0 {
		err := fmt.Errorf("用户ID不能小于等于0")
		z.Error(err.Error())
		return err
	}
	if !registration.ID.Valid || registration.ID.Int64 <= 0 {
		err := fmt.Errorf("报名计划ID不合法")
		if err != nil {
			z.Error(err.Error())
			return err
		}
	}
	forceErr, _ := ctx.Value("force-error").(string)
	authority, _ := ctx.Value("authority").(auth_mgt.Authority)
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
	args = append(args, userID)
	args = append(args, authority.AccessibleDomains)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = $%d AND (creator = $%d OR domain_id = ANY( $%d))", "assessuser.t_register_plan", strings.Join(clauses, ", "), idx, idx+1, idx+2)
	z.Sugar().Debugf("update sql:%v", query)
	z.Sugar().Debugf("update args:%v", args)
	sqlxDB := cmn.GetPgxConn()
	tx, err := sqlxDB.Begin(ctx)
	if err != nil || forceErr == "beginTx" {
		err = fmt.Errorf("beginTx called failed:%v", err)
		z.Error(err.Error())
		if forceErr == "beginTx" {
			err := tx.Rollback(ctx)
			if err != nil {
				return err
			}
		}
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
	if err != nil || forceErr == "queryRegisterf" {
		err = fmt.Errorf("updateRegister call failed:%v", err)
		z.Error(err.Error())
		return err
	}

	if practiceIds == nil || len(practiceIds) == 0 || reviewers == nil || len(reviewers) == 0 {
		switch action {
		case "clearr":
			err := UpsertReviewers(ctx, tx, registration.ID.Int64, userID, reviewers)
			if err != nil {
				err = fmt.Errorf("更新审核人失败:%v", err)
				return err
			}
			if len(practiceIds) != 0 && practiceIds != nil {
				err = UpsertRegisterPractice(ctx, tx, registration.ID.Int64, practiceIds, userID)
				if err != nil {
					return err
				}
			}
			return nil
		case "clearp":
			{
				err = UpsertRegisterPractice(ctx, tx, registration.ID.Int64, practiceIds, userID)
				if err != nil {
					return err
				}
				if len(reviewers) != 0 && reviewers != nil {
					err := UpsertReviewers(ctx, tx, registration.ID.Int64, userID, reviewers)
					if err != nil {
						return err
					}
				}
				return nil
			}
		case "clear":
			{
				err := UpsertReviewers(ctx, tx, registration.ID.Int64, userID, reviewers)
				if err != nil {
					return err
				}
				err = UpsertRegisterPractice(ctx, tx, registration.ID.Int64, practiceIds, userID)
				if err != nil {
					return err
				}
				return nil
			}
		default:
			{
				if len(reviewers) != 0 && reviewers != nil {
					err := UpsertReviewers(ctx, tx, registration.ID.Int64, userID, reviewers)
					if err != nil {
						return err
					}
				}
				if len(practiceIds) != 0 && practiceIds != nil {
					err = UpsertRegisterPractice(ctx, tx, registration.ID.Int64, practiceIds, userID)
					if err != nil {
						return err
					}
				}
				return nil
			}
		}
	}
	err = UpsertReviewers(ctx, tx, registration.ID.Int64, userID, reviewers)
	if err != nil {
		return err
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
	authority, _ := ctx.Value("authority").(auth_mgt.Authority)
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
				err = fmt.Errorf("rollback failed:%v", err)
			}
			if err != nil {
				z.Error(err.Error())
				return
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
	INSERT INTO assessuser.t_register_plan (name , start_time , end_time , review_end_time , reviewer_ids , max_number , course , exam_plan_location , creator , updated_by , create_time , update_time , status,domain_id)
	VALUES ($1,$2 ,$3 ,$4 , $5 ,$6 ,$7 ,$8 ,$9 ,$10 ,$11 ,$12 ,$13,$14) RETURNING id
`
	err = tx.QueryRow(ctx, s, registration.Name, registration.StartTime, registration.EndTime, registration.ReviewEndTime, registration.ReviewerIds, registration.MaxNumber, registration.Course, registration.ExamPlanLocation, userID, userID, now, now, RegisterStatus.PendingRelease, authority.Domain.ID.Int64).Scan(&id)
	if forceErr == "query" {
		err = fmt.Errorf("query failed")
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
func UpsertReviewers(ctx context.Context, tx pgx.Tx, registerID int64, userID int64, reviewerIds []int64) error {
	forceErr, _ := ctx.Value("force-error").(string)
	authority, _ := ctx.Value("authority").(auth_mgt.Authority)
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

	//更新当前报名列表下的所有的审核人
	delSQL := `
		UPDATE assessuser.t_register_plan  SET reviewer_ids=$1 ,updated_by =$2 , update_time = $3
		WHERE id =$4 AND (creator = $5 OR domain_id = ANY($6))
`
	_, err := tx.Exec(ctx, delSQL, reviewerIds, userID, now, registerID, userID, authority.AccessibleDomains)
	z.Sugar().Debugf("打印输出一下增加SQL语句:%v", delSQL)
	if err != nil || forceErr == "del" {
		err = fmt.Errorf("更新报名计划下的所有审核人失败:%v", err)
		z.Error(err.Error())
		return err
	}
	return nil
}
func UpsertRegisterPractice(ctx context.Context, tx pgx.Tx, registerID int64, practiceIds []int64, userID int64) error {
	forceErr, _ := ctx.Value("force-error").(string)
	authority, _ := ctx.Value("authority").(auth_mgt.Authority)
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
		WHERE register_id =$4 AND status != $1 AND (creator = $5 OR domain_id= ANY($6))
`
		_, err := tx.Exec(ctx, delSQL, RegisterPracticeStatus.Delete, now, userID, registerID, userID, authority.AccessibleDomains)
		z.Sugar().Debugf("打印输出一下增加SQL语句:%v", delSQL)
		if err != nil || forceErr == "delp" {
			err = fmt.Errorf("删除报名计划下的所有练习失败:%v", err)
			z.Error(err.Error())
			return err
		}

		err = DeleteRegisterPracticeStudent(ctx, tx, userID, []int64{registerID})
		if err != nil {
			return err
		}
		return nil
	}
	//upsert名单
	addRpStr := strings.Repeat("(?,?,?,?,?,?,?,?),", len(practiceIds)-1) + "(?,?,?,?,?,?,?,?)"
	addRpArgs := make([]interface{}, 0, len(practiceIds)*7+1)
	for _, practiceId := range practiceIds {
		addRpArgs = append(addRpArgs,
			registerID, practiceId, userID, userID, now, now, RegisterPracticeStatus.Normal, authority.Domain.ID.Int64,
		)
	}
	addRpArgs = append(addRpArgs, RegisterPracticeStatus.Normal)
	t := `
		INSERT INTO assessuser.t_register_practice 
			(register_id , practice_id , creator , updated_by , create_time , update_time , status,domain_id)
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
	if err != nil || forceErr == "sqlxInup" {
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
	//将新增的练习与报名学生进行关联
	//查询与报名计划相关联的学生
	t = `SELECT eps.student_id FROM assessuser.t_exam_plan_student eps WHERE eps.register_id=$1 AND eps.status NOT IN ($2 ,$3)`
	rows, err := tx.Query(ctx, t, registerID, RegisterStudentStatus.Apply, RegisterStudentStatus.Moved)
	defer rows.Close()
	if err != nil || forceErr == "query1" {
		err = fmt.Errorf("查询与报名计划相关联的学生失败:%v", err)
		z.Error(err.Error())
		return err
	}

	var studentIDs []int64
	for rows.Next() {
		var studentID int64
		err = rows.Scan(&studentID)
		if err != nil || forceErr == "scanrstudent" {
			err = fmt.Errorf("扫描与报名计划相关联的学生失败:%v", err)
			z.Error(err.Error())
			return err
		}
		studentIDs = append(studentIDs, studentID)
	}
	for _, practiceId := range practiceIds {
		err = practice_mgt.UpsertPracticeStudentV2(ctx, practiceId, userID, studentIDs)
		if err != nil {
			z.Error(err.Error())
			return err
		}
	}
	//删除没选择的练习
	var valueExpr []string
	var delRArgs []interface{}
	delRArgs = append(delRArgs, RegisterPracticeStatus.Delete, now, userID, registerID, userID, authority.AccessibleDomains)
	for _, practiceId := range practiceIds {
		valueExpr = append(valueExpr, fmt.Sprintf("($%d::bigint)", len(delRArgs)+1))
		delRArgs = append(delRArgs, practiceId)
	}
	t2 := `
		UPDATE assessuser.t_register_practice t
		SET status = $1, update_time = $2, updated_by = $3
		WHERE t.register_id = $4 AND (t.creator = $5 OR t.domain_id = ANY($6))
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
	if err != nil || forceErr == "query3p" {
		err = fmt.Errorf("删除名单失败:%v", err)
		z.Error(err.Error())
		return err
	}
	err = DeleteRegisterPracticeStudent(ctx, tx, userID, []int64{registerID})
	if err != nil {
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
	sqlxDB := cmn.GetPgxConn()
	row1, err := sqlxDB.Query(ctx, s, RegisterStudentStatus.Apply, RegisterStudentStatus.Moved, registerID)
	defer row1.Close()
	if err != nil || forceErr == "queryRow" {
		err = fmt.Errorf("查询报名计划失败:%v", err)
		z.Error(err.Error())
		return nil, nil, nil, -1, err
	}

	var register cmn.TRegisterPlan
	var currentNumber int64
	for row1.Next() {
		err = row1.Scan(&register.ID, &register.Name, &register.Course, &register.ReviewEndTime, &register.MaxNumber, &register.StartTime, &register.EndTime, &register.ReviewerIds, &register.ExamPlanLocation, &register.Status, &currentNumber)
		if err != nil || forceErr == "query" {
			err = fmt.Errorf("查询报名计划失败:%v", err)
			z.Error(err.Error())
			return nil, nil, nil, -1, err
		}
	}
	//查询报名计划相关练习
	s = `SELECT r.practice_id ,p.name,p.type FROM assessuser.t_register_practice r JOIN assessuser.t_practice p ON p.id=r.practice_id WHERE r.register_id =$1 AND r.status =$2`
	rows2, err := sqlxDB.Query(ctx, s, registerID, RegisterPracticeStatus.Normal)
	defer rows2.Close()
	if err != nil || forceErr == "query2" {
		err = fmt.Errorf("查询报名计划下的练习失败:%v", err)
		z.Error(err.Error())
		return nil, nil, nil, -1, err
	}
	var practices []cmn.TPractice
	for rows2.Next() {
		var practice cmn.TPractice
		err := rows2.Scan(&practice.ID, &practice.Name, &practice.Type)
		if err != nil || forceErr == "scann" {
			err = fmt.Errorf("扫描报名计划下的练习失败:%v", err)
			z.Error(err.Error())
			return nil, nil, nil, -1, err
		}
		practices = append(practices, practice)
	}
	//查询报名计划审查人名字
	s = `SELECT id,official_name FROM assessuser.t_user WHERE id IN (SELECT unnest(r.reviewer_ids) FROM assessuser.t_register_plan r 
    WHERE r.id = $1)`
	rows3, err := sqlxDB.Query(ctx, s, registerID)
	defer rows3.Close()
	if err != nil || forceErr == "query3l" {
		err = fmt.Errorf("查询报名计划下的审查人失败:%v", err)
		z.Error(err.Error())
		return nil, nil, nil, -1, err
	}
	var reviewers []Reviewer
	for rows3.Next() {
		var reviewer Reviewer
		err := rows3.Scan(&reviewer.ID, &reviewer.OfficialName)
		if err != nil || forceErr == "scanl" {
			err = fmt.Errorf("扫描报名计划下的审查人失败:%v", err)
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
	authority := ctx.Value("authority").(auth_mgt.Authority)
	now := time.Now().UnixMilli()
	//获取每个报名计划的信息
	Rs, err := LoadRegisterByIds(ctx, registerIDs)
	if err != nil {

		return err
	}
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
	sqlxDB := cmn.GetPgxConn()
	tx, err := sqlxDB.Begin(ctx)
	if err != nil || forceErr == "beginTx" {
		err = fmt.Errorf("beginTx called failed:%v", err)
		z.Error(err.Error())
		if forceErr == "beginTx" {
			err := tx.Rollback(ctx)
			if err != nil {
				return err
			}
		}
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
			return
		}
	}()

	if status == RegisterStatus.Released {
		for _, register := range Rs {
			s := `
			UPDATE assessuser.t_register_plan r  SET status = $1,update_time = $2, updated_by = $3  WHERE id = $4 AND (creator = $5 AND domain_id = ANY($6))
`
			_, err = tx.Exec(ctx, s, RegisterStatus.Released, now, userID, register.ID, userID, authority.AccessibleDomains)
			if err != nil || forceErr == "operate" {
				err = fmt.Errorf("更新报名计划状态失败:%v", err)
				z.Error(err.Error())
				return err
			}
		}

		//设置新的报名计划定时器
		for _, register := range registerIDs {
			err = SetRegisterTimers(ctx, register)
			if err != nil || forceErr == "setTimers" {
				err = fmt.Errorf("设置报名计划定时器失败:%v", err)
				return err
			}

		}

		return nil

	} else if status == RegisterStatus.Disabled {
		for _, register := range Rs {
			s := `
			UPDATE assessuser.t_register_plan  SET status = $1,update_time = $2, updated_by = $3  WHERE id = $4 AND (creator = $5 AND domain_id = ANY($6))
`
			_, err = tx.Exec(ctx, s, RegisterStatus.Disabled, now, userID, register.ID, userID, authority.AccessibleDomains)
			if err != nil || forceErr == "operate1" {
				err = fmt.Errorf("更新报名计划状态失败:%v", err)
				z.Error(err.Error())
				return err
			}
			//让与其相关联的练习关系失效
			s = `
			UPDATE assessuser.t_register_practice  SET status = $1,update_time = $2, updated_by = $3  WHERE register_id = $4 AND (creator = $5 AND domain_id = ANY($6))
`
			_, err = tx.Exec(ctx, s, RegisterPracticeStatus.Delete, now, userID, register.ID, authority.AccessibleDomains)
			if err != nil || forceErr == "operate2" {
				err = fmt.Errorf("更新报名计划状态失败:%v", err)
				z.Error(err.Error())
				return err
			}
			//让与其相关联的学生也失效
			s = `
			UPDATE assessuser.t_exam_plan_student   SET status = $1,update_time = $2, updated_by = $3  WHERE register_id = $4
`
			_, err = tx.Exec(ctx, s, RegisterStudentStatus.Deleted, now, userID, register.ID)
			if err != nil || forceErr == "operate3" {
				err = fmt.Errorf("更新报名计划状态失败:%v", err)
				z.Error(err.Error())
				return err
			}
		}
		err = DeleteRegisterPracticeStudent(ctx, tx, userID, registerIDs)
		if err != nil {
			return err
		}
		//取消报名计划定时器
		for _, registerId := range registerIDs {
			err = CancelRegisterTimers(ctx, registerId)
			if err != nil || forceErr == "cancelRegisterTimers1" {
				err = fmt.Errorf("取消报名计划定时器失败:%v", err)
				return err
			}
		}
		return nil
	} else if status == RegisterStatus.Deleted {
		registerIsUsed := false
		var invalidName []string
		var row1 pgx.Rows

		for _, register := range Rs {
			s := `
			SELECT EXISTS(SELECT 1 FROM assessuser.t_exam_plan_student eps WHERE eps.register_id =$1 )
`
			row1, err = tx.Query(ctx, s, register.ID)
			if err != nil || forceErr == "queryIsUsed" {
				err = fmt.Errorf("查询报名计划是否被使用失败:%v", err)
				z.Error(err.Error())
				row1.Close()
				return err
			}

			for row1.Next() {
				err := row1.Scan(&registerIsUsed)
				if err != nil || forceErr == "scanIsUsed" {
					err = fmt.Errorf("扫描查询报名计划是否被使用失败:%v", err)
					z.Error(err.Error())
					row1.Close()
					return err
				}
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
		UPDATE assessuser.t_register_plan SET status = $1,update_time = $2, updated_by = $3  WHERE id = ANY($4)AND (creator = $5 AND domain_id = ANY($6))
`
		_, err = tx.Exec(ctx, s, RegisterStatus.Deleted, now, userID, registerIDs, userID, authority.AccessibleDomains)
		if err != nil || forceErr == "operateRegister" {
			err = fmt.Errorf("删除报名计划失败:%v", err)
			z.Error(err.Error())
			return err
		}
		//删除与其关联的练习
		s = `
		UPDATE  assessuser.t_register_practice SET status = $1,update_time = $2, updated_by = $3  WHERE register_id = ANY($4)AND (creator = $5 AND domain_id = ANY($6))
`
		_, err = tx.Exec(ctx, s, RegisterPracticeStatus.Delete, now, userID, registerIDs, userID, authority.AccessibleDomains)
		if err != nil || forceErr == "tx.UpdateRegisterPractice" {
			err = fmt.Errorf("更新关联的练习失败:%v", err)
			z.Error(err.Error())
			return err
		}
		err = DeleteRegisterPracticeStudent(ctx, tx, userID, registerIDs)
		if err != nil {
			return err
		}
		//取消报名计划定时器
		for _, registerId := range registerIDs {
			err = CancelRegisterTimers(ctx, registerId)
			if err != nil || forceErr == "cancelTimers" {
				err = fmt.Errorf("取消报名计划定时器失败: %v", err)
				return err
			}
		}
		return nil
	}
	err = fmt.Errorf("异常状态， 无法操作")
	return err
}

// 删除报名计划相关联的练习下的学生
func DeleteRegisterPracticeStudent(ctx context.Context, tx pgx.Tx, userID int64, registerIDs []int64) error {
	forceErr := ""
	val := ctx.Value("force-error")
	if val != nil {
		forceErr = val.(string)
	}
	//获取已删除的练习ID
	delSQL := `SELECT rp.practice_id FROM assessuser.t_register_practice rp WHERE rp.status =$1 AND rp.register_id = ANY ($2) `
	rows, err := tx.Query(ctx, delSQL, RegisterPracticeStatus.Delete, registerIDs)
	defer rows.Close()
	if err != nil || forceErr == "query" {
		err = fmt.Errorf("查询已删除的练习ID失败:%v", err)
		z.Error(err.Error())
		return err
	}

	var deletePracticeIDs []int64
	for rows.Next() {
		var practiceId int64
		err = rows.Scan(&practiceId)
		if err != nil || forceErr == "scand" {
			err = fmt.Errorf("扫描已删除的练习ID失败:%v", err)
			z.Error(err.Error())
			return err
		}
		deletePracticeIDs = append(deletePracticeIDs, practiceId)
	}
	//删除与练习相关联的学生
	for _, practiceID := range deletePracticeIDs {
		err = practice_mgt.UpsertPracticeStudentV2(ctx, practiceID, userID, []int64{})
		if err != nil {
			z.Error(err.Error())
			return err
		}
	}
	return nil
}
func LoadRegisterByIds(ctx context.Context, registerIDs []int64) (registers []*cmn.TRegisterPlan, err error) {
	forceErr := ctx.Value("force-error")
	s := `
	SELECT r.id, r.name , r.course , r.review_end_time , r. max_number , r.start_time , r.end_time , r.reviewer_ids , r.exam_plan_location ,r.status
	FROM t_register_plan r WHERE r.id = ANY($1) AND r.status != $2
`
	sqlxDB := cmn.GetPgxConn()
	rows, err := sqlxDB.Query(ctx, s, registerIDs, RegisterStatus.Deleted)
	defer func() {
		defer rows.Close()
	}()
	if err != nil || forceErr == "queryls" {
		err = fmt.Errorf("查询报名计划失败:%v", err)
		z.Error(err.Error())
		return nil, err
	}
	for rows.Next() {
		var register cmn.TRegisterPlan
		err = rows.Scan(&register.ID, &register.Name, &register.Course, &register.ReviewEndTime, &register.MaxNumber, &register.StartTime, &register.EndTime, &register.ReviewerIds, &register.ExamPlanLocation, &register.Status)
		if err != nil || forceErr == "lsScan" {
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
	register, practices, _, _, err := LoadRegisterById(ctx, RegisterID)
	if err != nil {
		return err
	}
	if register.Status.String != RegisterStatus.Released && register.Status.String != RegisterStatus.Ending {
		err := fmt.Errorf("当前报名计划状态为：%v，不能进行操作", register.Status.String)
		z.Error(err.Error())
		return err
	}
	//获取学生状态
	var studentStatus = ""
	if status == RegisterStudentStatus.Moved {
		studentStatus = RegisterStudentStatus.Moved
	} else {
		students, err := LoadRegisterStudentStatusByIds(ctx, ids, RegisterID)
		if err != nil {
			return err
		}
		for _, student := range students {
			if studentStatus == "" {
				studentStatus = student.Status.String
			}
			if student.Status.String != studentStatus {
				err := fmt.Errorf("此时要批量操作的学生状态不一，无法进行批量操作 %v", students)
				z.Error(err.Error())
				return err
			}
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
		if err != nil || forceErr == "opQuery" {
			err = fmt.Errorf("设置迁移学生状态失败:%v", err)
			z.Error(err.Error())
			return err
		}

	} else {
		//批量通过或不通过学生审核
		sqlxDB := cmn.GetPgxConn()
		tx, err = sqlxDB.Begin(ctx)
		// 添加数据库连接池检查
		stat := sqlxDB.Stat()
		// 检查是否有空闲连接
		if stat.IdleConns() > 0 {
			z.Error(fmt.Sprintf("数据库连接池状态: 空闲连接数 %d, 已使用连接数 %d, 总连接数 %d",
				stat.IdleConns(), stat.AcquiredConns(), stat.TotalConns()))
		}
		if err != nil || forceErr == "begin" {
			if forceErr == "begin" {
				err := tx.Rollback(ctx)
				if err != nil {
					return err
				}
			}
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
		if err != nil || forceErr == "upQuery" {
			err = fmt.Errorf("更新学生状态失败:%v", err)
			z.Error(err.Error())
			return err
		}
		if status == RegisterStudentStatus.Approved {
			//把学生关联到练习当中
			for _, practiceId := range practices {
				err = practice_mgt.UpsertPracticeStudentV2(ctx, practiceId.ID.Int64, userID, ids)
				if err != nil {
					return err
				}
			}
		} else if status == RegisterStudentStatus.Rejected {
			//把学生从练习当中移除
			var rows pgx.Rows
			// 添加数据库连接池检查
			stat := sqlxDB.Stat()
			// 检查是否有空闲连接
			if stat.IdleConns() > 0 {
				z.Error(fmt.Sprintf("数据库连接池状态: 空闲连接数 %d, 已使用连接数 %d, 总连接数 %d",
					stat.IdleConns(), stat.AcquiredConns(), stat.TotalConns()))
			}
			for _, practiceId := range practices {
				//获取除当前选择的学生之外的与练习绑定的学生
				s = `SELECT student_id FROM assessuser.t_practice_student WHERE status =$1 AND practice_id =$2 AND student_id NOT IN ($3)`
				rows, err = tx.Query(ctx, s, "00", practiceId.ID.Int64, pq.Array(ids))
				if err != nil || forceErr == "querylrs" {
					err = fmt.Errorf("查询除当前选择学生的其他学生失败:%v", err)
					z.Error(err.Error())
					return err
				}
				// 确保在函数退出前关闭资源
				var otherStudents []int64
				for rows.Next() {
					var studentId int64
					err = rows.Scan(&studentId)
					if err != nil || forceErr == "lsScanps" {
						z.Error(err.Error())
						return err
					}
					otherStudents = append(otherStudents, studentId)
				}
				err = practice_mgt.UpsertPracticeStudentV2(ctx, practiceId.ID.Int64, userID, otherStudents)
				if err != nil {
					return err
				}
			}

		}
		// 添加数据库连接池检查
		stat = sqlxDB.Stat()
		// 检查是否有空闲连接
		if stat.IdleConns() > 0 {
			z.Error(fmt.Sprintf("数据库连接池状态: 空闲连接数 %d, 已使用连接数 %d, 总连接数 %d",
				stat.IdleConns(), stat.AcquiredConns(), stat.TotalConns()))
		}
		return nil
	}
	return nil
}
func LoadRegisterStudentStatusByIds(ctx context.Context, ids []int64, registerID int64) (students []cmn.TExamPlanStudent, err error) {
	forceErr, _ := ctx.Value("force-error").(string)
	s := `
	SELECT  eps.id ,eps.student_id , eps.status
	FROM t_exam_plan_student eps WHERE eps.student_id = ANY($1) AND  eps.register_id = $2
`
	sqlxDB := cmn.GetPgxConn()
	rows, err := sqlxDB.Query(ctx, s, ids, registerID)
	defer rows.Close()
	if err != nil || forceErr == "querylrs" {
		err = fmt.Errorf("查询学生状态失败:%v", err)
		z.Error(err.Error())
		return nil, err
	}
	for rows.Next() {
		var student cmn.TExamPlanStudent
		err = rows.Scan(&student.ID, &student.StudentID, &student.Status)
		if err != nil || forceErr == "lrsScan" {
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
		return err
	}
	if register.Status.String != RegisterStatus.Released {
		err := fmt.Errorf("当前报名计划状态为：%v，不能导入学生", register.Status.String)
		z.Error(err.Error())
		return err
	}
	if (currentNumber+int64(len(studentIDs)) > register.MaxNumber.Int64) && register.MaxNumber.Int64 > 0 {
		err := fmt.Errorf("导入学生人数超出报名计划可容纳人数, 剩余人数为: %v", register.MaxNumber.Int64-currentNumber)
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
	//批量将报名学生和报名计划的练习进行绑定
	studentIds := []int64{}
	for _, student := range studentIDs {
		studentIds = append(studentIds, student.StudentID)
	}
	//查询报名计划下的所有练习id
	_, practices, _, _, err := LoadRegisterById(ctx, registerID)
	if err != nil || forceErr == "loadRegister" {
		if forceErr == "loadRegister" {
			err = fmt.Errorf("查询报名计划下的所有练习失败:%v", err)
		}
		return err
	}
	for _, practice := range practices {
		err = practice_mgt.UpsertPracticeStudentV2(ctx, practice.ID.Int64, userID, studentIds)
		if err != nil {
			return err
		}
	}

	return nil
}

// 移动学生
func MoveStudent(ctx context.Context, fromRegisterID int64, toRegisterID int64, students []registerStudentType, status string, userID int64) error {
	if fromRegisterID <= 0 || toRegisterID <= 0 {
		err := fmt.Errorf("报名计划ID错误")
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
	register, practices, _, currentNumber, err := LoadRegisterById(ctx, toRegisterID)
	if err != nil {
		return err
	}
	var registerStatus string
	registerStatus = register.Status.String
	if registerStatus != RegisterStatus.Released {
		err = fmt.Errorf("目标报名计划状态为：%v，不能移动学生", registerStatus)
		z.Error(err.Error())
		return err
	}
	if (register.MaxNumber.Int64 < int64(len(students))+currentNumber) && register.MaxNumber.Int64 != 0 {
		err = fmt.Errorf("目标报名计划可容纳人数不足，无法进行移动 ,剩余人数为: %v", register.MaxNumber.Int64-currentNumber)
		z.Error(err.Error())
		return err
	}
	sqlxDB := cmn.GetPgxConn()
	tx, err := sqlxDB.Begin(ctx)
	if err != nil || forceErr == "begin" {
		err = fmt.Errorf("开启事务失败:%v", err)
		z.Error(err.Error())
		if forceErr == "begin" {
			err := tx.Rollback(ctx)
			if err != nil {
				return err
			}
		}
		return err
	}

	defer func() {
		if forceErr == "rollbackm" {
			err = fmt.Errorf("触发回滚")
		}
		if err != nil {
			// 操作失败回滚
			err = tx.Rollback(ctx)
			if forceErr == "rollbackm" {
				err = fmt.Errorf("回滚失败")
			}
			if err != nil {
				z.Error(err.Error())
				return
			}
		} else {
			// 无错误则提交
			err = tx.Commit(ctx)
			if forceErr == "commitm" {
				err = fmt.Errorf("触发commit")
			}
			if err != nil {
				z.Error(err.Error())
			}
		}
	}()
	var planStudents []cmn.TExamPlanStudent // 从 students 切片中提取 StudentID
	var studentIDs []int64
	for _, student := range students {
		studentIDs = append(studentIDs, student.StudentID)
	}

	//获取原本学生的状态
	s := `SELECT * FROM  assessuser.t_exam_plan_student WHERE register_id =$1 AND student_id = ANY ($2) `
	var rows pgx.Rows
	rows, err = tx.Query(ctx, s, fromRegisterID, studentIDs)

	if err != nil || forceErr == "queryMove" {
		err = fmt.Errorf("获取学生状态失败:%v", err)
		z.Error(err.Error())
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var student cmn.TExamPlanStudent
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
		if err != nil || forceErr == "scanMove" {
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
	//批量将报名学生和报名计划的练习进行绑定
	studentIds := []int64{}
	for _, student := range students {
		studentIds = append(studentIds, student.StudentID)
	}
	for _, practice := range practices {
		err = practice_mgt.UpsertPracticeStudentV2(ctx, practice.ID.Int64, userID, studentIds)
		if err != nil {
			return err
		}
	}
	//修改原来的报名计划的学生状态为已迁移
	//调用更新原本学生状态方法
	err = OperateRegisterStudentStatus(ctx, tx, studentIDs, status, userID, fromRegisterID, "")
	if err != nil {
		return err
	}
	//删除原来的报名计划下绑定的练习
	err = UpsertRegisterPractice(ctx, tx, fromRegisterID, nil, userID)
	if err != nil {
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
	authority := ctx.Value("authority").(auth_mgt.Authority)
	//获取报名计划的审查者id
	register, _, _, _, err := LoadRegisterById(ctx, registerID)
	if err != nil {

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
	clauses = append(clauses, fmt.Sprintf("(%s = $%d OR %s = ANY($%d))", "u.creator", len(args)+1, "u.domain_id", len(args)+2))
	args = append(args, userID, authority.AccessibleDomains)
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
	sqlxDB := cmn.GetPgxConn()
	rows1, err := sqlxDB.Query(ctx, s, args...)
	defer rows1.Close()
	if err != nil || forceErr == "queryr" {
		err = fmt.Errorf("search register failed:%v", err)
		z.Error(err.Error())
		return nil, 0, err
	}

	for rows1.Next() {
		M := Map{}
		var reviewer Reviewer
		err = rows1.Scan(&reviewer.ID, &reviewer.OfficialName, &reviewer.Gender, &reviewer.MobilePhone, &reviewer.IDCardNo, &reviewer.IDCardType)
		if err != nil || forceErr == "row scan" {
			err = fmt.Errorf("row scan failed:%v", err)
			z.Error(err.Error())
			return nil, 0, err

		}
		M["reviewer"] = reviewer
		result = append(result, M)
	}
	clauses = []string{}
	args = []interface{}{}
	if name != "" {
		clauses = append(clauses, fmt.Sprintf("%s LIKE $%d", "u.official_name", len(args)+1))
		args = append(args, "%"+name+"%")
	}
	clauses = append(clauses, fmt.Sprintf("%s =ANY ($%d)", "u.id", len(args)+1))
	args = append(args, register.ReviewerIds)
	clauses = append(clauses, fmt.Sprintf("(%s = $%d OR %s = ANY($%d))", "u.creator", len(args)+1, "u.domain_id", len(args)+2))
	args = append(args, userID, authority.AccessibleDomains)
	//查询总数
	s = ` SELECT COUNT(*) FROM assessuser.t_user u `
	if len(clauses) > 0 {
		s += " WHERE " + strings.Join(clauses, " AND ")
	}
	rows, err := sqlxDB.Query(ctx, s, args...)
	defer rows.Close()
	if err != nil || forceErr == "queryx" {
		err = fmt.Errorf("查询数据失败:%v", err)
		z.Error(err.Error())
		return nil, 0, err
	}
	var total int

	for rows.Next() {
		err = rows.Scan(&total)
		if err != nil || forceErr == "lScan" {
			err = fmt.Errorf("解析数据失败:%v", err)
			z.Error(err.Error())
			return nil, 0, err
		}
	}

	return result, total, nil

}
