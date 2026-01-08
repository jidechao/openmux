package balancer

import (
	"sync/atomic"

	"github.com/openmux/openmux/internal/config"
)

// Backend 后端实例
type Backend struct {
	Provider    string
	APIKey      *config.APIKeyConfig
	Weight      int
	Healthy     bool
	ActiveConns int32
	FailCount   int32
}

// Balancer 负载均衡器接口
type Balancer interface {
	// Select 选择一个后端
	Select() (*Backend, error)
	
	// Release 释放后端连接
	Release(backend *Backend)
	
	// MarkUnhealthy 标记后端不健康
	MarkUnhealthy(backend *Backend)
	
	// MarkHealthy 标记后端健康
	MarkHealthy(backend *Backend)
	
	// GetBackends 获取所有后端
	GetBackends() []*Backend
}

// AcquireConn 获取连接
func (b *Backend) AcquireConn() bool {
	if !b.Healthy {
		return false
	}
	if b.APIKey.RateLimit.Concurrent > 0 {
		current := atomic.LoadInt32(&b.ActiveConns)
		if current >= int32(b.APIKey.RateLimit.Concurrent) {
			return false
		}
	}
	atomic.AddInt32(&b.ActiveConns, 1)
	return true
}

// ReleaseConn 释放连接
func (b *Backend) ReleaseConn() {
	atomic.AddInt32(&b.ActiveConns, -1)
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
