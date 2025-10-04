package question_bank

import (
	"github.com/jmoiron/sqlx/types"
	"w2w.io/cmn"
)

// 查询题库参数结构体
type QueryQuestionBankParams struct {
	BankID   int64  // 题库ID
	Keyword  string // 关键字
	Page     int64  // 分页页码
	PageSize int64  // 分页大小
	Creator  int64  // 用户ID
	Type     string // 题库类型
}

// 查询题目参数结构体
type QueryQuestionsParams struct {
	BankID     int64    // 题库ID
	QuestionID int64    // 题目ID
	Content    string   // 题目内容
	Tags       []string // 题目标签
	Type       []string // 题目类型
	Difficulty []string // 题目难度
	Page       int64    // 分页页码
	PageSize   int64    // 分页大小
}

// QuestionOption 客观题题目选项结构
type QuestionOption struct {
	Label string `json:"label"` // 选项标签
	Value string `json:"value"` // 选项值
}

// SubjectiveAnswer 主观题答案结构
type SubjectiveAnswer struct {
	Index             int      `json:"index"`              // 答案索引
	Answer            string   `json:"answer"`             // 答案
	AlternativeAnswer []string `json:"alternative_answer"` // 备选答案
	Score             float64  `json:"score"`              // 分数
	GradingRule       string   `json:"grading_rule"`       // 评分规则
}

// QuestionBankDetail 题库详情
type QuestionBankDetail struct {
	ID                   int64    `json:"id"`                    // 题库ID
	Name                 string   `json:"name"`                  // 题库名称
	Type                 string   `json:"type"`                  // 题库类型
	Tags                 []string `json:"tags"`                  // 题库标签
	Creator              int64    `json:"creator"`               // 创建者ID
	CreatorName          string   `json:"creator_name"`          // 创建者名称
	CreateTime           string   `json:"create_time"`           // 创建时间
	UpdatedTime          string   `json:"updated_time"`          // 更新时间
	QuestionCount        int64    `json:"question_count"`        // 题目数量
	QuestionTypes        []string `json:"question_types"`        // 题目涉及所有类型(去重),00-单选,02-多选,04-判断,06-填空,08-简答
	QuestionTags         []string `json:"question_tags"`         // 题目涉及所有标签(去重)
	QuestionDifficulties []string `json:"question_difficulties"` // 题目涉及所有难度(去重),00-易,02-较易,04-中,06-较难,08-难
}

// QuestionFile 题目附件结构体
type QuestionFile struct {
	QuestionID int64  `json:"question_id" validate:"required"` // 题目ID
	CheckSum   string `json:"checksum" validate:"required"`    // 文件摘要
	Name       string `json:"name" validate:"required"`        // 文件名
	Size       int64  `json:"size" validate:"required"`        // 文件大小
}

// QuestionWithAllKnowledges 包含知识点库所有知识点的题目结构体
type QuestionWithAllKnowledges struct {
	cmn.TQuestion
	AllKnowledges types.JSONText `json:"allKnowledges,omitempty"` // 知识点库的所有知识点
}
