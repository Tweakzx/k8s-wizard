package tools

import (
	"context"
	"testing"
)

func TestRegisterDuplicateTool(t *testing.T) {
	registry := NewRegistry()

	tool := &mockTool{
		name: "test_tool",
	}

	// First registration should succeed
	err := registry.Register(tool)
	if err != nil {
		t.Errorf("expected first registration to succeed, got error: %v", err)
	}

	// Try to register the same tool again
	err = registry.Register(tool)
	if err == nil {
		t.Errorf("expected error on duplicate tool registration, got nil")
	}
}

func TestGetNonExistentTool(t *testing.T) {
	registry := NewRegistry()

	tool, exists := registry.Get("non_existent_tool")

	if exists {
		t.Errorf("expected tool to not exist")
	}

	if tool != nil {
		t.Errorf("expected tool to be nil")
	}
}

func TestExecuteTool(t *testing.T) {
	registry := NewRegistry()

	executor := func(ctx context.Context, args map[string]interface{}) (Result, error) {
		return Result{Success: true, Message: "executed"}, nil
	}

	tool := &operationTool{
		name:     "test_tool",
		executor: executor,
	}

	err := registry.Register(tool)
	if err != nil {
		t.Fatalf("failed to register tool: %v", err)
	}

	result, err := registry.Execute(context.Background(), "test_tool", map[string]interface{}{})

	if err != nil {
		t.Errorf("unexpected error executing tool: %v", err)
	}

	if !result.Success {
		t.Errorf("expected tool to succeed, got success=false")
	}

	if result.Message != "executed" {
		t.Errorf("unexpected tool result message, got: %s", result.Message)
	}
}

type mockTool struct {
	name string
}

func (m *mockTool) Name() string {
	return m.name
}

func (m *mockTool) Description() string {
	return "test tool description"
}

func (m *mockTool) Category() string {
	return "test"
}

func (m *mockTool) Parameters() []Parameter {
	return nil
}

func (m *mockTool) DangerLevel() DangerLevel {
	return DangerLow
}

func (m *mockTool) Execute(ctx context.Context, args map[string]interface{}) (Result, error) {
	return Result{Success: true, Message: "executed"}, nil
}
