// 本文件负责将领配置的加载、归一化、复制和安全校验。
package game

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"hero3/internal/core/general"
	_ "hero3/internal/core/general/traits"
)

const maxGeneralAttributeBonus = 10.0

// GeneralsConfig 将领系统总配置（GM 后台可编辑）
type GeneralsConfig struct {
	Common  GeneralsCommonConfig         `json:"common"`  // 通用配置（顶部）
	Heroes  map[string]GeneralHeroConfig `json:"heroes"`  // 单将领配置 map[generalId]
	Enabled bool                         `json:"enabled"` // 全局开关
}

// GeneralsCommonConfig 通用配置（所有将领共享）
type GeneralsCommonConfig struct {
	ExpCurve   []int                      `json:"expCurve"`   // 每级所需经验
	LevelBuffs map[int]map[string]float64 `json:"levelBuffs"` // 每级提供的通用 buff
}

// GeneralHeroConfig 单将领配置
type GeneralHeroConfig struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Faction string `json:"faction"`
	Title   string `json:"title"`
	Rarity  string `json:"rarity"`  // common | rare | epic | legendary
	Enabled bool   `json:"enabled"` // 该将领是否启用

	// 数值加成（叠加在通用 levelBuffs 之上）
	Buffs map[string]float64 `json:"buffs"`

	// 特性列表（特性 id 在代码注册中心查到，参数从这里读）
	Traits       []GeneralTraitConfig `json:"traits,omitempty"`
	SpecialTrait GeneralTraitConfig   `json:"specialTrait"`
	BonusTrait   GeneralTraitConfig   `json:"bonusTrait"`
}

// GeneralTraitConfig 单将领的某个特性的配置
type GeneralTraitConfig struct {
	TraitID         string             `json:"traitId"`                   // 对应 general.traits 注册的 id（如 "meiren"）
	TraitType       string             `json:"traitType,omitempty"`       // special / bonus
	Enabled         bool               `json:"enabled"`                   // 该特性是否启用
	Scope           string             `json:"scope,omitempty"`           // 作用范围
	TargetUnitType  string             `json:"targetUnitType,omitempty"`  // 目标兵种或兵种分类
	AllowedSides    []string           `json:"allowedSides,omitempty"`    // 允许触发方：attacker / defender / reinforcement
	AllowedScenes   []string           `json:"allowedScenes,omitempty"`   // 允许玩法场景：attack / plunder 等
	RequiredOutcome string             `json:"requiredOutcome,omitempty"` // 胜负条件：win / loss
	Params          map[string]float64 `json:"params"`                    // 当前参数（覆盖默认值）
}

// --- 全局管理 ---

var (
	generalsMu     sync.RWMutex
	activeGenerals = defaultGeneralsConfig()
)

func GetGeneralsConfig() GeneralsConfig {
	generalsMu.RLock()
	defer generalsMu.RUnlock()
	return cloneGeneralsConfig(activeGenerals)
}

func SetGeneralsConfig(cfg GeneralsConfig) error {
	cfg = NormalizeGeneralsConfig(cfg)
	// 校验配置一致性
	if err := ValidateGeneralsConfig(cfg); err != nil {
		return err
	}
	generalsMu.Lock()
	activeGenerals = cloneGeneralsConfig(cfg)
	generalsMu.Unlock()
	return nil
}

func LoadGeneralsConfig(path string) error {
	if path == "" {
		return nil
	}
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return SaveGeneralsConfig(path, GetGeneralsConfig())
		}
		return err
	}
	var cfg GeneralsConfig
	if err := json.Unmarshal(content, &cfg); err != nil {
		return err
	}
	return SetGeneralsConfig(cfg)
}

func SaveGeneralsConfig(path string, cfg GeneralsConfig) error {
	cfg = NormalizeGeneralsConfig(cfg)
	if err := SetGeneralsConfig(cfg); err != nil {
		return err
	}
	if path == "" {
		return nil
	}
	content, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(content, '\n'), 0o644)
}

