# OpenMux 使用示例

## Go 客户端示例

### 运行示例

```bash
# 确保 OpenMux 服务正在运行
cd examples
go run client.go
```

### 代码说明

`client.go` 包含三个示例：

1. **非流式聊天**: 发送请求并等待完整响应
2. **流式聊天**: 实时接收流式响应
3. **直通模式**: 使用 `provider/model` 格式直接指定 Provider

## Python 客户端示例

### 使用 OpenAI SDK

```python
from openai import OpenAI

# 配置客户端
client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="sk-your-service-key-1"
)

# 非流式请求
response = client.chat.completions.create(
    model="chat",
    messages=[
        {"role": "user", "content": "你好"}
    ]
)
print(response.choices[0].message.content)

# 流式请求
stream = client.chat.completions.create(
    model="chat",
    messages=[
        {"role": "user", "content": "写一首诗"}
    ],
    stream=True
)

for chunk in stream:
    if chunk.choices[0].delta.content:
        print(chunk.choices[0].delta.content, end="")
```

## JavaScript 客户端示例

### 使用 fetch API

```javascript
// 非流式请求
async function chat() {
  const response = await fetch('http://localhost:8080/v1/chat/completions', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': 'Bearer sk-your-service-key-1'
    },
    body: JSON.stringify({
      model: 'chat',
      messages: [
        { role: 'user', content: '你好' }
      ]
    })
  });
  
  const data = await response.json();
  console.log(data.choices[0].message.content);
}

// 流式请求
async function streamChat() {
  const response = await fetch('http://localhost:8080/v1/chat/completions', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': 'Bearer sk-your-service-key-1'
    },
    body: JSON.stringify({
      model: 'chat',
      messages: [
        { role: 'user', content: '写一首诗' }
      ],
      stream: true
    })
  });
  
  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    
    const chunk = decoder.decode(value);
    const lines = chunk.split('\n');
    
    for (const line of lines) {
      if (line.startsWith('data: ')) {
        const data = line.slice(6);
        if (data === '[DONE]') return;
        
        try {
          const json = JSON.parse(data);
          const content = json.choices[0]?.delta?.content;
          if (content) {
            process.stdout.write(content);
          }
        } catch (e) {
          // 忽略解析错误
        }
      }
    }
  }
}
```

## cURL 示例

### 非流式请求

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-service-key-1" \
  -d '{
    "model": "chat",
    "messages": [
      {"role": "user", "content": "你好"}
    ]
  }'
```

### 流式请求

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-service-key-1" \
  -d '{
    "model": "chat",
    "messages": [
      {"role": "user", "content": "写一首诗"}
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

### 健康检查

```bash
curl http://localhost:8080/health
```

## 注意事项

1. 如果配置中禁用了认证 (`auth.enabled: false`)，可以省略 `Authorization` header
2. 确保使用的模型名称在配置文件中已定义
3. 流式响应使用 SSE (Server-Sent Events) 格式
4. 所有端点都兼容 OpenAI API 格式
