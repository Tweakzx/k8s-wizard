package workflow

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	lgg "github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/store/sqlite"
)

const (
	DefaultSessionMaxAge   = 24 * time.Hour
	DefaultCleanupInterval = 1 * time.Hour
	DefaultMaxCheckpoints  = 100
)

type SessionInfo struct {
	ThreadID        string
	CreatedAt       time.Time
	LastActive      time.Time
	CheckpointCount int
}

type SessionStats struct {
	TotalSessions    int
	TotalCheckpoints int
	OldestSession    *time.Time
	NewestSession    *time.Time
}

type CheckpointerManager struct {
	store          *sqlite.SqliteCheckpointStore
	db             *sql.DB
	dataDir        string
	maxAge         time.Duration
	maxCheckpoints int
	mu             sync.RWMutex
	stopCleanup    chan struct{}
}

func NewCheckpointerManager(dataDir string) (*CheckpointerManager, error) {
	if dataDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		dataDir = filepath.Join(homeDir, ".k8s-wizard", "checkpoints")
	}

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

	db, err := sql.Open("sqlite3", dbPath+"?_loc=auto")
	if err != nil {
		store.Close()
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	m := &CheckpointerManager{
		store:          store,
		db:             db,
		dataDir:        dataDir,
		maxAge:         DefaultSessionMaxAge,
		maxCheckpoints: DefaultMaxCheckpoints,
		stopCleanup:    make(chan struct{}),
	}

	if err := m.ensureSessionTable(); err != nil {
		m.Close()
		return nil, fmt.Errorf("failed to ensure session table: %w", err)
	}

	go m.startCleanupWorker()

	return m, nil
}

func (m *CheckpointerManager) ensureSessionTable() error {
	_, err := m.db.Exec(`
		CREATE TABLE IF NOT EXISTS session_metadata (
			thread_id TEXT PRIMARY KEY,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_active DATETIME DEFAULT CURRENT_TIMESTAMP,
			checkpoint_count INTEGER DEFAULT 0
		);
		CREATE INDEX IF NOT EXISTS idx_session_last_active ON session_metadata(last_active);
		CREATE INDEX IF NOT EXISTS idx_session_created_at ON session_metadata(created_at);
	`)
	return err
}

func (m *CheckpointerManager) GetStore() lgg.CheckpointStore {
	return m.store
}

func (m *CheckpointerManager) Close() error {
	close(m.stopCleanup)
	if m.db != nil {
		m.db.Close()
	}
	return m.store.Close()
}

func (m *CheckpointerManager) ClearSession(ctx context.Context, threadID string) error {
	if err := m.store.Clear(ctx, threadID); err != nil {
		return err
	}
	_, err := m.db.ExecContext(ctx, "DELETE FROM session_metadata WHERE thread_id = ?", threadID)
	return err
}

func (m *CheckpointerManager) ListSessions(ctx context.Context) ([]SessionInfo, error) {
	rows, err := m.db.QueryContext(ctx, `
		SELECT thread_id, created_at, last_active, checkpoint_count 
		FROM session_metadata 
		ORDER BY last_active DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []SessionInfo
	for rows.Next() {
		var si SessionInfo
		var createdAt, lastActive sql.NullTime
		if err := rows.Scan(&si.ThreadID, &createdAt, &lastActive, &si.CheckpointCount); err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		if createdAt.Valid {
			si.CreatedAt = createdAt.Time
		}
		if lastActive.Valid {
			si.LastActive = lastActive.Time
		}
		sessions = append(sessions, si)
	}
	return sessions, rows.Err()
}

func (m *CheckpointerManager) GetSessionInfo(ctx context.Context, threadID string) (*SessionInfo, error) {
	var si SessionInfo
	var createdAt, lastActive sql.NullTime
	err := m.db.QueryRowContext(ctx, `
		SELECT thread_id, created_at, last_active, checkpoint_count 
		FROM session_metadata 
		WHERE thread_id = ?
	`, threadID).Scan(&si.ThreadID, &createdAt, &lastActive, &si.CheckpointCount)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session info: %w", err)
	}
	if createdAt.Valid {
		si.CreatedAt = createdAt.Time
	}
	if lastActive.Valid {
		si.LastActive = lastActive.Time
	}
	return &si, nil
}

func (m *CheckpointerManager) GetStats(ctx context.Context) (*SessionStats, error) {
	var stats SessionStats
	var oldestStr, newestStr sql.NullString
	err := m.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(*) as total_sessions,
			COALESCE(SUM(checkpoint_count), 0) as total_checkpoints,
			MIN(created_at) as oldest_session,
			MAX(created_at) as newest_session
		FROM session_metadata
	`).Scan(&stats.TotalSessions, &stats.TotalCheckpoints, &oldestStr, &newestStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}
	if oldestStr.Valid && oldestStr.String != "" {
		t, parseErr := time.Parse("2006-01-02 15:04:05-07:00", oldestStr.String)
		if parseErr != nil {
			t, parseErr = time.Parse("2006-01-02 15:04:05", oldestStr.String)
		}
		if parseErr == nil {
			stats.OldestSession = &t
		}
	}
	if newestStr.Valid && newestStr.String != "" {
		t, parseErr := time.Parse("2006-01-02 15:04:05-07:00", newestStr.String)
		if parseErr != nil {
			t, parseErr = time.Parse("2006-01-02 15:04:05", newestStr.String)
		}
		if parseErr == nil {
			stats.NewestSession = &t
		}
	}
	return &stats, nil
}

func (m *CheckpointerManager) TouchSession(ctx context.Context, threadID string) error {
	now := time.Now()
	result, err := m.db.ExecContext(ctx, `
		INSERT INTO session_metadata (thread_id, created_at, last_active, checkpoint_count)
		VALUES (?, ?, ?, 1)
		ON CONFLICT(thread_id) DO UPDATE SET 
			last_active = ?,
			checkpoint_count = checkpoint_count + 1
	`, threadID, now, now, now)
	if err != nil {
		return fmt.Errorf("failed to touch session: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("failed to update session metadata")
	}
	return nil
}

func (m *CheckpointerManager) SetMaxAge(maxAge time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.maxAge = maxAge
}

func (m *CheckpointerManager) SetMaxCheckpoints(max int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.maxCheckpoints = max
}

func (m *CheckpointerManager) CleanupExpiredSessions(ctx context.Context) (int, error) {
	m.mu.RLock()
	cutoff := time.Now().Add(-m.maxAge)
	m.mu.RUnlock()

	result, err := m.db.ExecContext(ctx, `
		DELETE FROM session_metadata WHERE last_active < ?
	`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}

	affected, _ := result.RowsAffected()

	threads, _ := m.db.QueryContext(ctx, `
		SELECT DISTINCT thread_id FROM checkpoints WHERE thread_id IS NOT NULL
	`)
	if threads != nil {
		defer threads.Close()
		for threads.Next() {
			var tid string
			if err := threads.Scan(&tid); err != nil {
				continue
			}
			var count int
			m.db.QueryRowContext(ctx, `
				SELECT COUNT(*) FROM session_metadata WHERE thread_id = ?
			`, tid).Scan(&count)
			if count == 0 {
				m.store.Clear(ctx, tid)
			}
		}
	}

	return int(affected), nil
}

func (m *CheckpointerManager) startCleanupWorker() {
	ticker := time.NewTicker(DefaultCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			m.CleanupExpiredSessions(ctx)
			cancel()
		case <-m.stopCleanup:
			return
		}
	}
}
