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
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	log, err := setupLogger(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to setup logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Close()

	ag, err := agent.NewGraphAgentFromConfig()
	if err != nil {
		logger.Error("failed to create agent", "error", err)
		os.Exit(1)
	}

	logger.Info("agent initialized", "model", ag.GetModelName())

	authCfg := middleware.NewAuthConfigFromEnv()
	logger.Info("auth configuration initialized",
		"requireAuth", authCfg.RequireAuth,
		"hasApiToken", authCfg.UserToken != "",
		"hasAdminToken", authCfg.AdminToken != "",
	)

	router := setupRouter(ag, authCfg)
	srv := &http.Server{
		Addr:    getBindAddr(cfg),
		Handler: router,
	}

	go startServer(srv)
	waitForShutdown(srv)
}

func setupLogger(cfg *config.Config) (*logger.Logger, error) {
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
	return logger.Init(logCfg)
}

func getBindAddr(cfg *config.Config) string {
	port := os.Getenv("PORT")
	if port == "" {
		port = fmt.Sprintf("%d", cfg.API.Port)
		if port == "0" {
			port = "8080"
		}
	}

	if cfg.API.Host != "" {
		return cfg.API.Host + ":" + port
	}
	return ":" + port
}

func setupRouter(ag agent.AgentInterface, authCfg middleware.AuthConfig) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())
	r.Use(requestLogger())

	r.GET("/health", handlers.HealthCheck)

	api := r.Group("/api")
	api.Use(middleware.TokenAuth(authCfg))
	{
		chatHandler := handlers.NewChatHandler(ag)
		api.POST("/chat", middleware.RequireDangerousOperationAuth(), chatHandler.Handle)

		resourcesHandler := handlers.NewResourcesHandler(ag)
		api.GET("/resources", resourcesHandler.Handle)

		configHandler := handlers.NewConfigHandler(ag)
		api.GET("/config/model", configHandler.GetModelInfo)
		api.PUT("/config/model", configHandler.SetModel)
		api.GET("/config", configHandler.GetConfig)
	}

	r.Static("/assets", "./web/dist/assets")
	r.StaticFile("/", "./web/dist/index.html")

	logger.Info("api endpoints registered", "endpoints", []string{
		"GET  /health",
		"POST /api/chat",
		"GET  /api/resources",
		"GET  /api/config/model",
		"PUT  /api/config/model",
		"GET  /api/config",
	})

	return r
}

func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		logger.Info("http request",
			"method", c.Request.Method,
			"path", path,
			"query", query,
			"status", c.Writer.Status(),
			"latency", time.Since(start).String(),
			"clientIP", c.ClientIP(),
		)
	}
}

func startServer(srv *http.Server) {
	logger.Info("server starting", "address", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func waitForShutdown(srv *http.Server) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced shutdown", "error", err)
	}

	logger.Info("server exited")
}
