/**
 * 建筑产量配置（与后端 balance.go / balance.json 保持同步）
 * productionByLevel[0] = Lv.0 产量, productionByLevel[1] = Lv.1 产量, ...
 */

export const RESOURCE_BUILDING_PRODUCTION: Record<string, number[]> = {
  wood_camp: [4, 10, 18, 30, 44, 66, 100, 140, 200, 290, 400, 560, 750, 990, 1270, 1600, 2000, 2600, 3200, 4000, 4900],
  stone_quarry: [0, 10, 18, 30, 44, 66, 100, 140, 200, 290, 400, 560, 750, 990, 1270, 1600, 2000, 2600, 3200, 4000, 4900],
  iron_mine: [4, 10, 18, 30, 44, 66, 100, 140, 200, 290, 400, 560, 750, 990, 1270, 1600, 2000, 2600, 3200, 4000, 4900],
  farm: [0, 10, 18, 30, 44, 66, 100, 140, 200, 290, 400, 560, 750, 990, 1270, 1600, 2000, 2600, 3200, 4000, 4900],
}

/** 获取某建筑类型在指定等级的每小时产量 */
export function getProductionAtLevel(buildingType: string, level: number): number {
  const table = RESOURCE_BUILDING_PRODUCTION[buildingType]
  if (!table) return 0
  if (level < 0) return 0
  if (level >= table.length) return table[table.length - 1]
  return table[level]
}
