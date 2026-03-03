package workflow

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewCheckpointerManager(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "checkpointer-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cm, err := NewCheckpointerManager(tmpDir)
	if err != nil {
		t.Fatalf("NewCheckpointerManager failed: %v", err)
	}
	defer cm.Close()

	// Verify directory was created
	dbPath := filepath.Join(tmpDir, "sessions.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("Database file not created at %s", dbPath)
	}
}

func TestNewCheckpointerManagerDefaultDir(t *testing.T) {
	cm, err := NewCheckpointerManager("")
	if err != nil {
		t.Fatalf("NewCheckpointerManager with empty dir failed: %v", err)
	}
	defer cm.Close()

	// Verify default directory was created
	if cm.dataDir == "" {
		t.Error("dataDir should not be empty")
	}
}

func TestTouchSession(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "checkpointer-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cm, err := NewCheckpointerManager(tmpDir)
	if err != nil {
		t.Fatalf("NewCheckpointerManager failed: %v", err)
	}
	defer cm.Close()

	ctx := context.Background()
	threadID := "test-thread-123"

	// First touch - should create session
	if err := cm.TouchSession(ctx, threadID); err != nil {
		t.Fatalf("TouchSession failed: %v", err)
	}

	// Verify session was created
	info, err := cm.GetSessionInfo(ctx, threadID)
	if err != nil {
		t.Fatalf("GetSessionInfo failed: %v", err)
	}
	if info == nil {
		t.Fatal("Expected session info, got nil")
	}
	if info.ThreadID != threadID {
		t.Errorf("ThreadID = %q, want %q", info.ThreadID, threadID)
	}
	if info.CheckpointCount != 1 {
		t.Errorf("CheckpointCount = %d, want 1", info.CheckpointCount)
	}

	// Second touch - should increment checkpoint count
	if err := cm.TouchSession(ctx, threadID); err != nil {
		t.Fatalf("Second TouchSession failed: %v", err)
	}

	info, err = cm.GetSessionInfo(ctx, threadID)
	if err != nil {
		t.Fatalf("GetSessionInfo after second touch failed: %v", err)
	}
	if info.CheckpointCount != 2 {
		t.Errorf("CheckpointCount after second touch = %d, want 2", info.CheckpointCount)
	}
}

func TestListSessions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "checkpointer-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cm, err := NewCheckpointerManager(tmpDir)
	if err != nil {
		t.Fatalf("NewCheckpointerManager failed: %v", err)
	}
	defer cm.Close()

	ctx := context.Background()

	// Initially should have no sessions
	sessions, err := cm.ListSessions(ctx)
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("Expected 0 sessions, got %d", len(sessions))
	}

	// Add some sessions
	threadIDs := []string{"thread-1", "thread-2", "thread-3"}
	for _, tid := range threadIDs {
		if err := cm.TouchSession(ctx, tid); err != nil {
			t.Fatalf("TouchSession failed for %s: %v", tid, err)
		}
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	sessions, err = cm.ListSessions(ctx)
	if err != nil {
		t.Fatalf("ListSessions after adding sessions failed: %v", err)
	}
	if len(sessions) != 3 {
		t.Errorf("Expected 3 sessions, got %d", len(sessions))
	}

	// Verify sessions are ordered by last_active DESC
	if sessions[0].ThreadID != "thread-3" {
		t.Errorf("First session should be thread-3 (most recent), got %s", sessions[0].ThreadID)
	}
}

func TestGetSessionInfo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "checkpointer-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cm, err := NewCheckpointerManager(tmpDir)
	if err != nil {
		t.Fatalf("NewCheckpointerManager failed: %v", err)
	}
	defer cm.Close()

	ctx := context.Background()

	// Non-existent session should return nil
	info, err := cm.GetSessionInfo(ctx, "non-existent")
	if err != nil {
		t.Fatalf("GetSessionInfo for non-existent session failed: %v", err)
	}
	if info != nil {
		t.Errorf("Expected nil for non-existent session, got %+v", info)
	}

	// Create a session
	threadID := "test-thread-info"
	if err := cm.TouchSession(ctx, threadID); err != nil {
		t.Fatalf("TouchSession failed: %v", err)
	}

	info, err = cm.GetSessionInfo(ctx, threadID)
	if err != nil {
		t.Fatalf("GetSessionInfo failed: %v", err)
	}
	if info == nil {
		t.Fatal("Expected session info, got nil")
	}
	if info.ThreadID != threadID {
		t.Errorf("ThreadID = %q, want %q", info.ThreadID, threadID)
	}
	if info.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
	if info.LastActive.IsZero() {
		t.Error("LastActive should not be zero")
	}
}

func TestGetStats(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "checkpointer-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cm, err := NewCheckpointerManager(tmpDir)
	if err != nil {
		t.Fatalf("NewCheckpointerManager failed: %v", err)
	}
	defer cm.Close()

	ctx := context.Background()

	// Empty database
	stats, err := cm.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if stats.TotalSessions != 0 {
		t.Errorf("TotalSessions = %d, want 0", stats.TotalSessions)
	}

	// Add sessions
	for i := 0; i < 3; i++ {
		if err := cm.TouchSession(ctx, "thread-stats-"+string(rune('a'+i))); err != nil {
			t.Fatalf("TouchSession failed: %v", err)
		}
	}

	stats, err = cm.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats after adding sessions failed: %v", err)
	}
	if stats.TotalSessions != 3 {
		t.Errorf("TotalSessions = %d, want 3", stats.TotalSessions)
	}
	if stats.TotalCheckpoints != 3 {
		t.Errorf("TotalCheckpoints = %d, want 3", stats.TotalCheckpoints)
	}
	if stats.OldestSession == nil {
		t.Error("OldestSession should not be nil")
	}
	if stats.NewestSession == nil {
		t.Error("NewestSession should not be nil")
	}
}

