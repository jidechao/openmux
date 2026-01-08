package handler

import (
	"encoding/json"
	"net/http"

	"github.com/openmux/openmux/internal/balancer"
)

// HealthHandler 健康检查处理器
type HealthHandler struct {
	balancerPool *balancer.BalancerPool
}

// NewHealthHandler 创建健康检查处理器
func NewHealthHandler(balancerPool *balancer.BalancerPool) *HealthHandler {
	return &HealthHandler{
		balancerPool: balancerPool,
	}
}

// Handle 处理健康检查请求
func (h *HealthHandler) Handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}
