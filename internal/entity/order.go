package domain

import "errors"

type Status string

const (
	StatusPending    Status = "PENDING"
	StatusProcessing Status = "PROCESSING"
	StatusConfirmed  Status = "CONFIRMED"
	StatusFailed     Status = "FAILED"
)

var ErrInvalidAmount = errors.New("invalid amount")

type Money struct {
	Cents    int64
	Currency string
}

type Order struct {
	ID        string
	UserID    string
	Status    Status
	Amount    Money
	ItemsJSON string // keep simple for now
}

func (o *Order) Validate() error {
	if o.Amount.Cents <= 0 || o.Amount.Currency == "" {
		return ErrInvalidAmount
	}
	return nil
}
