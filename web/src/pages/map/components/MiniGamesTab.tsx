import { useState, type FC } from 'react'
import { Fish, Dice5, ArrowLeft } from 'lucide-react'
import FishingGame from './minigames/FishingGame'
import GamblingGame from './minigames/GamblingGame'

type MiniGameView = 'list' | 'fishing' | 'gambling'

const MiniGamesTab: FC = () => {
  const [view, setView] = useState<MiniGameView>('list')

  if (view === 'fishing') {
    return (
      <div>
        <button
          type="button"
          onClick={() => setView('list')}
          className="flex items-center gap-1.5 text-sm text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] mb-4 cursor-pointer transition-colors"
        >
          <ArrowLeft size={14} />
          返回万象幻境
        </button>
        <FishingGame />
      </div>
    )
  }

  if (view === 'gambling') {
    return (
      <div>
        <button
          type="button"
          onClick={() => setView('list')}
          className="flex items-center gap-1.5 text-sm text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] mb-4 cursor-pointer transition-colors"
        >
          <ArrowLeft size={14} />
          返回万象幻境
        </button>
        <GamblingGame />
      </div>
    )
  }

  return (
    <div>
      <div className="mb-6">
        <h2 className="text-lg font-bold text-[var(--color-text-primary)]">万象幻境</h2>
        <p className="text-sm text-[var(--color-text-muted)] mt-1">在这里体验各种趣味小游戏，赢取丰厚奖励</p>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        {/* 钓鱼 */}
        <button
          type="button"
          onClick={() => setView('fishing')}
          className="group relative overflow-hidden rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-5 text-left cursor-pointer transition-all duration-200 hover:border-[var(--color-accent)]/40 hover:shadow-[0_4px_20px_rgba(15,23,42,0.08)]"
        >
          <div className="absolute top-0 right-0 w-24 h-24 bg-blue-500/5 rounded-full -translate-y-8 translate-x-8 group-hover:scale-150 transition-transform duration-300" />
          <div className="relative">
            <div className="w-10 h-10 rounded-xl bg-blue-500/10 flex items-center justify-center mb-3">
              <Fish size={20} className="text-blue-500" />
            </div>
            <h3 className="text-sm font-bold text-[var(--color-text-primary)] mb-1">仙池垂钓</h3>
            <p className="text-xs text-[var(--color-text-muted)] leading-relaxed">
              在灵气充沛的仙池中垂钓，有机会钓到珍稀灵兽，可兑换精锐兵种
            </p>
            <div className="mt-3 flex flex-wrap gap-1.5">
              <span className="text-[10px] px-2 py-0.5 rounded-full bg-blue-500/10 text-blue-600 font-medium">免费</span>
              <span className="text-[10px] px-2 py-0.5 rounded-full bg-green-500/10 text-green-600 font-medium">可兑换兵种</span>
            </div>
          </div>
        </button>

        {/* 赌博 */}
        <button
          type="button"
          onClick={() => setView('gambling')}
          className="group relative overflow-hidden rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-5 text-left cursor-pointer transition-all duration-200 hover:border-[var(--color-accent)]/40 hover:shadow-[0_4px_20px_rgba(15,23,42,0.08)]"
        >
          <div className="absolute top-0 right-0 w-24 h-24 bg-amber-500/5 rounded-full -translate-y-8 translate-x-8 group-hover:scale-150 transition-transform duration-300" />
          <div className="relative">
            <div className="w-10 h-10 rounded-xl bg-amber-500/10 flex items-center justify-center mb-3">
              <Dice5 size={20} className="text-amber-500" />
            </div>
            <h3 className="text-sm font-bold text-[var(--color-text-primary)] mb-1">军营豪赌</h3>
            <p className="text-xs text-[var(--color-text-muted)] leading-relaxed">
              押上你的兵力进行豪赌，赢了翻倍，输了血本无归
            </p>
            <div className="mt-3 flex flex-wrap gap-1.5">
              <span className="text-[10px] px-2 py-0.5 rounded-full bg-amber-500/10 text-amber-600 font-medium">押注兵力</span>
              <span className="text-[10px] px-2 py-0.5 rounded-full bg-red-500/10 text-red-600 font-medium">高风险高回报</span>
            </div>
          </div>
        </button>
      </div>
    </div>
  )
}

export default MiniGamesTab
