import { useState, type FC } from 'react'
import { Swords, FlaskConical, Users } from 'lucide-react'
import RecruitTab from './components/RecruitTab'

type MainTab = 'recruit' | 'generals' | 'tech'

const MilitaryPage: FC = () => {
  const [activeTab, setActiveTab] = useState<MainTab>('recruit')

  const tabs = [
    { key: 'recruit' as const, label: '征兵', icon: Swords },
    { key: 'generals' as const, label: '将领', icon: Users },
    { key: 'tech' as const, label: '科技', icon: FlaskConical },
  ]

  return (
    <div>
      {/* Main Tab Switcher */}
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
      {activeTab === 'recruit' && <RecruitTab />}
      {activeTab === 'generals' && (
        <div className="flex items-center justify-center py-16">
          <span className="text-sm text-[var(--color-text-muted)]">将领系统开发中，敬请期待</span>
        </div>
      )}
      {activeTab === 'tech' && (
        <div className="flex items-center justify-center py-16">
          <span className="text-sm text-[var(--color-text-muted)]">科技系统开发中，敬请期待</span>
        </div>
      )}
    </div>
  )
}

export default MilitaryPage
