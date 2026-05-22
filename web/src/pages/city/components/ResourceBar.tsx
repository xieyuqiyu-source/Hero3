import { useState, useEffect, type FC } from 'react'
import { TreePine, Mountain, Gem, Wheat } from 'lucide-react'
import { useProjectedResources } from '@/hooks/useProjectedResources'

const ResourceBar: FC = () => {
  const [scrolled, setScrolled] = useState(false)
  const gameResources = useProjectedResources()

  useEffect(() => {
    const handleScroll = () => {
      setScrolled(window.scrollY > 20)
    }
    window.addEventListener('scroll', handleScroll, { passive: true })
    return () => window.removeEventListener('scroll', handleScroll)
  }, [])

  const resources = [
    { key: 'wood', name: '木材', icon: TreePine, color: 'text-green-600' },
    { key: 'stone', name: '石料', icon: Mountain, color: 'text-slate-600' },
    { key: 'iron', name: '铁矿', icon: Gem, color: 'text-orange-600' },
    { key: 'food', name: '粮食', icon: Wheat, color: 'text-amber-600' },
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
              <Icon size={16} className={`${res.color} flex-shrink-0`} />
              <span className="text-xs text-[var(--color-text-muted)] flex-shrink-0">{res.name}</span>
              <span className="text-sm font-bold text-amber-400 truncate tabular-nums">
                {(gameResources?.items[res.key] ?? 0).toLocaleString()}/{(gameResources?.capacity[res.key] ?? 0).toLocaleString()}
              </span>
            </div>
          )
        })}
      </div>
    </div>
  )
}

export default ResourceBar
