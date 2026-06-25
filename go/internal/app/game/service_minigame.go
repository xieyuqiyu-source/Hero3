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
	GrantedRewards []Reward       `json:"grantedRewards,omitempty"`
}

type MiniGameRedeemAllResult struct {
	State           GameState      `json:"state"`
	RedeemedUnits   map[string]int `json:"redeemedUnits"`
	RedeemedAmount  int            `json:"redeemedAmount"`
	RedeemedRecords int            `json:"redeemedRecords"`
	SkippedUnits    map[string]int `json:"skippedUnits"`
	SkippedRecords  int            `json:"skippedRecords"`
	GrantedRewards  []Reward       `json:"grantedRewards,omitempty"`
}

type FishingBaitUseResult struct {
	State          GameState `json:"state"`
	BaitID         string    `json:"baitId"`
	CityGoldCost   int       `json:"cityGoldCost"`
	CityGoldRemain int       `json:"cityGoldRemain"`
}

func fishingBaitCost(baitID string) (int, bool) {
	return GetFishingBaitCost(baitID)
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

func (s *Service) UseFishingBait(playerID string, baitID string) (FishingBaitUseResult, error) {
	playerID = strings.TrimSpace(playerID)
	baitID = strings.TrimSpace(baitID)
	if playerID == "" {
		return FishingBaitUseResult{}, ErrPlayerNotFound
	}
	cost, ok := fishingBaitCost(baitID)
	if !ok {
		return FishingBaitUseResult{}, ErrInvalidBait
	}

	now := time.Now()
	refID := "fishing_bait_" + baitID
	var state GameState
	if cost > 0 {
		var err error
		state, err = s.repo.UpdatePlayerState(playerID, now, func(state *GameState) error {
			if int(state.CityGold) < cost {
				return ErrInsufficientCityGold
			}
			state.CityGold -= FlexInt(cost)
			state.ServerTime = now.UTC().Format(resourceDateLayout)
			return nil
		})
		if err != nil {
			return FishingBaitUseResult{}, err
		}
		s.recordLedger(GoldLedgerEntry{
			PlayerID:     playerID,
			Currency:     LedgerCurrencyCityGold,
			Direction:    LedgerDirectionDebit,
			Amount:       cost,
			BalanceAfter: int(state.CityGold),
			RefType:      LedgerRefMiniGameBait,
			RefID:        refID,
			Reason:       "钓鱼鱼饵消耗",
		})
		s.publishCurrencyChanged(playerID, "", refID, LedgerRefMiniGameBait)
	} else {
		var err error
		state, err = s.repo.GetState(playerID)
		if err != nil {
			return FishingBaitUseResult{}, err
		}
	}

	return FishingBaitUseResult{
		State:          state,
		BaitID:         baitID,
		CityGoldCost:   cost,
		CityGoldRemain: int(state.CityGold),
	}, nil
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
	redeemedAt := time.Now()
	var selectedRecord MiniGameRecord
	var redeemedUnitID string
	var redeemedUnit string
	var grantedRewards []Reward
	var applyResult RewardApplyResult
	state, _, err := s.repo.UpdateMiniGamePlayerState(playerID, redeemedAt, func(state *GameState, records []MiniGameRecord) ([]MiniGameRecord, error) {
		for i := range records {
			record := records[i]
			if record.ID != recordID {
				continue
			}
			if record.GameType != "fishing" || record.RewardUnit == "" || record.RewardAmount <= 0 {
				return nil, ErrInvalidMiniGame
			}
			if record.RemainingAmount <= 0 || amount > record.RemainingAmount {
				return nil, ErrMiniGameStockShort
			}
			unitID, unitCfg, ok := FindFactionUnitByName(state.Player.Faction, record.RewardUnit)
			if !ok {
				return nil, ErrCrossFactionReward
			}
			reward := Reward{Type: RewardTypeUnit, ID: unitID, Amount: amount}
			record.RemainingAmount -= amount
			records[i] = record
			effectResult, err := ExecuteEffectsOnState(state, rewardsToEffects("minigame", []Reward{reward}), EffectContext{
				PlayerID: playerID,
				RefType:  LedgerRefMiniGameRedeem,
				RefID:    record.ID,
				Reason:   "minigame_redeem",
				Source:   "minigame",
			}, redeemedAt)
			if err != nil {
				return nil, err
			}
			state.ServerTime = redeemedAt.UTC().Format(resourceDateLayout)
			selectedRecord = record
			redeemedUnitID = unitID
			redeemedUnit = unitCfg.Name
			grantedRewards = []Reward{reward}
			applyResult = effectResult.Reward
			return records, nil
		}
		return nil, ErrMiniGameNotFound
	})
	if err != nil {
		return MiniGameRedeemResult{}, err
	}
	s.flushRewardSideEffects(applyResult)
	result := MiniGameRedeemResult{
		Record:         selectedRecord,
		State:          state,
		RedeemedUnitID: redeemedUnitID,
		RedeemedUnit:   redeemedUnit,
		RedeemedAmount: amount,
		GrantedRewards: grantedRewards,
	}
	s.publishMiniGameRedeemEvents(playerID, result.Record.GameType, result.GrantedRewards, result.Record.ID, result.RedeemedAmount, result.State)
	return result, nil
}

func (s *Service) RedeemAllFactionMiniGameRewards(playerID string, gameType string) (MiniGameRedeemAllResult, error) {
	playerID = strings.TrimSpace(playerID)
	gameType = strings.TrimSpace(gameType)
	if playerID == "" {
		return MiniGameRedeemAllResult{}, ErrPlayerNotFound
	}
	if gameType == "" {
		gameType = "fishing"
	}
	redeemedAt := time.Now()
	redeemedUnits := map[string]int{}
	skippedUnits := map[string]int{}
	grantedRewards := []Reward{}
	redeemedRecords := 0
	skippedRecords := 0
	var applyResult RewardApplyResult
	state, _, err := s.repo.UpdateMiniGamePlayerState(playerID, redeemedAt, func(state *GameState, records []MiniGameRecord) ([]MiniGameRecord, error) {
		applyResult = RewardApplyResult{Granted: map[string]int{}}
		for i := range records {
			record := records[i]
			if record.GameType != gameType || record.RewardUnit == "" || record.RemainingAmount <= 0 {
				continue
			}
			unitID, unitCfg, ok := FindFactionUnitByName(state.Player.Faction, record.RewardUnit)
			if !ok {
				skippedUnits[record.RewardUnit] += record.RemainingAmount
				skippedRecords++
				continue
			}
			amount := record.RemainingAmount
			reward := Reward{Type: RewardTypeUnit, ID: unitID, Amount: amount}
			records[i].RemainingAmount = 0
			effectResult, err := ExecuteEffectsOnState(state, rewardsToEffects("minigame", []Reward{reward}), EffectContext{
				PlayerID: playerID,
				RefType:  LedgerRefMiniGameRedeem,
				RefID:    record.ID,
				Reason:   "minigame_redeem",
				Source:   "minigame",
			}, redeemedAt)
			if err != nil {
				return nil, err
			}
			mergeRewardApplyResult(&applyResult, effectResult.Reward)
			grantedRewards = append(grantedRewards, reward)
			redeemedUnits[unitCfg.Name] += amount
			redeemedRecords++
		}
		state.ServerTime = redeemedAt.UTC().Format(resourceDateLayout)
		return records, nil
	})
	if err != nil {
		return MiniGameRedeemAllResult{}, err
	}
	s.flushRewardSideEffects(applyResult)
	total := 0
	for _, amount := range redeemedUnits {
		total += amount
	}
	result := MiniGameRedeemAllResult{
		State:           state,
		RedeemedUnits:   redeemedUnits,
		RedeemedAmount:  total,
		RedeemedRecords: redeemedRecords,
		SkippedUnits:    skippedUnits,
		SkippedRecords:  skippedRecords,
		GrantedRewards:  grantedRewards,
	}
	s.publishMiniGameRedeemEvents(playerID, gameType, result.GrantedRewards, "", result.RedeemedAmount, result.State)
	return result, nil
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
	if records == nil {
		records = []MiniGameRecord{}
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

func (s *Service) publishMiniGameRedeemEvents(playerID string, gameType string, rewards []Reward, recordID string, redeemedAmount int, state GameState) {
	createdAt := state.ServerTime
	if strings.TrimSpace(createdAt) == "" {
		createdAt = time.Now().UTC().Format(resourceDateLayout)
	}
	s.publishEvent(GameEvent{
		Type:     EventMiniGameRedeemed,
		PlayerID: playerID,
		RefType:  LedgerRefMiniGameRedeem,
		RefID:    recordID,
		Payload: map[string]any{
			"gameType":        gameType,
			"redeemedAmount":  redeemedAmount,
			"redeemedRewards": rewards,
		},
		CreatedAt: createdAt,
	})
}
