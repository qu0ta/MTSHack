package messages

import (
	"context"
	"errors"
	"strings"
	"testing"
)

const validChatID = "550e8400-e29b-41d4-a716-446655440000"

type repositoryStub struct {
	createFn        func(context.Context, string, string, string, *int, Metadata) (*Message, error)
	listByChatIDFn  func(context.Context, string, int, int) ([]Message, error)
	countByChatIDFn func(context.Context, string) (int64, error)
}

func (r repositoryStub) Create(
	ctx context.Context,
	chatID string,
	role string,
	content string,
	responseTimeMS *int,
	metadata Metadata,
) (*Message, error) {
	return r.createFn(ctx, chatID, role, content, responseTimeMS, metadata)
}

func (r repositoryStub) ListByChatID(
	ctx context.Context,
	chatID string,
	limit int,
	offset int,
) ([]Message, error) {
	return r.listByChatIDFn(ctx, chatID, limit, offset)
}

func (r repositoryStub) CountByChatID(ctx context.Context, chatID string) (int64, error) {
	return r.countByChatIDFn(ctx, chatID)
}

func TestMessageServiceCreate(t *testing.T) {
	responseTime := 1250
	metadata := Metadata{Sources: []Source{{
		Document: "23-solar.md",
		Chunk:    "installation",
		Score:    0.92,
	}}}

	service := NewMessageService(repositoryStub{
		createFn: func(
			_ context.Context,
			chatID string,
			role string,
			content string,
			responseTimeMS *int,
			gotMetadata Metadata,
		) (*Message, error) {
			if chatID != validChatID {
				t.Fatalf("chatID = %q", chatID)
			}
			if role != RoleAssistant {
				t.Fatalf("role = %q", role)
			}
			if content != "Ответ консультанта" {
				t.Fatalf("content was not trimmed: %q", content)
			}
			if responseTimeMS == nil || *responseTimeMS != responseTime {
				t.Fatalf("responseTimeMS = %v", responseTimeMS)
			}
			if len(gotMetadata.Sources) != 1 || gotMetadata.Sources[0].Document != "23-solar.md" {
				t.Fatalf("metadata = %#v", gotMetadata)
			}
			return &Message{
				ID:             "message-id",
				ChatID:         chatID,
				Role:           role,
				Content:        content,
				ResponseTimeMS: responseTimeMS,
				Metadata:       gotMetadata,
			}, nil
		},
	})

	message, err := service.Create(
		context.Background(),
		validChatID,
		RoleAssistant,
		"  Ответ консультанта  ",
		&responseTime,
		metadata,
	)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if message.Content != "Ответ консультанта" {
		t.Fatalf("Create() content = %q", message.Content)
	}
}

func TestMessageServiceCreateValidation(t *testing.T) {
	tests := []struct {
		name           string
		chatID         string
		role           string
		content        string
		responseTimeMS *int
		wantErr        error
	}{
		{
			name:    "invalid chat id",
			chatID:  "not-a-uuid",
			role:    RoleUser,
			content: "Вопрос",
			wantErr: ErrInvalidChatID,
		},
		{
			name:    "invalid role",
			chatID:  validChatID,
			role:    "system",
			content: "Вопрос",
			wantErr: ErrInvalidRole,
		},
		{
			name:    "blank content",
			chatID:  validChatID,
			role:    RoleUser,
			content: "   ",
			wantErr: ErrInvalidContent,
		},
		{
			name:    "content too long",
			chatID:  validChatID,
			role:    RoleUser,
			content: strings.Repeat("я", 10001),
			wantErr: ErrInvalidContent,
		},
		{
			name:           "negative response time",
			chatID:         validChatID,
			role:           RoleAssistant,
			content:        "Ответ",
			responseTimeMS: new(-1),
			wantErr:        ErrInvalidResponseTime,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			service := NewMessageService(repositoryStub{
				createFn: func(context.Context, string, string, string, *int, Metadata) (*Message, error) {
					called = true
					return nil, nil
				},
			})

			_, err := service.Create(
				context.Background(),
				tc.chatID,
				tc.role,
				tc.content,
				tc.responseTimeMS,
				Metadata{},
			)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Create() error = %v, want %v", err, tc.wantErr)
			}
			if called {
				t.Fatal("repository was called for invalid input")
			}
		})
	}
}

