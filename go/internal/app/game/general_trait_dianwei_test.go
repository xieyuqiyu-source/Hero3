// 本文件验证典韦当前双特性的配置迁移、方向、概率、战力兵损和权威战报。
package game

import (
	"reflect"
	"testing"

	"hero3/internal/core/combat"
	"hero3/internal/core/general"
)

// TestNormalizeDianweiCurrentAndLegacyTraits 验证旧返兵配置迁移，当前 GM 自定义值继续保留。
func TestNormalizeDianweiCurrentAndLegacyTraits(t *testing.T) {
	legacy := NormalizeGeneralsConfig(GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"dianwei": {
			ID: "dianwei",
			SpecialTrait: GeneralTraitConfig{
				TraitID: "huzhu_sizhan", Scope: "self_army", RequiredOutcome: "loss",
				Params: map[string]float64{"triggerChance": 0.35, "lossReductionRate": 0.15, "maxReturnCount": 10000},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "sizhandaodi", Scope: "self_army", TargetUnitType: "infantry", AllowedSides: []string{"attacker"},
				Params: map[string]float64{"attackBonusRate": 0.35},
			},
		},
	}})
	hero := legacy.Heroes["dianwei"]
	if hero.SpecialTrait.TraitID != "huzhu_xuezhan" || hero.SpecialTrait.TraitType != general.TraitTypeSpecial || hero.SpecialTrait.Scope != "self_army" || hero.SpecialTrait.TargetUnitType != "jinWeiSoldier" || !reflect.DeepEqual(hero.SpecialTrait.AllowedSides, []string{"defender", "reinforcement"}) || hero.SpecialTrait.RequiredOutcome != "" || !reflect.DeepEqual(hero.SpecialTrait.Params, map[string]float64{"triggerChance": 1, "generalDefenseFlat": 20}) {
		t.Fatalf("unexpected migrated Huzhu Xuezhan: %+v", hero.SpecialTrait)
	}
	if hero.BonusTrait.TraitID != "sizhandaodi" || hero.BonusTrait.TraitType != general.TraitTypeBonus || !reflect.DeepEqual(hero.BonusTrait.Params, map[string]float64{"triggerChance": 0.6, "attackBonusRate": 0.35}) {
		t.Fatalf("unexpected migrated Sizhan Daodi: %+v", hero.BonusTrait)
	}

	current := NormalizeGeneralsConfig(GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"dianwei": {
			ID:           "dianwei",
			SpecialTrait: GeneralTraitConfig{TraitID: "huzhu_xuezhan", Params: map[string]float64{"triggerChance": 0.8, "generalDefenseFlat": 25, "lossReductionRate": 0.5, "maxReturnCount": 99}},
			BonusTrait:   GeneralTraitConfig{TraitID: "sizhandaodi", Params: map[string]float64{"triggerChance": 0.75, "attackBonusRate": 0.4}},
		},
	}}).Heroes["dianwei"]
	if !reflect.DeepEqual(current.SpecialTrait.Params, map[string]float64{"triggerChance": 0.8, "generalDefenseFlat": 25}) || current.SpecialTrait.TargetUnitType != "jinWeiSoldier" || !reflect.DeepEqual(current.SpecialTrait.AllowedSides, []string{"defender", "reinforcement"}) {
		t.Fatalf("expected current Huzhu GM values preserved and old fields removed, trait=%+v", current.SpecialTrait)
	}
	if !reflect.DeepEqual(current.BonusTrait.Params, map[string]float64{"triggerChance": 0.75, "attackBonusRate": 0.4}) || current.BonusTrait.TargetUnitType != "infantry" || !reflect.DeepEqual(current.BonusTrait.AllowedSides, []string{"attacker"}) {
		t.Fatalf("expected current Sizhan GM values preserved, trait=%+v", current.BonusTrait)
	}
}

