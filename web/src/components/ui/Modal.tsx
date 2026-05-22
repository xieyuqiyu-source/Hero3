/**
 * 统一弹窗卡片组件
 * 用法：
 *   <Modal open={showDialog} onClose={() => setShowDialog(false)} title="升级确认"
 *     footer={<button onClick={confirm}>确认</button>}
 *   >
 *     <p>确定要升级木场到 Lv.4 吗？</p>
 *   </Modal>
 */

import { useEffect, type FC, type ReactNode } from 'react'
import { X } from 'lucide-react'

interface ModalProps {
  open: boolean
  onClose: () => void
  title?: string
  children: ReactNode
  /** 固定在底部的内容 */
  footer?: ReactNode
  /** 宽度，默认 max-w-md */
  width?: string
}

const Modal: FC<ModalProps> = ({ open, onClose, title, children, footer, width = 'max-w-md' }) => {
  // ESC 关闭
  useEffect(() => {
    if (!open) return
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', handleKey)
    return () => document.removeEventListener('keydown', handleKey)
  }, [open, onClose])

  // 阻止背景滚动
  useEffect(() => {
    if (open) {
      document.body.style.overflow = 'hidden'
    } else {
      document.body.style.overflow = ''
    }
    return () => { document.body.style.overflow = '' }
  }, [open])

  if (!open) return null

  return (
    <div className="fixed inset-0 z-[9000] flex items-center justify-center p-4">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-slate-900/20 backdrop-blur-[8px] animate-in fade-in duration-200"
        onClick={onClose}
      />

      {/* Card */}
      <div
        className={`
          relative w-full ${width} rounded-3xl overflow-hidden
          bg-[var(--color-surface)] border border-[var(--color-border)]
          shadow-[0_24px_60px_rgba(15,23,42,0.2)]
          animate-in zoom-in-95 fade-in duration-200
          flex flex-col h-[70vh]
        `}
      >
        {/* Header */}
        {title && (
          <div className="flex items-center justify-between px-5 py-4 border-b border-[var(--color-border)] flex-shrink-0">
            <h3 className="text-base font-semibold text-[var(--color-text-primary)]">{title}</h3>
            <button
              type="button"
              onClick={onClose}
              className="
                w-8 h-8 flex items-center justify-center rounded-full
                text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)]
                hover:bg-[var(--color-accent-light)]
                cursor-pointer transition-colors duration-150
              "
              aria-label="关闭"
            >
              <X size={16} />
            </button>
          </div>
        )}

        {/* Body - scrollable */}
        <div className="px-5 py-4 flex-1 overflow-y-auto scrollbar-none">
          {children}
        </div>

        {/* Footer - fixed at bottom */}
        {footer && (
          <div className="flex-shrink-0 px-5 py-4 border-t border-[var(--color-border)]">
            {footer}
          </div>
        )}
      </div>
    </div>
  )
}

export default Modal
