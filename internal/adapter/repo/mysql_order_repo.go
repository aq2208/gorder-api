package repo

import (
	"context"
	"database/sql"
	"errors"

	"github.com/aq2208/gorder-api/internal/usecase"
)

type MySQLOrderRepo struct{ db *sql.DB }

func (r *MySQLOrderRepo) UpdateStatus(ctx context.Context, id, toStatus string) error {
	//TODO implement me
	panic("implement me")
}

func NewMySQLOrderRepo(db *sql.DB) *MySQLOrderRepo { return &MySQLOrderRepo{db: db} }

func (r *MySQLOrderRepo) Create(ctx context.Context, o *usecase.OrderRecord) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO orders (id,user_id,status,amount_cents,currency,items_json,idempotency_key,version,created_at,updated_at)
VALUES (?,?,?,?,?,?,?,0,NOW(),NOW())
`, o.ID, o.UserID, o.Status, o.AmountCents, o.Currency, o.ItemsJSON)
	return err
}

func (r *MySQLOrderRepo) GetByID(ctx context.Context, id string) (*usecase.OrderRecord, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id,user_id,status,amount_cents,currency,items_json,idempotency_key
FROM orders WHERE id=?`, id)
	var rec usecase.OrderRecord
	if err := row.Scan(&rec.ID, &rec.UserID, &rec.Status, &rec.AmountCents, &rec.Currency, &rec.ItemsJSON); err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *MySQLOrderRepo) GetByUserAndIdemKey(ctx context.Context, userID, idemKey string) (*usecase.OrderRecord, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id,user_id,status,amount_cents,currency,items_json,idempotency_key
FROM orders WHERE user_id=? AND idempotency_key=?`, userID, idemKey)
	var rec usecase.OrderRecord
	if err := row.Scan(&rec.ID, &rec.UserID, &rec.Status, &rec.AmountCents, &rec.Currency, &rec.ItemsJSON); err != nil {
		return nil, err
	}
	return &rec, nil
}

var _ usecase.OrderRepo = (*MySQLOrderRepo)(nil)

var ErrNotFound = errors.New("not found")
