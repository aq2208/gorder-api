package usecase

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var ErrDuplicate = errors.New("duplicate idempotency key")

type CreateOrderInput struct {
	UserID, IdempotencyKey, Currency, ItemsJSON string
	AmountCents                                 int64
}

type CreateOrderOutput struct {
	OrderID string
	Status  string
}

type CreateOrder struct {
	repo OrderRepo
	idem IdempotencyStore
	out  OutboxRepo
}

func NewCreateOrder(repo OrderRepo, idem IdempotencyStore, out OutboxRepo) *CreateOrder {
	return &CreateOrder{repo: repo, idem: idem, out: out}
}

func (uc *CreateOrder) Execute(ctx context.Context, in CreateOrderInput) (CreateOrderOutput, error) {
	// Fast path: idempotency recall
	if id, ok, _ := uc.idem.Recall(ctx, in.UserID, in.IdempotencyKey); ok {
		return CreateOrderOutput{OrderID: id, Status: "PENDING"}, nil
	}
	// Attempt to lock
	ok, err := uc.idem.TryLock(ctx, in.UserID, in.IdempotencyKey)
	if err != nil {
		return CreateOrderOutput{}, err
	}
	if !ok {
		return CreateOrderOutput{}, ErrDuplicate
	}

	orderID := uuid.NewString()
	rec := &OrderRecord{
		ID:          orderID,
		UserID:      in.UserID,
		Status:      "PENDING",
		AmountCents: in.AmountCents,
		Currency:    in.Currency,
		ItemsJSON:   in.ItemsJSON,
		// IdempotencyKey: in.IdempotencyKey, // not exported; used by repo Create
	}
	// Create order row
	if err := uc.repo.Create(ctx, rec); err != nil {
		return CreateOrderOutput{}, err
	}

	// Enqueue via outbox (publisher will drain later)
	payload := []byte(`{"type":"OrderCreateCmdV1","order_id":"` + orderID + `"}`)
	_ = uc.out.InsertOrderCreate(ctx, payload)

	_ = uc.idem.Remember(ctx, in.UserID, in.IdempotencyKey, orderID)
	return CreateOrderOutput{OrderID: orderID, Status: "PENDING"}, nil
}
