package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/qu0ta/MTSHack/internal/chats"
	"github.com/qu0ta/MTSHack/internal/platform"
)

func main() {
	loadEnv()

	db, err := platform.ConnectDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	chatRepository := chats.NewPostgresRepository(db)
	chatService := chats.NewService(chatRepository)
	chatHandler := chats.NewHandler(chatService)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health)
	mux.HandleFunc("GET /ready", ready(db))
	mux.HandleFunc("POST /api/v1/chats", chatHandler.Create)
	mux.HandleFunc("GET /api/v1/chats", chatHandler.List)
	mux.HandleFunc("GET /api/v1/chats/{id}", chatHandler.GetByID)
	mux.HandleFunc("PATCH /api/v1/chats/{id}", chatHandler.UpdateTitle)
	mux.HandleFunc("POST /api/v1/chats/{id}/close", chatHandler.Close)

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("server started on %s", addr)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("server shutdown failed: %v", err)
		}
	}
}

func loadEnv() {
	for _, path := range []string{".env", "../.env", ".env.example", "../.env.example"} {
		if err := godotenv.Load(path); err == nil {
			return
		}
	}
}

func health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("{\"status\":\"ok\"}\n"))
}

type databasePinger interface {
	Ping(context.Context) error
}

func ready(db databasePinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{\"status\":\"ready\"}\n"))
	}
}
