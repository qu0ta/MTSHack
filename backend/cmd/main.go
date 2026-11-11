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
	"github.com/qu0ta/MTSHack/internal/chats"
	"github.com/qu0ta/MTSHack/internal/platform"
)

func main() {
	_ = godotenv.Load()

	db, err := platform.ConnectDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	repository := chats.NewPostgresRepository(db)
	service := chats.NewService(repository)
	handler := chats.NewHandler(service)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health)
	handler.RegisterRoutes(mux)

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
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
