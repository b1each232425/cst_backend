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

// TestRepo_QueryUsers 测试QueryUsers方法
func TestRepo_QueryUsers(t *testing.T) {
	// 创建真实的repo实例
	repo := NewRepo()

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
			name:     "账号过滤查询测试",
			ctx:      context.WithValue(context.Background(), "force-error", ""),
			page:     1,
			pageSize: 10,
			filter: QueryUsersFilter{
				Account: null.NewString("admin", true),
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
				Name: null.NewString("测试", true),
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
				Phone: null.NewString("138", true),
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
				Email: null.NewString("@example.com", true),
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
				Account: null.NewString("nonexistent_user_12345", true),
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
				Account: null.NewString("nonexistent_user_12345", true),
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
				Account: null.NewString("nonexistent_user_12345", true),
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
				Account: null.NewString("nonexistent_user_12345", true),
			},
			wantUsersCount: 0,
			wantErr:        true,
			desc:           "测试触发读取行数据报错",
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
			if tt.filter.Account.Valid {
				for i, user := range users {
					if !containsIgnoreCase(user.Account, tt.filter.Account.String) {
						t.Errorf("用户[%d]的Account '%s' 不包含过滤条件 '%s'", i, user.Account, tt.filter.Account.String)
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

// TestRepo_QueryUsers_WithTransaction 测试带事务的查询
func TestRepo_QueryUsers_WithTransaction(t *testing.T) {
	repo := NewRepo()
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

// TestRepo_QueryUsers_Performance 性能测试
func TestRepo_QueryUsers_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过性能测试")
	}

	repo := NewRepo()
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

// TestRepo_QueryUsers_EdgeCases 边界情况测试
func TestRepo_QueryUsers_EdgeCases(t *testing.T) {
	repo := NewRepo()
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
				Account: null.NewString("test'user", true), // 包含单引号
			},
			wantErr: false,
			desc:    "测试包含特殊字符的过滤条件",
		},
		{
			name:     "SQL注入防护测试",
			page:     1,
			pageSize: 10,
			filter: QueryUsersFilter{
				Account: null.NewString("'; DROP TABLE t_user; --", true),
			},
			wantErr: false,
			desc:    "测试SQL注入防护",
		},
		{
			name:     "Unicode字符过滤测试",
			page:     1,
			pageSize: 10,
			filter: QueryUsersFilter{
				Name: null.NewString("测试用户🎉", true),
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
	repo := NewRepo()
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
	repo := NewRepo()
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
