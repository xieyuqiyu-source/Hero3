// 本文件验证战前真实伤亡与临时压制在服务结算、真实兵力和战报中的不同口径。
package game

import (
	"maps"
	"testing"
	"time"

	"hero3/internal/core/combat"
	"hero3/internal/core/general"
)

// TestConfiguredUnitBonusesModifyOnlyTheirRealTarget 验证正式配置中的全军防御和指定兵种攻击加成作用于真实目标。
func TestConfiguredUnitBonusesModifyOnlyTheirRealTarget(t *testing.T) {
	originalGenerals := GetGeneralsConfig()
	originalFactions := GetFactionsConfig()
	originalUnits := GetUnitsConfig()
	if err := LoadFactionsConfig("../../../config/factions.json"); err != nil {
		t.Fatalf("load factions config: %v", err)
	}
	if err := LoadUnitsConfig("../../../config/units"); err != nil {
		t.Fatalf("load units config: %v", err)
	}
	if err := LoadGeneralsConfig("../../../config/generals.json"); err != nil {
		t.Fatalf("load generals config: %v", err)
	}
	t.Cleanup(func() {
		generalsMu.Lock()
		activeGenerals = originalGenerals
		generalsMu.Unlock()
		factionsMu.Lock()
		activeFactions = originalFactions
		factionsMu.Unlock()
		unitsMu.Lock()
		activeUnits = originalUnits
		unitsMu.Unlock()
	})

	combatUnit := func(faction string, unitID string) combat.Unit {
		cfg, ok := GetUnitConfig(faction, unitID)
		if !ok {
			t.Fatalf("missing real unit config %s/%s", faction, unitID)
		}
		return combat.Unit{
			ID: unitID, Category: cfg.Category, Count: 100,
			Attack: cfg.Stats["attack"], InfantryDefense: cfg.Stats["infantryDefense"], CavalryDefense: cfg.Stats["cavalryDefense"],
		}
	}

	t.Run("曹操守城强化全军防御且进攻无效", func(t *testing.T) {
		attacker := combat.Army{Units: []combat.Unit{combatUnit("wu", "shadowGuard")}}
		defender := combat.Army{Units: []combat.Unit{combatUnit("wei", "huWei"), combatUnit("wei", "qingZhouArmy")}}
		ctx := &general.BeforeBattleContext{Attacker: &attacker, Defender: &defender, DefenderOwnsTrait: true, Scene: "attack"}
		general.Dispatch(ctx, buildActiveTraits(newGeneral("wei", "caocao")))

		if got := defender.Units[0]; got.Attack != 14 || got.InfantryDefense != 9 || got.CavalryDefense != 6 {
			t.Fatalf("expected huWei defense 8/5 -> 9/6 and attack unchanged, got %+v", got)
		}
		if got := defender.Units[1]; got.Attack != 8 || got.InfantryDefense != 8 || got.CavalryDefense != 12 {
			t.Fatalf("expected qingZhouArmy defense 7/10 -> 8/12 and attack unchanged, got %+v", got)
		}
		outcome, ok := ctx.Triggered["weiwu_tongyu"]
		infantryDefense, infantryOK := outcome.Detail["infantryDefenseModifiedUnits"].(map[string]int)
		cavalryDefense, cavalryOK := outcome.Detail["cavalryDefenseModifiedUnits"].(map[string]int)
		if !ok || !infantryOK || !cavalryOK || infantryDefense["huWei"] != 1 || cavalryDefense["huWei"] != 1 || infantryDefense["qingZhouArmy"] != 1 || cavalryDefense["qingZhouArmy"] != 2 {
			t.Fatalf("expected report to record all-army 15%% defense deltas, got %+v", outcome)
		}
		if _, exists := outcome.Detail["attackModifiedUnits"]; exists {
			t.Fatalf("expected Weiwu Tongyu not to modify attack, got %+v", outcome.Detail)
		}

		attackCtx := &general.BeforeBattleContext{Attacker: &defender, Defender: &attacker, AttackerOwnsTrait: true, Scene: "attack"}
		general.Dispatch(attackCtx, buildActiveTraits(newGeneral("wei", "caocao")))
		if _, triggered := attackCtx.Triggered["weiwu_tongyu"]; triggered {
			t.Fatalf("expected Weiwu Tongyu disabled on active attack, got %+v", attackCtx.Triggered)
		}
	})

	t.Run("孙策只强化霸王骑", func(t *testing.T) {
		attacker := combat.Army{Units: []combat.Unit{combatUnit("wu", "overlordRider"), combatUnit("wu", "zhuQueRider")}}
		defender := combat.Army{Units: []combat.Unit{combatUnit("wei", "qingZhouArmy")}}
		ctx := &general.BeforeBattleContext{Attacker: &attacker, Defender: &defender, AttackerOwnsTrait: true, Scene: "attack"}
		general.Dispatch(ctx, buildActiveTraits(newGeneral("wu", "sunce")))

		if attacker.Units[0].Attack != 78 || attacker.Units[1].Attack != 9 {
			t.Fatalf("expected only overlordRider attack 28+50, got %+v", attacker.Units)
		}
		outcome, ok := ctx.Triggered["xiaobawang_tieqi"]
		modified, detailOK := outcome.Detail["attackModifiedUnits"].(map[string]int)
		if !ok || !detailOK || modified["overlordRider"] != 50 || len(modified) != 1 {
			t.Fatalf("expected report outcome only for overlordRider +50, got %+v", ctx.Triggered)
		}
		if _, exists := outcome.Detail["effectRate"]; exists {
			t.Fatalf("expected no fake zero effectRate for fixed attack trait, got %+v", outcome.Detail)
		}
	})
}

// TestApplyPreBattleLossesToCombatResultPreservesSuppressedUnits 验证只把真实伤亡并入战损，未参战兵不被误扣。
func TestApplyPreBattleLossesToCombatResultPreservesSuppressedUnits(t *testing.T) {
	result := combat.CombatResult{
		AttackerLosses: []combat.UnitLoss{{ID: "infantry", Count: 60, Losses: 20}},
		DefenderLosses: []combat.UnitLoss{{ID: "cavalry", Count: 50, Losses: 10}},
	}
	ctx := &general.BeforeBattleContext{
		AttackerPreBattleLosses: map[string]int{"infantry": 35},
		DefenderPreBattleLosses: map[string]int{"cavalry": 25},
	}
	applyPreBattleLossesToCombatResult(&result, ctx)

	if got := result.AttackerLosses[0]; got.Count != 95 || got.Losses != 55 {
		t.Fatalf("expected attacker count/loss 95/55 with five suppressed outside settlement, got %+v", got)
	}
	if got := result.DefenderLosses[0]; got.Count != 75 || got.Losses != 35 {
		t.Fatalf("expected defender count/loss 75/35 with suppressed outside settlement, got %+v", got)
	}
	if result.AttackerLossRate != float64(55)/95 || result.DefenderLossRate != float64(35)/75 {
		t.Fatalf("expected recomputed loss rates, got attacker=%f defender=%f", result.AttackerLossRate, result.DefenderLossRate)
	}
}

