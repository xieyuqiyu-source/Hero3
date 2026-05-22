import { type FC } from 'react'
import { Swords, Target, Shield, Hammer } from 'lucide-react'
import BuildingCard from './BuildingCard'

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

export default MilitaryTab
