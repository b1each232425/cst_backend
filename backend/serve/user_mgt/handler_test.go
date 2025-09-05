package user_mgt

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/wneessen/go-mail"
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

	initTestData()
	defer clearTestData()

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

// Test_handler_HandleSendValidationCodeEmail 测试HandleSendValidationCodeEmail方法
func Test_handler_HandleSendValidationCodeEmail(t *testing.T) {
	type fields struct {
		srv Service
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name         string
		fields       fields
		args         args
		wantErr      bool
		desc         string
		setupRedis   func() // 用于设置Redis验证码
		cleanupRedis func() // 用于清理Redis数据
	}{
		{
			name: "成功发送验证码邮件",
			fields: fields{
				srv: &MockService{
					SendEmailFunc: func(ctx context.Context, recipient, subject, body string, contentType mail.ContentType) error {
						return nil
					},
				},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user/send-validation-code", url.Values{
					"recipient": []string{"test@example.com"},
				}, ""),
			},
			setupRedis: func() {
				rdb := cmn.GetRedisConn()
				if rdb != nil {
					rdb.Set(context.Background(), "verify:email:test@example.com", "123456", time.Minute*15)
				}
			},
			cleanupRedis: func() {
				rdb := cmn.GetRedisConn()
				if rdb != nil {
					rdb.Del(context.Background(), "verify:email:test@example.com")
				}
			},
			wantErr: false,
			desc:    "测试成功发送验证码邮件到有效邮箱地址",
		},
		{
			name: "不支持的HTTP方法 - POST",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContext("POST", "/api/user/send-validation-code", url.Values{
					"recipient": []string{"test@example.com"},
				}, ""),
			},
			wantErr: true,
			desc:    "测试POST方法应该返回错误",
		},
		{
			name: "不支持的HTTP方法 - PUT",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContext("PUT", "/api/user/send-validation-code", url.Values{
					"recipient": []string{"test@example.com"},
				}, ""),
			},
			wantErr: true,
			desc:    "测试PUT方法应该返回错误",
		},
		{
			name: "不支持的HTTP方法 - DELETE",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContext("DELETE", "/api/user/send-validation-code", url.Values{
					"recipient": []string{"test@example.com"},
				}, ""),
			},
			wantErr: true,
			desc:    "测试DELETE方法应该返回错误",
		},
		{
			name: "HTTP方法大小写不敏感 - get",
			fields: fields{
				srv: &MockService{
					SendEmailFunc: func(ctx context.Context, recipient, subject, body string, contentType mail.ContentType) error {
						return nil
					},
				},
			},
			args: args{
				ctx: createMockContext("get", "/api/user/send-validation-code", url.Values{
					"recipient": []string{"test@example.com"},
				}, ""),
			},
			setupRedis: func() {
				rdb := cmn.GetRedisConn()
				if rdb != nil {
					rdb.Set(context.Background(), "verify:email:test@example.com", "123456", time.Minute*15)
				}
			},
			cleanupRedis: func() {
				rdb := cmn.GetRedisConn()
				if rdb != nil {
					rdb.Del(context.Background(), "verify:email:test@example.com")
				}
			},
			wantErr: false,
			desc:    "测试小写get方法应该成功",
		},
		{
			name: "HTTP方法大小写不敏感 - Get",
			fields: fields{
				srv: &MockService{
					SendEmailFunc: func(ctx context.Context, recipient, subject, body string, contentType mail.ContentType) error {
						return nil
					},
				},
			},
			args: args{
				ctx: createMockContext("Get", "/api/user/send-validation-code", url.Values{
					"recipient": []string{"test@example.com"},
				}, ""),
			},
			setupRedis: func() {
				rdb := cmn.GetRedisConn()
				if rdb != nil {
					rdb.Set(context.Background(), "verify:email:test@example.com", "123456", time.Minute*15)
				}
			},
			cleanupRedis: func() {
				rdb := cmn.GetRedisConn()
				if rdb != nil {
					rdb.Del(context.Background(), "verify:email:test@example.com")
				}
			},
			wantErr: false,
			desc:    "测试首字母大写Get方法应该成功",
		},
		{
			name: "缺少邮箱地址参数",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user/send-validation-code", url.Values{}, ""),
			},
			wantErr: true,
			desc:    "测试缺少recipient参数应该返回错误",
		},
		{
			name: "邮箱地址参数为空字符串",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user/send-validation-code", url.Values{
					"recipient": []string{""},
				}, ""),
			},
			wantErr: true,
			desc:    "测试空邮箱地址应该返回错误",
		},
		{
			name: "强制rand.Int错误",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user/send-validation-code", url.Values{
					"recipient": []string{"test@example.com"},
				}, "rand.Int"),
			},
			wantErr: true,
			desc:    "测试强制rand.Int错误",
		},
		{
			name: "SendEmail服务错误",
			fields: fields{
				srv: &MockService{
					SendEmailFunc: func(ctx context.Context, recipient, subject, body string, contentType mail.ContentType) error {
						return fmt.Errorf("邮件服务器连接失败")
					},
				},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user/send-validation-code", url.Values{
					"recipient": []string{"test@example.com"},
				}, ""),
			},
			wantErr: true,
			desc:    "测试邮件发送失败应该返回错误",
		},
		{
			name: "强制rdb.Set错误",
			fields: fields{
				srv: &MockService{
					SendEmailFunc: func(ctx context.Context, recipient, subject, body string, contentType mail.ContentType) error {
						return nil
					},
				},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user/send-validation-code", url.Values{
					"recipient": []string{"test@example.com"},
				}, "rdb.Set"),
			},
			wantErr: true,
			desc:    "测试强制Redis Set错误",
		},
		{
			name: "复杂邮箱地址测试",
			fields: fields{
				srv: &MockService{
					SendEmailFunc: func(ctx context.Context, recipient, subject, body string, contentType mail.ContentType) error {
						return nil
					},
				},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user/send-validation-code", url.Values{
					"recipient": []string{"user.name+tag@example-domain.co.uk"},
				}, ""),
			},
			setupRedis: func() {
				rdb := cmn.GetRedisConn()
				if rdb != nil {
					rdb.Set(context.Background(), "verify:email:user.name+tag@example-domain.co.uk", "123456", time.Minute*15)
				}
			},
			cleanupRedis: func() {
				rdb := cmn.GetRedisConn()
				if rdb != nil {
					rdb.Del(context.Background(), "verify:email:user.name+tag@example-domain.co.uk")
				}
			},
			wantErr: false,
			desc:    "测试复杂格式的邮箱地址",
		},
		{
			name: "包含特殊字符的邮箱地址",
			fields: fields{
				srv: &MockService{
					SendEmailFunc: func(ctx context.Context, recipient, subject, body string, contentType mail.ContentType) error {
						return nil
					},
				},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user/send-validation-code", url.Values{
					"recipient": []string{"test+special@example.com"},
				}, ""),
			},
			setupRedis: func() {
				rdb := cmn.GetRedisConn()
				if rdb != nil {
					rdb.Set(context.Background(), "verify:email:test+special@example.com", "123456", time.Minute*15)
				}
			},
			cleanupRedis: func() {
				rdb := cmn.GetRedisConn()
				if rdb != nil {
					rdb.Del(context.Background(), "verify:email:test+special@example.com")
				}
			},
			wantErr: false,
			desc:    "测试包含特殊字符的邮箱地址",
		},
		{
			name: "大写邮箱地址测试",
			fields: fields{
				srv: &MockService{
					SendEmailFunc: func(ctx context.Context, recipient, subject, body string, contentType mail.ContentType) error {
						return nil
					},
				},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user/send-validation-code", url.Values{
					"recipient": []string{"TEST@EXAMPLE.COM"},
				}, ""),
			},
			setupRedis: func() {
				rdb := cmn.GetRedisConn()
				if rdb != nil {
					rdb.Set(context.Background(), "verify:email:test@example.com", "123456", time.Minute*15)
				}
			},
			cleanupRedis: func() {
				rdb := cmn.GetRedisConn()
				if rdb != nil {
					rdb.Del(context.Background(), "verify:email:test@example.com")
				}
			},
			wantErr: false,
			desc:    "测试大写邮箱地址",
		},
		{
			name: "中文域名邮箱测试",
			fields: fields{
				srv: &MockService{
					SendEmailFunc: func(ctx context.Context, recipient, subject, body string, contentType mail.ContentType) error {
						return nil
					},
				},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user/send-validation-code", url.Values{
					"recipient": []string{"test@测试.com"},
				}, ""),
			},
			setupRedis: func() {
				rdb := cmn.GetRedisConn()
				if rdb != nil {
					rdb.Set(context.Background(), "verify:email:test@测试.com", "123456", time.Minute*15)
				}
			},
			cleanupRedis: func() {
				rdb := cmn.GetRedisConn()
				if rdb != nil {
					rdb.Del(context.Background(), "verify:email:test@测试.com")
				}
			},
			wantErr: false,
			desc:    "测试中文域名邮箱地址",
		},
		{
			name: "验证邮件内容和格式",
			fields: fields{
				srv: &MockService{
					SendEmailFunc: func(ctx context.Context, recipient, subject, body string, contentType mail.ContentType) error {
						// 验证邮件参数
						if recipient != "verify@example.com" {
							return fmt.Errorf("收件人不正确: %s", recipient)
						}
						if subject != "3min学习平台 - 验证您的电子邮件地址" {
							return fmt.Errorf("邮件主题不正确: %s", subject)
						}
						if !strings.Contains(body, "3min学习平台") {
							return fmt.Errorf("邮件内容缺少平台名称")
						}
						if !strings.Contains(body, "15") {
							return fmt.Errorf("邮件内容缺少有效期信息")
						}
						if contentType != mail.TypeTextHTML {
							return fmt.Errorf("邮件内容类型不正确: %s", contentType)
						}
						return nil
					},
				},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user/send-validation-code", url.Values{
					"recipient": []string{"verify@example.com"},
				}, ""),
			},
			setupRedis: func() {
				rdb := cmn.GetRedisConn()
				if rdb != nil {
					rdb.Set(context.Background(), "verify:email:verify@example.com", "123456", time.Minute*15)
				}
			},
			cleanupRedis: func() {
				rdb := cmn.GetRedisConn()
				if rdb != nil {
					rdb.Del(context.Background(), "verify:email:verify@example.com")
				}
			},
			wantErr: false,
			desc:    "测试验证邮件内容和格式是否正确",
		},
	}

	initTestData()
	defer clearTestData()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("开始测试: %s", tt.desc)

			h := &handler{
				srv: tt.fields.srv,
			}

			// 设置Redis测试环境
			if tt.setupRedis != nil {
				tt.setupRedis()
			}
			if tt.cleanupRedis != nil {
				defer tt.cleanupRedis()
			}

			h.HandleSendValidationCodeEmail(tt.args.ctx)

			// 获取ServiceCtx以检查结果
			q := cmn.GetCtxValue(tt.args.ctx)
			if tt.wantErr {
				if q.Err == nil {
					t.Errorf("HandleSendValidationCodeEmail() 期望有错误，但没有错误")
				} else {
					t.Logf("预期错误已正确返回: %v", q.Err)
				}
			} else {
				if q.Err != nil {
					t.Errorf("HandleSendValidationCodeEmail() 不期望有错误，但出现错误: %v", q.Err)
				} else {
					// 检查成功响应
					if q.Msg.Status != 0 {
						t.Errorf("HandleSendValidationCodeEmail() 期望状态码为 0，实际为 %d", q.Msg.Status)
					}
					if q.Msg.Msg != "success" {
						t.Errorf("HandleSendValidationCodeEmail() 期望消息为 'success'，实际为 '%s'", q.Msg.Msg)
					}
					t.Logf("验证码邮件发送成功")

					// 验证Redis中是否保存了验证码
					if tt.setupRedis != nil {
						rdb := cmn.GetRedisConn()
						q := cmn.GetCtxValue(tt.args.ctx)
						query := q.R.URL.Query()
						recipient := query.Get("recipient")
						key := "verify:email:" + strings.ToLower(recipient)
						code, err := rdb.Get(tt.args.ctx, key).Result()
						if err != nil {
							t.Errorf("验证码未保存到Redis: %v", err)
						} else {
							if len(code) != 6 {
								t.Errorf("验证码长度不正确，期望6位，实际%d位: %s", len(code), code)
							}
							t.Logf("验证码已成功保存到Redis: %s", code)
						}
					}
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
					ValidateUserToBeInsertFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
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
					ValidateUserToBeInsertFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
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
					ValidateUserToBeInsertFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
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
					ValidateUserToBeInsertFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
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
			name: "请求体为空",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("POST", "/api/user/validate", nil, ""),
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
					ValidateUserToBeInsertFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
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
					ValidateUserToBeInsertFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
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
					ValidateUserToBeInsertFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
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
							"Domains": ["assess^teacher"],
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
					ValidateUserToBeInsertFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
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
					ValidateUserToBeInsertFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
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
					ValidateUserToBeInsertFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
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

	initTestData()
	defer clearTestData()

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

// Test_handler_HandleCurrentUser 测试HandleCurrentUser方法
func Test_handler_HandleCurrentUser(t *testing.T) {
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
		// 通用测试用例
		{
			name: "不支持的HTTP方法",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContext("POST", "/api/user/me", url.Values{}, ""),
			},
			wantErr: true,
		},
		{
			name: "不支持的HTTP方法 - DELETE",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContext("DELETE", "/api/user/me", url.Values{}, ""),
			},
			wantErr: true,
		},
		{
			name: "不支持的HTTP方法 - PATCH",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContext("PATCH", "/api/user/me", url.Values{}, ""),
			},
			wantErr: true,
		},
		{
			name: "用户未登录",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithoutUser("GET", "/api/user/me", "", ""),
			},
			wantErr: true,
		},

		// GET方法测试用例
		{
			name: "GET｜成功查询用户信息",
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
								null.StringFrom("assess^teacher"),
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
				ctx: createMockContext("GET", "/api/user/me", url.Values{}, ""),
			},
			wantErr: false,
		},
		{
			name: "GET｜查询用户失败",
			fields: fields{
				srv: &MockService{
					err: fmt.Errorf("数据库查询失败"),
				},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user/me", url.Values{}, ""),
			},
			wantErr: true,
		},
		{
			name: "GET｜用户不存在",
			fields: fields{
				srv: &MockService{
					users:     []User{},
					totalRows: 0,
				},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user/me", url.Values{}, ""),
			},
			wantErr: true,
		},
		{
			name: "GET｜json.Marshal强制错误",
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
								null.StringFrom("assess^teacher"),
							},
						},
					},
					totalRows: 1,
				},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user/me", url.Values{}, "json.Marshal"),
			},
			wantErr: true,
		},
		{
			name: "GET｜HTTP方法大小写不敏感 - get",
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
								null.StringFrom("assess^teacher"),
							},
						},
					},
					totalRows: 1,
				},
			},
			args: args{
				ctx: createMockContext("get", "/api/user/me", url.Values{}, ""),
			},
			wantErr: false,
		},
		{
			name: "GET｜HTTP方法大小写不敏感 - Get",
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
								null.StringFrom("assess^teacher"),
							},
						},
					},
					totalRows: 1,
				},
			},
			args: args{
				ctx: createMockContext("Get", "/api/user/me", url.Values{}, ""),
			},
			wantErr: false,
		},
		{
			name: "GET｜用户信息包含空值",
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
				ctx: createMockContext("GET", "/api/user/me", url.Values{}, ""),
			},
			wantErr: false,
		},
		{
			name: "GET｜用户信息包含多个域和API",
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
				ctx: createMockContext("GET", "/api/user/me", url.Values{}, ""),
			},
			wantErr: false,
		},

		// PUT方法测试用例
		{
			name: "PUT｜成功更新当前用户信息",
			fields: fields{
				srv: &MockService{
					users: []User{
						{
							TUser: cmn.TUser{
								ID:           null.NewInt(54242, true),
								Account:      "test_user",
								OfficialName: null.NewString("更新后的用户", true),
								Category:     "normal",
							},
							Domains: []null.String{
								null.StringFrom("assess^teacher"),
							},
						},
					},
					totalRows: 1,
					ValidateUserToBeUpdateFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, error) {
						return users, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("PUT", "/api/user/me", `{"Account":"test_user","OfficialName":"更新后的用户"}`, ""),
			},
			wantErr: false,
		},
		{
			name: "PUT｜更新的用户列表为空",
			fields: fields{
				srv: &MockService{
					users:     []User{},
					totalRows: 0,
					ValidateUserToBeUpdateFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, error) {
						return users, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("PUT", "/api/user/me", `{"Account":"test_user","OfficialName":"更新后的用户"}`, ""),
			},
			wantErr: true,
		},
		{
			name: "PUT｜读取请求体失败",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("PUT", "/api/user/me", `{"Account":"test_user"}`, "io.ReadAll"),
			},
			wantErr: true,
		},
		{
			name: "PUT｜请求体为空",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("PUT", "/api/user/me", nil, ""),
			},
			wantErr: true,
		},
		{
			name: "PUT｜io.Close强制错误",
			fields: fields{
				srv: &MockService{
					users: []User{
						{
							TUser: cmn.TUser{
								ID:           null.NewInt(54242, true),
								Account:      "test_user",
								OfficialName: null.NewString("更新后的用户", true),
								Category:     "normal",
							},
							Domains: []null.String{
								null.StringFrom("assess^teacher"),
							},
						},
					},
					totalRows: 1,
					ValidateUserToBeUpdateFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, error) {
						return users, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("PUT", "/api/user/me", `{"Account":"test_user","OfficialName":"更新后的用户"}`, "io.Close"),
			},
			wantErr: false,
		},
		{
			name: "PUT｜JSON解析失败",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("PUT", "/api/user/me", `{"invalid": json}`, "json.Unmarshal"),
			},
			wantErr: true,
		},
		{
			name: "PUT｜用户数据JSON解析失败",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("PUT", "/api/user/me", `{"invalid": "user data"}`, "json.Unmarshal_user"),
			},
			wantErr: true,
		},
		{
			name: "PUT｜事务开始失败",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("PUT", "/api/user/me", `{"Account":"test_user"}`, "tx.Begin"),
			},
			wantErr: true,
		},
		{
			name: "PUT｜用户验证失败",
			fields: fields{
				srv: &MockService{
					ValidateUserToBeUpdateFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, error) {
						return nil, nil, fmt.Errorf("用户验证失败")
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("PUT", "/api/user/me", `{"Account":"test_user"}`, ""),
			},
			wantErr: true,
		},
		{
			name: "PUT｜存在无效用户",
			fields: fields{
				srv: &MockService{
					ValidateUserToBeUpdateFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, error) {
						invalidUser := User{
							TUser: cmn.TUser{
								ID:      null.NewInt(54242, true),
								Account: "invalid_user",
							},
							ErrorMsg: []null.String{
								null.NewString("账号格式不正确", true),
							},
						}
						return []User{}, []User{invalidUser}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("PUT", "/api/user/me", `{"Account":"invalid_user"}`, ""),
			},
			wantErr: false,
		},
		{
			name: "PUT｜无效用户序列化失败",
			fields: fields{
				srv: &MockService{
					ValidateUserToBeUpdateFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, error) {
						invalidUser := User{
							TUser: cmn.TUser{
								ID:      null.NewInt(54242, true),
								Account: "invalid_user",
							},
							ErrorMsg: []null.String{
								null.NewString("账号格式不正确", true),
							},
						}
						return []User{}, []User{invalidUser}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("PUT", "/api/user/me", `{"Account":"invalid_user"}`, "json.Marshal"),
			},
			wantErr: true,
		},
		{
			name: "PUT｜更新用户失败",
			fields: fields{
				srv: &MockService{
					ValidateUserToBeUpdateFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, error) {
						return users, []User{}, nil
					},
					err: fmt.Errorf("更新用户失败"),
				},
			},
			args: args{
				ctx: createMockContextWithBody("PUT", "/api/user/me", `{"Account":"test_user"}`, ""),
			},
			wantErr: true,
		},
		{
			name: "PUT｜更新结果序列化失败",
			fields: fields{
				srv: &MockService{
					ValidateUserToBeUpdateFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, error) {
						return users, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("PUT", "/api/user/me", `{"Account":"test_user"}`, "json.Marshal"),
			},
			wantErr: true,
		},
		{
			name: "PUT｜HTTP方法大小写不敏感 - put",
			fields: fields{
				srv: &MockService{
					users: []User{
						{
							TUser: cmn.TUser{
								ID:           null.NewInt(54242, true),
								Account:      "test_user",
								OfficialName: null.NewString("更新后的用户", true),
								Category:     "normal",
							},
							Domains: []null.String{
								null.StringFrom("assess^teacher"),
							},
						},
					},
					totalRows: 1,
					ValidateUserToBeUpdateFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, error) {
						return users, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("put", "/api/user/me", `{"Account":"test_user"}`, ""),
			},
			wantErr: false,
		},
		{
			name: "PUT｜HTTP方法大小写不敏感 - Put",
			fields: fields{
				srv: &MockService{
					users: []User{
						{
							TUser: cmn.TUser{
								ID:           null.NewInt(54242, true),
								Account:      "test_user",
								OfficialName: null.NewString("更新后的用户", true),
								Category:     "normal",
							},
							Domains: []null.String{
								null.StringFrom("assess^teacher"),
							},
						},
					},
					totalRows: 1,
					ValidateUserToBeUpdateFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, error) {
						return users, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("Put", "/api/user/me", `{"Account":"test_user"}`, ""),
			},
			wantErr: false,
		},
		{
			name: "PUT｜包含完整用户信息的更新",
			fields: fields{
				srv: &MockService{
					users: []User{
						{
							TUser: cmn.TUser{
								ID:           null.NewInt(54242, true),
								Account:      "complete_user",
								OfficialName: null.NewString("完整用户信息", true),
								Gender:       null.NewString("F", true),
								MobilePhone:  null.NewString("13900139000", true),
								Email:        null.NewString("complete@example.com", true),
								Category:     "normal",
								Status:       null.NewString("active", true),
							},
							Domains: []null.String{
								null.StringFrom("assess^teacher"),
								null.StringFrom("other.domain"),
							},
						},
					},
					totalRows: 1,
					ValidateUserToBeUpdateFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, error) {
						return users, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("PUT", "/api/user/me", `{"Account":"complete_user","OfficialName":"完整用户信息","Gender":"F","MobilePhone":"13900139000","Email":"complete@example.com"}`, ""),
			},
			wantErr: false,
		},
		{
			name: "PUT｜包含特殊字符的用户数据",
			fields: fields{
				srv: &MockService{
					users: []User{
						{
							TUser: cmn.TUser{
								ID:           null.NewInt(54242, true),
								Account:      "test_user",
								OfficialName: null.NewString("更新后的用户", true),
								Category:     "normal",
							},
							Domains: []null.String{
								null.StringFrom("assess^teacher"),
							},
						},
					},
					totalRows: 1,
					ValidateUserToBeUpdateFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, error) {
						return users, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("PUT", "/api/user/me", `{"Account":"user@domain.com","OfficialName":"用户\"特殊\"字符"}`, ""),
			},
			wantErr: false,
		},
		{
			name: "PUT｜json.Marshal强制错误",
			fields: fields{
				srv: &MockService{
					users: []User{
						{
							TUser: cmn.TUser{
								ID:           null.NewInt(54242, true),
								Account:      "test_user",
								OfficialName: null.NewString("更新后的用户", true),
								Category:     "normal",
							},
							Domains: []null.String{
								null.StringFrom("assess^teacher"),
							},
						},
					},
					totalRows: 1,
					ValidateUserToBeUpdateFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, error) {
						return users, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("PUT", "/api/user/me", `{"Account":"test_user","OfficialName":"更新后的用户"}`, "json.Marshal"),
			},
			wantErr: true,
		},
	}

	initTestData()
	defer clearTestData()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &handler{
				srv: tt.fields.srv,
			}
			h.HandleCurrentUser(tt.args.ctx)

			// 获取ServiceCtx以检查结果
			q := cmn.GetCtxValue(tt.args.ctx)
			if tt.wantErr {
				if q.Err == nil {
					t.Errorf("HandleCurrentUser() 期望有错误，但没有错误")
				}
			} else {
				if q.Err != nil {
					t.Errorf("HandleCurrentUser() 不期望有错误，但出现错误: %v", q.Err)
				}
				// 检查成功响应
				if q.Msg.Status != 0 && q.Msg.Status != 405 {
					t.Errorf("HandleCurrentUser() 期望状态码为 0，实际为 %d", q.Msg.Status)
				}
				// 检查返回的数据是否为用户信息
				if q.Msg.Data == nil {
					t.Errorf("HandleCurrentUser() 期望返回用户数据，但数据为空")
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
				},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user", url.Values{
					"page":     {"1"},
					"pageSize": {"10"},
				}, "no-access"),
			},
			wantErr: true,
		},
		{
			name: "CheckUserAPIAccessible失败",
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
				}, "CheckUserAPIAccessible"),
			},
			wantErr: true,
		},
		{
			name: "查询当前用户权限信息失败",
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
				}, "GetUserAuthority"),
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
				ctx: createMockContext("YES", "/api/user", url.Values{}, ""),
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
		{
			name: "触发no-login错误",
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
				}, "no-login"),
			},
			wantErr: true,
		},
		{
			name: "触发IsDomainExist错误",
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
					"domain":      {"nonexistent_domain"},
					"createTime":  {strconv.FormatInt(time.Now().Unix()-86400, 10)},
				}, "IsDomainExist"),
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
				ctx: createMockContextWithBody("POST", "/api/user", nil, ""),
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
					ValidateUserToBeInsertFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
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
				ctx: createMockContextWithBody("POST", "/api/user", nil, ""),
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
					ValidateUserToBeInsertFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
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
					ValidateUserToBeInsertFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
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
					ValidateUserToBeInsertFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
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
					ValidateUserToBeInsertFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
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
					ValidateUserToBeInsertFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
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
		{
			name: "触发CheckUserAPIAccessible错误",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("POST", "/api/user", `[{
					"Account": "new_user_001",
					"OfficialName": "新用户001"
				}]`, "CheckUserAPIAccessible"),
			},
			wantErr: true,
		},
		{
			name: "触发no-access错误",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("POST", "/api/user", `[{
					"Account": "new_user_001",
					"OfficialName": "新用户001"
				}]`, "no-access"),
			},
			wantErr: true,
		},

		// PUT方法测试用例
		{
			name: "PUT - 成功更新用户",
			fields: fields{
				srv: &MockService{
					ValidateUserToBeUpdateFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, error) {
						return users, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("PUT", "/api/user", `[{
					"ID": 1,
					"Account": "update_user_001",
					"OfficialName": "更新用户001",
					"Email": "update001@example.com"
				}]`, ""),
			},
			wantErr: false,
		},
		{
			name: "PUT - 权限检查失败",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("PUT", "/api/user", `[{
					"ID": 1,
					"Account": "test_user"
				}]`, "CheckUserAPIAccessible"),
			},
			wantErr: true,
		},
		{
			name: "PUT - 用户无权限访问API",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("PUT", "/api/user", `[{
					"ID": 1,
					"Account": "test_user"
				}]`, "no-access"),
			},
			wantErr: true,
		},
		{
			name: "PUT - 读取请求体失败",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("PUT", "/api/user", `[{
					"ID": 1,
					"Account": "test_user"
				}]`, "io.ReadAll"),
			},
			wantErr: true,
		},
		{
			name: "PUT - 关闭请求体失败",
			fields: fields{
				srv: &MockService{
					ValidateUserToBeUpdateFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, error) {
						return users, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("PUT", "/api/user", `[{
					"ID": 1,
					"Account": "test_user"
				}]`, "io.Close"),
			},
			wantErr: false, // io.Close错误不会导致整个请求失败
		},
		{
			name: "PUT - 请求体为空",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("PUT", "/api/user", nil, ""),
			},
			wantErr: true,
		},
		{
			name: "PUT - JSON解析失败",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("PUT", "/api/user", `[{
					"ID": 1,
					"Account": "test_user"
				}]`, "json.Unmarshal"),
			},
			wantErr: true,
		},
		{
			name: "PUT - 用户数据JSON解析失败",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("PUT", "/api/user", `invalid json data`, ""),
			},
			wantErr: true,
		},
		{
			name: "PUT - 用户列表为空",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("PUT", "/api/user", `[]`, ""),
			},
			wantErr: true,
		},
		{
			name: "PUT - 事务开始失败",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("PUT", "/api/user", `[{
					"ID": 1,
					"Account": "test_user"
				}]`, "tx.Begin"),
			},
			wantErr: true,
		},
		{
			name: "PUT - 用户验证失败",
			fields: fields{
				srv: &MockService{
					ValidateUserToBeUpdateFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, error) {
						return nil, nil, fmt.Errorf("用户验证失败")
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("PUT", "/api/user", `[{
					"ID": 1,
					"Account": "test_user"
				}]`, ""),
			},
			wantErr: true,
		},
		{
			name: "PUT - 存在无效用户",
			fields: fields{
				srv: &MockService{
					ValidateUserToBeUpdateFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, error) {
						return []User{}, []User{
							{
								TUser: cmn.TUser{
									ID:      null.NewInt(1, true),
									Account: "invalid_user",
								},
								ErrorMsg: []null.String{
									null.NewString("邮箱格式不正确", true),
								},
							},
						}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("PUT", "/api/user", `[{
					"ID": 1,
					"Account": "invalid_user",
					"Email": "invalid-email"
				}]`, ""),
			},
			wantErr: false, // 返回405状态，但不是错误
		},
		{
			name: "PUT - 无效用户序列化失败",
			fields: fields{
				srv: &MockService{
					ValidateUserToBeUpdateFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, error) {
						return []User{}, []User{
							{
								TUser: cmn.TUser{
									ID:      null.NewInt(1, true),
									Account: "invalid_user",
								},
							},
						}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("PUT", "/api/user", `[{
					"ID": 1,
					"Account": "invalid_user"
				}]`, "json.Marshal"),
			},
			wantErr: true,
		},
		{
			name: "PUT - 更新用户失败",
			fields: fields{
				srv: &MockService{
					err: fmt.Errorf("数据库更新失败"),
					ValidateUserToBeUpdateFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, error) {
						return users, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("PUT", "/api/user", `[{
					"ID": 1,
					"Account": "test_user"
				}]`, ""),
			},
			wantErr: true,
		},
		{
			name: "PUT - 更新结果序列化失败",
			fields: fields{
				srv: &MockService{
					ValidateUserToBeUpdateFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, error) {
						return users, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("PUT", "/api/user", `[{
					"ID": 1,
					"Account": "test_user"
				}]`, "json.Marshal"),
			},
			wantErr: true,
		},
		{
			name: "PUT - HTTP方法大小写不敏感 - put",
			fields: fields{
				srv: &MockService{
					ValidateUserToBeUpdateFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, error) {
						return users, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("put", "/api/user", `[{
					"ID": 1,
					"Account": "test_user"
				}]`, ""),
			},
			wantErr: false,
		},
		{
			name: "PUT - HTTP方法大小写不敏感 - Put",
			fields: fields{
				srv: &MockService{
					ValidateUserToBeUpdateFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, error) {
						return users, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("Put", "/api/user", `[{
					"ID": 1,
					"Account": "test_user"
				}]`, ""),
			},
			wantErr: false,
		},
		{
			name: "PUT - 包含多个用户的更新",
			fields: fields{
				srv: &MockService{
					ValidateUserToBeUpdateFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, error) {
						return users, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("PUT", "/api/user", `[{
					"ID": 1,
					"Account": "user_001",
					"OfficialName": "用户001"
				}, {
					"ID": 2,
					"Account": "user_002",
					"OfficialName": "用户002"
				}]`, ""),
			},
			wantErr: false,
		},
		{
			name: "PUT - 包含特殊字符的用户数据",
			fields: fields{
				srv: &MockService{
					ValidateUserToBeUpdateFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, error) {
						return users, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("PUT", "/api/user", `[{
					"ID": 1,
					"Account": "test@user#001",
					"OfficialName": "测试用户@#$%"
				}]`, ""),
			},
			wantErr: false,
		},
		{
			name: "PUT - 包含Unicode字符的用户数据",
			fields: fields{
				srv: &MockService{
					ValidateUserToBeUpdateFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, error) {
						return users, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBody("PUT", "/api/user", `[{
					"ID": 1,
					"Account": "用户账号",
					"OfficialName": "张三李四王五"
				}]`, ""),
			},
			wantErr: false,
		},
	}

	initTestData()
	defer clearTestData()

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
				if response.Status == -1 {
					t.Errorf("HandleUser() 不期望状态为-1，但得到: %d，消息: %s", response.Status, response.Msg)
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
								null.StringFrom("assess^superAdmin"),
								null.StringFrom("other.domain"),
							},
						},
					},
					totalRows: 1,
				},
			},
			args: args{
				ctx: createMockContextWithBody("PATCH", "/api/user/select-domain", `"assess^superAdmin"`, ""),
			},
			wantErr: false,
		},
		{
			name: "不支持的HTTP方法",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("GET", "/api/user/select-domain", `"assess^teacher"`, ""),
			},
			wantErr: true,
		},
		{
			name: "不支持的HTTP方法 - POST",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("POST", "/api/user/select-domain", `"assess^teacher"`, ""),
			},
			wantErr: true,
		},
		{
			name: "不支持的HTTP方法 - PUT",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("PUT", "/api/user/select-domain", `"assess^teacher"`, ""),
			},
			wantErr: true,
		},
		{
			name: "不支持的HTTP方法 - DELETE",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("DELETE", "/api/user/select-domain", `"assess^teacher"`, ""),
			},
			wantErr: true,
		},
		{
			name: "io.ReadAll错误",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("PATCH", "/api/user/select-domain", `"assess^teacher"`, "io.ReadAll"),
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
								null.StringFrom("assess^teacher"),
							},
						},
					},
					totalRows: 1,
				},
			},
			args: args{
				ctx: createMockContextWithBody("PATCH", "/api/user/select-domain", `"assess^teacher"`, "io.Close"),
			},
			wantErr: false, // io.Close错误不会导致整个请求失败
		},
		{
			name: "空请求体",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("PATCH", "/api/user/select-domain", nil, ""),
			},
			wantErr: true,
		},
		{
			name: "json.Unmarshal错误",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("PATCH", "/api/user/select-domain", `"assess^teacher"`, "json.Unmarshal"),
			},
			wantErr: true,
		},
		{
			name: "json.UnmarshalDomain错误",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBody("PATCH", "/api/user/select-domain", `"assess^teacher"`, "json.UnmarshalDomain"),
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
				ctx: createMockContextWithBody("PATCH", "/api/user/select-domain", `"assess^teacher"`, ""),
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
				ctx: createMockContextWithBody("PATCH", "/api/user/select-domain", `"assess^teacher"`, ""),
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
				ctx: createMockContextWithBody("PATCH", "/api/user/select-domain", `"assess^teacher"`, ""),
			},
			wantErr: true,
		},
		{
			name: "IsDomainExist强制错误",
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
								null.StringFrom("assess^teacher"),
							},
						},
					},
					totalRows: 1,
				},
			},
			args: args{
				ctx: createMockContextWithBody("PATCH", "/api/user/select-domain", `"assess^teacher"`, "IsDomainExist"),
			},
			wantErr: true,
		},
		{
			name: "domain-not-exist强制错误",
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
								null.StringFrom("assess^teacher"),
							},
						},
					},
					totalRows: 1,
				},
			},
			args: args{
				ctx: createMockContextWithBody("PATCH", "/api/user/select-domain", `"assess^teacher"`, "domain-not-exist"),
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
								null.StringFrom("assess^teacher"),
							},
						},
					},
					totalRows: 1,
				},
			},
			args: args{
				ctx: createMockContextWithBody("PATCH", "/api/user/select-domain", `"assess^teacher"`, "UpdateUserRole"),
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
								null.StringFrom("assess^teacher"),
							},
						},
					},
					totalRows: 1,
				},
			},
			args: args{
				ctx: createMockContextWithBody("patch", "/api/user/select-domain", `"assess^teacher"`, ""),
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
								null.StringFrom("assess^teacher"),
							},
						},
					},
					totalRows: 1,
				},
			},
			args: args{
				ctx: createMockContextWithBody("Patch", "/api/user/select-domain", `"assess^teacher"`, ""),
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
								null.StringFrom("assess^teacher"),
							},
						},
					},
					totalRows: 1,
				},
			},
			args: args{
				ctx: createMockContextWithBody("PATCH", "/api/user/select-domain", `"assess^teacher"`, ""),
			},
			wantErr: false,
		},
	}

	initTestData()
	defer clearTestData()

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

	initTestData()
	defer clearTestData()

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