// TestNpcPreDamageMatchesRealArmyAndReport 验证疑兵偷袭的战前伤亡会真实扣除 NPC 守军并写入同值战报。
func TestNpcPreDamageMatchesRealArmyAndReport(t *testing.T) {
	setTestCombatUnitsConfig(t)
	setTestGeneralsConfig(t, GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"test_general": {
				ID: "test_general", Name: "测试将领", Faction: "wei", Enabled: true,
				SpecialTrait: GeneralTraitConfig{TraitID: "yibing_touxi", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army", Params: map[string]float64{"effectRate": 0.35, "triggerChance": 1}},
			},
		},
	})

	svc := NewService()
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_pre_damage_npc", Username: "pre_damage_npc", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_pre_damage_npc", "PreDamage", "wei", "test_general", now)
	state.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 200}}
	npc := testNpcCity("npc_pre_damage", now)
	npc.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	npc.MaxArmy = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	state.NpcState = &NpcState{Cities: []NpcCity{npc}, LastRefreshedAt: now.UTC().Format(resourceDateLayout)}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	response, err := svc.AttackNpc(AttackNpcRequest{
		PlayerID: state.Player.ID, NpcID: npc.ID, Mode: "plunder",
		Units: map[string]int{"weiInfantry": 200}, GeneralIDs: []string{"test_general"},
	})
	if err != nil {
		t.Fatalf("AttackNpc failed: %v", err)
	}
	report := response.BattleReport
	outcome := report.TraitOutcomes["yibing_touxi"]
	preDamage, ok := outcome.Detail["preBattleAffected"].(map[string]int)
	if !ok || preDamage["weiInfantry"] != 35 {
		t.Fatalf("expected report to record 35 pre-battle losses, got %+v", outcome)
	}
	if report.DefenderUnits["weiInfantry"] != 100 || report.DefenderLostUnits["weiInfantry"] < 35 {
		t.Fatalf("expected original defender count and losses including pre-damage, got before=%+v lost=%+v", report.DefenderUnits, report.DefenderLostUnits)
	}
	stored, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	remaining := stored.NpcState.Cities[0].Army[0].Amount
	if remaining != 100-report.DefenderLostUnits["weiInfantry"] {
		t.Fatalf("expected NPC remaining %d to match report, got %d", 100-report.DefenderLostUnits["weiInfantry"], remaining)
	}
	if report.Detail == nil || report.Detail.SecondarySide == nil {
		t.Fatalf("expected standard defender report detail, got %+v", report.Detail)
	}
	for _, unit := range report.Detail.SecondarySide.Units {
		if unit.UnitType == "weiInfantry" && (unit.AmountBefore != 100 || unit.Lost != report.DefenderLostUnits["weiInfantry"] || unit.Survived != remaining) {
			t.Fatalf("expected standard report to match real NPC state, got %+v", unit)
		}
	}
}

// TestNpcSuppressionPreservesTroopsAndReportBaseline 验证奇门压制兵不被扣除，战报仍保留完整战前兵力。
func TestNpcSuppressionPreservesTroopsAndReportBaseline(t *testing.T) {
	setTestCombatUnitsConfig(t)
	setTestGeneralsConfig(t, GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"test_general": {
				ID: "test_general", Name: "测试将领", Faction: "wei", Enabled: true,
				SpecialTrait: GeneralTraitConfig{TraitID: "qimen_dunjia", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army", Params: map[string]float64{"effectRate": 0.5, "triggerChance": 1}},
			},
		},
	})

	svc := NewService()
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_suppress_npc", Username: "suppress_npc", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_suppress_npc", "Suppress", "wei", "test_general", now)
	state.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 200}}
	npc := testNpcCity("npc_suppress", now)
	npc.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	npc.MaxArmy = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	state.NpcState = &NpcState{Cities: []NpcCity{npc}, LastRefreshedAt: now.UTC().Format(resourceDateLayout)}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	response, err := svc.AttackNpc(AttackNpcRequest{
		PlayerID: state.Player.ID, NpcID: npc.ID, Mode: "plunder",
		Units: map[string]int{"weiInfantry": 200}, GeneralIDs: []string{"test_general"},
	})
	if err != nil {
		t.Fatalf("AttackNpc failed: %v", err)
	}
	report := response.BattleReport
	outcome := report.TraitOutcomes["qimen_dunjia"]
	suppressed, ok := outcome.Detail["suppressedUnits"].(map[string]int)
	if !ok || suppressed["weiInfantry"] != 50 {
		t.Fatalf("expected report to record 50 suppressed troops, got %+v", outcome)
	}
	if report.DefenderUnits["weiInfantry"] != 100 || report.DefenderLostUnits["weiInfantry"] > 50 {
		t.Fatalf("expected complete baseline and losses limited to 50 participants, got before=%+v lost=%+v", report.DefenderUnits, report.DefenderLostUnits)
	}
	stored, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	remaining := stored.NpcState.Cities[0].Army[0].Amount
	if remaining != 100-report.DefenderLostUnits["weiInfantry"] || remaining < 50 {
		t.Fatalf("expected suppressed troops preserved in remaining army, remaining=%d lost=%+v", remaining, report.DefenderLostUnits)
	}
	if report.Detail == nil || report.Detail.SecondarySide == nil {
		t.Fatalf("expected standard defender report detail, got %+v", report.Detail)
	}
	for _, unit := range report.Detail.SecondarySide.Units {
		if unit.UnitType == "weiInfantry" && (unit.AmountBefore != 100 || unit.Lost != report.DefenderLostUnits["weiInfantry"] || unit.Survived != remaining) {
			t.Fatalf("expected standard suppression report to match real NPC state, got %+v", unit)
		}
	}
}

