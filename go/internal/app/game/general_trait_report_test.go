// 本文件验证将领特性结果写入旧战报与标准战报时不会丢失触发方和数值。
package game

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"hero3/internal/core/combat"
	"hero3/internal/core/general"
)

// TestMergeTraitOutcomesPreservesSameTraitFromBothSides 验证双方同名特性在战报中各保留一条。
func TestMergeTraitOutcomesPreservesSameTraitFromBothSides(t *testing.T) {
	report := BattleReport{ViewType: ReportViewAttack, Type: "attack", BattleType: "attack"}
	mergeTraitOutcomes(&report, map[string]general.TraitOutcome{
		"shared_trait": {
			TraitID: "shared_trait", Name: "同名特性", OwnerSide: "attacker", OwnerGeneralID: "general_a",
			Detail: map[string]interface{}{"extraLosses": map[string]int{"infantry": 10}},
		},
	})
	mergeTraitOutcomes(&report, map[string]general.TraitOutcome{
		"shared_trait": {
			TraitID: "shared_trait", Name: "同名特性", OwnerSide: "defender", OwnerGeneralID: "general_d",
			Detail: map[string]interface{}{"reducedLosses": map[string]int{"cavalry": 5}},
		},
	})

	if len(report.TraitTriggered) != 2 || len(report.TraitOutcomes) != 2 {
		t.Fatalf("expected two report outcomes, triggered=%+v outcomes=%+v", report.TraitTriggered, report.TraitOutcomes)
	}
	normalized := NormalizeBattleReport(report)
	if normalized.Detail == nil || len(normalized.Detail.Traits) != 2 {
		t.Fatalf("expected two standard report traits, got %+v", normalized.Detail)
	}
	seenSides := map[string]bool{}
	for _, trait := range normalized.Detail.Traits {
		if trait.TraitID != "shared_trait" {
			t.Fatalf("expected real trait id instead of internal storage key, got %+v", trait)
		}
		seenSides[trait.OwnerSide] = true
	}
	if !seenSides["primary"] || !seenSides["secondary"] {
		t.Fatalf("expected attacker and defender traits preserved, got %+v", normalized.Detail.Traits)
	}
}

// TestMergeTraitOutcomesCombinesSameOwnerAcrossPhases 验证同一将领的战前与掠夺结果不会互相覆盖。
func TestMergeTraitOutcomesCombinesSameOwnerAcrossPhases(t *testing.T) {
	report := BattleReport{ViewType: ReportViewDefense, Type: "defense", BattleType: "plunder"}
	owner := general.TraitOutcome{
		TraitID: "longdan_jiuyuan", Name: "龙胆救援", OwnerSide: "defender",
		OwnerGeneralID: "zhaoyun", OwnerPlayerID: "defender_player", Scope: "self_army",
	}
	before := owner
	before.Detail = map[string]interface{}{"infantryDefenseModifiedUnits": map[string]int{"qilinGuard": 25}}
	mergeTraitOutcomes(&report, map[string]general.TraitOutcome{"longdan_jiuyuan": before})
	plunder := owner
	plunder.Detail = map[string]interface{}{"protectedResources": map[string]int{"wood": 200}, "cumulativePlunderProtectionRate": 0.2}
	mergeTraitOutcomes(&report, map[string]general.TraitOutcome{"longdan_jiuyuan": plunder})

	if len(report.TraitOutcomes) != 1 || len(report.TraitTriggered) != 1 {
		t.Fatalf("expected one combined timeline entry, got triggered=%+v outcomes=%+v", report.TraitTriggered, report.TraitOutcomes)
	}
	detail := report.TraitOutcomes["longdan_jiuyuan"].Detail
	if detail["infantryDefenseModifiedUnits"] == nil || detail["protectedResources"] == nil || detail["cumulativePlunderProtectionRate"] != 0.2 {
		t.Fatalf("expected both pre-battle and plunder details, got %+v", detail)
	}
}

