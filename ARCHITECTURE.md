# OpenMux 架构设计

## 概述

OpenMux 是一个高性能的 LLM 代理服务，提供统一的 OpenAI 兼容 API，支持多 Provider、负载均衡、限流和自动重试。

## 核心架构

```
┌─────────────────────────────────────────────────────────────┐
│                        Client Request                        │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                     认证中间件 (Auth)                         │
│  - API Key 验证                                              │
│  - 客户端识别                                                 │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                    限流中间件 (RateLimit)                     │
│  - RPM (每分钟请求数)                                         │
│  - TPM (每分钟 Token 数)                                      │
│  - 并发限制                                                   │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                     模型路由器 (Router)                       │
│  - 解析模型名称                                               │
│  - 匹配自定义别名                                             │
│  - 支持直通模式 (provider/model)                              │
│  - 返回目标 Provider 列表                                     │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                   负载均衡器 (Balancer)                       │
│  - 加权轮询 (Weighted Round Robin)                           │
│  - 健康检查                                                   │
│  - 并发控制                                                   │
│  - 选择具体的 Provider + API Key                             │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                  Provider 适配器 (Provider)                   │
│  - OpenAI 兼容实现                                            │
│  - HTTP 请求转发                                              │
│  - 流式响应处理 (SSE)                                         │
│  - 错误处理和转换                                             │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                      重试机制 (Retry)                         │
│  - 失败自动切换 Provider                                      │
│  - 切换不同的 API Key                                         │
│  - 可配置重试策略                                             │
└─────────────────────────────────────────────────────────────┘
```

## 模块详解

### 1. 配置管理 (internal/config)

**职责**：
- 加载和解析 YAML 配置文件
- 环境变量展开
- 配置验证

**核心结构**：
```go
type Config struct {
    Server       ServerConfig
    Auth         AuthConfig
    Providers    map[string]ProviderConfig
    ModelRoutes  map[string]ModelRoute
    Passthrough  PassthroughConfig
    LoadBalancer LoadBalancerConfig
    Monitoring   MonitoringConfig
    Cache        CacheConfig
}
```

### 2. 认证管理 (internal/auth)

**职责**：
- API Key 验证
- 客户端信息管理
- 认证中间件

**核心组件**：
- `Manager`: 管理客户端 API Keys
- `Middleware`: HTTP 认证中间件
- `ClientInfo`: 客户端信息和限流器

**流程**：
1. 从 Authorization Header 提取 API Key
2. 验证 API Key 是否有效
3. 将客户端信息存入 Context
4. 检查客户端级别限流

### 3. 限流器 (internal/ratelimit)

**职责**：
- 请求频率限制
- Token 使用量限制
- 并发连接限制

**实现**：
- `TokenBucket`: Token Bucket 算法实现
- `MultiLimiter`: 多维度限流（RPM + TPM）

**特性**：
- 每分钟请求数 (RPM)
- 每分钟 Token 数 (TPM)
- 最大并发连接数

### 4. 模型路由 (internal/router)

**职责**：
- 模型名称解析
- 路由规则匹配
- 目标 Provider 选择

**路由模式**：

1. **自定义别名**：
   ```yaml
   model_routes:
     chat:
       targets:
         - provider: zhipu
           model: glm-4-flash
   ```
   请求 `chat` → 路由到 `zhipu/glm-4-flash`

2. **直通模式**：
   ```yaml
   passthrough:
     enabled: true
   ```
   请求 `zhipu/glm-4-flash` → 直接路由到指定 Provider

3. **多目标负载均衡**：
   ```yaml
   model_routes:
     chat:
       targets:
         - provider: zhipu
           model: glm-4-flash
           weight: 2
         - provider: aliyun
           model: qwen-turbo
           weight: 1
   ```
   按权重分配请求

### 5. 负载均衡 (internal/balancer)

**职责**：
- 选择具体的后端实例
- 健康检查
- 连接管理

**实现**：
- `WeightedRoundRobin`: 加权轮询算法
- `Backend`: 后端实例（Provider + API Key）
- `BalancerPool`: 负载均衡器池

**算法**：
```
1. 按权重轮询选择后端
2. 检查后端健康状态
3. 检查并发连接限制
4. 获取连接，增加计数
5. 请求完成后释放连接
```

**健康检查**：
- 连续失败达到阈值 → 标记为不健康
- 不健康的后端不参与负载均衡
- 连续成功达到阈值 → 恢复健康

### 6. Provider 适配器 (internal/provider)

**职责**：
- 统一的 Provider 接口
- HTTP 请求转发
- 流式响应处理

**接口定义**：
```go
type Provider interface {
    ChatCompletion(ctx, req, model, apiKey) (*Response, error)
    ChatCompletionStream(ctx, req, model, apiKey) (*StreamResponse, error)
    Name() string
}
```

**OpenAI 实现**：
- 兼容 OpenAI API 格式
- 支持非流式和流式响应
- SSE (Server-Sent Events) 流处理

