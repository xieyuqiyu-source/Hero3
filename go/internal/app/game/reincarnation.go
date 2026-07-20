// 本文件定义轮回绝境副本的领域模型和接口请求响应。
package game

import "time"

const (
	ReincarnationModuleID  = "reincarnation_abyss"
	ReincarnationWaveCount = 18

	ReincarnationRunRunning   = "running"
	ReincarnationRunCompleted = "completed"
	ReincarnationRunFailed    = "failed"
	ReincarnationRunExpired   = "expired"
	ReincarnationRunRewarded  = "rewarded"

	ReincarnationWaveLocked  = "locked"
	ReincarnationWaveActive  = "active"
	ReincarnationWaveCleared = "cleared"
	ReincarnationWaveFailed  = "failed"
	ReincarnationWaveExpired = "expired"

	ReincarnationWaveAttack  = "attack"
	ReincarnationWaveDefense = "defense"
)

type ReincarnationRun struct {
	ID              string                `json:"id"`
	PlayerID        string                `json:"playerId"`
	Level           int                   `json:"level"`
	LevelName       string                `json:"levelName"`
	Status          string                `json:"status"`
	CurrentWave     int                   `json:"currentWave"`
	StartedAt       time.Time             `json:"startedAt"`
	ExpiresAt       time.Time             `json:"expiresAt"`
	CompletedAt     *time.Time            `json:"completedAt,omitempty"`
	FailedAt        *time.Time            `json:"failedAt,omitempty"`
	EndedReason     string                `json:"endedReason,omitempty"`
	PendingRewards  []Reward              `json:"pendingRewards"`
	RewardGrantedAt *time.Time            `json:"rewardGrantedAt,omitempty"`
	ExitedAt        *time.Time            `json:"exitedAt,omitempty"`
	Waves           []ReincarnationWave   `json:"waves"`
	Battles         []ReincarnationBattle `json:"battles,omitempty"`
	CreatedAt       time.Time             `json:"createdAt"`
	UpdatedAt       time.Time             `json:"updatedAt"`
}

type ReincarnationWave struct {
	ID               string             `json:"id"`
	RunID            string             `json:"runId"`
	WaveIndex        int                `json:"waveIndex"`
	WaveType         string             `json:"waveType"`
	EnemyFaction     string             `json:"enemyFaction"`
	EnemyTroops      map[string]int     `json:"enemyTroops"`
	EnemyRemaining   map[string]int     `json:"enemyRemaining"`
	AllyBonus        ReincarnationBonus `json:"allyBonus"`
	EnemyBonus       ReincarnationBonus `json:"enemyBonus"`
	RewardPreview    []Reward           `json:"rewardPreview"`
	RewardResult     []Reward           `json:"rewardResult,omitempty"`
	TroopCap         int                `json:"troopCap"`
	Status           string             `json:"status"`
	StartedAt        time.Time          `json:"startedAt"`
	ClearedAt        *time.Time         `json:"clearedAt,omitempty"`
	DefenseReadyAt   *time.Time         `json:"defenseReadyAt,omitempty"`
	DefenseResolveAt *time.Time         `json:"defenseResolveAt,omitempty"`
}

type ReincarnationBonus struct {
	Side     string  `json:"side"`
	UnitType string  `json:"unitType"`
	Stat     string  `json:"stat"`
	Value    float64 `json:"value"`
	Label    string  `json:"label"`
	UnitName string  `json:"unitName,omitempty"`
	Faction  string  `json:"faction,omitempty"`
}

type ReincarnationBattle struct {
	ID             string                        `json:"id"`
	RunID          string                        `json:"runId"`
	WaveID         string                        `json:"waveId"`
	PlayerID       string                        `json:"playerId"`
	ClientActionID string                        `json:"clientActionId,omitempty"`
	WaveIndex      int                           `json:"waveIndex"`
	WaveType       string                        `json:"waveType"`
	AttackTroops   map[string]int                `json:"attackTroops"`
	Losses         map[string]int                `json:"losses"`
	RevivedUnits   map[string]int                `json:"revivedUnits,omitempty"`
	SurvivedTroops map[string]int                `json:"survivedTroops,omitempty"`
	EnemyLosses    map[string]int                `json:"enemyLosses"`
	EnemyCaptured  map[string]int                `json:"enemyCaptured,omitempty"`
	EnemyRemaining map[string]int                `json:"enemyRemaining,omitempty"`
	TraitOutcomes  map[string]TraitOutcomeReport `json:"traitOutcomes,omitempty"`
	Passed         bool                          `json:"passed"`
	ReportID       string                        `json:"reportId"`
	CreatedAt      time.Time                     `json:"createdAt"`
}

type ReincarnationRunResponse struct {
	Run        *ReincarnationRun `json:"run,omitempty"`
	Army       []ArmyUnit        `json:"army,omitempty"`
	ServerTime string            `json:"serverTime"`
}

type ReincarnationActionResult struct {
	Run            ReincarnationRun     `json:"run"`
	BattleReport   *BattleReport        `json:"battleReport,omitempty"`
	Army           []ArmyUnit           `json:"army,omitempty"`
	Inventory      map[string]ItemStack `json:"inventory,omitempty"`
	InventorySlots []ItemStack          `json:"inventorySlots,omitempty"`
	General        *General             `json:"general,omitempty"`
	Generals       []General            `json:"generals,omitempty"`
	AccountGold    int                  `json:"accountGold,omitempty"`
	Cost           int                  `json:"cost,omitempty"`
	ServerTime     string               `json:"serverTime"`
}

type StartReincarnationRequest struct {
	PlayerID string `json:"playerId"`
	Level    int    `json:"level"`
}

type ReincarnationTroopRequest struct {
	PlayerID       string         `json:"playerId"`
	Troops         map[string]int `json:"troops"`
	GeneralIDs     []string       `json:"generalIds,omitempty"`
	ClientActionID string         `json:"clientActionId,omitempty"`
}

type ReincarnationBonusResetRequest struct {
	PlayerID string `json:"playerId"`
}

type ReincarnationExitRequest struct {
	PlayerID string `json:"playerId"`
	RunID    string `json:"runId"`
}

type ReincarnationExitResult struct {
	RunID      string `json:"runId"`
	ExitedAt   string `json:"exitedAt"`
	ServerTime string `json:"serverTime"`
}
