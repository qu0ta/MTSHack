package chats

import (
	"context"
	"errors"
	"strings"
	"testing"
)

const validChatID = "550e8400-e29b-41d4-a716-446655440000"

type repositoryStub struct {
	createFn      func(context.Context, *string) (*Chat, error)
	getByIDFn     func(context.Context, string) (*Chat, error)
	listFn        func(context.Context, int, int) ([]Chat, error)
	countFn       func(context.Context) (int64, error)
	updateTitleFn func(context.Context, string, string) (*Chat, error)
	touchFn       func(context.Context, string) error
	closeFn       func(context.Context, string) (*Chat, error)
}

func (r repositoryStub) Create(ctx context.Context, title *string) (*Chat, error) {
	return r.createFn(ctx, title)
}

func (r repositoryStub) GetByID(ctx context.Context, id string) (*Chat, error) {
	return r.getByIDFn(ctx, id)
}

func (r repositoryStub) List(ctx context.Context, limit, offset int) ([]Chat, error) {
	return r.listFn(ctx, limit, offset)
}

func (r repositoryStub) Count(ctx context.Context) (int64, error) {
	return r.countFn(ctx)
}

func (r repositoryStub) UpdateTitle(ctx context.Context, id, title string) (*Chat, error) {
	return r.updateTitleFn(ctx, id, title)
}

func (r repositoryStub) Touch(ctx context.Context, id string) error {
	return r.touchFn(ctx, id)
}

func (r repositoryStub) Close(ctx context.Context, id string) (*Chat, error) {
	return r.closeFn(ctx, id)
}

func TestServiceCreate(t *testing.T) {
	t.Run("nil title", func(t *testing.T) {
		service := NewService(repositoryStub{createFn: func(_ context.Context, title *string) (*Chat, error) {
			if title != nil {
				t.Fatalf("expected nil title, got %q", *title)
			}
			return &Chat{ID: validChatID}, nil
		}})

		if _, err := service.Create(context.Background(), nil); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	})

	t.Run("trims title", func(t *testing.T) {
		title := "  Новый чат  "
		service := NewService(repositoryStub{createFn: func(_ context.Context, got *string) (*Chat, error) {
			if got == nil || *got != "Новый чат" {
				t.Fatalf("expected trimmed title, got %v", got)
			}
			return &Chat{ID: validChatID, Title: got}, nil
		}})

		if _, err := service.Create(context.Background(), &title); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	})

	t.Run("allows blank title", func(t *testing.T) {
		title := "   "
		service := NewService(repositoryStub{createFn: func(_ context.Context, got *string) (*Chat, error) {
			if got == nil || *got != "" {
				t.Fatalf("expected empty title, got %v", got)
			}
			return &Chat{ID: validChatID, Title: got}, nil
		}})

		if _, err := service.Create(context.Background(), &title); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	})

	t.Run("rejects long unicode title", func(t *testing.T) {
		title := strings.Repeat("я", 256)
		service := NewService(repositoryStub{createFn: func(context.Context, *string) (*Chat, error) {
			t.Fatal("repository must not be called")
			return nil, nil
		}})

		_, err := service.Create(context.Background(), &title)
		if !errors.Is(err, ErrInvalidTitle) {
			t.Fatalf("expected ErrInvalidTitle, got %v", err)
		}
	})
}

func TestServiceGetByID(t *testing.T) {
	service := NewService(repositoryStub{getByIDFn: func(_ context.Context, id string) (*Chat, error) {
		if id != validChatID {
			t.Fatalf("unexpected id %q", id)
		}
		return &Chat{ID: id}, nil
	}})

	if _, err := service.GetByID(context.Background(), "not-a-uuid"); !errors.Is(err, ErrInvalidChatID) {
		t.Fatalf("expected ErrInvalidChatID, got %v", err)
	}
	chat, err := service.GetByID(context.Background(), validChatID)
	if err != nil || chat.ID != validChatID {
		t.Fatalf("GetByID() = %#v, %v", chat, err)
	}
}

