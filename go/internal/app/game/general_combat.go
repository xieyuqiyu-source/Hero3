// 本文件归口武将系统接入战斗流程的适配逻辑。
package game

import (
	"strings"

	"hero3/internal/core/general"
)

// buildActiveTraits 从玩家武将构建当前激活的核心特性列表。
func buildActiveTraits(g *General) []general.ActiveTrait {
	if g == nil || len(g.Traits) == 0 {
		return nil
	}
	out := make([]general.ActiveTrait, 0, len(g.Traits))
	for _, t := range g.Traits {
		params := general.Params(t.Params)
		out = append(out, general.ActiveTrait{
			TraitID:        t.TraitID,
			Params:         params,
			TraitType:      t.TraitType,
			OwnerSide:      "",
			OwnerGeneralID: g.ID,
			Scope:          t.Scope,
			TargetUnitType: t.TargetUnitType,
		})
	}
	return out
}

// normalizeBattleGeneralIDs 校验本次真实参战的武将列表，空列表表示不带武将。
func normalizeBattleGeneralIDs(state *GameState, generalIDs []string) ([]string, error) {
	if state == nil || len(generalIDs) == 0 {
		return nil, nil
	}
	result := []string{}
	seen := map[string]bool{}
	for _, generalID := range generalIDs {
		generalID = strings.TrimSpace(generalID)
		if generalID == "" || seen[generalID] {
			continue
		}
		seen[generalID] = true
		if _, ok := findOwnedGeneral(state.Generals, generalID); !ok {
			return nil, ErrGeneralNotFound
		}
		if !generalAvailableForReinforcement(state.GeneralAssignments, generalID) {
			return nil, ErrGeneralBusy
		}
		result = append(result, generalID)
		if len(result) > 1 {
			return nil, ErrInvalidGeneral
		}
	}
	return result, nil
}

// modifierSourcesForBattleGenerals 构建一次真实战斗的加成来源：基础加成保留，武将加成只来自携带武将。
func modifierSourcesForBattleGenerals(state *GameState, generalIDs []string) []ModifierSource {
	if state == nil {
		return nil
	}
	base := *state
	base.General = nil
	sources := CollectModifierSources(&base)
	seen := map[string]bool{}
	for _, generalID := range generalIDs {
		generalID = strings.TrimSpace(generalID)
		if generalID == "" || seen[generalID] {
			continue
		}
		seen[generalID] = true
		if general, ok := findOwnedGeneral(state.Generals, generalID); ok {
			generalCopy := cloneGeneral(general)
			applyHeroConfigToGeneral(&generalCopy)
			sources = append(sources, &GeneralModifierSource{General: &generalCopy})
		}
	}
	return sources
}

// buildActiveTraitsForGeneralIDs 从本次携带武将构建特性列表，不带武将时不触发武将特性。
func buildActiveTraitsForGeneralIDs(state *GameState, generalIDs []string) []general.ActiveTrait {
	if state == nil || len(generalIDs) == 0 {
		return nil
	}
	out := []general.ActiveTrait{}
	seen := map[string]bool{}
	for _, generalID := range generalIDs {
		generalID = strings.TrimSpace(generalID)
		if generalID == "" || seen[generalID] {
			continue
		}
		seen[generalID] = true
		if owned, ok := findOwnedGeneral(state.Generals, generalID); ok {
			generalCopy := cloneGeneral(owned)
			applyHeroConfigToGeneral(&generalCopy)
			for _, trait := range buildActiveTraits(&generalCopy) {
				trait.OwnerPlayerID = state.Player.ID
				out = append(out, trait)
			}
		}
	}
	return out
}

// withTraitOwnerSide 给当前玩法批次的特性补充触发方阵营。
func withTraitOwnerSide(active []general.ActiveTrait, side string) []general.ActiveTrait {
	if side == "" || len(active) == 0 {
		return active
	}
	out := append([]general.ActiveTrait(nil), active...)
	for i := range out {
		out[i].OwnerSide = side
	}
	return out
}

// dispatchMarchCreateTraits 通过核心事件管线应用行军创建类特性。
func dispatchMarchCreateTraits(baseSeconds int, scene string, state *GameState, generalIDs []string) int {
	if baseSeconds <= 0 {
		return baseSeconds
	}
	ctx := &general.MarchCreateContext{
		BaseSeconds:  baseSeconds,
		FinalSeconds: baseSeconds,
		Scene:        scene,
	}
	general.Dispatch(ctx, buildActiveTraitsForGeneralIDs(state, generalIDs))
	if ctx.FinalSeconds <= 0 {
		return 1
	}
	return ctx.FinalSeconds
}

