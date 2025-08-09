package user_mgt

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"w2w.io/cmn"
	"w2w.io/null"
)

// TestService_QueryUsers 测试QueryUsers方法
func TestService_QueryUsers(t *testing.T) {
	// 创建真实的repo实例
	repo := NewService()

	tests := []struct {
		name           string
		ctx            context.Context
		page           int64
		pageSize       int64
		filter         QueryUsersFilter
		wantUsersCount int
		wantErr        bool
		desc           string
	}{
		{
			name:           "基本查询测试",
			ctx:            context.WithValue(context.Background(), "force-error", ""),
			page:           1,
			pageSize:       10,
			filter:         QueryUsersFilter{},
			wantUsersCount: 10,
			wantErr:        false,
			desc:           "测试基本的分页查询功能",
		},
		{
			name:           "无效页码测试",
			ctx:            context.WithValue(context.Background(), "force-error", ""),
			page:           0,
			pageSize:       10,
			filter:         QueryUsersFilter{},
			wantUsersCount: 0,
			wantErr:        true,
			desc:           "测试页码为0时应该返回错误",
		},
		{
			name:           "无效页面大小测试",
			ctx:            context.WithValue(context.Background(), "force-error", ""),
			page:           1,
			pageSize:       -1,
			filter:         QueryUsersFilter{},
			wantUsersCount: 0,
			wantErr:        true,
			desc:           "测试页面大小为负数时应该返回错误",
		},
		{
			name:     "ID过滤查询测试",
			ctx:      context.WithValue(context.Background(), "force-error", ""),
			page:     1,
			pageSize: 10,
			filter: QueryUsersFilter{
				ID: null.NewInt(1, true),
			},
			wantUsersCount: 1,
			wantErr:        false,
			desc:           "测试按ID过滤查询",
		},
		{
			name:     "账号过滤查询测试",
			ctx:      context.WithValue(context.Background(), "force-error", ""),
			page:     1,
			pageSize: 10,
			filter: QueryUsersFilter{
				FuzzyCondition: null.NewString("admin", true),
			},
			wantUsersCount: 2,
			wantErr:        false,
			desc:           "测试按账号过滤查询",
		},
		{
			name:     "姓名过滤查询测试",
			ctx:      context.WithValue(context.Background(), "force-error", ""),
			page:     1,
			pageSize: 10,
			filter: QueryUsersFilter{
				FuzzyCondition: null.NewString("测试", true),
			},
			wantUsersCount: 0,
			wantErr:        false,
			desc:           "测试按姓名过滤查询",
		},
		{
			name:     "手机号过滤查询测试",
			ctx:      context.WithValue(context.Background(), "force-error", ""),
			page:     1,
			pageSize: 10,
			filter: QueryUsersFilter{
				FuzzyCondition: null.NewString("138", true),
			},
			wantUsersCount: 1,
			wantErr:        false,
			desc:           "测试按手机号过滤查询",
		},
		{
			name:     "邮箱过滤查询测试",
			ctx:      context.WithValue(context.Background(), "force-error", ""),
			page:     1,
			pageSize: 10,
			filter: QueryUsersFilter{
				FuzzyCondition: null.NewString("@example.com", true),
			},
			wantUsersCount: 6,
			wantErr:        false,
			desc:           "测试按邮箱过滤查询",
		},
		{
			name:     "性别过滤查询测试",
			ctx:      context.WithValue(context.Background(), "force-error", ""),
			page:     1,
			pageSize: 10,
			filter: QueryUsersFilter{
				Gender: null.NewString("M", true),
			},
			wantUsersCount: 6,
			wantErr:        false,
			desc:           "测试按性别过滤查询",
		},
		{
			name:     "状态过滤查询测试",
			ctx:      context.WithValue(context.Background(), "force-error", ""),
			page:     1,
			pageSize: 10,
			filter: QueryUsersFilter{
				Status: null.NewString("00", true),
			},
			wantUsersCount: 8,
			wantErr:        false,
			desc:           "测试按状态过滤查询",
		},
		{
			name:     "创建时间过滤查询测试",
			ctx:      context.WithValue(context.Background(), "force-error", ""),
			page:     1,
			pageSize: 10,
			filter: QueryUsersFilter{
				CreateTime: null.NewInt(time.Now().AddDate(0, 0, -30).Unix(), true), // 30天前
			},
			wantUsersCount: 10,
			wantErr:        false,
			desc:           "测试按创建时间过滤查询",
		},
		{
			name:     "角色过滤查询测试",
			ctx:      context.WithValue(context.Background(), "force-error", ""),
			page:     1,
			pageSize: 10,
			filter: QueryUsersFilter{
				Domain: null.NewString("cst.school^student", true),
			},
			wantUsersCount: 1,
			wantErr:        false,
			desc:           "测试按性别过滤查询",
		},
		{
			name:     "多条件组合过滤查询测试",
			ctx:      context.WithValue(context.Background(), "force-error", ""),
			page:     1,
			pageSize: 5,
			filter: QueryUsersFilter{
				Gender: null.NewString("M", true),
				Status: null.NewString("00", true),
			},
			wantUsersCount: 5,
			wantErr:        false,
			desc:           "测试多个过滤条件组合查询",
		},
		{
			name:           "大页面查询测试",
			ctx:            context.WithValue(context.Background(), "force-error", ""),
			page:           1,
			pageSize:       100,
			filter:         QueryUsersFilter{},
			wantUsersCount: 12,
			wantErr:        false,
			desc:           "测试大页面大小的查询",
		},
		{
			name:           "第二页查询测试",
			ctx:            context.WithValue(context.Background(), "force-error", ""),
			page:           2,
			pageSize:       10,
			filter:         QueryUsersFilter{},
			wantUsersCount: 2,
			wantErr:        false,
			desc:           "测试第二页数据查询",
		},
		{
			name:     "不存在数据的过滤查询测试",
			ctx:      context.WithValue(context.Background(), "force-error", ""),
			page:     1,
			pageSize: 10,
			filter: QueryUsersFilter{
				FuzzyCondition: null.NewString("nonexistent_user_12345", true),
			},
			wantUsersCount: 0,
			wantErr:        false,
			desc:           "测试查询不存在的数据应该返回空结果",
		},
		{
			name:     "触发查询总记录数报错",
			ctx:      context.WithValue(context.Background(), "force-error", "count"),
			page:     1,
			pageSize: 10,
			filter: QueryUsersFilter{
				FuzzyCondition: null.NewString("nonexistent_user_12345", true),
			},
			wantUsersCount: 0,
			wantErr:        true,
			desc:           "测试触发查询总记录数报错",
		},
		{
			name:     "触发查询用户数据报错",
			ctx:      context.WithValue(context.Background(), "force-error", "query"),
			page:     1,
			pageSize: 10,
			filter: QueryUsersFilter{
				FuzzyCondition: null.NewString("nonexistent_user_12345", true),
			},
			wantUsersCount: 0,
			wantErr:        true,
			desc:           "测试触发查询用户数据报错",
		},
		{
			name:           "触发扫描行数据报错",
			ctx:            context.WithValue(context.Background(), "force-error", "scan"),
			page:           1,
			pageSize:       10,
			filter:         QueryUsersFilter{},
			wantUsersCount: 0,
			wantErr:        true,
			desc:           "测试触发扫描行数据报错",
		},
		{
			name:     "触发读取行数据报错",
			ctx:      context.WithValue(context.Background(), "force-error", "reading"),
			page:     1,
			pageSize: 10,
			filter: QueryUsersFilter{
				FuzzyCondition: null.NewString("nonexistent_user_12345", true),
			},
			wantUsersCount: 0,
			wantErr:        true,
			desc:           "测试触发读取行数据报错",
		},
		{
			name:           "触发json.Unmarshal报错",
			ctx:            context.WithValue(context.Background(), "force-error", "json.Unmarshal"),
			page:           1,
			pageSize:       10,
			filter:         QueryUsersFilter{},
			wantUsersCount: 10,
			wantErr:        false,
			desc:           "测试触发json.Unmarshal报错",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("开始测试: %s", tt.desc)

			// 执行查询
			users, totalCount, err := repo.QueryUsers(tt.ctx, nil, tt.page, tt.pageSize, tt.filter)

			// 验证错误
			if (err != nil) != tt.wantErr {
				t.Errorf("QueryUsers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// 如果期望有错误，则不需要进一步验证
			if tt.wantErr {
				t.Logf("预期错误已正确返回: %v", err)
				return
			}

			// 验证返回结果的基本属性
			if users == nil {
				t.Error("QueryUsers() 返回的用户列表不应该为nil")
				return
			}

			if totalCount < 0 {
				t.Errorf("QueryUsers() 返回的总数不应该为负数: %d", totalCount)
			}

			if len(users) != tt.wantUsersCount {
				t.Errorf("QueryUsers() 返回的总用户数 %d 不符合预期 %d", len(users), tt.wantUsersCount)
			}

			// 验证分页逻辑
			if int64(len(users)) > tt.pageSize {
				t.Errorf("QueryUsers() 返回的用户数量 %d 超过了页面大小 %d", len(users), tt.pageSize)
			}

			// 记录查询结果
			t.Logf("查询结果: 用户数量=%d, 总记录数=%d", len(users), totalCount)

			// 验证返回的用户数据结构
			for i, user := range users {
				if !user.ID.Valid {
					t.Errorf("用户[%d]的ID字段无效", i)
				}
				if user.Account == "" {
					t.Errorf("用户[%d]的Account字段为空", i)
				}
				if !user.Status.Valid {
					t.Errorf("用户[%d]的Status字段无效", i)
				}
			}

			// 验证过滤条件是否生效（仅对有过滤条件的测试）
			if tt.filter.FuzzyCondition.Valid {
				for i, user := range users {
					fc := tt.filter.FuzzyCondition.String
					matched := false

					if containsIgnoreCase(user.Account, fc) {
						matched = true
					}
					if containsIgnoreCase(user.OfficialName.String, fc) {
						matched = true
					}
					if containsIgnoreCase(user.MobilePhone.String, fc) {
						matched = true
					}
					if containsIgnoreCase(user.Email.String, fc) {
						matched = true
					}
					if containsIgnoreCase(user.IDCardNo.String, fc) {
						matched = true
					}

					if !matched {
						t.Errorf("用户[%d]的所有字段都不包含过滤条件 '%s'", i, fc)
					}
				}
			}

			if tt.filter.Gender.Valid {
				for i, user := range users {
					if user.Gender.Valid && user.Gender.String != tt.filter.Gender.String {
						t.Errorf("用户[%d]的Gender '%s' 不匹配过滤条件 '%s'", i, user.Gender.String, tt.filter.Gender.String)
					}
				}
			}

			if tt.filter.Status.Valid {
				for i, user := range users {
					if user.Status.Valid && user.Status.String != tt.filter.Status.String {
						t.Errorf("用户[%d]的Status '%s' 不匹配过滤条件 '%s'", i, user.Status.String, tt.filter.Status.String)
					}
				}
			}
		})
	}
}

// TestService_QueryUserCurrentRole 测试QueryUserCurrentRole方法
func TestService_QueryUserCurrentRole(t *testing.T) {
	// 创建真实的repo实例
	repo := NewService()

	tests := []struct {
		name         string
		ctx          context.Context
		userId       null.Int
		wantRoleId   null.Int
		wantRoleName null.String
		wantErr      bool
		desc         string
	}{
		{
			name:         "查询有效用户角色",
			ctx:          context.Background(),
			userId:       null.IntFrom(2), // 假设用户ID为1存在且有角色
			wantRoleId:   null.NewInt(2000, true),
			wantRoleName: null.NewString("cst.school^superAdmin", true),
			wantErr:      false,
			desc:         "测试查询存在且有角色的用户",
		},
		{
			name:         "查询得到的用户角色无效",
			ctx:          context.WithValue(context.Background(), "force-error", "InvalidRole"),
			userId:       null.IntFrom(2), // 假设用户ID为1存在且有角色
			wantRoleId:   null.NewInt(2000, true),
			wantRoleName: null.NewString("cst.school^superAdmin", true),
			wantErr:      true,
			desc:         "测试查询存在且有角色的用户",
		},
		{
			name:         "用户ID无效",
			ctx:          context.Background(),
			userId:       null.NewInt(0, false),
			wantRoleId:   null.NewInt(0, false),
			wantRoleName: null.NewString("", false),
			wantErr:      true,
			desc:         "测试用户ID无效时应该返回错误",
		},
		{
			name:         "用户不存在",
			ctx:          context.Background(),
			userId:       null.IntFrom(999999), // 假设这个用户ID不存在
			wantRoleId:   null.NewInt(0, false),
			wantRoleName: null.NewString("", false),
			wantErr:      true,
			desc:         "测试查询不存在的用户应该返回错误",
		},
		{
			name:         "强制查询错误",
			ctx:          context.WithValue(context.Background(), "force-error", "QueryUserCurrentRole"),
			userId:       null.IntFrom(1),
			wantRoleId:   null.NewInt(0, false),
			wantRoleName: null.NewString("", false),
			wantErr:      true,
			desc:         "测试强制查询错误",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("开始测试: %s", tt.desc)

			// 执行查询
			roleId, roleName, err := repo.QueryUserCurrentRole(tt.ctx, tt.userId)

			// 验证错误
			if (err != nil) != tt.wantErr {
				t.Errorf("QueryUserCurrentRole() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// 如果期望有错误，则不需要进一步验证
			if tt.wantErr {
				t.Logf("预期错误已正确返回: %v", err)
				return
			}

			// 验证返回结果
			if roleId.Valid != tt.wantRoleId.Valid {
				t.Errorf("QueryUserCurrentRole() roleId.Valid = %v, want %v", roleId.Valid, tt.wantRoleId.Valid)
			}

			if roleName.Valid != tt.wantRoleName.Valid {
				t.Errorf("QueryUserCurrentRole() roleName.Valid = %v, want %v", roleName.Valid, tt.wantRoleName.Valid)
			}

			// 如果角色ID和角色名称都有效，验证它们不为空
			if roleId.Valid && roleId.Int64 <= 0 {
				t.Errorf("QueryUserCurrentRole() roleId should be positive, got %d", roleId.Int64)
			}

			if roleName.Valid && roleName.String == "" {
				t.Error("QueryUserCurrentRole() roleName should not be empty when valid")
			}

			t.Logf("查询结果: roleId=%v, roleName=%v", roleId, roleName)
		})
	}
}

// TestService_QueryUserCurrentRole_EdgeCases 边界情况测试
func TestService_QueryUserCurrentRole_EdgeCases(t *testing.T) {
	repo := NewService()
	ctx := context.Background()

	tests := []struct {
		name    string
		userId  null.Int
		wantErr bool
		desc    string
	}{
		{
			name:    "极大用户ID测试",
			userId:  null.IntFrom(9223372036854775807), // int64最大值
			wantErr: true,
			desc:    "测试极大用户ID应该返回错误",
		},
		{
			name:    "负数用户ID测试",
			userId:  null.IntFrom(-1),
			wantErr: true,
			desc:    "测试负数用户ID应该返回错误",
		},
		{
			name:    "零用户ID测试",
			userId:  null.IntFrom(0),
			wantErr: true,
			desc:    "测试零用户ID应该返回错误",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("开始测试: %s", tt.desc)

			roleId, roleName, err := repo.QueryUserCurrentRole(ctx, tt.userId)

			if (err != nil) != tt.wantErr {
				t.Errorf("QueryUserCurrentRole() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				t.Logf("边界测试结果: roleId=%v, roleName=%v", roleId, roleName)
			} else {
				t.Logf("预期错误已正确返回: %v", err)
			}
		})
	}
}

// BenchmarkQueryUserCurrentRole 基准测试
func BenchmarkQueryUserCurrentRole(b *testing.B) {
	repo := NewService()
	ctx := context.Background()
	userId := null.IntFrom(1) // 假设用户ID为1存在

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := repo.QueryUserCurrentRole(ctx, userId)
		if err != nil {
			// 在基准测试中，如果用户不存在是正常的，不应该停止测试
			// b.Errorf("QueryUserCurrentRole() error = %v", err)
		}
	}
}

// TestService_QueryUsers_WithTransaction 测试带事务的查询
func TestService_QueryUsers_WithTransaction(t *testing.T) {
	ctx := context.Background()

	// 获取数据库连接
	pgxConn := cmn.GetPgxConn()
	if pgxConn == nil {
		t.Skip("数据库连接不可用，跳过事务测试")
		return
	}

	// 开始事务
	tx, err := pgxConn.Begin(ctx)
	if err != nil {
		t.Fatalf("开始事务失败: %v", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Logf("回滚事务失败: %v", err)
		}
	}()

	repo := NewService()

	tests := []struct {
		name     string
		page     int64
		pageSize int64
		filter   QueryUsersFilter
		wantErr  bool
		desc     string
	}{
		{
			name:     "事务中的基本查询测试",
			page:     1,
			pageSize: 10,
			filter:   QueryUsersFilter{},
			wantErr:  false,
			desc:     "测试在事务中执行基本查询",
		},
		{
			name:     "事务中的过滤查询测试",
			page:     1,
			pageSize: 5,
			filter: QueryUsersFilter{
				Status: null.NewString("00", true),
			},
			wantErr: false,
			desc:    "测试在事务中执行过滤查询",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("开始测试: %s", tt.desc)

			// 执行查询
			users, totalCount, err := repo.QueryUsers(ctx, tx, tt.page, tt.pageSize, tt.filter)

			// 验证错误
			if (err != nil) != tt.wantErr {
				t.Errorf("QueryUsers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if users == nil {
					t.Error("QueryUsers() 返回的用户列表不应该为nil")
				}
				if totalCount < 0 {
					t.Errorf("QueryUsers() 返回的总数不应该为负数: %d", totalCount)
				}
				t.Logf("事务查询结果: 用户数量=%d, 总记录数=%d", len(users), totalCount)
			}
		})
	}
}

