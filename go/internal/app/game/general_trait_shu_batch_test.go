// 本文件集中验收关羽、张飞、赵云、黄忠和魏延本批新特性的核心结果与战报数据。
package game

import (
	"math"
	"reflect"
	"strconv"
	"testing"

	"hero3/internal/core/combat"
	"hero3/internal/core/general"
)

// TestShuBatchPreBattleTraitsUseExactTroopTotals 验证真实伤亡与溃逃都按敌方全军总数计算。
func TestShuBatchPreBattleTraitsUseExactTroopTotals(t *testing.T) {
	tests := []struct {
		name          string
		traitID       string
		rate          float64
		wantRemaining int
		detailKey     string
	}{
		{name: "关羽水淹七军", traitID: "shuiyan_qijun", rate: 0.3, wantRemaining: 140, detailKey: "realCasualties"},
		{name: "张飞万人怒吼", traitID: "zhenhe_quanjun", rate: 0.5, wantRemaining: 100, detailKey: "fledUnits"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			attacker := combat.Army{Units: []combat.Unit{{ID: "azureDragon", Category: "infantry", Count: 100, Attack: 100}}}
			defender := combat.Army{Units: []combat.Unit{
				{ID: "weiInfantry", Category: "infantry", Count: 101, InfantryDefense: 100, CavalryDefense: 80},
				{ID: "weiCavalry", Category: "cavalry", Count: 99, InfantryDefense: 80, CavalryDefense: 100},
			}}
			ctx := &general.BeforeBattleContext{Attacker: &attacker, Defender: &defender, IsPvP: true, Scene: "attack"}
			general.Dispatch(ctx, []general.ActiveTrait{{
				TraitID: tc.traitID, OwnerSide: "attacker", OwnerGeneralID: "shu_general",
				Scope: "enemy_army", AllowedSides: []string{"attacker"},
				Params: general.Params{"triggerChance": 1, "effectRate": tc.rate},
			}})

			remaining := 0
			for _, unit := range defender.Units {
				remaining += unit.Count
			}
			if remaining != tc.wantRemaining {
				t.Fatalf("expected %d troops remaining, got %d: %+v", tc.wantRemaining, remaining, defender.Units)
			}
			outcome := ctx.Triggered[tc.traitID]
			if outcome.Detail[tc.detailKey] == nil {
				t.Fatalf("expected %s in authoritative outcome, got %+v", tc.detailKey, outcome)
			}
		})
	}
}

// TestPvpLongdanRescueMergesDefenseAndResourceProtectionIntoReports 验证真实掠夺事务同时结算双防、保资源和战报。
func TestPvpLongdanRescueMergesDefenseAndResourceProtectionIntoReports(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "zhaoyun", Name: "赵云"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
		"zhaoyun": {
			ID: "zhaoyun", Name: "赵云", Faction: "shu", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "longdan_jiuyuan", TraitType: general.TraitTypeSpecial, Enabled: true,
				Scope: "self_army", TargetUnitType: "qilinGuard", AllowedSides: []string{"defender", "reinforcement"},
				Params: map[string]float64{"defenseBonusRate": 0.25, "plunderProtectionRate": 0.2},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "qijin_qichu", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army",
				Params: map[string]float64{"speedBonusRate": 1, "minMarchSeconds": 60},
			},
		},
	}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wei", "caocao", "shu", "zhaoyun")
	attacker.Army = []ArmyUnit{{UnitType: "qiQiYing", Amount: 1000}}
	defender.Army = []ArmyUnit{{UnitType: "qilinGuard", Amount: 10}}
	attacker.Resources.Capacity["wood"] = 100000
	defender.Resources.Items = map[string]int{"wood": 1000}
	defender.Resources.Capacity = map[string]int{"wood": 0}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypePlunder,
		Troops: map[string]int{"qiQiYing": 1000}, GeneralIDs: []string{"caocao"},
	})
	if err != nil {
		t.Fatalf("start PVP plunder failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("resolve PVP plunder failed: %v", err)
	}
	if battle.Plunder["wood"] != 288 {
		t.Fatalf("expected Zhao Yun main defender to protect 20%% of the 360 plunderable wood, got %+v", battle.Plunder)
	}
	attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(attackerReports) == 0 {
		t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
	}
	defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || len(defenderReports) == 0 {
		t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
	}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		outcome, ok := report.TraitOutcomes["longdan_jiuyuan"]
		if !ok {
			t.Fatalf("expected Longdan in report %s, outcomes=%+v", report.ID, report.TraitOutcomes)
		}
		if !reflect.DeepEqual(outcome.Detail["infantryDefenseModifiedUnits"], map[string]int{"qilinGuard": 3}) ||
			!reflect.DeepEqual(outcome.Detail["cavalryDefenseModifiedUnits"], map[string]int{"qilinGuard": 3}) ||
			!reflect.DeepEqual(outcome.Detail["protectedResources"], map[string]int{"wood": 72}) ||
			outcome.Detail["cumulativePlunderProtectionRate"] != 0.2 {
			t.Fatalf("expected merged defense and resource result in report %s, outcome=%+v", report.ID, outcome)
		}
		if report.Rewards["wood"] != 288 && report.ViewType == ReportViewAttack {
			t.Fatalf("expected attacker report to show final 288 wood, report=%+v", report)
		}
	}
}

