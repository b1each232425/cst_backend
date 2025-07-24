package paper_respondence

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"
	"go.uber.org/zap"
	"time"
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

func insertOrUpdateAnswer(ctx context.Context, req SaveOrUpdateStudentAnswerReq, stmt *sqlx.Stmt) (cmn.TStudentAnswers, error) {
	//参数检测
	if err := cmn.Validate(req); err != nil {
		return cmn.TStudentAnswers{}, err
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
		AnswerScore:           null.Float{},
		Creator:               null.IntFrom(req.StudentId),
		CreateTime:            null.IntFrom(time.Now().UnixMilli()),
		UpdateTime:            null.IntFrom(time.Now().UnixMilli()),
		UpdatedBy:             null.IntFrom(req.StudentId),
		Addi:                  types.JSONText{},
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

	err := stmt.QueryRowContext(ctx,
		&studentAnswer.Type,
		&studentAnswer.ExamineeID,
		&studentAnswer.PracticeSubmissionID,
		&studentAnswer.QuestionID,
		&studentAnswer.Answer,
		&studentAnswer.AnswerScore,
		&studentAnswer.Creator,
		&studentAnswer.CreateTime,
		&studentAnswer.UpdatedBy,
		&studentAnswer.UpdateTime,
		&studentAnswer.Addi,
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

func getAnswerByExamineeIDAndQuestionID(ctx context.Context, req GetStudentAnswerReq, stmt *sqlx.Stmt) (cmn.TStudentAnswers, error) {
	if err := cmn.Validate(req); err != nil {
		return cmn.TStudentAnswers{}, err
	}

	r := cmn.TStudentAnswers{}

	err := stmt.QueryRow(ctx, req.ExamineeID, req.QuestionID).Scan(&r.ID, &r.Type, &r.ExamineeID, &r.QuestionID, &r.Answer, &r.Marker, &r.Creator, &r.CreateTime, &r.UpdatedBy, &r.UpdateTime, &r.Status)
	if err != nil {
		z.Error("getAnswerByExamineeID error", zap.Error(err))
		return cmn.TStudentAnswers{}, err
	}

	return r, nil
}

// checkPracticeIfSaveBeginTime
func checkPracticeIfSaveBeginTime(ctx context.Context, tx *sql.Tx, req SaveBeginTimeReq) error {
	checkSql := `select count(*) from t_practice_submissions where id=$1 AND start_time is not null`
	var count int
	err := tx.QueryRowContext(ctx, checkSql, req.PracticeSubmissionID).Scan(&count)
	if err != nil {
		z.Error("checkPracticeIfSaveBeginTime error", zap.Error(err))
		return err
	}
	if count > 0 {
		err := fmt.Errorf("practice of id%d already save begin time", req.PracticeSubmissionID)
		z.Info(err.Error())
		return err
	}
	return nil
}

func saveStudentBeginTimeForExam(ctx context.Context, tx *sql.Tx, req SaveBeginTimeReq) error {

	var err error

	examinee := cmn.TExaminee{
		ID:         null.IntFrom(req.ExamineeID),
		StartTime:  null.IntFrom(time.Now().UTC().UnixMilli()),
		UpdatedBy:  null.IntFrom(req.StudentId),
		UpdateTime: null.IntFrom(time.Now().UTC().UnixMilli()),
	}
	//只有start_time为空（说明是第一次进入作答）、end_time为空（说明没有提交过考试）、status为00（正常考试）04（补考）16（管理员允许进入）才能进行时间设置
	updateSql := `UPDATE t_examinee SET start_time = $1,update_time=$2,updated_by=$3 WHERE id = $4 AND end_time IS NULL AND start_time IS NULL AND (status = $5 OR status = $6 OR status = $7 ) RETURNING id`

	var updateId int64 = 0

	err = tx.QueryRowContext(ctx, updateSql, &examinee.StartTime, examinee.UpdateTime, examinee.UpdatedBy, &examinee.ID, CanBeEnterStatus, NormalStatus, MakeupExam).Scan(&updateId)
	if err != nil {
		z.Error("saveStudentBeginTime update error", zap.Error(err))
		return err
	}
	return nil
}

func submitExamInDataBase(ctx context.Context, tx *sql.Tx, req SubmitReq) (int64, error) {
	var err error

	now := time.Now().UTC()

	examinee := cmn.TExaminee{
		ID:         null.IntFrom(req.ExamineeID),
		EndTime:    null.IntFrom(now.UnixMilli()),
		UpdatedBy:  null.IntFrom(req.StudentId),
		UpdateTime: null.IntFrom(now.UnixMilli()),
	}

	//更新t_examinee表，如果end_time为空才能设置，不为空说明已经提交过了
	updateSqlForExaminee := `UPDATE t_examinee SET end_time = $1,status=$2,updated_by=$3,update_time=$4 WHERE id = $5 AND end_time IS NULL RETURNING id`
	var updateId int64 = 0

	err = tx.QueryRowContext(ctx, updateSqlForExaminee, &examinee.EndTime, ExamOverStatus, &examinee.UpdatedBy, &examinee.UpdateTime, &examinee.ID).Scan(&updateId)
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

	return updateId, nil
}

func setAnswerCanNotUpdate(ctx context.Context, examineeId, userId int64, tx *sql.Tx) error {
	//更新t_student_answer表所有考试记录的状态为02
	updateSqlForAnswer := `UPDATE t_student_answers SET status=$1,updated_by=$2,update_time=$3 WHERE examinee_id=$4`

	_, err := tx.ExecContext(ctx, updateSqlForAnswer, QuestionCanNotAnswer, &userId, time.Now(), &examineeId)
	if err != nil {
		z.Error("update answer status err", zap.Error(err))
		return err
	}
	return nil
}

// checkPracticeIfSubmitted 查看练习是否已经提交
func checkPracticeIfSubmitted(ctx context.Context, tx *sql.Tx, req SubmitReq) error {
	checkSql := `SELECT COUNT(*) FROM t_practice_submissions WHERE id =$1  AND status = $2`
	var count int64
	err := tx.QueryRowContext(ctx, checkSql, req.PracticeSubmissionID, practiceSubmitted).Scan(&count)
	if err != nil {
		z.Error("checkPracticeIfSubmitted error", zap.Error(err))
		return err
	}

	if count > 0 {
		err := fmt.Errorf("practice of submissionId %d is already submitted", req.PracticeSubmissionID)
		z.Error(err.Error())
		return err
	}
	return nil
}

func submitPractice(ctx context.Context, tx *sql.Tx, req SubmitReq) error {
	submitSql := `update t_practice_submissions set end_time = $1,status=$2 where id = $3 AND status=$4`
	result, err := tx.ExecContext(ctx, submitSql, time.Now().UTC(), practiceSubmitted, req.PracticeSubmissionID, NormalStatus)
	if err != nil {
		z.Error("submitPractice error", zap.Error(err))
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		z.Error("submitPractice error", zap.Error(err))
		return err
	}
	if rowsAffected == 0 {
		err := fmt.Errorf("practice_submission_id of %d not exist", req.PracticeSubmissionID)
		z.Error(err.Error())
		return err
	}
	return nil
}

// getCorrectMode 获取练习的批改模式
func getCorrectMode(ctx context.Context, practiceSubmissionId int64, tx *sql.Tx) (string, error) {
	selectSql := `SELECT 
  p.correct_mode
FROM 
  t_practice_submissions ps
JOIN 
  t_practice p ON ps.practice_id = p.id
WHERE 
  ps.id = $1;`
	var correctMode null.String

	err := tx.QueryRowContext(ctx, selectSql, practiceSubmissionId).Scan(&correctMode)
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

func UpdateLastStartTime(ctx context.Context, practiceSubmissionId int64, tx *sql.Tx) error {
	Sql := `UPDATE SET last_start_time = $1 WHERE id = $2 `

	_, err := tx.ExecContext(ctx, Sql, time.Now().UnixMilli(), practiceSubmissionId)
	if err != nil {
		z.Error("update last start Time error:" + err.Error())
		return err
	}
	return nil

}

// SaveBeginTimeForPracticeWithTx 练习保存开始时间调用，在创建练习试卷的时候调用
func SaveBeginTimeForPracticeWithTx(ctx context.Context, tx *sql.Tx, req SaveBeginTimeReq) error {
	if err := cmn.Validate(req); err != nil {
		return err
	}
	if req.PracticeSubmissionID <= 0 {
		err := fmt.Errorf("PracticeSubmissionID 需要大于0")
		z.Error(err.Error())
		return err
	}
	//查看是否已经提交过
	err := checkPracticeIfSaveBeginTime(ctx, tx, req)
	if err != nil {
		return err
	}
	err = saveStudentBeginTime(ctx, tx, req)
	if err != nil {
		return err
	}
	return nil
}
