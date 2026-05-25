/* 最小可用类型定义 - 随后端接口稳定后逐步扩展 */

export interface Player {
  id: string
  nickname: string
  faction: string
}

export interface AccountSession {
  accountId: string
  username: string
  gold: number
}

export interface PlayerSummary {
  id: string
  nickname: string
  faction: string
  totalArmy: number
  buildingLevel: number
  updatedAt: string
}

export interface ResourceState {
  items: Record<string, number>
  capacity: Record<string, number>
}

export type ResourceProduction = Record<string, number>

export interface Building {
  id: string
  type: string
  level: number
  upgradeEndsAt: string | null
}

export interface ArmyUnit {
  unitType: string
  amount: number
}

export interface RecruitQueue {
  id: string
  unitType: string
  amount: number
  endsAt: string
}

export interface MapTarget {
  id: string
  type: string
  level: number
  power: number
  rewards: Record<string, number>
}

export interface BattleReport {
  id: string
  playerId: string
  targetId: string
  targetName: string
  type: 'attack' | 'plunder' | 'scout' | 'reinforce'
  result: 'attacker_victory' | 'defender_victory' | 'draw'
  playerPower: number
  enemyPower: number
  dispatchedUnits: Record<string, number>
  lostUnits: Record<string, number>
  defenderFaction: string
  defenderUnits: Record<string, number>
  defenderLostUnits: Record<string, number>
  defenderRevealed: boolean
  defenderResources: Record<string, number>
  rewards: Record<string, number>
  overflow?: Record<string, number>
  overflowCityGold?: number
  read: boolean
  createdAt: string
}

export interface General {
  id: string
  name: string
  level: number
  exp: number
  buffs: Record<string, number>
}

export interface GameState {
  player: Player
  resources: ResourceState
  resourceProduction: ResourceProduction
  resourceSettledAt: string
  /** 存档级城金 */
  cityGold: number
  /** 上次兑换时间（冷却用） */
  lastExchangeAt?: string
  /** 产量加成倍率（1=无加成，2/4/8/16） */
  productionBoost?: number
  buildings: Building[]
  general: General | null
  army: ArmyUnit[]
  recruitQueues: RecruitQueue[]
  npcState?: NpcState | null
  mapTargets: MapTarget[]
  recentBattleReports: BattleReport[]
  unreadMessageCount: number
  serverTime: string
}

// --- NPC 城池类型 ---

export interface NpcTrait {
  id: string
  name: string
  buffs: Record<string, number>
}

export interface NpcCity {
  id: string
  name: string
  faction: string
  tier: 'small' | 'medium' | 'large' | 'golden'
  resources: Record<string, number>
  storageCapacity: Record<string, number>
  productionPerHour: Record<string, number>
  army: ArmyUnit[]
  maxArmy: ArmyUnit[]
  armyRecoveryRate: number
  recoveryProfile: string
  traits: NpcTrait[]
  resourceSettledAt: string
  armySettledAt: string
  generatedAt: string
}

export interface NpcState {
  cities: NpcCity[]
  lastRefreshedAt: string
}
