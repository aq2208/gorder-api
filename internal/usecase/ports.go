package usecase

import (
	"context"
	"errors"
)

// OrderRecord - Persistence shape (kept out of domain).
type OrderRecord struct {
	ID, UserID, Status, ItemsJSON, Currency string
	AmountCents                             int64
	IdempotencyKey                          string
}

var ErrInvalidAmount = errors.New("invalid amount")

func (rec *OrderRecord) Validate() error {
	if rec.AmountCents <= 0 || rec.Currency == "" {
		return ErrInvalidAmount
	}
	return nil
}

type OrderRepo interface {
	Create(ctx context.Context, o *OrderRecord) error
	UpdateStatus(ctx context.Context, id, toStatus string) error
	UpdateStatusIf(ctx context.Context, id string, fromStatus, toStatus string) (bool, error)
	GetByID(ctx context.Context, id string) (*OrderRecord, error)
	GetByUserAndIdemKey(ctx context.Context, userID, idemKey string) (*OrderRecord, error)
}

type OrderCache interface {
	SetStatus(ctx context.Context, orderID string, status string) error
	GetStatus(ctx context.Context, orderID string) (string, error)
}

type OutboxRepo interface {
	InsertOrderCreate(ctx context.Context, payload []byte) error
}

type IdempotencyStore interface {
	TryLock(ctx context.Context, key string) (bool, error)
	Remember(ctx context.Context, key, value string) error
	Recall(ctx context.Context, key string) (string, bool, error)
}

type OrderQueue interface {
	PublishCreated(ctx context.Context, msg CreatedMsg) error
}

type CreatedMsg struct {
	OrderID  string `json:"orderId"`
	UserID   string `json:"userId"`
	Cents    int64  `json:"cents"`
	Currency string `json:"currency"`
}
