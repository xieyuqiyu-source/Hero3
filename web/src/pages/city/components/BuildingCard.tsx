import { useState, useEffect, useRef, type FC } from 'react'
import { ArrowUpCircle, LoaderCircle, Zap } from 'lucide-react'
import { useGameStore } from '@/store/gameStore'
import { useConfigStore } from '@/store/configStore'
import { gameApi } from '@/api/game'
import { toast } from '@/components/ui'
import ConfirmCityGoldModal from '@/components/ConfirmCityGoldModal'

interface BuildingCardProps {
  buildingId?: string
  icon: React.ReactNode
  name: string
  description: string
  level: number
  production: string
  upgradeEndsAt?: string | null
  color: string
  bgColor: string
  locked?: boolean
}

const TICK_MS = 1000

function getRemainingSeconds(endsAt: string, now = Date.now()): number {
  return Math.max(0, Math.ceil((new Date(endsAt).getTime() - now) / 1000))
}

function formatCountdown(totalSeconds: number): string {
  if (totalSeconds <= 0) return '完成中...'
  const h = Math.floor(totalSeconds / 3600)
  const m = Math.floor((totalSeconds % 3600) / 60)
  const s = totalSeconds % 60
  if (h > 0) return `${h}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
  return `${m}:${String(s).padStart(2, '0')}`
}

const BuildingCard: FC<BuildingCardProps> = ({
  buildingId,
  icon,
  name,
  description,
  level,
  production,
  upgradeEndsAt,
  color,
  bgColor,
  locked,
}) => {
  const [loading, setLoading] = useState(false)
  const [now, setNow] = useState(() => Date.now())
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [instantLoading, setInstantLoading] = useState(false)
  const refreshedUpgradeRef = useRef<string | null>(null)
  const upgrade = useGameStore((s) => s.upgradeBuilding)
  const balance = useConfigStore((s) => s.balance)
  const cityGoldPerSecond = balance?.cityGoldPerSecond ?? 120
  const isUpgrading = upgradeEndsAt != null
  const countdown = upgradeEndsAt ? getRemainingSeconds(upgradeEndsAt, now) : 0
  const instantCost = Math.max(1, Math.ceil(countdown / cityGoldPerSecond))

  useEffect(() => {
    if (!upgradeEndsAt) return

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
    if (!buildingId || loading || isUpgrading) return
    setLoading(true)
    try {
      await upgrade(buildingId)
    } finally {
      setLoading(false)
    }
  }

  const handleInstantComplete = async () => {
    if (!buildingId || instantLoading) return
    setInstantLoading(true)
    try {
      const playerId = useGameStore.getState().activePlayerId
      if (!playerId) return
      const result = await gameApi.instantCompleteBuilding(playerId, buildingId)
      useGameStore.getState().setState(result.state)
      toast.success(`${name} 升级完成`)
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
            {!locked && buildingId && (
              isUpgrading ? (
                <button
                  type="button"
                  onClick={() => setConfirmOpen(true)}
                  className="flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs font-mono font-medium text-amber-500 hover:bg-amber-500/10 cursor-pointer transition-colors"
                  title="点击快速完成"
                >
                  <Zap size={10} />
                  {formatCountdown(countdown)}
                </button>
              ) : (
                <button
                  type="button"
                  onClick={handleUpgrade}
                  disabled={loading}
                  className="
                    flex items-center gap-1 px-2.5 py-1 rounded-lg
                    text-xs font-medium text-[var(--color-accent)]
                    bg-[var(--color-accent-light)] border border-transparent
                    hover:border-[var(--color-accent-border)]
                    cursor-pointer transition-all duration-200
                    disabled:opacity-50 disabled:cursor-not-allowed
                  "
                >
                  {loading ? <LoaderCircle size={12} className="animate-spin" /> : <ArrowUpCircle size={12} />}
                  升级
                </button>
              )
            )}
          </div>
        </div>
      </div>

      {/* 城金加速确认弹窗 */}
      <ConfirmCityGoldModal
        open={confirmOpen}
        onClose={() => setConfirmOpen(false)}
        onConfirm={handleInstantComplete}
        title={`${name} 快速完成`}
        description={`立即完成升级到 Lv.${level + 1}`}
        cost={instantCost}
        loading={instantLoading}
      />
    </div>
  )
}

export default BuildingCard
