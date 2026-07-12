/** V0.2 登录、账号、启动配置和存档列表接口类型。 */
export interface AccountSession {
  accountId: string
  username: string
  gold: number
  token: string
}

export interface AccountInfo {
  accountId: string
  username: string
  gold: number
}

export interface PlayerSummary {
  id: string
  nickname: string
  faction: string
  mailCode?: string
  totalArmy: number
  buildingLevel: number
  updatedAt: string
  deleteRequestedAt?: string
  deleteScheduledAt?: string
}

export interface BootstrapResponse {
  gameName: string
  modules: string[]
  balance: { buildings: Record<string, BuildingBalanceConfig>; cityGoldPerSecond?: number }
  units: UnitsConfig
  message: string
  [key: string]: unknown
}

export interface UnitConfig {
  name: string
  description: string
  category: string
  role?: string
  icon: string
  stats: Record<string, number>
  cost: Record<string, number>
  trainSeconds: number
  unlock: Record<string, unknown>
}

export type UnitsConfig = Record<string, Record<string, UnitConfig>>

export interface BuildingBalanceConfig {
  type: string
  name: string
  resourceType?: string
  productionByLevel?: number[]
  upgradeCostByLevel?: Record<number, Record<string, number>>
  goldUpgradeCostByLevel?: Record<number, number>
  upgradeSecondsByLevel?: Record<number, number>
}

export interface ErrorResponse {
  error?: string
  message?: string
}
