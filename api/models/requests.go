package models

// ChatRequest 聊天请求
type ChatRequest struct {
	Content   string                 `json:"content" binding:"required"`
	Namespace string                 `json:"namespace,omitempty"`
	FormData  map[string]interface{} `json:"formData,omitempty"`  // 表单数据
	Confirm   *bool                  `json:"confirm,omitempty"`    // 确认执行
	SessionID string                 `json:"sessionId,omitempty"`  // 会话 ID
}

// ChatResponse 聊天响应
type ChatResponse struct {
	Result        string                 `json:"result"`
	Message       string                 `json:"message,omitempty"`
	Error         string                 `json:"error,omitempty"`
	Model         string                 `json:"model,omitempty"`
	Clarification *ClarificationRequest  `json:"clarification,omitempty"`
	ActionPreview *ActionPreview         `json:"actionPreview,omitempty"`
	Status        string                 `json:"status,omitempty"` // pending, confirmed, executed
	Suggestions   []Suggestion            `json:"suggestions,omitempty"` // NEW: Add suggestions field
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

// ModelInfoResponse 模型信息响应
type ModelInfoResponse struct {
	Current string            `json:"current"`
	Models  []ModelInfo       `json:"models"`
	Config  map[string]string `json:"config"`
}

// ModelInfo 模型信息
type ModelInfo struct {
	Provider string `json:"provider"`
	ID       string `json:"id"`
	Name     string `json:"name"`
}

// ConfigResponse 配置响应
type ConfigResponse struct {
	Version string `json:"version"`
	Models  struct {
		Primary string `json:"primary"`
	} `json:"models"`
	Providers map[string]ProviderInfo `json:"providers"`
}

// ProviderInfo 提供商信息
type ProviderInfo struct {
	Name    string   `json:"name"`
	Models  []string `json:"models"`
	BaseURL string   `json:"baseUrl"`
}

// SetModelRequest 切换模型请求
type SetModelRequest struct {
	Model string `json:"model" binding:"required"` // 格式: provider/model-id, 例如: glm/glm-4-flash
}

// SetModelResponse 切换模型响应
type SetModelResponse struct {
	Success bool   `json:"success"`
	Model   string `json:"model"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// ClarificationField 澄清字段
type ClarificationField struct {
	Key         string                   `json:"key"`
	Label       string                   `json:"label"`
	Type        string                   `json:"type"`  // text, select, number, textarea
	Placeholder string                   `json:"placeholder,omitempty"`
	Default     interface{}              `json:"default,omitempty"`
	Options     []ClarificationOption    `json:"options,omitempty"`
	Required    bool                     `json:"required"`
	Group       string                   `json:"group,omitempty"` // 用于分组: "basic", "advanced"
}

// ClarificationOption 选项
type ClarificationOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// ClarificationRequest 澄清请求
type ClarificationRequest struct {
	Type        string              `json:"type"`  // form, options, confirm
	Title       string              `json:"title"`
	Description string              `json:"description,omitempty"`
	Fields      []ClarificationField `json:"fields"`
	Action      string              `json:"action,omitempty"` // create, delete, scale, get
}

// ActionPreview 操作预览
type ActionPreview struct {
	Type        string                 `json:"type"`        // create, delete, scale, get
	Resource    string                 `json:"resource"`    // deployment/nginx
	Namespace   string                 `json:"namespace"`
	YAML        string                 `json:"yaml,omitempty"`
	Params      map[string]interface{} `json:"params"`
	DangerLevel string                 `json:"dangerLevel"` // low, medium, high
	Summary     string                 `json:"summary"`     // 人类可读的摘要
}

