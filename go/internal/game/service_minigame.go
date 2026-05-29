package game

import (
	"strings"
	"time"
)

// MiniGameRecord 小游戏记录（钓鱼/赌博等）
type MiniGameRecord struct {
	ID           string `json:"id"`
	PlayerID     string `json:"playerId"`
	GameType     string `json:"gameType"`     // "fishing" | "gambling"
	ResultName   string `json:"resultName"`   // "金龙鱼" / "猜大 赢 ×2"
	Rarity       string `json:"rarity"`       // "common" | "rare" | "epic" | "legendary"
	RewardUnit   string `json:"rewardUnit"`   // 赢得的兵种名称
	RewardAmount int    `json:"rewardAmount"` // 赢得的数量
	BetUnit      string `json:"betUnit,omitempty"`   // 押注的兵种名称（赌博用）
	BetAmount    int    `json:"betAmount,omitempty"` // 押注的数量（赌博用）
	CreatedAt    string `json:"createdAt"`
}

// MiniGameSummary GM 查询用的汇总信息
type MiniGameSummary struct {
	TotalRecords int                    `json:"totalRecords"`
	Records      []MiniGameRecord       `json:"records"`
	RewardTotals map[string]int         `json:"rewardTotals"` // 兵种名 → 总可兑换数量
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

	if err := s.repo.SaveMiniGameRecord(record); err != nil {
		return MiniGameRecord{}, err
	}

	return record, nil
}

// GetMiniGameRecords GM 查询某玩家的小游戏记录（含汇总）
func (s *Service) GetMiniGameRecords(playerID string, limit int) (MiniGameSummary, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return MiniGameSummary{}, ErrPlayerNotFound
	}

	if limit <= 0 {
		limit = 100
	}

	records, err := s.repo.ListMiniGameRecords(playerID, limit)
	if err != nil {
		return MiniGameSummary{}, err
	}

	// 汇总各兵种可兑换总量
	totals := map[string]int{}
	for _, r := range records {
		if r.RewardUnit != "" && r.RewardAmount > 0 {
			totals[r.RewardUnit] += r.RewardAmount
		}
	}

	return MiniGameSummary{
		TotalRecords: len(records),
		Records:      records,
		RewardTotals: totals,
	}, nil
}
