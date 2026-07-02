package general

import (
	"sort"
)

// ActiveTrait 当前生效的特性实例（玩家激活的特性 + 当前参数）
type ActiveTrait struct {
	TraitID        string
	TraitType      string
	OwnerSide      string
	OwnerPlayerID  string
	OwnerGeneralID string
	Scope          string
	TargetUnitType string
	Params         Params
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

	for _, sub := range subs {
		applyActor(ctx, sub.active)
		if traitSideDisabled(ctx) {
			continue
		}
		sub.handler(ctx, sub.params)
	}
}

// applyActor 把当前分发特性的归属写入事件上下文。
func applyActor(ctx EventContext, at ActiveTrait) {
	actor := TraitActor{
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

// traitSideDisabled 判断当前触发方是否已被前置特性压制。
func traitSideDisabled(ctx EventContext) bool {
	c, ok := ctx.(*AfterCombatResolveContext)
	if !ok || len(c.DisabledTraitSide) == 0 {
		return false
	}
	side := c.Actor.Side
	if side == "" || c.DisabledTraitSide[side] <= 0 {
		return false
	}
	c.DisabledTraitSide[side]--
	return true
}
