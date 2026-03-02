package workflow

import (
	"context"
	"errors"
	"testing"

	"k8s-wizard/pkg/llm"
)

// ============================================================================
// Mock LLM Client
// ============================================================================

type mockLLMClient struct {
	response string
	err      error
}

func (m *mockLLMClient) Chat(ctx context.Context, prompt string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func (m *mockLLMClient) GetModel() string {
	return "mock-model"
}

var _ llm.Client = (*mockLLMClient)(nil)

// ============================================================================
// Parse Intent Node Tests
// ============================================================================

func TestMakeParseIntentNode_AlreadyParsed(t *testing.T) {
	mockLLM := &mockLLMClient{}
	node := MakeParseIntentNode(mockLLM)

	state := AgentState{
		Action: &K8sAction{Action: "create"},
	}

	result, err := node(context.Background(), state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return same state without calling LLM
	if result.Action == nil || result.Action.Action != "create" {
		t.Error("expected action to be preserved")
	}
}

func TestMakeParseIntentNode_K8sOperation(t *testing.T) {
	mockLLM := &mockLLMClient{
		response: `{"action":"get","resource":"pod","name":"","namespace":"default","params":{},"is_k8s_operation":true,"reply":""}`,
	}
	node := MakeParseIntentNode(mockLLM)

	state := AgentState{
		UserMessage: "查看所有 pod",
		Status:      StatusPending,
	}

	result, err := node(context.Background(), state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Action == nil {
		t.Fatal("expected action to be parsed")
	}
	if result.Action.Action != "get" {
		t.Errorf("expected action 'get', got %q", result.Action.Action)
	}
	if result.Action.Resource != "pod" {
		t.Errorf("expected resource 'pod', got %q", result.Action.Resource)
	}
	if !result.IsK8sOperation {
		t.Error("expected IsK8sOperation to be true")
	}
}

func TestMakeParseIntentNode_ChatResponse(t *testing.T) {
	mockLLM := &mockLLMClient{
		response: `{"action":"","resource":"","name":"","namespace":"","params":{},"is_k8s_operation":false,"reply":"你好！我是 K8s Wizard。"}`,
	}
	node := MakeParseIntentNode(mockLLM)

	state := AgentState{
		UserMessage: "你好",
		Status:      StatusPending,
	}

	result, err := node(context.Background(), state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != StatusChat {
		t.Errorf("expected status %q, got %q", StatusChat, result.Status)
	}
	if result.Reply != "你好！我是 K8s Wizard。" {
		t.Errorf("expected reply, got %q", result.Reply)
	}
}

func TestMakeParseIntentNode_LLMMarkdownJSON(t *testing.T) {
	mockLLM := &mockLLMClient{
		response: "```json\n{\"action\":\"list\",\"resource\":\"deployment\",\"name\":\"\",\"namespace\":\"\",\"params\":{},\"is_k8s_operation\":true,\"reply\":\"\"}\n```",
	}
	node := MakeParseIntentNode(mockLLM)

	state := AgentState{
		UserMessage: "查看 deployment",
		Status:      StatusPending,
	}

	result, err := node(context.Background(), state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Action == nil {
		t.Fatal("expected action to be parsed from markdown")
	}
	if result.Action.Resource != "deployment" {
		t.Errorf("expected resource 'deployment', got %q", result.Action.Resource)
	}
}

func TestMakeParseIntentNode_LLMError(t *testing.T) {
	mockLLM := &mockLLMClient{
		err: errors.New("LLM API error"),
	}
	node := MakeParseIntentNode(mockLLM)

	state := AgentState{
		UserMessage: "查看 pod",
		Status:      StatusPending,
	}

	result, err := node(context.Background(), state)
	if err != nil {
		t.Fatalf("node should not return error, got: %v", err)
	}

	if result.Status != StatusError {
		t.Errorf("expected status %q, got %q", StatusError, result.Status)
	}
	if result.Error == nil {
		t.Error("expected error to be set in state")
	}
}

func TestMakeParseIntentNode_InvalidJSON(t *testing.T) {
	mockLLM := &mockLLMClient{
		response: "this is not valid json",
	}
	node := MakeParseIntentNode(mockLLM)

	state := AgentState{
		UserMessage: "查看 pod",
		Status:      StatusPending,
	}

	result, err := node(context.Background(), state)
	if err != nil {
		t.Fatalf("node should not return error, got: %v", err)
	}

	if result.Status != StatusError {
		t.Errorf("expected status %q, got %q", StatusError, result.Status)
	}
}

// ============================================================================
// Merge Form Node Tests
// ============================================================================

func TestMakeMergeFormNode_NoAction(t *testing.T) {
	node := MakeMergeFormNode()

	state := AgentState{
		FormData: map[string]interface{}{"name": "nginx"},
	}

	result, err := node(context.Background(), state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should pass through without error
	if result.FormData["name"] != "nginx" {
		t.Error("expected form data to be preserved")
	}
}

func TestMakeMergeFormNode_NoFormData(t *testing.T) {
	node := MakeMergeFormNode()

	state := AgentState{
		Action: &K8sAction{Action: "create"},
	}

	_, err := node(context.Background(), state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should pass through without error
}

func TestMakeMergeFormNode_MergeName(t *testing.T) {
	node := MakeMergeFormNode()

	state := AgentState{
		Action: &K8sAction{
			Action: "create",
			Params: map[string]interface{}{},
		},
		FormData: map[string]interface{}{
			"name": "my-app",
		},
	}

	result, err := node(context.Background(), state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Action.Name != "my-app" {
		t.Errorf("expected name 'my-app', got %q", result.Action.Name)
	}
}

func TestMakeMergeFormNode_MergeNamespace(t *testing.T) {
	node := MakeMergeFormNode()

	state := AgentState{
		Action: &K8sAction{
			Action: "create",
			Params: map[string]interface{}{},
		},
		FormData: map[string]interface{}{
			"namespace": "production",
		},
	}

	result, err := node(context.Background(), state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Action.Namespace != "production" {
		t.Errorf("expected namespace 'production', got %q", result.Action.Namespace)
	}
}

func TestMakeMergeFormNode_MergeParams(t *testing.T) {
	node := MakeMergeFormNode()

	state := AgentState{
		Action: &K8sAction{
			Action: "create",
			Params: map[string]interface{}{},
		},
		FormData: map[string]interface{}{
			"image":    "nginx:latest",
			"replicas": 3,
		},
	}

	result, err := node(context.Background(), state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Action.Params["image"] != "nginx:latest" {
		t.Errorf("expected image 'nginx:latest', got %v", result.Action.Params["image"])
	}
	if result.Action.Params["replicas"] != 3 {
		t.Errorf("expected replicas 3, got %v", result.Action.Params["replicas"])
	}
}

// ============================================================================
// Check Clarify Node Tests
// ============================================================================

func TestMakeCheckClarifyNode_NoAction(t *testing.T) {
	node := MakeCheckClarifyNode()

	state := AgentState{}

	_, err := node(context.Background(), state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should pass through without panic
}

func TestMakeCheckClarifyNode_CreateNeedsInfo(t *testing.T) {
	node := MakeCheckClarifyNode()

	state := AgentState{
		Action: &K8sAction{
			Action:   "create",
			Resource: "deployment",
			Name:     "", // Missing name
		},
	}

	result, err := node(context.Background(), state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.NeedsClarification {
		t.Error("expected NeedsClarification to be true")
	}
	if result.ClarificationRequest == nil {
		t.Error("expected ClarificationRequest to be set")
	}
}

func TestMakeCheckClarifyNode_CreateHasAllInfo(t *testing.T) {
	node := MakeCheckClarifyNode()

	state := AgentState{
		Action: &K8sAction{
			Action:   "create",
			Resource: "deployment",
			Name:     "nginx",
			Params: map[string]interface{}{
				"image": "nginx:latest",
			},
		},
	}

	result, err := node(context.Background(), state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.NeedsClarification {
		t.Error("expected NeedsClarification to be false")
	}
}

func TestMakeCheckClarifyNode_ScaleNeedsInfo(t *testing.T) {
	node := MakeCheckClarifyNode()

	state := AgentState{
		Action: &K8sAction{
			Action:   "scale",
			Resource: "deployment",
			Name:     "", // Missing name
		},
	}

	result, err := node(context.Background(), state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.NeedsClarification {
		t.Error("expected NeedsClarification to be true for scale without name")
	}
	if result.ClarificationRequest == nil {
		t.Error("expected ClarificationRequest to be set")
	}
	if result.ClarificationRequest.Action != "scale" {
		t.Errorf("expected clarification action 'scale', got %q", result.ClarificationRequest.Action)
	}
}

func TestMakeCheckClarifyNode_DeleteNeedsInfo(t *testing.T) {
	node := MakeCheckClarifyNode()

	state := AgentState{
		Action: &K8sAction{
			Action:   "delete",
			Resource: "deployment",
			Name:     "", // Missing name
		},
	}

	result, err := node(context.Background(), state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.NeedsClarification {
		t.Error("expected NeedsClarification to be true for delete without name")
	}
}

func TestMakeCheckClarifyNode_GetActionNoClarification(t *testing.T) {
	node := MakeCheckClarifyNode()

	state := AgentState{
		Action: &K8sAction{
			Action:   "get",
			Resource: "pod",
		},
	}

	result, err := node(context.Background(), state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.NeedsClarification {
		t.Error("expected NeedsClarification to be false for get action")
	}
}

// ============================================================================
// Generate Preview Node Tests
// ============================================================================

func TestMakeGeneratePreviewNode_NoAction(t *testing.T) {
	node := MakeGeneratePreviewNode()

	state := AgentState{}

	result, err := node(context.Background(), state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ActionPreview != nil {
		t.Error("expected no ActionPreview when no action")
	}
}

func TestMakeGeneratePreviewNode_CreatePreview(t *testing.T) {
	node := MakeGeneratePreviewNode()

	state := AgentState{
		Action: &K8sAction{
			Action:    "create",
			Resource:  "deployment",
			Name:      "nginx",
			Namespace: "default",
			Params: map[string]interface{}{
				"image":    "nginx:latest",
				"replicas": 3,
			},
		},
	}

	result, err := node(context.Background(), state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ActionPreview == nil {
		t.Fatal("expected ActionPreview to be set")
	}
	if result.ActionPreview.Type != "create" {
		t.Errorf("expected type 'create', got %q", result.ActionPreview.Type)
	}
	if result.ActionPreview.DangerLevel != "low" {
		t.Errorf("expected danger level 'low', got %q", result.ActionPreview.DangerLevel)
	}
	if result.ActionPreview.YAML == "" {
		t.Error("expected YAML to be generated")
	}
}

func TestMakeGeneratePreviewNode_ScalePreview(t *testing.T) {
	node := MakeGeneratePreviewNode()

	state := AgentState{
		Action: &K8sAction{
			Action:    "scale",
			Resource:  "deployment",
			Name:      "web",
			Namespace: "default",
			Params: map[string]interface{}{
				"replicas": 5,
			},
		},
	}

	result, err := node(context.Background(), state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ActionPreview == nil {
		t.Fatal("expected ActionPreview to be set")
	}
	if result.ActionPreview.Type != "scale" {
		t.Errorf("expected type 'scale', got %q", result.ActionPreview.Type)
	}
	if result.ActionPreview.DangerLevel != "medium" {
		t.Errorf("expected danger level 'medium', got %q", result.ActionPreview.DangerLevel)
	}
}

func TestMakeGeneratePreviewNode_DeletePreview(t *testing.T) {
	node := MakeGeneratePreviewNode()

	state := AgentState{
		Action: &K8sAction{
			Action:    "delete",
			Resource:  "deployment",
			Name:      "nginx",
			Namespace: "default",
		},
	}

	result, err := node(context.Background(), state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ActionPreview == nil {
		t.Fatal("expected ActionPreview to be set")
	}
	if result.ActionPreview.Type != "delete" {
		t.Errorf("expected type 'delete', got %q", result.ActionPreview.Type)
	}
	if result.ActionPreview.DangerLevel != "high" {
		t.Errorf("expected danger level 'high', got %q", result.ActionPreview.DangerLevel)
	}
}

func TestMakeGeneratePreviewNode_GetPreview(t *testing.T) {
	node := MakeGeneratePreviewNode()

	state := AgentState{
		Action: &K8sAction{
			Action:    "get",
			Resource:  "pod",
			Namespace: "kube-system",
		},
	}

	result, err := node(context.Background(), state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ActionPreview == nil {
		t.Fatal("expected ActionPreview to be set")
	}
	if result.ActionPreview.Type != "get" {
		t.Errorf("expected type 'get', got %q", result.ActionPreview.Type)
	}
	if result.ActionPreview.Namespace != "kube-system" {
		t.Errorf("expected namespace 'kube-system', got %q", result.ActionPreview.Namespace)
	}
}

// ============================================================================
// Helper Functions Tests
// ============================================================================

func TestGetReplicas(t *testing.T) {
	tests := []struct {
		name     string
		params   map[string]interface{}
		expected int
	}{
		{"int", map[string]interface{}{"replicas": 3}, 3},
		{"int32", map[string]interface{}{"replicas": int32(5)}, 5},
		{"int64", map[string]interface{}{"replicas": int64(7)}, 7},
		{"float32", map[string]interface{}{"replicas": float32(2)}, 2},
		{"float64", map[string]interface{}{"replicas": float64(4)}, 4},
		{"missing", map[string]interface{}{}, 1},
		{"nil", nil, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getReplicas(tt.params)
			if result != tt.expected {
				t.Errorf("getReplicas() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestCleanMarkdownJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain json",
			input:    `{"key":"value"}`,
			expected: `{"key":"value"}`,
		},
		{
			name:     "json with code fence",
			input:    "```json\n{\"key\":\"value\"}\n```",
			expected: `{"key":"value"}`,
		},
		{
			name:     "json with plain code fence",
			input:    "```\n{\"key\":\"value\"}\n```",
			expected: `{"key":"value"}`,
		},
		{
			name:     "json with whitespace",
			input:    "  {\"key\":\"value\"}  ",
			expected: `{"key":"value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanMarkdownJSON(tt.input)
			if result != tt.expected {
				t.Errorf("cleanMarkdownJSON() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestMergeFormData(t *testing.T) {
	tests := []struct {
		name         string
		action       *K8sAction
		formData     map[string]interface{}
		expectedName string
		expectedNs   string
		expectedImg  interface{}
	}{
		{
			name: "merge all fields",
			action: &K8sAction{
				Params: map[string]interface{}{},
			},
			formData: map[string]interface{}{
				"name":      "my-app",
				"namespace": "prod",
				"image":     "nginx:latest",
			},
			expectedName: "my-app",
			expectedNs:   "prod",
			expectedImg:  "nginx:latest",
		},
		{
			name: "preserve existing params",
			action: &K8sAction{
				Params: map[string]interface{}{
					"replicas": 3,
				},
			},
			formData: map[string]interface{}{
				"name":  "my-app",
				"ports": 8080,
			},
			expectedName: "my-app",
			expectedImg:  nil,
		},
		{
			name: "nil params becomes map",
			action: &K8sAction{
				Params: nil,
			},
			formData: map[string]interface{}{
				"name": "test",
			},
			expectedName: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mergeFormData(tt.action, tt.formData)

			if tt.action.Name != tt.expectedName {
				t.Errorf("name = %q, want %q", tt.action.Name, tt.expectedName)
			}
			if tt.expectedNs != "" && tt.action.Namespace != tt.expectedNs {
				t.Errorf("namespace = %q, want %q", tt.action.Namespace, tt.expectedNs)
			}
			if tt.expectedImg != nil && tt.action.Params["image"] != tt.expectedImg {
				t.Errorf("image = %v, want %v", tt.action.Params["image"], tt.expectedImg)
			}
		})
	}
}

// ============================================================================
// Check Clarification Helper Tests
// ============================================================================

func TestCheckNeedsClarification(t *testing.T) {
	tests := []struct {
		name          string
		action        *K8sAction
		needsInfo     bool
		expectedTitle string
	}{
		{
			name: "create missing name",
			action: &K8sAction{
				Action: "create",
				Name:   "",
			},
			needsInfo:     true,
			expectedTitle: "📦 创建 Deployment",
		},
		{
			name: "scale missing name",
			action: &K8sAction{
				Action: "scale",
				Name:   "",
			},
			needsInfo:     true,
			expectedTitle: "⚖️ 扩缩容 Deployment",
		},
		{
			name: "delete missing name",
			action: &K8sAction{
				Action: "delete",
				Name:   "",
			},
			needsInfo:     true,
			expectedTitle: "🗑️ 删除资源",
		},
		{
			name: "get action no clarification",
			action: &K8sAction{
				Action: "get",
			},
			needsInfo: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clarReq, needsInfo := checkNeedsClarification(tt.action)

			if needsInfo != tt.needsInfo {
				t.Errorf("needsInfo = %v, want %v", needsInfo, tt.needsInfo)
			}

			if tt.needsInfo {
				if clarReq == nil {
					t.Fatal("expected ClarificationRequest when needsInfo is true")
				}
				if clarReq.Title != tt.expectedTitle {
					t.Errorf("title = %q, want %q", clarReq.Title, tt.expectedTitle)
				}
			}
		})
	}
}

func TestCheckCreateClarification(t *testing.T) {
	tests := []struct {
		name       string
		action     *K8sAction
		needsInfo  bool
		fieldCount int
	}{
		{
			name: "missing name and image",
			action: &K8sAction{
				Action: "create",
				Name:   "",
				Params: map[string]interface{}{},
			},
			needsInfo:  true,
			fieldCount: 4, // name, image, replicas, namespace
		},
		{
			name: "has name missing image",
			action: &K8sAction{
				Action: "create",
				Name:   "nginx",
				Params: map[string]interface{}{},
			},
			needsInfo:  true,
			fieldCount: 4,
		},
		{
			name: "has all info",
			action: &K8sAction{
				Action: "create",
				Name:   "nginx",
				Params: map[string]interface{}{
					"image": "nginx:latest",
				},
			},
			needsInfo: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clarReq, needsInfo := checkCreateClarification(tt.action)

			if needsInfo != tt.needsInfo {
				t.Errorf("needsInfo = %v, want %v", needsInfo, tt.needsInfo)
			}

			if tt.needsInfo && len(clarReq.Fields) != tt.fieldCount {
				t.Errorf("field count = %d, want %d", len(clarReq.Fields), tt.fieldCount)
			}
		})
	}
}

// ============================================================================
// Action Preview Helper Tests
// ============================================================================

func TestGenerateActionPreview(t *testing.T) {
	tests := []struct {
		name           string
		action         *K8sAction
		expectedType   string
		expectedDanger string
		shouldExist    bool
	}{
		{
			name: "create deployment",
			action: &K8sAction{
				Action:    "create",
				Resource:  "deployment",
				Name:      "nginx",
				Namespace: "default",
				Params:    map[string]interface{}{"image": "nginx:latest", "replicas": 3},
			},
			expectedType:   "create",
			expectedDanger: "low",
			shouldExist:    true,
		},
		{
			name: "scale deployment",
			action: &K8sAction{
				Action:    "scale",
				Resource:  "deployment",
				Name:      "web",
				Namespace: "default",
				Params:    map[string]interface{}{"replicas": 5},
			},
			expectedType:   "scale",
			expectedDanger: "medium",
			shouldExist:    true,
		},
		{
			name: "delete deployment",
			action: &K8sAction{
				Action:    "delete",
				Resource:  "deployment",
				Name:      "nginx",
				Namespace: "default",
			},
			expectedType:   "delete",
			expectedDanger: "high",
			shouldExist:    true,
		},
		{
			name: "get pods",
			action: &K8sAction{
				Action:    "get",
				Resource:  "pod",
				Namespace: "default",
			},
			expectedType:   "get",
			expectedDanger: "low",
			shouldExist:    true,
		},
		{
			name: "invalid resource type",
			action: &K8sAction{
				Action:   "get",
				Resource: "invalid_resource",
			},
			shouldExist: false,
		},
		{
			name:        "nil action",
			action:      nil,
			shouldExist: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			preview := generateActionPreview(tt.action)

			if !tt.shouldExist {
				if preview != nil {
					t.Error("expected nil preview")
				}
				return
			}

			if preview == nil {
				t.Fatal("expected preview to exist")
			}

			if preview.Type != tt.expectedType {
				t.Errorf("type = %q, want %q", preview.Type, tt.expectedType)
			}
			if preview.DangerLevel != tt.expectedDanger {
				t.Errorf("danger level = %q, want %q", preview.DangerLevel, tt.expectedDanger)
			}
		})
	}
}

func TestGenerateDeploymentYAML(t *testing.T) {
	yaml := generateDeploymentYAML("nginx", "default", "nginx:latest", 3)

	if yaml == "" {
		t.Fatal("expected YAML to be generated")
	}

	// Check key elements
	if !contains(yaml, "apiVersion: apps/v1") {
		t.Error("YAML missing apiVersion")
	}
	if !contains(yaml, "kind: Deployment") {
		t.Error("YAML missing kind")
	}
	if !contains(yaml, "name: nginx") {
		t.Error("YAML missing name")
	}
	if !contains(yaml, "namespace: default") {
		t.Error("YAML missing namespace")
	}
	if !contains(yaml, "replicas: 3") {
		t.Error("YAML missing replicas")
	}
	if !contains(yaml, "image: nginx:latest") {
		t.Error("YAML missing image")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
