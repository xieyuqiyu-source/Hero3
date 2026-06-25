package game

import (
	"strings"
	"time"
)

func (s *Service) UpgradeBuilding(playerID string, buildingID string) (GameState, error) {
	playerID = strings.TrimSpace(playerID)
	buildingID = strings.TrimSpace(buildingID)
	if playerID == "" || buildingID == "" {
		return GameState{}, ErrBuildingNotFound
	}

	now := time.Now()
	var before, after coreAssetSnapshot
	state, err := s.repo.UpdatePlayerState(playerID, now, func(state *GameState) error {
		nextState, _ := settleResources(*state, now)
		*state = nextState
		before = snapshotCoreAssets(state)

		buildingIdx := -1
		for i, b := range state.Buildings {
			if b.ID == buildingID {
				buildingIdx = i
				break
			}
		}
		if buildingIdx == -1 {
			return ErrBuildingNotFound
		}

		building := &state.Buildings[buildingIdx]
		if building.UpgradeEndsAt != nil {
			return ErrAlreadyUpgrading
		}

		config, exists := getBuildingConfig(building.Type)
		if !exists {
			return ErrBuildingNotFound
		}

		currentLevel := building.Level
		upgradeCost, hasCost := config.UpgradeCostByLevel[currentLevel]
		if !hasCost {
			return ErrMaxLevel
		}

		for resType, cost := range upgradeCost {
			if state.Resources.Items[resType] < cost {
				return ErrInsufficientRes
			}
		}
		for resType, cost := range upgradeCost {
			state.Resources.Items[resType] -= cost
		}

		upgradeSeconds := 60
		if seconds, ok := config.UpgradeSecondsByLevel[currentLevel]; ok {
			upgradeSeconds = seconds
		}
		modSources := CollectModifierSources(state)
		upgradeSeconds = applySpeedBonus(upgradeSeconds, "buildSpeedBonus", now, modSources)
		endsAt := now.Add(time.Duration(upgradeSeconds) * time.Second).UTC().Format(resourceDateLayout)
		building.UpgradeEndsAt = &endsAt

		state.ResourceSettledAt = now.UTC().Format(resourceDateLayout)
		state.ServerTime = now.UTC().Format(resourceDateLayout)
		after = snapshotCoreAssets(state)
		return nil
	})
	if err != nil {
		return GameState{}, err
	}
	s.publishCoreAssetDiff(playerID, "building_upgrade_start", buildingID, before, after, now)

	hydrateStateForResponse(&state, now)
	return state, nil
}

func (s *Service) UpgradeBuildingBatch(playerID string) (GameState, int, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return GameState{}, 0, ErrPlayerNotFound
	}

	now := time.Now()
	upgraded := 0
	var before, after coreAssetSnapshot
	state, err := s.repo.UpdatePlayerState(playerID, now, func(state *GameState) error {
		nextState, _ := settleResources(*state, now)
		*state = nextState
		before = snapshotCoreAssets(state)

		type candidate struct {
			index int
			level int
		}
		var candidates []candidate
		for i, b := range state.Buildings {
			if b.UpgradeEndsAt != nil {
				continue
			}
			config, exists := getBuildingConfig(b.Type)
			if !exists || config.ResourceType == "" {
				continue
			}
			if _, hasCost := config.UpgradeCostByLevel[b.Level]; !hasCost {
				continue
			}
			candidates = append(candidates, candidate{index: i, level: b.Level})
		}

		for i := 0; i < len(candidates)-1; i++ {
			for j := i + 1; j < len(candidates); j++ {
				if candidates[j].level < candidates[i].level {
					candidates[i], candidates[j] = candidates[j], candidates[i]
				}
			}
		}

		batchModSources := CollectModifierSources(state)
		for _, c := range candidates {
			building := &state.Buildings[c.index]
			config, _ := getBuildingConfig(building.Type)
			upgradeCost := config.UpgradeCostByLevel[building.Level]

			enough := true
			for resType, cost := range upgradeCost {
				if state.Resources.Items[resType] < cost {
					enough = false
					break
				}
			}
			if !enough {
				continue
			}

			for resType, cost := range upgradeCost {
				state.Resources.Items[resType] -= cost
			}

			upgradeSeconds := 60
			if seconds, ok := config.UpgradeSecondsByLevel[building.Level]; ok {
				upgradeSeconds = seconds
			}
			upgradeSeconds = applySpeedBonus(upgradeSeconds, "buildSpeedBonus", now, batchModSources)
			endsAt := now.Add(time.Duration(upgradeSeconds) * time.Second).UTC().Format(resourceDateLayout)
			building.UpgradeEndsAt = &endsAt
			upgraded++
		}

		if upgraded == 0 {
			return ErrInsufficientRes
		}

		state.ResourceSettledAt = now.UTC().Format(resourceDateLayout)
		state.ServerTime = now.UTC().Format(resourceDateLayout)
		after = snapshotCoreAssets(state)
		return nil
	})
	if err != nil {
		return GameState{}, 0, err
	}
	s.publishCoreAssetDiff(playerID, "building_upgrade_batch_start", "", before, after, now)

	hydrateStateForResponse(&state, now)
	return state, upgraded, nil
}

// InstantCompleteBuilding 极速完成建筑升级（消耗城金）
func (s *Service) InstantCompleteBuilding(playerID string, buildingID string) (GameState, error) {
	playerID = strings.TrimSpace(playerID)
	buildingID = strings.TrimSpace(buildingID)
	if playerID == "" {
		return GameState{}, ErrPlayerNotFound
	}
	if buildingID == "" {
		return GameState{}, ErrBuildingNotFound
	}

	now := time.Now()
	cost := 0
	var before, after coreAssetSnapshot
	state, err := s.repo.UpdatePlayerState(playerID, now, func(state *GameState) error {
		nextState, _ := settleResources(*state, now)
		*state = nextState
		before = snapshotCoreAssets(state)

		buildingIdx := -1
		for i, b := range state.Buildings {
			if b.ID == buildingID {
				buildingIdx = i
				break
			}
		}
		if buildingIdx == -1 {
			return ErrBuildingNotFound
		}

		building := &state.Buildings[buildingIdx]
		if building.UpgradeEndsAt == nil {
			return ErrNotUpgrading
		}

		endsAt, err := time.Parse(resourceDateLayout, *building.UpgradeEndsAt)
		if err != nil {
			return ErrNotUpgrading
		}
		remainingSecs := int(endsAt.Sub(now).Seconds())
		if remainingSecs > 0 {
			cost = speedUpCost(remainingSecs)
			if int(state.CityGold) < cost {
				return ErrInsufficientCityGold
			}
			state.CityGold -= FlexInt(cost)
		}

		building.Level++
		building.UpgradeEndsAt = nil
		state.ResourceSettledAt = now.UTC().Format(resourceDateLayout)
		state.ServerTime = now.UTC().Format(resourceDateLayout)

		nextState, _ = settleResources(*state, now)
		*state = nextState
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
			RefType:      LedgerRefInstantBuilding,
			RefID:        buildingID,
		})
		s.publishCurrencyChanged(playerID, "", buildingID, LedgerRefInstantBuilding)
	}
	s.publishCoreAssetDiff(playerID, LedgerRefInstantBuilding, buildingID, before, after, now)

	hydrateStateForResponse(&state, now)
	return state, nil
}
