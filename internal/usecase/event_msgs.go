package usecase

// Sent by order-gw on Kafka
type OrderStatusChangedMsg struct {
	OrderID  string `json:"orderId"`
	UserID   string `json:"userId"`
	Cents    int64  `json:"cents"`
	Currency string `json:"currency"`
	Status   string `json:"status"` // e.g. "SUCCESS"
}
