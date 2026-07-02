// Package traits 提供全将领双特性的配置化注册表。
package traits

import (
	"math"
	"math/rand"
	"strings"

	"hero3/internal/core/combat"
	"hero3/internal/core/general"
)

// configurableTrait 用统一模板承载可配置将领特性，避免玩法代码硬编码具体将领。
type configurableTrait struct {
	id          string
	name        string
	traitType   string
	description string
	effect      string
	events      []string
	schema      []general.ParamField
}

// registerConfigurableTrait 注册一条配置化特性。
func registerConfigurableTrait(t configurableTrait) {
	general.Register(&t)
}

// ID 返回特性唯一 ID。
func (t *configurableTrait) ID() string { return t.id }

// Name 返回特性中文名。
func (t *configurableTrait) Name() string { return t.name }

// Type 返回特性类型。
func (t *configurableTrait) Type() string { return t.traitType }

// Description 返回特性展示文案。
func (t *configurableTrait) Description(params general.Params) string { return t.description }

// ParamSchema 返回 GM 可配置参数 schema。
func (t *configurableTrait) ParamSchema() []general.ParamField {
	out := append([]general.ParamField(nil), t.schema...)
	out = append(out, commonScopeFields()...)
	return out
}

// Subscribe 把模板特性挂接到核心事件管线。
func (t *configurableTrait) Subscribe() []general.EventSubscription {
	out := make([]general.EventSubscription, 0, len(t.events))
	for _, event := range t.events {
		event := event
		out = append(out, general.EventSubscription{
			Event:    event,
			Priority: traitPriority(event, t.effect),
			Handle: func(ctx general.EventContext, params general.Params) {
				t.handle(ctx, params)
			},
		})
	}
	return out
}

// handle 根据事件类型执行模板效果。
func (t *configurableTrait) handle(ctx general.EventContext, params general.Params) {
	if !triggered(params) {
		return
	}
	switch c := ctx.(type) {
	case *general.BeforeBattleContext:
		t.handleBeforeBattle(c, params)
	case *general.AfterCombatResolveContext:
		t.handleAfterCombat(c, params)
	case *general.AfterBattleContext:
		t.handleAfterBattle(c, params)
	case *general.MarchCreateContext:
		t.handleMarchCreate(c, params)
	case *general.RecruitCostContext:
		t.handleRecruitCost(c, params)
	case *general.PlunderResolveContext:
		t.handlePlunder(c, params)
	}
}

// handleBeforeBattle 处理战斗前削弱、震慑和加成。
func (t *configurableTrait) handleBeforeBattle(c *general.BeforeBattleContext, params general.Params) {
	if c == nil {
		return
	}
	switch t.effect {
	case "pre_damage":
		affected := reduceArmyUnits(actorEnemyArmy(c), params, "effectRate", c.Actor.TargetUnitType)
		t.recordBefore(c, params, affected, "preBattleAffected")
	case "suppress":
		affected := reduceArmyUnits(actorEnemyArmy(c), params, "effectRate", c.Actor.TargetUnitType)
		t.recordBefore(c, params, affected, "suppressedUnits")
	case "capture":
		affected := captureArmyUnits(c, params)
		t.recordBefore(c, params, affected, "capturedUnits")
	case "unit_attack_bonus", "unit_defense_bonus", "army_attack_bonus", "army_defense_bonus", "enemy_defense_reduce", "general_attack_flat", "general_defense_flat", "unit_attack_flat":
		changed := applyBattleStatBonus(c, params, t.effect)
		t.recordBefore(c, params, changed, "modifiedUnits")
	}
}

