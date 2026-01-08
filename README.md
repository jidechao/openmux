# OpenMux

OpenMux 是一个高性能的 LLM 代理服务，支持多 Provider、负载均衡、限流和流式响应。

## 特性

- 🚀 **多 Provider 支持**：支持智谱、阿里云、DeepSeek 等多个 LLM Provider
- ⚖️ **负载均衡**：加权轮询、最少连接等多种负载均衡策略
- 🔐 **认证与限流**：API Key 认证，支持 RPM/TPM/并发限流
- 🔄 **自动重试**：失败自动切换其他 Provider/API Key
- 📊 **模型路由**：支持自定义模型别名和多云部署
- 🌊 **流式响应**：完整支持 SSE 流式输出
- 🔌 **OpenAI 兼容**：完全兼容 OpenAI API 格式

## 快速开始

### 使用 Docker Compose

1. 复制配置文件：
```bash
cp config.yaml.example config.yaml
```

2. 编辑 `config.yaml`，配置你的 Provider API Keys

3. 创建 `.env` 文件：
```bash
ZHIPU_KEY_1=your-zhipu-key-1
ZHIPU_KEY_2=your-zhipu-key-2
ALIYUN_KEY_1=your-aliyun-key
DEEPSEEK_KEY_1=your-deepseek-key
```

4. 启动服务：
```bash
docker-compose up -d
```

### 本地运行

1. 安装 Go 1.21+

2. 下载依赖：
```bash
make deps
```

3. 构建：
```bash
make build
```

4. 运行：
```bash
make run
```

## 配置说明

### Provider 配置

```yaml
providers:
  zhipu:
    base_url: "https://open.bigmodel.cn/api/paas/v4"
    type: "openai"
    timeout: 60s
    api_keys:
      - key: "${ZHIPU_KEY_1}"
        name: "zhipu-key-1"
        rate_limit:
          rpm: 100
          tpm: 50000
          concurrent: 10
        weight: 1
        enabled: true
```

### 模型路由

```yaml
model_routes:
  small-model:
    description: "通用小模型"
    targets:
      - provider: zhipu
        model: glm-4-flash
        weight: 2
      - provider: aliyun
        model: qwen-turbo
        weight: 1
```

### 直通模式

启用直通模式后，可以使用 `provider/model` 格式直接指定 Provider：

```yaml
passthrough:
  enabled: true
  allowed_providers:
    - zhipu
    - aliyun
```

## API 使用

### 聊天补全

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-service-key-1" \
  -d '{
    "model": "small-model",
    "messages": [
      {"role": "user", "content": "你好"}
    ]
  }'
```

### 流式响应

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-service-key-1" \
  -d '{
    "model": "small-model",
    "messages": [
      {"role": "user", "content": "你好"}
    ],
    "stream": true
  }'
```

### 直通模式

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-service-key-1" \
  -d '{
    "model": "zhipu/glm-4-flash",
    "messages": [
      {"role": "user", "content": "你好"}
    ]
  }'
```

### 列出模型

```bash
curl http://localhost:8080/v1/models \
  -H "Authorization: Bearer sk-your-service-key-1"
```

## 架构设计

```
Client Request
     ↓
[认证中间件] → API Key 验证
     ↓
[限流中间件] → 客户端级别限流
     ↓
[模型路由器] → 解析模型名，选择 Provider 池
     ↓
[负载均衡器] → 选择具体的 Provider + API Key
     ↓
[Provider 适配器] → 转发请求，处理流式响应
     ↓
[重试机制] → 失败时切换其他 Provider/Key
```

## 开发

### 运行测试

```bash
make test
```

### 代码格式化

```bash
make fmt
```

### 代码检查

```bash
make lint
```

## License

MIT
