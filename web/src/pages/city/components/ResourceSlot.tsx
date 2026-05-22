import { type FC } from 'react'
import { ArrowUpCircle } from 'lucide-react'

interface ResourceSlotProps {
  index: number
  level: number
  production: number
  color: string
  bgColor: string
}

const ResourceSlot: FC<ResourceSlotProps> = ({ index, level, production, color, bgColor }) => {
  return (
    <div className="flex items-center justify-between px-3 py-2 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] hover:shadow-[0_2px_8px_rgba(15,23,42,0.04)] transition-shadow duration-150">
      <div className="flex items-center gap-2">
        <div className={`w-5 h-5 rounded-md ${bgColor} flex items-center justify-center`}>
          <span className={`text-[10px] font-bold ${color}`}>{index}</span>
        </div>
        <span className="text-xs text-[var(--color-text-primary)]">Lv.{level}</span>
        <span className="text-[10px] text-amber-500 font-medium">+{production}</span>
      </div>
      <button
        type="button"
        className="
          flex items-center gap-1 px-2 py-1 rounded-lg
          text-[10px] font-medium text-[var(--color-accent)]
          bg-[var(--color-accent-light)]
          hover:border-[var(--color-accent-border)]
          cursor-pointer transition-all duration-200
        "
      >
        <ArrowUpCircle size={10} />
        升级
      </button>
    </div>
  )
}

export default ResourceSlot