func TestServiceList(t *testing.T) {
	called := false
	service := NewService(repositoryStub{listFn: func(_ context.Context, limit, offset int) ([]Chat, error) {
		called = true
		if limit != 20 || offset != 40 {
			t.Fatalf("unexpected pagination: %d, %d", limit, offset)
		}
		return []Chat{{ID: validChatID}}, nil
	}})

	for _, tc := range []struct {
		name   string
		limit  int
		offset int
	}{
		{name: "zero limit", limit: 0},
		{name: "limit too large", limit: 101},
		{name: "negative offset", limit: 20, offset: -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := service.List(context.Background(), tc.limit, tc.offset); !errors.Is(err, ErrInvalidPagination) {
				t.Fatalf("expected ErrInvalidPagination, got %v", err)
			}
		})
	}
	if called {
		t.Fatal("repository called for invalid pagination")
	}

	chats, err := service.List(context.Background(), 20, 40)
	if err != nil || len(chats) != 1 || !called {
		t.Fatalf("List() = %#v, %v; called = %v", chats, err, called)
	}
}

func TestServiceCount(t *testing.T) {
	service := NewService(repositoryStub{countFn: func(context.Context) (int64, error) {
		return 42, nil
	}})

	count, err := service.Count(context.Background())
	if err != nil || count != 42 {
		t.Fatalf("Count() = %d, %v", count, err)
	}
}

func TestServiceUpdateTitle(t *testing.T) {
	called := false
	service := NewService(repositoryStub{updateTitleFn: func(_ context.Context, id, title string) (*Chat, error) {
		called = true
		if id != validChatID || title != "Новое название" {
			t.Fatalf("unexpected arguments: %q, %q", id, title)
		}
		return &Chat{ID: id, Title: &title}, nil
	}})

	invalidCases := []struct {
		name  string
		id    string
		title string
		err   error
	}{
		{name: "invalid id", id: "bad", title: "title", err: ErrInvalidChatID},
		{name: "blank title", id: validChatID, title: "   ", err: ErrInvalidTitle},
		{name: "long unicode title", id: validChatID, title: strings.Repeat("я", 256), err: ErrInvalidTitle},
	}
	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := service.UpdateTitle(context.Background(), tc.id, tc.title); !errors.Is(err, tc.err) {
				t.Fatalf("expected %v, got %v", tc.err, err)
			}
		})
	}
	if called {
		t.Fatal("repository called for invalid input")
	}

	chat, err := service.UpdateTitle(context.Background(), validChatID, "  Новое название  ")
	if err != nil || chat.Title == nil || *chat.Title != "Новое название" || !called {
		t.Fatalf("UpdateTitle() = %#v, %v; called = %v", chat, err, called)
	}
}

func TestServiceClose(t *testing.T) {
	called := false
	service := NewService(repositoryStub{closeFn: func(_ context.Context, id string) (*Chat, error) {
		called = true
		return &Chat{ID: id, Status: "closed"}, nil
	}})

	if _, err := service.Close(context.Background(), "bad"); !errors.Is(err, ErrInvalidChatID) {
		t.Fatalf("expected ErrInvalidChatID, got %v", err)
	}
	if called {
		t.Fatal("repository called for invalid id")
	}
	chat, err := service.Close(context.Background(), validChatID)
	if err != nil || chat.Status != "closed" || !called {
		t.Fatalf("Close() = %#v, %v; called = %v", chat, err, called)
	}
}

func TestServiceTouch(t *testing.T) {
	called := false
	service := NewService(repositoryStub{touchFn: func(_ context.Context, id string) error {
		called = true
		if id != validChatID {
			t.Fatalf("unexpected id %q", id)
		}
		return nil
	}})

	if err := service.Touch(context.Background(), "bad"); !errors.Is(err, ErrInvalidChatID) {
		t.Fatalf("expected ErrInvalidChatID, got %v", err)
	}
	if called {
		t.Fatal("repository called for invalid id")
	}
	if err := service.Touch(context.Background(), validChatID); err != nil || !called {
		t.Fatalf("Touch() error = %v; called = %v", err, called)
	}
}
