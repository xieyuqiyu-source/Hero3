import { type FC } from 'react'
import {
  Hammer,
  Swords,
  Shield,
  Crosshair,
  HardHat,
  Landmark,
  Wheat,
  Castle,
  Route,
  Eye,
  Store,
} from 'lucide-react'
import { useGameStore } from '@/store/gameStore'
import BuildingCard from './BuildingCard'
import type { Building } from '@/types/game'

const EMPTY_BUILDINGS: Building[] = []

interface BuildingConfig {
  type: string
  name: string
  description: string
  icon: FC<{ size?: number }>
  color: string
  bgColor: string
}

/** 军事类建筑 */
const MILITARY_BUILDINGS: BuildingConfig[] = [
  {
    type: 'infantry_camp',
    name: '步兵营',
    description: '提高步兵征兵速度',
    icon: Swords,
    color: 'text-red-600',
    bgColor: 'bg-red-50 dark:bg-red-950/20',
  },
  {
    type: 'cavalry_camp',
    name: '骑兵营',
    description: '提高骑兵征兵速度',
    icon: Shield,
    color: 'text-yellow-600',
    bgColor: 'bg-yellow-50 dark:bg-yellow-950/20',
  },
  {
    type: 'weapon_bureau',
    name: '兵器司',
    description: '攻击力加成',
    icon: Crosshair,
    color: 'text-orange-600',
    bgColor: 'bg-orange-50 dark:bg-orange-950/20',
  },
  {
    type: 'armor_bureau',
    name: '防具司',
    description: '防御力加成',
    icon: Hammer,
    color: 'text-zinc-600',
    bgColor: 'bg-zinc-50 dark:bg-zinc-950/20',
  },
]

/** 内政类建筑 */
const CIVIL_BUILDINGS: BuildingConfig[] = [
  {
    type: 'construction_bureau',
    name: '建造司',
    description: '提高建筑速度',
    icon: HardHat,
    color: 'text-sky-600',
    bgColor: 'bg-sky-50 dark:bg-sky-950/20',
  },
  {
    type: 'administration',
    name: '内政厅',
    description: '加成资源产量',
    icon: Landmark,
    color: 'text-purple-600',
    bgColor: 'bg-purple-50 dark:bg-purple-950/20',
  },
  {
    type: 'granary',
    name: '粮仓',
    description: '提高口粮上限',
    icon: Wheat,
    color: 'text-amber-600',
    bgColor: 'bg-amber-50 dark:bg-amber-950/20',
  },
  {
    type: 'relay_station',
    name: '驿站',
    description: '提高行军速度',
    icon: Route,
    color: 'text-teal-600',
    bgColor: 'bg-teal-50 dark:bg-teal-950/20',
  },
]

/** 防御类建筑 */
const DEFENSE_BUILDINGS: BuildingConfig[] = [
  {
    type: 'city_wall',
    name: '城墙',
    description: '提高城池防御',
    icon: Castle,
    color: 'text-stone-600',
    bgColor: 'bg-stone-50 dark:bg-stone-950/20',
  },
  {
    type: 'beacon_tower',
    name: '烽火台',
    description: '侦查来犯敌军兵力',
    icon: Eye,
    color: 'text-rose-600',
    bgColor: 'bg-rose-50 dark:bg-rose-950/20',
  },
  {
    type: 'market',
    name: '集市',
    description: '玩家间资源交易',
    icon: Store,
    color: 'text-emerald-600',
    bgColor: 'bg-emerald-50 dark:bg-emerald-950/20',
  },
]

interface BuildingGroupProps {
  title: string
  icon: FC<{ size?: number; className?: string }>
  configs: BuildingConfig[]
  buildings: Building[]
}

const BuildingGroup: FC<BuildingGroupProps> = ({ title, icon: GroupIcon, configs, buildings }) => (
  <section>
    <h2 className="text-sm font-semibold text-[var(--color-text-primary)] mb-3 flex items-center gap-2">
      <GroupIcon size={16} className="text-[var(--color-accent)]" />
      {title}
    </h2>
    <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
      {configs.map((config) => {
        const building = buildings.find((b) => b.type === config.type)
        const Icon = config.icon
        return (
          <BuildingCard
            key={config.type}
            buildingId={building?.id}
            icon={<Icon size={20} />}
            name={config.name}
            description={config.description}
            level={building?.level ?? 0}
            production={building ? `Lv.${building.level}` : '未建造'}
            upgradeEndsAt={building?.upgradeEndsAt}
            color={config.color}
            bgColor={config.bgColor}
            locked={!building}
          />
        )
      })}
    </div>
  </section>
)

const MilitaryTab: FC = () => {
  const buildings = useGameStore((s) => s.state?.buildings ?? EMPTY_BUILDINGS)

  return (
    <div className="space-y-6">
      <BuildingGroup title="军事" icon={Swords} configs={MILITARY_BUILDINGS} buildings={buildings} />
      <BuildingGroup title="内政" icon={Landmark} configs={CIVIL_BUILDINGS} buildings={buildings} />
      <BuildingGroup title="防御" icon={Shield} configs={DEFENSE_BUILDINGS} buildings={buildings} />
    </div>
  )
}

export default MilitaryTab
