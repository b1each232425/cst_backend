package paper_respondence

//annotation:template-service
//author:{"name":"OuYangHaoBin","tel":"13712562121", "email":"1242968386@qq.com"}

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jmoiron/sqlx/types"
	"go.uber.org/zap"
	"io"
	"strconv"
	"strings"
	"time"
	"w2w.io/cmn"
	"w2w.io/null"
	"w2w.io/serve/examPaper"
	"w2w.io/serve/exam_mgt"
	"w2w.io/serve/mark"
	"w2w.io/serve/practice_mgt"
)

const (
	StartTimeNotArrived  = iota + 1 //未到达考试开始时间
	EndTimeArrived                  //考试结束时间已经到达
	ExamSubmitted                   //考试已经提交
	LateEntryTimeArrived            // 最迟进入时间已经到达
	ExamCanBeEnter                  //考试无论什么条件都能进入

	TIMEOUT     = 5 * time.Second //超时时间
	LONGTIMEOUT = 20 * time.Second

	ExamModeOnline = "00" //线上考试模式

	ExamTypeFixed    = "00" //固定时间考试模式
	ExamTypeFlexible = "02" //灵活时间考试模式

	TestSign = "test" //测试标志

	SUBMIT = "0" //用于提交的条件检测
	INIT   = "1" //用于初始化的条件检测
	STATUS = "2" //用于查看考试状态的条件检测
	ALLOW  = "3" //用于允许学生进入考试的条件检测

	ExamOverStatus = "10"

	NormalStatus = "00" //正常状态

	MakeupExam = "04" //补考状态

	CanBeEnterStatus = "16" //学生是被允许进入考试的，即使超过了最迟进入时间

	QuestionCanNotAnswer = "02" //不可作答的状态

	practiceSubmitted = "06" //练习提交状态

	ExamType     = "00" //考试类型
	PracticeType = "02" //练习类型

	ForceErr = "forceErr" //强制错误标记

	StudentDomainId = 2008 //学生域
	ExamInvigilator = 2004 //监考员域

	PracticeSubmissionDeleteStatus = "04"
)

