package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/openmux/openmux/pkg/errors"
	"github.com/openmux/openmux/pkg/openai"
)

// OpenAIProvider OpenAI 兼容的 Provider 实现
type OpenAIProvider struct {
	name    string
	baseURL string
	client  *http.Client
	timeout time.Duration
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
	req *openai.ChatCompletionRequest,
	model, apiKey string,
) (*openai.ChatCompletionResponse, error) {
	// 设置模型
	req.Model = model
	req.Stream = false

	body, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInvalidRequest, "failed to marshal request", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
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
		var errResp openai.ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err == nil {
			return nil, errors.New(errors.ErrCodeProviderError, errResp.Error.Message)
		}
		return nil, errors.New(errors.ErrCodeProviderError, fmt.Sprintf("http error: %d", resp.StatusCode))
	}

	var result openai.ChatCompletionResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, errors.Wrap(errors.ErrCodeProviderError, "failed to unmarshal response", err)
	}

	// 确保 object 字段符合 OpenAI 标准
	if result.Object == "" {
		result.Object = "chat.completion"
	}

	return &result, nil
}

// ChatCompletionStream 聊天补全（流式）
func (p *OpenAIProvider) ChatCompletionStream(
	ctx context.Context,
	req *openai.ChatCompletionRequest,
	model, apiKey string,
) (*StreamResponse, error) {
	// 设置模型和流式
	req.Model = model
	req.Stream = true

	body, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInvalidRequest, "failed to marshal request", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeProviderError, "failed to create request", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeProviderError, "request failed", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		var errResp openai.ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err == nil {
			return nil, errors.New(errors.ErrCodeProviderError, errResp.Error.Message)
		}
		return nil, errors.New(errors.ErrCodeProviderError, fmt.Sprintf("http error: %d", resp.StatusCode))
	}

	chunkCh := make(chan *openai.ChatCompletionChunk, 10)
	errCh := make(chan error, 1)

	go func() {
		defer close(chunkCh)
		defer close(errCh)
		defer resp.Body.Close()

		reader := openai.NewStreamReader(ctx, resp.Body)
		for {
			chunk, err := reader.Recv()
			if err != nil {
				if err != io.EOF {
					errCh <- err
				}
				return
			}

			select {
			case chunkCh <- chunk:
			case <-ctx.Done():
				return
			}
		}
	}()

	return &StreamResponse{
		ChunkCh: chunkCh,
		ErrCh:   errCh,
	}, nil
}
