package game

import (
	"crypto/rand"
	"encoding/json"
	"math/big"
	"sort"
	"strings"
	"time"
)

// 本文件归口道具使用后的效果执行。

// applyItemEffects 应用道具效果并返回本次实际产出。
func applyItemEffects(state *GameState, item ItemDefinition, count int) (map[string]int, error) {
	effects, err := itemEffectsToPipelineEffects(state, item, count)
	if err != nil {
		return nil, err
	}
	applied, err := ExecuteEffectsOnState(state, effects, EffectContext{
		PlayerID: state.Player.ID,
		RefType:  "item_use",
		RefID:    item.ID,
		Reason:   "item_use",
		Source:   "item",
	}, time.Now())
	if err != nil {
		return nil, err
	}
	return applied.Reward.Granted, nil
}

// itemEffectsToPipelineEffects 把道具效果定义转换为标准 Effect。
func itemEffectsToPipelineEffects(state *GameState, item ItemDefinition, count int) ([]Effect, error) {
	rewards := make([]Reward, 0, len(item.Effects))
	for _, effect := range item.Effects {
		switch effect.Type {
		case "pvp_protection":
			continue
		case "item":
			rewards = append(rewards, Reward{Type: RewardTypeItem, ID: strings.TrimSpace(effect.ID), Amount: effect.Amount * count})
		case "general":
			generalID := strings.TrimSpace(effect.GeneralID)
			if generalID == "" {
				return nil, ErrInvalidGeneral
			}
			rewards = append(rewards, Reward{Type: RewardTypeGeneral, ID: generalID, Amount: count})
		case "general_exp":
			rewards = append(rewards, Reward{Type: RewardTypeGeneralExp, ID: "current_general", Amount: effect.Amount * count})
		case "resources":
			for key, value := range effect.Resources {
				rewards = append(rewards, Reward{Type: RewardTypeResource, ID: key, Amount: value * count})
			}
		case "unit_by_faction":
			unitID := strings.TrimSpace(effect.UnitByFaction[state.Player.Faction])
			if unitID == "" {
				return nil, ErrUnitNotFound
			}
			if _, ok := GetUnitConfig(state.Player.Faction, unitID); !ok {
				return nil, ErrUnitNotFound
			}
			rewards = append(rewards, Reward{Type: RewardTypeUnit, ID: unitID, Amount: effect.Amount * count})
		case "random_unit_by_faction_category":
			for i := 0; i < count; i++ {
				unitID, err := randomFactionUnitIDByEffect(state.Player.Faction, effect)
				if err != nil {
					return nil, err
				}
				rewards = append(rewards, Reward{Type: RewardTypeUnit, ID: unitID, Amount: effect.Amount})
			}
		case "all_units_by_faction_category":
			unitIDs := factionUnitIDsByEffect(state.Player.Faction, effect)
			if len(unitIDs) == 0 {
				return nil, ErrUnitNotFound
			}
			for _, unitID := range unitIDs {
				rewards = append(rewards, Reward{Type: RewardTypeUnit, ID: unitID, Amount: effect.Amount * count})
			}
		case "currency":
			rewards = append(rewards, Reward{Type: strings.TrimSpace(effect.CurrencyType), ID: strings.TrimSpace(effect.CurrencyType), Amount: effect.Amount * count})
		case "buff":
			rewards = append(rewards, Reward{
				Type:   RewardTypeBuff,
				ID:     strings.TrimSpace(effect.BuffKey),
				Amount: effect.Amount,
				Metadata: map[string]any{
					"mode":   strings.TrimSpace(effect.BuffMode),
					"value":  effect.BuffValue,
					"source": "item",
					"note":   item.Name,
				},
			})
		case "random_reward":
			rolled, err := RollDropPoolRewards(effect.DropPoolID)
			if err != nil {
				return nil, err
			}
			for _, reward := range rolled {
				reward.Amount *= count
				rewards = append(rewards, reward)
			}
		default:
			return nil, ErrItemNotUsable
		}
	}
	return rewardsToEffects("item", rewards), nil
}

// validRecruitUnitCategory 判断招募券允许发放的兵种分类。
func validRecruitUnitCategory(category string) bool {
	switch strings.TrimSpace(category) {
	case "infantry", "cavalry", "siege", "special":
		return true
	default:
		return false
	}
}

// validRecruitUnitPool 判断招募券允许使用的兵种池范围。
func validRecruitUnitPool(pool string) bool {
	switch strings.TrimSpace(pool) {
	case "", "base", "all":
		return true
	default:
		return false
	}
}

// baseFactionUnitIDsByCategory 返回指定阵营、指定分类下最低解锁等级的作战兵种。
func baseFactionUnitIDsByCategory(faction string, category string) []string {
	return factionUnitIDsByCategory(faction, category, "base")
}

// factionUnitIDsByEffect 根据道具效果返回候选兵种池。
func factionUnitIDsByEffect(faction string, effect ItemEffect) []string {
	return factionUnitIDsByCategory(faction, effect.Category, effect.Pool)
}

// factionUnitIDsByCategory 返回指定阵营、分类和池范围下的作战兵种。
func factionUnitIDsByCategory(faction string, category string, pool string) []string {
	category = strings.TrimSpace(category)
	if !validRecruitUnitCategory(category) {
		return nil
	}
	units := GetFactionUnits(strings.TrimSpace(faction))
	bestLevel := 0
	result := []string{}
	for unitID, config := range units {
		if strings.TrimSpace(config.Category) != category || !isRecruitCombatUnit(config) {
			continue
		}
		if strings.TrimSpace(pool) == "all" {
			result = append(result, unitID)
			continue
		}
		level := unitUnlockLevel(config)
		if len(result) == 0 || level < bestLevel {
			bestLevel = level
			result = []string{unitID}
			continue
		}
		if level == bestLevel {
			result = append(result, unitID)
		}
	}
	sort.Strings(result)
	return result
}

// isRecruitCombatUnit 过滤侦察、运输等功能兵，避免招募券发放非作战单位。
func isRecruitCombatUnit(config UnitConfig) bool {
	switch strings.TrimSpace(config.Role) {
	case "scout", "transport":
		return false
	default:
		return true
	}
}

// unitUnlockLevel 从兵种配置中读取解锁等级，用于识别基础兵种。
func unitUnlockLevel(config UnitConfig) int {
	if config.Unlock == nil {
		return 9999
	}
	value, ok := config.Unlock["level"]
	if !ok {
		return 9999
	}
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		level, err := v.Int64()
		if err == nil {
			return int(level)
		}
	}
	return 9999
}

// randomFactionUnitIDByEffect 从道具效果指定的兵种池里随机选出一个兵种。
func randomFactionUnitIDByEffect(faction string, effect ItemEffect) (string, error) {
	unitIDs := factionUnitIDsByEffect(faction, effect)
	if len(unitIDs) == 0 {
		return "", ErrUnitNotFound
	}
	if len(unitIDs) == 1 {
		return unitIDs[0], nil
	}
	index, err := rand.Int(rand.Reader, big.NewInt(int64(len(unitIDs))))
	if err != nil {
		return "", err
	}
	return unitIDs[int(index.Int64())], nil
}
