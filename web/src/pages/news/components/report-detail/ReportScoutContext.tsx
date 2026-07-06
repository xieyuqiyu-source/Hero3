/* 本文件渲染侦查战报上下文。 */
import type { FC } from 'react'
import type { BattleReportScoutExtra } from '@/types/game'

interface ReportScoutContextProps {
  scout?: BattleReportScoutExtra
}

// ReportScoutContext 展示侦察兵派出、损失和情报结果。
const ReportScoutContext: FC<ReportScoutContextProps> = ({ scout }) => {
  if (!scout) return null
  return (
    <section className="rounded-lg border border-blue-500/30 bg-blue-500/5 p-3">
      <h3 className="text-xs font-bold text-blue-500">{scout.success ? '侦查成功' : '侦查失败'}</h3>
      <div className="mt-2 grid gap-2 text-[11px] text-[var(--color-text-secondary)] sm:grid-cols-3">
        <div><span className="font-bold text-[var(--color-text-primary)]">侦察兵</span><br />派出 {(scout.scoutSent ?? 0).toLocaleString()} · 阵亡 {(scout.scoutLost ?? 0).toLocaleString()} · 返回 {(scout.scoutReturned ?? 0).toLocaleString()}</div>
        <div><span className="font-bold text-[var(--color-text-primary)]">反侦查损失</span><br />{scout.counterScoutUnitType || '守方侦察兵'} {(scout.counterScoutLost ?? 0).toLocaleString()}</div>
        <div><span className="font-bold text-[var(--color-text-primary)]">获得情报</span><br />兵力 {scout.revealUnits ? '可见' : '隐藏'} · 资源 {scout.revealResources ? '可见' : '隐藏'}</div>
      </div>
    </section>
  )
}

export default ReportScoutContext
