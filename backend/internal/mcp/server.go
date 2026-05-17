package mcp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"chat-agent/internal/logger"
	"chat-agent/internal/tools"
)

// Server MCP 服务器
type Server struct {
	addr string
}

// NewServer 创建 MCP 服务器
func NewServer(addr string) *Server {
	return &Server{addr: addr}
}

// Start 启动 MCP 服务器
func (s *Server) Start() error {
	http.HandleFunc("/mcp", s.handleMCP)

	logger.Info("MCP服务器启动", logger.WithField("addr", s.addr))
	return http.ListenAndServe(s.addr, nil)
}

// handleMCP 处理 MCP 请求
func (s *Server) handleMCP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 只支持 POST
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{
			"jsonrpc": JSONRPCVersion,
			"error":  fmt.Sprintf("method %s not allowed", r.Method),
		})
		return
	}

	// 解析请求
	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(JSONRPCResponse{
			JSONRPC: JSONRPCVersion,
			Error:  ParseError(err),
		})
		return
	}

	// 处理请求
	resp := s.processRequest(&req)

	// 写入响应
	json.NewEncoder(w).Encode(resp)
}

// processRequest 处理请求
func (s *Server) processRequest(req *JSONRPCRequest) JSONRPCResponse {
	method := req.Method

	logger.Info("MCP请求", logger.WithField("method", method))

	switch method {
	case "initialize":
		return s.handleInitialize(req)
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(req)
	default:
		return JSONRPCResponse{
			JSONRPC: JSONRPCVersion,
			Error:  MethodNotFound(method),
			ID:     req.ID,
		}
	}
}

// handleInitialize 处理初始化
func (s *Server) handleInitialize(req *JSONRPCRequest) JSONRPCResponse {
	result := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": true,
		},
		"serverInfo": map[string]string{
			"name":    "chat-agent-mcp",
			"version": "1.0.0",
		},
	}

	resultJSON, _ := json.Marshal(result)
	return JSONRPCResponse{
		JSONRPC: JSONRPCVersion,
		Result:  resultJSON,
		ID:      req.ID,
	}
}

// handleToolsList 处理工具列表
func (s *Server) handleToolsList(req *JSONRPCRequest) JSONRPCResponse {
	toolDefs := tools.GetToolDefinitions()

	var mcpTools []ToolDefinition
	for _, td := range toolDefs {
		mcpTools = append(mcpTools, ToolDefinition{
			Name:        td.Name,
			Description: td.Description,
			InputSchema: InputSchema{
				Type:       "object",
				Properties: convertProps(td.Parameters),
				Required:   convertRequired(td.Parameters),
			},
		})
	}

	result := MCPResult{Tools: mcpTools}
	resultJSON, _ := json.Marshal(result)

	logger.Info("MCP工具列表",
		logger.WithField("count", len(mcpTools)),
		logger.WithField("tools", strings.Join(getToolNames(mcpTools), ", ")),
	)

	return JSONRPCResponse{
		JSONRPC: JSONRPCVersion,
		Result:  resultJSON,
		ID:      req.ID,
	}
}

// handleToolsCall 处理工具调用
func (s *Server) handleToolsCall(req *JSONRPCRequest) JSONRPCResponse {
	var callReq ToolCallRequest
	if err := json.Unmarshal(req.Params, &callReq); err != nil {
		return JSONRPCResponse{
			JSONRPC: JSONRPCVersion,
			Error:  InvalidParams(err),
			ID:     req.ID,
		}
	}

	logger.Info("MCP工具调用",
		logger.WithField("name", callReq.Name),
		logger.WithField("args", string(callReq.Arguments)),
	)

	// 执行工具
	result, err := tools.ExecuteTool(callReq.Name, string(callReq.Arguments))
	if err != nil {
		logger.Error("MCP工具执行失败",
			logger.WithField("name", callReq.Name),
			logger.WithField("error", err.Error()),
		)
		return JSONRPCResponse{
			JSONRPC: JSONRPCVersion,
			Error:  ToolExecuteError(callReq.Name, err),
			ID:     req.ID,
		}
	}

	logger.Info("MCP工具执行成功",
		logger.WithField("name", callReq.Name),
	)

	// 返回结果
	callResp := ToolCallResponse{
		Name:     callReq.Name,
		Result:   result,
		IsError:  false,
	}
	respJSON, _ := json.Marshal(callResp)

	return JSONRPCResponse{
		JSONRPC: JSONRPCVersion,
		Result:  respJSON,
		ID:      req.ID,
	}
}

// convertProps 转换属性
func convertProps(params map[string]any) map[string]Property {
	props := make(map[string]Property)
	if params == nil {
		return props
	}

	propsMap, ok := params["properties"].(map[string]any)
	if !ok {
		return props
	}

	for k, v := range propsMap {
		if propMap, ok := v.(map[string]any); ok {
			props[k] = Property{
				Type:        getString(propMap, "type"),
				Description: getString(propMap, "description"),
			}
		}
	}
	return props
}

// convertRequired 转换必需字段
func convertRequired(params map[string]any) []string {
	if params == nil {
		return nil
	}

	req, ok := params["required"].([]any)
	if !ok {
		return nil
	}

	var result []string
	for _, v := range req {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// getString 获取字符串
func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// getToolNames 获取工具名称列表
func getToolNames(tools []ToolDefinition) []string {
	var names []string
	for _, t := range tools {
		names = append(names, t.Name)
	}
	return names
}