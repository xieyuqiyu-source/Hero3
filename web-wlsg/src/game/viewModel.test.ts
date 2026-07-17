/** 验证真实状态转换的重复地块、未知类型、扩展数量和零值。 */
import { describe, expect, it } from 'vitest'
import type { GameStateResponse } from './types'
import { toCityGameViewModel } from './viewModel'

/** 创建可按测试覆盖的最小完整游戏状态。 */
function gameState(overrides: Partial<GameStateResponse> = {}): GameStateResponse {
  return {
    player: { id: 'p1', nickname: '主公', faction: 'wei' },
    resources: { items: { wood: 0, stone: 20, iron: 30, food: 40 }, capacity: { wood: 0, stone: 200, iron: 300, food: 400 } },
    resourceProduction: { wood: 0, stone: 2, iron: 3, food: 4 }, resourceSettledAt: '2026-07-12T08:00:00Z', cityGold: 0,
    buildings: [], resourceSlots: [], general: null, army: [], recruitQueues: [], unreadMessageCount: 0, unreadMailCount: 0, serverTime: '2026-07-12T08:00:00Z',
    ...overrides,
  }
}

describe('当前城池视图模型', () => {
  it('保留同类型多个地块且不去重', () => {
    const state = gameState({
      buildings: [
        { id: 'wood-1', type: 'wood_camp', level: 3, upgradeEndsAt: null },
        { id: 'wood-2', type: 'wood_camp', level: 4, upgradeEndsAt: null },
      ],
      resourceSlots: [
        { id: 'slot-1', resourceType: 'wood', buildingId: 'wood-1' },
        { id: 'slot-2', resourceType: 'wood', buildingId: 'wood-2' },
      ],
    })
    const model = toCityGameViewModel(state, 0)
    expect(model.resourceBuildings).toHaveLength(2)
    expect(model.resourceBuildings.map((item) => item.level)).toEqual([3, 4])
  })

  it('超过官方默认数量的地块全部进入动态网格', () => {
    const buildings = Array.from({ length: 24 }, (_, index) => ({ id: `wood-${index}`, type: 'wood_camp', level: index, upgradeEndsAt: null }))
    const resourceSlots = buildings.map((building, index) => ({ id: `slot-${index}`, resourceType: 'wood', buildingId: building.id }))
    expect(toCityGameViewModel(gameState({ buildings, resourceSlots }), 0).resourceBuildings).toHaveLength(24)
  })

  it('资源网格按木泥铁粮四列稳定交错且不改变类型内顺序', () => {
    const types = [
      ['wood', 'wood_camp'], ['stone', 'stone_quarry'], ['iron', 'iron_mine'], ['food', 'farm'],
    ] as const
    const buildings = types.flatMap(([, type]) => [1, 2].map((index) => ({ id: `${type}-${index}`, type, level: index, upgradeEndsAt: null })))
    const resourceSlots = types.flatMap(([resource, type]) => [1, 2].map((index) => ({ id: `${resource}-${index}`, resourceType: resource, buildingId: `${type}-${index}` })))
    const model = toCityGameViewModel(gameState({ buildings, resourceSlots }), 0)
    expect(model.resourceBuildings.map((item) => item.resourceKey)).toEqual(['wood', 'stone', 'iron', 'food', 'wood', 'stone', 'iron', 'food'])
    expect(model.resourceBuildings.map((item) => item.level)).toEqual([1, 1, 1, 1, 2, 2, 2, 2])
  })

  it('未知资源和建筑类型使用可见兜底并保留真实名称等级', () => {
    const model = toCityGameViewModel(gameState({
      buildings: [{ id: 'mystic-1', type: 'mystic_garden', level: 7, upgradeEndsAt: null }],
      resourceSlots: [{ id: 'slot-mystic', resourceType: 'crystal', buildingId: 'mystic-1' }],
      resources: { items: { crystal: 9 }, capacity: { crystal: 99 } }, resourceProduction: { crystal: 1 },
    }), 0)
    expect(model.resources[4]).toMatchObject({ key: 'crystal', amount: 9, capacity: 99 })
    expect(model.resourceBuildings[0]).toMatchObject({ buildingName: 'mystic_garden', level: 7, isFallback: true, image: null })
  })

  it('零值、缺少地块数组和空军队稳定展示', () => {
    const model = toCityGameViewModel(gameState({ buildings: [{ id: 'farm-1', type: 'farm', level: 0, upgradeEndsAt: null }], resourceSlots: undefined }), 0)
    expect(model.resources[0]).toMatchObject({ amount: 0, capacity: 0, productionPerHour: 0 })
    expect(model.resourceBuildings[0]).toMatchObject({ buildingName: '农田', level: 0 })
    expect(model.army).toEqual([])
  })

  it('魏蜀吴每个兵种使用官网 army_content 对应的独立缩略图', () => {
    const weiArmy = ['qingZhouArmy', 'jinWeiSoldier', 'huWei', 'zhanYingTanMa', 'qiQiYing', 'huBaoQi', 'chongZhuangChe', 'luLeiChe', 'jianzhuShi', 'tuZu']
      .map((unitType) => ({ unitType, amount: 1 }))
    const shuArmy = ['greedyWolf', 'qilinGuard', 'azureDragon', 'flyingKite', 'xiLiangCavalry', 'southernElephant', 'siegeTower', 'thunderBolt', 'woodenOx', 'hanRoyalty']
      .map((unitType) => ({ unitType, amount: 1 }))
    const wuArmy = ['shadowGuard', 'xiuLuo', 'secretAgent', 'divineWind', 'zhuQueRider', 'overlordRider', 'chongChe', 'juShiChe', 'fengShuiMaster', 'taiPingShi']
      .map((unitType) => ({ unitType, amount: 1 }))
    const wei = toCityGameViewModel(gameState({ general: { id: 'g0', name: '曹操', level: 10 }, army: weiArmy }), 0)
    const shu = toCityGameViewModel(gameState({ player: { id: 'p1', nickname: '蜀主', faction: 'shu' }, general: { id: 'g1', name: '关羽', level: 10 }, army: shuArmy }), 0)
    const wu = toCityGameViewModel(gameState({ player: { id: 'p2', nickname: '吴主', faction: 'wu' }, general: { id: 'g2', name: '周瑜', level: 10 }, army: wuArmy }), 0)
    expect(wei.army.map((unit) => unit.icon)).toEqual(['101.gif', '102.gif', '103.gif', '104.gif', '105.gif', '106.gif', '107.gif', '108.gif', '109.gif', '110.gif'])
    expect(shu.army.map((unit) => unit.icon)).toEqual(['112.gif', '113.gif', '114.gif', '115.gif', '116.gif', '117.gif', '118.gif', '119.gif', '120.gif', '121.gif'])
    expect(wu.army.map((unit) => unit.icon)).toEqual(['123.gif', '124.gif', '125.gif', '126.gif', '127.gif', '128.gif', '129.gif', '130.gif', '131.gif', '132.gif'])
    expect(wei.army.map((unit) => unit.name)).toEqual(['青州军', '禁卫甲士', '虎卫', '战鹰骑探', '骁骑营', '虎豹骑', '冲撞车', '霹雳车', '建筑师', '士族'])
    expect(shu.army.map((unit) => unit.name)).toEqual(['贪狼营', '麒麟卫', '青龙军', '飞鸢', '西凉铁骑', '南蛮象', '临冲车', '轰天雷', '木牛流马', '汉室宗亲'])
    expect(wu.army.map((unit) => unit.name)).toEqual(['影卫', '修罗', '密探', '神风', '朱雀骑', '霸王骑', '对楼车', '炬石车', '风水师', '太平术士'])
    expect(wei.general?.icon).toBe('general_tag_1.gif')
    expect(shu.general?.icon).toBe('general_tag_2.gif')
    expect(wu.general?.icon).toBe('general_tag_3.gif')
  })

  it('武将增援或出征后不再显示为本城直属武将', () => {
    const general = { id: 'g1', name: '曹操', level: 56 }
    const atHome = toCityGameViewModel(gameState({ general, generals: [general], generalAssignments: [{ id: 'main', generalId: 'g1', slot: 'main' }] }), 0)
    const reinforcing = toCityGameViewModel(gameState({ general, generals: [general], generalAssignments: [{ id: 'main', generalId: 'g1', slot: 'main' }, { id: 'reinforcement-r1-g1', generalId: 'g1', slot: 'reinforcement', moduleId: 'reinforcement', status: 'marching' }] }), 0)
    expect(atHome.general?.name).toBe('曹操')
    expect(reinforcing.general).toBeNull()
  })
})
