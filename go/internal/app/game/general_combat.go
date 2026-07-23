// 本文件归口武将系统接入战斗流程的适配逻辑。
package game

import (
	"sort"
	"strconv"
	"strings"

	"hero3/internal/core/combat"
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
			TraitID:         t.TraitID,
			Params:          params,
			TraitType:       t.TraitType,
			OwnerSide:       "",
			OwnerGeneralID:  g.ID,
			Scope:           t.Scope,
			TargetUnitType:  t.TargetUnitType,
			AllowedSides:    append([]string(nil), t.AllowedSides...),
			AllowedScenes:   append([]string(nil), t.AllowedScenes...),
			RequiredOutcome: t.RequiredOutcome,
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

// partitionMainDefenderSourceTraits 拆出只允许作用于所属守军来源的特性，避免误改同阵营援军。
func partitionMainDefenderSourceTraits(active []general.ActiveTrait) ([]general.ActiveTrait, []general.ActiveTrait) {
	shared := make([]general.ActiveTrait, 0, len(active))
	sourceOnly := make([]general.ActiveTrait, 0, len(active))
	for _, trait := range active {
		if strings.EqualFold(strings.TrimSpace(trait.Scope), "reinforcement_self") {
			sourceOnly = append(sourceOnly, trait)
			continue
		}
		shared = append(shared, trait)
	}
	return shared, sourceOnly
}

// applyMainDefenderAfterCombatTraits 在主守军自己的损失切片上执行来源限定的结算特性。
func applyMainDefenderAfterCombatTraits(before map[string]int, losses map[string]int, faction string, traits []general.ActiveTrait, shared *general.AfterCombatResolveContext) map[string]int {
	if len(losses) == 0 || len(traits) == 0 || shared == nil || shared.Result == nil {
		return losses
	}
	unitTypes := make([]string, 0, len(before)+len(losses))
	seen := map[string]bool{}
	for unitType := range before {
		seen[unitType] = true
		unitTypes = append(unitTypes, unitType)
	}
	for unitType := range losses {
		if !seen[unitType] {
			unitTypes = append(unitTypes, unitType)
		}
	}
	sort.Strings(unitTypes)
	result := combat.CombatResult{Winner: shared.Result.Winner}
	for _, unitType := range unitTypes {
		count := before[unitType]
		if count < losses[unitType] {
			count = losses[unitType]
		}
		if count > 0 {
			result.DefenderLosses = append(result.DefenderLosses, combat.UnitLoss{ID: unitType, Count: count, Losses: losses[unitType]})
		}
	}
	ctx := &general.AfterCombatResolveContext{
		Result:                   &result,
		Attacker:                 &combat.Army{},
		Defender:                 &combat.Army{Faction: faction},
		AttackerOwnsTrait:        false,
		DefenderOwnsTrait:        true,
		Scene:                    shared.Scene,
		DisabledTraitSide:        shared.DisabledTraitSide,
		DisabledTraitOutcomeKeys: shared.DisabledTraitOutcomeKeys,
		Triggered:                shared.Triggered,
	}
	general.Dispatch(ctx, withTraitOwnerSide(traits, "defender"))
	shared.DisabledTraitSide = ctx.DisabledTraitSide
	shared.DisabledTraitOutcomeKeys = ctx.DisabledTraitOutcomeKeys
	shared.Triggered = ctx.Triggered
	return combatLossMap(result.DefenderLosses)
}

// applyPreBattleLossesToCombatResult 把战前真实伤亡并入最终战损，压制兵仍只是不参战。
func applyPreBattleLossesToCombatResult(result *combat.CombatResult, contexts ...*general.BeforeBattleContext) {
	if result == nil {
		return
	}
	attackerLosses := map[string]int{}
	defenderLosses := map[string]int{}
	for _, ctx := range contexts {
		if ctx == nil {
			continue
		}
		mergePositiveTroops(attackerLosses, ctx.AttackerPreBattleLosses)
		mergePositiveTroops(defenderLosses, ctx.DefenderPreBattleLosses)
	}
	result.AttackerLosses = mergePreBattleLosses(result.AttackerLosses, attackerLosses)
	result.DefenderLosses = mergePreBattleLosses(result.DefenderLosses, defenderLosses)
	result.AttackerLossRate = unitLossRate(result.AttackerLosses)
	result.DefenderLossRate = unitLossRate(result.DefenderLosses)
}

// mergePositiveTroops 合并正数兵力变化。
func mergePositiveTroops(target map[string]int, values map[string]int) {
	for unitType, amount := range values {
		if amount > 0 {
			target[unitType] += amount
		}
	}
}

// mergePreBattleLosses 把战前伤亡补回结算基数，并保证伤亡不超过基数。
func mergePreBattleLosses(losses []combat.UnitLoss, preBattle map[string]int) []combat.UnitLoss {
	result := append([]combat.UnitLoss(nil), losses...)
	indexes := map[string]int{}
	for index := range result {
		indexes[result[index].ID] = index
	}
	missing := make([]string, 0)
	for unitType, amount := range preBattle {
		if amount <= 0 {
			continue
		}
		if index, ok := indexes[unitType]; ok {
			result[index].Count += amount
			result[index].Losses += amount
			if result[index].Losses > result[index].Count {
				result[index].Losses = result[index].Count
			}
			continue
		}
		missing = append(missing, unitType)
	}
	sort.Strings(missing)
	for _, unitType := range missing {
		amount := preBattle[unitType]
		result = append(result, combat.UnitLoss{ID: unitType, Count: amount, Losses: amount})
	}
	return result
}

// unitLossRate 按结算基数重新计算一方总损失率。
func unitLossRate(losses []combat.UnitLoss) float64 {
	total, lost := 0, 0
	for _, item := range losses {
		total += item.Count
		lost += item.Losses
	}
	if total <= 0 {
		return 0
	}
	return float64(lost) / float64(total)
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
func dispatchPlunderTraits(state *GameState, generalIDs []string, rewards map[string]int, scene string, ownerSide string, combatTraitsDisabled bool) (map[string]int, map[string]general.TraitOutcome) {
	if len(rewards) == 0 {
		return rewards, nil
	}
	ctx := &general.PlunderResolveContext{
		Rewards: rewards,
		Scene:   scene,
	}
	active := filterDisabledCombatTraits(buildActiveTraitsForGeneralIDs(state, generalIDs), combatTraitsDisabled)
	general.Dispatch(ctx, withTraitOwnerSide(active, ownerSide))
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
				TraitID:         trait.TraitID,
				TraitType:       trait.TraitType,
				OwnerSide:       "reinforcement",
				OwnerPlayerID:   record.FromPlayerID,
				OwnerGeneralID:  snapshot.ID,
				Scope:           scope,
				TargetUnitType:  trait.TargetUnitType,
				AllowedSides:    append([]string(nil), trait.AllowedSides...),
				AllowedScenes:   append([]string(nil), trait.AllowedScenes...),
				RequiredOutcome: trait.RequiredOutcome,
				Params:          params,
			})
		}
	}
	return out
}

