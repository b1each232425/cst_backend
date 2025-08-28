package user_mgt

//annotation:user-mgt-service
//author:{"name":"王皓","tel":"15819888226", "email":"xavier.wang.work@icloud.com"}

import (
	"crypto/tls"
	"encoding/json"

	"github.com/spf13/viper"
	"github.com/wneessen/go-mail"
	"go.uber.org/zap"
	"w2w.io/cmn"
	"w2w.io/serve/auth_mgt"
)

var z *zap.Logger
var mailServer *EmailServer
var mailClient *mail.Client

const (
	AccountLength = 12          // 账号长度
	InitialPwd    = "abc123456" // 初始密码
	DefaultRegion = "CN"        // 默认地区
)

func init() {
	//Setup package scope variables, just like logger, db connector, configure parameters, etc.
	cmn.PackageStarters = append(cmn.PackageStarters, func() {
		z = cmn.GetLogger()
		z.Info("message zLogger settled")
	})

	cmn.PackageStarters = append(cmn.PackageStarters, func() {
		var err error

		mailServer = &EmailServer{}
		mailServer.Host = viper.GetString("email.host")
		mailServer.Port = viper.GetInt("email.port")
		mailServer.User = viper.GetString("email.user")
		mailServer.Pwd = viper.GetString("email.pwd")
		mailServer.Sender = viper.GetString("email.sender")
		if mailServer.Host == "" || mailServer.Port == 0 || mailServer.User == "" || mailServer.Pwd == "" || mailServer.Sender == "" {
			z.Error("email server configuration is incomplete, please check your config file")
		}

		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
		}

		mailClient, err = mail.NewClient(mailServer.Host, mail.WithPort(mailServer.Port), mail.WithSMTPAuth(mail.SMTPAuthPlain),
			mail.WithUsername(mailServer.User), mail.WithPassword(mailServer.Pwd), mail.WithTLSConfig(tlsConfig))
		if err != nil {
			z.Error("failed to create email client", zap.Error(err))
		}
	})
}

func Enroll(author string) {
	z.Info("message.Enroll called")
	var developer *cmn.ModuleAuthor
	if author != "" {
		var d cmn.ModuleAuthor
		err := json.Unmarshal([]byte(author), &d)
		if err != nil {
			z.Error(err.Error())
			return
		}
		developer = &d
	}

	handler := NewHandler()

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: handler.HandleUser,

		Path: "/user",
		Name: "用户管理",

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "用户管理.查询用户",
				AccessAction: auth_mgt.CAPIAccessActionRead,
				Configurable: true,
			},
			{
				Name:         "用户管理.创建用户",
				AccessAction: auth_mgt.CAPIAccessActionCreate,
				Configurable: true,
			},
		},

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: handler.HandleGetNewAccount,

		Path: "/user/new-account",
		Name: "获取新账号",

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "用户管理.获取新账号",
				AccessAction: auth_mgt.CAPIAccessActionRead,
				Configurable: true,
			},
		},

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: handler.HandleQueryMyInfo,

		Path: "/user/me",
		Name: "查询个人信息",

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "用户管理.查询个人信息",
				AccessAction: auth_mgt.CAPIAccessActionRead,
				Configurable: true,
			},
		},

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: handler.HandleSelectLoginDomain,

		Path: "/user/login-domain",
		Name: "选择可登录域",

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "用户管理.登录管理.选择可登录域",
				AccessAction: auth_mgt.CAPIAccessActionRead,
				Configurable: true,
			},
		},

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: handler.HandleValidateUserToBeInsert,

		Path: "/user/validate",
		Name: "验证用户信息",

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "用户管理.验证用户信息",
				AccessAction: auth_mgt.CAPIAccessActionRead,
				Configurable: true,
			},
		},

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: handler.HandleLogout,

		Path: "/user/logout",
		Name: "用户登出",

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "用户管理.登录管理.用户登出",
				AccessAction: auth_mgt.CAPIAccessActionUpdate,
				Configurable: true,
			},
		},

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: handler.HandleSendValidationCodeEmail,

		Path: "/user/verification-code/email",
		Name: "发送邮箱验证码",

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "用户管理.登录管理.发送邮箱验证码",
				AccessAction: auth_mgt.CAPIAccessActionCreate,
				Configurable: true,
			},
		},

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: handler.HandleRegisterByEmail,

		Path: "/user/register/email",
		Name: "邮箱注册",

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "用户管理.登录管理.邮箱注册",
				AccessAction: auth_mgt.CAPIAccessActionCreate,
				Configurable: true,
			},
		},

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})
}