// TestShuBatchTargetedAttackTraitsOnlyModifyDedicatedUnits 验证关羽和张飞的加攻只作用于各自专属兵种。
func TestShuBatchTargetedAttackTraitsOnlyModifyDedicatedUnits(t *testing.T) {
	tests := []struct {
		name       string
		traitID    string
		target     string
		rate       float64
		wantAttack int
	}{
		{name: "武圣破军", traitID: "wusheng_pojun", target: "azureDragon", rate: 0.38, wantAttack: 138},
		{name: "勇冠三军", traitID: "wanren_nuhou", target: "southernElephant", rate: 0.35, wantAttack: 135},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			attacker := combat.Army{Units: []combat.Unit{
				{ID: tc.target, Count: 100, Attack: 100},
				{ID: "otherUnit", Count: 100, Attack: 100},
			}}
			defender := combat.Army{}
			ctx := &general.BeforeBattleContext{Attacker: &attacker, Defender: &defender, IsPvP: true, Scene: "attack"}
			general.Dispatch(ctx, []general.ActiveTrait{{
				TraitID: tc.traitID, OwnerSide: "attacker", OwnerGeneralID: "shu_general",
				Scope: "self_army", TargetUnitType: tc.target, AllowedSides: []string{"attacker"},
				Params: general.Params{"triggerChance": 1, "attackBonusRate": tc.rate},
			}})
			if attacker.Units[0].Attack != tc.wantAttack || attacker.Units[1].Attack != 100 {
				t.Fatalf("expected only %s attack to become %d, got %+v", tc.target, tc.wantAttack, attacker.Units)
			}
			if ctx.Triggered[tc.traitID].Detail["attackModifiedUnits"] == nil {
				t.Fatalf("expected actual attack changes in report, got %+v", ctx.Triggered)
			}
		})
	}
}