// TestMergeTraitOutcomesUsesDispatchOrder 验证同阶段结果按触发方及 special、bonus 配置顺序写入战报，而不是按 ID 字母排序。
func TestMergeTraitOutcomesUsesDispatchOrder(t *testing.T) {
	report := BattleReport{ViewType: ReportViewAttack, Type: "attack", BattleType: "attack"}
	mergeTraitOutcomes(&report, map[string]general.TraitOutcome{
		"aaa_attacker_bonus": {
			TraitID: "aaa_attacker_bonus", TraitType: general.TraitTypeBonus, OwnerSide: "attacker", OwnerGeneralID: "attacker_general",
		},
		"bbb_defender_special": {
			TraitID: "bbb_defender_special", TraitType: general.TraitTypeSpecial, OwnerSide: "defender", OwnerGeneralID: "defender_general",
		},
		"zzz_attacker_special": {
			TraitID: "zzz_attacker_special", TraitType: general.TraitTypeSpecial, OwnerSide: "attacker", OwnerGeneralID: "attacker_general",
		},
	})

	want := []string{"zzz_attacker_special", "aaa_attacker_bonus", "bbb_defender_special"}
	if !reflect.DeepEqual(report.TraitTriggered, want) {
		t.Fatalf("expected dispatch order %v, got %v", want, report.TraitTriggered)
	}
	normalized := NormalizeBattleReport(report)
	if normalized.Detail == nil || len(normalized.Detail.Traits) != len(want) {
		t.Fatalf("expected standard timeline with %d traits, got %+v", len(want), normalized.Detail)
	}
	for index, traitID := range want {
		if normalized.Detail.Traits[index].TraitID != traitID {
			t.Fatalf("expected standard trait %d to be %s, got %+v", index, traitID, normalized.Detail.Traits)
		}
	}
}

// TestReinforcementBeforeBattleTraitOnlyChangesOwnUnits 验证太史慈援军加防只修改当前援军批次。
func TestReinforcementBeforeBattleTraitOnlyChangesOwnUnits(t *testing.T) {
	record := Reinforcement{
		ID: "rein_taishici", FromPlayerID: "owner_taishici", FromPlayerFaction: "wu",
		Generals: []ReinforcementGeneralSnapshot{{ID: "taishici", Traits: []GeneralTraitInstance{{
			TraitID: "xinyi_yonglie", TraitType: "bonus", Name: "信义勇烈", Scope: "reinforcement_self",
			AllowedSides: []string{"reinforcement"}, Params: map[string]float64{"defenseBonusRate": 0.1},
		}}}},
	}
	original := []combat.Unit{{ID: "shadowGuard", Count: 100, Attack: 100, InfantryDefense: 80, CavalryDefense: 70}}
	changed, outcomes := applyReinforcementBeforeBattleTraits(record, original, "attack")
	if original[0].InfantryDefense != 80 || original[0].CavalryDefense != 70 || changed[0].InfantryDefense != 88 || changed[0].CavalryDefense != 77 {
		t.Fatalf("expected only copied reinforcement defenses 80/70 -> 88/77, original=%+v changed=%+v", original[0], changed[0])
	}
	if outcome, ok := outcomes["xinyi_yonglie"]; !ok || outcome.OwnerSide != "reinforcement" || outcome.OwnerGeneralID != "taishici" {
		t.Fatalf("expected reinforcement-owned trait outcome, got %+v", outcomes)
	}
}

