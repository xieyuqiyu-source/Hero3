package combat

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
)

// RuleConfig 单条战斗规则配置
type RuleConfig struct {
	ID                        string  `json:"id"`
	Name                      string  `json:"name"`
	Mode                      string  `json:"mode"`                      // "attack" or "plunder"
	Exponent                  float64 `json:"exponent"`                  // 损失指数（默认 1.422）
	NoLossPowerRatioThreshold float64 `json:"noLossPowerRatioThreshold"` // 弱强实力比低于该值时胜方无损
	EqualResult               string  `json:"equalResult"`               // "mutual_destruction" or "defender_wins"
	LossDistribution          string  `json:"lossDistribution"`          // "proportional" or "weak_first"
	DefenseFormula            string  `json:"defenseFormula"`            // "weighted"（按步骑加权）
}

// CombatConfig 战斗系统总配置
type CombatConfig struct {
	ActiveRules map[string]string     `json:"activeCombatRules"` // 场景 → 规则 ID
	Rules       map[string]RuleConfig `json:"rules"`
	WallConfig  map[string]WallEntry  `json:"wallConfig"`
}

// WallEntry 城墙配置
type WallEntry struct {
	Base                  float64 `json:"base"`                  // 城墙系数底数
	Hardness              float64 `json:"hardness,omitempty"`    // 城墙硬度预留，后续用于攻城武器损坏判定
	MinDamagedLevelFrom20 int     `json:"minDamagedLevelFrom20"` // 20 级被攻城武器破坏后的最低预期等级
	MaxDamagedLevelFrom20 int     `json:"maxDamagedLevelFrom20"` // 20 级被攻城武器破坏后的最高预期等级
}

const (
	ScenePVEAttack  = "pve_attack"
	ScenePVEPlunder = "pve_plunder"
	ScenePVPAttack  = "pvp_attack"
	ScenePVPPlunder = "pvp_plunder"

	RuleOfficialAttack  = "official_attack"
	RuleOfficialPlunder = "official_plunder"
)

// --- 全局配置管理 ---

var (
	combatMu     sync.RWMutex
	activeCombat = defaultCombatConfig()
)

func GetCombatConfig() CombatConfig {
	combatMu.RLock()
	defer combatMu.RUnlock()
	return cloneCombatConfig(activeCombat)
}

// cloneCombatConfig 深拷贝战斗规则、场景映射和城墙配置，隔离全局配置与调用方修改。
func cloneCombatConfig(config CombatConfig) CombatConfig {
	cloned := CombatConfig{
		ActiveRules: make(map[string]string, len(config.ActiveRules)),
		Rules:       make(map[string]RuleConfig, len(config.Rules)),
		WallConfig:  make(map[string]WallEntry, len(config.WallConfig)),
	}
	for scene, ruleID := range config.ActiveRules {
		cloned.ActiveRules[scene] = ruleID
	}
	for ruleID, rule := range config.Rules {
		cloned.Rules[ruleID] = rule
	}
	for faction, wall := range config.WallConfig {
		cloned.WallConfig[faction] = wall
	}
	return cloned
}

func GetRule(ruleID string) (RuleConfig, bool) {
	ruleID = strings.TrimSpace(ruleID)
	cfg := GetCombatConfig()
	rule, ok := cfg.Rules[ruleID]
	return rule, ok
}

func RegisterRule(rule RuleConfig) error {
	rule.ID = strings.TrimSpace(rule.ID)
	rule.Mode = strings.TrimSpace(rule.Mode)
	if rule.ID == "" {
		return errors.New("combat rule id is required")
	}
	if rule.Mode != "attack" && rule.Mode != "plunder" {
		return errors.New("combat rule mode is invalid: " + rule.Mode)
	}
	if rule.Exponent <= 0 {
		rule.Exponent = 1.422
	}
	if rule.NoLossPowerRatioThreshold <= 0 {
		rule.NoLossPowerRatioThreshold = 0.001
	}
	if rule.EqualResult == "" {
		rule.EqualResult = "mutual_destruction"
	}
	if rule.LossDistribution == "" {
		rule.LossDistribution = "proportional"
	}
	if rule.DefenseFormula == "" {
		rule.DefenseFormula = "weighted"
	}

	combatMu.Lock()
	defer combatMu.Unlock()
	if activeCombat.Rules == nil {
		activeCombat.Rules = map[string]RuleConfig{}
	}
	if _, exists := activeCombat.Rules[rule.ID]; exists {
		return errors.New("combat rule already registered: " + rule.ID)
	}
	activeCombat.Rules[rule.ID] = rule
	return nil
}

func SetActiveRule(scene string, ruleID string) error {
	scene = strings.TrimSpace(scene)
	ruleID = strings.TrimSpace(ruleID)
	if scene == "" {
		return errors.New("combat scene is required")
	}
	if ruleID == "" {
		return errors.New("combat rule id is required")
	}

	combatMu.Lock()
	defer combatMu.Unlock()
	if _, exists := activeCombat.Rules[ruleID]; !exists {
		return errors.New("combat rule not found: " + ruleID)
	}
	if activeCombat.ActiveRules == nil {
		activeCombat.ActiveRules = map[string]string{}
	}
	activeCombat.ActiveRules[scene] = ruleID
	return nil
}