// TestDianweiTraitSchemasMatchCurrentDesign 验证 GM schema 使用当前名称、默认概率和数值。
func TestDianweiTraitSchemasMatchCurrentDesign(t *testing.T) {
	huzhu, ok := general.Get("huzhu_xuezhan")
	if !ok {
		t.Fatal("huzhu_xuezhan trait not registered")
	}
	if _, oldExists := general.Get("huzhu_sizhan"); oldExists {
		t.Fatal("obsolete huzhu_sizhan must not remain registered")
	}
	sizhan, ok := general.Get("sizhandaodi")
	if !ok {
		t.Fatal("sizhandaodi trait not registered")
	}
	huzhuFields := map[string]general.ParamField{}
	for _, field := range huzhu.ParamSchema() {
		huzhuFields[field.Key] = field
	}
	sizhanFields := map[string]general.ParamField{}
	for _, field := range sizhan.ParamSchema() {
		sizhanFields[field.Key] = field
	}
	if huzhu.Name() != "护主血战" || huzhuFields["triggerChance"].Default != 1 || huzhuFields["generalDefenseFlat"].Default != 20 {
		t.Fatalf("unexpected Huzhu schema: name=%s fields=%+v", huzhu.Name(), huzhuFields)
	}
	if sizhanFields["triggerChance"].Default != 0.6 || sizhanFields["attackBonusRate"].Default != 0.35 {
		t.Fatalf("unexpected Sizhan schema: %+v", sizhanFields)
	}
}

type dianweiPvpResult struct {
	battle         PvpBattle
	storedMarch    PvpMarch
	attackerReport BattleReport
	defenderReport BattleReport
	attackerState  GameState
	defenderState  GameState
}

// runDianweiAttackPvp 执行典韦主动进攻，并固定死战到底命中或未命中。
func runDianweiAttackPvp(t *testing.T, triggerChance float64) dianweiPvpResult {
	t.Helper()
	setTestCombatUnitsConfig(t)
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "dianwei", Name: "典韦"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"dianwei": {
			ID: "dianwei", Name: "典韦", Faction: "wei", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "huzhu_xuezhan", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "self_army", TargetUnitType: "jinWeiSoldier",
				AllowedSides: []string{"defender", "reinforcement"}, Params: map[string]float64{"triggerChance": 1, "generalDefenseFlat": 20},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "sizhandaodi", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army", TargetUnitType: "infantry",
				AllowedSides: []string{"attacker"}, Params: map[string]float64{"triggerChance": triggerChance, "attackBonusRate": 0.35},
			},
		},
		"liubei": {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true},
	}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wei", "dianwei", "shu", "liubei")
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender
	started, err := svc.StartPvpAttack(PvpAttackRequest{PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypePlunder, Troops: map[string]int{"weiInfantry": 100}, GeneralIDs: []string{"dianwei"}})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	attackerReport, err := repo.GetReportForPlayer(attacker.Player.ID, battle.AttackerReportID)
	if err != nil {
		t.Fatalf("GetReportForPlayer attacker failed: %v", err)
	}
	defenderReport, err := repo.GetReportForPlayer(defender.Player.ID, battle.DefenderReportID)
	if err != nil {
		t.Fatalf("GetReportForPlayer defender failed: %v", err)
	}
	storedMarch, _ := repo.GetPvpMarch(started.March.ID)
	attackerState, _ := repo.GetState(attacker.Player.ID)
	defenderState, _ := repo.GetState(defender.Player.ID)
	return dianweiPvpResult{battle: battle, storedMarch: storedMarch, attackerReport: attackerReport, defenderReport: defenderReport, attackerState: attackerState, defenderState: defenderState}
}

