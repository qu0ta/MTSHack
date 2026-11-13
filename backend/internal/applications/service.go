package applications

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

type Repository interface {
	Upsert(context.Context, string, UpsertInput) (*Application, error)
	GetByChatID(context.Context, string) (*Application, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Upsert(ctx context.Context, chatID string, input UpsertInput) (*Application, error) {
	if _, err := uuid.Parse(chatID); err != nil {
		return nil, ErrInvalidChatID
	}

	input.FullName = strings.TrimSpace(input.FullName)
	input.Phone = strings.TrimSpace(input.Phone)
	input.Company = strings.TrimSpace(input.Company)
	input.Summary = strings.TrimSpace(input.Summary)

	if input.FullName == "" || utf8.RuneCountInString(input.FullName) > 255 {
		return nil, ErrInvalidFullName
	}
	if input.Phone == "" || utf8.RuneCountInString(input.Phone) > 30 {
		return nil, ErrInvalidPhone
	}
	if input.Company == "" || utf8.RuneCountInString(input.Company) > 255 {
		return nil, ErrInvalidCompany
	}
	if input.Summary == "" {
		return nil, ErrInvalidSummary
	}

	return s.repo.Upsert(ctx, chatID, input)
}

func (s *Service) GetByChatID(ctx context.Context, chatID string) (*Application, error) {
	if _, err := uuid.Parse(chatID); err != nil {
		return nil, ErrInvalidChatID
	}
	return s.repo.GetByChatID(ctx, chatID)
}
