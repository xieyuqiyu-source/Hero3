// 本文件定义黄巾起义玩法的状态、来袭队列和响应模型。
package game

const (
	YellowTurbanModuleID = "yellow_turban"

	WorldMapTargetYellowTurban = "yellow_turban"

	YellowTurbanMarchStatusMarching  = "marching"
	YellowTurbanMarchStatusResolving = "resolving"
	YellowTurbanMarchStatusResolved  = "resolved"
	YellowTurbanMarchStatusFailed    = "failed"

	ReportSourceYellowTurban = "yellow_turban"
	BattleTypeYellowTurban   = "yellow_turban"
)

// FoodPressureState 描述玩家当前口粮压力。
type FoodPressureState struct {
	CurrentFood       int     `json:"currentFood"`
	FoodCapacity      int     `json:"foodCapacity"`
	Pressure          float64 `json:"pressure"`
	OverCapacity      bool    `json:"overCapacity"`
	ThousandTentLevel int     `json:"thousandTentLevel"`
	RiskLevelID       int     `json:"riskLevelId,omitempty"`
	RiskLevelName     string  `json:"riskLevelName,omitempty"`
	RiskColor         string  `json:"riskColor,omitempty"`
}

// YellowTurbanCity 是地图上的黄巾假玩家城池。
type YellowTurbanCity struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	RegionID  string `json:"regionId"`
	Faction   string `json:"faction"`
	WorldID   string `json:"worldId"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"createdAt,omitempty"`
}

// YellowTurbanMarch 记录一条黄巾来袭队列。
type YellowTurbanMarch struct {
	ID               string         `json:"id"`
	TargetPlayerID   string         `json:"targetPlayerId"`
	SourceCityID     string         `json:"sourceCityId"`
	SourceName       string         `json:"sourceName"`
	SourceFaction    string         `json:"sourceFaction"`
	SourceRegionID   string         `json:"sourceRegionId"`
	RiskLevelID      int            `json:"riskLevelId"`
	RiskLevelName    string         `json:"riskLevelName"`
	PlayerFood       int            `json:"playerFood"`
	FoodCapacity     int            `json:"foodCapacity"`
	Pressure         float64        `json:"pressure"`
	Troops           map[string]int `json:"troops"`
	Status           string         `json:"status"`
	DurationSeconds  int            `json:"durationSeconds"`
	StartedAt        string         `json:"startedAt"`
	ArrivesAt        string         `json:"arrivesAt"`
	ResolvedAt       string         `json:"resolvedAt,omitempty"`
	DefenderReportID string         `json:"defenderReportId,omitempty"`
	Error            string         `json:"error,omitempty"`
	CreatedAt        string         `json:"createdAt"`
	UpdatedAt        string         `json:"updatedAt"`
}

// BuildYellowTurbanBattleEvent 生成与黄巾结算事务一同持久化的标准战斗事件。
func BuildYellowTurbanBattleEvent(march YellowTurbanMarch, report BattleReport) BattleEvent {
	event := BuildBattleEventFromReport(report)
	event.ID = report.EventID
	event.SourceType = ReportSourceYellowTurban
	event.SourceID = march.ID
	event.Scene = report.ViewType
	event.BattleType = BattleTypeYellowTurban
	event.Result = report.Result
	event.RelatedMarchID = march.ID
	event.Summary = map[string]interface{}{
		"yellowTurbanMarchId": march.ID,
		"sourceCityId":        march.SourceCityID,
		"riskLevelId":         march.RiskLevelID,
	}
	event.OccurredAt = report.CreatedAt
	event.CreatedAt = report.CreatedAt
	return event
}

// YellowTurbanStatusResponse 返回玩家黄巾起义据点状态。
type YellowTurbanStatusResponse struct {
	Enabled              bool                `json:"enabled"`
	FoodPressure         FoodPressureState   `json:"foodPressure"`
	CheckIntervalMinutes int                 `json:"checkIntervalMinutes"`
	NextCheckAt          string              `json:"nextCheckAt,omitempty"`
	IncomingCount        int                 `json:"incomingCount"`
	MaxIncoming          int                 `json:"maxIncoming"`
	Incoming             []YellowTurbanMarch `json:"incoming"`
	Cities               []YellowTurbanCity  `json:"cities,omitempty"`
	RecentReports        []BattleReport      `json:"recentReports,omitempty"`
	ServerTime           string              `json:"serverTime"`
}

// YellowTurbanCheckResult 描述一次黄巾检测是否派兵。
type YellowTurbanCheckResult struct {
	Checked       bool               `json:"checked"`
	Spawned       bool               `json:"spawned"`
	Reason        string             `json:"reason,omitempty"`
	FoodPressure  FoodPressureState  `json:"foodPressure"`
	March         *YellowTurbanMarch `json:"march,omitempty"`
	IncomingCount int                `json:"incomingCount"`
	MaxIncoming   int                `json:"maxIncoming"`
	ServerTime    string             `json:"serverTime"`
}
