// 本文件提供世界地图网格的纯计算逻辑，供组件和测试复用。

export interface GridPosition {
  x: number
  y: number
}

export interface WorldMapActionTarget {
  canScout?: boolean
  canReinforce?: boolean
  canAttack?: boolean
  canPlunder?: boolean
  reason?: string
  scoutReason?: string
  attackReason?: string
  plunderReason?: string
  reinforceReason?: string
}

export interface WorldMapPositionedTarget {
  position: GridPosition
  relation?: string
}

export interface WorldMapDistanceTarget extends WorldMapPositionedTarget {
  distance: number
}

export interface WorldMapTargetMetrics {
  direction: string
  distance: number
  seconds: number
}

export interface WorldMapCacheTarget extends WorldMapPositionedTarget {
  playerId: string
}

export interface WorldMapRelationCounts {
  self: number
  ally: number
  other: number
}

export interface WorldMapFactionCounts {
  wei: number
  shu: number
  wu: number
  other: number
}

export interface WorldMapFactionTarget {
  faction?: string
}

export interface WorldMapActionState {
  key: 'scout' | 'reinforce' | 'attack' | 'plunder'
  label: string
  disabled: boolean
  reason: string
  busy: boolean
}

export interface WorldMapAxisTick {
  value: number
  percent: string
}

export interface WorldMapDistanceGuide {
  distance: number
  side: number
  points: string
  labelLeft: string
  labelTop: string
}

export interface WorldMapViewportBounds {
  minX: number
  maxX: number
  minY: number
  maxY: number
  gridSize: number
  centerOffsetX: number
  centerOffsetY: number
}

export interface WorldMapMarchLike {
  attackerPlayerId: string
  defenderPlayerId: string
  status: string
  arrivesAt?: string
  returnsAt?: string
}

export interface WorldMapReinforcementLike {
  fromPlayerId: string
  toPlayerId: string
  status: string
  arriveAt?: string
  expectedReturnedAt?: string
}

export interface WorldMapMarchSummary {
  outgoing: number
  returning: number
  incoming: number
  resolving: number
}

export interface WorldMapDragDeltaInput {
  startX: number
  startY: number
  currentX: number
  currentY: number
  width: number
  height: number
  gridSize: number
}

export interface WorldMapOverviewPointerInput {
  clientX: number
  clientY: number
  left: number
  top: number
  width: number
  height: number
  worldSize: number
}

export interface WorldMapCoordinateSearchValue {
  x: string
  y: string
}

export interface WorldMapCoordinateSearchResult {
  position: GridPosition | null
  error: string
}

export const WORLD_MAP_FACTION_LEGEND = [
  { faction: 'wei', label: '魏', colorClass: 'bg-blue-500' },
  { faction: 'shu', label: '蜀', colorClass: 'bg-emerald-500' },
  { faction: 'wu', label: '吴', colorClass: 'bg-red-500' },
  { faction: 'other', label: '其它', colorClass: 'bg-stone-500' },
]

export const WORLD_MAP_MARCH_BADGE_LEGEND = [
  { badge: '出', label: '出征' },
  { badge: '返', label: '返程' },
  { badge: '袭', label: '被袭' },
  { badge: '结', label: '结算' },
]

export const WORLD_MAP_RELATION_BADGE_LEGEND = [
  { relation: 'self', badge: '己', label: '自己' },
  { relation: 'ally', badge: '盟', label: '同盟' },
  { relation: 'other', badge: '他', label: '其他' },
]

export const WORLD_MAP_STATUS_BADGE_LEGEND = [
  { status: 'protected', badge: '保', label: '保护' },
  { status: 'truce', badge: '免', label: '免战' },
  { status: 'unavailable', badge: '禁', label: '不可操作' },
]

export const WORLD_MAP_FULL_LOAD_RADIUS = 100
export const WORLD_MAP_SECONDS_PER_GRID_AT_SPEED_ONE = 300
export const WORLD_MAP_MAX_MARCH_SECONDS = 3 * 3600
export const WORLD_MAP_MAX_COORDINATE_LABEL_GRID_SIZE = 31
export const WORLD_MAP_MIN_VIEW_RADIUS = 5
export const WORLD_MAP_MAX_VIEW_RADIUS = 30
export const WORLD_MAP_MAX_AXIS_TICKS = 7
export const WORLD_MAP_DISTANCE_GUIDE_STEP = 5
export const WORLD_MAP_GRASS_VARIANT_COUNT = 5
export const WORLD_MAP_VIEW_PRESETS = [
  { key: 'near', label: '近景', radius: 5 },
  { key: 'middle', label: '中景', radius: 10 },
  { key: 'far', label: '远景', radius: 30 },
] as const

