package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"hero3/internal/auth"
	"hero3/internal/combat"
	"hero3/internal/config"
	"hero3/internal/game"
	"hero3/internal/general"
)

type Handlers struct {
	cfg         config.Config
	gameService *game.Service
}

type accountRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type createPlayerRequest struct {
	AccountID string `json:"accountId"`
	Nickname  string `json:"nickname"`
	Faction   string `json:"faction"`
	GeneralID string `json:"generalId"`
}

type upgradeBuildingRequest struct {
	PlayerID   string `json:"playerId"`
	BuildingID string `json:"buildingId"`
}

func NewHandlers(cfg config.Config, gameService *game.Service) *Handlers {
	return &Handlers{
		cfg:         cfg,
		gameService: gameService,
	}
}

func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":      "ok",
		"service":     h.cfg.ServiceName,
		"version":     h.cfg.Version,
		"environment": h.cfg.Environment,
		"time":        time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *Handlers) Meta(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"name":       "Hero3",
		"apiVersion": "v1",
		"service":    h.cfg.ServiceName,
		"version":    h.cfg.Version,
	})
}

func (h *Handlers) GameBootstrap(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.gameService.Bootstrap())
}

func (h *Handlers) GameState(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("playerId")
	if playerID == "" {
		writeError(w, http.StatusBadRequest, "playerId is required")
		return
	}
	if !h.requireOwnership(w, r, playerID) {
		return
	}
	state, err := h.gameService.GetState(playerID)
	if err != nil {
		if errors.Is(err, game.ErrPlayerNotFound) {
			writeError(w, http.StatusNotFound, "player not found")
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func (h *Handlers) RegisterAccount(w http.ResponseWriter, r *http.Request) {
	var payload accountRequest
	if !decodeJSON(w, r, &payload) {
		return
	}
	if h.cfg.JWTSecret == "" {
		writeError(w, http.StatusInternalServerError, "authentication is not configured")
		return
	}

	account, err := h.gameService.RegisterAccount(payload.Username, payload.Password)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, game.ErrAccountExists) {
			status = http.StatusConflict
		}
		writeError(w, status, err.Error())
		return
	}

	token, err := auth.IssueToken(auth.Config{
		JWTSecret: h.cfg.JWTSecret,
		TokenTTL:  h.cfg.TokenTTL,
	}, account.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "authentication is not configured")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"accountId": account.ID,
		"username":  account.Username,
		"gold":      account.Gold,
		"token":     token,
	})
}

func (h *Handlers) LoginAccount(w http.ResponseWriter, r *http.Request) {
	var payload accountRequest
	if !decodeJSON(w, r, &payload) {
		return
	}
	if h.cfg.JWTSecret == "" {
		writeError(w, http.StatusInternalServerError, "authentication is not configured")
		return
	}

	account, err := h.gameService.LoginAccount(payload.Username, payload.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	token, err := auth.IssueToken(auth.Config{
		JWTSecret: h.cfg.JWTSecret,
		TokenTTL:  h.cfg.TokenTTL,
	}, account.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "authentication is not configured")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"accountId": account.ID,
		"username":  account.Username,
		"gold":      account.Gold,
		"token":     token,
	})
}

