package game

import "time"

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
//            "infantryRecruitSpeedBonus", "cavalryRecruitSpeedBonus"
//   速度类:  "buildSpeedBonus", "recruitSpeedBonus", "marchSpeedBonus"
//   经济类:  "exchangeRateBonus"
//   其他:    按需添加，命名规范为 camelCase + "Bonus" 后缀
//
// =============================================================================

// --- StatKey 常量注册表 ---
// 所有合法的加成属性 key。新增 key 时必须在此注册，否则 GM 发放时会被拒绝。

const (
	StatProductionBonus           = "productionBonus"
	StatWoodProductionBonus       = "woodProductionBonus"
	StatStoneProductionBonus      = "stoneProductionBonus"
	StatIronProductionBonus       = "ironProductionBonus"
	StatFoodProductionBonus       = "foodProductionBonus"
	StatCapacityBonus             = "capacityBonus"
	StatAttackBonus               = "attackBonus"
	StatDefenseBonus              = "defenseBonus"
	StatInfantryDefenseBonus      = "infantryDefenseBonus"
	StatCavalryDefenseBonus       = "cavalryDefenseBonus"
	StatInfantryRecruitSpeedBonus = "infantryRecruitSpeedBonus"
	StatCavalryRecruitSpeedBonus  = "cavalryRecruitSpeedBonus"
	StatBuildSpeedBonus           = "buildSpeedBonus"
	StatRecruitSpeedBonus         = "recruitSpeedBonus"
	StatMarchSpeedBonus           = "marchSpeedBonus"
	StatExchangeRateBonus         = "exchangeRateBonus"
)

// ValidStatKeys 所有已注册的合法 key 集合
var ValidStatKeys = map[string]bool{
	StatProductionBonus:           true,
	StatWoodProductionBonus:       true,
	StatStoneProductionBonus:      true,
	StatIronProductionBonus:       true,
	StatFoodProductionBonus:       true,
	StatCapacityBonus:             true,
	StatAttackBonus:               true,
	StatDefenseBonus:              true,
	StatInfantryDefenseBonus:      true,
	StatCavalryDefenseBonus:       true,
	StatInfantryRecruitSpeedBonus: true,
	StatCavalryRecruitSpeedBonus:  true,
	StatBuildSpeedBonus:           true,
	StatRecruitSpeedBonus:         true,
	StatMarchSpeedBonus:           true,
	StatExchangeRateBonus:         true,
}

// IsValidStatKey 校验 key 是否已注册
func IsValidStatKey(key string) bool {
	return ValidStatKeys[key]
}

// IsValidModifierMode 校验 modifier mode 是否为统一管线支持的模式。
func IsValidModifierMode(mode string) bool {
	switch mode {
	case "flat", "percentAdd", "percentMultiply":
		return true
	default:
		return false
	}
}

// Modifier 表示一个属性修改器（所有加成的统一表达）
type Modifier struct {
	Key   string  `json:"key"`   // 属性键名，如 "productionBonus", "attackBonus", "buildSpeedBonus"
	Value float64 `json:"value"` // 数值：flat 模式为绝对值，percent 模式为小数（0.2 = +20%）
	Mode  string  `json:"mode"`  // "flat" | "percentAdd" | "percentMultiply"
}

// ModifierSource 所有能提供加成的来源都实现此接口
type ModifierSource interface {
	Modifiers(now time.Time) []Modifier
	SourceName() string
	// ExpiresAt 返回该来源所有加成的到期时间点（用于时间切片结算）
	// 永久加成返回空切片
	ExpiresAt() []time.Time
}

// ComputeAttribute 根据所有来源计算最终属性值
// 公式：(base + flatSum) × (1 + percentAddSum) × percentMultiply1 × percentMultiply2 × ...
func ComputeAttribute(base float64, key string, sources ...ModifierSource) float64 {
	now := time.Now()
	return ComputeAttributeAt(base, key, now, sources...)
}

// ComputeAttributeAt 指定时间点计算最终属性值
func ComputeAttributeAt(base float64, key string, now time.Time, sources ...ModifierSource) float64 {
	var flatSum float64
	var percentAddSum float64
	multipliers := make([]float64, 0, 4)

	for _, src := range sources {
		if src == nil {
			continue
		}
		for _, mod := range src.Modifiers(now) {
			if mod.Key != key {
				continue
			}
			switch mod.Mode {
			case "flat":
				flatSum += mod.Value
			case "percentAdd":
				percentAddSum += mod.Value
			case "percentMultiply":
				multipliers = append(multipliers, 1+mod.Value)
			}
		}
	}

	result := (base + flatSum) * (1 + percentAddSum)
	for _, m := range multipliers {
		result *= m
	}
	return result
}

// ComputeIntAttribute 整数版本（向下取整）
func ComputeIntAttribute(base int, key string, sources ...ModifierSource) int {
	return int(ComputeAttribute(float64(base), key, sources...))
}

// ComputeIntAttributeAt 整数版本，指定时间点
func ComputeIntAttributeAt(base int, key string, now time.Time, sources ...ModifierSource) int {
	return int(ComputeAttributeAt(float64(base), key, now, sources...))
}

// --- 内置 ModifierSource 实现 ---

// GeneralModifierSource 将领加成来源
type GeneralModifierSource struct {
	General *General
}

func (g *GeneralModifierSource) SourceName() string { return "将领" }

func (g *GeneralModifierSource) ExpiresAt() []time.Time { return nil }

