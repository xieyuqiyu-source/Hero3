// 本文件归口玩家局部视图查询，供前端按页面读取必要字段。
package game

import "time"

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
	Army               []ArmyUnit          `json:"army"`
	RecruitQueues      []RecruitQueue      `json:"recruitQueues"`
	General            *General            `json:"general"`
	Generals           []General           `json:"generals,omitempty"`
	GeneralAssignments []GeneralAssignment `json:"generalAssignments,omitempty"`
	ServerTime         string              `json:"serverTime"`
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
	Resources          ResourceState           `json:"resources,omitempty"`
	Army               []ArmyUnit              `json:"army,omitempty"`
	General            *General                `json:"general,omitempty"`
	Generals           []General               `json:"generals,omitempty"`
	GeneralAssignments []GeneralAssignment     `json:"generalAssignments,omitempty"`
	ActiveModifiers    []ModifierBreakdownItem `json:"activeModifiers,omitempty"`
	Buffs              []Buff                  `json:"buffs,omitempty"`
	CityGold           FlexInt                 `json:"cityGold"`
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
	if err := s.SettleDuePvpMarches(playerID); err != nil {
		return MilitaryView{}, err
	}
	return s.repo.GetMilitaryView(playerID)
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
	return ItemActionResult{
		Inventory:          state.Inventory,
		InventorySlots:     state.InventorySlots,
		Resources:          state.Resources,
		Army:               state.Army,
		General:            state.General,
		Generals:           state.Generals,
		GeneralAssignments: state.GeneralAssignments,
		ActiveModifiers:    state.ActiveModifiers,
		Buffs:              state.Buffs,
		CityGold:           state.CityGold,
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
