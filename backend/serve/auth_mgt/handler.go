package auth_mgt

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

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
		// 获取数据库连接
		pgConn := cmn.GetPgxConn()
		if pgConn == nil || forceErr == "GetPgxConn" {
			q.Err = fmt.Errorf("database connection is not available")
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 解析筛选条件
		domain := q.R.URL.Query().Get("domain")
		parentDomain := q.R.URL.Query().Get("parentDomain")
		onlyRole := q.R.URL.Query().Get("onlyRole")
		status := q.R.URL.Query().Get("status")
		fuzzyCondition := q.R.URL.Query().Get("fuzzyCondition")

		// 解析分页参数
		pageStr := q.R.URL.Query().Get("page")
		pageSizeStr := q.R.URL.Query().Get("pageSize")

		// 设置默认分页参数
		page := 1
		pageSize := 10

		// 解析页码
		if pageStr != "" {
			if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
				page = p
			}
		}

		// 解析页大小，限制上限为100
		if pageSizeStr != "" {
			if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
				if ps > 100 {
					pageSize = 100
				} else {
					pageSize = ps
				}
			}
		}

		// 计算偏移量
		offset := (page - 1) * pageSize

		result := make([]DomainData, 0)

		// 构建查询条件
		var conditions []string
		var args []interface{}
		argIndex := 1

		// 添加状态筛选条件
		if status != "" {
			conditions = append(conditions, "status = $"+fmt.Sprintf("%d", argIndex))
			args = append(args, status)
			argIndex++
		}

		// 添加域筛选条件
		if domain != "" {
			isDomain, isRole := IsValidDomainOrRole(domain)
			if !isDomain && !isRole {
				q.Err = fmt.Errorf("筛选域的格式不正确: %s", domain)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			conditions = append(conditions, "domain = $"+fmt.Sprintf("%d", argIndex))
			args = append(args, domain)
			argIndex++
		}

		if parentDomain != "" {
			isDomain, _ := IsValidDomainOrRole(parentDomain)
			if !isDomain {
				q.Err = fmt.Errorf("筛选父域的格式不正确: %s", parentDomain)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}
			likePattern := parentDomain + "%"
			conditions = append(conditions, "domain LIKE $"+fmt.Sprintf("%d", argIndex))
			args = append(args, likePattern)
			argIndex++
		}

		if onlyRole == "true" {
			conditions = append(conditions, "domain LIKE $"+fmt.Sprintf("%d", argIndex))
			args = append(args, "%^%")
			argIndex++
		}

		// 添加模糊查询条件
		if fuzzyCondition != "" {
			textPattern := "%" + fuzzyCondition + "%"
			conditions = append(conditions, fmt.Sprintf(`(
				name ILIKE $%d OR
				domain ILIKE $%d OR
				remark ILIKE $%d
			)`, argIndex, argIndex, argIndex))
			args = append(args, textPattern)
			argIndex++
		}

		// 构建WHERE子句
		whereClause := ""
		if len(conditions) > 0 {
			whereClause = "WHERE " + strings.Join(conditions, " AND ")
		}

		// 查询总行数
		countSQL := `
			SELECT COUNT(*)
			FROM t_domain
			` + whereClause + `
		`

		var totalCount int64
		err := pgConn.QueryRow(ctx, countSQL, args...).Scan(&totalCount)
		if err != nil || forceErr == "QueryDomainCount" {
			q.Err = fmt.Errorf("failed to query domain count: %w", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}

		// 从 t_domain 查询域基本信息（带分页）
		queryDomainSQL := `
			SELECT id, name, domain, priority, updated_by, update_time, creator, create_time, status
			FROM t_domain
			` + whereClause + `
			ORDER BY create_time DESC
			LIMIT $` + fmt.Sprintf("%d", argIndex) + ` OFFSET $` + fmt.Sprintf("%d", argIndex+1) + `
		`

		// 添加分页参数到查询参数列表
		args = append(args, pageSize, offset)

		rows, err := pgConn.Query(ctx, queryDomainSQL, args...)
		if err != nil || forceErr == "QueryDomains" {
			q.Err = fmt.Errorf("failed to query domains: %w", err)
			z.Error(q.Err.Error())
			q.RespErr()
			return
		}
		defer rows.Close()

		// 扫描域数据
		for rows.Next() {
			var domainData DomainData
			err = rows.Scan(
				&domainData.Base.ID,
				&domainData.Base.Name,
				&domainData.Base.Domain,
				&domainData.Base.Priority,
				&domainData.Base.UpdatedBy,
				&domainData.Base.UpdateTime,
				&domainData.Base.Creator,
				&domainData.Base.CreateTime,
				&domainData.Base.Status,
			)
			if err != nil || forceErr == "ScanDomains" {
				q.Err = fmt.Errorf("failed to scan domain data: %w", err)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			// 从 v_domain_api 查询该域的API列表
			queryAPISQL := `
				SELECT api_id, api_name, expose_path, access_action, creator, create_time, status
				FROM v_domain_api
				WHERE domain = $1 AND status = '01'
				ORDER BY api_id
			`
			apiRows, err := pgConn.Query(ctx, queryAPISQL, domainData.Base.Domain)
			if err != nil || forceErr == "QueryDomainAPIs" {
				q.Err = fmt.Errorf("failed to query domain APIs for %s: %w", domainData.Base.Domain, err)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			// 扫描API数据
			domainData.APIs = make([]*cmn.TAPI, 0)
			for apiRows.Next() {
				var api cmn.TAPI
				err = apiRows.Scan(
					&api.ID,
					&api.Name,
					&api.ExposePath,
					&api.AccessAction,
					&api.Creator,
					&api.CreateTime,
					&api.Status,
				)
				if err != nil || forceErr == "ScanDomainAPIs" {
					q.Err = fmt.Errorf("failed to scan API data for domain %s: %w", domainData.Base.Domain, err)
					z.Error(q.Err.Error())
					q.RespErr()
					apiRows.Close()
					return
				}
				domainData.APIs = append(domainData.APIs, &api)
			}
			apiRows.Close()

			// 检查API扫描过程中是否有错误
			err = apiRows.Err()
			if err != nil || forceErr == "apiRows.Err" {
				q.Err = fmt.Errorf("error occurred during API scanning for domain %s: %w", domainData.Base.Domain, err)
				z.Error(q.Err.Error())
				q.RespErr()
				return
			}

			result = append(result, domainData)
		}

		// 检查域扫描过程中是否有错误
		err = rows.Err()
		if err != nil || forceErr == "rows.Err" {
			q.Err = fmt.Errorf("error occurred during domain scanning: %w", err)
			z.Error(q.Err.Error())
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
