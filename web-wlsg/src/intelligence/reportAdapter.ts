/** 将 Hero3 标准战报详情映射为武林三国官方战报版式。 */
import type { BattleReportDetailState, BattleReportDropState, BattleReportGeneralState, BattleReportSideState, BattleReportState, BattleReportUnitState } from '../game/types'
import { officialCodeForUnitType } from '../game/recruitmentAdapter'
import { traitLabel } from '../game/traitLabels'
import { formatIntelligenceTime } from './adapter'

export interface OfficialReportUnitViewModel {
  key: string
  name: string
  icon: string | null
  dispatched: number
  lost: number
  survived: number
}

export interface OfficialReportDropViewModel {
  key: string
  name: string
  amount: number
}

export interface OfficialReportTraitViewModel {
  key: string
  name: string
  phase: string
  detailText: string
}

export interface OfficialReportSideViewModel {
  key: string
  role: 'attacker' | 'defender' | 'reinforcement'
  roleLabel: string
  name: string
  faction: string
  factionIcon: string
  general: BattleReportGeneralState | null
  showGeneralPlaceholder: boolean
  units: OfficialReportUnitViewModel[]
  power: number
  result: 'victory' | 'defeat' | 'unknown'
  resultLabel: string
  traitText: string
  traits: OfficialReportTraitViewModel[]
  generalExp: number | null
}

export interface OfficialBattleReportViewModel {
  title: string
  occurredAt: string
  summary: string
  sides: OfficialReportSideViewModel[]
  visibilityReason: string
  resourceText: string
  feedbackText: string
  dropsText: string
  dropItems: OfficialReportDropViewModel[]
  cityGold: number
  wallText: string
}

const factionIndex: Record<string, number> = { wei: 1, shu: 2, wu: 3 }
const factionUnitTypes: Record<string, string[]> = {
  wei: ['qingZhouArmy', 'jinWeiSoldier', 'huWei', 'zhanYingTanMa', 'qiQiYing', 'huBaoQi', 'chongZhuangChe', 'luLeiChe', 'jianzhuShi', 'tuZu'],
  shu: ['greedyWolf', 'qilinGuard', 'azureDragon', 'flyingKite', 'xiLiangCavalry', 'southernElephant', 'siegeTower', 'thunderBolt', 'woodenOx', 'hanRoyalty'],
  wu: ['shadowGuard', 'xiuLuo', 'secretAgent', 'divineWind', 'zhuQueRider', 'overlordRider', 'chongChe', 'juShiChe', 'fengShuiMaster', 'taiPingShi'],
}
const resourceLabels: Record<string, string> = { wood: '木材', clay: '泥土', stone: '泥土', iron: '铁矿', food: '粮食', cityGold: '城金' }
const traitDetailLabels: Record<string, string> = {
  effectRate: '效果比例', triggerChance: '触发概率', damagePercent: '伤害比例', suppressRate: '压制比例', foodRatio: '口粮比例',
  preBattleAffected: '战前损失', suppressedUnits: '压制兵力', capturedUnits: '俘虏兵力', modifiedUnits: '攻防修正',
  extraLosses: '追加损失', targetExtraLosses: '追加损失', reducedLosses: '减少损失', disabledTraits: '禁用特性',
  revivedUnits: '复活兵力', returnedUnits: '返还兵力', extraDamage: '额外伤害', totalRevived: '复活总数',
  totalSuppressed: '压制总数', totalCaptured: '俘虏总数', disabledTraitCount: '禁用特性数量',
  plunderDelta: '掠夺资源修正',
}

/** 把资源键值表转换为官方横排行文。 */
function amountLine(values?: Record<string, number>) {
  const entries = Object.entries(values ?? {}).filter(([, amount]) => Number.isFinite(amount))
  return entries.length ? entries.map(([key, amount]) => `${resourceLabels[key] ?? key}:${Math.max(0, amount).toLocaleString('zh-CN')}`).join('　') : '无'
}

/** 把后端情报脱敏状态码转换为玩家可读提示，并保留已经是中文的业务说明。 */
function visibilityReasonText(reason?: string, threshold?: number) {
  const normalized = (reason || '').trim()
  if (normalized === 'enemy_remaining_hidden') {
    const percent = Math.round(Math.max(0, threshold || 0.25) * 100)
    return `对防守方造成的战损不足 ${percent}%，无法获得防守方剩余兵力情报`
  }
  if (normalized === 'scout_failed') return '侦查失败，无法获得防守方情报'
  if (!normalized || /^[a-z0-9_]+$/i.test(normalized)) return '当前战报未公开防守方详细情报'
  return normalized
}

