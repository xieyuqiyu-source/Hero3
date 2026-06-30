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

	if err := validateBuffModifierSpec(key, mode); err != nil {
		return GameState{}, err
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
	state, err := s.repo.UpdateScopedRewardState(playerID, RewardAssetScope{Buffs: true}, now, func(state *GameState) error {
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
