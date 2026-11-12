package messages

import (
	"context"
	"fmt"
	"slices"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func (p *PostgresRepository) ListRecentByChatID(ctx context.Context, chatID string, limit int) ([]Message, error) {
	const query = `SELECT
    id,
    chat_id,
    role,
    content,
    created_at,
    response_time_ms,
    metadata
FROM messages
WHERE chat_id = $1
ORDER BY created_at DESC, id DESC
LIMIT $2;`
	rows, err := p.pool.Query(ctx, query, chatID, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent messages: %w", err)
	}
	messages, err := pgx.CollectRows(rows, pgx.RowToStructByName[Message])
	if err != nil {
		return nil, fmt.Errorf("collect messages: %w", err)
	}

	slices.Reverse(messages)
	return messages, nil

}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}
func (p *PostgresRepository) Create(ctx context.Context, chatID string, role string, content string, responseTimeMS *int, metadata Metadata) (*Message, error) {
	const query = `INSERT INTO messages (
    chat_id,
    role,
    content,
    response_time_ms,
    metadata
)
VALUES ($1, $2, $3, $4, $5)
RETURNING
    id,
    chat_id,
    role,
    content,
    created_at,
    response_time_ms,
    metadata;`
	rows, err := p.pool.Query(ctx, query, chatID, role, content, responseTimeMS, metadata)
	if err != nil {
		return nil, fmt.Errorf("create message: %w", err)
	}
	message, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Message])
	if err != nil {
		return nil, fmt.Errorf("collect created message: %w", err)
	}
	return &message, nil
}

func (p *PostgresRepository) ListByChatID(ctx context.Context, chatID string, limit int, offset int) ([]Message, error) {
	const query = `SELECT
    id,
    chat_id,
    role,
    content,
    created_at,
    response_time_ms,
    metadata
FROM messages
WHERE chat_id = $1
ORDER BY created_at ASC, id ASC
LIMIT $2 OFFSET $3;`
	rows, err := p.pool.Query(ctx, query, chatID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list messages by chat id: %w", err)
	}
	messages, err := pgx.CollectRows(rows, pgx.RowToStructByName[Message])
	if err != nil {
		return nil, fmt.Errorf("collect messages: %w", err)
	}
	return messages, nil
}

func (p *PostgresRepository) CountByChatID(ctx context.Context, chatID string) (int64, error) {
	const query = `
        SELECT COUNT(*)
        FROM messages
        WHERE chat_id = $1;
    `
	var count int64
	if err := p.pool.QueryRow(ctx, query, chatID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count messages by chat id: %w", err)
	}
	return count, nil
}
