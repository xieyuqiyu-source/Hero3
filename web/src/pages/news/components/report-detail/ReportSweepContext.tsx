/* 本文件渲染扫荡聚合战报上下文。 */
import type { FC } from 'react'
import type { BattleReportSweepExtra } from '@/types/game'

interface ReportSweepContextProps {
  sweep?: BattleReportSweepExtra
}

// resultLabel 返回单个扫荡目标结果文本。
function resultLabel(result?: string): string {
  if (result === 'attacker_victory') return '胜利'
  if (result === 'defender_victory') return '失败'
  if (result === 'draw') return '平局'
  return '已结算'
}

// ReportSweepContext 展示扫荡汇总和轻量目标列表。
const ReportSweepContext: FC<ReportSweepContextProps> = ({ sweep }) => {
  if (!sweep) return null
  const defenders = sweep.defenders ?? []
  return (
    <section className="rounded-lg border border-cyan-500/30 bg-cyan-500/5 p-3">
      <h3 className="text-xs font-bold text-cyan-500">扫荡摘要</h3>
      <div className="mt-2 grid grid-cols-2 gap-1.5 text-[11px] text-[var(--color-text-secondary)] sm:grid-cols-4">
        <span>目标 {sweep.requested ?? defenders.length}</span>
        <span>已结算 {sweep.success ?? 0}</span>
        <span>异常 {sweep.failed ?? 0}</span>
        <span>中止 {sweep.stopped ? '是' : '否'}</span>
      </div>
      {defenders.length > 0 && (
        <div className="mt-3 space-y-1.5">
          {defenders.map((item) => (
            <div key={item.targetId} className="flex items-center justify-between gap-2 rounded-md border border-cyan-500/20 bg-[var(--color-surface)] px-2 py-1.5 text-[11px]">
              <span className="min-w-0 truncate font-semibold text-[var(--color-text-primary)]">{item.targetName || item.targetId}</span>
              <span className="shrink-0 text-[var(--color-text-secondary)]">战力 {item.power.toLocaleString()} · {resultLabel(item.result)}</span>
            </div>
          ))}
        </div>
      )}
    </section>
  )
}

export default ReportSweepContext