// reinforcementCombatTraits 只保留明确声明可作用于援军自身的战斗阶段特性。
func reinforcementCombatTraits(active []general.ActiveTrait) []general.ActiveTrait {
	result := make([]general.ActiveTrait, 0, len(active))
	for _, trait := range active {
		allowed := strings.EqualFold(strings.TrimSpace(trait.Scope), "reinforcement_self")
		for _, side := range trait.AllowedSides {
			if strings.EqualFold(strings.TrimSpace(side), "reinforcement") {
				allowed = true
				break
			}
		}
		if allowed {
			result = append(result, trait)
		}
	}
	return result
}

// reinforcementEnemyCombatTraits 把援军对敌特性映射到防守联盟，交给核心统一排序和消费压制预算。
func reinforcementEnemyCombatTraits(active []general.ActiveTrait) []general.ActiveTrait {
	result := make([]general.ActiveTrait, 0, len(active))
	for _, trait := range active {
		scope := strings.ToLower(strings.TrimSpace(trait.Scope))
		if scope != "enemy_army" && scope != "enemy_traits" {
			continue
		}
		if len(trait.AllowedSides) > 0 {
			allowed := false
			for _, side := range trait.AllowedSides {
				if strings.EqualFold(strings.TrimSpace(side), "reinforcement") {
					allowed = true
					break
				}
			}
			if !allowed {
				continue
			}
		}
		trait.OwnerSide = "defender"
		trait.AllowedSides = []string{"defender"}
		result = append(result, trait)
	}
	return result
}

