package workflow

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	lgg "github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/store/sqlite"
)

// CheckpointerManager manages the SQLite checkpointer for session persistence.
type CheckpointerManager struct {
	store   *sqlite.SqliteCheckpointStore
	dataDir string
}

// NewCheckpointerManager creates a new checkpointer manager.
// If dataDir is empty, it defaults to ~/.k8s-wizard/checkpoints
func NewCheckpointerManager(dataDir string) (*CheckpointerManager, error) {
	if dataDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		dataDir = filepath.Join(homeDir, ".k8s-wizard", "checkpoints")
	}

	// Ensure the directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create checkpoints directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "sessions.db")
	store, err := sqlite.NewSqliteCheckpointStore(sqlite.SqliteOptions{
		Path: dbPath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create SQLite checkpoint store: %w", err)
	}

	return &CheckpointerManager{
		store:   store,
		dataDir: dataDir,
	}, nil
}

// GetStore returns the underlying checkpoint store.
func (m *CheckpointerManager) GetStore() lgg.CheckpointStore {
	return m.store
}

// Close closes the checkpointer store.
func (m *CheckpointerManager) Close() error {
	return m.store.Close()
}

// ClearSession clears all checkpoints for a given thread/session.
func (m *CheckpointerManager) ClearSession(ctx context.Context, threadID string) error {
	return m.store.Clear(ctx, threadID)
}

// ListSessions lists all unique thread IDs in the store.
// Note: This is a best-effort implementation as the store doesn't have a direct API for this.
func (m *CheckpointerManager) ListSessions(ctx context.Context) ([]string, error) {
	// This implementation depends on the store's internal structure.
	// For now, we return an empty list as there's no direct API to list all thread IDs.
	// In a production implementation, you might want to track thread IDs separately
	// or use a database query to get unique thread IDs.
	return []string{}, nil
}
