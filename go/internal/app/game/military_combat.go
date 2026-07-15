// 本文件归口兵种接入战斗引擎的场景限制、出兵校验和单位构建逻辑。
package game

import (
	"strings"
	"time"

	"hero3/internal/core/combat"
)

// combatSceneForPVE 根据玩家战斗模式解析 PVE 战斗场景。
func combatSceneForPVE(mode string) string {
	if strings.TrimSpace(mode) == "plunder" {
		return combat.ScenePVEPlunder
	}
	return combat.ScenePVEAttack
}

// activeCombatRuleID 获取当前场景绑定的战斗规则。
func activeCombatRuleID(scene string) string {
	return combat.RuleIDForScene(scene)
}

// validateAndConsumeArmy 校验玩家出兵并扣除派遣兵力。
func validateAndConsumeArmy(state *GameState, units map[string]int) ([]combat.Unit, error) {
	return validateAndConsumeArmyWithModifiers(state, units, CollectModifierSources(state)...)
}

// validateAndConsumeArmyWithModifiers 校验并扣除出兵，使用调用方明确传入的加成来源构建战斗单位。
func validateAndConsumeArmyWithModifiers(state *GameState, units map[string]int, modSources ...ModifierSource) ([]combat.Unit, error) {
	if len(units) == 0 {
		return nil, ErrNoUnitsSelected
	}

	faction := state.Player.Faction
	var combatUnits []combat.Unit
	now := time.Now()

	for unitType, count := range units {
		if count <= 0 {
			continue
		}

		unitCfg, exists := GetUnitConfig(faction, unitType)
		if !exists {
			return nil, ErrUnitNotFound
		}
		if isNonCombatUnit(unitCfg) {
			return nil, ErrNonCombatUnit
		}

		found := false
		for i, armyUnit := range state.Army {
			if armyUnit.UnitType == unitType {
				if armyUnit.Amount < count {
					return nil, ErrInsufficientArmy
				}
				state.Army[i].Amount -= count
				found = true
				break
			}
		}
		if !found {
			return nil, ErrInsufficientArmy
		}

		combatUnits = append(combatUnits, buildCombatUnitFromConfig(unitType, count, unitCfg, now, modSources...))
	}

	if len(combatUnits) == 0 {
		return nil, ErrNoUnitsSelected
	}

	cleanArmy := state.Army[:0]
	for _, u := range state.Army {
		if u.Amount > 0 {
			cleanArmy = append(cleanArmy, u)
		}
	}
	state.Army = cleanArmy

	return combatUnits, nil
}

// validateArmyAvailability 校验模拟战斗的出兵数量不超过玩家各兵种真实库存。
func validateArmyAvailability(state *GameState, units map[string]int) error {
	if len(units) == 0 {
		return ErrNoUnitsSelected
	}
	available := armySliceToMap(state.Army)
	selected := false
	for unitType, count := range units {
		if count <= 0 {
			continue
		}
		selected = true
		if available[unitType] < count {
			return ErrInsufficientArmy
		}
	}
	if !selected {
		return ErrNoUnitsSelected
	}
	return nil
}

// isNonCombatUnit 判断兵种是否禁止进入战斗。
func isNonCombatUnit(unitCfg UnitConfig) bool {
	if unitCfg.Role == "transport" {
		return true
	}
	return unitCfg.Stats["upkeep"] <= 0
}

// buildSimulatedCombatUnits 构建模拟战斗单位，不改动玩家兵力。
func buildSimulatedCombatUnits(faction string, units map[string]int, now time.Time, modSources ...ModifierSource) ([]combat.Unit, error) {
	if len(units) == 0 {
		return nil, ErrNoUnitsSelected
	}

	var combatUnits []combat.Unit
	for unitType, count := range units {
		if count <= 0 {
			continue
		}

		unitCfg, exists := GetUnitConfig(faction, unitType)
		if !exists {
			return nil, ErrUnitNotFound
		}
		if isNonCombatUnit(unitCfg) {
			return nil, ErrNonCombatUnit
		}

		combatUnits = append(combatUnits, buildCombatUnitFromConfig(unitType, count, unitCfg, now, modSources...))
	}

	if len(combatUnits) == 0 {
		return nil, ErrNoUnitsSelected
	}
	return combatUnits, nil
}

// buildCombatUnitFromConfig 把游戏侧兵种配置转换为核心战斗单位。
func buildCombatUnitFromConfig(unitType string, count int, unitCfg UnitConfig, now time.Time, modSources ...ModifierSource) combat.Unit {
	baseAttack := unitCfg.Stats["attack"]
	baseInfDef := unitCfg.Stats["infantryDefense"]
	baseCavDef := unitCfg.Stats["cavalryDefense"]
	infDefense := ComputeAttributeAt(float64(baseInfDef), StatDefenseBonus, now, modSources...)
	infDefense = ComputeAttributeAt(infDefense, StatInfantryDefenseBonus, now, modSources...)
	cavDefense := ComputeAttributeAt(float64(baseCavDef), StatDefenseBonus, now, modSources...)
	cavDefense = ComputeAttributeAt(cavDefense, StatCavalryDefenseBonus, now, modSources...)

	return combat.Unit{
		ID:              unitType,
		Category:        unitCfg.Category,
		Count:           count,
		Attack:          ComputeIntAttributeAt(baseAttack, StatAttackBonus, now, modSources...),
		InfantryDefense: int(infDefense),
		CavalryDefense:  int(cavDefense),
		CarryCapacity:   unitCfg.Stats["carryCapacity"],
		Upkeep:          unitCfg.Stats["upkeep"],
	}
}
