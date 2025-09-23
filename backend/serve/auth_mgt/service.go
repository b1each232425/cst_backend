package auth_mgt

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"w2w.io/null"

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
		AccessibleAPIs:    apis,
		AccessibleDomains: accessibleDomains,
	}

	return a, nil
}

// getAccessibleDomains 根据用户角色优先级获取可访问域列表
func getAccessibleDomains(ctx context.Context, pgConn *pgxpool.Pool, role cmn.TDomain, currentDomain cmn.TDomain) ([]int64, error) {
	var domains []cmn.TDomain
	var readableDomains []int64

	forceErr, _ := ctx.Value("force-error").(string) // 用于强制执行错误处理代码

	// 检查角色优先级
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

	for _, api := range authority.AccessibleAPIs {
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

// QueryDomains 分页查询域列表
func QueryDomains(ctx context.Context, page, pageSize int64, filter *QueryDomainsFilter) (result []DomainData, rowCount int64, err error) {
	forceErr, _ := ctx.Value("force-error").(string) // 用于强制执行错误处理代码

	result = make([]DomainData, 0)
	rowCount = 0

	// 获取数据库连接
	pgConn := cmn.GetPgxConn()
	if pgConn == nil || forceErr == "GetPgxConn" {
		err = fmt.Errorf("database connection is not available")
		z.Error(err.Error())
		return []DomainData{}, 0, err
	}

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 构建查询条件
	var conditions []string
	var args []interface{}
	argIndex := 1

	// 添加状态筛选条件
	if filter.Status != "" {
		conditions = append(conditions, "status = $"+fmt.Sprintf("%d", argIndex))
		args = append(args, filter.Status)
		argIndex++
	}

	// 添加域筛选条件
	if filter.Domain != "" {
		isDomain, isRole := IsValidDomainOrRole(filter.Domain)
		if !isDomain && !isRole {
			err = fmt.Errorf("筛选域的格式不正确: %s", filter.Domain)
			z.Error(err.Error())
			return []DomainData{}, 0, err
		}
		conditions = append(conditions, "domain = $"+fmt.Sprintf("%d", argIndex))
		args = append(args, filter.Domain)
		argIndex++
	}

	if filter.ParentDomain != "" {
		isDomain, _ := IsValidDomainOrRole(filter.ParentDomain)
		if !isDomain {
			err = fmt.Errorf("筛选父域的格式不正确: %s", filter.ParentDomain)
			z.Error(err.Error())
			return []DomainData{}, 0, err
		}
		likePattern := filter.ParentDomain + "%"
		conditions = append(conditions, "domain LIKE $"+fmt.Sprintf("%d", argIndex))
		args = append(args, likePattern)
		argIndex++
	}

	if filter.TargetType != "" {
		// 00只筛选domain字段不含^的数据，02只筛选domain字段含^的数据
		switch filter.TargetType {
		case "00":
			conditions = append(conditions, "domain NOT LIKE '%^%'")
		case "02":
			conditions = append(conditions, "domain LIKE '%^%'")
		default:
			err = fmt.Errorf("筛选目标的格式不正确: %s", filter.TargetType)
			z.Error(err.Error())
			return []DomainData{}, 0, err
		}
	}

	// 处理层级筛选参数
	childLevelInt := -1 // 默认值为-1，表示不进行层级筛选
	if filter.ChildLevel != "" {
		childLevelInt, err = strconv.Atoi(filter.ChildLevel)
		if err != nil || childLevelInt < 0 {
			childLevelInt = -1 // 解析失败时设为-1，不进行层级筛选
		}
	}

	// 根据层级参数进行筛选
	if childLevelInt >= 0 {
		// 构建层级筛选条件
		var levelConditions []string

		if childLevelInt == 0 {
			// level=0时，只查询直接隶属于当前父域的角色（A^xx格式）
			// 匹配格式：parentDomain^任意字符
			pattern := "^" + strings.ReplaceAll(filter.ParentDomain, ".", "\\.") + "\\^.+$"
			levelConditions = append(levelConditions, "domain ~ $"+fmt.Sprintf("%d", argIndex))
			args = append(args, pattern)
			argIndex++
		} else {
			// level>0时，按原有逻辑处理层级筛选
			if filter.ParentDomain != "" {
				// 如果存在父域筛选，基于父域构建层级条件
				for i := 1; i <= childLevelInt; i++ {
					// 构建正则表达式模式
					// 例如：父域为A.B，level为2时
					// 匹配 A.B.XX 和 A.B.XX.XX
					pattern := "^" + strings.ReplaceAll(filter.ParentDomain, ".", "\\.") + "(\\.[^.^]+){" + fmt.Sprintf("%d", i) + "}(\\^.*)?$"
					levelConditions = append(levelConditions, "domain ~ $"+fmt.Sprintf("%d", argIndex))
					args = append(args, pattern)
					argIndex++
				}
			} else {
				// 如果不存在父域筛选，直接基于层级数构建条件
				for i := 1; i <= childLevelInt; i++ {
					// 构建正则表达式模式
					// 例如：level为2时，匹配 XX 和 XX.XX
					pattern := "^([^.^]+)(\\.[^.^]+){" + fmt.Sprintf("%d", i-1) + "}(\\^.*)?$"
					levelConditions = append(levelConditions, "domain ~ $"+fmt.Sprintf("%d", argIndex))
					args = append(args, pattern)
					argIndex++
				}
			}
		}

		if len(levelConditions) > 0 {
			conditions = append(conditions, "("+strings.Join(levelConditions, " OR ")+")")
		}
	}

	// 添加模糊查询条件
	if filter.FuzzyCondition != "" {
		textPattern := "%" + filter.FuzzyCondition + "%"
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

	err = pgConn.QueryRow(ctx, countSQL, args...).Scan(&rowCount)
	if err != nil || forceErr == "QueryDomainCount" {
		err = fmt.Errorf("failed to query domain count: %w", err)
		z.Error(err.Error())
		return []DomainData{}, 0, err
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
		err = fmt.Errorf("failed to query domains: %w", err)
		z.Error(err.Error())
		return []DomainData{}, 0, err
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
			err = fmt.Errorf("failed to scan domain data: %w", err)
			z.Error(err.Error())
			return []DomainData{}, 0, err
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
			err = fmt.Errorf("failed to query domain APIs for %s: %w", domainData.Base.Domain, err)
			z.Error(err.Error())
			return []DomainData{}, 0, err
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
				err = fmt.Errorf("failed to scan API data for domain %s: %w", domainData.Base.Domain, err)
				apiRows.Close()
				z.Error(err.Error())
				return []DomainData{}, 0, err
			}
			domainData.APIs = append(domainData.APIs, &api)
		}
		apiRows.Close()

		// 检查API扫描过程中是否有错误
		err = apiRows.Err()
		if err != nil || forceErr == "apiRows.Err" {
			err = fmt.Errorf("error occurred during API scanning for domain %s: %w", domainData.Base.Domain, err)
			z.Error(err.Error())
			return []DomainData{}, 0, err
		}

		// 查询域的创建者信息
		if domainData.Base.Creator.Valid && domainData.Base.Creator.Int64 != CDefaultDomainCreatorID {
			var creator cmn.TUser
			creatorSQL := `
					SELECT id, account, official_name, mobile_phone, email
					FROM t_user
					WHERE id = $1
				`
			err = pgConn.QueryRow(ctx, creatorSQL, domainData.Base.Creator.Int64).Scan(
				&creator.ID,
				&creator.Account,
				&creator.OfficialName,
				&creator.MobilePhone,
				&creator.Email,
			)
			if err != nil || forceErr == "QueryCreator" {
				// 如果查询创建者失败，记录警告但不中断流程
				z.Warn(fmt.Sprintf("failed to query creator for domain %s: %v", domainData.Base.Domain, err))
			} else {
				// 将创建者信息赋值到域的详细信息
				domainData.Detail.Creator = creator
			}
		} else if domainData.Base.Creator.Int64 == CDefaultDomainCreatorID {
			// 填充默认创建者信息
			domainData.Detail.Creator = cmn.TUser{
				ID:           null.IntFrom(CDefaultDomainCreatorID),
				Account:      "system",
				OfficialName: null.StringFrom("系统"),
				MobilePhone:  null.StringFrom(""),
				Email:        null.StringFrom(""),
			}
		}

		result = append(result, domainData)
	}

	// 检查域扫描过程中是否有错误
	err = rows.Err()
	if err != nil || forceErr == "rows.Err" {
		err = fmt.Errorf("error occurred during domain scanning: %w", err)
		z.Error(err.Error())
		return []DomainData{}, 0, err
	}

	return result, rowCount, nil
}
