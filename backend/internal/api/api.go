package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"chat-agent/internal/agent"
	"chat-agent/internal/logger"

	"github.com/gorilla/mux"
)

// Handler API 处理器
type Handler struct {
	agent *agent.Agent
	router *mux.Router
}

// NewHandler 创建处理器
func NewHandler(a *agent.Agent) *Handler {
	h := &Handler{
		agent:  a,
		router: mux.NewRouter(),
	}
	h.setupRoutes()
	// 应用日志中间件到整个路由器
	h.router.Use(logMiddleware)
	// 应用超时中间件
	h.router.Use(timeoutMiddleware(60 * time.Second))
	return h
}

// setupRoutes 设置路由
func (h *Handler) setupRoutes() {
	// 健康检查
	h.router.HandleFunc("/health", h.handleHealth).Methods("GET")

	// API 路由
	api := h.router.PathPrefix("/api").Subrouter()

	api.HandleFunc("/chat", h.handleChat).Methods("POST")
	api.HandleFunc("/sessions", h.handleListSessions).Methods("GET")
	api.HandleFunc("/sessions/{id}", h.handleGetSession).Methods("GET")
	api.HandleFunc("/sessions/{id}/history", h.handleGetHistory).Methods("GET")
	api.HandleFunc("/sessions/{id}", h.handleDeleteSession).Methods("DELETE")
}

func (h *Handler) SetupStaticRoutes() {
	h.router.PathPrefix("/").Handler(http.FileServer(http.Dir("./frontend/dist")))
}

// GetRouter 获取路由器
func (h *Handler) GetRouter() *mux.Router {
	return h.router
}

// ChatRequest 聊天请求
type ChatRequest struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

// ChatResponse 聊天响应（SSE）
func (h *Handler) handleChat(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "无效的请求体", http.StatusBadRequest)
		return
	}

	// 记录请求日志，包括 prompt
	clientIP := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		clientIP = forwarded
	}
	logger.Info("聊天请求",
		logger.WithField("client_ip", clientIP),
		logger.WithField("session_id", req.SessionID),
		logger.WithField("message", req.Message),
	)

	// 验证必填字段
	if req.Message == "" {
		http.Error(w, "message 不能为空", http.StatusBadRequest)
		return
	}

	// 如果没有 session_id，创建新会话
	if req.SessionID == "" {
		session := h.agent.CreateSession()
		req.SessionID = session.ID
	}

	// 验证会话是否存在
	if _, ok := h.agent.GetSession(req.SessionID); !ok {
		http.Error(w, "会话不存在", http.StatusNotFound)
		return
	}

	// 设置 SSE 响应头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 获取 http.Flusher
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "不支持流式响应", http.StatusInternalServerError)
		return
	}

	// 发送初始会话 ID
	sessionData, _ := json.Marshal(map[string]string{"session_id": req.SessionID})
	fmt.Fprintf(w, "event: session\ndata: %s\n\n", sessionData)
	flusher.Flush()

	// 处理聊天
	ctx := r.Context()
	err := h.agent.Chat(ctx, req.SessionID, req.Message, func(event agent.StreamEvent) {
		data, err := json.Marshal(event)
		if err != nil {
			logger.Error("序列化事件失败",
		logger.WithField("error", err.Error()),
	)
			return
		}
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
		flusher.Flush()
	})

	if err != nil {
		logger.Error("聊天处理失败",
		logger.WithField("session_id", req.SessionID),
		logger.WithField("error", err.Error()),
	)
	}
}

// handleListSessions 列出会话
func (h *Handler) handleListSessions(w http.ResponseWriter, r *http.Request) {
	sessions := h.agent.ListSessions()
	jsonResponse(w, sessions)
}

// handleHealth 健康检查
func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleGetSession 获取会话详情
func (h *Handler) handleGetSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	session, ok := h.agent.GetSession(sessionID)
	if !ok {
		http.Error(w, "会话不存在", http.StatusNotFound)
		return
	}

	jsonResponse(w, session)
}

// handleGetHistory 获取会话历史
func (h *Handler) handleGetHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	messages := h.agent.GetSessionHistory(sessionID)
	jsonResponse(w, messages)
}

// handleDeleteSession 删除会话
func (h *Handler) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	h.agent.DeleteSession(sessionID)
	w.WriteHeader(http.StatusNoContent)
}

// jsonResponse 发送 JSON 响应
func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// logMiddleware 请求日志中间件
func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Info("请求开始",
			logger.WithField("method", r.Method),
			logger.WithField("path", r.URL.Path),
			logger.WithField("remote_addr", r.RemoteAddr),
		)

		next.ServeHTTP(w, r)
	})
}

// timeoutMiddleware 请求超时中间件
func timeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			done := make(chan struct{})
			go func() {
				next.ServeHTTP(w, r)
				close(done)
			}()

			select {
			case <-done:
				return
			case <-ctx.Done():
				logger.Warn("请求超时",
					logger.WithField("method", r.Method),
					logger.WithField("path", r.URL.Path),
					logger.WithField("timeout", timeout),
				)
				http.Error(w, "请求超时", http.StatusRequestTimeout)
			}
		})
	}
}
