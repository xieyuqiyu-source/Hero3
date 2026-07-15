// 本文件提供旧战报到标准战报详情的兼容转换。
package game

import (
	"sort"
	"strings"
)

const (
	ReportViewAttack        = "attack"
	ReportViewDefense       = "defense"
	ReportViewReinforcement = "reinforcement"
	ReportViewScout         = "scout"

	ReportSourceNPCCity    = "npc_city"
	ReportSourcePlayerCity = "player_city"
	ReportSourceDungeon    = "dungeon"
	ReportSourceSystem     = "system"

	ReportWinnerAttacker = "attacker"
	ReportWinnerDefender = "defender"
	ReportWinnerDraw     = "draw"
	ReportWinnerNone     = "none"

	ReportOwnerSideAttacker      = "attacker"
	ReportOwnerSideDefender      = "defender"
	ReportOwnerSideReinforcement = "reinforcement"
	ReportOwnerSideScout         = "scout"
	ReportOwnerSideObserver      = "observer"

	ReportOwnerOutcomeVictory      = "victory"
	ReportOwnerOutcomeDefeat       = "defeat"
	ReportOwnerOutcomeDraw         = "draw"
	ReportOwnerOutcomeIntelSuccess = "intel_success"
	ReportOwnerOutcomeIntelFailed  = "intel_failed"
	ReportOwnerOutcomeNotice       = "notice"
)

// NormalizeBattleReport 补齐标准战报字段，让旧战报也能按新结构展示。
func NormalizeBattleReport(report BattleReport) BattleReport {
	if report.OwnerPlayerID == "" {
		report.OwnerPlayerID = report.PlayerID
	}
	if report.EventID == "" {
		report.EventID = "event_" + report.ID
	}
	if report.BattleType == "" {
		report.BattleType = report.Type
	}
	if report.ViewType == "" {
		report.ViewType = inferReportViewType(report)
	}
	if report.SourceType == "" {
		report.SourceType = inferReportSourceType(report)
	}
	if report.WinnerSide == "" {
		report.WinnerSide = inferReportWinnerSide(report)
	}
	if report.OwnerSide == "" {
		report.OwnerSide = inferReportOwnerSide(report)
	}
	if report.OwnerOutcome == "" {
		report.OwnerOutcome = inferReportOwnerOutcome(report)
	}
	if report.Title == "" {
		report.Title = buildReportTitle(report)
	}
	if report.Summary == "" {
		report.Summary = buildReportSummary(report)
	}
	if report.Detail == nil {
		detail := BuildBattleReportDetail(report)
		report.Detail = &detail
	}
	report.Detail.Read = report.Read
	if report.Share != nil {
		report.Detail.Share = report.Share
	}
	syncBattleReportDetailOutcome(&report)
	syncBattleReportDetailGenerals(&report)
	return report
}

