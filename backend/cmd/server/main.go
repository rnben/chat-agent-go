package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"chat-agent/internal/agent"
	"chat-agent/internal/api"
	"chat-agent/internal/llm"
	"chat-agent/internal/logger"
	"chat-agent/internal/store"
)

func main() {
	// 初始化日志
	debug := os.Getenv("DEBUG") == "true"
	if err := logger.Init(debug); err != nil {
		logger.InitDefault()
	}
	defer logger.Close()

	// 命令行参数
	port := flag.String("port", "8080", "服务端口")
	dbPath := flag.String("db", "", "SQLite 数据库路径，留空使用内存存储")
	readTimeout := flag.Duration("read-timeout", 30*time.Second, "读取超时")
	writeTimeout := flag.Duration("write-timeout", 30*time.Second, "写入超时")
	idleTimeout := flag.Duration("idle-timeout", 60*time.Second, "空闲超时")
	flag.Parse()

	// 读取环境变量（优先级：命令行 > 环境变量 > 默认值）
	if envPort := os.Getenv("PORT"); envPort != "" {
		*port = envPort
	}

	// 初始化存储
	var storeImpl store.Store
	if *dbPath != "" {
		sqliteStore, err := store.NewSQLiteStore(*dbPath)
		if err != nil {
			logger.Fatal("初始化 SQLite 失败", logger.WithField("error", err))
		}
		storeImpl = sqliteStore
		logger.Info("使用 SQLite 存储", logger.WithField("path", *dbPath))
	} else {
		storeImpl = store.NewMemoryStore()
		logger.Info("使用内存存储")
	}

	// 初始化组件
	llmClient := llm.NewClient()
	agentImpl := agent.NewAgent(llmClient, storeImpl)
	handler := api.NewHandler(agentImpl)

	// 创建服务器
	server := &http.Server{
		Addr:         ":" + *port,
		Handler:      handler.GetRouter(),
		ReadTimeout:  *readTimeout,
		WriteTimeout: *writeTimeout,
		IdleTimeout:  *idleTimeout,
	}

	// 启动服务器
	go func() {
		logger.Info("服务器启动", logger.WithField("port", *port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("服务器启动失败", logger.WithError(err))
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("服务器关闭中...")

	// 优雅关闭（30秒超时）
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("服务器关闭失败", logger.WithError(err))
	}

	logger.Info("服务器已关闭")
}