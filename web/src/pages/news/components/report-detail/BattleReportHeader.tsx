/* 本文件渲染战报详情的紧凑导航、来源标签和时间。 */
import type { FC } from 'react'
import { ArrowLeft, Check, Share2 } from 'lucide-react'
import type { BattleReportDetailData } from '@/types/game'
import { REPORT_SOURCE_CONFIG, REPORT_VIEW_CONFIG } from '../../reportPresentation'

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

// BattleReportHeader 只展示导航、来源、视角、时间和分享，双方身份与胜负交给参战方区块。
const BattleReportHeader: FC<BattleReportHeaderProps> = ({ detail, copied, onBack, onShare }) => {
  const sourceConfig = REPORT_SOURCE_CONFIG[detail.sourceType] ?? { label: detail.sourceLabel || '来源', color: 'text-slate-600 bg-slate-500/10' }
  const viewConfig = REPORT_VIEW_CONFIG[detail.viewType] ?? { label: detail.viewLabel || '视角', color: 'text-slate-600 bg-slate-500/10' }
  return (
    <header className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2">
      <div className="flex flex-wrap items-center gap-2">
        <button type="button" onClick={onBack} className="inline-flex items-center gap-1.5 text-xs font-semibold text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]">
          <ArrowLeft size={14} />
          返回
        </button>
        <div className="flex min-w-0 flex-1 items-center justify-center gap-1.5">
          <span className={`rounded px-1.5 py-0.5 text-[10px] font-bold ${sourceConfig.color}`}>{sourceConfig.label}</span>
          <span className={`rounded px-1.5 py-0.5 text-[10px] font-bold ${viewConfig.color}`}>{viewConfig.label}</span>
          <span className="hidden truncate text-[10px] text-[var(--color-text-muted)] sm:inline">{formatBattleTime(detail.occurredAt)}</span>
        </div>
        <button type="button" onClick={onShare} className="inline-flex items-center gap-1 rounded px-2 py-1 text-[10px] font-bold text-blue-500 hover:bg-blue-500/10">
          {copied ? <Check size={12} /> : <Share2 size={12} />}
          {copied ? '已复制' : '分享'}
        </button>
      </div>
      <div className="mt-1 text-center text-[10px] text-[var(--color-text-muted)] sm:hidden">{formatBattleTime(detail.occurredAt)}</div>
    </header>
  )
}

export default BattleReportHeader
