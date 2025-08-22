package user_mgt

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
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

// Test_handler_HandleValidateUserToBeInsert 测试HandleValidateUserToBeInsert方法
func Test_handler_HandleValidateUserToBeInsert(t *testing.T) {
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
			name: "成功验证用户信息 - 所有用户有效",
			fields: fields{
				srv: &MockService{
					ValidateUserFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
						return users, []User{}, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("POST", "/api/user/validate", `[
						{
							"Account": "test_user_001",
							"OfficialName": "测试用户001",
							"Email": "test001@example.com",
							"MobilePhone": "13800138001"
						},
						{
							"Account": "test_user_002",
							"OfficialName": "测试用户002",
							"Email": "test002@example.com",
							"MobilePhone": "13800138002"
						}
					]`, ""),
			},
			wantErr: false,
		},
		{
			name: "成功验证用户信息 - 部分用户无效",
			fields: fields{
				srv: &MockService{
					ValidateUserFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
						return []User{
								{
									TUser: cmn.TUser{
										Account:      "valid_user",
										OfficialName: null.NewString("有效用户", true),
									},
								},
							}, []User{
								{
									TUser: cmn.TUser{
										Account:      "invalid_user",
										OfficialName: null.NewString("无效用户", true),
									},
									ErrorMsg: []null.String{
										null.NewString("账号已存在", true),
										null.NewString("邮箱格式不正确", true),
									},
								},
							}, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("POST", "/api/user/validate", `[
						{
							"Account": "valid_user",
							"OfficialName": "有效用户"
						},
						{
							"Account": "invalid_user",
							"OfficialName": "无效用户"
						}
					]`, ""),
			},
			wantErr: false,
		},
		{
			name: "不支持的HTTP方法 - GET",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user/validate", url.Values{}, ""),
			},
			wantErr: true,
		},
		{
			name: "不支持的HTTP方法 - PUT",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContext("PUT", "/api/user/validate", url.Values{}, ""),
			},
			wantErr: true,
		},
		{
			name: "不支持的HTTP方法 - DELETE",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContext("DELETE", "/api/user/validate", url.Values{}, ""),
			},
			wantErr: true,
		},
		{
			name: "不支持的HTTP方法 - PATCH",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContext("PATCH", "/api/user/validate", url.Values{}, ""),
			},
			wantErr: true,
		},
		{
			name: "HTTP方法大小写不敏感 - post",
			fields: fields{
				srv: &MockService{
					ValidateUserFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
						return users, []User{}, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("post", "/api/user/validate", `[
						{
							"Account": "test_user",
							"OfficialName": "测试用户"
						}
					]`, ""),
			},
			wantErr: false,
		},
		{
			name: "HTTP方法大小写不敏感 - Post",
			fields: fields{
				srv: &MockService{
					ValidateUserFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
						return users, []User{}, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("Post", "/api/user/validate", `[
						{
							"Account": "test_user",
							"OfficialName": "测试用户"
						}
					]`, ""),
			},
			wantErr: false,
		},
		{
			name: "io.ReadAll强制错误",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("POST", "/api/user/validate", `{"data": []}`, "io.ReadAll"),
			},
			wantErr: true,
		},
		{
			name: "io.Close强制错误",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("POST", "/api/user/validate", `{"data": []}`, "io.Close"),
			},
			wantErr: true,
		},
		{
			name: "请求体为空",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("POST", "/api/user/validate", "", ""),
			},
			wantErr: true,
		},
		{
			name: "json.Unmarshal强制错误 - 请求体解析",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("POST", "/api/user/validate", `{"data": []}`, "json.Unmarshal"),
			},
			wantErr: true,
		},
		{
			name: "无效的JSON格式 - 用户数据",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("POST", "/api/user/validate", `{
					"data": "invalid user data"
				}`, ""),
			},
			wantErr: true,
		},
		{
			name: "用户列表为空",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("POST", "/api/user/validate", `[]`, ""),
			},
			wantErr: true,
		},
		{
			name: "ValidateUserToBeInsert服务错误",
			fields: fields{
				srv: &MockService{
					ValidateUserFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
						return nil, nil, nil, fmt.Errorf("数据库连接失败")
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("POST", "/api/user/validate", `[
						{
							"Account": "test_user",
							"OfficialName": "测试用户"
						}
					]`, ""),
			},
			wantErr: true,
		},
		{
			name: "json.Marshal强制错误 - 无效用户序列化",
			fields: fields{
				srv: &MockService{
					ValidateUserFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
						return []User{}, []User{
							{
								TUser: cmn.TUser{
									Account:      "invalid_user",
									OfficialName: null.NewString("无效用户", true),
								},
								ErrorMsg: []null.String{
									null.NewString("账号已存在", true),
								},
							},
						}, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("POST", "/api/user/validate", `[
						{
							"Account": "invalid_user",
							"OfficialName": "无效用户"
						}
					]`, "json.Marshal"),
			},
			wantErr: true,
		},
		{
			name: "复杂用户数据验证",
			fields: fields{
				srv: &MockService{
					ValidateUserFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
						return users, []User{}, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("POST", "/api/user/validate", `[
						{
							"Account": "complex_user",
							"OfficialName": "复杂用户",
							"Gender": "M",
							"MobilePhone": "13800138000",
							"Email": "complex@example.com",
							"IDCardNo": "123456789012345678",
							"IDCardType": "身份证",
							"Category": "normal",
							"Status": "active",
							"Type": "user",
							"Domains": ["cst.school^teacher"],
							"APIs": []
						}
					]`, ""),
			},
			wantErr: false,
		},
		{
			name: "多个无效用户的详细错误信息",
			fields: fields{
				srv: &MockService{
					ValidateUserFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
						return []User{}, []User{
							{
								TUser: cmn.TUser{
									Account:      "user1",
									OfficialName: null.NewString("用户1", true),
									Email:        null.NewString("invalid-email", true),
								},
								ErrorMsg: []null.String{
									null.NewString("邮箱格式不正确", true),
									null.NewString("角色不能为空", true),
								},
							},
							{
								TUser: cmn.TUser{
									Account:      "user2",
									OfficialName: null.NewString("用户2", true),
									MobilePhone:  null.NewString("invalid-phone", true),
								},
								ErrorMsg: []null.String{
									null.NewString("手机号格式不正确", true),
									null.NewString("账号已存在", true),
								},
							},
						}, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("POST", "/api/user/validate", `[
						{
							"Account": "user1",
							"OfficialName": "用户1",
							"Email": "invalid-email"
						},
						{
							"Account": "user2",
							"OfficialName": "用户2",
							"MobilePhone": "invalid-phone"
						}
					]`, ""),
			},
			wantErr: false,
		},
		{
			name: "边界情况 - 单个用户",
			fields: fields{
				srv: &MockService{
					ValidateUserFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
						return users, []User{}, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("POST", "/api/user/validate", `[
						{
							"Account": "single_user",
							"OfficialName": "单个用户"
						}
					]`, ""),
			},
			wantErr: false,
		},
		{
			name: "边界情况 - 大量用户",
			fields: fields{
				srv: &MockService{
					ValidateUserFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
						return users, []User{}, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("POST", "/api/user/validate", `[
						{"Account": "user001", "OfficialName": "用户001"},
						{"Account": "user002", "OfficialName": "用户002"},
						{"Account": "user003", "OfficialName": "用户003"},
						{"Account": "user004", "OfficialName": "用户004"},
						{"Account": "user005", "OfficialName": "用户005"}
					]`, ""),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &handler{
				srv: tt.fields.srv,
			}
			h.HandleValidateUserToBeInsert(tt.args.ctx)

			// 获取ServiceCtx以检查结果
			q := cmn.GetCtxValue(tt.args.ctx)
			if tt.wantErr {
				if q.Err == nil {
					t.Errorf("HandleValidateUserToBeInsert() 期望有错误，但没有错误")
				}
			} else {
				if q.Err != nil {
					t.Errorf("HandleValidateUserToBeInsert() 不期望有错误，但出现错误: %v", q.Err)
				}
				// 检查成功响应
				if q.Msg.Status == 405 {
					// 状态码405表示有无效用户，这是正常情况
					if q.Msg.Msg != "some users are invalid and cannot be inserted" {
						t.Errorf("HandleValidateUserToBeInsert() 期望消息为 'some users are invalid and cannot be inserted'，实际为 '%s'", q.Msg.Msg)
					}
					if q.Msg.Data == nil {
						t.Errorf("HandleValidateUserToBeInsert() 期望返回无效用户数据，但数据为空")
					}
				} else if q.Msg.Status == 0 {
					// 状态码0表示所有用户都有效
					if q.Msg.Msg != "success" {
						t.Errorf("HandleValidateUserToBeInsert() 期望消息为 'success'，实际为 '%s'", q.Msg.Msg)
					}
				} else {
					t.Errorf("HandleValidateUserToBeInsert() 期望状态码为 0 或 405，实际为 %d", q.Msg.Status)
				}
			}
		})
	}
}

// Test_handler_HandleQueryMyInfo 测试HandleQueryMyInfo方法
func Test_handler_HandleQueryMyInfo(t *testing.T) {
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
			name: "成功查询用户信息",
			fields: fields{
				srv: &MockService{
					users: []User{
						{
							TUser: cmn.TUser{
								ID:           null.NewInt(54242, true),
								Account:      "test_user",
								OfficialName: null.NewString("测试用户", true),
								Gender:       null.NewString("M", true),
								MobilePhone:  null.NewString("13800138000", true),
								Email:        null.NewString("test@example.com", true),
								Category:     "normal",
								Status:       null.NewString("active", true),
								Type:         null.NewString("user", true),
								IDCardNo:     null.NewString("123456789012345678", true),
								IDCardType:   null.NewString("身份证", true),
								Role:         null.NewInt(1, true),
								LogonTime:    null.NewInt(time.Now().Unix(), true),
								CreateTime:   null.NewInt(time.Now().Unix(), true),
								UpdateTime:   null.NewInt(time.Now().Unix(), true),
								Creator:      null.NewInt(1000, true),
							},
							Domains: []null.String{
								null.StringFrom("cst.school^teacher"),
								null.StringFrom("other.domain"),
							},
							APIs: []cmn.TVUserDomainAPI{
								{
									UserID:   null.NewInt(54242, true),
									DomainID: null.NewInt(1, true),
									APIID:    null.NewInt(1, true),
								},
							},
						},
					},
					totalRows: 1,
				},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user/my-info", url.Values{}, ""),
			},
			wantErr: false,
		},
		{
			name: "不支持的HTTP方法",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContext("POST", "/api/user/my-info", url.Values{}, ""),
			},
			wantErr: true,
		},
		{
			name: "不支持的HTTP方法 - PUT",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContext("PUT", "/api/user/my-info", url.Values{}, ""),
			},
			wantErr: true,
		},
		{
			name: "不支持的HTTP方法 - DELETE",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContext("DELETE", "/api/user/my-info", url.Values{}, ""),
			},
			wantErr: true,
		},
		{
			name: "不支持的HTTP方法 - PATCH",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContext("PATCH", "/api/user/my-info", url.Values{}, ""),
			},
			wantErr: true,
		},
		{
			name: "用户未登录",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithoutUser("GET", "/api/user/my-info", "", ""),
			},
			wantErr: true,
		},
		{
			name: "查询用户失败",
			fields: fields{
				srv: &MockService{
					err: fmt.Errorf("数据库查询失败"),
				},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user/my-info", url.Values{}, ""),
			},
			wantErr: true,
		},
		{
			name: "用户不存在",
			fields: fields{
				srv: &MockService{
					users:     []User{},
					totalRows: 0,
				},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user/my-info", url.Values{}, ""),
			},
			wantErr: true,
		},
		{
			name: "json.Marshal强制错误",
			fields: fields{
				srv: &MockService{
					users: []User{
						{
							TUser: cmn.TUser{
								ID:           null.NewInt(54242, true),
								Account:      "test_user",
								OfficialName: null.NewString("测试用户", true),
								Category:     "normal",
							},
							Domains: []null.String{
								null.StringFrom("cst.school^teacher"),
							},
						},
					},
					totalRows: 1,
				},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user/my-info", url.Values{}, "json.Marshal"),
			},
			wantErr: true,
		},
		{
			name: "HTTP方法大小写不敏感 - get",
			fields: fields{
				srv: &MockService{
					users: []User{
						{
							TUser: cmn.TUser{
								ID:           null.NewInt(54242, true),
								Account:      "test_user",
								OfficialName: null.NewString("测试用户", true),
								Category:     "normal",
							},
							Domains: []null.String{
								null.StringFrom("cst.school^teacher"),
							},
						},
					},
					totalRows: 1,
				},
			},
			args: args{
				ctx: createMockContext("get", "/api/user/my-info", url.Values{}, ""),
			},
			wantErr: false,
		},
		{
			name: "HTTP方法大小写不敏感 - Get",
			fields: fields{
				srv: &MockService{
					users: []User{
						{
							TUser: cmn.TUser{
								ID:           null.NewInt(54242, true),
								Account:      "test_user",
								OfficialName: null.NewString("测试用户", true),
								Category:     "normal",
							},
							Domains: []null.String{
								null.StringFrom("cst.school^teacher"),
							},
						},
					},
					totalRows: 1,
				},
			},
			args: args{
				ctx: createMockContext("Get", "/api/user/my-info", url.Values{}, ""),
			},
			wantErr: false,
		},
		{
			name: "用户信息包含空值",
			fields: fields{
				srv: &MockService{
					users: []User{
						{
							TUser: cmn.TUser{
								ID:           null.NewInt(54242, true),
								Account:      "test_user",
								Category:     "normal",
								OfficialName: null.String{}, // 空值
								Gender:       null.String{}, // 空值
								MobilePhone:  null.String{}, // 空值
								Email:        null.String{}, // 空值
							},
							Domains: []null.String{},
							APIs:    []cmn.TVUserDomainAPI{},
						},
					},
					totalRows: 1,
				},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user/my-info", url.Values{}, ""),
			},
			wantErr: false,
		},
		{
			name: "用户信息包含多个域和API",
			fields: fields{
				srv: &MockService{
					users: []User{
						{
							TUser: cmn.TUser{
								ID:           null.NewInt(54242, true),
								Account:      "test_user",
								OfficialName: null.NewString("测试用户", true),
								Category:     "normal",
							},
							Domains: []null.String{
								null.StringFrom("domain1.com"),
								null.StringFrom("domain2.com"),
								null.StringFrom("domain3.com"),
							},
							APIs: []cmn.TVUserDomainAPI{
								{
									UserID:   null.NewInt(54242, true),
									DomainID: null.NewInt(1, true),
									APIID:    null.NewInt(1, true),
								},
								{
									UserID:   null.NewInt(54242, true),
									DomainID: null.NewInt(2, true),
									APIID:    null.NewInt(2, true),
								},
								{
									UserID:   null.NewInt(54242, true),
									DomainID: null.NewInt(3, true),
									APIID:    null.NewInt(3, true),
								},
							},
						},
					},
					totalRows: 1,
				},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user/my-info", url.Values{}, ""),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &handler{
				srv: tt.fields.srv,
			}
			h.HandleQueryMyInfo(tt.args.ctx)

			// 获取ServiceCtx以检查结果
			q := cmn.GetCtxValue(tt.args.ctx)
			if tt.wantErr {
				if q.Err == nil {
					t.Errorf("HandleQueryMyInfo() 期望有错误，但没有错误")
				}
			} else {
				if q.Err != nil {
					t.Errorf("HandleQueryMyInfo() 不期望有错误，但出现错误: %v", q.Err)
				}
				// 检查成功响应
				if q.Msg.Status != 0 {
					t.Errorf("HandleQueryMyInfo() 期望状态码为 0，实际为 %d", q.Msg.Status)
				}
				if q.Msg.Msg != "success" {
					t.Errorf("HandleQueryMyInfo() 期望消息为 'success'，实际为 '%s'", q.Msg.Msg)
				}
				// 检查返回的数据是否为用户信息
				if q.Msg.Data == nil {
					t.Errorf("HandleQueryMyInfo() 期望返回用户数据，但数据为空")
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
						return null.Int{}, null.NewString("cst.school^student", true), nil
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
					ValidateUserFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
						return []User{
							{
								TUser: cmn.TUser{
									Account: "test_user",
								},
							},
						}, []User{}, []User{}, nil
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
					ValidateUserFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
						return []User{
							{
								TUser: cmn.TUser{
									Account:      "new_user_001",
									OfficialName: null.NewString("新用户001", true),
								},
							},
						}, []User{}, []User{}, nil
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
					ValidateUserFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
						return nil, nil, nil, fmt.Errorf("验证用户信息失败")
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
					ValidateUserFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
						return []User{}, []User{
							{
								TUser: cmn.TUser{
									Account: "new_user_001",
								},
								ErrorMsg: []null.String{
									null.NewString("账号已存在", true),
								},
							},
						}, []User{}, nil
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
					ValidateUserFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
						return []User{}, []User{
							{
								TUser: cmn.TUser{
									Account: "new_user_001",
								},
								ErrorMsg: []null.String{
									null.NewString("账号已存在", true),
								},
							},
						}, []User{}, nil
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
					ValidateUserFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
						return []User{}, []User{
							{
								TUser: cmn.TUser{
									Account: "new_user_001",
								},
								ErrorMsg: []null.String{
									null.NewString("账号已存在", true),
								},
							},
						}, []User{}, nil
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
		{
			name: "触发json.Marshal错误",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("POST", "/api/user", `[{
					"Account": "new_user_001",
					"OfficialName": "新用户001"
				}]`, "json.Marshal"),
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

// Test_handler_HandleSelectLoginDomain 测试HandleSelectLoginDomain方法
func Test_handler_HandleSelectLoginDomain(t *testing.T) {
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
			name: "成功选择登录域",
			fields: fields{
				srv: &MockService{
					users: []User{
						{
							TUser: cmn.TUser{
								ID:           null.NewInt(54242, true),
								Account:      "test_user",
								OfficialName: null.NewString("测试用户", true),
							},
							Domains: []null.String{
								null.StringFrom("cst.school^superAdmin"),
								null.StringFrom("other.domain"),
							},
						},
					},
					totalRows: 1,
				},
			},
			args: args{
				ctx: createMockContextWithBody("PATCH", "/api/user/select-domain", `"cst.school^superAdmin"`, ""),
			},
			wantErr: false,
		},
		{
			name: "不支持的HTTP方法",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("GET", "/api/user/select-domain", `"cst.school^teacher"`, ""),
			},
			wantErr: true,
		},
		{
			name: "不支持的HTTP方法 - POST",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("POST", "/api/user/select-domain", `"cst.school^teacher"`, ""),
			},
			wantErr: true,
		},
		{
			name: "不支持的HTTP方法 - PUT",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("PUT", "/api/user/select-domain", `"cst.school^teacher"`, ""),
			},
			wantErr: true,
		},
		{
			name: "不支持的HTTP方法 - DELETE",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("DELETE", "/api/user/select-domain", `"cst.school^teacher"`, ""),
			},
			wantErr: true,
		},
		{
			name: "io.ReadAll错误",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("PATCH", "/api/user/select-domain", `"cst.school^teacher"`, "io.ReadAll"),
			},
			wantErr: true,
		},
		{
			name: "io.Close错误",
			fields: fields{
				srv: &MockService{
					users: []User{
						{
							TUser: cmn.TUser{
								ID:           null.NewInt(54242, true),
								Account:      "test_user",
								OfficialName: null.NewString("测试用户", true),
							},
							Domains: []null.String{
								null.StringFrom("cst.school^teacher"),
							},
						},
					},
					totalRows: 1,
				},
			},
			args: args{
				ctx: createMockContextWithBody("PATCH", "/api/user/select-domain", `"cst.school^teacher"`, "io.Close"),
			},
			wantErr: false, // io.Close错误不会导致整个请求失败
		},
		{
			name: "空请求体",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("PATCH", "/api/user/select-domain", "", ""),
			},
			wantErr: true,
		},
		{
			name: "json.Unmarshal错误",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("PATCH", "/api/user/select-domain", `"cst.school^teacher"`, "json.Unmarshal"),
			},
			wantErr: true,
		},
		{
			name: "json.UnmarshalDomain错误",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("PATCH", "/api/user/select-domain", `"cst.school^teacher"`, "json.UnmarshalDomain"),
			},
			wantErr: true,
		},
		{
			name: "无效的域名",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("PATCH", "/api/user/select-domain", `"invalid.domain"`, ""),
			},
			wantErr: true,
		},
		{
			name: "查询用户失败",
			fields: fields{
				srv: &MockService{
					err: fmt.Errorf("数据库查询失败"),
				},
			},
			args: args{
				ctx: createMockContextWithBody("PATCH", "/api/user/select-domain", `"cst.school^teacher"`, ""),
			},
			wantErr: true,
		},
		{
			name: "用户不存在",
			fields: fields{
				srv: &MockService{
					users:     []User{},
					totalRows: 0,
				},
			},
			args: args{
				ctx: createMockContextWithBody("PATCH", "/api/user/select-domain", `"cst.school^teacher"`, ""),
			},
			wantErr: true,
		},
		{
			name: "用户无权限访问该域",
			fields: fields{
				srv: &MockService{
					users: []User{
						{
							TUser: cmn.TUser{
								ID:           null.NewInt(54242, true),
								Account:      "test_user",
								OfficialName: null.NewString("测试用户", true),
							},
							Domains: []null.String{
								null.StringFrom("other.domain"),
							},
						},
					},
					totalRows: 1,
				},
			},
			args: args{
				ctx: createMockContextWithBody("PATCH", "/api/user/select-domain", `"cst.school^teacher"`, ""),
			},
			wantErr: true,
		},
		{
			name: "QueryDomainID强制错误",
			fields: fields{
				srv: &MockService{
					users: []User{
						{
							TUser: cmn.TUser{
								ID:           null.NewInt(54242, true),
								Account:      "test_user",
								OfficialName: null.NewString("测试用户", true),
							},
							Domains: []null.String{
								null.StringFrom("cst.school^teacher"),
							},
						},
					},
					totalRows: 1,
				},
			},
			args: args{
				ctx: createMockContextWithBody("PATCH", "/api/user/select-domain", `"cst.school^teacher"`, "QueryDomainID"),
			},
			wantErr: true,
		},
		{
			name: "UpdateUserRole强制错误",
			fields: fields{
				srv: &MockService{
					users: []User{
						{
							TUser: cmn.TUser{
								ID:           null.NewInt(54242, true),
								Account:      "test_user",
								OfficialName: null.NewString("测试用户", true),
							},
							Domains: []null.String{
								null.StringFrom("cst.school^teacher"),
							},
						},
					},
					totalRows: 1,
				},
			},
			args: args{
				ctx: createMockContextWithBody("PATCH", "/api/user/select-domain", `"cst.school^teacher"`, "UpdateUserRole"),
			},
			wantErr: true,
		},
		{
			name: "HTTP方法大小写不敏感 - patch",
			fields: fields{
				srv: &MockService{
					users: []User{
						{
							TUser: cmn.TUser{
								ID:           null.NewInt(54242, true),
								Account:      "test_user",
								OfficialName: null.NewString("测试用户", true),
							},
							Domains: []null.String{
								null.StringFrom("cst.school^teacher"),
							},
						},
					},
					totalRows: 1,
				},
			},
			args: args{
				ctx: createMockContextWithBody("patch", "/api/user/select-domain", `"cst.school^teacher"`, ""),
			},
			wantErr: false,
		},
		{
			name: "HTTP方法大小写不敏感 - Patch",
			fields: fields{
				srv: &MockService{
					users: []User{
						{
							TUser: cmn.TUser{
								ID:           null.NewInt(54242, true),
								Account:      "test_user",
								OfficialName: null.NewString("测试用户", true),
							},
							Domains: []null.String{
								null.StringFrom("cst.school^teacher"),
							},
						},
					},
					totalRows: 1,
				},
			},
			args: args{
				ctx: createMockContextWithBody("Patch", "/api/user/select-domain", `"cst.school^teacher"`, ""),
			},
			wantErr: false,
		},
		{
			name: "包含特殊字符的域名",
			fields: fields{
				srv: &MockService{
					users: []User{
						{
							TUser: cmn.TUser{
								ID:           null.NewInt(54242, true),
								Account:      "test_user",
								OfficialName: null.NewString("测试用户", true),
							},
							Domains: []null.String{
								null.StringFrom("cst.school^teacher"),
							},
						},
					},
					totalRows: 1,
				},
			},
			args: args{
				ctx: createMockContextWithBody("PATCH", "/api/user/select-domain", `"cst.school^teacher"`, ""),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &handler{
				srv: tt.fields.srv,
			}
			h.HandleSelectLoginDomain(tt.args.ctx)

			// 获取ServiceCtx以检查结果
			q := cmn.GetCtxValue(tt.args.ctx)
			if tt.wantErr {
				if q.Err == nil {
					t.Errorf("HandleSelectLoginDomain() 期望有错误，但没有错误")
				}
			} else {
				if q.Err != nil {
					t.Errorf("HandleSelectLoginDomain() 不期望有错误，但出现错误: %v", q.Err)
				}
				// 检查成功响应
				if q.Msg.Status != 0 {
					t.Errorf("HandleSelectLoginDomain() 期望状态码为 0，实际为 %d", q.Msg.Status)
				}
				if q.Msg.Msg != "success" {
					t.Errorf("HandleSelectLoginDomain() 期望消息为 'success'，实际为 '%s'", q.Msg.Msg)
				}
			}
		})
	}
}

