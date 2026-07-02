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
	ID                string `json:"id"`
	Nickname          string `json:"nickname"`
	Faction           string `json:"faction"`
	MailCode          string `json:"mailCode,omitempty"`
	TotalArmy         int    `json:"totalArmy"`
	BuildingLevel     int    `json:"buildingLevel"`
	UpdatedAt         string `json:"updatedAt"`
	DeleteRequestedAt string `json:"deleteRequestedAt,omitempty"`
	DeleteScheduledAt string `json:"deleteScheduledAt,omitempty"`
}

type PlayerDeletionResult struct {
	Status            string `json:"status"`
	PlayerID          string `json:"playerId"`
	DeleteRequestedAt string `json:"deleteRequestedAt,omitempty"`
	DeleteScheduledAt string `json:"deleteScheduledAt,omitempty"`
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
	MailCode string `json:"mailCode,omitempty"`
}

type ResourceState struct {
	Items    map[string]int `json:"items"`
	Capacity map[string]int `json:"capacity"`
}

type ResourceProduction map[string]int

type ItemStack struct {
	SlotID     string `json:"slotId,omitempty"`
	ItemID     string `json:"itemId"`
	Amount     int    `json:"amount"`
	ObtainedAt string `json:"obtainedAt,omitempty"`
	UpdatedAt  string `json:"updatedAt,omitempty"`
}

type Building struct {
	ID            string  `json:"id"`
	Type          string  `json:"type"`
	Level         int     `json:"level"`
	Status        string  `json:"status,omitempty"`
	UpgradeEndsAt *string `json:"upgradeEndsAt"`
	StatusEndsAt  *string `json:"statusEndsAt,omitempty"`
}

type ResourceSlot struct {
	ID           string `json:"id"`
	ResourceType string `json:"resourceType"`
	BuildingID   string `json:"buildingId,omitempty"`
	UnlockedBy   string `json:"unlockedBy,omitempty"`
	UnlockedAt   string `json:"unlockedAt,omitempty"`
}

type ArmyUnit struct {
	UnitType string `json:"unitType"`
	Amount   int    `json:"amount"`
}

type General struct {
	ID                  string                                     `json:"id"`
	Name                string                                     `json:"name"`
	Level               int                                        `json:"level"`
	Exp                 int                                        `json:"exp"`
	CurrentLevelExp     int                                        `json:"currentLevelExp,omitempty"`     // 当前等级起始累计经验，用于前端显示本级进度
	NextLevelExp        int                                        `json:"nextLevelExp,omitempty"`        // 下一等级所需累计经验；满级为 0
	AvailableStatPoints int                                        `json:"availableStatPoints,omitempty"` // 可分配四维点数
	Stats               map[string]int                             `json:"stats,omitempty"`               // 四维加点，单项上限 100
	Attributes          map[string]float64                         `json:"attributes,omitempty"`          // 展示用多维属性，来源于等级属性 + 四维加点 + 将领固定属性
	AttributeBreakdown  map[string][]GeneralAttributeBreakdownItem `json:"attributeBreakdown,omitempty"`  // 属性来源拆分，供前端 tooltip 展示
	Buffs               map[string]float64                         `json:"buffs"`                         // 兼容旧字段，Modifier 管线仍从这里读取
	Traits              []GeneralTraitInstance                     `json:"traits,omitempty"`              // 当前激活的特性（来自配置）
}

type GeneralAssignment struct {
	ID         string `json:"id"`
	GeneralID  string `json:"generalId"`
	Slot       string `json:"slot"`
	ModuleID   string `json:"moduleId,omitempty"`
	Status     string `json:"status,omitempty"`
	AssignedAt string `json:"assignedAt,omitempty"`
	EndsAt     string `json:"endsAt,omitempty"`
}

type GeneralAttributeBreakdownItem struct {
	Source string  `json:"source"`
	Value  float64 `json:"value"`
}

// GeneralTraitInstance 玩家身上激活的特性实例（trait id + 当前参数）
// 在玩家创建/读取时根据 GeneralsConfig 填充
type GeneralTraitInstance struct {
	TraitID string             `json:"traitId"` // 对应 traits 注册中心
	Name    string             `json:"name"`    // 显示名（冗余便于前端）
	Params  map[string]float64 `json:"params"`  // GM 配置的当前参数
}

