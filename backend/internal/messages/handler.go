package messages

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	chatspkg "github.com/qu0ta/MTSHack/internal/chats"
)

type ServiceAPI interface {
	ListByChatID(context.Context, string, int, int) ([]Message, error)
	CountByChatID(context.Context, string) (int64, error)
}

type ChatService interface {
	GetByID(context.Context, string) (*chatspkg.Chat, error)
}

type Handler struct {
	messages ServiceAPI
	chats    ChatService
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
	mux.HandleFunc("GET /api/v1/chats/{id}/messages", h.ListByChatID)
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
