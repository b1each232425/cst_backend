package cmn

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// TaskPayload 定义任务载荷的基础结构
type TaskPayload struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	UserID    int64       `json:"user_id,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

// AsynqManager asynq 管理器
type AsynqManager struct {
	client *asynq.Client
	server *asynq.Server
	mux    *asynq.ServeMux
	logger *zap.Logger
}

var asynqManager *AsynqManager

// InitAsynq 初始化 asynq
func InitAsynq(redisAddr, redisPassword string, redisDB int) error {
	redisOpt := asynq.RedisClientOpt{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	}

	// 创建客户端
	client := asynq.NewClient(redisOpt)

	// 创建服务器
	server := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: 10,
		Queues: map[string]int{
			"critical": 6,
			"default":  3,
			"low":      1,
		},
	})

	// 创建多路复用器
	mux := asynq.NewServeMux()

	asynqManager = &AsynqManager{
		client: client,
		server: server,
		mux:    mux,
		logger: z,
	}

	return nil
}

// GetAsynqManager 获取 asynq 管理器实例
func GetAsynqManager() *AsynqManager {
	return asynqManager
}

// EnqueueTask 将任务加入队列
// 常用 asynq.Option（来自 github.com/hibiken/asynq，常用）：
// asynq.Queue(name string)  指定任务进入哪个队列（如 "critical","default","low"）。
// asynq.ProcessIn(d time.Duration)  延迟 d 后执行（类似延时任务）。
// asynq.ProcessAt(t time.Time)  在指定时间点执行
// asynq.Timeout(d time.Duration)  任务执行超时时间（超过则认为失败）。
// asynq.MaxRetry(n int)  最大重试次数（失败后重试次数）。
// asynq.Retention(d time.Duration)  任务完成后保留结果的时间。
// asynq.Unique(ttl time.Duration)  保证任务在 ttl 时间内唯一，避免重复入队。
func (am *AsynqManager) EnqueueTask(taskType string, payload interface{}, opts ...asynq.Option) error {
	taskPayload := TaskPayload{
		Type:      taskType,
		Data:      payload,
		Timestamp: time.Now().Unix(),
	}

	payloadBytes, err := json.Marshal(taskPayload)
	if err != nil {
		am.logger.Error("Failed to marshal task payload", zap.Error(err))
		return err
	}

	task := asynq.NewTask(string(taskType), payloadBytes, opts...)

	_, err = am.client.Enqueue(task)
	if err != nil {
		am.logger.Error("Failed to enqueue task",
			zap.String("task_type", string(taskType)),
			zap.Error(err))
		return err
	}

	am.logger.Info("Task enqueued successfully",
		zap.String("task_type", string(taskType)))

	return nil
}

// RegisterHandler 注册任务处理器
func (am *AsynqManager) RegisterHandler(taskType string, handler func(context.Context, *asynq.Task) error) {
	am.mux.HandleFunc(string(taskType), handler)
	am.logger.Info("Task handler registered", zap.String("task_type", string(taskType)))
}

// StartServer 启动 asynq 服务器
func (am *AsynqManager) StartServer() error {
	am.logger.Info("Starting asynq server...")
	return am.server.Run(am.mux)
}

// StopServer 停止 asynq 服务器
func (am *AsynqManager) StopServer() {
	am.logger.Info("Stopping asynq server...")
	am.server.Shutdown()
}

// CloseClient 关闭 asynq 客户端
func (am *AsynqManager) CloseClient() error {
	return am.client.Close()
}

// ParseTaskPayload 解析任务载荷
func ParseTaskPayload(task *asynq.Task) (*TaskPayload, error) {
	var payload TaskPayload
	err := json.Unmarshal(task.Payload(), &payload)
	return &payload, err
}

// RegisterTaskHandler 注册任务处理器的便利函数
// 供各模块在Enroll函数中调用
func RegisterTaskHandler(taskType string, handler func(context.Context, *asynq.Task) error) {
	if asynqManager != nil {
		asynqManager.RegisterHandler(taskType, handler)
	}
}

// SendTask 发送任务的便利函数
// 供各模块发送任务时调用
func SendTask(taskType string, payload interface{}, opts ...asynq.Option) error {
	if asynqManager != nil {
		return asynqManager.EnqueueTask(taskType, payload, opts...)
	}
	return nil
}

// StartAsynqService 启动 asynq 服务并设置优雅关闭
// 该函数会自动处理启动、监听终止信号和优雅关闭
func StartAsynqService() {
	redisAddr := "localhost:6379" // 默认值
	redisPassword := "x2Jc5K^5"
	redisDB := 0
	host := "cst.gzhu.edu.cn"
	port := 16910
	var s string
	s = "dbms.redis.addr"
	if viper.IsSet(s) {
		host = viper.GetString(s)
	}
	s = "dbms.redis.port"
	if viper.IsSet(s) {
		port = viper.GetInt(s)
	}
	redisAddr = fmt.Sprintf("%s:%d", host, port)
	if viper.IsSet("dbms.redis.cert") {
		redisPassword = viper.GetString("dbms.redis.cert")
	}
	// 初始化 asynq
	z.Info(fmt.Sprintf("connecting redis to %s", redisAddr))
	if err := InitAsynq(redisAddr, redisPassword, redisDB); err != nil {
		z.Error("Failed to initialize asynq", zap.Error(err))
		return
	}

	z.Info("Asynq initialized successfully")

	// 启动 asynq 服务器
	go func() {
		if err := asynqManager.StartServer(); err != nil {
			z.Error("Asynq server failed", zap.Error(err))
		}
	}()
}

// StopAsynqService 停止 asynq 服务
// 供外部在收到停止信号时调用
func StopAsynqService() {
	if asynqManager == nil {
		return
	}

	z.Info("Shutting down asynq...")
	asynqManager.StopServer()

	if err := asynqManager.CloseClient(); err != nil {
		z.Error("Failed to close asynq client", zap.Error(err))
	} else {
		z.Info("Asynq shutdown completed")
	}
}
