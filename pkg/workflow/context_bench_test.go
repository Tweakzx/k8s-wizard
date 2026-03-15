package workflow

import (
	"testing"
	"time"
)

func BenchmarkContextManagerAddEntry(b *testing.B) {
	mgr, err := NewContextManager(nil)
	if err != nil {
		b.Fatalf("failed to create context manager: %v", err)
	}

	entry := ConversationEntry{
		Role:      "user",
		Content:   "test message",
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mgr.AddEntry("test-thread", entry)
	}
}

func BenchmarkContextManagerGet(b *testing.B) {
	mgr, err := NewContextManager(nil)
	if err != nil {
		b.Fatalf("failed to create context manager: %v", err)
	}

	threadID := "benchmark-thread"

	// Pre-populate with 100 entries
	for i := 0; i < 100; i++ {
		mgr.AddEntry(threadID, ConversationEntry{
			Role:      "user",
			Content:   "test message",
			Timestamp: time.Now(),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mgr.Get(threadID)
	}
}

func BenchmarkContextManagerGetContextString(b *testing.B) {
	mgr, err := NewContextManager(nil)
	if err != nil {
		b.Fatalf("failed to create context manager: %v", err)
	}

	threadID := "benchmark-thread"

	// Pre-populate with 100 entries
	for i := 0; i < 100; i++ {
		mgr.AddEntry(threadID, ConversationEntry{
			Role:      "user",
			Content:   "test message",
			Timestamp: time.Now(),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mgr.GetContextString(threadID, 50)
	}
}