// TestNpcFormalPreBattleTraitsMatchRealStateAndBothReports 验证五项正式战前伤亡或压制特性逐项进入 NPC 权威兵力和两套战报。
func TestNpcFormalPreBattleTraitsMatchRealStateAndBothReports(t *testing.T) {
	cases := []struct {
		name        string
		traitID     string
		generalID   string
		generalName string
		detailKey   string
		rate        float64
	}{
		{name: "疑兵偷袭", traitID: "yibing_touxi", generalID: "simayi", generalName: "司马懿", detailKey: "preBattleAffected", rate: 0.35},
		{name: "水淹七军", traitID: "shuiyan_qijun", generalID: "guanyu", generalName: "关羽", detailKey: "preBattleAffected", rate: 0.35},
		{name: "震慑全军", traitID: "weizhen_zhenhe", generalID: "zhangliao", generalName: "张辽", detailKey: "suppressedUnits", rate: 0.25},
		{name: "震慑全军", traitID: "zhenhe_quanjun", generalID: "zhangfei", generalName: "张飞", detailKey: "suppressedUnits", rate: 0.5},
		{name: "奇门遁甲", traitID: "qimen_dunjia", generalID: "zhugeliang", generalName: "诸葛亮", detailKey: "suppressedUnits", rate: 0.25},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			report, stored := resolveNpcFormalPreBattleTraitTest(t, tc.traitID, tc.generalID, tc.generalName, tc.rate)
			wantAffected := int(100 * tc.rate)
			outcome, ok := report.TraitOutcomes[tc.traitID]
			affected, detailOK := outcome.Detail[tc.detailKey].(map[string]int)
			maxRateValid := outcome.Detail["maxAffectedRate"] == tc.rate
			if tc.traitID == "yibing_touxi" || tc.traitID == "weizhen_zhenhe" {
				_, maxRateExists := outcome.Detail["maxAffectedRate"]
				maxRateValid = !maxRateExists
			}
			if !ok || !detailOK || affected["weiInfantry"] != wantAffected || outcome.Detail["effectRate"] != tc.rate || !maxRateValid || outcome.Detail["triggerChance"] != 1.0 {
				t.Fatalf("expected %s design and actual result %d, outcome=%+v", tc.traitID, wantAffected, outcome)
			}
			if outcome.OwnerSide != "attacker" || outcome.OwnerGeneralID != tc.generalID {
				t.Fatalf("expected %s owned by attacking general %s, outcome=%+v", tc.traitID, tc.generalID, outcome)
			}

			lost := report.DefenderLostUnits["weiInfantry"]
			remaining := armySliceToMap(stored.NpcState.Cities[0].Army)["weiInfantry"]
			if report.DefenderUnits["weiInfantry"] != 100 || remaining != 100-lost {
				t.Fatalf("expected %s NPC state to match legacy report, before=%+v lost=%+v remaining=%d", tc.traitID, report.DefenderUnits, report.DefenderLostUnits, remaining)
			}
			if tc.detailKey == "preBattleAffected" && lost < wantAffected {
				t.Fatalf("expected %s real losses to include %d pre-battle deaths, lost=%d", tc.traitID, wantAffected, lost)
			}
			if tc.detailKey == "suppressedUnits" && (lost > 100-wantAffected || remaining < wantAffected) {
				t.Fatalf("expected %s suppressed troops preserved, affected=%d lost=%d remaining=%d", tc.traitID, wantAffected, lost, remaining)
			}

			if report.Detail == nil || report.Detail.SecondarySide == nil {
				t.Fatalf("expected %s standard NPC report, detail=%+v", tc.traitID, report.Detail)
			}
			standardUnitFound := false
			for _, unit := range report.Detail.SecondarySide.Units {
				if unit.UnitType == "weiInfantry" {
					standardUnitFound = unit.AmountBefore == 100 && unit.Lost == lost && unit.Survived == remaining
				}
			}
			standardTraitFound := false
			for _, trait := range report.Detail.Traits {
				if trait.TraitID != tc.traitID {
					continue
				}
				standardAffected, standardOK := trait.Detail[tc.detailKey].(map[string]int)
				standardTraitFound = standardOK && standardAffected["weiInfantry"] == wantAffected && trait.OwnerSide == "primary" && trait.OwnerRole == "attacker" && trait.GeneralID == tc.generalID
			}
			if !standardUnitFound || !standardTraitFound {
				t.Fatalf("expected %s standard report to match real state and owner, detail=%+v", tc.traitID, report.Detail)
			}
		})
	}
}

// TestNpcFormalRandomPreBattleHitAndMissKeepBonusIndependent 验证正式将领的随机战前能力未命中时不会吞掉另一项战斗加成。
func TestNpcFormalRandomPreBattleHitAndMissKeepBonusIndependent(t *testing.T) {
	cases := []struct {
		name             string
		generalID        string
		generalName      string
		attackerUnit     string
		specialTraitID   string
		bonusTraitID     string
		specialDetailKey string
		specialActual    int
		specialSides     []string
		specialParams    map[string]float64
		bonusScope       string
		bonusTarget      string
		bonusParams      map[string]float64
		hitPlayerPower   int
		hitEnemyPower    int
		missPlayerPower  int
		missEnemyPower   int
		hitPlayerLosses  int
		missPlayerLosses int
		hitEnemyLosses   int
		missEnemyLosses  int
	}{
		{
			name: "关羽", generalID: "guanyu", generalName: "关羽", attackerUnit: "weiInfantry",
			specialTraitID: "shuiyan_qijun", bonusTraitID: "wusheng_pojun", specialDetailKey: "preBattleAffected", specialActual: 35, specialParams: map[string]float64{"effectRate": 0.35, "maxAffectedRate": 0.35},
			bonusScope: "self_army", bonusParams: map[string]float64{"attackBonusRate": 0.2},
			hitPlayerPower: 2400, hitEnemyPower: 650, missPlayerPower: 2400, missEnemyPower: 1000,
			hitPlayerLosses: 31, missPlayerLosses: 57, hitEnemyLosses: 100, missEnemyLosses: 100,
		},
		{
			name: "张辽", generalID: "zhangliao", generalName: "张辽", attackerUnit: "weiCavalry",
			specialTraitID: "weizhen_zhenhe", bonusTraitID: "weizhen_xiaoyao", specialDetailKey: "fledUnits", specialActual: 25, specialSides: []string{"attacker"}, specialParams: map[string]float64{"effectRate": 0.25},
			bonusScope: "self_army", bonusTarget: "cavalry", bonusParams: map[string]float64{"triggerChance": 1, "attackBonusRate": 0.35},
			hitPlayerPower: 3800, hitEnemyPower: 600, missPlayerPower: 3800, missEnemyPower: 800,
			hitPlayerLosses: 14, missPlayerLosses: 21, hitEnemyLosses: 75, missEnemyLosses: 100,
		},
		{
			name: "张飞", generalID: "zhangfei", generalName: "张飞", attackerUnit: "weiInfantry",
			specialTraitID: "zhenhe_quanjun", bonusTraitID: "wanren_nuhou", specialDetailKey: "suppressedUnits", specialActual: 50, specialParams: map[string]float64{"effectRate": 0.5, "maxAffectedRate": 0.5},
			bonusScope: "self_army", bonusTarget: "infantry", bonusParams: map[string]float64{"attackBonusRate": 0.2},
			hitPlayerPower: 2400, hitEnemyPower: 500, missPlayerPower: 2400, missEnemyPower: 1000,
			hitPlayerLosses: 21, missPlayerLosses: 57, hitEnemyLosses: 50, missEnemyLosses: 100,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, probability := range []struct {
				name             string
				triggerChance    float64
				wantPlayerPower  int
				wantEnemyPower   int
				wantPlayerLosses int
				wantEnemyLosses  int
				wantSpecial      bool
			}{
				{name: "命中", triggerChance: 1, wantPlayerPower: tc.hitPlayerPower, wantEnemyPower: tc.hitEnemyPower, wantPlayerLosses: tc.hitPlayerLosses, wantEnemyLosses: tc.hitEnemyLosses, wantSpecial: true},
				{name: "合法未命中", triggerChance: 0, wantPlayerPower: tc.missPlayerPower, wantEnemyPower: tc.missEnemyPower, wantPlayerLosses: tc.missPlayerLosses, wantEnemyLosses: tc.missEnemyLosses},
			} {
				t.Run(probability.name, func(t *testing.T) {
					specialParams := maps.Clone(tc.specialParams)
					specialParams["triggerChance"] = probability.triggerChance
					hero := GeneralHeroConfig{
						ID: tc.generalID, Name: tc.generalName, Faction: "wei", Enabled: true,
						SpecialTrait: GeneralTraitConfig{
							TraitID: tc.specialTraitID, TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army", AllowedSides: tc.specialSides, Params: specialParams,
						},
						BonusTrait: GeneralTraitConfig{
							TraitID: tc.bonusTraitID, TraitType: general.TraitTypeBonus, Enabled: true, Scope: tc.bonusScope, TargetUnitType: tc.bonusTarget,
							AllowedSides: []string{"attacker"}, Params: tc.bonusParams,
						},
					}
					report, stored := resolveNpcFormalRandomPreBattleHeroTest(t, tc.name+"_"+probability.name, hero, tc.attackerUnit)
					if report.Result != "attacker_victory" || report.PlayerPower != probability.wantPlayerPower || report.EnemyPower != probability.wantEnemyPower || report.LostUnits[tc.attackerUnit] != probability.wantPlayerLosses || report.DefenderLostUnits["weiInfantry"] != probability.wantEnemyLosses {
						t.Fatalf("expected exact %s NPC %s result, report=%+v", tc.name, probability.name, report)
					}
					if report.SurvivedUnits[tc.attackerUnit] != 200-probability.wantPlayerLosses || armySliceToMap(stored.Army)[tc.attackerUnit] != 200-probability.wantPlayerLosses || armySliceToMap(stored.NpcState.Cities[0].Army)["weiInfantry"] != 100-probability.wantEnemyLosses || report.GeneralExpGained != probability.wantEnemyLosses {
						t.Fatalf("expected %s NPC state and exp to match report, report=%+v stored=%+v", tc.name, report, stored)
					}
					if report.Detail == nil || !reportSideGeneralOwnsTrait(report.Detail.PrimarySide, tc.generalID, tc.specialTraitID) || !reportSideGeneralOwnsTrait(report.Detail.PrimarySide, tc.generalID, tc.bonusTraitID) {
						t.Fatalf("expected %s snapshot to preserve both traits, detail=%+v", tc.name, report.Detail)
					}
					if !standardReportHasTrait(report.Detail, tc.bonusTraitID) || standardReportHasTrait(report.Detail, tc.specialTraitID) != probability.wantSpecial {
						t.Fatalf("expected %s bonus always and special hit=%t, detail=%+v", tc.name, probability.wantSpecial, report.Detail)
					}
					if probability.wantSpecial {
						actual, ok := report.TraitOutcomes[tc.specialTraitID].Detail[tc.specialDetailKey].(map[string]int)
						if !ok || actual["weiInfantry"] != tc.specialActual {
							t.Fatalf("expected %s actual %s=%d, outcome=%+v", tc.name, tc.specialDetailKey, tc.specialActual, report.TraitOutcomes[tc.specialTraitID])
						}
					}
					wantTimeline := []string{tc.bonusTraitID}
					if probability.wantSpecial {
						wantTimeline = []string{tc.specialTraitID, tc.bonusTraitID}
					}
					if len(report.TraitTriggered) != len(wantTimeline) {
						t.Fatalf("expected %s timeline %v, got %v", tc.name, wantTimeline, report.TraitTriggered)
					}
					for index, traitID := range wantTimeline {
						if report.TraitTriggered[index] != traitID || report.Detail.Traits[index].TraitID != traitID {
							t.Fatalf("expected %s timeline %v, got legacy=%v standard=%+v", tc.name, wantTimeline, report.TraitTriggered, report.Detail.Traits)
						}
					}
				})
			}
		})
	}
}

