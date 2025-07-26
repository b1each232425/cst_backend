package time_sync

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"w2w.io/serve/time_sync/client"

	"github.com/gorilla/websocket"
	"w2w.io/cmn"
	"w2w.io/null"
)

func TestMain(m *testing.M) {
	// 配置测试环境
	cmn.Configure()

	// 运行测试
	code := m.Run()

	os.Exit(code)
}

// Test_NewService 测试 NewService 函数
func Test_NewService(t *testing.T) {
	tests := []struct {
		name             string
		timeSyncInterval time.Duration
		pool             *ants.Pool
		upgrader         websocket.Upgrader
		wantNil          bool
		wantErr          bool
		desc             string
	}{
		{
			name:             "正常创建服务实例",
			timeSyncInterval: 5 * time.Second,
			pool:             createTestPool(t, 10),
			upgrader:         websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
			wantNil:          false,
			desc:             "使用有效参数创建服务实例",
		},
		{
			name:             "零时间间隔",
			timeSyncInterval: 0,
			pool:             createTestPool(t, 10),
			upgrader:         websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
			wantNil:          false,
			wantErr:          true,
			desc:             "使用零时间间隔创建服务实例",
		},
		{
			name:             "负数时间间隔",
			timeSyncInterval: -1 * time.Second,
			pool:             createTestPool(t, 10),
			upgrader:         websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
			wantNil:          false,
			wantErr:          true,
			desc:             "使用负数时间间隔创建服务实例",
		},
		{
			name:             "小池容量",
			timeSyncInterval: 5 * time.Second,
			pool:             createTestPool(t, 1),
			upgrader:         websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
			wantNil:          false,
			desc:             "使用小容量协程池创建服务实例",
		},
		{
			name:             "大池容量",
			timeSyncInterval: 5 * time.Second,
			pool:             createTestPool(t, 1000),
			upgrader:         websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
			wantNil:          false,
			desc:             "使用大容量协程池创建服务实例",
		},
		{
			name:             "自定义WebSocket升级器",
			timeSyncInterval: 5 * time.Second,
			pool:             createTestPool(t, 10),
			upgrader: websocket.Upgrader{
				CheckOrigin:     func(r *http.Request) bool { return false },
				ReadBufferSize:  1024,
				WriteBufferSize: 1024,
			},
			wantNil: false,
			wantErr: false,
			desc:    "使用自定义WebSocket升级器创建服务实例",
		},
		{
			name:             "空的WebSocket升级器",
			timeSyncInterval: 5 * time.Second,
			pool:             createTestPool(t, 10),
			upgrader:         websocket.Upgrader{},
			wantNil:          false,
			wantErr:          true,
			desc:             "使用自定义WebSocket升级器创建服务实例",
		},
		{
			name:             "空的ants池",
			timeSyncInterval: 5 * time.Second,
			pool:             nil,
			upgrader:         websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
			wantNil:          false,
			wantErr:          true,
			desc:             "使用自定义WebSocket升级器创建服务实例",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 确保在测试结束后释放协程池
			defer func() {
				if tt.pool != nil {
					tt.pool.Release()
				}
			}()

			// 调用 NewService 函数
			service, err := NewService(tt.timeSyncInterval, tt.pool, tt.upgrader)

			// 验证返回值
			if tt.wantNil {
				if service != nil {
					t.Errorf("Expected nil service but got non-nil")
				}
				return
			}

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else {
					t.Logf("Expected error occurred: %v", err)
				}
				return
			}

			// 验证服务实例不为空
			if service == nil {
				t.Errorf("Expected non-nil service but got nil")
				return
			}

			// 验证服务实例类型
			srvImpl, ok := service.(*serviceImpl)
			if !ok {
				t.Errorf("Expected *serviceImpl but got %T", service)
				return
			}

			// 验证字段设置
			if srvImpl.timeSyncInterval != tt.timeSyncInterval {
				t.Errorf("Expected timeSyncInterval %v but got %v", tt.timeSyncInterval, srvImpl.timeSyncInterval)
			}

			if srvImpl.pool != tt.pool {
				t.Errorf("Expected pool %p but got %p", tt.pool, srvImpl.pool)
			}

			// 验证通道初始化
			if srvImpl.registerChanel == nil {
				t.Error("Expected registerChanel to be initialized but got nil")
			}

			if srvImpl.unregisterChanel == nil {
				t.Error("Expected unregisterChanel to be initialized but got nil")
			}

			// 验证客户端映射初始化
			if srvImpl.clients == nil {
				t.Error("Expected clients map to be initialized but got nil")
			}

			// 验证仓库初始化
			if srvImpl.repo == nil {
				t.Error("Expected repo to be initialized but got nil")
			}

			// 验证通道容量
			if cap(srvImpl.registerChanel) != 100 {
				t.Errorf("Expected registerChanel capacity 100 but got %d", cap(srvImpl.registerChanel))
			}

			if cap(srvImpl.unregisterChanel) != 100 {
				t.Errorf("Expected unregisterChanel capacity 100 but got %d", cap(srvImpl.unregisterChanel))
			}

			t.Logf("Test passed: %s", tt.desc)
		})
	}
}