// TestService_QueryUsers_Performance 性能测试
func TestService_QueryUsers_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过性能测试")
	}

	repo := NewService()
	ctx := context.Background()

	// 测试不同页面大小的性能
	pageSizes := []int64{10, 50, 100, 500}

	for _, pageSize := range pageSizes {
		t.Run(fmt.Sprintf("页面大小_%d", pageSize), func(t *testing.T) {
			start := time.Now()

			users, totalCount, err := repo.QueryUsers(ctx, nil, 1, pageSize, QueryUsersFilter{})
			if err != nil {
				t.Errorf("QueryUsers() error = %v", err)
				return
			}

			duration := time.Since(start)
			t.Logf("页面大小 %d: 查询时间=%v, 用户数量=%d, 总记录数=%d",
				pageSize, duration, len(users), totalCount)

			// 性能基准：查询时间不应超过5秒
			if duration > 5*time.Second {
				t.Errorf("查询时间过长: %v", duration)
			}
		})
	}
}

// TestService_QueryUsers_EdgeCases 边界情况测试
func TestService_QueryUsers_EdgeCases(t *testing.T) {
	repo := NewService()
	ctx := context.Background()

	tests := []struct {
		name     string
		page     int64
		pageSize int64
		filter   QueryUsersFilter
		wantErr  bool
		desc     string
	}{
		{
			name:     "极大页码测试",
			page:     999999,
			pageSize: 10,
			filter:   QueryUsersFilter{},
			wantErr:  false,
			desc:     "测试极大页码应该返回空结果",
		},
		{
			name:     "页面大小为1测试",
			page:     1,
			pageSize: 1,
			filter:   QueryUsersFilter{},
			wantErr:  false,
			desc:     "测试最小页面大小",
		},
		{
			name:     "特殊字符过滤测试",
			page:     1,
			pageSize: 10,
			filter: QueryUsersFilter{
				FuzzyCondition: null.NewString("test'user", true), // 包含单引号
			},
			wantErr: false,
			desc:    "测试包含特殊字符的过滤条件",
		},
		{
			name:     "SQL注入防护测试",
			page:     1,
			pageSize: 10,
			filter: QueryUsersFilter{
				FuzzyCondition: null.NewString("'; DROP TABLE t_user; --", true),
			},
			wantErr: false,
			desc:    "测试SQL注入防护",
		},
		{
			name:     "Unicode字符过滤测试",
			page:     1,
			pageSize: 10,
			filter: QueryUsersFilter{
				FuzzyCondition: null.NewString("测试用户🎉", true),
			},
			wantErr: false,
			desc:    "测试Unicode字符过滤",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("开始测试: %s", tt.desc)

			users, totalCount, err := repo.QueryUsers(ctx, nil, tt.page, tt.pageSize, tt.filter)

			if (err != nil) != tt.wantErr {
				t.Errorf("QueryUsers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if users == nil {
					t.Error("QueryUsers() 返回的用户列表不应该为nil")
				}
				if totalCount < 0 {
					t.Errorf("QueryUsers() 返回的总数不应该为负数: %d", totalCount)
				}
				t.Logf("边界测试结果: 用户数量=%d, 总记录数=%d", len(users), totalCount)
			}
		})
	}
}

