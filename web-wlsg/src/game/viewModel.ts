/** 将 Hero3 完整状态集中转换为武林三国资源页视图模型。 */
import type { ArmyUnitViewModel, BuildingState, CityGameViewModel, GameStateResponse, ResourceBuildingViewModel } from './types'

interface ResourceDefinition { name: string; buildingType: string; buildingName: string; image: string; icon: string }

export const resourceDefinitions: Record<string, ResourceDefinition> = {
  wood: { name: '木材', buildingType: 'wood_camp', buildingName: '伐木场', image: '3.gif', icon: 'pic_mc.gif' },
  stone: { name: '泥土', buildingType: 'stone_quarry', buildingName: '泥土场', image: '2.gif', icon: 'pic_nt.gif' },
  iron: { name: '铁矿', buildingType: 'iron_mine', buildingName: '铁矿场', image: '4.gif', icon: 'pic_tk.gif' },
  food: { name: '粮食', buildingType: 'farm', buildingName: '农田', image: '1.gif', icon: 'pic_ls.gif' },
}

const resourceOrder = ['wood', 'stone', 'iron', 'food']
const factionNames: Record<string, string> = { wei: '魏国', shu: '蜀国', wu: '吴国' }
const relatedBuildingNames: Record<string, string> = { warehouse: '仓库' }
const buildingNames: Record<string, string> = Object.fromEntries(
  Object.values(resourceDefinitions).map((definition) => [definition.buildingType, definition.buildingName]),
)

const unitDefinitions: Partial<Record<string, { name: string; category: 'infantry' | 'cavalry' | 'siege' | 'special' }>> = {
  azureDragon: { name: '青龙军', category: 'infantry' }, flyingKite: { name: '飞鸢', category: 'infantry' }, greedyWolf: { name: '贪狼营', category: 'infantry' },
  qilinGuard: { name: '麒麟卫', category: 'infantry' }, qingZhouArmy: { name: '青州军', category: 'infantry' }, jinWeiSoldier: { name: '禁卫甲士', category: 'infantry' },
  huWei: { name: '虎卫', category: 'infantry' }, shadowGuard: { name: '影卫', category: 'infantry' }, xiuLuo: { name: '修罗', category: 'infantry' },
  xiLiangCavalry: { name: '西凉铁骑', category: 'cavalry' }, southernElephant: { name: '南蛮象', category: 'cavalry' }, zhanYingTanMa: { name: '战鹰骑探', category: 'cavalry' },
  qiQiYing: { name: '骁骑营', category: 'cavalry' }, huBaoQi: { name: '虎豹骑', category: 'cavalry' }, secretAgent: { name: '密探', category: 'cavalry' },
  divineWind: { name: '神风', category: 'cavalry' }, zhuQueRider: { name: '朱雀骑', category: 'cavalry' }, overlordRider: { name: '霸王骑', category: 'cavalry' },
  siegeTower: { name: '临冲车', category: 'siege' }, thunderBolt: { name: '轰天雷', category: 'siege' }, chongZhuangChe: { name: '冲撞车', category: 'siege' },
  luLeiChe: { name: '霹雳车', category: 'siege' }, chongChe: { name: '对楼车', category: 'siege' }, juShiChe: { name: '炬石车', category: 'siege' },
  hanRoyalty: { name: '汉室宗亲', category: 'special' }, woodenOx: { name: '木牛流马', category: 'special' }, jianzhuShi: { name: '建筑师', category: 'special' },
  tuZu: { name: '士族', category: 'special' }, fengShuiMaster: { name: '风水师', category: 'special' }, taiPingShi: { name: '太平术士', category: 'special' },
  shuMerchant: { name: '蜀国商人', category: 'special' }, weiMerchant: { name: '魏国商人', category: 'special' }, wuMerchant: { name: '吴国商人', category: 'special' },
}

/** 以官网标准兵种名称为唯一依据的 army_content 缩略图映射。 */
export const officialUnitIconsByName: Record<string, string> = {
  '青州军': '101.gif', '禁卫甲士': '102.gif', '虎卫': '103.gif', '战鹰骑探': '104.gif', '骁骑营': '105.gif', '虎豹骑': '106.gif', '冲撞车': '107.gif', '霹雳车': '108.gif', '建筑师': '109.gif', '士族': '110.gif',
  '贪狼营': '111.gif', '麒麟卫': '112.gif', '青龙军': '113.gif', '飞鸢': '114.gif', '西凉铁骑': '115.gif', '南蛮象': '116.gif', '临冲车': '117.gif', '轰天雷': '118.gif', '木牛流马': '119.gif', '汉室宗亲': '120.gif',
  '影卫': '121.gif', '修罗': '123.gif', '密探': '124.gif', '神风': '125.gif', '朱雀骑': '126.gif', '霸王骑': '127.gif', '对楼车': '128.gif', '炬石车': '129.gif', '风水师': '130.gif', '太平术士': '131.gif',
}
const generalIcons: Record<string, string> = { wei: 'general_tag_1.gif', shu: 'general_tag_2.gif', wu: 'general_tag_3.gif' }
const categoryOrder = { infantry: 0, cavalry: 1, siege: 2, special: 3, unknown: 4 }

