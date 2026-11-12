package messages

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	chatspkg "github.com/qu0ta/MTSHack/internal/chats"
)

const maxRequestBodyBytes = 16 << 10

var errChatClosed = errors.New("chat is closed")

type ServiceAPI interface {
	Create(context.Context, string, string, string, *int, Metadata) (*Message, error)
	ListByChatID(context.Context, string, int, int) ([]Message, error)
	CountByChatID(context.Context, string) (int64, error)
}

type ChatService interface {
	GetByID(context.Context, string) (*chatspkg.Chat, error)
	Touch(context.Context, string) error
}

type Handler struct {
	messages ServiceAPI
	chats    ChatService
}

type createMessageRequest struct {
	Content *string `json:"content"`
}

type listMessagesResponse struct {
	Messages []Message `json:"messages"`
	Total    int64     `json:"total"`
	Limit    int       `json:"limit"`
	Offset   int       `json:"offset"`
}

func NewHandler(messages ServiceAPI, chats ChatService) *Handler {
	return &Handler{messages: messages, chats: chats}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/chats/{id}/messages", h.Create)
	mux.HandleFunc("GET /api/v1/chats/{id}/messages", h.ListByChatID)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createMessageRequest
	if err := decodeJSON(w, r, &req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Content == nil {
		writeError(w, ErrInvalidContent)
		return
	}

	chatID := r.PathValue("id")
	chat, err := h.chats.GetByID(r.Context(), chatID)
	if err != nil {
		writeError(w, err)
		return
	}
	if chat.Status == "closed" {
		writeError(w, errChatClosed)
		return
	}

	message, err := h.messages.Create(
		r.Context(),
		chatID,
		RoleUser,
		*req.Content,
		nil,
		Metadata{},
	)
	if err != nil {
		writeError(w, err)
		return
	}

	if err := h.chats.Touch(r.Context(), chatID); err != nil {
		slog.Error("touch chat after creating message", "chat_id", chatID, "error", err)
	}

	writeJSON(w, http.StatusCreated, message)
}

func (h *Handler) ListByChatID(w http.ResponseWriter, r *http.Request) {
	limit, err := parseQueryInt(r, "limit", 50)
	if err != nil {
		writeError(w, ErrInvalidPagination)
		return
	}
	offset, err := parseQueryInt(r, "offset", 0)
	if err != nil {
		writeError(w, ErrInvalidPagination)
		return
	}

	chatID := r.PathValue("id")
	if _, err := h.chats.GetByID(r.Context(), chatID); err != nil {
		writeError(w, err)
		return
	}

	items, err := h.messages.ListByChatID(r.Context(), chatID, limit, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	total, err := h.messages.CountByChatID(r.Context(), chatID)
	if err != nil {
		writeError(w, err)
		return
	}
	if items == nil {
		items = []Message{}
	}

	writeJSON(w, http.StatusOK, listMessagesResponse{
		Messages: items,
		Total:    total,
		Limit:    limit,
		Offset:   offset,
	})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain one JSON value")
	}
	return nil
}

func parseQueryInt(r *http.Request, name string, defaultValue int) (int, error) {
	value := r.URL.Query().Get(name)
	if value == "" {
		return defaultValue, nil
	}
	return strconv.Atoi(value)
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, chatspkg.ErrChatNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, errChatClosed):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, chatspkg.ErrInvalidChatID),
		errors.Is(err, ErrInvalidChatID),
		errors.Is(err, ErrInvalidRole),
		errors.Is(err, ErrInvalidContent),
		errors.Is(err, ErrInvalidPagination),
		errors.Is(err, ErrInvalidResponseTime):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		slog.Error("message request failed", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Error("encode JSON response", "error", err)
	}
}
