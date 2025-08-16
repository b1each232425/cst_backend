package user_mgt

import (
	"context"

	"github.com/jackc/pgx/v5"
	"w2w.io/null"
)

// MockService 模拟 Service 接口
type MockService struct {
	GenerateUniqueAccountFunc func(ctx context.Context, tx pgx.Tx, length int, maxAttempts int) (string, error)
	ValidateUserFunc          func(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error)
	QueryUserCurrentRoleFunc  func(ctx context.Context, userId null.Int) (null.Int, null.String, error)

	user      User
	users     []User
	totalRows int64
	Exist     bool
	err       error
}

func (m *MockService) QueryUsers(ctx context.Context, tx pgx.Tx, page, pageSize int64, filter QueryUsersFilter) ([]User, int64, error) {
	if m.err != nil {
		return nil, 0, m.err
	}
	return m.users, m.totalRows, nil
}

func (m *MockService) InsertUsers(ctx context.Context, tx pgx.Tx, users []User) ([]User, error) {
	if m.err != nil {
		return []User{}, m.err
	}
	m.users = append(m.users, users...)
	m.totalRows += int64(len(users))
	return users, nil
}

func (m *MockService) InsertUsersWithAccount(ctx context.Context, tx pgx.Tx, users []User) error {
	if m.err != nil {
		return m.err
	}
	m.users = append(m.users, users...)
	m.totalRows += int64(len(users))
	return nil
}

func (m *MockService) CheckTUserFieldExists(ctx context.Context, tx pgx.Tx, field string, value any) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.Exist, nil
}

func (m *MockService) CheckTUserRowExists(ctx context.Context, tx pgx.Tx, fields map[string]any) (bool, *User, error) {
	if m.err != nil {
		return false, nil, m.err
	}
	return m.Exist, &m.user, nil
}

func (m *MockService) GenerateUniqueAccount(ctx context.Context, tx pgx.Tx, length int, maxAttempts int) (string, error) {
	if m.GenerateUniqueAccountFunc != nil {
		return m.GenerateUniqueAccountFunc(ctx, tx, length, maxAttempts)
	}
	return "", nil // 默认返回空字符串和 nil 错误
}

func (m *MockService) ValidateUserToBeInsert(ctx context.Context, tx pgx.Tx, users []User) ([]User, []User, []User, error) {
	if m.ValidateUserFunc != nil {
		return m.ValidateUserFunc(ctx, tx, users)
	}
	return nil, nil, nil, nil // 默认返回空切片和 nil 错误
}

func (m *MockService) QueryUserCurrentRole(ctx context.Context, userId null.Int) (null.Int, null.String, error) {
	if m.QueryUserCurrentRoleFunc == nil {
		return null.Int{}, null.NewString("cst.school^superAdmin", true), nil // 默认超级管理员角色
	}
	return m.QueryUserCurrentRoleFunc(ctx, userId)
}
