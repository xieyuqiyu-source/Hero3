/* 本文件渲染战报双方身份和武将快照。 */
import type { FC } from 'react'
import type { BattleReportSide } from '@/types/game'

interface BattleSideCardProps {
  side: BattleReportSide
}

// formatGenerals 把武将快照压缩成紧凑文本。
function formatGenerals(side: BattleReportSide): string {
  const generals = side.generals ?? []
  if (generals.length === 0) return '无'
  return generals.map((general) => `${general.name || general.id}${general.level ? ` Lv.${general.level}` : ''}`).join('、')
}

// BattleSideCard 展示一侧的阵营、城池、战力和武将。
const BattleSideCard: FC<BattleSideCardProps> = ({ side }) => (
  <section className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] p-3">
    <div className="flex items-start justify-between gap-3">
      <div className="min-w-0">
        <div className="text-[10px] font-bold text-[var(--color-text-muted)]">{side.role || '参战方'} · {side.factionLabel || side.faction || '未知阵营'}</div>
        <div className="mt-1 truncate text-sm font-black text-[var(--color-text-primary)]">{side.cityName || side.playerName || side.targetName || '未知目标'}</div>
        {(side.playerName || side.targetName) && (
          <div className="mt-0.5 truncate text-[11px] text-[var(--color-text-secondary)]">{side.playerName || side.targetName}</div>
        )}
      </div>
      <div className="shrink-0 text-right">
        <div className="text-[10px] text-[var(--color-text-muted)]">战力</div>
        <div className="text-sm font-black text-amber-500">{(side.power || 0).toLocaleString()}</div>
      </div>
    </div>
    <div className="mt-3 border-t border-[var(--color-border)] pt-2 text-[11px] text-[var(--color-text-secondary)]">
      武将：<span className="font-semibold text-[var(--color-text-primary)]">{formatGenerals(side)}</span>
    </div>
  </section>
)

export default BattleSideCard