// TestLongdanRescueDefenseAndPlunderStacking 验证麒麟卫双防与主将、援军递减资源保护。
func TestLongdanRescueDefenseAndPlunderStacking(t *testing.T) {
	attacker := combat.Army{}
	defender := combat.Army{Units: []combat.Unit{
		{ID: "qilinGuard", Count: 100, InfantryDefense: 80, CavalryDefense: 60},
		{ID: "greedyWolf", Count: 100, InfantryDefense: 80, CavalryDefense: 60},
	}}
	before := &general.BeforeBattleContext{Attacker: &attacker, Defender: &defender, IsPvP: true, Scene: "plunder"}
	general.Dispatch(before, []general.ActiveTrait{{
		TraitID: "longdan_jiuyuan", OwnerSide: "defender", OwnerGeneralID: "zhaoyun",
		Scope: "self_army", TargetUnitType: "qilinGuard", AllowedSides: []string{"defender", "reinforcement"},
		Params: general.Params{"defenseBonusRate": 0.25, "plunderProtectionRate": 0.2},
	}})
	if defender.Units[0].InfantryDefense != 100 || defender.Units[0].CavalryDefense != 75 ||
		defender.Units[1].InfantryDefense != 80 || defender.Units[1].CavalryDefense != 60 {
		t.Fatalf("expected only Qilin Guard defenses to increase by 25%%, got %+v", defender.Units)
	}

	active := []general.ActiveTrait{{
		TraitID: "longdan_jiuyuan", OwnerSide: "defender", OwnerPlayerID: "city_owner", OwnerGeneralID: "zhaoyun",
		AllowedSides: []string{"defender", "reinforcement"}, Params: general.Params{"plunderProtectionRate": 0.2},
	}}
	for index := 1; index <= 3; index++ {
		active = append(active, general.ActiveTrait{
			TraitID: "longdan_jiuyuan", OwnerSide: "reinforcement", OwnerPlayerID: "reinforcement_owner_" + strconv.Itoa(index), OwnerGeneralID: "zhaoyun",
			AllowedSides: []string{"defender", "reinforcement"}, Params: general.Params{"plunderProtectionRate": 0.2},
		})
	}
	rewards, outcomes := dispatchPlunderActiveTraits(map[string]int{"wood": 1000, "food": 500}, "plunder", active)
	if rewards["wood"] != 450 || rewards["food"] != 225 {
		t.Fatalf("expected 55%% cumulative protection, got rewards=%+v outcomes=%+v", rewards, outcomes)
	}
	if len(outcomes) != 4 {
		t.Fatalf("expected main defender and three reinforcement outcomes, got %+v", outcomes)
	}
	foundFinalRate := false
	for _, outcome := range outcomes {
		if rate, ok := outcome.Detail["cumulativePlunderProtectionRate"].(float64); ok && math.Abs(rate-0.55) < 1e-9 {
			foundFinalRate = true
		}
		if outcome.Detail["protectedResources"] == nil || outcome.Detail["plunderDelta"] == nil {
			t.Fatalf("expected actual protected resource values in every outcome, got %+v", outcome)
		}
	}
	if !foundFinalRate {
		t.Fatalf("expected final cumulative protection rate 55%%, got %+v", outcomes)
	}
}

// TestHuangzhongAndWeiyanPassivesUseModifierPipeline 验证永久属性和兵种固定值进入核心 Modifier 管线。
func TestHuangzhongAndWeiyanPassivesUseModifierPipeline(t *testing.T) {
	huangzhong := General{
		EffectiveStats: map[string]int{}, Attributes: map[string]float64{},
		AttributeBreakdown: map[string][]GeneralAttributeBreakdownItem{}, Buffs: map[string]float64{},
	}
	applyPassiveGeneralStatTrait(&huangzhong, GeneralTraitConfig{
		TraitID: "laodang_yizhuang", Params: map[string]float64{"forceBonus": 12, "commandBonus": 12},
	})
	if huangzhong.EffectiveStats["force"] != 12 || huangzhong.EffectiveStats["command"] != 12 ||
		math.Abs(huangzhong.Attributes[StatAttackBonus]-0.24) > 1e-9 || math.Abs(huangzhong.Attributes[StatDefenseBonus]-0.24) > 1e-9 {
		t.Fatalf("expected Huangzhong +12 force/command and +24%% attack/defense, got %+v", huangzhong)
	}

	weiyan := General{Buffs: map[string]float64{}}
	applyPassiveUnitTraitModifiers(&weiyan, GeneralTraitConfig{
		TraitID: "qibing_raohou", TargetUnitType: "southernElephant",
		Params: map[string]float64{"unitAttackFlat": 18, "unitSpeedFlat": 15},
	})
	if weiyan.Buffs[unitAttackFlatModifierKey("southernElephant")] != 18 || weiyan.Buffs[unitSpeedFlatModifierKey("southernElephant")] != 15 {
		t.Fatalf("expected Weiyan Southern Elephant passive +18/+15, got %+v", weiyan.Buffs)
	}
	if trait, ok := general.Get("qibing_raohou"); !ok || len(trait.Subscribe()) != 0 {
		t.Fatalf("expected Qibing Raohou to be a non-triggering passive, trait=%+v exists=%v", trait, ok)
	}
}
