// 本文件实现地图页的 PVP 玩家目标列表和快捷操作。
import { memo, useCallback, useEffect, useMemo, useRef, useState, type FC, type ReactNode } from 'react'
import { LoaderCircle, LocateFixed, MapPin, Minus, Plus, RefreshCw, RotateCcw, Search, ShieldAlert, ShieldPlus, Swords, Zap } from 'lucide-react'
import { gameApi } from '@/api/game'
import { toast } from '@/components/ui'
import { useGameStore } from '@/store/gameStore'
import { useConfigStore } from '@/store/configStore'
import { FACTION_COLORS, FACTION_LABELS } from '@/utils/faction'
import type { BattleReport, PvpMarch, PvpRankingResponse, PvpStateResponse, PvpTargetSummary, PvpTargetsResponse, PvpWorldPosition } from '@/types/game'
import ScoutResultModal from './ScoutResultModal'

const PVP_STATUS_LABELS: Record<PvpMarch['status'], string> = {
  marching: '行军中',
  returning: '返回中',
  resolving: '结算中',
  resolved: '已结算',
  recalled: '已召回',
  cancelled: '已取消',
  failed: '异常',
}

// PlayerTargetsTab 展示 PVP 玩家目标和行军状态。
const PlayerTargetsTab: FC = () => {
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const army = useGameStore((s) => s.state?.army ?? [])
  const faction = useGameStore((s) => s.state?.player.faction ?? 'wei')
  const units = useConfigStore((s) => s.units)
  const [targets, setTargets] = useState<PvpTargetSummary[]>([])
  const [targetView, setTargetView] = useState<PvpTargetsResponse | null>(null)
  const [marches, setMarches] = useState<PvpMarch[]>([])
  const [pvpState, setPvpState] = useState<PvpStateResponse | null>(null)
  const [rankings, setRankings] = useState<PvpRankingResponse | null>(null)
  const [viewport, setViewport] = useState<{ centerX?: number; centerY?: number; radius: number }>({ radius: 420 })
  const [focusedTargetId, setFocusedTargetId] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [busyTarget, setBusyTarget] = useState<string | null>(null)
  const [busyMarch, setBusyMarch] = useState<string | null>(null)
  const [scoutReport, setScoutReport] = useState<BattleReport | null>(null)
  const [selectedMarchTarget, setSelectedMarchTarget] = useState<PvpTargetSummary | null>(null)
  const [selectedMarchMode, setSelectedMarchMode] = useState<'attack' | 'plunder'>('attack')
  const [selections, setSelections] = useState<Record<string, number>>({})
  const autoRefreshingRef = useRef(false)
  const lastAutoRefreshAtRef = useRef(0)

  const load = useCallback(async (silent = false) => {
    if (!activePlayerId) return
    if (!silent) setLoading(true)
    try {
      const [targetResult, marchResult, stateResult, rankingResult] = await Promise.all([
        gameApi.listPvpTargets(activePlayerId, { ...viewport, limit: 80 }),
        gameApi.listPvpMarches(activePlayerId),
        gameApi.getPvpState(activePlayerId),
        gameApi.listPvpRankings(activePlayerId, 10),
      ])
      setTargets(targetResult.items ?? [])
      setTargetView(targetResult)
      setMarches(marchResult.items ?? [])
      setPvpState(stateResult)
      setRankings(rankingResult)
    } finally {
      if (!silent) setLoading(false)
    }
  }, [activePlayerId, viewport])

  useEffect(() => {
    void load()
  }, [load])

  const activeMarches = useMemo(() => {
    return marches.filter((march) => march.status === 'marching' || march.status === 'returning' || march.status === 'resolving')
  }, [marches])

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

  const factionUnits = units?.[faction] ?? {}
  const availableArmy = useMemo(() => army.filter((unit) => unit.amount > 0), [army])
  const armyAmountByType = useMemo(() => {
    return Object.fromEntries(army.map((unit) => [unit.unitType, unit.amount])) as Record<string, number>
  }, [army])
  const totalSelected = Object.values(selections).reduce((sum, amount) => sum + amount, 0)
  const focusedTarget = useMemo(() => targets.find((target) => target.playerId === focusedTargetId) ?? null, [focusedTargetId, targets])

  // getUnitName 获取当前阵营兵种名。
  const getUnitName = (unitType: string) => factionUnits[unitType]?.name ?? unitType

  // openMarchSelector 打开 PVP 出征兵力选择面板。
  const openMarchSelector = useCallback((target: PvpTargetSummary, mode: 'attack' | 'plunder') => {
    setSelectedMarchTarget(target)
    setSelectedMarchMode(mode)
    setSelections({})
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

  // handleScout 执行玩家侦查。
  const handleScout = useCallback(async (target: PvpTargetSummary) => {
    if (!activePlayerId || busyTarget) return
    setBusyTarget(`${target.playerId}:scout`)
    try {
      const result = await gameApi.scoutPvpTarget(activePlayerId, target.playerId)
      setScoutReport(result.battleReport)
      toast.success('侦查完成，可在军情查看战报。')
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
      const result = await gameApi.startPvpAttack(activePlayerId, target.playerId, mode, troops)
      useGameStore.getState().patchState({
        army: result.army,
        generals: result.generals,
        serverTime: result.serverTime,
      })
      setMarches((prev) => [result.march, ...prev])
      setSelectedMarchTarget(null)
      setSelections({})
      void useGameStore.getState().loadMilitaryView()
      toast.success(`已向 ${target.nickname} 发起${mode === 'plunder' ? '掠夺' : '攻击'}行军。`)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '发起 PVP 行军失败')
    } finally {
      setBusyTarget(null)
    }
  }

  // handleReinforce 提示玩家使用增援页定向派兵。
  const handleReinforce = useCallback((target: PvpTargetSummary) => {
    navigator.clipboard?.writeText(target.playerId).catch(() => undefined)
    toast.info(`已复制目标 ID：${target.playerId}，可到增援页派出队伍。`)
  }, [])

  // panViewport 移动 PVP 地图视野。
  const panViewport = useCallback((dx: number, dy: number) => {
    setViewport((prev) => ({
      radius: prev.radius,
      centerX: (targetView?.centerX ?? targetView?.self.x ?? prev.centerX ?? 1000) + dx,
      centerY: (targetView?.centerY ?? targetView?.self.y ?? prev.centerY ?? 1000) + dy,
    }))
  }, [targetView])

  // zoomViewport 缩放 PVP 地图视野。
  const zoomViewport = useCallback((delta: number) => {
    setViewport((prev) => ({
      centerX: targetView?.centerX ?? prev.centerX,
      centerY: targetView?.centerY ?? prev.centerY,
      radius: Math.max(160, Math.min(1000, prev.radius + delta)),
    }))
  }, [targetView])

  // focusSelf 将视野定位回自己的城池。
  const focusSelf = useCallback(() => {
    if (!targetView?.self) return
    setViewport((prev) => ({ ...prev, centerX: targetView.self.x, centerY: targetView.self.y }))
  }, [targetView])

  // handleRecallMarch 召回一条自己的 PVP 行军。
  const handleRecallMarch = async (march: PvpMarch) => {
    if (!activePlayerId || busyMarch) return
    setBusyMarch(march.id)
    try {
      const result = await gameApi.recallPvpMarch(activePlayerId, march.id)
      useGameStore.getState().patchState({
        ...(result.army ? { army: result.army } : {}),
        ...(result.generals ? { generals: result.generals } : {}),
        serverTime: result.serverTime,
      })
      setMarches((prev) => prev.map((item) => item.id === result.march.id ? result.march : item))
      void useGameStore.getState().loadMilitaryView()
      toast.success('行军已召回，返程完成后兵力会归队。')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '召回失败')
    } finally {
      setBusyMarch(null)
    }
  }

  // handleAccelerateMarch 使用城金加速自己的 PVP 行军。
  const handleAccelerateMarch = async (march: PvpMarch) => {
    if (!activePlayerId || busyMarch) return
    setBusyMarch(march.id)
    try {
      const result = await gameApi.acceleratePvpMarch(activePlayerId, march.id)
      useGameStore.getState().patchState({
        ...(typeof result.cityGold === 'number' ? { cityGold: result.cityGold } : {}),
        serverTime: result.serverTime,
      })
      setMarches((prev) => prev.map((item) => item.id === result.march.id ? result.march : item))
      toast.success(`行军已加速，消耗 ${result.cost ?? 0} 城金。`)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '加速失败')
    } finally {
      setBusyMarch(null)
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-16">
        <LoaderCircle size={16} className="animate-spin text-[var(--color-accent)]" />
        <span className="ml-2 text-xs text-[var(--color-text-muted)]">正在读取玩家目标...</span>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-3">
        <div className="flex min-w-0 flex-wrap items-center gap-2 text-xs text-[var(--color-text-muted)]">
          <span>当前视野 {targets.length} 个玩家目标</span>
          {targetView && <span>我的坐标 ({targetView.self.x}, {targetView.self.y})</span>}
          {pvpState && (
            <>
              <span className="rounded bg-[var(--color-surface-dim)] px-2 py-1 font-semibold text-[var(--color-text-secondary)]">积分 {pvpState.seasonPoints}</span>
              {rankings?.self && <span className="rounded bg-[var(--color-surface-dim)] px-2 py-1 font-semibold text-[var(--color-text-secondary)]">排名 {rankings.self.rank}</span>}
              <span className="rounded bg-[var(--color-surface-dim)] px-2 py-1 font-semibold text-[var(--color-text-secondary)]">胜 {pvpState.attackWins + pvpState.defenseWins}</span>
              <span className="rounded bg-[var(--color-surface-dim)] px-2 py-1 font-semibold text-[var(--color-text-secondary)]">复仇 {pvpState.revengeRecords.length}</span>
            </>
          )}
        </div>
        <button
          type="button"
          onClick={() => void load()}
          className="inline-flex items-center gap-1.5 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-1.5 text-xs font-medium text-[var(--color-text-secondary)] cursor-pointer hover:text-[var(--color-accent)]"
        >
          <RefreshCw size={12} />
          刷新
        </button>
      </div>

      {targetView && (
        <PvpWorldMap
          view={targetView}
          targets={targets}
          focusedTargetId={focusedTargetId}
          onFocusTarget={setFocusedTargetId}
          onFocusSelf={focusSelf}
          onPan={panViewport}
          onZoom={zoomViewport}
        />
      )}

      {focusedTarget && (
        <div className="rounded-xl border border-[var(--color-accent-border)] bg-[var(--color-accent-light)] px-3 py-2 text-xs text-[var(--color-text-secondary)]">
          已定位：<b className="text-[var(--color-text-primary)]">{focusedTarget.nickname}</b>
          <span className="ml-2">坐标 ({focusedTarget.position.x}, {focusedTarget.position.y})</span>
          <span className="ml-2">距离 {focusedTarget.distance}</span>
        </div>
      )}

      {activeMarches.length > 0 && (
        <section className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] p-3">
          <div className="mb-2 text-xs font-bold text-[var(--color-text-primary)]">我的 PVP 行军</div>
          <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
            {activeMarches.slice(0, 6).map((march) => (
              <div key={march.id} className="rounded-lg bg-[var(--color-surface-dim)] px-3 py-2">
                <div className="flex items-center gap-2">
                  <span className="truncate text-xs font-semibold text-[var(--color-text-primary)]">{march.defenderName}</span>
                  <span className="ml-auto rounded bg-white/60 px-1.5 py-0.5 text-[9px] font-bold text-[var(--color-accent)]">{PVP_STATUS_LABELS[march.status]}</span>
                </div>
                <div className="mt-1 flex items-center gap-2">
                  <span className="min-w-0 flex-1 truncate text-[10px] text-[var(--color-text-muted)]">
                    {march.marchType === 'plunder' ? '掠夺' : '攻击'} · {march.status === 'returning' ? '返程' : '剩余'} <CountdownText value={march.status === 'returning' ? march.returnsAt : march.arrivesAt} />
                  </span>
                  {march.attackerPlayerId === activePlayerId && march.status === 'marching' && (
                    <div className="flex shrink-0 items-center gap-1">
                      <button
                        type="button"
                        onClick={() => void handleAccelerateMarch(march)}
                        disabled={busyMarch !== null}
                        className="inline-flex h-6 items-center justify-center gap-1 rounded-md bg-amber-500/10 px-2 text-[10px] font-bold text-amber-600 cursor-pointer hover:bg-amber-500/20 disabled:opacity-50 disabled:cursor-not-allowed"
                      >
                        {busyMarch === march.id ? <LoaderCircle size={10} className="animate-spin" /> : <Zap size={10} />}
                        加速
                      </button>
                      <button
                        type="button"
                        onClick={() => void handleRecallMarch(march)}
                        disabled={busyMarch !== null}
                        className="inline-flex h-6 items-center justify-center gap-1 rounded-md bg-slate-500/10 px-2 text-[10px] font-bold text-slate-600 cursor-pointer hover:bg-slate-500/20 disabled:opacity-50 disabled:cursor-not-allowed"
                      >
                        {busyMarch === march.id ? <LoaderCircle size={10} className="animate-spin" /> : <RotateCcw size={10} />}
                        召回
                      </button>
                    </div>
                  )}
                </div>
              </div>
            ))}
          </div>
        </section>
      )}

      <div className="grid gap-2 sm:grid-cols-2 xl:grid-cols-3">
        {targets.map((target) => (
          <PlayerTargetCard
            key={target.playerId}
            target={target}
            focused={target.playerId === focusedTargetId}
            busyTarget={busyTarget}
            onScout={handleScout}
            onReinforce={handleReinforce}
            onMarch={openMarchSelector}
          />
        ))}
      </div>

      {targets.length === 0 && (
        <div className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] py-12 text-center text-sm text-[var(--color-text-muted)]">
          暂无可展示玩家目标
        </div>
      )}

      {scoutReport && <ScoutResultModal report={scoutReport} onClose={() => setScoutReport(null)} />}
      {selectedMarchTarget && (
        <div className="fixed inset-0 z-[9000] flex items-end justify-center bg-slate-950/45 p-0 sm:items-center sm:p-4">
          <div className="w-full max-w-md rounded-t-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4 shadow-xl sm:rounded-2xl">
            <div className="flex items-center justify-between gap-3">
              <div>
                <h3 className="text-sm font-bold text-[var(--color-text-primary)]">
                  {selectedMarchMode === 'plunder' ? '掠夺' : '攻击'} · {selectedMarchTarget.nickname}
                </h3>
                <p className="mt-0.5 text-xs text-[var(--color-text-muted)]">选择本次出征兵力</p>
              </div>
              <button
                type="button"
                onClick={() => setSelectedMarchTarget(null)}
                className="rounded-lg border border-[var(--color-border)] px-2 py-1 text-xs text-[var(--color-text-secondary)]"
              >
                关闭
              </button>
            </div>
            <div className="mt-3 max-h-64 space-y-2 overflow-y-auto">
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
                      onChange={(event) => handleSelectionChange(unit.unitType, event.target.value)}
                      className="h-8 w-20 rounded-lg border border-[var(--color-border)] bg-white px-2 text-center text-xs font-bold text-[var(--color-text-primary)] outline-none dark:bg-slate-900"
                    />
                    <button
                      type="button"
                      onClick={() => handleSelectAll(unit.unitType)}
                      className="h-8 rounded-lg px-2 text-[10px] font-bold text-[var(--color-accent)]"
                    >
                      全部
                    </button>
                  </div>
                </div>
              ))}
              {availableArmy.length === 0 && (
                <p className="py-8 text-center text-xs text-[var(--color-text-muted)]">当前没有可出征兵力</p>
              )}
            </div>
            <div className="mt-4 flex items-center justify-between gap-3">
              <span className="text-xs text-[var(--color-text-muted)]">已选 <b className="text-[var(--color-accent)]">{totalSelected.toLocaleString()}</b></span>
              <button
                type="button"
                onClick={() => void handleMarch()}
                disabled={totalSelected <= 0 || busyTarget !== null}
                className="rounded-xl bg-[var(--color-accent)] px-5 py-2 text-xs font-bold text-white disabled:opacity-50"
              >
                确认出征
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