func TestMessageServiceCreatePropagatesRepositoryError(t *testing.T) {
	wantErr := errors.New("database unavailable")
	service := NewMessageService(repositoryStub{
		createFn: func(context.Context, string, string, string, *int, Metadata) (*Message, error) {
			return nil, wantErr
		},
	})

	_, err := service.Create(
		context.Background(),
		validChatID,
		RoleUser,
		"Вопрос",
		nil,
		Metadata{},
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Create() error = %v, want %v", err, wantErr)
	}
}

func TestMessageServiceListByChatID(t *testing.T) {
	service := NewMessageService(repositoryStub{
		listByChatIDFn: func(_ context.Context, chatID string, limit, offset int) ([]Message, error) {
			if chatID != validChatID || limit != 20 || offset != 40 {
				t.Fatalf("arguments = %q, %d, %d", chatID, limit, offset)
			}
			return []Message{{ID: "message-id", ChatID: chatID, Role: RoleUser}}, nil
		},
	})

	messages, err := service.ListByChatID(context.Background(), validChatID, 20, 40)
	if err != nil {
		t.Fatalf("ListByChatID() error = %v", err)
	}
	if len(messages) != 1 || messages[0].ChatID != validChatID {
		t.Fatalf("ListByChatID() = %#v", messages)
	}
}

func TestMessageServiceListByChatIDValidation(t *testing.T) {
	tests := []struct {
		name    string
		chatID  string
		limit   int
		offset  int
		wantErr error
	}{
		{name: "invalid chat id", chatID: "bad", limit: 20, wantErr: ErrInvalidChatID},
		{name: "zero limit", chatID: validChatID, limit: 0, wantErr: ErrInvalidPagination},
		{name: "limit too large", chatID: validChatID, limit: 101, wantErr: ErrInvalidPagination},
		{name: "negative offset", chatID: validChatID, limit: 20, offset: -1, wantErr: ErrInvalidPagination},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			service := NewMessageService(repositoryStub{
				listByChatIDFn: func(context.Context, string, int, int) ([]Message, error) {
					called = true
					return nil, nil
				},
			})

			_, err := service.ListByChatID(context.Background(), tc.chatID, tc.limit, tc.offset)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("ListByChatID() error = %v, want %v", err, tc.wantErr)
			}
			if called {
				t.Fatal("repository was called for invalid input")
			}
		})
	}
}

func TestMessageServiceCountByChatID(t *testing.T) {
	service := NewMessageService(repositoryStub{
		countByChatIDFn: func(_ context.Context, chatID string) (int64, error) {
			if chatID != validChatID {
				t.Fatalf("chatID = %q", chatID)
			}
			return 7, nil
		},
	})

	count, err := service.CountByChatID(context.Background(), validChatID)
	if err != nil || count != 7 {
		t.Fatalf("CountByChatID() = %d, %v", count, err)
	}
}

func TestMessageServiceCountByChatIDRejectsInvalidID(t *testing.T) {
	called := false
	service := NewMessageService(repositoryStub{
		countByChatIDFn: func(context.Context, string) (int64, error) {
			called = true
			return 0, nil
		},
	})

	_, err := service.CountByChatID(context.Background(), "bad")
	if !errors.Is(err, ErrInvalidChatID) {
		t.Fatalf("CountByChatID() error = %v, want %v", err, ErrInvalidChatID)
	}
	if called {
		t.Fatal("repository was called for invalid chat id")
	}
}
