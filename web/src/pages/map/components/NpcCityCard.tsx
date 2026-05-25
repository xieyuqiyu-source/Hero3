import { useState, type FC } from 'react'
import { LoaderCircle, CircleCheck, Swords, ShieldAlert, Search } from 'lucide-react'
import { useGameStore } from '@/store/gameStore'
import { gameApi } from '@/api/game'
import { FACTION_LABELS, FACTION_COLORS } from '@/utils/faction'
import type { NpcCity, BattleReport } from '@/types/game'

interface NpcCityCardProps {
  city: NpcCity
  selected: boolean
  onClick: () => void
  onBattleResult: (report: BattleReport) => void
  onScoutResult: (report: BattleReport) => void
}

const TIER_CONFIG = {
  small:  { label: '小型', color: 'text-slate-500', border: 'border-slate-300 dark:border-slate-600', bg: 'bg-slate-50 dark:bg-slate-800/30' },
  medium: { label: '中型', color: 'text-blue-500', border: 'border-blue-300 dark:border-blue-600', bg: 'bg-blue-50 dark:bg-blue-900/20' },
  large:  { label: '大型', color: 'text-purple-500', border: 'border-purple-300 dark:border-purple-600', bg: 'bg-purple-50 dark:bg-purple-900/20' },
  golden: { label: '金色', color: 'text-amber-500', border: 'border-amber-300 dark:border-amber-500', bg: 'bg-amber-50 dark:bg-amber-900/20' },
}

function isRecovering(city: NpcCity): boolean {
  for (const [res, cap] of Object.entries(city.storageCapacity)) {
    if ((city.resources[res] ?? 0) < cap) return true
  }
  for (const maxUnit of city.maxArmy) {
    const current = city.army.find(u => u.unitType === maxUnit.unitType)
    if (!current || current.amount < maxUnit.amount) return true
  }
  return false
}

const NpcCityCard: FC<NpcCityCardProps> = ({ city, selected, onClick, onBattleResult, onScoutResult }) => {
  const tier = TIER_CONFIG[city.tier] ?? TIER_CONFIG.small
  const recovering = isRecovering(city)
  const [busy, setBusy] = useState<string | null>(null)
  const activeTraits = city.traits.filter(t => t.id !== 'none')

  const handleQuickAction = async (e: React.MouseEvent, mode: 'attack' | 'plunder' | 'scout') => {
    e.stopPropagation()
    const playerId = useGameStore.getState().activePlayerId
    const army = useGameStore.getState().state?.army ?? []
    if (!playerId || busy) return

    if (mode === 'scout') {
      setBusy('scout')
      try {
        const result = await gameApi.scoutNpc(playerId, city.id)
        useGameStore.getState().setState(result.state)
        onScoutResult(result.battleReport)
      } catch { /* global handler */ }
      finally { setBusy(null) }
      return
    }

    const units: Record<string, number> = {}
    for (const u of army) {
      if (u.amount > 0) units[u.unitType] = u.amount
    }
    if (Object.keys(units).length === 0) return

    setBusy(mode)
    try {
      const result = await gameApi.attackNpc(playerId, city.id, mode, units)
      useGameStore.getState().setState(result.state)
      onBattleResult(result.battleReport)
    } catch { /* global handler */ }
    finally { setBusy(null) }
  }

  return (
    <div
      className={`
        rounded-2xl border p-3 transition-all duration-200
        ${selected
          ? 'border-[var(--color-accent)] bg-[var(--color-accent-light)] shadow-md'
          : `${tier.border} ${tier.bg} hover:border-[var(--color-accent-border)] hover:shadow-sm`
        }
      `}
    >
      {/* Header */}
      <button type="button" onClick={onClick} className="w-full text-left cursor-pointer">
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-sm font-bold text-[var(--color-text-primary)]">{city.name}</span>
          <span className={`text-[10px] font-semibold px-1.5 py-0.5 rounded ${tier.color} bg-white/60 dark:bg-white/10`}>
            {tier.label}
          </span>
          <span className={`text-[10px] font-bold ${FACTION_COLORS[city.faction] ?? 'text-[var(--color-text-muted)]'}`}>
            {FACTION_LABELS[city.faction] ?? city.faction}
          </span>
          {activeTraits.map((trait) => (
            <span
              key={trait.id}
              className="text-[9px] px-1.5 py-0.5 rounded bg-amber-500/10 text-amber-600 font-medium"
            >
              {trait.name}
            </span>
          ))}
          <span className="ml-auto flex-shrink-0">
            {recovering ? (
              <span title="恢复中">
                <LoaderCircle size={13} className="text-amber-500 animate-spin" />
              </span>
            ) : (
              <span title="完整状态">
                <CircleCheck size={13} className="text-green-500" />
              </span>
            )}
          </span>
        </div>
      </button>

      {/* Quick Actions: 侦查 掠夺 攻击 */}
      <div className="flex gap-1.5 mt-2 pt-2 border-t border-[var(--color-border)]">
        <button
          type="button"
          onClick={(e) => handleQuickAction(e, 'scout')}
          disabled={busy !== null}
          className="flex-1 flex items-center justify-center gap-1 px-2 py-1.5 rounded-lg text-[10px] font-medium bg-blue-500/10 text-blue-600 hover:bg-blue-500/20 cursor-pointer transition-colors disabled:opacity-50"
        >
          <Search size={10} />{busy === 'scout' ? '...' : '一键侦查'}
        </button>
        <button
          type="button"
          onClick={(e) => handleQuickAction(e, 'plunder')}
          disabled={busy !== null}
          className="flex-1 flex items-center justify-center gap-1 px-2 py-1.5 rounded-lg text-[10px] font-medium bg-amber-500/10 text-amber-600 hover:bg-amber-500/20 cursor-pointer transition-colors disabled:opacity-50"
        >
          <ShieldAlert size={10} />{busy === 'plunder' ? '...' : '一键掠夺'}
        </button>
        <button
          type="button"
          onClick={(e) => handleQuickAction(e, 'attack')}
          disabled={busy !== null}
          className="flex-1 flex items-center justify-center gap-1 px-2 py-1.5 rounded-lg text-[10px] font-medium bg-red-500/10 text-red-600 hover:bg-red-500/20 cursor-pointer transition-colors disabled:opacity-50"
        >
          <Swords size={10} />{busy === 'attack' ? '...' : '一键攻击'}
        </button>
      </div>
    </div>
  )
}

export default NpcCityCard
