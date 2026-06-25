package game

import (
	"strings"
	"time"

	"hero3/internal/core/combat"
)

const (
	GeneralMaxLevel            = 100
	GeneralMaxStatPointsPerKey = 100
	GeneralLevelPercentAtMax   = 2.0
	GeneralStatPercentPerPoint = 0.02
	GeneralResetStatsGoldCost  = 10
	generalExpQuadraticFactor  = 50
	generalExpQuarticFactor    = 500
)

var generalStatKeys = []string{"force", "intelligence", "politics", "command"}

type generalExpResult struct {
	Gained      int
	LevelBefore int
	LevelAfter  int
}

type GeneralActionResult struct {
	State       GameState `json:"state"`
	AccountGold int       `json:"accountGold"`
}

func applyGeneralBattleExp(g *General, gained int) generalExpResult {
	if g == nil {
		return generalExpResult{}
	}
	if gained <= 0 {
		applyHeroConfigToGeneral(g)
		return generalExpResult{}
	}

	if g.Level <= 0 {
		g.Level = 1
	}
	before := g.Level
	g.Exp += gained
	promoteGeneralByExp(g)
	applyHeroConfigToGeneral(g)

	return generalExpResult{
		Gained:      gained,
		LevelBefore: before,
		LevelAfter:  g.Level,
	}
}

func promoteGeneralByExp(g *General) {
	if g == nil {
		return
	}
	if g.Level <= 0 {
		g.Level = 1
	}
	for g.Level < GeneralMaxLevel && g.Exp >= generalExpRequiredForLevel(g.Level+1) {
		g.Level++
	}
	if g.Level > GeneralMaxLevel {
		g.Level = GeneralMaxLevel
	}
}

func nextGeneralLevelExp(level int) int {
	if level >= GeneralMaxLevel {
		return 0
	}
	return generalExpRequiredForLevel(level + 1)
}

func generalExpRequiredForLevel(level int) int {
	if level <= 1 {
		return 0
	}
	if level > GeneralMaxLevel {
		level = GeneralMaxLevel
	}
	cfg := GetGeneralsConfig()
	if idx := level - 1; idx >= 0 && idx < len(cfg.Common.ExpCurve) {
		return cfg.Common.ExpCurve[idx]
	}
	n := level - 1
	n2 := n * n
	n4 := n2 * n2
	return generalExpQuadraticFactor*n2 + generalExpQuarticFactor*n4
}

func calculateGeneralBattleExpFromLosses(faction string, losses []combat.UnitLoss) int {
	if len(losses) == 0 {
		return 0
	}
	units := GetFactionUnits(faction)
	total := 0
	for _, loss := range losses {
		if loss.Losses <= 0 {
			continue
		}
		upkeep := 0
		if unit, ok := units[loss.ID]; ok {
			upkeep = unit.Stats["upkeep"]
		}
		if upkeep <= 0 {
			continue
		}
		total += loss.Losses * upkeep
	}
	return total
}

func generalLevelAttributes(level int) map[string]float64 {
	if level <= 1 {
		return nil
	}
	if level > GeneralMaxLevel {
		level = GeneralMaxLevel
	}
	ratio := float64(level-1) / float64(GeneralMaxLevel-1)
	value := ratio * GeneralLevelPercentAtMax
	return map[string]float64{
		StatAttackBonus:  value,
		StatDefenseBonus: value,
	}
}

func normalizeGeneralStats(stats map[string]int) map[string]int {
	result := make(map[string]int, len(generalStatKeys))
	for _, key := range generalStatKeys {
		value := 0
		if stats != nil {
			value = stats[key]
		}
		if value < 0 {
			value = 0
		}
		if value > GeneralMaxStatPointsPerKey {
			value = GeneralMaxStatPointsPerKey
		}
		result[key] = value
	}
	return result
}

func availableGeneralStatPoints(level int, stats map[string]int) int {
	if level <= 0 {
		level = 1
	}
	if level > GeneralMaxLevel {
		level = GeneralMaxLevel
	}
	total := level
	for _, key := range generalStatKeys {
		total -= stats[key]
	}
	if total < 0 {
		return 0
	}
	return total
}

func generalStatAttributes(stats map[string]int) map[string]float64 {
	if len(stats) == 0 {
		return nil
	}

	attrs := map[string]float64{}
	addGeneralAttribute(attrs, StatAttackBonus, float64(stats["force"])*GeneralStatPercentPerPoint)
	addGeneralAttribute(attrs, StatRecruitSpeedBonus, float64(stats["intelligence"])*GeneralStatPercentPerPoint)
	addGeneralAttribute(attrs, StatMarchSpeedBonus, float64(stats["intelligence"])*GeneralStatPercentPerPoint)
	addGeneralAttribute(attrs, StatProductionBonus, float64(stats["politics"])*GeneralStatPercentPerPoint)
	addGeneralAttribute(attrs, StatCapacityBonus, float64(stats["politics"])*GeneralStatPercentPerPoint)
	addGeneralAttribute(attrs, StatDefenseBonus, float64(stats["command"])*GeneralStatPercentPerPoint)
	return attrs
}

func generalLevelAttributesForTest(level int) map[string]float64 {
	return generalLevelAttributes(level)
}

func generalExpRequiredForLevelForTest(level int) int {
	return generalExpRequiredForLevel(level)
}

