/** 验证 PVP 与增援活动行军统一映射、过滤和排序。 */
import { describe, expect, it } from 'vitest'
import { toOutgoingMarches } from './marchAdapter'
import type { PvpMarchListItem, ReinforcementListItem } from './types'

/** 创建最小 PVP 行军。 */
function pvp(overrides: Partial<PvpMarchListItem> = {}): PvpMarchListItem {
  return { id: 'm1', marchType: 'attack', status: 'marching', attackTroops: { huWei: 10 }, durationSeconds: 60, startedAt: '', arrivesAt: '2026-07-13T00:02:00Z', attackerPlayerId: 'p1', attackerName: '主公', defenderPlayerId: 'p2', defenderName: '目标', ...overrides }
}

/** 创建最小增援批次。 */
function reinforcement(overrides: Partial<ReinforcementListItem> = {}): ReinforcementListItem {
  return { reinforcementId: 'r1', status: 'marching', troops: { huWei: 5 }, marchSeconds: 60, sentAt: '', arriveAt: '2026-07-13T00:01:00Z', fromPlayerId: 'p1', toPlayerId: 'p3', toPlayerName: '盟友', ...overrides }
}

describe('出征状态适配', () => {
  it('只保留本人派出的活动行军并按最近时间排序', () => {
    const items = toOutgoingMarches('p1', [pvp(), pvp({ id: 'incoming', attackerPlayerId: 'p9' }), pvp({ id: 'done', status: 'resolved' })], [reinforcement()])
    expect(items.map((item) => item.id)).toEqual(['r1', 'm1'])
    expect(items[0]).toMatchObject({ kind: 'reinforce', label: '增援', targetName: '盟友' })
  })

  it('返回中使用返回时间且未知模式安全回退为攻击', () => {
    const items = toOutgoingMarches('p1', [pvp({ status: 'returning', marchType: 'future_mode', returnsAt: '2026-07-13T00:03:00Z' })], [reinforcement({ status: 'returning', expectedReturnedAt: '2026-07-13T00:04:00Z' })])
    expect(items[0]).toMatchObject({ kind: 'attack', status: 'returning', endsAt: '2026-07-13T00:03:00Z' })
    expect(items[1].endsAt).toBe('2026-07-13T00:04:00Z')
  })
})
