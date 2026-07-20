// 本文件验证正式进攻型将领特性在 NPC 单城战斗中的战力、兵力和战报一致性。
package game

import (
	"testing"
	"time"

	"hero3/internal/core/general"
)

// TestNpcMaChaoRandomHitAndMissKeepPassiveForceIndependent 验证马超西凉突击命中与否都不影响被动武力形成的战力，且被动不伪装成触发结果。
func TestNpcMaChaoRandomHitAndMissKeepPassiveForceIndependent(t *testing.T) {
	for _, tc := range []struct {
		name             string
		triggerChance    float64
		wantDefenderLoss int
		wantRemaining    int
		wantTriggered    bool
	}{
		{name: "命中", triggerChance: 1, wantDefenderLoss: 737, wantRemaining: 263, wantTriggered: true},
		{name: "未命中", triggerChance: 0, wantDefenderLoss: 617, wantRemaining: 383, wantTriggered: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hero := GeneralHeroConfig{
				ID: "machao", Name: "马超", Faction: "wei", Enabled: true,
				SpecialTrait: GeneralTraitConfig{
					TraitID: "xiliang_tuji", TraitType: general.TraitTypeSpecial, Enabled: true,
					Scope: "enemy_army", TargetUnitType: "cavalry",
					Params: map[string]float64{"triggerChance": tc.triggerChance, "effectRate": 0.12},
				},
				BonusTrait: GeneralTraitConfig{
					TraitID: "tianshen_xiafan", TraitType: general.TraitTypeBonus, Enabled: true,
					Scope: "self_army", Params: map[string]float64{"forceBonus": 20},
				},
			}
			report, stored := resolveNpcFormalRandomTraitTest(t, "machao_"+tc.name, hero, "weiInfantry", 1000, "weiCavalry", 1000, "plunder", true)
			if report.PlayerPower != 14000 || report.EnemyPower != 10000 || report.LostUnits["weiInfantry"] != 382 || report.SurvivedUnits["weiInfantry"] != 618 || report.DefenderLostUnits["weiCavalry"] != tc.wantDefenderLoss || report.GeneralExpGained != tc.wantDefenderLoss {
				t.Fatalf("expected exact Ma Chao NPC result 14000/10000, attacker 382/618 and defender loss %d, report=%+v", tc.wantDefenderLoss, report)
			}
			if got := armySliceToMap(stored.Army)["weiInfantry"]; got != 618 {
				t.Fatalf("expected Ma Chao authoritative survivors 618, got %d", got)
			}
			if got := armySliceToMap(stored.NpcState.Cities[0].Army)["weiCavalry"]; got != tc.wantRemaining {
				t.Fatalf("expected NPC cavalry remaining %d, got %d", tc.wantRemaining, got)
			}
			_, triggered := report.TraitOutcomes["xiliang_tuji"]
			wantTimelineCount := 0
			if tc.wantTriggered {
				wantTimelineCount = 1
			}
			if triggered != tc.wantTriggered || len(report.TraitTriggered) != wantTimelineCount {
				t.Fatalf("expected Xiliang triggered=%t and no passive timeline, outcomes=%+v timeline=%+v", tc.wantTriggered, report.TraitOutcomes, report.TraitTriggered)
			}
			if _, exists := report.TraitOutcomes["tianshen_xiafan"]; exists || standardReportHasTrait(report.Detail, "tianshen_xiafan") {
				t.Fatalf("expected passive Tianshen absent from trigger reports, outcomes=%+v detail=%+v", report.TraitOutcomes, report.Detail)
			}
			if tc.wantTriggered {
				extra := report.TraitOutcomes["xiliang_tuji"].Detail["targetExtraLosses"].(map[string]int)
				if len(extra) != 1 || extra["weiCavalry"] != 120 || report.TraitTriggered[0] != "xiliang_tuji" {
					t.Fatalf("expected Xiliang to add exactly 120 cavalry losses, outcome=%+v", report.TraitOutcomes["xiliang_tuji"])
				}
			}
			assertNpcFormalGeneralSnapshot(t, report, "machao", "xiliang_tuji", "tianshen_xiafan")
			if report.PvpAttackerGenerals[0].Buffs[StatAttackBonus] != 0.4 {
				t.Fatalf("expected passive force +20 to produce 40%% attack modifier, snapshot=%+v", report.PvpAttackerGenerals[0])
			}
			assertNpcFormalUnitRows(t, report, "weiInfantry", 1000, 382, 618, "weiCavalry", 1000, tc.wantDefenderLoss, tc.wantRemaining)
		})
	}
}

