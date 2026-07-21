// 本文件验证世界地图网格纯计算规则。
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

import {
  buildNearestWorldMapTargets,
  buildVisibleCells,
  buildWorldMapActionStates,
  buildWorldMapAxisTicks,
  buildWorldMapDistanceGuides,
  buildWorldMapFactionCounts,
  buildWorldMapMarchBadges,
  buildWorldMapReinforcementMarches,
  buildWorldMapMarchSummary,
  buildWorldMapRelationCounts,
  buildWorldMapTargetMetrics,
  buildWorldMapViewportBounds,
  clampWorldMapCenter,
  clampWorldMapRadius,
  directionFrom,
  distanceFrom,
  estimateWorldMarchSeconds,
  factionBorderClass,
  factionCityClass,
  filterWorldMapTargetsInViewport,
  findWorldMapTargetAtCell,
  findVisibleWorldMapTargetAtCell,
  formatDuration,
  formatWorldMapSyncTime,
  isPositionInView,
  isSameGridPosition,
  isWorldMapRelationVisible,
  mergeWorldMapTargetCache,
  moveWorldMapCenter,
  parseWorldMapCoordinateSearch,
  shouldShowCellCoordinate,
  worldMapDragGridDelta,
  worldMapGrassCellClass,
  worldMapOverviewCoordinateFromPoint,
  worldMapOverviewPointStyle,
  worldMapOverviewTargetClass,
  worldMapOverviewViewportStyle,
  worldMapRelationBadge,
  worldMapRelationBadgeClass,
  worldMapRelationLabel,
  worldMapRelationRingClass,
  worldMapStatusBadge,
  worldMapStatusBadgeClass,
  worldMapStatusPillClass,
  WORLD_MAP_FACTION_LEGEND,
  WORLD_MAP_FULL_LOAD_RADIUS,
  WORLD_MAP_GRASS_VARIANT_COUNT,
  WORLD_MAP_MAX_MARCH_SECONDS,
  WORLD_MAP_MARCH_BADGE_LEGEND,
  WORLD_MAP_RELATION_BADGE_LEGEND,
  WORLD_MAP_STATUS_BADGE_LEGEND,
  WORLD_MAP_DISTANCE_GUIDE_STEP,
  WORLD_MAP_MAX_COORDINATE_LABEL_GRID_SIZE,
  WORLD_MAP_MAX_VIEW_RADIUS,
  WORLD_MAP_MIN_VIEW_RADIUS,
  WORLD_MAP_SECONDS_PER_GRID_AT_SPEED_ONE,
  WORLD_MAP_VIEW_PRESETS,
} from '../src/pages/map/worldMapGridLogic.ts'

test('世界地图中心视野按一格一坐标生成草地格', () => {
  const cells = buildVisibleCells(100, 48, 48, 5)
  assert.equal(cells.length, 25)
  assert.deepEqual(cells[0], { x: 48, y: 48 })
  assert.deepEqual(cells.at(-1), { x: 52, y: 52 })
})

