package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/typewriterpost/typewriterpost/internal/config"
	"github.com/typewriterpost/typewriterpost/internal/httpapi"
	"github.com/typewriterpost/typewriterpost/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	handler := httpapi.NewRouter(cfg.EnableRequestLogging)

	srv := server.New(cfg.HTTPAddr, cfg.ReadHeaderTimeout, cfg.IdleTimeout, handler)
	go func() {
		log.Printf("JMAP API listening on %s", cfg.HTTPAddr)
		if err := srv.Start(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	log.Printf("shutting down, waiting up to %s", cfg.ShutdownTimeout)
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown error: %v", err)
	}

	log.Println("shutdown complete")
}
