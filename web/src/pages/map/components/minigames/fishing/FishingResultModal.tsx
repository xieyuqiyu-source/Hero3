// 本文件展示仙池垂钓单次钓获结果和奖励入库提示。
import { X } from 'lucide-react'
import { useEffect, useState, type FC } from 'react'
import { RARITY_CONFIG } from './fishingConfig'
import type { FishCatch } from './types'

interface FishingResultModalProps {
  fish: FishCatch
  combo: number
  onClose: () => void
}

export const FishingResultModal: FC<FishingResultModalProps> = ({ fish, combo, onClose }) => {
  const [visible, setVisible] = useState(false)
  const config = RARITY_CONFIG[fish.rarity] ?? RARITY_CONFIG.common

  useEffect(() => {
    requestAnimationFrame(() => setVisible(true))
  }, [])

  const handleClose = () => {
    setVisible(false)
    setTimeout(onClose, 200)
  }

  return (
    <div className="fixed inset-0 z-[9000] flex items-center justify-center p-4">
      <div
        className={`absolute inset-0 transition-opacity duration-300 ${visible ? 'opacity-100' : 'opacity-0'}
          ${fish.rarity === 'mythic' ? 'bg-rose-950/65 backdrop-blur-[7px]' :
            fish.rarity === 'legendary' ? 'bg-amber-950/60 backdrop-blur-[6px]' :
            fish.rarity === 'epic' ? 'bg-purple-950/60 backdrop-blur-[5px]' :
            'bg-slate-900/50 backdrop-blur-[4px]'}
        `}
        onClick={handleClose}
      />
      <div className={`
        relative w-full max-w-[300px] overflow-hidden rounded-2xl
        border bg-[var(--color-surface)]
        transition-all duration-300
        ${fish.rarity === 'mythic' ? 'border-rose-500/50 shadow-[0_0_80px_rgba(244,63,94,0.25)]' :
          fish.rarity === 'legendary' ? 'border-amber-500/40 shadow-[0_0_60px_rgba(245,158,11,0.2)]' :
          fish.rarity === 'epic' ? 'border-purple-500/40 shadow-[0_0_40px_rgba(168,85,247,0.15)]' :
          'border-[var(--color-border)] shadow-[0_24px_60px_rgba(15,23,42,0.3)]'}
        ${visible ? 'opacity-100 scale-100 translate-y-0' : 'opacity-0 scale-95 translate-y-4'}
      `}>
        <div className={`relative overflow-hidden px-4 py-5 text-center ${config.bg}`}>
          {fish.rarity === 'legendary' && (
            <>
              <div className="absolute inset-0 animate-pulse bg-gradient-to-r from-amber-500/0 via-amber-500/20 to-amber-500/0" />
              <div className="absolute inset-0 bg-[radial-gradient(circle,rgba(245,158,11,0.15)_0%,transparent_70%)]" />
            </>
          )}
          {fish.rarity === 'mythic' && (
            <>
              <div className="absolute inset-0 animate-pulse bg-gradient-to-r from-rose-500/0 via-rose-500/25 to-rose-500/0" />
              <div className="absolute inset-0 bg-[radial-gradient(circle,rgba(244,63,94,0.18)_0%,transparent_72%)]" />
            </>
          )}
          {fish.rarity === 'epic' && (
            <div className="absolute inset-0 animate-pulse bg-gradient-to-r from-purple-500/0 via-purple-500/15 to-purple-500/0" />
          )}
          <span className={`relative mb-2 block text-4xl ${fish.rarity === 'legendary' || fish.rarity === 'mythic' ? 'animate-bounce' : ''}`}>{fish.emoji}</span>
          <h2 className={`relative text-base font-bold ${config.color}`}>
            {fish.rarity === 'mythic' ? '🐉 神话现世！' :
              fish.rarity === 'legendary' ? '🐲 传说降临！' :
              fish.rarity === 'epic' ? '💎 史诗钓获！' :
                fish.rarity === 'rare' ? '🌟 稀有鱼种！' :
                  '钓到了'}
          </h2>
          {combo > 1 && (
            <p className="relative mt-1 text-[10px] font-medium text-amber-600">🔥 连击 ×{combo}</p>
          )}
          <button
            type="button"
            onClick={handleClose}
            className="absolute right-3 top-3 rounded-full p-1 hover:bg-white/20 cursor-pointer"
          >
            <X size={14} className="text-[var(--color-text-muted)]" />
          </button>
        </div>

        <div className="space-y-3 px-4 py-4 text-center">
          <div>
            <p className="text-base font-bold text-[var(--color-text-primary)]">{fish.name}</p>
            <span className={`mt-1 inline-block rounded-full border px-2 py-0.5 text-[9px] font-medium ${config.bg} ${config.color} ${config.border}`}>
              {config.label}
            </span>
            <p className="mt-2 text-[10px] italic leading-relaxed text-[var(--color-text-muted)]">"{fish.description}"</p>
          </div>

          <div className={`rounded-xl border p-3.5 ${config.bg} ${config.border} ${config.glow}`}>
            <p className="mb-1 text-[10px] text-[var(--color-text-muted)]">可兑换</p>
            <p className={`${fish.rarity === 'mythic' || fish.rarity === 'legendary' || fish.rarity === 'epic' ? 'text-xl' : 'text-base'} font-bold ${config.color}`}>
              {fish.reward}
            </p>
            <p className={`${fish.rarity === 'mythic' || fish.rarity === 'legendary' || fish.rarity === 'epic' ? 'text-lg' : 'text-sm'} mt-0.5 font-bold ${config.color}`}>
              ×{fish.rewardAmount.toLocaleString()}
            </p>
          </div>

          <p className="text-[9px] text-[var(--color-text-muted)]">* 奖励已进入钓鱼库存</p>
        </div>

        <div className="border-t border-[var(--color-border)] px-4 py-3">
          <button
            type="button"
            onClick={handleClose}
            className="w-full rounded-xl bg-[var(--color-accent)] px-4 py-2.5 text-sm font-bold text-white transition-opacity hover:opacity-90 cursor-pointer"
          >
            继续垂钓
          </button>
        </div>
      </div>
    </div>
  )
}
