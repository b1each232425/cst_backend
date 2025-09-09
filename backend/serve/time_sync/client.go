package time_sync

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"w2w.io/serve/paper_respondence"
)

type ReceiveMsg struct {
	MsgType int64 `json:"msg_type"` // 消息类型
}

type SendMsg struct {
	MsgType   int64 `json:"msg_type"`  // 消息类型
	EndTime   int64 `json:"end_time"`  // 考试结束时间
	Timestamp int64 `json:"timestamp"` // 当前时间戳
}

var _ Connection = (*Conn)(nil)

var (
	timeDeadline = 20 * time.Second // 心跳超时时间
)

const (
	TypeTimestamp   = 1 // 时间戳消息类型
	TypeExamEndtime = 2 // 考试结束时刻消息类型
)

type Connection interface {
	ListenMessage()                                     // 监听消息
	SendMessage(message interface{}) error              // 发送消息
	StartHeartbeatDetection()                           // 心跳检测
	Close()                                             // 关闭连接
	GeConnId() string                                   // 获取连接ID
	GetExamineeId() int64                               // 获取考生ID
	SetExamineeID(examineeId int64)                     // 设置考生ID
	SetPracticeSubmissionID(practiceSubmissionId int64) // 设置练习提交ID
	SetWrongSubmissionID(id int64)                      // 设置练习错题作答提交ID
}
type Conn struct {
	ID                   string // 连接ID
	userID               int64  // 用户ID
	ExamineeID           int64  // 考生ID
	PracticeSubmissionID int64  // 练习提交ID
	WrongSubmissionID    int64  // 练习错题作答提交ID
	logger               *zap.Logger
	wsConn               *websocket.Conn
	unregisterChan       chan Connection // 用于注销连接的通道
	mutex                sync.Mutex
}

// NewConnection 创建新的连接实例
func NewConnection(id string, conn *websocket.Conn, logger *zap.Logger, unregisterChan chan Connection, userId int64) Connection {
	return &Conn{
		ID:             id,
		userID:         userId,
		wsConn:         conn,
		logger:         logger,
		unregisterChan: unregisterChan,
		mutex:          sync.Mutex{},
	}
}

// SetExamineeID 设置考生ID
func (conn *Conn) SetExamineeID(id int64) {
	conn.ExamineeID = id
}

// SetPracticeSubmissionID 设置练习提交ID
func (conn *Conn) SetPracticeSubmissionID(id int64) {
	conn.PracticeSubmissionID = id
}

// SetWrongSubmissionID 设置练习错题作答提交ID
func (conn *Conn) SetWrongSubmissionID(id int64) {
	conn.WrongSubmissionID = id
}

// ListenMessage 开启消息监听
// 开启监听后会自动处理心跳 Pong 消息
func (conn *Conn) ListenMessage() {
	// 收到心跳 Pong 时，重置读取超时时间
	conn.wsConn.SetPongHandler(func(appData string) error {
		err := conn.wsConn.SetReadDeadline(time.Now().Add(timeDeadline))
		if err != nil {
			conn.logger.Error("error setting read deadline", zap.Error(err))
			return err
		}
		return nil
	})

	defer func() {
		if conn.wsConn != nil {
			err := conn.wsConn.Close()
			if err != nil {
				conn.logger.Error("error closing connection", zap.Error(err))
				return
			}
		}

		// 检查是否是练习或考试，是的话就调用退出处理服务
		if conn.PracticeSubmissionID > 0 || conn.ExamineeID > 0 || conn.WrongSubmissionID > 0 {

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			existReq := paper_respondence.ExitReq{
				PracticeSubmissionID: conn.PracticeSubmissionID,
				ExamineeID:           conn.ExamineeID,
				StudentId:            conn.userID,
				WrongSubmissionID:    conn.WrongSubmissionID,
			}

			err := paper_respondence.HandleExit(ctx, existReq)
			if err != nil {
				conn.logger.Error("error handling paper exit", zap.Error(err))
			}
		}
	}()

	// 监听消息
	for {
		_, message, err := conn.wsConn.ReadMessage()
		if err != nil {
			conn.logger.Error("error reading message", zap.Error(err))
			conn.unregisterChan <- conn // 监听消息失败则注销连接
			return
		}

		var msg ReceiveMsg
		if err = json.Unmarshal(message, &msg); err != nil {
			conn.logger.Error("error unmarshal message", zap.Error(err))
			continue
		}

		if msg.MsgType == TypeTimestamp {
			m := SendMsg{
				MsgType:   TypeTimestamp,
				Timestamp: time.Now().UnixMilli(),
			}
			// 发送时间戳消息
			if err = conn.SendMessage(m); err != nil {
				conn.logger.Error("error sending timestamp message", zap.Error(err))
				continue
			}
		}
	}
}

// StartHeartbeatDetection 开启心跳检测
func (conn *Conn) StartHeartbeatDetection() {
	//设置超时
	err := conn.wsConn.SetReadDeadline(time.Now().Add(timeDeadline))
	if err != nil {
		conn.logger.Error("error setting read deadline", zap.Error(err))
		return
	}

	ticker := time.NewTicker(2 * time.Second)

	defer func() {
		ticker.Stop()
		if conn.wsConn != nil {
			err = conn.wsConn.Close()
			if err != nil {
				conn.logger.Error("error closing connection", zap.Error(err))
				return
			}
		}
	}()

	for {
		select {
		case <-ticker.C:
			err = conn.sendPing()
			if err != nil {
				conn.logger.Error("error sending ping", zap.Error(err))
				return
			}
		}
	}
}

// SendMessage 发送消息
func (conn *Conn) SendMessage(message interface{}) error {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()
	err := conn.wsConn.SetWriteDeadline(time.Now().Add(timeDeadline))
	if err != nil {
		conn.logger.Error("error setting write deadline", zap.Error(err))
		return err
	}
	return conn.wsConn.WriteJSON(message)
}

// sendPing 发送心跳包
func (conn *Conn) sendPing() error {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()
	if err := conn.wsConn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(timeDeadline)); err != nil {
		conn.logger.Error("error writing ping message", zap.Error(err))
		return err
	}
	return nil
}

// Close 关闭连接
func (conn *Conn) Close() {
	err := conn.wsConn.Close()
	if err != nil {
		conn.logger.Error("error closing connection", zap.Error(err))
		return
	}
}

// GeConnId 获取连接ID
func (conn *Conn) GeConnId() string {
	return conn.ID
}

// GetExamineeId 获取考生ID
func (conn *Conn) GetExamineeId() int64 {
	return conn.ExamineeID
}
