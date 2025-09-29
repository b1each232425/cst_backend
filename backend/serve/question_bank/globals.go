package question_bank

import "time"

const (

	// Question bank types
	QuestionBankTypeTheory = "00" // 理论题库
	QuestionBankTypeCoding = "02" // 编程题库
)

const (
	// Question types
	QuestionTypeSingleChoice   = "00" // 单选题
	QuestionTypeMultipleChoice = "02" // 多选题
	QuestionTypeTrueFalse      = "04" // 判断题
	QuestionTypeFillInBlank    = "06" // 填空题
	QuestionTypeEssay          = "08" // 简答题
	QuestionTypeComprehensive  = "10" // 综合应用题
	QuestionTypeExercise       = "12" // 综合演练题
)

const (
	// allowDomain
	DomainSuperAdmin          = "cst.school^superAdmin"           // 超级管理员
	DomainAdmin               = "cst.school^admin"                // 管理员
	DomainAcademicAffairAdmin = "cst.school.academicAffair^admin" // 考务员
	DomainTeacher             = "cst.school^teacher"              // 教师
	DomainStudent             = "cst.school^student"              // 学生
)

var allowedDomains = map[string]struct{}{
	DomainSuperAdmin:          {},
	DomainAdmin:               {},
	DomainAcademicAffairAdmin: {},
	DomainTeacher:             {},
}

var QuestionTypes = map[string]string{
	QuestionTypeSingleChoice:   "单选题",
	QuestionTypeMultipleChoice: "多选题",
	QuestionTypeTrueFalse:      "判断题",
	QuestionTypeFillInBlank:    "填空题",
	QuestionTypeEssay:          "简答题",
	QuestionTypeComprehensive:  "综合应用题",
	QuestionTypeExercise:       "综合演练题",
	"test":                     "测试所用",
}

var QuestionDifficulty = map[string]string{
	"00": "易",
	"02": "较易",
	"04": "中",
	"06": "较难",
	"08": "难",
}

const (
	// 记录状态定义
	StatusNormal   = "00" // 正常状态
	StatusUnNormal = "02" // 已删除(软删除)
)

const (
	// 题目锁相关常量
	QuestionLockPrefix     = "question_lock:" // 题目锁前缀
	QuestionLockExpiration = 30 * time.Minute // 题目锁过期时间（秒）
)
