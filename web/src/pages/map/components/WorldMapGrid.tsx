// 本文件实现世界地图的精准坐标网格，每个草地格等于世界距离 1。
import { useMemo, useRef, type FC, type KeyboardEvent, type MouseEvent, type PointerEvent, type ReactNode } from 'react'
import { LocateFixed, Minus, Plus } from 'lucide-react'
import tileCityShu from '@/assets/map/tiles/tile-city-shu.png'
import tileCityWei from '@/assets/map/tiles/tile-city-wei.png'
import tileCityWu from '@/assets/map/tiles/tile-city-wu.png'
import tileGrass from '@/assets/map/tiles/tile-grass.png'
import type { PvpTargetSummary, PvpTargetsResponse, PvpWorldPosition } from '@/types/game'
import { buildVisibleCells, buildWorldMapAxisTicks, buildWorldMapDistanceGuides, buildWorldMapTargetMetrics, buildWorldMapViewportBounds, directionFrom, formatDuration, isPositionInView, isSameGridPosition, shouldShowCellCoordinate, worldMapDragGridDelta, worldMapOverviewCoordinateFromPoint, worldMapOverviewPointStyle, worldMapOverviewTargetClass, worldMapOverviewViewportStyle, worldMapRelationBadge, worldMapRelationBadgeClass, worldMapRelationRingClass, worldMapStatusBadge, worldMapStatusBadgeClass, WORLD_MAP_MAX_VIEW_RADIUS, WORLD_MAP_MIN_VIEW_RADIUS, WORLD_MAP_VIEW_PRESETS, type GridPosition, type WorldMapAxisTick } from '../worldMapGridLogic'

interface WorldMapGridProps {
  view: PvpTargetsResponse
  targets: PvpTargetSummary[]
  overviewTargets: PvpTargetSummary[]
  hiddenInViewportCount: number
  focusedTargetId: string | null
  focusedTargetPosition: GridPosition | null
  selectedCell: GridPosition | null
  marchBadges: Record<string, string>
  showSelf: boolean
  onFocusTarget: (playerId: string) => void
  onFocusSelf: () => void
  onPan: (dx: number, dy: number) => void
  onZoom: (delta: number) => void
  onSetRadius: (radius: number) => void
  onJump: (position: GridPosition) => void
  onSelectCell: (x: number, y: number) => void
  onClearSelection: () => void
}

const CITY_TILE_BY_FACTION = {
  wei: tileCityWei,
  shu: tileCityShu,
  wu: tileCityWu,
} as const

// cityTileSrc 按国家返回城池图片，未知国家使用魏城池作为安全兜底。
function cityTileSrc(faction?: string) {
  if (faction === 'shu') return CITY_TILE_BY_FACTION.shu
  if (faction === 'wu') return CITY_TILE_BY_FACTION.wu
  return CITY_TILE_BY_FACTION.wei
}

