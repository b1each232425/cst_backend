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

type registerStudent struct {
	RegisterID int64   `json:"register_id" validate:"required,gt=0"`
	Student    []int64 `json:"student" `
}
type RegisterInfo struct {
	Registration *cmn.TRegisterPlan
	PracticeIds  []int64
}
