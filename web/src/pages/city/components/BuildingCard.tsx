import { useState, useEffect, useRef, type FC } from 'react'
import { ArrowUpCircle, LoaderCircle, Zap } from 'lucide-react'
import { useGameStore } from '@/store/gameStore'
import { useConfigStore, getGoldUpgradeCost, getUpgradeCost, getUpgradeSeconds, formatDuration } from '@/store/configStore'
import { useAccountStore } from '@/store/accountStore'
import { useConfirmPreferenceStore } from '@/store/confirmPreferenceStore'
import { gameApi } from '@/api/game'
import { Tooltip, toast } from '@/components/ui'
import { getErrorMessage } from '@/utils/error'
import ConfirmCityGoldModal from '@/components/ConfirmCityGoldModal'

interface BuildingCardProps {
  buildingId?: string
  buildingType: string
  icon: React.ReactNode
  name: string
  description: string
  level: number
  production: string
  effectText?: string
  upgradeEndsAt?: string | null
  color: string
  bgColor: string
  locked?: boolean
}

const TICK_MS = 1000
const RESOURCE_LABELS: Record<string, string> = { wood: '木材', stone: '石料', iron: '铁矿', food: '粮食' }
const EMPTY_RESOURCES: Record<string, number> = {}

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
  buildingType,
  icon,
  name,
  description,
  level,
  production,
  effectText,
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
  const resources = useGameStore((s) => s.state?.resources.items ?? EMPTY_RESOURCES)
  const account = useAccountStore((s) => s.account)
  const skipConfirmations = useConfirmPreferenceStore((s) => s.skipConfirmations)
  const cityGoldPerSecond = balance?.cityGoldPerSecond ?? 120
  const isUpgrading = upgradeEndsAt != null
  const countdown = upgradeEndsAt ? getRemainingSeconds(upgradeEndsAt, now) : 0
  const instantCost = Math.max(1, Math.ceil(countdown / cityGoldPerSecond))
  const upgradeCost = getUpgradeCost(buildingType, level)
  const goldUpgradeCost = getGoldUpgradeCost(buildingType, level)
  const upgradeTime = getUpgradeSeconds(buildingType, level)
  const canAffordResources = upgradeCost
    ? Object.entries(upgradeCost).every(([res, cost]) => (resources[res] ?? 0) >= cost)
    : false
  const canAffordGold = goldUpgradeCost != null ? (account?.gold ?? 0) >= goldUpgradeCost : false
  const hasUpgradeCost = upgradeCost != null || goldUpgradeCost != null
  const canAffordUpgrade = upgradeCost != null ? canAffordResources : canAffordGold

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
    void useGameStore.getState().loadCityView()
    void useGameStore.getState().loadResourceView()
  }, [upgradeEndsAt, countdown])

  const handleUpgrade = async () => {
    if (!buildingId || loading || isUpgrading || !hasUpgradeCost || !canAffordUpgrade) return
    setLoading(true)
    try {
      await upgrade(buildingId)
      if (goldUpgradeCost != null && account) {
        const nextAccount = await gameApi.getAccountInfo(account.accountId)
        useAccountStore.setState({ account: nextAccount })
      }
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
      useGameStore.getState().patchCityAction(result)
      toast.success(`${name} 升级完成`)
    } catch (e: unknown) {
      const msg = getErrorMessage(e, '加速失败')
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

  const tooltipContent = hasUpgradeCost ? (
    <div className="space-y-1.5 text-[11px] min-w-[150px]">
      <p className="font-semibold text-white">升级到 Lv.{level + 1}</p>
      {upgradeCost && (
        <div className="space-y-0.5">
          {Object.entries(upgradeCost).map(([res, cost]) => {
            const have = resources[res] ?? 0
            const enough = have >= cost
            return (
              <p key={res} className={enough ? 'text-white/70' : 'text-red-400'}>
                {RESOURCE_LABELS[res] ?? res} {cost.toLocaleString()}
              </p>
            )
          })}
        </div>
      )}
      {goldUpgradeCost != null && (
        <p className={(account?.gold ?? 0) >= goldUpgradeCost ? 'text-white/70' : 'text-red-400'}>
          金币 {goldUpgradeCost.toLocaleString()}
        </p>
      )}
      <div className="pt-1 border-t border-white/10 space-y-0.5">
        {effectText && <p className="text-amber-300">{effectText}</p>}
        <p className="text-white/50">耗时 {formatDuration(upgradeTime)}</p>
      </div>
    </div>
  ) : (
    <p className="text-[11px] text-white/70">已达最高等级</p>
  )

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
          {effectText && (
            <p className="text-[11px] leading-4 text-[var(--color-text-muted)] mt-1 truncate" title={effectText}>
              {effectText}
            </p>
          )}
          <div className="flex items-center justify-between mt-2">
            <span className={`text-xs font-medium ${locked ? 'text-[var(--color-text-muted)]' : color}`}>
              {production}
            </span>
            {!locked && buildingId && (
              isUpgrading ? (
                <button
                  type="button"
                  onClick={() => {
                    if (skipConfirmations) handleInstantComplete()
                    else setConfirmOpen(true)
                  }}
                  className="flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs font-mono font-medium text-amber-500 hover:bg-amber-500/10 cursor-pointer transition-colors"
                  title="点击快速完成"
                >
                  <Zap size={10} />
                  {formatCountdown(countdown)}
                </button>
              ) : (
                <Tooltip content={tooltipContent} placement="top">
                  <button
                    type="button"
                    onClick={handleUpgrade}
                    disabled={loading || !hasUpgradeCost || !canAffordUpgrade}
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
                </Tooltip>
              )
            )}
          </div>
        </div>
      </div>

      {/* 城金加速确认弹窗 */}
      <ConfirmCityGoldModal
        open={confirmOpen}
        onClose={() => setConfirmOpen(false)}
        onConfirm={() => handleInstantComplete()}
        title={`${name} 快速完成`}
        description={`立即完成升级到 Lv.${level + 1}`}
        cost={instantCost}
        loading={instantLoading}
      />
    </div>
  )
}

export default BuildingCard
