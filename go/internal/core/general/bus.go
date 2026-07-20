// 本文件实现将领特性事件分发、归属注入和触发边界校验。
package general

import (
	"sort"
	"strings"
)

// ActiveTrait 当前生效的特性实例（玩家激活的特性 + 当前参数）
type ActiveTrait struct {
	TraitID         string
	TraitType       string
	OwnerSide       string
	OwnerPlayerID   string
	OwnerGeneralID  string
	Scope           string
	TargetUnitType  string
	AllowedSides    []string
	AllowedScenes   []string
	RequiredOutcome string
	Params          Params
}

// Dispatch 分发一个事件到所有订阅了它的特性
//
// activeTraits 是当前玩家激活的特性列表（来自 GeneralConfig 的 traits 字段）
// 调用方根据上下文（玩家的 General）准备好这个列表
func Dispatch(ctx EventContext, activeTraits []ActiveTrait) {
	if len(activeTraits) == 0 {
		return
	}

	eventType := ctx.EventType()

	// 收集所有订阅了该事件的 handler
	type subscription struct {
		handler  func(EventContext, Params)
		params   Params
		active   ActiveTrait
		priority int
	}
	var subs []subscription

	for _, at := range activeTraits {
		trait, ok := Get(at.TraitID)
		if !ok {
			continue
		}
		if at.TraitType != "" && trait.Type() != at.TraitType {
			continue
		}
		for _, sub := range trait.Subscribe() {
			if sub.Event == eventType {
				subs = append(subs, subscription{
					handler:  sub.Handle,
					params:   at.Params,
					active:   at,
					priority: sub.Priority,
				})
			}
		}
	}

	if len(subs) == 0 {
		return
	}

	// 按优先级降序（priority 大的先执行）
	sort.SliceStable(subs, func(i, j int) bool {
		return subs[i].priority > subs[j].priority
	})

	// 同一优先级视为同时结算，只使用本组开始前已经存在的压制次数。
	// 这样双方的压制特性可以同时生效，并且只影响优先级更低的后续特性。
	for start := 0; start < len(subs); {
		end := start + 1
		for end < len(subs) && subs[end].priority == subs[start].priority {
			end++
		}
		disabledBudget := snapshotDisabledTraitSides(ctx)
		for _, sub := range subs[start:end] {
			applyActor(ctx, sub.active)
			if !traitAppliesToContext(ctx, sub.active) {
				continue
			}
			if traitSideDisabledFromBudget(ctx, disabledBudget) {
				continue
			}
			sub.handler(ctx, sub.params)
		}
		start = end
	}
}

// traitAppliesToContext 统一校验特性的阵营、场景和胜负触发条件。
func traitAppliesToContext(ctx EventContext, active ActiveTrait) bool {
	side, scene := traitContextSideAndScene(ctx)
	if !containsTraitConstraint(active.AllowedSides, side) {
		return false
	}
	if !containsTraitConstraint(active.AllowedScenes, scene) {
		return false
	}
	requiredOutcome := strings.ToLower(strings.TrimSpace(active.RequiredOutcome))
	if requiredOutcome == "" {
		return true
	}
	won, known := traitActorWon(ctx, side)
	if !known {
		return false
	}
	return (requiredOutcome == "win" && won) || (requiredOutcome == "loss" && !won)
}

// containsTraitConstraint 判断当前值是否满足可选白名单，空白名单表示不限制。
func containsTraitConstraint(allowed []string, current string) bool {
	if len(allowed) == 0 {
		return true
	}
	current = strings.ToLower(strings.TrimSpace(current))
	for _, value := range allowed {
		if strings.ToLower(strings.TrimSpace(value)) == current {
			return true
		}
	}
	return false
}

// traitContextSideAndScene 读取当前特性执行时的归属方和玩法场景。
func traitContextSideAndScene(ctx EventContext) (string, string) {
	switch c := ctx.(type) {
	case *BeforeBattleContext:
		return c.Actor.Side, c.Scene
	case *AfterCombatResolveContext:
		return c.Actor.Side, c.Scene
	case *AfterBattleContext:
		return c.Actor.Side, c.Scene
	case *MarchCreateContext:
		return c.Actor.Side, c.Scene
	case *RecruitCostContext:
		return c.Actor.Side, "recruit"
	case *PlunderResolveContext:
		return c.Actor.Side, c.Scene
	default:
		return "", ""
	}
}

