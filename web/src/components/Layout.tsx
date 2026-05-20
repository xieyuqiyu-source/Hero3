import { useState, type FC, type ReactNode } from 'react'
import {
  Castle,
  Swords,
  Map,
  ScrollText,
  Package,
  Warehouse,
  Shield,
  Menu,
} from 'lucide-react'
import Sidebar from './Sidebar'

interface LayoutProps {
  activeKey: string
  onNavigate: (key: string) => void
  children: ReactNode
}

const Layout: FC<LayoutProps> = ({ activeKey, onNavigate, children }) => {
  const [collapsed, setCollapsed] = useState(false)
  const [mobileOpen, setMobileOpen] = useState(false)

  return (
    <div className="flex min-h-dvh relative">
      {/* Desktop Sidebar */}
      <Sidebar
        activeKey={activeKey}
        collapsed={collapsed}
        onNavigate={(key) => {
          onNavigate(key)
          setMobileOpen(false)
        }}
        onToggle={() => setCollapsed(!collapsed)}
      />

      {/* Mobile Sidebar Overlay */}
      {mobileOpen && (
        <div
          className="fixed inset-0 z-40 bg-slate-900/20 backdrop-blur-[6px] lg:hidden"
          onClick={() => setMobileOpen(false)}
        />
      )}

      {/* Mobile Sidebar */}
      <aside
        className={`
          fixed left-3 top-3 z-50 flex flex-col
          h-[calc(100dvh-24px)] w-[min(280px,calc(100vw-24px))] rounded-3xl
          bg-[var(--color-surface)] backdrop-blur-[14px]
          border border-[var(--color-border)]
          shadow-[0_18px_44px_rgba(15,23,42,0.12)]
          transition-transform duration-300 ease-in-out
          lg:hidden
          ${mobileOpen ? 'translate-x-0' : '-translate-x-[110%]'}
        `}
      >
        <MobileSidebarContent
          activeKey={activeKey}
          onNavigate={(key) => {
            onNavigate(key)
            setMobileOpen(false)
          }}
        />
      </aside>

      {/* Main Content */}
      <main
        className={`
          flex-1 transition-all duration-300 ease-in-out
          ${collapsed ? 'lg:ml-[100px]' : 'lg:ml-[312px]'}
        `}
      >
        <div className="max-w-[1320px] w-full mx-auto px-4 py-6 lg:px-6 lg:py-8">
          {children}
        </div>
      </main>

      {/* Mobile Menu Trigger */}
      <button
        type="button"
        onClick={() => setMobileOpen(true)}
        className={`
          fixed z-40 right-4 w-14 h-14
          flex items-center justify-center rounded-full
          text-white cursor-pointer
          bg-gradient-to-br from-indigo-500 to-indigo-600
          border border-white/20
          shadow-[0_14px_28px_rgba(79,70,229,0.25)]
          hover:-translate-y-0.5 hover:shadow-[0_18px_32px_rgba(79,70,229,0.3)]
          transition-all duration-200
          lg:hidden
          ${mobileOpen ? 'opacity-0 pointer-events-none' : 'opacity-100'}
        `}
        style={{ bottom: `calc(20px + env(safe-area-inset-bottom, 0px))` }}
        aria-label="打开菜单"
      >
        <Menu size={22} />
      </button>
    </div>
  )
}

/* Mobile sidebar content */
const MobileSidebarContent: FC<{ activeKey: string; onNavigate: (key: string) => void }> = ({
  activeKey,
  onNavigate,
}) => {
  const navItems = [
    { key: 'city', label: '城池', icon: Castle },
    { key: 'military', label: '军事', icon: Swords },
    { key: 'map', label: '地图', icon: Map },
    { key: 'reports', label: '战报', icon: ScrollText },
  ]

  return (
    <>
      {/* Brand */}
      <div className="flex items-center gap-3 px-5 py-5 border-b border-[var(--color-border)]">
        <div className="flex flex-col items-start">
          <span className="text-base font-bold tracking-tight text-[var(--color-text-primary)]">Hero3</span>
          <span className="text-[11px] text-[var(--color-text-muted)] tracking-widest">英雄三国</span>
        </div>
      </div>

      {/* Scrollable Content */}
      <div className="flex-1 overflow-y-auto px-2.5 py-3 scrollbar-none">
        {/* City Info */}
        <div className="mb-2.5 rounded-2xl p-3 bg-[var(--color-surface-dim)] border border-[var(--color-border)]">
          <div className="flex items-center gap-2 mb-2">
            <Castle size={14} className="text-[var(--color-accent)]" />
            <span className="text-sm font-semibold text-[var(--color-text-primary)]">城市信息</span>
            <span className="text-xs text-[var(--color-text-muted)] ml-auto">Lv.1</span>
          </div>
          <p className="text-xs text-[var(--color-text-secondary)] opacity-50">城市详情预留</p>
        </div>

        {/* Resources */}
        <div className="mb-2.5 rounded-2xl p-3 bg-[var(--color-surface-dim)] border border-[var(--color-border)]">
          <div className="flex items-center gap-2 mb-2">
            <Package size={14} className="text-[var(--color-accent)]" />
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
        </div>

        {/* Warehouse */}
        <div className="mb-2.5 rounded-2xl p-3 bg-[var(--color-surface-dim)] border border-[var(--color-border)]">
          <div className="flex items-center gap-2 mb-2">
            <Warehouse size={14} className="text-[var(--color-accent)]" />
            <span className="text-sm font-semibold text-[var(--color-text-primary)]">仓库</span>
            <span className="text-xs text-[var(--color-text-muted)] ml-auto">Lv.1</span>
          </div>
          <p className="text-xs text-[var(--color-text-secondary)] opacity-50">仓库容量预留</p>
        </div>

        {/* Army */}
        <div className="mb-2.5 rounded-2xl p-3 bg-[var(--color-surface-dim)] border border-[var(--color-border)]">
          <div className="flex items-center gap-2 mb-2">
            <Shield size={14} className="text-[var(--color-accent)]" />
            <span className="text-sm font-semibold text-[var(--color-text-primary)]">军队</span>
            <span className="text-xs font-semibold text-[var(--color-accent)] ml-auto">0</span>
          </div>
          <p className="text-xs text-[var(--color-text-secondary)] opacity-50">军队信息预留</p>
        </div>
      </div>

      {/* Bottom Nav */}
      <div className="border-t border-[var(--color-border)] bg-[var(--color-surface-dim)] rounded-b-3xl p-2">
        <div className="grid grid-cols-4 gap-1.5">
          {navItems.map((item) => {
            const Icon = item.icon
            return (
              <button
                key={item.key}
                type="button"
                onClick={() => onNavigate(item.key)}
                className={`
                  flex flex-col items-center justify-center gap-1
                  min-h-[44px] rounded-xl border cursor-pointer
                  transition-all duration-200
                  ${activeKey === item.key
                    ? 'bg-[var(--color-accent-light)] border-[var(--color-accent-border)] text-[var(--color-accent)]'
                    : 'bg-[var(--color-surface)] border-[var(--color-border)] text-[var(--color-text-secondary)]'
                  }
                `}
              >
                <Icon size={16} />
                <span className="text-[10px] font-bold leading-none">{item.label}</span>
              </button>
            )
          })}
        </div>
      </div>
    </>
  )
}

export default Layout
