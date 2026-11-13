package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/qu0ta/MTSHack/internal/applications"
	"github.com/qu0ta/MTSHack/internal/chats"
	"github.com/qu0ta/MTSHack/internal/consultations"
	"github.com/qu0ta/MTSHack/internal/messages"
	"github.com/qu0ta/MTSHack/internal/mlclient"
	"github.com/qu0ta/MTSHack/internal/platform"
)

func main() {
	_ = godotenv.Load()

	db, err := platform.ConnectDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	chatRepository := chats.NewPostgresRepository(db)
	chatService := chats.NewService(chatRepository)
	chatHandler := chats.NewHandler(chatService)

	messageRepository := messages.NewPostgresRepository(db)
	messageService := messages.NewMessageService(messageRepository)
	messageHandler := messages.NewHandler(messageService, chatService)

	mlClient, err := mlclient.New(os.Getenv("ML_SERVICE_URL"), time.Second*65)
	if err != nil {
		log.Fatal(err)
	}

	consultService := consultations.NewPostgresService(db, mlClient)
	consultHandler := consultations.NewHandler(consultService)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health)
	mux.HandleFunc("GET /ready", ready(db))
	chatHandler.RegisterRoutes(mux)
	messageHandler.RegisterRoutes(mux)
	consultHandler.RegisterRoutes(mux)
	applicationService := applications.NewService(applications.NewPostgresRepository(db))
	mux.HandleFunc("GET /api/v1/chats/{id}/application", applicationService.GetHandler)

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      95 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	shutdownCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-shutdownCtx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("HTTP server shutdown: %v", err)
		}
	}()

	log.Printf("HTTP server listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		log.Printf("encode health response: %v", err)
	}
}

func ready(db interface {
	Ping(context.Context) error
}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second*2)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(map[string]string{
			"status": "ready",
		}); err != nil {
			log.Printf("encode readiness response: %v", err)
		}

	}
}
