import { useState, type FC } from 'react'
import { Swords, Shield, Castle, Star } from 'lucide-react'
import { useGameStore } from '@/store/gameStore'
import { useConfigStore, type UnitConfig } from '@/store/configStore'
import UnitCard from './UnitCard'
import RecruitModal from './RecruitModal'
import type { ArmyUnit } from '@/types/game'

type UnitCategory = 'infantry' | 'cavalry' | 'siege' | 'special'

const CATEGORIES = [
  { key: 'infantry' as const, label: '步兵', icon: Swords },
  { key: 'cavalry' as const, label: '骑兵', icon: Shield },
  { key: 'siege' as const, label: '攻城', icon: Castle },
  { key: 'special' as const, label: '特殊', icon: Star },
]

const EMPTY_ARMY: ArmyUnit[] = []

const RecruitTab: FC = () => {
  const [category, setCategory] = useState<UnitCategory>('infantry')
  const [selectedUnit, setSelectedUnit] = useState<{ id: string; config: UnitConfig } | null>(null)
  const faction = useGameStore((s) => s.state?.player.faction ?? 'wei')
  const army = useGameStore((s) => s.state?.army ?? EMPTY_ARMY)
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