// TestXinyiYonglieDoesNotChangeMainDefender 验证信义勇烈不会作用于主城守军。
func TestXinyiYonglieDoesNotChangeMainDefender(t *testing.T) {
	attacker := combat.Army{}
	defender := combat.Army{Faction: "wu", Units: []combat.Unit{{
		ID: "shadowGuard", Count: 100, Attack: 100, InfantryDefense: 80, CavalryDefense: 70,
	}}}
	ctx := &general.BeforeBattleContext{
		Attacker: &attacker, Defender: &defender, DefenderOwnsTrait: true, IsPvP: true, Scene: "attack",
	}
	general.Dispatch(ctx, []general.ActiveTrait{{
		TraitID: "xinyi_yonglie", TraitType: general.TraitTypeBonus, OwnerSide: "defender", OwnerGeneralID: "taishici",
		Scope: "reinforcement_self", AllowedSides: []string{"reinforcement"}, Params: general.Params{"defenseBonusRate": 0.1},
	}})
	if defender.Units[0].InfantryDefense != 80 || defender.Units[0].CavalryDefense != 70 || len(ctx.Triggered) != 0 {
		t.Fatalf("expected main defender unchanged and no trigger, defender=%+v outcomes=%+v", defender.Units[0], ctx.Triggered)
	}
}

// TestDefenseOnlyTraitsModifyOwnReinforcementUnits 验证两项纯防御特性能修改各自援军副本并记录实际变化。
func TestDefenseOnlyTraitsModifyOwnReinforcementUnits(t *testing.T) {
	tests := []struct {
		name               string
		traitID            string
		generalID          string
		params             map[string]float64
		wantInfantryChange int
		wantCavalryChange  int
	}{
		{name: "盾阵防御", traitID: "dunzhen_fangyu", generalID: "xiahouyuan", params: map[string]float64{"defenseBonusRate": 0.3, "triggerChance": 1}, wantInfantryChange: 3, wantCavalryChange: 2},
		{name: "固守汉中", traitID: "gushou_hanzhong", generalID: "weiyan", params: map[string]float64{"generalDefenseFlat": 20, "triggerChance": 1}, wantInfantryChange: 20, wantCavalryChange: 20},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			record := Reinforcement{
				ID: "rein_" + tc.generalID, FromPlayerID: "owner_" + tc.generalID, FromPlayerFaction: "wei",
				Generals: []ReinforcementGeneralSnapshot{{ID: tc.generalID, Traits: []GeneralTraitInstance{{
					TraitID: tc.traitID, TraitType: "bonus", Name: tc.name, Scope: "self_army",
					AllowedSides: []string{"defender", "reinforcement"}, Params: tc.params,
				}}}},
			}
			original := []combat.Unit{{ID: "huWei", Count: 100, Attack: 10, InfantryDefense: 10, CavalryDefense: 8}}
			changed, outcomes := applyReinforcementBeforeBattleTraits(record, original, "attack")
			if original[0].InfantryDefense != 10 || original[0].CavalryDefense != 8 ||
				changed[0].InfantryDefense != 10+tc.wantInfantryChange || changed[0].CavalryDefense != 8+tc.wantCavalryChange {
				t.Fatalf("expected only reinforcement copy defense to change, original=%+v changed=%+v", original[0], changed[0])
			}
			outcome, ok := outcomes[tc.traitID]
			infantry, infantryOK := outcome.Detail["infantryDefenseModifiedUnits"].(map[string]int)
			cavalry, cavalryOK := outcome.Detail["cavalryDefenseModifiedUnits"].(map[string]int)
			if !ok || !infantryOK || !cavalryOK || infantry["huWei"] != tc.wantInfantryChange || cavalry["huWei"] != tc.wantCavalryChange ||
				outcome.OwnerSide != "reinforcement" || outcome.OwnerGeneralID != tc.generalID {
				t.Fatalf("expected reinforcement-owned actual defense outcome, got %+v", outcome)
			}
		})
	}
}

