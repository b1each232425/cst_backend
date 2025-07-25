package time_sync

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/rs/xid"
	"strconv"
	"time"
	"w2w.io/cmn"
	"w2w.io/serve/time_sync/client"

	"github.com/panjf2000/ants/v2"
	"go.uber.org/zap"
)

type Service interface {
	StartServe()                                // 启动时间同步服务
	GetClientsAmount() int                      // 获取客户端数量
	HandleInitTimeSyncConn(ctx context.Context) // 处理初始化时间同步连接
}

type serviceImpl struct {
	upgrader         websocket.Upgrader           // WebSocket升级器
	timeSyncInterval time.Duration                // 时间同步间隔
	clients          map[string]client.Connection // 客户端连接池，key为连接ID
	registerChanel   chan client.Connection       // 注册客户端连接管道
	unregisterChanel chan client.Connection       // 注销客户端连接管道
	pool             *ants.Pool                   // ants协程池
	repo             Repo
}

func NewService(timeSyncInterval time.Duration, pool *ants.Pool, upgrader websocket.Upgrader) Service {
	return &serviceImpl{
		upgrader:         upgrader,
		timeSyncInterval: timeSyncInterval,
		clients:          make(map[string]client.Connection),
		registerChanel:   make(chan client.Connection, 100),
		unregisterChanel: make(chan client.Connection, 100),
		pool:             pool,
		repo:             NewRepo(),
	}
}

// StartServe 启动时间同步服务
func (srv *serviceImpl) StartServe() {
	z.Info("---->" + cmn.FncName())

	timeTicker := time.NewTicker(srv.timeSyncInterval)

	defer timeTicker.Stop()

	defer func(pool *ants.Pool, timeout time.Duration) {
		err := pool.ReleaseTimeout(timeout)
		if err != nil {
			z.Error("error releasing ants pool", zap.Error(err))
		}
	}(srv.pool, releaseTimeout)

	for {
		select {
		case <-timeTicker.C: // 定时广播时间同步消息
			srv.broadcastTimeSyncMsg()

		case examineeId := <-cmn.ResetExamEndTimeChan: // 接收需要重置考试结束时刻的考生ID
			srv.sendResetExamEndTimeMsg(examineeId)

		case registerClient := <-srv.registerChanel: // 接收注册的客户端
			id := registerClient.GeConnId()
			z.Info("register client", zap.String("client id", id))
			srv.clients[id] = registerClient  // 将连接放入连接池
			go registerClient.ListenMessage() // 启动协程监听消息

		case unregisterClient := <-srv.unregisterChanel: // 接收注销的客户端
			id := unregisterClient.GeConnId()
			z.Info("unregister client", zap.String("client id", id))
			delete(srv.clients, id) // 从连接池中删除连接
		}
	}
}

// GetClientsAmount 获取客户端数量
func (srv *serviceImpl) GetClientsAmount() int {
	return len(srv.clients)
}

