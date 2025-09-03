package user_mgt

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/wneessen/go-mail"
	"w2w.io/cmn"
	"w2w.io/null"
	"w2w.io/serve/auth_mgt"
)

type Handler interface {
	HandleUser(ctx context.Context)
	HandleGetNewAccount(ctx context.Context)
	HandleSelectLoginDomain(ctx context.Context)
	HandleQueryMyInfo(ctx context.Context)
	HandleValidateUserToBeInsert(ctx context.Context)
	HandleLogout(ctx context.Context)
	HandleSendValidationCodeEmail(ctx context.Context)
	HandleRegisterByEmail(ctx context.Context)
}

type handler struct {
	srv Service
}

func NewHandler() Handler {
	return &handler{
		srv: NewService(),
	}
}

// HandleUser 处理用户管理相关请求
func (h *handler) HandleUser(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	forceErr, _ := ctx.Value("force-error").(string) // 用于强制执行错误处理代码
	var err error

	method := strings.ToLower(q.R.Method)

	if q.SysUser == nil || !q.SysUser.ID.Valid || forceErr == "no-login" {
		q.Err = fmt.Errorf("user not logged in or invalid user ID")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	authority, err := auth_mgt.GetUserAuthority(ctx)
	if err != nil || forceErr == "GetUserAuthority" {
		q.Err = fmt.Errorf("failed to get user authority: %w", err)
		q.RespErr()
		return
	}

	switch method {
	case "get": // 获取用户列表
		// 检查用户是否有权限访问该API
		accessible, err := auth_mgt.CheckUserAPIAccessible(ctx, authority, "/api/user", auth_mgt.CAPIAccessActionRead)
		if err != nil || forceErr == "CheckUserAPIAccessible" {
			q.Err = fmt.Errorf("failed to check user API access: %w", err)
			q.RespErr()
			return
		}
		if !accessible || forceErr == "no-access" {
			q.Err = fmt.Errorf("user does not have permission to access this API")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		query := q.R.URL.Query()

		page, err := strconv.ParseInt(query.Get("page"), 10, 64)
		if err != nil || page <= 0 {
			q.Err = fmt.Errorf("invalid page parameter, must be a positive integer")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		pageSize, err := strconv.ParseInt(query.Get("pageSize"), 10, 64)
		if err != nil || pageSize <= 0 {
			q.Err = fmt.Errorf("invalid pageSize parameter, must be a positive integer")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		domain := query.Get("domain")
		if domain != "" {
			domainExist, _, err := IsDomainExist(ctx, nil, domain)
			if err != nil || forceErr == "IsDomainExist" {
				q.Err = fmt.Errorf("failed to check domain existence: %w", err)
				q.RespErr()
				return
			}
			if !domainExist {
				q.Err = fmt.Errorf("filter domain does not exist")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		}

		// 构造过滤条件
		filter := QueryUsersFilter{
			FuzzyCondition: NullableString(query.Get("fuzzyCondition")),
			Gender:         NullableString(query.Get("gender")),
			Status:         NullableString(query.Get("status")),
			CreateTime:     NullableIntFromStr(query.Get("createTime")),
			Domain:         NullableString(domain),
		}

		users, totalRows, err := h.srv.QueryUsers(ctx, nil, page, pageSize, filter)
		if err != nil {
			q.Err = fmt.Errorf("failed to query users: %w", err)
			q.RespErr()
			return
		}

		usersJson, err := json.Marshal(users)
		if err != nil || forceErr == "json.Marshal" {
			q.Err = fmt.Errorf("failed to marshal users: %w", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Msg.Status = 0
		q.Msg.Msg = "success"
		q.Msg.RowCount = totalRows
		q.Msg.Data = usersJson
		q.Resp()
		return

	case "post": // 创建新用户
		// 检查用户是否有权限访问该API
		accessible, err := auth_mgt.CheckUserAPIAccessible(ctx, authority, "/api/user", auth_mgt.CAPIAccessActionCreate)
		if err != nil || forceErr == "CheckUserAPIAccessible" {
			q.Err = fmt.Errorf("failed to check user API access: %w", err)
			q.RespErr()
			return
		}
		if !accessible || forceErr == "no-access" {
			q.Err = fmt.Errorf("user does not have permission to access this API")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var buf []byte
		buf, err = io.ReadAll(q.R.Body)
		if err != nil || forceErr == "io.ReadAll" {
			q.Err = fmt.Errorf("failed to read body: %w", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer func() {
			err = q.R.Body.Close()
			if err != nil || forceErr == "io.Close" {
				e := fmt.Errorf("failed to close request body: %w", err)
				z.Error(e.Error())
				return
			}
		}()

		if len(buf) == 0 {
			q.Err = fmt.Errorf("request body cannot be empty")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var body cmn.ReqProto
		err = json.Unmarshal(buf, &body)
		if err != nil || forceErr == "json.Unmarshal" {
			q.Err = fmt.Errorf("failed to unmarshal request body: %w", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var users []User
		err = json.Unmarshal(body.Data, &users)
		if err != nil {
			q.Err = fmt.Errorf("failed to unmarshal request json data: %w", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if len(users) == 0 {
			q.Err = fmt.Errorf("no users provided in request json data")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 添加通过当前接口创建的用户的默认字段
		for i := range users {
			users[i].Category = "sys^user"
			if q.SysUser != nil && q.SysUser.ID.Valid {
				users[i].Creator = q.SysUser.ID
			}
		}

		// 创建事务
		var tx pgx.Tx
		pgxConn := cmn.GetPgxConn()
		tx, err = pgxConn.Begin(ctx)
		if err != nil || forceErr == "tx.Begin" {
			q.Err = fmt.Errorf("failed to begin transaction: %w", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer func() {
			if err != nil {
				err = tx.Rollback(ctx)
				if err != nil || forceErr == "tx.Rollback" {
					z.Error("transaction rolled back due to error: " + err.Error())
				}
				return
			}
			err = tx.Commit(ctx)
			if err != nil || forceErr == "tx.Commit" {
				z.Error("failed to commit transaction: " + err.Error())
			}
			return
		}()

		// 验证用户信息
		var validUsers []User
		var invalidUsers []User
		validUsers, invalidUsers, _, err = h.srv.ValidateUserToBeInsert(ctx, tx, users)
		if err != nil {
			q.Err = fmt.Errorf("failed to validate users: %w", err)
			q.RespErr()
			return
		}

		// 若存在不合法用户，则直接返回，不执行插入操作
		if len(invalidUsers) != 0 {
			invalidUsersBytes, err := json.Marshal(invalidUsers)
			if err != nil || forceErr == "json.Marshal" {
				q.Err = fmt.Errorf("failed to marshal invalid users: %w", err)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			q.Msg.Status = 405
			q.Msg.Msg = "some users are invalid and cannot be inserted"
			q.Msg.Data = invalidUsersBytes
			q.Resp()
			return
		}

		var insertedUsers []User
		if len(validUsers) > 0 {
			insertedUsers, err = h.srv.InsertUsers(ctx, tx, validUsers)
			if err != nil {
				q.Err = fmt.Errorf("failed to insert users: %w", err)
				q.RespErr()
				return
			}
		}

		insertedUsersJson, err := json.Marshal(insertedUsers)
		if err != nil || forceErr == "json.Marshal" {
			q.Err = fmt.Errorf("failed to marshal valid users: %w", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Msg.Status = 0
		q.Msg.Msg = "success"
		q.Msg.Data = insertedUsersJson
		q.Resp()
		return

	case "put": // 覆盖式更新已有用户
		// 检查用户是否有权限访问该API
		accessible, err := auth_mgt.CheckUserAPIAccessible(ctx, authority, "/api/user", auth_mgt.CAPIAccessActionUpdate)
		if err != nil || forceErr == "CheckUserAPIAccessible" {
			q.Err = fmt.Errorf("failed to check user API access: %w", err)
			q.RespErr()
			return
		}
		if !accessible || forceErr == "no-access" {
			q.Err = fmt.Errorf("user does not have permission to access this API")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var buf []byte
		buf, err = io.ReadAll(q.R.Body)
		if err != nil || forceErr == "io.ReadAll" {
			q.Err = fmt.Errorf("failed to read body: %w", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer func() {
			err = q.R.Body.Close()
			if err != nil || forceErr == "io.Close" {
				e := fmt.Errorf("failed to close request body: %w", err)
				z.Error(e.Error())
				return
			}
		}()

		if len(buf) == 0 {
			q.Err = fmt.Errorf("request body cannot be empty")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var body cmn.ReqProto
		err = json.Unmarshal(buf, &body)
		if err != nil || forceErr == "json.Unmarshal" {
			q.Err = fmt.Errorf("failed to unmarshal request body: %w", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		var users []User
		err = json.Unmarshal(body.Data, &users)
		if err != nil {
			q.Err = fmt.Errorf("failed to unmarshal request json data: %w", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if len(users) == 0 {
			q.Err = fmt.Errorf("no users provided in request json data")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 创建事务
		var tx pgx.Tx
		pgxConn := cmn.GetPgxConn()
		tx, err = pgxConn.Begin(ctx)
		if err != nil || forceErr == "tx.Begin" {
			q.Err = fmt.Errorf("failed to begin transaction: %w", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer func() {
			if err != nil {
				err = tx.Rollback(ctx)
				if err != nil || forceErr == "tx.Rollback" {
					z.Error("transaction rolled back due to error: " + err.Error())
				}
				return
			}
			err = tx.Commit(ctx)
			if err != nil || forceErr == "tx.Commit" {
				z.Error("failed to commit transaction: " + err.Error())
			}
			return
		}()

		// 验证用户信息
		// TODO: 不能用这个验证函数
		var validUsers []User
		var invalidUsers []User
		validUsers, invalidUsers, _, err = h.srv.ValidateUserToBeInsert(ctx, tx, users)
		if err != nil {
			q.Err = fmt.Errorf("failed to validate users: %w", err)
			q.RespErr()
			return
		}

		// 若存在不合法用户，则直接返回，不执行更新操作
		if len(invalidUsers) != 0 {
			invalidUsersBytes, err := json.Marshal(invalidUsers)
			if err != nil || forceErr == "json.Marshal" {
				q.Err = fmt.Errorf("failed to marshal invalid users: %w", err)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			q.Msg.Status = 405
			q.Msg.Msg = "some users are invalid and cannot be updated"
			q.Msg.Data = invalidUsersBytes
			q.Resp()
			return
		}

		var updatedUsers []User
		if len(validUsers) > 0 {
			updatedUsers, err = h.srv.OverwriteUpdateUsers(ctx, tx, validUsers)
			if err != nil {
				q.Err = fmt.Errorf("failed to update users: %w", err)
				q.RespErr()
				return
			}
		}

		insertedUsersJson, err := json.Marshal(updatedUsers)
		if err != nil || forceErr == "json.Marshal" {
			q.Err = fmt.Errorf("failed to marshal valid users: %w", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Msg.Status = 0
		q.Msg.Msg = "success"
		q.Msg.Data = insertedUsersJson
		q.Resp()
		return

	default:
		q.Err = fmt.Errorf("unsupported method: %s", method)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
}

// HandleGetNewAccount 处理获取新账户的请求
func (h *handler) HandleGetNewAccount(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	forceErr, _ := ctx.Value("force-error").(string) // 用于强制执行错误处理代码
	var err error

	method := strings.ToLower(q.R.Method)
	if method != "get" {
		q.Err = fmt.Errorf("unsupported method: %s", method)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	newAccount, err := h.srv.GenerateUniqueAccount(ctx, nil, AccountLength, 20)
	if err != nil {
		q.Err = fmt.Errorf("failed to generate new account: %w", err)
		q.RespErr()
		return
	}

	accountBytes, err := json.Marshal(newAccount)
	if err != nil || forceErr == "json.Marshal" {
		q.Err = fmt.Errorf("failed to marshal new account: %w", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	q.Msg.Status = 0
	q.Msg.Msg = "success"
	q.Msg.Data = accountBytes
	q.Resp()
	return
}

// HandleSelectLoginDomain 处理选择登录角色的请求
func (h *handler) HandleSelectLoginDomain(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	forceErr, _ := ctx.Value("force-error").(string) // 用于强制执行错误处理代码
	var err error

	method := strings.ToLower(q.R.Method)
	if method != "patch" {
		q.Err = fmt.Errorf("unsupported method: %s", method)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	var buf []byte
	buf, err = io.ReadAll(q.R.Body)
	if err != nil || forceErr == "io.ReadAll" {
		q.Err = fmt.Errorf("failed to read body: %w", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	defer func() {
		err = q.R.Body.Close()
		if err != nil || forceErr == "io.Close" {
			e := fmt.Errorf("failed to close request body: %w", err)
			z.Error(e.Error())
			return
		}
	}()

	if len(buf) == 0 {
		q.Err = fmt.Errorf("request body cannot be empty")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	var body cmn.ReqProto
	err = json.Unmarshal(buf, &body)
	if err != nil || forceErr == "json.Unmarshal" {
		q.Err = fmt.Errorf("failed to unmarshal request body: %w", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	var domain string
	if err = json.Unmarshal(body.Data, &domain); err != nil || forceErr == "json.UnmarshalDomain" {
		q.Err = fmt.Errorf("failed to parse domain from data: %w", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	users, _, err := h.srv.QueryUsers(ctx, nil, 1, 1, QueryUsersFilter{
		ID: q.SysUser.ID,
	})
	if err != nil {
		q.Err = fmt.Errorf("failed to query user: %w", err)
		q.RespErr()
		return
	}
	if len(users) == 0 {
		q.Err = fmt.Errorf("user not found")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	user := users[0]

	exist, existDomain, err := IsDomainExist(ctx, nil, domain)
	if err != nil || forceErr == "IsDomainExist" {
		q.Err = fmt.Errorf("failed to check domain existence: %w", err)
		q.RespErr()
		return
	}
	if !exist || forceErr == "domain-not-exist" {
		q.Err = fmt.Errorf("domain does not exist: %s", domain)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	if !Contains(null.StringFrom(domain), user.Domains) {
		q.Err = fmt.Errorf("user does not have permission to access this domain: %s", domain)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	pgxConn := cmn.GetPgxConn()

	const updateUserRole = "UPDATE t_user SET role = $1 WHERE id = $2"
	_, err = pgxConn.Exec(ctx, updateUserRole, existDomain.ID, user.ID)
	if err != nil || forceErr == "UpdateUserRole" {
		q.Err = fmt.Errorf("failed to update user role: %w", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	q.Msg.Status = 0
	q.Msg.Msg = "success"
	q.Resp()
	return
}

// HandleQueryMyInfo 处理查询我的信息请求
func (h *handler) HandleQueryMyInfo(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	forceErr, _ := ctx.Value("force-error").(string) // 用于强制执行错误处理代码
	var err error

	method := strings.ToLower(q.R.Method)
	if method != "get" {
		q.Err = fmt.Errorf("unsupported method: %s", method)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	if q.SysUser == nil || !q.SysUser.ID.Valid {
		q.Err = fmt.Errorf("user not logged in or invalid user ID")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	users, _, err := h.srv.QueryUsers(ctx, nil, 1, 1, QueryUsersFilter{
		ID: q.SysUser.ID,
	})
	if err != nil {
		q.Err = fmt.Errorf("failed to query user: %w", err)
		q.RespErr()
		return
	}
	if len(users) == 0 {
		q.Err = fmt.Errorf("user not found")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	user := users[0]

	userJson, err := json.Marshal(user)
	if err != nil || forceErr == "json.Marshal" {
		q.Err = fmt.Errorf("failed to marshal users: %w", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	q.Msg.Status = 0
	q.Msg.Msg = "success"
	q.Msg.Data = userJson
	q.Resp()
	return
}

// HandleValidateUserToBeInsert 验证用户信息是否可以插入
func (h *handler) HandleValidateUserToBeInsert(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	forceErr, _ := ctx.Value("force-error").(string) // 用于强制执行错误处理代码
	var err error

	method := strings.ToLower(q.R.Method)
	if method != "post" {
		q.Err = fmt.Errorf("unsupported method: %s", method)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	var buf []byte
	buf, err = io.ReadAll(q.R.Body)
	if err != nil || forceErr == "io.ReadAll" {
		q.Err = fmt.Errorf("failed to read body: %w", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	defer func() {
		err = q.R.Body.Close()
		if err != nil || forceErr == "io.Close" {
			e := fmt.Errorf("failed to close request body: %w", err)
			z.Error(e.Error())
			return
		}
	}()

	if len(buf) == 0 {
		q.Err = fmt.Errorf("request body cannot be empty")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	var body cmn.ReqProto
	err = json.Unmarshal(buf, &body)
	if err != nil || forceErr == "json.Unmarshal" {
		q.Err = fmt.Errorf("failed to unmarshal request body: %w", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	var users []User
	err = json.Unmarshal(body.Data, &users)
	if err != nil {
		q.Err = fmt.Errorf("failed to unmarshal request json data: %w", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	if len(users) == 0 {
		q.Err = fmt.Errorf("no users provided in request json data")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	var validUsers = make([]User, len(users))
	var invalidUsers = make([]User, len(users))
	var existUsers = make([]User, len(users))

	// 验证用户信息
	validUsers, invalidUsers, existUsers, err = h.srv.ValidateUserToBeInsert(ctx, nil, users)
	if err != nil {
		q.Err = fmt.Errorf("failed to validate users: %w", err)
		q.RespErr()
		return
	}

	type RespData struct {
		ValidUsers   []User `json:"validUsers"`
		InvalidUsers []User `json:"invalidUsers"`
		ExistUsers   []User `json:"existingUsers"`
	}

	respData := RespData{
		ValidUsers:   validUsers,
		InvalidUsers: invalidUsers,
		ExistUsers:   existUsers,
	}

	respDataJson, err := json.Marshal(respData)
	if err != nil || forceErr == "json.Marshal" {
		q.Err = fmt.Errorf("failed to marshal response data: %w", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	q.Msg.Status = 0
	q.Msg.Msg = "success"
	q.Msg.Data = respDataJson
	q.Resp()
	return
}

// HandleLogout 处理用户退出登录请求
func (h *handler) HandleLogout(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	forceErr, _ := ctx.Value("force-error").(string) // 用于强制执行错误处理代码

	if q.Session != nil {
		q.Session.Options.MaxAge = -1
		for k := range q.Session.Values {
			delete(q.Session.Values, k)
		}
		err := q.Session.Save(q.R, q.W)
		if err != nil || forceErr == "Session.Save" {
			q.Err = fmt.Errorf("failed to save session: %w", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
	}

	// 返回成功响应
	q.Msg.Status = 0
	q.Msg.Msg = "logout success"
	q.Resp()
}

// HandleSendValidationCodeEmail 处理发送验证代码邮件请求
func (h *handler) HandleSendValidationCodeEmail(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	forceErr, _ := ctx.Value("force-error").(string) // 用于强制执行错误处理代码
	var err error

	method := strings.ToLower(q.R.Method)
	if method != "get" {
		q.Err = fmt.Errorf("不支持的请求方法: %s", method)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	query := q.R.URL.Query()

	recipient := query.Get("recipient")
	if recipient == "" {
		q.Err = fmt.Errorf("缺少电子邮件地址参数")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil || forceErr == "rand.Int" {
		q.Err = fmt.Errorf("生成验证码失败，请稍后再试: %w", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	code := fmt.Sprintf("%06d", n.Int64()) // 6位验证码

	subject := "3min学习平台 - 验证您的电子邮件地址"
	body := fmt.Sprintf(`尊敬的学员，<br><br>
	感谢您选择加入「3min学习平台」！为了保障您的账户安全，我们需要验证您的电子邮件地址。<br><br>
	您的专属验证码是：<strong>%s</strong><br><br>
	该验证码有效期为<strong>%s</strong>分钟，请在验证码有效期内将此验证码输入到平台的验证页面，以完成认证。<br><br>
	如果您没有请求此验证码，请忽略此邮件，您的账户信息不会受到影响。<br><br>
	如需帮助或有任何疑问，您可以直接回复此邮件与我们联系，我们的客服团队将竭诚为您服务。<br><br>
	祝您在「3min学习平台」的学习之旅中收获知识与成长！<br><br>
	——3min学习平台团队`, code, "15")

	err = h.srv.SendEmail(ctx, recipient, subject, body, mail.TypeTextHTML)
	if err != nil {
		q.Err = fmt.Errorf("发送验证码邮件失败，请稍后再试: %w", err)
		q.RespErr()
		return
	}

	// 将验证码保存到redis
	rdb := cmn.GetRedisConn()
	key := "verify:email:" + strings.ToLower(recipient)
	err = rdb.Set(ctx, key, code, 15*time.Minute).Err()
	if err != nil || forceErr == "rdb.Set" {
		q.Err = fmt.Errorf("发送验证码邮件失败，请稍后再试: %w", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	// 返回成功响应
	q.Msg.Status = 0
	q.Msg.Msg = "success"
	q.Resp()
}

// HandleRegisterByEmail 处理通过邮箱注册新用户请求
func (h *handler) HandleRegisterByEmail(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	forceErr, _ := ctx.Value("force-error").(string) // 用于强制执行错误处理代码
	var err error

	method := strings.ToLower(q.R.Method)
	if method != "post" {
		q.Err = fmt.Errorf("unsupported method: %s", method)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	var buf []byte
	buf, err = io.ReadAll(q.R.Body)
	if err != nil || forceErr == "io.ReadAll" {
		q.Err = fmt.Errorf("failed to read body: %w", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	defer func() {
		err = q.R.Body.Close()
		if err != nil || forceErr == "io.Close" {
			e := fmt.Errorf("failed to close request body: %w", err)
			z.Error(e.Error())
			return
		}
	}()

	if len(buf) == 0 {
		q.Err = fmt.Errorf("request body cannot be empty")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	var body cmn.ReqProto
	err = json.Unmarshal(buf, &body)
	if err != nil || forceErr == "json.Unmarshal" {
		q.Err = fmt.Errorf("failed to unmarshal request body: %w", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	var user User
	err = json.Unmarshal(body.Data, &user)
	if err != nil || forceErr == "json.UnmarshalUser" {
		q.Err = fmt.Errorf("failed to unmarshal request json data: %w", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	if !user.Email.Valid {
		q.Err = fmt.Errorf("注册失败，缺少邮箱地址")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	if !user.UserToken.Valid {
		q.Err = fmt.Errorf("注册失败，缺少密码")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	user.Category = "sys^user"
	user.Status = null.NewString("00", true)
	user.Domains = []null.String{null.StringFrom(cmn.RoleName(cmn.CDomainAssessStudent))}

	pgxConn := cmn.GetPgxConn()

	tx, err := pgxConn.Begin(ctx)
	if err != nil || forceErr == "tx.Begin" {
		q.Err = fmt.Errorf("failed to begin transaction: %w", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	defer func() {
		if err != nil {
			err = tx.Rollback(ctx)
			if err != nil || forceErr == "tx.Rollback" {
				z.Error("transaction rolled back due to error: " + err.Error())
			}
			return
		}
		err = tx.Commit(ctx)
		if err != nil || forceErr == "tx.Commit" {
			z.Error("failed to commit transaction: " + err.Error())
		}
		return
	}()

	validUser, invalidUser, existUser, err := h.srv.ValidateUserToBeInsert(ctx, tx, []User{user})
	if err != nil {
		q.Err = fmt.Errorf("failed to validate user: %w", err)
		q.RespErr()
		return
	}

	// 若存在不合法用户，则直接返回，不执行插入操作
	if len(invalidUser) != 0 {
		invalidUserBytes, err := json.Marshal(invalidUser[0])
		if err != nil || forceErr == "json.Marshal" {
			q.Err = fmt.Errorf("failed to marshal invalid user: %w", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		q.Msg.Status = -1
		q.Msg.Msg = "注册失败，用户信息不合法"
		q.Msg.Data = invalidUserBytes
		q.Resp()
		return
	}

	// 若用户已存在，则直接返回，不执行插入操作
	if len(existUser) != 0 {
		q.Msg.Status = -1
		q.Msg.Msg = "注册失败，用户已存在"
		q.Resp()
		return
	}

	// 从查询参数验证验证码
	query := q.R.URL.Query()
	code := query.Get("verification-code")
	if code == "" {
		q.Err = fmt.Errorf("注册失败，缺少验证码参数")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	// 从redis获取验证码并比对
	rdb := cmn.GetRedisConn()
	key := "verify:email:" + strings.ToLower(validUser[0].Email.String)
	storedCode, err := rdb.Get(ctx, key).Result()
	if err != nil || forceErr == "rdb.Get" {
		q.Err = fmt.Errorf("注册失败，验证码已过期，请重新获取")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	if storedCode != code {
		q.Err = fmt.Errorf("注册失败，验证码错误")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	insertedUser, err := h.srv.InsertUsersWithAccount(ctx, tx, validUser)
	if err != nil {
		q.Err = fmt.Errorf("failed to insert user: %w", err)
		q.RespErr()
		return
	}

	insertedUserBytes, err := json.Marshal(insertedUser)
	if err != nil || forceErr == "json.Marshal" {
		q.Err = fmt.Errorf("failed to marshal inserted user: %w", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	q.Msg.Status = 0
	q.Msg.Msg = "注册成功"
	q.Msg.Data = insertedUserBytes
	q.Resp()
	return
}
