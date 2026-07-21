/** 本文件依据后端权威时间投影曹操留城期间的虎卫显示值，不写回玩家状态。 */
import type { ArmyUnitState, GameStateResponse } from './types'

const CAO_CAO_ID = 'caocao'
const TRAIT_ID = 'weiwu_haoling'
const DEFAULT_UNIT_TYPE = 'huWei'

/** 判断曹操是否仍在本城，主将占位记录不视为离城。 */
export function isCaoCaoAtHome(state: GameStateResponse): boolean {
  if (state.general?.id !== CAO_CAO_ID) return false
  return !(state.generalAssignments ?? []).some((assignment) => (
    assignment.generalId === CAO_CAO_ID && assignment.id !== 'main' && assignment.slot !== 'main'
  ))
}

/** 查找后端下发的有效魏武号令配置。 */
function guardTrait(state: GameStateResponse) {
  const general = state.general?.id === CAO_CAO_ID
    ? state.general
    : state.generals?.find((item) => item.id === CAO_CAO_ID)
  return general?.traits?.find((trait) => trait.traitId === TRAIT_ID && Number(trait.params?.guardPerMinute) > 0)
}

/** 将权威兵力叠加当前时刻应显示的虎卫增量，返回全新数组。 */
export function projectGuardArmy(state: GameStateResponse | null, receivedAt: number | null, projectedServerNow: number): ArmyUnitState[] {
  if (!state) return []
  const baseline = (state.army ?? []).map((unit) => ({ ...unit }))
  if (!receivedAt || !projectedServerNow || !isCaoCaoAtHome(state)) return baseline

  const trait = guardTrait(state)
  const perMinute = Number(trait?.params?.guardPerMinute ?? 0)
  const settledAt = Date.parse(state.resourceSettledAt)
  if (!trait || !Number.isFinite(settledAt) || perMinute <= 0) return baseline

  const elapsedSeconds = Math.max(0, (projectedServerNow - settledAt) / 1000)
  const unitType = trait.targetUnitType || DEFAULT_UNIT_TYPE
  const progressKey = `${CAO_CAO_ID}:${TRAIT_ID}:${unitType}`
  const progress = Math.max(0, Number(state.generalTraitProgress?.[progressKey] ?? 0))
  const produced = Math.floor(progress + (perMinute * elapsedSeconds) / 60 + 1e-9)
  if (produced <= 0) return baseline

  const existing = baseline.find((unit) => unit.unitType === unitType)
  if (existing) existing.amount += produced
  else baseline.push({ unitType, amount: produced })
  return baseline
}