// containsIgnoreCase 检查字符串是否包含子字符串（忽略大小写）
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			strings.Contains(strings.ToLower(s), strings.ToLower(substr)))
}

// BenchmarkQueryUsers 基准测试
func BenchmarkQueryUsers(b *testing.B) {
	repo := NewService()
	ctx := context.Background()
	filter := QueryUsersFilter{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := repo.QueryUsers(ctx, nil, 1, 10, filter)
		if err != nil {
			b.Errorf("QueryUsers() error = %v", err)
		}
	}
}

// BenchmarkQueryUsersWithFilter 带过滤条件的基准测试
func BenchmarkQueryUsersWithFilter(b *testing.B) {
	repo := NewService()
	ctx := context.Background()
	filter := QueryUsersFilter{
		Status: null.NewString("00", true),
		Gender: null.NewString("M", true),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := repo.QueryUsers(ctx, nil, 1, 10, filter)
		if err != nil {
			b.Errorf("QueryUsers() error = %v", err)
		}
	}
}

// TestService_InsertUsers 测试InsertUsers方法
func TestService_InsertUsers(t *testing.T) {
	// 创建真实的repo实例
	repo := NewService()

	tests := []struct {
		name    string
		ctx     context.Context
		users   []User
		wantErr bool
		desc    string
	}{
		{
			name: "成功插入单个用户",
			ctx:  context.Background(),
			users: []User{
				{
					TUser: cmn.TUser{
						Account:      fmt.Sprintf("test_user_%d", time.Now().UnixNano()),
						Category:     "normal",
						OfficialName: null.NewString("测试用户", true),
						Gender:       null.NewString("M", true),
						MobilePhone:  null.NewString("13800138000", true),
						Email:        null.NewString("test1@example.com", true),
						Creator:      null.NewInt(1, true),
						Status:       null.NewString("00", true),
						Remark:       null.NewString("test", true),
					},
					Domains: []null.String{
						null.NewString("cst.school^teacher", true),
						null.NewString("cst.school^admin", true),
					},
				},
			},
			wantErr: false,
			desc:    "测试成功插入单个用户数据",
		},
		{
			name: "成功插入多个用户",
			ctx:  context.Background(),
			users: []User{
				{
					TUser: cmn.TUser{
						Account:      fmt.Sprintf("batch_user_1_%d", time.Now().UnixNano()),
						Category:     "vip",
						OfficialName: null.NewString("批量用户1", true),
						Gender:       null.NewString("F", true),
						Creator:      null.NewInt(1, true),
						Remark:       null.NewString("test", true),
					},
					Domains: []null.String{
						null.NewString("cst.school^teacher", true),
						null.NewString("cst.school^admin", true),
					},
				},
				{
					TUser: cmn.TUser{
						Account:      fmt.Sprintf("batch_user_2_%d", time.Now().UnixNano()),
						Category:     "normal",
						OfficialName: null.NewString("批量用户2", true),
						Gender:       null.NewString("M", true),
						IDCardNo:     null.NewString("110101199001011234", true),
						Creator:      null.NewInt(1, true),
						Remark:       null.NewString("test", true),
					},
				},
			},
			wantErr: false,
			desc:    "测试成功插入多个用户数据",
		},
		{
			name:    "空用户列表",
			ctx:     context.Background(),
			users:   []User{},
			wantErr: true,
			desc:    "测试空用户列表应该返回错误",
		},
		{
			name: "缺少必要字段Account",
			ctx:  context.Background(),
			users: []User{
				{
					TUser: cmn.TUser{
						// Account 字段为空
						Category: "normal",
						Creator:  null.NewInt(1, true),
						Remark:   null.NewString("test", true),
					},
				},
			},
			wantErr: true,
			desc:    "测试缺少Account字段应该返回验证错误",
		},
		{
			name: "缺少必要字段Category",
			ctx:  context.Background(),
			users: []User{
				{
					TUser: cmn.TUser{
						Account: fmt.Sprintf("invalid_user_%d", time.Now().UnixNano()),
						// Category 字段为空
						Creator: null.NewInt(1, true),
						Remark:  null.NewString("test", true),
					},
				},
			},
			wantErr: true,
			desc:    "测试缺少Category字段应该返回验证错误",
		},
		{
			name: "缺少必要字段Account",
			ctx:  context.Background(),
			users: []User{
				{
					TUser: cmn.TUser{
						Remark: null.NewString("test", true),
					},
				},
			},
			wantErr: true,
			desc:    "测试缺少Creator字段应该返回验证错误",
		},
		{
			name: "匿名用户类型测试",
			ctx:  context.Background(),
			users: []User{
				{
					TUser: cmn.TUser{
						Account:      fmt.Sprintf("anonymous_user_%d", time.Now().UnixNano()),
						Category:     "anonymous",
						OfficialName: null.NewString("匿名用户", true),
						// 没有身份证、手机号、邮箱，应该被设置为匿名用户类型
						Creator: null.NewInt(1, true),
						Remark:  null.NewString("test", true),
					},
				},
			},
			wantErr: false,
			desc:    "测试匿名用户类型自动设置",
		},
		{
			name: "注册用户类型测试",
			ctx:  context.Background(),
			users: []User{
				{
					TUser: cmn.TUser{
						Account:      fmt.Sprintf("registered_user_%d", time.Now().UnixNano()),
						Category:     "registered",
						OfficialName: null.NewString("注册用户", true),
						MobilePhone:  null.NewString("13900139000", true),
						// 有手机号，应该被设置为注册用户类型
						Creator: null.NewInt(1, true),
						Remark:  null.NewString("test", true),
					},
				},
			},
			wantErr: false,
			desc:    "测试注册用户类型自动设置",
		},
		{
			name: "包含特殊字符的用户数据",
			ctx:  context.Background(),
			users: []User{
				{
					TUser: cmn.TUser{
						Account:      fmt.Sprintf("special_user_%d", time.Now().UnixNano()),
						Category:     "special",
						OfficialName: null.NewString("特殊字符用户@#$%^&*()", true),
						Email:        null.NewString("special+test@example.com", true),
						Creator:      null.NewInt(1, true),
						Remark:       null.NewString("test", true),
					},
				},
			},
			wantErr: false,
			desc:    "测试包含特殊字符的用户数据插入",
		},
		{
			name: "Unicode字符用户数据",
			ctx:  context.Background(),
			users: []User{
				{
					TUser: cmn.TUser{
						Account:      fmt.Sprintf("unicode_用户_%d", time.Now().UnixNano()),
						Category:     "unicode",
						OfficialName: null.NewString("张三李四王五赵六🎉", true),
						Gender:       null.NewString("M", true),
						Creator:      null.NewInt(1, true),
						Remark:       null.NewString("test", true),
					},
				},
			},
			wantErr: false,
			desc:    "测试Unicode字符用户数据插入",
		},
		{
			name: "强制执行SQL错误",
			ctx:  context.WithValue(context.Background(), "force-error", "Exec"),
			users: []User{
				{
					TUser: cmn.TUser{
						Account:  fmt.Sprintf("error_user_%d", time.Now().UnixNano()),
						Category: "normal",
						Creator:  null.NewInt(1, true),
						Remark:   null.NewString("test", true),
					},
				},
			},
			wantErr: true,
			desc:    "测试强制执行SQL错误",
		},
		{
			name: "查询用户ID错误",
			ctx:  context.WithValue(context.Background(), "force-error", "QueryUserID"),
			users: []User{
				{
					TUser: cmn.TUser{
						Account:      fmt.Sprintf("batch_user_1_%d", time.Now().UnixNano()),
						Category:     "vip",
						OfficialName: null.NewString("批量用户1", true),
						Gender:       null.NewString("F", true),
						Creator:      null.NewInt(1, true),
						Remark:       null.NewString("test", true),
					},
				},
			},
			wantErr: true,
			desc:    "测试查询用户ID错误",
		},
		{
			name: "插入角色错误",
			ctx:  context.WithValue(context.Background(), "force-error", "InsertUserDomain"),
			users: []User{
				{
					TUser: cmn.TUser{
						Account:      fmt.Sprintf("batch_user_1_%d", time.Now().UnixNano()),
						Category:     "vip",
						OfficialName: null.NewString("批量用户1", true),
						Gender:       null.NewString("F", true),
						Creator:      null.NewInt(1, true),
						Remark:       null.NewString("test", true),
					},
					Domains: []null.String{
						null.NewString("cst.school^teacher", true),
						null.NewString("cst.school^admin", true),
					},
				},
			},
			wantErr: true,
			desc:    "测试插入角色错误",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("开始测试: %s", tt.desc)

			// 执行插入操作
			err := repo.InsertUsers(tt.ctx, nil, tt.users)

			// 验证错误
			if (err != nil) != tt.wantErr {
				t.Errorf("InsertUsers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// 如果期望有错误，则不需要进一步验证
			if tt.wantErr {
				t.Logf("预期错误已正确返回: %v", err)
				return
			}

			// 验证插入成功的情况
			if err == nil && len(tt.users) > 0 {
				t.Logf("成功插入 %d 个用户", len(tt.users))

				// 验证插入的数据是否可以查询到（可选验证）
				for _, user := range tt.users {
					// 查询刚插入的用户
					filter := QueryUsersFilter{
						FuzzyCondition: null.NewString(user.Account, true),
					}
					users, _, queryErr := repo.QueryUsers(context.Background(), nil, 1, 1, filter)
					if queryErr != nil {
						t.Logf("查询插入的用户时出错: %v", queryErr)
						continue
					}
					if len(users) > 0 {
						t.Logf("成功查询到插入的用户: %s", users[0].Account)
						// 验证用户类型是否正确设置
						if users[0].Type.Valid {
							if !user.IDCardNo.Valid && !user.MobilePhone.Valid && !user.Email.Valid {
								// 应该是匿名用户
								if users[0].Type.String != "00" {
									t.Errorf("匿名用户类型设置错误，期望 '00'，实际 '%s'", users[0].Type.String)
								}
							} else {
								// 应该是注册用户
								if users[0].Type.String != "02" {
									t.Errorf("注册用户类型设置错误，期望 '02'，实际 '%s'", users[0].Type.String)
								}
							}
						}
					}
				}
			}
		})
	}
}

// TestService_InsertUsers_WithTransaction 测试带事务的插入操作
func TestService_InsertUsers_WithTransaction(t *testing.T) {
	ctx := context.Background()

	// 获取数据库连接
	pgxConn := cmn.GetPgxConn()
	if pgxConn == nil {
		t.Skip("数据库连接不可用，跳过事务测试")
		return
	}

	// 开始事务
	tx, err := pgxConn.Begin(ctx)
	if err != nil {
		t.Fatalf("开始事务失败: %v", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Logf("回滚事务失败: %v", err)
		}
	}()

	repo := NewService()

	tests := []struct {
		name    string
		users   []User
		wantErr bool
		desc    string
	}{
		{
			name: "事务中成功插入用户",
			users: []User{
				{
					TUser: cmn.TUser{
						Account:      fmt.Sprintf("tx_user_%d", time.Now().UnixNano()),
						Category:     "transaction",
						OfficialName: null.NewString("事务用户", true),
						Creator:      null.NewInt(1, true),
						Remark:       null.NewString("test", true),
					},
					Domains: []null.String{
						null.NewString("cst.school^teacher", true),
						null.NewString("cst.school^admin", true),
					},
				},
			},
			wantErr: false,
			desc:    "测试在事务中成功插入用户",
		},
		{
			name: "事务中插入多个用户",
			users: []User{
				{
					TUser: cmn.TUser{
						Account:  fmt.Sprintf("tx_batch_1_%d", time.Now().UnixNano()),
						Category: "batch_tx",
						Creator:  null.NewInt(1, true),
						Remark:   null.NewString("test", true),
					},
				},
				{
					TUser: cmn.TUser{
						Account:  fmt.Sprintf("tx_batch_2_%d", time.Now().UnixNano()),
						Category: "batch_tx",
						Creator:  null.NewInt(1, true),
						Remark:   null.NewString("test", true),
					},
				},
			},
			wantErr: false,
			desc:    "测试在事务中插入多个用户",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("开始测试: %s", tt.desc)

			// 执行插入操作
			err := repo.InsertUsers(ctx, tx, tt.users)

			// 验证错误
			if (err != nil) != tt.wantErr {
				t.Errorf("InsertUsers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				t.Logf("事务中成功插入 %d 个用户", len(tt.users))
			}
		})
	}
}

// TestService_InsertUsers_Performance 性能测试
func TestService_InsertUsers_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过性能测试")
	}

	repo := NewService()
	ctx := context.Background()

	// 测试不同批次大小的性能
	batchSizes := []int{1, 10, 50, 100}

	for _, batchSize := range batchSizes {
		t.Run(fmt.Sprintf("批次大小_%d", batchSize), func(t *testing.T) {
			// 准备测试数据
			users := make([]User, batchSize)
			for i := 0; i < batchSize; i++ {
				users[i] = User{
					TUser: cmn.TUser{
						Account:      fmt.Sprintf("perf_user_%d_%d", batchSize, time.Now().UnixNano()+int64(i)),
						Category:     "performance",
						OfficialName: null.NewString(fmt.Sprintf("性能测试用户%d", i+1), true),
						Creator:      null.NewInt(1, true),
						Remark:       null.NewString("test", true),
					},
				}
			}

			start := time.Now()
			err := repo.InsertUsers(ctx, nil, users)
			duration := time.Since(start)

			if err != nil {
				t.Errorf("InsertUsers() error = %v", err)
				return
			}

			t.Logf("批次大小 %d: 插入时间=%v, 平均每个用户=%v",
				batchSize, duration, duration/time.Duration(batchSize))

			// 性能基准：插入时间不应超过10秒
			if duration > 10*time.Second {
				t.Errorf("插入时间过长: %v", duration)
			}
		})
	}
}

