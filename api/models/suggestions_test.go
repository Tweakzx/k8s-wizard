package models

import "testing"

// TestSuggestionModelsCompile is a compilation test to ensure Suggestion and SuggestionRequest types exist
// This test should fail initially because the types don't exist yet
func TestSuggestionModelsCompile(t *testing.T) {
	// This test will fail to compile if Suggestion or SuggestionRequest don't exist
	_ = Suggestion{}
	_ = SuggestionRequest{}
}

// Test that ChatResponse includes Suggestions field
func TestChatResponseWithSuggestions(t *testing.T) {
	response := ChatResponse{}
	response.Suggestions = []Suggestion{
		{Type: "reuse", Name: "nginx"},
	}

	if len(response.Suggestions) == 0 {
		t.Error("expected suggestions to be present")
	}
}