// NormalizeGeneralsConfig 将旧 traits 数组兼容归一为双特性结构。
func NormalizeGeneralsConfig(cfg GeneralsConfig) GeneralsConfig {
	for id, hero := range cfg.Heroes {
		normalizeHeroDualTraits(&hero)
		normalizeTraitConfigParams(&hero.SpecialTrait)
		normalizeTraitConfigParams(&hero.BonusTrait)
		if strings.TrimSpace(hero.SpecialTrait.TraitID) != "" {
			if trait, ok := general.Get(hero.SpecialTrait.TraitID); ok {
				hero.SpecialTrait.TraitType = trait.Type()
			}
		}
		if strings.TrimSpace(hero.BonusTrait.TraitID) != "" {
			if trait, ok := general.Get(hero.BonusTrait.TraitID); ok {
				hero.BonusTrait.TraitType = trait.Type()
			}
		}
		cfg.Heroes[id] = hero
	}
	return cfg
}

// normalizeTraitConfigParams 清理已经废止的历史特性参数，避免旧配置继续影响当前规则。
func normalizeTraitConfigParams(traitCfg *GeneralTraitConfig) {
	if traitCfg == nil || traitCfg.Params == nil {
		return
	}
	switch traitCfg.TraitID {
	case "weiwu_haoling":
		delete(traitCfg.Params, "maxGuardPerDay")
		delete(traitCfg.Params, "maxGuardPerSettle")
	case "meiren":
		_, hadCaptureRate := traitCfg.Params["captureRate"]
		_, hadCaptureMax := traitCfg.Params["captureMax"]
		_, hadMaxCapturePerUnit := traitCfg.Params["maxCapturePerUnit"]
		delete(traitCfg.Params, "captureRate")
		delete(traitCfg.Params, "captureMax")
		delete(traitCfg.Params, "maxCapturePerUnit")
		if hadCaptureRate || hadCaptureMax || hadMaxCapturePerUnit {
			traitCfg.Params["attackBonusRate"] = 0.25
			traitCfg.Params["triggerChance"] = 0.5
		}
	case "huchi_chongzhen":
		// 旧版 35% 概率降低 20% 防御迁移为新版 50% 概率降低 30%。
		if chance, ok := traitCfg.Params["triggerChance"]; !ok || math.Abs(chance-0.35) < 1e-9 {
			traitCfg.Params["triggerChance"] = 0.5
		}
		if rate, ok := traitCfg.Params["enemyDefenseReductionRate"]; !ok || math.Abs(rate-0.2) < 1e-9 {
			traitCfg.Params["enemyDefenseReductionRate"] = 0.3
		}
		traitCfg.Scope = "enemy_army"
		traitCfg.TargetUnitType = ""
		traitCfg.AllowedSides = []string{"attacker"}
	case "pojun_pofang":
		// 许褚旧固定破防能力废止，存量配置统一迁移为虎豹骑永久固定属性。
		traitCfg.TraitID = "huhu_shengwei"
		traitCfg.Scope = "self_army"
		traitCfg.TargetUnitType = "huBaoQi"
		traitCfg.AllowedSides = nil
		traitCfg.AllowedScenes = nil
		traitCfg.RequiredOutcome = ""
		traitCfg.Params = map[string]float64{"unitAttackFlat": 12, "unitSpeedFlat": 5}
	case "huhu_shengwei":
		// 现行虎虎生威保留 GM 已配置固定值，并补足缺失参数与作用目标。
		delete(traitCfg.Params, "triggerChance")
		delete(traitCfg.Params, "enemyDefenseReductionRate")
		if _, ok := traitCfg.Params["unitAttackFlat"]; !ok {
			traitCfg.Params["unitAttackFlat"] = 12
		}
		if _, ok := traitCfg.Params["unitSpeedFlat"]; !ok {
			traitCfg.Params["unitSpeedFlat"] = 5
		}
		traitCfg.Scope = "self_army"
		traitCfg.TargetUnitType = "huBaoQi"
		traitCfg.AllowedSides = nil
		traitCfg.AllowedScenes = nil
		traitCfg.RequiredOutcome = ""
	case "huzhu_sizhan":
		// 典韦旧战败返兵能力废止，存量配置统一迁移为守城/增援禁卫甲士固定加防。
		traitCfg.TraitID = "huzhu_xuezhan"
		traitCfg.Scope = "self_army"
		traitCfg.TargetUnitType = "jinWeiSoldier"
		traitCfg.AllowedSides = []string{"defender", "reinforcement"}
		traitCfg.AllowedScenes = nil
		traitCfg.RequiredOutcome = ""
		traitCfg.Params = map[string]float64{"triggerChance": 1, "generalDefenseFlat": 20}
	case "huzhu_xuezhan":
		// 现行护主血战保留 GM 已配置固定防御值，并清除旧战败返兵参数。
		delete(traitCfg.Params, "lossReductionRate")
		delete(traitCfg.Params, "maxReturnCount")
		if _, ok := traitCfg.Params["generalDefenseFlat"]; !ok {
			traitCfg.Params["generalDefenseFlat"] = 20
		}
		if _, ok := traitCfg.Params["triggerChance"]; !ok {
			traitCfg.Params["triggerChance"] = 1
		}
		traitCfg.Scope = "self_army"
		traitCfg.TargetUnitType = "jinWeiSoldier"
		traitCfg.AllowedSides = []string{"defender", "reinforcement"}
		traitCfg.AllowedScenes = nil
		traitCfg.RequiredOutcome = ""
	case "sizhandaodi":
		// 旧版必定步兵加攻迁移为默认 60% 概率，保留 GM 已显式配置的新概率和比例。
		if _, ok := traitCfg.Params["attackBonusRate"]; !ok {
			traitCfg.Params["attackBonusRate"] = 0.35
		}
		if _, ok := traitCfg.Params["triggerChance"]; !ok {
			traitCfg.Params["triggerChance"] = 0.6
		}
		traitCfg.Scope = "self_army"
		traitCfg.TargetUnitType = "infantry"
		traitCfg.AllowedSides = []string{"attacker"}
		traitCfg.AllowedScenes = nil
		traitCfg.RequiredOutcome = ""
	case "yibing_touxi":
		// 旧上限与伤亡比例重复，移除后由 GM 的 effectRate 直接决定真实减员。
		delete(traitCfg.Params, "maxAffectedRate")
		traitCfg.Scope = "enemy_army"
		traitCfg.AllowedSides = nil
	case "mouding_houfa":
		// 旧版必定减攻 10% 自动迁移为新版概率全军加防，避免历史 GM 配置校验失败。
		delete(traitCfg.Params, "effectRate")
		delete(traitCfg.Params, "attackReductionRate")
		if _, ok := traitCfg.Params["defenseBonusRate"]; !ok {
			traitCfg.Params["defenseBonusRate"] = 0.35
		}
		if _, ok := traitCfg.Params["triggerChance"]; !ok {
			traitCfg.Params["triggerChance"] = 0.35
		}
		traitCfg.Scope = "self_army"
		traitCfg.AllowedSides = []string{"defender", "reinforcement"}
	case "jixing_benxi":
		// 旧版全军百分比行军加速迁移为骁骑营永久攻击与移动属性。
		_, hadOldSpeedRate := traitCfg.Params["speedBonusRate"]
		_, hadOldMinimum := traitCfg.Params["minMarchSeconds"]
		delete(traitCfg.Params, "speedBonusRate")
		delete(traitCfg.Params, "minMarchSeconds")
		delete(traitCfg.Params, "triggerChance")
		if hadOldSpeedRate || hadOldMinimum {
			traitCfg.Params["unitAttackFlat"] = 18
			traitCfg.Params["unitSpeedFlat"] = 5
		}
		traitCfg.Scope = "self_army"
		traitCfg.TargetUnitType = "qiQiYing"
		traitCfg.AllowedSides = nil
	case "dunzhen_fangyu":
		// 旧版必定增加 35% 防御迁移为新版 60% 概率增加 30%。
		_, hadTriggerChance := traitCfg.Params["triggerChance"]
		if rate, ok := traitCfg.Params["defenseBonusRate"]; !ok || (!hadTriggerChance && math.Abs(rate-0.35) < 1e-9) {
			traitCfg.Params["defenseBonusRate"] = 0.3
		}
		if !hadTriggerChance {
			traitCfg.Params["triggerChance"] = 0.6
		}
		traitCfg.Scope = "self_army"
		traitCfg.AllowedSides = []string{"defender", "reinforcement"}
	case "weizhen_zhenhe":
		// 旧版 20% 全方向临时压制迁移为主动进攻 25% 溃逃，效果比例本身即可表达上限。
		delete(traitCfg.Params, "maxAffectedRate")
		delete(traitCfg.Params, "maxAffectedCount")
		if rate, ok := traitCfg.Params["effectRate"]; !ok || math.Abs(rate-0.2) < 1e-9 {
			traitCfg.Params["effectRate"] = 0.25
		}
		if _, ok := traitCfg.Params["triggerChance"]; !ok {
			traitCfg.Params["triggerChance"] = 0.35
		}
		traitCfg.Scope = "enemy_army"
		traitCfg.AllowedSides = []string{"attacker"}
	case "weizhen_xiaoyao":
		// 旧版必定骑兵加攻迁移为默认 60% 概率，保留 GM 已显式设置的新概率。
		if _, ok := traitCfg.Params["attackBonusRate"]; !ok {
			traitCfg.Params["attackBonusRate"] = 0.35
		}
		if _, ok := traitCfg.Params["triggerChance"]; !ok {
			traitCfg.Params["triggerChance"] = 0.6
		}
		traitCfg.Scope = "self_army"
		traitCfg.TargetUnitType = "cavalry"
		traitCfg.AllowedSides = []string{"attacker"}
	}
}