// createTestPool 创建测试用的协程池
func createTestPool(t *testing.T, size int) *ants.Pool {
	pool, err := ants.NewPool(size)
	if err != nil {
		t.Fatalf("Failed to create test pool: %v", err)
	}
	return pool
}

// Test_serviceImpl_HandleInitTimeSyncConn 测试HandleInitTimeSyncConn方法
func Test_serviceImpl_HandleInitTimeSyncConn(t *testing.T) {
	tests := []struct {
		name         string
		examineeID   string
		submissionID string
		wantErr      bool
		desc         string
		mockRepoFunc func() Repo
	}{
		{
			name:         "缺少examinee_id和practice_submission_id参数",
			examineeID:   "",
			submissionID: "",
			wantErr:      true,
			desc:         "测试缺少必要参数时的错误处理",
			mockRepoFunc: func() Repo {
				return &MockRepo{}
			},
		},
		{
			name:         "无效的examinee_id格式",
			examineeID:   "invalid",
			submissionID: "67890",
			wantErr:      true,
			desc:         "测试无效的examinee_id格式时的错误处理",
			mockRepoFunc: func() Repo {
				return &MockRepo{}
			},
		},
		{
			name:         "无效的practice_submission_id格式",
			examineeID:   "12345",
			submissionID: "invalid",
			wantErr:      true,
			desc:         "测试无效的practice_submission_id格式时的错误处理",
			mockRepoFunc: func() Repo {
				return &MockRepo{}
			},
		},
		{
			name:         "无效的用户ID",
			examineeID:   "12345",
			submissionID: "67890",
			wantErr:      true,
			desc:         "测试无效用户ID时的错误处理",
			mockRepoFunc: func() Repo {
				return &MockRepo{}
			},
		},
		{
			name:         "用户ID小于0",
			examineeID:   "12345",
			submissionID: "67890",
			wantErr:      true,
			desc:         "测试用户ID小于0时的错误处理",
			mockRepoFunc: func() Repo {
				return &MockRepo{}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建mock服务实例
			mockRepo := tt.mockRepoFunc()
			pool, _ := ants.NewPool(10)
			defer pool.Release()

			srv := &serviceImpl{
				repo:             mockRepo,
				upgrader:         websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
				timeSyncInterval: time.Second,
				pool:             pool,
				clients:          make(map[string]client.Connection),
				registerChanel:   make(chan client.Connection, 100),
				unregisterChanel: make(chan client.Connection, 100),
			}

			// 构建请求URL
			queryParams := url.Values{}
			if tt.examineeID != "" {
				queryParams.Set("examinee_id", tt.examineeID)
			}
			if tt.submissionID != "" {
				queryParams.Set("practice_submission_id", tt.submissionID)
			}

			// 创建HTTP请求
			req := httptest.NewRequest("GET", "/api/time-sync?"+queryParams.Encode(), nil)
			// 添加WebSocket升级头
			req.Header.Set("Connection", "Upgrade")
			req.Header.Set("Upgrade", "websocket")
			req.Header.Set("Sec-WebSocket-Version", "13")
			req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")

			// 创建ResponseRecorder
			w := httptest.NewRecorder()

			// 创建ServiceCtx
			srvCtx := &cmn.ServiceCtx{
				Msg: &cmn.ReplyProto{
					API:    "/time-sync",
					Method: "GET",
				},
				BeginTime: time.Now(),
				Tag:       make(map[string]interface{}),
				R:         req,
				W:         w,
			}

			// 根据测试用例设置用户信息
			if tt.name == "无效的用户ID" {
				// 设置无效用户ID
				srvCtx.SysUser = &cmn.TUser{
					ID: null.NewInt(0, false), // 无效ID
				}
			} else if tt.name == "用户ID小于0" {
				// 设置负数用户ID
				srvCtx.SysUser = &cmn.TUser{
					ID: null.NewInt(-1, true), // 负数ID
				}
			} else {
				// 设置有效用户ID
				srvCtx.SysUser = &cmn.TUser{
					ID: null.NewInt(54242, true),
				}
			}

			ctx := context.WithValue(context.Background(), cmn.QNearKey, srvCtx)

			// 捕获panic（WebSocket升级失败时会panic）
			defer func() {
				if r := recover(); r != nil {
					// WebSocket升级失败是预期的，因为我们使用的是httptest
					t.Logf("WebSocket upgrade failed as expected: %v", r)
				}
			}()

			// 调用HandleInitTimeSyncConn
			srv.HandleInitTimeSyncConn(ctx)

			// 验证结果
			if tt.wantErr {
				// 期望有错误
				if srvCtx.Err == nil {
					t.Errorf("Expected error but got none")
				} else {
					t.Logf("Expected error occurred: %v", srvCtx.Err)
				}
			} else {
				// 期望成功（在WebSocket升级之前的参数验证应该通过）
				if srvCtx.Err != nil {
					t.Errorf("Expected success but got error: %v", srvCtx.Err)
				} else {
					t.Log("Parameter validation passed successfully")
				}
			}
		})
	}
}

