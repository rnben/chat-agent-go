package store

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStore SQLite 存储
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore 创建 SQLite 存储
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil {
		return nil, err
	}

	// 创建表
	schema := `
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		title TEXT DEFAULT '新会话',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL,
		role TEXT NOT NULL,
		content TEXT DEFAULT '',
		tool_calls TEXT DEFAULT '',
		tool_result TEXT DEFAULT '',
		tool_call_id TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_id);
	`

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}

	return &SQLiteStore{db: db}, nil
}

// CreateSession 创建会话
func (s *SQLiteStore) CreateSession(id, title string) *Session {
	now := time.Now()
	_, _ = s.db.Exec(
		"INSERT INTO sessions (id, title, created_at, updated_at) VALUES (?, ?, ?, ?)",
		id, title, now, now,
	)
	return &Session{
		ID:        id,
		Title:     title,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// GetSession 获取会话
func (s *SQLiteStore) GetSession(id string) (*Session, bool) {
	var session Session
	err := s.db.QueryRow(
		"SELECT id, title, created_at, updated_at FROM sessions WHERE id = ?", id,
	).Scan(&session.ID, &session.Title, &session.CreatedAt, &session.UpdatedAt)

	if err != nil {
		return nil, false
	}
	return &session, true
}

// ListSessions 列出所有会话
func (s *SQLiteStore) ListSessions() []*Session {
	rows, err := s.db.Query(
		"SELECT id, title, created_at, updated_at FROM sessions ORDER BY updated_at DESC",
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		var session Session
		if err := rows.Scan(&session.ID, &session.Title, &session.CreatedAt, &session.UpdatedAt); err == nil {
			sessions = append(sessions, &session)
		}
	}
	return sessions
}

// DeleteSession 删除会话
func (s *SQLiteStore) DeleteSession(id string) {
	s.db.Exec("DELETE FROM messages WHERE session_id = ?", id)
	s.db.Exec("DELETE FROM sessions WHERE id = ?", id)
}

// AddMessage 添加消息
func (s *SQLiteStore) AddMessage(msg *Message) {
	// 插入消息
	_, _ = s.db.Exec(
		"INSERT INTO messages (session_id, role, content, tool_calls, tool_result, tool_call_id, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		msg.SessionID, msg.Role, msg.Content, "", msg.ToolResult, msg.ToolCallID, time.Now(),
	)

	// 更新会话时间
	s.db.Exec("UPDATE sessions SET updated_at = ? WHERE id = ?", time.Now(), msg.SessionID)

	// 如果是第一条用户消息，设置会话标题
	if msg.Role == "user" {
		var count int
		s.db.QueryRow("SELECT COUNT(*) FROM messages WHERE session_id = ?", msg.SessionID).Scan(&count)
		if count == 1 {
			title := msg.Content
			if len(title) > 20 {
				title = title[:20] + "..."
			}
			s.db.Exec("UPDATE sessions SET title = ? WHERE id = ? AND title = '新会话'", title, msg.SessionID)
		}
	}
}

// GetMessages 获取会话消息历史
func (s *SQLiteStore) GetMessages(sessionID string) []*Message {
	rows, err := s.db.Query(
		"SELECT role, content, tool_result FROM messages WHERE session_id = ? ORDER BY created_at", sessionID,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		var msg Message
		msg.SessionID = sessionID
		if err := rows.Scan(&msg.Role, &msg.Content, &msg.ToolResult); err == nil {
			messages = append(messages, &msg)
		}
	}
	return messages
}

// UpdateSessionTitle 更新会话标题
func (s *SQLiteStore) UpdateSessionTitle(id, title string) {
	s.db.Exec("UPDATE sessions SET title = ?, updated_at = ? WHERE id = ?", title, time.Now(), id)
}

// Close 关闭数据库连接
func (s *SQLiteStore) Close() {
	if s.db != nil {
		s.db.Close()
	}
}