// HandleInitTimeSyncConn 处理初始化时间同步连接
func (srv *serviceImpl) HandleInitTimeSyncConn(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	examineeId := q.R.URL.Query().Get("examinee_id")
	practiceSubmissionId := q.R.URL.Query().Get("practice_submission_id")

	var examineeIdInt int
	var practiceSubmissionIdInt int
	var err error

	// 获取考生ID和练习提交ID
	if examineeId != "" {
		examineeIdInt, err = strconv.Atoi(examineeId)
		if err != nil {
			q.Err = fmt.Errorf("error parsing examinee_id: %w", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
	}
	if practiceSubmissionId != "" {
		practiceSubmissionIdInt, err = strconv.Atoi(practiceSubmissionId)
		if err != nil {
			q.Err = fmt.Errorf("error parsing practice_submission_id: %w", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
	}

	if q.SysUser == nil || !q.SysUser.ID.Valid {
		q.Err = fmt.Errorf("invalid user context, sysUser is nil or ID is not set")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	//获取学生ID
	userId := q.SysUser.ID.Int64
	if userId <= 0 {
		q.Err = fmt.Errorf("invalid user ID: %d", userId)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	// 升级HTTP连接到WebSocket
	conn, err := srv.upgrader.Upgrade(q.W, q.R, nil)
	if err != nil {
		q.Err = fmt.Errorf("error upgrading connection: %w", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	// 生成连接的唯一ID
	id := xid.New().String() + strconv.FormatInt(time.Now().UnixMilli(), 10)
	newClient := client.NewConnection(id, conn, z, srv.unregisterChanel, userId)

	// 启动心跳检测
	go newClient.StartHeartbeatDetection()

	// 设置考生ID和练习提交ID到客户端实例
	if examineeId != "" {
		newClient.SetExamineeID(int64(examineeIdInt))
	}
	if practiceSubmissionId != "" {
		newClient.SetPracticeSubmissionID(int64(practiceSubmissionIdInt))
	}
	if practiceSubmissionId == "" && examineeId == "" {
		q.Err = fmt.Errorf("examineeId and practiceSubmissionId are both empty")
		z.Error(q.Err.Error())
		_ = newClient.SendMessage("error: " + q.Err.Error())
		newClient.Close()
		q.RespErr()
		return
	}

	srv.registerChanel <- newClient

	// 发送时间同步消息
	err = srv.sendCurrentTimestampMsg(newClient)
	if err != nil {
		q.Err = fmt.Errorf("error sending current timestamp: %w", err)
		_ = newClient.SendMessage("error: " + q.Err.Error())
		newClient.Close()
		q.RespErr()
		return
	}
}

// broadcastTimeSyncMsg 广播时间同步消息
func (srv *serviceImpl) broadcastTimeSyncMsg() {
	z.Info("---->" + cmn.FncName())

	// 遍历连接池
	for _, wsClient := range srv.clients {
		// 提交任务到协程池
		err := srv.pool.Submit(func(client client.Connection) func() {
			return func() {
				err := srv.sendCurrentTimestampMsg(client)
				if err != nil {
					z.Error("error sending timestamp", zap.Error(err))
					return
				}
			}
		}(wsClient))
		if err != nil {
			z.Error("error submit task to ants pool", zap.Error(err))
			continue
		}
	}
}

// sendResetExamEndTimeMsg 发送重置考试结束时刻的消息
// 该方法会从数据库中查询考生的实际考试结束时间，并将其发送给对应的WebSocket客户端，以与考点管理模块配合实现延长考试时长功能
func (srv *serviceImpl) sendResetExamEndTimeMsg(examineeId int64) {
	z.Info("---->" + cmn.FncName())

	if examineeId <= 0 {
		z.Error("examineeId must be greater than 0")
		return
	}

	// 获取考生的实际考试结束时间
	actualEndTime, err := srv.repo.QueryActualExamEndTime(examineeId)
	if err != nil {
		z.Error("error querying actual exam end time", zap.Int64("examineeId", examineeId), zap.Error(err))
		return
	}
	if actualEndTime <= time.Now().UnixMilli() {
		z.Warn("actual exam end time is not greater than current time", zap.Int64("examineeId", examineeId), zap.Int64("actualEndTime", actualEndTime))
		return
	}

	// 获取WebSocket连接实例
	wsClient, err := srv.getClientByExamineeId(examineeId)
	if err != nil {
		z.Error("error getting WebSocket client by examineeId", zap.Int64("examineeId", examineeId), zap.Error(err))
		return
	}

	// 发送消息
	msg := client.SendMsg{
		MsgType: client.TypeExamEndtime,
		EndTime: actualEndTime,
	}
	err = wsClient.SendMessage(msg)
	if err != nil {
		z.Error("error sending timestamp", zap.Error(err))
		return
	}

	return
}

// sendCurrentTimestampMsg 发送当前时间戳消息
func (srv *serviceImpl) sendCurrentTimestampMsg(c client.Connection) error {
	z.Info("---->" + cmn.FncName())

	// 获取当前时间戳
	timestamp := time.Now().UnixMilli()

	// 发送消息
	msg := client.SendMsg{
		MsgType:   client.TypeExamEndtime,
		Timestamp: timestamp,
	}
	err := c.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("error sending timestamp msg: %w", err)
	}

	return nil
}

// getClientByExamineeId 根据考生ID获取连接实例
func (srv *serviceImpl) getClientByExamineeId(examineeId int64) (client.Connection, error) {
	z.Info("---->" + cmn.FncName())

	if examineeId <= 0 {
		return nil, fmt.Errorf("examineeId must be greater than 0")
	}

	for _, wsClient := range srv.clients {
		if wsClient.GetExamineeId() == examineeId {
			return wsClient, nil
		}
	}

	return nil, fmt.Errorf("no client found for examineeId: %d", examineeId)
}