const PvpWorldMap: FC<{
  view: PvpTargetsResponse
  targets: PvpTargetSummary[]
  focusedTargetId: string | null
  onFocusTarget: (playerId: string) => void
  onFocusSelf: () => void
  onPan: (dx: number, dy: number) => void
  onZoom: (delta: number) => void
}> = ({ view, targets, focusedTargetId, onFocusTarget, onFocusSelf, onPan, onZoom }) => {
  const radius = Math.max(1, view.radius)
  const step = Math.round(radius * 0.55)
  const toPercent = (position: PvpWorldPosition) => ({
    left: `${Math.max(3, Math.min(97, ((position.x - (view.centerX - radius)) / (radius * 2)) * 100))}%`,
    top: `${Math.max(5, Math.min(95, ((position.y - (view.centerY - radius)) / (radius * 2)) * 100))}%`,
  })

  return (
    <section className="overflow-hidden rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)]">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-[var(--color-border)] px-3 py-2">
        <div className="text-xs font-bold text-[var(--color-text-primary)]">
          世界地图
          <span className="ml-2 font-normal text-[var(--color-text-muted)]">
            中心 ({view.centerX}, {view.centerY}) · 半径 {view.radius}
          </span>
        </div>
        <div className="flex items-center gap-1">
          <MapControlButton label="左移" onClick={() => onPan(-step, 0)}>←</MapControlButton>
          <MapControlButton label="上移" onClick={() => onPan(0, -step)}>↑</MapControlButton>
          <MapControlButton label="下移" onClick={() => onPan(0, step)}>↓</MapControlButton>
          <MapControlButton label="右移" onClick={() => onPan(step, 0)}>→</MapControlButton>
          <button
            type="button"
            onClick={() => onZoom(-120)}
            className="inline-flex h-7 w-7 items-center justify-center rounded-lg border border-[var(--color-border)] text-[var(--color-text-secondary)] hover:text-[var(--color-accent)]"
            title="放大视野"
          >
            <Plus size={13} />
          </button>
          <button
            type="button"
            onClick={() => onZoom(120)}
            className="inline-flex h-7 w-7 items-center justify-center rounded-lg border border-[var(--color-border)] text-[var(--color-text-secondary)] hover:text-[var(--color-accent)]"
            title="缩小视野"
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

      <div className="relative h-[320px] bg-[linear-gradient(var(--color-border)_1px,transparent_1px),linear-gradient(90deg,var(--color-border)_1px,transparent_1px)] bg-[size:40px_40px]">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_center,rgba(14,165,233,0.08),transparent_55%)]" />
        <div
          className="absolute z-20 -translate-x-1/2 -translate-y-1/2"
          style={toPercent(view.self)}
          title={`我的城池 (${view.self.x}, ${view.self.y})`}
        >
          <div className="flex h-9 w-9 items-center justify-center rounded-full border-2 border-sky-500 bg-sky-500 text-[10px] font-black text-white shadow-lg shadow-sky-500/25">
            我
          </div>
        </div>
        {targets.map((target) => {
          const focused = target.playerId === focusedTargetId
          return (
            <button
              key={target.playerId}
              type="button"
              onClick={() => onFocusTarget(target.playerId)}
              className={`absolute z-10 -translate-x-1/2 -translate-y-1/2 rounded-full border bg-[var(--color-surface)] p-1 shadow-sm transition-all hover:scale-110 ${focused ? 'border-[var(--color-accent)] ring-4 ring-[var(--color-accent-light)]' : 'border-[var(--color-border)]'}`}
              style={toPercent(target.position)}
              title={`${target.nickname} (${target.position.x}, ${target.position.y}) 距离 ${target.distance}`}
            >
              <MapPin size={focused ? 22 : 18} className={FACTION_COLORS[target.faction] ?? 'text-[var(--color-text-muted)]'} />
            </button>
          )
        })}
        {targets.length === 0 && (
          <div className="absolute inset-0 flex items-center justify-center text-xs text-[var(--color-text-muted)]">
            当前视野没有其他玩家，可以缩小比例或移动地图。
          </div>
        )}
      </div>
    </section>
  )
}