// activeReinforcementEnemyTraits 汇总真实参战援军的对敌特性。
func activeReinforcementEnemyTraits(records []Reinforcement, combatTraitsDisabled bool) []general.ActiveTrait {
	result := []general.ActiveTrait{}
	for _, record := range records {
		normalizeGarrisonRecord(&record)
		if record.Status != ReinforcementStatusStationed || !record.Rules.CanFight {
			continue
		}
		active := filterDisabledCombatTraits(buildActiveTraitsForReinforcement(record), combatTraitsDisabled)
		result = append(result, reinforcementEnemyCombatTraits(active)...)
	}
	return result
}

// reinforcementOutcomeOwnerKey 返回援军特性结果的玩家和将领联合归属键。
func reinforcementOutcomeOwnerKey(playerID string, generalID string) string {
	return strings.TrimSpace(playerID) + "\x00" + strings.TrimSpace(generalID)
}

// splitReinforcementTraitOutcomes 从共享战斗上下文中拆出援军结果并恢复援军角色。
func splitReinforcementTraitOutcomes(outcomes map[string]general.TraitOutcome, records []Reinforcement) (map[string]general.TraitOutcome, map[string]map[string]general.TraitOutcome) {
	mainOutcomes := map[string]general.TraitOutcome{}
	reinforcementOutcomes := map[string]map[string]general.TraitOutcome{}
	owners := map[string]string{}
	for _, record := range records {
		for _, snapshot := range record.Generals {
			owners[reinforcementOutcomeOwnerKey(record.FromPlayerID, snapshot.ID)] = record.ID
		}
	}
	for key, outcome := range outcomes {
		reinforcementID := owners[reinforcementOutcomeOwnerKey(outcome.OwnerPlayerID, outcome.OwnerGeneralID)]
		if reinforcementID == "" {
			mainOutcomes[key] = outcome
			continue
		}
		outcome.OwnerSide = "reinforcement"
		if reinforcementOutcomes[reinforcementID] == nil {
			reinforcementOutcomes[reinforcementID] = map[string]general.TraitOutcome{}
		}
		reinforcementOutcomes[reinforcementID][key] = outcome
	}
	return mainOutcomes, reinforcementOutcomes
}

// applyReinforcementEnemyBeforeBattleTraits 让各援军批次对真实来袭军队执行战前对敌特性。
func applyReinforcementEnemyBeforeBattleTraits(records []Reinforcement, attacker *combat.Army, defender *combat.Army, scene string, combatTraitsDisabled bool) (map[string]map[string]general.TraitOutcome, []*general.BeforeBattleContext) {
	outcomes := map[string]map[string]general.TraitOutcome{}
	contexts := []*general.BeforeBattleContext{}
	for _, record := range records {
		normalizeGarrisonRecord(&record)
		if record.Status != ReinforcementStatusStationed || !record.Rules.CanFight {
			continue
		}
		active := filterDisabledCombatTraits(buildActiveTraitsForReinforcement(record), combatTraitsDisabled)
		active = reinforcementEnemyCombatTraits(active)
		if len(active) == 0 {
			continue
		}
		ctx := &general.BeforeBattleContext{
			Attacker: attacker, Defender: defender, DefenderOwnsTrait: true,
			IsPvP: true, SameFaction: attacker.Faction == defender.Faction, Scene: scene,
		}
		general.Dispatch(ctx, active)
		contexts = append(contexts, ctx)
		_, byReinforcement := splitReinforcementTraitOutcomes(ctx.Triggered, []Reinforcement{record})
		outcomes[record.ID] = byReinforcement[record.ID]
	}
	return outcomes, contexts
}

