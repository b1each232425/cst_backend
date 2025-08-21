/*
 * @Author: zdl <1311866870@qq.com>
 * @Description: 练习管理所需模型
 * @Date: 2025-07-15 19:59:25
 * @LastEditors: zdl <1311866870@qq.com>
 * @LastEditTime: 2025-08-20 10:29:20
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
	Allow     string // 允许作答 00
	Forbid    string // 不允许作答 02
	Deleted   string // 删除 04
	Submitted string // 已提交 06
	Marked    string // 已批改出分 08
	Disabled  string // 作废状态 10
}{
	Allow:     "00",
	Forbid:    "02",
	Deleted:   "04",
	Submitted: "06",
	Marked:    "08",
	Disabled:  "10",
}

var WrongSubmissionStatus = struct {
	Allow     string // 允许作答 00
	Forbid    string // 不允许作答 02
	Submitted string // 已提交 04
	Deleted   string // 已删除 06
}{
	Allow:     "00",
	Forbid:    "02",
	Submitted: "04",
	Deleted:   "06",
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
	Disabled       string // 已作废 06
}{
	PendingRelease: "00",
	Released:       "02",
	Deleted:        "04",
	Disabled:       "06",
}

// MarkMode 练习批改模式
var MarkMode = struct {
	AI     string // AI批改 00
	Normal string // 手动批改 10

}{
	AI:     "00",
	Normal: "10",
}

// PracticeStudentStatus 练习学生参与状态
var PracticeStudentStatus = struct {
	Normal  string // 正常 00
	Deleted string // 被删除 02
}{
	Normal:  "00",
	Deleted: "02",
}

var PracticeDomainID = struct {
	Student    int64 // 学生 2008
	Teacher    int64 // 教师 2003
	SuperAdmin int64 // 超级管理员 2000
	Admin      int64 // 管理员 2001
}{
	Student:    2008,
	Teacher:    2003,
	SuperAdmin: 2000,
	Admin:      2001,
}

type practiceInfo struct {
	Practice cmn.TPractice `json:"practice"`
	Student  []int64       `json:"student,omitempty"`
}

type practiceStudent struct {
	Pid     int64   `json:"practice_id" validate:"required,gt=0"`
	Student []int64 `json:"student" `
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
