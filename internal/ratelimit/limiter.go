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

// Return 归还 token
func (tb *TokenBucket) Return(n float64) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	
	tb.tokens += n
	if tb.tokens > float64(tb.capacity) {
		tb.tokens = float64(tb.capacity)
	}
}

// Consume 强制消耗 token (允许透支)
func (tb *TokenBucket) Consume(n float64) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	
	// 先更新
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens += elapsed * tb.refillRate
	if tb.tokens > float64(tb.capacity) {
		tb.tokens = float64(tb.capacity)
	}
	tb.lastRefill = now

	tb.tokens -= n
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

// Allow 检查是否允许请求 (RPM check)
func (ml *MultiLimiter) Allow() bool {
	return ml.rpmLimiter.Allow()
}

// Reserve 预留 Token
func (ml *MultiLimiter) Reserve(estimatedTokens int) bool {
	if !ml.rpmLimiter.Allow() {
		return false
	}
	// 如果配置了 TPM 限制 (tpm > 0)，则检查 TPM
	// 如果 TPM 很大 (默认值)，通常认为不限制
	if ml.tpmLimiter.capacity < 1000000 { 
		if !ml.tpmLimiter.AllowN(estimatedTokens) {
			// 如果 TPM 不足，记得归还 RPM
			ml.rpmLimiter.Return(1)
			return false
		}
	}
	return true
}

// Update 更新实际消耗
func (ml *MultiLimiter) Update(usedTokens, estimatedTokens int) {
	diff := float64(estimatedTokens - usedTokens)
	if diff > 0 {
		// 预估多了，归还
		ml.tpmLimiter.Return(diff)
	} else if diff < 0 {
		// 预估少了，补扣 (允许透支)
		ml.tpmLimiter.Consume(-diff)
	}
}

