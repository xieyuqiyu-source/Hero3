import { useState, useRef, type FC } from 'react'
import { Castle, Users, Flag, Scroll, Sparkles } from 'lucide-react'
import NpcCityTab from './components/NpcCityTab'
import MiniGamesTab from './components/MiniGamesTab'

type MapTab = 'npc' | 'players' | 'stronghold' | 'dungeon' | 'minigames'

const MapPage: FC = () => {
  const [activeTab, setActiveTab] = useState<MapTab>('npc')
  const scrollRef = useRef<HTMLDivElement>(null)
  const isDragging = useRef(false)
  const hasMoved = useRef(false)
  const startX = useRef(0)
  const scrollLeft = useRef(0)
  const velocity = useRef(0)
  const lastX = useRef(0)
  const lastTime = useRef(0)
  const animFrame = useRef<number>(0)
  const DRAG_THRESHOLD = 5

  const tabs = [
    { key: 'npc' as const, label: 'NPC', icon: Castle },
    { key: 'players' as const, label: '玩家', icon: Users },
    { key: 'stronghold' as const, label: '据点', icon: Flag },
    { key: 'dungeon' as const, label: '副本', icon: Scroll },
    { key: 'minigames' as const, label: '万象幻境', icon: Sparkles },
  ]

  const handlePointerDown = (e: React.PointerEvent) => {
    const el = scrollRef.current
    if (!el) return
    isDragging.current = true
    hasMoved.current = false
    startX.current = e.clientX
    scrollLeft.current = el.scrollLeft
    lastX.current = e.clientX
    lastTime.current = Date.now()
    velocity.current = 0
    cancelAnimationFrame(animFrame.current)
  }

  const handlePointerMove = (e: React.PointerEvent) => {
    if (!isDragging.current || !scrollRef.current) return
    const dx = e.clientX - startX.current

    // Only start dragging after threshold
    if (!hasMoved.current && Math.abs(dx) < DRAG_THRESHOLD) return
    if (!hasMoved.current) {
      hasMoved.current = true
      scrollRef.current.setPointerCapture(e.pointerId)
    }

    scrollRef.current.scrollLeft = scrollLeft.current - dx

    const now = Date.now()
    const dt = now - lastTime.current
    if (dt > 0) {
      velocity.current = (e.clientX - lastX.current) / dt
    }
    lastX.current = e.clientX
    lastTime.current = now
  }

  const handlePointerUp = (e: React.PointerEvent) => {
    if (!isDragging.current) return
    isDragging.current = false

    if (hasMoved.current && scrollRef.current) {
      scrollRef.current.releasePointerCapture(e.pointerId)

      // Inertia / momentum scrolling
      const el = scrollRef.current
      let v = velocity.current * 15

      const decelerate = () => {
        if (Math.abs(v) < 0.5) return
        el.scrollLeft -= v
        v *= 0.92
        animFrame.current = requestAnimationFrame(decelerate)
      }
      decelerate()
    }
  }

  const scrollToTab = (index: number) => {
    const el = scrollRef.current
    if (!el) return
    const buttons = el.querySelectorAll('button')
    const btn = buttons[index]
    if (!btn) return
    const containerWidth = el.clientWidth
    const btnLeft = (btn as HTMLElement).offsetLeft
    const btnWidth = (btn as HTMLElement).offsetWidth
    // Center the button in the visible area
    const target = btnLeft - (containerWidth / 2) + (btnWidth / 2)
    el.scrollTo({ left: target, behavior: 'smooth' })
  }

  const handleTabClick = (key: MapTab, index: number) => {
    setActiveTab(key)
    scrollToTab(index)
  }

  return (
    <div>
      {/* Tab Switcher */}
      <div className="rounded-xl bg-[var(--color-surface-dim)] border border-[var(--color-border)] mb-6 overflow-hidden w-fit max-w-full">
        <div
          ref={scrollRef}
          onPointerDown={handlePointerDown}
          onPointerMove={handlePointerMove}
          onPointerUp={handlePointerUp}
          onPointerCancel={handlePointerUp}
          className="overflow-x-hidden select-none touch-pan-x"
        >
          <div className="flex gap-1 p-1 w-max">
            {tabs.map((tab, index) => {
              const Icon = tab.icon
              const isActive = activeTab === tab.key
              return (
                <button
                  key={tab.key}
                  type="button"
                  onClick={() => handleTabClick(tab.key, index)}
                  className={`
                    flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium cursor-pointer whitespace-nowrap
                    transition-all duration-200
                    ${isActive
                      ? 'bg-[var(--color-surface)] text-[var(--color-accent)] shadow-[0_2px_8px_rgba(15,23,42,0.06)] border border-[var(--color-border)]'
                      : 'text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] border border-transparent'
                    }
                  `}
                >
                  <Icon size={14} />
                  {tab.label}
                </button>
              )
            })}
          </div>
        </div>
      </div>

      {/* Tab Content */}
      {activeTab === 'npc' && <NpcCityTab />}
      {activeTab === 'players' && (
        <div className="flex items-center justify-center py-16">
          <span className="text-sm text-[var(--color-text-muted)]">玩家城池系统开发中，敬请期待</span>
        </div>
      )}
      {activeTab === 'stronghold' && (
        <div className="flex items-center justify-center py-16">
          <span className="text-sm text-[var(--color-text-muted)]">据点系统开发中，敬请期待</span>
        </div>
      )}
      {activeTab === 'dungeon' && (
        <div className="flex items-center justify-center py-16">
          <span className="text-sm text-[var(--color-text-muted)]">副本系统开发中，敬请期待</span>
        </div>
      )}
      {activeTab === 'minigames' && <MiniGamesTab />}
    </div>
  )
}

export default MapPage