// resolveNpcFormalRandomPreBattleHeroTest 构造携带完整正式双特性的 NPC 战前概率事务。
func resolveNpcFormalRandomPreBattleHeroTest(t *testing.T, suffix string, hero GeneralHeroConfig, attackerUnit string) (BattleReport, GameState) {
	t.Helper()
	setTestCombatUnitsConfig(t)
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: hero.ID, Name: hero.Name}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{hero.ID: hero}})
	now := time.Now().UTC()
	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)
	account := Account{ID: "account_npc_random_prebattle_" + suffix, Username: "npc_random_prebattle_" + suffix, CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create NPC random pre-battle account: %v", err)
	}
	state := newPlayerState("player_npc_random_prebattle_"+suffix, "NPC 随机战前测试", "wei", hero.ID, now)
	state.Army = []ArmyUnit{{UnitType: attackerUnit, Amount: 200}}
	npc := testNpcCity("npc_random_prebattle_"+suffix, now)
	npc.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	npc.MaxArmy = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	state.NpcState = &NpcState{Cities: []NpcCity{npc}, LastRefreshedAt: now.Format(resourceDateLayout)}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create NPC random pre-battle player: %v", err)
	}
	result, err := svc.AttackNpc(AttackNpcRequest{
		PlayerID: state.Player.ID, NpcID: npc.ID, Mode: "attack", Units: map[string]int{attackerUnit: 200}, GeneralIDs: []string{hero.ID},
	})
	if err != nil {
		t.Fatalf("AttackNpc random pre-battle %s failed: %v", suffix, err)
	}
	stored, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("get NPC random pre-battle state: %v", err)
	}
	return result.BattleReport, stored
}

// resolveNpcFormalPreBattleTraitTest 构造固定 200 对 100 的 NPC 战斗并强制触发指定正式战前特性。
func resolveNpcFormalPreBattleTraitTest(t *testing.T, traitID string, generalID string, generalName string, rate float64) (BattleReport, GameState) {
	t.Helper()
	setTestCombatUnitsConfig(t)
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: generalID, Name: generalName}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		generalID: {
			ID: generalID, Name: generalName, Faction: "wei", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: traitID, TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army",
				Params: map[string]float64{"effectRate": rate, "maxAffectedRate": rate, "triggerChance": 1},
			},
		},
	}})

	now := time.Now().UTC()
	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)
	account := Account{ID: "account_npc_prebattle_" + traitID, Username: "npc_prebattle_" + traitID, CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create NPC pre-battle account: %v", err)
	}
	state := newPlayerState("player_npc_prebattle_"+traitID, "NPC 战前特性测试", "wei", generalID, now)
	state.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 200}}
	npc := testNpcCity("npc_prebattle_"+traitID, now)
	npc.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	npc.MaxArmy = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	state.NpcState = &NpcState{Cities: []NpcCity{npc}, LastRefreshedAt: now.Format(resourceDateLayout)}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create NPC pre-battle player: %v", err)
	}

	result, err := svc.AttackNpc(AttackNpcRequest{
		PlayerID: state.Player.ID, NpcID: npc.ID, Mode: "plunder",
		Units: map[string]int{"weiInfantry": 200}, GeneralIDs: []string{generalID},
	})
	if err != nil {
		t.Fatalf("AttackNpc %s failed: %v", traitID, err)
	}
	stored, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("get NPC pre-battle state: %v", err)
	}
	return result.BattleReport, stored
}

