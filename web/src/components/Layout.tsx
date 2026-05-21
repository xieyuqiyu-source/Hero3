import { useEffect, useState, type FC, type ReactNode } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import {
  Castle,
  Swords,
  Map,
  Package,
  Warehouse,
  Shield,
  Menu,
  Settings,
  LoaderCircle,
} from 'lucide-react'
import Sidebar from './Sidebar'
import ThemeToggle from './ThemeToggle'
import { useGameStore } from '@/store/gameStore'
import type { GameState } from '@/types/game'

interface LayoutProps {
  children: ReactNode
}

const Layout: FC<LayoutProps> = ({ children }) => {
  const [collapsed, setCollapsed] = useState(false)
  const [mobileOpen, setMobileOpen] = useState(false)
  const gameState = useGameStore((store) => store.state)
  const loading = useGameStore((store) => store.loading)
  const error = useGameStore((store) => store.error)
  const loadGameState = useGameStore((store) => store.loadGameState)
  const navigate = useNavigate()
  const location = useLocation()

  const activeKey = location.pathname.replace('/', '') || 'city'

  const handleNavigate = (key: string) => {
    navigate(`/${key}`)
    setMobileOpen(false)
  }

  useEffect(() => {
    void loadGameState()
  }, [loadGameState])

  return (
    <div className="flex min-h-dvh relative">
      {/* Desktop Sidebar */}
      <Sidebar
        activeKey={activeKey}
        collapsed={collapsed}
        gameState={gameState}
        onNavigate={handleNavigate}
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
          gameState={gameState}
          onNavigate={handleNavigate}
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
          {error && (
            <div className="mb-4 rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] px-4 py-3 text-sm text-[var(--color-text-secondary)]">
              游戏状态加载失败：{error}
            </div>
          )}
          {children}
        </div>
      </main>

      {loading && <GameStateLoadingOverlay />}

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

const GameStateLoadingOverlay: FC = () => (
  <div
    className="
      fixed inset-0 z-[80] flex items-center justify-center
      bg-[var(--color-bg)]/55 backdrop-blur-[3px]
      px-4
    "
    role="status"
    aria-live="polite"
  >
    <div
      className="
        flex items-center gap-3 rounded-2xl
        border border-[var(--color-border)]
        bg-[var(--color-surface)]/90
        px-4 py-3 text-sm font-medium text-[var(--color-text-secondary)]
        shadow-[0_18px_48px_rgba(15,23,42,0.12)]
      "
    >
      <LoaderCircle size={18} className="animate-spin text-[var(--color-accent)]" />
      <span>正在同步游戏状态...</span>
    </div>
  </div>
)

/* Mobile sidebar content */
const MobileSidebarContent: FC<{
  activeKey: string
  gameState: GameState | null
  onNavigate: (key: string) => void
}> = ({
  activeKey,
  gameState,
  onNavigate,
}) => {
  const navItems = [
    { key: 'city', label: '城池', icon: Castle },
    { key: 'military', label: '军事', icon: Swords },
    { key: 'map', label: '地图', icon: Map },
    { key: 'settings', label: '设置', icon: Settings },
  ]

  const quickActions = [
    { key: 'news', label: '军情', hasNotify: true },
    { key: 'mail', label: '信函', hasNotify: (gameState?.unreadMessageCount ?? 0) > 0 },
    { key: 'notice', label: '公告', hasNotify: true },
    { key: 'account', label: '账户', hasNotify: false },
  ]
  const resources = gameState?.resources
  const totalArmy = gameState?.army.reduce((sum, unit) => sum + unit.amount, 0) ?? 0

  return (
    <>
      {/* Brand */}
      <div className="flex items-center gap-3 px-5 py-4 border-b border-[var(--color-border)]">
        <div className="flex flex-col items-start">
          <span className="text-base font-bold tracking-tight text-[var(--color-text-primary)]">Hero3</span>
          <span className="text-[11px] text-[var(--color-text-muted)] tracking-widest">英雄三国</span>
        </div>
        <div className="ml-auto"><ThemeToggle /></div>
      </div>

      {/* Quick Actions */}
      <div className="flex items-center gap-1 px-3 py-2 border-b border-[var(--color-border)]">
        {quickActions.map((action) => (
          <button
            key={action.key}
            type="button"
            className={`
              px-2.5 py-1.5 rounded-lg
              text-[11px] font-medium
              text-[var(--color-text-secondary)]
              hover:text-[var(--color-accent)] hover:bg-[var(--color-accent-light)]
              cursor-pointer transition-all duration-200
              ${action.hasNotify ? 'animate-text-blink' : ''}
            `}
          >
            {action.label}
          </button>
        ))}
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
            {[
              ['木材', resources?.wood],
              ['石料', resources?.stone],
              ['铁矿', resources?.iron],
              ['粮食', resources?.food],
            ].map(([label, value]) => (
              <div key={label} className="flex items-center gap-1.5 px-2 py-1.5 rounded-xl bg-white/60 dark:bg-white/5 border border-[var(--color-border)]">
                <span className="text-xs">{label}</span>
                <span className="text-xs font-semibold text-[var(--color-accent)] ml-auto">
                  {typeof value === 'number' ? value.toLocaleString() : '--'}
                </span>
              </div>
            ))}
          </div>
        </div>

        {/* Warehouse */}
        <div className="mb-2.5 rounded-2xl p-3 bg-[var(--color-surface-dim)] border border-[var(--color-border)]">
          <div className="flex items-center gap-2 mb-2">
            <Warehouse size={14} className="text-[var(--color-accent)]" />
            <span className="text-sm font-semibold text-[var(--color-text-primary)]">仓库</span>
            <span className="text-xs text-[var(--color-text-muted)] ml-auto">
              容量 {resources?.capacity.toLocaleString() ?? '--'}
            </span>
          </div>
          <p className="text-xs text-[var(--color-text-secondary)] opacity-50">仓库容量预留</p>
        </div>

        {/* Army */}
        <div className="mb-2.5 rounded-2xl p-3 bg-[var(--color-surface-dim)] border border-[var(--color-border)]">
          <div className="flex items-center gap-2 mb-2">
            <Shield size={14} className="text-[var(--color-accent)]" />
            <span className="text-sm font-semibold text-[var(--color-text-primary)]">军队</span>
            <span className="text-xs font-semibold text-[var(--color-accent)] ml-auto">{totalArmy}</span>
          </div>
          <p className="text-xs text-[var(--color-text-secondary)] opacity-50">军队信息预留</p>
        </div>
      </div>

      {/* Bottom Nav */}
      <div className="border-t border-[var(--color-border)] bg-[var(--color-surface-dim)] rounded-b-3xl p-2">
        <div className="grid grid-cols-4 gap-1.5">
          {navItems.map((item) => {
            const Icon = item.icon
            const isActive = activeKey === item.key
            return (
              <button
                key={item.key}
                type="button"
                onClick={() => onNavigate(item.key)}
                className={`
                  flex flex-col items-center justify-center gap-1
                  min-h-[44px] rounded-xl border cursor-pointer
                  transition-all duration-200
                  ${isActive
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
