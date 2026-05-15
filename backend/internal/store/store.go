package store

import (
	"sync"
	"time"
)

// Store 存储接口
type Store interface {
	CreateSession(id, title string) *Session
	GetSession(id string) (*Session, bool)
	ListSessions() []*Session
	DeleteSession(id string)
	AddMessage(msg *Message)
	GetMessages(sessionID string) []*Message
	UpdateSessionTitle(id, title string)
}

// Session 会话
type Session struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Message 消息
type Message struct {
	ID         string     `json:"id"`
	SessionID  string     `json:"session_id"`
	Role       string     `json:"role"` // user, assistant, system, tool
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolResult string     `json:"tool_result,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"` // 工具调用ID，用于tool角色的消息
	CreatedAt  time.Time  `json:"created_at"`
}

// ToolCall 工具调用
type ToolCall struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ArgsJSON string `json:"args"`
}

// MemoryStore 内存存储
type MemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	messages map[string][]*Message // sessionID -> messages
}

// NewMemoryStore 创建内存存储
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		sessions: make(map[string]*Session),
		messages: make(map[string][]*Message),
	}
}

// CreateSession 创建会话
func (s *MemoryStore) CreateSession(id, title string) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	session := &Session{
		ID:        id,
		Title:     title,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.sessions[id] = session
	s.messages[id] = []*Message{}
	return session
}

// GetSession 获取会话
func (s *MemoryStore) GetSession(id string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[id]
	return session, ok
}

// ListSessions 列出所有会话
func (s *MemoryStore) ListSessions() []*Session {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessions := make([]*Session, 0, len(s.sessions))
	for _, session := range s.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}

// DeleteSession 删除会话
func (s *MemoryStore) DeleteSession(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
	delete(s.messages, id)
}

// AddMessage 添加消息
func (s *MemoryStore) AddMessage(msg *Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messages[msg.SessionID] = append(s.messages[msg.SessionID], msg)
	if session, ok := s.sessions[msg.SessionID]; ok {
		session.UpdatedAt = time.Now()
		// 如果是第一条用户消息，设置会话标题
		if msg.Role == "user" && session.Title == "新会话" && len(s.messages[msg.SessionID]) == 1 {
			if len(msg.Content) > 20 {
				session.Title = msg.Content[:20] + "..."
			} else {
				session.Title = msg.Content
			}
		}
	}
}

// GetMessages 获取会话消息历史
func (s *MemoryStore) GetMessages(sessionID string) []*Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.messages[sessionID]
}

// UpdateSessionTitle 更新会话标题
func (s *MemoryStore) UpdateSessionTitle(id, title string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if session, ok := s.sessions[id]; ok {
		session.Title = title
		session.UpdatedAt = time.Now()
	}
}
