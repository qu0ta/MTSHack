package consultations

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/qu0ta/MTSHack/internal/mlclient"
)

// TEST_DATABASE_URL must point to a migrated test database.
func TestPostgresLeadFlow(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL integration")
	}
	ctx := context.Background()
	db, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	newChat := func() string {
		var id string
		if err := db.QueryRow(ctx, "INSERT INTO chats DEFAULT VALUES RETURNING id").Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	first, second := newChat(), newChat()
	defer db.Exec(ctx, "DELETE FROM chats WHERE id=$1 OR id=$2", first, second)
	phone := "+70000000000"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req mlclient.AnswerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			return
		}
		resp := mlclient.AnswerResponse{Answer: "ok", Sources: []string{}, ClientData: req.ClientData}
		switch req.Message {
		case "start":
			if len(req.History) != 0 || req.LeadRequested {
				t.Error("unexpected initial history or confirmation")
			}
			resp.ClientData.Phone = &phone
		case "confirm":
			if req.ClientData.Phone == nil || *req.ClientData.Phone != phone || len(req.History) < 2 || req.History[0].Content != "start" || req.History[1].Content != "ok" || !req.LeadRequested {
				t.Error("lost history, consent or persisted phone")
			}
			resp.Lead = &mlclient.Lead{ContactName: "Anna", CompanyName: "Example", Phone: phone, Summary: "Call"}
		case "separate":
			if req.ClientData.Phone != nil || len(req.History) != 0 {
				t.Error("cross-chat data")
			}
		case "unavailable":
			w.WriteHeader(504)
			return
		case "unsolicited":
			resp.Lead = &mlclient.Lead{}
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()
	client, err := mlclient.New(server.URL, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	s := NewPostgresService(db, client)
	send := func(id, msg string, confirm bool) *SendResult {
		got, err := s.Send(ctx, id, SendRequest{Content: msg, LeadRequested: confirm})
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	if send(first, "start", false).Application != nil || send(second, "separate", false).Application != nil {
		t.Fatal("unexpected application")
	}
	for _, tc := range []struct {
		content string
		err     error
	}{{"unavailable", mlclient.ErrTimeout}, {"unsolicited", mlclient.ErrInvalidResponse}} {
		if _, err := s.Send(ctx, first, SendRequest{Content: tc.content}); !errors.Is(err, tc.err) {
			t.Fatalf("%s: %v", tc.content, err)
		}
	}
	a := send(first, "confirm", true)
	if !a.NeedsManager || a.Application == nil || a.Application.Phone != phone {
		t.Fatalf("lead not saved: %#v", a)
	}
	b := send(first, "confirm", true)
	if b.Application == nil || b.Application.ID != a.Application.ID {
		t.Fatal("duplicate application")
	}
	var count int
	if err := db.QueryRow(ctx, "SELECT count(*) FROM messages WHERE chat_id=$1", first).Scan(&count); err != nil || count != 6 {
		t.Fatalf("messages=%d err=%v", count, err)
	}
	var id string
	if err := db.QueryRow(ctx, "SELECT id FROM applications WHERE chat_id=$1 AND full_name='Anna' AND company='Example' AND summary='Call' AND phone=$2", first, phone).Scan(&id); err != nil || id != a.Application.ID {
		t.Fatalf("stored lead=%s err=%v", id, err)
	}
}
