import { type FC } from 'react'
import { Swords, Target, Shield, Hammer } from 'lucide-react'
import { useGameStore } from '@/store/gameStore'
import BuildingCard from './BuildingCard'
import type { Building } from '@/types/game'

const EMPTY_BUILDINGS: Building[] = []

const MilitaryTab: FC = () => {
  const buildings = useGameStore((s) => s.state?.buildings ?? EMPTY_BUILDINGS)

  const barracks = buildings.find((b) => b.type === 'barracks')
  const archeryRange = buildings.find((b) => b.type === 'archery_range')
  const stable = buildings.find((b) => b.type === 'stable')
  const blacksmith = buildings.find((b) => b.type === 'blacksmith')

  return (
    <div className="space-y-6">
      <section>
        <h2 className="text-sm font-semibold text-[var(--color-text-primary)] mb-3 flex items-center gap-2">
          <Swords size={16} className="text-[var(--color-accent)]" />
          军事建筑
        </h2>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <BuildingCard
            buildingId={barracks?.id}
            icon={<Swords size={20} />}
            name="兵营"
            description="训练步兵"
            level={barracks?.level ?? 0}
            production={barracks ? `步兵 Lv.${barracks.level}` : '未解锁'}
            upgradeEndsAt={barracks?.upgradeEndsAt}
            color="text-red-600"
            bgColor="bg-red-50 dark:bg-red-950/20"
            locked={!barracks}
          />
          <BuildingCard
            buildingId={archeryRange?.id}
            icon={<Target size={20} />}
            name="射箭场"
            description="训练弓兵"
            level={archeryRange?.level ?? 0}
            production={archeryRange ? `弓兵 Lv.${archeryRange.level}` : '未解锁'}
            upgradeEndsAt={archeryRange?.upgradeEndsAt}
            color="text-sky-600"
            bgColor="bg-sky-50 dark:bg-sky-950/20"
            locked={!archeryRange}
          />
          <BuildingCard
            buildingId={stable?.id}
            icon={<Shield size={20} />}
            name="马厩"
            description="训练骑兵"
            level={stable?.level ?? 0}
            production={stable ? `骑兵 Lv.${stable.level}` : '未解锁'}
            upgradeEndsAt={stable?.upgradeEndsAt}
            color="text-yellow-600"
            bgColor="bg-yellow-50 dark:bg-yellow-950/20"
            locked={!stable}
          />
          <BuildingCard
            buildingId={blacksmith?.id}
            icon={<Hammer size={20} />}
            name="铁匠铺"
            description="提升军队攻防加成"
            level={blacksmith?.level ?? 0}
            production={blacksmith ? `Lv.${blacksmith.level}` : '未解锁'}
            upgradeEndsAt={blacksmith?.upgradeEndsAt}
            color="text-zinc-600"
            bgColor="bg-zinc-50 dark:bg-zinc-950/20"
            locked={!blacksmith}
          />
        </div>
      </section>
    </div>
  )
}

export default MilitaryTab
