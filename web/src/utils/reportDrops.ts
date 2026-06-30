/* 本文件提供战报掉落物的前端展示聚合工具。 */
import type { BattleReport } from '@/types/game'

export type BattleReportDrop = NonNullable<BattleReport['drops']>[number]

// mergeBattleReportDrops 按物品标识合并掉落物数量，减少扫荡战报弹窗的重复展示。
export function mergeBattleReportDrops(drops: BattleReportDrop[]): BattleReportDrop[] {
  const merged = new Map<string, BattleReportDrop>()
  for (const drop of drops) {
    const itemKey = drop.itemId || drop.name || drop.type || 'drop'
    const qualityKey = drop.quality || ''
    const key = `${itemKey}::${qualityKey}`
    const current = merged.get(key)
    if (current) {
      current.amount += drop.amount
    } else {
      merged.set(key, { ...drop })
    }
  }
  return Array.from(merged.values())
}
