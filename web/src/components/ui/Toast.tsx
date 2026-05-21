/**
 * 统一通知组件
 * 用法：
 *   import { toast } from '@/components/ui'
 *   toast.success('升级成功')
 *   toast.error('资源不足')
 *   toast.info('征兵已开始')
 *
 * 在 App 中挂载 <ToastContainer />
 */

import { useState, useEffect, useCallback, type FC } from 'react'
import { CheckCircle, XCircle, Info, X } from 'lucide-react'
import { useToastStore, type ToastItemData, type ToastType } from './toastStore'

const icons: Record<ToastType, FC<{ size?: number; className?: string }>> = {
  success: CheckCircle,
  error: XCircle,
  info: Info,
}

const colors: Record<ToastType, string> = {
  success: 'text-green-500',
  error: 'text-red-500',
  info: 'text-[var(--color-accent)]',
}

export const ToastContainer: FC = () => {
  const items = useToastStore((s) => s.items)
  const remove = useToastStore((s) => s.remove)

  return (
    <div className="fixed top-4 right-4 z-[9999] flex flex-col gap-2 pointer-events-none max-sm:left-4 max-sm:right-4">
      {items.map((item) => (
        <ToastItem key={item.id} item={item} onClose={() => remove(item.id)} />
      ))}
    </div>
  )
}

const ToastItem: FC<{ item: ToastItemData; onClose: () => void }> = ({ item, onClose }) => {
  const [visible, setVisible] = useState(false)
  const Icon = icons[item.type]

  useEffect(() => {
    requestAnimationFrame(() => setVisible(true))
  }, [])

  const handleClose = useCallback(() => {
    setVisible(false)
    setTimeout(onClose, 200)
  }, [onClose])

  return (
    <div
      className={`
        pointer-events-auto flex items-center gap-2.5 px-4 py-3 rounded-xl
        bg-[var(--color-surface)] border border-[var(--color-border)]
        shadow-[0_8px_24px_rgba(15,23,42,0.1)]
        backdrop-blur-md
        transition-all duration-200 ease-out
        ${visible ? 'opacity-100 translate-y-0' : 'opacity-0 -translate-y-2'}
      `}
    >
      <Icon size={16} className={colors[item.type]} />
      <span className="text-sm text-[var(--color-text-primary)] flex-1">{item.message}</span>
      <button
        type="button"
        onClick={handleClose}
        className="text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] cursor-pointer transition-colors"
      >
        <X size={14} />
      </button>
    </div>
  )
}
