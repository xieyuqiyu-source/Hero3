// 本文件实现万象幻境小游戏记录、后端结算和库存兑换。
package game

import (
	cryptorand "crypto/rand"
	"math/big"
	"strconv"
	"strings"
	"time"
)

// MiniGameRecord 小游戏记录（钓鱼/赌博/老虎机等）。
type MiniGameRecord struct {
	ID              string `json:"id"`
	PlayerID        string `json:"playerId"`
	GameType        string `json:"gameType"`            // "fishing" | "gambling" | "slot"
	ResultName      string `json:"resultName"`          // "金龙鱼" / "猜大 赢 ×2" / "赤金符三连 ×30"
	Rarity          string `json:"rarity"`              // "common" | "rare" | "epic" | "legendary"
	RewardUnit      string `json:"rewardUnit"`          // 赢得的兵种名称
	RewardAmount    int    `json:"rewardAmount"`        // 原始赢得数量
	RemainingAmount int    `json:"remainingAmount"`     // 剩余可兑换数量
	BetUnit         string `json:"betUnit,omitempty"`   // 押注的兵种名称（赌博用）
	BetAmount       int    `json:"betAmount,omitempty"` // 押注的数量（赌博用）
	CreatedAt       string `json:"createdAt"`
}

// MiniGameSummary GM 查询用的汇总信息
type MiniGameSummary struct {
	TotalRecords int              `json:"totalRecords"`
	Limit        int              `json:"limit"`
	Offset       int              `json:"offset"`
	HasMore      bool             `json:"hasMore"`
	Records      []MiniGameRecord `json:"records"`
	RewardTotals map[string]int   `json:"rewardTotals"` // 兵种名 → 总可兑换数量
}

type MiniGameRedeemResult struct {
	Record         MiniGameRecord `json:"record"`
	Army           []ArmyUnit     `json:"army,omitempty"`
	ServerTime     string         `json:"serverTime"`
	RedeemedUnitID string         `json:"redeemedUnitId"`
	RedeemedUnit   string         `json:"redeemedUnit"`
	RedeemedAmount int            `json:"redeemedAmount"`
	RedeemedTarget string         `json:"redeemedTarget"`
	Garrison       *Reinforcement `json:"garrison,omitempty"`
	GrantedRewards []Reward       `json:"grantedRewards,omitempty"`
}

type MiniGameRedeemAllResult struct {
	Army            []ArmyUnit     `json:"army,omitempty"`
	ServerTime      string         `json:"serverTime"`
	RedeemedUnits   map[string]int `json:"redeemedUnits"`
	RedeemedAmount  int            `json:"redeemedAmount"`
	RedeemedRecords int            `json:"redeemedRecords"`
	GarrisonedUnits map[string]int `json:"garrisonedUnits,omitempty"`
	GarrisonRecords int            `json:"garrisonRecords,omitempty"`
	SkippedUnits    map[string]int `json:"skippedUnits"`
	SkippedRecords  int            `json:"skippedRecords"`
	GrantedRewards  []Reward       `json:"grantedRewards,omitempty"`
}

type GamblingRoundResult struct {
	Record       MiniGameRecord `json:"record"`
	Army         []ArmyUnit     `json:"army,omitempty"`
	ServerTime   string         `json:"serverTime"`
	Won          bool           `json:"won"`
	Multiplier   int            `json:"multiplier"`
	BetUnitID    string         `json:"betUnitId"`
	BetUnit      string         `json:"betUnit"`
	BetAmount    int            `json:"betAmount"`
	WinAmount    int            `json:"winAmount"`
	DiceTotal    int            `json:"diceTotal"`
	DiceValues   []int          `json:"diceValues"`
	BetLabel     string         `json:"betLabel"`
	RewardRarity string         `json:"rewardRarity"`
}

type SlotRoundResult struct {
	Record        MiniGameRecord       `json:"record"`
	Army          []ArmyUnit           `json:"army,omitempty"`
	ServerTime    string               `json:"serverTime"`
	Won           bool                 `json:"won"`
	Grid          [][]string           `json:"grid"`
	LineBet       int                  `json:"lineBet"`
	LineCount     int                  `json:"lineCount"`
	TotalBet      int                  `json:"totalBet"`
	WinningLines  []SlotWinningLine    `json:"winningLines"`
	FreeSpins     []SlotFreeSpinResult `json:"freeSpins"`
	BonusRewards  []SlotBonusReward    `json:"bonusRewards"`
	AllPayRewards []SlotAllPayReward   `json:"allPayRewards"`
	BetUnitID     string               `json:"betUnitId"`
	BetUnit       string               `json:"betUnit"`
	BetAmount     int                  `json:"betAmount"`
	WinAmount     int                  `json:"winAmount"`
	RewardRarity  string               `json:"rewardRarity"`
}

type SlotWinningLine struct {
	LineID     string  `json:"lineId"`
	Symbol     string  `json:"symbol"`
	SymbolName string  `json:"symbolName"`
	Multiplier int     `json:"multiplier"`
	Amount     int     `json:"amount"`
	Positions  [][]int `json:"positions"`
}

type SlotBonusReward struct {
	Multiplier int     `json:"multiplier"`
	Amount     int     `json:"amount"`
	Positions  [][]int `json:"positions"`
}

type SlotAllPayReward struct {
	Symbol     string  `json:"symbol"`
	SymbolName string  `json:"symbolName"`
	Count      int     `json:"count"`
	Multiplier int     `json:"multiplier"`
	Amount     int     `json:"amount"`
	Positions  [][]int `json:"positions"`
}

type SlotFreeSpinResult struct {
	SpinIndex            int                `json:"spinIndex"`
	Grid                 [][]string         `json:"grid"`
	WinningLines         []SlotWinningLine  `json:"winningLines"`
	BonusRewards         []SlotBonusReward  `json:"bonusRewards"`
	AllPayRewards        []SlotAllPayReward `json:"allPayRewards"`
	ScatterCount         int                `json:"scatterCount"`
	RetriggeredFreeSpins int                `json:"retriggeredFreeSpins"`
	WinAmount            int                `json:"winAmount"`
}

