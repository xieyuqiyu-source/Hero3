// 本文件归口军情战报页面的展示标签、颜色和纯展示判断。
import type { BattleReport, BattleReportDetailData, BattleReportExtra, BattleReportRewards, BattleReportSide, BattleReportTrait } from '@/types/game'

export const REPORT_VIEW_TABS = [
  { key: 'all', label: '全部' },
  { key: 'attack', label: '进攻' },
  { key: 'defense', label: '防守' },
  { key: 'reinforcement', label: '协防' },
  { key: 'scout', label: '侦查' },
  { key: 'sweep', label: '扫荡' },
]

export const REPORT_VIEW_CONFIG: Record<string, { label: string; color: string }> = {
  attack: { label: '进攻', color: 'text-red-600 bg-red-500/10' },
  defense: { label: '防守', color: 'text-blue-600 bg-blue-500/10' },
  reinforcement: { label: '协防', color: 'text-green-600 bg-green-500/10' },
  scout: { label: '侦查', color: 'text-yellow-600 bg-yellow-500/10' },
  system: { label: '系统', color: 'text-slate-600 bg-slate-500/10' },
}

export const REPORT_SOURCE_CONFIG: Record<string, { label: string; color: string }> = {
  npc_city: { label: 'NPC', color: 'text-cyan-600 bg-cyan-500/10' },
  player_city: { label: '玩家', color: 'text-pink-600 bg-pink-500/10' },
  yellow_turban: { label: '黄巾', color: 'text-yellow-700 bg-yellow-500/10' },
  stronghold: { label: '据点', color: 'text-amber-600 bg-amber-500/10' },
  dungeon: { label: '副本', color: 'text-purple-600 bg-purple-500/10' },
  resource_point: { label: '资源点', color: 'text-emerald-600 bg-emerald-500/10' },
  event_target: { label: '活动', color: 'text-fuchsia-600 bg-fuchsia-500/10' },
  world_boss: { label: 'Boss', color: 'text-rose-600 bg-rose-500/10' },
}

export const REPORT_OUTCOME_CONFIG: Record<string, { label: string; color: string }> = {
  victory: { label: '胜', color: 'text-emerald-600' },
  defeat: { label: '败', color: 'text-red-600' },
  draw: { label: '平', color: 'text-slate-500' },
  intel_success: { label: '侦查成功', color: 'text-emerald-600' },
  intel_failed: { label: '侦查失败', color: 'text-red-600' },
  notice: { label: '通知', color: 'text-amber-600' },
}

// uniqueReportsById 按后端战报 ID 去重，保留首次出现的权威记录和顺序。
export function uniqueReportsById<T extends Pick<BattleReport, 'id'>>(reports: T[]): T[] {
  const seen = new Set<string>()
  return reports.filter((report) => !seen.has(report.id) && Boolean(seen.add(report.id)))
}

// buildReportListParams 将军情 Tab 转换成后端筛选参数。
export function buildReportListParams(activeTab: string): { viewType?: string; battleType?: string } | undefined {
  if (activeTab === 'all') return undefined
  if (activeTab === 'scout') return { battleType: 'scout' }
  if (activeTab === 'sweep') return { battleType: 'sweep' }
  return { viewType: activeTab }
}

// resolveReportOutcome 统一返回当前玩家结果，兼容旧战报 result。
export function resolveReportOutcome(report: Pick<BattleReport, 'ownerOutcome' | 'viewType' | 'result' | 'battleType' | 'type'>): string {
  if (report.ownerOutcome) return report.ownerOutcome
  if (report.battleType === 'scout' || report.type === 'scout') {
    return report.result === 'attacker_victory' ? 'intel_success' : 'intel_failed'
  }
  if (report.result === 'draw') return 'draw'
  const viewType = report.viewType || 'attack'
  if (viewType === 'defense' || viewType === 'reinforcement') {
    return report.result === 'defender_victory' ? 'victory' : 'defeat'
  }
  return report.result === 'attacker_victory' ? 'victory' : 'defeat'
}

// legacyTraitDisplaySide 按旧战报的绝对角色和将领快照恢复标准详情位置。
function legacyTraitDisplaySide(report: BattleReport, ownerSide?: string, generalId?: string): string {
  const normalizedOwner = ownerSide?.trim().toLowerCase()
  if (normalizedOwner === 'attacker' || normalizedOwner === 'primary') return 'primary'
  if (normalizedOwner === 'defender' || normalizedOwner === 'secondary') return 'secondary'
  if (normalizedOwner === 'reinforcement') return report.viewType === 'reinforcement' ? 'primary' : 'reinforcement'

  if (generalId) {
    if ((report.pvpAttackerGenerals ?? []).some((general) => general.id === generalId)) return 'primary'
    if ((report.pvpDefenderGenerals ?? []).some((general) => general.id === generalId)) return 'secondary'
    if ((report.pvpReinforcements ?? []).some((item) => (item.generals ?? []).some((general) => general.id === generalId))) {
      return report.viewType === 'reinforcement' ? 'primary' : 'reinforcement'
    }
  }

  if (report.ownerSide === 'defender' || report.viewType === 'defense') return 'secondary'
  if (report.ownerSide === 'reinforcement' || report.viewType === 'reinforcement') return 'primary'
  return 'primary'
}

