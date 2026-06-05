import { useState, useEffect, useRef, type FC } from 'react'
import { ArrowUpCircle, LoaderCircle, Zap } from 'lucide-react'
import { useGameStore } from '@/store/gameStore'
import { useConfigStore } from '@/store/configStore'
import { useConfirmPreferenceStore } from '@/store/confirmPreferenceStore'
import { Tooltip, toast } from '@/components/ui'
import { gameApi } from '@/api/game'
import ConfirmCityGoldModal from '@/components/ConfirmCityGoldModal'
import {
  getProductionAtLevel,
  getUpgradeCost,
  getUpgradeSeconds,
  formatDuration,
} from '@/store/configStore'

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

const TICK_MS = 1000

/** 计算剩余秒数 */
function getRemainingSeconds(endsAt: string, now = Date.now()): number {
  const end = new Date(endsAt).getTime()
  return Math.max(0, Math.ceil((end - now) / 1000))
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
  const [now, setNow] = useState(() => Date.now())
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [instantLoading, setInstantLoading] = useState(false)
  const refreshedUpgradeRef = useRef<string | null>(null)
  const upgrade = useGameStore((s) => s.upgradeBuilding)
  const balance = useConfigStore((s) => s.balance)
  const skipConfirmations = useConfirmPreferenceStore((s) => s.skipConfirmations)
  const cityGoldPerSecond = balance?.cityGoldPerSecond ?? 120
  const isUpgrading = upgradeEndsAt !== null
  const countdown = upgradeEndsAt ? getRemainingSeconds(upgradeEndsAt, now) : 0
  const instantCost = Math.max(1, Math.ceil(countdown / cityGoldPerSecond))

  // 倒计时 tick
  useEffect(() => {
    if (!upgradeEndsAt) return

    // 立即刷新 now，避免用旧时间算出错误的倒计时
    setNow(Date.now())

    const timer = window.setInterval(() => {
      setNow(Date.now())
    }, TICK_MS)

    return () => clearInterval(timer)
  }, [upgradeEndsAt])

  useEffect(() => {
    if (!upgradeEndsAt) {
      refreshedUpgradeRef.current = null
      return
    }
    if (countdown > 0 || refreshedUpgradeRef.current === upgradeEndsAt) return

    refreshedUpgradeRef.current = upgradeEndsAt
    useGameStore.getState().loadGameState()
  }, [upgradeEndsAt, countdown])

  const handleUpgrade = async () => {
    if (isUpgrading || loading) return
    setLoading(true)
    try {
      await upgrade(buildingId)
    } finally {
      setLoading(false)
    }
  }

  const handleInstantComplete = async () => {
    if (instantLoading) return
    setInstantLoading(true)
    try {
      const playerId = useGameStore.getState().activePlayerId
      if (!playerId) return
      const result = await gameApi.instantCompleteBuilding(playerId, buildingId)
      useGameStore.getState().setState(result.state)
    } catch (e: any) {
      const msg = e?.message || '加速失败'
      if (msg.includes('insufficient')) {
        toast.error('城金不足')
      } else {
        toast.error(msg)
      }
    } finally {
      setInstantLoading(false)
      setConfirmOpen(false)
    }
  }

  // 升级信息
  const upgradeCost = getUpgradeCost(buildingType, level)

  // 检查资源是否足够
  const resources = useGameStore((s) => s.state?.resources.items ?? {})
  const canAfford = upgradeCost
    ? Object.entries(upgradeCost).every(([res, cost]) => (resources[res] ?? 0) >= cost)
    : false
  const nextProduction = getProductionAtLevel(buildingType, level + 1)
  const upgradeTime = getUpgradeSeconds(buildingType, level)
  const productionGain = nextProduction - production

  const tooltipContent = upgradeCost ? (
    <div className="space-y-1.5 text-[11px] min-w-[140px]">
      <p className="font-semibold text-white">升级到 Lv.{level + 1}</p>
      <div className="space-y-0.5">
        {Object.entries(upgradeCost).map(([res, cost]) => {
          const have = resources[res] ?? 0
          const enough = have >= cost
          const label: Record<string, string> = { wood: '木材', stone: '石料', iron: '铁矿', food: '粮食' }
          return (
            <p key={res} className={enough ? 'text-white/70' : 'text-red-400'}>
              {label[res] ?? res} {cost.toLocaleString()}
            </p>
          )
        })}
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
        <button
          type="button"
          onClick={() => {
            if (skipConfirmations) handleInstantComplete()
            else setConfirmOpen(true)
          }}
          className="flex items-center gap-1 px-2 py-1 rounded-lg text-[10px] font-mono font-medium text-amber-500 hover:bg-amber-500/10 cursor-pointer transition-colors"
          title="点击快速完成"
        >
          <Zap size={9} />
          {formatCountdown(countdown)}
        </button>
      ) : (
        <Tooltip content={tooltipContent} placement="top">
          <button
            type="button"
            onClick={handleUpgrade}
            disabled={loading || !upgradeCost || !canAfford}
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

      {/* 城金加速确认弹窗 */}
      <ConfirmCityGoldModal
        open={confirmOpen}
        onClose={() => setConfirmOpen(false)}
        onConfirm={handleInstantComplete}
        title="快速完成升级"
        description={`立即完成升级到 Lv.${level + 1}`}
        cost={instantCost}
        loading={instantLoading}
      />
    </div>
  )
}

export default ResourceSlot
