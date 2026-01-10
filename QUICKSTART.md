# OpenMux 快速开始指南

## 1. 安装

确保已安装 Go 1.21 或更高版本。

```bash
# 下载依赖
go mod tidy
# 构建
make build
```

## 2. 配置

复制示例配置文件并编辑：

```bash
cp config.yaml.example config.yaml
```

**核心配置示例：**

```yaml
auth:
  enabled: false # 开发环境可禁用

providers:
  zhipu:
    base_url: "https://open.bigmodel.cn/api/paas/v4"
    type: "openai"
    rate_limit:
      rpm: 100
      tpm: 50000
    api_keys:
      - "your-api-key-1"
      - "your-api-key-2"

aliases:
  chat:
    - zhipu/glm-4-flash
```

## 3. 运行

```bash
./bin/openmux -config config.yaml
```

## 4. 测试 API

### 聊天补全 (Chat)

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{ "model": "chat", "messages": [{"role": "user", "content": "你好"}] }'
```

### 嵌入 (Embedding)

```bash
curl http://localhost:8080/v1/embeddings \
  -H "Content-Type: application/json" \
  -d '{ `"model": "zhipu/embedding-3"` (或者 `"zhipu:embedding-3"`), "input": "hello" }'
```

## 5. 使用 Docker (推荐)

直接运行最新镜像：

```bash
docker run -p 8080:8080 -v $(pwd)/config.yaml:/app/config.yaml ghcr.io/evilkylin/openmux:latest
```

## 6. 开启精准限流

OpenMux 内置了 `tiktoken` 支持。只需在 `providers` 下配置 `tpm`，系统就会根据不同的分词器精准计算每个请求的消耗。

---
更多详细配置请参考 [README.md](README.md)。