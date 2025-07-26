package time_sync

//annotation:time-sync-service
//author:{"name":"王皓","tel":"15819888226", "email":"xavier.wang.work@icloud.com"}

import (
	"context"
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/panjf2000/ants/v2"
	"github.com/spf13/viper"
	"time"

	"go.uber.org/zap"
	"w2w.io/cmn"
)

const (
	releaseTimeout = 5 * time.Second // 释放超时时间
)

var _ Service = (*serviceImpl)(nil)

var (
	z   *zap.Logger
	Srv Service // 全局服务实例
)

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

	upgrader := websocket.Upgrader{}

	// 获取时间同步间隔
	timeSyncInterval := viper.GetInt("timeSync.timeSyncInterval")
	if timeSyncInterval <= 0 {
		z.Fatal("timeSyncInterval must be greater than 0")
	}
	timeSyncIntervalMillisecond := time.Duration(timeSyncInterval) * time.Millisecond

	// 初始化 ants goroutine 池
	poolInitialSize := viper.GetInt("timeSync.poolInitialSize")
	if poolInitialSize <= 0 {
		z.Fatal("poolInitialSize must be greater than 0")
	}
	pool, err := ants.NewPool(poolInitialSize)
	if err != nil {
		z.Fatal("Failed to create ants pool", zap.Error(err))
	}

	// 初始化时间同步服务
	Srv, err = NewService(timeSyncIntervalMillisecond, pool, upgrader)
	if err != nil {
		z.Fatal("Failed to create time sync service", zap.Error(err))
	}

	// 开启时间同步服务
	go Srv.startServe(context.Background())

	z.Info("time sync service init success", zap.Int("timeSyncInterval", timeSyncInterval), zap.Int("poolInitialSize", poolInitialSize))

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: Srv.handleInitTimeSyncConn,

		Path: "/time-sync",
		Name: "TimeSync",

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})
}
