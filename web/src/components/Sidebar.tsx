import { useState, type FC } from 'react'
import {
  Castle,
  Swords,
  Map,
  ScrollText,
  ChevronLeft,
  ChevronRight,
  Warehouse,
  Package,
  Shield,
} from 'lucide-react'

export interface NavItem {
  key: string
  label: string
  icon: FC<{ size?: number; className?: string }>
}

const NAV_ITEMS: NavItem[] = [
  { key: 'city', label: '城池', icon: Castle },
  { key: 'military', label: '军事', icon: Swords },
  { key: 'map', label: '地图', icon: Map },
  { key: 'reports', label: '战报', icon: ScrollText },
]

interface SidebarProps {
  activeKey: string
  collapsed: boolean
  onNavigate: (key: string) => void
  onToggle: () => void
}

const Sidebar: FC<SidebarProps> = ({ activeKey, collapsed, onNavigate, onToggle }) => {
  const [hoveredKey, setHoveredKey] = useState<string | null>(null)

  return (
    <aside
      className={`
        fixed left-4 top-4 z-50 flex flex-col
        h-[calc(100dvh-32px)] rounded-3xl
        bg-[var(--color-surface)] backdrop-blur-[14px]
        border border-[var(--color-border)]
        shadow-[0_18px_44px_rgba(15,23,42,0.12)]
        transition-all duration-300 ease-in-out
        ${collapsed ? 'w-[68px]' : 'w-[280px]'}
        max-lg:hidden
      `}
    >
      {/* Toggle Button */}
      <button
        type="button"
        onClick={onToggle}
        className="
          absolute -right-3.5 top-10 z-10
          w-7 h-7 flex items-center justify-center
          rounded-full
          bg-[var(--color-surface)] border border-[var(--color-border)]
          shadow-[0_4px_12px_rgba(15,23,42,0.08)]
          text-[var(--color-accent)] cursor-pointer
          hover:shadow-[0_6px_16px_rgba(15,23,42,0.12)]
          transition-all duration-200
        "
        aria-label={collapsed ? '展开侧边栏' : '收起侧边栏'}
      >
        {collapsed ? <ChevronRight size={14} /> : <ChevronLeft size={14} />}
      </button>

      {/* Brand */}
      <div className={`
        flex items-center gap-3 px-5 py-5
        border-b border-[var(--color-border)]
        transition-all duration-300
        ${collapsed ? 'justify-center px-0' : ''}
      `}>
        <div className={`
          flex flex-col items-start whitespace-nowrap
          transition-all duration-300
          ${collapsed ? 'items-center' : ''}
        `}>
          <span className="text-base font-bold tracking-tight text-[var(--color-text-primary)]">Hero3</span>
          <span className={`
            text-[11px] text-[var(--color-text-muted)] tracking-widest
            transition-all duration-300
            ${collapsed ? 'hidden' : ''}
          `}>英雄三国</span>
        </div>
      </div>

      {/* Scrollable Content Area */}
      <div className="flex-1 overflow-y-auto overflow-x-hidden px-2.5 py-3 scrollbar-none">
        {/* City Info Placeholder */}
        <div className={`
          mb-2.5 rounded-2xl p-3
          bg-[var(--color-surface-dim)] border border-[var(--color-border)]
          shadow-[0_4px_12px_rgba(15,23,42,0.03)]
          transition-all duration-300
          ${collapsed ? 'px-2' : ''}
        `}>
          {collapsed ? (
            <div className="flex flex-col items-center gap-1">
              <Castle size={18} className="text-[var(--color-text-secondary)]" />
            </div>
          ) : (
            <>
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm font-semibold text-[var(--color-text-primary)]">城市信息</span>
                <span className="text-xs text-[var(--color-text-muted)]">Lv.1</span>
              </div>
              <div className="text-xs text-[var(--color-text-secondary)]">
                <p className="opacity-50">城市详情预留</p>
              </div>
            </>
          )}
        </div>

        {/* Resources Placeholder */}
        <div className={`
          mb-2.5 rounded-2xl p-3
          bg-[var(--color-surface-dim)] border border-[var(--color-border)]
          shadow-[0_4px_12px_rgba(15,23,42,0.03)]
          transition-all duration-300
          ${collapsed ? 'px-2' : ''}
        `}>
          {collapsed ? (
            <div className="flex flex-col items-center gap-1">
              <Package size={18} className="text-[var(--color-text-secondary)]" />
            </div>
          ) : (
            <>
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm font-semibold text-[var(--color-text-primary)]">资源产出</span>
              </div>
              <div className="grid grid-cols-2 gap-1.5">
                {['木材', '石料', '铁矿', '粮食'].map((res) => (
                  <div key={res} className="flex items-center gap-1.5 px-2 py-1.5 rounded-xl bg-white/60 dark:bg-white/5 border border-[var(--color-border)]">
                    <span className="text-xs">{res}</span>
                    <span className="text-xs font-semibold text-[var(--color-accent)] ml-auto">--</span>
                  </div>
                ))}
              </div>
            </>
          )}
        </div>

        {/* Warehouse Placeholder */}
        <div className={`
          mb-2.5 rounded-2xl p-3
          bg-[var(--color-surface-dim)] border border-[var(--color-border)]
          shadow-[0_4px_12px_rgba(15,23,42,0.03)]
          transition-all duration-300
          ${collapsed ? 'px-2' : ''}
        `}>
          {collapsed ? (
            <div className="flex flex-col items-center gap-1">
              <Warehouse size={18} className="text-[var(--color-text-secondary)]" />
            </div>
          ) : (
            <>
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm font-semibold text-[var(--color-text-primary)]">仓库</span>
                <span className="text-xs text-[var(--color-text-muted)]">Lv.1</span>
              </div>
              <div className="text-xs text-[var(--color-text-secondary)]">
                <p className="opacity-50">仓库容量预留</p>
              </div>
            </>
          )}
        </div>

        {/* Army Placeholder */}
        <div className={`
          mb-2.5 rounded-2xl p-3
          bg-[var(--color-surface-dim)] border border-[var(--color-border)]
          shadow-[0_4px_12px_rgba(15,23,42,0.03)]
          transition-all duration-300
          ${collapsed ? 'px-2' : ''}
        `}>
          {collapsed ? (
            <div className="flex flex-col items-center gap-1">
              <Shield size={18} className="text-[var(--color-text-secondary)]" />
            </div>
          ) : (
            <>
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm font-semibold text-[var(--color-text-primary)]">军队</span>
                <span className="text-xs font-semibold text-[var(--color-accent)]">0</span>
              </div>
              <div className="text-xs text-[var(--color-text-secondary)]">
                <p className="opacity-50">军队信息预留</p>
              </div>
            </>
          )}
        </div>
      </div>

      {/* Bottom Navigation */}
      <div className="border-t border-[var(--color-border)] bg-[var(--color-surface-dim)] rounded-b-3xl p-2">
        <div className={`grid gap-1.5 ${collapsed ? 'grid-cols-1' : 'grid-cols-4'}`}>
          {NAV_ITEMS.map((item) => {
            const Icon = item.icon
            return (
              <button
                key={item.key}
                type="button"
                onClick={() => onNavigate(item.key)}
                onMouseEnter={() => setHoveredKey(item.key)}
                onMouseLeave={() => setHoveredKey(null)}
                className={`
                  relative flex flex-col items-center justify-center gap-1
                  min-h-[44px] rounded-xl border cursor-pointer
                  transition-all duration-200
                  ${activeKey === item.key
                    ? 'bg-[var(--color-accent-light)] border-[var(--color-accent-border)] text-[var(--color-accent)]'
                    : 'bg-[var(--color-surface)] border-[var(--color-border)] text-[var(--color-text-secondary)] hover:border-[var(--color-text-muted)] hover:shadow-[0_4px_12px_rgba(15,23,42,0.06)]'
                  }
                  ${hoveredKey === item.key && activeKey !== item.key ? '-translate-y-0.5' : ''}
                `}
                aria-current={activeKey === item.key ? 'page' : undefined}
              >
                <Icon size={18} />
                {!collapsed && (
                  <span className="text-[10px] font-bold leading-none">{item.label}</span>
                )}
              </button>
            )
          })}
        </div>
      </div>
    </aside>
  )
}

export default Sidebar
