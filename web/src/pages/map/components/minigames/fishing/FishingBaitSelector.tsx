import { Anchor, ChevronDown, PackageCheck } from 'lucide-react'
import type { FC } from 'react'
import type { BaitType } from './types'

interface FishingBaitSelectorProps {
  baits: BaitType[]
  selectedBait: BaitType
  showBaitSelect: boolean
  inventoryCount: number
  canChangeBait: boolean
  onToggleBaitSelect: () => void
  onSelectBait: (bait: BaitType) => void
  onOpenInventory: () => void
}

export const FishingBaitSelector: FC<FishingBaitSelectorProps> = ({
  baits,
  selectedBait,
  showBaitSelect,
  inventoryCount,
  canChangeBait,
  onToggleBaitSelect,
  onSelectBait,
  onOpenInventory,
}) => (
  <div className="flex flex-wrap items-center gap-2">
    <div className="relative">
      <button
        type="button"
        onClick={() => canChangeBait && onToggleBaitSelect()}
        className="inline-flex h-9 items-center gap-1.5 rounded-md border-2 border-[var(--color-border)] bg-[var(--color-surface)] px-2.5 text-[11px] transition-colors hover:border-blue-500/30 cursor-pointer"
      >
        <Anchor size={10} className="text-blue-500" />
        <span className="text-[var(--color-text-muted)]">鱼饵</span>
        <span className="font-bold text-[var(--color-text-primary)]">{selectedBait.name}</span>
        <span className="text-[9px] text-amber-600">{selectedBait.cityGoldCost} 城金</span>
        <ChevronDown size={10} className="text-[var(--color-text-muted)]" />
      </button>
      {showBaitSelect && (
        <div className="absolute left-0 top-full z-20 mt-1.5 grid w-[280px] grid-cols-2 gap-1.5 rounded-xl border-2 border-[var(--color-border)] bg-[var(--color-surface)] p-2 shadow-[0_16px_40px_rgba(15,23,42,0.18)]">
          {baits.map((bait) => (
            <button
              key={bait.id}
              type="button"
              onClick={() => onSelectBait(bait)}
              className={`
                flex flex-col items-start rounded-lg border px-2 py-1.5 text-left text-[10px] transition-all cursor-pointer
                ${selectedBait.id === bait.id
                  ? 'border-blue-500/40 bg-blue-500/10 text-blue-600'
                  : 'border-transparent bg-[var(--color-surface-dim)] text-[var(--color-text-secondary)] hover:border-[var(--color-border)]'
                }
              `}
            >
              <span className="font-bold">{bait.tier} · {bait.name}</span>
              <span className="mt-0.5 text-[8px] text-[var(--color-text-muted)]">{bait.cityGoldCost} 城金 · {bait.description}</span>
            </button>
          ))}
        </div>
      )}
    </div>
    <button
      type="button"
      onClick={onOpenInventory}
      className="inline-flex h-9 items-center gap-1.5 rounded-md border-2 border-[var(--color-border)] bg-[var(--color-surface)] px-2.5 text-[11px] text-[var(--color-text-secondary)] transition-colors hover:border-[var(--color-accent-border)] hover:text-[var(--color-text-primary)] cursor-pointer"
    >
      <PackageCheck size={11} className="text-[var(--color-accent)]" />
      钓鱼记录
      {inventoryCount > 0 && (
        <span className="rounded-full bg-[var(--color-accent)] px-1.5 py-0.5 text-[9px] font-bold text-white">
          {inventoryCount}
        </span>
      )}
    </button>
  </div>
)
