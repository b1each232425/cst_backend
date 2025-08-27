package cmn

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

// TestNewRateLimiterTaskRunner 测试构造函数
func TestNewRateLimiterTaskRunner(t *testing.T) {
	runner := NewRateLimiterTaskRunner(10, 5.0, 3)

	assert.NotNil(t, runner)
	assert.NotNil(t, runner.sem)
	assert.NotNil(t, runner.limiter)

	// 验证速率限制器参数
	expectedLimit := rate.Limit(5.0)
	assert.Equal(t, expectedLimit, runner.limiter.Limit())
	assert.Equal(t, 3, runner.limiter.Burst())
}

// TestTryRun_Success 测试正常执行任务的情况
func TestTryRun_Success(t *testing.T) {
	runner := NewRateLimiterTaskRunner(5, 100, 10) // 高QPS和并发数确保不会阻塞

	taskCalled := false
	task := func() error {
		taskCalled = true
		return nil
	}

	onSkipCalled := false
	onSkip := func(reason string) {
		onSkipCalled = true
	}

	result := runner.TryRun(task, 1*time.Second, onSkip)

	assert.True(t, result)        // 应该执行成功
	assert.True(t, taskCalled)    // 任务应该被调用
	assert.False(t, onSkipCalled) // 不应该触发跳过回调
}

// TestTryRun_TaskExecutionError 测试任务执行失败的情况
func TestTryRun_TaskExecutionError(t *testing.T) {
	runner := NewRateLimiterTaskRunner(5, 100, 10)

	expectedError := errors.New("task failed")
	task := func() error {
		return expectedError
	}

	skipReason := ""
	onSkip := func(reason string) {
		skipReason = reason
	}

	result := runner.TryRun(task, 1*time.Second, onSkip)

	assert.False(t, result) // 执行应该失败
	assert.Contains(t, skipReason, "task execution error")
	assert.Contains(t, skipReason, expectedError.Error())
}

// TestTryRun_RateLimitTimeout 测试速率限制超时的情况
func TestTryRun_RateLimitTimeout(t *testing.T) {
	// 设置较低的QPS
	runner := NewRateLimiterTaskRunner(5, 1, 1) // 1 QPS，突发容量1

	// 预先消耗掉所有令牌
	ctx := context.Background()
	// 获取唯一的令牌
	_ = runner.limiter.Wait(ctx)

	task := func() error {
		return nil
	}

	skipReason := ""
	onSkip := func(reason string) {
		skipReason = reason
	}

	// 使用很短的超时时间来触发超时
	result := runner.TryRun(task, 1*time.Millisecond, onSkip)

	assert.False(t, result) // 应该执行失败
	assert.Contains(t, skipReason, "rate limit acquire timeout or cancelled")
}

// TestTryRun_SemaphoreTimeout 测试并发限制超时的情况
func TestTryRun_SemaphoreTimeout(t *testing.T) {
	// 设置最大并发为1，并先占用它
	runner := NewRateLimiterTaskRunner(1, 100, 10)

	// 占用唯一的并发槽
	go func() {
		_ = runner.sem.Acquire(context.Background(), 1)
	}()

	// 等待上面的goroutine获取信号量
	time.Sleep(10 * time.Millisecond)

	task := func() error {
		return nil
	}

	skipReason := ""
	onSkip := func(reason string) {
		skipReason = reason
	}

	// 尝试在没有可用并发的情况下运行任务
	result := runner.TryRun(task, 10*time.Millisecond, onSkip)

	assert.False(t, result) // 应该执行失败
	assert.Contains(t, skipReason, "semaphore acquire error")
}

// TestTryRun_ConcurrentExecution 测试并发执行控制
func TestTryRun_ConcurrentExecution(t *testing.T) {
	maxConcurrency := int64(3)
	runner := NewRateLimiterTaskRunner(maxConcurrency, 100, 10)

	var activeTasks int64
	var peakConcurrency int64
	var taskCount = 10

	// 创建多个并发任务
	for i := 0; i < taskCount; i++ {
		go func() {
			task := func() error {
				// 增加活跃任务计数
				current := atomic.AddInt64(&activeTasks, 1)
				// 更新峰值并发数
				for {
					peak := atomic.LoadInt64(&peakConcurrency)
					if current > peak {
						if atomic.CompareAndSwapInt64(&peakConcurrency, peak, current) {
							break
						}
					} else {
						break
					}
				}

				// 模拟任务执行时间
				time.Sleep(50 * time.Millisecond)

				// 减少活跃任务计数
				atomic.AddInt64(&activeTasks, -1)

				return nil
			}

			runner.TryRun(task, 1*time.Second, nil)
		}()
	}

	// 等待所有任务完成
	time.Sleep(200 * time.Millisecond)

	finalPeak := atomic.LoadInt64(&peakConcurrency)

	// 验证并发数没有超过限制
	assert.True(t, finalPeak <= maxConcurrency,
		"Peak concurrency %d should not exceed max concurrency %d",
		finalPeak, maxConcurrency)
}

