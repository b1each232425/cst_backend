package user_mgt

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"io"
	"strconv"
	"strings"
	"w2w.io/cmn"
)

type Handler interface {
	HandleUser(ctx context.Context)
	HandleGetNewAccount(ctx context.Context)
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

	_, roleName, err := h.srv.QueryUserCurrentRole(ctx, q.SysUser.ID)
	if err != nil {
		q.Err = fmt.Errorf("failed to query user current role: %w", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
	if (roleName.String != DomainSuperAdmin && roleName.String != DomainAdmin) || !roleName.Valid {
		q.Err = fmt.Errorf("user does not have permission to access this resource")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	switch method {
	case "get": // 获取用户列表
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
		if domain != "" && !IsDomainExist(domain) {
			q.Err = fmt.Errorf("invalid filter domain: %s", query.Get("domain"))
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 构造过滤条件
		filter := QueryUsersFilter{
			Account:    NullableString(query.Get("account")),
			Name:       NullableString(query.Get("officialName")),
			Phone:      NullableString(query.Get("mobilePhone")),
			Email:      NullableString(query.Get("email")),
			Gender:     NullableString(query.Get("gender")),
			Status:     NullableString(query.Get("status")),
			CreateTime: NullableIntFromStr(query.Get("createTime")),
			Domain:     NullableString(domain),
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
				_ = tx.Rollback(ctx)
				z.Error("transaction rolled back due to error: " + err.Error())
			} else {
				err = tx.Commit(ctx)
				if err != nil || forceErr == "tx.Commit" {
					z.Error("failed to commit transaction: " + err.Error())
				}
			}
		}()

		// 验证用户信息
		var validUsers []User
		var invalidUsers []InvalidUser
		validUsers, invalidUsers, err = h.srv.ValidateUserToBeInsert(ctx, tx, users)
		if err != nil {
			q.Err = fmt.Errorf("failed to validate users: %w", err)
			q.RespErr()
			return
		}

		if len(validUsers) > 0 {
			err = h.srv.InsertUsers(ctx, tx, validUsers)
			if err != nil {
				q.Err = fmt.Errorf("failed to insert users: %w", err)
				q.RespErr()
				return
			}
		}

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

		q.Msg.Status = 0
		q.Msg.Msg = "success"
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
