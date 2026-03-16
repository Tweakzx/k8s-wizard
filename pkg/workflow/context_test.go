package workflow

import (
	"testing"
	"time"
)

func TestContextManager(t *testing.T) {
	// Test requires CheckpointerManager, will create fake
	manager, _ := NewContextManager(nil)

	entry := ConversationEntry{
		Role:      "user",
		Content:   "test message",
		Action:    nil,
		Timestamp: time.Now(),
	}

	err := manager.AddEntry("thread-1", entry)
	if err != nil {
		t.Fatalf("AddEntry failed: %v", err)
	}

	retrieved := manager.Get("thread-1")
	if retrieved == nil {
		t.Errorf("context should be created after AddEntry")
	}

	if len(retrieved.History) != 1 {
		t.Errorf("context should contain added entry, got %d entries", len(retrieved.History))
	}
}

func TestContextManager_Get(t *testing.T) {
	manager, _ := NewContextManager(nil)

	// First call creates context
	ctx1 := manager.Get("thread-new")
	if ctx1 == nil {
		t.Fatal("Get should create new context")
	}
	if ctx1.ThreadID != "thread-new" {
		t.Errorf("Expected thread ID 'thread-new', got '%s'", ctx1.ThreadID)
	}

	// Second call returns same context
	ctx2 := manager.Get("thread-new")
	if ctx1 != ctx2 {
		t.Error("Get should return same context instance")
	}
}

func TestContextManager_AddEntry(t *testing.T) {
	manager, _ := NewContextManager(nil)

	entry := ConversationEntry{
		Role:      "user",
		Content:   "hello",
		Timestamp: time.Now(),
	}

	err := manager.AddEntry("thread-1", entry)
	if err != nil {
		t.Fatalf("AddEntry failed: %v", err)
	}
	ctx := manager.Get("thread-1")

	if len(ctx.History) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(ctx.History))
	}

	if ctx.History[0].Role != "user" {
		t.Errorf("Expected role 'user', got '%s'", ctx.History[0].Role)
	}

	if ctx.History[0].Content != "hello" {
		t.Errorf("Expected content 'hello', got '%s'", ctx.History[0].Content)
	}
}

func TestContextManager_AddEntry_WithAction(t *testing.T) {
	manager, _ := NewContextManager(nil)

	action := &K8sAction{
		Action:    "create",
		Resource:  "pod",
		Namespace: "default",
	}

	entry := ConversationEntry{
		Role:      "assistant",
		Content:   "Creating pod...",
		Action:    action,
		Timestamp: time.Now(),
	}

	err := manager.AddEntry("thread-1", entry)
	if err != nil {
		t.Fatalf("AddEntry failed: %v", err)
	}
	ctx := manager.Get("thread-1")

	if ctx.LastOperation == nil {
		t.Fatal("LastOperation should be set")
	}

	if ctx.LastOperation.Action != "create" {
		t.Errorf("Expected action 'create', got '%s'", ctx.LastOperation.Action)
	}

	if ctx.LastResource != "pod" {
		t.Errorf("Expected resource 'pod', got '%s'", ctx.LastResource)
	}

	if ctx.LastNamespace != "default" {
		t.Errorf("Expected namespace 'default', got '%s'", ctx.LastNamespace)
	}
}

func TestContextManager_GetContextString(t *testing.T) {
	manager, _ := NewContextManager(nil)

	// Add some entries
	now := time.Now()
	err := manager.AddEntry("thread-1", ConversationEntry{
		Role:      "user",
		Content:   "Create a pod",
		Timestamp: now,
	})
	if err != nil {
		t.Fatalf("AddEntry failed: %v", err)
	}

	err = manager.AddEntry("thread-1", ConversationEntry{
		Role:      "assistant",
		Content:   "I'll create a pod for you",
		Timestamp: now,
	})
	if err != nil {
		t.Fatalf("AddEntry failed: %v", err)
	}

	// Get context string
	ctxStr := manager.GetContextString("thread-1", 10)

	// Check that it contains expected content
	if ctxStr == "" {
		t.Fatal("GetContextString should not return empty string")
	}

	// Should contain history label
	if !containsString(ctxStr, "对话历史") {
		t.Error("GetContextString should contain '对话历史'")
	}

	// Should contain user message
	if !containsString(ctxStr, "用户") || !containsString(ctxStr, "Create a pod") {
		t.Error("GetContextString should contain user message")
	}

	// Should contain assistant message
	if !containsString(ctxStr, "助手") || !containsString(ctxStr, "I'll create a pod for you") {
		t.Error("GetContextString should contain assistant message")
	}
}

