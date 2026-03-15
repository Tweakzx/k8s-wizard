package prompts

import "testing"

func TestLoadEmbeddedTemplates(t *testing.T) {
	loader, err := NewLoader()
	if err != nil {
		t.Fatalf("failed to create loader: %v", err)
	}

	intent := loader.GetIntentPrompt("")
	if intent == "" {
		t.Errorf("intent prompt should not be empty")
	}

	tools := loader.GetToolDescriptions("k8s")
	if len(tools) == 0 {
		t.Errorf("expected at least static tool description for k8s")
	}
}
