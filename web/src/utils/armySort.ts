// 本文件提供前端军队展示的稳定排序规则，避免兵力更新后列表跳动。
import type { ArmyUnit } from '@/types/game'
import type { UnitConfig } from '@/store/configStore'

const CATEGORY_ORDER: Record<string, number> = {
  infantry: 0,
  cavalry: 1,
  siege: 2,
  special: 3,
}

// sortArmyForDisplay 按步兵、骑兵、攻城、特殊的固定顺序稳定排列军队。
export function sortArmyForDisplay(
  army: ArmyUnit[] | undefined,
  factionUnits: Record<string, UnitConfig> | undefined,
): ArmyUnit[] {
  const order = new Map(Object.keys(factionUnits ?? {}).map((unitType, index) => [unitType, index]))
  return [...(army ?? [])]
    .filter((unit) => unit.amount > 0)
    .sort((left, right) => {
      const leftConfig = factionUnits?.[left.unitType]
      const rightConfig = factionUnits?.[right.unitType]
      const leftCategory = categoryRank(leftConfig)
      const rightCategory = categoryRank(rightConfig)
      if (leftCategory !== rightCategory) {
        return leftCategory - rightCategory
      }
      const leftUnlock = unlockLevel(leftConfig)
      const rightUnlock = unlockLevel(rightConfig)
      if (leftUnlock !== rightUnlock) {
        return leftUnlock - rightUnlock
      }
      const leftOrder = order.get(left.unitType)
      const rightOrder = order.get(right.unitType)
      if (leftOrder !== undefined && rightOrder !== undefined && leftOrder !== rightOrder) return leftOrder - rightOrder
      if (leftOrder !== undefined) return -1
      if (rightOrder !== undefined) return 1
      const leftName = leftConfig?.name ?? left.unitType
      const rightName = rightConfig?.name ?? right.unitType
      return leftName.localeCompare(rightName, 'zh-Hans')
    })
}

// categoryRank 返回兵种分类排序权重。
function categoryRank(config: UnitConfig | undefined): number {
  return CATEGORY_ORDER[config?.category ?? ''] ?? 99
}

// unlockLevel 返回兵种解锁等级，用于同分类内稳定排序。
function unlockLevel(config: UnitConfig | undefined): number {
  const level = config?.unlock?.level
  return typeof level === 'number' ? level : 999
}
