// 本文件验证带胜负、目标和被动条件的防守随机特性合法未命中边界。
package game

import (
	"math"
	"testing"

	"hero3/internal/core/general"
)

// assertDefenderMissPairReports 核对防守随机特性和同将领第二特性只保留在拥有快照中。
func assertDefenderMissPairReports(t *testing.T, repo *MemoryRepository, battle PvpBattle, attackerPlayerID string, defenderPlayerID string, generalID string, specialTraitID string, bonusTraitID string, attackerUnitType string, attackerBefore int, attackerLost int, defenderUnitType string, defenderBefore int, defenderLost int) []BattleReport {
	t.Helper()
	assertRandomMissDefenderPvpReports(t, repo, battle, attackerPlayerID, defenderPlayerID, generalID, specialTraitID)
	attackerReports, _, attackerErr := repo.ListReports(attackerPlayerID, 10, 0)
	defenderReports, _, defenderErr := repo.ListReports(defenderPlayerID, 10, 0)
	if attackerErr != nil || defenderErr != nil || len(attackerReports) != 1 || len(defenderReports) != 1 {
		t.Fatalf("expected one report per side, attacker=%+v/%v defender=%+v/%v", attackerReports, attackerErr, defenderReports, defenderErr)
	}
	reports := []BattleReport{attackerReports[0], defenderReports[0]}
	for _, report := range reports {
		if len(report.PvpDefenderGenerals) != 1 || !pvpSnapshotHasTrait(report.PvpDefenderGenerals[0], bonusTraitID) || !standardDetailGeneralHasTrait(report.Detail, specialTraitID) || !standardDetailGeneralHasTrait(report.Detail, bonusTraitID) {
			t.Fatalf("expected defender snapshots to retain both owned traits, report=%+v", report)
		}
		attackerUnit := attackDefenseCrossReportUnit(t, report.Detail.PrimarySide, attackerUnitType)
		defenderUnit := attackDefenseCrossReportUnit(t, *report.Detail.SecondarySide, defenderUnitType)
		if attackerUnit.AmountBefore != attackerBefore || attackerUnit.Lost != attackerLost || attackerUnit.Survived != attackerBefore-attackerLost || defenderUnit.AmountBefore != defenderBefore || defenderUnit.Lost != defenderLost || defenderUnit.Survived != defenderBefore-defenderLost {
			t.Fatalf("expected exact standard rows, report=%s rows=%+v/%+v", report.ID, attackerUnit, defenderUnit)
		}
	}
	return reports
}

// TestPvpDefenderDianweiLossMissDoesNotReturnTroops 验证典韦守城战败但护主死战未命中时不返兵。
func TestPvpDefenderDianweiLossMissDoesNotReturnTroops(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "opponent", Name: "对手"}}},
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "dianwei", Name: "典韦"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"opponent": {ID: "opponent", Name: "对手", Faction: "shu", Enabled: true},
		"dianwei": {
			ID: "dianwei", Name: "典韦", Faction: "wei", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "huzhu_sizhan", TraitType: general.TraitTypeSpecial, Enabled: true,
				Scope: "self_army", RequiredOutcome: "loss", Params: map[string]float64{"triggerChance": 0, "lossReductionRate": 0.15, "maxReturnCount": 10000},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "sizhandaodi", TraitType: general.TraitTypeBonus, Enabled: true,
				Scope: "self_army", TargetUnitType: "infantry", AllowedSides: []string{"attacker"}, Params: map[string]float64{"attackBonusRate": 0.35},
			},
		},
	}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "shu", "opponent", "wei", "dianwei")
	attacker.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}
	defender.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack, Troops: map[string]int{"shuInfantry": 1000}, GeneralIDs: []string{"opponent"}})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
	defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
	if battle.Result["winner"] != "attacker" || battle.Result["attackerPower"] != float64(10000) || battle.Result["defensePower"] != float64(1000) || attackerLosses["shuInfantry"] != 37 || defenderLosses["weiInfantry"] != 100 {
		t.Fatalf("expected exact defender-loss baseline without return, battle=%+v", battle)
	}
	reports := assertDefenderMissPairReports(t, repo, battle, attacker.Player.ID, defender.Player.ID, "dianwei", "huzhu_sizhan", "sizhandaodi", "shuInfantry", 1000, 37, "weiInfantry", 100, 100)
	storedMarch, _ := repo.GetPvpMarch(started.March.ID)
	storedAttacker, _ := repo.GetState(attacker.Player.ID)
	storedDefender, _ := repo.GetState(defender.Player.ID)
	if storedMarch.AttackTroops["shuInfantry"] != 963 || armySliceToMap(storedDefender.Army)["weiInfantry"] != 0 || pvpTestGeneralExp(storedAttacker, "opponent") != 100 || pvpTestGeneralExp(storedDefender, "dianwei") != 37 || reports[0].GeneralExpGained != 100 || reports[1].GeneralExpGained != 37 {
		t.Fatalf("expected authoritative troops and experience without defender return, march=%+v states=%+v/%+v reports=%+v", storedMarch, storedAttacker.Generals, storedDefender.Generals, reports)
	}
}

