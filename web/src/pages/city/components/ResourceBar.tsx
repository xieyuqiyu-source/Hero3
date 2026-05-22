import { useState, useEffect, type FC } from 'react'
import { TreePine, Mountain, Gem, Wheat } from 'lucide-react'
import { useGameStore } from '@/store/gameStore'

const ResourceBar: FC = () => {
  const [scrolled, setScrolled] = useState(false)
  const gameResources = useGameStore((store) => store.state?.resources)

  useEffect(() => {
    const handleScroll = () => {
      setScrolled(window.scrollY > 20)
    }
    window.addEventListener('scroll', handleScroll, { passive: true })
    return () => window.removeEventListener('scroll', handleScroll)
  }, [])

  const resources = [
    { name: '木材', icon: TreePine, value: gameResources?.wood ?? 0, capacity: gameResources?.capacity ?? 0, color: 'text-green-600' },
    { name: '石料', icon: Mountain, value: gameResources?.stone ?? 0, capacity: gameResources?.capacity ?? 0, color: 'text-slate-600' },
    { name: '铁矿', icon: Gem, value: gameResources?.iron ?? 0, capacity: gameResources?.capacity ?? 0, color: 'text-orange-600' },
    { name: '粮食', icon: Wheat, value: gameResources?.food ?? 0, capacity: gameResources?.capacity ?? 0, color: 'text-amber-600' },
  ]

  return (
    <div className={`
      sticky top-2 z-20 mb-5 px-3 py-2.5 rounded-2xl
      transition-all duration-300 ease-in-out
      ${scrolled
        ? 'bg-[var(--color-surface)]/80 backdrop-blur-md border border-[var(--color-border)] shadow-[0_4px_16px_rgba(15,23,42,0.06)]'
        : 'bg-transparent border border-transparent shadow-none'
      }
    `}>
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-2">
        {resources.map((res) => {
          const Icon = res.icon
          return (
            <div
              key={res.name}
              className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-xl min-w-0"
            >
              <Icon size={14} className={`${res.color} flex-shrink-0`} />
              <span className="text-[11px] text-[var(--color-text-muted)] flex-shrink-0">{res.name}</span>
              <span className="text-xs font-semibold text-[var(--color-text-primary)] truncate tabular-nums">
                {res.value.toLocaleString()}/{res.capacity.toLocaleString()}
              </span>
            </div>
          )
        })}
      </div>
    </div>
  )
}

export default ResourceBar
