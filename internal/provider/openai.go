package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openmux/openmux/pkg/errors"
	pkgopenai "github.com/openmux/openmux/pkg/openai"
)

// OpenAIProvider OpenAI 兼容的 Provider 实现
type OpenAIProvider struct {
	name    string
	baseURL string
	timeout time.Duration
}

// NewOpenAIProvider 创建 OpenAI Provider
func NewOpenAIProvider(name, baseURL string, timeout time.Duration) *OpenAIProvider {
	return &OpenAIProvider{
		name:    name,
		baseURL: baseURL,
		timeout: timeout,
	}
}

// Name 返回 Provider 名称
func (p *OpenAIProvider) Name() string {
	return p.name
}

// ChatCompletion 聊天补全（非流式）
func (p *OpenAIProvider) ChatCompletion(
	ctx context.Context,
	req *pkgopenai.ChatCompletionRequest,
	model, apiKey string,
) (*openai.ChatCompletion, error) {
	client := openai.NewClient(
		option.WithBaseURL(p.baseURL),
		option.WithAPIKey(apiKey),
		option.WithRequestTimeout(p.timeout),
	)

	params, err := p.convertRequest(req, model)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInvalidRequest, "failed to convert request", err)
	}

	resp, err := client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, p.handleError(err)
	}

	return resp, nil
}

// ChatCompletionStream 聊天补全（流式）
func (p *OpenAIProvider) ChatCompletionStream(
	ctx context.Context,
	req *pkgopenai.ChatCompletionRequest,
	model, apiKey string,
) (*StreamResponse, error) {
	client := openai.NewClient(
		option.WithBaseURL(p.baseURL),
		option.WithAPIKey(apiKey),
		option.WithRequestTimeout(p.timeout),
	)

	params, err := p.convertRequest(req, model)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInvalidRequest, "failed to convert request", err)
	}

	stream := client.Chat.Completions.NewStreaming(ctx, params)
	
	return &StreamResponse{
		Stream: stream,
	}, nil
}

// CreateEmbedding 创建 Embedding
func (p *OpenAIProvider) CreateEmbedding(
	ctx context.Context,
	req *pkgopenai.EmbeddingRequest,
	model, apiKey string,
) (*openai.CreateEmbeddingResponse, error) {
	client := openai.NewClient(
		option.WithBaseURL(p.baseURL),
		option.WithAPIKey(apiKey),
		option.WithRequestTimeout(p.timeout),
	)

	params, err := p.convertEmbeddingRequest(req, model)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInvalidRequest, "failed to convert request", err)
	}

	resp, err := client.Embeddings.New(ctx, params)
	if err != nil {
		return nil, p.handleError(err)
	}

	return resp, nil
}

// convertEmbeddingRequest 将 Embedding DTO 转换为 SDK 参数
func (p *OpenAIProvider) convertEmbeddingRequest(req *pkgopenai.EmbeddingRequest, model string) (openai.EmbeddingNewParams, error) {
	params := openai.EmbeddingNewParams{
		Model: model,
	}

	// 转换 Input
	switch v := req.Input.(type) {
	case string:
		params.Input = openai.EmbeddingNewParamsInputUnion{
			OfString: openai.String(v),
		}
	case []interface{}:
		if len(v) > 0 {
			switch v[0].(type) {
			case string:
				strs := make([]string, len(v))
				for i, elem := range v {
					if s, ok := elem.(string); ok {
						strs[i] = s
					} else {
						return params, fmt.Errorf("mixed types in input array")
					}
				}
				params.Input = openai.EmbeddingNewParamsInputUnion{
					OfArrayOfStrings: strs,
				}
			default:
				// 暂时只支持字符串数组，token 数组处理比较复杂且网关场景少见
				return params, fmt.Errorf("unsupported input array element type: %T", v[0])
			}
		}
	default:
		return params, fmt.Errorf("unsupported input type: %T", v)
	}

	if req.User != "" {
		params.User = openai.String(req.User)
	}

	if req.EncodingFormat != "" {
		params.EncodingFormat = openai.EmbeddingNewParamsEncodingFormat(req.EncodingFormat)
	}

	if req.Dimensions != nil {
		params.Dimensions = openai.Int(int64(*req.Dimensions))
	}

	return params, nil
}

// convertRequest 将 DTO 转换为 SDK 参数
func (p *OpenAIProvider) convertRequest(req *pkgopenai.ChatCompletionRequest, model string) (openai.ChatCompletionNewParams, error) {
	params := openai.ChatCompletionNewParams{
		Model: model,
	}

	// 转换 Messages
	var messages []openai.ChatCompletionMessageParamUnion
	for _, msg := range req.Messages {
		switch msg.Role {
		case "system":
			messages = append(messages, openai.SystemMessage(msg.Content))
		case "user":
			messages = append(messages, openai.UserMessage(msg.Content))
		case "assistant":
			messages = append(messages, openai.AssistantMessage(msg.Content))
		// 工具调用暂时简化处理，避免编译错误
		case "tool":
			messages = append(messages, openai.ToolMessage(msg.ToolCallID, msg.Content))
		default:
			messages = append(messages, openai.UserMessage(msg.Content))
		}
	}
	params.Messages = messages

	// 可选参数
	if req.Temperature != nil {
		params.Temperature = openai.Float(float64(*req.Temperature))
	}
	if req.TopP != nil {
		params.TopP = openai.Float(float64(*req.TopP))
	}
	if req.MaxTokens != nil {
		params.MaxTokens = openai.Int(int64(*req.MaxTokens))
	}
	if req.N != nil {
		params.N = openai.Int(int64(*req.N))
	}
	if req.PresencePenalty != nil {
		params.PresencePenalty = openai.Float(float64(*req.PresencePenalty))
	}
	if req.FrequencyPenalty != nil {
		params.FrequencyPenalty = openai.Float(float64(*req.FrequencyPenalty))
	}
	if req.User != "" {
		params.User = openai.String(req.User)
	}
	
	return params, nil
}

// handleError 处理 SDK 错误
func (p *OpenAIProvider) handleError(err error) error {
	// SDK 错误通常是 *openai.Error
	if apiErr, ok := err.(*openai.Error); ok {
		return errors.New(errors.ErrCodeProviderError, apiErr.Message)
	}
	return errors.Wrap(errors.ErrCodeProviderError, "provider error", err)
}