// --- 默认配置 ---

func defaultGeneralsConfig() GeneralsConfig {
	return GeneralsConfig{
		Enabled: true,
		Common: GeneralsCommonConfig{
			ExpCurve: []int{0, 100, 300, 600, 1000, 1500, 2200, 3000, 4000, 5500, 7500},
			LevelBuffs: map[int]map[string]float64{
				1:  {},
				2:  {"productionBonus": 0.02},
				3:  {"productionBonus": 0.04, "attackBonus": 0.02},
				5:  {"productionBonus": 0.08, "attackBonus": 0.05, "defenseBonus": 0.03},
				10: {"productionBonus": 0.15, "attackBonus": 0.12, "defenseBonus": 0.10},
			},
		},
		Heroes: map[string]GeneralHeroConfig{
			"zhenmi": {
				ID: "zhenmi", Name: "甄宓", Faction: "wei", Title: "美人", Rarity: "epic", Enabled: true,
				SpecialTrait: GeneralTraitConfig{
					TraitID: "meiren", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "self_army", AllowedSides: []string{"attacker"},
					Params: map[string]float64{"attackBonusRate": 0.25, "triggerChance": 0.5},
				},
				BonusTrait: GeneralTraitConfig{
					TraitID: "meihuo_raozhen", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "enemy_army", AllowedSides: []string{"attacker"},
					Params: map[string]float64{"enemyDefenseReductionRate": 0.25, "triggerChance": 0.5},
				},
			},
			"zhouyu": {
				ID: "zhouyu", Name: "周瑜", Faction: "wu", Title: "美周郎", Rarity: "epic", Enabled: true,
				Buffs: map[string]float64{"attackBonus": 0.15},
				Traits: []GeneralTraitConfig{
					{
						TraitID: "huogong", Enabled: true,
						Params: map[string]float64{"damagePercent": 0.15, "triggerChance": 0.6},
					},
				},
			},
			"liubei": {
				ID: "liubei", Name: "刘备", Faction: "shu", Title: "仁主", Rarity: "epic", Enabled: true,
				Buffs: map[string]float64{"defenseBonus": 0.10, "productionBonus": 0.05},
				Traits: []GeneralTraitConfig{
					{
						TraitID: "rende", Enabled: true,
						Params: map[string]float64{"reviveRate": 0.2, "triggerChance": 0.5},
					},
				},
			},
		},
	}
}

