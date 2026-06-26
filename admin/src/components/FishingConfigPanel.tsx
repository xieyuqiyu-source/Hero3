// 仙池垂钓 GM 配置面板，提供标准模板选择与配置预览。
import { useEffect, useMemo, useState } from 'react'
import { BarChart3, Fish, Plus, Save, Trash2 } from 'lucide-react'
import { adminApi } from '@/api/admin'
import type { FishingBaitConfig, FishingConfig, FishingFishConfig, FishingRarityConfig, UnitConfig } from '@/types'

const RARITY_LABELS: Record<string, string> = {
  common: '普通',
  rare: '稀有',
  epic: '史诗',
  legendary: '传说',
}

const CUSTOM_VALUE = '__custom__'

const RARITY_PRESETS: Record<string, FishingRarityConfig> = {
  common: { label: '普通', color: 'text-slate-500', bg: 'bg-slate-500/10', border: 'border-slate-500/20', weight: 65, glow: '' },
  rare: { label: '稀有', color: 'text-blue-500', bg: 'bg-blue-500/10', border: 'border-blue-500/20', weight: 22, glow: 'shadow-[0_0_15px_rgba(59,130,246,0.3)]' },
  epic: { label: '史诗', color: 'text-purple-500', bg: 'bg-purple-500/10', border: 'border-purple-500/20', weight: 8, glow: 'shadow-[0_0_25px_rgba(168,85,247,0.4)]' },
  legendary: { label: '传说', color: 'text-amber-500', bg: 'bg-amber-500/10', border: 'border-amber-500/20', weight: 1.5, glow: 'shadow-[0_0_40px_rgba(245,158,11,0.5)]' },
}

const RARITY_STYLE_OPTIONS: Record<keyof Pick<FishingRarityConfig, 'color' | 'bg' | 'border' | 'glow'>, Array<{ value: string; label: string }>> = {
  color: [
    { value: 'text-slate-500', label: '普通灰' },
    { value: 'text-blue-500', label: '稀有蓝' },
    { value: 'text-purple-500', label: '史诗紫' },
    { value: 'text-amber-500', label: '传说金' },
    { value: 'text-emerald-500', label: '灵气绿' },
    { value: 'text-rose-500', label: '赤炎红' },
  ],
  bg: [
    { value: 'bg-slate-500/10', label: '普通灰底' },
    { value: 'bg-blue-500/10', label: '稀有蓝底' },
    { value: 'bg-purple-500/10', label: '史诗紫底' },
    { value: 'bg-amber-500/10', label: '传说金底' },
    { value: 'bg-emerald-500/10', label: '灵气绿底' },
    { value: 'bg-rose-500/10', label: '赤炎红底' },
  ],
  border: [
    { value: 'border-slate-500/20', label: '普通灰边' },
    { value: 'border-blue-500/20', label: '稀有蓝边' },
    { value: 'border-purple-500/20', label: '史诗紫边' },
    { value: 'border-amber-500/20', label: '传说金边' },
    { value: 'border-emerald-500/20', label: '灵气绿边' },
    { value: 'border-rose-500/20', label: '赤炎红边' },
  ],
  glow: [
    { value: '', label: '无光效' },
    { value: 'shadow-[0_0_15px_rgba(59,130,246,0.3)]', label: '稀有蓝光' },
    { value: 'shadow-[0_0_25px_rgba(168,85,247,0.4)]', label: '史诗紫光' },
    { value: 'shadow-[0_0_40px_rgba(245,158,11,0.5)]', label: '传说金光' },
    { value: 'shadow-[0_0_20px_rgba(16,185,129,0.35)]', label: '灵气绿光' },
    { value: 'shadow-[0_0_20px_rgba(244,63,94,0.35)]', label: '赤炎红光' },
  ],
}

const RARITY_PREVIEW_COLORS: Record<string, string> = {
  common: '#64748b',
  rare: '#3b82f6',
  epic: '#a855f7',
  legendary: '#f59e0b',
}

