// Package general 实现将领特性系统
//
// ## 架构
//
//   [Service Code] → [Event Bus] → [Trait Handlers]
//                          ↓
//                   按订阅事件分发
//
// ## 添加新特性的步骤
//
//   1. 在 traits/ 目录新建一个文件（如 mynewtrait.go）
//   2. 实现 Trait 接口
//   3. 在 init() 里调用 general.Register(&MyNewTrait{})
//   4. 在 traits/init.go 里 import "_" 触发注册
//   5. GM 后台配置参数
//
// ## 事件类型扩展
//
//   如果新特性需要的事件还不存在，在 context.go 里加新的 Context 类型，
//   在事件分发点（service_combat.go 等）插入 Dispatch 调用。
//
package general

import (
	"sync"
)

// EventType 事件类型常量
const (
	EventBeforeBattle        = "before_battle"
	EventAfterCombatResolve  = "after_combat_resolve"
	EventAfterBattle         = "after_battle"
	EventIncomingAttack      = "incoming_attack"
	EventResourceSettle      = "resource_settle"
	EventBuildingUpgrade     = "building_upgrade"
	EventRecruitComplete     = "recruit_complete"
)

// Params 特性参数（来自 GM 配置）
type Params map[string]float64

func (p Params) Float(key string) float64 {
	return p[key]
}

func (p Params) Int(key string) int {
	return int(p[key])
}

func (p Params) FloatOr(key string, def float64) float64 {
	if v, ok := p[key]; ok {
		return v
	}
	return def
}

func (p Params) IntOr(key string, def int) int {
	if v, ok := p[key]; ok {
		return int(v)
	}
	return def
}

// EventContext 事件上下文（每个事件一个具体类型，实现此接口）
type EventContext interface {
	EventType() string
}

// Trait 将领特性接口
type Trait interface {
	// ID 特性唯一标识，与 GM 配置中的 trait id 对应
	ID() string

	// Name 显示名（中文）
	Name() string

	// Description 玩家可见的描述（参数会动态展示）
	// params 是 GM 配置的当前生效参数
	Description(params Params) string

	// Subscribe 该特性订阅的事件列表
	Subscribe() []EventSubscription

	// ParamSchema 该特性需要的参数定义（GM 后台用来动态生成表单）
	ParamSchema() []ParamField
}

// EventSubscription 一条事件订阅
type EventSubscription struct {
	Event    string                              // 订阅的事件类型
	Priority int                                 // 优先级（多个特性同事件时排序，大的先执行）
	Handle   func(ctx EventContext, params Params)
}

// ParamField 参数 schema（GM 后台动态渲染表单用）
type ParamField struct {
	Key         string  `json:"key"`         // 参数键，如 "captureRate"
	Label       string  `json:"label"`       // 中文显示，如 "俘虏比例"
	Description string  `json:"description"` // 帮助文字
	Default     float64 `json:"default"`     // 默认值
	Min         float64 `json:"min"`         // 最小值
	Max         float64 `json:"max"`         // 最大值
	Step        float64 `json:"step"`        // 步长（GM 后台输入控件用）
}

// --- 注册中心 ---

var (
	registryMu sync.RWMutex
	registry   = map[string]Trait{}
)

// Register 注册一个特性（在特性的 init() 里调用）
func Register(t Trait) {
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := registry[t.ID()]; exists {
		panic("general: trait already registered: " + t.ID())
	}
	registry[t.ID()] = t
}

// Get 根据 id 获取特性实现
func Get(id string) (Trait, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	t, ok := registry[id]
	return t, ok
}

// All 列出所有已注册的特性（GM 后台用，供选择）
func All() []Trait {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]Trait, 0, len(registry))
	for _, t := range registry {
		out = append(out, t)
	}
	return out
}
