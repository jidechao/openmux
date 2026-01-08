package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ChatRequest 聊天请求
type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream,omitempty"`
}

// ChatMessage 聊天消息
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponse 聊天响应
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
}

// Choice 选择项
type Choice struct {
	Index   int         `json:"index"`
	Message ChatMessage `json:"message"`
}

func main() {
	baseURL := "http://localhost:8080"
	apiKey := "sk-your-service-key-1" // 如果启用了认证

	// 示例 1: 非流式请求
	fmt.Println("=== 示例 1: 非流式聊天 ===")
	nonStreamChat(baseURL, apiKey)

	fmt.Println("\n=== 示例 2: 流式聊天 ===")
	streamChat(baseURL, apiKey)

	fmt.Println("\n=== 示例 3: 直通模式 ===")
	passthroughChat(baseURL, apiKey)
}

// nonStreamChat 非流式聊天
func nonStreamChat(baseURL, apiKey string) {
	req := ChatRequest{
		Model: "chat",
		Messages: []ChatMessage{
			{Role: "user", Content: "你好，请介绍一下你自己"},
		},
		Stream: false,
	}

	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", baseURL+"/v1/chat/completions", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("错误: %s\n", body)
		return
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		fmt.Printf("解析响应失败: %v\n", err)
		return
	}

	if len(chatResp.Choices) > 0 {
		fmt.Printf("回复: %s\n", chatResp.Choices[0].Message.Content)
	}
}

// streamChat 流式聊天
func streamChat(baseURL, apiKey string) {
	req := ChatRequest{
		Model: "chat",
		Messages: []ChatMessage{
			{Role: "user", Content: "写一首关于春天的诗"},
		},
		Stream: true,
	}

	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", baseURL+"/v1/chat/completions", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("错误: %s\n", body)
		return
	}

	fmt.Print("回复: ")
	
	// 读取 SSE 流
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		if err != nil {
			if err != io.EOF {
				fmt.Printf("\n读取流失败: %v\n", err)
			}
			break
		}

		data := string(buf[:n])
		lines := strings.Split(data, "\n")
		
		for _, line := range lines {
			if strings.HasPrefix(line, "data: ") {
				content := strings.TrimPrefix(line, "data: ")
				if content == "[DONE]" {
					fmt.Println()
					return
				}
				
				// 解析 JSON 并提取内容
				var chunk map[string]interface{}
				if err := json.Unmarshal([]byte(content), &chunk); err == nil {
					if choices, ok := chunk["choices"].([]interface{}); ok && len(choices) > 0 {
						if choice, ok := choices[0].(map[string]interface{}); ok {
							if delta, ok := choice["delta"].(map[string]interface{}); ok {
								if text, ok := delta["content"].(string); ok {
									fmt.Print(text)
								}
							}
						}
					}
				}
			}
		}
	}
}

// passthroughChat 直通模式聊天
func passthroughChat(baseURL, apiKey string) {
	req := ChatRequest{
		Model: "zhipu/glm-4-flash", // 使用 provider/model 格式
		Messages: []ChatMessage{
			{Role: "user", Content: "你好"},
		},
		Stream: false,
	}

	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", baseURL+"/v1/chat/completions", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("错误: %s\n", body)
		return
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		fmt.Printf("解析响应失败: %v\n", err)
		return
	}

	if len(chatResp.Choices) > 0 {
		fmt.Printf("回复: %s\n", chatResp.Choices[0].Message.Content)
	}
}
