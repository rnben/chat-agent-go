package store

import (
	"testing"
)

func TestMemoryStore_CreateSession(t *testing.T) {
	store := NewMemoryStore()
	session := store.CreateSession("sess_1", "测试会话")

	if session.ID != "sess_1" {
		t.Errorf("expected id sess_1, got %s", session.ID)
	}
	if session.Title != "测试会话" {
		t.Errorf("expected title 测试会话, got %s", session.Title)
	}
	if session.CreatedAt.IsZero() {
		t.Error("created_at should not be zero")
	}
}

func TestMemoryStore_GetSession(t *testing.T) {
	store := NewMemoryStore()
	store.CreateSession("sess_1", "测试会话")

	session, ok := store.GetSession("sess_1")
	if !ok {
		t.Error("expected session to exist")
	}
	if session.Title != "测试会话" {
		t.Errorf("expected title 测试会话, got %s", session.Title)
	}

	_, ok = store.GetSession("not_exist")
	if ok {
		t.Error("expected non-existent session to return false")
	}
}

func TestMemoryStore_ListSessions(t *testing.T) {
	store := NewMemoryStore()
	store.CreateSession("sess_1", "会话1")
	store.CreateSession("sess_2", "会话2")

	sessions := store.ListSessions()
	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(sessions))
	}
}

func TestMemoryStore_DeleteSession(t *testing.T) {
	store := NewMemoryStore()
	store.CreateSession("sess_1", "测试会话")

	store.DeleteSession("sess_1")

	_, ok := store.GetSession("sess_1")
	if ok {
		t.Error("expected session to be deleted")
	}
}

func TestMemoryStore_AddMessage(t *testing.T) {
	store := NewMemoryStore()
	store.CreateSession("sess_1", "新会话")

	msg := &Message{
		ID:        "msg_1",
		SessionID: "sess_1",
		Role:      "user",
		Content:   "你好",
	}
	store.AddMessage(msg)

	messages := store.GetMessages("sess_1")
	if len(messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(messages))
	}
	if messages[0].Content != "你好" {
		t.Errorf("expected content 你好, got %s", messages[0].Content)
	}
}

func TestMemoryStore_AddMessage_AutoTitle(t *testing.T) {
	store := NewMemoryStore()
	store.CreateSession("sess_1", "新会话")

	// 第一条用户消息应自动设置标题
	msg := &Message{
		ID:        "msg_1",
		SessionID: "sess_1",
		Role:      "user",
		Content:   "这是一个很长的第一条消息内容",
	}
	store.AddMessage(msg)

	session, _ := store.GetSession("sess_1")
	expected := "这是一个很长的第一条消息内容"[:20] + "..."
	if session.Title != expected {
		t.Errorf("expected title %s, got %s", expected, session.Title)
	}
}

func TestMemoryStore_Concurrency(t *testing.T) {
	store := NewMemoryStore()
	store.CreateSession("sess_1", "测试会话")

	done := make(chan bool)
	// 并发写入
	for i := 0; i < 100; i++ {
		go func(idx int) {
			msg := &Message{
				ID:        "msg_" + string(rune('0'+idx)),
				SessionID: "sess_1",
				Role:      "user",
				Content:   "消息",
			}
			store.AddMessage(msg)
			done <- true
		}(i)
	}

	// 等待所有 goroutine
	for i := 0; i < 100; i++ {
		<-done
	}

	messages := store.GetMessages("sess_1")
	if len(messages) != 100 {
		t.Errorf("expected 100 messages, got %d", len(messages))
	}
}