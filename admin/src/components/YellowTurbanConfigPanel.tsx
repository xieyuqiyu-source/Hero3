/* 本文件实现 GM 后台黄巾起义配置与检测面板。 */
import { useEffect, useState } from 'react'
import { AlertTriangle, RefreshCw, Save, Search } from 'lucide-react'
import { adminApi } from '@/api/admin'
import type { YellowTurbanConfig } from '@/types'

export default function YellowTurbanConfigPanel() {
  const [config, setConfig] = useState<YellowTurbanConfig | null>(null)
  const [jsonDraft, setJsonDraft] = useState('')
  const [message, setMessage] = useState('')
  const [saving, setSaving] = useState(false)
  const [checking, setChecking] = useState(false)

  const load = () => {
    adminApi.getYellowTurbanConfig()
      .then((data) => {
        setConfig(data)
        setJsonDraft(JSON.stringify(data, null, 2))
        setMessage('黄巾配置已刷新')
      })
      .catch((error) => setMessage(error instanceof Error ? error.message : '黄巾配置读取失败'))
  }

  useEffect(() => {
    load()
  }, [])

  const save = async () => {
    if (!config) return
    setSaving(true)
    try {
      const parsed = JSON.parse(jsonDraft) as YellowTurbanConfig
      const result = await adminApi.updateYellowTurbanConfig(parsed)
      setConfig(result)
      setJsonDraft(JSON.stringify(result, null, 2))
      setMessage('黄巾配置已保存')
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '黄巾配置保存失败')
    } finally {
      setSaving(false)
    }
  }

  const checkAll = async () => {
    setChecking(true)
    try {
      const result = await adminApi.checkYellowTurbanAll()
      const spawned = result.results.filter((item) => item.spawned).length
      setMessage(`全服检测完成，新增 ${spawned} 路黄巾来袭`)
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '全服检测失败')
    } finally {
      setChecking(false)
    }
  }

  const maxCapacity = config?.thousandTentCamp.capacityByLevel.at(-1) ?? 0
  const maxGoldCost = config?.thousandTentCamp.goldUpgradeCostByLevel.at(-1) ?? 0

  return (
    <section className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4 shadow-sm">
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="text-base font-bold text-[var(--color-text-primary)]">黄巾起义配置</h2>
          <p className="text-xs text-[var(--color-text-muted)]">千帐营、口粮检测、压力等级和黄巾行军倍率</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <button type="button" onClick={load} className="inline-flex items-center gap-1 rounded-lg border border-[var(--color-border)] px-3 py-2 text-xs font-bold">
            <RefreshCw size={14} />
            刷新
          </button>
          <button type="button" onClick={checkAll} disabled={checking} className="inline-flex items-center gap-1 rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-xs font-bold text-amber-700 disabled:opacity-60">
            <Search size={14} />
            {checking ? '检测中' : '全服检测'}
          </button>
          <button type="button" onClick={save} disabled={saving || !config} className="inline-flex items-center gap-1 rounded-lg bg-[var(--color-accent)] px-3 py-2 text-xs font-bold text-white disabled:opacity-60">
            <Save size={14} />
            {saving ? '保存中' : '保存配置'}
          </button>
        </div>
      </div>

      {message && (
        <div className="mb-3 flex items-center gap-2 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-xs text-[var(--color-text-secondary)]">
          <AlertTriangle size={14} />
          {message}
        </div>
      )}

      <div className="mb-3 grid gap-2 md:grid-cols-4">
        <Metric label="检测开关" value={config?.enabled ? '开启' : '关闭'} />
        <Metric label="检测间隔" value={`${config?.checkIntervalMinutes ?? 10} 分钟`} />
        <Metric label="满级承载" value={formatCompactNumber(maxCapacity)} />
        <Metric label="最高升级金币" value={maxGoldCost.toLocaleString()} />
      </div>

      <textarea
        value={jsonDraft}
        onChange={(event) => setJsonDraft(event.target.value)}
        spellCheck={false}
        className="h-[360px] w-full resize-y rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-3 font-mono text-xs text-[var(--color-text-primary)] outline-none focus:border-[var(--color-accent)]"
      />
    </section>
  )
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2">
      <div className="text-[10px] font-bold uppercase tracking-wider text-[var(--color-text-muted)]">{label}</div>
      <div className="mt-1 text-sm font-bold text-[var(--color-text-primary)]">{value}</div>
    </div>
  )
}

function formatCompactNumber(value: number) {
  if (value >= 100000000) return `${Number((value / 100000000).toFixed(1))}亿`
  if (value >= 10000) return `${Number((value / 10000).toFixed(1))}万`
  return value.toLocaleString()
}
