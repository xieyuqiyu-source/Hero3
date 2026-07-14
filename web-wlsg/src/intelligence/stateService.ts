/** 军情状态服务统一处理真实列表、详情、已读、删除和过期请求隔离。 */
import type { GameApi } from '../api/gameApi'
import type { BattleReportState, IntelligenceReportViewModel, IntelligenceTabKey, ReportActionResponse } from '../game/types'
import { intelligenceFilter, intelligenceTotalPages, toIntelligenceReport } from './adapter'

const latestBatchSize = 50

export type IntelligencePhase = 'idle' | 'loading' | 'ready' | 'error'

export interface IntelligenceStore {
  phase: IntelligencePhase
  playerId: string | null
  activeTab: IntelligenceTabKey
  page: number
  pageSize: number
  reports: IntelligenceReportViewModel[]
  total: number
  error: string
  detail: BattleReportState | null
  detailLoading: boolean
  detailError: string
  deleting: boolean
  actionMessage: string
}

export interface IntelligenceService {
  state: IntelligenceStore
  load: (playerId: string, tab?: IntelligenceTabKey, page?: number) => Promise<void>
  refresh: () => Promise<void>
  selectTab: (tab: IntelligenceTabKey) => Promise<void>
  selectPage: (page: number) => Promise<void>
  openReport: (reportId: string) => Promise<void>
  closeReport: () => void
  deleteReports: (reportIds: string[]) => Promise<void>
  clear: () => void
}

