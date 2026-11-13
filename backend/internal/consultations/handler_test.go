package consultations

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/qu0ta/MTSHack/internal/chats"
	"github.com/qu0ta/MTSHack/internal/messages"
	"github.com/qu0ta/MTSHack/internal/mlclient"
)

const validHandlerChatID = "550e8400-e29b-41d4-a716-446655440000"

type consultationSenderStub struct {
	sendFn func(context.Context, string, string) (*SendResult, error)
}

func (s consultationSenderStub) Send(ctx context.Context, chatID, content string) (*SendResult, error) {
	return s.sendFn(ctx, chatID, content)
}

func serveConsultations(sender ConsultationSender, body string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	NewHandler(sender).RegisterRoutes(mux)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/chats/"+validHandlerChatID+"/messages",
		strings.NewReader(body),
	)
	mux.ServeHTTP(recorder, request)
	return recorder
}

func TestHandlerSend(t *testing.T) {
	sender := consultationSenderStub{sendFn: func(_ context.Context, chatID, content string) (*SendResult, error) {
		if chatID != validHandlerChatID || content != "Вопрос" {
			t.Fatalf("arguments = %q, %q", chatID, content)
		}
		return &SendResult{}, nil
	}}

	recorder := serveConsultations(sender, `{"content":"Вопрос"}`)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q", got)
	}
}

func TestHandlerSendRejectsInvalidBody(t *testing.T) {
	sender := consultationSenderStub{sendFn: func(context.Context, string, string) (*SendResult, error) {
		t.Fatal("service must not be called")
		return nil, nil
	}}
	tests := []struct {
		name string
		body string
		want int
	}{
		{name: "empty", body: "", want: http.StatusBadRequest},
		{name: "unknown field", body: `{"content":"ok","role":"assistant"}`, want: http.StatusBadRequest},
		{name: "multiple values", body: `{"content":"one"} {"content":"two"}`, want: http.StatusBadRequest},
		{name: "too large", body: `{"content":"` + strings.Repeat("x", maxRequestBodyBytes) + `"}`, want: http.StatusRequestEntityTooLarge},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			recorder := serveConsultations(sender, tc.body)
			if recorder.Code != tc.want {
				t.Fatalf("status = %d, want %d, body = %s", recorder.Code, tc.want, recorder.Body.String())
			}
		})
	}
}

func TestHandlerSendMapsErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "chat not found", err: chats.ErrChatNotFound, want: http.StatusNotFound},
		{name: "invalid chat", err: chats.ErrInvalidChatID, want: http.StatusBadRequest},
		{name: "invalid content", err: messages.ErrInvalidContent, want: http.StatusBadRequest},
		{name: "closed", err: ErrChatClosed, want: http.StatusConflict},
		{name: "timeout", err: mlclient.ErrTimeout, want: http.StatusGatewayTimeout},
		{name: "unavailable", err: mlclient.ErrUnavailable, want: http.StatusServiceUnavailable},
		{name: "invalid response", err: mlclient.ErrInvalidResponse, want: http.StatusBadGateway},
		{name: "internal", err: errors.New("database unavailable"), want: http.StatusInternalServerError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sender := consultationSenderStub{sendFn: func(context.Context, string, string) (*SendResult, error) {
				return nil, tc.err
			}}
			recorder := serveConsultations(sender, `{"content":"Вопрос"}`)
			if recorder.Code != tc.want {
				t.Fatalf("status = %d, want %d", recorder.Code, tc.want)
			}
		})
	}
}