// TestNpcSuppressionUsesFullBaselineForRevealThreshold 验证情报阈值使用包含压制兵的完整守军基数。
func TestNpcSuppressionUsesFullBaselineForRevealThreshold(t *testing.T) {
	now := time.Now().UTC()
	state := newPlayerState("player_suppress_visibility", "SuppressVisibility", "wei", "", now)
	npc := testNpcCity("npc_suppress_visibility", now)
	npc.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	report := applyNpcBattleResult(&state, &npc, combat.CombatResult{
		Winner:         "defender",
		AttackerLosses: []combat.UnitLoss{{ID: "weiInfantry", Count: 10, Losses: 10}},
		DefenderLosses: []combat.UnitLoss{{ID: "weiInfantry", Count: 50, Losses: 20}},
	}, []combat.Unit{{ID: "weiInfantry", Count: 10}}, map[string]int{"weiInfantry": 50}, nil, "plunder", now)

	if report.DefenderRevealed || len(report.DefenderUnits) != 0 || len(report.DefenderLostUnits) != 0 {
		t.Fatalf("expected 20/100 loss ratio to keep defender hidden, got revealed=%t before=%+v lost=%+v", report.DefenderRevealed, report.DefenderUnits, report.DefenderLostUnits)
	}
	if npc.Army[0].Amount != 80 {
		t.Fatalf("expected only 20 real losses and all suppressed troops preserved, got %+v", npc.Army)
	}
}

// TestPvpDefenderPreDamageMatchesReturnTroopsAndBothReports 验证防守关羽的水淹伤亡进入进攻方返程和双方战报。
func TestPvpDefenderPreDamageMatchesReturnTroopsAndBothReports(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "guanyu", Name: "关羽"}}},
	}, GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
			"guanyu": {
				ID: "guanyu", Name: "关羽", Faction: "shu", Enabled: true,
				SpecialTrait: GeneralTraitConfig{TraitID: "shuiyan_qijun", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army", Params: map[string]float64{"effectRate": 0.35, "triggerChance": 1}},
			},
		},
	})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wei", "caocao", "shu", "guanyu")
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1000}}
	defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 100}}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{"weiInfantry": 1000},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
	}
	attackerReport := attackerReports[0]
	outcome := attackerReport.TraitOutcomes["shuiyan_qijun"]
	preDamage, ok := outcome.Detail["preBattleAffected"].(map[string]int)
	if !ok || preDamage["weiInfantry"] != 350 {
		t.Fatalf("expected defender water attack to cause 350 pre-battle losses, got %+v", outcome)
	}
	if attackerReport.LostUnits["weiInfantry"] < 350 {
		t.Fatalf("expected attacker losses to include pre-damage, got %+v", attackerReport.LostUnits)
	}
	march, err := repo.GetPvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("GetPvpMarch failed: %v", err)
	}
	expectedReturned := 1000 - attackerReport.LostUnits["weiInfantry"]
	if march.AttackTroops["weiInfantry"] != expectedReturned || attackerReport.SurvivedUnits["weiInfantry"] != expectedReturned {
		t.Fatalf("expected returning troops %d to match report, march=%+v survived=%+v", expectedReturned, march.AttackTroops, attackerReport.SurvivedUnits)
	}
	defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
	}
	if defenderReports[0].DefenderLostUnits["weiInfantry"] != attackerReport.LostUnits["weiInfantry"] {
		t.Fatalf("expected both reports to agree on attacker losses, attacker=%+v defender=%+v", attackerReport.LostUnits, defenderReports[0].DefenderLostUnits)
	}
	found := false
	for _, trait := range attackerReport.Detail.Traits {
		if trait.TraitID == "shuiyan_qijun" {
			found = true
			if trait.OwnerSide != "secondary" {
				t.Fatalf("expected defender trait on secondary side, got %+v", trait)
			}
		}
	}
	if !found {
		t.Fatalf("expected water attack in standard report, got %+v", attackerReport.Detail.Traits)
	}
}

