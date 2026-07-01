// 本文件实现增援与驻防队伍页面，收到列表按来源聚合展示驻防队伍。
import { useEffect, useMemo, useState, type FC } from 'react'
import { RotateCcw, Send, ShieldCheck, UserMinus, Zap } from 'lucide-react'
import { gameApi } from '@/api/game'
import { useGameStore } from '@/store/gameStore'
import { useConfigStore } from '@/store/configStore'
import type { Reinforcement } from '@/types/game'

type Tab = 'sent' | 'received'

const STATUS_LABELS: Record<Reinforcement['status'], string> = {
  marching: '行军中',
  stationed: '驻扎中',
  fighting: '战斗中',
  returning: '返回中',
  completed: '已归档',
  cancelled: '已取消',
  failed: '异常',
}

const SOURCE_LABELS: Record<'reinforcement' | 'obtained', string> = {
  reinforcement: '增援',
  obtained: '获得',
}

const ACTIVE_GARRISON_STATUSES = new Set<Reinforcement['status']>(['marching', 'stationed', 'fighting', 'returning'])

const STATUS_WEIGHT: Partial<Record<Reinforcement['status'], number>> = {
  fighting: 5,
  stationed: 4,
  marching: 3,
  returning: 2,
  completed: 1,
  cancelled: 0,
  failed: 0,
}

