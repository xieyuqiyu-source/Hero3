import { useState, type FC } from 'react'
import { Trash2 } from 'lucide-react'
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

const NewsPage: FC = () => {
  const [selectedReport, setSelectedReport] = useState<BattleReport | null>(null)
  const reports = useGameStore((s) => s.state?.recentBattleReports ?? EMPTY_REPORTS)
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const setState = useGameStore((s) => s.setState)

  // 点击单条战报时标记为已读
  const handleSelectReport = (report: BattleReport) => {
    setSelectedReport(report)
    if (!report.read && activePlayerId) {
      gameApi.markReportsRead(activePlayerId, report.id).then((res) => {
        setState(res.state)
      }).catch(() => {})
    }
  }

  // 过滤 3 天内
  const threeDaysAgo = Date.now() - 3 * 24 * 60 * 60 * 1000
  const recentReports = reports.filter((r) => new Date(r.createdAt).getTime() > threeDaysAgo)

  if (selectedReport) {
    return (
      <div className="animate-slide-in">
        <BattleReportDetail report={selectedReport} onBack={() => setSelectedReport(null)} />
      </div>
    )
  }

  if (recentReports.length === 0) {
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
    }).catch(() => {})
  }

  const handleDeleteAll = () => {
    if (!activePlayerId) return
    gameApi.deleteAllReports(activePlayerId).then((res) => {
      setState(res.state)
    }).catch(() => {})
  }

  return (
    <div className="space-y-2">
      {/* 一键删除 */}
      <div className="flex justify-end">
        <button
          type="button"
          onClick={handleDeleteAll}
          className="flex items-center gap-1 text-[10px] text-red-500 hover:text-red-600 cursor-pointer transition-colors px-2 py-1 rounded-lg hover:bg-red-500/10"
        >
          <Trash2 size={12} />
          清空全部
        </button>
      </div>

      {recentReports.map((report) => {
        const typeConfig = TYPE_CONFIG[report.type] ?? TYPE_CONFIG.attack
        const isVictory = report.result === 'attacker_victory'
        const hasRewards = Object.values(report.rewards).some(v => v > 0)
        const hasLosses = Object.values(report.lostUnits).some(v => v > 0)

        return (
          <button
            key={report.id}
            type="button"
            onClick={() => handleSelectReport(report)}
            className={`
              w-full text-left px-4 py-3 rounded-2xl border bg-[var(--color-surface)]
              hover:border-[var(--color-accent-border)] cursor-pointer transition-colors
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
