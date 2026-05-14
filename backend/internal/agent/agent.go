package agent

import (
	"context"
	"fmt"

	"chat-agent/internal/llm"
	"chat-agent/internal/store"
	"chat-agent/internal/tools"

	openai "github.com/sashabaranov/go-openai"
)

// Agent 对话代理
type Agent struct {
	llm        *llm.Client
	store      *store.MemoryStore
	orderStore *tools.MockOrderStore
}

// NewAgent 创建代理
func NewAgent(llmClient *llm.Client, store *store.MemoryStore) *Agent {
	return &Agent{
		llm:        llmClient,
		store:      store,
		orderStore: tools.NewMockOrderStore(),
	}
}

// StreamEvent 流式事件类型
type StreamEventType string

const (
	EventContent  StreamEventType = "content"
	EventToolCall StreamEventType = "tool_call"
	EventToolResult StreamEventType = "tool_result"
	EventDone     StreamEventType = "done"
	EventError    StreamEventType = "error"
)

// StreamEvent 流式事件
type StreamEvent struct {
	Type    StreamEventType     `json:"type"`
	Content string              `json:"content,omitempty"`
	Tool    *ToolCallEvent      `json:"tool,omitempty"`
	Error   string              `json:"error,omitempty"`
}

// ToolCallEvent 工具调用事件
type ToolCallEvent struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Args   string `json:"args"`
	Result string `json:"result,omitempty"`
}

// StreamCallback 流式回调
type StreamCallback func(event StreamEvent)

// Chat 处理聊天
func (a *Agent) Chat(ctx context.Context, sessionID string, userMessage string, callback StreamCallback) error {
	// 保存用户消息
	a.store.AddMessage(&store.Message{
		ID:        fmt.Sprintf("msg_%d", len(a.store.GetMessages(sessionID))+1),
		SessionID: sessionID,
		Role:      "user",
		Content:   userMessage,
	})

	// 构建消息历史
	messages := a.buildMessages(sessionID)

	// 获取工具定义
	tools := a.llm.GetTools()

	// 流式调用 LLM
	var collectedToolCalls []openai.ToolCall
	var fullContent string

	err := a.llm.Chat(ctx, messages, tools, func(content string, done bool, toolCalls []openai.ToolCall) {
		if content != "" {
			fullContent += content
			callback(StreamEvent{
				Type:    EventContent,
				Content: content,
			})
		}

		if len(toolCalls) > 0 {
			collectedToolCalls = append(collectedToolCalls, toolCalls...)
		}

		if done {
			// 处理工具调用
			if len(collectedToolCalls) > 0 {
				a.handleToolCalls(ctx, sessionID, collectedToolCalls, callback)
			}
		}
	})

	if err != nil {
		callback(StreamEvent{
			Type:  EventError,
			Error: err.Error(),
		})
		return err
	}

	// 保存助手回复
	if fullContent != "" {
		a.store.AddMessage(&store.Message{
			ID:        fmt.Sprintf("msg_%d", len(a.store.GetMessages(sessionID))+1),
			SessionID: sessionID,
			Role:      "assistant",
			Content:   fullContent,
		})
	}

	// 发送完成事件
	callback(StreamEvent{Type: EventDone})
	return nil
}

// handleToolCalls 处理工具调用
func (a *Agent) handleToolCalls(ctx context.Context, sessionID string, toolCalls []openai.ToolCall, callback StreamCallback) {
	for _, tc := range toolCalls {
		// 通知工具调用
		callback(StreamEvent{
			Type: EventToolCall,
			Tool: &ToolCallEvent{
				ID:   tc.ID,
				Name: tc.Function.Name,
				Args: tc.Function.Arguments,
			},
		})

		// 执行工具
		var result string
		var err error

		switch tc.Function.Name {
		case "query_order":
			result, err = tools.HandleQueryOrder(a.orderStore, tc.Function.Arguments)
		case "query_user_orders":
			result, err = tools.HandleQueryUserOrders(a.orderStore, tc.Function.Arguments)
		default:
			err = fmt.Errorf("未知工具: %s", tc.Function.Name)
		}

		if err != nil {
			result = fmt.Sprintf("工具调用失败: %s", err.Error())
		}

		// 通知工具结果
		callback(StreamEvent{
			Type: EventToolResult,
			Tool: &ToolCallEvent{
				ID:     tc.ID,
				Name:   tc.Function.Name,
				Result: result,
			},
		})

		// 将工具结果添加到消息历史
		a.store.AddMessage(&store.Message{
			ID:         fmt.Sprintf("msg_%d", len(a.store.GetMessages(sessionID))+1),
			SessionID:  sessionID,
			Role:       "tool",
			Content:    result,
			ToolResult: result,
		})
	}

	// 工具调用后，再次调用 LLM 生成最终回复
	a.generateFinalResponse(ctx, sessionID, callback)
}

// generateFinalResponse 生成最终回复（工具调用后）
func (a *Agent) generateFinalResponse(ctx context.Context, sessionID string, callback StreamCallback) {
	messages := a.buildMessages(sessionID)

	var fullContent string
	err := a.llm.Chat(ctx, messages, nil, func(content string, done bool, toolCalls []openai.ToolCall) {
		if content != "" {
			fullContent += content
			callback(StreamEvent{
				Type:    EventContent,
				Content: content,
			})
		}
	})

	if err != nil {
		callback(StreamEvent{
			Type:  EventError,
			Error: err.Error(),
		})
		return
	}

	// 保存助手回复
	if fullContent != "" {
		a.store.AddMessage(&store.Message{
			ID:        fmt.Sprintf("msg_%d", len(a.store.GetMessages(sessionID))+1),
			SessionID: sessionID,
			Role:      "assistant",
			Content:   fullContent,
		})
	}
}

// buildMessages 构建消息历史
func (a *Agent) buildMessages(sessionID string) []openai.ChatCompletionMessage {
	messages := a.store.GetMessages(sessionID)
	
	// 系统提示
	chatMessages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: "你是一个有用的订单查询助手。你可以帮助用户查询订单状态和订单详情。请用中文回答。",
		},
	}

	// 添加历史消息（保留最近20条）
	startIdx := 0
	if len(messages) > 20 {
		startIdx = len(messages) - 20
	}

	for _, msg := range messages[startIdx:] {
		chatMsg := openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
		chatMessages = append(chatMessages, chatMsg)
	}

	return chatMessages
}

// CreateSession 创建新会话
func (a *Agent) CreateSession() *store.Session {
	sessionID := fmt.Sprintf("sess_%d", len(a.store.ListSessions())+1)
	return a.store.CreateSession(sessionID, "新会话")
}

// GetSession 获取会话
func (a *Agent) GetSession(id string) (*store.Session, bool) {
	return a.store.GetSession(id)
}

// ListSessions 列出会话
func (a *Agent) ListSessions() []*store.Session {
	return a.store.ListSessions()
}

// GetSessionHistory 获取会话历史
func (a *Agent) GetSessionHistory(sessionID string) []*store.Message {
	return a.store.GetMessages(sessionID)
}

// DeleteSession 删除会话
func (a *Agent) DeleteSession(id string) {
	a.store.DeleteSession(id)
}
