package workflow

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// ConversationContext maintains conversation history and context.
type ConversationContext struct {
	ThreadID       string
	History        []ConversationEntry
	LastOperation  *K8sAction
	LastResource   string
	LastNamespace string
	Timestamp     time.Time
}

// ConversationEntry represents a single conversation turn.
type ConversationEntry struct {
	Role      string    // "user" or "assistant"
	Content   string
	Action    *K8sAction
	Timestamp time.Time
}

// ContextManager manages conversation contexts per thread.
type ContextManager struct {
	contexts    map[string]*ConversationContext
	checkpointer *CheckpointerManager
}

// NewContextManager creates a context manager.
func NewContextManager(checkpointer *CheckpointerManager) (*ContextManager, error) {
	return &ContextManager{
		contexts:    make(map[string]*ConversationContext),
		checkpointer: checkpointer,
	}, nil
}

// Get retrieves or creates a conversation context.
func (m *ContextManager) Get(threadID string) *ConversationContext {
	if ctx, exists := m.contexts[threadID]; exists {
		// Update timestamp
		ctx.Timestamp = time.Now()
		return ctx
	}

	// Try to load from checkpoint
	var history []ConversationEntry
	if m.checkpointer != nil {
		// Attempt to load saved history
		if saved := m.loadFromCheckpoint(threadID); saved != nil {
			history = saved
		}
	}

	ctx := &ConversationContext{
		ThreadID:  threadID,
		History:   history,
		Timestamp: time.Now(),
	}

	m.contexts[threadID] = ctx
	return ctx
}

// AddEntry adds an entry to conversation history.
func (m *ContextManager) AddEntry(threadID string, entry ConversationEntry) error {
	// Validate inputs
	if threadID == "" {
		return fmt.Errorf("threadID cannot be empty")
	}
	if entry.Role != "user" && entry.Role != "assistant" {
		return fmt.Errorf("invalid role: %s (must be 'user' or 'assistant')", entry.Role)
	}

	ctx := m.Get(threadID)
	ctx.History = append(ctx.History, entry)
	ctx.Timestamp = time.Now()

	// Track last operation for context
	if entry.Action != nil {
		ctx.LastOperation = entry.Action
		ctx.LastResource = entry.Action.Resource
		ctx.LastNamespace = entry.Action.Namespace
	}

	// Persist to checkpoint if available
	if m.checkpointer != nil {
		if err := m.saveToCheckpoint(threadID); err != nil {
			return fmt.Errorf("failed to save to checkpoint: %w", err)
		}
	}

	return nil
}

// GetContextString returns formatted context for LLM.
func (m *ContextManager) GetContextString(threadID string, maxHistory int) string {
	ctx := m.Get(threadID)

	var sb strings.Builder
	sb.WriteString("对话历史:\n")

	// Get recent history
	history := ctx.History
	if len(history) > maxHistory {
		history = history[len(history)-maxHistory:]
	}

	for _, entry := range history {
		role := "用户"
		if entry.Role == "assistant" {
			role = "助手"
		}
		sb.WriteString(fmt.Sprintf("  %s: %s\n", role, entry.Content))
	}

	// Add context from last operation
	if ctx.LastOperation != nil {
		sb.WriteString(fmt.Sprintf("\n最近操作: %s %s/%s\n",
			ctx.LastOperation.Action,
			ctx.LastNamespace,
			ctx.LastResource))
	}

	return sb.String()
}

// Clear removes conversation context from memory only.
// The checkpoint data is preserved and can be reloaded via Get().
func (m *ContextManager) Clear(threadID string) error {
	delete(m.contexts, threadID)
	return nil
}

// HasContext checks if a context exists without creating one.
func (m *ContextManager) HasContext(threadID string) bool {
	_, exists := m.contexts[threadID]
	return exists
}

func (m *ContextManager) loadFromCheckpoint(threadID string) []ConversationEntry {
	if m.checkpointer == nil {
		return nil
	}

	// Use database directly to store conversation history
	// We'll add a conversation_history table for this purpose
	rows, err := m.checkpointer.db.Query(`
		SELECT role, content, timestamp, action_json
		FROM conversation_history
		WHERE thread_id = ?
		ORDER BY timestamp ASC
	`, threadID)
	if err != nil {
		return nil // Table may not exist yet
	}
	defer rows.Close()

	var history []ConversationEntry
	for rows.Next() {
		var entry ConversationEntry
		var actionJSON sql.NullString
		var timestampStr string
		err := rows.Scan(&entry.Role, &entry.Content, &timestampStr, &actionJSON)
		if err != nil {
			log.Printf("ERROR: Failed to scan conversation entry for thread %s: %v", threadID, err)
			continue
		}

		// Parse timestamp with error handling
		entry.Timestamp, err = time.Parse(time.RFC3339, timestampStr)
		if err != nil {
			log.Printf("ERROR: Failed to parse timestamp '%s' for thread %s, using current time: %v", timestampStr, threadID, err)
			entry.Timestamp = time.Now()
		}

		// Parse action JSON if present
		if actionJSON.Valid && actionJSON.String != "" {
			// Simple JSON parsing - in production use proper JSON unmarshal
			entry.Action = parseActionFromJSON(actionJSON.String)
		}

		history = append(history, entry)
	}

	return history
}

func (m *ContextManager) saveToCheckpoint(threadID string) error {
	if m.checkpointer == nil {
		return nil
	}

	ctx := m.contexts[threadID]
	if ctx == nil {
		return fmt.Errorf("context not found for thread %s", threadID)
	}

	// Ensure conversation_history table exists with auto-increment ID to prevent timestamp collisions
	_, err := m.checkpointer.db.Exec(`
		CREATE TABLE IF NOT EXISTS conversation_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			thread_id TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			timestamp TEXT NOT NULL,
			action_json TEXT
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Save all entries
	for _, entry := range ctx.History {
		var actionJSON string
		if entry.Action != nil {
			// Serialize all 7 K8sAction fields using encoding/json
			actionBytes, err := json.Marshal(entry.Action)
			if err != nil {
				log.Printf("ERROR: Failed to serialize action for thread %s: %v", threadID, err)
				continue
			}
			actionJSON = string(actionBytes)
		}

		_, err := m.checkpointer.db.Exec(`
			INSERT INTO conversation_history
			(thread_id, role, content, timestamp, action_json)
			VALUES (?, ?, ?, ?, ?)
		`, threadID, entry.Role, entry.Content, entry.Timestamp.Format(time.RFC3339), actionJSON)
		if err != nil {
			return fmt.Errorf("failed to save conversation entry for thread %s: %w", threadID, err)
		}
	}

	return nil
}

// Helper function to parse action from JSON
func parseActionFromJSON(jsonStr string) *K8sAction {
	// Parse JSON string to reconstruct K8sAction with all 7 fields
	var action K8sAction
	if err := json.Unmarshal([]byte(jsonStr), &action); err != nil {
		log.Printf("ERROR: Failed to parse action JSON: %v", err)
		// Return empty action on error
		return &K8sAction{}
	}
	return &action
}