const MapControlButton: FC<{ label: string; onClick: () => void; children: ReactNode }> = ({ label, onClick, children }) => (
  <button
    type="button"
    onClick={onClick}
    title={label}
    className="inline-flex h-7 w-7 items-center justify-center rounded-lg border border-[var(--color-border)] text-xs font-black text-[var(--color-text-secondary)] hover:text-[var(--color-accent)]"
  >
    {children}
  </button>
)

const PlayerTargetCard = memo(function PlayerTargetCard({
  target,
  focused,
  busyTarget,
  onScout,
  onReinforce,
  onMarch,
}: {
  target: PvpTargetSummary
  focused: boolean
  busyTarget: string | null
  onScout: (target: PvpTargetSummary) => Promise<void>
  onReinforce: (target: PvpTargetSummary) => void
  onMarch: (target: PvpTargetSummary, mode: 'attack' | 'plunder') => void
}) {
  const actionBusy = busyTarget !== null
  const marchDisabled = !target.canAttack || actionBusy
  const busyPrefix = `${target.playerId}:`

  return (
    <article className={`rounded-xl border bg-[var(--color-surface)] px-3 py-2.5 shadow-[0_2px_8px_rgba(15,23,42,0.03)] ${focused ? 'border-[var(--color-accent)] ring-2 ring-[var(--color-accent-light)]' : 'border-[var(--color-border)]'}`}>
      <div className="flex items-center gap-2">
        <div className="min-w-0">
          <div className="truncate text-xs font-black text-[var(--color-text-primary)]">{target.nickname}</div>
          <div className="mt-0.5 flex items-center gap-1.5 text-[10px]">
            <span className={`font-bold ${FACTION_COLORS[target.faction] ?? 'text-[var(--color-text-muted)]'}`}>{FACTION_LABELS[target.faction] ?? target.faction}</span>
            <span className="text-[var(--color-text-muted)]">兵力 {target.totalArmy.toLocaleString()}</span>
            <span className="text-[var(--color-text-muted)]">({target.position.x}, {target.position.y})</span>
            <span className="text-[var(--color-text-muted)]">距 {target.distance}</span>
          </div>
        </div>
        <span className="ml-auto shrink-0 rounded bg-[var(--color-surface-dim)] px-1.5 py-0.5 text-[9px] font-bold text-[var(--color-text-muted)]">
          Lv.{target.buildingLevel}
        </span>
      </div>

      {target.reason && <div className="mt-1.5 text-[10px] text-amber-600">{target.reason}</div>}

      <div className="mt-2 grid grid-cols-4 gap-1">
        <ActionButton label="侦查" icon={<Search size={10} />} busy={busyTarget === `${busyPrefix}scout`} disabled={actionBusy} onClick={() => void onScout(target)} tone="blue" />
        <ActionButton label="增援" icon={<ShieldPlus size={10} />} busy={false} disabled={busyTarget !== null || !target.canReinforce} onClick={() => onReinforce(target)} tone="green" />
        <ActionButton label="攻击" icon={<Swords size={10} />} busy={busyTarget === `${busyPrefix}attack`} disabled={marchDisabled} onClick={() => void onMarch(target, 'attack')} tone="red" />
        <ActionButton label="掠夺" icon={<ShieldAlert size={10} />} busy={busyTarget === `${busyPrefix}plunder`} disabled={marchDisabled} onClick={() => void onMarch(target, 'plunder')} tone="amber" />
      </div>
    </article>
  )
})

