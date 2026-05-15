package agent

import (
	"context"
	"fmt"

	"chat-agent/internal/llm"
	"chat-agent/internal/logger"
	"chat-agent/internal/store"
	"chat-agent/internal/tools"

	openai "github.com/sashabaranov/go-openai"
)

// Agent 对话代理
type Agent struct {
	llm        *llm.Client
	store      store.Store
	orderStore *tools.MockOrderStore
}

// NewAgent 创建代理
func NewAgent(llmClient *llm.Client, s store.Store) *Agent {
	return &Agent{
		llm:        llmClient,
		store:      s,
		orderStore: tools.NewMockOrderStore(),
	}
}

// StreamEvent 流式事件类型
type StreamEventType string

const (
	EventContent    StreamEventType = "content"
	EventToolCall  StreamEventType = "tool_call"
	EventToolResult StreamEventType = "tool_result"
	EventDone       StreamEventType = "done"
	EventError      StreamEventType = "error"
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

	// 记录系统提示词
	if len(messages) > 0 && messages[0].Role == openai.ChatMessageRoleSystem {
		logger.Debug("系统提示词",
			logger.WithField("session_id", sessionID),
			logger.WithField("content", messages[0].Content),
		)
	}

	// 记录发送给LLM的prompt
	logger.Info("LLM请求",
		logger.WithField("session_id", sessionID),
		logger.WithField("messages", len(messages)),
	)

	// 获取工具定义
	tools := a.llm.GetTools()

	logger.Info("工具定义",
		logger.WithField("session_id", sessionID),
		logger.WithField("count", len(tools)),
	)
	for i, tool := range tools {
		if tool.Function != nil {
			logger.Debug("工具详情",
				logger.WithField("session_id", sessionID),
				logger.WithField("index", i+1),
				logger.WithField("name", tool.Function.Name),
				logger.WithField("description", tool.Function.Description),
			)
		}
	}

	// 流式调用 LLM
	var collectedToolCalls []openai.ToolCall
	var fullContent string

	err := a.llm.Chat(ctx, sessionID, messages, tools, func(content string, done bool, toolCalls []openai.ToolCall) {
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
	// 先保存助手消息（包含 ToolCalls）
	if len(toolCalls) > 0 {
		var tcList []store.ToolCall
		for _, tc := range toolCalls {
			tcList = append(tcList, store.ToolCall{
				ID:       tc.ID,
				Name:     tc.Function.Name,
				ArgsJSON: tc.Function.Arguments,
			})
		}
		a.store.AddMessage(&store.Message{
			ID:        fmt.Sprintf("msg_%d", len(a.store.GetMessages(sessionID))+1),
			SessionID: sessionID,
			Role:      "assistant",
			Content:   "",
			ToolCalls: tcList,
		})
		logger.Info("保存助手消息(含ToolCalls)",
			logger.WithField("session_id", sessionID),
			logger.WithField("count", len(toolCalls)),
		)
	}

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

		logger.Info("工具调用",
			logger.WithField("session_id", sessionID),
			logger.WithField("tool_id", tc.ID),
			logger.WithField("tool_name", tc.Function.Name),
			logger.WithField("arguments", tc.Function.Arguments),
		)

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

		logger.Info("工具结果",
			logger.WithField("session_id", sessionID),
			logger.WithField("tool_id", tc.ID),
		)

		// 将工具结果添加到消息历史
		a.store.AddMessage(&store.Message{
			ID:         fmt.Sprintf("msg_%d", len(a.store.GetMessages(sessionID))+1),
			SessionID:  sessionID,
			Role:       "tool",
			Content:    result,
			ToolResult: result,
			ToolCallID: tc.ID,
			ToolCalls: []store.ToolCall{
				{
					ID:       tc.ID,
					Name:     tc.Function.Name,
					ArgsJSON: tc.Function.Arguments,
				},
			},
		})
	}

	// 工具调用后，再次调用 LLM 生成最终回复
	a.generateFinalResponse(ctx, sessionID, callback)
}

// generateFinalResponse 生成最终回复（工具调用后）
func (a *Agent) generateFinalResponse(ctx context.Context, sessionID string, callback StreamCallback) {
	messages := a.buildMessages(sessionID)

	logger.Info("LLM最终回复请求",
		logger.WithField("session_id", sessionID),
		logger.WithField("messages", len(messages)),
	)

	var fullContent string
	err := a.llm.Chat(ctx, sessionID, messages, nil, func(content string, done bool, toolCalls []openai.ToolCall) {
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

	logger.Debug("buildMessages",
		logger.WithField("session_id", sessionID),
		logger.WithField("消息数", len(messages)),
	)

	// 系统提示
	chatMessages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: "你是一个有用的问答助手，可以帮助用户回答各种问题。请用中文回答。",
		},
	}

	// 需要追踪最近的工具调用，以便正确关联
	var pendingToolCalls []openai.ToolCall

	// 添加历史消息（保留最近20条）
	startIdx := 0
	if len(messages) > 20 {
		startIdx = len(messages) - 20
	}

	for i, msg := range messages[startIdx:] {
		// 处理 tool 消息 - 需要携带对应的 ToolCallID
		if msg.Role == "tool" && msg.ToolCallID != "" {
			logger.Debug("buildMessages tool消息",
				logger.WithField("session_id", sessionID),
				logger.WithField("index", i),
				logger.WithField("tool_call_id", msg.ToolCallID),
			)
		}

		if msg.Role == "tool" && msg.ToolCallID != "" {
			// 查找对应的 tool call
			var matchedToolCall *openai.ToolCall
			for _, tc := range pendingToolCalls {
				if tc.ID == msg.ToolCallID {
					matchedToolCall = &tc
					break
				}
			}
			if matchedToolCall != nil {
				logger.Debug("buildMessages 匹配成功",
					logger.WithField("session_id", sessionID),
					logger.WithField("tool", matchedToolCall.Function.Name),
				)
				chatMsg := openai.ChatCompletionMessage{
					Role:      msg.Role,
					Content:   msg.Content,
					ToolCallID: msg.ToolCallID,
					ToolCalls:  []openai.ToolCall{*matchedToolCall},
				}
				chatMessages = append(chatMessages, chatMsg)
			} else {
				logger.Warn("buildMessages 未找到匹配tool",
					logger.WithField("session_id", sessionID),
					logger.WithField("tool_call_id", msg.ToolCallID),
				)
				// 如果找不到匹配的 tool call，只传 content
				chatMsg := openai.ChatCompletionMessage{
					Role:    msg.Role,
					Content: msg.Content,
				}
				chatMessages = append(chatMessages, chatMsg)
			}
			continue
		}

		// 处理助手消息，提取工具调用
		chatMsg := openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}

		// 如果有助手的工具调用，需要设置 ToolCalls
		if len(msg.ToolCalls) > 0 {
			logger.Debug("buildMessages 助手消息有ToolCalls",
				logger.WithField("session_id", sessionID),
				logger.WithField("count", len(msg.ToolCalls)),
			)
			for _, tc := range msg.ToolCalls {
				logger.Debug("buildMessages tool",
					logger.WithField("session_id", sessionID),
					logger.WithField("id", tc.ID),
					logger.WithField("name", tc.Name),
				)
				toolCall := openai.ToolCall{
					ID:   tc.ID,
					Type: "function",
					Function: openai.FunctionCall{
						Name:      tc.Name,
						Arguments: tc.ArgsJSON,
					},
				}
				chatMsg.ToolCalls = append(chatMsg.ToolCalls, toolCall)
				pendingToolCalls = append(pendingToolCalls, toolCall)
			}
		}

		chatMessages = append(chatMessages, chatMsg)
	}

	logger.Debug("buildMessages 最终",
		logger.WithField("session_id", sessionID),
		logger.WithField("消息数", len(chatMessages)),
	)

	return chatMessages
}

// CreateSession 创建新会话
func (a *Agent) CreateSession() *store.Session {
	sessionID := fmt.Sprintf("sess_%d", len(a.store.ListSessions())+1)
	logger.Info("创建会话",
		logger.WithField("session_id", sessionID),
	)
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
	logger.Info("删除会话",
		logger.WithField("session_id", id),
	)
	a.store.DeleteSession(id)
}