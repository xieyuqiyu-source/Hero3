package game

import "time"

// 本文件归口武将配置装配、属性拆解和 Modifier 来源。

// applyHeroConfigToGeneral 根据配置把 buffs、成长属性和特性实例注入到武将。
func applyHeroConfigToGeneral(g *General) {
	if g == nil {
		return
	}
	if g.Level <= 0 {
		g.Level = 1
	}
	if g.Level > GeneralMaxLevel {
		g.Level = GeneralMaxLevel
	}

	cfg := GetGeneralsConfig()
	promoteGeneralByExp(g)
	g.NextLevelExp = nextGeneralLevelExp(g.Level)
	g.Stats = normalizeGeneralStats(g.Stats)
	g.AvailableStatPoints = availableGeneralStatPoints(g.Level, g.Stats)
	g.Attributes = map[string]float64{}
	g.AttributeBreakdown = map[string][]GeneralAttributeBreakdownItem{}
	g.Buffs = map[string]float64{}
	g.Traits = nil

	if !cfg.Enabled {
		return
	}
	hero, ok := cfg.Heroes[g.ID]
	if !ok || !hero.Enabled {
		return
	}

	for k, v := range generalLevelAttributes(g.Level) {
		addGeneralAttributeWithSource(g, k, v, "等级成长")
	}
	addGeneralStatAttributesWithBreakdown(g)
	for k, v := range hero.Buffs {
		addGeneralAttributeWithSource(g, k, v, "将领固定")
	}
	for k, v := range g.Attributes {
		g.Buffs[k] = v
	}

	for _, tc := range hero.Traits {
		if !tc.Enabled {
			continue
		}
		params := make(map[string]float64, len(tc.Params))
		for k, v := range tc.Params {
			params[k] = v
		}
		g.Traits = append(g.Traits, GeneralTraitInstance{
			TraitID: tc.TraitID,
			Name:    tc.TraitID,
			Params:  params,
		})
	}
}

// addGeneralAttributeWithSource 增加武将属性并记录来源。
func addGeneralAttributeWithSource(g *General, key string, value float64, source string) {
	if g == nil || key == "" || value == 0 {
		return
	}
	addGeneralAttribute(g.Attributes, key, value)
	if g.AttributeBreakdown == nil {
		g.AttributeBreakdown = map[string][]GeneralAttributeBreakdownItem{}
	}
	g.AttributeBreakdown[key] = append(g.AttributeBreakdown[key], GeneralAttributeBreakdownItem{
		Source: source,
		Value:  value,
	})
}

// addGeneralStatAttributesWithBreakdown 把武将属性点转换为带来源的属性加成。
func addGeneralStatAttributesWithBreakdown(g *General) {
	if g == nil || len(g.Stats) == 0 {
		return
	}
	addGeneralAttributeWithSource(g, StatAttackBonus, float64(g.Stats["force"])*GeneralStatPercentPerPoint, "武力")
	addGeneralAttributeWithSource(g, StatRecruitSpeedBonus, float64(g.Stats["intelligence"])*GeneralStatPercentPerPoint, "智谋")
	addGeneralAttributeWithSource(g, StatMarchSpeedBonus, float64(g.Stats["intelligence"])*GeneralStatPercentPerPoint, "智谋")
	addGeneralAttributeWithSource(g, StatProductionBonus, float64(g.Stats["politics"])*GeneralStatPercentPerPoint, "内政")
	addGeneralAttributeWithSource(g, StatCapacityBonus, float64(g.Stats["politics"])*GeneralStatPercentPerPoint, "内政")
	addGeneralAttributeWithSource(g, StatDefenseBonus, float64(g.Stats["command"])*GeneralStatPercentPerPoint, "统率")
}

// addGeneralAttribute 累加武将属性并限制最大值。
func addGeneralAttribute(attrs map[string]float64, key string, value float64) {
	if attrs == nil || key == "" || value == 0 {
		return
	}
	attrs[key] += value
	if attrs[key] > maxGeneralAttributeBonus {
		attrs[key] = maxGeneralAttributeBonus
	}
}

// GeneralModifierSource 将武将属性转换为 Modifier 管线来源。
type GeneralModifierSource struct {
	General *General
}

// SourceName 返回武将加成来源名称。
func (g *GeneralModifierSource) SourceName() string { return "将领" }

// ExpiresAt 返回武将加成过期时间，武将常驻加成没有单独过期时间。
func (g *GeneralModifierSource) ExpiresAt() []time.Time { return nil }

// Modifiers 返回武将当前提供的 Modifier。
func (g *GeneralModifierSource) Modifiers(now time.Time) []Modifier {
	if g.General == nil || g.General.Buffs == nil {
		return nil
	}
	mods := make([]Modifier, 0, len(g.General.Buffs))
	for key, value := range g.General.Buffs {
		if value == 0 {
			continue
		}
		mods = append(mods, Modifier{
			Key:   key,
			Value: value,
			Mode:  "percentAdd",
		})
	}
	return mods
}