// TestPvpDefenderMachaoXiliangMissKeepsPassiveSnapshotOnly 验证马超守城时西凉未命中且被动武力不伪装成触发。
func TestPvpDefenderMachaoXiliangMissKeepsPassiveSnapshotOnly(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wu":  {Name: "吴国", Generals: []GeneralInfo{{ID: "opponent", Name: "对手"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "machao", Name: "马超"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"opponent": {ID: "opponent", Name: "对手", Faction: "wu", Enabled: true},
		"machao": {
			ID: "machao", Name: "马超", Faction: "shu", Enabled: true,
			SpecialTrait: GeneralTraitConfig{TraitID: "xiliang_tuji", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army", TargetUnitType: "cavalry", Params: map[string]float64{"triggerChance": 0, "effectRate": 0.12}},
			BonusTrait:   GeneralTraitConfig{TraitID: "tianshen_xiafan", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army", Params: map[string]float64{"forceBonus": 20}},
		},
	}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wu", "opponent", "shu", "machao")
	unitsMu.Lock()
	activeUnits["wu"]["wuCavalry"] = UnitConfig{Name: "吴测试骑兵", Category: "cavalry", Stats: map[string]int{"attack": 10, "infantryDefense": 10, "cavalryDefense": 10, "carryCapacity": 5, "upkeep": 1}}
	unitsMu.Unlock()
	attacker.Army = []ArmyUnit{{UnitType: "wuInfantry", Amount: 1000}, {UnitType: "wuCavalry", Amount: 1000}}
	defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 100}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack, Troops: map[string]int{"wuInfantry": 1000, "wuCavalry": 1000}, GeneralIDs: []string{"opponent"}})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
	defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
	if battle.Result["winner"] != "attacker" || battle.Result["attackerPower"] != float64(20000) || battle.Result["defensePower"] != float64(900) || attackerLosses["wuInfantry"] != 12 || attackerLosses["wuCavalry"] != 12 || defenderLosses["shuInfantry"] != 100 {
		t.Fatalf("expected passive attack buff not to alter defender power and missed Xiliang not to add cavalry losses, battle=%+v", battle)
	}
	reports := assertDefenderMissPairReports(t, repo, battle, attacker.Player.ID, defender.Player.ID, "machao", "xiliang_tuji", "tianshen_xiafan", "wuInfantry", 1000, 12, "shuInfantry", 100, 100)
	for _, report := range reports {
		snapshot := report.PvpDefenderGenerals[0]
		if snapshot.EffectiveStats["force"]-snapshot.Stats["force"] != 20 || math.Abs(snapshot.Buffs[StatAttackBonus]-0.4) > 1e-9 {
			t.Fatalf("expected defender Machao snapshot to retain passive force and attack buff, report=%s snapshot=%+v", report.ID, snapshot)
		}
		cavalry := attackDefenseCrossReportUnit(t, report.Detail.PrimarySide, "wuCavalry")
		if cavalry.AmountBefore != 1000 || cavalry.Lost != 12 || cavalry.Survived != 988 {
			t.Fatalf("expected cavalry to keep core-only losses, report=%s row=%+v", report.ID, cavalry)
		}
	}
	storedMarch, _ := repo.GetPvpMarch(started.March.ID)
	storedAttacker, _ := repo.GetState(attacker.Player.ID)
	storedDefender, _ := repo.GetState(defender.Player.ID)
	if storedMarch.AttackTroops["wuInfantry"] != 988 || storedMarch.AttackTroops["wuCavalry"] != 988 || armySliceToMap(storedDefender.Army)["shuInfantry"] != 0 || pvpTestGeneralExp(storedAttacker, "opponent") != 100 || pvpTestGeneralExp(storedDefender, "machao") != 24 || reports[0].GeneralExpGained != 100 || reports[1].GeneralExpGained != 24 {
		t.Fatalf("expected authoritative mixed troops and experience, march=%+v states=%+v/%+v reports=%+v", storedMarch, storedAttacker.Generals, storedDefender.Generals, reports)
	}
}

