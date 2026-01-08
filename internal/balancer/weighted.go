package balancer

import (
	"fmt"
	"sync"

	"github.com/openmux/openmux/internal/config"
	"github.com/openmux/openmux/pkg/errors"
)

// WeightedRoundRobin 加权轮询负载均衡器
type WeightedRoundRobin struct {
	mu            sync.Mutex
	backends      []*Backend
	current       int
	gcd           int
	maxWeight     int
	currentWeight int
}

// NewWeightedRoundRobin 创建加权轮询负载均衡器
func NewWeightedRoundRobin(provider string, apiKeys []config.APIKeyConfig) *WeightedRoundRobin {
	backends := make([]*Backend, 0, len(apiKeys))
	weights := make([]int, 0, len(apiKeys))
	
	for i := range apiKeys {
		if !apiKeys[i].Enabled {
			continue
		}
		weight := apiKeys[i].Weight
		if weight <= 0 {
			weight = 1
		}
		backends = append(backends, &Backend{
			Provider: provider,
			APIKey:   &apiKeys[i],
			Weight:   weight,
			Healthy:  true,
		})
		weights = append(weights, weight)
	}
	
	gcd := gcdSlice(weights)
	maxWeight := maxSlice(weights)
	
	return &WeightedRoundRobin{
		backends:      backends,
		current:       -1,
		gcd:           gcd,
		maxWeight:     maxWeight,
		currentWeight: 0,
	}
}

// Select 选择一个后端
func (w *WeightedRoundRobin) Select() (*Backend, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	if len(w.backends) == 0 {
		return nil, errors.New(errors.ErrCodeNoAvailableBackend, "no backends available")
	}
	
	// 尝试选择后端
	for i := 0; i < len(w.backends)*w.maxWeight; i++ {
		w.current = (w.current + 1) % len(w.backends)
		
		if w.current == 0 {
			w.currentWeight -= w.gcd
			if w.currentWeight <= 0 {
				w.currentWeight = w.maxWeight
			}
		}
		
		backend := w.backends[w.current]
		if backend.Weight >= w.currentWeight && backend.AcquireConn() {
			return backend, nil
		}
	}
	
	return nil, errors.New(errors.ErrCodeNoAvailableBackend, "all backends are busy or unhealthy")
}

// Release 释放后端连接
func (w *WeightedRoundRobin) Release(backend *Backend) {
	backend.ReleaseConn()
}

// MarkUnhealthy 标记后端不健康
func (w *WeightedRoundRobin) MarkUnhealthy(backend *Backend) {
	w.mu.Lock()
	defer w.mu.Unlock()
	backend.Healthy = false
}

// MarkHealthy 标记后端健康
func (w *WeightedRoundRobin) MarkHealthy(backend *Backend) {
	w.mu.Lock()
	defer w.mu.Unlock()
	backend.Healthy = true
	backend.ResetFailCount()
}

// GetBackends 获取所有后端
func (w *WeightedRoundRobin) GetBackends() []*Backend {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.backends
}

// gcd 计算最大公约数
func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// gcdSlice 计算切片的最大公约数
func gcdSlice(nums []int) int {
	if len(nums) == 0 {
		return 1
	}
	result := nums[0]
	for i := 1; i < len(nums); i++ {
		result = gcd(result, nums[i])
	}
	return result
}

// maxSlice 获取切片最大值
func maxSlice(nums []int) int {
	if len(nums) == 0 {
		return 0
	}
	max := nums[0]
	for _, n := range nums[1:] {
		if n > max {
			max = n
		}
	}
	return max
}

// BalancerPool 负载均衡器池
type BalancerPool struct {
	mu        sync.RWMutex
	balancers map[string]Balancer
}

// NewBalancerPool 创建负载均衡器池
func NewBalancerPool() *BalancerPool {
	return &BalancerPool{
		balancers: make(map[string]Balancer),
	}
}

// Register 注册负载均衡器
func (p *BalancerPool) Register(provider string, balancer Balancer) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.balancers[provider] = balancer
}

// Get 获取负载均衡器
func (p *BalancerPool) Get(provider string) (Balancer, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	balancer, ok := p.balancers[provider]
	if !ok {
		return nil, fmt.Errorf("balancer not found for provider: %s", provider)
	}
	return balancer, nil
}

// InitFromConfig 从配置初始化负载均衡器池
func InitFromConfig(cfg *config.Config) *BalancerPool {
	pool := NewBalancerPool()
	
	for name, providerCfg := range cfg.Providers {
		var balancer Balancer
		
		switch cfg.LoadBalancer.Strategy {
		case "weighted_round_robin", "":
			balancer = NewWeightedRoundRobin(name, providerCfg.APIKeys)
		default:
			balancer = NewWeightedRoundRobin(name, providerCfg.APIKeys)
		}
		
		pool.Register(name, balancer)
	}
	
	return pool
}