// clampWorldMapCenter 把地图中心点限制在世界坐标范围内。
export function clampWorldMapCenter(position: GridPosition, worldSize: number) {
  const max = Math.max(0, Math.floor(worldSize) - 1)
  return {
    x: Math.max(0, Math.min(max, Math.floor(position.x))),
    y: Math.max(0, Math.min(max, Math.floor(position.y))),
  }
}

// moveWorldMapCenter 按格子偏移移动地图中心，并限制在世界范围内。
export function moveWorldMapCenter(current: GridPosition, dx: number, dy: number, worldSize: number) {
  return clampWorldMapCenter({ x: current.x + dx, y: current.y + dy }, worldSize)
}

// parseWorldMapCoordinateSearch 校验坐标查找输入并返回合法世界坐标。
export function parseWorldMapCoordinateSearch(value: WorldMapCoordinateSearchValue, worldSize: number): WorldMapCoordinateSearchResult {
  const max = Math.max(0, Math.floor(worldSize) - 1)
  const rawX = value.x.trim()
  const rawY = value.y.trim()
  if (rawX === '' || rawY === '') {
    return {
      position: null,
      error: `请输入 0-${max} 范围内的坐标`,
    }
  }
  const x = Number(rawX)
  const y = Number(rawY)
  if (!Number.isInteger(x) || !Number.isInteger(y) || x < 0 || x > max || y < 0 || y > max) {
    return {
      position: null,
      error: `请输入 0-${max} 范围内的坐标`,
    }
  }
  return {
    position: { x, y },
    error: '',
  }
}

// worldMapDragGridDelta 将指针拖动像素转换为跨过的真实地图格子数。
export function worldMapDragGridDelta(input: WorldMapDragDeltaInput) {
  const safeGridSize = Math.max(1, Math.floor(input.gridSize))
  const cellSize = Math.max(1, Math.min(input.width, input.height) / safeGridSize)
  const x = Math.trunc((input.currentX - input.startX) / cellSize)
  const y = Math.trunc((input.currentY - input.startY) / cellSize)
  return {
    x: Object.is(x, -0) ? 0 : x,
    y: Object.is(y, -0) ? 0 : y,
  }
}

// clampWorldMapRadius 限制前端当前视野半径，避免格子过大或过密。
export function clampWorldMapRadius(radius: number) {
  return Math.max(WORLD_MAP_MIN_VIEW_RADIUS, Math.min(WORLD_MAP_MAX_VIEW_RADIUS, Math.floor(radius)))
}

// worldMapOverviewPointStyle 把世界坐标转换成概览图百分比点位。
export function worldMapOverviewPointStyle(position: GridPosition, worldSize: number) {
  const size = Math.max(1, Math.floor(worldSize))
  const center = clampWorldMapCenter(position, size)
  return {
    left: `${((center.x + 0.5) / size) * 100}%`,
    top: `${((center.y + 0.5) / size) * 100}%`,
  }
}

// worldMapOverviewViewportStyle 把当前视野转换成概览图里的矩形框。
export function worldMapOverviewViewportStyle(center: GridPosition, radius: number, worldSize: number) {
  const size = Math.max(1, Math.floor(worldSize))
  const bounds = buildWorldMapViewportBounds(center, radius, size)
  return {
    left: `${(bounds.minX / size) * 100}%`,
    top: `${(bounds.minY / size) * 100}%`,
    width: `${((bounds.maxX - bounds.minX + 1) / size) * 100}%`,
    height: `${((bounds.maxY - bounds.minY + 1) / size) * 100}%`,
  }
}

// worldMapOverviewCoordinateFromPoint 把概览图点击位置换算成真实世界坐标。
export function worldMapOverviewCoordinateFromPoint(input: WorldMapOverviewPointerInput) {
  const size = Math.max(1, Math.floor(input.worldSize))
  const width = Math.max(1, input.width)
  const height = Math.max(1, input.height)
  const x = Math.floor(((input.clientX - input.left) / width) * size)
  const y = Math.floor(((input.clientY - input.top) / height) * size)
  return clampWorldMapCenter({ x, y }, size)
}

