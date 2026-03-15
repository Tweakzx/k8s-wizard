// Package prompts provides prompt template management for LLM-based Kubernetes operations.
//
// The prompts package manages loading, formatting, and rendering of prompt templates
// used for intent parsing and tool selection in the K8s Wizard system. It supports
// both static tool descriptions loaded from YAML files and dynamic tool descriptions
// from a live tool registry.
//
// # Usage
//
// Create a new loader and use it to generate prompts for user messages:
//
//	loader, err := prompts.NewLoader()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Generate intent prompt with static tool descriptions
//	prompt, err := loader.GetIntentPrompt("create a deployment", nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Or use dynamic tool descriptions from registry
//	registry := tools.NewRegistry()
//	// Register tools...
//	if err := loader.UpdateFromRegistry(registry); err != nil {
//	    log.Fatal(err)
//	}
//	prompt, err := loader.GetIntentPrompt("create a deployment", registry)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Prompt Structure
//
// Prompts are loaded from embedded YAML files in the templates directory:
//   - intent.yaml: System and user prompts for intent parsing
//   - tools.yaml: Static tool descriptions (optional, for reference)
//
// Each prompt contains:
//   - Name: Prompt identifier
//   - Version: Version string
//   - Description: Human-readable description
//   - System: System prompt content
//   - User: User prompt template (supports Go template syntax)
//
// # Tool Descriptions
//
// The loader supports two modes for tool descriptions:
//   1. Static: Loaded from tools.yaml file (fallback mode)
//   2. Dynamic: Retrieved from a tools.Registry via GetLLMDescriptions()
//
// When a tool registry is provided to GetIntentPrompt, dynamic descriptions
// are used. Otherwise, static descriptions are used as fallback.
//
// # Templates
//
// User prompts support Go template syntax with the following variables:
//   - UserMessage: The raw user input message
//   - ToolDescriptions: Formatted tool descriptions
//
// # Error Handling
//
// All prompt generation methods return errors for proper error handling:
//   - Template parsing errors
//   - Template execution errors
//   - Missing prompt errors
//   - Registry validation errors
//
// Package prompts is part of the K8s Wizard system and integrates with
// the tools package for dynamic tool management.
package prompts
