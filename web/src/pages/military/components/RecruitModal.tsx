import { useState, useEffect, type FC } from 'react'
import { Clock, X } from 'lucide-react'
import type { UnitConfig } from '@/store/configStore'

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

function formatTime(seconds: number): string {
  if (seconds < 60) return `${seconds}秒`
  if (seconds < 3600) {
    const m = Math.floor(seconds / 60)
    const s = seconds % 60
    return s > 0 ? `${m}分${s}秒` : `${m}分`
  }
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  return m > 0 ? `${h}时${m}分` : `${h}时`
}

const RecruitModal: FC<RecruitModalProps> = ({ open, onClose, config, owned }) => {
  const [amount, setAmount] = useState(1)
  const [visible, setVisible] = useState(false)

  // 动画控制
  useEffect(() => {
    if (open) {
      requestAnimationFrame(() => setVisible(true))
      setAmount(1)
    } else {
      setVisible(false)
    }
  }, [open])

  const handleClose = () => {
    setVisible(false)
    setTimeout(onClose, 200)
  }

  const totalCost = Object.fromEntries(
    Object.entries(config.cost).map(([res, val]) => [res, val * amount])
  )
  const totalTime = config.trainSeconds * amount

  const handleRecruit = () => {
    // TODO: 调用征兵接口
    console.log('recruit:', amount)
    handleClose()
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
          <span className="text-2xl">{config.icon}</span>
          <div className="flex-1 min-w-0">
            <div className="text-sm font-bold text-[var(--color-text-primary)]">征募 {config.name}</div>
            <div className="text-[10px] text-[var(--color-text-muted)] truncate">{config.description}</div>
          </div>
          <span className="text-[10px] font-semibold text-[var(--color-accent)]">拥有 {owned}</span>
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
            <div className="flex items-center gap-1">
              <button type="button" onClick={() => setAmount(Math.max(1, amount - 10))} className="w-6 h-6 rounded-md bg-[var(--color-surface-dim)] border border-[var(--color-border)] text-[10px] font-bold text-[var(--color-text-secondary)] hover:border-[var(--color-accent-border)] cursor-pointer transition-colors">-</button>
              <input
                type="number"
                value={amount}
                onChange={(e) => setAmount(Math.max(1, parseInt(e.target.value) || 1))}
                className="w-14 text-center text-xs font-bold bg-[var(--color-surface-dim)] border border-[var(--color-border)] rounded-md py-1 text-[var(--color-text-primary)] outline-none focus:border-[var(--color-accent-border)]"
              />
              <button type="button" onClick={() => setAmount(amount + 10)} className="w-6 h-6 rounded-md bg-[var(--color-surface-dim)] border border-[var(--color-border)] text-[10px] font-bold text-[var(--color-text-secondary)] hover:border-[var(--color-accent-border)] cursor-pointer transition-colors">+</button>
              <button type="button" onClick={() => setAmount(50)} className="px-1.5 py-1 rounded-md text-[9px] font-medium bg-[var(--color-surface-dim)] border border-[var(--color-border)] text-[var(--color-text-muted)] hover:text-[var(--color-accent)] cursor-pointer transition-colors">50</button>
              <button type="button" onClick={() => setAmount(100)} className="px-1.5 py-1 rounded-md text-[9px] font-medium bg-[var(--color-surface-dim)] border border-[var(--color-border)] text-[var(--color-text-muted)] hover:text-[var(--color-accent)] cursor-pointer transition-colors">MAX</button>
            </div>
          </div>

          {/* Total */}
          <div className="px-2.5 py-2 rounded-lg bg-amber-500/5 border border-amber-500/20">
            <div className="flex items-center justify-between text-[10px]">
              <span className="font-medium text-amber-600">总消耗</span>
              <div className="flex items-center gap-2">
                {Object.entries(totalCost).map(([res, val]) => (
                  <span key={res} className="text-[var(--color-text-secondary)]">
                    {RESOURCE_LABELS[res]}<span className="font-bold text-amber-600 ml-0.5">{val.toLocaleString()}</span>
                  </span>
                ))}
              </div>
            </div>
            <div className="flex items-center justify-between text-[10px] mt-1">
              <span className="font-medium text-amber-600">总时间</span>
              <span className="font-bold text-amber-600 flex items-center gap-0.5">
                <Clock size={9} />{formatTime(totalTime)}
              </span>
            </div>
          </div>
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
            disabled={amount <= 0}
            className="flex-1 px-3 py-2 rounded-xl text-xs font-bold bg-[var(--color-accent)] text-white hover:opacity-90 cursor-pointer transition-opacity disabled:opacity-50"
          >
            开始征兵
          </button>
        </div>
      </div>
    </div>
  )
}

export default RecruitModal