const BAIT_PRESETS: FishingBaitConfig[] = [
  { id: 'coarse', name: '粗饵', tier: '一阶', description: '低成本，命中框较窄', rarityBoost: 1, cityGoldCost: 0, biteChance: 0.72, biteWindowMs: 1500, sweetStart: 67, sweetEnd: 77 },
  { id: 'shrimp', name: '灵虾', tier: '二阶', description: '稀有提升，命中更稳', rarityBoost: 1.2, cityGoldCost: 30, biteChance: 0.8, biteWindowMs: 1850, sweetStart: 62, sweetEnd: 82 },
  { id: 'golden', name: '金鳞饵', tier: '三阶', description: '史诗提升，容错更高', rarityBoost: 1.55, cityGoldCost: 120, biteChance: 0.88, biteWindowMs: 2200, sweetStart: 56, sweetEnd: 86 },
  { id: 'dragon', name: '龙涎饵', tier: '四阶', description: '传说提升，命中框最大', rarityBoost: 2.15, cityGoldCost: 500, biteChance: 0.95, biteWindowMs: 2600, sweetStart: 48, sweetEnd: 90 },
]

const FISH_PRESETS: FishingFishConfig[] = [
  { name: '草鱼', rarity: 'common', reward: '青州军', rewardAmount: 300, description: '常见的淡水鱼，肉质鲜美', emoji: '🐟' },
  { name: '青虾', rarity: 'common', reward: '影卫', rewardAmount: 300, description: '江东水泽常见的小虾，行动灵巧', emoji: '🦐' },
  { name: '鲤鱼', rarity: 'common', reward: '贪狼营', rewardAmount: 300, description: '跃龙门的吉祥之鱼', emoji: '🐠' },
  { name: '鲈鱼', rarity: 'common', reward: '禁卫甲士', rewardAmount: 250, description: '清蒸最佳的上等食材', emoji: '🐡' },
  { name: '江鲈', rarity: 'common', reward: '修罗', rewardAmount: 250, description: '江东水域的凶猛鱼种', emoji: '🐡' },
  { name: '锦鲤', rarity: 'common', reward: '麒麟卫', rewardAmount: 250, description: '色彩斑斓的观赏鱼', emoji: '🎏' },
  { name: '泥鳅', rarity: 'common', reward: '青州军', rewardAmount: 400, description: '滑不溜秋但营养丰富', emoji: '🪱' },
  { name: '金龙鱼', rarity: 'rare', reward: '骁骑营', rewardAmount: 1800, description: '金光闪闪的珍贵鱼种', emoji: '✨' },
  { name: '赤鳞鱼', rarity: 'rare', reward: '神风', rewardAmount: 1800, description: '赤鳞如火，游速极快', emoji: '✨' },
  { name: '银鲨', rarity: 'rare', reward: '西凉铁骑', rewardAmount: 1800, description: '银色鳞片如铠甲般坚硬', emoji: '🦈' },
  { name: '玄武龟', rarity: 'rare', reward: '青龙军', rewardAmount: 2800, description: '传说中玄武的后裔', emoji: '🐢' },
  { name: '玄甲龟', rarity: 'rare', reward: '朱雀骑', rewardAmount: 2200, description: '龟甲坚厚，形似重骑列阵', emoji: '🐢' },
  { name: '雷电鳗', rarity: 'rare', reward: '虎卫', rewardAmount: 2200, description: '体内蕴含雷电之力', emoji: '⚡' },
  { name: '九尾金鲤', rarity: 'rare', reward: '骁骑营', rewardAmount: 3500, description: '九条尾鳍如扇般展开', emoji: '🌊' },
  { name: '虎鲸', rarity: 'epic', reward: '虎豹骑', rewardAmount: 12000, description: '海中霸主，力量惊人', emoji: '🐋' },
  { name: '江海霸主', rarity: 'epic', reward: '霸王骑', rewardAmount: 12000, description: '横行江海的巨兽，气势压人', emoji: '🐋' },
  { name: '蛟龙', rarity: 'epic', reward: '南蛮象', rewardAmount: 12000, description: '即将化龙的水中神兽', emoji: '🐲' },
  { name: '鳌匠鱼', rarity: 'epic', reward: '建筑师', rewardAmount: 300, description: '背负石纹，传说能引水筑基、固城修墙', emoji: '🪨' },
  { name: '凤凰鱼', rarity: 'epic', reward: '木牛流马', rewardAmount: 300, description: '浴火重生的神秘鱼种', emoji: '🔥' },
  { name: '风水灵鱼', rarity: 'epic', reward: '风水师', rewardAmount: 300, description: '灵气绕身，据说能辨山川水势', emoji: '🔥' },
  { name: '白泽', rarity: 'epic', reward: '虎豹骑', rewardAmount: 18000, description: '通晓万物的上古瑞兽', emoji: '🦄' },
  { name: '鲲鹏', rarity: 'legendary', reward: '汉室宗亲', rewardAmount: 4000, description: '北冥有鱼，其名为鲲', emoji: '🌌' },
  { name: '江东龙王', rarity: 'legendary', reward: '太平士', rewardAmount: 2000, description: '镇守江东水脉的传说灵物', emoji: '🐉' },
  { name: '神龙', rarity: 'legendary', reward: '士族', rewardAmount: 2000, description: '万灵之首，至高无上', emoji: '🐉' },
  { name: '混沌', rarity: 'legendary', reward: '汉室宗亲', rewardAmount: 8000, description: '天地未分之初的原始神兽', emoji: '🌀' },
]