// TestPvpFormalPreDamageTraitsMatchBothReportsAndRealState 验证疑兵偷袭和水淹七军在攻守双方都造成真实伤亡并完成战报、兵力对账。
func TestPvpFormalPreDamageTraitsMatchBothReportsAndRealState(t *testing.T) {
	traits := []struct {
		traitID     string
		traitName   string
		generalID   string
		generalName string
		faction     string
	}{
		{traitID: "yibing_touxi", traitName: "疑兵偷袭", generalID: "simayi", generalName: "司马懿", faction: "wei"},
		{traitID: "shuiyan_qijun", traitName: "水淹七军", generalID: "guanyu", generalName: "关羽", faction: "shu"},
	}

	for _, traitCase := range traits {
		traitCase := traitCase
		for _, ownerSide := range []string{"attacker", "defender"} {
			ownerSide := ownerSide
			t.Run(traitCase.traitName+"_"+ownerSide, func(t *testing.T) {
				enemyFaction := "shu"
				enemyGeneralID := "liubei"
				enemyGeneralName := "刘备"
				if traitCase.faction == "shu" {
					enemyFaction = "wei"
					enemyGeneralID = "caocao"
					enemyGeneralName = "曹操"
				}
				trait := GeneralTraitConfig{
					TraitID: traitCase.traitID, TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army",
					Params: map[string]float64{"effectRate": 0.35, "maxAffectedRate": 0.35, "triggerChance": 1},
				}
				setTestFactionsAndGenerals(t, FactionsConfig{
					traitCase.faction: {Name: traitCase.faction, Generals: []GeneralInfo{{ID: traitCase.generalID, Name: traitCase.generalName}}},
					enemyFaction:      {Name: enemyFaction, Generals: []GeneralInfo{{ID: enemyGeneralID, Name: enemyGeneralName}}},
				}, GeneralsConfig{
					Enabled: true,
					Heroes: map[string]GeneralHeroConfig{
						traitCase.generalID: {ID: traitCase.generalID, Name: traitCase.generalName, Faction: traitCase.faction, Enabled: true, SpecialTrait: trait},
						enemyGeneralID:      {ID: enemyGeneralID, Name: enemyGeneralName, Faction: enemyFaction, Enabled: true},
					},
				})

				attackerFaction, attackerGeneralID := traitCase.faction, traitCase.generalID
				defenderFaction, defenderGeneralID := enemyFaction, enemyGeneralID
				if ownerSide == "defender" {
					attackerFaction, attackerGeneralID = enemyFaction, enemyGeneralID
					defenderFaction, defenderGeneralID = traitCase.faction, traitCase.generalID
				}
				svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, attackerFaction, attackerGeneralID, defenderFaction, defenderGeneralID)
				attackerAmount, defenderAmount := 100, 1000
				if ownerSide == "defender" {
					attackerAmount, defenderAmount = 1000, 100
				}
				attackerUnit := attackerFaction + "Infantry"
				defenderUnit := defenderFaction + "Infantry"
				attacker.Army = []ArmyUnit{{UnitType: attackerUnit, Amount: attackerAmount}}
				defender.Army = []ArmyUnit{{UnitType: defenderUnit, Amount: defenderAmount}}
				defender.Buildings = nil
				repo.players[attacker.Player.ID] = attacker
				repo.players[defender.Player.ID] = defender

				started, err := svc.StartPvpAttack(PvpAttackRequest{
					PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypePlunder,
					Troops: map[string]int{attackerUnit: attackerAmount}, GeneralIDs: []string{attackerGeneralID},
				})
				if err != nil {
					t.Fatalf("StartPvpAttack failed: %v", err)
				}
				forcePvpMarchDue(t, repo, started.March.ID)
				battle, err := svc.ResolvePvpMarch(started.March.ID)
				if err != nil {
					t.Fatalf("ResolvePvpMarch failed: %v", err)
				}

				targetSide := "defender"
				targetUnit := defenderUnit
				targetBefore := defenderAmount
				if ownerSide == "defender" {
					targetSide = "attacker"
					targetUnit = attackerUnit
					targetBefore = attackerAmount
				}
				targetLosses := pvpTestLossesFromBattle(t, battle, targetSide)[targetUnit]
				if targetLosses < 350 || targetLosses > targetBefore {
					t.Fatalf("expected final target losses to include exact 350 pre-battle losses, got %d/%d", targetLosses, targetBefore)
				}

				attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
				if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
					t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
				}
				defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
				if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
					t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
				}

				reports := []struct {
					report          BattleReport
					reportOwnerSide string
				}{
					{report: attackerReports[0], reportOwnerSide: "attacker"},
					{report: defenderReports[0], reportOwnerSide: "defender"},
				}
				for _, reportCase := range reports {
					outcome, ok := reportCase.report.TraitOutcomes[traitCase.traitID]
					preDamage, detailOK := outcome.Detail["preBattleAffected"].(map[string]int)
					maxRateValid := outcome.Detail["maxAffectedRate"] == 0.35
					if traitCase.traitID == "yibing_touxi" {
						_, maxRateExists := outcome.Detail["maxAffectedRate"]
						maxRateValid = !maxRateExists
					}
					if !ok || !detailOK || preDamage[targetUnit] != 350 || outcome.Detail["effectRate"] != 0.35 || !maxRateValid || outcome.Detail["triggerChance"] != float64(1) || outcome.OwnerSide != ownerSide || outcome.OwnerGeneralID != traitCase.generalID {
						t.Fatalf("expected exact formal pre-damage outcome in both reports, got %+v", outcome)
					}
					legacyLosses := reportCase.report.DefenderLostUnits
					if reportCase.reportOwnerSide == targetSide {
						legacyLosses = reportCase.report.LostUnits
					}
					if legacyLosses[targetUnit] != targetLosses {
						t.Fatalf("expected legacy report target loss %d, got %+v", targetLosses, legacyLosses)
					}
					if reportCase.report.Detail == nil || reportCase.report.Detail.SecondarySide == nil {
						t.Fatalf("expected complete standard report detail, got %+v", reportCase.report.Detail)
					}
					standardSides := []BattleReportSide{reportCase.report.Detail.PrimarySide, *reportCase.report.Detail.SecondarySide}
					standardMatched := false
					for _, side := range standardSides {
						if side.Role != targetSide {
							continue
						}
						for _, unit := range side.Units {
							if unit.UnitType == targetUnit && unit.AmountBefore == targetBefore && unit.Lost == targetLosses && unit.Survived == targetBefore-targetLosses {
								standardMatched = true
							}
						}
					}
					if !standardMatched {
						t.Fatalf("expected standard report to reconcile target %s before/lost/survived=%d/%d/%d, detail=%+v", targetUnit, targetBefore, targetLosses, targetBefore-targetLosses, reportCase.report.Detail)
					}
					expectedDisplaySide := "secondary"
					if ownerSide == "attacker" {
						expectedDisplaySide = "primary"
					}
					traitMatched := false
					for _, reportTrait := range reportCase.report.Detail.Traits {
						if reportTrait.TraitID == traitCase.traitID && reportTrait.OwnerSide == expectedDisplaySide && reportTrait.OwnerRole == ownerSide {
							traitMatched = true
						}
					}
					if !traitMatched {
						t.Fatalf("expected standard trait owner side %s, traits=%+v", expectedDisplaySide, reportCase.report.Detail.Traits)
					}
				}

				if targetSide == "attacker" {
					storedMarch, err := repo.GetPvpMarch(started.March.ID)
					if err != nil || storedMarch.AttackTroops[targetUnit] != targetBefore-targetLosses {
						t.Fatalf("expected attacker march state %d, march=%+v err=%v", targetBefore-targetLosses, storedMarch, err)
					}
				} else {
					storedDefender, err := repo.GetState(defender.Player.ID)
					if err != nil || armySliceToMap(storedDefender.Army)[targetUnit] != targetBefore-targetLosses {
						t.Fatalf("expected defender army state %d, state=%+v err=%v", targetBefore-targetLosses, storedDefender.Army, err)
					}
				}
			})
		}
	}
}

