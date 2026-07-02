package game

import (
	"encoding/json"
	"errors"
	"os"
	"regexp"
	"strings"
	"sync"
)

type ItemsConfig map[string]ItemDefinition

const (
	InventoryCapacity = 9999

	ItemQualityCommon    = "common"
	ItemQualityRare      = "rare"
	ItemQualityEpic      = "epic"
	ItemQualityLegendary = "legendary"
	ItemQualityMythic    = "mythic"

	ItemConfirmAuto   = "auto"
	ItemConfirmAlways = "always"
	ItemConfirmNever  = "never"
)

type ItemDefinition struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Category     string                 `json:"category"`
	Quality      string                 `json:"quality"`
	Type         string                 `json:"type,omitempty"`
	Rarity       string                 `json:"rarity,omitempty"`
	Icon         string                 `json:"icon,omitempty"`
	Usable       bool                   `json:"usable"`
	Stackable    bool                   `json:"stackable"`
	MaxStack     int                    `json:"maxStack"`
	BindType     string                 `json:"bindType,omitempty"`
	UseTarget    string                 `json:"useTarget"`
	ConfirmOnUse string                 `json:"confirmOnUse,omitempty"`
	Effects      []ItemEffect           `json:"effects"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

type ItemEffect struct {
	Type            string            `json:"type"`
	ID              string            `json:"id,omitempty"`
	Amount          int               `json:"amount,omitempty"`
	Category        string            `json:"category,omitempty"`
	Pool            string            `json:"pool,omitempty"`
	GeneralID       string            `json:"generalId,omitempty"`
	Resources       map[string]int    `json:"resources,omitempty"`
	UnitByFaction   map[string]string `json:"unitByFaction,omitempty"`
	ProtectionType  string            `json:"protectionType,omitempty"`
	DurationSeconds int               `json:"durationSeconds,omitempty"`
	CurrencyType    string            `json:"currencyType,omitempty"`
	BuffKey         string            `json:"buffKey,omitempty"`
	BuffMode        string            `json:"buffMode,omitempty"`
	BuffValue       float64           `json:"buffValue,omitempty"`
	DropPoolID      string            `json:"dropPoolId,omitempty"`
}

var (
	itemsMu       sync.RWMutex
	itemsConfig   ItemsConfig = ItemsConfig{}
	itemIDPattern             = regexp.MustCompile(`^[a-z0-9_]+$`)
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
	itemsConfig = normalizeItemsConfig(cfg)
	itemsMu.Unlock()
	return nil
}

// SaveItemsConfig 校验并持久化物品配置。
func SaveItemsConfig(path string, cfg ItemsConfig) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("物品配置路径不能为空")
	}
	normalized := normalizeItemsConfig(cfg)
	if err := ValidateItemsConfig(normalized); err != nil {
		return err
	}
	data, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return err
	}
	itemsMu.Lock()
	itemsConfig = normalized
	itemsMu.Unlock()
	return nil
}

func ValidateItemsConfig(cfg ItemsConfig) error {
	if cfg == nil {
		return errors.New("物品配置不能为空")
	}
	cfg = normalizeItemsConfig(cfg)
	for id, item := range cfg {
		if strings.TrimSpace(id) == "" {
			return errors.New("物品 ID 不能为空")
		}
		if !itemIDPattern.MatchString(id) {
			return errors.New("物品 ID 只能包含小写英文、数字和下划线: " + id)
		}
		if strings.TrimSpace(item.Name) == "" {
			return errors.New("物品名称不能为空: " + id)
		}
		if item.ID != "" && item.ID != id {
			return errors.New("物品 ID 和配置 key 不一致: " + id)
		}
		if !validItemCategory(item.Category) {
			return errors.New("物品分类不合法: " + id)
		}
		if !validItemQuality(item.Quality) {
			return errors.New("物品品质不合法: " + id)
		}
		if !validItemConfirm(item.ConfirmOnUse) {
			return errors.New("物品使用确认规则不合法: " + id)
		}
		if item.MaxStack <= 0 {
			return errors.New("物品堆叠上限必须大于 0: " + id)
		}
		if !item.Stackable && item.MaxStack != 1 {
			return errors.New("不可堆叠物品 maxStack 必须为 1: " + id)
		}
		if item.Usable && len(item.Effects) == 0 {
			return errors.New("可使用物品必须配置效果: " + id)
		}
		if !item.Usable && len(item.Effects) > 0 && !itemCategoryAllowsPassiveEffects(item.Category) {
			return errors.New("不可使用物品不能配置普通使用效果: " + id)
		}
		for _, effect := range item.Effects {
			switch strings.TrimSpace(effect.Type) {
			case "item":
				if _, ok := cfg[strings.TrimSpace(effect.ID)]; !ok {
					return errors.New("物品效果引用的物品不存在: " + id)
				}
				if effect.Amount <= 0 {
					return errors.New("物品效果数量必须大于 0: " + id)
				}
			case "general":
				if strings.TrimSpace(effect.GeneralID) == "" {
					return errors.New("武将效果必须配置 generalId: " + id)
				}
				hero, ok := GetHeroConfig(strings.TrimSpace(effect.GeneralID))
				if !ok || !hero.Enabled {
					return errors.New("武将效果引用的武将不存在或未启用: " + id)
				}
			case "general_exp":
				if effect.Amount <= 0 {
					return errors.New("武将经验效果数量必须大于 0: " + id)
				}
			case "resources":
				if len(effect.Resources) == 0 {
					return errors.New("资源效果必须配置资源: " + id)
				}
				for key, value := range effect.Resources {
					if !isCoreResourceType(key) || value <= 0 {
						return errors.New("资源效果配置不合法: " + id)
					}
				}
			case "unit_by_faction":
				if effect.Amount <= 0 || len(effect.UnitByFaction) == 0 {
					return errors.New("按阵营发兵效果必须配置数量和兵种映射: " + id)
				}
				for _, faction := range []string{"wei", "shu", "wu"} {
					if strings.TrimSpace(effect.UnitByFaction[faction]) == "" {
						return errors.New("按阵营发兵效果必须覆盖魏蜀吴: " + id)
					}
				}
			case "random_unit_by_faction_category", "all_units_by_faction_category":
				if effect.Amount <= 0 {
					return errors.New("分类发兵效果数量必须大于 0: " + id)
				}
				if !validRecruitUnitCategory(effect.Category) {
					return errors.New("分类发兵效果兵种分类不合法: " + id)
				}
				if !validRecruitUnitPool(effect.Pool) {
					return errors.New("分类发兵效果兵种池不合法: " + id)
				}
			case "pvp_protection":
				if effect.DurationSeconds <= 0 {
					return errors.New("免战效果必须配置正数持续时间: " + id)
				}
				switch strings.TrimSpace(effect.ProtectionType) {
				case PvpProtectionTypeManual, PvpProtectionTypeSystem, PvpProtectionTypeMaintenance:
				default:
					return errors.New("免战保护类型不合法: " + id)
				}
			case "currency":
				switch strings.TrimSpace(effect.CurrencyType) {
				case RewardTypeCityGold, RewardTypeGold:
				default:
					return errors.New("货币效果类型不合法: " + id)
				}
				if effect.Amount <= 0 {
					return errors.New("货币效果数量必须大于 0: " + id)
				}
			case "buff":
				if effect.Amount <= 0 || strings.TrimSpace(effect.BuffKey) == "" || strings.TrimSpace(effect.BuffMode) == "" {
					return errors.New("Buff 效果必须配置 key、mode 和持续时间: " + id)
				}
				if err := validateBuffModifierSpec(effect.BuffKey, effect.BuffMode); err != nil {
					return errors.New("Buff 效果配置不合法: " + id)
				}
			case "random_reward":
				if strings.TrimSpace(effect.DropPoolID) == "" {
					return errors.New("随机奖励效果必须引用掉落池: " + id)
				}
				if pools := GetDropPoolsConfig(); len(pools) > 0 {
					if _, ok := pools[strings.TrimSpace(effect.DropPoolID)]; !ok {
						return errors.New("随机奖励引用的掉落池不存在: " + id)
					}
				}
			default:
				return errors.New("物品效果类型不合法: " + id)
			}
		}
	}
	return nil
}

// normalizeItemsConfig 补齐旧物品配置字段和默认值。
func normalizeItemsConfig(cfg ItemsConfig) ItemsConfig {
	next := make(ItemsConfig, len(cfg))
	for id, item := range cfg {
		id = strings.TrimSpace(id)
		item.ID = firstNonEmpty(item.ID, id)
		item.Category = firstNonEmpty(item.Category, item.Type)
		if !validItemCategory(item.Category) {
			item.Category = inferItemCategory(id, item)
		}
		item.Quality = firstNonEmpty(item.Quality, item.Rarity, ItemQualityCommon)
		item.ConfirmOnUse = firstNonEmpty(item.ConfirmOnUse, ItemConfirmAuto)
		item.BindType = firstNonEmpty(item.BindType, "bound")
		if item.UseTarget == "" {
			item.UseTarget = "self"
		}
		if item.MaxStack <= 0 {
			item.MaxStack = 1
		}
		if !item.Stackable {
			item.MaxStack = 1
		}
		next[id] = item
	}
	return next
}

func inferItemCategory(id string, item ItemDefinition) string {
	id = strings.TrimSpace(id)
	if strings.Contains(id, "general_exp") {
		return "general_exp"
	}
	if strings.Contains(id, "resource_pack") {
		return "resource_pack"
	}
	if strings.Contains(id, "recruit_ticket") {
		return "recruit_ticket"
	}
	if strings.HasPrefix(id, "pvp_") {
		return "pvp_item"
	}
	for _, effect := range item.Effects {
		switch strings.TrimSpace(effect.Type) {
		case "general_exp":
			return "general_exp"
		case "resources":
			return "resource_pack"
		case "unit_by_faction":
			return "recruit_ticket"
		case "random_unit_by_faction_category", "all_units_by_faction_category":
			return "recruit_ticket"
		case "pvp_protection":
			return "pvp_item"
		case "currency":
			return "currency_pack"
		case "buff":
			return "buff_item"
		}
	}
	if strings.TrimSpace(item.Type) == "token" {
		return "token"
	}
	return strings.TrimSpace(item.Category)
}

// ItemRequiresUseConfirm 判断物品使用时是否需要二次确认。
func ItemRequiresUseConfirm(item ItemDefinition) bool {
	switch strings.TrimSpace(item.ConfirmOnUse) {
	case ItemConfirmAlways:
		return true
	case ItemConfirmNever:
		return false
	}
	return itemQualityRank(item.Quality) >= itemQualityRank(ItemQualityEpic)
}

func validItemCategory(category string) bool {
	switch strings.TrimSpace(category) {
	case "resource_pack", "general_exp", "recruit_ticket", "buff_item", "pvp_item", "ticket", "material", "chest", "token", "currency_pack", "event_item", "equipment", "equipment_set":
		return true
	default:
		return false
	}
}

func validItemQuality(quality string) bool {
	return itemQualityRank(quality) > 0
}

func validItemConfirm(confirm string) bool {
	switch strings.TrimSpace(confirm) {
	case "", ItemConfirmAuto, ItemConfirmAlways, ItemConfirmNever:
		return true
	default:
		return false
	}
}

func itemQualityRank(quality string) int {
	switch strings.TrimSpace(quality) {
	case ItemQualityCommon:
		return 1
	case ItemQualityRare:
		return 2
	case ItemQualityEpic:
		return 3
	case ItemQualityLegendary:
		return 4
	case ItemQualityMythic:
		return 5
	default:
		return 0
	}
}

func itemCategoryAllowsPassiveEffects(category string) bool {
	switch strings.TrimSpace(category) {
	case "equipment", "equipment_set":
		return true
	default:
		return false
	}
}