// WorldMapGrid 渲染玩家可拖动的一格一坐标世界地图。
const WorldMapGrid: FC<WorldMapGridProps> = ({ view, targets, overviewTargets, hiddenInViewportCount, focusedTargetId, focusedTargetPosition, selectedCell, marchBadges, showSelf, onFocusTarget, onFocusSelf, onPan, onZoom, onSetRadius, onJump, onSelectCell, onClearSelection }) => {
  const radius = Math.max(1, view.radius)
  const step = 1
  const dragRef = useRef<{ startX: number; startY: number; gridX: number; gridY: number; moved: boolean } | null>(null)
  const suppressClickRef = useRef(false)
  const viewportBounds = useMemo(() => buildWorldMapViewportBounds({ x: view.centerX, y: view.centerY }, radius, view.worldSize), [radius, view.centerX, view.centerY, view.worldSize])
  const { gridSize, minX, minY, maxX: boundedMaxX, maxY: boundedMaxY, centerOffsetX, centerOffsetY } = viewportBounds
  const boundedMinX = minX
  const boundedMinY = minY
  const canPanWest = boundedMinX > 0
  const canPanEast = boundedMaxX < view.worldSize - 1
  const canPanNorth = boundedMinY > 0
  const canPanSouth = boundedMaxY < view.worldSize - 1
  const worldEdgeLabels = [
    boundedMinY === 0 ? '北界' : '',
    boundedMaxY === view.worldSize - 1 ? '南界' : '',
    boundedMinX === 0 ? '西界' : '',
    boundedMaxX === view.worldSize - 1 ? '东界' : '',
  ].filter(Boolean)
  const visibleCells = useMemo(() => buildVisibleCells(view.worldSize, minX, minY, gridSize), [gridSize, minX, minY, view.worldSize])
  const visibleTargets = useMemo(() => targets.filter((target) => isPositionInView(target.position, minX, minY, gridSize)), [gridSize, minX, minY, targets])
  const selfVisible = showSelf && isPositionInView(view.self, minX, minY, gridSize)
  const hasSelfTarget = visibleTargets.some((target) => target.relation === 'self' && target.position.x === view.self.x && target.position.y === view.self.y)
  const showCellCoordinate = shouldShowCellCoordinate(gridSize)
  const showCityDetail = showCellCoordinate
  const distanceGuides = useMemo(() => buildWorldMapDistanceGuides(radius, gridSize, undefined, centerOffsetX, centerOffsetY), [centerOffsetX, centerOffsetY, gridSize, radius])
  const viewportTargetCount = targets.length + hiddenInViewportCount
  const canZoomIn = radius > WORLD_MAP_MIN_VIEW_RADIUS
  const canZoomOut = radius < WORLD_MAP_MAX_VIEW_RADIUS
  const inspectedCell = selectedCell ?? focusedTargetPosition
  const centerCell = { x: view.centerX, y: view.centerY }
  const centerDirection = directionFrom(view.self, centerCell)
  const centerDistance = Math.abs(centerCell.x - view.self.x) + Math.abs(centerCell.y - view.self.y)
  const isInspectedAxisCell = (cell: GridPosition) => inspectedCell ? cell.x === inspectedCell.x || cell.y === inspectedCell.y : false

  // toGridStyle 将世界坐标转换成当前视野里的 CSS grid 单元。
  const toGridStyle = (position: PvpWorldPosition) => ({
    gridColumn: position.x - minX + 1,
    gridRow: position.y - minY + 1,
  })

  // handleMapPointerDown 记录拖动地图的起点。
  const handleMapPointerDown = (event: PointerEvent<HTMLDivElement>) => {
    if (event.button !== 0) return
    event.currentTarget.setPointerCapture(event.pointerId)
    dragRef.current = { startX: event.clientX, startY: event.clientY, gridX: 0, gridY: 0, moved: false }
  }

  // handleMapPointerMove 按真实格子数量移动地图中心点。
  const handleMapPointerMove = (event: PointerEvent<HTMLDivElement>) => {
    const drag = dragRef.current
    if (!drag) return
    const rect = event.currentTarget.getBoundingClientRect()
    const gridDelta = worldMapDragGridDelta({
      startX: drag.startX,
      startY: drag.startY,
      currentX: event.clientX,
      currentY: event.clientY,
      width: rect.width,
      height: rect.height,
      gridSize,
    })
    const gridX = gridDelta.x
    const gridY = gridDelta.y
    const dx = gridX - drag.gridX
    const dy = gridY - drag.gridY
    if (dx !== 0 || dy !== 0) {
      drag.gridX = gridX
      drag.gridY = gridY
      drag.moved = true
      onPan(-dx, -dy)
    }
  }

  // handleMapPointerUp 结束地图拖动并释放指针捕获。
  const handleMapPointerUp = (event: PointerEvent<HTMLDivElement>) => {
    suppressClickRef.current = dragRef.current?.moved ?? false
    dragRef.current = null
    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId)
    }
  }

  // handleMapKeyDown 允许玩家用方向键按真实格子移动地图。
  const handleMapKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (event.key === 'ArrowLeft') {
      event.preventDefault()
      onPan(-step, 0)
    } else if (event.key === 'ArrowRight') {
      event.preventDefault()
      onPan(step, 0)
    } else if (event.key === 'ArrowUp') {
      event.preventDefault()
      onPan(0, -step)
    } else if (event.key === 'ArrowDown') {
      event.preventDefault()
      onPan(0, step)
    } else if (event.key === '+' || event.key === '=') {
      event.preventDefault()
      if (canZoomIn) onZoom(-5)
    } else if (event.key === '-' || event.key === '_') {
      event.preventDefault()
      if (canZoomOut) onZoom(5)
    } else if (event.key === 'Home') {
      event.preventDefault()
      onFocusSelf()
    } else if (event.key === 'Escape') {
      event.preventDefault()
      onClearSelection()
    } else if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault()
      onSelectCell(view.centerX, view.centerY)
    }
  }

  // handleGrassClick 选中一个真实草地坐标格。
  const handleGrassClick = (x: number, y: number) => {
    if (suppressClickRef.current) {
      suppressClickRef.current = false
      return
    }
    onSelectCell(x, y)
  }

  return (
    <section className="overflow-hidden rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)]">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-[var(--color-border)] px-3 py-2">
        <div className="text-xs font-bold text-[var(--color-text-primary)]">
          世界地图
          <span className="ml-2 font-normal text-[var(--color-text-muted)]">
            中心 ({view.centerX}, {view.centerY}) · 范围 {view.radius} 格 · 当前视野 {targets.length}/{viewportTargetCount}
          </span>
          <span className="ml-2 font-normal text-[var(--color-text-muted)]">
            X {boundedMinX}-{boundedMaxX} · Y {boundedMinY}-{boundedMaxY}
          </span>
          <span className="ml-2 rounded bg-sky-500/10 px-1.5 py-0.5 text-[10px] font-black text-sky-700">
            中心距我 {centerDistance}格 · {centerDirection}
          </span>
          <span className="ml-2 rounded bg-emerald-500/10 px-1.5 py-0.5 text-[10px] font-black text-emerald-700">
            一格=距离1
          </span>
          {worldEdgeLabels.length > 0 && (
            <span className="ml-2 rounded bg-amber-500/10 px-1.5 py-0.5 text-[10px] font-black text-amber-600">
              {worldEdgeLabels.join(' · ')}
            </span>
          )}
          {inspectedCell && (
            <span className="ml-2 rounded bg-[var(--color-surface-dim)] px-1.5 py-0.5 text-[10px] font-bold text-[var(--color-text-secondary)]">
              指向 ({inspectedCell.x}, {inspectedCell.y}) · {directionFrom(view.self, inspectedCell)} · {Math.abs(inspectedCell.x - view.self.x) + Math.abs(inspectedCell.y - view.self.y)}格
            </span>
          )}
        </div>
        <div className="flex flex-wrap items-center gap-1">
          <div className="mr-1 inline-flex overflow-hidden rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
            {WORLD_MAP_VIEW_PRESETS.map((preset) => (
              <MapRadiusPresetButton
                key={preset.key}
                label={preset.label}
                active={radius === preset.radius}
                onClick={() => onSetRadius(preset.radius)}
              />
            ))}
          </div>
          <MapControlButton label="左移" disabled={!canPanWest} onClick={() => onPan(-step, 0)}>←</MapControlButton>
          <MapControlButton label="上移" disabled={!canPanNorth} onClick={() => onPan(0, -step)}>↑</MapControlButton>
          <MapControlButton label="下移" disabled={!canPanSouth} onClick={() => onPan(0, step)}>↓</MapControlButton>
          <MapControlButton label="右移" disabled={!canPanEast} onClick={() => onPan(step, 0)}>→</MapControlButton>
          <button
            type="button"
            onClick={() => onZoom(-5)}
            disabled={!canZoomIn}
            className="inline-flex h-7 w-7 items-center justify-center rounded-lg border border-[var(--color-border)] text-[var(--color-text-secondary)] hover:text-[var(--color-accent)] disabled:cursor-not-allowed disabled:opacity-40"
            title="放大格子"
          >
            <Plus size={13} />
          </button>
          <button
            type="button"
            onClick={() => onZoom(5)}
            disabled={!canZoomOut}
            className="inline-flex h-7 w-7 items-center justify-center rounded-lg border border-[var(--color-border)] text-[var(--color-text-secondary)] hover:text-[var(--color-accent)] disabled:cursor-not-allowed disabled:opacity-40"
            title="缩小格子"
          >
            <Minus size={13} />
          </button>
          <button
            type="button"
            onClick={onFocusSelf}
            className="inline-flex h-7 items-center gap-1 rounded-lg border border-[var(--color-border)] px-2 text-[10px] font-bold text-[var(--color-text-secondary)] hover:text-[var(--color-accent)]"
          >
            <LocateFixed size={12} />
            我的城
          </button>
        </div>
      </div>

      <div className="relative flex h-[min(76vw,520px)] min-h-[260px] items-center justify-center overflow-hidden bg-[var(--color-surface-dim)] p-3 sm:min-h-[320px] sm:p-5">
        <div
          className="relative z-10 aspect-square h-full max-h-full max-w-full cursor-grab touch-none active:cursor-grabbing"
          tabIndex={0}
          role="application"
          aria-label="世界地图，可用方向键按格移动，加号放大格子，减号缩小格子，Home 返回我的城，回车选择中心格，Escape 取消选择"
          onPointerDown={handleMapPointerDown}
          onPointerMove={handleMapPointerMove}
          onPointerUp={handleMapPointerUp}
          onPointerCancel={handleMapPointerUp}
          onKeyDown={handleMapKeyDown}
        >
          <MapAxisLabels view={view} gridSize={gridSize} minX={minX} minY={minY} maxX={boundedMaxX} maxY={boundedMaxY} />
          <div
            className="relative h-full w-full border border-emerald-950/20 bg-[#5f8f3e] shadow-inner"
            style={{
              display: 'grid',
              gridTemplateColumns: `repeat(${gridSize}, minmax(0, 1fr))`,
              gridTemplateRows: `repeat(${gridSize}, minmax(0, 1fr))`,
            }}
          >
            {visibleCells.map((cell) => (
              <button
                key={`${cell.x}:${cell.y}`}
                type="button"
                onClick={() => handleGrassClick(cell.x, cell.y)}
                aria-label={`草地坐标格 (${cell.x}, ${cell.y})，距离按 1 格计算`}
                className={`relative h-full w-full border border-emerald-950/10 bg-[#6f9b52] bg-cover bg-center bg-no-repeat p-0 hover:brightness-110 ${isInspectedAxisCell(cell) ? 'brightness-110 saturate-125' : ''} ${isSameGridPosition(centerCell, cell) ? 'ring-2 ring-emerald-950/45 ring-inset' : ''} ${isSameGridPosition(selectedCell, cell) ? 'z-10 outline outline-2 outline-amber-300 outline-offset-[-2px]' : ''}`}
                style={{ gridColumn: cell.x - minX + 1, gridRow: cell.y - minY + 1, backgroundImage: `url(${tileGrass})` }}
                title={`草地坐标格 (${cell.x}, ${cell.y})`}
              >
                {isSameGridPosition(centerCell, cell) && (
                  <span className="pointer-events-none absolute left-1/2 top-1/2 z-20 h-2 w-2 -translate-x-1/2 -translate-y-1/2 rounded-full border border-white/70 bg-emerald-950/70 shadow-sm" title="中心格" />
                )}
                {inspectedCell && cell.x === inspectedCell.x && (
                  <span className="pointer-events-none absolute inset-y-0 left-1/2 z-10 w-px -translate-x-1/2 bg-amber-300/50" aria-hidden="true" />
                )}
                {inspectedCell && cell.y === inspectedCell.y && (
                  <span className="pointer-events-none absolute left-0 top-1/2 z-10 h-px w-full -translate-y-1/2 bg-amber-300/50" aria-hidden="true" />
                )}
              </button>
            ))}
            <MapDistanceGuides guides={distanceGuides} gridSize={gridSize} centerOffsetX={centerOffsetX} centerOffsetY={centerOffsetY} />
            <MapSelectedRoute self={view.self} target={inspectedCell} minX={minX} minY={minY} gridSize={gridSize} />
            {selfVisible && !hasSelfTarget && (
              <button
                type="button"
                className="z-20 flex h-full w-full items-center justify-center"
                style={toGridStyle(view.self)}
                title={`我的城池 (${view.self.x}, ${view.self.y})`}
                aria-label={`我的城池，坐标 (${view.self.x}, ${view.self.y})`}
                onPointerDown={(event) => event.stopPropagation()}
                onClick={(event) => {
                  event.stopPropagation()
                  onFocusSelf()
                }}
              >
                <span className={`relative flex aspect-square h-full w-full min-h-3 items-center justify-center overflow-hidden rounded-[1px] border border-sky-800 text-[9px] font-black text-white shadow-[inset_0_0_0_1px_rgba(255,255,255,0.25)] ${worldMapRelationRingClass('self')}`}>
                  <img className="pointer-events-none absolute inset-0 h-full w-full object-cover" src={cityTileSrc()} alt="" aria-hidden="true" />
                  {!showCityDetail && <span className="relative z-20 rounded bg-sky-950/75 px-0.5 leading-none" aria-hidden="true">我</span>}
                </span>
              </button>
            )}
            {visibleTargets.map((target) => {
              const focused = target.playerId === focusedTargetId
              const self = target.relation === 'self'
              const isYellowTurban = target.targetType === 'yellow_turban'
              const statusBadge = worldMapStatusBadge(target.status)
              const marchBadge = marchBadges[target.playerId]
              const metrics = buildWorldMapTargetMetrics(view.self, target.position)
              return (
                <button
                  key={target.playerId}
                  type="button"
                  onPointerDown={(event) => event.stopPropagation()}
                  onClick={(event) => {
                    event.stopPropagation()
                    onFocusTarget(target.playerId)
                  }}
                  className={`z-20 flex h-full w-full items-center justify-center bg-transparent transition-colors hover:z-30 ${focused ? 'ring-2 ring-white/75' : ''}`}
                  style={toGridStyle(target.position)}
                  title={`${target.nickname} · ${metrics.direction} · (${target.position.x}, ${target.position.y}) 距离 ${metrics.distance} · 行军约 ${formatDuration(metrics.seconds)}`}
                  aria-label={`选中${target.nickname}，${metrics.direction}，坐标 (${target.position.x}, ${target.position.y})，距离 ${metrics.distance} 格，行军约 ${formatDuration(metrics.seconds)}`}
                >
                  <span className={`relative flex aspect-square h-full w-full min-h-3 items-center justify-center overflow-hidden rounded-[1px] border text-[8px] font-black text-white shadow-[inset_0_0_0_1px_rgba(255,255,255,0.25)] ${isYellowTurban ? 'border-amber-500 ring-1 ring-amber-400' : `border-white/40 ${worldMapRelationRingClass(target.relation)}`} ${self ? 'border-sky-800' : ''}`}>
                    <img className="pointer-events-none absolute inset-0 h-full w-full object-cover" src={cityTileSrc(target.faction)} alt="" aria-hidden="true" />
                    {isYellowTurban && (
                      <span className="absolute inset-0 z-10 flex items-center justify-center bg-amber-500/30 text-[9px] font-black text-yellow-50">
                        黄
                      </span>
                    )}
                    {showCityDetail && (
                      <span className={`absolute right-0 top-0 z-20 flex h-3 min-w-3 items-center justify-center rounded-bl-[1px] px-0.5 text-[7px] leading-none ${worldMapRelationBadgeClass(target.relation)}`}>
                        {worldMapRelationBadge(target.relation)}
                      </span>
                    )}
                    {showCityDetail && statusBadge && (
                      <span className={`absolute bottom-0 right-0 z-20 flex h-3 min-w-3 items-center justify-center rounded-tl-[1px] px-0.5 text-[7px] leading-none ${worldMapStatusBadgeClass(target.status)}`}>
                        {statusBadge}
                      </span>
                    )}
                    {showCityDetail && marchBadge && (
                      <span className="absolute bottom-0 left-0 z-20 flex h-3 min-w-3 items-center justify-center rounded-tr-[1px] bg-violet-600 px-0.5 text-[7px] leading-none text-white">
                        {marchBadge}
                      </span>
                    )}
                    {focused && (
                      <span className="pointer-events-none absolute inset-x-0 top-0 z-40 truncate bg-slate-950/75 px-0.5 py-px text-[7px] font-bold leading-none text-white shadow-sm">
                        {target.nickname} · {metrics.distance}格
                      </span>
                    )}
                    {self && !showCityDetail && <span className="relative z-20 rounded bg-sky-950/75 px-0.5 leading-none" aria-hidden="true">我</span>}
                  </span>
                </button>
              )
            })}
          </div>
          {visibleTargets.length === 0 && !selfVisible && (
            <div className="absolute inset-0 flex items-center justify-center text-xs text-[var(--color-text-muted)]">
              {hiddenInViewportCount > 0 ? `当前视野有 ${hiddenInViewportCount} 个玩家城池已被筛选隐藏。` : '当前视野没有玩家城池，可以缩小比例或移动地图。'}
            </div>
          )}
          <MapOverview view={view} radius={radius} selectedPosition={selectedCell ?? focusedTargetPosition} targets={overviewTargets} showSelf={showSelf} onJump={onJump} />
        </div>
      </div>
    </section>
  )
}