func (h *Handlers) AccountPlayers(w http.ResponseWriter, r *http.Request) {
	accountID := r.PathValue("accountId")
	if !h.requireAccount(w, r, accountID) {
		return
	}
	players, err := h.gameService.ListPlayers(accountID)
	if err != nil {
		writeError(w, http.StatusNotFound, "account not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"players": players})
}

func (h *Handlers) AccountInfo(w http.ResponseWriter, r *http.Request) {
	accountID := r.PathValue("accountId")
	if !h.requireAccount(w, r, accountID) {
		return
	}
	account, err := h.gameService.GetAccountByID(accountID)
	if err != nil {
		writeError(w, http.StatusNotFound, "account not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"accountId": account.ID,
		"username":  account.Username,
		"gold":      account.Gold,
	})
}

func (h *Handlers) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	accountID := r.PathValue("accountId")
	if !h.requireAccount(w, r, accountID) {
		return
	}
	if err := h.gameService.DeleteAccount(accountID); err != nil {
		writeError(w, http.StatusNotFound, "account not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handlers) AdminPlayerState(w http.ResponseWriter, r *http.Request) {
	playerID := r.PathValue("playerId")
	state, err := h.gameService.GetState(playerID)
	if err != nil {
		writeError(w, http.StatusNotFound, "player not found")
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func (h *Handlers) AdminAdjustResources(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID    string         `json:"playerId"`
		Adjustments map[string]int `json:"adjustments"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}

	if payload.PlayerID == "" || len(payload.Adjustments) == 0 {
		writeError(w, http.StatusBadRequest, "playerId and adjustments are required")
		return
	}

	state, err := h.gameService.AdjustResources(payload.PlayerID, payload.Adjustments)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, game.ErrPlayerNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"state": state})
}

func (h *Handlers) AdminAccounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := h.gameService.ListAccounts()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "accounts load failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"accounts": accounts})
}

func (h *Handlers) AdminBalance(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.gameService.GetBalance())
}

func (h *Handlers) UpdateAdminBalance(w http.ResponseWriter, r *http.Request) {
	var payload game.BalanceConfig
	if !decodeJSON(w, r, &payload) {
		return
	}

	if err := h.gameService.UpdateBalance(payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, h.gameService.GetBalance())
}

func (h *Handlers) AdminNpcConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.gameService.GetNpcConfig())
}

func (h *Handlers) UpdateAdminNpcConfig(w http.ResponseWriter, r *http.Request) {
	var payload game.NpcConfig
	if !decodeJSON(w, r, &payload) {
		return
	}

	if err := h.gameService.UpdateNpcConfig(payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, h.gameService.GetNpcConfig())
}

func (h *Handlers) AdminCombatConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, combat.GetCombatConfig())
}

func (h *Handlers) UpdateAdminCombatConfig(w http.ResponseWriter, r *http.Request) {
	var payload combat.CombatConfig
	if !decodeJSON(w, r, &payload) {
		return
	}

	if err := h.gameService.UpdateCombatConfig(payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, combat.GetCombatConfig())
}

func (h *Handlers) AdminFactionsConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, game.GetFactionsConfig())
}

func (h *Handlers) UpdateAdminFactionsConfig(w http.ResponseWriter, r *http.Request) {
	var payload game.FactionsConfig
	if !decodeJSON(w, r, &payload) {
		return
	}

	if err := h.gameService.UpdateFactionsConfig(payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, game.GetFactionsConfig())
}

func (h *Handlers) AdminUnitsConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, game.GetUnitsConfig())
}

func (h *Handlers) AdminFactionUnitsConfig(w http.ResponseWriter, r *http.Request) {
	faction := r.PathValue("faction")
	units := game.GetFactionUnits(faction)
	if units == nil {
		writeError(w, http.StatusNotFound, "faction not found")
		return
	}
	writeJSON(w, http.StatusOK, units)
}

func (h *Handlers) UpdateAdminFactionUnitsConfig(w http.ResponseWriter, r *http.Request) {
	faction := r.PathValue("faction")
	var payload game.FactionUnits
	if !decodeJSON(w, r, &payload) {
		return
	}

	if err := h.gameService.UpdateFactionUnits(faction, payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, game.GetFactionUnits(faction))
}

func (h *Handlers) CreatePlayer(w http.ResponseWriter, r *http.Request) {
	var payload createPlayerRequest
	if !decodeJSON(w, r, &payload) {
		return
	}
	// 创建玩家时校验 accountId 归属
	if !h.requireAccount(w, r, payload.AccountID) {
		return
	}

	playerID, state, err := h.gameService.CreatePlayer(payload.AccountID, payload.Nickname, payload.Faction, payload.GeneralID)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrAccountNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrInvalidGeneral):
			status = http.StatusUnprocessableEntity
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"playerId": playerID,
		"state":    state,
	})
}

func (h *Handlers) DeletePlayer(w http.ResponseWriter, r *http.Request) {
	playerID := r.PathValue("playerId")
	if !h.requireOwnership(w, r, playerID) {
		return
	}
	if err := h.gameService.DeletePlayer(playerID); err != nil {
		writeError(w, http.StatusNotFound, "player not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handlers) FillResources(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	state, err := h.gameService.FillResources(payload.PlayerID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"state": state})
}

func (h *Handlers) FillResourcesPaid(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	state, cost, err := h.gameService.FillResourcesPaid(payload.PlayerID)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrInsufficientCityGold):
			status = http.StatusUnprocessableEntity
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"state": state, "cost": cost})
}

func (h *Handlers) Recruit(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
		UnitID   string `json:"unitId"`
		Amount   int    `json:"amount"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	state, err := h.gameService.Recruit(payload.PlayerID, payload.UnitID, payload.Amount)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrUnitNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrInsufficientRes):
			status = http.StatusUnprocessableEntity
		case errors.Is(err, game.ErrInvalidAmount):
			status = http.StatusBadRequest
		case errors.Is(err, game.ErrQueueFull):
			status = http.StatusConflict
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"state": state})
}

func (h *Handlers) InstantCompleteRecruit(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
		QueueID  string `json:"queueId"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	state, err := h.gameService.InstantCompleteRecruit(payload.PlayerID, payload.QueueID)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrInsufficientCityGold):
			status = http.StatusUnprocessableEntity
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"state": state})
}

func (h *Handlers) AllocateGeneralStat(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
		StatKey  string `json:"statKey"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	state, err := h.gameService.AllocateGeneralStat(payload.PlayerID, payload.StatKey)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound), errors.Is(err, game.ErrGeneralNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrNoStatPoints), errors.Is(err, game.ErrStatMaxLevel):
			status = http.StatusConflict
		case errors.Is(err, game.ErrInvalidStatKey):
			status = http.StatusBadRequest
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"state": state})
}

