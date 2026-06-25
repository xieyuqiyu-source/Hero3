package general

import (
	"sort"
)

// ActiveTrait 当前生效的特性实例（玩家激活的特性 + 当前参数）
type ActiveTrait struct {
	TraitID string
	Params  Params
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
		priority int
	}
	var subs []subscription

	for _, at := range activeTraits {
		trait, ok := Get(at.TraitID)
		if !ok {
			continue
		}
		for _, sub := range trait.Subscribe() {
			if sub.Event == eventType {
				subs = append(subs, subscription{
					handler:  sub.Handle,
					params:   at.Params,
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
		sub.handler(ctx, sub.params)
	}
}
