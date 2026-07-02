// 本文件实现地图页的世界地图玩家城池视图和快捷操作。
import { useCallback, useEffect, useMemo, useRef, useState, type FC } from 'react'
import { Eye, LoaderCircle, RefreshCw } from 'lucide-react'
import { gameApi } from '@/api/game'
import { toast } from '@/components/ui'
import { useGameStore } from '@/store/gameStore'
import { useConfigStore } from '@/store/configStore'
import type { ArmyUnit, General, GeneralAssignment, PvpMarch, PvpTargetSummary, PvpTargetsResponse, PvpWorldPosition, Reinforcement, WorldMapTarget, WorldMapViewResponse } from '@/types/game'
import WorldMapCoordinateSearch from './WorldMapCoordinateSearch'
import WorldMapFilters from './WorldMapFilters'
import WorldMapGrid from './WorldMapGrid'
import WorldMapLegend from './WorldMapLegend'
import WorldMapTargetPanel from './WorldMapTargetPanel'
import { buildNearestWorldMapTargets, buildWorldMapFactionCounts, buildWorldMapMarchBadges, buildWorldMapMarchSummary, buildWorldMapReinforcementMarches, buildWorldMapRelationCounts, buildWorldMapTargetMetrics, clampWorldMapRadius, directionFrom, filterWorldMapTargetsInViewport, findVisibleWorldMapTargetAtCell, findWorldMapTargetAtCell, formatDuration, formatWorldMapSyncTime, isWorldMapRelationVisible, mergeWorldMapTargetCache, moveWorldMapCenter, parseWorldMapCoordinateSearch, worldMapRelationBadge, worldMapRelationBadgeClass, WORLD_MAP_FULL_LOAD_RADIUS, WORLD_MAP_MIN_VIEW_RADIUS } from '../worldMapGridLogic'

// 共享空数组保证 Zustand selector 在存档加载前返回稳定引用，避免刷新页面时触发无限更新。
const EMPTY_GENERALS: General[] = []
const EMPTY_GENERAL_ASSIGNMENTS: GeneralAssignment[] = []
const EMPTY_ARMY: ArmyUnit[] = []
const DEFAULT_VIEW_RADIUS = WORLD_MAP_MIN_VIEW_RADIUS