// traitActorWon 判断当前特性归属方是否赢得本场战斗。
func traitActorWon(ctx EventContext, side string) (bool, bool) {
	switch c := ctx.(type) {
	case *AfterCombatResolveContext:
		if c.Result == nil || strings.TrimSpace(c.Result.Winner) == "" || c.Result.Winner == "draw" {
			return false, false
		}
		winnerSide := side
		if winnerSide == "reinforcement" {
			winnerSide = "defender"
		}
		return c.Result.Winner == winnerSide, true
	case *AfterBattleContext:
		winner := strings.ToLower(strings.TrimSpace(c.Winner))
		if winner != "" {
			if winner == "draw" {
				return false, false
			}
			winnerSide := side
			if winnerSide == "reinforcement" {
				winnerSide = "defender"
			}
			return winner == winnerSide, true
		}
		return c.Won, true
	default:
		return false, false
	}
}

// applyActor 把当前分发特性的归属写入事件上下文。
func applyActor(ctx EventContext, at ActiveTrait) {
	actor := TraitActor{
		Side:           at.OwnerSide,
		PlayerID:       at.OwnerPlayerID,
		GeneralID:      at.OwnerGeneralID,
		Scope:          at.Scope,
		TargetUnitType: at.TargetUnitType,
	}
	switch c := ctx.(type) {
	case *BeforeBattleContext:
		if at.OwnerSide != "" {
			actor.Side = at.OwnerSide
		} else {
			actor.Side = "attacker"
			if c.DefenderOwnsTrait && !c.AttackerOwnsTrait {
				actor.Side = "defender"
			}
		}
		c.Actor = actor
	case *AfterCombatResolveContext:
		if at.OwnerSide != "" {
			actor.Side = at.OwnerSide
		} else {
			actor.Side = "attacker"
			if c.DefenderOwnsTrait && !c.AttackerOwnsTrait {
				actor.Side = "defender"
			}
		}
		c.Actor = actor
	case *AfterBattleContext:
		if at.OwnerSide != "" {
			actor.Side = at.OwnerSide
		} else if c.IsAttacker {
			actor.Side = "attacker"
		} else {
			actor.Side = "defender"
		}
		if at.Scope == "reinforcement_self" {
			actor.Side = "reinforcement"
		}
		c.Actor = actor
	case *MarchCreateContext:
		c.Actor = actor
	case *RecruitCostContext:
		c.Actor = actor
	case *PlunderResolveContext:
		c.Actor = actor
	}
}

// snapshotDisabledTraitSides 复制当前压制次数，作为同优先级批次的固定预算。
func snapshotDisabledTraitSides(ctx EventContext) map[string]int {
	c, ok := ctx.(*AfterCombatResolveContext)
	if !ok || len(c.DisabledTraitSide) == 0 {
		return nil
	}
	budget := make(map[string]int, len(c.DisabledTraitSide))
	for side, count := range c.DisabledTraitSide {
		budget[side] = count
	}
	return budget
}

// traitSideDisabledFromBudget 判断当前触发方是否被更高优先级特性压制。
func traitSideDisabledFromBudget(ctx EventContext, budget map[string]int) bool {
	c, ok := ctx.(*AfterCombatResolveContext)
	if !ok || len(budget) == 0 {
		return false
	}
	side := c.Actor.Side
	if side == "" || budget[side] <= 0 {
		return false
	}
	budget[side]--
	if c.DisabledTraitSide[side] > 0 {
		c.DisabledTraitSide[side]--
	}
	RecordActualTraitSuppressions(c, side, 1)
	return true
}

// RecordActualTraitSuppressions 在事件总线或后续援军流程确实跳过特性时回填实际拦截数量。
func RecordActualTraitSuppressions(c *AfterCombatResolveContext, side string, count int) {
	if c == nil || count <= 0 {
		return
	}
	for index := 0; index < count; index++ {
		keys := c.DisabledTraitOutcomeKeys[side]
		if len(keys) == 0 {
			return
		}
		outcomeKey := keys[0]
		c.DisabledTraitOutcomeKeys[side] = keys[1:]
		outcome, ok := c.Triggered[outcomeKey]
		if !ok || outcome.Detail == nil {
			continue
		}
		actual, _ := outcome.Detail["disabledTraitCount"].(int)
		outcome.Detail["disabledTraitCount"] = actual + 1
		c.Triggered[outcomeKey] = outcome
	}
}
