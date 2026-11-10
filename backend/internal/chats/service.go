package chats

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

type ChatRepository interface {
	Create(ctx context.Context, title *string) (*Chat, error)
	GetByID(ctx context.Context, id string) (*Chat, error)
	List(ctx context.Context, limit int, offset int) ([]Chat, error)
	Count(ctx context.Context) (int64, error)
	UpdateTitle(ctx context.Context, id string, title string) (*Chat, error)
	Touch(ctx context.Context, id string) error
	Close(ctx context.Context, id string) (*Chat, error)
}
type ChatService interface {
	Create(ctx context.Context, title *string) (*Chat, error)
	GetByID(ctx context.Context, id string) (*Chat, error)
	List(ctx context.Context, limit int, offset int) ([]Chat, error)
	Count(ctx context.Context) (int64, error)
	UpdateTitle(ctx context.Context, id string, title string) (*Chat, error)
	Touch(ctx context.Context, id string) error
	Close(ctx context.Context, id string) (*Chat, error)
}
type Service struct {
	repo ChatRepository
}

func (s *Service) Create(ctx context.Context, title *string) (*Chat, error) {
	if title != nil {
		trimmed := strings.TrimSpace(*title)
		if utf8.RuneCountInString(trimmed) > 255 {
			return nil, ErrInvalidTitle
		}
		if trimmed == "" {
			title = new("")
		} else {
			title = &trimmed
		}
	}
	return s.repo.Create(ctx, title)
}

func (s *Service) GetByID(ctx context.Context, id string) (*Chat, error) {
	_, err := uuid.Parse(id)
	if err != nil {
		return nil, ErrInvalidChatID
	}
	return s.repo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context, limit, offset int) ([]Chat, error) {
	if limit < 1 || limit > 100 || offset < 0 {
		return nil, ErrInvalidPagination
	}
	return s.repo.List(ctx, limit, offset)
}

func (s *Service) Count(ctx context.Context) (int64, error) {
	return s.repo.Count(ctx)
}

func (s *Service) UpdateTitle(ctx context.Context, id, title string) (*Chat, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, ErrInvalidChatID
	}
	trimmed := strings.TrimSpace(title)
	if trimmed == "" || utf8.RuneCountInString(title) > 255 {
		return nil, ErrInvalidTitle
	}
	return s.repo.UpdateTitle(ctx, id, title)
}

func (s *Service) Close(ctx context.Context, id string) (*Chat, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, ErrInvalidChatID
	}
	return s.repo.Close(ctx, id)
}

func (s *Service) Touch(ctx context.Context, id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return ErrInvalidChatID
	}
	return s.repo.Touch(ctx, id)
}

func NewService(repo ChatRepository) *Service {
	return &Service{repo: repo}
}
