/** 本文件验证武林三国前端的曹操虎卫投影遵守权威时间与离城边界。 */
import { describe, expect, it } from 'vitest'
import { projectGuardArmy } from './guardProjection'
import type { GameStateResponse } from './types'

/** 构造只包含投影所需字段的曹操状态。 */
function caoCaoState(overrides: Partial<GameStateResponse> = {}): GameStateResponse {
  return {
    player: { id: 'p1', nickname: '主公', faction: 'wei' }, resources: { items: {}, capacity: {} }, resourceProduction: {},
    resourceSettledAt: '2026-07-21T00:00:00Z', generalTraitProgress: { 'caocao:weiwu_haoling:huWei': 0.5 }, cityGold: 0,
    buildings: [], army: [{ unitType: 'huWei', amount: 100 }], recruitQueues: [], unreadMessageCount: 0, unreadMailCount: 0, serverTime: '2026-07-21T00:00:00Z',
    general: { id: 'caocao', name: '曹操', level: 1, traits: [{ traitId: 'weiwu_haoling', name: '魏武号令', targetUnitType: 'huWei', params: { guardPerMinute: 300 } }] },
    ...overrides,
  }
}

describe('魏武号令兵力投影', () => {
  it('留城 24 小时显示增加 432000 虎卫且没有上限', () => {
    const army = projectGuardArmy(caoCaoState(), 1, Date.parse('2026-07-22T00:00:00Z'))
    expect(army.find((unit) => unit.unitType === 'huWei')?.amount).toBe(432100)
  })

  it('出征后立即停止增长', () => {
    const state = caoCaoState({ generalAssignments: [{ id: 'march-1', generalId: 'caocao', slot: 'march' }] })
    expect(projectGuardArmy(state, 1, Date.parse('2026-07-22T00:00:00Z'))[0].amount).toBe(100)
  })

  it('投影不改变后端权威兵力', () => {
    const state = caoCaoState()
    projectGuardArmy(state, 1, Date.parse('2026-07-22T00:00:00Z'))
    expect(state.army[0].amount).toBe(100)
  })
})
