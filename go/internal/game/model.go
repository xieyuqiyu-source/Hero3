package game

import "time"

type Player struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
	Faction  string `json:"faction"`
}

type ResourceState struct {
	Wood     int `json:"wood"`
	Stone    int `json:"stone"`
	Iron     int `json:"iron"`
	Food     int `json:"food"`
	Capacity int `json:"capacity"`
}

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
	Player              Player         `json:"player"`
	Resources           ResourceState  `json:"resources"`
	Buildings           []Building     `json:"buildings"`
	Army                []ArmyUnit     `json:"army"`
	RecruitQueues       []RecruitQueue `json:"recruitQueues"`
	MapTargets          []MapTarget    `json:"mapTargets"`
	RecentBattleReports []BattleReport `json:"recentBattleReports"`
	UnreadMessageCount  int            `json:"unreadMessageCount"`
	ServerTime          string         `json:"serverTime"`
}

func newDemoState(now time.Time) GameState {
	return GameState{
		Player: Player{
			ID:       "demo-player",
			Nickname: "主公",
			Faction:  "wei",
		},
		Resources: ResourceState{
			Wood:     1200,
			Stone:    900,
			Iron:     600,
			Food:     1500,
			Capacity: 5000,
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
		ServerTime:          now.UTC().Format(time.RFC3339),
	}
}
