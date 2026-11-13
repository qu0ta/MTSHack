package applications

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

type PostgresRepository struct {
	pool queryer
}

func NewPostgresRepository(pool queryer) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Upsert(ctx context.Context, chatID string, input UpsertInput) (*Application, error) {
	const query = `INSERT INTO applications (
    chat_id,
    full_name,
    phone,
    company,
    summary
)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (chat_id) DO UPDATE SET
    full_name = EXCLUDED.full_name,
    phone = EXCLUDED.phone,
    company = EXCLUDED.company,
    summary = EXCLUDED.summary
RETURNING
    id,
    chat_id,
    full_name,
    phone,
    company,
    summary,
    status,
    created_at;`

	rows, err := r.pool.Query(ctx, query, chatID, input.FullName, input.Phone, input.Company, input.Summary)
	if err != nil {
		return nil, fmt.Errorf("upsert application: %w", err)
	}
	application, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Application])
	if err != nil {
		return nil, fmt.Errorf("collect upserted application: %w", err)
	}
	return &application, nil
}

func (r *PostgresRepository) GetByChatID(ctx context.Context, chatID string) (*Application, error) {
	const query = `SELECT
    id,
    chat_id,
    full_name,
    phone,
    company,
    summary,
    status,
    created_at
FROM applications
WHERE chat_id = $1;`

	rows, err := r.pool.Query(ctx, query, chatID)
	if err != nil {
		return nil, fmt.Errorf("get application by chat id: %w", err)
	}
	application, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Application])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrApplicationNotFound
		}
		return nil, fmt.Errorf("collect application by chat id: %w", err)
	}
	return &application, nil
}
