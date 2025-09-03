package user_mgt

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/sessions"
	"github.com/pkg/errors"
	w2wSrv "w2w.io/service"

	"w2w.io/cmn"
	"w2w.io/null"
)

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
		Users       []map[string]interface{} `json:"users"`
		UserDomains []struct {
			Account string   `json:"Account"`
			Domains []string `json:"Domains"`
		} `json:"user_domains"`
	}

	err = json.Unmarshal(data, &testData)
	if err != nil {
		e := fmt.Sprintf("Failed to unmarshal test data from %s: %v", testDataFile, err)
		z.Fatal(e)
	}

	// 转换并插入测试数据到数据库
	for _, userData := range testData.Users {
		user := convertMapToTUser(userData)
		err = createTUser(cmn.GetDbConn(), user)
		if err != nil {
			e := fmt.Sprintf("Failed to create user %v: %v", user.ID.Int64, err)
			z.Warn(e)
		}
	}

	// 处理用户域关系数据
	pgxConn := cmn.GetPgxConn()
	for _, userDomain := range testData.UserDomains {
		// 根据Account查询用户ID
		var userID int64
		err = pgxConn.QueryRow(context.Background(), "SELECT id FROM t_user WHERE account = $1", userDomain.Account).Scan(&userID)
		if err != nil {
			e := fmt.Sprintf("Failed to find user with account %s: %v", userDomain.Account, err)
			z.Warn(e)
			continue
		}

		// 为每个域创建用户域关系
		for _, domainStr := range userDomain.Domains {
			// 根据Domain字符串查询域ID
			var domainID int64
			err = pgxConn.QueryRow(context.Background(), "SELECT id FROM t_domain WHERE domain = $1", domainStr).Scan(&domainID)
			if err != nil {
				e := fmt.Sprintf("Failed to find domain with domain string %s: %v", domainStr, err)
				z.Warn(e)
				continue
			}

			// 创建用户域关系记录
			userDomainRecord := cmn.TUserDomain{
				SysUser: null.IntFrom(userID),
				Domain:  null.IntFrom(domainID),
			}

			err = userDomainRecord.Create(cmn.GetDbConn())
			if err != nil {
				e := fmt.Sprintf("Failed to create user domain relation for user %d and domain %d: %v", userID, domainID, err)
				z.Warn(e)
			}
		}
	}

	// 运行测试
	code := m.Run()

	// 清理测试数据
	clearSqlTUserDomain := "DELETE FROM t_user_domain"
	_, err = pgxConn.Exec(context.Background(), clearSqlTUserDomain)
	if err != nil {
		e := fmt.Sprintf("Failed to clear user domain data: %v", err)
		z.Warn(e)
	}
	clearSqlTUser := "DELETE FROM t_user WHERE remark = 'test'"
	_, err = pgxConn.Exec(context.Background(), clearSqlTUser)
	if err != nil {
		e := fmt.Sprintf("Failed to clear test data: %v", err)
		z.Warn(e)
	}

	os.Exit(code)
}

// Create inserts the TUser to the database.
func createTUser(db cmn.Queryer, r cmn.TUser) error {
	err := db.QueryRow(
		`INSERT INTO t_user (id, external_id_type, external_id, category, type, language, country, province, city, addr, fuse_name, official_name, id_card_type, id_card_no, mobile_phone, email, account, gender, birthday, nickname, avatar, avatar_type, dev_id, dev_user_id, dev_account, cert, user_token, role, grp, ip, port, auth_failed_count, lock_duration, visit_count, attack_count, lock_reason, logon_time, begin_lock_time, creator, create_time, updated_by, update_time, domain_id, dynamic_attr, addi, remark, status) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35, $36, $37, $38, $39, $40, $41, $42, $43, $44, $45, $46, $47) RETURNING id`,
		&r.ID, &r.ExternalIDType, &r.ExternalID, &r.Category, &r.Type, &r.Language, &r.Country, &r.Province, &r.City, &r.Addr, &r.FuseName, &r.OfficialName, &r.IDCardType, &r.IDCardNo, &r.MobilePhone, &r.Email, &r.Account, &r.Gender, &r.Birthday, &r.Nickname, &r.Avatar, &r.AvatarType, &r.DevID, &r.DevUserID, &r.DevAccount, &r.Cert, &r.UserToken, &r.Role, &r.Grp, &r.IP, &r.Port, &r.AuthFailedCount, &r.LockDuration, &r.VisitCount, &r.AttackCount, &r.LockReason, &r.LogonTime, &r.BeginLockTime, &r.Creator, &r.CreateTime, &r.UpdatedBy, &r.UpdateTime, &r.DomainID, &r.DynamicAttr, &r.Addi, &r.Remark, &r.Status).Scan(&r.ID)
	if err != nil {
		return errors.Wrap(err, "failed to insert t_user")
	}
	return nil
}

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
			ID:   null.NewInt(54242, true), // 请求用户ID
			Role: null.NewInt(20000, true), // 超级管理员角色
		},
	}

	ctx := context.WithValue(context.Background(), cmn.QNearKey, serviceCtx)

	// 使用QNearKey将ServiceCtx设置到context中
	return context.WithValue(ctx, "force-error", forceError)
}