/** 按物品 ID 优先、名称兜底合并同类掉落，避免同一宝物重复展示。 */
function mergeDrops(drops?: BattleReportDropState[]): OfficialReportDropViewModel[] {
  const merged = new Map<string, OfficialReportDropViewModel>()
  for (const drop of drops ?? []) {
    const name = drop.name || drop.itemId || drop.type || '未知物品'
    const key = drop.itemId ? `id:${drop.itemId}` : `name:${name}`
    const current = merged.get(key)
    if (current) current.amount += Math.max(0, drop.amount || 0)
    else merged.set(key, { key, name, amount: Math.max(0, drop.amount || 0) })
  }
  return [...merged.values()]
}

/** 判断特性详情值是否为普通键值对象。 */
function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value)
}

/** 从双方战报快照建立兵种 ID 到中文名映射。 */
function reportUnitNames(detail: BattleReportDetailState) {
  const names = new Map<string, string>()
  for (const side of [detail.primarySide, detail.secondarySide].filter(Boolean) as BattleReportSideState[]) {
    for (const unit of side.units ?? []) names.set(unit.unitType, unit.unitName || unit.unitType)
  }
  return names
}

/** 根据特性实际结果字段判断它发生在哪个战斗阶段。 */
function traitPhase(detail: Record<string, unknown>) {
  const keys = new Set(Object.keys(detail))
  if (['preBattleAffected', 'suppressedUnits', 'capturedUnits', 'modifiedUnits', 'totalSuppressed', 'totalCaptured'].some((key) => keys.has(key))) return '战斗前'
  if (['extraLosses', 'targetExtraLosses', 'reducedLosses', 'disabledTraits', 'extraDamage'].some((key) => keys.has(key))) return '战斗结算后'
  if (['revivedUnits', 'returnedUnits', 'totalRevived'].some((key) => keys.has(key))) return '战斗结束后'
  if (keys.has('plunderDelta')) return '掠夺结算'
  return '战斗触发'
}

/** 把比例、数量和逐兵种变化完整格式化为官方战报的一行说明。 */
function formatTraitDetail(detail: Record<string, unknown>, unitNames: Map<string, string>) {
  const orderedKeys = Object.keys(detail).sort((left, right) => {
    const rank = (key: string) => key === 'effectRate' ? 0 : isRecord(detail[key]) ? 1 : key === 'triggerChance' ? 3 : 2
    return rank(left) - rank(right)
  })
  return orderedKeys.flatMap((key) => {
    const value = detail[key]
    const label = traitDetailLabels[key] || key
    if (typeof value === 'number' && Number.isFinite(value)) {
      if (/Rate$|Chance$|Percent$|Ratio$/i.test(key)) return [`${label}：${Math.round(value * 10000) / 100}%`]
      return [`${label}：${value.toLocaleString('zh-CN')}`]
    }
    if (typeof value === 'string' && value) return [`${label}：${value}`]
    if (isRecord(value)) {
      const entries = Object.entries(value).flatMap(([itemKey, amount]) => typeof amount === 'number' && Number.isFinite(amount)
        ? [`${unitNames.get(itemKey) || resourceLabels[itemKey] || traitDetailLabels[itemKey] || itemKey} ${amount > 0 ? '+' : ''}${amount.toLocaleString('zh-CN')}`]
        : [])
      return entries.length ? [`${label}：${entries.join('、')}`] : []
    }
    return []
  }).join('；')
}

/** 标准化一方的十兵种顺序并保留未知扩展兵种。 */
function mapUnits(side: BattleReportSideState): OfficialReportUnitViewModel[] {
  const byType = new Map((side.units ?? []).map((unit) => [unit.unitType, unit]))
  const orderedKeys = [...(factionUnitTypes[side.faction ?? ''] ?? []), ...Array.from(byType.keys()).filter((key) => !(factionUnitTypes[side.faction ?? ''] ?? []).includes(key))]
  return orderedKeys.map((key) => {
    const unit: BattleReportUnitState | undefined = byType.get(key)
    const code = officialCodeForUnitType(key)
    return { key, name: unit?.unitName || key, icon: code ? `/assets/official/report/units/${code}.gif` : null, dispatched: Math.max(0, unit?.dispatched ?? unit?.amountBefore ?? 0), lost: Math.max(0, unit?.lost ?? 0), survived: Math.max(0, unit?.survived ?? 0) }
  })
}

