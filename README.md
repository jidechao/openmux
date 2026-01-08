# OpenMux

OpenMux 是一个高性能的 LLM 代理网关，提供统一的 OpenAI 兼容接口，支持多 Provider 聚合、自动负载均衡、精准限流和流式响应。

## 特性

- 🚀 **多 Provider 支持**：支持智谱、阿里云、DeepSeek 等多个 LLM Provider
- ⚖️ **负载均衡**：智能加权轮询，支持多 API Key 轮询，自动剔除不健康后端
- 🔐 **认证与限流**：API Key 认证，支持 RPM/TPM/并发限流，基于 `tiktoken` 的精准 Token 计算
- 🔄 **自动重试**：失败自动切换其他 Provider/API Key，提升可用性
- 📊 **模型路由**：支持自定义模型别名（Aliases）和多云部署
- 🌊 **流式响应**：完整支持 SSE 流式输出，集成官方 SDK 提升稳定性
- 🔌 **OpenAI 兼容**：支持 `/v1/chat/completions`, `/v1/embeddings`, `/v1/models` 等接口

## 快速开始

### 使用 Docker

您可以直接从 GitHub Container Registry 获取多架构镜像：

```bash
docker pull ghcr.io/evilkylin/openmux:latest
```

或者使用 Docker Compose 启动：

1. 复制配置文件：
```bash
cp config.yaml.example config.yaml
```

2. 编辑 `config.yaml`，配置您的 API Keys。

3. 启动：
```bash
docker-compose up -d
```

### 本地运行

1. 安装 Go 1.21+
2. 构建并运行：
```bash
make build
./bin/openmux -config config.yaml
```

## 配置说明

### Provider 配置

API Key 现在支持简单的数组格式，系统会自动在多个 Key 之间进行负载均衡：

```yaml
providers:
  zhipu:
    base_url: "https://open.bigmodel.cn/api/paas/v4"
    type: "openai"
    rate_limit:
      rpm: 100
      tpm: 50000
      concurrent: 10
    api_keys:
      - "sk-your-key-1"
      - "sk-your-key-2"
```

### 模型别名 (Aliases)

您可以使用简单的别名映射多个实际模型：

```yaml
alias:
  small:
    - zhipu/glm-4-flash
    - aliyun/qwen-turbo
```

## API 使用

### 聊天补全

```bash
curl http://localhost:8080/v1/chat/completions \
  -d '{
    "model": "small",
    "messages": [{"role": "user", "content": "你好"}]
  }'
```

### Embedding

```bash
curl http://localhost:8080/v1/embeddings \
  -d '{
    "model": "text-embedding-v1",
    "input": "测试文本"
  }'
```

## 架构设计

```
Client Request
     ↓
[认证中间件] → API Key 验证
     ↓
[限流检查] → 精准估算 Token (tiktoken) & RPM 检查
     ↓
[模型路由] → 解析别名或直通模式
     ↓
[负载均衡] → 选择最优 Backend (Provider + API Key)
     ↓
[官方 SDK] → 转发请求，处理流式响应
     ↓
[事后修正] → 根据实际 Usage 更新限流计数
```

## 开发

运行测试：`make test` | 代码检查：`make lint`

## License

MIT