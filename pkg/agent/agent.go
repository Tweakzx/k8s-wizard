package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"k8s-wizard/api/models"
	"k8s-wizard/pkg/config"
)

// LLMClient LLM 客户端接口
type LLMClient interface {
	Chat(ctx context.Context, prompt string) (string, error)
	GetModel() string
}

// ConfiguredLLMClient 带配置信息的 LLM 客户端
type ConfiguredLLMClient struct {
	Provider    string
	ModelID     string
	BaseURL     string
	AuthType    string
	apiKey      string
	httpClient  *http.Client
	apiFormat   string // "openai-completions" or "anthropic"
}

func NewConfiguredLLMClient(provider string, modelID string, providerConfig config.Provider) (*ConfiguredLLMClient, error) {
	apiKey, err := config.GetAPIKey(provider)
	if err != nil {
		return nil, err
	}

	return &ConfiguredLLMClient{
		Provider:   provider,
		ModelID:    modelID,
		BaseURL:    providerConfig.BaseURL,
		AuthType:   providerConfig.Auth,
		apiFormat:  providerConfig.API,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}, nil
}

func (c *ConfiguredLLMClient) GetModel() string {
	return fmt.Sprintf("%s/%s", c.Provider, c.ModelID)
}

func (c *ConfiguredLLMClient) Chat(ctx context.Context, prompt string) (string, error) {
	switch c.apiFormat {
	case "anthropic":
		return c.chatAnthropic(ctx, prompt)
	case "openai-completions", "":
		return c.chatOpenAIFormat(ctx, prompt)
	default:
		return "", fmt.Errorf("unsupported API format: %s", c.apiFormat)
	}
}

func (c *ConfiguredLLMClient) chatAnthropic(ctx context.Context, prompt string) (string, error) {
	requestBody := map[string]interface{}{
		"model":      c.ModelID,
		"max_tokens": 4096,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/messages", c.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("anthropic-dangerous-direct-browser-access", "true")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error (%s): %s", c.Provider, string(body))
	}

	var response struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}

	if len(response.Content) == 0 {
		return "", fmt.Errorf("empty response")
	}

	var result strings.Builder
	for _, block := range response.Content {
		if block.Type == "text" {
			result.WriteString(block.Text)
		}
	}

	return result.String(), nil
}

func (c *ConfiguredLLMClient) chatOpenAIFormat(ctx context.Context, prompt string) (string, error) {
	requestBody := map[string]interface{}{
		"model": c.ModelID,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	// Determine endpoint based on provider
	var endpoint string
	switch c.Provider {
	case "deepseek":
		endpoint = c.BaseURL + "/chat/completions"
	default:
		endpoint = c.BaseURL + "/chat/completions"
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error (%s): %s", c.Provider, string(body))
	}

	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("empty response")
	}

	return response.Choices[0].Message.Content, nil
}

// K8sAction 表示一个 K8s 操作
type K8sAction struct {
	Action    string                 `json:"action"`
	Resource  string                 `json:"resource"`
	Name      string                 `json:"name"`
	Namespace string                 `json:"namespace"`
	Params    map[string]interface{} `json:"params"`
}

// Agent 是主控制器
type Agent struct {
	client    *kubernetes.Clientset
	llm       LLMClient
	modelName string
	mu        sync.RWMutex
}

