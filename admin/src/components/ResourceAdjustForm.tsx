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

  const handleChange = (resource: string, value: string) => {
    setAdjustments((prev) => ({ ...prev, [resource]: value }))
  }

  const handleSubmit = async () => {
    const parsed: Record<string, number> = {}
    for (const [res, val] of Object.entries(adjustments)) {
      const num = parseInt(val, 10)
      if (!isNaN(num) && num !== 0) {
        parsed[res] = num
      }
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
    <section className="detail-section adjust-section">
      <h4>资源补发 / 扣减</h4>
      <div className="adjust-grid">
        {RESOURCE_TYPES.map((res) => (
          <label key={res} className="adjust-field">
            <span>{res}</span>
            <input
              type="number"
              placeholder="正数补发，负数扣减"
              value={adjustments[res] ?? ''}
              onChange={(e) => handleChange(res, e.target.value)}
              disabled={saving}
            />
          </label>
        ))}
      </div>
      <div className="adjust-actions">
        <button
          type="button"
          className="danger-button"
          onClick={() => void handleSubmit()}
          disabled={saving}
        >
          {saving ? '提交中' : '确认调整'}
        </button>
      </div>
      {message && <p className="inline-success">{message}</p>}
      {error && <p className="inline-error">{error}</p>}
    </section>
  )
}