func cloneGeneralsConfig(src GeneralsConfig) GeneralsConfig {
	dst := GeneralsConfig{
		Enabled: src.Enabled,
		Common: GeneralsCommonConfig{
			ExpCurve:   append([]int(nil), src.Common.ExpCurve...),
			LevelBuffs: make(map[int]map[string]float64, len(src.Common.LevelBuffs)),
		},
		Heroes: make(map[string]GeneralHeroConfig, len(src.Heroes)),
	}
	for level, buffs := range src.Common.LevelBuffs {
		clone := make(map[string]float64, len(buffs))
		for k, v := range buffs {
			clone[k] = v
		}
		dst.Common.LevelBuffs[level] = clone
	}
	for id, hero := range src.Heroes {
		dst.Heroes[id] = cloneHeroConfig(hero)
	}
	return dst
}

func cloneHeroConfig(src GeneralHeroConfig) GeneralHeroConfig {
	dst := src
	dst.Buffs = make(map[string]float64, len(src.Buffs))
	for k, v := range src.Buffs {
		dst.Buffs[k] = v
	}
	dst.Traits = make([]GeneralTraitConfig, len(src.Traits))
	for i, t := range src.Traits {
		dst.Traits[i] = cloneTraitConfig(t)
	}
	dst.SpecialTrait = cloneTraitConfig(src.SpecialTrait)
	dst.BonusTrait = cloneTraitConfig(src.BonusTrait)
	return dst
}

