import { useState, type FC } from 'react'
import { Swords, Shield, Castle, Star, Gem, Sparkles } from 'lucide-react'
import { useGameStore } from '@/store/gameStore'
import { getBuildingModifierProgressText, useConfigStore, type UnitConfig } from '@/store/configStore'
import UnitCard from './UnitCard'
import RecruitModal from './RecruitModal'
import RecruitQueuePanel from './RecruitQueuePanel'
import BuildingCard from '@/pages/city/components/BuildingCard'
import type { ArmyUnit } from '@/types/game'

type UnitCategory = 'infantry' | 'cavalry' | 'siege' | 'special'

const CATEGORIES = [
  { key: 'infantry' as const, label: '步兵', icon: Swords },
  { key: 'cavalry' as const, label: '骑兵', icon: Shield },
  { key: 'siege' as const, label: '攻城', icon: Castle },
  { key: 'special' as const, label: '特殊', icon: Star },
]

const EMPTY_ARMY: ArmyUnit[] = []
const SPECIAL_BUILDINGS = [
  {
    type: 'siege_camp',
    name: '攻城武器营',
    description: '金币强化攻城兵种征兵速度与消耗',
    icon: Castle,
    color: 'text-amber-500',
    bgColor: 'bg-amber-500/10',
  },
  {
    type: 'special_camp',
    name: '特殊建筑营',
    description: '金币强化特殊兵种征兵速度与消耗',
    icon: Sparkles,
    color: 'text-fuchsia-500',
    bgColor: 'bg-fuchsia-500/10',
  },
]

const RecruitTab: FC = () => {
  const [category, setCategory] = useState<UnitCategory>('infantry')
  const [selectedUnit, setSelectedUnit] = useState<{ id: string; config: UnitConfig } | null>(null)
  const faction = useGameStore((s) => s.state?.player.faction ?? 'wei')
  const army = useGameStore((s) => s.state?.army ?? EMPTY_ARMY)
  const buildings = useGameStore((s) => s.state?.buildings ?? [])
  const units = useConfigStore((s) => s.units)

  // 获取当前阵营的兵种配置
  const factionUnits = units?.[faction] ?? {}

  // 按分类过滤，按训练时间排序（弱→强）
  const filteredUnits = Object.entries(factionUnits)
    .filter(([, config]) => config.category === category)
    .sort((a, b) => a[1].trainSeconds - b[1].trainSeconds)

  // 获取当前拥有数量
  const getOwnedAmount = (unitId: string): number => {
    const unit = army.find((a) => a.unitType === unitId)
    return unit?.amount ?? 0
  }

  return (
    <div className="space-y-4">
      {/* Recruit Queue */}
      <RecruitQueuePanel />

      <section className="rounded-2xl border border-amber-400/35 bg-amber-400/5 p-3">
        <div className="mb-3 flex items-center gap-2">
          <Gem size={15} className="text-amber-500" />
          <h2 className="text-sm font-bold text-[var(--color-text-primary)]">金币训练强化</h2>
        </div>
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          {SPECIAL_BUILDINGS.map((config) => {
            const building = buildings.find((item) => item.type === config.type)
            const Icon = config.icon
            return (
              <div key={config.type} className="rounded-2xl ring-2 ring-amber-400/45 ring-offset-1 ring-offset-[var(--color-bg)]">
                <BuildingCard
                  buildingId={building?.id}
                  buildingType={config.type}
                  icon={<Icon size={20} />}
                  name={config.name}
                  description={config.description}
                  level={building?.level ?? 0}
                  production={building ? `Lv.${building.level}` : '未建造'}
                  effectText={building ? getBuildingModifierProgressText(config.type, building.level) : undefined}
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

      {/* Category Sub-tabs */}
      <div className="flex gap-1.5">
        {CATEGORIES.map((cat) => {
          const Icon = cat.icon
          const isActive = category === cat.key
          return (
            <button
              key={cat.key}
              type="button"
              onClick={() => setCategory(cat.key)}
              className={`
                flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium cursor-pointer
                transition-all duration-200
                ${isActive
                  ? 'bg-[var(--color-accent-light)] text-[var(--color-accent)] border border-[var(--color-accent-border)]'
                  : 'bg-[var(--color-surface-dim)] text-[var(--color-text-secondary)] border border-[var(--color-border)] hover:text-[var(--color-text-primary)]'
                }
              `}
            >
              <Icon size={12} />
              {cat.label}
            </button>
          )
        })}
      </div>

      {/* Unit Cards - Grid */}
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3">
        {!units && (
          <div className="col-span-full flex items-center justify-center py-12">
            <span className="text-sm text-[var(--color-text-muted)]">正在加载兵种配置...</span>
          </div>
        )}
        {units && filteredUnits.length === 0 && (
          <div className="col-span-full flex items-center justify-center py-12">
            <span className="text-sm text-[var(--color-text-muted)]">暂无该类型兵种</span>
          </div>
        )}
        {filteredUnits.map(([unitId, config]) => (
          <UnitCard
            key={unitId}
            unitId={unitId}
            config={config}
            owned={getOwnedAmount(unitId)}
            onClick={() => setSelectedUnit({ id: unitId, config })}
          />
        ))}
      </div>

      {/* Recruit Modal */}
      {selectedUnit && (
        <RecruitModal
          open={selectedUnit !== null}
          onClose={() => setSelectedUnit(null)}
          unitId={selectedUnit.id}
          config={selectedUnit.config}
          owned={getOwnedAmount(selectedUnit.id)}
        />
      )}
    </div>
  )
}

export default RecruitTab
