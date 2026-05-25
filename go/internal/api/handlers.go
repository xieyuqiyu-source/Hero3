package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"hero3/internal/combat"
	"hero3/internal/config"
	"hero3/internal/game"
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
	state, err := h.gameService.GetState(r.URL.Query().Get("playerId"))
	if err != nil {
		writeError(w, http.StatusNotFound, "player not found")
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func (h *Handlers) RegisterAccount(w http.ResponseWriter, r *http.Request) {
	var payload accountRequest
	if !decodeJSON(w, r, &payload) {
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

	writeJSON(w, http.StatusCreated, map[string]any{
		"accountId": account.ID,
		"username":  account.Username,
		"gold":      account.Gold,
	})
}

func (h *Handlers) LoginAccount(w http.ResponseWriter, r *http.Request) {
	var payload accountRequest
	if !decodeJSON(w, r, &payload) {
		return
	}

	account, err := h.gameService.LoginAccount(payload.Username, payload.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"accountId": account.ID,
		"username":  account.Username,
		"gold":      account.Gold,
	})
}

func (h *Handlers) AccountPlayers(w http.ResponseWriter, r *http.Request) {
	players, err := h.gameService.ListPlayers(r.PathValue("accountId"))
	if err != nil {
		writeError(w, http.StatusNotFound, "account not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"players": players})
}

func (h *Handlers) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	if err := h.gameService.DeleteAccount(r.PathValue("accountId")); err != nil {
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
	if err := h.gameService.DeletePlayer(r.PathValue("playerId")); err != nil {
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

	state, err := h.gameService.FillResources(payload.PlayerID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"state": state})
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

	state, err := h.gameService.InstantCompleteRecruit(payload.PlayerID, payload.QueueID)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound):
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"state": state})
}

func (h *Handlers) NpcCities(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("playerId")
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
		case errors.Is(err, game.ErrInsufficientArmy):
			status = http.StatusUnprocessableEntity
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

func (h *Handlers) MarkReportsRead(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
		ReportID string `json:"reportId"`
	}
	if !decodeJSON(w, r, &payload) {
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

	state, err := h.gameService.DeleteAllReports(payload.PlayerID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"state": state})
}

func (h *Handlers) UpgradeBuilding(w http.ResponseWriter, r *http.Request) {
	var payload upgradeBuildingRequest
	if !decodeJSON(w, r, &payload) {
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

func (h *Handlers) UpgradeBuildingBatch(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
	}
	if !decodeJSON(w, r, &payload) {
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
		case errors.Is(err, game.ErrInsufficientGold):
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

	writeJSON(w, http.StatusOK, map[string]any{"state": state})
}
