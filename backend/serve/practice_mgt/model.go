/*
 * @Author: zdl <1311866870@qq.com>
 * @Description: 练习管理所需模型
 * @Date: 2025-07-15 19:59:25
 * @LastEditors: zdl <1311866870@qq.com>
 * @LastEditTime: 2025-07-24 12:03:22
 */
package practice_mgt

import (
	"w2w.io/cmn"
	"w2w.io/null"
)

var PracticeStatus = struct {
	PendingRelease string // 未发布 00
	Released       string // 已发布 02
	Deleted        string // 已删除 04
}{
	PendingRelease: "00",
	Released:       "02",
	Deleted:        "04",
}
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