// applyReinforcementBeforeBattleTraits 只调整当前一批援军自己的战斗单位。
func applyReinforcementBeforeBattleTraits(record Reinforcement, units []combat.Unit, scene string, combatTraitsDisabled ...bool) ([]combat.Unit, map[string]general.TraitOutcome) {
	if len(units) == 0 {
		return units, nil
	}
	attackerArmy := combat.Army{}
	defenderArmy := combat.Army{Faction: record.FromPlayerFaction, Units: append([]combat.Unit(nil), units...)}
	ctx := &general.BeforeBattleContext{
		Attacker:          &attackerArmy,
		Defender:          &defenderArmy,
		AttackerOwnsTrait: false,
		DefenderOwnsTrait: true,
		IsPvP:             true,
		Scene:             scene,
	}
	disabled := len(combatTraitsDisabled) > 0 && combatTraitsDisabled[0]
	active := filterDisabledCombatTraits(buildActiveTraitsForReinforcement(record), disabled)
	general.Dispatch(ctx, reinforcementCombatTraits(active))
	return defenderArmy.Units, ctx.Triggered
}

type reinforcementTraitResolution struct {
	FinalLosses         map[string]map[string]int
	AfterCombatLosses   map[string]map[string]int
	AfterCombatOutcomes map[string]map[string]general.TraitOutcome
	AfterBattleOutcomes map[string]map[string]general.TraitOutcome
	Outcomes            map[string]map[string]general.TraitOutcome
	SuppressedCount     int
}

// resolveReinforcementAfterBattleTraits 分开保存援军结算减损后的阵亡与战斗结束返兵后的最终净损失。
func resolveReinforcementAfterBattleTraits(records []Reinforcement, losses map[string]map[string]int, winner string, scene string, disabledTraitCount int, combatTraitsDisabled ...bool) reinforcementTraitResolution {
	if len(records) == 0 || len(losses) == 0 {
		return reinforcementTraitResolution{FinalLosses: losses, AfterCombatLosses: cloneNestedStringIntMap(losses)}
	}
	initialDisabledTraitCount := disabledTraitCount
	next := cloneNestedStringIntMap(losses)
	afterCombatLosses := map[string]map[string]int{}
	afterCombatOutcomes := map[string]map[string]general.TraitOutcome{}
	afterBattleOutcomes := map[string]map[string]general.TraitOutcome{}
	outcomes := map[string]map[string]general.TraitOutcome{}
	disableCombatTraits := len(combatTraitsDisabled) > 0 && combatTraitsDisabled[0]
	for _, record := range records {
		recordLosses := next[record.ID]
		if len(recordLosses) == 0 {
			continue
		}
		result := combat.CombatResult{Winner: winner}
		for unitType, amount := range record.RemainingTroops {
			if amount <= 0 {
				continue
			}
			result.DefenderLosses = append(result.DefenderLosses, combat.UnitLoss{ID: unitType, Count: amount, Losses: recordLosses[unitType]})
		}
		afterCombatCtx := &general.AfterCombatResolveContext{
			Result:            &result,
			Attacker:          &combat.Army{},
			Defender:          &combat.Army{Faction: record.FromPlayerFaction},
			AttackerOwnsTrait: false,
			DefenderOwnsTrait: true,
			DisabledTraitSide: map[string]int{"reinforcement": disabledTraitCount},
			Scene:             scene,
		}
		active := filterDisabledCombatTraits(buildActiveTraitsForReinforcement(record), disableCombatTraits)
		general.Dispatch(afterCombatCtx, reinforcementCombatTraits(active))
		disabledTraitCount = afterCombatCtx.DisabledTraitSide["reinforcement"]
		recordLosses = combatLossMap(result.DefenderLosses)
		afterCombatLosses[record.ID] = cloneStringIntMap(recordLosses)
		afterCombatOutcomes[record.ID] = mergeGeneralTraitOutcomeMaps(afterCombatOutcomes[record.ID], afterCombatCtx.Triggered)
		outcomes[record.ID] = mergeGeneralTraitOutcomeMaps(outcomes[record.ID], afterCombatCtx.Triggered)
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
			Winner:       winner,
			Scene:        scene,
		}
		general.Dispatch(ctx, active)
		afterBattleOutcomes[record.ID] = mergeGeneralTraitOutcomeMaps(afterBattleOutcomes[record.ID], ctx.Triggered)
		outcomes[record.ID] = mergeGeneralTraitOutcomeMaps(outcomes[record.ID], ctx.Triggered)
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
	return reinforcementTraitResolution{
		FinalLosses: next, AfterCombatLosses: afterCombatLosses,
		AfterCombatOutcomes: afterCombatOutcomes, AfterBattleOutcomes: afterBattleOutcomes, Outcomes: outcomes,
		SuppressedCount: initialDisabledTraitCount - disabledTraitCount,
	}
}

