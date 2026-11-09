package chats

import (
	"context"
	"strings"
	"unicode/utf8"
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
	List(ctx context.Context, limit, offset int) ([]Chat, error)
	UpdateTitle(ctx context.Context, id, title string) (*Chat, error)
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
	//TODO implement me
	panic("implement me")
}

func (s *Service) List(ctx context.Context, limit, offset int) ([]Chat, error) {
	//TODO implement me
	panic("implement me")
}

func (s *Service) UpdateTitle(ctx context.Context, id, title string) (*Chat, error) {
	//TODO implement me
	panic("implement me")
}

func (s *Service) Close(ctx context.Context, id string) (*Chat, error) {
	//TODO implement me
	panic("implement me")
}

func NewService(repo ChatRepository) *Service {
	return &Service{repo: repo}
}
