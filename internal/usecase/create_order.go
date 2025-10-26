package usecase

import (
	"context"
	"errors"

	domain "github.com/aq2208/gorder-api/internal/entity"
	"github.com/google/uuid"
)

type CreateOrderInput struct {
	UserID, IdempotencyKey, Currency, ItemsJSON string
	AmountCents                                 int64
}

type CreateOrderOutput struct {
	OrderID string
	Status  string
}

type CreateOrder struct {
	repo  OrderRepo
	cache OrderCache
	idem  IdempotencyStore
	queue OrderQueue
}

var (
	ErrValidation = errors.New("invalid create order input")
	ErrDuplicate  = errors.New("duplicate idempotency key")
)

func NewCreateOrder(repo OrderRepo, cache OrderCache, idem IdempotencyStore, queue OrderQueue) *CreateOrder {
	return &CreateOrder{repo: repo, idem: idem, queue: queue}
}

// Execute orchestrates: validate -> idempotency -> persist -> enqueue -> return PROCESSING
func (uc *CreateOrder) Execute(ctx context.Context, in CreateOrderInput) (CreateOrderOutput, error) {
	// Input validation
	if in.UserID == "" || in.AmountCents <= 0 || in.Currency == "" || in.ItemsJSON == "" {
		return CreateOrderOutput{}, ErrValidation
	}

	// Idempotency recall
	if in.IdempotencyKey != "" && uc.idem != nil {
		if val, ok, err := uc.idem.Recall(ctx, in.IdempotencyKey); err == nil && ok {
			// Found an existing orderId for this key → return same response (PROCESSING)
			return CreateOrderOutput{
				OrderID: val,
				Status:  string(domain.StatusProcessing),
			}, nil
		}
	}

	// Try lock
	ok, err := uc.idem.TryLock(ctx, in.IdempotencyKey)
	if err != nil {
		return CreateOrderOutput{}, err
	}
	if !ok {
		return CreateOrderOutput{}, ErrDuplicate
	}

	// Build order record and validate
	orderID := uuid.NewString()
	rec := &OrderRecord{
		ID:          orderID,
		UserID:      in.UserID,
		Status:      string(domain.StatusProcessing),
		AmountCents: in.AmountCents,
		Currency:    in.Currency,
		ItemsJSON:   in.ItemsJSON,
	}
	if err := rec.Validate(); err != nil {
		return CreateOrderOutput{}, err
	}

	// Persist
	if err := uc.repo.Create(ctx, rec); err != nil {
		return CreateOrderOutput{}, err
	}

	// Cache
	_ = uc.cache.SetStatus(ctx, orderID, string(domain.StatusProcessing))

	// Enqueue event
	msg := CreatedMsg{
		OrderID:  rec.ID,
		UserID:   rec.UserID,
		Cents:    rec.AmountCents,
		Currency: rec.Currency,
	}
	if err := uc.queue.PublishCreated(ctx, msg); err != nil {
		// At this point, the row exists as PROCESSING.
		// Optionally: update to FAILED or rely on ops/retry/outbox (future improvement).
		return CreateOrderOutput{}, err
	}

	// Remember idempotency key
	if in.IdempotencyKey != "" && uc.idem != nil {
		_ = uc.idem.Remember(ctx, in.IdempotencyKey, orderID)
	}

	return CreateOrderOutput{OrderID: orderID, Status: string(domain.StatusProcessing)}, nil
}
