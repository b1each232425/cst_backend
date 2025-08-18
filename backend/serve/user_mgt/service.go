package user_mgt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"w2w.io/cmn"
	"w2w.io/null"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nyaruka/phonenumbers"
)

type Service interface {
	QueryUsers(ctx context.Context, tx pgx.Tx, page, pageSize int64, filter QueryUsersFilter) ([]User, int64, error)
	InsertUsers(ctx context.Context, tx pgx.Tx, users []User) ([]User, error)
	InsertUsersWithAccount(ctx context.Context, tx pgx.Tx, users []User) error
	CheckTUserFieldExists(ctx context.Context, tx pgx.Tx, field string, value any) (bool, error)
	CheckTUserRowExists(ctx context.Context, tx pgx.Tx, fields map[string]any) (bool, *User, error)
	GenerateUniqueAccount(ctx context.Context, tx pgx.Tx, length int, maxAttempts int) (string, error)
	ValidateUserToBeInsert(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error)
	QueryUserCurrentRole(ctx context.Context, userId null.Int) (null.Int, null.String, error)
}

type service struct {
	pgxConn *pgxpool.Pool
}

func NewService() Service {
	return &service{
		pgxConn: cmn.GetPgxConn(),
	}
}

// QueryUsers 查询用户列表
// 第一页从 1 开始，pageSize 为每页记录数
// 返回值: 用户列表、总记录数、错误
func (r *service) QueryUsers(ctx context.Context, tx pgx.Tx, page, pageSize int64, filter QueryUsersFilter) ([]User, int64, error) {
	z.Info("---->" + cmn.FncName())

	forceErr, _ := ctx.Value("force-error").(string)

	if page <= 0 || pageSize <= 0 {
		e := fmt.Errorf("page and page size must be positive integers")
		z.Error(e.Error())
		return []User{}, 0, e
	}

	// 构建 WHERE 条件和参数
	var where []string
	var args []interface{}
	argIndex := 1

	if filter.FuzzyCondition.Valid && filter.FuzzyCondition.String != "" {
		textPattern := "%" + filter.FuzzyCondition.String + "%"
		where = append(where, fmt.Sprintf(`(
			u.account ILIKE $%d OR
			u.official_name ILIKE $%d OR
			u.mobile_phone ILIKE $%d OR
			u.email ILIKE $%d OR
			u.id_card_no ILIKE $%d
		)`, argIndex, argIndex, argIndex, argIndex, argIndex))
		args = append(args, textPattern)
		argIndex++
	}
	if filter.ID.Valid {
		where = append(where, fmt.Sprintf("u.id = $%d", argIndex))
		args = append(args, filter.ID.Int64)
		argIndex++
	}
	if filter.Gender.Valid {
		where = append(where, fmt.Sprintf("u.gender = $%d", argIndex))
		args = append(args, filter.Gender.String)
		argIndex++
	}
	if filter.Status.Valid {
		where = append(where, fmt.Sprintf("u.status = $%d", argIndex))
		args = append(args, filter.Status.String)
		argIndex++
	}
	if filter.CreateTime.Valid {
		where = append(where, fmt.Sprintf("u.create_time >= $%d", argIndex))
		args = append(args, filter.CreateTime.Int64)
		argIndex++
	}
	if filter.Domain.Valid {
		// 在 v_user_domain 里找出包含指定 domain 的用户
		where = append(where, fmt.Sprintf(`
        EXISTS (
            SELECT 1 FROM v_user_domain vd
            WHERE vd.user_id = u.id
              AND vd.domain  = $%d
        )`, argIndex))
		args = append(args, filter.Domain.String)
		argIndex++
	}

	// 构建查询总记录数 SQL
	var whereClause string
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}
	countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM t_user u %s`, whereClause)

	var rowCount int64
	var err error

	// 查询总记录数
	if tx != nil {
		err = tx.QueryRow(ctx, countSQL, args...).Scan(&rowCount)
	} else {
		err = r.pgxConn.QueryRow(ctx, countSQL, args...).Scan(&rowCount)
	}
	if err != nil || forceErr == "count" {
		e := fmt.Errorf("failed to count user records: %w", err)
		z.Error(e.Error())
		return []User{}, 0, e
	}

	// 分页查询数据（追加 LIMIT 和 OFFSET 参数）
	offset := (page - 1) * pageSize
	args = append(args, pageSize, offset)
	querySQL := fmt.Sprintf(`
    SELECT u.id, 
           u.account, 
           u.official_name, 
           u.gender, 
           u.mobile_phone, 
           u.email,
           u.category,
           u.status,
           u.type,
           u.id_card_no,
           u.id_card_type,
           u.role,
           u.logon_time,
           u.create_time,
           u.update_time,
           u.creator,
           COALESCE((
               SELECT array_agg(d.domain)
               FROM v_user_domain d
               WHERE d.user_id = u.id
           ), '{}') AS domains,
           COALESCE((
               SELECT json_agg(
                   json_build_object(
                   	   'Role', api.role,
                       'APIID', api.api_id,
                       'APIName', api.api_name,
                       'APIExposePath', api.api_expose_path,
                       'DomainName', api.domain_name,
                       'DomainID', api.domain_id,
                       'Domain', api.domain,
                       'Priority', api.priority
                   )
               )
               FROM v_user_domain_api api
               WHERE api.user_id = u.id
           ), '[]') AS apis
    FROM t_user u
    %s
    ORDER BY u.create_time DESC
    LIMIT $%d OFFSET $%d`, whereClause, argIndex, argIndex+1)

	var rows pgx.Rows
	if tx != nil {
		rows, err = tx.Query(ctx, querySQL, args...)
	} else {
		rows, err = r.pgxConn.Query(ctx, querySQL, args...)
	}
	if err != nil || forceErr == "query" {
		e := fmt.Errorf("failed to query user list: %w", err)
		z.Error(e.Error())
		return []User{}, 0, e
	}
	defer rows.Close()

	var users = make([]User, 0, pageSize)
	for rows.Next() {
		var user User
		var apisJSON []byte
		err = rows.Scan(
			&user.ID,
			&user.Account,
			&user.OfficialName,
			&user.Gender,
			&user.MobilePhone,
			&user.Email,
			&user.Category,
			&user.Status,
			&user.Type,
			&user.IDCardNo,
			&user.IDCardType,
			&user.Role,
			&user.LogonTime,
			&user.CreateTime,
			&user.UpdateTime,
			&user.Creator,
			&user.Domains,
			&apisJSON,
		)
		if err != nil || forceErr == "scan" {
			e := fmt.Errorf("failed to scan user row: %w", err)
			z.Error(e.Error())
			return []User{}, 0, e
		}

		// 解析APIs JSON数据
		if len(apisJSON) > 0 && string(apisJSON) != "[]" {
			err = json.Unmarshal(apisJSON, &user.APIs)
			if err != nil || forceErr == "json.Unmarshal" {
				z.Warn(fmt.Sprintf("failed to unmarshal APIs JSON for user %d: %v", user.ID.Int64, err))
				user.APIs = []cmn.TVUserDomainAPI{} // 设置为空数组
			}
		} else {
			user.APIs = []cmn.TVUserDomainAPI{} // 设置为空数组
		}

		users = append(users, user)
	}

	if rows.Err() != nil || forceErr == "reading" {
		e := fmt.Errorf("error occurred during row iteration: %w", rows.Err())
		z.Error(e.Error())
		return []User{}, 0, e
	}

	return users, rowCount, nil
}

// InsertUsers 批量插入用户数据
// 必要字段: account, category
// 请不要轻易调用该方法插入用户，该方法不会对用户信息做全面的校验，需要先调用 ValidateUserToBeInsert 方法验证用户信息的合法性
// 返回值: 成功插入的用户列表（包含生成的ID）、错误
func (r *service) InsertUsers(ctx context.Context, tx pgx.Tx, users []User) ([]User, error) {
	z.Info("---->" + cmn.FncName())

	forceErr, _ := ctx.Value("force-error").(string)

	var err error

	if len(users) == 0 {
		e := fmt.Errorf("no users to insert")
		z.Error(e.Error())
		return []User{}, e
	}

	// 用于存储成功插入的用户
	var insertedUsers []User

	for i := range users {
		if users[i].Account == "" {
			e := fmt.Errorf("user account is required")
			z.Error(e.Error())
			return []User{}, e
		}
		if users[i].Category == "" {
			e := fmt.Errorf("user category is required")
			z.Error(e.Error())
			return []User{}, e
		}

		if users[i].MobilePhone.Valid {
			// 将手机号格式化为无空格无特殊字符的E.164标准格式
			number := strings.TrimSpace(users[i].MobilePhone.String)
			if number == "" {
				e := fmt.Errorf("mobile phone cannot be empty")
				z.Error(e.Error())
				return []User{}, e
			}

			region := strings.ToUpper(strings.TrimSpace(DefaultRegion))
			parseRegion := region
			if strings.HasPrefix(number, "+") {
				parseRegion = "" // 已含国家码
			}

			num, err := phonenumbers.Parse(number, parseRegion)
			if err != nil {
				e := fmt.Errorf("failed to parse mobile phone %s: %w", number, err)
				z.Error(e.Error())
				return []User{}, e
			}
			if !phonenumbers.IsPossibleNumber(num) || !phonenumbers.IsValidNumber(num) {
				e := fmt.Errorf("mobile phone %s is not a valid number", number)
				z.Error(e.Error())
				return []User{}, e
			}

			users[i].MobilePhone = null.StringFrom(phonenumbers.Format(num, phonenumbers.E164))
		}

		if !users[i].IDCardNo.Valid && !users[i].MobilePhone.Valid && !users[i].Email.Valid {
			users[i].Type = null.StringFrom("00") // 匿名用户
		} else {
			users[i].Type = null.StringFrom("02") // 注册用户
		}

		// 插入用户数据
		insertSQL := `INSERT INTO t_user (
			category,
			type,
			official_name,
			id_card_type,
			id_card_no,
			account,
			mobile_phone,
			email,
			gender,
			birthday,
			creator,
			status,
			remark,
            user_token,
			create_time,
			update_time
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, crypt($14, gen_salt('bf')), $15, $16
		)`

		if tx != nil {
			_, err = tx.Exec(ctx, insertSQL,
				users[i].Category,
				users[i].Type.String,
				users[i].OfficialName,
				users[i].IDCardType,
				users[i].IDCardNo,
				users[i].Account,
				users[i].MobilePhone,
				users[i].Email,
				users[i].Gender,
				users[i].Birthday,
				users[i].Creator.Int64,
				r.orDefault(users[i].Status, "00"),
				users[i].Remark,
				InitialPwd, // 设置初始密码
				time.Now().UnixMilli(),
				time.Now().UnixMilli(),
			)
		} else {
			_, err = r.pgxConn.Exec(ctx, insertSQL,
				users[i].Category,
				users[i].Type.String,
				users[i].OfficialName,
				users[i].IDCardType,
				users[i].IDCardNo,
				users[i].Account,
				users[i].MobilePhone,
				users[i].Email,
				users[i].Gender,
				users[i].Birthday,
				users[i].Creator.Int64,
				r.orDefault(users[i].Status, "00"),
				users[i].Remark,
				InitialPwd, // 设置初始密码
				time.Now().UnixMilli(),
				time.Now().UnixMilli(),
			)
		}

		if err != nil || forceErr == "Exec" {
			e := fmt.Errorf("failed to insert user %s: %w", users[i].Account, err)
			z.Error(e.Error())
			return []User{}, e
		}

		// 读取用户ID
		var userID int64
		if tx != nil {
			err = tx.QueryRow(ctx, "SELECT id FROM t_user WHERE account = $1", users[i].Account).Scan(&userID)
		} else {
			err = r.pgxConn.QueryRow(ctx, "SELECT id FROM t_user WHERE account = $1", users[i].Account).Scan(&userID)
		}
		if err != nil || forceErr == "QueryUserID" {
			e := fmt.Errorf("failed to retrieve user ID for %s: %w", users[i].Account, err)
			z.Error(e.Error())
			return []User{}, e
		}

		// 设置用户ID并添加到成功插入的用户列表
		users[i].ID = null.IntFrom(userID)
		users[i].CreateTime = null.IntFrom(time.Now().UnixMilli())
		users[i].UpdateTime = null.IntFrom(time.Now().UnixMilli())

		// 插入用户角色到 t_user_domain
		if len(users[i].Domains) > 0 {
			insertDomainSQL := `
				INSERT INTO t_user_domain (sys_user, domain)
				VALUES ($1, (SELECT id FROM t_domain WHERE domain = $2))
			`
			for _, domain := range users[i].Domains {
				if tx != nil {
					_, err = tx.Exec(ctx, insertDomainSQL, userID, domain.String)
				} else {
					_, err = r.pgxConn.Exec(ctx, insertDomainSQL, userID, domain.String)
				}
				if err != nil || forceErr == "InsertUserDomain" {
					e := fmt.Errorf("failed to insert user domain %s for user %s: %w", domain.String, users[i].Account, err)
					z.Error(e.Error())
					return []User{}, e
				}
			}
		}

		// 将成功插入的用户添加到结果列表
		insertedUsers = append(insertedUsers, users[i])
	}

	return insertedUsers, nil
}

// InsertUsersWithAccount 批量插入用户数据，并为每个用户生成唯一账号
// 必要字段: category
func (r *service) InsertUsersWithAccount(ctx context.Context, tx pgx.Tx, users []User) error {
	z.Info("---->" + cmn.FncName())

	forceErr, _ := ctx.Value("force-error").(string)

	var err error

	for i := range users {
		users[i].Account, err = r.GenerateUniqueAccount(ctx, tx, AccountLength, 20)
		if err != nil || forceErr == "GenerateUniqueAccount" {
			e := fmt.Errorf("failed to generate unique account for user %s: %w", users[i].Account, err)
			return e
		}
	}

	_, err = r.InsertUsers(ctx, tx, users)
	if err != nil || forceErr == "InsertUsers" {
		e := fmt.Errorf("failed to insert users with generated accounts: %w", err)
		return e
	}

	return nil
}

// CheckTUserFieldExists 检查 t_user 表中指定字段的值是否存在
func (r *service) CheckTUserFieldExists(ctx context.Context, tx pgx.Tx, field string, value any) (bool, error) {
	z.Info("---->" + cmn.FncName())

	forceErr, _ := ctx.Value("force-error").(string)

	if field == "" {
		return false, fmt.Errorf("field name cannot be empty")
	}

	// 防止 SQL 注入：仅允许检查白名单字段
	allowedFields := map[string]bool{
		"account":       true,
		"email":         true,
		"mobile_phone":  true,
		"id_card_no":    true,
		"official_name": true,
		"id":            true,
	}
	if !allowedFields[field] {
		return false, fmt.Errorf("field '%s' is not allowed to be queried", field)
	}

	querySQL := fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM t_user WHERE %s = $1)`, field)

	var exists bool
	var err error
	if tx != nil {
		err = tx.QueryRow(ctx, querySQL, value).Scan(&exists)
	} else {
		err = r.pgxConn.QueryRow(ctx, querySQL, value).Scan(&exists)
	}

	if err != nil || forceErr == "tx.QueryRow" {
		e := fmt.Errorf("failed to check if value exists for field '%s': %w", field, err)
		z.Error(e.Error())
		return false, e
	}

	return exists, nil
}

