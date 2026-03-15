package agent

import (
	"context"
	"errors"
	"testing"

	"k8s-wizard/api/models"
	"k8s-wizard/pkg/llm"
	"k8s-wizard/pkg/workflow"
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
// GraphAgent Tests
// ============================================================================

func TestNewGraphAgent(t *testing.T) {
	// Note: This test creates an agent without K8s client
	// The agent can be created, but operations that need K8s will fail
	agent, err := NewGraphAgent(nil, &mockLLMClient{}, "test-model")
	if err != nil {
		t.Fatalf("failed to create GraphAgent: %v", err)
	}

	if agent == nil {
		t.Fatal("expected agent to be created")
	}

	// Verify ToolRegistry is initialized
	if agent.deps == nil {
		t.Fatal("expected agent dependencies to be initialized")
	}
	if agent.deps.ToolRegistry == nil {
		t.Error("expected ToolRegistry to be initialized in agent dependencies")
	}
}

func TestGraphAgent_GetModelName(t *testing.T) {
	agent, err := NewGraphAgent(nil, &mockLLMClient{}, "gpt-4")
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	if agent.GetModelName() != "gpt-4" {
		t.Errorf("GetModelName() = %q, want %q", agent.GetModelName(), "gpt-4")
	}
}

func TestGraphAgent_GetModel(t *testing.T) {
	agent, err := NewGraphAgent(nil, &mockLLMClient{}, "claude-3")
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	if agent.GetModel() != "claude-3" {
		t.Errorf("GetModel() = %q, want %q", agent.GetModel(), "claude-3")
	}
}

func TestGraphAgent_SetModel(t *testing.T) {
	agent, err := NewGraphAgent(nil, &mockLLMClient{}, "initial-model")
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	err = agent.SetModel("new-model")
	if err != nil {
		t.Fatalf("SetModel() error = %v", err)
	}

	if agent.GetModelName() != "new-model" {
		t.Errorf("GetModelName() = %q, want %q", agent.GetModelName(), "new-model")
	}
}

func TestGraphAgent_ProcessCommand(t *testing.T) {
	mockLLM := &mockLLMClient{
		response: `{"action":"","resource":"","name":"","namespace":"","params":{},"is_k8s_operation":false,"reply":"你好！我是 K8s Wizard。"}`,
	}

	agent, err := NewGraphAgent(nil, mockLLM, "test-model")
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	result, err := agent.ProcessCommand(context.Background(), "你好")
	if err != nil {
		t.Fatalf("ProcessCommand() error = %v", err)
	}

	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestGraphAgent_ProcessCommandWithClarification_Chat(t *testing.T) {
	mockLLM := &mockLLMClient{
		response: `{"action":"","resource":"","name":"","namespace":"","params":{},"is_k8s_operation":false,"reply":"你好！我是 K8s Wizard。"}`,
	}

	agent, err := NewGraphAgent(nil, mockLLM, "test-model")
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	result, clarification, preview, err := agent.ProcessCommandWithClarification(
		context.Background(),
		"你好",
		nil,
		nil,
	)

	if err != nil {
		t.Fatalf("ProcessCommandWithClarification() error = %v", err)
	}

	if result == "" {
		t.Error("expected non-empty result for chat")
	}
	if clarification != nil {
		t.Error("expected no clarification for chat")
	}
	if preview != nil {
		t.Error("expected no preview for chat")
	}
}

func TestGraphAgent_ProcessCommandWithClarification_NeedsClarification(t *testing.T) {
	mockLLM := &mockLLMClient{
		response: `{"action":"create","resource":"deployment","name":"","namespace":"default","params":{},"is_k8s_operation":true,"reply":""}`,
	}

	agent, err := NewGraphAgent(nil, mockLLM, "test-model")
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	result, clarification, preview, err := agent.ProcessCommandWithClarification(
		context.Background(),
		"创建一个 deployment",
		nil,
		nil,
	)

	if err != nil {
		t.Fatalf("ProcessCommandWithClarification() error = %v", err)
	}

	// Should return clarification request for missing fields
	if clarification == nil {
		t.Error("expected clarification request for missing fields")
	}
	if preview != nil {
		t.Error("expected no preview before clarification")
	}
	_ = result // Result may be empty when clarification is needed
}

func TestGraphAgent_ProcessCommandWithClarification_NeedsConfirm(t *testing.T) {
	mockLLM := &mockLLMClient{
		response: `{"action":"create","resource":"deployment","name":"","namespace":"default","params":{},"is_k8s_operation":true,"reply":""}`,
	}

	agent, err := NewGraphAgent(nil, mockLLM, "test-model")
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	// Provide all required data
	formData := map[string]interface{}{
		"name":      "nginx",
		"namespace": "default",
		"image":     "nginx:latest",
		"replicas":  3,
	}

	result, clarification, preview, err := agent.ProcessCommandWithClarification(
		context.Background(),
		"创建 deployment",
		formData,
		nil, // No confirm yet
	)

	if err != nil {
		t.Fatalf("ProcessCommandWithClarification() error = %v", err)
	}

	// Should return preview for confirmation
	if preview == nil {
		t.Error("expected preview for confirmation")
	}
	if clarification != nil {
		t.Error("expected no clarification when all data provided")
	}
	_ = result
}

func TestGraphAgent_GetGraph(t *testing.T) {
	agent, err := NewGraphAgent(nil, &mockLLMClient{}, "test-model")
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	graph := agent.GetGraph()
	if graph == nil {
		t.Error("expected graph to be returned")
	}
}

// ============================================================================
// GraphAgentWithCheckpointer Tests
// ============================================================================

func TestNewGraphAgentWithCheckpointer(t *testing.T) {
	checkpointer, err := workflow.NewCheckpointerManager(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create checkpointer: %v", err)
	}
	defer checkpointer.Close()

	agent, err := NewGraphAgentWithCheckpointer(nil, &mockLLMClient{}, "test-model", checkpointer)
	if err != nil {
		t.Fatalf("failed to create GraphAgentWithCheckpointer: %v", err)
	}

	if agent == nil {
		t.Fatal("expected agent to be created")
	}

	// Verify ToolRegistry is initialized
	if agent.deps == nil {
		t.Fatal("expected agent dependencies to be initialized")
	}
	if agent.deps.ToolRegistry == nil {
		t.Error("expected ToolRegistry to be initialized in agent dependencies")
	}
}

func TestGraphAgentWithCheckpointer_ProcessCommand(t *testing.T) {
	checkpointer, err := workflow.NewCheckpointerManager(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create checkpointer: %v", err)
	}
	defer checkpointer.Close()

	mockLLM := &mockLLMClient{
		response: `{"action":"","resource":"","name":"","namespace":"","params":{},"is_k8s_operation":false,"reply":"Hello!"}`,
	}

	agent, err := NewGraphAgentWithCheckpointer(nil, mockLLM, "test-model", checkpointer)
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	result, err := agent.ProcessCommand(context.Background(), "Hello")
	if err != nil {
		t.Fatalf("ProcessCommand() error = %v", err)
	}

	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestGraphAgentWithCheckpointer_ProcessCommandWithClarificationAndThread(t *testing.T) {
	checkpointer, err := workflow.NewCheckpointerManager(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create checkpointer: %v", err)
	}
	defer checkpointer.Close()

	mockLLM := &mockLLMClient{
		response: `{"action":"","resource":"","name":"","namespace":"","params":{},"is_k8s_operation":false,"reply":"Hello!"}`,
	}

	agent, err := NewGraphAgentWithCheckpointer(nil, mockLLM, "test-model", checkpointer)
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	result, clarification, preview, err := agent.ProcessCommandWithClarificationAndThread(
		context.Background(),
		"Hello",
		nil,
		nil,
		"test-thread-123",
	)

	if err != nil {
		t.Fatalf("ProcessCommandWithClarificationAndThread() error = %v", err)
	}

	if result == "" {
		t.Error("expected non-empty result")
	}
	if clarification != nil {
		t.Error("expected no clarification")
	}
	if preview != nil {
		t.Error("expected no preview")
	}
}

func TestGraphAgentWithCheckpointer_ClearSession(t *testing.T) {
	checkpointer, err := workflow.NewCheckpointerManager(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create checkpointer: %v", err)
	}
	defer checkpointer.Close()

	mockLLM := &mockLLMClient{
		response: `{"action":"","resource":"","name":"","namespace":"","params":{},"is_k8s_operation":false,"reply":"Hello!"}`,
	}

	agent, err := NewGraphAgentWithCheckpointer(nil, mockLLM, "test-model", checkpointer)
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	// First, make a request with a thread
	_, _, _, err = agent.ProcessCommandWithClarificationAndThread(
		context.Background(),
		"Hello",
		nil,
		nil,
		"thread-to-clear",
	)
	if err != nil {
		t.Fatalf("ProcessCommand() error = %v", err)
	}

	// Clear the session
	err = agent.ClearSession(context.Background(), "thread-to-clear")
	if err != nil {
		t.Fatalf("ClearSession() error = %v", err)
	}
}

func TestGraphAgentWithCheckpointer_Close(t *testing.T) {
	checkpointer, err := workflow.NewCheckpointerManager(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create checkpointer: %v", err)
	}

	agent, err := NewGraphAgentWithCheckpointer(nil, &mockLLMClient{}, "test-model", checkpointer)
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	err = agent.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestGraphAgentWithCheckpointer_GetModel(t *testing.T) {
	checkpointer, err := workflow.NewCheckpointerManager(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create checkpointer: %v", err)
	}
	defer checkpointer.Close()

	agent, err := NewGraphAgentWithCheckpointer(nil, &mockLLMClient{}, "gpt-4", checkpointer)
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	if agent.GetModel() != "gpt-4" {
		t.Errorf("GetModel() = %q, want %q", agent.GetModel(), "gpt-4")
	}
}

func TestGraphAgentWithCheckpointer_SetModel(t *testing.T) {
	checkpointer, err := workflow.NewCheckpointerManager(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create checkpointer: %v", err)
	}
	defer checkpointer.Close()

	agent, err := NewGraphAgentWithCheckpointer(nil, &mockLLMClient{}, "initial", checkpointer)
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	err = agent.SetModel("new-model")
	if err != nil {
		t.Fatalf("SetModel() error = %v", err)
	}

	if agent.GetModelName() != "new-model" {
		t.Errorf("GetModelName() = %q, want %q", agent.GetModelName(), "new-model")
	}
}

func TestGraphAgentWithCheckpointer_GetGraph(t *testing.T) {
	checkpointer, err := workflow.NewCheckpointerManager(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create checkpointer: %v", err)
	}
	defer checkpointer.Close()

	agent, err := NewGraphAgentWithCheckpointer(nil, &mockLLMClient{}, "test-model", checkpointer)
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	graph := agent.GetGraph()
	if graph == nil {
		t.Error("expected graph to be returned")
	}
}

// ============================================================================
// Interface Compliance Tests
// ============================================================================

func TestGraphAgent_ImplementsInterface(t *testing.T) {
	agent, err := NewGraphAgent(nil, &mockLLMClient{}, "test-model")
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	// Compile-time check
	var _ AgentInterface = agent
}

func TestGraphAgentWithCheckpointer_ImplementsInterface(t *testing.T) {
	checkpointer, err := workflow.NewCheckpointerManager(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create checkpointer: %v", err)
	}
	defer checkpointer.Close()

	agent, err := NewGraphAgentWithCheckpointer(nil, &mockLLMClient{}, "test-model", checkpointer)
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	// Compile-time check
	var _ AgentInterface = agent
}

// ============================================================================
// Edge Cases
// ============================================================================

func TestGraphAgent_ProcessCommand_LLMError(t *testing.T) {
	mockLLM := &mockLLMClient{
		err: errors.New("LLM API error"),
	}

	agent, err := NewGraphAgent(nil, mockLLM, "test-model")
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	// LLM error is captured in state and returned as error
	_, err = agent.ProcessCommand(context.Background(), "test")
	if err == nil {
		t.Error("expected error when LLM fails")
	}
}

func TestGraphAgent_ProcessCommandWithClarification_ExecuteWithoutK8sClient(t *testing.T) {
	mockLLM := &mockLLMClient{
		response: `{"action":"get","resource":"pod","name":"","namespace":"default","params":{},"is_k8s_operation":true,"reply":""}`,
	}

	agent, err := NewGraphAgent(nil, mockLLM, "test-model")
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	// Provide confirmation to trigger execute
	confirm := true
	result, clarification, preview, err := agent.ProcessCommandWithClarification(
		context.Background(),
		"查看 pod",
		nil,
		&confirm,
	)

	// Since we don't have K8s client, execute will fail
	// The error is captured in state and returned
	if err == nil {
		t.Error("expected error when K8s client is nil")
	}
	_ = result
	_ = clarification
	_ = preview
}

// Helper function
func boolPtr(b bool) *bool {
	return &b
}

// Ensure mock ClarificationRequest is used correctly
var _ = &models.ClarificationRequest{}