// MapSelectedRoute 渲染从自己城池到当前指向格的曼哈顿距离折线。
const MapSelectedRoute: FC<{ self: GridPosition; target: GridPosition | null | undefined; minX: number; minY: number; gridSize: number }> = ({ self, target, minX, minY, gridSize }) => {
  if (!target) return null
  if (isSameGridPosition(self, target)) return null
  if (!isPositionInView(self, minX, minY, gridSize) || !isPositionInView(target, minX, minY, gridSize)) return null
  const selfX = ((self.x - minX + 0.5) / gridSize) * 100
  const selfY = ((self.y - minY + 0.5) / gridSize) * 100
  const targetX = ((target.x - minX + 0.5) / gridSize) * 100
  const targetY = ((target.y - minY + 0.5) / gridSize) * 100
  const horizontalLeft = Math.min(selfX, targetX)
  const horizontalWidth = Math.abs(targetX - selfX)
  const verticalTop = Math.min(selfY, targetY)
  const verticalHeight = Math.abs(targetY - selfY)
  return (
    <div className="pointer-events-none absolute inset-0 z-[15]" aria-hidden="true">
      {horizontalWidth > 0 && (
        <span
          className="absolute h-0.5 -translate-y-1/2 rounded bg-amber-300/80 shadow-[0_0_4px_rgba(251,191,36,0.7)]"
          style={{ left: `${horizontalLeft}%`, top: `${selfY}%`, width: `${horizontalWidth}%` }}
        />
      )}
      {verticalHeight > 0 && (
        <span
          className="absolute w-0.5 -translate-x-1/2 rounded bg-amber-300/80 shadow-[0_0_4px_rgba(251,191,36,0.7)]"
          style={{ left: `${targetX}%`, top: `${verticalTop}%`, height: `${verticalHeight}%` }}
        />
      )}
    </div>
  )
}

