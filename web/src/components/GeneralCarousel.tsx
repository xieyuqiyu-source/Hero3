/**
 * 武将轮播组件 - 中间大两侧小，支持鼠标拖动和触摸滑动
 * 滑动切换自动选中中间武将
 */
import { useState, useRef, useCallback, useEffect, type FC } from 'react'

// Dynamic import all general images
const generalImages = import.meta.glob<{ default: string }>(
  '/src/assets/generals/**/*.png',
  { eager: true }
)

function getGeneralImage(faction: string, id: string): string | null {
  const key = `/src/assets/generals/${faction}/${id}.png`
  return generalImages[key]?.default ?? null
}

interface General {
  id: string
  name: string
  title: string
  faction: string
}

interface GeneralCarouselProps {
  generals: General[]
  selectedId: string | null
  color: string
  borderActive: string
  onSelect: (id: string) => void
}

const GeneralCarousel: FC<GeneralCarouselProps> = ({
  generals,
  selectedId,
  color,
  borderActive,
  onSelect,
}) => {
  const [centerIndex, setCenterIndex] = useState(() => {
    // Initialize to selected general if exists
    if (selectedId) {
      const idx = generals.findIndex((g) => g.id === selectedId)
      if (idx >= 0) return idx
    }
    return 0
  })
  const [dragOffset, setDragOffset] = useState(0)
  const [isDragging, setIsDragging] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)
  const startX = useRef(0)
  const currentX = useRef(0)
  const hasMoved = useRef(false)
  const animating = useRef(false)

  const CARD_WIDTH = 120

  const goTo = useCallback((index: number) => {
    const clamped = Math.max(0, Math.min(generals.length - 1, index))
    setCenterIndex(clamped)
    // Auto-select when swiping to center
    onSelect(generals[clamped].id)
  }, [generals, onSelect])

  // Pointer drag
  const handlePointerDown = (e: React.PointerEvent) => {
    if (animating.current) return
    setIsDragging(true)
    hasMoved.current = false
    startX.current = e.clientX
    currentX.current = e.clientX
    ;(e.target as HTMLElement).setPointerCapture(e.pointerId)
  }

  const handlePointerMove = (e: React.PointerEvent) => {
    if (!isDragging) return
    currentX.current = e.clientX
    const delta = currentX.current - startX.current
    if (Math.abs(delta) > 3) hasMoved.current = true
    setDragOffset(delta)
  }

  const handlePointerUp = (e: React.PointerEvent) => {
    if (!isDragging) return
    setIsDragging(false)
    ;(e.target as HTMLElement).releasePointerCapture(e.pointerId)

    const delta = currentX.current - startX.current
    const threshold = CARD_WIDTH * 0.25

    if (delta > threshold && centerIndex > 0) {
      goTo(centerIndex - 1)
    } else if (delta < -threshold && centerIndex < generals.length - 1) {
      goTo(centerIndex + 1)
    } else if (!hasMoved.current) {
      // Tap without drag - already selected via goTo or click handler
    }

    animating.current = true
    setDragOffset(0)
    setTimeout(() => { animating.current = false }, 300)
  }

  // Keyboard support
  useEffect(() => {
    const el = containerRef.current
    if (!el) return
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === 'ArrowLeft') goTo(centerIndex - 1)
      if (e.key === 'ArrowRight') goTo(centerIndex + 1)
    }
    el.addEventListener('keydown', handleKey)
    return () => el.removeEventListener('keydown', handleKey)
  }, [centerIndex, goTo])

  const getCardStyle = (idx: number) => {
    const diff = idx - centerIndex
    const offsetPx = isDragging ? dragOffset : 0
    const baseTranslate = diff * 95 + offsetPx * 0.6

    const absDistance = Math.abs(diff - (offsetPx / CARD_WIDTH) * 0.3)
    const scale = Math.max(0.6, 1 - absDistance * 0.22)
    const opacity = Math.max(0.25, 1 - absDistance * 0.4)
    const zIndex = 10 - Math.abs(diff)

    return {
      transform: `translateX(${baseTranslate}px) scale(${scale})`,
      opacity,
      zIndex,
      filter: Math.abs(diff) >= 1 ? `blur(${Math.min(Math.abs(diff) * 1, 2.5)}px)` : 'none',
    }
  }

  return (
    <div
      ref={containerRef}
      className="relative h-full flex items-center justify-center overflow-hidden select-none cursor-grab active:cursor-grabbing"
      onPointerDown={handlePointerDown}
      onPointerMove={handlePointerMove}
      onPointerUp={handlePointerUp}
      onPointerCancel={handlePointerUp}
      tabIndex={0}
      role="listbox"
      aria-label="选择武将"
    >
      {generals.map((g, idx) => {
        const isCenter = idx === centerIndex
        const isSelected = selectedId === g.id
        const style = getCardStyle(idx)
        const visible = Math.abs(idx - centerIndex) <= 2

        if (!visible) return null

        return (
          <div
            key={g.id}
            role="option"
            aria-selected={isSelected}
            onClick={(e) => {
              e.stopPropagation()
              if (!hasMoved.current) {
                if (idx !== centerIndex) {
                  goTo(idx)
                } else {
                  onSelect(g.id)
                }
              }
            }}
            className={`
              absolute flex flex-col items-center justify-end
              w-28 rounded-2xl border
              cursor-pointer
              ${isDragging ? '' : 'transition-all duration-300 ease-out'}
              ${isCenter && isSelected
                ? `${borderActive} bg-white/10`
                : isCenter
                  ? 'border-slate-600 bg-slate-800/70'
                  : 'border-slate-800 bg-slate-800/40'
              }
            `}
            style={{ ...style, paddingBottom: '12px', aspectRatio: '3/4' }}
          >
            {/* Character art or placeholder */}
            <div className="absolute inset-0 flex items-center justify-center rounded-2xl overflow-hidden">
              {(() => {
                const imgSrc = getGeneralImage(g.faction, g.id)
                return imgSrc ? (
                  <img
                    src={imgSrc}
                    alt={g.name}
                    className="w-full h-full object-cover object-top"
                    draggable={false}
                  />
                ) : null
              })()}
            </div>
            <span className={`
              relative font-semibold text-center leading-tight
              ${isCenter
                ? `text-sm ${isSelected ? 'text-white' : color}`
                : 'text-[10px] text-slate-500'
              }
            `}>
              {g.name}
            </span>
            {isCenter && (
              <span className="relative text-[10px] text-slate-500 mt-0.5">{g.title}</span>
            )}
          </div>
        )
      })}
    </div>
  )
}

export default GeneralCarousel
