// 武将详情面板，负责展示将领成长、特性和背包物品操作。
import { type FC, useMemo, useState } from 'react'
import { Boxes, Package, RefreshCcw, Repeat2, Sparkles, Swords, UserRound, X } from 'lucide-react'
import { gameApi } from '@/api/game'
import { toast } from '@/components/ui'
import { useAccountStore } from '@/store/accountStore'
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
const STAT_BAR_COLORS: Record<string, string> = {
  force: 'bg-amber-500',
  intelligence: 'bg-blue-500',
  politics: 'bg-green-500',
  command: 'bg-purple-500',
}
const STAT_ATTRIBUTE_KEYS: Record<string, string[]> = {
  force: ['attackBonus'],
  intelligence: ['recruitSpeedBonus', 'marchSpeedBonus', 'buildSpeedBonus'],
  politics: ['productionBonus', 'woodProductionBonus', 'stoneProductionBonus', 'ironProductionBonus', 'foodProductionBonus', 'capacityBonus', 'exchangeRateBonus'],
  command: ['defenseBonus', 'infantryDefenseBonus', 'cavalryDefenseBonus'],
}
const STAT_ORDER = ['force', 'intelligence', 'politics', 'command']
const QUALITY_CLASS: Record<string, string> = {
  common: 'text-slate-500 bg-slate-500/10 border-slate-500/20',
  rare: 'text-blue-500 bg-blue-500/10 border-blue-500/20',
  epic: 'text-fuchsia-500 bg-fuchsia-500/10 border-fuchsia-500/20',
  legendary: 'text-amber-500 bg-amber-500/10 border-amber-500/20',
  mythic: 'text-rose-500 bg-rose-500/10 border-rose-500/20',
}
const QUALITY_LABEL: Record<string, string> = {
  common: '普通',
  rare: '稀有',
  epic: '史诗',
  legendary: '传说',
  mythic: '神话',
}

const effectIcon = (effects: ItemEffect[]) => {
  if (effects.some((effect) => effect.type === 'general_exp')) return UserRound
  if (effects.some((effect) => effect.type === 'unit_by_faction')) return Swords
  return Boxes
}

const itemQuality = (item?: ItemDefinition) => item?.quality ?? item?.rarity ?? 'common'

