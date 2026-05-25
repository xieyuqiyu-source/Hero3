import { useEffect, useState } from 'react'
import { MapPin, Save, Plus, Trash2 } from 'lucide-react'
import { adminApi } from '@/api/admin'

interface TierConfig {
  multiplier: number
  armyRange: { min: number; max: number }
  armyTypes: { min: number; max: number }
  traitCount: { min: number; max: number }
  count: { guaranteed: number; weight: number }
}

interface RecoveryProfile {
  id: string
  name: string
  armyMultiplier: number
  resourceMultiplier: number
  weight: number
}

interface TraitEntry {
  id: string
  name: string
  buffs: Record<string, number>
  weight: number
}

interface NpcConfig {
  baseProduction: number
  baseStorage: number
  refreshIntervalHours: number
  manualRefreshCostGold: number
  goldenAppearRate: number
  totalCities: number
  tiers: Record<string, TierConfig>
  recoveryProfiles: RecoveryProfile[]
  traitPool: TraitEntry[]
  scoutCost: Record<string, number>
  cityNames: string[]
  [key: string]: any
}

const TIER_LABELS: Record<string, string> = {
  small: '小型',
  medium: '中型',
  large: '大型',
  golden: '黄金',
}

const BUFF_LABELS: Record<string, string> = {
  productionBonus: '产量加成',
  cavalryDefenseBonus: '骑防加成',
  infantryDefenseBonus: '步防加成',
  attackBonus: '攻击加成',
  allDefenseBonus: '全防加成',
  armyRecoveryBonus: '兵力恢复',
  armyCapBonus: '兵力上限',
  armyAttackBonus: '兵攻加成',
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
    } else if (field === 'traitMin') {
      setConfig({ ...config, tiers: { ...config.tiers, [tierId]: { ...tier, traitCount: { ...tier.traitCount, min: value } } } })
    } else if (field === 'traitMax') {
      setConfig({ ...config, tiers: { ...config.tiers, [tierId]: { ...tier, traitCount: { ...tier.traitCount, max: value } } } })
    }
  }

  const updateScoutCost = (tierId: string, value: number) => {
    if (!config) return
    setConfig({ ...config, scoutCost: { ...config.scoutCost, [tierId]: value } })
  }

  // --- Trait Pool ---
  const updateTrait = (index: number, field: keyof TraitEntry, value: any) => {
    if (!config) return
    const next = [...config.traitPool]
    next[index] = { ...next[index], [field]: value }
    setConfig({ ...config, traitPool: next })
  }

  const updateTraitBuff = (index: number, buffKey: string, value: number) => {
    if (!config) return
    const next = [...config.traitPool]
    next[index] = { ...next[index], buffs: { ...next[index].buffs, [buffKey]: value } }
    setConfig({ ...config, traitPool: next })
  }

  const addTrait = () => {
    if (!config) return
    const newTrait: TraitEntry = { id: `trait_${Date.now()}`, name: '新词条', buffs: {}, weight: 10 }
    setConfig({ ...config, traitPool: [...config.traitPool, newTrait] })
  }

  const removeTrait = (index: number) => {
    if (!config) return
    const next = config.traitPool.filter((_, i) => i !== index)
    setConfig({ ...config, traitPool: next })
  }

  const addTraitBuff = (index: number) => {
    if (!config) return
    const next = [...config.traitPool]
    const existing = Object.keys(next[index].buffs)
    const newKey = `newBuff${existing.length}`
    next[index] = { ...next[index], buffs: { ...next[index].buffs, [newKey]: 0.1 } }
    setConfig({ ...config, traitPool: next })
  }

  const removeTraitBuff = (traitIndex: number, buffKey: string) => {
    if (!config) return
    const next = [...config.traitPool]
    const { [buffKey]: _, ...rest } = next[traitIndex].buffs
    next[traitIndex] = { ...next[traitIndex], buffs: rest }
    setConfig({ ...config, traitPool: next })
  }

  // --- Recovery Profiles ---
  const updateProfile = (index: number, field: keyof RecoveryProfile, value: any) => {
    if (!config) return
    const next = [...config.recoveryProfiles]
    next[index] = { ...next[index], [field]: value }
    setConfig({ ...config, recoveryProfiles: next })
  }

  const addProfile = () => {
    if (!config) return
    const newProfile: RecoveryProfile = { id: `profile_${Date.now()}`, name: '新档案', armyMultiplier: 1.0, resourceMultiplier: 1.0, weight: 10 }
    setConfig({ ...config, recoveryProfiles: [...config.recoveryProfiles, newProfile] })
  }

  const removeProfile = (index: number) => {
    if (!config) return
    const next = config.recoveryProfiles.filter((_, i) => i !== index)
    setConfig({ ...config, recoveryProfiles: next })
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
          {[
            { key: 'baseProduction', label: '基础产量', step: undefined },
            { key: 'baseStorage', label: '基础仓储', step: undefined },
            { key: 'refreshIntervalHours', label: '刷新间隔(h)', step: undefined },
            { key: 'manualRefreshCostGold', label: '手动刷新金币', step: undefined },
            { key: 'goldenAppearRate', label: '黄金出现率', step: '0.01' },
            { key: 'totalCities', label: '总城池数', step: undefined },
          ].map(({ key, label, step }) => (
            <label key={key} className="grid gap-1 px-2.5 py-2 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
              <span className="text-[10px] text-[var(--color-text-muted)]">{label}</span>
              <input
                type="number"
                step={step}
                value={(config as any)[key]}
                onChange={(e) => updateGlobal(key, step ? parseFloat(e.target.value) || 0 : parseInt(e.target.value) || 0)}
                className="h-7 px-2 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]"
              />
            </label>
          ))}
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
              <div className="grid grid-cols-3 gap-1.5">
                <label className="grid gap-0.5">
                  <span className="text-[9px] text-[var(--color-text-muted)]">资源倍率</span>
                  <input type="number" step="0.1" value={tier.multiplier} onChange={(e) => updateTier(tierId, 'multiplier', parseFloat(e.target.value) || 1)} className="h-6 px-1.5 rounded text-[11px] border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]" />
                </label>
                <label className="grid gap-0.5">
                  <span className="text-[9px] text-[var(--color-text-muted)]">保底数量</span>
                  <input type="number" value={tier.count.guaranteed} onChange={(e) => updateTier(tierId, 'guaranteed', parseInt(e.target.value) || 0)} className="h-6 px-1.5 rounded text-[11px] border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]" />
                </label>
                <label className="grid gap-0.5">
                  <span className="text-[9px] text-[var(--color-text-muted)]">词条数</span>
                  <div className="flex gap-1">
                    <input type="number" value={tier.traitCount.min} onChange={(e) => updateTier(tierId, 'traitMin', parseInt(e.target.value) || 0)} className="h-6 w-full px-1 rounded text-[11px] border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]" />
                    <input type="number" value={tier.traitCount.max} onChange={(e) => updateTier(tierId, 'traitMax', parseInt(e.target.value) || 0)} className="h-6 w-full px-1 rounded text-[11px] border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]" />
                  </div>
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

      {/* Trait Pool */}
      <section className="mb-4">
        <div className="flex items-center justify-between mb-2">
          <h3 className="text-[10px] font-bold text-[var(--color-text-muted)] uppercase tracking-wider">词条池 ({config.traitPool.length})</h3>
          <button type="button" onClick={addTrait} className="flex items-center gap-1 px-2 py-1 rounded-lg text-[10px] font-bold text-[var(--color-accent)] bg-[var(--color-accent-light)] border border-[var(--color-accent-border)] cursor-pointer hover:bg-[var(--color-accent)]/15 transition-colors">
            <Plus size={10} /> 添加
          </button>
        </div>
        <div className="grid gap-2">
          {config.traitPool.map((trait, index) => (
            <div key={trait.id} className="p-2.5 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
              <div className="flex items-center gap-2 mb-2">
                <input
                  type="text"
                  value={trait.name}
                  onChange={(e) => updateTrait(index, 'name', e.target.value)}
                  className="h-6 px-2 rounded-lg text-xs font-bold border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)] flex-1 min-w-0"
                />
                <input
                  type="text"
                  value={trait.id}
                  onChange={(e) => updateTrait(index, 'id', e.target.value)}
                  className="h-6 px-2 rounded-lg text-[10px] border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-muted)] w-28"
                  placeholder="id"
                />
                <label className="flex items-center gap-1">
                  <span className="text-[9px] text-[var(--color-text-muted)]">权重</span>
                  <input
                    type="number"
                    value={trait.weight}
                    onChange={(e) => updateTrait(index, 'weight', parseInt(e.target.value) || 0)}
                    className="h-6 w-12 px-1 rounded text-[11px] border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]"
                  />
                </label>
                <button type="button" onClick={() => removeTrait(index)} className="w-6 h-6 flex items-center justify-center rounded-lg text-red-500 hover:bg-red-500/10 cursor-pointer transition-colors">
                  <Trash2 size={11} />
                </button>
              </div>
              {/* Buffs */}
              <div className="flex flex-wrap items-center gap-1.5">
                {Object.entries(trait.buffs).map(([buffKey, buffVal]) => (
                  <div key={buffKey} className="flex items-center gap-1 px-1.5 py-0.5 rounded-lg bg-white/60 dark:bg-white/5 border border-[var(--color-border)]">
                    <span className="text-[9px] text-[var(--color-text-muted)]">{BUFF_LABELS[buffKey] ?? buffKey}</span>
                    <input
                      type="number"
                      step="0.01"
                      value={buffVal}
                      onChange={(e) => updateTraitBuff(index, buffKey, parseFloat(e.target.value) || 0)}
                      className="h-5 w-14 px-1 rounded text-[10px] border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]"
                    />
                    <button type="button" onClick={() => removeTraitBuff(index, buffKey)} className="text-red-400 hover:text-red-600 cursor-pointer">
                      <Trash2 size={9} />
                    </button>
                  </div>
                ))}
                <button type="button" onClick={() => addTraitBuff(index)} className="px-1.5 py-0.5 rounded-lg text-[9px] font-bold text-[var(--color-accent)] bg-[var(--color-accent-light)] border border-[var(--color-accent-border)] cursor-pointer hover:bg-[var(--color-accent)]/15 transition-colors">
                  +buff
                </button>
              </div>
            </div>
          ))}
        </div>
      </section>

      {/* Recovery Profiles */}
      <section className="mb-4">
        <div className="flex items-center justify-between mb-2">
          <h3 className="text-[10px] font-bold text-[var(--color-text-muted)] uppercase tracking-wider">恢复档案 ({config.recoveryProfiles.length})</h3>
          <button type="button" onClick={addProfile} className="flex items-center gap-1 px-2 py-1 rounded-lg text-[10px] font-bold text-[var(--color-accent)] bg-[var(--color-accent-light)] border border-[var(--color-accent-border)] cursor-pointer hover:bg-[var(--color-accent)]/15 transition-colors">
            <Plus size={10} /> 添加
          </button>
        </div>
        <div className="grid gap-2">
          {config.recoveryProfiles.map((profile, index) => (
            <div key={profile.id} className="flex items-center gap-2 px-2.5 py-2 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
              <input
                type="text"
                value={profile.name}
                onChange={(e) => updateProfile(index, 'name', e.target.value)}
                className="h-6 px-2 rounded-lg text-xs font-bold border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)] w-20"
              />
              <label className="flex items-center gap-1">
                <span className="text-[9px] text-[var(--color-text-muted)]">兵力×</span>
                <input type="number" step="0.1" value={profile.armyMultiplier} onChange={(e) => updateProfile(index, 'armyMultiplier', parseFloat(e.target.value) || 1)} className="h-6 w-14 px-1 rounded text-[11px] border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]" />
              </label>
              <label className="flex items-center gap-1">
                <span className="text-[9px] text-[var(--color-text-muted)]">资源×</span>
                <input type="number" step="0.1" value={profile.resourceMultiplier} onChange={(e) => updateProfile(index, 'resourceMultiplier', parseFloat(e.target.value) || 1)} className="h-6 w-14 px-1 rounded text-[11px] border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]" />
              </label>
              <label className="flex items-center gap-1">
                <span className="text-[9px] text-[var(--color-text-muted)]">权重</span>
                <input type="number" value={profile.weight} onChange={(e) => updateProfile(index, 'weight', parseInt(e.target.value) || 0)} className="h-6 w-12 px-1 rounded text-[11px] border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]" />
              </label>
              <button type="button" onClick={() => removeProfile(index)} className="ml-auto w-6 h-6 flex items-center justify-center rounded-lg text-red-500 hover:bg-red-500/10 cursor-pointer transition-colors">
                <Trash2 size={11} />
              </button>
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
