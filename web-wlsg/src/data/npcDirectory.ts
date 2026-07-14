/** NPC 页面真实字段适配：集中维护层级、阵营、恢复特性和稳定排序。 */
import type { NpcCityState, NpcTier } from '../game/types'

export interface NpcTierMeta {
  label: string
  description: string
}

export const npcTierMeta: Record<NpcTier, NpcTierMeta> = {
  small: { label: '小型', description: '基础资源与少量守军' },
  medium: { label: '中型', description: '资源和守军均有提升' },
  large: { label: '大型', description: '高额资源与强力守军' },
  golden: { label: '超大', description: '稀有高收益 NPC 城池' },
}

export const npcTierOrder: NpcTier[] = ['small', 'medium', 'large', 'golden']
export const npcFactionLabels: Record<string, string> = { wei: '魏', shu: '蜀', wu: '吴' }
export const npcRecoveryLabels: Record<string, string> = { normal: '平庸', rich_mine: '富矿', iron_wall: '铁壁', fortress: '要塞' }

/** 返回官方列表需要的层级中文名，未来新增层级保留原值。 */
export function npcTierLabel(tier: NpcTier | string): string { return npcTierMeta[tier as NpcTier]?.label ?? tier }

/** 返回阵营中文名，未知扩展阵营不丢弃后端原值。 */
export function npcFactionLabel(faction: string): string { return npcFactionLabels[faction] ?? faction }

/** 返回恢复特性中文名，未知扩展特性不丢弃后端原值。 */
export function npcRecoveryLabel(profile: string): string { return npcRecoveryLabels[profile] ?? profile }

/** 按小型到超大稳定排序，同层级保持后端顺序。 */
export function sortNpcCities(cities: NpcCityState[]): NpcCityState[] {
  return cities.map((city, index) => ({ city, index })).sort((left, right) => {
    const tierDifference = npcTierOrder.indexOf(left.city.tier) - npcTierOrder.indexOf(right.city.tier)
    return tierDifference || left.index - right.index
  }).map(({ city }) => city)
}

/** 将真实词条压缩成适合官方窄表格的单行说明。 */
export function npcTraitSummary(city: NpcCityState): string { return city.traits?.length ? city.traits.map((trait) => trait.name?.trim() || trait.id).join('、') : '无' }