// TestNpcLateRandomTraitsHitAndMissKeepDeterministicDamageIndependent 验证黄忠、陆逊和黄盖的随机特性不会吞掉各自确定性战后伤害。
func TestNpcLateRandomTraitsHitAndMissKeepDeterministicDamageIndependent(t *testing.T) {
	type expectedResult struct {
		playerPower, enemyPower int
		playerLoss, npcLoss     int
		timeline                []string
	}
	cases := []struct {
		name      string
		generalID string
		buildHero func(float64) GeneralHeroConfig
		hit       expectedResult
		miss      expectedResult
	}{
		{
			name: "黄忠", generalID: "huangzhong",
			buildHero: func(chance float64) GeneralHeroConfig {
				return GeneralHeroConfig{
					ID: "huangzhong", Name: "黄忠", Faction: "wei", Enabled: true,
					SpecialTrait: GeneralTraitConfig{TraitID: "baibu_chuanyang", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army", AllowedSides: []string{"attacker"}, Params: map[string]float64{"triggerChance": chance, "enemyDefenseReductionRate": 0.2}},
					BonusTrait:   GeneralTraitConfig{TraitID: "laodang_yizhuang", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "enemy_army", Params: map[string]float64{"effectRate": 0.1}},
				}
			},
			hit:  expectedResult{playerPower: 10000, enemyPower: 8000, playerLoss: 421, npcLoss: 678, timeline: []string{"baibu_chuanyang", "laodang_yizhuang"}},
			miss: expectedResult{playerPower: 10000, enemyPower: 10000, playerLoss: 500, npcLoss: 600, timeline: []string{"laodang_yizhuang"}},
		},
		{
			name: "陆逊", generalID: "luxun",
			buildHero: func(chance float64) GeneralHeroConfig {
				return GeneralHeroConfig{
					ID: "luxun", Name: "陆逊", Faction: "wei", Enabled: true,
					SpecialTrait: GeneralTraitConfig{TraitID: "huoshao_lianying", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army", TargetUnitType: "infantry", Params: map[string]float64{"triggerChance": chance, "effectRate": 1, "maxAffectedRate": 1}},
					BonusTrait:   GeneralTraitConfig{TraitID: "lianying_zengshang", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "enemy_army", TargetUnitType: "infantry", Params: map[string]float64{"effectRate": 0.1}},
				}
			},
			hit:  expectedResult{playerPower: 10000, enemyPower: 10000, playerLoss: 500, npcLoss: 1000, timeline: []string{"huoshao_lianying"}},
			miss: expectedResult{playerPower: 10000, enemyPower: 10000, playerLoss: 500, npcLoss: 600, timeline: []string{"lianying_zengshang"}},
		},
		{
			name: "黄盖", generalID: "huanggai",
			buildHero: func(chance float64) GeneralHeroConfig {
				return GeneralHeroConfig{
					ID: "huanggai", Name: "黄盖", Faction: "wei", Enabled: true,
					SpecialTrait: GeneralTraitConfig{TraitID: "kurouji", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_traits", Params: map[string]float64{"triggerChance": chance, "disableTraitCount": 1}},
					BonusTrait:   GeneralTraitConfig{TraitID: "kurou_fanji", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "enemy_army", Params: map[string]float64{"effectRate": 0.1}},
				}
			},
			hit:  expectedResult{playerPower: 10000, enemyPower: 10000, playerLoss: 500, npcLoss: 600, timeline: []string{"kurouji", "kurou_fanji"}},
			miss: expectedResult{playerPower: 10000, enemyPower: 10000, playerLoss: 500, npcLoss: 600, timeline: []string{"kurou_fanji"}},
		},
	}

	for _, tc := range cases {
		for _, chanceCase := range []struct {
			name   string
			chance float64
			want   expectedResult
		}{{name: "命中", chance: 1, want: tc.hit}, {name: "未命中", chance: 0, want: tc.miss}} {
			t.Run(tc.name+chanceCase.name, func(t *testing.T) {
				hero := tc.buildHero(chanceCase.chance)
				report, stored := resolveNpcFormalRandomTraitTest(t, tc.generalID+chanceCase.name, hero, "weiInfantry", 1000, "weiInfantry", 1000, "plunder", false)
				want := chanceCase.want
				if report.PlayerPower != want.playerPower || report.EnemyPower != want.enemyPower || report.LostUnits["weiInfantry"] != want.playerLoss || report.SurvivedUnits["weiInfantry"] != 1000-want.playerLoss || report.DefenderLostUnits["weiInfantry"] != want.npcLoss || report.GeneralExpGained != want.npcLoss {
					t.Fatalf("expected exact %s NPC result power %d/%d and losses %d/%d, report=%+v", tc.name, want.playerPower, want.enemyPower, want.playerLoss, want.npcLoss, report)
				}
				if got := armySliceToMap(stored.Army)["weiInfantry"]; got != 1000-want.playerLoss {
					t.Fatalf("expected %s authoritative player survivors %d, got %d", tc.name, 1000-want.playerLoss, got)
				}
				if got := armySliceToMap(stored.NpcState.Cities[0].Army)["weiInfantry"]; got != 1000-want.npcLoss {
					t.Fatalf("expected %s authoritative NPC survivors %d, got %d", tc.name, 1000-want.npcLoss, got)
				}
				if len(report.TraitTriggered) != len(want.timeline) {
					t.Fatalf("expected %s timeline %v, got %v", tc.name, want.timeline, report.TraitTriggered)
				}
				for index := range want.timeline {
					if report.TraitTriggered[index] != want.timeline[index] {
						t.Fatalf("expected %s timeline %v, got %v", tc.name, want.timeline, report.TraitTriggered)
					}
				}
				assertNpcFormalGeneralSnapshot(t, report, tc.generalID, hero.SpecialTrait.TraitID, hero.BonusTrait.TraitID)
				assertNpcFormalUnitRows(t, report, "weiInfantry", 1000, want.playerLoss, 1000-want.playerLoss, "weiInfantry", 1000, want.npcLoss, 1000-want.npcLoss)
				assertNpcLateRandomTraitDetails(t, report, tc.generalID, chanceCase.chance == 1)
			})
		}
	}
}

// TestNpcSunCePursuitHitAndLegalMissKeepTieqiIndependent 验证孙策在 NPC 掠夺获胜后追击命中或合法未命中都保留霸王骑加攻。
func TestNpcSunCePursuitHitAndLegalMissKeepTieqiIndependent(t *testing.T) {
	for _, tc := range []struct {
		name             string
		triggerChance    float64
		wantDefenderLoss int
		wantRemaining    int
		wantTimeline     []string
	}{
		{name: "命中", triggerChance: 1, wantDefenderLoss: 821, wantRemaining: 179, wantTimeline: []string{"xiaobawang_tieqi", "xiaobawang_zhuiji"}},
		{name: "合法未命中", triggerChance: 0, wantDefenderLoss: 721, wantRemaining: 279, wantTimeline: []string{"xiaobawang_tieqi"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hero := GeneralHeroConfig{
				ID: "sunce", Name: "孙策", Faction: "wei", Enabled: true,
				SpecialTrait: GeneralTraitConfig{TraitID: "xiaobawang_zhuiji", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army", AllowedScenes: []string{"plunder"}, RequiredOutcome: "win", Params: map[string]float64{"triggerChance": tc.triggerChance, "effectRate": 0.1}},
				BonusTrait:   GeneralTraitConfig{TraitID: "xiaobawang_tieqi", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army", TargetUnitType: "overlordRider", AllowedSides: []string{"attacker"}, Params: map[string]float64{"unitAttackFlat": 50}},
			}
			report, stored := resolveNpcFormalRandomTraitTest(t, "sunce_"+tc.name, hero, "overlordRider", 200, "weiInfantry", 1000, "plunder", false)
			if report.Result != "attacker_victory" || report.PlayerPower != 15600 || report.EnemyPower != 8000 || report.LostUnits["overlordRider"] != 55 || report.SurvivedUnits["overlordRider"] != 145 || report.DefenderLostUnits["weiInfantry"] != tc.wantDefenderLoss || report.GeneralExpGained != tc.wantDefenderLoss {
				t.Fatalf("expected exact Sun Ce NPC plunder 15600/8000, attacker 55/145 and defender loss %d, report=%+v", tc.wantDefenderLoss, report)
			}
			if got := armySliceToMap(stored.Army)["overlordRider"]; got != 145 {
				t.Fatalf("expected Sun Ce authoritative survivors 145, got %d", got)
			}
			if got := armySliceToMap(stored.NpcState.Cities[0].Army)["weiInfantry"]; got != tc.wantRemaining {
				t.Fatalf("expected NPC infantry remaining %d, got %d", tc.wantRemaining, got)
			}
			if len(report.TraitTriggered) != len(tc.wantTimeline) {
				t.Fatalf("expected Sun Ce timeline %v, got %v", tc.wantTimeline, report.TraitTriggered)
			}
			for index := range tc.wantTimeline {
				if report.TraitTriggered[index] != tc.wantTimeline[index] {
					t.Fatalf("expected Sun Ce timeline %v, got %v", tc.wantTimeline, report.TraitTriggered)
				}
			}
			attackModified := report.TraitOutcomes["xiaobawang_tieqi"].Detail["attackModifiedUnits"].(map[string]int)
			if len(attackModified) != 1 || attackModified["overlordRider"] != 50 {
				t.Fatalf("expected Tieqi to add exactly 50 attack in both cases, outcome=%+v", report.TraitOutcomes["xiaobawang_tieqi"])
			}
			if tc.triggerChance == 0 {
				if _, exists := report.TraitOutcomes["xiaobawang_zhuiji"]; exists || standardReportHasTrait(report.Detail, "xiaobawang_zhuiji") {
					t.Fatalf("expected legally missed pursuit absent from trigger reports, outcomes=%+v detail=%+v", report.TraitOutcomes, report.Detail)
				}
			} else {
				extra := report.TraitOutcomes["xiaobawang_zhuiji"].Detail["extraLosses"].(map[string]int)
				if extra["weiInfantry"] != 100 {
					t.Fatalf("expected pursuit to add exactly 100 losses, outcome=%+v", report.TraitOutcomes["xiaobawang_zhuiji"])
				}
			}
			assertNpcFormalGeneralSnapshot(t, report, "sunce", "xiaobawang_zhuiji", "xiaobawang_tieqi")
			assertNpcFormalUnitRows(t, report, "overlordRider", 200, 55, 145, "weiInfantry", 1000, tc.wantDefenderLoss, tc.wantRemaining)
		})
	}
}

// TestNpcAttackBonusTraitsMatchPowerStateAndReports 验证七项正式攻击加成真实进入 NPC 战力并保存实际整数变化。
func TestNpcAttackBonusTraitsMatchPowerStateAndReports(t *testing.T) {
	cases := []struct {
		name        string
		traitID     string
		generalID   string
		unitType    string
		target      string
		mode        string
		params      map[string]float64
		designKey   string
		designValue float64
		baseAttack  int
		attackDelta int
	}{
		{name: "死战到底", traitID: "sizhandaodi", generalID: "dianwei", unitType: "weiInfantry", target: "infantry", mode: "attack", params: map[string]float64{"attackBonusRate": 0.35}, designKey: "attackBonusRate", designValue: 0.35, baseAttack: 10, attackDelta: 4},
		{name: "威震逍遥", traitID: "weizhen_xiaoyao", generalID: "zhangliao", unitType: "weiCavalry", target: "cavalry", mode: "attack", params: map[string]float64{"attackBonusRate": 0.35}, designKey: "attackBonusRate", designValue: 0.35, baseAttack: 14, attackDelta: 5},
		{name: "武圣破军", traitID: "wusheng_pojun", generalID: "guanyu", unitType: "weiInfantry", mode: "attack", params: map[string]float64{"attackBonusRate": 0.2}, designKey: "attackBonusRate", designValue: 0.2, baseAttack: 10, attackDelta: 2},
		{name: "万人怒吼", traitID: "wanren_nuhou", generalID: "zhangfei", unitType: "weiInfantry", target: "infantry", mode: "attack", params: map[string]float64{"attackBonusRate": 0.2}, designKey: "attackBonusRate", designValue: 0.2, baseAttack: 10, attackDelta: 2},
		{name: "小霸王", traitID: "xiaobawang_tieqi", generalID: "sunce", unitType: "overlordRider", target: "overlordRider", mode: "attack", params: map[string]float64{"unitAttackFlat": 50}, designKey: "unitAttackFlat", designValue: 50, baseAttack: 28, attackDelta: 50},
		{name: "美周郎军略", traitID: "meizhoulang_junlue", generalID: "zhouyu", unitType: "weiInfantry", mode: "attack", params: map[string]float64{"attackBonusRate": 0.05}, designKey: "attackBonusRate", designValue: 0.05, baseAttack: 10, attackDelta: 1},
		{name: "锦帆奇袭", traitID: "jinfan_qixi", generalID: "ganning", unitType: "weiInfantry", mode: "plunder", params: map[string]float64{"attackBonusRate": 0.1}, designKey: "attackBonusRate", designValue: 0.1, baseAttack: 10, attackDelta: 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			report, stored := resolveNpcAttackBonusTraitTest(t, tc.traitID, tc.generalID, tc.unitType, tc.target, tc.mode, tc.params)
			wantPower := (tc.baseAttack + tc.attackDelta) * 100
			if report.PlayerPower != wantPower {
				t.Fatalf("expected %s NPC attack power %d, got %d", tc.traitID, wantPower, report.PlayerPower)
			}
			outcome, ok := report.TraitOutcomes[tc.traitID]
			modified, detailOK := outcome.Detail["attackModifiedUnits"].(map[string]int)
			if !ok || !detailOK || modified[tc.unitType] != tc.attackDelta || outcome.Detail[tc.designKey] != tc.designValue || outcome.OwnerSide != "attacker" || outcome.OwnerGeneralID != tc.generalID {
				t.Fatalf("expected %s design and actual attack result, outcome=%+v", tc.traitID, outcome)
			}

			standardFound := false
			if report.Detail != nil && report.Detail.PrimarySide.Power == report.PlayerPower {
				for _, trait := range report.Detail.Traits {
					if trait.TraitID == tc.traitID {
						standardModified, standardOK := trait.Detail["attackModifiedUnits"].(map[string]int)
						standardFound = standardOK && standardModified[tc.unitType] == tc.attackDelta && trait.OwnerSide == "primary" && trait.OwnerRole == "attacker" && trait.GeneralID == tc.generalID
					}
				}
			}
			if !standardFound {
				t.Fatalf("expected standard NPC report to preserve %s result, detail=%+v", tc.traitID, report.Detail)
			}

			if got, want := armySliceToMap(stored.Army)[tc.unitType], 200-report.LostUnits[tc.unitType]; got != want {
				t.Fatalf("expected %s player army %d to match losses, got %d", tc.traitID, want, got)
			}
			if stored.NpcState == nil || len(stored.NpcState.Cities) != 1 {
				t.Fatalf("expected stored NPC state after %s, state=%+v", tc.traitID, stored.NpcState)
			}
			if got, want := armySliceToMap(stored.NpcState.Cities[0].Army)["weiInfantry"], 100-report.DefenderLostUnits["weiInfantry"]; got != want {
				t.Fatalf("expected %s NPC army %d to match losses, got %d", tc.traitID, want, got)
			}
		})
	}
}

// TestNpcGeneralSnapshotStaysAtPreBattleLevelAfterUpgrade 验证 NPC 战后升级不会回写本场参战将领快照。
func TestNpcGeneralSnapshotStaysAtPreBattleLevelAfterUpgrade(t *testing.T) {
	report, stored := resolveNpcAttackBonusTraitTest(
		t, "meizhoulang_junlue", "zhouyu", "weiInfantry", "", "attack",
		map[string]float64{"attackBonusRate": 0.05}, true,
	)
	baselineExp := generalExpRequiredForLevelForTest(2) - 1
	if report.PlayerPower != 1100 || report.GeneralExpGained != 100 || report.GeneralLevelBefore != 1 || report.GeneralLevelAfter != 2 {
		t.Fatalf("expected NPC power 1100 and separate 100 exp Lv.1 -> Lv.2, report=%+v", report)
	}
	if len(report.PvpAttackerGenerals) != 1 || report.PvpAttackerGenerals[0].ID != "zhouyu" || report.PvpAttackerGenerals[0].Level != 1 {
		t.Fatalf("expected legacy NPC snapshot to keep pre-battle Lv.1 Zhou Yu, snapshots=%+v", report.PvpAttackerGenerals)
	}
	if report.Detail == nil || len(report.Detail.PrimarySide.Generals) != 1 || report.Detail.PrimarySide.Generals[0].ID != "zhouyu" || report.Detail.PrimarySide.Generals[0].Level != 1 ||
		report.Detail.Rewards.GeneralExp != 100 || report.Detail.Rewards.GeneralLevelBefore != 1 || report.Detail.Rewards.GeneralLevelAfter != 2 {
		t.Fatalf("expected standard NPC snapshot Lv.1 and reward Lv.1 -> Lv.2, detail=%+v", report.Detail)
	}
	if !standardReportHasTrait(report.Detail, "meizhoulang_junlue") || standardReportHasTrait(report.Detail, "huogong") {
		t.Fatalf("expected only real attack bonus timeline, detail=%+v", report.Detail)
	}
	if got := pvpTestGeneralExp(stored, "zhouyu"); got != baselineExp+100 {
		t.Fatalf("expected stored Zhou Yu exp %d, got %d", baselineExp+100, got)
	}
	if got := pvpTestGeneralLevel(stored, "zhouyu"); got != 2 {
		t.Fatalf("expected stored Zhou Yu level 2, got %d", got)
	}
	if got, want := armySliceToMap(stored.Army)["weiInfantry"], 200-report.LostUnits["weiInfantry"]; got != want {
		t.Fatalf("expected stored NPC army %d after losses, got %d", want, got)
	}
	if got, want := report.SurvivedUnits["weiInfantry"], report.DispatchedUnits["weiInfantry"]-report.LostUnits["weiInfantry"]+report.RevivedUnits["weiInfantry"]+report.CapturedUnits["weiInfantry"]; got != want {
		t.Fatalf("expected NPC report survivors %d, got %d", want, got)
	}
}

// TestNpcEnemyDefenseReductionTraitsMatchPowerStateAndReports 验证五项主动破防真实降低 NPC 防御战力并保存两类实际变化。
func TestNpcEnemyDefenseReductionTraitsMatchPowerStateAndReports(t *testing.T) {
	cases := []struct {
		name         string
		traitID      string
		traitType    string
		generalID    string
		rate         float64
		defensePower int
		defenseDelta int
	}{
		{name: "魅惑扰阵", traitID: "meihuo_raozhen", traitType: general.TraitTypeBonus, generalID: "zhenmi", rate: 0.1, defensePower: 900, defenseDelta: -1},
		{name: "虎痴冲阵", traitID: "huchi_chongzhen", traitType: general.TraitTypeSpecial, generalID: "xuchu", rate: 0.2, defensePower: 800, defenseDelta: -2},
		{name: "破敌防御", traitID: "pojun_pofang", traitType: general.TraitTypeBonus, generalID: "xuchu", rate: 0.35, defensePower: 700, defenseDelta: -3},
		{name: "百步穿杨", traitID: "baibu_chuanyang", traitType: general.TraitTypeSpecial, generalID: "huangzhong", rate: 0.2, defensePower: 800, defenseDelta: -2},
		{name: "奇兵绕后", traitID: "qibing_raohou", traitType: general.TraitTypeSpecial, generalID: "weiyan", rate: 0.2, defensePower: 800, defenseDelta: -2},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			report, stored := resolveNpcEnemyDefenseTraitTest(t, tc.traitID, tc.traitType, tc.generalID, tc.rate)
			if report.EnemyPower != tc.defensePower || report.PlayerPower != 1000 {
				t.Fatalf("expected %s attack/defense power 1000/%d, got %d/%d", tc.traitID, tc.defensePower, report.PlayerPower, report.EnemyPower)
			}
			outcome, ok := report.TraitOutcomes[tc.traitID]
			infantry, infantryOK := outcome.Detail["infantryDefenseModifiedUnits"].(map[string]int)
			cavalry, cavalryOK := outcome.Detail["cavalryDefenseModifiedUnits"].(map[string]int)
			if !ok || !infantryOK || !cavalryOK || infantry["weiInfantry"] != tc.defenseDelta || cavalry["weiInfantry"] != tc.defenseDelta || outcome.Detail["enemyDefenseReductionRate"] != tc.rate || outcome.OwnerSide != "attacker" || outcome.OwnerGeneralID != tc.generalID {
				t.Fatalf("expected %s design rate and actual defense changes, outcome=%+v", tc.traitID, outcome)
			}

			standardFound := false
			if report.Detail != nil && report.Detail.SecondarySide != nil && report.Detail.SecondarySide.Power == report.EnemyPower {
				for _, trait := range report.Detail.Traits {
					if trait.TraitID == tc.traitID {
						standardInfantry, standardInfantryOK := trait.Detail["infantryDefenseModifiedUnits"].(map[string]int)
						standardCavalry, standardCavalryOK := trait.Detail["cavalryDefenseModifiedUnits"].(map[string]int)
						standardFound = standardInfantryOK && standardCavalryOK && standardInfantry["weiInfantry"] == tc.defenseDelta && standardCavalry["weiInfantry"] == tc.defenseDelta && trait.OwnerSide == "primary" && trait.OwnerRole == "attacker" && trait.GeneralID == tc.generalID
					}
				}
			}
			if !standardFound {
				t.Fatalf("expected standard NPC report to preserve %s result, detail=%+v", tc.traitID, report.Detail)
			}

			if got, want := armySliceToMap(stored.Army)["weiInfantry"], 200-report.LostUnits["weiInfantry"]; got != want {
				t.Fatalf("expected %s player army %d to match losses, got %d", tc.traitID, want, got)
			}
			if got, want := armySliceToMap(stored.NpcState.Cities[0].Army)["weiInfantry"], 100-report.DefenderLostUnits["weiInfantry"]; got != want {
				t.Fatalf("expected %s NPC army %d to match losses, got %d", tc.traitID, want, got)
			}
		})
	}
}

// TestNpcZhenMiTraitsCaptureAndDefenseReductionReconcileState 验证甄宓先俘虏再破防时，NPC 库存、获得驻防和标准战报使用同一最终状态。
func TestNpcZhenMiTraitsCaptureAndDefenseReductionReconcileState(t *testing.T) {
	setTestCombatUnitsConfig(t)
	unitsMu.Lock()
	activeUnits["shu"] = FactionUnits{
		"shuInfantry": UnitConfig{
			Name: "蜀步兵", Category: "infantry",
			Stats: map[string]int{"attack": 10, "infantryDefense": 10, "cavalryDefense": 8, "carryCapacity": 5, "upkeep": 1},
		},
	}
	unitsMu.Unlock()
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "zhenmi", Name: "甄宓"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"zhenmi": {
			ID: "zhenmi", Name: "甄宓", Faction: "wei", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "meiren", TraitType: general.TraitTypeSpecial, Enabled: true,
				Scope: "self_army", AllowedSides: []string{"attacker"},
				Params: map[string]float64{"captureRate": 0.2, "captureMax": 10000, "triggerChance": 1},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "meihuo_raozhen", TraitType: general.TraitTypeBonus, Enabled: true,
				Scope: "enemy_army", AllowedSides: []string{"attacker"},
				Params: map[string]float64{"enemyDefenseReductionRate": 0.1},
			},
		},
	}})

	now := time.Now().UTC()
	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)
	account := Account{ID: "account_npc_zhenmi_combo", Username: "npc_zhenmi_combo", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}
	state := newPlayerState("player_npc_zhenmi_combo", "甄宓组合测试", "wei", "zhenmi", now)
	state.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1000}}
	state.NpcState = &NpcState{
		Cities: []NpcCity{{
			ID: "npc_zhenmi_combo", Name: "蜀军测试城", Faction: "shu",
			Resources: map[string]int{"wood": 0}, StorageCapacity: map[string]int{}, ProductionPerHour: map[string]int{},
			Army: []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}, MaxArmy: []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}},
			ResourceSettledAt: now.Format(resourceDateLayout), ArmySettledAt: now.Format(resourceDateLayout), GeneratedAt: now.Format(resourceDateLayout),
		}},
		LastRefreshedAt: now.Format(resourceDateLayout),
	}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("CreatePlayer failed: %v", err)
	}

	result, err := svc.AttackNpc(AttackNpcRequest{
		PlayerID: state.Player.ID, NpcID: state.NpcState.Cities[0].ID, Mode: "attack",
		Units: map[string]int{"weiInfantry": 1000}, GeneralIDs: []string{"zhenmi"},
	})
	if err != nil {
		t.Fatalf("AttackNpc failed: %v", err)
	}
	report := result.BattleReport
	if report.PlayerPower != 10000 || report.EnemyPower != 7200 {
		t.Fatalf("expected 10000 attack and 800 remaining defenders at 9 defense = 7200, got %d/%d", report.PlayerPower, report.EnemyPower)
	}
	if report.CapturedToGarrison["shuInfantry"] != 200 || len(report.CapturedUnits) != 0 {
		t.Fatalf("expected 200 cross-faction captives to obtained garrison, report=%+v", report)
	}
	if report.DefenderUnits["shuInfantry"] != 1000 || report.DefenderLostUnits["shuInfantry"] != 800 {
		t.Fatalf("expected original 1000 defenders, 200 captured and 800 core losses, report=%+v", report)
	}
	for _, traitID := range []string{"meiren", "meihuo_raozhen"} {
		if _, ok := report.TraitOutcomes[traitID]; !ok || !standardReportHasTrait(report.Detail, traitID) {
			t.Fatalf("expected %s in both report formats, outcomes=%+v detail=%+v", traitID, report.TraitOutcomes, report.Detail)
		}
	}
	captured, capturedOK := report.TraitOutcomes["meiren"].Detail["capturedToGarrison"].(map[string]int)
	infantryDefense, infantryOK := report.TraitOutcomes["meihuo_raozhen"].Detail["infantryDefenseModifiedUnits"].(map[string]int)
	cavalryDefense, cavalryOK := report.TraitOutcomes["meihuo_raozhen"].Detail["cavalryDefenseModifiedUnits"].(map[string]int)
	if !capturedOK || captured["shuInfantry"] != 200 || !infantryOK || !cavalryOK || infantryDefense["shuInfantry"] != -1 || cavalryDefense["shuInfantry"] != -1 {
		t.Fatalf("expected actual capture and defense changes, outcomes=%+v", report.TraitOutcomes)
	}

	stored, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	if got := armySliceToMap(stored.NpcState.Cities[0].Army)["shuInfantry"]; got != 0 {
		t.Fatalf("expected captured 200 plus 800 core losses to empty NPC army, got %d", got)
	}
	if got, want := armySliceToMap(stored.Army)["weiInfantry"], 1000-report.LostUnits["weiInfantry"]; got != want {
		t.Fatalf("expected player army and report losses to reconcile, got=%d want=%d", got, want)
	}
	garrison, err := repo.GetReinforcement(ObtainedGarrisonID(state.Player.ID))
	if err != nil || garrison.RemainingTroops["shuInfantry"] != 200 {
		t.Fatalf("expected obtained garrison with 200 captives, garrison=%+v err=%v", garrison, err)
	}
	if report.Detail == nil || report.Detail.SecondarySide == nil {
		t.Fatalf("expected standard NPC defender detail, report=%+v", report)
	}
	foundUnit := false
	for _, unit := range report.Detail.SecondarySide.Units {
		if unit.UnitType != "shuInfantry" {
			continue
		}
		if unit.AmountBefore != 1000 || unit.Dispatched != 1000 || unit.Lost != 800 || unit.Survived != 0 {
			t.Fatalf("expected standard defender 1000 dispatched, 800 dead, 200 captured and 0 remaining, unit=%+v", unit)
		}
		foundUnit = true
	}
	if !foundUnit {
		t.Fatalf("expected shuInfantry standard row, detail=%+v", report.Detail)
	}
}

