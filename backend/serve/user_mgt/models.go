package user_mgt

import (
	"github.com/wneessen/go-mail"
	"go.uber.org/zap"
	"w2w.io/cmn"
	"w2w.io/null"
)

var z *zap.Logger
var mailServer *EmailServer
var mailClient *mail.Client

const (
	AccountLength = 12          // 账号长度
	InitialPwd    = "abc123456" // 初始密码
	DefaultRegion = "CN"        // 默认地区
)

const (
	CUserStatusValid         = "00" // 有效
	CUserStatusProhibitLogin = "02" // 禁止登录
	CUserStatusBlock         = "04" // 封禁
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
	Domains       []null.String         `json:"Domains"`
	DomainObjects []cmn.TDomain         `json:"DomainObjects"`
	APIs          []cmn.TVUserDomainAPI `json:"APIs"`
	ErrorMsg      []null.String         `json:"ErrorMsg"` // 错误信息
}

type EmailServer struct {
	Host   string
	Port   int
	User   string
	Pwd    string
	Sender string
}

// IDCardFile 用户证件文件
type IDCardFile struct {
	FrontImgID string `json:"frontImgID"` // 正面图片
	BackImgID  string `json:"backImgID"`  // 反面图片
}
