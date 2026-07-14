/** 军情适配层集中维护标签、后端筛选和真实战报列表字段映射。 */
import type { BattleReportState, IntelligenceReportViewModel, IntelligenceTabKey } from '../game/types'

export const intelligenceTabs: Array<{ key: IntelligenceTabKey; label: string }> = [
  { key: 'all', label: '最新' },
  { key: 'attack', label: '攻击' },
  { key: 'defense', label: '防守' },
  { key: 'reinforcement', label: '军事增援' },
  { key: 'scout', label: '侦查' },
]

const reportTypeLabels: Record<string, string> = {
  attack: '攻 击',
  defense: '防 守',
  reinforcement: '增 援',
  scout: '侦 查',
}

/** 将页面标签转换为后端已支持的视角或战斗类型筛选。 */
export function intelligenceFilter(tab: IntelligenceTabKey): { viewType?: string; battleType?: string } | undefined {
  if (tab === 'all') return undefined
  if (tab === 'scout') return { battleType: 'scout' }
  return { viewType: tab }
}

/** 解析战报所属页面分类，未知类型保留通用兜底。 */
export function resolveIntelligenceType(report: BattleReportState): string {
  if (report.battleType === 'scout' || report.type === 'scout') return 'scout'
  if (report.viewType === 'reinforcement' || report.type === 'reinforce') return 'reinforcement'
  if (report.viewType === 'defense' || report.type === 'defense') return 'defense'
  if (report.viewType === 'attack' || report.type === 'attack' || report.type === 'plunder') return 'attack'
  return report.viewType || report.battleType || report.type || 'unknown'
}

/** 使用真实字段生成缺省主题，不伪造战斗结果或玩家身份。 */
function fallbackTitle(report: BattleReportState, type: string) {
  const action = reportTypeLabels[type]?.replace(/\s/g, '') || report.type || '军情'
  return [report.playerName, action, report.targetName].filter(Boolean).join(' ') || '未命名军情'
}

/** 将 RFC3339 时间转换为官网列表使用的本地日期时间。 */
export function formatIntelligenceTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value || '时间未知'
  const pad = (part: number) => String(part).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}

/** 把后端真实战报映射为官网军情列表行。 */
export function toIntelligenceReport(report: BattleReportState): IntelligenceReportViewModel {
  const type = resolveIntelligenceType(report)
  return {
    id: report.id,
    type,
    typeLabel: reportTypeLabels[type] || '军 情',
    title: report.title || fallbackTitle(report, type),
    createdAt: formatIntelligenceTime(report.createdAt),
    read: Boolean(report.read),
    source: report,
  }
}

/** 计算安全总页数，空列表仍保留第一页。 */
export function intelligenceTotalPages(total: number, pageSize: number) {
  return Math.max(1, Math.ceil(Math.max(0, total) / Math.max(1, pageSize)))
}
