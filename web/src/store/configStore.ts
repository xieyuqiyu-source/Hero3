import { create } from 'zustand'
import { gameApi } from '@/api/game'

export interface BuildingConfig {
  type: string
  name: string
  resourceType?: string
  productionByLevel?: number[]
  capacityByLevel?: number[]
  upgradeCostByLevel?: Record<number, Record<string, number>>
  upgradeSecondsByLevel?: Record<number, number>
}

export interface BalanceConfig {
  baseProduction: Record<string, number>
  buildings: Record<string, BuildingConfig>
}

interface ConfigStore {
  balance: BalanceConfig | null
  loaded: boolean
  loadBootstrap: () => Promise<void>
}

export const useConfigStore = create<ConfigStore>((set, get) => ({
  balance: null,
  loaded: false,

  loadBootstrap: async () => {
    if (get().loaded) return
    try {
      const data = await gameApi.bootstrap()
      set({ balance: data.balance, loaded: true })
    } catch {
      // 加载失败静默处理，前端会用 fallback
    }
  },
}))

/** 获取某建筑类型在指定等级的每小时产量 */
export function getProductionAtLevel(buildingType: string, level: number): number {
  const balance = useConfigStore.getState().balance
  if (!balance) return 0
  const config = balance.buildings[buildingType]
  if (!config?.productionByLevel) return 0
  const table = config.productionByLevel
  if (level < 0) return 0
  if (level >= table.length) return table[table.length - 1]
  return table[level]
}

/** 获取升级费用，返回 null 表示已满级 */
export function getUpgradeCost(buildingType: string, level: number): Record<string, number> | null {
  const balance = useConfigStore.getState().balance
  if (!balance) return null
  const config = balance.buildings[buildingType]
  if (!config?.upgradeCostByLevel) return null
  return config.upgradeCostByLevel[level] ?? null
}

/** 获取升级时间（秒） */
export function getUpgradeSeconds(buildingType: string, level: number): number {
  const balance = useConfigStore.getState().balance
  if (!balance) return 60
  const config = balance.buildings[buildingType]
  if (!config?.upgradeSecondsByLevel) return 60
  return config.upgradeSecondsByLevel[level] ?? 60
}

const RESOURCE_LABELS: Record<string, string> = {
  wood: '木',
  stone: '石',
  iron: '铁',
  food: '粮',
}

/** 格式化资源费用为简短文本 */
export function formatCost(cost: Record<string, number>): string {
  return Object.entries(cost)
    .map(([key, val]) => `${RESOURCE_LABELS[key] ?? key} ${val}`)
    .join('  ')
}

/** 格式化秒数为可读时间 */
export function formatDuration(seconds: number): string {
  if (seconds < 60) return `${seconds}秒`
  if (seconds < 3600) {
    const m = Math.floor(seconds / 60)
    const s = seconds % 60
    return s > 0 ? `${m}分${s}秒` : `${m}分`
  }
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  return m > 0 ? `${h}时${m}分` : `${h}时`
}