type FishingBaitUseResult struct {
	BaitID         string   `json:"baitId"`
	CityGold       *FlexInt `json:"cityGold,omitempty"`
	ServerTime     string   `json:"serverTime"`
	CityGoldCost   int      `json:"cityGoldCost"`
	CityGoldRemain *int     `json:"cityGoldRemain,omitempty"`
}

func fishingBaitCost(baitID string) (int, bool) {
	return GetFishingBaitCost(baitID)
}

// miniGameRecordHasRedeemableReward 判断小游戏记录是否允许沉淀为可兑换库存。
func miniGameRecordHasRedeemableReward(gameType string) bool {
	switch strings.TrimSpace(gameType) {
	case "fishing", "gambling", "slot":
		return true
	default:
		return false
	}
}

var gamblingExactOdds = map[int]int{
	3: 150, 4: 60, 5: 30, 6: 18, 7: 12, 8: 8, 9: 6, 10: 6,
	11: 6, 12: 6, 13: 8, 14: 12, 15: 18, 16: 30, 17: 60, 18: 150,
}

var slotGridRoller = rollSlotGrid
var slotBonusRoller = rollSlotBonusMultiplier

const slotMaxRewardAmount = 2147000000

var slotPaylines = []struct {
	id        string
	positions [][2]int
}{
	{id: "middle", positions: [][2]int{{1, 0}, {1, 1}, {1, 2}}},
	{id: "top", positions: [][2]int{{0, 0}, {0, 1}, {0, 2}}},
	{id: "bottom", positions: [][2]int{{2, 0}, {2, 1}, {2, 2}}},
	{id: "diagonal_down", positions: [][2]int{{0, 0}, {1, 1}, {2, 2}}},
	{id: "diagonal_up", positions: [][2]int{{2, 0}, {1, 1}, {0, 2}}},
}

var slotAllPayCountMultipliers = map[int]int{
	5: 2,
	6: 4,
	7: 8,
	8: 15,
	9: 30,
}

// ResolveGamblingRound 由后端完成赌场一局结算：扣押注、掷骰、写记录并返回最新军队。
func (s *Service) ResolveGamblingRound(playerID string, betUnitType string, betAmount int, betID string, exactNumber int) (GamblingRoundResult, error) {
	playerID = strings.TrimSpace(playerID)
	betUnitType = strings.TrimSpace(betUnitType)
	betID = strings.TrimSpace(betID)
	if playerID == "" {
		return GamblingRoundResult{}, ErrPlayerNotFound
	}
	if betUnitType == "" {
		return GamblingRoundResult{}, ErrUnitNotFound
	}
	if betAmount <= 0 {
		return GamblingRoundResult{}, ErrInvalidAmount
	}

	unlock := s.getPlayerLock(playerID)
	unlock.Lock()
	defer unlock.Unlock()

	now := time.Now()
	nowText := now.UTC().Format(resourceDateLayout)
	var result GamblingRoundResult
	state, _, err := s.repo.UpdateMiniGamePlayerState(playerID, now, func(state *GameState, records []MiniGameRecord) ([]MiniGameRecord, error) {
		unitCfg, exists := GetUnitConfig(state.Player.Faction, betUnitType)
		if !exists {
			return nil, ErrUnitNotFound
		}
		if isNonCombatUnit(unitCfg) {
			return nil, ErrNonCombatUnit
		}
		if _, err := validateAndConsumeArmy(state, map[string]int{betUnitType: betAmount}); err != nil {
			return nil, err
		}

		dice, err := rollGamblingDice()
		if err != nil {
			return nil, err
		}
		won, multiplier, betLabel, err := evaluateGamblingBet(betID, exactNumber, dice)
		if err != nil {
			return nil, err
		}
		winAmount := 0
		rewardUnit := ""
		if won {
			winAmount = betAmount * multiplier
			rewardUnit = unitCfg.Name
		}
		rarity := gamblingRewardRarity(won, multiplier)
		resultName := gamblingResultName(betLabel, won, multiplier)
		record := MiniGameRecord{
			ID:              "mg_" + randomID(10),
			PlayerID:        playerID,
			GameType:        "gambling",
			ResultName:      resultName,
			Rarity:          rarity,
			RewardUnit:      rewardUnit,
			RewardAmount:    winAmount,
			RemainingAmount: winAmount,
			BetUnit:         unitCfg.Name,
			BetAmount:       betAmount,
			CreatedAt:       nowText,
		}
		if !won {
			record.RemainingAmount = 0
		}
		state.ServerTime = nowText
		records = append([]MiniGameRecord{record}, records...)
		result = GamblingRoundResult{
			Record:       record,
			ServerTime:   nowText,
			Won:          won,
			Multiplier:   multiplier,
			BetUnitID:    betUnitType,
			BetUnit:      unitCfg.Name,
			BetAmount:    betAmount,
			WinAmount:    winAmount,
			DiceTotal:    dice[0] + dice[1] + dice[2],
			DiceValues:   []int{dice[0], dice[1], dice[2]},
			BetLabel:     betLabel,
			RewardRarity: rarity,
		}
		return records, nil
	})
	if err != nil {
		return GamblingRoundResult{}, err
	}
	result.Army = state.Army
	result.ServerTime = state.ServerTime
	return result, nil
}

