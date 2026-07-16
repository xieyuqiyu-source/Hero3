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
  buffs?: Record<string, number>
  traits?: GeneralTraitState[]
}

export interface GeneralTraitState {
  traitId: string
  traitType?: string
  name?: string
  scope?: string
  targetUnitType?: string
  params?: Record<string, number>
}

export interface GeneralAssignmentState {
  id: string
  generalId: string
  slot: string
  moduleId?: string
  status?: string
  assignedAt?: string
  endsAt?: string
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

export type NpcTier = 'small' | 'medium' | 'large' | 'golden'

export interface NpcTraitState {
  id: string
  name: string
  buffs: Record<string, number>
}

export interface NpcCityState {
  id: string
  name: string
  faction: string
  tier: NpcTier
  resources: Record<string, number>
  storageCapacity: Record<string, number>
  productionPerHour: Record<string, number>
  army: ArmyUnitState[]
  maxArmy: ArmyUnitState[]
  armyRecoveryRate: number
  recoveryProfile: string
  traits: NpcTraitState[]
  resourceSettledAt: string
  armySettledAt: string
  generatedAt: string
}

export interface NpcStateResponse {
  cities: NpcCityState[]
  lastRefreshedAt: string
  refreshCost?: number
}

export interface NpcRefreshResponse extends NpcStateResponse {
  accountGold: number
  cost: number
}

export interface NpcAttackResponse {
  battleReport: BattleReportState
  resources: ResourceState
  army: ArmyUnitState[]
  general?: GeneralState | null
  generals?: GeneralState[]
  cityGold: number
  npcState?: NpcStateResponse
  serverTime: string
}

export interface NpcScoutResponse {
  success: boolean
  battleReport: BattleReportState
  npcCity: NpcCityState | null
  army: ArmyUnitState[]
  npcState?: NpcStateResponse
  serverTime: string
}

export type NpcCommandAction = 'attack' | 'plunder' | 'scout'

export interface GameStateResponse {
  player: GamePlayer
  resources: ResourceState
  resourceProduction: Record<string, number>
  resourceSettledAt: string
  capacityBoost?: number
  capacityBoostEnd?: string
  cityGold: number
  buildings: BuildingState[]
  resourceSlots?: ResourceSlotState[]
  general: GeneralState | null
  generals?: GeneralState[]
  generalAssignments?: GeneralAssignmentState[]
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
  productionBoost?: number
  productionBoostEnd?: string
  capacityBoost?: number
  capacityBoostEnd?: string
  cityGold: number
  cost?: number
  serverTime: string
}

/** 后端按“倍率x_时长h”返回的城金价格表。 */
export type BoostPricesResponse = Record<string, number>

export interface MilitaryViewResponse {
  army: ArmyUnitState[]
  recruitQueues: RecruitQueueState[]
  resources: ResourceState
  cityGold: number
  buildings?: BuildingState[]
  general: GeneralState | null
  generals?: GeneralState[]
  generalAssignments?: GeneralAssignmentState[]
  serverTime: string
}

export interface MilitaryActionResponse {
  army: ArmyUnitState[]
  recruitQueues: RecruitQueueState[]
  resources: ResourceState
  cityGold: number
  serverTime: string
}

export type WorldMapMarchAction = 'attack' | 'plunder' | 'scout' | 'reinforce'

export interface PvpMarchState {
  id: string
  marchType: string
  status: string
  attackTroops: Record<string, number>
  attackGenerals?: string[]
  durationSeconds: number
  startedAt: string
  arrivesAt: string
  returnsAt?: string
  acceleratedTimes?: number
}

export interface PvpDispatchResponse {
  march: PvpMarchState
  army: ArmyUnitState[]
  generals?: GeneralState[]
  generalAssignments?: GeneralAssignmentState[]
  serverTime: string
}

export interface ReinforcementState {
  reinforcementId: string
  status: string
  troops: Record<string, number>
  remainingTroops?: Record<string, number>
  generals?: Array<{ id: string; name?: string; level?: number; assignment?: string }>
  marchSeconds: number
  sentAt: string
  arriveAt?: string
  expectedReturnedAt?: string
  metadata?: { acceleratedTimes?: number; [key: string]: unknown }
}

export interface ReinforcementDispatchResponse {
  reinforcement: ReinforcementState
  patch?: {
    army?: ArmyUnitState[]
    generals?: GeneralState[]
    generalAssignments?: GeneralAssignmentState[]
    serverTime: string
  }
  cityGold?: number
  cost?: number
  serverTime?: string
}

/** PVP 行军加速或召回后返回的权威局部状态。 */
export interface PvpMarchActionResponse {
  march: PvpMarchState
  army?: ArmyUnitState[]
  generals?: GeneralState[]
  cityGold?: number
  cost?: number
  serverTime: string
}

/** 增援加速后返回的城金、时间和军事局部状态。 */
export type ReinforcementActionResponse = ReinforcementDispatchResponse

export interface PvpMarchListItem extends PvpMarchState {
  attackerPlayerId: string
  attackerName: string
  defenderPlayerId: string
  defenderName: string
  returnsAt?: string
}

export interface ReinforcementListItem extends ReinforcementState {
  fromPlayerId: string
  fromPlayerName?: string
  toPlayerId: string
  toPlayerName?: string
  fromPlayerFaction?: string
  toPlayerFaction?: string
  sourceType?: string
  expectedReturnedAt?: string
}

export interface OutgoingMarchViewModel {
  id: string
  kind: WorldMapMarchAction
  label: string
  targetName: string
  troops: Record<string, number>
  status: string
  endsAt: string
  reinforcementRole?: 'sent' | 'received'
  acceleratedTimes?: number
}

export type IntelligenceTabKey = 'all' | 'attack' | 'defense' | 'reinforcement' | 'scout'

export interface BattleReportUnitState {
  unitType: string
  unitName?: string
  faction?: string
  amountBefore: number
  dispatched: number
  lost: number
  survived: number
}

export interface BattleReportGeneralState {
  id: string
  name?: string
  level?: number
  role?: string
  power?: number
  traits?: Array<{ traitId: string; name?: string; traitName?: string; summary?: string }>
}

export interface BattleReportSideState {
  role: string
  playerId?: string
  playerName?: string
  cityId?: string
  cityName?: string
  faction?: string
  factionLabel?: string
  targetType?: string
  targetId?: string
  targetName?: string
  level?: number
  power: number
  generals?: BattleReportGeneralState[]
  units: BattleReportUnitState[]
  resources?: Record<string, number>
}

export interface BattleReportDropState {
  type: string
  itemId?: string
  name?: string
  amount: number
  quality?: string
}

export interface BattleReportDetailState {
  id: string
  eventId?: string
  ownerPlayerId?: string
  viewType: string
  viewLabel?: string
  sourceType: string
  sourceLabel?: string
  battleType: string
  result: string
  winnerSide?: string
  ownerSide?: string
  ownerOutcome?: string
  title: string
  summary?: string
  occurredAt: string
  primarySide: BattleReportSideState
  secondarySide?: BattleReportSideState | null
  rewards: { resources?: Record<string, number>; drops?: BattleReportDropState[]; cityGold?: number; generalExp?: number; generalLevelBefore?: number; generalLevelAfter?: number; overflow?: Record<string, number> }
  traits?: Array<{ traitId: string; traitName?: string; ownerSide?: string; ownerRole?: string; generalId?: string; generalName?: string; summary?: string; detail?: Record<string, unknown> }>
  visibility: { showEnemyRemainingUnits: boolean; showEnemyResources: boolean; showEnemyGenerals: boolean; showEnemyCityDefense: boolean; reason?: string; threshold?: number; actualLossRatio?: number }
  extra?: Record<string, unknown>
  read: boolean
}

export interface BattleReportReinforcementState {
  reinforcementId: string
  fromPlayerId: string
  fromPlayerName?: string
  faction: string
  troops: Record<string, number>
  generals?: Array<{ id: string; name?: string; level?: number }>
}

export interface BattleReportState {
  id: string
  playerId: string
  viewType?: string
  sourceType?: string
  battleType?: string
  ownerOutcome?: string
  title?: string
  summary?: string
  detail?: BattleReportDetailState
  playerFaction?: string
  playerName?: string
  targetName?: string
  type: string
  result: string
  dispatchedUnits?: Record<string, number>
  lostUnits?: Record<string, number>
  defenderUnits?: Record<string, number>
  defenderLostUnits?: Record<string, number>
  defenderFaction?: string
  defenderResources?: Record<string, number>
  defenderRevealed?: boolean
  rewards?: Record<string, number>
  drops?: BattleReportDropState[]
  generalExpGained?: number
  generalLevelBefore?: number
  generalLevelAfter?: number
  traitTriggered?: string[]
  traitOutcomes?: Record<string, { traitId: string; name?: string; ownerSide?: string; ownerGeneralId?: string; detail?: Record<string, unknown> }>
  pvpReinforcements?: BattleReportReinforcementState[]
  pvpReinforcementLosses?: Record<string, Record<string, number>>
  pvpWall?: { faction: string; level: number; multiplier: number; totalDefenseBonus: number; hardness?: number }
  read: boolean
  createdAt: string
}

export interface BattleReportPageResponse {
  reports: BattleReportState[]
  page: number
  pageSize: number
  total: number
}

export interface ReportActionResponse {
  unreadMessageCount: number
  serverTime: string
}

export interface IntelligenceReportViewModel {
  id: string
  type: string
  typeLabel: string
  title: string
  createdAt: string
  read: boolean
  source: BattleReportState
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
  capacityBoost: number
  capacityBoostEnd: string
  resources: ResourceViewModel[]
  resourceBuildings: ResourceBuildingViewModel[]
  general: { name: string; level: number; icon: string } | null
  army: ArmyUnitViewModel[]
  buildQueues: Array<{ id: string; name: string; level: number; endsAt: string }>
  unreadMessageCount: number
  unreadMailCount: number
}