// MapOverview 渲染世界地图右下角全局视野概览。
const MapOverview: FC<{ view: PvpTargetsResponse; radius: number; selectedPosition: GridPosition | null; targets: PvpTargetSummary[]; showSelf: boolean; onJump: (position: GridPosition) => void }> = ({ view, radius, selectedPosition, targets, showSelf, onJump }) => {
  const viewportStyle = worldMapOverviewViewportStyle({ x: view.centerX, y: view.centerY }, radius, view.worldSize)
  const selfStyle = worldMapOverviewPointStyle(view.self, view.worldSize)
  const selectedStyle = selectedPosition ? worldMapOverviewPointStyle(selectedPosition, view.worldSize) : null
  const grassCellSize = `${100 / view.worldSize}% ${100 / view.worldSize}%`
  const handleOverviewClick = (event: MouseEvent<HTMLButtonElement>) => {
    const rect = event.currentTarget.getBoundingClientRect()
    onJump(worldMapOverviewCoordinateFromPoint({
      clientX: event.clientX,
      clientY: event.clientY,
      left: rect.left,
      top: rect.top,
      width: rect.width,
      height: rect.height,
      worldSize: view.worldSize,
    }))
  }
  return (
    <button
      type="button"
      onClick={handleOverviewClick}
      onPointerDown={(event) => event.stopPropagation()}
      className="absolute bottom-3 right-3 z-30 h-32 w-32 cursor-crosshair border border-emerald-950/40 bg-[#6f9b52]/90 p-0 shadow-md sm:h-36 sm:w-36"
      title={`点击概览跳转地图坐标，当前中心 (${view.centerX}, ${view.centerY})，我的城池 (${view.self.x}, ${view.self.y})`}
      aria-label={`世界地图概览，点击跳转坐标，当前中心 (${view.centerX}, ${view.centerY})，我的城池 (${view.self.x}, ${view.self.y})`}
    >
      <div
        className="absolute inset-0 bg-[linear-gradient(90deg,rgba(255,255,255,0.15)_1px,transparent_1px),linear-gradient(180deg,rgba(255,255,255,0.15)_1px,transparent_1px)]"
        style={{ backgroundSize: grassCellSize }}
      />
      <MapOverviewDirectionLabels />
      <div className="absolute border border-white bg-white/15" style={viewportStyle} />
      {targets.map((target) => {
        if (target.relation === 'self' && isSameGridPosition(target.position, view.self)) return null
        return (
          <div
            key={target.playerId}
            className={`absolute z-10 h-2 w-2 -translate-x-1/2 -translate-y-1/2 rounded-[1px] border border-white/70 shadow-sm ${worldMapOverviewTargetClass(target.faction)}`}
            style={worldMapOverviewPointStyle(target.position, view.worldSize)}
            title={`${target.nickname} (${target.position.x}, ${target.position.y})`}
            aria-hidden="true"
          />
        )
      })}
      {showSelf && (
        <div className="absolute z-20 h-2.5 w-2.5 -translate-x-1/2 -translate-y-1/2 rounded-[1px] border border-white bg-sky-500 shadow-sm" style={selfStyle} title={`我的城池 (${view.self.x}, ${view.self.y})`} aria-hidden="true" />
      )}
      {selectedStyle && (
        <div className="absolute z-30 h-2.5 w-2.5 -translate-x-1/2 -translate-y-1/2 rounded-[1px] border border-white bg-amber-400 shadow-sm" style={selectedStyle} title={`当前选中 (${selectedPosition?.x}, ${selectedPosition?.y})`} aria-hidden="true" />
      )}
    </button>
  )
}

