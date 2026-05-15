package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"chat-agent/internal/agent"
	"chat-agent/internal/logger"
	"chat-agent/internal/store"
)

func init() {
	logger.Init(false)
}

func TestChatRequest_JSON(t *testing.T) {
	req := ChatRequest{
		SessionID: "sess_1",
		Message:  "你好",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	var parsed ChatRequest
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if parsed.SessionID != "sess_1" {
		t.Errorf("expected sess_1, got %s", parsed.SessionID)
	}
	if parsed.Message != "你好" {
		t.Errorf("expected 你好, got %s", parsed.Message)
	}
}

func TestLogMiddleware(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := logMiddleware(nextHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestHandler_NewHandler(t *testing.T) {
	memoryStore := store.NewMemoryStore()
	memoryStore.CreateSession("sess_1", "测试会话")

	a := agent.NewAgent(nil, memoryStore)
	h := NewHandler(a)

	if h == nil {
		t.Error("expected handler to be created")
	}
	if h.router == nil {
		t.Error("expected router to be set")
	}
}