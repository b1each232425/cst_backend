package auth_mgt

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"w2w.io/cmn"
)

// GetUserAuthority 获取用户权限信息
func GetUserAuthority(ctx context.Context) (a *Authority, err error) {
	q := cmn.GetCtxValue(ctx)
	z.Info("---->" + cmn.FncName())

	if q.SysUser == nil || !q.SysUser.ID.Valid {
		e := fmt.Errorf("user not logged in or invalid user ID")
		z.Error(e.Error())
		return nil, e
	}

	forceErr, _ := ctx.Value("force-error").(string) // 用于强制执行错误处理代码

	pgConn := cmn.GetPgxConn()

	// 查询用户的当前角色
	var role cmn.TDomain
	domainQuery := "SELECT id, name, domain, priority FROM t_domain WHERE id = $1"
	err = pgConn.QueryRow(ctx, domainQuery, q.SysUser.Role).Scan(&role.ID, &role.Name, &role.Domain, &role.Priority)
	if err != nil || forceErr == "QueryRole" {
		e := fmt.Errorf("failed to get user role: %w", err)
		z.Error(e.Error())
		return nil, e
	}

	// 解析用户当前角色所在的域
	parts := strings.Split(role.Domain, "^")
	if len(parts) < 2 {
		e := fmt.Errorf("invalid user role format")
		z.Error(e.Error())
		return nil, e
	}
	domainStr := parts[0]

	// 查询用户所在域
	var domain cmn.TDomain
	domainQuery = "SELECT id, name, domain, priority FROM t_domain WHERE domain = $1"
	err = pgConn.QueryRow(ctx, domainQuery, domainStr).Scan(&domain.ID, &domain.Name, &domain.Domain, &domain.Priority)
	if err != nil || forceErr == "QueryDomain" {
		e := fmt.Errorf("failed to get user domain: %w", err)
		z.Error(e.Error())
		return nil, e
	}

	// 查询用户当前角色的API列表
	var apis []cmn.TVDomainAPI
	apiQuery := "SELECT auth_domain_id, domain_name, domain, priority, api_id, api_name, expose_path, access_action, access_control_level, data_access_mode, grant_source, data_scope FROM v_domain_api WHERE auth_domain_id = $1 AND status = '01'"
	rows, err := pgConn.Query(ctx, apiQuery, role.ID)
	if err != nil || forceErr == "QueryAPIs" {
		e := fmt.Errorf("failed to get user APIs: %w", err)
		z.Error(e.Error())
		return nil, e
	}
	defer rows.Close()
	for rows.Next() {
		var api cmn.TVDomainAPI
		err = rows.Scan(
			&api.AuthDomainID,
			&api.DomainName,
			&api.Domain,
			&api.Priority,
			&api.APIID,
			&api.APIName,
			&api.ExposePath,
			&api.AccessAction,
			&api.AccessControlLevel,
			&api.DataAccessMode,
			&api.GrantSource,
			&api.DataScope,
		)
		if err != nil || forceErr == "ScanAPI" {
			e := fmt.Errorf("failed to scan api row: %w", err)
			z.Error(e.Error())
			return nil, e
		}
		apis = append(apis, api)
	}
	if rows.Err() != nil || forceErr == "RowsErr" {
		e := fmt.Errorf("error occurred while iterating over API rows: %w", rows.Err())
		z.Error(e.Error())
		return nil, e
	}

	// 根据用户角色优先级获取可访问域列表
	accessibleDomains, err := getAccessibleDomains(ctx, pgConn, role, domain)
	if err != nil || forceErr == "QueryReadableDomains" {
		e := fmt.Errorf("failed to get readable domains: %w", err)
		z.Error(e.Error())
		return nil, e
	}

	a = &Authority{
		Role:              role,
		Domain:            domain,
		APIs:              apis,
		AccessibleDomains: accessibleDomains,
	}

	return a, nil
}

