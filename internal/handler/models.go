package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/openmux/openmux/internal/router"
	"github.com/openmux/openmux/pkg/openai"
)

// ModelsHandler 模型列表处理器
type ModelsHandler struct {
	router *router.Router
}

// NewModelsHandler 创建模型处理器
func NewModelsHandler(router *router.Router) *ModelsHandler {
	return &ModelsHandler{
		router: router,
	}
}

// Handle 处理模型列表请求
func (h *ModelsHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

	models := h.router.ListModels()
	
	modelList := openai.ModelList{
		Object: "list",
		Data:   make([]openai.Model, 0, len(models)),
	}

	for _, name := range models {
		modelList.Data = append(modelList.Data, openai.Model{
			ID:      name,
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "openmux",
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(modelList)
}
