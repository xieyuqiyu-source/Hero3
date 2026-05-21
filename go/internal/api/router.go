package api

import (
	"log/slog"
	"net/http"
	"slices"

	"hero3/internal/config"
	"hero3/internal/game"
)

type RouterOptions struct {
	Config      config.Config
	Logger      *slog.Logger
	GameService *game.Service
}

func NewRouter(options RouterOptions) http.Handler {
	mux := http.NewServeMux()
	gameService := options.GameService
	if gameService == nil {
		gameService = game.NewService()
	}
	handlers := NewHandlers(options.Config, gameService)

	mux.HandleFunc("GET /healthz", handlers.Health)
	mux.HandleFunc("GET /api/v1/meta", handlers.Meta)
	mux.HandleFunc("GET /api/v1/game/bootstrap", handlers.GameBootstrap)
	mux.HandleFunc("GET /api/v1/game/state", handlers.GameState)
	mux.HandleFunc("POST /api/v1/accounts/register", handlers.RegisterAccount)
	mux.HandleFunc("POST /api/v1/accounts/login", handlers.LoginAccount)
	mux.HandleFunc("GET /api/v1/accounts/{accountId}/players", handlers.AccountPlayers)
	mux.HandleFunc("POST /api/v1/players/create", handlers.CreatePlayer)
	mux.HandleFunc("GET /api/v1/admin/accounts", handlers.AdminAccounts)

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
