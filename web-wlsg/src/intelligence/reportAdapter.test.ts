/** 验证真实战报到官方双方、增援、情报隐藏和兵种图标结构的映射。 */
import { describe, expect, it } from 'vitest'
import type { BattleReportState } from '../game/types'
import { traitLabel } from '../game/traitLabels'
import { formatGeneralProgress, toOfficialBattleReport } from './reportAdapter'

/** 创建包含标准详情的攻击战报。 */
function report(): BattleReportState {
  return { id: 'r1', playerId: 'p1', viewType: 'attack', battleType: 'pvp', type: 'attack', result: 'attacker_victory', read: true, createdAt: '2026-07-13T08:00:00Z', title: '许昌 攻击 成都', detail: { id: 'r1', viewType: 'attack', sourceType: 'player_city', battleType: 'pvp', result: 'attacker_victory', winnerSide: 'attacker', ownerSide: 'attacker', title: '许昌 攻击 成都', occurredAt: '2026-07-13T08:00:00Z', primarySide: { role: 'attacker', playerId: 'p1', cityName: '许昌', faction: 'wei', power: 1200, generals: [{ id: 'g1', name: '曹操', level: 20, traits: [{ traitId: 'weiwu_haoling', name: 'weiwu_haoling' }, { traitId: 'weiwu_tongyu', name: 'weiwu_tongyu' }] }], units: [{ unitType: 'huWei', unitName: '虎卫', amountBefore: 100, dispatched: 100, lost: 10, survived: 90 }] }, secondarySide: { role: 'defender', playerId: 'p2', cityName: '成都', faction: 'shu', power: 900, generals: [{ id: 'g2', name: '刘备', level: 18, traits: [{ traitId: 'rende', name: 'rende' }, { traitId: 'renzhu_shouhu', name: 'renzhu_shouhu' }] }], units: [{ unitType: 'greedyWolf', unitName: '贪狼营', amountBefore: 80, dispatched: 80, lost: 40, survived: 40 }] }, rewards: { resources: { wood: 100 }, drops: [{ type: 'item', itemId: 'bag', name: '钻石锦囊', amount: 1 }, { type: 'item', itemId: 'bag', name: '钻石锦囊', amount: 2 }, { type: 'item', itemId: 'token', name: '征战令', amount: 4 }], cityGold: 88, generalExp: 25, generalLevelBefore: 20, generalLevelAfter: 21 }, traits: [{ traitId: 'wrong', traitName: '不应串方', ownerSide: 'primary', summary: '不应显示' }], visibility: { showEnemyRemainingUnits: true, showEnemyResources: true, showEnemyGenerals: true, showEnemyCityDefense: true }, read: true }, pvpReinforcements: [{ reinforcementId: 'rr1', fromPlayerId: 'p3', fromPlayerName: '援军城', faction: 'wu', troops: { shadowGuard: 30 }, generalExpGained: 45 }], pvpReinforcementLosses: { rr1: { shadowGuard: 5 } } }
}

describe('官方战报适配', () => {
  it('全部正式特性 ID 都能转换为玩家可读中文名称', () => {
    const ids = `weiwu_haoling weiwu_tongyu yibing_touxi mouding_houfa meiren meihuo_raozhen huchi_chongzhen huhu_shengwei huzhu_xuezhan sizhandaodi jixing_benxi dunzhen_fangyu weizhen_zhenhe weizhen_xiaoyao shengui_zhicai guicai_yice wangzuo_zhicai neizheng_jingying rende renzhu_shouhu shuiyan_qijun wusheng_pojun zhenhe_quanjun wanren_nuhou qimen_dunjia wolong_mouzhi longdan_jiuyuan qijin_qichu xiliang_tuji tianshen_xiafan baibu_chuanyang laodang_yizhuang qibing_raohou gushou_hanzhong jiangdong_haoling jiangdong_gushou xiaobawang_zhuiji xiaobawang_tieqi huogong meizhoulang_junlue huoshao_lianying lianying_zengshang baiyi_dujiang baiyi_jixing kuairu_shandian xinyi_yonglie jinfan_jielue jinfan_qixi kurouji kurou_fanji`.split(' ')
    expect(ids).toHaveLength(50)
    for (const id of ids) expect(traitLabel(id)).not.toBe(id)
  })

  it('全部随机特性未命中时拥有快照不会补造触发时间线', () => {
    const randomTraitIds = [
      'yibing_touxi', 'huchi_chongzhen', 'sizhandaodi', 'weizhen_zhenhe',
      'shuiyan_qijun', 'zhenhe_quanjun', 'longdan_jiuyuan', 'xiliang_tuji',
      'baibu_chuanyang', 'qibing_raohou', 'jiangdong_gushou', 'xiaobawang_zhuiji',
      'huoshao_lianying', 'baiyi_dujiang', 'kuairu_shandian', 'kurouji',
    ]
    const source = report()
    source.detail!.primarySide.generals = [{
      id: 'random_general', name: '随机特性测试将领', level: 1,
      traits: randomTraitIds.map((traitId) => ({ traitId, name: traitLabel(traitId) })),
    }]
    source.detail!.traits = []
    const model = toOfficialBattleReport(source)
    expect(source.detail!.primarySide.generals[0].traits).toHaveLength(16)
    expect(model.sides.flatMap((side) => side.traits)).toEqual([])
    expect(model.sides.every((side) => side.traitText === '')).toBe(true)
  })

  it('孙权防守方向正确但江东固守未命中时只展示基础战斗结果', () => {
    const source = report()
    source.battleType = 'plunder'
    source.detail!.battleType = 'plunder'
    source.detail!.primarySide.faction = 'wei'
    source.detail!.primarySide.power = 10000
    source.detail!.primarySide.units = [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 }]
    source.detail!.secondarySide!.faction = 'wu'
    source.detail!.secondarySide!.power = 10000
    source.detail!.secondarySide!.generals = [{
      id: 'sunquan', name: '孙权', level: 1,
      traits: [{ traitId: 'jiangdong_gushou', name: '江东固守', params: { triggerChance: 0.5, defenseBonusRate: 0.5 } }],
    }]
    source.detail!.secondarySide!.units = [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 }]
    source.detail!.traits = []
    const model = toOfficialBattleReport(source)
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 10000, traits: [], traitText: '' })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 10000, general: { id: 'sunquan', name: '孙权', level: 1 }, traits: [], traitText: '' })
    expect(model.sides[0].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 1000, lost: 500, survived: 500 })
    expect(model.sides[1].units.find((unit) => unit.key === 'wuInfantry')).toMatchObject({ dispatched: 1000, lost: 500, survived: 500 })
  })

  it('按进攻、防守和增援逐方渲染并补齐十兵种列', () => {
    const model = toOfficialBattleReport(report())
    expect(model.sides.map((side) => side.role)).toEqual(['attacker', 'defender', 'reinforcement'])
    expect(model.sides[0].units).toHaveLength(10)
    expect(model.sides[0].units.find((unit) => unit.key === 'huWei')).toMatchObject({ icon: '/assets/official/report/units/103.gif', dispatched: 100, lost: 10 })
    expect(model.sides[2].units.find((unit) => unit.key === 'shadowGuard')).toMatchObject({ dispatched: 30, lost: 5, survived: 25 })
    expect(model.sides[0]).toMatchObject({ generalExp: 25, generalLevelBefore: 20, generalLevelAfter: 21, traitText: '', traits: [] })
    expect(model.sides[1]).toMatchObject({ generalExp: null, traitText: '', traits: [] })
    expect(model.sides[2]).toMatchObject({ general: null, generalExp: 45 })
    expect(model.resourceText).toContain('木材:100')
    expect(model.dropItems).toEqual([{ key: 'id:bag', name: '钻石锦囊', amount: 3 }, { key: 'id:token', name: '征战令', amount: 4 }])
    expect(model.dropsText).toContain('钻石锦囊 3 个')
    expect(model.cityGold).toBe(88)
    expect(model.feedbackText).toBe('-')
  })

  it('无将援军不借用城内主将、经验或特性', () => {
    const source = report()
    source.pvpReinforcements = [{
      reinforcementId: 'rein_without_general', fromPlayerId: 'player_without_general', fromPlayerName: '无将援军', faction: 'wei',
      troops: { huWei: 100 }, generals: [], generalExpGained: 0,
    }]
    source.pvpReinforcementLosses = { rein_without_general: { huWei: 50 } }
    source.detail!.traits = []

    const model = toOfficialBattleReport(source)
    expect(model.sides[2]).toMatchObject({ role: 'reinforcement', general: null, generalExp: 0, traits: [], traitText: '' })
    expect(model.sides[2].units.find((unit) => unit.key === 'huWei')).toMatchObject({ dispatched: 100, lost: 50, survived: 50 })
  })

  it('不同玩家使用同一将领增援时各自只展示自己的特性', () => {
    const source = report()
    source.pvpReinforcements = [
      {
        reinforcementId: 'rein_helper_a', fromPlayerId: 'helper_a', fromPlayerName: '常山甲', faction: 'shu',
        troops: { greedyWolf: 100 }, generals: [{ id: 'zhaoyun', name: '赵云', level: 1 }], generalExpGained: 10,
      },
      {
        reinforcementId: 'rein_helper_b', fromPlayerId: 'helper_b', fromPlayerName: '常山乙', faction: 'shu',
        troops: { greedyWolf: 100 }, generals: [{ id: 'zhaoyun', name: '赵云', level: 1 }], generalExpGained: 20,
      },
    ]
    source.pvpReinforcementLosses = {
      rein_helper_a: { greedyWolf: 40 },
      rein_helper_b: { greedyWolf: 30 },
    }
    source.detail!.traits = [
      {
        traitId: 'longdan_jiuyuan', traitName: '龙胆救援', ownerSide: 'reinforcement', ownerRole: 'reinforcement',
        ownerPlayerId: 'helper_a', generalId: 'zhaoyun', detail: { lossReductionRate: 0.2, reducedLosses: { greedyWolf: 10 } },
      },
      {
        traitId: 'longdan_jiuyuan', traitName: '龙胆救援', ownerSide: 'reinforcement', ownerRole: 'reinforcement',
        ownerPlayerId: 'helper_b', generalId: 'zhaoyun', detail: { lossReductionRate: 0.2, reducedLosses: { greedyWolf: 20 } },
      },
    ]

    const model = toOfficialBattleReport(source)
    expect(model.sides).toHaveLength(4)
    expect(model.sides[2]).toMatchObject({ name: '常山甲', general: { id: 'zhaoyun' }, generalExp: 10 })
    expect(model.sides[3]).toMatchObject({ name: '常山乙', general: { id: 'zhaoyun' }, generalExp: 20 })
    expect(model.sides[2].traits).toEqual([
      expect.objectContaining({ name: '龙胆救援', detailText: '设计减损比例：20%；减少损失：贪狼营 +10' }),
    ])
    expect(model.sides[3].traits).toEqual([
      expect.objectContaining({ name: '龙胆救援', detailText: '设计减损比例：20%；减少损失：贪狼营 +20' }),
    ])
    expect(model.sides.slice(2).flatMap((side) => side.traits)).toHaveLength(2)
    expect(model.sides[2].traits[0].key).toContain('helper_a')
    expect(model.sides[3].traits[0].key).toContain('helper_b')
    expect(model.sides[2].traits[0].key).not.toBe(model.sides[3].traits[0].key)
    expect(model.sides[2].traitText).toContain('设计减损比例：20%；减少损失：贪狼营 +10')
    expect(model.sides[3].traitText).toContain('设计减损比例：20%；减少损失：贪狼营 +20')
  })

  it('增援配置变更后只消费派出快照对应的后端真实时间线', () => {
    const source = report()
    source.pvpReinforcements = [{
      reinforcementId: 'rein_dispatch_config', fromPlayerId: 'helper_wei', fromPlayerName: '魏国援军', faction: 'wei',
      troops: { huWei: 100 },
      generals: [{ id: 'caocao', name: '曹操', level: 1, traits: [{
        traitId: 'weiwu_tongyu', name: '魏武统御', params: { defenseBonusRate: 0.1 },
        allowedSides: ['attacker', 'defender', 'reinforcement'], allowedScenes: ['attack'],
      }] }],
    }]
    source.pvpReinforcementLosses = { rein_dispatch_config: { huWei: 20 } }
    source.detail!.traits = [{
      traitId: 'weiwu_tongyu', traitName: '魏武统御', ownerSide: 'reinforcement', ownerRole: 'reinforcement',
      ownerPlayerId: 'helper_wei', generalId: 'caocao',
      detail: { defenseBonusRate: 0.1, infantryDefenseModifiedUnits: { huWei: 1 } },
    }]

    const dispatchedEnabled = toOfficialBattleReport(source)
    expect(dispatchedEnabled.sides[2].general).toMatchObject({ id: 'caocao' })
    expect(dispatchedEnabled.sides[2].general?.traits?.[0]).toMatchObject({
      params: { defenseBonusRate: 0.1 }, allowedSides: ['attacker', 'defender', 'reinforcement'], allowedScenes: ['attack'],
    })
    expect(dispatchedEnabled.sides[2].traits).toEqual([
      expect.objectContaining({ name: '魏武统御', detailText: '设计防御加成：10%；实际步防修正：虎卫 +1' }),
    ])

    source.detail!.traits = []
    const dispatchedDisabled = toOfficialBattleReport(source)
    expect(source.pvpReinforcements[0].generals?.[0].traits).toHaveLength(1)
    expect(dispatchedDisabled.sides[2]).toMatchObject({ general: { id: 'caocao' }, traits: [], traitText: '' })
  })

  it('黄巾协防增援沿用派出快照并只消费后端真实时间线', () => {
    const source = report()
    source.sourceType = 'yellow_turban'
    source.battleType = 'yellow_turban'
    source.detail!.sourceType = 'yellow_turban'
    source.detail!.battleType = 'yellow_turban'
    source.detail!.viewType = 'defense'
    source.pvpReinforcements = [{
      reinforcementId: 'yt_rein_dispatch', fromPlayerId: 'yellow_turban_helper', fromPlayerName: '黄巾协防军', faction: 'wei',
      troops: { huWei: 100 }, generals: [{ id: 'caocao', name: '曹操', level: 1, traits: [{
        traitId: 'weiwu_tongyu', name: '魏武统御', params: { defenseBonusRate: 0.1 }, allowedSides: ['reinforcement'],
      }] }],
    }]
    source.pvpReinforcementLosses = { yt_rein_dispatch: { huWei: 20 } }
    source.detail!.traits = [{
      traitId: 'weiwu_tongyu', traitName: '魏武统御', ownerSide: 'reinforcement', ownerRole: 'reinforcement',
      ownerPlayerId: 'yellow_turban_helper', generalId: 'caocao',
      detail: { defenseBonusRate: 0.1, infantryDefenseModifiedUnits: { huWei: 1 } },
    }]

    const triggered = toOfficialBattleReport(source)
    expect(triggered.sides.find((side) => side.role === 'reinforcement')).toMatchObject({
      general: { id: 'caocao' }, traits: [expect.objectContaining({ name: '魏武统御' })],
    })

    source.detail!.traits = []
    const snapshotOnly = toOfficialBattleReport(source)
    expect(source.pvpReinforcements[0].generals?.[0].traits?.[0].params).toEqual({ defenseBonusRate: 0.1 })
    expect(snapshotOnly.sides.find((side) => side.role === 'reinforcement')).toMatchObject({
      general: { id: 'caocao' }, traits: [], traitText: '',
    })
  })

  it('PVP 行军途中攻方关闭守方开启后只展示结算时真实特性', () => {
    const source = report()
    source.detail!.primarySide.generals = [{ id: 'caocao', name: '曹操', level: 1, traits: [] }]
    source.detail!.secondarySide!.generals = [{
      id: 'sunquan', name: '孙权', level: 1,
      traits: [{ traitId: 'jiangdong_gushou', name: '江东固守', params: { defenseBonusRate: 0.5 } }],
    }]
    source.detail!.traits = [{
      traitId: 'jiangdong_gushou', traitName: '江东固守', ownerSide: 'secondary', ownerRole: 'defender',
      ownerPlayerId: source.detail!.secondarySide!.playerId, generalId: 'sunquan',
      detail: { defenseBonusRate: 0.5, infantryDefenseModifiedUnits: { greedyWolf: 5 } },
    }]

    const model = toOfficialBattleReport(source)
    expect(model.sides[0]).toMatchObject({ general: { id: 'caocao', traits: [] }, traits: [], traitText: '' })
    expect(model.sides[1]).toMatchObject({
      general: { id: 'sunquan' },
      traits: [expect.objectContaining({ name: '江东固守', detailText: '设计防御加成：50%；实际步防修正：贪狼营 +5' })],
    })
    expect(model.sides.flatMap((side) => side.traits).map((trait) => trait.name)).toEqual(['江东固守'])
  })

  it('黄巾来袭途中主将配置变化后只展示到达时真实防御特性', () => {
    const source = report()
    source.sourceType = 'yellow_turban'
    source.battleType = 'yellow_turban'
    source.detail!.sourceType = 'yellow_turban'
    source.detail!.battleType = 'yellow_turban'
    source.detail!.viewType = 'defense'
    source.detail!.secondarySide!.generals = [{
      id: 'sunquan', name: '孙权', level: 1,
      traits: [{ traitId: 'jiangdong_gushou', name: '江东固守', params: { defenseBonusRate: 0.5 } }],
    }]
    source.detail!.traits = [{
      traitId: 'jiangdong_gushou', traitName: '江东固守', ownerSide: 'secondary', ownerRole: 'defender',
      ownerPlayerId: source.detail!.secondarySide!.playerId, generalId: 'sunquan',
      detail: { defenseBonusRate: 0.5, infantryDefenseModifiedUnits: { shadowGuard: 5 } },
    }]

    const enabled = toOfficialBattleReport(source)
    expect(enabled.sides[1]).toMatchObject({
      general: { id: 'sunquan' },
      traits: [expect.objectContaining({ name: '江东固守', detailText: '设计防御加成：50%；实际步防修正：影卫 +5' })],
    })

    source.detail!.secondarySide!.generals = [{ id: 'sunquan', name: '孙权', level: 1, traits: [] }]
    source.detail!.traits = []
    const disabled = toOfficialBattleReport(source)
    expect(disabled.sides[1]).toMatchObject({ general: { id: 'sunquan', traits: [] }, traits: [], traitText: '' })
  })

  it('黄巾来袭途中主将离城或归城只展示结算时真实快照', () => {
    const source = report()
    source.sourceType = 'yellow_turban'
    source.battleType = 'yellow_turban'
    source.viewType = 'defense'
    source.generalExpGained = 0
    source.detail!.sourceType = 'yellow_turban'
    source.detail!.battleType = 'yellow_turban'
    source.detail!.viewType = 'defense'
    source.detail!.ownerSide = 'defender'
    source.detail!.secondarySide!.generals = []
    source.detail!.traits = []
    source.detail!.rewards.generalExp = 0
    source.detail!.rewards.generalLevelBefore = 0
    source.detail!.rewards.generalLevelAfter = 0

    const away = toOfficialBattleReport(source)
    expect(away.sides[1]).toMatchObject({ role: 'defender', general: null, generalExp: 0, traits: [], traitText: '' })

    source.generalExpGained = 60
    source.generalLevelBefore = 1
    source.generalLevelAfter = 1
    source.detail!.secondarySide!.generals = [{ id: 'sunquan', name: '孙权', level: 1 }]
    source.detail!.traits = [{
      traitId: 'jiangdong_gushou', traitName: '江东固守', ownerSide: 'secondary', ownerRole: 'defender',
      ownerPlayerId: source.detail!.secondarySide!.playerId, generalId: 'sunquan',
      detail: { defenseBonusRate: 0.5, infantryDefenseModifiedUnits: { greedyWolf: 5 }, cavalryDefenseModifiedUnits: { greedyWolf: 4 } },
    }]
    source.detail!.rewards.generalExp = 60
    source.detail!.rewards.generalLevelBefore = 1
    source.detail!.rewards.generalLevelAfter = 1

    const returned = toOfficialBattleReport(source)
    expect(returned.sides[1]).toMatchObject({
      role: 'defender', general: { id: 'sunquan', name: '孙权' }, generalExp: 60,
      traits: [expect.objectContaining({
        name: '江东固守',
        detailText: '设计防御加成：50%；实际步防修正：贪狼营 +5；实际骑防修正：贪狼营 +4',
      })],
    })
  })

  it('黄巾来袭途中换将后只展示新主将战前快照和战后升级', () => {
    const source = report()
    source.sourceType = 'yellow_turban'
    source.battleType = 'yellow_turban'
    source.viewType = 'defense'
    source.generalExpGained = 100
    source.generalLevelBefore = 1
    source.generalLevelAfter = 2
    source.detail!.sourceType = 'yellow_turban'
    source.detail!.battleType = 'yellow_turban'
    source.detail!.viewType = 'defense'
    source.detail!.ownerSide = 'defender'
    source.detail!.secondarySide!.power = 1339
    source.detail!.secondarySide!.units = [{
      unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: 59, survived: 41,
    }]
    source.detail!.secondarySide!.generals = [{
      id: 'xiahouyuan', name: '夏侯渊', level: 1,
      traits: [{ traitId: 'jixing_benxi', name: '疾行奔袭', params: { unitAttackFlat: 18, unitSpeedFlat: 5 } }, { traitId: 'dunzhen_fangyu', name: '盾阵防御' }],
    }]
    source.detail!.traits = [{
      traitId: 'dunzhen_fangyu', traitName: '盾阵防御', ownerSide: 'secondary', ownerRole: 'defender',
      ownerPlayerId: source.detail!.secondarySide!.playerId, generalId: 'xiahouyuan',
      detail: { defenseBonusRate: 0.3, infantryDefenseModifiedUnits: { weiInfantry: 3 }, cavalryDefenseModifiedUnits: { weiInfantry: 2 }, triggerChance: 0.6 },
    }]
    source.detail!.rewards.generalExp = 100
    source.detail!.rewards.generalLevelBefore = 1
    source.detail!.rewards.generalLevelAfter = 2

    const model = toOfficialBattleReport(source)
    expect(model.sides.some((side) => side.general?.id === 'simayi')).toBe(false)
    expect(model.sides[1]).toMatchObject({
      role: 'defender', power: 1339,
      general: { id: 'xiahouyuan', name: '夏侯渊', level: 1 },
      generalExp: 100, generalLevelBefore: 1, generalLevelAfter: 2,
      traits: [expect.objectContaining({
        name: '盾阵防御', phase: '防守/增援战斗前',
        detailText: '设计防御加成：30%；实际步防修正：魏步兵 +3；实际骑防修正：魏步兵 +2；触发概率：60%',
      })],
      passiveTraits: [expect.objectContaining({ name: '疾行奔袭', phase: '永久被动', detailText: '骁骑营攻击 +18；移动 +5' })],
    })
  })

  it('历史战报只靠 traitOutcomes 也能区分两名玩家的同将领援军特性', () => {
    const source = report()
    source.detail = undefined
    source.defenderRevealed = true
    source.pvpReinforcements = [
      {
        reinforcementId: 'legacy_rein_a', fromPlayerId: 'legacy_helper_a', fromPlayerName: '历史甲', faction: 'shu',
        troops: { greedyWolf: 100 }, generals: [{ id: 'zhaoyun', name: '赵云', level: 1 }],
      },
      {
        reinforcementId: 'legacy_rein_b', fromPlayerId: 'legacy_helper_b', fromPlayerName: '历史乙', faction: 'shu',
        troops: { greedyWolf: 200 }, generals: [{ id: 'zhaoyun', name: '赵云', level: 1 }],
      },
    ]
    source.pvpReinforcementLosses = {
      legacy_rein_a: { greedyWolf: 80 },
      legacy_rein_b: { greedyWolf: 160 },
    }
    source.traitTriggered = ['longdan_jiuyuan', 'longdan_jiuyuan::reinforcement::zhaoyun']
    source.traitOutcomes = {
      longdan_jiuyuan: {
        traitId: 'longdan_jiuyuan', name: '龙胆救援', traitType: 'special', ownerSide: 'reinforcement',
        ownerPlayerId: 'legacy_helper_a', ownerGeneralId: 'zhaoyun', scope: 'reinforcement_self',
        detail: { lossReductionRate: 0.2, reducedLosses: { greedyWolf: 20 } },
      },
      'longdan_jiuyuan::reinforcement::zhaoyun': {
        traitId: 'longdan_jiuyuan', name: '龙胆救援', traitType: 'special', ownerSide: 'reinforcement',
        ownerPlayerId: 'legacy_helper_b', ownerGeneralId: 'zhaoyun', scope: 'reinforcement_self',
        detail: { lossReductionRate: 0.2, reducedLosses: { greedyWolf: 40 } },
      },
    }

    const model = toOfficialBattleReport(source)
    expect(model.sides[2].traits).toEqual([
      expect.objectContaining({ name: '龙胆救援', detailText: '设计减损比例：20%；减少损失：贪狼营 +20' }),
    ])
    expect(model.sides[3].traits).toEqual([
      expect.objectContaining({ name: '龙胆救援', detailText: '设计减损比例：20%；减少损失：贪狼营 +40' }),
    ])
    expect(model.sides.slice(2).flatMap((side) => side.traits)).toHaveLength(2)
    expect(model.sides[2].traits[0].key).toContain('legacy_helper_a')
    expect(model.sides[3].traits[0].key).toContain('legacy_helper_b')
    expect(model.sides[2].traits[0].key).not.toBe(model.sides[3].traits[0].key)
    expect(model.sides[2].traitText).toContain('设计减损比例：20%；减少损失：贪狼营 +20')
    expect(model.sides[3].traitText).toContain('设计减损比例：20%；减少损失：贪狼营 +40')
  })

  it('历史防守战报交换攻守快照并把经验和特性归给守城方', () => {
    const source = report()
    source.detail = undefined
    source.viewType = 'defense'
    source.ownerSide = 'defender'
    source.type = 'defense'
    source.playerName = '守城方'
    source.playerFaction = 'shu'
    source.playerPower = 1000
    source.targetName = '来袭方'
    source.defenderFaction = 'wei'
    source.enemyPower = 2000
    source.dispatchedUnits = { greedyWolf: 100 }
    source.lostUnits = { greedyWolf: 20 }
    source.survivedUnits = { greedyWolf: 80 }
    source.defenderUnits = { huWei: 200 }
    source.defenderLostUnits = { huWei: 50 }
    source.defenderRevealed = true
    source.pvpReinforcements = []
    source.pvpReinforcementLosses = {}
    source.generalExpGained = 30
    source.pvpAttackerGenerals = [{ id: 'caocao', name: '曹操', level: 1 }]
    source.pvpDefenderGenerals = [{ id: 'liubei', name: '刘备', level: 1 }]
    source.pvpWall = { faction: 'shu', level: 10, base: 1, multiplier: 1.2, factionDefenseBonus: 0.02, totalDefenseBonus: 0.2, hardness: 1.35, minDamagedLevelFrom20: 16, maxDamagedLevelFrom20: 17 }
    source.traitTriggered = ['renzhu_shouhu']
    source.traitOutcomes = {
      renzhu_shouhu: { traitId: 'renzhu_shouhu', name: '仁主守护', ownerSide: 'defender', ownerGeneralId: 'liubei', detail: { returnedUnits: { greedyWolf: 10 } } },
    }

    const model = toOfficialBattleReport(source)
    expect(model.sides).toHaveLength(2)
    expect(model.sides[0]).toMatchObject({ role: 'attacker', name: '来袭方', general: { id: 'caocao' }, power: 2000, generalExp: null })
    expect(model.sides[0].units.find((unit) => unit.key === 'huWei')).toMatchObject({ dispatched: 200, lost: 50, survived: 150 })
    expect(model.sides[0].traits).toEqual([])
    expect(model.sides[1]).toMatchObject({ role: 'defender', name: '守城方', general: { id: 'liubei' }, power: 1000, generalExp: 30 })
    expect(model.sides[1].units.find((unit) => unit.key === 'greedyWolf')).toMatchObject({ dispatched: 100, lost: 20, survived: 80 })
    expect(model.sides[1].traitText).toContain('仁主守护')
    expect(model.sides[1].traitText).toContain('返还兵力：贪狼营 +10')
    expect(model.wallText).toBe('城墙等级 Lv 10　防御加成 20.0%　硬度 1.35')
  })

  it('历史援军独立战报只生成一张自身卡片', () => {
    const source = report()
    source.detail = undefined
    source.playerId = 'helper'
    source.viewType = 'reinforcement'
    source.ownerSide = 'reinforcement'
    source.type = 'reinforce'
    source.playerName = '援军方'
    source.playerFaction = 'shu'
    source.dispatchedUnits = { greedyWolf: 100 }
    source.lostUnits = { greedyWolf: 80 }
    source.survivedUnits = { greedyWolf: 20 }
    source.defenderRevealed = true
    source.pvpReinforcements = [{
      reinforcementId: 'legacy_self', fromPlayerId: 'helper', fromPlayerName: '援军方', faction: 'shu',
      troops: { greedyWolf: 100 }, generals: [{ id: 'zhaoyun', name: '赵云', level: 1 }],
    }]
    source.pvpReinforcementLosses = { legacy_self: { greedyWolf: 80 } }
    source.traitTriggered = ['longdan_jiuyuan']
    source.traitOutcomes = {
      longdan_jiuyuan: { traitId: 'longdan_jiuyuan', name: '龙胆救援', ownerSide: 'reinforcement', ownerPlayerId: 'helper', ownerGeneralId: 'zhaoyun', detail: { reducedLosses: { greedyWolf: 20 } } },
    }

    const model = toOfficialBattleReport(source)
    expect(model.sides).toHaveLength(1)
    expect(model.sides[0]).toMatchObject({ role: 'reinforcement', name: '援军方', general: { id: 'zhaoyun' } })
    expect(model.sides[0].units.find((unit) => unit.key === 'greedyWolf')).toMatchObject({ dispatched: 100, lost: 80, survived: 20 })
    expect(model.sides[0].traitText).toContain('龙胆救援')
  })

  it('部分标准 PVP 扩展不会遮蔽旧援军快照和真实特性结果', () => {
    const source = report()
    source.detail!.traits = []
    source.detail!.extra = { pvp: { wall: { level: 10 } } }
    source.defenderRevealed = true
    source.pvpReinforcements = [{
      reinforcementId: 'partial_rein', fromPlayerId: 'partial_helper', fromPlayerName: '部分迁移援军', faction: 'shu',
      troops: { greedyWolf: 100 }, generals: [{ id: 'zhaoyun', name: '赵云', level: 1 }],
    }]
    source.pvpReinforcementLosses = { partial_rein: { greedyWolf: 80 } }
    source.traitTriggered = ['longdan_jiuyuan']
    source.traitOutcomes = {
      longdan_jiuyuan: {
        traitId: 'longdan_jiuyuan', name: '龙胆救援', ownerSide: 'reinforcement',
        ownerPlayerId: 'partial_helper', ownerGeneralId: 'zhaoyun', detail: { reducedLosses: { greedyWolf: 20 } },
      },
    }

    const model = toOfficialBattleReport(source)
    expect(model.sides[2].name).toBe('部分迁移援军')
    expect(model.sides[2].units.find((unit) => unit.key === 'greedyWolf')).toMatchObject({
      name: '贪狼营', dispatched: 100, lost: 80, survived: 20,
    })
    expect(model.sides[2].traitText).toContain('龙胆救援')
    expect(model.sides[2].traitText).toContain('减少损失：贪狼营 +20')
  })

  it('部分迁移战报在标准 traits 为空时回退旧结果且标准时间线有值时保持优先', () => {
    const source = report()
    source.detail!.primarySide.generals = [{ id: 'caocao', name: '曹操', level: 1 }]
    source.detail!.traits = []
    source.detail!.rewards = {}
    source.rewards = { wood: 100 }
    source.drops = [{ type: 'item', itemId: 'token', name: '征战令', amount: 2 }]
    source.overflowCityGold = 5
    source.overflow = { wood: 3 }
    source.generalExpGained = 30
    source.generalLevelBefore = 1
    source.generalLevelAfter = 2
    source.traitTriggered = ['weiwu_tongyu']
    source.traitOutcomes = {
      weiwu_tongyu: {
        traitId: 'weiwu_tongyu', name: '魏武统御', ownerSide: 'attacker', ownerGeneralId: 'caocao',
        detail: { attackBonusRate: 0.1, attackModifiedUnits: { huWei: 1 } },
      },
    }

    const fallbackModel = toOfficialBattleReport(source)
    expect(fallbackModel.sides[0].traitText).toContain('魏武统御')
    expect(fallbackModel.sides[0].traitText).toContain('设计攻击加成：10%；实际攻击修正：虎卫 +1')
    expect(fallbackModel.resourceText).toBe('木材:100')
    expect(fallbackModel.dropsText).toBe('征战令 2 个')
    expect(fallbackModel.cityGold).toBe(5)
    expect(fallbackModel.sides[0].generalExp).toBe(30)

    source.detail!.traits = [{
      traitId: 'standard_only', traitName: '标准时间线', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'caocao',
      detail: { attackBonusRate: 0.2, attackModifiedUnits: { huWei: 2 } },
    }]
    source.detail!.rewards = { resources: { wood: 7 }, drops: [], cityGold: 0, generalExp: 0, generalLevelBefore: 0, generalLevelAfter: 0, overflow: {} }
    const standardModel = toOfficialBattleReport(source)
    expect(standardModel.sides[0].traitText).toContain('标准时间线')
    expect(standardModel.sides[0].traitText).toContain('实际攻击修正：虎卫 +2')
    expect(standardModel.sides[0].traitText).not.toContain('魏武统御')
    expect(standardModel.resourceText).toBe('木材:7')
    expect(standardModel.dropsText).toBe('无')
    expect(standardModel.cityGold).toBe(0)
    expect(standardModel.sides[0].generalExp).toBe(0)
  })

  it('经验展示只在真实升级时附加等级变化', () => {
    expect(formatGeneralProgress(25, 20, 21)).toBe('+25 · Lv.20 → Lv.21')
    expect(formatGeneralProgress(25, 20, 20)).toBe('+25')
  })

  it('黄巾协防援军按后端字段展示跨多级升级', () => {
    const source = report()
    source.viewType = 'defense'
    source.detail!.viewType = 'defense'
    source.detail!.sourceType = 'yellow_turban'
    source.detail!.ownerSide = 'defender'
    source.pvpReinforcements = [{
      reinforcementId: 'rein_yellow_turban', fromPlayerId: 'player_helper', fromPlayerName: '协防玩家', faction: 'wu',
      troops: { shadowGuard: 400 }, generals: [{ id: 'sunquan', name: '孙权', level: 1, exp: 99 }],
      generalExpGained: 600, generalLevelBefore: 1, generalLevelAfter: 4,
    }]
    source.pvpReinforcementLosses = { rein_yellow_turban: { shadowGuard: 258 } }

    const model = toOfficialBattleReport(source)
    expect(model.sides[2]).toMatchObject({
      role: 'reinforcement', general: { id: 'sunquan', name: '孙权', level: 1 },
      generalExp: 600, generalLevelBefore: 1, generalLevelAfter: 4,
    })
    expect(formatGeneralProgress(model.sides[2].generalExp, model.sides[2].generalLevelBefore, model.sides[2].generalLevelAfter)).toBe('+600 · Lv.1 → Lv.4')
  })

  it('重试返回相同援军快照时只展示一次权威经验', () => {
    const source = report()
    const snapshot = {
      reinforcementId: 'rein_retry', fromPlayerId: 'player_retry', fromPlayerName: '协防玩家', faction: 'wei',
      troops: { huWei: 100 }, generals: [{ id: 'xiahouyuan', name: '夏侯渊', level: 1, exp: 99 }],
      generalExpGained: 110, generalLevelBefore: 1, generalLevelAfter: 2,
    }
    source.pvpReinforcements = [snapshot, snapshot]
    source.pvpReinforcementLosses = { rein_retry: { huWei: 70 } }

    const model = toOfficialBattleReport(source)
    expect(model.sides.filter((side) => side.role === 'reinforcement')).toHaveLength(1)
    expect(model.sides[2]).toMatchObject({ generalExp: 110, generalLevelBefore: 1, generalLevelAfter: 2 })
  })

  it('防守战报把经验和等级变化归给防守武将', () => {
    const source = report()
    source.viewType = 'defense'
    source.detail!.viewType = 'defense'
    source.detail!.ownerSide = 'defender'
    const model = toOfficialBattleReport(source)
    expect(model.sides[0]).toMatchObject({ role: 'attacker', generalExp: null, generalLevelBefore: null, generalLevelAfter: null })
    expect(model.sides[1]).toMatchObject({ role: 'defender', generalExp: 25, generalLevelBefore: 20, generalLevelAfter: 21 })
  })

  it('副本防守波保持敌军进攻侧与玩家防守侧兵力、经验一致', () => {
    const source = report()
    source.viewType = 'defense'
    source.detail!.viewType = 'defense'
    source.detail!.sourceType = 'dungeon'
    source.detail!.battleType = 'dungeon_reincarnation_defense'
    source.detail!.ownerSide = 'defender'
    source.detail!.primarySide = {
      role: 'attacker', faction: 'shu', power: 1200, generals: [],
      units: [{ unitType: 'greedyWolf', unitName: '贪狼营', amountBefore: 100, dispatched: 100, lost: 100, survived: 0 }],
    }
    source.detail!.secondarySide = {
      role: 'defender', playerId: 'p1', faction: 'wei', power: 3500,
      generals: [{ id: 'caocao', name: '曹操', level: 3 }],
      units: [{ unitType: 'qingZhouArmy', unitName: '青州军', amountBefore: 500, dispatched: 500, lost: 109, survived: 391 }],
    }
    source.detail!.rewards = { generalExp: 300, generalLevelBefore: 1, generalLevelAfter: 3 }
    const model = toOfficialBattleReport(source)
    expect(model.sides[0]).toMatchObject({ role: 'attacker', general: null, generalExp: null })
    expect(model.sides[0].units.find((unit) => unit.key === 'greedyWolf')).toMatchObject({ dispatched: 100, lost: 100, survived: 0 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', general: { id: 'caocao', name: '曹操', level: 3 }, generalExp: 300, generalLevelBefore: 1, generalLevelAfter: 3 })
    expect(model.sides[1].units.find((unit) => unit.key === 'qingZhouArmy')).toMatchObject({ dispatched: 500, lost: 109, survived: 391 })
  })

  it('轮回副本按后端时间线展示战前扣兵、攻击修正和仁主复活', () => {
    const source = report()
    source.detail!.sourceType = 'dungeon'
    source.detail!.battleType = 'dungeon_reincarnation_attack'
    source.detail!.traits = [
      { traitId: 'shuiyan_qijun', traitName: '水淹七军', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'g1', detail: { effectRate: 0.35, maxAffectedRate: 0.35, preBattleAffected: { greedyWolf: 350 }, triggerChance: 1 } },
      { traitId: 'meiren', traitName: '美人心计', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'g1', detail: { attackBonusRate: 0.25, attackModifiedUnits: { huWei: 3 }, triggerChance: 0.5 } },
      { traitId: 'renzhu_shouhu', traitName: '仁主守护', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'g1', detail: { effectRate: 0.35, revivedUnits: { huWei: 35 }, totalRevived: 35, triggerChance: 0.6 } },
    ]
    const model = toOfficialBattleReport(source)
    expect(model.sides[0].traits).toEqual([
      { key: 'shuiyan_qijun-0', name: '水淹七军', phase: '战斗前', detailText: '设计效果比例：35%；设计最大影响比例：35%；战前真实伤亡：贪狼营 +350；触发概率：100%' },
      { key: 'meiren-1', name: '美人心计', phase: '主动进攻战斗前', detailText: '设计攻击加成：25%；实际攻击修正：虎卫 +3；触发概率：50%' },
      { key: 'renzhu_shouhu-2', name: '仁主守护', phase: '进攻/防守/增援战斗结束后', detailText: '设计效果比例：35%；复活兵力：虎卫 +35；复活总数：35；触发概率：60%' },
    ])
    expect(model.sides[1].traits).toEqual([])
  })

  it('轮回副本全正式特性矩阵只消费后端实际触发时间线', () => {
    const bothSides = 'weiwu_tongyu yibing_touxi weizhen_zhenhe guicai_yice renzhu_shouhu shuiyan_qijun zhenhe_quanjun qimen_dunjia wolong_mouzhi xiliang_tuji laodang_yizhuang huoshao_lianying lianying_zengshang kurouji kurou_fanji'.split(' ')
    const attackerOnly = 'meiren meihuo_raozhen huchi_chongzhen sizhandaodi weizhen_xiaoyao wusheng_pojun wanren_nuhou baibu_chuanyang qibing_raohou xiaobawang_tieqi huogong meizhoulang_junlue'.split(' ')
    const defenderOnly = 'mouding_houfa dunzhen_fangyu huzhu_xuezhan longdan_jiuyuan gushou_hanzhong jiangdong_gushou'.split(' ')
    const nonBattle = 'weiwu_haoling jixing_benxi huhu_shengwei shengui_zhicai rende wangzuo_zhicai neizheng_jingying qijin_qichu tianshen_xiafan jiangdong_haoling xiaobawang_zhuiji baiyi_dujiang baiyi_jixing kuairu_shandian xinyi_yonglie jinfan_jielue jinfan_qixi'.split(' ')
    const allTraitIds = [...bothSides, ...attackerOnly, ...defenderOnly, ...nonBattle]
    expect(new Set(allTraitIds).size).toBe(50)

    const attackSource = report()
    attackSource.detail!.sourceType = 'dungeon'
    attackSource.detail!.battleType = 'dungeon_reincarnation_attack'
    attackSource.detail!.traits = [...bothSides, ...attackerOnly].map((traitId) => ({
      traitId, traitName: traitLabel(traitId), ownerSide: 'primary', ownerRole: 'attacker', generalId: 'g1',
    }))
    const attackModel = toOfficialBattleReport(attackSource)
    expect(attackModel.sides[0].traits).toHaveLength(27)
    expect(attackModel.sides[0].traits.map((trait) => trait.name)).toEqual([...bothSides, ...attackerOnly].map((traitId) => traitLabel(traitId)))
    expect(attackModel.sides[1].traits).toEqual([])

    const defenseSource = report()
    defenseSource.detail!.sourceType = 'dungeon'
    defenseSource.detail!.battleType = 'dungeon_reincarnation_defense'
    defenseSource.detail!.traits = [...bothSides, ...defenderOnly].map((traitId) => ({
      traitId, traitName: traitLabel(traitId), ownerSide: 'secondary', ownerRole: 'defender', generalId: 'g2',
    }))
    const defenseModel = toOfficialBattleReport(defenseSource)
    expect(defenseModel.sides[0].traits).toEqual([])
    expect(defenseModel.sides[1].traits).toHaveLength(21)
    expect(defenseModel.sides[1].traits.map((trait) => trait.name)).toEqual([...bothSides, ...defenderOnly].map((traitId) => traitLabel(traitId)))

    const snapshotSource = report()
    snapshotSource.detail!.sourceType = 'dungeon'
    snapshotSource.detail!.primarySide.generals = [{
      id: 'g1', name: '矩阵将领', level: 1,
      traits: allTraitIds.map((traitId) => ({ traitId, name: traitLabel(traitId) })),
    }]
    snapshotSource.detail!.traits = []
    const snapshotModel = toOfficialBattleReport(snapshotSource)
    expect(snapshotSource.detail!.primarySide.generals[0].traits).toHaveLength(50)
    expect(snapshotModel.sides.flatMap((side) => side.traits)).toEqual([])
  })

  it('轮回攻防无将领时不借用城内主将快照、经验或特性', () => {
    for (const viewType of ['attack', 'defense'] as const) {
      const source = report()
      source.viewType = viewType
      source.detail!.viewType = viewType
      source.detail!.sourceType = 'dungeon'
      source.detail!.battleType = `dungeon_reincarnation_${viewType}`
      source.detail!.primarySide.generals = []
      source.detail!.secondarySide!.generals = []
      source.detail!.traits = []
      source.detail!.rewards = { generalExp: 0, generalLevelBefore: 0, generalLevelAfter: 0 }
      source.pvpAttackerGenerals = []
      source.pvpDefenderGenerals = []
      source.generalExpGained = 0
      source.generalLevelBefore = 0
      source.generalLevelAfter = 0

      const model = toOfficialBattleReport(source)
      expect(model.sides.slice(0, 2).every((side) => side.general === null)).toBe(true)
      expect(model.sides.slice(0, 2).every((side) => side.generalExp === null || side.generalExp === 0)).toBe(true)
      expect(model.sides.flatMap((side) => side.traits)).toEqual([])
      expect(model.sides.every((side) => side.traitText === '')).toBe(true)
    }
  })

  it('郭嘉在轮回攻防平局中展示后端真实阵亡与复活', () => {
    for (const [generalId, generalName, traitId, traitName] of [
      ['guojia', '郭嘉', 'guicai_yice', '鬼才遗策'],
    ] as const) {
      for (const viewType of ['attack', 'defense'] as const) {
        const source = report()
        source.viewType = viewType
        source.result = 'draw'
        source.battleType = `dungeon_reincarnation_${viewType}`
        source.detail!.viewType = viewType
        source.detail!.sourceType = 'dungeon'
        source.detail!.battleType = `dungeon_reincarnation_${viewType}`
        source.detail!.result = 'draw'
        source.detail!.winnerSide = 'draw'
        source.detail!.ownerSide = viewType === 'attack' ? 'attacker' : 'defender'
        const playerSide = {
          role: viewType === 'attack' ? 'attacker' as const : 'defender' as const, faction: 'wei', power: 1000,
          generals: [{ id: generalId, name: generalName, level: 1, traits: [{ traitId, name: traitName }] }],
          units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: 100, survived: 22 }],
        }
        const enemySide = {
          role: viewType === 'attack' ? 'defender' as const : 'attacker' as const, faction: 'shu', power: 1000, generals: [],
          units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 100, dispatched: 100, lost: 100, survived: 0 }],
        }
        source.detail!.primarySide = viewType === 'attack' ? playerSide : enemySide
        source.detail!.secondarySide = viewType === 'attack' ? enemySide : playerSide
        source.detail!.rewards = { generalExp: 100, generalLevelBefore: 1, generalLevelAfter: 1 }
        source.detail!.traits = [{
          traitId, traitName, generalId,
          ownerSide: viewType === 'attack' ? 'primary' : 'secondary',
          ownerRole: viewType === 'attack' ? 'attacker' : 'defender',
          detail: { effectRate: 0.22, actualLostUnits: { weiInfantry: 100 }, revivedUnits: { weiInfantry: 22 }, totalRevived: 22 },
        }]

        const model = toOfficialBattleReport(source)
        const ownedSide = model.sides.find((side) => side.role === (viewType === 'attack' ? 'attacker' : 'defender'))
        const enemySideModel = model.sides.find((side) => side.role === (viewType === 'attack' ? 'defender' : 'attacker'))
        expect(ownedSide).toMatchObject({ power: 1000, general: { id: generalId }, generalExp: 100 })
        expect(ownedSide?.units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 100, lost: 100, survived: 22 })
        expect(enemySideModel?.units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 100, lost: 100, survived: 0 })
        expect(ownedSide?.traits[0]).toMatchObject({
          name: '鬼才遗策',
          phase: '进攻/防守/增援战斗结束后',
          detailText: '设计效果比例：22%；本场真实阵亡：魏步兵 +100；复活兵力：魏步兵 +22；复活总数：22',
        })
      }
    }
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
    expect(model.sides[0].traits[0].detailText).toBe('设计效果比例：10%')
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

  it('敌方将领情报隐藏后不从旧快照补回将领或特性', () => {
    const source = report()
    source.detail!.visibility = {
      showEnemyRemainingUnits: true,
      showEnemyResources: false,
      showEnemyGenerals: false,
      showEnemyCityDefense: false,
      reason: 'enemy_general_hidden',
    }
    source.detail!.secondarySide!.generals = []
    source.detail!.traits = [{
      traitId: 'weiwu_tongyu', traitName: '魏武统御', ownerSide: 'primary', ownerRole: 'attacker',
      generalId: 'g1', detail: { defenseBonusRate: 0.1 },
    }]
    source.pvpDefenderGenerals = [{
      id: 'g2', name: '刘备', level: 18,
      traits: [{ traitId: 'rende', name: '仁德' }],
    }]

    const model = toOfficialBattleReport(source)
    const attacker = model.sides.find((side) => side.role === 'attacker')
    const defender = model.sides.find((side) => side.role === 'defender')
    expect(model.sides).toHaveLength(3)
    expect(attacker?.traits).toEqual([
      expect.objectContaining({ name: '魏武统御' }),
    ])
    expect(defender).toMatchObject({ role: 'defender', general: null, traits: [], traitText: '' })
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
	      detail: { effectRate: 0.1, extraLosses: { shadowGuard: 193, xiuLuo: 211 }, triggerCount: 2, triggerChance: 1 },
    }]
    const model = toOfficialBattleReport(source)
    expect(model.sides[0].traits.map((trait) => trait.name)).toEqual(['老当益壮'])
    expect(model.sides[0].traits.find((trait) => trait.name === '老当益壮')).toMatchObject({
      name: '老当益壮', phase: '战斗结算后',
      detailText: '设计效果比例：10%；追加损失：影卫 +193、修罗 +211；触发场次：2；触发概率：100%',
    })
    expect(model.sides[1].traits).toEqual([])
  })

  it('战报按后端结果准确展示 35%、50% 和 100% 触发概率', () => {
    for (const [traitId, chance, expected] of [['yibing_touxi', 0.35, '35%'], ['zhenhe_quanjun', 0.5, '50%'], ['huogong', 1, '100%']] as const) {
      const source = report()
      source.detail!.primarySide.generals = [{ id: traitId, name: traitId, level: 1 }]
      source.detail!.traits = [{ traitId, traitName: traitId, ownerSide: 'primary', generalId: traitId, detail: { triggerChance: chance } }]
      const model = toOfficialBattleReport(source)
      expect(model.sides[0].traits[0].detailText).toBe(`触发概率：${expected}`)
    }
  })

  it('四项 NPC 战后追加伤害展示后端返回的逐兵种实际增量', () => {
    const cases = [
      ['laodang_yizhuang', '老当益壮', 'huangzhong', '黄忠', 'extraLosses', 100, 0.1],
      ['huoshao_lianying', '火烧联营', 'luxun', '陆逊', 'targetExtraLosses', 963, 1],
      ['lianying_zengshang', '连营增伤', 'luxun', '陆逊', 'targetExtraLosses', 100, 0.1],
      ['kurou_fanji', '苦肉反击', 'huanggai', '黄盖', 'extraLosses', 100, 0.1],
    ] as const
    for (const [traitId, traitName, generalId, generalName, detailKey, amount, effectRate] of cases) {
      const source = report()
      source.detail!.sourceType = 'npc_city'
      source.detail!.primarySide.generals = [{ id: generalId, name: generalName, level: 1 }]
      source.detail!.traits = [{
        traitId, traitName, ownerSide: 'primary', ownerRole: 'attacker', generalId,
        detail: { effectRate, [detailKey]: { shadowGuard: amount }, triggerChance: 1 },
      }]
      const model = toOfficialBattleReport(source)
      expect(model.sides[0].traits[0]).toMatchObject({
        name: traitName, phase: '战斗结算后', detailText: `设计效果比例：${effectRate * 100}%；${detailKey === 'targetExtraLosses' ? '目标兵种追加损失' : '追加损失'}：影卫 +${amount}；触发概率：100%`,
      })
      expect(model.sides[1].traits).toEqual([])
    }
  })

  it('火烧联营展示封顶后的实际追加值而不是错误复述配置百分比', () => {
    const source = report()
    source.detail!.primarySide.generals = [{ id: 'luxun', name: '陆逊', level: 1 }]
    source.detail!.secondarySide!.units = [{ unitType: 'shadowGuard', unitName: '影卫', amountBefore: 1000, dispatched: 1000, lost: 1000, survived: 0 }]
    source.detail!.traits = [{
      traitId: 'huoshao_lianying', traitName: '火烧联营', ownerSide: 'primary', generalId: 'luxun',
      detail: { effectRate: 1, targetExtraLosses: { shadowGuard: 963 }, triggerChance: 1 },
    }]
    const model = toOfficialBattleReport(source)
    expect(model.sides[1].units.find((unit) => unit.key === 'shadowGuard')).toMatchObject({ dispatched: 1000, lost: 1000, survived: 0 })
    expect(model.sides[0].traits[0]).toMatchObject({
      detailText: '设计效果比例：100%；目标兵种追加损失：影卫 +963；触发概率：100%',
    })
  })

  it('陆逊双追加伤害只展示本场产生实际增量的特性', () => {
    const bonusOnly = report()
    bonusOnly.detail!.primarySide.generals = [{
      id: 'luxun', name: '陆逊', level: 1,
      traits: [{ traitId: 'huoshao_lianying', name: '火烧联营' }, { traitId: 'lianying_zengshang', name: '连营增伤' }],
    }]
    bonusOnly.detail!.traits = [{
      traitId: 'lianying_zengshang', traitName: '连营增伤', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'luxun',
      detail: { effectRate: 0.1, targetExtraLosses: { greedyWolf: 100 }, triggerChance: 1 },
    }]
    const bonusModel = toOfficialBattleReport(bonusOnly)
    expect(bonusModel.sides[0].traits).toHaveLength(1)
    expect(bonusModel.sides[0].traits[0]).toMatchObject({
      name: '连营增伤', phase: '战斗结算后',
      detailText: '设计效果比例：10%；目标兵种追加损失：贪狼营 +100；触发概率：100%',
    })

    const fireCapped = report()
    fireCapped.detail!.primarySide.generals = bonusOnly.detail!.primarySide.generals
    fireCapped.detail!.traits = [{
      traitId: 'huoshao_lianying', traitName: '火烧联营', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'luxun',
      detail: { effectRate: 1, targetExtraLosses: { greedyWolf: 436 }, triggerChance: 1 },
    }]
    const fireModel = toOfficialBattleReport(fireCapped)
    expect(fireModel.sides[0].traits).toHaveLength(1)
    expect(fireModel.sides[0].traits[0]).toMatchObject({
      name: '火烧联营', phase: '战斗结算后',
      detailText: '设计效果比例：100%；目标兵种追加损失：贪狼营 +436；触发概率：100%',
    })
    expect(fireModel.sides[0].traits.some((trait) => trait.name === '连营增伤')).toBe(false)
  })

  it('陆逊火烧未命中时连营增伤仍按真实兵力追加损失', () => {
    const source = report()
    source.battleType = 'plunder'
    source.detail!.battleType = 'plunder'
    source.detail!.primarySide.faction = 'wu'
    source.detail!.primarySide.power = 10000
    source.detail!.primarySide.generals = [{
      id: 'luxun', name: '陆逊', level: 1,
      traits: [{ traitId: 'huoshao_lianying', name: '火烧联营' }, { traitId: 'lianying_zengshang', name: '连营增伤' }],
    }]
    source.detail!.primarySide.units = [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 }]
    source.detail!.secondarySide!.faction = 'wei'
    source.detail!.secondarySide!.power = 10000
    source.detail!.secondarySide!.units = [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 600, survived: 400 }]
    source.detail!.rewards.generalExp = 600
    source.detail!.traits = [{
      traitId: 'lianying_zengshang', traitName: '连营增伤', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'luxun',
      detail: { effectRate: 0.1, targetExtraLosses: { weiInfantry: 100 } },
    }]
    const model = toOfficialBattleReport(source)
    expect(source.detail!.primarySide.generals![0]!.traits?.map((trait) => trait.traitId)).toEqual(['huoshao_lianying', 'lianying_zengshang'])
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 10000, generalExp: 600 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 10000 })
    expect(model.sides[0].units.find((unit) => unit.key === 'wuInfantry')).toMatchObject({ dispatched: 1000, lost: 500, survived: 500 })
    expect(model.sides[1].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 1000, lost: 600, survived: 400 })
    expect(model.sides[0].traits).toEqual([{
      key: 'lianying_zengshang-0', name: '连营增伤', phase: '战斗结算后',
      detailText: '设计效果比例：10%；目标兵种追加损失：魏步兵 +100',
    }])
    expect(model.sides.flatMap((side) => side.traits).some((trait) => trait.name === '火烧联营')).toBe(false)
  })

  it('防守陆逊火烧未命中时连营增伤只追加来袭步兵损失', () => {
    const source = report()
    source.viewType = 'defense'
    source.result = 'draw'
    source.battleType = 'plunder'
    source.detail!.viewType = 'defense'
    source.detail!.result = 'draw'
    source.detail!.winnerSide = 'none'
    source.detail!.ownerSide = 'defender'
    source.detail!.battleType = 'plunder'
    source.detail!.primarySide.faction = 'shu'
    source.detail!.primarySide.power = 10000
    source.detail!.primarySide.generals = [{ id: 'liubei', name: '刘备', level: 1 }]
    source.detail!.primarySide.units = [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 600, survived: 400 }]
    source.detail!.secondarySide!.faction = 'wu'
    source.detail!.secondarySide!.power = 10000
    source.detail!.secondarySide!.generals = [{
      id: 'luxun', name: '陆逊', level: 1,
      traits: [{ traitId: 'huoshao_lianying', name: '火烧联营' }, { traitId: 'lianying_zengshang', name: '连营增伤' }],
    }]
    source.detail!.secondarySide!.units = [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 }]
    source.detail!.rewards.generalExp = 600
    source.detail!.traits = [{
      traitId: 'lianying_zengshang', traitName: '连营增伤', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'luxun',
      detail: { effectRate: 0.1, targetExtraLosses: { shuInfantry: 100 } },
    }]
    const model = toOfficialBattleReport(source)
    expect(source.detail!.secondarySide!.generals![0]!.traits?.map((trait) => trait.traitId)).toEqual(['huoshao_lianying', 'lianying_zengshang'])
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 10000, generalExp: null, result: 'unknown' })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 10000, generalExp: 600, result: 'unknown' })
    expect(model.sides[0].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 1000, lost: 600, survived: 400 })
    expect(model.sides[1].units.find((unit) => unit.key === 'wuInfantry')).toMatchObject({ dispatched: 1000, lost: 500, survived: 500 })
    expect(model.sides[0].traits).toEqual([])
    expect(model.sides[1].traits).toEqual([{
      key: 'lianying_zengshang-0', name: '连营增伤', phase: '战斗结算后',
      detailText: '设计效果比例：10%；目标兵种追加损失：蜀步兵 +100',
    }])
    expect(model.sides.flatMap((side) => side.traits).some((trait) => trait.name === '火烧联营')).toBe(false)
  })

  it('龙胆救援展示战前麒麟卫双防与掠夺资源保护', () => {
    const source = report()
    source.detail!.secondarySide!.generals = [{ id: 'zhaoyun', name: '赵云', level: 1 }]
    source.detail!.traits = [{
      traitId: 'longdan_jiuyuan', traitName: '龙胆救援', ownerSide: 'secondary', generalId: 'zhaoyun',
      detail: { defenseBonusRate: 0.25, infantryDefenseModifiedUnits: { qilinGuard: 20 }, cavalryDefenseModifiedUnits: { qilinGuard: 15 }, plunderProtectionContributionRate: 0.2, protectedResources: { wood: 200 } },
    }]
    const model = toOfficialBattleReport(source)
    expect(model.sides[0].traits).toEqual([])
    expect(model.sides[1].traits[0]).toMatchObject({
      name: '龙胆救援', phase: '防守/增援战斗前及掠夺结算',
      detailText: '设计防御加成：25%；实际步防修正：麒麟卫 +20；实际骑防修正：麒麟卫 +15；实际保护资源：木材 +200；本次资源保护：20%',
    })

    const npcSource = report()
    npcSource.detail!.sourceType = 'npc_city'
    npcSource.detail!.primarySide.generals = [{ id: 'zhaoyun', name: '赵云', level: 1 }]
    npcSource.detail!.traits = []
    const npcModel = toOfficialBattleReport(npcSource)
    expect(npcModel.sides[0].traits).toEqual([])
    expect(npcModel.sides[1].traits).toEqual([])
  })

  it('NPC 刘备只在时间线展示仁主守护真实复活兵力', () => {
    const source = report()
    source.detail!.sourceType = 'npc_city'
    source.detail!.primarySide.generals = [{ id: 'liubei', name: '刘备', level: 1 }]
    source.detail!.traits = [
      {
        traitId: 'renzhu_shouhu', traitName: '仁主守护', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'liubei',
        detail: { revivedUnits: { greedyWolf: 35 }, totalRevived: 35, effectRate: 0.35, triggerChance: 0.6 },
      },
    ]
    const model = toOfficialBattleReport(source)
    expect(model.sides[0].traits).toEqual([
      { key: 'renzhu_shouhu-0', name: '仁主守护', phase: '进攻/防守/增援战斗结束后', detailText: '设计效果比例：35%；复活兵力：贪狼营 +35；复活总数：35；触发概率：60%' },
    ])
    expect(model.sides[1].traits).toEqual([])
  })

  it('后端正式特性结果字段全部转换为玩家可读中文', () => {
    const source = report()
    source.detail!.primarySide.generals = [{ id: 'field_contract_general', name: '字段契约将领', level: 1 }]
    source.detail!.primarySide.units = [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 100, dispatched: 100, lost: 10, survived: 90 }]
    const runtimeDetail = {
      triggerCount: 2, triggerChance: 1, effectRate: 0.1, captureRate: 0.2, captureMax: 1000, maxAffectedRate: 0.35,
      attackBonusRate: 0.1, unitAttackFlat: 50, attackReductionRate: 0.1, enemyDefenseReductionRate: 0.2,
      defenseBonusRate: 0.5, generalDefenseFlat: 20, lossReductionRate: 0.2, maxReviveCount: 10000, maxReturnCount: 10000,
      plunderBonusRate: -0.2, damagePercent: 0.25, preBattleAffected: { wuInfantry: 10 }, suppressedUnits: { wuInfantry: 10 },
      capturedUnits: { wuInfantry: 10 }, capturedToGarrison: { wuInfantry: 10 }, totalCaptured: 10, modifiedUnits: { wuInfantry: 1 },
      attackModifiedUnits: { wuInfantry: 1 }, infantryDefenseModifiedUnits: { wuInfantry: 1 }, cavalryDefenseModifiedUnits: { wuInfantry: 1 },
      extraLosses: { wuInfantry: 10 }, targetExtraLosses: { wuInfantry: 10 }, extraDamage: 10, reducedLosses: { wuInfantry: 10 },
      revivedUnits: { wuInfantry: 10 }, totalRevived: 10, returnedUnits: { wuInfantry: 10 }, disabledTraits: { disabledTraitCount: 1 },
      disableTraitCount: 1, disabledTraitCount: 1, totalSuppressed: 10, plunderDelta: { wood: -10 },
    }
    source.detail!.traits = [{
      traitId: 'field_contract', traitName: '字段契约', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'field_contract_general', detail: runtimeDetail,
    }]
    const text = toOfficialBattleReport(source).sides[0].traits[0].detailText
    for (const label of [
      '触发场次', '触发概率', '设计效果比例', '俘虏比例', '设计单兵种俘虏上限', '设计最大影响比例',
      '设计攻击加成', '设计单位攻击增加', '设计攻击降低', '设计敌方防御降低', '设计防御加成', '设计全军防御增加',
      '设计减损比例', '设计复活上限', '设计返还上限', '设计掠夺修正', '设计伤害比例', '战前真实伤亡',
      '本场压制兵力', '俘虏归队', '俘虏驻防', '俘虏总数', '实际攻防修正', '实际攻击修正', '实际步防修正',
      '实际骑防修正', '追加损失', '目标兵种追加损失', '额外伤害', '减少损失', '复活兵力', '复活总数',
      '返还兵力', '压制特性', '设计压制特性数', '实际压制特性数', '压制总数', '掠夺资源修正',
    ]) expect(text).toContain(`${label}：`)
    for (const key of Object.keys(runtimeDetail)) expect(text).not.toContain(key)
  })

  it('独立援军战报保留原始损失、最终存活和仁主复活明细', () => {
    const source = report()
    source.detail!.viewType = 'reinforcement'
    source.detail!.winnerSide = 'attacker'
    source.detail!.primarySide = {
      role: 'reinforcement', playerId: 'p3', cityName: '刘备援军', faction: 'shu', power: 1000,
      generals: [{ id: 'liubei', name: '刘备', level: 18 }],
      units: [{ unitType: 'greedyWolf', unitName: '贪狼营', amountBefore: 100, dispatched: 100, lost: 100, survived: 35 }],
    }
    source.detail!.secondarySide = null
    source.detail!.traits = [
      { traitId: 'renzhu_shouhu', traitName: '仁主守护', ownerSide: 'reinforcement', generalId: 'liubei', detail: { effectRate: 0.35, revivedUnits: { greedyWolf: 35 }, triggerChance: 0.6 } },
    ]
    source.pvpReinforcements = []
    source.pvpReinforcementLosses = {}

    const model = toOfficialBattleReport(source)
    expect(model.sides).toHaveLength(1)
    expect(model.sides[0]).toMatchObject({ role: 'reinforcement', result: 'defeat' })
    expect(model.sides[0].units.find((unit) => unit.key === 'greedyWolf')).toMatchObject({ dispatched: 100, lost: 100, survived: 35 })
    expect(model.sides[0].traits.map((trait) => trait.detailText)).toEqual([
      '设计效果比例：35%；复活兵力：贪狼营 +35；触发概率：60%',
    ])
  })

  it('NPC 郭嘉复活战报展示真实阵亡和实际复活', () => {
    for (const [traitId, traitName, generalId, generalName, amount] of [['guicai_yice', '鬼才遗策', 'guojia', '郭嘉', 22]] as const) {
      const source = report()
      source.detail!.sourceType = 'npc_city'
      source.detail!.primarySide.generals = [{ id: generalId, name: generalName, level: 1 }]
      const rate = amount / 100
      source.detail!.traits = [{ traitId, traitName, ownerSide: 'primary', ownerRole: 'attacker', generalId, detail: { effectRate: rate, actualLostUnits: { huWei: 100 }, revivedUnits: { huWei: amount }, totalRevived: amount } }]
      const model = toOfficialBattleReport(source)
      expect(model.sides[0].traits[0]).toMatchObject({
        name: traitName, phase: '进攻/防守/增援战斗结束后', detailText: `设计效果比例：${amount}%；本场真实阵亡：虎卫 +100；复活兵力：虎卫 +${amount}；复活总数：${amount}`,
      })
      expect(model.sides[1].traits).toEqual([])
    }
  })

  it('五个正式战前伤亡与临时压制特性逐项展示实际 NPC 影响值', () => {
    for (const [traitId, traitName, generalId, generalName] of [
      ['yibing_touxi', '疑兵偷袭', 'simayi', '司马懿'],
      ['shuiyan_qijun', '水淹七军', 'guanyu', '关羽'],
    ]) {
      const source = report()
      source.detail!.primarySide.generals = [{ id: generalId, name: generalName, level: 1 }]
      source.detail!.traits = [{
        traitId, traitName, ownerSide: 'primary', generalId,
        detail: { effectRate: 0.35, maxAffectedRate: 0.35, preBattleAffected: { greedyWolf: 350 }, triggerChance: 1 },
      }]
      const model = toOfficialBattleReport(source)
      expect(model.sides[0].traits[0]).toMatchObject({
        name: traitName, phase: '战斗前', detailText: '设计效果比例：35%；设计最大影响比例：35%；战前真实伤亡：贪狼营 +350；触发概率：100%',
      })
    }

    for (const [traitId, traitName, generalId, generalName, rate, affected] of [
      ['zhenhe_quanjun', '万人怒吼', 'zhangfei', '张飞', 0.5, 50],
      ['qimen_dunjia', '奇门遁甲', 'zhugeliang', '诸葛亮', 0.25, 25],
    ] as const) {
      const source = report()
      source.detail!.sourceType = 'npc_city'
      source.detail!.primarySide.generals = [{ id: generalId, name: generalName, level: 1 }]
      const detail = traitId === 'qimen_dunjia'
        ? { effectRate: rate, suppressedUnits: { greedyWolf: affected }, triggerChance: 1 }
        : { effectRate: rate, maxAffectedRate: rate, suppressedUnits: { greedyWolf: affected }, triggerChance: 1 }
      source.detail!.traits = [{
        traitId, traitName, ownerSide: 'primary', ownerRole: 'attacker', generalId,
        detail,
      }]
      const model = toOfficialBattleReport(source)
      expect(model.sides[0].traits[0]).toMatchObject({
        name: traitName,
        phase: traitId === 'qimen_dunjia' ? '进攻/防守/增援战斗前' : '战斗前',
        detailText: traitId === 'qimen_dunjia'
          ? `设计效果比例：${rate * 100}%；本场压制兵力：贪狼营 +${affected}；触发概率：100%`
          : `设计效果比例：${rate * 100}%；设计最大影响比例：${rate * 100}%；本场压制兵力：贪狼营 +${affected}；触发概率：100%`,
      })
      expect(model.sides[1].traits).toEqual([])
    }

    const zhangLiao = report()
    zhangLiao.detail!.sourceType = 'npc_city'
    zhangLiao.detail!.primarySide.generals = [{ id: 'zhangliao', name: '张辽', level: 1 }]
    zhangLiao.detail!.traits = [{
      traitId: 'weizhen_zhenhe', traitName: '震慑全军', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'zhangliao',
      detail: { effectRate: 0.25, suppressedUnits: { greedyWolf: 25 }, fledUnits: { greedyWolf: 25 }, returnedUnits: { greedyWolf: 25 }, triggerChance: 1 },
    }]
    expect(toOfficialBattleReport(zhangLiao).sides[0].traits[0]).toMatchObject({
      name: '震慑全军', phase: '主动进攻战斗前',
      detailText: '设计效果比例：25%；本场溃逃兵力：贪狼营 +25；战后返回兵力：贪狼营 +25；触发概率：100%',
    })
  })

  it('将领的战斗、非战斗和旧残留特性快照都不会伪造成真实触发', () => {
    const source = report()
    source.detail!.primarySide.generals = [
      {
        id: 'caocao', name: '曹操', level: 1,
        traits: [
          { traitId: 'weiwu_haoling', name: '魏武号令' },
          { traitId: 'weiwu_tongyu', name: '魏武统御' },
          { traitId: 'huogong', name: '旧快照残留火攻' },
        ],
      },
      {
        id: 'xunyu', name: '荀彧', level: 1,
        traits: [
          { traitId: 'wangzuo_zhicai', name: '王佐之才' },
          { traitId: 'neizheng_jingying', name: '内政精营' },
        ],
      },
      {
        id: 'machao', name: '马超', level: 1,
        traits: [{ traitId: 'tianshen_xiafan', name: '天神下凡' }],
      },
    ]
    source.detail!.traits = []
    const model = toOfficialBattleReport(source)
    expect(model.sides[0].traits).toEqual([])
  })

  it('吕蒙双行军特性留在拥有快照但不进入战斗时间线', () => {
    const source = report()
    source.detail!.primarySide.generals = [{
      id: 'lvmeng', name: '吕蒙', level: 1,
      traits: [{ traitId: 'baiyi_dujiang', name: '白衣渡江' }, { traitId: 'baiyi_jixing', name: '白衣急行' }],
    }]
    source.detail!.traits = []

    const model = toOfficialBattleReport(source)
    expect(model.sides[0]).toMatchObject({ general: { id: 'lvmeng', name: '吕蒙' }, traitText: '', traits: [] })
    expect(model.sides.flatMap((side) => side.traits)).toEqual([])
  })

  it('白衣渡江合法未命中时只展示白衣急行后的基础战斗结果', () => {
    const source = report()
    source.detail!.primarySide.faction = 'wu'
    source.detail!.primarySide.power = 1000
    source.detail!.primarySide.generals = [{
      id: 'lvmeng', name: '吕蒙', level: 1,
      traits: [{ traitId: 'baiyi_dujiang', name: '白衣渡江' }, { traitId: 'baiyi_jixing', name: '白衣急行' }],
    }]
    source.detail!.primarySide.units = [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 100, dispatched: 100, lost: 0, survived: 100 }]
    source.detail!.secondarySide!.faction = 'wei'
    source.detail!.secondarySide!.power = 10
    source.detail!.secondarySide!.generals = [{ id: 'caocao', name: '曹操', level: 1 }]
    source.detail!.secondarySide!.units = [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1, dispatched: 1, lost: 1, survived: 0 }]
    source.detail!.rewards.generalExp = 1
    source.detail!.traits = []

    const model = toOfficialBattleReport(source)
    expect(source.detail!.primarySide.generals![0]!.traits?.map((trait) => trait.traitId)).toEqual(['baiyi_dujiang', 'baiyi_jixing'])
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 1000, general: { id: 'lvmeng', name: '吕蒙', level: 1 }, generalExp: 1, traits: [], traitText: '' })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 10, traits: [], traitText: '' })
    expect(model.sides[0].units.find((unit) => unit.key === 'wuInfantry')).toMatchObject({ dispatched: 100, lost: 0, survived: 100 })
    expect(model.sides[1].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 1, lost: 1, survived: 0 })
    expect(model.sides.flatMap((side) => side.traits)).toEqual([])
  })

  it('吕蒙作为援军时双行军快照不产生援军战斗效果', () => {
    const source = report()
    source.pvpReinforcements = [{
      reinforcementId: 'rein_lvmeng', fromPlayerId: 'player_lvmeng', fromPlayerName: '吕蒙援军', faction: 'wu',
      troops: { wuInfantry: 100 }, generalExpGained: 100, generalLevelBefore: 1, generalLevelAfter: 2,
      generals: [{
        id: 'lvmeng', name: '吕蒙', level: 1,
        traits: [{ traitId: 'baiyi_dujiang', name: '白衣渡江' }, { traitId: 'baiyi_jixing', name: '白衣急行' }],
      }],
    }]
    source.pvpReinforcementLosses = { rein_lvmeng: { wuInfantry: 98 } }
    source.detail!.traits = []

    const model = toOfficialBattleReport(source)
    expect(model.sides[2]).toMatchObject({
      role: 'reinforcement', general: { id: 'lvmeng', name: '吕蒙' },
      generalExp: 100, generalLevelBefore: 1, generalLevelAfter: 2,
      traitText: '', traits: [],
    })
    expect(model.sides[2].units.find((unit) => unit.key === 'wuInfantry')).toMatchObject({ dispatched: 100, lost: 98, survived: 2 })
  })

  it('太史慈主动进攻只保留拥有快照且信义勇烈不进入触发时间线', () => {
    const source = report()
    source.detail!.primarySide.generals = [{
      id: 'taishici', name: '太史慈', level: 1,
      traits: [{ traitId: 'kuairu_shandian', name: '快如闪电' }, { traitId: 'xinyi_yonglie', name: '信义勇烈' }],
    }]
    source.detail!.traits = []

    const model = toOfficialBattleReport(source)
    expect(model.sides[0]).toMatchObject({ general: { id: 'taishici', name: '太史慈' }, traitText: '', traits: [] })
    expect(model.sides.flatMap((side) => side.traits)).toEqual([])
  })

  it('信义勇烈收到援军效果时只标记为援军战斗前', () => {
    const source = report()
    source.pvpReinforcements = [{
      reinforcementId: 'rein_taishici', fromPlayerId: 'player_taishici', fromPlayerName: '太史慈援军', faction: 'wu',
      troops: { shadowGuard: 100 }, generals: [{ id: 'taishici', name: '太史慈', level: 1 }],
    }]
    source.detail!.traits = [{
      traitId: 'xinyi_yonglie', traitName: '信义勇烈', ownerSide: 'reinforcement', generalId: 'taishici',
      detail: { defenseBonusRate: 0.1, infantryDefenseModifiedUnits: { shadowGuard: 1 }, cavalryDefenseModifiedUnits: { shadowGuard: 1 } },
    }]

    const model = toOfficialBattleReport(source)
    expect(model.sides[2].traits[0]).toMatchObject({
      name: '信义勇烈', phase: '援军战斗前',
      detailText: '设计防御加成：10%；实际步防修正：影卫 +1；实际骑防修正：影卫 +1',
    })
    expect(model.sides[0].traits).toEqual([])
  })

  it('快如闪电合法未命中时不进入时间线且信义勇烈保留真实援军结果', () => {
    const source = report()
    source.detail!.primarySide.faction = 'wei'
    source.detail!.primarySide.power = 1100
    source.detail!.primarySide.units = [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 110, dispatched: 110, lost: 110, survived: 0 }]
    source.detail!.secondarySide!.faction = 'shu'
    source.detail!.secondarySide!.power = 1110
    source.detail!.secondarySide!.units = [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1, dispatched: 1, lost: 1, survived: 0 }]
    source.pvpReinforcements = [{
      reinforcementId: 'rein_taishici', fromPlayerId: 'player_taishici', fromPlayerName: '太史慈援军', faction: 'wu',
      troops: { wuInfantry: 100 }, generalExpGained: 110, generalLevelBefore: 1, generalLevelAfter: 1,
      generals: [{
        id: 'taishici', name: '太史慈', level: 1,
        traits: [{ traitId: 'kuairu_shandian', name: '快如闪电' }, { traitId: 'xinyi_yonglie', name: '信义勇烈' }],
      }],
    }]
    source.pvpReinforcementLosses = { rein_taishici: { wuInfantry: 98 } }
    source.detail!.traits = [{
      traitId: 'xinyi_yonglie', traitName: '信义勇烈', ownerSide: 'reinforcement', ownerRole: 'reinforcement',
      ownerPlayerId: 'player_taishici', generalId: 'taishici',
      detail: { defenseBonusRate: 0.1, infantryDefenseModifiedUnits: { wuInfantry: 1 }, cavalryDefenseModifiedUnits: { wuInfantry: 1 } },
    }]

    const model = toOfficialBattleReport(source)
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 1100, traits: [], traitText: '' })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 1110, traits: [], traitText: '' })
    expect(model.sides[2]).toMatchObject({
      role: 'reinforcement', general: { id: 'taishici', name: '太史慈' },
      generalExp: 110, generalLevelBefore: 1, generalLevelAfter: 1,
    })
    expect(model.sides[2].general?.traits?.map((trait) => trait.traitId)).toEqual(['kuairu_shandian', 'xinyi_yonglie'])
    expect(model.sides[2].units.find((unit) => unit.key === 'wuInfantry')).toMatchObject({ dispatched: 100, lost: 98, survived: 2 })
    expect(model.sides[2].traits).toEqual([expect.objectContaining({
      name: '信义勇烈', phase: '援军战斗前',
      detailText: '设计防御加成：10%；实际步防修正：吴步兵 +1；实际骑防修正：吴步兵 +1',
    })])
    expect(model.sides.flatMap((side) => side.traits).map((trait) => trait.name)).toEqual(['信义勇烈'])
  })

  it('两名玩家的援军对敌特性分别展示真实扣兵和压制数', () => {
    const source = report()
    source.pvpReinforcements = [
      {
        reinforcementId: 'rein_huangzhong', fromPlayerId: 'player_huangzhong', fromPlayerName: '黄忠援军', faction: 'shu',
        troops: { greedyWolf: 100 }, generals: [{ id: 'huangzhong', name: '黄忠', level: 1 }],
      },
      {
        reinforcementId: 'rein_zhugeliang', fromPlayerId: 'player_zhugeliang', fromPlayerName: '诸葛亮援军', faction: 'shu',
        troops: { azureDragon: 100 }, generals: [{ id: 'zhugeliang', name: '诸葛亮', level: 1 }],
      },
    ]
    source.pvpReinforcementLosses = {
      rein_huangzhong: { greedyWolf: 10 }, rein_zhugeliang: { azureDragon: 10 },
    }
    source.detail!.traits = [
      {
        traitId: 'laodang_yizhuang', traitName: '老当益壮', ownerSide: 'reinforcement', ownerRole: 'reinforcement',
        ownerPlayerId: 'player_huangzhong', generalId: 'huangzhong',
        detail: { effectRate: 0.1, extraLosses: { huWei: 11 }, triggerChance: 1 },
      },
      {
        traitId: 'wolong_mouzhi', traitName: '卧龙奇谋', ownerSide: 'reinforcement', ownerRole: 'reinforcement',
        ownerPlayerId: 'player_zhugeliang', generalId: 'zhugeliang',
        detail: { disabledGeneralCount: 1, disabledTraitCount: 2, triggerChance: 1 },
      },
    ]

    const model = toOfficialBattleReport(source)
    const huangzhong = model.sides.find((side) => side.general?.id === 'huangzhong')
    const zhugeliang = model.sides.find((side) => side.general?.id === 'zhugeliang')
    expect(huangzhong?.traits).toEqual([expect.objectContaining({
      name: '老当益壮', phase: '战斗结算后', detailText: '设计效果比例：10%；追加损失：虎卫 +11；触发概率：100%',
    })])
    expect(zhugeliang?.traits).toEqual([expect.objectContaining({
      name: '卧龙奇谋', phase: '进攻/防守/增援战斗前', detailText: '封禁将领数：1；实际压制特性数：2；触发概率：100%',
    })])
    expect(model.sides.slice(0, 2).flatMap((side) => side.traits)).toEqual([])
  })

  it('未携带将领的战报不展示武将、经验或特性触发', () => {
    const source = report()
    source.generalExpGained = 0
    source.detail!.primarySide.generals = []
    source.detail!.rewards.generalExp = 0
    source.detail!.rewards.generalLevelBefore = undefined
    source.detail!.rewards.generalLevelAfter = undefined
    source.detail!.traits = []
    const model = toOfficialBattleReport(source)
    expect(model.sides[0]).toMatchObject({ general: null, generalExp: 0, generalLevelBefore: null, generalLevelAfter: null, traits: [] })
  })

  it('守城主将离城时防守方不展示将领、经验或特性触发', () => {
    const source = report()
    source.detail!.secondarySide!.generals = []
    source.detail!.traits = []
    const model = toOfficialBattleReport(source)
    expect(model.sides[1]).toMatchObject({ role: 'defender', general: null, generalExp: null, traits: [] })
  })

  it('守城主将归城后按后端时间线恢复将领和防御特性展示', () => {
    const source = report()
    source.detail!.secondarySide!.generals = [{ id: 'sunquan', name: '孙权', level: 1 }]
    source.detail!.traits = [{
      traitId: 'jiangdong_gushou', traitName: '江东固守', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'sunquan',
      detail: { defenseBonusRate: 0.5, infantryDefenseModifiedUnits: { greedyWolf: 5 }, cavalryDefenseModifiedUnits: { greedyWolf: 4 }, triggerChance: 1 },
    }]
    const model = toOfficialBattleReport(source)
    expect(model.sides[1].general).toMatchObject({ id: 'sunquan', name: '孙权' })
    expect(model.sides[1].traits[0]).toMatchObject({
      name: '江东固守', phase: '防守/增援战斗前',
      detailText: '设计防御加成：50%；实际步防修正：贪狼营 +5；实际骑防修正：贪狼营 +4；触发概率：100%',
    })
  })

  it('张飞与诸葛亮临时压制统一显示战斗前和本场压制兵力', () => {
    for (const [traitId, rate] of [['zhenhe_quanjun', 0.5], ['qimen_dunjia', 0.25]] as const) {
      const source = report()
      source.detail!.primarySide.generals = [{ id: traitId, name: traitId, level: 1 }]
      const detail = traitId === 'qimen_dunjia'
        ? { effectRate: rate, suppressedUnits: { greedyWolf: 250 }, triggerChance: 1 }
        : { effectRate: rate, maxAffectedRate: rate, suppressedUnits: { greedyWolf: 250 }, triggerChance: 1 }
      source.detail!.traits = [{
        traitId, traitName: traitId, ownerSide: 'primary', generalId: traitId,
        detail,
      }]
      const model = toOfficialBattleReport(source)
      expect(model.sides[0].traits[0]).toMatchObject({
        phase: traitId === 'qimen_dunjia' ? '进攻/防守/增援战斗前' : '战斗前',
        detailText: traitId === 'qimen_dunjia'
          ? `设计效果比例：${rate * 100}%；本场压制兵力：贪狼营 +250；触发概率：100%`
          : `设计效果比例：${rate * 100}%；设计最大影响比例：${rate * 100}%；本场压制兵力：贪狼营 +250；触发概率：100%`,
      })
    }
  })

  it('美人心计展示当前攻击加成、实际修正和 GM 概率', () => {
	const source = report()
	source.detail!.primarySide.generals = [{ id: 'zhenmi', name: '甄宓', level: 1 }]
	source.detail!.traits = [{
	  traitId: 'meiren', traitName: '美人心计', ownerSide: 'primary', generalId: 'zhenmi',
	  detail: { attackBonusRate: 0.25, attackModifiedUnits: { huWei: 3 }, triggerChance: 0.5 },
	}]
	const model = toOfficialBattleReport(source)
	expect(model.sides[0].traits[0]).toMatchObject({
	  name: '美人心计', phase: '主动进攻战斗前',
	  detailText: '设计攻击加成：25%；实际攻击修正：虎卫 +3；触发概率：50%',
	})
  })

  it('魅惑扰阵展示当前防御削减、实际修正和 GM 概率', () => {
	const source = report()
	source.detail!.primarySide.generals = [{ id: 'zhenmi', name: '甄宓', level: 1 }]
	source.detail!.traits = [{
	  traitId: 'meihuo_raozhen', traitName: '魅惑扰阵', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'zhenmi',
	  detail: { enemyDefenseReductionRate: 0.25, infantryDefenseModifiedUnits: { greedyWolf: -2 }, cavalryDefenseModifiedUnits: { greedyWolf: -2 }, triggerChance: 0.5 },
	}]
	const model = toOfficialBattleReport(source)
	expect(model.sides[0].traits).toEqual([{
	  key: 'meihuo_raozhen-0', name: '魅惑扰阵', phase: '主动进攻战斗前',
	  detailText: '设计敌方防御降低：25%；实际步防修正：贪狼营 -2；实际骑防修正：贪狼营 -2；触发概率：50%',
	}])
  })

  it('甄宓 NPC 与 PVP 双特性对齐攻击、破防和兵损实值', () => {
    for (const sourceType of ['npc_city', 'player_city'] as const) {
      const source = report()
      source.detail!.sourceType = sourceType
      source.detail!.battleType = 'attack'
      source.detail!.primarySide.generals = [{ id: 'zhenmi', name: '甄宓', level: 1 }]
      source.detail!.secondarySide!.units = [{ unitType: 'greedyWolf', unitName: '贪狼营', amountBefore: 1000, dispatched: 1000, lost: 800, survived: 0 }]
      source.detail!.traits = [
        {
		  traitId: 'meiren', traitName: '美人心计', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'zhenmi',
		  detail: { attackBonusRate: 0.25, attackModifiedUnits: { huWei: 3 }, triggerChance: 0.5 },
        },
        {
          traitId: 'meihuo_raozhen', traitName: '魅惑扰阵', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'zhenmi',
		  detail: { enemyDefenseReductionRate: 0.25, infantryDefenseModifiedUnits: { greedyWolf: -2 }, cavalryDefenseModifiedUnits: { greedyWolf: -2 }, triggerChance: 0.5 },
        },
      ]
      const model = toOfficialBattleReport(source)
      expect(model.sides[1].units.find((unit) => unit.key === 'greedyWolf')).toMatchObject({ dispatched: 1000, lost: 800, survived: 0 })
      expect(model.sides[0].traits).toEqual([
        {
		  key: 'meiren-0', name: '美人心计', phase: '主动进攻战斗前',
		  detailText: '设计攻击加成：25%；实际攻击修正：虎卫 +3；触发概率：50%',
        },
        {
          key: 'meihuo_raozhen-1', name: '魅惑扰阵', phase: '主动进攻战斗前',
		  detailText: '设计敌方防御降低：25%；实际步防修正：贪狼营 -2；实际骑防修正：贪狼营 -2；触发概率：50%',
        },
      ])
      expect(model.sides[1].traits).toEqual([])
    }
  })

  it('NPC 火攻按主动进攻结算阶段展示后端实际追加损失', () => {
    const source = report()
    source.detail!.sourceType = 'npc_city'
    source.detail!.primarySide.generals = [{ id: 'zhouyu', name: '周瑜', level: 1 }]
    source.detail!.traits = [{
      traitId: 'huogong', traitName: '火烧赤壁', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'zhouyu',
      detail: { damagePercent: 0.25, extraDamage: 250, targetExtraLosses: { greedyWolf: 250 }, triggerChance: 1 },
    }]
    const model = toOfficialBattleReport(source)
    expect(model.sides[0].traits[0]).toMatchObject({
      name: '火烧赤壁', phase: '主动进攻战斗结算后',
      detailText: '目标兵种追加损失：贪狼营 +250；设计伤害比例：25%；额外伤害：250；触发概率：100%',
    })
    expect(model.sides[1].traits).toEqual([])
  })

  it('NPC 战后升级仍展示战前将领快照和实际特性时间线', () => {
    const source = report()
    source.detail!.sourceType = 'npc_city'
    source.detail!.primarySide.power = 1100
    source.detail!.primarySide.units = [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: 87, survived: 13 }]
    source.detail!.primarySide.generals = [{
      id: 'zhouyu', name: '周瑜', level: 1,
      traits: [{ traitId: 'meizhoulang_junlue', name: '美周郎军略' }, { traitId: 'huogong', name: '火烧赤壁' }],
    }]
    source.detail!.rewards = { generalExp: 100, generalLevelBefore: 1, generalLevelAfter: 2 }
    source.detail!.traits = [{
      traitId: 'meizhoulang_junlue', traitName: '美周郎军略', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'zhouyu',
      detail: { attackBonusRate: 0.05, attackModifiedUnits: { weiInfantry: 1 } },
    }]
    const model = toOfficialBattleReport(source)
    expect(model.sides[0]).toMatchObject({
      role: 'attacker', power: 1100, general: { id: 'zhouyu', name: '周瑜', level: 1 },
      generalExp: 100, generalLevelBefore: 1, generalLevelAfter: 2,
    })
    expect(model.sides[0].traits).toEqual([
      expect.objectContaining({ name: '美周郎军略', phase: '主动进攻战斗前', detailText: '设计攻击加成：5%；实际攻击修正：魏步兵 +1' }),
    ])
    expect(model.sides[0].traits.some((trait) => trait.name === '火烧赤壁')).toBe(false)
  })

  it('NPC 双城扫荡使用初始将领快照并单独展示累计升级', () => {
    const source = report()
    source.battleType = 'sweep'
    source.detail!.sourceType = 'npc_city'
    source.detail!.battleType = 'sweep'
    source.detail!.primarySide.power = 1000
    source.detail!.primarySide.generals = [{ id: 'zhouyu', name: '周瑜', level: 1 }]
    source.detail!.rewards = { generalExp: 2, generalLevelBefore: 1, generalLevelAfter: 3 }
    source.detail!.traits = []
    const model = toOfficialBattleReport(source)
    expect(model.sides[0]).toMatchObject({
      role: 'attacker', power: 1000, general: { id: 'zhouyu', name: '周瑜', level: 1 },
      generalExp: 2, generalLevelBefore: 1, generalLevelAfter: 3,
    })
    expect(formatGeneralProgress(model.sides[0].generalExp, model.sides[0].generalLevelBefore, model.sides[0].generalLevelAfter)).toBe('+2 · Lv.1 → Lv.3')
  })

  it('官方战报适配只读后端扩展、援军和特性快照', () => {
    const source = report()
    source.detail!.extra = {
      pvp: { reinforcementLosses: { rr1: { shadowGuard: 5 } }, wall: { level: 10 } },
      sweep: { defenders: [{ targetId: 'npc_1', resources: { wood: 5 } }] },
    }
    const before = JSON.stringify(source)
    const model = toOfficialBattleReport(source)

    expect(JSON.stringify(source)).toBe(before)
    expect(model.sides.map((side) => side.role)).toEqual(['attacker', 'defender', 'reinforcement'])
    expect(model.sides[2].units.find((unit) => unit.key === 'shadowGuard')).toMatchObject({ dispatched: 30, lost: 5, survived: 25 })
  })

  it('周瑜双特性按后端顺序展示战前加攻和战后火攻', () => {
    const source = report()
    source.detail!.primarySide.generals = [{ id: 'zhouyu', name: '周瑜', level: 1 }]
    source.detail!.traits = [
      {
        traitId: 'meizhoulang_junlue', traitName: '美周郎军略', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'zhouyu',
        detail: { attackBonusRate: 0.05, attackModifiedUnits: { shadowGuard: 1 } },
      },
      {
        traitId: 'huogong', traitName: '火烧赤壁', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'zhouyu',
        detail: { damagePercent: 0.25, extraDamage: 250, targetExtraLosses: { greedyWolf: 250 }, triggerChance: 1 },
      },
    ]
    const model = toOfficialBattleReport(source)
    expect(model.sides[0].traits).toHaveLength(2)
    expect(model.sides[0].traits[0]).toMatchObject({
      name: '美周郎军略', phase: '主动进攻战斗前',
      detailText: '设计攻击加成：5%；实际攻击修正：影卫 +1',
    })
    expect(model.sides[0].traits[1]).toMatchObject({
      name: '火烧赤壁', phase: '主动进攻战斗结算后',
      detailText: '目标兵种追加损失：贪狼营 +250；设计伤害比例：25%；额外伤害：250；触发概率：100%',
    })
    expect(model.sides[1].traits).toEqual([])
  })

  it('孙策霸王铁骑展示真实兵种和实际攻击修正', () => {
    const source = report()
    source.detail!.primarySide.faction = 'wu'
    source.detail!.primarySide.generals = [{ id: 'sunce', name: '孙策', level: 1 }]
    source.detail!.primarySide.units = [{ unitType: 'overlordRider', unitName: '霸王骑', amountBefore: 100, dispatched: 100, lost: 10, survived: 90 }]
    source.detail!.traits = [{ traitId: 'xiaobawang_tieqi', traitName: '小霸王', ownerSide: 'primary', generalId: 'sunce', detail: { attackModifiedUnits: { overlordRider: 50 } } }]
    const model = toOfficialBattleReport(source)
    expect(model.sides[0].traits[0]).toMatchObject({ phase: '主动进攻战斗前', detailText: '实际攻击修正：霸王骑 +50' })
  })

  it('魏武统御仅展示守城或援军全军 15% 防御实际修正', () => {
    const source = report()
    source.detail!.secondarySide!.generals = [{ id: 'caocao', name: '曹操', level: 1 }]
    source.detail!.traits = [{
      traitId: 'weiwu_tongyu', traitName: '魏武统御', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'caocao',
      detail: {
        defenseBonusRate: 0.15,
        infantryDefenseModifiedUnits: { huWei: 1, qingZhouArmy: 1 },
        cavalryDefenseModifiedUnits: { huWei: 1, qingZhouArmy: 2 },
      },
    }]
    const model = toOfficialBattleReport(source)
    expect(model.sides[1].traits[0]).toMatchObject({
      phase: '守城/增援战斗前',
      detailText: '设计防御加成：15%；实际步防修正：虎卫 +1、青州军 +1；实际骑防修正：虎卫 +1、青州军 +2',
    })
    expect(model.sides[0].traits).toEqual([])

    const reinforcementSource = report()
    reinforcementSource.pvpReinforcements![0].generals = [{ id: 'caocao', name: '曹操', level: 1 }]
    reinforcementSource.detail!.traits = [{
      traitId: 'weiwu_tongyu', traitName: '魏武统御', ownerSide: 'reinforcement', ownerPlayerId: 'p3', generalId: 'caocao',
      detail: { defenseBonusRate: 0.15, infantryDefenseModifiedUnits: { huWei: 2 }, cavalryDefenseModifiedUnits: { huWei: 1 } },
    }]
    const reinforcementModel = toOfficialBattleReport(reinforcementSource)
    expect(reinforcementModel.sides[2].traits[0]).toMatchObject({
      name: '魏武统御', phase: '守城/增援战斗前',
      detailText: '设计防御加成：15%；实际步防修正：虎卫 +2；实际骑防修正：虎卫 +1',
    })
  })

  it('六项纯攻击加成特性统一标记为主动进攻战斗前', () => {
    const cases = [
      ['sizhandaodi', 'huWei', '虎卫', 0.35, 0, 4],
      ['weizhen_xiaoyao', 'huBaoQi', '虎豹骑', 0.35, 0, 4],
      ['wusheng_pojun', 'huWei', '虎卫', 0.2, 0, 2],
      ['wanren_nuhou', 'huWei', '虎卫', 0.2, 0, 2],
      ['xiaobawang_tieqi', 'overlordRider', '霸王骑', 0, 50, 50],
      ['meizhoulang_junlue', 'shadowGuard', '影卫', 0.05, 0, 1],
    ] as const
    for (const [traitId, unitType, unitName, attackBonusRate, unitAttackFlat, attackDelta] of cases) {
      const source = report()
      source.detail!.primarySide.generals = [{ id: traitId, name: traitId, level: 1 }]
      const designDetail = attackBonusRate > 0 ? { attackBonusRate } : { unitAttackFlat }
      source.detail!.traits = [{ traitId, traitName: traitId, ownerSide: 'primary', generalId: traitId, detail: { ...designDetail, attackModifiedUnits: { [unitType]: attackDelta }, triggerChance: 1 } }]
      const model = toOfficialBattleReport(source)
      const designText = attackBonusRate > 0 ? `设计攻击加成：${attackBonusRate * 100}%` : `设计单位攻击增加：${unitAttackFlat}`
      expect(model.sides[0].traits[0]).toMatchObject({
        phase: '主动进攻战斗前', detailText: `${designText}；实际攻击修正：${unitName} +${attackDelta}；触发概率：100%`,
      })
    }
  })

  it('NPC 单城战报按后端结果展示主动进攻与掠夺攻击加成', () => {
    const cases = [
      ['sizhandaodi', '死战到底', { attackBonusRate: 0.35, attackModifiedUnits: { huWei: 5 }, triggerChance: 1 }, '主动进攻战斗前'],
      ['jinfan_qixi', '锦帆奇袭', { attackBonusRate: 0.1, attackModifiedUnits: { shadowGuard: 1 }, triggerChance: 1 }, '掠夺战战斗前'],
    ] as const
    for (const [traitId, traitName, detailData, phase] of cases) {
      const source = report()
      source.detail!.sourceType = 'npc_city'
      source.detail!.battleType = traitId === 'jinfan_qixi' ? 'plunder' : 'attack'
      source.detail!.traits = [{ traitId, traitName, ownerSide: 'primary', ownerRole: 'attacker', generalId: 'g1', detail: detailData }]
      const model = toOfficialBattleReport(source)
      expect(model.sides[0].traits[0]).toMatchObject({ name: traitName, phase })
      expect(model.sides[1].traits).toHaveLength(0)
    }
  })

  it('分类伤害特性展示命中的真实兵种和追加损失', () => {
    const source = report()
    source.detail!.sourceType = 'npc_city'
    source.detail!.primarySide.generals = [{ id: 'machao', name: '马超', level: 1 }]
    source.detail!.secondarySide!.faction = 'wei'
    source.detail!.secondarySide!.units = [{ unitType: 'weiCavalry', unitName: '魏骑兵', amountBefore: 500, dispatched: 500, lost: 277, survived: 223 }]
    source.detail!.traits = [{ traitId: 'xiliang_tuji', traitName: '西凉突击', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'machao', detail: { effectRate: 0.12, targetExtraLosses: { weiCavalry: 60 }, triggerChance: 1 } }]
    const model = toOfficialBattleReport(source)
    expect(model.sides[0].traits[0]).toMatchObject({ phase: '战斗结算后', detailText: '设计效果比例：12%；目标兵种追加损失：魏骑兵 +60；触发概率：100%' })

    const defenseSource = report()
    defenseSource.detail!.primarySide.faction = 'wu'
    defenseSource.detail!.primarySide.units = [{ unitType: 'wuCavalry', unitName: '吴骑兵', amountBefore: 1000, dispatched: 1000, lost: 220, survived: 780 }]
    defenseSource.detail!.secondarySide!.generals = [{ id: 'machao', name: '马超', level: 1 }]
    defenseSource.detail!.traits = [{ traitId: 'xiliang_tuji', traitName: '西凉突击', ownerSide: 'secondary', generalId: 'machao', detail: { effectRate: 0.2, targetExtraLosses: { wuCavalry: 200 }, triggerChance: 1 } }]
    const defenseModel = toOfficialBattleReport(defenseSource)
    expect(defenseModel.sides[0].traits).toEqual([])
    expect(defenseModel.sides[1].traits[0]).toMatchObject({ phase: '战斗结算后', detailText: '设计效果比例：20%；目标兵种追加损失：吴骑兵 +200；触发概率：100%' })
  })

  it('固守汉中分别展示全军两类实际防御修正', () => {
    const source = report()
    source.detail!.primarySide.generals = [{ id: 'weiyan', name: '魏延', level: 1 }]
    source.detail!.traits = [{ traitId: 'gushou_hanzhong', traitName: '固守汉中', ownerSide: 'primary', generalId: 'weiyan', detail: { generalDefenseFlat: 20, infantryDefenseModifiedUnits: { huWei: 20 }, cavalryDefenseModifiedUnits: { huWei: 20 }, triggerChance: 1 } }]
    const model = toOfficialBattleReport(source)
    expect(model.sides[0].traits[0]).toMatchObject({
      phase: '防守/增援战斗前',
      detailText: '设计全军防御增加：20；实际步防修正：虎卫 +20；实际骑防修正：虎卫 +20；触发概率：100%',
    })
  })

  it('盾阵防御标记为防守或增援并展示百分比换算后的实际变化', () => {
    const source = report()
    source.detail!.secondarySide!.generals = [{ id: 'xiahouyuan', name: '夏侯渊', level: 1 }]
    source.detail!.secondarySide!.units = [{ unitType: 'huWei', unitName: '虎卫', amountBefore: 100, dispatched: 100, lost: 10, survived: 90 }]
    source.detail!.traits = [{ traitId: 'dunzhen_fangyu', traitName: '盾阵防御', ownerSide: 'secondary', generalId: 'xiahouyuan', detail: { defenseBonusRate: 0.3, infantryDefenseModifiedUnits: { huWei: 3 }, cavalryDefenseModifiedUnits: { huWei: 2 }, triggerChance: 0.6 } }]
    const model = toOfficialBattleReport(source)
    expect(model.sides[1].traits[0]).toMatchObject({
      name: '盾阵防御', phase: '防守/增援战斗前',
      detailText: '设计防御加成：30%；实际步防修正：虎卫 +3；实际骑防修正：虎卫 +2；触发概率：60%',
    })
  })

  it('江东固守作为援军时展示实际防御变化和援军归属', () => {
    const source = report()
    source.detail!.viewType = 'reinforcement'
    source.detail!.primarySide.role = 'reinforcement'
    source.detail!.primarySide.generals = [{ id: 'sunquan', name: '孙权', level: 1 }]
    source.detail!.primarySide.units = [{ unitType: 'shadowGuard', unitName: '影卫', amountBefore: 100, dispatched: 100, lost: 20, survived: 80 }]
    source.detail!.secondarySide = null
    source.detail!.traits = [{
      traitId: 'jiangdong_gushou', traitName: '江东固守', ownerSide: 'reinforcement', generalId: 'sunquan',
      detail: { defenseBonusRate: 0.5, infantryDefenseModifiedUnits: { shadowGuard: 5 }, cavalryDefenseModifiedUnits: { shadowGuard: 4 }, triggerChance: 1 },
    }]
    source.pvpReinforcements = []
    source.pvpReinforcementLosses = {}
    const model = toOfficialBattleReport(source)
    expect(model.sides[0]).toMatchObject({ role: 'reinforcement' })
    expect(model.sides[0].traits[0]).toMatchObject({
      name: '江东固守', phase: '防守/增援战斗前',
      detailText: '设计防御加成：50%；实际步防修正：影卫 +5；实际骑防修正：影卫 +4；触发概率：100%',
    })
  })

  it('孙权作为援军时江东固守未命中只展示基础防御和完整损失', () => {
    const source = report()
    source.detail!.primarySide.faction = 'wei'
    source.detail!.primarySide.power = 1100
    source.detail!.primarySide.generals = [{ id: 'caocao', name: '曹操', level: 1 }]
    source.detail!.primarySide.units = [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 110, dispatched: 110, lost: 97, survived: 13 }]
    source.detail!.secondarySide!.faction = 'shu'
    source.detail!.secondarySide!.power = 1010
    source.detail!.secondarySide!.generals = [{ id: 'liubei', name: '刘备', level: 1 }]
    source.detail!.secondarySide!.units = [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1, dispatched: 1, lost: 1, survived: 0 }]
    source.detail!.traits = []
    source.pvpReinforcements = [{
      reinforcementId: 'rein_sunquan_miss', fromPlayerId: 'player_sunquan', fromPlayerName: '孙权援军', faction: 'wu',
      troops: { wuInfantry: 100 }, generalExpGained: 97,
      generals: [{ id: 'sunquan', name: '孙权', level: 1, traits: [{ traitId: 'jiangdong_gushou', name: '江东固守' }] }],
    }]
    source.pvpReinforcementLosses = { rein_sunquan_miss: { wuInfantry: 100 } }

    const model = toOfficialBattleReport(source)
    expect(source.pvpReinforcements[0]!.generals?.[0]?.traits?.map((trait) => trait.traitId)).toEqual(['jiangdong_gushou'])
    expect(model.sides.map((side) => side.role)).toEqual(['attacker', 'defender', 'reinforcement'])
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 1100 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 1010 })
    expect(model.sides[2]).toMatchObject({ role: 'reinforcement', general: { id: 'sunquan', name: '孙权', level: 1 }, generalExp: 97, traits: [], traitText: '' })
    expect(model.sides[0].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 110, lost: 97, survived: 13 })
    expect(model.sides[1].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 1, lost: 1, survived: 0 })
    expect(model.sides[2].units.find((unit) => unit.key === 'wuInfantry')).toMatchObject({ dispatched: 100, lost: 100, survived: 0 })
    expect(model.sides.flatMap((side) => side.traits)).toEqual([])
  })

  it('谋定后发标记为防守或增援战斗前并展示实际加防', () => {
    const source = report()
    source.detail!.secondarySide!.generals = [{ id: 'simayi', name: '司马懿', level: 1 }]
    source.detail!.traits = [{ traitId: 'mouding_houfa', traitName: '谋定后发', ownerSide: 'secondary', generalId: 'simayi', detail: { defenseBonusRate: 0.35, infantryDefenseModifiedUnits: { huWei: 4 }, cavalryDefenseModifiedUnits: { huWei: 3 }, triggerChance: 1 } }]
    const model = toOfficialBattleReport(source)
    expect(model.sides[1].traits[0]).toMatchObject({
      name: '谋定后发', phase: '防守/增援战斗前', detailText: '设计防御加成：35%；实际步防修正：虎卫 +4；实际骑防修正：虎卫 +3；触发概率：100%',
    })
  })

  it('司马懿守城双特性同时展示真实战前伤亡和实际加防', () => {
    const source = report()
    source.detail!.secondarySide!.generals = [{ id: 'simayi', name: '司马懿', level: 1 }]
    source.detail!.traits = [
      {
        traitId: 'yibing_touxi', traitName: '疑兵偷袭', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'simayi',
        detail: { effectRate: 0.35, preBattleAffected: { greedyWolf: 350 }, triggerChance: 1 },
      },
	      {
	        traitId: 'mouding_houfa', traitName: '谋定后发', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'simayi',
	        detail: { defenseBonusRate: 0.35, infantryDefenseModifiedUnits: { weiInfantry: 4 }, cavalryDefenseModifiedUnits: { weiInfantry: 3 }, triggerChance: 1 },
	      },
    ]
    const model = toOfficialBattleReport(source)
    expect(model.sides[0].traits).toEqual([])
    expect(model.sides[1].traits).toHaveLength(2)
    expect(model.sides[1].traits[0]).toMatchObject({
	      name: '疑兵偷袭', phase: '战斗前',
	      detailText: '设计效果比例：35%；战前真实伤亡：贪狼营 +350；触发概率：100%',
    })
    expect(model.sides[1].traits[1]).toMatchObject({
	      name: '谋定后发', phase: '防守/增援战斗前',
	      detailText: '设计防御加成：35%；实际步防修正：魏步兵 +4；实际骑防修正：魏步兵 +3；触发概率：100%',
    })
  })

  it('防守司马懿疑兵未命中时只展示谋定后发真实加防', () => {
    const source = report()
    source.viewType = 'defense'
    source.result = 'defender_victory'
    source.battleType = 'plunder'
    source.detail!.viewType = 'defense'
    source.detail!.result = 'defender_victory'
    source.detail!.winnerSide = 'defender'
    source.detail!.ownerSide = 'defender'
    source.detail!.battleType = 'plunder'
    source.detail!.primarySide.faction = 'shu'
    source.detail!.primarySide.power = 10000
    source.detail!.primarySide.generals = [{ id: 'liubei', name: '刘备', level: 1 }]
    source.detail!.primarySide.units = [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 617, survived: 383 }]
    source.detail!.secondarySide!.faction = 'wei'
    source.detail!.secondarySide!.power = 14000
    source.detail!.secondarySide!.generals = [{
      id: 'simayi', name: '司马懿', level: 1,
      traits: [{ traitId: 'yibing_touxi', name: '疑兵偷袭' }, { traitId: 'mouding_houfa', name: '谋定后发' }],
    }]
    source.detail!.secondarySide!.units = [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 382, survived: 618 }]
    source.detail!.rewards.generalExp = 617
    source.detail!.traits = [{
      traitId: 'mouding_houfa', traitName: '谋定后发', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'simayi',
      detail: { defenseBonusRate: 0.35, infantryDefenseModifiedUnits: { weiInfantry: 4 }, cavalryDefenseModifiedUnits: { weiInfantry: 3 }, triggerChance: 1 },
    }]
    const model = toOfficialBattleReport(source)
    expect(source.detail!.secondarySide!.generals![0]!.traits?.map((trait) => trait.traitId)).toEqual(['yibing_touxi', 'mouding_houfa'])
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 10000, generalExp: null, result: 'defeat' })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 14000, generalExp: 617, result: 'victory' })
    expect(model.sides[0].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 1000, lost: 617, survived: 383 })
    expect(model.sides[1].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 1000, lost: 382, survived: 618 })
    expect(model.sides[0].traits).toEqual([])
    expect(model.sides[1].traits).toEqual([{
      key: 'mouding_houfa-0', name: '谋定后发', phase: '防守/增援战斗前',
      detailText: '设计防御加成：35%；实际步防修正：魏步兵 +4；实际骑防修正：魏步兵 +3；触发概率：100%',
    }])
    expect(model.sides.flatMap((side) => side.traits).some((trait) => trait.name === '疑兵偷袭')).toBe(false)
  })

  it('司马懿黄巾守城按真实阶段顺序展示伤亡、加防和最终战力', () => {
    const source = report()
    source.viewType = 'defense'
    source.detail!.viewType = 'defense'
    source.detail!.sourceType = 'yellow_turban'
    source.detail!.ownerSide = 'defender'
    source.detail!.primarySide = {
      role: 'attacker', faction: 'wei', power: 6500, generals: [],
      units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 1000, survived: 0 }],
    }
    source.detail!.secondarySide = {
      role: 'defender', faction: 'wei', power: 14420,
      generals: [{ id: 'simayi', name: '司马懿', level: 1, traits: [{ traitId: 'yibing_touxi', name: '疑兵偷袭' }, { traitId: 'mouding_houfa', name: '谋定后发' }] }],
      units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 447, survived: 553 }],
    }
    source.detail!.traits = [
      {
        traitId: 'yibing_touxi', traitName: '疑兵偷袭', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'simayi',
        detail: { effectRate: 0.35, preBattleAffected: { weiInfantry: 350 }, triggerChance: 1 },
      },
      {
        traitId: 'mouding_houfa', traitName: '谋定后发', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'simayi',
        detail: { defenseBonusRate: 0.35, infantryDefenseModifiedUnits: { weiInfantry: 4 }, cavalryDefenseModifiedUnits: { weiInfantry: 3 }, triggerChance: 1 },
      },
    ]
    const model = toOfficialBattleReport(source)
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 6500 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 14420 })
    expect(model.sides[0].traits).toEqual([])
    expect(model.sides[1].traits).toEqual([
      {
        key: 'yibing_touxi-0', name: '疑兵偷袭', phase: '战斗前',
        detailText: '设计效果比例：35%；战前真实伤亡：魏步兵 +350；触发概率：100%',
      },
      {
        key: 'mouding_houfa-1', name: '谋定后发', phase: '防守/增援战斗前',
        detailText: '设计防御加成：35%；实际步防修正：魏步兵 +4；实际骑防修正：魏步兵 +3；触发概率：100%',
      },
    ])
  })

  it('关羽与张飞黄巾守城随机战前特性命中或未命中时保持准确战报', () => {
    const cases = [
      {
        generalId: 'guanyu', generalName: '关羽', defenderFaction: 'shu', defenderUnit: 'shuInfantry', defenderUnitName: '蜀步兵',
        attackerFaction: 'wei', attackerUnit: 'weiInfantry', attackerUnitName: '魏步兵', traitId: 'shuiyan_qijun', traitName: '水淹七军', bonusId: 'wusheng_pojun', bonusName: '武圣破军',
        defensePower: 1020, hitAttackPower: 650, hitDefenseLost: 52, hitAttackLost: 100, missDefenseLost: 97,
        detail: { effectRate: 0.35, maxAffectedRate: 0.35, preBattleAffected: { weiInfantry: 35 }, triggerChance: 1 },
        detailText: '设计效果比例：35%；设计最大影响比例：35%；战前真实伤亡：魏步兵 +35；触发概率：100%',
      },
      {
        generalId: 'zhangfei', generalName: '张飞', defenderFaction: 'shu', defenderUnit: 'shuInfantry', defenderUnitName: '蜀步兵',
        attackerFaction: 'wei', attackerUnit: 'weiInfantry', attackerUnitName: '魏步兵', traitId: 'zhenhe_quanjun', traitName: '万人怒吼', bonusId: 'wanren_nuhou', bonusName: '勇冠三军',
        defensePower: 1020, hitAttackPower: 500, hitDefenseLost: 36, hitAttackLost: 50, missDefenseLost: 97,
        detail: { effectRate: 0.5, maxAffectedRate: 0.5, suppressedUnits: { weiInfantry: 50 }, triggerChance: 1 },
        detailText: '设计效果比例：50%；设计最大影响比例：50%；本场压制兵力：魏步兵 +50；触发概率：100%',
      },
    ] as const

    for (const current of cases) {
      const buildReport = (triggered: boolean) => {
        const source = report()
        const attackerLost = triggered ? current.hitAttackLost : 100
        const defenderLost = triggered ? current.hitDefenseLost : current.missDefenseLost
        source.id = `${current.generalId}-yellow-${triggered ? 'hit' : 'miss'}`
        source.playerId = `player_${current.generalId}`
        source.viewType = 'defense'
        source.battleType = 'yellow_turban'
        source.type = 'defense'
        source.result = 'defender_victory'
        source.pvpReinforcements = []
        source.pvpReinforcementLosses = {}
        source.detail!.id = source.id
        source.detail!.sourceType = 'yellow_turban'
        source.detail!.viewType = 'defense'
        source.detail!.battleType = 'yellow_turban'
        source.detail!.result = 'defender_victory'
        source.detail!.winnerSide = 'defender'
        source.detail!.ownerSide = 'defender'
        source.detail!.primarySide = {
          role: 'attacker', faction: current.attackerFaction, power: triggered ? current.hitAttackPower : 1000,
          units: [{ unitType: current.attackerUnit, unitName: current.attackerUnitName, amountBefore: 100, dispatched: 100, lost: attackerLost, survived: 100 - attackerLost }],
        }
        source.detail!.secondarySide = {
          role: 'defender', playerId: `player_${current.generalId}`, faction: current.defenderFaction, power: current.defensePower,
          generals: [{
            id: current.generalId, name: current.generalName, level: 1,
            traits: [{ traitId: current.traitId, name: current.traitName }, { traitId: current.bonusId, name: current.bonusName, allowedSides: ['attacker'] }],
          }],
          units: [{ unitType: current.defenderUnit, unitName: current.defenderUnitName, amountBefore: 100, dispatched: 100, lost: defenderLost, survived: 100 - defenderLost }],
        }
        source.detail!.rewards = { resources: {}, generalExp: attackerLost }
        source.detail!.traits = triggered ? [{
          traitId: current.traitId, traitName: current.traitName, ownerSide: 'secondary', ownerRole: 'defender', ownerPlayerId: `player_${current.generalId}`, generalId: current.generalId,
          detail: current.detail,
        }] : []
        return source
      }
      const hit = toOfficialBattleReport(buildReport(true))
      const miss = toOfficialBattleReport(buildReport(false))

      expect(hit.sides[0]).toMatchObject({ role: 'attacker', power: current.hitAttackPower })
      expect(miss.sides[0]).toMatchObject({ role: 'attacker', power: 1000 })
      expect(hit.sides[1]).toMatchObject({ role: 'defender', power: current.defensePower, general: { id: current.generalId }, generalExp: current.hitAttackLost })
      expect(miss.sides[1]).toMatchObject({ role: 'defender', power: current.defensePower, general: { id: current.generalId }, generalExp: 100 })
      expect(hit.sides[0].units.find((unit) => unit.key === current.attackerUnit)).toMatchObject({ dispatched: 100, lost: current.hitAttackLost, survived: 100 - current.hitAttackLost })
      expect(miss.sides[0].units.find((unit) => unit.key === current.attackerUnit)).toMatchObject({ dispatched: 100, lost: 100, survived: 0 })
      expect(hit.sides[1].units.find((unit) => unit.key === current.defenderUnit)).toMatchObject({ dispatched: 100, lost: current.hitDefenseLost, survived: 100 - current.hitDefenseLost })
      expect(miss.sides[1].units.find((unit) => unit.key === current.defenderUnit)).toMatchObject({ dispatched: 100, lost: current.missDefenseLost, survived: 100 - current.missDefenseLost })
      expect(hit.sides[1].traits).toEqual([{ key: `${current.traitId}-player_${current.generalId}-0`, name: current.traitName, phase: '战斗前', detailText: current.detailText }])
      expect(hit.sides[0].traits).toEqual([])
      expect(miss.sides.flatMap((side) => side.traits)).toEqual([])
      expect(hit.sides.flatMap((side) => side.traits).some((trait) => trait.name === current.bonusName)).toBe(false)
      expect(miss.sides.flatMap((side) => side.traits).some((trait) => trait.name === current.bonusName)).toBe(false)
    }
  })

  it('陆逊黄巾守城火烧命中或未命中时只展示实际生效的战后伤害', () => {
    const buildReport = (triggered: boolean) => {
      const source = report()
      const attackerLost = triggered ? 200 : 97
      const traitId = triggered ? 'huoshao_lianying' : 'lianying_zengshang'
      const traitName = triggered ? '火烧联营' : '连营增伤'
      source.id = `luxun-yellow-${triggered ? 'hit' : 'miss'}`
      source.playerId = 'player_luxun'
      source.viewType = 'defense'
      source.battleType = 'yellow_turban'
      source.type = 'defense'
      source.result = 'attacker_victory'
      source.pvpReinforcements = []
      source.pvpReinforcementLosses = {}
      source.detail!.id = source.id
      source.detail!.sourceType = 'yellow_turban'
      source.detail!.viewType = 'defense'
      source.detail!.battleType = 'yellow_turban'
      source.detail!.result = 'attacker_victory'
      source.detail!.winnerSide = 'attacker'
      source.detail!.ownerSide = 'defender'
      source.detail!.primarySide = {
        role: 'attacker', faction: 'wei', power: 2000,
        units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 200, dispatched: 200, lost: attackerLost, survived: 200 - attackerLost }],
      }
      source.detail!.secondarySide = {
        role: 'defender', playerId: 'player_luxun', faction: 'wu', power: 1025,
        generals: [{
          id: 'luxun', name: '陆逊', level: 1,
          traits: [{ traitId: 'huoshao_lianying', name: '火烧联营' }, { traitId: 'lianying_zengshang', name: '连营增伤' }],
        }],
        units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 100, dispatched: 100, lost: 100, survived: 0 }],
      }
      source.detail!.rewards = { resources: {}, generalExp: attackerLost }
      source.detail!.traits = [{
        traitId, traitName, ownerSide: 'secondary', ownerRole: 'defender', ownerPlayerId: 'player_luxun', generalId: 'luxun',
        detail: triggered
          ? { effectRate: 1, maxAffectedRate: 1, targetExtraLosses: { weiInfantry: 123 }, triggerChance: 1 }
          : { effectRate: 0.1, maxAffectedRate: 0.1, targetExtraLosses: { weiInfantry: 20 }, triggerChance: 1 },
      }]
      return source
    }
    const hit = toOfficialBattleReport(buildReport(true))
    const miss = toOfficialBattleReport(buildReport(false))

    expect(hit.sides[0]).toMatchObject({ role: 'attacker', power: 2000 })
    expect(miss.sides[0]).toMatchObject({ role: 'attacker', power: 2000 })
    expect(hit.sides[1]).toMatchObject({ role: 'defender', power: 1025, general: { id: 'luxun' }, generalExp: 200 })
    expect(miss.sides[1]).toMatchObject({ role: 'defender', power: 1025, general: { id: 'luxun' }, generalExp: 97 })
    expect(hit.sides[0].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 200, lost: 200, survived: 0 })
    expect(miss.sides[0].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 200, lost: 97, survived: 103 })
    for (const model of [hit, miss]) {
      expect(model.sides[1].units.find((unit) => unit.key === 'wuInfantry')).toMatchObject({ dispatched: 100, lost: 100, survived: 0 })
      expect(model.sides[0].traits).toEqual([])
    }
    expect(hit.sides[1].traits).toEqual([{
      key: 'huoshao_lianying-player_luxun-0', name: '火烧联营', phase: '战斗结算后',
      detailText: '设计效果比例：100%；设计最大影响比例：100%；目标兵种追加损失：魏步兵 +123；触发概率：100%',
    }])
    expect(miss.sides[1].traits).toEqual([{
      key: 'lianying_zengshang-player_luxun-0', name: '连营增伤', phase: '战斗结算后',
      detailText: '设计效果比例：10%；设计最大影响比例：10%；目标兵种追加损失：魏步兵 +20；触发概率：100%',
    }])
    expect(hit.sides.flatMap((side) => side.traits).some((trait) => trait.name === '连营增伤')).toBe(false)
    expect(miss.sides.flatMap((side) => side.traits).some((trait) => trait.name === '火烧联营')).toBe(false)
  })

  it('黄盖黄巾守城苦肉命中记录零压制且未命中不影响反击', () => {
    const buildReport = (triggered: boolean) => {
      const source = report()
      source.id = `huanggai-yellow-${triggered ? 'hit' : 'miss'}`
      source.playerId = 'player_huanggai'
      source.viewType = 'defense'
      source.battleType = 'yellow_turban'
      source.type = 'defense'
      source.result = 'attacker_victory'
      source.pvpReinforcements = []
      source.pvpReinforcementLosses = {}
      source.detail!.id = source.id
      source.detail!.sourceType = 'yellow_turban'
      source.detail!.viewType = 'defense'
      source.detail!.battleType = 'yellow_turban'
      source.detail!.result = 'attacker_victory'
      source.detail!.winnerSide = 'attacker'
      source.detail!.ownerSide = 'defender'
      source.detail!.primarySide = {
        role: 'attacker', faction: 'wei', power: 2000,
        units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 200, dispatched: 200, lost: 97, survived: 103 }],
      }
      source.detail!.secondarySide = {
        role: 'defender', playerId: 'player_huanggai', faction: 'wu', power: 1025,
        generals: [{
          id: 'huanggai', name: '黄盖', level: 1,
          traits: [{ traitId: 'kurouji', name: '苦肉计' }, { traitId: 'kurou_fanji', name: '苦肉反击' }],
        }],
        units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 100, dispatched: 100, lost: 100, survived: 0 }],
      }
      source.detail!.rewards = { resources: {}, generalExp: 97 }
      source.detail!.traits = [
        ...(triggered ? [{
          traitId: 'kurouji', traitName: '苦肉计', ownerSide: 'secondary', ownerRole: 'defender', ownerPlayerId: 'player_huanggai', generalId: 'huanggai',
          detail: { disableTraitCount: 1, disabledTraitCount: 0, triggerChance: 1 },
        }] : []),
        {
          traitId: 'kurou_fanji', traitName: '苦肉反击', ownerSide: 'secondary', ownerRole: 'defender', ownerPlayerId: 'player_huanggai', generalId: 'huanggai',
          detail: { effectRate: 0.1, extraLosses: { weiInfantry: 20 }, triggerChance: 1 },
        },
      ]
      return source
    }
    const hit = toOfficialBattleReport(buildReport(true))
    const miss = toOfficialBattleReport(buildReport(false))

    for (const model of [hit, miss]) {
      expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 2000 })
      expect(model.sides[1]).toMatchObject({ role: 'defender', power: 1025, general: { id: 'huanggai' }, generalExp: 97 })
      expect(model.sides[0].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 200, lost: 97, survived: 103 })
      expect(model.sides[1].units.find((unit) => unit.key === 'wuInfantry')).toMatchObject({ dispatched: 100, lost: 100, survived: 0 })
      expect(model.sides[0].traits).toEqual([])
    }
    expect(hit.sides[1].traits).toEqual([
      {
        key: 'kurouji-player_huanggai-0', name: '苦肉计', phase: '战斗结算后',
        detailText: '设计压制特性数：1；实际压制特性数：0；触发概率：100%',
      },
      {
        key: 'kurou_fanji-player_huanggai-1', name: '苦肉反击', phase: '战斗结算后',
        detailText: '设计效果比例：10%；追加损失：魏步兵 +20；触发概率：100%',
      },
    ])
    expect(miss.sides[1].traits).toEqual([{
      key: 'kurou_fanji-player_huanggai-0', name: '苦肉反击', phase: '战斗结算后',
      detailText: '设计效果比例：10%；追加损失：魏步兵 +20；触发概率：100%',
    }])
    expect(miss.sides.flatMap((side) => side.traits).some((trait) => trait.name === '苦肉计')).toBe(false)
  })

  it('马超黄巾守城西凉命中或未命中时保持骑兵损失和被动快照准确', () => {
    const buildReport = (triggered: boolean) => {
      const source = report()
      const attackerLost = triggered ? 58 : 34
      const generalExp = triggered ? 116 : 68
      source.id = `machao-yellow-${triggered ? 'hit' : 'miss'}`
      source.playerId = 'player_machao'
      source.viewType = 'defense'
      source.battleType = 'yellow_turban'
      source.type = 'defense'
      source.result = 'attacker_victory'
      source.pvpReinforcements = []
      source.pvpReinforcementLosses = {}
      source.detail!.id = source.id
      source.detail!.sourceType = 'yellow_turban'
      source.detail!.viewType = 'defense'
      source.detail!.battleType = 'yellow_turban'
      source.detail!.result = 'attacker_victory'
      source.detail!.winnerSide = 'attacker'
      source.detail!.ownerSide = 'defender'
      source.detail!.primarySide = {
        role: 'attacker', faction: 'wei', power: 2800,
        units: [{ unitType: 'weiCavalry', unitName: '魏骑兵', amountBefore: 200, dispatched: 200, lost: attackerLost, survived: 200 - attackerLost }],
      }
      source.detail!.secondarySide = {
        role: 'defender', playerId: 'player_machao', faction: 'shu', power: 816,
        generals: [{
          id: 'machao', name: '马超', level: 1,
          stats: { force: 0, intelligence: 0, command: 0, politics: 0 }, effectiveStats: { force: 20, intelligence: 0, command: 0, politics: 0 },
          buffs: { attackBonus: 0.4 }, traits: [{ traitId: 'xiliang_tuji', name: '西凉突击' }, { traitId: 'tianshen_xiafan', name: '天神下凡' }],
        }],
        units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 100, dispatched: 100, lost: 100, survived: 0 }],
      }
      source.detail!.rewards = { resources: {}, generalExp }
      source.detail!.traits = triggered ? [{
        traitId: 'xiliang_tuji', traitName: '西凉突击', ownerSide: 'secondary', ownerRole: 'defender', ownerPlayerId: 'player_machao', generalId: 'machao',
        detail: { effectRate: 0.12, targetExtraLosses: { weiCavalry: 24 }, triggerChance: 1 },
      }] : []
      return source
    }
    const hitSource = buildReport(true)
    const missSource = buildReport(false)
    const hit = toOfficialBattleReport(hitSource)
    const miss = toOfficialBattleReport(missSource)

    expect(hit.sides[0]).toMatchObject({ role: 'attacker', power: 2800 })
    expect(miss.sides[0]).toMatchObject({ role: 'attacker', power: 2800 })
    expect(hit.sides[1]).toMatchObject({ role: 'defender', power: 816, general: { id: 'machao' }, generalExp: 116 })
    expect(miss.sides[1]).toMatchObject({ role: 'defender', power: 816, general: { id: 'machao' }, generalExp: 68 })
    expect(hit.sides[0].units.find((unit) => unit.key === 'weiCavalry')).toMatchObject({ dispatched: 200, lost: 58, survived: 142 })
    expect(miss.sides[0].units.find((unit) => unit.key === 'weiCavalry')).toMatchObject({ dispatched: 200, lost: 34, survived: 166 })
    for (const model of [hit, miss]) {
      expect(model.sides[1].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 100, lost: 100, survived: 0 })
      expect(model.sides[0].traits).toEqual([])
    }
    for (const source of [hitSource, missSource]) {
      expect(source.detail!.secondarySide!.generals![0]).toMatchObject({
        effectiveStats: { force: 20 }, buffs: { attackBonus: 0.4 },
        traits: [{ traitId: 'xiliang_tuji' }, { traitId: 'tianshen_xiafan' }],
      })
    }
    expect(hit.sides[1].traits).toEqual([{
      key: 'xiliang_tuji-player_machao-0', name: '西凉突击', phase: '战斗结算后',
      detailText: '设计效果比例：12%；目标兵种追加损失：魏骑兵 +24；触发概率：100%',
    }])
    expect(miss.sides.flatMap((side) => side.traits)).toEqual([])
    expect(hit.sides.flatMap((side) => side.traits).some((trait) => trait.name === '天神下凡')).toBe(false)
  })

  it('赵云与孙权黄巾守城合法未命中时保持基础兵损和空时间线', () => {
    const cases = [
      {
        generalId: 'zhaoyun', generalName: '赵云', traitIds: ['longdan_jiuyuan', 'qijin_qichu'],
        attackerAmount: 200, attackerPower: 2000, attackerLost: 76, attackerSurvived: 124,
        defenderFaction: 'shu', defenderUnit: 'greedyWolf', defenderUnitName: '贪狼营', defenderPower: 1020, defenderLost: 100, defenderSurvived: 0, generalExp: 76,
        result: 'attacker_victory', winnerSide: 'attacker',
      },
      {
        generalId: 'sunquan', generalName: '孙权', traitIds: ['jiangdong_haoling', 'jiangdong_gushou'],
        attackerAmount: 100, attackerPower: 1000, attackerLost: 100, attackerSurvived: 0,
        defenderFaction: 'wu', defenderUnit: 'wuInfantry', defenderUnitName: '吴步兵', defenderPower: 1025, defenderLost: 96, defenderSurvived: 4, generalExp: 100,
        result: 'defender_victory', winnerSide: 'defender',
      },
    ] as const

    for (const current of cases) {
      const source = report()
      source.id = `${current.generalId}-yellow-miss`
      source.playerId = `player_${current.generalId}`
      source.viewType = 'defense'
      source.battleType = 'yellow_turban'
      source.type = 'defense'
      source.result = current.result
      source.pvpReinforcements = []
      source.pvpReinforcementLosses = {}
      source.detail!.id = source.id
      source.detail!.sourceType = 'yellow_turban'
      source.detail!.viewType = 'defense'
      source.detail!.battleType = 'yellow_turban'
      source.detail!.result = current.result
      source.detail!.winnerSide = current.winnerSide
      source.detail!.ownerSide = 'defender'
      source.detail!.primarySide = {
        role: 'attacker', faction: 'wei', power: current.attackerPower,
        units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: current.attackerAmount, dispatched: current.attackerAmount, lost: current.attackerLost, survived: current.attackerSurvived }],
      }
      source.detail!.secondarySide = {
        role: 'defender', playerId: `player_${current.generalId}`, faction: current.defenderFaction, power: current.defenderPower,
        generals: [{ id: current.generalId, name: current.generalName, level: 1, traits: current.traitIds.map((traitId) => ({ traitId, name: traitLabel(traitId) })) }],
        units: [{ unitType: current.defenderUnit, unitName: current.defenderUnitName, amountBefore: 100, dispatched: 100, lost: current.defenderLost, survived: current.defenderSurvived }],
      }
      source.detail!.rewards = { resources: {}, generalExp: current.generalExp }
      source.detail!.traits = []
      const model = toOfficialBattleReport(source)

      expect(model.sides[0]).toMatchObject({ role: 'attacker', power: current.attackerPower })
      expect(model.sides[1]).toMatchObject({ role: 'defender', power: current.defenderPower, general: { id: current.generalId }, generalExp: current.generalExp })
      expect(model.sides[0].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: current.attackerAmount, lost: current.attackerLost, survived: current.attackerSurvived })
      expect(model.sides[1].units.find((unit) => unit.key === current.defenderUnit)).toMatchObject({ dispatched: 100, lost: current.defenderLost, survived: current.defenderSurvived })
      expect(source.detail!.secondarySide!.generals![0]!.traits?.map((trait) => trait.traitId)).toEqual([...current.traitIds])
      expect(model.sides.flatMap((side) => side.traits)).toEqual([])
    }
  })

  it('司马懿黄巾与 NPC 疑兵命中或未命中时严格隔离谋定方向', () => {
    const cases = [
      {
        id: 'simayi-yellow-miss', sourceType: 'yellow_turban', viewType: 'defense', result: 'defender_victory', winnerSide: 'defender', ownerSide: 'defender',
        primaryPower: 10000, primaryAmount: 1000, primaryLost: 1000, primarySurvived: 0,
        secondaryPower: 14420, secondaryAmount: 1000, secondaryLost: 594, secondarySurvived: 406, generalExp: 1000,
        generalSide: 'secondary', traits: [{
          traitId: 'mouding_houfa', traitName: '谋定后发', ownerSide: 'secondary', ownerRole: 'defender', ownerPlayerId: 'player_simayi', generalId: 'simayi',
          detail: { defenseBonusRate: 0.35, infantryDefenseModifiedUnits: { weiInfantry: 4 }, cavalryDefenseModifiedUnits: { weiInfantry: 3 }, triggerChance: 1 },
        }],
      },
      {
        id: 'simayi-npc-hit', sourceType: 'npc_city', viewType: 'attack', result: 'attacker_victory', winnerSide: 'attacker', ownerSide: 'attacker',
        primaryPower: 2000, primaryAmount: 200, primaryLost: 40, primarySurvived: 160,
        secondaryPower: 650, secondaryAmount: 100, secondaryLost: 100, secondarySurvived: 0, generalExp: 100,
        generalSide: 'primary', traits: [{
          traitId: 'yibing_touxi', traitName: '疑兵偷袭', ownerSide: 'primary', ownerRole: 'attacker', ownerPlayerId: 'player_simayi', generalId: 'simayi',
          detail: { effectRate: 0.35, preBattleAffected: { weiInfantry: 35 }, triggerChance: 1 },
        }],
      },
      {
        id: 'simayi-npc-miss', sourceType: 'npc_city', viewType: 'attack', result: 'attacker_victory', winnerSide: 'attacker', ownerSide: 'attacker',
        primaryPower: 2000, primaryAmount: 200, primaryLost: 74, primarySurvived: 126,
        secondaryPower: 1000, secondaryAmount: 100, secondaryLost: 100, secondarySurvived: 0, generalExp: 100,
        generalSide: 'primary', traits: [],
      },
    ] as const

    for (const current of cases) {
      const source = report()
      const general = { id: 'simayi', name: '司马懿', level: 1, traits: [{ traitId: 'yibing_touxi', name: '疑兵偷袭' }, { traitId: 'mouding_houfa', name: '谋定后发', allowedSides: ['defender' as const, 'reinforcement' as const] }] }
      source.id = current.id
      source.playerId = 'player_simayi'
      source.viewType = current.viewType
      source.battleType = current.sourceType === 'yellow_turban' ? 'yellow_turban' : 'attack'
      source.type = current.viewType
      source.result = current.result
      source.pvpReinforcements = []
      source.pvpReinforcementLosses = {}
      source.detail!.id = current.id
      source.detail!.sourceType = current.sourceType
      source.detail!.viewType = current.viewType
      source.detail!.battleType = current.sourceType === 'yellow_turban' ? 'yellow_turban' : 'attack'
      source.detail!.result = current.result
      source.detail!.winnerSide = current.winnerSide
      source.detail!.ownerSide = current.ownerSide
      source.detail!.primarySide = {
        role: 'attacker', faction: 'wei', power: current.primaryPower, generals: current.generalSide === 'primary' ? [general] : [],
        units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: current.primaryAmount, dispatched: current.primaryAmount, lost: current.primaryLost, survived: current.primarySurvived }],
      }
      source.detail!.secondarySide = {
        role: 'defender', faction: 'wei', power: current.secondaryPower, generals: current.generalSide === 'secondary' ? [general] : [],
        units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: current.secondaryAmount, dispatched: current.secondaryAmount, lost: current.secondaryLost, survived: current.secondarySurvived }],
      }
      source.detail!.rewards = { resources: {}, generalExp: current.generalExp }
      source.detail!.traits = [...current.traits]
      const model = toOfficialBattleReport(source)

      expect(model.sides[0]).toMatchObject({ role: 'attacker', power: current.primaryPower })
      expect(model.sides[1]).toMatchObject({ role: 'defender', power: current.secondaryPower })
      expect(model.sides[0].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: current.primaryAmount, lost: current.primaryLost, survived: current.primarySurvived })
      expect(model.sides[1].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: current.secondaryAmount, lost: current.secondaryLost, survived: current.secondarySurvived })
      expect((current.generalSide === 'primary' ? source.detail!.primarySide : source.detail!.secondarySide)!.generals![0]!.traits?.map((trait) => trait.traitId)).toEqual(['yibing_touxi', 'mouding_houfa'])
      expect(model.sides.flatMap((side) => side.traits).map((trait) => trait.name)).toEqual(current.traits.map((trait) => trait.traitName))
      expect(model.sides.flatMap((side) => side.traits).some((trait) => trait.name === '谋定后发')).toBe(current.sourceType === 'yellow_turban')
      expect(model.sides.flatMap((side) => side.traits).some((trait) => trait.name === '疑兵偷袭')).toBe(current.id === 'simayi-npc-hit')
      expect(model.sides[current.generalSide === 'primary' ? 0 : 1]).toMatchObject({ generalExp: current.generalExp })
    }
  })

  it('典韦黄巾防守与 NPC 进攻分别展示护主血战和死战到底的实际结果', () => {
    const general = { id: 'dianwei', name: '典韦', level: 1, traits: [{ traitId: 'huzhu_xuezhan', name: '护主血战', allowedSides: ['defender' as const, 'reinforcement' as const] }, { traitId: 'sizhandaodi', name: '死战到底', allowedSides: ['attacker' as const] }] }
    const yellow = report()
    yellow.viewType = 'defense'
    yellow.battleType = 'yellow_turban'
    yellow.detail!.viewType = 'defense'
    yellow.detail!.sourceType = 'yellow_turban'
    yellow.detail!.battleType = 'yellow_turban'
    yellow.detail!.ownerSide = 'defender'
    yellow.detail!.primarySide = { role: 'attacker', faction: 'wei', power: 2000, generals: [], units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 200, dispatched: 200, lost: 200, survived: 0 }] }
    yellow.detail!.secondarySide = { role: 'defender', faction: 'wei', power: 3399, generals: [general], units: [{ unitType: 'jinWeiSoldier', unitName: '禁卫甲士', amountBefore: 100, dispatched: 100, lost: 47, survived: 53 }] }
    yellow.detail!.traits = [{ traitId: 'huzhu_xuezhan', traitName: '护主血战', ownerSide: 'secondary', ownerRole: 'defender', ownerPlayerId: 'player_dianwei', generalId: 'dianwei', detail: { generalDefenseFlat: 20, infantryDefenseModifiedUnits: { jinWeiSoldier: 20 }, cavalryDefenseModifiedUnits: { jinWeiSoldier: 20 }, triggerChance: 1 } }]
    const yellowModel = toOfficialBattleReport(yellow)
    expect(yellowModel.sides[1]).toMatchObject({ role: 'defender', power: 3399 })
    expect(yellowModel.sides[1].units.find((unit) => unit.key === 'jinWeiSoldier')).toMatchObject({ dispatched: 100, lost: 47, survived: 53 })
    expect(yellowModel.sides[1].traits[0]).toMatchObject({ name: '护主血战', phase: '防守/增援战斗前', detailText: '设计全军防御增加：20；实际步防修正：禁卫甲士 +20；实际骑防修正：禁卫甲士 +20；触发概率：100%' })

    for (const hit of [true, false]) {
      const npc = report()
      npc.detail!.sourceType = 'npc_city'
      npc.detail!.primarySide = { role: 'attacker', faction: 'wei', power: hit ? 1400 : 1000, generals: [general], units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: 100, survived: 0 }] }
      npc.detail!.secondarySide = { role: 'defender', faction: 'wei', power: 2000, generals: [], units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 200, dispatched: 200, lost: hit ? 120 : 74, survived: hit ? 80 : 126 }] }
      npc.detail!.traits = hit ? [{ traitId: 'sizhandaodi', traitName: '死战到底', ownerSide: 'primary', ownerRole: 'attacker', ownerPlayerId: 'player_dianwei', generalId: 'dianwei', detail: { attackBonusRate: 0.35, attackModifiedUnits: { weiInfantry: 4 }, triggerChance: 0.6 } }] : []
      const npcModel = toOfficialBattleReport(npc)
      expect(npcModel.sides[0].power).toBe(hit ? 1400 : 1000)
      expect(npcModel.sides[1].units.find((unit) => unit.key === 'weiInfantry')?.lost).toBe(hit ? 120 : 74)
      expect(npcModel.sides.flatMap((side) => side.traits).map((trait) => trait.name)).toEqual(hit ? ['死战到底'] : [])
    }
  })

  it('关羽张辽张飞 NPC 战前随机未命中时保留各自独立战斗加成', () => {
    const cases = [
      {
        generalId: 'guanyu', generalName: '关羽', attackerUnit: 'weiInfantry', attackerUnitName: '魏步兵', specialId: 'shuiyan_qijun', specialName: '水淹七军', bonusId: 'wusheng_pojun', bonusName: '武圣破军',
        hitPower: [2400, 650], missPower: [2400, 1000], hitLosses: [31, 100], missLosses: [57, 100], hitExp: 100, missExp: 100,
        hitSpecialDetail: { effectRate: 0.35, maxAffectedRate: 0.35, preBattleAffected: { weiInfantry: 35 }, triggerChance: 1 },
        hitBonusDetail: { attackBonusRate: 0.2, attackModifiedUnits: { weiInfantry: 2 }, triggerChance: 1 },
        missBonusDetail: { attackBonusRate: 0.2, attackModifiedUnits: { weiInfantry: 2 }, triggerChance: 1 },
      },
      {
        generalId: 'zhangliao', generalName: '张辽', attackerUnit: 'weiCavalry', attackerUnitName: '魏骑兵', specialId: 'weizhen_zhenhe', specialName: '震慑全军', bonusId: 'weizhen_xiaoyao', bonusName: '威震逍遥',
        hitPower: [3800, 600], missPower: [3800, 800], hitLosses: [14, 75], missLosses: [21, 100], hitExp: 75, missExp: 100,
        hitSpecialDetail: { effectRate: 0.25, suppressedUnits: { weiInfantry: 25 }, fledUnits: { weiInfantry: 25 }, returnedUnits: { weiInfantry: 25 }, triggerChance: 1 },
        hitBonusDetail: { attackBonusRate: 0.35, attackModifiedUnits: { weiCavalry: 5 }, triggerChance: 1 },
        missBonusDetail: { attackBonusRate: 0.35, attackModifiedUnits: { weiCavalry: 5 }, triggerChance: 1 },
      },
      {
        generalId: 'zhangfei', generalName: '张飞', attackerUnit: 'weiInfantry', attackerUnitName: '魏步兵', specialId: 'zhenhe_quanjun', specialName: '万人怒吼', bonusId: 'wanren_nuhou', bonusName: '勇冠三军',
        hitPower: [2400, 500], missPower: [2400, 1000], hitLosses: [21, 50], missLosses: [57, 100], hitExp: 50, missExp: 100,
        hitSpecialDetail: { effectRate: 0.5, maxAffectedRate: 0.5, suppressedUnits: { weiInfantry: 50 }, triggerChance: 1 },
        hitBonusDetail: { attackBonusRate: 0.2, attackModifiedUnits: { weiInfantry: 2 }, triggerChance: 1 },
        missBonusDetail: { attackBonusRate: 0.2, attackModifiedUnits: { weiInfantry: 2 }, triggerChance: 1 },
      },
    ] as const

    for (const current of cases) {
      for (const triggered of [true, false]) {
        const powers = triggered ? current.hitPower : current.missPower
        const losses = triggered ? current.hitLosses : current.missLosses
        const generalExp = triggered ? current.hitExp : current.missExp
        const source = report()
        source.id = `${current.generalId}-npc-${triggered ? 'hit' : 'miss'}`
        source.playerId = `player_${current.generalId}`
        source.viewType = 'attack'
        source.battleType = 'attack'
        source.type = 'attack'
        source.result = 'attacker_victory'
        source.pvpReinforcements = []
        source.pvpReinforcementLosses = {}
        source.detail!.id = source.id
        source.detail!.sourceType = 'npc_city'
        source.detail!.viewType = 'attack'
        source.detail!.battleType = 'attack'
        source.detail!.result = 'attacker_victory'
        source.detail!.winnerSide = 'attacker'
        source.detail!.ownerSide = 'attacker'
        source.detail!.primarySide = {
          role: 'attacker', faction: 'wei', power: powers[0],
          generals: [{ id: current.generalId, name: current.generalName, level: 1, traits: [{ traitId: current.specialId, name: current.specialName }, { traitId: current.bonusId, name: current.bonusName }] }],
          units: [{ unitType: current.attackerUnit, unitName: current.attackerUnitName, amountBefore: 200, dispatched: 200, lost: losses[0], survived: 200 - losses[0] }],
        }
        source.detail!.secondarySide = {
          role: 'defender', faction: 'wei', power: powers[1], generals: [],
          units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: losses[1], survived: 100 - losses[1] }],
        }
        source.detail!.rewards = { resources: {}, generalExp }
        source.detail!.traits = [
          ...(triggered ? [{ traitId: current.specialId, traitName: current.specialName, ownerSide: 'primary' as const, ownerRole: 'attacker' as const, ownerPlayerId: `player_${current.generalId}`, generalId: current.generalId, detail: current.hitSpecialDetail }] : []),
          { traitId: current.bonusId, traitName: current.bonusName, ownerSide: 'primary', ownerRole: 'attacker', ownerPlayerId: `player_${current.generalId}`, generalId: current.generalId, detail: triggered ? current.hitBonusDetail : current.missBonusDetail },
        ]
        const model = toOfficialBattleReport(source)

        expect(model.sides[0]).toMatchObject({ role: 'attacker', power: powers[0], general: { id: current.generalId }, generalExp })
        expect(model.sides[1]).toMatchObject({ role: 'defender', power: powers[1] })
        expect(model.sides[0].units.find((unit) => unit.key === current.attackerUnit)).toMatchObject({ dispatched: 200, lost: losses[0], survived: 200 - losses[0] })
        expect(model.sides[1].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 100, lost: losses[1], survived: 100 - losses[1] })
        expect(source.detail!.primarySide.generals![0]!.traits?.map((trait) => trait.traitId)).toEqual([current.specialId, current.bonusId])
        expect(model.sides[0].traits.map((trait) => trait.name)).toEqual(triggered ? [current.specialName, current.bonusName] : [current.bonusName])
        expect(model.sides[0].traits.some((trait) => trait.name === current.specialName)).toBe(triggered)
        expect(model.sides[1].traits).toEqual([])
      }
    }
  })

  it('马超黄忠陆逊黄盖孙策 NPC 随机命中或未命中时保持后续效果和精确兵损', () => {
    const cases = [
      {
        generalId: 'machao', generalName: '马超', traitIds: ['xiliang_tuji', 'tianshen_xiafan'], playerUnit: 'weiInfantry', playerUnitName: '魏步兵', playerAmount: 1000, npcUnit: 'weiCavalry', npcUnitName: '魏骑兵', npcAmount: 1000,
        hitPower: [14000, 10000], missPower: [14000, 10000], hitLosses: [382, 737], missLosses: [382, 617], hitTimeline: ['xiliang_tuji'], missTimeline: [],
        details: { xiliang_tuji: { effectRate: 0.12, targetExtraLosses: { weiCavalry: 120 }, triggerChance: 1 } },
        snapshot: { stats: { force: 0 }, effectiveStats: { force: 20 }, buffs: { attackBonus: 0.4 } },
      },
      {
        generalId: 'huangzhong', generalName: '黄忠', traitIds: ['baibu_chuanyang', 'laodang_yizhuang'], playerUnit: 'weiInfantry', playerUnitName: '魏步兵', playerAmount: 1000, npcUnit: 'weiInfantry', npcUnitName: '魏步兵', npcAmount: 1000,
        hitPower: [10000, 8000], missPower: [10000, 10000], hitLosses: [421, 678], missLosses: [500, 600], hitTimeline: ['baibu_chuanyang', 'laodang_yizhuang'], missTimeline: ['laodang_yizhuang'],
        details: {
          baibu_chuanyang: { enemyDefenseReductionRate: 0.2, infantryDefenseModifiedUnits: { weiInfantry: -2 }, cavalryDefenseModifiedUnits: { weiInfantry: -2 }, triggerChance: 1 },
          laodang_yizhuang: { effectRate: 0.1, extraLosses: { weiInfantry: 100 } },
        }, snapshot: {},
      },
      {
        generalId: 'luxun', generalName: '陆逊', traitIds: ['huoshao_lianying', 'lianying_zengshang'], playerUnit: 'weiInfantry', playerUnitName: '魏步兵', playerAmount: 1000, npcUnit: 'weiInfantry', npcUnitName: '魏步兵', npcAmount: 1000,
        hitPower: [10000, 10000], missPower: [10000, 10000], hitLosses: [500, 1000], missLosses: [500, 600], hitTimeline: ['huoshao_lianying'], missTimeline: ['lianying_zengshang'],
        details: {
          huoshao_lianying: { effectRate: 1, maxAffectedRate: 1, targetExtraLosses: { weiInfantry: 500 }, triggerChance: 1 },
          lianying_zengshang: { effectRate: 0.1, targetExtraLosses: { weiInfantry: 100 } },
        }, snapshot: {},
      },
      {
        generalId: 'huanggai', generalName: '黄盖', traitIds: ['kurouji', 'kurou_fanji'], playerUnit: 'weiInfantry', playerUnitName: '魏步兵', playerAmount: 1000, npcUnit: 'weiInfantry', npcUnitName: '魏步兵', npcAmount: 1000,
        hitPower: [10000, 10000], missPower: [10000, 10000], hitLosses: [500, 600], missLosses: [500, 600], hitTimeline: ['kurouji', 'kurou_fanji'], missTimeline: ['kurou_fanji'],
        details: {
          kurouji: { disableTraitCount: 1, disabledTraitCount: 0, triggerChance: 1 },
          kurou_fanji: { effectRate: 0.1, extraLosses: { weiInfantry: 100 } },
        }, snapshot: {},
      },
      {
        generalId: 'sunce', generalName: '孙策', traitIds: ['xiaobawang_zhuiji', 'xiaobawang_tieqi'], playerUnit: 'overlordRider', playerUnitName: '霸王骑', playerAmount: 200, npcUnit: 'weiInfantry', npcUnitName: '魏步兵', npcAmount: 1000,
        hitPower: [15600, 8000], missPower: [15600, 8000], hitLosses: [55, 821], missLosses: [55, 721], hitTimeline: ['xiaobawang_tieqi', 'xiaobawang_zhuiji'], missTimeline: ['xiaobawang_tieqi'],
        details: {
          xiaobawang_tieqi: { unitAttackFlat: 50, attackModifiedUnits: { overlordRider: 50 } },
          xiaobawang_zhuiji: { effectRate: 0.1, extraLosses: { weiInfantry: 100 }, triggerChance: 1 },
        }, snapshot: {},
      },
    ]

    for (const current of cases) {
      for (const hit of [true, false]) {
        const powers = hit ? current.hitPower : current.missPower
        const losses = hit ? current.hitLosses : current.missLosses
        const timeline = hit ? current.hitTimeline : current.missTimeline
        const details = current.details as unknown as Record<string, Record<string, unknown>>
        const source = report()
        source.id = `${current.generalId}-npc-late-${hit ? 'hit' : 'miss'}`
        source.playerId = `player_${current.generalId}`
        source.viewType = 'attack'
        source.battleType = 'plunder'
        source.type = 'plunder'
        source.result = 'attacker_victory'
        source.pvpReinforcements = []
        source.pvpReinforcementLosses = {}
        source.detail!.id = source.id
        source.detail!.sourceType = 'npc_city'
        source.detail!.viewType = 'attack'
        source.detail!.battleType = 'plunder'
        source.detail!.result = 'attacker_victory'
        source.detail!.winnerSide = 'attacker'
        source.detail!.ownerSide = 'attacker'
        source.detail!.primarySide = {
          role: 'attacker', faction: 'wei', power: powers[0],
          generals: [{ id: current.generalId, name: current.generalName, level: 1, ...current.snapshot, traits: current.traitIds.map((traitId) => ({ traitId, name: traitLabel(traitId) })) }],
          units: [{ unitType: current.playerUnit, unitName: current.playerUnitName, amountBefore: current.playerAmount, dispatched: current.playerAmount, lost: losses[0], survived: current.playerAmount - losses[0] }],
        }
        source.detail!.secondarySide = {
          role: 'defender', faction: 'wei', power: powers[1], generals: [],
          units: [{ unitType: current.npcUnit, unitName: current.npcUnitName, amountBefore: current.npcAmount, dispatched: current.npcAmount, lost: losses[1], survived: current.npcAmount - losses[1] }],
        }
        source.detail!.rewards = { resources: {}, generalExp: losses[1] }
        source.detail!.traits = timeline.map((traitId) => ({
          traitId, traitName: traitLabel(traitId), ownerSide: 'primary', ownerRole: 'attacker', ownerPlayerId: `player_${current.generalId}`, generalId: current.generalId, detail: details[traitId],
        }))
        const model = toOfficialBattleReport(source)

        expect(source.detail!.sourceType).toBe('npc_city')
        expect(model.sides[0]).toMatchObject({ role: 'attacker', power: powers[0], general: { id: current.generalId }, generalExp: losses[1] })
        expect(model.sides[1]).toMatchObject({ role: 'defender', power: powers[1] })
        expect(model.sides[0].units.find((unit) => unit.key === current.playerUnit)).toMatchObject({ dispatched: current.playerAmount, lost: losses[0], survived: current.playerAmount - losses[0] })
        expect(model.sides[1].units.find((unit) => unit.key === current.npcUnit)).toMatchObject({ dispatched: current.npcAmount, lost: losses[1], survived: current.npcAmount - losses[1] })
        expect(source.detail!.primarySide.generals![0]!.traits?.map((trait) => trait.traitId)).toEqual(current.traitIds)
        expect(model.sides[0].traits.map((trait) => trait.name)).toEqual(timeline.map((traitId) => traitLabel(traitId)))
        expect(model.sides[0].traits.some((trait) => trait.name === traitLabel(current.traitIds[0]))).toBe(hit)
        expect(model.sides[1].traits).toEqual([])
      }
    }
  })

  it('赵云黄巾守城只展示龙胆真实减损且不补造七进七出', () => {
    const source = report()
    source.viewType = 'defense'
    source.detail!.viewType = 'defense'
    source.detail!.sourceType = 'yellow_turban'
    source.detail!.ownerSide = 'defender'
    source.detail!.primarySide = {
      role: 'attacker', faction: 'wei', power: 2000, generals: [],
      units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 200, dispatched: 200, lost: 76, survived: 124 }],
    }
    source.detail!.secondarySide = {
      role: 'defender', faction: 'shu', power: 1020,
      generals: [{ id: 'zhaoyun', name: '赵云', level: 1, traits: [{ traitId: 'longdan_jiuyuan', name: '龙胆救援' }, { traitId: 'qijin_qichu', name: '七进七出' }] }],
      units: [{ unitType: 'greedyWolf', unitName: '贪狼营', amountBefore: 100, dispatched: 100, lost: 80, survived: 20 }],
    }
    source.detail!.traits = [{
      traitId: 'longdan_jiuyuan', traitName: '龙胆救援', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'zhaoyun',
      detail: { lossReductionRate: 0.2, reducedLosses: { greedyWolf: 20 }, triggerChance: 1 },
    }]
    const model = toOfficialBattleReport(source)
    expect(source.detail!.secondarySide!.generals?.[0]?.traits?.map((trait) => trait.traitId)).toEqual(['longdan_jiuyuan', 'qijin_qichu'])
    expect(model.sides[0].traits).toEqual([])
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 1020 })
    expect(model.sides[1].units.find((unit) => unit.key === 'greedyWolf')).toMatchObject({ dispatched: 100, lost: 80, survived: 20 })
    expect(model.sides[1].traits).toEqual([{
      key: 'longdan_jiuyuan-0', name: '龙胆救援', phase: '防守/增援战斗前及掠夺结算',
      detailText: '设计减损比例：20%；减少损失：贪狼营 +20；触发概率：100%',
    }])
  })

  it('主城赵云龙胆只减少主守军损失且不替同兵种援军减损', () => {
    const source = report()
    source.result = 'draw'
    source.battleType = 'plunder'
    source.detail!.result = 'draw'
    source.detail!.battleType = 'plunder'
    source.detail!.winnerSide = 'none'
    source.detail!.primarySide.faction = 'wei'
    source.detail!.primarySide.power = 10000
    source.detail!.primarySide.generals = [{ id: 'caocao', name: '曹操', level: 1 }]
    source.detail!.primarySide.units = [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 }]
    source.detail!.secondarySide!.faction = 'shu'
    source.detail!.secondarySide!.power = 10000
    source.detail!.secondarySide!.generals = [{ id: 'zhaoyun', name: '赵云', level: 1 }]
    source.detail!.secondarySide!.units = [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 500, dispatched: 500, lost: 200, survived: 300 }]
    source.detail!.rewards.generalExp = 450
    source.detail!.traits = [{
      traitId: 'longdan_jiuyuan', traitName: '龙胆救援', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'zhaoyun',
      detail: { lossReductionRate: 0.2, reducedLosses: { shuInfantry: 50 }, triggerChance: 1 },
    }]
    source.pvpReinforcements = [{
      reinforcementId: 'rein_liubei', fromPlayerId: 'player_liubei', fromPlayerName: '刘备援军', faction: 'shu',
      troops: { shuInfantry: 500 }, generalExpGained: 500, generals: [{ id: 'liubei', name: '刘备', level: 1 }],
    }]
    source.pvpReinforcementLosses = { rein_liubei: { shuInfantry: 250 } }

    const model = toOfficialBattleReport(source)
    expect(model.sides.map((side) => side.role)).toEqual(['attacker', 'defender', 'reinforcement'])
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 10000, general: { id: 'caocao' }, generalExp: 450 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 10000, general: { id: 'zhaoyun' } })
    expect(model.sides[2]).toMatchObject({ role: 'reinforcement', general: { id: 'liubei' }, generalExp: 500 })
    expect(model.sides[0].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 1000, lost: 500, survived: 500 })
    expect(model.sides[1].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 500, lost: 200, survived: 300 })
    expect(model.sides[2].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 500, lost: 250, survived: 250 })
    expect(model.sides[1].traits).toEqual([{
      key: 'longdan_jiuyuan-0', name: '龙胆救援', phase: '防守/增援战斗前及掠夺结算',
      detailText: '设计减损比例：20%；减少损失：蜀步兵 +50；触发概率：100%',
    }])
    expect(model.sides[0].traits).toEqual([])
    expect(model.sides[2].traits).toEqual([])
  })

  it('主城赵云龙胆合法未命中时主守军和同兵种援军都承担完整损失', () => {
    const source = report()
    source.result = 'draw'
    source.battleType = 'plunder'
    source.detail!.result = 'draw'
    source.detail!.battleType = 'plunder'
    source.detail!.winnerSide = 'none'
    source.detail!.primarySide.faction = 'wei'
    source.detail!.primarySide.power = 10000
    source.detail!.primarySide.generals = [{ id: 'caocao', name: '曹操', level: 1 }]
    source.detail!.primarySide.units = [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 }]
    source.detail!.secondarySide!.faction = 'shu'
    source.detail!.secondarySide!.power = 10000
    source.detail!.secondarySide!.generals = [{
      id: 'zhaoyun', name: '赵云', level: 1, traits: [{ traitId: 'longdan_jiuyuan', name: '龙胆救援' }],
    }]
    source.detail!.secondarySide!.units = [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 500, dispatched: 500, lost: 250, survived: 250 }]
    source.detail!.rewards.generalExp = 500
    source.detail!.traits = []
    source.pvpReinforcements = [{
      reinforcementId: 'rein_liubei_miss', fromPlayerId: 'player_liubei', fromPlayerName: '刘备援军', faction: 'shu',
      troops: { shuInfantry: 500 }, generalExpGained: 500, generals: [{ id: 'liubei', name: '刘备', level: 1 }],
    }]
    source.pvpReinforcementLosses = { rein_liubei_miss: { shuInfantry: 250 } }

    const model = toOfficialBattleReport(source)
    expect(source.detail!.secondarySide!.generals?.[0]?.traits?.map((trait) => trait.traitId)).toEqual(['longdan_jiuyuan'])
    expect(model.sides.map((side) => side.role)).toEqual(['attacker', 'defender', 'reinforcement'])
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 10000, generalExp: 500, traits: [], traitText: '' })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 10000, general: { id: 'zhaoyun' }, traits: [], traitText: '' })
    expect(model.sides[2]).toMatchObject({ role: 'reinforcement', general: { id: 'liubei' }, generalExp: 500, traits: [], traitText: '' })
    expect(model.sides[0].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 1000, lost: 500, survived: 500 })
    expect(model.sides[1].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 500, lost: 250, survived: 250 })
    expect(model.sides[2].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 500, lost: 250, survived: 250 })
    expect(model.sides.flatMap((side) => side.traits)).toEqual([])
  })

  it('主城刘备只复活主守军且同兵种援军保持原始阵亡', () => {
    const source = report()
    source.result = 'draw'
    source.battleType = 'plunder'
    source.detail!.result = 'draw'
    source.detail!.battleType = 'plunder'
    source.detail!.winnerSide = 'none'
    source.detail!.primarySide.faction = 'wei'
    source.detail!.primarySide.power = 10000
    source.detail!.primarySide.generals = [{ id: 'caocao', name: '曹操', level: 1 }]
    source.detail!.primarySide.units = [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 }]
    source.detail!.secondarySide!.faction = 'shu'
    source.detail!.secondarySide!.power = 10000
    source.detail!.secondarySide!.generals = [{ id: 'liubei', name: '刘备', level: 1 }]
    source.detail!.secondarySide!.units = [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 500, dispatched: 500, lost: 250, survived: 337 }]
    source.detail!.rewards.generalExp = 500
    source.detail!.traits = [
      {
        traitId: 'renzhu_shouhu', traitName: '仁主守护', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'liubei',
        detail: { effectRate: 0.35, revivedUnits: { shuInfantry: 87 }, triggerChance: 0.6 },
      },
    ]
    source.pvpReinforcements = [{
      reinforcementId: 'rein_guanyu', fromPlayerId: 'player_guanyu', fromPlayerName: '关羽援军', faction: 'shu',
      troops: { shuInfantry: 500 }, generalExpGained: 500, generals: [{ id: 'guanyu', name: '关羽', level: 1 }],
    }]
    source.pvpReinforcementLosses = { rein_guanyu: { shuInfantry: 250 } }

    const model = toOfficialBattleReport(source)
    expect(model.sides.map((side) => side.role)).toEqual(['attacker', 'defender', 'reinforcement'])
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 10000, general: { id: 'caocao' }, generalExp: 500 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 10000, general: { id: 'liubei' } })
    expect(model.sides[2]).toMatchObject({ role: 'reinforcement', general: { id: 'guanyu' }, generalExp: 500 })
    expect(model.sides[0].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 1000, lost: 500, survived: 500 })
    expect(model.sides[1].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 500, lost: 250, survived: 337 })
    expect(model.sides[2].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 500, lost: 250, survived: 250 })
    expect(model.sides[1].traits).toEqual([
      {
        key: 'renzhu_shouhu-0', name: '仁主守护', phase: '进攻/防守/增援战斗结束后',
        detailText: '设计效果比例：35%；复活兵力：蜀步兵 +87；触发概率：60%',
      },
    ])
    expect(model.sides[0].traits).toEqual([])
    expect(model.sides[2].traits).toEqual([])
  })

  it('郭嘉在 PVP 平局时展示真实阵亡和复活', () => {
    for (const [generalId, generalName, traitId, traitName] of [
      ['guojia', '郭嘉', 'guicai_yice', '鬼才遗策'],
    ] as const) {
      const source = report()
      source.result = 'draw'
      source.battleType = 'plunder'
      source.detail!.result = 'draw'
      source.detail!.battleType = 'plunder'
      source.detail!.winnerSide = 'none'
      source.detail!.primarySide.faction = 'wei'
      source.detail!.primarySide.power = 10000
      source.detail!.primarySide.generals = [{
        id: generalId, name: generalName, level: 1,
        traits: [{ traitId, name: traitName }],
      }]
      source.detail!.primarySide.units = [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 610 }]
      source.detail!.secondarySide!.faction = 'shu'
      source.detail!.secondarySide!.power = 10000
      source.detail!.secondarySide!.units = [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 }]
      source.detail!.rewards.generalExp = 500
      source.detail!.traits = [{
        traitId, traitName, ownerSide: 'primary', ownerRole: 'attacker', generalId,
        detail: { effectRate: 0.22, actualLostUnits: { weiInfantry: 500 }, revivedUnits: { weiInfantry: 110 }, totalRevived: 110 },
      }]
      source.pvpReinforcements = []
      source.pvpReinforcementLosses = {}

      const model = toOfficialBattleReport(source)
      expect(source.detail!.primarySide.generals?.[0]?.traits?.[0]?.traitId).toBe(traitId)
      expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 10000, general: { id: generalId }, generalExp: 500 })
      expect(model.sides[0].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 1000, lost: 500, survived: 610 })
      expect(model.sides[1].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 1000, lost: 500, survived: 500 })
      expect(model.sides[0].traits[0]).toMatchObject({
        name: traitName,
        phase: '进攻/防守/增援战斗结束后',
        detailText: '设计效果比例：22%；本场真实阵亡：魏步兵 +500；复活兵力：魏步兵 +110；复活总数：110',
      })
      expect(model.sides[1].traits).toEqual([])
    }
  })

  it('郭嘉在 NPC 与黄巾平局中按所属部队展示复活', () => {
    for (const [generalId, generalName, traitId, traitName] of [
      ['guojia', '郭嘉', 'guicai_yice', '鬼才遗策'],
    ] as const) {
      const npc = report()
      npc.result = 'draw'
      npc.battleType = 'attack'
      npc.detail!.result = 'draw'
      npc.detail!.battleType = 'attack'
      npc.detail!.winnerSide = 'none'
      npc.detail!.primarySide.faction = 'wei'
      npc.detail!.primarySide.power = 1000
      npc.detail!.primarySide.generals = [{ id: generalId, name: generalName, level: 1, traits: [{ traitId, name: traitName }] }]
      npc.detail!.primarySide.units = [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: 100, survived: 22 }]
      npc.detail!.secondarySide!.faction = 'wei'
      npc.detail!.secondarySide!.power = 1000
      npc.detail!.secondarySide!.units = [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: 100, survived: 0 }]
      npc.detail!.rewards.generalExp = 100
      npc.detail!.traits = [{
        traitId, traitName, ownerSide: 'primary', ownerRole: 'attacker', generalId,
        detail: { effectRate: 0.22, actualLostUnits: { weiInfantry: 100 }, revivedUnits: { weiInfantry: 22 }, totalRevived: 22 },
      }]

      const yellow = report()
      yellow.viewType = 'defense'
      yellow.result = 'draw'
      yellow.battleType = 'yellow_turban'
      yellow.detail!.viewType = 'defense'
      yellow.detail!.sourceType = 'yellow_turban'
      yellow.detail!.ownerSide = 'defender'
      yellow.detail!.result = 'draw'
      yellow.detail!.battleType = 'yellow_turban'
      yellow.detail!.winnerSide = 'none'
      yellow.detail!.primarySide = {
        role: 'attacker', faction: 'wei', power: 1030, generals: [],
        units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 103, dispatched: 103, lost: 103, survived: 0 }],
      }
      yellow.detail!.secondarySide = {
        role: 'defender', faction: 'wei', power: 1030,
        generals: [{ id: generalId, name: generalName, level: 1, traits: [{ traitId, name: traitName }] }],
        units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: 100, survived: 22 }],
      }
      yellow.detail!.rewards.generalExp = 103
      yellow.detail!.traits = [{
        traitId, traitName, ownerSide: 'secondary', ownerRole: 'defender', generalId,
        detail: { effectRate: 0.22, actualLostUnits: { weiInfantry: 100 }, revivedUnits: { weiInfantry: 22 }, totalRevived: 22 },
      }]

      const npcModel = toOfficialBattleReport(npc)
      const yellowModel = toOfficialBattleReport(yellow)
      expect(npcModel.sides[0]).toMatchObject({ role: 'attacker', power: 1000, general: { id: generalId }, generalExp: 100 })
      expect(npcModel.sides[0].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 100, lost: 100, survived: 22 })
      expect(npcModel.sides[0].traits[0]).toMatchObject({ name: traitName, phase: '进攻/防守/增援战斗结束后' })
      expect(yellowModel.sides[0]).toMatchObject({ role: 'attacker', power: 1030 })
      expect(yellowModel.sides[1]).toMatchObject({ role: 'defender', power: 1030, general: { id: generalId }, generalExp: 103 })
      expect(yellowModel.sides[0].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 103, lost: 103, survived: 0 })
      expect(yellowModel.sides[1].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 100, lost: 100, survived: 22 })
      expect(yellowModel.sides[1].traits[0]).toMatchObject({ name: traitName, phase: '进攻/防守/增援战斗结束后' })
    }
  })

  it('关羽主动进攻双特性对齐战力、伤亡、返程、经验和真实时间线', () => {
    const source = report()
    source.detail!.primarySide.faction = 'shu'
    source.detail!.primarySide.power = 12000
    source.detail!.primarySide.generals = [{ id: 'guanyu', name: '关羽', level: 1 }]
    source.detail!.primarySide.units = [{ unitType: 'greedyWolf', unitName: '贪狼营', amountBefore: 1000, dispatched: 1000, lost: 436, survived: 564 }]
    source.detail!.secondarySide!.faction = 'wei'
    source.detail!.secondarySide!.power = 6695
    source.detail!.secondarySide!.units = [{ unitType: 'qingZhouArmy', unitName: '青州军', amountBefore: 1000, dispatched: 1000, lost: 1000, survived: 0 }]
    source.detail!.rewards.generalExp = 1000
    source.detail!.traits = [
      {
        traitId: 'shuiyan_qijun', traitName: '水淹七军', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'guanyu',
        detail: { effectRate: 0.35, maxAffectedRate: 0.35, preBattleAffected: { qingZhouArmy: 350 }, triggerChance: 1 },
      },
      {
        traitId: 'wusheng_pojun', traitName: '武圣破军', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'guanyu',
        detail: { attackBonusRate: 0.2, attackModifiedUnits: { greedyWolf: 2 } },
      },
    ]
    const model = toOfficialBattleReport(source)
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 12000, generalExp: 1000 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 6695 })
    expect(model.sides[0].units.find((unit) => unit.key === 'greedyWolf')).toMatchObject({ dispatched: 1000, lost: 436, survived: 564 })
    expect(model.sides[1].units.find((unit) => unit.key === 'qingZhouArmy')).toMatchObject({ dispatched: 1000, lost: 1000, survived: 0 })
    expect(model.sides[0].traits).toEqual([
      {
        key: 'shuiyan_qijun-0', name: '水淹七军', phase: '战斗前',
        detailText: '设计效果比例：35%；设计最大影响比例：35%；战前真实伤亡：青州军 +350；触发概率：100%',
      },
      {
        key: 'wusheng_pojun-1', name: '武圣破军', phase: '主动进攻战斗前',
        detailText: '设计攻击加成：20%；实际攻击修正：贪狼营 +2',
      },
    ])
    expect(model.sides[1].traits).toEqual([])
  })

  it('关羽水淹未命中时武圣加攻仍生效且不产生战前伤亡', () => {
    const source = report()
    source.battleType = 'plunder'
    source.detail!.battleType = 'plunder'
    source.detail!.primarySide.faction = 'shu'
    source.detail!.primarySide.power = 12000
    source.detail!.primarySide.generals = [{
      id: 'guanyu', name: '关羽', level: 1,
      traits: [{ traitId: 'shuiyan_qijun', name: '水淹七军' }, { traitId: 'wusheng_pojun', name: '武圣破军' }],
    }]
    source.detail!.primarySide.units = [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 435, survived: 565 }]
    source.detail!.secondarySide!.faction = 'wei'
    source.detail!.secondarySide!.power = 10000
    source.detail!.secondarySide!.units = [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 564, survived: 436 }]
    source.detail!.rewards.generalExp = 564
    source.detail!.traits = [{
      traitId: 'wusheng_pojun', traitName: '武圣破军', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'guanyu',
      detail: { attackBonusRate: 0.2, attackModifiedUnits: { shuInfantry: 2 } },
    }]
    const model = toOfficialBattleReport(source)
    expect(source.detail!.primarySide.generals![0]!.traits?.map((trait) => trait.traitId)).toEqual(['shuiyan_qijun', 'wusheng_pojun'])
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 12000, generalExp: 564 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 10000 })
    expect(model.sides[0].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 1000, lost: 435, survived: 565 })
    expect(model.sides[1].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 1000, lost: 564, survived: 436 })
    expect(model.sides[0].traits).toEqual([{
      key: 'wusheng_pojun-0', name: '武圣破军', phase: '主动进攻战斗前',
      detailText: '设计攻击加成：20%；实际攻击修正：蜀步兵 +2',
    }])
    expect(model.sides.flatMap((side) => side.traits).some((trait) => trait.name === '水淹七军')).toBe(false)
  })

  it('张飞主动进攻双特性对齐临时压制、真实阵亡、存活和经验', () => {
    const source = report()
    source.detail!.primarySide.faction = 'shu'
    source.detail!.primarySide.power = 12000
    source.detail!.primarySide.generals = [{ id: 'zhangfei', name: '张飞', level: 1 }]
    source.detail!.primarySide.units = [{ unitType: 'greedyWolf', unitName: '贪狼营', amountBefore: 1000, dispatched: 1000, lost: 300, survived: 700 }]
    source.detail!.secondarySide!.faction = 'wei'
    source.detail!.secondarySide!.power = 5150
    source.detail!.secondarySide!.units = [{ unitType: 'qingZhouArmy', unitName: '青州军', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 }]
    source.detail!.rewards.generalExp = 500
    source.detail!.traits = [
      {
        traitId: 'zhenhe_quanjun', traitName: '万人怒吼', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'zhangfei',
        detail: { effectRate: 0.5, maxAffectedRate: 0.5, suppressedUnits: { qingZhouArmy: 500 }, triggerChance: 1 },
      },
      {
        traitId: 'wanren_nuhou', traitName: '勇冠三军', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'zhangfei',
        detail: { attackBonusRate: 0.2, attackModifiedUnits: { greedyWolf: 2 } },
      },
    ]
    const model = toOfficialBattleReport(source)
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 12000, generalExp: 500 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 5150 })
    expect(model.sides[0].units.find((unit) => unit.key === 'greedyWolf')).toMatchObject({ dispatched: 1000, lost: 300, survived: 700 })
    expect(model.sides[1].units.find((unit) => unit.key === 'qingZhouArmy')).toMatchObject({ dispatched: 1000, lost: 500, survived: 500 })
    expect(model.sides[0].traits).toEqual([
      {
      key: 'zhenhe_quanjun-0', name: '万人怒吼', phase: '战斗前',
        detailText: '设计效果比例：50%；设计最大影响比例：50%；本场压制兵力：青州军 +500；触发概率：100%',
      },
      {
      key: 'wanren_nuhou-1', name: '勇冠三军', phase: '主动进攻战斗前',
        detailText: '设计攻击加成：20%；实际攻击修正：贪狼营 +2',
      },
    ])
    expect(model.sides[1].traits).toEqual([])
  })

  it('张飞震慑未命中时步兵加攻仍生效且不产生临时压制', () => {
    const source = report()
    source.battleType = 'attack'
    source.detail!.battleType = 'attack'
    source.detail!.primarySide.faction = 'shu'
    source.detail!.primarySide.power = 12000
    source.detail!.primarySide.generals = [{
      id: 'zhangfei', name: '张飞', level: 1,
      traits: [{ traitId: 'zhenhe_quanjun', name: '万人怒吼' }, { traitId: 'wanren_nuhou', name: '勇冠三军' }],
    }]
    source.detail!.primarySide.units = [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 804, survived: 196 }]
    source.detail!.secondarySide!.faction = 'wei'
    source.detail!.secondarySide!.power = 10300
    source.detail!.secondarySide!.units = [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 1000, survived: 0 }]
    source.detail!.rewards.generalExp = 1000
    source.detail!.traits = [{
      traitId: 'wanren_nuhou', traitName: '勇冠三军', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'zhangfei',
      detail: { attackBonusRate: 0.2, attackModifiedUnits: { shuInfantry: 2 } },
    }]
    const model = toOfficialBattleReport(source)
    expect(source.detail!.primarySide.generals![0]!.traits?.map((trait) => trait.traitId)).toEqual(['zhenhe_quanjun', 'wanren_nuhou'])
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 12000, generalExp: 1000 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 10300 })
    expect(model.sides[0].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 1000, lost: 804, survived: 196 })
    expect(model.sides[1].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 1000, lost: 1000, survived: 0 })
    expect(model.sides[0].traits).toEqual([{
      key: 'wanren_nuhou-0', name: '勇冠三军', phase: '主动进攻战斗前',
      detailText: '设计攻击加成：20%；实际攻击修正：蜀步兵 +2',
    }])
    expect(model.sides.flatMap((side) => side.traits).some((trait) => trait.name === '万人怒吼')).toBe(false)
  })

  it('张辽主动进攻双特性对齐骑兵加攻、溃逃返回、战力和经验', () => {
    const source = report()
    source.detail!.primarySide.faction = 'wei'
    source.detail!.primarySide.power = 19000
    source.detail!.primarySide.generals = [{ id: 'zhangliao', name: '张辽', level: 1 }]
    source.detail!.primarySide.units = [{ unitType: 'weiCavalry', unitName: '魏骑兵', amountBefore: 1000, dispatched: 1000, lost: 199, survived: 801 }]
    source.detail!.secondarySide!.faction = 'shu'
    source.detail!.secondarySide!.power = 6120
    source.detail!.secondarySide!.units = [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 750, survived: 250 }]
    source.detail!.rewards.generalExp = 750
    source.detail!.traits = [
      {
        traitId: 'weizhen_zhenhe', traitName: '震慑全军', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'zhangliao',
        detail: { effectRate: 0.25, suppressedUnits: { shuInfantry: 250 }, fledUnits: { shuInfantry: 250 }, returnedUnits: { shuInfantry: 250 }, triggerChance: 1 },
      },
      {
        traitId: 'weizhen_xiaoyao', traitName: '威震逍遥', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'zhangliao',
        detail: { attackBonusRate: 0.35, attackModifiedUnits: { weiCavalry: 5 } },
      },
    ]
    const model = toOfficialBattleReport(source)
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 19000, generalExp: 750 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 6120 })
    expect(model.sides[0].units.find((unit) => unit.key === 'weiCavalry')).toMatchObject({ dispatched: 1000, lost: 199, survived: 801 })
    expect(model.sides[1].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 1000, lost: 750, survived: 250 })
    expect(model.sides[0].traits).toEqual([
      {
        key: 'weizhen_zhenhe-0', name: '震慑全军', phase: '主动进攻战斗前',
        detailText: '设计效果比例：25%；本场溃逃兵力：蜀步兵 +250；战后返回兵力：蜀步兵 +250；触发概率：100%',
      },
      {
        key: 'weizhen_xiaoyao-1', name: '威震逍遥', phase: '主动进攻战斗前',
        detailText: '设计攻击加成：35%；实际攻击修正：魏骑兵 +5',
      },
    ])
    expect(model.sides[1].traits).toEqual([])
  })

  it('张辽震慑未命中时骑兵加攻仍生效且不产生临时压制', () => {
    const source = report()
    source.battleType = 'attack'
    source.detail!.battleType = 'attack'
    source.detail!.primarySide.faction = 'wei'
    source.detail!.primarySide.power = 19000
    source.detail!.primarySide.generals = [{
      id: 'zhangliao', name: '张辽', level: 1,
      traits: [{ traitId: 'weizhen_zhenhe', name: '震慑全军' }, { traitId: 'weizhen_xiaoyao', name: '威震逍遥' }],
    }]
    source.detail!.primarySide.units = [{ unitType: 'weiCavalry', unitName: '魏骑兵', amountBefore: 1000, dispatched: 1000, lost: 300, survived: 700 }]
    source.detail!.secondarySide!.faction = 'shu'
    source.detail!.secondarySide!.power = 8160
    source.detail!.secondarySide!.units = [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 1000, survived: 0 }]
    source.detail!.rewards.generalExp = 1000
    source.detail!.traits = [{
      traitId: 'weizhen_xiaoyao', traitName: '威震逍遥', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'zhangliao',
      detail: { attackBonusRate: 0.35, attackModifiedUnits: { weiCavalry: 5 } },
    }]
    const model = toOfficialBattleReport(source)
    expect(source.detail!.primarySide.generals![0]!.traits?.map((trait) => trait.traitId)).toEqual(['weizhen_zhenhe', 'weizhen_xiaoyao'])
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 19000, generalExp: 1000 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 8160 })
    expect(model.sides[0].units.find((unit) => unit.key === 'weiCavalry')).toMatchObject({ dispatched: 1000, lost: 300, survived: 700 })
    expect(model.sides[1].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 1000, lost: 1000, survived: 0 })
    expect(model.sides[0].traits).toEqual([{
      key: 'weizhen_xiaoyao-0', name: '威震逍遥', phase: '主动进攻战斗前',
      detailText: '设计攻击加成：35%；实际攻击修正：魏骑兵 +5',
    }])
    expect(model.sides.flatMap((side) => side.traits).some((trait) => trait.name === '震慑全军')).toBe(false)
  })

  it('关羽张辽张飞防守随机未命中时进攻专属加成也不越界触发', () => {
    const cases = [
      { generalId: 'guanyu', generalName: '关羽', traitIds: ['shuiyan_qijun', 'wusheng_pojun'], attackerFaction: 'wei', attackerUnit: 'weiInfantry', attackerName: '魏步兵', defenderFaction: 'shu', defenderUnit: 'shuInfantry', defenderName: '蜀步兵' },
      { generalId: 'zhangliao', generalName: '张辽', traitIds: ['weizhen_zhenhe', 'weizhen_xiaoyao'], attackerFaction: 'shu', attackerUnit: 'shuInfantry', attackerName: '蜀步兵', defenderFaction: 'wei', defenderUnit: 'weiInfantry', defenderName: '魏步兵' },
      { generalId: 'zhangfei', generalName: '张飞', traitIds: ['zhenhe_quanjun', 'wanren_nuhou'], attackerFaction: 'wei', attackerUnit: 'weiInfantry', attackerName: '魏步兵', defenderFaction: 'shu', defenderUnit: 'shuInfantry', defenderName: '蜀步兵' },
    ]
    for (const tc of cases) {
      const source = report()
      source.viewType = 'defense'
      source.result = 'draw'
      source.battleType = 'plunder'
      source.detail!.viewType = 'defense'
      source.detail!.result = 'draw'
      source.detail!.winnerSide = 'none'
      source.detail!.ownerSide = 'defender'
      source.detail!.battleType = 'plunder'
      source.detail!.primarySide.faction = tc.attackerFaction
      source.detail!.primarySide.power = 10000
      source.detail!.primarySide.generals = []
      source.detail!.primarySide.units = [{ unitType: tc.attackerUnit, unitName: tc.attackerName, amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 }]
      source.detail!.secondarySide!.faction = tc.defenderFaction
      source.detail!.secondarySide!.power = 10000
      source.detail!.secondarySide!.generals = [{
        id: tc.generalId, name: tc.generalName, level: 1,
        traits: tc.traitIds.map((traitId) => ({ traitId, name: traitLabel(traitId) })),
      }]
      source.detail!.secondarySide!.units = [{ unitType: tc.defenderUnit, unitName: tc.defenderName, amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 }]
      source.detail!.rewards.generalExp = 500
      source.detail!.traits = []
      const model = toOfficialBattleReport(source)
      expect(source.detail!.secondarySide!.generals![0]!.traits?.map((trait) => trait.traitId)).toEqual(tc.traitIds)
      expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 10000, generalExp: null, result: 'unknown', traits: [], traitText: '' })
      expect(model.sides[1]).toMatchObject({ role: 'defender', power: 10000, generalExp: 500, result: 'unknown', traits: [], traitText: '' })
      expect(model.sides[0].units.find((unit) => unit.key === tc.attackerUnit)).toMatchObject({ dispatched: 1000, lost: 500, survived: 500 })
      expect(model.sides[1].units.find((unit) => unit.key === tc.defenderUnit)).toMatchObject({ dispatched: 1000, lost: 500, survived: 500 })
      expect(model.sides.flatMap((side) => side.traits)).toEqual([])
    }
  })

  it('许褚虎痴命中与虎虎生威永久被动按不同区域展示', () => {
    const source = report()
    source.detail!.primarySide.faction = 'wei'
    source.detail!.primarySide.power = 8400
    source.detail!.primarySide.generals = [{
      id: 'xuchu', name: '许褚', level: 1,
      traits: [
        { traitId: 'huchi_chongzhen', name: '虎痴冲阵', params: { triggerChance: 0.5, enemyDefenseReductionRate: 0.3 } },
        { traitId: 'huhu_shengwei', name: '虎虎生威', params: { unitAttackFlat: 12, unitSpeedFlat: 5 } },
      ],
    }]
    source.detail!.primarySide.units = [{ unitType: 'huBaoQi', unitName: '虎豹骑', amountBefore: 200, dispatched: 200, lost: 100, survived: 100 }]
    source.detail!.secondarySide!.faction = 'shu'
    source.detail!.secondarySide!.power = 7000
    source.detail!.secondarySide!.units = [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 600, survived: 400 }]
    source.detail!.traits = [{
      traitId: 'huchi_chongzhen', traitName: '虎痴冲阵', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'xuchu',
      detail: { enemyDefenseReductionRate: 0.3, infantryDefenseModifiedUnits: { shuInfantry: -3 }, cavalryDefenseModifiedUnits: { shuInfantry: -2 }, triggerChance: 0.5 },
    }]
    const model = toOfficialBattleReport(source)
    expect(model.sides[0].traits).toEqual([{
      key: 'huchi_chongzhen-0', name: '虎痴冲阵', phase: '主动进攻战斗前',
      detailText: '设计敌方防御降低：30%；实际步防修正：蜀步兵 -3；实际骑防修正：蜀步兵 -2；触发概率：50%',
    }])
    expect(model.sides[0].passiveTraits).toEqual([{
      key: 'passive-xuchu-huhu_shengwei', name: '虎虎生威', phase: '永久被动', detailText: '虎豹骑攻击 +12；移动 +5',
    }])
    expect(model.sides[1].traits).toEqual([])
    expect(model.sides[1].passiveTraits).toEqual([])
  })

  it('许褚虎痴合法未命中时不补造触发，虎虎生威仍按快照展示', () => {
    const source = report()
    source.detail!.primarySide.generals = [{
      id: 'xuchu', name: '许褚', level: 1,
      traits: [
        { traitId: 'huchi_chongzhen', name: '虎痴冲阵', params: { triggerChance: 0.5, enemyDefenseReductionRate: 0.3 } },
        { traitId: 'huhu_shengwei', name: '虎虎生威', params: { unitAttackFlat: 12, unitSpeedFlat: 5 } },
      ],
    }]
    source.detail!.traits = []
    const model = toOfficialBattleReport(source)
    expect(model.sides[0].traits).toEqual([])
    expect(model.sides[0].passiveTraits).toEqual([{
      key: 'passive-xuchu-huhu_shengwei', name: '虎虎生威', phase: '永久被动', detailText: '虎豹骑攻击 +12；移动 +5',
    }])
  })


  it('典韦主动进攻只展示死战到底的战前攻击修正', () => {
    const source = report()
    source.detail!.primarySide.faction = 'wei'
    source.detail!.primarySide.power = 1400
    source.detail!.primarySide.generals = [{ id: 'dianwei', name: '典韦', level: 1 }]
    source.detail!.primarySide.units = [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: 94, survived: 6 }]
    source.detail!.secondarySide!.faction = 'shu'
    source.detail!.secondarySide!.power = 10000
    source.detail!.secondarySide!.units = [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 57, survived: 943 }]
    source.detail!.rewards.generalExp = 57
    source.detail!.traits = [
      {
        traitId: 'sizhandaodi', traitName: '死战到底', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'dianwei',
        detail: { attackBonusRate: 0.35, attackModifiedUnits: { weiInfantry: 4 } },
      },
    ]
    const model = toOfficialBattleReport(source)
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 1400, generalExp: 57 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 10000 })
    expect(model.sides[0].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 100, lost: 94, survived: 6 })
    expect(model.sides[1].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 1000, lost: 57, survived: 943 })
    expect(model.sides[0].traits).toEqual([
      {
        key: 'sizhandaodi-0', name: '死战到底', phase: '主动进攻战斗前',
        detailText: '设计攻击加成：35%；实际攻击修正：魏步兵 +4',
      },
    ])
    expect(model.sides[1].traits).toEqual([])
  })

  it('黄忠掠夺战双特性区分战前破防、核心战损和战后追加扣兵', () => {
    const source = report()
    source.battleType = 'plunder'
    source.detail!.battleType = 'plunder'
    source.detail!.primarySide.faction = 'shu'
    source.detail!.primarySide.power = 10000
    source.detail!.primarySide.generals = [{ id: 'huangzhong', name: '黄忠', level: 1 }]
    source.detail!.primarySide.units = [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 421, survived: 579 }]
    source.detail!.secondarySide!.faction = 'wei'
    source.detail!.secondarySide!.power = 8000
    source.detail!.secondarySide!.units = [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 678, survived: 322 }]
    source.detail!.rewards.generalExp = 678
    source.detail!.traits = [
      {
        traitId: 'baibu_chuanyang', traitName: '百步穿杨', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'huangzhong',
        detail: { enemyDefenseReductionRate: 0.2, infantryDefenseModifiedUnits: { weiInfantry: -2 }, cavalryDefenseModifiedUnits: { weiInfantry: -2 }, triggerChance: 1 },
      },
      {
        traitId: 'laodang_yizhuang', traitName: '老当益壮', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'huangzhong',
        detail: { effectRate: 0.1, extraLosses: { weiInfantry: 100 } },
      },
    ]
    const model = toOfficialBattleReport(source)
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 10000, generalExp: 678 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 8000 })
    expect(model.sides[0].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 1000, lost: 421, survived: 579 })
    expect(model.sides[1].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 1000, lost: 678, survived: 322 })
    expect(model.sides[0].traits).toEqual([
      {
        key: 'baibu_chuanyang-0', name: '百步穿杨', phase: '主动进攻战斗前',
        detailText: '设计敌方防御降低：20%；实际步防修正：魏步兵 -2；实际骑防修正：魏步兵 -2；触发概率：100%',
      },
      {
        key: 'laodang_yizhuang-1', name: '老当益壮', phase: '战斗结算后',
        detailText: '设计效果比例：10%；追加损失：魏步兵 +100',
      },
    ])
    expect(model.sides[1].traits).toEqual([])
  })

  it('黄忠百步未命中时老当益壮仍按核心剩余兵力追加真实损失', () => {
    const source = report()
    source.battleType = 'plunder'
    source.detail!.battleType = 'plunder'
    source.detail!.primarySide.faction = 'shu'
    source.detail!.primarySide.power = 10000
    source.detail!.primarySide.generals = [{
      id: 'huangzhong', name: '黄忠', level: 1,
      traits: [{ traitId: 'baibu_chuanyang', name: '百步穿杨' }, { traitId: 'laodang_yizhuang', name: '老当益壮' }],
    }]
    source.detail!.primarySide.units = [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 }]
    source.detail!.secondarySide!.faction = 'wei'
    source.detail!.secondarySide!.power = 10000
    source.detail!.secondarySide!.units = [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 600, survived: 400 }]
    source.detail!.rewards.generalExp = 600
    source.detail!.traits = [{
      traitId: 'laodang_yizhuang', traitName: '老当益壮', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'huangzhong',
      detail: { effectRate: 0.1, extraLosses: { weiInfantry: 100 } },
    }]
    const model = toOfficialBattleReport(source)
    expect(source.detail!.primarySide.generals![0]!.traits?.map((trait) => trait.traitId)).toEqual(['baibu_chuanyang', 'laodang_yizhuang'])
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 10000, generalExp: 600 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 10000 })
    expect(model.sides[0].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 1000, lost: 500, survived: 500 })
    expect(model.sides[1].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 1000, lost: 600, survived: 400 })
    expect(model.sides[0].traits).toEqual([{
      key: 'laodang_yizhuang-0', name: '老当益壮', phase: '战斗结算后',
      detailText: '设计效果比例：10%；追加损失：魏步兵 +100',
    }])
    expect(model.sides.flatMap((side) => side.traits).some((trait) => trait.name === '百步穿杨')).toBe(false)
  })

  it('黄忠战后伤害完整汇总同兵种主守军与援军且三方数值守恒', () => {
    const source = report()
    source.battleType = 'plunder'
    source.detail!.battleType = 'plunder'
    source.detail!.result = 'draw'
    source.detail!.winnerSide = 'none'
    source.detail!.primarySide.faction = 'shu'
    source.detail!.primarySide.power = 10000
    source.detail!.primarySide.generals = [{ id: 'huangzhong', name: '黄忠', level: 1 }]
    source.detail!.primarySide.units = [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 }]
    source.detail!.secondarySide!.faction = 'wu'
    source.detail!.secondarySide!.power = 10000
    source.detail!.secondarySide!.generals = [{ id: 'sunquan', name: '孙权', level: 1 }]
    source.detail!.secondarySide!.units = [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 500, dispatched: 500, lost: 300, survived: 200 }]
    source.detail!.rewards.generalExp = 600
    source.detail!.traits = [{
      traitId: 'laodang_yizhuang', traitName: '老当益壮', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'huangzhong',
      detail: { effectRate: 0.1, extraLosses: { wuInfantry: 100 }, triggerChance: 1 },
    }]
    source.pvpReinforcements = [{
      reinforcementId: 'rein_taishici', fromPlayerId: 'player_taishici', fromPlayerName: '太史慈援军', faction: 'wu',
      troops: { wuInfantry: 500 }, generalExpGained: 500,
      generals: [{ id: 'taishici', name: '太史慈', level: 1 }],
    }]
    source.pvpReinforcementLosses = { rein_taishici: { wuInfantry: 300 } }

    const model = toOfficialBattleReport(source)
    expect(model.sides.map((side) => side.role)).toEqual(['attacker', 'defender', 'reinforcement'])
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 10000, general: { id: 'huangzhong' }, generalExp: 600 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 10000, general: { id: 'sunquan' } })
    expect(model.sides[2]).toMatchObject({ role: 'reinforcement', general: { id: 'taishici' }, generalExp: 500 })
    expect(model.sides[0].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 1000, lost: 500, survived: 500 })
    expect(model.sides[1].units.find((unit) => unit.key === 'wuInfantry')).toMatchObject({ dispatched: 500, lost: 300, survived: 200 })
    expect(model.sides[2].units.find((unit) => unit.key === 'wuInfantry')).toMatchObject({ dispatched: 500, lost: 300, survived: 200 })
    expect(model.sides[0].traits).toEqual([{
      key: 'laodang_yizhuang-0', name: '老当益壮', phase: '战斗结算后',
      detailText: '设计效果比例：10%；追加损失：吴步兵 +100；触发概率：100%',
    }])
    expect(model.sides.slice(1).flatMap((side) => side.traits)).toEqual([])
  })

  it('孙策掠夺战双特性只强化霸王骑并在胜利后追加真实追击损失', () => {
    const source = report()
    source.battleType = 'plunder'
    source.detail!.battleType = 'plunder'
    source.detail!.primarySide.faction = 'wu'
    source.detail!.primarySide.power = 15600
    source.detail!.primarySide.generals = [{ id: 'sunce', name: '孙策', level: 1 }]
    source.detail!.primarySide.units = [{ unitType: 'overlordRider', unitName: '霸王骑', amountBefore: 200, dispatched: 200, lost: 55, survived: 145 }]
    source.detail!.secondarySide!.faction = 'wei'
    source.detail!.secondarySide!.power = 8000
    source.detail!.secondarySide!.units = [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 821, survived: 179 }]
    source.detail!.rewards.generalExp = 821
    source.detail!.rewards.resources = { wood: 10000 }
    source.detail!.traits = [
      {
        traitId: 'xiaobawang_tieqi', traitName: '小霸王', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'sunce',
        detail: { unitAttackFlat: 50, attackModifiedUnits: { overlordRider: 50 } },
      },
      {
        traitId: 'xiaobawang_zhuiji', traitName: '小霸王追击', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'sunce',
        detail: { effectRate: 0.1, extraLosses: { weiInfantry: 100 }, triggerChance: 1 },
      },
    ]
    const model = toOfficialBattleReport(source)
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 15600, generalExp: 821 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 8000 })
    expect(model.sides[0].units.find((unit) => unit.key === 'overlordRider')).toMatchObject({ dispatched: 200, lost: 55, survived: 145 })
    expect(model.sides[1].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 1000, lost: 821, survived: 179 })
    expect(model.resourceText).toContain('木材:10,000')
    expect(model.sides[0].traits).toEqual([
      {
        key: 'xiaobawang_tieqi-0', name: '小霸王', phase: '主动进攻战斗前',
        detailText: '设计单位攻击增加：50；实际攻击修正：霸王骑 +50',
      },
      {
        key: 'xiaobawang_zhuiji-1', name: '小霸王追击', phase: '掠夺战结算后',
        detailText: '设计效果比例：10%；追加损失：魏步兵 +100；触发概率：100%',
      },
    ])
    expect(model.sides[1].traits).toEqual([])
  })

  it('孙策掠夺获胜但追击未命中时只展示铁骑加攻和核心兵损', () => {
    const source = report()
    source.battleType = 'plunder'
    source.detail!.battleType = 'plunder'
    source.detail!.primarySide.faction = 'wu'
    source.detail!.primarySide.power = 15600
    source.detail!.primarySide.generals = [{
      id: 'sunce', name: '孙策', level: 1,
      traits: [{ traitId: 'xiaobawang_zhuiji', name: '小霸王追击' }, { traitId: 'xiaobawang_tieqi', name: '小霸王' }],
    }]
    source.detail!.primarySide.units = [{ unitType: 'overlordRider', unitName: '霸王骑', amountBefore: 200, dispatched: 200, lost: 55, survived: 145 }]
    source.detail!.secondarySide!.faction = 'wei'
    source.detail!.secondarySide!.power = 8000
    source.detail!.secondarySide!.units = [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 721, survived: 279 }]
    source.detail!.rewards.generalExp = 721
    source.detail!.rewards.resources = { wood: 10000 }
    source.detail!.traits = [{
      traitId: 'xiaobawang_tieqi', traitName: '小霸王', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'sunce',
      detail: { unitAttackFlat: 50, attackModifiedUnits: { overlordRider: 50 } },
    }]
    const model = toOfficialBattleReport(source)
    expect(source.detail!.primarySide.generals![0]!.traits?.map((trait) => trait.traitId)).toEqual(['xiaobawang_zhuiji', 'xiaobawang_tieqi'])
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 15600, generalExp: 721 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 8000 })
    expect(model.sides[0].units.find((unit) => unit.key === 'overlordRider')).toMatchObject({ dispatched: 200, lost: 55, survived: 145 })
    expect(model.sides[1].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 1000, lost: 721, survived: 279 })
    expect(model.sides[0].traits).toEqual([{
      key: 'xiaobawang_tieqi-0', name: '小霸王', phase: '主动进攻战斗前',
      detailText: '设计单位攻击增加：50；实际攻击修正：霸王骑 +50',
    }])
    expect(model.sides.flatMap((side) => side.traits).some((trait) => trait.name === '小霸王追击')).toBe(false)
  })

  it('马超孙策防守随机条件成立但未命中时只展示核心结算', () => {
    const cases = [
      {
        result: 'attacker_victory', winnerSide: 'attacker', battleType: 'attack', attackerFaction: 'wu', defenderFaction: 'shu',
        general: { id: 'machao', name: '马超', level: 1, stats: { force: 0 }, effectiveStats: { force: 20 }, buffs: { attackBonus: 0.4 }, traits: [{ traitId: 'xiliang_tuji', name: '西凉突击' }, { traitId: 'tianshen_xiafan', name: '天神下凡' }] },
        attackerPower: 20000, attackerUnits: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1000, dispatched: 1000, lost: 12, survived: 988 }, { unitType: 'wuCavalry', unitName: '吴骑兵', amountBefore: 1000, dispatched: 1000, lost: 12, survived: 988 }],
        defenderPower: 900, defenderUnit: { unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 100, dispatched: 100, lost: 100, survived: 0 }, defenderExp: 24,
      },
      {
        result: 'defender_victory', winnerSide: 'defender', battleType: 'plunder', attackerFaction: 'wei', defenderFaction: 'wu',
        general: { id: 'sunce', name: '孙策', level: 1, traits: [{ traitId: 'xiaobawang_zhuiji', name: '小霸王追击' }, { traitId: 'xiaobawang_tieqi', name: '小霸王' }] },
        attackerPower: 1000, attackerUnits: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: 72, survived: 28 }],
        defenderPower: 2000, defenderUnit: { unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 200, dispatched: 200, lost: 54, survived: 146 }, defenderExp: 72,
      },
    ]
    for (const tc of cases) {
      const source = report()
      source.viewType = 'defense'
      source.result = tc.result
      source.battleType = tc.battleType
      source.detail!.viewType = 'defense'
      source.detail!.result = tc.result
      source.detail!.winnerSide = tc.winnerSide
      source.detail!.ownerSide = 'defender'
      source.detail!.battleType = tc.battleType
      source.detail!.primarySide.faction = tc.attackerFaction
      source.detail!.primarySide.power = tc.attackerPower
      source.detail!.primarySide.generals = []
      source.detail!.primarySide.units = tc.attackerUnits
      source.detail!.secondarySide!.faction = tc.defenderFaction
      source.detail!.secondarySide!.power = tc.defenderPower
      source.detail!.secondarySide!.generals = [tc.general]
      source.detail!.secondarySide!.units = [tc.defenderUnit]
      source.detail!.rewards.generalExp = tc.defenderExp
      source.detail!.rewards.resources = {}
      source.detail!.traits = []
      const model = toOfficialBattleReport(source)
      expect(source.detail!.secondarySide!.generals![0]!.traits?.map((trait) => trait.traitId)).toEqual(tc.general.traits.map((trait) => trait.traitId))
      expect(model.sides[0]).toMatchObject({ role: 'attacker', power: tc.attackerPower, generalExp: null, traits: [], traitText: '' })
      expect(model.sides[1]).toMatchObject({ role: 'defender', power: tc.defenderPower, generalExp: tc.defenderExp, traits: [], traitText: '' })
      for (const unit of tc.attackerUnits) expect(model.sides[0].units.find((item) => item.key === unit.unitType)).toMatchObject({ dispatched: unit.dispatched, lost: unit.lost, survived: unit.survived })
      expect(model.sides[1].units.find((unit) => unit.key === tc.defenderUnit.unitType)).toMatchObject({ dispatched: tc.defenderUnit.dispatched, lost: tc.defenderUnit.lost, survived: tc.defenderUnit.survived })
      expect(model.sides.flatMap((side) => side.traits)).toEqual([])
      if (tc.general.id === 'machao') expect(model.sides[1].general).toMatchObject({ stats: { force: 0 }, effectiveStats: { force: 20 }, buffs: { attackBonus: 0.4 } })
    }
  })

  it('孙策掠夺平局时拥有追击但不追加伤亡或生成时间线', () => {
    const source = report()
    source.result = 'draw'
    source.battleType = 'plunder'
    source.detail!.result = 'draw'
    source.detail!.battleType = 'plunder'
    source.detail!.winnerSide = 'none'
    source.detail!.primarySide.faction = 'wu'
    source.detail!.primarySide.power = 1000
    source.detail!.primarySide.generals = [{
      id: 'sunce', name: '孙策', level: 1,
      traits: [{ traitId: 'xiaobawang_zhuiji', name: '小霸王追击', requiredOutcome: 'win' }, { traitId: 'xiaobawang_tieqi', name: '小霸王' }],
    }]
    source.detail!.primarySide.units = [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 100, dispatched: 100, lost: 50, survived: 50 }]
    source.detail!.secondarySide!.faction = 'shu'
    source.detail!.secondarySide!.power = 1000
    source.detail!.secondarySide!.units = [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 100, dispatched: 100, lost: 50, survived: 50 }]
    source.detail!.rewards.generalExp = 50
    source.detail!.rewards.resources = {}
    source.detail!.traits = []
    source.pvpReinforcements = []
    source.pvpReinforcementLosses = {}

    const model = toOfficialBattleReport(source)
    expect(source.detail!.primarySide.generals![0]!.traits?.map((trait) => trait.traitId)).toEqual(['xiaobawang_zhuiji', 'xiaobawang_tieqi'])
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 1000, general: { id: 'sunce' }, generalExp: 50 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 1000 })
    expect(model.sides[0].units.find((unit) => unit.key === 'wuInfantry')).toMatchObject({ dispatched: 100, lost: 50, survived: 50 })
    expect(model.sides[1].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 100, lost: 50, survived: 50 })
    expect(model.sides[0].traits).toEqual([])
    expect(model.sides[1].traits).toEqual([])
  })

  it('孙策掠夺守城获胜后追击来袭方并归属防守将领', () => {
    const source = report()
    source.viewType = 'defense'
    source.result = 'defender_victory'
    source.battleType = 'plunder'
    source.detail!.viewType = 'defense'
    source.detail!.result = 'defender_victory'
    source.detail!.winnerSide = 'defender'
    source.detail!.ownerSide = 'defender'
    source.detail!.battleType = 'plunder'
    source.detail!.primarySide = {
      role: 'attacker', faction: 'wei', power: 1000, generals: [],
      units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: 100, survived: 0 }],
    }
    source.detail!.secondarySide = {
      role: 'defender', faction: 'wu', power: 2000,
      generals: [{ id: 'sunce', name: '孙策', level: 1, traits: [{ traitId: 'xiaobawang_zhuiji', name: '小霸王追击', requiredOutcome: 'win' }] }],
      units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 200, dispatched: 200, lost: 54, survived: 146 }],
    }
    source.detail!.rewards = { generalExp: 100, generalLevelBefore: 1, generalLevelAfter: 1 }
    source.detail!.traits = [{
      traitId: 'xiaobawang_zhuiji', traitName: '小霸王追击', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'sunce',
      detail: { effectRate: 0.5, extraLosses: { weiInfantry: 28 }, triggerChance: 1 },
    }]

    const model = toOfficialBattleReport(source)
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 1000 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 2000, general: { id: 'sunce' }, generalExp: 100 })
    expect(model.sides[0].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 100, lost: 100, survived: 0 })
    expect(model.sides[1].units.find((unit) => unit.key === 'wuInfantry')).toMatchObject({ dispatched: 200, lost: 54, survived: 146 })
    expect(model.sides[0].traits).toEqual([])
    expect(model.sides[1].traits).toEqual([{
      key: 'xiaobawang_zhuiji-0', name: '小霸王追击', phase: '掠夺战结算后',
      detailText: '设计效果比例：50%；追加损失：魏步兵 +28；触发概率：100%',
    }])
  })

  it('孙策援军只在掠夺守方获胜且命中时展示真实追击', () => {
    const buildModel = (triggered: boolean) => {
      const source = report()
      source.result = 'defender_victory'
      source.battleType = 'plunder'
      source.detail!.result = 'defender_victory'
      source.detail!.battleType = 'plunder'
      source.detail!.winnerSide = 'defender'
      source.detail!.primarySide = {
        role: 'attacker', faction: 'wei', power: 1000, generals: [{ id: 'caocao', name: '曹操', level: 1 }],
        units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: triggered ? 82 : 72, survived: triggered ? 18 : 28 }],
      }
      source.detail!.secondarySide = {
        role: 'defender', faction: 'shu', power: 2000, generals: [{ id: 'liubei', name: '刘备', level: 1 }],
        units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1, dispatched: 1, lost: 0, survived: 1 }],
      }
      source.detail!.rewards = { generalExp: 54 }
      source.detail!.traits = triggered ? [{
        traitId: 'xiaobawang_zhuiji', traitName: '小霸王追击', ownerSide: 'reinforcement', ownerRole: 'reinforcement', ownerPlayerId: 'helper_sunce', generalId: 'sunce',
        detail: { effectRate: 0.1, extraLosses: { weiInfantry: 10 }, triggerChance: 1 },
      }] : []
      source.pvpReinforcements = [{
        reinforcementId: 'rein_sunce', fromPlayerId: 'helper_sunce', fromPlayerName: '孙策援军', faction: 'wu',
        troops: { wuInfantry: 199 }, generalExpGained: triggered ? 82 : 72,
        generals: [{ id: 'sunce', name: '孙策', level: 1, traits: [{ traitId: 'xiaobawang_zhuiji', name: '小霸王追击', requiredOutcome: 'win' }] }],
      }]
      source.pvpReinforcementLosses = { rein_sunce: { wuInfantry: 54 } }
      return { source, model: toOfficialBattleReport(source) }
    }
    const hit = buildModel(true)
    const miss = buildModel(false)

    expect(hit.source.pvpReinforcements![0]!.generals?.[0]?.traits?.map((trait) => trait.traitId)).toEqual(['xiaobawang_zhuiji'])
    expect(miss.source.pvpReinforcements![0]!.generals?.[0]?.traits?.map((trait) => trait.traitId)).toEqual(['xiaobawang_zhuiji'])
    expect(hit.model.sides.map((side) => side.role)).toEqual(['attacker', 'defender', 'reinforcement'])
    expect(hit.model.sides[0]).toMatchObject({ role: 'attacker', power: 1000, generalExp: 54 })
    expect(hit.model.sides[1]).toMatchObject({ role: 'defender', power: 2000 })
    expect(hit.model.sides[2]).toMatchObject({ role: 'reinforcement', general: { id: 'sunce', name: '孙策', level: 1 }, generalExp: 82 })
    expect(hit.model.sides[0].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 100, lost: 82, survived: 18 })
    expect(miss.model.sides[0].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 100, lost: 72, survived: 28 })
    expect(hit.model.sides[1].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 1, lost: 0, survived: 1 })
    expect(hit.model.sides[2].units.find((unit) => unit.key === 'wuInfantry')).toMatchObject({ dispatched: 199, lost: 54, survived: 145 })
    expect(miss.model.sides[2]).toMatchObject({ role: 'reinforcement', generalExp: 72, traits: [], traitText: '' })
    expect(hit.model.sides[2].traits).toEqual([{
      key: 'xiaobawang_zhuiji-helper_sunce-0', name: '小霸王追击', phase: '掠夺战结算后',
      detailText: '设计效果比例：10%；追加损失：魏步兵 +10；触发概率：100%',
    }])
    expect(hit.model.sides.slice(0, 2).flatMap((side) => side.traits)).toEqual([])
    expect(miss.model.sides.flatMap((side) => side.traits)).toEqual([])
  })

  it('陆逊援军火烧命中或未命中时只展示实际生效的追加伤害', () => {
    const buildModel = (fireTriggered: boolean) => {
      const source = report()
      source.result = 'draw'
      source.battleType = 'plunder'
      source.detail!.result = 'draw'
      source.detail!.battleType = 'plunder'
      source.detail!.winnerSide = 'none'
      source.detail!.primarySide = {
        role: 'attacker', faction: 'wei', power: 1000, generals: [{ id: 'caocao', name: '曹操', level: 1 }],
        units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: fireTriggered ? 100 : 60, survived: fireTriggered ? 0 : 40 }],
      }
      source.detail!.secondarySide = {
        role: 'defender', faction: 'shu', power: 1000, generals: [{ id: 'liubei', name: '刘备', level: 1 }],
        units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1, dispatched: 1, lost: 1, survived: 0 }],
      }
      source.detail!.rewards = { generalExp: 50 }
      source.detail!.traits = [{
        traitId: fireTriggered ? 'huoshao_lianying' : 'lianying_zengshang', traitName: fireTriggered ? '火烧联营' : '连营增伤',
        ownerSide: 'reinforcement', ownerRole: 'reinforcement', ownerPlayerId: 'helper_luxun', generalId: 'luxun',
        detail: fireTriggered
          ? { effectRate: 1, maxAffectedRate: 1, targetExtraLosses: { weiInfantry: 50 }, triggerChance: 1 }
          : { effectRate: 0.1, targetExtraLosses: { weiInfantry: 10 } },
      }]
      source.pvpReinforcements = [{
        reinforcementId: 'rein_luxun', fromPlayerId: 'helper_luxun', fromPlayerName: '陆逊援军', faction: 'wu',
        troops: { wuInfantry: 99 }, generalExpGained: fireTriggered ? 100 : 60,
        generals: [{
          id: 'luxun', name: '陆逊', level: 1,
          traits: [{ traitId: 'huoshao_lianying', name: '火烧联营' }, { traitId: 'lianying_zengshang', name: '连营增伤' }],
        }],
      }]
      source.pvpReinforcementLosses = { rein_luxun: { wuInfantry: 49 } }
      return { source, model: toOfficialBattleReport(source) }
    }
    const hit = buildModel(true)
    const miss = buildModel(false)

    for (const current of [hit, miss]) {
      expect(current.source.pvpReinforcements![0]!.generals?.[0]?.traits?.map((trait) => trait.traitId)).toEqual(['huoshao_lianying', 'lianying_zengshang'])
      expect(current.model.sides.map((side) => side.role)).toEqual(['attacker', 'defender', 'reinforcement'])
      expect(current.model.sides[0]).toMatchObject({ role: 'attacker', power: 1000, generalExp: 50 })
      expect(current.model.sides[1]).toMatchObject({ role: 'defender', power: 1000 })
      expect(current.model.sides[1].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 1, lost: 1, survived: 0 })
      expect(current.model.sides[2].units.find((unit) => unit.key === 'wuInfantry')).toMatchObject({ dispatched: 99, lost: 49, survived: 50 })
      expect(current.model.sides.slice(0, 2).flatMap((side) => side.traits)).toEqual([])
    }
    expect(hit.model.sides[0].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 100, lost: 100, survived: 0 })
    expect(miss.model.sides[0].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 100, lost: 60, survived: 40 })
    expect(hit.model.sides[2]).toMatchObject({ role: 'reinforcement', general: { id: 'luxun', name: '陆逊', level: 1 }, generalExp: 100 })
    expect(miss.model.sides[2]).toMatchObject({ role: 'reinforcement', general: { id: 'luxun', name: '陆逊', level: 1 }, generalExp: 60 })
    expect(hit.model.sides[2].traits).toEqual([{
      key: 'huoshao_lianying-helper_luxun-0', name: '火烧联营', phase: '战斗结算后',
      detailText: '设计效果比例：100%；设计最大影响比例：100%；目标兵种追加损失：魏步兵 +50；触发概率：100%',
    }])
    expect(miss.model.sides[2].traits).toEqual([{
      key: 'lianying_zengshang-helper_luxun-0', name: '连营增伤', phase: '战斗结算后',
      detailText: '设计效果比例：10%；目标兵种追加损失：魏步兵 +10',
    }])
  })

  it('司马懿援军双特性同时命中或均未命中时准确展示', () => {
    const buildModel = (triggered: boolean) => {
      const source = report()
      source.result = triggered ? 'defender_victory' : 'draw'
      source.battleType = 'plunder'
      source.detail!.result = triggered ? 'defender_victory' : 'draw'
      source.detail!.battleType = 'plunder'
      source.detail!.winnerSide = triggered ? 'defender' : 'none'
      source.detail!.primarySide = {
        role: 'attacker', faction: 'shu', power: triggered ? 650 : 1000, generals: [{ id: 'liubei', name: '刘备', level: 1 }],
        units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 100, dispatched: 100, lost: triggered ? 77 : 50, survived: triggered ? 23 : 50 }],
      }
      source.detail!.secondarySide = {
        role: 'defender', faction: 'wu', power: triggered ? 1396 : 1000, generals: [{ id: 'sunquan', name: '孙权', level: 1 }],
        units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1, dispatched: 1, lost: triggered ? 0 : 1, survived: triggered ? 1 : 0 }],
      }
      source.detail!.rewards = { generalExp: triggered ? 35 : 50 }
      source.detail!.traits = triggered ? [
        {
          traitId: 'yibing_touxi', traitName: '疑兵偷袭', ownerSide: 'reinforcement', ownerRole: 'reinforcement', ownerPlayerId: 'helper_simayi', generalId: 'simayi',
          detail: { effectRate: 0.35, preBattleAffected: { shuInfantry: 35 }, triggerChance: 1 },
        },
        {
          traitId: 'mouding_houfa', traitName: '谋定后发', ownerSide: 'reinforcement', ownerRole: 'reinforcement', ownerPlayerId: 'helper_simayi', generalId: 'simayi',
          detail: { defenseBonusRate: 0.35, infantryDefenseModifiedUnits: { weiInfantry: 4 }, cavalryDefenseModifiedUnits: { weiInfantry: 3 }, triggerChance: 1 },
        },
      ] : []
      source.pvpReinforcements = [{
        reinforcementId: 'rein_simayi', fromPlayerId: 'helper_simayi', fromPlayerName: '司马懿援军', faction: 'wei',
        troops: { weiInfantry: 99 }, generalExpGained: triggered ? 77 : 50,
        generals: [{
          id: 'simayi', name: '司马懿', level: 1,
          traits: [{ traitId: 'yibing_touxi', name: '疑兵偷袭' }, { traitId: 'mouding_houfa', name: '谋定后发', allowedSides: ['defender', 'reinforcement'] }],
        }],
      }]
      source.pvpReinforcementLosses = { rein_simayi: { weiInfantry: triggered ? 35 : 49 } }
      return { source, model: toOfficialBattleReport(source) }
    }
    const hit = buildModel(true)
    const miss = buildModel(false)

    for (const current of [hit, miss]) {
      expect(current.source.pvpReinforcements![0]!.generals?.[0]?.traits?.map((trait) => trait.traitId)).toEqual(['yibing_touxi', 'mouding_houfa'])
      expect(current.model.sides.map((side) => side.role)).toEqual(['attacker', 'defender', 'reinforcement'])
      expect(current.model.sides[1]).toMatchObject({ role: 'defender', power: current === hit ? 1396 : 1000 })
	  expect(current.model.sides.slice(0, 2).flatMap((side) => side.traits)).toEqual([])
    }
    expect(hit.model.sides[0]).toMatchObject({ role: 'attacker', power: 650, generalExp: 35 })
    expect(miss.model.sides[0]).toMatchObject({ role: 'attacker', power: 1000, generalExp: 50 })
    expect(hit.model.sides[0].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 100, lost: 77, survived: 23 })
    expect(miss.model.sides[0].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 100, lost: 50, survived: 50 })
    expect(hit.model.sides[1].units.find((unit) => unit.key === 'wuInfantry')).toMatchObject({ dispatched: 1, lost: 0, survived: 1 })
    expect(miss.model.sides[1].units.find((unit) => unit.key === 'wuInfantry')).toMatchObject({ dispatched: 1, lost: 1, survived: 0 })
    expect(hit.model.sides[2]).toMatchObject({ role: 'reinforcement', general: { id: 'simayi', name: '司马懿', level: 1 }, generalExp: 77 })
    expect(miss.model.sides[2]).toMatchObject({ role: 'reinforcement', general: { id: 'simayi', name: '司马懿', level: 1 }, generalExp: 50, traits: [], traitText: '' })
    expect(hit.model.sides[2].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 99, lost: 35, survived: 64 })
    expect(miss.model.sides[2].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 99, lost: 49, survived: 50 })
    expect(hit.model.sides[2].traits).toEqual([
      {
        key: 'yibing_touxi-helper_simayi-0', name: '疑兵偷袭', phase: '战斗前',
        detailText: '设计效果比例：35%；战前真实伤亡：蜀步兵 +35；触发概率：100%',
      },
      {
        key: 'mouding_houfa-helper_simayi-1', name: '谋定后发', phase: '防守/增援战斗前',
        detailText: '设计防御加成：35%；实际步防修正：魏步兵 +4；实际骑防修正：魏步兵 +3；触发概率：100%',
      },
    ])
    expect(miss.model.sides.flatMap((side) => side.traits)).toEqual([])
  })

  it('关羽援军随机战前特性命中或未命中时保持准确兵力和方向', () => {
    const cases = [
      {
        generalId: 'guanyu', generalName: '关羽', helperFaction: 'shu', helperUnit: 'shuInfantry', traitId: 'shuiyan_qijun', traitName: '水淹七军', bonusId: 'wusheng_pojun', bonusName: '武圣破军',
        attackerFaction: 'wei', attackerUnit: 'weiInfantry', attackerUnitName: '魏步兵', hitPower: 650, hitAttackerLosses: 77, hitDefendingLosses: 35, hitReinforcementLosses: 35,
        detail: { effectRate: 0.35, maxAffectedRate: 0.35, preBattleAffected: { weiInfantry: 35 }, triggerChance: 1 },
        detailText: '设计效果比例：35%；设计最大影响比例：35%；战前真实伤亡：魏步兵 +35；触发概率：100%',
      },
    ] as const

    for (const current of cases) {
      const buildModel = (triggered: boolean) => {
        const source = report()
        source.result = triggered ? 'defender_victory' : 'draw'
        source.battleType = 'plunder'
        source.detail!.result = triggered ? 'defender_victory' : 'draw'
        source.detail!.battleType = 'plunder'
        source.detail!.winnerSide = triggered ? 'defender' : 'none'
        source.detail!.primarySide = {
          role: 'attacker', faction: current.attackerFaction, power: triggered ? current.hitPower : 1000, generals: [{ id: 'attacker_general', name: '进攻将领', level: 1 }],
          units: [{ unitType: current.attackerUnit, unitName: current.attackerUnitName, amountBefore: 100, dispatched: 100, lost: triggered ? current.hitAttackerLosses : 50, survived: triggered ? 100 - current.hitAttackerLosses : 50 }],
        }
        source.detail!.secondarySide = {
          role: 'defender', faction: 'wu', power: 1000, generals: [{ id: 'sunquan', name: '孙权', level: 1 }],
          units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1, dispatched: 1, lost: triggered ? 0 : 1, survived: triggered ? 1 : 0 }],
        }
        source.detail!.rewards = { generalExp: triggered ? current.hitDefendingLosses : 50 }
        source.detail!.traits = triggered ? [{
          traitId: current.traitId, traitName: current.traitName, ownerSide: 'reinforcement', ownerRole: 'reinforcement', ownerPlayerId: `helper_${current.generalId}`, generalId: current.generalId,
          detail: current.detail,
        }] : []
        source.pvpReinforcements = [{
          reinforcementId: `rein_${current.generalId}`, fromPlayerId: `helper_${current.generalId}`, fromPlayerName: `${current.generalName}援军`, faction: current.helperFaction,
          troops: { [current.helperUnit]: 99 }, generalExpGained: triggered ? current.hitAttackerLosses : 50,
          generals: [{
            id: current.generalId, name: current.generalName, level: 1,
            traits: [{ traitId: current.traitId, name: current.traitName }, { traitId: current.bonusId, name: current.bonusName, allowedSides: ['attacker'] }],
          }],
        }]
        source.pvpReinforcementLosses = { [`rein_${current.generalId}`]: { [current.helperUnit]: triggered ? current.hitReinforcementLosses : 49 } }
        return { source, model: toOfficialBattleReport(source) }
      }
      const hit = buildModel(true)
      const miss = buildModel(false)

      for (const result of [hit, miss]) {
        expect(result.source.pvpReinforcements![0]!.generals?.[0]?.traits?.map((trait) => trait.traitId)).toEqual([current.traitId, current.bonusId])
        expect(result.model.sides.map((side) => side.role)).toEqual(['attacker', 'defender', 'reinforcement'])
        expect(result.model.sides[1]).toMatchObject({ role: 'defender', power: 1000 })
        expect(result.model.sides.slice(0, 2).flatMap((side) => side.traits)).toEqual([])
        expect(result.model.sides.flatMap((side) => side.traits).some((trait) => trait.name === current.bonusName)).toBe(false)
      }
      expect(hit.model.sides[0]).toMatchObject({ role: 'attacker', power: current.hitPower, generalExp: current.hitDefendingLosses })
      expect(miss.model.sides[0]).toMatchObject({ role: 'attacker', power: 1000, generalExp: 50 })
      expect(hit.model.sides[0].units.find((unit) => unit.key === current.attackerUnit)).toMatchObject({ dispatched: 100, lost: current.hitAttackerLosses, survived: 100 - current.hitAttackerLosses })
      expect(miss.model.sides[0].units.find((unit) => unit.key === current.attackerUnit)).toMatchObject({ dispatched: 100, lost: 50, survived: 50 })
      expect(hit.model.sides[1].units.find((unit) => unit.key === 'wuInfantry')).toMatchObject({ dispatched: 1, lost: 0, survived: 1 })
      expect(miss.model.sides[1].units.find((unit) => unit.key === 'wuInfantry')).toMatchObject({ dispatched: 1, lost: 1, survived: 0 })
      expect(hit.model.sides[2]).toMatchObject({ role: 'reinforcement', general: { id: current.generalId, name: current.generalName, level: 1 }, generalExp: current.hitAttackerLosses })
      expect(miss.model.sides[2]).toMatchObject({ role: 'reinforcement', general: { id: current.generalId, name: current.generalName, level: 1 }, generalExp: 50, traits: [], traitText: '' })
      expect(hit.model.sides[2].units.find((unit) => unit.key === current.helperUnit)).toMatchObject({ dispatched: 99, lost: current.hitReinforcementLosses, survived: 99 - current.hitReinforcementLosses })
      expect(miss.model.sides[2].units.find((unit) => unit.key === current.helperUnit)).toMatchObject({ dispatched: 99, lost: 49, survived: 50 })
      expect(hit.model.sides[2].traits).toEqual([{
        key: `${current.traitId}-helper_${current.generalId}-0`, name: current.traitName, phase: '战斗前', detailText: current.detailText,
      }])
      expect(miss.model.sides.flatMap((side) => side.traits)).toEqual([])
    }
  })

  it('张飞援军震慑命中或未命中时区分临时压制与真实阵亡', () => {
    const buildModel = (triggered: boolean) => {
      const source = report()
      source.result = triggered ? 'defender_victory' : 'draw'
      source.battleType = 'plunder'
      source.detail!.result = triggered ? 'defender_victory' : 'draw'
      source.detail!.battleType = 'plunder'
      source.detail!.winnerSide = triggered ? 'defender' : 'none'
      source.detail!.primarySide = {
        role: 'attacker', faction: 'wei', power: triggered ? 500 : 1000, generals: [{ id: 'caocao', name: '曹操', level: 1 }],
        units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: triggered ? 36 : 50, survived: triggered ? 64 : 50 }],
      }
      source.detail!.secondarySide = {
        role: 'defender', faction: 'wu', power: 1000, generals: [{ id: 'sunquan', name: '孙权', level: 1 }],
        units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1, dispatched: 1, lost: triggered ? 0 : 1, survived: triggered ? 1 : 0 }],
      }
      source.detail!.rewards = { generalExp: triggered ? 27 : 50 }
      source.detail!.traits = triggered ? [{
        traitId: 'zhenhe_quanjun', traitName: '万人怒吼', ownerSide: 'reinforcement', ownerRole: 'reinforcement', ownerPlayerId: 'helper_zhangfei', generalId: 'zhangfei',
        detail: { effectRate: 0.5, maxAffectedRate: 0.5, suppressedUnits: { weiInfantry: 50 }, triggerChance: 1 },
      }] : []
      source.pvpReinforcements = [{
        reinforcementId: 'rein_zhangfei', fromPlayerId: 'helper_zhangfei', fromPlayerName: '张飞援军', faction: 'shu',
        troops: { shuInfantry: 99 }, generalExpGained: triggered ? 36 : 50,
        generals: [{
          id: 'zhangfei', name: '张飞', level: 1,
          traits: [{ traitId: 'zhenhe_quanjun', name: '万人怒吼' }, { traitId: 'wanren_nuhou', name: '勇冠三军', allowedSides: ['attacker'] }],
        }],
      }]
      source.pvpReinforcementLosses = { rein_zhangfei: { shuInfantry: triggered ? 27 : 49 } }
      return { source, model: toOfficialBattleReport(source) }
    }
    const hit = buildModel(true)
    const miss = buildModel(false)

    for (const current of [hit, miss]) {
      expect(current.source.pvpReinforcements![0]!.generals?.[0]?.traits?.map((trait) => trait.traitId)).toEqual(['zhenhe_quanjun', 'wanren_nuhou'])
      expect(current.model.sides.map((side) => side.role)).toEqual(['attacker', 'defender', 'reinforcement'])
      expect(current.model.sides[1]).toMatchObject({ role: 'defender', power: 1000 })
      expect(current.model.sides.slice(0, 2).flatMap((side) => side.traits)).toEqual([])
      expect(current.model.sides.flatMap((side) => side.traits).some((trait) => trait.name === '勇冠三军')).toBe(false)
    }
    expect(hit.model.sides[0]).toMatchObject({ role: 'attacker', power: 500, generalExp: 27 })
    expect(miss.model.sides[0]).toMatchObject({ role: 'attacker', power: 1000, generalExp: 50 })
    expect(hit.model.sides[0].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 100, lost: 36, survived: 64 })
    expect(miss.model.sides[0].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 100, lost: 50, survived: 50 })
    expect(hit.model.sides[1].units.find((unit) => unit.key === 'wuInfantry')).toMatchObject({ dispatched: 1, lost: 0, survived: 1 })
    expect(miss.model.sides[1].units.find((unit) => unit.key === 'wuInfantry')).toMatchObject({ dispatched: 1, lost: 1, survived: 0 })
    expect(hit.model.sides[2]).toMatchObject({ role: 'reinforcement', general: { id: 'zhangfei', name: '张飞', level: 1 }, generalExp: 36 })
    expect(miss.model.sides[2]).toMatchObject({ role: 'reinforcement', general: { id: 'zhangfei', name: '张飞', level: 1 }, generalExp: 50, traits: [], traitText: '' })
    expect(hit.model.sides[2].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 99, lost: 27, survived: 72 })
    expect(miss.model.sides[2].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 99, lost: 49, survived: 50 })
    expect(hit.model.sides[2].traits).toEqual([{
      key: 'zhenhe_quanjun-helper_zhangfei-0', name: '万人怒吼', phase: '战斗前',
      detailText: '设计效果比例：50%；设计最大影响比例：50%；本场压制兵力：魏步兵 +50；触发概率：100%',
    }])
    expect(miss.model.sides.flatMap((side) => side.traits)).toEqual([])
  })

  it('黄盖援军苦肉命中或未命中时保持双方后续特性和精确兵损', () => {
    const buildModel = (triggered: boolean) => {
      const source = report()
      source.result = 'draw'
      source.battleType = 'plunder'
      source.detail!.result = 'draw'
      source.detail!.battleType = 'plunder'
      source.detail!.winnerSide = 'none'
      source.detail!.primarySide = {
        role: 'attacker', faction: 'shu', power: 1000,
        generals: [{ id: 'huangzhong', name: '黄忠', level: 1, traits: [{ traitId: 'laodang_yizhuang', name: '老当益壮' }] }],
        units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 100, dispatched: 100, lost: 60, survived: 40 }],
      }
      source.detail!.secondarySide = {
        role: 'defender', faction: 'wu', power: 1000, generals: [{ id: 'sunquan', name: '孙权', level: 1 }],
        units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1, dispatched: 1, lost: 1, survived: 0 }],
      }
      source.detail!.rewards = { generalExp: triggered ? 50 : 60 }
      source.detail!.traits = triggered ? [
        {
          traitId: 'kurouji', traitName: '苦肉计', ownerSide: 'reinforcement', ownerRole: 'reinforcement', ownerPlayerId: 'helper_huanggai', generalId: 'huanggai',
          detail: { disableTraitCount: 1, disabledTraitCount: 1, triggerChance: 1 },
        },
        {
          traitId: 'kurou_fanji', traitName: '苦肉反击', ownerSide: 'reinforcement', ownerRole: 'reinforcement', ownerPlayerId: 'helper_huanggai', generalId: 'huanggai',
          detail: { effectRate: 0.1, extraLosses: { shuInfantry: 10 } },
        },
      ] : [
        {
          traitId: 'laodang_yizhuang', traitName: '老当益壮', ownerSide: 'primary', ownerRole: 'attacker', ownerPlayerId: 'attacker', generalId: 'huangzhong',
          detail: { effectRate: 0.1, extraLosses: { wuInfantry: 10 } },
        },
        {
          traitId: 'kurou_fanji', traitName: '苦肉反击', ownerSide: 'reinforcement', ownerRole: 'reinforcement', ownerPlayerId: 'helper_huanggai', generalId: 'huanggai',
          detail: { effectRate: 0.1, extraLosses: { shuInfantry: 10 } },
        },
      ]
      source.pvpReinforcements = [{
        reinforcementId: 'rein_huanggai', fromPlayerId: 'helper_huanggai', fromPlayerName: '黄盖援军', faction: 'wu',
        troops: { wuInfantry: 99 }, generalExpGained: 60,
        generals: [{
          id: 'huanggai', name: '黄盖', level: 1,
          traits: [{ traitId: 'kurouji', name: '苦肉计' }, { traitId: 'kurou_fanji', name: '苦肉反击' }],
        }],
      }]
      source.pvpReinforcementLosses = { rein_huanggai: { wuInfantry: triggered ? 49 : 59 } }
      return { source, model: toOfficialBattleReport(source) }
    }
    const hit = buildModel(true)
    const miss = buildModel(false)

    for (const current of [hit, miss]) {
      expect(current.source.pvpReinforcements![0]!.generals?.[0]?.traits?.map((trait) => trait.traitId)).toEqual(['kurouji', 'kurou_fanji'])
      expect(current.model.sides.map((side) => side.role)).toEqual(['attacker', 'defender', 'reinforcement'])
      expect(current.model.sides[0]).toMatchObject({ role: 'attacker', power: 1000, general: { id: 'huangzhong', name: '黄忠', level: 1 } })
      expect(current.model.sides[0].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 100, lost: 60, survived: 40 })
      expect(current.model.sides[1]).toMatchObject({ role: 'defender', power: 1000 })
      expect(current.model.sides[1].units.find((unit) => unit.key === 'wuInfantry')).toMatchObject({ dispatched: 1, lost: 1, survived: 0 })
      expect(current.model.sides[2]).toMatchObject({ role: 'reinforcement', general: { id: 'huanggai', name: '黄盖', level: 1 }, generalExp: 60 })
    }
    expect(hit.model.sides[0]).toMatchObject({ generalExp: 50, traits: [], traitText: '' })
    expect(miss.model.sides[0]).toMatchObject({ generalExp: 60 })
    expect(hit.model.sides[2].units.find((unit) => unit.key === 'wuInfantry')).toMatchObject({ dispatched: 99, lost: 49, survived: 50 })
    expect(miss.model.sides[2].units.find((unit) => unit.key === 'wuInfantry')).toMatchObject({ dispatched: 99, lost: 59, survived: 40 })
    expect(hit.model.sides[2].traits).toEqual([
      {
        key: 'kurouji-helper_huanggai-0', name: '苦肉计', phase: '战斗结算后',
        detailText: '设计压制特性数：1；实际压制特性数：1；触发概率：100%',
      },
      {
        key: 'kurou_fanji-helper_huanggai-1', name: '苦肉反击', phase: '战斗结算后',
        detailText: '设计效果比例：10%；追加损失：蜀步兵 +10',
      },
    ])
    expect(miss.model.sides[0].traits).toEqual([{
      key: 'laodang_yizhuang-attacker-0', name: '老当益壮', phase: '战斗结算后',
      detailText: '设计效果比例：10%；追加损失：吴步兵 +10',
    }])
    expect(miss.model.sides[2].traits).toEqual([{
      key: 'kurou_fanji-helper_huanggai-0', name: '苦肉反击', phase: '战斗结算后',
      detailText: '设计效果比例：10%；追加损失：蜀步兵 +10',
    }])
    expect(hit.model.sides.flatMap((side) => side.traits).some((trait) => trait.name === '老当益壮')).toBe(false)
    expect(miss.model.sides.flatMap((side) => side.traits).some((trait) => trait.name === '苦肉计')).toBe(false)
  })

  it('马超援军西凉突击只追加骑兵损失且被动武力不进入时间线', () => {
    const buildModel = (triggered: boolean) => {
      const source = report()
      source.result = 'attacker_victory'
      source.battleType = 'plunder'
      source.detail!.result = 'attacker_victory'
      source.detail!.battleType = 'plunder'
      source.detail!.winnerSide = 'attacker'
      source.detail!.primarySide = {
        role: 'attacker', faction: 'wei', power: 1200, generals: [{ id: 'caocao', name: '曹操', level: 1 }],
        units: [
          { unitType: 'weiCavalry', unitName: '魏骑兵', amountBefore: 50, dispatched: 50, lost: triggered ? 26 : 20, survived: triggered ? 24 : 30 },
          { unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 50, dispatched: 50, lost: 19, survived: 31 },
        ],
      }
      source.detail!.secondarySide = {
        role: 'defender', faction: 'wu', power: 883.3333333333334, generals: [{ id: 'sunquan', name: '孙权', level: 1 }],
        units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1, dispatched: 1, lost: 0, survived: 1 }],
      }
      source.detail!.rewards = { generalExp: 60 }
      source.detail!.traits = triggered ? [{
        traitId: 'xiliang_tuji', traitName: '西凉突击', ownerSide: 'reinforcement', ownerRole: 'reinforcement', ownerPlayerId: 'helper_machao', generalId: 'machao',
        detail: { effectRate: 0.12, targetExtraLosses: { weiCavalry: 6 }, triggerChance: 1 },
      }] : []
      source.pvpReinforcements = [{
        reinforcementId: 'rein_machao', fromPlayerId: 'helper_machao', fromPlayerName: '马超援军', faction: 'shu',
        troops: { shuInfantry: 99 }, generalExpGained: triggered ? 71 : 59,
        generals: [{
          id: 'machao', name: '马超', level: 1, stats: { force: 0 }, effectiveStats: { force: 20 }, buffs: { attackBonus: 0.4 },
          traits: [{ traitId: 'xiliang_tuji', name: '西凉突击' }, { traitId: 'tianshen_xiafan', name: '天神下凡' }],
        }],
      }]
      source.pvpReinforcementLosses = { rein_machao: { shuInfantry: 60 } }
      return { source, model: toOfficialBattleReport(source) }
    }
    const hit = buildModel(true)
    const miss = buildModel(false)

    for (const current of [hit, miss]) {
      expect(current.source.pvpReinforcements![0]!.generals?.[0]).toMatchObject({
        id: 'machao', stats: { force: 0 }, effectiveStats: { force: 20 }, buffs: { attackBonus: 0.4 },
        traits: [{ traitId: 'xiliang_tuji' }, { traitId: 'tianshen_xiafan' }],
      })
      expect(current.model.sides.map((side) => side.role)).toEqual(['attacker', 'defender', 'reinforcement'])
      expect(current.model.sides[0]).toMatchObject({ role: 'attacker', power: 1200, generalExp: 60 })
      expect(current.model.sides[0].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 50, lost: 19, survived: 31 })
      expect(current.model.sides[1]).toMatchObject({ role: 'defender', power: 883.3333333333334 })
      expect(current.model.sides[1].units.find((unit) => unit.key === 'wuInfantry')).toMatchObject({ dispatched: 1, lost: 0, survived: 1 })
      expect(current.model.sides[2].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 99, lost: 60, survived: 39 })
      expect(current.model.sides.slice(0, 2).flatMap((side) => side.traits)).toEqual([])
      expect(current.model.sides.flatMap((side) => side.traits).some((trait) => trait.name === '天神下凡')).toBe(false)
    }
    expect(hit.model.sides[0].units.find((unit) => unit.key === 'weiCavalry')).toMatchObject({ dispatched: 50, lost: 26, survived: 24 })
    expect(miss.model.sides[0].units.find((unit) => unit.key === 'weiCavalry')).toMatchObject({ dispatched: 50, lost: 20, survived: 30 })
    expect(hit.model.sides[2]).toMatchObject({ role: 'reinforcement', general: { id: 'machao', name: '马超', level: 1 }, generalExp: 71 })
    expect(miss.model.sides[2]).toMatchObject({ role: 'reinforcement', general: { id: 'machao', name: '马超', level: 1 }, generalExp: 59, traits: [], traitText: '' })
    expect(hit.model.sides[2].traits).toEqual([{
      key: 'xiliang_tuji-helper_machao-0', name: '西凉突击', phase: '战斗结算后',
      detailText: '设计效果比例：12%；目标兵种追加损失：魏骑兵 +6；触发概率：100%',
    }])
    expect(miss.model.sides.flatMap((side) => side.traits)).toEqual([])
  })

  it('诸葛亮援军战前困兵与全体封禁保持真实归属并对齐最终兵力', () => {
    const source = report()
    source.result = 'defender_victory'
    source.battleType = 'plunder'
    source.detail!.result = 'defender_victory'
    source.detail!.battleType = 'plunder'
    source.detail!.winnerSide = 'defender'
    source.detail!.primarySide = {
      role: 'attacker', faction: 'shu', power: 750, generals: [{ id: 'huangzhong', name: '黄忠', level: 1 }],
      units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 100, dispatched: 100, lost: 45, survived: 55 }],
    }
    source.detail!.secondarySide = {
      role: 'defender', faction: 'wu', power: 1000, generals: [{ id: 'sunquan', name: '孙权', level: 1 }],
      units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1, dispatched: 1, lost: 0, survived: 1 }],
    }
    source.detail!.rewards = { generalExp: 39 }
    source.detail!.traits = [
      {
        traitId: 'qimen_dunjia', traitName: '奇门遁甲', ownerSide: 'reinforcement', ownerRole: 'reinforcement', ownerPlayerId: 'helper_zhugeliang', generalId: 'zhugeliang',
        detail: { effectRate: 0.25, suppressedUnits: { shuInfantry: 25 }, triggerChance: 1 },
      },
      {
        traitId: 'wolong_mouzhi', traitName: '卧龙奇谋', ownerSide: 'reinforcement', ownerRole: 'reinforcement', ownerPlayerId: 'helper_zhugeliang', generalId: 'zhugeliang',
        detail: { disabledGeneralCount: 1, disabledTraitCount: 1, triggerChance: 1 },
      },
    ]
    source.pvpReinforcements = [{
      reinforcementId: 'rein_zhugeliang', fromPlayerId: 'helper_zhugeliang', fromPlayerName: '诸葛亮援军', faction: 'shu',
      troops: { shuInfantry: 99 }, generalExpGained: 45,
      generals: [{
        id: 'zhugeliang', name: '诸葛亮', level: 1,
        traits: [{ traitId: 'qimen_dunjia', name: '奇门遁甲' }, { traitId: 'wolong_mouzhi', name: '卧龙奇谋' }],
      }],
    }]
    source.pvpReinforcementLosses = { rein_zhugeliang: { shuInfantry: 39 } }
    const model = toOfficialBattleReport(source)

    expect(model.sides.map((side) => side.role)).toEqual(['attacker', 'defender', 'reinforcement'])
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 750, generalExp: 39 })
    expect(model.sides[0].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 100, lost: 45, survived: 55 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 1000 })
    expect(model.sides[1].units.find((unit) => unit.key === 'wuInfantry')).toMatchObject({ dispatched: 1, lost: 0, survived: 1 })
    expect(model.sides[2]).toMatchObject({ role: 'reinforcement', general: { id: 'zhugeliang', name: '诸葛亮', level: 1 }, generalExp: 45 })
    expect(model.sides[2].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 99, lost: 39, survived: 60 })
    expect(model.sides.slice(0, 2).flatMap((side) => side.traits)).toEqual([])
    expect(model.sides[2].traits).toEqual([
      {
        key: 'qimen_dunjia-helper_zhugeliang-0', name: '奇门遁甲', phase: '进攻/防守/增援战斗前',
        detailText: '设计效果比例：25%；本场压制兵力：蜀步兵 +25；触发概率：100%',
      },
      {
        key: 'wolong_mouzhi-helper_zhugeliang-1', name: '卧龙奇谋', phase: '进攻/防守/增援战斗前',
        detailText: '封禁将领数：1；实际压制特性数：1；触发概率：100%',
      },
    ])
    expect(model.sides.flatMap((side) => side.traits).some((trait) => trait.name === '老当益壮')).toBe(false)
  })

  it('孙策固定加攻与司马懿全军加防分别展示实际变化', () => {
    const source = report()
    source.battleType = 'attack'
    source.detail!.battleType = 'attack'
    source.detail!.primarySide.faction = 'wu'
    source.detail!.primarySide.power = 16000
    source.detail!.primarySide.generals = [{ id: 'sunce', name: '孙策', level: 1 }]
    source.detail!.primarySide.units = [{ unitType: 'overlordRider', unitName: '霸王骑', amountBefore: 200, dispatched: 200, lost: 90, survived: 110 }]
    source.detail!.secondarySide!.faction = 'wei'
    source.detail!.secondarySide!.power = 14000
    source.detail!.secondarySide!.generals = [{ id: 'simayi', name: '司马懿', level: 1 }]
    source.detail!.secondarySide!.units = [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 1000, survived: 0 }]
    source.detail!.rewards.generalExp = 1000
    source.detail!.traits = [
      {
        traitId: 'xiaobawang_tieqi', traitName: '小霸王', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'sunce',
        detail: { unitAttackFlat: 50, attackModifiedUnits: { overlordRider: 50 } },
      },
      {
        traitId: 'mouding_houfa', traitName: '谋定后发', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'simayi',
        detail: { defenseBonusRate: 0.35, infantryDefenseModifiedUnits: { weiInfantry: 4 }, cavalryDefenseModifiedUnits: { weiInfantry: 3 }, triggerChance: 1 },
      },
    ]
    const model = toOfficialBattleReport(source)
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 16000, generalExp: 1000 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 14000 })
    expect(model.sides[0].units.find((unit) => unit.key === 'overlordRider')).toMatchObject({ dispatched: 200, lost: 90, survived: 110 })
    expect(model.sides[1].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 1000, lost: 1000, survived: 0 })
    expect(model.sides[0].traits).toEqual([
      {
        key: 'xiaobawang_tieqi-0', name: '小霸王', phase: '主动进攻战斗前',
        detailText: '设计单位攻击增加：50；实际攻击修正：霸王骑 +50',
      },
    ])
    expect(model.sides[1].traits).toEqual([
      {
        key: 'mouding_houfa-0', name: '谋定后发', phase: '防守/增援战斗前',
        detailText: '设计防御加成：35%；实际步防修正：魏步兵 +4；实际骑防修正：魏步兵 +3；触发概率：100%',
      },
    ])
  })

  it('甘宁掠夺战双特性对齐攻击、战损、经验和最终掠夺资源', () => {
    const source = report()
    source.battleType = 'plunder'
    source.detail!.battleType = 'plunder'
    source.detail!.primarySide.faction = 'wu'
    source.detail!.primarySide.power = 1100
    source.detail!.primarySide.generals = [{ id: 'ganning', name: '甘宁', level: 1 }]
    source.detail!.primarySide.units = [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 100, dispatched: 100, lost: 48, survived: 52 }]
    source.detail!.secondarySide!.faction = 'wei'
    source.detail!.secondarySide!.power = 1050
    source.detail!.secondarySide!.units = [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 105, dispatched: 105, lost: 54, survived: 51 }]
    source.detail!.rewards.generalExp = 54
    source.detail!.rewards.resources = { wood: 312 }
    source.detail!.traits = [
      {
        traitId: 'jinfan_qixi', traitName: '锦帆奇袭', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'ganning',
        detail: { attackBonusRate: 0.1, attackModifiedUnits: { wuInfantry: 1 }, triggerChance: 1 },
      },
      {
        traitId: 'jinfan_jielue', traitName: '锦帆劫掠', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'ganning',
        detail: { plunderBonusRate: 0.2, plunderDelta: { wood: 52 }, triggerChance: 1 },
      },
    ]
    const model = toOfficialBattleReport(source)
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 1100, generalExp: 54 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 1050 })
    expect(model.sides[0].units.find((unit) => unit.key === 'wuInfantry')).toMatchObject({ dispatched: 100, lost: 48, survived: 52 })
    expect(model.sides[1].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 105, lost: 54, survived: 51 })
    expect(model.resourceText).toContain('木材:312')
    expect(model.sides[0].traits).toEqual([
      {
        key: 'jinfan_qixi-0', name: '锦帆奇袭', phase: '掠夺战战斗前',
        detailText: '设计攻击加成：10%；实际攻击修正：吴步兵 +1；触发概率：100%',
      },
      {
        key: 'jinfan_jielue-1', name: '锦帆劫掠', phase: '掠夺结算',
        detailText: '设计掠夺修正：20%；掠夺资源修正：木材 +52；触发概率：100%',
      },
    ])
    expect(model.sides[1].traits).toEqual([])
  })

  it('孙权掠夺防守双特性对齐防御、战损和最终资源减量', () => {
    const source = report()
    source.battleType = 'plunder'
    source.detail!.battleType = 'plunder'
    source.detail!.primarySide.faction = 'wei'
    source.detail!.primarySide.power = 2000
    source.detail!.primarySide.generals = [{ id: 'caocao', name: '曹操', level: 1 }]
    source.detail!.primarySide.units = [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 200, dispatched: 200, lost: 79, survived: 121 }]
    source.detail!.secondarySide!.faction = 'wu'
    source.detail!.secondarySide!.power = 1500
    source.detail!.secondarySide!.generals = [{ id: 'sunquan', name: '孙权', level: 1 }]
    source.detail!.secondarySide!.units = [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 100, dispatched: 100, lost: 60, survived: 40 }]
    source.detail!.rewards.generalExp = 60
    source.detail!.rewards.resources = { wood: 484 }
    source.detail!.traits = [
      {
        traitId: 'jiangdong_gushou', traitName: '江东固守', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'sunquan',
        detail: { defenseBonusRate: 0.5, infantryDefenseModifiedUnits: { wuInfantry: 5 }, cavalryDefenseModifiedUnits: { wuInfantry: 4 }, triggerChance: 1 },
      },
      {
        traitId: 'jiangdong_haoling', traitName: '江东号令', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'sunquan',
        detail: { plunderBonusRate: -0.2, plunderDelta: { wood: -121 } },
      },
    ]
    const model = toOfficialBattleReport(source)
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 2000, generalExp: 60 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 1500 })
    expect(model.sides[0].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 200, lost: 79, survived: 121 })
    expect(model.sides[1].units.find((unit) => unit.key === 'wuInfantry')).toMatchObject({ dispatched: 100, lost: 60, survived: 40 })
    expect(model.resourceText).toContain('木材:484')
    expect(model.sides[0].traits).toEqual([])
    expect(model.sides[1].traits).toEqual([
      {
        key: 'jiangdong_gushou-0', name: '江东固守', phase: '防守/增援战斗前',
        detailText: '设计防御加成：50%；实际步防修正：吴步兵 +5；实际骑防修正：吴步兵 +4；触发概率：100%',
      },
      {
        key: 'jiangdong_haoling-1', name: '江东号令', phase: '掠夺结算',
        detailText: '设计掠夺修正：-20%；掠夺资源修正：木材 -121',
      },
    ])
  })

  it('孙权进攻和甘宁防守时拥有快照不会补造掠夺特性', () => {
    const source = report()
    source.battleType = 'plunder'
    source.detail!.battleType = 'plunder'
    source.detail!.primarySide.faction = 'wu'
    source.detail!.primarySide.power = 10000
    source.detail!.primarySide.generals = [{
      id: 'sunquan', name: '孙权', level: 1,
      traits: [{ traitId: 'jiangdong_haoling', name: '江东号令' }],
    }]
    source.detail!.primarySide.units = [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1000, dispatched: 1000, lost: 1, survived: 999 }]
    source.detail!.secondarySide!.faction = 'wu'
    source.detail!.secondarySide!.power = 100
    source.detail!.secondarySide!.generals = [{
      id: 'ganning', name: '甘宁', level: 1,
      traits: [{ traitId: 'jinfan_jielue', name: '锦帆劫掠' }, { traitId: 'jinfan_qixi', name: '锦帆奇袭' }],
    }]
    source.detail!.secondarySide!.units = [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 10, dispatched: 10, lost: 9, survived: 1 }]
    source.detail!.rewards.generalExp = 9
    source.detail!.rewards.resources = { wood: 4995 }
    source.detail!.traits = []
    source.pvpReinforcements = []
    source.pvpReinforcementLosses = {}
    const model = toOfficialBattleReport(source)
    expect(model.sides.map((side) => side.role)).toEqual(['attacker', 'defender'])
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 10000, general: { id: 'sunquan', name: '孙权', level: 1 }, generalExp: 9, traitText: '', traits: [] })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 100, general: { id: 'ganning', name: '甘宁', level: 1 }, traitText: '', traits: [] })
    expect(model.sides[0].units.find((unit) => unit.key === 'wuInfantry')).toMatchObject({ dispatched: 1000, lost: 1, survived: 999 })
    expect(model.sides[1].units.find((unit) => unit.key === 'wuInfantry')).toMatchObject({ dispatched: 10, lost: 9, survived: 1 })
    expect(model.resourceText).toContain('木材:4,995')
    expect(model.sides.flatMap((side) => side.traits)).toEqual([])
  })

  it('黄盖同阶段双特性压制敌方后续伤害并保留自身反击', () => {
    const source = report()
    source.battleType = 'plunder'
    source.detail!.battleType = 'plunder'
    source.detail!.primarySide.faction = 'wu'
    source.detail!.primarySide.power = 10000
    source.detail!.primarySide.generals = [{ id: 'huanggai', name: '黄盖', level: 1 }]
    source.detail!.primarySide.units = [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 }]
    source.detail!.secondarySide!.faction = 'shu'
    source.detail!.secondarySide!.power = 10000
    source.detail!.secondarySide!.generals = [{ id: 'huangzhong', name: '黄忠', level: 1 }]
    source.detail!.secondarySide!.units = [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 600, survived: 400 }]
    source.detail!.rewards.generalExp = 600
    source.detail!.traits = [
      {
        traitId: 'kurouji', traitName: '苦肉计', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'huanggai',
        detail: { disableTraitCount: 1, disabledTraitCount: 1, triggerChance: 1 },
      },
      {
        traitId: 'kurou_fanji', traitName: '苦肉反击', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'huanggai',
        detail: { effectRate: 0.1, extraLosses: { shuInfantry: 100 } },
      },
    ]
    const model = toOfficialBattleReport(source)
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 10000, generalExp: 600 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 10000 })
    expect(model.sides[0].units.find((unit) => unit.key === 'wuInfantry')).toMatchObject({ dispatched: 1000, lost: 500, survived: 500 })
    expect(model.sides[1].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 1000, lost: 600, survived: 400 })
    expect(model.sides[0].traits).toEqual([
      {
        key: 'kurouji-0', name: '苦肉计', phase: '战斗结算后',
        detailText: '设计压制特性数：1；实际压制特性数：1；触发概率：100%',
      },
      {
        key: 'kurou_fanji-1', name: '苦肉反击', phase: '战斗结算后',
        detailText: '设计效果比例：10%；追加损失：蜀步兵 +100',
      },
    ])
    expect(model.sides[1].traits).toEqual([])
    expect(model.sides.flatMap((side) => side.traits).some((trait) => trait.name === '老当益壮')).toBe(false)
  })

  it('黄盖苦肉计未命中时双方后续伤害都正常生效', () => {
    const source = report()
    source.battleType = 'plunder'
    source.detail!.battleType = 'plunder'
    source.detail!.primarySide.faction = 'wu'
    source.detail!.primarySide.power = 10000
    source.detail!.primarySide.generals = [{
      id: 'huanggai', name: '黄盖', level: 1,
      traits: [{ traitId: 'kurouji', name: '苦肉计' }, { traitId: 'kurou_fanji', name: '苦肉反击' }],
    }]
    source.detail!.primarySide.units = [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1000, dispatched: 1000, lost: 600, survived: 400 }]
    source.detail!.secondarySide!.faction = 'shu'
    source.detail!.secondarySide!.power = 10000
    source.detail!.secondarySide!.generals = [{ id: 'huangzhong', name: '黄忠', level: 1, traits: [{ traitId: 'laodang_yizhuang', name: '老当益壮' }] }]
    source.detail!.secondarySide!.units = [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 600, survived: 400 }]
    source.detail!.rewards.generalExp = 600
    source.detail!.traits = [
      {
        traitId: 'kurou_fanji', traitName: '苦肉反击', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'huanggai',
        detail: { effectRate: 0.1, extraLosses: { shuInfantry: 100 } },
      },
      {
        traitId: 'laodang_yizhuang', traitName: '老当益壮', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'huangzhong',
        detail: { effectRate: 0.1, extraLosses: { wuInfantry: 100 } },
      },
    ]
    const model = toOfficialBattleReport(source)
    expect(source.detail!.primarySide.generals![0]!.traits?.map((trait) => trait.traitId)).toEqual(['kurouji', 'kurou_fanji'])
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 10000, generalExp: 600 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 10000 })
    expect(model.sides[0].units.find((unit) => unit.key === 'wuInfantry')).toMatchObject({ dispatched: 1000, lost: 600, survived: 400 })
    expect(model.sides[1].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 1000, lost: 600, survived: 400 })
    expect(model.sides[0].traits).toEqual([{
      key: 'kurou_fanji-0', name: '苦肉反击', phase: '战斗结算后',
      detailText: '设计效果比例：10%；追加损失：蜀步兵 +100',
    }])
    expect(model.sides[1].traits).toEqual([{
      key: 'laodang_yizhuang-0', name: '老当益壮', phase: '战斗结算后',
      detailText: '设计效果比例：10%；追加损失：吴步兵 +100',
    }])
    expect(model.sides.flatMap((side) => side.traits).some((trait) => trait.name === '苦肉计')).toBe(false)
  })

  it('防守黄盖苦肉计未命中时按攻守顺序展示双方真实追加损失', () => {
    const source = report()
    source.viewType = 'defense'
    source.result = 'draw'
    source.battleType = 'plunder'
    source.detail!.viewType = 'defense'
    source.detail!.result = 'draw'
    source.detail!.winnerSide = 'none'
    source.detail!.ownerSide = 'defender'
    source.detail!.battleType = 'plunder'
    source.detail!.primarySide.faction = 'shu'
    source.detail!.primarySide.power = 10000
    source.detail!.primarySide.generals = [{ id: 'huangzhong', name: '黄忠', level: 1, traits: [{ traitId: 'laodang_yizhuang', name: '老当益壮' }] }]
    source.detail!.primarySide.units = [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 600, survived: 400 }]
    source.detail!.secondarySide!.faction = 'wu'
    source.detail!.secondarySide!.power = 10000
    source.detail!.secondarySide!.generals = [{
      id: 'huanggai', name: '黄盖', level: 1,
      traits: [{ traitId: 'kurouji', name: '苦肉计' }, { traitId: 'kurou_fanji', name: '苦肉反击' }],
    }]
    source.detail!.secondarySide!.units = [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1000, dispatched: 1000, lost: 600, survived: 400 }]
    source.detail!.rewards.generalExp = 600
    source.detail!.traits = [
      {
        traitId: 'laodang_yizhuang', traitName: '老当益壮', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'huangzhong',
        detail: { effectRate: 0.1, extraLosses: { wuInfantry: 100 } },
      },
      {
        traitId: 'kurou_fanji', traitName: '苦肉反击', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'huanggai',
        detail: { effectRate: 0.1, extraLosses: { shuInfantry: 100 } },
      },
    ]
    const model = toOfficialBattleReport(source)
    expect(source.detail!.secondarySide!.generals![0]!.traits?.map((trait) => trait.traitId)).toEqual(['kurouji', 'kurou_fanji'])
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 10000, generalExp: null, result: 'unknown' })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 10000, generalExp: 600, result: 'unknown' })
    expect(model.sides[0].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 1000, lost: 600, survived: 400 })
    expect(model.sides[1].units.find((unit) => unit.key === 'wuInfantry')).toMatchObject({ dispatched: 1000, lost: 600, survived: 400 })
    expect(model.sides[0].traits).toEqual([{
      key: 'laodang_yizhuang-0', name: '老当益壮', phase: '战斗结算后',
      detailText: '设计效果比例：10%；追加损失：吴步兵 +100',
    }])
    expect(model.sides[1].traits).toEqual([{
      key: 'kurou_fanji-0', name: '苦肉反击', phase: '战斗结算后',
      detailText: '设计效果比例：10%；追加损失：蜀步兵 +100',
    }])
    expect(model.sides.flatMap((side) => side.traits).some((trait) => trait.name === '苦肉计')).toBe(false)
  })

  it('马超被动武力只留在参战快照，触发时间线仅展示西凉突击', () => {
    const source = report()
    source.battleType = 'plunder'
    source.detail!.battleType = 'plunder'
    source.detail!.primarySide.faction = 'shu'
    source.detail!.primarySide.power = 14000
    source.detail!.primarySide.generals = [{
      id: 'machao', name: '马超', level: 1,
      traits: [{ traitId: 'xiliang_tuji', name: '西凉突击' }, { traitId: 'tianshen_xiafan', name: '天神下凡' }],
    }]
    source.detail!.primarySide.units = [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 382, survived: 618 }]
    source.detail!.secondarySide!.faction = 'wei'
    source.detail!.secondarySide!.power = 10000
    source.detail!.secondarySide!.generals = [{ id: 'caocao', name: '曹操', level: 1 }]
    source.detail!.secondarySide!.units = [{ unitType: 'weiCavalry', unitName: '魏骑兵', amountBefore: 1000, dispatched: 1000, lost: 737, survived: 263 }]
    source.detail!.rewards.generalExp = 737
    source.detail!.rewards.generalLevelBefore = 1
    source.detail!.rewards.generalLevelAfter = 2
    source.detail!.traits = [{
      traitId: 'xiliang_tuji', traitName: '西凉突击', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'machao',
      detail: { effectRate: 0.12, targetExtraLosses: { weiCavalry: 120 }, triggerChance: 1 },
    }]
    const model = toOfficialBattleReport(source)
    expect(model.sides[0]).toMatchObject({
      role: 'attacker', power: 14000, general: { id: 'machao', name: '马超', level: 1 },
      generalExp: 737, generalLevelBefore: 1, generalLevelAfter: 2,
    })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 10000 })
    expect(model.sides[0].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 1000, lost: 382, survived: 618 })
    expect(model.sides[1].units.find((unit) => unit.key === 'weiCavalry')).toMatchObject({ dispatched: 1000, lost: 737, survived: 263 })
    expect(model.sides[0].traits).toEqual([{
      key: 'xiliang_tuji-0', name: '西凉突击', phase: '战斗结算后',
      detailText: '设计效果比例：12%；目标兵种追加损失：魏骑兵 +120；触发概率：100%',
    }])
    expect(model.sides[1].traits).toEqual([])
    expect(model.sides.flatMap((side) => side.traits).some((trait) => trait.name === '天神下凡')).toBe(false)
  })

  it('天神下凡生效但西凉突击未命中时保留被动战力且不展示触发项', () => {
    const source = report()
    source.battleType = 'plunder'
    source.detail!.battleType = 'plunder'
    source.detail!.primarySide.faction = 'shu'
    source.detail!.primarySide.power = 14000
    source.detail!.primarySide.generals = [{
      id: 'machao', name: '马超', level: 1, stats: { force: 0 }, effectiveStats: { force: 20 }, buffs: { attackBonus: 0.4 },
      traits: [{ traitId: 'xiliang_tuji', name: '西凉突击' }, { traitId: 'tianshen_xiafan', name: '天神下凡' }],
    }]
    source.detail!.primarySide.units = [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 382, survived: 618 }]
    source.detail!.secondarySide!.faction = 'wei'
    source.detail!.secondarySide!.power = 10000
    source.detail!.secondarySide!.generals = [{ id: 'caocao', name: '曹操', level: 1 }]
    source.detail!.secondarySide!.units = [{ unitType: 'weiCavalry', unitName: '魏骑兵', amountBefore: 1000, dispatched: 1000, lost: 617, survived: 383 }]
    source.detail!.rewards.generalExp = 617
    source.detail!.traits = []
    const model = toOfficialBattleReport(source)
    expect(source.detail!.primarySide.generals![0]!).toMatchObject({ stats: { force: 0 }, effectiveStats: { force: 20 }, buffs: { attackBonus: 0.4 }, traits: [{ traitId: 'xiliang_tuji' }, { traitId: 'tianshen_xiafan' }] })
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 14000, general: { id: 'machao', name: '马超', level: 1, stats: { force: 0 }, effectiveStats: { force: 20 }, buffs: { attackBonus: 0.4 } }, generalExp: 617, traits: [], traitText: '' })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 10000, traits: [], traitText: '' })
    expect(model.sides[0].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 1000, lost: 382, survived: 618 })
    expect(model.sides[1].units.find((unit) => unit.key === 'weiCavalry')).toMatchObject({ dispatched: 1000, lost: 617, survived: 383 })
    expect(model.sides.flatMap((side) => side.traits)).toEqual([])
  })

  it('魏延进攻破防与防守加防按方向互斥并对齐完整战报数值', () => {
    const ownedTraits = [{ traitId: 'qibing_raohou', name: '奇兵绕后' }, { traitId: 'gushou_hanzhong', name: '固守汉中' }]
    const attackSource = report()
    attackSource.battleType = 'plunder'
    attackSource.detail!.battleType = 'plunder'
    attackSource.detail!.primarySide.faction = 'shu'
    attackSource.detail!.primarySide.power = 10000
    attackSource.detail!.primarySide.generals = [{ id: 'weiyan', name: '魏延', level: 1, traits: ownedTraits }]
    attackSource.detail!.primarySide.units = [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 421, survived: 579 }]
    attackSource.detail!.secondarySide!.faction = 'wei'
    attackSource.detail!.secondarySide!.power = 8000
    attackSource.detail!.secondarySide!.generals = [{ id: 'caocao', name: '曹操', level: 1 }]
    attackSource.detail!.secondarySide!.units = [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 578, survived: 422 }]
    attackSource.detail!.rewards.generalExp = 578
    attackSource.detail!.rewards.generalLevelBefore = 1
    attackSource.detail!.rewards.generalLevelAfter = 2
    attackSource.detail!.traits = [{
      traitId: 'qibing_raohou', traitName: '奇兵绕后', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'weiyan',
      detail: { enemyDefenseReductionRate: 0.2, infantryDefenseModifiedUnits: { weiInfantry: -2 }, cavalryDefenseModifiedUnits: { weiInfantry: -2 }, triggerChance: 1 },
    }]
    const attackModel = toOfficialBattleReport(attackSource)
    expect(attackModel.sides[0]).toMatchObject({
      role: 'attacker', power: 10000, general: { id: 'weiyan', name: '魏延', level: 1 },
      generalExp: 578, generalLevelBefore: 1, generalLevelAfter: 2,
    })
    expect(attackModel.sides[1]).toMatchObject({ role: 'defender', power: 8000 })
    expect(attackModel.sides[0].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 1000, lost: 421, survived: 579 })
    expect(attackModel.sides[1].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 1000, lost: 578, survived: 422 })
    expect(attackModel.sides[0].traits).toEqual([{
      key: 'qibing_raohou-0', name: '奇兵绕后', phase: '永久被动',
      detailText: '设计敌方防御降低：20%；实际步防修正：魏步兵 -2；实际骑防修正：魏步兵 -2；触发概率：100%',
    }])
    expect(attackModel.sides[1].traits).toEqual([])
    expect(attackModel.sides.flatMap((side) => side.traits).some((trait) => trait.name === '固守汉中')).toBe(false)

    const defenseSource = report()
    defenseSource.viewType = 'defense'
    defenseSource.battleType = 'plunder'
    defenseSource.detail!.viewType = 'defense'
    defenseSource.detail!.battleType = 'plunder'
    defenseSource.detail!.ownerSide = 'defender'
    defenseSource.detail!.winnerSide = 'defender'
    defenseSource.detail!.primarySide.faction = 'wei'
    defenseSource.detail!.primarySide.power = 10000
    defenseSource.detail!.primarySide.generals = [{ id: 'caocao', name: '曹操', level: 1 }]
    defenseSource.detail!.primarySide.units = [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 826, survived: 174 }]
    defenseSource.detail!.secondarySide!.faction = 'shu'
    defenseSource.detail!.secondarySide!.power = 30000
    defenseSource.detail!.secondarySide!.generals = [{ id: 'weiyan', name: '魏延', level: 1, traits: ownedTraits }]
    defenseSource.detail!.secondarySide!.units = [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 173, survived: 827 }]
    defenseSource.detail!.rewards.generalExp = 826
    defenseSource.detail!.rewards.generalLevelBefore = 1
    defenseSource.detail!.rewards.generalLevelAfter = 2
    defenseSource.detail!.traits = [{
      traitId: 'gushou_hanzhong', traitName: '固守汉中', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'weiyan',
      detail: { generalDefenseFlat: 20, infantryDefenseModifiedUnits: { shuInfantry: 20 }, cavalryDefenseModifiedUnits: { shuInfantry: 20 }, triggerChance: 1 },
    }]
    const defenseModel = toOfficialBattleReport(defenseSource)
    expect(defenseModel.sides[0]).toMatchObject({ role: 'attacker', power: 10000 })
    expect(defenseModel.sides[1]).toMatchObject({
      role: 'defender', power: 30000, general: { id: 'weiyan', name: '魏延', level: 1 },
      generalExp: 826, generalLevelBefore: 1, generalLevelAfter: 2,
    })
    expect(defenseModel.sides[0].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 1000, lost: 826, survived: 174 })
    expect(defenseModel.sides[1].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 1000, lost: 173, survived: 827 })
    expect(defenseModel.sides[0].traits).toEqual([])
    expect(defenseModel.sides[1].traits).toEqual([{
      key: 'gushou_hanzhong-0', name: '固守汉中', phase: '防守/增援战斗前',
      detailText: '设计全军防御增加：20；实际步防修正：蜀步兵 +20；实际骑防修正：蜀步兵 +20；触发概率：100%',
    }])
    expect(defenseModel.sides.flatMap((side) => side.traits).some((trait) => trait.name === '奇兵绕后')).toBe(false)
  })

  it('魏延奇兵绕后合法未命中时只展示基础战斗结果', () => {
    const source = report()
    source.battleType = 'plunder'
    source.detail!.battleType = 'plunder'
    source.detail!.primarySide.faction = 'shu'
    source.detail!.primarySide.power = 10000
    source.detail!.primarySide.generals = [{
      id: 'weiyan', name: '魏延', level: 1,
      traits: [{ traitId: 'qibing_raohou', name: '奇兵绕后' }, { traitId: 'gushou_hanzhong', name: '固守汉中' }],
    }]
    source.detail!.primarySide.units = [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 }]
    source.detail!.secondarySide!.faction = 'wei'
    source.detail!.secondarySide!.power = 10000
    source.detail!.secondarySide!.generals = [{ id: 'caocao', name: '曹操', level: 1 }]
    source.detail!.secondarySide!.units = [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 }]
    source.detail!.rewards.generalExp = 500
    source.detail!.traits = []

    const model = toOfficialBattleReport(source)
    expect(source.detail!.primarySide.generals![0]!.traits?.map((trait) => trait.traitId)).toEqual(['qibing_raohou', 'gushou_hanzhong'])
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 10000, general: { id: 'weiyan', name: '魏延', level: 1 }, generalExp: 500, traits: [], traitText: '' })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 10000, traits: [], traitText: '' })
    expect(model.sides[0].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 1000, lost: 500, survived: 500 })
    expect(model.sides[1].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 1000, lost: 500, survived: 500 })
    expect(model.sides.flatMap((side) => side.traits)).toEqual([])
  })

  it('夏侯渊增援加速只留在拥有快照且盾阵对齐三方实损', () => {
    const source = report()
    source.result = 'defender_victory'
    source.detail!.result = 'defender_victory'
    source.detail!.winnerSide = 'defender'
    source.detail!.primarySide.faction = 'shu'
    source.detail!.primarySide.power = 1100
    source.detail!.primarySide.generals = [{ id: 'liubei', name: '刘备', level: 1 }]
    source.detail!.primarySide.units = [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 110, dispatched: 110, lost: 110, survived: 0 }]
    source.detail!.secondarySide!.faction = 'wu'
    source.detail!.secondarySide!.power = 1710
    source.detail!.secondarySide!.generals = [{ id: 'sunquan', name: '孙权', level: 1 }]
    source.detail!.secondarySide!.units = [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1, dispatched: 1, lost: 0, survived: 1 }]
    source.detail!.traits = [{
      traitId: 'dunzhen_fangyu', traitName: '盾阵防御', ownerSide: 'reinforcement', ownerRole: 'reinforcement', generalId: 'xiahouyuan',
      detail: { defenseBonusRate: 0.3, infantryDefenseModifiedUnits: { qiQiYing: 4 }, cavalryDefenseModifiedUnits: { qiQiYing: 3 }, triggerChance: 0.6 },
    }]
    source.pvpReinforcements = [{
      reinforcementId: 'rein_xiahouyuan', fromPlayerId: 'player_xiahouyuan', fromPlayerName: '夏侯渊援军', faction: 'wei',
      troops: { qiQiYing: 100 }, generalExpGained: 110, generalLevelBefore: 1, generalLevelAfter: 2,
      generals: [{
        id: 'xiahouyuan', name: '夏侯渊', level: 1,
        traits: [{ traitId: 'jixing_benxi', name: '疾行奔袭', params: { unitAttackFlat: 18, unitSpeedFlat: 5 } }, { traitId: 'dunzhen_fangyu', name: '盾阵防御' }],
      }],
    }]
    source.pvpReinforcementLosses = { rein_xiahouyuan: { qiQiYing: 60 } }
    const model = toOfficialBattleReport(source)
    expect(model.sides.map((side) => side.role)).toEqual(['attacker', 'defender', 'reinforcement'])
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 1100 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 1710 })
    expect(model.sides[2]).toMatchObject({
      role: 'reinforcement', general: { id: 'xiahouyuan', name: '夏侯渊', level: 1 }, generalExp: 110,
      generalLevelBefore: 1, generalLevelAfter: 2,
    })
    expect(model.sides[0].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 110, lost: 110, survived: 0 })
    expect(model.sides[1].units.find((unit) => unit.key === 'wuInfantry')).toMatchObject({ dispatched: 1, lost: 0, survived: 1 })
    expect(model.sides[2].units.find((unit) => unit.key === 'qiQiYing')).toMatchObject({ dispatched: 100, lost: 60, survived: 40 })
    expect(model.sides[0].traits).toEqual([])
    expect(model.sides[1].traits).toEqual([])
    expect(model.sides[2].traits).toEqual([{
      key: 'dunzhen_fangyu-0', name: '盾阵防御', phase: '防守/增援战斗前',
      detailText: '设计防御加成：30%；实际步防修正：骁骑营 +4；实际骑防修正：骁骑营 +3；触发概率：60%',
    }])
    expect(model.sides.flatMap((side) => side.traits).some((trait) => trait.name === '疾行奔袭')).toBe(false)
    expect(model.sides[2].passiveTraits).toEqual([{
      key: 'passive-xiahouyuan-jixing_benxi', name: '疾行奔袭', phase: '永久被动', detailText: '骁骑营攻击 +18；移动 +5',
    }])
  })

  it('赵云增援加速只留在拥有快照且龙胆对齐三方战后减损', () => {
    const source = report()
    source.detail!.primarySide.faction = 'wei'
    source.detail!.primarySide.power = 1100
    source.detail!.primarySide.generals = [{ id: 'caocao', name: '曹操', level: 1 }]
    source.detail!.primarySide.units = [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 110, dispatched: 110, lost: 97, survived: 13 }]
    source.detail!.secondarySide!.faction = 'shu'
    source.detail!.secondarySide!.power = 1010
    source.detail!.secondarySide!.generals = [{ id: 'liubei', name: '刘备', level: 1 }]
    source.detail!.secondarySide!.units = [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1, dispatched: 1, lost: 1, survived: 0 }]
    source.detail!.traits = [{
      traitId: 'longdan_jiuyuan', traitName: '龙胆救援', ownerSide: 'reinforcement', ownerRole: 'reinforcement', generalId: 'zhaoyun',
      detail: { lossReductionRate: 0.2, reducedLosses: { shuInfantry: 20 }, triggerChance: 1 },
    }]
    source.pvpReinforcements = [{
      reinforcementId: 'rein_zhaoyun', fromPlayerId: 'player_zhaoyun', fromPlayerName: '赵云援军', faction: 'shu',
      troops: { shuInfantry: 100 }, generalExpGained: 97,
      generals: [{
        id: 'zhaoyun', name: '赵云', level: 1,
        traits: [{ traitId: 'longdan_jiuyuan', name: '龙胆救援' }, { traitId: 'qijin_qichu', name: '七进七出' }],
      }],
    }]
    source.pvpReinforcementLosses = { rein_zhaoyun: { shuInfantry: 80 } }
    const model = toOfficialBattleReport(source)
    expect(model.sides.map((side) => side.role)).toEqual(['attacker', 'defender', 'reinforcement'])
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 1100 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 1010 })
    expect(model.sides[2]).toMatchObject({
      role: 'reinforcement', general: { id: 'zhaoyun', name: '赵云', level: 1 }, generalExp: 97,
    })
    expect(model.sides[0].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 110, lost: 97, survived: 13 })
    expect(model.sides[1].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 1, lost: 1, survived: 0 })
    expect(model.sides[2].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 100, lost: 80, survived: 20 })
    expect(model.sides[0].traits).toEqual([])
    expect(model.sides[1].traits).toEqual([])
    expect(model.sides[2].traits).toEqual([{
      key: 'longdan_jiuyuan-0', name: '龙胆救援', phase: '防守/增援战斗前及掠夺结算',
      detailText: '设计减损比例：20%；减少损失：蜀步兵 +20；触发概率：100%',
    }])
    expect(model.sides.flatMap((side) => side.traits).some((trait) => trait.name === '七进七出')).toBe(false)
  })

  it('赵云龙胆合法未命中时保留加速快照但援军承担完整损失', () => {
    const source = report()
    source.detail!.primarySide.faction = 'wei'
    source.detail!.primarySide.power = 1100
    source.detail!.primarySide.generals = [{ id: 'caocao', name: '曹操', level: 1 }]
    source.detail!.primarySide.units = [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 110, dispatched: 110, lost: 97, survived: 13 }]
    source.detail!.secondarySide!.faction = 'shu'
    source.detail!.secondarySide!.power = 1010
    source.detail!.secondarySide!.generals = [{ id: 'liubei', name: '刘备', level: 1 }]
    source.detail!.secondarySide!.units = [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1, dispatched: 1, lost: 1, survived: 0 }]
    source.detail!.traits = []
    source.pvpReinforcements = [{
      reinforcementId: 'rein_zhaoyun_miss', fromPlayerId: 'player_zhaoyun', fromPlayerName: '赵云援军', faction: 'shu',
      troops: { shuInfantry: 100 }, generalExpGained: 97,
      generals: [{
        id: 'zhaoyun', name: '赵云', level: 1,
        traits: [{ traitId: 'longdan_jiuyuan', name: '龙胆救援' }, { traitId: 'qijin_qichu', name: '七进七出' }],
      }],
    }]
    source.pvpReinforcementLosses = { rein_zhaoyun_miss: { shuInfantry: 100 } }

    const model = toOfficialBattleReport(source)
    expect(source.pvpReinforcements[0]!.generals?.[0]?.traits?.map((trait) => trait.traitId)).toEqual(['longdan_jiuyuan', 'qijin_qichu'])
    expect(model.sides.map((side) => side.role)).toEqual(['attacker', 'defender', 'reinforcement'])
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 1100 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 1010 })
    expect(model.sides[2]).toMatchObject({ role: 'reinforcement', general: { id: 'zhaoyun', name: '赵云', level: 1 }, generalExp: 97, traits: [], traitText: '' })
    expect(model.sides[0].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 110, lost: 97, survived: 13 })
    expect(model.sides[1].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 1, lost: 1, survived: 0 })
    expect(model.sides[2].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 100, lost: 100, survived: 0 })
    expect(model.sides.flatMap((side) => side.traits)).toEqual([])
  })

  it('郭嘉永久四维和鬼才遗策真实复活分别展示', () => {
    const source = report()
    source.result = 'defender_victory'
    source.detail!.result = 'defender_victory'
    source.detail!.winnerSide = 'defender'
    source.detail!.primarySide.faction = 'wei'
    source.detail!.primarySide.power = 1000
    source.detail!.primarySide.generals = [{
      id: 'guojia', name: '郭嘉', level: 1,
      stats: { intelligence: 0, politics: 0 }, effectiveStats: { intelligence: 10, politics: 10 },
      traits: [{ traitId: 'shengui_zhicai', name: '神鬼之才', params: { politicsBonus: 10, intelligenceBonus: 10 } }, { traitId: 'guicai_yice', name: '鬼才遗策' }],
    }]
    source.detail!.primarySide.units = [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: 100, survived: 22 }]
    source.detail!.secondarySide!.faction = 'shu'
    source.detail!.secondarySide!.power = 10000
    source.detail!.secondarySide!.generals = [{ id: 'liubei', name: '刘备', level: 1 }]
    source.detail!.secondarySide!.units = [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 37, survived: 963 }]
    source.detail!.rewards.generalExp = 37
    source.detail!.traits = [{
      traitId: 'guicai_yice', traitName: '鬼才遗策', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'guojia',
      detail: { effectRate: 0.22, actualLostUnits: { weiInfantry: 100 }, revivedUnits: { weiInfantry: 22 }, totalRevived: 22, triggerChance: 1 },
    }]
    source.pvpReinforcements = []
    source.pvpReinforcementLosses = {}
    const model = toOfficialBattleReport(source)
    expect(model.sides.map((side) => side.role)).toEqual(['attacker', 'defender'])
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 1000, general: { id: 'guojia', name: '郭嘉', level: 1 }, generalExp: 37 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 10000 })
    expect(model.sides[0].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 100, lost: 100, survived: 22 })
    expect(model.sides[1].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 1000, lost: 37, survived: 963 })
    expect(model.sides[0].traits).toEqual([{
      key: 'guicai_yice-0', name: '鬼才遗策', phase: '进攻/防守/增援战斗结束后',
      detailText: '设计效果比例：22%；本场真实阵亡：魏步兵 +100；复活兵力：魏步兵 +22；复活总数：22；触发概率：100%',
    }])
    expect(model.sides[0].passiveTraits).toEqual([{
      key: 'passive-guojia-shengui_zhicai', name: '神鬼之才', phase: '永久被动', detailText: '内政 +10；智谋 +10',
    }])
    expect(model.sides[1].traits).toEqual([])
    expect(model.sides[1].passiveTraits).toEqual([])
  })

  it('荀彧双留城能力只保留拥有快照且不伪造战斗触发', () => {
    const source = report()
    source.result = 'attacker_victory'
    source.detail!.result = 'attacker_victory'
    source.detail!.winnerSide = 'attacker'
    source.detail!.primarySide.faction = 'wei'
    source.detail!.primarySide.power = 1000
    source.detail!.primarySide.generals = [{
      id: 'xunyu', name: '荀彧', level: 1,
      traits: [{ traitId: 'wangzuo_zhicai', name: '王佐之才' }, { traitId: 'neizheng_jingying', name: '内政精营' }],
    }]
    source.detail!.primarySide.units = [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: 37, survived: 63 }]
    source.detail!.secondarySide!.faction = 'shu'
    source.detail!.secondarySide!.power = 500
    source.detail!.secondarySide!.generals = [{ id: 'liubei', name: '刘备', level: 1 }]
    source.detail!.secondarySide!.units = [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 50, dispatched: 50, lost: 50, survived: 0 }]
    source.detail!.rewards.generalExp = 50
    source.detail!.traits = []
    source.pvpReinforcements = []
    source.pvpReinforcementLosses = {}
    const model = toOfficialBattleReport(source)
    expect(model.sides.map((side) => side.role)).toEqual(['attacker', 'defender'])
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 1000, general: { id: 'xunyu', name: '荀彧', level: 1 }, generalExp: 50 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 500 })
    expect(model.sides[0].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 100, lost: 37, survived: 63 })
    expect(model.sides[1].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 50, lost: 50, survived: 0 })
    expect(model.sides[0].traits).toEqual([])
    expect(model.sides[1].traits).toEqual([])
    expect(model.sides.flatMap((side) => side.traits)).toEqual([])
  })

  it('曹操产出虎卫主动进攻时战报不触发魏武统御', () => {
    const source = report()
    source.result = 'draw'
    source.detail!.result = 'draw'
    source.detail!.winnerSide = ''
    source.detail!.primarySide.faction = 'wei'
    source.detail!.primarySide.power = 1000
    source.detail!.primarySide.generals = [{
      id: 'caocao', name: '曹操', level: 1,
      traits: [{ traitId: 'weiwu_haoling', name: '魏武号令' }, { traitId: 'weiwu_tongyu', name: '魏武统御' }],
    }]
    source.detail!.primarySide.units = [{ unitType: 'huWei', unitName: '虎卫', amountBefore: 100, dispatched: 100, lost: 100, survived: 0 }]
    source.detail!.secondarySide!.faction = 'shu'
    source.detail!.secondarySide!.power = 1000
    source.detail!.secondarySide!.generals = [{ id: 'liubei', name: '刘备', level: 1 }]
    source.detail!.secondarySide!.units = [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 100, dispatched: 100, lost: 100, survived: 0 }]
    source.detail!.rewards.generalExp = 100
    source.detail!.traits = []
    source.pvpReinforcements = []
    source.pvpReinforcementLosses = {}
    const model = toOfficialBattleReport(source)
    expect(model.sides.map((side) => side.role)).toEqual(['attacker', 'defender'])
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 1000, general: { id: 'caocao', name: '曹操', level: 1 }, generalExp: 100 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 1000 })
    expect(model.sides[0].units.find((unit) => unit.key === 'huWei')).toMatchObject({ dispatched: 100, lost: 100, survived: 0 })
    expect(model.sides[1].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 100, lost: 100, survived: 0 })
    expect(model.sides[0].traits).toEqual([])
    expect(model.sides[1].traits).toEqual([])
    expect(model.sides.flatMap((side) => side.traits).some((trait) => trait.name === '魏武号令')).toBe(false)
  })

  it('攻守双方黄忠的同一正式特性分别展示实际追加损失', () => {
    const source = report()
    source.detail!.primarySide.faction = 'shu'
    source.detail!.primarySide.power = 10000
    source.detail!.primarySide.generals = [{ id: 'huangzhong', name: '黄忠', level: 1 }]
    source.detail!.primarySide.units = [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 600, survived: 400 }]
    source.detail!.secondarySide!.faction = 'shu'
    source.detail!.secondarySide!.power = 10000
    source.detail!.secondarySide!.generals = [{ id: 'huangzhong', name: '黄忠', level: 1 }]
    source.detail!.secondarySide!.units = [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 600, survived: 400 }]
    source.detail!.rewards.generalExp = 600
    source.detail!.traits = [
      {
        traitId: 'laodang_yizhuang', traitName: '老当益壮', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'huangzhong',
        detail: { effectRate: 0.1, extraLosses: { shuInfantry: 100 }, triggerChance: 1 },
      },
      {
        traitId: 'laodang_yizhuang', traitName: '老当益壮', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'huangzhong',
        detail: { effectRate: 0.1, extraLosses: { shuInfantry: 100 }, triggerChance: 1 },
      },
    ]
    const model = toOfficialBattleReport(source)
    expect(model.sides.map((side) => side.role)).toEqual(['attacker', 'defender', 'reinforcement'])
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 10000, general: { id: 'huangzhong', name: '黄忠', level: 1 }, generalExp: 600 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 10000, general: { id: 'huangzhong', name: '黄忠', level: 1 } })
    expect(model.sides[0].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 1000, lost: 600, survived: 400 })
    expect(model.sides[1].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 1000, lost: 600, survived: 400 })
    expect(model.sides[0].traits).toEqual([{
      key: 'laodang_yizhuang-0', name: '老当益壮', phase: '战斗结算后',
      detailText: '设计效果比例：10%；追加损失：蜀步兵 +100；触发概率：100%',
    }])
    expect(model.sides[1].traits).toEqual([{
      key: 'laodang_yizhuang-0', name: '老当益壮', phase: '战斗结算后',
      detailText: '设计效果比例：10%；追加损失：蜀步兵 +100；触发概率：100%',
    }])
  })

  it('历史攻守双方曹操结果仍按当前阶段名称展示实际修正', () => {
    const source = report()
    source.detail!.primarySide.faction = 'wei'
    source.detail!.primarySide.power = 1100
    source.detail!.primarySide.generals = [{ id: 'caocao', name: '曹操', level: 1 }]
    source.detail!.primarySide.units = [{ unitType: 'huWei', unitName: '虎卫', amountBefore: 100, dispatched: 100, lost: 50, survived: 50 }]
    source.detail!.secondarySide!.faction = 'wei'
    source.detail!.secondarySide!.power = 1100
    source.detail!.secondarySide!.generals = [{ id: 'caocao', name: '曹操', level: 1 }]
    source.detail!.secondarySide!.units = [{ unitType: 'huWei', unitName: '虎卫', amountBefore: 100, dispatched: 100, lost: 50, survived: 50 }]
    source.detail!.rewards.generalExp = 50
    source.detail!.traits = [
      {
        traitId: 'weiwu_tongyu', traitName: '魏武统御', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'caocao',
        detail: { attackBonusRate: 0.1, defenseBonusRate: 0.1, attackModifiedUnits: { huWei: 1 }, infantryDefenseModifiedUnits: { huWei: 1 }, cavalryDefenseModifiedUnits: { huWei: 1 }, triggerChance: 1 },
      },
      {
        traitId: 'weiwu_tongyu', traitName: '魏武统御', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'caocao',
        detail: { attackBonusRate: 0.1, defenseBonusRate: 0.1, attackModifiedUnits: { huWei: 1 }, infantryDefenseModifiedUnits: { huWei: 1 }, cavalryDefenseModifiedUnits: { huWei: 1 }, triggerChance: 1 },
      },
    ]
    source.pvpReinforcements = []
    source.pvpReinforcementLosses = {}
    const model = toOfficialBattleReport(source)
    expect(model.sides.map((side) => side.role)).toEqual(['attacker', 'defender'])
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 1100, general: { id: 'caocao', name: '曹操', level: 1 }, generalExp: 50 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 1100, general: { id: 'caocao', name: '曹操', level: 1 } })
    expect(model.sides[0].units.find((unit) => unit.key === 'huWei')).toMatchObject({ dispatched: 100, lost: 50, survived: 50 })
    expect(model.sides[1].units.find((unit) => unit.key === 'huWei')).toMatchObject({ dispatched: 100, lost: 50, survived: 50 })
    const expectedTrait = {
      key: 'weiwu_tongyu-0', name: '魏武统御', phase: '守城/增援战斗前',
      detailText: '设计攻击加成：10%；设计防御加成：10%；实际攻击修正：虎卫 +1；实际步防修正：虎卫 +1；实际骑防修正：虎卫 +1；触发概率：100%',
    }
    expect(model.sides[0].traits).toEqual([expectedTrait])
    expect(model.sides[1].traits).toEqual([expectedTrait])
  })

  it('历史曹操与孙权交叉结果仍保留后端实际数值和归属', () => {
    const source = report()
    source.detail!.primarySide.faction = 'wei'
    source.detail!.primarySide.power = 11000
    source.detail!.primarySide.generals = [{ id: 'caocao', name: '曹操', level: 1 }]
    source.detail!.primarySide.units = [{ unitType: 'huWei', unitName: '虎卫', amountBefore: 1000, dispatched: 1000, lost: 608, survived: 392 }]
    source.detail!.secondarySide!.faction = 'wu'
    source.detail!.secondarySide!.power = 15000
    source.detail!.secondarySide!.generals = [{ id: 'sunquan', name: '孙权', level: 1 }]
    source.detail!.secondarySide!.units = [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1000, dispatched: 1000, lost: 391, survived: 609 }]
    source.detail!.rewards.generalExp = 391
    source.detail!.traits = [
      {
        traitId: 'weiwu_tongyu', traitName: '魏武统御', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'caocao',
        detail: { attackBonusRate: 0.1, defenseBonusRate: 0.1, attackModifiedUnits: { huWei: 1 }, infantryDefenseModifiedUnits: { huWei: 1 }, cavalryDefenseModifiedUnits: { huWei: 1 }, triggerChance: 1 },
      },
      {
        traitId: 'jiangdong_gushou', traitName: '江东固守', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'sunquan',
        detail: { defenseBonusRate: 0.5, infantryDefenseModifiedUnits: { wuInfantry: 5 }, cavalryDefenseModifiedUnits: { wuInfantry: 4 }, triggerChance: 1 },
      },
    ]
    source.pvpReinforcements = []
    source.pvpReinforcementLosses = {}

    const model = toOfficialBattleReport(source)
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 11000, general: { id: 'caocao' }, generalExp: 391 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 15000, general: { id: 'sunquan' } })
    expect(model.sides[0].units.find((unit) => unit.key === 'huWei')).toMatchObject({ dispatched: 1000, lost: 608, survived: 392 })
    expect(model.sides[1].units.find((unit) => unit.key === 'wuInfantry')).toMatchObject({ dispatched: 1000, lost: 391, survived: 609 })
    expect(model.sides[0].traits).toEqual([{
      key: 'weiwu_tongyu-0', name: '魏武统御', phase: '守城/增援战斗前',
      detailText: '设计攻击加成：10%；设计防御加成：10%；实际攻击修正：虎卫 +1；实际步防修正：虎卫 +1；实际骑防修正：虎卫 +1；触发概率：100%',
    }])
    expect(model.sides[1].traits).toEqual([{
      key: 'jiangdong_gushou-0', name: '江东固守', phase: '防守/增援战斗前',
      detailText: '设计防御加成：50%；实际步防修正：吴步兵 +5；实际骑防修正：吴步兵 +4；触发概率：100%',
    }])
  })

  it('甄宓破防后孙权按当前整数防御加防并分别展示实际变化', () => {
    const source = report()
    source.battleType = 'plunder'
    source.detail!.battleType = 'plunder'
    source.detail!.primarySide.faction = 'wei'
    source.detail!.primarySide.power = 10000
    source.detail!.primarySide.generals = [{ id: 'zhenmi', name: '甄宓', level: 1 }]
    source.detail!.primarySide.units = [{ unitType: 'huWei', unitName: '虎卫', amountBefore: 1000, dispatched: 1000, lost: 564, survived: 436 }]
    source.detail!.secondarySide!.faction = 'wu'
    source.detail!.secondarySide!.power = 12000
    source.detail!.secondarySide!.generals = [{ id: 'sunquan', name: '孙权', level: 1 }]
    source.detail!.secondarySide!.units = [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1000, dispatched: 1000, lost: 435, survived: 565 }]
    source.detail!.rewards.generalExp = 435
    source.detail!.traits = [
      {
        traitId: 'meihuo_raozhen', traitName: '魅惑扰阵', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'zhenmi',
        detail: { enemyDefenseReductionRate: 0.25, infantryDefenseModifiedUnits: { wuInfantry: -2 }, cavalryDefenseModifiedUnits: { wuInfantry: -2 }, triggerChance: 1 },
      },
      {
        traitId: 'jiangdong_gushou', traitName: '江东固守', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'sunquan',
        detail: { defenseBonusRate: 0.5, infantryDefenseModifiedUnits: { wuInfantry: 4 }, cavalryDefenseModifiedUnits: { wuInfantry: 3 }, triggerChance: 1 },
      },
    ]
    source.pvpReinforcements = []
    source.pvpReinforcementLosses = {}

    const model = toOfficialBattleReport(source)
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 10000, general: { id: 'zhenmi', name: '甄宓' }, generalExp: 435 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 12000, general: { id: 'sunquan', name: '孙权' } })
    expect(model.sides[0].units.find((unit) => unit.key === 'huWei')).toMatchObject({ dispatched: 1000, lost: 564, survived: 436 })
    expect(model.sides[1].units.find((unit) => unit.key === 'wuInfantry')).toMatchObject({ dispatched: 1000, lost: 435, survived: 565 })
    expect(model.sides[0].traits).toEqual([{
      key: 'meihuo_raozhen-0', name: '魅惑扰阵', phase: '主动进攻战斗前',
      detailText: '设计敌方防御降低：25%；实际步防修正：吴步兵 -2；实际骑防修正：吴步兵 -2；触发概率：100%',
    }])
    expect(model.sides[1].traits).toEqual([{
      key: 'jiangdong_gushou-0', name: '江东固守', phase: '防守/增援战斗前',
      detailText: '设计防御加成：50%；实际步防修正：吴步兵 +4；实际骑防修正：吴步兵 +3；触发概率：100%',
    }])
    expect(model.sides.flatMap((side) => side.traits).map((trait) => trait.name)).toEqual(['魅惑扰阵', '江东固守'])
  })

  it('攻守双方刘备仁主守护分别展示原始阵亡和复活后存活', () => {
    const source = report()
    source.detail!.primarySide.faction = 'shu'
    source.detail!.primarySide.power = 10000
    source.detail!.primarySide.generals = [{
      id: 'liubei', name: '刘备', level: 1,
      stats: { politics: 0, command: 0 }, effectiveStats: { politics: 10, command: 12 },
      traits: [{ traitId: 'rende', name: '仁德天下', params: { politicsBonus: 10, commandBonus: 12 } }, { traitId: 'renzhu_shouhu', name: '仁主守护' }],
    }]
    source.detail!.primarySide.units = [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 675 }]
    source.detail!.secondarySide!.faction = 'shu'
    source.detail!.secondarySide!.power = 10000
    source.detail!.secondarySide!.generals = [{
      id: 'liubei', name: '刘备', level: 1,
      stats: { politics: 0, command: 0 }, effectiveStats: { politics: 10, command: 12 },
      traits: [{ traitId: 'rende', name: '仁德天下', params: { politicsBonus: 10, commandBonus: 12 } }, { traitId: 'renzhu_shouhu', name: '仁主守护' }],
    }]
    source.detail!.secondarySide!.units = [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 675 }]
    source.detail!.rewards.generalExp = 500
    source.detail!.traits = [
      {
        traitId: 'renzhu_shouhu', traitName: '仁主守护', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'liubei',
        detail: { effectRate: 0.35, revivedUnits: { shuInfantry: 175 }, triggerChance: 0.6 },
      },
      {
        traitId: 'renzhu_shouhu', traitName: '仁主守护', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'liubei',
        detail: { effectRate: 0.35, revivedUnits: { shuInfantry: 175 }, triggerChance: 0.6 },
      },
    ]
    source.pvpReinforcements = []
    source.pvpReinforcementLosses = {}
    const model = toOfficialBattleReport(source)
    expect(model.sides.map((side) => side.role)).toEqual(['attacker', 'defender'])
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 10000, general: { id: 'liubei', name: '刘备', level: 1 }, generalExp: 500 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 10000, general: { id: 'liubei', name: '刘备', level: 1 }, generalExp: null })
    expect(model.sides[0].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 1000, lost: 500, survived: 675 })
    expect(model.sides[1].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 1000, lost: 500, survived: 675 })
    const expectedTraits = [
      {
        key: 'renzhu_shouhu-0', name: '仁主守护', phase: '进攻/防守/增援战斗结束后',
        detailText: '设计效果比例：35%；复活兵力：蜀步兵 +175；触发概率：60%',
      },
    ]
    expect(model.sides[0].traits).toEqual(expectedTraits)
    expect(model.sides[1].traits).toEqual(expectedTraits)
    expect(model.sides.flatMap((side) => side.traits)).toHaveLength(2)
    const expectedPassive = [{
      key: 'passive-liubei-rende', name: '仁德天下', phase: '永久被动', detailText: '内政 +10；统率 +12',
    }]
    expect(model.sides[0].passiveTraits).toEqual(expectedPassive)
    expect(model.sides[1].passiveTraits).toEqual(expectedPassive)
  })

  it('刘备多兵种大额阵亡逐兵种按百分比复活且不设人数上限', () => {
    const source = report()
    source.result = 'defender_victory'
    source.detail!.result = 'defender_victory'
    source.detail!.winnerSide = 'defender'
    source.detail!.primarySide.faction = 'shu'
    source.detail!.primarySide.power = 720000
    source.detail!.primarySide.generals = [{ id: 'liubei', name: '刘备', level: 1 }]
    source.detail!.primarySide.units = [
      { unitType: 'shuCavalry', unitName: '蜀骑兵', amountBefore: 30000, dispatched: 30000, lost: 30000, survived: 10500 },
      { unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 30000, dispatched: 30000, lost: 30000, survived: 10500 },
    ]
    source.detail!.secondarySide!.faction = 'wei'
    source.detail!.secondarySide!.power = 8833333
    source.detail!.secondarySide!.generals = [{ id: 'caocao', name: '曹操', level: 1 }]
    source.detail!.secondarySide!.units = [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000000, dispatched: 1000000, lost: 28296, survived: 971704 }]
    source.detail!.rewards.generalExp = 28296
    source.detail!.traits = [
      {
        traitId: 'renzhu_shouhu', traitName: '仁主守护', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'liubei',
        detail: { effectRate: 0.35, revivedUnits: { shuCavalry: 10500, shuInfantry: 10500 }, totalRevived: 21000, triggerChance: 0.6 },
      },
    ]
    source.pvpReinforcements = []
    source.pvpReinforcementLosses = {}
    const model = toOfficialBattleReport(source)
    expect(model.sides.map((side) => side.role)).toEqual(['attacker', 'defender'])
    expect(model.sides[0]).toMatchObject({ role: 'attacker', power: 720000, general: { id: 'liubei', name: '刘备', level: 1 }, generalExp: 28296 })
    expect(model.sides[1]).toMatchObject({ role: 'defender', power: 8833333 })
    expect(model.sides[0].units.find((unit) => unit.key === 'shuCavalry')).toMatchObject({ dispatched: 30000, lost: 30000, survived: 10500 })
    expect(model.sides[0].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 30000, lost: 30000, survived: 10500 })
    expect(model.sides[1].units.find((unit) => unit.key === 'weiInfantry')).toMatchObject({ dispatched: 1000000, lost: 28296, survived: 971704 })
    expect(model.sides[0].traits).toEqual([
      {
        key: 'renzhu_shouhu-0', name: '仁主守护', phase: '进攻/防守/增援战斗结束后',
        detailText: '设计效果比例：35%；复活兵力：蜀骑兵 +10,500、蜀步兵 +10,500；复活总数：21,000；触发概率：60%',
      },
    ])
    expect(model.sides[1].traits).toEqual([])
  })

  it('黄巾防守战报把四项战前防御结果归给防守武将', () => {
    const cases = [
      ['mouding_houfa', '司马懿', { defenseBonusRate: 0.35, infantryDefenseModifiedUnits: { weiInfantry: 4 }, cavalryDefenseModifiedUnits: { weiInfantry: 3 }, triggerChance: 1 }, '设计防御加成：35%；实际步防修正：魏步兵 +4；实际骑防修正：魏步兵 +3；触发概率：100%'],
      ['dunzhen_fangyu', '夏侯渊', { defenseBonusRate: 0.3, infantryDefenseModifiedUnits: { weiInfantry: 3 }, cavalryDefenseModifiedUnits: { weiInfantry: 2 }, triggerChance: 0.6 }, '设计防御加成：30%；实际步防修正：魏步兵 +3；实际骑防修正：魏步兵 +2；触发概率：60%'],
      ['gushou_hanzhong', '魏延', { generalDefenseFlat: 20, infantryDefenseModifiedUnits: { weiInfantry: 20 }, cavalryDefenseModifiedUnits: { weiInfantry: 20 }, triggerChance: 1 }, '设计全军防御增加：20；实际步防修正：魏步兵 +20；实际骑防修正：魏步兵 +20；触发概率：100%'],
      ['jiangdong_gushou', '孙权', { defenseBonusRate: 0.5, infantryDefenseModifiedUnits: { weiInfantry: 5 }, cavalryDefenseModifiedUnits: { weiInfantry: 4 }, triggerChance: 1 }, '设计防御加成：50%；实际步防修正：魏步兵 +5；实际骑防修正：魏步兵 +4；触发概率：100%'],
    ] as const
    for (const [traitId, generalName, detailData, detailText] of cases) {
      const source = report()
      source.viewType = 'defense'
      source.detail!.viewType = 'defense'
      source.detail!.sourceType = 'yellow_turban'
      source.detail!.ownerSide = 'defender'
      source.detail!.secondarySide!.faction = 'wei'
      source.detail!.secondarySide!.generals = [{ id: traitId, name: generalName, level: 1 }]
      source.detail!.secondarySide!.units = [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: 20, survived: 80 }]
      source.detail!.traits = [{ traitId, traitName: traitId, ownerSide: 'secondary', ownerRole: 'defender', generalId: traitId, detail: detailData }]
      const model = toOfficialBattleReport(source)
      expect(model.sides[0].traits).toHaveLength(0)
      expect(model.sides[1].traits[0]).toMatchObject({ name: traitLabel(traitId), detailText })
    }
  })

  it('孙权黄巾守城保留双特性快照但只展示江东固守真实结果', () => {
    const source = report()
    source.viewType = 'defense'
    source.detail!.viewType = 'defense'
    source.detail!.sourceType = 'yellow_turban'
    source.detail!.ownerSide = 'defender'
    source.detail!.primarySide = { role: 'attacker', faction: 'wei', power: 1000, generals: [], units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: 60, survived: 40 }] }
    source.detail!.secondarySide = {
      role: 'defender', faction: 'wu', power: 1500,
      generals: [{ id: 'sunquan', name: '孙权', level: 1, traits: [{ traitId: 'jiangdong_haoling', name: '江东号令' }, { traitId: 'jiangdong_gushou', name: '江东固守' }] }],
      units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 100, dispatched: 100, lost: 40, survived: 60 }],
    }
    source.detail!.traits = [{
      traitId: 'jiangdong_gushou', traitName: '江东固守', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'sunquan',
      detail: { defenseBonusRate: 0.5, infantryDefenseModifiedUnits: { wuInfantry: 5 }, cavalryDefenseModifiedUnits: { wuInfantry: 4 }, triggerChance: 1 },
    }]
    const model = toOfficialBattleReport(source)
    expect(source.detail!.secondarySide!.generals?.[0]?.traits?.map((trait) => trait.traitId)).toEqual(['jiangdong_haoling', 'jiangdong_gushou'])
    expect(model.sides[0].traits).toEqual([])
    expect(model.sides[1].traits).toEqual([{
      key: 'jiangdong_gushou-0', name: '江东固守', phase: '防守/增援战斗前',
      detailText: '设计防御加成：50%；实际步防修正：吴步兵 +5；实际骑防修正：吴步兵 +4；触发概率：100%',
    }])
  })

  it('黄巾防守战报把四项战后追加伤害归给防守将领', () => {
    const cases = [
      ['laodang_yizhuang', 'huangzhong', '黄忠', 'extraLosses', 20, 0.1],
      ['huoshao_lianying', 'luxun', '陆逊', 'targetExtraLosses', 126, 1],
      ['lianying_zengshang', 'luxun', '陆逊', 'targetExtraLosses', 20, 0.1],
      ['kurou_fanji', 'huanggai', '黄盖', 'extraLosses', 20, 0.1],
    ] as const
    for (const [traitId, generalId, generalName, detailKey, amount, rate] of cases) {
      const source = report()
      source.viewType = 'defense'
      source.detail!.viewType = 'defense'
      source.detail!.sourceType = 'yellow_turban'
      source.detail!.ownerSide = 'defender'
      source.detail!.primarySide.faction = 'wei'
      source.detail!.primarySide.units = [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 200, dispatched: 200, lost: amount, survived: 200 - amount }]
      source.detail!.secondarySide!.generals = [{ id: generalId, name: generalName, level: 1 }]
      source.detail!.traits = [{
        traitId, traitName: traitLabel(traitId), ownerSide: 'secondary', ownerRole: 'defender', generalId,
        detail: { effectRate: rate, [detailKey]: { weiInfantry: amount }, triggerChance: 1 },
      }]
      const model = toOfficialBattleReport(source)
      expect(model.sides[0].traits).toEqual([])
      expect(model.sides[1].traits[0]).toMatchObject({
        name: traitLabel(traitId), phase: '战斗结算后',
        detailText: `设计效果比例：${rate * 100}%；${detailKey === 'targetExtraLosses' ? '目标兵种追加损失' : '追加损失'}：魏步兵 +${amount}；触发概率：100%`,
      })
    }
  })

  it('三项破防特性统一标记为主动进攻战斗前', () => {
    const cases = [
      ['meihuo_raozhen', 0.25, -2],
	  ['huchi_chongzhen', 0.3, -3],
      ['baibu_chuanyang', 0.2, -2],
    ] as const
    for (const [traitId, designRate, actualDelta] of cases) {
      const source = report()
      source.detail!.primarySide.generals = [{ id: traitId, name: traitId, level: 1 }]
      source.detail!.traits = [{ traitId, traitName: traitId, ownerSide: 'primary', generalId: traitId, detail: { enemyDefenseReductionRate: designRate, infantryDefenseModifiedUnits: { greedyWolf: actualDelta }, cavalryDefenseModifiedUnits: { greedyWolf: actualDelta }, triggerChance: 1 } }]
      const model = toOfficialBattleReport(source)
      expect(model.sides[0].traits[0]).toMatchObject({
        phase: '主动进攻战斗前', detailText: `设计敌方防御降低：${designRate * 100}%；实际步防修正：贪狼营 ${actualDelta}；实际骑防修正：贪狼营 ${actualDelta}；触发概率：100%`,
      })
    }
  })

  it('未触发的将领特性保留在快照但不进入触发时间线', () => {
    const source = report()
    source.detail!.primarySide.generals![0].traits = [
      { traitId: 'weiwu_haoling', name: 'weiwu_haoling', scope: 'self_army', targetUnitType: 'huWei', params: { guardPerMinute: 500, maxGuardPerDay: 3000 } },
      { traitId: 'weiwu_tongyu', name: 'weiwu_tongyu', scope: 'self_army', targetUnitType: 'special', params: { attackBonusRate: 0.1, defenseBonusRate: 0.1 } },
    ]
    const model = toOfficialBattleReport(source)
    expect(source.detail!.primarySide.generals![0].traits).toHaveLength(2)
    expect(model.sides[0].traits).toEqual([])
  })

  it('行军和掠夺配置未触发时不伪造战斗时间线', () => {
    const source = report()
    source.detail!.primarySide.generals![0].traits = [
      { traitId: 'baiyi_dujiang', name: '白衣渡江', params: { speedBonusRate: 0.2, warningDelayRate: 0.3, minMarchSeconds: 60 } },
      { traitId: 'jiangdong_haoling', name: '江东号令', params: { plunderBonusRate: -0.2 } },
    ]
    const model = toOfficialBattleReport(source)
    expect(model.sides[0].traits).toEqual([])
  })

  it('已触发特性合并配置比例和本场实际作用数量且不补造其他特性', () => {
    const source = report()
    source.detail!.primarySide.generals![0].traits = [
      { traitId: 'baibu_chuanyang', name: '百步穿杨', params: { triggerChance: 0.35, enemyDefenseReductionRate: 0.2 } },
      { traitId: 'laodang_yizhuang', name: '老当益壮', params: { effectRate: 0.1 } },
    ]
    source.detail!.traits = [{ traitId: 'laodang_yizhuang', traitName: '老当益壮', ownerSide: 'primary', generalId: 'g1', detail: { extraLosses: { greedyWolf: 12 } } }]
    const model = toOfficialBattleReport(source)
    expect(model.sides[0].traits.map((trait) => trait.name)).toEqual(['老当益壮'])
    expect(model.sides[0].traits[0]).toMatchObject({ phase: '战斗结算后' })
    expect(model.sides[0].traits[0].detailText).toContain('设计效果比例：10%')
    expect(model.sides[0].traits[0].detailText).toContain('追加损失：贪狼营 +12')
  })

  it('攻防双方使用相同将领与特性时不会把一方实际结果串给另一方', () => {
    const source = report()
    source.detail!.primarySide.generals = [{ id: 'same-general', name: '镜像将领', traits: [{ traitId: 'laodang_yizhuang', name: '老当益壮', params: { effectRate: 0.1 } }] }]
    source.detail!.secondarySide!.generals = [{ id: 'same-general', name: '镜像将领', traits: [{ traitId: 'laodang_yizhuang', name: '老当益壮', params: { effectRate: 0.1 } }] }]
    source.detail!.traits = [{ traitId: 'laodang_yizhuang', traitName: '老当益壮', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'same-general', detail: { extraLosses: { greedyWolf: 12 } } }]
    source.traitOutcomes = { laodang_yizhuang: { traitId: 'laodang_yizhuang', ownerSide: 'attacker', ownerGeneralId: 'same-general', ownerPlayerId: 'p1', detail: { extraLosses: { greedyWolf: 12 } } } }
    const model = toOfficialBattleReport(source)
    expect(model.sides[0].traits[0].detailText).toContain('追加损失：贪狼营 +12')
    expect(model.sides[1].traits).toEqual([])
  })

  it('无标准 detail 的历史战报仍保留己方触发特性详情', () => {
    const source = report()
    delete source.detail
    source.defenderRevealed = true
    source.traitTriggered = ['huogong']
    source.traitOutcomes = { huogong: { traitId: 'huogong', name: '火烧赤壁', ownerSide: 'attacker', detail: { damagePercent: 0.25 } } }
    const model = toOfficialBattleReport(source)
    expect(model.sides[0].traits[0]).toMatchObject({ name: '火烧赤壁', detailText: '设计伤害比例：25%' })
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
    expect(reinforcementSides[1].traits).toEqual([])
  })

  it('把掠夺特性的阶段、详情字段和资源键全部转换为中文', () => {
    const source = report()
		source.detail!.primarySide.generals = [{ id: 'ganning', name: '甘宁', level: 1 }]
    source.detail!.secondarySide!.generals = [{ id: 'g2', name: '孙权', level: 1 }]
		source.detail!.traits = [
			{ traitId: 'jinfan_jielue', traitName: '锦帆劫掠', ownerSide: 'primary', generalId: 'ganning', detail: { plunderBonusRate: 0.2, plunderDelta: { wood: 60 }, triggerChance: 1 } },
			{ traitId: 'jiangdong_haoling', traitName: '江东号令', ownerSide: 'secondary', generalId: 'g2', detail: { plunderBonusRate: -0.2, plunderDelta: { food: -60, iron: -60, stone: -60, wood: -60 }, triggerChance: 1 } },
		]
    const model = toOfficialBattleReport(source)
		expect(model.sides[0].traits[0]).toMatchObject({
			name: '锦帆劫掠', phase: '掠夺结算',
			detailText: '设计掠夺修正：20%；掠夺资源修正：木材 +60；触发概率：100%',
		})
    expect(model.sides[1].traits[0]).toMatchObject({
      name: '江东号令', phase: '掠夺结算',
      detailText: '设计掠夺修正：-20%；掠夺资源修正：粮食 -60、铁矿 -60、泥土 -60、木材 -60；触发概率：100%',
    })
    expect(model.sides.flatMap((side) => side.traits).map((trait) => trait.name)).toEqual(['锦帆劫掠', '江东号令'])
    expect(model.sides[1].traitText).not.toMatch(/plunderDelta|food|iron|stone|wood/)
  })

  it('把 NPC 孙策追击明确标记为掠夺战结算后', () => {
    const source = report()
    source.detail!.sourceType = 'npc_city'
    source.detail!.primarySide.generals = [{ id: 'sunce', name: '孙策', level: 1 }]
    source.detail!.traits = [{ traitId: 'xiaobawang_zhuiji', traitName: '小霸王追击', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'sunce', detail: { effectRate: 0.1, extraLosses: { greedyWolf: 10 }, triggerChance: 1 } }]
    const model = toOfficialBattleReport(source)
    expect(model.sides[0].traits[0]).toMatchObject({ name: '小霸王追击', phase: '掠夺战结算后', detailText: '设计效果比例：10%；追加损失：贪狼营 +10；触发概率：100%' })
    expect(model.sides[1].traits).toEqual([])
  })

  it('把锦帆奇袭明确标记为掠夺战战斗前并展示实际攻击变化', () => {
    const source = report()
    source.detail!.primarySide.faction = 'wu'
    source.detail!.primarySide.generals = [{ id: 'ganning', name: '甘宁', level: 1 }]
    source.detail!.primarySide.units = [{ unitType: 'shadowGuard', unitName: '影卫', amountBefore: 100, dispatched: 100, lost: 10, survived: 90 }]
    source.detail!.traits = [{ traitId: 'jinfan_qixi', traitName: '锦帆奇袭', ownerSide: 'primary', generalId: 'ganning', detail: { attackBonusRate: 0.1, attackModifiedUnits: { shadowGuard: 1 }, triggerChance: 1 } }]
    const model = toOfficialBattleReport(source)
    expect(model.sides[0].traits[0]).toMatchObject({ name: '锦帆奇袭', phase: '掠夺战战斗前', detailText: '设计攻击加成：10%；实际攻击修正：影卫 +1；触发概率：100%' })
  })

  it('把奇兵绕后明确标记为主动进攻并展示实际防御变化', () => {
    const source = report()
    source.detail!.primarySide.generals = [{ id: 'weiyan', name: '魏延', level: 1 }]
    source.detail!.secondarySide!.units = [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: 10, survived: 90 }]
    source.detail!.traits = [{ traitId: 'qibing_raohou', traitName: '奇兵绕后', ownerSide: 'primary', generalId: 'weiyan', detail: { enemyDefenseReductionRate: 0.2, infantryDefenseModifiedUnits: { weiInfantry: -2 }, cavalryDefenseModifiedUnits: { weiInfantry: -2 }, triggerChance: 1 } }]
    const model = toOfficialBattleReport(source)
    expect(model.sides[0].traits[0]).toMatchObject({
      name: '奇兵绕后', phase: '永久被动',
      detailText: '设计敌方防御降低：20%；实际步防修正：魏步兵 -2；实际骑防修正：魏步兵 -2；触发概率：100%',
    })
  })

  it('兼容同一特性双方触发的唯一存储键并保留各自数值', () => {
    const source = report()
    source.detail = undefined
    source.defenderRevealed = true
    source.traitTriggered = ['shared_trait', 'shared_trait::defender::g2']
    source.traitOutcomes = {
      shared_trait: { traitId: 'shared_trait', name: '同名特性', ownerSide: 'attacker', detail: { extraLosses: { huWei: 10 } } },
      'shared_trait::defender::g2': { traitId: 'shared_trait', name: '同名特性', ownerSide: 'defender', ownerGeneralId: 'g2', detail: { reducedLosses: { greedyWolf: 5 } } },
    }
    const model = toOfficialBattleReport(source)
    expect(model.sides[0].traits[0]).toMatchObject({ name: '同名特性', detailText: '追加损失：虎卫 +10' })
    expect(model.sides[1].traits[0]).toMatchObject({ name: '同名特性', detailText: '减少损失：贪狼营 +5' })
  })

  it('两项正式压制特性分别展示真实压制数量', () => {
    const source = report()
    source.detail!.traits = [
      { traitId: 'wolong_mouzhi', traitName: '卧龙奇谋', ownerSide: 'primary', generalId: 'g1', detail: { disabledGeneralCount: 1, disabledTraitCount: 2, triggerChance: 1 } },
      { traitId: 'kurouji', traitName: '苦肉计', ownerSide: 'secondary', generalId: 'g2', detail: { disableTraitCount: 1, disabledTraitCount: 0, triggerChance: 1 } },
    ]
    const model = toOfficialBattleReport(source)
    expect(model.sides[0].traits[0]).toMatchObject({ name: '卧龙奇谋', phase: '进攻/防守/增援战斗前', detailText: '封禁将领数：1；实际压制特性数：2；触发概率：100%' })
    expect(model.sides[1].traits[0]).toMatchObject({ name: '苦肉计', phase: '战斗结算后', detailText: '设计压制特性数：1；实际压制特性数：0；触发概率：100%' })
  })

  it('双方苦肉计同时触发时保留两条同名压制并按攻守分侧', () => {
    const source = report()
    source.detail!.primarySide.power = 1000
    source.detail!.primarySide.generals = [{ id: 'attacker_general', name: '进攻将领', level: 1 }]
    source.detail!.primarySide.units = [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 100, dispatched: 100, lost: 50, survived: 50 }]
    source.detail!.secondarySide!.power = 1020
    source.detail!.secondarySide!.generals = [{ id: 'defender_general', name: '防守将领', level: 1 }]
    source.detail!.secondarySide!.units = [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 100, dispatched: 100, lost: 49, survived: 51 }]
    source.detail!.traits = [
      {
        traitId: 'kurouji', traitName: '苦肉计', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'attacker_general',
        detail: { disableTraitCount: 1, disabledTraitCount: 1, triggerChance: 1 },
      },
      {
        traitId: 'kurouji', traitName: '苦肉计', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'defender_general',
        detail: { disableTraitCount: 1, disabledTraitCount: 1, triggerChance: 1 },
      },
    ]
    const model = toOfficialBattleReport(source)
    expect(model.sides[0].traits).toEqual([
      expect.objectContaining({ name: '苦肉计', phase: '战斗结算后', detailText: '设计压制特性数：1；实际压制特性数：1；触发概率：100%' }),
    ])
    expect(model.sides[1].traits).toEqual([
      expect.objectContaining({ name: '苦肉计', phase: '战斗结算后', detailText: '设计压制特性数：1；实际压制特性数：1；触发概率：100%' }),
    ])
    expect(model.sides.flatMap((side) => side.traits).some((trait) => trait.name === '苦肉反击' || trait.name === '老当益壮')).toBe(false)
    expect(model.sides[0].units.find((unit) => unit.key === 'wuInfantry')).toMatchObject({ dispatched: 100, lost: 50, survived: 50 })
    expect(model.sides[1].units.find((unit) => unit.key === 'shuInfantry')).toMatchObject({ dispatched: 100, lost: 49, survived: 51 })
  })

  it('双方诸葛亮的卧龙奇谋都显示失效且不伪报封禁结果', () => {
    const source = report()
    source.detail!.traits = [
      { traitId: 'wolong_mouzhi', traitName: '卧龙奇谋', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'g1', detail: { status: '特性已失效', invalidReason: '双方均有诸葛亮' } },
      { traitId: 'wolong_mouzhi', traitName: '卧龙奇谋', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'g2', detail: { status: '特性已失效', invalidReason: '双方均有诸葛亮' } },
    ]
    const model = toOfficialBattleReport(source)
    for (const side of model.sides.slice(0, 2)) {
      expect(side.traits[0]).toMatchObject({
        name: '卧龙奇谋', phase: '进攻/防守/增援战斗前',
        detailText: '状态：特性已失效；失效原因：双方均有诸葛亮',
      })
    }
  })

  it('诸葛亮双特性区分本场压兵与战前全体封禁', () => {
    const source = report()
    source.detail!.primarySide.generals = [{ id: 'zhugeliang', name: '诸葛亮', level: 1 }]
    source.detail!.traits = [
      {
        traitId: 'qimen_dunjia', traitName: '奇门遁甲', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'zhugeliang',
        detail: { effectRate: 0.25, suppressedUnits: { greedyWolf: 25 }, triggerChance: 1 },
      },
      {
        traitId: 'wolong_mouzhi', traitName: '卧龙奇谋', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'zhugeliang',
        detail: { disabledGeneralCount: 1, disabledTraitCount: 2, triggerChance: 1 },
      },
    ]
    const model = toOfficialBattleReport(source)
    expect(model.sides[0].traits).toHaveLength(2)
    expect(model.sides[0].traits[0]).toMatchObject({
      name: '奇门遁甲', phase: '进攻/防守/增援战斗前',
      detailText: '设计效果比例：25%；本场压制兵力：贪狼营 +25；触发概率：100%',
    })
    expect(model.sides[0].traits[1]).toMatchObject({
      name: '卧龙奇谋', phase: '进攻/防守/增援战斗前',
      detailText: '封禁将领数：1；实际压制特性数：2；触发概率：100%',
    })
    expect(model.sides[1].traits).toEqual([])
    expect(model.sides.flatMap((side) => side.traits).some((trait) => trait.name === '老当益壮')).toBe(false)
  })

  it('旧格式战报优先使用后端给出的真实存活兵力', () => {
    const source = report()
    source.detail = undefined
    source.dispatchedUnits = { huWei: 100 }
    source.lostUnits = { huWei: 120 }
    source.survivedUnits = { huWei: 40 }
    const model = toOfficialBattleReport(source)
    expect(model.sides[0].units.find((unit) => unit.key === 'huWei')).toMatchObject({ dispatched: 100, lost: 120, survived: 40 })
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
