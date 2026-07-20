/* 本文件渲染战报中特性触发结果。 */
import type { FC } from 'react'
import type { BattleReportTrait } from '@/types/game'
import { formatTraitOutcomeDetails, type TraitOutcomeFormatOptions } from '@/utils/traits'
import { reportTraitRenderKey } from '../../reportPresentation'

interface ReportTraitPanelProps {
  traits?: BattleReportTrait[]
  formatOptions?: TraitOutcomeFormatOptions
}

// ReportTraitPanel 展示标准特性摘要和真实结算数值。
const ReportTraitPanel: FC<ReportTraitPanelProps> = ({ traits = [], formatOptions }) => {
  const visible = traits.filter((trait) => trait.traitId || trait.traitName || trait.summary)
  if (visible.length === 0) return null
  return (
    <section className="rounded-lg border border-indigo-500/30 bg-indigo-500/5 p-3">
      <h3 className="text-xs font-bold text-indigo-500">特性触发</h3>
      <div className="mt-2 space-y-1.5">
        {visible.map((trait, index) => {
          const detailText = formatTraitOutcomeDetails(trait.detail, formatOptions)
          return (
            <div key={reportTraitRenderKey(trait, index)} className="rounded-md border border-indigo-500/20 bg-[var(--color-surface)] px-2 py-1.5">
              <div className="text-[11px] font-bold text-[var(--color-text-primary)]">{trait.traitName || trait.traitId}</div>
              {(trait.generalName || trait.summary || detailText) && (
                <div className="mt-0.5 text-[10px] text-[var(--color-text-secondary)]">
                  {[trait.generalName, trait.summary, detailText].filter(Boolean).join(' · ')}
                </div>
              )}
            </div>
          )
        })}
      </div>
    </section>
  )
}

export default ReportTraitPanel
