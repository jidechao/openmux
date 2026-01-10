# OpenMux

OpenMux 是一个高性能的 LLM 代理网关，提供统一的 OpenAI 兼容接口，支持多 Provider 聚合、自动负载均衡、精准限流和流式响应。

## 特性

- 🚀 **多 Provider 支持**：支持智谱、阿里云、DeepSeek 等多个 LLM Provider
- ⚖️ **负载均衡**：智能加权轮询，支持在多个 Provider 或多个 API Key 之间进行负载均衡，自动剔除不健康后端。
- 📊 **高级路由**：通过 `model_routes` 实现复杂的加权、多云部署策略。
- ✨ **默认直通**：像 LiteLLM 一样，默认支持 `provider/model_name` 格式的直接请求，无需额外配置。
- 🔐 **认证与限流**：API Key 认证，支持 RPM/TPM/并发限流，基于 `tiktoken` 的精准 Token 计算
- 🔄 **自动重试**：失败自动切换其他 Provider/API Key，提升可用性
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

OpenMux 提供两种核心的请求模式：“默认直通”和“高级路由”。

### 模式一：默认直通 (Default Passthrough)

这是最简单、开箱即用的模式。您无需配置任何路由，可以直接像使用 LiteLLM 一样，通过 `provider_name/model_name` 或 `provider_name:model_name` 的格式来请求模型。

例如，要请求 `zhipu` 提供商的 `glm-4-flash` 模型，只需在 API 请求中指定：
`"model": "zhipu/glm-4-flash"` 或 `"model": "zhipu:glm-4-flash"`

#### Provider 内部的 API Key 轮询

如果一个 `provider` 配置了多个 `api_keys`，在直通模式下，系统会自动在这些 Key 之间进行**轮询（Round Robin）**。您也可以为指定的 `provider` 单独配置更复杂的均衡策略。

```yaml
providers:
  zhipu:
    base_url: "https://open.bigmodel.cn/api/paas/v4"
    type: "openai"
    api_keys:
      - "key-1"
      - "key-2"
      - "key-3"
    # load_balancer: # 可选，为 zhipu 单独配置密钥均衡策略
    #   strategy: "weighted_round_robin" # 默认为轮询
```

### 模式二：高级路由 (Advanced Routing)

当您需要更复杂的路由策略时（例如为模型起一个更简单的别名、多云部署、加权分发等），可以使用 `model_routes` 来定义。

`model_routes` 中定义的路由**优先级高于**直通模式。

```yaml
# 高级模型路由配置
model_routes:
  # 简单别名：将 my-fast-model 映射到 zhipu 的 glm-4-flash
  my-fast-model:
    targets:
      - provider: zhipu
        model: glm-4-flash

  # 多云负载均衡：将 a-great-model 同时映射到 deepseek 和 zhipu
  # 并按 3:1 的权重分发流量
  a-great-model:
    description: "一个强大的多云模型"
    targets:
      - provider: deepseek
        model: deepseek-chat
        weight: 3
      - provider: zhipu
        model: glm-4-plus
        weight: 1
    strategy: weighted_round_robin
```

请求时，您可以直接使用您定义的别名：
`"model": "my-fast-model"` 或 `"model": "a-great-model"`

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

### Rerank (重排)

OpenMux 支持通过 `/v1/rerank` 接口调用重排（Rerank）模型。
您可以在 `model_routes` 中定义您的 Rerank 模型别名，或者直接通过 `provider/rerank_model_name` 的格式进行直通访问。

```bash
curl http://localhost:8080/v1/rerank \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <您的OpenMux客户端API Key>" \
  -d '{
    "model": "rerank-jina", # 或者 "jina/jina-reranker-v1-base-en"
    "query": "搜索相关文档",
    "documents": [
      "这是一篇关于人工智能的文档。",
      "这是一篇关于机器学习的文档。",
      "这是一篇关于深度学习的文档。"
    ],
    "top_n": 2
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
[模型路由] → 优先匹配高级路由(model_routes)，失败则尝试直通
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