// TraitOutcomeReport 战报中单条特性触发结果
type TraitOutcomeReport struct {
	TraitID string                 `json:"traitId"`
	Name    string                 `json:"name,omitempty"`
	Detail  map[string]interface{} `json:"detail,omitempty"`
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
	ID                       string                        `json:"id"`
	EventID                  string                        `json:"eventId,omitempty"`
	PlayerID                 string                        `json:"playerId"`
	OwnerPlayerID            string                        `json:"ownerPlayerId,omitempty"`
	ViewType                 string                        `json:"viewType,omitempty"`
	SourceType               string                        `json:"sourceType,omitempty"`
	BattleType               string                        `json:"battleType,omitempty"`
	Title                    string                        `json:"title,omitempty"`
	Summary                  string                        `json:"summary,omitempty"`
	Detail                   *BattleReportDetail           `json:"detail,omitempty"`
	Share                    *BattleReportShare            `json:"share,omitempty"`
	PlayerFaction            string                        `json:"playerFaction"`
	PlayerName               string                        `json:"playerName,omitempty"`
	TargetID                 string                        `json:"targetId"`
	TargetName               string                        `json:"targetName"`
	Type                     string                        `json:"type"` // "attack", "plunder", "scout", "reinforce"
	Result                   string                        `json:"result"`
	PlayerPower              int                           `json:"playerPower"`
	EnemyPower               int                           `json:"enemyPower"`
	DispatchedUnits          map[string]int                `json:"dispatchedUnits"`
	LostUnits                map[string]int                `json:"lostUnits"`
	DefenderFaction          string                        `json:"defenderFaction"`
	DefenderUnits            map[string]int                `json:"defenderUnits"`
	DefenderLostUnits        map[string]int                `json:"defenderLostUnits"`
	DefenderRevealed         bool                          `json:"defenderRevealed"`
	DefenderResources        map[string]int                `json:"defenderResources"`
	EnemyLossRevealThreshold float64                       `json:"enemyLossRevealThreshold,omitempty"`
	EnemyLossRatio           float64                       `json:"enemyLossRatio,omitempty"`
	Rewards                  map[string]int                `json:"rewards"`
	GrantedRewards           []Reward                      `json:"grantedRewards,omitempty"`
	Drops                    []BattleReportDrop            `json:"drops,omitempty"`                  // 本场战斗掉落物品
	Overflow                 map[string]int                `json:"overflow,omitempty"`               // 各资源溢出量
	OverflowCityGold         int                           `json:"overflowCityGold"`                 // 溢出转换获得的城金
	GeneralExpGained         int                           `json:"generalExpGained,omitempty"`       // 本次战斗获得将领经验
	GeneralLevelBefore       int                           `json:"generalLevelBefore,omitempty"`     // 战斗前将领等级
	GeneralLevelAfter        int                           `json:"generalLevelAfter,omitempty"`      // 战斗后将领等级
	CapturedUnits            map[string]int                `json:"capturedUnits,omitempty"`          // 美人计俘虏到军队
	CapturedToGarrison       map[string]int                `json:"capturedToGarrison,omitempty"`     // 美人计俘虏到驻防
	RevivedUnits             map[string]int                `json:"revivedUnits,omitempty"`           // 仁德复活
	TraitTriggered           []string                      `json:"traitTriggered,omitempty"`         // 触发了哪些特性（前端展示）
	TraitOutcomes            map[string]TraitOutcomeReport `json:"traitOutcomes,omitempty"`          // 每个触发特性的具体结果
	PvpPointsDelta           map[string]int                `json:"pvpPointsDelta,omitempty"`         // PVP 积分变化
	PvpAttackerGenerals      []PvpGeneralSnapshot          `json:"pvpAttackerGenerals,omitempty"`    // PVP 攻击方参战武将
	PvpDefenderGenerals      []PvpGeneralSnapshot          `json:"pvpDefenderGenerals,omitempty"`    // PVP 防守方参战武将
	PvpReinforcements        []DefenseReinforcementUnit    `json:"pvpReinforcements,omitempty"`      // PVP 参战驻防/援军摘要
	PvpReinforcementLosses   map[string]map[string]int     `json:"pvpReinforcementLosses,omitempty"` // PVP 援军损耗
	PvpWall                  *PvpWallSnapshot              `json:"pvpWall,omitempty"`                // PVP 城墙系数和硬度预留快照
	Read                     bool                          `json:"read"`
	DeletedByPlayer          bool                          `json:"deletedByPlayer,omitempty"`
	CreatedAt                string                        `json:"createdAt"`
}

