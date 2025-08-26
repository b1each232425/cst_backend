package auth_mgt

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"w2w.io/cmn"
	"w2w.io/null"
)

type Handler interface {
	HandleDomain(ctx context.Context)
	HandleQuerySelectableAPIs(ctx context.Context)
}

type handler struct {
}

func NewHandler() Handler {
	return &handler{}
}

// HandleQuerySelectableAPIs 处理查询可选API的请求
func (h *handler) HandleQuerySelectableAPIs(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	forceErr, _ := ctx.Value("force-error").(string) // 用于强制执行错误处理代码
	var err error

	method := strings.ToLower(q.R.Method)
	if method != "get" {
		q.Err = fmt.Errorf("invalid method: %s", q.R.Method)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	// 解析父域
	parentDomain := q.R.URL.Query().Get("parentDomain")
	if parentDomain != "" && !IsValidDomain(parentDomain) {
		q.Err = fmt.Errorf("invalid parentDomain format")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	apis, err := querySelectableAPIs(ctx, parentDomain)
	if err != nil || forceErr == "querySelectableAPIs" {
		q.Err = fmt.Errorf("failed to query selectable APIs: %v", err)
		q.RespErr()
		return
	}

	apisBytes, err := json.Marshal(apis)
	if err != nil || forceErr == "json.Marshal" {
		q.Err = fmt.Errorf("failed to marshal APIs: %v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	q.Msg.Status = 0
	q.Msg.Msg = "success"
	q.Msg.Data = apisBytes
	q.Resp()
	return
}

// HandleDomain 处理域相关的请求
func (h *handler) HandleDomain(ctx context.Context) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	forceErr, _ := ctx.Value("force-error").(string) // 用于强制执行错误处理代码
	var err error

	method := strings.ToLower(q.R.Method)

	if q.SysUser == nil || !q.SysUser.ID.Valid {
		q.Err = fmt.Errorf("user not logged in or invalid user ID")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	authority, err := GetUserAuthority(ctx)
	if err != nil || forceErr == "GetUserAuthority" {
		q.Err = fmt.Errorf("failed to get user authority: %v", err)
		q.RespErr()
		return
	}

	if authority.Role.Priority.Int64 != CDomainPrioritySuperAdmin {
		q.Err = fmt.Errorf("only super admin can manage domains")
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	switch method {
	case "post": // 创建域
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

		var reqData DomainData
		err = json.Unmarshal(body.Data, &reqData)
		if err != nil || forceErr == "json.Unmarshal" {
			q.Err = fmt.Errorf("failed to unmarshal domain data: %w", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if reqData.Base.Domain == "" {
			q.Err = fmt.Errorf("domain EN code cannot be empty")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if reqData.Base.Name == "" {
			q.Err = fmt.Errorf("domain name cannot be empty")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		isDomain, isRole := IsValidDomainOrRole(reqData.Base.Domain)
		if isDomain {
			// 如果要创建的目标是域，就自动设置优先级
			reqData.Base.Priority = null.NewInt(CDomainPrioritySuperAdmin, true)
		} else if isRole {
			// 如果要创建的目标是角色，就校验优先级
			if reqData.Base.Priority.Int64 != CDomainPriorityUser && reqData.Base.Priority.Int64 != CDomainPriorityAdmin {
				q.Err = fmt.Errorf("invalid priority for role")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
		} else {
			q.Err = fmt.Errorf("invalid domain EN code format")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		parentDomain := ParseFirstDomain(reqData.Base.Domain)

		// 校验选择的合法API
		if err = validateSelectedAPIs(ctx, reqData.APIs, parentDomain); err != nil {
			q.Err = fmt.Errorf("selected APIs are invalid: %w", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 获取数据库连接
		pgConn := cmn.GetPgxConn()
		if pgConn == nil {
			q.Err = fmt.Errorf("database connection is not available")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 开始事务
		tx, err := pgConn.Begin(ctx)
		if err != nil {
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
		}()

		// 向 t_domain 插入数据
		var domainID int64
		insertDomainSQL := `
            INSERT INTO t_domain (name, domain, priority, domain_id, creator, create_time, updated_by, update_time, status)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
            RETURNING id
        `
		currentTime := time.Now().UnixMilli()
		err = tx.QueryRow(ctx, insertDomainSQL,
			reqData.Base.Name,
			reqData.Base.Domain,
			reqData.Base.Priority,
			0, // domain_id 先插入0
			q.SysUser.ID.Int64,
			currentTime,
			q.SysUser.ID.Int64,
			currentTime,
			"01", // 默认状态为有效
		).Scan(&domainID)
		if err != nil {
			q.Err = fmt.Errorf("failed to insert domain data: %w", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 向 t_domain_api 插入数据
		if len(reqData.APIs) > 0 {
			insertDomainAPISQL := `
                INSERT INTO t_domain_api (domain, api, creator, create_time, updated_by, update_time, status)
                VALUES ($1, $2, $3, $4, $5, $6, $7)
            `
			for _, api := range reqData.APIs {
				_, err = tx.Exec(ctx, insertDomainAPISQL,
					domainID,
					api.ID,
					q.SysUser.ID.Int64,
					currentTime,
					q.SysUser.ID.Int64,
					currentTime,
					"01", // 默认状态为有效
				)
				if err != nil {
					q.Err = fmt.Errorf("failed to insert domain API association data: %w", err)
					z.Error(q.Err.Error())
					q.RespErr()
					return
				}
			}
		}

		// 提交事务
		err = tx.Commit(ctx)
		if err != nil {
			q.Err = fmt.Errorf("failed to commit transaction: %w", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 返回成功结果
		reqData.Base.ID = null.IntFrom(domainID)
		reqData.Base.CreateTime = null.IntFrom(currentTime)
		reqData.Base.UpdateTime = null.IntFrom(currentTime)
		reqData.Base.Creator = q.SysUser.ID
		reqData.Base.UpdatedBy = q.SysUser.ID
		reqData.Base.Status = null.StringFrom("01")

		q.Msg.Data, err = json.Marshal(reqData)
		if err != nil {
			q.Err = fmt.Errorf("failed to serialize response data: %w", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Msg.Status = 0
		q.Msg.Msg = "success"
		q.Resp()
		return

	default:
		q.Err = fmt.Errorf("invalid method: %s", q.R.Method)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}
}
