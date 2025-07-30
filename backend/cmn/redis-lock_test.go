package cmn

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// 创建一个使用真实Redis连接的上下文
func createRealContext() (context.Context, *redis.Client) {
	// 使用项目中已有的Redis连接池获取连接
	conn := GetRedisConn()
	q := &ServiceCtx{
		RedisClient: conn,
	}
	return context.WithValue(context.Background(), QNearKey, q), conn
}

// 清理测试使用的Redis键
func cleanupTestKeys(conn *redis.Client, keyPrefix string, resourceID int64) {
	key := fmt.Sprintf("%s%d", keyPrefix, resourceID)
	_, err := conn.Do(context.Background(), "DEL", key).Result()
	if err != nil {
		fmt.Printf("清理测试键失败: %v\n", err)
	}
}

func TestTryLock(t *testing.T) {

	assert := assert.New(t)
	ConfigureForTest()
	z = GetLogger()
	type args struct {
		ctx        context.Context
		resourceID int64
		holderID   int64
		keyPrefix  string
		expiration time.Duration
	}
	tests := []struct {
		name    string
		args    args
		setup   func(*redis.Client)
		cleanup func(*redis.Client)
		want    bool
		want1   int64
		wantErr bool
	}{
		{
			name: "成功获取锁",
			args: args{
				resourceID: 123,
				holderID:   456,
				keyPrefix:  "test_lock:",
				expiration: 5 * time.Minute,
			},
			setup: func(conn *redis.Client) {
				// 确保测试前键不存在
				_, err := conn.Do(context.Background(), "DEL", "test_lock:123").Result()
				if err != nil {
					t.Fatalf("清理测试键失败: %v", err)
				}
			},
			cleanup: func(conn *redis.Client) {
				cleanupTestKeys(conn, "test_lock:", 123)
			},
			want:    true,
			want1:   -1,
			wantErr: false,
		},
		{
			name: "resourceID非法",
			args: args{
				resourceID: 0, // 非法值
				holderID:   456,
				keyPrefix:  "test_lock:",
				expiration: 5 * time.Minute,
			},
			setup:   func(conn *redis.Client) {},
			cleanup: func(conn *redis.Client) {},
			want:    false,
			want1:   -1,
			wantErr: true,
		},
		{
			name: "holderID非法",
			args: args{
				resourceID: 123,
				holderID:   0, // 非法值
				keyPrefix:  "test_lock:",
				expiration: 5 * time.Minute,
			},
			setup:   func(conn *redis.Client) {},
			cleanup: func(conn *redis.Client) {},
			want:    false,
			want1:   -1,
			wantErr: true,
		},
		{
			name: "锁已被当前用户持有",
			args: args{
				resourceID: 124,
				holderID:   456,
				keyPrefix:  "test_lock:",
				expiration: 5 * time.Minute,
			},
			setup: func(conn *redis.Client) {
				// 先清理可能存在的键
				_, err := conn.Do(context.Background(), "DEL", "test_lock:124").Result()
				if err != nil {
					t.Fatalf("清理测试键失败: %v", err)
				}
				// 预先设置锁，由当前用户持有
				_, err = conn.Do(context.Background(), "SET", "test_lock:124", 456, "EX", 300).Result()
				if err != nil {
					t.Fatalf("设置测试锁失败: %v", err)
				}
			},
			cleanup: func(conn *redis.Client) {
				cleanupTestKeys(conn, "test_lock:", 124)
			},
			want:    true,
			want1:   -1,
			wantErr: false,
		},
		{
			name: "锁被其他用户持有",
			args: args{
				resourceID: 125,
				holderID:   456,
				keyPrefix:  "test_lock:",
				expiration: 5 * time.Minute,
			},
			setup: func(conn *redis.Client) {
				// 先清理可能存在的键
				_, err := conn.Do(context.Background(), "DEL", "test_lock:125").Result()
				if err != nil {
					t.Fatalf("清理测试键失败: %v", err)
				}
				// 预先设置锁，由其他用户持有
				_, err = conn.Do(context.Background(), "SET", "test_lock:125", 789, "EX", 300).Result()
				if err != nil {
					t.Fatalf("设置测试锁失败: %v", err)
				}
			},
			cleanup: func(conn *redis.Client) {
				cleanupTestKeys(conn, "test_lock:", 125)
			},
			want:    false,
			want1:   789,
			wantErr: true,
		},
		{
			name: "使用默认过期时间",
			args: args{
				resourceID: 126,
				holderID:   456,
				keyPrefix:  "test_lock:",
				expiration: 0, // 使用默认过期时间
			},
			setup: func(conn *redis.Client) {
				// 确保测试前键不存在
				_, err := conn.Do(context.Background(), "DEL", "test_lock:126").Result()
				if err != nil {
					t.Fatalf("清理测试键失败: %v", err)
				}
			},
			cleanup: func(conn *redis.Client) {
				cleanupTestKeys(conn, "test_lock:", 126)
			},
			want:    true,
			want1:   -1,
			wantErr: false,
		},
		{
			name: "执行脚本报错",
			args: args{
				resourceID: 126,
				holderID:   456,
				keyPrefix:  "test_lock:",
				expiration: 0, // 使用默认过期时间
			},
			setup: func(conn *redis.Client) {
				// 确保测试前键不存在
				_, err := conn.Do(context.Background(), "DEL", "test_lock:126").Result()
				if err != nil {
					t.Fatalf("清理测试键失败: %v", err)
				}
				conn.Close()

			},
			cleanup: func(conn *redis.Client) {
				cleanupTestKeys(conn, "test_lock:", 126)
				redisClient = nil
			},
			want:    false,
			want1:   -1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建真实上下文和Redis连接
			ctx, conn := createRealContext()

			// 设置测试环境
			tt.setup(conn)
			//确保测试结束后清理
			defer tt.cleanup(conn)

			// 更新上下文
			tt.args.ctx = ctx

			got, err := TryLock(tt.args.ctx, tt.args.resourceID, tt.args.holderID, tt.args.keyPrefix, tt.args.expiration)
			t.Logf("got:%t", got)
			if (err != nil) != tt.wantErr {
				t.Errorf("TryLock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("TryLock() got = %v, want %v", got, tt.want)
			}
			if err != nil {
				return
			}
			resourceIdStr := strconv.Itoa(int(tt.args.resourceID))
			res, err := conn.Do(context.Background(), "GET", tt.args.keyPrefix+resourceIdStr).Result()
			if err != nil {
				panic(err)
			}
			s, ok := res.(string)
			if !ok {
				t.Fatalf("unexpected type: %T", res)
			}

			val, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			assert.Equal(tt.args.holderID, val)
			t.Logf("res:%T", res)
		})
	}
}

