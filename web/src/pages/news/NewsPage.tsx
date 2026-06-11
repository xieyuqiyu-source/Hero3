import { useCallback, useEffect, useState, type FC } from 'react'
import { ChevronLeft, ChevronRight, Loader2, Trash2 } from 'lucide-react'
import { useGameStore } from '@/store/gameStore'
import { gameApi } from '@/api/game'
import type { BattleReport } from '@/types/game'
import BattleReportDetail from './components/BattleReportDetail'

const TYPE_CONFIG: Record<string, { label: string; color: string }> = {
  attack:    { label: '攻击', color: 'text-red-600 bg-red-500/10' },
  plunder:   { label: '掠夺', color: 'text-amber-600 bg-amber-500/10' },
  scout:     { label: '侦查', color: 'text-blue-600 bg-blue-500/10' },
  reinforce: { label: '增援', color: 'text-green-600 bg-green-500/10' },
}

const EMPTY_REPORTS: BattleReport[] = []
const PAGE_SIZE = 10

const NewsPage: FC = () => {
  const [selectedReport, setSelectedReport] = useState<BattleReport | null>(null)
  const [currentPage, setCurrentPage] = useState(1)
  const [reports, setReports] = useState<BattleReport[]>(EMPTY_REPORTS)
  const [totalReports, setTotalReports] = useState(0)
  const [loading, setLoading] = useState(false)
  const [hasLoaded, setHasLoaded] = useState(false)
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const setState = useGameStore((s) => s.setState)

  const totalPages = Math.max(1, Math.ceil(totalReports / PAGE_SIZE))

  const loadReports = useCallback(async (page: number) => {
    if (!activePlayerId) {
      setReports(EMPTY_REPORTS)
      setTotalReports(0)
      setHasLoaded(true)
      return
    }
    setLoading(true)
    try {
      const result = await gameApi.listReports(activePlayerId, page, PAGE_SIZE)
      setReports(result.reports)
      setTotalReports(result.total)
      const nextTotalPages = Math.max(1, Math.ceil(result.total / PAGE_SIZE))
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
  }, [activePlayerId])

  useEffect(() => {
    loadReports(currentPage)
  }, [currentPage, loadReports])

  // 点击单条战报时标记为已读
  const handleSelectReport = (report: BattleReport) => {
    setSelectedReport(report)
    if (!report.read && activePlayerId) {
      setReports((items) => items.map((item) => item.id === report.id ? { ...item, read: true } : item))
      gameApi.markReportsRead(activePlayerId, report.id).then((res) => {
        setState(res.state)
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

  if (totalReports === 0) {
    return (
      <div className="flex items-center justify-center py-16">
        <span className="text-sm text-[var(--color-text-muted)]">暂无军情</span>
      </div>
    )
  }

  const handleDeleteReport = (e: React.MouseEvent, reportId: string) => {
    e.stopPropagation()
    if (!activePlayerId) return
    gameApi.deleteReport(activePlayerId, reportId).then((res) => {
      setState(res.state)
      loadReports(currentPage)
    }).catch(() => {})
  }

  const handleDeleteAll = () => {
    if (!activePlayerId) return
    gameApi.deleteAllReports(activePlayerId).then((res) => {
      setState(res.state)
      setCurrentPage(1)
      loadReports(1)
    }).catch(() => {})
  }

  return (
    <div className="space-y-2">
      {/* 一键删除 */}
      <div className="flex justify-end">
        <button
          type="button"
          onClick={handleDeleteAll}
          disabled={loading}
          className="flex items-center gap-1 text-[10px] text-red-500 hover:text-red-600 cursor-pointer transition-colors px-2 py-1 rounded-lg hover:bg-red-500/10 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <Trash2 size={12} />
          清空全部
        </button>
      </div>

      <div className="relative space-y-2">
        {loading && (
          <div className="absolute inset-0 z-10 flex items-center justify-center rounded-2xl bg-[var(--color-bg)]/55 backdrop-blur-[1px]">
            <div className="inline-flex items-center gap-2 rounded-full border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-xs text-[var(--color-text-secondary)] shadow-sm">
              <Loader2 size={14} className="animate-spin text-[var(--color-accent)]" />
              加载军情中
            </div>
          </div>
        )}

        {reports.map((report) => {
          const typeConfig = TYPE_CONFIG[report.type] ?? TYPE_CONFIG.attack
          const isVictory = report.result === 'attacker_victory'
          const hasRewards = Object.values(report.rewards).some(v => v > 0)
          const hasLosses = Object.values(report.lostUnits).some(v => v > 0)

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
                {/* 类型标签 */}
                <span className={`text-[10px] font-bold px-1.5 py-0.5 rounded ${typeConfig.color}`}>
                  {typeConfig.label}
                </span>
                {/* 胜负 */}
                <span className={`text-xs font-bold ${isVictory ? 'text-green-600' : 'text-red-600'}`}>
                  {isVictory ? '胜' : report.result === 'draw' ? '平' : '败'}
                </span>
                {/* 目标 */}
                <span className="text-xs text-[var(--color-text-primary)] truncate flex-1">
                  {report.targetName}
                </span>
                {/* 资源/损失摘要 */}
                {hasRewards && (
                  <span className="text-[10px] text-green-600 flex-shrink-0">
                    +{Object.values(report.rewards).reduce((s, v) => s + v, 0).toLocaleString()}
                  </span>
                )}
                {hasLosses && (
                  <span className="text-[10px] text-red-500 flex-shrink-0">
                    -{Object.values(report.lostUnits).reduce((s, v) => s + v, 0)}兵
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
