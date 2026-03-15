package handlers

import (
	"context"
	"fmt"

	"k8s-wizard/pkg/tools"
)

// Operation represents a specific action on a resource.
type Operation struct {
	Name        string
	Method      string    // create, get, list, delete, update, scale, describe, logs, exec, apply
	DangerLevel tools.DangerLevel
	Description string
	Parameters  []tools.Parameter
}

// BaseHandler provides common functionality for resource handlers.
type BaseHandler struct {
	clientset interface{} // Will be kubernetes.Interface, using interface{} to avoid import issues
	resource  string
	ops       []Operation
}

// NewBaseHandler creates a base handler.
func NewBaseHandler(clientset interface{}, resource string) *BaseHandler {
	return &BaseHandler{
		clientset: clientset,
		resource:  resource,
	}
}

// Resource implements Handler.
func (h *BaseHandler) Resource() string {
	return h.resource
}

// Operations returns a list of operations.
func (h *BaseHandler) Operations() []Operation {
	return h.ops
}

// Validate provides basic validation.
func (h *BaseHandler) Validate(op Operation, args map[string]interface{}) error {
	for _, param := range op.Parameters {
		if param.Required {
			if _, exists := args[param.Name]; !exists {
				return fmt.Errorf("required parameter %s missing", param.Name)
			}
		}
	}
	return nil
}

// CreateTool creates a tool from an operation.
func (h *BaseHandler) CreateTool(op Operation, executor ToolExecutor) tools.Tool {
	return &operationTool{
		handler:  h,
		op:       op,
		executor: executor,
	}
}

// ToolExecutor defines how to execute an operation.
type ToolExecutor func(ctx context.Context, args map[string]interface{}) (tools.Result, error)

// operationTool implements tools.Tool for an operation.
type operationTool struct {
	handler  *BaseHandler
	op       Operation
	executor ToolExecutor
}

// Name returns the operation name.
func (t *operationTool) Name() string {
	return t.op.Name
}

// Description returns the operation description.
func (t *operationTool) Description() string {
	return t.op.Description
}

// Category returns "k8s" for all k8s operations.
func (t *operationTool) Category() string {
	return "k8s"
}

// Parameters returns the operation parameters.
func (t *operationTool) Parameters() []tools.Parameter {
	return t.op.Parameters
}

// DangerLevel returns the operation danger level.
func (t *operationTool) DangerLevel() tools.DangerLevel {
	return t.op.DangerLevel
}

// Execute runs the tool with given arguments.
func (t *operationTool) Execute(ctx context.Context, args map[string]interface{}) (tools.Result, error) {
	return t.executor(ctx, args)
}
