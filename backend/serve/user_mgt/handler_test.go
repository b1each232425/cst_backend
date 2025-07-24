package user_mgt

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
	"w2w.io/cmn"
	"w2w.io/null"
	w2wSrv "w2w.io/service"
)

// createMockContext 创建符合GetCtxValue要求的mock context
func createMockContext(method, path string, queryParams url.Values, forceError string) context.Context {
	// 创建mock HTTP请求
	req := httptest.NewRequest(method, path, nil)
	req.URL.RawQuery = queryParams.Encode()

	// 创建mock HTTP响应
	w := httptest.NewRecorder()

	// 创建ServiceCtx
	serviceCtx := &cmn.ServiceCtx{
		R: req,
		W: w,
		Msg: &cmn.ReplyProto{
			API:    path,
			Method: method,
		},
		BeginTime: time.Now(),
		Tag:       make(map[string]interface{}),
		SysUser: &cmn.TUser{
			ID: null.NewInt(54242, true), // 请求用户ID
		},
	}

	ctx := context.WithValue(context.Background(), cmn.QNearKey, serviceCtx)

	// 使用QNearKey将ServiceCtx设置到context中
	return context.WithValue(ctx, "force-error", forceError)
}

// createMockContextWithBody 创建带有请求体数据的mock上下文
// 参数data应该是有效的JSON字符串，将作为ReqProto的Data字段
func createMockContextWithBody(method, path string, data string, forceError string) context.Context {
	var req *http.Request

	if data != "" {
		// 创建ReqProto结构体，Data字段使用json.RawMessage类型
		body := &cmn.ReqProto{
			Data: json.RawMessage(data),
		}

		// 将请求体转换为JSON字符串
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			e := fmt.Sprintf("Failed to marshal request data: %v", err)
			z.Fatal(e)
		}

		// 创建mock HTTP请求
		req = httptest.NewRequest(method, path, strings.NewReader(string(bodyBytes)))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	req.Header.Set("Content-Type", "application/json")

	// 创建mock HTTP响应
	w := httptest.NewRecorder()

	// 创建ServiceCtx
	serviceCtx := &cmn.ServiceCtx{
		R: req,
		W: w,
		Msg: &cmn.ReplyProto{
			API:    path,
			Method: method,
		},
		BeginTime: time.Now(),
		Tag:       make(map[string]interface{}),
		SysUser: &cmn.TUser{
			ID: null.NewInt(54242, true), // 请求用户ID
		},
	}

	ctx := context.WithValue(context.Background(), cmn.QNearKey, serviceCtx)

	// 使用QNearKey将ServiceCtx设置到context中
	return context.WithValue(ctx, "force-error", forceError)
}

// 确保MockRepo实现了Repo接口
var _ Service = (*MockService)(nil)

// TestMain 在测试开始前插入测试数据
func TestMain(m *testing.M) {
	cmn.Configure()
	go w2wSrv.WebServe(nil, nil)

	// 读取测试数据
	testDataFile := "test-data.json"
	data, err := os.ReadFile(testDataFile)
	if err != nil {
		e := fmt.Sprintf("Failed to read test data file %s: %v", testDataFile, err)
		z.Fatal(e)
	}

	var testData struct {
		Users []map[string]interface{} `json:"users"`
	}

	err = json.Unmarshal(data, &testData)
	if err != nil {
		e := fmt.Sprintf("Failed to unmarshal test data from %s: %v", testDataFile, err)
		z.Fatal(e)
	}

	// 转换并插入测试数据到数据库
	for _, userData := range testData.Users {
		user := convertMapToTUser(userData)
		err = user.Create(cmn.GetDbConn())
		if err != nil {
			e := fmt.Sprintf("Failed to create user %v: %v", user.ID.Int64, err)
			z.Warn(e)
		}
	}

	// 运行测试
	code := m.Run()

	// 清理测试数据
	clearSql := "DELETE FROM t_user WHERE remark = 'test'"
	pgxConn := cmn.GetPgxConn()
	_, err = pgxConn.Exec(context.Background(), clearSql)
	if err != nil {
		e := fmt.Sprintf("Failed to clear test data: %v", err)
		z.Warn(e)
	}

	os.Exit(code)
}

