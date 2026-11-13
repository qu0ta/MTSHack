package mlclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewValidation(t *testing.T) {
	for _, tc := range []struct {
		name    string
		baseURL string
		timeout time.Duration
	}{
		{name: "empty URL", timeout: time.Second},
		{name: "unsupported scheme", baseURL: "ftp://ml", timeout: time.Second},
		{name: "missing host", baseURL: "http://", timeout: time.Second},
		{name: "invalid timeout", baseURL: "http://ml", timeout: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := New(tc.baseURL, tc.timeout); err == nil {
				t.Fatal("New() error = nil")
			}
		})
	}
}

func TestHTTPClientAnswer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != answerPath {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Accept") != "application/json" {
			t.Fatalf("headers = %#v", r.Header)
		}

		var request AnswerRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request.SessionID != "chat-id" || request.Message != "Вопрос" || len(request.History) != 1 {
			t.Fatalf("request body = %#v", request)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"answer":"  Ответ  ",
			"client_data":{},
			"sources":["  22-askue.md  "],
			"lead":{"contact_name":"Анна","phone":"+70000000000","company_name":"ООО Тест","summary":"Нужен учёт"},
			"response_time_ms":1250
		}`))
	}))
	t.Cleanup(server.Close)

	client, err := New(server.URL+"/", time.Second)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	response, err := client.Answer(context.Background(), AnswerRequest{
		SessionID:  "chat-id",
		Message:    "Вопрос",
		History:    []HistoryItem{{Role: "user", Content: "История"}},
		ClientData: ClientData{},
	})
	if err != nil {
		t.Fatalf("Answer() error = %v", err)
	}
	if response.Answer != "Ответ" || response.Lead == nil || response.ResponseTimeMS != 1250 {
		t.Fatalf("Answer() = %#v", response)
	}
	if len(response.Sources) != 1 || response.Sources[0] != "22-askue.md" {
		t.Fatalf("Answer() sources = %#v", response.Sources)
	}
}

func TestHTTPClientAnswerResponseErrors(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr error
	}{
		{
			name:    "non-success status",
			status:  http.StatusInternalServerError,
			body:    `{}`,
			wantErr: ErrUnavailable,
		},
		{name: "invalid JSON", status: http.StatusOK, body: `{`, wantErr: ErrInvalidResponse},
		{
			name:    "negative response time",
			status:  http.StatusOK,
			body:    `{"answer":"Ответ","client_data":{},"sources":[],"lead":null,"response_time_ms":-1}`,
			wantErr: ErrInvalidResponse,
		},
		{
			name:    "negative response time",
			status:  http.StatusOK,
			body:    `{"answer":"Ответ","response_time_ms":-1}`,
			wantErr: ErrInvalidResponse,
		},
		{
			name:    "oversized response",
			status:  http.StatusOK,
			body:    strings.Repeat("x", maxResponseBodyBytes+1),
			wantErr: ErrInvalidResponse,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			t.Cleanup(server.Close)

			client, err := New(server.URL, time.Second)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			_, err = client.Answer(context.Background(), AnswerRequest{})
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Answer() error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestHTTPClientAnswerTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		time.Sleep(50 * time.Millisecond)
	}))
	t.Cleanup(server.Close)

	client, err := New(server.URL, 5*time.Millisecond)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	_, err = client.Answer(context.Background(), AnswerRequest{})
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("Answer() error = %v, want %v", err, ErrTimeout)
	}
}

func TestHTTPClientAnswerContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"answer":"Ответ"}`))
	}))
	t.Cleanup(server.Close)

	client, err := New(server.URL, time.Second)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = client.Answer(ctx, AnswerRequest{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Answer() error = %v, want %v", err, context.Canceled)
	}
}

func TestHTTPClientAnswerRequiresLeadField(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name: "explicit null lead is valid",
			body: `{"answer":"Ответ","client_data":{},"sources":[],"lead":null,"response_time_ms":0}`,
		},
		{
			name:    "missing lead is invalid",
			body:    `{"answer":"Ответ","client_data":{},"sources":[],"response_time_ms":0}`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tc.body))
			}))
			t.Cleanup(server.Close)

			client, err := New(server.URL, time.Second)
			if err != nil {
				t.Fatal(err)
			}

			response, err := client.Answer(context.Background(), AnswerRequest{})
			if tc.wantErr {
				if !errors.Is(err, ErrInvalidResponse) {
					t.Fatalf("Answer() error = %v, want ErrInvalidResponse", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Answer() error = %v", err)
			}
			if response == nil || response.Lead != nil {
				t.Fatalf("expected an answer without a lead, got %#v", response)
			}
		})
	}
}
