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

	state, err := s.repo.GetState(playerID)
	if err != nil {
		return GameState{}, err
	}

	now := time.Now()

	buff := Buff{
		ID:        "buff_" + randomID(8),
		Source:    "gm",
		Key:       key,
		Value:     value,
		Mode:      mode,
		CreatedAt: now.UTC().Format(resourceDateLayout),
		Note:      note,
	}

	if hours > 0 {
		buff.ExpiresAt = now.Add(time.Duration(hours) * time.Hour).UTC().Format(resourceDateLayout)
	}

	state.Buffs = append(state.Buffs, buff)
	state.ServerTime = now.UTC().Format(resourceDateLayout)

	if err := s.repo.SaveState(state, now); err != nil {
		return GameState{}, err
	}

	return state, nil
}

// RevokeBuff 撤销玩家的一个 buff
func (s *Service) RevokeBuff(playerID string, buffID string) (GameState, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return GameState{}, ErrPlayerNotFound
	}

	state, err := s.repo.GetState(playerID)
	if err != nil {
		return GameState{}, err
	}

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
		return GameState{}, errors.New("buff not found")
	}

	state.Buffs = remaining
	now := time.Now()
	state.ServerTime = now.UTC().Format(resourceDateLayout)

	if err := s.repo.SaveState(state, now); err != nil {
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
