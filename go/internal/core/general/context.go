package general

import (
	"hero3/internal/core/combat"
)

// TraitOutcome 特性触发后的结果详情（写入战报供前端展示）
type TraitOutcome struct {
	TraitID        string                 `json:"traitId"`                  // 特性 ID
	Name           string                 `json:"name,omitempty"`           // 中文名（冗余便于前端）
	TraitType      string                 `json:"traitType,omitempty"`      // 特性类型
	OwnerSide      string                 `json:"ownerSide,omitempty"`      // 触发方：attacker/defender/reinforcement
	OwnerGeneralID string                 `json:"ownerGeneralId,omitempty"` // 触发武将
	OwnerPlayerID  string                 `json:"ownerPlayerId,omitempty"`  // 触发玩家
	Scope          string                 `json:"scope,omitempty"`          // 作用范围
	Detail         map[string]interface{} `json:"detail,omitempty"`         // 关键数据（俘虏数、复活数、伤害量等）
}

// TraitActor 描述当前事件中某条特性的触发归属。
type TraitActor struct {
	Side            string // attacker / defender / reinforcement
	PlayerID        string
	GeneralID       string
	ReinforcementID string
	Scope           string
	TargetUnitType  string
}

// BeforeBattleContext 战斗开始前的上下文
//
// 特性可以在这里：
// - 修改双方军队组成（俘虏一部分敌方兵）
// - 添加/移除我方/敌方兵种
//
// 修改 Attacker.Units 和 Defender.Units 会影响真正的战斗计算
type BeforeBattleContext struct {
	Attacker          *combat.Army // 进攻方军队（可改）
	Defender          *combat.Army // 防守方军队（可改）
	AttackerOwnsTrait bool         // 进攻方是否拥有当前特性
	DefenderOwnsTrait bool         // 防守方是否拥有当前特性
	Actor             TraitActor   // 当前分发批次的特性归属
	Scene             string       // attack / plunder / reinforcement_defense / npc 等

	// 输出：俘虏到我方的兵（key: unitType → count）
	// 主城兵进入军队，跨阵营进入驻防
	CapturedToArmy     map[string]int
	CapturedToGarrison map[string]int

	// 元信息
	IsPvP       bool
	SameFaction bool // 同阵营（PvE 时按 SameFaction = true 处理俘虏入军队）

	// Triggered 特性自己写入（实际触发后写）
	Triggered map[string]TraitOutcome
}

func (c *BeforeBattleContext) EventType() string { return EventBeforeBattle }

// AfterCombatResolveContext 战斗计算之后、战报生成之前的上下文
//
// 特性可以在这里：
// - 修改 CombatResult.AttackerLosses / DefenderLosses（增减损失）
// - 修改 LossRate（影响掠夺/俘虏计算）
//
// 触发顺序：buildCombatArmy → combat.Resolve → 此事件 → applyBattleResult
type AfterCombatResolveContext struct {
	Result            *combat.CombatResult // 战斗结果（可改）
	Attacker          *combat.Army         // 进攻方
	Defender          *combat.Army         // 防守方
	AttackerOwnsTrait bool
	DefenderOwnsTrait bool
	Actor             TraitActor
	Scene             string
	DisabledTraitSide map[string]int // 已被压制的阵营特性数量，key: attacker/defender/reinforcement

	IsAttackerOnly bool // 是否只在进攻方触发（如周瑜火攻）

	// Triggered 由特性自己写入（特性逻辑实际触发后才写）
	// key: trait id, value: 触发详情（供战报展示）
	Triggered map[string]TraitOutcome
}

func (c *AfterCombatResolveContext) EventType() string { return EventAfterCombatResolve }

// AfterBattleContext 战斗完成、状态写入后的上下文
//
// 特性可以在这里：
// - 修改玩家军队（如刘备复活损失的兵）
// - 给战报附加额外信息
//
// 注意：此时 PlayerLosses 已经从军队扣除完成，复活就是把损失加回军队
type AfterBattleContext struct {
	PlayerArmy   map[string]int // 玩家当前军队（key: unitType → amount，可改）
	PlayerLosses map[string]int // 玩家本场损失（按 unitType）
	IsAttacker   bool           // 该玩家是进攻方还是防守方
	Won          bool           // 该玩家是否胜利
	Actor        TraitActor     // 当前分发批次的特性归属
	Scene        string         // attack / defense / reinforcement_defense 等

	// 输出：复活的兵（key: unitType → count），调用方应用回 state.Army
	Revived map[string]int

	// Triggered 特性自己写入（实际触发后写）
	Triggered map[string]TraitOutcome
}

func (c *AfterBattleContext) EventType() string { return EventAfterBattle }

// MarchCreateContext 行军创建前的上下文，特性可修改本次行军秒数。
type MarchCreateContext struct {
	BaseSeconds  int
	FinalSeconds int
	Scene        string
	Actor        TraitActor
	Triggered    map[string]TraitOutcome
}

func (c *MarchCreateContext) EventType() string { return EventMarchCreate }

// RecruitCostContext 征兵消耗计算上下文，特性可降低本次资源消耗。
type RecruitCostContext struct {
	UnitType  string
	Category  string
	Amount    int
	Cost      map[string]int
	Actor     TraitActor
	Triggered map[string]TraitOutcome
}

func (c *RecruitCostContext) EventType() string { return EventRecruitCost }

// PlunderResolveContext 掠夺结算上下文，特性可修正本次掠夺收益。
type PlunderResolveContext struct {
	Rewards   map[string]int
	Scene     string
	Actor     TraitActor
	Triggered map[string]TraitOutcome
}

func (c *PlunderResolveContext) EventType() string { return EventPlunderResolve }
