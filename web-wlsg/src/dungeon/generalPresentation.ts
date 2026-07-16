/** 轮回绝境武将展示规则：选择默认随军武将并格式化后端真实战斗加成。 */
import { traitLabel } from '../game/traitLabels'
import type { GeneralAssignmentState, GeneralState } from '../game/types'

const combatBuffLabels: Record<string, string> = {
  attackBonus: '攻击',
  defenseBonus: '防御',
  infantryDefenseBonus: '步防',
  cavalryDefenseBonus: '骑防',
}

/** 优先选择本城主将；主将不可用时选择第一名空闲武将。 */
export function preferredDungeonGeneralId(generals: GeneralState[], assignments: GeneralAssignmentState[]) {
  const mainGeneralId = assignments.find((assignment) => assignment.id === 'main' || assignment.slot === 'main')?.generalId
  return generals.find((general) => general.id === mainGeneralId)?.id ?? generals[0]?.id ?? ''
}

/** 把武将当前真实战斗 Buff 与已激活特性格式化为玩家可读摘要。 */
export function dungeonGeneralBonusText(general: GeneralState | null) {
  if (!general) return '未携带武将，本波不享受武将属性与特性加成'
  const buffs = Object.entries(general.buffs ?? {}).flatMap(([key, value]) => {
    const label = combatBuffLabels[key]
    if (!label || !Number.isFinite(value) || value === 0) return []
    return [`${label} ${value > 0 ? '+' : ''}${Math.round(value * 10000) / 100}%`]
  })
  const traits = (general.traits ?? []).map((trait) => traitLabel(trait.traitId, trait.name)).filter(Boolean)
  const parts: string[] = []
  if (buffs.length) parts.push(`属性：${buffs.join('、')}`)
  if (traits.length) parts.push(`特性：${traits.join('、')}（按实际作用范围触发）`)
  return parts.length ? parts.join('；') : '已携带武将，战斗按后端当前武将属性结算'
}
