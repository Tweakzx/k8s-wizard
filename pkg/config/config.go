package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Config represents the K8s Wizard configuration
type Config struct {
	Meta   Meta         `json:"meta"`
	Models ModelsConfig `json:"models"`
	Agents AgentsConfig `json:"agents"`
	API    APIConfig    `json:"api"`
	Log    LogConfig    `json:"log,omitempty"`
}

// Meta contains metadata about the config
type Meta struct {
	Version       string `json:"version,omitempty"`
	LastTouchedAt string `json:"lastTouchedAt,omitempty"`
	LastTouchedBy string `json:"lastTouchedBy,omitempty"`
}

// ModelsConfig contains model provider configurations
type ModelsConfig struct {
	Mode      string              `json:"mode,omitempty"`
	Providers map[string]Provider `json:"providers"`
}

// Provider represents an LLM provider
type Provider struct {
	BaseURL string  `json:"baseUrl"`
	Auth    string  `json:"auth"`
	API     string  `json:"api"`
	Models  []Model `json:"models"`
}

// Model represents a model definition
type Model struct {
	ID            string             `json:"id"`
	Name          string             `json:"name"`
	Reasoning     bool               `json:"reasoning,omitempty"`
	ContextWindow int                `json:"contextWindow,omitempty"`
	MaxTokens     int                `json:"maxTokens,omitempty"`
	Cost          map[string]float64 `json:"cost,omitempty"`
}

// AgentsConfig contains agent-related configurations
type AgentsConfig struct {
	Defaults AgentDefaults `json:"defaults"`
}

// AgentDefaults contains default agent settings
type AgentDefaults struct {
	Model  ModelConfig  `json:"model"`
	Models ModelAliases `json:"models,omitempty"`
}

// ModelConfig contains the primary model configuration
type ModelConfig struct {
	Primary string `json:"primary"`
}

// ModelAliases contains model alias mappings
type ModelAliases map[string]AliasConfig

// AliasConfig contains alias configuration
type AliasConfig struct {
	Alias string `json:"alias,omitempty"`
}

// APIConfig contains API server configuration
type APIConfig struct {
	Port int    `json:"port"`
	Host string `json:"host,omitempty"`
}

// LogConfig contains logging configuration
type LogConfig struct {
	EnableFile bool   `json:"enableFile,omitempty"` // Enable file logging
	FilePath   string `json:"filePath,omitempty"`   // Log file path
	MaxSize    int    `json:"maxSize,omitempty"`    // Max size in MB before rotation
	MaxBackups int    `json:"maxBackups,omitempty"` // Max number of old log files
	MaxAge     int    `json:"maxAge,omitempty"`     // Max age in days to retain
	Compress   bool   `json:"compress,omitempty"`   // Compress rotated files
	Level      string `json:"level,omitempty"`      // Log level: debug, info, warn, error
	Format     string `json:"format,omitempty"`     // Log format: json or text
	Console    bool   `json:"console,omitempty"`    // Also output to console
}
// Global config instance
var (
	globalConfig *Config
	configMutex  sync.RWMutex
	configPath   string
)

// Default models configuration
var defaultProviders = map[string]Provider{
	"glm": {
		BaseURL: "https://open.bigmodel.cn/api/coding/paas/v4",
		Auth:    "api-key",
		API:     "openai-completions",
		Models: []Model{
			{
				ID:            "glm-4-flash",
				Name:          "GLM-4 Flash",
				Reasoning:     true,
				ContextWindow: 128000,
				MaxTokens:     131072,
			},
			{
				ID:            "glm-4-air",
				Name:          "GLM-4 Air",
				Reasoning:     false,
				ContextWindow: 128000,
				MaxTokens:     131072,
			},
			{
				ID:            "glm-4",
				Name:          "GLM-4",
				Reasoning:     false,
				ContextWindow: 128000,
				MaxTokens:     131072,
			},
		},
	},
	"deepseek": {
		BaseURL: "https://api.deepseek.com/v1",
		Auth:    "api-key",
		API:     "openai-completions",
		Models: []Model{
			{
				ID:            "deepseek-chat",
				Name:          "DeepSeek Chat",
				Reasoning:     true,
				ContextWindow: 64000,
				MaxTokens:     8192,
			},
			{
				ID:            "deepseek-coder",
				Name:          "DeepSeek Coder",
				Reasoning:     false,
				ContextWindow: 64000,
				MaxTokens:     8192,
			},
		},
	},
	"claude": {
		BaseURL: "https://api.anthropic.com/v1",
		Auth:    "api-key",
		API:     "anthropic",
		Models: []Model{
			{
				ID:            "claude-sonnet-4-20250514",
				Name:          "Claude Sonnet 4",
				Reasoning:     true,
				ContextWindow: 200000,
				MaxTokens:     8192,
			},
		},
	},
}

