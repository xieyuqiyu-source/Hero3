import { Loader2, PackageCheck, X } from 'lucide-react'
import type { FC } from 'react'
import type { MiniGameRecord } from '@/types/game'
import { RARITY_CONFIG } from './fishingConfig'
import type { FishCatch } from './types'

interface FishingInventoryModalProps {
  records: MiniGameRecord[]
  recordsLoading: boolean
  redeemAmounts: Record<string, number>
  redeemingId: string
  isFactionUnit: (unitName: string) => boolean
  onClose: () => void
  onRefresh: () => void
  onRedeem: (record: MiniGameRecord) => void
  onChangeRedeemAmount: (recordId: string, amount: number) => void
}

export const FishingInventoryModal: FC<FishingInventoryModalProps> = ({
  records,
  recordsLoading,
  redeemAmounts,
  redeemingId,
  isFactionUnit,
  onClose,
  onRefresh,
  onRedeem,
  onChangeRedeemAmount,
}) => {
  const inventoryRecords = records.filter(record => record.remainingAmount > 0)
  const redeemableRecords = inventoryRecords.filter(record => isFactionUnit(record.rewardUnit))
  const totalInventoryAmount = inventoryRecords.reduce((sum, record) => sum + record.remainingAmount, 0)

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
                    {inventoryRecords.length > 0
                      ? `${inventoryRecords.length} 条库存 · ${totalInventoryAmount.toLocaleString()} 兵`
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
                  <p className="mt-0.5 text-sm font-bold text-emerald-600">{redeemableRecords.length}</p>
                </div>
                <div className="rounded-xl bg-[var(--color-surface-dim)] px-2.5 py-2">
                  <p className="text-[9px] text-[var(--color-text-muted)]">暂存</p>
                  <p className="mt-0.5 text-sm font-bold text-amber-600">{inventoryRecords.length - redeemableRecords.length}</p>
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
                {inventoryRecords.map(record => {
                  const cfg = RARITY_CONFIG[record.rarity as FishCatch['rarity']] ?? RARITY_CONFIG.common
                  const canRedeem = isFactionUnit(record.rewardUnit)
                  const amount = redeemAmounts[record.id] ?? record.remainingAmount
                  return (
                    <div key={record.id} className={`rounded-xl border px-2.5 py-2.5 ${canRedeem ? 'border-emerald-500/20 bg-emerald-500/[0.04]' : 'border-[var(--color-border)] bg-[var(--color-surface-dim)]'}`}>
                      <div className="flex min-w-0 items-start gap-2">
                        <span className={`shrink-0 rounded-md px-1.5 py-0.5 text-[9px] font-bold ${cfg.bg} ${cfg.color}`}>
                          {cfg.label}
                        </span>
                        <div className="min-w-0 flex-1">
                          <p className="truncate text-xs font-semibold text-[var(--color-text-primary)]">
                            {record.resultName} · {record.rewardUnit}
                          </p>
                          <p className="mt-0.5 truncate text-[10px] text-[var(--color-text-muted)]">
                            剩 {record.remainingAmount.toLocaleString()}
                          </p>
                        </div>
                      </div>

                      {canRedeem ? (
                        <div className="mt-2 grid grid-cols-[minmax(0,1fr)_auto] gap-2">
                          <input
                            type="number"
                            min={1}
                            max={record.remainingAmount}
                            value={amount}
                            onChange={(event) => {
                              const next = Math.max(1, Math.min(record.remainingAmount, Number(event.target.value) || 1))
                              onChangeRedeemAmount(record.id, next)
                            }}
                            className="min-w-0 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 py-1.5 text-xs outline-none focus:border-[var(--color-accent-border)]"
                          />
                          <button
                            type="button"
                            onClick={() => onRedeem(record)}
                            disabled={redeemingId === record.id}
                            className="inline-flex min-w-[58px] items-center justify-center gap-1 rounded-lg bg-[var(--color-accent)] px-2.5 py-1.5 text-[10px] font-semibold text-white transition-opacity hover:opacity-90 disabled:opacity-60 cursor-pointer"
                          >
                            {redeemingId === record.id && <Loader2 size={11} className="animate-spin" />}
                            兑换
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
        </div>
      </div>
    </div>
  )
}
