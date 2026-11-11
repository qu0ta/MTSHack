package chats

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type serviceStub struct {
	createFn      func(context.Context, *string) (*Chat, error)
	getByIDFn     func(context.Context, string) (*Chat, error)
	listFn        func(context.Context, int, int) ([]Chat, error)
	countFn       func(context.Context) (int64, error)
	updateTitleFn func(context.Context, string, string) (*Chat, error)
	closeFn       func(context.Context, string) (*Chat, error)
}

func (s serviceStub) Create(ctx context.Context, title *string) (*Chat, error) {
	return s.createFn(ctx, title)
}

func (s serviceStub) GetByID(ctx context.Context, id string) (*Chat, error) {
	return s.getByIDFn(ctx, id)
}

func (s serviceStub) List(ctx context.Context, limit, offset int) ([]Chat, error) {
	return s.listFn(ctx, limit, offset)
}

func (s serviceStub) Count(ctx context.Context) (int64, error) {
	return s.countFn(ctx)
}

func (s serviceStub) UpdateTitle(ctx context.Context, id, title string) (*Chat, error) {
	return s.updateTitleFn(ctx, id, title)
}

func (s serviceStub) Close(ctx context.Context, id string) (*Chat, error) {
	return s.closeFn(ctx, id)
}

func TestHandlerCreate(t *testing.T) {
	service := serviceStub{createFn: func(_ context.Context, title *string) (*Chat, error) {
		if title == nil || *title != "Consultation" {
			t.Fatalf("unexpected title: %v", title)
		}
		return &Chat{ID: validChatID, Title: title, Status: "active"}, nil
	}}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/chats", strings.NewReader(`{"title":"Consultation"}`))

	NewHandler(service).Create(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q", got)
	}
}

func TestHandlerCreateRejectsInvalidJSON(t *testing.T) {
	service := serviceStub{createFn: func(context.Context, *string) (*Chat, error) {
		t.Fatal("service must not be called")
		return nil, nil
	}}
	for _, body := range []string{``, `{"unknown":true}`, `{} {}`} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/v1/chats", strings.NewReader(body))
		NewHandler(service).Create(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("body %q: status = %d", body, recorder.Code)
		}
	}
}

func TestHandlerList(t *testing.T) {
	service := serviceStub{
		listFn: func(_ context.Context, limit, offset int) ([]Chat, error) {
			if limit != 10 || offset != 20 {
				t.Fatalf("pagination = %d, %d", limit, offset)
			}
			return []Chat{{ID: validChatID, Status: "active"}}, nil
		},
		countFn: func(context.Context) (int64, error) { return 31, nil },
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/chats?limit=10&offset=20", nil)

	NewHandler(service).List(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response listChatsResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if len(response.Chats) != 1 || response.Total != 31 || response.Limit != 10 || response.Offset != 20 {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestHandlerListUsesDefaultsAndReturnsEmptyArray(t *testing.T) {
	service := serviceStub{
		listFn: func(_ context.Context, limit, offset int) ([]Chat, error) {
			if limit != 20 || offset != 0 {
				t.Fatalf("pagination = %d, %d", limit, offset)
			}
			return nil, nil
		},
		countFn: func(context.Context) (int64, error) { return 0, nil },
	}
	recorder := httptest.NewRecorder()
	NewHandler(service).List(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/chats", nil))

	if !strings.Contains(recorder.Body.String(), `"chats":[]`) {
		t.Fatalf("body = %s", recorder.Body.String())
	}
}

func TestHandlerListRejectsInvalidQuery(t *testing.T) {
	service := serviceStub{listFn: func(context.Context, int, int) ([]Chat, error) {
		t.Fatal("service must not be called")
		return nil, nil
	}}
	recorder := httptest.NewRecorder()
	NewHandler(service).List(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/chats?limit=nope", nil))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", recorder.Code)
	}
}

func TestHandlerGetByID(t *testing.T) {
	service := serviceStub{getByIDFn: func(_ context.Context, id string) (*Chat, error) {
		return &Chat{ID: id, Status: "active"}, nil
	}}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/chats/"+validChatID, nil)
	request.SetPathValue("id", validChatID)

	NewHandler(service).GetByID(recorder, request)

	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), validChatID) {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestHandlerUpdateTitle(t *testing.T) {
	service := serviceStub{updateTitleFn: func(_ context.Context, id, title string) (*Chat, error) {
		if id != validChatID || title != "New title" {
			t.Fatalf("arguments = %q, %q", id, title)
		}
		return &Chat{ID: id, Title: &title, Status: "active"}, nil
	}}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/chats/"+validChatID, strings.NewReader(`{"title":"New title"}`))
	request.SetPathValue("id", validChatID)

	NewHandler(service).UpdateTitle(recorder, request)

	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "New title") {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestHandlerUpdateTitleRequiresTitle(t *testing.T) {
	service := serviceStub{updateTitleFn: func(context.Context, string, string) (*Chat, error) {
		t.Fatal("service must not be called")
		return nil, nil
	}}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/chats/"+validChatID, strings.NewReader(`{}`))
	request.SetPathValue("id", validChatID)

	NewHandler(service).UpdateTitle(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", recorder.Code)
	}
}

func TestHandlerClose(t *testing.T) {
	service := serviceStub{closeFn: func(_ context.Context, id string) (*Chat, error) {
		return &Chat{ID: id, Status: "closed"}, nil
	}}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/chats/"+validChatID+"/close", nil)
	request.SetPathValue("id", validChatID)

	NewHandler(service).Close(recorder, request)

	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"status":"closed"`) {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestHandlerMapsServiceErrors(t *testing.T) {
	internalErr := errors.New("database unavailable")
	tests := []struct {
		name   string
		err    error
		status int
	}{
		{name: "not found", err: ErrChatNotFound, status: http.StatusNotFound},
		{name: "invalid id", err: ErrInvalidChatID, status: http.StatusBadRequest},
		{name: "internal", err: internalErr, status: http.StatusInternalServerError},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			service := serviceStub{getByIDFn: func(context.Context, string) (*Chat, error) {
				return nil, tc.err
			}}
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/api/v1/chats/id", nil)
			request.SetPathValue("id", "id")

			NewHandler(service).GetByID(recorder, request)

			if recorder.Code != tc.status {
				t.Fatalf("status = %d, want %d", recorder.Code, tc.status)
			}
		})
	}
}

func TestHandlerRoutes(t *testing.T) {
	service := serviceStub{getByIDFn: func(_ context.Context, id string) (*Chat, error) {
		return &Chat{ID: id, Status: "active"}, nil
	}}
	mux := http.NewServeMux()
	NewHandler(service).RegisterRoutes(mux)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/chats/"+validChatID, nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}