// MapOverviewDirectionLabels 渲染全图概览的四向标识。
const MapOverviewDirectionLabels: FC = () => (
  <div className="pointer-events-none absolute inset-0 z-40 text-[8px] font-black leading-none text-white/85 drop-shadow" aria-hidden="true">
    <span className="absolute left-1/2 top-1 -translate-x-1/2">北</span>
    <span className="absolute bottom-1 left-1/2 -translate-x-1/2">南</span>
    <span className="absolute left-1 top-1/2 -translate-y-1/2">西</span>
    <span className="absolute right-1 top-1/2 -translate-y-1/2">东</span>
  </div>
)

// MapControlButton 渲染地图方向控制按钮。
const MapControlButton: FC<{ label: string; disabled?: boolean; onClick: () => void; children: ReactNode }> = ({ label, disabled = false, onClick, children }) => (
  <button
    type="button"
    disabled={disabled}
    onClick={onClick}
    title={label}
    className="inline-flex h-7 w-7 items-center justify-center rounded-lg border border-[var(--color-border)] text-xs font-black text-[var(--color-text-secondary)] hover:text-[var(--color-accent)] disabled:cursor-not-allowed disabled:opacity-40"
  >
    {children}
  </button>
)

// MapRadiusPresetButton 渲染地图视野密度预设按钮。
const MapRadiusPresetButton: FC<{ label: string; active: boolean; onClick: () => void }> = ({ label, active, onClick }) => (
  <button
    type="button"
    onClick={onClick}
    title={label}
    aria-pressed={active}
    className={`h-7 px-2 text-[10px] font-black ${active ? 'bg-[var(--color-accent)] text-white' : 'text-[var(--color-text-secondary)] hover:text-[var(--color-accent)]'}`}
  >
    {label}
  </button>
)

