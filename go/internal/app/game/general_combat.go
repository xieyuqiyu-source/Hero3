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
			TraitID: t.TraitID,
			Params:  params,
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
			out = append(out, buildActiveTraits(&generalCopy)...)
		}
	}
	return out
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
			TraitID: outcome.TraitID,
			Name:    outcome.Name,
			Detail:  outcome.Detail,
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
