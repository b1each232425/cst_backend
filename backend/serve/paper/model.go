package paper

import (
	"encoding/json"

	"w2w.io/cmn"
	"w2w.io/null"
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
	Order          int64     `json:"order" validate:"required,gt=0"`
	BankQuestionID int64     `json:"bank_question_id" validate:"required,min=1"`
	Score          float64   `json:"score" validate:"required,min=1"`
	SubScore       []float64 `json:"sub_score" validate:"omitempty,dive,min=1"`
}

type AddQuestionGroupRequest struct {
	Name  string `json:"name" validate:"required,min=1"`
	Order int    `json:"order" validate:"required,min=1"`
}

type UpdatePaperQuestionRequest struct {
	ID       int64     `json:"id" validate:"required,min=1"`
	GroupID  int64     `json:"group_id" validate:"omitempty,min=1"`
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
	ID   int64  `json:"id" validate:"min=1,required"`
	Name string `json:"name" validate:"required"`
}

// 请求试卷列表结构体
type PaperListRequest struct {
	Name     string `form:"name" validate:"max=50,omitempty"`
	Tags     string `form:"tags" validate:"omitempty,max=100"`
	Page     int    `form:"page" validate:"required,min=1"`
	PageSize int    `form:"page_size" validate:"required,oneof=5 10 20"`
	Category string `form:"category" validate:"omitempty,oneof=00 02"`
	Self     bool   `form:"self" validate:"omitempty"` // 是否只查询当前用户创建的试卷
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

// Group 学生试卷模版题组结构（试卷模块自用：解析存储在视图的JSON实体）
type Group struct {
	cmn.TPaperGroup
	Questions []Question `json:"questions"`
}

// Question 试卷模版中的题目（加上老师字修改的分数）
type Question struct {
	cmn.TQuestion
	BankQuestionID null.Int  `json:"bank_question_id"`
	SubScore       []float64 `json:"sub_score"`
}