// TestReinforcementResolutionSeparatesCombatLossesAndAfterBattleRevival 验证经验口径保留真实阵亡，但不被战斗结束后的复活倒扣。
func TestReinforcementResolutionSeparatesCombatReductionAndAfterBattleReturns(t *testing.T) {
	record := Reinforcement{
		ID: "rein_liubei", FromPlayerID: "owner_liubei", FromPlayerFaction: "shu",
		RemainingTroops: map[string]int{"shuInfantry": 100},
		Generals: []ReinforcementGeneralSnapshot{{ID: "liubei", Traits: []GeneralTraitInstance{
			{
				TraitID: "rende", TraitType: "special", Scope: "self_army",
				Params: map[string]float64{"politicsBonus": 10, "commandBonus": 12},
			},
			{
				TraitID: "renzhu_shouhu", TraitType: "bonus", Scope: "self_army",
				Params: map[string]float64{"effectRate": 0.35, "triggerChance": 1},
			},
		}}},
	}
	resolution := resolveReinforcementAfterBattleTraits(
		[]Reinforcement{record}, map[string]map[string]int{record.ID: {"shuInfantry": 100}}, "attacker", "plunder", 0,
	)
	if resolution.AfterCombatLosses[record.ID]["shuInfantry"] != 100 || resolution.FinalLosses[record.ID]["shuInfantry"] != 65 {
		t.Fatalf("expected combat losses 100 and final losses 65 after 35 revival, resolution=%+v", resolution)
	}
	if _, exists := resolution.Outcomes[record.ID]["rende"]; exists || resolution.Outcomes[record.ID]["renzhu_shouhu"].TraitID != "renzhu_shouhu" {
		t.Fatalf("expected only Renzhu Shouhu after-battle outcome, outcomes=%+v", resolution.Outcomes)
	}
}

// TestSweepReportUsesFinalSurvivorsInsteadOfGrossLossSubtraction 验证复活兵重复参战时聚合战报使用最终真实兵力。
func TestSweepReportUsesFinalSurvivorsInsteadOfGrossLossSubtraction(t *testing.T) {
	reports := []BattleReport{
		{
			ID: "sweep_1", PlayerID: "player", PlayerFaction: "wei", Type: "attack", Result: "attacker_victory", CreatedAt: "2026-07-19T01:00:00Z",
			DispatchedUnits: map[string]int{"weiInfantry": 100}, LostUnits: map[string]int{"weiInfantry": 70}, SurvivedUnits: map[string]int{"weiInfantry": 65}, RevivedUnits: map[string]int{"weiInfantry": 35},
		},
		{
			ID: "sweep_2", PlayerID: "player", PlayerFaction: "wei", Type: "attack", Result: "attacker_victory", CreatedAt: "2026-07-19T01:01:00Z",
			DispatchedUnits: map[string]int{"weiInfantry": 65}, LostUnits: map[string]int{"weiInfantry": 50}, SurvivedUnits: map[string]int{"weiInfantry": 40}, RevivedUnits: map[string]int{"weiInfantry": 25},
		},
	}
	aggregate := buildNpcSweepReport("sweep_aggregate", reports, "attack", 2, 0, false)
	if aggregate.LostUnits["weiInfantry"] != 120 || aggregate.RevivedUnits["weiInfantry"] != 60 || aggregate.SurvivedUnits["weiInfantry"] != 40 {
		t.Fatalf("expected gross losses 120, revivals 60 and real survivors 40, got %+v", aggregate)
	}
	if aggregate.Detail == nil {
		t.Fatal("expected normalized sweep detail")
	}
	found := false
	for _, unit := range aggregate.Detail.PrimarySide.Units {
		if unit.UnitType == "weiInfantry" && unit.Survived != 40 {
			t.Fatalf("expected standard report to preserve 40 real survivors, got %+v", unit)
		}
		if unit.UnitType == "weiInfantry" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected weiInfantry in standard sweep detail, got %+v", aggregate.Detail.PrimarySide.Units)
	}
}