// MapDistanceGuides 渲染以当前中心为基准的距离辅助环。
const MapDistanceGuides: FC<{ guides: ReturnType<typeof buildWorldMapDistanceGuides>; gridSize: number; centerOffsetX: number; centerOffsetY: number }> = ({ guides, gridSize, centerOffsetX, centerOffsetY }) => {
  const centerLeft = `${((centerOffsetX + 0.5) / gridSize) * 100}%`
  const centerTop = `${((centerOffsetY + 0.5) / gridSize) * 100}%`
  return (
    <>
    <div className="pointer-events-none absolute inset-y-0 z-10 w-px -translate-x-1/2 bg-emerald-950/25" style={{ left: centerLeft }} />
    <div className="pointer-events-none absolute inset-x-0 z-10 h-px -translate-y-1/2 bg-emerald-950/25" style={{ top: centerTop }} />
    <svg className="pointer-events-none absolute inset-0 z-10 h-full w-full" viewBox="0 0 100 100" preserveAspectRatio="none" aria-hidden="true">
      {guides.map((guide) => (
        <polygon
          key={guide.distance}
          points={guide.points}
          fill="none"
          stroke="rgba(6,78,59,0.35)"
          strokeWidth="0.35"
          vectorEffect="non-scaling-stroke"
        />
      ))}
    </svg>
    {guides.map((guide) => (
      <span
        key={guide.distance}
        className="pointer-events-none absolute z-20 -translate-x-1/2 rounded bg-emerald-950/60 px-1 py-0.5 text-[8px] font-bold leading-none text-white"
        style={{ left: guide.labelLeft, top: guide.labelTop }}
      >
        {guide.distance}格
      </span>
    ))}
    </>
  )
}

