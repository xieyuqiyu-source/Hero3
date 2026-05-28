import { useState, useRef, type FC, type ReactNode } from 'react'
import type { ModifierBreakdownItem } from '@/types/game'

/** Modifier key 到中文名的映射 */
const KEY_LABELS: Record<string, string> = {
  productionBonus: '全资源产量',
  woodProductionBonus: '木材产量',
  stoneProductionBonus: '石料产量',
  ironProductionBonus: '铁矿产量',
  foodProductionBonus: '粮食产量',
  capacityBonus: '仓库容量',
  attackBonus: '攻击力',
  defenseBonus: '防御力',
  infantryDefenseBonus: '步兵防御',
  cavalryDefenseBonus: '骑兵防御',
  buildSpeedBonus: '建筑速度',
  recruitSpeedBonus: '征兵速度',
  marchSpeedBonus: '行军速度',
}

/** Mode 到展示格式 */
function formatModValue(value: number, mode: string): string {
  switch (mode) {
    case 'flat':
      return value >= 0 ? `+${value}` : `${value}`
    case 'percentAdd':
      return `+${Math.round(value * 100)}%`
    case 'percentMultiply':
      return `×${value + 1}`
    default:
      return `${value}`
  }
}

interface ProductionTooltipProps {
  /** 资源类型 key，如 "wood", "stone" */
  resourceType: string
  /** 当前生效的所有加成明细 */
  modifiers?: ModifierBreakdownItem[]
  children: ReactNode
}

/**
 * 产量数字的 Tooltip 包装器
 * 悬浮时显示当前生效的加成来源明细
 * 使用 relative 定位，不影响 grid 布局
 */
const ProductionTooltip: FC<ProductionTooltipProps> = ({ resourceType, modifiers, children }) => {
  const [visible, setVisible] = useState(false)
  const timerRef = useRef<ReturnType<typeof setTimeout>>(null)

  // 筛选与当前资源相关的加成
  const relevantKeys = ['productionBonus', `${resourceType}ProductionBonus`]
  const relevant = (modifiers ?? []).filter(m => relevantKeys.includes(m.key))

  if (relevant.length === 0) {
    return <>{children}</>
  }

  const show = () => {
    timerRef.current = setTimeout(() => setVisible(true), 150)
  }
  const hide = () => {
    if (timerRef.current) clearTimeout(timerRef.current)
    setVisible(false)
  }

  return (
    <div className="relative" onMouseEnter={show} onMouseLeave={hide}>
      {children}
      <div
        className={`
          absolute z-50 px-3 py-2 rounded-xl
          bg-slate-900/90 text-white text-xs whitespace-nowrap
          pointer-events-none
          transition-all duration-150
          bottom-full left-1/2 -translate-x-1/2 mb-2
          ${visible ? 'opacity-100 scale-100' : 'opacity-0 scale-95'}
        `}
      >
        <div className="space-y-1 text-[11px] min-w-[120px]">
          <p className="font-semibold text-white/90 border-b border-white/10 pb-1">加成明细</p>
          {relevant.map((mod, i) => (
            <div key={i} className="flex justify-between gap-3">
              <span className="text-white/70">{mod.source}</span>
              <span className="text-amber-300 font-medium">
                {formatModValue(mod.value, mod.mode)}
              </span>
            </div>
          ))}
        </div>
        <div className="absolute top-full left-1/2 -translate-x-1/2 border-4 border-transparent border-t-slate-900/90" />
      </div>
    </div>
  )
}

export default ProductionTooltip
