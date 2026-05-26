import { useState, useEffect, type FC, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
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

  return createPortal(
    <div
      className={`fixed inset-0 z-[200] flex items-center justify-center transition-all duration-150 ${visible ? 'bg-black/30' : 'bg-transparent'}`}
      onClick={handleClose}
    >
      <div
        onClick={(e) => e.stopPropagation()}
        className={`
          rounded-xl overflow-hidden
          bg-[var(--color-surface)] border border-[var(--color-border)]
          shadow-[0_8px_24px_rgba(0,0,0,0.2)]
          transition-all duration-150 ease-out
          ${visible ? 'opacity-100 scale-100' : 'opacity-0 scale-95'}
        `}
      >
        <div className="px-4 pt-3 pb-1.5 text-center">
          <p className="text-xs font-bold text-[var(--color-text-primary)]">{title}</p>
          {description && (
            <p className="text-[10px] text-[var(--color-text-muted)] mt-0.5">{description}</p>
          )}
        </div>
        <div className="flex items-center justify-center gap-1 py-2">
          <Coins size={12} className="text-amber-500" />
          <span className="text-sm font-bold text-amber-500">{cost}</span>
          <span className="text-[10px] text-[var(--color-text-muted)]">城金</span>
        </div>
        <div className="flex border-t border-[var(--color-border)]">
          <button
            type="button"
            onClick={handleClose}
            disabled={loading}
            className="flex-1 py-2 text-[11px] font-medium text-[var(--color-text-secondary)] cursor-pointer hover:bg-[var(--color-surface-dim)] transition-colors border-r border-[var(--color-border)]"
          >
            取消
          </button>
          <button
            type="button"
            onClick={onConfirm}
            disabled={loading}
            className="flex-1 py-2 text-[11px] font-bold text-amber-500 cursor-pointer hover:bg-amber-500/5 transition-colors disabled:opacity-50"
          >
            {loading ? '...' : '确认'}
          </button>
        </div>
      </div>
    </div>,
    document.body
  )
}

export default ConfirmCityGoldModal
