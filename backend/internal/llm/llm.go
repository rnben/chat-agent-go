package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
		apiKey = "***" // Mock 模式占位符
		log.Printf("[LLM配置] API密钥: 未设置 (使用Mock模式)")
	} else {
		// 只显示前几位和后几位，中间用*代替
		maskedKey := apiKey
		if len(maskedKey) > 10 {
			maskedKey = maskedKey[:8] + "..." + maskedKey[len(maskedKey)-4:]
		}
		log.Printf("[LLM配置] API密钥: %s", maskedKey)
	}

	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL != "" {
		log.Printf("[LLM配置] 基础URL: %s", baseURL)
	} else {
		log.Printf("[LLM配置] 基础URL: https://api.openai.com (默认)")
	}

	config := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		config.BaseURL = baseURL
	}

	model := os.Getenv("LLM_MODEL")
	if model == "" {
		model = "gpt-3.5-turbo"
	}
	log.Printf("[LLM配置] 模型: %s", model)

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
	log.Printf("[LLM请求] 会话ID: %s, 模型: %s, 消息数: %d, 工具数: %d", sessionID, c.model, len(messages), len(tools))

	// 构建请求体
	req := openai.ChatCompletionRequest{
		Model:    c.model,
		Messages: messages,
		Tools:    tools,
		Stream:   true,
	}

	// 记录完整的请求体
	if reqJSON, err := json.Marshal(req); err == nil {
		log.Printf("[LLM请求体] %s", string(reqJSON))
	}

	stream, err := c.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		// 记录详细的错误信息
		log.Printf("[LLM请求失败] llm.go:82 - stream:%v 错误类型: %T, 错误详情: %v", stream, err, err)

		// 如果是 API 错误，尝试获取更多信息
		if apiErr, ok := err.(*openai.APIError); ok {
			log.Printf("[LLM API错误] 状态码: %d, 消息: %s", apiErr.HTTPStatusCode, apiErr.Message)
			// 尝试获取错误代码
			if apiErr.Code != "" {
				log.Printf("[LLM API错误] 错误代码: %s", apiErr.Code)
			}
		}

		return fmt.Errorf("LLM 请求失败: %w", err)
	}
	defer stream.Close()

	var toolCalls []openai.ToolCall
	var fullResponse string

	for {
		response, err := stream.Recv()
		if err != nil {
			// 记录详细的错误信息
			log.Printf("[LLM接收错误] llm.go:104 - 错误类型: %T, 错误详情: %v", err, err)

			if err.Error() == "EOF" {
				break
			}
			return fmt.Errorf("流式接收失败: %w", err)
		}

		// 记录原始响应
		if responseJSON, err := json.Marshal(response); err == nil {
			log.Printf("[LLM原始响应] %s", string(responseJSON))
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
		// 截断过长的响应
		logResponse := fullResponse
		if len(logResponse) > 500 {
			logResponse = logResponse[:500] + "...[截断]"
		}
		log.Printf("[LLM响应] 内容长度: %d, 内容: %s", len(fullResponse), logResponse)
	}
	if len(toolCalls) > 0 {
		log.Printf("[LLM工具调用] 数量: %d", len(toolCalls))
		for i, tc := range toolCalls {
			log.Printf("  工具 %d: %s, 参数: %s", i+1, tc.Function.Name, tc.Function.Arguments)
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