/** 根据战斗胜方解析一方的胜负印记。 */
function sideResult(role: OfficialReportSideViewModel['role'], winnerSide?: string) {
  if (!winnerSide || winnerSide === 'draw' || winnerSide === 'none') return { result: 'unknown' as const, resultLabel: '平' }
  const won = role === 'attacker' ? winnerSide === 'attacker' : winnerSide === 'defender'
  return { result: won ? 'victory' as const : 'defeat' as const, resultLabel: won ? '胜' : '败' }
}

/** 把标准战报一方转换为官方参战卡片。 */
function mapSide(side: BattleReportSideState, index: number, winnerSide: string | undefined, traits: OfficialReportTraitViewModel[], generalExp: number | null, forcedRole?: OfficialReportSideViewModel['role'], forceGeneralPlaceholder = false): OfficialReportSideViewModel {
  const role = forcedRole ?? (side.role === 'attacker' ? 'attacker' : side.role === 'reinforcement' ? 'reinforcement' : 'defender')
  const result = sideResult(role, winnerSide)
  const general = side.generals?.[0] ?? null
  const generalTraitText = (general?.traits ?? []).map((trait) => traitLabel(trait.traitId, trait.traitName || trait.name)).filter(Boolean).join('；')
  const showGeneralPlaceholder = role === 'defender' && (forceGeneralPlaceholder || side.targetType === 'npc_city')
  const displayTraits = traits.length ? traits : (general?.traits ?? []).map((trait) => ({ key: trait.traitId, name: traitLabel(trait.traitId, trait.traitName || trait.name), phase: '', detailText: '' }))
  return { key: `${role}-${side.playerId || side.targetId || index}`, role, roleLabel: role === 'attacker' ? '进攻方' : role === 'reinforcement' ? '增援方' : '防守方', name: side.cityName || side.playerName || side.targetName || '未知城池', faction: side.faction || 'wei', factionIcon: `/assets/official/report/country_${factionIndex[side.faction || ''] ?? 1}.gif`, general, showGeneralPlaceholder, units: mapUnits(side), power: Math.max(0, side.power || 0), result: result.result, resultLabel: result.resultLabel, traitText: displayTraits.map((trait) => [trait.name, trait.phase, trait.detailText].filter(Boolean).join(' ')).join('；') || generalTraitText, traits: displayTraits, generalExp }
}

/** 按武将 ID、名称和参战方归属筛选真实触发特性，并展开阶段与详细数据。 */
function sideTriggeredTraits(detail: BattleReportDetailState, side: BattleReportSideState, primary: boolean, outcomes: BattleReportState['traitOutcomes']): OfficialReportTraitViewModel[] {
  const general = side.generals?.[0]
  const unitNames = reportUnitNames(detail)
  return (detail.traits ?? []).filter((trait) => {
    const outcome = outcomes?.[trait.traitId]
    const generalId = trait.generalId || outcome?.ownerGeneralId
    const ownerSide = outcome?.ownerSide || trait.ownerSide
    if (generalId) return generalId === general?.id
    if (trait.generalName) return trait.generalName === general?.name
    if (general) return false
    if (ownerSide === 'primary') return primary
    if (ownerSide === 'secondary') return !primary
    return ownerSide === side.role || trait.ownerRole === side.role
  }).map((trait, index) => {
    const outcome = outcomes?.[trait.traitId]
    const detailData = isRecord(outcome?.detail) ? outcome.detail : isRecord(trait.detail) ? trait.detail : {}
    return { key: `${trait.traitId}-${index}`, name: traitLabel(trait.traitId, outcome?.name || trait.traitName || trait.summary), phase: traitPhase(detailData), detailText: formatTraitDetail(detailData, unitNames) }
  })
}

