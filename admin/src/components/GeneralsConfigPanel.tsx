import { useEffect, useState } from 'react'
import { Users, Save, ChevronDown, ChevronUp, AlertCircle } from 'lucide-react'
import { adminApi } from '@/api/admin'

// 将领配置结构
interface GeneralTraitConfig {
  traitId: string
  enabled: boolean
  params: Record<string, number>
}

interface GeneralHeroConfig {
  id: string
  name: string
  faction: string
  title: string
  rarity: string
  enabled: boolean
  buffs: Record<string, number>
  traits: GeneralTraitConfig[]
}

interface GeneralsCommonConfig {
  expCurve: number[]
  levelBuffs: Record<number, Record<string, number>>
}

interface GeneralsConfig {
  enabled: boolean
  common: GeneralsCommonConfig
  heroes: Record<string, GeneralHeroConfig>
}

// 特性元信息（从特性注册表获取）
interface TraitMeta {
  id: string
  name: string
  description: string
  paramSchema: Array<{
    key: string
    label: string
    description: string
    default: number
    min: number
    max: number
    step: number
  }>
}

interface TraitRegistryResponse {
  traits: TraitMeta[]
}

const RARITY_LABELS: Record<string, string> = {
  common: '普通',
  rare: '稀有',
  epic: '史诗',
  legendary: '传说',
}

const RARITY_COLORS: Record<string, string> = {
  common: 'bg-gray-100 text-gray-700',
  rare: 'bg-blue-100 text-blue-700',
  epic: 'bg-purple-100 text-purple-700',
  legendary: 'bg-amber-100 text-amber-700',
}

const FACTION_LABELS: Record<string, string> = {
  wei: '魏',
  shu: '蜀',
  wu: '吴',
}

const BUFF_LABELS: Record<string, string> = {
  productionBonus: '生产加成',
  attackBonus: '攻击加成',
  defenseBonus: '防御加成',
  economyBonus: '经济加成',
  militaryBonus: '军事加成',
}

