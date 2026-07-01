/* 本文件实现 GM 后台天机轮转配置面板。 */
import { useEffect, useMemo, useState } from 'react'
import { BarChart3, Plus, Save, Sparkles, Trash2 } from 'lucide-react'
import { adminApi } from '@/api/admin'
import type { SlotBonusMultiplierConfig, SlotConfig, SlotSymbolConfig } from '@/types'

const SYMBOL_TYPES = [
  { value: 'normal', label: '普通连线' },
  { value: 'wild', label: 'Wild 替代' },
  { value: 'scatter', label: 'Scatter 免费旋转' },
  { value: 'bonus', label: 'Bonus 宝匣' },
]

const RARITIES = [
  { value: 'common', label: '普通' },
  { value: 'rare', label: '稀有' },
  { value: 'epic', label: '史诗' },
  { value: 'legendary', label: '传说' },
]

// emptySymbol 创建一个新的天机轮转图案配置。
function emptySymbol(): SlotSymbolConfig {
  return {
    id: `symbol_${Date.now()}`,
    name: '新图案',
    rarity: 'common',
    type: 'normal',
    weight: 1,
    multiplier: 1,
  }
}

// emptyBonusMultiplier 创建宝匣倍率选项。
function emptyBonusMultiplier(): SlotBonusMultiplierConfig {
  return { multiplier: 5, weight: 1 }
}

