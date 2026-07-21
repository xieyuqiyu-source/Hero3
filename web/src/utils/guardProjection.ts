// 本文件依据后端权威时间投影曹操留城期间的虎卫显示值，不修改任何存档状态。
import type { ArmyUnit, GameState } from '../types/game'

const CAO_CAO_ID = 'caocao'
const TRAIT_ID = 'weiwu_haoling'
const DEFAULT_UNIT_TYPE = 'huWei'

/** 判断曹操当前是否仍在本城，主将占位记录本身不算离城任务。 */
export function isCaoCaoAtHome(state: GameState): boolean {
  if (state.general?.id !== CAO_CAO_ID) return false
  return !(state.generalAssignments ?? []).some((assignment) => (
    assignment.generalId === CAO_CAO_ID && assignment.id !== 'main' && assignment.slot !== 'main'
  ))
}

/** 读取后端下发的魏武号令参数，未配置或停用时不进行前端投影。 */
function guardTrait(state: GameState) {
  const general = state.general?.id === CAO_CAO_ID
    ? state.general
    : state.generals?.find((item) => item.id === CAO_CAO_ID)
  return general?.traits?.find((trait) => trait.traitId === TRAIT_ID && Number(trait.params?.guardPerMinute) > 0)
}

/** 按服务端基线、小数进度和本地经过时长计算仅用于展示的虎卫列表。 */
export function projectGuardArmy(state: GameState | null, receivedAt: number | null, now = Date.now()): ArmyUnit[] {
  if (!state) return []
  const baseline = (state.army ?? []).map((unit) => ({ ...unit }))
  if (!receivedAt || !isCaoCaoAtHome(state)) return baseline

  const trait = guardTrait(state)
  const perMinute = Number(trait?.params?.guardPerMinute ?? 0)
  const serverTime = Date.parse(state.serverTime)
  const settledAt = Date.parse(state.resourceSettledAt)
  if (!trait || !Number.isFinite(serverTime) || !Number.isFinite(settledAt) || perMinute <= 0) return baseline

  const projectedServerNow = serverTime + Math.max(0, now - receivedAt)
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
