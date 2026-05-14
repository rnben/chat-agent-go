# Chat Agent - 订单查询助手

一个基于 LLM 的智能订单查询助手，支持自然语言交互、工具调用（查询订单）、流式输出和会话管理。

## 功能特性

- **自然语言对话** - 用日常语言查询订单，如"查询订单 ORD-20260515-001"
- **工具调用** - LLM 自动识别意图并调用订单查询工具
- **流式输出** - SSE 实时流式响应，逐字显示
- **会话管理** - 创建、查看、删除对话会话
- **上下文记忆** - 保留最近 20 条对话历史

## 技术栈

| 组件 | 技术 |
|------|------|
| 后端 | Go 1.21+、gorilla/mux、go-openai |
| 前端 | Vue 3、Vite 5 |
| LLM | OpenAI 兼容接口（支持 Mock 模式） |
| 存储 | 内存存储（重启清空） |

## 快速开始

### 前置要求

- Go 1.21+
- Node.js 18+
- OpenAI API Key（可选，默认使用 Mock 模式）

### 1. 配置环境变量（可选）

```bash
export OPENAI_API_KEY="sk-xxx"
export OPENAI_BASE_URL="https://api.openai.com/v1"  # 或其他兼容接口
export LLM_MODEL="gpt-3.5-turbo"
export PORT="8080"
```

> 不配置则使用 Mock 模式，LLM 返回模拟数据

### 2. 启动方式

**方式一：完整模式（推荐）**

```bash
# 构建前端
cd frontend && npm install && npm run build && cd ..

# 启动后端（会同时提供前端静态文件）
cd backend && go run cmd/server/main.go
```

浏览器打开: http://localhost:8080

**方式二：开发模式**

```bash
# 终端 1：启动后端
cd backend && go run cmd/server/main.go

# 终端 2：启动前端开发服务器（API 会代理到后端）
cd frontend && npm install && npm run dev
```

浏览器打开: http://localhost:5173

### 3. 测试对话

```
用户: 查询订单 ORD-20260515-001
助手: [调用 query_order 工具]
      订单号: ORD-20260515-001
      状态: 已发货
      商品: MacBook Pro 14寸 x1 (14999.00元), AirPods Pro 2 x1 (1899.00元)
      ...
```

## API 接口

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/chat` | 发送消息（SSE 流式响应） |
| GET | `/api/sessions` | 获取会话列表 |
| POST | `/api/sessions` | 创建新会话 |
| GET | `/api/sessions/:id` | 获取会话详情 |
| GET | `/api/sessions/:id/history` | 获取会话历史 |
| DELETE | `/api/sessions/:id` | 删除会话 |

### 聊天接口示例

```bash
curl -N -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{"session_id": "", "message": "查询用户 user_001 的所有订单"}'
```

### 流式事件类型

| 事件类型 | 说明 |
|----------|------|
| `session` | 返回会话 ID（首次创建时） |
| `content` | LLM 生成的文本内容 |
| `tool_call` | 工具调用请求 |
| `tool_result` | 工具执行结果 |
| `done` | 对话完成 |
| `error` | 错误信息 |

## Mock 数据

系统内置 3 个测试订单：

| 订单号 | 用户 | 状态 | 金额 | 商品 |
|--------|------|------|------|------|
| ORD-20260515-001 | user_001 | 已发货 | ¥16,898 | MacBook Pro 14寸, AirPods Pro 2 |
| ORD-20260514-002 | user_001 | 已送达 | ¥8,999 | iPhone 16 Pro |
| ORD-20260515-003 | user_002 | 待支付 | ¥9,598 | iPad Air x2 |

支持的查询方式：
- 按订单号查询："查询订单 ORD-20260515-001"
- 按用户查询："查询 user_001 的订单"

## 项目结构

```
chat-agent/
├── README.md
├── backend/
│   ├── cmd/server/
│   │   └── main.go           # 服务入口
│   ├── internal/
│   │   ├── agent/
│   │   │   └── agent.go      # Agent 核心逻辑、工具调度
│   │   ├── api/
│   │   │   └── api.go        # HTTP API、路由
│   │   ├── llm/
│   │   │   └── llm.go        # LLM 客户端封装
│   │   ├── store/
│   │   │   └── store.go      # 内存存储（会话、消息）
│   │   └── tools/
│   │       └── tools.go      # 订单查询工具、Mock 数据
│   ├── go.mod
│   └── go.sum
├── frontend/
│   ├── src/
│   │   ├── App.vue           # 主应用组件
│   │   └── main.js           # 入口文件
│   ├── index.html
│   ├── package.json
│   ├── vite.config.js
│   └── dist/                 # 构建产物
└── .gitignore
```

## 扩展开发

### 添加新工具

1. 在 `backend/internal/tools/` 创建工具实现
2. 在 `agent.go` 的 `handleToolCalls` 中注册工具
3. 在 LLM 配置中添加工具定义（`llm.go` 的 `GetTools`）

### 替换真实数据库

修改 `backend/internal/store/store.go`，将内存存储替换为数据库实现。

## License

MIT