// ResolveSlotRound 由后端完成天机轮转一局结算：扣单次押注、生成 3x3 图案、展开免费旋转、写入一条库存记录。
func (s *Service) ResolveSlotRound(playerID string, betUnitType string, lineBet int) (SlotRoundResult, error) {
	playerID = strings.TrimSpace(playerID)
	betUnitType = strings.TrimSpace(betUnitType)
	if playerID == "" {
		return SlotRoundResult{}, ErrPlayerNotFound
	}
	if betUnitType == "" {
		return SlotRoundResult{}, ErrUnitNotFound
	}

	cfg := GetSlotConfig()
	if lineBet < cfg.MinLineBet {
		return SlotRoundResult{}, ErrMiniGameBetTooLow
	}

	unlock, ok := s.tryPlayerLockIfIdle(playerID)
	if !ok {
		return SlotRoundResult{}, ErrOperationTooFast
	}
	defer unlock()

	now := time.Now()
	nowText := now.UTC().Format(resourceDateLayout)
	var result SlotRoundResult
	state, _, err := s.repo.UpdateMiniGamePlayerState(playerID, now, func(state *GameState, records []MiniGameRecord) ([]MiniGameRecord, error) {
		unitCfg, exists := GetUnitConfig(state.Player.Faction, betUnitType)
		if !exists {
			return nil, ErrUnitNotFound
		}
		if isSlotBetUnitBlocked(unitCfg) {
			return nil, ErrNonCombatUnit
		}

		currentAmount := armyAmountByUnitType(state.Army, betUnitType)
		if currentAmount <= 0 {
			return nil, ErrInsufficientArmy
		}
		totalBet := lineBet
		if _, err := validateAndConsumeArmy(state, map[string]int{betUnitType: totalBet}); err != nil {
			return nil, err
		}

		grid, err := slotGridRoller(cfg)
		if err != nil {
			return nil, err
		}
		mainSpin, err := evaluateSlotSpin(cfg, grid, lineBet)
		if err != nil {
			return nil, err
		}
		totalWin := mainSpin.winAmount
		freeSpins := []SlotFreeSpinResult{}
		remainingFreeSpins := mainSpin.triggeredFreeSpins
		for remainingFreeSpins > 0 && len(freeSpins) < cfg.MaxFreeSpinsPerRound {
			remainingFreeSpins--
			freeGrid, err := slotGridRoller(cfg)
			if err != nil {
				return nil, err
			}
			spin, err := evaluateSlotSpin(cfg, freeGrid, lineBet)
			if err != nil {
				return nil, err
			}
			retriggered := 0
			if spin.triggeredFreeSpins > 0 {
				retriggered = spin.retriggerFreeSpins
				remainingFreeSpins += retriggered
			}
			totalWin += spin.winAmount
			freeSpins = append(freeSpins, SlotFreeSpinResult{
				SpinIndex:            len(freeSpins) + 1,
				Grid:                 slotGridIDs(freeGrid),
				WinningLines:         nonNilSlotWinningLines(spin.winningLines),
				BonusRewards:         nonNilSlotBonusRewards(spin.bonusRewards),
				AllPayRewards:        nonNilSlotAllPayRewards(spin.allPayRewards),
				ScatterCount:         spin.scatterCount,
				RetriggeredFreeSpins: retriggered,
				WinAmount:            spin.winAmount,
			})
		}
		if totalWin > slotMaxRewardAmount {
			return nil, ErrInvalidMiniGame
		}

		rewardUnit := ""
		rarity := "common"
		resultName := "未中奖"
		if totalWin > 0 {
			rewardUnit = unitCfg.Name
			rarity = highestSlotRarity(mainSpin, freeSpins)
			resultName = slotResultName(mainSpin, freeSpins)
		}
		record := MiniGameRecord{
			ID:              "mg_" + randomID(10),
			PlayerID:        playerID,
			GameType:        "slot",
			ResultName:      resultName,
			Rarity:          rarity,
			RewardUnit:      rewardUnit,
			RewardAmount:    totalWin,
			RemainingAmount: totalWin,
			BetUnit:         unitCfg.Name,
			BetAmount:       totalBet,
			CreatedAt:       nowText,
		}
		if totalWin <= 0 {
			record.RemainingAmount = 0
		}
		state.ServerTime = nowText
		records = append([]MiniGameRecord{record}, records...)
		result = SlotRoundResult{
			Record:        record,
			ServerTime:    nowText,
			Won:           totalWin > 0,
			Grid:          slotGridIDs(grid),
			LineBet:       lineBet,
			LineCount:     cfg.LineCount,
			TotalBet:      totalBet,
			WinningLines:  nonNilSlotWinningLines(mainSpin.winningLines),
			FreeSpins:     nonNilSlotFreeSpins(freeSpins),
			BonusRewards:  nonNilSlotBonusRewards(mainSpin.bonusRewards),
			AllPayRewards: nonNilSlotAllPayRewards(mainSpin.allPayRewards),
			BetUnitID:     betUnitType,
			BetUnit:       unitCfg.Name,
			BetAmount:     totalBet,
			WinAmount:     totalWin,
			RewardRarity:  rarity,
		}
		return records, nil
	})
	if err != nil {
		return SlotRoundResult{}, err
	}
	result.Army = state.Army
	result.ServerTime = state.ServerTime
	return result, nil
}

// nonNilSlotWinningLines 保证接口空中奖线编码为 [] 而不是 null。
func nonNilSlotWinningLines(lines []SlotWinningLine) []SlotWinningLine {
	if lines == nil {
		return []SlotWinningLine{}
	}
	return lines
}

// nonNilSlotBonusRewards 保证接口空宝匣奖励编码为 [] 而不是 null。
func nonNilSlotBonusRewards(rewards []SlotBonusReward) []SlotBonusReward {
	if rewards == nil {
		return []SlotBonusReward{}
	}
	return rewards
}

// nonNilSlotAllPayRewards 保证接口空满天星奖励编码为 [] 而不是 null。
func nonNilSlotAllPayRewards(rewards []SlotAllPayReward) []SlotAllPayReward {
	if rewards == nil {
		return []SlotAllPayReward{}
	}
	return rewards
}

// nonNilSlotFreeSpins 保证接口空免费旋转编码为 [] 而不是 null。
func nonNilSlotFreeSpins(spins []SlotFreeSpinResult) []SlotFreeSpinResult {
	if spins == nil {
		return []SlotFreeSpinResult{}
	}
	return spins
}

// isSlotBetUnitBlocked 判断天机轮转是否禁止该兵种押注。
func isSlotBetUnitBlocked(unitCfg UnitConfig) bool {
	if unitCfg.Role == "scout" || unitCfg.Role == "transport" {
		return true
	}
	return isNonCombatUnit(unitCfg)
}

// armyAmountByUnitType 获取玩家当前某兵种数量。
func armyAmountByUnitType(army []ArmyUnit, unitType string) int {
	for _, unit := range army {
		if unit.UnitType == unitType {
			return unit.Amount
		}
	}
	return 0
}

