package game

import (
	"encoding/json"
	"time"
)

// FlexInt 兼容 JSON 中 int 和 float 的整数类型（MySQL JSON 列可能存为 float）
type FlexInt int

func (fi *FlexInt) UnmarshalJSON(data []byte) error {
	var f float64
	if err := json.Unmarshal(data, &f); err != nil {
		return err
	}
	*fi = FlexInt(int(f))
	return nil
}

func (fi FlexInt) MarshalJSON() ([]byte, error) {
	return json.Marshal(int(fi))
}

type Account struct {
	ID           string
	Username     string
	PasswordHash string
	Gold         int // 账户级金币（充值/活动获得，可兑换为城金）
	CreatedAt    time.Time
}

type PlayerSummary struct {
	ID            string `json:"id"`
	Nickname      string `json:"nickname"`
	Faction       string `json:"faction"`
	TotalArmy     int    `json:"totalArmy"`
	BuildingLevel int    `json:"buildingLevel"`
	UpdatedAt     string `json:"updatedAt"`
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

type General struct {
	ID    string             `json:"id"`
	Name  string             `json:"name"`
	Level int                `json:"level"`
	Exp   int                `json:"exp"`
	Buffs map[string]float64 `json:"buffs"`
}

type RecruitQueue struct {
	ID       string `json:"id"`
	UnitType string `json:"unitType"`
	Amount   int    `json:"amount"`
	EndsAt   string `json:"endsAt"`
}

type MapTarget struct {
	ID      string         `json:"id"`
	Type    string         `json:"type"`
	Level   int            `json:"level"`
	Power   int            `json:"power"`
	Rewards map[string]int `json:"rewards"`
}

type BattleReport struct {
	ID                string         `json:"id"`
	PlayerID          string         `json:"playerId"`
	PlayerFaction     string         `json:"playerFaction"`
	PlayerName        string         `json:"playerName,omitempty"`
	TargetID          string         `json:"targetId"`
	TargetName        string         `json:"targetName"`
	Type              string         `json:"type"` // "attack", "plunder", "scout", "reinforce"
	Result            string         `json:"result"`
	PlayerPower       int            `json:"playerPower"`
	EnemyPower        int            `json:"enemyPower"`
	DispatchedUnits   map[string]int `json:"dispatchedUnits"`
	LostUnits         map[string]int `json:"lostUnits"`
	DefenderFaction   string         `json:"defenderFaction"`
	DefenderUnits     map[string]int `json:"defenderUnits"`
	DefenderLostUnits map[string]int `json:"defenderLostUnits"`
	DefenderRevealed  bool           `json:"defenderRevealed"`
	DefenderResources map[string]int `json:"defenderResources"`
	Rewards           map[string]int `json:"rewards"`
	Overflow          map[string]int `json:"overflow,omitempty"`    // 各资源溢出量
	OverflowCityGold  int            `json:"overflowCityGold"`     // 溢出转换获得的城金
	Read              bool           `json:"read"`
	DeletedByPlayer   bool           `json:"deletedByPlayer,omitempty"`
	CreatedAt         string         `json:"createdAt"`
}

type GameState struct {
	Player              Player             `json:"player"`
	Resources           ResourceState      `json:"resources"`
	ResourceProduction  ResourceProduction `json:"resourceProduction"`
	ResourceSettledAt   string             `json:"resourceSettledAt"`
	CityGold            FlexInt            `json:"cityGold"`
	LastExchangeAt      string             `json:"lastExchangeAt,omitempty"`
	ProductionBoost     int                `json:"productionBoost,omitempty"`          // 产量加成倍率（1=无，2/4/8/16）
	ProductionBoostEnd  string             `json:"productionBoostEnd,omitempty"`       // 加成到期时间
	CapacityBoost       int                `json:"capacityBoost,omitempty"`            // 仓库容量加成倍率（1=无，2/4/8/16）
	CapacityBoostEnd    string             `json:"capacityBoostEnd,omitempty"`         // 容量加成到期时间
	Buildings           []Building         `json:"buildings"`
	General             *General           `json:"general"`
	Army                []ArmyUnit         `json:"army"`
	RecruitQueues       []RecruitQueue     `json:"recruitQueues"`
	NpcState            *NpcState          `json:"npcState,omitempty"`
	MapTargets          []MapTarget        `json:"mapTargets"`
	RecentBattleReports []BattleReport     `json:"recentBattleReports"`
	UnreadMessageCount  int                `json:"unreadMessageCount"`
	ServerTime          string             `json:"serverTime"`
}

func newPlayerState(id string, nickname string, faction string, generalID string, now time.Time) GameState {
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
			// 木场 x5
			{ID: "wood_camp-1", Type: "wood_camp", Level: 1},
			{ID: "wood_camp-2", Type: "wood_camp", Level: 1},
			{ID: "wood_camp-3", Type: "wood_camp", Level: 1},
			{ID: "wood_camp-4", Type: "wood_camp", Level: 1},
			{ID: "wood_camp-5", Type: "wood_camp", Level: 1},
			// 采石场 x5
			{ID: "stone_quarry-1", Type: "stone_quarry", Level: 1},
			{ID: "stone_quarry-2", Type: "stone_quarry", Level: 1},
			{ID: "stone_quarry-3", Type: "stone_quarry", Level: 1},
			{ID: "stone_quarry-4", Type: "stone_quarry", Level: 1},
			{ID: "stone_quarry-5", Type: "stone_quarry", Level: 1},
			// 铁矿 x5
			{ID: "iron_mine-1", Type: "iron_mine", Level: 1},
			{ID: "iron_mine-2", Type: "iron_mine", Level: 1},
			{ID: "iron_mine-3", Type: "iron_mine", Level: 1},
			{ID: "iron_mine-4", Type: "iron_mine", Level: 1},
			{ID: "iron_mine-5", Type: "iron_mine", Level: 1},
			// 农田 x5
			{ID: "farm-1", Type: "farm", Level: 1},
			{ID: "farm-2", Type: "farm", Level: 1},
			{ID: "farm-3", Type: "farm", Level: 1},
			{ID: "farm-4", Type: "farm", Level: 1},
			{ID: "farm-5", Type: "farm", Level: 1},
			// 功能建筑（各一块）
			{ID: "warehouse-1", Type: "warehouse", Level: 1},
			{ID: "infantry_camp-1", Type: "infantry_camp", Level: 1},
			{ID: "cavalry_camp-1", Type: "cavalry_camp", Level: 1},
		},
		Army:          []ArmyUnit{},
		General:       newGeneral(faction, generalID),
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

	state.ResourceProduction = calculateResourceProduction(state.Buildings, state.General)
	state.Resources.Capacity = calculateResourceCapacity(state.Buildings)
	return state
}

func newDemoState(now time.Time) GameState {
	return newPlayerState("demo-player", "主公", "wei", "caocao", now)
}

func newGeneral(faction string, generalID string) *General {
	if generalID == "" {
		return nil
	}
	name := generalID // fallback
	factions := GetFactionsConfig()
	if fc, ok := factions[faction]; ok {
		for _, g := range fc.Generals {
			if g.ID == generalID {
				name = g.Name
				break
			}
		}
	}
	return &General{
		ID:    generalID,
		Name:  name,
		Level: 1,
		Exp:   0,
		Buffs: map[string]float64{},
	}
}

func buildPlayerSummary(state GameState, updatedAt time.Time) PlayerSummary {
	totalArmy := 0
	for _, unit := range state.Army {
		totalArmy += unit.Amount
	}
	buildingLevel := 0
	for _, b := range state.Buildings {
		buildingLevel += b.Level
	}
	return PlayerSummary{
		ID:            state.Player.ID,
		Nickname:      state.Player.Nickname,
		Faction:       state.Player.Faction,
		TotalArmy:     totalArmy,
		BuildingLevel: buildingLevel,
		UpdatedAt:     updatedAt.UTC().Format(time.RFC3339),
	}
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
