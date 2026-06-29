import { useState, useRef, useEffect, useMemo, type FC } from 'react'
import { Expand, Coins, Clock, Circle } from 'lucide-react'
import { gameApi } from '@/api/game'
import { useGameStore } from '@/store/gameStore'
import { useConfigStore } from '@/store/configStore'
import { useConfirmPreferenceStore } from '@/store/confirmPreferenceStore'
import { toast } from '@/components/ui'
import { getErrorMessage } from '@/utils/error'
import ConfirmCityGoldModal from './ConfirmCityGoldModal'

const DEFAULT_BOOST_MULTIPLIERS = [2, 4, 8, 16]
const DEFAULT_BOOST_DURATIONS = [1, 6, 12, 24]

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
  const patchResourceAction = useGameStore((s) => s.patchResourceAction)
  const balance = useConfigStore((s) => s.balance)
  const skipConfirmations = useConfirmPreferenceStore((s) => s.skipConfirmations)
  const boostEnd = useGameStore((s) => s.state?.capacityBoostEnd)
  const multiplierOptions = useMemo(
    () => getNumberOptions(balance?.boostMultiplierFactor, DEFAULT_BOOST_MULTIPLIERS),
    [balance?.boostMultiplierFactor],
  )
  const durationOptions = useMemo(
    () => getNumberOptions(balance?.boostDurationFactor, DEFAULT_BOOST_DURATIONS),
    [balance?.boostDurationFactor],
  )

  const isActive = currentBoost > 1
  const isSameMultiplierRenewal = isActive && currentBoost === selectedMultiplier

  useEffect(() => {
    if (multiplierOptions.length > 0 && !multiplierOptions.includes(selectedMultiplier)) {
      setSelectedMultiplier(multiplierOptions[0])
    }
  }, [multiplierOptions, selectedMultiplier])

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
      patchResourceAction(result)
      toast.success(isSameMultiplierRenewal ? `仓库 ×${selectedMultiplier} 已续时` : `仓库 ×${selectedMultiplier} 扩容已激活`)
      setOpen(false)
      setConfirmOpen(false)
    } catch (e: unknown) {
      const msg = getErrorMessage(e, '购买失败')
      if (msg.includes('insufficient')) toast.error('城金不足')
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
            {multiplierOptions.map((m) => (
              <button
                key={m}
                type="button"
                onClick={() => setSelectedMultiplier(m)}
                className={`
                  flex items-center gap-0.5 px-1.5 py-1 rounded-md cursor-pointer
                  transition-all duration-150
                  ${selectedMultiplier === m
                    ? 'text-indigo-400'
                    : 'text-white/40 hover:text-indigo-300'
                  }
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
            {durationOptions.map((hours) => (
              <button
                key={hours}
                type="button"
                onClick={() => handleSelectDuration(hours)}
                disabled={loading}
                className="
                  flex items-center justify-between px-2 py-1.5 rounded-lg
                  bg-white/5 border border-white/10
                  hover:bg-indigo-500/10 hover:border-indigo-500/30
                  cursor-pointer transition-all duration-150
                  disabled:opacity-40 disabled:cursor-not-allowed
                "
              >
                <span className="text-[10px] text-white/70 font-medium">{formatHoursLabel(hours)}</span>
                <span className="flex items-center gap-0.5 text-[9px] text-indigo-400 font-bold">
                  <Coins size={8} />
                  {calcPrice(selectedMultiplier, hours)}
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
        title={isSameMultiplierRenewal ? '续订仓库扩容' : '购买仓库扩容'}
        description={isSameMultiplierRenewal ? `仓库容量保持 ×${selectedMultiplier}，时间延长 ${pendingHours} 小时` : `仓库容量切换为 ×${selectedMultiplier}，持续 ${pendingHours} 小时`}
        cost={calcPrice(selectedMultiplier, pendingHours)}
        loading={loading}
      />
    </div>
  )
}

function getNumberOptions(source: Record<string, number> | undefined, fallback: number[]) {
  const values = source
    ? Object.keys(source).map((key) => Number(key)).filter((value) => Number.isFinite(value) && value > 0)
    : []
  return [...(values.length > 0 ? values : fallback)].sort((a, b) => a - b)
}

function formatHoursLabel(hours: number) {
  return `${hours}h`
}

export default CapacityBoostButton