// TestNpcAfterCombatDamageTraitsMatchRealStateAndReports 验证四项正式战后追加伤害真实扣除 NPC 兵力并写入两套战报。
func TestNpcAfterCombatDamageTraitsMatchRealStateAndReports(t *testing.T) {
	cases := []struct {
		name           string
		traitID        string
		traitType      string
		generalID      string
		targetUnitType string
		effectRate     float64
		detailKey      string
		killsTarget    bool
	}{
		{name: "老当益壮", traitID: "laodang_yizhuang", traitType: general.TraitTypeBonus, generalID: "huangzhong", effectRate: 0.1, detailKey: "extraLosses"},
		{name: "火烧联营", traitID: "huoshao_lianying", traitType: general.TraitTypeSpecial, generalID: "luxun", targetUnitType: "infantry", effectRate: 1, detailKey: "targetExtraLosses", killsTarget: true},
		{name: "连营增伤", traitID: "lianying_zengshang", traitType: general.TraitTypeBonus, generalID: "luxun", targetUnitType: "infantry", effectRate: 0.1, detailKey: "targetExtraLosses"},
		{name: "苦肉反击", traitID: "kurou_fanji", traitType: general.TraitTypeBonus, generalID: "huanggai", effectRate: 0.1, detailKey: "extraLosses"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			report, stored := resolveNpcAfterCombatDamageTraitTest(t, tc.traitID, tc.traitType, tc.generalID, tc.targetUnitType, tc.effectRate)
			outcome, ok := report.TraitOutcomes[tc.traitID]
			extraLosses, detailOK := outcome.Detail[tc.detailKey].(map[string]int)
			actualExtra := extraLosses["weiInfantry"]
			if !ok || !detailOK || actualExtra <= 0 || outcome.Detail["effectRate"] != tc.effectRate || outcome.Detail["triggerChance"] != 1.0 || outcome.OwnerSide != "attacker" || outcome.OwnerGeneralID != tc.generalID {
				t.Fatalf("expected %s design and actual NPC extra losses, outcome=%+v", tc.traitID, outcome)
			}
			if !tc.killsTarget && actualExtra != 100 {
				t.Fatalf("expected %s to add 100 losses from 1000 original troops, got %d", tc.traitID, actualExtra)
			}

			lost := report.DefenderLostUnits["weiInfantry"]
			remaining := armySliceToMap(stored.NpcState.Cities[0].Army)["weiInfantry"]
			if report.DefenderUnits["weiInfantry"] != 1000 || remaining != 1000-lost || lost < actualExtra {
				t.Fatalf("expected %s NPC state to match legacy losses, before=%+v lost=%+v remaining=%d extra=%d", tc.traitID, report.DefenderUnits, report.DefenderLostUnits, remaining, actualExtra)
			}
			if tc.killsTarget && (lost != 1000 || remaining != 0) {
				t.Fatalf("expected %s to eliminate all target infantry, lost=%d remaining=%d", tc.traitID, lost, remaining)
			}
			if got, want := armySliceToMap(stored.Army)["weiInfantry"], 400-report.LostUnits["weiInfantry"]; got != want {
				t.Fatalf("expected %s player army %d to match returned troops, got %d", tc.traitID, want, got)
			}

			if report.Detail == nil || report.Detail.SecondarySide == nil {
				t.Fatalf("expected %s standard NPC report, detail=%+v", tc.traitID, report.Detail)
			}
			standardUnitFound := false
			for _, unit := range report.Detail.SecondarySide.Units {
				if unit.UnitType == "weiInfantry" {
					standardUnitFound = unit.AmountBefore == 1000 && unit.Lost == lost && unit.Survived == remaining
				}
			}
			standardTraitFound := false
			for _, trait := range report.Detail.Traits {
				if trait.TraitID != tc.traitID {
					continue
				}
				standardExtra, standardOK := trait.Detail[tc.detailKey].(map[string]int)
				standardTraitFound = standardOK && standardExtra["weiInfantry"] == actualExtra && trait.OwnerSide == "primary" && trait.OwnerRole == "attacker" && trait.GeneralID == tc.generalID
			}
			if !standardUnitFound || !standardTraitFound {
				t.Fatalf("expected %s standard report to match real state and owner, detail=%+v", tc.traitID, report.Detail)
			}
		})
	}
}

