package messages

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	chatspkg "github.com/qu0ta/MTSHack/internal/chats"
)

const validHandlerChatID = "550e8400-e29b-41d4-a716-446655440000"

type messageServiceStub struct {
	listFn  func(context.Context, string, int, int) ([]Message, error)
	countFn func(context.Context, string) (int64, error)
}

func (s messageServiceStub) ListByChatID(
	ctx context.Context,
	chatID string,
	limit int,
	offset int,
) ([]Message, error) {
	return s.listFn(ctx, chatID, limit, offset)
}

func (s messageServiceStub) CountByChatID(ctx context.Context, chatID string) (int64, error) {
	return s.countFn(ctx, chatID)
}

type chatServiceStub struct {
	getByIDFn func(context.Context, string) (*chatspkg.Chat, error)
}

func (s chatServiceStub) GetByID(ctx context.Context, id string) (*chatspkg.Chat, error) {
	return s.getByIDFn(ctx, id)
}

func activeChatService() chatServiceStub {
	return chatServiceStub{
		getByIDFn: func(_ context.Context, id string) (*chatspkg.Chat, error) {
			return &chatspkg.Chat{ID: id, Status: "active"}, nil
		},
	}
}

func serveMessages(handler *Handler, method, target, body string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(method, target, strings.NewReader(body)))
	return recorder
}

func TestHandlerListByChatID(t *testing.T) {
	service := messageServiceStub{
		listFn: func(_ context.Context, chatID string, limit, offset int) ([]Message, error) {
			if chatID != validHandlerChatID || limit != 10 || offset != 20 {
				t.Fatalf("arguments = %q, %d, %d", chatID, limit, offset)
			}
			return []Message{{ID: "message-id", ChatID: chatID, Role: RoleUser}}, nil
		},
		countFn: func(_ context.Context, chatID string) (int64, error) {
			if chatID != validHandlerChatID {
				t.Fatalf("chatID = %q", chatID)
			}
			return 21, nil
		},
	}

	recorder := serveMessages(
		NewHandler(service, activeChatService()),
		http.MethodGet,
		"/api/v1/chats/"+validHandlerChatID+"/messages?limit=10&offset=20",
		"",
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response listMessagesResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if len(response.Messages) != 1 || response.Total != 21 || response.Limit != 10 || response.Offset != 20 {
		t.Fatalf("response = %#v", response)
	}
}

func TestHandlerListUsesDefaultsAndReturnsEmptyArray(t *testing.T) {
	service := messageServiceStub{
		listFn: func(_ context.Context, _ string, limit, offset int) ([]Message, error) {
			if limit != 50 || offset != 0 {
				t.Fatalf("pagination = %d, %d", limit, offset)
			}
			return nil, nil
		},
		countFn: func(context.Context, string) (int64, error) { return 0, nil },
	}

	recorder := serveMessages(
		NewHandler(service, activeChatService()),
		http.MethodGet,
		"/api/v1/chats/"+validHandlerChatID+"/messages",
		"",
	)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"messages":[]`) {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestHandlerListValidation(t *testing.T) {
	service := messageServiceStub{listFn: func(context.Context, string, int, int) ([]Message, error) {
		t.Fatal("message service must not be called")
		return nil, nil
	}}
	tests := []struct {
		name   string
		target string
		chats  chatServiceStub
		want   int
	}{
		{
			name:   "invalid query",
			target: "/api/v1/chats/" + validHandlerChatID + "/messages?limit=nope",
			chats:  activeChatService(),
			want:   http.StatusBadRequest,
		},
		{
			name:   "chat not found",
			target: "/api/v1/chats/" + validHandlerChatID + "/messages",
			chats: chatServiceStub{
				getByIDFn: func(context.Context, string) (*chatspkg.Chat, error) {
					return nil, chatspkg.ErrChatNotFound
				},
			},
			want: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			recorder := serveMessages(NewHandler(service, tc.chats), http.MethodGet, tc.target, "")
			if recorder.Code != tc.want {
				t.Fatalf("status = %d, want %d", recorder.Code, tc.want)
			}
		})
	}
}

func TestHandlerRoutesRejectWrongMethod(t *testing.T) {
	handler := NewHandler(messageServiceStub{}, activeChatService())
	recorder := serveMessages(
		handler,
		http.MethodDelete,
		"/api/v1/chats/"+validHandlerChatID+"/messages",
		"",
	)
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d", recorder.Code)
	}
}