func (h *Handlers) InstantCompleteBuilding(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID   string `json:"playerId"`
		BuildingID string `json:"buildingId"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	state, err := h.gameService.InstantCompleteBuilding(payload.PlayerID, payload.BuildingID)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrBuildingNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrNotUpgrading):
			status = http.StatusConflict
		case errors.Is(err, game.ErrInsufficientCityGold):
			status = http.StatusUnprocessableEntity
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"state": state})
}

func (h *Handlers) NpcCities(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("playerId")
	if playerID == "" {
		writeError(w, http.StatusBadRequest, "playerId is required")
		return
	}
	if !h.requireOwnership(w, r, playerID) {
		return
	}
	npcState, err := h.gameService.GetNpcCities(playerID)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, game.ErrPlayerNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, npcState)
}

func (h *Handlers) RefreshNpcCities(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	npcState, err := h.gameService.RefreshNpcCities(payload.PlayerID)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, game.ErrPlayerNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, npcState)
}

func (h *Handlers) AttackNpc(w http.ResponseWriter, r *http.Request) {
	var payload game.AttackNpcRequest
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	result, err := h.gameService.AttackNpc(payload)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrNpcNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrNoUnitsSelected):
			status = http.StatusBadRequest
		case errors.Is(err, game.ErrNonCombatUnit):
			status = http.StatusBadRequest
		case errors.Is(err, game.ErrInsufficientArmy):
			status = http.StatusUnprocessableEntity
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handlers) SimulateBattle(w http.ResponseWriter, r *http.Request) {
	var payload game.BattleSimulationRequest
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	result, err := h.gameService.SimulateBattle(payload)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrNoUnitsSelected):
			status = http.StatusBadRequest
		case errors.Is(err, game.ErrNonCombatUnit):
			status = http.StatusBadRequest
		case errors.Is(err, game.ErrUnitNotFound):
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handlers) ScoutNpc(w http.ResponseWriter, r *http.Request) {
	var payload game.ScoutNpcRequest
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	result, err := h.gameService.ScoutNpc(payload)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrNpcNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrInsufficientArmy):
			status = http.StatusUnprocessableEntity
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handlers) GetReport(w http.ResponseWriter, r *http.Request) {
	reportID := r.PathValue("reportId")
	if reportID == "" {
		writeError(w, http.StatusBadRequest, "reportId is required")
		return
	}

	report, err := h.gameService.GetReportByID(reportID)
	if err != nil {
		writeError(w, http.StatusNotFound, "report not found")
		return
	}

	writeJSON(w, http.StatusOK, report)
}

func (h *Handlers) ListReports(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("playerId")
	if playerID == "" {
		writeError(w, http.StatusBadRequest, "playerId is required")
		return
	}
	if !h.requireOwnership(w, r, playerID) {
		return
	}

	page := 1
	if raw := r.URL.Query().Get("page"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			writeError(w, http.StatusBadRequest, "page must be a positive integer")
			return
		}
		page = parsed
	}

	pageSize := 10
	if raw := r.URL.Query().Get("pageSize"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			writeError(w, http.StatusBadRequest, "pageSize must be a positive integer")
			return
		}
		pageSize = parsed
	}
	if pageSize > 50 {
		pageSize = 50
	}

	result, err := h.gameService.ListReports(playerID, page, pageSize)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handlers) MarkReportsRead(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
		ReportID string `json:"reportId"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	var state game.GameState
	var err error

	if payload.ReportID != "" {
		// 标记单条
		state, err = h.gameService.MarkSingleReportRead(payload.PlayerID, payload.ReportID)
	} else {
		// 标记全部
		state, err = h.gameService.MarkReportsRead(payload.PlayerID)
	}

	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"state": state})
}

