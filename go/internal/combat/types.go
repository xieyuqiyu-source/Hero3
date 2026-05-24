package combat

// Unit 参与战斗的兵种单位
type Unit struct {
	ID              string `json:"id"`
	Category        string `json:"category"` // infantry, cavalry, siege, special
	Count           int    `json:"count"`
	Attack          int    `json:"attack"`
	InfantryDefense int    `json:"infantryDefense"`
	CavalryDefense  int    `json:"cavalryDefense"`
	CarryCapacity   int    `json:"carryCapacity"`
}

// Army 参战军队
type Army struct {
	Faction string `json:"faction"`
	Units   []Unit `json:"units"`
}

// CombatInput 战斗引擎输入
type CombatInput struct {
	RuleID   string  `json:"ruleId"`
	Attacker Army    `json:"attacker"`
	Defender Army    `json:"defender"`
	WallLevel int    `json:"wallLevel"` // 防守方城墙等级（PvE 传 0）
	WallFaction string `json:"wallFaction"` // 防守方阵营（用于城墙系数）
}

// UnitLoss 单个兵种的损失
type UnitLoss struct {
	ID     string `json:"id"`
	Count  int    `json:"count"`  // 原始数量
	Losses int    `json:"losses"` // 损失数量
}

// CombatResult 战斗引擎输出
type CombatResult struct {
	Winner           string     `json:"winner"` // "attacker", "defender", "draw"
	Mode             string     `json:"mode"`   // "attack", "plunder"
	AttackerLosses   []UnitLoss `json:"attackerLosses"`
	DefenderLosses   []UnitLoss `json:"defenderLosses"`
	AttackerLossRate float64    `json:"attackerLossRate"`
	DefenderLossRate float64    `json:"defenderLossRate"`
	AttackPower      float64    `json:"attackPower"`  // a 值
	DefensePower     float64    `json:"defensePower"` // b 值（含城墙）
	SurvivingCarry   int        `json:"survivingCarry"` // 存活部队总运载量
}
