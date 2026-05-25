import { useCallback, useEffect, useState } from 'react'
import { X, RefreshCw, Coins, Swords, Building2 } from 'lucide-react'
import { adminApi } from '@/api/admin'
import type { GameState } from '@/types'
import ResourceAdjustForm from '@/components/ResourceAdjustForm'

interface PlayerDrawerProps {
  playerId: string
  onClose: () => void
}

export default function PlayerDrawer({ playerId, onClose }: PlayerDrawerProps) {
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

  // Close on Escape
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [onClose])

  return (
    <aside className="
      w-[380px] shrink-0 rounded-2xl border border-[var(--color-border)]
      bg-[var(--color-surface)] shadow-[var(--shadow-panel)]
      overflow-hidden flex flex-col
      max-lg:fixed max-lg:inset-y-0 max-lg:right-0 max-lg:z-50 max-lg:w-[90vw] max-lg:max-w-[400px]
      max-lg:rounded-none max-lg:border-l max-lg:shadow-2xl
      animate-in slide-in-from-right duration-200
    ">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-[var(--color-border)] bg-[var(--color-surface-dim)]">
        <div className="flex items-center gap-2">
          <h3 className="text-sm font-bold text-[var(--color-text-primary)]">
            {state?.player.nickname ?? '玩家详情'}
          </h3>
          {state?.player.faction && (
            <span className="px-1.5 py-0.5 rounded text-[10px] font-bold bg-[var(--color-accent-light)] text-[var(--color-accent)]">
              {state.player.faction}
            </span>
          )}
        </div>
        <div className="flex items-center gap-1">
          <button
            type="button"
            onClick={() => void loadState()}
            disabled={loading}
            className="p-1.5 rounded-lg text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-surface)] cursor-pointer transition-colors disabled:opacity-50"
            title="刷新"
          >
            <RefreshCw size={14} className={loading ? 'animate-spin' : ''} />
          </button>
          <button
            type="button"
            onClick={onClose}
            className="p-1.5 rounded-lg text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-surface)] cursor-pointer transition-colors"
            title="关闭"
          >
            <X size={14} />
          </button>
        </div>
      </div>

      {/* Body */}
      <div className="flex-1 overflow-y-auto p-4 space-y-3 scrollbar-none">
        {loading && (
          <div className="py-12 text-center text-sm text-[var(--color-text-secondary)]">
            <RefreshCw size={20} className="animate-spin mx-auto mb-2 text-[var(--color-accent)]" />
            加载中...
          </div>
        )}

        {error && (
          <div className="py-8 text-center">
            <p className="text-sm text-red-600 mb-2">{error}</p>
            <button
              type="button"
              onClick={() => void loadState()}
              className="px-3 py-1.5 rounded-lg text-xs font-bold border border-[var(--color-border)] text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-dim)] cursor-pointer"
            >
              重试
            </button>
          </div>
        )}

        {!loading && !error && state && (
          <>
            {/* Player ID */}
            <div className="px-3 py-2 rounded-xl bg-[var(--color-surface-dim)] border border-[var(--color-border)]">
              <span className="text-[10px] text-[var(--color-text-muted)] block">Player ID</span>
              <span className="text-xs text-[var(--color-text-primary)] font-mono select-all">{state.player.id}</span>
            </div>

            {/* Resources */}
            <section className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-3">
              <div className="flex items-center gap-1.5 mb-2">
                <Coins size={13} className="text-amber-500" />
                <h4 className="text-[10px] font-bold text-[var(--color-text-muted)] uppercase tracking-wider">资源</h4>
              </div>
              <div className="grid grid-cols-2 gap-2">
                {Object.entries(state.resources.items).map(([res, amount]) => (
                  <div key={res} className="px-2.5 py-2 rounded-lg bg-[var(--color-surface)] border border-[var(--color-border)]">
                    <span className="text-[10px] text-[var(--color-text-muted)] uppercase block">{res}</span>
                    <strong className="text-sm font-bold text-[var(--color-text-primary)]">{amount.toLocaleString()}</strong>
                    <span className="text-[10px] text-[var(--color-text-muted)]"> / {state.resources.capacity[res]?.toLocaleString() ?? '--'}</span>
                  </div>
                ))}
              </div>
              {/* Production rates */}
              <div className="flex flex-wrap gap-x-3 gap-y-1 mt-2 px-1">
                {Object.entries(state.resourceProduction).map(([res, rate]) => (
                  <span key={res} className="text-[10px] text-[var(--color-text-muted)]">
                    {res} <span className="text-emerald-600 font-bold">+{rate}/h</span>
                  </span>
                ))}
              </div>
            </section>

            {/* Resource Adjust (GM tool) */}
            <section className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-3">
              <h4 className="text-[10px] font-bold text-[var(--color-text-muted)] uppercase tracking-wider mb-2">GM 资源调整</h4>
              <ResourceAdjustForm playerId={playerId} onSuccess={(s) => setState(s)} />
            </section>

            {/* Buildings */}
            <section className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-3">
              <div className="flex items-center gap-1.5 mb-2">
                <Building2 size={13} className="text-indigo-500" />
                <h4 className="text-[10px] font-bold text-[var(--color-text-muted)] uppercase tracking-wider">
                  建筑 ({state.buildings.length})
                </h4>
              </div>
              <div className="flex flex-wrap gap-1.5">
                {state.buildings.map((b) => (
                  <span
                    key={b.id}
                    className="px-2 py-1 rounded-lg text-[10px] font-bold bg-[var(--color-gold-soft)] text-amber-700 border border-amber-500/10"
                  >
                    {b.type} Lv.{b.level}
                    {b.upgradeEndsAt && ' ⏳'}
                  </span>
                ))}
              </div>
            </section>

            {/* Army */}
            <section className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-3">
              <div className="flex items-center gap-1.5 mb-2">
                <Swords size={13} className="text-red-500" />
                <h4 className="text-[10px] font-bold text-[var(--color-text-muted)] uppercase tracking-wider">
                  军队 ({state.army.reduce((s, u) => s + u.amount, 0).toLocaleString()} 总兵力)
                </h4>
              </div>
              <div className="grid grid-cols-2 gap-2">
                {state.army.map((unit) => (
                  <div key={unit.unitType} className="px-2.5 py-2 rounded-lg bg-[var(--color-surface)] border border-[var(--color-border)]">
                    <span className="text-[10px] text-[var(--color-text-muted)]">{unit.unitType}</span>
                    <strong className="block text-sm font-bold text-[var(--color-text-primary)]">{unit.amount.toLocaleString()}</strong>
                  </div>
                ))}
              </div>
            </section>

            {/* Recruit Queues */}
            {state.recruitQueues.length > 0 && (
              <section className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-3">
                <h4 className="text-[10px] font-bold text-[var(--color-text-muted)] uppercase tracking-wider mb-2">
                  招募队列 ({state.recruitQueues.length})
                </h4>
                <div className="space-y-1.5">
                  {state.recruitQueues.map((q) => (
                    <div key={q.id} className="flex items-center justify-between px-2.5 py-1.5 rounded-lg bg-[var(--color-surface)] border border-[var(--color-border)]">
                      <span className="text-[11px] text-[var(--color-text-primary)] font-bold">{q.unitType} ×{q.amount}</span>
                      <span className="text-[10px] text-[var(--color-text-muted)]">{new Date(q.endsAt).toLocaleTimeString()}</span>
                    </div>
                  ))}
                </div>
              </section>
            )}

            {/* Server Time */}
            <div className="text-[10px] text-[var(--color-text-muted)] text-right pt-1">
              服务器时间: {state.serverTime}
            </div>
          </>
        )}
      </div>
    </aside>
  )
}
