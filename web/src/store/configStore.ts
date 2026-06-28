import { create } from 'zustand'
import { gameApi } from '@/api/game'
import type { ItemDefinition, ReincarnationConfig } from '@/types/game'

export interface BuildingConfig {
  type: string
  name: string
  resourceType?: string
  productionByLevel?: number[]
  capacityByLevel?: number[]
  modifiersByLevel?: Record<number, Array<{ key: string; value: number; mode: string }>>
  upgradeCostByLevel?: Record<number, Record<string, number>>
  goldUpgradeCostByLevel?: Record<number, number>
  upgradeSecondsByLevel?: Record<number, number>
}

const MODIFIER_LABELS: Record<string, string> = {
  attackBonus: '攻击',
  defenseBonus: '防御',
  productionBonus: '产量',
  buildSpeedBonus: '建造速度',
  marchSpeedBonus: '行军速度',
  enemyLossRevealThresholdBonus: '情报隐蔽',
  recruitSpeedBonus: '征兵速度',
  infantryRecruitSpeedBonus: '步兵征兵',
  cavalryRecruitSpeedBonus: '骑兵征兵',
  siegeRecruitSpeedBonus: '攻城征兵',
  specialRecruitSpeedBonus: '特殊征兵',
  recruitCostReduction: '征兵消耗',
  siegeRecruitCostReduction: '攻城消耗',
  specialRecruitCostReduction: '特殊消耗',
}

export interface BalanceConfig {
  baseProduction: Record<string, number>
  buildings: Record<string, BuildingConfig>
  overflowToCityGold: number // 多少溢出资源兑换 1 城金
  exchangeRate: number // 1 金币 = N 城金
  reverseExchangeRate: number // N 城金 = 1 金币
  exchangeCooldownSecs: number // 兑换冷却秒数
  cityGoldPerSecond: number // 1 城金折抵多少秒（加速用）
  boostBaseCost: number // 产量加成基础价格
  boostMultiplierFactor: Record<string, number> // 倍率系数
  boostDurationFactor: Record<string, number> // 时长系数
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
  unlock: Record<string, string | number>
}

export interface GeneralInfo {
  id: string
  name: string
  title: string
}

export interface FishingRarityConfig {
  label: string
  color: string
  bg: string
  border: string
  weight: number
  glow: string
}

export interface FishingBaitConfig {
  id: string
  name: string
  tier: string
  description: string
  rarityBoost: number
  cityGoldCost: number
  biteChance: number
  biteWindowMs: number
  sweetStart: number
  sweetEnd: number
}

export interface FishingFishConfig {
  name: string
  rarity: 'common' | 'rare' | 'epic' | 'legendary'
  reward: string
  rewardAmount: number
  description: string
  emoji: string
}

export interface FishingConfig {
  rarities: Record<string, FishingRarityConfig>
  baits: FishingBaitConfig[]
  fishPool: FishingFishConfig[]
}

export interface FactionConfig {
  name: string
  description: string
  icon: string
  traits: Record<string, number>
  generals: GeneralInfo[]
}

interface ConfigStore {
  balance: BalanceConfig | null
  factions: Record<string, FactionConfig> | null
  units: Record<string, Record<string, UnitConfig>> | null
  items: Record<string, ItemDefinition> | null
  fishing: FishingConfig | null
  reincarnation: ReincarnationConfig | null
  loaded: boolean
  loadBootstrap: () => Promise<void>
}

export const useConfigStore = create<ConfigStore>((set, get) => ({
  balance: null,
  factions: null,
  units: null,
  items: null,
  fishing: null,
  reincarnation: null,
  loaded: false,

  loadBootstrap: async () => {
    if (get().loaded) return
    try {
      const data = await gameApi.bootstrap()
      set({
        balance: data.balance,
        factions: data.factions,
        units: data.units,
        items: data.items,
        fishing: data.fishing,
        reincarnation: data.reincarnation,
        loaded: true,
      })
    } catch {
      // 加载失败静默处理
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

/** 获取金币升级费用，返回 null 表示不是金币建筑或已满级 */
export function getGoldUpgradeCost(buildingType: string, level: number): number | null {
  const balance = useConfigStore.getState().balance
  if (!balance) return null
  const config = balance.buildings[buildingType]
  if (!config?.goldUpgradeCostByLevel) return null
  return config.goldUpgradeCostByLevel[level] ?? null
}

/** 获取升级时间（秒） */
export function getUpgradeSeconds(buildingType: string, level: number): number {
  const balance = useConfigStore.getState().balance
  if (!balance) return 60
  const config = balance.buildings[buildingType]
  if (!config?.upgradeSecondsByLevel) return 60
  return config.upgradeSecondsByLevel[level] ?? 60
}

/** 获取建筑指定等级的加成说明 */
export function getBuildingModifierText(buildingType: string, level: number): string {
  const balance = useConfigStore.getState().balance
  if (!balance || level < 0) return ''
  const config = balance.buildings[buildingType]
  const modifiers = config?.modifiersByLevel?.[level]
  if (!modifiers || modifiers.length === 0) return ''
  return modifiers
    .filter((modifier) => modifier.value !== 0)
    .map((modifier) => `${MODIFIER_LABELS[modifier.key] ?? modifier.key} ${formatModifierAmount(modifier.value, modifier.mode)}`)
    .join('  ')
}

/** 获取建筑当前等级和下一级的加成说明 */
export function getBuildingModifierProgressText(buildingType: string, level: number): string {
  const current = getBuildingModifierText(buildingType, level)
  const next = getBuildingModifierText(buildingType, level + 1)
  if (current && next) return `${current} -> ${next}`
  return current || next
}

/** 格式化建筑加成数值 */
function formatModifierAmount(value: number, mode: string): string {
  if (mode === 'percentAdd' || mode === 'percentMultiply') {
    return `+${Math.round(value * 100)}%`
  }
  if (value > 0) return `+${formatCompactNumber(value)}`
  return formatCompactNumber(value)
}

/** 格式化紧凑数值 */
function formatCompactNumber(value: number): string {
  if (Math.abs(value - Math.round(value)) < 0.000001) return String(Math.round(value))
  return value.toFixed(2).replace(/\.?0+$/, '')
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
