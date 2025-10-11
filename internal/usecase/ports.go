package usecase

import "context"

// Persistence shape (kept out of domain).
type OrderRecord struct {
	ID, UserID, Status, ItemsJSON, Currency string
	AmountCents                             int64
}

type OrderRepo interface {
	Create(ctx context.Context, o *OrderRecord) error
	GetByID(ctx context.Context, id string) (*OrderRecord, error)
	GetByUserAndIdemKey(ctx context.Context, userID, idemKey string) (*OrderRecord, error)
}

type OutboxRepo interface {
	InsertOrderCreate(ctx context.Context, payload []byte) error
}

type IdempotencyStore interface {
	TryLock(ctx context.Context, scope, key string) (bool, error)
	Remember(ctx context.Context, scope, key, value string) error
	Recall(ctx context.Context, scope, key string) (string, bool, error)
}
