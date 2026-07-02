// 本文件实现左侧驻防队伍面板，按来源聚合展示增援与获得队伍。
import { useCallback, useEffect, useMemo, useState, type FC } from 'react'
import { ShieldPlus, UserMinus, UserRound } from 'lucide-react'
import { gameApi } from '@/api/game'
import { Modal } from '@/components/ui'
import { useGameStore } from '@/store/gameStore'
import { useConfigStore } from '@/store/configStore'
import type { Reinforcement, ReinforcementGeneralSnapshot } from '@/types/game'
import { sortUnitEntries } from '@/utils/unitOrder'
import { formatParamLabel, formatParamValue, getTraitMeta } from '@/utils/traits'

const STATUS_LABELS: Record<Reinforcement['status'], string> = {
  marching: '行军',
  stationed: '驻防',
  fighting: '战斗',
  returning: '返回',
  completed: '归档',
  cancelled: '取消',
  failed: '异常',
}

const ACTIVE_STATUSES = new Set<Reinforcement['status']>(['marching', 'stationed', 'fighting'])

const STATUS_WEIGHT: Partial<Record<Reinforcement['status'], number>> = {
  fighting: 5,
  stationed: 4,
  marching: 3,
  returning: 2,
  completed: 1,
  cancelled: 0,
  failed: 0,
}

const BUFF_LABELS: Record<string, string> = {
  attackBonus: '部队攻击',
  defenseBonus: '部队防御',
  infantryDefenseBonus: '步兵防御',
  cavalryDefenseBonus: '骑兵防御',
  marchSpeedBonus: '行军速度',
  recruitSpeedBonus: '征兵速度',
  buildSpeedBonus: '建造速度',
  productionBonus: '资源产量',
  woodProductionBonus: '木材产量',
  stoneProductionBonus: '石料产量',
  ironProductionBonus: '铁矿产量',
  foodProductionBonus: '粮食产量',
  capacityBonus: '仓库容量',
  exchangeRateBonus: '兑换收益',
}

const STAT_LABELS: Record<string, string> = {
  force: '武力',
  intelligence: '智谋',
  politics: '内政',
  command: '统率',
}

const STAT_ORDER = ['force', 'intelligence', 'politics', 'command']

interface GarrisonPanelProps {
  gameStateReady?: boolean
  compact?: boolean
}

