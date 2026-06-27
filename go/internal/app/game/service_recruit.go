package game

import (
	"strings"
	"time"
)

func (s *Service) Recruit(playerID string, unitID string, amount int) (GameState, error) {
	playerID = strings.TrimSpace(playerID)
	unitID = strings.TrimSpace(unitID)
	if playerID == "" {
		return GameState{}, ErrPlayerNotFound
	}
	if unitID == "" {
		return GameState{}, ErrUnitNotFound
	}
	if amount <= 0 || amount > 100000 {
		return GameState{}, ErrInvalidAmount
	}

	now := time.Now()
	var before, after coreAssetSnapshot
	state, err := s.repo.UpdateRecruitState(playerID, now, func(state *GameState) error {
		nextState, _ := settleResources(*state, now)
		*state = nextState
		before = snapshotCoreAssets(state)

		if len(state.RecruitQueues) >= 5 {
			return ErrQueueFull
		}

		unitConfig, exists := GetUnitConfig(state.Player.Faction, unitID)
		if !exists {
			return ErrUnitNotFound
		}

		totalCost := make(map[string]int, len(unitConfig.Cost))
		for resType, costPer := range unitConfig.Cost {
			totalCost[resType] = costPer * amount
		}
		if err := spendResources(state, totalCost); err != nil {
			return err
		}

		modSources := CollectModifierSources(state)
		totalSeconds := calculateRecruitDurationSeconds(unitConfig, amount, now, modSources)
		queueStart := now
		for _, q := range state.RecruitQueues {
			if parsed, err := time.Parse(resourceDateLayout, q.EndsAt); err == nil && parsed.After(queueStart) {
				queueStart = parsed
			}
		}
		endsAt := queueStart.Add(time.Duration(totalSeconds) * time.Second).UTC().Format(resourceDateLayout)

		queue := RecruitQueue{
			ID:       "rq_" + randomID(8),
			UnitType: unitID,
			Amount:   amount,
			EndsAt:   endsAt,
		}
		state.RecruitQueues = append(state.RecruitQueues, queue)

		state.ResourceSettledAt = now.UTC().Format(resourceDateLayout)
		state.ServerTime = now.UTC().Format(resourceDateLayout)
		after = snapshotCoreAssets(state)
		return nil
	})
	if err != nil {
		return GameState{}, err
	}
	s.publishCoreAssetDiff(playerID, "recruit_start", unitID, before, after, now)

	return state, nil
}

// InstantCompleteRecruit 极速完成征兵队列（消耗城金）
func (s *Service) InstantCompleteRecruit(playerID string, queueID string) (GameState, error) {
	playerID = strings.TrimSpace(playerID)
	queueID = strings.TrimSpace(queueID)
	if playerID == "" {
		return GameState{}, ErrPlayerNotFound
	}
	if queueID == "" {
		return GameState{}, ErrQueueFull
	}

	now := time.Now()
	cost := 0
	var before, after coreAssetSnapshot
	state, err := s.repo.UpdateRecruitState(playerID, now, func(state *GameState) error {
		nextState, _ := settleResources(*state, now)
		*state = nextState
		before = snapshotCoreAssets(state)

		queueIdx := -1
		for i, queue := range state.RecruitQueues {
			if queue.ID == queueID {
				queueIdx = i
				break
			}
		}
		if queueIdx == -1 {
			return ErrQueueFull
		}

		queue := state.RecruitQueues[queueIdx]
		endsAt, err := time.Parse(resourceDateLayout, queue.EndsAt)
		if err != nil {
			return ErrQueueFull
		}
		remainingSecs := int(endsAt.Sub(now).Seconds())
		if remainingSecs > 0 {
			cost = speedUpCost(remainingSecs)
			if int(state.CityGold) < cost {
				return ErrInsufficientCityGold
			}
			state.CityGold -= FlexInt(cost)
		}

		AddArmyUnit(state, queue.UnitType, queue.Amount)
		state.RecruitQueues = append(state.RecruitQueues[:queueIdx], state.RecruitQueues[queueIdx+1:]...)

		if queueIdx < len(state.RecruitQueues) {
			prevEnd := now
			if queueIdx > 0 {
				if parsed, err := time.Parse(resourceDateLayout, state.RecruitQueues[queueIdx-1].EndsAt); err == nil {
					prevEnd = parsed
				}
			}
			for j := queueIdx; j < len(state.RecruitQueues); j++ {
				rq := &state.RecruitQueues[j]
				unitCfg, exists := GetUnitConfig(state.Player.Faction, rq.UnitType)
				if !exists {
					continue
				}
				queueModSources := CollectModifierSources(state)
				durationSeconds := calculateRecruitDurationSeconds(unitCfg, rq.Amount, now, queueModSources)
				duration := time.Duration(durationSeconds) * time.Second
				newEnd := prevEnd.Add(duration)
				rq.EndsAt = newEnd.UTC().Format(resourceDateLayout)
				prevEnd = newEnd
			}
		}

		state.ResourceSettledAt = now.UTC().Format(resourceDateLayout)
		state.ServerTime = now.UTC().Format(resourceDateLayout)
		after = snapshotCoreAssets(state)
		return nil
	})
	if err != nil {
		return GameState{}, err
	}
	if cost > 0 {
		s.recordLedger(GoldLedgerEntry{
			PlayerID:     playerID,
			Currency:     LedgerCurrencyCityGold,
			Direction:    LedgerDirectionDebit,
			Amount:       cost,
			BalanceAfter: int(state.CityGold),
			RefType:      LedgerRefInstantRecruit,
			RefID:        queueID,
		})
		s.publishCurrencyChanged(playerID, "", queueID, LedgerRefInstantRecruit)
	}
	s.publishCoreAssetDiff(playerID, LedgerRefInstantRecruit, queueID, before, after, now)

	return state, nil
}
