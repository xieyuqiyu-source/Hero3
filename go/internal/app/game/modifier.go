package game

import (
	"errors"
	"strings"
	"sync"
	"time"

	coremodifier "hero3/internal/core/modifier"
)

// =============================================================================
// Modifier 管线 — 统一加成系统
// =============================================================================
//
// 所有影响数值的加成（将领、装备、购买加成、活动 buff、阵营特性、建筑效果、
// 战术卡、联盟科技、VIP 等）都通过此管线统一计算。
//
// ## 核心公式
//
//   最终值 = (基础值 + flat加成之和) × (1 + percentAdd加成之和) × percentMultiply1 × percentMultiply2 × ...
//
// 三层叠加：
//   - flat:            绝对值加成（如 +200 攻击力）
//   - percentAdd:      百分比加法（如 +20% 产量），所有来源先求和再乘，线性增长可控
//   - percentMultiply: 百分比乘法（如 ×4 产量加成），各自独立相乘，用于强力限时加成
//
// ## 如何新增一个加成来源
//
// 1. 定义一个 struct，实现 ModifierSource 接口：
//
//     type MyNewSource struct { ... }
//
//     func (s *MyNewSource) Modifiers(now time.Time) []Modifier {
//         // 根据自身状态返回当前生效的 Modifier 列表
//         // 如果有时效性，检查 now 是否在有效期内
//         return []Modifier{
//             {Key: "productionBonus", Value: 0.2, Mode: "percentAdd"},  // +20% 产量
//             {Key: "attackBonus", Value: 100, Mode: "flat"},            // +100 攻击
//         }
//     }
//
// 2. 在 CollectModifierSources 函数中注册：
//
//     sources = append(sources, &MyNewSource{...})
//
// 3. 完成。无需修改计算公式或结算逻辑。
//
// ## 已支持的 Modifier Key（可自由扩展，key 是字符串无需预注册）
//
//   产量类:  "productionBonus", "woodProductionBonus", "stoneProductionBonus" 等
//   容量类:  "capacityBonus"
//   军事类:  "attackBonus", "defenseBonus", "infantryDefenseBonus", "cavalryDefenseBonus"
//            "infantryRecruitSpeedBonus", "cavalryRecruitSpeedBonus",
//            "siegeRecruitSpeedBonus", "specialRecruitSpeedBonus",
//            "recruitCostReduction", "siegeRecruitCostReduction", "specialRecruitCostReduction"
//   速度类:  "buildSpeedBonus", "recruitSpeedBonus", "marchSpeedBonus"
//   经济类:  "exchangeRateBonus"
//   其他:    按需添加，命名规范为 camelCase + "Bonus" 后缀
//
// =============================================================================

// --- StatKey 常量注册表 ---
// 所有合法的加成属性 key。新增 key 时必须在此注册，否则 GM 发放时会被拒绝。

const (
	StatProductionBonus             = "productionBonus"
	StatWoodProductionBonus         = "woodProductionBonus"
	StatStoneProductionBonus        = "stoneProductionBonus"
	StatIronProductionBonus         = "ironProductionBonus"
	StatFoodProductionBonus         = "foodProductionBonus"
	StatCapacityBonus               = "capacityBonus"
	StatAttackBonus                 = "attackBonus"
	StatDefenseBonus                = "defenseBonus"
	StatInfantryDefenseBonus        = "infantryDefenseBonus"
	StatCavalryDefenseBonus         = "cavalryDefenseBonus"
	StatInfantryRecruitSpeedBonus   = "infantryRecruitSpeedBonus"
	StatCavalryRecruitSpeedBonus    = "cavalryRecruitSpeedBonus"
	StatSiegeRecruitSpeedBonus      = "siegeRecruitSpeedBonus"
	StatSpecialRecruitSpeedBonus    = "specialRecruitSpeedBonus"
	StatBuildSpeedBonus             = "buildSpeedBonus"
	StatRecruitSpeedBonus           = "recruitSpeedBonus"
	StatRecruitCostReduction        = "recruitCostReduction"
	StatSiegeRecruitCostReduction   = "siegeRecruitCostReduction"
	StatSpecialRecruitCostReduction = "specialRecruitCostReduction"
	StatMarchSpeedBonus             = "marchSpeedBonus"
	StatEnemyLossRevealThreshold    = "enemyLossRevealThresholdBonus"
	StatExchangeRateBonus           = "exchangeRateBonus"
)

