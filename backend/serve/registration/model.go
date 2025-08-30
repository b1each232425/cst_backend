package registration

import "w2w.io/cmn"

var RegisterDomainID = struct {
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

var RegisterCourse = struct {
	ALL       string
	Theory    string
	Practical string
}{
	ALL:       "00",
	Theory:    "02",
	Practical: "04",
}

// 考试状态
var ExamType = struct {
	Normal string // 正常考试 00
	Retake string // 补考 02
}{
	Normal: "00",
	Retake: "02",
}
var RegisterType = struct {
	Once   string // 自报名 00
	Import string // 人工导入 02
}{
	Once:   "00",
	Import: "02",
}

// 报名计划状态
var RegisterStatus = struct {
	Released       string // 已发布 00
	PendingRelease string // 未发布 02
	Ending         string //已结束 04
	ReviewEnding   string //审核截止 06
	Disabled       string // 已作废 08
	Deleted        string // 已删除 10
	Cancel         string // 已取消 12
}{
	Released:       "00",
	PendingRelease: "02",
	Ending:         "04",
	ReviewEnding:   "06",
	Disabled:       "08",
	Deleted:        "10",
	Cancel:         "12",
}

// 报名计划练习状态
var RegisterPracticeStatus = struct {
	Normal string
	Delete string
}{
	Normal: "00",
	Delete: "02",
}

// 报名计划学生状态
var RegisterStudentStatus = struct {
	Apply    string //报名中 00
	Pending  string //待审核 02
	Approved string //审核通过 04
	Rejected string //审核未通过 06
	Deleted  string //已删除 08
}{
	Apply:    "00",
	Pending:  "02",
	Approved: "04",
	Rejected: "06",
	Deleted:  "08",
}

type registerStudentType struct {
	StudentID int64
	ExamType  string
}
type registerStudent struct {
	RegisterID int64                 `json:"register_id" validate:"required,gt=0"`
	Student    []registerStudentType `json:"student" `
}
type registerStudentOnce struct {
	RegisterID int64  `json:"register_id" validate:"required,gt=0"`
	Status     string `json:"status"`
}
type RegisterInfo struct {
	Registration *cmn.TRegisterPlan
	PracticeIds  []int64
}