func TestContextManager_GetContextString_WithMaxHistory(t *testing.T) {
	manager, _ := NewContextManager(nil)

	// Add more entries than maxHistory
	now := time.Now()
	for i := 0; i < 10; i++ {
		err := manager.AddEntry("thread-1", ConversationEntry{
			Role:      "user",
			Content:   "Message",
			Timestamp: now,
		})
		if err != nil {
			t.Fatalf("AddEntry failed: %v", err)
		}
	}

	// Get context string with maxHistory=3
	ctxStr := manager.GetContextString("thread-1", 3)

	// Count occurrences of "用户:"
	userCount := countOccurrences(ctxStr, "用户:")
	if userCount != 3 {
		t.Errorf("Expected 3 user messages with maxHistory=3, got %d", userCount)
	}
}

func TestContextManager_GetContextString_WithAction(t *testing.T) {
	manager, _ := NewContextManager(nil)

	action := &K8sAction{
		Action:    "create",
		Resource:  "deployment",
		Namespace: "default",
	}

	err := manager.AddEntry("thread-1", ConversationEntry{
		Role:      "assistant",
		Content:   "Creating deployment",
		Action:    action,
		Timestamp: time.Now(),
	})
	if err != nil {
		t.Fatalf("AddEntry failed: %v", err)
	}

	ctxStr := manager.GetContextString("thread-1", 10)

	// Should contain recent operation
	if !containsString(ctxStr, "最近操作") {
		t.Error("GetContextString should contain '最近操作'")
	}

	if !containsString(ctxStr, "create") || !containsString(ctxStr, "deployment") || !containsString(ctxStr, "default") {
		t.Error("GetContextString should contain action details")
	}
}

func TestContextManager_Clear(t *testing.T) {
	manager, _ := NewContextManager(nil)

	// Add some data
	err := manager.AddEntry("thread-1", ConversationEntry{
		Role:      "user",
		Content:   "test",
		Timestamp: time.Now(),
	})
	if err != nil {
		t.Fatalf("AddEntry failed: %v", err)
	}

	// Verify it exists
	if manager.Get("thread-1") == nil {
		t.Fatal("Context should exist before clear")
	}

	// Clear it
	err = manager.Clear("thread-1")
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	// Verify it's gone - use HasContext to check without creating
	if manager.HasContext("thread-1") {
		t.Fatal("Context should not exist after clear")
	}

	// Verify that Get creates a new empty context
	newCtx := manager.Get("thread-1")
	if newCtx == nil {
		t.Fatal("Get should create new context after clear")
	}
	if len(newCtx.History) != 0 {
		t.Errorf("New context should be empty, got %d entries", len(newCtx.History))
	}
}

func TestContextManager_MultipleThreads(t *testing.T) {
	manager, _ := NewContextManager(nil)

	err := manager.AddEntry("thread-1", ConversationEntry{
		Role:      "user",
		Content:   "thread 1 message",
		Timestamp: time.Now(),
	})
	if err != nil {
		t.Fatalf("AddEntry failed: %v", err)
	}

	err = manager.AddEntry("thread-2", ConversationEntry{
		Role:      "user",
		Content:   "thread 2 message",
		Timestamp: time.Now(),
	})
	if err != nil {
		t.Fatalf("AddEntry failed: %v", err)
	}

	ctx1 := manager.Get("thread-1")
	ctx2 := manager.Get("thread-2")

	if len(ctx1.History) != 1 {
		t.Errorf("Thread 1 should have 1 entry, got %d", len(ctx1.History))
	}

	if len(ctx2.History) != 1 {
		t.Errorf("Thread 2 should have 1 entry, got %d", len(ctx2.History))
	}

	if ctx1.History[0].Content != "thread 1 message" {
		t.Errorf("Thread 1 content mismatch")
	}

	if ctx2.History[0].Content != "thread 2 message" {
		t.Errorf("Thread 2 content mismatch")
	}
}