// handleAfterCombat 处理战斗结算后的追加伤害和减损。
func (t *configurableTrait) handleAfterCombat(c *general.AfterCombatResolveContext, params general.Params) {
	if c == nil || c.Result == nil {
		return
	}
	switch t.effect {
	case "extra_damage", "counter_damage":
		changed := lossChange{}
		if c.Actor.Side == "defender" || c.Actor.Side == "reinforcement" {
			changed = addLosses(c.Result.AttackerLosses, params)
			c.Result.AttackerLosses = changed.losses
			recomputeAttackerLossRate(c.Result)
		} else {
			changed = addLosses(c.Result.DefenderLosses, params)
			c.Result.DefenderLosses = changed.losses
			recomputeDefenderLossRate(c.Result)
		}
		t.recordAfterCombat(c, params, changed.byUnit, "extraLosses")
	case "reduce_own_losses":
		changed := reduceLosses(c.Result.AttackerLosses, params)
		if c.Actor.Side == "defender" || c.Actor.Side == "reinforcement" {
			changed = reduceLosses(c.Result.DefenderLosses, params)
			c.Result.DefenderLosses = changed.losses
			recomputeDefenderLossRate(c.Result)
		} else {
			c.Result.AttackerLosses = changed.losses
			recomputeAttackerLossRate(c.Result)
		}
		t.recordAfterCombat(c, params, changed.byUnit, "reducedLosses")
	case "target_unit_damage":
		changed := lossChange{}
		if c.Actor.Side == "defender" || c.Actor.Side == "reinforcement" {
			changed = addTargetLosses(c.Result.AttackerLosses, params, c.Actor.TargetUnitType)
			c.Result.AttackerLosses = changed.losses
			recomputeAttackerLossRate(c.Result)
		} else {
			changed = addTargetLosses(c.Result.DefenderLosses, params, c.Actor.TargetUnitType)
			c.Result.DefenderLosses = changed.losses
			recomputeDefenderLossRate(c.Result)
		}
		t.recordAfterCombat(c, params, changed.byUnit, "targetExtraLosses")
	case "disable_traits":
		count := params.IntWithBounds("disableTraitCount", 1, 0, 24)
		side := enemySide(c.Actor.Side)
		if side != "" && count > 0 {
			if c.DisabledTraitSide == nil {
				c.DisabledTraitSide = map[string]int{}
			}
			c.DisabledTraitSide[side] += count
		}
		t.recordAfterCombat(c, params, map[string]int{"disabledTraitCount": count}, "disabledTraits")
	}
}

// handleAfterBattle 处理战斗结束后的复活、止损和追击记录。
func (t *configurableTrait) handleAfterBattle(c *general.AfterBattleContext, params general.Params) {
	if c == nil {
		return
	}
	switch t.effect {
	case "revive":
		revived := reviveLosses(c, params)
		t.recordAfterBattle(c, params, revived, "revivedUnits")
	case "reduce_final_losses":
		reduced := reduceFinalLosses(c, params)
		t.recordAfterBattle(c, params, reduced, "returnedUnits")
	}
}

// handleMarchCreate 处理行军创建时的速度特性。
func (t *configurableTrait) handleMarchCreate(c *general.MarchCreateContext, params general.Params) {
	if c == nil || c.FinalSeconds <= 0 {
		return
	}
	rate := params.FloatWithBounds("speedBonusRate", params.FloatOr("effectRate", 0), 0, 10)
	if rate <= 0 {
		return
	}
	next := int(math.Ceil(float64(c.FinalSeconds) / (1 + rate)))
	minSeconds := params.IntWithBounds("minMarchSeconds", 1, 1, 86400)
	if next < minSeconds {
		next = minSeconds
	}
	if next >= c.FinalSeconds {
		return
	}
	before := c.FinalSeconds
	c.FinalSeconds = next
	t.recordMarch(c, params, map[string]int{"beforeSeconds": before, "afterSeconds": next}, "marchSeconds")
}

// handleRecruitCost 处理征兵消耗降低。
func (t *configurableTrait) handleRecruitCost(c *general.RecruitCostContext, params general.Params) {
	if c == nil || len(c.Cost) == 0 {
		return
	}
	rate := params.FloatWithBounds("resourceCostReduction", params.FloatOr("effectRate", 0), 0, 0.8)
	if rate <= 0 {
		return
	}
	reduced := map[string]int{}
	for key, value := range c.Cost {
		next := int(math.Ceil(float64(value) * (1 - rate)))
		if next < 1 && value > 0 {
			next = 1
		}
		if next < value {
			reduced[key] = value - next
			c.Cost[key] = next
		}
	}
	t.recordRecruit(c, params, reduced, "costReduced")
}

// handlePlunder 处理掠夺收益修正。
func (t *configurableTrait) handlePlunder(c *general.PlunderResolveContext, params general.Params) {
	if c == nil || len(c.Rewards) == 0 {
		return
	}
	rate := params.FloatWithBounds("plunderBonusRate", params.FloatOr("effectRate", 0), -0.9, 1)
	if rate == 0 {
		return
	}
	changed := map[string]int{}
	for key, value := range c.Rewards {
		next := int(math.Floor(float64(value) * (1 + rate)))
		if next < 0 {
			next = 0
		}
		changed[key] = next - value
		c.Rewards[key] = next
	}
	t.recordPlunder(c, params, changed, "plunderDelta")
}

// triggered 按 GM 配置的触发概率判定是否触发。
func triggered(params general.Params) bool {
	chance := params.FloatWithBounds("triggerChance", 1, 0, 1)
	return chance >= 1 || rand.Float64() <= chance
}

