/* 地图玩家城池分页，负责玩家目标卡片、PVP 出征行军、行军队列和加速。 */

import { useCallback, useEffect, useMemo, useState, type FC } from 'react'
import { Castle, Clock, Loader2, RefreshCw, ShieldAlert, Swords, Users, X, Zap } from 'lucide-react'
import { gameApi } from '@/api/game'
import { toast } from '@/components/ui'
import { useConfigStore } from '@/store/configStore'
import { useGameStore } from '@/store/gameStore'
import { FACTION_COLORS, FACTION_LABELS } from '@/utils/faction'
import type { PvpMarchView, PvpTarget } from '@/types/game'

// formatDateTime 格式化行军时间。
function formatDateTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

// formatCountdown 格式化倒计时。
function formatCountdown(seconds: number) {
  const safe = Math.max(0, seconds)
  const h = Math.floor(safe / 3600)
  const m = Math.floor((safe % 3600) / 60)
  const s = safe % 60
  if (h > 0) return `${h}时${m}分${s}秒`
  if (m > 0) return `${m}分${s}秒`
  return `${s}秒`
}

const PlayerCityTab: FC = () => {
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const gameState = useGameStore((s) => s.state)
  const patchState = useGameStore((s) => s.patchState)
  const unitsConfig = useConfigStore((s) => s.units)
  const marchConfig = useConfigStore((s) => s.balance?.march)
  const [targets, setTargets] = useState<PvpTarget[]>([])
  const [marches, setMarches] = useState<PvpMarchView[]>([])
  const [selectedTarget, setSelectedTarget] = useState<PvpTarget | null>(null)
  const [dispatchUnits, setDispatchUnits] = useState<Record<string, number>>({})
  const [loadingTargets, setLoadingTargets] = useState(false)
  const [loadingMarches, setLoadingMarches] = useState(false)
  const [attacking, setAttacking] = useState(false)
  const [acceleratingId, setAcceleratingId] = useState('')
  const [message, setMessage] = useState('')
  const [, setTick] = useState(0)

  const armyOptions = useMemo(() => {
    const factionUnits = gameState?.player.faction ? unitsConfig?.[gameState.player.faction] : undefined
    return (gameState?.army ?? [])
      .filter((unit) => unit.amount > 0)
      .filter((unit) => {
        const cfg = factionUnits?.[unit.unitType]
        return cfg && cfg.role !== 'transport' && (cfg.stats?.upkeep ?? 0) > 0
      })
      .map((unit) => ({
        ...unit,
        name: factionUnits?.[unit.unitType]?.name ?? unit.unitType,
      }))
  }, [gameState?.army, gameState?.player.faction, unitsConfig])

  const loadTargets = useCallback(async () => {
    if (!activePlayerId) return
    setLoadingTargets(true)
    setMessage('')
    try {
      const result = await gameApi.listPvpTargets(activePlayerId, 1, 50)
      setTargets(result.targets)
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '目标列表加载失败')
    } finally {
      setLoadingTargets(false)
    }
  }, [activePlayerId])

  const loadMarches = useCallback(async () => {
    if (!activePlayerId) return
    setLoadingMarches(true)
    try {
      const result = await gameApi.listPvpMarches(activePlayerId)
      setMarches(result.marches)
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '行军列表加载失败')
    } finally {
      setLoadingMarches(false)
    }
  }, [activePlayerId])

  useEffect(() => {
    void loadTargets()
    void loadMarches()
  }, [loadTargets, loadMarches])

  useEffect(() => {
    const id = window.setInterval(() => setTick((value) => value + 1), 1000)
    return () => window.clearInterval(id)
  }, [])

  const handleUnitChange = (unitType: string, amount: number) => {
    setDispatchUnits((current) => {
      const next = { ...current }
      if (amount <= 0) delete next[unitType]
      else next[unitType] = amount
      return next
    })
  }

  const handleTargetSelect = (target: PvpTarget) => {
    setSelectedTarget(target)
    setDispatchUnits({})
  }

  const handleAttack = async () => {
    if (!activePlayerId || !selectedTarget || attacking) return
    const units = Object.fromEntries(Object.entries(dispatchUnits).filter(([, amount]) => amount > 0))
    if (Object.keys(units).length === 0) {
      setMessage('请选择出征兵力')
      return
    }
    setAttacking(true)
    setMessage('')
    try {
      const result = await gameApi.attackPlayer(activePlayerId, selectedTarget.playerId, 'attack', units)
      patchState(result.state)
      setDispatchUnits({})
      toast.success('行军已出发')
      setMessage(`已出发，预计 ${formatDateTime(result.arrivesAt)} 到达`)
      await loadMarches()
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '发起攻击失败')
    } finally {
      setAttacking(false)
    }
  }

  const handleAccelerate = async (march: PvpMarchView) => {
    if (!activePlayerId || acceleratingId) return
    setAcceleratingId(march.id)
    setMessage('')
    try {
      const result = await gameApi.acceleratePvpMarch(activePlayerId, march.id)
      patchState(result.state)
      setMarches((items) => items.map((item) => item.id === march.id ? result.march : item))
      toast.success('行军已加速')
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '加速失败')
    } finally {
      setAcceleratingId('')
    }
  }

  return (
    <div className="space-y-4">
      <div className="px-3 py-2 rounded-xl bg-[var(--color-surface-dim)] border border-[var(--color-border)] text-[10px] text-[var(--color-text-muted)] leading-relaxed">
        <span className="font-medium text-[var(--color-text-secondary)]">说明：</span>
        玩家城池只显示目标概要，不暴露完整军队。攻击会创建行军队列，抵达后自动结算，详细结果在军情查看。
      </div>

      <div className="flex items-center justify-between gap-3 flex-wrap">
        <span className="text-xs text-[var(--color-text-muted)]">共 {targets.length} 个玩家城池</span>
        <button
          type="button"
          onClick={() => void loadTargets()}
          disabled={loadingTargets}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium bg-[var(--color-surface-dim)] border border-[var(--color-border)] text-[var(--color-text-secondary)] hover:border-[var(--color-accent-border)] cursor-pointer transition-colors disabled:opacity-50"
        >
          <RefreshCw size={12} className={loadingTargets ? 'animate-spin' : ''} />
          刷新玩家
        </button>
      </div>

      {message && (
        <div className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-xs text-[var(--color-text-secondary)]">
          {message}
        </div>
      )}

      {loadingTargets && targets.length === 0 ? (
        <PlayerCardSkeleton />
      ) : targets.length === 0 ? (
        <div className="py-10 text-center text-sm text-[var(--color-text-muted)]">暂无可攻击玩家城池</div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
          {targets.map((target, i) => (
            <div
              key={target.playerId}
              className="animate-fade-in-up"
              style={{ animationDelay: `${i * 50}ms`, animationFillMode: 'both' }}
            >
              <PlayerCityCard
                target={target}
                selected={selectedTarget?.playerId === target.playerId}
                onClick={() => handleTargetSelect(target)}
              />
            </div>
          ))}
        </div>
      )}

      {selectedTarget && (
        <section className="fixed inset-x-0 bottom-0 z-[8000] lg:relative lg:inset-auto">
          <div className="rounded-t-2xl border-t border-[var(--color-border)] bg-[var(--color-surface)] p-4 shadow-[0_-8px_30px_rgba(15,23,42,0.1)] lg:rounded-2xl lg:border lg:shadow-md">
            <div className="mb-3 flex items-center justify-between gap-3">
              <div className="flex min-w-0 items-center gap-2">
                <ShieldAlert size={16} className="text-[var(--color-accent)]" />
                <span className="truncate text-sm font-bold text-[var(--color-text-primary)]">出征 → {selectedTarget.nickname}</span>
              </div>
              <button type="button" onClick={() => setSelectedTarget(null)} className="p-1 rounded-lg hover:bg-[var(--color-surface-dim)] cursor-pointer">
                <X size={16} className="text-[var(--color-text-muted)]" />
              </button>
            </div>
            <div className="grid gap-3 lg:grid-cols-[1fr_auto] lg:items-end">
              <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
                {armyOptions.length === 0 ? (
                  <div className="py-6 text-center text-sm text-[var(--color-text-muted)] sm:col-span-2 lg:col-span-3">暂无可出征主军队</div>
                ) : armyOptions.map((unit) => (
                  <label key={unit.unitType} className="grid gap-1 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-3 text-xs">
                    <span className="flex items-center justify-between text-[var(--color-text-secondary)]">
                      <span>{unit.name}</span>
                      <span>可用 {unit.amount.toLocaleString()}</span>
                    </span>
                    <div className="flex items-center gap-1.5">
                      <input
                        type="number"
                        min={0}
                        max={unit.amount}
                        value={dispatchUnits[unit.unitType] ?? 0}
                        onChange={(event) => handleUnitChange(unit.unitType, Math.min(unit.amount, Math.max(0, Number(event.target.value))))}
                        className="min-w-0 flex-1 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 py-2 text-sm outline-none focus:border-[var(--color-accent-border)]"
                      />
                      <button
                        type="button"
                        onClick={() => handleUnitChange(unit.unitType, unit.amount)}
                        className="rounded-lg px-2 py-2 text-[10px] font-medium text-[var(--color-accent)] hover:bg-[var(--color-surface)] cursor-pointer"
                      >
                        全部
                      </button>
                    </div>
                  </label>
                ))}
              </div>
              <button
                type="button"
                onClick={() => void handleAttack()}
                disabled={attacking}
                className="inline-flex h-10 items-center justify-center gap-2 rounded-xl bg-[var(--color-accent)] px-5 text-sm font-semibold text-white hover:opacity-90 disabled:opacity-50 cursor-pointer"
              >
                {attacking ? <Loader2 size={16} className="animate-spin" /> : <Swords size={16} />}
                创建行军
              </button>
            </div>
          </div>
        </section>
      )}

      <section className="min-w-0 overflow-hidden rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)]">
        <div className="flex items-center gap-2 border-b border-[var(--color-border)] px-4 py-3">
          <Clock size={16} className="text-[var(--color-accent)]" />
          <h2 className="text-sm font-semibold text-[var(--color-text-primary)]">行军队列</h2>
          {marchConfig && <span className="ml-auto text-[10px] text-[var(--color-text-muted)]">加速 {marchConfig.accelerate.costCityGold} 城金/次</span>}
          <button
            type="button"
            onClick={() => void loadMarches()}
            className="inline-flex h-8 w-8 items-center justify-center rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] text-[var(--color-text-secondary)] hover:border-[var(--color-accent-border)] cursor-pointer"
            title="刷新行军"
          >
            <RefreshCw size={14} className={loadingMarches ? 'animate-spin' : ''} />
          </button>
        </div>
        <div className="grid gap-2 p-3">
          {marches.length === 0 ? (
            <div className="py-8 text-center text-sm text-[var(--color-text-muted)]">暂无行军</div>
          ) : marches.map((march) => {
            const arrivesAt = new Date(march.arrivesAt)
            const remaining = Number.isNaN(arrivesAt.getTime()) ? march.remainingSeconds : Math.max(0, Math.ceil((arrivesAt.getTime() - Date.now()) / 1000))
            return (
              <div key={march.id} className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-3">
                <div className="flex flex-wrap items-center gap-2">
                  <span className={`rounded-full px-2 py-0.5 text-[10px] font-semibold ${march.direction === 'incoming' ? 'bg-red-500/10 text-red-500' : 'bg-emerald-500/10 text-emerald-600'}`}>
                    {march.direction === 'incoming' ? '敌军来袭' : '我方出征'}
                  </span>
                  <span className="text-sm font-semibold text-[var(--color-text-primary)]">
                    {march.direction === 'incoming' ? march.sourceName : march.targetName}
                  </span>
                  <span className="text-[11px] text-[var(--color-text-muted)]">{formatCountdown(remaining)}</span>
                  <span className="ml-auto text-[11px] text-[var(--color-text-muted)]">{formatDateTime(march.arrivesAt)}</span>
                </div>
                {march.units && (
                  <div className="mt-2 flex flex-wrap gap-1.5 text-[11px] text-[var(--color-text-secondary)]">
                    {Object.entries(march.units).map(([unitType, amount]) => (
                      <span key={unitType} className="rounded-lg bg-[var(--color-surface)] px-2 py-1">{unitType} x {amount}</span>
                    ))}
                  </div>
                )}
                {march.direction === 'outgoing' && march.canAccelerate && (
                  <button
                    type="button"
                    onClick={() => void handleAccelerate(march)}
                    disabled={Boolean(acceleratingId)}
                    className="mt-3 inline-flex h-8 items-center gap-1.5 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-3 text-xs text-[var(--color-text-secondary)] hover:border-[var(--color-accent-border)] hover:text-[var(--color-accent)] disabled:opacity-50 cursor-pointer"
                  >
                    {acceleratingId === march.id ? <Loader2 size={13} className="animate-spin" /> : <Zap size={13} />}
                    加速 {march.accelerateCost} 城金
                  </button>
                )}
                {march.status === 'resolved' && (
                  <div className="mt-2 text-[11px] text-[var(--color-text-muted)]">已抵达，可前往军情查看战报。</div>
                )}
              </div>
            )
          })}
        </div>
      </section>
    </div>
  )
}

