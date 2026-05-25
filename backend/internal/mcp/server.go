package mcp

import (
	"context"
	"fmt"
	"net/http"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"chat-agent/internal/logger"
	"chat-agent/internal/tools"
)

// Server MCP 服务器 (基于 mcp-go SDK)
type Server struct {
	mcpServer  *server.MCPServer
	httpServer *server.StreamableHTTPServer
	addr       string
}

// NewServer 创建 MCP 服务器
func NewServer(addr string) *Server {
	mcpServer := server.NewMCPServer(
		"chat-agent-mcp",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	orderStore := tools.NewMockOrderStore()

	queryOrderTool := mcp.NewTool("query_order",
		mcp.WithDescription("根据订单号查询订单状态和详细信息"),
		mcp.WithString("order_id",
			mcp.Required(),
			mcp.Description("订单号，例如: ORD-20260515-001"),
		),
	)

	mcpServer.AddTool(queryOrderTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		orderID, err := request.RequireString("order_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		logger.Info("MCP工具调用",
			logger.WithField("name", "query_order"),
			logger.WithField("order_id", orderID),
		)

		result, err := tools.HandleQueryOrder(orderStore, fmt.Sprintf(`{"order_id": "%s"}`, orderID))
		if err != nil {
			logger.Error("MCP工具执行失败",
				logger.WithField("name", "query_order"),
				logger.WithField("error", err.Error()),
			)
			return mcp.NewToolResultError(err.Error()), nil
		}

		logger.Info("MCP工具执行成功",
			logger.WithField("name", "query_order"),
		)

		return mcp.NewToolResultText(result), nil
	})

	queryUserOrdersTool := mcp.NewTool("query_user_orders",
		mcp.WithDescription("查询用户的所有订单列表"),
		mcp.WithString("user_id",
			mcp.Required(),
			mcp.Description("用户ID，例如: user_001"),
		),
	)

	mcpServer.AddTool(queryUserOrdersTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		userID, err := request.RequireString("user_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		logger.Info("MCP工具调用",
			logger.WithField("name", "query_user_orders"),
			logger.WithField("user_id", userID),
		)

		result, err := tools.HandleQueryUserOrders(orderStore, fmt.Sprintf(`{"user_id": "%s"}`, userID))
		if err != nil {
			logger.Error("MCP工具执行失败",
				logger.WithField("name", "query_user_orders"),
				logger.WithField("error", err.Error()),
			)
			return mcp.NewToolResultError(err.Error()), nil
		}

		logger.Info("MCP工具执行成功",
			logger.WithField("name", "query_user_orders"),
		)

		return mcp.NewToolResultText(result), nil
	})

	httpServer := server.NewStreamableHTTPServer(mcpServer,
		server.WithEndpointPath("/mcp"),
	)

	return &Server{
		mcpServer:  mcpServer,
		httpServer: httpServer,
		addr:       addr,
	}
}

// loggingResponseWriter 包装 http.ResponseWriter，捕获状态码
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *loggingResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// responseLogMiddleware MCP 响应日志中间件，打印响应状态码和响应头
func responseLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(lrw, r)

		fields := []logger.Field{
			logger.WithField("method", r.Method),
			logger.WithField("path", r.URL.Path),
			logger.WithField("status", lrw.statusCode),
			logger.WithField("headers", w.Header()),
		}

		logger.Info("MCP响应", fields...)
	})
}

// GetHandler 获取 HTTP handler (用于集成到现有 router)
func (s *Server) GetHandler() http.Handler {
	return responseLogMiddleware(s.httpServer)
}

// Start 启动 MCP 服务器
func (s *Server) Start() error {
	logger.Info("MCP服务器启动 (Streamable HTTP via mcp-go)", logger.WithField("addr", s.addr))
	return http.ListenAndServe(s.addr, s.httpServer)
}
