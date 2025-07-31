package user_mgt

import (
	"w2w.io/cmn"
	"w2w.io/null"
)

const (
	DomainSuperAdmin          = "cst.school^superAdmin"
	DomainAdmin               = "cst.school^admin"
	DomainAcademicAffairAdmin = "cst.school.academicAffair^admin"
	DomainTeacher             = "cst.school^teacher"
	DomainExamSupervisor      = "cst.school^examSupervisor"
	DomainExamGrader          = "cst.school^examGrader"
	DomainExamSiteAdmin       = "cst.school.examSite^admin"
	DomainScoreChecker        = "cst.school^scoreChecker"
	DomainStudent             = "cst.school^student"
)

var (
	Domains = []string{
		DomainSuperAdmin,
		DomainAdmin,
		DomainAcademicAffairAdmin,
		DomainTeacher,
		DomainExamSupervisor,
		DomainExamGrader,
		DomainExamSiteAdmin,
		DomainScoreChecker,
		DomainStudent,
	}
)

var domainSet = func() map[string]struct{} {
	m := make(map[string]struct{}, len(Domains))
	for _, d := range Domains {
		m[d] = struct{}{}
	}
	return m
}()

// QueryUsersFilter 查询用户列表的过滤条件
type QueryUsersFilter struct {
	Account    null.String `json:"account"`    // 用户账号
	Name       null.String `json:"name"`       // 用户姓名
	Phone      null.String `json:"phone"`      // 用户电话
	Email      null.String `json:"email"`      // 用户邮箱
	Gender     null.String `json:"gender"`     // 用户性别
	Status     null.String `json:"status"`     // 用户状态
	CreateTime null.Int    `json:"createTime"` // 用户创建时间
	Domain     null.String `json:"domain"`     // 用户所属域
}

// InvalidUser 不能被插入的无效用户
type InvalidUser struct {
	Account      null.String   `json:"account"`      // 用户账号
	OfficialName null.String   `json:"officialName"` // 用户姓名
	MobilePhone  null.String   `json:"mobilePhone"`  // 用户电话
	Email        null.String   `json:"email"`        // 用户邮箱
	IDCardNo     null.String   `json:"idCardNo"`     // 用户证件号
	ErrorMsg     []null.String `json:"errorMsg"`     // 错误信息
}

type User struct {
	cmn.TUser
	Domains []null.String `json:"Domains"`
}
