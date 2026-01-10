package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
	"github.com/openmux/openmux/pkg/errors"
	"github.com/openmux/openmux/pkg/logger"
	pkgopenai "github.com/openmux/openmux/pkg/openai"
)

// OpenAIProvider OpenAI 兼容的 Provider 实现
type OpenAIProvider struct {
	name    string
	baseURL string
	timeout time.Duration
	client  *http.Client
}

// NewOpenAIProvider 创建 OpenAI Provider
func NewOpenAIProvider(name, baseURL string, timeout time.Duration) *OpenAIProvider {
	return &OpenAIProvider{
		name:    name,
		baseURL: baseURL,
		timeout: timeout,
		client: &http.Client{
			Timeout: timeout,
		},
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
		option.WithHTTPClient(p.client), // 复用 HTTP Client
	)

	params, err := p.convertRequest(req, model)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInvalidRequest, "failed to convert request", err)
	}

	resp, err := client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, p.handleError(err)
	}

	if len(resp.Choices) > 0 {
		logger.Debugf("Response Choice 0: FinishReason=%s, ToolCalls=%d", 
			resp.Choices[0].FinishReason, len(resp.Choices[0].Message.ToolCalls))
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
		option.WithHTTPClient(p.client),
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
		option.WithHTTPClient(p.client),
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

// Rerank 重排序
func (p *OpenAIProvider) Rerank(
	ctx context.Context,
	req *pkgopenai.RerankRequest,
	model, apiKey string,
) (*pkgopenai.RerankResponse, error) {
	// 强制设置模型
	req.Model = model

	body, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInvalidRequest, "failed to marshal request", err)
	}

	// 假设下游 Rerank API 路径为 /rerank (兼容 Cohere/Jina/SiliconFlow 等)
	// BaseURL 通常是 https://api.xxx.com/v1
	// 所以完整 URL 是 https://api.xxx.com/v1/rerank
	url := p.baseURL + "/rerank"

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeProviderError, "failed to create request", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeProviderError, "request failed", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeProviderError, "failed to read response", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(errors.ErrCodeProviderError, fmt.Sprintf("http error: %d, body: %s", resp.StatusCode, string(respBody)))
	}

	var result pkgopenai.RerankResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, errors.Wrap(errors.ErrCodeProviderError, "failed to unmarshal response", err)
	}

	return &result, nil
}

// convertRequest 将 DTO 转换为 SDK 参数
func (p *OpenAIProvider) convertRequest(req *pkgopenai.ChatCompletionRequest, model string) (openai.ChatCompletionNewParams, error) {
	logger.Debugf("convertRequest incoming: Model=%s, Tools=%d, ToolChoice=%v", model, len(req.Tools), req.ToolChoice)

	// 使用 JSON 转换来实现真正的“透传”效果，避免手动映射漏掉字段
	data, err := json.Marshal(req)
	if err != nil {
		return openai.ChatCompletionNewParams{}, err
	}

	var params openai.ChatCompletionNewParams
	if err := json.Unmarshal(data, &params); err != nil {
		return openai.ChatCompletionNewParams{}, err
	}

	// 覆盖模型名称为路由指定的目标模型
	params.Model = shared.ChatModel(model)

	return params, nil
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

// handleError 处理 SDK 错误
func (p *OpenAIProvider) handleError(err error) error {
	// SDK 错误通常是 *openai.Error
	if apiErr, ok := err.(*openai.Error); ok {
		return errors.New(errors.ErrCodeProviderError, apiErr.Message)
	}
	return errors.Wrap(errors.ErrCodeProviderError, "provider error", err)
}