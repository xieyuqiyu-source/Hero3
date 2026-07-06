// 本文件归口玩家局部视图查询，供前端按页面读取必要字段。
package game

import (
	"strings"
	"time"
)

type PlayerSummaryView struct {
	Player             Player  `json:"player"`
	CityGold           FlexInt `json:"cityGold"`
	UnreadMessageCount int     `json:"unreadMessageCount"`
	UnreadMailCount    int     `json:"unreadMailCount"`
	ServerTime         string  `json:"serverTime"`
}

type CityView struct {
	Player             Player                  `json:"player"`
	Buildings          []Building              `json:"buildings"`
	ResourceSlots      []ResourceSlot          `json:"resourceSlots,omitempty"`
	Resources          ResourceState           `json:"resources"`
	ResourceProduction ResourceProduction      `json:"resourceProduction"`
	CityGold           FlexInt                 `json:"cityGold"`
	ActiveModifiers    []ModifierBreakdownItem `json:"activeModifiers,omitempty"`
	ServerTime         string                  `json:"serverTime"`
}

type ResourceView struct {
	Resources          ResourceState           `json:"resources"`
	ResourceProduction ResourceProduction      `json:"resourceProduction"`
	ResourceSettledAt  string                  `json:"resourceSettledAt"`
	ProductionBoost    int                     `json:"productionBoost,omitempty"`
	ProductionBoostEnd string                  `json:"productionBoostEnd,omitempty"`
	CapacityBoost      int                     `json:"capacityBoost,omitempty"`
	CapacityBoostEnd   string                  `json:"capacityBoostEnd,omitempty"`
	ActiveModifiers    []ModifierBreakdownItem `json:"activeModifiers,omitempty"`
	ServerTime         string                  `json:"serverTime"`
}

type MilitaryView struct {
	Army               []ArmyUnit              `json:"army"`
	RecruitQueues      []RecruitQueue          `json:"recruitQueues"`
	Resources          ResourceState           `json:"resources"`
	CityGold           FlexInt                 `json:"cityGold"`
	Buildings          []Building              `json:"buildings,omitempty"`
	ActiveModifiers    []ModifierBreakdownItem `json:"activeModifiers,omitempty"`
	General            *General                `json:"general"`
	Generals           []General               `json:"generals,omitempty"`
	GeneralAssignments []GeneralAssignment     `json:"generalAssignments,omitempty"`
	FoodPressure       FoodPressureState       `json:"foodPressure"`
	ServerTime         string                  `json:"serverTime"`
}

type InventoryView struct {
	Inventory      map[string]ItemStack `json:"inventory"`
	InventorySlots []ItemStack          `json:"inventorySlots,omitempty"`
	ServerTime     string               `json:"serverTime"`
}

type GeneralsView struct {
	General            *General                `json:"general"`
	Generals           []General               `json:"generals,omitempty"`
	GeneralAssignments []GeneralAssignment     `json:"generalAssignments,omitempty"`
	GeneralChangeUntil string                  `json:"generalChangeUntil,omitempty"`
	ActiveModifiers    []ModifierBreakdownItem `json:"activeModifiers,omitempty"`
	ServerTime         string                  `json:"serverTime"`
}

type CityActionResult struct {
	Building           *Building               `json:"building,omitempty"`
	Buildings          []Building              `json:"buildings,omitempty"`
	ResourceSlots      []ResourceSlot          `json:"resourceSlots,omitempty"`
	Resources          ResourceState           `json:"resources"`
	ResourceProduction ResourceProduction      `json:"resourceProduction"`
	CityGold           FlexInt                 `json:"cityGold"`
	ActiveModifiers    []ModifierBreakdownItem `json:"activeModifiers,omitempty"`
	Upgraded           int                     `json:"upgraded,omitempty"`
	Cost               int                     `json:"cost,omitempty"`
	ServerTime         string                  `json:"serverTime"`
}

type ResourceActionResult struct {
	Resources          ResourceState           `json:"resources"`
	ResourceProduction ResourceProduction      `json:"resourceProduction"`
	ResourceSettledAt  string                  `json:"resourceSettledAt"`
	ProductionBoost    int                     `json:"productionBoost,omitempty"`
	ProductionBoostEnd string                  `json:"productionBoostEnd,omitempty"`
	CapacityBoost      int                     `json:"capacityBoost,omitempty"`
	CapacityBoostEnd   string                  `json:"capacityBoostEnd,omitempty"`
	ActiveModifiers    []ModifierBreakdownItem `json:"activeModifiers,omitempty"`
	CityGold           FlexInt                 `json:"cityGold"`
	Cost               int                     `json:"cost,omitempty"`
	ServerTime         string                  `json:"serverTime"`
}

type MilitaryActionResult struct {
	Army          []ArmyUnit     `json:"army"`
	RecruitQueues []RecruitQueue `json:"recruitQueues"`
	Resources     ResourceState  `json:"resources"`
	CityGold      FlexInt        `json:"cityGold"`
	ServerTime    string         `json:"serverTime"`
}

