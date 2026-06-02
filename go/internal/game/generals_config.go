package game

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

// GeneralsConfig 将领系统总配置（GM 后台可编辑）
type GeneralsConfig struct {
	Common GeneralsCommonConfig         `json:"common"`              // 通用配置（顶部）
	Heroes map[string]GeneralHeroConfig `json:"heroes"`              // 单将领配置 map[generalId]
	Enabled bool                        `json:"enabled"`             // 全局开关
}

// GeneralsCommonConfig 通用配置（所有将领共享）
type GeneralsCommonConfig struct {
	ExpCurve   []int                       `json:"expCurve"`   // 每级所需经验
	LevelBuffs map[int]map[string]float64  `json:"levelBuffs"` // 每级提供的通用 buff
}

// GeneralHeroConfig 单将领配置
type GeneralHeroConfig struct {
	ID      string                  `json:"id"`
	Name    string                  `json:"name"`
	Faction string                  `json:"faction"`
	Title   string                  `json:"title"`
	Rarity  string                  `json:"rarity"`  // common | rare | epic | legendary
	Enabled bool                    `json:"enabled"` // 该将领是否启用

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