// buildVisibleCells 按当前视野生成真实可点击草地格。
export function buildVisibleCells(worldSize: number, minX: number, minY: number, gridSize: number) {
  const cells: GridPosition[] = []
  for (let y = minY; y < minY + gridSize; y += 1) {
    for (let x = minX; x < minX + gridSize; x += 1) {
      if (x < 0 || y < 0 || x >= worldSize || y >= worldSize) continue
      cells.push({ x, y })
    }
  }
  return cells
}

// buildWorldMapViewportBounds 计算贴边时仍补满的方形地图视野边界。
export function buildWorldMapViewportBounds(center: GridPosition, radius: number, worldSize: number): WorldMapViewportBounds {
  const size = Math.max(1, Math.floor(worldSize))
  const safeRadius = Math.max(0, Math.floor(radius))
  const gridSize = Math.min(size, safeRadius * 2 + 1)
  const safeCenter = clampWorldMapCenter(center, size)
  const maxMin = Math.max(0, size - gridSize)
  const minX = Math.max(0, Math.min(maxMin, safeCenter.x - safeRadius))
  const minY = Math.max(0, Math.min(maxMin, safeCenter.y - safeRadius))
  const maxX = minX + gridSize - 1
  const maxY = minY + gridSize - 1
  return {
    minX,
    maxX,
    minY,
    maxY,
    gridSize,
    centerOffsetX: safeCenter.x - minX,
    centerOffsetY: safeCenter.y - minY,
  }
}

// worldMapGrassCellClass 按坐标生成稳定草地格纹理，保持一格一坐标不漂移。
export function worldMapGrassCellClass(position: GridPosition) {
  const variant = Math.abs((position.x * 31 + position.y * 17 + position.x * position.y) % WORLD_MAP_GRASS_VARIANT_COUNT)
  if (variant === 0) return 'bg-[#78a85a] bg-[radial-gradient(circle_at_25%_25%,rgba(255,255,255,0.16)_0_1px,transparent_1px),linear-gradient(135deg,rgba(42,80,35,0.14)_25%,transparent_25%,transparent_75%,rgba(42,80,35,0.14)_75%)]'
  if (variant === 1) return 'bg-[#6f9d50] bg-[linear-gradient(45deg,rgba(36,76,33,0.18)_0_1px,transparent_1px),linear-gradient(135deg,rgba(255,255,255,0.10)_0_1px,transparent_1px)]'
  if (variant === 2) return 'bg-[#82b764] bg-[radial-gradient(circle_at_70%_65%,rgba(43,91,39,0.20)_0_1px,transparent_1px)]'
  if (variant === 3) return 'bg-[#74a754] bg-[linear-gradient(90deg,rgba(255,255,255,0.10)_0_1px,transparent_1px),linear-gradient(180deg,rgba(31,74,35,0.13)_0_1px,transparent_1px)]'
  return 'bg-[#7dad5d] bg-[radial-gradient(circle_at_35%_70%,rgba(255,255,255,0.18)_0_1px,transparent_1px),radial-gradient(circle_at_78%_24%,rgba(37,78,34,0.18)_0_1px,transparent_1px)]'
}

// buildWorldMapAxisTicks 生成坐标轴刻度，刻度位置与真实格子中心对齐。
export function buildWorldMapAxisTicks(worldSize: number, minCoordinate: number, gridSize: number, maxTicks = WORLD_MAP_MAX_AXIS_TICKS): WorldMapAxisTick[] {
  const size = Math.max(1, Math.floor(worldSize))
  const safeGridSize = Math.max(1, Math.floor(gridSize))
  const start = Math.max(0, Math.floor(minCoordinate))
  const end = Math.min(size - 1, Math.floor(minCoordinate) + safeGridSize - 1)
  if (end < start) return []
  const span = end - start + 1
  const step = worldMapAxisTickStep(span, maxTicks)
  const values: number[] = [start]
  const alignedStart = Math.ceil(start / step) * step
  for (let value = alignedStart; value <= end; value += step) {
    if (!values.includes(value)) values.push(value)
  }
  if (!values.includes(end)) values.push(end)
  return values.map((value) => ({
    value,
    percent: `${((value - minCoordinate + 0.5) / safeGridSize) * 100}%`,
  }))
}

