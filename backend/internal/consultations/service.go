package consultations

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/qu0ta/MTSHack/internal/applications"
	"github.com/qu0ta/MTSHack/internal/chats"
	"github.com/qu0ta/MTSHack/internal/messages"
	"github.com/qu0ta/MTSHack/internal/mlclient"
)

const historyLimit = 50

var ErrChatClosed = errors.New("chat is closed")

type ChatService interface {
	GetByID(ctx context.Context, id string) (*chats.Chat, error)
	SaveClientData(ctx context.Context, id string, data mlclient.ClientData) error
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
type turn struct {
	applications *applications.Service
	clientData   mlclient.ClientData
	chats        ChatService
	messages     MessageService
	ml           MLClient
}
type SendResult struct {
	UserMessage      *messages.Message         `json:"user_message"`
	AssistantMessage *messages.Message         `json:"assistant_message"`
	Application      *applications.Application `json:"application,omitempty"`
	NeedsManager     bool                      `json:"needs_manager"`
}

func newTurn(
	chatService ChatService,
	messageService MessageService,
	mlClient MLClient,
) *turn {
	return &turn{
		chats:    chatService,
		messages: messageService,
		ml:       mlClient,
	}
}
func (s *turn) Send(ctx context.Context, chatID string, request SendRequest) (*SendResult, error) {
	content := strings.TrimSpace(request.Content)
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

	applyClientData(&s.clientData, request.ClientData)
	if !s.clientData.Valid() {
		return nil, mlclient.ErrRejected
	}
	mlResponse, err := s.ml.Answer(ctx, mlclient.AnswerRequest{
		SessionID:     chatID,
		Message:       content,
		History:       history,
		ClientData:    s.clientData,
		LeadRequested: request.LeadRequested,
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

	if !mlResponse.ClientData.Valid() || (mlResponse.Lead != nil && !request.LeadRequested) {
		return nil, mlclient.ErrInvalidResponse
	}
	applyClientData(&s.clientData, mlResponse.ClientData)
	var application *applications.Application
	if lead := mlResponse.Lead; lead != nil {
		application, err = s.applications.Upsert(ctx, chatID, applications.UpsertInput{FullName: lead.ContactName, Phone: lead.Phone, Company: lead.CompanyName, Summary: lead.Summary})
		if err != nil {
			if errors.Is(err, applications.ErrInvalidFullName) || errors.Is(err, applications.ErrInvalidCompany) || errors.Is(err, applications.ErrInvalidPhone) || errors.Is(err, applications.ErrInvalidSummary) {
				return nil, mlclient.ErrInvalidResponse
			}
			return nil, err
		}
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

	if err := s.chats.SaveClientData(ctx, chatID, s.clientData); err != nil {
		return nil, err
	}

	return &SendResult{
		UserMessage:      userMessage,
		AssistantMessage: assistantMessage,
		NeedsManager:     application != nil,
		Application:      application,
	}, nil
}

func applyClientData(data *mlclient.ClientData, update mlclient.ClientData) {
	if update.CompanyName != nil {
		data.CompanyName = update.CompanyName
	}
	if update.ContactName != nil {
		data.ContactName = update.ContactName
	}
	if update.Phone != nil {
		data.Phone = update.Phone
	}
	if update.Region != nil {
		data.Region = update.Region
	}
	if update.Industry != nil {
		data.Industry = update.Industry
	}
	if update.ConnectedPowerKW != nil {
		data.ConnectedPowerKW = update.ConnectedPowerKW
	}
	if update.MonthlyConsumptionKWH != nil {
		data.MonthlyConsumptionKWH = update.MonthlyConsumptionKWH
	}
}