// ValidStatKeys 所有已注册的合法 key 集合
var ValidStatKeys = map[string]bool{
	StatProductionBonus:             true,
	StatWoodProductionBonus:         true,
	StatStoneProductionBonus:        true,
	StatIronProductionBonus:         true,
	StatFoodProductionBonus:         true,
	StatCapacityBonus:               true,
	StatAttackBonus:                 true,
	StatDefenseBonus:                true,
	StatInfantryDefenseBonus:        true,
	StatCavalryDefenseBonus:         true,
	StatInfantryRecruitSpeedBonus:   true,
	StatCavalryRecruitSpeedBonus:    true,
	StatSiegeRecruitSpeedBonus:      true,
	StatSpecialRecruitSpeedBonus:    true,
	StatBuildSpeedBonus:             true,
	StatRecruitSpeedBonus:           true,
	StatRecruitCostReduction:        true,
	StatSiegeRecruitCostReduction:   true,
	StatSpecialRecruitCostReduction: true,
	StatMarchSpeedBonus:             true,
	StatEnemyLossRevealThreshold:    true,
	StatExchangeRateBonus:           true,
}

// IsValidStatKey 校验 key 是否已注册
func IsValidStatKey(key string) bool {
	return ValidStatKeys[key]
}

// IsValidModifierMode 校验 modifier mode 是否为统一管线支持的模式。
func IsValidModifierMode(mode string) bool {
	return coremodifier.IsValidMode(mode)
}

// Modifier 表示一个属性修改器（所有加成的统一表达）
type Modifier = coremodifier.Modifier

// ModifierSource 所有能提供加成的来源都实现此接口
type ModifierSource = coremodifier.Source

type modifierSourceProvider struct {
	name     string
	provider ModifierSourceProvider
}

type ModifierSourceProvider func(state *GameState) []ModifierSource

var (
	modifierProvidersMu sync.RWMutex
	modifierProviders   []modifierSourceProvider
)

func RegisterModifierSourceProvider(name string, provider ModifierSourceProvider) error {
	name = strings.TrimSpace(name)
	if name == "" || provider == nil {
		return errors.New("modifier source provider name and handler are required")
	}
	modifierProvidersMu.Lock()
	defer modifierProvidersMu.Unlock()
	for _, item := range modifierProviders {
		if item.name == name {
			return errors.New("modifier source provider already registered: " + name)
		}
	}
	modifierProviders = append(modifierProviders, modifierSourceProvider{name: name, provider: provider})
	return nil
}

// ComputeAttribute 根据所有来源计算最终属性值
// 公式：(base + flatSum) × (1 + percentAddSum) × percentMultiply1 × percentMultiply2 × ...
func ComputeAttribute(base float64, key string, sources ...ModifierSource) float64 {
	return coremodifier.ComputeAttribute(base, key, sources...)
}

// ComputeAttributeAt 指定时间点计算最终属性值
func ComputeAttributeAt(base float64, key string, now time.Time, sources ...ModifierSource) float64 {
	return coremodifier.ComputeAttributeAt(base, key, now, sources...)
}

// ComputeIntAttribute 整数版本（向下取整）
func ComputeIntAttribute(base int, key string, sources ...ModifierSource) int {
	return coremodifier.ComputeIntAttribute(base, key, sources...)
}

// ComputeIntAttributeAt 整数版本，指定时间点
func ComputeIntAttributeAt(base int, key string, now time.Time, sources ...ModifierSource) int {
	return coremodifier.ComputeIntAttributeAt(base, key, now, sources...)
}

// --- 内置 ModifierSource 实现 ---

// FactionTraitModifierSource 阵营特性来源。
type FactionTraitModifierSource struct {
	Faction string
}

func (f *FactionTraitModifierSource) SourceName() string { return "阵营特性" }

func (f *FactionTraitModifierSource) ExpiresAt() []time.Time { return nil }

func (f *FactionTraitModifierSource) Modifiers(now time.Time) []Modifier {
	faction := strings.TrimSpace(f.Faction)
	if faction == "" {
		return nil
	}
	cfg, ok := GetFactionsConfig()[faction]
	if !ok || len(cfg.Traits) == 0 {
		return nil
	}
	mods := make([]Modifier, 0, len(cfg.Traits))
	for key, value := range cfg.Traits {
		if value == 0 {
			continue
		}
		mods = append(mods, Modifier{
			Key:   key,
			Value: value - 1,
			Mode:  "percentMultiply",
		})
	}
	return mods
}

// PurchaseBoostSource 购买的限时加成来源（产量 + 容量）
type PurchaseBoostSource struct {
	ProductionBoost    int
	ProductionBoostEnd string
	CapacityBoost      int
	CapacityBoostEnd   string
}

func (p *PurchaseBoostSource) SourceName() string { return "购买加成" }

func (p *PurchaseBoostSource) ExpiresAt() []time.Time {
	var times []time.Time
	if p.ProductionBoostEnd != "" {
		if t, err := time.Parse(resourceDateLayout, p.ProductionBoostEnd); err == nil {
			times = append(times, t)
		}
	}
	if p.CapacityBoostEnd != "" {
		if t, err := time.Parse(resourceDateLayout, p.CapacityBoostEnd); err == nil {
			times = append(times, t)
		}
	}
	return times
}