// buildWorldMapDistanceGuides 生成以当前中心为原点的 5 格倍数距离辅助环。
export function buildWorldMapDistanceGuides(radius: number, gridSize: number, step = WORLD_MAP_DISTANCE_GUIDE_STEP, centerOffsetX = radius, centerOffsetY = radius): WorldMapDistanceGuide[] {
  const safeRadius = Math.max(1, Math.floor(radius))
  const safeGridSize = Math.max(1, Math.floor(gridSize))
  const safeStep = Math.max(1, Math.floor(step))
  const centerX = Math.max(0, Math.min(safeGridSize - 1, Math.floor(centerOffsetX)))
  const centerY = Math.max(0, Math.min(safeGridSize - 1, Math.floor(centerOffsetY)))
  const guides: WorldMapDistanceGuide[] = []
  const point = (x: number, y: number) => `${((x + 0.5) / safeGridSize) * 100},${((y + 0.5) / safeGridSize) * 100}`
  for (let distance = safeStep; distance <= safeRadius; distance += safeStep) {
    const side = distance * 2 + 1
    guides.push({
      distance,
      side,
      points: [
        point(centerX, centerY - distance),
        point(centerX + distance, centerY),
        point(centerX, centerY + distance),
        point(centerX - distance, centerY),
      ].join(' '),
      labelLeft: `${((centerX + 0.5) / safeGridSize) * 100}%`,
      labelTop: `${((centerY - distance + 0.5) / safeGridSize) * 100}%`,
    })
  }
  return guides
}

// worldMapAxisTickStep 选择易读的坐标轴刻度间隔。
function worldMapAxisTickStep(span: number, maxTicks: number) {
  const rawStep = Math.max(1, Math.ceil(span / Math.max(1, maxTicks - 1)))
  if (rawStep <= 1) return 1
  if (rawStep <= 2) return 2
  if (rawStep <= 5) return 5
  if (rawStep <= 10) return 10
  if (rawStep <= 20) return 20
  return Math.ceil(rawStep / 25) * 25
}

// isPositionInView 判断坐标是否落在当前地图格子视野内。
export function isPositionInView(position: GridPosition, minX: number, minY: number, gridSize: number) {
  return position.x >= minX && position.x < minX + gridSize && position.y >= minY && position.y < minY + gridSize
}

// findWorldMapTargetAtCell 从缓存目标中查找指定格子的玩家城池。
export function findWorldMapTargetAtCell<T extends WorldMapPositionedTarget>(targets: T[], cell: GridPosition) {
  return targets.find((target) => target.position.x === cell.x && target.position.y === cell.y) ?? null
}

// isWorldMapRelationVisible 判断指定关系是否通过当前筛选。
export function isWorldMapRelationVisible(relation: string | undefined, relationFilters: Record<string, boolean>) {
  return relationFilters[relation ?? 'other'] !== false
}

// findVisibleWorldMapTargetAtCell 从当前筛选可见目标中查找指定格子的玩家城池。
export function findVisibleWorldMapTargetAtCell<T extends WorldMapPositionedTarget>(targets: T[], cell: GridPosition, relationFilters: Record<string, boolean>) {
  return targets.find((target) => {
    return target.position.x === cell.x && target.position.y === cell.y && isWorldMapRelationVisible(target.relation, relationFilters)
  }) ?? null
}

// buildWorldMapRelationCounts 统计当前缓存玩家城池的关系数量。
export function buildWorldMapRelationCounts<T extends WorldMapPositionedTarget>(targets: T[]): WorldMapRelationCounts {
  const counts: WorldMapRelationCounts = { self: 0, ally: 0, other: 0 }
  for (const target of targets) {
    if (target.relation === 'self') {
      counts.self += 1
    } else if (target.relation === 'ally') {
      counts.ally += 1
    } else {
      counts.other += 1
    }
  }
  return counts
}