// WorldMapTab 展示世界地图玩家城池、筛选、坐标查找和行军状态。
const WorldMapTab: FC = () => {
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const generals = useGameStore((s) => s.state?.generals ?? EMPTY_GENERALS)
  const generalAssignments = useGameStore((s) => s.state?.generalAssignments ?? EMPTY_GENERAL_ASSIGNMENTS)
  const army = useGameStore((s) => s.state?.army ?? EMPTY_ARMY)
  const faction = useGameStore((s) => s.state?.player.faction ?? 'wei')
  const units = useConfigStore((s) => s.units)
  const [targets, setTargets] = useState<PvpTargetSummary[]>([])
  const [targetView, setTargetView] = useState<PvpTargetsResponse | null>(null)
  const [marches, setMarches] = useState<PvpMarch[]>([])
  const [reinforcements, setReinforcements] = useState<Reinforcement[]>([])
  const [viewport, setViewport] = useState<{ centerX?: number; centerY?: number; radius: number }>({ radius: DEFAULT_VIEW_RADIUS })
  const [focusedTargetId, setFocusedTargetId] = useState<string | null>(null)
  const [selectedEmptyCell, setSelectedEmptyCell] = useState<{ x: number; y: number } | null>(null)
  const [relationFilters, setRelationFilters] = useState<Record<string, boolean>>({ self: true, ally: true, other: true })
  const [coordinateSearch, setCoordinateSearch] = useState({ x: '', y: '' })
  const [mapServerTime, setMapServerTime] = useState('')
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [loadError, setLoadError] = useState('')
  const [busyTarget, setBusyTarget] = useState<string | null>(null)
  const [selectedMarchTarget, setSelectedMarchTarget] = useState<PvpTargetSummary | null>(null)
  const [selectedReinforceTarget, setSelectedReinforceTarget] = useState<PvpTargetSummary | null>(null)
  const [selectedMarchMode, setSelectedMarchMode] = useState<'attack' | 'plunder'>('attack')
  const [selections, setSelections] = useState<Record<string, number>>({})
  const [selectedGeneralIds, setSelectedGeneralIds] = useState<string[]>([])
  const autoRefreshingRef = useRef(false)
  const lastAutoRefreshAtRef = useRef(0)
  const loadRequestRef = useRef(0)
  const globalLoadRequestRef = useRef(0)
  const auxiliaryLoadRequestRef = useRef(0)
  const globalLoadTimerRef = useRef<number | null>(null)
  const targetDetailRequestRef = useRef(0)
  const hasLoadedMapRef = useRef(false)
  const loadedPlayerRef = useRef<string | null>(null)

  // applyWorldMapResult 将世界地图接口结果写入目标缓存，视野半径仍由本地近中远控件控制。
  const applyWorldMapResult = useCallback((targetResult: WorldMapViewResponse, shouldSyncInitialCoordinate: boolean) => {
    const selfPosition = { worldId: targetResult.self.worldId, x: targetResult.self.x, y: targetResult.self.y, regionId: 0 }
    setTargets((targetResult.targets ?? []).map((target) => worldTargetToPvpTarget(target, targetResult.worldId, selfPosition)))
    setTargetView({
      items: [],
      self: selfPosition,
      worldSize: Math.min(targetResult.width, targetResult.height),
      worldWidth: targetResult.width,
      worldHeight: targetResult.height,
      centerX: targetResult.centerX,
      centerY: targetResult.centerY,
      radius: DEFAULT_VIEW_RADIUS,
    })
    if (shouldSyncInitialCoordinate) setCoordinateSearch({ x: String(selfPosition.x), y: String(selfPosition.y) })
    setMapServerTime(targetResult.serverTime)
  }, [])

  // loadGlobalWorldMapTargets 在首屏近景渲染后懒加载全局目标，避免大范围地图阻塞首次显示。
  const loadGlobalWorldMapTargets = useCallback(async (playerId: string, parentRequestId: number) => {
    const requestId = globalLoadRequestRef.current + 1
    globalLoadRequestRef.current = requestId
    try {
      const targetResult = await gameApi.getWorldMapView(playerId, { radius: WORLD_MAP_FULL_LOAD_RADIUS })
      if (requestId !== globalLoadRequestRef.current) return
      if (parentRequestId !== loadRequestRef.current) return
      if (loadedPlayerRef.current !== playerId) return
      applyWorldMapResult(targetResult, false)
    } catch {
      // 全局缓存只影响缩略图、统计和最近城池，不阻塞首屏地图使用。
    }
  }, [applyWorldMapResult])

  // scheduleGlobalWorldMapTargets 等首屏完成并留出渲染时间后再请求全局缓存。
  const scheduleGlobalWorldMapTargets = useCallback((playerId: string, parentRequestId: number) => {
    if (globalLoadTimerRef.current !== null) window.clearTimeout(globalLoadTimerRef.current)
    globalLoadTimerRef.current = window.setTimeout(() => {
      globalLoadTimerRef.current = null
      void loadGlobalWorldMapTargets(playerId, parentRequestId)
    }, 1200)
  }, [loadGlobalWorldMapTargets])

  // loadAuxiliaryWorldMapData 后台读取行军和增援，不再阻塞近景地图首屏。
  const loadAuxiliaryWorldMapData = useCallback(async (playerId: string, parentRequestId: number) => {
    const requestId = auxiliaryLoadRequestRef.current + 1
    auxiliaryLoadRequestRef.current = requestId
    try {
      const [marchResult, sentReinforcementResult, receivedReinforcementResult] = await Promise.all([
        gameApi.listPvpMarches(playerId),
        gameApi.listSentReinforcements(playerId),
        gameApi.listReceivedReinforcements(playerId),
      ])
      if (requestId !== auxiliaryLoadRequestRef.current) return
      if (parentRequestId !== loadRequestRef.current) return
      if (loadedPlayerRef.current !== playerId) return
      setMarches(marchResult.items ?? [])
      setReinforcements(mergeWorldMapReinforcements(sentReinforcementResult.items ?? [], receivedReinforcementResult.items ?? []))
    } catch {
      // 行军和增援状态可稍后刷新，失败时不阻塞地图本体。
    }
  }, [])

  const load = useCallback(async (silent = false) => {
    if (!activePlayerId) {
      globalLoadRequestRef.current += 1
      auxiliaryLoadRequestRef.current += 1
      if (globalLoadTimerRef.current !== null) {
        window.clearTimeout(globalLoadTimerRef.current)
        globalLoadTimerRef.current = null
      }
      hasLoadedMapRef.current = false
      loadedPlayerRef.current = null
      setTargets([])
      setTargetView(null)
      setMarches([])
      setReinforcements([])
      setMapServerTime('')
      setLoadError('')
      setRefreshing(false)
      if (!silent) setLoading(false)
      return
    }
    const requestId = loadRequestRef.current + 1
    loadRequestRef.current = requestId
    globalLoadRequestRef.current += 1
    auxiliaryLoadRequestRef.current += 1
    if (globalLoadTimerRef.current !== null) {
      window.clearTimeout(globalLoadTimerRef.current)
      globalLoadTimerRef.current = null
    }
    const switchingPlayer = loadedPlayerRef.current !== activePlayerId
    if (!silent) {
      if (switchingPlayer) {
        hasLoadedMapRef.current = false
        setTargets([])
        setTargetView(null)
        setMarches([])
        setReinforcements([])
        setMapServerTime('')
        setLoading(true)
      } else if (hasLoadedMapRef.current) {
        setRefreshing(true)
      } else {
        setLoading(true)
      }
    }
    if (!silent) setLoadError('')
    try {
      const targetResult = await gameApi.getWorldMapView(activePlayerId, { radius: DEFAULT_VIEW_RADIUS })
      if (requestId !== loadRequestRef.current) return
      const shouldSyncInitialCoordinate = switchingPlayer || !hasLoadedMapRef.current
      applyWorldMapResult(targetResult, shouldSyncInitialCoordinate)
      hasLoadedMapRef.current = true
      loadedPlayerRef.current = activePlayerId
      void loadAuxiliaryWorldMapData(activePlayerId, requestId)
      scheduleGlobalWorldMapTargets(activePlayerId, requestId)
    } catch (err) {
      if (requestId !== loadRequestRef.current) return
      const message = err instanceof Error ? err.message : '世界地图加载失败'
      setLoadError(message)
      if (!silent) toast.error(message)
    } finally {
      if (requestId === loadRequestRef.current && !silent) setLoading(false)
      if (requestId === loadRequestRef.current && !silent) setRefreshing(false)
    }
  }, [activePlayerId, applyWorldMapResult, loadAuxiliaryWorldMapData, scheduleGlobalWorldMapTargets])

  useEffect(() => {
    void load()
  }, [load])

  useEffect(() => {
    return () => {
      if (globalLoadTimerRef.current !== null) window.clearTimeout(globalLoadTimerRef.current)
    }
  }, [])

  useEffect(() => {
    setViewport({ radius: DEFAULT_VIEW_RADIUS })
    setFocusedTargetId(null)
    setSelectedEmptyCell(null)
    setCoordinateSearch({ x: '', y: '' })
    setSelectedMarchTarget(null)
    setSelectedReinforceTarget(null)
    setSelections({})
    setSelectedGeneralIds([])
    setBusyTarget(null)
  }, [activePlayerId])

  const activeMarches = useMemo(() => {
    const playerCityMarches = marches.filter((march) => march.status === 'marching' || march.status === 'returning' || march.status === 'resolving')
    const reinforcementMarches = buildWorldMapReinforcementMarches(reinforcements)
    return [...playerCityMarches, ...reinforcementMarches]
  }, [marches, reinforcements])
  const marchBadges = useMemo(() => buildWorldMapMarchBadges(activeMarches, activePlayerId), [activeMarches, activePlayerId])
  const marchSummary = useMemo(() => buildWorldMapMarchSummary(activeMarches, activePlayerId), [activeMarches, activePlayerId])

  useEffect(() => {
    if (!activePlayerId || activeMarches.length === 0) return
    const timer = window.setInterval(() => {
      if (autoRefreshingRef.current) return
      const nowMs = Date.now()
      const hasDueMarch = activeMarches.some((march) => isMarchDue(march, nowMs))
      if (!hasDueMarch) return
      if (nowMs - lastAutoRefreshAtRef.current < 3000) return
      autoRefreshingRef.current = true
      lastAutoRefreshAtRef.current = nowMs
      void load(true).finally(() => {
        autoRefreshingRef.current = false
        void useGameStore.getState().loadMilitaryView()
      })
    }, 1000)
    return () => window.clearInterval(timer)
  }, [activeMarches, activePlayerId, load])

  // factionUnits 保持稳定引用，避免地图交互触发无关的行军预览重算。
  const factionUnits = useMemo(() => units?.[faction] ?? {}, [faction, units])
  const availableArmy = useMemo(() => army.filter((unit) => unit.amount > 0), [army])
  const armyAmountByType = useMemo(() => {
    return Object.fromEntries(army.map((unit) => [unit.unitType, unit.amount])) as Record<string, number>
  }, [army])
  const totalSelected = Object.values(selections).reduce((sum, amount) => sum + amount, 0)
  const focusedTarget = useMemo(() => {
    const target = targets.find((item) => item.playerId === focusedTargetId) ?? null
    if (!target || !isWorldMapRelationVisible(target.relation, relationFilters)) return null
    return target
  }, [focusedTargetId, relationFilters, targets])
  const mapView = useMemo(() => {
    if (!targetView) return null
    return {
      ...targetView,
      centerX: viewport.centerX ?? targetView.centerX,
      centerY: viewport.centerY ?? targetView.centerY,
      radius: viewport.radius,
    }
  }, [targetView, viewport])
  const visibleTargets = useMemo(() => {
    const centerX = viewport.centerX ?? targetView?.centerX ?? 0
    const centerY = viewport.centerY ?? targetView?.centerY ?? 0
    return filterWorldMapTargetsInViewport(targets, { x: centerX, y: centerY }, viewport.radius, relationFilters, targetView?.worldSize)
  }, [relationFilters, targetView, targets, viewport])
  const viewportTargets = useMemo(() => {
    const centerX = viewport.centerX ?? targetView?.centerX ?? 0
    const centerY = viewport.centerY ?? targetView?.centerY ?? 0
    return filterWorldMapTargetsInViewport(targets, { x: centerX, y: centerY }, viewport.radius, { self: true, ally: true, other: true }, targetView?.worldSize)
  }, [targetView, targets, viewport])
  const overviewTargets = useMemo(() => {
    return targets.filter((target) => isWorldMapRelationVisible(target.relation, relationFilters))
  }, [relationFilters, targets])
  const hiddenInViewportCount = viewportTargets.length - visibleTargets.length
  const hiddenByFilterCount = targets.length - overviewTargets.length
  const nearestTargets = useMemo(() => buildNearestWorldMapTargets(overviewTargets, 5), [overviewTargets])
  const relationCounts = useMemo(() => buildWorldMapRelationCounts(targets), [targets])
  const factionCounts = useMemo(() => buildWorldMapFactionCounts(targets), [targets])
  const syncTimeLabel = useMemo(() => formatWorldMapSyncTime(mapServerTime), [mapServerTime])
  const worldWidth = targetView?.worldWidth ?? targetView?.worldSize ?? 0
  const worldHeight = targetView?.worldHeight ?? targetView?.worldSize ?? 0
  const worldCellCapacity = worldWidth > 0 && worldHeight > 0 ? worldWidth * worldHeight : 0
  const worldOccupiedCells = useMemo(() => {
    if (!targetView) return targets.length
    const hasSelfTarget = targets.some((target) => target.relation === 'self' || (target.position.x === targetView.self.x && target.position.y === targetView.self.y))
    return hasSelfTarget ? targets.length : targets.length + 1
  }, [targetView, targets])
  const worldOccupancyRate = worldCellCapacity > 0 ? worldOccupiedCells / worldCellCapacity : 0
  const resetRelationFilters = useCallback(() => {
    setRelationFilters({ self: true, ally: true, other: true })
  }, [])
  const reinforcePreview = useMemo(() => {
    if (!selectedReinforceTarget) return null
    const speed = slowestSelectedUnitSpeed(selections, factionUnits)
    const metrics = buildWorldMapTargetMetrics(viewSelf(targetView), selectedReinforceTarget.position, speed)
    return {
      speed,
      seconds: metrics.seconds,
    }
  }, [factionUnits, selectedReinforceTarget, selections, targetView])
  const marchPreview = useMemo(() => {
    if (!selectedMarchTarget) return null
    const speed = slowestSelectedUnitSpeed(selections, factionUnits)
    const metrics = buildWorldMapTargetMetrics(viewSelf(targetView), selectedMarchTarget.position, speed)
    return {
      speed,
      seconds: metrics.seconds,
    }
  }, [factionUnits, selectedMarchTarget, selections, targetView])
  const availableGenerals = useMemo(() => {
    const busy = new Set(generalAssignments.filter((item) => item.id !== 'main' && item.slot !== 'main').map((item) => item.generalId))
    return generals.filter((general) => !busy.has(general.id))
  }, [generalAssignments, generals])

  // refreshTargetDetail 刷新单个玩家城池详情，并合并回全量缓存。
  const refreshTargetDetail = useCallback(async (playerId: string) => {
    if (!activePlayerId || !targetView) return
    const requestId = targetDetailRequestRef.current + 1
    targetDetailRequestRef.current = requestId
    try {
      const detail = await gameApi.getWorldMapPlayerCityTarget(activePlayerId, playerId)
      if (requestId !== targetDetailRequestRef.current) return
      const nextTarget = worldTargetToPvpTarget(detail, targetView.self.worldId, targetView.self)
      setTargets((prev) => mergeWorldMapTargetCache(prev, nextTarget))
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '刷新城池状态失败')
    }
  }, [activePlayerId, targetView])

  // handleFocusTarget 选中玩家城池，清理旧空地高亮，并刷新目标详情。
  const handleFocusTarget = useCallback((playerId: string) => {
    const target = targets.find((item) => item.playerId === playerId)
    if (target) setCoordinateSearch({ x: String(target.position.x), y: String(target.position.y) })
    setFocusedTargetId(playerId)
    setSelectedEmptyCell(null)
    void refreshTargetDetail(playerId)
  }, [refreshTargetDetail, targets])

  // getUnitName 获取当前阵营兵种名。
  const getUnitName = (unitType: string) => factionUnits[unitType]?.name ?? unitType

  // openMarchSelector 打开世界地图出征兵力选择面板。
  const openMarchSelector = useCallback((target: PvpTargetSummary, mode: 'attack' | 'plunder') => {
    setSelectedReinforceTarget(null)
    setSelectedMarchTarget(target)
    setSelectedMarchMode(mode)
    setSelections({})
    setSelectedGeneralIds([])
  }, [])

  // openReinforceSelector 打开地图内增援兵力选择弹窗。
  const openReinforceSelector = useCallback((target: PvpTargetSummary) => {
    setSelectedMarchTarget(null)
    setSelectedReinforceTarget(target)
    setSelections({})
    setSelectedGeneralIds([])
  }, [])

  // handleSelectionChange 修改某个兵种出征数量。
  const handleSelectionChange = useCallback((unitType: string, value: string) => {
    const max = armyAmountByType[unitType] ?? 0
    const next = Math.min(max, Math.max(0, Number.parseInt(value, 10) || 0))
    setSelections((prev) => ({ ...prev, [unitType]: next }))
  }, [armyAmountByType])

  // handleSelectAll 选择某个兵种的全部可用数量。
  const handleSelectAll = useCallback((unitType: string) => {
    const max = armyAmountByType[unitType] ?? 0
    setSelections((prev) => ({ ...prev, [unitType]: max }))
  }, [armyAmountByType])

  // handleGeneralToggle 切换本次世界地图出征携带武将。
  const handleGeneralToggle = useCallback((generalId: string) => {
    setSelectedGeneralIds((prev) => prev.includes(generalId) ? [] : [generalId])
  }, [])

  // handleScout 派出玩家侦查行军。
  const handleScout = useCallback(async (target: PvpTargetSummary) => {
    if (!activePlayerId || busyTarget) return
    setBusyTarget(`${target.playerId}:scout`)
    try {
      const result = await gameApi.scoutPvpTarget(activePlayerId, target.playerId)
      useGameStore.getState().patchState({ army: result.army, serverTime: result.serverTime })
      setMarches((prev) => [result.march, ...prev])
      toast.success(`侦查队已出发，预计 ${formatDuration(result.march.durationSeconds)} 后抵达。`)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '侦查失败')
    } finally {
      setBusyTarget(null)
    }
  }, [activePlayerId, busyTarget])

  // handleMarch 执行玩家攻击或掠夺，成功后创建行军。
  const handleMarch = async () => {
    const target = selectedMarchTarget
    const mode = selectedMarchMode
    if (!activePlayerId || busyTarget) return
    if (!target) return
    const troops = Object.fromEntries(Object.entries(selections).filter(([, amount]) => amount > 0))
    if (Object.keys(troops).length === 0) {
      toast.info('请先选择出征兵力')
      return
    }
    setBusyTarget(`${target.playerId}:${mode}`)
    try {
      const result = await gameApi.startPvpAttack(activePlayerId, target.playerId, mode, troops, selectedGeneralIds)
      useGameStore.getState().patchState({
        army: result.army,
        generals: result.generals,
        generalAssignments: result.generalAssignments,
        serverTime: result.serverTime,
      })
      setMarches((prev) => [result.march, ...prev])
      setSelectedMarchTarget(null)
      setSelections({})
      setSelectedGeneralIds([])
      void useGameStore.getState().loadMilitaryView()
      void load(true)
      toast.success(`已向 ${target.nickname} 发起${mode === 'plunder' ? '掠夺' : '攻击'}行军。`)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '发起世界地图行军失败')
    } finally {
      setBusyTarget(null)
    }
  }

  // handleReinforce 执行地图内增援派兵。
  const handleReinforce = async () => {
    const target = selectedReinforceTarget
    if (!activePlayerId || busyTarget) return
    if (!target) return
    const troops = Object.fromEntries(Object.entries(selections).filter(([, amount]) => amount > 0))
    if (Object.keys(troops).length === 0) {
      toast.info('请先选择增援兵力')
      return
    }
    setBusyTarget(`${target.playerId}:reinforce`)
    try {
      const result = await gameApi.sendReinforcement(activePlayerId, target.playerId, troops, selectedGeneralIds)
      if (result.patch) {
        useGameStore.getState().patchState(result.patch)
      }
      setReinforcements((prev) => mergeWorldMapReinforcements(prev, [result.reinforcement]))
      setSelectedReinforceTarget(null)
      setSelections({})
      setSelectedGeneralIds([])
      void useGameStore.getState().loadMilitaryView()
      void load(true)
      toast.success(`已向 ${target.nickname} 派出增援。`)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '发起增援失败')
    } finally {
      setBusyTarget(null)
    }
  }

  // panViewport 移动世界地图视野。
  const panViewport = useCallback((dx: number, dy: number) => {
    setViewport((prev) => {
      const center = moveWorldMapCenter({
        x: prev.centerX ?? targetView?.centerX ?? targetView?.self.x ?? 0,
        y: prev.centerY ?? targetView?.centerY ?? targetView?.self.y ?? 0,
      }, dx, dy, targetView?.worldSize ?? 100)
      return {
        radius: prev.radius,
        centerX: center.x,
        centerY: center.y,
      }
    })
  }, [targetView])

  // zoomViewport 缩放世界地图视野。
  const zoomViewport = useCallback((delta: number) => {
    setViewport((prev) => ({
      centerX: prev.centerX,
      centerY: prev.centerY,
      radius: clampWorldMapRadius(prev.radius + delta),
    }))
  }, [])

  // setViewportRadius 切换地图视野预设，中心坐标保持不变。
  const setViewportRadius = useCallback((radius: number) => {
    setViewport((prev) => ({
      centerX: prev.centerX,
      centerY: prev.centerY,
      radius: clampWorldMapRadius(radius),
    }))
  }, [])

  // focusSelf 将视野定位回自己的城池。
  const focusSelf = useCallback(() => {
    if (!targetView?.self) return
    const selfTarget = findWorldMapTargetAtCell(targets, targetView.self)
    setRelationFilters((prev) => ({ ...prev, self: true }))
    setCoordinateSearch({ x: String(targetView.self.x), y: String(targetView.self.y) })
    setFocusedTargetId(selfTarget?.playerId ?? null)
    setSelectedEmptyCell(null)
    setViewport((prev) => ({ ...prev, centerX: targetView.self.x, centerY: targetView.self.y }))
  }, [targetView, targets])

  // selectMapPosition 统一处理坐标选中，避免隐藏城池被误判为空地。
  const selectMapPosition = useCallback((position: { x: number; y: number }, hiddenMessage: string) => {
    const target = findVisibleWorldMapTargetAtCell(targets, position, relationFilters)
    const hiddenTarget = target ? null : findWorldMapTargetAtCell(targets, position)
    if (target) {
      handleFocusTarget(target.playerId)
    } else if (hiddenTarget) {
      setRelationFilters((prev) => ({ ...prev, [hiddenTarget.relation ?? 'other']: true }))
      handleFocusTarget(hiddenTarget.playerId)
      toast.info(hiddenMessage)
    } else {
      setFocusedTargetId(null)
      setSelectedEmptyCell(position)
    }
  }, [handleFocusTarget, relationFilters, targets])

  // handleCoordinateSearch 移动到输入坐标，并选中该格城池或空地。
  const handleCoordinateSearch = useCallback(() => {
    const result = parseWorldMapCoordinateSearch(coordinateSearch, targetView?.worldSize ?? 100)
    if (!result.position) {
      toast.info(result.error)
      return
    }
    const center = result.position
    setCoordinateSearch({ x: String(center.x), y: String(center.y) })
    setViewport((prev) => ({ ...prev, centerX: center.x, centerY: center.y }))
    selectMapPosition(center, '该坐标有玩家城池，已自动显示并选中')
  }, [coordinateSearch, selectMapPosition, targetView])

  // handleSelectMapCell 选中地图格子。
  const handleSelectMapCell = useCallback((x: number, y: number) => {
    setCoordinateSearch({ x: String(x), y: String(y) })
    selectMapPosition({ x, y }, '该格有玩家城池，已自动显示并选中')
  }, [selectMapPosition])

  // handleOverviewJump 从全图概览跳转到指定坐标，并同步选中目标或空地。
  const handleOverviewJump = useCallback((position: { x: number; y: number }) => {
    setCoordinateSearch({ x: String(position.x), y: String(position.y) })
    setViewport((prev) => ({ ...prev, centerX: position.x, centerY: position.y }))
    selectMapPosition(position, '目标坐标有玩家城池，已自动显示并选中')
  }, [selectMapPosition])

  // clearMapSelection 清除当前地图选中的城池或空地。
  const clearMapSelection = useCallback(() => {
    setFocusedTargetId(null)
    setSelectedEmptyCell(null)
  }, [])

  if (loading) {
    return (
      <div className="flex items-center justify-center py-16">
        <LoaderCircle size={16} className="animate-spin text-[var(--color-accent)]" />
        <span className="ml-2 text-xs text-[var(--color-text-muted)]">正在读取世界地图...</span>
      </div>
    )
  }

  if (!activePlayerId) {
    return (
      <div className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] px-4 py-12 text-center">
        <div className="text-sm font-black text-[var(--color-text-primary)]">请选择玩家存档</div>
        <p className="mt-2 text-xs text-[var(--color-text-muted)]">选择存档后即可查看自己的世界地图坐标和附近玩家城池。</p>
      </div>
    )
  }

  if (loadError && !targetView) {
    return (
      <div className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] px-4 py-12 text-center">
        <div className="text-sm font-black text-[var(--color-text-primary)]">世界地图加载失败</div>
        <p className="mt-2 text-xs text-[var(--color-text-muted)]">{loadError}</p>
        <button
          type="button"
          onClick={() => void load()}
          className="mt-4 inline-flex items-center gap-1.5 rounded-lg bg-[var(--color-accent)] px-4 py-2 text-xs font-bold text-white"
        >
          <RefreshCw size={12} />
          重新加载
        </button>
      </div>
    )
  }

  return (
    <div className="space-y-4 pb-[52vh] xl:pb-0">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex min-w-0 flex-wrap items-center gap-2 text-xs text-[var(--color-text-muted)]">
          <span>当前视野 {visibleTargets.length}/{viewportTargets.length} 个玩家城池</span>
          {hiddenByFilterCount > 0 && (
            <span className="inline-flex items-center gap-1 font-bold text-amber-600">
              全图已隐藏 {hiddenByFilterCount} 个
              <button
                type="button"
                onClick={resetRelationFilters}
                className="inline-flex h-6 items-center gap-1 rounded border border-amber-500/30 bg-amber-500/10 px-1.5 text-[10px] text-amber-700 hover:bg-amber-500/20"
              >
                <Eye size={10} />
                显示全部
              </button>
            </span>
          )}
          {relationFilters.self === false && <span className="font-bold text-sky-600">我的城池已被筛选隐藏</span>}
          {targetView && <span>我的坐标 ({targetView.self.x}, {targetView.self.y})</span>}
          {targetView && <span>世界 {worldWidth} x {worldHeight}</span>}
          {worldCellCapacity > 0 && <span>占用 {worldOccupiedCells}/{worldCellCapacity} 格</span>}
          {worldOccupancyRate >= 0.8 && <span className="font-bold text-amber-600">接近扩容线</span>}
          {syncTimeLabel && <span>已同步 {syncTimeLabel}</span>}
          {activeMarches.length > 0 && (
            <span className="inline-flex flex-wrap items-center gap-1 rounded bg-[var(--color-surface-dim)] px-2 py-1 font-bold text-[var(--color-text-secondary)]">
              行军
              <b className="text-violet-600">出 {marchSummary.outgoing}</b>
              <b className="text-violet-600">返 {marchSummary.returning}</b>
              <b className="text-violet-600">袭 {marchSummary.incoming}</b>
              <b className="text-violet-600">结 {marchSummary.resolving}</b>
            </span>
          )}
        </div>
        <button
          type="button"
          onClick={() => void load()}
          disabled={refreshing}
          title="刷新世界地图玩家城池、行军和增援状态"
          aria-label="刷新世界地图玩家城池、行军和增援状态"
          className="inline-flex items-center gap-1.5 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-1.5 text-xs font-medium text-[var(--color-text-secondary)] cursor-pointer hover:text-[var(--color-accent)]"
        >
          <RefreshCw size={12} className={refreshing ? 'animate-spin' : ''} />
          {refreshing ? '刷新中' : '刷新'}
        </button>
      </div>

      <div className="flex flex-wrap items-center gap-2 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2">
        <WorldMapFilters filters={relationFilters} counts={relationCounts} onChange={setRelationFilters} />
        <div className="ml-0 flex w-full flex-wrap items-center gap-3 lg:ml-auto lg:w-auto">
          <WorldMapLegend counts={factionCounts} />
          <WorldMapCoordinateSearch value={coordinateSearch} worldSize={targetView?.worldSize ?? 100} onChange={setCoordinateSearch} onSearch={handleCoordinateSearch} onFocusSelf={focusSelf} />
        </div>
      </div>

      {nearestTargets.length > 0 && (
        <div className="flex flex-wrap items-center gap-2 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-xs">
          <span className="font-black text-[var(--color-text-primary)]">最近城池</span>
          {nearestTargets.map((target) => (
            <button
              key={target.playerId}
              type="button"
              onClick={() => handleOverviewJump(target.position)}
              className="inline-flex max-w-full items-center gap-1.5 rounded-lg bg-[var(--color-surface-dim)] px-2 py-1 font-bold text-[var(--color-text-secondary)] hover:text-[var(--color-accent)]"
              title={`跳转到 ${target.nickname} (${target.position.x}, ${target.position.y})，速度1预计行军 ${formatDuration(target.reinforceSeconds)}`}
              aria-label={`跳转到${target.nickname}，${target.direction || directionFrom(viewSelf(targetView), target.position)}，距离${target.distance}格，速度1预计行军${formatDuration(target.reinforceSeconds)}`}
            >
              <span className={`rounded px-1 py-0.5 text-[9px] leading-none ${worldMapRelationBadgeClass(target.relation)}`}>{worldMapRelationBadge(target.relation)}</span>
              <span className="max-w-20 truncate">{target.nickname}</span>
              <span className="text-[10px] text-[var(--color-text-muted)]">{target.direction || directionFrom(viewSelf(targetView), target.position)}</span>
              <span className="text-[10px] text-[var(--color-text-muted)]">{target.distance}格</span>
              <span className="text-[10px] text-emerald-600">{formatDuration(target.reinforceSeconds)}</span>
              <span className="text-[10px] text-[var(--color-text-muted)]">({target.position.x},{target.position.y})</span>
            </button>
          ))}
        </div>
      )}

      <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_360px]">
        {mapView && (
          <WorldMapGrid
            view={mapView}
            targets={visibleTargets}
            overviewTargets={overviewTargets}
            hiddenInViewportCount={hiddenInViewportCount}
            focusedTargetId={focusedTargetId}
            focusedTargetPosition={focusedTarget?.position ?? null}
            selectedCell={selectedEmptyCell}
            marchBadges={marchBadges}
            showSelf={relationFilters.self !== false}
            onFocusTarget={handleFocusTarget}
            onFocusSelf={focusSelf}
            onPan={panViewport}
            onZoom={zoomViewport}
            onSetRadius={setViewportRadius}
            onJump={handleOverviewJump}
            onSelectCell={handleSelectMapCell}
            onClearSelection={clearMapSelection}
          />
        )}
        <div className="fixed inset-x-0 bottom-0 z-[8000] max-h-[48vh] overflow-y-auto border-t border-[var(--color-border)] bg-[var(--color-surface)] p-3 shadow-[0_-10px_30px_rgba(15,23,42,0.18)] xl:static xl:z-auto xl:max-h-none xl:overflow-visible xl:border-t-0 xl:bg-transparent xl:p-0 xl:shadow-none">
          <WorldMapTargetPanel
            target={focusedTarget}
            emptyCell={selectedEmptyCell}
            selfPosition={mapView?.self ?? null}
            busyTarget={busyTarget}
            onScout={handleScout}
            onReinforce={openReinforceSelector}
            onMarch={openMarchSelector}
            onClearSelection={clearMapSelection}
          />
        </div>
      </div>

      {targets.length === 0 && (
        <div className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] py-12 text-center text-sm text-[var(--color-text-muted)]">
          当前视野暂无可展示玩家城池
        </div>
      )}

      {selectedReinforceTarget && (
        <TroopSelectionModal
          title={`增援 · ${selectedReinforceTarget.nickname}`}
          travelPreview={buildTravelPreviewItems(selectedReinforceTarget, reinforcePreview?.speed ?? 1, reinforcePreview?.seconds ?? selectedReinforceTarget.reinforceSeconds, viewSelf(targetView))}
          closeLabel="关闭"
          confirmLabel="确认增援"
          totalSelected={totalSelected}
          availableArmy={availableArmy}
          availableGenerals={availableGenerals}
          selectedGeneralIds={selectedGeneralIds}
          selections={selections}
          getUnitName={getUnitName}
          onClose={() => setSelectedReinforceTarget(null)}
          onGeneralToggle={handleGeneralToggle}
          onSelectionChange={handleSelectionChange}
          onSelectAll={handleSelectAll}
          onConfirm={() => void handleReinforce()}
          disabled={totalSelected <= 0 || busyTarget !== null}
        />
      )}
      {selectedMarchTarget && (
        <TroopSelectionModal
          title={`${selectedMarchMode === 'plunder' ? '掠夺' : '攻击'} · ${selectedMarchTarget.nickname}`}
          travelPreview={buildTravelPreviewItems(selectedMarchTarget, marchPreview?.speed ?? 1, marchPreview?.seconds ?? selectedMarchTarget.reinforceSeconds, viewSelf(targetView))}
          closeLabel="关闭"
          confirmLabel="确认出征"
          totalSelected={totalSelected}
          availableArmy={availableArmy}
          availableGenerals={availableGenerals}
          selectedGeneralIds={selectedGeneralIds}
          selections={selections}
          getUnitName={getUnitName}
          onClose={() => setSelectedMarchTarget(null)}
          onGeneralToggle={handleGeneralToggle}
          onSelectionChange={handleSelectionChange}
          onSelectAll={handleSelectAll}
          onConfirm={() => void handleMarch()}
          disabled={totalSelected <= 0 || busyTarget !== null}
        />
      )}
    </div>
  )
}

