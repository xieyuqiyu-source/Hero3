package game

// 本文件定义物品流水查询和记录模型。

type ItemLedgerEntry struct {
	ID           string         `json:"id"`
	PlayerID     string         `json:"playerId"`
	ItemID       string         `json:"itemId"`
	ChangeAmount int            `json:"changeAmount"`
	BeforeAmount int            `json:"beforeAmount"`
	AfterAmount  int            `json:"afterAmount"`
	Reason       string         `json:"reason"`
	RefType      string         `json:"refType,omitempty"`
	RefID        string         `json:"refId,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	CreatedAt    string         `json:"createdAt"`
}

type ItemLedgerFilter struct {
	PlayerID string
	ItemID   string
	RefType  string
	From     string
	To       string
	Limit    int
	Offset   int
}

type ItemLedgerPage struct {
	Entries []ItemLedgerEntry `json:"entries"`
	Total   int               `json:"total"`
	Limit   int               `json:"limit"`
	Offset  int               `json:"offset"`
}