// rollSlotGrid 使用服务器加密随机数按权重抽取 3x3 可视窗口。
func rollSlotGrid(cfg SlotConfig) ([3][3]SlotSymbol, error) {
	var result [3][3]SlotSymbol
	totalWeight := 0
	for _, symbol := range cfg.Symbols {
		totalWeight += symbol.Weight
	}
	if totalWeight <= 0 {
		return result, ErrInvalidMiniGame
	}
	for row := range result {
		for col := range result[row] {
			symbol, err := rollWeightedSlotSymbol(cfg.Symbols, totalWeight)
			if err != nil {
				return result, err
			}
			result[row][col] = symbol
		}
	}
	return result, nil
}

// rollWeightedSlotSymbol 按权重抽取一个天机轮转图案。
func rollWeightedSlotSymbol(symbols []SlotSymbol, totalWeight int) (SlotSymbol, error) {
	n, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(totalWeight)))
	if err != nil {
		return SlotSymbol{}, err
	}
	pick := int(n.Int64()) + 1
	running := 0
	for _, symbol := range symbols {
		running += symbol.Weight
		if pick <= running {
			return symbol, nil
		}
	}
	return SlotSymbol{}, ErrInvalidMiniGame
}

// rollSlotBonusMultiplier 按宝匣配置抽取奖励倍率。
func rollSlotBonusMultiplier(symbol SlotSymbol) (int, error) {
	totalWeight := 0
	for _, bonus := range symbol.BonusMultipliers {
		totalWeight += bonus.Weight
	}
	if totalWeight <= 0 {
		return 0, ErrInvalidMiniGame
	}
	n, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(totalWeight)))
	if err != nil {
		return 0, err
	}
	pick := int(n.Int64()) + 1
	running := 0
	for _, bonus := range symbol.BonusMultipliers {
		running += bonus.Weight
		if pick <= running {
			return bonus.Multiplier, nil
		}
	}
	return 0, ErrInvalidMiniGame
}

type slotSpinEvaluation struct {
	winningLines       []SlotWinningLine
	bonusRewards       []SlotBonusReward
	allPayRewards      []SlotAllPayReward
	scatterCount       int
	triggeredFreeSpins int
	retriggerFreeSpins int
	winAmount          int
	bestLine           *SlotWinningLine
	bestRarity         string
}

// evaluateSlotSpin 结算一个 3x3 窗口中的赔付线、Scatter 和 Bonus。
func evaluateSlotSpin(cfg SlotConfig, grid [3][3]SlotSymbol, lineBet int) (slotSpinEvaluation, error) {
	result := slotSpinEvaluation{bestRarity: "common"}
	symbols := slotSymbolsByID(cfg)
	for _, line := range slotPaylines {
		win, ok := evaluateSlotPayline(grid, symbols, line.id, line.positions, lineBet)
		if !ok {
			continue
		}
		result.winningLines = append(result.winningLines, win)
		result.winAmount += win.Amount
		rarity := symbols[win.Symbol].Rarity
		if slotRarityRank(rarity) > slotRarityRank(result.bestRarity) {
			result.bestRarity = rarity
		}
		if result.bestLine == nil || win.Amount > result.bestLine.Amount {
			copied := win
			result.bestLine = &copied
		}
	}
	var scatterSymbol *SlotSymbol
	var bonusSymbol *SlotSymbol
	bonusCount := 0
	wildPositions := [][2]int{}
	bonusPositions := [][2]int{}
	normalPositions := map[string][][2]int{}
	for row := range grid {
		for col := range grid[row] {
			symbol := grid[row][col]
			switch symbol.Type {
			case "normal":
				normalPositions[symbol.ID] = append(normalPositions[symbol.ID], [2]int{row, col})
			case "wild":
				wildPositions = append(wildPositions, [2]int{row, col})
			case "scatter":
				result.scatterCount++
				scatterCopy := symbol
				scatterSymbol = &scatterCopy
			case "bonus":
				bonusCount++
				bonusPositions = append(bonusPositions, [2]int{row, col})
				bonusCopy := symbol
				bonusSymbol = &bonusCopy
			}
		}
	}
	effectiveScatterCount := result.scatterCount
	if result.scatterCount > 0 && result.scatterCount+len(wildPositions) >= 3 {
		effectiveScatterCount = 3
	}
	if effectiveScatterCount >= 3 && scatterSymbol != nil {
		result.scatterCount = effectiveScatterCount
		result.triggeredFreeSpins = scatterSymbol.FreeSpins
		result.retriggerFreeSpins = scatterSymbol.RetriggerFreeSpins
		if slotRarityRank(scatterSymbol.Rarity) > slotRarityRank(result.bestRarity) {
			result.bestRarity = scatterSymbol.Rarity
		}
	}
	bonusTriggerPositions := slotSpecialTriggerPositions(bonusPositions, wildPositions)
	if len(bonusTriggerPositions) >= 3 && bonusSymbol != nil {
		multiplier, err := slotBonusRoller(*bonusSymbol)
		if err != nil {
			return result, err
		}
		amount := lineBet * multiplier
		result.bonusRewards = append(result.bonusRewards, SlotBonusReward{Multiplier: multiplier, Amount: amount, Positions: bonusTriggerPositions})
		result.winAmount += amount
		if slotRarityRank(bonusSymbol.Rarity) > slotRarityRank(result.bestRarity) {
			result.bestRarity = bonusSymbol.Rarity
		}
	}
	if reward, ok := evaluateSlotAllPay(symbols, normalPositions, wildPositions, lineBet); ok {
		result.allPayRewards = append(result.allPayRewards, reward)
		result.winAmount += reward.Amount
		if slotRarityRank(symbols[reward.Symbol].Rarity) > slotRarityRank(result.bestRarity) {
			result.bestRarity = symbols[reward.Symbol].Rarity
		}
	}
	return result, nil
}

