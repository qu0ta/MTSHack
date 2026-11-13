package applications

import (
	"errors"
	"time"
)

const (
	StatusNew       = "new"
	StatusProcessed = "processed"
)

var (
	ErrApplicationNotFound = errors.New("application not found")
	ErrInvalidChatID       = errors.New("invalid chat id")
	ErrInvalidFullName     = errors.New("invalid application full name")
	ErrInvalidPhone        = errors.New("invalid application phone")
	ErrInvalidCompany      = errors.New("invalid application company")
	ErrInvalidSummary      = errors.New("invalid application summary")
)

type Application struct {
	ID        string    `db:"id" json:"id"`
	ChatID    string    `db:"chat_id" json:"chat_id"`
	FullName  string    `db:"full_name" json:"full_name"`
	Phone     string    `db:"phone" json:"phone"`
	Company   string    `db:"company" json:"company"`
	Summary   string    `db:"summary" json:"summary"`
	Status    string    `db:"status" json:"status"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type UpsertInput struct {
	FullName string
	Phone    string
	Company  string
	Summary  string
}