// createMockContextWithoutUser 创建不包含用户信息的mock上下文（用于测试未登录状态）
func createMockContextWithoutUser(method, path string, queryParams string, forceError string) context.Context {
	// 创建mock HTTP请求
	req := httptest.NewRequest(method, path, nil)
	if queryParams != "" {
		req.URL.RawQuery = queryParams
	}

	// 创建mock HTTP响应
	w := httptest.NewRecorder()

	// 创建ServiceCtx，但不设置SysUser字段
	serviceCtx := &cmn.ServiceCtx{
		R: req,
		W: w,
		Msg: &cmn.ReplyProto{
			API:    path,
			Method: method,
		},
		BeginTime: time.Now(),
		Tag:       make(map[string]interface{}),
		// SysUser 字段为nil，模拟未登录状态
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
			ID:   null.NewInt(54242, true), // 请求用户ID
			Role: null.NewInt(20000, true), // 超级管理员角色
		},
	}

	ctx := context.WithValue(context.Background(), cmn.QNearKey, serviceCtx)

	// 使用QNearKey将ServiceCtx设置到context中
	return context.WithValue(ctx, "force-error", forceError)
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

// createMockContextWithBodyAndQuery 创建带有请求体和查询参数的mock上下文，专门用于测试邮箱注册功能
func createMockContextWithBodyAndQuery(method, path string, data string, queryParams url.Values, forceError string) context.Context {
	var req *http.Request

	if data != "" {
		// 检查data是否为有效的JSON
		var testJSON interface{}
		if json.Unmarshal([]byte(data), &testJSON) != nil {
			// 如果不是有效的JSON，直接使用原始字符串作为请求体（用于测试无效JSON的情况）
			req = httptest.NewRequest(method, path, strings.NewReader(data))
		} else {
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
		}
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	// 设置查询参数
	if queryParams != nil {
		req.URL.RawQuery = queryParams.Encode()
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

// createMockContextWithCookies 创建带有cookies的mock context，专门用于测试退出登录功能
// sessionValue 参数用于设置 qNearSessions cookie 的值
func createMockContextWithCookies(method, path string, queryParams url.Values, sessionValue string, forceError string) context.Context {
	// 创建mock HTTP请求
	req := httptest.NewRequest(method, path, nil)
	req.URL.RawQuery = queryParams.Encode()

	// 创建mock HTTP响应
	w := httptest.NewRecorder()

	// 创建 gorilla/sessions store 和 session 实例
	store := sessions.NewCookieStore([]byte("test-secret-key"))
	session, err := store.Get(req, "qNearSessions")
	if err != nil {
		// 如果获取失败，创建一个新的 session
		session = sessions.NewSession(store, "qNearSessions")
		session.IsNew = true
	}

	// 如果提供了 sessionValue，设置到 session 中
	if sessionValue != "" {
		// 模拟一些常见的 session 值
		session.Values["ID"] = int64(54242)
		session.Values["Account"] = "test-user"
		session.Values["Authenticated"] = true
		session.Values["sessionData"] = sessionValue
		session.IsNew = false
	}

	// 创建ServiceCtx
	serviceCtx := &cmn.ServiceCtx{
		R:       req,
		W:       w,
		Session: session, // 设置真正的 gorilla/sessions 实例
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
