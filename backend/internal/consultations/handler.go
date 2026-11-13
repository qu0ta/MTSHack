package consultations

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/qu0ta/MTSHack/internal/chats"
	"github.com/qu0ta/MTSHack/internal/messages"
	"github.com/qu0ta/MTSHack/internal/mlclient"
)

const maxRequestBodyBytes = 128 << 10

type ConsultationSender interface {
	Send(
		ctx context.Context,
		chatID string,
		request SendRequest,
	) (*SendResult, error)
}

type Handler struct {
	service ConsultationSender
}

type SendRequest struct {
	LeadRequested bool                `json:"lead_requested"`
	ClientData    mlclient.ClientData `json:"client_data"`
	Content       string              `json:"content"`
}

func NewHandler(service ConsultationSender) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/chats/{id}/messages", h.Send)
}

func (h *Handler) Send(w http.ResponseWriter, r *http.Request) {
	var request SendRequest
	if err := decodeJSON(w, r, &request); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	result, err := h.service.Send(r.Context(), r.PathValue("id"), request)
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		slog.Error("encode consultation response", "error", err)
	}
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

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, chats.ErrChatNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, chats.ErrInvalidChatID), errors.Is(err, messages.ErrInvalidContent):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, mlclient.ErrRejected):
		http.Error(w, "invalid consultation data", 422)
	case errors.Is(err, mlclient.ErrBusy):
		w.Header().Set("Retry-After", "2")
		http.Error(w, "ML busy", 503)
	case errors.Is(err, ErrTurnInProgress), errors.Is(err, ErrChatClosed):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, mlclient.ErrTimeout):
		http.Error(w, err.Error(), http.StatusGatewayTimeout)
	case errors.Is(err, mlclient.ErrUnavailable):
		http.Error(w, "ML service unavailable", http.StatusServiceUnavailable)
	case errors.Is(err, mlclient.ErrInvalidResponse):
		http.Error(w, "invalid ML response", http.StatusBadGateway)
	default:
		slog.Error("consultation request failed", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}
