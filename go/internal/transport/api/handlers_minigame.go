// 本文件实现万象幻境小游戏相关 HTTP 接口处理器。
package api

import (
	"errors"
	"hero3/internal/app/game"
	"net/http"
)

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
	if payload.GameType == "gambling" || payload.GameType == "slot" {
		writeError(w, http.StatusUnprocessableEntity, "该小游戏记录必须通过结算接口生成")
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

func (h *Handlers) ListMiniGameRecords(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("playerId")
	if playerID == "" {
		writeError(w, http.StatusBadRequest, "playerId is required")
		return
	}
	if !h.requireOwnership(w, r, playerID) {
		return
	}
	limit, offset := parseLimitOffset(r, 100, 500)
	summary, err := h.gameService.GetMiniGameRecords(playerID, r.URL.Query().Get("gameType"), limit, offset)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (h *Handlers) UseFishingBait(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
		BaitID   string `json:"baitId"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}
	result, err := h.gameService.UseFishingBait(payload.PlayerID, payload.BaitID)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrInsufficientCityGold), errors.Is(err, game.ErrInvalidBait):
			status = http.StatusUnprocessableEntity
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handlers) ResolveGamblingRound(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID    string `json:"playerId"`
		BetUnitType string `json:"betUnitType"`
		BetAmount   int    `json:"betAmount"`
		BetID       string `json:"betId"`
		ExactNumber int    `json:"exactNumber"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}
	result, err := h.gameService.ResolveGamblingRound(payload.PlayerID, payload.BetUnitType, payload.BetAmount, payload.BetID, payload.ExactNumber)
	if err != nil {
		status := http.StatusBadRequest
		message := err.Error()
		switch {
		case errors.Is(err, game.ErrPlayerNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrInsufficientArmy):
			status = http.StatusUnprocessableEntity
			message = "押注兵力不足"
		case errors.Is(err, game.ErrUnitNotFound):
			status = http.StatusUnprocessableEntity
			message = "押注兵种不存在"
		case errors.Is(err, game.ErrNonCombatUnit):
			status = http.StatusUnprocessableEntity
			message = "该兵种不能押注"
		case errors.Is(err, game.ErrInvalidAmount), errors.Is(err, game.ErrInvalidMiniGame):
			status = http.StatusUnprocessableEntity
		}
		writeError(w, status, message)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handlers) ResolveSlotRound(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID    string `json:"playerId"`
		BetUnitType string `json:"betUnitType"`
		BetAmount   int    `json:"betAmount"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}
	result, err := h.gameService.ResolveSlotRound(payload.PlayerID, payload.BetUnitType, payload.BetAmount)
	if err != nil {
		status := http.StatusBadRequest
		message := err.Error()
		switch {
		case errors.Is(err, game.ErrPlayerNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrOperationTooFast):
			status = http.StatusTooManyRequests
		case errors.Is(err, game.ErrInsufficientArmy):
			status = http.StatusUnprocessableEntity
			message = "押注兵力不足"
		case errors.Is(err, game.ErrUnitNotFound):
			status = http.StatusUnprocessableEntity
			message = "押注兵种不存在"
		case errors.Is(err, game.ErrNonCombatUnit):
			status = http.StatusUnprocessableEntity
			message = "该兵种不能押注"
		case errors.Is(err, game.ErrMiniGameBetTooLow):
			status = http.StatusUnprocessableEntity
			message = "押注数量太低"
		case errors.Is(err, game.ErrMiniGameBetTooHigh):
			status = http.StatusUnprocessableEntity
			message = "押注数量超过上限"
		case errors.Is(err, game.ErrInvalidAmount), errors.Is(err, game.ErrInvalidMiniGame):
			status = http.StatusUnprocessableEntity
		}
		writeError(w, status, message)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handlers) RedeemMiniGameReward(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
		RecordID string `json:"recordId"`
		Amount   int    `json:"amount"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}
	result, err := h.gameService.RedeemMiniGameReward(payload.PlayerID, payload.RecordID, payload.Amount)
	if err != nil {
		status := http.StatusBadRequest
		message := err.Error()
		switch {
		case errors.Is(err, game.ErrMiniGameNotFound), errors.Is(err, game.ErrPlayerNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrCrossFactionReward):
			message = "该奖励不是当前阵营兵种，驻防增援系统完成后即可兑换"
		case errors.Is(err, game.ErrMiniGameStockShort):
			message = "可兑换库存不足"
		}
		writeError(w, status, message)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handlers) RedeemAllMiniGameRewards(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
		GameType string `json:"gameType"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}
	result, err := h.gameService.RedeemAllFactionMiniGameRewards(payload.PlayerID, payload.GameType)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, game.ErrPlayerNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handlers) AdminMiniGameRecords(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("playerId")
	if playerID == "" {
		writeError(w, http.StatusBadRequest, "playerId is required")
		return
	}

	limit, offset := parseLimitOffset(r, 100, 500)
	summary, err := h.gameService.GetMiniGameRecords(playerID, r.URL.Query().Get("gameType"), limit, offset)
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