// BuildBattleReportDetail 从兼容战报构造标准详情结构。
func BuildBattleReportDetail(report BattleReport) BattleReportDetail {
	report.ViewType = valueOrDefault(report.ViewType, inferReportViewType(report))
	report.SourceType = valueOrDefault(report.SourceType, inferReportSourceType(report))
	report.BattleType = valueOrDefault(report.BattleType, report.Type)
	report.WinnerSide = valueOrDefault(report.WinnerSide, inferReportWinnerSide(report))
	report.OwnerSide = valueOrDefault(report.OwnerSide, inferReportOwnerSide(report))
	report.OwnerOutcome = valueOrDefault(report.OwnerOutcome, inferReportOwnerOutcome(report))
	ownerGenerals, targetGenerals := reportOwnerAndTargetGenerals(report)

	primary := BattleReportSide{
		Role:         primaryRoleForReport(report),
		PlayerID:     report.PlayerID,
		PlayerName:   report.PlayerName,
		CityName:     report.PlayerName,
		Faction:      report.PlayerFaction,
		FactionLabel: factionLabel(report.PlayerFaction),
		TargetType:   "player",
		TargetID:     report.PlayerID,
		TargetName:   report.PlayerName,
		Power:        report.PlayerPower,
		Generals:     convertPvpGenerals(ownerGenerals, primaryRoleForReport(report)),
		Units:        buildReportUnits(report.PlayerFaction, report.DispatchedUnits, report.LostUnits, nil),
	}
	secondary := &BattleReportSide{
		Role:         secondaryRoleForReport(report),
		PlayerID:     "",
		PlayerName:   report.TargetName,
		CityName:     report.TargetName,
		Faction:      report.DefenderFaction,
		FactionLabel: factionLabel(report.DefenderFaction),
		TargetType:   strings.TrimSpace(report.SourceType),
		TargetID:     report.TargetID,
		TargetName:   report.TargetName,
		Power:        report.EnemyPower,
		Generals:     convertPvpGenerals(targetGenerals, secondaryRoleForReport(report)),
		Units:        buildReportUnits(report.DefenderFaction, report.DefenderUnits, report.DefenderLostUnits, nil),
		Resources:    cloneReportIntMap(report.DefenderResources),
	}

	if report.ViewType == ReportViewDefense {
		primary, *secondary = *secondary, primary
		primary.Role = "attacker"
		secondary.Role = "defender"
	}
	if report.ViewType == ReportViewReinforcement {
		primary.Role = "reinforcement"
		primary.Generals = reinforcementReportGenerals(report, "reinforcement")
		secondary = nil
	}

	visibility := buildReportVisibility(report)
	return BattleReportDetail{
		ID:            report.ID,
		EventID:       report.EventID,
		OwnerPlayerID: valueOrDefault(report.OwnerPlayerID, report.PlayerID),
		ViewType:      report.ViewType,
		ViewLabel:     reportViewLabel(report.ViewType),
		SourceType:    report.SourceType,
		SourceLabel:   reportSourceLabel(report.SourceType),
		BattleType:    report.BattleType,
		Result:        report.Result,
		WinnerSide:    report.WinnerSide,
		OwnerSide:     report.OwnerSide,
		OwnerOutcome:  report.OwnerOutcome,
		Title:         valueOrDefault(report.Title, buildReportTitle(report)),
		Summary:       valueOrDefault(report.Summary, buildReportSummary(report)),
		OccurredAt:    report.CreatedAt,
		PrimarySide:   primary,
		SecondarySide: secondary,
		Rewards: BattleReportRewards{
			Resources:          cloneReportIntMap(report.Rewards),
			Drops:              append([]BattleReportDrop(nil), report.Drops...),
			CityGold:           report.OverflowCityGold,
			GeneralExp:         report.GeneralExpGained,
			GeneralLevelBefore: report.GeneralLevelBefore,
			GeneralLevelAfter:  report.GeneralLevelAfter,
			Overflow:           cloneReportIntMap(report.Overflow),
			Granted:            report.GrantedRewards,
		},
		Traits:     convertReportTraits(report),
		Visibility: visibility,
		Extra:      buildReportExtra(report, visibility),
		Read:       report.Read,
		Share:      report.Share,
	}
}

// inferReportViewType 根据旧字段推导玩家视角。
func inferReportViewType(report BattleReport) string {
	if report.BattleType == "scout" || report.Type == "scout" {
		return ReportViewScout
	}
	switch report.Type {
	case "reinforce", "reinforcement":
		return ReportViewReinforcement
	case "scout":
		return ReportViewAttack
	default:
		return ReportViewAttack
	}
}

// inferReportWinnerSide 从旧 result 推导客观胜方。
func inferReportWinnerSide(report BattleReport) string {
	switch report.Result {
	case "attacker_victory", "defender_defeat":
		return ReportWinnerAttacker
	case "defender_victory":
		return ReportWinnerDefender
	case "draw":
		return ReportWinnerDraw
	default:
		if report.BattleType == "scout" || report.Type == "scout" {
			return ReportWinnerNone
		}
		return ""
	}
}

// inferReportOwnerSide 从视角推导当前玩家所在侧。
func inferReportOwnerSide(report BattleReport) string {
	viewType := valueOrDefault(report.ViewType, inferReportViewType(report))
	switch viewType {
	case ReportViewDefense:
		return ReportOwnerSideDefender
	case ReportViewReinforcement:
		return ReportOwnerSideReinforcement
	case ReportViewScout:
		return ReportOwnerSideScout
	case ReportViewAttack:
		return ReportOwnerSideAttacker
	default:
		return ReportOwnerSideObserver
	}
}