// NewAgent 创建新的 Agent 实例
func NewAgent() (*Agent, error) {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// 初始化 K8s client
	var k8sConfig *rest.Config
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig != "" {
		k8sConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		k8sConfig, err = rest.InClusterConfig()
	}
	if err != nil {
		// 如果都失败，尝试默认位置
		k8sConfig, err = clientcmd.BuildConfigFromFlags("", os.Getenv("HOME")+"/.kube/config")
		if err != nil {
			return nil, fmt.Errorf("failed to create k8s config: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	// 获取模型配置
	modelString := cfg.Agents.Defaults.Model.Primary
	if envModel := os.Getenv("K8S_WIZARD_MODEL"); envModel != "" {
		modelString = envModel
	}

	provider, modelID, err := cfg.GetModelProvider(modelString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse model config: %w", err)
	}

	providerConfig, ok := cfg.Models.Providers[provider]
	if !ok {
		return nil, fmt.Errorf("provider not configured: %s", provider)
	}

	// 创建 LLM client
	llm, err := NewConfiguredLLMClient(provider, modelID, providerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}

	modelName := llm.GetModel()
	fmt.Printf("🤖 使用模型: %s\n", modelName)

	return &Agent{
		client:    clientset,
		llm:       llm,
		modelName: modelName,
	}, nil
}

// GetModelName 返回当前使用的模型名称
func (a *Agent) GetModelName() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.modelName
}

// SetModel 切换模型
func (a *Agent) SetModel(modelString string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	provider, modelID, err := cfg.GetModelProvider(modelString)
	if err != nil {
		return fmt.Errorf("failed to parse model config: %w", err)
	}

	providerConfig, ok := cfg.Models.Providers[provider]
	if !ok {
		return fmt.Errorf("provider not configured: %s", provider)
	}

	// 创建新的 LLM client
	newLLM, err := NewConfiguredLLMClient(provider, modelID, providerConfig)
	if err != nil {
		return fmt.Errorf("failed to create LLM client: %w", err)
	}

	// 更新 Agent
	a.mu.Lock()
	a.llm = newLLM
	a.modelName = newLLM.GetModel()
	a.mu.Unlock()

	fmt.Printf("🔄 模型已切换为: %s\n", a.modelName)
	return nil
}

// ProcessCommandWithClarification 处理命令，支持澄清流程
func (a *Agent) ProcessCommandWithClarification(ctx context.Context, userMsg string, formData map[string]interface{}, confirm *bool) (result string, clarification *models.ClarificationRequest, actionPreview *models.ActionPreview, err error) {
	// 1. 调用 LLM 解析用户意图
	action, err := a.ParseUserIntent(ctx, userMsg)
	if err != nil {
		return "", nil, nil, err
	}

	// 2. 如果有表单数据，先合并到 action
	if formData != nil {
		a.MergeFormData(action, formData)
	}

	// 3. 检查是否需要澄清（合并后再检查）
	if clarReq, needsInfo := a.CheckNeedsClarification(action); needsInfo {
		return "", clarReq, nil, nil
	}

	// 4. 生成操作预览
	preview := a.GenerateActionPreview(action)

	// 5. 如果未确认，返回预览等待确认
	if confirm == nil || !*confirm {
		return "请确认以下操作：", nil, preview, nil
	}

	// 6. 执行操作
	result, err = a.executeAction(ctx, *action)
	if err != nil {
		return "", nil, nil, err
	}

	return result, nil, nil, nil
}

// ProcessCommand 处理用户命令
func (a *Agent) ProcessCommand(ctx context.Context, userMsg string) (string, error) {
	// 调用 LLM 解析用户意图
	prompt := fmt.Sprintf(`你是一个 Kubernetes 集群操作助手。你的任务是理解用户的自然语言指令，并将其转换为具体的 K8s 操作。

用户指令: %s

请分析这个指令并返回对应的 K8s 操作 JSON。

支持的操作类型：
1. create/deploy - 创建部署
2. get/list/show - 查看资源
3. scale - 扩缩容
4. delete/remove - 删除资源

支持的资源类型：
- pod
- deployment
- service

返回 JSON 格式（只返回 JSON，不要其他文字）:
{
  "action": "create|get|scale|delete",
  "resource": "pod|deployment|service",
  "name": "资源名称",
  "namespace": "命名空间（默认 default）",
  "params": {
    "image": "镜像地址（创建时需要）",
    "replicas": 副本数（数字）
  }
}

示例：
用户: "部署一个 nginx，3个副本"
返回: {"action": "create", "resource": "deployment", "name": "nginx", "namespace": "default", "params": {"image": "nginx:latest", "replicas": 3}}

用户: "查看所有 pod"
返回: {"action": "get", "resource": "pod", "name": "", "namespace": "default", "params": {}}

用户: "把 nginx 扩容到 5 个副本"
返回: {"action": "scale", "resource": "deployment", "name": "nginx", "namespace": "default", "params": {"replicas": 5}}

用户: "删除名为 test 的 pod"
返回: {"action": "delete", "resource": "pod", "name": "test", "namespace": "default", "params": {}}`, userMsg)


	a.mu.RLock()
	llm := a.llm
	a.mu.RUnlock()

	llmOutput, err := llm.Chat(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	if llmOutput == "" {
		return "", fmt.Errorf("empty LLM response")
	}

	// 清理 markdown 格式
	llmOutput = cleanMarkdownJSON(llmOutput)

	// 解析 LLM 返回的 JSON
	var action K8sAction
	if err := json.Unmarshal([]byte(llmOutput), &action); err != nil {
		return "", fmt.Errorf("failed to parse LLM response: %w\nLLM output: %s", err, llmOutput)
	}

	// 执行 K8s 操作
	result, err := a.executeAction(ctx, action)
	if err != nil {
		return "", err
	}

	return result, nil
}

// executeAction 执行具体的 K8s 操作
func (a *Agent) executeAction(ctx context.Context, action K8sAction) (string, error) {
	namespace := action.Namespace
	if namespace == "" {
		namespace = "default"
	}

	switch action.Action {
	case "create", "deploy":
		return a.createDeployment(ctx, namespace, action)
	case "get", "list", "show":
		return a.getResources(ctx, namespace, action)
	case "scale":
		return a.scaleDeployment(ctx, namespace, action)
	case "delete", "remove":
		return a.deleteResource(ctx, namespace, action)
	default:
		return "", fmt.Errorf("不支持的操作: %s", action.Action)
	}
}

// createDeployment 创建 Deployment
func (a *Agent) createDeployment(ctx context.Context, namespace string, action K8sAction) (string, error) {
	image, _ := action.Params["image"].(string)
	if image == "" {
		image = action.Name + ":latest"
	}

	replicas := int32(3)
	if r, ok := action.Params["replicas"].(float64); ok {
		replicas = int32(r)
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: action.Name,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": action.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": action.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  action.Name,
							Image: image,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}

	created, err := a.client.AppsV1().Deployments(namespace).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("创建 deployment 失败: %w", err)
	}

	return fmt.Sprintf("✓ 已创建 Deployment %s/%s，副本数: %d，镜像: %s", namespace, created.Name, *created.Spec.Replicas, image), nil
}

// getResources 获取资源
func (a *Agent) getResources(ctx context.Context, namespace string, action K8sAction) (string, error) {
	switch action.Resource {
	case "pod":
		pods, err := a.client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return "", fmt.Errorf("获取 pod 列表失败: %w", err)
		}
		if len(pods.Items) == 0 {
			return fmt.Sprintf("命名空间 %s 中没有 Pod", namespace), nil
		}
		result := fmt.Sprintf("📦 命名空间 %s 中的 Pod (共 %d 个):\n", namespace, len(pods.Items))
		for _, pod := range pods.Items {
			age := pod.CreationTimestamp.Format("2006-01-02 15:04")
			result += fmt.Sprintf("  • %s (%s) - 年龄: %s\n", pod.Name, pod.Status.Phase, age)
		}
		return result, nil
	case "deployment":
		deps, err := a.client.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return "", fmt.Errorf("获取 deployment 列表失败: %w", err)
		}
		if len(deps.Items) == 0 {
			return fmt.Sprintf("命名空间 %s 中没有 Deployment", namespace), nil
		}
		result := fmt.Sprintf("🚀 命名空间 %s 中的 Deployment (共 %d 个):\n", namespace, len(deps.Items))
		for _, dep := range deps.Items {
			result += fmt.Sprintf("  • %s (副本: %d/%d)\n", dep.Name, dep.Status.ReadyReplicas, dep.Status.Replicas)
		}
		return result, nil
	case "service":
		svcs, err := a.client.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return "", fmt.Errorf("获取 service 列表失败: %w", err)
		}
		if len(svcs.Items) == 0 {
			return fmt.Sprintf("命名空间 %s 中没有 Service", namespace), nil
		}
		result := fmt.Sprintf("🔗 命名空间 %s 中的 Service (共 %d 个):\n", namespace, len(svcs.Items))
		for _, svc := range svcs.Items {
			if len(svc.Spec.Ports) > 0 {
				result += fmt.Sprintf("  • %s (类型: %s, 端口: %d)\n", svc.Name, svc.Spec.Type, svc.Spec.Ports[0].Port)
			} else {
				result += fmt.Sprintf("  • %s (类型: %s)\n", svc.Name, svc.Spec.Type)
			}
		}
		return result, nil
	default:
		return "", fmt.Errorf("不支持的资源类型: %s", action.Resource)
	}
}

