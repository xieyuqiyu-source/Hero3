/** 验证轮回绝境默认随军武将和真实加成摘要。 */
import { describe, expect, it } from 'vitest'
import { dungeonGeneralBonusText, preferredDungeonGeneralId } from './generalPresentation'

describe('轮回绝境随军武将展示', () => {
  it('优先默认选择主将并在主将不可用时回退第一名武将', () => {
    const generals = [{ id: 'g1', name: '副将', level: 10 }, { id: 'g2', name: '主将', level: 20 }]
    expect(preferredDungeonGeneralId(generals, [{ id: 'main', generalId: 'g2', slot: 'main' }])).toBe('g2')
    expect(preferredDungeonGeneralId(generals.slice(0, 1), [{ id: 'main', generalId: 'g2', slot: 'main' }])).toBe('g1')
  })

  it('展示后端真实战斗属性和中文特性名称', () => {
    const text = dungeonGeneralBonusText({ id: 'caocao', name: '曹操', level: 20, buffs: { attackBonus: 0.12, defenseBonus: 0.1, productionBonus: 0.2 }, traits: [{ traitId: 'weiwu_tongyu', name: 'weiwu_tongyu' }] })
    expect(text).toContain('攻击 +12%')
    expect(text).toContain('防御 +10%')
    expect(text).not.toContain('productionBonus')
    expect(text).toContain('魏武统御')
  })
})