// inferReportOwnerOutcome 从视角、胜方和侦查语义推导当前玩家结果。
func inferReportOwnerOutcome(report BattleReport) string {
	if report.BattleType == "scout" || report.Type == "scout" {
		if report.ViewType == ReportViewDefense {
			return ReportOwnerOutcomeNotice
		}
		if report.Result == "attacker_victory" {
			return ReportOwnerOutcomeIntelSuccess
		}
		return ReportOwnerOutcomeIntelFailed
	}
	winnerSide := valueOrDefault(report.WinnerSide, inferReportWinnerSide(report))
	ownerSide := valueOrDefault(report.OwnerSide, inferReportOwnerSide(report))
	if winnerSide == ReportWinnerDraw {
		return ReportOwnerOutcomeDraw
	}
	switch ownerSide {
	case ReportOwnerSideAttacker:
		if winnerSide == ReportWinnerAttacker {
			return ReportOwnerOutcomeVictory
		}
		if winnerSide == ReportWinnerDefender {
			return ReportOwnerOutcomeDefeat
		}
	case ReportOwnerSideDefender, ReportOwnerSideReinforcement:
		if winnerSide == ReportWinnerDefender {
			return ReportOwnerOutcomeVictory
		}
		if winnerSide == ReportWinnerAttacker {
			return ReportOwnerOutcomeDefeat
		}
	}
	if winnerSide == ReportWinnerNone {
		return ReportOwnerOutcomeNotice
	}
	return ""
}

// inferReportSourceType 根据旧目标字段推导来源类型。
func inferReportSourceType(report BattleReport) string {
	if strings.HasPrefix(report.TargetID, "player_") || len(report.PvpPointsDelta) > 0 || len(report.PvpAttackerGenerals) > 0 || len(report.PvpDefenderGenerals) > 0 {
		return ReportSourcePlayerCity
	}
	if strings.HasPrefix(report.TargetID, "npc_") || strings.Contains(strings.ToLower(report.TargetName), "npc") {
		return ReportSourceNPCCity
	}
	return ReportSourceNPCCity
}

// buildReportTitle 生成列表标题。
func buildReportTitle(report BattleReport) string {
	viewType := valueOrDefault(report.ViewType, inferReportViewType(report))
	battleType := valueOrDefault(report.BattleType, report.Type)
	action := reportViewLabel(viewType)
	if battleType == "plunder" {
		action = "掠夺"
	} else if battleType == "scout" {
		action = "侦查"
	}
	player := valueOrDefault(report.PlayerName, report.PlayerID)
	target := valueOrDefault(report.TargetName, report.TargetID)
	if player == "" {
		player = "我方"
	}
	if target == "" {
		target = "未知目标"
	}
	if viewType == ReportViewDefense {
		return target + " 攻击 " + player
	}
	if viewType == ReportViewReinforcement {
		return "增援 " + target
	}
	return player + " " + action + " " + target
}

// buildReportSummary 生成列表摘要。
func buildReportSummary(report BattleReport) string {
	result := "战斗结束"
	switch report.Result {
	case "attacker_victory":
		result = "进攻方胜利"
	case "defender_victory":
		result = "防守方胜利"
	case "draw":
		result = "双方平局"
	}
	return result
}

// buildReportUnits 生成标准兵种快照。
func buildReportUnits(faction string, dispatched map[string]int, lost map[string]int, survived map[string]int) []BattleReportUnit {
	keys := orderedReportUnitKeys(faction, dispatched, lost, survived)
	units := make([]BattleReportUnit, 0, len(keys))
	for _, unitType := range keys {
		if hiddenReportUnitType(unitType) {
			continue
		}
		unitFaction := faction
		unitName := ""
		if cfg, ok := GetUnitConfig(faction, unitType); ok {
			if cfg.Role == "transport" {
				continue
			}
			unitName = cfg.Name
		} else if foundFaction, cfg, ok := findReportUnitConfig(unitType); ok {
			if cfg.Role == "transport" {
				continue
			}
			unitFaction = foundFaction
			unitName = cfg.Name
		}
		dispatch := dispatched[unitType]
		loss := lost[unitType]
		survive := survived[unitType]
		if survived == nil {
			survive = dispatch - loss
			if survive < 0 {
				survive = 0
			}
		}
		units = append(units, BattleReportUnit{
			UnitType:     unitType,
			UnitName:     unitName,
			Faction:      unitFaction,
			AmountBefore: dispatch,
			Dispatched:   dispatch,
			Lost:         loss,
			Survived:     survive,
		})
	}
	return units
}