func (s *Service) AllocateGeneralStat(playerID string, statKey string) (GameState, error) {
	playerID = strings.TrimSpace(playerID)
	statKey = strings.TrimSpace(statKey)
	if playerID == "" {
		return GameState{}, ErrPlayerNotFound
	}
	if !isValidGeneralStatKey(statKey) {
		return GameState{}, ErrInvalidStatKey
	}

	now := time.Now()
	state, err := s.repo.UpdatePlayerState(playerID, now, func(state *GameState) error {
		nextState, _ := settleResources(*state, now)
		*state = nextState
		if state.General == nil {
			return ErrGeneralNotFound
		}

		applyHeroConfigToGeneral(state.General)
		if state.General.Stats[statKey] >= GeneralMaxStatPointsPerKey {
			return ErrStatMaxLevel
		}
		if state.General.AvailableStatPoints <= 0 {
			return ErrNoStatPoints
		}

		state.General.Stats[statKey]++
		applyHeroConfigToGeneral(state.General)
		refreshGeneralDerivedState(state, now)
		return nil
	})
	if err != nil {
		return GameState{}, err
	}
	return state, nil
}

func (s *Service) ResetGeneralStats(playerID string) (GeneralActionResult, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return GeneralActionResult{}, ErrPlayerNotFound
	}

	now := time.Now()
	accountID, err := s.repo.GetAccountIDByPlayerID(playerID)
	if err != nil {
		return GeneralActionResult{}, err
	}
	account, state, err := s.repo.UpdateAccountPlayerState(accountID, playerID, now, func(account *Account, state *GameState) error {
		nextState, _ := settleResources(*state, now)
		*state = nextState
		if state.General == nil {
			return ErrGeneralNotFound
		}
		if account.Gold < GeneralResetStatsGoldCost {
			return ErrInsufficientGold
		}
		account.Gold -= GeneralResetStatsGoldCost
		state.General.Stats = map[string]int{}
		applyHeroConfigToGeneral(state.General)
		refreshGeneralDerivedState(state, now)
		return nil
	})
	if err != nil {
		return GeneralActionResult{}, err
	}

	s.recordLedger(GoldLedgerEntry{
		AccountID:    accountID,
		PlayerID:     playerID,
		Currency:     LedgerCurrencyGold,
		Direction:    LedgerDirectionDebit,
		Amount:       GeneralResetStatsGoldCost,
		BalanceAfter: account.Gold,
		RefType:      LedgerRefGeneralReset,
		RefID:        state.General.ID,
		Reason:       "将领洗点",
	})
	s.publishCurrencyChanged(playerID, accountID, state.General.ID, LedgerRefGeneralReset)
	s.publishEvent(GameEvent{
		Type:      EventGeneralChanged,
		PlayerID:  playerID,
		AccountID: accountID,
		RefType:   LedgerRefGeneralReset,
		RefID:     state.General.ID,
		Payload: map[string]any{
			"action": "reset_stats",
		},
		CreatedAt: now.UTC().Format(resourceDateLayout),
	})

	return GeneralActionResult{State: state, AccountGold: account.Gold}, nil
}

func (s *Service) ChangeGeneral(playerID string, generalID string, itemID string) (GeneralActionResult, error) {
	playerID = strings.TrimSpace(playerID)
	generalID = strings.TrimSpace(generalID)
	itemID = strings.TrimSpace(itemID)
	if playerID == "" {
		return GeneralActionResult{}, ErrPlayerNotFound
	}
	if generalID == "" {
		return GeneralActionResult{}, ErrInvalidGeneral
	}

	now := time.Now()
	state, err := s.repo.UpdatePlayerState(playerID, now, func(state *GameState) error {
		nextState, _ := settleResources(*state, now)
		*state = nextState
		if state.General == nil {
			return ErrGeneralNotFound
		}
		if state.General.ID == generalID {
			return ErrInvalidGeneral
		}
		if !isGeneralAllowedForFaction(state.Player.Faction, generalID) {
			return ErrInvalidGeneral
		}
		hero, ok := GetHeroConfig(generalID)
		if !ok || !hero.Enabled || hero.Faction != state.Player.Faction {
			return ErrInvalidGeneral
		}

		if itemID != "" {
			if _, ok := GetItemDefinition(itemID); !ok {
				return ErrItemNotFound
			}
			if !consumeItemFromInventory(state, itemID, 1, now) {
				return ErrInsufficientItem
			}
		}

		state.General.ID = generalID
		state.General.Name = hero.Name
		state.General.Stats = map[string]int{}
		applyHeroConfigToGeneral(state.General)
		refreshGeneralDerivedState(state, now)
		return nil
	})
	if err != nil {
		return GeneralActionResult{}, err
	}

	accountGold := 0
	if accountID, err := s.repo.GetAccountIDByPlayerID(playerID); err == nil {
		if account, err := s.repo.GetAccountByID(accountID); err == nil {
			accountGold = account.Gold
		}
	}
	return GeneralActionResult{State: state, AccountGold: accountGold}, nil
}

func isGeneralAllowedForFaction(faction string, generalID string) bool {
	factions := GetFactionsConfig()
	fc, ok := factions[faction]
	if !ok {
		return false
	}
	for _, g := range fc.Generals {
		if g.ID == generalID {
			return true
		}
	}
	return false
}

func refreshGeneralDerivedState(state *GameState, now time.Time) {
	if state == nil {
		return
	}
	modSources := CollectModifierSources(state)
	production := calculateResourceProduction(state.Buildings, state.General)
	state.ResourceProduction = applyProductionModifiers(production, now, modSources)
	capacity := calculateResourceCapacity(state.Buildings)
	state.Resources.Capacity = applyCapacityModifiers(capacity, now, modSources)
	state.ActiveModifiers = GetModifierBreakdown(state, now)
	state.ServerTime = now.UTC().Format(resourceDateLayout)
}

func isValidGeneralStatKey(key string) bool {
	for _, statKey := range generalStatKeys {
		if key == statKey {
			return true
		}
	}
	return false
}
