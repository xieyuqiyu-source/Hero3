/** 将真实军事视图与增援接口转换为官方调兵遣将表格模型。 */
import type { UnitsConfig } from '../api/types'
import type { GeneralAssignmentState, GeneralState, ReinforcementListItem } from './types'
import { officialCodeForUnitType, type RecruitmentUnitViewModel } from './recruitmentAdapter'

export interface MilitaryWarUnitViewModel {
  id: string
  name: string
  officialCode: number | null
  amount: number
  upkeep: number
}

export interface MilitaryWarArmyRowViewModel {
  id: string
  playerName: string
  faction: string
  generalNames: string
  units: MilitaryWarUnitViewModel[]
  foodPerHour: number
  troops: string
  status: string
  endsAt: string
}

export interface MilitaryWarUnitCatalogItem {
  id: string
  faction: string
  name: string
  officialCode: number | null
  upkeep: number
}

const activeStatuses = new Set(['marching', 'stationed', 'fighting', 'returning'])
const statusLabels: Record<string, string> = { marching: '行军中', stationed: '已驻防', fighting: '战斗中', returning: '返回中' }

/** 判断武将是否被主将槽之外的真实任务占用。 */
export function generalIsAway(generalId: string, assignments: GeneralAssignmentState[]): boolean {
  return assignments.some((assignment) => assignment.generalId === generalId && assignment.id !== 'main' && assignment.slot !== 'main')
}

/** 返回仍驻守本城的当前主将。 */
export function homeGeneral(general: GeneralState | null, assignments: GeneralAssignmentState[]): GeneralState | null {
  return general && !generalIsAway(general.id, assignments) ? general : null
}

/** 保留当前阵营配置的完整兵种列，并直接使用后端已扣除出征兵力后的数量。 */
export function toMilitaryWarUnits(units: RecruitmentUnitViewModel[]): MilitaryWarUnitViewModel[] {
  return units.map((unit) => ({ id: unit.id, name: unit.name, officialCode: unit.officialCode, amount: Math.max(0, unit.owned), upkeep: Math.max(0, unit.stats[5]) }))
}

/** 汇总当前驻城部队的基础口粮消耗。 */
export function militaryWarFoodPerHour(units: MilitaryWarUnitViewModel[]): number {
  return units.reduce((total, unit) => total + unit.amount * unit.upkeep, 0)
}

/** 将全部阵营的真实兵种配置转换为增援军队可复用的列目录。 */
export function toMilitaryWarUnitCatalog(unitsConfig: UnitsConfig): MilitaryWarUnitCatalogItem[] {
  return Object.entries(unitsConfig).flatMap(([faction, units]) => Object.entries(units).map(([id, config]) => ({
    id,
    faction,
    name: config.name?.trim() || id,
    officialCode: officialCodeForUnitType(id),
    upkeep: Math.max(0, Number(config.stats?.upkeep ?? 0)),
  })))
}

/** 按援军所属阵营补全兵种列，同时保留后端新增的未知兵种。 */
function toReinforcementUnits(troopMap: Record<string, number>, faction: string, catalog: MilitaryWarUnitCatalogItem[]): MilitaryWarUnitViewModel[] {
  const configured = catalog.filter((unit) => unit.faction === faction)
  const configuredIds = new Set(configured.map((unit) => unit.id))
  const units = configured.map((unit) => ({ ...unit, amount: Math.max(0, Number(troopMap[unit.id] ?? 0)) }))
  const unknown = Object.entries(troopMap).filter(([id]) => !configuredIds.has(id)).map(([id, amount]) => ({
    id,
    name: catalog.find((unit) => unit.id === id)?.name ?? id,
    officialCode: officialCodeForUnitType(id),
    amount: Math.max(0, Number(amount ?? 0)),
    upkeep: Math.max(0, catalog.find((unit) => unit.id === id)?.upkeep ?? 0),
  }))
  return [...units, ...unknown]
}

/** 将一条真实增援记录转换为不合并的军队行。 */
function toArmyRow(item: ReinforcementListItem, direction: 'incoming' | 'outgoing', catalog: MilitaryWarUnitCatalogItem[]): MilitaryWarArmyRowViewModel {
  const troopMap = Object.keys(item.remainingTroops ?? {}).length ? item.remainingTroops ?? {} : item.troops ?? {}
  const faction = item.fromPlayerFaction ?? ''
  const units = toReinforcementUnits(troopMap, faction, catalog)
  const troops = units.filter((unit) => unit.amount > 0).map((unit) => `${unit.name}×${unit.amount.toLocaleString('zh-CN')}`).join('、') || '无剩余兵力'
  const generalNames = (item.generals ?? []).map((general) => `${general.name?.trim() || general.id}${general.level ? `(Lv ${general.level})` : ''}`).join('、') || '无将领'
  return {
    id: item.reinforcementId,
    playerName: direction === 'incoming' ? (item.fromPlayerName || item.fromPlayerId) : (item.toPlayerName || item.toPlayerId),
    faction,
    generalNames,
    units,
    foodPerHour: militaryWarFoodPerHour(units),
    troops,
    status: statusLabels[item.status] ?? item.status,
    endsAt: item.status === 'returning' ? (item.expectedReturnedAt || '') : item.status === 'marching' ? (item.arriveAt || '') : '',
  }
}

/** 映射真实来援与出援列表，保留同来源的多个独立批次。 */
export function toMilitaryWarReinforcements(items: ReinforcementListItem[], direction: 'incoming' | 'outgoing', catalog: MilitaryWarUnitCatalogItem[] = []): MilitaryWarArmyRowViewModel[] {
  return items.filter((item) => activeStatuses.has(item.status)).map((item) => toArmyRow(item, direction, catalog)).sort((left, right) => left.endsAt.localeCompare(right.endsAt) || left.id.localeCompare(right.id))
}