// buildWorldMapFactionCounts 统计当前缓存玩家城池的国家数量。
export function buildWorldMapFactionCounts<T extends WorldMapFactionTarget>(targets: T[]): WorldMapFactionCounts {
  const counts: WorldMapFactionCounts = { wei: 0, shu: 0, wu: 0, other: 0 }
  for (const target of targets) {
    if (target.faction === 'wei') {
      counts.wei += 1
    } else if (target.faction === 'shu') {
      counts.shu += 1
    } else if (target.faction === 'wu') {
      counts.wu += 1
    } else {
      counts.other += 1
    }
  }
  return counts
}

// filterWorldMapTargetsInViewport 从全量缓存目标中裁剪当前视野可见城池。
export function filterWorldMapTargetsInViewport<T extends WorldMapPositionedTarget>(
  targets: T[],
  center: GridPosition,
  radius: number,
  relationFilters: Record<string, boolean>,
  worldSize?: number,
) {
  const bounds = typeof worldSize === 'number' ? buildWorldMapViewportBounds(center, radius, worldSize) : null
  return targets.filter((target) => {
    if (!isWorldMapRelationVisible(target.relation, relationFilters)) return false
    if (bounds) return isPositionInView(target.position, bounds.minX, bounds.minY, bounds.gridSize)
    return Math.abs(target.position.x - center.x) <= radius && Math.abs(target.position.y - center.y) <= radius
  })
}

// buildNearestWorldMapTargets 按距离挑选最近的可见玩家城池。
export function buildNearestWorldMapTargets<T extends WorldMapDistanceTarget>(targets: T[], limit: number) {
  const safeLimit = Math.max(0, Math.floor(limit))
  return [...targets]
    .filter((target) => target.relation !== 'self')
    .sort((a, b) => {
      if (a.distance !== b.distance) return a.distance - b.distance
      if (a.position.y !== b.position.y) return a.position.y - b.position.y
      return a.position.x - b.position.x
    })
    .slice(0, safeLimit)
}

// mergeWorldMapTargetCache 将刷新后的单个城池目标合并回全量缓存。
export function mergeWorldMapTargetCache<T extends WorldMapCacheTarget>(targets: T[], nextTarget: T) {
  let replaced = false
  const next = targets.map((target) => {
    if (target.playerId !== nextTarget.playerId) return target
    replaced = true
    return nextTarget
  })
  return replaced ? next : [nextTarget, ...targets]
}

// buildWorldMapMarchBadges 按当前玩家视角生成地图城池行军角标。
export function buildWorldMapMarchBadges<T extends WorldMapMarchLike>(marches: T[], activePlayerId: string | null | undefined) {
  const badges: Record<string, string> = {}
  if (!activePlayerId) return badges
  for (const march of marches) {
    if (march.status !== 'marching' && march.status !== 'returning' && march.status !== 'resolving') continue
    if (march.attackerPlayerId === activePlayerId) {
      badges[march.defenderPlayerId] = march.status === 'returning' ? '返' : march.status === 'resolving' ? '结' : '出'
    } else if (march.defenderPlayerId === activePlayerId) {
      badges[march.attackerPlayerId] = march.status === 'resolving' ? '结' : '袭'
    }
  }
  return badges
}

// buildWorldMapReinforcementMarches 将增援记录转换成地图行军角标可复用的轻量结构。
export function buildWorldMapReinforcementMarches<T extends WorldMapReinforcementLike>(reinforcements: T[]): WorldMapMarchLike[] {
  return reinforcements
    .filter((record) => record.status === 'marching' || record.status === 'returning')
    .map((record) => ({
      attackerPlayerId: record.fromPlayerId,
      defenderPlayerId: record.toPlayerId,
      status: record.status,
      arrivesAt: record.arriveAt,
      returnsAt: record.expectedReturnedAt,
    }))
}

// buildWorldMapMarchSummary 按当前玩家视角统计地图行军状态数量。
export function buildWorldMapMarchSummary<T extends WorldMapMarchLike>(marches: T[], activePlayerId: string | null | undefined): WorldMapMarchSummary {
  const summary: WorldMapMarchSummary = { outgoing: 0, returning: 0, incoming: 0, resolving: 0 }
  if (!activePlayerId) return summary
  for (const march of marches) {
    if (march.status === 'resolving') {
      summary.resolving += 1
    } else if (march.attackerPlayerId === activePlayerId && march.status === 'returning') {
      summary.returning += 1
    } else if (march.attackerPlayerId === activePlayerId && march.status === 'marching') {
      summary.outgoing += 1
    } else if (march.defenderPlayerId === activePlayerId && march.status === 'marching') {
      summary.incoming += 1
    }
  }
  return summary
}

