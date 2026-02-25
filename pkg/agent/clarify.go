package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s-wizard/api/models"
)

// ParseUserIntent 解析用户意图
func (a *Agent) ParseUserIntent(ctx context.Context, userMsg string) (*K8sAction, error) {
	prompt := fmt.Sprintf(`你是一个智能的 Kubernetes 助手。理解用户的自然语言指令，判断意图并提取关键信息。

用户指令: %s

请返回 JSON（只返回 JSON，不要其他文字）:
{
  "action": "操作类型",
  "resource": "资源类型",
  "name": "资源名称",
  "namespace": "命名空间",
  "params": {},
  "is_k8s_operation": true/false,
  "reply": "如果不是 K8s 操作，给用户的友好回复"
}

规则:
1. is_k8s_operation=true 表示这是 K8s 相关操作
2. is_k8s_operation=false 表示这是闲聊、打招呼、提问等，需要在 reply 中回复用户
3. action 可以是任何动词，如: create, get, list, delete, scale, update, describe, logs, exec, apply, restart 等
4. resource 可以是任何 K8s 资源，如: pod, deployment, service, configmap, secret, ingress, pvc, namespace, node 等
5. namespace 规则:
   - 如果用户明确指定命名空间（如 "查看 kube-system 的 pod"），使用指定的命名空间
   - 如果用户说 "所有"、"全部"、"集群中"、"列出" 等，设置 namespace 为空字符串 "" 表示查询所有命名空间
   - 其他情况默认为 "default"
6. 对于非 K8s 问题，设置 is_k8s_operation=false 并在 reply 中给出友好回复

示例:
"查看所有 pod" -> {"action":"get","resource":"pod","name":"","namespace":"","params":{},"is_k8s_operation":true,"reply":""}
"看看有哪些 deployment" -> {"action":"get","resource":"deployment","name":"","namespace":"","params":{},"is_k8s_operation":true,"reply":""}
"查看 kube-system 的 pod" -> {"action":"get","resource":"pod","name":"","namespace":"kube-system","params":{},"is_k8s_operation":true,"reply":""}
"部署一个 nginx" -> {"action":"create","resource":"deployment","name":"nginx","namespace":"default","params":{"replicas":1},"is_k8s_operation":true,"reply":""}
"把 web 扩容到 5 个" -> {"action":"scale","resource":"deployment","name":"web","namespace":"default","params":{"replicas":5},"is_k8s_operation":true,"reply":""}
"你好" -> {"action":"","resource":"","name":"","namespace":"default","params":{},"is_k8s_operation":false,"reply":"你好！我是 K8s Wizard。"}`, userMsg)

	a.mu.RLock()
	llm := a.llm
	a.mu.RUnlock()

	llmOutput, err := llm.Chat(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	llmOutput = cleanMarkdownJSON(llmOutput)
	fmt.Printf("🔍 LLM 原始输出: %s\n", llmOutput)

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

case "get", "list", "show":
		// 查看操作 - 检查资源类型是否有效
		validResources := map[string]bool{
			"pod": true, "pods": true,
			"deployment": true, "deployments": true, "deploy": true,
			"service": true, "services": true, "svc": true,
			"configmap": true, "configmaps": true, "cm": true,
			"secret": true, "secrets": true,
			"ingress": true, "ingresses": true,
			"pvc": true, "persistentvolumeclaim": true,
			"namespace": true, "namespaces": true, "ns": true,
			"node": true, "nodes": true,
		}
		if !validResources[action.Resource] {
			return nil
		}
		// 查看操作不需要预览，直接执行
		return &models.ActionPreview{
			Type:        "get",
			Resource:    action.Resource,
			Namespace:   ns,
			DangerLevel: "low",
			Summary:     fmt.Sprintf("查看 %s 命名空间中的 %s", ns, action.Resource),
		}

	case "unknown":
		// 未知操作 - 返回 nil 触发帮助消息
		return nil
	}

	return nil
}
