/** 验证 PVP 与增援活动行军统一映射、过滤和排序。 */
import { describe, expect, it } from 'vitest'
import { splitOutgoingMarches, toOutgoingMarches } from './marchAdapter'
import { traitLabel } from './traitLabels'
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
  it('同时保留本人出征和脱敏来袭队列并按最近时间排序', () => {
    const items = toOutgoingMarches('p1', [pvp({ acceleratedTimes: 1 }), pvp({ id: 'incoming', attackerPlayerId: 'p9', attackerName: '来袭者', defenderPlayerId: 'p1', defenderName: '主公', viewerRole: 'incoming', attackTroops: { secret: 999 } }), pvp({ id: 'done', status: 'resolved' })], [reinforcement({ metadata: { acceleratedTimes: 2 } })])
    expect(items.map((item) => item.id)).toEqual(['r1', 'incoming', 'm1'])
    expect(items[0]).toMatchObject({ kind: 'reinforce', label: '增援', targetName: '盟友', acceleratedTimes: 2 })
    expect(items[1]).toMatchObject({ kind: 'attack', label: '被攻击', targetName: '来袭者', troops: {}, pvpRole: 'incoming' })
    expect(items[2].acceleratedTimes).toBe(1)
  })

  it('返回中使用返回时间且未知模式安全回退为攻击', () => {
    const items = toOutgoingMarches('p1', [pvp({ status: 'returning', marchType: 'future_mode', returnsAt: '2026-07-13T00:03:00Z' })], [reinforcement({ status: 'returning', expectedReturnedAt: '2026-07-13T00:04:00Z' })])
    expect(items[0]).toMatchObject({ kind: 'attack', status: 'returning', endsAt: '2026-07-13T00:03:00Z' })
    expect(items[1].endsAt).toBe('2026-07-13T00:04:00Z')
  })

  it('区分攻击、掠夺和侦查来袭，并过滤已返程的敌方队列', () => {
    const incoming = { attackerPlayerId: 'p9', attackerName: '敌方', defenderPlayerId: 'p1', viewerRole: 'incoming' as const }
    const items = toOutgoingMarches('p1', [
      pvp({ ...incoming, id: 'attack-incoming', marchType: 'attack' }),
      pvp({ ...incoming, id: 'plunder-incoming', marchType: 'plunder' }),
      pvp({ ...incoming, id: 'scout-incoming', marchType: 'scout' }),
      pvp({ ...incoming, id: 'returning-incoming', status: 'returning' }),
    ], [])
    expect(items.map((item) => item.label).sort()).toEqual(['被侦查', '被攻击', '被掠夺'].sort())
    expect(items.some((item) => item.id === 'returning-incoming')).toBe(false)
    expect(items.every((item) => item.pvpRole === 'incoming' && Object.keys(item.troops).length === 0)).toBe(true)
  })

  it('防守方 ID 匹配但后端未明确标记来袭时不信任也不展示', () => {
    const items = toOutgoingMarches('p1', [
      pvp({ id: 'missing-role', attackerPlayerId: 'p9', defenderPlayerId: 'p1' }),
      pvp({ id: 'sent-role', attackerPlayerId: 'p8', defenderPlayerId: 'p1', viewerRole: 'sent' }),
    ], [])
    expect(items).toEqual([])
  })

  it('五项正式行军特性都直接使用后端最终到达时间和最低时长结果', () => {
    const cases = [
      { traitId: 'jixing_benxi', name: '疾行奔袭', duration: 2475, speed: 1.2, arrivesAt: '2026-07-13T00:41:15Z' },
      { traitId: 'qijin_qichu', name: '七进七出', duration: 60, speed: 2, arrivesAt: '2026-07-13T00:01:00Z' },
      { traitId: 'baiyi_dujiang', name: '白衣渡江', duration: 2475, speed: 1.2, arrivesAt: '2026-07-13T00:41:15Z' },
      { traitId: 'baiyi_jixing', name: '白衣急行', duration: 2063, speed: 1.44, arrivesAt: '2026-07-13T00:34:23Z' },
      { traitId: 'kuairu_shandian', name: '快如闪电', duration: 30, speed: 5, arrivesAt: '2026-07-13T00:00:30Z' },
    ]
    for (const tc of cases) {
      expect(traitLabel(tc.traitId)).toBe(tc.name)
      const items = toOutgoingMarches('p1', [
        pvp({ durationSeconds: tc.duration, speedMultiplier: tc.speed, arrivesAt: tc.arrivesAt }),
      ], [
        reinforcement({ marchSeconds: tc.duration, speedMultiplier: tc.speed, arriveAt: tc.arrivesAt }),
      ])
      expect(items).toHaveLength(2)
      expect(items.every((item) => item.endsAt === tc.arrivesAt)).toBe(true)
    }
  })

  it('白衣渡江未命中时直接采用白衣急行后的后端到达时间', () => {
    const arrivesAt = '2026-07-20T00:41:15Z'
    const items = toOutgoingMarches('p1', [
      pvp({ durationSeconds: 2475, speedMultiplier: 3000 / 2475, startedAt: '2026-07-20T00:00:00Z', arrivesAt }),
    ], [
      reinforcement({ marchSeconds: 2475, speedMultiplier: 3000 / 2475, sentAt: '2026-07-20T00:00:00Z', arriveAt: arrivesAt }),
    ])
    expect(items).toHaveLength(2)
    expect(items.map((item) => item.endsAt)).toEqual([arrivesAt, arrivesAt])
    expect(items.map((item) => item.kind).sort()).toEqual(['attack', 'reinforce'])
  })

  it('快如闪电未命中时直接采用后端基线到达时间', () => {
    const arrivesAt = '2026-07-20T00:49:30Z'
    const items = toOutgoingMarches('p1', [
      pvp({ durationSeconds: 2970, speedMultiplier: 3000 / 2970, startedAt: '2026-07-20T00:00:00Z', arrivesAt }),
    ], [
      reinforcement({ marchSeconds: 2970, speedMultiplier: 3000 / 2970, sentAt: '2026-07-20T00:00:00Z', arriveAt: arrivesAt }),
    ])
    expect(items).toHaveLength(2)
    expect(items.map((item) => item.endsAt)).toEqual([arrivesAt, arrivesAt])
    expect(items.map((item) => item.kind).sort()).toEqual(['attack', 'reinforce'])
  })

  it('将增援与攻击、掠夺和侦查拆成独立状态组', () => {
    const received = reinforcement({ reinforcementId: 'received', fromPlayerId: 'p9', fromPlayerName: '援军主公', toPlayerId: 'p1', toPlayerName: '主公' })
    const items = toOutgoingMarches('p1', [pvp(), pvp({ id: 'scout', marchType: 'scout' })], [reinforcement()], [received])
    const groups = splitOutgoingMarches(items)
    expect(groups.expeditions.map((item) => item.id)).toEqual(['m1', 'scout'])
    expect(groups.reinforcements.map((item) => item.id)).toEqual(['r1'])
    expect(groups.receivedReinforcements[0]).toMatchObject({ id: 'received', label: '被增援', targetName: '援军主公', reinforcementRole: 'received' })
  })

  it('被增援只显示仍在赶往本城的批次', () => {
    const items = toOutgoingMarches('p1', [], [], [reinforcement({ reinforcementId: 'marching' }), reinforcement({ reinforcementId: 'stationed', status: 'stationed' }), reinforcement({ reinforcementId: 'returning', status: 'returning' })])
    expect(items.map((item) => item.id)).toEqual(['marching'])
  })
})
