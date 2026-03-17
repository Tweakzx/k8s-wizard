package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"k8s-wizard/api/models"
	"k8s-wizard/pkg/k8s"
	"k8s-wizard/pkg/llm"
	"k8s-wizard/pkg/logger"
)

// ============================================================================
// Parse Intent Node
// ============================================================================

func MakeParseIntentNode(llmClient llm.Client) NodeFunc {
	return func(ctx context.Context, state AgentState) (AgentState, error) {
		if state.Action != nil {
			return state, nil
		}

		prompt := buildIntentPrompt(state.UserMessage)

		llmOutput, err := llmClient.Chat(ctx, prompt)
		if err != nil {
			state.Error = fmt.Errorf("LLM call failed: %w", err)
			state.Status = StatusError
			return state, nil
		}

		llmOutput = cleanMarkdownJSON(llmOutput)
		logger.Debug("LLM 原始输出", "output", llmOutput)

		var action K8sAction
		if err := json.Unmarshal([]byte(llmOutput), &action); err != nil {
			state.Error = fmt.Errorf("parse failed: %w", err)
			state.Status = StatusError
			return state, nil
		}

		state.Action = &action
		state.IsK8sOperation = action.IsK8sOperation

		// Add resource type defaulting if empty
		if action.Resource == "" {
			action.Resource = "deployment"
		}

		if !action.IsK8sOperation {
			state.Reply = action.Reply
			state.Status = StatusChat
		}

		logger.Info("LLM 解析结果", "action", action.Action, "resource", action.Resource, "name", action.Name, "isK8s", action.IsK8sOperation)

		return state, nil
	}
}

func buildIntentPrompt(userMsg string) string {
	return fmt.Sprintf(`你是一个智能的 Kubernetes 助手。理解用户的自然语言指令，判断意图并提取关键信息。

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
}

func cleanMarkdownJSON(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
	}
	if strings.HasSuffix(s, "```") {
		s = strings.TrimSuffix(s, "```")
	}
	return strings.TrimSpace(s)
}

// ============================================================================
// Merge Form Node
// ============================================================================

func MakeMergeFormNode() NodeFunc {
	return func(ctx context.Context, state AgentState) (AgentState, error) {
		if state.Action == nil || state.FormData == nil {
			return state, nil
		}

		mergeFormData(state.Action, state.FormData)
		logger.Info("合并 formData", "name", state.Action.Name, "image", state.Action.Params["image"], "replicas", state.Action.Params["replicas"])

		return state, nil
	}
}

func mergeFormData(action *K8sAction, formData map[string]interface{}) {
	if action.Params == nil {
		action.Params = make(map[string]interface{})
	}
	if name, ok := formData["name"].(string); ok && name != "" {
		action.Name = name
	}
	if ns, ok := formData["namespace"].(string); ok && ns != "" {
		action.Namespace = ns
	}
	for key, value := range formData {
		if key == "name" || key == "namespace" {
			continue
		}
		action.Params[key] = value
	}
}

// ============================================================================
// Check Clarify Node
// ============================================================================

func MakeCheckClarifyNode() NodeFunc {
	return func(ctx context.Context, state AgentState) (AgentState, error) {
		if state.Action == nil {
			return state, nil
		}

		clarReq, needsInfo := checkNeedsClarification(state.Action)
		state.NeedsClarification = needsInfo
		state.ClarificationRequest = clarReq
		logger.Info("检查澄清", "needsInfo", needsInfo)

		return state, nil
	}
}

func checkNeedsClarification(action *K8sAction) (*models.ClarificationRequest, bool) {
	switch action.Action {
	case "create", "deploy":
		return checkCreateClarification(action)
	case "scale":
		return checkScaleClarification(action)
	case "delete":
		return checkDeleteClarification(action)
	default:
		return nil, false
	}
}

func checkCreateClarification(action *K8sAction) (*models.ClarificationRequest, bool) {
	var fields []models.ClarificationField
	needInfo := false

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

	image, _ := action.Params["image"].(string)
	if image == "" {
		fields = append(fields, models.ClarificationField{
			Key: "image", Label: "镜像地址", Type: "text", Required: true, Group: "basic",
			Placeholder: "例如: nginx:latest",
		})
		needInfo = true
	} else {
		fields = append(fields, models.ClarificationField{
			Key: "image", Label: "镜像地址", Type: "text", Default: image, Group: "basic",
		})
	}

	replicas := getReplicas(action.Params)
	fields = append(fields, models.ClarificationField{
		Key: "replicas", Label: "副本数", Type: "number", Default: replicas, Group: "basic",
	})

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
		Type:   "form",
		Title:  "📦 创建 Deployment",
		Action: "create",
		Fields: fields,
	}, true
}

func checkScaleClarification(action *K8sAction) (*models.ClarificationRequest, bool) {
	var fields []models.ClarificationField
	needInfo := false

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

	if _, ok := action.Params["replicas"]; !ok {
		fields = append(fields, models.ClarificationField{
			Key: "replicas", Label: "副本数", Type: "number", Required: true, Group: "basic",
		})
		needInfo = true
	} else {
		fields = append(fields, models.ClarificationField{
			Key: "replicas", Label: "副本数", Type: "number", Default: action.Params["replicas"], Group: "basic",
		})
	}

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
		Type:   "form",
		Title:  "⚖️ 扩缩容 Deployment",
		Action: "scale",
		Fields: fields,
	}, true
}

