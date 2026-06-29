// 本文件实现左侧驻防队伍面板，按来源聚合展示增援与获得队伍。
import { useEffect, useMemo, useState, type FC } from 'react'
import { ShieldPlus } from 'lucide-react'
import { gameApi } from '@/api/game'
import { useGameStore } from '@/store/gameStore'
import { useConfigStore } from '@/store/configStore'
import type { Reinforcement } from '@/types/game'
import { sortUnitEntries } from '@/utils/unitOrder'

const SOURCE_LABELS: Record<'reinforcement' | 'obtained', string> = {
  reinforcement: '增援',
  obtained: '获得',
}

const STATUS_LABELS: Record<Reinforcement['status'], string> = {
  marching: '行军',
  stationed: '驻防',
  fighting: '战斗',
  returning: '返回',
  completed: '归档',
  cancelled: '取消',
  failed: '异常',
}

const ACTIVE_STATUSES = new Set<Reinforcement['status']>(['marching', 'stationed', 'fighting', 'returning'])

const STATUS_WEIGHT: Partial<Record<Reinforcement['status'], number>> = {
  fighting: 5,
  stationed: 4,
  marching: 3,
  returning: 2,
  completed: 1,
  cancelled: 0,
  failed: 0,
}

interface GarrisonPanelProps {
  gameStateReady?: boolean
  compact?: boolean
}

const GarrisonPanel: FC<GarrisonPanelProps> = ({ gameStateReady = true, compact = false }) => {
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const units = useConfigStore((s) => s.units)
  const [records, setRecords] = useState<Reinforcement[]>([])

  useEffect(() => {
    let cancelled = false
    const load = async () => {
      if (!activePlayerId || !gameStateReady) {
        setRecords([])
        return
      }
      try {
        const received = await gameApi.listReceivedReinforcements(activePlayerId)
        if (cancelled) return
        setRecords(aggregateGarrisonDisplayRecords(activePlayerId, received.items ?? []))
      } catch {
        if (!cancelled) setRecords([])
      }
    }
    void load()
    window.addEventListener('hero3:garrison-updated', load)
    const timer = window.setInterval(load, 60_000)
    return () => {
      cancelled = true
      window.removeEventListener('hero3:garrison-updated', load)
      window.clearInterval(timer)
    }
  }, [activePlayerId, gameStateReady])

  const totalTroops = useMemo(() => {
    return records.reduce((sum, record) => sum + Object.values(record.remainingTroops ?? {}).reduce((inner, amount) => inner + amount, 0), 0)
  }, [records])

  return (
    <section className={`rounded-2xl border border-emerald-500/25 bg-emerald-500/10 p-3 shadow-[0_4px_12px_rgba(16,185,129,0.08)] ${compact ? '' : 'mb-2.5'}`}>
      <div className="flex items-center gap-2 mb-2">
        <ShieldPlus size={14} className="text-emerald-600" />
        <span className="text-sm font-semibold text-[var(--color-text-primary)]">驻防队伍</span>
        <span className="ml-auto text-xs font-semibold text-emerald-600">{totalTroops.toLocaleString()}</span>
      </div>
      {records.length > 0 ? (
        <div className="space-y-1.5">
          {records.slice(0, 5).map((record) => (
            <GarrisonCard key={record.reinforcementId} record={record} units={units} />
          ))}
          {records.length > 5 && (
            <div className="px-2 py-1 text-[10px] text-[var(--color-text-muted)] text-center">
              另有 {records.length - 5} 队
            </div>
          )}
        </div>
      ) : (
        <p className="text-xs text-[var(--color-text-secondary)] opacity-50">暂无驻防队伍</p>
      )}
    </section>
  )
}

