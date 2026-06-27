// 本文件实现地图页的 PVP 玩家目标列表和快捷操作。
import { memo, useCallback, useEffect, useMemo, useRef, useState, type FC, type ReactNode } from 'react'
import { LoaderCircle, RefreshCw, RotateCcw, Search, ShieldAlert, ShieldPlus, Swords, Zap } from 'lucide-react'
import { gameApi } from '@/api/game'
import { toast } from '@/components/ui'
import { useGameStore } from '@/store/gameStore'
import { useConfigStore } from '@/store/configStore'
import { FACTION_COLORS, FACTION_LABELS } from '@/utils/faction'
import type { BattleReport, PvpMarch, PvpRankingResponse, PvpStateResponse, PvpTargetSummary } from '@/types/game'
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
  const [marches, setMarches] = useState<PvpMarch[]>([])
  const [pvpState, setPvpState] = useState<PvpStateResponse | null>(null)
  const [rankings, setRankings] = useState<PvpRankingResponse | null>(null)
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
        gameApi.listPvpTargets(activePlayerId),
        gameApi.listPvpMarches(activePlayerId),
        gameApi.getPvpState(activePlayerId),
        gameApi.listPvpRankings(activePlayerId, 10),
      ])
      setTargets(targetResult.items ?? [])
      setMarches(marchResult.items ?? [])
      setPvpState(stateResult)
      setRankings(rankingResult)
    } finally {
      if (!silent) setLoading(false)
    }
  }, [activePlayerId])

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
          <span>共 {targets.length} 个玩家目标</span>
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

const PlayerTargetCard = memo(function PlayerTargetCard({
  target,
  busyTarget,
  onScout,
  onReinforce,
  onMarch,
}: {
  target: PvpTargetSummary
  busyTarget: string | null
  onScout: (target: PvpTargetSummary) => Promise<void>
  onReinforce: (target: PvpTargetSummary) => void
  onMarch: (target: PvpTargetSummary, mode: 'attack' | 'plunder') => void
}) {
  const actionBusy = busyTarget !== null
  const marchDisabled = !target.canAttack || actionBusy
  const busyPrefix = `${target.playerId}:`

  return (
    <article className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2.5 shadow-[0_2px_8px_rgba(15,23,42,0.03)]">
      <div className="flex items-center gap-2">
        <div className="min-w-0">
          <div className="truncate text-xs font-black text-[var(--color-text-primary)]">{target.nickname}</div>
          <div className="mt-0.5 flex items-center gap-1.5 text-[10px]">
            <span className={`font-bold ${FACTION_COLORS[target.faction] ?? 'text-[var(--color-text-muted)]'}`}>{FACTION_LABELS[target.faction] ?? target.faction}</span>
            <span className="text-[var(--color-text-muted)]">兵力 {target.totalArmy.toLocaleString()}</span>
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
