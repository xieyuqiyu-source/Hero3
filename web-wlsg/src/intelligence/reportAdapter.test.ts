/** 验证真实战报到官方双方、增援、情报隐藏和兵种图标结构的映射。 */
import { describe, expect, it } from 'vitest'
import type { BattleReportState } from '../game/types'
import { toOfficialBattleReport } from './reportAdapter'

/** 创建包含标准详情的攻击战报。 */
function report(): BattleReportState {
  return { id: 'r1', playerId: 'p1', viewType: 'attack', battleType: 'pvp', type: 'attack', result: 'attacker_victory', read: true, createdAt: '2026-07-13T08:00:00Z', title: '许昌 攻击 成都', detail: { id: 'r1', viewType: 'attack', sourceType: 'player_city', battleType: 'pvp', result: 'attacker_victory', winnerSide: 'attacker', title: '许昌 攻击 成都', occurredAt: '2026-07-13T08:00:00Z', primarySide: { role: 'attacker', playerId: 'p1', cityName: '许昌', faction: 'wei', power: 1200, generals: [{ id: 'g1', name: '曹操', level: 20, traits: [{ traitId: 'weiwu_haoling', name: 'weiwu_haoling' }, { traitId: 'weiwu_tongyu', name: 'weiwu_tongyu' }] }], units: [{ unitType: 'huWei', unitName: '虎卫', amountBefore: 100, dispatched: 100, lost: 10, survived: 90 }] }, secondarySide: { role: 'defender', playerId: 'p2', cityName: '成都', faction: 'shu', power: 900, generals: [{ id: 'g2', name: '刘备', level: 18, traits: [{ traitId: 'rende', name: 'rende' }, { traitId: 'renzhu_shouhu', name: 'renzhu_shouhu' }] }], units: [{ unitType: 'greedyWolf', unitName: '贪狼营', amountBefore: 80, dispatched: 80, lost: 40, survived: 40 }] }, rewards: { resources: { wood: 100 }, drops: [{ type: 'item', itemId: 'bag', name: '钻石锦囊', amount: 1 }, { type: 'item', itemId: 'bag', name: '钻石锦囊', amount: 2 }, { type: 'item', itemId: 'token', name: '征战令', amount: 4 }], cityGold: 88, generalExp: 25 }, traits: [{ traitId: 'wrong', traitName: '不应串方', ownerSide: 'primary', summary: '不应显示' }], visibility: { showEnemyRemainingUnits: true, showEnemyResources: true, showEnemyGenerals: true, showEnemyCityDefense: true }, read: true }, pvpReinforcements: [{ reinforcementId: 'rr1', fromPlayerId: 'p3', fromPlayerName: '援军城', faction: 'wu', troops: { shadowGuard: 30 } }], pvpReinforcementLosses: { rr1: { shadowGuard: 5 } } }
}