func (g *GeneralModifierSource) Modifiers(now time.Time) []Modifier {
	if g.General == nil || g.General.Buffs == nil {
		return nil
	}
	mods := make([]Modifier, 0, len(g.General.Buffs))
	for key, value := range g.General.Buffs {
		if value == 0 {
			continue
		}
		mods = append(mods, Modifier{
			Key:   key,
			Value: value,
			Mode:  "percentAdd",
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

// --- Buff 通用加成记录（GM 发放 / 活动 / 任务 / 购买 等） ---

// Buff 表示一条加成记录，所有动态加成统一用此结构存储
type Buff struct {
	ID        string  `json:"id"`                  // 唯一标识
	Source    string  `json:"source"`              // 来源："gm", "event", "purchase", "quest", "system"
	Key       string  `json:"key"`                 // 属性键名，如 "productionBonus"
	Value     float64 `json:"value"`               // 数值
	Mode      string  `json:"mode"`                // "flat" | "percentAdd" | "percentMultiply"
	ExpiresAt string  `json:"expiresAt,omitempty"` // 到期时间（空 = 永久）
	CreatedAt string  `json:"createdAt"`           // 创建时间
	Note      string  `json:"note,omitempty"`      // GM 备注
}

// BuffListSource 通用 Buff 列表加成来源
// 从 GameState.Buffs 中读取所有动态 buff，自动过滤过期的
type BuffListSource struct {
	Buffs []Buff
}

func (b *BuffListSource) SourceName() string { return "活动/GM" }

func (b *BuffListSource) ExpiresAt() []time.Time {
	var times []time.Time
	for _, buff := range b.Buffs {
		if buff.ExpiresAt == "" {
			continue
		}
		if t, err := time.Parse(resourceDateLayout, buff.ExpiresAt); err == nil {
			times = append(times, t)
		}
	}
	return times
}

func (b *BuffListSource) Modifiers(now time.Time) []Modifier {
	var mods []Modifier
	for _, buff := range b.Buffs {
		// 检查是否过期
		if buff.ExpiresAt != "" {
			if t, err := time.Parse(resourceDateLayout, buff.ExpiresAt); err == nil && now.After(t) {
				continue // 已过期，跳过
			}
		}
		mods = append(mods, Modifier{
			Key:   buff.Key,
			Value: buff.Value,
			Mode:  buff.Mode,
		})
	}
	return mods
}

// BuildingBonusSource 功能建筑加成来源。
type BuildingBonusSource struct {
	Buildings []Building
}

func (b *BuildingBonusSource) SourceName() string { return "军事建筑" }

func (b *BuildingBonusSource) ExpiresAt() []time.Time { return nil }

func (b *BuildingBonusSource) Modifiers(now time.Time) []Modifier {
	mods := make([]Modifier, 0, len(b.Buildings))
	for _, building := range b.Buildings {
		if building.Level <= 0 {
			continue
		}
		config, exists := getBuildingConfig(building.Type)
		if !exists {
			continue
		}
		for _, mod := range config.ModifiersByLevel[building.Level] {
			if mod.Value == 0 {
				continue
			}
			mods = append(mods, mod)
		}
	}
	return mods
}

// CollectModifierSources 从 GameState 中收集所有当前生效的加成来源
//
// 新增加成来源时，在此函数中 append 即可自动参与所有属性计算。
// 顺序不影响结果（加法交换律），但建议按"永久 → 限时 → 条件"排列便于调试。
func CollectModifierSources(state *GameState) []ModifierSource {
	sources := make([]ModifierSource, 0, 4)

	// 来源1：将领等级/属性加成（永久，随将领成长）
	sources = append(sources, &GeneralModifierSource{General: state.General})

	// 来源2：购买的限时加成（有到期时间，过期自动失效）
	sources = append(sources, &PurchaseBoostSource{
		ProductionBoost:    state.ProductionBoost,
		ProductionBoostEnd: state.ProductionBoostEnd,
		CapacityBoost:      state.CapacityBoost,
		CapacityBoostEnd:   state.CapacityBoostEnd,
	})

	// 来源3：通用 Buff 列表（GM 发放 / 活动 / 任务奖励等）
	if len(state.Buffs) > 0 {
		sources = append(sources, &BuffListSource{Buffs: state.Buffs})
	}

	// 来源4：军事/功能建筑加成（按建筑等级永久生效）
	if len(state.Buildings) > 0 {
		sources = append(sources, &BuildingBonusSource{Buildings: state.Buildings})
	}

	// --- 后续扩展点（按需取消注释或新增） ---
	// 来源3：装备加成（穿上生效，脱下失效）
	// sources = append(sources, &EquipmentModifierSource{Equipment: state.Equipment})
	//
	// 来源4：战术卡（战斗前选择，战斗期间生效）
	// sources = append(sources, &TacticCardSource{Cards: state.TacticCards})
	//
	// 来源5：联盟科技（联盟全员共享，永久生效）
	// sources = append(sources, &AllianceTechSource{Tech: state.AllianceTech})
	//
	// 来源7：活动 buff（运营配置的限时全服加成）
	// sources = append(sources, &EventBuffSource{Events: activeEvents})
	//
	// 来源8：VIP 等级（账户级永久加成）
	// sources = append(sources, &VIPSource{Level: account.VIPLevel})
	//
	// 来源9：阵营特性（从配置读取，永久生效）
	// sources = append(sources, &FactionTraitSource{Faction: state.Player.Faction})

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
