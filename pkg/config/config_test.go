package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParseModelString(t *testing.T) {
	tests := []struct {
		name         string
		model        string
		wantProvider string
		wantModelID  string
		wantErr      bool
	}{
		{
			name:         "valid glm model",
			model:        "glm/glm-4-flash",
			wantProvider: "glm",
			wantModelID:  "glm-4-flash",
			wantErr:      false,
		},
		{
			name:         "valid deepseek model",
			model:        "deepseek/deepseek-chat",
			wantProvider: "deepseek",
			wantModelID:  "deepseek-chat",
			wantErr:      false,
		},
		{
			name:         "valid claude model",
			model:        "claude/claude-sonnet-4-20250514",
			wantProvider: "claude",
			wantModelID:  "claude-sonnet-4-20250514",
			wantErr:      false,
		},
		{
			name:    "missing slash",
			model:   "glm-4-flash",
			wantErr: true,
		},
		{
			name:    "empty string",
			model:   "",
			wantErr: true,
		},
		{
			name:         "only slash",
			model:        "/",
			wantProvider: "",
			wantModelID:  "",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, modelID, err := parseModelString(tt.model)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if provider != tt.wantProvider {
				t.Errorf("provider = %q, want %q", provider, tt.wantProvider)
			}
			if modelID != tt.wantModelID {
				t.Errorf("modelID = %q, want %q", modelID, tt.wantModelID)
			}
		})
	}
}

func TestModelToName(t *testing.T) {
	tests := []struct {
		id       string
		expected string
	}{
		{"glm-4-flash", "GLM-4 Flash"},
		{"glm-4-air", "GLM-4 Air"},
		{"glm-4", "GLM-4"},
		{"deepseek-chat", "DeepSeek Chat"},
		{"deepseek-coder", "DeepSeek Coder"},
		{"unknown-model", "unknown-model"}, // Falls back to ID
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			result := modelToName(tt.id)
			if result != tt.expected {
				t.Errorf("modelToName(%q) = %q, want %q", tt.id, result, tt.expected)
			}
		})
	}
}

func TestCreateDefaultConfig(t *testing.T) {
	// Use temp directory
	tmpDir := t.TempDir()
	configPath = filepath.Join(tmpDir, "config.json")
	defer func() { configPath = "" }()

	cfg, err := createDefaultConfig()
	if err != nil {
		t.Fatalf("createDefaultConfig() error = %v", err)
	}

	if cfg == nil {
		t.Fatal("expected config to be created")
	}

	// Verify default values
	if cfg.Meta.Version != "1.0.0" {
		t.Errorf("version = %q, want %q", cfg.Meta.Version, "1.0.0")
	}

	if cfg.Agents.Defaults.Model.Primary != "glm/glm-4-flash" {
		t.Errorf("primary model = %q, want %q", cfg.Agents.Defaults.Model.Primary, "glm/glm-4-flash")
	}

	if cfg.API.Port != 8080 {
		t.Errorf("port = %d, want 8080", cfg.API.Port)
	}

	// Verify providers exist
	if _, ok := cfg.Models.Providers["glm"]; !ok {
		t.Error("expected glm provider")
	}
	if _, ok := cfg.Models.Providers["deepseek"]; !ok {
		t.Error("expected deepseek provider")
	}
	if _, ok := cfg.Models.Providers["claude"]; !ok {
		t.Error("expected claude provider")
	}
}

func TestConfig_Save(t *testing.T) {
	tmpDir := t.TempDir()
	configPath = filepath.Join(tmpDir, "config.json")
	defer func() { configPath = "" }()

	cfg := &Config{
		Meta: Meta{Version: "1.0.0"},
		API:  APIConfig{Port: 9090},
	}

	err := cfg.Save()
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}

	// Verify content
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var loaded Config
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if loaded.API.Port != 9090 {
		t.Errorf("port = %d, want 9090", loaded.API.Port)
	}
}

func TestGetModelProvider(t *testing.T) {
	cfg := &Config{
		Models: ModelsConfig{
			Providers: defaultProviders,
		},
		Agents: AgentsConfig{
			Defaults: AgentDefaults{
				Model: ModelConfig{Primary: "glm/glm-4-flash"},
			},
		},
	}

	tests := []struct {
		name         string
		model        string
		wantProvider string
		wantModelID  string
		wantErr      bool
	}{
		{
			name:         "explicit glm model",
			model:        "glm/glm-4-flash",
			wantProvider: "glm",
			wantModelID:  "glm-4-flash",
			wantErr:      false,
		},
		{
			name:         "empty string uses default",
			model:        "",
			wantProvider: "glm",
			wantModelID:  "glm-4-flash",
			wantErr:      false,
		},
		{
			name:    "unknown provider",
			model:   "unknown/model",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, modelID, err := cfg.GetModelProvider(tt.model)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if provider != tt.wantProvider {
				t.Errorf("provider = %q, want %q", provider, tt.wantProvider)
			}
			if modelID != tt.wantModelID {
				t.Errorf("modelID = %q, want %q", modelID, tt.wantModelID)
			}
		})
	}
}