// legacyTraitOwnerRole 把旧结果的绝对触发方恢复为参战角色。
function legacyTraitOwnerRole(report: BattleReport, ownerSide: string | undefined, displaySide: string): string {
  const normalizedOwner = ownerSide?.trim().toLowerCase()
  if (normalizedOwner === 'attacker' || normalizedOwner === 'defender' || normalizedOwner === 'reinforcement') return normalizedOwner
  if (displaySide === 'secondary') return 'defender'
  if (displaySide === 'reinforcement' || report.viewType === 'reinforcement') return 'reinforcement'
  return 'attacker'
}

// legacyTraitGeneralName 从旧主战和援军快照恢复触发将领名称。
function legacyTraitGeneralName(report: BattleReport, generalId?: string): string | undefined {
  if (!generalId) return undefined
  for (const general of [...(report.pvpAttackerGenerals ?? []), ...(report.pvpDefenderGenerals ?? [])]) {
    if (general.id === generalId) return general.name
  }
  for (const reinforcement of report.pvpReinforcements ?? []) {
    const general = (reinforcement.generals ?? []).find((item) => item.id === generalId)
    if (general) return general.name
  }
  return undefined
}

// buildLegacyReportTraits 按旧触发顺序恢复标准特性时间线，不从拥有快照推测触发。
export function buildLegacyReportTraits(report: BattleReport): BattleReportTrait[] {
  return (report.traitTriggered ?? []).map((storageKey) => {
    const outcome = report.traitOutcomes?.[storageKey]
    const traitId = outcome?.traitId?.trim() || storageKey
    const generalId = outcome?.ownerGeneralId
    const ownerSide = legacyTraitDisplaySide(report, outcome?.ownerSide, generalId)
    return {
      traitId,
      traitName: outcome?.name || traitId,
      ownerSide,
      ownerRole: legacyTraitOwnerRole(report, outcome?.ownerSide, ownerSide),
      ownerPlayerId: outcome?.ownerPlayerId,
      generalId,
      generalName: legacyTraitGeneralName(report, generalId),
      detail: outcome?.detail,
    }
  })
}

