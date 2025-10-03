/*
 * @Author: WangKaidun 1597225095@qq.com
 * @Date: 2025-10-01 10:58:31
 * @LastEditors: WangKaidun 1597225095@qq.com
 * @LastEditTime: 2025-10-03 22:35:41
 * @FilePath: \assess\backend\serve\paper\model.go
 * @Description: 组卷计划关于请求和响应的结构体
 * Copyright (c) 2025 by WangKaidun 1597225095@qq.com, All Rights Reserved.
 */
package paper

import (
	"encoding/json"

	"github.com/jmoiron/sqlx/types"
	"w2w.io/cmn"
	"w2w.io/null"
)

// 自定义组卷
type InitialManualPaperResponse struct {
	Paper  cmn.TPaper        `json:"paper"`
	Groups []cmn.TPaperGroup `json:"paper_groups"`
}

// 删除试卷
type DeletePapersRequest struct {
	PaperIDs []int64 `json:"paper_ids" validate:"required,min=1,dive,gt=0"`
}

// 更新试卷
type UpdateManualPaperRequest struct {
	Actions []UpdateManualPaperAction `json:"actions" validate:"required,min=1"`
}

// 更新试卷的Action
type UpdateManualPaperAction struct {
	Action  string          `json:"action"`  // 如：add_group、delete_question、update_info等
	Payload json.RawMessage `json:"payload"` // 动态解析，根据action而变
}

// 更新试卷的Action结果
type ActionResult struct {
	Action string `json:"action"`
	Result any    `json:"result,omitempty"`
}

// 更新试卷的Response
type UpdateManualPaperResponse struct {
	QuestionIDMap map[string]int64 `json:"question_id_map"`
	GroupIDMap    map[string]int64 `json:"group_id_map"`
}

// 添加题目
type AddQuestionsRequest struct {
	TempID         int64     `json:"temp_id" validate:"required"` // 客户端生成的唯一标识(如UUID)
	GroupID        int64     `json:"group_id" validate:"omitempty"`
	Order          int64     `json:"order" validate:"required,gt=0"`
	Type           string    `json:"type" validate:"required,oneof=00 02 04 06 08 10 12"`
	BankQuestionID int64     `json:"bank_question_id" validate:"required,min=1"`
	Score          float64   `json:"score" validate:"required,min=1"`
	SubScore       []float64 `json:"sub_score" validate:"omitempty,dive,min=1"`
}

// 添加题组
type AddQuestionGroupRequest struct {
	Name  string `json:"name" validate:"required,min=1"`
	Order int    `json:"order" validate:"required,min=1"`
}

// 更新题目
type UpdatePaperQuestionRequest struct {
	ID       int64     `json:"id" validate:"required,min=1"`
	GroupID  int64     `json:"group_id" validate:"omitempty,min=1"`
	Score    float64   `json:"score,omitempty" validate:"omitempty,min=1"`
	SubScore []float64 `json:"sub_score,omitempty" validate:"omitempty,dive,min=0.5"`
}

// 更新试卷基本信息
type UpdatePaperBasicInfoRequest struct {
	Name        string   `json:"name,omitempty" validate:"omitempty,min=2,max=50"`
	Category    string   `json:"category,omitempty" validate:"omitempty,oneof=00 02"`
	Level       string   `json:"level,omitempty" validate:"omitempty,oneof=00 02 04 06 08"`
	Duration    *int     `json:"duration,omitempty" validate:"omitempty,min=0"`
	Description *string  `json:"description,omitempty" validate:"omitempty,max=500"`
	Tags        []string `json:"tags,omitempty" validate:"omitempty,min=1,dive,min=1,max=20"`
}

// 更新题组
type UpdateQuestionsGroupRequest struct {
	ID   int64  `json:"id" validate:"min=1,required"`
	Name string `json:"name" validate:"required"`
}

