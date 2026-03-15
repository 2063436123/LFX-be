package server

import (
	"net/http"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// RateLimiter 简单的内存速率限制器
type RateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int           // 最大请求次数
	window   time.Duration // 时间窗口
}

// NewRateLimiter 创建速率限制器
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
	// 定期清理过期数据
	go rl.cleanup()
	return rl
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// 获取该 key 的请求时间
	times, ok := rl.requests[key]
	if !ok {
		rl.requests[key] = []time.Time{now}
		return true
	}

	// 过滤掉过期的请求
	valid := make([]time.Time, 0, len(times))
	for _, t := range times {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	// 检查是否超过限制
	if len(valid) >= rl.limit {
		rl.requests[key] = valid
		return false
	}

	rl.requests[key] = append(valid, now)
	return true
}

// cleanup 定期清理过期数据
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		cutoff := now.Add(-rl.window)
		for key, times := range rl.requests {
			valid := make([]time.Time, 0, len(times))
			for _, t := range times {
				if t.After(cutoff) {
					valid = append(valid, t)
				}
			}
			if len(valid) == 0 {
				delete(rl.requests, key)
			} else {
				rl.requests[key] = valid
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimitMiddleware 创建速率限制中间件
func (s *Server) RateLimitMiddleware(limiter *RateLimiter, getClientIP func(*core.RequestEvent) string) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		key := getClientIP(e)
		if key == "" {
			key = "unknown"
		}
		if !limiter.Allow(key) {
			return e.JSON(http.StatusTooManyRequests, map[string]string{"error": "too many requests, please try again later"})
		}
		return nil
	}
}

// getAuthLoginLimiter 获取登录端点的速率限制器（更严格的限制）
func (s *Server) getAuthLoginLimiter() *RateLimiter {
	// 每分钟最多 5 次登录尝试
	return NewRateLimiter(5, time.Minute)
}

// getGeneralLimiter 获取通用端点的速率限制器
func (s *Server) getGeneralLimiter() *RateLimiter {
	// 每分钟最多 60 次请求
	return NewRateLimiter(60, time.Minute)
}
