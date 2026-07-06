/* 本文件渲染战报详情顶部导航、标签、标题和结果。 */
import type { FC } from 'react'
import { ArrowLeft, Check, Share2 } from 'lucide-react'
import type { BattleReportDetailData } from '@/types/game'
import { REPORT_SOURCE_CONFIG, REPORT_VIEW_CONFIG } from '../../reportPresentation'
import BattleOutcomeSeal from './BattleOutcomeSeal'

interface BattleReportHeaderProps {
  detail: BattleReportDetailData
  copied: boolean
  onBack: () => void
  onShare: () => void
}

// formatBattleTime 把 ISO 时间转换为中文本地时间。
function formatBattleTime(value?: string): string {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString('zh-CN')
}

// resolveSideName 从标准参战方中取玩家、NPC 或城池名称。
function resolveSideName(side?: BattleReportDetailData['primarySide']): string {
  if (!side) return ''
  return side.playerName || side.targetName || side.cityName || side.targetId || side.playerId || ''
}

// battleActionLabel 返回顶部标题中间的战斗动作。
function battleActionLabel(detail: BattleReportDetailData): string {
  if (detail.viewType === 'scout' || detail.battleType === 'scout') return '侦查'
  if (detail.battleType === 'sweep') return '扫荡'
  if (detail.viewType === 'reinforcement') return '协防'
  if (detail.battleType === 'plunder') return '掠夺'
  return '进攻'
}

// factionTitleClass 按阵营区分顶部双方名称颜色。
function factionTitleClass(faction?: string): string {
  const key = (faction || '').toLowerCase()
  if (key.includes('wei') || key.includes('魏')) return 'text-sky-300'
  if (key.includes('shu') || key.includes('蜀')) return 'text-emerald-300'
  if (key.includes('wu') || key.includes('吴')) return 'text-red-300'
  return 'text-amber-300'
}

// BattleTitle 渲染可独立高亮双方名称的结构化标题。
const BattleTitle: FC<{ detail: BattleReportDetailData }> = ({ detail }) => {
  const attackerName = resolveSideName(detail.primarySide)
  const defenderName = resolveSideName(detail.secondarySide)
  if (!attackerName || !defenderName) {
    return <h2 className="text-center text-xl font-black text-[var(--color-text-primary)] sm:text-2xl">{detail.title || '战报详情'}</h2>
  }
  return (
    <h2 className="flex flex-wrap items-center justify-center gap-x-3 gap-y-1 text-center text-xl font-black sm:text-2xl">
      <span className={factionTitleClass(detail.primarySide.faction)}>{attackerName}</span>
      <span className="text-[var(--color-text-primary)]">{battleActionLabel(detail)}</span>
      <span className={factionTitleClass(detail.secondarySide?.faction)}>{defenderName}</span>
    </h2>
  )
}

// BattleReportHeader 展示来源、视角、分享和主标题。
const BattleReportHeader: FC<BattleReportHeaderProps> = ({ detail, copied, onBack, onShare }) => {
  const sourceConfig = REPORT_SOURCE_CONFIG[detail.sourceType] ?? { label: detail.sourceLabel || '来源', color: 'text-slate-600 bg-slate-500/10' }
  const viewConfig = REPORT_VIEW_CONFIG[detail.viewType] ?? { label: detail.viewLabel || '视角', color: 'text-slate-600 bg-slate-500/10' }
  return (
    <header className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)]">
      <div className="flex items-center justify-between border-b border-[var(--color-border)] px-3 py-2">
        <button type="button" onClick={onBack} className="inline-flex items-center gap-1.5 text-xs font-semibold text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]">
          <ArrowLeft size={14} />
          返回
        </button>
        <div className="flex items-center gap-1.5">
          <span className={`rounded px-1.5 py-0.5 text-[10px] font-bold ${sourceConfig.color}`}>{sourceConfig.label}</span>
          <span className={`rounded px-1.5 py-0.5 text-[10px] font-bold ${viewConfig.color}`}>{viewConfig.label}</span>
          <button type="button" onClick={onShare} className="inline-flex items-center gap-1 rounded px-2 py-1 text-[10px] font-bold text-blue-500 hover:bg-blue-500/10">
            {copied ? <Check size={12} /> : <Share2 size={12} />}
            {copied ? '已复制' : '分享'}
          </button>
        </div>
      </div>
      <div className="grid gap-3 px-3 py-4 sm:grid-cols-[72px_minmax(0,1fr)_72px] sm:items-center">
        <div className="hidden sm:block" />
        <div className="min-w-0 text-center">
          <BattleTitle detail={detail} />
          <div className="mt-1 text-[11px] text-[var(--color-text-muted)]">{formatBattleTime(detail.occurredAt)}</div>
          {detail.summary && <p className="mx-auto mt-2 max-w-3xl text-xs text-[var(--color-text-secondary)]">{detail.summary}</p>}
        </div>
        <div className="flex justify-center sm:justify-end">
          <BattleOutcomeSeal outcome={detail.ownerOutcome} />
        </div>
      </div>
    </header>
  )
}

export default BattleReportHeader
