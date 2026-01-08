package router

import (
	"fmt"
	"strings"

	"github.com/openmux/openmux/internal/config"
	"github.com/openmux/openmux/pkg/errors"
)

// Router 模型路由器
type Router struct {
	routes            map[string]config.ModelRoute
	passthrough       bool
	allowedProviders  map[string]bool
	providers         map[string]bool
}

// NewRouter 创建路由器
func NewRouter(cfg *config.Config) *Router {
	allowedProviders := make(map[string]bool)
	if cfg.Passthrough.Enabled && len(cfg.Passthrough.AllowedProviders) > 0 {
		for _, p := range cfg.Passthrough.AllowedProviders {
			allowedProviders[p] = true
		}
	}
	
	providers := make(map[string]bool)
	for name := range cfg.Providers {
		providers[name] = true
	}
	
	return &Router{
		routes:           cfg.ModelRoutes,
		passthrough:      cfg.Passthrough.Enabled,
		allowedProviders: allowedProviders,
		providers:        providers,
	}
}

// Route 路由模型请求
func (r *Router) Route(modelName string) ([]config.Target, error) {
	// 1. 优先匹配自定义别名
	if route, ok := r.routes[modelName]; ok {
		return route.Targets, nil
	}
	
	// 2. 尝试解析 provider/model 格式
	if r.passthrough {
		provider, model, ok := parseProviderModel(modelName)
		if ok {
			// 检查 provider 是否存在
			if !r.providers[provider] {
				return nil, errors.New(errors.ErrCodeModelNotFound, 
					fmt.Sprintf("provider not found: %s", provider))
			}
			
			// 检查是否在允许列表中
			if len(r.allowedProviders) > 0 && !r.allowedProviders[provider] {
				return nil, errors.New(errors.ErrCodeModelNotFound, 
					fmt.Sprintf("provider not allowed: %s", provider))
			}
			
			return []config.Target{{
				Provider: provider,
				Model:    model,
				Weight:   1,
			}}, nil
		}
	}
	
	return nil, errors.New(errors.ErrCodeModelNotFound, 
		fmt.Sprintf("model not found: %s", modelName))
}

// ListModels 列出所有可用模型
func (r *Router) ListModels() []string {
	models := make([]string, 0, len(r.routes))
	for name := range r.routes {
		models = append(models, name)
	}
	return models
}

// parseProviderModel 解析 provider/model 格式
func parseProviderModel(modelName string) (provider, model string, ok bool) {
	parts := strings.SplitN(modelName, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1], true
	}
	return "", "", false
}
