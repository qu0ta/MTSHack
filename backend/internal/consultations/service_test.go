package consultations

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/qu0ta/MTSHack/internal/chats"
	"github.com/qu0ta/MTSHack/internal/messages"
	"github.com/qu0ta/MTSHack/internal/mlclient"
)

const validChatID = "550e8400-e29b-41d4-a716-446655440000"

type chatServiceStub struct {
	getByIDFn func(context.Context, string) (*chats.Chat, error)
	touchFn   func(context.Context, string) error
}

func (s chatServiceStub) GetByID(ctx context.Context, id string) (*chats.Chat, error) {
	return s.getByIDFn(ctx, id)
}

func (s chatServiceStub) Touch(ctx context.Context, id string) error {
	if s.touchFn == nil {
		return nil
	}
	return s.touchFn(ctx, id)
}

type messageServiceStub struct {
	createFn     func(context.Context, string, string, string, *int, messages.Metadata) (*messages.Message, error)
	listRecentFn func(context.Context, string, int) ([]messages.Message, error)
}

func (s messageServiceStub) Create(
	ctx context.Context,
	chatID string,
	role string,
	content string,
	responseTimeMS *int,
	metadata messages.Metadata,
) (*messages.Message, error) {
	return s.createFn(ctx, chatID, role, content, responseTimeMS, metadata)
}

func (s messageServiceStub) ListRecentByChatID(
	ctx context.Context,
	chatID string,
	limit int,
) ([]messages.Message, error) {
	return s.listRecentFn(ctx, chatID, limit)
}

type mlClientStub struct {
	answerFn func(context.Context, mlclient.AnswerRequest) (*mlclient.AnswerResponse, error)
}

func (s mlClientStub) Answer(
	ctx context.Context,
	request mlclient.AnswerRequest,
) (*mlclient.AnswerResponse, error) {
	return s.answerFn(ctx, request)
}

func TestConsultationServiceSend(t *testing.T) {
	history := []messages.Message{
		{Role: messages.RoleUser, Content: "Предыдущий вопрос"},
		{Role: messages.RoleAssistant, Content: "Предыдущий ответ"},
	}
	createCalls := 0
	touched := false

	service := NewConsultationService(
		chatServiceStub{
			getByIDFn: func(_ context.Context, id string) (*chats.Chat, error) {
				if id != validChatID {
					t.Fatalf("GetByID() id = %q", id)
				}
				return &chats.Chat{ID: id, Status: "active"}, nil
			},
			touchFn: func(_ context.Context, id string) error {
				touched = true
				if id != validChatID {
					t.Fatalf("Touch() id = %q", id)
				}
				return nil
			},
		},
		messageServiceStub{
			listRecentFn: func(_ context.Context, chatID string, limit int) ([]messages.Message, error) {
				if chatID != validChatID || limit != historyLimit {
					t.Fatalf("ListRecentByChatID() arguments = %q, %d", chatID, limit)
				}
				return history, nil
			},
			createFn: func(
				_ context.Context,
				chatID string,
				role string,
				content string,
				responseTimeMS *int,
				metadata messages.Metadata,
			) (*messages.Message, error) {
				createCalls++
				if chatID != validChatID {
					t.Fatalf("Create() chatID = %q", chatID)
				}
				if createCalls == 1 {
					if role != messages.RoleUser || content != "Новый вопрос" || responseTimeMS != nil {
						t.Fatalf("user Create() = %q, %q, %v", role, content, responseTimeMS)
					}
				} else {
					if role != messages.RoleAssistant || content != "Ответ ML" {
						t.Fatalf("assistant Create() = %q, %q", role, content)
					}
					if responseTimeMS == nil || *responseTimeMS < 0 {
						t.Fatalf("responseTimeMS = %v", responseTimeMS)
					}
					if len(metadata.Sources) != 1 || metadata.Sources[0].Document != "22-askue.md" {
						t.Fatalf("metadata = %#v", metadata)
					}
				}
				return &messages.Message{ID: role + "-id", ChatID: chatID, Role: role, Content: content}, nil
			},
		},
		mlClientStub{answerFn: func(_ context.Context, request mlclient.AnswerRequest) (*mlclient.AnswerResponse, error) {
			if request.SessionID != validChatID || request.Message != "Новый вопрос" {
				t.Fatalf("Answer() request = %#v", request)
			}
			if len(request.History) != 2 || request.History[0].Content != history[0].Content {
				t.Fatalf("Answer() history = %#v", request.History)
			}
			return &mlclient.AnswerResponse{
				Answer:         "Ответ ML",
				Sources:        []string{"22-askue.md"},
				Lead:           &mlclient.Lead{ContactName: "Анна"},
				ResponseTimeMS: 1250,
			}, nil
		}},
	)

	result, err := service.Send(context.Background(), validChatID, "  Новый вопрос  ")
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if result.UserMessage.Role != messages.RoleUser || result.AssistantMessage.Role != messages.RoleAssistant {
		t.Fatalf("Send() result = %#v", result)
	}
	if !result.NeedsManager || createCalls != 2 || !touched {
		t.Fatalf("Send() result = %#v, createCalls = %d, touched = %v", result, createCalls, touched)
	}
}

