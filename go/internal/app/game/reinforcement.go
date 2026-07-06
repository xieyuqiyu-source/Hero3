// 本文件定义增援系统的领域模型、状态和请求响应结构。
package game

const (
	ReinforcementModuleID = "reinforcement"

	ReinforcementTargetPlayerCity = "player_city"

	GarrisonSourceReinforcement = "reinforcement"
	GarrisonSourceObtained      = "obtained"
	GarrisonSourceCaptured      = "captured"
	GarrisonSourceMercenary     = "mercenary"
	GarrisonSourceEventReward   = "event_reward"
	GarrisonSourceSystem        = "system"

	ReinforcementStatusMarching  = "marching"
	ReinforcementStatusStationed = "stationed"
	ReinforcementStatusFighting  = "fighting"
	ReinforcementStatusReturning = "returning"
	ReinforcementStatusCompleted = "completed"
	ReinforcementStatusCancelled = "cancelled"
	ReinforcementStatusFailed    = "failed"

	defaultReinforcementMarchSeconds = 10800
	minReinforcementMarchSeconds     = 1
	maxReinforcementMarchSeconds     = defaultReinforcementMarchSeconds
	reinforcementSecondsPerGrid      = 300
	defaultReinforcementMaxSources   = 5
)

// GarrisonRules 定义一支驻防队伍允许参与的操作。
type GarrisonRules struct {
	CanRecall  bool `json:"canRecall"`
	CanExpel   bool `json:"canExpel"`
	CanReturn  bool `json:"canReturn"`
	CanFight   bool `json:"canFight"`
	CanConvert bool `json:"canConvert"`
	CanRelease bool `json:"canRelease"`
}

// ReinforcementGeneralSnapshot 保存增援携带武将的快照。
type ReinforcementGeneralSnapshot struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name,omitempty"`
	Level      int                    `json:"level,omitempty"`
	Stats      map[string]int         `json:"stats,omitempty"`
	Attributes map[string]float64     `json:"attributes,omitempty"`
	Buffs      map[string]float64     `json:"buffs,omitempty"`
	Traits     []GeneralTraitInstance `json:"traits,omitempty"`
	Assignment string                 `json:"assignment,omitempty"`
}

// Reinforcement 记录一批增援的完整生命周期。
type Reinforcement struct {
	ID                 string                         `json:"reinforcementId"`
	FromPlayerID       string                         `json:"fromPlayerId"`
	FromPlayerName     string                         `json:"fromPlayerName,omitempty"`
	FromPlayerFaction  string                         `json:"fromPlayerFaction,omitempty"`
	ToPlayerID         string                         `json:"toPlayerId"`
	ToPlayerName       string                         `json:"toPlayerName,omitempty"`
	ToPlayerFaction    string                         `json:"toPlayerFaction,omitempty"`
	OwnerPlayerID      string                         `json:"ownerPlayerId,omitempty"`
	HostPlayerID       string                         `json:"hostPlayerId,omitempty"`
	SourceType         string                         `json:"sourceType,omitempty"`
	SourceID           string                         `json:"sourceId,omitempty"`
	TargetType         string                         `json:"targetType"`
	TargetID           string                         `json:"targetId"`
	Status             string                         `json:"status"`
	Troops             map[string]int                 `json:"troops"`
	RemainingTroops    map[string]int                 `json:"remainingTroops"`
	Generals           []ReinforcementGeneralSnapshot `json:"generals,omitempty"`
	Losses             map[string]int                 `json:"losses,omitempty"`
	BuffSnapshot       []ModifierBreakdownItem        `json:"buffSnapshot,omitempty"`
	Rules              GarrisonRules                  `json:"rules"`
	SpeedMultiplier    float64                        `json:"speedMultiplier"`
	MarchSeconds       int                            `json:"marchSeconds"`
	ReturnSeconds      int                            `json:"returnSeconds"`
	SentAt             string                         `json:"sentAt"`
	ExpectedArriveAt   string                         `json:"arriveAt,omitempty"`
	ArrivedAt          string                         `json:"arrivedAt,omitempty"`
	RecalledAt         string                         `json:"recalledAt,omitempty"`
	ExpelledAt         string                         `json:"expelledAt,omitempty"`
	ReturnStartedAt    string                         `json:"returnStartedAt,omitempty"`
	ExpectedReturnedAt string                         `json:"expectedReturnedAt,omitempty"`
	ReturnedAt         string                         `json:"returnedAt,omitempty"`
	LastBattleReportID string                         `json:"lastBattleReportId,omitempty"`
	LastBattleAt       string                         `json:"lastBattleAt,omitempty"`
	IsAnnihilated      bool                           `json:"isAnnihilated"`
	RewardState        map[string]any                 `json:"rewardState,omitempty"`
	MailState          map[string]any                 `json:"mailState,omitempty"`
	Metadata           map[string]any                 `json:"metadata,omitempty"`
	CreatedAt          string                         `json:"createdAt"`
	UpdatedAt          string                         `json:"updatedAt"`
}

