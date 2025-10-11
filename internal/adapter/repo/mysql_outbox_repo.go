package repo

import (
	"context"
	"database/sql"

	"github.com/aq2208/gorder-api/internal/usecase"
)

type MySQLOutboxRepo struct{ db *sql.DB }

func NewMySQLOutboxRepo(db *sql.DB) *MySQLOutboxRepo { return &MySQLOutboxRepo{db: db} }

func (r *MySQLOutboxRepo) InsertOrderCreate(ctx context.Context, payload []byte) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO outbox (channel,payload,status,retry_count,next_attempt_at,created_at)
VALUES ('orders.create.v1', ?, 'PENDING', 0, NOW(), NOW())
`, payload)
	return err
}

var _ usecase.OutboxRepo = (*MySQLOutboxRepo)(nil)
