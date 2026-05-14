package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"chat-agent/internal/agent"

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
	return h
}

// setupRoutes 设置路由
func (h *Handler) setupRoutes() {
	// API 路由
	api := h.router.PathPrefix("/api").Subrouter()
	
	api.HandleFunc("/chat", h.handleChat).Methods("POST")
	api.HandleFunc("/sessions", h.handleListSessions).Methods("GET")
	api.HandleFunc("/sessions", h.handleCreateSession).Methods("POST")
	api.HandleFunc("/sessions/{id}", h.handleGetSession).Methods("GET")
	api.HandleFunc("/sessions/{id}/history", h.handleGetHistory).Methods("GET")
	api.HandleFunc("/sessions/{id}", h.handleDeleteSession).Methods("DELETE")

	// 前端静态文件
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
	log.Printf("[请求] 客户端IP: %s, 会话ID: %s, 用户消息: %s", clientIP, req.SessionID, req.Message)

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
			log.Printf("序列化事件失败: %v", err)
			return
		}
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
		flusher.Flush()
	})

	if err != nil {
		log.Printf("聊天处理失败: %v", err)
	}
}

// handleListSessions 列出会话
func (h *Handler) handleListSessions(w http.ResponseWriter, r *http.Request) {
	sessions := h.agent.ListSessions()
	jsonResponse(w, sessions)
}

// CreateSessionRequest 创建会话请求
type CreateSessionRequest struct {
	Title string `json:"title"`
}

// handleCreateSession 创建会话
func (h *Handler) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var req CreateSessionRequest
	json.NewDecoder(r.Body).Decode(&req)

	session := h.agent.CreateSession()
	if req.Title != "" {
		// 可以更新标题
	}

	w.WriteHeader(http.StatusCreated)
	jsonResponse(w, session)
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
		// 记录请求开始
		log.Printf("[请求开始] %s %s %s", r.Method, r.URL.Path, r.RemoteAddr)
		
		// 调用下一个处理器
		next.ServeHTTP(w, r)
		
		// 记录请求结束（可选，如果需要记录响应状态码，可以使用ResponseWriter包装）
		// log.Printf("[请求结束] %s %s", r.Method, r.URL.Path)
	})
}
