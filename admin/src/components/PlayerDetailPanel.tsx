import { useCallback, useEffect, useState } from 'react'
import { X } from 'lucide-react'
import { adminApi } from '@/api/admin'
import type { GameState } from '@/types'
import ResourceAdjustForm from '@/components/ResourceAdjustForm'

interface PlayerDetailPanelProps {
  playerId: string
  onClose: () => void
}

export default function PlayerDetailPanel({ playerId, onClose }: PlayerDetailPanelProps) {
  const [state, setState] = useState<GameState | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

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
    void loadState()
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
