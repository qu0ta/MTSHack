package chats

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
)

type Handler struct {
	service ChatService
}

func NewHandler(service ChatService) *Handler {
	return &Handler{service: service}
}

func matchServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrChatNotFound):
		http.Error(w, ErrChatNotFound.Error(), http.StatusNotFound)
	case errors.Is(err, ErrInvalidChatID):
		http.Error(w, ErrInvalidChatID.Error(), http.StatusBadRequest)
	case errors.Is(err, ErrInvalidPagination):
		http.Error(w, ErrInvalidPagination.Error(), http.StatusBadRequest)
	case errors.Is(err, ErrInvalidTitle):
		http.Error(w, ErrInvalidTitle.Error(), http.StatusBadRequest)
	default:
		log.Printf("internal service error: %v", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
	}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req *CreateChatRequest
	if !decodeChatJSON(w, r, &req) {
		return
	}
	if req == nil {
		http.Error(w, "JSON object is required", http.StatusBadRequest)
		return
	}
	chat, err := h.service.Create(r.Context(), req.Title)
	if err != nil {
		matchServiceError(w, err)
		return
	}
	writeChatJSON(w, http.StatusCreated, chat)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := 20, 0
	query := r.URL.Query()
	if query.Has("limit") {
		value, err := strconv.Atoi(query.Get("limit"))
		if err != nil {
			matchServiceError(w, ErrInvalidPagination)
			return
		}
		limit = value
	}
	if query.Has("offset") {
		value, err := strconv.Atoi(query.Get("offset"))
		if err != nil {
			matchServiceError(w, ErrInvalidPagination)
			return
		}
		offset = value
	}
	if limit < 1 || limit > 100 || offset < 0 {
		matchServiceError(w, ErrInvalidPagination)
		return
	}
	chats, err := h.service.List(r.Context(), limit, offset)
	if err != nil {
		matchServiceError(w, err)
		return
	}
	total, err := h.service.Count(r.Context())
	if err != nil {
		matchServiceError(w, err)
		return
	}
	if chats == nil {
		chats = []Chat{}
	}
	writeChatJSON(w, http.StatusOK, ListChatsResponse{
		Chats: chats, Total: total, Limit: limit, Offset: offset,
	})
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	chat, err := h.service.GetByID(r.Context(), r.PathValue("id"))
	if err != nil {
		matchServiceError(w, err)
		return
	}
	writeChatJSON(w, http.StatusOK, chat)
}

func (h *Handler) UpdateTitle(w http.ResponseWriter, r *http.Request) {
	var req *UpdateChatTitleRequest
	if !decodeChatJSON(w, r, &req) {
		return
	}
	if req == nil || req.Title == nil {
		matchServiceError(w, ErrInvalidTitle)
		return
	}
	chat, err := h.service.UpdateTitle(r.Context(), r.PathValue("id"), *req.Title)
	if err != nil {
		matchServiceError(w, err)
		return
	}
	writeChatJSON(w, http.StatusOK, chat)
}

func (h *Handler) Close(w http.ResponseWriter, r *http.Request) {
	chat, err := h.service.Close(r.Context(), r.PathValue("id"))
	if err != nil {
		matchServiceError(w, err)
		return
	}
	writeChatJSON(w, http.StatusOK, chat)
}

func decodeChatJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(dst)
	if err == nil {
		var extra any
		err = decoder.Decode(&extra)
		if errors.Is(err, io.EOF) {
			return true
		}
	}
	if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
		http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
	} else {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
	}
	return false
}

func writeChatJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("failed to encode chat response: %v", err)
	}
}
