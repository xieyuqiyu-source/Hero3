// 本文件验证曹操虎卫显示投影严格遵守后端时间、留城条件和不限产量规则。
import assert from 'node:assert/strict'
import test from 'node:test'
import { projectGuardArmy } from '../src/utils/guardProjection.ts'
import type { GameState } from '../src/types/game.ts'

/** 构造只包含投影所需字段的曹操权威状态。 */
function caoCaoState(overrides: Partial<GameState> = {}): GameState {
  return {
    player: { id: 'p1', nickname: '主公', faction: 'wei' },
    resources: { items: {}, capacity: {} }, resourceProduction: {}, resourceSettledAt: '2026-07-21T00:00:00Z',
    generalTraitProgress: { 'caocao:weiwu_haoling:huWei': 0.5 }, cityGold: 0, buildings: [], army: [{ unitType: 'huWei', amount: 100 }], recruitQueues: [], serverTime: '2026-07-21T00:00:00Z',
    general: { id: 'caocao', name: '曹操', level: 1, exp: 0, buffs: {}, traits: [{ traitId: 'weiwu_haoling', name: '魏武号令', params: { guardPerMinute: 300 }, targetUnitType: 'huWei' }] },
    ...overrides,
  } as GameState
}

test('曹操留城时按每分钟 300 虎卫实时增长且不设显示上限', () => {
  const receivedAt = Date.parse('2026-07-21T08:00:00Z')
  const projected = projectGuardArmy(caoCaoState(), receivedAt, receivedAt + 24 * 60 * 60 * 1000)
  assert.equal(projected.find((unit) => unit.unitType === 'huWei')?.amount, 432100)
})

test('曹操出征或增援期间停止虎卫投影', () => {
  const receivedAt = Date.parse('2026-07-21T08:00:00Z')
  const state = caoCaoState({ generalAssignments: [{ id: 'march-1', generalId: 'caocao', slot: 'march' }] })
  assert.equal(projectGuardArmy(state, receivedAt, receivedAt + 60_000)[0].amount, 100)
})

test('前端投影不修改后端权威兵力对象', () => {
  const state = caoCaoState()
  projectGuardArmy(state, 1_000, 61_000)
  assert.equal(state.army[0].amount, 100)
})