func (h *Handlers) DeleteReport(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
		ReportID string `json:"reportId"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	state, err := h.gameService.DeleteReport(payload.PlayerID, payload.ReportID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"state": state})
}

func (h *Handlers) DeleteAllReports(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	state, err := h.gameService.DeleteAllReports(payload.PlayerID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"state": state})
}

func (h *Handlers) ListMails(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("playerId")
	if playerID == "" {
		writeError(w, http.StatusBadRequest, "playerId is required")
		return
	}
	if !h.requireOwnership(w, r, playerID) {
		return
	}

	page, pageSize, ok := parsePageQuery(w, r)
	if !ok {
		return
	}
	result, err := h.gameService.ListMails(playerID, page, pageSize)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handlers) GetMail(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("playerId")
	if playerID == "" {
		writeError(w, http.StatusBadRequest, "playerId is required")
		return
	}
	if !h.requireOwnership(w, r, playerID) {
		return
	}

	mailID := r.PathValue("mailId")
	mail, err := h.gameService.GetMail(playerID, mailID)
	if err != nil {
		writeError(w, http.StatusNotFound, "mail not found")
		return
	}
	writeJSON(w, http.StatusOK, mail)
}

func (h *Handlers) DeleteMail(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	if err := h.gameService.DeleteMail(payload.PlayerID, r.PathValue("mailId")); err != nil {
		writeError(w, http.StatusNotFound, "mail not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handlers) AdminSendMail(w http.ResponseWriter, r *http.Request) {
	var payload game.SendMailRequest
	if !decodeJSON(w, r, &payload) {
		return
	}
	payload.SenderType = "gm"
	if payload.SenderName == "" {
		payload.SenderName = "Hero3 GM"
	}
	if payload.SourceType == "" {
		payload.SourceType = "manual"
	}

	mail, err := h.gameService.SendMail(payload)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, game.ErrPlayerNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, mail)
}

func (h *Handlers) AdminPlayerMails(w http.ResponseWriter, r *http.Request) {
	playerID := r.PathValue("playerId")
	if playerID == "" {
		writeError(w, http.StatusBadRequest, "playerId is required")
		return
	}
	page, pageSize, ok := parsePageQuery(w, r)
	if !ok {
		return
	}
	result, err := h.gameService.ListMails(playerID, page, pageSize)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func parsePageQuery(w http.ResponseWriter, r *http.Request) (int, int, bool) {
	page := 1
	if raw := r.URL.Query().Get("page"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			writeError(w, http.StatusBadRequest, "page must be a positive integer")
			return 0, 0, false
		}
		page = parsed
	}

	pageSize := 10
	if raw := r.URL.Query().Get("pageSize"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			writeError(w, http.StatusBadRequest, "pageSize must be a positive integer")
			return 0, 0, false
		}
		pageSize = parsed
	}
	if pageSize > 50 {
		pageSize = 50
	}
	return page, pageSize, true
}

func (h *Handlers) UpgradeBuilding(w http.ResponseWriter, r *http.Request) {
	var payload upgradeBuildingRequest
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	state, err := h.gameService.UpgradeBuilding(payload.PlayerID, payload.BuildingID)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrBuildingNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrInsufficientRes):
			status = http.StatusUnprocessableEntity
		case errors.Is(err, game.ErrAlreadyUpgrading):
			status = http.StatusConflict
		case errors.Is(err, game.ErrMaxLevel):
			status = http.StatusUnprocessableEntity
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"state": state})
}

// --- Boost Handlers ---

func (h *Handlers) PurchaseBoost(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID   string `json:"playerId"`
		Multiplier int    `json:"multiplier"` // 2, 4, 8, 16
		Hours      int    `json:"hours"`      // 1, 6, 12, 24
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	state, err := h.gameService.PurchaseBoost(payload.PlayerID, payload.Multiplier, payload.Hours)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrInsufficientCityGold):
			status = http.StatusUnprocessableEntity
		case errors.Is(err, game.ErrBoostActive):
			status = http.StatusConflict
		case errors.Is(err, game.ErrInvalidBoost):
			status = http.StatusBadRequest
		case errors.Is(err, game.ErrInvalidDuration):
			status = http.StatusBadRequest
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"state": state})
}

func (h *Handlers) BoostPrices(w http.ResponseWriter, r *http.Request) {
	multipliers := []int{2, 4, 8, 16}
	hours := []int{1, 6, 12, 24}

	prices := map[string]int{}
	for _, m := range multipliers {
		for _, h := range hours {
			key := fmt.Sprintf("%dx_%dh", m, h)
			prices[key] = game.GetBoostCost(m, h)
		}
	}

	writeJSON(w, http.StatusOK, prices)
}

func (h *Handlers) PurchaseCapacityBoost(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID   string `json:"playerId"`
		Multiplier int    `json:"multiplier"`
		Hours      int    `json:"hours"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	state, err := h.gameService.PurchaseCapacityBoost(payload.PlayerID, payload.Multiplier, payload.Hours)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrInsufficientCityGold):
			status = http.StatusUnprocessableEntity
		case errors.Is(err, game.ErrBoostActive):
			status = http.StatusConflict
		case errors.Is(err, game.ErrInvalidBoost):
			status = http.StatusBadRequest
		case errors.Is(err, game.ErrInvalidDuration):
			status = http.StatusBadRequest
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"state": state})
}