// dispatchRecruitCostTraits 通过核心事件管线应用征兵消耗类特性。
func dispatchRecruitCostTraits(state *GameState, unitConfig UnitConfig, unitID string, amount int, cost ResourceMap) ResourceMap {
	if state == nil || len(cost) == 0 {
		return cost
	}
	mainGeneralIDs := pvpDefenseGeneralIDs(state)
	ctx := &general.RecruitCostContext{
		UnitType: unitID,
		Category: unitConfig.Category,
		Amount:   amount,
		Cost:     map[string]int(cost),
	}
	general.Dispatch(ctx, buildActiveTraitsForGeneralIDs(state, mainGeneralIDs))
	return ResourceMap(ctx.Cost)
}

// dispatchPlunderTraits 通过核心事件管线应用掠夺收益类特性。
func dispatchPlunderTraits(state *GameState, generalIDs []string, rewards map[string]int, scene string, ownerSide string) (map[string]int, map[string]general.TraitOutcome) {
	if len(rewards) == 0 {
		return rewards, nil
	}
	ctx := &general.PlunderResolveContext{
		Rewards: rewards,
		Scene:   scene,
	}
	general.Dispatch(ctx, withTraitOwnerSide(buildActiveTraitsForGeneralIDs(state, generalIDs), ownerSide))
	return ctx.Rewards, ctx.Triggered
}

// buildActiveTraitsForReinforcement 从增援快照构建核心特性列表。
func buildActiveTraitsForReinforcement(record Reinforcement) []general.ActiveTrait {
	out := []general.ActiveTrait{}
	for _, snapshot := range record.Generals {
		for _, trait := range snapshot.Traits {
			params := general.Params(trait.Params)
			scope := trait.Scope
			if strings.TrimSpace(scope) == "" {
				scope = "reinforcement_self"
			}
			out = append(out, general.ActiveTrait{
				TraitID:        trait.TraitID,
				TraitType:      trait.TraitType,
				OwnerSide:      "reinforcement",
				OwnerPlayerID:  record.FromPlayerID,
				OwnerGeneralID: snapshot.ID,
				Scope:          scope,
				TargetUnitType: trait.TargetUnitType,
				Params:         params,
			})
		}
	}
	return out
}

// applyReinforcementAfterBattleTraits 让增援武将特性只修正自己的援军损失。
func applyReinforcementAfterBattleTraits(records []Reinforcement, losses map[string]map[string]int, winner string) map[string]map[string]int {
	if len(records) == 0 || len(losses) == 0 {
		return losses
	}
	next := cloneNestedStringIntMap(losses)
	for _, record := range records {
		recordLosses := next[record.ID]
		if len(recordLosses) == 0 {
			continue
		}
		armyAfterLoss := cloneStringIntMap(record.RemainingTroops)
		for unitType, lost := range recordLosses {
			armyAfterLoss[unitType] -= lost
			if armyAfterLoss[unitType] < 0 {
				armyAfterLoss[unitType] = 0
			}
		}
		ctx := &general.AfterBattleContext{
			PlayerArmy:   armyAfterLoss,
			PlayerLosses: cloneStringIntMap(recordLosses),
			IsAttacker:   false,
			Won:          winner == "defender",
			Scene:        "reinforcement_defense",
		}
		general.Dispatch(ctx, buildActiveTraitsForReinforcement(record))
		for unitType, revived := range ctx.Revived {
			if revived <= 0 {
				continue
			}
			recordLosses[unitType] -= revived
			if recordLosses[unitType] < 0 {
				recordLosses[unitType] = 0
			}
		}
		next[record.ID] = recordLosses
	}
	return next
}

// mergeTraitOutcomes 把武将特性触发结果合并到战报。
func mergeTraitOutcomes(report *BattleReport, outcomes map[string]general.TraitOutcome) {
	if len(outcomes) == 0 {
		return
	}
	if report.TraitOutcomes == nil {
		report.TraitOutcomes = map[string]TraitOutcomeReport{}
	}
	for traitID, outcome := range outcomes {
		report.TraitOutcomes[traitID] = TraitOutcomeReport{
			TraitID:        outcome.TraitID,
			Name:           outcome.Name,
			TraitType:      outcome.TraitType,
			OwnerSide:      outcome.OwnerSide,
			OwnerGeneralID: outcome.OwnerGeneralID,
			OwnerPlayerID:  outcome.OwnerPlayerID,
			Scope:          outcome.Scope,
			Detail:         outcome.Detail,
		}
		alreadyIn := false
		for _, id := range report.TraitTriggered {
			if id == traitID {
				alreadyIn = true
				break
			}
		}
		if !alreadyIn {
			report.TraitTriggered = append(report.TraitTriggered, traitID)
		}
	}
}
