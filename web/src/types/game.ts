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
  updatedAt: string
}

export interface ResourceState {
  wood: number
  stone: number
  iron: number
  food: number
  capacity: number
}

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
  status: 'pending' | 'completed' | 'claimed'
}

export interface MapTarget {
  id: string
  type: string
  level: number
  power: number
  rewards: Partial<ResourceState>
}

export interface BattleReport {
  id: string
  targetId: string
  result: 'victory' | 'defeat'
  playerPower: number
  enemyPower: number
  lostUnits: Record<string, number>
  rewards: Partial<ResourceState>
  createdAt: string
}

export interface GameState {
  player: Player
  resources: ResourceState
  buildings: Building[]
  army: ArmyUnit[]
  recruitQueues: RecruitQueue[]
  mapTargets: MapTarget[]
  recentBattleReports: BattleReport[]
  unreadMessageCount: number
  serverTime: string
}