export default function GeneralsConfigPanel() {
  const [config, setConfig] = useState<GeneralsConfig | null>(null)
  const [traitRegistry, setTraitRegistry] = useState<TraitMeta[]>([])
  const [activeFaction, setActiveFaction] = useState<'wei' | 'shu' | 'wu'>('wei')
  const [expandedHero, setExpandedHero] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  // 加载配置和特性注册表
  useEffect(() => {
    let cancelled = false
    Promise.all([
      adminApi.getGeneralsConfig(),
      adminApi.getGeneralTraitRegistry(),
    ])
      .then(([configData, registryData]) => {
        if (!cancelled) {
          setConfig(configData as GeneralsConfig)
          setTraitRegistry((registryData as TraitRegistryResponse).traits)
        }
      })
      .catch((err) => {
        if (!cancelled) setError(err instanceof Error ? err.message : '加载失败')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => { cancelled = true }
  }, [])

  const handleSave = async () => {
    if (!config) return
    setSaving(true)
    setMessage(null)
    setError(null)
    try {
      const result = await adminApi.updateGeneralsConfig(config)
      setConfig(result as GeneralsConfig)
      setMessage('将领配置已保存')
    } catch (err) {
      setError(err instanceof Error ? err.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  const toggleGlobalEnabled = () => {
    if (!config) return
    setConfig({ ...config, enabled: !config.enabled })
  }

  const toggleHeroEnabled = (heroId: string) => {
    if (!config) return
    const hero = config.heroes[heroId]
    setConfig({
      ...config,
      heroes: {
        ...config.heroes,
        [heroId]: { ...hero, enabled: !hero.enabled },
      },
    })
  }

  const toggleTraitEnabled = (heroId: string, traitIndex: number) => {
    if (!config) return
    const hero = config.heroes[heroId]
    const traits = [...hero.traits]
    traits[traitIndex] = { ...traits[traitIndex], enabled: !traits[traitIndex].enabled }
    setConfig({
      ...config,
      heroes: {
        ...config.heroes,
        [heroId]: { ...hero, traits },
      },
    })
  }

  const updateTraitParam = (heroId: string, traitIndex: number, paramKey: string, value: number) => {
    if (!config) return
    const hero = config.heroes[heroId]
    const traits = [...hero.traits]
    traits[traitIndex] = {
      ...traits[traitIndex],
      params: { ...traits[traitIndex].params, [paramKey]: value },
    }
    setConfig({
      ...config,
      heroes: {
        ...config.heroes,
        [heroId]: { ...hero, traits },
      },
    })
  }

  const updateHeroBuff = (heroId: string, buffKey: string, value: number) => {
    if (!config) return
    const hero = config.heroes[heroId]
    setConfig({
      ...config,
      heroes: {
        ...config.heroes,
        [heroId]: {
          ...hero,
          buffs: { ...hero.buffs, [buffKey]: value },
        },
      },
    })
  }

  if (loading) {
    return (
      <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4">
        <p className="text-sm text-[var(--color-text-muted)]">加载中...</p>
      </div>
    )
  }

  if (!config) {
    return (
      <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4">
        <p className="text-sm text-red-600">{error ?? '加载失败'}</p>
      </div>
    )
  }

  // 按阵营分组将领
  const heroesByFaction = {
    wei: Object.values(config.heroes).filter((h) => h.faction === 'wei'),
    shu: Object.values(config.heroes).filter((h) => h.faction === 'shu'),
    wu: Object.values(config.heroes).filter((h) => h.faction === 'wu'),
  }

  const currentHeroes = heroesByFaction[activeFaction]

  return (
    <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] shadow-[var(--shadow-panel)] p-4">
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <Users size={16} className="text-[var(--color-accent)]" />
          <h2 className="text-base font-bold text-[var(--color-text-primary)]">将领配置</h2>
          <span className="text-[11px] text-[var(--color-text-muted)]">{Object.keys(config.heroes).length} 位将领</span>
        </div>
        <div className="flex items-center gap-2">
          <label className="flex items-center gap-1.5 cursor-pointer">
            <input
              type="checkbox"
              checked={config.enabled}
              onChange={toggleGlobalEnabled}
              className="w-4 h-4 rounded border-[var(--color-border)]"
            />
            <span className="text-xs font-medium text-[var(--color-text-secondary)]">启用将领系统</span>
          </label>
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
      </div>

      {/* Faction Tabs */}
      <div className="flex gap-2 mb-4 border-b border-[var(--color-border)]">
        {(['wei', 'shu', 'wu'] as const).map((faction) => (
          <button
            key={faction}
            type="button"
            onClick={() => setActiveFaction(faction)}
            className={`px-4 py-2 text-sm font-bold transition-all relative ${
              activeFaction === faction
                ? 'text-[var(--color-accent)]'
                : 'text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)]'
            }`}
          >
            {FACTION_LABELS[faction]}
            <span className="ml-1.5 text-[10px] font-normal">({heroesByFaction[faction].length})</span>
            {activeFaction === faction && (
              <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-[var(--color-accent)]" />
            )}
          </button>
        ))}
      </div>

      {/* Hero List */}
      <div className="grid gap-3">
        {currentHeroes.map((hero) => {
          const isExpanded = expandedHero === hero.id
          const traitMetas = hero.traits.map((t) => traitRegistry.find((tm) => tm.id === t.traitId))

          return (
            <div key={hero.id} className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
              {/* Hero Header */}
              <div
                className="flex items-center justify-between p-3.5 cursor-pointer"
                onClick={() => setExpandedHero(isExpanded ? null : hero.id)}
              >
                <div className="flex items-center gap-3">
                  {/* Avatar */}
                  <div className="w-10 h-10 flex items-center justify-center rounded-lg bg-gradient-to-br from-[var(--color-accent)] to-indigo-600 text-white text-sm font-black">
                    {hero.name[0]}
                  </div>

                  <div>
                    <div className="flex items-center gap-2">
                      <strong className="text-sm text-[var(--color-text-primary)]">{hero.name}</strong>
                      <span className="text-[11px] text-[var(--color-text-muted)]">{hero.title}</span>
                      <span className={`px-1.5 py-0.5 rounded text-[10px] font-bold ${RARITY_COLORS[hero.rarity] || RARITY_COLORS.common}`}>
                        {RARITY_LABELS[hero.rarity] || hero.rarity}
                      </span>
                      <span className="px-1.5 py-0.5 rounded text-[10px] font-bold bg-[var(--color-accent-light)] text-[var(--color-accent)]">
                        {FACTION_LABELS[hero.faction] || hero.faction}
                      </span>
                    </div>
                    <div className="flex items-center gap-2 mt-0.5">
                      <span className="text-[10px] text-[var(--color-text-muted)]">
                        {hero.traits.length} 个特性
                      </span>
                      {!hero.enabled && (
                        <span className="text-[10px] text-red-600 font-medium">已禁用</span>
                      )}
                    </div>
                  </div>
                </div>

                <div className="flex items-center gap-2">
                  <label
                    className="flex items-center gap-1.5"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <input
                      type="checkbox"
                      checked={hero.enabled}
                      onChange={() => toggleHeroEnabled(hero.id)}
                      className="w-4 h-4 rounded border-[var(--color-border)]"
                    />
                    <span className="text-xs text-[var(--color-text-muted)]">启用</span>
                  </label>
                  {isExpanded ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
                </div>
              </div>

              {/* Hero Details (Expanded) */}
              {isExpanded && (
                <div className="px-3.5 pb-3.5 space-y-3 border-t border-[var(--color-border)]">
                  {/* Buffs */}
                  {Object.keys(hero.buffs).length > 0 && (
                    <div className="pt-3">
                      <span className="text-[10px] font-bold text-[var(--color-text-muted)] uppercase tracking-wider">
                        固定属性加成
                      </span>
                      <div className="grid grid-cols-3 gap-2 mt-1.5">
                        {Object.entries(hero.buffs).map(([buffKey, value]) => (
                          <label key={buffKey} className="grid gap-0.5">
                            <span className="text-[10px] text-[var(--color-text-muted)]">
                              {BUFF_LABELS[buffKey] || buffKey}
                            </span>
                            <input
                              type="number"
                              step="0.01"
                              value={value}
                              onChange={(e) => updateHeroBuff(hero.id, buffKey, parseFloat(e.target.value) || 0)}
                              className="h-7 px-2 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]"
                            />
                          </label>
                        ))}
                      </div>
                    </div>
                  )}

                  {/* Traits */}
                  <div>
                    <span className="text-[10px] font-bold text-[var(--color-text-muted)] uppercase tracking-wider">
                      特性配置
                    </span>
                    <div className="space-y-2 mt-1.5">
                      {hero.traits.map((trait, traitIndex) => {
                        const meta = traitMetas[traitIndex]
                        const hasInvalidParams = meta?.paramSchema.some((field) => {
                          const value = trait.params[field.key] ?? field.default
                          return value < field.min || value > field.max
                        })

                        return (
                          <div
                            key={traitIndex}
                            className={`p-2.5 rounded-lg border ${
                              trait.enabled
                                ? 'border-[var(--color-accent)]/30 bg-[var(--color-accent-light)]'
                                : 'border-[var(--color-border)] bg-[var(--color-surface)]'
                            }`}
                          >
                            {/* Trait Header */}
                            <div className="flex items-center justify-between mb-2">
                              <div className="flex items-center gap-2">
                                <span className="text-xs font-bold text-[var(--color-text-primary)]">
                                  {meta?.name || trait.traitId}
                                </span>
                                {hasInvalidParams && (
                                  <span className="flex items-center gap-1 text-[10px] text-amber-600">
                                    <AlertCircle size={12} />
                                    参数超出范围
                                  </span>
                                )}
                              </div>
                              <label className="flex items-center gap-1.5">
                                <input
                                  type="checkbox"
                                  checked={trait.enabled}
                                  onChange={() => toggleTraitEnabled(hero.id, traitIndex)}
                                  className="w-3.5 h-3.5 rounded border-[var(--color-border)]"
                                />
                                <span className="text-[10px] text-[var(--color-text-muted)]">启用</span>
                              </label>
                            </div>

                            {/* Trait Description */}
                            {meta?.description && (
                              <p className="text-[10px] text-[var(--color-text-muted)] mb-2">{meta.description}</p>
                            )}

                            {/* Trait Parameters */}
                            {meta && (
                              <div className="grid grid-cols-2 gap-2">
                                {meta.paramSchema.map((field) => {
                                  const value = trait.params[field.key] ?? field.default
                                  const isOutOfRange = value < field.min || value > field.max

                                  return (
                                    <label key={field.key} className="grid gap-0.5">
                                      <div className="flex items-center justify-between">
                                        <span className="text-[10px] text-[var(--color-text-muted)]">
                                          {field.label}
                                        </span>
                                        {isOutOfRange && (
                                          <span className="text-[9px] text-amber-600 font-medium">
                                            范围: {field.min}~{field.max}
                                          </span>
                                        )}
                                      </div>
                                      <input
                                        type="number"
                                        step={field.step}
                                        min={field.min}
                                        max={field.max}
                                        value={value}
                                        onChange={(e) =>
                                          updateTraitParam(hero.id, traitIndex, field.key, parseFloat(e.target.value) || 0)
                                        }
                                        className={`h-7 px-2 rounded-lg text-xs border ${
                                          isOutOfRange
                                            ? 'border-amber-500 bg-amber-50'
                                            : 'border-[var(--color-border)] bg-[var(--color-surface)]'
                                        } text-[var(--color-text-primary)]`}
                                        title={field.description}
                                      />
                                    </label>
                                  )
                                })}
                              </div>
                            )}
                          </div>
                        )
                      })}
                    </div>
                  </div>
                </div>
              )}
            </div>
          )
        })}
      </div>

      {/* Messages */}
      {message && (
        <p className="mt-3 text-xs font-bold text-emerald-600">{message}</p>
      )}
      {error && (
        <p className="mt-3 text-xs font-bold text-red-600">{error}</p>
      )}
    </div>
  )
}
