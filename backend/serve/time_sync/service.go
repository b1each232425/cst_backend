package time_sync

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/rs/xid"
	"strconv"
	"time"
	"w2w.io/cmn"

	"github.com/panjf2000/ants/v2"
	"go.uber.org/zap"
)

type Service interface {
	SendResetExamEndTimeMsg(ctx context.Context, ids []int64) error // 发送重置考试结束时刻的消息

	startServe(ctx context.Context)                                                  // 启动时间同步服务
	handleInitTimeSyncConn(ctx context.Context)                                      // 处理初始化时间同步连接
	getClientByExamineeId(ctx context.Context, examineeId int64) (Connection, error) // 根据考生ID获取连接实例
	sendCurrentTimestampMsg(ctx context.Context, c Connection) error                 // 发送当前时间戳消息
	broadcastTimeSyncMsg(ctx context.Context)                                        // 广播时间同步消息
}

type serviceImpl struct {
	upgrader         websocket.Upgrader    // WebSocket升级器
	timeSyncInterval time.Duration         // 时间同步间隔
	clients          map[string]Connection // 客户端连接池，key为连接ID
	registerChanel   chan Connection       // 注册客户端连接管道
	unregisterChanel chan Connection       // 注销客户端连接管道
	pool             *ants.Pool            // ants协程池
	repo             Repo
}

func NewService(timeSyncInterval time.Duration, pool *ants.Pool, upgrader websocket.Upgrader) (Service, error) {
	if timeSyncInterval <= 0 {
		return nil, fmt.Errorf("invalid timeSyncInterval: %v, must be greater than 0", timeSyncInterval)
	}
	if pool == nil {
		return nil, fmt.Errorf("ants pool cannot be nil")
	}

	return &serviceImpl{
		upgrader:         upgrader,
		timeSyncInterval: timeSyncInterval,
		clients:          make(map[string]Connection),
		registerChanel:   make(chan Connection, 100),
		unregisterChanel: make(chan Connection, 100),
		pool:             pool,
		repo:             NewRepo(),
	}, nil
}

// StartServe 启动时间同步服务
func (srv *serviceImpl) startServe(ctx context.Context) {
	z.Info("---->" + cmn.FncName())

	forceErr, _ := ctx.Value("force-error").(string) // 用于强制执行错误处理代码

	timeTicker := time.NewTicker(srv.timeSyncInterval)

	defer timeTicker.Stop()

	defer func(pool *ants.Pool, timeout time.Duration) {
		err := pool.ReleaseTimeout(timeout)
		if err != nil || forceErr == "pool.ReleaseTimeout" {
			z.Error("error releasing ants pool", zap.Error(err))
		}
	}(srv.pool, releaseTimeout)

	for {
		select {
		case <-timeTicker.C: // 定时广播时间同步消息
			srv.broadcastTimeSyncMsg(ctx)

		case registerClient := <-srv.registerChanel: // 接收注册的客户端
			id := registerClient.GeConnId()
			z.Info("register client", zap.String("client id", id))
			srv.clients[id] = registerClient  // 将连接放入连接池
			go registerClient.ListenMessage() // 启动协程监听消息

		case unregisterClient := <-srv.unregisterChanel: // 接收注销的客户端
			id := unregisterClient.GeConnId()
			z.Info("unregister client", zap.String("client id", id))
			delete(srv.clients, id) // 从连接池中删除连接

		case <-ctx.Done():
			z.Info("context done, stopping time sync service")
			// 关闭所有连接
			for _, clientConn := range srv.clients {
				clientConn.Close()
			}
			srv.clients = make(map[string]Connection) // 清空连接池
			return
		}
	}
}