func (h *Handlers) UpgradeBuildingBatch(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	state, upgraded, err := h.gameService.UpgradeBuildingBatch(payload.PlayerID)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrInsufficientRes):
			status = http.StatusUnprocessableEntity
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"state": state, "upgraded": upgraded})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return false
	}
	return true
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// --- Gold Handlers ---

func (h *Handlers) AddAccountGold(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		AccountID string `json:"accountId"`
		Amount    int    `json:"amount"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if payload.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "amount must be positive")
		return
	}

	if err := h.gameService.AddAccountGoldAdmin(payload.AccountID, payload.Amount); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	account, _ := h.gameService.GetAccountByID(payload.AccountID)
	writeJSON(w, http.StatusOK, map[string]any{"gold": account.Gold})
}

func (h *Handlers) AdminGoldLedger(w http.ResponseWriter, r *http.Request) {
	limit := 200
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	filter := game.GoldLedgerFilter{
		AccountID: r.URL.Query().Get("accountId"),
		PlayerID:  r.URL.Query().Get("playerId"),
		Currency:  r.URL.Query().Get("currency"),
		RefType:   r.URL.Query().Get("refType"),
		Limit:     limit,
	}
	if raw := r.URL.Query().Get("from"); raw != "" {
		if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			filter.From = parsed
		}
	}
	if raw := r.URL.Query().Get("to"); raw != "" {
		if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			filter.To = parsed
		}
	}
	entries, err := h.gameService.ListGoldLedger(filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": entries})
}

func (h *Handlers) AddGold(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
		Amount   int    `json:"amount"`
		Reason   string `json:"reason"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}

	state, err := h.gameService.AddGold(payload.PlayerID, payload.Amount, payload.Reason)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrInvalidGoldAmount):
			status = http.StatusUnprocessableEntity
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"state": state})
}

