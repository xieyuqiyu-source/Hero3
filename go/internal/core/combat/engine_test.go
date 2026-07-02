package combat

import (
	"math"
	"testing"
)

func TestAttackerVictoryAttackMode(t *testing.T) {
	// 200 青州军(步兵 atk=8) vs 50 青州军(步防=7, 骑防=10)
	input := CombatInput{
		RuleID: "official_attack",
		Attacker: Army{
			Faction: "wei",
			Units: []Unit{
				{ID: "qingZhouArmy", Category: "infantry", Count: 200, Attack: 8, InfantryDefense: 7, CavalryDefense: 10, CarryCapacity: 80},
			},
		},
		Defender: Army{
			Faction: "shu",
			Units: []Unit{
				{ID: "qingZhouArmy", Category: "infantry", Count: 50, Attack: 8, InfantryDefense: 7, CavalryDefense: 10, CarryCapacity: 80},
			},
		},
	}

	result := Resolve(input)

	if result.Winner != "attacker" {
		t.Fatalf("expected attacker victory, got %s", result.Winner)
	}

	// a = 200*8 = 1600, b = (1600*350)/1600 = 350
	expectedA := 1600.0
	expectedB := 350.0
	if math.Abs(result.AttackPower-expectedA) > 0.1 {
		t.Errorf("expected a=%.0f, got %.0f", expectedA, result.AttackPower)
	}
	if math.Abs(result.DefensePower-expectedB) > 0.1 {
		t.Errorf("expected b=%.0f, got %.0f", expectedB, result.DefensePower)
	}

	// 进攻方损失 = (350/1600)^1.422 ≈ 0.112
	if result.AttackerLossRate > 0.15 || result.AttackerLossRate < 0.08 {
		t.Errorf("attacker loss rate out of range: %.4f", result.AttackerLossRate)
	}

	// 防守方全灭
	if result.DefenderLossRate != 1.0 {
		t.Errorf("expected defender full wipe, got %.4f", result.DefenderLossRate)
	}

	// 防守方损失 = 50
	totalDefLoss := 0
	for _, l := range result.DefenderLosses {
		totalDefLoss += l.Losses
	}
	if totalDefLoss != 50 {
		t.Errorf("expected 50 defender losses, got %d", totalDefLoss)
	}
}

func TestDefenderVictoryAttackMode(t *testing.T) {
	// 30 青州军 vs 100 青州军 → 防守方胜
	input := CombatInput{
		RuleID: "official_attack",
		Attacker: Army{
			Faction: "wei",
			Units: []Unit{
				{ID: "qingZhouArmy", Category: "infantry", Count: 30, Attack: 8, InfantryDefense: 7, CavalryDefense: 10, CarryCapacity: 80},
			},
		},
		Defender: Army{
			Faction: "wei",
			Units: []Unit{
				{ID: "qingZhouArmy", Category: "infantry", Count: 100, Attack: 8, InfantryDefense: 7, CavalryDefense: 10, CarryCapacity: 80},
			},
		},
	}

	result := Resolve(input)

	if result.Winner != "defender" {
		t.Fatalf("expected defender victory, got %s", result.Winner)
	}

	// 进攻方全灭
	if result.AttackerLossRate != 1.0 {
		t.Errorf("expected attacker full wipe, got %.4f", result.AttackerLossRate)
	}

	// 防守方有损失但不全灭
	if result.DefenderLossRate >= 1.0 || result.DefenderLossRate <= 0 {
		t.Errorf("defender loss rate should be between 0 and 1, got %.4f", result.DefenderLossRate)
	}
}

func TestNoLossPowerRatioThreshold(t *testing.T) {
	original := GetCombatConfig()
	t.Cleanup(func() {
		if err := SaveCombatConfig("", original); err != nil {
			t.Fatalf("restore combat config: %v", err)
		}
	})
	config := defaultCombatConfig()
	config.Rules["test_no_loss"] = RuleConfig{
		ID:                        "test_no_loss",
		Name:                      "无损阈值测试",
		Mode:                      "attack",
		Exponent:                  1.422,
		NoLossPowerRatioThreshold: 0.001,
		EqualResult:               "mutual_destruction",
		LossDistribution:          "proportional",
		DefenseFormula:            "weighted",
	}
	if err := SaveCombatConfig("", config); err != nil {
		t.Fatalf("save combat config: %v", err)
	}

	input := CombatInput{
		RuleID: "test_no_loss",
		Attacker: Army{
			Faction: "wei",
			Units: []Unit{
				{ID: "huBaoQi", Category: "cavalry", Count: 50000, Attack: 36, InfantryDefense: 16, CavalryDefense: 21, CarryCapacity: 140},
			},
		},
		Defender: Army{
			Faction: "shu",
			Units: []Unit{
				{ID: "huge_defender", Category: "infantry", Count: 2000000000, Attack: 1, InfantryDefense: 1, CavalryDefense: 1, CarryCapacity: 1},
			},
		},
	}

	result := Resolve(input)

	if result.Winner != "defender" {
		t.Fatalf("expected defender victory, got %s", result.Winner)
	}
	if result.DefenderLossRate != 0 {
		t.Fatalf("expected defender no loss, got loss rate %.12f", result.DefenderLossRate)
	}
	for _, loss := range result.DefenderLosses {
		if loss.Losses != 0 {
			t.Fatalf("expected no defender losses, got %+v", result.DefenderLosses)
		}
	}
}