// reduceArmyUnits 按比例让敌军部分兵力不参战或战前损失。
func reduceArmyUnits(army *combat.Army, params general.Params, rateKey string, targetUnitType string) map[string]int {
	if army == nil {
		return nil
	}
	rate := params.FloatWithBounds(rateKey, 0, 0, params.FloatWithBounds("maxAffectedRate", 1, 0, 1))
	maxCount := params.IntWithBounds("maxAffectedCount", 100000000, 0, 100000000)
	targetUnitType = strings.TrimSpace(targetUnitType)
	affected := map[string]int{}
	total := 0
	for i := range army.Units {
		unit := &army.Units[i]
		if unit.Count <= 0 || (targetUnitType != "" && unit.ID != targetUnitType && unit.Category != targetUnitType) {
			continue
		}
		count := int(math.Floor(float64(unit.Count) * rate))
		if maxCount > 0 && total+count > maxCount {
			count = maxCount - total
		}
		if count <= 0 {
			continue
		}
		unit.Count -= count
		affected[unit.ID] += count
		total += count
		if maxCount > 0 && total >= maxCount {
			break
		}
	}
	return affected
}

// captureArmyUnits 处理俘虏类战前效果。
func captureArmyUnits(c *general.BeforeBattleContext, params general.Params) map[string]int {
	rate := params.FloatWithBounds("captureRate", params.FloatOr("effectRate", 0), 0, 1)
	maxPerUnit := params.IntWithBounds("maxCapturePerUnit", params.IntOr("captureMax", 10000), 0, 1000000)
	if rate <= 0 || c.Defender == nil {
		return nil
	}
	if c.CapturedToArmy == nil {
		c.CapturedToArmy = map[string]int{}
	}
	affected := map[string]int{}
	for i := range c.Defender.Units {
		unit := &c.Defender.Units[i]
		count := int(math.Floor(float64(unit.Count) * rate))
		if count > maxPerUnit {
			count = maxPerUnit
		}
		if count <= 0 {
			continue
		}
		unit.Count -= count
		c.CapturedToArmy[unit.ID] += count
		affected[unit.ID] += count
	}
	return affected
}

// applyBattleStatBonus 在战斗前直接调整本次战斗单位属性。
func applyBattleStatBonus(c *general.BeforeBattleContext, params general.Params, effect string) map[string]int {
	army := actorSelfArmy(c)
	if effect == "enemy_defense_reduce" || effect == "enemy_attack_reduce" {
		army = actorEnemyArmy(c)
	}
	if army == nil {
		return nil
	}
	target := strings.TrimSpace(c.Actor.TargetUnitType)
	changed := map[string]int{}
	for i := range army.Units {
		unit := &army.Units[i]
		if target != "" && unit.ID != target && unit.Category != target {
			continue
		}
		before := unit.Attack + unit.InfantryDefense + unit.CavalryDefense
		switch effect {
		case "unit_attack_bonus", "army_attack_bonus":
			unit.Attack += int(math.Round(float64(unit.Attack) * params.FloatWithBounds("attackBonusRate", params.FloatOr("effectRate", 0), 0, 2)))
		case "unit_defense_bonus", "army_defense_bonus":
			bonus := params.FloatWithBounds("defenseBonusRate", params.FloatOr("effectRate", 0), 0, 2)
			unit.InfantryDefense += int(math.Round(float64(unit.InfantryDefense) * bonus))
			unit.CavalryDefense += int(math.Round(float64(unit.CavalryDefense) * bonus))
		case "enemy_defense_reduce":
			reduction := params.FloatWithBounds("enemyDefenseReductionRate", params.FloatOr("effectRate", 0), 0, 0.9)
			unit.InfantryDefense = int(math.Round(float64(unit.InfantryDefense) * (1 - reduction)))
			unit.CavalryDefense = int(math.Round(float64(unit.CavalryDefense) * (1 - reduction)))
		case "enemy_attack_reduce":
			reduction := params.FloatWithBounds("attackReductionRate", params.FloatOr("effectRate", 0), 0, 0.9)
			unit.Attack = int(math.Round(float64(unit.Attack) * (1 - reduction)))
		case "general_attack_flat":
			unit.Attack += params.IntWithBounds("generalAttackFlat", 0, 0, 100000)
		case "general_defense_flat":
			flat := params.IntWithBounds("generalDefenseFlat", 0, 0, 100000)
			unit.InfantryDefense += flat
			unit.CavalryDefense += flat
		case "unit_attack_flat":
			unit.Attack += params.IntWithBounds("unitAttackFlat", 0, 0, 100000)
		}
		after := unit.Attack + unit.InfantryDefense + unit.CavalryDefense
		if after != before {
			changed[unit.ID] = after - before
		}
	}
	return changed
}

// actorSelfArmy 返回当前触发方自己的参战军队。
func actorSelfArmy(c *general.BeforeBattleContext) *combat.Army {
	if c == nil {
		return nil
	}
	if c.Actor.Side == "defender" || c.Actor.Side == "reinforcement" {
		return c.Defender
	}
	return c.Attacker
}

// actorEnemyArmy 返回当前触发方的敌方参战军队。
func actorEnemyArmy(c *general.BeforeBattleContext) *combat.Army {
	if c == nil {
		return nil
	}
	if c.Actor.Side == "defender" || c.Actor.Side == "reinforcement" {
		return c.Attacker
	}
	return c.Defender
}