// orderedReportUnitKeys 按阵营配置生成全量兵种顺序，确保未参战兵种也展示 0。
func orderedReportUnitKeys(faction string, maps ...map[string]int) []string {
	seen := map[string]bool{}
	keys := make([]string, 0)
	for _, unitID := range sortedFactionUnitIDs(faction) {
		if hiddenReportUnitType(unitID) {
			continue
		}
		if cfg, ok := GetUnitConfig(faction, unitID); ok && cfg.Role == "transport" {
			continue
		}
		seen[unitID] = true
		keys = append(keys, unitID)
	}
	for _, unitID := range mergeIntMapKeys(maps...) {
		if !seen[unitID] {
			if hiddenReportUnitType(unitID) {
				continue
			}
			if _, cfg, ok := findReportUnitConfig(unitID); ok && cfg.Role == "transport" {
				continue
			}
			seen[unitID] = true
			keys = append(keys, unitID)
		}
	}
	return keys
}

// hiddenReportUnitType 判断战报中应隐藏的非战斗兵种。
func hiddenReportUnitType(unitType string) bool {
	normalized := strings.ToLower(strings.TrimSpace(unitType))
	return strings.Contains(normalized, "merchant")
}

// sortedFactionUnitIDs 返回固定兵种展示顺序：步兵、骑兵、攻城、特殊、其他。
func sortedFactionUnitIDs(faction string) []string {
	units := GetFactionUnits(faction)
	if len(units) == 0 {
		return nil
	}
	keys := make([]string, 0, len(units))
	for unitID := range units {
		keys = append(keys, unitID)
	}
	sort.SliceStable(keys, func(i, j int) bool {
		left := units[keys[i]]
		right := units[keys[j]]
		leftRank := reportUnitCategoryRank(left.Category)
		rightRank := reportUnitCategoryRank(right.Category)
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		if left.Role != right.Role {
			return left.Role < right.Role
		}
		return keys[i] < keys[j]
	})
	return keys
}

// reportUnitCategoryRank 返回战报兵种分类排序权重。
func reportUnitCategoryRank(category string) int {
	switch category {
	case "infantry":
		return 0
	case "cavalry":
		return 1
	case "siege":
		return 2
	case "special":
		return 3
	default:
		return 99
	}
}

// findReportUnitConfig 在所有阵营中查找兵种配置，用于跨阵营驻防或旧快照兜底。
func findReportUnitConfig(unitType string) (string, UnitConfig, bool) {
	units := GetUnitsConfig()
	for faction, factionUnits := range units {
		if cfg, ok := factionUnits[unitType]; ok {
			return faction, cfg, true
		}
	}
	return "", UnitConfig{}, false
}

// buildReportVisibility 计算当前视角可见性。
func buildReportVisibility(report BattleReport) BattleReportVisibility {
	visibility := BattleReportVisibility{
		ShowEnemyRemainingUnits: true,
		ShowEnemyResources:      true,
		ShowEnemyGenerals:       true,
		ShowEnemyCityDefense:    true,
		Threshold:               0.25,
	}
	if report.ViewType == ReportViewReinforcement {
		visibility.Threshold = 0
		visibility.ActualLossRatio = 0
		return visibility
	}
	if report.EnemyLossRevealThreshold > 0 {
		visibility.Threshold = report.EnemyLossRevealThreshold
	}
	visibility.ActualLossRatio = report.EnemyLossRatio
	if !report.DefenderRevealed {
		visibility.ShowEnemyRemainingUnits = false
		visibility.Reason = "enemy_remaining_hidden"
	}
	if report.Type == "scout" && report.Result != "attacker_victory" {
		visibility.ShowEnemyRemainingUnits = false
		visibility.ShowEnemyResources = false
		visibility.ShowEnemyGenerals = false
		visibility.ShowEnemyCityDefense = false
		visibility.Reason = "scout_failed"
	}
	return visibility
}