func TestClearSession(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "checkpointer-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cm, err := NewCheckpointerManager(tmpDir)
	if err != nil {
		t.Fatalf("NewCheckpointerManager failed: %v", err)
	}
	defer cm.Close()

	ctx := context.Background()
	threadID := "thread-to-clear"

	// Create a session
	if err := cm.TouchSession(ctx, threadID); err != nil {
		t.Fatalf("TouchSession failed: %v", err)
	}

	// Verify it exists
	info, _ := cm.GetSessionInfo(ctx, threadID)
	if info == nil {
		t.Fatal("Session should exist before clear")
	}

	// Clear the session
	if err := cm.ClearSession(ctx, threadID); err != nil {
		t.Fatalf("ClearSession failed: %v", err)
	}

	// Verify it's gone
	info, _ = cm.GetSessionInfo(ctx, threadID)
	if info != nil {
		t.Error("Session should be nil after clear")
	}
}

func TestCleanupExpiredSessions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "checkpointer-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cm, err := NewCheckpointerManager(tmpDir)
	if err != nil {
		t.Fatalf("NewCheckpointerManager failed: %v", err)
	}
	defer cm.Close()

	ctx := context.Background()

	// Set a very short max age for testing
	cm.SetMaxAge(100 * time.Millisecond)

	// Create a session
	threadID := "thread-to-expire"
	if err := cm.TouchSession(ctx, threadID); err != nil {
		t.Fatalf("TouchSession failed: %v", err)
	}

	// Wait for it to expire
	time.Sleep(200 * time.Millisecond)

	// Run cleanup
	count, err := cm.CleanupExpiredSessions(ctx)
	if err != nil {
		t.Fatalf("CleanupExpiredSessions failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 session cleaned up, got %d", count)
	}

	// Verify session is gone
	info, _ := cm.GetSessionInfo(ctx, threadID)
	if info != nil {
		t.Error("Expired session should be nil after cleanup")
	}
}

func TestSetMaxAge(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "checkpointer-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cm, err := NewCheckpointerManager(tmpDir)
	if err != nil {
		t.Fatalf("NewCheckpointerManager failed: %v", err)
	}
	defer cm.Close()

	// Default max age
	if cm.maxAge != DefaultSessionMaxAge {
		t.Errorf("Default maxAge = %v, want %v", cm.maxAge, DefaultSessionMaxAge)
	}

	// Set new max age
	newMaxAge := 48 * time.Hour
	cm.SetMaxAge(newMaxAge)
	if cm.maxAge != newMaxAge {
		t.Errorf("After SetMaxAge, maxAge = %v, want %v", cm.maxAge, newMaxAge)
	}
}

func TestSetMaxCheckpoints(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "checkpointer-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cm, err := NewCheckpointerManager(tmpDir)
	if err != nil {
		t.Fatalf("NewCheckpointerManager failed: %v", err)
	}
	defer cm.Close()

	// Default max checkpoints
	if cm.maxCheckpoints != DefaultMaxCheckpoints {
		t.Errorf("Default maxCheckpoints = %d, want %d", cm.maxCheckpoints, DefaultMaxCheckpoints)
	}

	// Set new max checkpoints
	newMax := 50
	cm.SetMaxCheckpoints(newMax)
	if cm.maxCheckpoints != newMax {
		t.Errorf("After SetMaxCheckpoints, maxCheckpoints = %d, want %d", cm.maxCheckpoints, newMax)
	}
}

func TestGetStore(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "checkpointer-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cm, err := NewCheckpointerManager(tmpDir)
	if err != nil {
		t.Fatalf("NewCheckpointerManager failed: %v", err)
	}
	defer cm.Close()

	store := cm.GetStore()
	if store == nil {
		t.Error("GetStore should not return nil")
	}
}

func TestSessionInfoFields(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "checkpointer-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cm, err := NewCheckpointerManager(tmpDir)
	if err != nil {
		t.Fatalf("NewCheckpointerManager failed: %v", err)
	}
	defer cm.Close()

	ctx := context.Background()
	threadID := "thread-fields-test"

	// Create session with multiple touches
	if err := cm.TouchSession(ctx, threadID); err != nil {
		t.Fatalf("First TouchSession failed: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	if err := cm.TouchSession(ctx, threadID); err != nil {
		t.Fatalf("Second TouchSession failed: %v", err)
	}

	info, err := cm.GetSessionInfo(ctx, threadID)
	if err != nil {
		t.Fatalf("GetSessionInfo failed: %v", err)
	}

	// Verify LastActive is after CreatedAt (they should be different due to sleep)
	if !info.LastActive.After(info.CreatedAt) && info.LastActive != info.CreatedAt {
		t.Errorf("LastActive (%v) should be >= CreatedAt (%v)", info.LastActive, info.CreatedAt)
	}

	// Verify checkpoint count is 2
	if info.CheckpointCount != 2 {
		t.Errorf("CheckpointCount = %d, want 2", info.CheckpointCount)
	}
}
