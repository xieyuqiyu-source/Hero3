/* 本文件实现地图页入口和地图玩法页签切换，默认展示 NPC 城池。 */
import { Suspense, lazy, useEffect, useRef, useState, type FC, type PointerEvent as ReactPointerEvent } from 'react'
import { useLocation } from 'react-router-dom'
import { Castle, Users, Flag, Scroll, Sparkles } from 'lucide-react'
import WorldMapTab from './components/WorldMapTab'

type MapTab = 'npc' | 'players' | 'stronghold' | 'dungeon' | 'minigames'

const NpcCityTab = lazy(() => import('./components/NpcCityTab'))
const DungeonTab = lazy(() => import('./components/DungeonTab'))
const MiniGamesTab = lazy(() => import('./components/MiniGamesTab'))

const MAP_TABS = [
  { key: 'npc' as const, label: 'NPC', icon: Castle },
  { key: 'players' as const, label: '玩家', icon: Users },
  { key: 'stronghold' as const, label: '据点', icon: Flag },
  { key: 'dungeon' as const, label: '副本', icon: Scroll },
  { key: 'minigames' as const, label: '万象幻境', icon: Sparkles },
]

// MapPage 使用原有地图页签样式切换玩法，并默认展示 NPC 城池。
const MapPage: FC = () => {
  const location = useLocation()
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
  const dragThreshold = 5

  useEffect(() => {
    const tab = new URLSearchParams(location.search).get('tab')
    if (tab === 'npc' || tab === 'players' || tab === 'stronghold' || tab === 'dungeon' || tab === 'minigames') {
      setActiveTab(tab)
    }
  }, [location.search])

  useEffect(() => {
    return () => cancelAnimationFrame(animFrame.current)
  }, [])

  // handlePointerDown 记录页签横向拖动的起点。
  const handlePointerDown = (event: ReactPointerEvent<HTMLDivElement>) => {
    const element = scrollRef.current
    if (!element) return
    isDragging.current = true
    hasMoved.current = false
    startX.current = event.clientX
    scrollLeft.current = element.scrollLeft
    lastX.current = event.clientX
    lastTime.current = Date.now()
    velocity.current = 0
    cancelAnimationFrame(animFrame.current)
  }

  // handlePointerMove 拖动页签栏并计算惯性速度。
  const handlePointerMove = (event: ReactPointerEvent<HTMLDivElement>) => {
    const element = scrollRef.current
    if (!isDragging.current || !element) return
    const deltaX = event.clientX - startX.current
    if (!hasMoved.current && Math.abs(deltaX) < dragThreshold) return
    if (!hasMoved.current) {
      hasMoved.current = true
      element.setPointerCapture(event.pointerId)
    }

    element.scrollLeft = scrollLeft.current - deltaX
    const now = Date.now()
    const deltaTime = now - lastTime.current
    if (deltaTime > 0) {
      velocity.current = (event.clientX - lastX.current) / deltaTime
    }
    lastX.current = event.clientX
    lastTime.current = now
  }

  // handlePointerUp 结束拖动并继续短暂的惯性滚动。
  const handlePointerUp = (event: ReactPointerEvent<HTMLDivElement>) => {
    if (!isDragging.current) return
    isDragging.current = false
    const element = scrollRef.current
    if (!hasMoved.current || !element) return
    if (element.hasPointerCapture(event.pointerId)) {
      element.releasePointerCapture(event.pointerId)
    }

    let currentVelocity = velocity.current * 15
    // decelerate 逐帧降低页签栏的惯性滚动速度。
    const decelerate = () => {
      if (Math.abs(currentVelocity) < 0.5) return
      element.scrollLeft -= currentVelocity
      currentVelocity *= 0.92
      animFrame.current = requestAnimationFrame(decelerate)
    }
    decelerate()
  }

  // scrollToTab 把选中的页签平滑移动到可视区域中间。
  const scrollToTab = (index: number) => {
    const element = scrollRef.current
    if (!element) return
    const button = element.querySelectorAll('button')[index]
    if (!button) return
    const target = button.offsetLeft - element.clientWidth / 2 + button.offsetWidth / 2
    element.scrollTo({ left: target, behavior: 'smooth' })
  }

  // handleTabClick 切换地图玩法并定位当前页签。
  const handleTabClick = (key: MapTab, index: number) => {
    setActiveTab(key)
    scrollToTab(index)
  }

  // renderActiveTab 渲染当前地图玩法内容。
  const renderActiveTab = () => {
    if (activeTab === 'npc') return <NpcCityTab />
    if (activeTab === 'players') return <WorldMapTab />
    if (activeTab === 'stronghold') {
      return (
        <div className="flex items-center justify-center py-16">
          <span className="text-sm text-[var(--color-text-muted)]">据点系统开发中，敬请期待</span>
        </div>
      )
    }
    if (activeTab === 'dungeon') return <DungeonTab />
    return <MiniGamesTab />
  }

  return (
    <div>
      <div className="mb-6 w-fit max-w-full overflow-hidden rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
        <div
          ref={scrollRef}
          onPointerDown={handlePointerDown}
          onPointerMove={handlePointerMove}
          onPointerUp={handlePointerUp}
          onPointerCancel={handlePointerUp}
          className="touch-pan-x select-none overflow-x-hidden"
        >
          <div className="flex w-max gap-1 p-1">
            {MAP_TABS.map((tab, index) => {
              const Icon = tab.icon
              const isActive = activeTab === tab.key
              return (
                <button
                  key={tab.key}
                  type="button"
                  onClick={() => handleTabClick(tab.key, index)}
                  aria-pressed={isActive}
                  className={`flex cursor-pointer items-center gap-1.5 whitespace-nowrap rounded-lg border px-4 py-2 text-sm font-medium transition-all duration-200 ${
                    isActive
                      ? 'border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-accent)] shadow-[0_2px_8px_rgba(15,23,42,0.06)]'
                      : 'border-transparent text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]'
                  }`}
                >
                  <Icon size={14} />
                  {tab.label}
                </button>
              )
            })}
          </div>
        </div>
      </div>

      <Suspense fallback={<div className="py-16 text-center text-sm text-[var(--color-text-muted)]">正在读取地图页签...</div>}>
        {renderActiveTab()}
      </Suspense>
    </div>
  )
}

export default MapPage
