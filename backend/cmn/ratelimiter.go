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

// ContentSizeFunc 是用于获取当前任务的内容大小的函数，单位：字节
type ContentSizeFunc func() int

// RateLimiterTaskRunner 封装了并发数和速率的双重限流逻辑
type RateLimiterTaskRunner struct {
	sem               *semaphore.Weighted // 控制并发
	limiter           *rate.Limiter       // 控制速率
	contentLimiter    *rate.Limiter       // 控制每分钟内容大小（比如 20万字节/分钟）
	contentSizeGetter ContentSizeFunc     // 获取当前任务的内容大小（字节）
	maxContentSize    int                 // 最大允许的内容大小，比如 400_000 字节/分钟
}

// NewRateLimiterTaskRunner 创建一个新的限流任务执行器
// maxConcurrency: 最大并发数
// qps: 每秒允许的请求数
// burst: 令牌桶突发容量
func NewRateLimiterTaskRunner(maxConcurrency int64, qps float64, burst int) *RateLimiterTaskRunner {
	contentLimiter := rate.NewLimiter(
		rate.Limit(float64(40_000)/60.0), // 每秒允许的令牌数 = 总大小 / 60秒
		400_000,                          // burst 最好 >= contentMaxBytesPerMinute，允许短时间突发
	)
	return &RateLimiterTaskRunner{
		sem:               semaphore.NewWeighted(maxConcurrency),
		limiter:           rate.NewLimiter(rate.Limit(qps), burst),
		contentLimiter:    contentLimiter,
		contentSizeGetter: nil,
		maxContentSize:    400_000,
	}
}

// SetContentSizeGetter 设置一个方法，用于在运行时获取任务的内容大小（字节）
func (r *RateLimiterTaskRunner) SetContentSizeGetter(getter ContentSizeFunc) *RateLimiterTaskRunner {
	r.contentSizeGetter = getter
	return r
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

	// --- 2. 内容大小限流（每分钟 40万Token为例）---
	if r.contentSizeGetter != nil {
		contentSize := r.contentSizeGetter()
		// 使用 WaitN 等待对应字节数的令牌
		if err := r.contentLimiter.WaitN(ctx, contentSize); err != nil {
			return fmt.Errorf("content size limit acquire timeout or cancelled: %w", err), releaseFunc
		}
	}

	// 2. 再等待并发信号量
	if err := r.sem.Acquire(ctx, 1); err != nil {
		return fmt.Errorf("semaphore acquire error: %w", err), releaseFunc
	}

	return nil, releaseFunc
}
