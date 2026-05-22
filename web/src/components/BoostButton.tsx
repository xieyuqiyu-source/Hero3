import { useState, useRef, useEffect, type FC } from 'react'
import { Zap, Coins } from 'lucide-react'

const BOOST_MULTIPLIERS = [2, 4, 8, 16] as const
const BOOST_DURATIONS = [
  { label: '1 小时', hours: 1 },
  { label: '6 小时', hours: 6 },
  { label: '12 小时', hours: 12 },
  { label: '24 小时', hours: 24 },
] as const

interface BoostButtonProps {
  currentBoost?: number
}

const BoostButton: FC<BoostButtonProps> = ({ currentBoost = 1 }) => {
  const [open, setOpen] = useState(false)
  const [selectedMultiplier, setSelectedMultiplier] = useState<number>(2)
  const containerRef = useRef<HTMLDivElement>(null)

  // 点击外部关闭
  useEffect(() => {
    if (!open) return
    const handleClick = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [open])

  const handlePurchase = (hours: number) => {
    // TODO: 调用后端加成接口
    console.log('boost:', selectedMultiplier, 'x for', hours, 'hours')
    setOpen(false)
  }

  const isActive = currentBoost > 1

  return (
    <div ref={containerRef} className="relative">
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className={`
          flex items-center gap-1 px-2 py-1 rounded-lg
          text-[10px] font-bold cursor-pointer
          transition-all duration-200
          ${isActive
            ? 'bg-amber-500/20 text-amber-400 border border-amber-500/40'
            : 'bg-[var(--color-surface-dim)] text-[var(--color-text-muted)] border border-[var(--color-border)] hover:text-amber-400 hover:border-amber-500/40'
          }
        `}
      >
        <Zap size={11} />
        {isActive ? `×${currentBoost}` : '加成'}
      </button>

      {/* Popover */}
      <div
        className={`
          absolute right-0 top-full mt-2 z-[100]
          w-52 p-3 rounded-xl
          bg-slate-900/95 backdrop-blur-md border border-slate-700/50
          shadow-[0_12px_32px_rgba(0,0,0,0.3)]
          transition-all duration-200 origin-top-right
          ${open ? 'opacity-100 scale-100' : 'opacity-0 scale-90 pointer-events-none'}
        `}
      >
        <p className="text-[11px] font-semibold text-amber-300 mb-2.5">产量加成</p>

        {/* Multiplier row */}
        <div className="flex gap-1.5 mb-3">
          {BOOST_MULTIPLIERS.map((m) => (
            <button
              key={m}
              type="button"
              onClick={() => setSelectedMultiplier(m)}
              className={`
                flex-1 py-1.5 rounded-lg text-xs font-bold cursor-pointer
                transition-all duration-150 text-center
                ${selectedMultiplier === m
                  ? 'bg-amber-500 text-white shadow-[0_4px_12px_rgba(245,158,11,0.3)]'
                  : 'bg-white/10 text-white/70 hover:bg-amber-500/20 hover:text-amber-300'
                }
              `}
            >
              ×{m}
            </button>
          ))}
        </div>

        {/* Duration options */}
        <div className="space-y-1.5">
          {BOOST_DURATIONS.map((d) => (
            <button
              key={d.hours}
              type="button"
              onClick={() => handlePurchase(d.hours)}
              className="
                w-full flex items-center justify-between px-3 py-2 rounded-lg
                bg-white/5 border border-white/10
                hover:bg-amber-500/10 hover:border-amber-500/30
                cursor-pointer transition-all duration-150
              "
            >
              <span className="text-[11px] text-white/80 font-medium">{d.label}</span>
              <span className="flex items-center gap-1 text-[11px] text-amber-400 font-semibold">
                <Coins size={10} />
                --
              </span>
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}

export default BoostButton
