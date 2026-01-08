package balancer

import (
	"sync/atomic"

	"github.com/openmux/openmux/internal/config"
	"github.com/openmux/openmux/internal/ratelimit"
)

// Backend 后端实例
type Backend struct {
	Provider    string
	APIKey      string
	RateLimit   config.RateLimit
	Limiter     *ratelimit.MultiLimiter
	Weight      int
	Healthy     bool
	ActiveConns int32
	FailCount   int32
}

// Balancer 负载均衡器接口
type Balancer interface {
	// Select 选择一个后端
	Select(estimatedTokens int) (*Backend, error)
	
	// Release 释放后端连接
	Release(backend *Backend, usedTokens, estimatedTokens int)
	
	// MarkUnhealthy 标记后端不健康
	MarkUnhealthy(backend *Backend)
	
	// MarkHealthy 标记后端健康
	MarkHealthy(backend *Backend)
	
	// GetBackends 获取所有后端
	GetBackends() []*Backend
}

// AcquireConn 获取连接
func (b *Backend) AcquireConn(estimatedTokens int) bool {
	if !b.Healthy {
		return false
	}
	if b.RateLimit.Concurrent > 0 {
		current := atomic.LoadInt32(&b.ActiveConns)
		if current >= int32(b.RateLimit.Concurrent) {
			return false
		}
	}
	
	// 检查限流 (RPM/TPM)
	if b.Limiter != nil {
		if !b.Limiter.Reserve(estimatedTokens) {
			return false
		}
	}
	
	atomic.AddInt32(&b.ActiveConns, 1)
	return true
}

// ReleaseConn 释放连接
func (b *Backend) ReleaseConn(usedTokens, estimatedTokens int) {
	atomic.AddInt32(&b.ActiveConns, -1)
	if b.Limiter != nil {
		b.Limiter.Update(usedTokens, estimatedTokens)
	}
}

// IncrFailCount 增加失败计数
func (b *Backend) IncrFailCount() int32 {
	return atomic.AddInt32(&b.FailCount, 1)
}

// ResetFailCount 重置失败计数
func (b *Backend) ResetFailCount() {
	atomic.StoreInt32(&b.FailCount, 0)
}

// GetFailCount 获取失败计数
func (b *Backend) GetFailCount() int32 {
	return atomic.LoadInt32(&b.FailCount)
}