func TestContextManager_AddEntry_EmptyThreadID(t *testing.T) {
	manager, _ := NewContextManager(nil)

	entry := ConversationEntry{
		Role:      "user",
		Content:   "test",
		Timestamp: time.Now(),
	}

	err := manager.AddEntry("", entry)
	if err == nil {
		t.Error("AddEntry should return error for empty threadID")
	}

	expectedErr := "threadID cannot be empty"
	if err.Error() != expectedErr {
		t.Errorf("Expected error '%s', got '%s'", expectedErr, err.Error())
	}
}

func TestContextManager_AddEntry_InvalidRole(t *testing.T) {
	manager, _ := NewContextManager(nil)

	entry := ConversationEntry{
		Role:      "invalid",
		Content:   "test",
		Timestamp: time.Now(),
	}

	err := manager.AddEntry("thread-1", entry)
	if err == nil {
		t.Error("AddEntry should return error for invalid role")
	}

	if !containsString(err.Error(), "invalid role") {
		t.Errorf("Expected error about invalid role, got '%s'", err.Error())
	}
}

func TestContextManager_AddEntry_ValidRoles(t *testing.T) {
	manager, _ := NewContextManager(nil)

	// Test "user" role
	err := manager.AddEntry("thread-1", ConversationEntry{
		Role:      "user",
		Content:   "test",
		Timestamp: time.Now(),
	})
	if err != nil {
		t.Errorf("AddEntry with 'user' role should succeed, got error: %v", err)
	}

	// Test "assistant" role
	err = manager.AddEntry("thread-1", ConversationEntry{
		Role:      "assistant",
		Content:   "test",
		Timestamp: time.Now(),
	})
	if err != nil {
		t.Errorf("AddEntry with 'assistant' role should succeed, got error: %v", err)
	}
}

func TestConversationEntry_Actions(t *testing.T) {
	action := &K8sAction{
		Action:    "delete",
		Resource:  "pod",
		Name:      "test-pod",
		Namespace: "kube-system",
	}

	entry := ConversationEntry{
		Role:      "assistant",
		Content:   "Deleting pod",
		Action:    action,
		Timestamp: time.Now(),
	}

	if entry.Action == nil {
		t.Fatal("Action should be set")
	}

	if entry.Action.Action != "delete" {
		t.Errorf("Expected action 'delete', got '%s'", entry.Action.Action)
	}
}

func TestDependenciesWithContextManager(t *testing.T) {
	mgr, err := NewContextManager(nil)
	if err != nil {
		t.Fatalf("NewContextManager failed: %v", err)
	}

	deps := Dependencies{
		ContextMgr: mgr,
	}

	if deps.ContextMgr == nil {
		t.Errorf("ContextMgr should not be nil in Dependencies")
	}
}

func TestContextManagerPersistence(t *testing.T) {
	// Create temp checkpointer for persistence testing
	checkpointer, err := NewCheckpointerManager(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create checkpointer: %v", err)
	}
	defer checkpointer.Close()

	manager, err := NewContextManager(checkpointer)
	if err != nil {
		t.Fatalf("NewContextManager failed: %v", err)
	}

	entry := ConversationEntry{
		Role:      "user",
		Content:   "test message",
		Action:    nil,
		Timestamp: time.Now(),
	}

	// Add entry to context
	err = manager.AddEntry("thread-1", entry)
	if err != nil {
		t.Fatalf("AddEntry failed: %v", err)
	}

	// Verify context exists and has the entry
	retrieved := manager.Get("thread-1")
	if retrieved == nil {
		t.Errorf("context should be retrieved after AddEntry")
	}

	if len(retrieved.History) != 1 {
		t.Errorf("context should contain 1 entry after AddEntry, got %d", len(retrieved.History))
	}

	// Clear should persist data to checkpoint but remove from memory
	err = manager.Clear("thread-1")
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	// Verify context is removed from memory
	if manager.HasContext("thread-1") {
		t.Errorf("context should not exist in memory after Clear")
	}

	// Get should reload persisted data from checkpoint
	reloadedContext := manager.Get("thread-1")
	if reloadedContext == nil {
		t.Errorf("Get should reload context from checkpoint after Clear")
	}

	if len(reloadedContext.History) != 1 {
		t.Errorf("reloaded context should contain 1 entry (data should persist after Clear), got %d entries", len(reloadedContext.History))
	} else if reloadedContext.History[0].Content != "test message" {
		t.Errorf("reloaded context should preserve original content, got '%s'", reloadedContext.History[0].Content)
	}
}


// Helper functions
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) >= 0
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func countOccurrences(s, substr string) int {
	count := 0
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			count++
		}
	}
	return count
}