// TestSweepReportAggregatesRepeatedTraitOutcomeValues 验证多场扫荡累计实际数量，但不把单场攻防修正错误叠高。
func TestSweepReportAggregatesRepeatedTraitOutcomeValues(t *testing.T) {
	reports := []BattleReport{
		{
			ID: "sweep_trait_1", PlayerID: "player", PlayerFaction: "shu", Type: "attack", Result: "attacker_victory", CreatedAt: "2026-07-19T02:00:00Z",
			DispatchedUnits: map[string]int{"greedyWolf": 100}, LostUnits: map[string]int{"greedyWolf": 10}, SurvivedUnits: map[string]int{"greedyWolf": 90},
			DefenderUnits: map[string]int{"shadowGuard": 100}, DefenderLostUnits: map[string]int{"shadowGuard": 20},
			TraitTriggered: []string{"laodang_yizhuang"}, TraitOutcomes: map[string]TraitOutcomeReport{
				"laodang_yizhuang": {TraitID: "laodang_yizhuang", Name: "老当益壮", OwnerSide: "attacker", OwnerGeneralID: "huangzhong", Detail: map[string]interface{}{
					"effectRate": 0.1, "extraLosses": map[string]int{"shadowGuard": 7}, "attackModifiedUnits": map[string]int{"greedyWolf": 10},
				}},
			},
		},
		{
			ID: "sweep_trait_2", PlayerID: "player", PlayerFaction: "shu", Type: "attack", Result: "attacker_victory", CreatedAt: "2026-07-19T02:01:00Z",
			DispatchedUnits: map[string]int{"greedyWolf": 90}, LostUnits: map[string]int{"greedyWolf": 12}, SurvivedUnits: map[string]int{"greedyWolf": 78},
			DefenderUnits: map[string]int{"shadowGuard": 80}, DefenderLostUnits: map[string]int{"shadowGuard": 18},
			TraitTriggered: []string{"laodang_yizhuang"}, TraitOutcomes: map[string]TraitOutcomeReport{
				"laodang_yizhuang": {TraitID: "laodang_yizhuang", Name: "老当益壮", OwnerSide: "attacker", OwnerGeneralID: "huangzhong", Detail: map[string]interface{}{
					"effectRate": 0.1, "extraLosses": map[string]int{"shadowGuard": 5}, "attackModifiedUnits": map[string]int{"greedyWolf": 10},
				}},
			},
		},
	}

	aggregate := buildNpcSweepReport("sweep_trait_aggregate", reports, "attack", 2, 0, false)
	outcome := aggregate.TraitOutcomes["laodang_yizhuang"]
	extraLosses, ok := outcome.Detail["extraLosses"].(map[string]int)
	if !ok || extraLosses["shadowGuard"] != 12 {
		t.Fatalf("expected repeated extra losses 7+5=12, got %+v", outcome.Detail)
	}
	if outcome.Detail["triggerCount"] != 2 {
		t.Fatalf("expected two triggering battles, got %+v", outcome.Detail)
	}
	attackChanges, ok := outcome.Detail["attackModifiedUnits"].(map[string]int)
	if !ok || attackChanges["greedyWolf"] != 10 {
		t.Fatalf("expected per-battle attack change to remain +10 instead of becoming +20, got %+v", outcome.Detail)
	}
	if aggregate.DefenderLostUnits["shadowGuard"] != 38 || aggregate.Detail == nil || len(aggregate.Detail.Traits) != 1 {
		t.Fatalf("expected aggregate loss and one merged trait row, got report=%+v detail=%+v", aggregate.DefenderLostUnits, aggregate.Detail)
	}
}

// TestSweepReportAggregatesCapturedGarrisonDetail 验证跨阵营 NPC 扫荡会累计每场实际进入驻防的俘虏。
func TestSweepReportAggregatesCapturedGarrisonDetail(t *testing.T) {
	current := TraitOutcomeReport{TraitID: "meiren", Detail: map[string]interface{}{
		"capturedToGarrison": map[string]int{"greedyWolf": 7}, "totalCaptured": 7,
	}}
	next := TraitOutcomeReport{TraitID: "meiren", Detail: map[string]interface{}{
		"capturedToGarrison": map[string]int{"greedyWolf": 5}, "totalCaptured": 5,
	}}

	merged := mergeRepeatedSweepTraitOutcome(current, next)
	captured, ok := merged.Detail["capturedToGarrison"].(map[string]int)
	if !ok || captured["greedyWolf"] != 12 || merged.Detail["totalCaptured"] != 12 || merged.Detail["triggerCount"] != 2 {
		t.Fatalf("expected two capture results to aggregate to 12 garrison troops, got %+v", merged.Detail)
	}
}

