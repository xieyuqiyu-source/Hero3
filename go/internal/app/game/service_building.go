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

	current, err := s.repo.GetState(playerID)
	if err != nil {
		return GameState{}, err
	}
	if building := findBuildingByID(&current, buildingID); building != nil {
		if config, exists := getBuildingConfig(building.Type); exists && len(config.GoldUpgradeCostByLevel) > 0 {
			return s.upgradeBuildingWithGold(playerID, buildingID)
		}
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
		if !buildingCanStartUpgrade(*building) {
			return ErrBuildingStatusBlocked
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

		if err := spendResources(state, upgradeCost); err != nil {
			return err
		}

		upgradeSeconds := 60
		if seconds, ok := config.UpgradeSecondsByLevel[currentLevel]; ok {
			upgradeSeconds = seconds
		}
		modSources := CollectModifierSources(state)
		upgradeSeconds = applySpeedBonus(upgradeSeconds, "buildSpeedBonus", now, modSources)
		endsAt := now.Add(time.Duration(upgradeSeconds) * time.Second).UTC().Format(resourceDateLayout)
		building.UpgradeEndsAt = &endsAt
		building.Status = BuildingStatusUpgrading

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

// upgradeBuildingWithGold 使用账号金币开始建筑升级，当前用于建造司。
func (s *Service) upgradeBuildingWithGold(playerID string, buildingID string) (GameState, error) {
	accountID, err := s.repo.GetAccountIDByPlayerID(playerID)
	if err != nil {
		return GameState{}, err
	}

	now := time.Now()
	cost := 0
	var before, after coreAssetSnapshot
	account, state, err := s.repo.UpdateAccountPlayerState(accountID, playerID, now, func(account *Account, state *GameState) error {
		nextState, _ := settleResources(*state, now)
		*state = nextState
		before = snapshotCoreAssets(state)

		building := findBuildingByID(state, buildingID)
		if building == nil {
			return ErrBuildingNotFound
		}
		if building.UpgradeEndsAt != nil {
			return ErrAlreadyUpgrading
		}
		if !buildingCanStartUpgrade(*building) {
			return ErrBuildingStatusBlocked
		}

		config, exists := getBuildingConfig(building.Type)
		if !exists {
			return ErrBuildingNotFound
		}
		nextCost, hasCost := config.GoldUpgradeCostByLevel[building.Level]
		if !hasCost {
			return ErrMaxLevel
		}
		if account.Gold < nextCost {
			return ErrInsufficientGold
		}
		account.Gold -= nextCost
		cost = nextCost

		upgradeSeconds := 60
		if seconds, ok := config.UpgradeSecondsByLevel[building.Level]; ok {
			upgradeSeconds = seconds
		}
		modSources := CollectModifierSources(state)
		upgradeSeconds = applySpeedBonus(upgradeSeconds, "buildSpeedBonus", now, modSources)
		endsAt := now.Add(time.Duration(upgradeSeconds) * time.Second).UTC().Format(resourceDateLayout)
		building.UpgradeEndsAt = &endsAt
		building.Status = BuildingStatusUpgrading

		state.ResourceSettledAt = now.UTC().Format(resourceDateLayout)
		state.ServerTime = now.UTC().Format(resourceDateLayout)
		after = snapshotCoreAssets(state)
		return nil
	})
	if err != nil {
		return GameState{}, err
	}
	s.recordLedger(GoldLedgerEntry{
		AccountID:    account.ID,
		PlayerID:     playerID,
		Currency:     LedgerCurrencyGold,
		Direction:    LedgerDirectionDebit,
		Amount:       cost,
		BalanceAfter: account.Gold,
		RefType:      LedgerRefBuildingGoldUpgrade,
		RefID:        buildingID,
	})
	s.publishCurrencyChanged(playerID, account.ID, buildingID, LedgerRefBuildingGoldUpgrade)
	s.publishCoreAssetDiff(playerID, LedgerRefBuildingGoldUpgrade, buildingID, before, after, now)

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
			if !buildingCanStartUpgrade(b) {
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

			if !canSpendResources(state, upgradeCost) {
				continue
			}

			if err := spendResources(state, upgradeCost); err != nil {
				continue
			}

			upgradeSeconds := 60
			if seconds, ok := config.UpgradeSecondsByLevel[building.Level]; ok {
				upgradeSeconds = seconds
			}
			upgradeSeconds = applySpeedBonus(upgradeSeconds, "buildSpeedBonus", now, batchModSources)
			endsAt := now.Add(time.Duration(upgradeSeconds) * time.Second).UTC().Format(resourceDateLayout)
			building.UpgradeEndsAt = &endsAt
			building.Status = BuildingStatusUpgrading
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

		if err := applyBuildingMutation(building, BuildingMutation{Type: BuildingMutationCompleteUpgrade}, now); err != nil {
			return err
		}
		ApplyConstructionBureauResourceSlots(state, now)
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
