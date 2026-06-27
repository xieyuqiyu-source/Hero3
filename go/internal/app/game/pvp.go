// 本文件定义 PVP 系统的领域模型、状态和请求响应结构。
package game

const (
	PVPModuleID = "pvp"

	PvpMarchTypeAttack  = "attack"
	PvpMarchTypePlunder = "plunder"

	PvpMarchStatusMarching  = "marching"
	PvpMarchStatusReturning = "returning"
	PvpMarchStatusResolving = "resolving"
	PvpMarchStatusResolved  = "resolved"
	PvpMarchStatusRecalled  = "recalled"
	PvpMarchStatusCancelled = "cancelled"
	PvpMarchStatusFailed    = "failed"

	PvpBattleStatusCreated  = "created"
	PvpBattleStatusResolved = "resolved"

	defaultPvpMarchSeconds         = 3 * 3600
	pvpAccelerateFixedCityGoldCost = 10
	defaultPvpDailyAttackLimit     = 30
	defaultPvpAttackCooldownSec    = 30
	defaultPvpDefeatProtectSec     = 600
	defaultPvpRevengeExpireSec     = 72 * 3600

	PvpProtectionTypeNewbie      = "newbie"
	PvpProtectionTypeDefeat      = "defeat"
	PvpProtectionTypeManual      = "manual"
	PvpProtectionTypeSystem      = "system"
	PvpProtectionTypeMaintenance = "maintenance"

	PvpSeasonStatusActive  = "active"
	PvpSeasonStatusSettled = "settled"

	PvpSeasonRewardMailType = "pvp_season_reward"
)

// PvpTargetSummary 是 PVP 目标列表中的玩家摘要。
type PvpTargetSummary struct {
	PlayerID       string `json:"playerId"`
	Nickname       string `json:"nickname"`
	Faction        string `json:"faction"`
	TotalArmy      int    `json:"totalArmy"`
	BuildingLevel  int    `json:"buildingLevel"`
	CanAttack      bool   `json:"canAttack"`
	CanReinforce   bool   `json:"canReinforce"`
	Protected      bool   `json:"protected"`
	ProtectedUntil string `json:"protectedUntil,omitempty"`
	CooldownUntil  string `json:"cooldownUntil,omitempty"`
	Reason         string `json:"reason,omitempty"`
}

// PvpPlayerState 保存玩家 PVP 保护、冷却和每日次数状态。
type PvpPlayerState struct {
	PlayerID         string            `json:"playerId"`
	Status           string            `json:"status"`
	ProtectionType   string            `json:"protectionType,omitempty"`
	ProtectedUntil   string            `json:"protectedUntil,omitempty"`
	CooldownUntil    string            `json:"cooldownUntil,omitempty"`
	DailyAttackCount int               `json:"dailyAttackCount"`
	DailyAttackLimit int               `json:"dailyAttackLimit"`
	DailyResetAt     string            `json:"dailyResetAt,omitempty"`
	TargetCooldown   map[string]string `json:"targetCooldown,omitempty"`
	Metadata         map[string]any    `json:"metadata,omitempty"`
	CreatedAt        string            `json:"createdAt"`
	UpdatedAt        string            `json:"updatedAt"`
}

// PvpTargetsResponse 是 PVP 目标列表响应。
type PvpTargetsResponse struct {
	Items []PvpTargetSummary `json:"items"`
}

// PvpMarch 记录一次 PVP 行军生命周期。
type PvpMarch struct {
	ID               string         `json:"id"`
	AttackerPlayerID string         `json:"attackerPlayerId"`
	AttackerName     string         `json:"attackerName"`
	AttackerFaction  string         `json:"attackerFaction"`
	DefenderPlayerID string         `json:"defenderPlayerId"`
	DefenderName     string         `json:"defenderName"`
	DefenderFaction  string         `json:"defenderFaction"`
	MarchType        string         `json:"marchType"`
	Status           string         `json:"status"`
	AttackTroops     map[string]int `json:"attackTroops"`
	AttackGenerals   []string       `json:"attackGenerals,omitempty"`
	SpeedMultiplier  float64        `json:"speedMultiplier"`
	DurationSeconds  int            `json:"durationSeconds"`
	StartedAt        string         `json:"startedAt"`
	ArrivesAt        string         `json:"arrivesAt"`
	ReturnStartedAt  string         `json:"returnStartedAt,omitempty"`
	ReturnsAt        string         `json:"returnsAt,omitempty"`
	ResolvedAt       string         `json:"resolvedAt,omitempty"`
	AttackerReportID string         `json:"attackerReportId,omitempty"`
	DefenderReportID string         `json:"defenderReportId,omitempty"`
	BattleID         string         `json:"battleId,omitempty"`
	AcceleratedTimes int            `json:"acceleratedTimes"`
	CreatedAt        string         `json:"createdAt"`
	UpdatedAt        string         `json:"updatedAt"`
}