interface TravelPreviewItem {
  label: string
  value: string
  tone?: 'accent'
}

const TroopSelectionModal: FC<{
  title: string
  travelPreview: TravelPreviewItem[]
  closeLabel: string
  confirmLabel: string
  totalSelected: number
  availableArmy: ArmyUnit[]
  availableGenerals: General[]
  selectedGeneralIds: string[]
  selections: Record<string, number>
  getUnitName: (unitType: string) => string
  onClose: () => void
  onGeneralToggle: (generalId: string) => void
  onSelectionChange: (unitType: string, value: string) => void
  onSelectAll: (unitType: string) => void
  onConfirm: () => void
  disabled: boolean
}> = ({
  title,
  travelPreview,
  closeLabel,
  confirmLabel,
  totalSelected,
  availableArmy,
  availableGenerals,
  selectedGeneralIds,
  selections,
  getUnitName,
  onClose,
  onGeneralToggle,
  onSelectionChange,
  onSelectAll,
  onConfirm,
  disabled,
}) => (
  <div className="fixed inset-0 z-[9000] flex items-end justify-center bg-slate-950/45 p-0 sm:items-center sm:p-4">
    <div className="w-full max-w-md rounded-t-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4 shadow-xl sm:rounded-2xl">
      <div className="flex items-center justify-between gap-3">
        <div className="min-w-0">
          <h3 className="text-sm font-bold text-[var(--color-text-primary)]">{title}</h3>
        </div>
        <button
          type="button"
          onClick={onClose}
          className="rounded-lg border border-[var(--color-border)] px-2 py-1 text-xs text-[var(--color-text-secondary)]"
        >
          {closeLabel}
        </button>
      </div>
      <div className="mt-3 grid grid-cols-2 gap-1.5 sm:grid-cols-5">
        {travelPreview.map((item) => (
          <div key={item.label} className={`rounded-xl px-2 py-1.5 ${item.tone === 'accent' ? 'bg-[var(--color-accent-light)] text-[var(--color-accent)]' : 'bg-[var(--color-surface-dim)] text-[var(--color-text-secondary)]'}`}>
            <div className="text-[9px] font-bold text-[var(--color-text-muted)]">{item.label}</div>
            <div className="mt-0.5 truncate text-xs font-black">{item.value}</div>
          </div>
        ))}
      </div>
      <div className="mt-3 max-h-72 space-y-3 overflow-y-auto">
        <div className="rounded-xl bg-[var(--color-surface-dim)] px-3 py-2">
          <div className="mb-2 flex items-center justify-between">
            <span className="text-xs font-bold text-[var(--color-text-primary)]">随军武将</span>
            <span className="text-[10px] text-[var(--color-text-muted)]">{selectedGeneralIds.length > 0 ? `已带 ${selectedGeneralIds.length}` : '不带将'}</span>
          </div>
          <div className="grid gap-1.5">
            {availableGenerals.map((general) => {
              const selected = selectedGeneralIds.includes(general.id)
              return (
                <button
                  key={general.id}
                  type="button"
                  onClick={() => onGeneralToggle(general.id)}
                  className={`flex h-9 items-center justify-between rounded-lg border px-2 text-left text-xs transition-colors ${selected ? 'border-amber-400 bg-amber-400/15 text-amber-700' : 'border-[var(--color-border)] bg-white/60 text-[var(--color-text-secondary)] dark:bg-white/5'}`}
                >
                  <span className="font-bold">{general.name}</span>
                  <span className="text-[10px]">Lv.{general.level}</span>
                </button>
              )
            })}
            {availableGenerals.length === 0 && (
              <p className="py-2 text-center text-xs text-[var(--color-text-muted)]">暂无可随军武将</p>
            )}
          </div>
        </div>
        {availableArmy.map((unit) => (
          <div key={unit.unitType} className="flex items-center justify-between gap-3 rounded-xl bg-[var(--color-surface-dim)] px-3 py-2">
            <span className="min-w-0 flex-1 truncate text-xs font-semibold text-[var(--color-text-primary)]">
              {getUnitName(unit.unitType)}
              <span className="ml-1 text-[var(--color-text-muted)]">({unit.amount.toLocaleString()})</span>
            </span>
            <div className="flex items-center gap-1.5">
              <input
                type="number"
                min={0}
                max={unit.amount}
                value={selections[unit.unitType] ?? 0}
                onChange={(event) => onSelectionChange(unit.unitType, event.target.value)}
                className="h-8 w-20 rounded-lg border border-[var(--color-border)] bg-white px-2 text-center text-xs font-bold text-[var(--color-text-primary)] outline-none dark:bg-slate-900"
              />
              <button
                type="button"
                onClick={() => onSelectAll(unit.unitType)}
                className="h-8 rounded-lg px-2 text-[10px] font-bold text-[var(--color-accent)]"
              >
                全部
              </button>
            </div>
          </div>
        ))}
        {availableArmy.length === 0 && (
          <p className="py-8 text-center text-xs text-[var(--color-text-muted)]">当前没有可派出兵力</p>
        )}
      </div>
      <div className="mt-4 flex items-center justify-between gap-3">
        <span className="text-xs text-[var(--color-text-muted)]">已选 <b className="text-[var(--color-accent)]">{totalSelected.toLocaleString()}</b></span>
        <button
          type="button"
          onClick={onConfirm}
          disabled={disabled}
          className="rounded-xl bg-[var(--color-accent)] px-5 py-2 text-xs font-bold text-white disabled:opacity-50"
        >
          {confirmLabel}
        </button>
      </div>
    </div>
  </div>
)

