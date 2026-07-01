// 本文件提供天机轮转奖励库存弹窗，用于查看并兑换三连奖励记录。
import { Loader2, PackageCheck, X } from 'lucide-react'
import { useMemo, type FC } from 'react'
import type { MiniGameRecord } from '@/types/game'
import { RARITY_CONFIG } from './fishing/fishingConfig'
import type { FishCatch } from './fishing/types'

interface SlotInventoryModalProps {
  records: MiniGameRecord[]
  recordsLoading: boolean
  recordsTotal: number
  recordsHasMore: boolean
  recordsOffset: number
  recordsPageSize: number
  redeemingId: string
  redeemingAll: boolean
  isFactionUnit: (unitName: string) => boolean
  onClose: () => void
  onRefresh: () => void
  onPageChange: (offset: number) => void
  onRedeemAll: () => void
  onRedeemGroup: (unitName: string, records: MiniGameRecord[]) => void
}

type SlotInventoryGroup = {
  rewardUnit: string
  records: MiniGameRecord[]
  totalAmount: number
  roundTags: Array<{ name: string; count: number; amount: number; rarity: FishCatch['rarity'] }>
  highestRarity: FishCatch['rarity']
  target: 'army' | 'garrison'
}

const rarityRank: Record<FishCatch['rarity'], number> = {
  common: 1,
  rare: 2,
  epic: 3,
  legendary: 4,
}

// normalizeRarity 归一化后端记录稀有度，避免旧数据破坏样式。
const normalizeRarity = (rarity: string): FishCatch['rarity'] => {
  return (rarity in RARITY_CONFIG ? rarity : 'common') as FishCatch['rarity']
}

