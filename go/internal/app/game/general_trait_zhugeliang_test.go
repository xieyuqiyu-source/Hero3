// 本文件验证诸葛亮战前困兵、全体触发特性封禁和双方诸葛失效边界。
package game

import (
	"reflect"
	"testing"

	"hero3/internal/core/general"
)

// TestZhugeLiangTraitConfigMigrationAndSchema 验证旧字段会迁移且 GM 只暴露当前有效参数。
func TestZhugeLiangTraitConfigMigrationAndSchema(t *testing.T) {
	cfg := NormalizeGeneralsConfig(GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"zhugeliang": {
			ID: "zhugeliang", Name: "诸葛亮", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "qimen_dunjia", TraitType: general.TraitTypeSpecial, Enabled: true,
				Params: map[string]float64{"effectRate": 0.25, "maxAffectedRate": 0.25, "maxAffectedCount": 1000},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: wolongTraitID, TraitType: general.TraitTypeBonus, Enabled: true,
				Params: map[string]float64{"disableTraitCount": 1},
			},
		},
	}})
	hero := cfg.Heroes["zhugeliang"]
	if !reflect.DeepEqual(hero.SpecialTrait.Params, map[string]float64{"effectRate": 0.25, "triggerChance": 1}) ||
		!reflect.DeepEqual(hero.BonusTrait.Params, map[string]float64{"triggerChance": 0.6}) {
		t.Fatalf("expected only current Zhuge Liang parameters after migration, hero=%+v", hero)
	}
	wantSides := []string{"attacker", "defender", "reinforcement"}
	if !reflect.DeepEqual(hero.SpecialTrait.AllowedSides, wantSides) || !reflect.DeepEqual(hero.BonusTrait.AllowedSides, wantSides) {
		t.Fatalf("expected both traits available in attack, defense and reinforcement, hero=%+v", hero)
	}

	schemaKeys := func(traitID string) []string {
		trait, ok := general.Get(traitID)
		if !ok {
			t.Fatalf("trait %s not registered", traitID)
		}
		keys := make([]string, 0, len(trait.ParamSchema()))
		for _, field := range trait.ParamSchema() {
			keys = append(keys, field.Key)
		}
		return keys
	}
	if got := schemaKeys("qimen_dunjia"); !reflect.DeepEqual(got, []string{"effectRate", "triggerChance"}) {
		t.Fatalf("expected Qimen GM schema to expose rate and chance only, keys=%+v", got)
	}
	if got := schemaKeys(wolongTraitID); !reflect.DeepEqual(got, []string{"triggerChance"}) {
		t.Fatalf("expected Wolong GM schema to expose chance only, keys=%+v", got)
	}
}

// TestWolongControlDisablesAllEnemyCombatTraitsButKeepsPassives 验证卧龙奇谋只清除敌方触发型特性。
func TestWolongControlDisablesAllEnemyCombatTraitsButKeepsPassives(t *testing.T) {
	attacker := []general.ActiveTrait{
		{TraitID: "qimen_dunjia", TraitType: general.TraitTypeSpecial, OwnerPlayerID: "attacker", OwnerGeneralID: "zhugeliang", Params: general.Params{"effectRate": 0.25, "triggerChance": 1}},
		{TraitID: wolongTraitID, TraitType: general.TraitTypeBonus, OwnerPlayerID: "attacker", OwnerGeneralID: "zhugeliang", Params: general.Params{"triggerChance": 1}},
	}
	defender := []general.ActiveTrait{
		{TraitID: "xiliang_tuji", TraitType: general.TraitTypeSpecial, OwnerPlayerID: "defender", OwnerGeneralID: "machao", Params: general.Params{"effectRate": 0.12, "triggerChance": 1}},
		{TraitID: "tianshen_xiafan", TraitType: general.TraitTypeBonus, OwnerPlayerID: "defender", OwnerGeneralID: "machao", Params: general.Params{"forceBonus": 20}},
	}

	resolution := resolveBattleTraitControl(attacker, defender, nil, PvpMarchTypeAttack)
	if !resolution.DisabledSides["defender"] || resolution.DisabledSides["attacker"] {
		t.Fatalf("expected only defender combat traits disabled, resolution=%+v", resolution)
	}
	outcome := resolution.MainOutcomes[wolongTraitID]
	if outcome.Name != "卧龙奇谋" || outcome.Detail["disabledGeneralCount"] != 1 || outcome.Detail["disabledTraitCount"] != 1 {
		t.Fatalf("expected one enemy general and one trigger trait disabled, outcome=%+v", outcome)
	}
	filtered := filterDisabledCombatTraits(defender, true)
	if len(filtered) != 1 || filtered[0].TraitID != "tianshen_xiafan" {
		t.Fatalf("expected permanent passive retained while combat trait is removed, traits=%+v", filtered)
	}
}

