package game

import "time"

type coreAssetSnapshot struct {
	Resources        map[string]int
	Buildings        map[string]int
	BuildingStatuses map[string]string
	Army             map[string]int
}

func snapshotCoreAssets(state *GameState) coreAssetSnapshot {
	if state == nil {
		return coreAssetSnapshot{}
	}
	buildings := make(map[string]int, len(state.Buildings))
	buildingStatuses := make(map[string]string, len(state.Buildings))
	for _, building := range state.Buildings {
		if building.ID == "" {
			continue
		}
		buildings[building.ID] = building.Level
		buildingStatuses[building.ID] = normalizeBuildingStatus(building.Status)
	}
	army := make(map[string]int, len(state.Army))
	for _, unit := range state.Army {
		if unit.UnitType == "" {
			continue
		}
		army[unit.UnitType] = unit.Amount
	}
	return coreAssetSnapshot{
		Resources:        copyResourceMap(state.Resources.Items),
		Buildings:        buildings,
		BuildingStatuses: buildingStatuses,
		Army:             army,
	}
}

func (s *Service) publishCoreAssetDiff(playerID string, refType string, refID string, before coreAssetSnapshot, after coreAssetSnapshot, now time.Time) {
	if changed := diffIntMaps(before.Resources, after.Resources); len(changed) > 0 {
		s.publishEvent(GameEvent{
			Type:     EventResourceChanged,
			PlayerID: playerID,
			RefType:  refType,
			RefID:    refID,
			Payload: map[string]any{
				"changes": changed,
			},
			CreatedAt: now.UTC().Format(resourceDateLayout),
		})
	}
	buildingLevelChanges := diffIntMaps(before.Buildings, after.Buildings)
	buildingStatusChanges := diffStringMaps(before.BuildingStatuses, after.BuildingStatuses)
	if len(buildingLevelChanges) > 0 || len(buildingStatusChanges) > 0 {
		s.publishEvent(GameEvent{
			Type:     EventBuildingUpgraded,
			PlayerID: playerID,
			RefType:  refType,
			RefID:    refID,
			Payload: map[string]any{
				"changes":       buildingLevelChanges,
				"statusChanges": buildingStatusChanges,
			},
			CreatedAt: now.UTC().Format(resourceDateLayout),
		})
	}
	if changed := diffIntMaps(before.Army, after.Army); len(changed) > 0 {
		s.publishEvent(GameEvent{
			Type:     EventUnitChanged,
			PlayerID: playerID,
			RefType:  refType,
			RefID:    refID,
			Payload: map[string]any{
				"changes": changed,
			},
			CreatedAt: now.UTC().Format(resourceDateLayout),
		})
	}
}

func diffIntMaps(before map[string]int, after map[string]int) map[string]int {
	changes := map[string]int{}
	for key, next := range after {
		if delta := next - before[key]; delta != 0 {
			changes[key] = delta
		}
	}
	for key, prev := range before {
		if _, ok := after[key]; ok {
			continue
		}
		if prev != 0 {
			changes[key] = -prev
		}
	}
	if len(changes) == 0 {
		return nil
	}
	return changes
}

func diffStringMaps(before map[string]string, after map[string]string) map[string]string {
	changes := map[string]string{}
	for key, next := range after {
		if before[key] != next {
			changes[key] = next
		}
	}
	for key := range before {
		if _, ok := after[key]; !ok {
			changes[key] = ""
		}
	}
	if len(changes) == 0 {
		return nil
	}
	return changes
}
