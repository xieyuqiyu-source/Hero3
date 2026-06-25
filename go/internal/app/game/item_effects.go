package game

import "strings"

// 本文件归口道具使用后的效果执行。

// applyItemEffects 应用道具效果并返回本次实际产出。
func applyItemEffects(state *GameState, item ItemDefinition, count int) (map[string]int, error) {
	result := map[string]int{}
	if state.Resources.Items == nil {
		state.Resources.Items = map[string]int{}
	}
	for _, effect := range item.Effects {
		switch effect.Type {
		case "general_exp":
			if state.General == nil {
				return nil, ErrGeneralNotFound
			}
			gained := effect.Amount * count
			applyGeneralBattleExp(state.General, gained)
			result["general_exp"] += gained
		case "resources":
			for key, value := range effect.Resources {
				add := value * count
				capacity := state.Resources.Capacity[key]
				current := state.Resources.Items[key]
				next := current + add
				if capacity > 0 && next > capacity {
					next = capacity
				}
				state.Resources.Items[key] = next
				result[key] += next - current
			}
		case "unit_by_faction":
			unitID := strings.TrimSpace(effect.UnitByFaction[state.Player.Faction])
			if unitID == "" {
				return nil, ErrUnitNotFound
			}
			if _, ok := GetUnitConfig(state.Player.Faction, unitID); !ok {
				return nil, ErrUnitNotFound
			}
			amount := effect.Amount * count
			AddArmyUnit(state, unitID, amount)
			result[unitID] += amount
		default:
			return nil, ErrItemNotUsable
		}
	}
	return result, nil
}
