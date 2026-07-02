// 本文件实现军事主页面，集中管理战争、征兵、将领和科技页签。
import { useState, useEffect, type FC } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Swords, FlaskConical, Users, ShieldPlus } from 'lucide-react'
import RecruitTab from './components/RecruitTab'
import GeneralPanel from './components/GeneralPanel'
import WarTab from './components/WarTab'
import { useGameStore } from '@/store/gameStore'

type MainTab = 'war' | 'recruit' | 'generals' | 'tech'

const MAIN_TABS: MainTab[] = ['war', 'recruit', 'generals', 'tech']

const MilitaryPage: FC = () => {
  const [searchParams, setSearchParams] = useSearchParams()
  const rawInitialTab = searchParams.get('tab')
  const initialTab = normalizeTab(rawInitialTab)
  const [activeTab, setActiveTab] = useState<MainTab>(initialTab)
  const loadMilitaryView = useGameStore((s) => s.loadMilitaryView)
  const loadGeneralsView = useGameStore((s) => s.loadGeneralsView)

  // URL tab 变化时同步切换，兼容历史 reinforcements 参数。
  useEffect(() => {
    const t = normalizeTab(searchParams.get('tab'))
    if (t !== activeTab) {
      setActiveTab(t)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchParams])

  useEffect(() => {
    if (activeTab === 'generals') {
      void loadGeneralsView()
      return
    }
    void loadMilitaryView()
  }, [activeTab, loadGeneralsView, loadMilitaryView])

  const handleTabChange = (key: MainTab) => {
    setActiveTab(key)
    setSearchParams(key === 'war' ? {} : { tab: key }, { replace: true })
  }

  const tabs = [
    { key: 'war' as const, label: '战争', icon: ShieldPlus },
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
              onClick={() => handleTabChange(tab.key)}
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
      {activeTab === 'war' && <WarTab />}
      {activeTab === 'recruit' && <RecruitTab />}
      {activeTab === 'generals' && <GeneralPanel />}
      {activeTab === 'tech' && (
        <div className="flex items-center justify-center py-16">
          <span className="text-sm text-[var(--color-text-muted)]">科技系统开发中，敬请期待</span>
        </div>
      )}
    </div>
  )
}

// normalizeTab 规范化军事页签，旧增援入口默认落到战争模块。
function normalizeTab(value: string | null): MainTab {
  if (value === 'reinforcements') return 'war'
  return MAIN_TABS.includes(value as MainTab) ? value as MainTab : 'war'
}

export default MilitaryPage
