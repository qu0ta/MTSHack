package applications

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

func (s *Service) GetHandler(w http.ResponseWriter, r *http.Request) {
	application, err := s.GetByChatID(r.Context(), r.PathValue("id"))
	switch {
	case errors.Is(err, ErrInvalidChatID):
		http.Error(w, "invalid chat id", 400)
	case errors.Is(err, ErrApplicationNotFound):
		http.Error(w, "application not found", 404)
	case err != nil:
		slog.Error("read application", "error", err)
		http.Error(w, "internal server error", 500)
	default:
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(application); err != nil {
			slog.Error("encode application", "error", err)
		}
	}
}
