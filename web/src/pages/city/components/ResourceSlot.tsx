import { useState, useEffect, type FC } from 'react'
import { ArrowUpCircle, LoaderCircle } from 'lucide-react'
import { useGameStore } from '@/store/gameStore'
import { Tooltip } from '@/components/ui'
import {
  getProductionAtLevel,
  getUpgradeCost,
  getUpgradeSeconds,
  formatCost,
  formatDuration,
} from '../data/buildingConfig'

interface ResourceSlotProps {
  buildingId: string
  buildingType: string
  index: number
  level: number
  production: number
  upgradeEndsAt: string | null
  color: string
  bgColor: string
}

/** 计算剩余秒数 */
function getRemainingSeconds(endsAt: string): number {
  const end = new Date(endsAt).getTime()
  return Math.max(0, Math.ceil((end - Date.now()) / 1000))
}

/** 格式化倒计时 mm:ss 或 hh:mm:ss */
function formatCountdown(totalSeconds: number): string {
  if (totalSeconds <= 0) return '完成中...'
  const h = Math.floor(totalSeconds / 3600)
  const m = Math.floor((totalSeconds % 3600) / 60)
  const s = totalSeconds % 60
  if (h > 0) return `${h}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
  return `${m}:${String(s).padStart(2, '0')}`
}

const ResourceSlot: FC<ResourceSlotProps> = ({
  buildingId,
  buildingType,
  index,
  level,
  production,
  upgradeEndsAt,
  color,
  bgColor,
}) => {
  const [loading, setLoading] = useState(false)
  const [countdown, setCountdown] = useState(() =>
    upgradeEndsAt ? getRemainingSeconds(upgradeEndsAt) : 0
  )
  const upgrade = useGameStore((s) => s.upgradeBuilding)
  const isUpgrading = upgradeEndsAt !== null

  // 倒计时 tick
  useEffect(() => {
    if (!upgradeEndsAt) {
      setCountdown(0)
      return
    }
    setCountdown(getRemainingSeconds(upgradeEndsAt))
    const timer = window.setInterval(() => {
      const remaining = getRemainingSeconds(upgradeEndsAt)
      setCountdown(remaining)
      if (remaining <= 0) {
        clearInterval(timer)
        // 倒计时归零，刷新状态
        useGameStore.getState().loadGameState()
      }
    }, 1000)
    return () => clearInterval(timer)
  }, [upgradeEndsAt])

  const handleUpgrade = async () => {
    if (isUpgrading || loading) return
    setLoading(true)
    try {
      await upgrade(buildingId)
    } finally {
      setLoading(false)
    }
  }

  // 升级信息
  const upgradeCost = getUpgradeCost(buildingType, level)
  const nextProduction = getProductionAtLevel(buildingType, level + 1)
  const upgradeTime = getUpgradeSeconds(buildingType, level)
  const productionGain = nextProduction - production

  const tooltipContent = upgradeCost ? (
    <div className="space-y-1.5 text-[11px] min-w-[140px]">
      <p className="font-semibold text-white">升级到 Lv.{level + 1}</p>
      <div className="space-y-0.5 text-white/70">
        <p>{formatCost(upgradeCost)}</p>
      </div>
      <div className="pt-1 border-t border-white/10 space-y-0.5">
        <p className="text-amber-300">产量 +{production} → +{nextProduction} <span className="text-green-400">(+{productionGain})</span></p>
        <p className="text-white/50">耗时 {formatDuration(upgradeTime)}</p>
      </div>
    </div>
  ) : (
    <p className="text-[11px] text-white/70">已达最高等级</p>
  )

  return (
    <div className="flex items-center justify-between px-3 py-2 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] hover:shadow-[0_2px_8px_rgba(15,23,42,0.04)] transition-shadow duration-150">
      <div className="flex items-center gap-2">
        <div className={`w-5 h-5 rounded-md ${bgColor} flex items-center justify-center`}>
          <span className={`text-[10px] font-bold ${color}`}>{index}</span>
        </div>
        <span className="text-xs text-[var(--color-text-primary)]">Lv.{level}</span>
        <span className="text-[10px] text-amber-500 font-medium">+{production}</span>
      </div>

      {isUpgrading ? (
        <span className="flex items-center gap-1 px-2 py-1 text-[10px] font-mono font-medium text-amber-500">
          <LoaderCircle size={10} className="animate-spin" />
          {formatCountdown(countdown)}
        </span>
      ) : (
        <Tooltip content={tooltipContent} placement="left">
          <button
            type="button"
            onClick={handleUpgrade}
            disabled={loading || !upgradeCost}
            className="
              flex items-center gap-1 px-2 py-1 rounded-lg
              text-[10px] font-medium text-[var(--color-accent)]
              bg-[var(--color-accent-light)]
              hover:border-[var(--color-accent-border)]
              cursor-pointer transition-all duration-200
              disabled:opacity-50 disabled:cursor-not-allowed
            "
          >
            {loading ? <LoaderCircle size={10} className="animate-spin" /> : <ArrowUpCircle size={10} />}
            升级
          </button>
        </Tooltip>
      )}
    </div>
  )
}

export default ResourceSlot
