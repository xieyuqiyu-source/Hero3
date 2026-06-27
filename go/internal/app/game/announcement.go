// 本文件定义公告系统的领域模型、投放规则和请求响应结构。
package game

const (
	AnnouncementTypeSystem       = "system"
	AnnouncementTypeMaintenance  = "maintenance"
	AnnouncementTypeUpdate       = "update"
	AnnouncementTypeActivity     = "activity"
	AnnouncementTypeCompensation = "compensation"
	AnnouncementTypeEmergency    = "emergency"

	AnnouncementStatusDraft     = "draft"
	AnnouncementStatusScheduled = "scheduled"
	AnnouncementStatusPublished = "published"
	AnnouncementStatusWithdrawn = "withdrawn"
	AnnouncementStatusArchived  = "archived"

	AnnouncementDisplayCenterOnly = "center_only"
	AnnouncementDisplayPopup      = "popup"
	AnnouncementDisplayBanner     = "banner"

	AnnouncementTargetAll            = "all"
	AnnouncementTargetPlayerIDs      = "player_ids"
	AnnouncementTargetAccountIDs     = "account_ids"
	AnnouncementTargetFactions       = "factions"
	AnnouncementTargetLevelRange     = "level_range"
	AnnouncementTargetCreatedAtRange = "created_at_range"
)

// Announcement 保存公告正文、状态、展示策略和有效期。
type Announcement struct {
	ID          string               `json:"id"`
	Title       string               `json:"title"`
	Summary     string               `json:"summary"`
	Content     string               `json:"content,omitempty"`
	Type        string               `json:"type"`
	Status      string               `json:"status"`
	DisplayMode string               `json:"displayMode"`
	Pinned      bool                 `json:"pinned"`
	Priority    int                  `json:"priority"`
	ForcePopup  bool                 `json:"forcePopup"`
	StartsAt    string               `json:"startsAt,omitempty"`
	EndsAt      string               `json:"endsAt,omitempty"`
	PublishedAt string               `json:"publishedAt,omitempty"`
	WithdrawnAt string               `json:"withdrawnAt,omitempty"`
	ArchivedAt  string               `json:"archivedAt,omitempty"`
	CreatedAt   string               `json:"createdAt"`
	UpdatedAt   string               `json:"updatedAt"`
	Targets     []AnnouncementTarget `json:"targets,omitempty"`
}

// AnnouncementTarget 保存一条公告投放规则。
type AnnouncementTarget struct {
	Type  string `json:"type"`
	Value any    `json:"value,omitempty"`
}

// AnnouncementReadState 保存玩家对公告的已读、弹出和关闭状态。
type AnnouncementReadState struct {
	AnnouncementID string `json:"announcementId"`
	PlayerID       string `json:"playerId"`
	AccountID      string `json:"accountId,omitempty"`
	IsRead         bool   `json:"isRead"`
	ReadAt         string `json:"readAt,omitempty"`
	IsPopupShown   bool   `json:"isPopupShown"`
	PopupShownAt   string `json:"popupShownAt,omitempty"`
	IsDismissed    bool   `json:"isDismissed"`
	DismissedAt    string `json:"dismissedAt,omitempty"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
}

// AnnouncementSummary 是玩家公告列表和弹窗队列返回的轻量摘要。
type AnnouncementSummary struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Summary      string `json:"summary"`
	Type         string `json:"type"`
	Status       string `json:"status"`
	DisplayMode  string `json:"displayMode"`
	Pinned       bool   `json:"pinned"`
	Priority     int    `json:"priority"`
	ForcePopup   bool   `json:"forcePopup"`
	PublishedAt  string `json:"publishedAt,omitempty"`
	StartsAt     string `json:"startsAt,omitempty"`
	EndsAt       string `json:"endsAt,omitempty"`
	IsRead       bool   `json:"isRead"`
	IsPopupShown bool   `json:"isPopupShown"`
	IsDismissed  bool   `json:"isDismissed"`
}

// AnnouncementDetail 是玩家公告详情返回，包含正文。
type AnnouncementDetail struct {
	AnnouncementSummary
	Content string `json:"content"`
}

// AnnouncementPage 是公告分页结果。
type AnnouncementPage struct {
	Items    []AnnouncementSummary `json:"items"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"pageSize"`
	Total    int                   `json:"total"`
	Unread   bool                  `json:"unread"`
}

// AdminAnnouncementPage 是 GM 公告分页结果。
type AdminAnnouncementPage struct {
	Items    []Announcement `json:"items"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
	Total    int            `json:"total"`
}

// AnnouncementPlayerContext 是公告投放判断所需的玩家上下文。
type AnnouncementPlayerContext struct {
	PlayerID  string
	AccountID string
	Faction   string
	Level     int
	CreatedAt string
}

// AnnouncementListFilter 是玩家公告列表查询条件。
type AnnouncementListFilter struct {
	PlayerID        string
	AccountID       string
	Type            string
	IncludeArchived bool
	Page            int
	PageSize        int
}

// AdminAnnouncementFilter 是 GM 公告列表查询条件。
type AdminAnnouncementFilter struct {
	Type     string
	Status   string
	Page     int
	PageSize int
}

// SaveAnnouncementRequest 是 GM 新建或编辑公告请求。
type SaveAnnouncementRequest struct {
	Title       string               `json:"title"`
	Summary     string               `json:"summary"`
	Content     string               `json:"content"`
	Type        string               `json:"type"`
	Status      string               `json:"status,omitempty"`
	DisplayMode string               `json:"displayMode"`
	Pinned      bool                 `json:"pinned"`
	Priority    int                  `json:"priority"`
	ForcePopup  bool                 `json:"forcePopup"`
	StartsAt    string               `json:"startsAt,omitempty"`
	EndsAt      string               `json:"endsAt,omitempty"`
	Targets     []AnnouncementTarget `json:"targets"`
}