// cloneTraitConfig 深拷贝一条将领特性配置。
func cloneTraitConfig(t GeneralTraitConfig) GeneralTraitConfig {
	clonedParams := make(map[string]float64, len(t.Params))
	for k, v := range t.Params {
		clonedParams[k] = v
	}
	t.Params = clonedParams
	t.AllowedSides = append([]string(nil), t.AllowedSides...)
	t.AllowedScenes = append([]string(nil), t.AllowedScenes...)
	return t
}

// ValidateGeneralsConfig 校验将领配置的一致性和数值边界。
func ValidateGeneralsConfig(cfg GeneralsConfig) error {
	// 获取阵营配置，构建 generalID -> faction 的映射
	factions := GetFactionsConfig()
	factionGenerals := make(map[string]string) // generalID -> faction

	for faction, fc := range factions {
		for _, g := range fc.Generals {
			if existing, ok := factionGenerals[g.ID]; ok {
				return errors.New("general " + g.ID + " appears in multiple factions: " + existing + " and " + faction)
			}
			factionGenerals[g.ID] = faction
		}
	}

	if err := validateLevelBuffs(cfg.Common.LevelBuffs); err != nil {
		return err
	}
	if err := validateExpCurve(cfg.Common.ExpCurve); err != nil {
		return err
	}

	// 校验每个将领的 faction 字段、属性和特性参数。
	for generalID, hero := range cfg.Heroes {
		if hero.ID != "" && hero.ID != generalID {
			return fmt.Errorf("general %s has mismatched id %s", generalID, hero.ID)
		}
		if hero.Faction == "" {
			if hero.Enabled {
				return fmt.Errorf("enabled general %s has empty faction", generalID)
			}
		} else {
			// 检查该将领是否在阵营配置中
			factionInConfig, exists := factionGenerals[generalID]
			if !exists {
				if hero.Enabled {
					return fmt.Errorf("enabled general %s is not listed in factions config", generalID)
				}
			} else if hero.Faction != factionInConfig {
				return fmt.Errorf("general %s has faction=%s but is listed in faction %s in factions config", generalID, hero.Faction, factionInConfig)
			}
		}

		if err := validateGeneralBuffs("heroes."+generalID+".buffs", hero.Buffs); err != nil {
			return err
		}
		normalizeHeroDualTraits(&hero)
		if err := validateGeneralDualTraits(generalID, hero); err != nil {
			return err
		}
	}

	return nil
}

