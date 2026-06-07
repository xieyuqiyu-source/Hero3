package game

import "time"

// 货币流水（gold ledger）
//
// 所有 gold（账户金币） / cityGold（城金）的变动都在这里留一条不可变记录，
// 用于客服申诉、对账、补偿、异常追查。每次余额变动写一条，failed 则降级为 warn 日志。
//
// 字段语义：
//   - Currency:    "gold"（账户金币）或 "cityGold"（玩家城金）
//   - Direction:   "credit"（+） 或 "debit"（-）
//   - Amount:      正整数，方向通过 Direction 表示
//   - BalanceAfter:操作完成后的快照余额
//   - RefType:     操作大类，如 "exchange", "instant_recruit", "instant_building",
//                  "boost_purchase", "battle_overflow", "admin_adjust"
//   - RefID:       具体业务关联 ID（可空），如 queueID/buildingID/reportID/exchangeRequestID
//   - Reason:      运营/客服可读文本，GM 操作必填
//
// AccountID/PlayerID 至少一个非空：
//   - cityGold 流水必填 PlayerID，AccountID 通过 player→account 反查可补
//   - gold 流水必填 AccountID
//   - 兑换会写两条记录（一条 gold debit / 一条 cityGold credit），用相同 RefID 串起来

const (
	LedgerCurrencyGold     = "gold"
	LedgerCurrencyCityGold = "cityGold"

	LedgerDirectionCredit = "credit"
	LedgerDirectionDebit  = "debit"

	LedgerRefExchange        = "exchange"
	LedgerRefInstantRecruit  = "instant_recruit"
	LedgerRefInstantBuilding = "instant_building"
	LedgerRefBoostPurchase   = "boost_purchase"
	LedgerRefBattleOverflow  = "battle_overflow"
	LedgerRefAdminAdjust     = "admin_adjust"
)

// GoldLedgerEntry 一条货币流水记录
type GoldLedgerEntry struct {
	ID           int64  `json:"id"`
	AccountID    string `json:"accountId,omitempty"`
	PlayerID     string `json:"playerId,omitempty"`
	Currency     string `json:"currency"`
	Direction    string `json:"direction"`
	Amount       int    `json:"amount"`
	BalanceAfter int    `json:"balanceAfter"`
	RefType      string `json:"refType,omitempty"`
	RefID        string `json:"refId,omitempty"`
	Reason       string `json:"reason,omitempty"`
	CreatedAt    string `json:"createdAt"`
}

// GoldLedgerFilter 流水查询过滤条件
type GoldLedgerFilter struct {
	AccountID string
	PlayerID  string
	Currency  string // 空 = 不限
	RefType   string // 空 = 不限
	From      time.Time
	To        time.Time
	Limit     int
}
