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
			wantUsersCount: 5,
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
			wantUsersCount: 10,
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
			wantUsersCount: 4,
			wantErr:        false,
			desc:           "测试多个过滤条件组合查询",
		},
		{
			name:           "大页面查询测试",
			ctx:            context.WithValue(context.Background(), "force-error", ""),
			page:           1,
			pageSize:       100,
			filter:         QueryUsersFilter{},
			wantUsersCount: 21,
			wantErr:        false,
			desc:           "测试大页面大小的查询",
		},
		{
			name:           "第二页查询测试",
			ctx:            context.WithValue(context.Background(), "force-error", ""),
			page:           2,
			pageSize:       10,
			filter:         QueryUsersFilter{},
			wantUsersCount: 10,
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

// TestRepo_InsertUsers 测试InsertUsers方法
func TestRepo_InsertUsers(t *testing.T) {
	// 创建真实的repo实例
	repo := NewRepo()

	tests := []struct {
		name    string
		ctx     context.Context
		tx      pgx.Tx
		users   []cmn.TUser
		wantErr bool
		desc    string
	}{
		{
			name: "成功插入单个用户",
			ctx:  context.Background(),
			tx:   nil,
			users: []cmn.TUser{
				{
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
			},
			wantErr: false,
			desc:    "测试成功插入单个用户数据",
		},
		{
			name: "成功插入多个用户",
			ctx:  context.Background(),
			tx:   nil,
			users: []cmn.TUser{
				{
					Account:      fmt.Sprintf("batch_user_1_%d", time.Now().UnixNano()),
					Category:     "vip",
					OfficialName: null.NewString("批量用户1", true),
					Gender:       null.NewString("F", true),
					Creator:      null.NewInt(1, true),
					Remark:       null.NewString("test", true),
				},
				{
					Account:      fmt.Sprintf("batch_user_2_%d", time.Now().UnixNano()),
					Category:     "normal",
					OfficialName: null.NewString("批量用户2", true),
					Gender:       null.NewString("M", true),
					IDCardNo:     null.NewString("110101199001011234", true),
					Creator:      null.NewInt(1, true),
					Remark:       null.NewString("test", true),
				},
			},
			wantErr: false,
			desc:    "测试成功插入多个用户数据",
		},
		{
			name:    "空用户列表",
			ctx:     context.Background(),
			tx:      nil,
			users:   []cmn.TUser{},
			wantErr: true,
			desc:    "测试空用户列表应该返回错误",
		},
		{
			name: "缺少必要字段Account",
			ctx:  context.Background(),
			tx:   nil,
			users: []cmn.TUser{
				{
					// Account 字段为空
					Category: "normal",
					Creator:  null.NewInt(1, true),
					Remark:   null.NewString("test", true),
				},
			},
			wantErr: true,
			desc:    "测试缺少Account字段应该返回验证错误",
		},
		{
			name: "缺少必要字段Category",
			ctx:  context.Background(),
			tx:   nil,
			users: []cmn.TUser{
				{
					Account: fmt.Sprintf("invalid_user_%d", time.Now().UnixNano()),
					// Category 字段为空
					Creator: null.NewInt(1, true),
					Remark:  null.NewString("test", true),
				},
			},
			wantErr: true,
			desc:    "测试缺少Category字段应该返回验证错误",
		},
		{
			name: "缺少必要字段Account",
			ctx:  context.Background(),
			tx:   nil,
			users: []cmn.TUser{
				{
					Remark: null.NewString("test", true),
				},
			},
			wantErr: true,
			desc:    "测试缺少Creator字段应该返回验证错误",
		},
		{
			name: "匿名用户类型测试",
			ctx:  context.Background(),
			tx:   nil,
			users: []cmn.TUser{
				{
					Account:      fmt.Sprintf("anonymous_user_%d", time.Now().UnixNano()),
					Category:     "anonymous",
					OfficialName: null.NewString("匿名用户", true),
					// 没有身份证、手机号、邮箱，应该被设置为匿名用户类型
					Creator: null.NewInt(1, true),
					Remark:  null.NewString("test", true),
				},
			},
			wantErr: false,
			desc:    "测试匿名用户类型自动设置",
		},
		{
			name: "注册用户类型测试",
			ctx:  context.Background(),
			tx:   nil,
			users: []cmn.TUser{
				{
					Account:      fmt.Sprintf("registered_user_%d", time.Now().UnixNano()),
					Category:     "registered",
					OfficialName: null.NewString("注册用户", true),
					MobilePhone:  null.NewString("13900139000", true),
					// 有手机号，应该被设置为注册用户类型
					Creator: null.NewInt(1, true),
					Remark:  null.NewString("test", true),
				},
			},
			wantErr: false,
			desc:    "测试注册用户类型自动设置",
		},
		{
			name: "包含特殊字符的用户数据",
			ctx:  context.Background(),
			tx:   nil,
			users: []cmn.TUser{
				{
					Account:      fmt.Sprintf("special_user_%d", time.Now().UnixNano()),
					Category:     "special",
					OfficialName: null.NewString("特殊字符用户@#$%^&*()", true),
					Email:        null.NewString("special+test@example.com", true),
					Creator:      null.NewInt(1, true),
					Remark:       null.NewString("test", true),
				},
			},
			wantErr: false,
			desc:    "测试包含特殊字符的用户数据插入",
		},
		{
			name: "Unicode字符用户数据",
			ctx:  context.Background(),
			tx:   nil,
			users: []cmn.TUser{
				{
					Account:      fmt.Sprintf("unicode_用户_%d", time.Now().UnixNano()),
					Category:     "unicode",
					OfficialName: null.NewString("张三李四王五赵六🎉", true),
					Gender:       null.NewString("M", true),
					Creator:      null.NewInt(1, true),
					Remark:       null.NewString("test", true),
				},
			},
			wantErr: false,
			desc:    "测试Unicode字符用户数据插入",
		},
		{
			name: "强制事务开始错误",
			ctx:  context.WithValue(context.Background(), "force-error", "pgxConn.Begin"),
			tx:   nil,
			users: []cmn.TUser{
				{
					Account:  fmt.Sprintf("error_user_%d", time.Now().UnixNano()),
					Category: "normal",
					Creator:  null.NewInt(1, true),
					Remark:   null.NewString("test", true),
				},
			},
			wantErr: true,
			desc:    "测试强制事务开始错误",
		},
		{
			name: "强制执行SQL错误",
			ctx:  context.WithValue(context.Background(), "force-error", "tx.Exec"),
			tx:   nil,
			users: []cmn.TUser{
				{
					Account:  fmt.Sprintf("error_user_%d", time.Now().UnixNano()),
					Category: "normal",
					Creator:  null.NewInt(1, true),
					Remark:   null.NewString("test", true),
				},
			},
			wantErr: true,
			desc:    "测试强制执行SQL错误",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("开始测试: %s", tt.desc)

			// 执行插入操作
			err := repo.InsertUsers(tt.ctx, tt.tx, tt.users)

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
						Account: null.NewString(user.Account, true),
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

// TestRepo_InsertUsers_WithTransaction 测试带事务的插入操作
func TestRepo_InsertUsers_WithTransaction(t *testing.T) {
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
		name    string
		users   []cmn.TUser
		wantErr bool
		desc    string
	}{
		{
			name: "事务中成功插入用户",
			users: []cmn.TUser{
				{
					Account:      fmt.Sprintf("tx_user_%d", time.Now().UnixNano()),
					Category:     "transaction",
					OfficialName: null.NewString("事务用户", true),
					Creator:      null.NewInt(1, true),
					Remark:       null.NewString("test", true),
				},
			},
			wantErr: false,
			desc:    "测试在事务中成功插入用户",
		},
		{
			name: "事务中插入多个用户",
			users: []cmn.TUser{
				{
					Account:  fmt.Sprintf("tx_batch_1_%d", time.Now().UnixNano()),
					Category: "batch_tx",
					Creator:  null.NewInt(1, true),
					Remark:   null.NewString("test", true),
				},
				{
					Account:  fmt.Sprintf("tx_batch_2_%d", time.Now().UnixNano()),
					Category: "batch_tx",
					Creator:  null.NewInt(1, true),
					Remark:   null.NewString("test", true),
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

// TestRepo_InsertUsers_Performance 性能测试
func TestRepo_InsertUsers_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过性能测试")
	}

	repo := NewRepo()
	ctx := context.Background()

	// 测试不同批次大小的性能
	batchSizes := []int{1, 10, 50, 100}

	for _, batchSize := range batchSizes {
		t.Run(fmt.Sprintf("批次大小_%d", batchSize), func(t *testing.T) {
			// 准备测试数据
			users := make([]cmn.TUser, batchSize)
			for i := 0; i < batchSize; i++ {
				users[i] = cmn.TUser{
					Account:      fmt.Sprintf("perf_user_%d_%d", batchSize, time.Now().UnixNano()+int64(i)),
					Category:     "performance",
					OfficialName: null.NewString(fmt.Sprintf("性能测试用户%d", i+1), true),
					Creator:      null.NewInt(1, true),
					Remark:       null.NewString("test", true),
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
	repo := NewRepo()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		users := []cmn.TUser{
			{
				Account:  fmt.Sprintf("bench_user_%d_%d", i, time.Now().UnixNano()),
				Category: "benchmark",
				Creator:  null.NewInt(1, true),
				Remark:   null.NewString("test", true),
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
	repo := NewRepo()
	ctx := context.Background()
	batchSize := 10

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		users := make([]cmn.TUser, batchSize)
		for j := 0; j < batchSize; j++ {
			users[j] = cmn.TUser{
				Account:  fmt.Sprintf("bench_batch_user_%d_%d_%d", i, j, time.Now().UnixNano()),
				Category: "benchmark_batch",
				Creator:  null.NewInt(1, true),
				Remark:   null.NewString("test", true),
			}
		}
		err := repo.InsertUsers(ctx, nil, users)
		if err != nil {
			b.Errorf("InsertUsers() error = %v", err)
		}
	}
}