func (h *Handlers) DeductGold(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
		Amount   int    `json:"amount"`
		Reason   string `json:"reason"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}

	state, err := h.gameService.DeductGold(payload.PlayerID, payload.Amount, payload.Reason)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrInsufficientCityGold):
			status = http.StatusUnprocessableEntity
		case errors.Is(err, game.ErrInvalidGoldAmount):
			status = http.StatusUnprocessableEntity
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"state": state})
}

func (h *Handlers) ExchangeGold(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		AccountID string `json:"accountId"`
		PlayerID  string `json:"playerId"`
		Amount    int    `json:"amount"` // 金币数量
	}
	if !decodeJSON(w, r, &payload) {
		return
	}

	// 校验账户归属和玩家归属
	if !h.requireAccount(w, r, payload.AccountID) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	state, err := h.gameService.ExchangeGoldToCityGold(payload.AccountID, payload.PlayerID, payload.Amount)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrAccountNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrInvalidGoldAmount):
			status = http.StatusUnprocessableEntity
		case errors.Is(err, game.ErrInsufficientGold):
			status = http.StatusUnprocessableEntity
		case errors.Is(err, game.ErrExchangeCooldown):
			status = http.StatusTooManyRequests
		}
		writeError(w, status, err.Error())
		return
	}

	// 读取最新账户信息，返回更新后的账户金币
	account, _ := h.gameService.GetAccountByID(payload.AccountID)
	writeJSON(w, http.StatusOK, map[string]any{"state": state, "accountGold": account.Gold})
}

func (h *Handlers) ReverseExchangeGold(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		AccountID      string `json:"accountId"`
		PlayerID       string `json:"playerId"`
		CityGoldAmount int    `json:"cityGoldAmount"` // 要消耗的城金数量
	}
	if !decodeJSON(w, r, &payload) {
		return
	}

	// 校验账户归属和玩家归属
	if !h.requireAccount(w, r, payload.AccountID) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	state, err := h.gameService.ExchangeCityGoldToGold(payload.AccountID, payload.PlayerID, payload.CityGoldAmount)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrAccountNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrInvalidGoldAmount):
			status = http.StatusUnprocessableEntity
		case errors.Is(err, game.ErrInsufficientGold):
			status = http.StatusUnprocessableEntity
		case errors.Is(err, game.ErrInsufficientCityGold):
			status = http.StatusUnprocessableEntity
		case errors.Is(err, game.ErrExchangeCooldown):
			status = http.StatusTooManyRequests
		}
		writeError(w, status, err.Error())
		return
	}

	// 读取最新账户信息，返回更新后的账户金币
	account, _ := h.gameService.GetAccountByID(payload.AccountID)
	writeJSON(w, http.StatusOK, map[string]any{"state": state, "accountGold": account.Gold})
}

// --- Buff 管理 ---

func (h *Handlers) GrantBuff(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string  `json:"playerId"`
		Key      string  `json:"key"`
		Value    float64 `json:"value"`
		Mode     string  `json:"mode"`
		Hours    int     `json:"hours"` // 0 = 永久
		Note     string  `json:"note"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}

	if payload.PlayerID == "" || payload.Key == "" {
		writeError(w, http.StatusBadRequest, "playerId and key are required")
		return
	}
	if payload.Mode == "" {
		payload.Mode = "percentAdd"
	}

	state, err := h.gameService.GrantBuff(payload.PlayerID, payload.Key, payload.Value, payload.Mode, payload.Hours, payload.Note)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"state": state})
}

