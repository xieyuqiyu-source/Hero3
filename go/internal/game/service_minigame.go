package game

import (
	"strings"
	"time"
)

// MiniGameRecord 小游戏记录（钓鱼/赌博等）
type MiniGameRecord struct {
	ID              string `json:"id"`
	PlayerID        string `json:"playerId"`
	GameType        string `json:"gameType"`            // "fishing" | "gambling"
	ResultName      string `json:"resultName"`          // "金龙鱼" / "猜大 赢 ×2"
	Rarity          string `json:"rarity"`              // "common" | "rare" | "epic" | "legendary"
	RewardUnit      string `json:"rewardUnit"`          // 赢得的兵种名称
	RewardAmount    int    `json:"rewardAmount"`        // 原始赢得数量
	RemainingAmount int    `json:"remainingAmount"`     // 剩余可兑换数量
	BetUnit         string `json:"betUnit,omitempty"`   // 押注的兵种名称（赌博用）
	BetAmount       int    `json:"betAmount,omitempty"` // 押注的数量（赌博用）
	CreatedAt       string `json:"createdAt"`
}

// MiniGameSummary GM 查询用的汇总信息
type MiniGameSummary struct {
	TotalRecords int              `json:"totalRecords"`
	Limit        int              `json:"limit"`
	Offset       int              `json:"offset"`
	HasMore      bool             `json:"hasMore"`
	Records      []MiniGameRecord `json:"records"`
	RewardTotals map[string]int   `json:"rewardTotals"` // 兵种名 → 总可兑换数量
}

type MiniGameRedeemResult struct {
	Record         MiniGameRecord `json:"record"`
	State          GameState      `json:"state"`
	RedeemedUnitID string         `json:"redeemedUnitId"`
	RedeemedUnit   string         `json:"redeemedUnit"`
	RedeemedAmount int            `json:"redeemedAmount"`
}

// SaveMiniGameRecord 保存一条小游戏记录
func (s *Service) SaveMiniGameRecord(playerID string, gameType string, resultName string, rarity string, rewardUnit string, rewardAmount int, betUnit string, betAmount int) (MiniGameRecord, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return MiniGameRecord{}, ErrPlayerNotFound
	}

	now := time.Now()
	record := MiniGameRecord{
		ID:           "mg_" + randomID(10),
		PlayerID:     playerID,
		GameType:     gameType,
		ResultName:   resultName,
		Rarity:       rarity,
		RewardUnit:   rewardUnit,
		RewardAmount: rewardAmount,
		BetUnit:      betUnit,
		BetAmount:    betAmount,
		CreatedAt:    now.UTC().Format(resourceDateLayout),
	}
	if record.GameType == "fishing" && record.RewardAmount > 0 {
		record.RemainingAmount = record.RewardAmount
	}

	if err := s.repo.SaveMiniGameRecord(record); err != nil {
		return MiniGameRecord{}, err
	}

	return record, nil
}

func (s *Service) RedeemMiniGameReward(playerID string, recordID string, amount int) (MiniGameRedeemResult, error) {
	playerID = strings.TrimSpace(playerID)
	recordID = strings.TrimSpace(recordID)
	if playerID == "" {
		return MiniGameRedeemResult{}, ErrPlayerNotFound
	}
	if recordID == "" {
		return MiniGameRedeemResult{}, ErrMiniGameNotFound
	}
	if amount <= 0 {
		return MiniGameRedeemResult{}, ErrInvalidAmount
	}
	return s.repo.RedeemMiniGameRecord(playerID, recordID, amount, time.Now())
}

// GetMiniGameRecords GM 查询某玩家的小游戏记录（含汇总）
func (s *Service) GetMiniGameRecords(playerID string, gameType string, limit int, offset int) (MiniGameSummary, error) {
	playerID = strings.TrimSpace(playerID)
	gameType = strings.TrimSpace(gameType)
	if playerID == "" {
		return MiniGameSummary{}, ErrPlayerNotFound
	}

	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}

	records, total, err := s.repo.ListMiniGameRecords(playerID, gameType, limit, offset)
	if err != nil {
		return MiniGameSummary{}, err
	}

	// 汇总各兵种可兑换总量
	totals := map[string]int{}
	for _, r := range records {
		if r.RewardUnit != "" && r.RemainingAmount > 0 {
			totals[r.RewardUnit] += r.RemainingAmount
		}
	}

	return MiniGameSummary{
		TotalRecords: total,
		Limit:        limit,
		Offset:       offset,
		HasMore:      offset+len(records) < total,
		Records:      records,
		RewardTotals: totals,
	}, nil
}
