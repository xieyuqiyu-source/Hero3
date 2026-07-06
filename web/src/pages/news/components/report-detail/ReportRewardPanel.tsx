/* 本文件渲染战报奖励、掉落和武将经验。 */
import type { FC } from 'react'
import type { BattleReportRewards } from '@/types/game'
import { mergeBattleReportDrops } from '@/utils/reportDrops'

const RESOURCE_LABELS: Record<string, string> = { wood: '木材', stone: '石料', iron: '铁矿', food: '粮食' }
const RESOURCE_ORDER = ['wood', 'stone', 'iron', 'food']

interface ReportRewardPanelProps {
  rewards: BattleReportRewards
}

// hasRewards 判断奖励面板是否有可展示内容。
function hasRewards(rewards: BattleReportRewards): boolean {
  return Object.values(rewards.resources ?? {}).some((amount) => amount > 0) ||
    (rewards.cityGold ?? 0) > 0 ||
    (rewards.generalExp ?? 0) > 0 ||
    (rewards.drops ?? []).length > 0
}

// ReportRewardPanel 展示已获得资源、城金、经验和掉落。
const ReportRewardPanel: FC<ReportRewardPanelProps> = ({ rewards }) => {
  if (!hasRewards(rewards)) return null
  const drops = mergeBattleReportDrops(rewards.drops ?? [])
  return (
    <section className="rounded-lg border border-amber-500/30 bg-amber-500/5 p-3">
      <h3 className="text-xs font-bold text-amber-500">奖励与掉落</h3>
      <div className="mt-2 flex flex-wrap gap-1.5">
        {RESOURCE_ORDER.filter((key) => (rewards.resources?.[key] ?? 0) > 0).map((key) => (
          <span key={key} className="rounded border border-amber-500/20 bg-[var(--color-surface)] px-2 py-1 text-[11px] font-semibold text-[var(--color-text-primary)]">
            {RESOURCE_LABELS[key]} +{rewards.resources?.[key]?.toLocaleString()}
          </span>
        ))}
        {(rewards.cityGold ?? 0) > 0 && <span className="rounded border border-amber-500/20 bg-[var(--color-surface)] px-2 py-1 text-[11px] font-semibold text-amber-500">城金 +{rewards.cityGold}</span>}
        {(rewards.generalExp ?? 0) > 0 && <span className="rounded border border-emerald-500/20 bg-[var(--color-surface)] px-2 py-1 text-[11px] font-semibold text-emerald-500">武将经验 +{rewards.generalExp}</span>}
        {drops.map((drop) => (
          <span key={`${drop.itemId || drop.name}-${drop.quality || ''}`} className="rounded border border-purple-500/20 bg-[var(--color-surface)] px-2 py-1 text-[11px] font-semibold text-purple-500">
            {drop.name || drop.itemId} ×{drop.amount.toLocaleString()}
          </span>
        ))}
      </div>
    </section>
  )
}

export default ReportRewardPanel
