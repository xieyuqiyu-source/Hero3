package api

import (
	"log/slog"
	"net/http"
	"slices"

	"hero3/internal/config"
	"hero3/internal/game"
)

type RouterOptions struct {
	Config config.Config
	Logger *slog.Logger
}

func NewRouter(options RouterOptions) http.Handler {
	mux := http.NewServeMux()
	handlers := NewHandlers(options.Config, game.NewService())

	mux.HandleFunc("GET /healthz", handlers.Health)
	mux.HandleFunc("GET /api/v1/meta", handlers.Meta)
	mux.HandleFunc("GET /api/v1/game/bootstrap", handlers.GameBootstrap)

	return corsMiddleware(options.Config, mux)
}

func corsMiddleware(cfg config.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && slices.Contains(cfg.AllowedOrigins, origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
