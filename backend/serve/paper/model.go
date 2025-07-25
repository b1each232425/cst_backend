package paper

import (
	"encoding/json"
	"math/rand"
	"w2w.io/cmn"
)

type InitialManualPaperResponse struct {
	Paper  cmn.TPaper        `json:"paper"`
	Groups []cmn.TPaperGroup `json:"paper_groups"`
}

type DeletePapersRequest struct {
	PaperIDs []int64 `json:"paper_ids" validate:"required,min=1,dive,gt=0"`
}

type UpdateManualPaperRequest struct {
	Actions []UpdateManualPaperAction `json:"actions" validate:"required,min=1"`
}

type UpdateManualPaperAction struct {
	Action  string          `json:"action"`  // 如：add_group、delete_question、update_info等
	Payload json.RawMessage `json:"payload"` // 动态解析，根据action而变
}

type ActionResult struct {
	Action string `json:"action"`
	Result any    `json:"result,omitempty"`
}

type UpdateManualPaperResponse struct {
	QuestionIDMap map[string]int64 `json:"question_id_map"`
	GroupIDMap    map[string]int64 `json:"group_id_map"`
}

type AddQuestionsRequest struct {
	TempID         string    `json:"temp_id" validate:"required,startswith=temp_question"` // 客户端生成的唯一标识(如UUID)
	GroupID        int64     `json:"group_id" validate:"omitempty"`
	Order          int64     `json:"order" validate:"omitempty"`
	BankQuestionID int64     `json:"bank_question_id" validate:"required,min=1"`
	Score          float64   `json:"score" validate:"required,min=1"`
	SubScore       []float64 `json:"sub_score" validate:"omitempty,dive,min=1"`
}

type AddQuestionGroupRequest struct {
	Name  string `json:"name" validate:"required"`
	Order int    `json:"order" validate:"required,min=1"`
}

type UpdatePaperQuestionRequest struct {
	ID       int64     `json:"id" validate:"required,min=1"`
	GroupID  int64     `json:"group_id" validate:"omitempty"`
	Score    float64   `json:"score,omitempty" validate:"omitempty,min=1"`
	SubScore []float64 `json:"sub_score,omitempty" validate:"omitempty,dive,min=0.5"`
}

type UpdatePaperBasicInfoRequest struct {
	Name        string   `json:"name,omitempty" validate:"omitempty,min=2,max=50"`
	Category    string   `json:"category,omitempty" validate:"omitempty,oneof=00 02"`
	Level       string   `json:"level,omitempty" validate:"omitempty,oneof=00 02 04"`
	Duration    int      `json:"duration,omitempty" validate:"omitempty,min=0"`
	Description string   `json:"description,omitempty" validate:"omitempty,max=500"`
	Tags        []string `json:"tags,omitempty" validate:"omitempty,min=1,dive,min=1,max=20"`
}

type UpdateQuestionsGroupRequest struct {
	ID   int64  `json:"id" validate:"min=1,omitempty"`
	Name string `json:"name,omitempty" validate:"omitempty"`
}

// 请求试卷列表结构体
type PaperListRequest struct {
	Name     string `form:"name" validate:"max=50,omitempty"`
	Tags     string `form:"tags" validate:"omitempty,max=100"`
	Page     int    `form:"page" validate:"required,min=1"`
	PageSize int    `form:"page_size" validate:"required,oneof=5 10 20"`
	Category string `form:"category" validate:"omitempty,oneof=00 02"`
}

// --------------------------------------------共享试卷--------------------------------------------

// 管理共享用户(添加/删除)
type ManagePaperShareRequest struct {
	PaperID int64   `json:"paper_id" validate:"required,gt=0"`
	UserIDs []int64 `json:"user_ids" validate:"required,gt=0,dive,gt=0"`
	Action  string  `json:"action" validate:"required,oneof=add remove"`
}

type GetSharedUserListRequest struct {
	PageSize int    `form:"page_size" validate:"required,oneof= 5 10 20"`
	Page     int    `form:"page" validate:"required,min=1"`
	Filter   string `form:"filter" validate:"omitempty,max=50"`
}

type UpdatePaperAccessModeRequest struct {
	PaperID    int64  `json:"paper_id" validate:"required,gt=0"`
	AccessMode string `json:"access_mode" validate:"required,oneof=00 02 04"`
}

// --------------------------------------------下发考卷--------------------------------------------

type GenerateExamPapersRequest struct {
	PaperID  int64  `json:"paper_id"    validate:"required,gt=0"`
	Category string `json:"category"    validate:"required,oneof=00 02"`
	// 当 Category == "00" 时必须有 ExamID
	ExamID int64 `json:"exam_id,omitempty"    validate:"required_if=Category 00"`
	// 当 Category == "02" 时必须有 PracticeID
	PracticeID int64 `json:"practice_id,omitempty" validate:"required_if=Category 02"`
}

type GenerateAnswerQuestionsRequest struct {
	ExamPaperID int64  `json:"exam_paper_id" validate:"required,gt=0"`
	Category    string `json:"category" validate:"required,oneof=00 02"`

	// 考试场景：Category == "00" 时必填，并且每个元素都要大于 0
	ExamineeIDs []int64 `json:"examinee_ids" validate:"required_if=Category 00,dive,gt=0"`

	// 练习场景：Category == "02" 时必填，并且每个元素都要大于 0
	PracticeSubmissionID []int64 `json:"practice_submission_id" validate:"required_if=Category 02,dive,gt=0"`

	// 布尔类型不需要额外标签，bool 默认就只有 true/false
	IsQuestionRandom bool       `json:"is_question_random"`
	IsOptionRandom   bool       `json:"is_option_random"`
	Attempt          int64      `json:"attempt" validate:"omitempty,min=1"`
	Shuffler         *rand.Rand `json:"-"`
}
