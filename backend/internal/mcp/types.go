package mcp

import (
	"encoding/json"
)

// JSONRPCVersion JSON-RPC 版本
const JSONRPCVersion = "2.0"

// JSONRPCRequest JSON-RPC 请求
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
}

// JSONRPCResponse JSON-RPC 响应
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError      `json:"error,omitempty"`
	ID      interface{}    `json:"id,omitempty"`
}

// RPCError RPC 错误
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any   `json:"data,omitempty"`
}

// 错误码
const (
	ErrCodeParseError     = -32700 // Parse error - Invalid JSON was received
	ErrCodeInvalidReq   = -32600 // Invalid Request - the JSON sent is not a valid Request object
	ErrCodeMethodNotFound = -32601 // Method not found - Procedure not found
	ErrCodeInvalidParams = -32602 // Invalid params - Invalid method parameter(s)
	ErrCodeInternal     = -32603 // Internal error - Internal JSON-RPC error
	ErrCodeServerError  = -32000 // Server error - Internal server error

	// MCP 扩展错误码
	ErrCodeToolNotFound  = -32001 // Tool not found
	ErrCodeToolExecute  = -32002 // Tool execute error
	ErrCodeNoTools    = -32003 // No tools available
)

// MCPRequest MCP 请求
type MCPRequest struct {
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
}

// MCPResponse MCP 响应
type MCPResponse struct {
	Result  *MCPResult `json:"result,omitempty"`
	Error   *RPCError  `json:"error,omitempty"`
	ID      interface{} `json:"id,omitempty"`
}

// MCPResult MCP 结果
type MCPResult struct {
	Tools []ToolDefinition `json:"tools,omitempty"`
}

// ToolDefinition MCP 工具定义
type ToolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

// InputSchema 输入模式
type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties"`
	Required   []string           `json:"required,omitempty"`
}

// Property 属性
type Property struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// ToolCallRequest 工具调用请求
type ToolCallRequest struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// ToolCallResponse 工具调用响应
type ToolCallResponse struct {
	Name      string `json:"name"`
	Result   string `json:"result"`
	IsError  bool   `json:"isError,omitempty"`
}

// NewRPCError 创建 RPC 错误
func NewRPCError(code int, message string, data any) *RPCError {
	return &RPCError{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// ParseError 解析错误
func ParseError(err error) *RPCError {
	return NewRPCError(ErrCodeParseError, "Parse error", err.Error())
}

// MethodNotFound 方法未找到
func MethodNotFound(method string) *RPCError {
	return NewRPCError(ErrCodeMethodNotFound, "Method not found: "+method, nil)
}

// InvalidParams 参数无效
func InvalidParams(err error) *RPCError {
	return NewRPCError(ErrCodeInvalidParams, "Invalid params", err.Error())
}

// InternalError 内部错误
func InternalError(err error) *RPCError {
	return NewRPCError(ErrCodeInternal, "Internal error", err.Error())
}

// ToolNotFound 工具未找到
func ToolNotFound(name string) *RPCError {
	return NewRPCError(ErrCodeToolNotFound, "Tool not found: "+name, nil)
}

// ToolExecuteError 工具执行错误
func ToolExecuteError(name string, err error) *RPCError {
	return NewRPCError(ErrCodeToolExecute, "Tool execute error: "+name, err.Error())
}

// NoTools 没有可用工具
func NoTools() *RPCError {
	return NewRPCError(ErrCodeNoTools, "No tools available", nil)
}