// evaluateSlotPayline 判断一条固定赔付线是否中奖。
func evaluateSlotPayline(grid [3][3]SlotSymbol, symbols map[string]SlotSymbol, lineID string, positions [][2]int, lineBet int) (SlotWinningLine, bool) {
	normalID := ""
	for _, pos := range positions {
		symbol := grid[pos[0]][pos[1]]
		if symbol.Type == "scatter" || symbol.Type == "bonus" {
			return SlotWinningLine{}, false
		}
		if symbol.Type == "normal" {
			if normalID == "" {
				normalID = symbol.ID
				continue
			}
			if normalID != symbol.ID {
				return SlotWinningLine{}, false
			}
		}
	}
	if normalID == "" {
		normalID = "heaven_order"
	}
	target, ok := symbols[normalID]
	if !ok || target.Type != "normal" || target.Multiplier <= 0 {
		return SlotWinningLine{}, false
	}
	return SlotWinningLine{
		LineID:     lineID,
		Symbol:     target.ID,
		SymbolName: target.Name,
		Multiplier: target.Multiplier,
		Amount:     lineBet * target.Multiplier,
		Positions:  slotLinePositions(positions),
	}, true
}

// evaluateSlotAllPay 计算全屏满天星奖励，天机令可补齐，但同局只取最佳一组。
func evaluateSlotAllPay(symbols map[string]SlotSymbol, normalPositions map[string][][2]int, wildPositions [][2]int, lineBet int) (SlotAllPayReward, bool) {
	best := SlotAllPayReward{}
	hasBest := false
	for symbolID, positions := range normalPositions {
		symbol, ok := symbols[symbolID]
		if !ok || symbol.Type != "normal" || symbol.Multiplier <= 0 {
			continue
		}
		effectiveCount := len(positions) + len(wildPositions)
		if effectiveCount > 9 {
			effectiveCount = 9
		}
		countMultiplier, ok := slotAllPayCountMultipliers[effectiveCount]
		if !ok {
			continue
		}
		triggerPositions := slotAllPayPositions(positions, wildPositions, effectiveCount)
		reward := SlotAllPayReward{
			Symbol:     symbol.ID,
			SymbolName: symbol.Name,
			Count:      effectiveCount,
			Multiplier: symbol.Multiplier * countMultiplier,
			Amount:     lineBet * symbol.Multiplier * countMultiplier,
			Positions:  triggerPositions,
		}
		if !hasBest || reward.Amount > best.Amount || (reward.Amount == best.Amount && slotRarityRank(symbol.Rarity) > slotRarityRank(symbols[best.Symbol].Rarity)) {
			best = reward
			hasBest = true
		}
	}
	return best, hasBest
}

// slotGridIDs 转换 3x3 图案为接口约定的行优先 ID 矩阵。
func slotGridIDs(grid [3][3]SlotSymbol) [][]string {
	result := make([][]string, 3)
	for row := range grid {
		result[row] = make([]string, 3)
		for col := range grid[row] {
			result[row][col] = grid[row][col].ID
		}
	}
	return result
}

// slotAllPayPositions 组合真实普通图案和必要天机令坐标。
func slotAllPayPositions(normalPositions [][2]int, wildPositions [][2]int, count int) [][]int {
	combined := make([][2]int, 0, count)
	combined = append(combined, normalPositions...)
	for _, pos := range wildPositions {
		if len(combined) >= count {
			break
		}
		combined = append(combined, pos)
	}
	if len(combined) > count {
		combined = combined[:count]
	}
	return slotLinePositions(combined)
}

// slotLinePositions 转换赔付线坐标，避免共享底层数组。
func slotLinePositions(positions [][2]int) [][]int {
	result := make([][]int, 0, len(positions))
	for _, pos := range positions {
		result = append(result, []int{pos[0], pos[1]})
	}
	return result
}

// slotSpecialTriggerPositions 返回特殊图案被天机令补齐后的触发坐标。
func slotSpecialTriggerPositions(specialPositions [][2]int, wildPositions [][2]int) [][]int {
	if len(specialPositions) == 0 || len(specialPositions)+len(wildPositions) < 3 {
		return [][]int{}
	}
	combined := make([][2]int, 0, 3)
	combined = append(combined, specialPositions...)
	for _, pos := range wildPositions {
		if len(combined) >= 3 {
			break
		}
		combined = append(combined, pos)
	}
	return slotLinePositions(combined[:3])
}

// slotSymbolsByID 建立图案索引，便于赔付线解析。
func slotSymbolsByID(cfg SlotConfig) map[string]SlotSymbol {
	result := map[string]SlotSymbol{}
	for _, symbol := range cfg.Symbols {
		result[symbol.ID] = symbol
	}
	return result
}

// highestSlotRarity 汇总主旋转和免费旋转中最高记录稀有度。
func highestSlotRarity(main slotSpinEvaluation, freeSpins []SlotFreeSpinResult) string {
	rarity := main.bestRarity
	for _, spin := range freeSpins {
		for _, line := range spin.WinningLines {
			if slotRarityRank(lineRarityFromName(line.Symbol)) > slotRarityRank(rarity) {
				rarity = lineRarityFromName(line.Symbol)
			}
		}
		for _, reward := range spin.AllPayRewards {
			if slotRarityRank(lineRarityFromName(reward.Symbol)) > slotRarityRank(rarity) {
				rarity = lineRarityFromName(reward.Symbol)
			}
		}
		if len(spin.BonusRewards) > 0 && slotRarityRank("rare") > slotRarityRank(rarity) {
			rarity = "rare"
		}
		if spin.ScatterCount >= 3 && slotRarityRank("epic") > slotRarityRank(rarity) {
			rarity = "epic"
		}
	}
	if rarity == "" {
		return "common"
	}
	return rarity
}

// lineRarityFromName 按第二版固定图案 ID 估算免费旋转最高稀有度。
func lineRarityFromName(symbolID string) string {
	switch symbolID {
	case "heaven_order":
		return "legendary"
	case "jade_seal", "tiger_tally", "wild":
		return "epic"
	case "silver_charm", "gold_charm":
		return "rare"
	default:
		return "common"
	}
}

