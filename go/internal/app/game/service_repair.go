// 本文件归口玩家核心资产显式修复入口，避免普通查询请求承担修复职责。
package game

import (
	"strings"
	"time"
)

type PlayerCoreAssetsRepairResult struct {
	State    GameState `json:"state"`
	Changed  bool      `json:"changed"`
	Repaired []string  `json:"repaired,omitempty"`
}

// RepairPlayerCoreAssets 显式修复旧玩家缺失的核心资产，普通 GetState 不再执行这些修复。
func (s *Service) RepairPlayerCoreAssets(playerID string) (PlayerCoreAssetsRepairResult, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return PlayerCoreAssetsRepairResult{}, ErrPlayerNotFound
	}

	now := time.Now()
	mailCode, err := s.repairMailCodeBeforeTransaction(playerID)
	if err != nil {
		return PlayerCoreAssetsRepairResult{}, err
	}
	repaired := []string{}
	state, err := s.repo.UpdatePlayerState(playerID, now, func(state *GameState) error {
		if repairMissingMailCode(state, mailCode) {
			repaired = append(repaired, "mail_code")
		}
		if repairDefaultGeneral(state, now) {
			repaired = append(repaired, "general")
		}
		if EnsureCoreBuildings(state) {
			repaired = append(repaired, "core_buildings")
		}
		if ApplyConstructionBureauResourceSlots(state, now) {
			repaired = append(repaired, "construction_resource_slots")
		}
		if state.Inventory == nil {
			state.Inventory = map[string]ItemStack{}
			repaired = append(repaired, "inventory")
		}
		if repairGeneralRoster(state, now) {
			repaired = append(repaired, "general_roster")
		}
		return nil
	})
	if err != nil {
		return PlayerCoreAssetsRepairResult{}, err
	}

	hydrateStateForResponse(&state, now)
	return PlayerCoreAssetsRepairResult{State: state, Changed: len(repaired) > 0, Repaired: repaired}, nil
}

// repairMailCodeBeforeTransaction 在写事务外生成信函编码，避免事务回调内再次查询仓储。
func (s *Service) repairMailCodeBeforeTransaction(playerID string) (string, error) {
	state, err := s.repo.GetState(playerID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(state.Player.MailCode) != "" {
		return "", nil
	}
	return s.generateMailCode(state.Player.Nickname)
}

// repairMissingMailCode 为旧玩家补齐预生成的信函编码。
func repairMissingMailCode(state *GameState, mailCode string) bool {
	if state == nil || strings.TrimSpace(state.Player.MailCode) != "" {
		return false
	}
	mailCode = strings.TrimSpace(mailCode)
	if mailCode == "" {
		return false
	}
	state.Player.MailCode = mailCode
	return true
}

// repairDefaultGeneral 为缺少主将的旧玩家补齐阵营默认武将。
func repairDefaultGeneral(state *GameState, now time.Time) bool {
	if state == nil || state.General != nil || strings.TrimSpace(state.Player.Faction) == "" {
		return false
	}
	factions := GetFactionsConfig()
	faction, ok := factions[state.Player.Faction]
	generalID := ""
	if ok {
		generalID = defaultGeneralForFaction(state.Player.Faction, faction)
	}
	if generalID == "" {
		defaultGenerals := map[string]string{
			"wei": "caocao",
			"shu": "liubei",
			"wu":  "sunquan",
		}
		generalID = defaultGenerals[state.Player.Faction]
	}
	if strings.TrimSpace(generalID) == "" {
		return false
	}
	state.General = newGeneral(state.Player.Faction, generalID)
	EnsureGeneralRoster(state, now)
	return true
}

// repairGeneralRoster 为旧单武将存档补齐多武将名册和主将占用。
func repairGeneralRoster(state *GameState, now time.Time) bool {
	if state == nil {
		return false
	}
	beforeGenerals := len(state.Generals)
	beforeAssignments := len(state.GeneralAssignments)
	beforeGeneralID := ""
	if state.General != nil {
		beforeGeneralID = state.General.ID
	}
	EnsureGeneralRoster(state, now)
	afterGeneralID := ""
	if state.General != nil {
		afterGeneralID = state.General.ID
	}
	return beforeGenerals != len(state.Generals) ||
		beforeAssignments != len(state.GeneralAssignments) ||
		beforeGeneralID != afterGeneralID
}
