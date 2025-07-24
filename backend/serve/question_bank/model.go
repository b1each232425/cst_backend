package question_bank

import "w2w.io/cmn"

// 查询题库参数结构体
type QueryQuestionBankParams struct {
	BankID   int64  // 题库ID
	Keyword  string // 关键字
	Page     int64  // 分页页码
	PageSize int64  // 分页大小
	UserID   int64  // 用户ID
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

// AddQuestionRequest 添加题目请求结构
type AddQuestionRequest struct {
	BankID    int64           `json:"bank_id"`   // 题库ID
	Questions []cmn.TQuestion `json:"questions"` // 题目列表
}
