export interface HealthState {
  status: string
  service: string
  version: string
  environment: string
  time: string
}

export interface GameState {
  player: {
    id: string
    nickname: string
    faction: string
  }
  resources: {
    wood: number
    stone: number
    iron: number
    food: number
    capacity: number
  }
  buildings: Array<{
    id: string
    type: string
    level: number
    upgradeEndsAt: string | null
  }>
  army: Array<{
    unitType: string
    amount: number
  }>
  recruitQueues: Array<{
    id: string
    unitType: string
    amount: number
    endsAt: string
    status: string
  }>
  mapTargets: Array<{
    id: string
    type: string
    level: number
    power: number
  }>
  recentBattleReports: Array<{
    id: string
    result: string
  }>
  unreadMessageCount: number
  serverTime: string
}

export interface PlayerSummary {
  id: string
  nickname: string
  faction: string
  updatedAt: string
}

export interface AccountSummary {
  id: string
  username: string
  createdAt: string
  players: PlayerSummary[]
}