// PvpWallSnapshot 保存 PVP 战斗发生时的防守方城墙配置快照。
type PvpWallSnapshot struct {
	Faction               string  `json:"faction"`
	Level                 int     `json:"level"`
	Base                  float64 `json:"base"`
	Multiplier            float64 `json:"multiplier"`
	FactionDefenseBonus   float64 `json:"factionDefenseBonus"`
	TotalDefenseBonus     float64 `json:"totalDefenseBonus"`
	Hardness              float64 `json:"hardness,omitempty"`
	MinDamagedLevelFrom20 int     `json:"minDamagedLevelFrom20"`
	MaxDamagedLevelFrom20 int     `json:"maxDamagedLevelFrom20"`
}

type BattleReportPage struct {
	Reports  []BattleReport `json:"reports"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
	Total    int            `json:"total"`
}

// BattleReportCreateInput 是玩法模块接入统一战报服务的标准输入。
type BattleReportCreateInput struct {
	EventID                string                 `json:"eventId,omitempty"`
	SourceType             string                 `json:"sourceType"`
	SourceID               string                 `json:"sourceId,omitempty"`
	BattleType             string                 `json:"battleType"`
	Result                 string                 `json:"result"`
	RelatedMarchID         string                 `json:"relatedMarchId,omitempty"`
	RelatedReinforcementID string                 `json:"relatedReinforcementId,omitempty"`
	OccurredAt             string                 `json:"occurredAt,omitempty"`
	Reports                []BattleReport         `json:"reports"`
	Extra                  map[string]interface{} `json:"extra,omitempty"`
}

// BattleReportCreateResult 是统一战报服务创建事件和多视角战报后的结果。
type BattleReportCreateResult struct {
	Event   BattleEvent    `json:"event"`
	Reports []BattleReport `json:"reports"`
}

// BattleEvent 记录一次真实发生的战斗或侦查事件，供多份玩家视角战报归档。
type BattleEvent struct {
	ID                     string                 `json:"id"`
	SourceType             string                 `json:"sourceType"`
	SourceID               string                 `json:"sourceId,omitempty"`
	Scene                  string                 `json:"scene,omitempty"`
	BattleType             string                 `json:"battleType"`
	Result                 string                 `json:"result"`
	AttackerPlayerID       string                 `json:"attackerPlayerId,omitempty"`
	DefenderPlayerID       string                 `json:"defenderPlayerId,omitempty"`
	AttackerName           string                 `json:"attackerName,omitempty"`
	DefenderName           string                 `json:"defenderName,omitempty"`
	AttackerFaction        string                 `json:"attackerFaction,omitempty"`
	DefenderFaction        string                 `json:"defenderFaction,omitempty"`
	RelatedMarchID         string                 `json:"relatedMarchId,omitempty"`
	RelatedReinforcementID string                 `json:"relatedReinforcementId,omitempty"`
	Summary                map[string]interface{} `json:"summary,omitempty"`
	Snapshot               map[string]interface{} `json:"snapshot,omitempty"`
	ResultData             map[string]interface{} `json:"resultData,omitempty"`
	OccurredAt             string                 `json:"occurredAt"`
	CreatedAt              string                 `json:"createdAt"`
}

// BattleReportDetail 是玩家端和分享页优先消费的标准战报详情结构。
type BattleReportDetail struct {
	ID            string                 `json:"id"`
	EventID       string                 `json:"eventId,omitempty"`
	OwnerPlayerID string                 `json:"ownerPlayerId,omitempty"`
	ViewType      string                 `json:"viewType"`
	ViewLabel     string                 `json:"viewLabel"`
	SourceType    string                 `json:"sourceType"`
	SourceLabel   string                 `json:"sourceLabel"`
	BattleType    string                 `json:"battleType"`
	Result        string                 `json:"result"`
	Title         string                 `json:"title"`
	Summary       string                 `json:"summary,omitempty"`
	OccurredAt    string                 `json:"occurredAt"`
	PrimarySide   BattleReportSide       `json:"primarySide"`
	SecondarySide *BattleReportSide      `json:"secondarySide,omitempty"`
	Rewards       BattleReportRewards    `json:"rewards"`
	Traits        []BattleReportTrait    `json:"traits,omitempty"`
	Visibility    BattleReportVisibility `json:"visibility"`
	Extra         map[string]interface{} `json:"extra,omitempty"`
	Read          bool                   `json:"read"`
	Share         *BattleReportShare     `json:"share,omitempty"`
}

// BattleReportSweepDefender 保存扫荡聚合战报中的单个 NPC 防守方快照。
type BattleReportSweepDefender struct {
	TargetID         string             `json:"targetId"`
	TargetName       string             `json:"targetName"`
	Faction          string             `json:"faction,omitempty"`
	FactionLabel     string             `json:"factionLabel,omitempty"`
	Power            int                `json:"power"`
	Result           string             `json:"result,omitempty"`
	DefenderRevealed bool               `json:"defenderRevealed"`
	Units            []BattleReportUnit `json:"units"`
	Resources        map[string]int     `json:"resources,omitempty"`
}

// BattleReportSide 保存战斗某一方在战斗发生时的快照。
type BattleReportSide struct {
	Role         string                `json:"role"`
	PlayerID     string                `json:"playerId,omitempty"`
	PlayerName   string                `json:"playerName,omitempty"`
	CityID       string                `json:"cityId,omitempty"`
	CityName     string                `json:"cityName,omitempty"`
	Faction      string                `json:"faction,omitempty"`
	FactionLabel string                `json:"factionLabel,omitempty"`
	TargetType   string                `json:"targetType,omitempty"`
	TargetID     string                `json:"targetId,omitempty"`
	TargetName   string                `json:"targetName,omitempty"`
	Level        int                   `json:"level,omitempty"`
	Power        int                   `json:"power"`
	Generals     []BattleReportGeneral `json:"generals,omitempty"`
	Units        []BattleReportUnit    `json:"units"`
	Resources    map[string]int        `json:"resources,omitempty"`
}

// BattleReportUnit 保存单个兵种的出动、损耗和剩余快照。
type BattleReportUnit struct {
	UnitType     string `json:"unitType"`
	UnitName     string `json:"unitName,omitempty"`
	Faction      string `json:"faction,omitempty"`
	AmountBefore int    `json:"amountBefore"`
	Dispatched   int    `json:"dispatched"`
	Lost         int    `json:"lost"`
	Survived     int    `json:"survived"`
}

// BattleReportGeneral 保存参战武将快照，不读取当前玩家武将状态。
type BattleReportGeneral struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name,omitempty"`
	Level      int                    `json:"level,omitempty"`
	Role       string                 `json:"role,omitempty"`
	Power      int                    `json:"power,omitempty"`
	Attributes map[string]float64     `json:"attributes,omitempty"`
	Traits     []GeneralTraitInstance `json:"traits,omitempty"`
}

