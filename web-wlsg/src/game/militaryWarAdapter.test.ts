/** 验证战争页真实兵力、武将占用和多批增援映射。 */
import { describe, expect, it } from 'vitest'
import type { UnitsConfig } from '../api/types'
import type { ReinforcementListItem } from './types'
import { generalIsAway, homeGeneral, militaryWarFoodPerHour, toMilitaryWarReinforcements, toMilitaryWarUnitCatalog, toMilitaryWarUnits } from './militaryWarAdapter'

describe('军事战争页真实数据适配', () => {
  it('保留零兵力、未知图标和真实口粮计算', () => {
    const units = toMilitaryWarUnits([
      { id: 'known', name: '虎卫', officialCode: 103, description: '', category: 'infantry', stats: [0, 0, 0, 0, 0, 2], owned: 5, cost: {}, trainSeconds: 0, dispatchable: true },
      { id: 'future', name: '未来兵种', officialCode: null, description: '', category: 'other', stats: [0, 0, 0, 0, 0, 3], owned: 0, cost: {}, trainSeconds: 0, dispatchable: true },
    ])
    expect(units).toHaveLength(2)
    expect(units[1]).toMatchObject({ id: 'future', officialCode: null, amount: 0 })
    expect(militaryWarFoodPerHour(units)).toBe(10)
  })

  it('非主将槽占用会把增援武将从本城隐藏', () => {
    const general = { id: 'g1', name: '曹操', level: 56 }
    const assignments = [{ id: 'main', generalId: 'g1', slot: 'main' }, { id: 'reinforcement-r1-g1', generalId: 'g1', slot: 'reinforcement', status: 'marching' }]
    expect(generalIsAway('g1', assignments)).toBe(true)
    expect(homeGeneral(general, assignments)).toBeNull()
  })

  it('同一目标的多个增援批次逐条展示并保留武将与剩余兵力', () => {
    const base: ReinforcementListItem = { reinforcementId: 'r1', fromPlayerId: 'p1', fromPlayerName: '主公', toPlayerId: 'p2', toPlayerName: '盟友', status: 'marching', troops: { huWei: 10 }, remainingTroops: { huWei: 8 }, generals: [{ id: 'g1', name: '曹操', level: 56 }], marchSeconds: 60, sentAt: '', arriveAt: '2026-07-14T08:01:00Z' }
    const rows = toMilitaryWarReinforcements([base, { ...base, reinforcementId: 'r2', arriveAt: '2026-07-14T08:02:00Z' }], 'outgoing')
    expect(rows).toHaveLength(2)
    expect(rows[0]).toMatchObject({ playerName: '盟友', generalNames: '曹操(Lv 56)', troops: 'huWei×8', status: '行军中' })
  })

  it('每批来援按来源阵营补全独立守军结构和耗粮', () => {
    const unit = (name: string, upkeep: number) => ({ name, description: '', category: 'infantry', icon: '', stats: { upkeep }, cost: {}, trainSeconds: 1, unlock: {} })
    const catalog = toMilitaryWarUnitCatalog({ wei: { huWei: unit('虎卫', 3), jinWeiSoldier: unit('禁卫甲士', 2) } } satisfies UnitsConfig)
    const base: ReinforcementListItem = { reinforcementId: 'r1', fromPlayerId: 'p1', fromPlayerName: '许昌', fromPlayerFaction: 'wei', toPlayerId: 'p2', status: 'stationed', troops: { huWei: 10 }, remainingTroops: { huWei: 8 }, generals: [], marchSeconds: 0, sentAt: '' }
    const rows = toMilitaryWarReinforcements([base, { ...base, reinforcementId: 'r2', fromPlayerName: '洛阳', remainingTroops: { huWei: 4 } }], 'incoming', catalog)
    expect(rows).toHaveLength(2)
    expect(rows[0]).toMatchObject({ playerName: '许昌', faction: 'wei', foodPerHour: 24 })
    expect(rows[0].units).toEqual([
      expect.objectContaining({ id: 'huWei', name: '虎卫', amount: 8, officialCode: 103 }),
      expect.objectContaining({ id: 'jinWeiSoldier', name: '禁卫甲士', amount: 0, officialCode: 102 }),
    ])
    expect(rows[1]).toMatchObject({ playerName: '洛阳', foodPerHour: 12 })
  })

  it('完成和失败记录不混入当前军队区块', () => {
    const item = (status: string): ReinforcementListItem => ({ reinforcementId: status, fromPlayerId: 'p1', toPlayerId: 'p2', status, troops: {}, marchSeconds: 0, sentAt: '' })
    expect(toMilitaryWarReinforcements([item('completed'), item('failed'), item('stationed')], 'incoming').map((row) => row.id)).toEqual(['stationed'])
  })
})