func validateExpCurve(expCurve []int) error {
	if len(expCurve) > GeneralMaxLevel {
		return fmt.Errorf("common.expCurve has %d entries, max is %d", len(expCurve), GeneralMaxLevel)
	}
	for i, value := range expCurve {
		level := i + 1
		if value < 0 {
			return fmt.Errorf("common.expCurve level %d must be >= 0", level)
		}
		if level == 1 && value != 0 {
			return fmt.Errorf("common.expCurve level 1 must be 0")
		}
		if i > 0 && value <= expCurve[i-1] {
			return fmt.Errorf("common.expCurve level %d must be greater than level %d", level, level-1)
		}
	}
	return nil
}

func validateLevelBuffs(levelBuffs map[int]map[string]float64) error {
	for level, buffs := range levelBuffs {
		if level <= 0 || level > GeneralMaxLevel {
			return fmt.Errorf("common.levelBuffs contains invalid level %d", level)
		}
		if err := validateGeneralBuffs(fmt.Sprintf("common.levelBuffs.%d", level), buffs); err != nil {
			return err
		}
	}
	return nil
}

func validateGeneralBuffs(label string, buffs map[string]float64) error {
	for key, value := range buffs {
		if !IsValidStatKey(key) {
			return fmt.Errorf("%s contains unknown stat key %s", label, key)
		}
		if value < 0 || value > maxGeneralAttributeBonus {
			return fmt.Errorf("%s.%s=%g out of range [0,%g]", label, key, value, maxGeneralAttributeBonus)
		}
	}
	return nil
}

func validateGeneralDualTraits(generalID string, hero GeneralHeroConfig) error {
	traits := activeHeroTraitConfigs(hero)
	if hero.Enabled {
		if strings.TrimSpace(hero.SpecialTrait.TraitID) == "" {
			return fmt.Errorf("enabled general %s must configure specialTrait", generalID)
		}
		if strings.TrimSpace(hero.BonusTrait.TraitID) == "" {
			return fmt.Errorf("enabled general %s must configure bonusTrait", generalID)
		}
	}
	if hero.Enabled || strings.TrimSpace(hero.SpecialTrait.TraitID) != "" {
		if err := validateSingleGeneralTrait(generalID, hero.SpecialTrait, general.TraitTypeSpecial); err != nil {
			return err
		}
	}
	if hero.Enabled || strings.TrimSpace(hero.BonusTrait.TraitID) != "" {
		if err := validateSingleGeneralTrait(generalID, hero.BonusTrait, general.TraitTypeBonus); err != nil {
			return err
		}
	}
	for _, traitCfg := range traits {
		if traitCfg.TraitID == hero.SpecialTrait.TraitID || traitCfg.TraitID == hero.BonusTrait.TraitID {
			continue
		}
		if err := validateSingleGeneralTrait(generalID, traitCfg, ""); err != nil {
			return err
		}
	}
	return nil
}

func validateSingleGeneralTrait(generalID string, traitCfg GeneralTraitConfig, expectedType string) error {
	if strings.TrimSpace(traitCfg.TraitID) == "" {
		if expectedType == "" {
			return nil
		}
		return fmt.Errorf("general %s %s trait is empty", generalID, expectedType)
	}
	trait, ok := general.Get(traitCfg.TraitID)
	if !ok {
		return fmt.Errorf("general %s uses unknown trait %s", generalID, traitCfg.TraitID)
	}
	if expectedType != "" && trait.Type() != expectedType {
		return fmt.Errorf("general %s trait %s type mismatch: expected %s got %s", generalID, traitCfg.TraitID, expectedType, trait.Type())
	}
	if traitCfg.TraitType != "" && traitCfg.TraitType != trait.Type() {
		return fmt.Errorf("general %s trait %s config type %s does not match registered type %s", generalID, traitCfg.TraitID, traitCfg.TraitType, trait.Type())
	}
	if err := validateTraitActivation(generalID, traitCfg); err != nil {
		return err
	}
	if err := validateTraitTargetUnitType(generalID, traitCfg); err != nil {
		return err
	}
	schemaByKey := map[string]general.ParamField{}
	for _, field := range trait.ParamSchema() {
		schemaByKey[field.Key] = field
		if value, ok := traitCfg.Params[field.Key]; ok {
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return fmt.Errorf("general %s trait %s param %s must be finite", generalID, traitCfg.TraitID, field.Key)
			}
			if value < field.Min || value > field.Max {
				return fmt.Errorf("general %s trait %s param %s=%g out of range [%g,%g]", generalID, traitCfg.TraitID, field.Key, value, field.Min, field.Max)
			}
		}
	}
	for key := range traitCfg.Params {
		if _, ok := schemaByKey[key]; !ok {
			return fmt.Errorf("general %s trait %s contains unknown param %s", generalID, traitCfg.TraitID, key)
		}
	}
	return nil
}