func TestNoLossPowerRatioThresholdCanBeTightened(t *testing.T) {
	original := GetCombatConfig()
	t.Cleanup(func() {
		if err := SaveCombatConfig("", original); err != nil {
			t.Fatalf("restore combat config: %v", err)
		}
	})
	config := defaultCombatConfig()
	config.Rules["test_tight_no_loss"] = RuleConfig{
		ID:                        "test_tight_no_loss",
		Name:                      "严格无损阈值测试",
		Mode:                      "attack",
		Exponent:                  1.422,
		NoLossPowerRatioThreshold: 0.0001,
		EqualResult:               "mutual_destruction",
		LossDistribution:          "proportional",
		DefenseFormula:            "weighted",
	}
	if err := SaveCombatConfig("", config); err != nil {
		t.Fatalf("save combat config: %v", err)
	}

	input := CombatInput{
		RuleID: "test_tight_no_loss",
		Attacker: Army{
			Faction: "wei",
			Units: []Unit{
				{ID: "attacker", Category: "infantry", Count: 1000, Attack: 1, InfantryDefense: 1, CavalryDefense: 1, CarryCapacity: 1},
			},
		},
		Defender: Army{
			Faction: "shu",
			Units: []Unit{
				{ID: "defender", Category: "infantry", Count: 1000000, Attack: 1, InfantryDefense: 1, CavalryDefense: 1, CarryCapacity: 1},
			},
		},
	}

	result := Resolve(input)

	if result.Winner != "defender" {
		t.Fatalf("expected defender victory, got %s", result.Winner)
	}
	if result.DefenderLossRate <= 0 {
		t.Fatalf("expected defender loss when threshold is tightened, got %.12f", result.DefenderLossRate)
	}
	totalDefenderLosses := 0
	for _, loss := range result.DefenderLosses {
		totalDefenderLosses += loss.Losses
	}
	if totalDefenderLosses == 0 {
		t.Fatalf("expected actual defender losses when threshold is tightened, got %+v", result.DefenderLosses)
	}
}

func TestPlunderMode(t *testing.T) {
	// 100 青州军 掠夺 80 青州军
	input := CombatInput{
		RuleID: "official_plunder",
		Attacker: Army{
			Faction: "wei",
			Units: []Unit{
				{ID: "qingZhouArmy", Category: "infantry", Count: 100, Attack: 8, InfantryDefense: 7, CavalryDefense: 10, CarryCapacity: 80},
			},
		},
		Defender: Army{
			Faction: "wei",
			Units: []Unit{
				{ID: "qingZhouArmy", Category: "infantry", Count: 80, Attack: 8, InfantryDefense: 7, CavalryDefense: 10, CarryCapacity: 80},
			},
		},
	}

	result := Resolve(input)

	if result.Winner != "attacker" {
		t.Fatalf("expected attacker victory, got %s", result.Winner)
	}

	// 掠夺模式：双方损失之和 ≈ 100%
	sum := result.AttackerLossRate + result.DefenderLossRate
	if math.Abs(sum-1.0) > 0.001 {
		t.Errorf("plunder loss rates should sum to 1.0, got %.4f + %.4f = %.4f",
			result.AttackerLossRate, result.DefenderLossRate, sum)
	}

	// 双方都不全灭
	if result.AttackerLossRate >= 1.0 {
		t.Error("attacker should not be fully wiped in plunder mode")
	}
	if result.DefenderLossRate >= 1.0 {
		t.Error("defender should not be fully wiped in plunder mode")
	}
}

func TestEqualForcesAttackMode(t *testing.T) {
	// 相同兵力 → 同归于尽
	input := CombatInput{
		RuleID: "official_attack",
		Attacker: Army{
			Faction: "wei",
			Units: []Unit{
				{ID: "qingZhouArmy", Category: "infantry", Count: 100, Attack: 7, InfantryDefense: 7, CavalryDefense: 10, CarryCapacity: 80},
			},
		},
		Defender: Army{
			Faction: "wei",
			Units: []Unit{
				{ID: "qingZhouArmy", Category: "infantry", Count: 100, Attack: 7, InfantryDefense: 7, CavalryDefense: 10, CarryCapacity: 80},
			},
		},
	}

	result := Resolve(input)

	if result.Winner != "draw" {
		t.Fatalf("expected draw, got %s", result.Winner)
	}
	if result.AttackerLossRate != 1.0 || result.DefenderLossRate != 1.0 {
		t.Errorf("expected mutual destruction, got atk=%.2f def=%.2f",
			result.AttackerLossRate, result.DefenderLossRate)
	}
}

