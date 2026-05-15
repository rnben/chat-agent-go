# Go build
BINARY_NAME := chat-agent
BUILD_DIR := build
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# 颜色输出
RED := \033[31m
GREEN := \033[32m
YELLOW := \033[33m
NC := \033[0m

.PHONY: help build test test-v lint run clean install docker docker-build docker-run

help:
	@echo "$(GREEN)chat-agent 构建工具$(NC)"
	@echo ""
	@echo "可用命令:"
	@echo "  make build      - 编译二进制"
	@echo "  make test     - 运行测试"
	@echo "  make test-v   - 运行测试 (verbose)"
	@echo "  make lint    - 运行 lint"
	@echo "  make run     - 运行服务"
	@echo "  make clean   - 清理构建产物"
	@echo "  make docker  - 构建 Docker 镜像"

build:
	@echo "$(GREEN)编译 $(BINARY_NAME)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin cmd/server/main.go
	GOOS=linux GOARCH=amd64 GOHOSTOS=darwin CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux cmd/server/main.go
	@echo "$(GREEN)编译完成: $(BUILD_DIR)/$(BINARY_NAME)-*$(NC)"

test:
	@echo "$(GREEN)运行测试...$(NC)"
	go test -race -cover ./...

test-v:
	go test -v -race -cover ./...

lint:
	@echo "$(GREEN)运行 lint...$(NC)"
	golangci-lint run ./...

run:
	@echo "$(GREEN)启动服务...$(NC)"
	go run cmd/server/main.go

clean:
	@echo "$(YELLOW)清理构建产物...$(NC)"
	rm -rf $(BUILD_DIR)
	@echo "$(GREEN)清理完成$(NC)"

install:
	go install $(LDFLAGS) ./cmd/server

docker:
	docker build -t chat-agent:latest .

docker-build:
	docker build --build-arg VERSION=$(VERSION) -t chat-agent:$(VERSION) .

docker-run:
	docker run -p 8080:8080 chat-agent:latest