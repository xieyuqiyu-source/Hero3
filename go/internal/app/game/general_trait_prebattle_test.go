// 本文件验证战前真实伤亡与临时压制在服务结算、真实兵力和战报中的不同口径。
package game

import (
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
		{name: "水淹七军", traitID: "shuiyan_qijun", generalID: "guanyu", generalName: "关羽", detailKey: "preBattleAffected", rate: 0.3},
		{name: "震慑全军", traitID: "weizhen_zhenhe", generalID: "zhangliao", generalName: "张辽", detailKey: "suppressedUnits", rate: 0.25},
		{name: "万人怒吼", traitID: "zhenhe_quanjun", generalID: "zhangfei", generalName: "张飞", detailKey: "suppressedUnits", rate: 0.5},
		{name: "奇门遁甲", traitID: "qimen_dunjia", generalID: "zhugeliang", generalName: "诸葛亮", detailKey: "suppressedUnits", rate: 0.25},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			report, stored := resolveNpcFormalPreBattleTraitTest(t, tc.traitID, tc.generalID, tc.generalName, tc.rate)
			wantAffected := int(100 * tc.rate)
			outcome, ok := report.TraitOutcomes[tc.traitID]
			affected, detailOK := outcome.Detail[tc.detailKey].(map[string]int)
			maxRateValid := outcome.Detail["maxAffectedRate"] == tc.rate
			if tc.traitID == "yibing_touxi" || tc.traitID == "shuiyan_qijun" || tc.traitID == "weizhen_zhenhe" || tc.traitID == "zhenhe_quanjun" || tc.traitID == "qimen_dunjia" {
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
