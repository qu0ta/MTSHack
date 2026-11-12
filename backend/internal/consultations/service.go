package consultations

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"unicode/utf8"

	"github.com/qu0ta/MTSHack/internal/chats"
	"github.com/qu0ta/MTSHack/internal/messages"
	"github.com/qu0ta/MTSHack/internal/mlclient"
)

const historyLimit = 50

var ErrChatClosed = errors.New("chat is closed")

type ChatService interface {
	GetByID(ctx context.Context, id string) (*chats.Chat, error)
	Touch(ctx context.Context, id string) error
}
type MessageService interface {
	Create(
		ctx context.Context,
		chatID string,
		role string,
		content string,
		responseTimeMS *int,
		metadata messages.Metadata,
	) (*messages.Message, error)

	ListRecentByChatID(
		ctx context.Context,
		chatID string,
		limit int,
	) ([]messages.Message, error)
}
type MLClient interface {
	Answer(
		ctx context.Context,
		request mlclient.AnswerRequest,
	) (*mlclient.AnswerResponse, error)
}
type ConsultationService struct {
	chats    ChatService
	messages MessageService
	ml       MLClient
}
type SendResult struct {
	UserMessage      *messages.Message `json:"user_message"`
	AssistantMessage *messages.Message `json:"assistant_message"`
	NeedsManager     bool              `json:"needs_manager"`
}

func NewConsultationService(
	chatService ChatService,
	messageService MessageService,
	mlClient MLClient,
) *ConsultationService {
	return &ConsultationService{
		chats:    chatService,
		messages: messageService,
		ml:       mlClient,
	}
}
func (s *ConsultationService) Send(ctx context.Context, chatID, content string) (*SendResult, error) {
	content = strings.TrimSpace(content)
	if content == "" || utf8.RuneCountInString(content) > 10000 {
		return nil, messages.ErrInvalidContent
	}

	chat, err := s.chats.GetByID(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("get chat: %w", err)
	}
	if chat.Status == "closed" {
		return nil, ErrChatClosed
	}

	historyMessages, err := s.messages.ListRecentByChatID(
		ctx,
		chatID,
		historyLimit,
	)
	if err != nil {
		return nil, fmt.Errorf("list recent messages: %w", err)
	}

	history := make([]mlclient.HistoryItem, 0, len(historyMessages))
	for _, message := range historyMessages {
		history = append(history, mlclient.HistoryItem{
			Role:    message.Role,
			Content: message.Content,
		})
	}

	mlResponse, err := s.ml.Answer(ctx, mlclient.AnswerRequest{
		SessionID:  chatID,
		Message:    content,
		History:    history,
		ClientData: mlclient.ClientData{},
	})
	if err != nil {
		return nil, fmt.Errorf("get ML answer: %w", err)
	}
	if mlResponse == nil {
		return nil, fmt.Errorf(
			"get ML answer: %w",
			mlclient.ErrInvalidResponse,
		)
	}

	userMessage, err := s.messages.Create(
		ctx,
		chatID,
		messages.RoleUser,
		content,
		nil,
		messages.Metadata{},
	)
	if err != nil {
		return nil, fmt.Errorf("create user message: %w", err)
	}
	sources := make([]messages.Source, 0, len(mlResponse.Sources))
	for _, source := range mlResponse.Sources {
		sources = append(sources, messages.Source{
			Document: source,
		})
	}
	responseTimeMS := mlResponse.ResponseTimeMS

	assistantMessage, err := s.messages.Create(
		ctx,
		chatID,
		messages.RoleAssistant,
		mlResponse.Answer,
		&responseTimeMS,
		messages.Metadata{
			Sources: sources,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create assistant message: %w", err)
	}

	if err := s.chats.Touch(ctx, chatID); err != nil {
		log.Println(
			"touch chat after consultation",
			"chat_id", chatID,
			"error", err,
		)
	}

	return &SendResult{
		UserMessage:      userMessage,
		AssistantMessage: assistantMessage,
		NeedsManager:     mlResponse.Lead != nil,
	}, nil
}