// TestWolongControlMissLeavesEnemyTraitsEnabled 验证 GM 概率为零时不产生封禁和战报结果。
func TestWolongControlMissLeavesEnemyTraitsEnabled(t *testing.T) {
	attacker := []general.ActiveTrait{{
		TraitID: wolongTraitID, TraitType: general.TraitTypeBonus,
		OwnerPlayerID: "attacker", OwnerGeneralID: "zhugeliang",
		Params: general.Params{"triggerChance": 0},
	}}
	defender := []general.ActiveTrait{{
		TraitID: "renzhu_shouhu", TraitType: general.TraitTypeBonus,
		OwnerPlayerID: "defender", OwnerGeneralID: "liubei",
		Params: general.Params{"effectRate": 0.35, "triggerChance": 1},
	}}

	resolution := resolveBattleTraitControl(attacker, defender, nil, PvpMarchTypeAttack)
	if len(resolution.DisabledSides) != 0 || len(resolution.MainOutcomes) != 0 {
		t.Fatalf("expected zero chance Wolong to leave battle unchanged, resolution=%+v", resolution)
	}
}

// TestMutualWolongControlsBothBecomeInvalid 验证双方诸葛亮时卧龙奇谋均失效且不封禁其他特性。
func TestMutualWolongControlsBothBecomeInvalid(t *testing.T) {
	attacker := []general.ActiveTrait{{
		TraitID: wolongTraitID, TraitType: general.TraitTypeBonus,
		OwnerPlayerID: "attacker", OwnerGeneralID: "zhugeliang",
		Params: general.Params{"triggerChance": 1},
	}}
	defender := []general.ActiveTrait{{
		TraitID: wolongTraitID, TraitType: general.TraitTypeBonus,
		OwnerPlayerID: "defender", OwnerGeneralID: "zhugeliang",
		Params: general.Params{"triggerChance": 1},
	}}

	resolution := resolveBattleTraitControl(attacker, defender, nil, PvpMarchTypeAttack)
	if len(resolution.DisabledSides) != 0 || len(resolution.MainOutcomes) != 2 {
		t.Fatalf("expected two invalid hints and no disabled side, resolution=%+v", resolution)
	}
	seenOwners := map[string]bool{}
	for _, outcome := range resolution.MainOutcomes {
		if outcome.Name != "卧龙奇谋" || outcome.Detail["status"] != "特性已失效" || outcome.Detail["invalidReason"] != "双方均有诸葛亮" {
			t.Fatalf("expected explicit mutual invalid hint, outcome=%+v", outcome)
		}
		seenOwners[outcome.OwnerPlayerID] = true
	}
	if !seenOwners["attacker"] || !seenOwners["defender"] {
		t.Fatalf("expected both Zhuge Liang owners in invalid outcomes, owners=%+v", seenOwners)
	}
}

// TestReinforcementWolongCountsAllAttackingGenerals 验证援军诸葛亮封禁进攻方全部参战将领的触发型特性。
func TestReinforcementWolongCountsAllAttackingGenerals(t *testing.T) {
	attacker := []general.ActiveTrait{
		{TraitID: "meiren", TraitType: general.TraitTypeSpecial, OwnerPlayerID: "attacker", OwnerGeneralID: "zhenmi"},
		{TraitID: "meihuo_raozhen", TraitType: general.TraitTypeBonus, OwnerPlayerID: "attacker", OwnerGeneralID: "zhenmi"},
	}
	reinforcement := Reinforcement{
		ID: "reinforcement", FromPlayerID: "helper", Status: ReinforcementStatusStationed,
		Rules: GarrisonRules{CanFight: true},
		Generals: []ReinforcementGeneralSnapshot{{
			ID: "zhugeliang",
			Traits: []GeneralTraitInstance{{
				TraitID: wolongTraitID, TraitType: general.TraitTypeBonus, Name: "卧龙奇谋",
				Scope: "enemy_traits", Params: map[string]float64{"triggerChance": 1},
			}},
		}},
	}

	resolution := resolveBattleTraitControl(attacker, nil, []Reinforcement{reinforcement}, PvpMarchTypeAttack)
	if !resolution.DisabledSides["attacker"] || len(resolution.ReinforcementOutcomes["reinforcement"]) != 1 {
		t.Fatalf("expected reinforcement Wolong to disable attacker, resolution=%+v", resolution)
	}
	outcome := resolution.ReinforcementOutcomes["reinforcement"][wolongTraitID]
	if outcome.Detail["disabledGeneralCount"] != 1 || outcome.Detail["disabledTraitCount"] != 2 {
		t.Fatalf("expected all two trigger traits of the attacking general disabled, outcome=%+v", outcome)
	}
}

