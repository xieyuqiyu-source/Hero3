/* 本文件把同一玩家或同一来源城市的多批增援聚合为一个战报参战方。 */
import type { DefenseReinforcementUnit, ReinforcementGeneralSnapshot } from '@/types/game'

export interface AggregatedReinforcementSnapshot {
  groupKey: string
  reinforcementIds: string[]
  sourcePlayerId: string
  sourceType: string
  sourceId: string
  playerName: string
  faction: string
  troops: Record<string, number>
  losses: Record<string, number>
  generals: ReinforcementGeneralSnapshot[]
}

// reinforcementGroupIdentity 优先按玩家归并，没有玩家身份时按来源城市或来源实体归并。
function reinforcementGroupIdentity(item: DefenseReinforcementUnit): { key: string; playerId: string; sourceType: string; sourceId: string } {
  const sourceType = item.sourceTags?.source_type?.trim() || 'reinforcement'
  const playerId = item.sourceTags?.source_player_id?.trim() || item.fromPlayerId?.trim() || ''
  const sourceId = item.sourceTags?.source_id?.trim() || ''
  if (playerId) return { key: `player:${playerId}`, playerId, sourceType, sourceId }
  if (sourceId) return { key: `source:${sourceType}:${sourceId}`, playerId: '', sourceType, sourceId }
  return { key: `batch:${item.reinforcementId}`, playerId: '', sourceType, sourceId: item.reinforcementId }
}

// mergeAmounts 把当前批次的兵力或损失累加到参战方汇总。
function mergeAmounts(target: Record<string, number>, source?: Record<string, number>): void {
  Object.entries(source ?? {}).forEach(([unitType, amount]) => {
    if (!Number.isFinite(amount) || amount <= 0) return
    target[unitType] = (target[unitType] ?? 0) + amount
  })
}

// aggregateReinforcementSnapshots 聚合同源增援；业务规则限定每个玩家只保留一个武将快照。
export function aggregateReinforcementSnapshots(
  items: DefenseReinforcementUnit[],
  lossesByReinforcement: Record<string, Record<string, number>>,
): AggregatedReinforcementSnapshot[] {
  const groups: AggregatedReinforcementSnapshot[] = []
  const indexByKey = new Map<string, number>()

  items.forEach((item) => {
    const identity = reinforcementGroupIdentity(item)
    let index = indexByKey.get(identity.key)
    if (index === undefined) {
      index = groups.length
      indexByKey.set(identity.key, index)
      groups.push({
        groupKey: identity.key,
        reinforcementIds: [],
        sourcePlayerId: identity.playerId,
        sourceType: identity.sourceType,
        sourceId: identity.sourceId,
        playerName: item.fromPlayerName?.trim() || item.fromPlayerId?.trim() || identity.sourceId || '未知增援方',
        faction: item.faction,
        troops: {},
        losses: {},
        generals: item.generals?.[0] ? [item.generals[0]] : [],
      })
    }

    const group = groups[index]
    group.reinforcementIds.push(item.reinforcementId)
    if (!group.playerName && item.fromPlayerName) group.playerName = item.fromPlayerName
    if (!group.faction && item.faction) group.faction = item.faction
    if (group.generals.length === 0 && item.generals?.[0]) group.generals = [item.generals[0]]
    mergeAmounts(group.troops, item.troops)
    mergeAmounts(group.losses, lossesByReinforcement[item.reinforcementId])
  })

  return groups
}
