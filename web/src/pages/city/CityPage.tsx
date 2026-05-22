import { useState, type FC } from 'react'
import ResourceTab from './components/ResourceTab'
import MilitaryTab from './components/MilitaryTab'

type Tab = 'resource' | 'military'

const CityPage: FC = () => {
  const [activeTab, setActiveTab] = useState<Tab>('resource')
  const [resourceExpanded, setResourceExpanded] = useState(true)

  return (
    <div>

      {/* Tab Switcher */}
      <div className="flex gap-1 p-1 rounded-xl bg-[var(--color-surface-dim)] border border-[var(--color-border)] w-fit mb-6">
        <button
          type="button"
          onClick={() => setActiveTab('resource')}
          className={`
            px-4 py-2 rounded-lg text-sm font-medium cursor-pointer
            transition-all duration-200
            ${activeTab === 'resource'
              ? 'bg-[var(--color-surface)] text-[var(--color-accent)] shadow-[0_2px_8px_rgba(15,23,42,0.06)] border border-[var(--color-border)]'
              : 'text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] border border-transparent'
            }
          `}
        >
          资源建筑
        </button>
        <button
          type="button"
          onClick={() => setActiveTab('military')}
          className={`
            px-4 py-2 rounded-lg text-sm font-medium cursor-pointer
            transition-all duration-200
            ${activeTab === 'military'
              ? 'bg-[var(--color-surface)] text-[var(--color-accent)] shadow-[0_2px_8px_rgba(15,23,42,0.06)] border border-[var(--color-border)]'
              : 'text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] border border-transparent'
            }
          `}
        >
          军事建筑
        </button>
      </div>

      {/* Tab Content */}
      {activeTab === 'resource' ? (
        <ResourceTab expanded={resourceExpanded} onToggle={() => setResourceExpanded(!resourceExpanded)} />
      ) : (
        <MilitaryTab />
      )}
    </div>
  )
}

export default CityPage
