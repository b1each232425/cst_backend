package auth_mgt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"w2w.io/cmn"
	"w2w.io/null"
)

type Handler interface {
	HandleQuerySelectableAPIs(ctx context.Context)
	HandleDomain(ctx context.Context)
	HandleGetCurrentUserAuthority(ctx context.Context)
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

// HandleDomain 处理创建域的请求
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
		if err != nil || forceErr == "json.UnmarshalDomainData" {
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
			if !reqData.Base.Priority.Valid {
				q.Err = fmt.Errorf("priority is required for role")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
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
		if pgConn == nil || forceErr == "GetPgxConn" {
			q.Err = fmt.Errorf("database connection is not available")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 开始事务
		tx, err := pgConn.Begin(ctx)
		if err != nil || forceErr == "tx.Begin" {
			q.Err = fmt.Errorf("failed to begin transaction: %w", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer func() {
			if err != nil || q.Err != nil {
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

		// 向 t_domain 插入数据
		var domainID int64
		insertDomainSQL := `
            INSERT INTO t_domain (name, domain, priority, domain_id, creator, create_time, updated_by, update_time, remark, status)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
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
			reqData.Base.Remark,
			"01", // 默认状态为有效
		).Scan(&domainID)
		if err != nil || forceErr == "tx.QueryRow" {
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
				if err != nil || forceErr == "tx.Exec" {
					q.Err = fmt.Errorf("failed to insert domain API association data: %w", err)
					z.Error(q.Err.Error())
					q.RespErr()
					return
				}
			}
		}

		// 返回成功结果
		reqData.Base.ID = null.IntFrom(domainID)
		reqData.Base.CreateTime = null.IntFrom(currentTime)
		reqData.Base.UpdateTime = null.IntFrom(currentTime)
		reqData.Base.Creator = q.SysUser.ID
		reqData.Base.UpdatedBy = q.SysUser.ID
		reqData.Base.Status = null.StringFrom("01")

		q.Msg.Data, err = json.Marshal(reqData)
		if err != nil || forceErr == "json.MarshalResponse" {
			q.Err = fmt.Errorf("failed to serialize response data: %w", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Msg.Status = 0
		q.Msg.Msg = "success"
		q.Resp()
		return

	case "get": // 查询域列表

		// 解析筛选条件
		domain := q.R.URL.Query().Get("domain")
		parentDomain := q.R.URL.Query().Get("parentDomain")
		targetType := q.R.URL.Query().Get("targetType")
		status := q.R.URL.Query().Get("status")
		fuzzyCondition := q.R.URL.Query().Get("fuzzyCondition")
		childLevel := q.R.URL.Query().Get("childLevel")

		// 解析分页参数
		pageStr := q.R.URL.Query().Get("page")
		pageSizeStr := q.R.URL.Query().Get("pageSize")

		// 设置默认分页参数
		var page int64 = 1
		var pageSize int64 = 10

		// 解析页码
		if pageStr != "" {
			if p, err := strconv.ParseInt(pageStr, 10, 64); err == nil && p > 0 {
				page = p
			}
		}

		// 解析页大小，限制上限为100
		if pageSizeStr != "" {
			if ps, err := strconv.ParseInt(pageSizeStr, 10, 64); err == nil && ps > 0 {
				if ps > 100 {
					pageSize = 100
				} else {
					pageSize = ps
				}
			}
		}

		filter := QueryDomainsFilter{
			Domain:         domain,
			ParentDomain:   parentDomain,
			TargetType:     targetType,
			Status:         status,
			FuzzyCondition: fuzzyCondition,
			ChildLevel:     childLevel,
		}

		result, totalCount, err := QueryDomains(ctx, page, pageSize, &filter)
		if err != nil {
			q.Err = fmt.Errorf("failed to query domain list: %w", err)
			q.RespErr()
			return
		}

		// 序列化响应数据
		q.Msg.Data, err = json.Marshal(result)
		if err != nil || forceErr == "json.MarshalDomainList" {
			q.Err = fmt.Errorf("failed to serialize domain list: %w", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 设置总行数
		q.Msg.RowCount = totalCount

		q.Msg.Status = 0
		q.Msg.Msg = "success"
		q.Resp()
		return

	case "put": // 覆盖式更新域
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

		// 支持批量更新，解析为域数据数组
		var reqDataList []DomainData
		err = json.Unmarshal(body.Data, &reqDataList)
		if err != nil || forceErr == "json.UnmarshalDomainDataList" {
			q.Err = fmt.Errorf("failed to unmarshal domain data list: %w", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		if len(reqDataList) == 0 {
			q.Err = fmt.Errorf("domain data list cannot be empty")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 获取数据库连接
		pgConn := cmn.GetPgxConn()
		if pgConn == nil || forceErr == "GetPgxConn" {
			q.Err = fmt.Errorf("database connection is not available")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 开始事务
		tx, err := pgConn.Begin(ctx)
		if err != nil || forceErr == "tx.Begin" {
			q.Err = fmt.Errorf("failed to begin transaction: %w", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer func() {
			if err != nil || q.Err != nil {
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

		var updatedDomains []DomainData

		// 批量处理每个域的更新
		for _, reqData := range reqDataList {
			// 验证必要字段
			if !reqData.Base.ID.Valid || reqData.Base.ID.Int64 <= 0 {
				q.Err = fmt.Errorf("domain ID is required and must be valid")
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

			// 验证域的状态
			if reqData.Base.Status.Valid && (reqData.Base.Status.String != "00" && reqData.Base.Status.String != "01" && reqData.Base.Status.String != "02") {
				q.Err = fmt.Errorf("invalid domain status, must be one of '00', '01', '02'")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			// 检查域是否存在
			var existingDomain string
			checkSQL := `SELECT domain FROM t_domain WHERE id = $1`
			err = tx.QueryRow(ctx, checkSQL, reqData.Base.ID.Int64).Scan(&existingDomain)
			if err != nil || forceErr == "CheckDomainExists" {
				if errors.Is(err, pgx.ErrNoRows) {
					q.Err = fmt.Errorf("domain with ID %d does not exist", reqData.Base.ID.Int64)
				} else {
					q.Err = fmt.Errorf("failed to check domain existence: %w", err)
				}
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			// 校验域优先级字段
			isDomain, isRole := IsValidDomainOrRole(existingDomain)
			if isDomain {
				// 如果要创建的目标是域，就自动设置优先级
				reqData.Base.Priority = null.NewInt(CDomainPrioritySuperAdmin, true)
			} else if isRole {
				// 如果要创建的目标是角色，就校验优先级
				if !reqData.Base.Priority.Valid {
					q.Err = fmt.Errorf("priority is required for role")
					z.Error(q.Err.Error())
					q.RespErr()
					return
				}
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

			parentDomain := ParseFirstDomain(existingDomain)

			// 校验选择的合法API
			if err = validateSelectedAPIs(ctx, reqData.APIs, parentDomain); err != nil {
				q.Err = fmt.Errorf("selected APIs are invalid: %w", err)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			// 更新 t_domain 表
			updateDomainSQL := `
				UPDATE t_domain 
				SET name = $1, priority = $2, updated_by = $3, update_time = $4, remark = $5, status = $6
				WHERE id = $7
			`
			currentTime := time.Now().UnixMilli()
			_, err = tx.Exec(ctx, updateDomainSQL,
				reqData.Base.Name,
				reqData.Base.Priority,
				q.SysUser.ID.Int64,
				currentTime,
				reqData.Base.Remark,
				reqData.Base.Status.String,
				reqData.Base.ID.Int64,
			)
			if err != nil || forceErr == "UpdateDomain" {
				q.Err = fmt.Errorf("failed to update domain data: %w", err)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			// 删除现有的 API 关联
			deleteAPISQL := `DELETE FROM t_domain_api WHERE domain = $1`
			_, err = tx.Exec(ctx, deleteAPISQL, reqData.Base.ID.Int64)
			if err != nil || forceErr == "DeleteDomainAPIs" {
				q.Err = fmt.Errorf("failed to delete existing domain API associations: %w", err)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			// 重新插入 API 关联
			if len(reqData.APIs) > 0 {
				insertDomainAPISQL := `
					INSERT INTO t_domain_api (domain, api, creator, create_time, updated_by, update_time, status)
					VALUES ($1, $2, $3, $4, $5, $6, $7)
				`
				for _, api := range reqData.APIs {
					_, err = tx.Exec(ctx, insertDomainAPISQL,
						reqData.Base.ID.Int64,
						api.ID,
						q.SysUser.ID.Int64,
						currentTime,
						q.SysUser.ID.Int64,
						currentTime,
						"01", // 默认状态为有效
					)
					if err != nil || forceErr == "InsertDomainAPI" {
						q.Err = fmt.Errorf("failed to insert domain API association data: %w", err)
						z.Error(q.Err.Error())
						q.RespErr()
						return
					}
				}
			}

			// 使用domain进行筛选查询更新后的域数据
			updatedDomain, _, err := QueryDomains(ctx, 1, 1, &QueryDomainsFilter{Domain: existingDomain})
			if err != nil || len(updatedDomain) == 0 || forceErr == "QueryUpdatedDomain" {
				q.Err = fmt.Errorf("failed to query updated domain data: %w", err)
				q.RespErr()
				return
			}
			if len(updatedDomain) < 1 || forceErr == "QueryUpdatedDomain.NoRows" {
				q.Err = fmt.Errorf("updated domain not found")
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			updatedDomains = append(updatedDomains, updatedDomain[0])
		}

		// 返回成功结果
		q.Msg.Data, err = json.Marshal(updatedDomains)
		if err != nil || forceErr == "json.MarshalResponse" {
			q.Err = fmt.Errorf("failed to serialize response data: %w", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		q.Msg.Status = 0
		q.Msg.Msg = "success"
		q.Resp()
		return
	}

	q.Err = fmt.Errorf("invalid method: %s", q.R.Method)
	z.Error(q.Err.Error())
	q.RespErr()
	return
}

// HandleGetCurrentUserAuthority 处理获取当前用户权限信息的请求
func (h *handler) HandleGetCurrentUserAuthority(ctx context.Context) {
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

	authorityBytes, err := json.Marshal(authority)
	if err != nil || forceErr == "json.Marshal" {
		q.Err = fmt.Errorf("failed to marshal authority: %v", err)
		z.Error(q.Err.Error())
		q.RespErr()
		return
	}

	q.Msg.Status = 0
	q.Msg.Msg = "success"
	q.Msg.Data = authorityBytes
	q.Resp()
	return
}
