/** V0.3 实际读取的玩家完整状态字段类型。 */
export interface GamePlayer {
  id: string
  nickname: string
  faction: string
}

export interface ResourceState {
  items: Record<string, number>
  capacity: Record<string, number>
}

export interface BuildingState {
  id: string
  type: string
  level: number
  status?: string
  upgradeEndsAt: string | null
  statusEndsAt?: string | null
}

export interface ResourceSlotState {
  id: string
  resourceType: string
  buildingId?: string
  unlockedBy?: string
  unlockedAt?: string
}

export interface GeneralState {
  id: string
  name: string
  level: number
}

export interface ArmyUnitState {
  unitType: string
  amount: number
}

export interface RecruitQueueState {
  id: string
  unitType: string
  amount: number
  endsAt: string
}

export interface GameStateResponse {
  player: GamePlayer
  resources: ResourceState
  resourceProduction: Record<string, number>
  resourceSettledAt: string
  cityGold: number
  buildings: BuildingState[]
  resourceSlots?: ResourceSlotState[]
  general: GeneralState | null
  generals?: GeneralState[]
  army: ArmyUnitState[]
  recruitQueues: RecruitQueueState[]
  unreadMessageCount: number
  unreadMailCount: number
  serverTime: string
}

export interface CityActionResponse {
  building?: BuildingState
  buildings: BuildingState[]
  resourceSlots?: ResourceSlotState[]
  resources: ResourceState
  resourceProduction: Record<string, number>
  cityGold: number
  serverTime: string
}

export interface ResourceActionResponse {
  resources: ResourceState
  resourceProduction: Record<string, number>
  resourceSettledAt: string
  cityGold: number
  cost?: number
  serverTime: string
}

export interface MilitaryViewResponse {
  army: ArmyUnitState[]
  recruitQueues: RecruitQueueState[]
  resources: ResourceState
  cityGold: number
  buildings?: BuildingState[]
  general: GeneralState | null
  generals?: GeneralState[]
  serverTime: string
}

export interface MilitaryActionResponse {
  army: ArmyUnitState[]
  recruitQueues: RecruitQueueState[]
  resources: ResourceState
  cityGold: number
  serverTime: string
}

export interface ResourceViewModel {
  key: string
  name: string
  icon: string
  amount: number
  capacity: number
  productionPerHour: number
}

export interface ResourceBuildingViewModel {
  id: string
  slotId: string
  resourceKey: string
  resourceName: string
  buildingName: string
  buildingType: string
  image: string | null
  level: number
  status: string
  endsAt: string | null
  isFallback: boolean
}

export interface ArmyUnitViewModel {
  key: string
  name: string
  category: 'infantry' | 'cavalry' | 'siege' | 'special' | 'unknown'
  amount: number
  icon: string
}

export interface CityGameViewModel {
  player: { nickname: string; faction: string; factionName: string }
  serverTime: string
  accountGold: number
  cityGold: number
  resources: ResourceViewModel[]
  resourceBuildings: ResourceBuildingViewModel[]
  general: { name: string; level: number; icon: string } | null
  army: ArmyUnitViewModel[]
  buildQueues: Array<{ id: string; name: string; level: number; endsAt: string }>
  unreadMessageCount: number
  unreadMailCount: number
}
