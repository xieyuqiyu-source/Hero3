import { useState, useEffect, type FC, type ReactNode } from 'react'
import { Coins } from 'lucide-react'

interface ConfirmCityGoldModalProps {
  open: boolean
  onClose: () => void
  onConfirm: () => void
  title: string
  description?: ReactNode
  cost: number
  loading?: boolean
}

const ConfirmCityGoldModal: FC<ConfirmCityGoldModalProps> = ({
  open,
  onClose,
  onConfirm,
  title,
  description,
  cost,
  loading = false,
}) => {
  const [visible, setVisible] = useState(false)

  useEffect(() => {
    if (open) {
      requestAnimationFrame(() => setVisible(true))
    } else {
      setVisible(false)
    }
  }, [open])

  const handleClose = () => {
    setVisible(false)
    setTimeout(onClose, 150)
  }

  if (!open) return null

  return (
    <div
      className={`fixed inset-0 z-[200] flex items-center justify-center transition-all duration-150 ${visible ? 'bg-black/40 backdrop-blur-[2px]' : 'bg-transparent'}`}
      onClick={handleClose}
    >
      <div
        onClick={(e) => e.stopPropagation()}
        className={`
          w-72 rounded-2xl overflow-hidden
          bg-[var(--color-surface)] border border-[var(--color-border)]
          shadow-[0_16px_48px_rgba(0,0,0,0.25)]
          transition-all duration-150 ease-out
          ${visible ? 'opacity-100 scale-100 translate-y-0' : 'opacity-0 scale-95 translate-y-2'}
        `}
      >
        {/* Header */}
        <div className="px-4 pt-4 pb-2 text-center">
          <p className="text-sm font-bold text-[var(--color-text-primary)]">{title}</p>
          {description && (
            <p className="text-[11px] text-[var(--color-text-muted)] mt-1">{description}</p>
          )}
        </div>

        {/* Cost */}
        <div className="flex items-center justify-center gap-1.5 py-3">
          <Coins size={14} className="text-amber-500" />
          <span className="text-lg font-bold text-amber-500">{cost}</span>
          <span className="text-xs text-[var(--color-text-muted)]">城金</span>
        </div>

        {/* Buttons */}
        <div className="flex border-t border-[var(--color-border)]">
          <button
            type="button"
            onClick={handleClose}
            disabled={loading}
            className="flex-1 py-3 text-xs font-semibold text-[var(--color-text-secondary)] cursor-pointer hover:bg-[var(--color-surface-dim)] transition-colors border-r border-[var(--color-border)]"
          >
            取消
          </button>
          <button
            type="button"
            onClick={onConfirm}
            disabled={loading}
            className="flex-1 py-3 text-xs font-bold text-amber-500 cursor-pointer hover:bg-amber-500/5 transition-colors disabled:opacity-50"
          >
            {loading ? '...' : '确认'}
          </button>
        </div>
      </div>
    </div>
  )
}

export default ConfirmCityGoldModal
