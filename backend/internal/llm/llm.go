package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"chat-agent/internal/logger"

	openai "github.com/sashabaranov/go-openai"
)

// Client LLM 客户端
type Client struct {
	client *openai.Client
	model  string
}

// NewClient 创建 LLM 客户端
func NewClient() *Client {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = "***" // Mock 模式占位符
		logger.Info("LLM配置", logger.WithField("api_key", "未设置 (使用Mock模式)"))
	} else {
		// 只显示前几位和后几位，中间用*代替
		maskedKey := apiKey
		if len(maskedKey) > 10 {
			maskedKey = maskedKey[:8] + "..." + maskedKey[len(maskedKey)-4:]
		}
		logger.Info("LLM配置", logger.WithField("api_key", maskedKey))
	}

	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL != "" {
		logger.Info("LLM配置", logger.WithField("base_url", baseURL))
	} else {
		logger.Info("LLM配置", logger.WithField("base_url", "https://api.openai.com (默认)"))
	}

	config := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		config.BaseURL = baseURL
	}

	model := os.Getenv("LLM_MODEL")
	if model == "" {
		model = "gpt-3.5-turbo"
	}
	logger.Info("LLM配置", logger.WithField("model", model))

	return &Client{
		client: openai.NewClientWithConfig(config),
		model:  model,
	}
}

// StreamCallback 流式回调
type StreamCallback func(content string, done bool, toolCalls []openai.ToolCall)

// Chat 流式聊天
func (c *Client) Chat(ctx context.Context, sessionID string, messages []openai.ChatCompletionMessage, tools []openai.Tool, callback StreamCallback) error {
	// 记录LLM请求详情
	logger.Info("LLM请求",
		logger.WithField("session_id", sessionID),
		logger.WithField("model", c.model),
		logger.WithField("messages", len(messages)),
		logger.WithField("tools", len(tools)),
	)

	// 构建请求体
	req := openai.ChatCompletionRequest{
		Model:    c.model,
		Messages: messages,
		Tools:    tools,
		Stream:   true,
	}

	// 记录完整的请求体
	if reqJSON, err := json.Marshal(req); err == nil {
		logger.Info("LLM请求体", logger.WithField("body", string(reqJSON)))
	}

	stream, err := c.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		logger.Error("LLM请求失败",
			logger.WithField("session_id", sessionID),
			logger.WithField("error_type", fmt.Sprintf("%T", err)),
			logger.WithField("error", err.Error()),
		)

		// 如果是 API 错误，尝试获取更多信息
		if apiErr, ok := err.(*openai.APIError); ok {
			logger.Error("LLM API错误",
				logger.WithField("session_id", sessionID),
				logger.WithField("status", apiErr.HTTPStatusCode),
				logger.WithField("message", apiErr.Message),
			)
		}

		return fmt.Errorf("LLM 请求失败: %w", err)
	}
	defer stream.Close()

	var toolCalls []openai.ToolCall
	var fullResponse string

	for {
		response, err := stream.Recv()
		if err != nil {
			logger.Error("LLM接收错误",
				logger.WithField("session_id", sessionID),
				logger.WithField("error_type", fmt.Sprintf("%T", err)),
				logger.WithField("error", err.Error()),
			)

			if err.Error() == "EOF" {
				break
			}
			return fmt.Errorf("流式接收失败: %w", err)
		}

		// 记录原始响应（截断）
		if responseJSON, err := json.Marshal(response); err == nil {
			respStr := string(responseJSON)
			if len(respStr) > 500 {
				respStr = respStr[:500] + "..."
			}
			logger.Debug("LLM原始响应", logger.WithField("session_id", sessionID), logger.WithField("response", respStr))
		}

		// 检查是否有内容
		if len(response.Choices) > 0 {
			delta := response.Choices[0].Delta

			// 处理内容
			if delta.Content != "" {
				fullResponse += delta.Content
				callback(delta.Content, false, nil)
			}

			// 处理工具调用
			if len(delta.ToolCalls) > 0 {
				for _, tc := range delta.ToolCalls {
					// 累积工具调用
					if tc.ID != "" {
						// 新的工具调用
						toolCalls = append(toolCalls, openai.ToolCall{
							ID:   tc.ID,
							Type: tc.Type,
							Function: openai.FunctionCall{
								Name:      tc.Function.Name,
								Arguments: tc.Function.Arguments,
							},
						})
					} else if len(toolCalls) > 0 {
						// 继续填充参数
						last := &toolCalls[len(toolCalls)-1]
						last.Function.Arguments += tc.Function.Arguments
					}
				}
			}
		}
	}

	// 记录LLM响应详情
	if fullResponse != "" {
		logResponse := fullResponse
		if len(logResponse) > 500 {
			logResponse = logResponse[:500] + "..."
		}
		logger.Info("LLM响应",
			logger.WithField("session_id", sessionID),
			logger.WithField("content_length", len(fullResponse)),
			logger.WithField("content", logResponse),
		)
	}
	if len(toolCalls) > 0 {
		logger.Info("LLM工具调用",
			logger.WithField("session_id", sessionID),
			logger.WithField("count", len(toolCalls)),
		)
		for i, tc := range toolCalls {
			logger.Info("工具调用详情",
				logger.WithField("session_id", sessionID),
				logger.WithField("index", i+1),
				logger.WithField("name", tc.Function.Name),
				logger.WithField("arguments", tc.Function.Arguments),
			)
		}
	}

	// 发送完成信号，包含工具调用
	callback("", true, toolCalls)
	return nil
}

// GetTools 获取工具定义
func (c *Client) GetTools() []openai.Tool {
	return []openai.Tool{
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "query_order",
				Description: "根据订单号查询订单状态和详细信息",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"order_id": map[string]interface{}{
							"type":        "string",
							"description": "订单号，例如: ORD-20260515-001",
						},
					},
					"required": []string{"order_id"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "query_user_orders",
				Description: "查询用户的所有订单列表",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"user_id": map[string]interface{}{
							"type":        "string",
							"description": "用户ID，例如: user_001",
						},
					},
					"required": []string{"user_id"},
				},
			},
		},
	}
}