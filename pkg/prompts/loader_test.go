package prompts

import (
	"strings"
	"testing"

	"k8s-wizard/pkg/tools"
)

func TestLoadEmbeddedTemplates(t *testing.T) {
	loader, err := NewLoader()
	if err != nil {
		t.Fatalf("failed to create loader: %v", err)
	}

	intent, err := loader.GetIntentPrompt("", nil)
	if err != nil {
		t.Fatalf("failed to get intent prompt: %v", err)
	}
	if intent == "" {
		t.Errorf("intent prompt should not be empty")
	}

	// Check that system prompt is included
	if !strings.Contains(intent, "你是一个智能的 Kubernetes 助手") {
		t.Errorf("intent prompt should contain system prompt")
	}

	// Check that user prompt is included
	if !strings.Contains(intent, "用户指令:") {
		t.Errorf("intent prompt should contain user prompt")
	}

	tools := loader.GetToolDescriptions("k8s")
	if len(tools) == 0 {
		t.Errorf("expected at least static tool description for k8s")
	}
}

func TestGetIntentPrompt_ParsingError(t *testing.T) {
	loader, err := NewLoader()
	if err != nil {
		t.Fatalf("failed to create loader: %v", err)
	}

	// Manually corrupt the prompt template to trigger parsing error
	intentPrompt, ok := loader.GetPrompt("intent")
	if !ok {
		t.Fatalf("intent prompt not found")
	}

	// Store original user prompt
	originalUserPrompt := intentPrompt.User
	defer func() {
		intentPrompt.User = originalUserPrompt
	}()

	// Set invalid template syntax
	intentPrompt.User = "{{.UserMessage" // Missing closing brace

	_, err = loader.GetIntentPrompt("test message", nil)
	if err == nil {
		t.Errorf("expected parsing error, got nil")
	}

	if !strings.Contains(err.Error(), "failed to parse template") {
		t.Errorf("error should indicate template parsing failure, got: %v", err)
	}
}

func TestGetIntentPrompt_RenderingError(t *testing.T) {
	loader, err := NewLoader()
	if err != nil {
		t.Fatalf("failed to create loader: %v", err)
	}

	// Manually corrupt the prompt template to trigger rendering error
	intentPrompt, ok := loader.GetPrompt("intent")
	if !ok {
		t.Fatalf("intent prompt not found")
	}

	// Store original user prompt
	originalUserPrompt := intentPrompt.User

	// Set template with an invalid function that will cause execution error
	intentPrompt.User = "{{call .NonExistentFunction}}"

	// Restore after test
	defer func() {
		intentPrompt.User = originalUserPrompt
	}()

	_, err = loader.GetIntentPrompt("test message", nil)
	if err == nil {
		t.Errorf("expected rendering error, got nil")
	}

	if !strings.Contains(err.Error(), "failed to execute template") {
		t.Errorf("error should indicate template execution failure, got: %v", err)
	}
}

func TestUpdateFromRegistry(t *testing.T) {
	loader, err := NewLoader()
	if err != nil {
		t.Fatalf("failed to create loader: %v", err)
	}

	registry := tools.NewRegistry()

	// Test with nil registry
	err = loader.UpdateFromRegistry(nil)
	if err == nil {
		t.Errorf("expected error for nil registry, got nil")
	}

	if !strings.Contains(err.Error(), "registry cannot be nil") {
		t.Errorf("error should indicate nil registry, got: %v", err)
	}

	// Test with valid registry
	err = loader.UpdateFromRegistry(registry)
	if err != nil {
		t.Errorf("failed to update from registry: %v", err)
	}

	// Verify that static tool descriptions are cleared
	tools := loader.GetToolDescriptions("k8s")
	if len(tools) != 0 {
		t.Errorf("expected tool descriptions to be cleared after UpdateFromRegistry, got %d tools", len(tools))
	}
}

func TestGetIntentPrompt_WithRegistry(t *testing.T) {
	loader, err := NewLoader()
	if err != nil {
		t.Fatalf("failed to create loader: %v", err)
	}

	// Test with nil registry (should use static descriptions)
	prompt, err := loader.GetIntentPrompt("test message", nil)
	if err != nil {
		t.Fatalf("failed to get intent prompt with nil registry: %v", err)
	}

	if !strings.Contains(prompt, "test message") {
		t.Errorf("prompt should contain user message, got: %s", prompt)
	}

	if !strings.Contains(prompt, "可用的工具:") {
		t.Errorf("prompt should contain tool descriptions section, got: %s", prompt)
	}

	// Test with empty registry (should not error)
	emptyRegistry := tools.NewRegistry()
	prompt, err = loader.GetIntentPrompt("test message", emptyRegistry)
	if err != nil {
		t.Fatalf("failed to get intent prompt with empty registry: %v", err)
	}

	if !strings.Contains(prompt, "test message") {
		t.Errorf("prompt should contain user message with empty registry, got: %s", prompt)
	}
}