// CheckTUserRowExists 检查 t_user 表中是否存在满足所有字段值的行
func (r *service) CheckTUserRowExists(ctx context.Context, tx pgx.Tx, fields map[string]any) (bool, *User, error) {
	z.Info("---->" + cmn.FncName())

	forceErr, _ := ctx.Value("force-error").(string)

	if len(fields) == 0 {
		return false, nil, fmt.Errorf("fields cannot be empty")
	}

	// 字段白名单，防止 SQL 注入
	allowedFields := map[string]bool{
		"account":       true,
		"email":         true,
		"mobile_phone":  true,
		"id_card_no":    true,
		"official_name": true,
		"id":            true,
	}

	// 构建 WHERE 子句和参数列表
	var whereClauses []string
	var args []any
	argIndex := 1

	emptyNullStrCount := 0
	totalFields := len(fields)

	for field, value := range fields {
		if !allowedFields[field] {
			return false, nil, fmt.Errorf("field '%s' is not allowed to be queried", field)
		}
		// 检查是否为 null.String 类型的空值
		if nullStr, ok := value.(null.String); ok && !nullStr.Valid {
			emptyNullStrCount++
			whereClauses = append(whereClauses, fmt.Sprintf("%s IS NULL", field))
			// 不添加到 args 中，因为 IS NULL 不需要参数
		} else {
			whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", field, argIndex))
			args = append(args, value)
			argIndex++
		}
	}

	// 如果所有字段都是空的 null.String，直接返回无数据
	if emptyNullStrCount == totalFields {
		return false, nil, nil
	}

	querySQL := fmt.Sprintf(
		`SELECT id FROM t_user WHERE %s LIMIT 1`,
		strings.Join(whereClauses, " AND "),
	)

	var userId null.Int
	var err error
	if tx != nil {
		err = tx.QueryRow(ctx, querySQL, args...).Scan(&userId)
	} else {
		err = r.pgxConn.QueryRow(ctx, querySQL, args...).Scan(&userId)
	}

	if err != nil || forceErr == "tx.QueryRow" {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil, nil // 没查到返回 0 和 nil
		}
		e := fmt.Errorf("failed to check if row exists: %w", err)
		z.Error(e.Error())
		return false, nil, e
	}

	if !userId.Valid || forceErr == "InvalidUserID" {
		return false, nil, nil
	}

	// 查询该用户信息
	user, _, err := r.QueryUsers(ctx, tx, 1, 1, QueryUsersFilter{ID: userId})
	if err != nil || forceErr == "QueryUsers" {
		e := fmt.Errorf("failed to query user details: %w", err)
		return false, nil, e
	}

	return true, &user[0], nil
}