// enemySide 返回当前触发方在双方战斗里的敌对阵营。
func enemySide(side string) string {
	switch side {
	case "defender", "reinforcement":
		return "attacker"
	case "attacker":
		return "defender"
	default:
		return ""
	}
}

type lossChange struct {
	losses []combat.UnitLoss
	byUnit map[string]int
}

// addLosses 按比例追加敌方损失。
func addLosses(losses []combat.UnitLoss, params general.Params) lossChange {
	next := append([]combat.UnitLoss(nil), losses...)
	rate := params.FloatWithBounds("effectRate", params.FloatOr("damagePercent", 0), 0, 1)
	maxRate := params.FloatWithBounds("maxAffectedRate", 1, 0, 1)
	if rate > maxRate {
		rate = maxRate
	}
	changed := map[string]int{}
	for i := range next {
		extra := int(math.Floor(float64(next[i].Count) * rate))
		if extra <= 0 {
			continue
		}
		before := next[i].Losses
		next[i].Losses += extra
		if next[i].Losses > next[i].Count {
			next[i].Losses = next[i].Count
		}
		changed[next[i].ID] = next[i].Losses - before
	}
	return lossChange{losses: next, byUnit: changed}
}

// addTargetLosses 按目标兵种追加损失。
func addTargetLosses(losses []combat.UnitLoss, params general.Params, target string) lossChange {
	next := append([]combat.UnitLoss(nil), losses...)
	target = strings.TrimSpace(target)
	rate := params.FloatWithBounds("effectRate", 0, 0, 1)
	changed := map[string]int{}
	for i := range next {
		if target != "" && next[i].ID != target {
			continue
		}
		before := next[i].Losses
		next[i].Losses += int(math.Floor(float64(next[i].Count) * rate))
		if next[i].Losses > next[i].Count {
			next[i].Losses = next[i].Count
		}
		changed[next[i].ID] = next[i].Losses - before
	}
	return lossChange{losses: next, byUnit: changed}
}

// reduceLosses 按比例降低损失。
func reduceLosses(losses []combat.UnitLoss, params general.Params) lossChange {
	next := append([]combat.UnitLoss(nil), losses...)
	rate := params.FloatWithBounds("lossReductionRate", params.FloatOr("effectRate", 0), 0, 0.8)
	changed := map[string]int{}
	for i := range next {
		reduced := int(math.Floor(float64(next[i].Losses) * rate))
		if reduced <= 0 {
			continue
		}
		next[i].Losses -= reduced
		changed[next[i].ID] = reduced
	}
	return lossChange{losses: next, byUnit: changed}
}

// reviveLosses 按损失比例复活士兵。
func reviveLosses(c *general.AfterBattleContext, params general.Params) map[string]int {
	rate := params.FloatWithBounds("effectRate", params.FloatOr("reviveRate", 0), 0, 1)
	maxCount := params.IntWithBounds("maxReviveCount", 10000, 0, 1000000)
	revived := map[string]int{}
	total := 0
	if c.Revived == nil {
		c.Revived = map[string]int{}
	}
	for unitType, lost := range c.PlayerLosses {
		count := int(math.Floor(float64(lost) * rate))
		if maxCount > 0 && total+count > maxCount {
			count = maxCount - total
		}
		if count <= 0 {
			continue
		}
		c.PlayerArmy[unitType] += count
		c.Revived[unitType] += count
		revived[unitType] += count
		total += count
	}
	return revived
}

// reduceFinalLosses 在战斗结束后返还一部分损失。
func reduceFinalLosses(c *general.AfterBattleContext, params general.Params) map[string]int {
	return reviveLosses(c, general.Params{"effectRate": params.FloatWithBounds("lossReductionRate", params.FloatOr("effectRate", 0), 0, 0.8), "maxReviveCount": params.FloatOr("maxReturnCount", 10000)})
}

// recomputeDefenderLossRate 重新计算防守方损失率。
func recomputeDefenderLossRate(result *combat.CombatResult) {
	total, lost := sumLosses(result.DefenderLosses)
	if total > 0 {
		result.DefenderLossRate = float64(lost) / float64(total)
	}
}

// recomputeAttackerLossRate 重新计算进攻方损失率。
func recomputeAttackerLossRate(result *combat.CombatResult) {
	total, lost := sumLosses(result.AttackerLosses)
	if total > 0 {
		result.AttackerLossRate = float64(lost) / float64(total)
	}
}

// sumLosses 汇总损失数量。
func sumLosses(losses []combat.UnitLoss) (int, int) {
	total, lost := 0, 0
	for _, item := range losses {
		total += item.Count
		lost += item.Losses
	}
	return total, lost
}

