// 本文件实现世界地图关系筛选控件。
import type { FC } from 'react'
import type { WorldMapRelationCounts } from '../worldMapGridLogic'

export type WorldMapRelationFilters = Record<string, boolean>

// WorldMapFilters 渲染自己、同盟和其他玩家的关系筛选。
const WorldMapFilters: FC<{
  filters: WorldMapRelationFilters
  counts: WorldMapRelationCounts
  onChange: (filters: WorldMapRelationFilters) => void
}> = ({ filters, counts, onChange }) => {
  // toggle 更新单个关系筛选状态。
  const toggle = (key: string, checked: boolean) => onChange({ ...filters, [key]: checked })
  return (
    <div className="flex flex-wrap items-center gap-2">
      <label className="inline-flex items-center gap-1.5 text-xs font-bold text-[var(--color-text-secondary)]">
        <input type="checkbox" checked={filters.self !== false} onChange={(event) => toggle('self', event.target.checked)} />
        自己
        <FilterCount value={counts.self} />
      </label>
      <label className="inline-flex items-center gap-1.5 text-xs font-bold text-[var(--color-text-secondary)]">
        <input type="checkbox" checked={filters.ally !== false} onChange={(event) => toggle('ally', event.target.checked)} />
        同盟
        <FilterCount value={counts.ally} />
      </label>
      <label className="inline-flex items-center gap-1.5 text-xs font-bold text-[var(--color-text-secondary)]">
        <input type="checkbox" checked={filters.other !== false} onChange={(event) => toggle('other', event.target.checked)} />
        其他玩家
        <FilterCount value={counts.other} />
      </label>
    </div>
  )
}

// FilterCount 展示当前关系分类下缓存城池数量。
const FilterCount: FC<{ value: number }> = ({ value }) => (
  <span className="rounded bg-[var(--color-surface-dim)] px-1.5 py-0.5 text-[10px] font-black text-[var(--color-text-muted)]">
    {value}
  </span>
)

export default WorldMapFilters