// TestNpcRemainingAfterCombatDamageTraitsMatchRealStateAndReports 验证西凉突击与火攻以正式将领身份真实扣除指定 NPC 兵种。
func TestNpcRemainingAfterCombatDamageTraitsMatchRealStateAndReports(t *testing.T) {
	cases := []struct {
		name            string
		traitID         string
		generalID       string
		targetUnitType  string
		params          map[string]float64
		detailKey       string
		targetUnit      string
		expectedExtra   int
		initialPlayer   int
		dispatched      int
		npcArmy         []ArmyUnit
		expectedTargets int
	}{
		{
			name: "西凉突击", traitID: "xiliang_tuji", generalID: "machao", targetUnitType: "cavalry",
			params: map[string]float64{"effectRate": 0.12, "triggerChance": 1}, detailKey: "targetExtraLosses", targetUnit: "weiCavalry", expectedExtra: 60,
			initialPlayer: 600, dispatched: 500, npcArmy: []ArmyUnit{{UnitType: "weiInfantry", Amount: 500}, {UnitType: "weiCavalry", Amount: 500}}, expectedTargets: 2,
		},
		{
			name: "火烧赤壁", traitID: "huogong", generalID: "zhouyu",
			params: map[string]float64{"effectRate": 0.25, "damagePercent": 0.25, "triggerChance": 1}, detailKey: "targetExtraLosses", targetUnit: "weiInfantry", expectedExtra: 250,
			initialPlayer: 400, dispatched: 300, npcArmy: []ArmyUnit{{UnitType: "weiInfantry", Amount: 1000}}, expectedTargets: 1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			report, stored := resolveNpcRemainingAfterCombatDamageTraitTest(t, tc.traitID, tc.generalID, tc.targetUnitType, tc.params, tc.initialPlayer, tc.dispatched, tc.npcArmy)
			outcome, ok := report.TraitOutcomes[tc.traitID]
			extraLosses, detailOK := outcome.Detail[tc.detailKey].(map[string]int)
			if !ok || !detailOK || len(extraLosses) != 1 || extraLosses[tc.targetUnit] != tc.expectedExtra || outcome.OwnerSide != "attacker" || outcome.OwnerGeneralID != tc.generalID {
				t.Fatalf("expected %s to add %d losses only to %s, outcome=%+v", tc.traitID, tc.expectedExtra, tc.targetUnit, outcome)
			}
			if outcome.Detail["triggerChance"] != 1.0 {
				t.Fatalf("expected %s forced trigger chance in report, detail=%+v", tc.traitID, outcome.Detail)
			}
			if tc.traitID == "huogong" {
				if outcome.Detail["damagePercent"] != 0.25 || outcome.Detail["extraDamage"] != tc.expectedExtra {
					t.Fatalf("expected fire design and aggregate damage 25%%/%d, detail=%+v", tc.expectedExtra, outcome.Detail)
				}
			} else if outcome.Detail["effectRate"] != 0.12 {
				t.Fatalf("expected Xiliang design rate 12%%, detail=%+v", outcome.Detail)
			}

			storedNPC := armySliceToMap(stored.NpcState.Cities[0].Army)
			for _, before := range tc.npcArmy {
				lost := report.DefenderLostUnits[before.UnitType]
				if report.DefenderUnits[before.UnitType] != before.Amount || storedNPC[before.UnitType] != before.Amount-lost {
					t.Fatalf("expected %s NPC %s state to match report, before=%d lost=%d stored=%d", tc.traitID, before.UnitType, before.Amount, lost, storedNPC[before.UnitType])
				}
			}
			if got, want := armySliceToMap(stored.Army)["weiInfantry"], tc.initialPlayer-report.LostUnits["weiInfantry"]; got != want {
				t.Fatalf("expected %s player army %d to match returned troops, got %d", tc.traitID, want, got)
			}

			if report.Detail == nil || report.Detail.SecondarySide == nil {
				t.Fatalf("expected %s standard NPC report, detail=%+v", tc.traitID, report.Detail)
			}
			matchedUnits := 0
			for _, unit := range report.Detail.SecondarySide.Units {
				for _, before := range tc.npcArmy {
					if unit.UnitType == before.UnitType && unit.AmountBefore == before.Amount && unit.Lost == report.DefenderLostUnits[before.UnitType] && unit.Survived == storedNPC[before.UnitType] {
						matchedUnits++
					}
				}
			}
			standardTraitFound := false
			for _, trait := range report.Detail.Traits {
				if trait.TraitID != tc.traitID {
					continue
				}
				standardExtra, standardOK := trait.Detail[tc.detailKey].(map[string]int)
				standardTraitFound = standardOK && standardExtra[tc.targetUnit] == tc.expectedExtra && trait.OwnerSide == "primary" && trait.OwnerRole == "attacker" && trait.GeneralID == tc.generalID
			}
			if matchedUnits != tc.expectedTargets || !standardTraitFound {
				t.Fatalf("expected %s standard report to match all NPC state and owner, detail=%+v", tc.traitID, report.Detail)
			}
		})
	}
}

