// Package api 负责 Hero3 HTTP API 路由和接口处理器注册。
package api

import (
	"log/slog"
	"net/http"
	"slices"

	"hero3/internal/app/game"
	"hero3/internal/app/helpdocs"
	"hero3/internal/platform/auth"
	"hero3/internal/platform/config"
	"hero3/internal/transport/apidocs"
	helpdocstransport "hero3/internal/transport/helpdocs"
)

type RouterOptions struct {
	Config      config.Config
	Logger      *slog.Logger
	GameService *game.Service
}

// NewRouter 创建包含业务接口、在线文档、认证和 CORS 的 HTTP handler。
func NewRouter(options RouterOptions) http.Handler {
	mux := http.NewServeMux()
	gameService := options.GameService
	if gameService == nil {
		gameService = game.NewService()
	}
	handlers := NewHandlers(options.Config, gameService)

	apidocs.RegisterRoutes(mux, apidocs.Options{})
	registerPublicRoutes(mux, handlers)
	registerAccountRoutes(mux, handlers)
	registerCityRoutes(mux, handlers)
	registerMilitaryRoutes(mux, handlers)
	registerReinforcementRoutes(mux, handlers)
	registerPvpRoutes(mux, handlers)
	registerMapRoutes(mux, handlers)
	registerCombatRoutes(mux, handlers)
	registerReportRoutes(mux, handlers)
	registerMailRoutes(mux, handlers)
	registerAnnouncementRoutes(mux, handlers)
	registerGoldRoutes(mux, handlers)
	registerItemRoutes(mux, handlers)
	registerMiniGameRoutes(mux, handlers)
	registerReincarnationRoutes(mux, handlers)
	registerAdminRoutes(mux, handlers)
	helpdocstransport.RegisterRoutes(mux, helpdocs.NewService(options.Config.HelpDocsDir))

	// 公开路径白名单（不需要认证）
	publicPaths := []string{
		"/healthz",
		"/api/v1/meta",
		"/api/v1/game/bootstrap",
		"/api/v1/accounts/register",
		"/api/v1/accounts/login",
		"/api/v1/city/boost/prices",
	}
	publicPaths = append(publicPaths, apidocs.PublicPaths()...)

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

// registerPublicRoutes 注册公开和基础游戏状态路由。
func registerPublicRoutes(mux *http.ServeMux, handlers *Handlers) {
	mux.HandleFunc("GET /healthz", handlers.Health)
	mux.HandleFunc("GET /api/v1/meta", handlers.Meta)
	mux.HandleFunc("GET /api/v1/game/bootstrap", handlers.GameBootstrap)
	mux.HandleFunc("GET /api/v1/game/state", handlers.GameState)
	mux.HandleFunc("GET /api/v1/game/summary", handlers.GameSummary)
}

// registerAccountRoutes 注册账号和存档路由。
func registerAccountRoutes(mux *http.ServeMux, handlers *Handlers) {
	mux.HandleFunc("POST /api/v1/accounts/register", handlers.RegisterAccount)
	mux.HandleFunc("POST /api/v1/accounts/login", handlers.LoginAccount)
	mux.HandleFunc("GET /api/v1/accounts/{accountId}/players", handlers.AccountPlayers)
	mux.HandleFunc("GET /api/v1/accounts/{accountId}", handlers.AccountInfo)
	mux.HandleFunc("DELETE /api/v1/accounts/{accountId}", handlers.DeleteAccount)
	mux.HandleFunc("POST /api/v1/players/create", handlers.CreatePlayer)
	mux.HandleFunc("DELETE /api/v1/players/{playerId}", handlers.DeletePlayer)
	mux.HandleFunc("POST /api/v1/players/{playerId}/restore-delete", handlers.RestorePlayerDeletion)
}

// registerCityRoutes 注册城池、建筑和资源路由。
func registerCityRoutes(mux *http.ServeMux, handlers *Handlers) {
	mux.HandleFunc("GET /api/v1/city/view", handlers.CityView)
	mux.HandleFunc("GET /api/v1/resources/view", handlers.ResourceView)
	mux.HandleFunc("POST /api/v1/city/buildings/upgrade", handlers.UpgradeBuilding)
	mux.HandleFunc("POST /api/v1/city/buildings/upgrade-batch", handlers.UpgradeBuildingBatch)
	mux.HandleFunc("POST /api/v1/city/buildings/instant", handlers.InstantCompleteBuilding)
	mux.HandleFunc("POST /api/v1/city/resources/fill", handlers.FillResources)
	mux.HandleFunc("POST /api/v1/city/resources/fill-paid", handlers.FillResourcesPaid)
	mux.HandleFunc("POST /api/v1/city/boost", handlers.PurchaseBoost)
	mux.HandleFunc("POST /api/v1/city/capacity-boost", handlers.PurchaseCapacityBoost)
	mux.HandleFunc("GET /api/v1/city/boost/prices", handlers.BoostPrices)
}

// registerMilitaryRoutes 注册征兵和武将操作路由。
func registerMilitaryRoutes(mux *http.ServeMux, handlers *Handlers) {
	mux.HandleFunc("GET /api/v1/military/view", handlers.MilitaryView)
	mux.HandleFunc("GET /api/v1/generals/view", handlers.GeneralsView)
	mux.HandleFunc("POST /api/v1/military/recruit", handlers.Recruit)
	mux.HandleFunc("POST /api/v1/military/recruit/instant", handlers.InstantCompleteRecruit)
	mux.HandleFunc("POST /api/v1/military/general/stat", handlers.AllocateGeneralStat)
	mux.HandleFunc("POST /api/v1/military/general/reset-stats", handlers.ResetGeneralStats)
	mux.HandleFunc("POST /api/v1/military/general/change", handlers.ChangeGeneral)
}

// registerReinforcementRoutes 注册增援系统路由。
func registerReinforcementRoutes(mux *http.ServeMux, handlers *Handlers) {
	mux.HandleFunc("POST /api/v1/reinforcements", handlers.SendReinforcement)
	mux.HandleFunc("GET /api/v1/reinforcements/sent", handlers.ListSentReinforcements)
	mux.HandleFunc("GET /api/v1/reinforcements/received", handlers.ListReceivedReinforcements)
	mux.HandleFunc("GET /api/v1/reinforcements/{reinforcementId}", handlers.GetReinforcement)
	mux.HandleFunc("POST /api/v1/reinforcements/{reinforcementId}/recall", handlers.RecallReinforcement)
	mux.HandleFunc("POST /api/v1/reinforcements/{reinforcementId}/expel", handlers.ExpelReinforcement)
	mux.HandleFunc("POST /api/v1/reinforcements/{reinforcementId}/accelerate", handlers.AccelerateReinforcement)
}

// registerPvpRoutes 注册 PVP 玩家战斗路由。
func registerPvpRoutes(mux *http.ServeMux, handlers *Handlers) {
	mux.HandleFunc("GET /api/v1/pvp/targets", handlers.PvpTargets)
	mux.HandleFunc("GET /api/v1/pvp/targets/{targetPlayerId}", handlers.PvpTarget)
	mux.HandleFunc("POST /api/v1/pvp/scout", handlers.ScoutPvpTarget)
	mux.HandleFunc("POST /api/v1/pvp/attacks", handlers.StartPvpAttack)
	mux.HandleFunc("GET /api/v1/pvp/marches", handlers.PvpMarches)
	mux.HandleFunc("POST /api/v1/pvp/marches/{marchId}/accelerate", handlers.AcceleratePvpMarch)
	mux.HandleFunc("POST /api/v1/pvp/marches/{marchId}/recall", handlers.RecallPvpMarch)
	mux.HandleFunc("GET /api/v1/pvp/battles", handlers.PvpBattles)
	mux.HandleFunc("GET /api/v1/pvp/battles/{battleId}", handlers.PvpBattle)
	mux.HandleFunc("GET /api/v1/pvp/state", handlers.PvpState)
	mux.HandleFunc("GET /api/v1/pvp/revenge", handlers.PvpRevenge)
	mux.HandleFunc("GET /api/v1/pvp/season", handlers.PvpSeason)
	mux.HandleFunc("GET /api/v1/pvp/rankings", handlers.PvpRankings)
}

// registerMapRoutes 注册地图和 NPC 城池路由。
func registerMapRoutes(mux *http.ServeMux, handlers *Handlers) {
	mux.HandleFunc("GET /api/v1/world-map/view", handlers.WorldMapView)
	mux.HandleFunc("GET /api/v1/world-map/targets/player_city/{playerId}", handlers.WorldMapPlayerCityTarget)
	mux.HandleFunc("GET /api/v1/map/npc-cities", handlers.NpcCities)
	mux.HandleFunc("POST /api/v1/map/npc-cities/refresh", handlers.RefreshNpcCities)
	mux.HandleFunc("POST /api/v1/map/npc-cities/attack", handlers.AttackNpc)
	mux.HandleFunc("POST /api/v1/map/npc-cities/sweep", handlers.SweepNpc)
	mux.HandleFunc("POST /api/v1/map/npc-cities/scout", handlers.ScoutNpc)
}

// registerReincarnationRoutes 注册轮回绝境副本路由。
func registerReincarnationRoutes(mux *http.ServeMux, handlers *Handlers) {
	mux.HandleFunc("GET /api/v1/dungeons/reincarnation/config", handlers.ReincarnationConfig)
	mux.HandleFunc("GET /api/v1/dungeons/reincarnation/run", handlers.ReincarnationRun)
	mux.HandleFunc("POST /api/v1/dungeons/reincarnation/start", handlers.StartReincarnation)
	mux.HandleFunc("POST /api/v1/dungeons/reincarnation/waves/{waveId}/attack", handlers.AttackReincarnationWave)
	mux.HandleFunc("POST /api/v1/dungeons/reincarnation/waves/{waveId}/defense-ready", handlers.ReadyReincarnationDefense)
	mux.HandleFunc("POST /api/v1/dungeons/reincarnation/waves/{waveId}/bonus-reset", handlers.ResetReincarnationWaveBonus)
	mux.HandleFunc("POST /api/v1/dungeons/reincarnation/settle", handlers.SettleReincarnation)
	mux.HandleFunc("GET /api/v1/dungeons/reincarnation/reports", handlers.ReincarnationReports)
}

// registerCombatRoutes 注册战斗模拟路由。
func registerCombatRoutes(mux *http.ServeMux, handlers *Handlers) {
	mux.HandleFunc("POST /api/v1/combat/simulate", handlers.SimulateBattle)
}

// registerReportRoutes 注册战报路由。
func registerReportRoutes(mux *http.ServeMux, handlers *Handlers) {
	mux.HandleFunc("GET /api/v1/news/reports", handlers.ListReports)
	mux.HandleFunc("POST /api/v1/news/mark-read", handlers.MarkReportsRead)
	mux.HandleFunc("POST /api/v1/news/delete-report", handlers.DeleteReport)
	mux.HandleFunc("POST /api/v1/news/delete-all-reports", handlers.DeleteAllReports)
	mux.HandleFunc("GET /api/v1/reports", handlers.ListReports)
	mux.HandleFunc("GET /api/v1/reports/{reportId}", handlers.GetReport)
	mux.HandleFunc("POST /api/v1/reports/{reportId}/read", handlers.MarkReportReadByPath)
	mux.HandleFunc("POST /api/v1/reports/read-all", handlers.MarkAllReportsReadByPath)
	mux.HandleFunc("POST /api/v1/reports/{reportId}/delete", handlers.DeleteReportByPath)
	mux.HandleFunc("POST /api/v1/reports/delete-all", handlers.DeleteAllReportsByPath)
	mux.HandleFunc("POST /api/v1/reports/{reportId}/share", handlers.ShareReport)
	mux.HandleFunc("GET /api/v1/reports/shared/{token}", handlers.GetSharedReport)
}

// registerMailRoutes 注册信函路由。
func registerMailRoutes(mux *http.ServeMux, handlers *Handlers) {
	mux.HandleFunc("GET /api/v1/mails", handlers.ListMails)
	mux.HandleFunc("GET /api/v1/mails/{mailId}", handlers.GetMail)
	mux.HandleFunc("POST /api/v1/mails/{mailId}/delete", handlers.DeleteMail)
	mux.HandleFunc("POST /api/v1/mails/{mailId}/claim", handlers.ClaimMailAttachments)
	mux.HandleFunc("POST /api/v1/mails/send-player", handlers.SendPlayerMail)
	mux.HandleFunc("POST /api/v1/mails/server-broadcast", handlers.SendServerBroadcastMail)
}

// registerAnnouncementRoutes 注册公告系统路由。
func registerAnnouncementRoutes(mux *http.ServeMux, handlers *Handlers) {
	mux.HandleFunc("GET /api/v1/announcements", handlers.ListAnnouncements)
	mux.HandleFunc("GET /api/v1/announcements/popups", handlers.ListAnnouncementPopups)
	mux.HandleFunc("GET /api/v1/announcements/{announcementId}", handlers.GetAnnouncement)
	mux.HandleFunc("POST /api/v1/announcements/{announcementId}/read", handlers.MarkAnnouncementRead)
	mux.HandleFunc("POST /api/v1/announcements/{announcementId}/popup-shown", handlers.MarkAnnouncementPopupShown)
	mux.HandleFunc("POST /api/v1/announcements/{announcementId}/dismiss", handlers.DismissAnnouncement)
}

// registerGoldRoutes 注册金币和城金兑换路由。
func registerGoldRoutes(mux *http.ServeMux, handlers *Handlers) {
	mux.HandleFunc("POST /api/v1/gold/exchange", handlers.ExchangeGold)
	mux.HandleFunc("POST /api/v1/gold/reverse-exchange", handlers.ReverseExchangeGold)
}

// registerItemRoutes 注册道具路由。
func registerItemRoutes(mux *http.ServeMux, handlers *Handlers) {
	mux.HandleFunc("GET /api/v1/items/config", handlers.ItemsConfig)
	mux.HandleFunc("GET /api/v1/inventory/view", handlers.InventoryView)
	mux.HandleFunc("POST /api/v1/items/use", handlers.UseItem)
}

// registerMiniGameRoutes 注册小游戏路由。
func registerMiniGameRoutes(mux *http.ServeMux, handlers *Handlers) {
	mux.HandleFunc("POST /api/v1/minigame/record", handlers.SaveMiniGameRecord)
	mux.HandleFunc("GET /api/v1/minigame/records", handlers.ListMiniGameRecords)
	mux.HandleFunc("POST /api/v1/minigame/fishing/use-bait", handlers.UseFishingBait)
	mux.HandleFunc("POST /api/v1/minigame/gambling/resolve", handlers.ResolveGamblingRound)
	mux.HandleFunc("POST /api/v1/minigame/slot/resolve", handlers.ResolveSlotRound)
	mux.HandleFunc("POST /api/v1/minigame/redeem", handlers.RedeemMiniGameReward)
	mux.HandleFunc("POST /api/v1/minigame/redeem-all", handlers.RedeemAllMiniGameRewards)
}

// registerAdminRoutes 注册 GM 后台路由。
func registerAdminRoutes(mux *http.ServeMux, handlers *Handlers) {
	mux.HandleFunc("GET /api/v1/admin/accounts", handlers.AdminAccounts)
	mux.HandleFunc("GET /api/v1/admin/announcements", handlers.AdminAnnouncements)
	mux.HandleFunc("POST /api/v1/admin/announcements", handlers.AdminCreateAnnouncement)
	mux.HandleFunc("PUT /api/v1/admin/announcements/{announcementId}", handlers.AdminUpdateAnnouncement)
	mux.HandleFunc("POST /api/v1/admin/announcements/{announcementId}/publish", handlers.AdminPublishAnnouncement)
	mux.HandleFunc("POST /api/v1/admin/announcements/{announcementId}/schedule", handlers.AdminScheduleAnnouncement)
	mux.HandleFunc("POST /api/v1/admin/announcements/{announcementId}/withdraw", handlers.AdminWithdrawAnnouncement)
	mux.HandleFunc("POST /api/v1/admin/announcements/{announcementId}/archive", handlers.AdminArchiveAnnouncement)
	mux.HandleFunc("DELETE /api/v1/admin/announcements/{announcementId}", handlers.AdminDeleteAnnouncement)
	mux.HandleFunc("GET /api/v1/admin/players/{playerId}/state", handlers.AdminPlayerState)
	mux.HandleFunc("GET /api/v1/admin/world-map/occupancy", handlers.AdminWorldMapOccupancy)
	mux.HandleFunc("GET /api/v1/admin/world-map/positions/check", handlers.AdminWorldCoordinateCheck)
	mux.HandleFunc("GET /api/v1/admin/world-map/positions/{playerId}", handlers.AdminWorldPosition)
	mux.HandleFunc("PUT /api/v1/admin/world-map/positions/{playerId}", handlers.UpdateAdminWorldPosition)
	mux.HandleFunc("GET /api/v1/admin/pvp/overview", handlers.AdminPvpOverview)
	mux.HandleFunc("GET /api/v1/admin/pvp/marches", handlers.AdminPvpMarches)
	mux.HandleFunc("GET /api/v1/admin/pvp/battles", handlers.AdminPvpBattles)
	mux.HandleFunc("GET /api/v1/admin/reports", handlers.AdminReports)
	mux.HandleFunc("GET /api/v1/admin/reports/{reportId}", handlers.AdminReport)
	mux.HandleFunc("GET /api/v1/admin/battle-events", handlers.AdminBattleEvents)
	mux.HandleFunc("GET /api/v1/admin/battle-events/{eventId}", handlers.AdminBattleEvent)
	mux.HandleFunc("GET /api/v1/admin/battle-events/{eventId}/reports", handlers.AdminBattleEventReports)
	mux.HandleFunc("GET /api/v1/admin/battle-events/{eventId}/participants", handlers.AdminBattleEventParticipants)
	mux.HandleFunc("GET /api/v1/admin/pvp/seasons", handlers.AdminPvpSeasons)
	mux.HandleFunc("POST /api/v1/admin/pvp/seasons", handlers.AdminCreatePvpSeason)
	mux.HandleFunc("PUT /api/v1/admin/pvp/seasons/{seasonId}", handlers.AdminUpdatePvpSeason)
	mux.HandleFunc("POST /api/v1/admin/pvp/seasons/{seasonId}/settle", handlers.AdminSettlePvpSeason)
	mux.HandleFunc("GET /api/v1/admin/pvp/players/{playerId}", handlers.AdminPvpPlayer)
	mux.HandleFunc("POST /api/v1/admin/pvp/players/{playerId}/protection", handlers.AdminSetPvpProtection)
	mux.HandleFunc("POST /api/v1/admin/pvp/marches/{marchId}/force-resolve", handlers.AdminForceResolvePvpMarch)
	mux.HandleFunc("POST /api/v1/admin/pvp/marches/{marchId}/cancel", handlers.AdminCancelPvpMarch)
	mux.HandleFunc("POST /api/v1/admin/resources/adjust", handlers.AdminAdjustResources)
	mux.HandleFunc("GET /api/v1/admin/items/config", handlers.AdminItemsConfig)
	mux.HandleFunc("PUT /api/v1/admin/items/config", handlers.UpdateAdminItemsConfig)
	mux.HandleFunc("POST /api/v1/admin/items/config/validate", handlers.ValidateAdminItemsConfig)
	mux.HandleFunc("GET /api/v1/admin/items/drop-pools", handlers.AdminDropPoolsConfig)
	mux.HandleFunc("PUT /api/v1/admin/items/drop-pools", handlers.UpdateAdminDropPoolsConfig)
	mux.HandleFunc("POST /api/v1/admin/items/drop-pools/validate", handlers.ValidateAdminDropPoolsConfig)
	mux.HandleFunc("POST /api/v1/admin/items/grant", handlers.AdminGrantItem)
	mux.HandleFunc("GET /api/v1/admin/items/inventory", handlers.AdminInventoryView)
	mux.HandleFunc("GET /api/v1/admin/items/ledger", handlers.AdminItemLedger)
	mux.HandleFunc("GET /api/v1/admin/dungeons/reincarnation/config", handlers.AdminReincarnationConfig)
	mux.HandleFunc("PUT /api/v1/admin/dungeons/reincarnation/config", handlers.UpdateAdminReincarnationConfig)
	mux.HandleFunc("GET /api/v1/admin/dungeons/reincarnation/runs", handlers.AdminReincarnationRuns)
	mux.HandleFunc("GET /api/v1/admin/dungeons/reincarnation/runs/{runId}", handlers.AdminReincarnationRun)
	mux.HandleFunc("POST /api/v1/admin/dungeons/reincarnation/runs/{runId}/force-settle", handlers.AdminForceSettleReincarnationRun)
	mux.HandleFunc("POST /api/v1/admin/dungeons/reincarnation/runs/{runId}/repair-reward", handlers.AdminRepairReincarnationReward)
	mux.HandleFunc("POST /api/v1/admin/gold/add", handlers.AddGold)
	mux.HandleFunc("POST /api/v1/admin/gold/deduct", handlers.DeductGold)
	mux.HandleFunc("POST /api/v1/admin/gold/add-account", handlers.AddAccountGold)
	mux.HandleFunc("GET /api/v1/admin/gold/ledger", handlers.AdminGoldLedger)
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
	mux.HandleFunc("GET /api/v1/admin/minigame/records", handlers.AdminMiniGameRecords)
	mux.HandleFunc("GET /api/v1/admin/generals-config", handlers.AdminGeneralsConfig)
	mux.HandleFunc("PUT /api/v1/admin/generals-config", handlers.UpdateAdminGeneralsConfig)
	mux.HandleFunc("GET /api/v1/admin/fishing-config", handlers.AdminFishingConfig)
	mux.HandleFunc("PUT /api/v1/admin/fishing-config", handlers.UpdateAdminFishingConfig)
	mux.HandleFunc("GET /api/v1/admin/slot-config", handlers.AdminSlotConfig)
	mux.HandleFunc("PUT /api/v1/admin/slot-config", handlers.UpdateAdminSlotConfig)
	mux.HandleFunc("GET /api/v1/admin/general-traits", handlers.AdminGeneralTraitRegistry)
	mux.HandleFunc("POST /api/v1/admin/mails/send", handlers.AdminSendMail)
	mux.HandleFunc("GET /api/v1/admin/players/{playerId}/mails", handlers.AdminPlayerMails)
}

// corsMiddleware 处理允许的跨域请求和预检请求。
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