func TestGetModelConfig(t *testing.T) {
	cfg := &Config{
		Models: ModelsConfig{
			Providers: defaultProviders,
		},
		Agents: AgentsConfig{
			Defaults: AgentDefaults{
				Model: ModelConfig{Primary: "glm/glm-4-flash"},
			},
		},
	}

	tests := []struct {
		name         string
		model        string
		wantProvider bool
		wantModel    bool
		wantErr      bool
	}{
		{
			name:         "existing model",
			model:        "glm/glm-4-flash",
			wantProvider: true,
			wantModel:    true,
			wantErr:      false,
		},
		{
			name:         "provider exists but model doesn't",
			model:        "glm/nonexistent",
			wantProvider: true,
			wantModel:    false,
			wantErr:      true,
		},
		{
			name:    "invalid format",
			model:   "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, model, err := cfg.GetModelConfig(tt.model)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if tt.wantProvider && provider == nil {
				t.Error("expected provider to be returned")
			}
			if tt.wantModel && model == nil {
				t.Error("expected model to be returned")
			}
		})
	}
}

func TestGetAPIKey_EnvironmentVariable(t *testing.T) {
	// Set environment variables
	os.Setenv("GLM_API_KEY", "test-glm-key")
	os.Setenv("DEEPSEEK_API_KEY", "test-deepseek-key")
	os.Setenv("ANTHROPIC_API_KEY", "test-anthropic-key")
	defer func() {
		os.Unsetenv("GLM_API_KEY")
		os.Unsetenv("DEEPSEEK_API_KEY")
		os.Unsetenv("ANTHROPIC_API_KEY")
	}()

	tests := []struct {
		provider string
		expected string
		wantErr  bool
	}{
		{"glm", "test-glm-key", false},
		{"deepseek", "test-deepseek-key", false},
		{"claude", "test-anthropic-key", false},
		{"unknown", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			key, err := GetAPIKey(tt.provider)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if key != tt.expected {
				t.Errorf("key = %q, want %q", key, tt.expected)
			}
		})
	}
}

func TestGetAPIKey_CredentialsFile(t *testing.T) {
	// Clear environment variables
	os.Unsetenv("GLM_API_KEY")

	// Create temp credentials file
	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "credentials.json")

	credentials := map[string]interface{}{
		"profiles": map[string]interface{}{
			"glm:default": map[string]string{
				"apiKey": "file-glm-key",
			},
		},
	}

	data, _ := json.Marshal(credentials)
	os.WriteFile(credPath, data, 0600)

	// Override credential path check
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)

	// Create the expected directory structure
	configDir := filepath.Join(tmpDir, ".config", "k8s-wizard")
	os.MkdirAll(configDir, 0755)
	os.WriteFile(filepath.Join(configDir, "credentials.json"), data, 0600)

	defer os.Setenv("HOME", originalHome)

	key, err := GetAPIKey("glm")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}
	if key != "file-glm-key" {
		t.Errorf("key = %q, want %q", key, "file-glm-key")
	}
}

func TestGetConfigPath_XDG(t *testing.T) {
	tmpDir := t.TempDir()

	// Create XDG config directory with config file
	xdgConfigDir := filepath.Join(tmpDir, ".config", "k8s-wizard")
	os.MkdirAll(xdgConfigDir, 0755)
	xdgConfigPath := filepath.Join(xdgConfigDir, "config.json")
	os.WriteFile(xdgConfigPath, []byte(`{"meta":{"version":"test"}}`), 0600)

	// Create legacy directory (should be ignored)
	legacyDir := filepath.Join(tmpDir, ".k8s-wizard")
	os.MkdirAll(legacyDir, 0755)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	configPath = "" // Reset
	defer func() {
		os.Setenv("HOME", originalHome)
		configPath = ""
	}()

	result := GetConfigPath()
	if result != xdgConfigPath {
		t.Errorf("GetConfigPath() = %q, want %q", result, xdgConfigPath)
	}
}

func TestGetConfigPath_Legacy(t *testing.T) {
	tmpDir := t.TempDir()

	// Only create legacy directory
	legacyDir := filepath.Join(tmpDir, ".k8s-wizard")
	os.MkdirAll(legacyDir, 0755)
	legacyPath := filepath.Join(legacyDir, "config.json")
	os.WriteFile(legacyPath, []byte(`{"meta":{"version":"test"}}`), 0600)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	configPath = "" // Reset
	defer func() {
		os.Setenv("HOME", originalHome)
		configPath = ""
	}()

	result := GetConfigPath()
	if result != legacyPath {
		t.Errorf("GetConfigPath() = %q, want %q", result, legacyPath)
	}
}