func TestMixedArmyWeightedDefense(t *testing.T) {
	// 80 步兵(atk=8) + 20 骑兵(atk=24) vs 100 步兵(步防=7, 骑防=10)
	input := CombatInput{
		RuleID: "official_attack",
		Attacker: Army{
			Faction: "wei",
			Units: []Unit{
				{ID: "infantry", Category: "infantry", Count: 80, Attack: 8, InfantryDefense: 7, CavalryDefense: 10, CarryCapacity: 80},
				{ID: "cavalry", Category: "cavalry", Count: 20, Attack: 24, InfantryDefense: 13, CavalryDefense: 10, CarryCapacity: 200},
			},
		},
		Defender: Army{
			Faction: "shu",
			Units: []Unit{
				{ID: "infantry", Category: "infantry", Count: 100, Attack: 8, InfantryDefense: 7, CavalryDefense: 10, CarryCapacity: 80},
			},
		},
	}

	result := Resolve(input)

	// A1=640, A2=480, a=1120
	// D1=700, D2=1000
	// b = (640*700 + 480*1000) / 1120 = 928000/1120 ≈ 828.6
	expectedB := 828.57
	if math.Abs(result.DefensePower-expectedB) > 1.0 {
		t.Errorf("expected b≈%.1f, got %.1f", expectedB, result.DefensePower)
	}

	if result.Winner != "attacker" {
		t.Fatalf("expected attacker victory, got %s", result.Winner)
	}

	// 混编骑兵让 b 变大（骑防高），损失应该比纯步兵高
	// 纯步兵时 b=700, 混编时 b≈828.6
	if result.AttackerLossRate < 0.5 {
		t.Errorf("mixed army should have higher loss rate due to high cavalry defense, got %.4f", result.AttackerLossRate)
	}
}

func TestWallBonus(t *testing.T) {
	// 有城墙 vs 无城墙，防御力应该更高
	baseInput := CombatInput{
		RuleID: "official_attack",
		Attacker: Army{
			Faction: "wei",
			Units: []Unit{
				{ID: "infantry", Category: "infantry", Count: 100, Attack: 8, InfantryDefense: 7, CavalryDefense: 10, CarryCapacity: 80},
			},
		},
		Defender: Army{
			Faction: "wei",
			Units: []Unit{
				{ID: "infantry", Category: "infantry", Count: 50, Attack: 8, InfantryDefense: 7, CavalryDefense: 10, CarryCapacity: 80},
			},
		},
	}

	noWall := Resolve(baseInput)

	baseInput.WallLevel = 20
	baseInput.WallFaction = "wei"
	withWall := Resolve(baseInput)

	// 魏国 Lv.20 城墙：1.03^20 ≈ 1.81
	if withWall.DefensePower <= noWall.DefensePower {
		t.Error("wall should increase defense power")
	}

	expectedRatio := math.Pow(1.03, 20)
	actualRatio := withWall.DefensePower / noWall.DefensePower
	if math.Abs(actualRatio-expectedRatio) > 0.01 {
		t.Errorf("wall multiplier should be ≈%.3f, got %.3f", expectedRatio, actualRatio)
	}
}

func TestNormalizeCombatConfigFillsWallHardnessDefaults(t *testing.T) {
	cfg := CombatConfig{
		Rules: map[string]RuleConfig{
			RuleOfficialAttack: {ID: RuleOfficialAttack, Mode: "attack"},
		},
		WallConfig: map[string]WallEntry{
			"wei": {Base: 1.03},
		},
	}

	normalizeCombatConfig(&cfg)

	wei := cfg.WallConfig["wei"]
	if wei.Hardness <= 0 || wei.MinDamagedLevelFrom20 != 5 || wei.MaxDamagedLevelFrom20 != 5 {
		t.Fatalf("expected wei wall hardness defaults, got %+v", wei)
	}
	shu := cfg.WallConfig["shu"]
	if shu.Hardness <= wei.Hardness || shu.MinDamagedLevelFrom20 != 16 || shu.MaxDamagedLevelFrom20 != 17 {
		t.Fatalf("expected shu wall hardness defaults, got %+v", shu)
	}
}

func TestSurvivingCarryCapacity(t *testing.T) {
	input := CombatInput{
		RuleID: "official_attack",
		Attacker: Army{
			Faction: "wei",
			Units: []Unit{
				{ID: "infantry", Category: "infantry", Count: 100, Attack: 8, InfantryDefense: 7, CavalryDefense: 10, CarryCapacity: 80},
			},
		},
		Defender: Army{
			Faction: "shu",
			Units: []Unit{
				{ID: "infantry", Category: "infantry", Count: 20, Attack: 8, InfantryDefense: 7, CavalryDefense: 10, CarryCapacity: 80},
			},
		},
	}

	result := Resolve(input)

	// 进攻方应该存活大部分
	totalSurvived := 0
	for _, l := range result.AttackerLosses {
		totalSurvived += l.Count - l.Losses
	}

	expectedCarry := totalSurvived * 80
	if result.SurvivingCarry != expectedCarry {
		t.Errorf("expected carry=%d, got %d", expectedCarry, result.SurvivingCarry)
	}
}
