package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"k8s-wizard/api/handlers"
	"k8s-wizard/api/middleware"
	"k8s-wizard/pkg/agent"
	"k8s-wizard/pkg/config"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("❌ 加载配置失败: %v", err)
	}

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

	// 创建 Agent 实例
	a, err := agent.NewAgent()
	if err != nil {
		log.Fatalf("❌ 创建 Agent 失败: %v", err)
	}

	// 创建路由
	r := gin.Default()

	// 添加中间件
	r.Use(middleware.CORS())

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

	// 启动服务器
	log.Printf("🚀 K8s Wizard API Server 启动在端口 %s", port)
	log.Printf("🤖 当前模型: %s", a.GetModelName())
	log.Printf("📡 API 端点:")
	log.Printf("   GET  /health            - 健康检查")
	log.Printf("   POST /api/chat          - 聊天接口")
	log.Printf("   GET  /api/resources     - 获取资源列表")
	log.Printf("   GET  /api/config/model   - 获取当前模型信息")
	log.Printf("   PUT  /api/config/model   - 切换模型")
	log.Printf("   GET  /api/config        - 获取完整配置")

	bindAddr := cfg.API.Host + ":" + port
	if cfg.API.Host == "" {
		bindAddr = ":" + port
	}

	if err := r.Run(bindAddr); err != nil && err != http.ErrServerClosed {
		log.Fatalf("❌ 启动服务器失败: %v", err)
	}
}
