package chats

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
)

const maxRequestBodyBytes = 16 << 10

type Handler struct {
	service ChatService
}

type createChatRequest struct {
	Title *string `json:"title"`
}

type updateTitleRequest struct {
	Title *string `json:"title"`
}

type listChatsResponse struct {
	Chats  []Chat `json:"chats"`
	Total  int64  `json:"total"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

func NewHandler(service ChatService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/chats", h.Create)
	mux.HandleFunc("GET /api/v1/chats", h.List)
	mux.HandleFunc("GET /api/v1/chats/{id}", h.GetByID)
	mux.HandleFunc("PATCH /api/v1/chats/{id}", h.UpdateTitle)
	mux.HandleFunc("POST /api/v1/chats/{id}/close", h.Close)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createChatRequest
	if err := decodeJSON(w, r, &req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	chat, err := h.service.Create(r.Context(), req.Title)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, chat)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	limit, err := parseQueryInt(r, "limit", 20)
	if err != nil {
		http.Error(w, ErrInvalidPagination.Error(), http.StatusBadRequest)
		return
	}
	offset, err := parseQueryInt(r, "offset", 0)
	if err != nil {
		http.Error(w, ErrInvalidPagination.Error(), http.StatusBadRequest)
		return
	}

	items, err := h.service.List(r.Context(), limit, offset)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	total, err := h.service.Count(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if items == nil {
		items = []Chat{}
	}

	writeJSON(w, http.StatusOK, listChatsResponse{
		Chats:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	chat, err := h.service.GetByID(r.Context(), r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, chat)
}

func (h *Handler) UpdateTitle(w http.ResponseWriter, r *http.Request) {
	var req updateTitleRequest
	if err := decodeJSON(w, r, &req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Title == nil {
		writeServiceError(w, ErrInvalidTitle)
		return
	}

	chat, err := h.service.UpdateTitle(r.Context(), r.PathValue("id"), *req.Title)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, chat)
}

func (h *Handler) Close(w http.ResponseWriter, r *http.Request) {
	chat, err := h.service.Close(r.Context(), r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, chat)
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

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrChatNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, ErrInvalidChatID),
		errors.Is(err, ErrInvalidPagination),
		errors.Is(err, ErrInvalidTitle):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		slog.Error("chat request failed", "error", err)
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
