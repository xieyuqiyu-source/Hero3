// 本文件实现世界地图坐标查找控件。
import type { FC, KeyboardEvent } from 'react'
import { LocateFixed, Search } from 'lucide-react'

// WorldMapCoordinateSearch 渲染 X/Y 坐标查找和返回本城入口。
const WorldMapCoordinateSearch: FC<{
  value: { x: string; y: string }
  worldSize: number
  onChange: (value: { x: string; y: string }) => void
  onSearch: () => void
  onFocusSelf: () => void
}> = ({ value, worldSize, onChange, onSearch, onFocusSelf }) => {
  const maxCoordinate = Math.max(0, Math.floor(worldSize) - 1)
  const maxCoordinateLength = String(maxCoordinate).length
  const coordinateSummary = `(${value.x || '?'}, ${value.y || '?'})`

  // handleKeyDown 支持在任一坐标输入框按回车直接查找。
  const handleKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key !== 'Enter') return
    event.preventDefault()
    onSearch()
  }

  // handleCoordinateChange 只接受空值或数字串，避免小数/科学计数法被当成格子坐标。
  const handleCoordinateChange = (axis: 'x' | 'y', nextValue: string) => {
    if (nextValue !== '' && !/^\d+$/.test(nextValue)) return
    onChange({ ...value, [axis]: nextValue })
  }

  return (
    <div className="flex flex-wrap items-center gap-1.5">
      <input
        type="text"
        pattern="[0-9]*"
        inputMode="numeric"
        maxLength={maxCoordinateLength}
        value={value.x}
        onKeyDown={handleKeyDown}
        onChange={(event) => handleCoordinateChange('x', event.target.value)}
        placeholder="X"
        aria-label={`X 坐标，范围 0-${maxCoordinate}`}
        title={`X 坐标范围 0-${maxCoordinate}`}
        className="h-8 w-16 rounded-lg border border-[var(--color-border)] bg-white px-2 text-xs font-bold text-[var(--color-text-primary)] outline-none dark:bg-slate-900"
      />
      <input
        type="text"
        pattern="[0-9]*"
        inputMode="numeric"
        maxLength={maxCoordinateLength}
        value={value.y}
        onKeyDown={handleKeyDown}
        onChange={(event) => handleCoordinateChange('y', event.target.value)}
        placeholder="Y"
        aria-label={`Y 坐标，范围 0-${maxCoordinate}`}
        title={`Y 坐标范围 0-${maxCoordinate}`}
        className="h-8 w-16 rounded-lg border border-[var(--color-border)] bg-white px-2 text-xs font-bold text-[var(--color-text-primary)] outline-none dark:bg-slate-900"
      />
      <span className="text-[10px] font-semibold text-[var(--color-text-muted)]">0-{maxCoordinate}</span>
      <button
        type="button"
        onClick={onSearch}
        title={`查找坐标 ${coordinateSummary}`}
        aria-label={`查找世界地图坐标 ${coordinateSummary}`}
        className="inline-flex h-8 items-center gap-1 rounded-lg bg-[var(--color-accent)] px-3 text-xs font-bold text-white"
      >
        <Search size={12} />
        查找
      </button>
      <button
        type="button"
        onClick={onFocusSelf}
        className="inline-flex h-8 items-center gap-1 rounded-lg border border-[var(--color-border)] px-3 text-xs font-bold text-[var(--color-text-secondary)] hover:text-[var(--color-accent)]"
      >
        <LocateFixed size={12} />
        我的城
      </button>
    </div>
  )
}

export default WorldMapCoordinateSearch
