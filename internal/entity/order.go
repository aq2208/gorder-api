package domain

type Status string

const (
	StatusPending    Status = "PENDING"
	StatusProcessing Status = "PROCESSING"
	StatusConfirmed  Status = "CONFIRMED"
	StatusFailed     Status = "FAILED"
)

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
