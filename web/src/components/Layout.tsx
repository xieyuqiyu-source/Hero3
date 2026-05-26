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
  ChevronDown,
} from 'lucide-react'
import Sidebar from './Sidebar'
import ThemeToggle from './ThemeToggle'
import ResourceBar from './ResourceBar'
import BoostButton from './BoostButton'
import { useGameStore } from '@/store/gameStore'
import { useAccountStore } from '@/store/accountStore'
import { useProjectedResources } from '@/hooks/useProjectedResources'
import { useConfigStore } from '@/store/configStore'
import { FACTION_LABELS, FACTION_COLORS } from '@/utils/faction'
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
    <div className="flex min-h-dvh relative overflow-x-hidden">
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
          flex-1 min-w-0 overflow-x-hidden transition-all duration-300 ease-in-out
          ${collapsed ? 'lg:ml-[100px]' : 'lg:ml-[312px]'}
        `}
      >
        <div className="max-w-[1320px] w-full min-w-0 mx-auto px-4 py-6 lg:px-6 lg:py-8">
          {error && (
            <div className="mb-4 rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] px-4 py-3 text-sm text-[var(--color-text-secondary)]">
              游戏状态加载失败：{error}
            </div>
          )}
          <ResourceBar />
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
    { key: 'news', label: '军情', hasNotify: (gameState?.recentBattleReports?.some(r => !r.read) ?? false) },
    { key: 'mail', label: '信函', hasNotify: (gameState?.unreadMessageCount ?? 0) > 0 },
    { key: 'notice', label: '公告', hasNotify: true },
    { key: 'account', label: '账户', hasNotify: false },
  ]
  const resources = useProjectedResources()
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
            onClick={() => {
              if (action.key === 'account') onNavigate('account')
              if (action.key === 'news') onNavigate('news')
            }}
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
        {/* City Info - Player Switcher */}
        <div className="mb-2.5 rounded-2xl p-3 bg-[var(--color-surface-dim)] border border-[var(--color-border)]">
          <MobilePlayerSwitcher gameState={gameState} />
        </div>

        {/* Resources */}
        <div className="mb-2.5 rounded-2xl p-3 bg-[var(--color-surface-dim)] border border-[var(--color-border)]">
          <div className="flex items-center gap-2 mb-2">
            <Package size={14} className="text-[var(--color-accent)]" />
            <span className="text-sm font-semibold text-[var(--color-text-primary)]">资源产出</span>
            <div className="ml-auto">
              <BoostButton currentBoost={gameState?.productionBoost} />
            </div>
          </div>
          <div className="grid grid-cols-1 gap-1.5">
            {[
              ['木材', gameState?.resourceProduction?.wood],
              ['石料', gameState?.resourceProduction?.stone],
              ['铁矿', gameState?.resourceProduction?.iron],
              ['粮食', gameState?.resourceProduction?.food],
            ].map(([label, value]) => (
              <div key={label} className="flex items-center gap-1.5 px-2 py-1.5 rounded-xl bg-white/60 dark:bg-white/5 border border-[var(--color-border)]">
                <span className="text-xs">{label}</span>
                <span className="text-xs font-semibold text-[var(--color-accent)] ml-auto">
                  +{typeof value === 'number' ? value.toLocaleString() : '--'}/h
                </span>
              </div>
            ))}
            <div className="flex items-center gap-1.5 px-2 py-1.5 rounded-xl bg-white/60 dark:bg-white/5 border border-[var(--color-border)]">
              <span className="text-xs">口粮</span>
              <span className="text-xs font-semibold text-[var(--color-text-muted)] ml-auto">--</span>
            </div>
          </div>
        </div>

        {/* Warehouse */}
        <div className="mb-2.5 rounded-2xl p-3 bg-[var(--color-surface-dim)] border border-[var(--color-border)]">
          <div className="flex items-center gap-2 mb-2">
            <Warehouse size={14} className="text-[var(--color-accent)]" />
            <span className="text-sm font-semibold text-[var(--color-text-primary)]">仓库</span>
            <span className="text-xs text-[var(--color-text-muted)] ml-auto">
              容量 {resources?.capacity.wood.toLocaleString() ?? '--'}
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
          {gameState?.army && gameState.army.filter(u => u.amount > 0).length > 0 ? (
            <div className="space-y-1">
              {gameState.army.filter(u => u.amount > 0).map((unit) => {
                const factionUnits = useConfigStore.getState().units?.[gameState.player.faction]
                const unitName = factionUnits?.[unit.unitType]?.name ?? unit.unitType
                return (
                  <div key={unit.unitType} className="flex items-center justify-between px-2 py-1 rounded-lg bg-white/60 dark:bg-white/5 border border-[var(--color-border)]">
                    <span className="text-[10px] text-[var(--color-text-secondary)]">{unitName}</span>
                    <span className="text-[10px] font-semibold text-[var(--color-accent)]">{unit.amount.toLocaleString()}</span>
                  </div>
                )
              })}
            </div>
          ) : (
            <p className="text-xs text-[var(--color-text-secondary)] opacity-50">暂无兵力</p>
          )}
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

// --- Mobile Player Switcher ---
const MobilePlayerSwitcher: FC<{ gameState: GameState | null }> = ({ gameState }) => {
  const [open, setOpen] = useState(false)
  const players = useAccountStore((s) => s.players)
  const account = useAccountStore((s) => s.account)
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const setActivePlayer = useGameStore((s) => s.setActivePlayer)
  const loadGameState = useGameStore((s) => s.loadGameState)

  const nickname = gameState?.player.nickname ?? '未同步'
  const civilizationLevel = gameState?.buildings.reduce((sum, b) => sum + b.level, 0) ?? 0

  const handleSwitch = (playerId: string) => {
    if (playerId === activePlayerId) { setOpen(false); return }
    setActivePlayer(playerId)
    loadGameState(playerId)
    setOpen(false)
  }

  return (
    <div>
      <button
        type="button"
        onClick={() => account && setOpen(!open)}
        className={`w-full flex items-center justify-between ${account ? 'cursor-pointer' : 'cursor-default'}`}
      >
        <div className="flex items-center gap-2">
          <Castle size={14} className="text-[var(--color-accent)]" />
          <span className="text-sm font-semibold text-[var(--color-text-primary)]">{nickname}</span>
          <span className="text-[10px] px-1.5 py-0.5 rounded bg-[var(--color-accent-light)] text-[var(--color-accent)] font-bold">
            {civilizationLevel}
          </span>
        </div>
        {account && (
          <ChevronDown size={14} className={`text-[var(--color-text-muted)] transition-transform duration-200 ${open ? 'rotate-180' : ''}`} />
        )}
      </button>
      <div className={`overflow-hidden transition-all duration-200 ease-out ${open ? 'max-h-[200px] mt-2 opacity-100' : 'max-h-0 opacity-0'}`}>
        {players.length > 0 ? (
          <div className="space-y-1 pt-2 border-t border-[var(--color-border)]">
            {players.map((p) => (
              <button
                key={p.id}
                type="button"
                onClick={() => handleSwitch(p.id)}
                className={`w-full flex items-center gap-2 px-2 py-1.5 rounded-lg text-xs cursor-pointer transition-colors ${
                  p.id === activePlayerId
                    ? 'bg-[var(--color-accent-light)] text-[var(--color-accent)] font-bold'
                    : 'text-[var(--color-text-secondary)] hover:bg-[var(--color-surface)] hover:text-[var(--color-text-primary)]'
                }`}
              >
                <span className={`text-[10px] font-bold ${FACTION_COLORS[p.faction] ?? 'text-[var(--color-text-muted)]'}`}>{FACTION_LABELS[p.faction] ?? p.faction}</span>
                <span className="flex-1 text-left truncate">{p.nickname}</span>
                <span className="text-[10px] text-[var(--color-text-muted)] tabular-nums">{p.buildingLevel}</span>
              </button>
            ))}
          </div>
        ) : (
          <p className="text-[10px] text-[var(--color-text-muted)] pt-2 border-t border-[var(--color-border)]">
            {account ? '加载中...' : '本地模式'}
          </p>
        )}
      </div>
    </div>
  )
}

export default Layout
