// Package logger provides structured logging with file output and rotation support.
package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Config holds logger configuration.
type Config struct {
	// Enable file logging
	EnableFile bool `json:"enableFile"`
	// Log file path (default: ~/.k8s-wizard/logs/k8s-wizard.log)
	FilePath string `json:"filePath"`
	// Max size in MB before rotation (default: 100)
	MaxSize int `json:"maxSize"`
	// Max number of old log files to retain (default: 3)
	MaxBackups int `json:"maxBackups"`
	// Max age in days to retain old log files (default: 30)
	MaxAge int `json:"maxAge"`
	// Compress rotated files
	Compress bool `json:"compress"`
	// Log level: debug, info, warn, error (default: info)
	Level string `json:"level"`
	// Log format: json or text (default: json)
	Format string `json:"format"`
	// Also output to console
	Console bool `json:"console"`
}

// DefaultConfig returns default logger configuration.
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	return &Config{
		EnableFile: true,
		FilePath:   filepath.Join(homeDir, ".k8s-wizard", "logs", "k8s-wizard.log"),
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     30,
		Compress:   true,
		Level:      "info",
		Format:     "json",
		Console:    true,
	}
}

// Logger wraps slog.Logger with additional functionality.
type Logger struct {
	*slog.Logger
	config  *Config
	closers []io.Closer
	mu      sync.Mutex
}

var (
	globalLogger *Logger
	once         sync.Once
)

// Init initializes the global logger with the given configuration.
func Init(cfg *Config) (*Logger, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Apply defaults
	if cfg.MaxSize <= 0 {
		cfg.MaxSize = 100
	}
	if cfg.MaxBackups <= 0 {
		cfg.MaxBackups = 3
	}
	if cfg.MaxAge <= 0 {
		cfg.MaxAge = 30
	}
	if cfg.Level == "" {
		cfg.Level = "info"
	}
	if cfg.Format == "" {
		cfg.Format = "json"
	}

	var writers []io.Writer
	var closers []io.Closer

	// Console output
	if cfg.Console {
		writers = append(writers, os.Stdout)
	}

	// File output with rotation
	if cfg.EnableFile && cfg.FilePath != "" {
		// Ensure directory exists
		logDir := filepath.Dir(cfg.FilePath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, err
		}

		fileWriter := &lumberjack.Logger{
			Filename:   cfg.FilePath,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		}
		writers = append(writers, fileWriter)
		closers = append(closers, fileWriter)
	}

	// Multi-writer for concurrent output
	var writer io.Writer
	if len(writers) == 0 {
		writer = os.Stdout
	} else if len(writers) == 1 {
		writer = writers[0]
	} else {
		writer = io.MultiWriter(writers...)
	}

	// Create slog handler
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: parseLevel(cfg.Level),
	}

	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(writer, opts)
	} else {
		handler = slog.NewTextHandler(writer, opts)
	}

	logger := &Logger{
		Logger:  slog.New(handler),
		config:  cfg,
		closers: closers,
	}

	// Set as global logger
	globalLogger = logger
	slog.SetDefault(logger.Logger)

	return logger, nil
}

// Get returns the global logger instance.
func Get() *Logger {
	once.Do(func() {
		if globalLogger == nil {
			globalLogger, _ = Init(DefaultConfig())
		}
	})
	return globalLogger
}

// Close closes all log writers.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	var lastErr error
	for _, closer := range l.closers {
		if err := closer.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// With returns a logger with additional context.
func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		Logger:  l.Logger.With(args...),
		config:  l.config,
		closers: l.closers,
	}
}

// WithGroup returns a logger with a group prefix.
func (l *Logger) WithGroup(name string) *Logger {
	return &Logger{
		Logger:  l.Logger.WithGroup(name),
		config:  l.config,
		closers: l.closers,
	}
}

// Sync flushes any buffered log entries.
func (l *Logger) Sync() error {
	// slog doesn't have a Sync method, but we implement it for compatibility
	return nil
}

// IsDebugEnabled returns true if debug level is enabled.
func (l *Logger) IsDebugEnabled() bool {
	return l.config.Level == "debug"
}

// parseLevel parses log level string.
func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Convenience functions using global logger

// Debug logs at debug level.
func Debug(msg string, args ...any) {
	Get().Debug(msg, args...)
}

// Info logs at info level.
func Info(msg string, args ...any) {
	Get().Info(msg, args...)
}

// Warn logs at warn level.
func Warn(msg string, args ...any) {
	Get().Warn(msg, args...)
}

// Error logs at error level.
func Error(msg string, args ...any) {
	Get().Error(msg, args...)
}

// With returns a logger with additional context.
func With(args ...any) *Logger {
	return Get().With(args...)
}

// Close closes the global logger.
func Close() error {
	if globalLogger != nil {
		return globalLogger.Close()
	}
	return nil
}
