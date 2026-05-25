import { useState, type FC } from 'react'
import {
  Castle,
  Swords,
  Map,
  ChevronLeft,
  ChevronRight,
  ChevronDown,
  Warehouse,
  Package,
  Shield,
  Settings,
} from 'lucide-react'
import ThemeToggle from './ThemeToggle'
import BoostButton from './BoostButton'
import type { GameState } from '@/types/game'
import { useProjectedResources } from '@/hooks/useProjectedResources'
import { useConfigStore } from '@/store/configStore'
import { useAccountStore } from '@/store/accountStore'
import { useGameStore } from '@/store/gameStore'

export interface NavItem {
  key: string
  label: string
  icon: FC<{ size?: number; className?: string }>
}

const NAV_ITEMS: NavItem[] = [
  { key: 'city', label: '城池', icon: Castle },
  { key: 'military', label: '军事', icon: Swords },
  { key: 'map', label: '地图', icon: Map },
  { key: 'settings', label: '设置', icon: Settings },
]

interface SidebarProps {
  activeKey: string
  collapsed: boolean
  gameState: GameState | null
  onNavigate: (key: string) => void
  onToggle: () => void
}

const Sidebar: FC<SidebarProps> = ({ activeKey, collapsed, gameState, onNavigate, onToggle }) => {
  const [hoveredKey, setHoveredKey] = useState<string | null>(null)
  const resources = useProjectedResources()
  const totalArmy = gameState?.army.reduce((sum, unit) => sum + unit.amount, 0) ?? 0
  const unreadMessageCount = gameState?.unreadMessageCount ?? 0
  const quickActions = [
    { key: 'news', label: '军情', hasNotify: (gameState?.recentBattleReports?.some(r => !r.read) ?? false) },
    { key: 'mail', label: '信函', hasNotify: unreadMessageCount > 0 },
    { key: 'notice', label: '公告', hasNotify: true },
    { key: 'account', label: '账户', hasNotify: false },
  ]

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
        flex items-center gap-3 px-5 py-4
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
        {!collapsed && <div className="ml-auto"><ThemeToggle /></div>}
      </div>

      {/* Quick Actions - below brand */}
      {!collapsed && (
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
      )}

      {collapsed && (
        <div className="flex flex-col items-center gap-1 py-2 border-b border-[var(--color-border)]">
          {quickActions.map((action) => (
            <button
              key={action.key}
              type="button"
              onClick={() => {
                if (action.key === 'account') onNavigate('account')
                if (action.key === 'news') onNavigate('news')
              }}
              className={`
                px-1.5 py-1.5 rounded-lg
                text-[10px] font-medium
                text-[var(--color-text-secondary)]
                hover:text-[var(--color-accent)] hover:bg-[var(--color-accent-light)]
                cursor-pointer transition-all duration-200
                ${action.hasNotify ? 'animate-text-blink' : ''}
              `}
            >
              {action.label.charAt(0)}
            </button>
          ))}
        </div>
      )}

      {/* Scrollable Content Area */}
      <div className="flex-1 overflow-y-auto overflow-x-hidden px-2.5 py-3 scrollbar-none">
        {/* City Info + Player Switch */}
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
            <PlayerSwitcher
              nickname={gameState?.player.nickname ?? '未同步'}
              civilizationLevel={gameState?.buildings.reduce((sum, b) => sum + b.level, 0) ?? 0}
            />
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
                <BoostButton currentBoost={gameState?.productionBoost} />
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
                <span className="text-xs text-[var(--color-text-muted)]">
                  容量 {resources?.capacity.wood.toLocaleString() ?? '--'}
                </span>
              </div>
              <div className="text-xs text-[var(--color-text-secondary)]">
                <p className="opacity-50">仓库容量预留</p>
              </div>
            </>
          )}
        </div>

        {/* General */}
        {!collapsed && gameState?.general && (
          <button
            type="button"
            onClick={() => onNavigate('military')}
            className="
              w-full mb-2.5 rounded-2xl p-3
              bg-[var(--color-surface-dim)] border border-[var(--color-border)]
              shadow-[0_4px_12px_rgba(15,23,42,0.03)]
              hover:border-[var(--color-accent-border)]
              cursor-pointer transition-all duration-200 text-left
            "
          >
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <span className="text-sm font-semibold text-[var(--color-text-primary)]">{gameState.general.name}</span>
                <span className="text-[10px] px-1.5 py-0.5 rounded bg-amber-500/15 text-amber-600 font-bold">Lv.{gameState.general.level}</span>
              </div>
              <span className="text-[10px] text-[var(--color-text-muted)]">将领</span>
            </div>
          </button>
        )}

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
                <span className="text-xs font-semibold text-[var(--color-accent)]">{totalArmy}</span>
              </div>
              {gameState?.army && gameState.army.length > 0 ? (
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
            </>
          )}
        </div>
      </div>

      {/* Bottom Navigation */}
      <div className="border-t border-[var(--color-border)] bg-[var(--color-surface-dim)] rounded-b-3xl p-2">
        <div className={`grid gap-1.5 ${collapsed ? 'grid-cols-1' : 'grid-cols-4'}`}>
          {NAV_ITEMS.map((item) => {
            const Icon = item.icon
            const isActive = activeKey === item.key
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
                  ${isActive
                    ? 'bg-[var(--color-accent-light)] border-[var(--color-accent-border)] text-[var(--color-accent)]'
                    : 'bg-[var(--color-surface)] border-[var(--color-border)] text-[var(--color-text-secondary)] hover:border-[var(--color-text-muted)] hover:shadow-[0_4px_12px_rgba(15,23,42,0.06)]'
                  }
                  ${hoveredKey === item.key && !isActive ? '-translate-y-0.5' : ''}
                `}
                aria-current={isActive ? 'page' : undefined}
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

import { FACTION_LABELS, FACTION_COLORS } from '@/utils/faction'

// --- Player Switcher Sub-component ---

const PlayerSwitcher: FC<{ nickname: string; civilizationLevel: number }> = ({ nickname, civilizationLevel }) => {
  const [open, setOpen] = useState(false)
  const players = useAccountStore((s) => s.players)
  const account = useAccountStore((s) => s.account)
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const setActivePlayer = useGameStore((s) => s.setActivePlayer)
  const loadGameState = useGameStore((s) => s.loadGameState)

  const handleSwitch = (playerId: string) => {
    if (playerId === activePlayerId) {
      setOpen(false)
      return
    }
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
            文明度 {civilizationLevel}
          </span>
        </div>
        {account && (
          <ChevronDown
            size={14}
            className={`text-[var(--color-text-muted)] transition-transform duration-200 ${open ? 'rotate-180' : ''}`}
          />
        )}
      </button>

      {/* Player List Dropdown */}
      <div className={`
        overflow-hidden transition-all duration-200 ease-out
        ${open ? 'max-h-[200px] mt-2 opacity-100' : 'max-h-0 opacity-0'}
      `}>
        {players.length > 0 ? (
          <div className="space-y-1 pt-2 border-t border-[var(--color-border)]">
            {players.map((p) => (

              <button
                key={p.id}
                type="button"
                onClick={() => handleSwitch(p.id)}
                className={`
                  w-full flex items-center gap-2 px-2 py-1.5 rounded-lg text-xs cursor-pointer transition-colors
                  ${p.id === activePlayerId
                    ? 'bg-[var(--color-accent-light)] text-[var(--color-accent)] font-bold'
                    : 'text-[var(--color-text-secondary)] hover:bg-[var(--color-surface)] hover:text-[var(--color-text-primary)]'
                  }
                `}
              >
                <span className={`text-[10px] font-bold ${FACTION_COLORS[p.faction] ?? 'text-[var(--color-text-muted)]'}`}>{FACTION_LABELS[p.faction] ?? p.faction}</span>
                <span className="flex-1 text-left truncate">{p.nickname}</span>
                <span className="text-[10px] text-[var(--color-text-muted)] tabular-nums">{p.buildingLevel}</span>
              </button>
            ))}
          </div>
        ) : (
          <p className="text-[10px] text-[var(--color-text-muted)] pt-2 border-t border-[var(--color-border)]">
            {account ? '加载中...' : '请先登录云同步以切换存档'}
          </p>
        )}
      </div>
    </div>
  )
}

export default Sidebar
