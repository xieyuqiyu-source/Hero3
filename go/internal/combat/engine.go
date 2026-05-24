package combat

import (
	"math"
)

// Resolve 执行战斗计算，返回战斗结果
func Resolve(input CombatInput) CombatResult {
	rule, ok := GetRule(input.RuleID)
	if !ok {
		// fallback 到默认攻击规则
		rule = defaultCombatConfig().Rules["official_attack"]
	}

	// 计算进攻方攻击力（按步/骑分类）
	a1, a2 := calculateAttackPools(input.Attacker)
	a := a1 + a2

	// 进攻方无攻击力时，防守方直接胜（防止 0 攻击兵种换掉守军）
	if a <= 0 {
		return CombatResult{
			Winner:           "defender",
			Mode:             rule.Mode,
			AttackerLosses:   distributeLosses(input.Attacker.Units, 1.0),
			DefenderLosses:   distributeLosses(input.Defender.Units, 0),
			AttackerLossRate: 1.0,
			DefenderLossRate: 0,
			AttackPower:      0,
			DefensePower:     0,
			SurvivingCarry:   0,
		}
	}

	// 计算防守方有效防御
	b := calculateWeightedDefense(input.Defender, a1, a2)

	// 城墙加成（仅 PvP，wallLevel > 0 时生效）
	if input.WallLevel > 0 && input.WallFaction != "" {
		wallMult := calculateWallMultiplier(input.WallFaction, input.WallLevel)
		b *= wallMult
	}

	// 判定胜负
	winner := determineWinner(a, b, rule)

	// 计算损失比例
	attackerLossRate, defenderLossRate := calculateLossRates(a, b, winner, rule)

	// 分配损失到各兵种
	attackerLosses := distributeLosses(input.Attacker.Units, attackerLossRate)
	defenderLosses := distributeLosses(input.Defender.Units, defenderLossRate)

	// 计算存活部队运载量
	survivingCarry := calculateSurvivingCarry(input.Attacker.Units, attackerLosses)

	return CombatResult{
		Winner:           winner,
		Mode:             rule.Mode,
		AttackerLosses:   attackerLosses,
		DefenderLosses:   defenderLosses,
		AttackerLossRate: attackerLossRate,
		DefenderLossRate: defenderLossRate,
		AttackPower:      a,
		DefensePower:     b,
		SurvivingCarry:   survivingCarry,
	}
}

// calculateAttackPools 计算步兵攻击(A1)和骑兵攻击(A2)
func calculateAttackPools(army Army) (float64, float64) {
	var a1, a2 float64
	for _, unit := range army.Units {
		attackTotal := float64(unit.Attack) * float64(unit.Count)
		if unit.Category == "cavalry" {
			a2 += attackTotal
		} else {
			// infantry, siege, special 都算步兵类
			a1 += attackTotal
		}
	}
	return a1, a2
}

// calculateWeightedDefense 按进攻方步/骑比例加权计算有效防御
// b = (A1×D1 + A2×D2) / (A1+A2)
func calculateWeightedDefense(defender Army, a1 float64, a2 float64) float64 {
	totalAttack := a1 + a2
	if totalAttack <= 0 {
		return 0
	}

	var d1, d2 float64
	for _, unit := range defender.Units {
		count := float64(unit.Count)
		d1 += float64(unit.InfantryDefense) * count
		d2 += float64(unit.CavalryDefense) * count
	}

	return (a1*d1 + a2*d2) / totalAttack
}

// calculateWallMultiplier 计算城墙加成系数
// B = b × base^level
func calculateWallMultiplier(faction string, level int) float64 {
	cfg := GetCombatConfig()
	entry, ok := cfg.WallConfig[faction]
	if !ok {
		return 1.0
	}
	return math.Pow(entry.Base, float64(level))
}

// determineWinner 判定胜负
func determineWinner(a float64, b float64, rule RuleConfig) string {
	if a > b {
		return "attacker"
	}
	if a < b {
		return "defender"
	}
	// a == b
	if rule.EqualResult == "defender_wins" {
		return "defender"
	}
	return "draw"
}

// calculateLossRates 计算双方损失比例
func calculateLossRates(a float64, b float64, winner string, rule RuleConfig) (float64, float64) {
	exp := rule.Exponent
	if exp <= 0 {
		exp = 1.422
	}

	switch winner {
	case "attacker":
		ratio := math.Pow(b/a, exp)
		if rule.Mode == "plunder" {
			return ratio / (1 + ratio), 1.0 / (1 + ratio)
		}
		// 攻击模式：败方全灭
		return ratio, 1.0

	case "defender":
		ratio := math.Pow(a/b, exp)
		if rule.Mode == "plunder" {
			return 1.0 / (1 + ratio), ratio / (1 + ratio)
		}
		// 攻击模式：败方全灭
		return 1.0, ratio

	default: // draw
		if rule.Mode == "plunder" {
			return 0.5, 0.5
		}
		// 攻击模式：同归于尽
		return 1.0, 1.0
	}
}

// distributeLosses 按比例分配损失到各兵种
func distributeLosses(units []Unit, lossRate float64) []UnitLoss {
	if lossRate <= 0 {
		result := make([]UnitLoss, len(units))
		for i, u := range units {
			result[i] = UnitLoss{ID: u.ID, Count: u.Count, Losses: 0}
		}
		return result
	}

	if lossRate >= 1.0 {
		result := make([]UnitLoss, len(units))
		for i, u := range units {
			result[i] = UnitLoss{ID: u.ID, Count: u.Count, Losses: u.Count}
		}
		return result
	}

	// 按比例分配，向下取整
	result := make([]UnitLoss, len(units))
	totalTarget := 0
	totalActual := 0

	for i, u := range units {
		target := int(math.Floor(float64(u.Count) * lossRate))
		result[i] = UnitLoss{ID: u.ID, Count: u.Count, Losses: target}
		totalTarget += u.Count
		totalActual += target
	}

	// 补余数：总损失应为 floor(总兵力 × lossRate)
	expectedTotal := int(math.Floor(float64(totalTarget) * lossRate))
	remainder := expectedTotal - totalActual

	// 按小数部分从大到小补 1
	if remainder > 0 {
		type fracEntry struct {
			index    int
			fraction float64
		}
		fracs := make([]fracEntry, len(units))
		for i, u := range units {
			exact := float64(u.Count) * lossRate
			fracs[i] = fracEntry{i, exact - math.Floor(exact)}
		}
		// 简单排序（数量少，冒泡即可）
		for i := 0; i < len(fracs)-1; i++ {
			for j := i + 1; j < len(fracs); j++ {
				if fracs[j].fraction > fracs[i].fraction {
					fracs[i], fracs[j] = fracs[j], fracs[i]
				}
			}
		}
		for _, f := range fracs {
			if remainder <= 0 {
				break
			}
			if result[f.index].Losses < units[f.index].Count {
				result[f.index].Losses++
				remainder--
			}
		}
	}

	return result
}

// calculateSurvivingCarry 计算存活部队的总运载量
// TODO: 增援机制实现后，Units 可能有重复 ID，需要先聚合再计算
func calculateSurvivingCarry(units []Unit, losses []UnitLoss) int {
	total := 0
	lossMap := make(map[string]int, len(losses))
	for _, l := range losses {
		lossMap[l.ID] = l.Losses
	}

	for _, u := range units {
		survived := u.Count - lossMap[u.ID]
		if survived > 0 {
			total += survived * u.CarryCapacity
		}
	}
	return total
}