const PlayerCardSkeleton: FC = () => (
  <div className="space-y-4">
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
      {Array.from({ length: 9 }).map((_, i) => (
        <div
          key={i}
          className="rounded-2xl border border-[var(--color-border)] p-3 h-[112px] backdrop-blur-sm bg-white/40 dark:bg-white/5 animate-pulse"
          style={{ animationDelay: `${i * 80}ms` }}
        />
      ))}
    </div>
    <div className="flex items-center justify-center pt-4">
      <Loader2 size={16} className="text-[var(--color-accent)] animate-spin" />
      <span className="text-xs text-[var(--color-text-muted)] ml-2">正在搜索玩家城池...</span>
    </div>
  </div>
)

const PlayerCityCard: FC<{
  target: PvpTarget
  selected: boolean
  onClick: () => void
}> = ({ target, selected, onClick }) => (
  <div
    className={`
      rounded-2xl border p-3 transition-all duration-200
      ${selected
        ? 'border-[var(--color-accent)] bg-[var(--color-accent-light)] shadow-md'
        : 'border-blue-300 bg-blue-50 hover:border-[var(--color-accent-border)] hover:shadow-sm dark:border-blue-600 dark:bg-blue-900/20'
      }
    `}
  >
    <button type="button" onClick={onClick} className="w-full text-left cursor-pointer">
      <div className="flex items-center gap-2 flex-wrap">
        <span className="text-sm font-bold text-[var(--color-text-primary)]">{target.nickname}</span>
        <span className="text-[10px] font-semibold px-1.5 py-0.5 rounded text-blue-500 bg-white/60 dark:bg-white/10">玩家</span>
        <span className={`text-[10px] font-bold ${FACTION_COLORS[target.faction] ?? 'text-[var(--color-text-muted)]'}`}>
          {FACTION_LABELS[target.faction] ?? target.faction}
        </span>
        <span className="ml-auto flex-shrink-0" title="玩家城池">
          <Castle size={13} className="text-[var(--color-accent)]" />
        </span>
      </div>
      <div className="mt-3 grid grid-cols-2 gap-2">
        <div className="rounded-xl border border-[var(--color-border)] bg-white/60 px-2 py-1.5 dark:bg-white/5">
          <div className="text-[9px] text-[var(--color-text-muted)]">兵力估算</div>
          <div className="mt-0.5 text-sm font-black text-[var(--color-text-primary)]">{target.totalArmy.toLocaleString()}</div>
        </div>
        <div className="rounded-xl border border-[var(--color-border)] bg-white/60 px-2 py-1.5 dark:bg-white/5">
          <div className="text-[9px] text-[var(--color-text-muted)]">主城等级</div>
          <div className="mt-0.5 text-sm font-black text-[var(--color-text-primary)]">{target.buildingLevel}</div>
        </div>
      </div>
    </button>

    <div className="flex gap-1.5 mt-2 pt-2 border-t border-[var(--color-border)]">
      <button
        type="button"
        onClick={onClick}
        className="flex-1 flex items-center justify-center gap-1 px-2 py-1.5 rounded-lg text-[10px] font-medium bg-red-500/10 text-red-600 hover:bg-red-500/20 cursor-pointer transition-colors"
      >
        <Swords size={10} />选择出征
      </button>
      <div className="flex flex-1 items-center justify-center gap-1 px-2 py-1.5 rounded-lg text-[10px] font-medium bg-blue-500/10 text-blue-600">
        <Users size={10} />可被攻击
      </div>
    </div>
  </div>
)

export default PlayerCityTab
