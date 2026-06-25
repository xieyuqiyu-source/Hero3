package game

import "hero3/internal/core/general"

// 本文件归口武将系统接入战斗流程的适配逻辑。

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
