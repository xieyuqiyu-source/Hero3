/* 本文件维护战报和战斗弹窗使用的固定兵种展示顺序。 */

const FACTION_UNIT_ORDER: Record<string, string[]> = {
  shu: [
    'flyingKite',
    'greedyWolf',
    'qilinGuard',
    'azureDragon',
    'xiLiangCavalry',
    'southernElephant',
    'siegeTower',
    'thunderBolt',
    'woodenOx',
    'hanRoyalty',
  ],
  wei: [
    'qingZhouArmy',
    'jinWeiSoldier',
    'huWei',
    'zhanYingTanMa',
    'qiQiYing',
    'huBaoQi',
    'chongZhuangChe',
    'luLeiChe',
    'jianzhuShi',
    'tuZu',
  ],
  wu: [
    'shadowGuard',
    'xiuLuo',
    'secretAgent',
    'divineWind',
    'zhuQueRider',
    'overlordRider',
    'chongChe',
    'juShiChe',
    'fengShuiMaster',
    'taiPingShi',
  ],
}

type UnitConfigMap = Record<string, Record<string, { name?: string; role?: string }>>

// getUnitDisplayOrder 返回指定阵营的固定兵种展示顺序。
export const getUnitDisplayOrder = (faction?: string): string[] => FACTION_UNIT_ORDER[faction ?? ''] ?? []

// sortUnitIds 按固定兵种顺序排列 ID，未知兵种排在已知兵种之后。
export const sortUnitIds = (unitIds: string[], faction?: string, units?: UnitConfigMap): string[] => {
  const order = getUnitDisplayOrder(faction)
  const rank = new Map(order.map((unitId, index) => [unitId, index]))
  const fallbackOrder = Object.keys(units?.[faction ?? ''] ?? {})
  const fallbackRank = new Map(fallbackOrder.map((unitId, index) => [unitId, index + order.length]))
  return [...unitIds].sort((a, b) => {
    const aRank = rank.get(a) ?? fallbackRank.get(a) ?? Number.MAX_SAFE_INTEGER
    const bRank = rank.get(b) ?? fallbackRank.get(b) ?? Number.MAX_SAFE_INTEGER
    if (aRank !== bRank) return aRank - bRank
    return a.localeCompare(b)
  })
}

// sortUnitEntries 按固定兵种顺序排列兵种数量键值对。
export const sortUnitEntries = (
  troops?: Record<string, number> | null,
  faction?: string,
  units?: UnitConfigMap,
): Array<[string, number]> => {
  const source = troops ?? {}
  return sortUnitIds(Object.keys(source), faction, units).map((unitId) => [unitId, source[unitId] ?? 0])
}
