package tools

import (
	"testing"

	"chat-agent/internal/logger"
)

func init() {
	// 初始化日志，避免 nil pointer
	logger.Init(false)
}

func TestMockOrderStore_InitMockData(t *testing.T) {
	store := NewMockOrderStore()

	if len(store.orders) == 0 {
		t.Error("expected mock data to be initialized")
	}
}

func TestMockOrderStore_QueryOrder(t *testing.T) {
	store := NewMockOrderStore()

	order, err := store.QueryOrder("ORD-20260515-001")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if order.UserID != "user_001" {
		t.Errorf("expected user_id user_001, got %s", order.UserID)
	}
	if order.Status != StatusShipped {
		t.Errorf("expected status shipped, got %s", order.Status)
	}
}

func TestMockOrderStore_QueryOrder_NotFound(t *testing.T) {
	store := NewMockOrderStore()

	_, err := store.QueryOrder("NOT-EXIST")
	if err == nil {
		t.Error("expected error for non-existent order")
	}
}

func TestMockOrderStore_QueryUserOrders(t *testing.T) {
	store := NewMockOrderStore()

	orders, err := store.QueryUserOrders("user_001")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(orders) != 2 {
		t.Errorf("expected 2 orders, got %d", len(orders))
	}
}

func TestMockOrderStore_QueryUserOrders_NotFound(t *testing.T) {
	store := NewMockOrderStore()

	_, err := store.QueryUserOrders("user_not_exist")
	if err == nil {
		t.Error("expected error for non-existent user")
	}
}

func TestHandleQueryOrder(t *testing.T) {
	store := NewMockOrderStore()
	result, err := HandleQueryOrder(store, `{"order_id": "ORD-20260515-001"}`)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == "" {
		t.Error("expected result to not be empty")
	}
	// 验证包含订单号
	if len(result) < 10 {
		t.Errorf("result too short: %s", result)
	}
}

func TestHandleQueryOrder_InvalidArgs(t *testing.T) {
	store := NewMockOrderStore()
	_, err := HandleQueryOrder(store, `invalid json`)

	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestHandleQueryUserOrders(t *testing.T) {
	store := NewMockOrderStore()
	result, err := HandleQueryUserOrders(store, `{"user_id": "user_001"}`)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == "" {
		t.Error("expected result to not be empty")
	}
}