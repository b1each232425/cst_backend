package user_mgt

import (
	"context"
	"github.com/jackc/pgx/v5"
	"w2w.io/cmn"
)

// MockRepo 模拟Repo接口
type MockRepo struct {
	users     []cmn.TUser
	totalRows int64
	err       error
}

func (m *MockRepo) QueryUsers(ctx context.Context, tx pgx.Tx, page, pageSize int64, filter QueryUsersFilter) ([]cmn.TUser, int64, error) {
	if m.err != nil {
		return nil, 0, m.err
	}
	return m.users, m.totalRows, nil
}

func (m *MockRepo) InsertUsers(ctx context.Context, tx pgx.Tx, users []cmn.TUser) error {
	if m.err != nil {
		return m.err
	}
	m.users = append(m.users, users...)
	m.totalRows += int64(len(users))
	return nil
}
