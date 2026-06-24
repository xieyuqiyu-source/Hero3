import { useEffect, useMemo, useState } from 'react'
import { Fish, Plus, Save, Trash2 } from 'lucide-react'
import { adminApi } from '@/api/admin'
import type { FishingBaitConfig, FishingConfig, FishingFishConfig, FishingRarityConfig, UnitConfig } from '@/types'

const RARITY_LABELS: Record<string, string> = {
  common: '普通',
  rare: '稀有',
  epic: '史诗',
  legendary: '传说',
}

const emptyFish = (): FishingFishConfig => ({
  name: '新鱼获',
  rarity: 'common',
  reward: '',
  rewardAmount: 100,
  description: '',
  emoji: '🐟',
})

export default function FishingConfigPanel() {
  const [config, setConfig] = useState<FishingConfig | null>(null)
  const [unitsConfig, setUnitsConfig] = useState<Record<string, Record<string, UnitConfig>>>({})
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [jsonOpen, setJsonOpen] = useState(false)
  const [jsonDraft, setJsonDraft] = useState('')

  useEffect(() => {
    let cancelled = false
    Promise.all([adminApi.getFishingConfig(), adminApi.getUnitsConfig()])
      .then(([fishing, units]) => {
        if (cancelled) return
        setConfig(fishing)
        setUnitsConfig(units)
        setJsonDraft(JSON.stringify(fishing, null, 2))
      })
      .catch((err) => {
        if (!cancelled) setError(err instanceof Error ? err.message : '加载失败')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => { cancelled = true }
  }, [])

  const unitOptions = useMemo(() => {
    const map = new Map<string, string>()
    Object.entries(unitsConfig).forEach(([faction, units]) => {
      Object.entries(units).forEach(([unitId, unit]) => {
        if (!unit.name || unit.role === 'scout' || unit.role === 'transport') return
        map.set(unit.name, `${unit.name}（${faction}/${unitId}）`)
      })
    })
    return Array.from(map.entries()).sort(([a], [b]) => a.localeCompare(b, 'zh-Hans'))
  }, [unitsConfig])

  const updateConfig = (next: FishingConfig) => {
    setConfig(next)
    setJsonDraft(JSON.stringify(next, null, 2))
  }

  const handleSave = async () => {
    if (!config) return
    setSaving(true)
    setMessage(null)
    setError(null)
    try {
      const result = await adminApi.updateFishingConfig(config)
      setConfig(result)
      setJsonDraft(JSON.stringify(result, null, 2))
      setMessage('钓鱼配置已保存')
    } catch (err) {
      setError(err instanceof Error ? err.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  const updateRarity = (rarityId: string, field: keyof FishingRarityConfig, value: string | number) => {
    if (!config) return
    updateConfig({
      ...config,
      rarities: {
        ...config.rarities,
        [rarityId]: { ...config.rarities[rarityId], [field]: value },
      },
    })
  }

  const updateBait = (index: number, field: keyof FishingBaitConfig, value: string | number) => {
    if (!config) return
    const baits = [...config.baits]
    baits[index] = { ...baits[index], [field]: value }
    updateConfig({ ...config, baits })
  }

  const updateFish = (index: number, field: keyof FishingFishConfig, value: string | number) => {
    if (!config) return
    const fishPool = [...config.fishPool]
    fishPool[index] = { ...fishPool[index], [field]: value }
    updateConfig({ ...config, fishPool })
  }

  const addFish = () => {
    if (!config) return
    const defaultReward = unitOptions[0]?.[0] ?? ''
    updateConfig({ ...config, fishPool: [...config.fishPool, { ...emptyFish(), reward: defaultReward }] })
  }

  const removeFish = (index: number) => {
    if (!config) return
    updateConfig({ ...config, fishPool: config.fishPool.filter((_, i) => i !== index) })
  }

  const applyJsonDraft = () => {
    try {
      const parsed = JSON.parse(jsonDraft) as FishingConfig
      updateConfig(parsed)
      setMessage('JSON 已应用，保存后生效')
      setError(null)
    } catch {
      setError('JSON 格式错误')
    }
  }

  if (loading) return <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4 text-sm text-[var(--color-text-muted)]">加载中...</div>
  if (!config) return <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4 text-sm text-red-600">{error ?? '加载失败'}</div>

  return (
    <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] shadow-[var(--shadow-panel)] p-4">
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <Fish size={16} className="text-[var(--color-accent)]" />
          <div>
            <h2 className="text-base font-bold text-[var(--color-text-primary)]">仙池垂钓配置</h2>
            <p className="text-xs text-[var(--color-text-muted)]">鱼池、鱼饵、稀有度权重统一由后端配置下发</p>
          </div>
        </div>
        <button
          type="button"
          onClick={() => void handleSave()}
          disabled={saving}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-bold text-white bg-gradient-to-r from-[var(--color-accent)] to-indigo-600 border border-indigo-600/30 cursor-pointer transition-all disabled:opacity-50"
        >
          <Save size={12} />
          {saving ? '保存中...' : '保存配置'}
        </button>
      </div>

      {(message || error) && (
        <div className={`mb-4 rounded-xl border px-3 py-2 text-xs ${error ? 'border-red-500/30 bg-red-500/8 text-red-600' : 'border-emerald-500/30 bg-emerald-500/8 text-emerald-700'}`}>
          {error ?? message}
        </div>
      )}

      <section className="mb-5">
        <h3 className="mb-2 text-[10px] font-bold uppercase tracking-wider text-[var(--color-text-muted)]">稀有度概率权重</h3>
        <div className="grid gap-2 md:grid-cols-4">
          {Object.entries(config.rarities).map(([rarityId, rarity]) => (
            <label key={rarityId} className="grid gap-1 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2">
              <span className="text-[10px] text-[var(--color-text-muted)]">{RARITY_LABELS[rarityId] ?? rarity.label}</span>
              <input
                type="number"
                min={0.1}
                step={0.1}
                value={rarity.weight}
                onChange={(e) => updateRarity(rarityId, 'weight', Number(e.target.value))}
                className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 py-1 text-sm font-bold text-[var(--color-text-primary)] outline-none"
              />
            </label>
          ))}
        </div>
      </section>

      <section className="mb-5">
        <h3 className="mb-2 text-[10px] font-bold uppercase tracking-wider text-[var(--color-text-muted)]">鱼饵规则</h3>
        <div className="grid gap-3 lg:grid-cols-2">
          {config.baits.map((bait, index) => (
            <div key={bait.id} className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-3">
              <div className="mb-2 grid gap-2 sm:grid-cols-3">
                <input value={bait.name} onChange={(e) => updateBait(index, 'name', e.target.value)} className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 py-1 text-sm font-bold" />
                <input value={bait.tier} onChange={(e) => updateBait(index, 'tier', e.target.value)} className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 py-1 text-sm" />
                <input type="number" min={0} value={bait.cityGoldCost} onChange={(e) => updateBait(index, 'cityGoldCost', Number(e.target.value))} className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 py-1 text-sm" />
              </div>
              <input value={bait.description} onChange={(e) => updateBait(index, 'description', e.target.value)} className="mb-2 w-full rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 py-1 text-sm" />
              <div className="grid grid-cols-2 gap-2 sm:grid-cols-5">
                <NumberField label="稀有倍率" value={bait.rarityBoost} step={0.01} onChange={(value) => updateBait(index, 'rarityBoost', value)} />
                <NumberField label="咬钩率" value={bait.biteChance} step={0.01} onChange={(value) => updateBait(index, 'biteChance', value)} />
                <NumberField label="窗口ms" value={bait.biteWindowMs} onChange={(value) => updateBait(index, 'biteWindowMs', value)} />
                <NumberField label="命中起点" value={bait.sweetStart} onChange={(value) => updateBait(index, 'sweetStart', value)} />
                <NumberField label="命中终点" value={bait.sweetEnd} onChange={(value) => updateBait(index, 'sweetEnd', value)} />
              </div>
            </div>
          ))}
        </div>
      </section>

      <section>
        <div className="mb-2 flex items-center justify-between gap-2">
          <h3 className="text-[10px] font-bold uppercase tracking-wider text-[var(--color-text-muted)]">鱼池奖励</h3>
          <button type="button" onClick={addFish} className="flex items-center gap-1 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-2 py-1 text-xs font-bold text-[var(--color-text-primary)]">
            <Plus size={12} />
            加鱼
          </button>
        </div>
        <div className="grid gap-2">
          {config.fishPool.map((fish, index) => (
            <div key={`${fish.name}-${index}`} className="grid gap-2 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-2 lg:grid-cols-[70px_1fr_110px_1.2fr_110px_1.4fr_40px]">
              <input value={fish.emoji} onChange={(e) => updateFish(index, 'emoji', e.target.value)} className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 py-1 text-sm" />
              <input value={fish.name} onChange={(e) => updateFish(index, 'name', e.target.value)} className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 py-1 text-sm font-bold" />
              <select value={fish.rarity} onChange={(e) => updateFish(index, 'rarity', e.target.value)} className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 py-1 text-sm">
                {Object.keys(config.rarities).map((rarityId) => <option key={rarityId} value={rarityId}>{RARITY_LABELS[rarityId] ?? rarityId}</option>)}
              </select>
              <select value={fish.reward} onChange={(e) => updateFish(index, 'reward', e.target.value)} className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 py-1 text-sm">
                {!fish.reward && <option value="">选择兵种</option>}
                {unitOptions.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
                {fish.reward && !unitOptions.some(([value]) => value === fish.reward) && <option value={fish.reward}>{fish.reward}（自定义）</option>}
              </select>
              <input type="number" min={1} value={fish.rewardAmount} onChange={(e) => updateFish(index, 'rewardAmount', Number(e.target.value))} className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 py-1 text-sm" />
              <input value={fish.description} onChange={(e) => updateFish(index, 'description', e.target.value)} className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 py-1 text-sm" />
              <button type="button" onClick={() => removeFish(index)} className="grid place-items-center rounded-lg border border-red-500/20 bg-red-500/8 text-red-600">
                <Trash2 size={14} />
              </button>
            </div>
          ))}
        </div>
      </section>

      <details className="mt-4 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-3" open={jsonOpen} onToggle={(e) => setJsonOpen(e.currentTarget.open)}>
        <summary className="cursor-pointer text-xs font-bold text-[var(--color-text-secondary)]">高级 JSON</summary>
        <textarea
          value={jsonDraft}
          onChange={(e) => setJsonDraft(e.target.value)}
          className="mt-3 min-h-[260px] w-full rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] p-3 font-mono text-xs text-[var(--color-text-primary)] outline-none"
        />
        <button type="button" onClick={applyJsonDraft} className="mt-2 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-1.5 text-xs font-bold text-[var(--color-text-primary)]">
          应用 JSON 到表单
        </button>
      </details>
    </div>
  )
}

function NumberField({ label, value, step, onChange }: { label: string; value: number; step?: number; onChange: (value: number) => void }) {
  return (
    <label className="grid gap-1">
      <span className="text-[10px] text-[var(--color-text-muted)]">{label}</span>
      <input
        type="number"
        step={step}
        value={value}
        onChange={(e) => onChange(Number(e.target.value))}
        className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 py-1 text-sm"
      />
    </label>
  )
}
