package logger

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("expected config to be returned")
	}

	if !cfg.EnableFile {
		t.Error("expected EnableFile to be true by default")
	}
	if cfg.MaxSize != 100 {
		t.Errorf("expected MaxSize 100, got %d", cfg.MaxSize)
	}
	if cfg.MaxBackups != 3 {
		t.Errorf("expected MaxBackups 3, got %d", cfg.MaxBackups)
	}
	if cfg.MaxAge != 30 {
		t.Errorf("expected MaxAge 30, got %d", cfg.MaxAge)
	}
	if !cfg.Compress {
		t.Error("expected Compress to be true by default")
	}
	if cfg.Level != "info" {
		t.Errorf("expected Level 'info', got %q", cfg.Level)
	}
	if cfg.Format != "json" {
		t.Errorf("expected Format 'json', got %q", cfg.Format)
	}
	if !cfg.Console {
		t.Error("expected Console to be true by default")
	}
}

func TestInit_ConsoleOnly(t *testing.T) {
	cfg := &Config{
		EnableFile: false,
		Console:    true,
		Level:      "debug",
		Format:     "text",
	}

	log, err := Init(cfg)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if log == nil {
		t.Fatal("expected logger to be returned")
	}

	// Test logging
	log.Debug("test debug message")
	log.Info("test info message")
	log.Warn("test warn message")
	log.Error("test error message")

	// Clean up
	log.Close()
}

func TestInit_FileLogging(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	cfg := &Config{
		EnableFile: true,
		FilePath:   logFile,
		MaxSize:    10,
		MaxBackups: 1,
		MaxAge:     7,
		Compress:   false,
		Level:      "info",
		Format:     "json",
		Console:    false,
	}

	log, err := Init(cfg)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer log.Close()

	// Write some logs
	log.Info("test message", "key", "value")

	// Close to flush
	log.Close()

	// Verify file was created
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Fatal("log file was not created")
	}

	// Read and verify content
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "test message") {
		t.Errorf("log file does not contain 'test message': %s", content)
	}

	// Verify JSON format
	var entry map[string]interface{}
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Errorf("log is not valid JSON: %v", err)
	}
}

func TestInit_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "nested", "logs")
	logFile := filepath.Join(logDir, "test.log")

	cfg := &Config{
		EnableFile: true,
		FilePath:   logFile,
		Level:      "info",
		Format:     "json",
		Console:    false,
	}

	log, err := Init(cfg)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer log.Close()

	// Verify directory was created
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		t.Error("log directory was not created")
	}
}

func TestLogger_With(t *testing.T) {
	cfg := &Config{
		EnableFile: false,
		Console:    true,
		Level:      "info",
		Format:     "json",
	}

	log, err := Init(cfg)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer log.Close()

	// Create child logger with context
	childLog := log.With("service", "test", "version", "1.0")
	if childLog == nil {
		t.Fatal("expected child logger to be created")
	}

	childLog.Info("test message with context")
}

func TestLogger_Levels(t *testing.T) {
	tests := []struct {
		level   string
		allowed []string
		blocked []string
	}{
		{
			level:   "debug",
			allowed: []string{"debug", "info", "warn", "error"},
			blocked: []string{},
		},
		{
			level:   "info",
			allowed: []string{"info", "warn", "error"},
			blocked: []string{"debug"},
		},
		{
			level:   "warn",
			allowed: []string{"warn", "error"},
			blocked: []string{"debug", "info"},
		},
		{
			level:   "error",
			allowed: []string{"error"},
			blocked: []string{"debug", "info", "warn"},
		},
	}

	for _, tt := range tests {
		t.Run("level_"+tt.level, func(t *testing.T) {
			cfg := &Config{
				EnableFile: false,
				Console:    true,
				Level:      tt.level,
				Format:     "json",
			}

			log, err := Init(cfg)
			if err != nil {
				t.Fatalf("Init() error = %v", err)
			}
			defer log.Close()

			// Test that allowed levels work
			for _, lvl := range tt.allowed {
				switch lvl {
				case "debug":
					log.Debug("test")
				case "info":
					log.Info("test")
				case "warn":
					log.Warn("test")
				case "error":
					log.Error("test")
				}
			}
		})
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"debug", "DEBUG"},
		{"info", "INFO"},
		{"warn", "WARN"},
		{"error", "ERROR"},
		{"unknown", "INFO"}, // defaults to info
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			level := parseLevel(tt.input)
			if level.String() != tt.expected {
				t.Errorf("parseLevel(%q) = %q, want %q", tt.input, level.String(), tt.expected)
			}
		})
	}
}

func TestGet(t *testing.T) {
	// Reset global logger
	globalLogger = nil

	// First call should create default logger
	log := Get()
	if log == nil {
		t.Fatal("expected logger to be returned")
	}

	// Second call should return same instance
	log2 := Get()
	if log2 != log {
		t.Error("expected same logger instance")
	}
}

func TestConvenienceFunctions(t *testing.T) {
	// Reset and initialize
	globalLogger = nil
	_, err := Init(&Config{
		EnableFile: false,
		Console:    true,
		Level:      "debug",
		Format:     "text",
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Test convenience functions
	Debug("debug message", "key", "value")
	Info("info message", "key", "value")
	Warn("warn message", "key", "value")
	Error("error message", "key", "value")

	// Test With
	childLog := With("context", "test")
	if childLog == nil {
		t.Error("expected child logger from With()")
	}
	childLog.Info("child message")
}

func TestClose(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	log, err := Init(&Config{
		EnableFile: true,
		FilePath:   logFile,
		Level:      "info",
		Format:     "json",
		Console:    false,
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Close should not error
	if err := log.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestIsDebugEnabled(t *testing.T) {
	tests := []struct {
		level    string
		expected bool
	}{
		{"debug", true},
		{"info", false},
		{"warn", false},
		{"error", false},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			log, _ := Init(&Config{
				EnableFile: false,
				Console:    true,
				Level:      tt.level,
				Format:     "text",
			})
			defer log.Close()

			if log.IsDebugEnabled() != tt.expected {
				t.Errorf("IsDebugEnabled() = %v, want %v", log.IsDebugEnabled(), tt.expected)
			}
		})
	}
}

func TestContextCancellation(t *testing.T) {
	// Test that context cancellation doesn't cause issues
	log, _ := Init(&Config{
		EnableFile: false,
		Console:    true,
		Level:      "info",
		Format:     "text",
	})
	defer log.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Should still work
	log.InfoContext(ctx, "message after cancel")
}