func (r *service) orDefault(s null.String, def string) string {
	if s.Valid {
		return s.String
	}
	return def
}

// GenerateUniqueAccount 生成指定长度的唯一账号（由数字和小写字母组成）
func (r *service) GenerateUniqueAccount(ctx context.Context, tx pgx.Tx, length int, maxAttempts int) (string, error) {
	if length <= 0 {
		e := fmt.Errorf("length must be greater than zero")
		z.Error(e.Error())
		return "", e
	}
	if maxAttempts <= 0 {
		e := fmt.Errorf("maxAttempts must be greater than zero")
		z.Error(e.Error())
		return "", e
	}

	forceErr, _ := ctx.Value("force-error").(string)

	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"

	rand2 := rand.New(rand.NewSource(time.Now().UnixNano()))

	for attempts := 0; attempts < maxAttempts; attempts++ {
		account := make([]byte, length)
		for i := range account {
			account[i] = charset[rand2.Intn(len(charset))]
		}
		accountStr := string(account)

		// 检查是否唯一
		exist, err := r.CheckTUserFieldExists(ctx, tx, "account", accountStr)
		if err != nil || forceErr == "CheckTUserFieldExists" {
			return "", fmt.Errorf("error checking account existence: %w", err)
		}
		if !exist && forceErr != "exist" {
			return accountStr, nil
		}
	}

	e := fmt.Errorf("failed to generate unique account after %d attempts", maxAttempts)
	z.Error(e.Error())
	return "", e
}