// aggregateGarrisonDisplayRecords 将驻防批次聚合成 UI 来源队伍：获得一队，增援按派出玩家一队。
function aggregateGarrisonDisplayRecords(activePlayerId: string, items: Reinforcement[]): Reinforcement[] {
  const merged = new Map<string, Reinforcement>()
  for (const item of items) {
    if (!ACTIVE_STATUSES.has(item.status)) continue
    const sourceType = normalizeDisplaySourceType(item.sourceType)
    const sourceOwner = item.ownerPlayerId || item.fromPlayerId || item.reinforcementId
    const key = sourceType === 'obtained' ? `obtained:${activePlayerId}` : `reinforcement:${sourceOwner}`
    const current = merged.get(key)
    if (!current) {
      merged.set(key, {
        ...item,
        sourceType,
        remainingTroops: { ...(item.remainingTroops ?? {}) },
        troops: { ...(item.troops ?? {}) },
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

// normalizeDisplaySourceType 将历史来源统一为面板的两类来源。
function normalizeDisplaySourceType(sourceType?: string): 'reinforcement' | 'obtained' {
  return sourceType === 'reinforcement' ? 'reinforcement' : 'obtained'
}

// mergeTroops 合并同兵种数量，保证同来源卡片内不再按批次拆开。
function mergeTroops(a?: Record<string, number>, b?: Record<string, number>) {
  const result: Record<string, number> = { ...(a ?? {}) }
  for (const [unitType, amount] of Object.entries(b ?? {})) {
    if (amount > 0) result[unitType] = (result[unitType] ?? 0) + amount
  }
  return result
}

// laterText 返回较新的时间文本，用于聚合卡片排序。
function laterText(a?: string, b?: string) {
  if (!a) return b ?? ''
  if (!b) return a
  return a > b ? a : b
}

// earlierText 返回较早的时间文本，用于保留来源队伍创建时间。
function earlierText(a?: string, b?: string) {
  if (!a) return b ?? ''
  if (!b) return a
  return a < b ? a : b
}

const GarrisonCard: FC<{ record: Reinforcement; units: ReturnType<typeof useConfigStore.getState>['units'] }> = ({ record, units }) => {
  const sourceType = normalizeDisplaySourceType(record.sourceType)
  const total = Object.values(record.remainingTroops ?? {}).reduce((sum, amount) => sum + amount, 0)
  const troopEntries = sortUnitEntries(record.remainingTroops, record.fromPlayerFaction, units ?? undefined).filter(([, amount]) => amount > 0)
  const title = SOURCE_LABELS[sourceType]

  return (
    <div className="rounded-xl border border-emerald-500/20 bg-white/70 px-2.5 py-2 dark:bg-emerald-500/10">
      <div className="flex items-center gap-1.5">
        <span className="min-w-0 flex-1 text-[10px] font-bold text-emerald-600">{title}</span>
        <span className="shrink-0 text-[10px] font-semibold text-emerald-600">{total.toLocaleString()}</span>
      </div>
      <div className="mt-1.5 space-y-1">
        {troopEntries.length > 0 ? troopEntries.map(([unitType, amount]) => (
          <div key={unitType} className="flex items-center gap-2 text-[10px]">
            <span className="min-w-0 flex-1 truncate text-[var(--color-text-secondary)]">{findUnitName(units, unitType, record.fromPlayerFaction)}</span>
            <span className="shrink-0 font-semibold text-[var(--color-text-primary)]">{amount.toLocaleString()}</span>
          </div>
        )) : (
          <div className="text-[10px] text-[var(--color-text-muted)]">空队</div>
        )}
      </div>
      <div className="mt-1 flex justify-end text-[9px] text-[var(--color-text-muted)]">
        <span>{STATUS_LABELS[record.status]}</span>
      </div>
    </div>
  )
}

// findUnitName 根据兵种 ID 查找展示名，兼容跨阵营驻防兵种。
function findUnitName(units: ReturnType<typeof useConfigStore.getState>['units'], unitType: string, faction?: string): string {
  if (faction && units?.[faction]?.[unitType]?.name) return units[faction][unitType].name
  for (const factionUnits of Object.values(units ?? {})) {
    if (factionUnits[unitType]?.name) return factionUnits[unitType].name
  }
  return unitType
}

export default GarrisonPanel