// TestNpcXiaobawangZhuijiRequiresPlunderVictory 验证小霸王追击只在 NPC 掠夺获胜后追加真实损失。
func TestNpcXiaobawangZhuijiRequiresPlunderVictory(t *testing.T) {
	cases := []struct {
		name          string
		mode          string
		playerCount   int
		npcCount      int
		wantResult    string
		wantTriggered bool
	}{
		{name: "掠夺获胜触发", mode: "plunder", playerCount: 200, npcCount: 100, wantResult: "attacker_victory", wantTriggered: true},
		{name: "普通进攻无效", mode: "attack", playerCount: 200, npcCount: 100, wantResult: "attacker_victory", wantTriggered: false},
		{name: "掠夺战败无效", mode: "plunder", playerCount: 100, npcCount: 200, wantResult: "defender_victory", wantTriggered: false},
		{name: "掠夺平局无效", mode: "plunder", playerCount: 100, npcCount: 100, wantResult: "draw", wantTriggered: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			report, stored := resolveNpcPursuitTraitTest(t, tc.mode, tc.playerCount, tc.npcCount)
			if report.Result != tc.wantResult {
				t.Fatalf("expected result %s, got %s", tc.wantResult, report.Result)
			}
			outcome, triggered := report.TraitOutcomes["xiaobawang_zhuiji"]
			if triggered != tc.wantTriggered {
				t.Fatalf("expected pursuit triggered=%t, outcomes=%+v", tc.wantTriggered, report.TraitOutcomes)
			}
			if got, want := armySliceToMap(stored.NpcState.Cities[0].Army)["weiInfantry"], tc.npcCount-report.DefenderLostUnits["weiInfantry"]; got != want {
				t.Fatalf("expected pursuit NPC army %d to match report, got %d", want, got)
			}
			if got, want := armySliceToMap(stored.Army)["weiInfantry"], tc.playerCount-report.LostUnits["weiInfantry"]; got != want {
				t.Fatalf("expected pursuit player army %d to match report, got %d", want, got)
			}
			if tc.wantResult == "draw" && (report.PlayerPower != 1000 || report.EnemyPower != 1000 || report.LostUnits["weiInfantry"] != 50 || report.DefenderLostUnits["weiInfantry"] != 50 || report.SurvivedUnits["weiInfantry"] != 50 || report.GeneralExpGained != 50) {
				t.Fatalf("expected exact NPC draw 1000/1000, losses and survivors 50/50, exp 50, report=%+v", report)
			}
			standardFound := false
			if report.Detail != nil {
				for _, trait := range report.Detail.Traits {
					if trait.TraitID == "xiaobawang_zhuiji" {
						standardFound = true
					}
				}
			}
			if !tc.wantTriggered {
				if standardFound {
					t.Fatalf("expected no fake pursuit standard outcome, detail=%+v", report.Detail)
				}
				return
			}

			extra, detailOK := outcome.Detail["extraLosses"].(map[string]int)
			if !detailOK || extra["weiInfantry"] != 10 || outcome.Detail["effectRate"] != 0.1 || outcome.Detail["triggerChance"] != 1.0 || outcome.OwnerGeneralID != "sunce" {
				t.Fatalf("expected pursuit to add 10 real NPC losses, outcome=%+v", outcome)
			}
			if !standardFound {
				t.Fatalf("expected pursuit in standard NPC report, detail=%+v", report.Detail)
			}
		})
	}
}

// TestNpcFormalLossReturnTraitsRespectOutcome 验证典韦和郭嘉以正式将领身份只在 NPC 战败后返还真实兵力。
func TestNpcFormalLossReturnTraitsRespectOutcome(t *testing.T) {
	cases := []struct {
		name       string
		traitID    string
		traitType  string
		generalID  string
		rate       float64
		wantReturn int
	}{
		{name: "护主死战", traitID: "huzhu_sizhan", traitType: general.TraitTypeSpecial, generalID: "dianwei", rate: 0.15, wantReturn: 15},
		{name: "鬼才遗策", traitID: "guicai_yice", traitType: general.TraitTypeBonus, generalID: "guojia", rate: 0.1, wantReturn: 10},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			traitCfg := GeneralTraitConfig{
				TraitID: tc.traitID, TraitType: tc.traitType, Enabled: true, Scope: "self_army", RequiredOutcome: "loss",
				Params: map[string]float64{"lossReductionRate": tc.rate, "maxReturnCount": 10000, "triggerChance": 1},
			}
			hero := GeneralHeroConfig{ID: tc.generalID, Name: tc.generalID, Faction: "wei", Enabled: true}
			if tc.traitType == general.TraitTypeSpecial {
				hero.SpecialTrait = traitCfg
			} else {
				hero.BonusTrait = traitCfg
			}

			t.Run("战败触发", func(t *testing.T) {
				report, stored := resolveNpcAfterBattleHeroTest(t, tc.traitID+"_loss", hero, 100, 200)
				outcome, ok := report.TraitOutcomes[tc.traitID]
				returned, detailOK := outcome.Detail["returnedUnits"].(map[string]int)
				returnCap, capOK := traitDetailInt(outcome.Detail["maxReturnCount"])
				if !ok || !detailOK || !capOK || returned["weiInfantry"] != tc.wantReturn || outcome.Detail["lossReductionRate"] != tc.rate || returnCap != 10000 || outcome.Detail["triggerChance"] != 1.0 || outcome.OwnerGeneralID != tc.generalID {
					t.Fatalf("expected %s to return %d troops after NPC defeat, outcome=%+v", tc.traitID, tc.wantReturn, outcome)
				}
				if report.LostUnits["weiInfantry"] != 100 || report.RevivedUnits["weiInfantry"] != tc.wantReturn || report.SurvivedUnits["weiInfantry"] != tc.wantReturn || armySliceToMap(stored.Army)["weiInfantry"] != tc.wantReturn {
					t.Fatalf("expected %s loss/return/survivor 100/%d/%d, report=%+v stored=%+v", tc.traitID, tc.wantReturn, tc.wantReturn, report, stored.Army)
				}
				assertNpcAfterBattleStandardResult(t, report, tc.traitID, tc.generalID, tc.wantReturn)
			})

			t.Run("获胜不触发", func(t *testing.T) {
				report, stored := resolveNpcAfterBattleHeroTest(t, tc.traitID+"_win", hero, 200, 100)
				if _, triggered := report.TraitOutcomes[tc.traitID]; triggered || len(report.RevivedUnits) != 0 {
					t.Fatalf("expected %s not to trigger after NPC victory, outcomes=%+v returned=%+v", tc.traitID, report.TraitOutcomes, report.RevivedUnits)
				}
				if got, want := armySliceToMap(stored.Army)["weiInfantry"], 200-report.LostUnits["weiInfantry"]; got != want {
					t.Fatalf("expected winning %s army %d without return, got %d", tc.traitID, want, got)
				}
				for _, trait := range report.Detail.Traits {
					if trait.TraitID == tc.traitID {
						t.Fatalf("expected no fake %s standard outcome after victory, detail=%+v", tc.traitID, report.Detail)
					}
				}
			})

			t.Run("平局不触发", func(t *testing.T) {
				report, stored := resolveNpcAfterBattleHeroTest(t, tc.traitID+"_draw", hero, 100, 100)
				if report.Result != "draw" || report.PlayerPower != 1000 || report.EnemyPower != 1000 || report.LostUnits["weiInfantry"] != 100 || report.DefenderLostUnits["weiInfantry"] != 100 {
					t.Fatalf("expected exact NPC draw 1000/1000 and attack-rule losses 100/100, report=%+v", report)
				}
				if _, triggered := report.TraitOutcomes[tc.traitID]; triggered || len(report.RevivedUnits) != 0 || len(report.TraitTriggered) != 0 || standardReportHasTrait(report.Detail, tc.traitID) {
					t.Fatalf("expected %s not to trigger after NPC draw, report=%+v", tc.traitID, report)
				}
				if report.SurvivedUnits["weiInfantry"] != 0 || report.GeneralExpGained != 100 || armySliceToMap(stored.Army)["weiInfantry"] != 0 {
					t.Fatalf("expected NPC draw full losses and exp 100 without return, report=%+v stored=%+v", report, stored.Army)
				}
			})
		})
	}
}

// TestNpcSimaYiYibingHitAndMissExcludeDefenderTrait 验证司马懿主动进攻 NPC 时疑兵概率与防守专属谋定彼此隔离。
func TestNpcSimaYiYibingHitAndMissExcludeDefenderTrait(t *testing.T) {
	for _, tc := range []struct {
		name             string
		triggerChance    float64
		wantEnemyPower   int
		wantPlayerLosses int
		wantYibing       bool
	}{
		{name: "疑兵命中", triggerChance: 1, wantEnemyPower: 650, wantPlayerLosses: 40, wantYibing: true},
		{name: "疑兵未命中", triggerChance: 0, wantEnemyPower: 1000, wantPlayerLosses: 74},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hero := GeneralHeroConfig{
				ID: "simayi", Name: "司马懿", Faction: "wei", Enabled: true,
				SpecialTrait: GeneralTraitConfig{
					TraitID: "yibing_touxi", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army",
					Params: map[string]float64{"triggerChance": tc.triggerChance, "effectRate": 0.35, "maxAffectedRate": 0.35},
				},
				BonusTrait: GeneralTraitConfig{
					TraitID: "mouding_houfa", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "enemy_army",
					AllowedSides: []string{"defender"}, Params: map[string]float64{"effectRate": 0.1},
				},
			}
			report, stored := resolveNpcAfterBattleHeroTest(t, "simayi_"+tc.name, hero, 200, 100)
			if report.Result != "attacker_victory" || report.PlayerPower != 2000 || report.EnemyPower != tc.wantEnemyPower || report.LostUnits["weiInfantry"] != tc.wantPlayerLosses || report.DefenderLostUnits["weiInfantry"] != 100 || report.SurvivedUnits["weiInfantry"] != 200-tc.wantPlayerLosses || report.GeneralExpGained != 100 {
				t.Fatalf("expected exact Sima Yi NPC result, report=%+v", report)
			}
			if armySliceToMap(stored.Army)["weiInfantry"] != 200-tc.wantPlayerLosses || armySliceToMap(stored.NpcState.Cities[0].Army)["weiInfantry"] != 0 {
				t.Fatalf("expected Sima Yi NPC state to match report, stored=%+v npc=%+v", stored.Army, stored.NpcState.Cities[0].Army)
			}
			if report.Detail == nil || !reportSideGeneralOwnsTrait(report.Detail.PrimarySide, "simayi", "yibing_touxi") || !reportSideGeneralOwnsTrait(report.Detail.PrimarySide, "simayi", "mouding_houfa") {
				t.Fatalf("expected Sima Yi NPC snapshot to preserve both traits, detail=%+v", report.Detail)
			}
			if standardReportHasTrait(report.Detail, "mouding_houfa") {
				t.Fatalf("expected defender-only Mouding not to trigger on NPC attack, detail=%+v", report.Detail)
			}
			if !tc.wantYibing {
				if len(report.TraitTriggered) != 0 || len(report.TraitOutcomes) != 0 || len(report.Detail.Traits) != 0 {
					t.Fatalf("expected legal Yibing miss and invalid Mouding to keep empty timeline, report=%+v", report)
				}
				return
			}
			preDamage, ok := report.TraitOutcomes["yibing_touxi"].Detail["preBattleAffected"].(map[string]int)
			if !ok || preDamage["weiInfantry"] != 35 || len(report.TraitTriggered) != 1 || report.TraitTriggered[0] != "yibing_touxi" || len(report.Detail.Traits) != 1 {
				t.Fatalf("expected only Yibing to deal 35 pre-battle losses, report=%+v", report)
			}
		})
	}
}

