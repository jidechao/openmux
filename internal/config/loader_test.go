package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// 创建临时配置文件
	content := `
server:
  port: 8080
  host: "0.0.0.0"
  timeout: 120s
  read_timeout: 60s
  write_timeout: 60s

auth:
  enabled: false

providers:
  test:
    base_url: "https://api.test.com"
    type: "openai"
    timeout: 60s
    api_keys:
      - key: "test-key"
        name: "test"
        weight: 1
        enabled: true

model_routes:
  test-model:
    targets:
      - provider: test
        model: test
        weight: 1

passthrough:
  enabled: true

load_balancer:
  strategy: weighted_round_robin

monitoring:
  enabled: false

cache:
  enabled: false
`

	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", cfg.Server.Port)
	}

	if len(cfg.Providers) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(cfg.Providers))
	}
}

func TestParseProviderModel(t *testing.T) {
	tests := []struct {
		input    string
		provider string
		model    string
		ok       bool
	}{
		{"zhipu/glm-4-flash", "zhipu", "glm-4-flash", true},
		{"aliyun/qwen-turbo", "aliyun", "qwen-turbo", true},
		{"simple-model", "", "", false},
	}

	for _, tt := range tests {
		provider, model, ok := ParseProviderModel(tt.input)
		if ok != tt.ok {
			t.Errorf("ParseProviderModel(%q) ok = %v, want %v", tt.input, ok, tt.ok)
		}
		if provider != tt.provider {
			t.Errorf("ParseProviderModel(%q) provider = %q, want %q", tt.input, provider, tt.provider)
		}
		if model != tt.model {
			t.Errorf("ParseProviderModel(%q) model = %q, want %q", tt.input, model, tt.model)
		}
	}
}
