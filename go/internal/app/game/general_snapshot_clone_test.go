// 本文件验证将领特性在参战、增援和标准战报快照之间完整深拷贝。
package game

import "testing"

// TestGeneralTraitSnapshotBuildersDeepCopyNestedFields 验证所有快照构造器隔离特性嵌套字段。
func TestGeneralTraitSnapshotBuildersDeepCopyNestedFields(t *testing.T) {
	tests := []struct {
		name  string
		build func(GeneralTraitInstance) []GeneralTraitInstance
	}{
		{
			name: "玩家将领克隆",
			build: func(trait GeneralTraitInstance) []GeneralTraitInstance {
				return cloneGeneral(General{Traits: []GeneralTraitInstance{trait}}).Traits
			},
		},
		{
			name: "PVP参战快照",
			build: func(trait GeneralTraitInstance) []GeneralTraitInstance {
				return snapshotPvpGeneral(General{ID: "caocao", Traits: []GeneralTraitInstance{trait}}).Traits
			},
		},
		{
			name: "PVP快照聚合复制",
			build: func(trait GeneralTraitInstance) []GeneralTraitInstance {
				return clonePvpGeneralSnapshots([]PvpGeneralSnapshot{{ID: "caocao", Traits: []GeneralTraitInstance{trait}}})[0].Traits
			},
		},
		{
			name: "PVP标准战报转换",
			build: func(trait GeneralTraitInstance) []GeneralTraitInstance {
				return convertPvpGenerals([]PvpGeneralSnapshot{{ID: "caocao", Traits: []GeneralTraitInstance{trait}}}, "attacker")[0].Traits
			},
		},
		{
			name: "增援将领快照",
			build: func(trait GeneralTraitInstance) []GeneralTraitInstance {
				return cloneReinforcementGenerals([]ReinforcementGeneralSnapshot{{ID: "caocao", Traits: []GeneralTraitInstance{trait}}})[0].Traits
			},
		},
		{
			name: "增援标准战报转换",
			build: func(trait GeneralTraitInstance) []GeneralTraitInstance {
				report := BattleReport{ViewType: ReportViewReinforcement, PvpReinforcements: []DefenseReinforcementUnit{{
					Generals: []ReinforcementGeneralSnapshot{{ID: "caocao", Traits: []GeneralTraitInstance{trait}}},
				}}}
				return reinforcementReportGenerals(report, "reinforcement")[0].Traits
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			original := GeneralTraitInstance{
				TraitID:       "weiwu_tongyu",
				Params:        map[string]float64{"attackBonusRate": 0.1},
				AllowedSides:  []string{"attacker", "reinforcement"},
				AllowedScenes: []string{"attack"},
			}
			cloned := tc.build(original)
			cloned[0].Params["attackBonusRate"] = 9
			cloned[0].AllowedSides[0] = "defender"
			cloned[0].AllowedScenes[0] = "plunder"

			if original.Params["attackBonusRate"] != 0.1 || original.AllowedSides[0] != "attacker" || original.AllowedScenes[0] != "attack" {
				t.Fatalf("expected nested trait fields to remain isolated, original=%+v cloned=%+v", original, cloned[0])
			}
		})
	}

	original := []PvpGeneralSnapshot{{
		ID: "caocao", Stats: map[string]int{"force": 70}, EffectiveStats: map[string]int{"force": 90},
		Attributes: map[string]float64{"attack": 9}, Buffs: map[string]float64{StatAttackBonus: 0.1},
	}}
	cloned := clonePvpGeneralSnapshots(original)
	cloned[0].Stats["force"] = 7
	cloned[0].EffectiveStats["force"] = 9
	cloned[0].Attributes["attack"] = 1
	cloned[0].Buffs[StatAttackBonus] = 9
	if original[0].Stats["force"] != 70 || original[0].EffectiveStats["force"] != 90 || original[0].Attributes["attack"] != 9 || original[0].Buffs[StatAttackBonus] != 0.1 {
		t.Fatalf("expected PVP snapshot attributes and Buffs to remain isolated, original=%+v cloned=%+v", original, cloned)
	}

	standard := convertPvpGenerals(original, "attacker")
	standard[0].Stats["force"] = 6
	standard[0].EffectiveStats["force"] = 8
	standard[0].Attributes["attack"] = 2
	standard[0].Buffs[StatAttackBonus] = 8
	if original[0].Stats["force"] != 70 || original[0].EffectiveStats["force"] != 90 || original[0].Attributes["attack"] != 9 || original[0].Buffs[StatAttackBonus] != 0.1 {
		t.Fatalf("expected standard report conversion to deep-copy attributes and Buffs, original=%+v standard=%+v", original, standard)
	}

	reinforcementOriginal := []ReinforcementGeneralSnapshot{{
		ID: "machao", Stats: map[string]int{"force": 70}, EffectiveStats: map[string]int{"force": 90},
		Attributes: map[string]float64{"attack": 9}, Buffs: map[string]float64{StatAttackBonus: 0.4},
	}}
	reinforcementCloned := cloneReinforcementGenerals(reinforcementOriginal)
	reinforcementCloned[0].Stats["force"] = 7
	reinforcementCloned[0].EffectiveStats["force"] = 9
	reinforcementCloned[0].Attributes["attack"] = 1
	reinforcementCloned[0].Buffs[StatAttackBonus] = 1
	if reinforcementOriginal[0].Stats["force"] != 70 || reinforcementOriginal[0].EffectiveStats["force"] != 90 || reinforcementOriginal[0].Attributes["attack"] != 9 || reinforcementOriginal[0].Buffs[StatAttackBonus] != 0.4 {
		t.Fatalf("expected reinforcement snapshot fields to remain isolated, original=%+v cloned=%+v", reinforcementOriginal, reinforcementCloned)
	}
}