// TestPvpDianweiSizhanHitMissRecalculatesLosses 验证概率加攻先进入核心，并清除旧返兵结果。
func TestPvpDianweiSizhanHitMissRecalculatesLosses(t *testing.T) {
	miss := runDianweiAttackPvp(t, 0)
	hit := runDianweiAttackPvp(t, 1)
	if miss.battle.Result["attackerPower"] != float64(1000) || hit.battle.Result["attackerPower"] != float64(1400) || miss.battle.Result["defensePower"] != float64(10000) || hit.battle.Result["defensePower"] != float64(10000) {
		t.Fatalf("expected Sizhan attack power 1000 -> 1400, miss=%+v hit=%+v", miss.battle.Result, hit.battle.Result)
	}
	missAttackerLoss := pvpTestLossesFromBattle(t, miss.battle, "attacker")["weiInfantry"]
	hitAttackerLoss := pvpTestLossesFromBattle(t, hit.battle, "attacker")["weiInfantry"]
	missDefenderLoss := pvpTestLossesFromBattle(t, miss.battle, "defender")["shuInfantry"]
	hitDefenderLoss := pvpTestLossesFromBattle(t, hit.battle, "defender")["shuInfantry"]
	if missAttackerLoss != 96 || hitAttackerLoss != 94 || missDefenderLoss != 36 || hitDefenderLoss != 57 {
		t.Fatalf("expected exact hit/miss losses 96/36 and 94/57, got %d/%d and %d/%d", missAttackerLoss, missDefenderLoss, hitAttackerLoss, hitDefenderLoss)
	}
	for _, report := range []BattleReport{miss.attackerReport, miss.defenderReport} {
		if len(report.TraitTriggered) != 0 || len(report.TraitOutcomes) != 0 || report.Detail == nil || len(report.Detail.Traits) != 0 {
			t.Fatalf("expected legal Sizhan miss and invalid Huzhu to keep empty timeline, report=%+v", report)
		}
		if !standardDetailGeneralHasTrait(report.Detail, "huzhu_xuezhan") || !standardDetailGeneralHasTrait(report.Detail, "sizhandaodi") {
			t.Fatalf("expected owned current Dian Wei traits in snapshot, detail=%+v", report.Detail)
		}
	}
	for _, report := range []BattleReport{hit.attackerReport, hit.defenderReport} {
		outcome, ok := report.TraitOutcomes["sizhandaodi"]
		modified, modifiedOK := outcome.Detail["attackModifiedUnits"].(map[string]int)
		if !ok || !modifiedOK || modified["weiInfantry"] != 4 || outcome.Detail["attackBonusRate"] != 0.35 || outcome.Detail["triggerChance"] != 1.0 || len(report.TraitTriggered) != 1 || report.TraitTriggered[0] != "sizhandaodi" || standardReportHasTrait(report.Detail, "huzhu_xuezhan") {
			t.Fatalf("expected only Sizhan +4 attack in hit timeline, report=%+v", report)
		}
	}
	for _, result := range []dianweiPvpResult{miss, hit} {
		attackerLoss := pvpTestLossesFromBattle(t, result.battle, "attacker")["weiInfantry"]
		defenderLoss := pvpTestLossesFromBattle(t, result.battle, "defender")["shuInfantry"]
		if result.storedMarch.AttackTroops["weiInfantry"] != 100-attackerLoss || armySliceToMap(result.defenderState.Army)["shuInfantry"] != 1000-defenderLoss || pvpTestGeneralExp(result.attackerState, "dianwei") != defenderLoss || result.attackerReport.GeneralExpGained != defenderLoss {
			t.Fatalf("expected authoritative troops and exp without old return, result=%+v", result)
		}
		if result.attackerReport.RevivedUnits["weiInfantry"] != 0 || result.defenderReport.RevivedUnits["weiInfantry"] != 0 {
			t.Fatalf("expected obsolete Huzhu troop return removed, reports=%+v/%+v", result.attackerReport, result.defenderReport)
		}
	}
}

