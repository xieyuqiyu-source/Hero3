/* 本文件渲染战报情报可见性提示。 */
import type { FC } from 'react'
import type { BattleReportVisibility } from '@/types/game'

interface ReportIntelPanelProps {
  visibility: BattleReportVisibility
}

// ReportIntelPanel 仅在情报隐藏时展示原因。
const ReportIntelPanel: FC<ReportIntelPanelProps> = ({ visibility }) => {
  if (visibility.showEnemyRemainingUnits && visibility.showEnemyResources && visibility.showEnemyGenerals) return null
  const text = visibility.reason === 'scout_failed'
    ? '侦查失败，未获得目标资源和兵力情报'
    : '敌方详细兵力未对当前视角公开'
  return (
    <section className="rounded-lg border border-amber-500/30 bg-amber-500/5 px-3 py-2 text-xs font-semibold text-amber-500">
      {text}
    </section>
  )
}

export default ReportIntelPanel