func TestReleaseLock(t *testing.T) {
	assert := assert.New(t)
	ConfigureForTest()
	z = GetLogger()
	type args struct {
		ctx        context.Context
		resourceID int64
		holderID   int64
		keyPrefix  string
	}
	tests := []struct {
		name    string
		args    args
		setup   func(*redis.Client)
		cleanup func(*redis.Client)
		wantErr bool
	}{
		{
			name: "成功释放锁",
			args: args{
				resourceID: 123,
				holderID:   456,
				keyPrefix:  "test_lock:",
			},
			setup: func(conn *redis.Client) {
				// 先清理可能存在的键
				_, err := conn.Do(context.Background(), "DEL", "test_lock:123").Result()
				if err != nil {
					t.Fatalf("清理测试键失败: %v", err)
				}
				// 设置锁，由当前用户持有
				_, err = conn.Do(context.Background(), "SET", "test_lock:123", 456, "EX", 300).Result()
				if err != nil {
					t.Fatalf("设置测试锁失败: %v", err)
				}
			},
			cleanup: func(conn *redis.Client) {
				cleanupTestKeys(conn, "test_lock:", 123)
			},
			wantErr: false,
		},
		{
			name: "resourceID非法",
			args: args{
				resourceID: 0, // 非法值
				holderID:   456,
				keyPrefix:  "test_lock:",
			},
			setup:   func(conn *redis.Client) {},
			cleanup: func(conn *redis.Client) {},
			wantErr: true,
		},
		{
			name: "holderID非法",
			args: args{
				resourceID: 123,
				holderID:   0, // 非法值
				keyPrefix:  "test_lock:",
			},
			setup:   func(conn *redis.Client) {},
			cleanup: func(conn *redis.Client) {},
			wantErr: true,
		},
		{
			name: "锁不存在",
			args: args{
				resourceID: 124,
				holderID:   456,
				keyPrefix:  "test_lock:",
			},
			setup: func(conn *redis.Client) {
				// 确保锁不存在
				_, err := conn.Do(context.Background(), "DEL", "test_lock:124").Result()
				if err != nil {
					t.Fatalf("清理测试键失败: %v", err)
				}
			},
			cleanup: func(conn *redis.Client) {},
			wantErr: true,
		},
		{
			name: "锁被其他用户持有",
			args: args{
				resourceID: 125,
				holderID:   456,
				keyPrefix:  "test_lock:",
			},
			setup: func(conn *redis.Client) {
				// 先清理可能存在的键
				_, err := conn.Do(context.Background(), "DEL", "test_lock:125").Result()
				if err != nil {
					t.Fatalf("清理测试键失败: %v", err)
				}
				// 设置锁，由其他用户持有
				_, err = conn.Do(context.Background(), "SET", "test_lock:125", 789, "EX", 300).Result()
				if err != nil {
					t.Fatalf("设置测试锁失败: %v", err)
				}
			},
			cleanup: func(conn *redis.Client) {
				cleanupTestKeys(conn, "test_lock:", 125)
			},
			wantErr: true,
		},
		{
			name: "执行脚本报错",
			args: args{
				resourceID: 126,
				holderID:   456,
				keyPrefix:  "test_lock:",
			},
			setup: func(conn *redis.Client) {
				// 确保测试前键不存在
				_, err := conn.Do(context.Background(), "DEL", "test_lock:126").Result()
				if err != nil {
					t.Fatalf("清理测试键失败: %v", err)
				}
				conn.Close()
			},
			cleanup: func(conn *redis.Client) {
				cleanupTestKeys(conn, "test_lock:", 126)
				redisClient = nil
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建真实上下文和Redis连接
			ctx, conn := createRealContext()

			// 设置测试环境
			tt.setup(conn)
			// 确保测试结束后清理
			defer tt.cleanup(conn)

			// 更新上下文
			tt.args.ctx = ctx

			err := ReleaseLock(tt.args.ctx, tt.args.resourceID, tt.args.holderID, tt.args.keyPrefix)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReleaseLock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// 如果期望成功释放锁，验证锁是否已被删除
			if !tt.wantErr {
				resourceIdStr := strconv.Itoa(int(tt.args.resourceID))
				res, err := conn.Do(context.Background(), "EXISTS", tt.args.keyPrefix+resourceIdStr).Result()
				if err != nil {
					t.Fatalf("检查锁是否存在失败: %v", err)
				}
				exists, ok := res.(int64)
				if !ok {
					t.Fatalf("unexpected type: %T", res)
				}
				assert.Equal(int64(0), exists, "锁应该已被删除")
			}
		})
	}
}

