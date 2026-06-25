package reward

const (
	TypeResource   = "resource"
	TypeCityGold   = "city_gold"
	TypeGold       = "gold"
	TypeItem       = "item"
	TypeUnit       = "unit"
	TypeGeneralExp = "general_exp"
	TypeBuff       = "buff"
)

type Reward struct {
	Type     string         `json:"type"`
	ID       string         `json:"id"`
	Amount   int            `json:"amount"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type GrantContext struct {
	AccountID string
	PlayerID  string
	RefType   string
	RefID     string
	Reason    string
}
