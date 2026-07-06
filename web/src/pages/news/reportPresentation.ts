// 本文件归口军情战报页面的展示标签、颜色和纯展示判断。
import type { BattleReport, BattleReportDetailData } from '@/types/game'

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

// buildReportShareURL 根据 token 构造公开分享链接，避免暴露内部战报 ID。
export function buildReportShareURL(origin: string, report: Pick<BattleReport, 'id' | 'share' | 'detail'>, token?: string): string {
  const shareToken = token || report.share?.token || report.detail?.share?.token
  return `${origin}/report/${shareToken || report.id}`
}
