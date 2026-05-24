import { useState } from 'react'
import { adminApi } from '@/api/admin'
import type { GameState } from '@/types'

const RESOURCE_TYPES = ['wood', 'stone', 'iron', 'food']

interface ResourceAdjustFormProps {
  playerId: string
  onSuccess: (state: GameState) => void
}

export default function ResourceAdjustForm({ playerId, onSuccess }: ResourceAdjustFormProps) {
  const [adjustments, setAdjustments] = useState<Record<string, string>>({})
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const handleSubmit = async () => {
    const parsed: Record<string, number> = {}
    for (const [res, val] of Object.entries(adjustments)) {
      const num = parseInt(val, 10)
      if (!isNaN(num) && num !== 0) parsed[res] = num
    }
    if (Object.keys(parsed).length === 0) {
      setError('请至少填写一项非零调整值')
      return
    }

    setSaving(true)
    setMessage(null)
    setError(null)
    try {
      const result = await adminApi.adjustResources(playerId, parsed)
      onSuccess(result.state)
      setAdjustments({})
      setMessage('资源已调整')
    } catch (err) {
      setError(err instanceof Error ? err.message : '调整失败')
    } finally {
      setSaving(false)
    }
  }

  return (
    <section className="mb-4 p-3.5 rounded-2xl border border-amber-500/30 bg-amber-500/5">
      <h4 className="text-xs font-bold text-amber-700 uppercase tracking-wider mb-2.5">资源补发 / 扣减</h4>
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-2 mb-3">
        {RESOURCE_TYPES.map((res) => (
          <label key={res} className="grid gap-1">
            <span className="text-[10px] font-bold text-[var(--color-text-muted)] uppercase">{res}</span>
            <input
              type="number"
              placeholder="+/-"
              value={adjustments[res] ?? ''}
              onChange={(e) => setAdjustments((prev) => ({ ...prev, [res]: e.target.value }))}
              disabled={saving}
              className="
                h-8 px-2.5 rounded-lg text-sm
                border border-[var(--color-border)] bg-[var(--color-surface)]
                text-[var(--color-text-primary)]
                focus:outline-2 focus:outline-[var(--color-accent-border)] focus:outline-offset-1
                disabled:opacity-50
              "
            />
          </label>
        ))}
      </div>
      <button
        type="button"
        onClick={() => void handleSubmit()}
        disabled={saving}
        className="
          px-4 py-2 rounded-xl text-xs font-bold
          text-white bg-gradient-to-r from-amber-500 to-orange-600
          border border-amber-600/30
          hover:shadow-[0_8px_20px_rgba(245,158,11,0.25)]
          cursor-pointer transition-all
          disabled:opacity-50 disabled:cursor-not-allowed
        "
      >
        {saving ? '提交中...' : '确认调整'}
      </button>
      {message && <p className="mt-2 text-xs font-bold text-emerald-600">{message}</p>}
      {error && <p className="mt-2 text-xs font-bold text-red-600">{error}</p>}
    </section>
  )
}