func TestConsultationServiceSendRejectsInvalidContent(t *testing.T) {
	for _, content := range []string{"   ", strings.Repeat("я", 10001)} {
		service := NewConsultationService(
			chatServiceStub{getByIDFn: func(context.Context, string) (*chats.Chat, error) {
				t.Fatal("chat service must not be called")
				return nil, nil
			}},
			messageServiceStub{},
			mlClientStub{},
		)

		_, err := service.Send(context.Background(), validChatID, content)
		if !errors.Is(err, messages.ErrInvalidContent) {
			t.Fatalf("Send() error = %v, want %v", err, messages.ErrInvalidContent)
		}
	}
}

func TestConsultationServiceSendRejectsClosedChat(t *testing.T) {
	service := NewConsultationService(
		chatServiceStub{getByIDFn: func(context.Context, string) (*chats.Chat, error) {
			return &chats.Chat{ID: validChatID, Status: "closed"}, nil
		}},
		messageServiceStub{},
		mlClientStub{},
	)

	_, err := service.Send(context.Background(), validChatID, "Вопрос")
	if !errors.Is(err, ErrChatClosed) {
		t.Fatalf("Send() error = %v, want %v", err, ErrChatClosed)
	}
}

func TestConsultationServiceSendPropagatesDependencyErrors(t *testing.T) {
	wantErr := errors.New("dependency failed")
	tests := []struct {
		name     string
		chats    ChatService
		messages MessageService
		ml       MLClient
	}{
		{
			name: "get chat",
			chats: chatServiceStub{getByIDFn: func(context.Context, string) (*chats.Chat, error) {
				return nil, wantErr
			}},
			messages: messageServiceStub{},
			ml:       mlClientStub{},
		},
		{
			name: "list history",
			chats: chatServiceStub{getByIDFn: func(context.Context, string) (*chats.Chat, error) {
				return &chats.Chat{Status: "active"}, nil
			}},
			messages: messageServiceStub{listRecentFn: func(context.Context, string, int) ([]messages.Message, error) {
				return nil, wantErr
			}},
			ml: mlClientStub{},
		},
		{
			name: "ML answer",
			chats: chatServiceStub{getByIDFn: func(context.Context, string) (*chats.Chat, error) {
				return &chats.Chat{Status: "active"}, nil
			}},
			messages: messageServiceStub{listRecentFn: func(context.Context, string, int) ([]messages.Message, error) {
				return nil, nil
			}},
			ml: mlClientStub{answerFn: func(context.Context, mlclient.AnswerRequest) (*mlclient.AnswerResponse, error) {
				return nil, wantErr
			}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			service := NewConsultationService(tc.chats, tc.messages, tc.ml)
			_, err := service.Send(context.Background(), validChatID, "Вопрос")
			if !errors.Is(err, wantErr) {
				t.Fatalf("Send() error = %v, want wrapped %v", err, wantErr)
			}
		})
	}
}

func TestConsultationServiceSendRejectsNilMLResponse(t *testing.T) {
	service := NewConsultationService(
		chatServiceStub{getByIDFn: func(context.Context, string) (*chats.Chat, error) {
			return &chats.Chat{Status: "active"}, nil
		}},
		messageServiceStub{listRecentFn: func(context.Context, string, int) ([]messages.Message, error) {
			return nil, nil
		}},
		mlClientStub{answerFn: func(context.Context, mlclient.AnswerRequest) (*mlclient.AnswerResponse, error) {
			return nil, nil
		}},
	)

	_, err := service.Send(context.Background(), validChatID, "Вопрос")
	if !errors.Is(err, mlclient.ErrInvalidResponse) {
		t.Fatalf("Send() error = %v, want %v", err, mlclient.ErrInvalidResponse)
	}
}

func TestConsultationServiceSendPropagatesCreateErrors(t *testing.T) {
	wantErr := errors.New("create failed")
	for _, failAtCall := range []int{1, 2} {
		t.Run(string(rune('0'+failAtCall)), func(t *testing.T) {
			calls := 0
			service := NewConsultationService(
				chatServiceStub{getByIDFn: func(context.Context, string) (*chats.Chat, error) {
					return &chats.Chat{Status: "active"}, nil
				}},
				messageServiceStub{
					listRecentFn: func(context.Context, string, int) ([]messages.Message, error) { return nil, nil },
					createFn: func(_ context.Context, _ string, role, content string, _ *int, _ messages.Metadata) (*messages.Message, error) {
						calls++
						if calls == failAtCall {
							return nil, wantErr
						}
						return &messages.Message{Role: role, Content: content}, nil
					},
				},
				mlClientStub{answerFn: func(context.Context, mlclient.AnswerRequest) (*mlclient.AnswerResponse, error) {
					return &mlclient.AnswerResponse{Answer: "Ответ"}, nil
				}},
			)

			_, err := service.Send(context.Background(), validChatID, "Вопрос")
			if !errors.Is(err, wantErr) {
				t.Fatalf("Send() error = %v, want wrapped %v", err, wantErr)
			}
		})
	}
}
