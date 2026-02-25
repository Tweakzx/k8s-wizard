package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s-wizard/api/models"
)

// ParseUserIntent 解析用户意图
func (a *Agent) ParseUserIntent(ctx context.Context, userMsg string) (*K8sAction, error) {
	prompt := fmt.Sprintf(`你是 Kubernetes 助手。分析用户指令返回 JSON。

用户指令: %s

返回格式（只返回 JSON）:
{
  "action": "create|get|scale|delete",
  "resource": "pod|deployment|service",
  "name": "名称（未指定为空）",
  "namespace": "default",
  "params": {"image": "", "replicas": 0}
}

示例:
"部署 nginx" -> {"action":"create","resource":"deployment","name":"nginx","namespace":"default","params":{"image":"","replicas":0}}
"查看 pod" -> {"action":"get","resource":"pod","name":"","namespace":"default","params":{}}
"扩容到 5 个" -> {"action":"scale","resource":"deployment","name":"","namespace":"default","params":{"replicas":5}}`, userMsg)

	a.mu.RLock()
	llm := a.llm
	a.mu.RUnlock()

	llmOutput, err := llm.Chat(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	llmOutput = cleanMarkdownJSON(llmOutput)

	var action K8sAction
	if err := json.Unmarshal([]byte(llmOutput), &action); err != nil {
		return nil, fmt.Errorf("parse failed: %w", err)
	}

	return &action, nil
}

// CheckNeedsClarification 检查是否需要澄清
func (a *Agent) CheckNeedsClarification(action *K8sAction) (*models.ClarificationRequest, bool) {
	switch action.Action {
	case "create", "deploy":
		return a.checkCreateClarification(action)
	case "scale":
		return a.checkScaleClarification(action)
	case "delete":
		return a.checkDeleteClarification(action)
	default:
		return nil, false
	}
}

func (a *Agent) checkCreateClarification(action *K8sAction) (*models.ClarificationRequest, bool) {
	var fields []models.ClarificationField
	needInfo := false

	// 名称
	if action.Name == "" {
		fields = append(fields, models.ClarificationField{
			Key: "name", Label: "应用名称", Type: "text", Required: true, Group: "basic",
		})
		needInfo = true
	} else {
		fields = append(fields, models.ClarificationField{
			Key: "name", Label: "应用名称", Type: "text", Default: action.Name, Group: "basic",
		})
	}

	// 镜像
	image, _ := action.Params["image"].(string)
	if image == "" {
		fields = append(fields, models.ClarificationField{
			Key: "image", Label: "镜像地址", Type: "text", Placeholder: "nginx:latest", Required: true, Group: "basic",
		})
		needInfo = true
	} else {
		fields = append(fields, models.ClarificationField{
			Key: "image", Label: "镜像地址", Type: "text", Default: image, Group: "basic",
		})
	}

	// 副本数
	replicas := 1
	if r, ok := action.Params["replicas"].(float64); ok && r > 0 {
		replicas = int(r)
	}
	fields = append(fields, models.ClarificationField{
		Key: "replicas", Label: "副本数", Type: "number", Default: replicas, Group: "basic",
	})

	// 命名空间
	ns := action.Namespace
	if ns == "" {
		ns = "default"
	}
	fields = append(fields, models.ClarificationField{
		Key: "namespace", Label: "命名空间", Type: "text", Default: ns, Group: "basic",
	})

	if !needInfo {
		return nil, false
	}

	return &models.ClarificationRequest{
		Type:        "form",
		Title:       "📦 创建 Deployment",
		Description: "请补充以下信息：",
		Fields:      fields,
		Action:      "create",
	}, true
}

func (a *Agent) checkScaleClarification(action *K8sAction) (*models.ClarificationRequest, bool) {
	var fields []models.ClarificationField
	needInfo := false

	if action.Name == "" {
		fields = append(fields, models.ClarificationField{
			Key: "name", Label: "Deployment 名称", Type: "text", Required: true,
		})
		needInfo = true
	}

	replicas, hasReplicas := action.Params["replicas"].(float64)
	if !hasReplicas || replicas == 0 {
		fields = append(fields, models.ClarificationField{
			Key: "replicas", Label: "目标副本数", Type: "number", Required: true,
		})
		needInfo = true
	}

	if !needInfo {
		return nil, false
	}

	return &models.ClarificationRequest{
		Type:   "form",
		Title:  "⚙️ 扩缩容",
		Fields: fields,
		Action: "scale",
	}, true
}

func (a *Agent) checkDeleteClarification(action *K8sAction) (*models.ClarificationRequest, bool) {
	if action.Name == "" {
		return &models.ClarificationRequest{
			Type:   "form",
			Title:  "🗑️ 删除资源",
			Fields: []models.ClarificationField{{Key: "name", Label: "资源名称", Type: "text", Required: true}},
			Action: "delete",
		}, true
	}
	return nil, false
}

// MergeFormData 合并表单数据
func (a *Agent) MergeFormData(action *K8sAction, formData map[string]interface{}) {
	if name, ok := formData["name"].(string); ok && name != "" {
		action.Name = name
	}
	if ns, ok := formData["namespace"].(string); ok && ns != "" {
		action.Namespace = ns
	}
	if action.Params == nil {
		action.Params = make(map[string]interface{})
	}
	if image, ok := formData["image"].(string); ok {
		action.Params["image"] = image
	}
	// 处理 replicas，支持多种数值类型
	switch v := formData["replicas"].(type) {
	case float64:
		action.Params["replicas"] = int(v)
	case float32:
		action.Params["replicas"] = int(v)
	case int:
		action.Params["replicas"] = v
	case int64:
		action.Params["replicas"] = int(v)
	case int32:
		action.Params["replicas"] = int(v)
	}
}

// GenerateActionPreview 生成操作预览
func (a *Agent) GenerateActionPreview(action *K8sAction) *models.ActionPreview {
	ns := action.Namespace
	if ns == "" {
		ns = "default"
	}

	switch action.Action {
	case "create", "deploy":
		image, _ := action.Params["image"].(string)
		replicas := 1
		// 支持多种数值类型
		switch v := action.Params["replicas"].(type) {
		case float64:
			replicas = int(v)
		case float32:
			replicas = int(v)
		case int:
			replicas = v
		case int64:
			replicas = int(v)
		case int32:
			replicas = int(v)
		}

		yaml := fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
spec:
  replicas: %d
  selector:
    matchLabels:
      app: %s
  template:
    spec:
      containers:
      - name: %s
        image: %s`, action.Name, ns, replicas, action.Name, action.Name, image)

		return &models.ActionPreview{
			Type:        "create",
			Resource:    "deployment/" + action.Name,
			Namespace:   ns,
			YAML:        yaml,
			DangerLevel: "low",
			Summary:     fmt.Sprintf("创建 Deployment %s (副本: %d, 镜像: %s)", action.Name, replicas, image),
		}

	case "scale":
		replicas := 1
		if r, ok := action.Params["replicas"].(float64); ok {
			replicas = int(r)
		}
		return &models.ActionPreview{
			Type:        "scale",
			Resource:    "deployment/" + action.Name,
			Namespace:   ns,
			DangerLevel: "medium",
			Summary:     fmt.Sprintf("扩缩容 %s 到 %d 副本", action.Name, replicas),
		}

	case "delete":
		return &models.ActionPreview{
			Type:        "delete",
			Resource:    action.Resource + "/" + action.Name,
			Namespace:   ns,
			DangerLevel: "high",
			Summary:     fmt.Sprintf("删除 %s/%s", ns, action.Name),
		}
	}

	return nil
}
