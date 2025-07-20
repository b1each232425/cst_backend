package cmn

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gomodule/redigo/redis"
)

const (
	// defaultLockExpiration 是分布式锁的默认过期时间（5分钟）。
	defaultLockExpiration = 5 * time.Minute

	luaLock = `local current = redis.call("GET", KEYS[1])
if not current then
  redis.call("SET", KEYS[1], ARGV[1])
  redis.call("EXPIRE", KEYS[1], ARGV[2])
  return 1
elseif current == ARGV[1] then
  redis.call("EXPIRE", KEYS[1], ARGV[2])
  return 1
else
  return 0
end`

	luaUnlock = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
else
  return 0
end
`
	luaRefresh = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("EXPIRE", KEYS[1], ARGV[2])
else
  return 0
end`
)

// TryLock 尝试为指定资源加分布式锁。
//
// 参数：
//
//	ctx        —— 上下文（用于获取 Redis 连接）
//	resourceID —— 资源唯一ID（如文档ID、试卷ID等，必须为正数）
//	holderID   —— 锁持有者ID（如用户ID，必须为正数）
//	keyPrefix  —— 锁的 Redis key 前缀（如 "paper_lock:"）
//	expiration —— 锁的过期时间（<=0 时使用默认值）
//
// 返回值：
//
//	bool  —— 是否成功获取锁
//	int64 —— 如果锁被占用，返回当前持有者ID，否则为 -1
//	error —— 错误信息（如参数非法、Redis 操作失败等）
//
// 注意：
//   - 若锁已被当前 holderID 持有，则自动刷新锁并返回 true。
//   - 若锁被其他用户持有，返回 false、持有者ID 和错误信息。
//   - 若 resourceID 或 holderID 非正数，直接返回错误。
func TryLock(ctx context.Context, resourceID, holderID int64, keyPrefix string, expiration time.Duration) (bool, error) {
	// 参数校验，resourceID 和 holderID 必须为正数
	if resourceID <= 0 {
		err := fmt.Errorf("resourceID must be positive")
		z.Error(err.Error())
		return false, err
	}
	if holderID <= 0 {
		err := fmt.Errorf("holderID must be positive")
		z.Error(err.Error())
		return false, err
	}
	// 过期时间校验，若未指定则使用默认值
	if expiration <= 0 {
		expiration = defaultLockExpiration
	}

	q := GetCtxValue(ctx)
	key := fmt.Sprintf("%s%d", keyPrefix, resourceID)

	// 原子执行 Lua
	res, err := redis.Int(
		q.Redis.Do("EVAL", luaLock,
			1,                         // 1 个 key
			key,                       // KEYS[1]
			holderID,                  // ARGV[1]
			int(expiration.Seconds()), // ARGV[2]
		),
	)
	if err != nil {
		z.Error(err.Error())
		return false, err
	}
	if res == 1 {
		return true, nil
	} else {
		err := fmt.Errorf("key %s is locked", key)
		z.Error(err.Error())
		return false, err
	}

}

// ReleaseLock 释放指定资源的分布式锁（仅限锁持有者）。
//
// 参数：
//
//	ctx        —— 上下文（用于获取 Redis 连接）
//	resourceID —— 资源唯一ID
//	holderID   —— 期望的锁持有者ID
//	keyPrefix  —— 锁的 Redis key 前缀
//
// 返回值：
//
//	error —— 释放成功返回 nil，失败返回错误信息
//
// 注意：
//   - 只有当前持有者才能释放锁，否则返回错误。
//   - 若锁不存在，返回 redis.ErrNil。
func ReleaseLock(ctx context.Context, resourceID, holderID int64, keyPrefix string) error {
	q := GetCtxValue(ctx)
	key := fmt.Sprintf("%s%d", keyPrefix, resourceID)

	res, err := redis.Int(q.Redis.Do("EVAL", luaUnlock, 1, key, holderID))
	if err != nil {
		return err
	}
	if res == 0 {
		return errors.New("lock not held by current client")
	}
	return nil
}

// RefreshLock 刷新指定资源锁的过期时间（仅限锁持有者）。
//
// 参数：
//
//	ctx        —— 上下文（用于获取 Redis 连接）
//	resourceID —— 资源唯一ID
//	holderID   —— 期望的锁持有者ID
//	keyPrefix  —— 锁的 Redis key 前缀
//	expiration —— 新的过期时间
//
// 返回值：
//
//	error —— 刷新成功返回 nil，失败返回错误信息
//
// 注意：
//   - 只有当前持有者才能刷新锁，否则返回错误。
//   - 若锁不存在，返回 redis.ErrNil。
func RefreshLock(ctx context.Context,
	resourceID, holderID int64,
	keyPrefix string,
	expiration time.Duration) error {

	q := GetCtxValue(ctx)
	key := fmt.Sprintf("%s%d", keyPrefix, resourceID)

	if int(expiration.Seconds()) <= 0 {
		expiration = defaultLockExpiration
	}

	// 原子执行 Lua
	res, err := redis.Int(
		q.Redis.Do("EVAL", luaRefresh,
			1,                         // 1 个 key
			key,                       // KEYS[1]
			holderID,                  // ARGV[1]
			int(expiration.Seconds()), // ARGV[2]
		),
	)

	switch {
	case err != nil:
		z.Error("RefreshLock eval error:" + err.Error())
		return err
	case res == 0:
		// 两种情况：锁不存在，或者持有者不匹配
		err = fmt.Errorf("RefreshLock: lock not exist or not held by %d", holderID)
		z.Info(err.Error())
		return err
	default:
		return nil
	}
}

// GetLockHolder 获取指定资源锁的当前持有者ID。
//
// 参数：
//
//	ctx        —— 上下文（用于获取 Redis 连接）
//	resourceID —— 资源唯一ID
//	keyPrefix  —— 锁的 Redis key 前缀
//
// 返回值：
//
//	int64 —— 当前锁持有者ID，若锁不存在返回 -1
//	error —— 获取成功返回 nil，失败返回错误信息
//
// 注意：
//   - 若锁不存在，返回 redis.ErrNil。
func GetLockHolder(ctx context.Context, resourceID int64, keyPrefix string) (int64, error) {
	q := GetCtxValue(ctx)
	key := fmt.Sprintf("%s%d", keyPrefix, resourceID)

	// 查询锁持有者ID
	holder, err := redis.Int64(q.Redis.Do("GET", key))
	if err != nil {
		// 锁不存在，直接返回 redis.ErrNil
		if errors.Is(err, redis.ErrNil) {
			z.Info(err.Error())
			return -1, err
		}
		z.Error(err.Error())
		return -1, err
	}
	return holder, nil
}