// HandleInitTimeSyncConn 处理初始化时间同步连接
func (srv *serviceImpl) handleInitTimeSyncConn(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	forceErr, _ := ctx.Value("force-error").(string) // 用于强制执行错误处理代码

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
	if err != nil || forceErr == "websocket.Upgrade" {
		q.Err = fmt.Errorf("error upgrading connection: %w", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	// 生成连接的唯一ID
	id := xid.New().String() + strconv.FormatInt(time.Now().UnixMilli(), 10)
	newClient := NewConnection(id, conn, z, srv.unregisterChanel, userId)

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
	err = srv.sendCurrentTimestampMsg(ctx, newClient)
	if err != nil {
		q.Err = fmt.Errorf("error sending current timestamp: %w", err)
		_ = newClient.SendMessage("error: " + q.Err.Error())
		newClient.Close()
		q.RespErr()
		return
	}
}

// broadcastTimeSyncMsg 广播时间同步消息
func (srv *serviceImpl) broadcastTimeSyncMsg(ctx context.Context) {
	z.Info("---->" + cmn.FncName())

	forceErr, _ := ctx.Value("force-error").(string) // 用于强制执行错误处理代码

	// 遍历连接池
	for _, wsClient := range srv.clients {
		// 提交任务到协程池
		err := srv.pool.Submit(func(client Connection) func() {
			return func() {
				err := srv.sendCurrentTimestampMsg(ctx, client)
				if err != nil || forceErr == "sendCurrentTimestampMsg" {
					z.Error("error sending timestamp", zap.Error(err))
					return
				}
			}
		}(wsClient))
		if err != nil || forceErr == "ants.Pool.Submit" {
			z.Error("error submit task to ants pool", zap.Error(err))
			continue
		}
	}
}

// sendCurrentTimestampMsg 发送当前时间戳消息
func (srv *serviceImpl) sendCurrentTimestampMsg(ctx context.Context, c Connection) error {
	z.Info("---->" + cmn.FncName())

	forceErr, _ := ctx.Value("force-error").(string) // 用于强制执行错误处理代码

	// 获取当前时间戳
	timestamp := time.Now().UnixMilli()

	// 发送消息
	msg := SendMsg{
		MsgType:   TypeTimestamp,
		Timestamp: timestamp,
	}
	err := c.SendMessage(msg)
	if err != nil || forceErr == "sendMessage" {
		return fmt.Errorf("error sending timestamp msg: %w", err)
	}

	return nil
}

// getClientByExamineeId 根据考生ID获取连接实例
func (srv *serviceImpl) getClientByExamineeId(ctx context.Context, examineeId int64) (Connection, error) {
	z.Info("---->" + cmn.FncName())

	for _, wsClient := range srv.clients {
		if wsClient.GetExamineeId() == examineeId {
			return wsClient, nil
		}
	}

	e := fmt.Errorf("no client found for examineeId: %d", examineeId)
	z.Error(e.Error())
	return nil, e
}

// SendResetExamEndTimeMsg 发送重置考试结束时刻的消息
// 该方法会从数据库中查询考生的实际考试结束时间，并将其发送给对应的WebSocket客户端，以与考点管理模块配合实现延长考试时长功能
func (srv *serviceImpl) SendResetExamEndTimeMsg(ctx context.Context, examineeIds []int64) error {
	z.Info("---->" + cmn.FncName())

	var err error

	forceErr, _ := ctx.Value("force-error").(string) // 用于强制执行错误处理代码

	for _, id := range examineeIds {
		if id <= 0 {
			e := fmt.Errorf("examineeId must be greater than 0, got: %v", id)
			z.Error(e.Error())
			return e
		}

		// 获取考生的实际考试结束时间
		var actualEndTime int64
		actualEndTime, err = srv.repo.QueryActualExamEndTime(id)
		if err != nil {
			return err
		}
		if actualEndTime <= time.Now().UnixMilli() {
			e := fmt.Errorf("actual exam end time is not greater than current time, examineeID: %v, actualEndTime: %v", id, actualEndTime)
			z.Error(e.Error())
			return e
		}

		// 获取WebSocket连接实例
		var wsClient Connection
		wsClient, err = srv.getClientByExamineeId(ctx, id)
		if err != nil {
			return err
		}

		// 发送消息
		msg := SendMsg{
			MsgType: TypeExamEndtime,
			EndTime: actualEndTime,
		}
		err = wsClient.SendMessage(msg)
		if err != nil || forceErr == "sendMessage" {
			e := fmt.Errorf("error sending reset end time message: %w, examineeID: %v", err, id)
			z.Error(e.Error())
			return e
		}
	}

	return nil
}
