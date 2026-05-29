import { useState, useEffect, useRef, type FC } from 'react'
import { Dice5, Trophy, Skull, X, TrendingDown, Flame, History } from 'lucide-react'
import { useGameStore } from '@/store/gameStore'
import { useConfigStore } from '@/store/configStore'
import { gameApi } from '@/api/game'

/* ---------- Bet Types ---------- */
interface BetOption {
  id: string
  label: string
  description: string
  odds: number // multiplier if won
  chance: number // probability of winning
  color: string
  bg: string
}

const BET_OPTIONS: BetOption[] = [
  { id: 'big', label: '大', description: '总点数 11-18', odds: 2, chance: 0.486, color: 'text-red-500', bg: 'bg-red-500/10' },
  { id: 'small', label: '小', description: '总点数 3-10', odds: 2, chance: 0.486, color: 'text-blue-500', bg: 'bg-blue-500/10' },
  { id: 'odd', label: '单', description: '总点数为奇数', odds: 2, chance: 0.50, color: 'text-green-500', bg: 'bg-green-500/10' },
  { id: 'even', label: '双', description: '总点数为偶数', odds: 2, chance: 0.50, color: 'text-purple-500', bg: 'bg-purple-500/10' },
  { id: 'triple', label: '豹子', description: '三颗骰子相同', odds: 30, chance: 0.028, color: 'text-amber-500', bg: 'bg-amber-500/10' },
  { id: 'exact', label: '猜点数', description: '猜中总点数', odds: 0, chance: 0, color: 'text-pink-500', bg: 'bg-pink-500/10' },
]

// Exact number odds: the fewer combinations, the higher the payout
const EXACT_ODDS: Record<number, number> = {
  3: 150, 4: 60, 5: 30, 6: 18, 7: 12, 8: 8, 9: 6, 10: 6,
  11: 6, 12: 6, 13: 8, 14: 12, 15: 18, 16: 30, 17: 60, 18: 150,
}

const BET_AMOUNTS = [5000, 10000, 30000, 50000, 100000]

type GamePhase = 'betting' | 'rolling' | 'result'

interface GameResult {
  won: boolean
  multiplier: number
  unitName: string
  betAmount: number
  winAmount: number
  diceTotal: number
  betLabel: string
}

interface HistoryEntry {
  unitName: string
  betAmount: number
  won: boolean
  multiplier: number
  winAmount: number
  diceTotal: number
  betLabel: string
}

interface GambleStats {
  totalGames: number
  wins: number
  losses: number
  streak: number
  bestStreak: number
  biggestWin: number
  totalWon: number
  totalLost: number
}

interface PlayerUnit {
  unitType: string
  name: string
  amount: number
}

