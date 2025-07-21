package user_mgt

import "w2w.io/null"

// QueryUsersFilter 查询用户列表的过滤条件
type QueryUsersFilter struct {
	Account    null.String `json:"account"`     // 用户账号
	Name       null.String `json:"name"`        // 用户姓名
	Phone      null.String `json:"phone"`       // 用户电话
	Email      null.String `json:"email"`       // 用户邮箱
	Gender     null.String `json:"gender"`      // 用户性别
	Status     null.String `json:"status"`      // 用户状态
	CreateTime null.Int    `json:"create_time"` // 用户创建时间
	Roles      []null.Int  `json:"roles"`       // 用户角色
}
