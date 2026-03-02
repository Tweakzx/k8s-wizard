package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"k8s-wizard/api/handlers"
	"k8s-wizard/api/middleware"
	"k8s-wizard/pkg/agent"
	"k8s-wizard/pkg/config"
	"k8s-wizard/pkg/logger"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	logCfg := &logger.Config{
		EnableFile: cfg.Log.EnableFile,
		MaxSize:    cfg.Log.MaxSize,
		MaxBackups: cfg.Log.MaxBackups,
		MaxAge:     cfg.Log.MaxAge,
		Compress:   cfg.Log.Compress,
		Level:      cfg.Log.Level,
		Format:     cfg.Log.Format,
		Console:    cfg.Log.Console,
	}
	if cfg.Log.FilePath != "" {
		logCfg.FilePath = cfg.Log.FilePath
	}

	log, err := logger.Init(logCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	defer log.Close()

	// 使用配置中的端口，允许环境变量覆盖
	port := os.Getenv("PORT")
	if port == "" {
		port = fmt.Sprintf("%d", cfg.API.Port)
		if port == "0" {
			port = "8080"
		}
	}

	// 设置 Gin 为 Release 模式
	gin.SetMode(gin.ReleaseMode)

	logger.Info("🚀 启动 K8s Wizard API Server",
		"port", port,
		"logFile", logCfg.FilePath,
		"logLevel", logCfg.Level,
	)

	// 创建 GraphAgent 实例
	a, err := agent.NewGraphAgentFromConfig()
	if err != nil {
		logger.Error("❌ 创建 GraphAgent 失败", "error", err)
		os.Exit(1)
	}

	logger.Info("🤖 Agent 初始化完成", "model", a.GetModelName())

	// 创建路由
	r := gin.New()

	// 使用自定义日志中间件
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())
	r.Use(requestLogger())

	// 健康检查
	r.GET("/health", handlers.HealthCheck)

	// API 路由组
	api := r.Group("/api")
	{
		chatHandler := handlers.NewChatHandler(a)
		api.POST("/chat", chatHandler.Handle)

		resourcesHandler := handlers.NewResourcesHandler(a)
		api.GET("/resources", resourcesHandler.Handle)

		// 配置相关路由
		configHandler := handlers.NewConfigHandler(a)
		api.GET("/config/model", configHandler.GetModelInfo)
		api.PUT("/config/model", configHandler.SetModel)
		api.GET("/config", configHandler.GetConfig)
	}

	// 静态文件服务（为前端构建产物提供服务）
	r.Static("/assets", "./web/dist/assets")
	r.StaticFile("/", "./web/dist/index.html")

	// 记录 API 端点
	logger.Info("📡 API 端点已注册",
		"endpoints", []string{
			"GET  /health",
			"POST /api/chat",
			"GET  /api/resources",
			"GET  /api/config/model",
			"PUT  /api/config/model",
			"GET  /api/config",
		},
	)

	bindAddr := cfg.API.Host + ":" + port
	if cfg.API.Host == "" {
		bindAddr = ":" + port
	}

	// 启动服务器（优雅关闭）
	srv := &http.Server{
		Addr:    bindAddr,
		Handler: r,
	}

	// 在 goroutine 中启动服务器
	go func() {
		logger.Info("🌐 服务器启动", "address", bindAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("❌ 启动服务器失败", "error", err)
			os.Exit(1)
		}
	}()

	// 等待中断信号以优雅关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("🛑 正在关闭服务器...")

	// 给服务器 5 秒时间完成当前处理的请求
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("❌ 服务器强制关闭", "error", err)
	}

	logger.Info("✅ 服务器已退出")
}

// requestLogger 创建请求日志中间件
func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		// 记录请求
		logger.Info("HTTP 请求",
			"method", c.Request.Method,
			"path", path,
			"query", query,
			"status", status,
			"latency", latency.String(),
			"clientIP", c.ClientIP(),
		)
	}
}