// TestNpcDianWeiDualTraitsKeepLossReturnProbabilityIndependent 验证典韦主动加攻后战败返兵的命中与未命中不互相污染。
func TestNpcDianWeiDualTraitsKeepLossReturnProbabilityIndependent(t *testing.T) {
	for _, tc := range []struct {
		name          string
		triggerChance float64
		wantReturned  int
	}{
		{name: "护主死战命中", triggerChance: 1, wantReturned: 15},
		{name: "护主死战未命中", triggerChance: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hero := GeneralHeroConfig{
				ID: "dianwei", Name: "典韦", Faction: "wei", Enabled: true,
				SpecialTrait: GeneralTraitConfig{
					TraitID: "huzhu_sizhan", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "self_army", RequiredOutcome: "loss",
					Params: map[string]float64{"triggerChance": tc.triggerChance, "lossReductionRate": 0.15, "maxReturnCount": 10000},
				},
				BonusTrait: GeneralTraitConfig{
					TraitID: "sizhandaodi", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army", TargetUnitType: "infantry",
					AllowedSides: []string{"attacker"}, Params: map[string]float64{"attackBonusRate": 0.35},
				},
			}
			report, stored := resolveNpcAfterBattleHeroTest(t, "dianwei_dual_"+tc.name, hero, 100, 200)
			if report.Result != "defender_victory" || report.PlayerPower != 1400 || report.EnemyPower != 2000 || report.LostUnits["weiInfantry"] != 100 || report.DefenderLostUnits["weiInfantry"] != 120 || report.GeneralExpGained != 120 {
				t.Fatalf("expected exact Dian Wei NPC core result after Sizhan bonus, report=%+v", report)
			}
			if report.RevivedUnits["weiInfantry"] != tc.wantReturned || report.SurvivedUnits["weiInfantry"] != tc.wantReturned || armySliceToMap(stored.Army)["weiInfantry"] != tc.wantReturned || armySliceToMap(stored.NpcState.Cities[0].Army)["weiInfantry"] != 80 {
				t.Fatalf("expected Dian Wei NPC return and state %d, report=%+v stored=%+v", tc.wantReturned, report, stored)
			}
			if report.Detail == nil || !reportSideGeneralOwnsTrait(report.Detail.PrimarySide, "dianwei", "huzhu_sizhan") || !reportSideGeneralOwnsTrait(report.Detail.PrimarySide, "dianwei", "sizhandaodi") {
				t.Fatalf("expected Dian Wei NPC snapshot to preserve both traits, detail=%+v", report.Detail)
			}
			attackModified, ok := report.TraitOutcomes["sizhandaodi"].Detail["attackModifiedUnits"].(map[string]int)
			wantTimeline := 1
			if tc.wantReturned > 0 {
				wantTimeline = 2
			}
			if !ok || attackModified["weiInfantry"] != 4 || len(report.TraitTriggered) != wantTimeline || report.TraitTriggered[0] != "sizhandaodi" {
				t.Fatalf("expected Sizhan to add 4 attack before optional Huzhu, report=%+v", report)
			}
			if tc.wantReturned == 0 {
				if standardReportHasTrait(report.Detail, "huzhu_sizhan") || len(report.TraitOutcomes) != 1 || len(report.Detail.Traits) != 1 {
					t.Fatalf("expected legal Huzhu miss to keep only Sizhan timeline, report=%+v", report)
				}
				return
			}
			returned, returnedOK := report.TraitOutcomes["huzhu_sizhan"].Detail["returnedUnits"].(map[string]int)
			if !returnedOK || returned["weiInfantry"] != tc.wantReturned || report.TraitTriggered[1] != "huzhu_sizhan" || len(report.Detail.Traits) != 2 {
				t.Fatalf("expected Huzhu to return %d after Sizhan, report=%+v", tc.wantReturned, report)
			}
		})
	}
}

// TestNpcFormalLiubeiDualReturnsAndLongdanDirection 验证刘备双返兵叠加，并锁定赵云主动 NPC 进攻无效。
func TestNpcFormalLiubeiDualReturnsAndLongdanDirection(t *testing.T) {
	t.Run("刘备双特性叠加", func(t *testing.T) {
		hero := GeneralHeroConfig{
			ID: "liubei", Name: "刘备", Faction: "wei", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "rende", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "self_army",
				Params: map[string]float64{"effectRate": 0.5, "reviveRate": 0.5, "maxReviveCount": 10000, "triggerChance": 1},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "renzhu_shouhu", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army",
				Params: map[string]float64{"lossReductionRate": 0.1, "maxReturnCount": 10000, "triggerChance": 1},
			},
		}
		report, stored := resolveNpcAfterBattleHeroTest(t, "liubei_dual", hero, 100, 200)
		rende, rendeOK := report.TraitOutcomes["rende"].Detail["revivedUnits"].(map[string]int)
		guard, guardOK := report.TraitOutcomes["renzhu_shouhu"].Detail["returnedUnits"].(map[string]int)
		if !rendeOK || !guardOK || rende["weiInfantry"] != 50 || guard["weiInfantry"] != 10 {
			t.Fatalf("expected Liu Bei NPC returns 50 + 10, outcomes=%+v", report.TraitOutcomes)
		}
		if report.LostUnits["weiInfantry"] != 100 || report.RevivedUnits["weiInfantry"] != 60 || report.SurvivedUnits["weiInfantry"] != 60 || armySliceToMap(stored.Army)["weiInfantry"] != 60 {
			t.Fatalf("expected Liu Bei NPC loss/return/survivor 100/60/60, report=%+v stored=%+v", report, stored.Army)
		}
		assertNpcAfterBattleStandardResult(t, report, "rende", "liubei", 60)
		assertNpcAfterBattleStandardResult(t, report, "renzhu_shouhu", "liubei", 60)
	})

	t.Run("赵云主动进攻不触发", func(t *testing.T) {
		hero := GeneralHeroConfig{
			ID: "zhaoyun", Name: "赵云", Faction: "wei", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "longdan_jiuyuan", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "reinforcement_self", AllowedSides: []string{"defender", "reinforcement"},
				Params: map[string]float64{"lossReductionRate": 0.2, "triggerChance": 1},
			},
		}
		report, stored := resolveNpcAfterBattleHeroTest(t, "longdan_attack", hero, 100, 200)
		if _, triggered := report.TraitOutcomes["longdan_jiuyuan"]; triggered || len(report.RevivedUnits) != 0 || armySliceToMap(stored.Army)["weiInfantry"] != 0 {
			t.Fatalf("expected attacking Zhao Yun to lose all troops without Longdan, report=%+v stored=%+v", report, stored.Army)
		}
		for _, trait := range report.Detail.Traits {
			if trait.TraitID == "longdan_jiuyuan" {
				t.Fatalf("expected no fake Longdan in NPC attack report, detail=%+v", report.Detail)
			}
		}
	})
}

// assertNpcAfterBattleStandardResult 核对 NPC 标准战报中的特性归属与返兵后的最终存活。
func assertNpcAfterBattleStandardResult(t *testing.T, report BattleReport, traitID string, generalID string, survived int) {
	t.Helper()
	if report.Detail == nil {
		t.Fatalf("expected %s standard NPC report", traitID)
	}
	unitFound := false
	for _, unit := range report.Detail.PrimarySide.Units {
		if unit.UnitType == "weiInfantry" && unit.AmountBefore == 100 && unit.Lost == 100 && unit.Survived == survived {
			unitFound = true
		}
	}
	traitFound := false
	for _, trait := range report.Detail.Traits {
		if trait.TraitID == traitID && trait.GeneralID == generalID && trait.OwnerSide == "primary" && trait.OwnerRole == "attacker" {
			traitFound = true
		}
	}
	if !unitFound || !traitFound {
		t.Fatalf("expected %s standard NPC result to reconcile, detail=%+v", traitID, report.Detail)
	}
}

// assertNpcFormalGeneralSnapshot 核对 NPC 战报保留正式将领的完整拥有特性，但只把真实生效项放入时间线。
func assertNpcFormalGeneralSnapshot(t *testing.T, report BattleReport, generalID string, traitIDs ...string) {
	t.Helper()
	if len(report.PvpAttackerGenerals) != 1 || report.PvpAttackerGenerals[0].ID != generalID {
		t.Fatalf("expected one %s NPC general snapshot, snapshots=%+v", generalID, report.PvpAttackerGenerals)
	}
	if report.Detail == nil || report.Detail.SecondarySide == nil || len(report.Detail.PrimarySide.Generals) != 1 || report.Detail.PrimarySide.Generals[0].ID != generalID {
		t.Fatalf("expected one %s standard NPC general snapshot, detail=%+v", generalID, report.Detail)
	}
	for _, traitID := range traitIDs {
		if !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], traitID) || !standardDetailGeneralHasTrait(report.Detail, traitID) {
			t.Fatalf("expected %s snapshots to retain owned trait %s, legacy=%+v standard=%+v", generalID, traitID, report.PvpAttackerGenerals, report.Detail)
		}
	}
	if len(report.Detail.Traits) != len(report.TraitTriggered) {
		t.Fatalf("expected standard and legacy timelines to have the same length, legacy=%v standard=%+v", report.TraitTriggered, report.Detail.Traits)
	}
	for index, traitID := range report.TraitTriggered {
		trait := report.Detail.Traits[index]
		if trait.TraitID != traitID || trait.GeneralID != generalID || trait.OwnerSide != "primary" || trait.OwnerRole != "attacker" {
			t.Fatalf("expected standard timeline item %d to match %s attacker ownership, trait=%+v", index, traitID, trait)
		}
	}
}