// BenchmarkInsertUsers 基准测试
func BenchmarkInsertUsers(b *testing.B) {
	repo := NewService()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		users := []User{
			{
				TUser: cmn.TUser{
					Account:  fmt.Sprintf("bench_user_%d_%d", i, time.Now().UnixNano()),
					Category: "benchmark",
					Creator:  null.NewInt(1, true),
					Remark:   null.NewString("test", true),
				},
			},
		}
		err := repo.InsertUsers(ctx, nil, users)
		if err != nil {
			b.Errorf("InsertUsers() error = %v", err)
		}
	}
}

// BenchmarkInsertUsersBatch 批量插入基准测试
func BenchmarkInsertUsersBatch(b *testing.B) {
	repo := NewService()
	ctx := context.Background()
	batchSize := 10

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		users := make([]User, batchSize)
		for j := 0; j < batchSize; j++ {
			users[j] = User{
				TUser: cmn.TUser{
					Account:  fmt.Sprintf("bench_batch_user_%d_%d_%d", i, j, time.Now().UnixNano()),
					Category: "benchmark_batch",
					Creator:  null.NewInt(1, true),
					Remark:   null.NewString("test", true),
				},
			}
		}
		err := repo.InsertUsers(ctx, nil, users)
		if err != nil {
			b.Errorf("InsertUsers() error = %v", err)
		}
	}
}