// isSameGridPosition 判断两个格子坐标是否相同。
export function isSameGridPosition(a: GridPosition | null | undefined, b: GridPosition | null | undefined) {
  return Boolean(a && b && a.x === b.x && a.y === b.y)
}

// shouldShowCellCoordinate 判断当前格子密度是否适合直接显示坐标。
export function shouldShowCellCoordinate(gridSize: number) {
  return gridSize <= WORLD_MAP_MAX_COORDINATE_LABEL_GRID_SIZE
}

// factionBorderClass 返回地图城池的阵营边框色。
export function factionBorderClass(faction: string) {
  if (faction === 'wei') return 'border-blue-600 text-blue-700'
  if (faction === 'shu') return 'border-emerald-600 text-emerald-700'
  if (faction === 'wu') return 'border-red-600 text-red-700'
  return 'border-stone-600 text-stone-700'
}

// factionCityClass 返回地图城池主体的国家颜色。
export function factionCityClass(faction: string) {
  if (faction === 'wei') return 'border-blue-700 bg-blue-500 text-white'
  if (faction === 'shu') return 'border-emerald-700 bg-emerald-500 text-white'
  if (faction === 'wu') return 'border-red-700 bg-red-500 text-white'
  return 'border-stone-700 bg-stone-500 text-white'
}

// worldMapRelationRingClass 返回城池格子的关系高亮描边。
export function worldMapRelationRingClass(relation?: string) {
  if (relation === 'self') return 'ring-4 ring-sky-100 ring-offset-1 ring-offset-sky-900/60 shadow-[0_0_0_1px_rgba(255,255,255,0.75)]'
  if (relation === 'ally') return 'ring-2 ring-emerald-200 ring-offset-1 ring-offset-emerald-900/40'
  return 'ring-1 ring-white/30'
}

// worldMapOverviewTargetClass 返回概览图玩家城池点的国家颜色。
export function worldMapOverviewTargetClass(faction: string) {
  if (faction === 'wei') return 'bg-blue-500'
  if (faction === 'shu') return 'bg-emerald-500'
  if (faction === 'wu') return 'bg-red-500'
  return 'bg-stone-500'
}

// worldMapRelationBadge 返回城池关系短标识。
export function worldMapRelationBadge(relation?: string) {
  if (relation === 'self') return '己'
  if (relation === 'ally') return '盟'
  return '他'
}

// worldMapRelationLabel 返回城池关系的完整中文名称。
export function worldMapRelationLabel(relation?: string) {
  if (relation === 'self') return '自己'
  if (relation === 'ally') return '同盟'
  return '其他玩家'
}

// worldMapRelationBadgeClass 返回城池关系角标样式。
export function worldMapRelationBadgeClass(relation?: string) {
  if (relation === 'self') return 'bg-sky-600 text-white'
  if (relation === 'ally') return 'bg-emerald-600 text-white'
  return 'bg-stone-700 text-white'
}

// worldMapStatusBadge 返回城池状态短标识。
export function worldMapStatusBadge(status?: string) {
  if (status === 'protected') return '保'
  if (status === 'truce') return '免'
  if (status === 'unavailable') return '禁'
  return ''
}

// worldMapStatusBadgeClass 返回城池状态角标样式。
export function worldMapStatusBadgeClass(status?: string) {
  if (status === 'protected') return 'bg-amber-500 text-white'
  if (status === 'truce') return 'bg-fuchsia-600 text-white'
  if (status === 'unavailable') return 'bg-slate-700 text-white'
  return 'bg-amber-500 text-white'
}

// worldMapStatusPillClass 返回目标面板状态胶囊样式。
export function worldMapStatusPillClass(status?: string) {
  if (status === 'self') return 'bg-sky-500/10 text-sky-700'
  if (status === 'protected') return 'bg-amber-500/10 text-amber-700'
  if (status === 'truce') return 'bg-fuchsia-500/10 text-fuchsia-700'
  if (status === 'attackable') return 'bg-emerald-500/10 text-emerald-700'
  if (status === 'unavailable') return 'bg-slate-500/10 text-slate-700'
  return 'bg-[var(--color-accent-light)] text-[var(--color-accent)]'
}