test('世界地图草地纹理按坐标稳定变化', () => {
  assert.equal(WORLD_MAP_GRASS_VARIANT_COUNT, 5)
  assert.equal(worldMapGrassCellClass({ x: 12, y: 34 }), worldMapGrassCellClass({ x: 12, y: 34 }))
  const variants = new Set(buildVisibleCells(100, 10, 10, 5).map((cell) => worldMapGrassCellClass(cell)))
  assert.ok(variants.size > 1)
  assert.match(worldMapGrassCellClass({ x: 12, y: 34 }), /bg-\[#/)
})

test('世界地图边缘视野会裁剪越界坐标', () => {
  const cells = buildVisibleCells(100, -2, -2, 5)
  assert.equal(cells.length, 9)
  assert.deepEqual(cells[0], { x: 0, y: 0 })
  assert.deepEqual(cells.at(-1), { x: 2, y: 2 })
})

test('世界地图边缘视野会补满真实坐标格', () => {
  const bounds = buildWorldMapViewportBounds({ x: 0, y: 0 }, 10, 100)
  assert.deepEqual(bounds, {
    minX: 0,
    maxX: 20,
    minY: 0,
    maxY: 20,
    gridSize: 21,
    centerOffsetX: 0,
    centerOffsetY: 0,
  })
  const cells = buildVisibleCells(100, bounds.minX, bounds.minY, bounds.gridSize)
  assert.equal(cells.length, 441)
  assert.deepEqual(cells[0], { x: 0, y: 0 })
  assert.deepEqual(cells.at(-1), { x: 20, y: 20 })
  assert.deepEqual(buildWorldMapViewportBounds({ x: 99, y: 99 }, 10, 100), {
    minX: 79,
    maxX: 99,
    minY: 79,
    maxY: 99,
    gridSize: 21,
    centerOffsetX: 20,
    centerOffsetY: 20,
  })
})

test('世界地图坐标轴刻度与真实格子中心对齐', () => {
  const ticks = buildWorldMapAxisTicks(100, 40, 21, 7)
  assert.deepEqual(ticks.map((tick) => tick.value), [40, 45, 50, 55, 60])
  assert.equal(ticks[0].percent, `${((40 - 40 + 0.5) / 21) * 100}%`)
  assert.equal(ticks.at(-1)?.percent, `${((60 - 40 + 0.5) / 21) * 100}%`)
})

test('世界地图边缘坐标轴刻度会保留真实格子位置', () => {
  const ticks = buildWorldMapAxisTicks(100, -2, 5, 7)
  assert.deepEqual(ticks.map((tick) => tick.value), [0, 1, 2])
  assert.equal(ticks[0].percent, '50%')
  assert.equal(ticks.at(-1)?.percent, '90%')
})

test('世界地图距离辅助环按当前中心和真实格子生成', () => {
  assert.equal(WORLD_MAP_DISTANCE_GUIDE_STEP, 5)
  const guides = buildWorldMapDistanceGuides(15, 31)
  assert.deepEqual(guides.map((guide) => `${guide.distance}:${guide.side}`), ['5:11', '10:21', '15:31'])
  assert.deepEqual(guides[0], {
    distance: 5,
    side: 11,
    points: [
      `${((15 + 0.5) / 31) * 100},${((15 - 5 + 0.5) / 31) * 100}`,
      `${((15 + 5 + 0.5) / 31) * 100},${((15 + 0.5) / 31) * 100}`,
      `${((15 + 0.5) / 31) * 100},${((15 + 5 + 0.5) / 31) * 100}`,
      `${((15 - 5 + 0.5) / 31) * 100},${((15 + 0.5) / 31) * 100}`,
    ].join(' '),
    labelLeft: `${((15 + 0.5) / 31) * 100}%`,
    labelTop: `${((15 - 5 + 0.5) / 31) * 100}%`,
  })
  assert.deepEqual(buildWorldMapDistanceGuides(4, 9), [])
  const edgeGuides = buildWorldMapDistanceGuides(10, 21, 5, 0, 0)
  assert.equal(edgeGuides[0].points, [
    `${((0 + 0.5) / 21) * 100},${((0 - 5 + 0.5) / 21) * 100}`,
    `${((0 + 5 + 0.5) / 21) * 100},${((0 + 0.5) / 21) * 100}`,
    `${((0 + 0.5) / 21) * 100},${((0 + 5 + 0.5) / 21) * 100}`,
    `${((0 - 5 + 0.5) / 21) * 100},${((0 + 0.5) / 21) * 100}`,
  ].join(' '))
})

test('世界地图视野判断使用当前格子范围', () => {
  assert.equal(isPositionInView({ x: 50, y: 50 }, 48, 48, 5), true)
  assert.equal(isPositionInView({ x: 53, y: 50 }, 48, 48, 5), false)
  assert.equal(isPositionInView({ x: 48, y: 52 }, 48, 48, 5), true)
})

test('世界地图从全量缓存中按视野和关系筛选目标', () => {
  const targets = [
    { id: 'self', relation: 'self', position: { x: 50, y: 50 } },
    { id: 'ally', relation: 'ally', position: { x: 55, y: 50 } },
    { id: 'near', relation: 'other', position: { x: 60, y: 60 } },
    { id: 'far', relation: 'other', position: { x: 70, y: 50 } },
  ]
  const visible = filterWorldMapTargetsInViewport(targets, { x: 50, y: 50 }, 10, { self: true, ally: false, other: true })
  assert.deepEqual(visible.map((target) => target.id), ['self', 'near'])
  const edgeTargets = [
    { id: 'edge', relation: 'self', position: { x: 0, y: 0 } },
    { id: 'filled', relation: 'other', position: { x: 20, y: 20 } },
    { id: 'outside', relation: 'other', position: { x: 21, y: 20 } },
  ]
  const edgeVisible = filterWorldMapTargetsInViewport(edgeTargets, { x: 0, y: 0 }, 10, { self: true, ally: true, other: true }, 100)
  assert.deepEqual(edgeVisible.map((target) => target.id), ['edge', 'filled'])
  assert.equal(findWorldMapTargetAtCell(targets, { x: 55, y: 50 })?.id, 'ally')
  assert.equal(findWorldMapTargetAtCell(targets, { x: 51, y: 50 }), null)
  assert.equal(findVisibleWorldMapTargetAtCell(targets, { x: 55, y: 50 }, { ally: false })?.id, undefined)
  assert.equal(findVisibleWorldMapTargetAtCell(targets, { x: 55, y: 50 }, { ally: true })?.id, 'ally')
  assert.equal(isWorldMapRelationVisible('ally', { ally: false }), false)
  assert.equal(isWorldMapRelationVisible(undefined, { other: true }), true)
  assert.equal(isWorldMapRelationVisible(undefined, { other: false }), false)
  assert.deepEqual(buildWorldMapRelationCounts(targets), { self: 1, ally: 1, other: 2 })
  assert.deepEqual(buildNearestWorldMapTargets([
    { id: 'self', relation: 'self', distance: 0, position: { x: 50, y: 50 } },
    { id: 'far', relation: 'other', distance: 20, position: { x: 70, y: 50 } },
    { id: 'near_b', relation: 'other', distance: 5, position: { x: 50, y: 55 } },
    { id: 'near_a', relation: 'ally', distance: 5, position: { x: 49, y: 55 } },
  ], 2).map((target) => target.id), ['near_a', 'near_b'])
})

test('世界地图关系筛选控件固定提供三类玩家选项', () => {
  const filtersSource = readFileSync(new URL('../src/pages/map/components/WorldMapFilters.tsx', import.meta.url), 'utf8')
  assert.match(filtersSource, /counts: WorldMapRelationCounts/)
  assert.match(filtersSource, /toggle\('self'/)
  assert.match(filtersSource, /自己/)
  assert.match(filtersSource, /counts\.self/)
  assert.match(filtersSource, /toggle\('ally'/)
  assert.match(filtersSource, /同盟/)
  assert.match(filtersSource, /counts\.ally/)
  assert.match(filtersSource, /toggle\('other'/)
  assert.match(filtersSource, /其他玩家/)
  assert.match(filtersSource, /counts\.other/)
  assert.match(filtersSource, /const FilterCount/)
  const tabSource = readFileSync(new URL('../src/pages/map/components/WorldMapTab.tsx', import.meta.url), 'utf8')
  assert.match(tabSource, /relationFilters.*self: true, ally: true, other: true/)
  assert.match(tabSource, /const relationCounts = useMemo\(\(\) => buildWorldMapRelationCounts\(targets\), \[targets\]\)/)
  assert.match(tabSource, /const factionCounts = useMemo\(\(\) => buildWorldMapFactionCounts\(targets\), \[targets\]\)/)
  assert.match(tabSource, /const hiddenByFilterCount = targets\.length - overviewTargets\.length/)
  assert.match(tabSource, /const resetRelationFilters = useCallback\(\(\) =>/)
  assert.match(tabSource, /setRelationFilters\(\{ self: true, ally: true, other: true \}\)/)
  assert.match(tabSource, /<WorldMapFilters filters=\{relationFilters\} counts=\{relationCounts\} onChange=\{setRelationFilters\}/)
  assert.match(tabSource, /onClick=\{resetRelationFilters\}/)
  assert.match(tabSource, /<Eye size=\{10\} \/>/)
  assert.match(tabSource, /显示全部/)
  assert.match(tabSource, /relationFilters\.self === false && <span className="font-bold text-sky-600">我的城池已被筛选隐藏<\/span>/)
  assert.match(tabSource, /<WorldMapLegend counts=\{factionCounts\} \/>/)
  assert.match(tabSource, /showSelf=\{relationFilters\.self !== false\}/)
  assert.match(tabSource, /isWorldMapRelationVisible\(target\.relation, relationFilters\)/)
  assert.match(tabSource, /const selectMapPosition = useCallback/)
  assert.match(tabSource, /findVisibleWorldMapTargetAtCell\(targets, position, relationFilters\)/)
  assert.match(tabSource, /const hiddenTarget = target \? null : findWorldMapTargetAtCell\(targets, position\)/)
  assert.match(tabSource, /setRelationFilters\(\(prev\) => \(\{ \.\.\.prev, \[hiddenTarget\.relation \?\? 'other'\]: true \}\)\)/)
  assert.match(tabSource, /handleFocusTarget\(hiddenTarget\.playerId\)/)
  assert.match(tabSource, /该坐标有玩家城池，已自动显示并选中/)
  assert.match(tabSource, /该格有玩家城池，已自动显示并选中/)
  assert.match(tabSource, /目标坐标有玩家城池，已自动显示并选中/)
})

test('世界地图单目标详情刷新会合并回全量缓存', () => {
  const targets = [
    { playerId: 'a', status: 'normal', position: { x: 1, y: 1 } },
    { playerId: 'b', status: 'normal', position: { x: 2, y: 2 } },
  ]
  const replaced = mergeWorldMapTargetCache(targets, { playerId: 'b', status: 'protected', position: { x: 2, y: 2 } })
  assert.deepEqual(replaced.map((target) => `${target.playerId}:${target.status}`), ['a:normal', 'b:protected'])
  const inserted = mergeWorldMapTargetCache(targets, { playerId: 'c', status: 'normal', position: { x: 3, y: 3 } })
  assert.deepEqual(inserted.map((target) => target.playerId), ['c', 'a', 'b'])
})

test('世界地图中心移动和缩放会稳定裁剪到合法范围', () => {
  assert.deepEqual(clampWorldMapCenter({ x: -3, y: 120 }, 100), { x: 0, y: 99 })
  assert.deepEqual(clampWorldMapCenter({ x: 49.8, y: 50.2 }, 100), { x: 49, y: 50 })
  assert.deepEqual(moveWorldMapCenter({ x: 98, y: 1 }, 5, -5, 100), { x: 99, y: 0 })
  assert.equal(clampWorldMapRadius(1), WORLD_MAP_MIN_VIEW_RADIUS)
  assert.equal(clampWorldMapRadius(99), WORLD_MAP_MAX_VIEW_RADIUS)
  assert.equal(clampWorldMapRadius(12.8), 12)
  assert.deepEqual(WORLD_MAP_VIEW_PRESETS.map((preset) => `${preset.label}:${preset.radius}`), ['近景:5', '中景:10', '远景:30'])
})

test('世界地图坐标查找只接受世界范围内坐标', () => {
  assert.deepEqual(parseWorldMapCoordinateSearch({ x: '0', y: '99' }, 100), {
    position: { x: 0, y: 99 },
    error: '',
  })
  assert.deepEqual(parseWorldMapCoordinateSearch({ x: '100', y: '20' }, 100), {
    position: null,
    error: '请输入 0-99 范围内的坐标',
  })
  assert.deepEqual(parseWorldMapCoordinateSearch({ x: '-1', y: '20' }, 100), {
    position: null,
    error: '请输入 0-99 范围内的坐标',
  })
  assert.deepEqual(parseWorldMapCoordinateSearch({ x: '', y: '20' }, 100), {
    position: null,
    error: '请输入 0-99 范围内的坐标',
  })
  assert.deepEqual(parseWorldMapCoordinateSearch({ x: '1.5', y: '20' }, 100), {
    position: null,
    error: '请输入 0-99 范围内的坐标',
  })
  assert.deepEqual(parseWorldMapCoordinateSearch({ x: '12abc', y: '20' }, 100), {
    position: null,
    error: '请输入 0-99 范围内的坐标',
  })
})

test('世界地图拖拽按跨过的真实格子移动视野', () => {
  assert.deepEqual(worldMapDragGridDelta({
    startX: 100,
    startY: 100,
    currentX: 151,
    currentY: 51,
    width: 210,
    height: 210,
    gridSize: 21,
  }), { x: 5, y: -4 })
  assert.deepEqual(worldMapDragGridDelta({
    startX: 100,
    startY: 100,
    currentX: 107,
    currentY: 94,
    width: 210,
    height: 210,
    gridSize: 21,
  }), { x: 0, y: 0 })
})

test('世界地图主画布支持方向键按格移动', () => {
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapGrid.tsx', import.meta.url), 'utf8')
  assert.match(source, /type KeyboardEvent/)
  assert.match(source, /const handleMapKeyDown = \(event: KeyboardEvent<HTMLDivElement>\)/)
  assert.match(source, /event\.key === 'ArrowLeft'[\s\S]*onPan\(-step, 0\)/)
  assert.match(source, /event\.key === 'ArrowRight'[\s\S]*onPan\(step, 0\)/)
  assert.match(source, /event\.key === 'ArrowUp'[\s\S]*onPan\(0, -step\)/)
  assert.match(source, /event\.key === 'ArrowDown'[\s\S]*onPan\(0, step\)/)
  assert.match(source, /event\.key === '\+' \|\| event\.key === '='[\s\S]*if \(canZoomIn\) onZoom\(-5\)/)
  assert.match(source, /event\.key === '-' \|\| event\.key === '_'[\s\S]*if \(canZoomOut\) onZoom\(5\)/)
  assert.match(source, /event\.key === 'Home'[\s\S]*onFocusSelf\(\)/)
  assert.match(source, /event\.key === 'Escape'[\s\S]*onClearSelection\(\)/)
  assert.match(source, /event\.key === 'Enter' \|\| event\.key === ' '/)
  assert.match(source, /onSelectCell\(view\.centerX, view\.centerY\)/)
  assert.match(source, /tabIndex=\{0\}/)
  assert.match(source, /aria-label="世界地图，可用方向键按格移动，加号放大格子，减号缩小格子，Home 返回我的城，回车选择中心格，Escape 取消选择"/)
  assert.match(source, /onKeyDown=\{handleMapKeyDown\}/)
  assert.match(source, /onClearSelection: \(\) => void/)
})

test('世界地图概览点位和视野框按全图百分比计算', () => {
  assert.deepEqual(worldMapOverviewPointStyle({ x: 0, y: 99 }, 100), { left: '0.5%', top: '99.5%' })
  assert.deepEqual(worldMapOverviewPointStyle({ x: 120, y: -5 }, 100), { left: '99.5%', top: '0.5%' })
  assert.match(worldMapOverviewTargetClass('wei'), /blue/)
  assert.match(worldMapOverviewTargetClass('shu'), /emerald/)
  assert.match(worldMapOverviewTargetClass('wu'), /red/)
  assert.match(worldMapOverviewTargetClass('unknown'), /stone/)
  assert.deepEqual(worldMapOverviewViewportStyle({ x: 50, y: 50 }, 10, 100), {
    left: '40%',
    top: '40%',
    width: '21%',
    height: '21%',
  })
  assert.deepEqual(worldMapOverviewViewportStyle({ x: 0, y: 0 }, 10, 100), {
    left: '0%',
    top: '0%',
    width: '21%',
    height: '21%',
  })
  assert.deepEqual(worldMapOverviewCoordinateFromPoint({
    clientX: 150,
    clientY: 175,
    left: 100,
    top: 100,
    width: 100,
    height: 100,
    worldSize: 100,
  }), { x: 50, y: 75 })
  assert.deepEqual(worldMapOverviewCoordinateFromPoint({
    clientX: 260,
    clientY: 20,
    left: 100,
    top: 100,
    width: 100,
    height: 100,
    worldSize: 100,
  }), { x: 99, y: 0 })
})

test('世界地图格子选中和坐标标签密度稳定', () => {
  assert.equal(isSameGridPosition({ x: 12, y: 14 }, { x: 12, y: 14 }), true)
  assert.equal(isSameGridPosition({ x: 12, y: 14 }, { x: 12, y: 15 }), false)
  assert.equal(isSameGridPosition(null, { x: 12, y: 14 }), false)
  assert.equal(shouldShowCellCoordinate(WORLD_MAP_MAX_COORDINATE_LABEL_GRID_SIZE), true)
  assert.equal(shouldShowCellCoordinate(WORLD_MAP_MAX_COORDINATE_LABEL_GRID_SIZE + 1), false)
})

test('世界地图阵营颜色和方向文本稳定', () => {
  assert.match(factionBorderClass('wei'), /blue/)
  assert.match(factionBorderClass('shu'), /emerald/)
  assert.match(factionBorderClass('wu'), /red/)
  assert.match(factionBorderClass('unknown'), /stone/)
  assert.match(factionCityClass('wei'), /bg-blue/)
  assert.match(factionCityClass('shu'), /bg-emerald/)
  assert.match(factionCityClass('wu'), /bg-red/)
  assert.match(factionCityClass('unknown'), /bg-stone/)
  assert.equal(directionFrom({ x: 10, y: 10 }, { x: 13, y: 14 }), '东南')
  assert.equal(directionFrom({ x: 10, y: 10 }, { x: 8, y: 7 }), '西北')
  assert.equal(directionFrom({ x: 10, y: 10 }, { x: 10, y: 10 }), '原地')
  assert.equal(distanceFrom({ x: 10, y: 10 }, { x: 13, y: 14 }), 7)
  assert.equal(distanceFrom({ x: 0, y: 99 }, { x: 99, y: 0 }), 198)
  assert.deepEqual(buildWorldMapTargetMetrics({ x: 10, y: 10 }, { x: 13, y: 14 }, 5), {
    direction: '东南',
    distance: 7,
    seconds: 420,
  })
})

test('世界地图国家图例固定展示魏蜀吴和其它', () => {
  assert.deepEqual(WORLD_MAP_FACTION_LEGEND.map((item) => item.label), ['魏', '蜀', '吴', '其它'])
  assert.deepEqual(WORLD_MAP_FACTION_LEGEND.map((item) => item.faction), ['wei', 'shu', 'wu', 'other'])
  assert.match(WORLD_MAP_FACTION_LEGEND[0].colorClass, /blue/)
  assert.match(WORLD_MAP_FACTION_LEGEND[1].colorClass, /emerald/)
  assert.match(WORLD_MAP_FACTION_LEGEND[2].colorClass, /red/)
  assert.match(WORLD_MAP_FACTION_LEGEND[3].colorClass, /stone/)
  assert.deepEqual(buildWorldMapFactionCounts([
    { faction: 'wei' },
    { faction: 'shu' },
    { faction: 'wu' },
    { faction: 'other' },
    { faction: 'unknown' },
  ]), { wei: 1, shu: 1, wu: 1, other: 2 })
})

test('世界地图关系状态和行军角标图例固定展示', () => {
  assert.deepEqual(WORLD_MAP_RELATION_BADGE_LEGEND.map((item) => item.badge), ['己', '盟', '他'])
  assert.deepEqual(WORLD_MAP_RELATION_BADGE_LEGEND.map((item) => item.label), ['自己', '同盟', '其他'])
  assert.deepEqual(WORLD_MAP_STATUS_BADGE_LEGEND.map((item) => item.badge), ['保', '免', '禁'])
  assert.deepEqual(WORLD_MAP_STATUS_BADGE_LEGEND.map((item) => item.label), ['保护', '免战', '不可操作'])
  assert.deepEqual(WORLD_MAP_MARCH_BADGE_LEGEND.map((item) => item.badge), ['出', '侦', '返', '袭', '探', '结'])
  assert.deepEqual(WORLD_MAP_MARCH_BADGE_LEGEND.map((item) => item.label), ['出征', '侦查', '返程', '被袭', '被侦查', '结算'])
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapLegend.tsx', import.meta.url), 'utf8')
  assert.match(source, /WORLD_MAP_RELATION_BADGE_LEGEND/)
  assert.match(source, /WORLD_MAP_STATUS_BADGE_LEGEND/)
  assert.match(source, /WORLD_MAP_MARCH_BADGE_LEGEND/)
  assert.match(source, /worldMapRelationBadgeClass\(item\.relation\)/)
  assert.match(source, /worldMapStatusBadgeClass\(item\.status\)/)
  assert.doesNotMatch(source, /rounded-full bg-amber-500/)
  assert.match(source, /counts: WorldMapFactionCounts/)
  assert.match(source, /counts\[item\.faction as keyof WorldMapFactionCounts\]/)
  assert.match(source, /const LegendCount/)
  assert.match(source, /bg-violet-600/)
})

test('世界地图城池关系和状态角标稳定', () => {
  assert.equal(worldMapRelationBadge('self'), '己')
  assert.equal(worldMapRelationBadge('ally'), '盟')
  assert.equal(worldMapRelationBadge('other'), '他')
  assert.equal(worldMapRelationLabel('self'), '自己')
  assert.equal(worldMapRelationLabel('ally'), '同盟')
  assert.equal(worldMapRelationLabel('other'), '其他玩家')
  assert.match(worldMapRelationBadgeClass('self'), /sky/)
  assert.match(worldMapRelationBadgeClass('ally'), /emerald/)
  assert.match(worldMapRelationBadgeClass('other'), /stone/)
  assert.match(worldMapRelationRingClass('self'), /sky/)
  assert.match(worldMapRelationRingClass('self'), /ring-4/)
  assert.match(worldMapRelationRingClass('self'), /shadow/)
  assert.match(worldMapRelationRingClass('ally'), /emerald/)
  assert.match(worldMapRelationRingClass('other'), /white/)
  assert.equal(worldMapStatusBadge('protected'), '保')
  assert.equal(worldMapStatusBadge('truce'), '免')
  assert.equal(worldMapStatusBadge('unavailable'), '禁')
  assert.equal(worldMapStatusBadge('normal'), '')
  assert.match(worldMapStatusBadgeClass('protected'), /amber/)
  assert.match(worldMapStatusBadgeClass('truce'), /fuchsia/)
  assert.match(worldMapStatusBadgeClass('unavailable'), /slate/)
  assert.match(worldMapStatusBadgeClass('normal'), /amber/)
  assert.match(worldMapStatusPillClass('self'), /sky/)
  assert.match(worldMapStatusPillClass('protected'), /amber/)
  assert.match(worldMapStatusPillClass('truce'), /fuchsia/)
  assert.match(worldMapStatusPillClass('attackable'), /emerald/)
  assert.match(worldMapStatusPillClass('unavailable'), /slate/)
  assert.match(worldMapStatusPillClass('normal'), /color-accent/)
  const gridSource = readFileSync(new URL('../src/pages/map/components/WorldMapGrid.tsx', import.meta.url), 'utf8')
  assert.match(gridSource, /worldMapStatusBadgeClass\(target\.status\)/)
  assert.doesNotMatch(gridSource, /rounded-tl-\[1px\] bg-amber-500/)
  const panelSource = readFileSync(new URL('../src/pages/map/components/WorldMapTargetPanel.tsx', import.meta.url), 'utf8')
  assert.match(panelSource, /worldMapStatusPillClass\(target\.status\)/)
  assert.doesNotMatch(panelSource, /bg-\[var\(--color-accent-light\)\] px-1\.5 py-0\.5 text-\[10px\] font-bold text-\[var\(--color-accent\)\]/)
})

test('世界地图行军提示时间格式稳定', () => {
  assert.equal(formatDuration(0), '-')
  assert.equal(formatDuration(20), '20秒')
  assert.equal(formatDuration(420), '7分')
  assert.equal(formatDuration(3600), '1时')
  assert.equal(formatDuration(3900), '1时5分')
  assert.equal(formatWorldMapSyncTime(''), '')
  assert.equal(formatWorldMapSyncTime('bad-time'), '')
  assert.match(formatWorldMapSyncTime('2026-07-02T12:34:56Z'), /^\d{2}:\d{2}:\d{2}$/)
})

test('世界地图一格按距离 1 和速度 1 五分钟计算行军时间', () => {
  assert.equal(WORLD_MAP_SECONDS_PER_GRID_AT_SPEED_ONE, 300)
  assert.equal(WORLD_MAP_MAX_MARCH_SECONDS, 3 * 3600)
  assert.equal(estimateWorldMarchSeconds(0, 1), 0)
  assert.equal(estimateWorldMarchSeconds(7, 5), 420)
  assert.equal(estimateWorldMarchSeconds(1, 15), 20)
  assert.equal(estimateWorldMarchSeconds(100, 1), 10800)
  assert.equal(estimateWorldMarchSeconds(100, 5), 6000)
})

test('世界地图目标操作按钮以后端权限字段为准', () => {
  const actions = buildWorldMapActionStates({
    canScout: false,
    canReinforce: false,
    canAttack: false,
    canPlunder: false,
    reason: '自己的城池',
    attackReason: '目标处于免战保护',
    plunderReason: '目标处于免战保护',
    reinforceReason: '目标增援来源已满',
  }, null, 'player_self')
  assert.deepEqual(actions.map((action) => action.key), ['scout', 'attack', 'plunder', 'reinforce'])
  assert.deepEqual(actions.map((action) => action.disabled), [true, true, true, true])
  assert.deepEqual(actions.map((action) => action.reason), ['自己的城池', '目标处于免战保护', '目标处于免战保护', '目标增援来源已满'])
})

test('世界地图目标操作忙碌时只标记当前动作但禁用全部按钮', () => {
  const actions = buildWorldMapActionStates({
    canScout: true,
    canReinforce: true,
    canAttack: true,
    canPlunder: true,
  }, 'player_target:attack', 'player_target')
  assert.deepEqual(actions.map((action) => action.disabled), [true, true, true, true])
  assert.deepEqual(actions.map((action) => action.busy), [false, true, false, false])
})

test('世界地图行军角标按当前玩家视角生成', () => {
  const marches = [
    { attackerPlayerId: 'me', defenderPlayerId: 'targetA', status: 'marching' },
    { attackerPlayerId: 'me', defenderPlayerId: 'targetB', status: 'returning' },
    { attackerPlayerId: 'targetC', defenderPlayerId: 'me', status: 'marching' },
    { attackerPlayerId: 'me', defenderPlayerId: 'targetD', status: 'resolving' },
    { attackerPlayerId: 'me', defenderPlayerId: 'targetE', status: 'resolved' },
    { attackerPlayerId: 'me', defenderPlayerId: 'targetF', marchType: 'scout', status: 'marching' },
    { attackerPlayerId: 'targetG', defenderPlayerId: 'me', marchType: 'scout', status: 'marching' },
  ]
  const badges = buildWorldMapMarchBadges(marches, 'me')
  assert.equal(badges.targetA, '出')
  assert.equal(badges.targetB, '返')
  assert.equal(badges.targetC, '袭')
  assert.equal(badges.targetD, '结')
  assert.equal(badges.targetE, undefined)
  assert.equal(badges.targetF, '侦')
  assert.equal(badges.targetG, '探')
  assert.deepEqual(buildWorldMapMarchSummary(marches, 'me'), {
    outgoing: 2,
    returning: 1,
    incoming: 2,
    resolving: 1,
  })
})

test('世界地图把增援行军合并进地图角标口径', () => {
  const reinforcements = buildWorldMapReinforcementMarches([
    { fromPlayerId: 'me', toPlayerId: 'allyA', status: 'marching', arriveAt: '2026-07-02T10:05:00Z' },
    { fromPlayerId: 'me', toPlayerId: 'allyB', status: 'returning', expectedReturnedAt: '2026-07-02T10:10:00Z' },
    { fromPlayerId: 'allyC', toPlayerId: 'me', status: 'marching', arriveAt: '2026-07-02T10:15:00Z' },
    { fromPlayerId: 'allyD', toPlayerId: 'me', status: 'stationed' },
  ])
  assert.deepEqual(reinforcements, [
    { attackerPlayerId: 'me', defenderPlayerId: 'allyA', status: 'marching', arrivesAt: '2026-07-02T10:05:00Z', returnsAt: undefined },
    { attackerPlayerId: 'me', defenderPlayerId: 'allyB', status: 'returning', arrivesAt: undefined, returnsAt: '2026-07-02T10:10:00Z' },
    { attackerPlayerId: 'allyC', defenderPlayerId: 'me', status: 'marching', arrivesAt: '2026-07-02T10:15:00Z', returnsAt: undefined },
  ])
  assert.deepEqual(buildWorldMapMarchBadges(reinforcements, 'me'), {
    allyA: '出',
    allyB: '返',
    allyC: '袭',
  })
  assert.deepEqual(buildWorldMapMarchSummary(reinforcements, 'me'), {
    outgoing: 1,
    returning: 1,
    incoming: 1,
    resolving: 0,
  })
})

test('世界地图首屏使用近景半径并懒加载全量目标缓存', () => {
  assert.equal(WORLD_MAP_FULL_LOAD_RADIUS, 100)
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapTab.tsx', import.meta.url), 'utf8')
  assert.match(source, /const DEFAULT_VIEW_RADIUS = WORLD_MAP_MIN_VIEW_RADIUS/)
  assert.match(source, /gameApi\.getWorldMapView\(activePlayerId, \{ radius: DEFAULT_VIEW_RADIUS \}\)/)
  assert.match(source, /scheduleGlobalWorldMapTargets\(activePlayerId, requestId\)/)
  assert.match(source, /window\.setTimeout\(\(\) => \{[\s\S]*void loadGlobalWorldMapTargets\(playerId, parentRequestId\)[\s\S]*\}, 1200\)/)
  assert.match(source, /const loadGlobalWorldMapTargets = useCallback\(async \(playerId: string, parentRequestId: number\) =>/)
  assert.match(source, /gameApi\.getWorldMapView\(playerId, \{ radius: WORLD_MAP_FULL_LOAD_RADIUS \}\)/)
  assert.match(source, /void loadAuxiliaryWorldMapData\(activePlayerId, requestId\)/)
  assert.match(source, /const loadAuxiliaryWorldMapData = useCallback\(async \(playerId: string, parentRequestId: number\) =>/)
  assert.match(source, /const targetResult = await gameApi\.getWorldMapView\(activePlayerId, \{ radius: DEFAULT_VIEW_RADIUS \}\)/)
  assert.doesNotMatch(source, /const \[targetResult,[\s\S]*await Promise\.all/)
  assert.match(source, /parentRequestId !== loadRequestRef\.current/)
  assert.match(source, /loadedPlayerRef\.current !== playerId/)
  assert.doesNotMatch(source, /gameApi\.getWorldMapView\(activePlayerId, \{ radius: WORLD_MAP_FULL_LOAD_RADIUS \}\)/)
  assert.doesNotMatch(source, /gameApi\.getWorldMapView\(activePlayerId, \{ radius: viewport\.radius/)
})

test('PVP 兼容目标接口允许查询 0 坐标中心', () => {
  const source = readFileSync(new URL('../src/api/game.ts', import.meta.url), 'utf8')
  assert.match(source, /params\?\.centerX !== undefined/)
  assert.match(source, /params\?\.centerY !== undefined/)
})

test('世界地图目标详情接口编码路径和查询参数', () => {
  const source = readFileSync(new URL('../src/api/game.ts', import.meta.url), 'utf8')
  assert.match(source, /const query = new URLSearchParams\(\{ viewerId \}\)/)
  assert.match(source, /encodeURIComponent\(playerId\)/)
  assert.doesNotMatch(source, /player_city\/\$\{playerId\}\?viewerId=\$\{viewerId\}/)
})

test('GM 世界坐标接口编码玩家路径参数', () => {
  const source = readFileSync(new URL('../../admin/src/api/admin.ts', import.meta.url), 'utf8')
  assert.match(source, /getWorldPosition\(playerId: string\) \{[\s\S]*positions\/\$\{encodeURIComponent\(playerId\)\}/)
  assert.match(source, /updateWorldPosition\(playerId: string, x: number, y: number\) \{[\s\S]*positions\/\$\{encodeURIComponent\(playerId\)\}/)
  assert.doesNotMatch(source, /world-map\/positions\/\$\{playerId\}/)
})

test('GM 世界坐标面板保存前检查目标格占用', () => {
  const apiSource = readFileSync(new URL('../../admin/src/api/admin.ts', import.meta.url), 'utf8')
  const panelSource = readFileSync(new URL('../../admin/src/components/WorldPositionPanel.tsx', import.meta.url), 'utf8')
  const typeSource = readFileSync(new URL('../../admin/src/types/index.ts', import.meta.url), 'utf8')
  assert.match(typeSource, /export interface WorldMapCoordinateCheck/)
  assert.match(apiSource, /checkWorldCoordinate\(x: number, y: number\)/)
  assert.match(apiSource, /new URLSearchParams\(\{ x: String\(x\), y: String\(y\) \}\)/)
  assert.match(apiSource, /world-map\/positions\/check\?\$\{query\.toString\(\)\}/)
  assert.match(panelSource, /const \[coordinateCheck, setCoordinateCheck\] = useState<WorldMapCoordinateCheck \| null>\(null\)/)
  assert.match(panelSource, /const handleCheckCoordinate = async \(\) =>/)
  assert.match(panelSource, /adminApi\.checkWorldCoordinate\(coordinate\.x, coordinate\.y\)/)
  assert.match(panelSource, /setError\(`坐标已被玩家 \$\{result\.playerId\} 占用`\)/)
  assert.match(panelSource, /const check = await handleCheckCoordinate\(\)/)
  assert.match(panelSource, /if \(check\?\.occupied && check\.playerId !== playerId\) \{[\s\S]*return[\s\S]*\}/)
  assert.match(panelSource, /disabled=\{saving \|\| checking \|\| \(coordinateCheck\?\.occupied === true && coordinateCheck\.playerId !== playerId\)\}/)
  assert.match(panelSource, /检查格 \(\{coordinateCheck\.x\}, \{coordinateCheck\.y\}\)：/)
})

test('世界地图页使用统一坐标查找解析规则', () => {
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapTab.tsx', import.meta.url), 'utf8')
  assert.match(source, /parseWorldMapCoordinateSearch\(coordinateSearch/)
  assert.match(source, /selectMapPosition\(center, '该坐标有玩家城池，已自动显示并选中'\)/)
  assert.match(source, /else if \(hiddenTarget\) \{[\s\S]*setRelationFilters\(\(prev\) => \(\{ \.\.\.prev, \[hiddenTarget\.relation \?\? 'other'\]: true \}\)\)[\s\S]*handleFocusTarget\(hiddenTarget\.playerId\)[\s\S]*toast\.info\(hiddenMessage\)/)
  assert.match(source, /setSelectedEmptyCell\(position\)/)
  assert.match(source, /const target = targets\.find\(\(item\) => item\.playerId === playerId\)/)
  assert.match(source, /if \(target\) setCoordinateSearch\(\{ x: String\(target\.position\.x\), y: String\(target\.position\.y\) \}\)/)
  assert.match(source, /handleFocusTarget\(target\.playerId\)/)
  assert.match(source, /setCoordinateSearch\(\{ x: String\(center\.x\), y: String\(center\.y\) \}\)/)
  assert.match(source, /setCoordinateSearch\(\{ x: String\(x\), y: String\(y\) \}\)/)
  assert.match(source, /onFocusSelf=\{focusSelf\}/)
  assert.match(source, /worldSize=\{targetView\?\.worldSize \?\? 100\}/)
})

test('世界地图加载失败时提供重试入口', () => {
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapTab.tsx', import.meta.url), 'utf8')
  assert.match(source, /if \(!activePlayerId\) \{[\s\S]*hasLoadedMapRef\.current = false[\s\S]*loadedPlayerRef\.current = null[\s\S]*setTargets\(\[\]\)[\s\S]*setTargetView\(null\)[\s\S]*setMarches\(\[\]\)[\s\S]*setMapServerTime\(''\)[\s\S]*setLoadError\(''\)[\s\S]*setRefreshing\(false\)[\s\S]*setLoading\(false\)[\s\S]*return/)
  assert.match(source, /const \[refreshing, setRefreshing\] = useState\(false\)/)
  assert.match(source, /const hasLoadedMapRef = useRef\(false\)/)
  assert.match(source, /if \(hasLoadedMapRef\.current\) \{[\s\S]*setRefreshing\(true\)/)
  assert.match(source, /hasLoadedMapRef\.current = true/)
  assert.match(source, /setRefreshing\(false\)/)
  assert.match(source, /const \[loadError, setLoadError\] = useState\(''\)/)
  assert.match(source, /if \(!silent\) setLoadError\(''\)/)
  assert.match(source, /catch \(err\)/)
  assert.match(source, /setLoadError\(message\)/)
  assert.match(source, /toast\.error\(message\)/)
  assert.match(source, /if \(loadError && !targetView\)/)
  assert.match(source, /世界地图加载失败/)
  assert.match(source, /onClick=\{\(\) => void load\(\)\}/)
  assert.match(source, /重新加载/)
})

test('世界地图已有内容时刷新不清空地图', () => {
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapTab.tsx', import.meta.url), 'utf8')
  assert.match(source, /disabled=\{refreshing\}/)
  assert.match(source, /className=\{refreshing \? 'animate-spin' : ''\}/)
  assert.match(source, /\{refreshing \? '刷新中' : '刷新'\}/)
  assert.match(source, /if \(hasLoadedMapRef\.current\) \{[\s\S]*setRefreshing\(true\)[\s\S]*\} else \{[\s\S]*setLoading\(true\)/)
})

test('世界地图切换玩家存档时清空旧地图重新加载', () => {
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapTab.tsx', import.meta.url), 'utf8')
  assert.match(source, /const loadedPlayerRef = useRef<string \| null>\(null\)/)
  assert.match(source, /const switchingPlayer = loadedPlayerRef\.current !== activePlayerId/)
  assert.match(source, /if \(switchingPlayer\) \{[\s\S]*hasLoadedMapRef\.current = false/)
  assert.match(source, /setTargets\(\[\]\)/)
  assert.match(source, /setTargetView\(null\)/)
  assert.match(source, /setMarches\(\[\]\)/)
  assert.match(source, /setMapServerTime\(''\)/)
  assert.match(source, /setLoading\(true\)/)
  assert.match(source, /const shouldSyncInitialCoordinate = switchingPlayer \|\| !hasLoadedMapRef\.current/)
  assert.match(source, /if \(shouldSyncInitialCoordinate\) setCoordinateSearch\(\{ x: String\(selfPosition\.x\), y: String\(selfPosition\.y\) \}\)/)
  assert.match(source, /loadedPlayerRef\.current = activePlayerId/)
  assert.match(source, /useEffect\(\(\) => \{[\s\S]*setCoordinateSearch\(\{ x: '', y: '' \}\)[\s\S]*setSelectedMarchTarget\(null\)[\s\S]*setSelectedReinforceTarget\(null\)[\s\S]*setSelections\(\{\}\)[\s\S]*setSelectedGeneralIds\(\[\]\)[\s\S]*setBusyTarget\(null\)[\s\S]*\}, \[activePlayerId\]\)/)
})

test('世界地图没有玩家存档时显示空状态', () => {
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapTab.tsx', import.meta.url), 'utf8')
  assert.match(source, /if \(!activePlayerId\)/)
  assert.match(source, /请选择玩家存档/)
  assert.match(source, /选择存档后即可查看自己的世界地图坐标和附近玩家城池/)
})

test('世界地图顶部展示行军状态摘要', () => {
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapTab.tsx', import.meta.url), 'utf8')
  assert.match(source, /const \[reinforcements, setReinforcements\] = useState<Reinforcement\[\]>\(\[\]\)/)
  assert.match(source, /gameApi\.listSentReinforcements\(playerId\)/)
  assert.match(source, /gameApi\.listReceivedReinforcements\(playerId\)/)
  assert.match(source, /setReinforcements\(mergeWorldMapReinforcements\(sentReinforcementResult\.items \?\? \[\], receivedReinforcementResult\.items \?\? \[\]\)\)/)
  assert.match(source, /const playerCityMarches = marches\.filter/)
  assert.match(source, /const reinforcementMarches = buildWorldMapReinforcementMarches\(reinforcements\)/)
  assert.match(source, /return \[\.\.\.playerCityMarches, \.\.\.reinforcementMarches\]/)
  assert.doesNotMatch(source, /const pvpMarches = marches\.filter/)
  assert.match(source, /function isMarchDue\(march: \{ status: string; returnsAt\?: string; arrivesAt\?: string \}, nowMs: number\)/)
  assert.match(source, /const marchSummary = useMemo\(\(\) => buildWorldMapMarchSummary\(activeMarches, activePlayerId\), \[activeMarches, activePlayerId\]\)/)
  assert.match(source, /activeMarches\.length > 0/)
  assert.match(source, /行军/)
  assert.match(source, /出 \{marchSummary\.outgoing\}/)
  assert.match(source, /返 \{marchSummary\.returning\}/)
  assert.match(source, /袭 \{marchSummary\.incoming\}/)
  assert.match(source, /结 \{marchSummary\.resolving\}/)
  assert.match(source, /function mergeWorldMapReinforcements\(\.\.\.groups: Reinforcement\[\]\[\]\)/)
  assert.match(source, /seen\.has\(record\.reinforcementId\)/)
})

test('世界地图顶部展示服务端同步时间', () => {
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapTab.tsx', import.meta.url), 'utf8')
  assert.match(source, /const \[mapServerTime, setMapServerTime\] = useState\(''\)/)
  assert.match(source, /setMapServerTime\(targetResult\.serverTime\)/)
  assert.match(source, /const syncTimeLabel = useMemo\(\(\) => formatWorldMapSyncTime\(mapServerTime\), \[mapServerTime\]\)/)
  assert.match(source, /已同步 \{syncTimeLabel\}/)
  assert.match(source, /title="刷新世界地图玩家城池、行军和增援状态"/)
  assert.match(source, /aria-label="刷新世界地图玩家城池、行军和增援状态"/)
  assert.match(source, /\{refreshing \? '刷新中' : '刷新'\}/)
})

test('世界地图顶部展示世界格子占用量', () => {
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapTab.tsx', import.meta.url), 'utf8')
  const typeSource = readFileSync(new URL('../src/types/game.ts', import.meta.url), 'utf8')
  assert.match(typeSource, /worldWidth\?: number/)
  assert.match(typeSource, /worldHeight\?: number/)
  assert.match(source, /worldSize: Math\.min\(targetResult\.width, targetResult\.height\)/)
  assert.match(source, /worldWidth: targetResult\.width/)
  assert.match(source, /worldHeight: targetResult\.height/)
  assert.match(source, /const worldWidth = targetView\?\.worldWidth \?\? targetView\?\.worldSize \?\? 0/)
  assert.match(source, /const worldHeight = targetView\?\.worldHeight \?\? targetView\?\.worldSize \?\? 0/)
  assert.match(source, /const worldCellCapacity = worldWidth > 0 && worldHeight > 0 \? worldWidth \* worldHeight : 0/)
  assert.match(source, /const worldOccupiedCells = useMemo/)
  assert.match(source, /const hasSelfTarget = targets\.some/)
  assert.match(source, /target\.relation === 'self'/)
  assert.match(source, /target\.position\.x === targetView\.self\.x && target\.position\.y === targetView\.self\.y/)
  assert.match(source, /hasSelfTarget \? targets\.length : targets\.length \+ 1/)
  assert.match(source, /const worldOccupancyRate = worldCellCapacity > 0 \? worldOccupiedCells \/ worldCellCapacity : 0/)
  assert.match(source, /世界 \{worldWidth\} x \{worldHeight\}/)
  assert.match(source, /当前视野 \{visibleTargets\.length\}\/\{viewportTargets\.length\} 个玩家城池/)
  assert.match(source, /占用 \{worldOccupiedCells\}\/\{worldCellCapacity\} 格/)
  assert.match(source, /worldOccupancyRate >= 0\.8/)
  assert.match(source, /接近扩容线/)
  assert.match(source, /hiddenByFilterCount > 0/)
  assert.match(source, /全图已隐藏 \{hiddenByFilterCount\} 个/)
  assert.doesNotMatch(source, /当前视野 \{visibleTargets\.length\}\/\{targets\.length\} 个玩家城池/)
})

test('世界地图移动端目标面板固定为底部抽屉', () => {
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapTab.tsx', import.meta.url), 'utf8')
  assert.match(source, /space-y-4 pb-\[52vh\] xl:pb-0/)
  assert.match(source, /fixed inset-x-0 bottom-0 z-\[8000\]/)
  assert.match(source, /max-h-\[48vh\] overflow-y-auto/)
  assert.match(source, /xl:static xl:z-auto xl:max-h-none xl:overflow-visible/)
  assert.match(source, /<WorldMapTargetPanel/)
  assert.match(source, /ml-0 flex w-full flex-wrap items-center gap-3 lg:ml-auto lg:w-auto/)
})

test('世界地图坐标查找控件提供返回本城入口', () => {
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapCoordinateSearch.tsx', import.meta.url), 'utf8')
  const tabSource = readFileSync(new URL('../src/pages/map/components/WorldMapTab.tsx', import.meta.url), 'utf8')
  assert.match(source, /placeholder="X"/)
  assert.match(source, /placeholder="Y"/)
  assert.match(source, /worldSize: number/)
  assert.match(source, /const maxCoordinate = Math\.max\(0, Math\.floor\(worldSize\) - 1\)/)
  assert.match(source, /const maxCoordinateLength = String\(maxCoordinate\)\.length/)
  assert.match(source, /const coordinateSummary = `\(\$\{value\.x \|\| '\?'\}, \$\{value\.y \|\| '\?'\}\)`/)
  assert.match(source, /type="text"/)
  assert.match(source, /pattern="\[0-9\]\*"/)
  assert.match(source, /inputMode="numeric"/)
  assert.match(source, /maxLength=\{maxCoordinateLength\}/)
  assert.match(source, /const handleCoordinateChange = \(axis: 'x' \| 'y', nextValue: string\) =>/)
  assert.match(source, /nextValue !== '' && !\/\^\\d\+\$\/\.test\(nextValue\)/)
  assert.match(source, /onChange\(\{ \.\.\.value, \[axis\]: nextValue \}\)/)
  assert.match(source, /onKeyDown=\{handleKeyDown\}/)
  assert.match(source, /event\.key !== 'Enter'/)
  assert.match(source, /onSearch\(\)/)
  assert.match(source, /0-\{maxCoordinate\}/)
  assert.match(source, /onClick=\{onSearch\}/)
  assert.match(source, /title=\{`查找坐标 \$\{coordinateSummary\}`\}/)
  assert.match(source, /aria-label=\{`查找世界地图坐标 \$\{coordinateSummary\}`\}/)
  assert.match(source, /onClick=\{onFocusSelf\}/)
  assert.match(source, /我的城/)
  assert.match(tabSource, /const selfTarget = findWorldMapTargetAtCell\(targets, targetView\.self\)/)
  assert.match(tabSource, /setRelationFilters\(\(prev\) => \(\{ \.\.\.prev, self: true \}\)\)/)
  assert.match(tabSource, /setCoordinateSearch\(\{ x: String\(targetView\.self\.x\), y: String\(targetView\.self\.y\) \}\)/)
  assert.match(tabSource, /setFocusedTargetId\(selfTarget\?\.playerId \?\? null\)/)
  assert.match(tabSource, /setSelectedEmptyCell\(null\)[\s\S]*setViewport\(\(prev\) => \(\{ \.\.\.prev, centerX: targetView\.self\.x, centerY: targetView\.self\.y \}\)\)/)
})

test('世界地图主画布保持正方形格子', () => {
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapGrid.tsx', import.meta.url), 'utf8')
  assert.match(source, /aspect-square/)
  assert.match(source, /h-\[min\(76vw,520px\)\] min-h-\[260px\]/)
  assert.match(source, /sm:min-h-\[320px\] sm:p-5/)
  assert.match(source, /flex flex-wrap items-center gap-1/)
  assert.match(source, /gridTemplateColumns: `repeat\(\$\{gridSize\}, minmax\(0, 1fr\)\)`/)
  assert.match(source, /gridTemplateRows: `repeat\(\$\{gridSize\}, minmax\(0, 1fr\)\)`/)
  assert.match(source, /buildWorldMapAxisTicks/)
  assert.match(source, /buildWorldMapDistanceGuides/)
  assert.match(source, /worldMapDragGridDelta/)
  assert.match(source, /import tileCityShu from '@\/assets\/map\/tiles\/tile-city-shu\.png'/)
  assert.match(source, /import tileCityWei from '@\/assets\/map\/tiles\/tile-city-wei\.png'/)
  assert.match(source, /import tileCityWu from '@\/assets\/map\/tiles\/tile-city-wu\.png'/)
  assert.match(source, /import tileGrass from '@\/assets\/map\/tiles\/tile-grass\.png'/)
  assert.match(source, /cityTileSrc\(target\.faction\)/)
  assert.match(source, /backgroundImage: `url\(\$\{tileGrass\}\)`/)
  assert.match(source, /worldMapRelationRingClass\(target\.relation\)/)
  assert.match(source, /worldMapRelationRingClass\('self'\)/)
  assert.doesNotMatch(source, /bg-\[#78a85a\] bg-\[linear-gradient\(135deg/)
})

test('世界地图概览同步展示筛选后的全量玩家城池点', () => {
  const gridSource = readFileSync(new URL('../src/pages/map/components/WorldMapGrid.tsx', import.meta.url), 'utf8')
  assert.match(gridSource, /overviewTargets: PvpTargetSummary\[\]/)
  assert.match(gridSource, /onJump: \(position: GridPosition\) => void/)
  assert.match(gridSource, /<MapOverview view=\{view\} radius=\{radius\} selectedPosition=\{selectedCell \?\? focusedTargetPosition\} targets=\{overviewTargets\} showSelf=\{showSelf\} onJump=\{onJump\} \/>/)
  assert.match(gridSource, /showSelf: boolean; onJump/)
  assert.match(gridSource, /showSelf && \(/)
  assert.match(gridSource, /targets\.map\(\(target\) =>/)
  assert.match(gridSource, /worldMapOverviewTargetClass\(target\.faction\)/)
  assert.match(gridSource, /worldMapOverviewPointStyle\(target\.position, view\.worldSize\)/)
  assert.match(gridSource, /worldMapOverviewCoordinateFromPoint/)
  assert.match(gridSource, /onClick=\{handleOverviewClick\}/)
  assert.match(gridSource, /onPointerDown=\{\(event\) => event\.stopPropagation\(\)\}/)
  assert.match(gridSource, /title=\{`点击概览跳转地图坐标，当前中心 \(\$\{view\.centerX\}, \$\{view\.centerY\}\)，我的城池 \(\$\{view\.self\.x\}, \$\{view\.self\.y\}\)`\}/)
  assert.match(gridSource, /aria-label=\{`世界地图概览，点击跳转坐标，当前中心 \(\$\{view\.centerX\}, \$\{view\.centerY\}\)，我的城池 \(\$\{view\.self\.x\}, \$\{view\.self\.y\}\)`\}/)
  assert.match(gridSource, /<MapOverviewDirectionLabels \/>/)
  assert.match(gridSource, /const MapOverviewDirectionLabels: FC = \(\) =>/)
  assert.match(gridSource, /pointer-events-none absolute inset-0 z-40/)
  assert.match(gridSource, />北<\/span>/)
  assert.match(gridSource, />南<\/span>/)
  assert.match(gridSource, />西<\/span>/)
  assert.match(gridSource, />东<\/span>/)
  assert.match(gridSource, /title=\{`\$\{target\.nickname\} \(\$\{target\.position\.x\}, \$\{target\.position\.y\}\)`\}/)
  assert.match(gridSource, /title=\{`我的城池 \(\$\{view\.self\.x\}, \$\{view\.self\.y\}\)`\}/)
  assert.match(gridSource, /title=\{`当前选中 \(\$\{selectedPosition\?\.x\}, \$\{selectedPosition\?\.y\}\)`\}/)
  const tabSource = readFileSync(new URL('../src/pages/map/components/WorldMapTab.tsx', import.meta.url), 'utf8')
  assert.match(tabSource, /const overviewTargets = useMemo/)
  assert.match(tabSource, /targets\.filter\(\(target\) => isWorldMapRelationVisible\(target\.relation, relationFilters\)\)/)
  assert.match(tabSource, /const viewportTargets = useMemo/)
  assert.match(tabSource, /filterWorldMapTargetsInViewport\(targets, \{ x: centerX, y: centerY \}, viewport\.radius, \{ self: true, ally: true, other: true \}, targetView\?\.worldSize\)/)
  assert.match(tabSource, /const hiddenInViewportCount = viewportTargets\.length - visibleTargets\.length/)
  assert.match(tabSource, /const nearestTargets = useMemo\(\(\) => buildNearestWorldMapTargets\(overviewTargets, 5\), \[overviewTargets\]\)/)
  assert.match(tabSource, /overviewTargets=\{overviewTargets\}/)
  assert.match(tabSource, /hiddenInViewportCount=\{hiddenInViewportCount\}/)
  assert.match(tabSource, /const handleOverviewJump = useCallback/)
  assert.match(tabSource, /setCoordinateSearch\(\{ x: String\(position\.x\), y: String\(position\.y\) \}\)/)
  assert.match(tabSource, /setViewport\(\(prev\) => \(\{ \.\.\.prev, centerX: position\.x, centerY: position\.y \}\)\)/)
  assert.match(tabSource, /selectMapPosition\(position, '目标坐标有玩家城池，已自动显示并选中'\)/)
  assert.match(tabSource, /onJump=\{handleOverviewJump\}/)
})

test('世界地图提供最近城池快捷跳转', () => {
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapTab.tsx', import.meta.url), 'utf8')
  assert.match(source, /最近城池/)
  assert.match(source, /nearestTargets\.map\(\(target\) =>/)
  assert.match(source, /onClick=\{\(\) => handleOverviewJump\(target\.position\)\}/)
  assert.match(source, /worldMapRelationBadge\(target\.relation\)/)
  assert.match(source, /worldMapRelationBadgeClass\(target\.relation\)/)
  assert.match(source, /target\.direction \|\| directionFrom\(viewSelf\(targetView\), target\.position\)/)
  assert.match(source, /aria-label=\{`跳转到\$\{target\.nickname\}/)
  assert.match(source, /速度1预计行军 \$\{formatDuration\(target\.reinforceSeconds\)\}/)
  assert.match(source, /速度1预计行军\$\{formatDuration\(target\.reinforceSeconds\)\}/)
  assert.match(source, /target\.nickname/)
  assert.match(source, /target\.distance\}格/)
  assert.match(source, /formatDuration\(target\.reinforceSeconds\)/)
  assert.match(source, /target\.position\.x/)
  assert.match(source, /target\.position\.y/)
})

test('世界地图目标入缓存时按自己坐标重算距离和方位', () => {
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapTab.tsx', import.meta.url), 'utf8')
  assert.match(source, /const selfPosition = \{ worldId: targetResult\.self\.worldId, x: targetResult\.self\.x, y: targetResult\.self\.y, regionId: 0 \}/)
  assert.match(source, /worldTargetToPvpTarget\(target, targetResult\.worldId, selfPosition\)/)
  assert.match(source, /worldTargetToPvpTarget\(detail, targetView\.self\.worldId, targetView\.self\)/)
  assert.match(source, /function worldTargetToPvpTarget\(target: WorldMapTarget, worldId: string, self: PvpWorldPosition\): PvpTargetSummary/)
  assert.match(source, /const position = \{ worldId, x: target\.x, y: target\.y, regionId: 0 \}/)
  assert.match(source, /const metrics = buildWorldMapTargetMetrics\(self, position\)/)
  assert.match(source, /distance: metrics\.distance/)
  assert.match(source, /direction: metrics\.direction/)
  assert.match(source, /reinforceSeconds: metrics\.seconds/)
  assert.match(source, /attackReason: target\.attackReason/)
  assert.match(source, /plunderReason: target\.plunderReason/)
  assert.match(source, /reinforceReason: target\.reinforceReason/)
  assert.match(source, /protected: target\.status === 'protected' \|\| target\.status === 'truce'/)
  assert.doesNotMatch(source, /distance: target\.distance/)
  assert.doesNotMatch(source, /direction: target\.direction/)
  assert.doesNotMatch(source, /reinforceSeconds: estimateWorldMarchSeconds\(target\.distance, 1\)/)
})

test('世界地图缩放按钮按视野半径边界禁用', () => {
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapGrid.tsx', import.meta.url), 'utf8')
  assert.match(source, /const canZoomIn = radius > WORLD_MAP_MIN_VIEW_RADIUS/)
  assert.match(source, /const canZoomOut = radius < WORLD_MAP_MAX_VIEW_RADIUS/)
  assert.match(source, /disabled=\{!canZoomIn\}/)
  assert.match(source, /disabled=\{!canZoomOut\}/)
  assert.match(source, /title="放大格子"/)
  assert.match(source, /title="缩小格子"/)
  assert.doesNotMatch(source, /title="放大视野"/)
  assert.doesNotMatch(source, /title="缩小视野"/)
})

test('世界地图提供近中远视野预设切换', () => {
  const gridSource = readFileSync(new URL('../src/pages/map/components/WorldMapGrid.tsx', import.meta.url), 'utf8')
  assert.match(gridSource, /WORLD_MAP_VIEW_PRESETS/)
  assert.match(gridSource, /onSetRadius: \(radius: number\) => void/)
  assert.match(gridSource, /WORLD_MAP_VIEW_PRESETS\.map\(\(preset\) =>/)
  assert.match(gridSource, /active=\{radius === preset\.radius\}/)
  assert.match(gridSource, /onClick=\{\(\) => onSetRadius\(preset\.radius\)\}/)
  assert.match(gridSource, /aria-pressed=\{active\}/)
  assert.match(gridSource, /MapRadiusPresetButton/)
  const tabSource = readFileSync(new URL('../src/pages/map/components/WorldMapTab.tsx', import.meta.url), 'utf8')
  assert.match(tabSource, /const DEFAULT_VIEW_RADIUS = WORLD_MAP_MIN_VIEW_RADIUS/)
  assert.match(tabSource, /useState<\{ centerX\?: number; centerY\?: number; radius: number \}>\(\{ radius: DEFAULT_VIEW_RADIUS \}\)/)
  assert.match(tabSource, /radius: DEFAULT_VIEW_RADIUS/)
  assert.match(tabSource, /setViewport\(\{ radius: DEFAULT_VIEW_RADIUS \}\)/)
  assert.match(tabSource, /const setViewportRadius = useCallback\(\(radius: number\) =>/)
  assert.match(tabSource, /radius: clampWorldMapRadius\(radius\)/)
  assert.match(tabSource, /onSetRadius=\{setViewportRadius\}/)
})

test('世界地图主画布展示中心轴和距离辅助环', () => {
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapGrid.tsx', import.meta.url), 'utf8')
  assert.doesNotMatch(source, /<MapCompassRose \/>/)
  assert.doesNotMatch(source, /const MapCompassRose: FC = \(\) =>/)
  assert.doesNotMatch(source, /pointer-events-none absolute left-3 top-3 z-30/)
  assert.match(source, /<MapDistanceGuides guides=\{distanceGuides\} gridSize=\{gridSize\} centerOffsetX=\{centerOffsetX\} centerOffsetY=\{centerOffsetY\} \/>/)
  assert.match(source, /const centerLeft = `\$\{\(\(centerOffsetX \+ 0\.5\) \/ gridSize\) \* 100\}%`/)
  assert.match(source, /const centerTop = `\$\{\(\(centerOffsetY \+ 0\.5\) \/ gridSize\) \* 100\}%`/)
  assert.match(source, /style=\{\{ left: centerLeft \}\}/)
  assert.match(source, /style=\{\{ top: centerTop \}\}/)
  assert.match(source, /<polygon/)
  assert.match(source, /points=\{guide\.points\}/)
  assert.match(source, /style=\{\{ left: guide\.labelLeft, top: guide\.labelTop \}\}/)
  assert.match(source, /guide\.distance\}格/)
  assert.doesNotMatch(source, /style=\{\{ left: guide\.left, top: guide\.top, width: guide\.width, height: guide\.height \}\}/)
})

test('世界地图当前视野空状态提示筛选隐藏城池数量', () => {
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapGrid.tsx', import.meta.url), 'utf8')
  const tabSource = readFileSync(new URL('../src/pages/map/components/WorldMapTab.tsx', import.meta.url), 'utf8')
  assert.match(source, /hiddenInViewportCount: number/)
  assert.match(source, /const viewportTargetCount = targets\.length \+ hiddenInViewportCount/)
  assert.match(source, /当前视野 \{targets\.length\}\/\{viewportTargetCount\}/)
  assert.match(source, /hiddenInViewportCount > 0/)
  assert.match(source, /当前视野有 \$\{hiddenInViewportCount\} 个玩家城池已被筛选隐藏/)
  assert.match(source, /当前视野没有玩家城池，可以缩小比例或移动地图/)
  assert.doesNotMatch(source, /totalTargets/)
  assert.doesNotMatch(tabSource, /totalTargets=\{targets\.length\}/)
})

test('世界地图顶部展示当前指向格子的坐标方位和距离', () => {
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapGrid.tsx', import.meta.url), 'utf8')
  assert.match(source, /const viewportBounds = useMemo\(\(\) => buildWorldMapViewportBounds\(\{ x: view\.centerX, y: view\.centerY \}, radius, view\.worldSize\)/)
  assert.match(source, /const \{ gridSize, minX, minY, maxX: boundedMaxX, maxY: boundedMaxY, centerOffsetX, centerOffsetY \} = viewportBounds/)
  assert.match(source, /const boundedMinX = minX/)
  assert.match(source, /const boundedMinY = minY/)
  assert.match(source, /const canPanWest = boundedMinX > 0/)
  assert.match(source, /const canPanEast = boundedMaxX < view\.worldSize - 1/)
  assert.match(source, /const canPanNorth = boundedMinY > 0/)
  assert.match(source, /const canPanSouth = boundedMaxY < view\.worldSize - 1/)
  assert.match(source, /X \{boundedMinX\}-\{boundedMaxX\} · Y \{boundedMinY\}-\{boundedMaxY\}/)
  assert.match(source, /const worldEdgeLabels = \[/)
  assert.match(source, /boundedMinY === 0 \? '北界' : ''/)
  assert.match(source, /boundedMaxY === view\.worldSize - 1 \? '南界' : ''/)
  assert.match(source, /boundedMinX === 0 \? '西界' : ''/)
  assert.match(source, /boundedMaxX === view\.worldSize - 1 \? '东界' : ''/)
  assert.match(source, /worldEdgeLabels\.length > 0/)
  assert.match(source, /worldEdgeLabels\.join\(' · '\)/)
  assert.match(source, /label="左移" disabled=\{!canPanWest\}/)
  assert.match(source, /label="上移" disabled=\{!canPanNorth\}/)
  assert.match(source, /label="下移" disabled=\{!canPanSouth\}/)
  assert.match(source, /label="右移" disabled=\{!canPanEast\}/)
  assert.match(source, /const MapControlButton: FC<\{ label: string; disabled\?: boolean; onClick: \(\) => void; children: ReactNode \}>/)
  assert.doesNotMatch(source, /hoveredCell/)
  assert.doesNotMatch(source, /setHoveredCell/)
  assert.match(source, /const inspectedCell = selectedCell \?\? focusedTargetPosition/)
  assert.match(source, /const centerCell = \{ x: view\.centerX, y: view\.centerY \}/)
  assert.match(source, /const centerDirection = directionFrom\(view\.self, centerCell\)/)
  assert.match(source, /const centerDistance = Math\.abs\(centerCell\.x - view\.self\.x\) \+ Math\.abs\(centerCell\.y - view\.self\.y\)/)
  assert.match(source, /中心距我 \{centerDistance\}格 · \{centerDirection\}/)
  assert.match(source, /指向 \(\{inspectedCell\.x\}, \{inspectedCell\.y\}\)/)
  assert.match(source, /directionFrom\(view\.self, inspectedCell\)/)
  assert.match(source, /Math\.abs\(inspectedCell\.x - view\.self\.x\) \+ Math\.abs\(inspectedCell\.y - view\.self\.y\)/)
  assert.doesNotMatch(source, /onPointerEnter=\{\(\) => setHoveredCell/)
  assert.doesNotMatch(source, /onPointerLeave=\{\(\) => setHoveredCell/)
  assert.match(source, /<MapSelectedRoute self=\{view\.self\} target=\{inspectedCell\} minX=\{minX\} minY=\{minY\} gridSize=\{gridSize\} \/>/)
  assert.match(source, /const MapSelectedRoute: FC<\{ self: GridPosition; target: GridPosition \| null \| undefined; minX: number; minY: number; gridSize: number \}>/)
  assert.match(source, /if \(!isPositionInView\(self, minX, minY, gridSize\) \|\| !isPositionInView\(target, minX, minY, gridSize\)\) return null/)
  assert.match(source, /const horizontalWidth = Math\.abs\(targetX - selfX\)/)
  assert.match(source, /const verticalHeight = Math\.abs\(targetY - selfY\)/)
  assert.match(source, /pointer-events-none absolute inset-0 z-\[15\]/)
  assert.match(source, /style=\{\{ left: `\$\{horizontalLeft\}%`, top: `\$\{selfY\}%`, width: `\$\{horizontalWidth\}%` \}\}/)
  assert.match(source, /style=\{\{ left: `\$\{targetX\}%`, top: `\$\{verticalTop\}%`, height: `\$\{verticalHeight\}%` \}\}/)
})

test('世界地图自己城池兜底显示受自己筛选控制', () => {
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapGrid.tsx', import.meta.url), 'utf8')
  assert.match(source, /showSelf: boolean/)
  assert.match(source, /const selfVisible = showSelf && isPositionInView\(view\.self, minX, minY, gridSize\)/)
  assert.doesNotMatch(source, /const selfVisible = isPositionInView\(view\.self, minX, minY, gridSize\)/)
  assert.match(source, /selfVisible && !hasSelfTarget && \(/)
  assert.match(source, /<button\s+type="button"\s+className="z-20 flex h-full w-full items-center justify-center"/)
  assert.match(source, /aria-label=\{`我的城池，坐标 \(\$\{view\.self\.x\}, \$\{view\.self\.y\}\)`\}/)
  assert.match(source, /onPointerDown=\{\(event\) => event\.stopPropagation\(\)\}/)
  assert.match(source, /onClick=\{\(event\) => \{\s*event\.stopPropagation\(\)\s*onFocusSelf\(\)\s*\}\}/)
})

test('世界地图草地格和城池格使用同一个坐标网格', () => {
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapGrid.tsx', import.meta.url), 'utf8')
  assert.match(source, /const centerCell = \{ x: view\.centerX, y: view\.centerY \}/)
  assert.match(source, /const isInspectedAxisCell = \(cell: GridPosition\) => inspectedCell \? cell\.x === inspectedCell\.x \|\| cell\.y === inspectedCell\.y : false/)
  assert.match(source, /display: 'grid'/)
  assert.match(source, /gridTemplateColumns: `repeat\(\$\{gridSize\}, minmax\(0, 1fr\)\)`/)
  assert.match(source, /gridTemplateRows: `repeat\(\$\{gridSize\}, minmax\(0, 1fr\)\)`/)
  assert.match(source, /visibleCells\.map\(\(cell\) =>/)
  assert.match(source, /key=\{`\$\{cell\.x\}:\$\{cell\.y\}`\}/)
  assert.match(source, /aria-label=\{`草地坐标格 \(\$\{cell\.x\}, \$\{cell\.y\}\)，距离按 1 格计算`\}/)
  assert.match(source, /className=\{`relative h-full w-full border border-emerald-950\/10 bg-\[#6f9b52\] bg-cover bg-center bg-no-repeat p-0/)
  assert.match(source, /title=\{`草地坐标格 \(\$\{cell\.x\}, \$\{cell\.y\}\)`\}/)
  assert.match(source, /style=\{\{ gridColumn: cell\.x - minX \+ 1, gridRow: cell\.y - minY \+ 1, backgroundImage: `url\(\$\{tileGrass\}\)` \}\}/)
  assert.match(source, /isInspectedAxisCell\(cell\) \? 'brightness-110 saturate-125' : ''/)
  assert.match(source, /inspectedCell && cell\.x === inspectedCell\.x &&/)
  assert.match(source, /inspectedCell && cell\.y === inspectedCell\.y &&/)
  assert.match(source, /bg-amber-300\/50/)
  assert.match(source, /isSameGridPosition\(centerCell, cell\) \? 'ring-2 ring-emerald-950\/45 ring-inset' : ''/)
  assert.match(source, /isSameGridPosition\(centerCell, cell\) &&/)
  assert.match(source, /title="中心格"/)
  assert.doesNotMatch(source, /text-\[7px\] font-bold leading-none text-emerald-950\/55/)
  assert.doesNotMatch(source, /absolute inset-x-0 bottom-0 z-30 truncate bg-amber-400\/90/)
  assert.match(source, /const toGridStyle = \(position: PvpWorldPosition\) => \(\{\s*gridColumn: position\.x - minX \+ 1,\s*gridRow: position\.y - minY \+ 1,\s*\}\)/)
  assert.match(source, /style=\{toGridStyle\(target\.position\)\}/)
  assert.match(source, /style=\{toGridStyle\(view\.self\)\}/)
  assert.match(source, /className=\{`z-20 flex h-full w-full items-center justify-center bg-transparent/)
  assert.match(source, /relative flex aspect-square h-full w-full min-h-3/)
  assert.match(source, /一格=距离1/)
  assert.match(source, /bg-\[#5f8f3e\] shadow-inner/)
  assert.doesNotMatch(source, /overflow-hidden bg-\[#5d8441\]/)
  assert.match(source, /const grassCellSize = `\$\{100 \/ view\.worldSize\}% \$\{100 \/ view\.worldSize\}%`/)
  assert.match(source, /style=\{\{ backgroundSize: grassCellSize \}\}/)
  assert.doesNotMatch(source, /bg-\[length:20%_20%\]/)
})

test('世界地图城池视觉占满对应坐标格', () => {
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapGrid.tsx', import.meta.url), 'utf8')
  assert.doesNotMatch(source, /h-\[68%\]/)
  assert.doesNotMatch(source, /hover:scale-125/)
  assert.match(source, /aspect-square h-full w-full/)
  assert.doesNotMatch(source, /absolute -right-1 -top-1/)
  assert.doesNotMatch(source, /absolute -bottom-1 -right-1/)
  assert.doesNotMatch(source, /absolute -bottom-1 -left-1/)
  assert.doesNotMatch(source, /absolute -top-6/)
  assert.match(source, /absolute right-0 top-0/)
  assert.match(source, /absolute bottom-0 right-0/)
  assert.match(source, /absolute bottom-0 left-0/)
  assert.match(source, /absolute inset-x-0 top-0/)
})

test('世界地图远景城池自动隐藏拥挤角标', () => {
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapGrid.tsx', import.meta.url), 'utf8')
  assert.match(source, /import \{ LocateFixed, Minus, Plus \} from 'lucide-react'/)
  assert.doesNotMatch(source, /Castle/)
  assert.doesNotMatch(source, /House/)
  assert.match(source, /const showCityDetail = showCellCoordinate/)
  assert.match(source, /<img className="pointer-events-none absolute inset-0 h-full w-full object-cover" src=\{cityTileSrc\(\)\} alt="" aria-hidden="true" \/>/)
  assert.match(source, /<img className="pointer-events-none absolute inset-0 h-full w-full object-cover" src=\{cityTileSrc\(target\.faction\)\} alt="" aria-hidden="true" \/>/)
  assert.match(source, /!showCityDetail && <span className="relative z-20 rounded bg-sky-950\/75 px-0\.5 leading-none" aria-hidden="true">我<\/span>/)
  assert.match(source, /showCityDetail && \(/)
  assert.match(source, /showCityDetail && statusBadge &&/)
  assert.match(source, /showCityDetail && marchBadge &&/)
})

test('世界地图选中城池会在格子上显示目标标牌', () => {
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapGrid.tsx', import.meta.url), 'utf8')
  assert.match(source, /focused &&/)
  assert.match(source, /const metrics = buildWorldMapTargetMetrics\(view\.self, target\.position\)/)
  assert.match(source, /target\.nickname/)
  assert.match(source, /metrics\.distance/)
  assert.match(source, /metrics\.direction/)
  assert.match(source, /formatDuration\(metrics\.seconds\)/)
  assert.match(source, /aria-label=\{`选中\$\{target\.nickname\}，\$\{metrics\.direction\}，坐标 \(\$\{target\.position\.x\}, \$\{target\.position\.y\}\)，距离 \$\{metrics\.distance\} 格，行军约 \$\{formatDuration\(metrics\.seconds\)\}`\}/)
  assert.match(source, /marchBadges/)
  assert.match(source, /marchBadge/)
})

test('世界地图草地和城池点击分别进入空地或目标选择', () => {
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapGrid.tsx', import.meta.url), 'utf8')
  const tabSource = readFileSync(new URL('../src/pages/map/components/WorldMapTab.tsx', import.meta.url), 'utf8')
  assert.match(source, /onClick=\{\(\) => handleGrassClick\(cell\.x, cell\.y\)\}/)
  assert.match(source, /onSelectCell\(x, y\)/)
  assert.match(tabSource, /selectMapPosition\(\{ x, y \}, '该格有玩家城池，已自动显示并选中'\)/)
  assert.match(source, /onPointerDown=\{\(event\) => event\.stopPropagation\(\)\}/)
  assert.match(source, /event\.stopPropagation\(\)[\s\S]*onFocusTarget\(target\.playerId\)/)
})

test('地图页恢复分类页签并把世界地图放在玩家页签', () => {
  const source = readFileSync(new URL('../src/pages/map/MapPage.tsx', import.meta.url), 'utf8')
  const indexSource = readFileSync(new URL('../src/pages/map/index.ts', import.meta.url), 'utf8')
  assert.match(indexSource, /export \{ default \} from '\.\/MapPage'/)
  assert.match(source, /type MapTab = 'npc' \| 'players' \| 'stronghold' \| 'dungeon' \| 'minigames'/)
  assert.match(source, /Castle, Users, Flag, Scroll, Sparkles/)
  assert.match(source, /const NpcCityTab = lazy\(\(\) => import\('\.\/components\/NpcCityTab'\)\)/)
  assert.match(source, /const DungeonTab = lazy\(\(\) => import\('\.\/components\/DungeonTab'\)\)/)
  assert.match(source, /const MiniGamesTab = lazy\(\(\) => import\('\.\/components\/MiniGamesTab'\)\)/)
  assert.match(source, /useState<MapTab>\('npc'\)/)
  assert.match(source, /\{ key: 'npc' as const, label: 'NPC'/)
  assert.match(source, /\{ key: 'players' as const, label: '玩家'/)
  assert.match(source, /\{ key: 'stronghold' as const, label: '据点'/)
  assert.match(source, /\{ key: 'dungeon' as const, label: '副本'/)
  assert.match(source, /\{ key: 'minigames' as const, label: '万象幻境'/)
  assert.match(source, /if \(activeTab === 'npc'\) return <NpcCityTab \/>/)
  assert.match(source, /if \(activeTab === 'players'\) return <WorldMapTab \/>/)
  assert.match(source, /if \(activeTab === 'stronghold'\) return <StrongholdTab \/>/)
  assert.match(source, /if \(activeTab === 'dungeon'\) return <DungeonTab \/>/)
  assert.match(source, /return <MiniGamesTab \/>/)
  assert.match(source, /<Suspense fallback=/)
  assert.match(source, /onClick=\{\(\) => handleTabClick\(tab\.key, index\)\}/)
  assert.match(source, /aria-pressed=\{isActive\}/)
  assert.match(source, /bg-\[var\(--color-surface-dim\)\]/)
  assert.match(source, /shadow-\[0_2px_8px_rgba\(15,23,42,0\.06\)\]/)
})

test('世界地图目标面板不展示目标兵力或资源详情', () => {
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapTargetPanel.tsx', import.meta.url), 'utf8')
  assert.match(source, /target\.buildingLevel/)
  assert.match(source, /statusLabel\(target\.status\)/)
  assert.match(source, /worldMapRelationLabel\(target\.relation\)/)
  assert.match(source, /worldMapRelationBadge\(target\.relation\).*worldMapRelationLabel\(target\.relation\)/)
  assert.match(source, /速度1 预计行军/)
  assert.match(source, /buildWorldMapTargetMetrics\(selfPosition, target\.position\)/)
  assert.match(source, /formatDuration\(metrics\.seconds\)/)
  assert.doesNotMatch(source, /target\.totalArmy/)
  assert.doesNotMatch(source, /兵力/)
  assert.doesNotMatch(source, /资源/)
  assert.doesNotMatch(source, /驻防/)
  const gridSource = readFileSync(new URL('../src/pages/map/components/WorldMapGrid.tsx', import.meta.url), 'utf8')
  assert.match(gridSource, /行军约/)
  assert.doesNotMatch(gridSource, /增援约/)
})

test('世界地图目标面板接入四类玩家城池操作', () => {
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapTargetPanel.tsx', import.meta.url), 'utf8')
  assert.match(source, /actionByKey\.scout/)
  assert.match(source, /actionByKey\.attack/)
  assert.match(source, /actionByKey\.plunder/)
  assert.match(source, /actionByKey\.reinforce/)
  assert.match(source, /const disabledReasons = actions\.filter\(\(action\) => action\.reason\)/)
  assert.match(source, /label="侦查"[\s\S]*onClick=\{\(\) => void onScout\(target\)\}/)
  assert.match(source, /label="侦查"[\s\S]*label="攻击"[\s\S]*label="掠夺"[\s\S]*label="增援"/)
  assert.match(source, /label="攻击"[\s\S]*onClick=\{\(\) => onMarch\(target, 'attack'\)\}/)
  assert.match(source, /label="掠夺"[\s\S]*onClick=\{\(\) => onMarch\(target, 'plunder'\)\}/)
  assert.match(source, /label="增援"[\s\S]*onClick=\{\(\) => onReinforce\(target\)\}/)
  assert.match(source, /disabledReasons\.length > 0/)
  assert.match(source, /disabledReasons\.map\(\(action\) =>/)
  assert.match(source, /<span key=\{`\$\{action\.key\}:\$\{action\.reason\}`\}/)
  assert.match(source, /\{action\.label\}：\{action\.reason\}/)
  assert.doesNotMatch(source, /target\.reason && <p/)
})

test('世界地图目标面板可以取消当前选择', () => {
  const panelSource = readFileSync(new URL('../src/pages/map/components/WorldMapTargetPanel.tsx', import.meta.url), 'utf8')
  assert.match(panelSource, /onClearSelection: \(\) => void/)
  assert.match(panelSource, /PanelCloseButton onClick=\{onClearSelection\}/)
  assert.match(panelSource, /title="取消选择"/)
  assert.match(panelSource, /<X size=\{13\} \/>/)
  const tabSource = readFileSync(new URL('../src/pages/map/components/WorldMapTab.tsx', import.meta.url), 'utf8')
  assert.match(tabSource, /const clearMapSelection = useCallback\(\(\) => \{/)
  assert.match(tabSource, /setFocusedTargetId\(null\)/)
  assert.match(tabSource, /setSelectedEmptyCell\(null\)/)
  assert.match(tabSource, /onClearSelection=\{clearMapSelection\}/)
})

test('世界地图出征和增援弹窗使用结构化行军预览', () => {
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapTab.tsx', import.meta.url), 'utf8')
  assert.match(source, /travelPreview=\{buildTravelPreviewItems\(selectedReinforceTarget/)
  assert.match(source, /travelPreview=\{buildTravelPreviewItems\(selectedMarchTarget/)
  assert.match(source, /interface TravelPreviewItem/)
  assert.match(source, /travelPreview: TravelPreviewItem\[\]/)
  assert.match(source, /travelPreview\.map\(\(item\) =>/)
  assert.match(source, /buildWorldMapTargetMetrics\(viewSelf\(targetView\), selectedReinforceTarget\.position, speed\)/)
  assert.match(source, /buildWorldMapTargetMetrics\(viewSelf\(targetView\), selectedMarchTarget\.position, speed\)/)
  assert.match(source, /const metrics = buildWorldMapTargetMetrics\(self, target\.position, speed\)/)
  assert.match(source, /label: '坐标'/)
  assert.match(source, /label: '方位'/)
  assert.match(source, /label: '距离'/)
  assert.match(source, /label: '最慢速度'/)
  assert.match(source, /label: '预计行军'/)
  assert.match(source, /value: metrics\.direction/)
  assert.match(source, /value: `\$\{metrics\.distance\}格`/)
  assert.match(source, /formatDuration\(seconds\)/)
  assert.doesNotMatch(source, /description=\{`坐标/)
})

test('世界地图主组件不再沿用旧 PVP 地图文案', () => {
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapTab.tsx', import.meta.url), 'utf8')
  assert.match(source, /打开世界地图出征兵力选择面板/)
  assert.match(source, /发起世界地图行军失败/)
  assert.match(source, /移动世界地图视野/)
  assert.match(source, /缩放世界地图视野/)
  assert.match(source, /const playerCityMarches = marches\.filter/)
  assert.doesNotMatch(source, /const pvpMarches = marches\.filter/)
  assert.doesNotMatch(source, /PVP 地图/)
  assert.doesNotMatch(source, /PVP 出征/)
  assert.doesNotMatch(source, /PVP 行军/)
})

test('世界地图操作成功后静默刷新目标缓存', () => {
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapTab.tsx', import.meta.url), 'utf8')
  assert.match(source, /const result = await gameApi\.scoutPvpTarget[\s\S]*patchState\(\{[\s\S]*army: result\.army[\s\S]*serverTime: result\.serverTime[\s\S]*\}\)[\s\S]*setMarches\(\(prev\) => \[result\.march, \.\.\.prev\]\)[\s\S]*侦查队已出发/)
  assert.doesNotMatch(source, /setScoutReport/)
  assert.match(source, /const result = await gameApi\.startPvpAttack[\s\S]*setMarches\(\(prev\) => \[result\.march, \.\.\.prev\]\)[\s\S]*void useGameStore\.getState\(\)\.loadMilitaryView\(\)[\s\S]*void load\(true\)[\s\S]*toast\.success\(`已向 \$\{target\.nickname\} 发起/)
  assert.match(source, /const result = await gameApi\.sendReinforcement[\s\S]*setReinforcements\(\(prev\) => mergeWorldMapReinforcements\(prev, \[result\.reinforcement\]\)\)[\s\S]*setSelectedReinforceTarget\(null\)[\s\S]*void useGameStore\.getState\(\)\.loadMilitaryView\(\)[\s\S]*void load\(true\)[\s\S]*toast\.success\(`已向 \$\{target\.nickname\} 派出增援。`\)/)
})

test('世界地图行军命令使用同步请求锁阻止快速重复提交', () => {
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapTab.tsx', import.meta.url), 'utf8')
  assert.match(source, /const commandInFlightRef = useRef\(false\)/)
  assert.equal((source.match(/commandInFlightRef\.current = true/g) ?? []).length, 3)
  assert.equal((source.match(/commandInFlightRef\.current = false/g) ?? []).length, 3)
  assert.equal((source.match(/busyTarget \|\| commandInFlightRef\.current/g) ?? []).length, 3)
})

test('世界地图攻击与增援共享命令锁且失败请求不回写权威状态', () => {
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapTab.tsx', import.meta.url), 'utf8')
  const march = source.match(/const handleMarch = async \(\) => \{[\s\S]*?\n {2}\}\n\n {2}\/\/ handleReinforce/)
  const reinforce = source.match(/const handleReinforce = async \(\) => \{[\s\S]*?\n {2}\}\n\n {2}\/\/ panViewport/)
  assert.ok(march)
  assert.ok(reinforce)
  for (const handler of [march[0], reinforce[0]]) {
    assert.match(handler, /busyTarget \|\| commandInFlightRef\.current/)
    assert.match(handler, /commandInFlightRef\.current = true/)
    assert.match(handler, /commandInFlightRef\.current = false/)
    const failurePath = handler.match(/\} catch \(err\) \{[\s\S]*?\} finally/)
    assert.ok(failurePath)
    assert.match(failurePath[0], /toast\.error/)
    assert.doesNotMatch(failurePath[0], /patchState|setMarches|setReinforcements/)
  }
})

test('世界地图增援使用同步请求锁并采用后端权威兵力和到达时间', () => {
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapTab.tsx', import.meta.url), 'utf8')
  const handler = source.match(/const handleReinforce = async \(\) => \{[\s\S]*?\n {2}\}\n\n {2}\/\/ panViewport/)
  assert.ok(handler)
  assert.match(handler[0], /busyTarget \|\| commandInFlightRef\.current/)
  assert.match(handler[0], /commandInFlightRef\.current = true[\s\S]*gameApi\.sendReinforcement/)
  assert.match(handler[0], /useGameStore\.getState\(\)\.patchState\(result\.patch\)/)
  assert.match(handler[0], /mergeWorldMapReinforcements\(prev, \[result\.reinforcement\]\)/)
  assert.match(handler[0], /commandInFlightRef\.current = false/)
  assert.doesNotMatch(handler[0], /estimateWorldMarchSeconds/)
})

test('世界地图侦查失败不会提前回写兵力或创建本地行军', () => {
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapTab.tsx', import.meta.url), 'utf8')
  const handler = source.match(/const handleScout = useCallback\(async \(target: PvpTargetSummary\) => \{[\s\S]*?\n {2}\}, \[activePlayerId, busyTarget\]\)/)
  assert.ok(handler)
  assert.match(handler[0], /const result = await gameApi\.scoutPvpTarget[\s\S]*patchState\(\{[\s\S]*army: result\.army[\s\S]*serverTime: result\.serverTime[\s\S]*\}\)[\s\S]*setMarches/)
  const failurePath = handler[0].match(/\} catch \(err\) \{[\s\S]*?\} finally/)
  assert.ok(failurePath)
  assert.match(failurePath[0], /toast\.error/)
  assert.doesNotMatch(failurePath[0], /patchState|setMarches/)
})

test('世界地图三类派兵成功都同步留城特性的权威结算状态', () => {
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapTab.tsx', import.meta.url), 'utf8')
  const types = readFileSync(new URL('../src/types/game.ts', import.meta.url), 'utf8')
  for (const handlerPattern of [
    /const handleScout = useCallback[\s\S]*?\n {2}\}, \[activePlayerId, busyTarget\]\)/,
    /const handleMarch = async \(\) => \{[\s\S]*?\n {2}\}/,
  ]) {
    const handler = source.match(handlerPattern)
    assert.ok(handler)
    for (const field of ['resources', 'resourceProduction', 'resourceSettledAt', 'generalTraitProgress']) {
      assert.match(handler[0], new RegExp(`${field}: result\\.${field}`))
    }
  }
  assert.match(source, /gameApi\.sendReinforcement[\s\S]*patchState\(result\.patch\)/)
  assert.match(types, /interface GarrisonActionResult[\s\S]*resources\?: ResourceState[\s\S]*resourceProduction\?: ResourceProduction[\s\S]*resourceSettledAt\?: string[\s\S]*generalTraitProgress\?: Record<string, number>/)
})

test('世界地图空地面板显示方位和距离', () => {
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapTargetPanel.tsx', import.meta.url), 'utf8')
  assert.match(source, /selfPosition/)
  assert.match(source, /directionFrom\(selfPosition, emptyCell\)/)
  assert.match(source, /distanceFrom\(selfPosition, emptyCell\)/)
  assert.match(source, /const estimatedSeconds = estimateWorldMarchSeconds\(distance, 1\)/)
  assert.match(source, /速度1 预计行军/)
  assert.match(source, /第一版暂无操作/)
})

test('世界地图存档加载前的 Zustand selector 与兵力投影保持稳定', () => {
  const source = readFileSync(new URL('../src/pages/map/components/WorldMapTab.tsx', import.meta.url), 'utf8')
  assert.match(source, /const EMPTY_GENERALS: General\[\] = \[\]/)
  assert.match(source, /const EMPTY_GENERAL_ASSIGNMENTS: GeneralAssignment\[\] = \[\]/)
  assert.match(source, /s\.state\?\.generals \?\? EMPTY_GENERALS/)
  assert.match(source, /s\.state\?\.generalAssignments \?\? EMPTY_GENERAL_ASSIGNMENTS/)
  assert.match(source, /const army = useProjectedArmy\(\)/)
  assert.doesNotMatch(source, /useGameStore\(\(s\) => s\.state\?\.(?:generals|generalAssignments|army) \?\? \[\]\)/)
})