// SlotConfigPanel 渲染天机轮转配置编辑器。
export default function SlotConfigPanel() {
  const [config, setConfig] = useState<SlotConfig | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [jsonDraft, setJsonDraft] = useState('')
  const [jsonOpen, setJsonOpen] = useState(false)

  useEffect(() => {
    let cancelled = false
    adminApi.getSlotConfig()
      .then((slotConfig) => {
        if (cancelled) return
        setConfig(slotConfig)
        setJsonDraft(JSON.stringify(slotConfig, null, 2))
      })
      .catch((err) => {
        if (!cancelled) setError(err instanceof Error ? err.message : '加载失败')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => { cancelled = true }
  }, [])

  const totalWeight = useMemo(() => {
    return config?.symbols.reduce((sum, symbol) => sum + Math.max(0, Number(symbol.weight) || 0), 0) ?? 0
  }, [config])

  // updateConfig 更新表单配置并同步高级 JSON 草稿。
  const updateConfig = (next: SlotConfig) => {
    setConfig(next)
    setJsonDraft(JSON.stringify(next, null, 2))
  }

  // handleSave 保存配置到后端并立即启用。
  const handleSave = async () => {
    if (!config) return
    setSaving(true)
    setMessage(null)
    setError(null)
    try {
      const result = await adminApi.updateSlotConfig(config)
      setConfig(result)
      setJsonDraft(JSON.stringify(result, null, 2))
      setMessage('天机轮转配置已保存')
    } catch (err) {
      setError(err instanceof Error ? err.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  // updateBase 更新天机轮转基础参数。
  const updateBase = (field: keyof Pick<SlotConfig, 'minLineBet' | 'lineCount' | 'maxFreeSpinsPerRound'>, value: number) => {
    if (!config) return
    updateConfig({ ...config, [field]: value })
  }

  // updateSymbol 更新单个图案配置。
  const updateSymbol = (index: number, updater: (symbol: SlotSymbolConfig) => SlotSymbolConfig) => {
    if (!config) return
    const symbols = [...config.symbols]
    symbols[index] = normalizeSymbolForType(updater(symbols[index]))
    updateConfig({ ...config, symbols })
  }

  // addSymbol 增加一个图案。
  const addSymbol = () => {
    if (!config) return
    updateConfig({ ...config, symbols: [...config.symbols, emptySymbol()] })
  }

  // removeSymbol 删除一个图案。
  const removeSymbol = (index: number) => {
    if (!config) return
    updateConfig({ ...config, symbols: config.symbols.filter((_, i) => i !== index) })
  }

  // updateBonusMultiplier 更新宝匣倍率配置。
  const updateBonusMultiplier = (symbolIndex: number, bonusIndex: number, field: keyof SlotBonusMultiplierConfig, value: number) => {
    updateSymbol(symbolIndex, (symbol) => {
      const bonusMultipliers = [...(symbol.bonusMultipliers ?? [])]
      bonusMultipliers[bonusIndex] = { ...bonusMultipliers[bonusIndex], [field]: value }
      return { ...symbol, bonusMultipliers }
    })
  }

  // addBonusMultiplier 给宝匣增加倍率档。
  const addBonusMultiplier = (symbolIndex: number) => {
    updateSymbol(symbolIndex, (symbol) => ({
      ...symbol,
      bonusMultipliers: [...(symbol.bonusMultipliers ?? []), emptyBonusMultiplier()],
    }))
  }

  // removeBonusMultiplier 删除宝匣倍率档。
  const removeBonusMultiplier = (symbolIndex: number, bonusIndex: number) => {
    updateSymbol(symbolIndex, (symbol) => ({
      ...symbol,
      bonusMultipliers: (symbol.bonusMultipliers ?? []).filter((_, i) => i !== bonusIndex),
    }))
  }

  // applyJsonDraft 将高级 JSON 写回表单。
  const applyJsonDraft = () => {
    try {
      const parsed = JSON.parse(jsonDraft) as SlotConfig
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
          <Sparkles size={16} className="text-[var(--color-accent)]" />
          <div>
            <h2 className="text-base font-bold text-[var(--color-text-primary)]">天机轮转配置</h2>
            <p className="text-xs text-[var(--color-text-muted)]">押注、图案权重、连线倍率和特殊图案规则</p>
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

      <section className="mb-5 grid gap-3 md:grid-cols-3">
        <NumberField label="每线最小押注" value={config.minLineBet} min={1} onChange={(value) => updateBase('minLineBet', value)} />
        <NumberField label="赔付线数量" value={config.lineCount} min={1} onChange={(value) => updateBase('lineCount', value)} />
        <NumberField label="单局免费旋转上限" value={config.maxFreeSpinsPerRound} min={1} onChange={(value) => updateBase('maxFreeSpinsPerRound', value)} />
      </section>

      <SlotWeightPreview symbols={config.symbols} totalWeight={totalWeight} />

      <section>
        <div className="mb-2 flex items-center justify-between">
          <h3 className="text-[10px] font-bold uppercase tracking-wider text-[var(--color-text-muted)]">图案与特殊规则</h3>
          <button type="button" onClick={addSymbol} className="flex items-center gap-1 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-2 py-1 text-xs font-bold text-[var(--color-text-primary)]">
            <Plus size={12} />
            加图案
          </button>
        </div>

        <div className="grid gap-3">
          {config.symbols.map((symbol, index) => {
            const chance = totalWeight > 0 ? (Math.max(0, Number(symbol.weight) || 0) / totalWeight) * 100 : 0
            return (
              <div key={`${symbol.id}-${index}`} className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-3">
                <div className="grid gap-2 lg:grid-cols-[minmax(130px,1fr)_minmax(120px,0.9fr)_minmax(115px,0.8fr)_minmax(115px,0.8fr)_minmax(90px,0.65fr)_minmax(90px,0.65fr)_40px]">
                  <TextField label="图案 ID" value={symbol.id} onChange={(value) => updateSymbol(index, (current) => ({ ...current, id: value }))} />
                  <TextField label="名称" value={symbol.name} onChange={(value) => updateSymbol(index, (current) => ({ ...current, name: value }))} />
                  <SelectField label="类型" value={symbol.type} options={withCustomOption(SYMBOL_TYPES, symbol.type)} onChange={(value) => updateSymbol(index, (current) => ({ ...current, type: value }))} />
                  <SelectField label="品质" value={symbol.rarity} options={withCustomOption(RARITIES, symbol.rarity)} onChange={(value) => updateSymbol(index, (current) => ({ ...current, rarity: value }))} />
                  <NumberField label="权重" value={symbol.weight} min={1} onChange={(value) => updateSymbol(index, (current) => ({ ...current, weight: value }))} />
                  <div className="grid gap-1">
                    <span className="text-[10px] text-[var(--color-text-muted)]">概率</span>
                    <div className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 py-1 text-sm text-[var(--color-text-secondary)]">
                      {chance.toFixed(1)}%
                    </div>
                  </div>
                  <button type="button" onClick={() => removeSymbol(index)} className="grid h-[34px] place-items-center self-end rounded-lg border border-red-500/20 bg-red-500/8 text-red-600">
                    <Trash2 size={14} />
                  </button>
                </div>

                {symbol.type === 'normal' || symbol.type === 'wild' ? (
                  <div className="mt-2 max-w-[180px]">
                    <NumberField label="三连倍率" value={symbol.multiplier ?? 1} min={1} onChange={(value) => updateSymbol(index, (current) => ({ ...current, multiplier: value }))} />
                  </div>
                ) : null}

                {symbol.type === 'scatter' ? (
                  <div className="mt-2 grid gap-2 sm:grid-cols-2">
                    <NumberField label="首次免费旋转" value={symbol.freeSpins ?? 1} min={1} onChange={(value) => updateSymbol(index, (current) => ({ ...current, freeSpins: value }))} />
                    <NumberField label="免费旋转再触发" value={symbol.retriggerFreeSpins ?? 1} min={1} onChange={(value) => updateSymbol(index, (current) => ({ ...current, retriggerFreeSpins: value }))} />
                  </div>
                ) : null}

                {symbol.type === 'bonus' ? (
                  <div className="mt-3 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] p-2">
                    <div className="mb-2 flex items-center justify-between gap-2">
                      <span className="text-xs font-bold text-[var(--color-text-secondary)]">宝匣倍率池</span>
                      <button type="button" onClick={() => addBonusMultiplier(index)} className="flex items-center gap-1 rounded-lg border border-[var(--color-border)] px-2 py-1 text-xs font-bold text-[var(--color-text-primary)]">
                        <Plus size={12} />
                        加倍率
                      </button>
                    </div>
                    <div className="grid gap-2 md:grid-cols-2 xl:grid-cols-4">
                      {(symbol.bonusMultipliers ?? []).map((bonus, bonusIndex) => (
                        <div key={`${symbol.id}-bonus-${bonusIndex}`} className="grid grid-cols-[1fr_1fr_32px] gap-2 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-2">
                          <NumberField label="倍率" value={bonus.multiplier} min={1} onChange={(value) => updateBonusMultiplier(index, bonusIndex, 'multiplier', value)} />
                          <NumberField label="权重" value={bonus.weight} min={1} onChange={(value) => updateBonusMultiplier(index, bonusIndex, 'weight', value)} />
                          <button type="button" onClick={() => removeBonusMultiplier(index, bonusIndex)} className="grid h-[34px] place-items-center self-end rounded-lg text-red-600 hover:bg-red-500/10">
                            <Trash2 size={14} />
                          </button>
                        </div>
                      ))}
                    </div>
                  </div>
                ) : null}
              </div>
            )
          })}
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

// SlotWeightPreview 展示图案权重占比和配置摘要。
function SlotWeightPreview({ symbols, totalWeight }: { symbols: SlotSymbolConfig[]; totalWeight: number }) {
  return (
    <section className="mb-5 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-3">
      <div className="mb-3 flex items-center gap-2 text-sm font-bold text-[var(--color-text-primary)]">
        <BarChart3 size={15} className="text-[var(--color-accent)]" />
        权重预览
      </div>
      <div className="grid gap-2 md:grid-cols-2 xl:grid-cols-3">
        {symbols.map((symbol) => {
          const percent = totalWeight > 0 ? (Math.max(0, Number(symbol.weight) || 0) / totalWeight) * 100 : 0
          return (
            <div key={symbol.id} className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] p-2">
              <div className="flex items-center justify-between gap-2 text-xs">
                <span className="font-bold text-[var(--color-text-primary)]">{symbol.name}</span>
                <span className="text-[var(--color-text-muted)]">{percent.toFixed(1)}%</span>
              </div>
              <div className="mt-1 h-2 overflow-hidden rounded-full bg-[var(--color-border)]">
                <div className="h-full rounded-full bg-[var(--color-accent)]" style={{ width: `${Math.min(100, percent)}%` }} />
              </div>
              <p className="mt-1 text-[11px] text-[var(--color-text-muted)]">
                {symbol.type} · 权重 {symbol.weight}{symbol.multiplier ? ` · x${symbol.multiplier}` : ''}
              </p>
            </div>
          )
        })}
      </div>
    </section>
  )
}

// normalizeSymbolForType 根据图案类型补齐或清理互斥字段。
function normalizeSymbolForType(symbol: SlotSymbolConfig): SlotSymbolConfig {
  if (symbol.type === 'normal' || symbol.type === 'wild') {
    return {
      ...symbol,
      multiplier: symbol.multiplier ?? 1,
      freeSpins: undefined,
      retriggerFreeSpins: undefined,
      bonusMultipliers: undefined,
    }
  }
  if (symbol.type === 'scatter') {
    return {
      ...symbol,
      multiplier: undefined,
      freeSpins: symbol.freeSpins ?? 1,
      retriggerFreeSpins: symbol.retriggerFreeSpins ?? 1,
      bonusMultipliers: undefined,
    }
  }
  if (symbol.type === 'bonus') {
    return {
      ...symbol,
      multiplier: undefined,
      freeSpins: undefined,
      retriggerFreeSpins: undefined,
      bonusMultipliers: symbol.bonusMultipliers?.length ? symbol.bonusMultipliers : [emptyBonusMultiplier()],
    }
  }
  return symbol
}

// withCustomOption 给标准下拉补入当前自定义值。
function withCustomOption(options: Array<{ value: string; label: string }>, currentValue: string) {
  if (!currentValue || options.some((option) => option.value === currentValue)) return options
  return [...options, { value: currentValue, label: `${currentValue}（自定义）` }]
}

// SelectField 渲染选择输入。
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

// TextField 渲染文本输入。
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

// NumberField 渲染数字输入。
function NumberField({ label, value, min, step = 1, onChange }: { label: string; value: number; min?: number; step?: number; onChange: (value: number) => void }) {
  return (
    <label className="grid gap-1">
      <span className="text-[10px] text-[var(--color-text-muted)]">{label}</span>
      <input
        type="number"
        value={Number.isFinite(value) ? value : 0}
        min={min}
        step={step}
        onChange={(e) => onChange(Number(e.target.value))}
        className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 py-1 text-sm"
      />
    </label>
  )
}