// BattleReportTrait 保存战斗中特性触发的标准展示数据。
type BattleReportTrait struct {
	TraitID     string                 `json:"traitId"`
	TraitName   string                 `json:"traitName,omitempty"`
	OwnerSide   string                 `json:"ownerSide,omitempty"`
	OwnerRole   string                 `json:"ownerRole,omitempty"`
	GeneralID   string                 `json:"generalId,omitempty"`
	GeneralName string                 `json:"generalName,omitempty"`
	Summary     string                 `json:"summary,omitempty"`
	Detail      map[string]interface{} `json:"detail,omitempty"`
}

// BattleReportRewards 保存奖励快照，实际发放仍由奖励系统完成。
type BattleReportRewards struct {
	Resources          map[string]int     `json:"resources,omitempty"`
	Drops              []BattleReportDrop `json:"drops,omitempty"`
	CityGold           int                `json:"cityGold,omitempty"`
	GeneralExp         int                `json:"generalExp,omitempty"`
	GeneralLevelBefore int                `json:"generalLevelBefore,omitempty"`
	GeneralLevelAfter  int                `json:"generalLevelAfter,omitempty"`
	Overflow           map[string]int     `json:"overflow,omitempty"`
	Granted            []Reward           `json:"granted,omitempty"`
}

// BattleReportDrop 保存道具掉落快照。
type BattleReportDrop struct {
	Type    string `json:"type"`
	ItemID  string `json:"itemId,omitempty"`
	Name    string `json:"name,omitempty"`
	Amount  int    `json:"amount"`
	Quality string `json:"quality,omitempty"`
}

// BattleReportVisibility 控制当前视角能看到哪些敌方信息。
type BattleReportVisibility struct {
	ShowEnemyRemainingUnits bool    `json:"showEnemyRemainingUnits"`
	ShowEnemyResources      bool    `json:"showEnemyResources"`
	ShowEnemyGenerals       bool    `json:"showEnemyGenerals"`
	ShowEnemyCityDefense    bool    `json:"showEnemyCityDefense"`
	Reason                  string  `json:"reason,omitempty"`
	Threshold               float64 `json:"threshold,omitempty"`
	ActualLossRatio         float64 `json:"actualLossRatio,omitempty"`
}