// buildReportExtra 汇总旧字段中的玩法扩展信息。
func buildReportExtra(report BattleReport, visibility BattleReportVisibility) map[string]interface{} {
	extra := map[string]interface{}{}
	if len(report.PvpPointsDelta) > 0 || len(report.PvpReinforcements) > 0 || len(report.PvpReinforcementLosses) > 0 || report.PvpWall != nil {
		pvp := map[string]interface{}{
			"pointsDelta":         report.PvpPointsDelta,
			"reinforcements":      report.PvpReinforcements,
			"reinforcementLosses": report.PvpReinforcementLosses,
			"wall":                report.PvpWall,
		}
		if report.ViewType != ReportViewReinforcement {
			pvp["enemyLossRevealThreshold"] = visibility.Threshold
			pvp["enemyLossRatio"] = visibility.ActualLossRatio
			pvp["enemyRemainingRevealed"] = visibility.ShowEnemyRemainingUnits
		}
		extra["pvp"] = pvp
	}
	if len(report.CapturedUnits) > 0 || len(report.CapturedToGarrison) > 0 {
		extra["capture"] = map[string]interface{}{
			"capturedUnits":      report.CapturedUnits,
			"capturedToGarrison": report.CapturedToGarrison,
		}
	}
	if len(report.RevivedUnits) > 0 {
		extra["revive"] = map[string]interface{}{"revivedUnits": report.RevivedUnits}
	}
	if report.Type == "scout" || report.BattleType == "scout" {
		scoutUnitType := firstReportUnitType(report.DispatchedUnits)
		counterScoutUnitType := firstReportUnitType(report.DefenderLostUnits)
		scoutSent := report.DispatchedUnits[scoutUnitType]
		scoutLost := report.LostUnits[scoutUnitType]
		counterScoutLost := report.DefenderLostUnits[counterScoutUnitType]
		extra["scout"] = map[string]interface{}{
			"success":              report.Result == "attacker_victory",
			"scoutUnitType":        scoutUnitType,
			"scoutSent":            scoutSent,
			"scoutLost":            scoutLost,
			"scoutReturned":        maxInt(0, scoutSent-scoutLost),
			"counterScoutUnitType": counterScoutUnitType,
			"counterScoutLost":     counterScoutLost,
			"revealResources":      visibility.ShowEnemyResources,
			"revealUnits":          visibility.ShowEnemyRemainingUnits,
			"reason":               visibility.Reason,
			"scoutPower":           report.PlayerPower,
			"counterScoutPower":    report.EnemyPower,
		}
	}
	if report.BattleType == "sweep" || report.Type == "sweep" {
		sweep, _ := extra["sweep"].(map[string]interface{})
		if sweep == nil {
			sweep = map[string]interface{}{}
		}
		sweep["detailMode"] = "lightweight"
		extra["sweep"] = sweep
	}
	if report.SourceType == ReportSourceDungeon {
		extra["dungeon"] = map[string]interface{}{"rewardMode": "preview"}
	}
	return extra
}

// firstReportUnitType 返回兵种 map 中稳定排序后的第一个兵种。
func firstReportUnitType(units map[string]int) string {
	keys := mergeIntMapKeys(units)
	if len(keys) == 0 {
		return ""
	}
	return keys[0]
}

// convertReportTraits 将旧特性结果转为标准特性结构。
func convertReportTraits(report BattleReport) []BattleReportTrait {
	traits := make([]BattleReportTrait, 0, len(report.TraitTriggered))
	for _, traitID := range report.TraitTriggered {
		outcome := report.TraitOutcomes[traitID]
		ownerSide := reportTraitDisplaySide(report, outcome)
		traits = append(traits, BattleReportTrait{
			TraitID:     traitID,
			TraitName:   outcome.Name,
			OwnerSide:   ownerSide,
			OwnerRole:   reportTraitOwnerRole(report, outcome, ownerSide),
			GeneralID:   outcome.OwnerGeneralID,
			GeneralName: reportTraitGeneralName(report, outcome.OwnerGeneralID),
			Detail:      outcome.Detail,
		})
	}
	return traits
}

