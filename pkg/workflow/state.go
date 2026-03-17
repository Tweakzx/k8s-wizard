package workflow

import (
	"context"

	"k8s-wizard/api/models"
	"k8s-wizard/pkg/k8s"
	"k8s-wizard/pkg/k8s/handlers"
	"k8s-wizard/pkg/llm"
	"k8s-wizard/pkg/tools"
)

// AgentState is the state that flows through the K8s Agent workflow.
// It contains all the information needed for each node to process.
type AgentState struct {
	// === Input ===
	UserMessage string                 // 用户输入的自然语言
	FormData    map[string]interface{} // 用户填写的表单数据
	Confirm     *bool                  // 用户是否确认执行
	ThreadID    string                 // 会话 ID，用于状态持久化

	// === Parsed Intent ===
	Action         *K8sAction // 解析后的 K8s 操作
	IsK8sOperation bool       // 是否是 K8s 操作（非闲聊）

	// === Clarification ===
	ClarificationRequest *models.ClarificationRequest // 需要用户填写的表单
	NeedsClarification   bool                         // 是否需要澄清

	// === Preview ===
	ActionPreview *models.ActionPreview // 操作预览

	// === Execution ===
	Result string // 执行结果
	Error  error  // 执行错误

	// === Status ===
	Status string // pending, needs_info, needs_confirm, executed, error, chat

	// === Chat Response (for non-K8s operations) ===
	Reply string // AI 对闲聊的回复

	// === Suggestions ===
	Suggestions []models.Suggestion // 智能建议（从集群状态生成）
}

// K8sAction represents a parsed K8s operation intent.
type K8sAction struct {
	Action         string                 `json:"action"`    // create, get, list, delete, scale, update, describe
	Resource       string                 `json:"resource"`  // pod, deployment, service, configmap, secret, ingress, pvc, namespace, node
	Name           string                 `json:"name"`      // 资源名称
	Namespace      string                 `json:"namespace"` // 命名空间（空=所有命名空间）
	Params         map[string]interface{} `json:"params"`    // 操作参数
	IsK8sOperation bool                   `json:"is_k8s_operation"`
	Reply          string                 `json:"reply"` // 如果不是 K8s 操作，这里是 AI 的回复
}

// Dependencies holds the dependencies needed by the workflow nodes.
type Dependencies struct {
	K8sClient    k8s.Client
	LLM          llm.Client
	ModelName    string
	ToolRegistry *tools.Registry    // NEW - Phase 1
	PromptLoader interface{}        // NEW - Phase 2 (placeholder)
	SubGraphMgr  interface{}        // NEW - Phase 3 (placeholder)
	ContextMgr   *ContextManager    // NEW - Phase 3
	SuggestionEngine *SuggestionEngine // NEW - Intelligent suggestions
}

// Status constants
const (
	StatusPending      = "pending"
	StatusNeedsInfo    = "needs_info"
	StatusNeedsConfirm = "needs_confirm"
	StatusExecuted     = "executed"
	StatusError        = "error"
	StatusChat         = "chat"
)

// NodeFunc is the type for workflow node functions.
type NodeFunc func(ctx context.Context, state AgentState) (AgentState, error)

// Ensure handlers package is imported for future handler registration
// This will be used in later phases when integrating the full handler system
var _ = handlers.NewBaseHandler
