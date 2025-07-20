package user_mgt

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"strings"
	"w2w.io/cmn"
)

type Repo interface {
	QueryUsers(ctx context.Context, tx pgx.Tx, page, pageSize int64, filter QueryUsersFilter) ([]cmn.TUser, int64, error)
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
