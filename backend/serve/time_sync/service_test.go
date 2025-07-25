package time_sync

import (
	"context"
	"github.com/panjf2000/ants/v2"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"
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

// Test_serviceImpl_HandleInitTimeSyncConn 测试HandleInitTimeSyncConn方法
func Test_serviceImpl_HandleInitTimeSyncConn(t *testing.T) {
	tests := []struct {
		name         string
		examineeID   string
		submissionID string
		forceError   string
		wantErr      bool
		desc         string
		mockRepoFunc func() Repo
	}{
		{
			name:         "成功建立时间同步连接",
			examineeID:   "12345",
			submissionID: "67890",
			forceError:   "",
			wantErr:      false,
			desc:         "测试正常情况下建立时间同步连接",
			mockRepoFunc: func() Repo {
				mockRepo := &MockRepo{}
				mockRepo.QueryActualExamEndTimeFunc = func(examineeId int64) (int64, error) {
					return time.Now().Add(time.Hour).UnixMilli(), nil
				}
				return mockRepo
			},
		},
		{
			name:         "缺少examinee_id和practice_submission_id参数",
			examineeID:   "",
			submissionID: "",
			forceError:   "",
			wantErr:      true,
			desc:         "测试缺少examinee_id参数时的错误处理",
			mockRepoFunc: func() Repo {
				return &MockRepo{}
			},
		},
		{
			name:         "缺少examinee_id参数",
			examineeID:   "",
			submissionID: "67890",
			forceError:   "",
			wantErr:      false,
			desc:         "测试缺少examinee_id参数时的错误处理",
			mockRepoFunc: func() Repo {
				return &MockRepo{}
			},
		},
		{
			name:         "缺少practice_submission_id参数",
			examineeID:   "12345",
			submissionID: "",
			forceError:   "",
			wantErr:      false,
			desc:         "测试缺少practice_submission_id参数时的错误处理",
			mockRepoFunc: func() Repo {
				return &MockRepo{}
			},
		},
		{
			name:         "无效的examinee_id格式",
			examineeID:   "invalid",
			submissionID: "67890",
			forceError:   "",
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
			forceError:   "",
			wantErr:      true,
			desc:         "测试无效的practice_submission_id格式时的错误处理",
			mockRepoFunc: func() Repo {
				return &MockRepo{}
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

			// 启动时间同步服务
			go srv.StartServe()

			srvCtx := &cmn.ServiceCtx{
				Msg: &cmn.ReplyProto{
					API:    "/time-sync",
					Method: "GET",
				},
				BeginTime: time.Now(),
				Tag:       make(map[string]interface{}),
				SysUser: &cmn.TUser{
					ID: null.NewInt(54242, true),
				},
			}

			// 创建HTTP服务器来处理WebSocket升级
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// 创建ServiceCtx
				serviceCtx := srvCtx
				serviceCtx.R = r
				serviceCtx.W = w

				ctx := context.WithValue(context.Background(), cmn.QNearKey, serviceCtx)
				ctx = context.WithValue(ctx, "force-error", tt.forceError)

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

			// 验证结果
			if tt.wantErr {
				// 期望有错误，检查连接是否失败或响应状态码不是101
				if srvCtx.Err == nil {
					t.Errorf("Expected error but connection succeeded")
					if conn != nil {
						conn.Close()
					}
					return
				}
				t.Logf("Expected error occurred: %v", err)
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

				// 测试连接是否可以正常通信
				defer conn.Close()

				// 可以在这里添加更多的WebSocket通信测试
				t.Log("WebSocket connection established successfully")
			}
		})
	}
}