// Test_handler_HandleRegisterByEmail 测试HandleRegisterByEmail方法
func Test_handler_HandleRegisterByEmail(t *testing.T) {
	type fields struct {
		srv Service
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name         string
		fields       fields
		args         args
		wantErr      bool
		setupRedis   func() // 用于设置Redis验证码
		cleanupRedis func() // 用于清理Redis数据
	}{
		{
			name: "成功注册用户",
			fields: fields{
				srv: &MockService{
					ValidateUserToBeInsertFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
						return users, []User{}, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBodyAndQuery("POST", "/api/user/register", `{
					"Email": "test@example.com",
					"UserToken": "password123",
					"OfficialName": "测试用户"
				}`, url.Values{"verification-code": []string{"123456"}}, ""),
			},
			wantErr: false,
			setupRedis: func() {
				rdb := cmn.GetRedisConn()
				if rdb != nil {
					rdb.Set(context.Background(), "verify:email:test@example.com", "123456", time.Minute*5)
				}
			},
			cleanupRedis: func() {
				rdb := cmn.GetRedisConn()
				if rdb != nil {
					rdb.Del(context.Background(), "verify:email:test@example.com")
				}
			},
		},
		{
			name: "不支持的HTTP方法 - GET",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContext("GET", "/api/user/register", url.Values{}, ""),
			},
			wantErr: true,
		},
		{
			name: "不支持的HTTP方法 - PUT",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContext("PUT", "/api/user/register", url.Values{}, ""),
			},
			wantErr: true,
		},
		{
			name: "不支持的HTTP方法 - DELETE",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContext("DELETE", "/api/user/register", url.Values{}, ""),
			},
			wantErr: true,
		},
		{
			name: "HTTP方法大小写不敏感 - post",
			fields: fields{
				srv: &MockService{
					ValidateUserToBeInsertFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
						return users, []User{}, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBodyAndQuery("post", "/api/user/register", `{
					"Email": "test2@example.com",
					"UserToken": "password123",
					"OfficialName": "测试用户2"
				}`, url.Values{"verification-code": []string{"654321"}}, ""),
			},
			wantErr: false,
			setupRedis: func() {
				rdb := cmn.GetRedisConn()
				if rdb != nil {
					rdb.Set(context.Background(), "verify:email:test2@example.com", "654321", time.Minute*5)
				}
			},
			cleanupRedis: func() {
				rdb := cmn.GetRedisConn()
				if rdb != nil {
					rdb.Del(context.Background(), "verify:email:test2@example.com")
				}
			},
		},
		{
			name: "io.ReadAll强制错误",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBodyAndQuery("POST", "/api/user/register", `{"Email": "test@example.com"}`, url.Values{}, "io.ReadAll"),
			},
			wantErr: true,
		},
		{
			name: "io.Close强制错误",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBodyAndQuery("POST", "/api/user/register", `{"Email": "test@example.com"}`, url.Values{}, "io.Close"),
			},
			wantErr: true,
		},
		{
			name: "请求体为空",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBodyAndQuery("POST", "/api/user/register", "", url.Values{}, ""),
			},
			wantErr: true,
		},
		{
			name: "json.Unmarshal强制错误 - 请求体解析",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBodyAndQuery("POST", "/api/user/register", `{"Email": "test@example.com"}`, url.Values{}, "json.Unmarshal"),
			},
			wantErr: true,
		},
		{
			name: "无效的JSON格式 - 用户数据",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBodyAndQuery("POST", "/api/user/register", `invalid json`, url.Values{}, ""),
			},
			wantErr: true,
		},
		{
			name: "缺少邮箱地址",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBodyAndQuery("POST", "/api/user/register", `{
					"UserToken": "password123",
					"OfficialName": "测试用户"
				}`, url.Values{}, ""),
			},
			wantErr: true,
		},
		{
			name: "缺少密码",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBodyAndQuery("POST", "/api/user/register", `{
					"Email": "test@example.com",
					"OfficialName": "测试用户"
				}`, url.Values{}, ""),
			},
			wantErr: true,
		},
		{
			name: "tx.Begin强制错误",
			fields: fields{
				srv: &MockService{},
			},
			args: args{
				ctx: createMockContextWithBodyAndQuery("POST", "/api/user/register", `{
					"Email": "test@example.com",
					"UserToken": "password123",
					"OfficialName": "测试用户"
				}`, url.Values{}, "tx.Begin"),
			},
			wantErr: true,
		},
		{
			name: "ValidateUserToBeInsert服务错误",
			fields: fields{
				srv: &MockService{
					ValidateUserToBeInsertFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
						return nil, nil, nil, fmt.Errorf("数据库连接失败")
					},
				},
			},
			args: args{
				ctx: createMockContextWithBodyAndQuery("POST", "/api/user/register", `{
					"Email": "test@example.com",
					"UserToken": "password123",
					"OfficialName": "测试用户"
				}`, url.Values{}, ""),
			},
			wantErr: true,
		},
		{
			name: "用户信息不合法",
			fields: fields{
				srv: &MockService{
					ValidateUserToBeInsertFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
						return []User{}, []User{
							{
								TUser: cmn.TUser{
									Account:      "invalid_user",
									OfficialName: null.NewString("无效用户", true),
									Email:        null.NewString("test@example.com", true),
								},
								ErrorMsg: []null.String{
									null.NewString("邮箱已存在", true),
								},
							},
						}, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBodyAndQuery("POST", "/api/user/register", `{
					"Email": "test@example.com",
					"UserToken": "password123",
					"OfficialName": "测试用户"
				}`, url.Values{}, ""),
			},
			wantErr: false,
		},
		{
			name: "json.Marshal强制错误 - 无效用户序列化",
			fields: fields{
				srv: &MockService{
					ValidateUserToBeInsertFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
						return []User{}, []User{
							{
								TUser: cmn.TUser{
									Account:      "invalid_user",
									OfficialName: null.NewString("无效用户", true),
									Email:        null.NewString("test@example.com", true),
								},
								ErrorMsg: []null.String{
									null.NewString("邮箱已存在", true),
								},
							},
						}, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBodyAndQuery("POST", "/api/user/register", `{
					"Email": "test@example.com",
					"UserToken": "password123",
					"OfficialName": "测试用户"
				}`, url.Values{}, "json.Marshal"),
			},
			wantErr: true,
		},
		{
			name: "用户已存在",
			fields: fields{
				srv: &MockService{
					ValidateUserToBeInsertFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
						return []User{}, []User{}, []User{
							{
								TUser: cmn.TUser{
									Account:      "existing_user",
									OfficialName: null.NewString("已存在用户", true),
									Email:        null.NewString("test@example.com", true),
								},
							},
						}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBodyAndQuery("POST", "/api/user/register", `{
					"Email": "test@example.com",
					"UserToken": "password123",
					"OfficialName": "测试用户"
				}`, url.Values{}, ""),
			},
			wantErr: false,
		},
		{
			name: "缺少验证码参数",
			fields: fields{
				srv: &MockService{
					ValidateUserToBeInsertFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
						return users, []User{}, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBodyAndQuery("POST", "/api/user/register", `{
					"Email": "test@example.com",
					"UserToken": "password123",
					"OfficialName": "测试用户"
				}`, url.Values{}, ""),
			},
			wantErr: true,
		},
		{
			name: "rdb.Get强制错误",
			fields: fields{
				srv: &MockService{
					ValidateUserToBeInsertFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
						return users, []User{}, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBodyAndQuery("POST", "/api/user/register", `{
					"Email": "test@example.com",
					"UserToken": "password123",
					"OfficialName": "测试用户"
				}`, url.Values{"verification-code": []string{"123456"}}, "rdb.Get"),
			},
			wantErr: true,
		},
		{
			name: "验证码已过期",
			fields: fields{
				srv: &MockService{
					ValidateUserToBeInsertFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
						return users, []User{}, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBodyAndQuery("POST", "/api/user/register", `{
					"Email": "expired@example.com",
					"UserToken": "password123",
					"OfficialName": "测试用户"
				}`, url.Values{"verification-code": []string{"123456"}}, ""),
			},
			wantErr: true,
		},
		{
			name: "验证码错误",
			fields: fields{
				srv: &MockService{
					ValidateUserToBeInsertFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
						return users, []User{}, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBodyAndQuery("POST", "/api/user/register", `{
					"Email": "wrong@example.com",
					"UserToken": "password123",
					"OfficialName": "测试用户"
				}`, url.Values{"verification-code": []string{"wrong-code"}}, ""),
			},
			wantErr: true,
			setupRedis: func() {
				rdb := cmn.GetRedisConn()
				if rdb != nil {
					rdb.Set(context.Background(), "verify:email:wrong@example.com", "123456", time.Minute*5)
				}
			},
			cleanupRedis: func() {
				rdb := cmn.GetRedisConn()
				if rdb != nil {
					rdb.Del(context.Background(), "verify:email:wrong@example.com")
				}
			},
		},
		{
			name: "InsertUsersWithAccount服务错误",
			fields: fields{
				srv: &MockService{
					ValidateUserToBeInsertFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
						return users, []User{}, []User{}, nil
					},
					err: fmt.Errorf("插入用户失败"),
				},
			},
			args: args{
				ctx: createMockContextWithBodyAndQuery("POST", "/api/user/register", `{
					"Email": "insert-error@example.com",
					"UserToken": "password123",
					"OfficialName": "测试用户"
				}`, url.Values{"verification-code": []string{"123456"}}, ""),
			},
			wantErr: true,
			setupRedis: func() {
				rdb := cmn.GetRedisConn()
				if rdb != nil {
					rdb.Set(context.Background(), "verify:email:insert-error@example.com", "123456", time.Minute*5)
				}
			},
			cleanupRedis: func() {
				rdb := cmn.GetRedisConn()
				if rdb != nil {
					rdb.Del(context.Background(), "verify:email:insert-error@example.com")
				}
			},
		},
		{
			name: "json.Marshal强制错误 - 插入用户序列化",
			fields: fields{
				srv: &MockService{
					ValidateUserToBeInsertFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
						return users, []User{}, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBodyAndQuery("POST", "/api/user/register", `{
					"Email": "marshal-error@example.com",
					"UserToken": "password123",
					"OfficialName": "测试用户"
				}`, url.Values{"verification-code": []string{"123456"}}, "json.Marshal"),
			},
			wantErr: true,
			setupRedis: func() {
				rdb := cmn.GetRedisConn()
				if rdb != nil {
					rdb.Set(context.Background(), "verify:email:marshal-error@example.com", "123456", time.Minute*5)
				}
			},
			cleanupRedis: func() {
				rdb := cmn.GetRedisConn()
				if rdb != nil {
					rdb.Del(context.Background(), "verify:email:marshal-error@example.com")
				}
			},
		},
		{
			name: "json.UnmarshalUser强制错误",
			fields: fields{
				srv: &MockService{
					ValidateUserToBeInsertFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
						return users, []User{}, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBodyAndQuery("POST", "/api/user/register", `{
					"Email": "marshal-error@example.com",
					"UserToken": "password123",
					"OfficialName": "测试用户"
				}`, url.Values{"verification-code": []string{"123456"}}, "json.UnmarshalUser"),
			},
			wantErr: true,
			setupRedis: func() {
				rdb := cmn.GetRedisConn()
				if rdb != nil {
					rdb.Set(context.Background(), "verify:email:marshal-error@example.com", "123456", time.Minute*5)
				}
			},
			cleanupRedis: func() {
				rdb := cmn.GetRedisConn()
				if rdb != nil {
					rdb.Del(context.Background(), "verify:email:marshal-error@example.com")
				}
			},
		},
		{
			name: "复杂用户数据注册",
			fields: fields{
				srv: &MockService{
					ValidateUserToBeInsertFunc: func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
						return users, []User{}, []User{}, nil
					},
				},
			},
			args: args{
				ctx: createMockContextWithBodyAndQuery("POST", "/api/user/register", `{
					"Email": "complex@example.com",
					"UserToken": "password123",
					"OfficialName": "复杂用户",
					"Gender": "M",
					"MobilePhone": "13800138000",
					"IDCardNo": "123456789012345678",
					"IDCardType": "身份证",
					"Nickname": "复杂昵称"
				}`, url.Values{"verification-code": []string{"complex123"}}, ""),
			},
			wantErr: false,
			setupRedis: func() {
				rdb := cmn.GetRedisConn()
				if rdb != nil {
					rdb.Set(context.Background(), "verify:email:complex@example.com", "complex123", time.Minute*5)
				}
			},
			cleanupRedis: func() {
				rdb := cmn.GetRedisConn()
				if rdb != nil {
					rdb.Del(context.Background(), "verify:email:complex@example.com")
				}
			},
		},
	}

	initTestData()
	defer clearTestData()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置Redis数据
			if tt.setupRedis != nil {
				tt.setupRedis()
			}

			// 清理Redis数据
			if tt.cleanupRedis != nil {
				defer tt.cleanupRedis()
			}

			h := &handler{
				srv: tt.fields.srv,
			}
			h.HandleRegisterByEmail(tt.args.ctx)

			// 获取ServiceCtx以检查结果
			q := cmn.GetCtxValue(tt.args.ctx)
			if tt.wantErr {
				if q.Err == nil {
					t.Errorf("HandleRegisterByEmail() 期望有错误，但没有错误")
				}
			} else {
				if q.Err != nil {
					t.Errorf("HandleRegisterByEmail() 不期望有错误，但出现错误: %v", q.Err)
				}
				// 检查成功响应
				if q.Msg.Status == 0 {
					if q.Msg.Msg != "注册成功" {
						t.Errorf("HandleRegisterByEmail() 期望消息为 '注册成功'，实际为 '%s'", q.Msg.Msg)
					}
					if len(q.Msg.Data) == 0 {
						t.Errorf("HandleRegisterByEmail() 期望返回用户数据，但数据为空")
					}
				} else if q.Msg.Status == -1 {
					// 状态码-1表示注册失败（用户信息不合法或用户已存在）
					if q.Msg.Msg != "注册失败，用户信息不合法" && q.Msg.Msg != "注册失败，用户已存在" {
						t.Errorf("HandleRegisterByEmail() 期望消息为注册失败相关信息，实际为 '%s'", q.Msg.Msg)
					}
				} else {
					t.Errorf("HandleRegisterByEmail() 期望状态码为 0 或 -1，实际为 %d", q.Msg.Status)
				}
			}
		})
	}
}