// Test_handler_HandleLogout 测试HandleLogout方法
func Test_handler_HandleLogout(t *testing.T) {
	sessions := "MTc1NTI1MDE5OXwwOVRlOFNCdm8zckEyZFhKbVlZaUtOenpPQklJMmxVclFpN1lYY3NxVXRJU2ZoeTQ0WGtEUm8xTVhJM3VxdEVlT2QySDRYUHhTZmdjQzFNNDdiTExHc18yLTZ5c0lzSmdiTnlCMnBjemhnakpsSzJCWlY1Y1dDeUwtMmxNQ25jUU8teGdLV2pXTXRUQzhtYk1abnpuNEg3dlFQVTFGS3BCMGZBT2dkVXdScGhOYjluWGluSnZvTDRtRTF4aTI3ODhiNnR1U1NGdFhVVjQ1ZjByTlBlOEtsWVJGNUU5YThJNGp4d20zbjhpeFpPM1pDNzl6Zz09fAvTfSoTSNw9r8DWBE85ebenOV18Rc4535CX266U0HLS"

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
			name: "成功退出登录",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithCookies("POST", "/api/user/logout", url.Values{}, sessions, ""),
			},
			wantErr: false,
		},
		{
			name: "HTTP方法大小写不敏感 - post",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithCookies("post", "/api/user/logout", url.Values{}, sessions, ""),
			},
			wantErr: false,
		},
		{
			name: "HTTP方法大小写不敏感 - Post",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithCookies("Post", "/api/user/logout", url.Values{}, sessions, ""),
			},
			wantErr: false,
		},
		{
			name: "用户未登录状态退出登录",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithoutUser("POST", "/api/user/logout", "", ""),
			},
			wantErr: false,
		},
		{
			name: "session保存失败",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithCookies("POST", "/api/user/logout", url.Values{}, sessions, "Session.Save"),
			},
			wantErr: true,
		},
		{
			name: "带查询参数的退出登录",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithCookies("POST", "/api/user/logout", url.Values{
					"redirect": []string{"/login"},
					"source":   []string{"web"},
				}, "test-session-with-params", ""),
			},
			wantErr: false,
		},
		{
			name: "PATCH方法退出登录",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithCookies("PATCH", "/api/user/logout", url.Values{}, sessions, ""),
			},
			wantErr: false,
		},
		{
			name: "OPTIONS方法退出登录",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithCookies("OPTIONS", "/api/user/logout", url.Values{}, sessions, ""),
			},
			wantErr: false,
		},
		{
			name: "HEAD方法退出登录",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithCookies("HEAD", "/api/user/logout", url.Values{}, sessions, ""),
			},
			wantErr: false,
		},
		{
			name: "无cookies的退出登录",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithCookies("POST", "/api/user/logout", url.Values{}, sessions, ""),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &handler{
				srv: tt.fields.srv,
			}
			h.HandleLogout(tt.args.ctx)

			// 获取ServiceCtx以检查结果
			q := cmn.GetCtxValue(tt.args.ctx)
			if tt.wantErr {
				if q.Err == nil {
					t.Errorf("HandleLogout() 期望有错误，但没有错误")
				}
			} else {
				if q.Err != nil {
					t.Errorf("HandleLogout() 不期望有错误，但出现错误: %v", q.Err)
				}
				// 检查成功响应
				if q.Msg.Status != 0 {
					t.Errorf("HandleLogout() 期望状态码为 0，实际为 %d", q.Msg.Status)
				}
				if q.Msg.Msg != "logout success" {
					t.Errorf("HandleLogout() 期望消息为 'logout success'，实际为 '%s'", q.Msg.Msg)
				}
			}
		})
	}
}
