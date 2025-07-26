package paper_respondence

//annotation:template-service
//author:{"name":"OuYangHaoBin","tel":"13712562121", "email":"1242968386@qq.com"}

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
	"w2w.io/cmn"
	"w2w.io/serve/examPaper"
	"w2w.io/serve/exam_mgt"
	"w2w.io/serve/practice_mgt"
)

const (
	StartTimeNotArrived  = iota + 1 //未到达考试开始时间
	EndTimeArrived                  //考试结束时间已经到达
	ExamSubmitted                   //考试已经提交
	LateEntryTimeArrived            // 最迟进入时间已经到达
	ExamCanBeEnter                  //考试无论什么条件都能进入

	TIMEOUT = 5 * time.Second

	AiCorrectMode = "02"

	ExamModeOnline = "00"

	ExamTypeFixed    = "00"
	ExamTypeFlexible = "02"

	TestSign = "test"

	SUBMIT = "submit"
	INIT   = "init"
	STATUS = "status"
)

var (
	ErrExamNotStart           = errors.New("exam has not started yet")
	ErrExamOverEntryTime      = errors.New("exam can not be entry,because over entry time")
	ErrExamFinished           = errors.New("exam has finished")
	ErrAllowedSubmitNotArrive = errors.New("allowed submit time not arrive")
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
		Fn: InitRespondent,

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
	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: CheckExamStatus,

		Path: "/respondent/exam/status",
		Name: "respondent_exam_status",

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
	//执行数据库操作
	db := cmn.GetPgxConn()
	dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
	defer cancel()

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

		tx, err := db.BeginTx(dmlCtx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
		if err != nil {
			z.Error(err.Error())
			q.Err = err
			q.RespErr()
			return
		}
		defer func() {
			if q.Err = tx.Rollback(dmlCtx); q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}()

		var result cmn.TStudentAnswers
		result, q.Err = insertOrUpdateAnswer(dmlCtx, u, tx)
		if q.Err != nil {
			q.RespErr()
			return
		}
		//提交
		if q.Err = tx.Commit(dmlCtx); q.Err != nil {
			z.Error(q.Err.Error())
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

		//表示考生id或者练习的submissionId
		var id int64
		//查询sql语句
		var selectSql string
		//查看哪个不为空
		if ed != "" {
			id, q.Err = strconv.ParseInt(ed, 10, 64)
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			selectSql = `SELECT id, type, examinee_id, question_id, answer, marker, creator, create_time, updated_by, update_time, status FROM t_student_answers WHERE examinee_id =$1 AND question_id =$2`
		} else {
			id, q.Err = strconv.ParseInt(pd, 10, 64)
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			selectSql = `SELECT id, type, examinee_id, question_id, answer, marker, creator, create_time, updated_by, update_time, status FROM t_student_answers WHERE practice_submission_id =$1 AND question_id =$2`
		}

		var result cmn.TStudentAnswers

		//开始查询
		r := cmn.TStudentAnswers{}

		q.Err = db.QueryRow(ctx, selectSql, id, questionId).Scan(&r.ID, &r.Type, &r.ExamineeID, &r.QuestionID, &r.Answer, &r.Marker, &r.Creator, &r.CreateTime, &r.UpdatedBy, &r.UpdateTime, &r.Status)
		if q.Err != nil {
			z.Error("getAnswerByExamineeID error", zap.Error(q.Err))

		}

		var buf []byte
		buf, q.Err = cmn.MarshalJSON(&result)

		q.Msg.RowCount = 1
		q.Msg.Data = buf
		q.Resp()
	}

}

// InitRespondent 作答初始化
func InitRespondent(ctx context.Context) {
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

	var u InitRespondentReq

	q.Err = json.Unmarshal(qry.Data, &u)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	studentId := q.SysUser.ID.Int64
	if studentId <= 0 {
		err := errors.New("student id is smaller than 0")
		z.Error(err.Error())
		q.RespErr()
		return
	}
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
	db := cmn.GetPgxConn()
	tx, err := db.BeginTx(dmlCtx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		q.Err = err
		z.Error(err.Error())
		q.RespErr()
		return
	}
	defer func() {
		//如果不是tx done错误就返回给前端
		q.Err = tx.Rollback(dmlCtx)
		if q.Err != nil && !errors.Is(q.Err, pgx.ErrTxCommitRollback) {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
	}()
	var data []byte
	switch u.Type {
	case ExamType:
		if u.ExamId <= 0 || u.ExamSessionId <= 0 {
			q.Err = fmt.Errorf("当前是考试，请输入大于0的考试id大于0的考试场次id")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		// 获取考试信息，role参数暂定1
		examInfo, err := exam_mgt.GetExamInfo(dmlCtx, u.ExamId, 1)
		if err != nil {
			q.Err = err
			q.RespErr()
			return
		}
		//获取场次信息
		examSessions, err := exam_mgt.GetExamSessions(dmlCtx, u.ExamId, 1)
		if err != nil {
			q.Err = err
			q.RespErr()
			return
		}

		examineeInfo, err := getExamineeInfo(dmlCtx, u.ExamSessionId, u.StudentId, tx)
		if err != nil {
			q.Err = err
			q.RespErr()
			return
		}

		//如果是监考员设置了学生可以进入考试的话，就不需要检查条件
		if examineeInfo.ExamineeStatus.String != CanBeEnterStatus {
			// 查考当前是否符合条件去初始化
			_, err = checkExamCondition(dmlCtx, u.ExamSessionId, u.StudentId, tx, INIT)
			if err != nil {
				q.Err = err
				q.RespErr()
				return
			}
		}

		//保存开始时间
		q.Err = saveStudentBeginTimeForExam(dmlCtx, tx, u)
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		//获取考卷
		_, groupInfo, questions, err := examPaper.LoadExamPaperDetailByUserId(dmlCtx, examineeInfo.ExamPaperID.Int64, 0, examineeInfo.ID.Int64, false, false, false)
		if err != nil {
			q.Err = err
			q.RespErr()
			return
		}

		//定义结构体用于整合数据发送给前端
		type Msg struct {
			Sessions          []cmn.TExamSession `json:"session"`
			ExamInfo          cmn.TExamInfo      `json:"exam_info"`
			ExamineeId        int64              `json:"examinee_id"`
			QuestionGroupInfo map[int64]*cmn.TExamPaperGroup
			Questions         map[int64][]*examPaper.ExamQuestion
		}

		msg := Msg{
			Sessions:          examSessions,
			ExamInfo:          examInfo,
			ExamineeId:        examineeInfo.ID.Int64,
			QuestionGroupInfo: groupInfo,
			Questions:         questions,
		}

		data, err = cmn.MarshalJSON(&msg)
		if err != nil {
			q.Err = err
			q.RespErr()
			return
		}

	case PracticeType:

		if u.PracticeId <= 0 {
			err := fmt.Errorf("practice id is smaller than 0")
			z.Error(err.Error())
			q.Err = err
			q.RespErr()
			return
		}

		//练习初始化并获取试卷数据
		info, groupInfo, questions, err := practice_mgt.EnterPracticeGetPaperDetails(dmlCtx, tx, u.PracticeId, u.StudentId)
		if err != nil {
			q.Err = err
			q.RespErr()
			return
		}
		//如果是第一次进入，就要保存练习开始时间
		if err := saveBeginTimeForPractice(dmlCtx, tx, u); err != nil {
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

		//定义结构体用于整合数据发送给前端
		type Msg struct {
			Info              practice_mgt.EnterPracticeInfo
			QuestionGroupInfo map[int64]*cmn.TExamPaperGroup
			Questions         map[int64][]*examPaper.ExamQuestion
		}

		msg := &Msg{
			Info:              *info,
			Questions:         questions,
			QuestionGroupInfo: groupInfo,
		}
		data, err = cmn.MarshalJSON(&msg)
		if err != nil {
			q.Err = err
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
	if err := tx.Commit(dmlCtx); err != nil {
		q.Err = err
		z.Error(err.Error())
		q.RespErr()
		return
	}
	q.Err = nil
	q.Msg.Status = 0
	q.Msg.Data = data
	q.Msg.Msg = "success"
	q.Resp()
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

	//获取考生id
	examSessionId := r.URL.Query().Get("exam_session_id")
	if examSessionId == "" {
		err := fmt.Errorf("examSessionId is required")
		z.Error(err.Error())
		q.Err = err
		q.RespErr()
		return
	}

	examSessionIdInt, err := strconv.ParseInt(examSessionId, 10, 64)
	if err != nil {
		q.Err = err
		z.Error(err.Error())
		q.RespErr()
		return
	}

	studentId := q.SysUser.ID.Int64
	dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
	defer cancel()
	db := cmn.GetPgxConn()
	tx, err := db.BeginTx(dmlCtx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		q.Err = err
		z.Error(err.Error())
		q.RespErr()
		return
	}
	defer func() {
		//如果不是tx done错误就返回给前端
		q.Err = tx.Rollback(dmlCtx)
		if q.Err != nil && !errors.Is(q.Err, sql.ErrTxDone) {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
	}()

	//查考当前考生当场考试正处于什么状态
	result, err := checkExamCondition(dmlCtx, examSessionIdInt, studentId, tx, STATUS)
	if err != nil {
		q.Err = err
		q.RespErr()
		return
	}

	if err := tx.Commit(dmlCtx); err != nil {
		q.Err = err
		z.Error(err.Error())
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
		//执行数据库操作
		db := cmn.GetPgxConn()

		//开启事务
		tx, err := db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
		if err != nil {
			z.Error(err.Error())
			q.Err = err
			q.RespErr()
			return
		}
		defer func() {
			if q.Err = tx.Rollback(ctx); q.Err != nil && !errors.Is(q.Err, pgx.ErrTxCommitRollback) {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}()
		//查考当前是否符合条件去提交
		_, err = checkExamCondition(ctx, u.ExamSessionId, u.StudentId, tx, SUBMIT)
		if err != nil {
			q.Err = err
			q.RespErr()
			return
		}
		updateId, err := submitExam(ctx, tx, u)
		if err != nil {
			q.Err = err
			q.RespErr()
			return
		}
		z.Info("update success", zap.Int64("updateId", updateId))

		if err := tx.Commit(ctx); err != nil {
			z.Error(err.Error())
			q.Err = err
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
		//执行数据库操作
		db := cmn.GetPgxConn()

		//开启事务
		tx, err := db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
		if err != nil {
			z.Error(err.Error())
			q.Err = err
			q.RespErr()
			return
		}
		defer func() {
			if q.Err = tx.Rollback(ctx); q.Err != nil && !errors.Is(q.Err, pgx.ErrTxCommitRollback) {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}()
		//将练习置为提交状态
		err = submitPractice(ctx, tx, u)
		if err != nil {
			q.Err = err
			q.RespErr()
			return
		}

		//获取练习的批改模式
		correctMode, err := getCorrectMode(ctx, u.PracticeSubmissionID, tx)
		if err != nil {
			q.Err = err
			q.RespErr()
			return
		}

		if err := tx.Commit(dmlCtx); err != nil {
			z.Error(err.Error())
			q.Err = err
			q.RespErr()
			return
		}

		go func(correctMode string) {
			if correctMode != AiCorrectMode {
				return
			}
			//TODO 对接ai批改接口
		}(correctMode)
	default:
		q.Err = fmt.Errorf("unknown student answer type: %s", u.Type)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
}

//--------------------封装一些给外部调用，或者常用的函数

// HandleExit 给予时间同步模块处理学生退出作答，用在websocket连接断开的时候
func HandleExit(ctx context.Context, req ExitReq) (err error) {
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
WHERE id = $1 AND status = $2;`

		stmt, err := sqlxDB.Prepare(Sql)
		if err != nil {
			z.Error(err.Error())
			return err
		}
		defer func() {
			if err = stmt.Close(); err != nil {
				z.Error(err.Error())
			}
		}()

		_, err = stmt.ExecContext(ctx, req.PracticeSubmissionID, NormalStatus)
		if err != nil {
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
		_, err = stmt.ExecContext(ctx, Sql, req.StudentId, time.Now(), req.ExamineeID, NormalStatus, CanBeEnterStatus)
		if err != nil {
			z.Error("更新考试异常状态失败", zap.Error(err))
			return err
		}

		return nil
	}
}

// checkExamCondition 为考试初始化以及考试提交查考当前是否符合条件，多处地方需要使用，因此封装
func checkExamCondition(ctx context.Context, examSession, studentID int64, tx pgx.Tx, use string) (int, error) {
	examineeInfo, err := getExamineeInfo(ctx, examSession, studentID, tx)
	if err != nil {
		return 0, err
	}

	//查考是否在考试开始之后提交的
	now := time.Now()

	switch use {
	case INIT:
		if now.UnixMilli() < examineeInfo.StartTime.Int64 {
			z.Error(ErrExamNotStart.Error(), zap.Int64("examineeId", examineeInfo.ID.Int64))
			return 0, ErrExamNotStart
		}

		//查考当前是否是在考试结束后进入考试
		if now.UnixMilli() > examineeInfo.ActualEndTime.Int64 {
			z.Error(ErrExamFinished.Error())
			return 0, ErrExamFinished
		}
		//必须满足考试模式为线上固定时段考试、设置了最迟几分钟计入考试、超过进入时间才会触发错误
		if now.UnixMilli() > examineeInfo.AllowEntryTime.Int64 && examineeInfo.AllowEntryTime.Int64 != examineeInfo.StartTime.Int64 && examineeInfo.PeriodMode.String == ExamTypeFixed && examineeInfo.Mode.String == ExamModeOnline {
			z.Error(ErrExamOverEntryTime.Error())
			return 0, ErrExamOverEntryTime
		}
	case SUBMIT:
		if now.UnixMilli() < examineeInfo.StartTime.Int64 {
			z.Error(ErrExamNotStart.Error(), zap.Int64("examineeId", examineeInfo.ID.Int64))
			return 0, ErrExamNotStart
		}

		//必须满足考试模式为线上固定时段考试、设置了提前几分钟交卷、时间还未到达交卷时间才会触发错误
		if now.UnixMilli() < examineeInfo.AllowSubmitTime.Int64 && examineeInfo.PeriodMode.String == ExamTypeFixed && examineeInfo.Mode.String == ExamModeOnline {
			z.Error(ErrAllowedSubmitNotArrive.Error())
			return 0, ErrAllowedSubmitNotArrive
		}
	case STATUS:
		//查考考试是否开始
		if now.UnixMilli() < examineeInfo.StartTime.Int64 {
			return StartTimeNotArrived, nil
		}
		//查考考试是否结束
		if now.UnixMilli() > examineeInfo.ActualEndTime.Int64 {
			return EndTimeArrived, nil
		}

		// 如果监考员允许或者学生之前已经进入过考试了，就允许他进入考试
		if examineeInfo.ExamineeStatus.String == CanBeEnterStatus || examineeInfo.ExamineeStartTime.Valid {
			return ExamCanBeEnter, nil
		}
		//线上需要查考最迟进入时间
		if now.UnixMilli() > examineeInfo.AllowEntryTime.Int64 && examineeInfo.Mode.String == ExamModeOnline {
			return LateEntryTimeArrived, nil
		}
	default:
		err := fmt.Errorf("unknown use %s", use)
		z.Error(err.Error())
		return 0, err
	}
	return 0, nil
}
