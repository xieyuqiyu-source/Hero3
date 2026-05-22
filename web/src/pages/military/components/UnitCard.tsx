import { type FC } from 'react'
import { Clock, Swords, Shield, Zap, Package, Wheat, TreePine, Mountain, Gem } from 'lucide-react'
import type { UnitConfig } from '@/store/configStore'

interface UnitCardProps {
  unitId: string
  config: UnitConfig
  owned: number
  onClick: () => void
}

const UnitCard: FC<UnitCardProps> = ({ config, owned, onClick }) => {
  return (
    <div
      className="
        flex flex-col rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)]
        overflow-hidden transition-all duration-200
        hover:border-[var(--color-accent-border)] hover:shadow-[0_6px_20px_rgba(15,23,42,0.06)]
      "
    >
      {/* Header: name + owned */}
      <div className="px-3 pt-3 pb-1.5">
        <div className="flex items-baseline justify-between">
          <span className="text-sm font-bold text-[var(--color-text-primary)]">{config.name}</span>
          <span className="text-base font-bold text-[var(--color-accent)]">{owned}</span>
        </div>
        <p className="text-[10px] text-[var(--color-text-muted)] mt-0.5 leading-tight">{config.description}</p>
      </div>

      {/* Body: stats left + cost right */}
      <div className="flex flex-1 px-3 py-2 gap-3">
        {/* Left: stats list */}
        <div className="flex-1 flex flex-col justify-between">
          {[
            [Swords, '攻击', config.stats.attack],
            [Shield, '步防', config.stats.infantryDefense],
            [Shield, '骑防', config.stats.cavalryDefense],
            [Zap, '速度', config.stats.speed],
            [Package, '运载', config.stats.carryCapacity],
          ].map(([Icon, label, val]) => {
            const IconComp = Icon as FC<{ size?: number; className?: string }>
            return (
              <div key={label as string} className="flex items-center justify-between py-[3px] border-b border-[var(--color-border)] last:border-b-0">
                <span className="flex items-center gap-1 text-[10px] text-[var(--color-text-secondary)]">
                  <IconComp size={10} className="text-[var(--color-text-muted)]" />
                  {label as string}
                </span>
                <span className="text-[11px] font-bold text-[var(--color-text-primary)]">{(val as number) ?? 0}</span>
              </div>
            )
          })}
        </div>

        {/* Right: cost table */}
        <div className="flex-1 flex flex-col justify-between">
          {[
            [TreePine, '木', config.cost.wood],
            [Mountain, '石', config.cost.stone],
            [Gem, '铁', config.cost.iron],
            [Wheat, '粮', config.cost.food],
            [Wheat, '口粮', config.stats.upkeep],
          ].map(([Icon, label, val]) => {
            const IconComp = Icon as FC<{ size?: number; className?: string }>
            return (
              <div key={label as string} className="flex items-center justify-between py-[3px] border-b border-[var(--color-border)] last:border-b-0">
                <span className="flex items-center gap-1 text-[10px] text-amber-400">
                  <IconComp size={10} />
                  {label as string}
                </span>
                <span className="text-[11px] font-bold text-amber-400">{(val as number) ?? 0}</span>
              </div>
            )
          })}
        </div>
      </div>

      {/* Footer: recruit button with time */}
      <div className="px-3 pb-3 pt-1">
        <button
          type="button"
          onClick={(e) => { e.stopPropagation(); onClick() }}
          className="
            w-full flex items-center justify-center gap-2 py-2 rounded-xl text-xs font-bold
            bg-[var(--color-accent)] text-white
            hover:opacity-90 cursor-pointer transition-opacity
          "
        >
          征 募
          <span className="flex items-center gap-0.5 text-amber-200 font-semibold">
            <Clock size={10} />
            {config.trainSeconds}s
          </span>
        </button>
      </div>
    </div>
  )
}

export default UnitCard
