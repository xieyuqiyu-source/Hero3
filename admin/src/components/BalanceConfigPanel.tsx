/* 本文件实现 GM 后台建筑与经济数值配置面板。 */
import { useEffect, useState } from 'react'
import { Sliders, Save, ChevronDown, ChevronUp, Plus, Trash2 } from 'lucide-react'
import { adminApi } from '@/api/admin'
import type { BalanceConfig, BuildingConfig, ModifierConfig } from '@/types'

const BUILDING_LABELS: Record<string, string> = {
  wood_camp: '伐木场',
  stone_quarry: '采石场',
  iron_mine: '铁矿',
  farm: '农田',
  warehouse: '仓库',
  infantry_camp: '步兵营',
  cavalry_camp: '骑兵营',
  siege_camp: '攻城武器营',
  special_camp: '特殊建筑营',
  weapon_bureau: '兵器司',
  armor_bureau: '防具司',
  construction_bureau: '建造司',
  administration: '政务厅',
  relay_station: '驿站',
  city_wall: '城墙',
  beacon_tower: '烽火台',
}

const RES_LABELS: Record<string, string> = {
  wood: '木材',
  stone: '石料',
  iron: '铁矿',
  food: '粮食',
}

const STAT_LABELS: Record<string, string> = {
  productionBonus: '全资源产量',
  woodProductionBonus: '木材产量',
  stoneProductionBonus: '石料产量',
  ironProductionBonus: '铁矿产量',
  foodProductionBonus: '粮食产量',
  capacityBonus: '仓库容量',
  attackBonus: '攻击',
  defenseBonus: '防御',
  infantryDefenseBonus: '步防',
  cavalryDefenseBonus: '骑防',
  infantryRecruitSpeedBonus: '步兵征兵',
  cavalryRecruitSpeedBonus: '骑兵征兵',
  siegeRecruitSpeedBonus: '攻城征兵',
  specialRecruitSpeedBonus: '特殊征兵',
  infantryRecruitCostReduction: '步兵减耗',
  cavalryRecruitCostReduction: '骑兵减耗',
  siegeRecruitCostReduction: '攻城减耗',
  specialRecruitCostReduction: '特殊减耗',
  buildSpeedBonus: '建造速度',
  recruitSpeedBonus: '征兵速度',
  marchSpeedBonus: '行军速度',
  exchangeRateBonus: '兑换比例',
}

const MODIFIER_MODES = ['flat', 'percentAdd', 'percentMultiply']

