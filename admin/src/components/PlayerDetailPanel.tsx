// 玩家详情弹层用于查看核心资产权威状态并执行轻量 GM 调试。
import { useCallback, useEffect, useState } from 'react'
import { X } from 'lucide-react'
import { adminApi } from '@/api/admin'
import type { GameState } from '@/types'
import ResourceAdjustForm from '@/components/ResourceAdjustForm'
import WorldPositionPanel from '@/components/WorldPositionPanel'

interface PlayerDetailPanelProps {
  playerId: string
  onClose: () => void
}

// PlayerDetailPanel 展示单个玩家的资源、建筑、军队、武将和通用资产状态。
export default function PlayerDetailPanel({ playerId, onClose }: PlayerDetailPanelProps) {
  const [state, setState] = useState<GameState | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // loadState 从 Admin API 拉取玩家当前权威状态。
  const loadState = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const result = await adminApi.getPlayerState(playerId)
      setState(result)
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }, [playerId])

  useEffect(() => {
    queueMicrotask(() => {
      void loadState()
    })
  }, [loadState])

  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center bg-slate-900/40 backdrop-blur-sm p-4" onClick={onClose}>
      <div
        className="w-full max-w-[720px] max-h-[calc(100dvh-48px)] overflow-y-auto rounded-3xl border border-[var(--color-border)] bg-[var(--color-surface)] shadow-[0_24px_68px_rgba(15,23,42,0.24)] p-5"
        onClick={(e) => e.stopPropagation()}
      >
        {loading && (
          <div className="py-12 text-center text-sm text-[var(--color-text-secondary)]">加载中...</div>
        )}

        {error && (
          <div className="py-12 text-center">
            <p className="text-sm text-red-600">{error}</p>
            <button type="button" onClick={onClose} className="mt-3 px-4 py-2 rounded-xl border border-[var(--color-border)] text-sm cursor-pointer">关闭</button>
          </div>
        )}

        {!loading && !error && state && (
          <>
            {/* Header */}
            <div className="flex items-center justify-between mb-5">
              <div>
                <h3 className="text-lg font-bold text-[var(--color-text-primary)]">{state.player.nickname}</h3>
                <span className="text-xs text-[var(--color-text-muted)]">{state.player.id} · {state.player.faction}</span>
              </div>
              <button type="button" onClick={onClose} className="w-8 h-8 flex items-center justify-center rounded-xl border border-[var(--color-border)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] cursor-pointer transition-colors">
                <X size={16} />
              </button>
            </div>

            {/* Resources */}
            <section className="mb-4 p-3.5 rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
              <h4 className="text-xs font-bold text-[var(--color-text-secondary)] uppercase tracking-wider mb-2.5">资源</h4>
              <div className="grid grid-cols-2 sm:grid-cols-4 gap-2">
                {Object.entries(state.resources.items).map(([res, amount]) => (
                  <div key={res} className="px-2.5 py-2 rounded-xl bg-white/70 dark:bg-white/5 border border-[var(--color-border)]">
                    <span className="text-[10px] text-[var(--color-text-muted)] uppercase">{res}</span>
                    <strong className="block text-sm font-bold text-[var(--color-text-primary)]">{amount.toLocaleString()}</strong>
                    <small className="text-[10px] text-[var(--color-text-muted)]">/ {state.resources.capacity[res]?.toLocaleString() ?? '--'}</small>
                  </div>
                ))}
              </div>
              <div className="flex flex-wrap gap-2 mt-2.5 text-xs text-[var(--color-text-secondary)]">
                {Object.entries(state.resourceProduction).map(([res, rate]) => (
                  <span key={res}>{res} +{rate}/h</span>
                ))}
              </div>
            </section>

            {/* Resource Adjust */}
            <ResourceAdjustForm playerId={playerId} onSuccess={(s) => setState(s)} />

            {/* 世界坐标 */}
            <WorldPositionPanel playerId={playerId} />

            {/* Inventory */}
            {state.inventory && Object.keys(state.inventory).length > 0 && (
              <section className="mb-4 p-3.5 rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
                <h4 className="text-xs font-bold text-[var(--color-text-secondary)] uppercase tracking-wider mb-2.5">背包 ({Object.keys(state.inventory).length})</h4>
                <div className="flex flex-wrap gap-1.5">
                  {Object.entries(state.inventory).map(([itemId, stack]) => (
                    <span key={itemId} className="px-2 py-1 rounded-lg text-[11px] font-bold bg-sky-500/10 text-sky-700">
                      {itemId} ×{stack.amount}
                    </span>
                  ))}
                </div>
              </section>
            )}

            {/* Buildings */}
            <section className="mb-4 p-3.5 rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
              <h4 className="text-xs font-bold text-[var(--color-text-secondary)] uppercase tracking-wider mb-2.5">建筑 ({state.buildings.length})</h4>
              <div className="flex flex-wrap gap-1.5">
                {state.buildings.map((b) => (
                  <span key={b.id} className="px-2 py-1 rounded-lg text-[11px] font-bold bg-[var(--color-gold-soft)] text-amber-700">
                    {b.type} Lv.{b.level}{b.upgradeEndsAt ? ' ⏳' : ''}
                  </span>
                ))}
              </div>
            </section>

            {/* Resource Slots */}
            {state.resourceSlots && state.resourceSlots.length > 0 && (
              <section className="mb-4 p-3.5 rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
                <h4 className="text-xs font-bold text-[var(--color-text-secondary)] uppercase tracking-wider mb-2.5">资源田格子 ({state.resourceSlots.length})</h4>
                <div className="flex flex-wrap gap-1.5">
                  {state.resourceSlots.map((slot) => (
                    <span key={slot.id} className="px-2 py-1 rounded-lg text-[11px] font-bold bg-emerald-500/10 text-emerald-700">
                      {slot.resourceType} · {slot.buildingId || slot.id}
                    </span>
                  ))}
                </div>
              </section>
            )}

            {/* Generals */}
            {state.generals && state.generals.length > 0 && (
              <section className="mb-4 p-3.5 rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
                <h4 className="text-xs font-bold text-[var(--color-text-secondary)] uppercase tracking-wider mb-2.5">武将 ({state.generals.length})</h4>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                  {state.generals.map((general) => (
                    <div key={general.id} className="px-2.5 py-2 rounded-xl bg-white/70 dark:bg-white/5 border border-[var(--color-border)]">
                      <div className="flex items-center justify-between gap-2">
                        <span className="text-xs font-bold text-[var(--color-text-primary)]">{general.name || general.id}</span>
                        <span className="text-[10px] text-[var(--color-text-muted)]">Lv.{general.level}</span>
                      </div>
                      <div className="mt-1 text-[10px] text-[var(--color-text-muted)]">
                        EXP {general.exp.toLocaleString()} · 点数 {general.availableStatPoints ?? 0}
                      </div>
                    </div>
                  ))}
                </div>
              </section>
            )}

            {/* General Assignments */}
            {state.generalAssignments && state.generalAssignments.length > 0 && (
              <section className="mb-4 p-3.5 rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
                <h4 className="text-xs font-bold text-[var(--color-text-secondary)] uppercase tracking-wider mb-2.5">武将派驻 ({state.generalAssignments.length})</h4>
                <div className="space-y-1.5">
                  {state.generalAssignments.map((assignment) => (
                    <div key={assignment.id} className="flex items-center justify-between gap-2 px-2.5 py-1.5 rounded-lg bg-white/70 dark:bg-white/5 border border-[var(--color-border)]">
                      <span className="text-[11px] text-[var(--color-text-primary)] font-bold">{assignment.generalId}</span>
                      <span className="text-[10px] text-[var(--color-text-muted)]">{assignment.slot} · {assignment.status ?? 'active'}</span>
                    </div>
                  ))}
                </div>
              </section>
            )}

            {/* Buffs */}
            {state.buffs && state.buffs.length > 0 && (
              <section className="mb-4 p-3.5 rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
                <h4 className="text-xs font-bold text-[var(--color-text-secondary)] uppercase tracking-wider mb-2.5">Buff ({state.buffs.length})</h4>
                <div className="space-y-1.5">
                  {state.buffs.map((buff) => (
                    <div key={buff.id} className="flex items-center justify-between gap-2 px-2.5 py-1.5 rounded-lg bg-white/70 dark:bg-white/5 border border-[var(--color-border)]">
                      <span className="text-[11px] text-[var(--color-text-primary)] font-bold">{buff.key} {buff.mode}</span>
                      <span className="text-[10px] text-[var(--color-text-muted)]">{buff.value}</span>
                    </div>
                  ))}
                </div>
              </section>
            )}

            {/* Army */}
            <section className="mb-4 p-3.5 rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
              <h4 className="text-xs font-bold text-[var(--color-text-secondary)] uppercase tracking-wider mb-2.5">军队</h4>
              <div className="grid grid-cols-2 sm:grid-cols-4 gap-2">
                {state.army.map((unit) => (
                  <div key={unit.unitType} className="px-2.5 py-2 rounded-xl bg-white/70 dark:bg-white/5 border border-[var(--color-border)]">
                    <span className="text-[10px] text-[var(--color-text-muted)]">{unit.unitType}</span>
                    <strong className="block text-sm font-bold text-[var(--color-text-primary)]">{unit.amount.toLocaleString()}</strong>
                  </div>
                ))}
              </div>
            </section>

            {/* Server Time */}
            <div className="text-[11px] text-[var(--color-text-muted)] text-right">
              服务器时间: {state.serverTime}
            </div>
          </>
        )}
      </div>
    </div>
  )
}
