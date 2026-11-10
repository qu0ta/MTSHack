package chats

import (
	"errors"
	"time"
)

var ErrChatNotFound = errors.New("chat not found")
var ErrInvalidTitle = errors.New("invalid chat title")
var ErrInvalidChatID = errors.New("invalid chat id")
var ErrInvalidPagination = errors.New("invalid pagination parameters")

type Chat struct {
	ID        string     `db:"id" json:"id"`
	Title     *string    `db:"title" json:"title"`
	Status    string     `db:"status" json:"status"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
	ClosedAt  *time.Time `db:"closed_at" json:"closed_at"`
}

type CreateChatRequest struct {
	Title *string `json:"title"`
}

type UpdateChatTitleRequest struct {
	Title *string `json:"title"`
}

type ListChatsResponse struct {
	Chats  []Chat `json:"chats"`
	Total  int64  `json:"total"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}
