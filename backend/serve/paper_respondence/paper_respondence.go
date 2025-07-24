package paper_respondence

//annotation:template-service
//author:{"name":"OuYangHaoBin","tel":"13712562121", "email":"1242968386@qq.com"}

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"io"
	"strconv"
	"strings"
	"time"
	"w2w.io/cmn"
)

const (
	StartTimeNotArrived  = iota + 1 //未到达考试开始时间
	EndTimeArrived                  //考试结束时间已经到达
	ExamSubmitted                   //考试已经提交
	LastEntryTimeArrived            // 最迟进入时间已经到达
	ExamCanBeEnter                  //考试无论什么条件都能进入

	TIMEOUT = 5 * time.Second

	AiCorrectMode = "02"

	TestSign = "test"
)

var z *zap.Logger

func init() {
	//Setup package scope variables, just like logger, db connector, configure parameters, etc.
	cmn.PackageStarters = append(cmn.PackageStarters, func() {
		z = cmn.GetLogger()
		z.Info("message zLogger settled")
	})
}
func Enroll(author string) {
	z.Info("message.Enroll called")
	var developer *cmn.ModuleAuthor
	if author != "" {
		var d cmn.ModuleAuthor
		err := json.Unmarshal([]byte(author), &d)
		if err != nil {
			z.Error(err.Error())
			return
		}
		developer = &d
	}

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: StudentAnswer,

		Path: "/respondent",
		Name: "respondent",

		Developer: developer,
		WhiteList: false,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: InitForExam,

		Path: "/respondent/init",
		Name: "respondent_init",

		Developer: developer,
		WhiteList: false,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})
	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: Submit,

		Path: "/respondent/submit",
		Name: "respondent_Submit",

		Developer: developer,
		WhiteList: false,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})
}

// --------------------http接口暴露函数区域

