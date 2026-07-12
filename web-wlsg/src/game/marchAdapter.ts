/** 把 PVP 与增援两套后端行军统一为右侧栏出征状态。 */
import type { OutgoingMarchViewModel, PvpMarchListItem, ReinforcementListItem, WorldMapMarchAction } from './types'

const activeStatuses = new Set(['marching', 'returning'])
const actionLabels: Record<WorldMapMarchAction, string> = { attack: '攻击', plunder: '掠夺', scout: '侦查', reinforce: '增援' }

/** 将玩家本人派出的活动 PVP 行军转换为统一模型。 */
export function mapPvpMarches(playerId: string, items: PvpMarchListItem[]): OutgoingMarchViewModel[] {
  return items.filter((item) => item.attackerPlayerId === playerId && activeStatuses.has(item.status)).map((item) => {
    const kind = (['attack', 'plunder', 'scout'].includes(item.marchType) ? item.marchType : 'attack') as Exclude<WorldMapMarchAction, 'reinforce'>
    return { id: item.id, kind, label: actionLabels[kind], targetName: item.defenderName || item.defenderPlayerId, troops: item.attackTroops ?? {}, status: item.status, endsAt: item.status === 'returning' ? (item.returnsAt || item.arrivesAt) : item.arrivesAt }
  })
}

/** 将本人派出的活动增援批次转换为统一模型。 */
export function mapReinforcements(items: ReinforcementListItem[]): OutgoingMarchViewModel[] {
  return items.filter((item) => activeStatuses.has(item.status)).map((item) => ({ id: item.reinforcementId, kind: 'reinforce', label: actionLabels.reinforce, targetName: item.toPlayerName || item.toPlayerId, troops: item.troops ?? {}, status: item.status, endsAt: item.status === 'returning' ? (item.expectedReturnedAt || item.arriveAt || '') : (item.arriveAt || '') }))
}

/** 合并两套行军并按最近结束时间排序，未知时间稳定排在最后。 */
export function toOutgoingMarches(playerId: string, pvp: PvpMarchListItem[], reinforcements: ReinforcementListItem[]): OutgoingMarchViewModel[] {
  return [...mapPvpMarches(playerId, pvp), ...mapReinforcements(reinforcements)].sort((left, right) => {
    const leftTime = new Date(left.endsAt).getTime()
    const rightTime = new Date(right.endsAt).getTime()
    return (Number.isFinite(leftTime) ? leftTime : Number.MAX_SAFE_INTEGER) - (Number.isFinite(rightTime) ? rightTime : Number.MAX_SAFE_INTEGER) || left.id.localeCompare(right.id)
  })
}
