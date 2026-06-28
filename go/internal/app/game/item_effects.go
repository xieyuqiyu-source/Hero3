package game

import (
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
