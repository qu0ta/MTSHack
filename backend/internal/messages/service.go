package messages

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

type MessageRepository interface {
	Create(
		ctx context.Context,
		chatID string,
		role string,
		content string,
		responseTimeMS *int,
		metadata Metadata,
	) (*Message, error)

	ListByChatID(
		ctx context.Context,
		chatID string,
		limit int,
		offset int,
	) ([]Message, error)

	CountByChatID(
		ctx context.Context,
		chatID string,
	) (int64, error)

	ListRecentByChatID(ctx context.Context, chatID string, limit int) ([]Message, error)
}

type MessageService struct {
	repo MessageRepository
}

func NewMessageService(repo MessageRepository) *MessageService {
	return &MessageService{repo: repo}
}

func (m *MessageService) Create(ctx context.Context, chatID string, role string, content string, responseTimeMS *int, metadata Metadata) (*Message, error) {
	if _, err := uuid.Parse(chatID); err != nil {
		return nil, ErrInvalidChatID
	}
	if role != RoleAssistant && role != RoleUser {
		return nil, ErrInvalidRole
	}
	trimmedContent := strings.TrimSpace(content)
	if trimmedContent == "" || utf8.RuneCountInString(trimmedContent) > 10000 {
		return nil, ErrInvalidContent
	}

	if responseTimeMS != nil && *responseTimeMS < 0 {
		return nil, ErrInvalidResponseTime
	}
	return m.repo.Create(ctx, chatID, role, trimmedContent, responseTimeMS, metadata)

}

func (m *MessageService) ListByChatID(ctx context.Context, chatID string, limit int, offset int) ([]Message, error) {
	if _, err := uuid.Parse(chatID); err != nil {
		return nil, ErrInvalidChatID
	}
	if limit < 1 || limit > 100 || offset < 0 {
		return nil, ErrInvalidPagination
	}
	return m.repo.ListByChatID(ctx, chatID, limit, offset)
}

func (m *MessageService) CountByChatID(ctx context.Context, chatID string) (int64, error) {
	if _, err := uuid.Parse(chatID); err != nil {
		return 0, ErrInvalidChatID
	}
	return m.repo.CountByChatID(ctx, chatID)
}
func (m *MessageService) ListRecentByChatID(ctx context.Context, chatID string, limit int) ([]Message, error) {
	if _, err := uuid.Parse(chatID); err != nil {
		return nil, ErrInvalidChatID
	}
	if limit < 1 || limit > 100 {
		return nil, ErrInvalidPagination
	}
	return m.repo.ListRecentByChatID(ctx, chatID, limit)
}