func TestService_InsertUsersWithAccount(t *testing.T) {
	// 创建真实的repo实例
	srv := NewService()

	tests := []struct {
		name    string
		ctx     context.Context
		users   []User
		wantErr bool
		desc    string
	}{
		{
			name: "成功插入单个用户",
			ctx:  context.Background(),
			users: []User{
				{
					TUser: cmn.TUser{
						Category:     "normal",
						OfficialName: null.NewString("测试用户", true),
						Gender:       null.NewString("M", true),
						MobilePhone:  null.NewString("13900138001", true),
						Email:        null.NewString("test2@example.com", true),
						Creator:      null.NewInt(1, true),
						Status:       null.NewString("00", true),
						Remark:       null.NewString("test", true),
					},
				},
			},
			wantErr: false,
			desc:    "测试成功插入单个用户数据",
		},
		{
			name: "成功插入多个用户",
			ctx:  context.Background(),
			users: []User{
				{
					TUser: cmn.TUser{
						Category:     "vip",
						OfficialName: null.NewString("批量用户11", true),
						Gender:       null.NewString("F", true),
						Creator:      null.NewInt(1, true),
						Remark:       null.NewString("test", true),
					},
				},
				{
					TUser: cmn.TUser{
						Category:     "normal",
						OfficialName: null.NewString("批量用户22", true),
						Gender:       null.NewString("M", true),
						IDCardNo:     null.NewString("110101199003011234", true),
						Creator:      null.NewInt(1, true),
						Remark:       null.NewString("test", true),
					},
				},
			},
			wantErr: false,
			desc:    "测试成功插入多个用户数据",
		},
		{
			name:    "空用户列表",
			ctx:     context.Background(),
			users:   []User{},
			wantErr: true,
			desc:    "测试空用户列表应该返回错误",
		},
		{
			name: "缺少必要字段Category",
			ctx:  context.Background(),
			users: []User{
				{
					TUser: cmn.TUser{
						Account: fmt.Sprintf("invalid_user_%d", time.Now().UnixNano()),
						// Category 字段为空
						Creator: null.NewInt(1, true),
						Remark:  null.NewString("test", true),
					},
				},
			},
			wantErr: true,
			desc:    "测试缺少Category字段应该返回验证错误",
		},
		{
			name: "匿名用户类型测试",
			ctx:  context.Background(),
			users: []User{
				{
					TUser: cmn.TUser{
						Category:     "anonymous",
						OfficialName: null.NewString("匿名用户", true),
						// 没有身份证、手机号、邮箱，应该被设置为匿名用户类型
						Creator: null.NewInt(1, true),
						Remark:  null.NewString("test", true),
					},
				},
			},
			wantErr: false,
			desc:    "测试匿名用户类型自动设置",
		},
		{
			name: "注册用户类型测试",
			ctx:  context.Background(),
			users: []User{
				{
					TUser: cmn.TUser{
						Category:     "registered",
						OfficialName: null.NewString("注册用户", true),
						MobilePhone:  null.NewString("13900139040", true),
						// 有手机号，应该被设置为注册用户类型
						Creator: null.NewInt(1, true),
						Remark:  null.NewString("test", true),
					},
				},
			},
			wantErr: false,
			desc:    "测试注册用户类型自动设置",
		},
		{
			name: "包含特殊字符的用户数据",
			ctx:  context.Background(),
			users: []User{
				{
					TUser: cmn.TUser{
						Category:     "special",
						OfficialName: null.NewString("特殊字符用户@#$%^&*()", true),
						Email:        null.NewString("special+test1@example.com", true),
						Creator:      null.NewInt(1, true),
						Remark:       null.NewString("test", true),
					},
				},
			},
			wantErr: false,
			desc:    "测试包含特殊字符的用户数据插入",
		},
		{
			name: "Unicode字符用户数据",
			ctx:  context.Background(),
			users: []User{
				{
					TUser: cmn.TUser{
						Category:     "unicode",
						OfficialName: null.NewString("张三李四王五赵六🎉", true),
						Gender:       null.NewString("M", true),
						Creator:      null.NewInt(1, true),
						Remark:       null.NewString("test", true),
					},
				},
			},
			wantErr: false,
			desc:    "测试Unicode字符用户数据插入",
		},
		{
			name: "强制执行SQL错误",
			ctx:  context.WithValue(context.Background(), "force-error", "Exec"),
			users: []User{
				{
					TUser: cmn.TUser{
						Category: "normal",
						Creator:  null.NewInt(1, true),
						Remark:   null.NewString("test", true),
					},
				},
			},
			wantErr: true,
			desc:    "测试强制执行SQL错误",
		},
		{
			name: "强制触发生成唯一帐号错误",
			ctx:  context.WithValue(context.Background(), "force-error", "GenerateUniqueAccount"),
			users: []User{
				{
					TUser: cmn.TUser{
						Category: "normal",
						Creator:  null.NewInt(1, true),
						Remark:   null.NewString("test", true),
					},
				},
			},
			wantErr: true,
			desc:    "测试强制触发生成唯一帐号错误",
		},
		{
			name: "强制触发插入用户错误",
			ctx:  context.WithValue(context.Background(), "force-error", "InsertUsers"),
			users: []User{
				{
					TUser: cmn.TUser{
						Category: "normal",
						Creator:  null.NewInt(1, true),
						Remark:   null.NewString("test", true),
					},
				},
			},
			wantErr: true,
			desc:    "测试强制触插入用户错误",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("开始测试: %s", tt.desc)

			// 执行插入操作
			err := srv.InsertUsersWithAccount(tt.ctx, nil, tt.users)

			// 验证错误
			if (err != nil) != tt.wantErr {
				t.Errorf("InsertUsers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// 如果期望有错误，则不需要进一步验证
			if tt.wantErr {
				t.Logf("预期错误已正确返回: %v", err)
				return
			}

			// 验证插入成功的情况
			if err == nil && len(tt.users) > 0 {
				t.Logf("成功插入 %d 个用户", len(tt.users))

				// 验证插入的数据是否可以查询到（可选验证）
				for _, user := range tt.users {
					// 查询刚插入的用户
					filter := QueryUsersFilter{
						FuzzyCondition: null.NewString(user.Account, true),
					}
					users, _, queryErr := srv.QueryUsers(context.Background(), nil, 1, 1, filter)
					if queryErr != nil {
						t.Logf("查询插入的用户时出错: %v", queryErr)
						continue
					}
					if len(users) > 0 {
						t.Logf("成功查询到插入的用户: %s", users[0].Account)
						// 验证用户类型是否正确设置
						if users[0].Type.Valid {
							if !user.IDCardNo.Valid && !user.MobilePhone.Valid && !user.Email.Valid {
								// 应该是匿名用户
								if users[0].Type.String != "00" {
									t.Errorf("匿名用户类型设置错误，期望 '00'，实际 '%s'", users[0].Type.String)
								}
							} else {
								// 应该是注册用户
								if users[0].Type.String != "02" {
									t.Errorf("注册用户类型设置错误，期望 '02'，实际 '%s'", users[0].Type.String)
								}
							}
						}
						if users[0].Account == "" {
							t.Errorf("插入的用户Account字段为空，期望非空")
						}
					}
				}
			}
		})
	}
}

// Test_service_CheckTUserFieldExists 测试CheckTUserFieldExists方法
func Test_service_CheckTUserFieldExists(t *testing.T) {
	srv := NewService()

	type args struct {
		ctx   context.Context
		field string
		value any
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "字段存在",
			args: args{
				ctx:   context.Background(),
				field: "account",
				value: "zhangsan",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "字段不存在",
			args: args{
				ctx:   context.Background(),
				field: "email",
				value: "no@example.com",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "空字段名",
			args: args{
				ctx:   context.Background(),
				field: "",
				value: "test_value",
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "不允许的字段",
			args: args{
				ctx:   context.Background(),
				field: "invalid_field",
				value: "test_value",
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "数据库查询错误",
			args: args{
				ctx:   context.WithValue(context.Background(), "force-error", "tx.QueryRow"),
				field: "account",
				value: "test_user",
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "检查手机号存在",
			args: args{
				ctx:   context.Background(),
				field: "mobile_phone",
				value: "13800138001",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "检查身份证号存在",
			args: args{
				ctx:   context.Background(),
				field: "id_card_no",
				value: "110101199502021234",
			},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := srv.CheckTUserFieldExists(tt.args.ctx, nil, tt.args.field, tt.args.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckTUserFieldExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CheckTUserFieldExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test_service_CheckTUserFieldExists_WithTransaction 测试CheckTUserFieldExists方法（启用事务）
func Test_service_CheckTUserFieldExists_WithTransaction(t *testing.T) {
	srv := NewService()
	ctx := context.Background()

	// 获取数据库连接
	pgxConn := cmn.GetPgxConn()
	if pgxConn == nil {
		t.Skip("数据库连接不可用，跳过事务测试")
		return
	}

	// 开始事务
	tx, err := pgxConn.Begin(ctx)
	if err != nil {
		t.Fatalf("开始事务失败: %v", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Logf("回滚事务失败: %v", err)
		}
	}()

	type args struct {
		ctx   context.Context
		field string
		value any
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "字段存在",
			args: args{
				ctx:   context.Background(),
				field: "account",
				value: "zhangsan",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "字段不存在",
			args: args{
				ctx:   context.Background(),
				field: "email",
				value: "no@example.com",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "空字段名",
			args: args{
				ctx:   context.Background(),
				field: "",
				value: "test_value",
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "不允许的字段",
			args: args{
				ctx:   context.Background(),
				field: "invalid_field",
				value: "test_value",
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "数据库查询错误",
			args: args{
				ctx:   context.WithValue(context.Background(), "force-error", "tx.QueryRow"),
				field: "account",
				value: "test_user",
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "检查手机号存在",
			args: args{
				ctx:   context.Background(),
				field: "mobile_phone",
				value: "13800138001",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "检查身份证号存在",
			args: args{
				ctx:   context.Background(),
				field: "id_card_no",
				value: "110101199502021234",
			},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := srv.CheckTUserFieldExists(tt.args.ctx, tx, tt.args.field, tt.args.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckTUserFieldExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CheckTUserFieldExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test_service_CheckTUserRowExists 测试CheckTUserRowExists方法
func Test_service_CheckTUserRowExists(t *testing.T) {
	srv := NewService()

	type args struct {
		ctx    context.Context
		fields map[string]any
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "用户行存在",
			args: args{
				ctx: context.Background(),
				fields: map[string]any{
					"account":       "zhangsan",
					"official_name": "张三",
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "用户行不存在",
			args: args{
				ctx: context.Background(),
				fields: map[string]any{
					"email":        "test@example.com",
					"mobile_phone": "13800138000",
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "空字段映射",
			args: args{
				ctx:    context.Background(),
				fields: map[string]any{},
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "包含不允许的字段",
			args: args{
				ctx: context.Background(),
				fields: map[string]any{
					"account":       "test_user",
					"invalid_field": "invalid_value",
				},
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "数据库查询错误",
			args: args{
				ctx: context.WithValue(context.Background(), "force-error", "tx.QueryRow"),
				fields: map[string]any{
					"account": "test_user",
				},
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "多字段查询存在",
			args: args{
				ctx: context.Background(),
				fields: map[string]any{
					"official_name": "张三",
					"mobile_phone":  "13800138001",
					"email":         "zhangsan@example.com",
					"id_card_no":    "440106199001011234",
				},
			},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := srv.CheckTUserRowExists(tt.args.ctx, nil, tt.args.fields)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckTUserRowExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CheckTUserRowExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test_service_CheckTUserRowExists_WithTransaction 测试CheckTUserRowExists方法（开启事务）
func Test_service_CheckTUserRowExists_WithTransaction(t *testing.T) {
	srv := NewService()

	ctx := context.Background()

	// 获取数据库连接
	pgxConn := cmn.GetPgxConn()
	if pgxConn == nil {
		t.Skip("数据库连接不可用，跳过事务测试")
		return
	}

	// 开始事务
	tx, err := pgxConn.Begin(ctx)
	if err != nil {
		t.Fatalf("开始事务失败: %v", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Logf("回滚事务失败: %v", err)
		}
	}()

	type args struct {
		ctx    context.Context
		fields map[string]any
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "用户行存在",
			args: args{
				ctx: context.Background(),
				fields: map[string]any{
					"account":       "zhangsan",
					"official_name": "张三",
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "用户行不存在",
			args: args{
				ctx: context.Background(),
				fields: map[string]any{
					"email":        "test@example.com",
					"mobile_phone": "13800138000",
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "空字段映射",
			args: args{
				ctx:    context.Background(),
				fields: map[string]any{},
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "包含不允许的字段",
			args: args{
				ctx: context.Background(),
				fields: map[string]any{
					"account":       "test_user",
					"invalid_field": "invalid_value",
				},
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "数据库查询错误",
			args: args{
				ctx: context.WithValue(context.Background(), "force-error", "tx.QueryRow"),
				fields: map[string]any{
					"account": "test_user",
				},
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "多字段查询存在",
			args: args{
				ctx: context.Background(),
				fields: map[string]any{
					"official_name": "张三",
					"mobile_phone":  "13800138001",
					"email":         "zhangsan@example.com",
					"id_card_no":    "440106199001011234",
				},
			},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := srv.CheckTUserRowExists(tt.args.ctx, tx, tt.args.fields)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckTUserRowExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CheckTUserRowExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test_service_GenerateUniqueAccount 测试GenerateUniqueAccount方法
func Test_service_GenerateUniqueAccount(t *testing.T) {
	srv := NewService()

	type args struct {
		ctx         context.Context
		length      int
		maxAttempts int
	}
	tests := []struct {
		name       string
		args       args
		wantLength int
		wantErr    bool
	}{
		{
			name: "成功生成唯一账号",
			args: args{
				ctx:         context.Background(),
				length:      9,
				maxAttempts: 10,
			},
			wantLength: 9,
			wantErr:    false,
		},
		{
			name: "长度参数无效",
			args: args{
				ctx:         context.Background(),
				length:      0,
				maxAttempts: 10,
			},
			wantLength: 0,
			wantErr:    true,
		},
		{
			name: "最大尝试次数参数无效",
			args: args{
				ctx:         context.Background(),
				length:      9,
				maxAttempts: -1,
			},
			wantLength: 0,
			wantErr:    true,
		},
		{
			name: "检查账号存在性时出错",
			args: args{
				ctx:         context.WithValue(context.Background(), "force-error", "CheckTUserFieldExists"),
				length:      8,
				maxAttempts: 10,
			},
			wantLength: 0,
			wantErr:    true,
		},
		{
			name: "超过最大尝试次数",
			args: args{
				ctx:         context.WithValue(context.Background(), "force-error", "exist"),
				length:      8,
				maxAttempts: 5,
			},
			wantLength: 0,
			wantErr:    true,
		},
		{
			name: "生成短账号",
			args: args{
				ctx:         context.Background(),
				length:      4,
				maxAttempts: 5,
			},
			wantLength: 4,
			wantErr:    false,
		},
		{
			name: "生成长账号",
			args: args{
				ctx:         context.Background(),
				length:      18,
				maxAttempts: 10,
			},
			wantLength: 18,
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := srv.GenerateUniqueAccount(tt.args.ctx, nil, tt.args.length, tt.args.maxAttempts)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateUniqueAccount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != tt.wantLength {
				t.Errorf("GenerateUniqueAccount() length = %v, want %v", got, tt.wantLength)
			}
		})
	}
}

// Test_service_GenerateUniqueAccount_WithTransaction 测试GenerateUniqueAccount方法（启用事务）
func Test_service_GenerateUniqueAccount_WithTransaction(t *testing.T) {
	srv := NewService()

	ctx := context.Background()

	// 获取数据库连接
	pgxConn := cmn.GetPgxConn()
	if pgxConn == nil {
		t.Skip("数据库连接不可用，跳过事务测试")
		return
	}

	// 开始事务
	tx, err := pgxConn.Begin(ctx)
	if err != nil {
		t.Fatalf("开始事务失败: %v", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Logf("回滚事务失败: %v", err)
		}
	}()

	type args struct {
		ctx         context.Context
		length      int
		maxAttempts int
	}
	tests := []struct {
		name       string
		args       args
		wantLength int
		wantErr    bool
	}{
		{
			name: "成功生成唯一账号",
			args: args{
				ctx:         context.Background(),
				length:      9,
				maxAttempts: 10,
			},
			wantLength: 9,
			wantErr:    false,
		},
		{
			name: "长度参数无效",
			args: args{
				ctx:         context.Background(),
				length:      0,
				maxAttempts: 10,
			},
			wantLength: 0,
			wantErr:    true,
		},
		{
			name: "检查账号存在性时出错",
			args: args{
				ctx:         context.WithValue(context.Background(), "force-error", "CheckTUserFieldExists"),
				length:      8,
				maxAttempts: 10,
			},
			wantLength: 0,
			wantErr:    true,
		},
		{
			name: "生成短账号",
			args: args{
				ctx:         context.Background(),
				length:      4,
				maxAttempts: 5,
			},
			wantLength: 4,
			wantErr:    false,
		},
		{
			name: "生成长账号",
			args: args{
				ctx:         context.Background(),
				length:      18,
				maxAttempts: 10,
			},
			wantLength: 18,
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := srv.GenerateUniqueAccount(tt.args.ctx, tx, tt.args.length, tt.args.maxAttempts)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateUniqueAccount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != tt.wantLength {
				t.Errorf("GenerateUniqueAccount() length = %v, want %v", got, tt.wantLength)
			}
		})
	}
}

// Test_service_ValidateUser 测试ValidateUser方法
func Test_service_ValidateUser(t *testing.T) {
	srv := NewService()

	type args struct {
		ctx   context.Context
		users []User
	}
	tests := []struct {
		name        string
		args        args
		wantValid   []User
		wantInvalid []InvalidUser
		wantErr     bool
	}{
		{
			name: "所有用户都有效",
			args: args{
				ctx: context.Background(),
				users: []User{
					{
						TUser: cmn.TUser{
							Account:      "new_user_001",
							OfficialName: null.NewString("新用户001_Test_service_ValidateUser", true),
							Email:        null.NewString("new001ValidateUser@example.com", true),
						},
						Domains: []null.String{
							null.NewString("cst.school^teacher", true),
						},
					},
					{
						TUser: cmn.TUser{
							Account:      "new_user_002",
							OfficialName: null.NewString("新用户002_Test_service_ValidateUser", true),
							MobilePhone:  null.NewString("13900139111", true),
						},
						Domains: []null.String{
							null.NewString("cst.school^teacher", true),
						},
					},
				},
			},
			wantValid: []User{
				{
					TUser: cmn.TUser{
						Account:      "new_user_001",
						OfficialName: null.NewString("新用户001", true),
						Email:        null.NewString("new001@example.com", true),
					},
				},
				{
					TUser: cmn.TUser{
						Account:      "new_user_002",
						OfficialName: null.NewString("新用户002", true),
						MobilePhone:  null.NewString("13900139000", true),
					},
				},
			},
			wantInvalid: nil,
			wantErr:     false,
		},
		{
			name: "存在无效用户",
			args: args{
				ctx: context.Background(),
				users: []User{
					{
						TUser: cmn.TUser{
							Account: "zhangsan",
						},
					},
				},
			},
			wantValid: nil,
			wantInvalid: []InvalidUser{
				{
					Account: null.NewString("zhangsan", true),
					ErrorMsg: []null.String{
						null.NewString("账号已存在", true),
						null.NewString("角色不能为空", true),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "尝试插入超级管理员用户",
			args: args{
				ctx: context.Background(),
				users: []User{
					{
						TUser: cmn.TUser{
							Account: "zhangsan",
						},
						Domains: []null.String{
							null.NewString("cst.school^superAdmin", true),
						},
					},
				},
			},
			wantValid: nil,
			wantInvalid: []InvalidUser{
				{
					Account: null.NewString("zhangsan", true),
					ErrorMsg: []null.String{
						null.NewString("账号已存在", true),
						null.NewString("不允许为超级管理员角色", true),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "邮箱地址不合法",
			args: args{
				ctx: context.Background(),
				users: []User{
					{
						TUser: cmn.TUser{
							Account: "zhangsanyes",
							Email:   null.NewString("invalid-email", true),
						},
						Domains: []null.String{
							null.NewString("cst.school^teacher", true),
						},
					},
				},
			},
			wantValid: nil,
			wantInvalid: []InvalidUser{
				{
					Account: null.NewString("zhangsanyes", true),
					Email:   null.NewString("invalid-email", true),
					ErrorMsg: []null.String{
						null.NewString("邮箱格式不正确", true),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "空用户列表",
			args: args{
				ctx:   context.Background(),
				users: []User{},
			},
			wantValid:   nil,
			wantInvalid: []InvalidUser{},
			wantErr:     true,
		},
		{
			name: "检查用户存在性时出错",
			args: args{
				ctx: context.WithValue(context.Background(), "force-error", "CheckTUserRowExists"),
				users: []User{
					{
						TUser: cmn.TUser{
							Account: "test_user",
						},
					},
				},
			},
			wantValid:   nil,
			wantInvalid: []InvalidUser{},
			wantErr:     true,
		},
		{
			name: "混合有效和无效用户",
			args: args{
				ctx: context.Background(),
				users: []User{
					{
						TUser: cmn.TUser{
							Account:      "valid_user",
							OfficialName: null.NewString("有效用户", true),
						},
						Domains: []null.String{
							null.NewString("cst.school^teacher", true),
						},
					},
					{
						TUser: cmn.TUser{
							Account: "invalid_user",
							Email:   null.NewString("zhangsan@example.com", true),
						},
					},
				},
			},
			wantValid: []User{
				{
					TUser: cmn.TUser{
						Account:      "valid_user",
						OfficialName: null.NewString("有效用户", true),
					},
					Domains: []null.String{
						null.NewString("cst.school^teacher", true),
					},
				},
			},
			wantInvalid: []InvalidUser{
				{
					Account: null.NewString("invalid_user", true),
					ErrorMsg: []null.String{
						null.NewString("邮箱已存在", true),
						null.NewString("角色不能为空", true),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "多个错误信息的无效用户",
			args: args{
				ctx: context.Background(),
				users: []User{
					{
						TUser: cmn.TUser{
							Account:     "lisi",
							Email:       null.NewString("zhangsan@example.com", true),
							MobilePhone: null.NewString("13900139002", true),
							IDCardNo:    null.NewString("310115198801011234", true),
						},
						Domains: []null.String{
							null.NewString("cst.school^invalid", true),
						},
					},
				},
			},
			wantValid: nil,
			wantInvalid: []InvalidUser{
				{
					Account: null.NewString("lisi", true),
					ErrorMsg: []null.String{
						null.NewString("账号已存在", true),
						null.NewString("邮箱已存在", true),
						null.NewString("手机号已存在", true),
						null.NewString("证件号已存在", true),
						null.NewString("角色不合法", true),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "多个已存在用户",
			args: args{
				ctx: context.Background(),
				users: []User{
					{
						TUser: cmn.TUser{
							Account:      "lisi",
							OfficialName: null.NewString("李四", true),
							Email:        null.NewString("lisi@example.com", true),
							MobilePhone:  null.NewString("13900139002", true),
							IDCardNo:     null.NewString("110101199502021234", true),
						},
					},
					{
						TUser: cmn.TUser{
							Account:      "zhangsan",
							OfficialName: null.NewString("张三", true),
							Email:        null.NewString("zhangsan@example.com", true),
							MobilePhone:  null.NewString("13800138001", true),
							IDCardNo:     null.NewString("440106199001011234", true),
						},
					},
				},
			},
			wantValid:   nil,
			wantInvalid: nil,
			wantErr:     false,
		},
		{
			name: "检查Account字段失败",
			args: args{
				ctx: context.WithValue(context.Background(), "force-error", "CheckTUserFieldExists_account"),
				users: []User{
					{
						TUser: cmn.TUser{
							Account:      "lisi",
							OfficialName: null.NewString("李五", true),
							Email:        null.NewString("lisi@example.com", true),
							MobilePhone:  null.NewString("13900139002", true),
							IDCardNo:     null.NewString("110101199502021234", true),
						},
					},
				},
			},
			wantValid:   nil,
			wantInvalid: nil,
			wantErr:     true,
		},
		{
			name: "检查MobilePhone字段失败",
			args: args{
				ctx: context.WithValue(context.Background(), "force-error", "CheckTUserFieldExists_mobile_phone"),
				users: []User{
					{
						TUser: cmn.TUser{
							Account:      "lisi",
							OfficialName: null.NewString("李五", true),
							Email:        null.NewString("lisi@example.com", true),
							MobilePhone:  null.NewString("13900139002", true),
							IDCardNo:     null.NewString("110101199502021234", true),
						},
					},
				},
			},
			wantValid:   nil,
			wantInvalid: nil,
			wantErr:     true,
		},
		{
			name: "检查Email字段失败",
			args: args{
				ctx: context.WithValue(context.Background(), "force-error", "CheckTUserFieldExists_email"),
				users: []User{
					{
						TUser: cmn.TUser{
							Account:      "lisi",
							OfficialName: null.NewString("李五", true),
							Email:        null.NewString("lisi@example.com", true),
							MobilePhone:  null.NewString("13900139002", true),
							IDCardNo:     null.NewString("110101199502021234", true),
						},
					},
				},
			},
			wantValid:   nil,
			wantInvalid: nil,
			wantErr:     true,
		},
		{
			name: "检查IDCardNo字段失败",
			args: args{
				ctx: context.WithValue(context.Background(), "force-error", "CheckTUserFieldExists_id_card_no"),
				users: []User{
					{
						TUser: cmn.TUser{
							Account:      "lisi",
							OfficialName: null.NewString("李五", true),
							Email:        null.NewString("lisi@example.com", true),
							MobilePhone:  null.NewString("13900139002", true),
							IDCardNo:     null.NewString("110101199502021234", true),
						},
					},
				},
			},
			wantValid:   nil,
			wantInvalid: nil,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValid, gotInvalid, err := srv.ValidateUserToBeInsert(tt.args.ctx, nil, tt.args.users)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUserToBeInsert() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// 比较有效用户列表
			if len(gotValid) != len(tt.wantValid) {
				t.Errorf("ValidateUserToBeInsert() gotValid length = %v, want %v", len(gotValid), len(tt.wantValid))
				return
			}
			for i, user := range gotValid {
				if user.Account != tt.wantValid[i].Account {
					t.Errorf("ValidateUserToBeInsert() gotValid[%d].Account = %v, want %v", i, user.Account, tt.wantValid[i].Account)
				}
			}

			// 比较无效用户列表
			if len(gotInvalid) != len(tt.wantInvalid) {
				t.Errorf("ValidateUserToBeInsert() gotInvalid length = %v, want %v", len(gotInvalid), len(tt.wantInvalid))
				return
			}
			for i, user := range gotInvalid {
				if user.Account.String != tt.wantInvalid[i].Account.String {
					t.Errorf("ValidateUserToBeInsert() gotInvalid[%d].Account = %v, want %v", i, user.Account.String, tt.wantInvalid[i].Account.String)
				}
				if len(user.ErrorMsg) != len(tt.wantInvalid[i].ErrorMsg) {
					t.Errorf("ValidateUserToBeInsert() gotInvalid[%d].ErrorMsg length = %v, want %v", i, len(user.ErrorMsg), len(tt.wantInvalid[i].ErrorMsg))
				}
			}
		})
	}
}

// Test_service_ValidateUser_WithTransaction 测试ValidateUser方法（启用事务）
func Test_service_ValidateUser_WithTransaction(t *testing.T) {
	srv := NewService()

	ctx := context.Background()

	// 获取数据库连接
	pgxConn := cmn.GetPgxConn()
	if pgxConn == nil {
		t.Skip("数据库连接不可用，跳过事务测试")
		return
	}

	// 开始事务
	tx, err := pgxConn.Begin(ctx)
	if err != nil {
		t.Fatalf("开始事务失败: %v", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Logf("回滚事务失败: %v", err)
		}
	}()

	type args struct {
		ctx   context.Context
		users []User
	}
	tests := []struct {
		name        string
		args        args
		wantValid   []User
		wantInvalid []InvalidUser
		wantErr     bool
	}{
		{
			name: "所有用户都有效",
			args: args{
				ctx: context.Background(),
				users: []User{
					{
						TUser: cmn.TUser{
							Account:      "new_user_001",
							OfficialName: null.NewString("新用户001_Test_service_ValidateUser_WithTransaction", true),
							Email:        null.NewString("new001ValidateUser_WithTransaction@example.com", true),
						},
						Domains: []null.String{
							null.NewString("cst.school^teacher", true),
						},
					},
					{
						TUser: cmn.TUser{
							Account:      "new_user_002",
							OfficialName: null.NewString("新用户002_Test_service_ValidateUser_WithTransaction", true),
							MobilePhone:  null.NewString("13900139222", true),
						},
						Domains: []null.String{
							null.NewString("cst.school^teacher", true),
						},
					},
				},
			},
			wantValid: []User{
				{
					TUser: cmn.TUser{
						Account:      "new_user_001",
						OfficialName: null.NewString("新用户001", true),
						Email:        null.NewString("new001@example.com", true),
					},
				},
				{
					TUser: cmn.TUser{
						Account:      "new_user_002",
						OfficialName: null.NewString("新用户002", true),
						MobilePhone:  null.NewString("13900139000", true),
					},
				},
			},
			wantInvalid: nil,
			wantErr:     false,
		},
		{
			name: "存在无效用户",
			args: args{
				ctx: context.Background(),
				users: []User{
					{
						TUser: cmn.TUser{
							Account: "zhangsan",
						},
						Domains: []null.String{
							null.NewString("cst.school^teacher", true),
						},
					},
				},
			},
			wantValid: nil,
			wantInvalid: []InvalidUser{
				{
					Account: null.NewString("zhangsan", true),
					ErrorMsg: []null.String{
						null.NewString("账号已存在", true),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "空用户列表",
			args: args{
				ctx:   context.Background(),
				users: []User{},
			},
			wantValid:   nil,
			wantInvalid: []InvalidUser{},
			wantErr:     true,
		},
		{
			name: "检查用户存在性时出错",
			args: args{
				ctx: context.WithValue(context.Background(), "force-error", "CheckTUserRowExists"),
				users: []User{
					{
						TUser: cmn.TUser{
							Account: "test_user",
						},
					},
				},
			},
			wantValid:   nil,
			wantInvalid: []InvalidUser{},
			wantErr:     true,
		},
		{
			name: "混合有效和无效用户",
			args: args{
				ctx: context.Background(),
				users: []User{
					{
						TUser: cmn.TUser{
							Account:      "valid_user",
							OfficialName: null.NewString("有效用户", true),
						},
						Domains: []null.String{
							null.NewString("cst.school^teacher", true),
						},
					},
					{
						TUser: cmn.TUser{
							Account: "invalid_user",
							Email:   null.NewString("zhangsan@example.com", true),
						},
					},
				},
			},
			wantValid: []User{
				{
					TUser: cmn.TUser{
						Account:      "valid_user",
						OfficialName: null.NewString("有效用户", true),
					},
				},
			},
			wantInvalid: []InvalidUser{
				{
					Account: null.NewString("invalid_user", true),
					ErrorMsg: []null.String{
						null.NewString("邮箱已存在", true),
						null.NewString("角色不能为空", true),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "多个错误信息的无效用户",
			args: args{
				ctx: context.Background(),
				users: []User{
					{
						TUser: cmn.TUser{
							Account:     "lisi",
							Email:       null.NewString("zhangsan@example.com", true),
							MobilePhone: null.NewString("13900139002", true),
						},
					},
				},
			},
			wantValid: nil,
			wantInvalid: []InvalidUser{
				{
					Account: null.NewString("lisi", true),
					ErrorMsg: []null.String{
						null.NewString("账号已存在", true),
						null.NewString("邮箱已存在", true),
						null.NewString("手机号已存在", true),
						null.NewString("角色不能为空", true),
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValid, gotInvalid, err := srv.ValidateUserToBeInsert(tt.args.ctx, tx, tt.args.users)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUserToBeInsert() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// 比较有效用户列表
			if len(gotValid) != len(tt.wantValid) {
				t.Errorf("ValidateUserToBeInsert() gotValid length = %v, want %v", len(gotValid), len(tt.wantValid))
				return
			}
			for i, user := range gotValid {
				if user.Account != tt.wantValid[i].Account {
					t.Errorf("ValidateUserToBeInsert() gotValid[%d].Account = %v, want %v", i, user.Account, tt.wantValid[i].Account)
				}
			}

			// 比较无效用户列表
			if len(gotInvalid) != len(tt.wantInvalid) {
				t.Errorf("ValidateUserToBeInsert() gotInvalid length = %v, want %v", len(gotInvalid), len(tt.wantInvalid))
				return
			}
			for i, user := range gotInvalid {
				if user.Account.String != tt.wantInvalid[i].Account.String {
					t.Errorf("ValidateUserToBeInsert() gotInvalid[%d].Account = %v, want %v", i, user.Account.String, tt.wantInvalid[i].Account.String)
				}
				if len(user.ErrorMsg) != len(tt.wantInvalid[i].ErrorMsg) {
					t.Errorf("ValidateUserToBeInsert() gotInvalid[%d].ErrorMsg length = %v, want %v", i, len(user.ErrorMsg), len(tt.wantInvalid[i].ErrorMsg))
				}
			}
		})
	}
}
