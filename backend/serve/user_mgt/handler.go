package user_mgt

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"w2w.io/cmn"
)

type Handler interface {
	HandleUser(ctx context.Context)
}

type handler struct {
	repo Repo
}

func NewHandler() Handler {
	return &handler{
		repo: NewRepo(),
	}
}

// HandleUser 处理用户用户管理相关请求
func (h *handler) HandleUser(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	forceErr, _ := ctx.Value("force-error").(string)

	method := strings.ToLower(q.R.Method)

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

		pageSize, err := strconv.ParseInt(query.Get("page_size"), 10, 64)
		if err != nil || pageSize <= 0 {
			q.Err = fmt.Errorf("invalid page_size parameter, must be a positive integer")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 构造过滤条件
		filter := QueryUsersFilter{
			Account:    NullableString(query.Get("account")),
			Name:       NullableString(query.Get("name")),
			Phone:      NullableString(query.Get("phone")),
			Email:      NullableString(query.Get("email")),
			Gender:     NullableString(query.Get("gender")),
			Status:     NullableString(query.Get("status")),
			CreateTime: NullableIntFromStr(query.Get("create_time")),
		}

		users, totalRows, err := h.repo.QueryUsers(ctx, nil, page, pageSize, filter)
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

	default:
		q.Err = fmt.Errorf("unsupported method: %s", method)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
}
