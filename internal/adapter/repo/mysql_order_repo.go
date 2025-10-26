package repo

import (
	"context"
	"database/sql"
	"errors"

	"github.com/aq2208/gorder-api/internal/usecase"
)

type MySQLOrderRepo struct{ db *sql.DB }

func (r *MySQLOrderRepo) UpdateStatusIf(ctx context.Context, id string, fromStatus, toStatus string) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
        UPDATE orders 
        SET status = ?, updated_at = NOW()
        WHERE id = ? AND status = ?`,
		toStatus, id, fromStatus,
	)
	if err != nil {
		return false, err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	// rows == 0 → nothing matched (either not found or status mismatch)
	return rows > 0, nil
}

func (r *MySQLOrderRepo) UpdateStatus(ctx context.Context, id, toStatus string) error {
	res, err := r.db.ExecContext(ctx, `
        UPDATE orders 
        SET status = ?, updated_at = NOW()
        WHERE id = ?`,
		toStatus, id,
	)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func NewMySQLOrderRepo(db *sql.DB) *MySQLOrderRepo { return &MySQLOrderRepo{db: db} }

func (r *MySQLOrderRepo) Create(ctx context.Context, o *usecase.OrderRecord) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO orders (id,user_id,status,amount_cents,currency,items_json,idempotency_key,version,created_at,updated_at)
VALUES (?,?,?,?,?,?,?,0,NOW(),NOW())
`, o.ID, o.UserID, o.Status, o.AmountCents, o.Currency, o.ItemsJSON, o.IdempotencyKey)
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
