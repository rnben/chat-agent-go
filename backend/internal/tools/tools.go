package tools

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Order 订单
type Order struct {
	OrderID     string        `json:"order_id"`
	UserID      string        `json:"user_id"`
	Status      string        `json:"status"`
	Items       []OrderItem   `json:"items"`
	TotalAmount float64       `json:"total_amount"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
	Timeline    []OrderEvent  `json:"timeline"`
}

// OrderItem 订单项
type OrderItem struct {
	Name     string  `json:"name"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
}

// OrderEvent 订单事件
type OrderEvent struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Note      string    `json:"note"`
}

// OrderStatus 订单状态
const (
	StatusPending   = "pending"   // 待支付
	StatusPaid      = "paid"      // 已支付
	StatusShipped   = "shipped"   // 已发货
	StatusDelivered = "delivered" // 已送达
	StatusCancelled = "cancelled" // 已取消
)

// MockOrderStore Mock 订单存储
type MockOrderStore struct {
	mu     sync.RWMutex
	orders map[string]*Order
}

// NewMockOrderStore 创建 Mock 订单存储
func NewMockOrderStore() *MockOrderStore {
	store := &MockOrderStore{
		orders: make(map[string]*Order),
	}
	store.initMockData()
	return store
}

// 初始化 Mock 数据
func (s *MockOrderStore) initMockData() {
	now := time.Now()

	orders := []*Order{
		{
			OrderID: "ORD-20260515-001",
			UserID:  "user_001",
			Status:  StatusShipped,
			Items: []OrderItem{
				{Name: "MacBook Pro 14寸", Quantity: 1, Price: 14999.00},
				{Name: "AirPods Pro 2", Quantity: 1, Price: 1899.00},
			},
			TotalAmount: 16898.00,
			CreatedAt:   now.Add(-48 * time.Hour),
			UpdatedAt:   now.Add(-2 * time.Hour),
			Timeline: []OrderEvent{
				{Status: StatusPending, Timestamp: now.Add(-48 * time.Hour), Note: "订单创建"},
				{Status: StatusPaid, Timestamp: now.Add(-47 * time.Hour), Note: "支付成功"},
				{Status: StatusShipped, Timestamp: now.Add(-2 * time.Hour), Note: "已由顺丰快递发出，运单号: SF1234567890"},
			},
		},
		{
			OrderID: "ORD-20260514-002",
			UserID:  "user_001",
			Status:  StatusDelivered,
			Items: []OrderItem{
				{Name: "iPhone 16 Pro", Quantity: 1, Price: 8999.00},
			},
			TotalAmount: 8999.00,
			CreatedAt:   now.Add(-72 * time.Hour),
			UpdatedAt:   now.Add(-24 * time.Hour),
			Timeline: []OrderEvent{
				{Status: StatusPending, Timestamp: now.Add(-72 * time.Hour), Note: "订单创建"},
				{Status: StatusPaid, Timestamp: now.Add(-71 * time.Hour), Note: "支付成功"},
				{Status: StatusShipped, Timestamp: now.Add(-60 * time.Hour), Note: "已发货"},
				{Status: StatusDelivered, Timestamp: now.Add(-24 * time.Hour), Note: "已签收"},
			},
		},
		{
			OrderID: "ORD-20260515-003",
			UserID:  "user_002",
			Status:  StatusPending,
			Items: []OrderItem{
				{Name: "iPad Air", Quantity: 2, Price: 4799.00},
			},
			TotalAmount: 9598.00,
			CreatedAt:   now.Add(-1 * time.Hour),
			UpdatedAt:   now.Add(-1 * time.Hour),
			Timeline: []OrderEvent{
				{Status: StatusPending, Timestamp: now.Add(-1 * time.Hour), Note: "订单创建，等待支付"},
			},
		},
	}

	for _, order := range orders {
		s.orders[order.OrderID] = order
	}
}

// QueryOrder 查询订单
func (s *MockOrderStore) QueryOrder(orderID string) (*Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	order, ok := s.orders[orderID]
	if !ok {
		return nil, fmt.Errorf("未找到订单: %s", orderID)
	}
	return order, nil
}

// QueryUserOrders 查询用户订单
func (s *MockOrderStore) QueryUserOrders(userID string) ([]*Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Order
	for _, order := range s.orders {
		if order.UserID == userID {
			result = append(result, order)
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("用户 %s 没有订单", userID)
	}
	return result, nil
}

// HandleQueryOrder 处理订单查询工具调用
func HandleQueryOrder(store *MockOrderStore, argsJSON string) (string, error) {
	var args struct {
		OrderID string `json:"order_id"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("参数解析失败: %w", err)
	}

	order, err := store.QueryOrder(args.OrderID)
	if err != nil {
		return "", err
	}

	statusText := map[string]string{
		StatusPending:   "待支付",
		StatusPaid:      "已支付",
		StatusShipped:   "已发货",
		StatusDelivered: "已送达",
		StatusCancelled: "已取消",
	}

	result := fmt.Sprintf(`订单号: %s
状态: %s
商品: `, order.OrderID, statusText[order.Status])

	for i, item := range order.Items {
		if i > 0 {
			result += ", "
		}
		result += fmt.Sprintf("%s x%d (%.2f元)", item.Name, item.Quantity, item.Price)
	}

	result += fmt.Sprintf("\n总金额: %.2f元", order.TotalAmount)
	result += "\n\n订单时间线:"

	for _, event := range order.Timeline {
		result += fmt.Sprintf("\n- [%s] %s: %s", 
			event.Timestamp.Format("01-02 15:04"), 
			statusText[event.Status], 
			event.Note)
	}

	return result, nil
}

// HandleQueryUserOrders 处理用户订单查询工具调用
func HandleQueryUserOrders(store *MockOrderStore, argsJSON string) (string, error) {
	var args struct {
		UserID string `json:"user_id"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("参数解析失败: %w", err)
	}

	orders, err := store.QueryUserOrders(args.UserID)
	if err != nil {
		return "", err
	}

	statusText := map[string]string{
		StatusPending:   "待支付",
		StatusPaid:      "已支付",
		StatusShipped:   "已发货",
		StatusDelivered: "已送达",
		StatusCancelled: "已取消",
	}

	result := fmt.Sprintf("用户 %s 共有 %d 个订单:\n\n", args.UserID, len(orders))

	for _, order := range orders {
		result += fmt.Sprintf("订单: %s\n  状态: %s\n  金额: %.2f元\n  时间: %s\n\n",
			order.OrderID,
			statusText[order.Status],
			order.TotalAmount,
			order.CreatedAt.Format("2006-01-02 15:04"))
	}

	return result, nil
}
