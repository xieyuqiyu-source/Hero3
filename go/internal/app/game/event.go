package game

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
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

// BuildEventProcessingKey 生成模块事件处理幂等 key。
func BuildEventProcessingKey(event GameEvent) string {
	parts := []string{
		strings.TrimSpace(event.Type),
		strings.TrimSpace(event.PlayerID),
		strings.TrimSpace(event.AccountID),
		strings.TrimSpace(event.RefType),
		strings.TrimSpace(event.RefID),
		strings.TrimSpace(event.CreatedAt),
	}
	hash := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(hash[:])
}

// ClaimEventForModule 为玩法模块声明一次事件处理权，重复事件返回 false。
func (s *Service) ClaimEventForModule(moduleID string, handlerKey string, event GameEvent) (bool, error) {
	moduleID = strings.TrimSpace(moduleID)
	handlerKey = strings.TrimSpace(handlerKey)
	if moduleID == "" || handlerKey == "" || event.Type == "" {
		return false, ErrInvalidEffectType
	}
	if _, ok := GetGameplayModuleDefinition(moduleID); !ok {
		return false, ErrInvalidEffectType
	}
	if !IsEventTypeRegistered(event.Type) {
		return false, ErrInvalidEffectType
	}
	return s.repo.ClaimEventProcessing(moduleID, handlerKey, BuildEventProcessingKey(event), time.Now())
}