// applyReinforcementAfterBattleTraits 保留原调用契约并返回最终净损失、结果和压制消费数。
func applyReinforcementAfterBattleTraits(records []Reinforcement, losses map[string]map[string]int, winner string, scene string, disabledTraitCount int) (map[string]map[string]int, map[string]map[string]general.TraitOutcome, int) {
	resolution := resolveReinforcementAfterBattleTraits(records, losses, winner, scene, disabledTraitCount)
	return resolution.FinalLosses, resolution.Outcomes, resolution.SuppressedCount
}

// combatLossesFromSources 汇总主守军与各援军在结算减损后的真实阵亡，供经验和情报比例使用。
func combatLossesFromSources(main map[string]int, reinforcements map[string]map[string]int) []combat.UnitLoss {
	totals := cloneStringIntMap(main)
	if totals == nil {
		totals = map[string]int{}
	}
	for _, losses := range reinforcements {
		mergePositiveTroops(totals, losses)
	}
	unitTypes := make([]string, 0, len(totals))
	for unitType, amount := range totals {
		if amount > 0 {
			unitTypes = append(unitTypes, unitType)
		}
	}
	sort.Strings(unitTypes)
	result := make([]combat.UnitLoss, 0, len(unitTypes))
	for _, unitType := range unitTypes {
		amount := totals[unitType]
		result = append(result, combat.UnitLoss{ID: unitType, Count: amount, Losses: amount})
	}
	return result
}

// mergeGeneralTraitOutcomeMaps 合并同一参战批次不同阶段的特性结果。
func mergeGeneralTraitOutcomeMaps(base map[string]general.TraitOutcome, extra map[string]general.TraitOutcome) map[string]general.TraitOutcome {
	if len(extra) == 0 {
		return base
	}
	if base == nil {
		base = map[string]general.TraitOutcome{}
	}
	for key, outcome := range extra {
		storageKey := key
		for suffix := 2; ; suffix++ {
			if _, exists := base[storageKey]; !exists {
				break
			}
			storageKey = key + "::" + strconv.Itoa(suffix)
		}
		base[storageKey] = outcome
	}
	return base
}

// flattenReinforcementTraitOutcomes 汇总全部援军批次结果供主战报展示。
func flattenReinforcementTraitOutcomes(byReinforcement map[string]map[string]general.TraitOutcome) map[string]general.TraitOutcome {
	result := map[string]general.TraitOutcome{}
	keys := make([]string, 0, len(byReinforcement))
	for reinforcementID := range byReinforcement {
		keys = append(keys, reinforcementID)
	}
	sort.Strings(keys)
	for _, reinforcementID := range keys {
		result = mergeGeneralTraitOutcomeMaps(result, byReinforcement[reinforcementID])
	}
	return result
}

// mergeReinforcementTraitOutcomeMaps 按援军批次合并战前、结算和战后结果。
func mergeReinforcementTraitOutcomeMaps(base map[string]map[string]general.TraitOutcome, extra map[string]map[string]general.TraitOutcome) map[string]map[string]general.TraitOutcome {
	if base == nil {
		base = map[string]map[string]general.TraitOutcome{}
	}
	for reinforcementID, outcomes := range extra {
		base[reinforcementID] = mergeGeneralTraitOutcomeMaps(base[reinforcementID], outcomes)
	}
	return base
}

