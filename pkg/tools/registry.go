package tools

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Registry manages tool discovery and routing.
type Registry struct {
	tools map[string]Tool
	mu    sync.RWMutex
}

// NewRegistry creates an empty tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry.
func (r *Registry) Register(tool Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[tool.Name()]; exists {
		return fmt.Errorf("tool %s already registered", tool.Name())
	}

	r.tools[tool.Name()] = tool
	return nil
}

// Get retrieves a tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	return tool, exists
}

// List returns all registered tools sorted by category and name.
func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		result = append(result, tool)
	}

	// Sort by category, then name
	sort.Slice(result, func(i, j int) bool {
		if result[i].Category() != result[j].Category() {
			return result[i].Category() < result[j].Category()
		}
		return result[i].Name() < result[j].Name()
	})

	return result
}

// ListByCategory returns tools in a specific category.
func (r *Registry) ListByCategory(category string) []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Tool
	for _, tool := range r.tools {
		if tool.Category() == category {
			result = append(result, tool)
		}
	}
	return result
}

// Execute calls a tool by name with given arguments.
func (r *Registry) Execute(ctx context.Context, name string, args map[string]interface{}) (Result, error) {
	tool, exists := r.Get(name)
	if !exists {
		return Result{}, fmt.Errorf("tool %s not found", name)
	}

	return tool.Execute(ctx, args)
}

// GetLLMDescriptions returns tool info formatted for LLM prompting.
func (r *Registry) GetLLMDescriptions() string {
	tools := r.List()

	var sb strings.Builder
	for _, tool := range tools {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name(), tool.Description()))

		for _, param := range tool.Parameters() {
			req := ""
			if param.Required {
				req = " (required)"
			}
			sb.WriteString(fmt.Sprintf("  • %s: %s%s\n", param.Name, param.Type, req))
		}
	}

	return sb.String()
}
