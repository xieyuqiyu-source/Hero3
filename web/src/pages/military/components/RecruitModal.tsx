import { useState, useEffect, type FC } from 'react'
import { Clock, X } from 'lucide-react'
import type { UnitConfig } from '@/store/configStore'
import { useGameStore } from '@/store/gameStore'
import { useProjectedResources } from '@/hooks/useProjectedResources'
import { gameApi } from '@/api/game'

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

const RecruitModal: FC<RecruitModalProps> = ({ open, onClose, unitId, config, owned }) => {
  const [amount, setAmount] = useState(1)
  const [visible, setVisible] = useState(false)
  const [recruiting, setRecruiting] = useState(false)
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const setState = useGameStore((s) => s.setState)
  const resources = useProjectedResources()

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
      setAmount(0)
    } else {
      setVisible(false)
    }
  }, [open])

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
      setState(result.state)
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
        relative w-full max-w-sm rounded-2xl overflow-hidden
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
            {[
              ['攻击', config.stats.attack],
              ['步防', config.stats.infantryDefense],
              ['骑防', config.stats.cavalryDefense],
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
              <span className="flex items-center gap-0.5 text-[var(--color-text-muted)]">
                <Clock size={8} />{config.trainSeconds}s
              </span>
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