// directionFrom 根据自身和目标坐标计算地图方向。
export function directionFrom(self: GridPosition, target: GridPosition) {
  const dx = target.x - self.x
  const dy = target.y - self.y
  if (dx === 0 && dy === 0) return '原地'
  const vertical = dy < 0 ? '北' : dy > 0 ? '南' : ''
  const horizontal = dx < 0 ? '西' : dx > 0 ? '东' : ''
  return `${horizontal}${vertical}` || '同轴'
}

// distanceFrom 按世界地图曼哈顿规则计算两格之间的距离。
export function distanceFrom(self: GridPosition, target: GridPosition) {
  return Math.abs(target.x - self.x) + Math.abs(target.y - self.y)
}

// buildWorldMapTargetMetrics 从真实坐标计算目标方位、距离和预计行军时间。
export function buildWorldMapTargetMetrics(self: GridPosition, target: GridPosition, unitSpeed = 1): WorldMapTargetMetrics {
  const distance = distanceFrom(self, target)
  return {
    direction: directionFrom(self, target),
    distance,
    seconds: estimateWorldMarchSeconds(distance, unitSpeed),
  }
}

// formatDuration 将秒数格式化为地图提示里的短时间。
export function formatDuration(seconds: number) {
  if (!Number.isFinite(seconds) || seconds <= 0) return '-'
  if (seconds < 60) return `${Math.ceil(seconds)}秒`
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  if (h > 0) return m > 0 ? `${h}时${m}分` : `${h}时`
  return `${Math.max(1, m)}分`
}

// formatWorldMapSyncTime 将服务端时间格式化为地图同步时间。
export function formatWorldMapSyncTime(value: string | null | undefined) {
  if (!value) return ''
  const time = new Date(value)
  if (Number.isNaN(time.getTime())) return ''
  return time.toLocaleTimeString('zh-CN', { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

// estimateWorldMarchSeconds 按一格五分钟和部队最低速度预估世界地图行军时间。
export function estimateWorldMarchSeconds(distance: number, unitSpeed: number, maxSeconds?: number) {
  if (!Number.isFinite(distance) || distance <= 0) return 0
  const safeDistance = Math.floor(distance)
  const safeSpeed = Math.max(1, Math.floor(unitSpeed || 1))
  const seconds = Math.ceil((safeDistance * WORLD_MAP_SECONDS_PER_GRID_AT_SPEED_ONE) / safeSpeed)
  const limit = typeof maxSeconds === 'number' ? maxSeconds : WORLD_MAP_MAX_MARCH_SECONDS
  return Math.min(limit, seconds)
}

// buildWorldMapActionStates 根据后端返回的权限字段生成地图目标操作状态。
export function buildWorldMapActionStates(target: WorldMapActionTarget, busyTarget: string | null, targetPlayerId: string): WorldMapActionState[] {
  const anyBusy = busyTarget !== null
  const busyPrefix = `${targetPlayerId}:`
  return [
    {
      key: 'scout',
      label: '侦查',
      disabled: anyBusy || target.canScout === false,
      reason: target.canScout === false ? (target.scoutReason || target.reason || '当前不可侦查') : '',
      busy: busyTarget === `${busyPrefix}scout`,
    },
    {
      key: 'attack',
      label: '攻击',
      disabled: anyBusy || target.canAttack === false,
      reason: target.canAttack === false ? (target.attackReason || target.reason || '当前不可攻击') : '',
      busy: busyTarget === `${busyPrefix}attack`,
    },
    {
      key: 'plunder',
      label: '掠夺',
      disabled: anyBusy || target.canPlunder === false,
      reason: target.canPlunder === false ? (target.plunderReason || target.reason || '当前不可掠夺') : '',
      busy: busyTarget === `${busyPrefix}plunder`,
    },
    {
      key: 'reinforce',
      label: '增援',
      disabled: anyBusy || target.canReinforce === false,
      reason: target.canReinforce === false ? (target.reinforceReason || target.reason || '当前不可增援') : '',
      busy: busyTarget === `${busyPrefix}reinforce`,
    },
  ]
}
