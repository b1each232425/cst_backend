package question_bank

const (

	// Question bank types
	QuestionBankTypeTheory = "00" // 理论题库
	QuestionBankTypeCoding = "02" // 编程题库
)

const (
	GlobalLevelSign  = "00" // 全局权限
	NormalLevelSign  = "02" // 普通权限
	CurrentLevelSign = "00" // 当前权限(TODO: 目前未使用, 可能会在后续版本中使用, 仅作为占位符保留)
)

var QuestionTypes = map[string]string{
	"00": "单选题",
	"02": "多选题",
	"04": "判断题",
	"06": "填空题",
	"08": "简答题",
}

var QuestionDifficulty = map[int64]string{
	1: "简单",
	2: "中等",
	3: "困难",
}
