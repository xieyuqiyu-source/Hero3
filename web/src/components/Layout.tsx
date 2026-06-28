import { useEffect, useState, type FC, type ReactNode } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import {
  Castle,
  Swords,
  ShieldPlus,
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
import FillButton from './FillButton'
import CapacityBoostButton from './CapacityBoostButton'
import ProductionTooltip from './ProductionTooltip'
import GarrisonPanel from './GarrisonPanel'
import MarchAlertTags from './MarchAlertTags'
import { useGameStore } from '@/store/gameStore'
import { useAccountStore } from '@/store/accountStore'
import { useAnnouncementUnread } from '@/hooks/useAnnouncementUnread'
import { gameApi } from '@/api/game'
import type { AnnouncementSummary } from '@/types/game'
import { useProjectedResources } from '@/hooks/useProjectedResources'
import { useConfigStore } from '@/store/configStore'
import { FACTION_LABELS, FACTION_COLORS } from '@/utils/faction'
import { sortArmyForDisplay } from '@/utils/armySort'
import type { GameState } from '@/types/game'

interface LayoutProps {
  children: ReactNode
}

const Layout: FC<LayoutProps> = ({ children }) => {
  const [collapsed, setCollapsed] = useState(false)
  const [mobileOpen, setMobileOpen] = useState(false)
  const gameState = useGameStore((store) => store.state)
  const activePlayerId = useGameStore((store) => store.activePlayerId)
  const loading = useGameStore((store) => store.loading)
  const error = useGameStore((store) => store.error)
  const loadGameState = useGameStore((store) => store.loadGameState)
  const navigate = useNavigate()
  const location = useLocation()
  const [popupQueue, setPopupQueue] = useState<AnnouncementSummary[]>([])
  const currentPopup = popupQueue[0]

  const activeKey = location.pathname.replace('/', '') || 'city'

  const handleNavigate = (key: string) => {
    navigate(`/${key}`)
    setMobileOpen(false)
  }

  useEffect(() => {
    void loadGameState()
  }, [loadGameState])

  useEffect(() => {
    let cancelled = false
    const refreshPopups = () => {
      if (!activePlayerId) {
        setPopupQueue([])
        return
      }
      gameApi.listAnnouncementPopups(activePlayerId).then((result) => {
        if (!cancelled) setPopupQueue(Array.isArray(result.items) ? result.items : [])
      }).catch(() => {
        if (!cancelled) setPopupQueue([])
      })
    }
    if (!activePlayerId) {
      setPopupQueue([])
      return
    }
    refreshPopups()
    const timer = window.setInterval(refreshPopups, 30000)
    const handleFocus = () => refreshPopups()
    window.addEventListener('focus', handleFocus)
    return () => {
      cancelled = true
      window.clearInterval(timer)
      window.removeEventListener('focus', handleFocus)
    }
  }, [activePlayerId])

  useEffect(() => {
    if (!activePlayerId || !currentPopup || currentPopup.isPopupShown) return
    gameApi.markAnnouncementPopupShown(activePlayerId, currentPopup.id).catch(() => {})
  }, [activePlayerId, currentPopup])

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
      {currentPopup && activePlayerId && (
        <div className="fixed inset-0 z-[90] flex items-center justify-center bg-slate-950/35 px-4 backdrop-blur-[4px]">
          <div className="w-full max-w-lg rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-5 shadow-[0_24px_70px_rgba(15,23,42,0.24)]">
            <div className="mb-3 flex items-center justify-between gap-3">
              <span className="rounded-full bg-[var(--color-accent-light)] px-2.5 py-1 text-xs font-semibold text-[var(--color-accent)]">公告</span>
              {currentPopup.pinned && <span className="text-xs text-[var(--color-text-muted)]">置顶</span>}
            </div>
            <h2 className="text-lg font-bold text-[var(--color-text-primary)]">{currentPopup.title}</h2>
            <p className="mt-2 text-sm leading-6 text-[var(--color-text-secondary)]">{currentPopup.summary}</p>
            <div className="mt-5 flex justify-end gap-2">
              <button
                type="button"
                onClick={() => {
                  gameApi.dismissAnnouncement(activePlayerId, currentPopup.id).catch(() => {})
                  setPopupQueue((items) => items.slice(1))
                }}
                className="rounded-lg border border-[var(--color-border)] px-4 py-2 text-sm text-[var(--color-text-secondary)] transition-colors hover:bg-[var(--color-surface-dim)]"
              >
                关闭
              </button>
              <button
                type="button"
                onClick={() => {
                  navigate(`/notice?announcementId=${currentPopup.id}`)
                  setPopupQueue((items) => items.slice(1))
                }}
                className="rounded-lg bg-[var(--color-accent)] px-4 py-2 text-sm font-semibold text-white transition-opacity hover:opacity-90"
              >
                查看详情
              </button>
            </div>
          </div>
        </div>
      )}

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
    { key: 'reinforcements', label: '增援', icon: ShieldPlus },
    { key: 'map', label: '地图', icon: Map },
    { key: 'settings', label: '设置', icon: Settings },
  ]

  const unreadMessageCount = gameState?.unreadMessageCount ?? 0
  const unreadMailCount = gameState?.unreadMailCount ?? 0
  const announcementUnread = useAnnouncementUnread()
  const newsHasNotify = unreadMessageCount > 0
  const quickActions = [
    { key: 'news', label: '军情', hasNotify: newsHasNotify },
    { key: 'mail', label: '信函', hasNotify: unreadMailCount > 0 },
    { key: 'notice', label: '公告', hasNotify: announcementUnread },
    { key: 'account', label: '账户', hasNotify: false },
    { key: 'help', label: '帮助', hasNotify: false },
  ]
  const resources = useProjectedResources()
  const units = useConfigStore((s) => s.units)
  const factionUnits = units?.[gameState?.player.faction ?? '']
  const visibleArmy = sortArmyForDisplay(gameState?.army, factionUnits)
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
              if (action.key === 'mail') onNavigate('mail')
              if (action.key === 'notice') onNavigate('notice')
              if (action.key === 'help') onNavigate('help')
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
            <span className="text-sm font-semibold text-[var(--color-text-primary)]">产出</span>
            <div className="ml-auto flex items-center gap-1.5">
              <FillButton />
              <CapacityBoostButton currentBoost={gameState?.capacityBoost} />
              <BoostButton currentBoost={gameState?.productionBoost} />
            </div>
          </div>
          <div className="grid grid-cols-1 gap-1.5">
            {([
              ['木材', 'wood', gameState?.resourceProduction?.wood],
              ['石料', 'stone', gameState?.resourceProduction?.stone],
              ['铁矿', 'iron', gameState?.resourceProduction?.iron],
              ['粮食', 'food', gameState?.resourceProduction?.food],
            ] as const).map(([label, resType, value]) => (
              <ProductionTooltip key={label} resourceType={resType} modifiers={gameState?.activeModifiers}>
                <div className="flex items-center gap-1.5 px-2 py-1.5 rounded-xl bg-white/60 dark:bg-white/5 border border-[var(--color-border)]">
                  <span className="text-xs">{label}</span>
                  <span className="text-xs font-semibold text-[var(--color-accent)] ml-auto">
                    +{typeof value === 'number' ? value.toLocaleString() : '--'}/h
                  </span>
                </div>
              </ProductionTooltip>
            ))}
            {/* 预留功能：口粮产出入口暂时隐藏 */}
          </div>
        </div>

        {/* 预留功能：仓库卡片暂时隐藏 */}
        {false && (
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
        )}

        {/* Army */}
        <div className="mb-2.5 rounded-2xl p-3 bg-[var(--color-surface-dim)] border border-[var(--color-border)]">
          <div className="flex items-center gap-2 mb-2">
            <Shield size={14} className="text-[var(--color-accent)]" />
            <span className="text-sm font-semibold text-[var(--color-text-primary)]">军队</span>
            <span className="text-xs font-semibold text-[var(--color-accent)] ml-auto">{totalArmy}</span>
          </div>
          {visibleArmy.length > 0 ? (
            <div className="space-y-1">
              {visibleArmy.map((unit) => {
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
          <MarchAlertTags />
        </div>
        <div className="mb-2.5">
          <GarrisonPanel gameStateReady={gameState !== null} compact />
        </div>
      </div>

      {/* Bottom Nav */}
      <div className="border-t border-[var(--color-border)] bg-[var(--color-surface-dim)] rounded-b-3xl p-2">
        <div className="grid grid-cols-5 gap-1.5">
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
          <span className="text-[10px] px-1.5 py-0.5 rounded bg-amber-500/15 text-amber-600 font-bold">
            🪙 {(gameState?.cityGold ?? 0).toLocaleString()}
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
                <span className="text-[10px] text-[var(--color-text-muted)] tabular-nums">建 {p.buildingLevel}</span>
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
