package messages

import (
	"errors"
	"time"
)

const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

var (
	ErrInvalidChatID       = errors.New("invalid chat id")
	ErrInvalidRole         = errors.New("invalid message role")
	ErrInvalidContent      = errors.New("invalid message content")
	ErrInvalidPagination   = errors.New("invalid pagination parameters")
	ErrInvalidResponseTime = errors.New("invalid response time")
)

type Source struct {
	Document string  `json:"document"`
	Chunk    string  `json:"chunk,omitempty"`
	Score    float64 `json:"score,omitempty"`
}

type Metadata struct {
	Sources []Source `json:"sources,omitempty"`
}
type Message struct {
	ID             string    `db:"id" json:"id"`
	ChatID         string    `db:"chat_id" json:"chat_id"`
	Role           string    `db:"role" json:"role"`
	Content        string    `db:"content" json:"content"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	ResponseTimeMS *int      `db:"response_time_ms" json:"response_time_ms"`
	Metadata       Metadata  `db:"metadata" json:"metadata"`
}
