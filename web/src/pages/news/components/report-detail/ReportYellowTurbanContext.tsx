/* 本文件渲染黄巾起义战报上下文。 */
import type { FC } from 'react'
import type { BattleReportYellowTurbanExtra } from '@/types/game'

interface ReportYellowTurbanContextProps {
  yellowTurban?: BattleReportYellowTurbanExtra
}

// percentText 把小数压力转换为百分比文本。
function percentText(value?: number): string {
  if (typeof value !== 'number') return '未知'
  return `${(value * 100).toFixed(0)}%`
}

// ReportYellowTurbanContext 展示风险等级、来源和口粮压力。
const ReportYellowTurbanContext: FC<ReportYellowTurbanContextProps> = ({ yellowTurban }) => {
  if (!yellowTurban) return null
  return (
    <section className="rounded-lg border border-yellow-500/30 bg-yellow-500/5 p-3">
      <h3 className="text-xs font-bold text-yellow-600">黄巾来袭</h3>
      <div className="mt-2 grid gap-2 text-[11px] text-[var(--color-text-secondary)] sm:grid-cols-3">
        <div><span className="font-bold text-[var(--color-text-primary)]">风险等级</span><br />{yellowTurban.riskLevelName || yellowTurban.riskLevelId || '未知'}</div>
        <div><span className="font-bold text-[var(--color-text-primary)]">来源</span><br />{yellowTurban.sourceCityName || yellowTurban.sourceCityId || '未知黄巾城'}</div>
        <div><span className="font-bold text-[var(--color-text-primary)]">口粮压力</span><br />{(yellowTurban.currentFood ?? 0).toLocaleString()} / {(yellowTurban.foodCapacity ?? 0).toLocaleString()} · {percentText(yellowTurban.foodPressure)}</div>
      </div>
    </section>
  )
}

export default ReportYellowTurbanContext