// TestNpcSweepAggregatesRealRepeatedFireDamage 验证真实连续扫荡会累计每场火攻，并与 NPC 最终兵力一致。
func TestNpcSweepAggregatesRealRepeatedFireDamage(t *testing.T) {
	setTestCombatUnitsConfig(t)
	setTestGeneralsConfig(t, GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"test_general": {
				ID: "test_general", Name: "测试将领", Faction: "wei", Enabled: true,
				SpecialTrait: GeneralTraitConfig{
					TraitID: "huogong", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army",
					AllowedSides: []string{"attacker"}, Params: map[string]float64{"effectRate": 0.1, "triggerChance": 1},
				},
			},
		},
	})

	svc := NewService()
	repo := svc.repo.(*MemoryRepository)
	now := time.Now().UTC()
	account := Account{ID: "acc_sweep_fire", Username: "sweep_fire", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_sweep_fire", "SweepFire", "wei", "test_general", now)
	state.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 300}}
	firstNPC := testNpcCity("npc_sweep_fire_1", now)
	secondNPC := testNpcCity("npc_sweep_fire_2", now)
	firstNPC.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	firstNPC.MaxArmy = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	secondNPC.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	secondNPC.MaxArmy = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	state.NpcState = &NpcState{Cities: []NpcCity{firstNPC, secondNPC}, LastRefreshedAt: now.Format(resourceDateLayout)}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	response, err := svc.SweepNpc(SweepNpcRequest{
		PlayerID: state.Player.ID, NpcIDs: []string{firstNPC.ID, secondNPC.ID}, Mode: "plunder", GeneralIDs: []string{"test_general"},
	})
	if err != nil {
		t.Fatalf("SweepNpc failed: %v", err)
	}
	if response.Done != 2 || response.Failed != 0 || response.Stopped {
		t.Fatalf("expected two completed battles, got %+v", response)
	}
	report := response.BattleReport
	outcome, triggered := report.TraitOutcomes["huogong"]
	if !triggered || outcome.Detail["triggerCount"] != 2 {
		t.Fatalf("expected fire attack in both sweep battles, got %+v", outcome)
	}
	extraDamage, ok := traitDetailInt(outcome.Detail["extraDamage"])
	if !ok || extraDamage <= 0 {
		t.Fatalf("expected positive aggregated fire damage, got %+v", outcome.Detail)
	}
	stored, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	remainingDefenders := 0
	for _, npc := range stored.NpcState.Cities {
		remainingDefenders += armySliceToMap(npc.Army)["weiInfantry"]
	}
	if report.DefenderUnits["weiInfantry"] != 200 || report.DefenderLostUnits["weiInfantry"] != 200-remainingDefenders {
		t.Fatalf("expected aggregate defender losses to match both NPC states, report=%+v remaining=%d", report.DefenderLostUnits, remainingDefenders)
	}
	if report.Detail == nil || len(report.Detail.Traits) != 1 || report.Detail.Traits[0].Detail["triggerCount"] != 2 {
		t.Fatalf("expected standard report to preserve one aggregated fire row, got %+v", report.Detail)
	}
}

