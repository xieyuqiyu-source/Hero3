import { useMemo, useState, type FC } from 'react'
import { FlaskConical, RotateCcw, Shield, Swords, Zap } from 'lucide-react'
import { gameApi, type BattleSimulationResponse, type CombatUnitLoss } from '@/api/game'
import { useConfigStore, type UnitConfig } from '@/store/configStore'
import { useGameStore } from '@/store/gameStore'

type BattleMode = 'attack' | 'plunder'

interface UnitEntry {
  id: string
  config: UnitConfig
}

const categoryLabels: Record<string, string> = {
  infantry: '步',
  cavalry: '骑',
  siege: '器',
  special: '特',
}

const modeOptions: Array<{ key: BattleMode; label: string }> = [
  { key: 'attack', label: '攻击' },
  { key: 'plunder', label: '掠夺' },
]

function sanitizeUnits(units: Record<string, number>): Record<string, number> {
  const clean: Record<string, number> = {}
  for (const [key, value] of Object.entries(units)) {
    const amount = Math.max(0, Math.floor(Number(value) || 0))
    if (amount > 0) clean[key] = amount
  }
  return clean
}

function isCombatUnit(config: UnitConfig): boolean {
  return config.role !== 'transport' && (config.stats.upkeep ?? 0) > 0
}

function getCombatUnitEntries(units: Record<string, UnitConfig> | undefined): UnitEntry[] {
  return Object.entries(units ?? {})
    .filter(([, config]) => isCombatUnit(config))
    .map(([id, config]) => ({ id, config }))
}

function pickUnit(entries: UnitEntry[], category: string): UnitEntry | undefined {
  return entries.find((entry) => entry.config.category === category) ?? entries[0]
}

function totalCount(units: Record<string, number>): number {
  return Object.values(units).reduce((sum, value) => sum + (Number(value) || 0), 0)
}

function totalLosses(losses: CombatUnitLoss[]): number {
  return losses.reduce((sum, loss) => sum + loss.losses, 0)
}

function formatPower(value: number): string {
  return Math.round(value).toLocaleString()
}

function formatRate(value: number): string {
  return `${Math.round(value * 1000) / 10}%`
}