// recordBefore 写入战斗前特性结果。
func (t *configurableTrait) recordBefore(c *general.BeforeBattleContext, params general.Params, values map[string]int, key string) {
	if len(values) == 0 {
		return
	}
	if c.Triggered == nil {
		c.Triggered = map[string]general.TraitOutcome{}
	}
	c.Triggered[t.id] = t.outcome(c.Actor, params, map[string]interface{}{key: values})
}

// recordAfterCombat 写入战斗结算特性结果。
func (t *configurableTrait) recordAfterCombat(c *general.AfterCombatResolveContext, params general.Params, values map[string]int, key string) {
	if len(values) == 0 {
		return
	}
	if c.Triggered == nil {
		c.Triggered = map[string]general.TraitOutcome{}
	}
	c.Triggered[t.id] = t.outcome(c.Actor, params, map[string]interface{}{key: values})
}

// recordAfterBattle 写入战斗结束特性结果。
func (t *configurableTrait) recordAfterBattle(c *general.AfterBattleContext, params general.Params, values map[string]int, key string) {
	if len(values) == 0 {
		return
	}
	if c.Triggered == nil {
		c.Triggered = map[string]general.TraitOutcome{}
	}
	c.Triggered[t.id] = t.outcome(c.Actor, params, map[string]interface{}{key: values})
}

// recordMarch 写入行军特性结果。
func (t *configurableTrait) recordMarch(c *general.MarchCreateContext, params general.Params, values map[string]int, key string) {
	if c.Triggered == nil {
		c.Triggered = map[string]general.TraitOutcome{}
	}
	c.Triggered[t.id] = t.outcome(c.Actor, params, map[string]interface{}{key: values})
}

// recordRecruit 写入征兵特性结果。
func (t *configurableTrait) recordRecruit(c *general.RecruitCostContext, params general.Params, values map[string]int, key string) {
	if len(values) == 0 {
		return
	}
	if c.Triggered == nil {
		c.Triggered = map[string]general.TraitOutcome{}
	}
	c.Triggered[t.id] = t.outcome(c.Actor, params, map[string]interface{}{key: values})
}

// recordPlunder 写入掠夺特性结果。
func (t *configurableTrait) recordPlunder(c *general.PlunderResolveContext, params general.Params, values map[string]int, key string) {
	if len(values) == 0 {
		return
	}
	if c.Triggered == nil {
		c.Triggered = map[string]general.TraitOutcome{}
	}
	c.Triggered[t.id] = t.outcome(c.Actor, params, map[string]interface{}{key: values})
}

// outcome 生成标准战报特性结果。
func (t *configurableTrait) outcome(actor general.TraitActor, params general.Params, detail map[string]interface{}) general.TraitOutcome {
	detail["triggerChance"] = params.FloatOr("triggerChance", 1)
	detail["effectRate"] = params.FloatOr("effectRate", 0)
	return general.TraitOutcome{
		TraitID:        t.id,
		Name:           t.name,
		TraitType:      t.traitType,
		OwnerSide:      actor.Side,
		OwnerGeneralID: actor.GeneralID,
		OwnerPlayerID:  actor.PlayerID,
		Scope:          actor.Scope,
		Detail:         detail,
	}
}

// traitPriority 根据事件和效果返回执行优先级。
func traitPriority(event string, effect string) int {
	if event == general.EventAfterCombatResolve && effect == "disable_traits" {
		return 90
	}
	if event == general.EventBeforeBattle {
		return 70
	}
	return 40
}

// commonScopeFields 返回所有特性共享的安全边界字段。
func commonScopeFields() []general.ParamField {
	return []general.ParamField{
		{Key: "triggerChance", Label: "触发概率", Description: "1 表示必定触发，0.5 表示 50% 概率", Default: 1, Min: 0, Max: 1, Step: 0.01},
		{Key: "maxAffectedRate", Label: "最大影响比例", Description: "限制强效果最多影响的兵力比例", Default: 1, Min: 0, Max: 1, Step: 0.01},
		{Key: "maxAffectedCount", Label: "最大影响数量", Description: "限制强效果最多影响的士兵数量", Default: 1000000, Min: 0, Max: 100000000, Step: 100},
	}
}

// rateField 创建百分比参数字段。
func rateField(key string, label string, def float64, max float64) general.ParamField {
	return general.ParamField{Key: key, Label: label, Description: "GM 可配置百分比，使用 0 到 1 的小数", Default: def, Min: 0, Max: max, Step: 0.01}
}

// countField 创建数量参数字段。
func countField(key string, label string, def float64, max float64) general.ParamField {
	return general.ParamField{Key: key, Label: label, Description: "GM 可配置固定值", Default: def, Min: 0, Max: max, Step: 1}
}

