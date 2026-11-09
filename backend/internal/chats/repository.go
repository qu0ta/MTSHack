package chats

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, title *string) (*Chat, error) {
	const query = `
		INSERT INTO chats (title)
		VALUES ($1)
		RETURNING id, title, status, created_at, updated_at, closed_at
	`
	rows, err := r.pool.Query(ctx, query, title)
	if err != nil {
		return nil, fmt.Errorf("create chat: %w", err)
	}
	chat, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Chat])
	if err != nil {
		return nil, fmt.Errorf("create chat: %w", err)
	}
	return &chat, nil

}

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (*Chat, error) {
	const query = `SELECT id, title, status, created_at, updated_at, closed_at
					FROM chats
					WHERE id = $1;`
	rows, err := r.pool.Query(ctx, query, id)
	if err != nil {
		return nil, fmt.Errorf("get chat by id: %w", err)
	}
	chat, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Chat])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrChatNotFound
		}

		return nil, fmt.Errorf("get chat by id: %w", err)
	}
	return &chat, nil
}

func (r *PostgresRepository) List(ctx context.Context, limit int, offset int) ([]Chat, error) {
	const query = `SELECT id, title, status, created_at, updated_at, closed_at
FROM chats
ORDER BY updated_at DESC, id DESC
LIMIT $1 OFFSET $2;`
	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list chats: %w", err)
	}
	chats, err := pgx.CollectRows(rows, pgx.RowToStructByName[Chat])
	if err != nil {
		return nil, fmt.Errorf("collect chats: %w", err)
	}
	return chats, nil
}

func (r *PostgresRepository) Count(ctx context.Context) (int64, error) {
	const query = `SELECT COUNT(*) FROM chats;`
	var count int64
	err := r.pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count chats: %w", err)
	}
	return count, nil

}

func (r *PostgresRepository) UpdateTitle(ctx context.Context, id string, title string) (*Chat, error) {
	const query = `UPDATE chats
SET title = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING id, title, status, created_at, updated_at, closed_at;`
	rows, err := r.pool.Query(ctx, query, id, title)
	if err != nil {
		return nil, fmt.Errorf("update chat title: %w", err)
	}
	chat, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Chat])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrChatNotFound
		}
		return nil, fmt.Errorf("update chat title: %w", err)

	}
	return &chat, nil
}

func (r *PostgresRepository) Touch(ctx context.Context, id string) error {
	const query = `UPDATE chats
SET updated_at = NOW()
WHERE id = $1;`
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("touch chat: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrChatNotFound
	}
	return nil
}

func (r *PostgresRepository) Close(ctx context.Context, id string) (*Chat, error) {
	const query = `UPDATE chats
SET status = 'closed',
    closed_at = COALESCE(closed_at, NOW()),
    updated_at = CASE
        WHEN status = 'closed' THEN updated_at
        ELSE NOW()
    END
WHERE id = $1
RETURNING id, title, status, created_at, updated_at, closed_at;`
	rows, err := r.pool.Query(ctx, query, id)
	if err != nil {
		return nil, fmt.Errorf("close chat: %w", err)
	}
	chat, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Chat])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrChatNotFound
		}
		return nil, fmt.Errorf("close chat: %w", err)

	}
	return &chat, nil

}