// runDianweiDefensePvp 执行典韦率禁卫甲士守城的真实 PVP。
func runDianweiDefensePvp(t *testing.T, enabled bool) dianweiPvpResult {
	t.Helper()
	setTestCombatUnitsConfig(t)
	setTestFactionsAndGenerals(t, FactionsConfig{
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "dianwei", Name: "典韦"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"liubei": {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true},
		"dianwei": {
			ID: "dianwei", Name: "典韦", Faction: "wei", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "huzhu_xuezhan", TraitType: general.TraitTypeSpecial, Enabled: enabled, Scope: "self_army", TargetUnitType: "jinWeiSoldier",
				AllowedSides: []string{"defender", "reinforcement"}, Params: map[string]float64{"triggerChance": 1, "generalDefenseFlat": 20},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "sizhandaodi", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army", TargetUnitType: "infantry",
				AllowedSides: []string{"attacker"}, Params: map[string]float64{"triggerChance": 1, "attackBonusRate": 0.35},
			},
		},
	}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "shu", "liubei", "wei", "dianwei")
	attacker.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 200}}
	defender.Army = []ArmyUnit{{UnitType: "jinWeiSoldier", Amount: 100}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender
	started, err := svc.StartPvpAttack(PvpAttackRequest{PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack, Troops: map[string]int{"shuInfantry": 200}, GeneralIDs: []string{"liubei"}})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	attackerReport, _ := repo.GetReportForPlayer(attacker.Player.ID, battle.AttackerReportID)
	defenderReport, _ := repo.GetReportForPlayer(defender.Player.ID, battle.DefenderReportID)
	storedMarch, _ := repo.GetPvpMarch(started.March.ID)
	attackerState, _ := repo.GetState(attacker.Player.ID)
	defenderState, _ := repo.GetState(defender.Player.ID)
	return dianweiPvpResult{battle: battle, storedMarch: storedMarch, attackerReport: attackerReport, defenderReport: defenderReport, attackerState: attackerState, defenderState: defenderState}
}

// TestPvpHuzhuXuezhanRecalculatesDefenseAndLosses 验证护主血战先固定加防，再结算战力、兵损和库存。
func TestPvpHuzhuXuezhanRecalculatesDefenseAndLosses(t *testing.T) {
	control := runDianweiDefensePvp(t, false)
	active := runDianweiDefensePvp(t, true)
	if control.battle.Result["attackerPower"] != float64(2000) || active.battle.Result["attackerPower"] != float64(2000) || control.battle.Result["defensePower"] != float64(1300) || active.battle.Result["defensePower"] != float64(3300) {
		t.Fatalf("expected Huzhu defense power 1300 -> 3300, control=%+v active=%+v", control.battle.Result, active.battle.Result)
	}
	controlAttackerLoss := pvpTestLossesFromBattle(t, control.battle, "attacker")["shuInfantry"]
	activeAttackerLoss := pvpTestLossesFromBattle(t, active.battle, "attacker")["shuInfantry"]
	controlDefenderLoss := pvpTestLossesFromBattle(t, control.battle, "defender")["jinWeiSoldier"]
	activeDefenderLoss := pvpTestLossesFromBattle(t, active.battle, "defender")["jinWeiSoldier"]
	if controlAttackerLoss != 108 || controlDefenderLoss != 100 || activeAttackerLoss != 200 || activeDefenderLoss != 49 {
		t.Fatalf("expected exact +20/+20 defense losses control=108/100 active=200/49, got control=%d/%d active=%d/%d", controlAttackerLoss, controlDefenderLoss, activeAttackerLoss, activeDefenderLoss)
	}
	for _, report := range []BattleReport{active.attackerReport, active.defenderReport} {
		outcome, ok := report.TraitOutcomes["huzhu_xuezhan"]
		infantry, infantryOK := outcome.Detail["infantryDefenseModifiedUnits"].(map[string]int)
		cavalry, cavalryOK := outcome.Detail["cavalryDefenseModifiedUnits"].(map[string]int)
		if !ok || !infantryOK || !cavalryOK || infantry["jinWeiSoldier"] != 20 || cavalry["jinWeiSoldier"] != 20 || outcome.Detail["generalDefenseFlat"] != float64(20) || outcome.OwnerSide != "defender" || outcome.OwnerGeneralID != "dianwei" || len(report.TraitTriggered) != 1 || report.TraitTriggered[0] != "huzhu_xuezhan" || standardReportHasTrait(report.Detail, "sizhandaodi") {
			t.Fatalf("expected only defender Huzhu +20/+20 in timeline, report=%+v", report)
		}
	}
	if active.storedMarch.AttackTroops["shuInfantry"] != 200-activeAttackerLoss || armySliceToMap(active.defenderState.Army)["jinWeiSoldier"] != 100-activeDefenderLoss || pvpTestGeneralExp(active.defenderState, "dianwei") != activeAttackerLoss || active.defenderReport.GeneralExpGained != activeAttackerLoss {
		t.Fatalf("expected authoritative Huzhu troops and exp, result=%+v", active)
	}
	if active.defenderReport.RevivedUnits["jinWeiSoldier"] != 0 {
		t.Fatalf("expected obsolete Dian Wei return removed, report=%+v", active.defenderReport)
	}
}