// PvpBattle 记录一次 PVP 战斗结算结果。
type PvpBattle struct {
	ID                    string                     `json:"id"`
	MarchID               string                     `json:"marchId"`
	AttackerPlayerID      string                     `json:"attackerPlayerId"`
	DefenderPlayerID      string                     `json:"defenderPlayerId"`
	Status                string                     `json:"status"`
	AttackerSnapshot      map[string]any             `json:"attackerSnapshot,omitempty"`
	DefenderSnapshot      map[string]any             `json:"defenderSnapshot,omitempty"`
	ReinforcementSnapshot []DefenseReinforcementUnit `json:"reinforcementSnapshot,omitempty"`
	Result                map[string]any             `json:"result,omitempty"`
	Losses                map[string]any             `json:"losses,omitempty"`
	Plunder               map[string]int             `json:"plunder,omitempty"`
	AttackerReportID      string                     `json:"attackerReportId,omitempty"`
	DefenderReportID      string                     `json:"defenderReportId,omitempty"`
	ResolvedAt            string                     `json:"resolvedAt,omitempty"`
	CreatedAt             string                     `json:"createdAt"`
	UpdatedAt             string                     `json:"updatedAt"`
}

// PvpGeneralSnapshot 记录 PVP 战斗中实际参战的玩家武将快照。
type PvpGeneralSnapshot struct {
	ID    string             `json:"id"`
	Name  string             `json:"name,omitempty"`
	Level int                `json:"level,omitempty"`
	Buffs map[string]float64 `json:"buffs,omitempty"`
}

// PvpRevengeRecord 记录一次玩家被攻击后生成的复仇机会。
type PvpRevengeRecord struct {
	ID               string `json:"id"`
	DefenderPlayerID string `json:"defenderPlayerId"`
	AttackerPlayerID string `json:"attackerPlayerId"`
	MarchID          string `json:"marchId"`
	BattleID         string `json:"battleId,omitempty"`
	Status           string `json:"status"`
	CreatedAt        string `json:"createdAt"`
	ExpiresAt        string `json:"expiresAt"`
	ClosedAt         string `json:"closedAt,omitempty"`
}

// PvpStateResponse 返回玩家 PVP 状态和第一版赛季统计。
type PvpStateResponse struct {
	State          PvpPlayerState     `json:"state"`
	SeasonPoints   int                `json:"seasonPoints"`
	Rating         int                `json:"rating"`
	AttackWins     int                `json:"attackWins"`
	DefenseWins    int                `json:"defenseWins"`
	Losses         int                `json:"losses"`
	RevengeRecords []PvpRevengeRecord `json:"revengeRecords"`
	ServerTime     string             `json:"serverTime"`
}

// PvpRevengeListResponse 返回玩家可用复仇记录。
type PvpRevengeListResponse struct {
	Items      []PvpRevengeRecord `json:"items"`
	ServerTime string             `json:"serverTime"`
}

// PvpSeasonSummary 是 PVP 赛季摘要。
type PvpSeasonSummary struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	StartsAt  string `json:"startsAt"`
	EndsAt    string `json:"endsAt"`
	UpdatedAt string `json:"updatedAt"`
}

// PvpSeasonRecord 是独立赛季表的领域记录。
type PvpSeasonRecord struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Status    string         `json:"status"`
	StartsAt  string         `json:"startsAt"`
	EndsAt    string         `json:"endsAt"`
	SettledAt string         `json:"settledAt,omitempty"`
	Rules     map[string]any `json:"rules,omitempty"`
	Rewards   map[string]any `json:"rewards,omitempty"`
	CreatedAt string         `json:"createdAt"`
	UpdatedAt string         `json:"updatedAt"`
}