// BattleReportShare 保存分享 token 信息，避免公开内部战报 ID。
type BattleReportShare struct {
	Token      string `json:"token,omitempty"`
	URL        string `json:"url,omitempty"`
	Visibility string `json:"visibility,omitempty"`
	ExpiresAt  string `json:"expiresAt,omitempty"`
}

// BattleReportQuery 描述玩家战报分页查询条件。
type BattleReportQuery struct {
	PlayerID       string
	ViewType       string
	SourceType     string
	BattleType     string
	Result         string
	Page           int
	PageSize       int
	IncludeDeleted bool
	TimeFrom       time.Time
	TimeTo         time.Time
}

// BattleReportShareLink 是持久化的分享链接记录。
type BattleReportShareLink struct {
	ID         string `json:"id"`
	ReportID   string `json:"reportId"`
	Token      string `json:"token"`
	Visibility string `json:"visibility"`
	ExpiresAt  string `json:"expiresAt,omitempty"`
	CreatedAt  string `json:"createdAt"`
}

// BattleReportParticipant 保存一次战斗事件中的某个玩家或守军参与方快照。
type BattleReportParticipant struct {
	ID             string                 `json:"id"`
	EventID        string                 `json:"eventId"`
	ReportID       string                 `json:"reportId,omitempty"`
	PlayerID       string                 `json:"playerId,omitempty"`
	Role           string                 `json:"role"`
	Faction        string                 `json:"faction,omitempty"`
	Nickname       string                 `json:"nickname,omitempty"`
	CityName       string                 `json:"cityName,omitempty"`
	TroopsBefore   map[string]int         `json:"troopsBefore,omitempty"`
	TroopsLost     map[string]int         `json:"troopsLost,omitempty"`
	TroopsSurvived map[string]int         `json:"troopsSurvived,omitempty"`
	Generals       []BattleReportGeneral  `json:"generals,omitempty"`
	Rewards        BattleReportRewards    `json:"rewards,omitempty"`
	PointsDelta    map[string]int         `json:"pointsDelta,omitempty"`
	Extra          map[string]interface{} `json:"extra,omitempty"`
	CreatedAt      string                 `json:"createdAt"`
}

// BattleEventQuery 描述 GM 战斗事件查询条件。
type BattleEventQuery struct {
	PlayerID               string
	EventID                string
	SourceType             string
	SourceID               string
	BattleType             string
	Result                 string
	RelatedMarchID         string
	RelatedReinforcementID string
	Page                   int
	PageSize               int
	TimeFrom               time.Time
	TimeTo                 time.Time
}

// BattleEventPage 是 GM 战斗事件分页结果。
type BattleEventPage struct {
	Items    []BattleEvent `json:"items"`
	Page     int           `json:"page"`
	PageSize int           `json:"pageSize"`
	Total    int           `json:"total"`
}

