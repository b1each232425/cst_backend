package question_bank

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

func isAllowedDomain(domain string) bool {
	_, exists := allowedDomains[domain]
	return exists
}

var QuestionTypes = map[string]string{
	QuestionTypeSingleChoice:   "单选题",
	QuestionTypeMultipleChoice: "多选题",
	QuestionTypeTrueFalse:      "判断题",
	QuestionTypeFillInBlank:    "填空题",
	QuestionTypeEssay:          "简答题",
	"test":                     "测试所用",
}

var QuestionDifficulty = map[int64]string{
	1: "简单",
	2: "中等",
	3: "困难",
}

const (
	// 记录状态定义
	StatusNormal   = "00" // 正常状态
	StatusUnNormal = "02" // 已删除(软删除)
)