### 7. HTTP 处理器 (internal/handler)

**职责**：
- HTTP 请求处理
- 响应格式化
- 错误处理

**端点**：
- `/v1/chat/completions`: 聊天补全
- `/v1/models`: 模型列表
- `/health`: 健康检查

**流式响应处理**：
```go
1. 设置 SSE Headers
2. 从 Provider 获取流式响应
3. 逐块转发给客户端
4. 处理错误和结束信号
```

### 8. 中间件 (internal/middleware)

**职责**：
- 请求日志
- 错误恢复
- CORS 处理

**中间件链**：
```
Recovery → Logger → CORS → Auth → Handler
```

## 数据流

### 非流式请求

```
1. Client → HTTP Request
2. Auth Middleware → 验证 API Key
3. RateLimit → 检查限流
4. Router → 解析模型，获取 Targets
5. Balancer → 选择 Backend (Provider + API Key)
6. Provider → 发送请求到上游
7. Provider ← 接收完整响应
8. Client ← 返回 JSON 响应
```

### 流式请求

```
1. Client → HTTP Request (stream=true)
2. Auth Middleware → 验证 API Key
3. RateLimit → 检查限流
4. Router → 解析模型，获取 Targets
5. Balancer → 选择 Backend
6. Provider → 建立 SSE 连接
7. Provider ← 接收流式数据块
8. Client ← 实时转发数据块
9. Provider ← 接收 [DONE] 信号
10. Client ← 关闭连接
```

## 重试机制

### 重试策略

```go
for each target in targets {
    backend = balancer.Select()
    response, err = provider.Request(backend)
    
    if err == nil {
        return response  // 成功
    }
    
    if isRateLimitError(err) {
        balancer.MarkUnhealthy(backend)
        continue  // 尝试下一个
    }
    
    if isRetryable(err) {
        continue  // 尝试下一个
    }
    
    return err  // 不可重试的错误
}

return "all targets failed"
```

### 错误分类

- **可重试错误**：
  - 限流错误 (429)
  - 超时错误
  - 网络错误
  - Provider 错误 (5xx)

- **不可重试错误**：
  - 认证错误 (401)
  - 参数错误 (400)
  - 模型不存在 (404)

## 性能优化

### 1. 连接池

- HTTP Client 复用
- Keep-Alive 连接
- 连接超时控制

### 2. 并发控制

- 每个 API Key 的并发限制
- 防止单个 Key 过载
- 优雅的降级

### 3. 内存管理

- 流式响应使用 Channel
- 及时释放资源
- 避免内存泄漏

### 4. 错误处理

- 快速失败
- 自动重试
- 降级策略

## 扩展性

### 添加新 Provider

1. 实现 `Provider` 接口
2. 在 `pool.go` 中注册
3. 配置文件中添加配置

### 添加新的负载均衡算法

1. 实现 `Balancer` 接口
2. 在 `InitFromConfig` 中添加分支
3. 配置文件中指定策略

### 添加监控指标

1. 定义 Prometheus 指标
2. 在关键路径埋点
3. 暴露 `/metrics` 端点

## 安全性

### 1. 认证

- API Key 验证
- 支持禁用认证（开发环境）

### 2. 限流

- 防止滥用
- 保护上游 Provider
- 多维度限流

### 3. 隔离

- 客户端级别隔离
- Provider 级别隔离
- 错误不传播

## 可靠性

### 1. 健康检查

- 自动检测不健康的后端
- 自动恢复
- 避免雪崩

### 2. 重试机制

- 自动切换 Provider
- 自动切换 API Key
- 可配置重试次数

### 3. 优雅关闭

- 等待请求完成
- 超时强制关闭
- 资源清理

## 监控和日志

### 日志

- 请求日志：方法、路径、状态码、耗时
- 错误日志：详细的错误信息和堆栈
- 访问日志：客户端、模型、Provider

### 指标（待实现）

- 请求总数
- 请求延迟
- 错误率
- Provider 健康状态
- Token 使用量

## 部署架构

### 单机部署

```
┌─────────────┐
│   OpenMux   │
│   :8080     │
└─────────────┘
      ↓
┌─────────────┐
│  Providers  │
│  (上游 API) │
└─────────────┘
```

### 高可用部署

```
┌─────────────┐
│   Nginx     │
│  (负载均衡)  │
└─────────────┘
      ↓
┌─────────────┐  ┌─────────────┐
│  OpenMux 1  │  │  OpenMux 2  │
└─────────────┘  └─────────────┘
      ↓                ↓
┌─────────────────────────────┐
│        Providers            │
└─────────────────────────────┘
```

## 未来规划

- [ ] Prometheus 监控集成
- [ ] 响应缓存
- [ ] 更多负载均衡算法（最少连接、随机）
- [ ] 更多 Provider 支持（Azure、AWS）
- [ ] 管理 API（动态配置）
- [ ] WebUI 管理界面
- [ ] 请求日志持久化
- [ ] 分布式限流（Redis）
