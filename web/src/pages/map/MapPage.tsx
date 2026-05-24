import { useState, type FC } from 'react'
import { Castle, Users, Flag, Scroll } from 'lucide-react'
import NpcCityTab from './components/NpcCityTab'

type MapTab = 'npc' | 'players' | 'stronghold' | 'dungeon'

const MapPage: FC = () => {
  const [activeTab, setActiveTab] = useState<MapTab>('npc')

  const tabs = [
    { key: 'npc' as const, label: 'NPC', icon: Castle },
    { key: 'players' as const, label: '玩家', icon: Users },
    { key: 'stronghold' as const, label: '据点', icon: Flag },
    { key: 'dungeon' as const, label: '副本', icon: Scroll },
  ]

  return (
    <div>
      {/* Tab Switcher */}
      <div className="flex gap-1 p-1 rounded-xl bg-[var(--color-surface-dim)] border border-[var(--color-border)] w-fit mb-6">
        {tabs.map((tab) => {
          const Icon = tab.icon
          const isActive = activeTab === tab.key
          return (
            <button
              key={tab.key}
              type="button"
              onClick={() => setActiveTab(tab.key)}
              className={`
                flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium cursor-pointer
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
    </div>
  )
}

export default MapPage