// validateTraitTargetUnitType 校验特性目标必须是已注册兵种 ID 或已存在的兵种分类。
func validateTraitTargetUnitType(generalID string, traitCfg GeneralTraitConfig) error {
	target := strings.TrimSpace(traitCfg.TargetUnitType)
	if target == "" {
		return nil
	}
	units := GetUnitsConfig()
	hasRegisteredUnits := false
	for _, factionUnits := range units {
		for unitID, unit := range factionUnits {
			hasRegisteredUnits = true
			if unitID == target || strings.EqualFold(strings.TrimSpace(unit.Category), target) {
				return nil
			}
		}
	}
	if !hasRegisteredUnits {
		return nil
	}
	return fmt.Errorf("general %s trait %s targets unknown unit or category %s", generalID, traitCfg.TraitID, target)
}

// validateTraitActivation 校验特性触发阵营、场景和胜负条件。
func validateTraitActivation(generalID string, traitCfg GeneralTraitConfig) error {
	validSides := map[string]bool{"attacker": true, "defender": true, "reinforcement": true}
	for _, side := range traitCfg.AllowedSides {
		if !validSides[strings.ToLower(strings.TrimSpace(side))] {
			return fmt.Errorf("general %s trait %s contains invalid allowed side %s", generalID, traitCfg.TraitID, side)
		}
	}
	validScenes := map[string]bool{
		"attack": true, "plunder": true, "sweep": true, "yellow_turban": true,
		"reinforcement": true, "reinforcement_defense": true, "recruit": true,
	}
	for _, scene := range traitCfg.AllowedScenes {
		if !validScenes[strings.ToLower(strings.TrimSpace(scene))] {
			return fmt.Errorf("general %s trait %s contains invalid allowed scene %s", generalID, traitCfg.TraitID, scene)
		}
	}
	outcome := strings.ToLower(strings.TrimSpace(traitCfg.RequiredOutcome))
	if outcome != "" && outcome != "win" && outcome != "loss" {
		return fmt.Errorf("general %s trait %s contains invalid required outcome %s", generalID, traitCfg.TraitID, traitCfg.RequiredOutcome)
	}
	return nil
}

func normalizeHeroDualTraits(hero *GeneralHeroConfig) {
	if hero == nil {
		return
	}
	if strings.TrimSpace(hero.SpecialTrait.TraitID) != "" && strings.TrimSpace(hero.BonusTrait.TraitID) != "" {
		return
	}
	for _, traitCfg := range hero.Traits {
		trait, ok := general.Get(traitCfg.TraitID)
		if !ok {
			continue
		}
		traitCfg.TraitType = trait.Type()
		if trait.Type() == general.TraitTypeSpecial && strings.TrimSpace(hero.SpecialTrait.TraitID) == "" {
			hero.SpecialTrait = traitCfg
		}
		if trait.Type() == general.TraitTypeBonus && strings.TrimSpace(hero.BonusTrait.TraitID) == "" {
			hero.BonusTrait = traitCfg
		}
	}
}

func activeHeroTraitConfigs(hero GeneralHeroConfig) []GeneralTraitConfig {
	traits := []GeneralTraitConfig{}
	if strings.TrimSpace(hero.SpecialTrait.TraitID) != "" {
		traits = append(traits, hero.SpecialTrait)
	}
	if strings.TrimSpace(hero.BonusTrait.TraitID) != "" {
		traits = append(traits, hero.BonusTrait)
	}
	traits = append(traits, hero.Traits...)
	return traits
}
