import { Loader2, PackageCheck, X } from 'lucide-react'
import { useMemo, type FC } from 'react'
import type { MiniGameRecord } from '@/types/game'
import { RARITY_CONFIG } from './fishingConfig'
import type { FishCatch } from './types'

interface FishingInventoryModalProps {
  records: MiniGameRecord[]
  recordsLoading: boolean
  recordsTotal: number
  recordsHasMore: boolean
  recordsOffset: number
  recordsPageSize: number
  redeemingId: string
  isFactionUnit: (unitName: string) => boolean
  onClose: () => void
  onRefresh: () => void
  onPageChange: (offset: number) => void
  onRedeemGroup: (unitName: string, records: MiniGameRecord[]) => void
}

type InventoryGroup = {
  rewardUnit: string
  records: MiniGameRecord[]
  totalAmount: number
  originalAmount: number
  fishTags: Array<{ name: string; count: number; amount: number; rarity: FishCatch['rarity'] }>
  highestRarity: FishCatch['rarity']
  canRedeem: boolean
}

const rarityRank: Record<FishCatch['rarity'], number> = {
  common: 1,
  rare: 2,
  epic: 3,
  legendary: 4,
}

export const FishingInventoryModal: FC<FishingInventoryModalProps> = ({
  records,
  recordsLoading,
  recordsTotal,
  recordsHasMore,
  recordsOffset,
  recordsPageSize,
  redeemingId,
  isFactionUnit,
  onClose,
  onRefresh,
  onPageChange,
  onRedeemGroup,
}) => {
  const inventoryRecords = records.filter(record => record.remainingAmount > 0)
  const groups = useMemo<InventoryGroup[]>(() => {
    const map = new Map<string, MiniGameRecord[]>()
    for (const record of inventoryRecords) {
      if (!record.rewardUnit) continue
      map.set(record.rewardUnit, [...(map.get(record.rewardUnit) ?? []), record])
    }
    return Array.from(map.entries()).map(([rewardUnit, groupRecords]) => {
      const fishMap = new Map<string, { name: string; count: number; amount: number; rarity: FishCatch['rarity'] }>()
      let highestRarity: FishCatch['rarity'] = 'common'
      for (const record of groupRecords) {
        const rarity = (record.rarity in RARITY_CONFIG ? record.rarity : 'common') as FishCatch['rarity']
        if (rarityRank[rarity] > rarityRank[highestRarity]) highestRarity = rarity
        const tag = fishMap.get(record.resultName) ?? { name: record.resultName, count: 0, amount: 0, rarity }
        tag.count += 1
        tag.amount += record.remainingAmount
        if (rarityRank[rarity] > rarityRank[tag.rarity]) tag.rarity = rarity
        fishMap.set(record.resultName, tag)
      }
      return {
        rewardUnit,
        records: groupRecords,
        totalAmount: groupRecords.reduce((sum, record) => sum + record.remainingAmount, 0),
        originalAmount: groupRecords.reduce((sum, record) => sum + record.rewardAmount, 0),
        fishTags: Array.from(fishMap.values()).sort((a, b) => rarityRank[b.rarity] - rarityRank[a.rarity] || b.amount - a.amount),
        highestRarity,
        canRedeem: isFactionUnit(rewardUnit),
      }
    }).sort((a, b) => Number(b.canRedeem) - Number(a.canRedeem) || rarityRank[b.highestRarity] - rarityRank[a.highestRarity] || b.totalAmount - a.totalAmount)
  }, [inventoryRecords, isFactionUnit])
  const redeemableGroups = groups.filter(group => group.canRedeem)
  const totalInventoryAmount = inventoryRecords.reduce((sum, record) => sum + record.remainingAmount, 0)
  const redeemableAmount = redeemableGroups.reduce((sum, group) => sum + group.totalAmount, 0)
  const currentPage = Math.floor(recordsOffset / recordsPageSize) + 1
  const totalPages = Math.max(1, Math.ceil(recordsTotal / recordsPageSize))
  const canPrevPage = recordsOffset > 0
  const canNextPage = recordsHasMore

  return (
    <div className="fixed inset-0 z-[9000] flex items-center justify-center bg-slate-950/55 p-3 backdrop-blur-sm">
      <button
        type="button"
        aria-label="关闭钓鱼库存"
        onClick={onClose}
        className="absolute inset-0 cursor-default"
      />
      <div className="relative h-[82vh] w-full max-w-[420px] animate-in fade-in-0 zoom-in-95 duration-200">
        <div className="flex h-full min-h-0 flex-col rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] shadow-[0_18px_50px_rgba(15,23,42,0.12)]">
          <div className="shrink-0 border-b border-[var(--color-border)] px-3.5 py-3">
            <div className="flex items-center justify-between gap-3">
              <div className="flex min-w-0 items-center gap-2">
                <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-xl bg-[var(--color-accent)]/10 text-[var(--color-accent)]">
                  <PackageCheck size={16} />
                </div>
                <div className="min-w-0">
                  <p className="truncate text-sm font-bold leading-4 text-[var(--color-text-primary)]">钓鱼库存</p>
                  <p className="mt-0.5 truncate text-[10px] leading-4 text-[var(--color-text-muted)]">
                    {recordsTotal > 0
                      ? `第 ${currentPage}/${totalPages} 页 · 本页 ${records.length} 条`
                      : '钓到奖励后会存入这里'}
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
              <div className="mt-3 grid grid-cols-2 gap-2">
                <div className="rounded-xl bg-[var(--color-surface-dim)] px-2.5 py-2">
                  <p className="text-[9px] text-[var(--color-text-muted)]">可兑换</p>
                  <p className="mt-0.5 text-sm font-bold text-emerald-600">{redeemableAmount.toLocaleString()}</p>
                </div>
                <div className="rounded-xl bg-[var(--color-surface-dim)] px-2.5 py-2">
                  <p className="text-[9px] text-[var(--color-text-muted)]">暂存</p>
                  <p className="mt-0.5 text-sm font-bold text-amber-600">{(totalInventoryAmount - redeemableAmount).toLocaleString()}</p>
                </div>
              </div>
            )}
          </div>

          <div className="min-h-0 flex-1 overflow-y-auto px-3 py-3">
            {inventoryRecords.length === 0 ? (
              <div className="flex min-h-[180px] flex-col items-center justify-center rounded-xl border border-dashed border-[var(--color-border)] bg-[var(--color-surface-dim)] px-4 text-center">
                <PackageCheck size={24} className="mb-2 text-[var(--color-text-muted)]" />
                <p className="text-xs font-medium text-[var(--color-text-primary)]">暂无库存</p>
                <p className="mt-1 text-[10px] leading-5 text-[var(--color-text-muted)]">
                  钓到兵种奖励后会先存储，可在这里分批兑换。
                </p>
              </div>
            ) : (
              <div className="space-y-2">
                {groups.map(group => {
                  const cfg = RARITY_CONFIG[group.highestRarity] ?? RARITY_CONFIG.common
                  const isRedeeming = redeemingId === group.rewardUnit
                  return (
                    <div key={group.rewardUnit} className={`rounded-xl border px-2.5 py-2.5 ${group.canRedeem ? 'border-emerald-500/20 bg-emerald-500/[0.04]' : 'border-[var(--color-border)] bg-[var(--color-surface-dim)]'}`}>
                      <div className="mb-2 flex flex-wrap gap-1">
                        {group.fishTags.map(tag => {
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

                      {group.canRedeem ? (
                        <div className="mt-2 flex items-center justify-between gap-2">
                          <span className="text-[10px] text-emerald-700">可一键兑换全部库存</span>
                          <button
                            type="button"
                            onClick={() => onRedeemGroup(group.rewardUnit, group.records)}
                            disabled={Boolean(redeemingId)}
                            className="inline-flex min-w-[86px] items-center justify-center gap-1 rounded-lg bg-[var(--color-accent)] px-2.5 py-1.5 text-[10px] font-semibold text-white transition-opacity hover:opacity-90 disabled:opacity-60 cursor-pointer"
                          >
                            {isRedeeming && <Loader2 size={11} className="animate-spin" />}
                            一键兑换
                          </button>
                        </div>
                      ) : (
                        <p className="mt-2 rounded-lg bg-amber-500/10 px-2 py-1.5 text-[10px] leading-5 text-amber-700">
                          非当前阵营兵种，暂时只能存储；驻防增援系统完成后即可兑换。
                        </p>
                      )}
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
                  disabled={recordsLoading || !canPrevPage}
                  className="rounded-lg border border-[var(--color-border)] px-3 py-1.5 text-[10px] font-medium text-[var(--color-text-secondary)] transition-colors hover:border-[var(--color-accent-border)] hover:text-[var(--color-text-primary)] disabled:opacity-50 cursor-pointer"
                >
                  上一页
                </button>
                <span className="text-[10px] text-[var(--color-text-muted)]">
                  {recordsLoading ? '加载中...' : `${currentPage} / ${totalPages}`}
                </span>
                <button
                  type="button"
                  onClick={() => onPageChange(recordsOffset + recordsPageSize)}
                  disabled={recordsLoading || !canNextPage}
                  className="rounded-lg border border-[var(--color-border)] px-3 py-1.5 text-[10px] font-medium text-[var(--color-text-secondary)] transition-colors hover:border-[var(--color-accent-border)] hover:text-[var(--color-text-primary)] disabled:opacity-50 cursor-pointer"
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
