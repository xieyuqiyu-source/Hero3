// 本文件实现增援与驻防队伍页面，收到的真实增援按批次展示以支持分别遣返。
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

const ACTIVE_SENT_STATUSES = new Set<Reinforcement['status']>(['marching', 'stationed', 'fighting', 'returning'])
const ACTIVE_GARRISON_STATUSES = new Set<Reinforcement['status']>(['marching', 'stationed', 'fighting'])
const REINFORCEMENT_ACCELERATE_COST = 10

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

  const units = useConfigStore((s) => s.units)
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
      window.dispatchEvent(new Event('hero3:marches-updated'))
      window.dispatchEvent(new Event('hero3:garrison-updated'))
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
    setError(null)
    try {
      const result = await gameApi.accelerateReinforcement(activePlayerId, record.reinforcementId)
      if (result.patch) patchState(result.patch)
      patchState({
        ...(typeof result.cityGold === 'number' ? { cityGold: result.cityGold } : {}),
        ...(result.serverTime ? { serverTime: result.serverTime } : {}),
      })
      await refresh()
      window.dispatchEvent(new Event('hero3:marches-updated'))
    } catch (err) {
      const message = err instanceof Error ? err.message : ''
      setError(message.includes('insufficient city gold') ? `城金不足，本次加速需要 ${REINFORCEMENT_ACCELERATE_COST} 城金` : (message || '增援加速失败'))
    }
  }

  const sentDisplay = useMemo(() => buildSentDisplayRecords(sent), [sent])
  const receivedDisplay = useMemo(() => buildReceivedDisplayRecords(received), [received])
  const items = tab === 'sent' ? sentDisplay : receivedDisplay
  const cityGold = Number(state?.cityGold ?? 0)

  return (
    <div className="min-w-0 space-y-5 overflow-hidden">
      <div className="flex flex-wrap items-center gap-2">
        <button type="button" onClick={() => setTab('sent')} className={`px-4 py-2 rounded-lg border text-sm font-semibold cursor-pointer ${tab === 'sent' ? 'bg-[var(--color-accent-light)] border-[var(--color-accent-border)] text-[var(--color-accent)]' : 'border-[var(--color-border)] text-[var(--color-text-secondary)]'}`}>
          我派出的
        </button>
        <button type="button" onClick={() => setTab('received')} className={`px-4 py-2 rounded-lg border text-sm font-semibold cursor-pointer ${tab === 'received' ? 'bg-[var(--color-accent-light)] border-[var(--color-accent-border)] text-[var(--color-accent)]' : 'border-[var(--color-border)] text-[var(--color-text-secondary)]'}`}>
          驻防队伍
        </button>
        <button type="button" onClick={() => void refresh()} className="ml-auto inline-flex h-9 w-9 items-center justify-center rounded-lg border border-[var(--color-border)] text-[var(--color-text-secondary)] cursor-pointer hover:text-[var(--color-accent)]">
          <RotateCcw size={16} />
        </button>
      </div>

      {error && <div className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">{error}</div>}

      <div className="grid min-w-0 gap-4 xl:grid-cols-[320px_minmax(0,1fr)]">
        <section className="min-w-0 space-y-3 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] p-4">
          <div className="flex items-center gap-2 text-sm font-bold text-[var(--color-text-primary)]">
            <Send size={16} className="text-[var(--color-accent)]" />
            发起增援
          </div>
          <input value={targetPlayerId} onChange={(e) => setTargetPlayerId(e.target.value)} placeholder="目标玩家 ID" className="w-full rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-sm outline-none focus:border-[var(--color-accent-border)]" />
          <div className="grid min-w-0 grid-cols-[minmax(0,1fr)_88px] gap-2">
            <select value={selectedUnit} onChange={(e) => setSelectedUnit(e.target.value)} className="min-w-0 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-sm outline-none">
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

        <section className="grid min-w-0 gap-3">
          {items.map((record) => (
            <ReinforcementCard key={record.reinforcementId} record={record} mode={tab} units={units} cityGold={cityGold} onRecall={handleRecall} onExpel={handleExpel} onAccelerate={handleAccelerate} />
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
  units: ReturnType<typeof useConfigStore.getState>['units']
  cityGold: number
  onRecall: (record: Reinforcement) => Promise<void>
  onExpel: (record: Reinforcement) => Promise<void>
  onAccelerate: (record: Reinforcement) => Promise<void>
}> = ({ record, mode, units, cityGold, onRecall, onExpel, onAccelerate }) => {
  const sourceType = normalizeDisplaySourceType(record.sourceType)
  const sourceTitle = SOURCE_LABELS[sourceType]
  const sourceName = sourceType === 'obtained' ? '自己' : (record.fromPlayerName || record.fromPlayerId)
  const name = mode === 'sent' ? (record.toPlayerName || record.toPlayerId) : `${sourceTitle} · ${sourceName}`
  const canRecall = mode === 'sent' && (record.status === 'marching' || record.status === 'stationed')
  const canAccelerate = mode === 'sent' && record.status === 'marching' && readAcceleratedTimes(record.metadata) < 2
  const canPayAccelerate = cityGold >= REINFORCEMENT_ACCELERATE_COST
  const canExpel = mode === 'received' && sourceType === 'reinforcement' && (record.status === 'marching' || record.status === 'stationed')
  return (
    <article className="min-w-0 overflow-hidden rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] p-4">
      <div className="flex min-w-0 flex-wrap items-center gap-2">
        <span className="min-w-0 max-w-full truncate text-sm font-bold text-[var(--color-text-primary)]">{name}</span>
        <span className="rounded bg-[var(--color-accent-light)] px-2 py-0.5 text-xs font-semibold text-[var(--color-accent)]">{STATUS_LABELS[record.status]}</span>
        <span className="ml-auto shrink-0 text-xs text-[var(--color-text-muted)]">{formatShortDateTime(record.arriveAt || record.expectedReturnedAt || record.createdAt)}</span>
      </div>
      <div className="mt-3 grid min-w-0 gap-2 md:grid-cols-3">
        <Metric label="派出兵力" value={formatTroops(record.troops, units, record.fromPlayerFaction)} />
        <Metric label="剩余兵力" value={formatTroops(record.remainingTroops, units, record.fromPlayerFaction)} />
        <Metric label="携带武将" value={(record.generals ?? []).map((general) => general.name || general.id).join('、') || '-'} />
      </div>
      <div className="mt-3 flex flex-wrap justify-end gap-2">
        {canAccelerate && (
          <button
            type="button"
            onClick={() => void onAccelerate(record)}
            disabled={!canPayAccelerate}
            className="inline-flex items-center gap-1 rounded-lg border border-amber-300 bg-amber-50 px-3 py-1.5 text-xs font-semibold text-amber-700 cursor-pointer hover:bg-amber-100 disabled:cursor-not-allowed disabled:opacity-50"
            title={canPayAccelerate ? `消耗 ${REINFORCEMENT_ACCELERATE_COST} 城金，将剩余时间减半` : `城金不足，需要 ${REINFORCEMENT_ACCELERATE_COST} 城金`}
          >
            <Zap size={14} />加速 {REINFORCEMENT_ACCELERATE_COST}
          </button>
        )}
        {canRecall && <button type="button" onClick={() => void onRecall(record)} className="inline-flex items-center gap-1 rounded-lg border border-[var(--color-border)] px-3 py-1.5 text-xs font-semibold cursor-pointer hover:text-[var(--color-accent)]"><RotateCcw size={14} />召回</button>}
        {canExpel && <button type="button" onClick={() => void onExpel(record)} className="inline-flex items-center gap-1 rounded-lg border border-[var(--color-border)] px-3 py-1.5 text-xs font-semibold cursor-pointer hover:text-[var(--color-accent)]"><UserMinus size={14} />遣返</button>}
      </div>
    </article>
  )
}

const Metric: FC<{ label: string; value: string }> = ({ label, value }) => (
  <div className="min-w-0 rounded-lg bg-[var(--color-surface-dim)] px-3 py-2">
    <div className="text-[11px] text-[var(--color-text-muted)]">{label}</div>
    <div className="mt-1 break-words text-sm font-semibold text-[var(--color-text-primary)]">{value}</div>
  </div>
)

// formatTroops 格式化增援兵力，优先按来源阵营翻译兵种名称。
function formatTroops(troops: Record<string, number>, units: ReturnType<typeof useConfigStore.getState>['units'], faction?: string) {
  const entries = Object.entries(troops ?? {}).filter(([, amount]) => amount > 0)
  if (entries.length === 0) return '-'
  return entries.map(([unit, amount]) => `${findUnitName(units, unit, faction)} ${amount.toLocaleString()}`).join('、')
}

// findUnitName 根据兵种 ID 查找展示名，兼容跨阵营援军和活动兵种。
function findUnitName(units: ReturnType<typeof useConfigStore.getState>['units'], unitType: string, faction?: string): string {
  if (faction && units?.[faction]?.[unitType]?.name) return units[faction][unitType].name
  for (const factionUnits of Object.values(units ?? {})) {
    if (factionUnits[unitType]?.name) return factionUnits[unitType].name
  }
  return unitType
}

// formatShortDateTime 缩短增援时间展示，避免嵌入军事页后撑开卡片。
function formatShortDateTime(value?: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hour = String(date.getHours()).padStart(2, '0')
  const minute = String(date.getMinutes()).padStart(2, '0')
  return `${month}-${day} ${hour}:${minute}`
}

// buildSentDisplayRecords 展示我派出的进行中增援；完成归档不作为玩家记录展示。
function buildSentDisplayRecords(items: Reinforcement[]): Reinforcement[] {
  return items.filter((item) => ACTIVE_SENT_STATUSES.has(item.status))
}

// buildReceivedDisplayRecords 展示收到的活跃驻防；真实增援不再按来源合并，便于逐支遣返。
function buildReceivedDisplayRecords(items: Reinforcement[]): Reinforcement[] {
  return items
    .filter((item) => ACTIVE_GARRISON_STATUSES.has(item.status))
    .map((item) => ({ ...item, sourceType: normalizeDisplaySourceType(item.sourceType) }))
}

// normalizeDisplaySourceType 将历史驻防来源压成页面只展示的两类来源。
function normalizeDisplaySourceType(sourceType?: string): 'reinforcement' | 'obtained' {
  return sourceType === 'reinforcement' ? 'reinforcement' : 'obtained'
}

// readAcceleratedTimes 从增援 metadata 读取已加速次数。
function readAcceleratedTimes(metadata?: Record<string, unknown>) {
  const value = metadata?.acceleratedTimes
  return typeof value === 'number' && Number.isFinite(value) ? value : 0
}

export default ReinforcementsPage