// TestTryRun_SkipCallbackNotCalledWhenNil 测试onSkip为nil时不崩溃
func TestTryRun_SkipCallbackNotCalledWhenNil(t *testing.T) {
	runner := NewRateLimiterTaskRunner(0, 100, 10) // 0并发确保任务会被跳过

	task := func() error {
		return nil
	}

	// 不应该出现panic
	result := runner.TryRun(task, 10*time.Millisecond, nil)

	assert.False(t, result) // 应该执行失败，但不调用回调
}

func TestRateLimiterTaskRunner_Wait_Success(t *testing.T) {
	// 创建一个具有足够资源的限流器
	runner := NewRateLimiterTaskRunner(5, 100.0, 10)

	// 使用带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// 应该能成功获取资源
	err, releaseFunc := runner.Wait(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, releaseFunc)

	// 验证资源释放函数能正常调用
	releaseFunc()
}

func TestRateLimiterTaskRunner_Wait_Success_2(t *testing.T) {
	// 创建一个具有足够资源的限流器
	runner := NewRateLimiterTaskRunner(5, 2, 2)

	// 使用带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			err, releaseFunc := runner.Wait(ctx)

			assert.NoError(t, err)
			assert.NotNil(t, releaseFunc)

			time.Sleep(1 * time.Second)

			// 验证资源释放函数能正常调用
			releaseFunc()

			t.Logf("[SUCCESS] Task %d executed at %v\n", i, time.Now().Format("15:04:05.000"))
			defer wg.Done()
		}()
	}

	wg.Wait()

}

func TestRateLimiterTaskRunner_Wait_RateLimitTimeout(t *testing.T) {
	// 创建一个极低QPS的限流器
	runner := NewRateLimiterTaskRunner(5, 0.000001, 1) // 极低的QPS

	// 预先消耗令牌
	_ = runner.limiter.Wait(context.Background())

	// 使用很短的超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// 应该因为速率限制超时而失败
	err, releaseFunc := runner.Wait(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rate limit acquire timeout or cancelled")
	assert.NotNil(t, releaseFunc)
}

func TestRateLimiterTaskRunner_Wait_SemaphoreTimeout(t *testing.T) {
	// 创建一个只能并发1个任务的限流器
	runner := NewRateLimiterTaskRunner(1, 100.0, 10)

	// 占用唯一的并发槽
	ctx := context.Background()
	_ = runner.sem.Acquire(ctx, 1)

	// 使用很短的超时时间
	waitCtx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// 应该因为信号量获取超时而失败
	err, releaseFunc := runner.Wait(waitCtx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "semaphore acquire error")
	assert.NotNil(t, releaseFunc)
}

func TestRateLimiterTaskRunner_Wait_ContextCancelled(t *testing.T) {
	runner := NewRateLimiterTaskRunner(5, 100.0, 10)

	// 创建一个已取消的上下文
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	// 应该因为上下文取消而失败
	err, releaseFunc := runner.Wait(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rate limit acquire timeout or cancelled")
	assert.NotNil(t, releaseFunc)
}

func TestRateLimiterTaskRunner_Wait_releaseFuncReleasesSemaphore(t *testing.T) {
	// 创建一个只能并发1个任务的限流器
	runner := NewRateLimiterTaskRunner(1, 100.0, 10)

	// 第一次获取资源应该成功
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err, releaseFunc := runner.Wait(ctx)
	assert.NoError(t, err)

	// 尝试再次获取资源应该失败（因为并发限制）
	waitCtx, waitCancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer waitCancel()

	err2, _ := runner.Wait(waitCtx)
	assert.Error(t, err2)
	assert.Contains(t, err2.Error(), "semaphore acquire error")

	// 释放第一次获取的资源
	releaseFunc()

	// 现在应该能再次获取资源
	waitCtx2, waitCancel2 := context.WithTimeout(context.Background(), 1*time.Second)
	defer waitCancel2()

	err3, releaseFunc3 := runner.Wait(waitCtx2)
	assert.NoError(t, err3)
	assert.NotNil(t, releaseFunc3)

	// 清理资源
	releaseFunc3()
}