func (p *PurchaseBoostSource) Modifiers(now time.Time) []Modifier {
	var mods []Modifier

	// 产量加成：倍率 N 表示 ×N，转为 percentAdd = N-1
	if p.ProductionBoost > 1 && p.ProductionBoostEnd != "" {
		if expiresAt, err := time.Parse(resourceDateLayout, p.ProductionBoostEnd); err == nil && now.Before(expiresAt) {
			mods = append(mods, Modifier{
				Key:   "productionBonus",
				Value: float64(p.ProductionBoost - 1),
				Mode:  "percentMultiply",
			})
		}
	}

	// 容量加成：倍率 N 表示 ×N，转为 percentAdd = N-1
	if p.CapacityBoost > 1 && p.CapacityBoostEnd != "" {
		if expiresAt, err := time.Parse(resourceDateLayout, p.CapacityBoostEnd); err == nil && now.Before(expiresAt) {
			mods = append(mods, Modifier{
				Key:   "capacityBonus",
				Value: float64(p.CapacityBoost - 1),
				Mode:  "percentMultiply",
			})
		}
	}

	return mods
}

// ModifierBreakdownItem 加成明细条目（用于前端展示）
type ModifierBreakdownItem struct {
	Source string  `json:"source"` // 来源名称，如 "将领", "产量加成(购买)"
	Key    string  `json:"key"`    // 属性键名
	Value  float64 `json:"value"`  // 数值
	Mode   string  `json:"mode"`   // "flat" | "percentAdd" | "percentMultiply"
}

// StaticModifierSource 静态加成来源（阵营特性、配置等不随时间变化的）
type StaticModifierSource struct {
	Name string
	Mods []Modifier
}

func (s *StaticModifierSource) SourceName() string { return s.Name }

func (s *StaticModifierSource) ExpiresAt() []time.Time { return nil }

func (s *StaticModifierSource) Modifiers(now time.Time) []Modifier {
	return s.Mods
}

func init() {
	_ = RegisterModifierSourceProvider("general", func(state *GameState) []ModifierSource {
		return []ModifierSource{&GeneralModifierSource{General: state.General}}
	})
	_ = RegisterModifierSourceProvider("purchase_boost", func(state *GameState) []ModifierSource {
		return []ModifierSource{&PurchaseBoostSource{
			ProductionBoost:    state.ProductionBoost,
			ProductionBoostEnd: state.ProductionBoostEnd,
			CapacityBoost:      state.CapacityBoost,
			CapacityBoostEnd:   state.CapacityBoostEnd,
		}}
	})
	_ = RegisterModifierSourceProvider("buff_list", func(state *GameState) []ModifierSource {
		if len(state.Buffs) == 0 {
			return nil
		}
		return []ModifierSource{&BuffListSource{Buffs: state.Buffs}}
	})
	_ = RegisterModifierSourceProvider("building_bonus", func(state *GameState) []ModifierSource {
		if len(state.Buildings) == 0 {
			return nil
		}
		return []ModifierSource{&BuildingBonusSource{Buildings: state.Buildings}}
	})
	_ = RegisterModifierSourceProvider("faction_trait", func(state *GameState) []ModifierSource {
		if state == nil {
			return nil
		}
		if strings.TrimSpace(state.Player.Faction) == "" {
			return nil
		}
		return []ModifierSource{&FactionTraitModifierSource{Faction: state.Player.Faction}}
	})
}

// CollectModifierSources 从 GameState 中收集所有当前生效的加成来源
//
// 新增加成来源时，在此函数中 append 即可自动参与所有属性计算。
// 顺序不影响结果（加法交换律），但建议按"永久 → 限时 → 条件"排列便于调试。
func CollectModifierSources(state *GameState) []ModifierSource {
	sources := make([]ModifierSource, 0, 6)
	modifierProvidersMu.RLock()
	providers := append([]modifierSourceProvider(nil), modifierProviders...)
	modifierProvidersMu.RUnlock()
	for _, item := range providers {
		if item.provider == nil {
			continue
		}
		sources = append(sources, item.provider(state)...)
	}
	return sources
}

// GetModifierBreakdown 获取当前所有生效的加成明细（用于前端 tooltip 展示）
func GetModifierBreakdown(state *GameState, now time.Time) []ModifierBreakdownItem {
	sources := CollectModifierSources(state)
	var items []ModifierBreakdownItem

	for _, src := range sources {
		if src == nil {
			continue
		}
		mods := src.Modifiers(now)
		if len(mods) == 0 {
			continue
		}
		sourceName := src.SourceName()
		for _, mod := range mods {
			items = append(items, ModifierBreakdownItem{
				Source: sourceName,
				Key:    mod.Key,
				Value:  mod.Value,
				Mode:   mod.Mode,
			})
		}
	}

	return items
}