// isMarchDue 判断行军或返程是否已经到期，需要触发刷新结算。
function isMarchDue(march: { status: string; returnsAt?: string; arrivesAt?: string }, nowMs: number) {
  const targetTime = march.status === 'returning' ? march.returnsAt : march.arrivesAt
  if (!targetTime) return march.status === 'resolving'
  const targetMs = new Date(targetTime).getTime()
  if (Number.isNaN(targetMs)) return false
  return targetMs <= nowMs
}

// mergeWorldMapReinforcements 合并派出和收到的增援记录，避免地图角标重复统计。
function mergeWorldMapReinforcements(...groups: Reinforcement[][]) {
  const result: Reinforcement[] = []
  const seen = new Set<string>()
  for (const group of groups) {
    for (const record of group) {
      if (seen.has(record.reinforcementId)) continue
      seen.add(record.reinforcementId)
      result.push(record)
    }
  }
  return result
}

// buildTravelPreviewItems 构建出征/增援弹窗里的地图行军预览。
function buildTravelPreviewItems(target: PvpTargetSummary, speed: number, seconds: number, self: PvpWorldPosition): TravelPreviewItem[] {
  const metrics = buildWorldMapTargetMetrics(self, target.position, speed)
  return [
    { label: '坐标', value: `(${target.position.x}, ${target.position.y})` },
    { label: '方位', value: metrics.direction },
    { label: '距离', value: `${metrics.distance}格` },
    { label: '最慢速度', value: `${speed}` },
    { label: '预计行军', value: formatDuration(seconds), tone: 'accent' },
  ]
}