/** 从兼容旧字段构造最小标准详情，保证历史战报仍可展示。 */
function legacyDetail(report: BattleReportState): BattleReportDetailState {
  const units = (values?: Record<string, number>, losses?: Record<string, number>) => Object.keys({ ...(values ?? {}), ...(losses ?? {}) }).map((unitType) => ({ unitType, amountBefore: values?.[unitType] ?? 0, dispatched: values?.[unitType] ?? 0, lost: losses?.[unitType] ?? 0, survived: Math.max(0, (values?.[unitType] ?? 0) - (losses?.[unitType] ?? 0)) }))
  return { id: report.id, viewType: report.viewType || 'attack', sourceType: report.sourceType || 'player_city', battleType: report.battleType || report.type, result: report.result, winnerSide: report.result === 'attacker_victory' ? 'attacker' : report.result === 'defender_victory' ? 'defender' : 'none', title: report.title || `${report.playerName || '我方'} 攻击 ${report.targetName || '目标'}`, summary: report.summary, occurredAt: report.createdAt, primarySide: { role: 'attacker', playerName: report.playerName, cityName: report.playerName, faction: report.playerFaction, power: 0, units: units(report.dispatchedUnits, report.lostUnits) }, secondarySide: report.defenderRevealed ? { role: 'defender', playerName: report.targetName, cityName: report.targetName, faction: report.defenderFaction, power: 0, units: units(report.defenderUnits, report.defenderLostUnits), resources: report.defenderResources } : null, rewards: { resources: report.rewards, drops: report.drops, generalExp: report.generalExpGained }, traits: (report.traitTriggered ?? []).map((name) => ({ traitId: name, traitName: name })), visibility: { showEnemyRemainingUnits: Boolean(report.defenderRevealed), showEnemyResources: Boolean(report.defenderRevealed), showEnemyGenerals: Boolean(report.defenderRevealed), showEnemyCityDefense: Boolean(report.defenderRevealed), reason: report.defenderRevealed ? '' : '对防守方造成的损伤不足，因此无法准确获得防守方的战斗情报' }, read: report.read }
}

/** 把一份真实战报转换为完整官方战报视图模型。 */
export function toOfficialBattleReport(report: BattleReportState): OfficialBattleReportViewModel {
  const detail = report.detail ?? legacyDetail(report)
  const sides = [mapSide(detail.primarySide, 0, detail.winnerSide, sideTriggeredTraits(detail, detail.primarySide, true, report.traitOutcomes), Math.max(0, detail.rewards.generalExp ?? 0))]
  const enemyVisible = detail.viewType === 'defense' || detail.visibility.showEnemyRemainingUnits
  if (detail.secondarySide && enemyVisible) {
    const npcDefender = detail.sourceType === 'npc_city' || /NPC/i.test(detail.secondarySide.targetName || detail.secondarySide.cityName || '')
    sides.push(mapSide(detail.secondarySide, 1, detail.winnerSide, sideTriggeredTraits(detail, detail.secondarySide, false, report.traitOutcomes), null, undefined, npcDefender))
  }
  for (const [index, reinforcement] of (enemyVisible ? (report.pvpReinforcements ?? []) : []).entries()) {
    const losses = report.pvpReinforcementLosses?.[reinforcement.reinforcementId] ?? {}
    const reinforcementSide: BattleReportSideState = { role: 'reinforcement', playerId: reinforcement.fromPlayerId, playerName: reinforcement.fromPlayerName, cityName: reinforcement.fromPlayerName, faction: reinforcement.faction, power: 0, generals: reinforcement.generals, units: Object.keys({ ...reinforcement.troops, ...losses }).map((unitType) => ({ unitType, amountBefore: reinforcement.troops[unitType] ?? 0, dispatched: reinforcement.troops[unitType] ?? 0, lost: losses[unitType] ?? 0, survived: Math.max(0, (reinforcement.troops[unitType] ?? 0) - (losses[unitType] ?? 0)) })) }
    sides.push(mapSide(reinforcementSide, index + 2, detail.winnerSide, sideTriggeredTraits(detail, reinforcementSide, false, report.traitOutcomes), null, 'reinforcement'))
  }
  const resources = detail.rewards.resources ?? report.rewards
  const drops = detail.rewards.drops ?? report.drops
  const dropItems = mergeDrops(drops)
  const wall = report.pvpWall
  return { title: detail.title || report.title || '未命名战报', occurredAt: formatIntelligenceTime(detail.occurredAt || report.createdAt), summary: detail.summary || report.summary || '', sides, visibilityReason: enemyVisible ? '' : visibilityReasonText(detail.visibility.reason, detail.visibility.threshold), resourceText: amountLine(resources), feedbackText: '-', dropsText: dropItems.length ? dropItems.map((drop) => `${drop.name} ${drop.amount.toLocaleString('zh-CN')} 个`).join('、') : '无', dropItems, cityGold: Math.max(0, detail.rewards.cityGold ?? 0), wallText: wall ? `城墙等级 Lv ${wall.level}　防御加成 ${(wall.totalDefenseBonus * 100).toFixed(1)}%${wall.hardness ? `　硬度 ${wall.hardness}` : ''}` : '' }
}
