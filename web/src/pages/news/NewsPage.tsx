import { useCallback, useEffect, useState, type FC } from 'react'
import { ChevronLeft, ChevronRight, Loader2, Trash2 } from 'lucide-react'
import { useGameStore } from '@/store/gameStore'
import { gameApi } from '@/api/game'
import type { BattleReport } from '@/types/game'
import BattleReportDetail from './components/BattleReportDetail'
import { REPORT_SOURCE_CONFIG, REPORT_VIEW_CONFIG, REPORT_VIEW_TABS, reportTotalPages, shouldShowEmptyReports } from './reportPresentation'

const EMPTY_REPORTS: BattleReport[] = []
const PAGE_SIZE = 10

// safeReportMap 兼容旧战报或异常空字段，避免列表渲染时读取 null。
function safeReportMap(value?: Record<string, number> | null): Record<string, number> {
  return value ?? {}
}

const NewsPage: FC = () => {
  const [selectedReport, setSelectedReport] = useState<BattleReport | null>(null)
  const [currentPage, setCurrentPage] = useState(1)
  const [reports, setReports] = useState<BattleReport[]>(EMPTY_REPORTS)
  const [totalReports, setTotalReports] = useState(0)
  const [activeView, setActiveView] = useState('all')
  const [loading, setLoading] = useState(false)
  const [clearingReports, setClearingReports] = useState(false)
  const [hasLoaded, setHasLoaded] = useState(false)
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const patchState = useGameStore((s) => s.patchState)

  const totalPages = reportTotalPages(totalReports, PAGE_SIZE)

  const loadReports = useCallback(async (page: number) => {
    if (!activePlayerId) {
      setReports(EMPTY_REPORTS)
      setTotalReports(0)
      setHasLoaded(true)
      return
    }
    setLoading(true)
    try {
      const result = await gameApi.listReports(activePlayerId, page, PAGE_SIZE, activeView === 'all' ? undefined : { viewType: activeView })
      const nextReports = Array.isArray(result.reports) ? result.reports : EMPTY_REPORTS
      const nextTotal = typeof result.total === 'number' ? result.total : nextReports.length
      setReports(nextReports)
      setTotalReports(nextTotal)
      const nextTotalPages = reportTotalPages(nextTotal, PAGE_SIZE)
      if (page > nextTotalPages) {
        setCurrentPage(nextTotalPages)
      }
    } catch {
      setReports(EMPTY_REPORTS)
      setTotalReports(0)
    } finally {
      setLoading(false)
      setHasLoaded(true)
    }
  }, [activePlayerId, activeView])

  useEffect(() => {
    loadReports(currentPage)
  }, [currentPage, loadReports])

  // 切换 Tab 时回到第一页，避免上一 Tab 的页码越界。
  const handleChangeView = (viewType: string) => {
    setActiveView(viewType)
    setSelectedReport(null)
    setCurrentPage(1)
  }

  // 点击单条战报时标记为已读，并按需读取完整详情。
  const handleSelectReport = (report: BattleReport) => {
    setSelectedReport(report)
    if (!report.read && activePlayerId) {
      setReports((items) => items.map((item) => item.id === report.id ? { ...item, read: true } : item))
      gameApi.markReportsRead(activePlayerId, report.id).then((res) => {
        patchState({ unreadMessageCount: res.unreadMessageCount, serverTime: res.serverTime })
      }).catch(() => {})
    }
    if (activePlayerId) {
      gameApi.getReport(report.id, activePlayerId).then((fullReport) => {
        setSelectedReport(fullReport)
      }).catch(() => {})
    }
  }

  if (selectedReport) {
    return (
      <div className="animate-slide-in">
        <BattleReportDetail report={selectedReport} onBack={() => setSelectedReport(null)} />
      </div>
    )
  }

  if (!hasLoaded || (loading && reports.length === 0)) {
    return (
      <div className="flex items-center justify-center py-16">
        <span className="text-sm text-[var(--color-text-muted)]">军情加载中...</span>
      </div>
    )
  }

  const handleDeleteReport = (e: React.MouseEvent, reportId: string) => {
    e.stopPropagation()
    if (!activePlayerId) return
    gameApi.deleteReport(activePlayerId, reportId).then((res) => {
      patchState({ unreadMessageCount: res.unreadMessageCount, serverTime: res.serverTime })
      loadReports(currentPage)
    }).catch(() => {})
  }

  const handleDeleteAll = async () => {
    if (!activePlayerId || clearingReports) return
    setClearingReports(true)
    try {
      const res = await gameApi.deleteAllReports(activePlayerId, activeView === 'all' ? undefined : activeView)
      patchState({ unreadMessageCount: res.unreadMessageCount, serverTime: res.serverTime })
      setReports(EMPTY_REPORTS)
      setTotalReports(0)
      setCurrentPage(1)
      await loadReports(1)
    } catch {
      // 统一 API 层已经负责错误提示，这里只恢复按钮状态。
    } finally {
      setClearingReports(false)
    }
  }

  return (
    <div className="space-y-2">
      <div className="grid grid-cols-5 gap-1 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-1">
        {REPORT_VIEW_TABS.map((tab) => (
          <button
            key={tab.key}
            type="button"
            onClick={() => handleChangeView(tab.key)}
            className={`h-8 rounded-lg text-xs font-semibold transition-colors ${
              activeView === tab.key
                ? 'bg-[var(--color-surface)] text-[var(--color-text-primary)] shadow-sm'
                : 'text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)]'
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* 一键删除 */}
      <div className="flex justify-end">
        <button
          type="button"
          onClick={handleDeleteAll}
          disabled={loading || clearingReports}
          className="flex items-center gap-1 text-[10px] text-red-500 hover:text-red-600 cursor-pointer transition-colors px-2 py-1 rounded-lg hover:bg-red-500/10 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {clearingReports ? <Loader2 size={12} className="animate-spin" /> : <Trash2 size={12} />}
          {clearingReports ? '清空中' : '清空全部'}
        </button>
      </div>

      <div className="relative space-y-2">
        {(loading || clearingReports) && (
          <div className="absolute inset-0 z-10 flex items-center justify-center rounded-2xl bg-[var(--color-bg)]/55 backdrop-blur-[1px]">
            <div className="inline-flex items-center gap-2 rounded-full border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-xs text-[var(--color-text-secondary)] shadow-sm">
              <Loader2 size={14} className="animate-spin text-[var(--color-accent)]" />
              {clearingReports ? '清空军情中' : '加载军情中'}
            </div>
          </div>
        )}

        {shouldShowEmptyReports(totalReports, loading) && (
          <div className="flex items-center justify-center py-16">
            <span className="text-sm text-[var(--color-text-muted)]">暂无军情</span>
          </div>
        )}

        {reports.map((report) => {
          const viewType = report.viewType || (report.type === 'reinforce' ? 'reinforcement' : 'attack')
          const sourceType = report.sourceType || 'npc_city'
          const viewConfig = REPORT_VIEW_CONFIG[viewType] ?? REPORT_VIEW_CONFIG.attack
          const sourceConfig = REPORT_SOURCE_CONFIG[sourceType] ?? REPORT_SOURCE_CONFIG.npc_city
          const isVictory = report.result === 'attacker_victory'
          const rewards = safeReportMap(report.rewards)
          const lostUnits = safeReportMap(report.lostUnits)
          const hasRewards = Object.values(rewards).some(v => v > 0)
          const hasLosses = Object.values(lostUnits).some(v => v > 0)
          const title = report.title || report.detail?.title || report.targetName
          const summary = report.summary || report.detail?.summary

          return (
            <button
              key={report.id}
              type="button"
              onClick={() => handleSelectReport(report)}
              disabled={loading}
              className={`
                w-full text-left px-4 py-3 rounded-2xl border bg-[var(--color-surface)]
                hover:border-[var(--color-accent-border)] cursor-pointer transition-colors
                disabled:cursor-wait disabled:opacity-75
                ${report.read ? 'border-[var(--color-border)]' : 'border-[var(--color-accent)]'}
              `}
            >
              <div className="flex items-center gap-2">
                <span className={`text-[10px] font-bold px-1.5 py-0.5 rounded ${sourceConfig.color}`}>
                  {sourceConfig.label}
                </span>
                <span className={`text-[10px] font-bold px-1.5 py-0.5 rounded ${viewConfig.color}`}>
                  {viewConfig.label}
                </span>
                {/* 胜负 */}
                <span className={`text-xs font-bold ${isVictory ? 'text-green-600' : 'text-red-600'}`}>
                  {isVictory ? '胜' : report.result === 'draw' ? '平' : '败'}
                </span>
                {/* 目标 */}
                <span className="text-xs text-[var(--color-text-primary)] truncate flex-1">{title}</span>
                {/* 资源/损失摘要 */}
                {hasRewards && (
                  <span className="text-[10px] text-green-600 flex-shrink-0">
                    +{Object.values(rewards).reduce((s, v) => s + v, 0).toLocaleString()}
                  </span>
                )}
                {hasLosses && (
                  <span className="text-[10px] text-red-500 flex-shrink-0">
                    -{Object.values(lostUnits).reduce((s, v) => s + v, 0)}兵
                  </span>
                )}
                {/* 时间 */}
                <span className="text-[10px] text-[var(--color-text-muted)] flex-shrink-0">
                  {formatTimeAgo(report.createdAt)}
                </span>
                {/* 未读标记 */}
                {!report.read && (
                  <span className="w-2 h-2 rounded-full bg-red-500 flex-shrink-0" />
                )}
                {/* 删除按钮 */}
                <span
                  role="button"
                  tabIndex={0}
                  onClick={(e) => handleDeleteReport(e, report.id)}
                  onKeyDown={(e) => { if (e.key === 'Enter') handleDeleteReport(e as unknown as React.MouseEvent, report.id) }}
                  className="p-1 rounded-lg hover:bg-red-500/10 text-[var(--color-text-muted)] hover:text-red-500 transition-colors flex-shrink-0"
                >
                  <Trash2 size={12} />
                </span>
              </div>
              {summary && (
                <div className="mt-1 truncate pl-0.5 text-[10px] text-[var(--color-text-muted)]">{summary}</div>
              )}
            </button>
          )
        })}
      </div>

      {totalPages > 1 && (
        <div className="flex items-center justify-between gap-3 pt-2">
          <button
            type="button"
            onClick={() => setCurrentPage((page) => Math.max(1, page - 1))}
            disabled={loading || currentPage <= 1}
            className="h-9 w-9 inline-flex items-center justify-center rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-secondary)] disabled:opacity-40 disabled:cursor-not-allowed hover:border-[var(--color-accent-border)] transition-colors"
            aria-label="上一页"
          >
            <ChevronLeft size={16} />
          </button>
          <div className="text-xs text-[var(--color-text-muted)]">
            第 <span className="font-semibold text-[var(--color-text-primary)]">{currentPage}</span> / {totalPages} 页
            <span className="ml-2">共 {totalReports} 条</span>
          </div>
          <button
            type="button"
            onClick={() => setCurrentPage((page) => Math.min(totalPages, page + 1))}
            disabled={loading || currentPage >= totalPages}
            className="h-9 w-9 inline-flex items-center justify-center rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-secondary)] disabled:opacity-40 disabled:cursor-not-allowed hover:border-[var(--color-accent-border)] transition-colors"
            aria-label="下一页"
          >
            <ChevronRight size={16} />
          </button>
        </div>
      )}
    </div>
  )
}

function formatTimeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime()
  const minutes = Math.floor(diff / 60000)
  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes}分钟前`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}小时前`
  const days = Math.floor(hours / 24)
  return `${days}天前`
}

export default NewsPage
