import { useState, useEffect, type FC } from 'react'
import { Swords, ShieldAlert, Search, X } from 'lucide-react'
import { useGameStore } from '@/store/gameStore'
import { useConfigStore } from '@/store/configStore'
import { gameApi } from '@/api/game'
import type { NpcCity, BattleReport } from '@/types/game'
import BattleResultModal from './BattleResultModal'
import ScoutResultModal from './ScoutResultModal'

interface AttackPanelProps {
  city: NpcCity
  onClose: () => void
  onComplete: () => void
}

const AttackPanel: FC<AttackPanelProps> = ({ city, onClose, onComplete }) => {
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const army = useGameStore((s) => s.state?.army ?? [])
  const faction = useGameStore((s) => s.state?.player.faction ?? 'wei')
  const setState = useGameStore((s) => s.setState)
  const units = useConfigStore((s) => s.units)

  const [selections, setSelections] = useState<Record<string, number>>({})
  const [mode, setMode] = useState<'scout' | 'attack' | 'plunder'>('scout')
  const [dispatching, setDispatching] = useState(false)
  const [battleReport, setBattleReport] = useState<BattleReport | null>(null)
  const [scoutReport, setScoutReport] = useState<BattleReport | null>(null)
  const [visible, setVisible] = useState(false)

  useEffect(() => {
    requestAnimationFrame(() => setVisible(true))
  }, [])

  const factionUnits = units?.[faction] ?? {}

  const getUnitName = (unitType: string): string => {
    return factionUnits[unitType]?.name ?? unitType
  }

  const totalSelected = Object.values(selections).reduce((sum, v) => sum + (v || 0), 0)

  const handleSelectionChange = (unitType: string, value: string) => {
    const num = Math.max(0, parseInt(value) || 0)
    const max = army.find(u => u.unitType === unitType)?.amount ?? 0
    setSelections(prev => ({ ...prev, [unitType]: Math.min(num, max) }))
  }

  const handleSelectAll = (unitType: string) => {
    const max = army.find(u => u.unitType === unitType)?.amount ?? 0
    setSelections(prev => ({ ...prev, [unitType]: max }))
  }

  const handleDispatch = async () => {
    if (!activePlayerId || totalSelected <= 0 || dispatching) return

    // 过滤掉 0 的
    const dispatchUnits: Record<string, number> = {}
    for (const [unitType, count] of Object.entries(selections)) {
      if (count > 0) dispatchUnits[unitType] = count
    }

    setDispatching(true)
    try {
      const result = await gameApi.attackNpc(activePlayerId, city.id, mode as 'attack' | 'plunder', dispatchUnits)
      setState(result.state)
      setBattleReport(result.battleReport)
    } catch {
      // 错误由全局拦截器处理
    } finally {
      setDispatching(false)
    }
  }

  const handleScout = async () => {
    if (!activePlayerId || dispatching) return
    setDispatching(true)
    try {
      const result = await gameApi.scoutNpc(activePlayerId, city.id)
      setState(result.state)
      setScoutReport(result.battleReport)
    } catch {
      // 错误由全局拦截器处理
    } finally {
      setDispatching(false)
    }
  }

  const handleBattleClose = () => {
    setBattleReport(null)
    onComplete()
  }

  return (
    <>
      <div className="fixed inset-x-0 bottom-0 z-[8000] lg:relative lg:inset-auto lg:mt-4">
        <div className={`
          bg-[var(--color-surface)] border-t lg:border border-[var(--color-border)] lg:rounded-2xl rounded-t-2xl
          shadow-[0_-8px_30px_rgba(15,23,42,0.1)] lg:shadow-md p-4
          transition-transform duration-300 ease-out
          ${visible ? 'translate-y-0' : 'translate-y-full lg:translate-y-0'}
        `}>
          {/* Header */}
          <div className="flex items-center justify-between mb-3">
            <div className="flex items-center gap-2">
              <span className="text-sm font-bold text-[var(--color-text-primary)]">出征 → {city.name}</span>
            </div>
            <button type="button" onClick={onClose} className="p-1 rounded-lg hover:bg-[var(--color-surface-dim)] cursor-pointer">
              <X size={16} className="text-[var(--color-text-muted)]" />
            </button>
          </div>

          {/* Mode Toggle: 侦查 掠夺 攻击 */}
          <div className="flex gap-2 mb-3">
            <button
              type="button"
              onClick={() => setMode('scout')}
              className={`flex-1 flex items-center justify-center gap-1.5 px-3 py-2 rounded-xl text-xs font-medium cursor-pointer transition-all ${
                mode === 'scout'
                  ? 'bg-blue-500/10 border border-blue-500/30 text-blue-600'
                  : 'bg-[var(--color-surface-dim)] border border-[var(--color-border)] text-[var(--color-text-secondary)]'
              }`}
            >
              <Search size={12} />侦查
            </button>
            <button
              type="button"
              onClick={() => setMode('plunder')}
              className={`flex-1 flex items-center justify-center gap-1.5 px-3 py-2 rounded-xl text-xs font-medium cursor-pointer transition-all ${
                mode === 'plunder'
                  ? 'bg-amber-500/10 border border-amber-500/30 text-amber-600'
                  : 'bg-[var(--color-surface-dim)] border border-[var(--color-border)] text-[var(--color-text-secondary)]'
              }`}
            >
              <ShieldAlert size={12} />掠夺
            </button>
            <button
              type="button"
              onClick={() => setMode('attack')}
              className={`flex-1 flex items-center justify-center gap-1.5 px-3 py-2 rounded-xl text-xs font-medium cursor-pointer transition-all ${
                mode === 'attack'
                  ? 'bg-red-500/10 border border-red-500/30 text-red-600'
                  : 'bg-[var(--color-surface-dim)] border border-[var(--color-border)] text-[var(--color-text-secondary)]'
              }`}
            >
              <Swords size={12} />攻击
            </button>
          </div>

          {/* Unit Selection (hidden in scout mode) */}
          {mode !== 'scout' && (
            <div className="space-y-2 max-h-[200px] overflow-y-auto mb-3">
              {army.filter(u => u.amount > 0).map((unit) => (
                <div key={unit.unitType} className="flex items-center justify-between px-3 py-2 rounded-xl bg-[var(--color-surface-dim)] border border-[var(--color-border)]">
                  <span className="text-xs font-medium text-[var(--color-text-primary)]">
                    {getUnitName(unit.unitType)}
                    <span className="text-[var(--color-text-muted)] ml-1">({unit.amount})</span>
                  </span>
                  <div className="flex items-center gap-1.5">
                    <input
                      type="number"
                      value={selections[unit.unitType] || 0}
                      onChange={(e) => handleSelectionChange(unit.unitType, e.target.value)}
                      className="w-16 text-center text-xs font-bold bg-white dark:bg-slate-800 border border-[var(--color-border)] rounded-lg py-1 text-[var(--color-text-primary)] outline-none focus:border-[var(--color-accent-border)]"
                    />
                    <button
                      type="button"
                      onClick={() => handleSelectAll(unit.unitType)}
                      className="text-[10px] font-medium text-[var(--color-accent)] hover:underline cursor-pointer"
                    >
                      全部
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}

          {/* Footer */}
          <div className="flex items-center justify-between">
            {mode === 'scout' ? (
              <span className="text-xs text-[var(--color-text-muted)]">派出全部侦察兵</span>
            ) : (
              <span className="text-xs text-[var(--color-text-muted)]">
                已选 <span className="font-bold text-[var(--color-accent)]">{totalSelected}</span> 人
              </span>
            )}
            <button
              type="button"
              onClick={mode === 'scout' ? handleScout : handleDispatch}
              disabled={mode !== 'scout' && (totalSelected <= 0 || dispatching)}
              className="px-5 py-2 rounded-xl text-xs font-bold bg-[var(--color-accent)] text-white hover:opacity-90 cursor-pointer transition-opacity disabled:opacity-50"
            >
              {dispatching ? '执行中...' : mode === 'scout' ? '发起侦查' : mode === 'attack' ? '发起攻击' : '发起掠夺'}
            </button>
          </div>
        </div>
      </div>

      {/* Battle Result */}
      {battleReport && (
        <BattleResultModal report={battleReport} onClose={handleBattleClose} />
      )}

      {/* Scout Result */}
      {scoutReport && (
        <ScoutResultModal report={scoutReport} onClose={() => setScoutReport(null)} />
      )}
    </>
  )
}

export default AttackPanel