// reportTraitDisplaySide 把战斗绝对阵营映射到标准详情的参战方位置。
func reportTraitDisplaySide(report BattleReport, outcome TraitOutcomeReport) string {
	switch strings.ToLower(strings.TrimSpace(outcome.OwnerSide)) {
	case "attacker", "primary":
		return "primary"
	case "defender", "secondary":
		return "secondary"
	case "reinforcement":
		if report.ViewType == ReportViewReinforcement {
			return "primary"
		}
		return "reinforcement"
	}
	if generalID := strings.TrimSpace(outcome.OwnerGeneralID); generalID != "" {
		for _, general := range report.PvpAttackerGenerals {
			if general.ID == generalID {
				return "primary"
			}
		}
		for _, general := range report.PvpDefenderGenerals {
			if general.ID == generalID {
				return "secondary"
			}
		}
		for _, reinforcement := range report.PvpReinforcements {
			for _, general := range reinforcement.Generals {
				if general.ID == generalID {
					if report.ViewType == ReportViewReinforcement {
						return "primary"
					}
					return "reinforcement"
				}
			}
		}
	}
	switch valueOrDefault(report.OwnerSide, inferReportOwnerSide(report)) {
	case ReportOwnerSideDefender:
		return "secondary"
	case ReportOwnerSideReinforcement:
		return "primary"
	default:
		return "primary"
	}
}

// reportTraitOwnerRole 返回前端参战方使用的绝对角色。
func reportTraitOwnerRole(report BattleReport, outcome TraitOutcomeReport, displaySide string) string {
	if role := strings.ToLower(strings.TrimSpace(outcome.OwnerSide)); role == "attacker" || role == "defender" || role == "reinforcement" {
		return role
	}
	if displaySide == "secondary" {
		return "defender"
	}
	if displaySide == "reinforcement" || report.ViewType == ReportViewReinforcement {
		return "reinforcement"
	}
	return "attacker"
}

// reportTraitGeneralName 从参战快照读取触发特性的武将名称。
func reportTraitGeneralName(report BattleReport, generalID string) string {
	generalID = strings.TrimSpace(generalID)
	if generalID == "" {
		return ""
	}
	for _, general := range append(append([]PvpGeneralSnapshot(nil), report.PvpAttackerGenerals...), report.PvpDefenderGenerals...) {
		if general.ID == generalID {
			return general.Name
		}
	}
	for _, reinforcement := range report.PvpReinforcements {
		for _, general := range reinforcement.Generals {
			if general.ID == generalID {
				return general.Name
			}
		}
	}
	return ""
}

// convertPvpGenerals 将 PVP 武将快照转为标准战报武将。
func convertPvpGenerals(generals []PvpGeneralSnapshot, role string) []BattleReportGeneral {
	result := make([]BattleReportGeneral, 0, len(generals))
	for _, general := range generals {
		result = append(result, BattleReportGeneral{
			ID:     general.ID,
			Name:   general.Name,
			Level:  general.Level,
			Role:   role,
			Traits: append([]GeneralTraitInstance(nil), general.Traits...),
		})
	}
	return result
}

// reinforcementReportGenerals 将协防战报中的援军武将快照转为标准战报武将。
func reinforcementReportGenerals(report BattleReport, role string) []BattleReportGeneral {
	if report.ViewType != ReportViewReinforcement {
		return nil
	}
	result := []BattleReportGeneral{}
	for _, reinforcement := range report.PvpReinforcements {
		for _, general := range reinforcement.Generals {
			result = append(result, BattleReportGeneral{
				ID:         general.ID,
				Name:       general.Name,
				Level:      general.Level,
				Role:       role,
				Attributes: cloneFloatMap(general.Attributes),
				Traits:     append([]GeneralTraitInstance(nil), general.Traits...),
			})
		}
	}
	return result
}

// reportOwnerAndTargetGenerals 返回战报拥有者和目标两侧的武将快照。
func reportOwnerAndTargetGenerals(report BattleReport) ([]PvpGeneralSnapshot, []PvpGeneralSnapshot) {
	if report.ViewType != ReportViewDefense {
		return report.PvpAttackerGenerals, report.PvpDefenderGenerals
	}
	if report.SourceType == ReportSourceDungeon {
		if len(report.PvpDefenderGenerals) > 0 {
			return report.PvpDefenderGenerals, nil
		}
		return report.PvpAttackerGenerals, nil
	}
	return report.PvpDefenderGenerals, report.PvpAttackerGenerals
}