// TestNpcSweepLossReductionTraitsMatchReturnedArmy 验证鬼才遗策在真实扫荡后复活并写入聚合状态。
func TestNpcSweepLossReductionTraitsMatchReturnedArmy(t *testing.T) {
	cases := []struct {
		name      string
		traitID   string
		traitType string
	}{
		{name: "郭嘉鬼才遗策", traitID: "guicai_yice", traitType: general.TraitTypeBonus},
	}
	for index, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setTestCombatUnitsConfig(t)
			traitCfg := GeneralTraitConfig{
				TraitID: tc.traitID, TraitType: tc.traitType, Enabled: true, Scope: "self_army",
				AllowedSides: []string{"attacker", "defender", "reinforcement"},
				Params:       map[string]float64{"effectRate": 0.22, "triggerChance": 1},
			}
			hero := GeneralHeroConfig{ID: "test_general", Name: "测试将领", Faction: "wei", Enabled: true}
			if tc.traitType == general.TraitTypeSpecial {
				hero.SpecialTrait = traitCfg
			} else {
				hero.BonusTrait = traitCfg
			}
			setTestGeneralsConfig(t, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{"test_general": hero}})

			svc := NewService()
			repo := svc.repo.(*MemoryRepository)
			now := time.Now().UTC()
			suffix := string(rune('a' + index))
			account := Account{ID: "acc_sweep_loss_" + suffix, Username: "sweep_loss_" + suffix, PasswordHash: "x", CreatedAt: now}
			if err := repo.CreateAccount(account); err != nil {
				t.Fatalf("create account: %v", err)
			}
			state := newPlayerState("player_sweep_loss_"+suffix, "SweepLoss", "wei", "test_general", now)
			state.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
			npc := testNpcCity("npc_sweep_loss_"+suffix, now)
			npc.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 200}}
			npc.MaxArmy = []ArmyUnit{{UnitType: "weiInfantry", Amount: 200}}
			state.NpcState = &NpcState{Cities: []NpcCity{npc}, LastRefreshedAt: now.Format(resourceDateLayout)}
			if err := repo.CreatePlayer(account.ID, state, now); err != nil {
				t.Fatalf("create player: %v", err)
			}

			response, err := svc.SweepNpc(SweepNpcRequest{
				PlayerID: state.Player.ID, NpcIDs: []string{npc.ID}, Mode: "attack", GeneralIDs: []string{"test_general"},
			})
			if err != nil {
				t.Fatalf("SweepNpc failed: %v", err)
			}
			report := response.BattleReport
			if report.Result != "defender_victory" || report.LostUnits["weiInfantry"] != 100 || report.RevivedUnits["weiInfantry"] != 22 || report.SurvivedUnits["weiInfantry"] != 22 {
				t.Fatalf("expected real loss with 22 revived troops, got result=%s lost=%+v revived=%+v survived=%+v", report.Result, report.LostUnits, report.RevivedUnits, report.SurvivedUnits)
			}
			revived, ok := report.TraitOutcomes[tc.traitID].Detail["revivedUnits"].(map[string]int)
			if !ok || revived["weiInfantry"] != 22 {
				t.Fatalf("expected %s to report 22 revived troops, got %+v", tc.traitID, report.TraitOutcomes[tc.traitID])
			}
			stored, err := repo.GetState(state.Player.ID)
			if err != nil {
				t.Fatalf("get state: %v", err)
			}
			if armySliceToMap(stored.Army)["weiInfantry"] != 22 || report.Detail == nil {
				t.Fatalf("expected real army and standard report to keep 22 troops, army=%+v detail=%+v", stored.Army, report.Detail)
			}
			unitMatched := false
			for _, unit := range report.Detail.PrimarySide.Units {
				if unit.UnitType == "weiInfantry" && unit.Survived == 22 {
					unitMatched = true
				}
			}
			if !unitMatched {
				t.Fatalf("expected standard report to keep 22 troops, detail=%+v", report.Detail)
			}
			if report.Summary == "" || !strings.Contains(report.Summary, "负 1") {
				t.Fatalf("expected sweep summary to disclose one defeat, got %q", report.Summary)
			}
		})
	}
}
