package workflow

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewCheckpointerManager_DefaultDir(t *testing.T) {
	// Use temp directory as base
	tmpDir := t.TempDir()

	// Override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	cm, err := NewCheckpointerManager("")
	if err != nil {
		t.Fatalf("NewCheckpointerManager() error = %v", err)
	}

	if cm == nil {
		t.Fatal("expected checkpointer manager to be created")
	}
	defer cm.Close()

	// Verify default directory was created
	expectedDir := filepath.Join(tmpDir, ".k8s-wizard", "checkpoints")
	if cm.dataDir != expectedDir {
		t.Errorf("dataDir = %q, want %q", cm.dataDir, expectedDir)
	}
}

func TestNewCheckpointerManager_CustomDir(t *testing.T) {
	customDir := t.TempDir()

	cm, err := NewCheckpointerManager(customDir)
	if err != nil {
		t.Fatalf("NewCheckpointerManager() error = %v", err)
	}

	if cm == nil {
		t.Fatal("expected checkpointer manager to be created")
	}
	defer cm.Close()

	if cm.dataDir != customDir {
		t.Errorf("dataDir = %q, want %q", cm.dataDir, customDir)
	}
}

func TestNewCheckpointerManager_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	customDir := filepath.Join(tmpDir, "nested", "checkpoints")

	cm, err := NewCheckpointerManager(customDir)
	if err != nil {
		t.Fatalf("NewCheckpointerManager() error = %v", err)
	}
	defer cm.Close()

	// Verify directory was created
	info, err := os.Stat(customDir)
	if err != nil {
		t.Fatalf("directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory to be created")
	}
}

func TestCheckpointerManager_GetStore(t *testing.T) {
	cm, err := NewCheckpointerManager(t.TempDir())
	if err != nil {
		t.Fatalf("NewCheckpointerManager() error = %v", err)
	}
	defer cm.Close()

	store := cm.GetStore()
	if store == nil {
		t.Error("expected store to be returned")
	}
}

func TestCheckpointerManager_Close(t *testing.T) {
	cm, err := NewCheckpointerManager(t.TempDir())
	if err != nil {
		t.Fatalf("NewCheckpointerManager() error = %v", err)
	}

	err = cm.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestCheckpointerManager_ClearSession(t *testing.T) {
	cm, err := NewCheckpointerManager(t.TempDir())
	if err != nil {
		t.Fatalf("NewCheckpointerManager() error = %v", err)
	}
	defer cm.Close()

	ctx := context.Background()
	threadID := "test-thread-123"

	err = cm.ClearSession(ctx, threadID)
	if err != nil {
		t.Errorf("ClearSession() error = %v", err)
	}
}

func TestCheckpointerManager_ListSessions(t *testing.T) {
	cm, err := NewCheckpointerManager(t.TempDir())
	if err != nil {
		t.Fatalf("NewCheckpointerManager() error = %v", err)
	}
	defer cm.Close()

	ctx := context.Background()

	sessions, err := cm.ListSessions(ctx)
	if err != nil {
		t.Errorf("ListSessions() error = %v", err)
	}

	// Currently returns empty list as documented
	if sessions == nil {
		t.Error("expected sessions slice, got nil")
	}
}

func TestCheckpointerManager_DatabaseFile(t *testing.T) {
	tmpDir := t.TempDir()

	cm, err := NewCheckpointerManager(tmpDir)
	if err != nil {
		t.Fatalf("NewCheckpointerManager() error = %v", err)
	}
	defer cm.Close()

	// Verify database file was created
	dbPath := filepath.Join(tmpDir, "sessions.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("expected database file to be created")
	}
}

func TestCheckpointerManager_MultipleInstances(t *testing.T) {
	tmpDir := t.TempDir()

	// Create first instance
	cm1, err := NewCheckpointerManager(tmpDir)
	if err != nil {
		t.Fatalf("First NewCheckpointerManager() error = %v", err)
	}

	// Create second instance with same directory should work
	cm2, err := NewCheckpointerManager(tmpDir)
	if err != nil {
		cm1.Close()
		t.Fatalf("Second NewCheckpointerManager() error = %v", err)
	}

	cm1.Close()
	cm2.Close()
}
