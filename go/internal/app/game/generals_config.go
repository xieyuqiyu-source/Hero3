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

// normalizeTraitConfigParams 清理魏武号令的历史上限参数，保证当前规则不限累计产兵。
func normalizeTraitConfigParams(traitCfg *GeneralTraitConfig) {
	if traitCfg == nil || traitCfg.TraitID != "weiwu_haoling" || traitCfg.Params == nil {
		return
	}
	delete(traitCfg.Params, "maxGuardPerDay")
	delete(traitCfg.Params, "maxGuardPerSettle")
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
