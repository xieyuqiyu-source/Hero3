// 本文件实现军事页的战争页签，集中展示本城守军、派出增援和他城来援部队。
import { useCallback, useEffect, useState, type FC } from 'react'
import { RotateCcw, Shield, UserMinus, UsersRound, Zap } from 'lucide-react'
import { gameApi } from '@/api/game'
import { useConfigStore } from '@/store/configStore'
import { useGameStore } from '@/store/gameStore'
import type { ArmyUnit, Reinforcement } from '@/types/game'
import { sortUnitIds } from '@/utils/unitOrder'
import { useProjectedArmy } from '@/hooks/useProjectedArmy'

const ACTIVE_GARRISON_STATUSES = new Set<Reinforcement['status']>(['marching', 'stationed', 'fighting'])
const ACTIVE_SENT_STATUSES = new Set<Reinforcement['status']>(['marching', 'stationed', 'fighting', 'returning'])
const REINFORCEMENT_ACCELERATE_COST = 10
const REINFORCEMENT_MAX_ACCELERATE_TIMES = 2

const STATUS_LABELS: Record<Reinforcement['status'], string> = {
  marching: '行军中',
  stationed: '驻防中',
  fighting: '战斗中',
  returning: '返回中',
  completed: '已归档',
  cancelled: '已取消',
  failed: '异常',
}

