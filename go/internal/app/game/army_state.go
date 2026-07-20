// 本文件归口玩家兵力数组的通用变更和转换 helper。
package game

import "sort"

// AddArmyUnit 给玩家增加指定兵种兵力。
func AddArmyUnit(state *GameState, unitType string, amount int) {
	if state == nil || unitType == "" || amount <= 0 {
		return
	}
	addToArmy(&state.Army, unitType, amount)
}

// mergeIntoArmy 把指定数量的某兵种合并进玩家军队。
func mergeIntoArmy(state *GameState, unitType string, count int) {
	if count <= 0 {
		return
	}
	// TODO: 玩家军队容量系统上线后，俘虏入军队也要走统一容量检查，避免绕过总兵力上限。
	AddArmyUnit(state, unitType, count)
}

// splitCapturedUnitsByOwnerFaction 按玩家阵营拆分俘虏兵：本阵营进军队，非本阵营进驻防。
func splitCapturedUnitsByOwnerFaction(ownerFaction string, captured map[string]int) (map[string]int, map[string]int) {
	toArmy := map[string]int{}
	toGarrison := map[string]int{}
	for unitType, amount := range captured {
		if amount <= 0 || unitType == "" {
			continue
		}
		if _, ok := GetUnitConfig(ownerFaction, unitType); ok {
			toArmy[unitType] += amount
			continue
		}
		toGarrison[unitType] += amount
	}
	return toArmy, toGarrison
}

// mergeTroopMaps 合并多组兵力 map，并过滤空兵种和非正数。
func mergeTroopMaps(groups ...map[string]int) map[string]int {
	merged := map[string]int{}
	for _, group := range groups {
		for unitType, amount := range group {
			if amount <= 0 || unitType == "" {
				continue
			}
			merged[unitType] += amount
		}
	}
	return merged
}

// addToArmy 给兵力切片增加指定兵种数量。
func addToArmy(army *[]ArmyUnit, unitType string, amount int) {
	if army == nil || unitType == "" || amount <= 0 {
		return
	}
	for i := range *army {
		if (*army)[i].UnitType == unitType {
			(*army)[i].Amount += amount
			return
		}
	}
	*army = append(*army, ArmyUnit{UnitType: unitType, Amount: amount})
}

// armySliceToMap 把兵力切片转换为 map[unitType]amount。
func armySliceToMap(army []ArmyUnit) map[string]int {
	m := make(map[string]int, len(army))
	for _, u := range army {
		m[u.UnitType] = u.Amount
	}
	return m
}

// armyMapToSlice 把 map[unitType]amount 转回兵力切片。
func armyMapToSlice(m map[string]int) []ArmyUnit {
	out := make([]ArmyUnit, 0, len(m))
	for unitType, amount := range m {
		if amount > 0 {
			out = append(out, ArmyUnit{UnitType: unitType, Amount: amount})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].UnitType < out[j].UnitType
	})
	return out
}