/** 创建带取消、版本隔离和写操作防重的军情状态服务。 */
export function createIntelligenceService(api: GameApi, state: IntelligenceStore, patchGameState?: (result: ReportActionResponse, playerId: string) => void): IntelligenceService {
  let listVersion = 0
  let detailVersion = 0
  let listController: AbortController | null = null
  let detailController: AbortController | null = null
  let visibleReports: IntelligenceReportViewModel[] = []

  /** 把当前标签已完整读取并按读状态筛选的军情投影到前端页码。 */
  function applyVisiblePage(page: number) {
    const total = visibleReports.length
    const pages = intelligenceTotalPages(total, state.pageSize)
    const safePage = Math.min(pages, Math.max(1, page))
    const offset = (safePage - 1) * state.pageSize
    Object.assign(state, { page: safePage, reports: visibleReports.slice(offset, offset + state.pageSize), total })
  }

  /** 分批读取当前标签全部军情，确保按未读/已读筛选后不会漏掉后续服务端分页。 */
  async function fetchVisibleReports(playerId: string, tab: IntelligenceTabKey, signal: AbortSignal) {
    const filter = intelligenceFilter(tab)
    const first = await api.listReports(playerId, 1, latestBatchSize, filter, signal)
    const serverPageSize = Math.max(1, Number(first.pageSize) || latestBatchSize)
    const serverTotal = Number.isFinite(first.total) ? Math.max(0, first.total) : first.reports.length
    const serverPages = intelligenceTotalPages(serverTotal, serverPageSize)
    const pages = [first]
    for (let page = 2; page <= serverPages; page += 1) {
      pages.push(await api.listReports(playerId, page, latestBatchSize, filter, signal))
    }
    const seen = new Set<string>()
    return pages.flatMap((result) => Array.isArray(result.reports) ? result.reports : [])
      .filter((report) => (tab === 'all' ? !report.read : report.read) && !seen.has(report.id) && Boolean(seen.add(report.id)))
      .map(toIntelligenceReport)
  }

  /** 清除军情状态并取消当前全部只读请求。 */
  function clear() {
    listVersion += 1
    detailVersion += 1
    listController?.abort()
    detailController?.abort()
    listController = null
    detailController = null
    visibleReports = []
    Object.assign(state, { phase: 'idle', playerId: null, activeTab: 'all', page: 1, reports: [], total: 0, error: '', detail: null, detailLoading: false, detailError: '', deleting: false, actionMessage: '' })
  }

  /** 加载已校验存档的真实军情页，并忽略切档或快速切页产生的旧响应。 */
  async function load(playerId: string, tab = state.activeTab, page = state.page) {
    listVersion += 1
    const version = listVersion
    listController?.abort()
    listController = new AbortController()
    Object.assign(state, { phase: 'loading', playerId, activeTab: tab, page: Math.max(1, page), error: '', actionMessage: '' })
    try {
      const reports = await fetchVisibleReports(playerId, tab, listController.signal)
      if (version !== listVersion || state.playerId !== playerId) return
      visibleReports = reports
      applyVisiblePage(state.page)
      Object.assign(state, { phase: 'ready', error: '' })
    } catch (error) {
      if (version !== listVersion || (error instanceof DOMException && error.name === 'AbortError')) return
      visibleReports = []
      Object.assign(state, { phase: 'error', reports: [], total: 0, error: error instanceof Error ? error.message : '军情加载失败' })
    }
  }

  /** 刷新当前玩家与当前筛选页，加载中不重复请求。 */
  async function refresh() {
    if (!state.playerId || state.phase === 'loading') return
    await load(state.playerId, state.activeTab, state.page)
  }

  /** 切换军情标签并从第一页重新读取。 */
  async function selectTab(tab: IntelligenceTabKey) {
    if (!state.playerId || state.phase === 'loading' || tab === state.activeTab) return
    state.detail = null
    await load(state.playerId, tab, 1)
  }

  /** 切换到有效页码并重新读取后端分页。 */
  async function selectPage(page: number) {
    if (!state.playerId || state.phase === 'loading') return
    const target = Math.min(intelligenceTotalPages(state.total, state.pageSize), Math.max(1, page))
    if (target === state.page) return
    applyVisiblePage(target)
  }

  /** 读取完整战报；未读记录同时调用现有已读接口并回写未读数量。 */
  async function openReport(reportId: string) {
    if (!state.playerId || state.detailLoading) return
    const playerId = state.playerId
    const report = state.reports.find((item) => item.id === reportId)
    detailVersion += 1
    const version = detailVersion
    detailController?.abort()
    detailController = new AbortController()
    Object.assign(state, { detail: null, detailLoading: true, detailError: '', actionMessage: '' })
    try {
      const detail = await api.report(playerId, reportId, detailController.signal)
      if (version !== detailVersion || state.playerId !== playerId) return
      state.detail = detail
      if (report && !report.read) {
        try {
          const readResult = await api.markReportRead(playerId, reportId)
          if (version !== detailVersion || state.playerId !== playerId) return
          report.read = true
          if (state.activeTab === 'all') {
            visibleReports = visibleReports.filter((item) => item.id !== reportId)
            applyVisiblePage(state.page)
          }
          patchGameState?.(readResult, playerId)
        } catch (error) {
          if (version === detailVersion && state.playerId === playerId) state.actionMessage = error instanceof Error ? error.message : '战报已打开，但标记已读失败'
        }
      }
    } catch (error) {
      if (version !== detailVersion || (error instanceof DOMException && error.name === 'AbortError')) return
      state.detailError = error instanceof Error ? error.message : '战报详情加载失败'
    } finally {
      if (version === detailVersion) state.detailLoading = false
    }
  }

  /** 关闭详情并取消尚未完成的详情请求。 */
  function closeReport() {
    detailVersion += 1
    detailController?.abort()
    detailController = null
    Object.assign(state, { detail: null, detailLoading: false, detailError: '' })
  }

  /** 串行删除勾选军情，避免并发写入并在完成后重新读取权威列表。 */
  async function deleteReports(reportIds: string[]) {
    if (!state.playerId || state.deleting) return
    const ids = [...new Set(reportIds.filter((id) => state.reports.some((report) => report.id === id)))]
    if (!ids.length) {
      state.actionMessage = '请先选择需要删除的军情'
      return
    }
    const playerId = state.playerId
    const version = listVersion
    state.deleting = true
    state.actionMessage = ''
    try {
      let result: ReportActionResponse | null = null
      for (const reportId of ids) result = await api.deleteReport(playerId, reportId)
      if (version !== listVersion || state.playerId !== playerId) return
      if (result) patchGameState?.(result, playerId)
      state.actionMessage = `已删除 ${ids.length} 条军情`
      await load(playerId, state.activeTab, state.page)
      state.actionMessage = `已删除 ${ids.length} 条军情`
    } catch (error) {
      if (version !== listVersion || state.playerId !== playerId) return
      state.actionMessage = error instanceof Error ? error.message : '删除军情失败'
    } finally {
      if (state.playerId === playerId) state.deleting = false
    }
  }

  return { state, load, refresh, selectTab, selectPage, openReport, closeReport, deleteReports, clear }
}