/** 渲染战争页签 */
const WarTab: FC = () => {
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const state = useGameStore((s) => s.state)
  const patchState = useGameStore((s) => s.patchState)
  const units = useConfigStore((s) => s.units)
  const army = useProjectedArmy()
  const cityGold = typeof state?.cityGold === 'number' ? state.cityGold : 0
  const [sent, setSent] = useState<Reinforcement[]>([])
  const [received, setReceived] = useState<Reinforcement[]>([])
  const [loading, setLoading] = useState(false)
  const [actingId, setActingId] = useState<string | null>(null)
  const [expellingId, setExpellingId] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  // refresh 读取本城相关的活跃增援。
  const refresh = useCallback(async () => {
    if (!activePlayerId) {
      setSent([])
      setReceived([])
      return
    }
    setLoading(true)
    setError(null)
    try {
      const [sentResult, receivedResult] = await Promise.all([
        gameApi.listSentReinforcements(activePlayerId),
        gameApi.listReceivedReinforcements(activePlayerId),
      ])
      setSent((sentResult.items ?? []).filter((item) => ACTIVE_SENT_STATUSES.has(item.status)))
      setReceived((receivedResult.items ?? []).filter((item) => ACTIVE_GARRISON_STATUSES.has(item.status)))
    } catch (err) {
      setError(err instanceof Error ? err.message : '战争状态加载失败')
      setSent([])
      setReceived([])
    } finally {
      setLoading(false)
    }
  }, [activePlayerId])

  useEffect(() => {
    void refresh()
    const handleUpdate = () => void refresh()
    window.addEventListener('hero3:garrison-updated', handleUpdate)
    window.addEventListener('hero3:marches-updated', handleUpdate)
    return () => {
      window.removeEventListener('hero3:garrison-updated', handleUpdate)
      window.removeEventListener('hero3:marches-updated', handleUpdate)
    }
  }, [refresh])

  // handleExpel 遣返单支他城增援。
  const handleExpel = async (record: Reinforcement) => {
    if (!activePlayerId || expellingId) return
    setExpellingId(record.reinforcementId)
    setError(null)
    try {
      const result = await gameApi.expelReinforcement(activePlayerId, record.reinforcementId)
      if (result.patch) patchState(result.patch)
      await refresh()
      window.dispatchEvent(new Event('hero3:garrison-updated'))
      window.dispatchEvent(new Event('hero3:marches-updated'))
    } catch (err) {
      setError(err instanceof Error ? err.message : '遣返失败')
    } finally {
      setExpellingId(null)
    }
  }

  // handleRecall 召回自己派出的援军。
  const handleRecall = async (record: Reinforcement) => {
    if (!activePlayerId || actingId) return
    setActingId(record.reinforcementId)
    setError(null)
    try {
      const result = await gameApi.recallReinforcement(activePlayerId, record.reinforcementId)
      if (result.patch) patchState(result.patch)
      setSent((prev) => prev.map((item) => item.reinforcementId === result.reinforcement.reinforcementId ? result.reinforcement : item))
      await refresh()
      window.dispatchEvent(new Event('hero3:garrison-updated'))
      window.dispatchEvent(new Event('hero3:marches-updated'))
    } catch (err) {
      setError(err instanceof Error ? err.message : '召回失败')
    } finally {
      setActingId(null)
    }
  }

  // handleAccelerate 加速自己派出的行军中援军。
  const handleAccelerate = async (record: Reinforcement) => {
    if (!activePlayerId || actingId) return
    setActingId(record.reinforcementId)
    setError(null)
    try {
      const result = await gameApi.accelerateReinforcement(activePlayerId, record.reinforcementId)
      if (result.patch) patchState(result.patch)
      patchState({
        ...(typeof result.cityGold === 'number' ? { cityGold: result.cityGold } : {}),
        ...(result.serverTime ? { serverTime: result.serverTime } : {}),
      })
      setSent((prev) => prev.map((item) => item.reinforcementId === result.reinforcement.reinforcementId ? result.reinforcement : item))
      await refresh()
      window.dispatchEvent(new Event('hero3:marches-updated'))
    } catch (err) {
      setError(err instanceof Error ? err.message : '加速失败')
    } finally {
      setActingId(null)
    }
  }

  return (
    <div className="min-w-0 space-y-4">
      {error && <div className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">{error}</div>}

      <section className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] p-4">
        <div className="mb-3 flex items-center gap-2">
          <Shield size={16} className="text-[var(--color-accent)]" />
          <h2 className="text-sm font-bold text-[var(--color-text-primary)]">本城所属守城军队</h2>
          <button type="button" onClick={() => void useGameStore.getState().loadMilitaryView()} className="ml-auto inline-flex h-8 w-8 items-center justify-center rounded-lg border border-[var(--color-border)] text-[var(--color-text-secondary)] hover:text-[var(--color-accent)]">
            <RotateCcw size={14} />
          </button>
        </div>
        {state?.general && (
          <div className="mb-3 rounded-lg bg-[var(--color-surface-dim)] px-3 py-2">
            <div className="text-[11px] text-[var(--color-text-muted)]">守城主将</div>
            <div className="mt-0.5 text-sm font-bold text-[var(--color-text-primary)]">{state.general.name} Lv.{state.general.level}</div>
          </div>
        )}
        <WarTroopTable troops={armyToTroops(army)} units={units} faction={state?.player.faction} />
      </section>

      <section className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] p-4">
        <div className="mb-3 flex items-center gap-2">
          <UsersRound size={16} className="text-amber-600" />
          <h2 className="text-sm font-bold text-[var(--color-text-primary)]">我派出去的军队</h2>
          {loading && <span className="ml-auto text-xs text-[var(--color-text-muted)]">刷新中</span>}
        </div>
        {sent.length > 0 ? (
          <div className="grid gap-3">
            {sent.map((record) => (
              <ReinforcementSentCard
                key={record.reinforcementId}
                record={record}
                units={units}
                cityGold={cityGold}
                acting={actingId === record.reinforcementId}
                onRecall={handleRecall}
                onAccelerate={handleAccelerate}
              />
            ))}
          </div>
        ) : (
          <div className="rounded-lg bg-[var(--color-surface-dim)] px-4 py-8 text-center text-sm text-[var(--color-text-muted)]">
            暂无派出的军队
          </div>
        )}
      </section>

      <section className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] p-4">
        <div className="mb-3 flex items-center gap-2">
          <UsersRound size={16} className="text-emerald-600" />
          <h2 className="text-sm font-bold text-[var(--color-text-primary)]">他城来增援本城的军队</h2>
          {loading && <span className="ml-auto text-xs text-[var(--color-text-muted)]">刷新中</span>}
        </div>
        {received.length > 0 ? (
          <div className="grid gap-3">
            {received.map((record) => (
              <ReinforcementDefenseCard
                key={record.reinforcementId}
                record={record}
                units={units}
                expelling={expellingId === record.reinforcementId}
                onExpel={handleExpel}
              />
            ))}
          </div>
        ) : (
          <div className="rounded-lg bg-[var(--color-surface-dim)] px-4 py-8 text-center text-sm text-[var(--color-text-muted)]">
            暂无他城增援本城
          </div>
        )}
      </section>
    </div>
  )
}

