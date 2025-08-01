package user_mgt

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"
	"w2w.io/cmn"
	"w2w.io/null"
)

// Test_handler_HandleGetNewAccount 测试HandleGetNewAccount方法
func Test_handler_HandleGetNewAccount(t *testing.T) {
	type fields struct {
		srv Service
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "成功生成新账号",
			fields: fields{
				srv: &MockService{
					GenerateUniqueAccountFunc: func(ctx context.Context, tx pgx.Tx, length int, maxAttempts int) (string, error) {
						return "abc123def", nil
					},
				},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user/new-account", url.Values{}, ""),
			},
			wantErr: false,
		},
		{
			name: "不支持的HTTP方法",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContext("POST", "/api/user/new-account", url.Values{}, ""),
			},
			wantErr: true,
		},
		{
			name: "不支持的HTTP方法 - PUT",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContext("PUT", "/api/user/new-account", url.Values{}, ""),
			},
			wantErr: true,
		},
		{
			name: "不支持的HTTP方法 - DELETE",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContext("DELETE", "/api/user/new-account", url.Values{}, ""),
			},
			wantErr: true,
		},
		{
			name: "生成账号失败",
			fields: fields{
				srv: &MockService{
					GenerateUniqueAccountFunc: func(ctx context.Context, tx pgx.Tx, length int, maxAttempts int) (string, error) {
						return "", fmt.Errorf("生成账号失败")
					},
				},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user/new-account", url.Values{}, ""),
			},
			wantErr: true,
		},
		{
			name: "生成账号参数验证 - 正确的长度和最大尝试次数",
			fields: fields{
				srv: &MockService{
					GenerateUniqueAccountFunc: func(ctx context.Context, tx pgx.Tx, length int, maxAttempts int) (string, error) {
						// 验证传入的参数是否正确
						if length != AccountLength {
							return "", fmt.Errorf("期望长度为 %d，实际为 %d", AccountLength, length)
						}
						if maxAttempts != 20 {
							return "", fmt.Errorf("期望最大尝试次数为 20，实际为 %d", maxAttempts)
						}
						return "test12345", nil
					},
				},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user/new-account", url.Values{}, ""),
			},
			wantErr: false,
		},
		{
			name: "生成空账号",
			fields: fields{
				srv: &MockService{
					GenerateUniqueAccountFunc: func(ctx context.Context, tx pgx.Tx, length int, maxAttempts int) (string, error) {
						return "", nil // 返回空字符串但无错误
					},
				},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user/new-account", url.Values{}, ""),
			},
			wantErr: false,
		},
		{
			name: "HTTP方法大小写不敏感 - Get",
			fields: fields{
				srv: &MockService{
					GenerateUniqueAccountFunc: func(ctx context.Context, tx pgx.Tx, length int, maxAttempts int) (string, error) {
						return "mixedcase1", nil
					},
				},
			},
			args: args{
				ctx: createMockContext("Get", "/api/user/new-account", url.Values{}, ""),
			},
			wantErr: false,
		},
		{
			name: "HTTP方法大小写不敏感 - GET",
			fields: fields{
				srv: &MockService{
					GenerateUniqueAccountFunc: func(ctx context.Context, tx pgx.Tx, length int, maxAttempts int) (string, error) {
						return "uppercase1", nil
					},
				},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user/new-account", url.Values{}, ""),
			},
			wantErr: false,
		},
		{
			name: "json.Marshal错误",
			fields: fields{
				srv: &MockService{
					GenerateUniqueAccountFunc: func(ctx context.Context, tx pgx.Tx, length int, maxAttempts int) (string, error) {
						return "uppercase1", nil
					},
				},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user/new-account", url.Values{}, "json.Marshal"),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &handler{
				srv: tt.fields.srv,
			}
			h.HandleGetNewAccount(tt.args.ctx)

			// 获取ServiceCtx以检查结果
			q := cmn.GetCtxValue(tt.args.ctx)
			if tt.wantErr {
				if q.Err == nil {
					t.Errorf("HandleGetNewAccount() 期望有错误，但没有错误")
				}
			} else {
				if q.Err != nil {
					t.Errorf("HandleGetNewAccount() 不期望有错误，但出现错误: %v", q.Err)
				}
				// 检查成功响应
				if q.Msg.Status != 0 {
					t.Errorf("HandleGetNewAccount() 期望状态码为 0，实际为 %d", q.Msg.Status)
				}
				if q.Msg.Msg != "success" {
					t.Errorf("HandleGetNewAccount() 期望消息为 'success'，实际为 '%s'", q.Msg.Msg)
				}
				if len(q.Msg.Data) == 0 {
					t.Errorf("HandleGetNewAccount() 期望返回账号数据，但数据为空")
				}
			}
		})
	}
}

