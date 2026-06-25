package game

import (
	"errors"
	"strings"
	"time"
)

// GrantBuff 给玩家发放一个 buff（GM/活动/系统均可调用）
func (s *Service) GrantBuff(playerID string, key string, value float64, mode string, hours int, note string) (GameState, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return GameState{}, ErrPlayerNotFound
	}

	// 校验 key 是否合法
	if !IsValidStatKey(key) {
		return GameState{}, errors.New("invalid stat key: " + key)
	}

	// 校验 mode 是否合法
	if mode != "flat" && mode != "percentAdd" && mode != "percentMultiply" {
		return GameState{}, errors.New("invalid mode: must be flat, percentAdd, or percentMultiply")
	}

	result, err := s.GrantRewards(playerID, []Reward{{
		Type:   RewardTypeBuff,
		ID:     key,
		Amount: 1,
		Metadata: map[string]any{
			"value":  value,
			"mode":   mode,
			"hours":  hours,
			"source": "gm",
			"note":   note,
		},
	}}, RewardGrantContext{
		PlayerID: playerID,
		RefType:  "grant_buff",
		RefID:    "buff_" + randomID(8),
		Reason:   note,
	})
	if err != nil {
		return GameState{}, err
	}

	return result.State, nil
}

// RevokeBuff 撤销玩家的一个 buff
func (s *Service) RevokeBuff(playerID string, buffID string) (GameState, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return GameState{}, ErrPlayerNotFound
	}

	now := time.Now()
	state, err := s.repo.UpdatePlayerState(playerID, now, func(state *GameState) error {
		found := false
		remaining := state.Buffs[:0]
		for _, b := range state.Buffs {
			if b.ID == buffID {
				found = true
				continue
			}
			remaining = append(remaining, b)
		}

		if !found {
			return errors.New("buff not found")
		}

		state.Buffs = remaining
		state.ServerTime = now.UTC().Format(resourceDateLayout)
		return nil
	})
	if err != nil {
		return GameState{}, err
	}

	return state, nil
}

// CleanExpiredBuffs 清理过期的 buff（在结算时调用）
func cleanExpiredBuffs(state *GameState, now time.Time) {
	if len(state.Buffs) == 0 {
		return
	}
	remaining := state.Buffs[:0]
	for _, b := range state.Buffs {
		if b.ExpiresAt == "" {
			remaining = append(remaining, b)
			continue
		}
		if t, err := time.Parse(resourceDateLayout, b.ExpiresAt); err == nil && now.After(t) {
			continue // 已过期，丢弃
		}
		remaining = append(remaining, b)
	}
	state.Buffs = remaining
}
