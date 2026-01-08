package provider

import (
	"context"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/ssestream"
	pkgopenai "github.com/openmux/openmux/pkg/openai"
)

// Provider 定义 Provider 接口
type Provider interface {
	// ChatCompletion 聊天补全（非流式）
	ChatCompletion(ctx context.Context, req *pkgopenai.ChatCompletionRequest, model, apiKey string) (*openai.ChatCompletion, error)
	
	// ChatCompletionStream 聊天补全（流式）
	ChatCompletionStream(ctx context.Context, req *pkgopenai.ChatCompletionRequest, model, apiKey string) (*StreamResponse, error)

	// CreateEmbedding 创建 Embedding
	CreateEmbedding(ctx context.Context, req *pkgopenai.EmbeddingRequest, model, apiKey string) (*openai.CreateEmbeddingResponse, error)
	
	// Name 返回 Provider 名称
	Name() string
}

// StreamResponse 流式响应
type StreamResponse struct {
	Stream *ssestream.Stream[openai.ChatCompletionChunk]
}