const GeneralPanel: FC = () => {
  const state = useGameStore((s) => s.state)
  const general = useGameStore((s) => s.state?.general)
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const patchState = useGameStore((s) => s.patchState)
  const allocateGeneralStat = useGameStore((s) => s.allocateGeneralStat)
  const resetGeneralStats = useGameStore((s) => s.resetGeneralStats)
  const changeGeneral = useGameStore((s) => s.changeGeneral)
  const account = useAccountStore((s) => s.account)
  const factions = useConfigStore((s) => s.factions)
  const itemsConfig = useConfigStore((s) => s.items)
  const [allocatingStat, setAllocatingStat] = useState<string | null>(null)
  const [usingItemId, setUsingItemId] = useState<string | null>(null)
  const [resettingStats, setResettingStats] = useState(false)
  const [changingGeneral, setChangingGeneral] = useState(false)
  const [selectedGeneralId, setSelectedGeneralId] = useState('')
  const [showChangeGeneralDialog, setShowChangeGeneralDialog] = useState(false)
  const [selectedInventoryItem, setSelectedInventoryItem] = useState<{ stack: ItemStack; item?: ItemDefinition } | null>(null)
  const [selectedItemUseAmount, setSelectedItemUseAmount] = useState(1)
  const inventoryRows = useMemo(() => {
    const slots = state?.inventorySlots?.length ? state.inventorySlots : Object.values(state?.inventory ?? {})
    return slots
      .filter((stack) => stack.amount > 0)
      .map((stack) => ({
        stack,
        item: itemsConfig?.[stack.itemId],
      }))
      .sort((a, b) => {
        return (a.stack.slotId ?? a.stack.itemId).localeCompare(b.stack.slotId ?? b.stack.itemId, 'zh-Hans-CN')
      })
  }, [itemsConfig, state?.inventory, state?.inventorySlots])

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
  const nextLevelExp = general.nextLevelExp ?? 0
  const expToNext = nextLevelExp > 0 ? Math.max(nextLevelExp - general.exp, 0) : 0
  const expProgress = nextLevelExp > 0 ? Math.min(100, (general.exp / nextLevelExp) * 100) : 100
  const expProgressText = `${expProgress.toFixed(2)}%`
  const statEntries = STAT_ORDER.map((key) => [key, general.stats?.[key] ?? 0] as const)
  const availableStatPoints = general.availableStatPoints ?? 0
  const factionGenerals = state?.player.faction ? (factions?.[state.player.faction]?.generals ?? []) : []
  const changeTargets = factionGenerals.filter((item) => item.id !== general.id)
  const targetGeneralId = selectedGeneralId || changeTargets[0]?.id || ''
  const selectedChangeTarget = changeTargets.find((item) => item.id === targetGeneralId)

  // 根据四维归类展示对应的属性加成，让信息直接贴在进度条上。
  const getStatAttributeEntries = (statKey: string) => {
    return (STAT_ATTRIBUTE_KEYS[statKey] ?? [])
      .map((attributeKey) => [attributeKey, attributes[attributeKey] ?? 0] as const)
      .filter(([, value]) => value !== 0)
  }

  // 提升指定四维属性点。
  const handleAllocateStat = async (statKey: string) => {
    if (availableStatPoints <= 0 || allocatingStat) return
    setAllocatingStat(statKey)
    try {
      await allocateGeneralStat(statKey)
    } finally {
      setAllocatingStat(null)
    }
  }

  // 打开背包物品详情弹窗。
  const openInventoryItem = (stack: ItemStack, item?: ItemDefinition) => {
    setSelectedInventoryItem({ stack, item })
    setSelectedItemUseAmount(1)
  }

  // 关闭背包物品详情弹窗。
  const closeInventoryItem = () => {
    setSelectedInventoryItem(null)
    setSelectedItemUseAmount(1)
  }

  // 使用背包中的指定物品。
  const handleUseItem = async (stack: ItemStack, item?: ItemDefinition, amount = 1) => {
    if (!activePlayerId || !item?.usable || usingItemId) return
    const useAmount = Math.max(1, Math.min(stack.amount, Math.floor(amount) || 1))
    setUsingItemId(stack.itemId)
    try {
      const result = await gameApi.useItem(activePlayerId, stack.itemId, useAmount)
      patchState(result.patch)
      closeInventoryItem()
      toast.success(`已使用 ${item.name} ×${useAmount.toLocaleString()}`)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '物品使用失败')
    } finally {
      setUsingItemId(null)
    }
  }

  // 消耗金币重置四维加点。
  const handleResetStats = async () => {
    if (resettingStats) return
    const confirmed = window.confirm('确认消耗 10 金币重置四维加点？等级和经验会保留。')
    if (!confirmed) return
    setResettingStats(true)
    try {
      const accountGold = await resetGeneralStats()
      if (account && accountGold !== undefined) {
        useAccountStore.setState({ account: { ...account, gold: accountGold } })
      }
      toast.success('将领洗点成功')
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '洗点失败')
    } finally {
      setResettingStats(false)
    }
  }

  // 打开换将选择弹窗。
  const handleOpenChangeGeneral = () => {
    if (changeTargets.length === 0 || changingGeneral) return
    setSelectedGeneralId((current) => current || changeTargets[0]?.id || '')
    setShowChangeGeneralDialog(true)
  }

  // 确认更换当前将领。
  const handleChangeGeneral = async () => {
    if (!targetGeneralId || changingGeneral) return
    setChangingGeneral(true)
    try {
      const accountGold = await changeGeneral(targetGeneralId)
      if (account && accountGold !== undefined) {
        useAccountStore.setState({ account: { ...account, gold: accountGold } })
      }
      setSelectedGeneralId('')
      setShowChangeGeneralDialog(false)
      toast.success('换将成功')
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '换将失败')
    } finally {
      setChangingGeneral(false)
    }
  }

  return (
    <div className="flex flex-col lg:flex-row gap-4 h-[calc(100vh-220px)] min-h-[400px]">
      {/* Left: General Info */}
      <div className="flex-1 min-h-0 overflow-y-auto rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4 flex flex-col scrollbar-none">
        {/* Header */}
        <div className="flex items-center gap-3 mb-4 pb-3 border-b border-[var(--color-border)]">
          <div className="w-14 h-14 rounded-xl bg-[var(--color-surface-dim)] border border-[var(--color-border)] flex items-center justify-center">
            <span className="text-2xl">⚔️</span>
          </div>
          <div className="min-w-0 flex-1">
            <h2 className="text-lg font-bold text-[var(--color-text-primary)]">{general.name}</h2>
            <div className="flex items-center gap-2 mt-0.5">
              <span className="text-xs px-2 py-0.5 rounded-md bg-amber-500/15 text-amber-600 font-bold">Lv.{general.level}</span>
              <span className="text-[10px] text-[var(--color-text-muted)]">EXP {expProgressText}</span>
            </div>
          </div>
          <div className="flex shrink-0 items-center gap-2">
            <button
              type="button"
              onClick={() => void handleResetStats()}
              disabled={resettingStats}
              className="inline-flex h-8 items-center justify-center gap-1.5 rounded-lg border border-amber-500/30 bg-amber-500/10 px-2.5 text-[11px] font-bold text-amber-600 transition hover:bg-amber-500/15 disabled:cursor-not-allowed disabled:opacity-50"
              title={`消耗 10 金币洗点，当前金币 ${account?.gold ?? 0}`}
            >
              <RefreshCcw size={12} />
              {resettingStats ? '洗点中' : '洗点'}
            </button>
            <button
              type="button"
              onClick={handleOpenChangeGeneral}
              disabled={changeTargets.length === 0 || changingGeneral}
              className="inline-flex h-8 items-center justify-center gap-1.5 rounded-lg border border-blue-500/30 bg-blue-500/10 px-2.5 text-[11px] font-bold text-blue-600 transition hover:bg-blue-500/15 disabled:cursor-not-allowed disabled:opacity-50"
              title={changeTargets.length === 0 ? '无可更换将领' : '选择要更换的将领'}
            >
              <Repeat2 size={12} />
              换将
            </button>
          </div>
        </div>

        <div className="mb-4 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2">
          <div className="flex items-center justify-between text-[10px] text-[var(--color-text-muted)] mb-1.5">
            <span>当前经验进度 {expProgressText}</span>
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
            {availableStatPoints > 0 && <span className="text-[10px] font-semibold text-amber-600">可加点</span>}
          </div>
          <div className="space-y-2">
            {statEntries.map(([key, value]) => {
              const statAttributes = getStatAttributeEntries(key)
              const progress = Math.min(100, Math.max(0, value))
              return (
                <div key={key} className="relative h-10 overflow-hidden rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
                  <div
                    className={`absolute inset-y-0 left-0 rounded-xl ${STAT_BAR_COLORS[key]} opacity-25 transition-all`}
                    style={{ width: `${progress}%` }}
                  />
                  <div className="relative flex h-full items-center gap-2 px-3">
                    <span className={`shrink-0 text-xs font-bold ${STAT_COLORS[key]}`}>{STAT_LABELS[key]}</span>
                    <div className="ml-auto flex min-w-0 flex-wrap justify-end gap-1 overflow-hidden">
                      {statAttributes.slice(0, 3).map(([attributeKey, attributeValue]) => (
                        <span
                          key={attributeKey}
                          className="truncate rounded-md border border-green-500/20 bg-green-500/10 px-1.5 py-0.5 text-[10px] font-bold text-green-500"
                          title={formatBreakdownTitle(ATTRIBUTE_LABELS[attributeKey] ?? attributeKey, attributeValue, attributeBreakdown[attributeKey] ?? [])}
                        >
                          {ATTRIBUTE_LABELS[attributeKey] ?? attributeKey} {formatAttributeValue(attributeValue)}
                        </span>
                      ))}
                    </div>
                    <button
                      type="button"
                      onClick={() => handleAllocateStat(key)}
                      disabled={availableStatPoints <= 0 || value >= 100 || allocatingStat !== null}
                      className="grid h-7 w-7 flex-shrink-0 place-items-center rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] text-xs font-bold text-amber-600 transition-colors enabled:hover:bg-amber-500/10 disabled:cursor-not-allowed disabled:opacity-40"
                      title={`提升${STAT_LABELS[key]}`}
                    >
                      {allocatingStat === key ? '…' : '+'}
                    </button>
                  </div>
                </div>
              )
            })}
          </div>
        </div>

        {/* Traits */}
        <div className="min-h-0 flex-1">
          <h3 className="text-xs font-semibold text-[var(--color-text-primary)] mb-2">将领特性</h3>
          {traits.length > 0 ? (
            <div className="max-h-[220px] space-y-2 overflow-y-auto pr-1 scrollbar-none">
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
      <div className="flex-1 min-h-0 rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4 flex flex-col">
        <div className="flex items-center gap-2 mb-3">
          <Package size={14} className="text-[var(--color-accent)]" />
          <h3 className="text-xs font-semibold text-[var(--color-text-primary)]">背包</h3>
          <span className="ml-auto text-[10px] text-[var(--color-text-muted)]">{inventoryRows.length} / 9999</span>
        </div>
        {inventoryRows.length === 0 ? (
          <div className="flex-1 rounded-xl border border-dashed border-[var(--color-border)] bg-[var(--color-surface-dim)] flex items-center justify-center">
            <span className="text-xs text-[var(--color-text-muted)]">暂无物品</span>
          </div>
        ) : (
          <div className="grid grid-cols-[repeat(auto-fill,72px)] content-start gap-3 overflow-y-auto pr-1 scrollbar-none">
            {inventoryRows.map(({ stack, item }) => {
              const Icon = effectIcon(item?.effects ?? [])
              const quality = itemQuality(item)
              const qualityClass = QUALITY_CLASS[quality] ?? QUALITY_CLASS.common
              return (
                <button
                  key={stack.slotId ?? stack.itemId}
                  type="button"
                  onClick={() => openInventoryItem(stack, item)}
                  className={`relative grid h-[72px] w-[72px] place-items-center rounded-lg border bg-[var(--color-surface-dim)] cursor-pointer transition hover:brightness-110 ${qualityClass}`}
                  title={item?.name ?? stack.itemId}
                >
                  <Icon size={24} />
                  <span className="absolute right-1.5 top-1 text-[10px] font-bold text-[var(--color-text-primary)]">
                    x{stack.amount.toLocaleString()}
                  </span>
                </button>
              )
            })}
          </div>
        )}
      </div>
      {showChangeGeneralDialog && (
        <div className="fixed inset-0 z-[9000] flex items-center justify-center bg-slate-950/50 p-4 backdrop-blur-sm" onClick={() => setShowChangeGeneralDialog(false)}>
          <div className="w-full max-w-[380px] rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4 shadow-2xl" onClick={(event) => event.stopPropagation()}>
            <div className="mb-3 flex items-center gap-2">
              <div className="grid h-9 w-9 place-items-center rounded-xl border border-blue-500/25 bg-blue-500/10 text-blue-500">
                <Repeat2 size={16} />
              </div>
              <div className="min-w-0 flex-1">
                <h4 className="text-sm font-bold text-[var(--color-text-primary)]">选择将领</h4>
                <p className="text-[10px] text-[var(--color-text-muted)]">保留等级和经验，四维加点会重置</p>
              </div>
              <button
                type="button"
                onClick={() => setShowChangeGeneralDialog(false)}
                className="grid h-7 w-7 place-items-center rounded-lg text-[var(--color-text-muted)] hover:bg-[var(--color-surface-dim)] hover:text-[var(--color-text-primary)]"
              >
                <X size={15} />
              </button>
            </div>
            <div className="max-h-[260px] space-y-2 overflow-y-auto pr-1 scrollbar-none">
              {changeTargets.map((item) => (
                <button
                  key={item.id}
                  type="button"
                  onClick={() => setSelectedGeneralId(item.id)}
                  className={`flex w-full items-center justify-between rounded-xl border px-3 py-2 text-left transition ${
                    targetGeneralId === item.id
                      ? 'border-blue-500/50 bg-blue-500/10'
                      : 'border-[var(--color-border)] bg-[var(--color-surface-dim)] hover:border-blue-500/30'
                  }`}
                >
                  <span className="text-xs font-bold text-[var(--color-text-primary)]">{item.name}</span>
                  {targetGeneralId === item.id && <span className="text-[10px] font-bold text-blue-500">已选择</span>}
                </button>
              ))}
            </div>
            <div className="mt-4 grid grid-cols-2 gap-2">
              <button
                type="button"
                onClick={() => setShowChangeGeneralDialog(false)}
                className="h-9 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] text-xs font-bold text-[var(--color-text-secondary)] hover:border-[var(--color-accent-border)]"
              >
                取消
              </button>
              <button
                type="button"
                disabled={!selectedChangeTarget || changingGeneral}
                onClick={() => void handleChangeGeneral()}
                className="inline-flex h-9 items-center justify-center gap-1.5 rounded-xl bg-blue-500 text-xs font-bold text-white transition hover:brightness-105 disabled:cursor-not-allowed disabled:opacity-45"
              >
                <Repeat2 size={13} />
                {changingGeneral ? '更换中' : '确认换将'}
              </button>
            </div>
          </div>
        </div>
      )}
      {selectedInventoryItem && (() => {
        const { stack, item } = selectedInventoryItem
        const Icon = effectIcon(item?.effects ?? [])
        const quality = itemQuality(item)
        const qualityClass = QUALITY_CLASS[quality] ?? QUALITY_CLASS.common
        const canBatchUse = Boolean(item?.usable && item.stackable && stack.amount > 1)
        const useAmount = Math.max(1, Math.min(stack.amount, Math.floor(selectedItemUseAmount) || 1))
        return (
          <div className="fixed inset-0 z-[9000] flex items-center justify-center bg-slate-950/50 p-4 backdrop-blur-sm" onClick={closeInventoryItem}>
            <div className="w-full max-w-[360px] rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4 shadow-2xl" onClick={(event) => event.stopPropagation()}>
              <div className="mb-3 flex items-start gap-3">
                <div className={`relative grid h-[72px] w-[72px] shrink-0 place-items-center rounded-lg border bg-[var(--color-surface-dim)] ${qualityClass}`}>
                  <Icon size={26} />
                  <span className="absolute right-1.5 top-1 text-[10px] font-bold text-[var(--color-text-primary)]">x{stack.amount.toLocaleString()}</span>
                </div>
                <div className="min-w-0 flex-1">
                  <div className="flex items-start gap-2">
                    <div className="min-w-0 flex-1">
                      <h4 className="text-sm font-bold text-[var(--color-text-primary)]">{item?.name ?? stack.itemId}</h4>
                      <div className={`mt-1 inline-flex rounded-md border px-1.5 py-0.5 text-[10px] font-semibold ${qualityClass}`}>
                        {QUALITY_LABEL[quality] ?? quality}
                      </div>
                    </div>
                    <button
                      type="button"
                      onClick={closeInventoryItem}
                      className="grid h-7 w-7 place-items-center rounded-lg text-[var(--color-text-muted)] hover:bg-[var(--color-surface-dim)] hover:text-[var(--color-text-primary)]"
                    >
                      <X size={15} />
                    </button>
                  </div>
                </div>
              </div>
              <p className="min-h-16 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-3 text-xs leading-relaxed text-[var(--color-text-secondary)]">
                {item?.description ?? '配置不存在，请检查物品表。'}
              </p>
              {canBatchUse && (
                <div className="mt-3 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-3">
                  <div className="mb-2 flex items-center justify-between">
                    <span className="text-[11px] font-bold text-[var(--color-text-primary)]">使用数量</span>
                    <span className="text-[10px] text-[var(--color-text-muted)]">拥有 {stack.amount.toLocaleString()}</span>
                  </div>
                  <div className="grid grid-cols-[1fr_auto] gap-2">
                    <input
                      type="number"
                      min={1}
                      max={stack.amount}
                      value={selectedItemUseAmount}
                      onChange={(event) => {
                        const next = Math.max(1, Math.min(stack.amount, Math.floor(Number(event.target.value)) || 1))
                        setSelectedItemUseAmount(next)
                      }}
                      className="h-9 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-3 text-sm font-bold text-[var(--color-text-primary)] outline-none focus:border-[var(--color-accent-border)]"
                    />
                    <button
                      type="button"
                      onClick={() => setSelectedItemUseAmount(stack.amount)}
                      className="h-9 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-3 text-xs font-bold text-[var(--color-text-secondary)] hover:border-[var(--color-accent-border)]"
                    >
                      全部
                    </button>
                  </div>
                </div>
              )}
              <div className="mt-4 grid grid-cols-2 gap-2">
                <button
                  type="button"
                  onClick={closeInventoryItem}
                  className="h-9 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] text-xs font-bold text-[var(--color-text-secondary)] hover:border-[var(--color-accent-border)]"
                >
                  取消
                </button>
                <button
                  type="button"
                  disabled={!item?.usable || usingItemId === stack.itemId}
                  onClick={() => handleUseItem(stack, item, canBatchUse ? useAmount : 1)}
                  className="inline-flex h-9 items-center justify-center gap-1.5 rounded-xl bg-[var(--color-accent)] text-xs font-bold text-white transition hover:brightness-105 disabled:cursor-not-allowed disabled:opacity-45"
                >
                  <Sparkles size={13} />
                  {canBatchUse ? `使用 ×${useAmount.toLocaleString()}` : '使用'}
                </button>
              </div>
            </div>
          </div>
        )
      })()}
    </div>
  )
}

export default GeneralPanel