describe('官方战报适配', () => {
  it('按进攻、防守和增援逐方渲染并补齐十兵种列', () => {
    const model = toOfficialBattleReport(report())
    expect(model.sides.map((side) => side.role)).toEqual(['attacker', 'defender', 'reinforcement'])
    expect(model.sides[0].units).toHaveLength(10)
    expect(model.sides[0].units.find((unit) => unit.key === 'huWei')).toMatchObject({ icon: '/assets/official/report/units/103.gif', dispatched: 100, lost: 10 })
    expect(model.sides[2].units.find((unit) => unit.key === 'shadowGuard')).toMatchObject({ dispatched: 30, lost: 5, survived: 25 })
    expect(model.sides[0]).toMatchObject({ generalExp: 25, traitText: '魏武号令；魏武统御' })
    expect(model.sides[1]).toMatchObject({ generalExp: null, traitText: '仁德天下；仁主守护' })
    expect(model.sides[2].general).toBeNull()
    expect(model.resourceText).toContain('木材:100')
    expect(model.dropItems).toEqual([{ key: 'id:bag', name: '钻石锦囊', amount: 3 }, { key: 'id:token', name: '征战令', amount: 4 }])
    expect(model.dropsText).toContain('钻石锦囊 3 个')
    expect(model.cityGold).toBe(88)
    expect(model.feedbackText).toBe('-')
  })

  it('未打穿时隐藏防守方并展示真实情报不足原因', () => {
    const source = report()
    source.detail!.visibility = { showEnemyRemainingUnits: false, showEnemyResources: false, showEnemyGenerals: false, showEnemyCityDefense: false, reason: '总损伤低于25%' }
    source.detail!.secondarySide!.units = []
    const model = toOfficialBattleReport(source)
    expect(model.sides.map((side) => side.role)).toEqual(['attacker'])
    expect(model.visibilityReason).toBe('总损伤低于25%')
  })

  it('把后端防守情报状态码按真实阈值转换为中文提示', () => {
    const source = report()
    source.detail!.visibility = { showEnemyRemainingUnits: false, showEnemyResources: false, showEnemyGenerals: false, showEnemyCityDefense: false, reason: 'enemy_remaining_hidden', threshold: 0.25, actualLossRatio: 0.1 }
    const model = toOfficialBattleReport(source)
    expect(model.visibilityReason).toBe('对防守方造成的战损不足 25%，无法获得防守方剩余兵力情报')
    expect(model.visibilityReason).not.toContain('enemy_remaining_hidden')
  })

  it('在将领特性行完整展示触发阶段、比例和逐兵种实际数据', () => {
    const source = report()
    source.detail!.secondarySide!.units.push(
      { unitType: 'shadowGuard', unitName: '影卫', amountBefore: 1000, dispatched: 1000, lost: 193, survived: 807 },
      { unitType: 'xiuLuo', unitName: '修罗', amountBefore: 1000, dispatched: 1000, lost: 211, survived: 789 },
    )
    source.detail!.traits = [{
      traitId: 'laodang_yizhuang', traitName: '老当益壮', ownerSide: 'primary', generalId: 'g1',
      detail: { effectRate: 0.1, extraLosses: { shadowGuard: 193, xiuLuo: 211 }, triggerChance: 1 },
    }]
    const model = toOfficialBattleReport(source)
    expect(model.sides[0].traits).toEqual([{
      key: 'laodang_yizhuang-0', name: '老当益壮', phase: '战斗结算后',
      detailText: '效果比例：10%；追加损失：影卫 +193、修罗 +211；触发概率：100%',
    }])
    expect(model.sides[1].traits.map((trait) => trait.name)).toEqual(['仁德天下', '仁主守护'])
  })

  it('把掠夺特性的阶段、详情字段和资源键全部转换为中文', () => {
    const source = report()
    source.detail!.secondarySide!.generals = [{ id: 'g2', name: '孙权', level: 1 }]
    source.detail!.traits = [{ traitId: 'jiangdong_haoling', traitName: '江东号令', ownerSide: 'secondary', generalId: 'g2', detail: { effectRate: 0, plunderDelta: { food: -60, iron: -60, stone: -60, wood: -60 }, triggerChance: 1 } }]
    const model = toOfficialBattleReport(source)
    expect(model.sides[1].traits[0]).toMatchObject({
      name: '江东号令', phase: '掠夺结算',
      detailText: '效果比例：0%；掠夺资源修正：粮食 -60、铁矿 -60、泥土 -60、木材 -60；触发概率：100%',
    })
    expect(model.sides[1].traitText).not.toMatch(/plunderDelta|food|iron|stone|wood/)
  })

  it('NPC 没有将领时仍保留防守方标准将领占位栏', () => {
    const source = report()
    source.detail!.sourceType = 'npc_city'
    source.detail!.secondarySide!.targetType = undefined
    source.detail!.secondarySide!.generals = []
    const model = toOfficialBattleReport(source)
    expect(model.sides[1]).toMatchObject({ role: 'defender', general: null, showGeneralPlaceholder: true })
  })
})
