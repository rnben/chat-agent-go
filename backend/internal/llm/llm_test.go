package llm

import (
	"testing"

	"chat-agent/internal/logger"
)

func init() {
	// 初始化日志，避免 nil pointer
	logger.Init(false)
}

func TestClient_GetTools(t *testing.T) {
	client := &Client{}
	tools := client.GetTools()

	if len(tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(tools))
	}

	// 验证 query_order 工具
	if tools[0].Function.Name != "query_order" {
		t.Errorf("expected first tool name query_order, got %s", tools[0].Function.Name)
	}

	// 验证参数定义
	if tools[0].Function.Parameters == nil {
		t.Error("expected parameters to be defined")
	}
}

func TestClient_GetTools_Schema(t *testing.T) {
	client := &Client{}
	tools := client.GetTools()

	// 验证 query_order 有必填参数 order_id
	params := tools[0].Function.Parameters
	props, ok := params.(map[string]interface{})
	if !ok {
		t.Fatal("expected parameters to be map")
	}
	propsMap, ok := props["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("expected properties to be map")
	}
	orderID, ok := propsMap["order_id"].(map[string]interface{})
	if !ok {
		t.Fatal("expected order_id to be map")
	}
	if orderID["type"] != "string" {
		t.Errorf("expected order_id type string, got %v", orderID["type"])
	}

	// 验证 required
	required, ok := props["required"].([]interface{})
	if !ok {
		// try []string
		if reqStr, ok := props["required"].([]string); ok {
			if len(reqStr) != 1 || reqStr[0] != "order_id" {
				t.Errorf("expected required [order_id], got %v", reqStr)
			}
		} else {
			t.Error("expected required to be slice")
		}
	} else if len(required) != 1 || required[0] != "order_id" {
		t.Errorf("expected required [order_id], got %v", required)
	}
}