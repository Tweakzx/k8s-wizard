package tools

import "context"

// Tool represents a discrete operation that agent can perform.
type Tool interface {
	// Name returns a unique identifier for this tool.
	Name() string

	// Description explains what this tool does (used by LLM).
	Description() string

	// Category groups related tools (e.g., "k8s", "llm", "builtin").
	Category() string

	// Parameters describes expected inputs for LLM prompting.
	Parameters() []Parameter

	// DangerLevel indicates the risk level of this operation.
	DangerLevel() DangerLevel

	// Execute runs the tool with given arguments.
	Execute(ctx context.Context, args map[string]interface{}) (Result, error)
}

// operationTool implements Tool for a simple operation.
type operationTool struct {
	name     string
	executor func(context.Context, map[string]interface{}) (Result, error)
}

func (t *operationTool) Name() string {
	return t.name
}

func (t *operationTool) Description() string {
	return "Operation tool"
}

func (t *operationTool) Category() string {
	return "builtin"
}

func (t *operationTool) Parameters() []Parameter {
	return nil
}

func (t *operationTool) DangerLevel() DangerLevel {
	return DangerLow
}

func (t *operationTool) Execute(ctx context.Context, args map[string]interface{}) (Result, error) {
	return t.executor(ctx, args)
}
