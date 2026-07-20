// 本文件验证战报增援批次按玩家或来源城市聚合为标准参战方。
import assert from 'node:assert/strict'
import test from 'node:test'

import { aggregateReinforcementSnapshots } from '../src/pages/news/reportAggregation.ts'

test('同一玩家多次增援合并兵力战损且只保留一个武将', () => {
  const groups = aggregateReinforcementSnapshots([
    {
      reinforcementId: 'rein_a',
      fromPlayerId: 'player_same',
      fromPlayerName: '同一玩家',
      faction: 'wei',
      troops: { weiInfantry: 100, weiCavalry: 20 },
      generals: [{ id: 'caocao', name: '曹操', level: 10 }],
      generalExpGained: 120,
      generalLevelBefore: 10,
      generalLevelAfter: 11,
      sourceTags: { source_type: 'reinforcement', source_player_id: 'player_same', source_id: 'rein_a' },
    },
    {
      reinforcementId: 'rein_b',
      fromPlayerId: 'player_same',
      fromPlayerName: '同一玩家',
      faction: 'wei',
      troops: { weiInfantry: 30 },
      generals: [{ id: 'zhangliao', name: '张辽', level: 10 }],
      generalExpGained: 80,
      sourceTags: { source_type: 'reinforcement', source_player_id: 'player_same', source_id: 'rein_b' },
    },
  ], {
    rein_a: { weiInfantry: 10 },
    rein_b: { weiInfantry: 5 },
  })

  assert.equal(groups.length, 1)
  assert.equal(groups[0].groupKey, 'player:player_same')
  assert.deepEqual(groups[0].troops, { weiInfantry: 130, weiCavalry: 20 })
  assert.deepEqual(groups[0].losses, { weiInfantry: 15 })
  assert.equal(groups[0].generals.length, 1)
  assert.equal(groups[0].generals[0].id, 'caocao')
  assert.equal(groups[0].generalExpGained, 120)
  assert.equal(groups[0].generalLevelBefore, 10)
  assert.equal(groups[0].generalLevelAfter, 11)
  assert.deepEqual(groups[0].reinforcementIds, ['rein_a', 'rein_b'])
})

test('没有玩家身份的驻防按来源城市合并，不同城市保持独立', () => {
  const groups = aggregateReinforcementSnapshots([
    {
      reinforcementId: 'city_batch_a',
      fromPlayerId: '',
      fromPlayerName: '东城驻防',
      faction: 'shu',
      troops: { shuInfantry: 40 },
      sourceTags: { source_type: 'city_garrison', source_id: 'city_east' },
    },
    {
      reinforcementId: 'city_batch_b',
      fromPlayerId: '',
      fromPlayerName: '东城驻防',
      faction: 'shu',
      troops: { shuInfantry: 20 },
      sourceTags: { source_type: 'city_garrison', source_id: 'city_east' },
    },
    {
      reinforcementId: 'city_batch_c',
      fromPlayerId: '',
      fromPlayerName: '西城驻防',
      faction: 'shu',
      troops: { shuInfantry: 30 },
      sourceTags: { source_type: 'city_garrison', source_id: 'city_west' },
    },
  ], {})

  assert.equal(groups.length, 2)
  assert.equal(groups[0].groupKey, 'source:city_garrison:city_east')
  assert.equal(groups[0].troops.shuInfantry, 60)
  assert.equal(groups[1].groupKey, 'source:city_garrison:city_west')
  assert.equal(groups[1].troops.shuInfantry, 30)
})

test('同一武将的多批援军经验只按后端快照精确累加', () => {
  const groups = aggregateReinforcementSnapshots([
    { reinforcementId: 'rein_exp_a', fromPlayerId: 'player_exp', faction: 'wu', troops: {}, generals: [{ id: 'sunquan' }], generalExpGained: 30 },
    { reinforcementId: 'rein_exp_b', fromPlayerId: 'player_exp', faction: 'wu', troops: {}, generals: [{ id: 'sunquan' }], generalExpGained: 20 },
  ], {})

  assert.equal(groups.length, 1)
  assert.equal(groups[0].generalExpGained, 50)
})

test('无将援军不借用城内主将快照或经验', () => {
  const groups = aggregateReinforcementSnapshots([
    {
      reinforcementId: 'rein_without_general', fromPlayerId: 'player_without_general', faction: 'wei',
      troops: { huWei: 100 }, generals: [], generalExpGained: 0,
    },
  ], { rein_without_general: { huWei: 50 } })

  assert.equal(groups.length, 1)
  assert.deepEqual(groups[0].generals, [])
  assert.equal(groups[0].generalExpGained, 0)
  assert.deepEqual(groups[0].troops, { huWei: 100 })
  assert.deepEqual(groups[0].losses, { huWei: 50 })
})

test('增援升级只透传后端等级结果，不在前端推算经验曲线', () => {
  const groups = aggregateReinforcementSnapshots([
    {
      reinforcementId: 'rein_level', fromPlayerId: 'player_level', faction: 'wu', troops: {},
      generals: [{ id: 'sunquan', level: 20, exp: 12345 }], generalExpGained: 88,
      generalLevelBefore: 20, generalLevelAfter: 21,
    },
  ], {})

  assert.equal(groups[0].generalExpGained, 88)
  assert.equal(groups[0].generalLevelBefore, 20)
  assert.equal(groups[0].generalLevelAfter, 21)
})

test('重试返回相同援军快照时不重复累计战报经验', () => {
  const snapshot = {
    reinforcementId: 'rein_retry', fromPlayerId: 'player_retry', faction: 'wei', troops: { huWei: 100 },
    generals: [{ id: 'xiahouyuan', level: 1, exp: 99 }], generalExpGained: 110,
    generalLevelBefore: 1, generalLevelAfter: 2,
  }
  const groups = aggregateReinforcementSnapshots([snapshot, snapshot], { rein_retry: { huWei: 70 } })

  assert.equal(groups.length, 1)
  assert.deepEqual(groups[0].reinforcementIds, ['rein_retry'])
  assert.equal(groups[0].troops.huWei, 100)
  assert.equal(groups[0].generalExpGained, 110)
})

test('黄巾协防援军支持后端给出的跨多级升级结果', () => {
  const groups = aggregateReinforcementSnapshots([
    {
      reinforcementId: 'rein_yellow_turban', fromPlayerId: 'player_helper', faction: 'wu', troops: { shadowGuard: 400 },
      generals: [{ id: 'sunquan', name: '孙权', level: 1, exp: 99 }], generalExpGained: 600,
      generalLevelBefore: 1, generalLevelAfter: 4,
    },
  ], { rein_yellow_turban: { shadowGuard: 258 } })

  assert.equal(groups[0].generalExpGained, 600)
  assert.equal(groups[0].generalLevelBefore, 1)
  assert.equal(groups[0].generalLevelAfter, 4)
  assert.deepEqual(groups[0].losses, { shadowGuard: 258 })
})
