/* 本文件提供 NPC 快捷战斗的默认参战武将选择逻辑。 */
import type { GameState } from '@/types/game'

// getNpcQuickBattleGeneralIds 返回 NPC 一键攻击可自动携带的首个空闲武将。
export const getNpcQuickBattleGeneralIds = (state?: GameState | null): string[] => {
  if (!state) return []
  const busy = new Set((state.generalAssignments ?? [])
    .filter((item) => item.id !== 'main' && item.slot !== 'main')
    .map((item) => item.generalId))
  const roster = state.generals ?? (state.general ? [state.general] : [])
  const general = roster.find((item) => item.id && !busy.has(item.id))
  return general?.id ? [general.id] : []
}
