package workflow

import (
	"context"
	"testing"

	"k8s-wizard/pkg/llm"
)

// ============================================================================
// Graph Tests (without real K8s client)
// ============================================================================

func TestNewK8sWizardGraph_NilDependencies(t *testing.T) {
	// This test verifies the graph can be created even with nil dependencies
	// (the graph itself doesn't validate dependencies, nodes do at runtime)
	deps := &Dependencies{
		LLM: &mockLLMClient{},
	}

	graph, err := NewK8sWizardGraph(deps)
	if err != nil {
		t.Fatalf("failed to create graph: %v", err)
	}

	if graph == nil {
		t.Fatal("expected graph to be created")
	}
}

func TestNewK8sWizardGraph_WithDependencies(t *testing.T) {
	deps := &Dependencies{
		LLM:       &mockLLMClient{},
		ModelName: "test-model",
	}

	graph, err := NewK8sWizardGraph(deps)
	if err != nil {
		t.Fatalf("failed to create graph: %v", err)
	}

	if graph == nil {
		t.Fatal("expected graph to be created")
	}
}

func TestNewK8sWizardGraph_InvokeChat(t *testing.T) {
	mockLLM := &mockLLMClient{
		response: `{"action":"","resource":"","name":"","namespace":"","params":{},"is_k8s_operation":false,"reply":"你好！我是 K8s Wizard。"}`,
	}

	deps := &Dependencies{
		LLM:       mockLLM,
		ModelName: "test-model",
	}

	graph, err := NewK8sWizardGraph(deps)
	if err != nil {
		t.Fatalf("failed to create graph: %v", err)
	}

	state := AgentState{
		UserMessage: "你好",
		Status:      StatusPending,
	}

	result, err := graph.Invoke(context.Background(), state)
	if err != nil {
		t.Fatalf("failed to invoke graph: %v", err)
	}

	if result.Status != StatusChat {
		t.Errorf("expected status %q, got %q", StatusChat, result.Status)
	}
}

func TestNewK8sWizardGraph_InvokeGetPods(t *testing.T) {
	mockLLM := &mockLLMClient{
		response: `{"action":"get","resource":"pod","name":"","namespace":"default","params":{},"is_k8s_operation":true,"reply":""}`,
	}

	deps := &Dependencies{
		LLM:       mockLLM,
		ModelName: "test-model",
	}

	graph, err := NewK8sWizardGraph(deps)
	if err != nil {
		t.Fatalf("failed to create graph: %v", err)
	}

	state := AgentState{
		UserMessage: "查看所有 pod",
		Status:      StatusPending,
		Confirm:     boolPtr(true), // Auto-confirm to skip preview
	}

	result, err := graph.Invoke(context.Background(), state)
	if err != nil {
		t.Fatalf("failed to invoke graph: %v", err)
	}

	// Since we don't have a real K8s client, the execute node will fail
	// But we can verify the flow reached the execute node
	if result.Action == nil {
		t.Error("expected action to be parsed")
	}
	// Execute fails with nil client, error is captured in state
	if result.Error == nil {
		t.Error("expected error when K8s client is nil")
	}
}

func TestNewK8sWizardGraph_InvokeNeedsClarification(t *testing.T) {
	mockLLM := &mockLLMClient{
		response: `{"action":"create","resource":"deployment","name":"","namespace":"default","params":{},"is_k8s_operation":true,"reply":""}`,
	}

	deps := &Dependencies{
		LLM:       mockLLM,
		ModelName: "test-model",
	}

	graph, err := NewK8sWizardGraph(deps)
	if err != nil {
		t.Fatalf("failed to create graph: %v", err)
	}

	state := AgentState{
		UserMessage: "创建一个 deployment",
		Status:      StatusPending,
	}

	result, err := graph.Invoke(context.Background(), state)
	if err != nil {
		t.Fatalf("failed to invoke graph: %v", err)
	}

	if !result.NeedsClarification {
		t.Error("expected NeedsClarification to be true")
	}
	if result.ClarificationRequest == nil {
		t.Error("expected ClarificationRequest to be set")
	}
}

func TestNewK8sWizardGraph_InvokeWithFormData(t *testing.T) {
	mockLLM := &mockLLMClient{
		response: `{"action":"create","resource":"deployment","name":"","namespace":"default","params":{},"is_k8s_operation":true,"reply":""}`,
	}

	deps := &Dependencies{
		LLM:       mockLLM,
		ModelName: "test-model",
	}

	graph, err := NewK8sWizardGraph(deps)
	if err != nil {
		t.Fatalf("failed to create graph: %v", err)
	}

	state := AgentState{
		UserMessage: "创建 deployment",
		Status:      StatusPending,
		FormData: map[string]interface{}{
			"name":      "nginx",
			"namespace": "default",
			"image":     "nginx:latest",
			"replicas":  3,
		},
		Confirm: boolPtr(true),
	}

	result, err := graph.Invoke(context.Background(), state)
	if err != nil {
		t.Fatalf("failed to invoke graph: %v", err)
	}

	// Verify form data was merged
	if result.Action == nil {
		t.Fatal("expected action to be set")
	}
	if result.Action.Name != "nginx" {
		t.Errorf("expected name 'nginx', got %q", result.Action.Name)
	}
	// Execute fails with nil client, error is captured in state
	if result.Error == nil {
		t.Error("expected error when K8s client is nil")
	}
}

// ============================================================================
// Graph With Checkpointer Tests
// ============================================================================

func TestNewK8sWizardGraphWithCheckpointer(t *testing.T) {
	// Create temp checkpointer
	checkpointer, err := NewCheckpointerManager(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create checkpointer: %v", err)
	}
	defer checkpointer.Close()

	deps := &Dependencies{
		LLM:       &mockLLMClient{},
		ModelName: "test-model",
	}

	graph, err := NewK8sWizardGraphWithCheckpointer(deps, checkpointer.GetStore())
	if err != nil {
		t.Fatalf("failed to create graph with checkpointer: %v", err)
	}

	if graph == nil {
		t.Fatal("expected graph to be created")
	}
}

func TestNewK8sWizardGraphWithCheckpointer_InvokeWithThread(t *testing.T) {
	checkpointer, err := NewCheckpointerManager(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create checkpointer: %v", err)
	}
	defer checkpointer.Close()

	mockLLM := &mockLLMClient{
		response: `{"action":"","resource":"","name":"","namespace":"","params":{},"is_k8s_operation":false,"reply":"Hello!"}`,
	}

	deps := &Dependencies{
		LLM:       mockLLM,
		ModelName: "test-model",
	}

	graph, err := NewK8sWizardGraphWithCheckpointer(deps, checkpointer.GetStore())
	if err != nil {
		t.Fatalf("failed to create graph: %v", err)
	}

	state := AgentState{
		UserMessage: "Hello",
		Status:      StatusPending,
		ThreadID:    "test-thread-123",
	}

	result, err := graph.Invoke(context.Background(), state)
	if err != nil {
		t.Fatalf("failed to invoke graph: %v", err)
	}

	if result.Status != StatusChat {
		t.Errorf("expected status %q, got %q", StatusChat, result.Status)
	}
}

// Mock LLM Client (reuse from nodes_test.go if in same package, or redefine)
type mockLLMClientForGraph struct {
	response string
	err      error
}

func (m *mockLLMClientForGraph) Chat(ctx context.Context, prompt string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func (m *mockLLMClientForGraph) GetModel() string {
	return "mock-model"
}

var _ llm.Client = (*mockLLMClientForGraph)(nil)
