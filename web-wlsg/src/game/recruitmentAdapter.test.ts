/** 验证真实兵种配置、未知类型、重复兵力和队列的征兵页映射。 */
import { describe, expect, it } from 'vitest'
import type { UnitsConfig } from '../api/types'
import { baseRecruitCost, estimateInstantCost, toRecruitmentCategories, toRecruitmentQueues } from './recruitmentAdapter'

const units: UnitsConfig = {
  wei: {
    qingZhouArmy: { name: '青州军', description: '基础步兵', category: 'infantry', icon: '', stats: { attack: 8, infantryDefense: 7, cavalryDefense: 10, speed: 6, carryCapacity: 80, upkeep: 2 }, cost: { wood: 240 }, trainSeconds: 60, unlock: {} },
    futureUnit: { name: '未来兵种', description: '', category: 'future', icon: '', stats: {}, cost: {}, trainSeconds: 0, unlock: {} },
  },
}

describe('征兵页真实数据适配', () => {
  it('映射真实配置并汇总重复兵力', () => {
    const categories = toRecruitmentCategories('wei', units, [{ unitType: 'qingZhouArmy', amount: 2 }, { unitType: 'qingZhouArmy', amount: 3 }])
    expect(categories[0]).toMatchObject({ queueLimit: 20 })
    expect(categories[0].units[0]).toMatchObject({ id: 'qingZhouArmy', name: '青州军', officialCode: 101, owned: 5, stats: [8, 7, 10, 6, 80, 2] })
  })

  it('未知分类与缺失字段使用可见兜底且不丢失', () => {
    const categories = toRecruitmentCategories('wei', units, [])
    const other = categories.find((category) => category.id === 'other')
    expect(other?.units[0]).toMatchObject({ id: 'futureUnit', name: '未来兵种', officialCode: null, stats: [0, 0, 0, 0, 0, 0] })
  })

  it('零配置阵营安全返回空分类', () => {
    expect(toRecruitmentCategories('shu', units, []).slice(0, 4).every((category) => category.units.length === 0)).toBe(true)
  })

  it('队列保留未知兵种并正确计算基础消耗与城金', () => {
    const categories = toRecruitmentCategories('wei', units, [])
    const queues = toRecruitmentQueues([{ id: 'q1', unitType: 'unknown', amount: 0, endsAt: '' }], categories)
    expect(queues[0]).toMatchObject({ unitName: 'unknown', amount: 0 })
    expect(baseRecruitCost(categories[0].units[0], 3)).toEqual({ wood: 720 })
    expect(estimateInstantCost(121, 120)).toBe(2)
  })
})