// getAccessibleDomains 根据用户角色优先级获取可访问域列表
func getAccessibleDomains(ctx context.Context, pgConn *pgxpool.Pool, role cmn.TDomain, currentDomain cmn.TDomain) ([]int64, error) {
	var domains []cmn.TDomain
	var readableDomains []int64

	forceErr, _ := ctx.Value("force-error").(string) // 用于强制执行错误处理代码

	// 检查角色优先级（Priority是null.Int类型）
	if !role.Priority.Valid {
		e := fmt.Errorf("role priority is not valid")
		z.Error(e.Error())
		return nil, e
	}

	switch role.Priority.Int64 {
	case CDomainPrioritySuperAdmin: // 超级管理员：可读所有顶层域下的域
		// 解析顶层域
		topDomain := strings.Split(currentDomain.Domain, ".")[0]
		pattern := topDomain + "%"

		query := "SELECT id, name, domain, priority FROM t_domain WHERE status = '01' AND domain LIKE $1 AND domain NOT LIKE '%^%' ORDER BY priority, domain"
		rows, err := pgConn.Query(ctx, query, pattern)
		if err != nil || forceErr == "CDomainPrioritySuperAdmin.Query" {
			return nil, fmt.Errorf("failed to query all domains: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var domain cmn.TDomain
			err = rows.Scan(&domain.ID, &domain.Name, &domain.Domain, &domain.Priority)
			if err != nil || forceErr == "CDomainPrioritySuperAdmin.Scan" {
				return nil, fmt.Errorf("failed to scan domain: %w", err)
			}
			domains = append(domains, domain)
		}
		if rows.Err() != nil || forceErr == "CDomainPrioritySuperAdmin.RowsErr" {
			return nil, fmt.Errorf("error occurred while iterating domains: %w", rows.Err())
		}

	case CDomainPriorityAdmin: // 普通管理员：本域、所有父域、所有子域
		// 添加本域
		domains = append(domains, currentDomain)

		// 获取所有父域
		parentDomains, err := getParentDomains(ctx, pgConn, currentDomain.Domain)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent domains: %w", err)
		}
		domains = append(domains, parentDomains...)

		// 获取所有子域
		childDomains, err := getChildDomains(ctx, pgConn, currentDomain.Domain)
		if err != nil {
			return nil, fmt.Errorf("failed to get child domains: %w", err)
		}
		domains = append(domains, childDomains...)

	case CDomainPriorityUser: // 普通用户：本域、所有父域
		// 添加本域
		domains = append(domains, currentDomain)

		// 获取所有父域
		parentDomains, err := getParentDomains(ctx, pgConn, currentDomain.Domain)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent domains: %w", err)
		}
		domains = append(domains, parentDomains...)

	default:
		e := fmt.Errorf("unsupported role priority: %d", role.Priority.Int64)
		z.Error(e.Error())
		return nil, e
	}

	// 构建DomainList结构体
	for _, domain := range domains {
		if domain.ID.Valid {
			readableDomains = append(readableDomains, domain.ID.Int64)
		}
	}

	return readableDomains, nil
}

// getParentDomains 获取指定域的所有父域列表
func getParentDomains(ctx context.Context, pgConn *pgxpool.Pool, domainPath string) ([]cmn.TDomain, error) {
	forceErr, _ := ctx.Value("force-error").(string) // 用于强制执行错误处理代码

	if pgConn == nil || forceErr == "getParentDomains.pgConn.nil" {
		e := fmt.Errorf("pgConn is nil")
		z.Error(e.Error())
		return nil, e
	}
	if domainPath == "" || forceErr == "getParentDomains.domainPath.empty" {
		e := fmt.Errorf("domainPath is empty")
		z.Error(e.Error())
		return nil, e
	}

	var parentDomains []cmn.TDomain

	// 解析域路径，获取所有父域路径
	parts := strings.Split(domainPath, ".")
	for i := 1; i < len(parts); i++ {
		parentPath := strings.Join(parts[:i], ".")

		// 查询父域信息
		var parentDomain cmn.TDomain
		query := "SELECT id, name, domain, priority FROM t_domain WHERE domain = $1 AND domain NOT LIKE '%^%' AND status = '01'"
		err := pgConn.QueryRow(ctx, query, parentPath).Scan(
			&parentDomain.ID, &parentDomain.Name, &parentDomain.Domain, &parentDomain.Priority)
		if err != nil || forceErr == "getParentDomains.QueryRow" {
			// 如果父域不存在，跳过
			continue
		}
		parentDomains = append(parentDomains, parentDomain)
	}

	return parentDomains, nil
}