type GeneralViewActionResult struct {
	General            *General                `json:"general"`
	Generals           []General               `json:"generals,omitempty"`
	GeneralAssignments []GeneralAssignment     `json:"generalAssignments,omitempty"`
	GeneralChangeUntil string                  `json:"generalChangeUntil,omitempty"`
	ActiveModifiers    []ModifierBreakdownItem `json:"activeModifiers,omitempty"`
	AccountGold        int                     `json:"accountGold"`
	ServerTime         string                  `json:"serverTime"`
}

type CurrencyActionResult struct {
	CityGold       FlexInt `json:"cityGold"`
	AccountGold    int     `json:"accountGold,omitempty"`
	LastExchangeAt string  `json:"lastExchangeAt,omitempty"`
	ServerTime     string  `json:"serverTime"`
}

type ReportActionResult struct {
	UnreadMessageCount int    `json:"unreadMessageCount"`
	ServerTime         string `json:"serverTime"`
}

type ItemActionResult struct {
	Inventory          map[string]ItemStack    `json:"inventory,omitempty"`
	InventorySlots     []ItemStack             `json:"inventorySlots,omitempty"`
	Resources          *ResourceState          `json:"resources,omitempty"`
	Army               []ArmyUnit              `json:"army,omitempty"`
	General            *General                `json:"general,omitempty"`
	Generals           []General               `json:"generals,omitempty"`
	GeneralAssignments []GeneralAssignment     `json:"generalAssignments,omitempty"`
	ActiveModifiers    []ModifierBreakdownItem `json:"activeModifiers,omitempty"`
	Buffs              []Buff                  `json:"buffs,omitempty"`
	CityGold           *FlexInt                `json:"cityGold,omitempty"`
	ServerTime         string                  `json:"serverTime"`
}

type GarrisonActionResult struct {
	Army               []ArmyUnit          `json:"army,omitempty"`
	Generals           []General           `json:"generals,omitempty"`
	GeneralAssignments []GeneralAssignment `json:"generalAssignments,omitempty"`
	ServerTime         string              `json:"serverTime"`
}

// GetPlayerSummaryView 返回玩家摘要视图。
func (s *Service) GetPlayerSummaryView(playerID string) (PlayerSummaryView, error) {
	return s.repo.GetPlayerSummaryView(playerID)
}

// GetCityView 返回城池、建筑和资源田视图。
func (s *Service) GetCityView(playerID string) (CityView, error) {
	return s.repo.GetCityView(playerID)
}

// GetResourceView 返回资源栏和产量视图。
func (s *Service) GetResourceView(playerID string) (ResourceView, error) {
	return s.repo.GetResourceView(playerID)
}

// GetMilitaryView 返回军事视图。
func (s *Service) GetMilitaryView(playerID string) (MilitaryView, error) {
	now := time.Now().UTC()
	if err := s.SettleDuePvpMarches(playerID); err != nil {
		return MilitaryView{}, err
	}
	if err := s.SettleDueYellowTurbanMarches(playerID); err != nil {
		return MilitaryView{}, err
	}
	if _, err := s.settlePlayerProduction(playerID, now); err != nil {
		return MilitaryView{}, err
	}
	view, err := s.repo.GetMilitaryView(playerID)
	if err != nil {
		return MilitaryView{}, err
	}
	state, err := s.repo.GetState(playerID)
	if err != nil {
		return MilitaryView{}, err
	}
	view.FoodPressure = CalculateFoodPressure(state, GetYellowTurbanConfig())
	return view, nil
}

// settlePlayerProduction 把离线资源和产兵类武将特性结算进权威资产表。
func (s *Service) settlePlayerProduction(playerID string, now time.Time) (GameState, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return GameState{}, ErrPlayerNotFound
	}
	return s.repo.UpdatePlayerState(playerID, now, func(state *GameState) error {
		EnsureGeneralRoster(state, now)
		if !hasPendingGuardProduction(state, now) {
			return nil
		}
		next, _ := settleResources(*state, now)
		*state = next
		return nil
	})
}

// hasPendingGuardProduction 判断主将产兵特性是否已经累积到至少 1 个兵。
func hasPendingGuardProduction(state *GameState, now time.Time) bool {
	if state == nil || state.General == nil {
		return false
	}
	settledAtText := strings.TrimSpace(state.ResourceSettledAt)
	if settledAtText == "" {
		return false
	}
	settledAt, err := time.Parse(resourceDateLayout, settledAtText)
	if err != nil {
		return false
	}
	elapsedSeconds := now.UTC().Sub(settledAt.UTC()).Seconds()
	if elapsedSeconds <= 0 {
		return false
	}
	generalCopy := cloneGeneral(*state.General)
	applyHeroConfigToGeneral(&generalCopy)
	for _, trait := range generalCopy.Traits {
		perMinute := trait.Params["guardPerMinute"]
		if perMinute <= 0 {
			continue
		}
		amount := int(perMinute * elapsedSeconds / 60)
		if maxPerSettle := int(trait.Params["maxGuardPerDay"]); maxPerSettle > 0 && amount > maxPerSettle {
			amount = maxPerSettle
		}
		if amount > 0 {
			return true
		}
	}
	return false
}

