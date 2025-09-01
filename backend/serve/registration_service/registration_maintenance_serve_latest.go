package registration_service

//annotation:register-service
//author:{"name":"LilEYi","tel":"13535215794", "email":"3102128343@qq.com"}

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"sync"
	"time"
	"w2w.io/cmn"
)

const (
	//事件类型
	EVENT_TYPE_REGISTRATION_START = "registration_start"
	EVENT_TYPE_REGISTRATION_END   = "registration_end"
	EVENT_TYPE_REVIEW_END         = "review_end"
	//默认最大并发数
	DEFAULT_MAX_WORKERS = 10
)

var (
	z       *zap.Logger
	pgxConn *pgxpool.Pool
)

// 事件数据结构
type RegistrationEvent struct {
	Type           string `json:"type"`
	RegistrationID int64  `json:"registration_id"`
}

// 定时器管理器
type RegistrationTimerManager struct {
	timers     map[string]*time.Timer
	mutex      *sync.Mutex
	ctx        context.Context
	cancel     context.CancelFunc
	eventQueue chan RegistrationEvent
	maxWorkers int //最大并发worker数量
}

func init() {
	//Setup package scope variables, just like logger, db connector, configure parameters, etc.
	cmn.PackageStarters = append(cmn.PackageStarters, func() {
		z = cmn.GetLogger()
		z.Info("registration_service zLogger settled")
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

	_ = cmn.AddService(&cmn.ServeEndPoint{
		Fn: func(ctx context.Context) {

		},

		Path: "/registration",
		Name: "registration",

		Developer: developer,
		WhiteList: true,

		//DomainID 创建该API的账号归属的domain
		DomainID: int64(cmn.CDomainSys),

		//DefaultDomain 该API将默认授权给的用户
		DefaultDomain: int64(cmn.CDomainSys),
	})

}