// SendReinforcementRequest 是发起增援请求。
type SendReinforcementRequest struct {
	FromPlayerID    string         `json:"playerId"`
	TargetPlayerID  string         `json:"targetPlayerId"`
	Troops          map[string]int `json:"troops"`
	GeneralIDs      []string       `json:"generalIds,omitempty"`
	SpeedMultiplier float64        `json:"speedMultiplier,omitempty"`
}

// CreateGarrisonDetachmentRequest 是创建非增援驻防队伍的内部标准请求。
type CreateGarrisonDetachmentRequest struct {
	OwnerPlayerID string         `json:"ownerPlayerId"`
	HostPlayerID  string         `json:"hostPlayerId"`
	SourceType    string         `json:"sourceType"`
	SourceID      string         `json:"sourceId,omitempty"`
	SourceFaction string         `json:"sourceFaction,omitempty"`
	Troops        map[string]int `json:"troops"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

// ReinforcementResponse 是单个增援操作响应。
type ReinforcementResponse struct {
	Reinforcement Reinforcement        `json:"reinforcement"`
	Patch         GarrisonActionResult `json:"patch,omitempty"`
}

// ReinforcementActionResponse 是会改变城金或局部状态的增援操作响应。
type ReinforcementActionResponse struct {
	Reinforcement Reinforcement        `json:"reinforcement"`
	Patch         GarrisonActionResult `json:"patch,omitempty"`
	CityGold      FlexInt              `json:"cityGold,omitempty"`
	Cost          int                  `json:"cost,omitempty"`
	ServerTime    string               `json:"serverTime,omitempty"`
}

// ReinforcementListResponse 是增援列表响应。
type ReinforcementListResponse struct {
	Items []Reinforcement `json:"items"`
}

// DefenseReinforcementUnit 是防守战斗接入时的一批援军。
type DefenseReinforcementUnit struct {
	ReinforcementID string                         `json:"reinforcementId"`
	FromPlayerID    string                         `json:"fromPlayerId"`
	FromPlayerName  string                         `json:"fromPlayerName,omitempty"`
	Faction         string                         `json:"faction"`
	Troops          map[string]int                 `json:"troops"`
	Generals        []ReinforcementGeneralSnapshot `json:"generals,omitempty"`
	Buffs           []ModifierBreakdownItem        `json:"buffs,omitempty"`
	SourceTags      map[string]string              `json:"sourceTags,omitempty"`
}

// ReinforcementLoss 记录战后某一批援军的单兵种损耗。
type ReinforcementLoss struct {
	ReinforcementID string `json:"reinforcementId"`
	FromPlayerID    string `json:"fromPlayerId"`
	UnitType        string `json:"unitType"`
	BeforeAmount    int    `json:"beforeAmount"`
	LostAmount      int    `json:"lostAmount"`
	RemainingAmount int    `json:"remainingAmount"`
}
