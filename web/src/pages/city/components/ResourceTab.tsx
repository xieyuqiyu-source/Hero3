import { type FC } from 'react'
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
} from 'lucide-react'
import ResourceSlot from './ResourceSlot'
import BuildingCard from './BuildingCard'

interface ResourceTabProps {
  expanded: boolean
  onToggle: () => void
}

const ResourceTab: FC<ResourceTabProps> = ({ expanded, onToggle }) => {
  const resourceGroups = [
    {
      key: 'wood',
      name: '木场',
      icon: TreePine,
      color: 'text-green-600',
      bgColor: 'bg-green-50 dark:bg-green-950/20',
      slots: [
        { level: 3 },
        { level: 2 },
        { level: 1 },
        { level: 1 },
        { level: 1 },
      ],
    },
    {
      key: 'stone',
      name: '采石场',
      icon: Mountain,
      color: 'text-slate-600',
      bgColor: 'bg-slate-50 dark:bg-slate-950/20',
      slots: [
        { level: 2 },
        { level: 2 },
        { level: 1 },
        { level: 1 },
        { level: 1 },
      ],
    },
    {
      key: 'iron',
      name: '铁矿',
      icon: Gem,
      color: 'text-orange-600',
      bgColor: 'bg-orange-50 dark:bg-orange-950/20',
      slots: [
        { level: 2 },
        { level: 1 },
        { level: 1 },
        { level: 1 },
        { level: 1 },
      ],
    },
    {
      key: 'food',
      name: '农田',
      icon: Wheat,
      color: 'text-amber-600',
      bgColor: 'bg-amber-50 dark:bg-amber-950/20',
      slots: [
        { level: 3 },
        { level: 2 },
        { level: 2 },
        { level: 1 },
        { level: 1 },
      ],
    },
  ]

  return (
    <div className="space-y-4">
      {/* Section Header with expand/collapse */}
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

      {/* 4-column resource grid */}
      <div className={`
        transition-all duration-300 ease-in-out overflow-hidden
        ${expanded ? 'max-h-[2000px] opacity-100' : 'max-h-0 opacity-0'}
      `}>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          {resourceGroups.map((group) => {
            const Icon = group.icon
            return (
              <div key={group.key} className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] overflow-hidden">
                {/* Column Header */}
                <div className="flex items-center gap-2 px-3 py-2.5 border-b border-[var(--color-border)]">
                  <div className={`w-7 h-7 rounded-lg ${group.bgColor} flex items-center justify-center ${group.color} flex-shrink-0`}>
                    <Icon size={14} />
                  </div>
                  <span className="text-sm font-semibold text-[var(--color-text-primary)] flex-1">{group.name}</span>
                  <span className="text-[10px] text-[var(--color-text-muted)]">{group.slots.length} 块</span>
                </div>

                {/* 5 Slots */}
                <div className="px-2 py-2 space-y-1.5">
                  {group.slots.map((slot, i) => (
                    <ResourceSlot
                      key={i}
                      index={i + 1}
                      level={slot.level}
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
            icon={<Warehouse size={20} />}
            name="仓库"
            description="提升资源容量上限"
            level={1}
            production="容量 5000"
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
