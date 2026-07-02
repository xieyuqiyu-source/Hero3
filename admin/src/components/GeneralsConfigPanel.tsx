import { useEffect, useState } from 'react'
import { Users, Save, ChevronDown, ChevronUp, AlertCircle, Plus, Trash2 } from 'lucide-react'
import { adminApi } from '@/api/admin'
import type { GeneralHeroConfig, GeneralsConfig, TraitMeta } from '@/types'

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
  capacityBonus: '容量加成',
  recruitSpeedBonus: '征兵速度',
  marchSpeedBonus: '行军速度',
  buildSpeedBonus: '建造速度',
}

const BUFF_OPTIONS = Object.keys(BUFF_LABELS)

const TRAIT_TYPE_LABELS: Record<'special' | 'bonus', string> = {
  special: '特殊特性',
  bonus: '加成特性',
}

const TRAIT_SCOPE_LABELS: Record<string, string> = {
  self_army: '自己的参战队伍',
  enemy_army: '敌方参战队伍',
  all_army: '全体参战队伍',
  reinforcement_self: '自己的增援队伍',
  defense_self: '守城队伍',
  attack_self: '出征队伍',
}

const TRAIT_TARGET_LABELS: Record<string, string> = {
  infantry: '步兵',
  cavalry: '骑兵',
  archer: '弓兵',
  special: '特殊兵',
  huWei: '虎卫',
  huBaoQi: '虎豹骑',
  baWangQi: '霸王骑',
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
          setConfig(configData)
          setTraitRegistry(registryData.traits)
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
      setConfig(result)
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
    const key = traitIndex === 0 ? 'specialTrait' : 'bonusTrait'
    const trait = hero[key]
    setConfig({
      ...config,
      heroes: {
        ...config.heroes,
        [heroId]: { ...hero, [key]: { ...trait, enabled: !trait.enabled } },
      },
    })
  }

  const updateTraitParam = (heroId: string, traitIndex: number, paramKey: string, value: number) => {
    if (!config) return
    const hero = config.heroes[heroId]
    const key = traitIndex === 0 ? 'specialTrait' : 'bonusTrait'
    const trait = hero[key]
    setConfig({
      ...config,
      heroes: {
        ...config.heroes,
        [heroId]: { ...hero, [key]: { ...trait, params: { ...trait.params, [paramKey]: value } } },
      },
    })
  }

  const updateTraitField = (heroId: string, traitIndex: number, field: 'scope' | 'targetUnitType', value: string) => {
    if (!config) return
    const hero = config.heroes[heroId]
    const key = traitIndex === 0 ? 'specialTrait' : 'bonusTrait'
    const trait = hero[key]
    setConfig({
      ...config,
      heroes: {
        ...config.heroes,
        [heroId]: { ...hero, [key]: { ...trait, [field]: value } },
      },
    })
  }

  const updateTraitId = (heroId: string, traitIndex: number, traitId: string) => {
    if (!config) return
    const hero = config.heroes[heroId]
    const meta = traitRegistry.find((item) => item.id === traitId)
    const key = traitIndex === 0 ? 'specialTrait' : 'bonusTrait'
    const params = Object.fromEntries((meta?.paramSchema ?? []).map((field) => [field.key, field.default]))
    setConfig({
      ...config,
      heroes: { ...config.heroes, [heroId]: { ...hero, [key]: { traitId, traitType: meta?.traitType ?? (traitIndex === 0 ? 'special' : 'bonus'), enabled: true, params } } },
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

  const updateHeroField = <K extends keyof GeneralHeroConfig,>(heroId: string, field: K, value: GeneralHeroConfig[K]) => {
    if (!config) return
    const hero = config.heroes[heroId]
    setConfig({
      ...config,
      heroes: { ...config.heroes, [heroId]: { ...hero, [field]: value } },
    })
  }

  const addHeroBuff = (heroId: string) => {
    if (!config) return
    const hero = config.heroes[heroId]
    const buffKey = BUFF_OPTIONS.find((key) => !(key in hero.buffs)) ?? `customBuff${Object.keys(hero.buffs).length + 1}`
    setConfig({
      ...config,
      heroes: { ...config.heroes, [heroId]: { ...hero, buffs: { ...hero.buffs, [buffKey]: 0 } } },
    })
  }

  const removeHeroBuff = (heroId: string, buffKey: string) => {
    if (!config) return
    const hero = config.heroes[heroId]
    const buffs = { ...hero.buffs }
    delete buffs[buffKey]
    setConfig({
      ...config,
      heroes: { ...config.heroes, [heroId]: { ...hero, buffs } },
    })
  }

  const updateLevelBuff = (level: string, buffKey: string, value: number) => {
    if (!config) return
    setConfig({
      ...config,
      common: {
        ...config.common,
        levelBuffs: {
          ...config.common.levelBuffs,
          [level]: { ...(config.common.levelBuffs[level] ?? {}), [buffKey]: value },
        },
      },
    })
  }

  const addLevelBuff = () => {
    if (!config) return
    const existing = Object.keys(config.common.levelBuffs).map(Number).filter(Number.isFinite)
    const level = String((existing.length > 0 ? Math.max(...existing) : 0) + 1)
    setConfig({
      ...config,
      common: {
        ...config.common,
        levelBuffs: { ...config.common.levelBuffs, [level]: { productionBonus: 0 } },
      },
    })
  }

  const removeLevelBuff = (level: string) => {
    if (!config) return
    const levelBuffs = { ...config.common.levelBuffs }
    delete levelBuffs[level]
    setConfig({ ...config, common: { ...config.common, levelBuffs } })
  }

  const updateExpCurve = (value: string) => {
    if (!config) return
    const expCurve = value
      .split(/[\s,，]+/)
      .map((item) => parseInt(item, 10))
      .filter((item) => Number.isFinite(item))
    setConfig({
      ...config,
      common: {
        ...config.common,
        expCurve,
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
      <div className="mb-4 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-3">
        <div className="flex items-center justify-between gap-3 mb-2">
          <span className="text-xs font-bold text-[var(--color-text-primary)]">经验曲线</span>
          <span className="text-[10px] text-[var(--color-text-muted)]">{config.common.expCurve.length} 级配置</span>
        </div>
        <textarea
          value={config.common.expCurve.join(', ')}
          onChange={(e) => updateExpCurve(e.target.value)}
          rows={2}
          className="w-full px-2.5 py-2 rounded-lg text-xs font-mono border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)] outline-none focus:border-[var(--color-accent)] resize-y"
          placeholder="0, 100, 300, 600"
        />
        <p className="mt-1.5 text-[10px] text-[var(--color-text-muted)]">
          第 N 项表示升到 N 级所需累计经验；未配置的高等级使用默认公式。
        </p>
      </div>

      <div className="mb-4 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-3">
        <div className="mb-2 flex items-center justify-between gap-2">
          <span className="text-xs font-bold text-[var(--color-text-primary)]">等级通用加成</span>
          <button type="button" onClick={addLevelBuff} className="inline-flex items-center gap-1 rounded-lg border border-[var(--color-border)] px-2 py-1 text-[10px] font-bold text-[var(--color-accent)]">
            <Plus size={10} />
            添加等级
          </button>
        </div>
        <div className="grid gap-2 md:grid-cols-2">
          {Object.entries(config.common.levelBuffs)
            .sort(([a], [b]) => Number(a) - Number(b))
            .map(([level, buffs]) => (
              <div key={level} className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] p-2">
                <div className="mb-1.5 flex items-center justify-between">
                  <span className="text-[10px] font-bold text-[var(--color-text-muted)]">Lv.{level}</span>
                  <button type="button" onClick={() => removeLevelBuff(level)} className="grid h-6 w-6 place-items-center rounded text-red-500 hover:bg-red-500/10">
                    <Trash2 size={11} />
                  </button>
                </div>
                <div className="grid grid-cols-2 gap-1.5">
                  {BUFF_OPTIONS.map((buffKey) => (
                    <label key={buffKey} className="grid gap-0.5">
                      <span className="text-[9px] text-[var(--color-text-muted)]">{BUFF_LABELS[buffKey]}</span>
                      <input
                        type="number"
                        step="0.01"
                        value={buffs[buffKey] ?? 0}
                        onChange={(e) => updateLevelBuff(level, buffKey, parseFloat(e.target.value) || 0)}
                        className="h-6 rounded border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-1.5 text-[11px] text-[var(--color-text-primary)]"
                      />
                    </label>
                  ))}
                </div>
              </div>
            ))}
        </div>
      </div>

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
          const heroTraits = [hero.specialTrait, hero.bonusTrait]
          const traitMetas = heroTraits.map((t) => traitRegistry.find((tm) => tm.id === t.traitId))

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
                        特殊特性 + 加成特性
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
                  <div className="grid gap-2 pt-3 md:grid-cols-[minmax(120px,1fr)_minmax(120px,1fr)_minmax(110px,0.7fr)_minmax(110px,0.7fr)]">
                    <input value={hero.name} onChange={(e) => updateHeroField(hero.id, 'name', e.target.value)} className="h-8 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 text-xs text-[var(--color-text-primary)]" />
                    <input value={hero.title} onChange={(e) => updateHeroField(hero.id, 'title', e.target.value)} className="h-8 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 text-xs text-[var(--color-text-primary)]" />
                    <select value={hero.faction} onChange={(e) => updateHeroField(hero.id, 'faction', e.target.value)} className="h-8 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 text-xs text-[var(--color-text-primary)]">
                      {Object.entries(FACTION_LABELS).map(([key, label]) => <option key={key} value={key}>{label}</option>)}
                      {!FACTION_LABELS[hero.faction] && <option value={hero.faction}>{hero.faction}</option>}
                    </select>
                    <select value={hero.rarity} onChange={(e) => updateHeroField(hero.id, 'rarity', e.target.value)} className="h-8 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 text-xs text-[var(--color-text-primary)]">
                      {Object.entries(RARITY_LABELS).map(([key, label]) => <option key={key} value={key}>{label}</option>)}
                      {!RARITY_LABELS[hero.rarity] && <option value={hero.rarity}>{hero.rarity}</option>}
                    </select>
                  </div>
                  {/* Buffs */}
                  <div>
                    <div className="flex items-center justify-between">
                      <span className="text-[10px] font-bold text-[var(--color-text-muted)] uppercase tracking-wider">
                        固定属性加成
                      </span>
                      <button type="button" onClick={() => addHeroBuff(hero.id)} className="inline-flex items-center gap-1 rounded-lg border border-[var(--color-border)] px-2 py-1 text-[10px] font-bold text-[var(--color-accent)]">
                        <Plus size={10} />
                        添加
                      </button>
                    </div>
                      <div className="grid grid-cols-3 gap-2 mt-1.5">
                        {Object.entries(hero.buffs).map(([buffKey, value]) => (
                          <label key={buffKey} className="grid gap-0.5">
                            <span className="flex items-center justify-between text-[10px] text-[var(--color-text-muted)]">
                              {BUFF_LABELS[buffKey] || buffKey}
                              <button type="button" onClick={() => removeHeroBuff(hero.id, buffKey)} className="text-red-500">
                                <Trash2 size={10} />
                              </button>
                            </span>
                            <input type="number" step="0.01" value={value} onChange={(e) => updateHeroBuff(hero.id, buffKey, parseFloat(e.target.value) || 0)} className="h-7 px-2 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]" />
                          </label>
                        ))}
                      </div>
                  </div>

                  {/* Traits */}
                  <div>
                    <div className="flex items-center justify-between">
                      <span className="text-[10px] font-bold text-[var(--color-text-muted)] uppercase tracking-wider">双特性配置</span>
                    </div>
                    <div className="space-y-2 mt-1.5">
                      {heroTraits.map((trait, traitIndex) => {
                        const meta = traitMetas[traitIndex]
                        const requiredType = traitIndex === 0 ? 'special' : 'bonus'
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
                                <span className="rounded bg-[var(--color-surface)] px-2 py-1 text-[10px] font-bold text-[var(--color-text-muted)]">
                                  {TRAIT_TYPE_LABELS[requiredType]}
                                </span>
                                <select value={trait.traitId} onChange={(e) => updateTraitId(hero.id, traitIndex, e.target.value)} className="h-7 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 text-xs font-bold text-[var(--color-text-primary)]">
                                  {traitRegistry.filter((item) => item.traitType === requiredType).map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}
                                  {!meta && <option value={trait.traitId}>{trait.traitId}</option>}
                                </select>
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

                            <div className="mb-2 grid grid-cols-2 gap-2">
                              <label className="grid gap-0.5">
                                <span className="text-[10px] text-[var(--color-text-muted)]">作用范围</span>
                                <span className="text-[9px] font-medium text-[var(--color-accent)]">
                                  {formatTraitScope(trait.scope)}
                                </span>
                                <input
                                  value={trait.scope ?? ''}
                                  onChange={(e) => updateTraitField(hero.id, traitIndex, 'scope', e.target.value)}
                                  className="h-7 px-2 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]"
                                  placeholder="留空默认；可填自己的参战队伍、自己的增援队伍"
                                />
                              </label>
                              <label className="grid gap-0.5">
                                <span className="text-[10px] text-[var(--color-text-muted)]">目标兵种</span>
                                <span className="text-[9px] font-medium text-[var(--color-accent)]">
                                  {formatTraitTarget(trait.targetUnitType)}
                                </span>
                                <input
                                  value={trait.targetUnitType ?? ''}
                                  onChange={(e) => updateTraitField(hero.id, traitIndex, 'targetUnitType', e.target.value)}
                                  className="h-7 px-2 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]"
                                  placeholder="留空不限；可填步兵、骑兵、特殊兵或具体兵种"
                                />
                              </label>
                            </div>

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

// formatTraitScope 把核心作用范围值转成 GM 可读中文。
function formatTraitScope(value?: string) {
  if (!value) return '默认范围'
  return TRAIT_SCOPE_LABELS[value] ?? value
}

// formatTraitTarget 把核心目标兵种值转成 GM 可读中文。
function formatTraitTarget(value?: string) {
  if (!value) return '不限制兵种'
  return TRAIT_TARGET_LABELS[value] ?? value
}