func (h *Handlers) RevokeBuff(w http.ResponseWriter, r *http.Request) {
	buffId := r.PathValue("buffId")
	playerID := r.URL.Query().Get("playerId")

	if buffId == "" || playerID == "" {
		writeError(w, http.StatusBadRequest, "buffId and playerId are required")
		return
	}

	state, err := h.gameService.RevokeBuff(playerID, buffId)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"state": state})
}

// --- MiniGame Records ---

func (h *Handlers) SaveMiniGameRecord(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID     string `json:"playerId"`
		GameType     string `json:"gameType"`
		ResultName   string `json:"resultName"`
		Rarity       string `json:"rarity"`
		RewardUnit   string `json:"rewardUnit"`
		RewardAmount int    `json:"rewardAmount"`
		BetUnit      string `json:"betUnit"`
		BetAmount    int    `json:"betAmount"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}

	if payload.PlayerID == "" || payload.GameType == "" || payload.ResultName == "" {
		writeError(w, http.StatusBadRequest, "playerId, gameType, and resultName are required")
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	record, err := h.gameService.SaveMiniGameRecord(
		payload.PlayerID, payload.GameType, payload.ResultName,
		payload.Rarity, payload.RewardUnit, payload.RewardAmount,
		payload.BetUnit, payload.BetAmount,
	)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, record)
}

func (h *Handlers) AdminMiniGameRecords(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("playerId")
	if playerID == "" {
		writeError(w, http.StatusBadRequest, "playerId is required")
		return
	}

	summary, err := h.gameService.GetMiniGameRecords(playerID, 200)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

// --- 归属校验辅助函数 ---

// requireOwnership 校验当前请求的 accountID 是否拥有 playerID
// admin 请求直接通过，否则需要 JWT 中的 accountID 拥有该 playerID
// 返回 false 时已经写入了错误响应
func (h *Handlers) requireOwnership(w http.ResponseWriter, r *http.Request, playerID string) bool {
	if auth.IsAdminFromContext(r.Context()) {
		return true
	}

	accountID, ok := auth.AccountIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return false
	}

	owns, err := h.gameService.OwnsPlayer(accountID, playerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ownership check failed")
		return false
	}
	if !owns {
		writeError(w, http.StatusForbidden, "you don't own this player")
		return false
	}
	return true
}

// requireAccount 校验当前请求的 accountID 是否匹配指定 accountID
// admin 请求直接通过
func (h *Handlers) requireAccount(w http.ResponseWriter, r *http.Request, accountID string) bool {
	if auth.IsAdminFromContext(r.Context()) {
		return true
	}

	ctxAccountID, ok := auth.AccountIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return false
	}

	if ctxAccountID != accountID {
		writeError(w, http.StatusForbidden, "account mismatch")
		return false
	}
	return true
}

// --- Generals 配置（GM） ---

func (h *Handlers) AdminGeneralsConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.gameService.GetGeneralsConfig())
}

func (h *Handlers) UpdateAdminGeneralsConfig(w http.ResponseWriter, r *http.Request) {
	var payload game.GeneralsConfig
	if !decodeJSON(w, r, &payload) {
		return
	}
	if err := h.gameService.UpdateGeneralsConfig(payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, h.gameService.GetGeneralsConfig())
}

// AdminGeneralTraitRegistry 返回所有已注册的特性元信息（GM 后台用来选择特性）
func (h *Handlers) AdminGeneralTraitRegistry(w http.ResponseWriter, r *http.Request) {
	type traitMeta struct {
		ID          string               `json:"id"`
		Name        string               `json:"name"`
		Description string               `json:"description"`
		ParamSchema []general.ParamField `json:"paramSchema"`
	}
	traits := general.All()
	out := make([]traitMeta, 0, len(traits))
	for _, t := range traits {
		out = append(out, traitMeta{
			ID:          t.ID(),
			Name:        t.Name(),
			Description: t.Description(general.Params{}),
			ParamSchema: t.ParamSchema(),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"traits": out})
}