// slotResultName 生成库存记录中的简短结果名。
func slotResultName(main slotSpinEvaluation, freeSpins []SlotFreeSpinResult) string {
	parts := []string{}
	if main.triggeredFreeSpins > 0 {
		parts = append(parts, "星陨免费旋转")
	}
	if len(main.bonusRewards) > 0 {
		parts = append(parts, "宝匣奖励 ×"+strconv.Itoa(main.bonusRewards[0].Multiplier))
	}
	if len(main.allPayRewards) > 0 {
		reward := main.allPayRewards[0]
		parts = append(parts, reward.SymbolName+"满天星"+strconv.Itoa(reward.Count)+" ×"+strconv.Itoa(reward.Multiplier))
	}
	if main.bestLine != nil {
		parts = append(parts, main.bestLine.SymbolName+"三连 ×"+strconv.Itoa(main.bestLine.Multiplier))
	}
	for _, spin := range freeSpins {
		if len(spin.BonusRewards) > 0 {
			parts = append(parts, "宝匣奖励 ×"+strconv.Itoa(spin.BonusRewards[0].Multiplier))
		}
		if len(spin.AllPayRewards) > 0 {
			reward := spin.AllPayRewards[0]
			parts = append(parts, reward.SymbolName+"满天星"+strconv.Itoa(reward.Count)+" ×"+strconv.Itoa(reward.Multiplier))
		}
		if len(spin.WinningLines) > 0 {
			line := spin.WinningLines[0]
			parts = append(parts, line.SymbolName+"三连 ×"+strconv.Itoa(line.Multiplier))
		}
		if len(parts) >= 3 {
			break
		}
	}
	if len(parts) == 0 {
		return "未中奖"
	}
	if len(parts) > 3 {
		parts = parts[:3]
	}
	return strings.Join(parts, " + ")
}

// slotRarityRank 用于比较记录稀有度。
func slotRarityRank(rarity string) int {
	switch rarity {
	case "legendary":
		return 4
	case "epic":
		return 3
	case "rare":
		return 2
	default:
		return 1
	}
}

// rollGamblingDice 生成三颗 1-6 的服务器骰子。
func rollGamblingDice() ([3]int, error) {
	var dice [3]int
	for i := range dice {
		n, err := cryptorand.Int(cryptorand.Reader, big.NewInt(6))
		if err != nil {
			return dice, err
		}
		dice[i] = int(n.Int64()) + 1
	}
	return dice, nil
}

// evaluateGamblingBet 根据押注玩法和骰子判断输赢、赔率和展示标签。
func evaluateGamblingBet(betID string, exactNumber int, dice [3]int) (bool, int, string, error) {
	total := dice[0] + dice[1] + dice[2]
	isTriple := dice[0] == dice[1] && dice[1] == dice[2]
	switch betID {
	case "big":
		return total >= 11 && !isTriple, 2, "大", nil
	case "small":
		return total <= 10 && !isTriple, 2, "小", nil
	case "odd":
		return total%2 == 1 && !isTriple, 2, "单", nil
	case "even":
		return total%2 == 0 && !isTriple, 2, "双", nil
	case "triple":
		return isTriple, 30, "豹子", nil
	case "exact":
		odds, ok := gamblingExactOdds[exactNumber]
		if !ok {
			return false, 0, "", ErrInvalidMiniGame
		}
		return total == exactNumber, odds, "猜" + strconv.Itoa(exactNumber) + "点", nil
	default:
		return false, 0, "", ErrInvalidMiniGame
	}
}

// gamblingRewardRarity 根据赔率转换赌场记录稀有度。
func gamblingRewardRarity(won bool, multiplier int) string {
	if !won {
		return "common"
	}
	if multiplier >= 30 {
		return "legendary"
	}
	if multiplier >= 10 {
		return "epic"
	}
	return "rare"
}

// gamblingResultName 生成赌场记录展示名称。
func gamblingResultName(betLabel string, won bool, multiplier int) string {
	if won {
		return betLabel + " 赢 ×" + strconv.Itoa(multiplier)
	}
	return betLabel + " 输"
}

// SaveMiniGameRecord 保存一条小游戏记录
func (s *Service) SaveMiniGameRecord(playerID string, gameType string, resultName string, rarity string, rewardUnit string, rewardAmount int, betUnit string, betAmount int) (MiniGameRecord, error) {
	playerID = strings.TrimSpace(playerID)
	gameType = strings.TrimSpace(gameType)
	if playerID == "" {
		return MiniGameRecord{}, ErrPlayerNotFound
	}
	if gameType == "slot" {
		return MiniGameRecord{}, ErrInvalidMiniGame
	}

	now := time.Now()
	record := MiniGameRecord{
		ID:           "mg_" + randomID(10),
		PlayerID:     playerID,
		GameType:     gameType,
		ResultName:   resultName,
		Rarity:       rarity,
		RewardUnit:   rewardUnit,
		RewardAmount: rewardAmount,
		BetUnit:      betUnit,
		BetAmount:    betAmount,
		CreatedAt:    now.UTC().Format(resourceDateLayout),
	}
	if miniGameRecordHasRedeemableReward(record.GameType) && record.RewardAmount > 0 {
		record.RemainingAmount = record.RewardAmount
	}

	if err := s.repo.SaveMiniGameRecord(record); err != nil {
		return MiniGameRecord{}, err
	}

	return record, nil
}