// TestPvpMutualZhugeLiangReportsBothWolongInvalidAndKeepsQimen 验证真实 PVP 双方卧龙失效但奇门仍各自生效。
func TestPvpMutualZhugeLiangReportsBothWolongInvalidAndKeepsQimen(t *testing.T) {
	hero := GeneralHeroConfig{
		ID: "zhugeliang", Name: "诸葛亮", Faction: "shu", Enabled: true,
		SpecialTrait: GeneralTraitConfig{
			TraitID: "qimen_dunjia", TraitType: general.TraitTypeSpecial, Enabled: true,
			Scope: "enemy_army", AllowedSides: []string{"attacker", "defender", "reinforcement"},
			Params: map[string]float64{"effectRate": 0.25, "triggerChance": 1},
		},
		BonusTrait: GeneralTraitConfig{
			TraitID: wolongTraitID, TraitType: general.TraitTypeBonus, Enabled: true,
			Scope: "enemy_traits", AllowedSides: []string{"attacker", "defender", "reinforcement"},
			Params: map[string]float64{"triggerChance": 1},
		},
	}
	setTestFactionsAndGenerals(t, FactionsConfig{
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "zhugeliang", Name: "诸葛亮"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{"zhugeliang": hero}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "shu", "zhugeliang", "shu", "zhugeliang")
	attacker.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 100}}
	defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 100}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{"shuInfantry": 100}, GeneralIDs: []string{"zhugeliang"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	if battle.Result["attackerPower"] != float64(750) || battle.Result["defensePower"] != float64(750) {
		t.Fatalf("expected both Qimen traits to keep working at 750/750, result=%+v", battle.Result)
	}

	attackerReport, err := repo.GetReportForPlayer(attacker.Player.ID, battle.AttackerReportID)
	if err != nil {
		t.Fatalf("GetReportForPlayer attacker failed: %v", err)
	}
	defenderReport, err := repo.GetReportForPlayer(defender.Player.ID, battle.DefenderReportID)
	if err != nil {
		t.Fatalf("GetReportForPlayer defender failed: %v", err)
	}
	for _, report := range []BattleReport{attackerReport, defenderReport} {
		invalidOwners := map[string]bool{}
		qimenOwners := map[string]bool{}
		for _, outcome := range report.TraitOutcomes {
			switch outcome.TraitID {
			case wolongTraitID:
				if outcome.Detail["status"] != "特性已失效" || outcome.Detail["invalidReason"] != "双方均有诸葛亮" {
					t.Fatalf("expected explicit mutual Wolong invalid reason, report=%s outcome=%+v", report.ID, outcome)
				}
				invalidOwners[outcome.OwnerPlayerID] = true
			case "qimen_dunjia":
				suppressed, ok := outcome.Detail["suppressedUnits"].(map[string]int)
				if !ok || suppressed["shuInfantry"] != 25 {
					t.Fatalf("expected each Qimen to suppress 25 troops, report=%s outcome=%+v", report.ID, outcome)
				}
				qimenOwners[outcome.OwnerPlayerID] = true
			}
		}
		if len(invalidOwners) != 2 || len(qimenOwners) != 2 {
			t.Fatalf("expected two Wolong invalid hints and two Qimen outcomes, report=%s outcomes=%+v", report.ID, report.TraitOutcomes)
		}
		if report.Detail == nil || len(report.Detail.Traits) != 4 {
			t.Fatalf("expected standard timeline to preserve all four owned results, report=%s detail=%+v", report.ID, report.Detail)
		}
	}
}
