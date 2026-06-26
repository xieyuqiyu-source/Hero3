import { useEffect, useState } from 'react'
import { Swords, Save, Plus, Trash2 } from 'lucide-react'
import { adminApi } from '@/api/admin'
import type { CombatConfig, CombatRuleConfig } from '@/types'

const SCENE_LABELS: Record<string, string> = {
  pve_attack: 'PVE 攻击',
  pve_plunder: 'PVE 掠夺',
  pvp_attack: 'PVP 攻击',
  pvp_plunder: 'PVP 掠夺',
}

const FACTION_LABELS: Record<string, string> = {
  wei: '魏',
  shu: '蜀',
  wu: '吴',
}

export default function CombatConfigPanel() {
  const [config, setConfig] = useState<CombatConfig | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    adminApi.getCombatConfig()
      .then((data) => { if (!cancelled) setConfig(data) })
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
      const result = await adminApi.updateCombatConfig(config)
      setConfig(result)
      setMessage('战斗配置已保存')
    } catch (err) {
      setError(err instanceof Error ? err.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  const updateRule = (ruleId: string, field: keyof CombatRuleConfig, value: string | number) => {
    if (!config) return
    setConfig({
      ...config,
      rules: {
        ...config.rules,
        [ruleId]: { ...config.rules[ruleId], [field]: value },
      },
    })
  }

  const addRule = () => {
    if (!config) return
    const id = `custom_rule_${Date.now()}`
    setConfig({
      ...config,
      rules: {
        ...config.rules,
        [id]: {
          id,
          name: '自定义规则',
          mode: 'attack',
          exponent: 1.422,
          equalResult: 'mutual_destruction',
          lossDistribution: 'proportional',
          defenseFormula: 'weighted',
        },
      },
    })
  }

  const removeRule = (ruleId: string) => {
    if (!config) return
    const activeScenes = Object.entries(config.activeCombatRules).filter(([, activeRuleId]) => activeRuleId === ruleId)
    if (activeScenes.length > 0) {
      setError('该规则仍被场景引用，先调整场景规则映射')
      return
    }
    const rules = { ...config.rules }
    delete rules[ruleId]
    setConfig({ ...config, rules })
  }

  const updateActiveRule = (scene: string, ruleId: string) => {
    if (!config) return
    setConfig({
      ...config,
      activeCombatRules: { ...config.activeCombatRules, [scene]: ruleId },
    })
  }

  const updateWall = (faction: string, base: number) => {
    if (!config) return
    setConfig({
      ...config,
      wallConfig: { ...config.wallConfig, [faction]: { base } },
    })
  }

  if (loading) return <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4"><p className="text-sm text-[var(--color-text-muted)]">加载中...</p></div>

  if (!config) return <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4"><p className="text-sm text-red-600">{error ?? '加载失败'}</p></div>

  const ruleIds = Object.keys(config.rules)

  return (
    <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] shadow-[var(--shadow-panel)] p-4">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <Swords size={16} className="text-[var(--color-accent)]" />
          <h2 className="text-base font-bold text-[var(--color-text-primary)]">战斗规则</h2>
        </div>
        <div className="flex items-center gap-2">
          <button type="button" onClick={addRule} className="flex items-center gap-1.5 rounded-xl border border-[var(--color-border)] px-3 py-1.5 text-xs font-bold text-[var(--color-accent)]">
            <Plus size={12} />
            新规则
          </button>
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

      {/* Active Rules Mapping */}
      <section className="mb-4">
        <h3 className="text-xs font-bold text-[var(--color-text-secondary)] uppercase tracking-wider mb-2">场景规则映射</h3>
        <div className="grid grid-cols-2 gap-2">
          {Object.entries(config.activeCombatRules).map(([scene, ruleId]) => (
            <div key={scene} className="px-3 py-2 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
              <label className="grid gap-1">
                <span className="text-[10px] font-bold text-[var(--color-text-muted)]">{SCENE_LABELS[scene] ?? scene}</span>
                <select
                  value={ruleId}
                  onChange={(e) => updateActiveRule(scene, e.target.value)}
                  className="h-7 px-2 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)] cursor-pointer"
                >
                  {ruleIds.map((id) => (
                    <option key={id} value={id}>{config.rules[id].name}</option>
                  ))}
                </select>
              </label>
            </div>
          ))}
        </div>
      </section>

      {/* Rules */}
      <section className="mb-4">
        <h3 className="text-xs font-bold text-[var(--color-text-secondary)] uppercase tracking-wider mb-2">规则详情</h3>
        <div className="grid gap-3">
          {Object.entries(config.rules).map(([ruleId, rule]) => (
            <div key={ruleId} className="p-3 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
              <div className="grid gap-2 mb-2 sm:grid-cols-[minmax(160px,1fr)_minmax(120px,0.45fr)_36px]">
                <input value={rule.name} onChange={(e) => updateRule(ruleId, 'name', e.target.value)} className="h-8 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 text-sm font-bold text-[var(--color-text-primary)]" />
                <select value={rule.mode} onChange={(e) => updateRule(ruleId, 'mode', e.target.value)} className="h-8 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 text-xs text-[var(--color-text-primary)]">
                  <option value="attack">attack</option>
                  <option value="plunder">plunder</option>
                </select>
                <button type="button" onClick={() => removeRule(ruleId)} className="grid h-8 place-items-center rounded-lg text-red-500 hover:bg-red-500/10">
                  <Trash2 size={14} />
                </button>
              </div>
              <div className="grid grid-cols-2 sm:grid-cols-4 gap-2">
                <label className="grid gap-1">
                  <span className="text-[10px] text-[var(--color-text-muted)]">损失指数</span>
                  <input
                    type="number"
                    step="0.001"
                    value={rule.exponent}
                    onChange={(e) => updateRule(ruleId, 'exponent', parseFloat(e.target.value) || 0)}
                    className="h-7 px-2 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]"
                  />
                </label>
                <label className="grid gap-1">
                  <span className="text-[10px] text-[var(--color-text-muted)]">平局结果</span>
                  <select
                    value={rule.equalResult}
                    onChange={(e) => updateRule(ruleId, 'equalResult', e.target.value)}
                    className="h-7 px-2 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)] cursor-pointer"
                  >
                    <option value="mutual_destruction">双方全灭</option>
                    <option value="defender_wins">防守方胜</option>
                    <option value="half_each">各损一半</option>
                  </select>
                </label>
                <label className="grid gap-1">
                  <span className="text-[10px] text-[var(--color-text-muted)]">损失分配</span>
                  <select
                    value={rule.lossDistribution}
                    onChange={(e) => updateRule(ruleId, 'lossDistribution', e.target.value)}
                    className="h-7 px-2 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)] cursor-pointer"
                  >
                    <option value="proportional">按比例</option>
                    <option value="weak_first">弱者优先</option>
                  </select>
                </label>
                <label className="grid gap-1">
                  <span className="text-[10px] text-[var(--color-text-muted)]">防御公式</span>
                  <select
                    value={rule.defenseFormula}
                    onChange={(e) => updateRule(ruleId, 'defenseFormula', e.target.value)}
                    className="h-7 px-2 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)] cursor-pointer"
                  >
                    <option value="weighted">加权</option>
                    <option value="max">取最大</option>
                    <option value="average">平均</option>
                  </select>
                </label>
              </div>
            </div>
          ))}
        </div>
      </section>

      {/* Wall Config */}
      <section>
        <h3 className="text-xs font-bold text-[var(--color-text-secondary)] uppercase tracking-wider mb-2">城墙系数</h3>
        <div className="grid grid-cols-3 gap-2">
          {Object.entries(config.wallConfig).map(([faction, entry]) => (
            <label key={faction} className="grid gap-1 px-3 py-2 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
              <span className="text-[10px] font-bold text-[var(--color-text-muted)]">{FACTION_LABELS[faction] ?? faction}</span>
              <input
                type="number"
                step="0.001"
                value={entry.base}
                onChange={(e) => updateWall(faction, parseFloat(e.target.value) || 1)}
                className="h-7 px-2 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]"
              />
            </label>
          ))}
        </div>
      </section>

      {message && <p className="mt-3 text-xs font-bold text-emerald-600">{message}</p>}
      {error && <p className="mt-3 text-xs font-bold text-red-600">{error}</p>}
    </div>
  )
}