// TestPvpFormalSuppressionTraitsMatchBothReportsAndPreserveState 验证三项临时压制按各自合法方向减少本场参战兵力且不形成真实伤亡。
func TestPvpFormalSuppressionTraitsMatchBothReportsAndPreserveState(t *testing.T) {
	traits := []struct {
		traitID     string
		traitName   string
		generalID   string
		generalName string
		faction     string
		rate        float64
		ownerSides  []string
		noMaxField  bool
	}{
		{traitID: "weizhen_zhenhe", traitName: "震慑全军", generalID: "zhangliao", generalName: "张辽", faction: "wei", rate: 0.25, ownerSides: []string{"attacker"}, noMaxField: true},
		{traitID: "zhenhe_quanjun", traitName: "震慑全军", generalID: "zhangfei", generalName: "张飞", faction: "shu", rate: 0.5},
		{traitID: "qimen_dunjia", traitName: "奇门遁甲", generalID: "zhugeliang", generalName: "诸葛亮", faction: "shu", rate: 0.25},
	}

	for _, traitCase := range traits {
		traitCase := traitCase
		ownerSides := traitCase.ownerSides
		if len(ownerSides) == 0 {
			ownerSides = []string{"attacker", "defender"}
		}
		for _, ownerSide := range ownerSides {
			ownerSide := ownerSide
			t.Run(traitCase.traitName+"_"+ownerSide, func(t *testing.T) {
				enemyFaction := "shu"
				enemyGeneralID := "liubei"
				enemyGeneralName := "刘备"
				if traitCase.faction == "shu" {
					enemyFaction = "wei"
					enemyGeneralID = "caocao"
					enemyGeneralName = "曹操"
				}
				trait := GeneralTraitConfig{
					TraitID: traitCase.traitID, TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army",
					Params: map[string]float64{"effectRate": traitCase.rate, "maxAffectedRate": traitCase.rate, "triggerChance": 1},
				}
				if traitCase.noMaxField {
					delete(trait.Params, "maxAffectedRate")
					trait.AllowedSides = []string{"attacker"}
				}
				setTestFactionsAndGenerals(t, FactionsConfig{
					traitCase.faction: {Name: traitCase.faction, Generals: []GeneralInfo{{ID: traitCase.generalID, Name: traitCase.generalName}}},
					enemyFaction:      {Name: enemyFaction, Generals: []GeneralInfo{{ID: enemyGeneralID, Name: enemyGeneralName}}},
				}, GeneralsConfig{
					Enabled: true,
					Heroes: map[string]GeneralHeroConfig{
						traitCase.generalID: {ID: traitCase.generalID, Name: traitCase.generalName, Faction: traitCase.faction, Enabled: true, SpecialTrait: trait},
						enemyGeneralID:      {ID: enemyGeneralID, Name: enemyGeneralName, Faction: enemyFaction, Enabled: true},
					},
				})

				attackerFaction, attackerGeneralID := traitCase.faction, traitCase.generalID
				defenderFaction, defenderGeneralID := enemyFaction, enemyGeneralID
				if ownerSide == "defender" {
					attackerFaction, attackerGeneralID = enemyFaction, enemyGeneralID
					defenderFaction, defenderGeneralID = traitCase.faction, traitCase.generalID
				}
				svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, attackerFaction, attackerGeneralID, defenderFaction, defenderGeneralID)
				attackerAmount, defenderAmount := 100, 1000
				if ownerSide == "defender" {
					attackerAmount, defenderAmount = 1000, 100
				}
				attackerUnit := attackerFaction + "Infantry"
				defenderUnit := defenderFaction + "Infantry"
				attacker.Army = []ArmyUnit{{UnitType: attackerUnit, Amount: attackerAmount}}
				defender.Army = []ArmyUnit{{UnitType: defenderUnit, Amount: defenderAmount}}
				defender.Buildings = nil
				repo.players[attacker.Player.ID] = attacker
				repo.players[defender.Player.ID] = defender

				started, err := svc.StartPvpAttack(PvpAttackRequest{
					PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypePlunder,
					Troops: map[string]int{attackerUnit: attackerAmount}, GeneralIDs: []string{attackerGeneralID},
				})
				if err != nil {
					t.Fatalf("StartPvpAttack failed: %v", err)
				}
				forcePvpMarchDue(t, repo, started.March.ID)
				battle, err := svc.ResolvePvpMarch(started.March.ID)
				if err != nil {
					t.Fatalf("ResolvePvpMarch failed: %v", err)
				}

				targetSide, targetUnit, targetBefore := "defender", defenderUnit, defenderAmount
				if ownerSide == "defender" {
					targetSide, targetUnit, targetBefore = "attacker", attackerUnit, attackerAmount
				}
				expectedSuppressed := int(float64(targetBefore) * traitCase.rate)
				targetLosses := pvpTestLossesFromBattle(t, battle, targetSide)[targetUnit]
				if targetLosses > targetBefore-expectedSuppressed {
					t.Fatalf("expected at most %d participating troops to die, got %d", targetBefore-expectedSuppressed, targetLosses)
				}

				attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
				if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
					t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
				}
				defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
				if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
					t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
				}
				for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
					outcome, ok := report.TraitOutcomes[traitCase.traitID]
					suppressed, detailOK := outcome.Detail["suppressedUnits"].(map[string]int)
					maxRateValid := outcome.Detail["maxAffectedRate"] == traitCase.rate
					if traitCase.noMaxField {
						_, maxRateExists := outcome.Detail["maxAffectedRate"]
						maxRateValid = !maxRateExists
					}
					if !ok || !detailOK || suppressed[targetUnit] != expectedSuppressed || outcome.Detail["effectRate"] != traitCase.rate || !maxRateValid || outcome.Detail["triggerChance"] != float64(1) || outcome.OwnerSide != ownerSide || outcome.OwnerGeneralID != traitCase.generalID {
						t.Fatalf("expected exact formal suppression outcome in both reports, got %+v", outcome)
					}
					if traitCase.traitID == "weizhen_zhenhe" {
						fled, fledOK := outcome.Detail["fledUnits"].(map[string]int)
						returned, returnedOK := outcome.Detail["returnedUnits"].(map[string]int)
						if !fledOK || !returnedOK || fled[targetUnit] != expectedSuppressed || returned[targetUnit] != expectedSuppressed {
							t.Fatalf("expected Zhang Liao exact flee and return values, outcome=%+v", outcome)
						}
					}
					if report.Detail == nil {
						t.Fatalf("expected standard report detail, got %+v", report)
					}
					standardMatched := false
					for _, reportTrait := range report.Detail.Traits {
						standardSuppressed, standardDetailOK := reportTrait.Detail["suppressedUnits"].(map[string]int)
						standardMaxValid := reportTrait.Detail["maxAffectedRate"] == traitCase.rate
						if traitCase.noMaxField {
							_, standardMaxExists := reportTrait.Detail["maxAffectedRate"]
							standardMaxValid = !standardMaxExists
						}
						if reportTrait.TraitID == traitCase.traitID && standardDetailOK && standardSuppressed[targetUnit] == expectedSuppressed && reportTrait.Detail["effectRate"] == traitCase.rate && standardMaxValid {
							standardMatched = true
						}
					}
					if !standardMatched {
						t.Fatalf("expected standard report to preserve suppression design and actual values, traits=%+v", report.Detail.Traits)
					}
				}

				if targetSide == "attacker" {
					storedMarch, err := repo.GetPvpMarch(started.March.ID)
					if err != nil || storedMarch.AttackTroops[targetUnit] != targetBefore-targetLosses {
						t.Fatalf("expected all suppressed survivors to return, march=%+v err=%v", storedMarch, err)
					}
				} else {
					storedDefender, err := repo.GetState(defender.Player.ID)
					if err != nil || armySliceToMap(storedDefender.Army)[targetUnit] != targetBefore-targetLosses {
						t.Fatalf("expected all suppressed survivors to remain, state=%+v err=%v", storedDefender.Army, err)
					}
				}
			})
		}
	}
}

