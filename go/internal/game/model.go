package game

import (
	"encoding/json"
	"time"
)

type Account struct {
	ID           string
	Username     string
	PasswordHash string
	CreatedAt    time.Time
}

type PlayerSummary struct {
	ID        string `json:"id"`
	Nickname  string `json:"nickname"`
	Faction   string `json:"faction"`
	UpdatedAt string `json:"updatedAt"`
}

type AccountSummary struct {
	ID        string          `json:"id"`
	Username  string          `json:"username"`
	CreatedAt string          `json:"createdAt"`
	Players   []PlayerSummary `json:"players"`
}

type Player struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
	Faction  string `json:"faction"`
}

type ResourceState struct {
	Items    map[string]int `json:"items"`
	Capacity map[string]int `json:"capacity"`
}

type ResourceProduction map[string]int

type Building struct {
	ID            string  `json:"id"`
	Type          string  `json:"type"`
	Level         int     `json:"level"`
	UpgradeEndsAt *string `json:"upgradeEndsAt"`
}

type ArmyUnit struct {
	UnitType string `json:"unitType"`
	Amount   int    `json:"amount"`
}

type RecruitQueue struct {
	ID       string `json:"id"`
	UnitType string `json:"unitType"`
	Amount   int    `json:"amount"`
	EndsAt   string `json:"endsAt"`
	Status   string `json:"status"`
}

type MapTarget struct {
	ID      string         `json:"id"`
	Type    string         `json:"type"`
	Level   int            `json:"level"`
	Power   int            `json:"power"`
	Rewards map[string]int `json:"rewards"`
}

type BattleReport struct {
	ID          string         `json:"id"`
	TargetID    string         `json:"targetId"`
	Result      string         `json:"result"`
	PlayerPower int            `json:"playerPower"`
	EnemyPower  int            `json:"enemyPower"`
	LostUnits   map[string]int `json:"lostUnits"`
	Rewards     map[string]int `json:"rewards"`
	CreatedAt   string         `json:"createdAt"`
}

type GameState struct {
	Player              Player             `json:"player"`
	Resources           ResourceState      `json:"resources"`
	ResourceProduction  ResourceProduction `json:"resourceProduction"`
	ResourceSettledAt   string             `json:"resourceSettledAt"`
	Buildings           []Building         `json:"buildings"`
	Army                []ArmyUnit         `json:"army"`
	RecruitQueues       []RecruitQueue     `json:"recruitQueues"`
	MapTargets          []MapTarget        `json:"mapTargets"`
	RecentBattleReports []BattleReport     `json:"recentBattleReports"`
	UnreadMessageCount  int                `json:"unreadMessageCount"`
	ServerTime          string             `json:"serverTime"`
}

func newPlayerState(id string, nickname string, faction string, now time.Time) GameState {
	state := GameState{
		Player: Player{
			ID:       id,
			Nickname: nickname,
			Faction:  faction,
		},
		Resources: ResourceState{
			Items: map[string]int{
				"wood":  1200,
				"stone": 900,
				"iron":  600,
				"food":  1500,
			},
			Capacity: map[string]int{},
		},
		Buildings: []Building{
			{ID: "wood-camp", Type: "wood_camp", Level: 3},
			{ID: "stone-quarry", Type: "stone_quarry", Level: 2},
			{ID: "iron-mine", Type: "iron_mine", Level: 2},
			{ID: "farm", Type: "farm", Level: 3},
			{ID: "warehouse", Type: "warehouse", Level: 1},
			{ID: "barracks", Type: "barracks", Level: 1},
			{ID: "archery-range", Type: "archery_range", Level: 1},
		},
		Army: []ArmyUnit{
			{UnitType: "infantry", Amount: 80},
			{UnitType: "archer", Amount: 30},
			{UnitType: "cavalry", Amount: 0},
		},
		RecruitQueues: []RecruitQueue{},
		MapTargets: []MapTarget{
			{
				ID:    "target-001",
				Type:  "bandit_camp",
				Level: 1,
				Power: 100,
				Rewards: map[string]int{
					"wood":  120,
					"stone": 80,
					"food":  100,
				},
			},
			{
				ID:    "target-002",
				Type:  "mountain_fort",
				Level: 2,
				Power: 220,
				Rewards: map[string]int{
					"wood":  180,
					"stone": 160,
					"iron":  90,
					"food":  140,
				},
			},
		},
		RecentBattleReports: []BattleReport{},
		UnreadMessageCount:  0,
		ResourceSettledAt:   now.UTC().Format(time.RFC3339),
		ServerTime:          now.UTC().Format(time.RFC3339),
	}

	state.ResourceProduction = calculateResourceProduction(state.Buildings)
	state.Resources.Capacity = calculateResourceCapacity(state.Buildings)
	return state
}

func newDemoState(now time.Time) GameState {
	return newPlayerState("demo-player", "主公", "wei", now)
}

func (r *ResourceState) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	next := ResourceState{
		Items:    map[string]int{},
		Capacity: map[string]int{},
	}
	if value, exists := raw["items"]; exists {
		if err := json.Unmarshal(value, &next.Items); err != nil {
			return err
		}
	}
	if value, exists := raw["capacity"]; exists {
		if err := json.Unmarshal(value, &next.Capacity); err == nil {
			*r = next
			return nil
		}

		var legacyCapacity int
		if err := json.Unmarshal(value, &legacyCapacity); err != nil {
			return err
		}
		for _, resourceType := range coreResourceTypes() {
			next.Capacity[resourceType] = legacyCapacity
		}
	}

	if len(next.Items) == 0 {
		for _, resourceType := range coreResourceTypes() {
			var amount int
			if value, exists := raw[resourceType]; exists {
				if err := json.Unmarshal(value, &amount); err != nil {
					return err
				}
			}
			next.Items[resourceType] = amount
		}
	}

	*r = next
	return nil
}

func (p *ResourceProduction) UnmarshalJSON(data []byte) error {
	var values map[string]int
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}

	next := ResourceProduction{}
	for key, value := range values {
		switch key {
		case "woodPerHour":
			next["wood"] = value
		case "stonePerHour":
			next["stone"] = value
		case "ironPerHour":
			next["iron"] = value
		case "foodPerHour":
			next["food"] = value
		default:
			next[key] = value
		}
	}

	*p = next
	return nil
}