// convertMapToTUser 将map数据转换为TUser结构体
func convertMapToTUser(data map[string]interface{}) cmn.TUser {
	user := cmn.TUser{}

	// 处理基本字段，添加nil检查和类型安全转换
	if v, ok := data["ID"]; ok && v != nil {
		if id, ok := v.(float64); ok {
			user.ID = null.NewInt(int64(id), true)
		}
	}
	if v, ok := data["Account"]; ok && v != nil {
		if account, ok := v.(string); ok {
			user.Account = account
		}
	}
	if v, ok := data["ExternalIDType"]; ok && v != nil {
		if externalIDType, ok := v.(string); ok {
			user.ExternalIDType = null.NewString(externalIDType, true)
		}
	}
	if v, ok := data["ExternalID"]; ok && v != nil {
		if externalID, ok := v.(string); ok {
			user.ExternalID = null.NewString(externalID, true)
		}
	}
	if v, ok := data["Category"]; ok && v != nil {
		if category, ok := v.(string); ok {
			user.Category = category
		}
	}
	if v, ok := data["Type"]; ok && v != nil {
		if userType, ok := v.(string); ok {
			user.Type = null.NewString(userType, true)
		}
	}
	if v, ok := data["Language"]; ok && v != nil {
		if language, ok := v.(string); ok {
			user.Language = null.NewString(language, true)
		}
	}
	if v, ok := data["Country"]; ok && v != nil {
		if country, ok := v.(string); ok {
			user.Country = null.NewString(country, true)
		}
	}
	if v, ok := data["Province"]; ok && v != nil {
		if province, ok := v.(string); ok {
			user.Province = null.NewString(province, true)
		}
	}
	if v, ok := data["City"]; ok && v != nil {
		if city, ok := v.(string); ok {
			user.City = null.NewString(city, true)
		}
	}
	if v, ok := data["Addr"]; ok && v != nil {
		if addr, ok := v.(string); ok {
			user.Addr = null.NewString(addr, true)
		}
	}
	if v, ok := data["FuseName"]; ok && v != nil {
		if fuseName, ok := v.(string); ok {
			user.FuseName = null.NewString(fuseName, true)
		}
	}
	if v, ok := data["OfficialName"]; ok && v != nil {
		if officialName, ok := v.(string); ok {
			user.OfficialName = null.NewString(officialName, true)
		}
	}
	if v, ok := data["IDCardType"]; ok && v != nil {
		if idCardType, ok := v.(string); ok {
			user.IDCardType = null.NewString(idCardType, true)
		}
	}
	if v, ok := data["IDCardNo"]; ok && v != nil {
		if idCardNo, ok := v.(string); ok {
			user.IDCardNo = null.NewString(idCardNo, true)
		}
	}
	if v, ok := data["MobilePhone"]; ok && v != nil {
		if mobilePhone, ok := v.(string); ok {
			user.MobilePhone = null.NewString(mobilePhone, true)
		}
	}
	if v, ok := data["Email"]; ok && v != nil {
		if email, ok := v.(string); ok {
			user.Email = null.NewString(email, true)
		}
	}
	if v, ok := data["Gender"]; ok && v != nil {
		if gender, ok := v.(string); ok {
			user.Gender = null.NewString(gender, true)
		}
	}
	if v, ok := data["Birthday"]; ok && v != nil {
		if birthday, ok := v.(float64); ok {
			user.Birthday = null.NewInt(int64(birthday), true)
		}
	}
	if v, ok := data["Nickname"]; ok && v != nil {
		if nickname, ok := v.(string); ok {
			user.Nickname = null.NewString(nickname, true)
		}
	}
	if v, ok := data["AvatarType"]; ok && v != nil {
		if avatarType, ok := v.(string); ok {
			user.AvatarType = null.NewString(avatarType, true)
		}
	}
	if v, ok := data["DevID"]; ok && v != nil {
		if devID, ok := v.(string); ok {
			user.DevID = null.NewString(devID, true)
		}
	}
	if v, ok := data["DevUserID"]; ok && v != nil {
		if devUserID, ok := v.(string); ok {
			user.DevUserID = null.NewString(devUserID, true)
		}
	}
	if v, ok := data["DevAccount"]; ok && v != nil {
		if devAccount, ok := v.(string); ok {
			user.DevAccount = null.NewString(devAccount, true)
		}
	}
	if v, ok := data["Role"]; ok && v != nil {
		if role, ok := v.(float64); ok {
			user.Role = null.NewInt(int64(role), true)
		}
	}
	if v, ok := data["Grp"]; ok && v != nil {
		if grp, ok := v.(float64); ok {
			user.Grp = null.NewInt(int64(grp), true)
		}
	}
	if v, ok := data["IP"]; ok && v != nil {
		if ip, ok := v.(string); ok {
			user.IP = null.NewString(ip, true)
		}
	}
	if v, ok := data["Port"]; ok && v != nil {
		if port, ok := v.(string); ok {
			user.Port = null.NewString(port, true)
		}
	}
	if v, ok := data["AuthFailedCount"]; ok && v != nil {
		if authFailedCount, ok := v.(float64); ok {
			user.AuthFailedCount = null.NewInt(int64(authFailedCount), true)
		}
	}
	if v, ok := data["LockDuration"]; ok && v != nil {
		if lockDuration, ok := v.(float64); ok {
			user.LockDuration = null.NewInt(int64(lockDuration), true)
		}
	}
	if v, ok := data["VisitCount"]; ok && v != nil {
		if visitCount, ok := v.(float64); ok {
			user.VisitCount = null.NewInt(int64(visitCount), true)
		}
	}
	if v, ok := data["AttackCount"]; ok && v != nil {
		if attackCount, ok := v.(float64); ok {
			user.AttackCount = null.NewInt(int64(attackCount), true)
		}
	}
	if v, ok := data["LockReason"]; ok && v != nil {
		if lockReason, ok := v.(string); ok {
			user.LockReason = null.NewString(lockReason, true)
		}
	}
	if v, ok := data["LogonTime"]; ok && v != nil {
		if logonTime, ok := v.(float64); ok {
			user.LogonTime = null.NewInt(int64(logonTime), true)
		}
	}
	if v, ok := data["BeginLockTime"]; ok && v != nil {
		if beginLockTime, ok := v.(float64); ok {
			user.BeginLockTime = null.NewInt(int64(beginLockTime), true)
		}
	}
	if v, ok := data["Creator"]; ok && v != nil {
		if creator, ok := v.(float64); ok {
			user.Creator = null.NewInt(int64(creator), true)
		}
	}
	if v, ok := data["CreateTime"]; ok && v != nil {
		if createTime, ok := v.(float64); ok {
			user.CreateTime = null.NewInt(int64(createTime), true)
		}
	}
	if v, ok := data["UpdatedBy"]; ok && v != nil {
		if updatedBy, ok := v.(float64); ok {
			user.UpdatedBy = null.NewInt(int64(updatedBy), true)
		}
	}
	if v, ok := data["UpdateTime"]; ok && v != nil {
		if updateTime, ok := v.(float64); ok {
			user.UpdateTime = null.NewInt(int64(updateTime), true)
		}
	}
	if v, ok := data["DomainID"]; ok && v != nil {
		if domainID, ok := v.(float64); ok {
			user.DomainID = null.NewInt(int64(domainID), true)
		}
	}
	if v, ok := data["Remark"]; ok && v != nil {
		if remark, ok := v.(string); ok {
			user.Remark = null.NewString(remark, true)
		}
	}
	if v, ok := data["Status"]; ok && v != nil {
		if status, ok := v.(string); ok {
			user.Status = null.NewString(status, true)
		}
	}

	// 处理Addi字段（JSON对象）
	if addi, ok := data["Addi"]; ok && addi != nil {
		addiBytes, err := json.Marshal(addi)
		if err == nil {
			user.Addi = addiBytes
		}
	}

	return user
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
					users: []cmn.TUser{
						{
							ID:           null.NewInt(1, true),
							Account:      "test_user_001",
							OfficialName: null.NewString("测试用户001", true),
							Gender:       null.NewString("M", true),
							Status:       null.NewString("00", true),
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
					users: []cmn.TUser{
						{
							ID:           null.NewInt(2, true),
							Account:      "admin_user",
							OfficialName: null.NewString("管理员", true),
							Gender:       null.NewString("F", true),
							Status:       null.NewString("00", true),
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
					users:     []cmn.TUser{},
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
					users:     make([]cmn.TUser, 100),
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
					users: []cmn.TUser{
						{
							ID:           null.NewInt(3, true),
							Account:      "test_female",
							OfficialName: null.NewString("女性测试用户", true),
							Gender:       null.NewString("F", true),
							MobilePhone:  null.NewString("13800138000", true),
							Email:        null.NewString("test@example.com", true),
							Status:       null.NewString("00", true),
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
					users: []cmn.TUser{
						{
							ID:           null.NewInt(3, true),
							Account:      "test_female",
							OfficialName: null.NewString("女性测试用户", true),
							Gender:       null.NewString("F", true),
							MobilePhone:  null.NewString("13800138000", true),
							Email:        null.NewString("test@example.com", true),
							Status:       null.NewString("00", true),
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
					ValidateUserFunc: func(ctx context.Context, tx pgx.Tx, users []cmn.TUser) ([]cmn.TUser, []InvalidUser, error) {
						return []cmn.TUser{
							{
								Account: "test_user",
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
					ValidateUserFunc: func(ctx context.Context, tx pgx.Tx, users []cmn.TUser) ([]cmn.TUser, []InvalidUser, error) {
						return []cmn.TUser{
							{
								Account:      "new_user_001",
								OfficialName: null.NewString("新用户001", true),
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
					ValidateUserFunc: func(ctx context.Context, tx pgx.Tx, users []cmn.TUser) ([]cmn.TUser, []InvalidUser, error) {
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
					ValidateUserFunc: func(ctx context.Context, tx pgx.Tx, users []cmn.TUser) ([]cmn.TUser, []InvalidUser, error) {
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
					ValidateUserFunc: func(ctx context.Context, tx pgx.Tx, users []cmn.TUser) ([]cmn.TUser, []InvalidUser, error) {
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
					ValidateUserFunc: func(ctx context.Context, tx pgx.Tx, users []cmn.TUser) ([]cmn.TUser, []InvalidUser, error) {
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
