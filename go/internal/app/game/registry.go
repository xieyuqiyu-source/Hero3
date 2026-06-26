package game

import coreregistry "hero3/internal/core/registry"

type RewardTypeDefinition = coreregistry.RewardTypeDefinition
type EventTypeDefinition = coreregistry.EventTypeDefinition
type ResourceTypeDefinition = coreregistry.ResourceTypeDefinition

func init() {
	mustRegisterResourceType(ResourceTypeDefinition{Type: "wood", Description: "木材"})
	mustRegisterResourceType(ResourceTypeDefinition{Type: "stone", Description: "石料"})
	mustRegisterResourceType(ResourceTypeDefinition{Type: "iron", Description: "铁矿"})
	mustRegisterResourceType(ResourceTypeDefinition{Type: "food", Description: "粮食"})

	mustRegisterRewardType(RewardTypeDefinition{Type: RewardTypeResource, Description: "资源奖励"})
	mustRegisterRewardType(RewardTypeDefinition{Type: RewardTypeCityGold, Description: "城金奖励"})
	mustRegisterRewardType(RewardTypeDefinition{Type: RewardTypeGold, Description: "账号金币奖励", RequiresAccount: true})
	mustRegisterRewardType(RewardTypeDefinition{Type: RewardTypeItem, Description: "道具奖励"})
	mustRegisterRewardType(RewardTypeDefinition{Type: RewardTypeUnit, Description: "兵力奖励"})
	mustRegisterRewardType(RewardTypeDefinition{Type: RewardTypeGeneral, Description: "武将获取奖励"})
	mustRegisterRewardType(RewardTypeDefinition{Type: RewardTypeGeneralExp, Description: "武将经验奖励"})
	mustRegisterRewardType(RewardTypeDefinition{Type: RewardTypeBuff, Description: "Buff/Modifier 奖励"})

	mustRegisterEventType(EventTypeDefinition{Type: EventPlayerCreated, Description: "玩家创建"})
	mustRegisterEventType(EventTypeDefinition{Type: EventResourceChanged, Description: "资源变化"})
	mustRegisterEventType(EventTypeDefinition{Type: EventBuildingUpgraded, Description: "建筑升级"})
	mustRegisterEventType(EventTypeDefinition{Type: EventUnitChanged, Description: "兵力变化"})
	mustRegisterEventType(EventTypeDefinition{Type: EventCurrencyChanged, Description: "货币变化"})
	mustRegisterEventType(EventTypeDefinition{Type: EventGeneralChanged, Description: "武将变化"})
	mustRegisterEventType(EventTypeDefinition{Type: EventItemUsed, Description: "道具使用"})
	mustRegisterEventType(EventTypeDefinition{Type: EventRewardGranted, Description: "奖励发放"})
	mustRegisterEventType(EventTypeDefinition{Type: EventBattleFinished, Description: "战斗完成"})
	mustRegisterEventType(EventTypeDefinition{Type: EventMailClaimed, Description: "信函附件领取"})
	mustRegisterEventType(EventTypeDefinition{Type: EventMiniGameRedeemed, Description: "万象幻境兑换"})
}

func RegisterRewardType(def RewardTypeDefinition) error {
	return coreregistry.RegisterRewardType(def)
}

func GetRewardTypeDefinition(rewardType string) (RewardTypeDefinition, bool) {
	return coreregistry.GetRewardTypeDefinition(rewardType)
}

func ListRewardTypeDefinitions() []RewardTypeDefinition {
	return coreregistry.ListRewardTypeDefinitions()
}

func RegisterResourceType(def ResourceTypeDefinition) error {
	return coreregistry.RegisterResourceType(def)
}

func IsResourceTypeRegistered(resourceType string) bool {
	return coreregistry.IsResourceTypeRegistered(resourceType)
}

func ListResourceTypeDefinitions() []ResourceTypeDefinition {
	return coreregistry.ListResourceTypeDefinitions()
}

func RegisterEventType(def EventTypeDefinition) error {
	return coreregistry.RegisterEventType(def)
}

func IsEventTypeRegistered(eventType string) bool {
	return coreregistry.IsEventTypeRegistered(eventType)
}

func mustRegisterResourceType(def ResourceTypeDefinition) {
	if err := RegisterResourceType(def); err != nil {
		panic(err)
	}
}

func mustRegisterRewardType(def RewardTypeDefinition) {
	if err := RegisterRewardType(def); err != nil {
		panic(err)
	}
}

func mustRegisterEventType(def EventTypeDefinition) {
	if err := RegisterEventType(def); err != nil {
		panic(err)
	}
}
