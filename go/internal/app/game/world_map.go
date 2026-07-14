// 本文件定义世界地图玩家城池坐标、目标和视图响应模型。
package game

const (
	WorldMapModuleID = "world_map"

	defaultWorldID       = "world_1"
	defaultWorldWidth    = 100
	defaultWorldHeight   = 100
	defaultWorldRadius   = 10
	maxWorldViewRadius   = 100
	maxWorldMarchSeconds = 3 * 3600

	WorldMapTargetPlayerCity = "player_city"

	WorldRelationSelf  = "self"
	WorldRelationAlly  = "ally"
	WorldRelationOther = "other"

	WorldTargetStatusSelf        = "self"
	WorldTargetStatusNormal      = "normal"
	WorldTargetStatusProtected   = "protected"
	WorldTargetStatusTruce       = "truce"
	WorldTargetStatusAttackable  = "attackable"
	WorldTargetStatusUnavailable = "unavailable"
)

// WorldCoordinate 是世界地图上的一个格子坐标。
type WorldCoordinate struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// WorldPosition 是玩家城池在世界地图中的权威坐标。
type WorldPosition struct {
	PlayerID   string `json:"playerId"`
	WorldID    string `json:"worldId"`
	X          int    `json:"x"`
	Y          int    `json:"y"`
	AssignedBy string `json:"assignedBy"`
	CreatedAt  string `json:"createdAt,omitempty"`
	UpdatedAt  string `json:"updatedAt,omitempty"`
}

// WorldMapPlayerCity 是地图范围查询直接返回的玩家城池轻量投影。
type WorldMapPlayerCity struct {
	Position      WorldPosition
	AccountID     string
	Name          string
	Faction       string
	BuildingLevel int
}

// WorldMapTarget 是世界地图视图中可点击的目标。
type WorldMapTarget struct {
	TargetType      string `json:"targetType"`
	TargetID        string `json:"targetId"`
	PlayerID        string `json:"playerId,omitempty"`
	Name            string `json:"name"`
	Faction         string `json:"faction"`
	Relation        string `json:"relation"`
	SameAccount     bool   `json:"sameAccount"`
	Level           int    `json:"level"`
	Status          string `json:"status"`
	X               int    `json:"x"`
	Y               int    `json:"y"`
	Distance        int    `json:"distance"`
	Direction       string `json:"direction"`
	CanScout        bool   `json:"canScout"`
	CanAttack       bool   `json:"canAttack"`
	CanPlunder      bool   `json:"canPlunder"`
	CanReinforce    bool   `json:"canReinforce"`
	Reason          string `json:"reason,omitempty"`
	ScoutReason     string `json:"scoutReason,omitempty"`
	AttackReason    string `json:"attackReason,omitempty"`
	PlunderReason   string `json:"plunderReason,omitempty"`
	ReinforceReason string `json:"reinforceReason,omitempty"`
}

// WorldMapViewResponse 是世界地图视野响应。
type WorldMapViewResponse struct {
	WorldID    string           `json:"worldId"`
	Width      int              `json:"width"`
	Height     int              `json:"height"`
	Self       WorldPosition    `json:"self"`
	CenterX    int              `json:"centerX"`
	CenterY    int              `json:"centerY"`
	Radius     int              `json:"radius"`
	Targets    []WorldMapTarget `json:"targets"`
	ServerTime string           `json:"serverTime"`
}

// WorldMapMigrationResult 汇总一次世界坐标补齐迁移结果。
type WorldMapMigrationResult struct {
	Total           int      `json:"total"`
	Created         int      `json:"created"`
	Skipped         int      `json:"skipped"`
	Conflicts       int      `json:"conflicts"`
	ConflictDetails []string `json:"conflictDetails,omitempty"`
	Failed          int      `json:"failed"`
	Failures        []string `json:"failures,omitempty"`
}

// WorldMapOccupancyStats 汇总世界地图坐标占用情况。
type WorldMapOccupancyStats struct {
	WorldID        string  `json:"worldId"`
	Width          int     `json:"width"`
	Height         int     `json:"height"`
	TotalCells     int     `json:"totalCells"`
	OccupiedCells  int     `json:"occupiedCells"`
	AvailableCells int     `json:"availableCells"`
	OccupancyRate  float64 `json:"occupancyRate"`
}

// WorldMapCoordinateCheck 描述 GM 查询某个世界坐标格的占用状态。
type WorldMapCoordinateCheck struct {
	WorldID  string `json:"worldId"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Occupied bool   `json:"occupied"`
	PlayerID string `json:"playerId,omitempty"`
}
