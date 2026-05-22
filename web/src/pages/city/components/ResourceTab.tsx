import { useState, type FC } from 'react'
import {
  TreePine,
  Mountain,
  Gem,
  Wheat,
  Warehouse,
  TrendingUp,
  ChevronDown,
  ChevronRight,
  Plus,
  ArrowUpCircle,
  LoaderCircle,
} from 'lucide-react'
import { useGameStore } from '@/store/gameStore'
import { useProjectedResources } from '@/hooks/useProjectedResources'
import { gameApi } from '@/api/game'
import { toast } from '@/components/ui'
import { getProductionAtLevel } from '../data/buildingConfig'
import ResourceSlot from './ResourceSlot'
import BuildingCard from './BuildingCard'
import type { Building } from '@/types/game'

const EMPTY_BUILDINGS: Building[] = []

/** 资源建筑分组配置 */
const RESOURCE_GROUPS = [
  {
    key: 'wood',
    type: 'wood_camp',
    name: '木场',
    icon: TreePine,
    color: 'text-green-600',
    bgColor: 'bg-green-50 dark:bg-green-950/20',
  },
  {
    key: 'stone',
    type: 'stone_quarry',
    name: '采石场',
    icon: Mountain,
    color: 'text-slate-600',
    bgColor: 'bg-slate-50 dark:bg-slate-950/20',
  },
  {
    key: 'iron',
    type: 'iron_mine',
    name: '铁矿',
    icon: Gem,
    color: 'text-orange-600',
    bgColor: 'bg-orange-50 dark:bg-orange-950/20',
  },
  {
    key: 'food',
    type: 'farm',
    name: '农田',
    icon: Wheat,
    color: 'text-amber-600',
    bgColor: 'bg-amber-50 dark:bg-amber-950/20',
  },
] as const

/** 按 type 过滤建筑列表 */
function filterBuildings(buildings: Building[], type: string): Building[] {
  return buildings.filter((b) => b.type === type)
}

interface ResourceTabProps {
  expanded: boolean
  onToggle: () => void
}

const ResourceTab: FC<ResourceTabProps> = ({ expanded, onToggle }) => {
  const buildings = useGameStore((s) => s.state?.buildings ?? EMPTY_BUILDINGS)
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const setState = useGameStore((s) => s.setState)
  const resources = useProjectedResources()
  const [batchLoading, setBatchLoading] = useState(false)

  // 找仓库建筑
  const warehouse = buildings.find((b) => b.type === 'warehouse')
  const warehouseLevel = warehouse?.level ?? 0
  const warehouseCapacity = resources?.capacity.wood ?? 5000

  const handleBatchUpgrade = async () => {
    if (!activePlayerId || batchLoading) return
    setBatchLoading(true)
    try {
      const result = await gameApi.upgradeBuildingBatch(activePlayerId)
      setState(result.state)
      toast.success(`成功升级 ${result.upgraded} 块田地`)
    } catch {
      // 错误已由全局拦截器处理
    } finally {
      setBatchLoading(false)
    }
  }

  return (
    <div className="space-y-4">
      {/* Section Header with expand/collapse */}
      <div className="flex items-center gap-2">
        <button
          type="button"
          onClick={onToggle}
          className="flex items-center gap-2 cursor-pointer group"
        >
          <TreePine size={16} className="text-[var(--color-accent)]" />
          <span className="text-sm font-semibold text-[var(--color-text-primary)]">资源建筑</span>
          {expanded ? (
            <ChevronDown size={14} className="text-[var(--color-text-muted)] group-hover:text-[var(--color-accent)] transition-colors" />
          ) : (
            <ChevronRight size={14} className="text-[var(--color-text-muted)] group-hover:text-[var(--color-accent)] transition-colors" />
          )}
        </button>
        <button
          type="button"
          onClick={handleBatchUpgrade}
          disabled={batchLoading}
          className="
            ml-auto flex items-center gap-1 px-2.5 py-1 rounded-lg
            text-[10px] font-semibold
            bg-[var(--color-accent-light)] text-[var(--color-accent)]
            border border-[var(--color-accent-border)]
            hover:opacity-80 cursor-pointer transition-opacity
            disabled:opacity-50 disabled:cursor-not-allowed
          "
        >
          {batchLoading ? <LoaderCircle size={11} className="animate-spin" /> : <ArrowUpCircle size={11} />}
          一键升级
        </button>
      </div>

      {/* 4-column resource grid */}
      <div className={`
        transition-all duration-300 ease-in-out overflow-hidden
        ${expanded ? 'max-h-[2000px] opacity-100' : 'max-h-0 opacity-0'}
      `}>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          {RESOURCE_GROUPS.map((group) => {
            const Icon = group.icon
            const slots = filterBuildings(buildings, group.type)
            const totalProduction = slots.reduce((sum, b) => sum + getProductionAtLevel(b.type, b.level), 0)
            return (
              <div key={group.key} className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] overflow-hidden">
                {/* Column Header */}
                <div className="flex items-center gap-2 px-3 py-2.5 border-b border-[var(--color-border)]">
                  <div className={`w-7 h-7 rounded-lg ${group.bgColor} flex items-center justify-center ${group.color} flex-shrink-0`}>
                    <Icon size={14} />
                  </div>
                  <span className="text-sm font-semibold text-[var(--color-text-primary)] flex-1">{group.name}</span>
                  <span className="text-[10px] text-amber-500 font-semibold">+{totalProduction}/h</span>
                </div>

                {/* Slots */}
                <div className="px-2 py-2 space-y-1.5">
                  {slots.map((slot, i) => (
                    <ResourceSlot
                      key={slot.id}
                      buildingId={slot.id}
                      buildingType={slot.type}
                      index={i + 1}
                      level={slot.level}
                      production={getProductionAtLevel(slot.type, slot.level)}
                      upgradeEndsAt={slot.upgradeEndsAt}
                      color={group.color}
                      bgColor={group.bgColor}
                    />
                  ))}

                  {/* Add more slot button */}
                  <button
                    type="button"
                    className="
                      w-full flex items-center justify-center gap-1 px-3 py-2 rounded-xl
                      border border-dashed border-[var(--color-border)]
                      text-[11px] font-medium text-[var(--color-text-muted)]
                      hover:border-[var(--color-accent-border)] hover:text-[var(--color-accent)]
                      cursor-pointer transition-all duration-200
                    "
                  >
                    <Plus size={12} />
                    开拓新田地
                  </button>
                </div>
              </div>
            )
          })}
        </div>
      </div>

      {/* Bonus Buildings */}
      <section>
        <h2 className="text-sm font-semibold text-[var(--color-text-primary)] mb-3 flex items-center gap-2">
          <TrendingUp size={16} className="text-[var(--color-accent)]" />
          加成建筑
        </h2>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <BuildingCard
            buildingId={warehouse?.id}
            icon={<Warehouse size={20} />}
            name="仓库"
            description="提升资源容量上限"
            level={warehouseLevel}
            production={`容量 ${warehouseCapacity.toLocaleString()}`}
            upgradeEndsAt={warehouse?.upgradeEndsAt}
            color="text-indigo-600"
            bgColor="bg-indigo-50 dark:bg-indigo-950/20"
          />
          <BuildingCard
            icon={<TrendingUp size={20} />}
            name="市集"
            description="提升全资源产出加成"
            level={0}
            production="未解锁"
            color="text-purple-600"
            bgColor="bg-purple-50 dark:bg-purple-950/20"
            locked
          />
        </div>
      </section>
    </div>
  )
}

export default ResourceTab
