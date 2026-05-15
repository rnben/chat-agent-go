# Issue: MiniMax 工具调用 API 兼容性

## 问题描述

使用 MiniMax API 时，工具调用后再次请求模型报错：
```
tool result's tool id() not found (2013)
```

其他 LLM（如 OpenAI）无此问题。

## 原因分析

MiniMax API 要求严格：发送 tool 角色消息时，必须同时包含：
1. `tool_call_id` - 关联到哪个调用
2. `content` - 工具返回结果
3. **必须**：`tool_calls` - 完整的工具调用信息（之前漏了这个！）

```json
{
  "role": "tool",
  "tool_call_id": "call_xxx",  // 需要
  "content": "订单信息...",    // 需要
  "tool_calls": [...]         // 需要！之前漏了
}
```

其他 LLM 更宽松，只要 `tool_call_id` 匹配即可，不强制要求 `tool_calls`。

## 修复方案

在调用工具时，先保存助手消息的 tool_calls 信息：

```go
// handleToolCalls 中
// 1. 先保存助手消息（含 tool_calls）
a.store.AddMessage(&Message{
    Role:      "assistant",
    ToolCalls: []ToolCall{{ID: tc.ID, Name: tc.Function.Name, ...}},
})

// 2. 再保存 tool 结果
a.store.AddMessage(&Message{
    Role:       "tool",
    Content:    result,
    ToolCallID: tc.ID,
    ToolCalls:  []ToolCall{{ID: tc.ID, ...}},  // 重复保存即可
})
```

## 相关提交

- `630ca22` - fix: 修复工具调用ID不匹配问题

## 学习教训

1. 不同 LLM 供应商对 API 规范实现严格程度不同
2. OpenAI 兼容接口不意味着完全兼容
3. 工具调用场景需要兼容性和边界测试