// mergeTraitOutcomes 把武将特性触发结果合并到战报。
func mergeTraitOutcomes(report *BattleReport, outcomes map[string]general.TraitOutcome) {
	if len(outcomes) == 0 {
		return
	}
	if report.TraitOutcomes == nil {
		report.TraitOutcomes = map[string]TraitOutcomeReport{}
	}
	keys := make([]string, 0, len(outcomes))
	for key := range outcomes {
		keys = append(keys, key)
	}
	sort.SliceStable(keys, func(i, j int) bool {
		left := outcomes[keys[i]]
		right := outcomes[keys[j]]
		if leftOwnerRank, rightOwnerRank := traitOutcomeOwnerRank(left.OwnerSide), traitOutcomeOwnerRank(right.OwnerSide); leftOwnerRank != rightOwnerRank {
			return leftOwnerRank < rightOwnerRank
		}
		if leftTypeRank, rightTypeRank := traitOutcomeTypeRank(left.TraitType), traitOutcomeTypeRank(right.TraitType); leftTypeRank != rightTypeRank {
			return leftTypeRank < rightTypeRank
		}
		return keys[i] < keys[j]
	})
	for _, sourceKey := range keys {
		outcome := outcomes[sourceKey]
		reportOutcome := TraitOutcomeReport{
			TraitID:        outcome.TraitID,
			Name:           outcome.Name,
			TraitType:      outcome.TraitType,
			OwnerSide:      outcome.OwnerSide,
			OwnerGeneralID: outcome.OwnerGeneralID,
			OwnerPlayerID:  outcome.OwnerPlayerID,
			Scope:          outcome.Scope,
			Detail:         outcome.Detail,
		}
		traitKey := reportTraitOutcomeStorageKey(report.TraitOutcomes, sourceKey, reportOutcome)
		report.TraitOutcomes[traitKey] = reportOutcome
		alreadyIn := false
		for _, id := range report.TraitTriggered {
			if id == traitKey {
				alreadyIn = true
				break
			}
		}
		if !alreadyIn {
			report.TraitTriggered = append(report.TraitTriggered, traitKey)
		}
	}
}

// traitOutcomeOwnerRank 按核心组合分发时的触发方顺序排列同一批战报结果。
func traitOutcomeOwnerRank(ownerSide string) int {
	switch strings.ToLower(strings.TrimSpace(ownerSide)) {
	case "attacker":
		return 0
	case "defender":
		return 1
	case "reinforcement":
		return 2
	default:
		return 3
	}
}

// traitOutcomeTypeRank 按将领配置的特殊特性、加成特性顺序排列同阶段结果。
func traitOutcomeTypeRank(traitType string) int {
	switch strings.ToLower(strings.TrimSpace(traitType)) {
	case general.TraitTypeSpecial:
		return 0
	case general.TraitTypeBonus:
		return 1
	default:
		return 2
	}
}

// reportTraitOutcomeStorageKey 为同一特性的不同触发者生成兼容旧 map 的唯一键。
func reportTraitOutcomeStorageKey(existing map[string]TraitOutcomeReport, preferred string, outcome TraitOutcomeReport) string {
	preferred = strings.TrimSpace(preferred)
	if preferred == "" {
		preferred = outcome.TraitID
	}
	if current, exists := existing[preferred]; !exists || sameReportTraitOutcomeOwner(current, outcome) {
		return preferred
	}
	base := outcome.TraitID + "::" + outcome.OwnerSide + "::" + outcome.OwnerGeneralID
	key := base
	for index := 2; ; index++ {
		if current, exists := existing[key]; !exists || sameReportTraitOutcomeOwner(current, outcome) {
			return key
		}
		key = base + "::" + strconv.Itoa(index)
	}
}

// sameReportTraitOutcomeOwner 判断两条战报特性结果是否属于同一个触发者。
func sameReportTraitOutcomeOwner(left TraitOutcomeReport, right TraitOutcomeReport) bool {
	return left.TraitID == right.TraitID && left.OwnerSide == right.OwnerSide && left.OwnerGeneralID == right.OwnerGeneralID && left.OwnerPlayerID == right.OwnerPlayerID
}
