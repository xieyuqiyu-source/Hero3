import { type FC } from 'react'
import { ArrowUpCircle } from 'lucide-react'

interface BuildingCardProps {
  icon: React.ReactNode
  name: string
  description: string
  level: number
  production: string
  color: string
  bgColor: string
  locked?: boolean
}

const BuildingCard: FC<BuildingCardProps> = ({
  icon,
  name,
  description,
  level,
  production,
  color,
  bgColor,
  locked,
}) => {
  return (
    <div className={`
      relative rounded-2xl p-4 border border-[var(--color-border)]
      bg-[var(--color-surface)]
      transition-all duration-200
      ${locked ? 'opacity-60' : 'hover:shadow-[0_8px_24px_rgba(15,23,42,0.06)] hover:-translate-y-0.5'}
    `}>
      <div className="flex items-start gap-3">
        <div className={`w-10 h-10 rounded-xl ${bgColor} flex items-center justify-center ${color} flex-shrink-0`}>
          {icon}
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold text-[var(--color-text-primary)]">{name}</h3>
            <span className="text-xs text-[var(--color-text-muted)]">
              {locked ? '未解锁' : `Lv.${level}`}
            </span>
          </div>
          <p className="text-xs text-[var(--color-text-secondary)] mt-0.5">{description}</p>
          <div className="flex items-center justify-between mt-2">
            <span className={`text-xs font-medium ${locked ? 'text-[var(--color-text-muted)]' : color}`}>
              {production}
            </span>
            {!locked && (
              <button
                type="button"
                className="
                  flex items-center gap-1 px-2.5 py-1 rounded-lg
                  text-xs font-medium text-[var(--color-accent)]
                  bg-[var(--color-accent-light)] border border-transparent
                  hover:border-[var(--color-accent-border)]
                  cursor-pointer transition-all duration-200
                "
              >
                <ArrowUpCircle size={12} />
                升级
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

export default BuildingCard
