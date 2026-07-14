/** NPC 页面层级映射测试，防止四档颜色与中文名称在改版中错位。 */
import { describe, expect, it } from 'vitest'
import type { NpcCityState } from '../game/types'
import { npcFactionLabel, npcRecoveryLabel, npcTierLabel, npcTraitSummary, sortNpcCities } from './npcDirectory'

/** 创建只关注列表映射的最小 NPC 城池。 */
function city(id: string, tier: NpcCityState['tier'], traits: NpcCityState['traits'] = []): NpcCityState {
  return { id, name: id, faction: 'wei', tier, resources: {}, storageCapacity: {}, productionPerHour: {}, army: [], maxArmy: [], armyRecoveryRate: 0, recoveryProfile: 'normal', traits, resourceSettledAt: '', armySettledAt: '', generatedAt: '' }
}

describe('npcDirectory', () => {
  it('按四档业务顺序展示全部真实返回项且不错误去重', () => {
    const result = sortNpcCities([city('g1', 'golden'), city('s1', 'small'), city('s2', 'small'), city('m1', 'medium'), city('l1', 'large')])
    expect(result.map((item) => item.id)).toEqual(['s1', 's2', 'm1', 'l1', 'g1'])
  })

  it('maps the golden backend tier to the Hero3 super-large label', () => {
    expect(npcTierLabel('golden')).toBe('超大')
  })

  it('未知阵营和恢复特性保留真实值并安全映射空词条', () => {
    expect(npcFactionLabel('other')).toBe('other')
    expect(npcRecoveryLabel('new_profile')).toBe('new_profile')
    expect(npcTraitSummary(city('x', 'small'))).toBe('无')
  })

  it('多个真实词条按后端顺序展示', () => {
    expect(npcTraitSummary(city('x', 'large', [{ id: 'a', name: '铁壁', buffs: {} }, { id: 'b', name: '富矿', buffs: {} }]))).toBe('铁壁、富矿')
  })
})
