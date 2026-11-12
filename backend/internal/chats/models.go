package chats

import (
	"errors"
	"time"
)

var (
	ErrChatNotFound      = errors.New("chat not found")
	ErrInvalidChatID     = errors.New("invalid chat id")
	ErrInvalidPagination = errors.New("invalid pagination parameters")
	ErrInvalidTitle      = errors.New("invalid chat title")
)

type Chat struct {
	ID        string     `db:"id" json:"id"`
	Title     *string    `db:"title" json:"title"`
	Status    string     `db:"status" json:"status"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
	ClosedAt  *time.Time `db:"closed_at" json:"closed_at"`
}
