/**
 * 统一鼠标悬浮提示组件
 * 用法：
 *   <Tooltip content="提示文字">
 *     <button>悬浮我</button>
 *   </Tooltip>
 *
 *   <Tooltip content={<div>富文本内容</div>} placement="bottom">
 *     <span>目标元素</span>
 *   </Tooltip>
 */

import { useState, useRef, type FC, type ReactNode } from 'react'

type Placement = 'top' | 'bottom' | 'left' | 'right'

interface TooltipProps {
  content: ReactNode
  placement?: Placement
  children: ReactNode
  delay?: number
}

const placementStyles: Record<Placement, string> = {
  top: 'bottom-full left-1/2 -translate-x-1/2 mb-2',
  bottom: 'top-full left-1/2 -translate-x-1/2 mt-2',
  left: 'right-full top-1/2 -translate-y-1/2 mr-2',
  right: 'left-full top-1/2 -translate-y-1/2 ml-2',
}

const arrowStyles: Record<Placement, string> = {
  top: 'top-full left-1/2 -translate-x-1/2 border-4 border-transparent border-t-slate-900/90',
  bottom: 'bottom-full left-1/2 -translate-x-1/2 border-4 border-transparent border-b-slate-900/90',
  left: 'left-full top-1/2 -translate-y-1/2 border-4 border-transparent border-l-slate-900/90',
  right: 'right-full top-1/2 -translate-y-1/2 border-4 border-transparent border-r-slate-900/90',
}

const Tooltip: FC<TooltipProps> = ({ content, placement = 'top', children, delay = 150 }) => {
  const [visible, setVisible] = useState(false)
  const timerRef = useRef<ReturnType<typeof setTimeout>>(null)

  const show = () => {
    timerRef.current = setTimeout(() => setVisible(true), delay)
  }

  const hide = () => {
    if (timerRef.current) clearTimeout(timerRef.current)
    setVisible(false)
  }

  return (
    <div className="relative inline-flex" onMouseEnter={show} onMouseLeave={hide}>
      {children}
      <div
        className={`
          absolute z-50 px-3 py-2 rounded-xl
          bg-slate-900/90 text-white text-xs whitespace-nowrap
          pointer-events-none
          transition-all duration-150
          ${placementStyles[placement]}
          ${visible ? 'opacity-100 scale-100' : 'opacity-0 scale-95'}
        `}
      >
        {content}
        <div className={`absolute ${arrowStyles[placement]}`} />
      </div>
    </div>
  )
}

export default Tooltip
