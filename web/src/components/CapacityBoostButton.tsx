import { useState, useRef, useEffect, type FC } from 'react'
import { Expand, Coins, Clock, Circle } from 'lucide-react'
import { gameApi } from '@/api/game'
import { useGameStore } from '@/store/gameStore'
import { useConfigStore } from '@/store/configStore'
import { useConfirmPreferenceStore } from '@/store/confirmPreferenceStore'
import { toast } from '@/components/ui'
import ConfirmCityGoldModal from './ConfirmCityGoldModal'

const BOOST_MULTIPLIERS = [2, 4, 8, 16] as const
const BOOST_DURATIONS = [
  { label: '1h', hours: 1 },
  { label: '6h', hours: 6 },
  { label: '12h', hours: 12 },
  { label: '24h', hours: 24 },
] as const

interface CapacityBoostButtonProps {
  currentBoost?: number
}

const CapacityBoostButton: FC<CapacityBoostButtonProps> = ({ currentBoost = 1 }) => {
  const [open, setOpen] = useState(false)
  const [selectedMultiplier, setSelectedMultiplier] = useState<number>(2)
  const [loading, setLoading] = useState(false)
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [pendingHours, setPendingHours] = useState(0)
  const [now, setNow] = useState(Date.now())
  const containerRef = useRef<HTMLDivElement>(null)
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const setState = useGameStore((s) => s.setState)
  const balance = useConfigStore((s) => s.balance)
  const skipConfirmations = useConfirmPreferenceStore((s) => s.skipConfirmations)
  const boostEnd = useGameStore((s) => s.state?.capacityBoostEnd)

  const isActive = currentBoost > 1

  useEffect(() => {
    if (!isActive || !boostEnd) return
    const timer = setInterval(() => setNow(Date.now()), 1000)
    return () => clearInterval(timer)
  }, [isActive, boostEnd])

  const remainingSeconds = (() => {
    if (!isActive || !boostEnd) return 0
    return Math.max(0, Math.floor((new Date(boostEnd).getTime() - now) / 1000))
  })()

  const formatRemaining = (s: number) => {
    if (s <= 0) return '已到期'
    const h = Math.floor(s / 3600)
    const m = Math.floor((s % 3600) / 60)
    const sec = s % 60
    if (h > 0) return `${h}:${String(m).padStart(2, '0')}:${String(sec).padStart(2, '0')}`
    return `${m}:${String(sec).padStart(2, '0')}`
  }

  const calcPrice = (multiplier: number, hours: number): number => {
    const baseCost = balance?.boostBaseCost ?? 30
    const mf = balance?.boostMultiplierFactor?.[String(multiplier)] ?? { '2': 1, '4': 3, '8': 8, '16': 20 }[String(multiplier)] ?? 1
    const df = balance?.boostDurationFactor?.[String(hours)] ?? { '1': 1, '6': 5, '12': 9, '24': 16 }[String(hours)] ?? 1
    return baseCost * mf * df
  }

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

  const handleSelectDuration = (hours: number) => {
    setPendingHours(hours)
    if (skipConfirmations) {
      handleConfirmPurchase(hours)
      return
    }
    setConfirmOpen(true)
  }

  const handleConfirmPurchase = async (hours = pendingHours) => {
    if (!activePlayerId || loading) return
    setLoading(true)
    try {
      const result = await gameApi.purchaseCapacityBoost(activePlayerId, selectedMultiplier, hours)
      setState(result.state)
      toast.success(`仓库 ×${selectedMultiplier} 扩容已激活`)
      setOpen(false)
      setConfirmOpen(false)
    } catch (e: any) {
      const msg = e?.message || '购买失败'
      if (msg.includes('still active')) toast.error('当前扩容尚未到期')
      else if (msg.includes('insufficient')) toast.error('城金不足')
      else toast.error(msg)
    } finally {
      setLoading(false)
    }
  }

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
            ? 'bg-indigo-500/20 text-indigo-400 border border-indigo-500/40'
            : 'bg-[var(--color-surface-dim)] text-[var(--color-text-muted)] border border-[var(--color-border)] hover:text-indigo-400 hover:border-indigo-500/40'
          }
        `}
      >
        <Expand size={11} />
        {isActive ? `×${currentBoost}` : '扩容'}
      </button>

      {/* Popover */}
      <div
        className={`
          absolute right-0 top-full mt-2 z-[100]
          w-44 rounded-xl
          bg-slate-900/95 backdrop-blur-md border border-slate-700/50
          shadow-[0_12px_32px_rgba(0,0,0,0.3)]
          transition-all duration-200 origin-top-right
          ${open ? 'opacity-100 scale-100' : 'opacity-0 scale-90 pointer-events-none'}
        `}
      >
        {/* Header + countdown */}
        <div className="px-3 pt-2.5 pb-1.5">
          {isActive ? (
            <div className="flex items-center gap-1.5 px-2 py-1.5 rounded-lg bg-indigo-500/10 border border-indigo-500/20">
              <Clock size={10} className="text-indigo-400" />
              <span className="text-[10px] text-indigo-300 font-mono font-bold">{formatRemaining(remainingSeconds)}</span>
              <span className="text-[9px] text-indigo-300/60 ml-auto">×{currentBoost}</span>
            </div>
          ) : (
            <p className="text-[10px] font-semibold text-indigo-300">仓库扩容</p>
          )}
        </div>

        {/* Multiplier dots */}
        <div className="px-3 pb-1.5">
          <div className="flex items-center justify-between">
            {BOOST_MULTIPLIERS.map((m) => (
              <button
                key={m}
                type="button"
                onClick={() => setSelectedMultiplier(m)}
                disabled={isActive}
                className={`
                  flex items-center gap-0.5 px-1.5 py-1 rounded-md cursor-pointer
                  transition-all duration-150
                  ${selectedMultiplier === m
                    ? 'text-indigo-400'
                    : 'text-white/40 hover:text-indigo-300'
                  }
                  disabled:opacity-40 disabled:cursor-not-allowed
                `}
              >
                <Circle size={6} className={selectedMultiplier === m ? 'fill-indigo-400' : ''} />
                <span className="text-[10px] font-bold">×{m}</span>
              </button>
            ))}
          </div>
        </div>

        {/* Duration grid 2x2 */}
        <div className="px-3 pb-2.5">
          <div className="grid grid-cols-2 gap-1">
            {BOOST_DURATIONS.map((d) => (
              <button
                key={d.hours}
                type="button"
                onClick={() => handleSelectDuration(d.hours)}
                disabled={isActive || loading}
                className="
                  flex items-center justify-between px-2 py-1.5 rounded-lg
                  bg-white/5 border border-white/10
                  hover:bg-indigo-500/10 hover:border-indigo-500/30
                  cursor-pointer transition-all duration-150
                  disabled:opacity-40 disabled:cursor-not-allowed
                "
              >
                <span className="text-[10px] text-white/70 font-medium">{d.label}</span>
                <span className="flex items-center gap-0.5 text-[9px] text-indigo-400 font-bold">
                  <Coins size={8} />
                  {calcPrice(selectedMultiplier, d.hours)}
                </span>
              </button>
            ))}
          </div>
        </div>
      </div>

      <ConfirmCityGoldModal
        open={confirmOpen}
        onClose={() => setConfirmOpen(false)}
        onConfirm={() => handleConfirmPurchase()}
        title="购买仓库扩容"
        description={`仓库容量 ×${selectedMultiplier}，持续 ${pendingHours} 小时`}
        cost={calcPrice(selectedMultiplier, pendingHours)}
        loading={loading}
      />
    </div>
  )
}

export default CapacityBoostButton
