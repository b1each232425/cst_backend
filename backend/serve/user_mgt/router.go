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
		Name: "User Management Service",

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
		Name: "Get New Account",

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
		Name: "Query My Info",

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
		Name: "Select Login Domain",

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
		Name: "Validate User Info To be Insert",

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
		Name: "Logout",

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
		Name: "Get Email Verification Code",

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
		Name: "Register By Email",

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})
}
