# Chat Agent - 订单查询助手

一个前后端分离的智能聊天 Agent，支持订单状态查询（工具调用）、流式输出、上下文记忆。

## 技术栈

- **后端**: Go + net/http + gorilla/mux
- **前端**: Vue 3 + Vite
- **LLM**: OpenAI 兼容接口
- **数据**: 内存 Mock

## 快速开始

### 1. 配置环境变量

```bash
# 可选 - 默认使用 Mock 模式
export OPENAI_API_KEY="your-api-key"
export OPENAI_BASE_URL="https://api.openai.com/v1"  # 或其他兼容接口
export LLM_MODEL="gpt-3.5-turbo"
export PORT="8080"
```

### 2. 运行后端

```bash
cd backend
go run cmd/server/main.go
```

### 3. 访问应用

浏览器打开: http://localhost:8080

## API 接口

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /api/chat | 发送消息（SSE 流式响应） |
| GET | /api/sessions | 获取会话列表 |
| POST | /api/sessions | 创建新会话 |
| GET | /api/sessions/:id | 获取会话详情 |
| GET | /api/sessions/:id/history | 获取会话历史 |
| DELETE | /api/sessions/:id | 删除会话 |

## 聊天 API 示例

```bash
curl -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{"session_id": "", "message": "查询订单 ORD-20260515-001"}'
```

## Mock 订单数据

| 订单号 | 用户 | 状态 | 金额 |
|--------|------|------|------|
| ORD-20260515-001 | user_001 | 已发货 | ¥16,898 |
| ORD-20260514-002 | user_001 | 已送达 | ¥8,999 |
| ORD-20260515-003 | user_002 | 待支付 | ¥9,598 |

## 项目结构

```
chat-agent/
├── backend/
│   ├── cmd/server/      # 入口
│   ├── internal/
│   │   ├── agent/       # Agent 核心逻辑
│   │   ├── api/         # HTTP API
│   │   ├── llm/         # LLM 客户端
│   │   ├── store/       # 数据存储
│   │   └── tools/       # 工具实现
│   └── go.mod
└── frontend/
    ├── src/
    │   ├── api/         # API 客户端
    │   ├── components/  # UI 组件
    │   └── App.vue      # 主应用
    └── package.json
```
