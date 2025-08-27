package cmn

import (
	"context"
	"fmt"
	"golang.org/x/sync/semaphore"
	"golang.org/x/time/rate"
	"time"
)

// TaskFunc 是用户定义的要执行的任务函数，比如调用第三方 API
type TaskFunc func() error

// SkipFunc 是当任务被跳过时触发的回调（比如超时未能获取资源）
type SkipFunc func(reason string)

// RateLimiterTaskRunner 封装了并发数和速率的双重限流逻辑
type RateLimiterTaskRunner struct {
	sem     *semaphore.Weighted // 控制并发
	limiter *rate.Limiter       // 控制速率
}

// NewRateLimiterTaskRunner 创建一个新的限流任务执行器
// maxConcurrency: 最大并发数
// qps: 每秒允许的请求数
// burst: 令牌桶突发容量
func NewRateLimiterTaskRunner(maxConcurrency int64, qps float64, burst int) *RateLimiterTaskRunner {
	return &RateLimiterTaskRunner{
		sem:     semaphore.NewWeighted(maxConcurrency),
		limiter: rate.NewLimiter(rate.Limit(qps), burst),
	}
}

// TryRun 尝试运行一个任务，受并发数和速率限制，并可设置超时
// task: 要执行的任务函数
// acquireTimeout: 获取信号量和令牌的最长等待时间
// onSkip: 当任务由于超时等原因被跳过时执行的回调
// 返回值：是否成功执行了任务
func (r *RateLimiterTaskRunner) TryRun(
	task TaskFunc,
	acquireTimeout time.Duration,
	onSkip SkipFunc,
) bool {
	ctx := context.Background()
	var err error

	// --- 1. 速率控制（带超时） ---
	limiterCtx, cancelLimiter := context.WithTimeout(ctx, acquireTimeout)
	defer cancelLimiter()

	err = r.limiter.Wait(limiterCtx)
	if err != nil {
		if onSkip != nil {
			onSkip(fmt.Sprintf("rate limit acquire timeout or cancelled: %v", err))
		}
		return false
	}

	// --- 2. 并发控制（带超时） ---
	semCtx, cancelSem := context.WithTimeout(ctx, acquireTimeout)
	defer cancelSem()

	err = r.sem.Acquire(semCtx, 1)
	if err != nil {
		if onSkip != nil {
			onSkip(fmt.Sprintf("semaphore acquire error: %v", err))
		}
		return false
	}
	// 确保释放信号量
	defer r.sem.Release(1)

	// --- 3. 执行实际任务 ---
	err = task()
	if err != nil {
		if onSkip != nil {
			onSkip(fmt.Sprintf("task execution error: %v", err))
		}
		return false
	}

	// 执行成功
	return true
}

// Wait 等待获取执行任务所需的资源，包括速率限制和并发控制
// 参数:
//
//	ctx: 上下文，用于控制等待超时和取消
//
// 返回值:
//
//	error: 等待过程中出现的错误，如超时或被取消
//	func(): 释放函数，用于释放已获取的资源
func (r *RateLimiterTaskRunner) Wait(ctx context.Context) (error, func()) {
	var releaseFunc = func() {
		r.sem.Release(1)
	}

	// 1. 先等待速率限制器
	if err := r.limiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limit acquire timeout or cancelled: %w", err), releaseFunc
	}

	// 2. 再等待并发信号量
	if err := r.sem.Acquire(ctx, 1); err != nil {
		return fmt.Errorf("semaphore acquire error: %w", err), releaseFunc
	}

	return nil, releaseFunc
}
