package api

import (
	"log/slog"
	"net/http"
	"slices"

	"hero3/internal/auth"
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
	mux.HandleFunc("GET /api/v1/accounts/{accountId}", handlers.AccountInfo)
	mux.HandleFunc("DELETE /api/v1/accounts/{accountId}", handlers.DeleteAccount)
	mux.HandleFunc("POST /api/v1/players/create", handlers.CreatePlayer)
	mux.HandleFunc("DELETE /api/v1/players/{playerId}", handlers.DeletePlayer)
	mux.HandleFunc("POST /api/v1/city/buildings/upgrade", handlers.UpgradeBuilding)
	mux.HandleFunc("POST /api/v1/city/buildings/upgrade-batch", handlers.UpgradeBuildingBatch)
	mux.HandleFunc("POST /api/v1/city/resources/fill", handlers.FillResources)
	mux.HandleFunc("POST /api/v1/city/resources/fill-paid", handlers.FillResourcesPaid)
	mux.HandleFunc("POST /api/v1/military/recruit", handlers.Recruit)
	mux.HandleFunc("POST /api/v1/military/recruit/instant", handlers.InstantCompleteRecruit)
	mux.HandleFunc("POST /api/v1/city/buildings/instant", handlers.InstantCompleteBuilding)
	mux.HandleFunc("POST /api/v1/city/boost", handlers.PurchaseBoost)
	mux.HandleFunc("POST /api/v1/city/capacity-boost", handlers.PurchaseCapacityBoost)
	mux.HandleFunc("GET /api/v1/city/boost/prices", handlers.BoostPrices)
	mux.HandleFunc("GET /api/v1/map/npc-cities", handlers.NpcCities)
	mux.HandleFunc("POST /api/v1/map/npc-cities/refresh", handlers.RefreshNpcCities)
	mux.HandleFunc("POST /api/v1/map/npc-cities/attack", handlers.AttackNpc)
	mux.HandleFunc("POST /api/v1/map/npc-cities/scout", handlers.ScoutNpc)
	mux.HandleFunc("POST /api/v1/news/mark-read", handlers.MarkReportsRead)
	mux.HandleFunc("POST /api/v1/news/delete-report", handlers.DeleteReport)
	mux.HandleFunc("POST /api/v1/news/delete-all-reports", handlers.DeleteAllReports)
	mux.HandleFunc("GET /api/v1/reports/{reportId}", handlers.GetReport)
	mux.HandleFunc("POST /api/v1/gold/exchange", handlers.ExchangeGold)
	mux.HandleFunc("POST /api/v1/gold/reverse-exchange", handlers.ReverseExchangeGold)
	mux.HandleFunc("GET /api/v1/admin/accounts", handlers.AdminAccounts)
	mux.HandleFunc("GET /api/v1/admin/players/{playerId}/state", handlers.AdminPlayerState)
	mux.HandleFunc("POST /api/v1/admin/resources/adjust", handlers.AdminAdjustResources)
	mux.HandleFunc("POST /api/v1/admin/gold/add", handlers.AddGold)
	mux.HandleFunc("POST /api/v1/admin/gold/deduct", handlers.DeductGold)
	mux.HandleFunc("POST /api/v1/admin/gold/add-account", handlers.AddAccountGold)
	mux.HandleFunc("GET /api/v1/admin/balance", handlers.AdminBalance)
	mux.HandleFunc("PUT /api/v1/admin/balance", handlers.UpdateAdminBalance)
	mux.HandleFunc("GET /api/v1/admin/npc-config", handlers.AdminNpcConfig)
	mux.HandleFunc("PUT /api/v1/admin/npc-config", handlers.UpdateAdminNpcConfig)
	mux.HandleFunc("GET /api/v1/admin/combat-config", handlers.AdminCombatConfig)
	mux.HandleFunc("PUT /api/v1/admin/combat-config", handlers.UpdateAdminCombatConfig)
	mux.HandleFunc("GET /api/v1/admin/factions-config", handlers.AdminFactionsConfig)
	mux.HandleFunc("PUT /api/v1/admin/factions-config", handlers.UpdateAdminFactionsConfig)
	mux.HandleFunc("GET /api/v1/admin/units-config", handlers.AdminUnitsConfig)
	mux.HandleFunc("GET /api/v1/admin/units-config/{faction}", handlers.AdminFactionUnitsConfig)
	mux.HandleFunc("PUT /api/v1/admin/units-config/{faction}", handlers.UpdateAdminFactionUnitsConfig)
	mux.HandleFunc("POST /api/v1/admin/buff/grant", handlers.GrantBuff)
	mux.HandleFunc("DELETE /api/v1/admin/buff/{buffId}", handlers.RevokeBuff)
	mux.HandleFunc("POST /api/v1/minigame/record", handlers.SaveMiniGameRecord)
	mux.HandleFunc("GET /api/v1/admin/minigame/records", handlers.AdminMiniGameRecords)
	mux.HandleFunc("GET /api/v1/admin/generals-config", handlers.AdminGeneralsConfig)
	mux.HandleFunc("PUT /api/v1/admin/generals-config", handlers.UpdateAdminGeneralsConfig)
	mux.HandleFunc("GET /api/v1/admin/general-traits", handlers.AdminGeneralTraitRegistry)

	// 公开路径白名单（不需要认证）
	publicPaths := []string{
		"/healthz",
		"/api/v1/meta",
		"/api/v1/game/bootstrap",
		"/api/v1/accounts/register",
		"/api/v1/accounts/login",
		"/api/v1/city/boost/prices",
	}

	authCfg := auth.Config{
		JWTSecret:  options.Config.JWTSecret,
		AdminToken: options.Config.AdminToken,
		TokenTTL:   options.Config.TokenTTL,
	}

	if authCfg.JWTSecret == "" {
		options.Logger.Error("HERO3_JWT_SECRET not set, player authentication will reject protected requests")
	}

	authed := auth.AuthMiddleware(authCfg, publicPaths)(mux)
	return corsMiddleware(options.Config, authed)
}

func corsMiddleware(cfg config.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && slices.Contains(cfg.AllowedOrigins, origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Admin-Token")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