var (
	ErrExamNotStart           = errors.New("考试还未开始")
	ErrExamOverEntryTime      = errors.New("无法进入考试，因为超过最迟进入考试时间")
	ErrExamFinished           = errors.New("考试已经结束")
	ErrAllowedSubmitNotArrive = errors.New("还未到达交卷的时间")
	ErrExamineeHaveSubmitted  = errors.New("考生已经提交试卷")
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

		Path:      "/respondent",
		Name:      "respondent",
		Developer: developer,
		WhiteList: true,

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
		WhiteList: true,

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
		WhiteList: true,

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
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})
	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: AllowStudentCanBeInExam,

		Path: "/respondent/allow",
		Name: "respondent_allow",

		Developer: developer,
		WhiteList: true,

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

	q.Err = checkDomain(q, StudentDomainId)
	if q.Err != nil {
		q.RespErr()
		return
	}

	method := strings.ToLower(q.R.Method)
	//执行数据库操作
	db := cmn.GetPgxConn()
	dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
	defer cancel()

	//强制错误，用于使得难以触发的错误强制它报错
	forceErr := ""
	forceErr, ok := ctx.Value(ForceErr).(string)
	if !ok {
		forceErr = ""
	}
	var result cmn.TStudentAnswers
	switch method {
	case "post":
		var buf []byte
		buf, q.Err = io.ReadAll(q.R.Body)
		if forceErr == "io-readAll" {
			q.Err = errors.New("io-readAll error")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer func() {
			q.Err = q.R.Body.Close()
			if forceErr == "body-close" {
				q.Err = errors.New("body-close")
			}
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
		if forceErr == "begin-tx" {
			err = errors.New("begin tx error")
		}
		if err != nil {
			z.Error(err.Error())
			q.Err = err
			q.RespErr()
			return
		}
		defer func() {
			//失败回滚
			if q.Err != nil {
				q.Err = tx.Rollback(dmlCtx)
				if forceErr == "rollback-tx" {
					q.Err = pgx.ErrTxCommitRollback
				}
				if q.Err != nil {
					z.Error(q.Err.Error())
				}
				return
			}
			//成功提交
			q.Err = tx.Commit(dmlCtx)
			if forceErr == "commit-tx" {
				q.Err = errors.New("commit tx error")
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
				return
			}
		}()

		result, q.Err = insertOrUpdateAnswer(dmlCtx, u, tx)
		if q.Err != nil {
			q.RespErr()
			return
		}

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
		selectSql := `SELECT id, type, examinee_id, question_id, answer, marker, creator, create_time, updated_by, update_time, status,answer_attachments_path FROM assessuser.t_student_answers WHERE `
		//查看哪个不为空
		if ed != "" {
			id, q.Err = strconv.ParseInt(ed, 10, 64)
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			selectSql += ` examinee_id =$1 AND question_id =$2`
		} else {
			id, q.Err = strconv.ParseInt(pd, 10, 64)
			if q.Err != nil {
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			selectSql += ` practice_submission_id =$1 AND question_id =$2`
		}
		//开始查询
		q.Err = db.QueryRow(ctx, selectSql, id, questionId).Scan(&result.ID, &result.Type, &result.ExamineeID, &result.QuestionID, &result.Answer, &result.Marker, &result.Creator, &result.CreateTime, &result.UpdatedBy, &result.UpdateTime, &result.Status, &result.AnswerAttachmentsPath)
		if q.Err != nil {
			z.Error("error", zap.Error(q.Err))
			q.RespErr()
			return
		}
	default:
		q.Err = fmt.Errorf("unknown method %s", method)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	var newBuf []byte
	resultPtr := &result
	if forceErr == "marshal-Err" {
		resultPtr = nil
	}
	newBuf, q.Err = cmn.MarshalJSON(resultPtr)
	if q.Err != nil {
		q.RespErr()
		return
	}

	q.Msg.Data = newBuf
	q.Msg.Msg = "success"
	q.Msg.RowCount = 1
	if forceErr != "rollback-tx" {
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
	q.Err = checkDomain(q, StudentDomainId)
	if q.Err != nil {
		q.RespErr()
		return
	}

	//强制错误，用于使得难以触发的错误强制它报错
	forceErr := ""
	forceErr, ok := ctx.Value(ForceErr).(string)
	if !ok {
		forceErr = ""
	}
	var buf []byte
	buf, q.Err = io.ReadAll(q.R.Body)
	if forceErr == "io.ReadAll" {
		q.Err = errors.New("io read all error")
	}
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	defer func() {
		q.Err = q.R.Body.Close()
		if forceErr == "close body err" {
			q.Err = errors.New("close body err")
		}
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
		q.Err = errors.New("student id is smaller than 0 or equal to 0")
		z.Error(q.Err.Error())
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
	if forceErr == "begin-tx" {
		err = errors.New("begin tx error")
	}
	if err != nil {
		q.Err = err
		z.Error(err.Error())
		q.RespErr()
		return
	}
	defer func() {
		if q.Err != nil {
			q.Err = tx.Rollback(dmlCtx)
			if forceErr == "rollback-tx" {
				q.Err = pgx.ErrTxCommitRollback
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
			}
			return
		}

		//提交事务
		err := tx.Commit(dmlCtx)
		if forceErr == "commit-tx" {
			err = errors.New("commit tx error")
		}
		if err != nil {
			z.Error(err.Error())
			return
		}
	}()
	var data []byte
	if forceErr == "type-err" {
		u.Type = "invalid-type"
	}
	switch u.Type {
	case ExamType:
		if u.ExamId <= 0 || u.ExamSessionId <= 0 {
			q.Err = fmt.Errorf("当前是考试，请输入大于0的考试id大于0的考试场次id")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		//获取当前用户的角色
		roleId := q.SysUser.Role.Int64
		z.Info("查看当前用户角色id", zap.Int64("roleId", roleId))
		var role null.String
		q.Err = tx.QueryRow(dmlCtx, `SELECT domain FROM assessuser.t_domain WHERE id=$1 `, roleId).Scan(&role)
		if q.Err != nil {
			err := fmt.Errorf("查找用户角色发生错误:" + q.Err.Error())
			q.Err = err
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 获取考试信息，role参数暂定1
		examInfo, err := exam_mgt.GetExamInfo(dmlCtx, u.ExamId, role.String)
		if err != nil {
			q.Err = err
			q.RespErr()
			return
		}
		//获取场次信息
		examSessions, err := exam_mgt.GetExamSessions(dmlCtx, role.String, u.ExamId)
		if forceErr == "get sessions err" {
			err = errors.New("get sessions err")
		}
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
		u.ExamineeID = examineeInfo.ID.Int64

		//保存开始时间
		q.Err = saveStudentBeginTimeForExam(dmlCtx, tx, u)
		if q.Err != nil {
			q.RespErr()
			return
		}

		//获取考卷
		_, groupInfo, questions, err := examPaper.LoadExamPaperDetailByUserId(dmlCtx, tx, examineeInfo.ExamPaperID.Int64, 0, examineeInfo.ID.Int64, false, false, false)
		if forceErr == "load paper detail err" {
			err = errors.New("load paper detail err")
		}
		if err != nil {
			q.Err = err
			q.RespErr()
			return
		}

		//定义结构体用于整合数据发送给前端
		type Msg struct {
			Sessions          []cmn.TExamSession `json:"session"`
			ExamInfo          cmn.TExamInfo      `json:"exam_info"`
			ExamineeInfo      cmn.TVExamineeInfo
			QuestionGroupInfo map[int64]*cmn.TExamPaperGroup
			Questions         map[int64][]*examPaper.ExamQuestion
		}

		msg := Msg{
			Sessions:          examSessions,
			ExamInfo:          examInfo,
			ExamineeInfo:      examineeInfo,
			QuestionGroupInfo: groupInfo,
			Questions:         questions,
		}

		data, err = json.Marshal(&msg)
		if forceErr == "marshal err" {
			err = errors.New("marshal err")
		}
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
		practiceInfo, groupInfo, questions, err := practice_mgt.EnterPracticeGetPaperDetails(dmlCtx, tx, u.PracticeId, u.StudentId)
		if err != nil {
			q.Err = err
			q.RespErr()
			return
		}

		u.PracticeSubmissionID = practiceInfo.PracticeSubmissionID
		//如果是第一次进入，就要保存练习开始时间
		q.Err = saveBeginTimeForPractice(dmlCtx, tx, u)
		if forceErr == "save time" {
			q.Err = errors.New("save time err")
		}
		if q.Err != nil {
			q.RespErr()
			return
		}

		//获取已经过去的时间
		var t cmn.TPracticeSubmissions
		elapsedSecondsSql := `SELECT elapsed_seconds FROM assessuser.t_practice_submissions WHERE id=$1 AND status=$2`
		q.Err = tx.QueryRow(dmlCtx, elapsedSecondsSql, u.PracticeSubmissionID, NormalStatus).Scan(&t.ElapsedSeconds)
		if forceErr == "select elapsed seconds" {
			q.Err = errors.New("select elapsed seconds err")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		//更新最近一次进入练习的时间
		Sql := `UPDATE t_practice_submissions SET last_start_time = EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint * 1000 WHERE id = $1 AND status=$2 RETURNING id`

		var updateId null.Int
		q.Err = tx.QueryRow(ctx, Sql, u.PracticeSubmissionID, NormalStatus).Scan(&updateId)
		if forceErr == "update-last-start-time-err" {
			q.Err = errors.New("update last start time err")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		//定义结构体用于整合数据发送给前端
		type Msg struct {
			Info              practice_mgt.EnterPracticeInfo
			QuestionGroupInfo map[int64]*cmn.TExamPaperGroup
			Questions         map[int64][]*examPaper.ExamQuestion
			ElapsedSeconds    int64
		}

		msg := &Msg{
			Info:              *practiceInfo,
			Questions:         questions,
			QuestionGroupInfo: groupInfo,
			ElapsedSeconds:    t.ElapsedSeconds.Int64,
		}
		data, err = json.Marshal(&msg)
		if forceErr == "marshal err" {
			err = errors.New("marshal err")
		}
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

	if forceErr == "close body err" {
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
	q.Err = checkDomain(q, StudentDomainId)
	if q.Err != nil {
		q.RespErr()
		return
	}
	//强制错误，用于使得难以触发的错误强制它报错
	forceErr := ""
	forceErr, ok := ctx.Value(ForceErr).(string)
	if !ok {
		forceErr = ""
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
	//获取学生id
	studentId := q.SysUser.ID.Int64
	if studentId <= 0 {
		err := fmt.Errorf("studentId is invalid")
		z.Error(err.Error())
		q.Err = err
		q.RespErr()
		return
	}

	dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
	defer cancel()
	db := cmn.GetPgxConn()
	tx, err := db.BeginTx(dmlCtx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if forceErr == "begin-tx" {
		err = errors.New("begin tx error")
	}
	if err != nil {
		q.Err = err
		z.Error(err.Error())
		q.RespErr()
		return
	}
	defer func() {
		if q.Err != nil {
			//如果不是tx done错误就返回给前端
			q.Err = tx.Rollback(dmlCtx)
			if forceErr == "rollback-tx" {
				q.Err = pgx.ErrTxCommitRollback
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
			}
			return
		}

		err := tx.Commit(dmlCtx)
		if forceErr == "commit-tx" {
			err = errors.New("commit tx error")
		}
		if err != nil {
			z.Error(err.Error())
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

	data, err := json.Marshal(&result)
	if forceErr == "marshal-Err" {
		err = errors.New("marshal-Err")
	}
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
	q.Err = checkDomain(q, StudentDomainId)
	if q.Err != nil {
		q.RespErr()
		return
	}
	//强制错误，用于使得难以触发的错误强制它报错
	forceErr := ""
	forceErr, ok := ctx.Value(ForceErr).(string)
	if !ok {
		forceErr = ""
	}
	var buf []byte
	buf, q.Err = io.ReadAll(q.R.Body)
	if forceErr == "io.ReadAll" {
		q.Err = errors.New("io read all error")
	}
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	defer func() {
		q.Err = q.R.Body.Close()
		if forceErr == "close body err" {
			q.Err = errors.New("close body err")
		}
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

	//执行数据库操作
	db := cmn.GetPgxConn()

	//开启事务
	tx, err := db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if forceErr == "begin-tx" {
		err = errors.New("begin tx error")
	}
	if err != nil {
		z.Error(err.Error())
		q.Err = err
		q.RespErr()
		return
	}
	defer func() {
		if q.Err != nil {
			q.Err = tx.Rollback(ctx)
			if forceErr == "rollback-tx" {
				q.Err = pgx.ErrTxCommitRollback
			}
			if q.Err != nil {
				z.Error(q.Err.Error())
			}
			return
		}
		q.Err = tx.Commit(ctx)
		if forceErr == "commit-tx" {
			q.Err = errors.New("commit-tx-error")
		}
		if q.Err != nil {
			z.Error(q.Err.Error())
		}
	}()

	dmlCtx, cancel := context.WithTimeout(ctx, TIMEOUT)
	defer cancel()
	if forceErr == "type-err" {
		u.Type = "invalid-type"
	}
	switch u.Type {
	case ExamType:
		if u.ExamId <= 0 || u.ExamSessionId <= 0 || u.ExamineeID <= 0 {
			q.Err = fmt.Errorf("当前是考试，请输入大于0的考试id、大于0的考试场次id、大于0的考生id")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		//查考当前是否符合条件去提交
		status, err := checkExamCondition(ctx, u.ExamSessionId, u.StudentId, tx, SUBMIT)
		if err != nil {
			q.Err = err
			q.RespErr()
			return
		}
		if status != ExamSubmitted {
			now := time.Now().UTC()

			examinee := cmn.TExaminee{
				ID:         null.IntFrom(u.ExamineeID),
				EndTime:    null.IntFrom(now.UnixMilli()),
				UpdatedBy:  null.IntFrom(u.StudentId),
				UpdateTime: null.IntFrom(now.UnixMilli()),
			}
			if forceErr == "exam-submit-err" {
				tx.Rollback(dmlCtx)
			}
			//更新t_examinee表，如果end_time为空、start_time不为空才能设置，end_time不为空说明已经提交过了
			updateSqlForExaminee := `UPDATE t_examinee SET end_time = $1,status=$2,updated_by=$3,update_time=$4 WHERE id = $5 AND end_time IS NULL AND start_time IS NOT NULL RETURNING id`
			var updateId null.Int

			q.Err = tx.QueryRow(ctx, updateSqlForExaminee, &examinee.EndTime, ExamOverStatus, &examinee.UpdatedBy, &examinee.UpdateTime, &examinee.ID).Scan(&updateId)
			if q.Err != nil {
				z.Error("QueryRow", zap.Error(q.Err))
				q.RespErr()
				return
			}
			if forceErr == "setAnswerCanNotUpdate error" {
				tx.Rollback(dmlCtx)
			}
			//设置作答为禁止作答状态
			q.Err = setAnswerCanNotUpdate(ctx, u.ExamineeID, 0, u.StudentId, tx)
			if q.Err != nil {
				q.RespErr()
				return
			}
		}
		// 异步批改
		go func() {
			mark.AutoMark(ctx, mark.QueryCondition{
				ExamineeID:    u.ExamineeID,
				ExamSessionID: u.ExamSessionId,
			})
		}()

	case PracticeType:
		if u.PracticeSubmissionID <= 0 || u.PracticeId <= 0 {
			q.Err = fmt.Errorf("当前是练习，请输入大于0的PracticeSubmissionID以及大于0的PracticeId")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		if forceErr == "setAnswerCanNotUpdate error" {
			tx.Rollback(dmlCtx)
		}

		//查看练习是否被其他人删除了
		q.Err = checkPracticeSubmission(ctx, tx, u.PracticeSubmissionID)
		if q.Err != nil {
			q.RespErr()
			return
		}

		//将练习置为提交状态
		q.Err = setAnswerCanNotUpdate(ctx, 0, u.PracticeSubmissionID, u.StudentId, tx)
		if forceErr == "set answer can not update err" {
			q.Err = errors.New("set answer can not update err")
		}
		if q.Err != nil {
			q.RespErr()
			return
		}
		if forceErr == "practice-submit-err" {
			tx.Rollback(dmlCtx)
		}
		//只有状态为正常作答练习以及结束时间为空的，才能进行更新
		submitSql := `update t_practice_submissions set end_time = (EXTRACT(EPOCH FROM NOW()) * 1000)::bigint,status=$1 where id = $2 AND status=$3 AND end_time IS NULL RETURNING id`
		var updateId null.Int
		q.Err = tx.QueryRow(ctx, submitSql, practiceSubmitted, u.PracticeSubmissionID, NormalStatus).Scan(&updateId)
		if q.Err != nil {
			z.Error("submitPractice error", zap.Error(q.Err))
			q.RespErr()
			return
		}
		// 异步批改
		go func() {
			mark.AutoMark(ctx, mark.QueryCondition{
				PracticeSubmissionID: u.PracticeSubmissionID,
				PracticeID:           u.PracticeId,
			})
		}()
	default:
		q.Err = fmt.Errorf("unknown student answer type: %s", u.Type)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	if forceErr == "rollback-tx" {
		return
	}
	if forceErr == "close body err" {
		return
	}
	q.Msg.Msg = "success"
	q.Msg.Status = 0
	q.Resp()
}

// AllowStudentCanBeInExam 允许学生进入考试（只有监考员才允许）
func AllowStudentCanBeInExam(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	method := strings.ToLower(q.R.Method)
	if method != "post" {
		q.Err = fmt.Errorf("please call /api/upLogin with  http POST method")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	//检查是否为监考员
	q.Err = checkDomain(q, ExamInvigilator)
	if q.Err != nil {
		q.RespErr()
		return
	}
	//强制错误，用于使得难以触发的错误强制它报错
	forceErr := ""
	forceErr, ok := ctx.Value(ForceErr).(string)
	if !ok {
		forceErr = ""
	}
	var buf []byte
	buf, q.Err = io.ReadAll(q.R.Body)
	if forceErr == "io.ReadAll" {
		q.Err = errors.New("io read all error")
	}
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	defer func() {
		q.Err = q.R.Body.Close()
		if forceErr == "close body err" {
			q.Err = errors.New("close body err")
		}
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
	var u AllowStudentEnterReq
	q.Err = json.Unmarshal(qry.Data, &u)
	if q.Err != nil {
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	//获取老师的id
	TeacherId := q.SysUser.ID.Int64
	if TeacherId <= 0 {
		q.Err = errors.New("TeacherId is smaller than 0 or equal to 0")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	u.TeacherId = TeacherId

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
	if forceErr == "begin-tx" {
		err = errors.New("begin tx error")
	}
	if err != nil {
		q.Err = err
		z.Error(err.Error())
		q.RespErr()
		return
	}
	defer func() {
		if q.Err != nil {
			q.Err = tx.Rollback(dmlCtx)
			if forceErr == "rollback-tx" {
				q.Err = pgx.ErrTxCommitRollback
			}
			if q.Err != nil {
				z.Error(q.Err.Error())

			}
			return
		}

		//提交事务
		err := tx.Commit(dmlCtx)
		if forceErr == "commit-tx" {
			err = errors.New("commit-tx error")
		}
		if err != nil {
			z.Error(err.Error())
			return
		}
	}()
	z.Info("查看参数", zap.Any("params", u))
	//检查考试状态是否允许操作
	_, q.Err = checkExamCondition(dmlCtx, u.ExamSessionId, u.StudentId, tx, ALLOW)
	if q.Err != nil {
		q.RespErr()
		return
	}

	//开始设置考生状态
	updateSql := `UPDATE t_examinee e
SET updated_by = $1,
    update_time = (EXTRACT(EPOCH FROM NOW()) * 1000)::bigint,
    status = $2
WHERE e.exam_session_id = $3
  AND e.student_id = $4
  AND e.status = $5
  AND EXISTS (
      SELECT 1
      FROM t_exam_session s
      WHERE s.id = e.exam_session_id
        AND $6=ANY(COALESCE(s.reviewer_ids, '{}'))
  ) RETURNING id;`
	var updateId null.Int
	q.Err = tx.QueryRow(ctx, updateSql, u.TeacherId, CanBeEnterStatus, u.ExamSessionId, u.StudentId, NormalStatus, u.TeacherId).Scan(&updateId)
	if q.Err != nil {
		z.Error("允许学生进入考试失败", zap.Error(err))
		q.RespErr()
		return
	}

	if forceErr == "close body err" {
		return
	}
	q.Err = nil
	q.Msg.Status = 0
	q.Msg.Msg = "success"
	q.Resp()
}

//--------------------封装一些给外部调用，或者常用的函数

// checkDomainIfStudent 查看是否是学生账号
func checkDomain(q *cmn.ServiceCtx, domainID int64) error {
	for _, domain := range q.Domains {
		if domain.ID.Int64 == domainID {
			return nil
		}
	}
	err := errors.New("invalid domain")
	z.Error(err.Error(), zap.Any("domainIds", q.Domains))
	return err
}

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
	db := cmn.GetPgxConn()

	var updateReturnId null.Int

	Sql := ""

	params := make([]interface{}, 0, 5)

	if req.PracticeSubmissionID > 0 {

		Sql = `UPDATE t_practice_submissions
SET
  last_end_time = EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint * 1000,
  elapsed_seconds = elapsed_seconds + (
    ((EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint * 1000) - last_start_time) / 1000.0
),
	updated_by=$1,
    update_time=EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint * 1000
WHERE id = $2 AND status = $3 RETURNING id`

		params = append(params, req.StudentId, req.PracticeSubmissionID, NormalStatus)
	} else {
		Sql = `UPDATE t_examinee SET  updated_by = $1, update_time = EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint * 1000, exit_cnt = exit_cnt + 1 WHERE id = $2 AND (status = $3 OR status = $4) RETURNING id`
		params = append(params, req.StudentId, req.ExamineeID, CanBeEnterStatus, NormalStatus)
	}
	err = db.QueryRow(ctx, Sql, params...).Scan(&updateReturnId)
	if err != nil {
		z.Error(err.Error())
		return err
	}

	return nil
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
		//查看当前考试是否开始
		if now.UnixMilli() < examineeInfo.StartTime.Int64 {
			z.Error(ErrExamNotStart.Error(), zap.Int64("examineeId", examineeInfo.ID.Int64))
			return 0, ErrExamNotStart
		}

		//查考当前是否是在考试结束
		if now.UnixMilli() > examineeInfo.ActualEndTime.Int64 {
			z.Error(ErrExamFinished.Error())
			return 0, ErrExamFinished
		}
		// 如果监考员允许或者学生之前已经进入过考试了，就允许他进入考试
		if examineeInfo.ExamineeStatus.String == CanBeEnterStatus || examineeInfo.ExamineeStartTime.Valid {
			return 0, nil
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

		if examineeInfo.ExamineeEndTime.Valid {
			return ExamSubmitted, nil
		}

		//必须满足考试模式为线上固定时段考试、设置了提前几分钟交卷、时间还未到达交卷时间才会触发错误
		if now.UnixMilli() < examineeInfo.AllowSubmitTime.Int64-1000 && examineeInfo.PeriodMode.String == ExamTypeFixed && examineeInfo.Mode.String == ExamModeOnline {
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
		//查看考生是否已经提交
		if examineeInfo.ExamineeEndTime.Valid {
			return ExamSubmitted, nil
		}

		// 如果监考员允许或者学生之前已经进入过考试了，就允许他进入考试
		if examineeInfo.ExamineeStatus.String == CanBeEnterStatus || examineeInfo.ExamineeStartTime.Valid {
			return ExamCanBeEnter, nil
		}
		//线上需要查考最迟进入时间
		if now.UnixMilli() > examineeInfo.AllowEntryTime.Int64 && examineeInfo.Mode.String == ExamModeOnline {
			return LateEntryTimeArrived, nil
		}
		return ExamCanBeEnter, nil
	case ALLOW:
		//查看当前考试是否开始
		if now.UnixMilli() < examineeInfo.StartTime.Int64 {
			z.Error(ErrExamNotStart.Error(), zap.Int64("examineeId", examineeInfo.ID.Int64))
			return 0, ErrExamNotStart
		}

		//查考当前是否是在考试结束
		if now.UnixMilli() > examineeInfo.ActualEndTime.Int64 {
			z.Error(ErrExamFinished.Error())
			return 0, ErrExamFinished
		}
		//查看考生是否已经提交
		if examineeInfo.ExamineeEndTime.Valid {
			z.Error(ErrExamineeHaveSubmitted.Error(), zap.Int64("examineeId", examineeInfo.ID.Int64))
			return 0, ErrExamineeHaveSubmitted
		}
	default:
		err := fmt.Errorf("unknown use %s", use)
		z.Error(err.Error())
		return 0, err
	}
	return 0, nil
}

// getActualEndTime 获取学生的当场考试的信息
func getExamineeInfo(ctx context.Context, examSessionId, studentId int64, tx pgx.Tx) (cmn.TVExamineeInfo, error) {
	sql := `SELECT id,
    actual_end_time,
       examinee_status,
       period_mode,
       mode,
       allow_entry_time,
       allow_submit_time,
       start_time ,
       examinee_start_time,
       examinee_end_time,
       exam_paper_id
	FROM v_examinee_info WHERE exam_session_id = $1 AND student_id=$2`

	var t cmn.TVExamineeInfo

	err := tx.QueryRow(ctx, sql, examSessionId, studentId).Scan(&t.ID, &t.ActualEndTime, &t.ExamineeStatus, &t.PeriodMode, &t.Mode, &t.AllowEntryTime, &t.AllowSubmitTime, &t.StartTime, &t.ExamineeStartTime, &t.ExamineeEndTime, &t.ExamPaperID)
	if err != nil {
		z.Error(err.Error())
		return cmn.TVExamineeInfo{}, err
	}
	return t, nil
}

func insertOrUpdateAnswer(ctx context.Context, req SaveOrUpdateStudentAnswerReq, tx pgx.Tx) (cmn.TStudentAnswers, error) {
	var sql string
	if req.ExamineeID > 0 {
		// 直接一条SQL搞定插入或更新，如果是禁止作答的状态，说明已经提交，不能改了
		sql = `
	INSERT INTO t_student_answers 
		(type, examinee_id, practice_submission_id, question_id, answer, creator, create_time, updated_by, update_time, status,answer_attachments_path)
	VALUES
		($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	ON CONFLICT (examinee_id, question_id)
	DO UPDATE SET
		type = EXCLUDED.type,
		answer = EXCLUDED.answer,
		answer_attachments_path = EXCLUDED.answer_attachments_path,
		updated_by = EXCLUDED.updated_by,
		update_time = EXCLUDED.update_time
	WHERE t_student_answers.status <> $12
	RETURNING id,creator,updated_by
	`
	} else if req.PracticeSubmissionId > 0 {
		//查看练习是否被其他人删除了
		err := checkPracticeSubmission(ctx, tx, req.PracticeSubmissionId)
		if err != nil {
			return cmn.TStudentAnswers{}, err
		}

		// 直接一条SQL搞定插入或更新，如果是禁止作答的状态，说明已经提交，不能改了
		sql = `
	INSERT INTO t_student_answers 
		(type, examinee_id, practice_submission_id, question_id, answer, creator, create_time, updated_by, update_time, status,answer_attachments_path)
	VALUES
		($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	ON CONFLICT (practice_submission_id, question_id)
	DO UPDATE SET
		type = EXCLUDED.type,
		answer = EXCLUDED.answer,
		answer_attachments_path = EXCLUDED.answer_attachments_path,
		updated_by = EXCLUDED.updated_by,
		update_time = EXCLUDED.update_time
	WHERE t_student_answers.status <> $12
	RETURNING id,creator,updated_by
	`
	}

	//如果没有附件，就给个空的切片
	var AttachmentPaths types.JSONText
	if req.AttachmentPaths == nil {
		AttachmentPaths = types.JSONText("[]")
	} else {
		AttachmentPaths = types.JSONText(req.AttachmentPaths)
	}
	z.Debug("attachment_paths", zap.Any("attachment_paths", AttachmentPaths.String()))
	studentAnswer := cmn.TStudentAnswers{
		Type:                  null.NewString(req.Type, true),
		QuestionID:            null.IntFrom(req.QuestionID),
		Answer:                types.JSONText(req.Answer),
		Creator:               null.IntFrom(req.StudentId),
		CreateTime:            null.IntFrom(time.Now().UnixMilli()),
		UpdateTime:            null.IntFrom(time.Now().UnixMilli()),
		UpdatedBy:             null.IntFrom(req.StudentId),
		Status:                null.NewString("00", true),
		AnswerAttachmentsPath: AttachmentPaths,
	}
	//根据type来查看是练习还是考试
	switch req.Type {
	case ExamType:
		studentAnswer.ExamineeID = null.IntFrom(req.ExamineeID)
		studentAnswer.PracticeSubmissionID = null.Int{}
	case PracticeType:
		studentAnswer.ExamineeID = null.Int{}
		studentAnswer.PracticeSubmissionID = null.IntFrom(req.PracticeSubmissionId)
	default:
		err := fmt.Errorf("unknown student answer type: %s", req.Type)
		z.Error(err.Error())
		return cmn.TStudentAnswers{}, err
	}

	err := tx.QueryRow(ctx, sql,
		&studentAnswer.Type,
		&studentAnswer.ExamineeID,
		&studentAnswer.PracticeSubmissionID,
		&studentAnswer.QuestionID,
		&studentAnswer.Answer,
		&studentAnswer.Creator,
		&studentAnswer.CreateTime,
		&studentAnswer.UpdatedBy,
		&studentAnswer.UpdateTime,
		&studentAnswer.Status,
		&studentAnswer.AnswerAttachmentsPath,
		QuestionCanNotAnswer,
	).Scan(&studentAnswer.ID, &studentAnswer.Creator, &studentAnswer.UpdatedBy)

	if err != nil {
		z.Error("SaveStudentExamAnswer insertOrUpdate error", zap.Error(err))
		return cmn.TStudentAnswers{}, err
	}
	z.Info("insert or update exam answer result", zap.Any("result", studentAnswer))

	return studentAnswer, nil
}

// saveStudentBeginTimeForExam 保存考试作答开始时间
func saveStudentBeginTimeForExam(ctx context.Context, tx pgx.Tx, req InitRespondentReq) error {

	var err error
	//查看start_time是否已经设置过，如果设置过的话就不报错，直接返回nil
	var startTime null.Int
	selectSql := `SELECT start_time FROM t_examinee WHERE id=$1 FOR UPDATE `
	err = tx.QueryRow(ctx, selectSql, req.ExamineeID).Scan(&startTime)
	if err != nil {
		z.Error("saveStudentBeginTimeForExam error", zap.Error(err))
		return err
	}
	if startTime.Valid {
		return nil
	}

	//start_time为空，进行操作
	examinee := cmn.TExaminee{
		ID:         null.IntFrom(req.ExamineeID),
		StartTime:  null.IntFrom(time.Now().UTC().UnixMilli()),
		UpdatedBy:  null.IntFrom(req.StudentId),
		UpdateTime: null.IntFrom(time.Now().UTC().UnixMilli()),
	}
	//只有start_time为空（说明是第一次进入作答）、end_time为空（说明没有提交过考试）、status为00（正常考试）04（补考）16（管理员允许进入）才能进行时间设置
	updateSql := `UPDATE t_examinee SET start_time = $1,update_time=$2,updated_by=$3 WHERE id = $4 AND end_time IS NULL AND start_time IS NULL AND (status = $5 OR status = $6 OR status = $7 ) RETURNING id`

	var updateId int64 = 0

	err = tx.QueryRow(ctx, updateSql, &examinee.StartTime, examinee.UpdateTime, examinee.UpdatedBy, &examinee.ID, CanBeEnterStatus, NormalStatus, MakeupExam).Scan(&updateId)
	if err != nil {
		z.Error("saveStudentBeginTime update error", zap.Error(err))
		return err
	}
	return nil
}

// setAnswerCanNotUpdate 设置作答不可作答
func setAnswerCanNotUpdate(ctx context.Context, examineeId, practiceSubmissionId, userId int64, tx pgx.Tx) error {

	var updateSqlForAnswer string
	var id int64
	if examineeId > 0 {
		updateSqlForAnswer = `UPDATE t_student_answers SET status=$1,updated_by=$2,update_time=(EXTRACT(EPOCH FROM NOW()) * 1000)::bigint WHERE examinee_id=$3`
		id = examineeId
	} else if practiceSubmissionId > 0 {
		updateSqlForAnswer = `UPDATE t_student_answers SET status=$1,updated_by=$2,update_time=(EXTRACT(EPOCH FROM NOW()) * 1000)::bigint WHERE practice_submission_id=$3`
		id = practiceSubmissionId
	}

	//更新t_student_answer表所有考试记录的状态为02
	_, err := tx.Exec(ctx, updateSqlForAnswer, QuestionCanNotAnswer, &userId, &id)
	if err != nil {
		z.Error("update answer status err", zap.Error(err))
		return err
	}
	return nil
}

// saveBeginTimeForPractice 练习保存开始时间调用，在创建练习试卷的时候调用
func saveBeginTimeForPractice(ctx context.Context, tx pgx.Tx, req InitRespondentReq) error {

	//查看是否已经保存过开始时间，有就直接返回不报错
	checkSql := `select start_time from t_practice_submissions where id=$1 `
	var startTime null.Int
	err := tx.QueryRow(ctx, checkSql, req.PracticeSubmissionID).Scan(&startTime)
	if err != nil {
		z.Error("checkPracticeIfSaveBeginTime error", zap.Error(err))
		return err
	}
	if startTime.Valid {
		return nil
	}

	// 开始保存开始时间
	updateSql := `UPDATE t_practice_submissions SET start_time =(EXTRACT(EPOCH FROM NOW()) * 1000)::bigint WHERE id = $1 AND status = $2`
	_, err = tx.Exec(ctx, updateSql, req.PracticeSubmissionID, NormalStatus)
	if err != nil {
		z.Error("updatePracticeIfSaveBeginTime error", zap.Error(err))
		return err
	}
	return nil
}

// checkPracticeSubmission 查看练习的状态
func checkPracticeSubmission(ctx context.Context, tx pgx.Tx, practiceSubmissionId int64) error {

	sql := `SELECT EXISTS(
    SELECT 1
    FROM t_practice_submissions
    WHERE id = $1 AND status=$2 AND attempt=-1
) AS result;`
	var result null.Bool
	err := tx.QueryRow(ctx, sql, practiceSubmissionId, PracticeSubmissionDeleteStatus).Scan(&result)
	if err != nil {
		z.Error("checkPracticeSubmission error", zap.Error(err))
		return err
	}
	z.Info("result", zap.Bool("result", result.Bool))
	if !result.Bool {
		return nil
	}
	err = fmt.Errorf("当前练习已经被删除")
	z.Info(err.Error(), zap.Int64("practiceSubmissionId", practiceSubmissionId))
	return err
}
