package handlers

import (
	"context"
	"log"
	"net/http"
	"time"

	"k8s-wizard/api/models"
	"k8s-wizard/pkg/agent"

	"github.com/gin-gonic/gin"
)

// GetResources 获取资源列表
type ResourcesHandler struct {
	agent *agent.Agent
}

func NewResourcesHandler(a *agent.Agent) *ResourcesHandler {
	return &ResourcesHandler{agent: a}
}

func (h *ResourcesHandler) Handle(c *gin.Context) {
	resourceType := c.DefaultQuery("type", "pod")
	namespace := c.DefaultQuery("namespace", "default")

	log.Printf("📋 获取资源: type=%s, namespace=%s", resourceType, namespace)

	// 构造查询命令
	var query string
	switch resourceType {
	case "pod":
		query = "查看所有 pod"
	case "deployment":
		query = "查看所有 deployment"
	case "service":
		query = "查看所有 service"
	default:
		c.JSON(http.StatusBadRequest, models.ChatResponse{
			Error: "不支持的资源类型: " + resourceType,
		})
		return
	}

	// 添加命名空间信息
	if namespace != "default" {
		query += " 在命名空间 " + namespace
	}

	// 设置超时上下文
	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	// 调用 Agent 获取资源
	result, err := h.agent.ProcessCommand(ctx, query)
	if err != nil {
		log.Printf("❌ 获取资源失败: %v", err)
		c.JSON(http.StatusInternalServerError, models.ChatResponse{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.ChatResponse{
		Result:  result,
		Message: "success",
	})
}