// WarTroopTable 展示一张战报式兵种表，没有的兵种显示 0。
const WarTroopTable: FC<{
  troops?: Record<string, number>
  units: ReturnType<typeof useConfigStore.getState>['units']
  faction?: string
}> = ({ troops, units, faction }) => {
  const unitIds = sortUnitIds(Object.keys(units?.[faction ?? ''] ?? troops ?? {}), faction, units ?? undefined)
  if (unitIds.length === 0) return <div className="rounded-lg bg-[var(--color-surface-dim)] px-4 py-8 text-center text-sm text-[var(--color-text-muted)]">正在加载兵种配置...</div>
  return (
    <div className="overflow-x-auto rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)]">
      <div className="min-w-max">
        <div className="grid border-b border-[var(--color-border)]" style={{ gridTemplateColumns: `repeat(${unitIds.length}, minmax(96px, 1fr))` }}>
          {unitIds.map((unitType) => {
            return (
              <div
                key={unitType}
                className="min-w-0 border-r border-[var(--color-border)] bg-[var(--color-surface-dim)] px-2.5 py-2 text-center last:border-r-0"
              >
                <div className="truncate text-xs font-bold text-[var(--color-text-secondary)]">
                  {findUnitName(units, unitType, faction)}
                </div>
              </div>
            )
          })}
        </div>
        <div className="grid" style={{ gridTemplateColumns: `repeat(${unitIds.length}, minmax(96px, 1fr))` }}>
          {unitIds.map((unitType) => {
            const amount = troops?.[unitType] ?? 0
            return (
              <div
                key={unitType}
                className="min-w-0 border-r border-[var(--color-border)] bg-[var(--color-surface)] px-2.5 py-2.5 text-center last:border-r-0"
              >
                <div className={`text-base font-black tabular-nums ${amount > 0 ? 'text-emerald-600' : 'text-[var(--color-text-muted)]'}`}>
                  {amount.toLocaleString()}
                </div>
              </div>
            )
          })}
        </div>
      </div>
    </div>
  )
}

// armyToTroops 将军队数组转为兵种数量表。
function armyToTroops(army: ArmyUnit[]) {
  const result: Record<string, number> = {}
  for (const unit of army) {
    if (!unit.unitType) continue
    result[unit.unitType] = unit.amount
  }
  return result
}

// ReinforcementSentCard 展示自己派出的增援队伍。
const ReinforcementSentCard: FC<{
  record: Reinforcement
  units: ReturnType<typeof useConfigStore.getState>['units']
  cityGold: number
  acting: boolean
  onRecall: (record: Reinforcement) => Promise<void>
  onAccelerate: (record: Reinforcement) => Promise<void>
}> = ({ record, units, cityGold, acting, onRecall, onAccelerate }) => {
  const generals = record.generals ?? []
  const acceleratedTimes = readAcceleratedTimes(record.metadata)
  const canRecall = record.status === 'marching' || record.status === 'stationed'
  const canAccelerate = record.status === 'marching' && acceleratedTimes < REINFORCEMENT_MAX_ACCELERATE_TIMES
  const canPayAccelerate = cityGold >= REINFORCEMENT_ACCELERATE_COST
  return (
    <article className="min-w-0 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-3">
      <div className="flex min-w-0 flex-wrap items-center gap-2">
        <span className="min-w-0 truncate text-sm font-bold text-[var(--color-text-primary)]">{record.toPlayerName || record.toPlayerId || '未知城池'}</span>
        <span className="rounded bg-amber-500/10 px-2 py-0.5 text-xs font-bold text-amber-700">{STATUS_LABELS[record.status]}</span>
        <span className="ml-auto text-sm font-black text-amber-700">{totalTroops(record.remainingTroops).toLocaleString()}</span>
      </div>
      {generals.length > 0 && (
        <div className="mt-2 flex flex-wrap gap-1.5">
          {generals.map((general) => (
            <span key={general.id} className="text-xs font-bold text-amber-700">{general.name || general.id} Lv.{general.level}</span>
          ))}
        </div>
      )}
      <div className="mt-2">
        <WarTroopTable troops={record.remainingTroops} units={units} faction={record.fromPlayerFaction} />
      </div>
      {(canAccelerate || canRecall) && (
        <div className="mt-3 flex flex-wrap justify-end gap-2">
          {canAccelerate && (
            <button
              type="button"
              onClick={() => void onAccelerate(record)}
              disabled={acting || !canPayAccelerate}
              className="inline-flex items-center gap-1 rounded-lg border border-amber-300 bg-amber-50 px-3 py-1.5 text-xs font-bold text-amber-700 hover:bg-amber-100 disabled:cursor-not-allowed disabled:opacity-50"
              title={canPayAccelerate ? `消耗 ${REINFORCEMENT_ACCELERATE_COST} 城金，将剩余时间减半` : `城金不足，需要 ${REINFORCEMENT_ACCELERATE_COST} 城金`}
            >
              <Zap size={13} />
              加速
            </button>
          )}
          {canRecall && (
            <button
              type="button"
              onClick={() => void onRecall(record)}
              disabled={acting}
              className="inline-flex items-center gap-1 rounded-lg border border-[var(--color-border)] px-3 py-1.5 text-xs font-bold text-[var(--color-text-secondary)] hover:border-amber-400 hover:text-amber-600 disabled:cursor-not-allowed disabled:opacity-50"
            >
              <RotateCcw size={13} />
              {acting ? '召回中' : '召回'}
            </button>
          )}
        </div>
      )}
    </article>
  )
}

