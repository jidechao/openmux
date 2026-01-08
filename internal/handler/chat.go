package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/openmux/openmux/internal/balancer"
	"github.com/openmux/openmux/internal/config"
	"github.com/openmux/openmux/internal/provider"
	"github.com/openmux/openmux/internal/router"
	"github.com/openmux/openmux/pkg/errors"
	"github.com/openmux/openmux/pkg/openai"
)

// ChatHandler 聊天补全处理器
type ChatHandler struct {
	router         *router.Router
	providerPool   *provider.Pool
	balancerPool   *balancer.BalancerPool
}

// NewChatHandler 创建聊天处理器
func NewChatHandler(
	router *router.Router,
	providerPool *provider.Pool,
	balancerPool *balancer.BalancerPool,
) *ChatHandler {
	return &ChatHandler{
		router:       router,
		providerPool: providerPool,
		balancerPool: balancerPool,
	}
}

// Handle 处理聊天补全请求
func (h *ChatHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

	// 解析请求
	var req openai.ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	// 路由模型
	targets, err := h.router.Route(req.Model)
	if err != nil {
		if e, ok := err.(*errors.Error); ok {
			writeError(w, http.StatusNotFound, string(e.Code), e.Message)
		} else {
			writeError(w, http.StatusNotFound, "model_not_found", err.Error())
		}
		return
	}

	// 处理请求
	if req.Stream {
		h.handleStream(w, r, &req, targets)
	} else {
		h.handleNonStream(w, r, &req, targets)
	}
}

// handleNonStream 处理非流式请求
func (h *ChatHandler) handleNonStream(
	w http.ResponseWriter,
	r *http.Request,
	req *openai.ChatCompletionRequest,
	targets []config.Target,
) {
	resp, err := h.handleWithRetry(r.Context(), req, targets)
	if err != nil {
		if e, ok := err.(*errors.Error); ok {
			writeError(w, http.StatusInternalServerError, string(e.Code), e.Message)
		} else {
			writeError(w, http.StatusInternalServerError, "provider_error", err.Error())
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleStream 处理流式请求
func (h *ChatHandler) handleStream(
	w http.ResponseWriter,
	r *http.Request,
	req *openai.ChatCompletionRequest,
	targets []config.Target,
) {
	// 设置 SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming_not_supported", "Streaming not supported")
		return
	}

	// 尝试每个 target
	var lastErr error
	for _, target := range targets {
		backend, err := h.selectBackend(target.Provider)
		if err != nil {
			lastErr = err
			continue
		}
		defer func(providerName string, b *balancer.Backend) {
			if bal, _ := h.balancerPool.Get(providerName); bal != nil {
				bal.Release(b)
			}
		}(target.Provider, backend)

		prov, err := h.providerPool.Get(target.Provider)
		if err != nil {
			lastErr = err
			continue
		}

		streamResp, err := prov.ChatCompletionStream(r.Context(), req, target.Model, backend.APIKey.Key)
		if err != nil {
			h.markBackendUnhealthy(target.Provider, backend)
			lastErr = err
			continue
		}

		// 转发流式响应
		h.forwardStream(w, flusher, streamResp)
		return
	}

	// 所有 target 都失败
	writeSSEError(w, flusher, "provider_error", fmt.Sprintf("All targets failed: %v", lastErr))
}

// forwardStream 转发流式响应
func (h *ChatHandler) forwardStream(w http.ResponseWriter, flusher http.Flusher, streamResp *provider.StreamResponse) {
	for {
		select {
		case chunk, ok := <-streamResp.ChunkCh:
			if !ok {
				fmt.Fprintf(w, "data: [DONE]\n\n")
				flusher.Flush()
				return
			}

			data, err := json.Marshal(chunk)
			if err != nil {
				continue
			}

			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()

		case err := <-streamResp.ErrCh:
			if err != nil && err != io.EOF {
				writeSSEError(w, flusher, "stream_error", err.Error())
			}
			return
		}
	}
}

// handleWithRetry 带重试的请求处理
func (h *ChatHandler) handleWithRetry(
	ctx context.Context,
	req *openai.ChatCompletionRequest,
	targets []config.Target,
) (*openai.ChatCompletionResponse, error) {
	var lastErr error

	for _, target := range targets {
		backend, err := h.selectBackend(target.Provider)
		if err != nil {
			lastErr = err
			continue
		}

		prov, err := h.providerPool.Get(target.Provider)
		if err != nil {
			lastErr = err
			continue
		}

		resp, err := prov.ChatCompletion(ctx, req, target.Model, backend.APIKey.Key)

		// 释放连接
		if bal, _ := h.balancerPool.Get(target.Provider); bal != nil {
			bal.Release(backend)
		}

		if err == nil {
			return resp, nil
		}

		// 处理错误
		if errors.IsRateLimitError(err) {
			h.markBackendUnhealthy(target.Provider, backend)
			lastErr = err
			continue
		}

		if errors.IsRetryable(err) {
			lastErr = err
			continue
		}

		// 不可重试的错误直接返回
		return nil, err
	}

	return nil, errors.Wrap(errors.ErrCodeProviderError, "all targets failed", lastErr)
}

// selectBackend 选择后端
func (h *ChatHandler) selectBackend(providerName string) (*balancer.Backend, error) {
	bal, err := h.balancerPool.Get(providerName)
	if err != nil {
		return nil, err
	}
	return bal.Select()
}

// markBackendUnhealthy 标记后端不健康
func (h *ChatHandler) markBackendUnhealthy(providerName string, backend *balancer.Backend) {
	if bal, err := h.balancerPool.Get(providerName); err == nil {
		bal.MarkUnhealthy(backend)
	}
}

// writeError 写入错误响应
func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(openai.ErrorResponse{
		Error: openai.ErrorDetail{
			Code:    code,
			Message: message,
			Type:    "error",
		},
	})
}

// writeSSEError 写入 SSE 错误
func writeSSEError(w http.ResponseWriter, flusher http.Flusher, code, message string) {
	errData, _ := json.Marshal(openai.ErrorResponse{
		Error: openai.ErrorDetail{
			Code:    code,
			Message: message,
			Type:    "error",
		},
	})
	fmt.Fprintf(w, "data: %s\n\n", errData)
	flusher.Flush()
}