const levelRecordValue = <T,>(record: Record<string, T> | undefined, level: number) => {
  return record?.[String(level)]
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

  const updateBoostFactor = (field: 'boostMultiplierFactor' | 'boostDurationFactor', key: string, value: number) => {
    if (!balance) return
    setBalance({ ...balance, [field]: { ...balance[field], [key]: value } })
  }

  const updateUpgradeSeconds = (buildingType: string, level: number, value: number) => {
    if (!balance) return
    const building = balance.buildings[buildingType]
    setBalance({
      ...balance,
      buildings: {
        ...balance.buildings,
        [buildingType]: {
          ...building,
          upgradeSecondsByLevel: { ...building.upgradeSecondsByLevel, [String(level)]: value },
        },
      },
    })
  }

  const updateUpgradeCost = (buildingType: string, level: number, resource: string, value: number) => {
    if (!balance) return
    const building = balance.buildings[buildingType]
    const currentCost = levelRecordValue(building.upgradeCostByLevel, level) ?? {}
    setBalance({
      ...balance,
      buildings: {
        ...balance.buildings,
        [buildingType]: {
          ...building,
          upgradeCostByLevel: {
            ...building.upgradeCostByLevel,
            [String(level)]: { ...currentCost, [resource]: value },
          },
        },
      },
    })
  }

  const updateGoldUpgradeCost = (buildingType: string, level: number, value: number) => {
    if (!balance) return
    const building = balance.buildings[buildingType]
    setBalance({
      ...balance,
      buildings: {
        ...balance.buildings,
        [buildingType]: {
          ...building,
          goldUpgradeCostByLevel: {
            ...building.goldUpgradeCostByLevel,
            [String(level)]: value,
          },
        },
      },
    })
  }

  const updateModifier = (buildingType: string, level: number, index: number, next: ModifierConfig) => {
    if (!balance) return
    const building = balance.buildings[buildingType]
    const modifiers = [...(levelRecordValue(building.modifiersByLevel, level) ?? [])]
    modifiers[index] = next
    setBalance({
      ...balance,
      buildings: {
        ...balance.buildings,
        [buildingType]: {
          ...building,
          modifiersByLevel: { ...building.modifiersByLevel, [String(level)]: modifiers },
        },
      },
    })
  }

  const addModifier = (buildingType: string, level: number) => {
    if (!balance) return
    const building = balance.buildings[buildingType]
    const modifiers = [...(levelRecordValue(building.modifiersByLevel, level) ?? [])]
    modifiers.push({ key: 'attackBonus', value: 0, mode: 'percentAdd' })
    setBalance({
      ...balance,
      buildings: {
        ...balance.buildings,
        [buildingType]: {
          ...building,
          modifiersByLevel: { ...building.modifiersByLevel, [String(level)]: modifiers },
        },
      },
    })
  }

  const removeModifier = (buildingType: string, level: number, index: number) => {
    if (!balance) return
    const building = balance.buildings[buildingType]
    const modifiers = (levelRecordValue(building.modifiersByLevel, level) ?? []).filter((_, idx) => idx !== index)
    setBalance({
      ...balance,
      buildings: {
        ...balance.buildings,
        [buildingType]: {
          ...building,
          modifiersByLevel: { ...building.modifiersByLevel, [String(level)]: modifiers },
        },
      },
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
        <div className="grid gap-2 sm:grid-cols-2 xl:grid-cols-4">
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
        <label className="grid gap-2 px-2.5 py-2 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] sm:grid-cols-[auto_minmax(80px,1fr)_auto] sm:items-center">
          <span className="text-[10px] font-bold text-[var(--color-text-muted)] whitespace-nowrap">兑换比例</span>
          <input
            type="number"
            value={balance.overflowToCityGold ?? 200}
            onChange={(e) => setBalance({ ...balance, overflowToCityGold: parseInt(e.target.value) || 200 })}
            className="h-7 px-2 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]"
          />
          <span className="text-[10px] text-[var(--color-text-muted)]">资源 = 1 城金</span>
        </label>
      </section>

      {/* Gold Exchange Config */}
      <section className="mb-4">
        <h3 className="text-[10px] font-bold text-[var(--color-text-muted)] uppercase tracking-wider mb-2">金币兑换配置</h3>
        <div className="grid gap-2">
          <label className="grid gap-2 px-2.5 py-2 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] sm:grid-cols-[auto_auto_minmax(70px,1fr)_auto] sm:items-center">
            <span className="text-[10px] font-bold text-[var(--color-text-muted)] whitespace-nowrap">金币→城金</span>
            <span className="text-[10px] text-[var(--color-text-muted)]">1 金币 =</span>
            <input
              type="number"
              value={balance.exchangeRate ?? 10}
              onChange={(e) => setBalance({ ...balance, exchangeRate: parseInt(e.target.value) || 10 })}
              className="h-7 px-2 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]"
            />
            <span className="text-[10px] text-[var(--color-text-muted)]">城金</span>
          </label>
          <label className="grid gap-2 px-2.5 py-2 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] sm:grid-cols-[auto_minmax(70px,1fr)_auto] sm:items-center">
            <span className="text-[10px] font-bold text-[var(--color-text-muted)] whitespace-nowrap">城金→金币</span>
            <input
              type="number"
              value={balance.reverseExchangeRate ?? 15}
              onChange={(e) => setBalance({ ...balance, reverseExchangeRate: parseInt(e.target.value) || 15 })}
              className="h-7 px-2 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]"
            />
            <span className="text-[10px] text-[var(--color-text-muted)]">城金 = 1 金币</span>
          </label>
          <label className="grid gap-2 px-2.5 py-2 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] sm:grid-cols-[auto_minmax(80px,1fr)_auto] sm:items-center">
            <span className="text-[10px] font-bold text-[var(--color-text-muted)] whitespace-nowrap">兑换冷却</span>
            <input
              type="number"
              value={balance.exchangeCooldownSecs ?? 3600}
              onChange={(e) => setBalance({ ...balance, exchangeCooldownSecs: parseInt(e.target.value) || 0 })}
              className="h-7 px-2 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]"
            />
            <span className="text-[10px] text-[var(--color-text-muted)]">秒（0=无冷却）</span>
          </label>
          <label className="grid gap-2 px-2.5 py-2 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] sm:grid-cols-[auto_auto_minmax(70px,1fr)_auto] sm:items-center">
            <span className="text-[10px] font-bold text-[var(--color-text-muted)] whitespace-nowrap">加速折抵</span>
            <span className="text-[10px] text-[var(--color-text-muted)]">1 城金 =</span>
            <input
              type="number"
              value={balance.cityGoldPerSecond ?? 120}
              onChange={(e) => setBalance({ ...balance, cityGoldPerSecond: parseInt(e.target.value) || 120 })}
              className="h-7 px-2 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]"
            />
            <span className="text-[10px] text-[var(--color-text-muted)]">秒（征兵/建筑加速）</span>
          </label>
          <label className="grid gap-2 px-2.5 py-2 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] sm:grid-cols-[auto_minmax(70px,1fr)_auto] sm:items-center">
            <span className="text-[10px] font-bold text-[var(--color-text-muted)] whitespace-nowrap">加成基价</span>
            <input
              type="number"
              value={balance.boostBaseCost ?? 30}
              onChange={(e) => setBalance({ ...balance, boostBaseCost: parseInt(e.target.value) || 30 })}
              className="h-7 px-2 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]"
            />
            <span className="text-[10px] text-[var(--color-text-muted)]">城金（产量加成基础价）</span>
          </label>
          <div className="grid gap-2 sm:grid-cols-2">
            <FactorEditor
              title="产量加成倍率因子"
              factors={balance.boostMultiplierFactor}
              suffix="倍"
              onChange={(key, value) => updateBoostFactor('boostMultiplierFactor', key, value)}
            />
            <FactorEditor
              title="产量加成时长因子"
              factors={balance.boostDurationFactor}
              suffix="小时"
              onChange={(key, value) => updateBoostFactor('boostDurationFactor', key, value)}
            />
          </div>
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
                      onUpgradeSecondsChange={updateUpgradeSeconds}
                      onUpgradeCostChange={updateUpgradeCost}
                      onGoldUpgradeCostChange={updateGoldUpgradeCost}
                      onModifierChange={updateModifier}
                      onModifierAdd={addModifier}
                      onModifierRemove={removeModifier}
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
  onUpgradeSecondsChange,
  onUpgradeCostChange,
  onGoldUpgradeCostChange,
  onModifierChange,
  onModifierAdd,
  onModifierRemove,
}: {
  building: BuildingConfig
  buildingType: string
  onProductionChange: (type: string, level: number, value: number) => void
  onCapacityChange: (type: string, level: number, value: number) => void
  onUpgradeSecondsChange: (type: string, level: number, value: number) => void
  onUpgradeCostChange: (type: string, level: number, resource: string, value: number) => void
  onGoldUpgradeCostChange: (type: string, level: number, value: number) => void
  onModifierChange: (type: string, level: number, index: number, modifier: ModifierConfig) => void
  onModifierAdd: (type: string, level: number) => void
  onModifierRemove: (type: string, level: number, index: number) => void
}) {
  const levelKeys = [
    ...(building.productionByLevel?.map((_, index) => index) ?? []),
    ...(building.capacityByLevel?.map((_, index) => index) ?? []),
    ...Object.keys(building.upgradeCostByLevel ?? {}).map(Number),
    ...Object.keys(building.goldUpgradeCostByLevel ?? {}).map(Number),
    ...Object.keys(building.upgradeSecondsByLevel ?? {}).map(Number),
    ...Object.keys(building.modifiersByLevel ?? {}).map(Number),
  ].filter((level) => Number.isFinite(level))
  const levels = levelKeys.length > 0 ? Math.max(...levelKeys) + 1 : 0
  const hasProduction = (building.productionByLevel?.length ?? 0) > 0
  const hasCapacity = (building.capacityByLevel?.length ?? 0) > 0
  const hasResourceCost = Object.keys(building.upgradeCostByLevel ?? {}).length > 0
  const hasGoldCost = Object.keys(building.goldUpgradeCostByLevel ?? {}).length > 0
  const hasModifiers = Object.keys(building.modifiersByLevel ?? {}).length > 0

  return (
    <div className="mt-2 overflow-x-auto">
      <table className="w-full text-[10px]">
        <thead>
          <tr className="text-[var(--color-text-muted)]">
            <th className="text-left py-1 pr-2 font-bold">Lv</th>
            {hasProduction && <th className="text-left py-1 pr-2 font-bold">产量/h</th>}
            {hasCapacity && <th className="text-left py-1 pr-2 font-bold">容量</th>}
            <th className="text-left py-1 pr-2 font-bold">升级时间(s)</th>
            {hasResourceCost && <th className="text-left py-1 font-bold">资源消耗</th>}
            {hasGoldCost && <th className="text-left py-1 font-bold">账号金币</th>}
            {hasModifiers && <th className="text-left py-1 pl-2 font-bold">属性加成</th>}
          </tr>
        </thead>
        <tbody>
          {Array.from({ length: levels }, (_, i) => {
            const upgradeCost = levelRecordValue(building.upgradeCostByLevel, i)
            const goldUpgradeCost = levelRecordValue(building.goldUpgradeCostByLevel, i)
            const upgradeSeconds = levelRecordValue(building.upgradeSecondsByLevel, i)
            const modifiers = levelRecordValue(building.modifiersByLevel, i) ?? []
            return (
              <tr key={i} className="border-t border-[var(--color-border)]/50">
                <td className="py-1.5 pr-2 font-bold text-[var(--color-text-primary)]">{i}</td>
                {hasProduction && (
                  <td className="py-1.5 pr-2">
                    <input
                      type="number"
                      value={building.productionByLevel?.[i] ?? 0}
                      onChange={(e) => onProductionChange(buildingType, i, parseInt(e.target.value) || 0)}
                      className="h-6 min-w-[64px] px-1 rounded text-[10px] border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]"
                    />
                  </td>
                )}
                {hasCapacity && (
                  <td className="py-1.5 pr-2">
                    <input
                      type="number"
                      value={building.capacityByLevel?.[i] ?? 0}
                      onChange={(e) => onCapacityChange(buildingType, i, parseInt(e.target.value) || 0)}
                      className="h-6 min-w-[72px] px-1 rounded text-[10px] border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]"
                    />
                  </td>
                )}
                <td className="py-1.5 pr-2">
                  <input
                    type="number"
                    value={upgradeSeconds ?? 0}
                    onChange={(e) => onUpgradeSecondsChange(buildingType, i, parseInt(e.target.value) || 0)}
                    className="h-6 min-w-[72px] px-1 rounded text-[10px] border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]"
                  />
                </td>
                {hasResourceCost && (
                  <td className="py-1.5">
                    <div className="grid min-w-[220px] grid-cols-2 gap-1">
                      {Object.keys(RES_LABELS).map((res) => (
                        <label key={res} className="flex items-center gap-1">
                          <span className="shrink-0 text-[9px] text-[var(--color-text-muted)]">{RES_LABELS[res]}</span>
                          <input
                            type="number"
                            value={upgradeCost?.[res] ?? 0}
                            onChange={(e) => onUpgradeCostChange(buildingType, i, res, parseInt(e.target.value) || 0)}
                            className="h-6 min-w-[64px] px-1 rounded text-[10px] border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]"
                          />
                        </label>
                      ))}
                    </div>
                  </td>
                )}
                {hasGoldCost && (
                  <td className="py-1.5">
                    <label className="flex min-w-[96px] items-center gap-1">
                      <span className="shrink-0 text-[9px] text-amber-600">金币</span>
                      <input
                        type="number"
                        value={goldUpgradeCost ?? 0}
                        onChange={(e) => onGoldUpgradeCostChange(buildingType, i, parseInt(e.target.value) || 0)}
                        className="h-6 min-w-[72px] px-1 rounded text-[10px] border border-amber-500/30 bg-amber-500/5 text-[var(--color-text-primary)]"
                      />
                    </label>
                  </td>
                )}
                {hasModifiers && (
                  <td className="py-1.5 pl-2">
                    <div className="grid gap-1 min-w-[360px]">
                      {modifiers.map((modifier, index) => (
                        <div key={`${modifier.key}-${index}`} className="grid grid-cols-[minmax(120px,1fr)_minmax(72px,0.55fr)_minmax(112px,0.7fr)_24px] gap-1">
                          <select
                            value={modifier.key}
                            onChange={(e) => onModifierChange(buildingType, i, index, { ...modifier, key: e.target.value })}
                            className="h-6 rounded text-[10px] border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]"
                          >
                            {Object.entries(STAT_LABELS).map(([key, label]) => <option key={key} value={key}>{label}</option>)}
                            {!STAT_LABELS[modifier.key] && <option value={modifier.key}>{modifier.key}</option>}
                          </select>
                          <input
                            type="number"
                            step="0.01"
                            value={modifier.value}
                            onChange={(e) => onModifierChange(buildingType, i, index, { ...modifier, value: parseFloat(e.target.value) || 0 })}
                            className="h-6 rounded text-[10px] border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]"
                          />
                          <select
                            value={modifier.mode}
                            onChange={(e) => onModifierChange(buildingType, i, index, { ...modifier, mode: e.target.value })}
                            className="h-6 rounded text-[10px] border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)]"
                          >
                            {MODIFIER_MODES.map((mode) => <option key={mode} value={mode}>{mode}</option>)}
                          </select>
                          <button type="button" onClick={() => onModifierRemove(buildingType, i, index)} className="grid h-6 place-items-center rounded text-red-500 hover:bg-red-500/10">
                            <Trash2 size={11} />
                          </button>
                        </div>
                      ))}
                      <button type="button" onClick={() => onModifierAdd(buildingType, i)} className="inline-flex h-6 w-fit items-center gap-1 rounded border border-[var(--color-border)] px-2 text-[10px] font-bold text-[var(--color-accent)]">
                        <Plus size={10} />
                        加成
                      </button>
                    </div>
                  </td>
                )}
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

function FactorEditor({
  title,
  factors,
  suffix,
  onChange,
}: {
  title: string
  factors: Record<string, number>
  suffix: string
  onChange: (key: string, value: number) => void
}) {
  return (
    <div className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-2.5 py-2">
      <h4 className="mb-2 text-[10px] font-bold text-[var(--color-text-muted)]">{title}</h4>
      <div className="grid gap-1.5 sm:grid-cols-2">
        {Object.entries(factors ?? {}).map(([key, value]) => (
          <label key={key} className="grid grid-cols-[auto_minmax(72px,1fr)] items-center gap-1">
            <span className="text-[10px] text-[var(--color-text-muted)]">{key}{suffix}</span>
            <input
              type="number"
              value={value}
              onChange={(e) => onChange(key, parseInt(e.target.value) || 0)}
              className="h-7 min-w-0 flex-1 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 text-xs text-[var(--color-text-primary)]"
            />
          </label>
        ))}
      </div>
    </div>
  )
}
