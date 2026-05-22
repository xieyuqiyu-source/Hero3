/**
 * 建筑配置（与后端 balance.go / balance.json 保持同步）
 * productionByLevel[0] = Lv.0 产量, productionByLevel[1] = Lv.1 产量, ...
 * upgradeCostByLevel[level] = 从 level 升到 level+1 的费用
 */

export const RESOURCE_BUILDING_PRODUCTION: Record<string, number[]> = {
  wood_camp: [4, 10, 18, 30, 44, 66, 100, 140, 200, 290, 400, 560, 750, 990, 1270, 1600, 2000, 2600, 3200, 4000, 4900],
  stone_quarry: [0, 10, 18, 30, 44, 66, 100, 140, 200, 290, 400, 560, 750, 990, 1270, 1600, 2000, 2600, 3200, 4000, 4900],
  iron_mine: [4, 10, 18, 30, 44, 66, 100, 140, 200, 290, 400, 560, 750, 990, 1270, 1600, 2000, 2600, 3200, 4000, 4900],
  farm: [0, 10, 18, 30, 44, 66, 100, 140, 200, 290, 400, 560, 750, 990, 1270, 1600, 2000, 2600, 3200, 4000, 4900],
}

export type ResourceCost = Record<string, number>

export const UPGRADE_COST_BY_LEVEL: Record<string, Record<number, ResourceCost>> = {
  wood_camp: {
    0: { wood: 80, stone: 200, iron: 100, food: 120 },
    1: { wood: 130, stone: 330, iron: 170, food: 200 },
    2: { wood: 220, stone: 560, iron: 280, food: 330 },
    3: { wood: 370, stone: 930, iron: 470, food: 560 },
    4: { wood: 620, stone: 1560, iron: 780, food: 930 },
    5: { wood: 1040, stone: 2600, iron: 1300, food: 1560 },
    6: { wood: 1740, stone: 4340, iron: 2170, food: 2600 },
    7: { wood: 2900, stone: 7250, iron: 3620, food: 4350 },
    8: { wood: 4840, stone: 12100, iron: 6050, food: 7260 },
    9: { wood: 8080, stone: 20210, iron: 10100, food: 12120 },
  },
  stone_quarry: {
    0: { wood: 160, stone: 80, iron: 160, food: 100 },
    1: { wood: 270, stone: 130, iron: 270, food: 170 },
    2: { wood: 450, stone: 220, iron: 450, food: 280 },
    3: { wood: 750, stone: 370, iron: 750, food: 470 },
    4: { wood: 1240, stone: 620, iron: 1240, food: 780 },
    5: { wood: 2080, stone: 1040, iron: 2080, food: 1300 },
    6: { wood: 3470, stone: 1740, iron: 3470, food: 2170 },
    7: { wood: 5800, stone: 2900, iron: 5800, food: 3620 },
    8: { wood: 9680, stone: 4840, iron: 9680, food: 6050 },
    9: { wood: 16160, stone: 8080, iron: 16160, food: 10100 },
  },
  iron_mine: {
    0: { wood: 200, stone: 160, iron: 60, food: 120 },
    1: { wood: 330, stone: 270, iron: 100, food: 200 },
    2: { wood: 560, stone: 450, iron: 170, food: 330 },
    3: { wood: 930, stone: 750, iron: 280, food: 560 },
    4: { wood: 1560, stone: 1240, iron: 470, food: 930 },
    5: { wood: 2600, stone: 2080, iron: 780, food: 1560 },
    6: { wood: 4340, stone: 3470, iron: 1300, food: 2600 },
    7: { wood: 7250, stone: 5800, iron: 2170, food: 4350 },
    8: { wood: 12100, stone: 9680, iron: 3630, food: 7260 },
    9: { wood: 20210, stone: 16160, iron: 6060, food: 12120 },
  },
  farm: {
    0: { wood: 120, stone: 100, iron: 120, food: 60 },
    1: { wood: 200, stone: 170, iron: 200, food: 100 },
    2: { wood: 330, stone: 280, iron: 330, food: 170 },
    3: { wood: 560, stone: 470, iron: 560, food: 280 },
    4: { wood: 930, stone: 780, iron: 930, food: 470 },
    5: { wood: 1560, stone: 1300, iron: 1560, food: 780 },
    6: { wood: 2600, stone: 2170, iron: 2600, food: 1300 },
    7: { wood: 4350, stone: 3620, iron: 4350, food: 2170 },
    8: { wood: 7260, stone: 6050, iron: 7260, food: 3630 },
    9: { wood: 12120, stone: 10100, iron: 12120, food: 6060 },
  },
}

export const UPGRADE_SECONDS_BY_LEVEL: Record<string, Record<number, number>> = {
  wood_camp: { 0: 48, 1: 60, 2: 75, 3: 93, 4: 117, 5: 146, 6: 183, 7: 228, 8: 286, 9: 357 },
  stone_quarry: { 0: 20, 1: 30, 2: 45, 3: 67, 4: 101, 5: 151, 6: 227, 7: 341, 8: 512, 9: 768 },
  iron_mine: { 0: 48, 1: 60, 2: 75, 3: 93, 4: 117, 5: 146, 6: 183, 7: 228, 8: 286, 9: 357 },
  farm: { 0: 20, 1: 30, 2: 45, 3: 67, 4: 101, 5: 151, 6: 227, 7: 341, 8: 512, 9: 768 },
}

const RESOURCE_LABELS: Record<string, string> = {
  wood: '木',
  stone: '石',
  iron: '铁',
  food: '粮',
}

/** 获取某建筑类型在指定等级的每小时产量 */
export function getProductionAtLevel(buildingType: string, level: number): number {
  const table = RESOURCE_BUILDING_PRODUCTION[buildingType]
  if (!table) return 0
  if (level < 0) return 0
  if (level >= table.length) return table[table.length - 1]
  return table[level]
}

/** 获取升级费用，返回 null 表示已满级 */
export function getUpgradeCost(buildingType: string, level: number): ResourceCost | null {
  const table = UPGRADE_COST_BY_LEVEL[buildingType]
  if (!table) return null
  return table[level] ?? null
}

/** 获取升级时间（秒） */
export function getUpgradeSeconds(buildingType: string, level: number): number {
  const table = UPGRADE_SECONDS_BY_LEVEL[buildingType]
  if (!table) return 60
  return table[level] ?? 60
}

/** 格式化资源费用为简短文本 */
export function formatCost(cost: ResourceCost): string {
  return Object.entries(cost)
    .map(([key, val]) => `${RESOURCE_LABELS[key] ?? key} ${val}`)
    .join('  ')
}

/** 格式化秒数为可读时间 */
export function formatDuration(seconds: number): string {
  if (seconds < 60) return `${seconds}秒`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}分${seconds % 60 > 0 ? `${seconds % 60}秒` : ''}`
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  return `${h}时${m > 0 ? `${m}分` : ''}`
}
