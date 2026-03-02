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

// ChatHandler 处理聊天请求
type ChatHandler struct {
	agent agent.AgentInterface
}

func NewChatHandler(a agent.AgentInterface) *ChatHandler {
	return &ChatHandler{agent: a}
}

func (h *ChatHandler) Handle(c *gin.Context) {
	var req models.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ChatResponse{
			Error: "无效的请求格式: " + err.Error(),
		})
		return
	}

    log.Printf("📨 收到聊天请求: %s, formData: %v, confirm: %v", req.Content, req.FormData, req.Confirm)

	// 设置超时上下文
	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	// 使用新的带澄清流程的处理方法
	result, clarification, actionPreview, err := h.agent.ProcessCommandWithClarification(
		ctx,
		req.Content,
		req.FormData,
		req.Confirm,
	)

	if err != nil {
		log.Printf("❌ 处理失败: %v", err)
		c.JSON(http.StatusInternalServerError, models.ChatResponse{
			Error: err.Error(),
			Model: h.agent.GetModelName(),
		})
		return
	}

	// 构建响应
	resp := models.ChatResponse{
		Result:  result,
		Message: "success",
		Model:   h.agent.GetModelName(),
	}

	// 如果需要澄清，添加澄清请求
	if clarification != nil {
		resp.Clarification = clarification
		resp.Status = "needs_info"
		log.Printf("❓ 需要澄清: %s", clarification.Title)
	}

	// 如果有操作预览，添加预览
	if actionPreview != nil {
		resp.ActionPreview = actionPreview
		resp.Status = "needs_confirm"
		log.Printf("👀 操作预览: %s", actionPreview.Summary)
	}

	// 如果都成功，标记为已执行
	if clarification == nil && actionPreview == nil {
		resp.Status = "executed"
		log.Printf("✅ 处理成功: %s", result)
	}

	c.JSON(http.StatusOK, resp)
}
