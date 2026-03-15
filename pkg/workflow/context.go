package workflow

import (
	"context"
	"database/sql"
	"fmt"
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
func (m *ContextManager) AddEntry(threadID string, entry ConversationEntry) {
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
		m.saveToCheckpoint(threadID)
	}
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

// Clear removes conversation context.
func (m *ContextManager) Clear(threadID string) {
	delete(m.contexts, threadID)
	if m.checkpointer != nil {
		_ = m.checkpointer.ClearSession(context.Background(), threadID)
	}
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
			continue
		}

		// Parse timestamp
		entry.Timestamp, _ = time.Parse(time.RFC3339, timestampStr)

		// Parse action JSON if present
		if actionJSON.Valid && actionJSON.String != "" {
			// Simple JSON parsing - in production use proper JSON unmarshal
			entry.Action = parseActionFromJSON(actionJSON.String)
		}

		history = append(history, entry)
	}

	return history
}

func (m *ContextManager) saveToCheckpoint(threadID string) {
	if m.checkpointer == nil {
		return
	}

	ctx := m.contexts[threadID]
	if ctx == nil {
		return
	}

	// Ensure conversation_history table exists
	_, _ = m.checkpointer.db.Exec(`
		CREATE TABLE IF NOT EXISTS conversation_history (
			thread_id TEXT,
			role TEXT,
			content TEXT,
			timestamp TEXT,
			action_json TEXT,
			PRIMARY KEY (thread_id, role, timestamp)
		)
	`)

	// Save all entries
	for _, entry := range ctx.History {
		var actionJSON string
		if entry.Action != nil {
			actionJSON = fmt.Sprintf(`{"action":"%s","resource":"%s","namespace":"%s"}`,
				entry.Action.Action, entry.Action.Resource, entry.Action.Namespace)
		}

		_, err := m.checkpointer.db.Exec(`
			INSERT OR REPLACE INTO conversation_history
			(thread_id, role, content, timestamp, action_json)
			VALUES (?, ?, ?, ?, ?)
		`, threadID, entry.Role, entry.Content, entry.Timestamp.Format(time.RFC3339), actionJSON)
		if err != nil {
			// Log error but continue
			continue
		}
	}
}

// Helper function to parse action from JSON
func parseActionFromJSON(jsonStr string) *K8sAction {
	// Parse JSON string to reconstruct K8sAction
	// Using simple string parsing - can be improved with encoding/json if needed
	action := &K8sAction{
		Action:    "",
		Resource:  "",
		Namespace: "",
	}

	// Simple JSON field extraction
	if len(jsonStr) < 10 {
		return action
	}

	// Extract action field
	if idx := strings.Index(jsonStr, `"action":"`); idx >= 0 {
		endIdx := strings.Index(jsonStr[idx+10:], `"`)
		if endIdx >= 0 {
			action.Action = jsonStr[idx+10 : idx+10+endIdx]
		}
	}

	// Extract resource field
	if idx := strings.Index(jsonStr, `"resource":"`); idx >= 0 {
		endIdx := strings.Index(jsonStr[idx+12:], `"`)
		if endIdx >= 0 {
			action.Resource = jsonStr[idx+12 : idx+12+endIdx]
		}
	}

	// Extract namespace field
	if idx := strings.Index(jsonStr, `"namespace":"`); idx >= 0 {
		endIdx := strings.Index(jsonStr[idx+13:], `"`)
		if endIdx >= 0 {
			action.Namespace = jsonStr[idx+13 : idx+13+endIdx]
		}
	}

	return action
}
