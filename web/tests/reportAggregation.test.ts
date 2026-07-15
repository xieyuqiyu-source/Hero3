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
