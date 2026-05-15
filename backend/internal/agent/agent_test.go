package agent

import (
	"testing"

	"chat-agent/internal/logger"
	"chat-agent/internal/store"
)

func init() {
	// 初始化日志，避免 nil pointer
	logger.Init(false)
}

func TestAgent_CreateSession(t *testing.T) {
	memoryStore := store.NewMemoryStore()
	// 用 nil LLM 避免实际调用
	a := &Agent{
		store: memoryStore,
	}

	session := a.CreateSession()
	if session == nil {
		t.Error("expected session to be created")
	}
	if session.ID == "" {
		t.Error("expected session ID to be set")
	}
}

func TestAgent_GetSession(t *testing.T) {
	memoryStore := store.NewMemoryStore()
	memoryStore.CreateSession("sess_1", "测试会话")

	a := &Agent{
		store: memoryStore,
	}

	session, ok := a.GetSession("sess_1")
	if !ok {
		t.Error("expected session to exist")
	}
	if session.Title != "测试会话" {
		t.Errorf("expected title 测试会话, got %s", session.Title)
	}
}

func TestAgent_ListSessions(t *testing.T) {
	memoryStore := store.NewMemoryStore()
	memoryStore.CreateSession("sess_1", "会话1")
	memoryStore.CreateSession("sess_2", "会话2")

	a := &Agent{
		store: memoryStore,
	}

	sessions := a.ListSessions()
	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(sessions))
	}
}

func TestAgent_DeleteSession(t *testing.T) {
	memoryStore := store.NewMemoryStore()
	memoryStore.CreateSession("sess_1", "测试会话")

	a := &Agent{
		store: memoryStore,
	}

	a.DeleteSession("sess_1")

	_, ok := a.GetSession("sess_1")
	if ok {
		t.Error("expected session to be deleted")
	}
}

func TestAgent_GetSessionHistory(t *testing.T) {
	memoryStore := store.NewMemoryStore()
	memoryStore.CreateSession("sess_1", "测试会话")
	memoryStore.AddMessage(&store.Message{
		ID:        "msg_1",
		SessionID: "sess_1",
		Role:     "user",
		Content:  "你好",
	})
	memoryStore.AddMessage(&store.Message{
		ID:        "msg_2",
		SessionID: "sess_1",
		Role:     "assistant",
		Content:  "你好，有什么可以帮助？",
	})

	a := &Agent{
		store: memoryStore,
	}

	history := a.GetSessionHistory("sess_1")
	if len(history) != 2 {
		t.Errorf("expected 2 messages, got %d", len(history))
	}
}

func TestStreamEvent_JSON(t *testing.T) {
	event := StreamEvent{
		Type:    EventContent,
		Content: "Hello",
	}

	// 验证可以序列化
	_ = event
}

func TestStreamEvent_ToolCall(t *testing.T) {
	event := StreamEvent{
		Type: EventToolCall,
		Tool: &ToolCallEvent{
			ID:   "tool_1",
			Name: "query_order",
			Args: `{"order_id": "ORD-001"}`,
		},
	}

	_ = event
}