// ValidateUserToBeInsert 验证即将被插入数据库的用户信息
// 返回 允许插入的有效用户列表 和 不允许插入的不合法用户列表 和 已存在的用户列表
// 会使用用户已有的信息（除了帐号）检索这个用户是否存在，已存在的用户会被跳过，既不会被归为有效用户，也不会被归为无效用户
func (r *service) ValidateUserToBeInsert(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
	if len(users) == 0 {
		e := fmt.Errorf("users cannot be empty")
		z.Error(e.Error())
		return []User{}, []User{}, []User{}, e
	}

	forceErr, _ := ctx.Value("force-error").(string)

	invalidUsers := make([]User, 0)  // 不允许被插入的无效用户列表
	validUsers := make([]User, 0)    // 允许被插入的有效用户列表
	existingUsers := make([]User, 0) // 已存在的用户列表

	// 构造错误信息map
	errorMessages := map[string]string{
		"account_exists":        "账号已存在",
		"mobile_phone_exists":   "手机号已存在",
		"email_exists":          "邮箱已存在",
		"id_card_no_exists":     "证件号已存在",
		"invalid_email":         "邮箱格式不正确",
		"invalid_domain":        "角色不合法",
		"empty_domain":          "角色不能为空",
		"can_not_be_superAdmin": "不允许为超级管理员角色",
		"empty_mobile_phone":    "无法检测到手机号",
		"mobile_not_e164":       "手机号格式不符合E.164标准",
		"id_card_type_invalid":  "证件类型不合法",
		"empty_id_card_type":    "证件类型不能为空",
	}

	for i := range users {

		// 用当前用户有的信息（除了帐号）检索这个用户实例是否已存在
		userExist, existUserInfo, err := r.CheckTUserRowExists(ctx, tx, map[string]any{
			"official_name": users[i].OfficialName,
			"mobile_phone":  users[i].MobilePhone,
			"email":         users[i].Email,
			"id_card_no":    users[i].IDCardNo,
		})
		if err != nil || forceErr == "CheckTUserRowExists" {
			return []User{}, []User{}, []User{}, fmt.Errorf("error checking user existence: %w", err)
		}
		if userExist {
			// 如果用户实例已存在，则跳过，不需要重复插入
			existingUsers = append(existingUsers, *existUserInfo)
			continue
		}

		// 若果用户实例不存在，则继续验证其信息是否与其他用户实例冲突

		errorMessage := make([]null.String, 0)
		errorCount := 0

		if users[i].Account != "" {
			// 检查帐号是否已存在
			exist, err := r.CheckTUserFieldExists(ctx, tx, "account", users[i].Account)
			if err != nil || forceErr == "CheckTUserFieldExists_account" {
				return []User{}, []User{}, []User{}, fmt.Errorf("error checking account existence: %w", err)
			}
			if exist {
				errorCount++
				errorMessage = append(errorMessage, null.StringFrom(errorMessages["account_exists"]))
			}
		}

		if users[i].MobilePhone.Valid {
			// 检测手机号格式是否符合E.164标准
			number := strings.TrimSpace(users[i].MobilePhone.String)
			if number == "" {
				errorCount++
				errorMessage = append(errorMessage, null.StringFrom(errorMessages["empty_mobile_phone"]))
			}

			region := strings.ToUpper(strings.TrimSpace(DefaultRegion))
			if region == "" {
				region = "CN" // 默认地区为中国
			}

			switch strings.HasPrefix(number, "+") {
			case true:
				// 如果传入的手机号有 + 前缀，则按国际格式处理
				num, err := phonenumbers.Parse(number, "")
				if err != nil {
					errorCount++
					errorMessage = append(errorMessage, null.StringFrom(errorMessages["mobile_not_e164"]))
					break
				}
				if !phonenumbers.IsValidNumber(num) {
					errorCount++
					errorMessage = append(errorMessage, null.StringFrom(errorMessages["mobile_not_e164"]))
					break
				}
			case false:
				// 如果没有 + 前缀，则按默认地区处理
				num, err := phonenumbers.Parse(number, region)
				if err != nil {
					errorCount++
					errorMessage = append(errorMessage, null.StringFrom(errorMessages["mobile_not_e164"]))
					break
				}
				if !phonenumbers.IsValidNumber(num) {
					errorCount++
					errorMessage = append(errorMessage, null.StringFrom(errorMessages["mobile_not_e164"]))
					break
				}
			}

			// 检查手机号是否已存在
			exist, err := r.CheckTUserFieldExists(ctx, tx, "mobile_phone", users[i].MobilePhone)
			if err != nil || forceErr == "CheckTUserFieldExists_mobile_phone" {
				return []User{}, []User{}, []User{}, fmt.Errorf("error checking mobile phone existence: %w", err)
			}
			if exist {
				errorCount++
				errorMessage = append(errorMessage, null.StringFrom(errorMessages["mobile_phone_exists"]))
			}
		}

		if users[i].Email.Valid {
			// 检查邮箱格式是否有效
			if !IsValidEmail(users[i].Email.String) {
				errorCount++
				errorMessage = append(errorMessage, null.StringFrom(errorMessages["invalid_email"]))
			}
			// 检查邮箱是否已存在
			exist, err := r.CheckTUserFieldExists(ctx, tx, "email", users[i].Email)
			if err != nil || forceErr == "CheckTUserFieldExists_email" {
				return []User{}, []User{}, []User{}, fmt.Errorf("error checking email existence: %w", err)
			}
			if exist {
				errorCount++
				errorMessage = append(errorMessage, null.StringFrom(errorMessages["email_exists"]))
			}
		}

		if users[i].IDCardNo.Valid {
			// 如果有证件号，则必须有证件号类型
			if !users[i].IDCardType.Valid {
				errorCount++
				errorMessage = append(errorMessage, null.StringFrom(errorMessages["empty_id_card_type"]))
			} else {
				// 检查证件号类型是否合法
				if !cmn.CheckIDCardTypeValid(users[i].IDCardType.String) {
					errorCount++
					errorMessage = append(errorMessage, null.StringFrom(errorMessages["id_card_type_invalid"]))
				}
			}

			// 检查证件号是否已存在
			exist, err := r.CheckTUserFieldExists(ctx, tx, "id_card_no", users[i].IDCardNo)
			if err != nil || forceErr == "CheckTUserFieldExists_id_card_no" {
				return []User{}, []User{}, []User{}, fmt.Errorf("error checking ID card number existence: %w", err)
			}
			if exist {
				errorCount++
				errorMessage = append(errorMessage, null.StringFrom(errorMessages["id_card_no_exists"]))
			}
		}

		if len(users[i].Domains) != 0 {
			// 检查角色是否合法
			for _, domain := range users[i].Domains {
				if !IsDomainExist(domain.String) {
					errorCount++
					errorMessage = append(errorMessage, null.StringFrom(fmt.Sprintf("%s: %s", errorMessages["invalid_domain"], domain.String)))
					break // 一旦发现一个角色不合法，就不需要继续检查其他角色
				}
				if domain.String == DomainSuperAdmin {
					// 如果角色是超级管理员，则不允许添加
					errorCount++
					errorMessage = append(errorMessage, null.StringFrom(errorMessages["can_not_be_superAdmin"]))
					break // 一旦发现是超级管理员角色，就不需要继续检查其他角色
				}
			}
		} else {
			// 如果角色列表为空，则添加错误信息
			errorCount++
			errorMessage = append(errorMessage, null.StringFrom(errorMessages["empty_domain"]))
		}

		if errorCount > 0 {
			// 如果有错误，则将用户添加到无效列表
			invalidUsers = append(invalidUsers, User{
				TUser: cmn.TUser{
					Account:      users[i].Account,
					OfficialName: users[i].OfficialName,
					MobilePhone:  users[i].MobilePhone,
					Email:        users[i].Email,
					IDCardNo:     users[i].IDCardNo,
				},
				ErrorMsg: errorMessage,
			})
		}

		if errorCount == 0 {
			// 如果没有错误，则将用户添加到有效列表
			validUsers = append(validUsers, users[i])
		}
	}

	return validUsers, invalidUsers, existingUsers, nil
}

