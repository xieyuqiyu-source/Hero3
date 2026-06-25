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
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	Mode             string  `json:"mode"`             // "attack" or "plunder"
	Exponent         float64 `json:"exponent"`         // 损失指数（默认 1.422）
	EqualResult      string  `json:"equalResult"`      // "mutual_destruction" or "defender_wins"
	LossDistribution string  `json:"lossDistribution"` // "proportional" or "weak_first"
	DefenseFormula   string  `json:"defenseFormula"`   // "weighted"（按步骑加权）
}

// CombatConfig 战斗系统总配置
type CombatConfig struct {
	ActiveRules map[string]string     `json:"activeCombatRules"` // 场景 → 规则 ID
	Rules       map[string]RuleConfig `json:"rules"`
	WallConfig  map[string]WallEntry  `json:"wallConfig"`
}

// WallEntry 城墙配置
type WallEntry struct {
	Base float64 `json:"base"` // 城墙系数底数
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
	return activeCombat
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

	combatMu.Lock()
	activeCombat = config
	combatMu.Unlock()
	return nil
}

func SaveCombatConfig(path string, config CombatConfig) error {
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
				ID:               RuleOfficialAttack,
				Name:             "官方攻击规则",
				Mode:             "attack",
				Exponent:         1.422,
				EqualResult:      "mutual_destruction",
				LossDistribution: "proportional",
				DefenseFormula:   "weighted",
			},
			RuleOfficialPlunder: {
				ID:               RuleOfficialPlunder,
				Name:             "官方掠夺规则",
				Mode:             "plunder",
				Exponent:         1.422,
				EqualResult:      "half_each",
				LossDistribution: "proportional",
				DefenseFormula:   "weighted",
			},
		},
		WallConfig: map[string]WallEntry{
			"wei": {Base: 1.03},
			"shu": {Base: 1.02},
			"wu":  {Base: 1.025},
		},
	}
}
