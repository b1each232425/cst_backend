package user_mgt

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
	"w2w.io/null"

	"strings"
	"w2w.io/cmn"
)

type Repo interface {
	QueryUsers(ctx context.Context, tx pgx.Tx, page, pageSize int64, filter QueryUsersFilter) ([]cmn.TUser, int64, error)
	InsertUsers(ctx context.Context, tx pgx.Tx, users []cmn.TUser) error
}

type repo struct {
	pgxConn *pgxpool.Pool
}

func NewRepo() Repo {
	return &repo{
		pgxConn: cmn.GetPgxConn(),
	}
}

// QueryUsers 查询用户列表
// 返回值: 用户列表、总记录数、错误
func (r *repo) QueryUsers(ctx context.Context, tx pgx.Tx, page, pageSize int64, filter QueryUsersFilter) ([]cmn.TUser, int64, error) {
	z.Info("---->" + cmn.FncName())

	forceErr, _ := ctx.Value("force-error").(string)

	if page <= 0 || pageSize <= 0 {
		e := fmt.Errorf("page and page size must be positive integers")
		z.Error(e.Error())
		return []cmn.TUser{}, 0, e
	}

	// 构建 WHERE 条件和参数
	var where []string
	var args []interface{}
	argIndex := 1

	if filter.Account.Valid {
		where = append(where, fmt.Sprintf("u.account ILIKE $%d", argIndex))
		args = append(args, "%"+filter.Account.String+"%")
		argIndex++
	}
	if filter.Name.Valid {
		where = append(where, fmt.Sprintf("u.official_name ILIKE $%d", argIndex))
		args = append(args, "%"+filter.Name.String+"%")
		argIndex++
	}
	if filter.Phone.Valid {
		where = append(where, fmt.Sprintf("u.mobile_phone ILIKE $%d", argIndex))
		args = append(args, "%"+filter.Phone.String+"%")
		argIndex++
	}
	if filter.Email.Valid {
		where = append(where, fmt.Sprintf("u.email ILIKE $%d", argIndex))
		args = append(args, "%"+filter.Email.String+"%")
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

	// 构建查询 SQL
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
		return []cmn.TUser{}, 0, e
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
		       u.id_card_no,
		       u.logon_time,
			   u.create_time,
			   u.update_time,
			   u.creator
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
		return []cmn.TUser{}, 0, e
	}
	defer rows.Close()

	var users = make([]cmn.TUser, 0, pageSize)
	for rows.Next() {
		var user cmn.TUser
		err = rows.Scan(
			&user.ID,
			&user.Account,
			&user.OfficialName,
			&user.Gender,
			&user.MobilePhone,
			&user.Email,
			&user.Category,
			&user.Status,
			&user.IDCardNo,
			&user.LogonTime,
			&user.CreateTime,
			&user.UpdateTime,
			&user.Creator,
		)
		if err != nil || forceErr == "scan" {
			e := fmt.Errorf("failed to scan user row: %w", err)
			z.Error(e.Error())
			return []cmn.TUser{}, 0, e
		}

		users = append(users, user)
	}

	if rows.Err() != nil || forceErr == "reading" {
		e := fmt.Errorf("error occurred during row iteration: %w", rows.Err())
		z.Error(e.Error())
		return []cmn.TUser{}, 0, e
	}

	return users, rowCount, nil
}

// InsertUsers 批量插入用户数据
// 必要字段: account, category, creator
func (r *repo) InsertUsers(ctx context.Context, tx pgx.Tx, users []cmn.TUser) error {
	z.Info("---->" + cmn.FncName())

	forceErr, _ := ctx.Value("force-error").(string)

	var err error

	if len(users) == 0 {
		e := fmt.Errorf("no users to insert")
		z.Error(e.Error())
		return e
	}

	// 是否是内部开启的事务
	if tx == nil {
		var newTx pgx.Tx
		newTx, err = r.pgxConn.Begin(ctx)
		if err != nil || forceErr == "pgxConn.Begin" {
			e := fmt.Errorf("failed to begin transaction: %w", err)
			z.Error(e.Error())
			return e
		}
		tx = newTx

		defer func() {
			if err != nil {
				_ = tx.Rollback(ctx)
			} else {
				err = tx.Commit(ctx)
			}
		}()
	}

	for _, user := range users {
		if err = r.validateUser(user); err != nil {
			return err
		}

		if !user.IDCardNo.Valid && !user.MobilePhone.Valid && !user.Email.Valid {
			user.Type = null.StringFrom("00") // 匿名用户
		} else {
			user.Type = null.StringFrom("02") // 注册用户
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
			create_time,
			update_time
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
		)`

		_, err = tx.Exec(ctx, insertSQL,
			user.Category,
			user.Type.String,
			user.OfficialName,
			r.orDefault(user.IDCardType, "居民身份证"),
			user.IDCardNo,
			user.Account,
			user.MobilePhone,
			user.Email,
			user.Gender,
			user.Birthday,
			user.Creator.Int64,
			r.orDefault(user.Status, "00"),
			user.Remark,
			time.Now().UnixMilli(),
			time.Now().UnixMilli(),
		)
		if err != nil || forceErr == "tx.Exec" {
			e := fmt.Errorf("failed to insert user %s: %w", user.Account, err)
			z.Error(e.Error())
			return e
		}
	}

	return nil
}

func (r *repo) validateUser(user cmn.TUser) error {
	if user.Account == "" {
		return fmt.Errorf("user account is required")
	}
	if user.Category == "" {
		return fmt.Errorf("user category is required")
	}
	if !user.Creator.Valid {
		return fmt.Errorf("user creator is required")
	}
	return nil
}

func (r *repo) orDefault(s null.String, def string) string {
	if s.Valid {
		return s.String
	}
	return def
}