// TestYellowTurbanDefenderPreDamageMatchesEnemyLossReport 验证黄巾守城水淹会计入来袭军真实战损口径。
func TestYellowTurbanDefenderPreDamageMatchesEnemyLossReport(t *testing.T) {
	setTestCombatUnitsConfig(t)
	setTestFactionsAndGenerals(t, FactionsConfig{
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "guanyu", Name: "关羽"}}},
	}, GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"guanyu": {
				ID: "guanyu", Name: "关羽", Faction: "shu", Enabled: true,
				SpecialTrait: GeneralTraitConfig{TraitID: "shuiyan_qijun", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army", Params: map[string]float64{"effectRate": 0.35, "triggerChance": 1}},
			},
		},
	})
	SetYellowTurbanConfig(defaultYellowTurbanConfig())
	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)
	now := time.Now().UTC()
	account := Account{ID: "account_yt_pre_damage", Username: "yt_pre_damage", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}
	state := newPlayerState("player_yt_pre_damage", "关羽守城", "shu", "guanyu", now)
	EnsureGeneralRoster(&state, now)
	state.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("CreatePlayer failed: %v", err)
	}
	march, err := repo.CreateYellowTurbanMarch(YellowTurbanMarch{
		ID: "yt_pre_damage_march", TargetPlayerID: state.Player.ID,
		SourceCityID: "yt_source", SourceName: "黄巾来袭", SourceFaction: "wei", SourceRegionID: "wei",
		RiskLevelID: 1, RiskLevelName: "黄巾测试", PlayerFood: 10000, FoodCapacity: 1000, Pressure: 10,
		Troops: map[string]int{"weiInfantry": 1000}, Status: YellowTurbanMarchStatusMarching, DurationSeconds: 1,
		StartedAt: now.Add(-2 * time.Minute).Format(resourceDateLayout), ArrivesAt: now.Add(-time.Minute).Format(resourceDateLayout),
		CreatedAt: now.Add(-2 * time.Minute).Format(resourceDateLayout), UpdatedAt: now.Add(-2 * time.Minute).Format(resourceDateLayout),
	})
	if err != nil {
		t.Fatalf("CreateYellowTurbanMarch failed: %v", err)
	}

	report, err := svc.ResolveYellowTurbanMarch(march.ID)
	if err != nil {
		t.Fatalf("ResolveYellowTurbanMarch failed: %v", err)
	}
	outcome := report.TraitOutcomes["shuiyan_qijun"]
	preDamage, ok := outcome.Detail["preBattleAffected"].(map[string]int)
	if !ok || preDamage["weiInfantry"] != 350 {
		t.Fatalf("expected yellow turban attackers to take 350 pre-battle losses, got %+v", outcome)
	}
	if report.DefenderLostUnits["weiInfantry"] < 350 || report.DefenderLostUnits["weiInfantry"] > 1000 {
		t.Fatalf("expected enemy losses to include bounded pre-damage, got %+v", report.DefenderLostUnits)
	}
	if report.Detail == nil || report.Detail.PrimarySide.Units == nil {
		t.Fatalf("expected standard yellow turban attacker detail, got %+v", report.Detail)
	}
	for _, unit := range report.Detail.PrimarySide.Units {
		if unit.UnitType == "weiInfantry" && (unit.AmountBefore != 1000 || unit.Lost != report.DefenderLostUnits["weiInfantry"] || unit.Survived != 1000-report.DefenderLostUnits["weiInfantry"]) {
			t.Fatalf("expected standard enemy losses to match raw report, got %+v", unit)
		}
	}
}

// TestPvpPreDamageAllocatesAcrossDefenderAndReinforcement 验证战前伤亡在主城守军和驻防援军间按来源真实分配。
func TestPvpPreDamageAllocatesAcrossDefenderAndReinforcement(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "simayi", Name: "司马懿"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
	}, GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"simayi": {
				ID: "simayi", Name: "司马懿", Faction: "wei", Enabled: true,
				SpecialTrait: GeneralTraitConfig{TraitID: "yibing_touxi", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army", Params: map[string]float64{"effectRate": 0.5, "triggerChance": 1}},
			},
			"liubei": {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true},
		},
	})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wei", "simayi", "shu", "liubei")
	now := time.Now().UTC()
	helperAccount := Account{ID: "account_pre_damage_helper", Username: "pre_damage_helper", PasswordHash: "hash", CreatedAt: now}
	if err := repo.CreateAccount(helperAccount); err != nil {
		t.Fatalf("CreateAccount helper failed: %v", err)
	}
	helper := newPlayerState("player_pre_damage_helper", "援军方", "wu", "", now)
	if err := repo.CreatePlayer(helperAccount.ID, helper, now); err != nil {
		t.Fatalf("CreatePlayer helper failed: %v", err)
	}
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 200}}
	defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 100}}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender
	reinforcement := Reinforcement{
		ID: "rein_pre_damage", FromPlayerID: helper.Player.ID, FromPlayerName: helper.Player.Nickname, FromPlayerFaction: "wu",
		ToPlayerID: defender.Player.ID, ToPlayerName: defender.Player.Nickname, ToPlayerFaction: defender.Player.Faction,
		OwnerPlayerID: helper.Player.ID, HostPlayerID: defender.Player.ID, SourceType: GarrisonSourceReinforcement,
		SourceID: "rein_pre_damage", TargetType: ReinforcementTargetPlayerCity, TargetID: defender.Player.ID,
		Status: ReinforcementStatusStationed, Troops: map[string]int{"shuInfantry": 100}, RemainingTroops: map[string]int{"shuInfantry": 100}, Losses: map[string]int{},
		Rules: defaultGarrisonRules(GarrisonSourceReinforcement), SentAt: now.Add(-2 * time.Hour).Format(resourceDateLayout),
		ArrivedAt: now.Add(-time.Hour).Format(resourceDateLayout), CreatedAt: now.Add(-2 * time.Hour).Format(resourceDateLayout), UpdatedAt: now.Add(-time.Hour).Format(resourceDateLayout),
	}
	repo.reinforcements[reinforcement.ID] = reinforcement

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypePlunder,
		Troops: map[string]int{"weiInfantry": 200}, GeneralIDs: []string{"simayi"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
	}
	attackerReport := attackerReports[0]
	preDamage, ok := attackerReport.TraitOutcomes["yibing_touxi"].Detail["preBattleAffected"].(map[string]int)
	if !ok || preDamage["shuInfantry"] != 100 {
		t.Fatalf("expected 100 aggregate pre-battle losses, got %+v", attackerReport.TraitOutcomes["yibing_touxi"])
	}
	defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
	}
	mainLoss := defenderReports[0].LostUnits["shuInfantry"]
	reinforcementLoss := defenderReports[0].PvpReinforcementLosses[reinforcement.ID]["shuInfantry"]
	totalLoss := mainLoss + reinforcementLoss
	if mainLoss <= 0 || reinforcementLoss <= 0 || totalLoss <= preDamage["shuInfantry"] || totalLoss > 200 {
		t.Fatalf("expected bounded losses including 100 pre-damage split across main and reinforcement, main=%d reinforcement=%d total=%d", mainLoss, reinforcementLoss, totalLoss)
	}
	if battleMainLoss := pvpTestLossesFromBattle(t, battle, "defender")["shuInfantry"]; battleMainLoss != mainLoss {
		t.Fatalf("expected battle main loss %d to match defender report, got %d", mainLoss, battleMainLoss)
	}
	storedDefender, err := repo.GetState(defender.Player.ID)
	if err != nil {
		t.Fatalf("GetState defender failed: %v", err)
	}
	if got := armySliceToMap(storedDefender.Army)["shuInfantry"]; got != 100-mainLoss {
		t.Fatalf("expected real main army %d, got %d", 100-mainLoss, got)
	}
	storedReinforcement, err := repo.GetReinforcement(reinforcement.ID)
	if err != nil {
		t.Fatalf("GetReinforcement failed: %v", err)
	}
	if storedReinforcement.RemainingTroops["shuInfantry"] != 100-reinforcementLoss {
		t.Fatalf("expected real reinforcement %d, got %+v", 100-reinforcementLoss, storedReinforcement.RemainingTroops)
	}
	helperReports, total, err := repo.ListReportsByQuery(BattleReportQuery{PlayerID: helper.Player.ID, ViewType: ReportViewReinforcement, Page: 1, PageSize: 10})
	if err != nil || total != 1 || len(helperReports) != 1 || helperReports[0].LostUnits["shuInfantry"] != reinforcementLoss {
		t.Fatalf("expected independent reinforcement report loss %d, reports=%+v total=%d err=%v", reinforcementLoss, helperReports, total, err)
	}
}
