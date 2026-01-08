# OpenMux 快速开始指南

## 1. 安装依赖

确保已安装 Go 1.21 或更高版本：

```bash
go version
```

## 2. 克隆并构建

```bash
# 下载依赖
make deps

# 构建项目
make build
```

## 3. 配置

复制示例配置文件：

```bash
cp config.yaml.example config.yaml
```

编辑 `config.yaml`，配置你的 Provider API Keys。最简单的配置示例：

```yaml
server:
  port: 8080
  host: "0.0.0.0"
  timeout: 120s
  read_timeout: 60s
  write_timeout: 60s

auth:
  enabled: false  # 开发环境可以禁用认证

providers:
  zhipu:
    base_url: "https://open.bigmodel.cn/api/paas/v4"
    type: "openai"
    timeout: 60s
    api_keys:
      - key: "your-zhipu-api-key"  # 替换为你的 API Key
        name: "zhipu-key-1"
        rate_limit:
          rpm: 100
          tpm: 50000
          concurrent: 10
        weight: 1
        enabled: true

model_routes:
  chat:
    description: "通用聊天模型"
    targets:
      - provider: zhipu
        model: glm-4-flash
        weight: 1

passthrough:
  enabled: true
  allowed_providers:
    - zhipu

load_balancer:
  strategy: weighted_round_robin

monitoring:
  enabled: false

cache:
  enabled: false
```

## 4. 运行

```bash
# 方式 1: 使用 make
make run

# 方式 2: 直接运行
./bin/openmux -config config.yaml

# 方式 3: 使用 go run
go run ./cmd/server -config config.yaml
```

## 5. 测试

### 健康检查

```bash
curl http://localhost:8080/health
```

### 列出模型

```bash
curl http://localhost:8080/v1/models
```

### 聊天补全（非流式）

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "chat",
    "messages": [
      {"role": "user", "content": "你好，介绍一下你自己"}
    ]
  }'
```

### 聊天补全（流式）

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "chat",
    "messages": [
      {"role": "user", "content": "写一首关于春天的诗"}
    ],
    "stream": true
  }'
```

### 直通模式（直接指定 Provider）

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "zhipu/glm-4-flash",
    "messages": [
      {"role": "user", "content": "你好"}
    ]
  }'
```

## 6. 使用 Docker

### 构建镜像

```bash
make docker-build
```

### 运行容器

```bash
# 创建 .env 文件
cp .env.example .env
# 编辑 .env 文件，填入你的 API Keys

# 启动服务
make docker-run

# 查看日志
docker-compose logs -f

# 停止服务
make docker-stop
```

## 7. 启用认证

编辑 `config.yaml`：

```yaml
auth:
  enabled: true
  api_keys:
    - key: "sk-your-custom-key-123"
      name: "client-1"
      rate_limit:
        rpm: 500
        tpm: 200000
        concurrent: 50
```

使用 API Key 请求：

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-custom-key-123" \
  -d '{
    "model": "chat",
    "messages": [
      {"role": "user", "content": "你好"}
    ]
  }'
```

## 8. 多 Provider 负载均衡

配置多个 Provider 和 API Keys：

```yaml
providers:
  zhipu:
    base_url: "https://open.bigmodel.cn/api/paas/v4"
    type: "openai"
    timeout: 60s
    api_keys:
      - key: "${ZHIPU_KEY_1}"
        name: "zhipu-key-1"
        weight: 2
        enabled: true
      - key: "${ZHIPU_KEY_2}"
        name: "zhipu-key-2"
        weight: 1
        enabled: true

  aliyun:
    base_url: "https://dashscope.aliyuncs.com/compatible-mode/v1"
    type: "openai"
    timeout: 60s
    api_keys:
      - key: "${ALIYUN_KEY_1}"
        name: "aliyun-key-1"
        weight: 1
        enabled: true

model_routes:
  chat:
    description: "多云部署的聊天模型"
    targets:
      - provider: zhipu
        model: glm-4-flash
        weight: 2
      - provider: aliyun
        model: qwen-turbo
        weight: 1
```

这样配置后，请求会按照权重分配到不同的 Provider，实现负载均衡和高可用。

## 9. 常见问题

### 端口被占用

修改 `config.yaml` 中的端口：

```yaml
server:
  port: 9090  # 改为其他端口
```

### Provider 连接失败

检查：
1. API Key 是否正确
2. 网络是否可以访问 Provider 的 base_url
3. Provider 是否有限流

### 查看详细日志

修改日志级别：

```yaml
monitoring:
  enabled: true
  log_level: "debug"  # debug, info, warn, error
```

## 10. 下一步

- 阅读 [README.md](README.md) 了解完整功能
- 查看 [config.yaml.example](config.yaml.example) 了解所有配置选项
- 运行测试：`make test`
- 查看代码：浏览 `internal/` 和 `pkg/` 目录