// GetConfigPath returns the path to the config file
func GetConfigPath() string {
	if configPath != "" {
		return configPath
	}

	// Check for config in XDG config directory
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	// Try XDG config location first
	xdgConfigPath := filepath.Join(home, ".config", "k8s-wizard", "config.json")
	if _, err := os.Stat(xdgConfigPath); err == nil {
		configPath = xdgConfigPath
		return xdgConfigPath
	}

	// Fall back to ~/.k8s-wizard/config.json
	legacyPath := filepath.Join(home, ".k8s-wizard", "config.json")
	configPath = legacyPath
	return legacyPath
}

// GetCredentialPath returns the path to the credentials file
func GetCredentialPath() string {
	home, _ := os.UserHomeDir()

	// Try XDG config location first
	xdgConfigPath := filepath.Join(home, ".config", "k8s-wizard", "credentials.json")
	if _, err := os.Stat(xdgConfigPath); err == nil {
		return xdgConfigPath
	}

	// Fall back to ~/.k8s-wizard/credentials.json
	return filepath.Join(home, ".k8s-wizard", "credentials.json")
}

// LoadConfig loads the configuration from file
func LoadConfig() (*Config, error) {
	configMutex.Lock()
	defer configMutex.Unlock()

	configPath := GetConfigPath()
	if configPath == "" {
		return nil, fmt.Errorf("unable to determine config path")
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create default config
			return createDefaultConfig()
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	globalConfig = &cfg
	return &cfg, nil
}

// createDefaultConfig creates a default configuration
func createDefaultConfig() (*Config, error) {
	cfg := &Config{
		Meta: Meta{
			Version: "1.0.0",
		},
		Models: ModelsConfig{
			Mode:      "merge",
			Providers: defaultProviders,
		},
		Agents: AgentsConfig{
			Defaults: AgentDefaults{
				Model: ModelConfig{
					Primary: "glm/glm-4-flash",
				},
			},
		},
		API: APIConfig{
			Port: 8080,
			Host: "0.0.0.0",
		},
		Log: LogConfig{
			EnableFile: true,
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     30,
			Compress:   true,
			Level:      "info",
			Format:     "json",
			Console:    true,
		},
	}

	// Save default config
	configPath := GetConfigPath()
	if configPath != "" {
		if err := cfg.Save(); err != nil {
			fmt.Printf("Warning: failed to save default config: %v\n", err)
		}
	}

	globalConfig = cfg
	return cfg, nil
}

// Save saves the configuration to file
func (c *Config) Save() error {
	configPath := GetConfigPath()
	if configPath == "" {
		return fmt.Errorf("unable to determine config path")
	}

	// Create config directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Update meta
	c.Meta.LastTouchedAt = ""

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetConfig returns the global config instance
func GetConfig() *Config {
	configMutex.RLock()
	defer configMutex.RUnlock()
	return globalConfig
}

// GetModelProvider parses a model string like "glm/glm-4-flash" and returns provider and model ID
func (c *Config) GetModelProvider(model string) (provider string, modelID string, err error) {
	if model == "" {
		// Use default model
		model = c.Agents.Defaults.Model.Primary
	}

	// Parse model string
	provider, modelID, err = parseModelString(model)
	if err != nil {
		return "", "", err
	}

	// Verify provider exists
	if _, ok := c.Models.Providers[provider]; !ok {
		return "", "", fmt.Errorf("unknown provider: %s", provider)
	}

	return provider, modelID, nil
}

// parseModelString parses a model string like "glm/glm-4-flash"
func parseModelString(model string) (provider string, modelID string, err error) {
	for i := 0; i < len(model); i++ {
		if model[i] == '/' {
			return model[:i], model[i+1:], nil
		}
	}
	return "", "", fmt.Errorf("invalid model format: %s (expected format: provider/model-id)", model)
}

// GetModelConfig returns the configuration for a specific model
func (c *Config) GetModelConfig(model string) (*Provider, *Model, error) {
	provider, modelID, err := c.GetModelProvider(model)
	if err != nil {
		return nil, nil, err
	}

	p, ok := c.Models.Providers[provider]
	if !ok {
		return nil, nil, fmt.Errorf("provider not found: %s", provider)
	}

	// Find the model
	for i := range p.Models {
		if p.Models[i].ID == modelID {
			return &p, &p.Models[i], nil
		}
	}

	return &p, nil, fmt.Errorf("model not found: %s", modelID)
}

// GetAPIKey returns the API key for a provider (from environment or credentials)
func GetAPIKey(provider string) (string, error) {
	// Try environment variables first
	switch provider {
	case "glm":
		if key := os.Getenv("GLM_API_KEY"); key != "" {
			return key, nil
		}
	case "deepseek":
		if key := os.Getenv("DEEPSEEK_API_KEY"); key != "" {
			return key, nil
		}
	case "claude":
		if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
			return key, nil
		}
	}

	// Try credentials file
	credPath := GetCredentialPath()
	if credPath != "" {
		data, err := os.ReadFile(credPath)
		if err == nil {
			type CredProfile map[string]string
			type CredProfiles map[string]CredProfile
			type Creds struct {
				Profiles CredProfiles `json:"profiles"`
			}
			var creds Creds
			if json.Unmarshal(data, &creds) == nil {
				if profile, ok := creds.Profiles[provider+":default"]; ok {
					if key, ok := profile["apiKey"]; ok {
						return key, nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("API key not found for provider: %s (set %s_API_KEY environment variable)", provider, provider)
}

// FetchAvailableModels 从 provider API 获取可用模型列表
func FetchAvailableModels(providerName string, providerConfig Provider, apiKey string) ([]Model, error) {
	switch providerConfig.API {
	case "openai-completions", "":
		return fetchOpenAIModels(providerConfig.BaseURL, apiKey)
	case "anthropic":
		return fetchAnthropicModels(providerConfig.BaseURL, apiKey)
	default:
		return nil, fmt.Errorf("unsupported API format: %s", providerConfig.API)
	}
}

// fetchOpenAIModels 从 OpenAI 兼容 API 获取模型列表
func fetchOpenAIModels(baseURL, apiKey string) ([]Model, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	// 构建 models 端点 URL
	url := strings.TrimSuffix(baseURL, "/") + "/models"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s", string(body))
	}

	var response struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// 转换为 Model 列表
	var models []Model
	for _, m := range response.Data {
		// 过滤掉 embedding 等非聊天模型
		modelID := m.ID
		if strings.Contains(modelID, "embedding") ||
			strings.Contains(modelID, "whisper") ||
			strings.Contains(modelID, "tts") ||
			strings.Contains(modelID, "davinci") ||
			strings.Contains(modelID, "babbage") {
			continue
		}

		models = append(models, Model{
			ID:   modelID,
			Name: modelToName(modelID),
		})
	}

	if len(models) == 0 {
		return nil, fmt.Errorf("no chat models found")
	}

	return models, nil
}

// modelToName 将模型 ID 转换为可读名称
func modelToName(id string) string {
	// 常见模型名称映射
	names := map[string]string{
		// GLM 系列
		"glm-4-flash":      "GLM-4 Flash",
		"glm-4-air":        "GLM-4 Air",
		"glm-4":            "GLM-4",
		"glm-4-plus":       "GLM-4 Plus",
		"glm-4-long":       "GLM-4 Long",
		"glm-4v":           "GLM-4V",
		"glm-4.5":          "GLM-4.5",
		"glm-4.5-air":      "GLM-4.5 Air",
		"glm-4.6":          "GLM-4.6",
		"glm-4.7":          "GLM-4.7",
		"glm-5":            "GLM-5",
		// DeepSeek 系列
		"deepseek-chat":    "DeepSeek Chat",
		"deepseek-coder":   "DeepSeek Coder",
		"deepseek-reasoner": "DeepSeek Reasoner",
	}

	if name, ok := names[id]; ok {
		return name
	}

	// 默认使用 ID 作为名称
	return id
}

// fetchAnthropicModels 从 Anthropic API 获取模型列表
func fetchAnthropicModels(baseURL, apiKey string) ([]Model, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	// 构建 models 端点 URL
	url := strings.TrimSuffix(baseURL, "/") + "/models"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s", string(body))
	}

	var response struct {
		Data []struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// 转换为 Model 列表
	var models []Model
	for _, m := range response.Data {
		name := m.DisplayName
		if name == "" {
			name = m.ID
		}
		models = append(models, Model{
			ID:   m.ID,
			Name: name,
		})
	}

	if len(models) == 0 {
		return nil, fmt.Errorf("no models found")
	}

	return models, nil
}