// PvpSeasonPlayerRecord 是赛季结算时固化的玩家成绩。
type PvpSeasonPlayerRecord struct {
	SeasonID      string `json:"seasonId"`
	PlayerID      string `json:"playerId"`
	Nickname      string `json:"nickname,omitempty"`
	Faction       string `json:"faction,omitempty"`
	Rank          int    `json:"rank"`
	Points        int    `json:"points"`
	Rating        int    `json:"rating"`
	Wins          int    `json:"wins"`
	Losses        int    `json:"losses"`
	DefenseWins   int    `json:"defenseWins"`
	DefenseLosses int    `json:"defenseLosses"`
	LastBattleAt  string `json:"lastBattleAt,omitempty"`
	RewardMailID  string `json:"rewardMailId,omitempty"`
	RewardSentAt  string `json:"rewardSentAt,omitempty"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
}

// PvpRankingEntry 是 PVP 排行榜条目。
type PvpRankingEntry struct {
	Rank        int    `json:"rank"`
	PlayerID    string `json:"playerId"`
	Nickname    string `json:"nickname"`
	Faction     string `json:"faction"`
	Points      int    `json:"points"`
	Rating      int    `json:"rating"`
	AttackWins  int    `json:"attackWins"`
	DefenseWins int    `json:"defenseWins"`
	Losses      int    `json:"losses"`
	UpdatedAt   string `json:"updatedAt"`
}

// PvpSeasonResponse 返回当前赛季和我的排行状态。
type PvpSeasonResponse struct {
	Season     PvpSeasonSummary `json:"season"`
	Self       *PvpRankingEntry `json:"self,omitempty"`
	ServerTime string           `json:"serverTime"`
}

// PvpRankingResponse 返回 PVP 排行榜。
type PvpRankingResponse struct {
	Season     PvpSeasonSummary  `json:"season"`
	Items      []PvpRankingEntry `json:"items"`
	Self       *PvpRankingEntry  `json:"self,omitempty"`
	ServerTime string            `json:"serverTime"`
}

// AdminPvpOverviewResponse 是 GM 后台 PVP 只读工作台响应。
type AdminPvpOverviewResponse struct {
	PlayerID   string            `json:"playerId,omitempty"`
	Player     *PvpStateResponse `json:"player,omitempty"`
	Season     PvpSeasonSummary  `json:"season"`
	Rankings   []PvpRankingEntry `json:"rankings"`
	Marches    []PvpMarch        `json:"marches"`
	Battles    []PvpBattle       `json:"battles"`
	ServerTime string            `json:"serverTime"`
}

// AdminPvpSeasonListResponse 是 GM 赛季列表响应。
type AdminPvpSeasonListResponse struct {
	Current    PvpSeasonSummary  `json:"current"`
	Seasons    []PvpSeasonRecord `json:"seasons"`
	ServerTime string            `json:"serverTime"`
}

// AdminSettlePvpSeasonResponse 是 GM 结算赛季响应。
type AdminSettlePvpSeasonResponse struct {
	Season     PvpSeasonRecord         `json:"season"`
	Players    []PvpSeasonPlayerRecord `json:"players"`
	RewardMail int                     `json:"rewardMail"`
	ServerTime string                  `json:"serverTime"`
}

// AdminSetPvpProtectionRequest 是 GM 设置 PVP 保护的请求。
type AdminSetPvpProtectionRequest struct {
	ProtectionType string `json:"protectionType"`
	Hours          int    `json:"hours"`
	Reason         string `json:"reason,omitempty"`
}

// AdminSavePvpSeasonRequest 是 GM 创建或编辑 PVP 赛季的请求。
type AdminSavePvpSeasonRequest struct {
	ID       string         `json:"id,omitempty"`
	Name     string         `json:"name"`
	Status   string         `json:"status,omitempty"`
	StartsAt string         `json:"startsAt"`
	EndsAt   string         `json:"endsAt"`
	Rules    map[string]any `json:"rules,omitempty"`
	Rewards  map[string]any `json:"rewards,omitempty"`
}

// PvpAttackRequest 是发起 PVP 攻击或掠夺的请求。
type PvpAttackRequest struct {
	PlayerID       string         `json:"playerId"`
	TargetPlayerID string         `json:"targetPlayerId"`
	Troops         map[string]int `json:"troops"`
	GeneralIDs     []string       `json:"generalIds,omitempty"`
	MarchMode      string         `json:"marchMode,omitempty"`
}

// PvpAttackResponse 是发起 PVP 行军后的响应。
type PvpAttackResponse struct {
	March      PvpMarch   `json:"march"`
	Army       []ArmyUnit `json:"army"`
	Generals   []General  `json:"generals,omitempty"`
	ServerTime string     `json:"serverTime"`
}

// PvpMarchActionResponse 是召回、加速等行军操作响应。
type PvpMarchActionResponse struct {
	March      PvpMarch   `json:"march"`
	Army       []ArmyUnit `json:"army,omitempty"`
	Generals   []General  `json:"generals,omitempty"`
	CityGold   FlexInt    `json:"cityGold,omitempty"`
	Cost       int        `json:"cost,omitempty"`
	ServerTime string     `json:"serverTime"`
}

// PvpMarchListResponse 是玩家行军列表响应。
type PvpMarchListResponse struct {
	Items []PvpMarch `json:"items"`
}

// PvpBattleListResponse 是 PVP 战斗列表响应。
type PvpBattleListResponse struct {
	Items []PvpBattle `json:"items"`
}

// PvpScoutRequest 是侦查玩家请求。
type PvpScoutRequest struct {
	PlayerID       string `json:"playerId"`
	TargetPlayerID string `json:"targetPlayerId"`
}

// PvpScoutResponse 是侦查玩家响应。
type PvpScoutResponse struct {
	Success      bool         `json:"success"`
	BattleReport BattleReport `json:"battleReport"`
	ServerTime   string       `json:"serverTime"`
}
