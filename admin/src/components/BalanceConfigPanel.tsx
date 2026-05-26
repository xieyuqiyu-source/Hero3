import { useEffect, useState } from 'react'
import { Sliders, Save, ChevronDown, ChevronUp } from 'lucide-react'
import { adminApi } from '@/api/admin'
import type { BalanceConfig, BuildingConfig } from '@/types'

const BUILDING_LABELS: Record<string, string> = {
  wood_camp: '伐木场',
  stone_quarry: '采石场',
  iron_mine: '铁矿',
  farm: '农田',
  warehouse: '仓库',
}

const RES_LABELS: Record<string, string> = {
  wood: '木材',
  stone: '石料',
  iron: '铁矿',
  food: '粮食',
}

export default function BalanceConfigPanel() {
  const [balance, setBalance] = useState<BalanceConfig | null>(null)
  const [expandedBuilding, setExpandedBuilding] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    adminApi.getBalance()
      .then((data) => { if (!cancelled) setBalance(data) })
      .catch((err) => { if (!cancelled) setError(err instanceof Error ? err.message : '加载失败') })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [])

  const handleSave = async () => {
    if (!balance) return
    setSaving(true)
    setMessage(null)
    setError(null)
    try {
      const result = await adminApi.updateBalance(balance)
      setBalance(result)
      setMessage('建筑数值已保存')
    } catch (err) {
      setError(err instanceof Error ? err.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  const updateBaseProduction = (res: string, value: number) => {
    if (!balance) return
    setBalance({ ...balance, baseProduction: { ...balance.baseProduction, [res]: value } })
  }

  const updateProductionAtLevel = (buildingType: string, level: number, value: number) => {
    if (!balance) return
    const building = balance.buildings[buildingType]
    if (!building?.productionByLevel) return
    const next = [...building.productionByLevel]
    next[level] = value
    setBalance({
      ...balance,
      buildings: { ...balance.buildings, [buildingType]: { ...building, productionByLevel: next } },
    })
  }

  const updateCapacityAtLevel = (buildingType: string, level: number, value: number) => {
    if (!balance) return
    const building = balance.buildings[buildingType]
    if (!building?.capacityByLevel) return
    const next = [...building.capacityByLevel]
    next[level] = value
    setBalance({
      ...balance,
      buildings: { ...balance.buildings, [buildingType]: { ...building, capacityByLevel: next } },
    })
  }

  if (loading) return <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4"><p className="text-sm text-[var(--color-text-muted)]">加载中...</p></div>
  if (!balance) return <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4"><p className="text-sm text-red-600">{error ?? '加载失败'}</p></div>

  return (
    <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] shadow-[var(--shadow-panel)] p-4">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <Sliders size={16} className="text-[var(--color-accent)]" />
          <h2 className="text-base font-bold text-[var(--color-text-primary)]">建筑数值</h2>
          <span className="text-[11px] text-[var(--color-text-muted)]">{Object.keys(balance.buildings).length} 类</span>
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

      {/* Base Production */}
      <section className="mb-4">
        <h3 className="text-[10px] font-bold text-[var(--color-text-muted)] uppercase tracking-wider mb-2">基础产量 (每小时)</h3>
        <div className="grid grid-cols-4 gap-2">
          {Object.entries(balance.baseProduction).map(([res, val]) => (
            <label key={res} className="grid gap-1 px-2.5 py-2 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
              <span className="text-[10px] font-bold text-[var(--color-text-muted)]">{RES_LABELS[res] ?? res}</span>
              <input
                type="number"
                value={val}
                onChange={(e) => updateBaseProduction(res, parseInt(e.target.value) || 0)}
                className="h-7 px-2 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]"
              />
            </label>
          ))}
        </div>
      </section>

      {/* Overflow to CityGold */}
      <section className="mb-4">
        <h3 className="text-[10px] font-bold text-[var(--color-text-muted)] uppercase tracking-wider mb-2">溢出转城金</h3>
        <label className="flex items-center gap-3 px-2.5 py-2 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
          <span className="text-[10px] font-bold text-[var(--color-text-muted)] whitespace-nowrap">兑换比例</span>
          <input
            type="number"
            value={balance.overflowToCityGold ?? 200}
            onChange={(e) => setBalance({ ...balance, overflowToCityGold: parseInt(e.target.value) || 200 })}
            className="h-7 w-20 px-2 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]"
          />
          <span className="text-[10px] text-[var(--color-text-muted)]">资源 = 1 城金</span>
        </label>
      </section>

      {/* Gold Exchange Config */}
      <section className="mb-4">
        <h3 className="text-[10px] font-bold text-[var(--color-text-muted)] uppercase tracking-wider mb-2">金币兑换配置</h3>
        <div className="grid gap-2">
          <label className="flex items-center gap-3 px-2.5 py-2 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
            <span className="text-[10px] font-bold text-[var(--color-text-muted)] whitespace-nowrap">金币→城金</span>
            <span className="text-[10px] text-[var(--color-text-muted)]">1 金币 =</span>
            <input
              type="number"
              value={balance.exchangeRate ?? 10}
              onChange={(e) => setBalance({ ...balance, exchangeRate: parseInt(e.target.value) || 10 })}
              className="h-7 w-16 px-2 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]"
            />
            <span className="text-[10px] text-[var(--color-text-muted)]">城金</span>
          </label>
          <label className="flex items-center gap-3 px-2.5 py-2 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
            <span className="text-[10px] font-bold text-[var(--color-text-muted)] whitespace-nowrap">城金→金币</span>
            <input
              type="number"
              value={balance.reverseExchangeRate ?? 15}
              onChange={(e) => setBalance({ ...balance, reverseExchangeRate: parseInt(e.target.value) || 15 })}
              className="h-7 w-16 px-2 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]"
            />
            <span className="text-[10px] text-[var(--color-text-muted)]">城金 = 1 金币</span>
          </label>
          <label className="flex items-center gap-3 px-2.5 py-2 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
            <span className="text-[10px] font-bold text-[var(--color-text-muted)] whitespace-nowrap">兑换冷却</span>
            <input
              type="number"
              value={balance.exchangeCooldownSecs ?? 3600}
              onChange={(e) => setBalance({ ...balance, exchangeCooldownSecs: parseInt(e.target.value) || 0 })}
              className="h-7 w-20 px-2 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]"
            />
            <span className="text-[10px] text-[var(--color-text-muted)]">秒（0=无冷却）</span>
          </label>
          <label className="flex items-center gap-3 px-2.5 py-2 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
            <span className="text-[10px] font-bold text-[var(--color-text-muted)] whitespace-nowrap">加速折抵</span>
            <span className="text-[10px] text-[var(--color-text-muted)]">1 城金 =</span>
            <input
              type="number"
              value={balance.cityGoldPerSecond ?? 120}
              onChange={(e) => setBalance({ ...balance, cityGoldPerSecond: parseInt(e.target.value) || 120 })}
              className="h-7 w-16 px-2 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]"
            />
            <span className="text-[10px] text-[var(--color-text-muted)]">秒（征兵/建筑加速）</span>
          </label>
          <label className="flex items-center gap-3 px-2.5 py-2 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
            <span className="text-[10px] font-bold text-[var(--color-text-muted)] whitespace-nowrap">加成基价</span>
            <input
              type="number"
              value={balance.boostBaseCost ?? 30}
              onChange={(e) => setBalance({ ...balance, boostBaseCost: parseInt(e.target.value) || 30 })}
              className="h-7 w-16 px-2 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]"
            />
            <span className="text-[10px] text-[var(--color-text-muted)]">城金（产量加成基础价）</span>
          </label>
        </div>
      </section>

      {/* Buildings */}
      <section>
        <h3 className="text-[10px] font-bold text-[var(--color-text-muted)] uppercase tracking-wider mb-2">建筑列表</h3>
        <div className="grid gap-2">
          {Object.entries(balance.buildings).map(([type, building]) => {
            const isExpanded = expandedBuilding === type
            const maxLevel = (building.productionByLevel?.length ?? building.capacityByLevel?.length ?? 1) - 1
            const maxProduction = building.productionByLevel?.[maxLevel] ?? 0
            const maxCapacity = building.capacityByLevel?.[maxLevel] ?? 0

            return (
              <div key={type} className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] overflow-hidden">
                {/* Building Header */}
                <button
                  type="button"
                  onClick={() => setExpandedBuilding(isExpanded ? null : type)}
                  className="w-full flex items-center gap-3 px-3 py-2.5 cursor-pointer hover:bg-white/50 dark:hover:bg-white/5 transition-colors"
                >
                  <strong className="text-sm text-[var(--color-text-primary)] flex-1 text-left">
                    {BUILDING_LABELS[type] ?? building.name ?? type}
                  </strong>
                  <div className="flex items-center gap-3 text-[10px] text-[var(--color-text-muted)]">
                    {building.resourceType && <span className="px-1.5 py-0.5 rounded bg-emerald-500/10 text-emerald-700 font-bold">{building.resourceType}</span>}
                    {maxProduction > 0 && <span>满产 {maxProduction}/h</span>}
                    {maxCapacity > 0 && <span>满容 {maxCapacity.toLocaleString()}</span>}
                    <span>Lv.0-{maxLevel}</span>
                  </div>
                  {isExpanded ? <ChevronUp size={14} className="text-[var(--color-text-muted)]" /> : <ChevronDown size={14} className="text-[var(--color-text-muted)]" />}
                </button>

                {/* Expanded Level Table */}
                {isExpanded && (
                  <div className="px-3 pb-3 border-t border-[var(--color-border)]">
                    <BuildingLevelTable
                      building={building}
                      buildingType={type}
                      onProductionChange={updateProductionAtLevel}
                      onCapacityChange={updateCapacityAtLevel}
                    />
                  </div>
                )}
              </div>
            )
          })}
        </div>
      </section>

      {message && <p className="mt-3 text-xs font-bold text-emerald-600">{message}</p>}
      {error && <p className="mt-3 text-xs font-bold text-red-600">{error}</p>}
    </div>
  )
}

