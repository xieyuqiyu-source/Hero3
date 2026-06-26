package game

import (
	"errors"
	"sort"
	"strings"
	"sync"
)

// 本文件归口应用层玩法模块边界声明，供后续活动、副本按标准接入核心。

type GameplayModuleDefinition struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Description      string   `json:"description,omitempty"`
	StateOwner       string   `json:"stateOwner,omitempty"`
	RepositoryPort   string   `json:"repositoryPort,omitempty"`
	RewardEntrypoint string   `json:"rewardEntrypoint,omitempty"`
	EventTypes       []string `json:"eventTypes,omitempty"`
	RewardTypes      []string `json:"rewardTypes,omitempty"`
	CoreEntrypoints  []string `json:"coreEntrypoints,omitempty"`
	BoundaryRules    []string `json:"boundaryRules,omitempty"`
}

var (
	gameplayModulesMu sync.RWMutex
	gameplayModules   = map[string]GameplayModuleDefinition{}
)

func init() {
	mustRegisterGameplayModule(GameplayModuleDefinition{
		ID:               "mail",
		Name:             "信函",
		Description:      "玩家信箱、系统邮件、GM 邮件和附件领取模块。",
		StateOwner:       "mails table + player unread counters",
		RepositoryPort:   "MailRepository",
		RewardEntrypoint: "ApplyRewardsToStateWithContext",
		EventTypes:       []string{EventMailClaimed, EventRewardGranted, EventCurrencyChanged, EventItemUsed},
		RewardTypes:      []string{RewardTypeResource, RewardTypeCityGold, RewardTypeGold, RewardTypeItem, RewardTypeUnit, RewardTypeGeneral, RewardTypeGeneralExp, RewardTypeBuff},
		CoreEntrypoints:  []string{"UpdateMailPlayerState", "ApplyRewardsToStateWithContext", "flushRewardSideEffects", "publishMailClaimEvents"},
		BoundaryRules: []string{
			"信函模块拥有信函列表、阅读、删除、发送和附件领取规则。",
			"附件发放必须转换为标准 Reward，并通过奖励应用入口进入核心长期资产。",
			"账号金币附件必须使用信函 + 账号 + 玩家组合事务，不允许先改信函再单独改账号资产。",
		},
	})
	mustRegisterGameplayModule(GameplayModuleDefinition{
		ID:               "minigame",
		Name:             "万象幻境",
		Description:      "小游戏记录、鱼饵消耗、奖励库存和兑换模块。",
		StateOwner:       "minigame_records table",
		RepositoryPort:   "MiniGameRecordRepository",
		RewardEntrypoint: "ApplyRewardsToStateWithContext",
		EventTypes:       []string{EventMiniGameRedeemed, EventRewardGranted, EventCurrencyChanged},
		RewardTypes:      []string{RewardTypeUnit},
		CoreEntrypoints:  []string{"UpdateMiniGamePlayerState", "ApplyRewardsToStateWithContext", "flushRewardSideEffects", "publishMiniGameRedeemEvents"},
		BoundaryRules: []string{
			"万象幻境模块拥有玩法记录、库存和兑换规则。",
			"兑换产出必须转换为标准 Reward，并通过奖励应用入口进入核心长期资产。",
			"跨阵营或不可识别产出不能绕过兵种注册表直接写入玩家兵力。",
		},
	})
}

// RegisterGameplayModule 注册玩法模块边界声明。
func RegisterGameplayModule(def GameplayModuleDefinition) error {
	def.ID = strings.TrimSpace(def.ID)
	def.Name = strings.TrimSpace(def.Name)
	if def.ID == "" || def.Name == "" {
		return errors.New("gameplay module id and name are required")
	}
	gameplayModulesMu.Lock()
	defer gameplayModulesMu.Unlock()
	if _, exists := gameplayModules[def.ID]; exists {
		return errors.New("gameplay module already registered: " + def.ID)
	}
	gameplayModules[def.ID] = cloneGameplayModuleDefinition(def)
	return nil
}

// GetGameplayModuleDefinition 获取玩法模块边界声明。
func GetGameplayModuleDefinition(moduleID string) (GameplayModuleDefinition, bool) {
	moduleID = strings.TrimSpace(moduleID)
	gameplayModulesMu.RLock()
	defer gameplayModulesMu.RUnlock()
	def, exists := gameplayModules[moduleID]
	if !exists {
		return GameplayModuleDefinition{}, false
	}
	return cloneGameplayModuleDefinition(def), true
}

// ListGameplayModuleDefinitions 列出全部玩法模块边界声明。
func ListGameplayModuleDefinitions() []GameplayModuleDefinition {
	gameplayModulesMu.RLock()
	defer gameplayModulesMu.RUnlock()
	defs := make([]GameplayModuleDefinition, 0, len(gameplayModules))
	for _, def := range gameplayModules {
		defs = append(defs, cloneGameplayModuleDefinition(def))
	}
	sort.Slice(defs, func(i, j int) bool {
		return defs[i].ID < defs[j].ID
	})
	return defs
}

// ListGameplayModuleIDs 列出全部玩法模块 ID。
func ListGameplayModuleIDs() []string {
	defs := ListGameplayModuleDefinitions()
	ids := make([]string, 0, len(defs))
	for _, def := range defs {
		ids = append(ids, def.ID)
	}
	return ids
}

func mustRegisterGameplayModule(def GameplayModuleDefinition) {
	if err := RegisterGameplayModule(def); err != nil {
		panic(err)
	}
}

func cloneGameplayModuleDefinition(source GameplayModuleDefinition) GameplayModuleDefinition {
	next := source
	next.EventTypes = append([]string(nil), source.EventTypes...)
	next.RewardTypes = append([]string(nil), source.RewardTypes...)
	next.CoreEntrypoints = append([]string(nil), source.CoreEntrypoints...)
	next.BoundaryRules = append([]string(nil), source.BoundaryRules...)
	return next
}