func GetActiveRuleID(scene string) (string, bool) {
	scene = strings.TrimSpace(scene)
	cfg := GetCombatConfig()
	ruleID, ok := cfg.ActiveRules[scene]
	if !ok || strings.TrimSpace(ruleID) == "" {
		return "", false
	}
	if _, exists := cfg.Rules[ruleID]; !exists {
		return "", false
	}
	return ruleID, true
}

func RuleIDForScene(scene string) string {
	if ruleID, ok := GetActiveRuleID(scene); ok {
		return ruleID
	}
	if scene == ScenePVEPlunder || scene == ScenePVPPlunder {
		return RuleOfficialPlunder
	}
	return RuleOfficialAttack
}

func GetActiveRule(scene string) (RuleConfig, bool) {
	ruleID, ok := GetActiveRuleID(scene)
	if !ok {
		return RuleConfig{}, false
	}
	cfg := GetCombatConfig()
	rule, exists := cfg.Rules[ruleID]
	return rule, exists
}

func LoadCombatConfig(path string) error {
	if path == "" {
		return nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	var config CombatConfig
	if err := json.Unmarshal(content, &config); err != nil {
		return err
	}

	normalizeCombatConfig(&config)

	combatMu.Lock()
	activeCombat = config
	combatMu.Unlock()
	return nil
}

func SaveCombatConfig(path string, config CombatConfig) error {
	config = cloneCombatConfig(config)
	normalizeCombatConfig(&config)

	if path == "" {
		combatMu.Lock()
		activeCombat = config
		combatMu.Unlock()
		return nil
	}

	content, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, append(content, '\n'), 0o644); err != nil {
		return err
	}

	combatMu.Lock()
	activeCombat = config
	combatMu.Unlock()
	return nil
}

func defaultCombatConfig() CombatConfig {
	return CombatConfig{
		ActiveRules: map[string]string{
			ScenePVEAttack:  RuleOfficialAttack,
			ScenePVEPlunder: RuleOfficialPlunder,
			ScenePVPAttack:  RuleOfficialAttack,
			ScenePVPPlunder: RuleOfficialPlunder,
		},
		Rules: map[string]RuleConfig{
			RuleOfficialAttack: {
				ID:                        RuleOfficialAttack,
				Name:                      "官方攻击规则",
				Mode:                      "attack",
				Exponent:                  1.422,
				NoLossPowerRatioThreshold: 0.001,
				EqualResult:               "mutual_destruction",
				LossDistribution:          "proportional",
				DefenseFormula:            "weighted",
			},
			RuleOfficialPlunder: {
				ID:                        RuleOfficialPlunder,
				Name:                      "官方掠夺规则",
				Mode:                      "plunder",
				Exponent:                  1.422,
				NoLossPowerRatioThreshold: 0.001,
				EqualResult:               "half_each",
				LossDistribution:          "proportional",
				DefenseFormula:            "weighted",
			},
		},
		WallConfig: map[string]WallEntry{
			"wei": {Base: 1.03, Hardness: 0.75, MinDamagedLevelFrom20: 5, MaxDamagedLevelFrom20: 5},
			"shu": {Base: 1.02, Hardness: 1.35, MinDamagedLevelFrom20: 16, MaxDamagedLevelFrom20: 17},
			"wu":  {Base: 1.025, Hardness: 1.0, MinDamagedLevelFrom20: 9, MaxDamagedLevelFrom20: 12},
		},
	}
}

// normalizeCombatConfig 补齐旧配置缺失字段，保证默认规则在加载和保存后稳定。
func normalizeCombatConfig(config *CombatConfig) {
	if config == nil {
		return
	}
	for id, rule := range config.Rules {
		rule.ID = strings.TrimSpace(rule.ID)
		if rule.ID == "" {
			rule.ID = id
		}
		rule.Mode = strings.TrimSpace(rule.Mode)
		if rule.Mode == "" {
			rule.Mode = "attack"
		}
		if rule.Exponent <= 0 {
			rule.Exponent = 1.422
		}
		if rule.NoLossPowerRatioThreshold <= 0 {
			rule.NoLossPowerRatioThreshold = 0.001
		}
		if rule.EqualResult == "" {
			rule.EqualResult = "mutual_destruction"
		}
		if rule.LossDistribution == "" {
			rule.LossDistribution = "proportional"
		}
		if rule.DefenseFormula == "" {
			rule.DefenseFormula = "weighted"
		}
		config.Rules[id] = rule
	}
	defaults := defaultCombatConfig()
	if config.WallConfig == nil {
		config.WallConfig = map[string]WallEntry{}
	}
	for faction, fallback := range defaults.WallConfig {
		entry := config.WallConfig[faction]
		if entry.Base <= 0 {
			entry.Base = fallback.Base
		}
		if entry.Hardness <= 0 {
			entry.Hardness = fallback.Hardness
		}
		if entry.MinDamagedLevelFrom20 <= 0 {
			entry.MinDamagedLevelFrom20 = fallback.MinDamagedLevelFrom20
		}
		if entry.MaxDamagedLevelFrom20 <= 0 {
			entry.MaxDamagedLevelFrom20 = fallback.MaxDamagedLevelFrom20
		}
		if entry.MinDamagedLevelFrom20 > entry.MaxDamagedLevelFrom20 {
			entry.MinDamagedLevelFrom20, entry.MaxDamagedLevelFrom20 = entry.MaxDamagedLevelFrom20, entry.MinDamagedLevelFrom20
		}
		config.WallConfig[faction] = entry
	}
}
