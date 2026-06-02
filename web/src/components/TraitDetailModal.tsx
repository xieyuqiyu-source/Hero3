import { useState, useEffect, type FC } from 'react'
import { X } from 'lucide-react'
import { type TraitMeta } from '@/utils/traits'

interface TraitDetailModalProps {
  trait: TraitMeta | null
  onClose: () => void
}

const TraitDetailModal: FC<TraitDetailModalProps> = ({ trait, onClose }) => {
  const [visible, setVisible] = useState(false)

  useEffect(() => {
    if (trait) {
      requestAnimationFrame(() => setVisible(true))
    } else {
      setVisible(false)
    }
  }, [trait])

  const handleClose = () => {
    setVisible(false)
    setTimeout(onClose, 200)
  }

  if (!trait) return null

  return (
    <div className="fixed inset-0 z-[9500] flex items-center justify-center p-4">
      <div
        className={`absolute inset-0 bg-slate-900/50 backdrop-blur-[4px] transition-opacity duration-200 ${visible ? 'opacity-100' : 'opacity-0'}`}
        onClick={handleClose}
      />
      <div className={`
        relative w-full max-w-sm rounded-2xl overflow-hidden
        bg-[var(--color-surface)] border border-amber-500/30
        shadow-[0_24px_60px_rgba(245,158,11,0.25)]
        transition-all duration-200
        ${visible ? 'opacity-100 scale-100' : 'opacity-0 scale-95'}
      `}>
        {/* Header */}
        <div className="px-4 py-4 bg-gradient-to-r from-amber-500/15 to-amber-500/5 border-b border-amber-500/20">
          <div className="flex items-center gap-2">
            <span className="text-2xl">{trait.icon}</span>
            <div className="flex-1 min-w-0">
              <h2 className="text-base font-bold text-amber-600">{trait.name}</h2>
            </div>
            <button
              type="button"
              onClick={handleClose}
              className="w-7 h-7 flex items-center justify-center rounded-full text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-white/10 cursor-pointer transition-colors"
            >
              <X size={14} />
            </button>
          </div>
        </div>

        {/* Body */}
        <div className="px-4 py-4">
          <p className="text-xs leading-relaxed text-[var(--color-text-secondary)]">
            {trait.details.summary || trait.description || '暂无特性说明。'}
          </p>
        </div>

        {/* Footer */}
        <div className="px-4 py-3 border-t border-[var(--color-border)]">
          <button
            type="button"
            onClick={handleClose}
            className="w-full px-4 py-2 rounded-xl text-xs font-medium bg-[var(--color-surface-dim)] text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-dim)]/80 cursor-pointer transition-colors"
          >
            知道了
          </button>
        </div>
      </div>
    </div>
  )
}

export default TraitDetailModal