// StudentAnswer 保存或者更新作答
func StudentAnswer(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	method := strings.ToLower(q.R.Method)
	switch method {
	case "post":
		var buf []byte
		buf, q.Err = io.ReadAll(q.R.Body)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer func() {
			q.Err = q.R.Body.Close()
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}()

		if len(buf) == 0 {
			q.Err = fmt.Errorf("Call /api/respondent with  empty body")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		//获取请求的结构体
		var qry cmn.ReqProto
		q.Err = json.Unmarshal(buf, &qry)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		studentId := q.SysUser.ID

		//获取需要保存到数据库的数据
		var u SaveOrUpdateStudentAnswerReq
		q.Err = json.Unmarshal(qry.Data, &u)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		u.StudentId = studentId.Int64

		//参数校验
		q.Err = cmn.Validate(u)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		//执行数据库操作
		sqlxDB := cmn.GetDbConn()

		// 直接一条SQL搞定插入或更新，如果是禁止作答的状态，说明已经提交，不能改了
		sql := `
	INSERT INTO t_student_answers 
		(type, examinee_id, practice_submission_id, question_id, answer, answer_score, creator, create_time, updated_by, update_time, addi, status,answer_attachments_path)
	VALUES
		($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	ON CONFLICT (examinee_id, question_id)
	DO UPDATE SET
		type = EXCLUDED.type,
		answer = EXCLUDED.answer,
		answer_attachments_path = EXCLUDED.answer_attachments_path,
		updated_by = EXCLUDED.updated_by,
		update_time = EXCLUDED.update_time
	WHERE t_student_answers.status <> $14
	RETURNING id,creator,updated_by
	`
		var stmt *sqlx.Stmt
		stmt, q.Err = sqlxDB.Preparex(sql)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		defer func() {
			q.Err = stmt.Close()
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}()

		dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
		defer cancel()
		var result cmn.TStudentAnswers
		result, q.Err = insertOrUpdateAnswer(dmlCtx, u, stmt)
		if q.Err != nil {
			q.RespErr()
			return
		}

		buf, q.Err = cmn.MarshalJSON(&result)
		if q.Err != nil {
			q.RespErr()
			return
		}

		q.Msg.Data = buf
		q.Resp()
	case "get":
		//获取题目的id
		qd := q.R.URL.Query().Get("question_id")
		if qd == "" {
			err := fmt.Errorf("请提供题目的id")
			z.Error(err.Error())
			q.Err = err
			q.RespErr()
			return
		}
		//转化题目id为int64
		var questionId int64
		questionId, q.Err = strconv.ParseInt(qd, 10, 64)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		//获取考生id以及练习id并判断哪个是有值的，但是两个不能同时有值
		ed := q.R.URL.Query().Get("examinee_id")

		pd := q.R.URL.Query().Get("practice_submission_id")

		if ed == "" && pd == "" {
			err := fmt.Errorf("考生id和练习id不能同时为空")
			z.Error(err.Error())
			q.Err = err
			q.RespErr()
			return
		}

		dmlReq := GetStudentAnswerReq{
			QuestionID: questionId,
		}

		//查看哪个不为空
		if ed != "" {
			dmlReq.ExamineeID, q.Err = strconv.ParseInt(ed, 10, 64)
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		} else {
			dmlReq.PracticeSubmissionId, q.Err = strconv.ParseInt(pd, 10, 64)
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}

		//执行数据库操作
		sqlxDB := cmn.GetDbConn()
		selectSql := `SELECT id, type, examinee_id, question_id, answer, marker, creator, create_time, updated_by, update_time, status FROM t_student_answers WHERE examinee_id =$1 AND question_id =$2`
		var stmt *sqlx.Stmt
		stmt, q.Err = sqlxDB.Preparex(selectSql)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		defer func() {
			q.Err = stmt.Close()
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}()

		var result cmn.TStudentAnswers
		dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
		defer cancel()
		result, q.Err = getAnswerByExamineeIDAndQuestionID(dmlCtx, dmlReq, stmt)
		if q.Err != nil {
			q.RespErr()
			return
		}
		var buf []byte
		buf, q.Err = cmn.MarshalJSON(&result)

		q.Msg.RowCount = 1
		q.Msg.Data = buf
		q.Resp()
	}

}

// InitForExam 考试初始化
func InitForExam(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	method := strings.ToLower(q.R.Method)
	if method != "post" {
		q.Err = fmt.Errorf("please call /api/upLogin with  http POST method")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	var buf []byte
	buf, q.Err = io.ReadAll(q.R.Body)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	defer func() {
		q.Err = q.R.Body.Close()
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
	}()

	if len(buf) == 0 {
		q.Err = fmt.Errorf("Call /api/respondent with  empty body")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	//获取请求的结构体
	var qry cmn.ReqProto
	q.Err = json.Unmarshal(buf, &qry)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	var u SaveBeginTimeReq

	q.Err = json.Unmarshal(qry.Data, &u)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	studentId := q.SysUser.ID.Int64
	u.StudentId = studentId

	//参数校验
	q.Err = cmn.Validate(u)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	//创建事务
	dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
	defer cancel()
	db := cmn.GetDbConn()
	tx, err := db.BeginTx(dmlCtx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		q.Err = err
		z.Error(err.Error())
		q.RespErr()
		return
	}
	defer func() {
		//如果不是tx done错误就返回给前端
		q.Err = tx.Rollback()
		if q.Err != nil && !errors.Is(q.Err, sql.ErrTxDone) {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
	}()

	switch u.Type {
	case ExamType:
		if u.ExamId <= 0 || u.ExamineeID <= 0 {
			q.Err = fmt.Errorf("当前是考试，请输入大于0的考试id大于0的考生id")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		//保存开始时间
		q.Err = saveBeginTimeForExam(dmlCtx, u, tx)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		//TODO 获取考试信息

		//TODO 获取考卷
		q.Err = nil
		q.Msg.Status = 0
		q.Msg.Msg = "success"
		q.Resp()
	case PracticeType:

		//TODO 生成试卷并获取试卷数据
		//如果是第一次进入，就要保存练习开始时间
		if err := SaveBeginTimeForPracticeWithTx(dmlCtx, tx, u); err != nil {
			q.Err = err
			q.RespErr()
			return
		}

		//更新最近一次进入练习的时间
		q.Err = UpdateLastStartTime(ctx, u.PracticeSubmissionID, tx)
		if q.Err != nil {
			q.RespErr()
			return
		}
	default:
		q.Err = fmt.Errorf("unknown respondence type: %s", u.Type)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	//提交事务
	if err := tx.Commit(); err != nil {
		q.Err = err
		z.Error(err.Error())
		q.RespErr()
		return
	}

}

// CheckExamStatus 将查看当前考试的状态函数暴露http接口
func CheckExamStatus(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	method := strings.ToLower(q.R.Method)
	if method != "get" {
		q.Err = fmt.Errorf("please call /api/upLogin with  http get method")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	r := q.R

	//获取考试id
	examId := r.URL.Query().Get("exam_id")
	if examId == "" {
		err := fmt.Errorf("exam_id is required")
		q.Err = err
		z.Error(err.Error())
		q.RespErr()
		return
	}

	examIdInt, err := strconv.ParseInt(examId, 10, 64)
	if err != nil {
		q.Err = err
		z.Error(err.Error())
		q.RespErr()
		return
	}
	//获取考生id
	examineeId := r.URL.Query().Get("examinee_id")
	if examineeId == "" {
		err := fmt.Errorf("examinee_id is required")
		z.Error(err.Error())
		q.Err = err
		q.RespErr()
		return
	}

	examineeIdInt, err := strconv.ParseInt(examineeId, 10, 64)
	if err != nil {
		q.Err = err
		z.Error(err.Error())
		q.RespErr()
		return
	}

	req := CheckExamStatusReq{
		ExamineeID: examineeIdInt,
		ExamId:     examIdInt,
		StudentId:  q.SysUser.ID.Int64,
	}
	checkCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
	defer cancel()
	var result int
	result, q.Err = checkExamStatus(checkCtx, req)
	if q.Err != nil {
		q.RespErr()
		return
	}

	data, err := json.Marshal(result)
	if err != nil {
		z.Error(err.Error())
		q.Err = err
		q.RespErr()
		return
	}
	q.Msg.Status = 0
	q.Msg.Data = data
	q.Resp()
}

// Submit 提交作答
func Submit(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	method := strings.ToLower(q.R.Method)
	if method != "post" {
		q.Err = fmt.Errorf("please call /api/upLogin with  http POST method")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	var buf []byte
	buf, q.Err = io.ReadAll(q.R.Body)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	defer func() {
		q.Err = q.R.Body.Close()
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
	}()

	if len(buf) == 0 {
		q.Err = fmt.Errorf("Call /api/respondent with  empty body")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	//获取请求的结构体
	var qry cmn.ReqProto
	q.Err = json.Unmarshal(buf, &qry)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	var u SubmitReq

	q.Err = json.Unmarshal(qry.Data, &u)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	studentId := q.SysUser.ID.Int64
	u.StudentId = studentId

	//参数校验
	q.Err = cmn.Validate(u)
	if q.Err != nil {
		q.RespErr()
		return
	}
	dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
	defer cancel()
	switch u.Type {
	case ExamType:
		if u.ExamId <= 0 || u.ExamineeID <= 0 {
			q.Err = fmt.Errorf("当前是考试，请输入大于0的考试id大于0的考生id")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		q.Err = submitForExam(dmlCtx, u)
		if q.Err != nil {
			q.RespErr()
			return
		}
		q.Msg.Msg = "success"
		q.Msg.Status = 0
		q.Resp()
	case PracticeType:
		if u.PracticeSubmissionID <= 0 {
			q.Err = fmt.Errorf("当前是练习，请输入大于0的PracticeSubmissionID")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		q.Err = submitForPractice(dmlCtx, u)
		if q.Err != nil {
			q.RespErr()
			return
		}
	default:
		q.Err = fmt.Errorf("unknown student answer type: %s", u.Type)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
}

//--------------------由于逻辑太长，将一些长逻辑封装到此处，http接口暴露区域直接调用这里的

func saveBeginTimeForExam(ctx context.Context, req SaveBeginTimeReq, tx *sql.Tx) error {

	//获取学生进行考试的开始时间、结束时间
	se, err := checkStartTimeAndEndTimeAndStatus(ctx, tx, req.ExamineeID)
	if err != nil {
		return err
	}

	//先查看结束时间是否为空，不为空，说明已经提交
	if se.EndTime.Valid {
		err := fmt.Errorf("examineeId为d的用户已经提交考试", req.ExamineeID)
		z.Error(err.Error())
		return err
	}

	//查看是否有开始时间，如果有，说明已经开始过了，不需要任何操作
	if se.StartTime.Valid {
		info := fmt.Sprintf("examineeId为%d的用户已经开始过考试了，不需要任何操作", req.ExamineeID)
		z.Info(info)
		return nil
	}

	//查看是否有监考管理员提供的进入权限
	if se.Status.String == CanBeEnterStatus {
		err := saveStudentBeginTime(ctx, tx, req)
		if err != nil {
			return err
		}
	} else {
		//TODO 查看学生的考试场次信息，并查看是否有超过最迟进入考试时间
		err := saveStudentBeginTime(ctx, tx, req)
		if err != nil {
			return err
		}
	}

	return nil
}

func submitForExam(ctx context.Context, req SubmitReq) error {
	//TODO 查看学生考试场次信息
	//执行数据库操作
	sqlxDB := cmn.GetDbConn()

	//开启事务
	tx, err := sqlxDB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		z.Error(err.Error())
		return err
	}
	defer tx.Rollback()
	updateId, err := submitExamInDataBase(ctx, tx, req)
	if err != nil {
		return err
	}
	z.Info("update success", zap.Int64("updateId", updateId))

	if err := tx.Commit(); err != nil {
		z.Error(err.Error())
		return err
	}
	return nil
}

// submitForPractice 提交练习
func submitForPractice(ctx context.Context, req SubmitReq) error {
	//执行数据库操作
	sqlxDB := cmn.GetDbConn()

	//开启事务
	tx, err := sqlxDB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		z.Error(err.Error())
		return err
	}
	defer tx.Rollback()
	err = checkPracticeIfSubmitted(ctx, tx, req)
	if err != nil {
		return err
	}
	//将练习置为提交状态
	err = submitPractice(ctx, tx, req)
	if err != nil {
		return err
	}

	//获取练习的批改模式
	correctMode, err := getCorrectMode(ctx, req.PracticeSubmissionID, tx)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		z.Error(err.Error())
		return err
	}

	go func(correctMode string) {
		if correctMode != AiCorrectMode {
			return
		}
		//TODO 对接ai批改接口
	}(correctMode)
	return nil
}

// HandleExit 处理学生退出作答，用在websocket连接断开的时候
func HandleExit(ctx context.Context, req ExitReq) error {
	// 用于测试mock数据
	testSign, ok := ctx.Value(TestSign).(string)
	if ok && testSign != "" {
		switch testSign {
		default:
			return nil
		}
	}
	// 参数检查
	if req.ExamineeID <= 0 && req.PracticeSubmissionID <= 0 {
		err := errors.New("examinee id and practice submission id both are smaller than 0 or equal to 0")
		z.Error(err.Error())
		return err
	}
	if err := cmn.Validate(req); err != nil {
		return err
	}

	//执行数据库操作
	sqlxDB := cmn.GetDbConn()
	if req.PracticeSubmissionID > 0 {

		Sql := `UPDATE t_practice_submissions
SET
  last_end_time = EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint * 1000,
  elapsed_seconds = elapsed_seconds + (
    (EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint * 1000) - last_start_time
  )
WHERE id = $1;`

		stmt, err := sqlxDB.Prepare(Sql)
		if err != nil {
			z.Error(err.Error())
			return err
		}
		defer stmt.Close()

		result, err := stmt.ExecContext(ctx, req.PracticeSubmissionID)
		if err != nil {
			z.Error(err.Error())
			return err
		}
		cnt, err := result.RowsAffected()
		if err != nil {
			z.Error(err.Error())
			return err
		}
		if cnt == 0 {
			err := fmt.Errorf("handle exit practice: practice submission of id:%d not found", req.PracticeSubmissionID)
			z.Error(err.Error())
			return err
		}
		return nil
	} else {
		Sql := `UPDATE t_examinee SET  updated_by = $1, update_time = $2, exit_cnt = exit_cnt + 1 WHERE id = $3 AND (status = $4 OR status = $5) `
		stmt, err := sqlxDB.Prepare(Sql)
		if err != nil {
			z.Error(err.Error())
			return err
		}
		defer stmt.Close()
		result, err := stmt.ExecContext(ctx, Sql, req.StudentId, time.Now(), req.ExamineeID, NormalStatus, CanBeEnterStatus)
		if err != nil {
			z.Error("更新考试异常状态失败", zap.Error(err))
			return err
		}
		row, err := result.RowsAffected()
		if err != nil {
			z.Error(err.Error())
			return err
		}
		if row == 0 {
			z.Error("can't find record to update", zap.Int64("examineeId", req.ExamineeID))
			return errors.New("can't find record to update")
		}

		return nil
	}
}

// checkExamStatus 查看当前考试的状态
func checkExamStatus(ctx context.Context, req CheckExamStatusReq) (int, error) {

	return 0, nil
}
