package paper_respondence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jmoiron/sqlx/types"
	"go.uber.org/zap"
	"w2w.io/cmn"
	"w2w.io/null"
)

const (
	ExamOverStatus       = "10"
	QuestionCanNotAnswer = "02"
	NormalStatus         = "00"
	CanBeEnterStatus     = "16"
	ExamineeDeleteStatus = " 08"
	MakeupExam           = "04"

	practiceSubmitted = "06"

	ExamType     = "00"
	PracticeType = "02"
)

var (
	ErrExamSessionIdInvalid = errors.New("exam session id must be > 0")
	ErrStudentInvalid       = errors.New("student id must be > 0")
	ErrExamineeIdIsNull     = errors.New("examinee id is null")
	ErrExamineeIdInvalid    = errors.New("examinee id must be > 0")
)

func insertOrUpdateAnswer(ctx context.Context, req SaveOrUpdateStudentAnswerReq, tx pgx.Tx) (cmn.TStudentAnswers, error) {
	//参数检测
	if err := cmn.Validate(req); err != nil {
		return cmn.TStudentAnswers{}, err
	}
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

func submitExam(ctx context.Context, tx pgx.Tx, req SubmitReq) (int64, error) {
	var err error

	now := time.Now().UTC()

	examinee := cmn.TExaminee{
		ID:         null.IntFrom(req.ExamineeID),
		EndTime:    null.IntFrom(now.UnixMilli()),
		UpdatedBy:  null.IntFrom(req.StudentId),
		UpdateTime: null.IntFrom(now.UnixMilli()),
	}

	//更新t_examinee表，如果end_time为空、start_time不为空才能设置，end_time不为空说明已经提交过了
	updateSqlForExaminee := `UPDATE t_examinee SET end_time = $1,status=$2,updated_by=$3,update_time=$4 WHERE id = $5 AND end_time IS NULL AND start_time IS NOT NULL RETURNING id`
	var updateId null.Int

	err = tx.QueryRow(ctx, updateSqlForExaminee, &examinee.EndTime, ExamOverStatus, &examinee.UpdatedBy, &examinee.UpdateTime, &examinee.ID).Scan(&updateId)
	if err != nil {
		z.Error("submitExamInDataBase update error", zap.Error(err))
		return -1, err
	}
	//设置作答为禁止作答状态
	err = setAnswerCanNotUpdate(ctx, req.ExamineeID, req.StudentId, tx)
	if err != nil {
		z.Error("submitExamInDataBase setAnswerCanNotUpdate error", zap.Error(err))
		return -1, err
	}

	return updateId.Int64, nil
}

func setAnswerCanNotUpdate(ctx context.Context, examineeId, userId int64, tx pgx.Tx) error {
	//更新t_student_answer表所有考试记录的状态为02
	updateSqlForAnswer := `UPDATE t_student_answers SET status=$1,updated_by=$2,update_time=$3 WHERE examinee_id=$4`

	_, err := tx.Exec(ctx, updateSqlForAnswer, QuestionCanNotAnswer, &userId, time.Now(), &examineeId)
	if err != nil {
		z.Error("update answer status err", zap.Error(err))
		return err
	}
	return nil
}

func submitPractice(ctx context.Context, tx pgx.Tx, req SubmitReq) error {
	//只有状态为正常作答练习以及结束时间为空的，才能进行更新
	submitSql := `update t_practice_submissions set end_time = $1,status=$2 where id = $3 AND status=$4 AND end_time IS NULL`
	var updateId null.Int
	err := tx.QueryRow(ctx, submitSql, time.Now().UTC(), practiceSubmitted, req.PracticeSubmissionID, NormalStatus).Scan(&updateId)
	if err != nil {
		z.Error("submitPractice error", zap.Error(err))
		return err
	}
	z.Info("submit success", zap.Int64("update id", updateId.Int64))
	return nil
}

// getCorrectMode 获取练习的批改模式
func getCorrectMode(ctx context.Context, practiceSubmissionId int64, tx pgx.Tx) (string, error) {
	selectSql := `SELECT 
  p.correct_mode
FROM 
  t_practice_submissions ps
JOIN 
  t_practice p ON ps.practice_id = p.id
WHERE 
  ps.id = $1;`
	var correctMode null.String

	err := tx.QueryRow(ctx, selectSql, practiceSubmissionId).Scan(&correctMode)
	if err != nil {
		z.Error("getCorrectMode error", zap.Error(err))
		return "", err
	}
	if !correctMode.Valid {
		err := fmt.Errorf("correct_mode of practiceSubmissionId:%d not exist", practiceSubmissionId)
		z.Error(err.Error())
		return "", err
	}
	return correctMode.String, nil

}

func UpdateLastStartTime(ctx context.Context, practiceSubmissionId int64, tx pgx.Tx) error {
	Sql := `UPDATE SET last_start_time = $1 WHERE id = $2 `

	_, err := tx.Exec(ctx, Sql, time.Now().UnixMilli(), practiceSubmissionId)
	if err != nil {
		z.Error("update last start Time error:" + err.Error())
		return err
	}
	return nil

}

// saveBeginTimeForPractice 练习保存开始时间调用，在创建练习试卷的时候调用
func saveBeginTimeForPractice(ctx context.Context, tx pgx.Tx, req InitRespondentReq) error {
	if err := cmn.Validate(req); err != nil {
		return err
	}
	if req.PracticeSubmissionID <= 0 {
		err := fmt.Errorf("PracticeSubmissionID 需要大于0")
		z.Error(err.Error())
		return err
	}
	//查看是否已经保存过开始时间，有就直接返回不报错
	checkSql := `select start_time from t_practice_submissions where id=$1`
	var start_time null.Int
	err := tx.QueryRow(ctx, checkSql, req.PracticeSubmissionID).Scan(&start_time)
	if err != nil {
		z.Error("checkPracticeIfSaveBeginTime error", zap.Error(err))
		return err
	}
	if start_time.Valid {
		return nil
	}

	// 开始保存开始时间
	updateSql := `UPDATE t_practice_submissions SET start_time = $1 WHERE id = $2 AND status = $3`
	_, err = tx.Exec(ctx, updateSql, req.PracticeSubmissionID, NormalStatus)
	if err != nil {
		z.Error("updatePracticeIfSaveBeginTime error", zap.Error(err))
		return err
	}
	return nil
}

// getActualEndTime 获取学生的当场考试的信息
func getExamineeInfo(ctx context.Context, examSessionId, studentId int64, tx pgx.Tx) (cmn.TVExamineeInfo, error) {
	if examSessionId <= 0 {

		z.Error(ErrExamSessionIdInvalid.Error())
		return cmn.TVExamineeInfo{}, ErrExamSessionIdInvalid
	}
	if studentId <= 0 {
		z.Error(ErrStudentInvalid.Error())
		return cmn.TVExamineeInfo{}, ErrStudentInvalid
	}
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
