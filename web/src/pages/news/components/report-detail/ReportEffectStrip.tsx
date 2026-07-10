/* 本文件渲染战报中的战前效果、城墙和玩法提示条。 */
import type { FC } from 'react'
import type { BattleReportDetailData } from '@/types/game'

interface ReportEffectStripProps {
  detail: BattleReportDetailData
}

// ReportEffectStrip 汇总轻量效果信息，避免详情首屏散乱。
const ReportEffectStrip: FC<ReportEffectStripProps> = ({ detail }) => {
  const pvp = detail.extra?.pvp as { wall?: { level?: number; totalDefenseBonus?: number } } | undefined
  const items: string[] = []
  if (pvp?.wall) {
    items.push(`城墙 Lv.${pvp.wall.level ?? 0}`)
    if (typeof pvp.wall.totalDefenseBonus === 'number') items.push(`防守加成 ${(pvp.wall.totalDefenseBonus * 100).toFixed(0)}%`)
  }
  if ((detail.extra?.yellowTurban?.foodPressure ?? 0) > 1) items.push('口粮超载触发')
  if (detail.extra?.sweep?.detailMode === 'lightweight') items.push('扫荡轻量摘要')
  if (items.length === 0) return null
  return (
    <section className="flex flex-wrap gap-1.5 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] p-2">
      {items.map((item) => <span key={item} className="rounded bg-[var(--color-surface-dim)] px-2 py-1 text-[10px] font-semibold text-[var(--color-text-secondary)]">{item}</span>)}
    </section>
  )
}

export default ReportEffectStrip
