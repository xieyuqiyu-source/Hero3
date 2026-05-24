import { useEffect, useState } from 'react'
import { Flag, Save } from 'lucide-react'
import { adminApi } from '@/api/admin'

interface GeneralInfo {
  id: string
  name: string
  title: string
}

interface FactionConfig {
  name: string
  description: string
  icon: string
  traits: Record<string, number>
  generals: GeneralInfo[]
}

type FactionsConfig = Record<string, FactionConfig>

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
              <div>
                <strong className="text-sm text-[var(--color-text-primary)]">{faction.name}</strong>
                <p className="text-[11px] text-[var(--color-text-muted)]">{faction.description}</p>
              </div>
            </div>

            {/* Traits */}
            <div className="mb-3">
              <span className="text-[10px] font-bold text-[var(--color-text-muted)] uppercase tracking-wider">加成系数</span>
              <div className="grid grid-cols-3 gap-2 mt-1.5">
                {Object.entries(faction.traits).map(([traitKey, value]) => (
                  <label key={traitKey} className="grid gap-0.5">
                    <span className="text-[10px] text-[var(--color-text-muted)]">{traitKey}</span>
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
              <span className="text-[10px] font-bold text-[var(--color-text-muted)] uppercase tracking-wider">将领 ({faction.generals.length})</span>
              <div className="flex flex-wrap gap-1.5 mt-1.5">
                {faction.generals.map((g) => (
                  <span key={g.id} className="px-2 py-1 rounded-lg text-[11px] font-bold bg-[var(--color-gold-soft)] text-amber-700">
                    {g.name}
                    <span className="ml-1 text-[10px] font-normal text-amber-600/70">{g.title}</span>
                  </span>
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