func (s *Service) UseFishingBait(playerID string, baitID string) (FishingBaitUseResult, error) {
	playerID = strings.TrimSpace(playerID)
	baitID = strings.TrimSpace(baitID)
	if playerID == "" {
		return FishingBaitUseResult{}, ErrPlayerNotFound
	}
	cost, ok := fishingBaitCost(baitID)
	if !ok {
		return FishingBaitUseResult{}, ErrInvalidBait
	}

	now := time.Now()
	refID := "fishing_bait_" + baitID
	var state GameState
	var cityGold *FlexInt
	var cityGoldRemain *int
	if cost > 0 {
		var err error
		state, err = s.repo.UpdateScopedRewardState(playerID, RewardAssetScope{Currency: true}, now, func(state *GameState) error {
			if int(state.CityGold) < cost {
				return ErrInsufficientCityGold
			}
			state.CityGold -= FlexInt(cost)
			state.ServerTime = now.UTC().Format(resourceDateLayout)
			return nil
		})
		if err != nil {
			return FishingBaitUseResult{}, err
		}
		s.recordLedger(GoldLedgerEntry{
			PlayerID:     playerID,
			Currency:     LedgerCurrencyCityGold,
			Direction:    LedgerDirectionDebit,
			Amount:       cost,
			BalanceAfter: int(state.CityGold),
			RefType:      LedgerRefMiniGameBait,
			RefID:        refID,
			Reason:       "钓鱼鱼饵消耗",
		})
		s.publishCurrencyChanged(playerID, "", refID, LedgerRefMiniGameBait)
		currentCityGold := state.CityGold
		currentCityGoldRemain := int(state.CityGold)
		cityGold = &currentCityGold
		cityGoldRemain = &currentCityGoldRemain
	} else {
		state.ServerTime = now.UTC().Format(resourceDateLayout)
	}

	return FishingBaitUseResult{
		BaitID:         baitID,
		CityGold:       cityGold,
		ServerTime:     state.ServerTime,
		CityGoldCost:   cost,
		CityGoldRemain: cityGoldRemain,
	}, nil
}

func (s *Service) RedeemMiniGameReward(playerID string, recordID string, amount int) (MiniGameRedeemResult, error) {
	playerID = strings.TrimSpace(playerID)
	recordID = strings.TrimSpace(recordID)
	if playerID == "" {
		return MiniGameRedeemResult{}, ErrPlayerNotFound
	}
	if recordID == "" {
		return MiniGameRedeemResult{}, ErrMiniGameNotFound
	}
	if amount <= 0 {
		return MiniGameRedeemResult{}, ErrInvalidAmount
	}
	redeemedAt := time.Now()
	var selectedRecord MiniGameRecord
	var redeemedUnitID string
	var redeemedUnit string
	var redeemedTarget string
	var grantedRewards []Reward
	var applyResult RewardApplyResult
	var pendingGarrison *CreateGarrisonDetachmentRequest
	state, _, err := s.repo.UpdateMiniGamePlayerState(playerID, redeemedAt, func(state *GameState, records []MiniGameRecord) ([]MiniGameRecord, error) {
		for i := range records {
			record := records[i]
			if record.ID != recordID {
				continue
			}
			if !miniGameRecordHasRedeemableReward(record.GameType) || record.RewardUnit == "" || record.RewardAmount <= 0 {
				return nil, ErrInvalidMiniGame
			}
			if record.RemainingAmount <= 0 || amount > record.RemainingAmount {
				return nil, ErrMiniGameStockShort
			}
			unitID, unitCfg, ok := FindFactionUnitByName(state.Player.Faction, record.RewardUnit)
			record.RemainingAmount -= amount
			records[i] = record
			state.ServerTime = redeemedAt.UTC().Format(resourceDateLayout)
			selectedRecord = record
			if ok {
				reward := Reward{Type: RewardTypeUnit, ID: unitID, Amount: amount}
				effectResult, err := ExecuteEffectsOnState(state, rewardsToEffects("minigame", []Reward{reward}), EffectContext{
					PlayerID: playerID,
					RefType:  LedgerRefMiniGameRedeem,
					RefID:    record.ID,
					Reason:   "minigame_redeem",
					Source:   "minigame",
				}, redeemedAt)
				if err != nil {
					return nil, err
				}
				redeemedUnitID = unitID
				redeemedUnit = unitCfg.Name
				redeemedTarget = "army"
				grantedRewards = []Reward{reward}
				applyResult = effectResult.Reward
				return records, nil
			}
			sourceFaction, crossUnitID, crossUnitCfg, exists := FindAnyFactionUnitByName(record.RewardUnit)
			if !exists {
				return nil, ErrUnitNotFound
			}
			redeemedUnitID = crossUnitID
			redeemedUnit = crossUnitCfg.Name
			redeemedTarget = "garrison"
			pendingGarrison = &CreateGarrisonDetachmentRequest{
				OwnerPlayerID: state.Player.ID,
				HostPlayerID:  state.Player.ID,
				SourceType:    GarrisonSourceEventReward,
				SourceID:      record.ID,
				SourceFaction: sourceFaction,
				Troops:        map[string]int{crossUnitID: amount},
				Metadata: map[string]any{
					"gameType":   record.GameType,
					"resultName": record.ResultName,
					"rewardUnit": record.RewardUnit,
				},
			}
			return records, nil
		}
		return nil, ErrMiniGameNotFound
	})
	if err != nil {
		return MiniGameRedeemResult{}, err
	}
	s.flushRewardSideEffects(applyResult)
	var garrison *Reinforcement
	if pendingGarrison != nil {
		garrisonResult, err := s.CreateGarrisonDetachment(*pendingGarrison)
		if err != nil {
			return MiniGameRedeemResult{}, err
		}
		state.Army = garrisonResult.Patch.Army
		state.Generals = garrisonResult.Patch.Generals
		state.GeneralAssignments = garrisonResult.Patch.GeneralAssignments
		state.ServerTime = garrisonResult.Patch.ServerTime
		record := garrisonResult.Reinforcement
		garrison = &record
	}
	result := MiniGameRedeemResult{
		Record:         selectedRecord,
		Army:           state.Army,
		ServerTime:     state.ServerTime,
		RedeemedUnitID: redeemedUnitID,
		RedeemedUnit:   redeemedUnit,
		RedeemedAmount: amount,
		RedeemedTarget: redeemedTarget,
		Garrison:       garrison,
		GrantedRewards: grantedRewards,
	}
	s.publishMiniGameRedeemEvents(playerID, result.Record.GameType, result.GrantedRewards, result.Record.ID, result.RedeemedAmount, state)
	return result, nil
}