// buildLegacyBattleReportDetail 为没有标准 detail 的历史战报恢复完整可展示结构。
export function buildLegacyBattleReportDetail(report: BattleReport): BattleReportDetailData {
  const outcome = resolveReportOutcome(report)
  const viewType = report.viewType || (report.type === 'reinforce' ? 'reinforcement' : report.type === 'scout' ? 'scout' : 'attack')
  const hasPvpExtra = (report.pvpReinforcements?.length ?? 0) > 0 || Object.keys(report.pvpReinforcementLosses ?? {}).length > 0 || Boolean(report.pvpWall)
  const reinforcementGenerals = (report.pvpReinforcements ?? []).find((item) => item.fromPlayerId === (report.ownerPlayerId || report.playerId))?.generals
    ?? report.pvpReinforcements?.[0]?.generals
  const ownerGenerals = viewType === 'defense'
    ? report.pvpDefenderGenerals
    : viewType === 'reinforcement'
      ? reinforcementGenerals
      : report.pvpAttackerGenerals
  const targetGenerals = viewType === 'defense' ? report.pvpAttackerGenerals : report.pvpDefenderGenerals
  const ownerParticipant: BattleReportSide = {
    role: viewType === 'defense' ? 'defender' : viewType === 'reinforcement' ? 'reinforcement' : 'attacker',
    playerId: report.playerId,
    playerName: report.playerName,
    cityName: report.playerName,
    faction: report.playerFaction,
    power: report.playerPower,
    generals: ownerGenerals,
    units: Object.entries(report.dispatchedUnits ?? {}).map(([unitType, dispatched]) => ({
      unitType,
      amountBefore: dispatched,
      dispatched,
      lost: report.lostUnits?.[unitType] ?? 0,
      survived: report.survivedUnits?.[unitType] ?? Math.max(0, dispatched - (report.lostUnits?.[unitType] ?? 0)),
    })),
  }
  const targetParticipant: BattleReportSide = {
    role: viewType === 'defense' ? 'attacker' : 'defender',
    targetId: report.targetId,
    targetName: report.targetName,
    cityName: report.targetName,
    faction: report.defenderFaction,
    power: report.enemyPower,
    generals: targetGenerals,
    units: Object.entries(report.defenderUnits ?? {}).map(([unitType, dispatched]) => ({
      unitType,
      amountBefore: dispatched,
      dispatched,
      lost: report.defenderLostUnits?.[unitType] ?? 0,
      survived: Math.max(0, dispatched - (report.defenderLostUnits?.[unitType] ?? 0)),
    })),
    resources: report.defenderResources,
  }
  const primarySide = viewType === 'defense' ? targetParticipant : ownerParticipant
  const secondarySide = viewType === 'reinforcement' ? undefined : viewType === 'defense' ? ownerParticipant : targetParticipant
  return {
    id: report.id,
    eventId: report.eventId,
    ownerPlayerId: report.ownerPlayerId || report.playerId,
    viewType,
    viewLabel: viewType === 'defense' ? '防守' : viewType === 'reinforcement' ? '协防' : viewType === 'scout' ? '侦查' : '进攻',
    sourceType: report.sourceType || 'npc_city',
    sourceLabel: report.sourceType || 'NPC',
    battleType: report.battleType || report.type,
    result: report.result,
    winnerSide: report.winnerSide,
    ownerSide: report.ownerSide,
    ownerOutcome: outcome,
    title: report.title || `${report.playerName || '我方'} ${report.type} ${report.targetName || report.targetId}`,
    summary: report.summary,
    occurredAt: report.createdAt,
    primarySide,
    secondarySide,
    rewards: {
      resources: report.rewards,
      drops: report.drops,
      cityGold: report.overflowCityGold,
      generalExp: report.generalExpGained,
      generalLevelBefore: report.generalLevelBefore,
      generalLevelAfter: report.generalLevelAfter,
      overflow: report.overflow,
    },
    traits: buildLegacyReportTraits(report),
    visibility: {
      showEnemyRemainingUnits: report.defenderRevealed,
      showEnemyResources: report.defenderRevealed,
      showEnemyGenerals: report.defenderRevealed,
      showEnemyCityDefense: report.defenderRevealed,
    },
    extra: hasPvpExtra ? {
      pvp: {
        reinforcements: report.pvpReinforcements ?? [],
        reinforcementLosses: report.pvpReinforcementLosses ?? {},
        wall: report.pvpWall,
      },
    } : undefined,
    read: report.read,
    share: report.share,
  }
}

// reportExtraRecord 只把普通对象视为可合并的战报扩展分区。
function reportExtraRecord(value: unknown): Record<string, unknown> | undefined {
  return value && typeof value === 'object' && !Array.isArray(value) ? value as Record<string, unknown> : undefined
}

// mergeBattleReportExtra 深合并 PVP 扩展，避免部分标准字段覆盖旧援军快照。
function mergeBattleReportExtra(legacy?: BattleReportExtra, standard?: BattleReportExtra): BattleReportExtra | undefined {
  if (!legacy && !standard) return undefined
  const extra: BattleReportExtra = { ...legacy, ...standard }
  const legacyPvp = reportExtraRecord(legacy?.pvp)
  const standardPvp = reportExtraRecord(standard?.pvp)
  if (legacyPvp || standardPvp) extra.pvp = { ...legacyPvp, ...standardPvp }
  return extra
}

// mergeBattleReportRewards 按字段补齐部分迁移奖励，标准零值和空列表仍保持权威。
function mergeBattleReportRewards(legacy: BattleReportRewards, standard?: BattleReportRewards): BattleReportRewards {
  return {
    resources: standard?.resources ?? legacy.resources,
    drops: standard?.drops ?? legacy.drops,
    cityGold: standard?.cityGold ?? legacy.cityGold,
    generalExp: standard?.generalExp ?? legacy.generalExp,
    generalLevelBefore: standard?.generalLevelBefore ?? legacy.generalLevelBefore,
    generalLevelAfter: standard?.generalLevelAfter ?? legacy.generalLevelAfter,
    overflow: standard?.overflow ?? legacy.overflow,
  }
}

// normalizeBattleReportDetail 合并标准详情和历史字段，标准真实时间线优先、缺失时才回退旧结果。
export function normalizeBattleReportDetail(report: BattleReport): BattleReportDetailData {
  const legacy = buildLegacyBattleReportDetail(report)
  const detail = report.detail ?? legacy
  const legacyTraits = legacy.traits ?? []
  const traits = (detail.traits?.length ?? 0) > 0 || legacyTraits.length === 0 ? detail.traits : legacyTraits
  const extra = mergeBattleReportExtra(legacy.extra, detail.extra)
  return {
    ...detail,
    ownerOutcome: detail.ownerOutcome || report.ownerOutcome || resolveReportOutcome(report),
    winnerSide: detail.winnerSide || report.winnerSide,
    ownerSide: detail.ownerSide || report.ownerSide,
    traits,
    rewards: mergeBattleReportRewards(legacy.rewards, detail.rewards),
    extra,
    share: detail.share || report.share,
  }
}

