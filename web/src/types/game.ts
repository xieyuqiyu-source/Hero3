/* 最小可用类型定义 - 随后端接口稳定后逐步扩展 */

export interface Player {
  id: string
  nickname: string
  faction: string
}

export interface AccountSession {
  accountId: string
  username: string
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
  targetId: string
  result: 'victory' | 'defeat'
  playerPower: number
  enemyPower: number
  lostUnits: Record<string, number>
  rewards: Record<string, number>
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
  /** 产量加成倍率（1=无加成，2/4/8/16） */
  productionBoost?: number
  buildings: Building[]
  general: General | null
  army: ArmyUnit[]
  recruitQueues: RecruitQueue[]
  mapTargets: MapTarget[]
  recentBattleReports: BattleReport[]
  unreadMessageCount: number
  serverTime: string
}