func TestRefreshLock(t *testing.T) {
	assert := assert.New(t)
	ConfigureForTest()
	z = GetLogger()
	type args struct {
		ctx        context.Context
		resourceID int64
		holderID   int64
		keyPrefix  string
		expiration time.Duration
	}
	tests := []struct {
		name    string
		args    args
		setup   func(*redis.Client)
		cleanup func(*redis.Client)
		wantErr bool
	}{
		{
			name: "成功刷新锁",
			args: args{
				resourceID: 123,
				holderID:   456,
				keyPrefix:  "test_lock:",
				expiration: 10 * time.Minute, // 新的过期时间
			},
			setup: func(conn *redis.Client) {
				// 先清理可能存在的键
				_, err := conn.Do(context.Background(), "DEL", "test_lock:123").Result()
				if err != nil {
					t.Fatalf("清理测试键失败: %v", err)
				}
				// 设置锁，由当前用户持有，初始过期时间为5分钟
				_, err = conn.Do(context.Background(), "SET", "test_lock:123", 456, "EX", 300).Result()
				if err != nil {
					t.Fatalf("设置测试锁失败: %v", err)
				}
			},
			cleanup: func(conn *redis.Client) {
				cleanupTestKeys(conn, "test_lock:", 123)
			},
			wantErr: false,
		},
		{
			name: "resourceID非法",
			args: args{
				resourceID: 0, // 非法值
				holderID:   456,
				keyPrefix:  "test_lock:",
				expiration: 5 * time.Minute,
			},
			setup:   func(conn *redis.Client) {},
			cleanup: func(conn *redis.Client) {},
			wantErr: true,
		},
		{
			name: "holderID非法",
			args: args{
				resourceID: 123,
				holderID:   0, // 非法值
				keyPrefix:  "test_lock:",
				expiration: 5 * time.Minute,
			},
			setup:   func(conn *redis.Client) {},
			cleanup: func(conn *redis.Client) {},
			wantErr: true,
		},
		{
			name: "锁不存在",
			args: args{
				resourceID: 124,
				holderID:   456,
				keyPrefix:  "test_lock:",
				expiration: 5 * time.Minute,
			},
			setup: func(conn *redis.Client) {
				// 确保锁不存在
				_, err := conn.Do(context.Background(), "DEL", "test_lock:124").Result()
				if err != nil {
					t.Fatalf("清理测试键失败: %v", err)
				}
			},
			cleanup: func(conn *redis.Client) {},
			wantErr: true,
		},
		{
			name: "锁被其他用户持有",
			args: args{
				resourceID: 125,
				holderID:   456,
				keyPrefix:  "test_lock:",
				expiration: 5 * time.Minute,
			},
			setup: func(conn *redis.Client) {
				// 先清理可能存在的键
				_, err := conn.Do(context.Background(), "DEL", "test_lock:125").Result()
				if err != nil {
					t.Fatalf("清理测试键失败: %v", err)
				}
				// 设置锁，由其他用户持有
				_, err = conn.Do(context.Background(), "SET", "test_lock:125", 789, "EX", 300).Result()
				if err != nil {
					t.Fatalf("设置测试锁失败: %v", err)
				}
			},
			cleanup: func(conn *redis.Client) {
				cleanupTestKeys(conn, "test_lock:", 125)
			},
			wantErr: true,
		},
		{
			name: "使用默认过期时间",
			args: args{
				resourceID: 126,
				holderID:   456,
				keyPrefix:  "test_lock:",
				expiration: 0, // 使用默认过期时间
			},
			setup: func(conn *redis.Client) {
				// 先清理可能存在的键
				_, err := conn.Do(context.Background(), "DEL", "test_lock:126").Result()
				if err != nil {
					t.Fatalf("清理测试键失败: %v", err)
				}
				// 设置锁，由当前用户持有
				_, err = conn.Do(context.Background(), "SET", "test_lock:126", 456, "EX", 300).Result()
				if err != nil {
					t.Fatalf("设置测试锁失败: %v", err)
				}
			},
			cleanup: func(conn *redis.Client) {
				cleanupTestKeys(conn, "test_lock:", 126)
			},
			wantErr: false,
		},
		{
			name: "执行脚本报错",
			args: args{
				resourceID: 127,
				holderID:   456,
				keyPrefix:  "test_lock:",
				expiration: 5 * time.Minute,
			},
			setup: func(conn *redis.Client) {
				// 确保测试前键不存在
				_, err := conn.Do(context.Background(), "DEL", "test_lock:127").Result()
				if err != nil {
					t.Fatalf("清理测试键失败: %v", err)
				}
				conn.Close()
			},
			cleanup: func(conn *redis.Client) {
				cleanupTestKeys(conn, "test_lock:", 127)
				redisClient = nil
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建真实上下文和Redis连接
			ctx, conn := createRealContext()

			// 设置测试环境
			tt.setup(conn)
			// 确保测试结束后清理
			defer tt.cleanup(conn)

			// 更新上下文
			tt.args.ctx = ctx

			// 记录刷新前的TTL
			var beforeTTL int64 = -1
			if !tt.wantErr && tt.name != "锁不存在" {
				resourceIdStr := strconv.Itoa(int(tt.args.resourceID))
				res, err := conn.Do(context.Background(), "TTL", tt.args.keyPrefix+resourceIdStr).Result()
				if err != nil {
					t.Fatalf("获取锁TTL失败: %v", err)
				}
				var ok bool
				beforeTTL, ok = res.(int64)
				if !ok {
					t.Fatalf("unexpected type: %T", res)
				}
			}

			err := RefreshLock(tt.args.ctx, tt.args.resourceID, tt.args.holderID, tt.args.keyPrefix, tt.args.expiration)
			if (err != nil) != tt.wantErr {
				t.Errorf("RefreshLock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// 如果期望成功刷新锁，验证锁的TTL是否已更新
			if !tt.wantErr {
				resourceIdStr := strconv.Itoa(int(tt.args.resourceID))
				res, err := conn.Do(context.Background(), "TTL", tt.args.keyPrefix+resourceIdStr).Result()
				if err != nil {
					t.Fatalf("获取锁TTL失败: %v", err)
				}
				afterTTL, ok := res.(int64)
				if !ok {
					t.Fatalf("unexpected type: %T", res)
				}

				// 验证TTL是否已更新（应该大于或等于之前的TTL）
				if tt.args.expiration > 0 {
					// 如果指定了过期时间，验证TTL是否接近指定的值
					expectedTTL := int64(tt.args.expiration.Seconds())
					// 允许1秒的误差
					assert.InDelta(expectedTTL, afterTTL, 1, "锁的TTL应该已更新为指定的过期时间")
				} else {
					// 如果使用默认过期时间，验证TTL是否已更新且大于之前的值
					assert.GreaterOrEqual(afterTTL, beforeTTL, "锁的TTL应该已更新且大于之前的值")
				}

				// 验证锁的持有者是否未变
				res, err = conn.Do(context.Background(), "GET", tt.args.keyPrefix+resourceIdStr).Result()
				if err != nil {
					t.Fatalf("获取锁持有者失败: %v", err)
				}
				s, ok := res.(string)
				if !ok {
					t.Fatalf("unexpected type: %T", res)
				}

				val, err := strconv.ParseInt(s, 10, 64)
				if err != nil {
					t.Fatalf("parse error: %v", err)
				}
				assert.Equal(tt.args.holderID, val, "锁的持有者不应该改变")
			}
		})
	}
}

func TestGetLockHolder(t *testing.T) {
	assert := assert.New(t)
	ConfigureForTest()
	z = GetLogger()
	type args struct {
		ctx        context.Context
		resourceID int64
		keyPrefix  string
	}
	tests := []struct {
		name    string
		args    args
		setup   func(*redis.Client)
		cleanup func(*redis.Client)
		want    int64
		wantErr bool
	}{
		{
			name: "成功获取锁持有者",
			args: args{
				resourceID: 123,
				keyPrefix:  "test_lock:",
			},
			setup: func(conn *redis.Client) {
				// 先清理可能存在的键
				_, err := conn.Do(context.Background(), "DEL", "test_lock:123").Result()
				if err != nil {
					t.Fatalf("清理测试键失败: %v", err)
				}
				// 设置锁，由指定用户持有
				_, err = conn.Do(context.Background(), "SET", "test_lock:123", 456, "EX", 300).Result()
				if err != nil {
					t.Fatalf("设置测试锁失败: %v", err)
				}
			},
			cleanup: func(conn *redis.Client) {
				cleanupTestKeys(conn, "test_lock:", 123)
			},
			want:    456, // 期望返回的持有者ID
			wantErr: false,
		},
		{
			name: "resourceID非法",
			args: args{
				resourceID: 0, // 非法值
				keyPrefix:  "test_lock:",
			},
			setup:   func(conn *redis.Client) {},
			cleanup: func(conn *redis.Client) {},
			want:    -1,
			wantErr: true,
		},
		{
			name: "锁不存在",
			args: args{
				resourceID: 124,
				keyPrefix:  "test_lock:",
			},
			setup: func(conn *redis.Client) {
				// 确保锁不存在
				_, err := conn.Do(context.Background(), "DEL", "test_lock:124").Result()
				if err != nil {
					t.Fatalf("清理测试键失败: %v", err)
				}
			},
			cleanup: func(conn *redis.Client) {},
			want:    -1,
			wantErr: true,
		},
		{
			name: "执行命令报错",
			args: args{
				resourceID: 125,
				keyPrefix:  "test_lock:",
			},
			setup: func(conn *redis.Client) {
				// 确保测试前键不存在
				_, err := conn.Do(context.Background(), "DEL", "test_lock:125").Result()
				if err != nil {
					t.Fatalf("清理测试键失败: %v", err)
				}

			},
			cleanup: func(conn *redis.Client) {
				cleanupTestKeys(conn, "test_lock:", 125)
			},
			want:    -1,
			wantErr: true,
		},
		{
			name: "执行脚本报错",
			args: args{
				resourceID: 125,
				keyPrefix:  "test_lock:",
			},
			setup: func(conn *redis.Client) {
				// 确保测试前键不存在
				_, err := conn.Do(context.Background(), "DEL", "test_lock:125").Result()
				if err != nil {
					t.Fatalf("清理测试键失败: %v", err)
				}
				conn.Close()

			},
			cleanup: func(conn *redis.Client) {
				cleanupTestKeys(conn, "test_lock:", 125)
				redisClient = nil
			},
			want:    -1,
			wantErr: true,
		},
		{
			name: "str转int错误",
			args: args{
				resourceID: 125,
				keyPrefix:  "test_lock:",
			},
			setup: func(conn *redis.Client) {
				// 确保测试前键不存在
				_, err := conn.Do(context.Background(), "DEL", "test_lock:125").Result()
				if err != nil {
					t.Fatalf("清理测试键失败: %v", err)
				}
				// 设置锁，由指定用户持有
				_, err = conn.Do(context.Background(), "SET", "test_lock:125", "test", "EX", 300).Result()
				if err != nil {
					t.Fatalf("设置测试锁失败: %v", err)
				}

			},
			cleanup: func(conn *redis.Client) {
				cleanupTestKeys(conn, "test_lock:", 125)
				redisClient = nil
			},
			want:    -1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建真实上下文和Redis连接
			ctx, conn := createRealContext()

			// 设置测试环境
			tt.setup(conn)
			// 确保测试结束后清理
			defer tt.cleanup(conn)

			// 更新上下文
			tt.args.ctx = ctx

			got, err := GetLockHolder(tt.args.ctx, tt.args.resourceID, tt.args.keyPrefix)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetLockHolder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// 验证返回的持有者ID是否符合预期
			assert.Equal(tt.want, got, "返回的持有者ID应该符合预期")
		})
	}
}
