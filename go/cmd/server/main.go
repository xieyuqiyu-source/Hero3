package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"hero3/internal/api"
	"hero3/internal/config"
	"hero3/internal/httpserver"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))

	router := api.NewRouter(api.RouterOptions{
		Config: cfg,
		Logger: logger,
	})

	server := httpserver.New(cfg, logger, router)

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("Hero3 API server starting", "addr", cfg.Addr, "env", cfg.Environment)
		serverErrors <- server.ListenAndServe()
	}()

	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		logger.Error("server stopped unexpectedly", "error", err)
		os.Exit(1)
	case signal := <-shutdownSignals:
		logger.Info("shutdown signal received", "signal", signal.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server shutdown failed", "error", err)
		os.Exit(1)
	}

	logger.Info("Hero3 API server stopped")
}
