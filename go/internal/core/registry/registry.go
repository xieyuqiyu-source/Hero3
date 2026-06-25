package registry

import (
	"errors"
	"strings"
	"sync"
)

type RewardTypeDefinition struct {
	Type            string
	Description     string
	RequiresAccount bool
}

type EventTypeDefinition struct {
	Type        string
	Description string
}

type ResourceTypeDefinition struct {
	Type        string
	Description string
}

var (
	rewardTypesMu sync.RWMutex
	rewardTypes   = map[string]RewardTypeDefinition{}

	eventTypesMu sync.RWMutex
	eventTypes   = map[string]EventTypeDefinition{}

	resourceTypesMu sync.RWMutex
	resourceTypes   = map[string]ResourceTypeDefinition{}
)

func RegisterRewardType(def RewardTypeDefinition) error {
	def.Type = strings.TrimSpace(def.Type)
	if def.Type == "" {
		return errors.New("reward type is required")
	}
	rewardTypesMu.Lock()
	defer rewardTypesMu.Unlock()
	if _, exists := rewardTypes[def.Type]; exists {
		return errors.New("reward type already registered: " + def.Type)
	}
	rewardTypes[def.Type] = def
	return nil
}

func GetRewardTypeDefinition(rewardType string) (RewardTypeDefinition, bool) {
	rewardType = strings.TrimSpace(rewardType)
	rewardTypesMu.RLock()
	defer rewardTypesMu.RUnlock()
	def, ok := rewardTypes[rewardType]
	return def, ok
}

func ListRewardTypeDefinitions() []RewardTypeDefinition {
	rewardTypesMu.RLock()
	defer rewardTypesMu.RUnlock()
	result := make([]RewardTypeDefinition, 0, len(rewardTypes))
	for _, def := range rewardTypes {
		result = append(result, def)
	}
	return result
}

func RegisterResourceType(def ResourceTypeDefinition) error {
	def.Type = strings.TrimSpace(def.Type)
	if def.Type == "" {
		return errors.New("resource type is required")
	}
	resourceTypesMu.Lock()
	defer resourceTypesMu.Unlock()
	if _, exists := resourceTypes[def.Type]; exists {
		return errors.New("resource type already registered: " + def.Type)
	}
	resourceTypes[def.Type] = def
	return nil
}

func IsResourceTypeRegistered(resourceType string) bool {
	resourceType = strings.TrimSpace(resourceType)
	resourceTypesMu.RLock()
	defer resourceTypesMu.RUnlock()
	_, ok := resourceTypes[resourceType]
	return ok
}

func ListResourceTypeDefinitions() []ResourceTypeDefinition {
	resourceTypesMu.RLock()
	defer resourceTypesMu.RUnlock()
	result := make([]ResourceTypeDefinition, 0, len(resourceTypes))
	for _, def := range resourceTypes {
		result = append(result, def)
	}
	return result
}

func RegisterEventType(def EventTypeDefinition) error {
	def.Type = strings.TrimSpace(def.Type)
	if def.Type == "" {
		return errors.New("event type is required")
	}
	eventTypesMu.Lock()
	defer eventTypesMu.Unlock()
	if _, exists := eventTypes[def.Type]; exists {
		return errors.New("event type already registered: " + def.Type)
	}
	eventTypes[def.Type] = def
	return nil
}

func IsEventTypeRegistered(eventType string) bool {
	eventType = strings.TrimSpace(eventType)
	eventTypesMu.RLock()
	defer eventTypesMu.RUnlock()
	_, ok := eventTypes[eventType]
	return ok
}
