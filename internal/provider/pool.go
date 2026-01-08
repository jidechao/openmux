package provider

import (
	"fmt"
	"sync"

	"github.com/openmux/openmux/internal/config"
)

// Pool Provider 池
type Pool struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

// NewPool 创建 Provider 池
func NewPool() *Pool {
	return &Pool{
		providers: make(map[string]Provider),
	}
}

// Register 注册 Provider
func (p *Pool) Register(name string, provider Provider) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.providers[name] = provider
}

// Get 获取 Provider
func (p *Pool) Get(name string) (Provider, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	provider, ok := p.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", name)
	}
	return provider, nil
}

// List 列出所有 Provider
func (p *Pool) List() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	names := make([]string, 0, len(p.providers))
	for name := range p.providers {
		names = append(names, name)
	}
	return names
}

// InitFromConfig 从配置初始化 Provider 池
func InitFromConfig(cfg *config.Config) *Pool {
	pool := NewPool()
	
	for name, providerCfg := range cfg.Providers {
		var provider Provider
		
		switch providerCfg.Type {
		case "openai", "":
			provider = NewOpenAIProvider(name, providerCfg.BaseURL, providerCfg.Timeout)
		default:
			// 默认使用 OpenAI 兼容实现
			provider = NewOpenAIProvider(name, providerCfg.BaseURL, providerCfg.Timeout)
		}
		
		pool.Register(name, provider)
	}
	
	return pool
}
