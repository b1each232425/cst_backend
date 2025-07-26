/*
 * @Author: zdl <1311866870@qq.com>
 * @Description: 练习管理所需模型
 * @Date: 2025-07-15 19:59:25
 * @LastEditors: zdl <1311866870@qq.com>
 * @LastEditTime: 2025-07-26 23:04:36
 */
package practice_mgt

import (
	"w2w.io/cmn"
	"w2w.io/null"
)

// PracticeType 练习类型定义
var PracticeType = struct {
	Classical   string // 经典巩固 00
	PracticeNew string // 常练常新 02
	Intelligent string // 智能提升 04
}{
	Classical:   "00",
	PracticeNew: "02",
	Intelligent: "04",
}

// PracticeDifficulty 练习难度定义
var PracticeDifficulty = struct {
	Simple string // 简单 00
	Medium string // 中等 02
	Hard   string // 困难 04
}{
	Simple: "00",
	Medium: "02",
	Hard:   "04",
}

// PracticeSubmissionStatus 练习状态
var PracticeSubmissionStatus = struct {
	Allow   string // 允许作答 00
	Forbid  string // 不允许作答 02
	Deleted string // 删除 04
}{
	Allow:   "00",
	Forbid:  "02",
	Deleted: "04",
}

var StudentSubmissionStatus = struct {
	Submitted   string // 00 所有练习记录已提交
	UnSubmitted string // 02 上次练习记录未提交
	NeverAnswer string // 04 以前从来没有作答过
}{
	Submitted:   "00",
	UnSubmitted: "02",
	NeverAnswer: "04",
}

// PracticeStatus 练习状态
var PracticeStatus = struct {
	PendingRelease string // 未发布 00
	Released       string // 已发布 02
	Deleted        string // 已删除 04
}{
	PendingRelease: "00",
	Released:       "02",
	Deleted:        "04",
}

// MarkMode 练习批改模式
var MarkMode = struct {
	Normal string // 手动批改 00
	AI     string // AI批改 02

}{
	Normal: "00",
	AI:     "02",
}

// PracticeStudentStatus 练习学生参与状态
var PracticeStudentStatus = struct {
	Normal  string // 正常 00
	Deleted string // 被删除 02
}{
	Normal:  "00",
	Deleted: "02",
}

type practiceInfo struct {
	Practice cmn.TPractice `json:"practice"`
	Student  []int64       `json:"student"`
}

type EnterPracticeInfo struct {
	PracticeSubmissionID int64   `json:"PracticeSubmissionID"`
	PaperName            string  `json:"PaperName,omitempty"`
	Duration             int64   `json:"Duration,omitempty"`
	TotalScore           float64 `json:"TotalScore,omitempty"`
	QuestionCount        int64   `json:"QuestionCount,omitempty" `
	GroupCount           int64   `json:"GroupCount,omitempty"`
}

type StudentInfo struct {
	ID           int64       `json:"id"`            // 学生ID
	Account      null.String `json:"account"`       // 学生账号
	OfficialName null.String `json:"official_name"` // 学生姓名
	Password     null.String `json:"password"`      // 学生密码
	IdCardNo     null.String `json:"id_card_no" `   // 学生身份证号
	Phone        null.String `json:"phone"`         // 学生电话
}