// assertNpcFormalUnitRows 核对 NPC 标准战报的出征、原始阵亡与最终存活均和旧字段一致。
func assertNpcFormalUnitRows(t *testing.T, report BattleReport, playerUnit string, playerBefore int, playerLost int, playerSurvived int, npcUnit string, npcBefore int, npcLost int, npcSurvived int) {
	t.Helper()
	if report.Detail == nil || report.Detail.SecondarySide == nil {
		t.Fatalf("expected complete NPC standard report, detail=%+v", report.Detail)
	}
	playerRow := attackDefenseCrossReportUnit(t, report.Detail.PrimarySide, playerUnit)
	npcRow := attackDefenseCrossReportUnit(t, *report.Detail.SecondarySide, npcUnit)
	if playerRow.AmountBefore != playerBefore || playerRow.Dispatched != playerBefore || playerRow.Lost != playerLost || playerRow.Survived != playerSurvived {
		t.Fatalf("expected player standard row %d/%d/%d, row=%+v", playerBefore, playerLost, playerSurvived, playerRow)
	}
	if npcRow.AmountBefore != npcBefore || npcRow.Dispatched != npcBefore || npcRow.Lost != npcLost || npcRow.Survived != npcSurvived {
		t.Fatalf("expected NPC standard row %d/%d/%d, row=%+v", npcBefore, npcLost, npcSurvived, npcRow)
	}
}

// assertNpcLateRandomTraitDetails 核对三名战后随机将领在命中与未命中时只报告真实数值变化。
func assertNpcLateRandomTraitDetails(t *testing.T, report BattleReport, generalID string, hit bool) {
	t.Helper()
	switch generalID {
	case "huangzhong":
		_, specialTriggered := report.TraitOutcomes["baibu_chuanyang"]
		if specialTriggered != hit {
			t.Fatalf("expected Baibu triggered=%t, outcomes=%+v", hit, report.TraitOutcomes)
		}
		if hit {
			outcome := report.TraitOutcomes["baibu_chuanyang"]
			infantry := outcome.Detail["infantryDefenseModifiedUnits"].(map[string]int)
			cavalry := outcome.Detail["cavalryDefenseModifiedUnits"].(map[string]int)
			if infantry["weiInfantry"] != -2 || cavalry["weiInfantry"] != -2 || outcome.Detail["enemyDefenseReductionRate"] != 0.2 {
				t.Fatalf("expected Baibu to reduce both defenses by 2, outcome=%+v", outcome)
			}
		}
		extra := report.TraitOutcomes["laodang_yizhuang"].Detail["extraLosses"].(map[string]int)
		if extra["weiInfantry"] != 100 {
			t.Fatalf("expected Laodang to add exactly 100 losses, outcome=%+v", report.TraitOutcomes["laodang_yizhuang"])
		}
	case "luxun":
		_, specialTriggered := report.TraitOutcomes["huoshao_lianying"]
		if specialTriggered != hit {
			t.Fatalf("expected Huoshao Lianying triggered=%t, outcomes=%+v", hit, report.TraitOutcomes)
		}
		if hit {
			extra := report.TraitOutcomes["huoshao_lianying"].Detail["targetExtraLosses"].(map[string]int)
			if extra["weiInfantry"] != 500 || standardReportHasTrait(report.Detail, "lianying_zengshang") {
				t.Fatalf("expected fire to consume the remaining 500 and omit zero-change bonus, outcomes=%+v", report.TraitOutcomes)
			}
			return
		}
		extra := report.TraitOutcomes["lianying_zengshang"].Detail["targetExtraLosses"].(map[string]int)
		if extra["weiInfantry"] != 100 {
			t.Fatalf("expected missed fire to leave exact 100 Lianying losses, outcome=%+v", report.TraitOutcomes["lianying_zengshang"])
		}
	case "huanggai":
		_, specialTriggered := report.TraitOutcomes["kurouji"]
		if specialTriggered != hit {
			t.Fatalf("expected Kurouji triggered=%t, outcomes=%+v", hit, report.TraitOutcomes)
		}
		if hit {
			outcome := report.TraitOutcomes["kurouji"]
			if outcome.Detail["disableTraitCount"] != 1 || outcome.Detail["disabledTraitCount"] != 0 {
				t.Fatalf("expected NPC without traits to report designed suppression 1 and actual 0, outcome=%+v", outcome)
			}
		}
		extra := report.TraitOutcomes["kurou_fanji"].Detail["extraLosses"].(map[string]int)
		if extra["weiInfantry"] != 100 {
			t.Fatalf("expected Kurou counter to add exactly 100 losses, outcome=%+v", report.TraitOutcomes["kurou_fanji"])
		}
	}
}

// resolveNpcFormalRandomTraitTest 构造正式双特性将领的真实 NPC 单城战斗并返回持久化后的权威状态。
func resolveNpcFormalRandomTraitTest(t *testing.T, suffix string, hero GeneralHeroConfig, playerUnit string, playerCount int, npcUnit string, npcCount int, mode string, normalizeNpcCavalry bool) (BattleReport, GameState) {
	t.Helper()
	setTestCombatUnitsConfig(t)
	unitsMu.Lock()
	activeUnits["wei"]["overlordRider"] = UnitConfig{
		Name: "霸王骑", Category: "cavalry",
		Stats: map[string]int{"attack": 28, "infantryDefense": 10, "cavalryDefense": 33, "carryCapacity": 130, "upkeep": 4},
	}
	if normalizeNpcCavalry {
		activeUnits["wei"]["weiCavalry"] = UnitConfig{
			Name: "魏骑兵", Category: "cavalry",
			Stats: map[string]int{"attack": 10, "infantryDefense": 10, "cavalryDefense": 8, "carryCapacity": 5, "upkeep": 1},
		}
	}
	unitsMu.Unlock()
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: hero.ID, Name: hero.Name}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{hero.ID: hero}})

	now := time.Now().UTC()
	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)
	account := Account{ID: "account_npc_formal_random_" + suffix, Username: "npc_formal_random_" + suffix, CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create formal random NPC account: %v", err)
	}
	state := newPlayerState("player_npc_formal_random_"+suffix, "NPC 正式随机特性测试", "wei", hero.ID, now)
	EnsureGeneralRoster(&state, now)
	state.Army = []ArmyUnit{{UnitType: playerUnit, Amount: playerCount}}
	npc := testNpcCity("npc_formal_random_"+suffix, now)
	npc.Army = []ArmyUnit{{UnitType: npcUnit, Amount: npcCount}}
	npc.MaxArmy = []ArmyUnit{{UnitType: npcUnit, Amount: npcCount}}
	state.NpcState = &NpcState{Cities: []NpcCity{npc}, LastRefreshedAt: now.Format(resourceDateLayout)}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create formal random NPC player: %v", err)
	}
	result, err := svc.AttackNpc(AttackNpcRequest{
		PlayerID: state.Player.ID, NpcID: npc.ID, Mode: mode,
		Units: map[string]int{playerUnit: playerCount}, GeneralIDs: []string{hero.ID},
	})
	if err != nil {
		t.Fatalf("AttackNpc formal random %s failed: %v", suffix, err)
	}
	stored, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("get formal random NPC state: %v", err)
	}
	return result.BattleReport, stored
}

// resolveNpcAfterBattleHeroTest 构造正式将领在指定 NPC 兵力下的单城战斗并返回权威状态。
func resolveNpcAfterBattleHeroTest(t *testing.T, suffix string, hero GeneralHeroConfig, playerCount int, npcCount int) (BattleReport, GameState) {
	t.Helper()
	setTestCombatUnitsConfig(t)
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: hero.ID, Name: hero.Name}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{hero.ID: hero}})
	now := time.Now().UTC()
	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)
	account := Account{ID: "account_npc_after_battle_" + suffix, Username: "npc_after_battle_" + suffix, CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create NPC after-battle account: %v", err)
	}
	state := newPlayerState("player_npc_after_battle_"+suffix, "NPC 战后返兵测试", "wei", hero.ID, now)
	state.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: playerCount}}
	npc := testNpcCity("npc_after_battle_"+suffix, now)
	npc.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: npcCount}}
	npc.MaxArmy = []ArmyUnit{{UnitType: "weiInfantry", Amount: npcCount}}
	state.NpcState = &NpcState{Cities: []NpcCity{npc}, LastRefreshedAt: now.Format(resourceDateLayout)}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create NPC after-battle player: %v", err)
	}
	result, err := svc.AttackNpc(AttackNpcRequest{
		PlayerID: state.Player.ID, NpcID: npc.ID, Mode: "attack",
		Units: map[string]int{"weiInfantry": playerCount}, GeneralIDs: []string{hero.ID},
	})
	if err != nil {
		t.Fatalf("AttackNpc after-battle %s failed: %v", suffix, err)
	}
	stored, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("get NPC after-battle state: %v", err)
	}
	return result.BattleReport, stored
}

// resolveNpcRemainingAfterCombatDamageTraitTest 构造指定正式将领、出征规模和 NPC 兵种的单城战斗。
func resolveNpcRemainingAfterCombatDamageTraitTest(t *testing.T, traitID string, generalID string, targetUnitType string, params map[string]float64, initialPlayer int, dispatched int, npcArmy []ArmyUnit) (BattleReport, GameState) {
	t.Helper()
	setTestCombatUnitsConfig(t)
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: generalID, Name: generalID}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		generalID: {
			ID: generalID, Name: generalID, Faction: "wei", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: traitID, TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army", TargetUnitType: targetUnitType,
				AllowedSides: []string{"attacker"}, Params: params,
			},
		},
	}})

	now := time.Now().UTC()
	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)
	account := Account{ID: "account_npc_remaining_" + traitID, Username: "npc_remaining_" + traitID, CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create remaining NPC damage account: %v", err)
	}
	state := newPlayerState("player_npc_remaining_"+traitID, "NPC 剩余战后特性测试", "wei", generalID, now)
	state.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: initialPlayer}}
	npc := testNpcCity("npc_remaining_"+traitID, now)
	npc.Army = append([]ArmyUnit(nil), npcArmy...)
	npc.MaxArmy = append([]ArmyUnit(nil), npcArmy...)
	state.NpcState = &NpcState{Cities: []NpcCity{npc}, LastRefreshedAt: now.Format(resourceDateLayout)}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create remaining NPC damage player: %v", err)
	}

	result, err := svc.AttackNpc(AttackNpcRequest{
		PlayerID: state.Player.ID, NpcID: npc.ID, Mode: "attack",
		Units: map[string]int{"weiInfantry": dispatched}, GeneralIDs: []string{generalID},
	})
	if err != nil {
		t.Fatalf("AttackNpc %s failed: %v", traitID, err)
	}
	stored, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("get remaining NPC damage state: %v", err)
	}
	return result.BattleReport, stored
}