func Test_handler_HandleUser(t *testing.T) {
	type fields struct {
		srv Service
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "成功获取用户列表",
			fields: fields{
				srv: &MockService{
					users: []User{
						{
							TUser: cmn.TUser{
								ID:           null.NewInt(1, true),
								Account:      "test_user_001",
								OfficialName: null.NewString("测试用户001", true),
								Gender:       null.NewString("M", true),
								Status:       null.NewString("00", true),
							},
						},
					},
					totalRows: 1,
				},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user", url.Values{
					"page":     {"1"},
					"pageSize": {"10"},
				}, ""),
			},
			wantErr: false,
		},
		{
			name: "带过滤条件的用户查询",
			fields: fields{
				srv: &MockService{
					users: []User{
						{
							TUser: cmn.TUser{
								ID:           null.NewInt(2, true),
								Account:      "admin_user",
								OfficialName: null.NewString("管理员", true),
								Gender:       null.NewString("F", true),
								Status:       null.NewString("00", true),
							},
						},
					},
					totalRows: 1,
				},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user", url.Values{
					"page":     {"1"},
					"pageSize": {"10"},
					"account":  {"admin"},
					"gender":   {"F"},
				}, ""),
			},
			wantErr: false,
		},
		{
			name: "当前用户权限不足",
			fields: fields{
				srv: &MockService{
					users: []User{
						{
							TUser: cmn.TUser{
								ID:           null.NewInt(1, true),
								Account:      "test_user_001",
								OfficialName: null.NewString("测试用户001", true),
								Gender:       null.NewString("M", true),
								Status:       null.NewString("00", true),
							},
						},
					},
					totalRows: 1,
					QueryUserCurrentRoleFunc: func(ctx context.Context, userId null.Int) (null.Int, null.String, error) {
						return null.Int{}, null.NewString("cst.school^teacher", true), nil
					},
				},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user", url.Values{
					"page":     {"1"},
					"pageSize": {"10"},
				}, ""),
			},
			wantErr: true,
		},
		{
			name: "查询当前用户角色失败",
			fields: fields{
				srv: &MockService{
					users: []User{
						{
							TUser: cmn.TUser{
								ID:           null.NewInt(1, true),
								Account:      "test_user_001",
								OfficialName: null.NewString("测试用户001", true),
								Gender:       null.NewString("M", true),
								Status:       null.NewString("00", true),
							},
						},
					},
					totalRows: 1,
					QueryUserCurrentRoleFunc: func(ctx context.Context, userId null.Int) (null.Int, null.String, error) {
						return null.Int{}, null.String{}, fmt.Errorf("查询角色失败")
					},
				},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user", url.Values{
					"page":     {"1"},
					"pageSize": {"10"},
				}, ""),
			},
			wantErr: true,
		},
		{
			name: "不合法的domain过滤条件",
			fields: fields{
				srv: &MockService{
					users: []User{
						{
							TUser: cmn.TUser{
								ID:           null.NewInt(2, true),
								Account:      "admin_user",
								OfficialName: null.NewString("管理员", true),
								Gender:       null.NewString("F", true),
								Status:       null.NewString("00", true),
							},
						},
					},
					totalRows: 1,
				},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user", url.Values{
					"page":     {"1"},
					"pageSize": {"10"},
					"account":  {"admin"},
					"gender":   {"F"},
					"domain":   {"invalid_domain"}, // 假设这个域名不存在
				}, ""),
			},
			wantErr: true,
		},
		{
			name: "无效的页码参数",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user", url.Values{
					"page":     {"0"},
					"pageSize": {"10"},
				}, ""),
			},
			wantErr: true,
		},
		{
			name: "无效的页面大小参数",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user", url.Values{
					"page":     {"1"},
					"pageSize": {"-1"},
				}, ""),
			},
			wantErr: true,
		},
		{
			name: "数据库查询错误",
			fields: fields{
				srv: &MockService{
					err: fmt.Errorf("数据库连接失败"),
				},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user", url.Values{
					"page":     {"1"},
					"pageSize": {"10"},
				}, ""),
			},
			wantErr: true,
		},
		{
			name: "不支持的HTTP方法",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContext("PUT", "/api/user", url.Values{}, ""),
			},
			wantErr: true,
		},
		{
			name: "空的查询结果",
			fields: fields{
				srv: &MockService{
					users:     []User{},
					totalRows: 0,
				},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user", url.Values{
					"page":     {"1"},
					"pageSize": {"10"},
					"account":  {"nonexistent"},
				}, ""),
			},
			wantErr: false,
		},
		{
			name: "大页面大小查询",
			fields: fields{
				srv: &MockService{
					users:     make([]User, 100),
					totalRows: 1000,
				},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user", url.Values{
					"page":     {"1"},
					"pageSize": {"100"},
				}, ""),
			},
			wantErr: false,
		},
		{
			name: "多条件过滤查询",
			fields: fields{
				srv: &MockService{
					users: []User{
						{
							TUser: cmn.TUser{
								ID:           null.NewInt(3, true),
								Account:      "test_female",
								OfficialName: null.NewString("女性测试用户", true),
								Gender:       null.NewString("F", true),
								MobilePhone:  null.NewString("13800138000", true),
								Email:        null.NewString("test@example.com", true),
								Status:       null.NewString("00", true),
							},
						},
					},
					totalRows: 1,
				},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user", url.Values{
					"page":        {"1"},
					"pageSize":    {"10"},
					"account":     {"test"},
					"name":        {"女性"},
					"mobilePhone": {"138"},
					"email":       {"test"},
					"gender":      {"F"},
					"status":      {"00"},
					"createTime":  {strconv.FormatInt(time.Now().Unix()-86400, 10)},
				}, ""),
			},
			wantErr: false,
		},
		{
			name: "触发json.Marshal错误",
			fields: fields{
				srv: &MockService{
					users: []User{
						{
							TUser: cmn.TUser{
								ID:           null.NewInt(3, true),
								Account:      "test_female",
								OfficialName: null.NewString("女性测试用户", true),
								Gender:       null.NewString("F", true),
								MobilePhone:  null.NewString("13800138000", true),
								Email:        null.NewString("test@example.com", true),
								Status:       null.NewString("00", true),
							},
						},
					},
					totalRows: 1,
				},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user", url.Values{
					"page":        {"1"},
					"pageSize":    {"10"},
					"account":     {"test"},
					"name":        {"女性"},
					"mobilePhone": {"138"},
					"email":       {"test"},
					"gender":      {"F"},
					"status":      {"00"},
					"createTime":  {strconv.FormatInt(time.Now().Unix()-86400, 10)},
				}, "json.Marshal"),
			},
			wantErr: true,
		},

		// POST方法测试用例
		{
			name: "成功创建单个用户",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("POST", "/api/user", `[{
					"Account": "new_user_001",
					"OfficialName": "新用户001"
				}]`, ""),
			},
			wantErr: false,
		},
		{
			name: "成功创建多个用户",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("POST", "/api/user", `[{
					"Account": "user_001",
					"OfficialName": "用户001"
				}, {
					"Account": "user_002",
					"OfficialName": "用户002"
				}]`, ""),
			},
			wantErr: false,
		},
		{
			name: "请求体为空",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("POST", "/api/user", "", ""),
			},
			wantErr: true,
		},
		{
			name: "JSON格式正确但不是数组",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("POST", "/api/user", `{"account": "test"}`, ""),
			},
			wantErr: true,
		},
		{
			name: "空的用户数组",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("POST", "/api/user", `[]`, ""),
			},
			wantErr: true,
		},
		{
			name: "数据库插入失败",
			fields: fields{
				srv: &MockService{
					err: fmt.Errorf("数据库连接失败"),
					ValidateUserFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []InvalidUser, error) {
						return []User{
							{
								TUser: cmn.TUser{
									Account: "test_user",
								},
							},
						}, nil, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("POST", "/api/user", `[{
					"Account": "test_user"
				}]`, ""),
			},
			wantErr: true,
		},
		{
			name: "包含特殊字符的用户数据",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("POST", "/api/user", `[{
					"Account": "test@user#001",
					"OfficialName": "测试用户@#$%"
				}]`, ""),
			},
			wantErr: false,
		},
		{
			name: "包含Unicode字符的用户数据",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("POST", "/api/user", `[{
					"Account": "用户账号",
					"OfficialName": "张三李四王五"
				}]`, ""),
			},
			wantErr: false,
		},
		{
			name: "空请求体",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("POST", "/api/user", "", ""),
			},
			wantErr: true,
		},
		{
			name: "io.ReadAll错误",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("POST", "/api/user", `[{
					"Account": "用户账号",
					"OfficialName": "张三李四王五"
				}]`, "io.ReadAll"),
			},
			wantErr: true,
		},
		{
			name: "io.Close错误",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("POST", "/api/user", `[{
					"Account": "用户账号",
					"OfficialName": "张三李四王五"
				}]`, "io.Close"),
			},
			wantErr: false,
		},
		{
			name: "json.Unmarshal错误",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("POST", "/api/user", `[{
					"Account": "用户账号",
					"OfficialName": "张三李四王五"
				}]`, "json.Unmarshal"),
			},
			wantErr: true,
		},
		{
			name: "插入用户时发生错误",
			fields: fields{
				srv: &MockService{
					err: fmt.Errorf("插入用户失败"),
					ValidateUserFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []InvalidUser, error) {
						return []User{
							{
								TUser: cmn.TUser{
									Account:      "new_user_001",
									OfficialName: null.NewString("新用户001", true),
								},
							},
						}, nil, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("POST", "/api/user", `[{
					"Account": "new_user_001",
					"OfficialName": "新用户001"
				}]`, ""),
			},
			wantErr: true,
		},
		{
			name: "验证用户信息时发生错误",
			fields: fields{
				srv: &MockService{
					ValidateUserFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []InvalidUser, error) {
						return nil, nil, fmt.Errorf("验证用户信息失败")
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("POST", "/api/user", `[{
					"Account": "new_user_001",
					"OfficialName": "新用户001"
				}]`, ""),
			},
			wantErr: true,
		},
		{
			name: "存在不合法的无法被插入的用户",
			fields: fields{
				srv: &MockService{
					ValidateUserFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []InvalidUser, error) {
						return nil, []InvalidUser{
							{
								Account: null.NewString("new_user_001", true),
								ErrorMsg: []null.String{
									null.NewString("账号已存在", true),
								},
							},
						}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("POST", "/api/user", `[{
					"Account": "new_user_001",
					"OfficialName": "新用户001"
				}]`, ""),
			},
			wantErr: true,
		},
		{
			name: "触发json.Marshal错误",
			fields: fields{
				srv: &MockService{
					ValidateUserFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []InvalidUser, error) {
						return nil, []InvalidUser{
							{
								Account: null.NewString("new_user_001", true),
								ErrorMsg: []null.String{
									null.NewString("账号已存在", true),
								},
							},
						}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("POST", "/api/user", `[{
					"Account": "new_user_001",
					"OfficialName": "新用户001"
				}]`, "json.Marshal"),
			},
			wantErr: true,
		},
		{
			name: "触发开启事务错误",
			fields: fields{
				srv: &MockService{
					ValidateUserFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []InvalidUser, error) {
						return nil, []InvalidUser{
							{
								Account: null.NewString("new_user_001", true),
								ErrorMsg: []null.String{
									null.NewString("账号已存在", true),
								},
							},
						}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("POST", "/api/user", `[{
					"Account": "new_user_001",
					"OfficialName": "新用户001"
				}]`, "tx.Begin"),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &handler{
				srv: tt.fields.srv,
			}

			// 执行测试
			h.HandleUser(tt.args.ctx)

			// 从mock的HTTP响应中解析结果
			q := cmn.GetCtxValue(tt.args.ctx)
			w := q.W.(*httptest.ResponseRecorder)

			// 解析响应体
			var response cmn.ReplyProto
			responseBody := w.Body.String()

			// 验证是否有响应内容
			if responseBody == "" {
				if tt.wantErr {
					return
				}
				t.Errorf("HandleUser() 响应体为空，期望有响应内容")
			}

			// 解析JSON响应
			err := json.Unmarshal([]byte(responseBody), &response)
			if err != nil {
				t.Errorf("HandleUser() 无法解析响应JSON: %v, 响应体: %s", err, responseBody)
				return
			}

			// 验证错误状态
			if tt.wantErr {
				if response.Status == 0 {
					t.Errorf("HandleUser() 期望有错误状态，但状态为0，响应: %+v", response)
				}
			} else {
				if response.Status != 0 {
					t.Errorf("HandleUser() 期望状态为0，但得到: %d，消息: %s", response.Status, response.Msg)
				}
			}
		})
	}
}

func Test_NewHandler(t *testing.T) {
	h := NewHandler()

	if h == nil {
		t.Fatal("expected non-nil handler")
	}

	// 可选断言类型（若 handler 是私有结构体）
	_, ok := h.(*handler)
	if !ok {
		t.Fatalf("expected *handler, got %T", h)
	}

	// 可选断言内部 service 是否非空（需要暴露或通过接口）
	internalHandler := h.(*handler)
	if internalHandler.srv == nil {
		t.Error("expected srv to be initialized")
	}
}
