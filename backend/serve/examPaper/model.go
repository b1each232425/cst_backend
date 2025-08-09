/*
 * @Author: zdl <1311866870@qq.com>
 * @Description: 考卷-答卷模型
 * @Date: 2025-07-21 13:14:44
 * @LastEditors: zdl <1311866870@qq.com>
 * @LastEditTime: 2025-07-27 22:26:27
 */
package examPaper

import (
	"github.com/jmoiron/sqlx/types"
	"w2w.io/cmn"
	"w2w.io/null"
)

type ExamPaper struct {
	ID            null.Int    `json:"ID,omitempty" db:"id,true,integer"`
	ExamSessionID null.Int    `json:"ExamSessionID" db:"exam_session_id,false,bigint"`
	PracticeID    null.Int    `json:"PracticeID" db:"practice_id,false,bigint"`
	Name          null.String `json:"Name" db:"name,false,character varying"`
	TotalScore    null.Float  `json:"TotalScore" db:"total_score,double precision"`
}

type SubjectiveQuestionGroup struct {
	GroupID     int64   `json:"group_id"`
	QuestionIDs []int64 `json:"question_ids"`
}

// NonSelectQuestionAnswer 填空题/简答题答案结构
type NonSelectQuestionAnswer struct {
	Index             int      `json:"index"`  // 答案索引
	Answer            string   `json:"answer"` // 答案
	AlternativeAnswer []string `json:"alternative_answer"`
	Score             float64  `json:"score"`        // 分数
	GradingRule       string   `json:"grading_rule"` // 评分规则
}

var QuestionCategory = struct {
	SingleChoice string // 单选 00
	MultiChoice  string // 多选 02
	TrueFalse    string // 判断 04
	FillInBlank  string //填空 06
	ShortAnswer  string // 简答 08

}{
	SingleChoice: "00",
	MultiChoice:  "02",
	TrueFalse:    "04",
	FillInBlank:  "06",
	ShortAnswer:  "08",
}

var PaperCategory = struct {
	Exam     string // 考试用途 00
	Practice string // 练习用途 02
}{
	Exam:     "00",
	Practice: "02",
}

var PaperStatus = struct {
	Normal  string // 正常 00
	Invalid string // 异常 02

}{
	Normal:  "00",
	Invalid: "02",
}

// Group 学生试卷模版题组结构（试卷模块自用：解析存储在视图的JSON实体）
type Group struct {
	cmn.TPaperGroup
	Questions []Question `json:"questions"`
}

// Question 试卷模版中的题目（加上老师字修改的分数）
/*
特别说明：
	ID字段为试卷题目ID
	BankQuestionID字段为题库内题目ID
*/
type Question struct {
	cmn.TQuestion
	SubScore       []float64 `json:"sub_score" db:"sub_score,true,integer"`
	BankQuestionID null.Int  `json:"BankQuestionID,omitempty" db:"bank_question_id,true,integer"`
}

// ExamGroup 学生试卷题题组结构（考卷模块自用：解析存储在视图的JSON实体）
type ExamGroup struct {
	cmn.TExamPaperGroup
	Questions []ExamQuestion `json:"questions"`
}

// ExamQuestion 学生试卷题目结构
type ExamQuestion struct {
	cmn.TExamPaperQuestion
	AnswerNum     int            `json:"AnswerNum"`
	StudentAnswer types.JSONText `json:"StudentAnswer"`
	StudentScore  null.Float     `json:"StudentScore"`
}

// GenerateAnswerQuestionsRequest 考试/练习生成答卷
type GenerateAnswerQuestionsRequest struct {
	ExamPaperID int64  `json:"exam_paper_id" validate:"required,gt=0"`
	Category    string `json:"category" validate:"required,oneof=00 02"`

	// 考试场景：Category == "00" 时必填，并且每个元素都要大于 0
	ExamineeIDs []int64 `json:"examinee_ids" validate:"required_if=Category 00,dive,gt=0"`

	// 练习场景：Category == "02" 时必填，并且每个元素都要大于 0
	PracticeSubmissionID []int64 `json:"practice_submission_id" validate:"required_if=Category 02,dive,gt=0"`

	// 布尔类型不需要额外标签，bool 默认就只有 true/false
	IsQuestionRandom bool  `json:"is_question_random"`
	IsOptionRandom   bool  `json:"is_option_random"`
	Attempt          int64 `json:"attempt" validate:"omitempty,min=1"`
}

// QuestionOption 处理乱序
type QuestionOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}
