package integration

import (
	"context"
	"testing"
	"time"

	"k8s-wizard/pkg/tools"
	"k8s-wizard/pkg/workflow"
)

// TestScenario1: Simple operations
func TestSimpleK8sOperations(t *testing.T) {
	// Test that tool routing works for create/get operations
	t.Skip("requires K8s cluster setup")
}

// TestScenario2: Complex workflows
func TestComplexWorkflows(t *testing.T) {
	// Test multi-step operations like create -> scale -> delete
	t.Skip("requires K8s cluster setup")
}

// TestScenario3: Context awareness
func TestContextAwareness(t *testing.T) {
	ctxMgr, err := workflow.NewContextManager(nil)
	if err != nil {
		t.Fatalf("failed to create context manager: %v", err)
	}
	threadID := "test-thread"

	// Add user message
	err = ctxMgr.AddEntry(threadID, workflow.ConversationEntry{
		Role:      "user",
		Content:   "create a deployment",
		Timestamp: time.Now(),
	})
	if err != nil {
		t.Fatalf("failed to add entry: %v", err)
	}

	// Add assistant response
	err = ctxMgr.AddEntry(threadID, workflow.ConversationEntry{
		Role:      "assistant",
		Content:   "creating deployment...",
		Timestamp: time.Now(),
	})
	if err != nil {
		t.Fatalf("failed to add entry: %v", err)
	}

	// Verify context is retrieved
	ctx := ctxMgr.Get(threadID)
	if len(ctx.History) != 2 {
		t.Errorf("expected 2 history entries, got %d", len(ctx.History))
	}

	// Verify context string
	contextStr := ctxMgr.GetContextString(threadID, 10)
	if len(contextStr) == 0 {
		t.Errorf("expected non-empty context string")
	}
}

// TestScenario4: Error handling
func TestErrorHandling(t *testing.T) {
	// Test error handling for invalid operations
	t.Skip("requires K8s cluster setup")
}

// TestScenario5: Coexistence (old and new paths)
func TestCoexistencePaths(t *testing.T) {
	// Test that old code paths work alongside new ones
	t.Skip("requires setup of both code paths")
}

// TestToolRegistryIntegration tests tool registry integration
func TestToolRegistryIntegration(t *testing.T) {
	registry := tools.NewRegistry()

	// Register a mock tool
	mock := &mockTool{name: "test_tool"}
	err := registry.Register(mock)
	if err != nil {
		t.Fatalf("failed to register tool: %v", err)
	}

	// Execute tool
	result, err := registry.Execute(context.Background(), "test_tool", map[string]interface{}{})
	if err != nil {
		t.Fatalf("failed to execute tool: %v", err)
	}

	if !result.Success {
		t.Errorf("expected tool execution to succeed")
	}

	// Verify tool can be retrieved
	tool, exists := registry.Get("test_tool")
	if !exists {
		t.Errorf("expected tool to exist")
	}
	if tool.Name() != "test_tool" {
		t.Errorf("expected tool name to be 'test_tool', got '%s'", tool.Name())
	}
}

// TestToolRegistryDuplicate tests duplicate registration error
func TestToolRegistryDuplicate(t *testing.T) {
	registry := tools.NewRegistry()

	mock := &mockTool{name: "duplicate_tool"}
	err := registry.Register(mock)
	if err != nil {
		t.Fatalf("failed to register tool: %v", err)
	}

	// Try to register same tool again
	err = registry.Register(mock)
	if err == nil {
		t.Errorf("expected error when registering duplicate tool")
	}
}

// TestToolRegistryExecuteNotFound tests executing non-existent tool
func TestToolRegistryExecuteNotFound(t *testing.T) {
	registry := tools.NewRegistry()

	// Try to execute non-existent tool
	_, err := registry.Execute(context.Background(), "non_existent_tool", map[string]interface{}{})
	if err == nil {
		t.Errorf("expected error when executing non-existent tool")
	}
}

// TestContextManagerClear tests clearing context
func TestContextManagerClear(t *testing.T) {
	ctxMgr, err := workflow.NewContextManager(nil)
	if err != nil {
		t.Fatalf("failed to create context manager: %v", err)
	}
	threadID := "test-clear-thread"

	// Add some entries
	err = ctxMgr.AddEntry(threadID, workflow.ConversationEntry{
		Role:      "user",
		Content:   "test message",
		Timestamp: time.Now(),
	})
	if err != nil {
		t.Fatalf("failed to add entry: %v", err)
	}

	// Verify context exists
	if !ctxMgr.HasContext(threadID) {
		t.Errorf("expected context to exist before clear")
	}

	// Clear context
	err = ctxMgr.Clear(threadID)
	if err != nil {
		t.Fatalf("failed to clear context: %v", err)
	}

	// Verify context is cleared
	if ctxMgr.HasContext(threadID) {
		t.Errorf("expected context to be cleared")
	}
}

// TestContextManagerInvalidEntry tests validation of invalid entries
func TestContextManagerInvalidEntry(t *testing.T) {
	ctxMgr, err := workflow.NewContextManager(nil)
	if err != nil {
		t.Fatalf("failed to create context manager: %v", err)
	}

	// Test empty threadID
	err = ctxMgr.AddEntry("", workflow.ConversationEntry{
		Role:      "user",
		Content:   "test",
		Timestamp: time.Now(),
	})
	if err == nil {
		t.Errorf("expected error for empty threadID")
	}

	// Test invalid role
	err = ctxMgr.AddEntry("test-thread", workflow.ConversationEntry{
		Role:      "invalid",
		Content:   "test",
		Timestamp: time.Now(),
	})
	if err == nil {
		t.Errorf("expected error for invalid role")
	}
}

// TestContextManagerContextString tests context string formatting
func TestContextManagerContextString(t *testing.T) {
	ctxMgr, err := workflow.NewContextManager(nil)
	if err != nil {
		t.Fatalf("failed to create context manager: %v", err)
	}
	threadID := "test-context-string"

	// Add multiple entries
	entries := []workflow.ConversationEntry{
		{Role: "user", Content: "create deployment", Timestamp: time.Now()},
		{Role: "assistant", Content: "creating deployment...", Timestamp: time.Now()},
		{Role: "user", Content: "scale to 3 replicas", Timestamp: time.Now()},
	}

	for _, entry := range entries {
		err = ctxMgr.AddEntry(threadID, entry)
		if err != nil {
			t.Fatalf("failed to add entry: %v", err)
		}
	}

	// Get full context
	fullCtx := ctxMgr.GetContextString(threadID, 10)
	if len(fullCtx) == 0 {
		t.Errorf("expected non-empty full context string")
	}

	// Get limited context
	limitedCtx := ctxMgr.GetContextString(threadID, 2)
	if len(limitedCtx) == 0 {
		t.Errorf("expected non-empty limited context string")
	}

	// Verify limited context is shorter
	if len(limitedCtx) >= len(fullCtx) {
		t.Errorf("expected limited context to be shorter than full context")
	}
}

// mockTool is a mock tool implementation for testing
type mockTool struct {
	name string
}

func (m *mockTool) Name() string { return m.name }

func (m *mockTool) Description() string { return "test tool" }

func (m *mockTool) Category() string { return "test" }

func (m *mockTool) Parameters() []tools.Parameter { return nil }

func (m *mockTool) DangerLevel() tools.DangerLevel { return tools.DangerLow }

func (m *mockTool) Execute(ctx context.Context, args map[string]interface{}) (tools.Result, error) {
	return tools.Result{Success: true, Message: "executed"}, nil
}
