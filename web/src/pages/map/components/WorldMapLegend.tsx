// 本文件实现世界地图国家颜色、关系、状态和行军角标图例。
import type { FC } from 'react'
import { WORLD_MAP_FACTION_LEGEND, WORLD_MAP_MARCH_BADGE_LEGEND, WORLD_MAP_RELATION_BADGE_LEGEND, WORLD_MAP_STATUS_BADGE_LEGEND, worldMapRelationBadgeClass, worldMapStatusBadgeClass, type WorldMapFactionCounts } from '../worldMapGridLogic'

// WorldMapLegend 展示阵营颜色、关系角标、状态角标和行军角标含义。
const WorldMapLegend: FC<{ counts: WorldMapFactionCounts }> = ({ counts }) => (
  <div className="flex flex-wrap items-center gap-2 text-xs text-[var(--color-text-secondary)]">
    {WORLD_MAP_FACTION_LEGEND.map((item) => (
      <span key={item.faction} className="inline-flex items-center gap-1.5 font-bold">
        <span className={`h-2.5 w-2.5 rounded-sm ${item.colorClass}`} />
        {item.label}
        <LegendCount value={counts[item.faction as keyof WorldMapFactionCounts] ?? 0} />
      </span>
    ))}
    <span className="h-3 w-px bg-[var(--color-border)]" />
    {WORLD_MAP_RELATION_BADGE_LEGEND.map((item) => (
      <span key={item.relation} className="inline-flex items-center gap-1 font-bold">
        <span className={`inline-flex h-3.5 min-w-3.5 items-center justify-center rounded-full px-0.5 text-[8px] leading-none ${worldMapRelationBadgeClass(item.relation)}`}>
          {item.badge}
        </span>
        {item.label}
      </span>
    ))}
    <span className="h-3 w-px bg-[var(--color-border)]" />
    {WORLD_MAP_STATUS_BADGE_LEGEND.map((item) => (
      <span key={item.status} className="inline-flex items-center gap-1 font-bold">
        <span className={`inline-flex h-3.5 min-w-3.5 items-center justify-center rounded-full px-0.5 text-[8px] leading-none ${worldMapStatusBadgeClass(item.status)}`}>
          {item.badge}
        </span>
        {item.label}
      </span>
    ))}
    <span className="h-3 w-px bg-[var(--color-border)]" />
    {WORLD_MAP_MARCH_BADGE_LEGEND.map((item) => (
      <span key={item.badge} className="inline-flex items-center gap-1 font-bold">
        <span className="inline-flex h-3.5 min-w-3.5 items-center justify-center rounded-full bg-violet-600 px-0.5 text-[8px] leading-none text-white">
          {item.badge}
        </span>
        {item.label}
      </span>
    ))}
  </div>
)

// LegendCount 展示国家图例中的缓存城池数量。
const LegendCount: FC<{ value: number }> = ({ value }) => (
  <span className="rounded bg-[var(--color-surface-dim)] px-1 py-0.5 text-[10px] font-black text-[var(--color-text-muted)]">
    {value}
  </span>
)

export default WorldMapLegend