const TONE_CLASS: Record<string, string> = {
  blue: 'bg-blue-500/10 text-blue-600 hover:bg-blue-500/20',
  green: 'bg-emerald-500/10 text-emerald-600 hover:bg-emerald-500/20',
  red: 'bg-red-500/10 text-red-600 hover:bg-red-500/20',
  amber: 'bg-amber-500/10 text-amber-600 hover:bg-amber-500/20',
}

const ActionButton: FC<{ label: string; icon: ReactNode; busy: boolean; disabled: boolean; onClick: () => void; tone: string }> = ({ label, icon, busy, disabled, onClick, tone }) => (
  <button
    type="button"
    disabled={disabled}
    onClick={onClick}
    className={`inline-flex h-7 items-center justify-center gap-1 rounded-lg px-1 text-[10px] font-bold transition-colors cursor-pointer disabled:cursor-not-allowed disabled:opacity-45 ${TONE_CLASS[tone] ?? TONE_CLASS.blue}`}
  >
    {busy ? <LoaderCircle size={10} className="animate-spin" /> : icon}
    <span>{label}</span>
  </button>
)

// CountdownText 独立刷新倒计时，避免整个玩家目标页每秒重渲染。
const CountdownText: FC<{ value?: string }> = memo(({ value }) => {
  const [nowMs, setNowMs] = useState(Date.now())

  useEffect(() => {
    const timer = window.setInterval(() => setNowMs(Date.now()), 1000)
    return () => window.clearInterval(timer)
  }, [])

  return <>{formatCountdown(value, nowMs)}</>
})

// formatCountdown 将行军时间显示为剩余倒计时。
function formatCountdown(value: string | undefined, nowMs: number) {
  if (!value) return '-'
  const targetMs = new Date(value).getTime()
  if (Number.isNaN(targetMs)) return '-'
  const remaining = Math.max(0, Math.ceil((targetMs - nowMs) / 1000))
  const minutes = Math.floor(remaining / 60)
  const seconds = remaining % 60
  return `${minutes}:${seconds.toString().padStart(2, '0')}`
}

// isMarchDue 判断行军或返程是否已经到期，需要触发刷新结算。
function isMarchDue(march: PvpMarch, nowMs: number) {
  const targetTime = march.status === 'returning' ? march.returnsAt : march.arrivesAt
  if (!targetTime) return march.status === 'resolving'
  const targetMs = new Date(targetTime).getTime()
  if (Number.isNaN(targetMs)) return false
  return targetMs <= nowMs
}

export default PlayerTargetsTab