// QueryUserCurrentRole 查询用户当前角色
// 返回值: 角色ID、角色名称、错误
func (r *service) QueryUserCurrentRole(ctx context.Context, userId null.Int) (null.Int, null.String, error) {
	if !userId.Valid {
		e := fmt.Errorf("userId is required")
		z.Error(e.Error())
		return null.Int{}, null.String{}, e
	}

	forceErr, _ := ctx.Value("force-error").(string)

	const query = `
	SELECT u.role AS role_id, d.domain AS role_name
	FROM t_user u
	JOIN t_domain d ON u.role = d.id
	WHERE u.id = $1
	`

	var roleId null.Int
	var roleName null.String

	var err error

	err = r.pgxConn.QueryRow(ctx, query, userId.Int64).Scan(&roleId, &roleName)
	if err != nil || forceErr == "QueryUserCurrentRole" {
		e := fmt.Errorf("failed to query user current role: %w", err)
		z.Error(e.Error())
		return null.Int{}, null.String{}, e
	}

	if !roleId.Valid || !roleName.Valid || forceErr == "InvalidRole" {
		e := fmt.Errorf("user %d does not have a valid role", userId.Int64)
		z.Error(e.Error())
		return null.Int{}, null.String{}, e
	}

	return roleId, roleName, nil
}
