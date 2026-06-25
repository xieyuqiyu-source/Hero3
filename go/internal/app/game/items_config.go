package game

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
)

type ItemsConfig map[string]ItemDefinition

type ItemDefinition struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Rarity      string                 `json:"rarity"`
	Icon        string                 `json:"icon,omitempty"`
	Usable      bool                   `json:"usable"`
	Stackable   bool                   `json:"stackable"`
	MaxStack    int                    `json:"maxStack"`
	UseTarget   string                 `json:"useTarget"`
	Effects     []ItemEffect           `json:"effects"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type ItemEffect struct {
	Type          string            `json:"type"`
	Amount        int               `json:"amount,omitempty"`
	Resources     map[string]int    `json:"resources,omitempty"`
	UnitByFaction map[string]string `json:"unitByFaction,omitempty"`
}

var (
	itemsMu     sync.RWMutex
	itemsConfig ItemsConfig = ItemsConfig{}
)

func LoadItemsConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var cfg ItemsConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}
	if err := ValidateItemsConfig(cfg); err != nil {
		return err
	}
	itemsMu.Lock()
	itemsConfig = cfg
	itemsMu.Unlock()
	return nil
}

func ValidateItemsConfig(cfg ItemsConfig) error {
	for id, item := range cfg {
		if strings.TrimSpace(id) == "" || strings.TrimSpace(item.Name) == "" {
			return errors.New("item id and name are required")
		}
		if item.ID != "" && item.ID != id {
			return errors.New("item id mismatch: " + id)
		}
		if item.MaxStack <= 0 {
			return errors.New("item maxStack must be positive: " + id)
		}
		if item.Usable && len(item.Effects) == 0 {
			return errors.New("usable item requires effects: " + id)
		}
		for _, effect := range item.Effects {
			switch strings.TrimSpace(effect.Type) {
			case "general_exp":
				if effect.Amount <= 0 {
					return errors.New("general_exp amount must be positive: " + id)
				}
			case "resources":
				if len(effect.Resources) == 0 {
					return errors.New("resources effect requires resources: " + id)
				}
				for key, value := range effect.Resources {
					if !isCoreResourceType(key) || value <= 0 {
						return errors.New("invalid resource effect: " + id)
					}
				}
			case "unit_by_faction":
				if effect.Amount <= 0 || len(effect.UnitByFaction) == 0 {
					return errors.New("unit_by_faction effect requires amount and unit map: " + id)
				}
			default:
				return errors.New("invalid item effect type: " + id)
			}
		}
	}
	return nil
}
