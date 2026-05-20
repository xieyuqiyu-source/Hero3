import { useState, useEffect, type FC } from 'react'
import {
  TreePine,
  Mountain,
  Gem,
  Wheat,
  Warehouse,
  TrendingUp,
  Swords,
  Shield,
  Target,
  Hammer,
  ArrowUpCircle,
  ChevronDown,
  ChevronRight,
  Plus,
} from 'lucide-react'

type Tab = 'resource' | 'military'

const CityPage: FC = () => {
  const [activeTab, setActiveTab] = useState<Tab>('resource')
  const [resourceExpanded, setResourceExpanded] = useState(true)

  return (
    <div>
      {/* Resource Summary Bar - always floating, no background */}
      <ResourceBar />

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

/* ===== Resource Summary Bar ===== */
const ResourceBar: FC = () => {
  const [scrolled, setScrolled] = useState(false)

  useEffect(() => {
    const handleScroll = () => {
      setScrolled(window.scrollY > 20)
    }
    window.addEventListener('scroll', handleScroll, { passive: true })
    return () => window.removeEventListener('scroll', handleScroll)
  }, [])

  const resources = [
    { name: '木材', icon: TreePine, value: 1200, capacity: 5000, color: 'text-green-600' },
    { name: '石料', icon: Mountain, value: 800, capacity: 5000, color: 'text-slate-600' },
    { name: '铁矿', icon: Gem, value: 500, capacity: 5000, color: 'text-orange-600' },
    { name: '粮食', icon: Wheat, value: 2000, capacity: 5000, color: 'text-amber-600' },
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

/* ===== Resource Buildings Tab ===== */
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

      {/* 4-column resource grid - desktop shows all, mobile collapses per group */}
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

/* ===== Resource Slot ===== */
interface ResourceSlotProps {
  index: number
  level: number
  color: string
  bgColor: string
}

const ResourceSlot: FC<ResourceSlotProps> = ({ index, level, color, bgColor }) => {
  return (
    <div className="flex items-center justify-between px-3 py-2 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] hover:shadow-[0_2px_8px_rgba(15,23,42,0.04)] transition-shadow duration-150">
      <div className="flex items-center gap-2">
        <div className={`w-5 h-5 rounded-md ${bgColor} flex items-center justify-center`}>
          <span className={`text-[10px] font-bold ${color}`}>{index}</span>
        </div>
        <span className="text-xs text-[var(--color-text-primary)]">Lv.{level}</span>
      </div>
      <button
        type="button"
        className="
          flex items-center gap-1 px-2 py-1 rounded-lg
          text-[10px] font-medium text-[var(--color-accent)]
          bg-[var(--color-accent-light)]
          hover:border-[var(--color-accent-border)]
          cursor-pointer transition-all duration-200
        "
      >
        <ArrowUpCircle size={10} />
        升级
      </button>
    </div>
  )
}

/* ===== Military Buildings Tab ===== */
const MilitaryTab: FC = () => {
  return (
    <div className="space-y-6">
      <section>
        <h2 className="text-sm font-semibold text-[var(--color-text-primary)] mb-3 flex items-center gap-2">
          <Swords size={16} className="text-[var(--color-accent)]" />
          军事建筑
        </h2>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <BuildingCard
            icon={<Swords size={20} />}
            name="兵营"
            description="训练步兵"
            level={1}
            production="步兵 Lv.1"
            color="text-red-600"
            bgColor="bg-red-50 dark:bg-red-950/20"
          />
          <BuildingCard
            icon={<Target size={20} />}
            name="射箭场"
            description="训练弓兵"
            level={1}
            production="弓兵 Lv.1"
            color="text-sky-600"
            bgColor="bg-sky-50 dark:bg-sky-950/20"
          />
          <BuildingCard
            icon={<Shield size={20} />}
            name="马厩"
            description="训练骑兵"
            level={0}
            production="未解锁"
            color="text-yellow-600"
            bgColor="bg-yellow-50 dark:bg-yellow-950/20"
            locked
          />
          <BuildingCard
            icon={<Hammer size={20} />}
            name="铁匠铺"
            description="提升军队攻防加成"
            level={0}
            production="未解锁"
            color="text-zinc-600"
            bgColor="bg-zinc-50 dark:bg-zinc-950/20"
            locked
          />
        </div>
      </section>
    </div>
  )
}

/* ===== Building Card Component ===== */
interface BuildingCardProps {
  icon: React.ReactNode
  name: string
  description: string
  level: number
  production: string
  color: string
  bgColor: string
  locked?: boolean
}

const BuildingCard: FC<BuildingCardProps> = ({
  icon,
  name,
  description,
  level,
  production,
  color,
  bgColor,
  locked,
}) => {
  return (
    <div className={`
      relative rounded-2xl p-4 border border-[var(--color-border)]
      bg-[var(--color-surface)]
      transition-all duration-200
      ${locked ? 'opacity-60' : 'hover:shadow-[0_8px_24px_rgba(15,23,42,0.06)] hover:-translate-y-0.5'}
    `}>
      <div className="flex items-start gap-3">
        <div className={`w-10 h-10 rounded-xl ${bgColor} flex items-center justify-center ${color} flex-shrink-0`}>
          {icon}
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold text-[var(--color-text-primary)]">{name}</h3>
            <span className="text-xs text-[var(--color-text-muted)]">
              {locked ? '未解锁' : `Lv.${level}`}
            </span>
          </div>
          <p className="text-xs text-[var(--color-text-secondary)] mt-0.5">{description}</p>
          <div className="flex items-center justify-between mt-2">
            <span className={`text-xs font-medium ${locked ? 'text-[var(--color-text-muted)]' : color}`}>
              {production}
            </span>
            {!locked && (
              <button
                type="button"
                className="
                  flex items-center gap-1 px-2.5 py-1 rounded-lg
                  text-xs font-medium text-[var(--color-accent)]
                  bg-[var(--color-accent-light)] border border-transparent
                  hover:border-[var(--color-accent-border)]
                  cursor-pointer transition-all duration-200
                "
              >
                <ArrowUpCircle size={12} />
                升级
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

export default CityPage
