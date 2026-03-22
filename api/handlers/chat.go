package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"k8s-wizard/api/models"
	"k8s-wizard/pkg/agent"

	"github.com/gin-gonic/gin"
)

// ChatHandler 处理聊天请求
type ChatHandler struct {
	agent agent.AgentInterface
}

type threadAwareAgent interface {
	ProcessCommandWithClarificationAndThread(
		ctx context.Context,
		userMsg string,
		formData map[string]interface{},
		confirm *bool,
		threadID string,
	) (result string, clarification *models.ClarificationRequest, actionPreview *models.ActionPreview, err error)
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

	log.Printf(
		"📨 收到聊天请求: content=[REDACTED len=%d], formData=%v, confirm=%v, sessionId=%s",
		len(req.Content),
		redactFormData(req.FormData),
		req.Confirm,
		req.SessionID,
	)

	// 设置超时上下文
	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	var (
		result        string
		clarification *models.ClarificationRequest
		actionPreview *models.ActionPreview
		err           error
	)

	// If the agent supports thread-aware processing, wire sessionId through.
	// Otherwise, fall back to the existing method for compatibility.
	if threadAgent, ok := h.agent.(threadAwareAgent); ok {
		result, clarification, actionPreview, err = threadAgent.ProcessCommandWithClarificationAndThread(
			ctx,
			req.Content,
			req.FormData,
			req.Confirm,
			req.SessionID,
		)
	} else {
		result, clarification, actionPreview, err = h.agent.ProcessCommandWithClarification(
			ctx,
			req.Content,
			req.FormData,
			req.Confirm,
		)
	}

	if err != nil {
		sanitizedErr := sanitizeProviderError(err)
		log.Printf("❌ 处理失败: %s", sanitizedErr)
		c.JSON(http.StatusInternalServerError, models.ChatResponse{
			Error: sanitizedErr,
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
		log.Printf("✅ 处理成功: result=[REDACTED len=%d]", len(result))
	}

	c.JSON(http.StatusOK, resp)
}

var sensitiveFieldPattern = regexp.MustCompile(`(?i)(password|passwd|secret|token|api[-_]?key|authorization|credential|private[-_]?key|client[-_]?secret|access[-_]?token|refresh[-_]?token|session|cookie|prompt|content|yaml|manifest|body)`)
var statusCodePattern = regexp.MustCompile(`\b(4\d{2}|5\d{2})\b`)

func redactFormData(formData map[string]interface{}) map[string]interface{} {
	if formData == nil {
		return nil
	}

	redacted := make(map[string]interface{}, len(formData))
	for k, v := range formData {
		if sensitiveFieldPattern.MatchString(strings.TrimSpace(k)) {
			redacted[k] = "[REDACTED]"
			continue
		}
		redacted[k] = redactValue(v)
	}
	return redacted
}

func redactValue(v interface{}) interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		return redactFormData(t)
	case []interface{}:
		out := make([]interface{}, len(t))
		for i, item := range t {
			out[i] = redactValue(item)
		}
		return out
	case string:
		trimmed := strings.TrimSpace(t)
		if trimmed == "" {
			return ""
		}
		return "[REDACTED]"
	default:
		return v
	}
}

func sanitizeProviderError(err error) string {
	if err == nil {
		return "请求失败"
	}

	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return "请求失败"
	}

	if status := statusCodePattern.FindString(msg); status != "" {
		switch status {
		case "400":
			return "上游服务拒绝了请求"
		case "401", "403":
			return "上游服务认证失败"
		case "404":
			return "上游服务资源不存在"
		case "408", "504":
			return "上游服务超时"
		case "429":
			return "上游服务请求过于频繁，请稍后重试"
		default:
			if strings.HasPrefix(status, "5") {
				return "上游服务暂时不可用，请稍后重试"
			}
			return "请求处理失败"
		}
	}

	lowerMsg := strings.ToLower(msg)
	if strings.Contains(lowerMsg, "timeout") || strings.Contains(lowerMsg, "deadline exceeded") {
		return "上游服务超时"
	}
	if strings.Contains(lowerMsg, "connection refused") || strings.Contains(lowerMsg, "no such host") {
		return "上游服务连接失败"
	}
	if json.Valid([]byte(msg)) || strings.Contains(msg, "{") || strings.Contains(msg, "[") {
		return "上游服务请求失败"
	}

	return "请求处理失败"
}
