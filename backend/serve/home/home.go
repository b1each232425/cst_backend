// Package home management
package home

//annotation:static-service
//author:{"name":"home","tel":"18928776452","email":"XUnion@GMail.com"}

import (
	"encoding/json"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"os"
	"path/filepath"
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

	basePath, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		z.Error(err.Error())
		return
	}

	staticDocRoot := basePath + "/f"
	if viper.IsSet("webServe.home") {
		staticDocRoot = viper.GetString("webServe.home")
	}

	z.Info("this is static.Enroll called")
	_ = cmn.AddService(&cmn.ServeEndPoint{
		Path: "/",
		Name: "home",

		IsFileServe: true,

		PageRoute: true,

		DocRoot: staticDocRoot,

		Developer: developer,
		WhiteList: true,

		DomainID: int64(cmn.CDomainSys),

		DefaultDomain: int64(cmn.CDomainSys),
	})
}
