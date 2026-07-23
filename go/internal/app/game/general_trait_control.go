// 本文件负责在战斗开始前统一判定诸葛亮的触发型特性封禁规则。
package game

import (
	"strings"

	"hero3/internal/core/general"
)

const wolongTraitID = "wolong_mouzhi"

type battleTraitControlResolution struct {
	DisabledSides         map[string]bool
	MainOutcomes          map[string]general.TraitOutcome
	ReinforcementOutcomes map[string]map[string]general.TraitOutcome
}

// resolveBattleTraitControl 在所有战斗触发型特性执行前判定卧龙奇谋。
func resolveBattleTraitControl(attackerTraits []general.ActiveTrait, defenderTraits []general.ActiveTrait, reinforcements []Reinforcement, scene string) battleTraitControlResolution {
	attackerControls := wolongControlTraits(attackerTraits, "attacker")
	defenderCoalitionTraits := append([]general.ActiveTrait(nil), defenderTraits...)
	for _, record := range activeBattleReinforcements(reinforcements) {
		defenderCoalitionTraits = append(defenderCoalitionTraits, buildActiveTraitsForReinforcement(record)...)
	}
	defenderControls := wolongControlTraits(defenderCoalitionTraits, "defender")
	allControls := append(append([]general.ActiveTrait(nil), attackerControls...), defenderControls...)
	if len(allControls) == 0 {
		return battleTraitControlResolution{}
	}

	if len(attackerControls) > 0 && len(defenderControls) > 0 {
		outcomes := map[string]general.TraitOutcome{}
		for _, trait := range allControls {
			outcomes = mergeGeneralTraitOutcomeMaps(outcomes, map[string]general.TraitOutcome{
				wolongTraitID: invalidWolongOutcome(trait),
			})
		}
		main, byReinforcement := splitReinforcementTraitOutcomes(outcomes, reinforcements)
		return battleTraitControlResolution{MainOutcomes: main, ReinforcementOutcomes: byReinforcement}
	}

	attackerGeneralCount, attackerTraitCount := combatTriggerStats(attackerTraits)
	defenderGeneralCount, defenderTraitCount := combatTriggerStats(defenderCoalitionTraits)
	ctx := &general.BattleTraitControlContext{
		Scene: scene,
		CombatGeneralCounts: map[string]int{
			"attacker": attackerGeneralCount,
			"defender": defenderGeneralCount,
		},
		CombatTraitCounts: map[string]int{
			"attacker": attackerTraitCount,
			"defender": defenderTraitCount,
		},
	}
	general.Dispatch(ctx, allControls)
	main, byReinforcement := splitReinforcementTraitOutcomes(ctx.Triggered, reinforcements)
	return battleTraitControlResolution{
		DisabledSides:         ctx.DisabledCombatTraitSides,
		MainOutcomes:          main,
		ReinforcementOutcomes: byReinforcement,
	}
}

// activeBattleReinforcements 只返回本场真实参与守城的援军。
func activeBattleReinforcements(records []Reinforcement) []Reinforcement {
	result := make([]Reinforcement, 0, len(records))
	for _, record := range records {
		normalizeGarrisonRecord(&record)
		if record.Status == ReinforcementStatusStationed && record.Rules.CanFight {
			result = append(result, record)
		}
	}
	return result
}

// wolongControlTraits 提取卧龙奇谋并映射到攻方或防守联盟。
func wolongControlTraits(active []general.ActiveTrait, side string) []general.ActiveTrait {
	result := []general.ActiveTrait{}
	for _, trait := range active {
		if trait.TraitID != wolongTraitID {
			continue
		}
		trait.OwnerSide = side
		trait.AllowedSides = []string{side}
		result = append(result, trait)
	}
	return result
}

// invalidWolongOutcome 生成双方诸葛亮相遇时的明确失效提示。
func invalidWolongOutcome(trait general.ActiveTrait) general.TraitOutcome {
	return general.TraitOutcome{
		TraitID:        wolongTraitID,
		Name:           "卧龙奇谋",
		TraitType:      trait.TraitType,
		OwnerSide:      trait.OwnerSide,
		OwnerGeneralID: trait.OwnerGeneralID,
		OwnerPlayerID:  trait.OwnerPlayerID,
		Scope:          trait.Scope,
		Detail: map[string]interface{}{
			"status":        "特性已失效",
			"invalidReason": "双方均有诸葛亮",
		},
	}
}

// combatTriggerStats 统计会被卧龙奇谋封禁的战斗触发特性和对应将领。
func combatTriggerStats(active []general.ActiveTrait) (int, int) {
	generals := map[string]bool{}
	traitCount := 0
	for _, trait := range active {
		if !isCombatTriggeredTrait(trait) {
			continue
		}
		traitCount++
		ownerKey := strings.TrimSpace(trait.OwnerPlayerID) + "\x00" + strings.TrimSpace(trait.OwnerGeneralID)
		if ownerKey != "\x00" {
			generals[ownerKey] = true
		}
	}
	return len(generals), traitCount
}

// filterDisabledCombatTraits 清除被封禁方的战斗触发特性，永久被动和非战斗事件保留。
func filterDisabledCombatTraits(active []general.ActiveTrait, disabled bool) []general.ActiveTrait {
	if !disabled || len(active) == 0 {
		return active
	}
	result := make([]general.ActiveTrait, 0, len(active))
	for _, trait := range active {
		if !isCombatTriggeredTrait(trait) {
			result = append(result, trait)
		}
	}
	return result
}

// isCombatTriggeredTrait 判断特性是否属于本场战斗可封禁的触发阶段。
func isCombatTriggeredTrait(active general.ActiveTrait) bool {
	trait, ok := general.Get(active.TraitID)
	if !ok {
		return false
	}
	for _, subscription := range trait.Subscribe() {
		switch subscription.Event {
		case general.EventBeforeBattle, general.EventAfterCombatResolve, general.EventAfterBattle, general.EventPlunderResolve:
			return true
		}
	}
	return false
}
