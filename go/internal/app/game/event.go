package game

import (
	"time"

	coreevent "hero3/internal/core/event"
)

const (
	EventPlayerCreated    = "player.created"
	EventResourceChanged  = "resource.changed"
	EventBuildingUpgraded = "building.upgraded"
	EventUnitChanged      = "unit.changed"
	EventCurrencyChanged  = "currency.changed"
	EventGeneralChanged   = "general.changed"
	EventItemUsed         = "item.used"
	EventRewardGranted    = "reward.granted"
	EventBattleFinished   = "battle.finished"
	EventMailClaimed      = "mail.claimed"
	EventMiniGameRedeemed = "minigame.redeemed"
)

type GameEvent = coreevent.Event

type EventHandler = coreevent.Handler

type EventBus = coreevent.Bus

func NewEventBus() *EventBus {
	return coreevent.NewBus(IsEventTypeRegistered, func(t time.Time) string {
		return t.UTC().Format(resourceDateLayout)
	})
}