type MailAttachment struct {
	Type     string                 `json:"type"`
	ItemID   string                 `json:"itemId"`
	Amount   int                    `json:"amount"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type Mail struct {
	ID              string           `json:"id"`
	PlayerID        string           `json:"playerId"`
	MailType        string           `json:"mailType"`
	SenderType      string           `json:"senderType"`
	SenderID        string           `json:"senderId,omitempty"`
	SenderName      string           `json:"senderName"`
	Title           string           `json:"title"`
	Content         string           `json:"content"`
	Attachments     []MailAttachment `json:"attachments,omitempty"`
	SourceType      string           `json:"sourceType,omitempty"`
	SourceID        string           `json:"sourceId,omitempty"`
	IsRead          bool             `json:"isRead"`
	IsClaimed       bool             `json:"isClaimed"`
	DeletedByPlayer bool             `json:"deletedByPlayer,omitempty"`
	ExpiresAt       string           `json:"expiresAt,omitempty"`
	CreatedAt       string           `json:"createdAt"`
	ReadAt          string           `json:"readAt,omitempty"`
	ClaimedAt       string           `json:"claimedAt,omitempty"`
}

type MailPage struct {
	Mails    []Mail `json:"mails"`
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
	Total    int    `json:"total"`
	Unread   int    `json:"unread"`
}

type MailClaimResult struct {
	Mail         Mail                 `json:"mail"`
	Resources    ResourceState        `json:"resources"`
	Inventory    map[string]ItemStack `json:"inventory,omitempty"`
	CityGold     int                  `json:"cityGold"`
	AccountGold  int                  `json:"accountGold,omitempty"`
	GrantedItems map[string]int       `json:"grantedItems"`
}

type GameState struct {
	Player              Player                  `json:"player"`
	Resources           ResourceState           `json:"resources"`
	Inventory           map[string]ItemStack    `json:"inventory,omitempty"`
	InventorySlots      []ItemStack             `json:"inventorySlots,omitempty"`
	ResourceProduction  ResourceProduction      `json:"resourceProduction"`
	ResourceSettledAt   string                  `json:"resourceSettledAt"`
	CityGold            FlexInt                 `json:"cityGold"`
	LastExchangeAt      string                  `json:"lastExchangeAt,omitempty"`
	ProductionBoost     int                     `json:"productionBoost,omitempty"`    // 当前产量加成倍率，同倍率购买续时，不同倍率购买重算
	ProductionBoostEnd  string                  `json:"productionBoostEnd,omitempty"` // 加成到期时间
	CapacityBoost       int                     `json:"capacityBoost,omitempty"`      // 当前仓库容量加成倍率，同倍率购买续时，不同倍率购买重算
	CapacityBoostEnd    string                  `json:"capacityBoostEnd,omitempty"`   // 容量加成到期时间
	Buildings           []Building              `json:"buildings"`
	ResourceSlots       []ResourceSlot          `json:"resourceSlots,omitempty"`
	General             *General                `json:"general"`
	Generals            []General               `json:"generals,omitempty"`
	GeneralAssignments  []GeneralAssignment     `json:"generalAssignments,omitempty"`
	GeneralChangeUntil  string                  `json:"generalChangeUntil,omitempty"` // 换将冷却结束时间，空表示可更换
	Army                []ArmyUnit              `json:"army"`
	RecruitQueues       []RecruitQueue          `json:"recruitQueues"`
	NpcState            *NpcState               `json:"npcState,omitempty"`
	MapTargets          []MapTarget             `json:"mapTargets"`
	RecentBattleReports []BattleReport          `json:"recentBattleReports"`
	UnreadMessageCount  int                     `json:"unreadMessageCount"`
	UnreadMailCount     int                     `json:"unreadMailCount"`
	ActiveModifiers     []ModifierBreakdownItem `json:"activeModifiers,omitempty"`
	Buffs               []Buff                  `json:"buffs,omitempty"` // 通用加成列表（GM/活动/任务等）
	DeleteRequestedAt   string                  `json:"deleteRequestedAt,omitempty"`
	DeleteScheduledAt   string                  `json:"deleteScheduledAt,omitempty"`
	ServerTime          string                  `json:"serverTime"`
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
		Inventory: map[string]ItemStack{},
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
			{ID: "siege_camp-1", Type: "siege_camp", Level: 1},
			{ID: "special_camp-1", Type: "special_camp", Level: 1},
			{ID: "weapon_bureau-1", Type: "weapon_bureau", Level: 1},
			{ID: "armor_bureau-1", Type: "armor_bureau", Level: 1},
			{ID: "construction_bureau-1", Type: "construction_bureau", Level: 1},
			{ID: "administration-1", Type: "administration", Level: 1},
			{ID: "relay_station-1", Type: "relay_station", Level: 1},
			{ID: "city_wall-1", Type: "city_wall", Level: 1},
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
	EnsureGeneralRoster(&state, now)

	ApplyConstructionBureauResourceSlots(&state, now)
	state.ResourceProduction = calculateResourceProduction(state.Buildings, state.General)
	state.Resources.Capacity = calculateResourceCapacity(state.Buildings)
	state.ResourceSlots = BuildResourceSlotsFromBuildings(state.Buildings, now)
	return state
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
	g := &General{
		ID:    generalID,
		Name:  name,
		Level: 1,
		Exp:   0,
		Stats: map[string]int{},
		Buffs: map[string]float64{},
	}
	// 从 GeneralsConfig 注入 buffs 和特性
	applyHeroConfigToGeneral(g)
	return g
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
		ID:                state.Player.ID,
		Nickname:          state.Player.Nickname,
		Faction:           state.Player.Faction,
		TotalArmy:         totalArmy,
		BuildingLevel:     buildingLevel,
		UpdatedAt:         updatedAt.UTC().Format(time.RFC3339),
		DeleteRequestedAt: state.DeleteRequestedAt,
		DeleteScheduledAt: state.DeleteScheduledAt,
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