const GarrisonPanel: FC<GarrisonPanelProps> = ({ gameStateReady = true, compact = false }) => {
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const patchState = useGameStore((s) => s.patchState)
  const units = useConfigStore((s) => s.units)
  const [records, setRecords] = useState<Reinforcement[]>([])
  const [expellingId, setExpellingId] = useState<string | null>(null)

  // loadGarrisonRecords 读取驻防队伍，真实增援保留批次用于单独遣返。
  const loadGarrisonRecords = useCallback(async () => {
    if (!activePlayerId || !gameStateReady) {
      setRecords([])
      return
    }
    try {
      const received = await gameApi.listReceivedReinforcements(activePlayerId)
      setRecords(aggregateGarrisonDisplayRecords(activePlayerId, received.items ?? []))
    } catch {
      setRecords([])
    }
  }, [activePlayerId, gameStateReady])

  useEffect(() => {
    void loadGarrisonRecords()
    window.addEventListener('hero3:garrison-updated', loadGarrisonRecords)
    const timer = window.setInterval(loadGarrisonRecords, 60_000)
    return () => {
      window.removeEventListener('hero3:garrison-updated', loadGarrisonRecords)
      window.clearInterval(timer)
    }
  }, [loadGarrisonRecords])

  // handleExpelReinforcement 遣返单支别人派来的增援。
  const handleExpelReinforcement = async (record: Reinforcement) => {
    if (!activePlayerId || expellingId) return
    setExpellingId(record.reinforcementId)
    try {
      const result = await gameApi.expelReinforcement(activePlayerId, record.reinforcementId)
      if (result.patch) patchState(result.patch)
      await loadGarrisonRecords()
    } finally {
      setExpellingId(null)
    }
  }

  const totalTroops = useMemo(() => {
    return records.reduce((sum, record) => sum + Object.values(record.remainingTroops ?? {}).reduce((inner, amount) => inner + amount, 0), 0)
  }, [records])

  return (
    <section className={`rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-3 ${compact ? '' : 'mb-2.5'}`}>
      <div className="flex items-center gap-2 mb-2">
        <ShieldPlus size={14} className="text-emerald-600" />
        <span className="text-sm font-semibold text-[var(--color-text-primary)]">驻防队伍</span>
        <span className="ml-auto text-xs font-semibold text-emerald-600">{totalTroops.toLocaleString()}</span>
      </div>
      {records.length > 0 ? (
        <div className="space-y-1.5">
          {records.slice(0, 5).map((record) => (
            <GarrisonCard
              key={record.reinforcementId}
              record={record}
              units={units}
              expelling={expellingId === record.reinforcementId}
              onExpel={handleExpelReinforcement}
            />
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

// aggregateGarrisonDisplayRecords 将驻防批次聚合成 UI 队伍；真实增援按批次展示，获得队伍才聚合。
function aggregateGarrisonDisplayRecords(activePlayerId: string, items: Reinforcement[]): Reinforcement[] {
  const merged = new Map<string, Reinforcement>()
  for (const item of items) {
    if (!ACTIVE_STATUSES.has(item.status)) continue
    const sourceType = normalizeDisplaySourceType(item.sourceType)
    const key = sourceType === 'obtained' ? `obtained:${activePlayerId}` : `reinforcement:${item.reinforcementId}`
    const current = merged.get(key)
    if (!current) {
      merged.set(key, {
        ...item,
        sourceType,
        remainingTroops: { ...(item.remainingTroops ?? {}) },
        troops: { ...(item.troops ?? {}) },
        generals: [...(item.generals ?? [])],
      })
      continue
    }
    current.troops = mergeTroops(current.troops, item.troops)
    current.remainingTroops = mergeTroops(current.remainingTroops, item.remainingTroops)
    current.generals = mergeGenerals(current.generals, item.generals)
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

// mergeGenerals 合并同来源驻防的携带武将，按 ID 去重。
function mergeGenerals(a?: Reinforcement['generals'], b?: Reinforcement['generals']) {
  const result = [...(a ?? [])]
  const seen = new Set(result.map((general) => general.id))
  for (const general of b ?? []) {
    if (!general.id || seen.has(general.id)) continue
    seen.add(general.id)
    result.push(general)
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

const GarrisonCard: FC<{
  record: Reinforcement
  units: ReturnType<typeof useConfigStore.getState>['units']
  expelling: boolean
  onExpel: (record: Reinforcement) => Promise<void>
}> = ({ record, units, expelling, onExpel }) => {
  const sourceType = normalizeDisplaySourceType(record.sourceType)
  const total = Object.values(record.remainingTroops ?? {}).reduce((sum, amount) => sum + amount, 0)
  const troopEntries = sortUnitEntries(record.remainingTroops, record.fromPlayerFaction, units ?? undefined).filter(([, amount]) => amount > 0)
  const title = sourceType === 'obtained' ? '自己' : `增援 · ${record.fromPlayerName || record.fromPlayerId || '未知'}`
  const generals = record.generals ?? []
  const canExpel = sourceType === 'reinforcement' && (record.status === 'marching' || record.status === 'stationed')
  const [selectedGeneral, setSelectedGeneral] = useState<ReinforcementGeneralSnapshot | null>(null)

  return (
    <>
      <div className="rounded-lg border border-[var(--color-border)] bg-white/60 px-2.5 py-2 dark:bg-white/5">
        <div className="flex items-center gap-1.5">
          <span className="min-w-0 flex-1 text-[10px] font-bold text-emerald-600">{title}</span>
          <span className="shrink-0 text-[10px] font-semibold text-emerald-600">{total.toLocaleString()}</span>
        </div>
        {generals.length > 0 && (
          <div className="mt-1.5 space-y-1">
            {generals.map((general) => (
              <button
                key={general.id}
                type="button"
                onClick={() => setSelectedGeneral(general)}
                className="
                  flex w-full items-center gap-2 text-left text-[10px]
                  cursor-pointer transition-colors
                "
                title="查看驻防武将信息"
              >
                <span className="min-w-0 flex-1 truncate font-bold text-emerald-600 hover:text-emerald-500">
                  {general.name || general.id}
                </span>
                {general.level ? <span className="shrink-0 font-semibold text-[var(--color-text-muted)]">Lv.{general.level}</span> : null}
              </button>
            ))}
          </div>
        )}
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
        <div className="mt-1 flex items-center justify-end gap-2 text-[9px] text-[var(--color-text-muted)]">
          <span className="min-w-0 flex-1" />
          <span className="shrink-0">{STATUS_LABELS[record.status]}</span>
          {canExpel && (
            <button
              type="button"
              onClick={() => void onExpel(record)}
              disabled={expelling}
              className="
                inline-flex shrink-0 items-center gap-1 rounded-lg border border-[var(--color-border)]
                px-2 py-0.5 text-[10px] font-semibold text-[var(--color-text-secondary)]
                hover:border-red-400 hover:text-red-500
                disabled:cursor-not-allowed disabled:opacity-50 cursor-pointer transition-colors
              "
              title="遣返这支援军"
            >
              <UserMinus size={10} />
              {expelling ? '遣返中' : '遣返'}
            </button>
          )}
        </div>
      </div>
      <GarrisonGeneralModal general={selectedGeneral} onClose={() => setSelectedGeneral(null)} />
    </>
  )
}

// GarrisonGeneralModal 展示驻防武将快照详情。
const GarrisonGeneralModal: FC<{ general: ReinforcementGeneralSnapshot | null; onClose: () => void }> = ({ general, onClose }) => {
  const statEntries = STAT_ORDER.map((key) => [key, general?.stats?.[key] ?? 0] as const)
  const attributeEntries = Object.entries(general?.attributes ?? general?.buffs ?? {}).filter(([, value]) => value !== 0)
  const traits = general?.traits ?? []
  return (
    <Modal open={Boolean(general)} onClose={onClose} title="驻防武将信息" width="max-w-sm">
      {general && (
        <div className="space-y-4">
          <div className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-3">
            <div className="flex items-start gap-3">
              <div className="grid h-10 w-10 shrink-0 place-items-center rounded-lg border border-emerald-500/30 bg-emerald-500/10 text-emerald-600">
                <UserRound size={18} />
              </div>
              <div className="min-w-0 flex-1">
                <div className="truncate text-sm font-bold text-[var(--color-text-primary)]">{general.name || general.id}</div>
                <div className="mt-1 flex flex-wrap gap-1.5 text-[10px] text-[var(--color-text-muted)]">
                  {general.level ? <span>等级 Lv.{general.level}</span> : null}
                  {general.assignment ? <span>{formatGeneralAssignment(general.assignment)}</span> : null}
                </div>
              </div>
            </div>
          </div>

          <div>
            <div className="mb-2 text-xs font-semibold text-[var(--color-text-secondary)]">四维</div>
            <div className="grid grid-cols-4 gap-2">
              {statEntries.map(([key, value]) => (
                <div key={key} className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-2 py-2 text-center">
                  <div className="text-[10px] text-[var(--color-text-muted)]">{STAT_LABELS[key]}</div>
                  <div className="mt-0.5 text-xs font-bold text-[var(--color-text-primary)]">{value}</div>
                </div>
              ))}
            </div>
          </div>

          <div>
            <div className="mb-2 text-xs font-semibold text-[var(--color-text-secondary)]">武将特性</div>
            {traits.length > 0 ? (
              <div className="space-y-2">
                {traits.map((trait) => {
                  const meta = getTraitMeta(trait.traitId)
                  return (
                    <div key={trait.traitId} className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2">
                      <div className="flex items-center gap-2">
                        <span className="text-sm">{meta.icon}</span>
                        <span className="text-xs font-bold text-emerald-600">{trait.name || meta.name}</span>
                        {meta.trigger && <span className="ml-auto text-[10px] text-[var(--color-text-muted)]">{meta.trigger}</span>}
                      </div>
                      {meta.description && <div className="mt-1 text-[10px] text-[var(--color-text-secondary)]">{meta.description}</div>}
                      {Object.keys(trait.params ?? {}).length > 0 && (
                        <div className="mt-1 flex flex-wrap gap-1">
                          {Object.entries(trait.params).map(([key, value]) => (
                            <span key={key} className="text-[10px] text-[var(--color-text-muted)]">
                              {formatParamLabel(key)} {formatParamValue(key, value)}
                            </span>
                          ))}
                        </div>
                      )}
                    </div>
                  )
                })}
              </div>
            ) : (
              <div className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-3 text-center text-xs text-[var(--color-text-muted)]">
                暂无可展示特性
              </div>
            )}
          </div>

          <div>
            <div className="mb-2 text-xs font-semibold text-[var(--color-text-secondary)]">携带加成</div>
            {attributeEntries.length > 0 ? (
              <div className="grid grid-cols-2 gap-2">
                {attributeEntries.map(([key, value]) => (
                  <div key={key} className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-2.5 py-2">
                    <div className="truncate text-[10px] text-[var(--color-text-muted)]">{BUFF_LABELS[key] ?? key}</div>
                    <div className="mt-0.5 text-xs font-bold text-emerald-600">{formatBuffValue(value)}</div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-4 text-center text-xs text-[var(--color-text-muted)]">
                暂无可展示加成
              </div>
            )}
          </div>
        </div>
      )}
    </Modal>
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

// formatBuffValue 格式化驻防武将加成。
function formatBuffValue(value: number) {
  return `${value >= 0 ? '+' : ''}${Math.round(value * 100)}%`
}

// formatGeneralAssignment 格式化武将占用位置。
function formatGeneralAssignment(value: string) {
  if (value === 'reinforcement' || value.startsWith('reinforcement_')) return '驻防中'
  return value
}

export default GarrisonPanel