function BuildingLevelTable({
  building,
  buildingType,
  onProductionChange,
  onCapacityChange,
}: {
  building: BuildingConfig
  buildingType: string
  onProductionChange: (type: string, level: number, value: number) => void
  onCapacityChange: (type: string, level: number, value: number) => void
}) {
  const levels = building.productionByLevel?.length ?? building.capacityByLevel?.length ?? 0
  const hasProduction = (building.productionByLevel?.length ?? 0) > 0
  const hasCapacity = (building.capacityByLevel?.length ?? 0) > 0

  return (
    <div className="mt-2 overflow-x-auto">
      <table className="w-full text-[10px]">
        <thead>
          <tr className="text-[var(--color-text-muted)]">
            <th className="text-left py-1 pr-2 font-bold">Lv</th>
            {hasProduction && <th className="text-left py-1 pr-2 font-bold">产量/h</th>}
            {hasCapacity && <th className="text-left py-1 pr-2 font-bold">容量</th>}
            <th className="text-left py-1 pr-2 font-bold">升级时间(s)</th>
            <th className="text-left py-1 font-bold">升级消耗</th>
          </tr>
        </thead>
        <tbody>
          {Array.from({ length: levels }, (_, i) => {
            const upgradeCost = building.upgradeCostByLevel?.[String(i)] ?? building.upgradeCostByLevel?.[i as unknown as string]
            const upgradeSeconds = building.upgradeSecondsByLevel?.[String(i)] ?? building.upgradeSecondsByLevel?.[i as unknown as string]
            return (
              <tr key={i} className="border-t border-[var(--color-border)]/50">
                <td className="py-1.5 pr-2 font-bold text-[var(--color-text-primary)]">{i}</td>
                {hasProduction && (
                  <td className="py-1.5 pr-2">
                    <input
                      type="number"
                      value={building.productionByLevel?.[i] ?? 0}
                      onChange={(e) => onProductionChange(buildingType, i, parseInt(e.target.value) || 0)}
                      className="h-5 w-16 px-1 rounded text-[10px] border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]"
                    />
                  </td>
                )}
                {hasCapacity && (
                  <td className="py-1.5 pr-2">
                    <input
                      type="number"
                      value={building.capacityByLevel?.[i] ?? 0}
                      onChange={(e) => onCapacityChange(buildingType, i, parseInt(e.target.value) || 0)}
                      className="h-5 w-20 px-1 rounded text-[10px] border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]"
                    />
                  </td>
                )}
                <td className="py-1.5 pr-2 text-[var(--color-text-secondary)]">{upgradeSeconds ?? '-'}</td>
                <td className="py-1.5 text-[var(--color-text-muted)]">
                  {upgradeCost ? Object.entries(upgradeCost).map(([r, v]) => `${r}:${v}`).join(' ') : '-'}
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}