// TestHuzhuXuezhanOnlyStrengthensTargetOnDefense 验证护主血战只在防守/增援方向修改禁卫甲士。
func TestHuzhuXuezhanOnlyStrengthensTargetOnDefense(t *testing.T) {
	newArmies := func() (combat.Army, combat.Army) {
		return combat.Army{Units: []combat.Unit{{ID: "shuInfantry", Category: "infantry", Count: 100, Attack: 10}}}, combat.Army{Units: []combat.Unit{
			{ID: "jinWeiSoldier", Category: "infantry", Count: 100, InfantryDefense: 13, CavalryDefense: 7},
			{ID: "weiInfantry", Category: "infantry", Count: 100, InfantryDefense: 10, CavalryDefense: 8},
		}}
	}
	for _, side := range []string{"defender", "reinforcement"} {
		t.Run(side, func(t *testing.T) {
			attackerArmy, defenderArmy := newArmies()
			ctx := &general.BeforeBattleContext{Attacker: &attackerArmy, Defender: &defenderArmy}
			general.Dispatch(ctx, []general.ActiveTrait{{
				TraitID: "huzhu_xuezhan", OwnerSide: side, AllowedSides: []string{"defender", "reinforcement"}, TargetUnitType: "jinWeiSoldier",
				Params: general.Params{"triggerChance": 1, "generalDefenseFlat": 20},
			}})
			if defenderArmy.Units[0].InfantryDefense != 33 || defenderArmy.Units[0].CavalryDefense != 27 || defenderArmy.Units[1].InfantryDefense != 10 || defenderArmy.Units[1].CavalryDefense != 8 {
				t.Fatalf("expected only jinWeiSoldier defense +20/+20 for %s, army=%+v", side, defenderArmy)
			}
			outcome, ok := ctx.Triggered["huzhu_xuezhan"]
			infantry, infantryOK := outcome.Detail["infantryDefenseModifiedUnits"].(map[string]int)
			cavalry, cavalryOK := outcome.Detail["cavalryDefenseModifiedUnits"].(map[string]int)
			if !ok || !infantryOK || !cavalryOK || infantry["jinWeiSoldier"] != 20 || cavalry["jinWeiSoldier"] != 20 || outcome.OwnerSide != side {
				t.Fatalf("expected authoritative Huzhu defense deltas for %s, outcome=%+v", side, outcome)
			}
		})
	}

	attackerArmy, defenderArmy := newArmies()
	ctx := &general.BeforeBattleContext{Attacker: &attackerArmy, Defender: &defenderArmy}
	general.Dispatch(ctx, []general.ActiveTrait{{
		TraitID: "huzhu_xuezhan", OwnerSide: "attacker", AllowedSides: []string{"defender", "reinforcement"}, TargetUnitType: "jinWeiSoldier",
		Params: general.Params{"triggerChance": 1, "generalDefenseFlat": 20},
	}})
	if attackerArmy.Units[0].InfantryDefense != 0 || defenderArmy.Units[0].InfantryDefense != 13 || len(ctx.Triggered) != 0 {
		t.Fatalf("expected Huzhu Xuezhan invalid on active attack, attacker=%+v defender=%+v ctx=%+v", attackerArmy, defenderArmy, ctx)
	}
}
