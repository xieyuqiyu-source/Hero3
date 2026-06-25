package game

// NpcTraitBuffs 汇总后的 NPC 词条加成
type NpcTraitBuffs struct {
	ProductionBonus      float64 // 资源产量加成（如 0.3 = +30%）
	AttackBonus          float64 // 全体攻击加成
	InfantryDefenseBonus float64 // 步兵防御加成
	CavalryDefenseBonus  float64 // 骑兵防御加成
	AllDefenseBonus      float64 // 全体防御加成
	ArmyRecoveryBonus    float64 // 守军恢复速度加成
	ArmyCapBonus         float64 // 守军上限加成
}

// collectTraitBuffs 从 NPC 城池的词条列表中汇总所有 buff（多词条叠加）
func collectTraitBuffs(city *NpcCity) NpcTraitBuffs {
	var buffs NpcTraitBuffs
	for _, trait := range city.Traits {
		for key, value := range trait.Buffs {
			switch key {
			case "productionBonus":
				buffs.ProductionBonus += value
			case "attackBonus", "armyAttackBonus":
				buffs.AttackBonus += value
			case "infantryDefenseBonus":
				buffs.InfantryDefenseBonus += value
			case "cavalryDefenseBonus":
				buffs.CavalryDefenseBonus += value
			case "allDefenseBonus":
				buffs.AllDefenseBonus += value
			case "armyRecoveryBonus":
				buffs.ArmyRecoveryBonus += value
			case "armyCapBonus":
				buffs.ArmyCapBonus += value
			}
		}
	}
	return buffs
}

// applyAttackBuff 应用攻击加成
func (b NpcTraitBuffs) applyAttack(base int) int {
	if b.AttackBonus == 0 {
		return base
	}
	return int(float64(base) * (1 + b.AttackBonus))
}

// applyInfantryDefense 应用步兵防御加成（含全体防御）
func (b NpcTraitBuffs) applyInfantryDefense(base int) int {
	bonus := b.InfantryDefenseBonus + b.AllDefenseBonus
	if bonus == 0 {
		return base
	}
	return int(float64(base) * (1 + bonus))
}

// applyCavalryDefense 应用骑兵防御加成（含全体防御）
func (b NpcTraitBuffs) applyCavalryDefense(base int) int {
	bonus := b.CavalryDefenseBonus + b.AllDefenseBonus
	if bonus == 0 {
		return base
	}
	return int(float64(base) * (1 + bonus))
}

// applyProductionRate 应用产量加成
func (b NpcTraitBuffs) applyProductionRate(base int) int {
	if b.ProductionBonus == 0 {
		return base
	}
	return int(float64(base) * (1 + b.ProductionBonus))
}

// applyArmyRecovery 应用守军恢复速度加成
func (b NpcTraitBuffs) applyArmyRecovery(base float64) float64 {
	if b.ArmyRecoveryBonus == 0 {
		return base
	}
	return base * (1 + b.ArmyRecoveryBonus)
}

// applyArmyCap 应用守军上限加成
func (b NpcTraitBuffs) applyArmyCap(base int) int {
	if b.ArmyCapBonus == 0 {
		return base
	}
	return int(float64(base) * (1 + b.ArmyCapBonus))
}
