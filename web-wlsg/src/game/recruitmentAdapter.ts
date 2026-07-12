/** 将 Hero3 真实兵种配置、兵力和官方视觉编号转换为征兵页视图模型。 */
import type { UnitConfig, UnitsConfig } from '../api/types'
import type { ArmyUnitState, RecruitQueueState } from './types'

export type RecruitmentCategoryKey = 'infantry' | 'cavalry' | 'siege' | 'special' | 'other'

export interface RecruitmentUnitViewModel {
  id: string
  officialCode: number | null
  name: string
  description: string
  category: RecruitmentCategoryKey
  stats: [number, number, number, number, number, number]
  owned: number
  cost: Record<string, number>
  trainSeconds: number
}

export interface RecruitmentCategoryViewModel {
  id: RecruitmentCategoryKey | 'trap'
  label: string
  description: string
  unitLabel: string
  queueLimit: number
  batchLimit: number
  units: RecruitmentUnitViewModel[]
}

export interface RecruitmentQueueViewModel extends RecruitQueueState {
  unitName: string
  officialCode: number | null
}

export const recruitmentStatLabels = ['攻击', '步防', '骑防', '速度', '运载', '口粮']
export const RECRUIT_QUEUE_LIMIT = 20

const officialCodeByUnitType: Record<string, number> = {
  qingZhouArmy: 101, jinWeiSoldier: 102, huWei: 103, zhanYingTanMa: 104, qiQiYing: 105, huBaoQi: 106, chongZhuangChe: 107, luLeiChe: 108, jianzhuShi: 109, tuZu: 110,
  greedyWolf: 112, qilinGuard: 113, azureDragon: 114, flyingKite: 115, xiLiangCavalry: 116, southernElephant: 117, siegeTower: 118, thunderBolt: 119, woodenOx: 120, hanRoyalty: 121,
  shadowGuard: 123, xiuLuo: 124, secretAgent: 125, divineWind: 126, zhuQueRider: 127, overlordRider: 128, chongChe: 129, juShiChe: 130, fengShuiMaster: 131, taiPingShi: 132,
}

const categoryDefinitions: Array<Omit<RecruitmentCategoryViewModel, 'units'>> = [
  { id: 'infantry', label: '步兵营', unitLabel: '步兵', queueLimit: RECRUIT_QUEUE_LIMIT, batchLimit: 100000, description: '这里是城池的步兵中心，可以征募本国特色步兵来扩充军备。最终资源消耗、训练时间和队列结果以 Hero3 后端为准。' },
  { id: 'cavalry', label: '骑兵营', unitLabel: '骑兵', queueLimit: RECRUIT_QUEUE_LIMIT, batchLimit: 100000, description: '这里是城池的骑兵中心，可以征募当前阵营配置中的骑兵和侦察单位。' },
  { id: 'siege', label: '攻城武器', unitLabel: '攻城器械', queueLimit: RECRUIT_QUEUE_LIMIT, batchLimit: 100000, description: '攻城武器营负责征募当前阵营配置中的破墙和破坏建筑单位。' },
  { id: 'special', label: '特殊兵种', unitLabel: '特殊兵种', queueLimit: RECRUIT_QUEUE_LIMIT, batchLimit: 100000, description: '特殊兵种包含拓城、说服和运输等当前阵营注册单位。' },
]

const unknownCategory = { id: 'other' as const, label: '其他兵种', unitLabel: '兵种', queueLimit: RECRUIT_QUEUE_LIMIT, batchLimit: 100000, description: '以下是 Hero3 后端新增或未知分类的真实兵种，保留原始名称与数值。' }

/** 将后端分类安全转换为页面分类。 */
function normalizeCategory(category: string): RecruitmentCategoryKey {
  return category === 'infantry' || category === 'cavalry' || category === 'siege' || category === 'special' ? category : 'other'
}

/** 从真实兵力列表汇总同类型数量，兼容异常重复项。 */
function armyAmounts(army: ArmyUnitState[]): Record<string, number> {
  return army.reduce<Record<string, number>>((result, unit) => {
    result[unit.unitType] = (result[unit.unitType] ?? 0) + (unit.amount ?? 0)
    return result
  }, {})
}

/** 将单个后端兵种配置转换为卡片视图模型。 */
function mapUnit(id: string, config: UnitConfig, owned: number): RecruitmentUnitViewModel {
  const stat = (key: string) => Number(config.stats?.[key] ?? 0)
  return {
    id,
    officialCode: officialCodeByUnitType[id] ?? null,
    name: config.name?.trim() || id,
    description: config.description?.trim() || '后端暂未提供该兵种说明。',
    category: normalizeCategory(config.category),
    stats: [stat('attack'), stat('infantryDefense'), stat('cavalryDefense'), stat('speed'), stat('carryCapacity'), stat('upkeep')],
    owned,
    cost: { ...(config.cost ?? {}) },
    trainSeconds: Math.max(0, Number(config.trainSeconds ?? 0)),
  }
}

/** 建立当前阵营的真实征兵分类和动态卡片。 */
export function toRecruitmentCategories(faction: string, unitsConfig: UnitsConfig, army: ArmyUnitState[]): RecruitmentCategoryViewModel[] {
  const amounts = armyAmounts(army)
  const units = Object.entries(unitsConfig[faction] ?? {}).map(([id, config]) => mapUnit(id, config, amounts[id] ?? 0))
    .sort((left, right) => left.trainSeconds - right.trainSeconds || left.name.localeCompare(right.name, 'zh-CN'))
  const categories = categoryDefinitions.map((definition) => ({ ...definition, units: units.filter((unit) => unit.category === definition.id) }))
  const unknownUnits = units.filter((unit) => unit.category === 'other')
  if (unknownUnits.length) categories.push({ ...unknownCategory, units: unknownUnits })
  categories.push({ id: 'trap', label: '机关阵', unitLabel: '机关', queueLimit: RECRUIT_QUEUE_LIMIT, batchLimit: 100000, description: 'Hero3 现有兵种配置未提供可征募的机关单位。', units: [] })
  return categories
}

/** 把后端征兵队列补充为可读名称和官方图标。 */
export function toRecruitmentQueues(queues: RecruitQueueState[], categories: RecruitmentCategoryViewModel[]): RecruitmentQueueViewModel[] {
  const units = new Map(categories.flatMap((category) => category.units).map((unit) => [unit.id, unit]))
  return queues.map((queue) => ({ ...queue, unitName: units.get(queue.unitType)?.name ?? queue.unitType, officialCode: units.get(queue.unitType)?.officialCode ?? null }))
}

/** 计算基础配置下的兵种消耗，仅用于提交前参考。 */
export function baseRecruitCost(unit: RecruitmentUnitViewModel, amount: number): Record<string, number> {
  return Object.fromEntries(Object.entries(unit.cost).map(([key, value]) => [key, value * amount]))
}

/** 计算队列立即完成的基础城金估算。 */
export function estimateInstantCost(remainingSeconds: number, cityGoldPerSecond: number): number {
  return Math.max(1, Math.ceil(Math.max(0, remainingSeconds) / Math.max(1, cityGoldPerSecond)))
}
