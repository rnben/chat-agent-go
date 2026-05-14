package llm

import (
	"context"
	"fmt"
	"os"

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
		apiKey = "sk-placeholder" // Mock 模式占位符
	}
	
	baseURL := os.Getenv("OPENAI_BASE_URL")
	
	config := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		config.BaseURL = baseURL
	}

	model := os.Getenv("LLM_MODEL")
	if model == "" {
		model = "gpt-3.5-turbo"
	}

	return &Client{
		client: openai.NewClientWithConfig(config),
		model:  model,
	}
}

// StreamCallback 流式回调
type StreamCallback func(content string, done bool, toolCalls []openai.ToolCall)

// Chat 流式聊天
func (c *Client) Chat(ctx context.Context, messages []openai.ChatCompletionMessage, tools []openai.Tool, callback StreamCallback) error {
	stream, err := c.client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model:    c.model,
		Messages: messages,
		Tools:    tools,
		Stream:   true,
	})
	if err != nil {
		return fmt.Errorf("LLM 请求失败: %w", err)
	}
	defer stream.Close()

	var toolCalls []openai.ToolCall
	
	for {
		response, err := stream.Recv()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return fmt.Errorf("流式接收失败: %w", err)
		}

		// 检查是否有内容
		if len(response.Choices) > 0 {
			delta := response.Choices[0].Delta
			
			// 处理内容
			if delta.Content != "" {
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
