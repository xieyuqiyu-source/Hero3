import { useEffect, useState } from 'react'
import { Flag, Save, Plus, Trash2 } from 'lucide-react'
import { adminApi } from '@/api/admin'
import type { FactionConfig, FactionGeneralInfo, FactionsConfig } from '@/types'

const TRAIT_LABELS: Record<string, string> = {
  economyBonus: '经济加成',
  militaryBonus: '军事加成',
  buildingBonus: '建筑加成',
}

export default function FactionsConfigPanel() {
  const [config, setConfig] = useState<FactionsConfig | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    adminApi.getFactionsConfig()
      .then((data) => { if (!cancelled) setConfig(data as FactionsConfig) })
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
      const result = await adminApi.updateFactionsConfig(config)
      setConfig(result as FactionsConfig)
      setMessage('阵营配置已保存')
    } catch (err) {
      setError(err instanceof Error ? err.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  const updateTrait = (factionId: string, traitKey: string, value: number) => {
    if (!config) return
    setConfig({
      ...config,
      [factionId]: {
        ...config[factionId],
        traits: { ...config[factionId].traits, [traitKey]: value },
      },
    })
  }

  const updateFactionField = <K extends keyof FactionConfig,>(factionId: string, field: K, value: FactionConfig[K]) => {
    if (!config) return
    setConfig({
      ...config,
      [factionId]: { ...config[factionId], [field]: value },
    })
  }

  const updateGeneral = <K extends keyof FactionGeneralInfo,>(factionId: string, index: number, field: K, value: FactionGeneralInfo[K]) => {
    if (!config) return
    const generals = [...config[factionId].generals]
    generals[index] = { ...generals[index], [field]: value }
    updateFactionField(factionId, 'generals', generals)
  }

  const addGeneral = (factionId: string) => {
    if (!config) return
    updateFactionField(factionId, 'generals', [...config[factionId].generals, { id: '', name: '新将领', title: '' }])
  }

  const removeGeneral = (factionId: string, index: number) => {
    if (!config) return
    updateFactionField(factionId, 'generals', config[factionId].generals.filter((_, idx) => idx !== index))
  }

  if (loading) return <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4"><p className="text-sm text-[var(--color-text-muted)]">加载中...</p></div>

  if (!config) return <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4"><p className="text-sm text-red-600">{error ?? '加载失败'}</p></div>

  return (
    <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] shadow-[var(--shadow-panel)] p-4">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <Flag size={16} className="text-[var(--color-accent)]" />
          <h2 className="text-base font-bold text-[var(--color-text-primary)]">阵营配置</h2>
          <span className="text-[11px] text-[var(--color-text-muted)]">{Object.keys(config).length} 个阵营</span>
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

      <div className="grid gap-3">
        {Object.entries(config).map(([factionId, faction]) => (
          <div key={factionId} className="p-3.5 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
            {/* Header */}
            <div className="flex items-center gap-2 mb-3">
              <span className="w-8 h-8 flex items-center justify-center rounded-lg bg-[var(--color-accent-light)] text-[var(--color-accent)] text-xs font-black uppercase">
                {factionId.slice(0, 2)}
              </span>
              <div className="grid flex-1 gap-2 md:grid-cols-[minmax(120px,0.7fr)_minmax(180px,1.4fr)_minmax(80px,0.45fr)]">
                <input
                  type="text"
                  value={faction.name}
                  onChange={(e) => updateFactionField(factionId, 'name', e.target.value)}
                  className="h-8 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 text-sm font-bold text-[var(--color-text-primary)]"
                />
                <input
                  type="text"
                  value={faction.description}
                  onChange={(e) => updateFactionField(factionId, 'description', e.target.value)}
                  className="h-8 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 text-xs text-[var(--color-text-primary)]"
                />
                <input
                  type="text"
                  value={faction.icon}
                  onChange={(e) => updateFactionField(factionId, 'icon', e.target.value)}
                  className="h-8 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 text-xs text-[var(--color-text-primary)]"
                />
              </div>
            </div>

            {/* Traits */}
            <div className="mb-3">
              <span className="text-[10px] font-bold text-[var(--color-text-muted)] uppercase tracking-wider">加成系数</span>
              <div className="grid grid-cols-3 gap-2 mt-1.5">
                {Object.entries(faction.traits).map(([traitKey, value]) => (
                  <label key={traitKey} className="grid gap-0.5">
                    <span className="text-[10px] text-[var(--color-text-muted)]">{TRAIT_LABELS[traitKey] ?? traitKey}</span>
                    <input
                      type="number"
                      step="0.01"
                      value={value}
                      onChange={(e) => updateTrait(factionId, traitKey, parseFloat(e.target.value) || 1)}
                      className="h-7 px-2 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]"
                    />
                  </label>
                ))}
              </div>
            </div>

            {/* Generals */}
            <div>
              <div className="mb-1.5 flex items-center justify-between">
                <span className="text-[10px] font-bold text-[var(--color-text-muted)] uppercase tracking-wider">将领 ({faction.generals.length})</span>
                <button type="button" onClick={() => addGeneral(factionId)} className="inline-flex items-center gap-1 rounded-lg border border-[var(--color-border)] px-2 py-1 text-[10px] font-bold text-[var(--color-accent)]">
                  <Plus size={10} />
                  添加
                </button>
              </div>
              <div className="grid gap-1.5">
                {faction.generals.map((g, index) => (
                  <div key={`${g.id}-${index}`} className="grid gap-1.5 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] p-1.5 md:grid-cols-[minmax(120px,1fr)_minmax(100px,1fr)_minmax(100px,1fr)_28px]">
                    <input value={g.id} onChange={(e) => updateGeneral(factionId, index, 'id', e.target.value)} placeholder="general id" className="h-7 rounded border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-2 text-xs text-[var(--color-text-primary)]" />
                    <input value={g.name} onChange={(e) => updateGeneral(factionId, index, 'name', e.target.value)} placeholder="名称" className="h-7 rounded border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-2 text-xs text-[var(--color-text-primary)]" />
                    <input value={g.title} onChange={(e) => updateGeneral(factionId, index, 'title', e.target.value)} placeholder="称号" className="h-7 rounded border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-2 text-xs text-[var(--color-text-primary)]" />
                    <button type="button" onClick={() => removeGeneral(factionId, index)} className="grid h-7 place-items-center rounded text-red-500 hover:bg-red-500/10">
                      <Trash2 size={12} />
                    </button>
                  </div>
                ))}
              </div>
            </div>
          </div>
        ))}
      </div>

      {message && <p className="mt-3 text-xs font-bold text-emerald-600">{message}</p>}
      {error && <p className="mt-3 text-xs font-bold text-red-600">{error}</p>}
    </div>
  )
}