// ReinforcementDefenseCard 展示单支他城增援守军。
const ReinforcementDefenseCard: FC<{
  record: Reinforcement
  units: ReturnType<typeof useConfigStore.getState>['units']
  expelling: boolean
  onExpel: (record: Reinforcement) => Promise<void>
}> = ({ record, units, expelling, onExpel }) => {
  const generals = record.generals ?? []
  const canExpel = record.sourceType === 'reinforcement' && (record.status === 'marching' || record.status === 'stationed')
  return (
    <article className="min-w-0 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-3">
      <div className="flex min-w-0 flex-wrap items-center gap-2">
        <span className="min-w-0 truncate text-sm font-bold text-[var(--color-text-primary)]">{record.fromPlayerName || record.fromPlayerId || '未知城池'}</span>
        <span className="rounded bg-emerald-500/10 px-2 py-0.5 text-xs font-bold text-emerald-600">{STATUS_LABELS[record.status]}</span>
        <span className="ml-auto text-sm font-black text-emerald-600">{totalTroops(record.remainingTroops).toLocaleString()}</span>
      </div>
      {generals.length > 0 && (
        <div className="mt-2 flex flex-wrap gap-1.5">
          {generals.map((general) => (
            <span key={general.id} className="text-xs font-bold text-emerald-600">{general.name || general.id} Lv.{general.level}</span>
          ))}
        </div>
      )}
      <div className="mt-2">
        <WarTroopTable troops={record.remainingTroops} units={units} faction={record.fromPlayerFaction} />
      </div>
      {canExpel && (
        <div className="mt-3 flex justify-end">
          <button
            type="button"
            onClick={() => void onExpel(record)}
            disabled={expelling}
            className="inline-flex items-center gap-1 rounded-lg border border-[var(--color-border)] px-3 py-1.5 text-xs font-bold text-[var(--color-text-secondary)] hover:border-red-400 hover:text-red-500 disabled:cursor-not-allowed disabled:opacity-50"
          >
            <UserMinus size={13} />
            {expelling ? '遣返中' : '遣返'}
          </button>
        </div>
      )}
    </article>
  )
}

// findUnitName 根据兵种 ID 查找展示名。
function findUnitName(units: ReturnType<typeof useConfigStore.getState>['units'], unitType: string, faction?: string): string {
  if (faction && units?.[faction]?.[unitType]?.name) return units[faction][unitType].name
  for (const factionUnits of Object.values(units ?? {})) {
    if (factionUnits[unitType]?.name) return factionUnits[unitType].name
  }
  return unitType
}

// totalTroops 统计兵力总数。
function totalTroops(troops?: Record<string, number>) {
  return Object.values(troops ?? {}).reduce((sum, amount) => sum + amount, 0)
}

// readAcceleratedTimes 读取援军已加速次数。
function readAcceleratedTimes(metadata?: Record<string, unknown>) {
  const value = metadata?.acceleratedTimes
  return typeof value === 'number' && Number.isFinite(value) ? value : 0
}

export default WarTab
