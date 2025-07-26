package paper_respondence

import "encoding/json"

// SaveOrUpdateStudentAnswerReq   保存学生答题请求
type SaveOrUpdateStudentAnswerReq struct {
	Id                   int64           `json:"id"`
	ExamineeID           int64           `json:"examinee_id" `
	PracticeSubmissionId int64           `json:"practice_submission_id" `
	Type                 string          `json:"type" validate:"required,oneof=00 02"`
	QuestionID           int64           `json:"question_id" validate:"required"`
	Answer               json.RawMessage `json:"answer" validate:"required"`
	StudentId            int64           `json:"student_id" validate:"required"`
	AttachmentPaths      json.RawMessage `json:"attachment_paths"`
}

type InitRespondentReq struct {
	Type                 string `json:"type" validate:"required,oneof=00 02"`
	ExamId               int64  `json:"exam_id" `
	ExamSessionId        int64  `json:"exam_session_id" `
	ExamineeID           int64  `json:"examinee_id" `
	PracticeSubmissionID int64  `json:"practice_submission_id" `
	PracticeId           int64  `json:"practice_id" `
	StudentId            int64  `json:"student_id" validate:"required"`
}

type SubmitReq struct {
	Type                 string `json:"type" validate:"required,oneof=00 02"`
	ExamId               int64  `json:"exam_id" `
	ExamineeID           int64  `json:"examinee_id" `
	ExamSessionId        int64  `json:"exam_session_id" `
	PracticeSubmissionID int64  `json:"practice_submission_id" `
	StudentId            int64  `json:"student_id" validate:"required"`
}

type SaveLastStartTimeReq struct {
	PracticeSubmissionID int64 `json:"practice_submission_id" validate:"required,gt=0"`
}

type ExitReq struct {
	ExamineeID           int64 `json:"examinee_id" `
	PracticeSubmissionID int64 `json:"practice_submission_id" `
	StudentId            int64 `json:"student_id" validate:"required,gte=0"`
}

type CheckExamStatusReq struct {
	ExamId        int64 `json:"exam_id" `
	ExamSessionId int64 `json:"exam_session_id" `
	ExamineeID    int64 `json:"examinee_id" validate:"required,gte=1"`
	StudentId     int64 `json:"student_id" validate:"required,gte=1"`
}