const BattleSimulator: FC = () => {
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const playerFaction = useGameStore((s) => s.state?.player.faction ?? 'wei')
  const factions = useConfigStore((s) => s.factions)
  const units = useConfigStore((s) => s.units)

  const factionKeys = useMemo(() => Object.keys(factions ?? {}), [factions])
  const [mode, setMode] = useState<BattleMode>('attack')
  const [attackerFaction, setAttackerFaction] = useState(playerFaction)
  const [defenderFaction, setDefenderFaction] = useState(playerFaction)
  const [attackerUnits, setAttackerUnits] = useState<Record<string, number>>({})
  const [defenderUnits, setDefenderUnits] = useState<Record<string, number>>({})
  const [applyAttackerBonuses, setApplyAttackerBonuses] = useState(true)
  const [applyDefenderBonuses, setApplyDefenderBonuses] = useState(false)
  const [result, setResult] = useState<BattleSimulationResponse | null>(null)
  const [simulating, setSimulating] = useState(false)

  const attackerEntries = useMemo(() => getCombatUnitEntries(units?.[attackerFaction]), [units, attackerFaction])
  const defenderEntries = useMemo(() => getCombatUnitEntries(units?.[defenderFaction]), [units, defenderFaction])

  const setUnitAmount = (
    side: 'attacker' | 'defender',
    unitId: string,
    value: string,
  ) => {
    const amount = Math.max(0, Math.floor(Number(value) || 0))
    const setter = side === 'attacker' ? setAttackerUnits : setDefenderUnits
    setter((prev) => {
      const next = { ...prev }
      if (amount <= 0) delete next[unitId]
      else next[unitId] = amount
      return next
    })
  }

  const applyPreset = (kind: 'infantry' | 'cavalry' | 'mixed') => {
    const attackerInfantry = pickUnit(attackerEntries, 'infantry')
    const defenderInfantry = pickUnit(defenderEntries, 'infantry')
    const attackerCavalry = pickUnit(attackerEntries, 'cavalry')
    const defenderCavalry = pickUnit(defenderEntries, 'cavalry')

    if (kind === 'infantry' && attackerInfantry && defenderInfantry) {
      setAttackerUnits({ [attackerInfantry.id]: 100 })
      setDefenderUnits({ [defenderInfantry.id]: 100 })
    }
    if (kind === 'cavalry' && attackerCavalry && defenderCavalry) {
      setAttackerUnits({ [attackerCavalry.id]: 100 })
      setDefenderUnits({ [defenderCavalry.id]: 100 })
    }
    if (kind === 'mixed' && attackerInfantry && defenderInfantry) {
      const nextAttacker: Record<string, number> = { [attackerInfantry.id]: 120 }
      const nextDefender: Record<string, number> = { [defenderInfantry.id]: 160 }
      if (attackerCavalry) nextAttacker[attackerCavalry.id] = 80
      if (defenderCavalry) nextDefender[defenderCavalry.id] = 60
      setAttackerUnits(nextAttacker)
      setDefenderUnits(nextDefender)
    }
    setResult(null)
  }

  const runSimulation = async () => {
    if (!activePlayerId || simulating) return
    const cleanAttacker = sanitizeUnits(attackerUnits)
    const cleanDefender = sanitizeUnits(defenderUnits)
    if (Object.keys(cleanAttacker).length === 0 || Object.keys(cleanDefender).length === 0) return

    setSimulating(true)
    try {
      const response = await gameApi.simulateBattle({
        playerId: activePlayerId,
        mode,
        attackerFaction,
        defenderFaction,
        attackerUnits: cleanAttacker,
        defenderUnits: cleanDefender,
        applyAttackerBonuses,
        applyDefenderBonuses,
      })
      setResult(response)
    } catch {
      // 错误由全局提示处理
    } finally {
      setSimulating(false)
    }
  }

  const reset = () => {
    setAttackerUnits({})
    setDefenderUnits({})
    setResult(null)
  }

  const unitName = (faction: string, unitId: string): string => {
    return units?.[faction]?.[unitId]?.name ?? unitId
  }

  const renderUnitEditor = (
    side: 'attacker' | 'defender',
    faction: string,
    entries: UnitEntry[],
    values: Record<string, number>,
  ) => (
    <div className="grid grid-cols-2 gap-1.5 max-h-40 overflow-y-auto pr-1">
      {entries.map(({ id, config }) => (
        <div key={id} className="grid grid-cols-[1fr_58px] items-center gap-1.5 px-2 py-1.5 rounded-lg bg-[var(--color-surface-dim)] border border-[var(--color-border)]">
          <div className="min-w-0">
            <div className="flex items-center gap-1.5">
              <span className="text-[11px] font-semibold text-[var(--color-text-primary)] truncate">{config.name}</span>
              <span className="shrink-0 text-[9px] px-1 py-0.5 rounded bg-[var(--color-surface)] text-[var(--color-text-muted)] border border-[var(--color-border)]">
                {categoryLabels[config.category] ?? config.category}
              </span>
            </div>
          </div>
          <input
            type="number"
            min={0}
            value={values[id] ?? ''}
            placeholder="0"
            onChange={(e) => setUnitAmount(side, id, e.target.value)}
            className="w-full text-right text-[11px] font-bold bg-white dark:bg-slate-800 border border-[var(--color-border)] rounded-md px-1.5 py-1 text-[var(--color-text-primary)] outline-none focus:border-[var(--color-accent-border)]"
            aria-label={`${faction}-${id}`}
          />
        </div>
      ))}
    </div>
  )

  const winnerText = result?.result.winner === 'attacker'
    ? '攻方胜'
    : result?.result.winner === 'defender'
      ? '守方胜'
      : '平局'

  return (
    <section className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] overflow-hidden">
      <div className="flex flex-wrap items-center justify-between gap-2 px-4 py-2.5 border-b border-[var(--color-border)]">
        <div className="flex items-center gap-2">
          <FlaskConical size={16} className="text-[var(--color-accent)]" />
          <h2 className="text-sm font-semibold text-[var(--color-text-primary)]">战斗模拟</h2>
        </div>
        <div className="flex items-center gap-1.5">
          {modeOptions.map((opt) => (
            <button
              key={opt.key}
              type="button"
              onClick={() => setMode(opt.key)}
              className={`px-2.5 py-1 rounded-lg text-[11px] font-semibold border cursor-pointer transition-all ${
                mode === opt.key
                  ? 'bg-[var(--color-accent-light)] border-[var(--color-accent-border)] text-[var(--color-accent)]'
                  : 'bg-[var(--color-surface-dim)] border-[var(--color-border)] text-[var(--color-text-secondary)]'
              }`}
            >
              {opt.label}
            </button>
          ))}
          <button
            type="button"
            onClick={runSimulation}
            disabled={simulating || totalCount(attackerUnits) <= 0 || totalCount(defenderUnits) <= 0}
            className="px-3 py-1 rounded-lg text-[11px] font-bold bg-[var(--color-accent)] text-white hover:opacity-90 cursor-pointer transition-opacity disabled:opacity-50"
          >
            {simulating ? '计算中' : '模拟'}
          </button>
        </div>
      </div>

      <div className="p-3 space-y-3">
        <div className="grid grid-cols-3 gap-1.5">
          <button type="button" onClick={() => applyPreset('infantry')} className="flex items-center justify-center gap-1 px-2 py-1.5 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] text-[11px] font-semibold text-[var(--color-text-primary)] cursor-pointer hover:border-[var(--color-accent-border)]">
            <Swords size={12} />步兵
          </button>
          <button type="button" onClick={() => applyPreset('cavalry')} className="flex items-center justify-center gap-1 px-2 py-1.5 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] text-[11px] font-semibold text-[var(--color-text-primary)] cursor-pointer hover:border-[var(--color-accent-border)]">
            <Zap size={12} />骑兵
          </button>
          <button type="button" onClick={() => applyPreset('mixed')} className="flex items-center justify-center gap-1 px-2 py-1.5 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] text-[11px] font-semibold text-[var(--color-text-primary)] cursor-pointer hover:border-[var(--color-accent-border)]">
            <Shield size={12} />混编
          </button>
        </div>

        <div className="grid lg:grid-cols-2 gap-3">
          <div className="space-y-2">
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-2">
                <Swords size={14} className="text-red-500" />
                <span className="text-xs font-bold text-[var(--color-text-primary)]">攻方</span>
                <span className="text-xs text-[var(--color-text-muted)]">{totalCount(attackerUnits).toLocaleString()}</span>
              </div>
              <select
                value={attackerFaction}
                onChange={(e) => { setAttackerFaction(e.target.value); setAttackerUnits({}); setResult(null) }}
                className="text-[11px] bg-[var(--color-surface-dim)] border border-[var(--color-border)] rounded-lg px-2 py-1 text-[var(--color-text-primary)] outline-none"
              >
                {factionKeys.map((key) => <option key={key} value={key}>{factions?.[key]?.name ?? key}</option>)}
              </select>
            </div>
            <label className="flex items-center justify-between gap-2 px-2 py-1.5 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] cursor-pointer">
              <span className="text-[11px] font-medium text-[var(--color-text-primary)]">当前加成</span>
              <input type="checkbox" checked={applyAttackerBonuses} onChange={(e) => setApplyAttackerBonuses(e.target.checked)} className="w-3.5 h-3.5 accent-[var(--color-accent)]" />
            </label>
            {renderUnitEditor('attacker', attackerFaction, attackerEntries, attackerUnits)}
          </div>

          <div className="space-y-2">
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-2">
                <Shield size={14} className="text-blue-500" />
                <span className="text-xs font-bold text-[var(--color-text-primary)]">守方</span>
                <span className="text-xs text-[var(--color-text-muted)]">{totalCount(defenderUnits).toLocaleString()}</span>
              </div>
              <select
                value={defenderFaction}
                onChange={(e) => { setDefenderFaction(e.target.value); setDefenderUnits({}); setResult(null) }}
                className="text-[11px] bg-[var(--color-surface-dim)] border border-[var(--color-border)] rounded-lg px-2 py-1 text-[var(--color-text-primary)] outline-none"
              >
                {factionKeys.map((key) => <option key={key} value={key}>{factions?.[key]?.name ?? key}</option>)}
              </select>
            </div>
            <label className="flex items-center justify-between gap-2 px-2 py-1.5 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] cursor-pointer">
              <span className="text-[11px] font-medium text-[var(--color-text-primary)]">当前加成</span>
              <input type="checkbox" checked={applyDefenderBonuses} onChange={(e) => setApplyDefenderBonuses(e.target.checked)} className="w-3.5 h-3.5 accent-[var(--color-accent)]" />
            </label>
            {renderUnitEditor('defender', defenderFaction, defenderEntries, defenderUnits)}
          </div>
        </div>

        <div className="flex items-center justify-end gap-2">
          <button type="button" onClick={reset} className="flex items-center gap-1 px-2 py-1 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] text-[11px] font-semibold text-[var(--color-text-secondary)] cursor-pointer">
            <RotateCcw size={12} />清空
          </button>
        </div>

        {result && (
          <div className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] overflow-hidden">
            <div className="grid grid-cols-3 divide-x divide-[var(--color-border)]">
              <div className="px-2 py-2 text-center">
                <div className="text-[10px] text-[var(--color-text-muted)]">结果</div>
                <div className={`text-xs font-bold ${result.result.winner === 'attacker' ? 'text-red-500' : result.result.winner === 'defender' ? 'text-blue-500' : 'text-[var(--color-text-primary)]'}`}>{winnerText}</div>
              </div>
              <div className="px-2 py-2 text-center">
                <div className="text-[10px] text-[var(--color-text-muted)]">攻 / 防</div>
                <div className="text-xs font-bold text-[var(--color-text-primary)]">{formatPower(result.result.attackPower)} / {formatPower(result.result.defensePower)}</div>
              </div>
              <div className="px-2 py-2 text-center">
                <div className="text-[10px] text-[var(--color-text-muted)]">运载</div>
                <div className="text-xs font-bold text-[var(--color-text-primary)]">{result.result.survivingCarry.toLocaleString()}</div>
              </div>
            </div>
            <div className="grid lg:grid-cols-2 gap-2 p-2 border-t border-[var(--color-border)]">
              <LossPanel title="攻方损失" faction={attackerFaction} losses={result.result.attackerLosses} unitName={unitName} rate={result.result.attackerLossRate} />
              <LossPanel title="守方损失" faction={defenderFaction} losses={result.result.defenderLosses} unitName={unitName} rate={result.result.defenderLossRate} />
            </div>
          </div>
        )}
      </div>
    </section>
  )
}

interface LossPanelProps {
  title: string
  faction: string
  losses: CombatUnitLoss[]
  rate: number
  unitName: (faction: string, unitId: string) => string
}

const LossPanel: FC<LossPanelProps> = ({ title, faction, losses, rate, unitName }) => {
  const visibleLosses = losses.filter((loss) => loss.count > 0)
  return (
    <div className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2">
      <div className="flex items-center justify-between mb-2">
        <span className="text-xs font-bold text-[var(--color-text-primary)]">{title}</span>
        <span className="text-[10px] text-[var(--color-text-muted)]">{formatRate(rate)} / {totalLosses(losses).toLocaleString()}</span>
      </div>
      <div className="flex flex-wrap gap-1.5">
        {visibleLosses.map((loss) => (
          <span key={loss.id} className={`text-[10px] px-2 py-1 rounded-lg font-medium ${loss.losses > 0 ? 'bg-red-500/10 text-red-600' : 'bg-[var(--color-surface-dim)] text-[var(--color-text-muted)]'}`}>
            {unitName(faction, loss.id)} {loss.losses.toLocaleString()} / {loss.count.toLocaleString()}
          </span>
        ))}
      </div>
    </div>
  )
}

export default BattleSimulator
