import { useEffect, useState } from 'react'
import { MapPin, Save } from 'lucide-react'
import { adminApi } from '@/api/admin'

interface TierConfig {
  multiplier: number
  armyRange: { min: number; max: number }
  armyTypes: { min: number; max: number }
  traitCount: { min: number; max: number }
  count: { guaranteed: number; weight: number }
}

interface NpcConfig {
  baseProduction: number
  baseStorage: number
  refreshIntervalHours: number
  manualRefreshCostGold: number
  goldenAppearRate: number
  totalCities: number
  tiers: Record<string, TierConfig>
  scoutCost: Record<string, number>
  [key: string]: any
}

const TIER_LABELS: Record<string, string> = {
  small: '小型',
  medium: '中型',
  large: '大型',
  golden: '黄金',
}

export default function NpcConfigPanel() {
  const [config, setConfig] = useState<NpcConfig | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    adminApi.getNpcConfig()
      .then((data) => { if (!cancelled) setConfig(data as NpcConfig) })
      .catch((err) => { if (!cancelled) setError(err instanceof Error ? err.message : '加载失败') })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [])

  const handleSave = async () => {
    if (!config) return
    setSaving(true)
    setMessage(null)
    setError(null)
    try {
      const result = await adminApi.updateNpcConfig(config as any)
      setConfig(result as NpcConfig)
      setMessage('NPC 配置已保存')
    } catch (err) {
      setError(err instanceof Error ? err.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  const updateGlobal = (key: string, value: number) => {
    if (!config) return
    setConfig({ ...config, [key]: value })
  }

  const updateTier = (tierId: string, field: string, value: number) => {
    if (!config) return
    const tier = config.tiers[tierId]
    if (!tier) return

    if (field === 'multiplier') {
      setConfig({ ...config, tiers: { ...config.tiers, [tierId]: { ...tier, multiplier: value } } })
    } else if (field === 'armyMin') {
      setConfig({ ...config, tiers: { ...config.tiers, [tierId]: { ...tier, armyRange: { ...tier.armyRange, min: value } } } })
    } else if (field === 'armyMax') {
      setConfig({ ...config, tiers: { ...config.tiers, [tierId]: { ...tier, armyRange: { ...tier.armyRange, max: value } } } })
    } else if (field === 'guaranteed') {
      setConfig({ ...config, tiers: { ...config.tiers, [tierId]: { ...tier, count: { ...tier.count, guaranteed: value } } } })
    }
  }

  const updateScoutCost = (tierId: string, value: number) => {
    if (!config) return
    setConfig({ ...config, scoutCost: { ...config.scoutCost, [tierId]: value } })
  }

  if (loading) return <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4"><p className="text-sm text-[var(--color-text-muted)]">加载中...</p></div>
  if (!config) return <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4"><p className="text-sm text-red-600">{error ?? '加载失败'}</p></div>

  return (
    <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] shadow-[var(--shadow-panel)] p-4">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <MapPin size={16} className="text-[var(--color-accent)]" />
          <h2 className="text-base font-bold text-[var(--color-text-primary)]">NPC 城池</h2>
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

      {/* Global Settings */}
      <section className="mb-4">
        <h3 className="text-[10px] font-bold text-[var(--color-text-muted)] uppercase tracking-wider mb-2">全局参数</h3>
        <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
          <label className="grid gap-1 px-2.5 py-2 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
            <span className="text-[10px] text-[var(--color-text-muted)]">基础产量</span>
            <input type="number" value={config.baseProduction} onChange={(e) => updateGlobal('baseProduction', parseInt(e.target.value) || 0)} className="h-7 px-2 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]" />
          </label>
          <label className="grid gap-1 px-2.5 py-2 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
            <span className="text-[10px] text-[var(--color-text-muted)]">基础仓储</span>
            <input type="number" value={config.baseStorage} onChange={(e) => updateGlobal('baseStorage', parseInt(e.target.value) || 0)} className="h-7 px-2 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]" />
          </label>
          <label className="grid gap-1 px-2.5 py-2 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
            <span className="text-[10px] text-[var(--color-text-muted)]">刷新间隔(h)</span>
            <input type="number" value={config.refreshIntervalHours} onChange={(e) => updateGlobal('refreshIntervalHours', parseInt(e.target.value) || 24)} className="h-7 px-2 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]" />
          </label>
          <label className="grid gap-1 px-2.5 py-2 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
            <span className="text-[10px] text-[var(--color-text-muted)]">手动刷新金币</span>
            <input type="number" value={config.manualRefreshCostGold} onChange={(e) => updateGlobal('manualRefreshCostGold', parseInt(e.target.value) || 0)} className="h-7 px-2 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]" />
          </label>
          <label className="grid gap-1 px-2.5 py-2 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
            <span className="text-[10px] text-[var(--color-text-muted)]">黄金出现率</span>
            <input type="number" step="0.01" value={config.goldenAppearRate} onChange={(e) => updateGlobal('goldenAppearRate', parseFloat(e.target.value) || 0)} className="h-7 px-2 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]" />
          </label>
          <label className="grid gap-1 px-2.5 py-2 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
            <span className="text-[10px] text-[var(--color-text-muted)]">总城池数</span>
            <input type="number" value={config.totalCities} onChange={(e) => updateGlobal('totalCities', parseInt(e.target.value) || 12)} className="h-7 px-2 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]" />
          </label>
        </div>
      </section>

      {/* Tier Multipliers */}
      <section className="mb-4">
        <h3 className="text-[10px] font-bold text-[var(--color-text-muted)] uppercase tracking-wider mb-2">等级倍率</h3>
        <div className="grid grid-cols-2 gap-2">
          {Object.entries(config.tiers).map(([tierId, tier]) => (
            <div key={tierId} className={`p-3 rounded-xl border bg-[var(--color-surface-dim)] ${tierId === 'golden' ? 'border-amber-500/40' : 'border-[var(--color-border)]'}`}>
              <div className="flex items-center gap-2 mb-2">
                <span className={`px-1.5 py-0.5 rounded text-[9px] font-black uppercase ${tierId === 'golden' ? 'bg-amber-500/15 text-amber-700' : tierId === 'large' ? 'bg-red-500/15 text-red-700' : tierId === 'medium' ? 'bg-blue-500/15 text-blue-700' : 'bg-emerald-500/15 text-emerald-700'}`}>
                  {TIER_LABELS[tierId] ?? tierId}
                </span>
                <span className="text-[10px] text-[var(--color-text-muted)]">×{tier.multiplier}</span>
              </div>
              <div className="grid grid-cols-2 gap-1.5">
                <label className="grid gap-0.5">
                  <span className="text-[9px] text-[var(--color-text-muted)]">资源倍率</span>
                  <input type="number" step="0.1" value={tier.multiplier} onChange={(e) => updateTier(tierId, 'multiplier', parseFloat(e.target.value) || 1)} className="h-6 px-1.5 rounded text-[11px] border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]" />
                </label>
                <label className="grid gap-0.5">
                  <span className="text-[9px] text-[var(--color-text-muted)]">保底数量</span>
                  <input type="number" value={tier.count.guaranteed} onChange={(e) => updateTier(tierId, 'guaranteed', parseInt(e.target.value) || 0)} className="h-6 px-1.5 rounded text-[11px] border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]" />
                </label>
                <label className="grid gap-0.5">
                  <span className="text-[9px] text-[var(--color-text-muted)]">兵力下限</span>
                  <input type="number" value={tier.armyRange.min} onChange={(e) => updateTier(tierId, 'armyMin', parseInt(e.target.value) || 0)} className="h-6 px-1.5 rounded text-[11px] border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]" />
                </label>
                <label className="grid gap-0.5">
                  <span className="text-[9px] text-[var(--color-text-muted)]">兵力上限</span>
                  <input type="number" value={tier.armyRange.max} onChange={(e) => updateTier(tierId, 'armyMax', parseInt(e.target.value) || 0)} className="h-6 px-1.5 rounded text-[11px] border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]" />
                </label>
              </div>
            </div>
          ))}
        </div>
      </section>

      {/* Scout Cost */}
      <section>
        <h3 className="text-[10px] font-bold text-[var(--color-text-muted)] uppercase tracking-wider mb-2">侦察消耗(金币)</h3>
        <div className="grid grid-cols-4 gap-2">
          {Object.entries(config.scoutCost ?? {}).map(([tierId, cost]) => (
            <label key={tierId} className="grid gap-0.5 px-2.5 py-2 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
              <span className="text-[9px] font-bold text-[var(--color-text-muted)] uppercase">{TIER_LABELS[tierId] ?? tierId}</span>
              <input type="number" value={cost} onChange={(e) => updateScoutCost(tierId, parseInt(e.target.value) || 0)} className="h-6 px-1.5 rounded text-[11px] border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]" />
            </label>
          ))}
        </div>
      </section>

      {message && <p className="mt-3 text-xs font-bold text-emerald-600">{message}</p>}
      {error && <p className="mt-3 text-xs font-bold text-red-600">{error}</p>}
    </div>
  )
}
