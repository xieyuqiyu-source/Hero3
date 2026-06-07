package game

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"

	"hero3/internal/general"
	_ "hero3/internal/general/traits"
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
	Traits []GeneralTraitConfig `json:"traits"`
}

// GeneralTraitConfig 单将领的某个特性的配置
type GeneralTraitConfig struct {
	TraitID string             `json:"traitId"` // 对应 general.traits 注册的 id（如 "meiren"）
	Enabled bool               `json:"enabled"` // 该特性是否启用
	Params  map[string]float64 `json:"params"`  // 当前参数（覆盖默认值）
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

// GetHeroConfig 根据 generalId 获取该将领的配置（用于注入 General.Traits）
func GetHeroConfig(generalID string) (GeneralHeroConfig, bool) {
	generalsMu.RLock()
	defer generalsMu.RUnlock()
	hero, ok := activeGenerals.Heroes[generalID]
	return hero, ok
}

func SetGeneralsConfig(cfg GeneralsConfig) error {
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
				Buffs: map[string]float64{"productionBonus": 0.10},
				Traits: []GeneralTraitConfig{
					{
						TraitID: "meiren", Enabled: true,
						Params: map[string]float64{"captureRate": 0.1, "captureMax": 1000, "triggerChance": 1.0},
					},
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
		clonedParams := make(map[string]float64, len(t.Params))
		for k, v := range t.Params {
			clonedParams[k] = v
		}
		dst.Traits[i] = GeneralTraitConfig{
			TraitID: t.TraitID,
			Enabled: t.Enabled,
			Params:  clonedParams,
		}
	}
	return dst
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
		if err := validateGeneralTraits(generalID, hero.Traits); err != nil {
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

func validateGeneralTraits(generalID string, traits []GeneralTraitConfig) error {
	for _, traitCfg := range traits {
		trait, ok := general.Get(traitCfg.TraitID)
		if !ok {
			return fmt.Errorf("general %s uses unknown trait %s", generalID, traitCfg.TraitID)
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
		if err := validateTraitParamConsistency(generalID, traitCfg.TraitID, traitCfg.Params, schemaByKey); err != nil {
			return err
		}
	}
	return nil
}

func validateTraitParamConsistency(generalID string, traitID string, params map[string]float64, fields map[string]general.ParamField) error {
	valueOrDefault := func(key string) float64 {
		if value, ok := params[key]; ok {
			return value
		}
		return fields[key].Default
	}
	switch traitID {
	case "weizhenxiaoyao":
		if valueOrDefault("maxChance") < valueOrDefault("baseChance") {
			return fmt.Errorf("general %s trait %s: maxChance must be >= baseChance", generalID, traitID)
		}
		if valueOrDefault("maxSuppressRate") < valueOrDefault("baseSuppressRate") {
			return fmt.Errorf("general %s trait %s: maxSuppressRate must be >= baseSuppressRate", generalID, traitID)
		}
	}
	return nil
}
