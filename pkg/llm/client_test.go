package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"k8s-wizard/pkg/config"
)

func TestNewClient(t *testing.T) {
	// Set API key
	t.Setenv("GLM_API_KEY", "test-key")

	providerConfig := config.Provider{
		BaseURL: "https://api.test.com/v1",
		Auth:    "api-key",
		API:     "openai-completions",
	}

	client, err := NewClient("glm", "test-model", providerConfig)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if client == nil {
		t.Fatal("expected client to be created")
	}

	if client.Provider != "glm" {
		t.Errorf("Provider = %q, want %q", client.Provider, "glm")
	}

	if client.ModelID != "test-model" {
		t.Errorf("ModelID = %q, want %q", client.ModelID, "test-model")
	}

	if client.BaseURL != "https://api.test.com/v1" {
		t.Errorf("BaseURL = %q, want %q", client.BaseURL, "https://api.test.com/v1")
	}
}

func TestNewClient_MissingAPIKey(t *testing.T) {
	// Ensure no API key is set
	t.Setenv("GLM_API_KEY", "")

	providerConfig := config.Provider{
		BaseURL: "https://api.test.com/v1",
		Auth:    "api-key",
		API:     "openai-completions",
	}

	_, err := NewClient("unknown", "test-model", providerConfig)
	if err == nil {
		t.Error("expected error when API key is missing")
	}
}

func TestGetModel(t *testing.T) {
	client := &ConfiguredClient{
		Provider:   "glm",
		ModelID:    "glm-4-flash",
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}

	model := client.GetModel()
	expected := "glm/glm-4-flash"

	if model != expected {
		t.Errorf("GetModel() = %q, want %q", model, expected)
	}
}

func TestChat_OpenAIFormat(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Authorization header")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type header")
		}

		// Decode request body
		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)
		if reqBody["model"] != "test-model" {
			t.Errorf("expected model in request")
		}

		// Send response
		response := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": "Hello, I am an AI assistant.",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &ConfiguredClient{
		Provider:   "glm",
		ModelID:    "test-model",
		BaseURL:    server.URL,
		apiKey:     "test-key",
		apiFormat:  "openai-completions",
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}

	ctx := context.Background()
	response, err := client.Chat(ctx, "Hello")
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	expected := "Hello, I am an AI assistant."
	if response != expected {
		t.Errorf("Chat() = %q, want %q", response, expected)
	}
}

func TestChat_AnthropicFormat(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("expected x-api-key header")
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("expected anthropic-version header")
		}

		// Decode request body
		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)
		if reqBody["model"] != "claude-test" {
			t.Errorf("expected model in request")
		}

		// Send response
		response := map[string]interface{}{
			"content": []map[string]string{
				{"type": "text", "text": "Hello from Claude!"},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &ConfiguredClient{
		Provider:   "claude",
		ModelID:    "claude-test",
		BaseURL:    server.URL,
		apiKey:     "test-key",
		apiFormat:  "anthropic",
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}

	ctx := context.Background()
	response, err := client.Chat(ctx, "Hello")
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	expected := "Hello from Claude!"
	if response != expected {
		t.Errorf("Chat() = %q, want %q", response, expected)
	}
}

func TestChat_APIError(t *testing.T) {
	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal server error"}`))
	}))
	defer server.Close()

	client := &ConfiguredClient{
		Provider:   "glm",
		ModelID:    "test-model",
		BaseURL:    server.URL,
		apiKey:     "test-key",
		apiFormat:  "openai-completions",
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}

	ctx := context.Background()
	_, err := client.Chat(ctx, "Hello")
	if err == nil {
		t.Error("expected error on API failure")
	}
}

func TestChat_EmptyResponse(t *testing.T) {
	// Create test server that returns empty response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"choices": []interface{}{},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &ConfiguredClient{
		Provider:   "glm",
		ModelID:    "test-model",
		BaseURL:    server.URL,
		apiKey:     "test-key",
		apiFormat:  "openai-completions",
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}

	ctx := context.Background()
	_, err := client.Chat(ctx, "Hello")
	if err == nil {
		t.Error("expected error on empty response")
	}
}

func TestChat_UnsupportedAPIFormat(t *testing.T) {
	client := &ConfiguredClient{
		Provider:   "test",
		ModelID:    "test-model",
		apiFormat:  "unsupported",
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}

	ctx := context.Background()
	_, err := client.Chat(ctx, "Hello")
	if err == nil {
		t.Error("expected error for unsupported API format")
	}
}

func TestChat_ContextCancellation(t *testing.T) {
	// Create test server with delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := &ConfiguredClient{
		Provider:   "glm",
		ModelID:    "test-model",
		BaseURL:    server.URL,
		apiKey:     "test-key",
		apiFormat:  "openai-completions",
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.Chat(ctx, "Hello")
	if err == nil {
		t.Error("expected error on context cancellation")
	}
}

func TestChat_AnthropicMultipleContentBlocks(t *testing.T) {
	// Create test server with multiple content blocks
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"content": []map[string]string{
				{"type": "text", "text": "First "},
				{"type": "other", "text": "Ignored"},
				{"type": "text", "text": "Second"},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &ConfiguredClient{
		Provider:   "claude",
		ModelID:    "claude-test",
		BaseURL:    server.URL,
		apiKey:     "test-key",
		apiFormat:  "anthropic",
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}

	ctx := context.Background()
	response, err := client.Chat(ctx, "Hello")
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	expected := "First Second"
	if response != expected {
		t.Errorf("Chat() = %q, want %q", response, expected)
	}
}

func TestChat_AnthropicEmptyContent(t *testing.T) {
	// Create test server that returns empty content
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"content": []interface{}{},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &ConfiguredClient{
		Provider:   "claude",
		ModelID:    "claude-test",
		BaseURL:    server.URL,
		apiKey:     "test-key",
		apiFormat:  "anthropic",
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}

	ctx := context.Background()
	_, err := client.Chat(ctx, "Hello")
	if err == nil {
		t.Error("expected error on empty content")
	}
}

// Ensure ConfiguredClient implements Client interface
var _ Client = (*ConfiguredClient)(nil)
