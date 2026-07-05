// 城池军事建筑页签，按防御、军事、内政展示建筑卡片。
import { useEffect, useState, type FC } from 'react'
import {
  Hammer,
  Swords,
  Shield,
  Crosshair,
  HardHat,
  Landmark,
  Castle,
  Sparkles,
  Route,
  Eye,
  Store,
  Tent,
} from 'lucide-react'
import { useGameStore } from '@/store/gameStore'
import { getBuildingModifierProgressText, getCityWallDefenseBreakdownText, useConfigStore } from '@/store/configStore'
import BuildingCard from './BuildingCard'
import type { Building } from '@/types/game'

const EMPTY_BUILDINGS: Building[] = []

/** 千帐营每级对应的黄巾安全口粮上限，与后端黄巾配置保持一致。 */
const THOUSAND_TENT_CAPACITY_BY_LEVEL = [
  100000,
  300000,
  600000,
  1000000,
  2000000,
  4000000,
  8000000,
  15000000,
  30000000,
  50000000,
  80000000,
  120000000,
  180000000,
  280000000,
  400000000,
  600000000,
  900000000,
  1200000000,
  1600000000,
  2000000000,
]

interface BuildingConfig {
  type: string
  name: string
  description: string
  icon: FC<{ size?: number }>
  color: string
  bgColor: string
  goldBorder?: boolean
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
  {
    type: 'siege_camp',
    name: '攻城武器营',
    description: '金币强化攻城兵种征兵速度与消耗',
    icon: Castle,
    color: 'text-amber-500',
    bgColor: 'bg-amber-500/10',
    goldBorder: true,
  },
  {
    type: 'special_camp',
    name: '特殊建筑营',
    description: '金币强化特殊兵种征兵速度与消耗',
    icon: Sparkles,
    color: 'text-fuchsia-500',
    bgColor: 'bg-fuchsia-500/10',
    goldBorder: true,
  },
  {
    type: 'thousand_tent_camp',
    name: '千帐营',
    description: '金币提升黄巾口粮安全承载上限',
    icon: Tent,
    color: 'text-lime-600',
    bgColor: 'bg-lime-50 dark:bg-lime-950/20',
    goldBorder: true,
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
    type: 'market',
    name: '集市',
    description: '玩家间资源交易',
    icon: Store,
    color: 'text-emerald-600',
    bgColor: 'bg-emerald-50 dark:bg-emerald-950/20',
  },
  // 预留功能：粮仓暂时隐藏，后续接入口粮上限玩法后再恢复。
  // {
  //   type: 'granary',
  //   name: '粮仓',
  //   description: '提高口粮上限',
  //   icon: Wheat,
  //   color: 'text-amber-600',
  //   bgColor: 'bg-amber-50 dark:bg-amber-950/20',
  // },
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
]

interface BuildingGroupProps {
  title: string
  icon: FC<{ size?: number; className?: string }>
  configs: BuildingConfig[]
  buildings: Building[]
  faction?: string
  highlightedType?: string | null
}

/** 渲染单个建筑分组 */
const BuildingGroup: FC<BuildingGroupProps> = ({ title, icon: GroupIcon, configs, buildings, faction, highlightedType }) => (
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
          <div
            key={config.type}
            id={`city-building-${config.type}`}
            className={`
              rounded-2xl transition-all duration-300
              ${highlightedType === config.type
                ? 'ring-2 ring-sky-400 ring-offset-2 ring-offset-[var(--color-bg)] animate-pulse'
                : config.goldBorder
                  ? 'ring-2 ring-amber-400/45 ring-offset-1 ring-offset-[var(--color-bg)]'
                  : ''
              }
            `}
          >
            <BuildingCard
              buildingId={building?.id}
              buildingType={config.type}
              icon={<Icon size={20} />}
              name={config.name}
              description={config.description}
              level={building?.level ?? 0}
              production={building ? `Lv.${building.level}` : '未建造'}
              effectText={building ? buildingEffectText(config.type, building.level, faction) : undefined}
              upgradeEndsAt={building?.upgradeEndsAt}
              color={config.color}
              bgColor={config.bgColor}
              locked={!building}
            />
          </div>
        )
      })}
    </div>
  </section>
)

interface MilitaryTabProps {
  focusBuildingType?: string | null
  focusBuildingNonce?: number
}

/** 渲染军事建筑页签 */
const MilitaryTab: FC<MilitaryTabProps> = ({ focusBuildingType = null, focusBuildingNonce = 0 }) => {
  const buildings = useGameStore((s) => s.state?.buildings ?? EMPTY_BUILDINGS)
  const faction = useGameStore((s) => s.state?.player.faction)
  const [highlightedType, setHighlightedType] = useState<string | null>(null)
  useConfigStore((s) => Boolean(s.balance && s.combat))

  useEffect(() => {
    if (focusBuildingNonce <= 0 || !focusBuildingType) return

    setHighlightedType(focusBuildingType)
    window.setTimeout(() => {
      document.getElementById(`city-building-${focusBuildingType}`)?.scrollIntoView({
        behavior: 'smooth',
        block: 'center',
      })
    }, 80)
    const timer = window.setTimeout(() => setHighlightedType(null), 1800)
    return () => window.clearTimeout(timer)
  }, [focusBuildingNonce, focusBuildingType])

  return (
    <div className="space-y-6">
      <BuildingGroup title="防御" icon={Shield} configs={DEFENSE_BUILDINGS} buildings={buildings} faction={faction} highlightedType={highlightedType} />
      <BuildingGroup title="军事" icon={Swords} configs={MILITARY_BUILDINGS} buildings={buildings} faction={faction} highlightedType={highlightedType} />
      <BuildingGroup title="内政" icon={Landmark} configs={CIVIL_BUILDINGS} buildings={buildings} faction={faction} highlightedType={highlightedType} />
    </div>
  )
}

export default MilitaryTab

/** 返回建筑卡片展示的效果文本。 */
function buildingEffectText(buildingType: string, level: number, faction?: string): string {
  if (buildingType === 'city_wall') {
    return getCityWallDefenseBreakdownText(level, faction)
  }
  if (buildingType === 'thousand_tent_camp') {
    return thousandTentCampEffectText(level)
  }
  return getBuildingModifierProgressText(buildingType, level)
}

/** 返回千帐营当前等级到下一级的安全口粮上限。 */
function thousandTentCampEffectText(level: number): string {
  const current = THOUSAND_TENT_CAPACITY_BY_LEVEL[Math.max(0, level - 1)] ?? THOUSAND_TENT_CAPACITY_BY_LEVEL[0]
  const next = THOUSAND_TENT_CAPACITY_BY_LEVEL[level]
  if (!next) return `安全口粮 ${formatYellowTurbanCapacity(current)}`
  return `安全口粮 ${formatYellowTurbanCapacity(current)} -> ${formatYellowTurbanCapacity(next)}`
}

/** 格式化黄巾口粮上限数字。 */
function formatYellowTurbanCapacity(value: number): string {
  if (value >= 100000000) return `${Number((value / 100000000).toFixed(1))}亿`
  if (value >= 10000) return `${Number((value / 10000).toFixed(1))}万`
  return String(value)
}