// getChildDomains 获取指定域的所有子域列表
func getChildDomains(ctx context.Context, pgConn *pgxpool.Pool, domainPath string) ([]cmn.TDomain, error) {
	forceErr, _ := ctx.Value("force-error").(string) // 用于强制执行错误处理代码

	if pgConn == nil || forceErr == "getChildDomains.pgConn.nil" {
		e := fmt.Errorf("pgConn is nil")
		z.Error(e.Error())
		return nil, e
	}
	if domainPath == "" || forceErr == "getChildDomains.domainPath.empty" {
		e := fmt.Errorf("domainPath is empty")
		z.Error(e.Error())
		return nil, e
	}

	var childDomains []cmn.TDomain

	// 查询所有以当前域路径开头的子域
	query := "SELECT id, name, domain, priority FROM t_domain WHERE domain LIKE $1 AND domain != $2 AND domain NOT LIKE '%^%' AND status = '01' ORDER BY domain"
	rows, err := pgConn.Query(ctx, query, domainPath+".%", domainPath)
	if err != nil || forceErr == "getChildDomains.Query" {
		return nil, fmt.Errorf("failed to query child domains: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var childDomain cmn.TDomain
		err = rows.Scan(&childDomain.ID, &childDomain.Name, &childDomain.Domain, &childDomain.Priority)
		if err != nil || forceErr == "getChildDomains.Scan" {
			return nil, fmt.Errorf("failed to scan child domain: %w", err)
		}
		childDomains = append(childDomains, childDomain)
	}

	if rows.Err() != nil || forceErr == "getChildDomains.RowsErr" {
		return nil, fmt.Errorf("error occurred while iterating child domains: %w", rows.Err())
	}

	return childDomains, nil
}

// CheckUserAPIAccessible 检查用户是否可访问特定API
// authority 参数可以为 nil，如果为 nil，则会自动获取当前用户的权限信息
func CheckUserAPIAccessible(ctx context.Context, authority *Authority, apiPath string, accessAction string) (bool, error) {
	var err error
	a := authority

	if a == nil {
		a, err = GetUserAuthority(ctx)
		if err != nil {
			return false, err
		}
	}

	switch accessAction {
	case CAPIAccessActionCreate:
		return checkUserAPIAccessible(ctx, a, apiPath, CAPIAccessActionCreate)
	case CAPIAccessActionRead:
		return checkUserAPIAccessible(ctx, a, apiPath, CAPIAccessActionRead)
	case CAPIAccessActionUpdate:
		return checkUserAPIAccessible(ctx, a, apiPath, CAPIAccessActionUpdate)
	case CAPIAccessActionDelete:
		return checkUserAPIAccessible(ctx, a, apiPath, CAPIAccessActionDelete)
	case CAPIAccessActionFull:
		writable, err := checkUserAPIAccessible(ctx, a, apiPath, CAPIAccessActionCreate)
		if err != nil {
			return false, err
		}
		readable, err := checkUserAPIAccessible(ctx, a, apiPath, CAPIAccessActionRead)
		if err != nil {
			return false, err
		}
		editable, err := checkUserAPIAccessible(ctx, a, apiPath, CAPIAccessActionUpdate)
		if err != nil {
			return false, err
		}
		deleted, err := checkUserAPIAccessible(ctx, a, apiPath, CAPIAccessActionDelete)
		if err != nil {
			return false, err
		}
		return writable && readable && editable && deleted, nil
	default:
		e := fmt.Errorf("invalid access mode: %s", accessAction)
		z.Error(e.Error())
		return false, e
	}
}

// CheckUserAPIWritable 检查用户对特定API是否有写权限
func checkUserAPIAccessible(ctx context.Context, authority *Authority, apiPath string, accessAction string) (bool, error) {
	z.Info("---->" + cmn.FncName())

	forceErr, _ := ctx.Value("force-error").(string) // 用于强制执行错误处理代码

	if authority == nil || forceErr == "checkUserAPIAccessible.authority.nil" {
		e := fmt.Errorf("authority is nil")
		z.Error(e.Error())
		return false, e
	}

	for _, api := range authority.APIs {
		if strings.EqualFold(api.ExposePath.String, apiPath) && (api.AccessAction.String == CAPIAccessActionFull || api.AccessAction.String == accessAction) {
			return true, nil
		}
	}

	return false, nil
}

// GetDomainRelationship 获得用户所在域与目标域的关系
// 判断目标域是用户所在域的什么关系域
func GetDomainRelationship(ctx context.Context, authority *Authority, targetDomain string) (string, error) {
	z.Info("---->" + cmn.FncName())

	forceErr, _ := ctx.Value("force-error").(string) // 用于强制执行错误处理代码

	if !IsValidDomain(targetDomain) {
		e := fmt.Errorf("invalid target domain: %s", targetDomain)
		z.Error(e.Error())
		return "", e
	}

	if authority == nil {
		e := fmt.Errorf("authority is nil")
		z.Error(e.Error())
		return "", e
	}

	normalize := func(s string) string {
		s = strings.TrimSpace(s)
		if s == "" {
			return ""
		}
		if i := strings.IndexByte(s, '^'); i >= 0 {
			s = s[:i]
		}
		return strings.ToLower(s)
	}
	src := normalize(authority.Domain.Domain)
	dst := normalize(targetDomain)
	if src == "" || dst == "" {
		return "", fmt.Errorf("empty domain after normalization")
	}

	// 检查是否为同一域
	if src == dst {
		return CDomainRelationshipSelf, nil
	}

	// 检查父子关系
	isParentOf := func(parent, child string) bool {
		if parent == "" || child == "" || parent == child || forceErr == "GetDomainRelationship.isParentOf.empty" {
			return false
		}
		if len(child) <= len(parent) {
			return false
		}
		return strings.HasPrefix(child, parent+".")
	}
	if isParentOf(src, dst) {
		return CDomainRelationshipChild, nil
	}
	if isParentOf(dst, src) {
		return CDomainRelationshipParent, nil
	}

	return CDomainRelationshipPeer, nil
}

// querySelectableAPIs 查询可选API列表
// 根据父域级别决定查询范围：机构级别查询所有API，部门/组/角色级别查询父域的API子集
func querySelectableAPIs(ctx context.Context, parentDomain string) (apis []*cmn.TAPI, err error) {
	forceErr, _ := ctx.Value("force-error").(string) // 用于强制执行错误处理代码

	apis = make([]*cmn.TAPI, 0)

	// 获取数据库连接
	pgConn := cmn.GetPgxConn()
	if pgConn == nil || forceErr == "querySelectableAPIs.pgConn.nil" {
		e := fmt.Errorf("pgx connection is nil")
		z.Error(e.Error())
		return nil, e
	}

	switch DomainLevel(parentDomain) {
	case 0: // 不存在父域，判断为机构，可选系统的所有API
		// 直接从t_api表中查询所有有效的API
		apiQuery := `SELECT id, name, expose_path, access_action, access_control_level
					FROM t_api 
					WHERE status = '01' AND configurable = true
					ORDER BY name`

		rows, err := pgConn.Query(ctx, apiQuery)
		if err != nil || forceErr == "querySelectableAPIs.QueryAllAPIs" {
			e := fmt.Errorf("failed to query APIs: %w", err)
			z.Error(e.Error())
			return nil, e
		}
		defer rows.Close()

		for rows.Next() {
			var api cmn.TAPI
			err = rows.Scan(
				&api.ID,
				&api.Name,
				&api.ExposePath,
				&api.AccessAction,
				&api.AccessControlLevel,
			)
			if err != nil || forceErr == "querySelectableAPIs.ScanAllAPIs" {
				e := fmt.Errorf("failed to scan rows: %w", err)
				z.Error(e.Error())
				return nil, e
			}
			apis = append(apis, &api)
		}

		if rows.Err() != nil || forceErr == "querySelectableAPIs.AllAPIsRowsErr" {
			e := fmt.Errorf("error occured while scanning rows: %w", rows.Err())
			z.Error(e.Error())
			return nil, e
		}

	default: // 存在父域，判断为部门/组/角色，可选父域的API子集
		// 从v_domain_api视图中查询父域的API列表
		apiQuery := `SELECT DISTINCT api_id, api_name, expose_path, access_action, access_control_level
					FROM v_domain_api 
					WHERE domain = $1 AND status = '01' AND configurable = true
					ORDER BY api_name`

		rows, err := pgConn.Query(ctx, apiQuery, parentDomain)
		if err != nil || forceErr == "querySelectableAPIs.QueryDomainAPIs" {
			e := fmt.Errorf("failed to query APIs: %w", err)
			z.Error(e.Error())
			return nil, e
		}
		defer rows.Close()

		for rows.Next() {
			var api cmn.TAPI
			err = rows.Scan(
				&api.ID,
				&api.Name,
				&api.ExposePath,
				&api.AccessAction,
				&api.AccessControlLevel,
			)
			if err != nil || forceErr == "querySelectableAPIs.ScanDomainAPIs" {
				e := fmt.Errorf("failed to scan rows: %w", err)
				z.Error(e.Error())
				return nil, e
			}
			apis = append(apis, &api)
		}

		if rows.Err() != nil || forceErr == "querySelectableAPIs.DomainAPIsRowsErr" {
			e := fmt.Errorf("error occured while scanning rows: %w", rows.Err())
			z.Error(e.Error())
			return nil, e
		}
	}

	return apis, nil
}