// shouldRenderSecondarySide 判断详情页是否展示下半部分阵营。
export function shouldRenderSecondarySide(detail: Pick<BattleReportDetailData, 'visibility'> & Partial<Pick<BattleReportDetailData, 'secondarySide'>>): boolean {
  return Boolean(detail.secondarySide && detail.visibility.showEnemyRemainingUnits)
}

// reportTotalPages 计算战报分页总页数，空列表保持 1 页用于稳定 UI。
export function reportTotalPages(totalReports: number, pageSize: number): number {
  if (pageSize <= 0) return 1
  return Math.max(1, Math.ceil(totalReports / pageSize))
}

// shouldShowEmptyReports 判断军情页是否展示空列表状态。
export function shouldShowEmptyReports(totalReports: number, loading: boolean): boolean {
  return totalReports === 0 && !loading
}

// hasStandardUnitRows 判断标准详情侧边是否具备固定兵种行。
export function hasStandardUnitRows(side: Pick<BattleReportDetailData['primarySide'], 'units'>): boolean {
  return side.units.every((unit) => (
    typeof unit.unitType === 'string' &&
    typeof unit.dispatched === 'number' &&
    typeof unit.lost === 'number' &&
    typeof unit.survived === 'number'
  ))
}

// hasTraitEntries 判断标准详情是否包含可展示的特性触发。
export function hasTraitEntries(detail: Pick<BattleReportDetailData, 'traits'>): boolean {
  return (detail.traits ?? []).some((trait) => Boolean(trait.traitId || trait.traitName || trait.summary))
}

// reportTraitRenderKey 为同 ID、同将领的多条真实特性结果生成稳定且唯一的渲染键。
export function reportTraitRenderKey(trait: BattleReportTrait, index: number): string {
  return [trait.traitId, trait.ownerRole, trait.ownerPlayerId, trait.generalId, index]
    .map((value) => value ?? '')
    .join(':')
}

// buildReportShareURL 根据 token 构造公开分享链接，避免暴露内部战报 ID。
export function buildReportShareURL(origin: string, report: Pick<BattleReport, 'id' | 'share' | 'detail'>, token?: string): string {
  const shareToken = token || report.share?.token || report.detail?.share?.token
  return shareToken ? `${origin}/report/${shareToken}` : ''
}

// isReportShareToken 判断路由参数是否为公开分享 token，而不是内部战报 ID。
export function isReportShareToken(value: string): boolean {
  return /^br_[0-9a-f]{48}$/i.test(value.trim())
}

// resolveReportTraitDisplaySide 兼容旧黄巾防守战报把无归属守方特性误写成 primary。
export function resolveReportTraitDisplaySide(
  detail: Pick<BattleReportDetailData, 'sourceType' | 'viewType'>,
  trait: Pick<BattleReportTrait, 'ownerSide' | 'ownerRole' | 'generalId'>,
): string | undefined {
  if (detail.sourceType === 'yellow_turban'
    && detail.viewType === 'defense'
    && trait.ownerSide === 'primary'
    && !trait.ownerRole
    && !trait.generalId) {
    return 'secondary'
  }
  return trait.ownerSide
}

// reportTraitBelongsToSide 按明确归属优先匹配主战方，避免把防守特性复制到同将领援军区。
export function reportTraitBelongsToSide(
  detail: Pick<BattleReportDetailData, 'sourceType' | 'viewType'>,
  trait: Pick<BattleReportTrait, 'ownerSide' | 'ownerRole' | 'ownerPlayerId' | 'generalId'>,
  side: Pick<BattleReportSide, 'role' | 'playerId' | 'generals'>,
  sideKey: 'primary' | 'secondary' | 'reinforcement',
): boolean {
  const ownerSide = resolveReportTraitDisplaySide(detail, trait)
  if (trait.ownerRole && trait.ownerRole !== side.role) return false
  if (side.role === 'reinforcement') {
    if (trait.ownerPlayerId) return trait.ownerPlayerId === side.playerId
    if (trait.generalId) return (side.generals ?? []).some((general) => general.id === trait.generalId)
    if (trait.ownerRole) return true
  }
  if (trait.ownerRole) return true
  if (ownerSide) {
    const normalizedOwner = ownerSide === 'attacker' ? 'primary' : ownerSide === 'defender' ? 'secondary' : ownerSide
    if (normalizedOwner === 'reinforcement') {
      if (trait.ownerPlayerId) return trait.ownerPlayerId === side.playerId
      if (trait.generalId) return (side.generals ?? []).some((general) => general.id === trait.generalId)
    }
    return normalizedOwner === sideKey
  }
  if (trait.generalId) return (side.generals ?? []).some((general) => general.id === trait.generalId)
  return sideKey === 'primary'
}
