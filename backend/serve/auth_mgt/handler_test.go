package auth_mgt

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"w2w.io/cmn"
)

// TestHandleQuerySelectableAPIs 测试查询可选API的处理函数
func TestHandleQuerySelectableAPIs(t *testing.T) {
	type args struct {
		method       string
		parentDomain string
		forceError   string
	}
	tests := []struct {
		name       string
		args       args
		wantStatus int
		wantErr    bool
	}{
		{
			name: "成功查询机构级别API｜GET方法",
			args: args{
				method:       "GET",
				parentDomain: "",
				forceError:   "",
			},
			wantStatus: 0,
			wantErr:    false,
		},
		{
			name: "成功查询部门级别API｜有效父域",
			args: args{
				method:       "GET",
				parentDomain: "cst.school",
				forceError:   "",
			},
			wantStatus: 0,
			wantErr:    false,
		},
		{
			name: "失败｜无效HTTP方法",
			args: args{
				method:       "POST",
				parentDomain: "",
				forceError:   "",
			},
			wantStatus: -1,
			wantErr:    true,
		},
		{
			name: "失败｜无效父域格式",
			args: args{
				method:       "GET",
				parentDomain: "invalid^domain^format",
				forceError:   "",
			},
			wantStatus: -1,
			wantErr:    true,
		},
		{
			name: "失败｜强制querySelectableAPIs错误",
			args: args{
				method:       "GET",
				parentDomain: "",
				forceError:   "querySelectableAPIs",
			},
			wantStatus: -1,
			wantErr:    true,
		},
		{
			name: "失败｜强制JSON序列化错误",
			args: args{
				method:       "GET",
				parentDomain: "",
				forceError:   "json.Marshal",
			},
			wantStatus: -1,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建HTTP请求
			req := httptest.NewRequest(tt.args.method, "/api/querySelectableAPIs", nil)
			if tt.args.parentDomain != "" {
				q := req.URL.Query()
				q.Set("parentDomain", tt.args.parentDomain)
				req.URL.RawQuery = q.Encode()
			}

			// 创建响应记录器
			w := httptest.NewRecorder()

			// 创建上下文并设置必要的值
			ctx := context.Background()
			if tt.args.forceError != "" {
				ctx = context.WithValue(ctx, "force-error", tt.args.forceError)
			}

			// 创建cmn.ServiceCtx对象并设置到上下文中
			q := &cmn.ServiceCtx{
				R:   req,
				W:   w,
				Msg: &cmn.ReplyProto{},
			}
			ctx = context.WithValue(ctx, cmn.QNearKey, q)

			// 创建handler并调用方法
			h := NewHandler()
			h.HandleQuerySelectableAPIs(ctx)

			// 验证结果
			if tt.wantErr {
				if q.Msg.Status != tt.wantStatus {
					t.Errorf("HandleQuerySelectableAPIs() status = %v, wantStatus %v", q.Msg.Status, tt.wantStatus)
				}
				if q.Err == nil {
					t.Errorf("HandleQuerySelectableAPIs() expected error but got none")
				}
			} else {
				if q.Msg.Status != tt.wantStatus {
					t.Errorf("HandleQuerySelectableAPIs() status = %v, wantStatus %v", q.Msg.Status, tt.wantStatus)
				}
				if q.Err != nil {
					t.Errorf("HandleQuerySelectableAPIs() unexpected error = %v", q.Err)
				}

				// 验证返回的数据是否为有效的JSON
				if len(q.Msg.Data) > 0 {
					var apis []interface{}
					if err := json.Unmarshal(q.Msg.Data, &apis); err != nil {
						t.Errorf("HandleQuerySelectableAPIs() returned invalid JSON data: %v", err)
					}
				}
			}

			// 输出调试信息
			if testing.Verbose() {
				t.Logf("测试用例: %s", tt.name)
				t.Logf("  请求方法: %s", tt.args.method)
				t.Logf("  父域: %s", tt.args.parentDomain)
				t.Logf("  强制错误: %s", tt.args.forceError)
				t.Logf("  响应状态: %d", q.Msg.Status)
				t.Logf("  响应消息: %s", q.Msg.Msg)
				if q.Err != nil {
					t.Logf("  错误信息: %v", q.Err)
				}
			}
		})
	}
}
