package game

import "time"

type coreAssetSnapshot struct {
	Resources map[string]int
	Buildings map[string]int
	Army      map[string]int
}

func snapshotCoreAssets(state *GameState) coreAssetSnapshot {
	if state == nil {
		return coreAssetSnapshot{}
	}
	buildings := make(map[string]int, len(state.Buildings))
	for _, building := range state.Buildings {
		if building.ID == "" {
			continue
		}
		buildings[building.ID] = building.Level
	}
	army := make(map[string]int, len(state.Army))
	for _, unit := range state.Army {
		if unit.UnitType == "" {
			continue
		}
		army[unit.UnitType] = unit.Amount
	}
	return coreAssetSnapshot{
		Resources: copyResourceMap(state.Resources.Items),
		Buildings: buildings,
		Army:      army,
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
	if changed := diffIntMaps(before.Buildings, after.Buildings); len(changed) > 0 {
		s.publishEvent(GameEvent{
			Type:     EventBuildingUpgraded,
			PlayerID: playerID,
			RefType:  refType,
			RefID:    refID,
			Payload: map[string]any{
				"changes": changed,
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
