package consultations

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/qu0ta/MTSHack/internal/applications"
	"github.com/qu0ta/MTSHack/internal/chats"
	"github.com/qu0ta/MTSHack/internal/messages"
	"github.com/qu0ta/MTSHack/internal/mlclient"
)

var ErrTurnInProgress = errors.New("chat update in progress")

type PostgresService struct {
	db *pgxpool.Pool
	ml MLClient
}

func NewPostgresService(db *pgxpool.Pool, ml MLClient) *PostgresService {
	return &PostgresService{db, ml}
}

type lockedChat struct {
	chat chats.Chat
	tx   pgx.Tx
}

func (c lockedChat) GetByID(context.Context, string) (*chats.Chat, error) { return &c.chat, nil }
func (c lockedChat) SaveClientData(ctx context.Context, id string, data mlclient.ClientData) error {
	_, err := c.tx.Exec(ctx, "UPDATE chats SET client_data=$2, updated_at=clock_timestamp() WHERE id=$1", id, data)
	return err
}
func (s *PostgresService) Send(ctx context.Context, id string, request SendRequest) (*SendResult, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, chats.ErrInvalidChatID
	}
	ctx, cancel := context.WithTimeout(ctx, 75*time.Second)
	defer cancel()
	var result *SendResult
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		var data mlclient.ClientData
		chat := lockedChat{tx: tx}
		err := tx.QueryRow(ctx, "SELECT status,client_data FROM chats WHERE id=$1 FOR UPDATE NOWAIT", id).Scan(&chat.chat.Status, &data)
		if errors.Is(err, pgx.ErrNoRows) {
			return chats.ErrChatNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "55P03" {
			return ErrTurnInProgress
		}
		if err != nil {
			return err
		}
		turn := newTurn(chat, messages.NewMessageService(messages.NewPostgresRepository(tx)), s.ml)
		turn.clientData = data
		turn.applications = applications.NewService(applications.NewPostgresRepository(tx))
		result, err = turn.Send(ctx, id, request)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