const FISH_ICON_OPTIONS = Array.from(new Set(FISH_PRESETS.map((fish) => fish.emoji))).map((emoji) => ({ value: emoji, label: emoji }))

const TIER_OPTIONS = ['一阶', '二阶', '三阶', '四阶'].map((tier) => ({ value: tier, label: tier }))

const emptyFish = (): FishingFishConfig => ({
  name: '新鱼获',
  rarity: 'common',
  reward: '',
  rewardAmount: 100,
  description: '',
  emoji: '🐟',
})

const emptyBait = (): FishingBaitConfig => ({
  ...BAIT_PRESETS[0],
  id: `bait_${Date.now()}`,
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

  const applyRarityPreset = (rarityId: string, presetId: string) => {
    if (!config || presetId === CUSTOM_VALUE) return
    const preset = RARITY_PRESETS[presetId]
    if (!preset) return
    updateConfig({
      ...config,
      rarities: {
        ...config.rarities,
        [rarityId]: { ...preset },
      },
    })
  }

  const updateBait = (index: number, field: keyof FishingBaitConfig, value: string | number) => {
    if (!config) return
    const baits = [...config.baits]
    baits[index] = { ...baits[index], [field]: value }
    updateConfig({ ...config, baits })
  }

  const applyBaitPreset = (index: number, presetId: string) => {
    if (!config || presetId === CUSTOM_VALUE) return
    const preset = BAIT_PRESETS.find((bait) => bait.id === presetId)
    if (!preset) return
    const baits = [...config.baits]
    baits[index] = { ...preset }
    updateConfig({ ...config, baits })
  }

  const addBait = () => {
    if (!config) return
    updateConfig({ ...config, baits: [...config.baits, emptyBait()] })
  }

  const removeBait = (index: number) => {
    if (!config) return
    updateConfig({ ...config, baits: config.baits.filter((_, i) => i !== index) })
  }

  const updateFish = (index: number, field: keyof FishingFishConfig, value: string | number) => {
    if (!config) return
    const fishPool = [...config.fishPool]
    fishPool[index] = { ...fishPool[index], [field]: value }
    updateConfig({ ...config, fishPool })
  }

  const applyFishPreset = (index: number, presetKey: string) => {
    if (!config || presetKey === CUSTOM_VALUE) return
    const preset = FISH_PRESETS[Number(presetKey)]
    if (!preset) return
    const fishPool = [...config.fishPool]
    fishPool[index] = { ...preset }
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

      <FishingConfigPreview config={config} />

      <section className="mb-5">
        <h3 className="mb-2 text-[10px] font-bold uppercase tracking-wider text-[var(--color-text-muted)]">稀有度概率权重</h3>
        <div className="grid gap-2 md:grid-cols-2">
          {Object.entries(config.rarities).map(([rarityId, rarity]) => (
            <div key={rarityId} className="grid gap-2 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <span className="text-[10px] font-bold text-[var(--color-text-muted)]">{RARITY_LABELS[rarityId] ?? rarity.label}</span>
                <select
                  value={findRarityPresetId(rarity)}
                  onChange={(e) => applyRarityPreset(rarityId, e.target.value)}
                  className="max-w-[180px] rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 py-1 text-xs"
                  aria-label={`${RARITY_LABELS[rarityId] ?? rarity.label}模板`}
                >
                  {Object.entries(RARITY_PRESETS).map(([presetId, preset]) => <option key={presetId} value={presetId}>{preset.label}模板</option>)}
                  <option value={CUSTOM_VALUE}>自定义</option>
                </select>
              </div>
              <div className="grid grid-cols-2 gap-2">
                <TextField label="标签" value={rarity.label} onChange={(value) => updateRarity(rarityId, 'label', value)} />
                <NumberField label="权重" value={rarity.weight} step={0.1} onChange={(value) => updateRarity(rarityId, 'weight', value)} />
                <SelectField label="文字色" value={rarity.color} options={withCustomOption(RARITY_STYLE_OPTIONS.color, rarity.color)} onChange={(value) => updateRarity(rarityId, 'color', value)} />
                <SelectField label="背景" value={rarity.bg} options={withCustomOption(RARITY_STYLE_OPTIONS.bg, rarity.bg)} onChange={(value) => updateRarity(rarityId, 'bg', value)} />
                <SelectField label="边框" value={rarity.border} options={withCustomOption(RARITY_STYLE_OPTIONS.border, rarity.border)} onChange={(value) => updateRarity(rarityId, 'border', value)} />
                <SelectField label="光效" value={rarity.glow} options={withCustomOption(RARITY_STYLE_OPTIONS.glow, rarity.glow)} onChange={(value) => updateRarity(rarityId, 'glow', value)} />
              </div>
            </div>
          ))}
        </div>
      </section>

      <section className="mb-5">
        <div className="mb-2 flex items-center justify-between">
          <h3 className="text-[10px] font-bold uppercase tracking-wider text-[var(--color-text-muted)]">鱼饵规则</h3>
          <button type="button" onClick={addBait} className="flex items-center gap-1 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-2 py-1 text-xs font-bold text-[var(--color-text-primary)]">
            <Plus size={12} />
            加鱼饵
          </button>
        </div>
        <div className="grid gap-3 lg:grid-cols-2">
          {config.baits.map((bait, index) => (
            <div key={bait.id} className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-3">
              <div className="mb-2 grid gap-2 sm:grid-cols-[minmax(130px,1fr)_minmax(110px,0.85fr)_minmax(120px,1fr)_minmax(90px,0.7fr)_32px]">
                <SelectField
                  label="鱼饵模板"
                  value={findBaitPresetId(bait)}
                  options={[
                    ...BAIT_PRESETS.map((preset) => ({ value: preset.id, label: `${preset.name}（${preset.tier}）` })),
                    { value: CUSTOM_VALUE, label: '自定义' },
                  ]}
                  onChange={(value) => applyBaitPreset(index, value)}
                />
                <SelectField
                  label="鱼饵 ID"
                  value={bait.id}
                  options={withCustomOption(BAIT_PRESETS.map((preset) => ({ value: preset.id, label: preset.id })), bait.id)}
                  onChange={(value) => updateBait(index, 'id', value)}
                />
                <TextField label="名称" value={bait.name} onChange={(value) => updateBait(index, 'name', value)} />
                <NumberField label="铜钱" value={bait.cityGoldCost} min={0} onChange={(value) => updateBait(index, 'cityGoldCost', value)} />
                <button type="button" onClick={() => removeBait(index)} className="grid place-items-center rounded-lg border border-red-500/20 bg-red-500/8 text-red-600">
                  <Trash2 size={14} />
                </button>
              </div>
              <div className="mb-2 grid gap-2 sm:grid-cols-[minmax(120px,0.4fr)_minmax(220px,1fr)]">
                <SelectField label="阶级" value={bait.tier} options={withCustomOption(TIER_OPTIONS, bait.tier)} onChange={(value) => updateBait(index, 'tier', value)} />
                <TextField label="描述" value={bait.description} onChange={(value) => updateBait(index, 'description', value)} />
              </div>
              <div className="grid grid-cols-2 gap-2 sm:grid-cols-5">
                <NumberField label="稀有倍率" value={bait.rarityBoost} step={0.01} onChange={(value) => updateBait(index, 'rarityBoost', value)} />
                <NumberField label="咬钩率" value={bait.biteChance} min={0} max={1} step={0.01} onChange={(value) => updateBait(index, 'biteChance', value)} />
                <NumberField label="窗口ms" value={bait.biteWindowMs} min={0} onChange={(value) => updateBait(index, 'biteWindowMs', value)} />
                <NumberField label="命中起点" value={bait.sweetStart} min={0} max={100} onChange={(value) => updateBait(index, 'sweetStart', value)} />
                <NumberField label="命中终点" value={bait.sweetEnd} min={0} max={100} onChange={(value) => updateBait(index, 'sweetEnd', value)} />
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
            <div key={`${fish.name}-${index}`} className="grid gap-2 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-2 lg:grid-cols-[minmax(150px,1.1fr)_minmax(74px,0.45fr)_minmax(120px,0.9fr)_minmax(100px,0.65fr)_minmax(150px,1fr)_minmax(96px,0.6fr)_minmax(180px,1.2fr)_40px]">
              <SelectField
                label="鱼获模板"
                value={findFishPresetKey(fish)}
                options={[
                  ...FISH_PRESETS.map((preset, presetIndex) => ({
                    value: String(presetIndex),
                    label: `${preset.name} / ${RARITY_LABELS[preset.rarity] ?? preset.rarity}`,
                  })),
                  { value: CUSTOM_VALUE, label: '自定义' },
                ]}
                onChange={(value) => applyFishPreset(index, value)}
              />
              <SelectField label="外观" value={fish.emoji} options={withCustomOption(FISH_ICON_OPTIONS, fish.emoji)} onChange={(value) => updateFish(index, 'emoji', value)} />
              <TextField label="名称" value={fish.name} onChange={(value) => updateFish(index, 'name', value)} />
              <SelectField
                label="稀有度"
                value={fish.rarity}
                options={Object.keys(config.rarities).map((rarityId) => ({ value: rarityId, label: RARITY_LABELS[rarityId] ?? rarityId }))}
                onChange={(value) => updateFish(index, 'rarity', value)}
              />
              <label className="grid gap-1">
                <span className="text-[10px] text-[var(--color-text-muted)]">奖励兵种</span>
                <select value={fish.reward} onChange={(e) => updateFish(index, 'reward', e.target.value)} className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 py-1 text-sm">
                  {!fish.reward && <option value="">选择兵种</option>}
                  {unitOptions.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
                  {fish.reward && !unitOptions.some(([value]) => value === fish.reward) && <option value={fish.reward}>{fish.reward}（自定义）</option>}
                </select>
              </label>
              <NumberField label="数量" value={fish.rewardAmount} min={1} onChange={(value) => updateFish(index, 'rewardAmount', value)} />
              <TextField label="描述" value={fish.description} onChange={(value) => updateFish(index, 'description', value)} />
              <button type="button" onClick={() => removeFish(index)} className="grid h-[34px] place-items-center self-end rounded-lg border border-red-500/20 bg-red-500/8 text-red-600">
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

// 生成当前钓鱼配置的概率、鱼饵和鱼池预览，便于 GM 保存前校验。
function FishingConfigPreview({ config }: { config: FishingConfig }) {
  const totalWeight = Object.values(config.rarities).reduce((sum, rarity) => sum + Math.max(0, Number(rarity.weight) || 0), 0)
  const fishByRarity = Object.keys(config.rarities).map((rarityId) => {
    const fishes = config.fishPool.filter((fish) => fish.rarity === rarityId)
    const rewardTotal = fishes.reduce((sum, fish) => sum + (Number(fish.rewardAmount) || 0), 0)
    return { rarityId, fishes, rewardTotal }
  })

  return (
    <section className="mb-5 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-3">
      <div className="mb-3 flex items-center gap-2 text-sm font-bold text-[var(--color-text-primary)]">
        <BarChart3 size={15} className="text-[var(--color-accent)]" />
        配置预览
      </div>
      <div className="grid gap-3 xl:grid-cols-[minmax(260px,0.9fr)_minmax(320px,1.1fr)_minmax(300px,1fr)]">
        <div className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] p-3">
          <h4 className="mb-2 text-xs font-bold text-[var(--color-text-secondary)]">概率预览</h4>
          <div className="grid gap-2">
            {Object.entries(config.rarities).map(([rarityId, rarity]) => {
              const percent = totalWeight > 0 ? (Math.max(0, Number(rarity.weight) || 0) / totalWeight) * 100 : 0
              return (
                <div key={rarityId} className="grid gap-1">
                  <div className="flex items-center justify-between gap-2 text-xs">
                    <span className="font-bold text-[var(--color-text-primary)]">{rarity.label || RARITY_LABELS[rarityId] || rarityId}</span>
                    <span className="text-[var(--color-text-muted)]">{percent.toFixed(1)}%</span>
                  </div>
                  <div className="h-2 overflow-hidden rounded-full bg-[var(--color-border)]">
                    <div
                      className="h-full rounded-full"
                      style={{ width: `${Math.min(100, percent)}%`, backgroundColor: RARITY_PREVIEW_COLORS[rarityId] ?? '#64748b' }}
                    />
                  </div>
                </div>
              )
            })}
          </div>
        </div>

        <div className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] p-3">
          <h4 className="mb-2 text-xs font-bold text-[var(--color-text-secondary)]">鱼饵预览</h4>
          <div className="grid gap-2 sm:grid-cols-2">
            {config.baits.map((bait) => {
              const sweetStart = clampPercent(Number(bait.sweetStart) || 0)
              const sweetEnd = clampPercent(Number(bait.sweetEnd) || 0)
              const left = Math.min(sweetStart, sweetEnd)
              const width = Math.max(2, Math.abs(sweetEnd - sweetStart))
              return (
                <div key={bait.id} className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-2">
                  <div className="flex items-center justify-between gap-2 text-xs">
                    <span className="font-bold text-[var(--color-text-primary)]">{bait.name}</span>
                    <span className="text-[var(--color-text-muted)]">{bait.tier}</span>
                  </div>
                  <div className="mt-1 grid grid-cols-2 gap-x-2 gap-y-1 text-[11px] text-[var(--color-text-muted)]">
                    <span>铜钱 {bait.cityGoldCost}</span>
                    <span>咬钩 {(bait.biteChance * 100).toFixed(0)}%</span>
                    <span>倍率 x{bait.rarityBoost}</span>
                    <span>窗口 {bait.biteWindowMs}ms</span>
                  </div>
                  <div className="mt-2 h-2 overflow-hidden rounded-full bg-[var(--color-border)]">
                    <div className="h-full rounded-full bg-[var(--color-accent)]" style={{ marginLeft: `${left}%`, width: `${Math.min(100 - left, width)}%` }} />
                  </div>
                </div>
              )
            })}
          </div>
        </div>

        <div className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] p-3">
          <h4 className="mb-2 text-xs font-bold text-[var(--color-text-secondary)]">鱼池奖励预览</h4>
          <div className="grid gap-2">
            {fishByRarity.map(({ rarityId, fishes, rewardTotal }) => (
              <div key={rarityId} className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-2">
                <div className="flex items-center justify-between gap-2 text-xs">
                  <span className="font-bold text-[var(--color-text-primary)]">{config.rarities[rarityId]?.label ?? RARITY_LABELS[rarityId] ?? rarityId}</span>
                  <span className="text-[var(--color-text-muted)]">{fishes.length} 种 / {rewardTotal} 总量</span>
                </div>
                <p className="mt-1 truncate text-[11px] text-[var(--color-text-muted)]">
                  {fishes.slice(0, 5).map((fish) => fish.name).join('、') || '暂无鱼获'}
                </p>
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>
  )
}

// 给标准下拉补入当前自定义值，避免旧配置无法显示。
function withCustomOption(options: Array<{ value: string; label: string }>, currentValue: string) {
  if (!currentValue || options.some((option) => option.value === currentValue)) return options
  return [...options, { value: currentValue, label: `${currentValue}（自定义）` }]
}

// 判断当前稀有度配置是否完全匹配某个标准模板。
function findRarityPresetId(rarity: FishingRarityConfig) {
  return Object.entries(RARITY_PRESETS).find(([, preset]) => isSameRarityPreset(rarity, preset))?.[0] ?? CUSTOM_VALUE
}

// 判断当前鱼饵配置是否完全匹配某个标准模板。
function findBaitPresetId(bait: FishingBaitConfig) {
  return BAIT_PRESETS.find((preset) => isSameBaitPreset(bait, preset))?.id ?? CUSTOM_VALUE
}

// 判断当前鱼获配置是否完全匹配某个标准鱼获模板。
function findFishPresetKey(fish: FishingFishConfig) {
  const index = FISH_PRESETS.findIndex((preset) => isSameFishPreset(fish, preset))
  return index >= 0 ? String(index) : CUSTOM_VALUE
}

// 限制百分比预览值，避免异常配置撑破进度条。
function clampPercent(value: number) {
  return Math.min(100, Math.max(0, value))
}

// 对比稀有度模板字段。
function isSameRarityPreset(left: FishingRarityConfig, right: FishingRarityConfig) {
  return left.label === right.label
    && left.color === right.color
    && left.bg === right.bg
    && left.border === right.border
    && left.weight === right.weight
    && left.glow === right.glow
}

// 对比鱼饵模板字段。
function isSameBaitPreset(left: FishingBaitConfig, right: FishingBaitConfig) {
  return left.id === right.id
    && left.name === right.name
    && left.tier === right.tier
    && left.description === right.description
    && left.rarityBoost === right.rarityBoost
    && left.cityGoldCost === right.cityGoldCost
    && left.biteChance === right.biteChance
    && left.biteWindowMs === right.biteWindowMs
    && left.sweetStart === right.sweetStart
    && left.sweetEnd === right.sweetEnd
}

// 对比鱼获模板字段。
function isSameFishPreset(left: FishingFishConfig, right: FishingFishConfig) {
  return left.name === right.name
    && left.rarity === right.rarity
    && left.reward === right.reward
    && left.rewardAmount === right.rewardAmount
    && left.description === right.description
    && left.emoji === right.emoji
}

function SelectField({ label, value, options, onChange }: { label: string; value: string; options: Array<{ value: string; label: string }>; onChange: (value: string) => void }) {
  return (
    <label className="grid gap-1">
      <span className="text-[10px] text-[var(--color-text-muted)]">{label}</span>
      <select
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 py-1 text-sm"
      >
        {options.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
      </select>
    </label>
  )
}

function TextField({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  return (
    <label className="grid gap-1">
      <span className="text-[10px] text-[var(--color-text-muted)]">{label}</span>
      <input
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 py-1 text-sm"
      />
    </label>
  )
}

function NumberField({ label, value, min, max, step, onChange }: { label: string; value: number; min?: number; max?: number; step?: number; onChange: (value: number) => void }) {
  return (
    <label className="grid gap-1">
      <span className="text-[10px] text-[var(--color-text-muted)]">{label}</span>
      <input
        type="number"
        min={min}
        max={max}
        step={step}
        value={value}
        onChange={(e) => onChange(Number(e.target.value))}
        className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 py-1 text-sm"
      />
    </label>
  )
}