// Test_serviceImpl_HandleInitTimeSyncConn_Websocket 测试HandleInitTimeSyncConn方法（启用WebSocket）
func Test_serviceImpl_HandleInitTimeSyncConn_Websocket(t *testing.T) {
	tests := []struct {
		name                   string
		sysUserID              null.Int
		examineeID             string
		submissionID           string
		doResetEndTime         bool
		resetEndTimeExamineeID int64
		sleepDuration1         time.Duration
		sleepDuration2         time.Duration
		msgToSend              interface{}
		reqForceError          string
		serveForceError        string
		wantErr                bool
		desc                   string
		mockRepoFunc           func() Repo
	}{
		{
			name:            "只有examinee_id参数",
			examineeID:      "12345",
			submissionID:    "",
			sysUserID:       null.NewInt(12345, true),
			sleepDuration1:  0,
			reqForceError:   "",
			serveForceError: "",
			wantErr:         false,
			desc:            "测试只有examinee_id参数",
			mockRepoFunc: func() Repo {
				return &MockRepo{}
			},
		},
		{
			name:            "只有submission_id参数",
			examineeID:      "",
			submissionID:    "12345",
			sysUserID:       null.NewInt(12345, true),
			sleepDuration1:  0,
			reqForceError:   "",
			serveForceError: "",
			wantErr:         false,
			desc:            "测试只有examinee_id参数",
			mockRepoFunc: func() Repo {
				return &MockRepo{}
			},
		},
		{
			name:            "examinee_id和submission_id参数都没有",
			examineeID:      "",
			submissionID:    "",
			sysUserID:       null.NewInt(12345, true),
			sleepDuration1:  0,
			reqForceError:   "",
			serveForceError: "",
			wantErr:         true,
			desc:            "测试xaminee_id和submission_id参数都没有",
			mockRepoFunc: func() Repo {
				return &MockRepo{}
			},
		},
		{
			name:            "pool.ReleaseTimeout错误",
			examineeID:      "",
			submissionID:    "12345",
			sysUserID:       null.NewInt(12345, true),
			sleepDuration1:  0,
			reqForceError:   "",
			serveForceError: "pool.ReleaseTimeout",
			wantErr:         false,
			desc:            "测试只有examinee_id参数",
			mockRepoFunc: func() Repo {
				return &MockRepo{}
			},
		},
		{
			name:            "websocket.Upgrade错误",
			examineeID:      "12345",
			submissionID:    "",
			sysUserID:       null.NewInt(12345, true),
			sleepDuration1:  0,
			reqForceError:   "websocket.Upgrade",
			serveForceError: "",
			wantErr:         true,
			desc:            "测试websocket.Upgrade错误的处理",
			mockRepoFunc: func() Repo {
				return &MockRepo{}
			},
		},
		{
			name:           "测试广播时间同步消息",
			examineeID:     "12345",
			submissionID:   "67890",
			sysUserID:      null.NewInt(54242, true),
			sleepDuration1: 10 * time.Second,
			reqForceError:  "",
			wantErr:        false,
			desc:           "测试测试广播时间同步消息",
			mockRepoFunc: func() Repo {
				mockRepo := &MockRepo{}
				mockRepo.QueryActualExamEndTimeFunc = func(examineeId int64) (int64, error) {
					return time.Now().Add(time.Hour).UnixMilli(), nil
				}
				return mockRepo
			},
		},
		{
			name:            "注册客户端后发送时间同步消息错误",
			examineeID:      "12345",
			submissionID:    "",
			sysUserID:       null.NewInt(12345, true),
			sleepDuration1:  5 * time.Second,
			reqForceError:   "sendCurrentTimestampMsg",
			serveForceError: "",
			wantErr:         true,
			desc:            "测试注册客户端后发送时间同步消息错误的错误处理",
			mockRepoFunc: func() Repo {
				return &MockRepo{}
			},
		},
		{
			name:            "测试广播时间同步消息时发送消息错误",
			examineeID:      "12345",
			submissionID:    "67890",
			sysUserID:       null.NewInt(54242, true),
			sleepDuration1:  10 * time.Second,
			reqForceError:   "",
			serveForceError: "sendCurrentTimestampMsg",
			wantErr:         false,
			desc:            "测试测试广播时间同步消息时发送消息错误",
			mockRepoFunc: func() Repo {
				mockRepo := &MockRepo{}
				mockRepo.QueryActualExamEndTimeFunc = func(examineeId int64) (int64, error) {
					return time.Now().Add(time.Hour).UnixMilli(), nil
				}
				return mockRepo
			},
		},
		{
			name:            "测试广播时间同步消息时提交协程池错误",
			examineeID:      "12345",
			submissionID:    "67890",
			sysUserID:       null.NewInt(54242, true),
			sleepDuration1:  10 * time.Second,
			reqForceError:   "",
			serveForceError: "ants.Pool.Submit",
			wantErr:         false,
			desc:            "测试测试广播时间同步消息时提交协程池错误",
			mockRepoFunc: func() Repo {
				mockRepo := &MockRepo{}
				mockRepo.QueryActualExamEndTimeFunc = func(examineeId int64) (int64, error) {
					return time.Now().Add(time.Hour).UnixMilli(), nil
				}
				return mockRepo
			},
		},
		{
			name:                   "测试重置考试结束时间｜重置成功",
			examineeID:             "12345",
			submissionID:           "67890",
			sysUserID:              null.NewInt(54242, true),
			resetEndTimeExamineeID: 12345,
			doResetEndTime:         true,
			sleepDuration1:         5 * time.Second,
			sleepDuration2:         5 * time.Second,
			reqForceError:          "",
			wantErr:                false,
			desc:                   "测试重置考试结束时间",
			mockRepoFunc: func() Repo {
				mockRepo := &MockRepo{}
				mockRepo.QueryActualExamEndTimeFunc = func(examineeId int64) (int64, error) {
					return time.Now().Add(time.Hour).UnixMilli(), nil
				}
				return mockRepo
			},
		},
		{
			name:                   "测试重置考试结束时间｜发送消息错误",
			examineeID:             "12345",
			submissionID:           "67890",
			sysUserID:              null.NewInt(54242, true),
			resetEndTimeExamineeID: 12345,
			doResetEndTime:         true,
			sleepDuration1:         5 * time.Second,
			sleepDuration2:         5 * time.Second,
			reqForceError:          "",
			serveForceError:        "sendMessage",
			wantErr:                false,
			desc:                   "测试重置考试结束时间",
			mockRepoFunc: func() Repo {
				mockRepo := &MockRepo{}
				mockRepo.QueryActualExamEndTimeFunc = func(examineeId int64) (int64, error) {
					return time.Now().Add(time.Hour).UnixMilli(), nil
				}
				return mockRepo
			},
		},
		{
			name:                   "测试重置考试结束时间｜ants.Pool.Submit错误",
			examineeID:             "12345",
			submissionID:           "67890",
			sysUserID:              null.NewInt(54242, true),
			resetEndTimeExamineeID: 12345,
			doResetEndTime:         true,
			sleepDuration1:         5 * time.Second,
			sleepDuration2:         5 * time.Second,
			reqForceError:          "",
			serveForceError:        "ants.Pool.Submit",
			wantErr:                false,
			desc:                   "测试重置考试结束时间",
			mockRepoFunc: func() Repo {
				mockRepo := &MockRepo{}
				mockRepo.QueryActualExamEndTimeFunc = func(examineeId int64) (int64, error) {
					return time.Now().Add(time.Hour).UnixMilli(), nil
				}
				return mockRepo
			},
		},
		{
			name:                   "测试重置考试结束时间｜目标考生ID不合法",
			examineeID:             "12345",
			submissionID:           "67890",
			sysUserID:              null.NewInt(54242, true),
			resetEndTimeExamineeID: -1,
			doResetEndTime:         true,
			sleepDuration1:         5 * time.Second,
			sleepDuration2:         5 * time.Second,
			reqForceError:          "",
			wantErr:                false,
			desc:                   "测试重置考试结束时间",
			mockRepoFunc: func() Repo {
				mockRepo := &MockRepo{}
				mockRepo.QueryActualExamEndTimeFunc = func(examineeId int64) (int64, error) {
					return time.Now().Add(time.Hour).UnixMilli(), nil
				}
				return mockRepo
			},
		},
		{
			name:                   "测试重置考试结束时间｜不存在的考生ID",
			examineeID:             "12345",
			submissionID:           "67890",
			sysUserID:              null.NewInt(54242, true),
			resetEndTimeExamineeID: 45634,
			doResetEndTime:         true,
			sleepDuration1:         5 * time.Second,
			sleepDuration2:         5 * time.Second,
			reqForceError:          "",
			wantErr:                false,
			desc:                   "测试重置考试结束时间",
			mockRepoFunc: func() Repo {
				mockRepo := &MockRepo{}
				mockRepo.QueryActualExamEndTimeFunc = func(examineeId int64) (int64, error) {
					return time.Now().Add(time.Hour).UnixMilli(), nil
				}
				return mockRepo
			},
		},
		{
			name:                   "测试重置考试结束时间｜获取考生的实际考试结束时间错误",
			examineeID:             "12345",
			submissionID:           "67890",
			sysUserID:              null.NewInt(54242, true),
			resetEndTimeExamineeID: 54242,
			doResetEndTime:         true,
			sleepDuration1:         5 * time.Second,
			sleepDuration2:         5 * time.Second,
			reqForceError:          "",
			wantErr:                false,
			desc:                   "测试重置考试结束时间",
			mockRepoFunc: func() Repo {
				mockRepo := &MockRepo{}
				mockRepo.QueryActualExamEndTimeFunc = func(examineeId int64) (int64, error) {
					return time.Now().Add(time.Hour).UnixMilli(), fmt.Errorf("failed to get actual exam end time")
				}
				return mockRepo
			},
		},
		{
			name:                   "测试重置考试结束时间｜考生的实际考试结束时间比当前时间小",
			examineeID:             "12345",
			submissionID:           "67890",
			sysUserID:              null.NewInt(54242, true),
			resetEndTimeExamineeID: 54242,
			doResetEndTime:         true,
			sleepDuration1:         5 * time.Second,
			sleepDuration2:         5 * time.Second,
			reqForceError:          "",
			wantErr:                false,
			desc:                   "测试重置考试结束时间",
			mockRepoFunc: func() Repo {
				mockRepo := &MockRepo{}
				mockRepo.QueryActualExamEndTimeFunc = func(examineeId int64) (int64, error) {
					return time.Now().Add(-time.Hour).UnixMilli(), nil
				}
				return mockRepo
			},
		},
		{
			name:                   "发送非json格式的消息",
			examineeID:             "12345",
			submissionID:           "67890",
			sysUserID:              null.NewInt(54242, true),
			resetEndTimeExamineeID: 12345,
			doResetEndTime:         true,
			sleepDuration1:         5 * time.Second,
			sleepDuration2:         5 * time.Second,
			reqForceError:          "",
			msgToSend:              "invalid message format",
			wantErr:                false,
			desc:                   "测试重置考试结束时间",
			mockRepoFunc: func() Repo {
				mockRepo := &MockRepo{}
				mockRepo.QueryActualExamEndTimeFunc = func(examineeId int64) (int64, error) {
					return time.Now().Add(time.Hour).UnixMilli(), nil
				}
				return mockRepo
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			// 创建mock服务实例
			mockRepo := tt.mockRepoFunc()
			srv := &serviceImpl{
				repo:             mockRepo,
				upgrader:         websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
				timeSyncInterval: timeSyncIntervalMillisecond,
				pool:             pool,
				clients:          make(map[string]client.Connection),
				registerChanel:   make(chan client.Connection, 100),
				unregisterChanel: make(chan client.Connection, 100),
			}

			serveCtx, cancel := context.WithCancel(context.Background())
			serveCtx = context.WithValue(serveCtx, "force-error", tt.serveForceError)
			defer cancel()

			// 启动时间同步服务
			go srv.StartServe(serveCtx)

			srvCtx := &cmn.ServiceCtx{
				Msg: &cmn.ReplyProto{
					API:    "/time-sync",
					Method: "GET",
				},
				BeginTime: time.Now(),
				Tag:       make(map[string]interface{}),
				SysUser: &cmn.TUser{
					ID: tt.sysUserID,
				},
			}

			// 创建HTTP服务器来处理WebSocket升级
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// 创建ServiceCtx
				serviceCtx := srvCtx
				serviceCtx.R = r
				serviceCtx.W = w

				ctx := context.WithValue(context.Background(), cmn.QNearKey, serviceCtx)
				ctx = context.WithValue(ctx, "force-error", tt.reqForceError)

				// 调用HandleInitTimeSyncConn处理WebSocket连接
				srv.HandleInitTimeSyncConn(ctx)
			}))
			defer server.Close()

			// 构建WebSocket URL
			queryParams := url.Values{}
			if tt.examineeID != "" {
				queryParams.Set("examinee_id", tt.examineeID)
			}
			if tt.submissionID != "" {
				queryParams.Set("practice_submission_id", tt.submissionID)
			}

			// 将HTTP URL转换为WebSocket URL
			wsURL := "ws" + server.URL[4:] + "?" + queryParams.Encode()

			// 创建WebSocket客户端连接
			conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)

			// 启动消息读取循环
			go func() {
				for {
					_, _, err := conn.ReadMessage()
					if err != nil {
						log.Println("读取消息错误:", err)
						return
					}
				}
			}()

			// 启动消息发送循环：定时发送 JSON 格式的 ReceiveMsg
			go func() {
				ticker := time.NewTicker(3 * time.Second) // 每 3 秒发送一次
				defer ticker.Stop()

				for range ticker.C {
					var msg interface{}
					if tt.msgToSend != nil {
						msg = tt.msgToSend
					} else {
						msg = client.ReceiveMsg{
							MsgType: 1, // 示例：你可以自定义类型
						}
					}

					data, err := json.Marshal(msg)
					if err != nil {
						log.Println("编码消息失败:", err)
						continue
					}

					err = conn.WriteMessage(websocket.TextMessage, data)
					if err != nil {
						log.Println("发送消息失败:", err)
						return
					}

					log.Println("发送消息:", string(data))
				}
			}()

			// 启动心跳机制
			// 设置收到 ping 控制帧时的处理函数
			conn.SetPingHandler(func(appData string) error {
				log.Println("客户端收到 ping，回复 pong")
				// 响应 pong 消息，携带相同内容，设置写入超时时间
				return conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(5*time.Second))
			})

			time.Sleep(tt.sleepDuration1)

			if tt.doResetEndTime {
				cmn.ResetExamEndTimeChan <- tt.resetEndTimeExamineeID
			}

			time.Sleep(tt.sleepDuration2)

			// 验证结果
			if tt.wantErr {
				// 期望有错误，检查连接是否失败或响应状态码不是101
				if srvCtx.Err == nil {
					t.Errorf("Expected error but connection succeeded")
					return
				}
				t.Logf("Expected error occurred: %v", srvCtx.Err)
			} else {
				// 期望成功，检查连接是否建立
				if err != nil {
					t.Errorf("Expected success but got error: %v", err)
					return
				}
				if resp == nil || resp.StatusCode != 101 {
					t.Errorf("Expected WebSocket upgrade but got status: %v", resp.StatusCode)
					return
				}
				if conn == nil {
					t.Error("Expected valid connection but got nil")
					return
				}

				// 可以在这里添加更多的WebSocket通信测试
				t.Log("WebSocket connection established successfully")
			}

			if conn != nil {
				conn.Close()
			}
		})
	}
}