// 请求试卷列表结构体
type PaperListRequest struct {
	Name      string `form:"name" validate:"max=50,omitempty"`
	Tags      string `form:"tags" validate:"omitempty,max=100"`
	Page      int    `form:"page" validate:"required,min=1"`
	PageSize  int    `form:"page_size" validate:"required,oneof=5 10 20"`
	Category  string `form:"category" validate:"omitempty,oneof=00 02"`
	Self      bool   `form:"self" validate:"omitempty"`      // 是否只查询当前用户创建的试卷
	Published bool   `form:"published" validate:"omitempty"` // 是否只查询已发布的试卷
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
	AnswerNum      int64     `json:"answer_num"`
}

// --------------------------------------------智能组卷--------------------------------------------

// validate:"omitempty" 参数可传可不传
// 带* 传了不能为空

// 配置试卷题目
type QuestionConfig struct {
	Name                   string           `json:"name" validate:"required,min=1,max=50"`
	Type                   string           `json:"type" validate:"required,oneof=00 02 04 06 08 10 12"`
	Count                  int64            `json:"count" validate:"required,min=1"`
	AverageScore           float64          `json:"average_score" validate:"required,min=0.5"`
	DifficultyDistribution map[string]int64 `json:"difficulty_distribution"`
}

// 新建组卷计划
type PostPaperPlanRequest struct {
	ID                int64            `form:"id"`
	Name              string           `json:"name" validate:"required,min=1,max=200"`
	Category          string           `json:"category" validate:"required,oneof=00 02"`
	Level             string           `json:"level" validate:"required,oneof=00 02 04 06 08"`
	SuggestedDuration int64            `json:"suggested_duration" validate:"required,min=1"`
	Description       string           `json:"description" validate:"max=2000"`
	Tags              []string         `json:"tags" validate:"omitempty,max=20,dive,min=1,max=50"`
	KnowledgeBankID   int64            `json:"knowledge_bank_id" validate:"required,min=1"`
	QuestionBankIDs   []int64          `json:"question_bank_ids" validate:"required,dive,min=1"`
	PaperCount        int64            `json:"paper_count" validate:"required,min=1"`
	QuestionConfig    []QuestionConfig `json:"question_config" validate:"omitempty,dive"`
}

// 删除组卷计划（批量）
type DeletePaperPlanRequest struct {
	IDs []int64 `json:"ids" validate:"required,min=1,dive,min=1"`
}

// 获取组卷计划列表
type GetPaperPlanListRequest struct {
	Name     string `form:"name" validate:"omitempty,max=200"`
	Tags     string `form:"tags" validate:"omitempty,max=100"`
	PageSize int    `form:"page_size" validate:"required,oneof=5 10 20"`
	Page     int    `form:"page" validate:"required,min=1"`
}

// 用于接收数据库组卷计划列表项
type PlanListItem struct {
	ID                null.Int       `json:"ID,omitempty" db:"id,true,integer"`                                            /* id 组卷计划ID */
	Name              null.String    `json:"Name,omitempty" db:"name,false,character varying"`                             /* name 组卷计划名称 */
	KnowledgeBankName null.String    `json:"KnowledgeBankName,omitempty" db:"knowledge_bank_name,false,character varying"` /* knowledge_bank_name 知识点库名称 */
	Category          null.String    `json:"Category,omitempty" db:"category,false,character varying"`                     /* category 试卷用途 */
	QuestionCount     null.Int       `json:"QuestionCount,omitempty" db:"question_count,false,integer"`                    /* question_count 试题数量 */
	SuggestedDuration null.Int       `json:"SuggestedDuration,omitempty" db:"suggested_duration,false,integer"`            /* suggested_duration 建议时长(默认120分钟) */
	Tags              types.JSONText `json:"Tags,omitempty" db:"tags,false,jsonb"`                                         /* tags 试卷标签 */
	Level             null.String    `json:"Level,omitempty" db:"level,false,character varying"`                           /* level 试卷难度(00:易 02:较易 04:中 06:较难 08:难) */
	CreateTime        null.Int       `json:"CreateTime,omitempty" db:"create_time,false,bigint"`                           /* create_time 创建时间 */
	UpdateTime        null.Int       `json:"UpdateTime,omitempty" db:"update_time,false,bigint"`                           /* update_time 更新时间 */
	Status            null.String    `json:"Status,omitempty" db:"status,false,character varying"`                         /* status 状态(00:草稿 02:已组卷 04:已删除) */
}
