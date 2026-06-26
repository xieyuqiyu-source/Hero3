/* Hero3 GM 行军配置面板，负责维护 PVP 行军时间和加速规则。 */

import { useEffect, useState } from 'react'
import { Clock, Save } from 'lucide-react'
import { adminApi } from '@/api/admin'
import type { MarchConfig } from '@/types'

const DEFAULT_CONFIG: MarchConfig = {
  maxDurationSeconds: 10800,
  minDurationSeconds: 300,
  speedScale: 1,
  accelerate: {
    enabled: true,
    costCityGold: 50,
    reduceRate: 0.5,
    minRemainingSeconds: 300,
  },
}

export default function MarchConfigPanel() {
  const [config, setConfig] = useState<MarchConfig>(DEFAULT_CONFIG)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    adminApi.getMarchConfig()
      .then((data) => {
        if (!cancelled) setConfig(data)
      })
      .catch((err) => {
        if (!cancelled) setError(err instanceof Error ? err.message : '加载失败')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => { cancelled = true }
  }, [])

  const updateNumber = (key: keyof Omit<MarchConfig, 'accelerate'>, value: number) => {
    setConfig((current) => ({ ...current, [key]: value }))
  }

  const updateAccelerate = (key: keyof MarchConfig['accelerate'], value: number | boolean) => {
    setConfig((current) => ({
      ...current,
      accelerate: { ...current.accelerate, [key]: value },
    }))
  }

  const handleSave = async () => {
    setSaving(true)
    setMessage(null)
    setError(null)
    try {
      const result = await adminApi.updateMarchConfig(config)
      setConfig(result)
      setMessage('行军配置已保存')
    } catch (err) {
      setError(err instanceof Error ? err.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return (
      <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4">
        <p className="text-sm text-[var(--color-text-muted)]">加载中...</p>
      </div>
    )
  }

  return (
    <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] shadow-[var(--shadow-panel)] p-4">
      <div className="flex items-center justify-between gap-3 mb-4">
        <div className="flex items-center gap-2">
          <Clock size={16} className="text-[var(--color-accent)]" />
          <h2 className="text-base font-bold text-[var(--color-text-primary)]">PVP 行军规则</h2>
        </div>
        <button
          type="button"
          onClick={() => void handleSave()}
          disabled={saving}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-bold text-white bg-gradient-to-r from-[var(--color-accent)] to-indigo-600 border border-indigo-600/30 cursor-pointer transition-all disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <Save size={12} />
          {saving ? '保存中...' : '保存'}
        </button>
      </div>

      <div className="grid gap-3 lg:grid-cols-3">
        <NumberField
          label="最大行军时间"
          suffix="秒"
          min={1}
          value={config.maxDurationSeconds}
          onChange={(value) => updateNumber('maxDurationSeconds', value)}
        />
        <NumberField
          label="最短行军时间"
          suffix="秒"
          min={1}
          value={config.minDurationSeconds}
          onChange={(value) => updateNumber('minDurationSeconds', value)}
        />
        <NumberField
          label="速度换算倍率"
          suffix="倍"
          min={0.01}
          step={0.01}
          value={config.speedScale}
          onChange={(value) => updateNumber('speedScale', value)}
        />
      </div>

      <div className="mt-4 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-3">
        <div className="mb-3 flex items-center justify-between gap-3">
          <div>
            <h3 className="text-xs font-bold text-[var(--color-text-primary)]">行军加速</h3>
            <p className="mt-0.5 text-[10px] text-[var(--color-text-muted)]">已出发行军使用创建时固化的加速配置。</p>
          </div>
          <label className="inline-flex items-center gap-2 text-xs font-bold text-[var(--color-text-secondary)]">
            <input
              type="checkbox"
              checked={config.accelerate.enabled}
              onChange={(event) => updateAccelerate('enabled', event.target.checked)}
              className="h-4 w-4 accent-[var(--color-accent)]"
            />
            启用
          </label>
        </div>
        <div className="grid gap-3 lg:grid-cols-3">
          <NumberField
            label="固定消耗城金"
            suffix="城金"
            min={1}
            value={config.accelerate.costCityGold}
            onChange={(value) => updateAccelerate('costCityGold', value)}
          />
          <NumberField
            label="剩余时间倍率"
            suffix="倍"
            min={0.01}
            max={0.99}
            step={0.01}
            value={config.accelerate.reduceRate}
            onChange={(value) => updateAccelerate('reduceRate', value)}
          />
          <NumberField
            label="加速后最短剩余"
            suffix="秒"
            min={1}
            value={config.accelerate.minRemainingSeconds}
            onChange={(value) => updateAccelerate('minRemainingSeconds', value)}
          />
        </div>
      </div>

      {message && <p className="mt-3 text-xs font-bold text-emerald-600">{message}</p>}
      {error && <p className="mt-3 text-xs font-bold text-red-600">{error}</p>}
    </div>
  )
}

function NumberField({
  label,
  suffix,
  value,
  min,
  max,
  step = 1,
  onChange,
}: {
  label: string
  suffix: string
  value: number
  min?: number
  max?: number
  step?: number
  onChange: (value: number) => void
}) {
  return (
    <label className="grid gap-1 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2">
      <span className="text-[10px] font-bold text-[var(--color-text-muted)]">{label}</span>
      <div className="flex items-center gap-2">
        <input
          type="number"
          min={min}
          max={max}
          step={step}
          value={value}
          onChange={(event) => onChange(Number(event.target.value) || 0)}
          className="h-8 min-w-0 flex-1 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 text-xs text-[var(--color-text-primary)]"
        />
        <span className="w-10 text-right text-[10px] text-[var(--color-text-muted)]">{suffix}</span>
      </div>
    </label>
  )
}
