// Package traits 提供全将领双特性的配置化注册表。
package traits

import (
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"

	"hero3/internal/core/combat"
	"hero3/internal/core/general"
)

// configurableTrait 用统一模板承载可配置将领特性，避免玩法代码硬编码具体将领。
type configurableTrait struct {
	id                   string
	name                 string
	traitType            string
	description          string
	effect               string
	events               []string
	schema               []general.ParamField
	defaultTriggerChance float64
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
	if t.effect == "passive_unit_stats" || t.effect == "passive_stat" ||
		t.id == "longdan_jiuyuan" || t.id == "qijin_qichu" || t.id == "gushou_hanzhong" {
		return out
	}
	for _, field := range commonScopeFields(t.triggerChanceDefault()) {
		if (t.id == "yibing_touxi" || t.id == "shuiyan_qijun" || t.id == "wusheng_pojun" ||
			t.id == "zhenhe_quanjun" || t.id == "wanren_nuhou" || t.id == "baibu_chuanyang" ||
			t.id == "guicai_yice" || t.id == "renzhu_shouhu" || t.id == "qimen_dunjia" || t.effect == "disable_all_combat_traits") &&
			(field.Key == "maxAffectedRate" || field.Key == "maxAffectedCount") {
			continue
		}
		out = append(out, field)
	}
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
	if !triggeredWithDefault(params, t.triggerChanceDefault()) {
		return
	}
	switch c := ctx.(type) {
	case *general.BeforeBattleContext:
		t.handleBeforeBattle(c, params)
	case *general.BattleTraitControlContext:
		t.handleBattleTraitControl(c, params)
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

// handleBattleTraitControl 在所有战斗触发阶段前封禁敌方触发型特性。
func (t *configurableTrait) handleBattleTraitControl(c *general.BattleTraitControlContext, params general.Params) {
	if c == nil || t.effect != "disable_all_combat_traits" {
		return
	}
	side := enemySide(c.Actor.Side)
	if side == "" {
		return
	}
	if c.DisabledCombatTraitSides == nil {
		c.DisabledCombatTraitSides = map[string]bool{}
	}
	c.DisabledCombatTraitSides[side] = true
	if c.Triggered == nil {
		c.Triggered = map[string]general.TraitOutcome{}
	}
	storeTraitOutcome(c.Triggered, t.outcome(c.Actor, params, map[string]interface{}{
		"disabledGeneralCount": c.CombatGeneralCounts[side],
		"disabledTraitCount":   c.CombatTraitCounts[side],
	}))
}

// handleBeforeBattle 处理战斗前削弱、震慑和加成。
func (t *configurableTrait) handleBeforeBattle(c *general.BeforeBattleContext, params general.Params) {
	if c == nil {
		return
	}
	switch t.effect {
	case "pre_damage":
		var affected map[string]int
		if t.id == "shuiyan_qijun" {
			// 水淹七军按敌方参战总兵力精确形成真实伤亡，避免逐兵种取整少扣。
			affected = reduceArmyUnitsByTotalRate(actorEnemyArmy(c), params, "effectRate", c.Actor.TargetUnitType)
		} else {
			effectParams := params
			if t.id == "yibing_touxi" {
				// 疑兵真实伤亡只由 effectRate 决定，不接受历史通用上限截断。
				effectParams = general.Params{"effectRate": params.FloatWithBounds("effectRate", 0, 0, 1)}
			}
			affected = reduceArmyUnits(actorEnemyArmy(c), effectParams, "effectRate", c.Actor.TargetUnitType)
		}
		recordPreBattleLosses(c, affected)
		if t.id == "shuiyan_qijun" {
			t.recordBeforeDetail(c, params, map[string]interface{}{
				"preBattleAffected":   affected,
				"realCasualties":      affected,
				"totalRealCasualties": sumPositiveMapValues(affected),
			})
		} else {
			t.recordBefore(c, params, affected, "preBattleAffected")
		}
	case "suppress":
		var affected map[string]int
		if t.id == "weizhen_zhenhe" || t.id == "zhenhe_quanjun" {
			// 张辽和张飞按敌方本次参战总兵力精确计算溃逃人数，不受旧通用上限字段截断。
			affected = reduceArmyUnitsByTotalRate(actorEnemyArmy(c), params, "effectRate", c.Actor.TargetUnitType)
		} else {
			affected = reduceArmyUnits(actorEnemyArmy(c), params, "effectRate", c.Actor.TargetUnitType)
		}
		recordSuppressedUnits(c, affected)
		if (t.id == "weizhen_zhenhe" || t.id == "zhenhe_quanjun") && len(affected) > 0 {
			t.recordBeforeDetail(c, params, map[string]interface{}{
				"suppressedUnits": affected,
				"fledUnits":       affected,
				"returnedUnits":   affected,
			})
		} else {
			t.recordBefore(c, params, affected, "suppressedUnits")
		}
	case "longdan_rescue":
		changed := applyBattleStatBonus(c, params, "unit_defense_bonus")
		t.recordBeforeStatChanges(c, params, changed)
	case "unit_attack_bonus", "unit_defense_bonus", "army_attack_bonus", "army_defense_bonus", "enemy_defense_reduce", "enemy_attack_reduce", "general_attack_flat", "general_defense_flat", "unit_attack_flat":
		changed := applyBattleStatBonus(c, params, t.effect)
		t.recordBeforeStatChanges(c, params, changed)
	}
}

// recordSuppressedUnits 把本场未参战但战后保留的兵力按受影响方写入上下文。
func recordSuppressedUnits(c *general.BeforeBattleContext, affected map[string]int) {
	if c == nil || len(affected) == 0 {
		return
	}
	target := &c.DefenderSuppressedUnits
	if c.Actor.Side == "defender" || c.Actor.Side == "reinforcement" {
		target = &c.AttackerSuppressedUnits
	}
	if *target == nil {
		*target = map[string]int{}
	}
	for unitType, amount := range affected {
		if amount > 0 {
			(*target)[unitType] += amount
		}
	}
}

// recordPreBattleLosses 把战前真实伤亡按受损方写入上下文，供应用层并入最终扣兵。
func recordPreBattleLosses(c *general.BeforeBattleContext, affected map[string]int) {
	if c == nil || len(affected) == 0 {
		return
	}
	target := &c.DefenderPreBattleLosses
	if c.Actor.Side == "defender" || c.Actor.Side == "reinforcement" {
		target = &c.AttackerPreBattleLosses
	}
	if *target == nil {
		*target = map[string]int{}
	}
	for unitType, amount := range affected {
		if amount > 0 {
			(*target)[unitType] += amount
		}
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
			changed = addTargetLosses(c.Result.AttackerLosses, params, c.Actor.TargetUnitType, c.Attacker)
			c.Result.AttackerLosses = changed.losses
			recomputeAttackerLossRate(c.Result)
		} else {
			changed = addTargetLosses(c.Result.DefenderLosses, params, c.Actor.TargetUnitType, c.Defender)
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
		outcomeKey := t.recordAfterCombatDetail(c, params, map[string]interface{}{"disabledTraitCount": 0})
		if side != "" && outcomeKey != "" && count > 0 {
			if c.DisabledTraitOutcomeKeys == nil {
				c.DisabledTraitOutcomeKeys = map[string][]string{}
			}
			for index := 0; index < count; index++ {
				c.DisabledTraitOutcomeKeys[side] = append(c.DisabledTraitOutcomeKeys[side], outcomeKey)
			}
		}
	}
}

// handleAfterBattle 处理战斗结束后的复活、止损和追击记录。
func (t *configurableTrait) handleAfterBattle(c *general.AfterBattleContext, params general.Params) {
	if c == nil {
		return
	}
	switch t.effect {
	case "revive":
		reviveParams := params
		if t.id == "guicai_yice" || t.id == "renzhu_shouhu" {
			reviveParams = cloneParams(params)
			reviveParams["maxReviveCount"] = 0
		}
		revived := reviveLosses(c, reviveParams)
		totalRevived := 0
		for _, amount := range revived {
			totalRevived += amount
		}
		t.recordAfterBattleDetail(c, params, map[string]interface{}{
			"actualLostUnits": clonePositiveUnitCounts(c.PlayerLosses),
			"revivedUnits":    revived,
			"totalRevived":    totalRevived,
		})
	case "reduce_final_losses":
		reduced := reduceFinalLosses(c, params)
		t.recordAfterBattle(c, params, reduced, "returnedUnits")
	}
}

// cloneParams 复制特性参数，供单个正式特性补充内部结算约束。
func cloneParams(params general.Params) general.Params {
	cloned := make(general.Params, len(params)+1)
	for key, value := range params {
		cloned[key] = value
	}
	return cloned
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
	target := strings.TrimSpace(c.Actor.TargetUnitType)
	if target != "" && target != c.UnitType && target != c.Category {
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
	if t.effect == "longdan_rescue" {
		t.handleLongdanPlunder(c, params)
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

// handleLongdanPlunder 按主将固定值和援军递减序列累计保护被掠夺城池的资源。
func (t *configurableTrait) handleLongdanPlunder(c *general.PlunderResolveContext, params general.Params) {
	baseRate := params.FloatWithBounds("plunderProtectionRate", 0.2, 0, 0.9)
	if baseRate <= 0 {
		return
	}
	contribution := 0.0
	switch c.Actor.Side {
	case "defender":
		if c.DefenderProtectionApplied {
			return
		}
		c.DefenderProtectionApplied = true
		contribution = baseRate
	case "reinforcement":
		contribution = baseRate / math.Pow(2, float64(c.ReinforcementProtectionCount))
		c.ReinforcementProtectionCount++
	default:
		return
	}
	if contribution <= 0 {
		return
	}
	if c.BaseRewards == nil {
		c.BaseRewards = cloneIntMap(c.Rewards)
	}
	before := cloneIntMap(c.Rewards)
	c.PlunderProtectionRate += contribution
	maxRate := baseRate * 3
	if maxRate > 0.9 {
		maxRate = 0.9
	}
	if c.PlunderProtectionRate > maxRate {
		c.PlunderProtectionRate = maxRate
	}
	protected := map[string]int{}
	changed := map[string]int{}
	for key, baseValue := range c.BaseRewards {
		next := int(math.Floor(float64(baseValue)*(1-c.PlunderProtectionRate) + 1e-9))
		if next < 0 {
			next = 0
		}
		protected[key] = before[key] - next
		changed[key] = next - before[key]
		c.Rewards[key] = next
	}
	t.recordPlunderDetail(c, params, map[string]interface{}{
		"plunderProtectionContributionRate": contribution,
		"cumulativePlunderProtectionRate":   c.PlunderProtectionRate,
		"protectedResources":                protected,
		"plunderDelta":                      changed,
	})
}

// cloneIntMap 复制整数映射，避免累计掠夺保护时丢失原始基数。
func cloneIntMap(source map[string]int) map[string]int {
	cloned := make(map[string]int, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

// sumPositiveMapValues 汇总逐兵种正数结果，供战报展示真实伤亡总量。
func sumPositiveMapValues(values map[string]int) int {
	total := 0
	for _, value := range values {
		if value > 0 {
			total += value
		}
	}
	return total
}

// triggered 按 GM 配置的触发概率判定是否触发。
func triggered(params general.Params) bool {
	return triggeredWithDefault(params, 1)
}

// triggeredWithDefault 使用指定默认概率，并严格处理必不触发和必定触发边界。
func triggeredWithDefault(params general.Params, defaultChance float64) bool {
	chance := params.FloatWithBounds("triggerChance", defaultChance, 0, 1)
	if chance <= 0 {
		return false
	}
	if chance >= 1 {
		return true
	}
	return rand.Float64() < chance
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
		if unit.Count <= 0 || !combatUnitMatchesTarget(*unit, targetUnitType) {
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

// reduceArmyUnitsByTotalRate 按目标兵种总人数精确抽离指定比例，并把取整余数稳定分配到原始兵种顺序。
func reduceArmyUnitsByTotalRate(army *combat.Army, params general.Params, rateKey string, targetUnitType string) map[string]int {
	if army == nil {
		return nil
	}
	rate := params.FloatWithBounds(rateKey, 0, 0, 1)
	targetUnitType = strings.TrimSpace(targetUnitType)
	total := 0
	for _, unit := range army.Units {
		if unit.Count > 0 && combatUnitMatchesTarget(unit, targetUnitType) {
			total += unit.Count
		}
	}
	target := int(math.Floor(float64(total) * rate))
	if target <= 0 {
		return nil
	}
	affected := map[string]int{}
	remainingTotal := total
	remainingTarget := target
	for index := range army.Units {
		unit := &army.Units[index]
		if unit.Count <= 0 || !combatUnitMatchesTarget(*unit, targetUnitType) {
			continue
		}
		count := remainingTarget
		if remainingTotal > unit.Count {
			count = int(math.Floor(float64(unit.Count*remainingTarget) / float64(remainingTotal)))
		}
		if count > unit.Count {
			count = unit.Count
		}
		if count > 0 {
			unit.Count -= count
			affected[unit.ID] += count
			remainingTarget -= count
		}
		remainingTotal -= unit.Count + count
	}
	return affected
}

type battleStatChanges struct {
	attack          map[string]int
	infantryDefense map[string]int
	cavalryDefense  map[string]int
}

// applyBattleStatBonus 在战斗前直接调整本次战斗单位属性，并分别记录攻击、步防和骑防实际变化。
func applyBattleStatBonus(c *general.BeforeBattleContext, params general.Params, effect string) battleStatChanges {
	army := actorSelfArmy(c)
	if effect == "enemy_defense_reduce" || effect == "enemy_attack_reduce" {
		army = actorEnemyArmy(c)
	}
	if army == nil {
		return battleStatChanges{}
	}
	target := strings.TrimSpace(c.Actor.TargetUnitType)
	changed := battleStatChanges{
		attack:          map[string]int{},
		infantryDefense: map[string]int{},
		cavalryDefense:  map[string]int{},
	}
	for i := range army.Units {
		unit := &army.Units[i]
		if !combatUnitMatchesTarget(*unit, target) {
			continue
		}
		beforeAttack := unit.Attack
		beforeInfantryDefense := unit.InfantryDefense
		beforeCavalryDefense := unit.CavalryDefense
		switch effect {
		case "unit_attack_bonus", "army_attack_bonus":
			unit.Attack += int(math.Round(float64(unit.Attack) * params.FloatWithBounds("attackBonusRate", params.FloatOr("effectRate", 0), 0, 2)))
			defenseBonus := params.FloatWithBounds("defenseBonusRate", 0, 0, 2)
			unit.InfantryDefense += int(math.Round(float64(unit.InfantryDefense) * defenseBonus))
			unit.CavalryDefense += int(math.Round(float64(unit.CavalryDefense) * defenseBonus))
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
		if delta := unit.Attack - beforeAttack; delta != 0 {
			changed.attack[unit.ID] = delta
		}
		if delta := unit.InfantryDefense - beforeInfantryDefense; delta != 0 {
			changed.infantryDefense[unit.ID] = delta
		}
		if delta := unit.CavalryDefense - beforeCavalryDefense; delta != 0 {
			changed.cavalryDefense[unit.ID] = delta
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
	rate := params.FloatWithBounds("effectRate", params.FloatOr("damagePercent", 0), 0, 1)
	maxRate := params.FloatWithBounds("maxAffectedRate", 1, 0, 1)
	if rate > maxRate {
		rate = maxRate
	}
	return addGroupedLosses(losses, rate, func(combat.UnitLoss) bool { return true })
}

// addTargetLosses 按具体兵种 ID 或兵种分类追加损失。
func addTargetLosses(losses []combat.UnitLoss, params general.Params, target string, army *combat.Army) lossChange {
	target = strings.TrimSpace(target)
	rate := params.FloatWithBounds("effectRate", 0, 0, 1)
	return addGroupedLosses(losses, rate, func(loss combat.UnitLoss) bool {
		return unitLossMatchesTarget(loss.ID, target, army)
	})
}

// addGroupedLosses 先按兵种汇总设计伤害，再按来源比例稳定分配，避免多支同兵种部队分别取整后少扣兵。
func addGroupedLosses(losses []combat.UnitLoss, rate float64, matches func(combat.UnitLoss) bool) lossChange {
	next := append([]combat.UnitLoss(nil), losses...)
	changed := map[string]int{}
	groups := map[string][]int{}
	unitOrder := []string{}
	for index, loss := range next {
		if !matches(loss) || loss.Count <= 0 {
			continue
		}
		if _, exists := groups[loss.ID]; !exists {
			unitOrder = append(unitOrder, loss.ID)
		}
		groups[loss.ID] = append(groups[loss.ID], index)
	}
	for _, unitType := range unitOrder {
		indices := groups[unitType]
		totalCount := 0
		totalAvailable := 0
		for _, index := range indices {
			totalCount += next[index].Count
			available := next[index].Count - next[index].Losses
			if available > 0 {
				totalAvailable += available
			}
		}
		wanted := int(math.Floor(float64(totalCount) * rate))
		if wanted > totalAvailable {
			wanted = totalAvailable
		}
		type lossRemainder struct {
			index    int
			fraction float64
		}
		remainders := make([]lossRemainder, 0, len(indices))
		assigned := 0
		for _, index := range indices {
			exact := float64(next[index].Count) * rate
			extra := int(math.Floor(exact))
			available := next[index].Count - next[index].Losses
			if extra > available {
				extra = available
			}
			if extra > 0 {
				next[index].Losses += extra
				assigned += extra
				changed[unitType] += extra
			}
			remainders = append(remainders, lossRemainder{index: index, fraction: exact - math.Floor(exact)})
		}
		sort.SliceStable(remainders, func(i, j int) bool {
			return remainders[i].fraction > remainders[j].fraction
		})
		for remaining := wanted - assigned; remaining > 0; {
			progressed := false
			for _, remainder := range remainders {
				if remaining <= 0 {
					break
				}
				index := remainder.index
				if next[index].Losses >= next[index].Count {
					continue
				}
				next[index].Losses++
				changed[unitType]++
				remaining--
				progressed = true
			}
			if !progressed {
				break
			}
		}
	}
	return lossChange{losses: next, byUnit: changed}
}

// combatUnitMatchesTarget 判断战斗单位是否命中特性指定的兵种 ID 或分类。
func combatUnitMatchesTarget(unit combat.Unit, target string) bool {
	target = strings.TrimSpace(target)
	return target == "" || unit.ID == target || strings.EqualFold(strings.TrimSpace(unit.Category), target)
}

// unitLossMatchesTarget 从战斗军团还原损失项的兵种分类，避免分类目标与真实兵种 ID 无法匹配。
func unitLossMatchesTarget(unitID string, target string, army *combat.Army) bool {
	target = strings.TrimSpace(target)
	if target == "" || unitID == target {
		return true
	}
	if army == nil {
		return false
	}
	for _, unit := range army.Units {
		if unit.ID == unitID {
			return strings.EqualFold(strings.TrimSpace(unit.Category), target)
		}
	}
	return false
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
		changed[next[i].ID] += reduced
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
	for _, unitType := range sortedLossUnitTypes(c.PlayerLosses) {
		lost := c.PlayerLosses[unitType]
		count := int(math.Floor(float64(lost) * rate))
		remainingLoss := lost - c.Revived[unitType]
		if count > remainingLoss {
			count = remainingLoss
		}
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

// clonePositiveUnitCounts 复制正数兵力明细，避免战报结果持有可变上下文。
func clonePositiveUnitCounts(source map[string]int) map[string]int {
	result := map[string]int{}
	for unitType, amount := range source {
		if amount > 0 {
			result[unitType] = amount
		}
	}
	return result
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
	t.recordBeforeDetail(c, params, map[string]interface{}{key: values})
}

// recordBeforeStatChanges 把三种战斗属性的实际变化拆开写入战报，避免合计值含义不明。
func (t *configurableTrait) recordBeforeStatChanges(c *general.BeforeBattleContext, params general.Params, changed battleStatChanges) {
	detail := map[string]interface{}{}
	if len(changed.attack) > 0 {
		detail["attackModifiedUnits"] = changed.attack
	}
	if len(changed.infantryDefense) > 0 {
		detail["infantryDefenseModifiedUnits"] = changed.infantryDefense
	}
	if len(changed.cavalryDefense) > 0 {
		detail["cavalryDefenseModifiedUnits"] = changed.cavalryDefense
	}
	if len(detail) == 0 {
		return
	}
	t.recordBeforeDetail(c, params, detail)
}

// recordBeforeDetail 写入一条结构化战前特性结果。
func (t *configurableTrait) recordBeforeDetail(c *general.BeforeBattleContext, params general.Params, detail map[string]interface{}) {
	if c.Triggered == nil {
		c.Triggered = map[string]general.TraitOutcome{}
	}
	storeTraitOutcome(c.Triggered, t.outcome(c.Actor, params, detail))
}

// recordAfterCombat 写入战斗结算特性结果。
func (t *configurableTrait) recordAfterCombat(c *general.AfterCombatResolveContext, params general.Params, values map[string]int, key string) string {
	if len(values) == 0 {
		return ""
	}
	return t.recordAfterCombatDetail(c, params, map[string]interface{}{key: values})
}

// recordAfterCombatDetail 写入结构化战斗结算特性结果，并返回实际存储 key。
func (t *configurableTrait) recordAfterCombatDetail(c *general.AfterCombatResolveContext, params general.Params, detail map[string]interface{}) string {
	if c.Triggered == nil {
		c.Triggered = map[string]general.TraitOutcome{}
	}
	return storeTraitOutcome(c.Triggered, t.outcome(c.Actor, params, detail))
}

// recordAfterBattle 写入战斗结束特性结果。
func (t *configurableTrait) recordAfterBattle(c *general.AfterBattleContext, params general.Params, values map[string]int, key string) {
	if len(values) == 0 {
		return
	}
	if c.Triggered == nil {
		c.Triggered = map[string]general.TraitOutcome{}
	}
	storeTraitOutcome(c.Triggered, t.outcome(c.Actor, params, map[string]interface{}{key: values}))
}

// recordAfterBattleDetail 写入包含多项真实结果的战斗结束特性明细。
func (t *configurableTrait) recordAfterBattleDetail(c *general.AfterBattleContext, params general.Params, detail map[string]interface{}) {
	if len(detail) == 0 {
		return
	}
	if c.Triggered == nil {
		c.Triggered = map[string]general.TraitOutcome{}
	}
	storeTraitOutcome(c.Triggered, t.outcome(c.Actor, params, detail))
}

// recordMarch 写入行军特性结果。
func (t *configurableTrait) recordMarch(c *general.MarchCreateContext, params general.Params, values map[string]int, key string) {
	if c.Triggered == nil {
		c.Triggered = map[string]general.TraitOutcome{}
	}
	storeTraitOutcome(c.Triggered, t.outcome(c.Actor, params, map[string]interface{}{key: values}))
}

// recordRecruit 写入征兵特性结果。
func (t *configurableTrait) recordRecruit(c *general.RecruitCostContext, params general.Params, values map[string]int, key string) {
	if len(values) == 0 {
		return
	}
	if c.Triggered == nil {
		c.Triggered = map[string]general.TraitOutcome{}
	}
	storeTraitOutcome(c.Triggered, t.outcome(c.Actor, params, map[string]interface{}{key: values}))
}

// recordPlunder 写入掠夺特性结果。
func (t *configurableTrait) recordPlunder(c *general.PlunderResolveContext, params general.Params, values map[string]int, key string) {
	if len(values) == 0 {
		return
	}
	t.recordPlunderDetail(c, params, map[string]interface{}{key: values})
}

// recordPlunderDetail 写入包含累计保护比例和实际资源变化的掠夺特性结果。
func (t *configurableTrait) recordPlunderDetail(c *general.PlunderResolveContext, params general.Params, detail map[string]interface{}) {
	if len(detail) == 0 {
		return
	}
	if c.Triggered == nil {
		c.Triggered = map[string]general.TraitOutcome{}
	}
	storeTraitOutcome(c.Triggered, t.outcome(c.Actor, params, detail))
}

// storeTraitOutcome 保存特性结果，并在双方同特性同时触发时避免 map 覆盖。
func storeTraitOutcome(outcomes map[string]general.TraitOutcome, outcome general.TraitOutcome) string {
	key := outcome.TraitID
	if current, exists := outcomes[key]; !exists || sameTraitOutcomeOwner(current, outcome) {
		outcomes[key] = outcome
		return key
	}
	base := key + "::" + outcome.OwnerSide + "::" + outcome.OwnerGeneralID
	key = base
	for index := 2; ; index++ {
		if current, exists := outcomes[key]; !exists || sameTraitOutcomeOwner(current, outcome) {
			outcomes[key] = outcome
			return key
		}
		key = base + "::" + strconv.Itoa(index)
	}
}

// sameTraitOutcomeOwner 判断两条结果是否来自同一特性归属者。
func sameTraitOutcomeOwner(left general.TraitOutcome, right general.TraitOutcome) bool {
	return left.TraitID == right.TraitID && left.OwnerSide == right.OwnerSide && left.OwnerGeneralID == right.OwnerGeneralID && left.OwnerPlayerID == right.OwnerPlayerID
}

// outcome 生成标准战报特性结果。
func (t *configurableTrait) outcome(actor general.TraitActor, params general.Params, detail map[string]interface{}) general.TraitOutcome {
	detail["triggerChance"] = params.FloatWithBounds("triggerChance", t.triggerChanceDefault(), 0, 1)
	if t.effect == "enemy_attack_reduce" {
		detail["attackReductionRate"] = params.FloatWithBounds("attackReductionRate", params.FloatOr("effectRate", 0), 0, 0.9)
	} else if effectRate, configured := params["effectRate"]; configured {
		detail["effectRate"] = effectRate
	}
	if maxAffectedRate, configured := params["maxAffectedRate"]; configured && t.id != "yibing_touxi" && t.id != "weizhen_zhenhe" {
		detail["maxAffectedRate"] = params.FloatWithBounds("maxAffectedRate", maxAffectedRate, 0, 1)
	}
	if attackBonusRate, configured := params["attackBonusRate"]; configured {
		detail["attackBonusRate"] = attackBonusRate
	}
	if unitAttackFlat, configured := params["unitAttackFlat"]; configured {
		detail["unitAttackFlat"] = unitAttackFlat
	}
	if enemyDefenseReductionRate, configured := params["enemyDefenseReductionRate"]; configured {
		detail["enemyDefenseReductionRate"] = enemyDefenseReductionRate
	}
	if defenseBonusRate, configured := params["defenseBonusRate"]; configured {
		detail["defenseBonusRate"] = defenseBonusRate
	}
	if plunderProtectionRate, configured := params["plunderProtectionRate"]; configured {
		detail["plunderProtectionRate"] = params.FloatWithBounds("plunderProtectionRate", plunderProtectionRate, 0, 0.9)
	}
	if generalDefenseFlat, configured := params["generalDefenseFlat"]; configured {
		detail["generalDefenseFlat"] = generalDefenseFlat
	}
	if lossReductionRate, configured := params["lossReductionRate"]; configured {
		detail["lossReductionRate"] = params.FloatWithBounds("lossReductionRate", lossReductionRate, 0, 0.8)
	}
	if _, configured := params["maxReturnCount"]; configured {
		detail["maxReturnCount"] = params.IntWithBounds("maxReturnCount", 10000, 0, 1000000)
	}
	if plunderBonusRate, configured := params["plunderBonusRate"]; configured {
		detail["plunderBonusRate"] = params.FloatWithBounds("plunderBonusRate", plunderBonusRate, -0.9, 1)
	}
	if _, configured := params["disableTraitCount"]; configured {
		detail["disableTraitCount"] = params.IntWithBounds("disableTraitCount", 1, 0, 24)
	}
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
	if event == general.EventBattleTraitControl && effect == "disable_all_combat_traits" {
		return 100
	}
	if event == general.EventAfterCombatResolve && effect == "disable_traits" {
		return 90
	}
	if event == general.EventBeforeBattle {
		return 70
	}
	return 40
}

// triggerChanceDefault 返回特性的默认触发概率，未单独指定时保持历史默认必定触发。
func (t *configurableTrait) triggerChanceDefault() float64 {
	if t.defaultTriggerChance > 0 && t.defaultTriggerChance <= 1 {
		return t.defaultTriggerChance
	}
	return 1
}

// commonScopeFields 返回所有特性共享的安全边界字段。
func commonScopeFields(defaultTriggerChance float64) []general.ParamField {
	return []general.ParamField{
		{Key: "triggerChance", Label: "触发概率", Description: "1 表示必定触发，0.5 表示 50% 概率", Default: defaultTriggerChance, Min: 0, Max: 1, Step: 0.01},
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
		{id: "weiwu_haoling", name: "魏武号令", traitType: general.TraitTypeSpecial, description: "曹操留城时每分钟自动获得 300 虎卫，按后端经过时间权威结算且不设产兵上限；离城期间停止且不作为战斗触发。", effect: "resource_settlement_guard", events: nil, schema: []general.ParamField{countField("guardPerMinute", "每分钟虎卫", 300, 10000)}},
		{id: "weiwu_tongyu", name: "魏武统御", traitType: general.TraitTypeBonus, description: "曹操所率全军防御提升 15%，仅在守城或增援战斗前生效，主动进攻无效。", effect: "army_defense_bonus", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("defenseBonusRate", "防御加成", 0.15, 2)}},
		{id: "yibing_touxi", name: "疑兵偷袭", traitType: general.TraitTypeSpecial, description: "进攻、防守或增援战斗前概率使敌方全军直接形成真实伤亡，剩余兵力再进入正常攻防计算。", effect: "pre_damage", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("effectRate", "真实伤亡比例", 0.35, 1)}, defaultTriggerChance: 0.35},
		{id: "mouding_houfa", name: "谋定后发", traitType: general.TraitTypeBonus, description: "防守或增援战斗前概率提升所率全军防御，主动进攻无效。", effect: "army_defense_bonus", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("defenseBonusRate", "全军防御提升", 0.35, 2)}, defaultTriggerChance: 0.35},
		{id: "meiren", name: "美人心计", traitType: general.TraitTypeSpecial, description: "主动进攻战斗前概率提升所率全军攻击 25%，防守和增援无效。", effect: "army_attack_bonus", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("attackBonusRate", "全军攻击提升", 0.25, 2)}, defaultTriggerChance: 0.5},
		{id: "meihuo_raozhen", name: "魅惑扰阵", traitType: general.TraitTypeBonus, description: "主动进攻战斗前概率降低敌方全军防御 25%，防守和增援无效。", effect: "enemy_defense_reduce", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("enemyDefenseReductionRate", "敌方防御降低", 0.25, 0.9)}, defaultTriggerChance: 0.5},
		{id: "huchi_chongzhen", name: "虎痴冲阵", traitType: general.TraitTypeSpecial, description: "主动进攻战斗前概率降低敌方全军防御，防守和增援无效。", effect: "enemy_defense_reduce", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("enemyDefenseReductionRate", "敌方全军防御降低", 0.3, 0.9)}, defaultTriggerChance: 0.5},
		{id: "huhu_shengwei", name: "虎虎生威", traitType: general.TraitTypeBonus, description: "被动使所率虎豹骑固定增加攻击和移动，不作为战斗触发特性。", effect: "passive_unit_stats", events: nil, schema: []general.ParamField{countField("unitAttackFlat", "虎豹骑攻击增加", 12, 100000), countField("unitSpeedFlat", "虎豹骑移动增加", 5, 100000)}},
		{id: "huzhu_xuezhan", name: "护主血战", traitType: general.TraitTypeSpecial, description: "防守或增援战斗前使所率禁卫甲士固定增加步兵防御和骑兵防御，主动进攻无效。", effect: "general_defense_flat", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{countField("generalDefenseFlat", "禁卫甲士步防与骑防增加", 20, 100000)}},
		{id: "sizhandaodi", name: "死战到底", traitType: general.TraitTypeBonus, description: "主动进攻战斗前概率提升所率步兵攻击，防守和增援无效。", effect: "unit_attack_bonus", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("attackBonusRate", "步兵攻击提升", 0.35, 2)}, defaultTriggerChance: 0.6},
		{id: "jixing_benxi", name: "疾行奔袭", traitType: general.TraitTypeSpecial, description: "被动使所率骁骑营固定增加攻击和移动，不作为战斗触发特性。", effect: "passive_unit_stats", events: nil, schema: []general.ParamField{countField("unitAttackFlat", "骁骑营攻击增加", 18, 100000), countField("unitSpeedFlat", "骁骑营移动增加", 5, 100000)}},
		{id: "dunzhen_fangyu", name: "盾阵防御", traitType: general.TraitTypeBonus, description: "防守或增援战斗前概率提升所率全军防御。", effect: "army_defense_bonus", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("defenseBonusRate", "全军防御提升", 0.3, 2)}, defaultTriggerChance: 0.6},
		{id: "weizhen_zhenhe", name: "震慑全军", traitType: general.TraitTypeSpecial, description: "主动进攻战斗前概率使敌方部分兵力溃逃；溃逃兵不参与本次攻防、不计死亡，战后完整返回敌方部队。", effect: "suppress", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("effectRate", "溃逃比例", 0.25, 1)}, defaultTriggerChance: 0.35},
		{id: "weizhen_xiaoyao", name: "威震逍遥", traitType: general.TraitTypeBonus, description: "主动进攻战斗前概率提升所率骑兵攻击。", effect: "unit_attack_bonus", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("attackBonusRate", "骑兵攻击提升", 0.35, 2)}, defaultTriggerChance: 0.6},
		{id: "shengui_zhicai", name: "神鬼之才", traitType: general.TraitTypeSpecial, description: "全天候被动增加郭嘉的内政和智谋，不作为战斗触发特性。", effect: "passive_stat", events: nil, schema: []general.ParamField{countField("politicsBonus", "内政增加", 10, 100), countField("intelligenceBonus", "智谋增加", 10, 100)}},
		{id: "guicai_yice", name: "鬼才遗策", traitType: general.TraitTypeBonus, description: "进攻、防守或增援战斗结束后，按所率部队本场真实阵亡复活士兵。", effect: "revive", events: []string{general.EventAfterBattle}, schema: []general.ParamField{rateField("effectRate", "真实阵亡复活比例", 0.22, 1)}},
		{id: "wangzuo_zhicai", name: "王佐之才", traitType: general.TraitTypeSpecial, description: "将领留城时降低征兵资源消耗，离城后失效。", effect: "recruit_cost_reduce", events: []string{general.EventRecruitCost}, schema: []general.ParamField{rateField("resourceCostReduction", "征兵消耗降低", 0.05, 0.8)}},
		{id: "neizheng_jingying", name: "内政精营", traitType: general.TraitTypeBonus, description: "被动提升资源产量。", effect: "passive_modifier", events: nil, schema: []general.ParamField{rateField("productionBonusRate", "资源产量提升", 0.05, 0.5)}},
		{id: "rende", name: "仁德天下", traitType: general.TraitTypeSpecial, description: "全天候被动增加刘备的内政和统率，不作为战斗触发特性。", effect: "passive_stat", events: nil, schema: []general.ParamField{countField("politicsBonus", "内政增加", 10, 100), countField("commandBonus", "统率增加", 12, 100)}},
		{id: "renzhu_shouhu", name: "仁主守护", traitType: general.TraitTypeBonus, description: "进攻、防守或增援战斗结束后，概率按所率部队本场真实阵亡复活士兵。", effect: "revive", events: []string{general.EventAfterBattle}, schema: []general.ParamField{rateField("effectRate", "真实阵亡复活比例", 0.35, 1)}, defaultTriggerChance: 0.6},
		{id: "shuiyan_qijun", name: "水淹七军", traitType: general.TraitTypeSpecial, description: "主动进攻战斗前概率使敌方全军直接形成真实伤亡，剩余兵力再进入正常攻防。", effect: "pre_damage", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("effectRate", "真实伤亡比例", 0.3, 1)}, defaultTriggerChance: 0.35},
		{id: "wusheng_pojun", name: "武圣破军", traitType: general.TraitTypeBonus, description: "主动进攻战斗前概率提升所率青龙军攻击。", effect: "unit_attack_bonus", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("attackBonusRate", "青龙军攻击提升", 0.38, 2)}, defaultTriggerChance: 0.5},
		{id: "zhenhe_quanjun", name: "万人怒吼", traitType: general.TraitTypeSpecial, description: "主动进攻战斗前概率使敌方全军部分兵力逃跑；逃兵不参战、不死亡，战后完整返回。", effect: "suppress", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("effectRate", "逃跑比例", 0.5, 1)}, defaultTriggerChance: 0.5},
		{id: "wanren_nuhou", name: "勇冠三军", traitType: general.TraitTypeBonus, description: "主动进攻战斗前概率提升所率南蛮象攻击。", effect: "unit_attack_bonus", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("attackBonusRate", "南蛮象攻击提升", 0.35, 2)}, defaultTriggerChance: 0.4},
		{id: "qimen_dunjia", name: "奇门遁甲", traitType: general.TraitTypeSpecial, description: "进攻、防守或增援战斗前使敌方部分兵力仅本场不参战，战后完整保留。", effect: "suppress", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("effectRate", "困敌比例", 0.25, 1)}},
		{id: "wolong_mouzhi", name: "卧龙奇谋", traitType: general.TraitTypeBonus, description: "进攻、防守或增援战斗前概率封禁敌方所有参战将领的战斗触发型特性，不影响被动特性。", effect: "disable_all_combat_traits", events: []string{general.EventBattleTraitControl}, defaultTriggerChance: 0.6},
		{id: "longdan_jiuyuan", name: "龙胆救援", traitType: general.TraitTypeSpecial, description: "防守或增援战斗前使所率麒麟卫步防和骑防提升 25%；掠夺结算时为被保护城池保留资源。", effect: "longdan_rescue", events: []string{general.EventBeforeBattle, general.EventPlunderResolve}, schema: []general.ParamField{rateField("defenseBonusRate", "麒麟卫双防提升", 0.25, 2), rateField("plunderProtectionRate", "基础资源保护比例", 0.2, 0.9)}},
		{id: "qijin_qichu", name: "七进七出", traitType: general.TraitTypeBonus, description: "主动出征或增援创建时缩短全军行军时间，不作为战斗触发特性。", effect: "march_speed", events: []string{general.EventMarchCreate}, schema: []general.ParamField{rateField("speedBonusRate", "速度提升", 1, 10), countField("minMarchSeconds", "最短行军秒数", 60, 86400)}},
		{id: "xiliang_tuji", name: "西凉突击", traitType: general.TraitTypeSpecial, description: "进攻、守城或作为援军时只对敌方骑兵追加冲锋损失。", effect: "target_unit_damage", events: []string{general.EventAfterCombatResolve}, schema: []general.ParamField{rateField("effectRate", "冲锋伤害", 0.12, 1)}},
		{id: "tianshen_xiafan", name: "天神下凡", traitType: general.TraitTypeBonus, description: "被动增加武将武力，不作为战斗触发特性。", effect: "passive_stat", events: nil, schema: []general.ParamField{countField("forceBonus", "武力增加", 20, 100)}},
		{id: "baibu_chuanyang", name: "百步穿杨", traitType: general.TraitTypeSpecial, description: "主动进攻战斗前概率使敌方全军防御降低 30%。", effect: "enemy_defense_reduce", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("enemyDefenseReductionRate", "敌方防御降低", 0.3, 0.9)}, defaultTriggerChance: 0.45},
		{id: "laodang_yizhuang", name: "老当益壮", traitType: general.TraitTypeBonus, description: "永久增加黄忠武力 12 点、统率 12 点，不作为战斗触发特性。", effect: "passive_stat", events: nil, schema: []general.ParamField{countField("forceBonus", "武力增加", 12, 100), countField("commandBonus", "统率增加", 12, 100)}},
		{id: "qibing_raohou", name: "奇兵绕后", traitType: general.TraitTypeSpecial, description: "被动使所率南蛮象固定增加攻击 18 点、移动 15 点，不作为战斗触发特性。", effect: "passive_unit_stats", events: nil, schema: []general.ParamField{countField("unitAttackFlat", "南蛮象攻击增加", 18, 100000), countField("unitSpeedFlat", "南蛮象移动增加", 15, 100000)}},
		{id: "gushou_hanzhong", name: "固守汉中", traitType: general.TraitTypeBonus, description: "防守或增援时固定提升全军步兵、骑兵防御。", effect: "general_defense_flat", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{countField("generalDefenseFlat", "全军防御", 20, 100000)}},
		{id: "jiangdong_haoling", name: "江东号令", traitType: general.TraitTypeSpecial, description: "防守失败时降低敌方掠夺收益。", effect: "plunder_reduce", events: []string{general.EventPlunderResolve}, schema: []general.ParamField{{Key: "plunderBonusRate", Label: "掠夺收益修正", Description: "负数降低敌方掠夺收益，正数提高收益", Default: -0.2, Min: -0.9, Max: 1, Step: 0.01}}},
		{id: "jiangdong_gushou", name: "江东固守", traitType: general.TraitTypeBonus, description: "防守或增援时概率提升全军防御。", effect: "army_defense_bonus", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("defenseBonusRate", "防御提升", 0.5, 2)}},
		{id: "xiaobawang_zhuiji", name: "小霸王追击", traitType: general.TraitTypeSpecial, description: "掠夺战胜利后追加追击损失。", effect: "extra_damage", events: []string{general.EventAfterCombatResolve}, schema: []general.ParamField{rateField("effectRate", "追击伤害", 0.1, 1)}},
		{id: "xiaobawang_tieqi", name: "小霸王", traitType: general.TraitTypeBonus, description: "主动进攻时固定提升霸王骑攻击。", effect: "unit_attack_flat", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{countField("unitAttackFlat", "兵种攻击", 50, 100000)}},
		{id: "meizhoulang_junlue", name: "美周郎军略", traitType: general.TraitTypeBonus, description: "主动进攻时提升全军攻击。", effect: "army_attack_bonus", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("attackBonusRate", "攻击提升", 0.05, 2)}},
		{id: "huoshao_lianying", name: "火烧联营", traitType: general.TraitTypeSpecial, description: "触发后按敌方步兵原始人数追加 100% 损失，使目标步兵最终全灭。", effect: "target_unit_damage", events: []string{general.EventAfterCombatResolve}, schema: []general.ParamField{rateField("effectRate", "步兵伤害", 1, 1)}},
		{id: "lianying_zengshang", name: "连营增伤", traitType: general.TraitTypeBonus, description: "对步兵伤害提升。", effect: "target_unit_damage", events: []string{general.EventAfterCombatResolve}, schema: []general.ParamField{rateField("effectRate", "对步兵伤害", 0.1, 1)}},
		{id: "baiyi_dujiang", name: "白衣渡江", traitType: general.TraitTypeSpecial, description: "出征或增援创建时概率提升行军速度，最低时长由配置约束。", effect: "march_speed", events: []string{general.EventMarchCreate}, schema: []general.ParamField{rateField("speedBonusRate", "速度提升", 0.2, 10), countField("minMarchSeconds", "最短行军秒数", 60, 86400)}},
		{id: "baiyi_jixing", name: "白衣急行", traitType: general.TraitTypeBonus, description: "出征或增援创建时固定提升行军速度，可与其他行军特性逐次叠加。", effect: "march_speed", events: []string{general.EventMarchCreate}, schema: []general.ParamField{rateField("speedBonusRate", "速度提升", 0.2, 10), countField("minMarchSeconds", "最短行军秒数", 60, 86400)}},
		{id: "kuairu_shandian", name: "快如闪电", traitType: general.TraitTypeSpecial, description: "出征或增援创建时概率大幅提升行军速度，最低时长由配置约束。", effect: "march_speed", events: []string{general.EventMarchCreate}, schema: []general.ParamField{rateField("speedBonusRate", "速度提升", 4, 10), countField("minMarchSeconds", "最短行军秒数", 30, 86400)}},
		{id: "xinyi_yonglie", name: "信义勇烈", traitType: general.TraitTypeBonus, description: "只提升将领所带援军自身的步兵、骑兵防御。", effect: "army_defense_bonus", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("defenseBonusRate", "援军防御提升", 0.1, 2)}},
		{id: "jinfan_jielue", name: "锦帆劫掠", traitType: general.TraitTypeSpecial, description: "掠夺收益提升。", effect: "plunder_bonus", events: []string{general.EventPlunderResolve}, schema: []general.ParamField{rateField("plunderBonusRate", "掠夺收益提升", 0.2, 1)}},
		{id: "jinfan_qixi", name: "锦帆奇袭", traitType: general.TraitTypeBonus, description: "掠夺战攻击提升。", effect: "army_attack_bonus", events: []string{general.EventBeforeBattle}, schema: []general.ParamField{rateField("attackBonusRate", "攻击提升", 0.1, 2)}},
		{id: "kurouji", name: "苦肉计", traitType: general.TraitTypeSpecial, description: "概率压制敌方一定数量的后续特性。", effect: "disable_traits", events: []string{general.EventAfterCombatResolve}, schema: []general.ParamField{countField("disableTraitCount", "压制特性数", 1, 24)}},
		{id: "kurou_fanji", name: "苦肉反击", traitType: general.TraitTypeBonus, description: "战斗结算后追加敌方损失。", effect: "counter_damage", events: []string{general.EventAfterCombatResolve}, schema: []general.ParamField{rateField("effectRate", "反击增伤", 0.1, 1)}},
	} {
		registerConfigurableTrait(item)
	}
}
