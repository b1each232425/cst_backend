// Package coursefe course frontend
package coursefe

//annotation:course-fe-service
//author:{"name":"kzz","tel":"18928776452","email":"XUnion@GMail.com"}

import (
	"encoding/json"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"w2w.io/cmn"
)

var z *zap.Logger

func init() {
	//Setup package scope variables, just like logger, db connector, configure parameters, etc.
	cmn.PackageStarters = append(cmn.PackageStarters, func() {
		z = cmn.GetLogger()
		z.Info("user zLogger settled")
	})
}

func Enroll(author string) {
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

	courFeStorePath := "/var/deploy/i3l/fe/course"
	key := "webServe.coursefe"
	if viper.IsSet(key) {
		courFeStorePath = viper.GetString(key)
	}

	storePath := courFeStorePath
	z.Info("using course store path: ", zap.String("storePath", storePath))
	_ = cmn.AddService(&cmn.ServeEndPoint{
		Path: "/coursefe",
		Name: "coursefe",

		IsFileServe: true,

		PageRoute: true,

		DocRoot: storePath,

		Developer: developer,
		WhiteList: true,

		DomainID:      int64(cmn.CDomainSys),
		DefaultDomain: int64(cmn.CDomainSys),
	})
}
