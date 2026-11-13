package applications

import (
	"context"
	"errors"
	"strings"
	"testing"
)

const validChatID = "550e8400-e29b-41d4-a716-446655440000"

type repositoryStub struct {
	upsertFn      func(context.Context, string, UpsertInput) (*Application, error)
	getByChatIDFn func(context.Context, string) (*Application, error)
}

func (r repositoryStub) Upsert(ctx context.Context, chatID string, input UpsertInput) (*Application, error) {
	return r.upsertFn(ctx, chatID, input)
}

func (r repositoryStub) GetByChatID(ctx context.Context, chatID string) (*Application, error) {
	return r.getByChatIDFn(ctx, chatID)
}

func TestServiceUpsert(t *testing.T) {
	called := false
	service := NewService(repositoryStub{upsertFn: func(_ context.Context, chatID string, input UpsertInput) (*Application, error) {
		called = true
		if chatID != validChatID {
			t.Fatalf("chatID = %q", chatID)
		}
		want := UpsertInput{
			FullName: "Анна Иванова",
			Phone:    "+79991234567",
			Company:  "ООО Пример",
			Summary:  "Нужна консультация",
		}
		if input != want {
			t.Fatalf("input = %#v, want %#v", input, want)
		}
		return &Application{ChatID: chatID, FullName: input.FullName}, nil
	}})

	application, err := service.Upsert(context.Background(), validChatID, UpsertInput{
		FullName: "  Анна Иванова ",
		Phone:    " +79991234567 ",
		Company:  " ООО Пример ",
		Summary:  " Нужна консультация ",
	})
	if err != nil || !called || application.FullName != "Анна Иванова" {
		t.Fatalf("Upsert() = %#v, %v; called = %v", application, err, called)
	}
}

func TestServiceUpsertValidation(t *testing.T) {
	valid := UpsertInput{
		FullName: "Анна Иванова",
		Phone:    "+79991234567",
		Company:  "ООО Пример",
		Summary:  "Нужна консультация",
	}

	tests := []struct {
		name   string
		chatID string
		mutate func(*UpsertInput)
		want   error
	}{
		{name: "invalid chat id", chatID: "bad", want: ErrInvalidChatID},
		{name: "empty full name", chatID: validChatID, mutate: func(input *UpsertInput) { input.FullName = " " }, want: ErrInvalidFullName},
		{name: "long full name", chatID: validChatID, mutate: func(input *UpsertInput) { input.FullName = strings.Repeat("я", 256) }, want: ErrInvalidFullName},
		{name: "empty phone", chatID: validChatID, mutate: func(input *UpsertInput) { input.Phone = " " }, want: ErrInvalidPhone},
		{name: "long phone", chatID: validChatID, mutate: func(input *UpsertInput) { input.Phone = strings.Repeat("1", 31) }, want: ErrInvalidPhone},
		{name: "empty company", chatID: validChatID, mutate: func(input *UpsertInput) { input.Company = " " }, want: ErrInvalidCompany},
		{name: "long company", chatID: validChatID, mutate: func(input *UpsertInput) { input.Company = strings.Repeat("я", 256) }, want: ErrInvalidCompany},
		{name: "empty summary", chatID: validChatID, mutate: func(input *UpsertInput) { input.Summary = " " }, want: ErrInvalidSummary},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := valid
			if tc.mutate != nil {
				tc.mutate(&input)
			}
			service := NewService(repositoryStub{upsertFn: func(context.Context, string, UpsertInput) (*Application, error) {
				t.Fatal("repository must not be called")
				return nil, nil
			}})
			_, err := service.Upsert(context.Background(), tc.chatID, input)
			if !errors.Is(err, tc.want) {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestServiceGetByChatID(t *testing.T) {
	called := false
	service := NewService(repositoryStub{getByChatIDFn: func(_ context.Context, chatID string) (*Application, error) {
		called = true
		return &Application{ChatID: chatID}, nil
	}})

	if _, err := service.GetByChatID(context.Background(), "bad"); !errors.Is(err, ErrInvalidChatID) {
		t.Fatalf("error = %v, want %v", err, ErrInvalidChatID)
	}
	if called {
		t.Fatal("repository called for invalid chat id")
	}

	application, err := service.GetByChatID(context.Background(), validChatID)
	if err != nil || !called || application.ChatID != validChatID {
		t.Fatalf("GetByChatID() = %#v, %v; called = %v", application, err, called)
	}
}
