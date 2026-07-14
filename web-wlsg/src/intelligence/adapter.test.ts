/** 本文件验证真实军情筛选、字段映射、未知类型和空页边界。 */
import { describe, expect, it } from 'vitest'
import type { BattleReportState } from '../game/types'
import { intelligenceFilter, intelligenceTabs, intelligenceTotalPages, resolveIntelligenceType, toIntelligenceReport } from './adapter'

/** 创建字段完整且允许覆盖的战报测试数据。 */
function report(overrides: Partial<BattleReportState> = {}): BattleReportState {
  return { id: 'r1', playerId: 'p1', viewType: 'attack', battleType: 'pvp', title: '主公 攻击 敌城', type: 'attack', result: 'attacker_victory', read: false, createdAt: '2026-07-13T08:00:00Z', ...overrides }
}

describe('军情适配层', () => {
  it('固定展示五个标签且不包含饿死兵', () => {
    expect(intelligenceTabs.map((item) => item.key)).toEqual(['all', 'attack', 'defense', 'reinforcement', 'scout'])
    expect(intelligenceTabs.map((item) => item.label)).not.toContain('饿死兵')
  })

  it('将页面标签映射为后端真实筛选字段', () => {
    expect(intelligenceFilter('all')).toBeUndefined()
    expect(intelligenceFilter('defense')).toEqual({ viewType: 'defense' })
    expect(intelligenceFilter('scout')).toEqual({ battleType: 'scout' })
  })

  it('映射真实主题、未读和未知类型兜底', () => {
    expect(toIntelligenceReport(report())).toMatchObject({ id: 'r1', type: 'attack', typeLabel: '攻 击', title: '主公 攻击 敌城', read: false })
    const unknown = toIntelligenceReport(report({ viewType: 'future', battleType: 'future', type: 'future', title: '' }))
    expect(unknown.typeLabel).toBe('军 情')
    expect(unknown.title).toContain('future')
    expect(resolveIntelligenceType(report({ type: 'scout', battleType: 'scout' }))).toBe('scout')
  })

  it('空列表仍保留安全页数', () => {
    expect(intelligenceTotalPages(0, 8)).toBe(1)
    expect(intelligenceTotalPages(17, 8)).toBe(3)
  })
})
