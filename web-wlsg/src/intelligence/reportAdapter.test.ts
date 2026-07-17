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
    source.detail!.secondarySide!.generals = [{ id: 'hidden-general', name: '隐藏将领', traits: [{ traitId: 'rende', name: '仁德天下', params: { reviveRate: 0.99 } }] }]
    source.detail!.secondarySide!.units = []
    const model = toOfficialBattleReport(source)
    expect(model.sides.map((side) => side.role)).toEqual(['attacker'])
    expect(model.visibilityReason).toBe('总损伤低于25%')
    expect(JSON.stringify(model)).not.toContain('99%')
  })

  it('未公开敌方情报时不会通过己方特性结果泄露敌军兵种和数量', () => {
    const source = report()
    source.detail!.visibility = { showEnemyRemainingUnits: false, showEnemyResources: false, showEnemyGenerals: false, showEnemyCityDefense: false, reason: 'enemy_remaining_hidden' }
    source.detail!.primarySide.generals![0].traits = [{ traitId: 'laodang_yizhuang', name: '老当益壮', params: { effectRate: 0.1 } }]
    source.detail!.traits = [{
      traitId: 'laodang_yizhuang', traitName: '老当益壮', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'g1',
      detail: {
        preBattleAffected: { greedyWolf: 1 }, suppressedUnits: { greedyWolf: 2 }, capturedUnits: { greedyWolf: 3 },
        modifiedUnits: { greedyWolf: 4 }, extraLosses: { greedyWolf: 5 }, targetExtraLosses: { greedyWolf: 6 },
        disabledTraits: { rende: 1 }, extraDamage: 7, totalSuppressed: 8, totalCaptured: 9,
        foodRatio: 3, triggerChance: 0.16, suppressRate: 0.2, futureEnemyStat: 99,
      },
    }]
    source.traitOutcomes = {
      laodang_yizhuang: {
        traitId: 'laodang_yizhuang', ownerSide: 'attacker', ownerGeneralId: 'g1', ownerPlayerId: 'p1',
        detail: source.detail!.traits[0].detail,
      },
    }

    const model = toOfficialBattleReport(source)
    expect(model.sides[0].traits[0].detailText).toBe('效果比例：10%')
    expect(JSON.stringify(model)).not.toMatch(/贪狼营|greedyWolf|仁德天下|\+5|追加损失|压制兵力/)
  })

  it('隐藏敌情时仍可展示明确属于己方的复活实际结果', () => {
    const source = report()
    source.detail!.visibility = { showEnemyRemainingUnits: false, showEnemyResources: false, showEnemyGenerals: false, showEnemyCityDefense: false, reason: 'enemy_remaining_hidden' }
    source.detail!.primarySide.generals![0].traits = [{ traitId: 'rende', name: '仁德天下', scope: 'self_army', params: { reviveRate: 0.5 } }]
    source.detail!.traits = [{ traitId: 'rende', traitName: '仁德天下', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'g1', detail: { revivedUnits: { huWei: 10 }, totalRevived: 10, triggerChance: 1 } }]
    source.traitOutcomes = { rende: { traitId: 'rende', ownerSide: 'attacker', ownerGeneralId: 'g1', ownerPlayerId: 'p1', scope: 'self_army', detail: source.detail!.traits[0].detail } }

    expect(toOfficialBattleReport(source).sides[0].traits[0].detailText).toBe('复活兵力：虎卫 +10；复活比例：50%；复活总数：10')
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
    expect(model.sides[0].traits.map((trait) => trait.name)).toEqual(['魏武号令', '魏武统御', '老当益壮'])
    expect(model.sides[0].traits.find((trait) => trait.name === '老当益壮')).toMatchObject({
      name: '老当益壮', phase: '战斗结算后',
      detailText: '效果比例：10%；追加损失：影卫 +193、修罗 +211；触发概率：100%',
    })
    expect(model.sides[1].traits.map((trait) => trait.name)).toEqual(['仁德天下', '仁主守护'])
  })

  it('未触发的将领特性也输出当前配置的详细数值', () => {
    const source = report()
    source.detail!.primarySide.generals![0].traits = [
      { traitId: 'weiwu_haoling', name: 'weiwu_haoling', scope: 'self_army', targetUnitType: 'huWei', params: { guardPerMinute: 500, maxGuardPerDay: 3000 } },
      { traitId: 'weiwu_tongyu', name: 'weiwu_tongyu', scope: 'self_army', targetUnitType: 'special', params: { attackBonusRate: 0.1, defenseBonusRate: 0.1 } },
    ]
    const model = toOfficialBattleReport(source)
    expect(model.sides[0].traits[0].detailText).toContain('每分钟产兵：500')
    expect(model.sides[0].traits[0].detailText).toContain('每日产兵上限：3,000')
    expect(model.sides[0].traits[1].detailText).toBe('攻击加成：10%；防御加成：10%')
  })

  it('特性行正确格式化速度比例、负修正和最低秒数', () => {
    const source = report()
    source.detail!.primarySide.generals![0].traits = [
      { traitId: 'baiyi_dujiang', name: '白衣渡江', params: { speedBonusRate: 0.2, warningDelayRate: 0.3, minMarchSeconds: 60 } },
      { traitId: 'jiangdong_haoling', name: '江东号令', params: { plunderBonusRate: -0.2 } },
    ]
    const model = toOfficialBattleReport(source)
    expect(model.sides[0].traits[0].detailText).toBe('行军速度加成：20%；预警延迟：30%；最低行军时间：60 秒')
    expect(model.sides[0].traits[1].detailText).toBe('掠夺收益修正：-20%')
  })

  it('已触发特性合并配置比例和本场实际作用数量且保留其他特性', () => {
    const source = report()
    source.detail!.primarySide.generals![0].traits = [
      { traitId: 'baibu_chuanyang', name: '百步穿杨', params: { triggerChance: 0.35, enemyDefenseReductionRate: 0.2 } },
      { traitId: 'laodang_yizhuang', name: '老当益壮', params: { effectRate: 0.1 } },
    ]
    source.detail!.traits = [{ traitId: 'laodang_yizhuang', traitName: '老当益壮', ownerSide: 'primary', generalId: 'g1', detail: { extraLosses: { greedyWolf: 12 } } }]
    const model = toOfficialBattleReport(source)
    expect(model.sides[0].traits.map((trait) => trait.name)).toEqual(['百步穿杨', '老当益壮'])
    expect(model.sides[0].traits[0].detailText).toContain('触发概率：35%')
    expect(model.sides[0].traits[0].detailText).toContain('降低敌方防御：20%')
    expect(model.sides[0].traits[1]).toMatchObject({ phase: '战斗结算后' })
    expect(model.sides[0].traits[1].detailText).toContain('效果比例：10%')
    expect(model.sides[0].traits[1].detailText).toContain('追加损失：贪狼营 +12')
  })

  it('攻防双方使用相同将领与特性时不会把一方实际结果串给另一方', () => {
    const source = report()
    source.detail!.primarySide.generals = [{ id: 'same-general', name: '镜像将领', traits: [{ traitId: 'laodang_yizhuang', name: '老当益壮', params: { effectRate: 0.1 } }] }]
    source.detail!.secondarySide!.generals = [{ id: 'same-general', name: '镜像将领', traits: [{ traitId: 'laodang_yizhuang', name: '老当益壮', params: { effectRate: 0.1 } }] }]
    source.detail!.traits = [{ traitId: 'laodang_yizhuang', traitName: '老当益壮', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'same-general', detail: { extraLosses: { greedyWolf: 12 } } }]
    source.traitOutcomes = { laodang_yizhuang: { traitId: 'laodang_yizhuang', ownerSide: 'attacker', ownerGeneralId: 'same-general', ownerPlayerId: 'p1', detail: { extraLosses: { greedyWolf: 12 } } } }
    const model = toOfficialBattleReport(source)
    expect(model.sides[0].traits[0].detailText).toContain('追加损失：贪狼营 +12')
    expect(model.sides[1].traits[0].detailText).toBe('效果比例：10%')
  })

  it('无标准 detail 的历史战报仍保留己方触发特性详情', () => {
    const source = report()
    delete source.detail
    source.defenderRevealed = true
    source.traitTriggered = ['huogong']
    source.traitOutcomes = { huogong: { traitId: 'huogong', name: '火烧赤壁', ownerSide: 'attacker', detail: { damagePercent: 0.25 } } }
    const model = toOfficialBattleReport(source)
    expect(model.sides[0].traits[0]).toMatchObject({ name: '火烧赤壁', detailText: '伤害比例：25%' })
  })

  it('只有旧 traitTriggered 名称时也不会丢失特性行', () => {
    const source = report()
    delete source.detail
    source.traitTriggered = ['rende']
    source.traitOutcomes = undefined
    expect(toOfficialBattleReport(source).sides[0].traits[0]).toMatchObject({ name: '仁德天下', detailText: '' })
  })

  it('无标准 detail 的历史防守战报把己方特性归到防守方', () => {
    const source = report()
    delete source.detail
    source.viewType = undefined
    source.type = 'defense'
    source.traitTriggered = ['rende']
    source.traitOutcomes = { rende: { traitId: 'rende', name: '仁德天下', detail: { reviveRate: 0.5 } } }
    source.pvpReinforcements = []

    const model = toOfficialBattleReport(source)
    expect(model.sides.map((side) => side.role)).toEqual(['attacker', 'defender'])
    expect(model.sides[0].traits).toEqual([])
    expect(model.sides[1].traits[0]).toMatchObject({ name: '仁德天下', detailText: '复活比例：50%' })
  })

  it('无标准 detail 的历史增援战报把己方特性归到增援方', () => {
    const source = report()
    delete source.detail
    source.viewType = 'reinforcement'
    source.type = 'reinforce'
    source.traitTriggered = ['rende']
    source.traitOutcomes = { rende: { traitId: 'rende', name: '仁德天下', detail: { reviveRate: 0.5 } } }
    source.pvpReinforcements = []

    const model = toOfficialBattleReport(source)
    expect(model.sides).toHaveLength(1)
    expect(model.sides[0]).toMatchObject({ role: 'reinforcement' })
    expect(model.sides[0].traits[0]).toMatchObject({ name: '仁德天下', detailText: '复活比例：50%' })
  })

  it('多支援军携带相同将领特性时只给真实拥有者合并实际结果', () => {
    const source = report()
    const sharedGeneral = { id: 'same-helper-general', name: '同名援将', traits: [{ traitId: 'laodang_yizhuang', name: '老当益壮', params: { effectRate: 0.1 } }] }
    source.pvpReinforcements = [
      { reinforcementId: 'rr1', fromPlayerId: 'p3', fromPlayerName: '援军甲', faction: 'wu', troops: { shadowGuard: 30 }, generals: [sharedGeneral] },
      { reinforcementId: 'rr2', fromPlayerId: 'p4', fromPlayerName: '援军乙', faction: 'wu', troops: { shadowGuard: 30 }, generals: [sharedGeneral] },
    ]
    source.detail!.traits = [{ traitId: 'laodang_yizhuang', traitName: '老当益壮', ownerSide: 'reinforcement', ownerRole: 'reinforcement', generalId: 'same-helper-general', detail: { reducedLosses: { shadowGuard: 5 } } }]
    source.traitOutcomes = { laodang_yizhuang: { traitId: 'laodang_yizhuang', ownerSide: 'reinforcement', ownerGeneralId: 'same-helper-general', ownerPlayerId: 'p3', detail: { reducedLosses: { shadowGuard: 5 } } } }
    const reinforcementSides = toOfficialBattleReport(source).sides.filter((side) => side.role === 'reinforcement')
    expect(reinforcementSides[0].traits[0].detailText).toContain('减少损失：影卫 +5')
    expect(reinforcementSides[1].traits[0].detailText).toBe('效果比例：10%')
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

  it('进攻方没有携带将领时仍保留固定将领占位栏', () => {
    const source = report()
    source.detail!.primarySide.generals = []
    const model = toOfficialBattleReport(source)
    expect(model.sides[0]).toMatchObject({ role: 'attacker', general: null, showGeneralPlaceholder: true })
  })

  it('增援视角展示完整攻防双方及各自将领经验', () => {
    const source = report()
    source.viewType = 'reinforcement'
    source.detail!.viewType = 'reinforcement'
    source.detail!.ownerSide = 'reinforcement'
    source.detail!.ownerPlayerId = 'p3'
    source.detail!.rewards.generalExp = 88
    source.detail!.primarySide.generals![0].generalExpGained = 30
    source.detail!.secondarySide!.generals![0].generalExpGained = 40
    source.pvpReinforcements![0].generalExpGained = 88
    source.pvpReinforcements![0].generals = [{ id: 'g3', name: '孙权', level: 16 }]

    const model = toOfficialBattleReport(source)
    expect(model.sides.map((side) => side.role)).toEqual(['attacker', 'defender', 'reinforcement'])
    expect(model.sides[0].generalExp).toBe(30)
    expect(model.sides[1].generalExp).toBe(40)
    expect(model.sides[2]).toMatchObject({ role: 'reinforcement', generalExp: 88, general: { id: 'g3', name: '孙权' } })
  })

  it('防守视角同时展示进攻方和防守方各自的武将经验', () => {
    const source = report()
    source.viewType = 'defense'
    source.detail!.viewType = 'defense'
    source.detail!.ownerSide = 'defender'
    source.detail!.primarySide.generals![0].generalExpGained = 380

    const model = toOfficialBattleReport(source)
    expect(model.sides[0].generalExp).toBe(380)
    expect(model.sides[1].generalExp).toBe(25)
  })
})
