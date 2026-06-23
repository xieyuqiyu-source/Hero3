import { type FC, useMemo, useState } from 'react'
import { Boxes, Package, Sparkles, Swords, UserRound } from 'lucide-react'
import { gameApi } from '@/api/game'
import { toast } from '@/components/ui'
import { useConfigStore } from '@/store/configStore'
import { useGameStore } from '@/store/gameStore'
import type { ItemDefinition, ItemEffect, ItemStack } from '@/types/game'
import { getTraitMeta, formatParamLabel, formatParamValue } from '@/utils/traits'

const ATTRIBUTE_LABELS: Record<string, string> = {
  productionBonus: '资源产量',
  woodProductionBonus: '木材产量',
  stoneProductionBonus: '石料产量',
  ironProductionBonus: '铁矿产量',
  foodProductionBonus: '粮食产量',
  capacityBonus: '仓库容量',
  attackBonus: '部队攻击',
  defenseBonus: '部队防御',
  infantryDefenseBonus: '步兵防御',
  cavalryDefenseBonus: '骑兵防御',
  buildSpeedBonus: '建造速度',
  recruitSpeedBonus: '征兵速度',
  marchSpeedBonus: '行军速度',
  exchangeRateBonus: '兑换收益',
}

const ATTRIBUTE_ORDER = [
  'attackBonus',
  'defenseBonus',
  'infantryDefenseBonus',
  'cavalryDefenseBonus',
  'productionBonus',
  'woodProductionBonus',
  'stoneProductionBonus',
  'ironProductionBonus',
  'foodProductionBonus',
  'capacityBonus',
  'buildSpeedBonus',
  'recruitSpeedBonus',
  'marchSpeedBonus',
  'exchangeRateBonus',
]

const formatAttributeValue = (value: number) => `${value >= 0 ? '+' : ''}${Math.round(value * 100)}%`
const formatBreakdownTitle = (label: string, total: number, items: Array<{ source: string; value: number }>) => {
  if (items.length === 0) return `${label} ${formatAttributeValue(total)}`
  return [
    `${label} ${formatAttributeValue(total)}`,
    ...items.map((item) => `${item.source} ${formatAttributeValue(item.value)}`),
  ].join('\n')
}
const STAT_LABELS: Record<string, string> = {
  force: '武力',
  intelligence: '智谋',
  politics: '内政',
  command: '统率',
}
const STAT_COLORS: Record<string, string> = {
  force: 'text-amber-600',
  intelligence: 'text-blue-500',
  politics: 'text-green-500',
  command: 'text-purple-500',
}
const STAT_ORDER = ['force', 'intelligence', 'politics', 'command']
const RARITY_CLASS: Record<string, string> = {
  common: 'text-slate-500 bg-slate-500/10 border-slate-500/20',
  rare: 'text-blue-500 bg-blue-500/10 border-blue-500/20',
  epic: 'text-fuchsia-500 bg-fuchsia-500/10 border-fuchsia-500/20',
  legendary: 'text-amber-500 bg-amber-500/10 border-amber-500/20',
}
const RARITY_LABEL: Record<string, string> = {
  common: '普通',
  rare: '稀有',
  epic: '史诗',
  legendary: '传说',
}

const effectIcon = (effects: ItemEffect[]) => {
  if (effects.some((effect) => effect.type === 'general_exp')) return UserRound
  if (effects.some((effect) => effect.type === 'unit_by_faction')) return Swords
  return Boxes
}

