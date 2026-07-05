// 城池页面，管理资源建筑和军事建筑两个页签的联动。
import { useEffect, useState, type FC } from 'react'
import { useLocation } from 'react-router-dom'
import ResourceTab from './components/ResourceTab'
import MilitaryTab from './components/MilitaryTab'
import { useGameStore } from '@/store/gameStore'

type Tab = 'resource' | 'military'

interface CityRouteState {
  tab?: Tab
  focusBuildingType?: string
}

const CityPage: FC = () => {
  const [activeTab, setActiveTab] = useState<Tab>('resource')
  const [resourceExpanded, setResourceExpanded] = useState(true)
  const [focusBuildingType, setFocusBuildingType] = useState<string | null>(null)
  const [focusBuildingNonce, setFocusBuildingNonce] = useState(0)
  const loadCityView = useGameStore((s) => s.loadCityView)
  const location = useLocation()

  useEffect(() => {
    void loadCityView()
  }, [loadCityView])

  useEffect(() => {
    const state = location.state as CityRouteState | null
    if (!state?.focusBuildingType && state?.tab !== 'military') return

    setActiveTab('military')
    if (state.focusBuildingType) {
      setFocusBuildingType(state.focusBuildingType)
      setFocusBuildingNonce((value) => value + 1)
    }
  }, [location.key, location.state])

  /** 跳转到军事建筑并定位指定建筑 */
  const focusMilitaryBuilding = (buildingType: string) => {
    setActiveTab('military')
    setFocusBuildingType(buildingType)
    setFocusBuildingNonce((value) => value + 1)
  }

  /** 跳转到军事建筑并定位建造司 */
  const handleFocusConstructionBureau = () => {
    focusMilitaryBuilding('construction_bureau')
  }

  return (
    <div>

      {/* Tab Switcher */}
      <div className="grid grid-cols-2 gap-1 p-1 rounded-xl bg-[var(--color-surface-dim)] border border-[var(--color-border)] w-full max-w-full sm:flex sm:w-fit mb-6">
        <button
          type="button"
          onClick={() => setActiveTab('resource')}
          className={`
            px-4 py-2 rounded-lg text-sm font-medium cursor-pointer w-full sm:w-auto
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
            px-4 py-2 rounded-lg text-sm font-medium cursor-pointer w-full sm:w-auto
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
        <ResourceTab
          expanded={resourceExpanded}
          onToggle={() => setResourceExpanded(!resourceExpanded)}
          onRequestNewResourceSlot={handleFocusConstructionBureau}
        />
      ) : (
        <MilitaryTab focusBuildingType={focusBuildingType} focusBuildingNonce={focusBuildingNonce} />
      )}
    </div>
  )
}

export default CityPage