const ReinforcementsPage: FC = () => {
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const state = useGameStore((s) => s.state)
  const patchState = useGameStore((s) => s.patchState)
  const [tab, setTab] = useState<Tab>('sent')
  const [sent, setSent] = useState<Reinforcement[]>([])
  const [received, setReceived] = useState<Reinforcement[]>([])
  const [targetPlayerId, setTargetPlayerId] = useState('')
  const [selectedUnit, setSelectedUnit] = useState('')
  const [amount, setAmount] = useState(1)
  const [selectedGeneral, setSelectedGeneral] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const factionUnits = useConfigStore((s) => (state?.player.faction ? s.units?.[state.player.faction] : undefined))
  const unitOptions = useMemo(() => Object.entries(factionUnits ?? {}), [factionUnits])
  const availableGenerals = useMemo(() => {
    const busy = new Set((state?.generalAssignments ?? []).filter((item) => item.id !== 'main' && item.slot !== 'main').map((item) => item.generalId))
    return (state?.generals ?? []).filter((general) => !busy.has(general.id))
  }, [state?.generalAssignments, state?.generals])

  const refresh = async () => {
    if (!activePlayerId) return
    setLoading(true)
    setError(null)
    try {
      const [sentResult, receivedResult] = await Promise.all([
        gameApi.listSentReinforcements(activePlayerId),
        gameApi.listReceivedReinforcements(activePlayerId),
      ])
      setSent(sentResult.items)
      setReceived(receivedResult.items)
    } catch (err) {
      setError(err instanceof Error ? err.message : '增援列表加载失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void refresh()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activePlayerId])

  useEffect(() => {
    if (!selectedUnit && unitOptions.length > 0) {
      setSelectedUnit(unitOptions[0][0])
    }
  }, [selectedUnit, unitOptions])

  const handleSend = async () => {
    if (!activePlayerId || !selectedUnit || amount <= 0) return
    setLoading(true)
    setError(null)
    try {
      const result = await gameApi.sendReinforcement(
        activePlayerId,
        targetPlayerId.trim(),
        { [selectedUnit]: amount },
        selectedGeneral ? [selectedGeneral] : [],
      )
      if (result.patch) patchState(result.patch)
      setTargetPlayerId('')
      setAmount(1)
      setSelectedGeneral('')
      await refresh()
    } catch (err) {
      setError(err instanceof Error ? err.message : '发起增援失败')
    } finally {
      setLoading(false)
    }
  }

  const handleRecall = async (record: Reinforcement) => {
    if (!activePlayerId) return
    const result = await gameApi.recallReinforcement(activePlayerId, record.reinforcementId)
    if (result.patch) patchState(result.patch)
    await refresh()
  }

  const handleExpel = async (record: Reinforcement) => {
    if (!activePlayerId) return
    const result = await gameApi.expelReinforcement(activePlayerId, record.reinforcementId)
    if (result.patch) patchState(result.patch)
    await refresh()
  }

  const handleAccelerate = async (record: Reinforcement) => {
    if (!activePlayerId) return
    const result = await gameApi.accelerateReinforcement(activePlayerId, record.reinforcementId)
    if (result.patch) patchState(result.patch)
    patchState({
      ...(typeof result.cityGold === 'number' ? { cityGold: result.cityGold } : {}),
      ...(result.serverTime ? { serverTime: result.serverTime } : {}),
    })
    await refresh()
  }

  const receivedDisplay = useMemo(() => aggregateGarrisonDisplayRecords(activePlayerId ?? '', received), [activePlayerId, received])
  const items = tab === 'sent' ? sent : receivedDisplay

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-center gap-2">
        <button type="button" onClick={() => setTab('sent')} className={`px-4 py-2 rounded-lg border text-sm font-semibold cursor-pointer ${tab === 'sent' ? 'bg-[var(--color-accent-light)] border-[var(--color-accent-border)] text-[var(--color-accent)]' : 'border-[var(--color-border)] text-[var(--color-text-secondary)]'}`}>
          我派出的
        </button>
        <button type="button" onClick={() => setTab('received')} className={`px-4 py-2 rounded-lg border text-sm font-semibold cursor-pointer ${tab === 'received' ? 'bg-[var(--color-accent-light)] border-[var(--color-accent-border)] text-[var(--color-accent)]' : 'border-[var(--color-border)] text-[var(--color-text-secondary)]'}`}>
          驻防队伍
        </button>
        <button type="button" onClick={() => void refresh()} className="ml-auto w-9 h-9 inline-flex items-center justify-center rounded-lg border border-[var(--color-border)] text-[var(--color-text-secondary)] cursor-pointer hover:text-[var(--color-accent)]">
          <RotateCcw size={16} />
        </button>
      </div>

      {error && <div className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">{error}</div>}

      <div className="grid gap-4 lg:grid-cols-[360px_1fr]">
        <section className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] p-4 space-y-3">
          <div className="flex items-center gap-2 text-sm font-bold text-[var(--color-text-primary)]">
            <Send size={16} className="text-[var(--color-accent)]" />
            发起增援
          </div>
          <input value={targetPlayerId} onChange={(e) => setTargetPlayerId(e.target.value)} placeholder="目标玩家 ID" className="w-full rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-sm outline-none focus:border-[var(--color-accent-border)]" />
          <div className="grid grid-cols-[1fr_96px] gap-2">
            <select value={selectedUnit} onChange={(e) => setSelectedUnit(e.target.value)} className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-sm outline-none">
              {unitOptions.map(([unitId, config]) => <option key={unitId} value={unitId}>{config.name ?? unitId}</option>)}
            </select>
            <input type="number" min={1} value={amount} onChange={(e) => setAmount(Math.max(1, Number(e.target.value) || 1))} className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-sm outline-none" />
          </div>
          <select value={selectedGeneral} onChange={(e) => setSelectedGeneral(e.target.value)} className="w-full rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-sm outline-none">
            <option value="">不携带武将</option>
            {availableGenerals.map((general) => <option key={general.id} value={general.id}>{general.name} Lv.{general.level}</option>)}
          </select>
          <button type="button" disabled={loading || !targetPlayerId.trim()} onClick={() => void handleSend()} className="w-full inline-flex items-center justify-center gap-2 rounded-lg bg-[var(--color-accent)] px-4 py-2 text-sm font-bold text-white cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed">
            <ShieldCheck size={16} />
            派出
          </button>
        </section>

        <section className="grid gap-3">
          {items.map((record) => (
            <ReinforcementCard key={record.reinforcementId} record={record} mode={tab} onRecall={handleRecall} onExpel={handleExpel} onAccelerate={handleAccelerate} />
          ))}
          {items.length === 0 && (
            <div className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-4 py-12 text-center text-sm text-[var(--color-text-muted)]">
              暂无队伍
            </div>
          )}
        </section>
      </div>
    </div>
  )
}

const ReinforcementCard: FC<{
  record: Reinforcement
  mode: Tab
  onRecall: (record: Reinforcement) => Promise<void>
  onExpel: (record: Reinforcement) => Promise<void>
  onAccelerate: (record: Reinforcement) => Promise<void>
}> = ({ record, mode, onRecall, onExpel, onAccelerate }) => {
  const sourceType = normalizeDisplaySourceType(record.sourceType)
  const sourceTitle = SOURCE_LABELS[sourceType]
  const sourceName = sourceType === 'obtained' ? '自己' : (record.fromPlayerName || record.fromPlayerId)
  const name = mode === 'sent' ? (record.toPlayerName || record.toPlayerId) : `${sourceTitle} · ${sourceName}`
  const canRecall = mode === 'sent' && (record.status === 'marching' || record.status === 'stationed')
  const canAccelerate = mode === 'sent' && record.status === 'marching' && readAcceleratedTimes(record.metadata) < 2
  const canExpel = mode === 'received' && sourceType === 'reinforcement' && (record.status === 'marching' || record.status === 'stationed')
  return (
    <article className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] p-4">
      <div className="flex flex-wrap items-center gap-2">
        <span className="text-sm font-bold text-[var(--color-text-primary)]">{name}</span>
        <span className="rounded bg-[var(--color-accent-light)] px-2 py-0.5 text-xs font-semibold text-[var(--color-accent)]">{STATUS_LABELS[record.status]}</span>
        <span className="ml-auto text-xs text-[var(--color-text-muted)]">{record.arriveAt || record.expectedReturnedAt || record.createdAt}</span>
      </div>
      <div className="mt-3 grid gap-2 sm:grid-cols-3">
        <Metric label="派出兵力" value={formatTroops(record.troops)} />
        <Metric label="剩余兵力" value={formatTroops(record.remainingTroops)} />
        <Metric label="携带武将" value={(record.generals ?? []).map((general) => general.name || general.id).join('、') || '-'} />
      </div>
      <div className="mt-3 flex justify-end gap-2">
        {canAccelerate && <button type="button" onClick={() => void onAccelerate(record)} className="inline-flex items-center gap-1 rounded-lg border border-amber-300 bg-amber-50 px-3 py-1.5 text-xs font-semibold text-amber-700 cursor-pointer hover:bg-amber-100"><Zap size={14} />加速</button>}
        {canRecall && <button type="button" onClick={() => void onRecall(record)} className="inline-flex items-center gap-1 rounded-lg border border-[var(--color-border)] px-3 py-1.5 text-xs font-semibold cursor-pointer hover:text-[var(--color-accent)]"><RotateCcw size={14} />召回</button>}
        {canExpel && <button type="button" onClick={() => void onExpel(record)} className="inline-flex items-center gap-1 rounded-lg border border-[var(--color-border)] px-3 py-1.5 text-xs font-semibold cursor-pointer hover:text-[var(--color-accent)]"><UserMinus size={14} />遣返</button>}
      </div>
    </article>
  )
}

const Metric: FC<{ label: string; value: string }> = ({ label, value }) => (
  <div className="rounded-lg bg-[var(--color-surface-dim)] px-3 py-2">
    <div className="text-[11px] text-[var(--color-text-muted)]">{label}</div>
    <div className="mt-1 truncate text-sm font-semibold text-[var(--color-text-primary)]">{value}</div>
  </div>
)

function formatTroops(troops: Record<string, number>) {
  const entries = Object.entries(troops ?? {}).filter(([, amount]) => amount > 0)
  if (entries.length === 0) return '-'
  return entries.map(([unit, amount]) => `${unit} ${amount}`).join('、')
}

// aggregateGarrisonDisplayRecords 将收到的驻防批次聚合成 UI 队伍：获得一队，增援按来源玩家一队。
function aggregateGarrisonDisplayRecords(activePlayerId: string, items: Reinforcement[]): Reinforcement[] {
  const merged = new Map<string, Reinforcement>()
  for (const item of items) {
    if (!ACTIVE_GARRISON_STATUSES.has(item.status)) continue
    const sourceType = normalizeDisplaySourceType(item.sourceType)
    const sourceOwner = item.ownerPlayerId || item.fromPlayerId || item.reinforcementId
    const key = sourceType === 'obtained' ? `obtained:${activePlayerId}` : `reinforcement:${sourceOwner}`
    const current = merged.get(key)
    if (!current) {
      merged.set(key, {
        ...item,
        sourceType,
        troops: { ...(item.troops ?? {}) },
        remainingTroops: { ...(item.remainingTroops ?? {}) },
      })
      continue
    }
    current.troops = mergeTroops(current.troops, item.troops)
    current.remainingTroops = mergeTroops(current.remainingTroops, item.remainingTroops)
    current.updatedAt = laterText(current.updatedAt, item.updatedAt)
    current.createdAt = earlierText(current.createdAt, item.createdAt)
    if ((STATUS_WEIGHT[item.status] ?? 0) > (STATUS_WEIGHT[current.status] ?? 0)) {
      current.status = item.status
    }
  }
  return Array.from(merged.values()).sort((a, b) => {
    const sourceOrder = Number((a.sourceType ?? 'reinforcement') !== 'obtained') - Number((b.sourceType ?? 'reinforcement') !== 'obtained')
    if (sourceOrder !== 0) return sourceOrder
    return (b.updatedAt || '').localeCompare(a.updatedAt || '')
  })
}

// normalizeDisplaySourceType 将历史驻防来源压成页面只展示的两类来源。
function normalizeDisplaySourceType(sourceType?: string): 'reinforcement' | 'obtained' {
  return sourceType === 'reinforcement' ? 'reinforcement' : 'obtained'
}

// mergeTroops 合并同兵种数量，避免同来源队伍按获得批次拆开。
function mergeTroops(a?: Record<string, number>, b?: Record<string, number>) {
  const result: Record<string, number> = { ...(a ?? {}) }
  for (const [unitType, amount] of Object.entries(b ?? {})) {
    if (amount > 0) result[unitType] = (result[unitType] ?? 0) + amount
  }
  return result
}

// readAcceleratedTimes 从增援 metadata 读取已加速次数。
function readAcceleratedTimes(metadata?: Record<string, unknown>) {
  const value = metadata?.acceleratedTimes
  return typeof value === 'number' && Number.isFinite(value) ? value : 0
}

// laterText 返回更晚的时间文本，用于聚合卡片排序。
function laterText(a?: string, b?: string) {
  if (!a) return b ?? ''
  if (!b) return a
  return a > b ? a : b
}

// earlierText 返回更早的时间文本，用于保留来源队伍创建时间。
function earlierText(a?: string, b?: string) {
  if (!a) return b ?? ''
  if (!b) return a
  return a < b ? a : b
}

export default ReinforcementsPage