func checkDeleteClarification(action *K8sAction) (*models.ClarificationRequest, bool) {
	var fields []models.ClarificationField
	needInfo := false

	if action.Name == "" {
		fields = append(fields, models.ClarificationField{
			Key: "name", Label: "资源名称", Type: "text", Required: true, Group: "basic",
		})
		needInfo = true
	} else {
		fields = append(fields, models.ClarificationField{
			Key: "name", Label: "资源名称", Type: "text", Default: action.Name, Group: "basic",
		})
	}

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
		Type:   "form",
		Title:  "🗑️ 删除资源",
		Action: "delete",
		Fields: fields,
	}, true
}

func getReplicas(params map[string]interface{}) int {
	switch v := params["replicas"].(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float32:
		return int(v)
	case float64:
		return int(v)
	default:
		return 1
	}
}

// ============================================================================
// Generate Preview Node
// ============================================================================

func MakeGeneratePreviewNode() NodeFunc {
	return func(ctx context.Context, state AgentState) (AgentState, error) {
		if state.Action == nil {
			return state, nil
		}

		preview := generateActionPreview(state.Action)
		state.ActionPreview = preview
		logger.Info("操作预览", "hasPreview", preview != nil)

		return state, nil
	}
}

func generateActionPreview(action *K8sAction) *models.ActionPreview {
	if action == nil {
		return nil
	}

	ns := action.Namespace
	if ns == "" && action.Action != "get" && action.Action != "list" {
		ns = "default"
	}

	switch action.Action {
	case "create", "deploy":
		image, _ := action.Params["image"].(string)
		if image == "" {
			image = action.Name + ":latest"
		}
		replicas := getReplicas(action.Params)
		yaml := generateDeploymentYAML(action.Name, ns, image, replicas)
		return &models.ActionPreview{
			Type:        "create",
			Resource:    "deployment/" + action.Name,
			Namespace:   ns,
			YAML:        yaml,
			DangerLevel: "low",
			Summary:     fmt.Sprintf("创建 Deployment %s (副本: %d, 镜像: %s)", action.Name, replicas, image),
			Params:      action.Params,
		}

	case "scale":
		replicas := getReplicas(action.Params)
		return &models.ActionPreview{
			Type:        "scale",
			Resource:    "deployment/" + action.Name,
			Namespace:   ns,
			DangerLevel: "medium",
			Summary:     fmt.Sprintf("扩缩容 Deployment %s 到 %d 个副本", action.Name, replicas),
			Params:      action.Params,
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
		return &models.ActionPreview{
			Type:        "get",
			Resource:    action.Resource,
			Namespace:   ns,
			DangerLevel: "low",
			Summary:     fmt.Sprintf("查看 %s 命名空间中的 %s", ns, action.Resource),
		}
	}

	return nil
}

func generateDeploymentYAML(name, namespace, image string, replicas int) string {
	return fmt.Sprintf(`apiVersion: apps/v1
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
    metadata:
      labels:
        app: %s
    spec:
      containers:
      - name: %s
        image: %s
        ports:
        - containerPort: 80
`, name, namespace, replicas, name, name, name, image)
}

// ============================================================================
// Execute Node
// ============================================================================

func MakeExecuteNode(client k8s.Client) NodeFunc {
	return func(ctx context.Context, state AgentState) (AgentState, error) {
		if state.Action == nil {
			state.Error = fmt.Errorf("no action to execute")
			state.Status = StatusError
			return state, nil
		}

		if client == nil {
			state.Error = fmt.Errorf("kubernetes client not configured")
			state.Status = StatusError
			return state, nil
		}

		result, err := executeAction(ctx, client, state.Action)
		if err != nil {
			state.Error = err
			state.Status = StatusError
			return state, nil
		}

		state.Result = result
		state.Status = StatusExecuted
		return state, nil
	}
}

func executeAction(ctx context.Context, client k8s.Client, action *K8sAction) (string, error) {
	namespace := action.Namespace
	if namespace == "" && action.Action != "get" && action.Action != "list" {
		namespace = "default"
	}

	switch action.Action {
	case "create", "deploy":
		image, _ := action.Params["image"].(string)
		if image == "" {
			image = action.Name + ":latest"
		}
		replicas := int32(getReplicas(action.Params))
		return client.CreateDeployment(ctx, namespace, action.Name, image, replicas)

	case "get", "list", "show":
		return client.GetResources(ctx, namespace, action.Resource)

	case "scale":
		replicas := int32(getReplicas(action.Params))
		return client.ScaleDeployment(ctx, namespace, action.Name, replicas)

	case "delete":
		return client.DeleteResource(ctx, namespace, action.Name, action.Resource)

	default:
		return "", fmt.Errorf("unsupported action: %s", action.Action)
	}
}
