package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"k8s-wizard/pkg/config"
)

// Client defines the interface for LLM clients.
type Client interface {
	Chat(ctx context.Context, prompt string) (string, error)
	GetModel() string
}

// ConfiguredClient is an LLM client with configuration.
type ConfiguredClient struct {
	Provider   string
	ModelID    string
	BaseURL    string
	AuthType   string
	apiKey     string
	httpClient *http.Client
	apiFormat  string // "openai-completions" or "anthropic"
}

// NewClient creates a new configured LLM client.
func NewClient(provider string, modelID string, providerConfig config.Provider) (*ConfiguredClient, error) {
	apiKey, err := config.GetAPIKey(provider)
	if err != nil {
		return nil, err
	}

	return &ConfiguredClient{
		Provider:   provider,
		ModelID:    modelID,
		BaseURL:    providerConfig.BaseURL,
		AuthType:   providerConfig.Auth,
		apiFormat:  providerConfig.API,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}, nil
}

// GetModel returns the model identifier string.
func (c *ConfiguredClient) GetModel() string {
	return fmt.Sprintf("%s/%s", c.Provider, c.ModelID)
}

// Chat sends a prompt to the LLM and returns the response.
func (c *ConfiguredClient) Chat(ctx context.Context, prompt string) (string, error) {
	switch c.apiFormat {
	case "anthropic":
		return c.chatAnthropic(ctx, prompt)
	case "openai-completions", "":
		return c.chatOpenAIFormat(ctx, prompt)
	default:
		return "", fmt.Errorf("unsupported API format: %s", c.apiFormat)
	}
}

func (c *ConfiguredClient) chatAnthropic(ctx context.Context, prompt string) (string, error) {
	requestBody := map[string]interface{}{
		"model":      c.ModelID,
		"max_tokens": 4096,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/messages", c.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("anthropic-dangerous-direct-browser-access", "true")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error (%s): %s", c.Provider, string(body))
	}

	var response struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}

	if len(response.Content) == 0 {
		return "", fmt.Errorf("empty response")
	}

	var result strings.Builder
	for _, block := range response.Content {
		if block.Type == "text" {
			result.WriteString(block.Text)
		}
	}

	return result.String(), nil
}

func (c *ConfiguredClient) chatOpenAIFormat(ctx context.Context, prompt string) (string, error) {
	requestBody := map[string]interface{}{
		"model": c.ModelID,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	// Determine endpoint based on provider
	var endpoint string
	switch c.Provider {
	case "deepseek":
		endpoint = c.BaseURL + "/chat/completions"
	default:
		endpoint = c.BaseURL + "/chat/completions"
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error (%s): %s", c.Provider, string(body))
	}

	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("empty response")
	}

	return response.Choices[0].Message.Content, nil
}