/** 返回建筑当前可读状态。 */
function buildingStatus(building: BuildingState | undefined): string {
  if (!building) return '尚未建造'
  if (building.upgradeEndsAt) return '升级中'
  if (building.status && building.status !== 'active') return building.status
  return '正常'
}

/** 从地块和建筑数组建立不去重的资源建筑卡片。 */
function mapResourceBuildings(state: GameStateResponse): ResourceBuildingViewModel[] {
  const byId = new Map(state.buildings.map((building) => [building.id, building]))
  const slots = state.resourceSlots?.length
    ? state.resourceSlots
    : state.buildings.filter((building) => Object.values(resourceDefinitions).some((definition) => definition.buildingType === building.type))
      .map((building) => ({ id: `derived-${building.id}`, resourceType: Object.entries(resourceDefinitions).find(([, definition]) => definition.buildingType === building.type)?.[0] ?? building.type, buildingId: building.id }))

  const cards: ResourceBuildingViewModel[] = slots.map((slot) => {
    const building = slot.buildingId ? byId.get(slot.buildingId) : undefined
    const definition = resourceDefinitions[slot.resourceType]
    return {
      id: building?.id ?? slot.buildingId ?? slot.id,
      slotId: slot.id,
      resourceKey: slot.resourceType,
      resourceName: definition?.name ?? slot.resourceType,
      buildingName: building ? buildingNames[building.type] ?? building.type : definition?.buildingName ?? '未知资源地块',
      buildingType: building?.type ?? definition?.buildingType ?? slot.resourceType,
      image: definition?.image ?? null,
      level: building?.level ?? 0,
      status: buildingStatus(building),
      endsAt: building?.upgradeEndsAt ?? building?.statusEndsAt ?? null,
      isFallback: !definition || Boolean(building && !buildingNames[building.type]),
    }
  })

  const orderedCards: ResourceBuildingViewModel[] = []
  const knownGroups = resourceOrder.map((key) => cards.filter((card) => card.resourceKey === key))
  const longestGroup = Math.max(0, ...knownGroups.map((group) => group.length))
  for (let index = 0; index < longestGroup; index += 1) {
    for (const group of knownGroups) {
      if (group[index]) orderedCards.push(group[index])
    }
  }
  orderedCards.push(...cards.filter((card) => !resourceOrder.includes(card.resourceKey)))

  const usedBuildingIds = new Set(orderedCards.map((card) => card.id))
  for (const building of state.buildings) {
    if (!relatedBuildingNames[building.type] || usedBuildingIds.has(building.id)) continue
    orderedCards.push({ id: building.id, slotId: `related-${building.id}`, resourceKey: building.type, resourceName: '资源建筑', buildingName: relatedBuildingNames[building.type], buildingType: building.type, image: null, level: building.level, status: buildingStatus(building), endsAt: building.upgradeEndsAt ?? building.statusEndsAt ?? null, isFallback: true })
  }
  return orderedCards
}

/** 将后端完整状态转换为本页唯一展示数据源。 */
export function toCityGameViewModel(state: GameStateResponse, accountGold: number): CityGameViewModel {
  const allResourceKeys = [...resourceOrder, ...Object.keys(state.resources.items).filter((key) => !resourceOrder.includes(key)).sort()]
  const busyGeneralIds = new Set((state.generalAssignments ?? []).filter((assignment) => assignment.id !== 'main' && assignment.slot !== 'main').map((assignment) => assignment.generalId))
  const homeGeneral = state.general && !busyGeneralIds.has(state.general.id) ? state.general : null
  return {
    player: { nickname: state.player.nickname, faction: state.player.faction, factionName: factionNames[state.player.faction] ?? state.player.faction },
    serverTime: state.serverTime,
    accountGold,
    cityGold: state.cityGold ?? 0,
    capacityBoost: state.capacityBoost ?? 0,
    capacityBoostEnd: state.capacityBoostEnd ?? '',
    resources: allResourceKeys.map((key) => ({
      key, name: resourceDefinitions[key]?.name ?? key, icon: resourceDefinitions[key]?.icon ?? 'icon-operation.gif',
      amount: state.resources.items[key] ?? 0, capacity: state.resources.capacity[key] ?? 0, productionPerHour: state.resourceProduction[key] ?? 0,
    })),
    resourceBuildings: mapResourceBuildings(state),
    general: homeGeneral ? { name: homeGeneral.name, level: homeGeneral.level, icon: generalIcons[state.player.faction] ?? 'general_tag_1.gif' } : null,
    army: (state.army ?? []).map((unit) => {
      const definition = unitDefinitions[unit.unitType]
      const category: ArmyUnitViewModel['category'] = definition?.category ?? 'unknown'
      const icon = definition ? officialUnitIconsByName[definition.name] ?? 'icon-operation.gif' : 'icon-operation.gif'
      return { key: unit.unitType, name: definition?.name ?? unit.unitType, category, amount: unit.amount ?? 0, icon }
    }).sort((left, right) => categoryOrder[left.category] - categoryOrder[right.category]),
    buildQueues: state.buildings.filter((building) => Boolean(building.upgradeEndsAt)).map((building) => ({ id: building.id, name: buildingNames[building.type] ?? relatedBuildingNames[building.type] ?? building.type, level: building.level, endsAt: building.upgradeEndsAt as string })),
    unreadMessageCount: state.unreadMessageCount ?? 0,
    unreadMailCount: state.unreadMailCount ?? 0,
  }
}