// scaleDeployment 扩缩容 Deployment
func (a *Agent) scaleDeployment(ctx context.Context, namespace string, action K8sAction) (string, error) {
	replicas := int32(1)
	if r, ok := action.Params["replicas"].(float64); ok {
		replicas = int32(r)
	}

	// 获取当前 deployment
	dep, err := a.client.AppsV1().Deployments(namespace).Get(ctx, action.Name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("获取 deployment 失败: %w", err)
	}

	currentReplicas := *dep.Spec.Replicas

	// 更新副本数
	dep.Spec.Replicas = &replicas

	_, err = a.client.AppsV1().Deployments(namespace).Update(ctx, dep, metav1.UpdateOptions{})
	if err != nil {
		return "", fmt.Errorf("更新 deployment 失败: %w", err)
	}

	return fmt.Sprintf("✓ 已将 Deployment %s/%s 从 %d 个副本扩缩容到 %d 个副本", namespace, action.Name, currentReplicas, replicas), nil
}

// deleteResource 删除资源
func (a *Agent) deleteResource(ctx context.Context, namespace string, action K8sAction) (string, error) {
	switch action.Resource {
	case "pod":
		err := a.client.CoreV1().Pods(namespace).Delete(ctx, action.Name, metav1.DeleteOptions{})
		if err != nil {
			return "", fmt.Errorf("删除 pod 失败: %w", err)
		}
		return fmt.Sprintf("✓ 已删除 Pod %s/%s", namespace, action.Name), nil
	case "deployment":
		err := a.client.AppsV1().Deployments(namespace).Delete(ctx, action.Name, metav1.DeleteOptions{})
		if err != nil {
			return "", fmt.Errorf("删除 deployment 失败: %w", err)
		}
		return fmt.Sprintf("✓ 已删除 Deployment %s/%s", namespace, action.Name), nil
	default:
		return "", fmt.Errorf("不支持的资源类型: %s", action.Resource)
	}
}

// cleanMarkdownJSON 清理 markdown 格式的 JSON
func cleanMarkdownJSON(text string) string {
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	return strings.TrimSpace(text)
}
