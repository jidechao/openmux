package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// Load 加载配置文件
func Load(path string) (*Config, error) {
	// 加载 .env 文件（如果存在）
	// 忽略错误，因为 .env 文件是可选的
	_ = godotenv.Load()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 展开环境变量
	content := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(content), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

// validate 验证配置
func validate(cfg *Config) error {
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", cfg.Server.Port)
	}

	if len(cfg.Providers) == 0 {
		return fmt.Errorf("no providers configured")
	}

	// 验证 Provider
	for name, provider := range cfg.Providers {
		if provider.BaseURL == "" {
			return fmt.Errorf("provider %s: base_url is required", name)
		}
		if len(provider.APIKeys) == 0 {
			return fmt.Errorf("provider %s: no api_keys configured", name)
		}
		for i, key := range provider.APIKeys {
			if key == "" {
				return fmt.Errorf("provider %s: api_key[%d] is empty", name, i)
			}
		}
	}

	// 验证模型路由
	for modelName, route := range cfg.ModelRoutes {
		if len(route.Targets) == 0 {
			return fmt.Errorf("model route %s: no targets configured", modelName)
		}
		for i, target := range route.Targets {
			if _, exists := cfg.Providers[target.Provider]; !exists {
				return fmt.Errorf("model route %s: target[%d] references unknown provider %s", 
					modelName, i, target.Provider)
			}
			if target.Weight < 0 {
				return fmt.Errorf("model route %s: target[%d] has invalid weight", modelName, i)
			}
		}
	}

	// 验证直通模式
	if cfg.Passthrough.Enabled && len(cfg.Passthrough.AllowedProviders) > 0 {
		for _, provider := range cfg.Passthrough.AllowedProviders {
			if _, exists := cfg.Providers[provider]; !exists {
				return fmt.Errorf("passthrough: allowed_provider %s not found", provider)
			}
		}
	}

	return nil
}

// ParseProviderModel 解析 provider/model 格式
func ParseProviderModel(modelName string) (provider, model string, ok bool) {
	parts := strings.SplitN(modelName, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1], true
	}
	return "", "", false
}