func (s *Service) RedeemAllFactionMiniGameRewards(playerID string, gameType string) (MiniGameRedeemAllResult, error) {
	playerID = strings.TrimSpace(playerID)
	gameType = strings.TrimSpace(gameType)
	if playerID == "" {
		return MiniGameRedeemAllResult{}, ErrPlayerNotFound
	}
	if gameType == "" {
		gameType = "fishing"
	}
	redeemedAt := time.Now()
	redeemedUnits := map[string]int{}
	garrisonedUnits := map[string]int{}
	skippedUnits := map[string]int{}
	grantedRewards := []Reward{}
	redeemedRecords := 0
	garrisonRecords := 0
	skippedRecords := 0
	var applyResult RewardApplyResult
	pendingGarrisons := []CreateGarrisonDetachmentRequest{}
	state, _, err := s.repo.UpdateMiniGamePlayerState(playerID, redeemedAt, func(state *GameState, records []MiniGameRecord) ([]MiniGameRecord, error) {
		applyResult = RewardApplyResult{Granted: map[string]int{}}
		for i := range records {
			record := records[i]
			if record.GameType != gameType || record.RewardUnit == "" || record.RemainingAmount <= 0 {
				continue
			}
			unitID, unitCfg, ok := FindFactionUnitByName(state.Player.Faction, record.RewardUnit)
			amount := record.RemainingAmount
			records[i].RemainingAmount = 0
			if ok {
				reward := Reward{Type: RewardTypeUnit, ID: unitID, Amount: amount}
				effectResult, err := ExecuteEffectsOnState(state, rewardsToEffects("minigame", []Reward{reward}), EffectContext{
					PlayerID: playerID,
					RefType:  LedgerRefMiniGameRedeem,
					RefID:    record.ID,
					Reason:   "minigame_redeem",
					Source:   "minigame",
				}, redeemedAt)
				if err != nil {
					return nil, err
				}
				mergeRewardApplyResult(&applyResult, effectResult.Reward)
				grantedRewards = append(grantedRewards, reward)
				redeemedUnits[unitCfg.Name] += amount
				redeemedRecords++
				continue
			}
			sourceFaction, crossUnitID, crossUnitCfg, exists := FindAnyFactionUnitByName(record.RewardUnit)
			if !exists {
				records[i].RemainingAmount = amount
				skippedUnits[record.RewardUnit] += record.RemainingAmount
				skippedRecords++
				continue
			}
			garrisonedUnits[crossUnitCfg.Name] += amount
			redeemedRecords++
			garrisonRecords++
			pendingGarrisons = append(pendingGarrisons, CreateGarrisonDetachmentRequest{
				OwnerPlayerID: state.Player.ID,
				HostPlayerID:  state.Player.ID,
				SourceType:    GarrisonSourceEventReward,
				SourceID:      record.ID,
				SourceFaction: sourceFaction,
				Troops:        map[string]int{crossUnitID: amount},
				Metadata: map[string]any{
					"gameType":   record.GameType,
					"resultName": record.ResultName,
					"rewardUnit": record.RewardUnit,
				},
			})
		}
		state.ServerTime = redeemedAt.UTC().Format(resourceDateLayout)
		return records, nil
	})
	if err != nil {
		return MiniGameRedeemAllResult{}, err
	}
	s.flushRewardSideEffects(applyResult)
	for _, pending := range pendingGarrisons {
		garrisonResult, err := s.CreateGarrisonDetachment(pending)
		if err != nil {
			return MiniGameRedeemAllResult{}, err
		}
		state.Army = garrisonResult.Patch.Army
		state.Generals = garrisonResult.Patch.Generals
		state.GeneralAssignments = garrisonResult.Patch.GeneralAssignments
		state.ServerTime = garrisonResult.Patch.ServerTime
	}
	total := 0
	for _, amount := range redeemedUnits {
		total += amount
	}
	for _, amount := range garrisonedUnits {
		total += amount
	}
	result := MiniGameRedeemAllResult{
		Army:            state.Army,
		ServerTime:      state.ServerTime,
		RedeemedUnits:   redeemedUnits,
		RedeemedAmount:  total,
		RedeemedRecords: redeemedRecords,
		GarrisonedUnits: garrisonedUnits,
		GarrisonRecords: garrisonRecords,
		SkippedUnits:    skippedUnits,
		SkippedRecords:  skippedRecords,
		GrantedRewards:  grantedRewards,
	}
	s.publishMiniGameRedeemEvents(playerID, gameType, result.GrantedRewards, "", result.RedeemedAmount, state)
	return result, nil
}

// GetMiniGameRecords GM 查询某玩家的小游戏记录（含汇总）
func (s *Service) GetMiniGameRecords(playerID string, gameType string, limit int, offset int) (MiniGameSummary, error) {
	playerID = strings.TrimSpace(playerID)
	gameType = strings.TrimSpace(gameType)
	if playerID == "" {
		return MiniGameSummary{}, ErrPlayerNotFound
	}

	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}

	records, total, err := s.repo.ListMiniGameRecords(playerID, gameType, limit, offset)
	if err != nil {
		return MiniGameSummary{}, err
	}
	if records == nil {
		records = []MiniGameRecord{}
	}

	// 汇总各兵种可兑换总量
	totals := map[string]int{}
	for _, r := range records {
		if r.RewardUnit != "" && r.RemainingAmount > 0 {
			totals[r.RewardUnit] += r.RemainingAmount
		}
	}

	return MiniGameSummary{
		TotalRecords: total,
		Limit:        limit,
		Offset:       offset,
		HasMore:      offset+len(records) < total,
		Records:      records,
		RewardTotals: totals,
	}, nil
}

func (s *Service) publishMiniGameRedeemEvents(playerID string, gameType string, rewards []Reward, recordID string, redeemedAmount int, state GameState) {
	createdAt := state.ServerTime
	if strings.TrimSpace(createdAt) == "" {
		createdAt = time.Now().UTC().Format(resourceDateLayout)
	}
	s.publishEvent(GameEvent{
		Type:     EventMiniGameRedeemed,
		PlayerID: playerID,
		RefType:  LedgerRefMiniGameRedeem,
		RefID:    recordID,
		Payload: map[string]any{
			"gameType":        gameType,
			"redeemedAmount":  redeemedAmount,
			"redeemedRewards": rewards,
		},
		CreatedAt: createdAt,
	})
}
