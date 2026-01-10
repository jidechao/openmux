package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/openmux/openmux/internal/auth"
	"github.com/openmux/openmux/internal/balancer"
	"github.com/openmux/openmux/internal/config"
	"github.com/openmux/openmux/internal/handler"
	"github.com/openmux/openmux/internal/middleware"
	"github.com/openmux/openmux/internal/provider"
	"github.com/openmux/openmux/internal/router"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to config file")
	flag.Parse()

	// 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Starting OpenMux server on %s:%d", cfg.Server.Host, cfg.Server.Port)

	// Log active providers
	log.Println("Active Providers:")
	for name, p := range cfg.Providers {
		log.Printf("- %s (Type: %s, BaseURL: %s, Keys: %d)", name, p.Type, p.BaseURL, len(p.APIKeys))
	}

	// Log configured model routes
	log.Println("Configured Model Routes:")
	for name, route := range cfg.ModelRoutes {
		log.Printf("- %s: %d targets (Strategy: %s)", name, len(route.Targets), route.Strategy)
		for _, t := range route.Targets {
			log.Printf("  -> %s/%s (Weight: %d)", t.Provider, t.Model, t.Weight)
		}
	}

	// 初始化组件
	providerPool := provider.InitFromConfig(cfg)
	balancerPool := balancer.InitFromConfig(cfg)
	modelRouter := router.NewRouter(cfg)
	authManager := auth.NewManager(&cfg.Auth)

	// 创建处理器
	chatHandler := handler.NewChatHandler(modelRouter, providerPool, balancerPool)
	embeddingHandler := handler.NewEmbeddingHandler(modelRouter, providerPool, balancerPool)
	rerankHandler := handler.NewRerankHandler(modelRouter, providerPool, balancerPool)
	modelsHandler := handler.NewModelsHandler(modelRouter)
	healthHandler := handler.NewHealthHandler(balancerPool)

	// 创建中间件
	authMiddleware := auth.NewMiddleware(authManager)

	// 设置路由
	mux := http.NewServeMux()
	
	// OpenAI 兼容端点
	mux.Handle("/v1/chat/completions", 
		middleware.Recovery(
			middleware.Logger(
				middleware.CORS(
					authMiddleware.Authenticate(
						http.HandlerFunc(chatHandler.Handle),
					),
				),
			),
		),
	)

	mux.Handle("/v1/embeddings",
		middleware.Recovery(
			middleware.Logger(
				middleware.CORS(
					authMiddleware.Authenticate(
						http.HandlerFunc(embeddingHandler.Handle),
					),
				),
			),
		),
	)

	mux.Handle("/v1/rerank",
		middleware.Recovery(
			middleware.Logger(
				middleware.CORS(
					authMiddleware.Authenticate(
						http.HandlerFunc(rerankHandler.Handle),
					),
				),
			),
		),
	)
	
	mux.Handle("/v1/models",
		middleware.Recovery(
			middleware.Logger(
				middleware.CORS(
					authMiddleware.Authenticate(
						http.HandlerFunc(modelsHandler.Handle),
					),
				),
			),
		),
	)
	
	// 健康检查端点（无需认证）
	mux.Handle("/health",
		middleware.Recovery(
			middleware.Logger(
				http.HandlerFunc(healthHandler.Handle),
			),
		),
	)

	// 创建服务器
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// 启动服务器
	go func() {
		log.Printf("Server listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
