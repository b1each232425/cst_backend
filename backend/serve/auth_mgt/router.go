package auth_mgt

//annotation:auth-mgt-service
//author:{"name":"王皓","tel":"15819888226", "email":"xavier.wang.work@icloud.com"}

import (
	"encoding/json"

	"go.uber.org/zap"
	"w2w.io/cmn"
)

var z *zap.Logger

func init() {
	//Setup package scope variables, just like logger, db connector, configure parameters, etc.
	cmn.PackageStarters = append(cmn.PackageStarters, func() {
		z = cmn.GetLogger()
		z.Info("message zLogger settled")
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
		Fn: handler.HandleQuerySelectableAPIs,

		Path: "/auth/selectable-apis",
		Name: "查询可选API",

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "权限管理.查询可选API",
				AccessAction: CAPIAccessActionRead,
				Configurable: false,
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
		Fn: handler.HandleDomain,

		Path: "/auth/domain",
		Name: "域管理",

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "权限管理.创建域",
				AccessAction: CAPIAccessActionCreate,
				Configurable: false,
			},
			{
				Name:         "权限管理.查询域",
				AccessAction: CAPIAccessActionRead,
				Configurable: false,
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
		Fn: handler.HandleGetCurrentUserAuthority,

		Path: "/auth/authority/me",
		Name: "查询当前用户权限信息",

		ApiEntries: []*cmn.EndPointApiEntries{
			{
				Name:         "权限管理.查询当前用户权限信息",
				AccessAction: CAPIAccessActionRead,
				Configurable: false,
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