// resolveNpcPursuitTraitTest 构造孙策在指定 NPC 模式和兵力下的追击正反场景。
func resolveNpcPursuitTraitTest(t *testing.T, mode string, playerCount int, npcCount int) (BattleReport, GameState) {
	t.Helper()
	setTestCombatUnitsConfig(t)
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "sunce", Name: "孙策"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"sunce": {
			ID: "sunce", Name: "孙策", Faction: "wei", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "xiaobawang_zhuiji", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army",
				AllowedScenes: []string{"plunder"}, RequiredOutcome: "win", Params: map[string]float64{"effectRate": 0.1, "triggerChance": 1},
			},
		},
	}})

	now := time.Now().UTC()
	suffix := mode
	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)
	account := Account{ID: "account_npc_pursuit_" + suffix, Username: "npc_pursuit_" + suffix, CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create NPC pursuit account: %v", err)
	}
	state := newPlayerState("player_npc_pursuit_"+suffix, "NPC 追击测试", "wei", "sunce", now)
	state.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: playerCount}}
	npc := testNpcCity("npc_pursuit_"+suffix, now)
	npc.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: npcCount}}
	npc.MaxArmy = []ArmyUnit{{UnitType: "weiInfantry", Amount: npcCount}}
	state.NpcState = &NpcState{Cities: []NpcCity{npc}, LastRefreshedAt: now.Format(resourceDateLayout)}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create NPC pursuit player: %v", err)
	}

	result, err := svc.AttackNpc(AttackNpcRequest{
		PlayerID: state.Player.ID, NpcID: npc.ID, Mode: mode,
		Units: map[string]int{"weiInfantry": playerCount}, GeneralIDs: []string{"sunce"},
	})
	if err != nil {
		t.Fatalf("AttackNpc pursuit %s failed: %v", mode, err)
	}
	stored, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("get NPC pursuit state: %v", err)
	}
	return result.BattleReport, stored
}

// resolveNpcAfterCombatDamageTraitTest 构造固定 300 对 1000 的 NPC 战斗并强制触发指定战后追加伤害。
func resolveNpcAfterCombatDamageTraitTest(t *testing.T, traitID string, traitType string, generalID string, targetUnitType string, effectRate float64) (BattleReport, GameState) {
	t.Helper()
	setTestCombatUnitsConfig(t)
	params := map[string]float64{"effectRate": effectRate, "triggerChance": 1}
	if traitID == "huoshao_lianying" {
		params["maxAffectedRate"] = 1
	}
	traitCfg := GeneralTraitConfig{
		TraitID: traitID, TraitType: traitType, Enabled: true, Scope: "enemy_army", TargetUnitType: targetUnitType, Params: params,
	}
	hero := GeneralHeroConfig{ID: generalID, Name: generalID, Faction: "wei", Enabled: true}
	if traitType == general.TraitTypeSpecial {
		hero.SpecialTrait = traitCfg
	} else {
		hero.BonusTrait = traitCfg
	}
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: generalID, Name: generalID}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{generalID: hero}})

	now := time.Now().UTC()
	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)
	account := Account{ID: "account_npc_after_" + traitID, Username: "npc_after_" + traitID, CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create NPC after-combat account: %v", err)
	}
	state := newPlayerState("player_npc_after_"+traitID, "NPC 战后特性测试", "wei", generalID, now)
	state.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 400}}
	npc := testNpcCity("npc_after_"+traitID, now)
	npc.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1000}}
	npc.MaxArmy = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1000}}
	state.NpcState = &NpcState{Cities: []NpcCity{npc}, LastRefreshedAt: now.Format(resourceDateLayout)}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create NPC after-combat player: %v", err)
	}

	result, err := svc.AttackNpc(AttackNpcRequest{
		PlayerID: state.Player.ID, NpcID: npc.ID, Mode: "attack",
		Units: map[string]int{"weiInfantry": 300}, GeneralIDs: []string{generalID},
	})
	if err != nil {
		t.Fatalf("AttackNpc %s failed: %v", traitID, err)
	}
	stored, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("get NPC after-combat state: %v", err)
	}
	return result.BattleReport, stored
}

// TestNpcJinfanQixiDoesNotTriggerInNormalAttack 验证锦帆奇袭离开掠夺场景后不会生成加成或虚假战报。
func TestNpcJinfanQixiDoesNotTriggerInNormalAttack(t *testing.T) {
	report, _ := resolveNpcAttackBonusTraitTest(t, "jinfan_qixi", "ganning", "weiInfantry", "", "attack", map[string]float64{"attackBonusRate": 0.1})
	if report.PlayerPower != 1000 {
		t.Fatalf("expected normal NPC attack power 1000 without Jinfan Qixi, got %d", report.PlayerPower)
	}
	if _, triggered := report.TraitOutcomes["jinfan_qixi"]; triggered {
		t.Fatalf("expected Jinfan Qixi not to trigger in normal NPC attack, outcomes=%+v", report.TraitOutcomes)
	}
	if report.Detail != nil {
		for _, trait := range report.Detail.Traits {
			if trait.TraitID == "jinfan_qixi" {
				t.Fatalf("expected no standard Jinfan Qixi result in normal NPC attack, detail=%+v", report.Detail)
			}
		}
	}
}

// resolveNpcAttackBonusTraitTest 构造一场固定 100 人出征的 NPC 战斗并返回战报与权威状态。
func resolveNpcAttackBonusTraitTest(t *testing.T, traitID string, generalID string, unitType string, target string, mode string, params map[string]float64, prepareUpgrade ...bool) (BattleReport, GameState) {
	t.Helper()
	setTestCombatUnitsConfig(t)
	unitsMu.Lock()
	activeUnits["wei"]["overlordRider"] = UnitConfig{
		Name: "霸王骑", Category: "cavalry",
		Stats: map[string]int{"attack": 28, "infantryDefense": 10, "cavalryDefense": 33, "carryCapacity": 130, "upkeep": 4},
	}
	unitsMu.Unlock()

	allowedScenes := []string{}
	if traitID == "jinfan_qixi" {
		allowedScenes = []string{"plunder"}
	}
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: generalID, Name: generalID}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		generalID: {
			ID: generalID, Name: generalID, Faction: "wei", Enabled: true,
			BonusTrait: GeneralTraitConfig{
				TraitID: traitID, TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army",
				TargetUnitType: target, AllowedSides: []string{"attacker"}, AllowedScenes: allowedScenes, Params: params,
			},
		},
	}})

	now := time.Now().UTC()
	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)
	account := Account{ID: "account_npc_trait_" + traitID + "_" + mode, Username: "npc_trait_" + traitID + "_" + mode, CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create NPC trait account: %v", err)
	}
	state := newPlayerState("player_npc_trait_"+traitID+"_"+mode, "NPC 特性测试", "wei", generalID, now)
	EnsureGeneralRoster(&state, now)
	if len(prepareUpgrade) > 0 && prepareUpgrade[0] {
		setPvpTestGeneralProgress(&state, generalID, 1, generalExpRequiredForLevelForTest(2)-1)
	}
	state.Army = []ArmyUnit{{UnitType: unitType, Amount: 200}}
	state.NpcState = &NpcState{
		Cities: []NpcCity{{
			ID: "npc_trait_city_" + traitID + "_" + mode, Name: "NPC 特性城", Faction: "wei",
			Resources: map[string]int{"wood": 0}, StorageCapacity: map[string]int{}, ProductionPerHour: map[string]int{},
			Army: []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}, MaxArmy: []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}},
			ResourceSettledAt: now.Format(resourceDateLayout), ArmySettledAt: now.Format(resourceDateLayout), GeneratedAt: now.Format(resourceDateLayout),
		}},
		LastRefreshedAt: now.Format(resourceDateLayout),
	}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create NPC trait player: %v", err)
	}

	result, err := svc.AttackNpc(AttackNpcRequest{
		PlayerID: state.Player.ID, NpcID: state.NpcState.Cities[0].ID, Mode: mode,
		Units: map[string]int{unitType: 100}, GeneralIDs: []string{generalID},
	})
	if err != nil {
		t.Fatalf("AttackNpc %s failed: %v", traitID, err)
	}
	stored, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("get NPC trait state: %v", err)
	}
	return result.BattleReport, stored
}

// resolveNpcEnemyDefenseTraitTest 构造固定攻防属性的 NPC 战斗并挂载一项主动破防特性。
func resolveNpcEnemyDefenseTraitTest(t *testing.T, traitID string, traitType string, generalID string, rate float64) (BattleReport, GameState) {
	t.Helper()
	setTestCombatUnitsConfig(t)
	traitCfg := GeneralTraitConfig{
		TraitID: traitID, TraitType: traitType, Enabled: true, Scope: "enemy_army", AllowedSides: []string{"attacker"},
		Params: map[string]float64{"enemyDefenseReductionRate": rate, "triggerChance": 1},
	}
	hero := GeneralHeroConfig{ID: generalID, Name: generalID, Faction: "wei", Enabled: true}
	if traitType == general.TraitTypeSpecial {
		hero.SpecialTrait = traitCfg
	} else {
		hero.BonusTrait = traitCfg
	}
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: generalID, Name: generalID}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{generalID: hero}})

	now := time.Now().UTC()
	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)
	account := Account{ID: "account_npc_break_" + traitID, Username: "npc_break_" + traitID, CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create NPC break account: %v", err)
	}
	state := newPlayerState("player_npc_break_"+traitID, "NPC 破防测试", "wei", generalID, now)
	state.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 200}}
	state.NpcState = &NpcState{
		Cities: []NpcCity{{
			ID: "npc_break_city_" + traitID, Name: "NPC 破防城", Faction: "wei",
			Resources: map[string]int{"wood": 0}, StorageCapacity: map[string]int{}, ProductionPerHour: map[string]int{},
			Army: []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}, MaxArmy: []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}},
			ResourceSettledAt: now.Format(resourceDateLayout), ArmySettledAt: now.Format(resourceDateLayout), GeneratedAt: now.Format(resourceDateLayout),
		}},
		LastRefreshedAt: now.Format(resourceDateLayout),
	}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create NPC break player: %v", err)
	}
	result, err := svc.AttackNpc(AttackNpcRequest{
		PlayerID: state.Player.ID, NpcID: state.NpcState.Cities[0].ID, Mode: "attack",
		Units: map[string]int{"weiInfantry": 100}, GeneralIDs: []string{generalID},
	})
	if err != nil {
		t.Fatalf("AttackNpc %s failed: %v", traitID, err)
	}
	stored, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("get NPC break state: %v", err)
	}
	return result.BattleReport, stored
}