func init() {
	for _, item := range []configurableTrait{
		{id: "weiwu_haoling", name: "魏武号令", traitType: general.TraitTypeSpecial, description: "按时间强化虎卫或特殊兵征兵进度。", effect: "recruit_guard", events: []string{general.EventRecruitComplete}, schema: []general.ParamField{countField("guardPerMinute", "每分钟虎卫", 500, 10000), countField("maxGuardPerDay", "每日上限", 3000, 100000)}},
		{id: "weiwu_tongyu", name: "魏武统御", traitType: general.TraitTypeBonus, description: "虎卫或特殊兵攻防提升。", effect: "unit_attack_bonus", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("attackBonusRate", "攻击加成", 0.1, 2), rateField("defenseBonusRate", "防御加成", 0.1, 2)}},
		{id: "yibing_touxi", name: "疑兵偷袭", traitType: general.TraitTypeSpecial, description: "战前偷袭削弱敌方兵力。", effect: "pre_damage", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("effectRate", "削弱比例", 0.35, 1)}},
		{id: "mouding_houfa", name: "谋定后发", traitType: general.TraitTypeBonus, description: "削弱敌方攻击爆发。", effect: "enemy_attack_reduce", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("effectRate", "攻击削弱", 0.1, 0.9)}},
		{id: "meihuo_raozhen", name: "魅惑扰阵", traitType: general.TraitTypeBonus, description: "降低敌方攻防表现。", effect: "enemy_defense_reduce", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("enemyDefenseReductionRate", "敌方防御降低", 0.1, 0.9)}},
		{id: "huchi_chongzhen", name: "虎痴冲阵", traitType: general.TraitTypeSpecial, description: "概率突破敌方防守类加成。", effect: "enemy_defense_reduce", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("enemyDefenseReductionRate", "突破防御比例", 0.2, 0.9)}},
		{id: "pojun_pofang", name: "破敌防御", traitType: general.TraitTypeBonus, description: "敌方防御降低。", effect: "enemy_defense_reduce", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("enemyDefenseReductionRate", "敌方防御降低", 0.35, 0.9)}},
		{id: "huzhu_sizhan", name: "护主死战", traitType: general.TraitTypeSpecial, description: "濒临失败时降低最终损失。", effect: "reduce_final_losses", events: []string{general.EventAfterBattle}, schema: []general.ParamField{rateField("lossReductionRate", "最终减损", 0.15, 0.8), countField("maxReturnCount", "返还上限", 10000, 1000000)}},
		{id: "sizhandaodi", name: "死战到底", traitType: general.TraitTypeBonus, description: "步兵攻击提升。", effect: "unit_attack_bonus", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("attackBonusRate", "步兵攻击提升", 0.35, 2)}},
		{id: "jixing_benxi", name: "疾行奔袭", traitType: general.TraitTypeSpecial, description: "出征或增援行军时间缩短。", effect: "march_speed", events: []string{general.EventMarchCreate}, schema: []general.ParamField{rateField("speedBonusRate", "速度提升", 0.2, 10), countField("minMarchSeconds", "最短行军秒数", 60, 86400)}},
		{id: "dunzhen_fangyu", name: "盾阵防御", traitType: general.TraitTypeBonus, description: "我军防御提升。", effect: "army_defense_bonus", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("defenseBonusRate", "防御提升", 0.35, 2)}},
		{id: "weizhen_zhenhe", name: "威震震慑", traitType: general.TraitTypeSpecial, description: "以少打多时让敌方部分兵不参战。", effect: "suppress", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("effectRate", "震慑比例", 0.2, 1)}},
		{id: "weizhen_xiaoyao", name: "威震逍遥", traitType: general.TraitTypeBonus, description: "骑兵攻击提升。", effect: "unit_attack_bonus", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("attackBonusRate", "骑兵攻击提升", 0.35, 2)}},
		{id: "shengui_zhicai", name: "神鬼之才", traitType: general.TraitTypeSpecial, description: "征兵资源消耗降低。", effect: "recruit_cost_reduce", events: []string{general.EventRecruitCost}, schema: []general.ParamField{rateField("resourceCostReduction", "征兵消耗降低", 0.5, 0.8)}},
		{id: "guicai_yice", name: "鬼才遗策", traitType: general.TraitTypeBonus, description: "战败时最终损失降低。", effect: "reduce_final_losses", events: []string{general.EventAfterBattle}, schema: []general.ParamField{rateField("lossReductionRate", "最终减损", 0.1, 0.8), countField("maxReturnCount", "返还上限", 10000, 1000000)}},
		{id: "wangzuo_zhicai", name: "王佐之才", traitType: general.TraitTypeSpecial, description: "资源、建筑、征兵效率提升。", effect: "recruit_cost_reduce", events: []string{general.EventRecruitCost}, schema: []general.ParamField{rateField("resourceCostReduction", "征兵返还", 0.05, 0.8)}},
		{id: "neizheng_jingying", name: "内政精营", traitType: general.TraitTypeBonus, description: "资源产量提升。", effect: "resource_bonus", events: []string{general.EventResourceSettle}, schema: []general.ParamField{rateField("productionBonusRate", "资源产量提升", 0.05, 0.5)}},
		{id: "renzhu_shouhu", name: "仁主守护", traitType: general.TraitTypeBonus, description: "己方最终损失降低。", effect: "reduce_final_losses", events: []string{general.EventAfterBattle}, schema: []general.ParamField{rateField("lossReductionRate", "最终减损", 0.1, 0.8), countField("maxReturnCount", "返还上限", 10000, 1000000)}},
		{id: "shuiyan_qijun", name: "水淹七军", traitType: general.TraitTypeSpecial, description: "战前水淹削弱敌军。", effect: "pre_damage", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("effectRate", "水淹比例", 0.35, 1)}},
		{id: "wusheng_pojun", name: "武圣破军", traitType: general.TraitTypeBonus, description: "主力攻击提升。", effect: "army_attack_bonus", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("attackBonusRate", "攻击提升", 0.2, 2)}},
		{id: "zhenhe_quanjun", name: "震慑全军", traitType: general.TraitTypeSpecial, description: "概率让敌军部分兵不参战。", effect: "suppress", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("effectRate", "震慑比例", 0.5, 1)}},
		{id: "wanren_nuhou", name: "万人怒吼", traitType: general.TraitTypeBonus, description: "步兵攻击提升。", effect: "unit_attack_bonus", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("attackBonusRate", "步兵攻击提升", 0.2, 2)}},
		{id: "qimen_dunjia", name: "奇门遁甲", traitType: general.TraitTypeSpecial, description: "困住敌军，降低参战兵力。", effect: "suppress", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("effectRate", "困敌比例", 0.25, 1)}},
		{id: "wolong_mouzhi", name: "卧龙谋制", traitType: general.TraitTypeBonus, description: "降低敌方特性触发率。", effect: "disable_traits", events: []string{general.EventAfterCombatResolve}, schema: []general.ParamField{rateField("disableTraitRate", "压制比例", 0.1, 0.8), countField("disableTraitCount", "压制数量", 1, 24)}},
		{id: "longdan_jiuyuan", name: "龙胆救援", traitType: general.TraitTypeSpecial, description: "增援或防守时保护己方损失。", effect: "reduce_own_losses", events: []string{general.EventAfterCombatResolve}, schema: []general.ParamField{rateField("lossReductionRate", "减损比例", 0.2, 0.8)}},
		{id: "qijin_qichu", name: "七进七出", traitType: general.TraitTypeBonus, description: "全军速度提升。", effect: "march_speed", events: []string{general.EventMarchCreate}, schema: []general.ParamField{rateField("speedBonusRate", "速度提升", 1, 10), countField("minMarchSeconds", "最短行军秒数", 60, 86400)}},
		{id: "xiliang_tuji", name: "西凉突击", traitType: general.TraitTypeSpecial, description: "骑兵开战追加冲锋伤害。", effect: "target_unit_damage", events: []string{general.EventAfterCombatResolve}, schema: []general.ParamField{rateField("effectRate", "冲锋伤害", 0.12, 1)}},
		{id: "tianshen_xiafan", name: "天神下凡", traitType: general.TraitTypeBonus, description: "武将攻击固定提升。", effect: "general_attack_flat", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{countField("generalAttackFlat", "武将攻击", 20, 100000)}},
		{id: "baibu_chuanyang", name: "百步穿杨", traitType: general.TraitTypeSpecial, description: "概率使敌方失去防御加成。", effect: "enemy_defense_reduce", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("enemyDefenseReductionRate", "防御削弱", 0.2, 0.9)}},
		{id: "laodang_yizhuang", name: "老当益壮", traitType: general.TraitTypeBonus, description: "对高口粮兵伤害提升。", effect: "extra_damage", events: []string{general.EventAfterCombatResolve}, schema: []general.ParamField{rateField("effectRate", "额外伤害", 0.1, 1)}},
		{id: "qibing_raohou", name: "奇兵绕后", traitType: general.TraitTypeSpecial, description: "主动进攻时绕过部分防御。", effect: "enemy_defense_reduce", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("enemyDefenseReductionRate", "绕防比例", 0.2, 0.9)}},
		{id: "gushou_hanzhong", name: "固守汉中", traitType: general.TraitTypeBonus, description: "武将防御固定提升。", effect: "general_defense_flat", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{countField("generalDefenseFlat", "武将防御", 20, 100000)}},
		{id: "jiangdong_haoling", name: "江东号令", traitType: general.TraitTypeSpecial, description: "防守失败时降低敌方掠夺收益。", effect: "plunder_reduce", events: []string{general.EventPlunderResolve}, schema: []general.ParamField{{Key: "plunderBonusRate", Label: "掠夺收益修正", Description: "负数降低敌方掠夺收益，正数提高收益", Default: -0.2, Min: -0.9, Max: 1, Step: 0.01}}},
		{id: "jiangdong_gushou", name: "江东固守", traitType: general.TraitTypeBonus, description: "全军防御提升。", effect: "army_defense_bonus", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("defenseBonusRate", "防御提升", 0.5, 2)}},
		{id: "xiaobawang_zhuiji", name: "小霸王追击", traitType: general.TraitTypeSpecial, description: "胜利后追加追击损失。", effect: "extra_damage", events: []string{general.EventAfterCombatResolve}, schema: []general.ParamField{rateField("effectRate", "追击伤害", 0.1, 1)}},
		{id: "xiaobawang_tieqi", name: "小霸王", traitType: general.TraitTypeBonus, description: "霸王骑攻击固定提升。", effect: "unit_attack_flat", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{countField("unitAttackFlat", "兵种攻击", 50, 100000)}},
		{id: "meizhoulang_junlue", name: "美周郎军略", traitType: general.TraitTypeBonus, description: "火攻伤害或全军攻击提升。", effect: "army_attack_bonus", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("attackBonusRate", "攻击提升", 0.05, 2), rateField("fireDamageBonusRate", "火攻强化", 0.1, 1)}},
		{id: "huoshao_lianying", name: "火烧联营", traitType: general.TraitTypeSpecial, description: "概率烧死敌方步兵。", effect: "target_unit_damage", events: []string{general.EventAfterCombatResolve}, schema: []general.ParamField{rateField("effectRate", "步兵伤害", 1, 1)}},
		{id: "lianying_zengshang", name: "连营增伤", traitType: general.TraitTypeBonus, description: "对步兵伤害提升。", effect: "target_unit_damage", events: []string{general.EventAfterCombatResolve}, schema: []general.ParamField{rateField("effectRate", "对步兵伤害", 0.1, 1)}},
		{id: "baiyi_dujiang", name: "白衣渡江", traitType: general.TraitTypeSpecial, description: "概率隐秘行踪并加快行军。", effect: "march_speed", events: []string{general.EventMarchCreate}, schema: []general.ParamField{rateField("speedBonusRate", "速度提升", 0.2, 10), countField("minMarchSeconds", "最短行军秒数", 60, 86400), rateField("warningDelayRate", "预警延迟", 0.3, 1)}},
		{id: "baiyi_jixing", name: "白衣急行", traitType: general.TraitTypeBonus, description: "队伍速度提升。", effect: "march_speed", events: []string{general.EventMarchCreate}, schema: []general.ParamField{rateField("speedBonusRate", "速度提升", 0.2, 10), countField("minMarchSeconds", "最短行军秒数", 60, 86400)}},
		{id: "kuairu_shandian", name: "快如闪电", traitType: general.TraitTypeSpecial, description: "概率触发闪电战，极大缩短行军时间。", effect: "march_speed", events: []string{general.EventMarchCreate}, schema: []general.ParamField{rateField("speedBonusRate", "速度提升", 4, 10), countField("minMarchSeconds", "最短行军秒数", 30, 86400)}},
		{id: "xinyi_yonglie", name: "信义勇烈", traitType: general.TraitTypeBonus, description: "将领或援军攻击提升。", effect: "army_attack_bonus", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("attackBonusRate", "攻击提升", 0.1, 2)}},
		{id: "jinfan_jielue", name: "锦帆劫掠", traitType: general.TraitTypeSpecial, description: "掠夺收益提升。", effect: "plunder_bonus", events: []string{general.EventPlunderResolve}, schema: []general.ParamField{rateField("plunderBonusRate", "掠夺收益提升", 0.2, 1)}},
		{id: "jinfan_qixi", name: "锦帆奇袭", traitType: general.TraitTypeBonus, description: "掠夺战攻击提升。", effect: "army_attack_bonus", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("attackBonusRate", "攻击提升", 0.1, 2)}},
		{id: "kurouji", name: "苦肉计", traitType: general.TraitTypeSpecial, description: "概率压制敌方特性。", effect: "disable_traits", events: []string{general.EventAfterCombatResolve}, schema: []general.ParamField{countField("disableTraitCount", "压制特性数", 1, 24), rateField("disableTraitRate", "压制比例", 0.5, 0.8)}},
		{id: "kurou_fanji", name: "苦肉反击", traitType: general.TraitTypeBonus, description: "承受代价后提高敌方损失。", effect: "counter_damage", events: []string{general.EventAfterCombatResolve}, schema: []general.ParamField{rateField("effectRate", "反击增伤", 0.1, 1), rateField("selfCostRate", "自身代价", 0.03, 0.5)}},
	} {
		registerConfigurableTrait(item)
	}
}