// TestPvpDefenderSunceVictoryMissDoesNotPursue 验证孙策掠夺守城获胜但追击未命中时不追加来袭方伤亡。
func TestPvpDefenderSunceVictoryMissDoesNotPursue(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "opponent", Name: "对手"}}},
		"wu":  {Name: "吴国", Generals: []GeneralInfo{{ID: "sunce", Name: "孙策"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"opponent": {ID: "opponent", Name: "对手", Faction: "wei", Enabled: true},
		"sunce": {
			ID: "sunce", Name: "孙策", Faction: "wu", Enabled: true,
			SpecialTrait: GeneralTraitConfig{TraitID: "xiaobawang_zhuiji", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army", AllowedScenes: []string{"plunder"}, RequiredOutcome: "win", Params: map[string]float64{"triggerChance": 0, "effectRate": 0.1}},
			BonusTrait:   GeneralTraitConfig{TraitID: "xiaobawang_tieqi", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army", TargetUnitType: "overlordRider", AllowedSides: []string{"attacker"}, Params: map[string]float64{"unitAttackFlat": 50}},
		},
	}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wei", "opponent", "wu", "sunce")
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	defender.Army = []ArmyUnit{{UnitType: "wuInfantry", Amount: 200}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypePlunder, Troops: map[string]int{"weiInfantry": 100}, GeneralIDs: []string{"opponent"}})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
	defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
	if battle.Result["winner"] != "defender" || battle.Result["attackerPower"] != float64(1000) || battle.Result["defensePower"] != float64(2000) || attackerLosses["weiInfantry"] != 72 || defenderLosses["wuInfantry"] != 54 {
		t.Fatalf("expected defender victory with core-only losses when pursuit misses, battle=%+v", battle)
	}
	reports := assertDefenderMissPairReports(t, repo, battle, attacker.Player.ID, defender.Player.ID, "sunce", "xiaobawang_zhuiji", "xiaobawang_tieqi", "weiInfantry", 100, 72, "wuInfantry", 200, 54)
	storedMarch, _ := repo.GetPvpMarch(started.March.ID)
	storedAttacker, _ := repo.GetState(attacker.Player.ID)
	storedDefender, _ := repo.GetState(defender.Player.ID)
	if storedMarch.AttackTroops["weiInfantry"] != 28 || armySliceToMap(storedDefender.Army)["wuInfantry"] != 146 || pvpTestGeneralExp(storedAttacker, "opponent") != 54 || pvpTestGeneralExp(storedDefender, "sunce") != 72 || reports[0].GeneralExpGained != 54 || reports[1].GeneralExpGained != 72 {
		t.Fatalf("expected authoritative core-only troops and experience, march=%+v states=%+v/%+v reports=%+v", storedMarch, storedAttacker.Generals, storedDefender.Generals, reports)
	}
}
