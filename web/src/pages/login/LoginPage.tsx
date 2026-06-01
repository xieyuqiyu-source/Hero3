import { useState, useEffect, useRef, useCallback, type FC, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { Cloud, ChevronDown } from 'lucide-react'
import { gameApi } from '@/api/game'
import GeneralCarousel from '@/components/GeneralCarousel'
import CloudSyncModal from '@/components/CloudSyncModal'
import { useGameStore } from '@/store/gameStore'
import { useAccountStore } from '@/store/accountStore'
import { useConfigStore } from '@/store/configStore'
import heroBg from '@/assets/hero3background.webp'
import { FACTION_UI } from './data/factions'
import type { Faction } from './types'

// 极小的模糊占位图（112 bytes webp）
const BG_PLACEHOLDER = 'data:image/webp;base64,UklGRmgAAABXRUJQVlA4IFwAAAAwBACdASoUAAsAPzmEuVOvKKWisAgB4CcJYwCdACLKCmWJ6Pdw43v4S4AA/rvqQrUD2UJiHN wrOTDEqMVex056iFRlKJ3FVgyuxcy8zTd9ta7nZoEQFTcly74AAA=='

const LoginPage: FC = () => {
  const [nickname, setNickname] = useState('')
  const [faction, setFaction] = useState<Faction | null>(null)
  const [selectedGeneral, setSelectedGeneral] = useState<string | null>(null)
  const [cloudSyncOpen, setCloudSyncOpen] = useState(false)
  const account = useAccountStore((s) => s.account)
  const [creating, setCreating] = useState(false)
  const [showSyncReminder, setShowSyncReminder] = useState(false)
  const [bgLoaded, setBgLoaded] = useState(false)
  const [showScrollHint, setShowScrollHint] = useState(false)
  const scrollRef = useRef<HTMLDivElement>(null)
  const setGameState = useGameStore((store) => store.setState)
  const setActivePlayer = useGameStore((store) => store.setActivePlayer)
  const navigate = useNavigate()

  // 从 configStore 获取将领数据，合并 UI 配置
  const factionsConfig = useConfigStore((s) => s.factions)
  const FACTIONS = FACTION_UI.map((ui) => {
    const serverFaction = factionsConfig?.[ui.key]
    const generals = serverFaction?.generals?.map((g) => ({
      id: g.id,
      name: g.name,
      title: g.title,
      faction: ui.key,
    })) ?? []
    return { ...ui, generals }
  })

  // 预加载背景大图
  useEffect(() => {
    const img = new Image()
    img.src = heroBg
    img.onload = () => setBgLoaded(true)
  }, [])

  // Track whether scrollable content has more to scroll
  const checkScrollHint = useCallback(() => {
    const el = scrollRef.current
    if (!el) { setShowScrollHint(false); return }
    const canScroll = el.scrollHeight > el.clientHeight + 10
    const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 20
    setShowScrollHint(canScroll && !atBottom)
  }, [])

  useEffect(() => {
    checkScrollHint()
  }, [faction, selectedGeneral, checkScrollHint])

  const canSubmit = nickname.trim().length > 0 && faction !== null && selectedGeneral !== null

  const handleFactionClick = (key: Faction) => {
    if (faction === key) return
    setFaction(key)
    // Auto-select first general when switching faction
    const factionData = FACTIONS.find((f) => f.key === key)
    if (factionData && factionData.generals.length > 0) {
      setSelectedGeneral(factionData.generals[0].id)
    } else {
      setSelectedGeneral(null)
    }
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    if (!canSubmit || !faction) return

    // 必须登录后才能进入游戏
    if (!account) {
      setShowSyncReminder(true)
      return
    }

    setCreating(true)
    try {
      const result = await gameApi.createPlayer(account.accountId, nickname, faction, selectedGeneral ?? undefined)
      setActivePlayer(result.playerId)
      setGameState(result.state)
      navigate('/city')
    } finally {
      setCreating(false)
    }
  }

  const handleCloseSyncReminder = () => {
    setShowSyncReminder(false)
  }

  return (
    <div className="min-h-dvh flex flex-col relative overflow-hidden">
      {/* Background: blur placeholder + full image fade-in */}
      <div
        className="absolute inset-0 bg-cover bg-center bg-no-repeat"
        style={{ backgroundImage: `url(${BG_PLACEHOLDER})`, filter: 'blur(20px)', transform: 'scale(1.1)' }}
      />
      <div
        className="absolute inset-0 bg-cover bg-center bg-no-repeat transition-opacity duration-700"
        style={{ backgroundImage: `url(${heroBg})`, opacity: bgLoaded ? 1 : 0 }}
      />

      <form onSubmit={handleSubmit} className="relative z-10 flex flex-col h-dvh p-4 sm:p-6 lg:p-8">

        {/* Top: Logo left-aligned */}
        <div className="flex items-center mb-5 flex-shrink-0">
          <h1 className="text-3xl sm:text-4xl font-black tracking-wide text-white drop-shadow-lg">
            <span style={{ fontFamily: "'STKaiti', 'KaiTi', 'SimKai', serif" }}>英雄三国</span>
            <span className="text-xl sm:text-2xl font-bold tracking-tight ml-3 opacity-80">Hero3</span>
          </h1>
          <button
            type="button"
            onClick={() => setCloudSyncOpen(true)}
            className="
              ml-4 flex items-center gap-1.5 px-3 py-1.5 rounded-lg
              bg-green-600 text-white text-xs font-medium
              hover:bg-green-500 hover:-translate-y-0.5
              cursor-pointer transition-all duration-200
              shadow-[0_4px_12px_rgba(22,163,74,0.3)]
            "
          >
            <Cloud size={13} />
            {account ? account.username : '云同步'}
          </button>
        </div>

        {/* Main: Three Faction Columns */}
        <div className="flex-1 flex flex-col sm:grid sm:grid-cols-3 gap-2 sm:gap-4 min-h-0">
          {FACTIONS.map((f) => {
            const isActive = faction === f.key
            // On mobile: collapsed factions show as single line
            const isMobileCollapsed = faction !== null && !isActive

            return (
              <div
                key={f.key}
                onClick={() => handleFactionClick(f.key)}
                className={`
                  relative flex flex-col rounded-2xl border overflow-hidden cursor-pointer
                  backdrop-blur-sm
                  transition-all duration-400 ease-in-out
                  ${isActive
                    ? `${f.borderActive} bg-[var(--color-surface)]/90 opacity-100 shadow-lg sm:opacity-100 max-sm:flex-1`
                    : 'border-white/10 bg-[var(--color-surface)]/50 opacity-80 hover:opacity-90'
                  }
                  ${isMobileCollapsed ? 'max-sm:flex-none max-sm:h-11' : 'max-sm:flex-1'}
                `}
              >
                {/* Faction Header */}
                <div className={`
                  flex items-center justify-center transition-all duration-400 ease-in-out
                  ${isMobileCollapsed
                    ? 'py-2 sm:py-6 sm:flex-col'
                    : 'py-6 sm:py-8 flex-col'
                  }
                `}>
                  <span
                    className={`
                      font-black transition-all duration-400 ease-in-out
                      ${isActive ? f.color : 'text-[var(--color-text-muted)]'}
                      ${isMobileCollapsed
                        ? 'text-xl sm:text-5xl sm:text-6xl lg:text-7xl'
                        : 'text-5xl sm:text-6xl lg:text-7xl'
                      }
                    `}
                    style={{ fontFamily: "'STKaiti', 'KaiTi', 'SimKai', serif" }}
                  >
                    {f.name}
                  </span>
                  <span className={`
                    transition-all duration-400 ease-in-out
                    ${isActive ? 'text-[var(--color-text-primary)]' : 'text-[var(--color-text-muted)]'}
                    ${isMobileCollapsed
                      ? 'ml-2 text-xs sm:ml-0 sm:mt-2 sm:text-sm'
                      : 'mt-2 text-sm'
                    }
                  `}>
                    {f.subtitle}
                  </span>
                </div>

                {/* Content - hidden on mobile when collapsed, scrollable when expanded */}
                <div
                  ref={isActive && !isMobileCollapsed ? scrollRef : undefined}
                  onScroll={checkScrollHint}
                  className={`
                    flex flex-col min-h-0
                    transition-all duration-400 ease-in-out
                    ${isMobileCollapsed
                      ? 'max-sm:h-0 max-sm:opacity-0 overflow-hidden'
                      : 'flex-1 max-sm:opacity-100 overflow-y-auto'
                    }
                  `}
                >
                  {/* Motto */}
                  <div className="px-5 pb-3">
                    <p
                      className={`text-center text-xs italic ${isActive ? 'text-[var(--color-text-secondary)]' : 'text-[var(--color-text-muted)]'} transition-colors duration-200`}
                      style={{ fontFamily: "'STKaiti', 'KaiTi', 'SimKai', serif", fontSize: 20 }}
                    >
                      「{f.motto}」
                    </p>
                  </div>

                  {/* Description */}
                  <div className="px-5 pb-4">
                    <p className={`text-sm leading-relaxed ${isActive ? 'text-[var(--color-text-secondary)]' : 'text-[var(--color-text-muted)]'} transition-colors duration-200`}>
                      {f.desc}
                    </p>
                    <div className="flex flex-wrap gap-2 mt-4">
                      {f.traits.map((trait) => (
                        <span
                          key={trait}
                          className={`
                            px-3 py-1 rounded-lg text-xs font-medium
                            ${isActive ? 'bg-[var(--color-accent-light)] text-[var(--color-text-primary)] border border-[var(--color-border)]' : 'bg-[var(--color-surface-dim)]/50 text-[var(--color-text-muted)] border border-[var(--color-border)]'}
                            transition-colors duration-200
                          `}
                        >
                          {trait}
                        </span>
                      ))}
                    </div>
                  </div>

                  {/* Generals - carousel in center area */}
                  <div className="flex-1 flex items-center justify-center px-2 py-2 min-h-[180px]">
                    <div className="w-full h-full">
                      <GeneralCarousel
                        generals={f.generals}
                        selectedId={faction === f.key ? selectedGeneral : null}
                        color={f.color}
                        borderActive={f.borderActive}
                        onSelect={(id) => {
                          setFaction(f.key)
                          setSelectedGeneral(id)
                        }}
                      />
                    </div>
                  </div>

                  {/* Bottom action - only show when this faction is fully selected */}
                <div className={`
                  border-t border-[var(--color-border)] px-4 py-3 flex-shrink-0
                  transition-all duration-300 overflow-hidden
                  ${isActive && selectedGeneral ? 'max-h-40 opacity-100' : 'max-h-0 opacity-0 py-0 border-t-0'}
                `}>

                {/* Floating scroll hint - shows when content is scrollable and not at bottom */}
                {isActive && !isMobileCollapsed && showScrollHint && (
                  <div className="sm:hidden absolute bottom-3 left-0 right-0 flex justify-center pointer-events-none z-10">
                    <span className="flex items-center gap-1 px-3 py-1.5 rounded-full bg-black/70 text-white/90 text-[11px] font-medium backdrop-blur-sm shadow-lg animate-pulse">
                      <ChevronDown size={12} />
                      下滑进入游戏
                    </span>
                  </div>
                )}
                  {/* Sync reminder */}
                  {showSyncReminder && !account && isActive && (
                    <div className="mb-3 p-3 rounded-xl bg-amber-500/10 border border-amber-500/30">
                      <p className="text-xs text-amber-200 mb-2">需要先登录才能开始游戏，登录后可云端同步进度。</p>
                      <div className="flex items-center gap-2">
                        <button
                          type="button"
                          onClick={(e) => { e.stopPropagation(); setCloudSyncOpen(true); setShowSyncReminder(false) }}
                          className="px-3 py-1.5 rounded-lg text-xs font-medium bg-green-600 text-white cursor-pointer hover:bg-green-500 transition-colors"
                        >
                          去登录
                        </button>
                        <button
                          type="button"
                          onClick={(e) => { e.stopPropagation(); handleCloseSyncReminder() }}
                          className="px-3 py-1.5 rounded-lg text-xs font-medium text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] cursor-pointer transition-colors"
                        >
                          稍后
                        </button>
                      </div>
                    </div>
                  )}

                  {!showSyncReminder && (
                    <div className="flex items-center gap-3">
                      <input
                        type="text"
                        value={nickname}
                        onChange={(e) => setNickname(e.target.value)}
                        onClick={(e) => e.stopPropagation()}
                        placeholder="输入主公名号"
                        maxLength={12}
                        className="
                          flex-1 px-3 py-2 rounded-xl
                          bg-[var(--color-surface-dim)] border border-[var(--color-border)]
                          text-[var(--color-text-primary)] placeholder-[var(--color-text-muted)]
                          text-sm outline-none
                          focus:border-[var(--color-accent-border)]
                          transition-colors duration-200
                        "
                      />
                      <button
                        type="submit"
                        disabled={!canSubmit || creating}
                        onClick={(e) => e.stopPropagation()}
                        className={`
                          px-4 py-2 rounded-xl text-sm font-semibold whitespace-nowrap
                          transition-all duration-200
                          ${canSubmit && !creating
                            ? 'bg-[var(--color-accent)] text-white hover:-translate-y-0.5 cursor-pointer'
                            : 'bg-[var(--color-surface-dim)] text-[var(--color-text-muted)] cursor-not-allowed'
                          }
                        `}
                      >
                        {creating ? '创建中...' : '征战天下'}
                      </button>
                    </div>
                  )}
                </div>
                </div>
              </div>
            )
          })}
        </div>

        {/* Footer */}
        <div className="text-center pt-4 flex-shrink-0">
          <p className="text-[11px] text-white/40">天下大势，分久必合，合久必分</p>
        </div>
      </form>
      <CloudSyncModal
        open={cloudSyncOpen}
        onClose={() => setCloudSyncOpen(false)}
      />
    </div>
  )
}

export default LoginPage
