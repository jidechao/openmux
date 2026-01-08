package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

type contextKey string

const ClientContextKey contextKey = "client"

// Middleware 认证中间件
type Middleware struct {
	manager *Manager
}

// NewMiddleware 创建认证中间件
func NewMiddleware(manager *Manager) *Middleware {
	return &Middleware{
		manager: manager,
	}
}

// Authenticate 认证处理
func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 提取 API Key
		apiKey := extractAPIKey(r)
		
		// 验证 API Key
		client, ok := m.manager.Verify(apiKey)
		if !ok {
			writeError(w, http.StatusUnauthorized, "invalid_api_key", "Invalid API key")
			return
		}
		
		// 检查限流
		if !m.manager.CheckRateLimit(client) {
			writeError(w, http.StatusTooManyRequests, "rate_limit_exceeded", "Rate limit exceeded")
			return
		}
		
		// 将客户端信息存入 context
		ctx := context.WithValue(r.Context(), ClientContextKey, client)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// extractAPIKey 从请求中提取 API Key
func extractAPIKey(r *http.Request) string {
	// 从 Authorization header 提取
	auth := r.Header.Get("Authorization")
	if auth != "" {
		// Bearer token
		if strings.HasPrefix(auth, "Bearer ") {
			return strings.TrimPrefix(auth, "Bearer ")
		}
		return auth
	}
	
	// 从 query parameter 提取
	return r.URL.Query().Get("api_key")
}

// writeError 写入错误响应
func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

// GetClient 从 context 获取客户端信息
func GetClient(ctx context.Context) *ClientInfo {
	client, _ := ctx.Value(ClientContextKey).(*ClientInfo)
	return client
}