// SlotInventoryModal 展示天机库存并按兵种分组兑换。
export const SlotInventoryModal: FC<SlotInventoryModalProps> = ({
  records,
  recordsLoading,
  recordsTotal,
  recordsHasMore,
  recordsOffset,
  recordsPageSize,
  redeemingId,
  redeemingAll,
  isFactionUnit,
  onClose,
  onRefresh,
  onPageChange,
  onRedeemAll,
  onRedeemGroup,
}) => {
  const inventoryRecords = records.filter(record => record.remainingAmount > 0)
  const groups = useMemo<SlotInventoryGroup[]>(() => {
    const map = new Map<string, MiniGameRecord[]>()
    for (const record of inventoryRecords) {
      if (!record.rewardUnit) continue
      map.set(record.rewardUnit, [...(map.get(record.rewardUnit) ?? []), record])
    }
    return Array.from(map.entries()).map(([rewardUnit, groupRecords]) => {
      const roundMap = new Map<string, { name: string; count: number; amount: number; rarity: FishCatch['rarity'] }>()
      let highestRarity: FishCatch['rarity'] = 'common'
      for (const record of groupRecords) {
        const rarity = normalizeRarity(record.rarity)
        if (rarityRank[rarity] > rarityRank[highestRarity]) highestRarity = rarity
        const betText = record.betUnit && record.betAmount ? ` · 押 ${record.betUnit} ×${record.betAmount.toLocaleString()}` : ''
        const tagName = `${record.resultName}${betText}`
        const tag = roundMap.get(tagName) ?? { name: tagName, count: 0, amount: 0, rarity }
        tag.count += 1
        tag.amount += record.remainingAmount
        if (rarityRank[rarity] > rarityRank[tag.rarity]) tag.rarity = rarity
        roundMap.set(tagName, tag)
      }
      return {
        rewardUnit,
        records: groupRecords,
        totalAmount: groupRecords.reduce((sum, record) => sum + record.remainingAmount, 0),
        roundTags: Array.from(roundMap.values()).sort((a, b) => rarityRank[b.rarity] - rarityRank[a.rarity] || b.amount - a.amount),
        highestRarity,
        target: (isFactionUnit(rewardUnit) ? 'army' : 'garrison') as SlotInventoryGroup['target'],
      }
    }).sort((a, b) => Number(a.target === 'garrison') - Number(b.target === 'garrison') || rarityRank[b.highestRarity] - rarityRank[a.highestRarity] || b.totalAmount - a.totalAmount)
  }, [inventoryRecords, isFactionUnit])
  const redeemableAmount = groups.reduce((sum, group) => sum + group.totalAmount, 0)
  const garrisonAmount = groups.filter(group => group.target === 'garrison').reduce((sum, group) => sum + group.totalAmount, 0)
  const currentPage = Math.floor(recordsOffset / recordsPageSize) + 1
  const totalPages = Math.max(1, Math.ceil(recordsTotal / recordsPageSize))
  const canPrevPage = recordsOffset > 0
  const canNextPage = recordsHasMore

  return (
    <div className="fixed inset-0 z-[9000] flex items-center justify-center bg-slate-950/55 p-3 backdrop-blur-sm animate-fishing-inventory-backdrop-in">
      <button type="button" aria-label="关闭天机库存" onClick={onClose} className="absolute inset-0 cursor-default" />
      <div className="relative h-[82vh] w-full max-w-[420px] animate-fishing-inventory-dialog-in">
        <div className="flex h-full min-h-0 flex-col rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] shadow-[0_18px_50px_rgba(15,23,42,0.12)]">
          <div className="shrink-0 border-b border-[var(--color-border)] px-3.5 py-3">
            <div className="flex items-center justify-between gap-3">
              <div className="flex min-w-0 items-center gap-2">
                <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-xl bg-violet-500/10 text-violet-600">
                  <PackageCheck size={16} />
                </div>
                <div className="min-w-0">
                  <p className="truncate text-sm font-bold leading-4 text-[var(--color-text-primary)]">天机库存</p>
                  <p className="mt-0.5 truncate text-[10px] leading-4 text-[var(--color-text-muted)]">
                    {recordsTotal > 0 ? `第 ${currentPage}/${totalPages} 页 · 本页 ${records.length} 条` : '转出三连奖励后会存入这里'}
                  </p>
                </div>
              </div>
              <div className="flex shrink-0 items-center gap-1.5">
                {recordsLoading && <Loader2 size={13} className="animate-spin text-[var(--color-text-muted)]" />}
                <button
                  type="button"
                  onClick={onRefresh}
                  className="flex h-8 items-center justify-center rounded-lg border border-[var(--color-border)] px-2.5 text-[10px] text-[var(--color-text-secondary)] transition-colors hover:border-[var(--color-accent-border)] hover:text-[var(--color-text-primary)] cursor-pointer"
                >
                  刷新
                </button>
                <button
                  type="button"
                  onClick={onClose}
                  className="flex h-8 w-8 items-center justify-center rounded-lg border border-[var(--color-border)] text-[var(--color-text-muted)] transition-colors hover:text-[var(--color-text-primary)] cursor-pointer"
                  aria-label="关闭"
                >
                  <X size={15} />
                </button>
              </div>
            </div>
            {inventoryRecords.length > 0 && (
              <div className="mt-3 grid grid-cols-[1fr_1fr_auto] gap-2">
                <div className="min-w-0 rounded-xl bg-[var(--color-surface-dim)] px-2.5 py-2">
                  <p className="text-[9px] text-[var(--color-text-muted)]">可兑换</p>
                  <p className="mt-0.5 text-sm font-bold text-emerald-600">{redeemableAmount.toLocaleString()}</p>
                </div>
                <div className="min-w-0 rounded-xl bg-[var(--color-surface-dim)] px-2.5 py-2">
                  <p className="text-[9px] text-[var(--color-text-muted)]">入驻防</p>
                  <p className="mt-0.5 text-sm font-bold text-violet-600">{garrisonAmount.toLocaleString()}</p>
                </div>
                <button
                  type="button"
                  onClick={onRedeemAll}
                  disabled={recordsLoading || Boolean(redeemingId) || redeemableAmount <= 0}
                  className="inline-flex min-w-[76px] items-center justify-center gap-1 rounded-xl bg-violet-600 px-2.5 py-2 text-[10px] font-semibold text-white transition-opacity hover:opacity-90 disabled:opacity-50 cursor-pointer"
                >
                  {redeemingAll && <Loader2 size={11} className="animate-spin" />}
                  全部兑换
                </button>
              </div>
            )}
          </div>

          <div className="min-h-0 flex-1 overflow-y-auto px-3 py-3">
            {inventoryRecords.length === 0 ? (
              <div className="flex min-h-[180px] flex-col items-center justify-center rounded-xl border border-dashed border-[var(--color-border)] bg-[var(--color-surface-dim)] px-4 text-center">
                <PackageCheck size={24} className="mb-2 text-[var(--color-text-muted)]" />
                <p className="text-xs font-medium text-[var(--color-text-primary)]">暂无库存</p>
                <p className="mt-1 text-[10px] leading-5 text-[var(--color-text-muted)]">
                  三符同现的奖励会先存储，可在这里分批兑换。
                </p>
              </div>
            ) : (
              <div className="space-y-2">
                {groups.map((group, index) => {
                  const cfg = RARITY_CONFIG[group.highestRarity] ?? RARITY_CONFIG.common
                  const isRedeeming = redeemingId === group.rewardUnit
                  return (
                    <div
                      key={group.rewardUnit}
                      className="animate-fishing-inventory-item-in rounded-xl border border-violet-500/20 bg-violet-500/[0.04] px-2.5 py-2.5"
                      style={{ animationDelay: `${Math.min(index * 35, 210)}ms` }}
                    >
                      <div className="mb-2 flex flex-wrap gap-1">
                        {group.roundTags.map(tag => {
                          const tagCfg = RARITY_CONFIG[tag.rarity] ?? RARITY_CONFIG.common
                          return (
                            <span key={tag.name} className={`rounded-full px-2 py-0.5 text-[9px] font-semibold ${tagCfg.bg} ${tagCfg.color}`}>
                              {tag.name}{tag.count > 1 ? ` ×${tag.count}` : ''}
                            </span>
                          )
                        })}
                      </div>
                      <div className="flex min-w-0 items-center gap-2">
                        <span className={`shrink-0 rounded-md px-1.5 py-0.5 text-[9px] font-bold ${cfg.bg} ${cfg.color}`}>{cfg.label}</span>
                        <div className="min-w-0 flex-1">
                          <p className="truncate text-xs font-semibold text-[var(--color-text-primary)]">{group.rewardUnit}</p>
                          <p className="mt-0.5 truncate text-[10px] text-[var(--color-text-muted)]">
                            剩余 {group.totalAmount.toLocaleString()} · {group.records.length} 条记录
                          </p>
                        </div>
                      </div>
                      <div className="mt-2 flex items-center justify-between gap-2">
                        <span className={`text-[10px] ${group.target === 'army' ? 'text-emerald-700' : 'text-violet-700'}`}>
                          {group.target === 'army' ? '兑换后加入军队' : '兑换后加入驻防队伍'}
                        </span>
                        <button
                          type="button"
                          onClick={() => onRedeemGroup(group.rewardUnit, group.records)}
                          disabled={Boolean(redeemingId)}
                          className="inline-flex min-w-[86px] items-center justify-center gap-1 rounded-lg bg-violet-600 px-2.5 py-1.5 text-[10px] font-semibold text-white transition-opacity hover:opacity-90 disabled:opacity-60 cursor-pointer"
                        >
                          {isRedeeming && <Loader2 size={11} className="animate-spin" />}
                          一键兑换
                        </button>
                      </div>
                    </div>
                  )
                })}
              </div>
            )}
          </div>

          {recordsTotal > recordsPageSize && (
            <div className="shrink-0 border-t border-[var(--color-border)] px-3 py-2">
              <div className="flex items-center justify-between gap-2">
                <button
                  type="button"
                  onClick={() => onPageChange(Math.max(0, recordsOffset - recordsPageSize))}
                  disabled={!canPrevPage || recordsLoading}
                  className="rounded-lg border border-[var(--color-border)] px-2.5 py-1.5 text-[10px] text-[var(--color-text-secondary)] disabled:opacity-40 cursor-pointer"
                >
                  上一页
                </button>
                <span className="text-[10px] text-[var(--color-text-muted)]">{currentPage}/{totalPages}</span>
                <button
                  type="button"
                  onClick={() => onPageChange(recordsOffset + recordsPageSize)}
                  disabled={!canNextPage || recordsLoading}
                  className="rounded-lg border border-[var(--color-border)] px-2.5 py-1.5 text-[10px] text-[var(--color-text-secondary)] disabled:opacity-40 cursor-pointer"
                >
                  下一页
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