// viewSelf 返回当前地图视图中的自己坐标，视图未加载时提供安全默认值。
function viewSelf(view: PvpTargetsResponse | null): PvpWorldPosition {
  return view?.self ?? { worldId: 'world_1', x: 0, y: 0, regionId: 0 }
}

// slowestSelectedUnitSpeed 获取当前选择部队里的最慢兵种速度。
function slowestSelectedUnitSpeed(selections: Record<string, number>, factionUnits: Record<string, { stats?: Record<string, number> }>) {
  let slowest = 0
  for (const [unitType, amount] of Object.entries(selections)) {
    if (amount <= 0) continue
    const speed = Math.max(1, Math.floor(factionUnits[unitType]?.stats?.speed ?? 1))
    if (slowest === 0 || speed < slowest) slowest = speed
  }
  return slowest > 0 ? slowest : 1
}

// worldTargetToPvpTarget 把世界地图玩家城池目标转换为现有操作组件可复用的数据结构。
function worldTargetToPvpTarget(target: WorldMapTarget, worldId: string, self: PvpWorldPosition): PvpTargetSummary {
  const position = { worldId, x: target.x, y: target.y, regionId: 0 }
  const metrics = buildWorldMapTargetMetrics(self, position)
  return {
    playerId: target.playerId ?? target.targetId,
    nickname: target.name,
    faction: target.faction,
    position,
    distance: metrics.distance,
    direction: metrics.direction,
    reinforceSeconds: metrics.seconds,
    totalArmy: 0,
    buildingLevel: target.level,
    relation: target.relation,
    status: target.status,
    canScout: target.canScout,
    canPlunder: target.canPlunder,
    canAttack: target.canAttack,
    canReinforce: target.canReinforce,
    protected: target.status === 'protected' || target.status === 'truce',
    reason: target.reason,
    scoutReason: target.scoutReason,
    attackReason: target.attackReason,
    plunderReason: target.plunderReason,
    reinforceReason: target.reinforceReason,
  }
}

export default WorldMapTab
