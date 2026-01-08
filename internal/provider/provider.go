package provider

import (
	"context"

	"github.com/openmux/openmux/pkg/openai"
)

// Provider 定义 Provider 接口
type Provider interface {
	// ChatCompletion 聊天补全（非流式）
	ChatCompletion(ctx context.Context, req *openai.ChatCompletionRequest, model, apiKey string) (*openai.ChatCompletionResponse, error)
	
	// ChatCompletionStream 聊天补全（流式）
	ChatCompletionStream(ctx context.Context, req *openai.ChatCompletionRequest, model, apiKey string) (*StreamResponse, error)
	
	// Name 返回 Provider 名称
	Name() string
}

// StreamResponse 流式响应
type StreamResponse struct {
	ChunkCh <-chan *openai.ChatCompletionChunk
	ErrCh   <-chan error
}
