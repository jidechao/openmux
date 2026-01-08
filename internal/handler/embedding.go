package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/openai/openai-go"
	"github.com/openmux/openmux/internal/balancer"
	"github.com/openmux/openmux/internal/config"
	"github.com/openmux/openmux/internal/provider"
	"github.com/openmux/openmux/internal/router"
	"github.com/openmux/openmux/pkg/errors"
	pkgopenai "github.com/openmux/openmux/pkg/openai"
)

// EmbeddingHandler Embedding 处理器
type EmbeddingHandler struct {
	router         *router.Router
	providerPool   *provider.Pool
	balancerPool   *balancer.BalancerPool
}

// NewEmbeddingHandler 创建 Embedding 处理器
func NewEmbeddingHandler(
	router *router.Router,
	providerPool *provider.Pool,
	balancerPool *balancer.BalancerPool,
) *EmbeddingHandler {
	return &EmbeddingHandler{
		router:       router,
		providerPool: providerPool,
		balancerPool: balancerPool,
	}
}

// Handle 处理 Embedding 请求
func (h *EmbeddingHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

	var req pkgopenai.EmbeddingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	targetSelector, err := h.router.Route(req.Model)
	if err != nil {
		if e, ok := err.(*errors.Error); ok {
			writeError(w, http.StatusNotFound, string(e.Code), e.Message)
		} else {
			writeError(w, http.StatusNotFound, "model_not_found", err.Error())
		}
		return
	}

	resp, err := h.handleWithRetry(r.Context(), &req, targetSelector)
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

func (h *EmbeddingHandler) handleWithRetry(
	ctx context.Context,
	req *pkgopenai.EmbeddingRequest,
	targetSelector router.TargetSelector,
) (*openai.CreateEmbeddingResponse, error) {
	var lastErr error

	target, err := targetSelector.Select()
	if err == nil {
		if resp, err := h.tryTarget(ctx, req, target); err == nil {
			return resp, nil
		}
		lastErr = err
	}

	allTargets := targetSelector.GetAll()
	for _, target := range allTargets {
		resp, err := h.tryTarget(ctx, req, &target)
		if err == nil {
			return resp, nil
		}

		if errors.IsRateLimitError(err) {
			lastErr = err
			continue
		}
		if errors.IsRetryable(err) {
			lastErr = err
			continue
		}
		return nil, err
	}

	return nil, errors.Wrap(errors.ErrCodeProviderError, "all targets failed", lastErr)
}

func (h *EmbeddingHandler) tryTarget(
	ctx context.Context,
	req *pkgopenai.EmbeddingRequest,
	target *config.Target,
) (*openai.CreateEmbeddingResponse, error) {
	estimatedTokens := h.estimateTokens(req)
	backend, err := h.selectBackend(target.Provider, estimatedTokens)
	if err != nil {
		return nil, err
	}
	
	actualUsage := 0
	defer func() {
		h.releaseBackend(target.Provider, backend, actualUsage, estimatedTokens)
	}()

	prov, err := h.providerPool.Get(target.Provider)
	if err != nil {
		return nil, err
	}

	resp, err := prov.CreateEmbedding(ctx, req, target.Model, backend.APIKey)

	if resp != nil {
		actualUsage = int(resp.Usage.TotalTokens)
	}

	if err != nil {
		if errors.IsRateLimitError(err) {
			h.markBackendUnhealthy(target.Provider, backend)
		}
		return nil, err
	}

	return resp, nil
}

func (h *EmbeddingHandler) selectBackend(providerName string, estimatedTokens int) (*balancer.Backend, error) {
	bal, err := h.balancerPool.Get(providerName)
	if err != nil {
		return nil, err
	}
	return bal.Select(estimatedTokens)
}

func (h *EmbeddingHandler) releaseBackend(providerName string, backend *balancer.Backend, actualUsage, estimatedTokens int) {
	if bal, err := h.balancerPool.Get(providerName); err == nil {
		bal.Release(backend, actualUsage, estimatedTokens)
	}
}

func (h *EmbeddingHandler) markBackendUnhealthy(providerName string, backend *balancer.Backend) {
	if bal, err := h.balancerPool.Get(providerName); err == nil {
		bal.MarkUnhealthy(backend)
	}
}

func (h *EmbeddingHandler) estimateTokens(req *pkgopenai.EmbeddingRequest) int {
	// 简单估算
	switch v := req.Input.(type) {
	case string:
		return len(v) / 4
	case []interface{}:
		count := 0
		for _, item := range v {
			if s, ok := item.(string); ok {
				count += len(s)
			}
		}
		return count / 4
	default:
		return 100 // 默认值
	}
}