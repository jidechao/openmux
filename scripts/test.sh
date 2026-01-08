#!/bin/bash

# 测试脚本

BASE_URL="http://localhost:8080"

echo "=== 测试健康检查 ==="
curl -s $BASE_URL/health | jq .

echo -e "\n=== 测试模型列表 ==="
curl -s $BASE_URL/v1/models | jq .

echo -e "\n=== 测试聊天补全（非流式）==="
curl -s $BASE_URL/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "test-model",
    "messages": [
      {"role": "user", "content": "你好"}
    ]
  }' | jq .

echo -e "\n=== 测试聊天补全（流式）==="
curl -s $BASE_URL/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "test-model",
    "messages": [
      {"role": "user", "content": "你好"}
    ],
    "stream": true
  }'

echo -e "\n\n=== 测试直通模式 ==="
curl -s $BASE_URL/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "zhipu/glm-4-flash",
    "messages": [
      {"role": "user", "content": "你好"}
    ]
  }' | jq .