// MapAxisLabels 显示当前视野边缘坐标和坐标轴刻度，帮助玩家判断方向。
const MapAxisLabels: FC<{ view: PvpTargetsResponse; gridSize: number; minX: number; minY: number; maxX: number; maxY: number }> = ({ view, gridSize, minX, minY, maxX, maxY }) => {
  const west = minX
  const east = maxX
  const north = minY
  const south = maxY
  const xTicks = buildWorldMapAxisTicks(view.worldSize, minX, gridSize)
  const yTicks = buildWorldMapAxisTicks(view.worldSize, minY, gridSize)
  return (
    <>
      <div className="pointer-events-none absolute left-1/2 top-2 -translate-x-1/2 rounded bg-[var(--color-surface)]/85 px-2 py-0.5 text-[10px] font-black text-[var(--color-text-secondary)] shadow-sm">北 y {north}</div>
      <div className="pointer-events-none absolute bottom-2 left-1/2 -translate-x-1/2 rounded bg-[var(--color-surface)]/85 px-2 py-0.5 text-[10px] font-black text-[var(--color-text-secondary)] shadow-sm">南 y {south}</div>
      <div className="pointer-events-none absolute left-2 top-1/2 -translate-y-1/2 rounded bg-[var(--color-surface)]/85 px-2 py-0.5 text-[10px] font-black text-[var(--color-text-secondary)] shadow-sm">西 x {west}</div>
      <div className="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 rounded bg-[var(--color-surface)]/85 px-2 py-0.5 text-[10px] font-black text-[var(--color-text-secondary)] shadow-sm">东 x {east}</div>
      <AxisTicks axis="x" ticks={xTicks} />
      <AxisTicks axis="y" ticks={yTicks} />
    </>
  )
}

// AxisTicks 渲染坐标轴上的离散刻度。
const AxisTicks: FC<{ axis: 'x' | 'y'; ticks: WorldMapAxisTick[] }> = ({ axis, ticks }) => (
  <>
    {ticks.map((tick) => (
      <span
        key={`${axis}:${tick.value}`}
        className={`pointer-events-none absolute z-30 rounded bg-emerald-950/55 px-1 py-0.5 text-[8px] font-bold leading-none text-white shadow-sm ${axis === 'x' ? 'top-7 -translate-x-1/2' : 'left-7 -translate-y-1/2'}`}
        style={axis === 'x' ? { left: tick.percent } : { top: tick.percent }}
      >
        {tick.value}
      </span>
    ))}
  </>
)

export default WorldMapGrid