// GetInventoryView 返回背包视图。
func (s *Service) GetInventoryView(playerID string) (InventoryView, error) {
	return s.repo.GetInventoryView(playerID)
}

// GetGeneralsView 返回武将与派驻视图。
func (s *Service) GetGeneralsView(playerID string) (GeneralsView, error) {
	return s.repo.GetGeneralsView(playerID)
}

// ProjectStateForView 在内存中投影展示字段，不负责写回存储。
func ProjectStateForView(state *GameState, now time.Time) {
	if state == nil {
		return
	}
	next, _ := settleResources(*state, now)
	*state = next
	hydrateStateForResponse(state, now)
}

// BuildCityActionResult 从业务结果中裁剪城池相关字段。
func BuildCityActionResult(state GameState, buildingID string, upgraded int, cost int) CityActionResult {
	var building *Building
	if buildingID != "" {
		for i := range state.Buildings {
			if state.Buildings[i].ID == buildingID {
				item := state.Buildings[i]
				building = &item
				break
			}
		}
	}
	return CityActionResult{
		Building:           building,
		Buildings:          state.Buildings,
		ResourceSlots:      state.ResourceSlots,
		Resources:          state.Resources,
		ResourceProduction: state.ResourceProduction,
		CityGold:           state.CityGold,
		ActiveModifiers:    state.ActiveModifiers,
		Upgraded:           upgraded,
		Cost:               cost,
		ServerTime:         state.ServerTime,
	}
}

// BuildResourceActionResult 从业务结果中裁剪资源相关字段。
func BuildResourceActionResult(state GameState, cost int) ResourceActionResult {
	return ResourceActionResult{
		Resources:          state.Resources,
		ResourceProduction: state.ResourceProduction,
		ResourceSettledAt:  state.ResourceSettledAt,
		ProductionBoost:    state.ProductionBoost,
		ProductionBoostEnd: state.ProductionBoostEnd,
		CapacityBoost:      state.CapacityBoost,
		CapacityBoostEnd:   state.CapacityBoostEnd,
		ActiveModifiers:    state.ActiveModifiers,
		CityGold:           state.CityGold,
		Cost:               cost,
		ServerTime:         state.ServerTime,
	}
}

// BuildMilitaryActionResult 从业务结果中裁剪军事相关字段。
func BuildMilitaryActionResult(state GameState) MilitaryActionResult {
	return MilitaryActionResult{
		Army:          state.Army,
		RecruitQueues: state.RecruitQueues,
		Resources:     state.Resources,
		CityGold:      state.CityGold,
		ServerTime:    state.ServerTime,
	}
}

// BuildGeneralViewActionResult 从业务结果中裁剪武将相关字段。
func BuildGeneralViewActionResult(state GameState, accountGold int) GeneralViewActionResult {
	return GeneralViewActionResult{
		General:            state.General,
		Generals:           state.Generals,
		GeneralAssignments: state.GeneralAssignments,
		GeneralChangeUntil: state.GeneralChangeUntil,
		ActiveModifiers:    state.ActiveModifiers,
		AccountGold:        accountGold,
		ServerTime:         state.ServerTime,
	}
}

// BuildCurrencyActionResult 从局部状态中裁剪货币变化字段。
func BuildCurrencyActionResult(state GameState, accountGold int) CurrencyActionResult {
	return CurrencyActionResult{
		CityGold:       state.CityGold,
		AccountGold:    accountGold,
		LastExchangeAt: state.LastExchangeAt,
		ServerTime:     state.ServerTime,
	}
}

// BuildItemActionResult 从道具事务状态中裁剪被道具效果影响的字段。
func BuildItemActionResult(state GameState) ItemActionResult {
	if state.Inventory == nil {
		state.Inventory = map[string]ItemStack{}
	}
	resources := state.Resources
	cityGold := state.CityGold
	return ItemActionResult{
		Inventory:          state.Inventory,
		InventorySlots:     state.InventorySlots,
		Resources:          &resources,
		Army:               state.Army,
		General:            state.General,
		Generals:           state.Generals,
		GeneralAssignments: state.GeneralAssignments,
		ActiveModifiers:    state.ActiveModifiers,
		Buffs:              state.Buffs,
		CityGold:           &cityGold,
		ServerTime:         state.ServerTime,
	}
}

// BuildGeneralExpItemActionResult 从经验包小事务状态中裁剪背包和武将字段。
func BuildGeneralExpItemActionResult(state GameState) ItemActionResult {
	if state.Inventory == nil {
		state.Inventory = map[string]ItemStack{}
	}
	return ItemActionResult{
		Inventory:          state.Inventory,
		InventorySlots:     state.InventorySlots,
		General:            state.General,
		Generals:           state.Generals,
		GeneralAssignments: state.GeneralAssignments,
		ServerTime:         state.ServerTime,
	}
}

// BuildGarrisonActionResult 从增援事务状态中裁剪兵力和武将占用字段。
func BuildGarrisonActionResult(state GameState) GarrisonActionResult {
	return GarrisonActionResult{
		Army:               state.Army,
		Generals:           state.Generals,
		GeneralAssignments: state.GeneralAssignments,
		ServerTime:         state.ServerTime,
	}
}