const GeneralPanel: FC = () => {
  const state = useGameStore((s) => s.state)
  const general = useGameStore((s) => s.state?.general)
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const setState = useGameStore((s) => s.setState)
  const allocateGeneralStat = useGameStore((s) => s.allocateGeneralStat)
  const itemsConfig = useConfigStore((s) => s.items)
  const [allocatingStat, setAllocatingStat] = useState<string | null>(null)
  const [usingItemId, setUsingItemId] = useState<string | null>(null)
  const inventoryRows = useMemo(() => {
    const inventory = state?.inventory ?? {}
    return Object.values(inventory)
      .filter((stack) => stack.amount > 0)
      .map((stack) => ({
        stack,
        item: itemsConfig?.[stack.itemId],
      }))
      .sort((a, b) => {
        const nameA = a.item?.name ?? a.stack.itemId
        const nameB = b.item?.name ?? b.stack.itemId
        return nameA.localeCompare(nameB, 'zh-Hans-CN')
      })
  }, [itemsConfig, state?.inventory])

  if (!general) {
    return (
      <div className="flex items-center justify-center py-16">
        <span className="text-sm text-[var(--color-text-muted)]">暂无将领，请重新创建存档选择将领</span>
      </div>
    )
  }

  const traits = general.traits ?? []
  const attributes = general.attributes ?? general.buffs ?? {}
  const attributeBreakdown = general.attributeBreakdown ?? {}
  const attributeEntries = Object.entries(attributes)
    .filter(([, value]) => value !== 0)
    .sort(([a], [b]) => {
      const ai = ATTRIBUTE_ORDER.indexOf(a)
      const bi = ATTRIBUTE_ORDER.indexOf(b)
      return (ai === -1 ? 999 : ai) - (bi === -1 ? 999 : bi)
    })
  const nextLevelExp = general.nextLevelExp ?? 0
  const expToNext = nextLevelExp > 0 ? Math.max(nextLevelExp - general.exp, 0) : 0
  const expProgress = nextLevelExp > 0 ? Math.min(100, Math.round((general.exp / nextLevelExp) * 100)) : 100
  const statEntries = STAT_ORDER.map((key) => [key, general.stats?.[key] ?? 0] as const)
  const availableStatPoints = general.availableStatPoints ?? 0

  const handleAllocateStat = async (statKey: string) => {
    if (availableStatPoints <= 0 || allocatingStat) return
    setAllocatingStat(statKey)
    try {
      await allocateGeneralStat(statKey)
    } finally {
      setAllocatingStat(null)
    }
  }

  const handleUseItem = async (stack: ItemStack, item?: ItemDefinition) => {
    if (!activePlayerId || !item?.usable || usingItemId) return
    setUsingItemId(stack.itemId)
    try {
      const result = await gameApi.useItem(activePlayerId, stack.itemId, 1)
      setState(result.state)
      toast.success(`已使用 ${item.name}`)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '物品使用失败')
    } finally {
      setUsingItemId(null)
    }
  }

  return (
    <div className="flex flex-col lg:flex-row gap-4 h-[calc(100vh-220px)] min-h-[400px]">
      {/* Left: General Info */}
      <div className="flex-1 rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4 flex flex-col">
        {/* Header */}
        <div className="flex items-center gap-3 mb-4 pb-3 border-b border-[var(--color-border)]">
          <div className="w-14 h-14 rounded-xl bg-[var(--color-surface-dim)] border border-[var(--color-border)] flex items-center justify-center">
            <span className="text-2xl">⚔️</span>
          </div>
          <div>
            <h2 className="text-lg font-bold text-[var(--color-text-primary)]">{general.name}</h2>
            <div className="flex items-center gap-2 mt-0.5">
              <span className="text-xs px-2 py-0.5 rounded-md bg-amber-500/15 text-amber-600 font-bold">Lv.{general.level}</span>
              <span className="text-[10px] text-[var(--color-text-muted)]">EXP {general.exp}</span>
            </div>
          </div>
        </div>

        <div className="mb-4 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2">
          <div className="flex items-center justify-between text-[10px] text-[var(--color-text-muted)] mb-1.5">
            <span>当前累计经验 {general.exp.toLocaleString()}</span>
            <span>{nextLevelExp > 0 ? `下级还需 ${expToNext.toLocaleString()}` : '已满级'}</span>
          </div>
          <div className="h-2 rounded-full bg-black/10 dark:bg-white/10 overflow-hidden">
            <div
              className="h-full rounded-full bg-amber-500 transition-all"
              style={{ width: `${expProgress}%` }}
            />
          </div>
        </div>

        <div className="mb-4">
          <div className="flex items-center justify-between mb-2">
            <h3 className="text-xs font-semibold text-[var(--color-text-primary)]">四维</h3>
            <span className="text-[10px] text-[var(--color-text-muted)]">可用点数 {availableStatPoints}</span>
          </div>
          
          {/* 四维属性说明 */}
          <div className="mb-3 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2">
            <p className="text-[10px] text-[var(--color-text-muted)] leading-relaxed">
              <span className="text-amber-600 font-semibold">武力</span> 提升部队攻击 · 
              <span className="text-blue-500 font-semibold"> 智谋</span> 提升征兵速度和行军速度 · 
              <span className="text-green-500 font-semibold"> 内政</span> 提升资源产量和仓库容量 · 
              <span className="text-purple-500 font-semibold"> 统率</span> 提升部队防御
            </p>
          </div>

          <div className="grid grid-cols-2 gap-2">
            {statEntries.map(([key, value]) => (
              <div key={key} className="flex items-center justify-between gap-2 px-3 py-2 rounded-xl bg-[var(--color-surface-dim)] border border-[var(--color-border)]">
                <div className="flex items-center gap-2 flex-1 justify-between">
                  <span className={`text-xs font-semibold ${STAT_COLORS[key]}`}>{STAT_LABELS[key]}</span>
                  <span className="text-xs font-bold text-[var(--color-text-primary)]">{value}/100</span>
                </div>
                <button
                  type="button"
                  onClick={() => handleAllocateStat(key)}
                  disabled={availableStatPoints <= 0 || value >= 100 || allocatingStat !== null}
                  className="h-7 w-7 flex-shrink-0 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] text-sm font-bold text-amber-600 disabled:opacity-40 disabled:cursor-not-allowed enabled:hover:bg-amber-500/10 transition-colors"
                  title={`提升${STAT_LABELS[key]}`}
                >
                  {allocatingStat === key ? '…' : '+'}
                </button>
              </div>
            ))}
          </div>
        </div>

        {/* Attributes */}
        <div className="mb-4">
          <h3 className="text-xs font-semibold text-[var(--color-text-primary)] mb-2">属性</h3>
          {attributeEntries.length > 0 ? (
            <div className="grid grid-cols-2 gap-2">
              {attributeEntries.map(([key, value]) => (
                <div
                  key={key}
                  className="group relative flex items-center justify-between px-3 py-2 rounded-xl bg-[var(--color-surface-dim)] border border-[var(--color-border)]"
                  title={formatBreakdownTitle(ATTRIBUTE_LABELS[key] ?? key, value, attributeBreakdown[key] ?? [])}
                >
                  <span className="text-[11px] text-[var(--color-text-secondary)]">{ATTRIBUTE_LABELS[key] ?? key}</span>
                  <span className="text-xs font-bold text-green-500">{formatAttributeValue(value)}</span>
                  {(attributeBreakdown[key]?.length ?? 0) > 0 && (
                    <div className="pointer-events-none absolute left-2 right-2 bottom-[calc(100%+6px)] z-20 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 shadow-lg opacity-0 invisible translate-y-1 scale-95 transition-all duration-150 ease-out group-hover:visible group-hover:opacity-100 group-hover:translate-y-0 group-hover:scale-100">
                      <div className="mb-1 flex items-center justify-between gap-3">
                        <span className="text-[11px] font-semibold text-[var(--color-text-primary)]">{ATTRIBUTE_LABELS[key] ?? key}</span>
                        <span className="text-[11px] font-bold text-green-500">{formatAttributeValue(value)}</span>
                      </div>
                      <div className="space-y-0.5">
                        {(attributeBreakdown[key] ?? []).map((item, index) => (
                          <div key={`${item.source}-${index}`} className="flex items-center justify-between gap-3 text-[10px]">
                            <span className="text-[var(--color-text-secondary)]">{item.source}</span>
                            <span className="font-semibold text-[var(--color-text-primary)]">{formatAttributeValue(item.value)}</span>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              ))}
            </div>
          ) : (
            <p className="text-[11px] text-[var(--color-text-muted)]">暂无属性加成，升级或配置将领后生效</p>
          )}
        </div>

        {/* Traits */}
        <div className="flex-1">
          <h3 className="text-xs font-semibold text-[var(--color-text-primary)] mb-2">将领特性</h3>
          {traits.length > 0 ? (
            <div className="space-y-2">
              {traits.map((trait) => {
                const meta = getTraitMeta(trait.traitId)
                return (
                  <div key={trait.traitId} className="rounded-xl border border-amber-500/30 bg-amber-500/5 p-3">
                    <div className="flex items-center gap-2 mb-1.5">
                      <span className="text-base">{meta.icon}</span>
                      <span className="text-sm font-bold text-amber-600">{meta.name}</span>
                      <span className="text-[10px] text-amber-600/70 ml-auto">{meta.trigger}</span>
                    </div>
                    <p className="text-[11px] text-[var(--color-text-secondary)] mb-2">{meta.description}</p>
                    {Object.keys(trait.params).length > 0 && (
                      <div className="flex flex-wrap gap-1.5">
                        {Object.entries(trait.params).map(([key, val]) => (
                          <span key={key} className="text-[10px] px-2 py-0.5 rounded bg-white/60 dark:bg-white/5 border border-amber-500/20 text-[var(--color-text-secondary)]">
                            {formatParamLabel(key)}: <span className="font-bold text-amber-600">{formatParamValue(key, val)}</span>
                          </span>
                        ))}
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          ) : (
            <p className="text-[11px] text-[var(--color-text-muted)]">该将领暂无特性</p>
          )}
        </div>
      </div>

      {/* Right: Inventory */}
      <div className="flex-1 rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4 flex flex-col">
        <div className="flex items-center gap-2 mb-3">
          <Package size={14} className="text-[var(--color-accent)]" />
          <h3 className="text-xs font-semibold text-[var(--color-text-primary)]">背包</h3>
          <span className="ml-auto text-[10px] text-[var(--color-text-muted)]">{inventoryRows.length} 类</span>
        </div>
        {inventoryRows.length === 0 ? (
          <div className="flex-1 rounded-xl border border-dashed border-[var(--color-border)] bg-[var(--color-surface-dim)] flex items-center justify-center">
            <span className="text-xs text-[var(--color-text-muted)]">暂无物品</span>
          </div>
        ) : (
          <div className="space-y-2 overflow-y-auto pr-1 scrollbar-none">
            {inventoryRows.map(({ stack, item }) => {
              const Icon = effectIcon(item?.effects ?? [])
              const rarity = RARITY_CLASS[item?.rarity ?? 'common'] ?? RARITY_CLASS.common
              return (
                <div key={stack.itemId} className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2">
                  <div className="flex items-start gap-2">
                    <div className="w-9 h-9 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] flex items-center justify-center shrink-0">
                      <Icon size={16} className="text-[var(--color-accent)]" />
                    </div>
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-1.5 min-w-0">
                        <span className="text-xs font-semibold text-[var(--color-text-primary)] truncate">{item?.name ?? stack.itemId}</span>
                        <span className={`shrink-0 text-[9px] font-medium px-1.5 py-0.5 rounded-md border ${rarity}`}>
                          {RARITY_LABEL[item?.rarity ?? 'common'] ?? item?.rarity ?? '普通'}
                        </span>
                      </div>
                      <p className="mt-0.5 text-[10px] leading-relaxed text-[var(--color-text-muted)] line-clamp-2">
                        {item?.description ?? '配置不存在，请检查物品表。'}
                      </p>
                    </div>
                  </div>
                  <div className="mt-2 flex items-center justify-between gap-2">
                    <span className="text-xs font-bold text-[var(--color-text-primary)]">x{stack.amount.toLocaleString()}</span>
                    <button
                      type="button"
                      disabled={!item?.usable || usingItemId === stack.itemId}
                      onClick={() => handleUseItem(stack, item)}
                      className="
                        inline-flex items-center justify-center gap-1 h-7 px-2.5 rounded-lg
                        text-[11px] font-semibold bg-[var(--color-accent)] text-white
                        disabled:opacity-45 disabled:cursor-not-allowed
                        hover:brightness-105 cursor-pointer transition
                      "
                    >
                      <Sparkles size={12} />
                      使用
                    </button>
                  </div>
                </div>
              )
            })}
          </div>
        )}
            </div>
    </div>
  )
}

export default GeneralPanel
