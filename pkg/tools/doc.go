package tools

import "fmt"

const (
	// Package documentation
	PackageName     = "tools"
	PackageVersion  = "1.0.0"
	PackageDesc     = "Tool system for K8s operations - uniform abstraction and registry"
)

// Doc returns package documentation.
func Doc() string {
	return fmt.Sprintf(`%s v%s - %s

%s

Provides:
  - Tool interface for uniform K8s operation abstraction
  - Tool registry for dynamic discovery and routing
  - LLM-friendly tool descriptions

Key types:
  - Tool: Uniform operation interface
  - Parameter: Tool input parameter definition
  - Result: Tool execution output
  - DangerLevel: Risk level enumeration

Usage:
  1. Create a tool by implementing the Tool interface
  2. Register tool with registry.Register(tool)
  3. LLM gets tool descriptions via registry.GetLLMDescriptions()
  4. Execute tool via registry.Execute(ctx, name, args)

Example:
  registry := tools.NewRegistry()
  tool := &MyTool{}
  registry.Register(tool)
  result, err := registry.Execute(ctx, "my_tool", args)
`, PackageName, PackageVersion, PackageDesc, PackageDesc)
}
