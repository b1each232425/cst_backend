package user_mgt

import (
	"w2w.io/cmn"
	"w2w.io/null"
)

// QueryUsersFilter 查询用户列表的过滤条件
type QueryUsersFilter struct {
	FuzzyCondition null.String `json:"fuzzyCondition"` // 模糊查询条件
	ID             null.Int    `json:"id"`             // 用户ID
	Gender         null.String `json:"gender"`         // 用户性别
	Status         null.String `json:"status"`         // 用户状态
	CreateTime     null.Int    `json:"createTime"`     // 用户创建时间
	Domain         null.String `json:"domain"`         // 用户所属域
}

type User struct {
	cmn.TUser
	Domains  []null.String         `json:"Domains"`
	APIs     []cmn.TVUserDomainAPI `json:"APIs"`
	ErrorMsg []null.String         `json:"ErrorMsg"` // 错误信息
}

type EmailServer struct {
	Host   string
	Port   int
	User   string
	Pwd    string
	Sender string
}
