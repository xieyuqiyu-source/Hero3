import { useState, useEffect, type FC } from 'react'
import { Clock, X } from 'lucide-react'
import type { UnitConfig } from '@/store/configStore'
import { useGameStore } from '@/store/gameStore'
import { useProjectedResources } from '@/hooks/useProjectedResources'
import { gameApi } from '@/api/game'
import { formatBaseFinal, formatModifierValue, formatSecondsBaseFinal, formatUnitStatTitle, getEffectiveRecruitSeconds, getEffectiveUnitStat, type EffectiveUnitStat } from '@/utils/unitStats'

interface RecruitModalProps {
  open: boolean
  onClose: () => void
  unitId: string
  config: UnitConfig
  owned: number
}

const RESOURCE_LABELS: Record<string, string> = {
  wood: '木',
  stone: '石',
  iron: '铁',
  food: '粮',
}

const ModalStatValue: FC<{ label: string; stat: EffectiveUnitStat }> = ({ label, stat }) => {
  const boosted = stat.final !== stat.base
  return (
    <div
      className="group relative px-2 py-1.5 rounded-lg bg-[var(--color-surface-dim)] border border-[var(--color-border)] text-center"
      title={formatUnitStatTitle(label, stat)}
    >
      <div className="text-[9px] text-[var(--color-text-muted)]">{label}</div>
      <div className={`text-xs font-bold ${boosted ? 'text-green-500' : 'text-[var(--color-text-primary)]'}`}>{formatBaseFinal(stat)}</div>
      {stat.breakdown.length > 0 && (
        <div className="pointer-events-none absolute left-0 right-0 bottom-[calc(100%+6px)] z-20 min-w-44 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 shadow-lg opacity-0 invisible translate-y-1 scale-95 transition-all duration-150 ease-out group-hover:visible group-hover:opacity-100 group-hover:translate-y-0 group-hover:scale-100">
          <div className="mb-1 flex items-center justify-between gap-2">
            <span className="text-[11px] font-semibold text-[var(--color-text-primary)]">{label}</span>
            <span className="text-[11px] font-bold text-green-500">{formatBaseFinal(stat)}</span>
          </div>
          <div className="space-y-0.5">
            <div className="flex items-center justify-between gap-2 text-[10px]">
              <span className="text-[var(--color-text-secondary)]">基础</span>
              <span className="font-semibold text-[var(--color-text-primary)]">{stat.base}</span>
            </div>
            {stat.breakdown.map((item, index) => (
              <div key={`${item.source}-${index}`} className="flex items-center justify-between gap-2 text-[10px]">
                <span className="text-[var(--color-text-secondary)]">{item.source}</span>
                <span className="font-semibold text-[var(--color-text-primary)]">{formatModifierValue(item)}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

const ModalTimeValue: FC<{ stat: EffectiveUnitStat }> = ({ stat }) => (
  <span
    className={`relative group flex items-center gap-0.5 ${stat.final !== stat.base ? 'text-green-500' : 'text-[var(--color-text-muted)]'}`}
    title={formatUnitStatTitle('训练时间', stat)}
  >
    <Clock size={8} />{formatSecondsBaseFinal(stat)}
    {stat.breakdown.length > 0 && (
      <div className="pointer-events-none absolute right-0 bottom-[calc(100%+6px)] z-20 w-44 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 shadow-lg opacity-0 invisible translate-y-1 scale-95 transition-all duration-150 ease-out group-hover:visible group-hover:opacity-100 group-hover:translate-y-0 group-hover:scale-100">
        <div className="mb-1 flex items-center justify-between gap-2">
          <span className="text-[11px] font-semibold text-[var(--color-text-primary)]">训练时间</span>
          <span className="text-[11px] font-bold text-green-500">{formatSecondsBaseFinal(stat)}</span>
        </div>
        <div className="space-y-0.5">
          <div className="flex items-center justify-between gap-2 text-[10px]">
            <span className="text-[var(--color-text-secondary)]">基础</span>
            <span className="font-semibold text-[var(--color-text-primary)]">{stat.base}s</span>
          </div>
          {stat.breakdown.map((item, index) => (
            <div key={`${item.source}-${index}`} className="flex items-center justify-between gap-2 text-[10px]">
              <span className="text-[var(--color-text-secondary)]">{item.source}</span>
              <span className="font-semibold text-[var(--color-text-primary)]">{formatModifierValue(item)}</span>
            </div>
          ))}
        </div>
      </div>
    )}
  </span>
)

const RecruitModal: FC<RecruitModalProps> = ({ open, onClose, unitId, config, owned }) => {
  const [amount, setAmount] = useState(1)
  const [visible, setVisible] = useState(false)
  const [recruiting, setRecruiting] = useState(false)
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const setState = useGameStore((s) => s.setState)
  const gameState = useGameStore((s) => s.state)
  const resources = useProjectedResources()
  const attack = getEffectiveUnitStat(gameState, 'attack', config.stats.attack ?? 0)
  const infantryDefense = getEffectiveUnitStat(gameState, 'infantryDefense', config.stats.infantryDefense ?? 0)
  const cavalryDefense = getEffectiveUnitStat(gameState, 'cavalryDefense', config.stats.cavalryDefense ?? 0)
  const recruitSeconds = getEffectiveRecruitSeconds(gameState, config.category, config.trainSeconds)

  // 计算当前资源能征募的最大数量（任一资源不足则为0），上限 100000
  const maxAmount = (() => {
    if (!resources) return 0
    let max = Infinity
    for (const [res, costPer] of Object.entries(config.cost)) {
      if (costPer <= 0) continue
      const available = resources.items[res] ?? 0
      max = Math.min(max, Math.floor(available / costPer))
    }
    if (max === Infinity) return 0
    return Math.min(Math.max(0, max), 100000)
  })()

  // 动画控制
  useEffect(() => {
    if (open) {
      requestAnimationFrame(() => setVisible(true))
      setAmount(maxAmount)
    } else {
      setVisible(false)
    }
  }, [open, maxAmount])

  const handleClose = () => {
    setVisible(false)
    setTimeout(onClose, 200)
  }

  const handleRecruit = async () => {
    if (!activePlayerId || amount <= 0 || recruiting) return
    setRecruiting(true)
    try {
      const result = await gameApi.recruit(activePlayerId, unitId, amount)
      handleClose()
      // 延迟更新 state，等弹窗动画结束
      setTimeout(() => setState(result.state), 200)
    } catch {
      // 错误由全局拦截器处理
    } finally {
      setRecruiting(false)
    }
  }

  if (!open) return null

  return (
    <div className="fixed inset-0 z-[9000] flex items-center justify-center p-4">
      {/* Backdrop */}
      <div
        className={`absolute inset-0 bg-slate-900/40 backdrop-blur-[4px] transition-opacity duration-200 ${visible ? 'opacity-100' : 'opacity-0'}`}
        onClick={handleClose}
      />

      {/* Modal */}
      <div className={`
        relative w-full max-w-sm rounded-2xl overflow-visible
        bg-[var(--color-surface)] border border-[var(--color-border)]
        shadow-[0_24px_60px_rgba(15,23,42,0.25)]
        transition-all duration-200
        ${visible ? 'opacity-100 scale-100 translate-y-0' : 'opacity-0 scale-95 translate-y-2'}
      `}>
        {/* Header */}
        <div className="flex items-center gap-3 px-4 py-3 border-b border-[var(--color-border)]">
          <div className="flex-1 min-w-0">
            <div className="text-sm font-bold text-[var(--color-text-primary)]">征募 {config.name}</div>
            <div className="text-[10px] text-[var(--color-text-muted)] truncate">{config.description}</div>
          </div>
          <span className="text-sm font-bold text-[var(--color-accent)]">拥有 {owned}</span>
          <button
            type="button"
            onClick={handleClose}
            className="w-6 h-6 flex items-center justify-center rounded-full text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-accent-light)] cursor-pointer transition-colors"
          >
            <X size={14} />
          </button>
        </div>

        {/* Body */}
        <div className="px-4 py-3 space-y-3">
          {/* Stats */}
          <div className="grid grid-cols-3 gap-1.5">
            <ModalStatValue label="攻击" stat={attack} />
            <ModalStatValue label="步防" stat={infantryDefense} />
            <ModalStatValue label="骑防" stat={cavalryDefense} />
            {[
              ['速度', config.stats.speed],
              ['运载', config.stats.carryCapacity],
              ['口粮', config.stats.upkeep],
            ].map(([label, val]) => (
              <div key={label as string} className="px-2 py-1.5 rounded-lg bg-[var(--color-surface-dim)] border border-[var(--color-border)] text-center">
                <div className="text-[9px] text-[var(--color-text-muted)]">{label}</div>
                <div className="text-xs font-bold text-[var(--color-text-primary)]">{val ?? 0}</div>
              </div>
            ))}
          </div>

          {/* Single cost */}
          <div className="flex items-center justify-between px-2.5 py-1.5 rounded-lg bg-[var(--color-surface-dim)] border border-[var(--color-border)]">
            <span className="text-[10px] text-[var(--color-text-muted)]">单个消耗</span>
            <div className="flex items-center gap-2 text-[10px]">
              {Object.entries(config.cost).map(([res, val]) => (
                <span key={res} className="text-[var(--color-text-secondary)]">
                  {RESOURCE_LABELS[res]}<span className="font-semibold ml-0.5">{val}</span>
                </span>
              ))}
              <ModalTimeValue stat={recruitSeconds} />
            </div>
          </div>

          {/* Quantity */}
          <div className="flex items-center justify-between">
            <span className="text-[11px] font-medium text-[var(--color-text-primary)]">数量</span>
            <div className="flex items-center gap-1.5">
              <input
                type="number"
                value={amount}
                onChange={(e) => setAmount(Math.max(0, Math.min(maxAmount, parseInt(e.target.value) || 0)))}
                className="w-16 text-center text-sm font-bold bg-[var(--color-surface-dim)] border border-[var(--color-border)] rounded-lg py-1.5 text-[var(--color-text-primary)] outline-none focus:border-[var(--color-accent-border)]"
              />
              <span className="text-[11px] text-[var(--color-text-muted)]">/</span>
              <button
                type="button"
                onClick={() => setAmount(maxAmount)}
                className="text-sm font-bold text-[var(--color-accent)] hover:underline cursor-pointer"
              >
                {maxAmount}
              </button>
            </div>
          </div>

          {/* Insufficient warning */}
          {maxAmount === 0 && (
            <div className="px-2.5 py-2 rounded-lg bg-red-500/5 border border-red-500/20">
              <span className="text-[10px] font-medium text-red-500">资源不足，暂时不能招募</span>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex gap-2 px-4 py-3 border-t border-[var(--color-border)]">
          <button
            type="button"
            onClick={handleClose}
            className="flex-1 px-3 py-2 rounded-xl text-xs font-medium bg-[var(--color-surface-dim)] border border-[var(--color-border)] text-[var(--color-text-secondary)] hover:border-[var(--color-text-muted)] cursor-pointer transition-colors"
          >
            取消
          </button>
          <button
            type="button"
            onClick={handleRecruit}
            disabled={amount <= 0 || amount > maxAmount || recruiting}
            className="flex-1 px-3 py-2 rounded-xl text-xs font-bold bg-[var(--color-accent)] text-white hover:opacity-90 cursor-pointer transition-opacity disabled:opacity-50"
          >
            {recruiting ? '征兵中...' : '开始征兵'}
          </button>
        </div>
      </div>
    </div>
  )
}

export default RecruitModal
