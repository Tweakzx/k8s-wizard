package handlers

import (
	"context"
	"testing"

	"k8s-wizard/pkg/tools"
	"k8s.io/client-go/kubernetes/fake"
)

func TestNewBaseHandler(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	handler := NewBaseHandler(clientset, "test-resource")
	if handler.Resource() != "test-resource" {
		t.Errorf("expected resource to be 'test-resource', got %s", handler.Resource())
	}
}

func TestBaseHandler_Validate(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	handler := NewBaseHandler(clientset, "pods")

	op := Operation{
		Name:        "get",
		Method:      "get",
		DangerLevel: tools.DangerLow,
		Description: "Get a pod",
		Parameters: []tools.Parameter{
			{Name: "name", Type: "string", Description: "Pod name", Required: true},
			{Name: "namespace", Type: "string", Description: "Namespace", Required: false},
		},
	}

	// Test missing required parameter
	args := map[string]interface{}{"namespace": "default"}
	err := handler.Validate(op, args)
	if err == nil {
		t.Error("expected error for missing required parameter 'name'")
	}

	// Test all required parameters present
	args = map[string]interface{}{"name": "test-pod"}
	err = handler.Validate(op, args)
	if err != nil {
		t.Errorf("unexpected error with all required params: %v", err)
	}
}

func TestBaseHandler_CreateTool(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	handler := NewBaseHandler(clientset, "pods")

	op := Operation{
		Name:        "get",
		Method:      "get",
		DangerLevel: tools.DangerLow,
		Description: "Get a pod",
		Parameters: []tools.Parameter{
			{Name: "name", Type: "string", Description: "Pod name", Required: true},
		},
	}

	executor := func(ctx context.Context, args map[string]interface{}) (tools.Result, error) {
		return tools.Result{Success: true, Message: "success"}, nil
	}

	tool := handler.CreateTool(op, executor)

	if tool.Name() != "get" {
		t.Errorf("expected tool name 'get', got %s", tool.Name())
	}

	if tool.Description() != "Get a pod" {
		t.Errorf("expected description 'Get a pod', got %s", tool.Description())
	}

	if tool.Category() != "k8s" {
		t.Errorf("expected category 'k8s', got %s", tool.Category())
	}

	if tool.DangerLevel() != tools.DangerLow {
		t.Errorf("expected danger level low, got %s", tool.DangerLevel())
	}

	params := tool.Parameters()
	if len(params) != 1 {
		t.Errorf("expected 1 parameter, got %d", len(params))
	}

	if params[0].Name != "name" {
		t.Errorf("expected parameter name 'name', got %s", params[0].Name)
	}

	result, err := tool.Execute(context.Background(), map[string]interface{}{"name": "test"})
	if err != nil {
		t.Errorf("unexpected error executing tool: %v", err)
	}

	if !result.Success {
		t.Error("expected success result")
	}
}

func TestBaseHandler_Operations(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	handler := NewBaseHandler(clientset, "pods")

	// Initially should return empty slice
	ops := handler.Operations()
	if len(ops) != 0 {
		t.Errorf("expected empty operations slice, got %d operations", len(ops))
	}
}