// syncBattleReportDetailGenerals 校正已有标准详情中的武将侧，兼容已保存的旧错位战报。
func syncBattleReportDetailGenerals(report *BattleReport) {
	if report == nil || report.Detail == nil {
		return
	}
	if report.ViewType == ReportViewReinforcement {
		report.Detail.PrimarySide.Generals = reinforcementReportGenerals(*report, "reinforcement")
		return
	}
	if len(report.PvpAttackerGenerals)+len(report.PvpDefenderGenerals) == 0 {
		return
	}
	ownerGenerals, targetGenerals := reportOwnerAndTargetGenerals(*report)
	if report.ViewType == ReportViewDefense {
		report.Detail.PrimarySide.Generals = convertPvpGenerals(targetGenerals, "attacker")
		if report.Detail.SecondarySide != nil {
			report.Detail.SecondarySide.Generals = convertPvpGenerals(ownerGenerals, "defender")
		}
		return
	}
	report.Detail.PrimarySide.Generals = convertPvpGenerals(ownerGenerals, primaryRoleForReport(*report))
	if report.Detail.SecondarySide != nil {
		report.Detail.SecondarySide.Generals = convertPvpGenerals(targetGenerals, secondaryRoleForReport(*report))
	}
}

// mergeIntMapKeys 合并多个兵种 map 的键并保持稳定顺序。
func mergeIntMapKeys(maps ...map[string]int) []string {
	seen := map[string]bool{}
	var keys []string
	for _, item := range maps {
		for key := range item {
			if !seen[key] {
				seen[key] = true
				keys = append(keys, key)
			}
		}
	}
	sort.Strings(keys)
	return keys
}

// cloneIntMap 复制整数 map，避免详情结构被调用方意外修改。
func cloneReportIntMap(input map[string]int) map[string]int {
	if len(input) == 0 {
		return map[string]int{}
	}
	output := make(map[string]int, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

// mergeReportExtraMap 合并标准战报 extra，玩法上下文覆盖同名旧值。
func mergeReportExtraMap(base map[string]interface{}, patch map[string]interface{}) map[string]interface{} {
	output := map[string]interface{}{}
	for key, value := range base {
		output[key] = value
	}
	for key, value := range patch {
		output[key] = value
	}
	return output
}

// maxInt 返回两个整数中的较大值。
func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

// valueOrDefault 返回非空字符串或默认值。
func valueOrDefault(value string, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

// primaryRoleForReport 返回标准详情上半部分角色。
func primaryRoleForReport(report BattleReport) string {
	if report.ViewType == ReportViewReinforcement {
		return "reinforcement"
	}
	return "attacker"
}

// secondaryRoleForReport 返回标准详情下半部分角色。
func secondaryRoleForReport(report BattleReport) string {
	if report.SourceType == ReportSourceNPCCity {
		return "npc"
	}
	return "defender"
}

// reportViewLabel 返回视角中文标签。
func reportViewLabel(viewType string) string {
	switch viewType {
	case ReportViewDefense:
		return "防守"
	case ReportViewReinforcement:
		return "协防"
	case ReportViewScout:
		return "侦查"
	default:
		return "进攻"
	}
}

// reportSourceLabel 返回来源中文标签。
func reportSourceLabel(sourceType string) string {
	switch sourceType {
	case ReportSourcePlayerCity:
		return "玩家"
	case ReportSourceYellowTurban:
		return "黄巾"
	case "stronghold":
		return "据点"
	case "dungeon":
		return "副本"
	case "resource_point":
		return "资源点"
	case "event_target":
		return "活动"
	case "world_boss":
		return "世界 Boss"
	case ReportSourceSystem:
		return "系统"
	default:
		return "NPC"
	}
}

// syncBattleReportDetailOutcome 校正已有详情中的新结果语义，兼容已保存的旧战报。
func syncBattleReportDetailOutcome(report *BattleReport) {
	if report == nil || report.Detail == nil {
		return
	}
	report.Detail.WinnerSide = valueOrDefault(report.Detail.WinnerSide, report.WinnerSide)
	report.Detail.OwnerSide = valueOrDefault(report.Detail.OwnerSide, report.OwnerSide)
	report.Detail.OwnerOutcome = valueOrDefault(report.Detail.OwnerOutcome, report.OwnerOutcome)
}

// factionLabel 返回阵营中文标签。
func factionLabel(faction string) string {
	switch faction {
	case "wei":
		return "魏"
	case "shu":
		return "蜀"
	case "wu":
		return "吴"
	default:
		return faction
	}
}