const GamblingGame: FC = () => {
  const gameState = useGameStore((s) => s.state)
  const faction = gameState?.player.faction ?? 'wei'
  const units = useConfigStore((s) => s.units)
  const factionUnits = units?.[faction] ?? {}

  // Get player's available units (only those with amount > 0, exclude scouts/merchants)
  const playerUnits: PlayerUnit[] = (gameState?.army ?? [])
    .filter(u => u.amount > 0)
    .map(u => ({
      unitType: u.unitType,
      name: factionUnits[u.unitType]?.name ?? u.unitType,
      amount: u.amount,
    }))
    .filter(u => {
      const cfg = factionUnits[u.unitType]
      return cfg && cfg.category !== 'special' && !('role' in cfg && cfg.role === 'scout')
    })

  const [phase, setPhase] = useState<GamePhase>('betting')
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const [selectedUnit, setSelectedUnit] = useState<PlayerUnit | null>(playerUnits[0] ?? null)
  const [betAmount, setBetAmount] = useState(BET_AMOUNTS[0])
  const [customBet, setCustomBet] = useState('')
  const [selectedBet, setSelectedBet] = useState<BetOption>(BET_OPTIONS[0])
  const [exactNumber, setExactNumber] = useState(10)
  const [result, setResult] = useState<GameResult | null>(null)
  const [diceValues, setDiceValues] = useState<[number, number, number]>([1, 1, 1])
  const [revealedCount, setRevealedCount] = useState(0) // 0=rolling, 1=first locked, 2=second locked, 3=all locked
  const [, setRolling] = useState(false)
  const [history, setHistory] = useState<HistoryEntry[]>([])
  const [showHistory, setShowHistory] = useState(false)
  const [stats, setStats] = useState<GambleStats>({
    totalGames: 0, wins: 0, losses: 0, streak: 0, bestStreak: 0, biggestWin: 0, totalWon: 0, totalLost: 0,
  })
  const rollIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const finalDice = useRef<[number, number, number]>([1, 1, 1])

  const actualBet = customBet ? parseInt(customBet) || betAmount : betAmount
  const currentOdds = selectedBet.id === 'exact' ? (EXACT_ODDS[exactNumber] ?? 6) : selectedBet.odds

  const roll = () => {
    if (!selectedUnit) return
    setPhase('rolling')
    setRolling(true)
    setRevealedCount(0)

    // Pre-determine final dice values
    finalDice.current = [
      Math.ceil(Math.random() * 6),
      Math.ceil(Math.random() * 6),
      Math.ceil(Math.random() * 6),
    ]

    // Phase 1: All dice spinning together (1s)
    let count = 0
    rollIntervalRef.current = setInterval(() => {
      setDiceValues([
        Math.ceil(Math.random() * 6),
        Math.ceil(Math.random() * 6),
        Math.ceil(Math.random() * 6),
      ])
      count++
      if (count > 12) {
        if (rollIntervalRef.current) clearInterval(rollIntervalRef.current)
        // Lock first die
        setDiceValues(prev => [finalDice.current[0], prev[1], prev[2]])
        setRevealedCount(1)

        // Phase 2: Second die still spinning (0.8s)
        let count2 = 0
        rollIntervalRef.current = setInterval(() => {
          setDiceValues(prev => [finalDice.current[0], Math.ceil(Math.random() * 6), prev[2]])
          count2++
          if (count2 > 8) {
            if (rollIntervalRef.current) clearInterval(rollIntervalRef.current)
            // Lock second die
            setDiceValues([finalDice.current[0], finalDice.current[1], Math.ceil(Math.random() * 6)])
            setRevealedCount(2)

            // Phase 3: Third die still spinning (1.2s for suspense)
            let count3 = 0
            rollIntervalRef.current = setInterval(() => {
              setDiceValues([finalDice.current[0], finalDice.current[1], Math.ceil(Math.random() * 6)])
              count3++
              if (count3 > 12) {
                if (rollIntervalRef.current) clearInterval(rollIntervalRef.current)
                // Lock third die
                setDiceValues(finalDice.current)
                setRevealedCount(3)
                setRolling(false)

                // Pause for suspense before showing result
                setTimeout(() => {
                  resolveResult()
                }, 1200)
              }
            }, 100)
          }
        }, 100)
      }
    }, 80)
  }

  const resolveResult = () => {
    const [d1, d2, d3] = finalDice.current
    const total = d1 + d2 + d3
    const isTriple = d1 === d2 && d2 === d3

    // Determine win
    let won = false
    switch (selectedBet.id) {
      case 'big': won = total >= 11 && !isTriple; break
      case 'small': won = total <= 10 && !isTriple; break
      case 'odd': won = total % 2 === 1 && !isTriple; break
      case 'even': won = total % 2 === 0 && !isTriple; break
      case 'triple': won = isTriple; break
      case 'exact': won = total === exactNumber; break
    }

    const multiplier = currentOdds
    const winAmount = won ? actualBet * multiplier : 0
    const betLabel = selectedBet.id === 'exact' ? `猜${exactNumber}点` : selectedBet.label

    const gameResult: GameResult = {
      won, multiplier, unitName: selectedUnit!.name, betAmount: actualBet, winAmount, diceTotal: total, betLabel,
    }
    setResult(gameResult)

    // History
    setHistory(prev => [{ ...gameResult, unitName: selectedUnit!.name }, ...prev].slice(0, 20))

    // Stats
    setStats(s => {
      const newStreak = won ? (s.streak > 0 ? s.streak + 1 : 1) : (s.streak < 0 ? s.streak - 1 : -1)
      return {
        totalGames: s.totalGames + 1,
        wins: s.wins + (won ? 1 : 0),
        losses: s.losses + (won ? 0 : 1),
        streak: newStreak,
        bestStreak: Math.max(s.bestStreak, newStreak),
        biggestWin: Math.max(s.biggestWin, winAmount),
        totalWon: s.totalWon + winAmount,
        totalLost: s.totalLost + (won ? 0 : actualBet),
      }
    })

    // 上报赌博记录到后端
    if (activePlayerId) {
      const resultName = won ? `${betLabel} 赢 ×${multiplier}` : `${betLabel} 输`
      const rarity = won && multiplier >= 30 ? 'legendary' : won && multiplier >= 10 ? 'epic' : won ? 'rare' : 'common'
      gameApi.saveMiniGameRecord(activePlayerId, 'gambling', resultName, rarity, won ? selectedUnit!.name : '', winAmount).catch(() => {})
    }

    setPhase('result')
  }

  const reset = () => {
    setPhase('betting')
    setResult(null)
    setDiceValues([1, 1, 1])
  }

  // No units available
  if (playerUnits.length === 0) {
    return (
      <div className="max-w-md mx-auto text-center py-12">
        <Dice5 size={40} className="text-[var(--color-text-muted)] mx-auto mb-4" />
        <p className="text-sm text-[var(--color-text-primary)] font-medium">暂无可押注兵种</p>
        <p className="text-xs text-[var(--color-text-muted)] mt-1">先去征兵，有了兵力才能来赌</p>
      </div>
    )
  }

  return (
    <div className="max-w-md mx-auto">
      {/* Title & Streak */}
      <div className="flex items-center justify-between mb-4">
        <div>
          <h2 className="text-lg font-bold text-[var(--color-text-primary)] flex items-center gap-2">
            <Dice5 size={20} className="text-amber-500" />
            军营豪赌
          </h2>
          <p className="text-[11px] text-[var(--color-text-muted)] mt-0.5">押大小、猜点数，搏一搏单车变摩托</p>
        </div>
        <div className="flex items-center gap-2">
          {stats.streak > 0 && (
            <div className="flex items-center gap-1 px-2 py-1 rounded-lg bg-green-500/10 border border-green-500/20">
              <Flame size={12} className="text-green-500" />
              <span className="text-[10px] font-bold text-green-600">{stats.streak}连胜</span>
            </div>
          )}
          {stats.streak < 0 && (
            <div className="flex items-center gap-1 px-2 py-1 rounded-lg bg-red-500/10 border border-red-500/20">
              <TrendingDown size={12} className="text-red-500" />
              <span className="text-[10px] font-bold text-red-600">{Math.abs(stats.streak)}连败</span>
            </div>
          )}
          <button
            type="button"
            onClick={() => setShowHistory(!showHistory)}
            className="p-1.5 rounded-lg bg-[var(--color-surface)] border border-[var(--color-border)] cursor-pointer hover:border-[var(--color-accent)]/40 transition-colors"
          >
            <History size={14} className="text-[var(--color-text-muted)]" />
          </button>
        </div>
      </div>

      {/* Stats Row */}
      {stats.totalGames > 0 && (
        <div className="grid grid-cols-4 gap-2 mb-4">
          <div className="text-center px-2 py-1.5 rounded-lg bg-[var(--color-surface)] border border-[var(--color-border)]">
            <p className="text-[10px] text-[var(--color-text-muted)]">总局</p>
            <p className="text-sm font-bold text-[var(--color-text-primary)]">{stats.totalGames}</p>
          </div>
          <div className="text-center px-2 py-1.5 rounded-lg bg-[var(--color-surface)] border border-[var(--color-border)]">
            <p className="text-[10px] text-[var(--color-text-muted)]">胜率</p>
            <p className="text-sm font-bold text-green-600">{Math.round(stats.wins / stats.totalGames * 100)}%</p>
          </div>
          <div className="text-center px-2 py-1.5 rounded-lg bg-[var(--color-surface)] border border-[var(--color-border)]">
            <p className="text-[10px] text-[var(--color-text-muted)]">最大赢</p>
            <p className="text-sm font-bold text-amber-600">{stats.biggestWin > 0 ? `${Math.round(stats.biggestWin / 10000)}万` : '-'}</p>
          </div>
          <div className="text-center px-2 py-1.5 rounded-lg bg-[var(--color-surface)] border border-[var(--color-border)]">
            <p className="text-[10px] text-[var(--color-text-muted)]">净盈亏</p>
            <p className={`text-sm font-bold ${stats.totalWon - stats.totalLost >= 0 ? 'text-green-600' : 'text-red-600'}`}>
              {stats.totalWon - stats.totalLost >= 0 ? '+' : ''}{((stats.totalWon - stats.totalLost) / 10000).toFixed(1)}万
            </p>
          </div>
        </div>
      )}

      {/* History */}
      {showHistory && history.length > 0 && (
        <div className="mb-4 rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-3 max-h-[180px] overflow-y-auto">
          <h3 className="text-xs font-semibold text-[var(--color-text-primary)] mb-2">历史记录</h3>
          <div className="space-y-1.5">
            {history.map((h, i) => (
              <div key={i} className="flex items-center justify-between px-2.5 py-1.5 rounded-lg bg-[var(--color-surface-dim)]">
                <div className="flex items-center gap-2">
                  <span className={`w-1.5 h-1.5 rounded-full ${h.won ? 'bg-green-500' : 'bg-red-500'}`} />
                  <span className="text-[10px] text-[var(--color-text-secondary)]">{h.betLabel}</span>
                  <span className="text-[10px] text-[var(--color-text-muted)]">点数{h.diceTotal}</span>
                </div>
                <span className={`text-[10px] font-bold ${h.won ? 'text-green-600' : 'text-red-600'}`}>
                  {h.won ? `+${h.winAmount.toLocaleString()}` : `-${h.betAmount.toLocaleString()}`}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Betting Phase */}
      {phase === 'betting' && (
        <div className="space-y-4">
          {/* Select Unit from player's army */}
          <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4">
            <h3 className="text-xs font-semibold text-[var(--color-text-primary)] mb-3">选择押注兵种（你的兵力）</h3>
            <div className="grid grid-cols-2 gap-2">
              {playerUnits.map((unit) => (
                <button
                  key={unit.unitType}
                  type="button"
                  onClick={() => setSelectedUnit(unit)}
                  className={`
                    flex items-center gap-2 p-2.5 rounded-xl text-left cursor-pointer transition-all duration-150
                    ${selectedUnit?.unitType === unit.unitType
                      ? 'bg-amber-500/10 border-2 border-amber-500/40 shadow-sm'
                      : 'bg-[var(--color-surface-dim)] border-2 border-transparent hover:border-[var(--color-border)]'
                    }
                  `}
                >
                  <div className="flex-1 min-w-0">
                    <p className="text-[11px] font-medium text-[var(--color-text-primary)] truncate">{unit.name}</p>
                    <p className="text-[9px] text-[var(--color-text-muted)]">拥有 {unit.amount.toLocaleString()}</p>
                  </div>
                </button>
              ))}
            </div>
          </div>

          {/* Bet Amount */}
          <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4">
            <h3 className="text-xs font-semibold text-[var(--color-text-primary)] mb-3">押注数量</h3>
            <div className="grid grid-cols-3 gap-2 mb-3">
              {BET_AMOUNTS.map((amount) => (
                <button
                  key={amount}
                  type="button"
                  onClick={() => { setBetAmount(amount); setCustomBet('') }}
                  className={`
                    px-2 py-2 rounded-xl text-xs font-medium cursor-pointer transition-all duration-150
                    ${!customBet && betAmount === amount
                      ? 'bg-amber-500/10 border-2 border-amber-500/40 text-amber-600'
                      : 'bg-[var(--color-surface-dim)] border-2 border-transparent text-[var(--color-text-secondary)] hover:border-[var(--color-border)]'
                    }
                  `}
                >
                  {amount >= 10000 ? `${amount / 10000}万` : amount.toLocaleString()}
                </button>
              ))}
            </div>
            <input
              type="number"
              placeholder="自定义数量..."
              value={customBet}
              onChange={(e) => setCustomBet(e.target.value)}
              className="w-full px-3 py-2 rounded-xl text-xs bg-[var(--color-surface-dim)] border border-[var(--color-border)] text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] focus:outline-none focus:border-amber-500/40"
            />
            {selectedUnit && actualBet > selectedUnit.amount && (
              <p className="text-[10px] text-red-500 mt-1.5">⚠️ 超出拥有数量（{selectedUnit.amount.toLocaleString()}）</p>
            )}
          </div>

          {/* Bet Type Selection */}
          <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4">
            <h3 className="text-xs font-semibold text-[var(--color-text-primary)] mb-3">选择玩法</h3>
            <div className="grid grid-cols-3 gap-2">
              {BET_OPTIONS.map((opt) => (
                <button
                  key={opt.id}
                  type="button"
                  onClick={() => setSelectedBet(opt)}
                  className={`
                    flex flex-col items-center gap-0.5 p-2.5 rounded-xl cursor-pointer transition-all duration-150
                    ${selectedBet.id === opt.id
                      ? `${opt.bg} border-2 border-current ${opt.color}`
                      : 'bg-[var(--color-surface-dim)] border-2 border-transparent hover:border-[var(--color-border)] text-[var(--color-text-secondary)]'
                    }
                  `}
                >
                  <span className="text-sm font-bold">{opt.label}</span>
                  <span className="text-[9px] text-[var(--color-text-muted)]">{opt.description}</span>
                  {opt.id !== 'exact' && (
                    <span className={`text-[9px] font-medium mt-0.5 ${opt.color}`}>×{opt.odds}</span>
                  )}
                </button>
              ))}
            </div>

            {/* Exact number picker */}
            {selectedBet.id === 'exact' && (
              <div className="mt-3 p-3 rounded-xl bg-[var(--color-surface-dim)]">
                <p className="text-[10px] text-[var(--color-text-muted)] mb-2">选择点数（3-18），越极端赔率越高</p>
                <div className="grid grid-cols-8 gap-1.5">
                  {Array.from({ length: 16 }, (_, i) => i + 3).map(n => (
                    <button
                      key={n}
                      type="button"
                      onClick={() => setExactNumber(n)}
                      className={`
                        py-1.5 rounded-lg text-[10px] font-bold cursor-pointer transition-all
                        ${exactNumber === n
                          ? 'bg-pink-500 text-white'
                          : 'bg-[var(--color-surface)] text-[var(--color-text-secondary)] hover:bg-pink-500/10'
                        }
                      `}
                    >
                      {n}
                    </button>
                  ))}
                </div>
                <p className="text-[10px] text-pink-500 font-medium mt-2 text-center">
                  猜中 {exactNumber} 点 → ×{EXACT_ODDS[exactNumber]} 倍赔率
                </p>
              </div>
            )}
          </div>

          {/* Summary & Roll */}
          <div className="rounded-2xl border border-amber-500/30 bg-amber-500/5 p-4 text-center space-y-3">
            <div className="space-y-1">
              <p className="text-sm text-[var(--color-text-primary)]">
                押注 <span className="font-bold text-amber-600">{actualBet.toLocaleString()}</span> {selectedUnit?.name ?? ''}
              </p>
              <p className="text-xs text-[var(--color-text-muted)]">
                玩法：<span className={`font-medium ${selectedBet.color}`}>
                  {selectedBet.id === 'exact' ? `猜${exactNumber}点` : selectedBet.label}
                </span>
                {' '}| 赔率：<span className="font-bold text-amber-600">×{currentOdds}</span>
                {' '}| 赢得：<span className="font-bold text-green-600">{(actualBet * currentOdds).toLocaleString()}</span>
              </p>
            </div>
            <button
              type="button"
              onClick={roll}
              disabled={!selectedUnit || actualBet <= 0 || (selectedUnit && actualBet > selectedUnit.amount)}
              className="px-8 py-3 rounded-xl bg-amber-500 text-white text-sm font-bold hover:bg-amber-600 cursor-pointer transition-colors shadow-lg shadow-amber-500/20 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <span className="flex items-center justify-center gap-2">
                <Dice5 size={16} />
                掷骰子
              </span>
            </button>
          </div>
        </div>
      )}

      {/* Rolling Phase */}
      {phase === 'rolling' && (
        <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-8 text-center space-y-6">
          <p className="text-xs text-[var(--color-text-muted)]">
            押注：{selectedBet.id === 'exact' ? `猜${exactNumber}点` : selectedBet.label} | ×{currentOdds}
          </p>
          <div className="flex items-center justify-center gap-4">
            <DiceFace value={diceValues[0]} spinning={revealedCount < 1} delay={0} locked={revealedCount >= 1} />
            <DiceFace value={diceValues[1]} spinning={revealedCount < 2} delay={0.1} locked={revealedCount >= 2} />
            <DiceFace value={diceValues[2]} spinning={revealedCount < 3} delay={0.2} locked={revealedCount >= 3} />
          </div>
          <div className="space-y-2">
            {revealedCount < 3 ? (
              <p className="text-sm text-[var(--color-text-muted)] animate-pulse">
                {revealedCount === 0 ? '骰子滚动中...' : revealedCount === 1 ? '第二颗...' : '最后一颗！'}
              </p>
            ) : (
              <p className="text-sm text-amber-500 font-medium animate-pulse">
                总点数：{diceValues[0] + diceValues[1] + diceValues[2]}
              </p>
            )}
            <div className="flex justify-center gap-1">
              {[0, 1, 2, 3, 4].map(i => (
                <div key={i} className="w-1.5 h-1.5 rounded-full bg-amber-500 animate-bounce" style={{ animationDelay: `${i * 0.1}s` }} />
              ))}
            </div>
          </div>
        </div>
      )}

      {/* Result */}
      {phase === 'result' && result && (
        <GamblingResultModal result={result} onClose={reset} streak={stats.streak} />
      )}
    </div>
  )
}

/* ---------- Dice Face ---------- */
const DiceFace: FC<{ value: number; spinning: boolean; delay: number; locked?: boolean }> = ({ value, spinning, delay, locked }) => {
  const dots: Record<number, number[][]> = {
    1: [[1, 1]],
    2: [[0, 0], [2, 2]],
    3: [[0, 0], [1, 1], [2, 2]],
    4: [[0, 0], [0, 2], [2, 0], [2, 2]],
    5: [[0, 0], [0, 2], [1, 1], [2, 0], [2, 2]],
    6: [[0, 0], [0, 2], [1, 0], [1, 2], [2, 0], [2, 2]],
  }

  return (
    <div
      className={`
        w-16 h-16 rounded-xl border-2 shadow-md
        grid grid-cols-3 grid-rows-3 p-2 gap-0.5
        transition-all duration-300
        ${spinning ? 'animate-bounce bg-white border-[var(--color-border)]' : ''}
        ${locked ? 'bg-amber-50 border-amber-400 scale-110 shadow-amber-200/50' : 'bg-white border-[var(--color-border)]'}
      `}
      style={{ animationDelay: `${delay}s` }}
    >
      {Array.from({ length: 9 }).map((_, idx) => {
        const row = Math.floor(idx / 3)
        const col = idx % 3
        const hasDot = dots[value]?.some(([r, c]) => r === row && c === col)
        return (
          <div key={idx} className="flex items-center justify-center">
            {hasDot && <div className={`w-2.5 h-2.5 rounded-full ${locked ? 'bg-amber-700' : 'bg-slate-800'}`} />}
          </div>
        )
      })}
    </div>
  )
}

/* ---------- Result Modal ---------- */
interface GamblingResultModalProps {
  result: GameResult
  onClose: () => void
  streak: number
}

const GamblingResultModal: FC<GamblingResultModalProps> = ({ result, onClose, streak }) => {
  const [visible, setVisible] = useState(false)

  useEffect(() => {
    requestAnimationFrame(() => setVisible(true))
  }, [])

  const handleClose = () => {
    setVisible(false)
    setTimeout(onClose, 200)
  }

  return (
    <div className="fixed inset-0 z-[9000] flex items-center justify-center p-4">
      <div
        className={`absolute inset-0 bg-slate-900/50 backdrop-blur-[4px] transition-opacity duration-200 ${visible ? 'opacity-100' : 'opacity-0'}`}
        onClick={handleClose}
      />
      <div className={`
        relative w-full max-w-xs rounded-2xl overflow-hidden
        bg-[var(--color-surface)] border border-[var(--color-border)]
        shadow-[0_24px_60px_rgba(15,23,42,0.3)]
        transition-all duration-200
        ${visible ? 'opacity-100 scale-100 translate-y-0' : 'opacity-0 scale-95 translate-y-4'}
      `}>
        {/* Header */}
        <div className={`px-4 py-4 text-center relative overflow-hidden ${result.won ? 'bg-green-500/10' : 'bg-red-500/10'}`}>
          {result.won && result.multiplier >= 10 && (
            <div className="absolute inset-0 bg-gradient-to-r from-amber-500/0 via-amber-500/20 to-amber-500/0 animate-pulse" />
          )}
          {result.won ? (
            <Trophy size={28} className="mx-auto text-green-500 mb-1 relative" />
          ) : (
            <Skull size={28} className="mx-auto text-red-500 mb-1 relative" />
          )}
          <h2 className={`text-lg font-bold relative ${result.won ? 'text-green-600' : 'text-red-600'}`}>
            {result.won
              ? result.multiplier >= 30 ? '逆天大奖！！！' : result.multiplier >= 10 ? '大赢特赢！' : '赢了！'
              : '输了！'
            }
          </h2>
          <p className="text-[10px] text-[var(--color-text-muted)] mt-1 relative">
            骰子点数：{result.diceTotal} | 押注：{result.betLabel}
          </p>
          {streak > 1 && result.won && (
            <p className="text-[10px] text-green-600 font-medium mt-1 relative">🔥 {streak}连胜！</p>
          )}
          {streak < -1 && !result.won && (
            <p className="text-[10px] text-red-600 font-medium mt-1 relative">💀 {Math.abs(streak)}连败...</p>
          )}
          <button
            type="button"
            onClick={handleClose}
            className="absolute top-3 right-3 p-1 rounded-full hover:bg-white/20 cursor-pointer"
          >
            <X size={16} className="text-[var(--color-text-muted)]" />
          </button>
        </div>

        {/* Body */}
        <div className="px-4 py-4 space-y-3 text-center">
          {result.won ? (
            <div className="rounded-xl p-3 bg-green-500/10 border border-green-500/20">
              <p className="text-[11px] text-[var(--color-text-muted)] mb-1">恭喜获得</p>
              <p className="text-lg font-bold text-green-600">
                {result.unitName} ×{result.winAmount.toLocaleString()}
              </p>
              <p className="text-[10px] text-green-600/70 mt-1">
                (押注 {result.betAmount.toLocaleString()} × {result.multiplier} 倍)
              </p>
            </div>
          ) : (
            <div className="rounded-xl p-3 bg-red-500/10 border border-red-500/20">
              <p className="text-[11px] text-[var(--color-text-muted)] mb-1">损失</p>
              <p className="text-lg font-bold text-red-600">
                {result.unitName} ×{result.betAmount.toLocaleString()}
              </p>
              <p className="text-[10px] text-red-600/70 mt-1">血本无归...</p>
            </div>
          )}

          <p className="text-[10px] text-[var(--color-text-muted)]">
            * 奖励/扣除将在系统对接后生效
          </p>
        </div>

        {/* Footer */}
        <div className="px-4 py-3 border-t border-[var(--color-border)]">
          <button
            type="button"
            onClick={handleClose}
            className="w-full px-4 py-2.5 rounded-xl text-sm font-bold bg-[var(--color-accent)] text-white hover:opacity-90 cursor-pointer transition-opacity"
          >
            {result.won ? '收下奖励' : '再来一把'}
          </button>
        </div>
      </div>
    </div>
  )
}

export default GamblingGame