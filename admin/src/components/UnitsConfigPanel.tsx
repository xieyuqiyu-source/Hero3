import { useEffect, useState } from 'react'
import { Shield, Save, ChevronDown, ChevronUp } from 'lucide-react'
import { adminApi } from '@/api/admin'

interface UnitConfig {
  name: string
  description: string
  category: string
  icon: string
  stats: Record<string, number>
  cost: Record<string, number>
  trainSeconds: number
  unlock: Record<string, any>
}

type FactionUnits = Record<string, UnitConfig>
type UnitsConfig = Record<string, FactionUnits>

const CATEGORY_LABELS: Record<string, string> = {
  infantry: '步兵',
  cavalry: '骑兵',
  siege: '攻城',
  special: '特殊',
}

const STAT_LABELS: Record<string, string> = {
  attack: '攻击',
  infantryDefense: '步防',
  cavalryDefense: '骑防',
  speed: '速度',
  carryCapacity: '负重',
  upkeep: '粮耗',
}

const RES_LABELS: Record<string, string> = {
  wood: '木材',
  stone: '石料',
  iron: '铁矿',
  food: '粮食',
}

export default function UnitsConfigPanel() {
  const [config, setConfig] = useState<UnitsConfig | null>(null)
  const [activeFaction, setActiveFaction] = useState<string>('')
  const [expandedUnit, setExpandedUnit] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    adminApi.getUnitsConfig()
      .then((data) => {
        if (cancelled) return
        const typed = data as UnitsConfig
        setConfig(typed)
        const factions = Object.keys(typed)
        if (factions.length > 0) setActiveFaction(factions[0])
      })
      .catch((err) => { if (!cancelled) setError(err instanceof Error ? err.message : '加载失败') })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [])

  const handleSave = async () => {
    if (!config || !activeFaction) return
    setSaving(true)
    setMessage(null)
    setError(null)
    try {
      const result = await adminApi.updateUnitsConfig(activeFaction, config[activeFaction])
      setConfig((prev) => prev ? { ...prev, [activeFaction]: result as FactionUnits } : prev)
      setMessage(`${activeFaction} 兵种已保存`)
    } catch (err) {
      setError(err instanceof Error ? err.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  const updateStat = (unitId: string, statKey: string, value: number) => {
    if (!config || !activeFaction) return
    setConfig({
      ...config,
      [activeFaction]: {
        ...config[activeFaction],
        [unitId]: {
          ...config[activeFaction][unitId],
          stats: { ...config[activeFaction][unitId].stats, [statKey]: value },
        },
      },
    })
  }

  const updateCost = (unitId: string, resKey: string, value: number) => {
    if (!config || !activeFaction) return
    setConfig({
      ...config,
      [activeFaction]: {
        ...config[activeFaction],
        [unitId]: {
          ...config[activeFaction][unitId],
          cost: { ...config[activeFaction][unitId].cost, [resKey]: value },
        },
      },
    })
  }

  const updateTrainSeconds = (unitId: string, value: number) => {
    if (!config || !activeFaction) return
    setConfig({
      ...config,
      [activeFaction]: {
        ...config[activeFaction],
        [unitId]: { ...config[activeFaction][unitId], trainSeconds: value },
      },
    })
  }

  if (loading) return <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4"><p className="text-sm text-[var(--color-text-muted)]">加载中...</p></div>
  if (!config) return <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4"><p className="text-sm text-red-600">{error ?? '加载失败'}</p></div>

  const factions = Object.keys(config)
  const units = config[activeFaction] ?? {}

  return (
    <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] shadow-[var(--shadow-panel)] p-4">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <Shield size={16} className="text-[var(--color-accent)]" />
          <h2 className="text-base font-bold text-[var(--color-text-primary)]">兵种配置</h2>
          <span className="text-[11px] text-[var(--color-text-muted)]">{Object.keys(units).length} 个兵种</span>
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

      {/* Faction Tabs */}
      <div className="flex gap-1 p-1 rounded-xl bg-[var(--color-surface-dim)] border border-[var(--color-border)] w-fit mb-4">
        {factions.map((faction) => (
          <button
            key={faction}
            type="button"
            onClick={() => { setActiveFaction(faction); setExpandedUnit(null) }}
            className={`
              px-3 py-1.5 rounded-lg text-xs font-bold cursor-pointer
              transition-all duration-200 border
              ${activeFaction === faction
                ? 'bg-[var(--color-surface)] text-[var(--color-accent)] border-[var(--color-border)] shadow-sm'
                : 'text-[var(--color-text-secondary)] border-transparent hover:text-[var(--color-text-primary)]'
              }
            `}
          >
            {faction}
          </button>
        ))}
      </div>

      {/* Units List */}
      <div className="grid gap-2">
        {Object.entries(units).map(([unitId, unit]) => {
          const isExpanded = expandedUnit === unitId
          return (
            <div key={unitId} className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] overflow-hidden">
              {/* Unit Header */}
              <button
                type="button"
                onClick={() => setExpandedUnit(isExpanded ? null : unitId)}
                className="w-full flex items-center gap-3 px-3 py-2.5 cursor-pointer hover:bg-white/50 dark:hover:bg-white/5 transition-colors"
              >
                <span className={`
                  px-1.5 py-0.5 rounded text-[9px] font-black uppercase
                  ${unit.category === 'infantry' ? 'bg-blue-500/15 text-blue-700' :
                    unit.category === 'cavalry' ? 'bg-amber-500/15 text-amber-700' :
                    unit.category === 'siege' ? 'bg-red-500/15 text-red-700' :
                    'bg-purple-500/15 text-purple-700'}
                `}>
                  {CATEGORY_LABELS[unit.category] ?? unit.category}
                </span>
                <strong className="text-sm text-[var(--color-text-primary)] flex-1 text-left">{unit.name}</strong>
                <div className="flex items-center gap-3 text-[10px] text-[var(--color-text-muted)]">
                  <span>攻{unit.stats.attack}</span>
                  <span>步防{unit.stats.infantryDefense}</span>
                  <span>骑防{unit.stats.cavalryDefense}</span>
                  <span>速{unit.stats.speed}</span>
                </div>
                {isExpanded ? <ChevronUp size={14} className="text-[var(--color-text-muted)]" /> : <ChevronDown size={14} className="text-[var(--color-text-muted)]" />}
              </button>

              {/* Expanded Detail */}
              {isExpanded && (
                <div className="px-3 pb-3 border-t border-[var(--color-border)]">
                  <p className="text-[11px] text-[var(--color-text-secondary)] mt-2 mb-3">{unit.description}</p>

                  {/* Stats */}
                  <div className="mb-3">
                    <span className="text-[10px] font-bold text-[var(--color-text-muted)] uppercase">属性</span>
                    <div className="grid grid-cols-3 sm:grid-cols-6 gap-1.5 mt-1">
                      {Object.entries(unit.stats).map(([key, val]) => (
                        <label key={key} className="grid gap-0.5">
                          <span className="text-[9px] text-[var(--color-text-muted)]">{STAT_LABELS[key] ?? key}</span>
                          <input
                            type="number"
                            value={val}
                            onChange={(e) => updateStat(unitId, key, parseInt(e.target.value) || 0)}
                            className="h-6 px-1.5 rounded text-[11px] border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)] w-full"
                          />
                        </label>
                      ))}
                    </div>
                  </div>

                  {/* Cost */}
                  <div className="mb-3">
                    <span className="text-[10px] font-bold text-[var(--color-text-muted)] uppercase">训练消耗</span>
                    <div className="grid grid-cols-5 gap-1.5 mt-1">
                      {Object.entries(unit.cost).map(([res, val]) => (
                        <label key={res} className="grid gap-0.5">
                          <span className="text-[9px] text-[var(--color-text-muted)]">{RES_LABELS[res] ?? res}</span>
                          <input
                            type="number"
                            value={val}
                            onChange={(e) => updateCost(unitId, res, parseInt(e.target.value) || 0)}
                            className="h-6 px-1.5 rounded text-[11px] border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)] w-full"
                          />
                        </label>
                      ))}
                      <label className="grid gap-0.5">
                        <span className="text-[9px] text-[var(--color-text-muted)]">训练秒</span>
                        <input
                          type="number"
                          value={unit.trainSeconds}
                          onChange={(e) => updateTrainSeconds(unitId, parseInt(e.target.value) || 0)}
                          className="h-6 px-1.5 rounded text-[11px] border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)] w-full"
                        />
                      </label>
                    </div>
                  </div>

                  {/* Unlock */}
                  <div className="text-[10px] text-[var(--color-text-muted)]">
                    解锁条件：{unit.unlock.building} Lv.{unit.unlock.level}
                  </div>
                </div>
              )}
            </div>
          )
        })}
      </div>

      {message && <p className="mt-3 text-xs font-bold text-emerald-600">{message}</p>}
      {error && <p className="mt-3 text-xs font-bold text-red-600">{error}</p>}
    </div>
  )
}
