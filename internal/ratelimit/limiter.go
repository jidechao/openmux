package ratelimit

import (
	"sync"
	"time"
)

// Limiter 限流器接口
type Limiter interface {
	Allow() bool
	AllowN(n int) bool
}

// TokenBucket Token Bucket 限流器
type TokenBucket struct {
	mu           sync.Mutex
	capacity     int
	tokens       float64
	refillRate   float64
	lastRefill   time.Time
}

// NewTokenBucket 创建 Token Bucket 限流器
func NewTokenBucket(rpm int) *TokenBucket {
	capacity := rpm
	if capacity <= 0 {
		capacity = 1000000 // 无限制
	}
	
	return &TokenBucket{
		capacity:   capacity,
		tokens:     float64(capacity),
		refillRate: float64(capacity) / 60.0, // 每秒补充的 token 数
		lastRefill: time.Now(),
	}
}

// Allow 检查是否允许请求
func (tb *TokenBucket) Allow() bool {
	return tb.AllowN(1)
}

// AllowN 检查是否允许 n 个请求
func (tb *TokenBucket) AllowN(n int) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	
	// 补充 token
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens += elapsed * tb.refillRate
	if tb.tokens > float64(tb.capacity) {
		tb.tokens = float64(tb.capacity)
	}
	tb.lastRefill = now
	
	// 检查是否有足够的 token
	if tb.tokens >= float64(n) {
		tb.tokens -= float64(n)
		return true
	}
	
	return false
}

// MultiLimiter 多维度限流器
type MultiLimiter struct {
	rpmLimiter *TokenBucket
	tpmLimiter *TokenBucket
}

// NewMultiLimiter 创建多维度限流器
func NewMultiLimiter(rpm, tpm int) *MultiLimiter {
	return &MultiLimiter{
		rpmLimiter: NewTokenBucket(rpm),
		tpmLimiter: NewTokenBucket(tpm),
	}
}

// Allow 检查是否允许请求
func (ml *MultiLimiter) Allow() bool {
	return ml.rpmLimiter.Allow()
}

// AllowTokens 检查是否允许使用指定数量的 token
func (ml *MultiLimiter) AllowTokens(tokens int) bool {
	if !ml.rpmLimiter.Allow() {
		return false
	}
	if !ml.tpmLimiter.AllowN(tokens) {
		return false
	}
	return true
}
