// 本文件实现天机轮转老虎机小游戏，并接入后端权威 3x3 结算与天机库存兑换。
import { useCallback, useEffect, useMemo, useRef, useState, type FC } from 'react'
import { CircleHelp, History, Loader2, PackageCheck, Play, RotateCw, Trophy, XCircle } from 'lucide-react'
import { gameApi } from '@/api/game'
import bronzeCharmIcon from '@/assets/minigames/slot/bronze-charm.svg'
import silverCharmIcon from '@/assets/minigames/slot/silver-charm.svg'
import goldCharmIcon from '@/assets/minigames/slot/gold-charm.svg'
import jadeSealIcon from '@/assets/minigames/slot/jade-seal.svg'
import tigerTallyIcon from '@/assets/minigames/slot/tiger-tally.svg'
import heavenOrderIcon from '@/assets/minigames/slot/heaven-order.svg'
import wildOrderIcon from '@/assets/minigames/slot/wild-order.svg'
import scatterStarIcon from '@/assets/minigames/slot/scatter-star.svg'
import bonusChestIcon from '@/assets/minigames/slot/bonus-chest.svg'
import { toast } from '@/components/ui'
import { useConfigStore, type SlotSymbolConfig } from '@/store/configStore'
import { useGameStore } from '@/store/gameStore'
import type { ArmyUnit, MiniGameRecord, SlotAllPayReward, SlotBonusReward, SlotFreeSpinResult, SlotRoundResult, SlotWinningLine } from '@/types/game'
import { RARITY_CONFIG } from './fishing/fishingConfig'
import type { FishCatch } from './fishing/types'
import { SlotInventoryModal } from './SlotInventoryModal'

const FALLBACK_SLOT_SYMBOLS: SlotSymbolConfig[] = [
  { id: 'bronze_charm', name: '玄铜符', rarity: 'common', type: 'normal', weight: 30, multiplier: 3 },
  { id: 'silver_charm', name: '白银符', rarity: 'rare', type: 'normal', weight: 22, multiplier: 6 },
  { id: 'gold_charm', name: '赤金符', rarity: 'rare', type: 'normal', weight: 16, multiplier: 12 },
  { id: 'jade_seal', name: '玉玺', rarity: 'epic', type: 'normal', weight: 10, multiplier: 30 },
  { id: 'tiger_tally', name: '虎符', rarity: 'epic', type: 'normal', weight: 6, multiplier: 80 },
  { id: 'heaven_order', name: '天命令', rarity: 'legendary', type: 'normal', weight: 2, multiplier: 250 },
  { id: 'wild', name: '天机令', rarity: 'epic', type: 'wild', weight: 5, multiplier: 250 },
  { id: 'scatter', name: '星陨', rarity: 'epic', type: 'scatter', weight: 4, freeSpins: 5, retriggerFreeSpins: 3 },
  { id: 'bonus', name: '宝匣', rarity: 'rare', type: 'bonus', weight: 5, bonusMultipliers: [{ multiplier: 5, weight: 50 }] },
]

const SLOT_SYMBOL_ICONS: Record<string, string> = {
  bronze_charm: bronzeCharmIcon,
  silver_charm: silverCharmIcon,
  gold_charm: goldCharmIcon,
  jade_seal: jadeSealIcon,
  tiger_tally: tigerTallyIcon,
  heaven_order: heavenOrderIcon,
  wild: wildOrderIcon,
  scatter: scatterStarIcon,
  bonus: bonusChestIcon,
}

const LINE_BET_AMOUNTS = [1000, 5000, 10000, 50000]
const REEL_LOCK_DELAYS = [2680, 4080, 5580]
const REEL_FILLER_ROWS = 14
const WIN_REACTION_DELAY_MS = 3000
const LOSE_REACTION_DELAY_MS = 1200
const DEFAULT_GRID = [
  ['bronze_charm', 'silver_charm', 'gold_charm'],
  ['jade_seal', 'wild', 'tiger_tally'],
  ['scatter', 'bonus', 'heaven_order'],
]
const RECORD_PAGE_SIZE = 100
const BULK_REDEEM_ID = '__all__'

type SlotPhase = 'betting' | 'spinning' | 'result'
type SlotDebugWindow = Window & {
  render_game_to_text?: () => string
  advanceTime?: (ms?: number) => void
}

interface PlayerUnit {
  unitType: string
  name: string
  amount: number
}

interface SlotStats {
  totalGames: number
  wins: number
  biggestWin: number
  totalWon: number
  totalBet: number
}

// normalizeRarity 将老虎机稀有度映射到现有奖励样式。
const normalizeRarity = (rarity: string): FishCatch['rarity'] => {
  return (rarity in RARITY_CONFIG ? rarity : 'common') as FishCatch['rarity']
}

// symbolById 根据图案 ID 获取配置，旧数据缺失时返回兜底符号。
const symbolById = (symbols: SlotSymbolConfig[], id: string): SlotSymbolConfig => {
  return symbols.find(symbol => symbol.id === id) ?? FALLBACK_SLOT_SYMBOLS.find(symbol => symbol.id === id) ?? FALLBACK_SLOT_SYMBOLS[0]
}

// symbolIconById 根据图案 ID 获取对应小图标。
const symbolIconById = (id: string): string => {
  return SLOT_SYMBOL_ICONS[id] ?? bronzeCharmIcon
}

// positionKey 把行列坐标转换为高亮集合键。
const positionKey = (row: number, col: number) => `${row}:${col}`

// winningPositionSet 汇总中奖线坐标。
const winningPositionSet = (lines: SlotWinningLine[]): Set<string> => {
  const result = new Set<string>()
  lines.forEach(line => line.positions.forEach(pos => result.add(positionKey(pos[0], pos[1]))))
  return result
}

// slotHighlightPositionSet 汇总普通连线、宝匣触发和满天星坐标。
const slotHighlightPositionSet = (lines: SlotWinningLine[], bonusRewards: SlotBonusReward[], allPayRewards: SlotAllPayReward[]): Set<string> => {
  const result = winningPositionSet(lines)
  bonusRewards.forEach(reward => slotArray(reward.positions).forEach(pos => result.add(positionKey(pos[0], pos[1]))))
  allPayRewards.forEach(reward => slotArray(reward.positions).forEach(pos => result.add(positionKey(pos[0], pos[1]))))
  return result
}

// slotArray 兼容旧响应里的 null 列表，避免结果弹窗读取 length 崩溃。
const slotArray = <T,>(value: T[] | null | undefined): T[] => Array.isArray(value) ? value : []

// reelStripSymbols 生成滚轴滑动长条，末尾三格固定为后端最终结果。
const reelStripSymbols = (symbols: SlotSymbolConfig[], targetGrid: string[][], col: number): string[] => {
  const pool = symbols.length > 0 ? symbols : FALLBACK_SLOT_SYMBOLS
  const filler = Array.from({ length: REEL_FILLER_ROWS }, (_, index) => pool[(index * 2 + col * 3) % pool.length]?.id ?? FALLBACK_SLOT_SYMBOLS[0].id)
  const target = [0, 1, 2].map(row => targetGrid[row]?.[col] ?? FALLBACK_SLOT_SYMBOLS[0].id)
  return [...filler, ...target]
}

// SlotMachineGame 渲染天机轮转主界面和一局结算流程。
const SlotMachineGame: FC = () => {
  const gameState = useGameStore((s) => s.state)
  const patchState = useGameStore((s) => s.patchState)
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const units = useConfigStore((s) => s.units)
  const slotConfig = useConfigStore((s) => s.slot)
  const faction = gameState?.player.faction ?? 'wei'
  const factionUnits = units?.[faction] ?? {}
  const symbols = slotConfig?.symbols?.length ? slotConfig.symbols : FALLBACK_SLOT_SYMBOLS
  const minLineBet = slotConfig?.minLineBet ?? 1000
  const lineCount = slotConfig?.lineCount ?? 5

  const playerUnits = useMemo<PlayerUnit[]>(() => {
    return (gameState?.army ?? [])
      .filter(unit => unit.amount > 0)
      .map(unit => {
        const cfg = factionUnits[unit.unitType]
        return {
          unitType: unit.unitType,
          name: cfg?.name ?? unit.unitType,
          amount: unit.amount,
        }
      })
      .filter(unit => {
        const cfg = factionUnits[unit.unitType]
        if (!cfg) return false
        if (cfg.role === 'scout' || cfg.role === 'transport') return false
        return (cfg.stats?.upkeep ?? 0) > 0 && unit.amount >= minLineBet
      })
  }, [factionUnits, gameState?.army, minLineBet])

  const [selectedUnit, setSelectedUnit] = useState<PlayerUnit | null>(playerUnits[0] ?? null)
  const [lineBet, setLineBet] = useState(LINE_BET_AMOUNTS[0])
  const [customLineBet, setCustomLineBet] = useState('')
  const [phase, setPhase] = useState<SlotPhase>('betting')
  const [grid, setGrid] = useState<string[][]>(DEFAULT_GRID)
  const [spinningTargetGrid, setSpinningTargetGrid] = useState<string[][] | null>(null)
  const [lockedReels, setLockedReels] = useState(0)
  const [activeLines, setActiveLines] = useState<SlotWinningLine[]>([])
  const [activeBonusRewards, setActiveBonusRewards] = useState<SlotBonusReward[]>([])
  const [activeAllPayRewards, setActiveAllPayRewards] = useState<SlotAllPayReward[]>([])
  const [currentSpinLabel, setCurrentSpinLabel] = useState('')
  const [pendingRound, setPendingRound] = useState<SlotRoundResult | null>(null)
  const [history, setHistory] = useState<SlotRoundResult[]>([])
  const [showHistory, setShowHistory] = useState(false)
  const [showRules, setShowRules] = useState(false)
  const [stats, setStats] = useState<SlotStats>({ totalGames: 0, wins: 0, biggestWin: 0, totalWon: 0, totalBet: 0 })
  const [records, setRecords] = useState<MiniGameRecord[]>([])
  const [recordsLoading, setRecordsLoading] = useState(false)
  const [recordsTotal, setRecordsTotal] = useState(0)
  const [recordsHasMore, setRecordsHasMore] = useState(false)
  const [recordsOffset, setRecordsOffset] = useState(0)
  const [redeemingId, setRedeemingId] = useState('')
  const [showInventory, setShowInventory] = useState(false)
  const [resolvingRound, setResolvingRound] = useState(false)
  const timeoutRefs = useRef<Array<ReturnType<typeof setTimeout>>>([])
  const lockedReelsRef = useRef(0)

  const actualLineBet = customLineBet ? parseInt(customLineBet, 10) || 0 : lineBet
  const totalBet = actualLineBet
  const canBet = Boolean(selectedUnit) && actualLineBet >= minLineBet && totalBet <= (selectedUnit?.amount ?? 0) && phase === 'betting' && !resolvingRound
  const inventoryRecords = records.filter(record => record.remainingAmount > 0)
  const highlightedPositions = slotHighlightPositionSet(activeLines, activeBonusRewards, activeAllPayRewards)

  // clearAnimationTimers 清理滚轮动画计时器。
  const clearAnimationTimers = useCallback(() => {
    timeoutRefs.current.forEach(timer => clearTimeout(timer))
    timeoutRefs.current = []
    lockedReelsRef.current = 0
  }, [])

  // loadRecords 拉取天机轮转记录和库存。
  const loadRecords = useCallback(async (offset = 0) => {
    if (!activePlayerId) return
    setRecordsLoading(true)
    try {
      const result = await gameApi.listMiniGameRecords(activePlayerId, RECORD_PAGE_SIZE, offset, 'slot')
      setRecords(Array.isArray(result.records) ? result.records : [])
      setRecordsTotal(result.totalRecords)
      setRecordsHasMore(result.hasMore)
      setRecordsOffset(result.offset)
    } finally {
      setRecordsLoading(false)
    }
  }, [activePlayerId])

  useEffect(() => {
    void loadRecords()
  }, [loadRecords])

  useEffect(() => {
    if (!selectedUnit) {
      setSelectedUnit(playerUnits[0] ?? null)
      return
    }
    const latestUnit = playerUnits.find(unit => unit.unitType === selectedUnit.unitType)
    if (!latestUnit) {
      setSelectedUnit(playerUnits[0] ?? null)
      return
    }
    if (latestUnit.amount !== selectedUnit.amount) {
      setSelectedUnit(latestUnit)
    }
  }, [playerUnits, selectedUnit])

  useEffect(() => {
    return () => clearAnimationTimers()
  }, [clearAnimationTimers])

  useEffect(() => {
    const debugWindow = window as SlotDebugWindow
    debugWindow.render_game_to_text = () => JSON.stringify({
      mode: phase,
      grid,
      lockedReels,
      lineBet: actualLineBet,
      totalBet,
      currentSpinLabel,
      highlighted: Array.from(highlightedPositions),
    })
    debugWindow.advanceTime = () => undefined
    return () => {
      delete debugWindow.render_game_to_text
      delete debugWindow.advanceTime
    }
  }, [actualLineBet, currentSpinLabel, grid, highlightedPositions, lockedReels, phase, totalBet])

  // isFactionUnit 判断库存奖励是否属于当前玩家阵营兵种。
  const isFactionUnit = (unitName: string): boolean => {
    if (!units?.[faction]) return false
    return Object.values(units[faction]).some(config => config.name === unitName)
  }

  // handleRedeemGroup 按兵种分组兑换天机库存。
  const handleRedeemGroup = async (unitName: string, groupRecords: MiniGameRecord[]) => {
    if (!activePlayerId || redeemingId) return
    const targets = groupRecords.filter(record => record.remainingAmount > 0)
    const totalAmount = targets.reduce((sum, record) => sum + record.remainingAmount, 0)
    if (targets.length === 0 || totalAmount <= 0) {
      toast.error('没有可兑换库存')
      return
    }
    setRedeemingId(unitName)
    try {
      let redeemed = 0
      let latestArmy: ArmyUnit[] | null = null
      let latestServerTime = ''
      let garrisonAmount = 0
      for (const record of targets) {
        const result = await gameApi.redeemMiniGameReward(activePlayerId, record.id, record.remainingAmount)
        redeemed += result.redeemedAmount
        if (result.redeemedTarget === 'garrison') garrisonAmount += result.redeemedAmount
        latestArmy = result.army ?? latestArmy
        latestServerTime = result.serverTime || latestServerTime
        setRecords(prev => prev.map(item => item.id === result.record.id ? result.record : item))
      }
      if (latestArmy) patchState({ army: latestArmy, serverTime: latestServerTime })
      if (garrisonAmount > 0) window.dispatchEvent(new Event('hero3:garrison-updated'))
      toast.success(`${unitName} ×${redeemed.toLocaleString()} 已兑换`)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '兑换失败')
    } finally {
      setRedeemingId('')
    }
  }

  // handleRedeemAllSlotInventory 一次性兑换当前天机库存。
  const handleRedeemAllSlotInventory = async () => {
    if (!activePlayerId || redeemingId) return
    setRedeemingId(BULK_REDEEM_ID)
    try {
      const result = await gameApi.redeemAllMiniGameRewards(activePlayerId, 'slot')
      if (result.redeemedAmount <= 0) {
        toast.error('没有可兑换库存')
        return
      }
      patchState({ army: result.army, serverTime: result.serverTime })
      if ((result.garrisonRecords ?? 0) > 0) window.dispatchEvent(new Event('hero3:garrison-updated'))
      await loadRecords(0)
      toast.success(`已兑换 ${result.redeemedAmount.toLocaleString()} 兵力`)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '兑换失败')
    } finally {
      setRedeemingId('')
    }
  }

  // playGridAnimation 播放单次 3 轮滚动，并在停靠时对齐后端返回 grid。
  const playGridAnimation = (targetGrid: string[][], lines: SlotWinningLine[], bonusRewards: SlotBonusReward[], allPayRewards: SlotAllPayReward[], label: string, onDone: () => void) => {
    clearAnimationTimers()
    setCurrentSpinLabel(label)
    setActiveLines([])
    setActiveBonusRewards([])
    setActiveAllPayRewards([])
    setLockedReels(0)
    setSpinningTargetGrid(targetGrid)
    lockedReelsRef.current = 0

    const lockColumn = (col: number) => {
      lockedReelsRef.current = col + 1
      setLockedReels(col + 1)
      setGrid(prev => prev.map((row, rowIndex) => row.map((cell, colIndex) => colIndex === col ? targetGrid[rowIndex]?.[colIndex] ?? cell : cell)))
    }
    REEL_LOCK_DELAYS.forEach((delay, col) => {
      timeoutRefs.current.push(setTimeout(() => lockColumn(col), delay))
    })
    timeoutRefs.current.push(setTimeout(() => {
      setGrid(targetGrid)
      setSpinningTargetGrid(null)
      setActiveLines(lines)
      setActiveBonusRewards(bonusRewards)
      setActiveAllPayRewards(allPayRewards)
      timeoutRefs.current.push(setTimeout(onDone, lines.length > 0 || bonusRewards.length > 0 || allPayRewards.length > 0 ? WIN_REACTION_DELAY_MS : LOSE_REACTION_DELAY_MS))
    }, REEL_LOCK_DELAYS[2] + 120))
  }

  // finishRound 收尾一局统计和历史记录。
  const finishRound = (round: SlotRoundResult) => {
    setStats(prev => ({
      totalGames: prev.totalGames + 1,
      wins: prev.wins + (round.won ? 1 : 0),
      biggestWin: Math.max(prev.biggestWin, round.winAmount),
      totalWon: prev.totalWon + round.winAmount,
      totalBet: prev.totalBet + round.totalBet,
    }))
    setHistory(prev => [round, ...prev].slice(0, 20))
    setResolvingRound(false)
    setPhase('result')
  }

  // playRoundSequence 依次播放主旋转和后端返回的免费旋转。
  const playRoundSequence = (round: SlotRoundResult) => {
    const roundFreeSpins = slotArray(round.freeSpins)
    const spins: Array<{ grid: string[][]; lines: SlotWinningLine[]; bonusRewards: SlotBonusReward[]; allPayRewards: SlotAllPayReward[]; label: string }> = [
      { grid: round.grid, lines: slotArray(round.winningLines), bonusRewards: slotArray(round.bonusRewards), allPayRewards: slotArray(round.allPayRewards), label: '主旋转' },
      ...roundFreeSpins.map((spin: SlotFreeSpinResult) => ({ grid: spin.grid, lines: slotArray(spin.winningLines), bonusRewards: slotArray(spin.bonusRewards), allPayRewards: slotArray(spin.allPayRewards), label: `免费旋转 ${spin.spinIndex}/${roundFreeSpins.length}` })),
    ]
    const playAt = (index: number) => {
      const spin = spins[index]
      if (!spin) {
        finishRound(round)
        return
      }
      playGridAnimation(spin.grid, spin.lines, spin.bonusRewards, spin.allPayRewards, spin.label, () => playAt(index + 1))
    }
    playAt(0)
  }

  // startSlotRound 请求后端结算并播放服务端结果动画。
  const startSlotRound = async () => {
    if (!activePlayerId || !selectedUnit || !canBet) return
    setResolvingRound(true)
    try {
      const round = await gameApi.resolveSlotRound(activePlayerId, selectedUnit.unitType, actualLineBet)
      if (round.army) patchState({ army: round.army, serverTime: round.serverTime })
      setRecords(prev => [round.record, ...prev].slice(0, 200))
      setRecordsTotal(prev => prev + 1)
      setPendingRound(round)
      setPhase('spinning')
      playRoundSequence(round)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '天机轮转结算失败')
      setResolvingRound(false)
    }
  }

  // resetRound 返回押注态，等待下一局。
  const resetRound = () => {
    clearAnimationTimers()
    setPhase('betting')
    setPendingRound(null)
    setLockedReels(0)
    setSpinningTargetGrid(null)
    setActiveLines([])
    setActiveBonusRewards([])
    setActiveAllPayRewards([])
    setCurrentSpinLabel('')
  }

  if (playerUnits.length === 0 && phase === 'betting') {
    return (
      <div className="max-w-md mx-auto text-center py-12">
        <RotateCw size={40} className="text-[var(--color-text-muted)] mx-auto mb-4" />
        <p className="text-sm text-[var(--color-text-primary)] font-medium">暂无可押注兵种</p>
        <p className="text-xs text-[var(--color-text-muted)] mt-1">至少需要一个战斗兵种可支付单次押注</p>
      </div>
    )
  }

  return (
    <div className="max-w-md mx-auto">
      <div className="mb-4 flex items-center justify-between gap-3">
        <div>
          <h2 className="flex items-center gap-2 text-lg font-bold text-[var(--color-text-primary)]">
            <RotateCw size={20} className="text-violet-500" />
            天机轮转
          </h2>
          <p className="mt-0.5 text-[11px] text-[var(--color-text-muted)]">固定 {lineCount} 线 · 本局押注 {totalBet > 0 ? totalBet.toLocaleString() : '-'}</p>
        </div>
        <div className="flex items-center gap-2">
          <button type="button" onClick={() => setShowRules(true)} className="p-1.5 rounded-lg bg-[var(--color-surface)] border border-[var(--color-border)] cursor-pointer hover:border-violet-500/40 transition-colors" aria-label="查看天机轮转玩法">
            <CircleHelp size={14} className="text-[var(--color-text-muted)]" />
          </button>
          <button type="button" onClick={() => setShowHistory(!showHistory)} className="p-1.5 rounded-lg bg-[var(--color-surface)] border border-[var(--color-border)] cursor-pointer hover:border-violet-500/40 transition-colors" aria-label="查看天机轮转历史">
            <History size={14} className="text-[var(--color-text-muted)]" />
          </button>
          <button type="button" onClick={() => setShowInventory(true)} className="relative p-1.5 rounded-lg bg-[var(--color-surface)] border border-[var(--color-border)] cursor-pointer hover:border-violet-500/40 transition-colors" aria-label="打开天机库存">
            <PackageCheck size={14} className="text-violet-600" />
            {inventoryRecords.length > 0 && (
              <span className="absolute -right-1 -top-1 min-w-[14px] rounded-full bg-emerald-500 px-1 text-center text-[8px] font-bold leading-[14px] text-white">
                {inventoryRecords.length > 99 ? '99+' : inventoryRecords.length}
              </span>
            )}
          </button>
        </div>
      </div>

      {stats.totalGames > 0 && (
        <div className="mb-4 grid grid-cols-4 gap-2">
          <StatTile label="总局" value={stats.totalGames.toLocaleString()} />
          <StatTile label="中奖" value={stats.wins.toLocaleString()} tone="text-emerald-600" />
          <StatTile label="最大赢" value={stats.biggestWin > 0 ? `${Math.round(stats.biggestWin / 10000)}万` : '-'} tone="text-violet-600" />
          <StatTile label="返奖" value={stats.totalBet > 0 ? `${Math.round(stats.totalWon / stats.totalBet * 100)}%` : '-'} tone="text-amber-600" />
        </div>
      )}

      {showHistory && history.length > 0 && (
        <div className="mb-4 max-h-[180px] overflow-y-auto rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] p-3">
          <h3 className="mb-2 text-xs font-semibold text-[var(--color-text-primary)]">最近记录</h3>
          <div className="space-y-1.5">
            {history.map(round => (
              <div key={round.record.id} className="flex items-center justify-between rounded-lg bg-[var(--color-surface-dim)] px-2.5 py-1.5">
                <div className="min-w-0">
                  <p className="truncate text-[10px] font-medium text-[var(--color-text-secondary)]">{round.record.resultName}</p>
                  <p className="truncate text-[9px] text-[var(--color-text-muted)]">押 {round.betUnit} ×{round.totalBet.toLocaleString()}</p>
                </div>
                <span className={`text-[10px] font-bold ${round.won ? 'text-emerald-600' : 'text-red-500'}`}>
                  {round.won ? `+${round.winAmount.toLocaleString()}` : `-${round.totalBet.toLocaleString()}`}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="mb-4 rounded-lg border border-violet-500/20 bg-[var(--color-surface)] p-4">
        <div className="mb-2 flex items-center justify-between text-[10px] text-[var(--color-text-muted)]">
          <span>{currentSpinLabel || '待机'}</span>
          <span>{lockedReels}/3</span>
        </div>
        <div className="grid grid-cols-3 gap-2">
          {[0, 1, 2].map(col => (
            <div key={col} className={`relative h-64 overflow-hidden rounded-lg border bg-[var(--color-surface-dim)] p-2 transition-colors duration-200 ${lockedReels > col ? 'border-violet-500/40 shadow-sm' : 'border-[var(--color-border)]'}`}>
              <div className="pointer-events-none absolute inset-x-2 top-2 z-10 h-5 rounded-t-lg bg-gradient-to-b from-[var(--color-surface-dim)] to-transparent" />
              <div className="pointer-events-none absolute inset-x-2 bottom-2 z-10 h-5 rounded-b-lg bg-gradient-to-t from-[var(--color-surface-dim)] to-transparent" />
              {phase === 'spinning' && lockedReels <= col ? (
                <div className={`slot-reel-strip animate-slot-reel-spin-${col}`}>
                  {reelStripSymbols(symbols, spinningTargetGrid ?? grid, col).map((symbolID, index) => (
                    <SlotSymbolTile
                      key={`${col}-spin-${index}-${symbolID}`}
                      symbol={symbolById(symbols, symbolID)}
                      highlighted={false}
                    />
                  ))}
                </div>
              ) : (
                <div className="grid h-full grid-rows-3 gap-2">
                  {[0, 1, 2].map(row => (
                    <SlotSymbolTile
                      key={`${row}-${col}-${grid[row]?.[col] ?? 'empty'}`}
                      symbol={symbolById(symbols, grid[row]?.[col] ?? FALLBACK_SLOT_SYMBOLS[0].id)}
                      highlighted={highlightedPositions.has(positionKey(row, col))}
                    />
                  ))}
                </div>
              )}
            </div>
          ))}
        </div>
      </div>

      {phase === 'betting' && (
        <div className="space-y-4">
          <div className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] p-4">
            <h3 className="mb-3 text-xs font-semibold text-[var(--color-text-primary)]">选择押注兵种</h3>
            <div className="grid grid-cols-2 gap-2">
              {playerUnits.map(unit => (
                <button key={unit.unitType} type="button" onClick={() => setSelectedUnit(unit)} className={`flex items-center gap-2 rounded-lg border-2 p-2.5 text-left transition-all duration-150 cursor-pointer ${selectedUnit?.unitType === unit.unitType ? 'border-violet-500/40 bg-violet-500/10 shadow-sm' : 'border-transparent bg-[var(--color-surface-dim)] hover:border-[var(--color-border)]'}`}>
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-[11px] font-medium text-[var(--color-text-primary)]">{unit.name}</p>
                    <p className="text-[9px] text-[var(--color-text-muted)]">拥有 {unit.amount.toLocaleString()}</p>
                  </div>
                </button>
              ))}
            </div>
          </div>

          <div className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] p-4">
            <h3 className="mb-3 text-xs font-semibold text-[var(--color-text-primary)]">单次押注</h3>
            <div className="mb-3 grid grid-cols-4 gap-2">
              {LINE_BET_AMOUNTS.map(amount => (
                <button key={amount} type="button" onClick={() => { setLineBet(amount); setCustomLineBet('') }} disabled={amount < minLineBet || amount > (selectedUnit?.amount ?? 0)} className={`rounded-lg border-2 px-2 py-2 text-xs font-medium transition-all duration-150 cursor-pointer disabled:cursor-not-allowed disabled:opacity-40 ${!customLineBet && lineBet === amount ? 'border-violet-500/40 bg-violet-500/10 text-violet-600' : 'border-transparent bg-[var(--color-surface-dim)] text-[var(--color-text-secondary)] hover:border-[var(--color-border)]'}`}>
                  {amount >= 10000 ? `${amount / 10000}万` : amount.toLocaleString()}
                </button>
              ))}
            </div>
            <input type="number" placeholder="自定义单次押注..." value={customLineBet} onChange={(event) => setCustomLineBet(event.target.value)} className="w-full rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-xs text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] focus:border-violet-500/40 focus:outline-none" />
            {actualLineBet > 0 && actualLineBet < minLineBet && <p className="mt-1.5 text-[10px] text-red-500">低于最小单次押注 {minLineBet.toLocaleString()}</p>}
            {selectedUnit && totalBet > selectedUnit.amount && <p className="mt-1.5 text-[10px] text-red-500">当前兵力不足支付本局押注 {totalBet.toLocaleString()}</p>}
          </div>

          <div className="rounded-lg border border-violet-500/30 bg-violet-500/5 p-4 text-center">
            <p className="text-sm text-[var(--color-text-primary)]">
              单次押注 <span className="font-bold text-violet-600">{actualLineBet.toLocaleString()}</span> · 可连 <span className="font-bold text-violet-600">{lineCount}</span> 线
            </p>
            <button type="button" onClick={() => void startSlotRound()} disabled={!canBet} className="mt-3 inline-flex items-center justify-center gap-2 rounded-lg bg-violet-600 px-8 py-3 text-sm font-bold text-white shadow-lg shadow-violet-500/20 transition-colors hover:bg-violet-700 disabled:cursor-not-allowed disabled:opacity-50 cursor-pointer">
              {resolvingRound ? <Loader2 size={16} className="animate-spin" /> : <Play size={16} />}
              {resolvingRound ? '结算中...' : '启动轮转'}
            </button>
          </div>
        </div>
      )}

      {phase === 'spinning' && (
        <div className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] p-5 text-center">
          <p className="text-sm font-medium text-violet-600">{currentSpinLabel || '天机轮盘转动中...'}</p>
          <p className="mt-1 text-xs text-[var(--color-text-muted)]">押注 {pendingRound?.betUnit} ×{pendingRound?.totalBet.toLocaleString()}</p>
        </div>
      )}

      {phase === 'result' && pendingRound && (
        <SlotResultModal round={pendingRound} symbols={symbols} onClose={resetRound} />
      )}

      {showRules && (
        <SlotRulesModal lineCount={lineCount} minLineBet={minLineBet} onClose={() => setShowRules(false)} />
      )}

      {showInventory && (
        <SlotInventoryModal
          records={records}
          recordsLoading={recordsLoading}
          recordsTotal={recordsTotal}
          recordsHasMore={recordsHasMore}
          recordsOffset={recordsOffset}
          recordsPageSize={RECORD_PAGE_SIZE}
          redeemingId={redeemingId}
          redeemingAll={redeemingId === BULK_REDEEM_ID}
          isFactionUnit={isFactionUnit}
          onClose={() => setShowInventory(false)}
          onRefresh={() => void loadRecords(0)}
          onPageChange={(nextOffset) => void loadRecords(nextOffset)}
          onRedeemAll={() => void handleRedeemAllSlotInventory()}
          onRedeemGroup={(unitName, groupRecords) => void handleRedeemGroup(unitName, groupRecords)}
        />
      )}
    </div>
  )
}

interface SlotSymbolTileProps {
  symbol: SlotSymbolConfig
  highlighted: boolean
}

// SlotSymbolTile 渲染滚轴中的单个图案格。
const SlotSymbolTile: FC<SlotSymbolTileProps> = ({ symbol, highlighted }) => {
  const rarity = normalizeRarity(symbol.rarity)
  const cfg = RARITY_CONFIG[rarity] ?? RARITY_CONFIG.common
  const special = symbol.type === 'scatter' || symbol.type === 'bonus'
  return (
    <div className={`flex min-h-0 flex-col items-center justify-center rounded-lg border bg-[var(--color-surface)] px-1 text-center transition-all duration-200 ${highlighted ? `${cfg.border} ${cfg.glow} animate-slot-win-flash` : special ? 'border-amber-400/40 shadow-[0_0_14px_rgba(245,158,11,0.22)]' : 'border-transparent'}`}>
      <img src={symbolIconById(symbol.id)} alt="" className="h-9 w-9 object-contain drop-shadow-[0_4px_10px_rgba(0,0,0,0.28)]" draggable={false} />
      <p className="mt-1 max-w-full truncate text-[10px] font-bold text-[var(--color-text-primary)]">{symbol.name}</p>
      <p className={`mt-0.5 text-[9px] font-semibold ${cfg.color}`}>{symbol.type === 'normal' || symbol.type === 'wild' ? `×${symbol.multiplier ?? 0}` : symbol.type === 'scatter' ? 'FREE' : 'BONUS'}</p>
    </div>
  )
}

interface StatTileProps {
  label: string
  value: string
  tone?: string
}

// StatTile 渲染天机轮转局内统计。
const StatTile: FC<StatTileProps> = ({ label, value, tone = 'text-[var(--color-text-primary)]' }) => (
  <div className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 py-1.5 text-center">
    <p className="text-[10px] text-[var(--color-text-muted)]">{label}</p>
    <p className={`text-sm font-bold ${tone}`}>{value}</p>
  </div>
)

interface SlotRulesModalProps {
  lineCount: number
  minLineBet: number
  onClose: () => void
}

// SlotRulesModal 展示天机轮转玩法说明。
const SlotRulesModal: FC<SlotRulesModalProps> = ({ lineCount, minLineBet, onClose }) => {
  const [visible, setVisible] = useState(false)

  useEffect(() => {
    requestAnimationFrame(() => setVisible(true))
  }, [])

  // handleClose 播放关闭过渡后关闭玩法弹窗。
  const handleClose = () => {
    setVisible(false)
    setTimeout(onClose, 160)
  }

  return (
    <div className="fixed inset-0 z-[9000] flex items-center justify-center p-4">
      <div className={`absolute inset-0 bg-slate-900/50 backdrop-blur-[4px] transition-opacity duration-200 ${visible ? 'opacity-100' : 'opacity-0'}`} onClick={handleClose} />
      <div className={`relative w-full max-w-sm rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] shadow-[0_24px_60px_rgba(15,23,42,0.3)] transition-all duration-200 ${visible ? 'translate-y-0 scale-100 opacity-100' : 'translate-y-4 scale-95 opacity-0'}`}>
        <div className="flex items-center justify-between border-b border-[var(--color-border)] px-4 py-3">
          <div>
            <h2 className="text-sm font-bold text-[var(--color-text-primary)]">天机轮转玩法</h2>
            <p className="mt-0.5 text-[10px] text-[var(--color-text-muted)]">固定 {lineCount} 线 · 单次最低 {minLineBet.toLocaleString()}</p>
          </div>
          <button type="button" onClick={handleClose} className="rounded-lg p-1.5 text-[var(--color-text-muted)] transition-colors hover:bg-[var(--color-surface-dim)] cursor-pointer" aria-label="关闭玩法说明">
            <XCircle size={16} />
          </button>
        </div>
        <div className="space-y-3 p-4 text-xs text-[var(--color-text-secondary)]">
          <div className="rounded-lg bg-[var(--color-surface-dim)] p-3">
            <p className="font-semibold text-[var(--color-text-primary)]">基础押注</p>
            <p className="mt-1 leading-5">选择兵种和单次押注后启动，实际只扣除本次押注；系统仍固定结算 {lineCount} 条中奖线。</p>
          </div>
          <div className="rounded-lg bg-[var(--color-surface-dim)] p-3">
            <p className="font-semibold text-[var(--color-text-primary)]">中奖连线</p>
            <p className="mt-1 leading-5">横向上、中、下三线，以及两条对角线，只要同一条线上形成 3 个普通图案或天机令替代组合，就按该图案倍率入库奖励。</p>
          </div>
          <div className="rounded-lg bg-[var(--color-surface-dim)] p-3">
            <p className="font-semibold text-[var(--color-text-primary)]">特殊图案</p>
            <p className="mt-1 leading-5">天机令是大小王，可补齐普通图案、星陨和宝匣；星陨凑满 3 个触发免费旋转，宝匣凑满 3 个触发额外奖励倍率。</p>
          </div>
          <div className="rounded-lg bg-[var(--color-surface-dim)] p-3">
            <p className="font-semibold text-[var(--color-text-primary)]">满天星</p>
            <p className="mt-1 leading-5">任意位置同类普通图案加天机令凑满 5 个及以上，会按 5/6/7/8/9 个获得额外奖励；同局只取收益最高的一组满天星。</p>
          </div>
          <div className="rounded-lg bg-[var(--color-surface-dim)] p-3">
            <p className="font-semibold text-[var(--color-text-primary)]">结果展示</p>
            <p className="mt-1 leading-5">命中的连线、宝匣和满天星会在滚轴上快速闪烁约 3 秒，随后弹出结算；未中奖会直接显示很遗憾提示。</p>
          </div>
        </div>
      </div>
    </div>
  )
}

interface SlotResultModalProps {
  round: SlotRoundResult
  symbols: SlotSymbolConfig[]
  onClose: () => void
}

// SlotResultModal 展示一局天机轮转结果详情。
const SlotResultModal: FC<SlotResultModalProps> = ({ round, symbols, onClose }) => {
  const [visible, setVisible] = useState(false)
  const winningLines = slotArray(round.winningLines)
  const bonusRewards = slotArray(round.bonusRewards)
  const allPayRewards = slotArray(round.allPayRewards)
  const freeSpins = slotArray(round.freeSpins)
  const resultTitle = round.won ? round.record.resultName : '很遗憾，未中奖'
  const resultSubtitle = round.won ? `奖励 ${round.winAmount.toLocaleString()} · 押注 ${round.totalBet.toLocaleString()}` : `本次没有形成有效连线，已扣除押注 ${round.totalBet.toLocaleString()}`

  useEffect(() => {
    requestAnimationFrame(() => setVisible(true))
  }, [])

  // handleClose 播放关闭过渡后回到押注态。
  const handleClose = () => {
    setVisible(false)
    setTimeout(onClose, 180)
  }

  return (
    <div className="fixed inset-0 z-[9000] flex items-center justify-center p-4">
      <div className={`absolute inset-0 bg-slate-900/50 backdrop-blur-[4px] transition-opacity duration-200 ${visible ? 'opacity-100' : 'opacity-0'}`} onClick={handleClose} />
      <div className={`relative max-h-[88vh] w-full max-w-sm overflow-y-auto rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] shadow-[0_24px_60px_rgba(15,23,42,0.3)] transition-all duration-200 ${visible ? 'translate-y-0 scale-100 opacity-100' : 'translate-y-4 scale-95 opacity-0'}`}>
        <div className={`px-4 py-4 text-center ${round.won ? 'bg-emerald-500/10' : 'bg-red-500/10'}`}>
          {round.won ? <Trophy size={28} className="mx-auto mb-1 text-emerald-500" /> : <XCircle size={28} className="mx-auto mb-1 text-red-500" />}
          <h2 className={`text-lg font-bold ${round.won ? 'text-emerald-600' : 'text-red-600'}`}>{resultTitle}</h2>
          <p className="mt-1 text-[10px] text-[var(--color-text-muted)]">{resultSubtitle}</p>
        </div>
        <div className="space-y-3 p-4">
          <ResultGrid grid={round.grid} symbols={symbols} lines={winningLines} bonusRewards={bonusRewards} allPayRewards={allPayRewards} />
          <div className="rounded-lg bg-[var(--color-surface-dim)] p-3">
            <ResultRow label="单次押注" value={round.lineBet.toLocaleString()} />
            <ResultRow label="可中奖线" value={`${round.lineCount}`} />
            <ResultRow label="本局扣除" value={`${round.betUnit} ×${round.totalBet.toLocaleString()}`} />
            <ResultRow label="库存奖励" value={round.winAmount.toLocaleString()} tone={round.won ? 'text-emerald-600' : 'text-red-500'} />
          </div>
          {winningLines.length > 0 && (
            <ResultList title="中奖线" items={winningLines.map(line => `${line.lineId} · ${line.symbolName} ×${line.multiplier} · ${line.amount.toLocaleString()}`)} />
          )}
          {bonusRewards.length > 0 && (
            <ResultList title="宝匣" items={bonusRewards.map(bonus => `×${bonus.multiplier} · ${bonus.amount.toLocaleString()}`)} />
          )}
          {allPayRewards.length > 0 && (
            <ResultList title="满天星" items={allPayRewards.map(reward => `${reward.symbolName} ×${reward.count} · ×${reward.multiplier} · ${reward.amount.toLocaleString()}`)} />
          )}
          {freeSpins.length > 0 && (
            <ResultList title="免费旋转" items={freeSpins.map(spin => `第 ${spin.spinIndex} 次 · +${spin.winAmount.toLocaleString()}${spin.retriggeredFreeSpins > 0 ? ` · 追加 ${spin.retriggeredFreeSpins}` : ''}`)} />
          )}
          {!round.won && (
            <div className="rounded-lg border border-red-500/20 bg-red-500/5 p-3 text-center">
              <p className="text-xs font-medium text-red-500">差一点就连成了，再试一次或调整单次押注。</p>
            </div>
          )}
          <button type="button" onClick={handleClose} className="w-full rounded-lg bg-violet-600 py-2.5 text-sm font-bold text-white transition-colors hover:bg-violet-700 cursor-pointer">
            {round.won ? '继续' : '再来一局'}
          </button>
        </div>
      </div>
    </div>
  )
}

interface ResultGridProps {
  grid: string[][]
  symbols: SlotSymbolConfig[]
  lines: SlotWinningLine[]
  bonusRewards: SlotBonusReward[]
  allPayRewards: SlotAllPayReward[]
}

// ResultGrid 渲染结果弹窗中的主旋转矩阵。
const ResultGrid: FC<ResultGridProps> = ({ grid, symbols, lines, bonusRewards, allPayRewards }) => {
  const highlights = slotHighlightPositionSet(lines, bonusRewards, allPayRewards)
  return (
    <div className="grid grid-cols-3 gap-1.5">
      {[0, 1, 2].map(row => (
        [0, 1, 2].map(col => {
          const symbol = symbolById(symbols, grid[row]?.[col] ?? FALLBACK_SLOT_SYMBOLS[0].id)
          const rarity = normalizeRarity(symbol.rarity)
          const cfg = RARITY_CONFIG[rarity] ?? RARITY_CONFIG.common
          return (
            <div key={`${row}-${col}`} className={`flex h-16 flex-col items-center justify-center rounded-lg border bg-[var(--color-surface-dim)] px-1 text-center ${highlights.has(positionKey(row, col)) ? `${cfg.border} ${cfg.glow}` : 'border-[var(--color-border)]'}`}>
              <img src={symbolIconById(symbol.id)} alt="" className="h-7 w-7 object-contain drop-shadow-[0_3px_8px_rgba(0,0,0,0.28)]" draggable={false} />
              <span className="mt-1 max-w-full truncate text-[10px] font-semibold text-[var(--color-text-primary)]">{symbol.name}</span>
            </div>
          )
        })
      ))}
    </div>
  )
}

interface ResultRowProps {
  label: string
  value: string
  tone?: string
}

// ResultRow 渲染结果数值行。
const ResultRow: FC<ResultRowProps> = ({ label, value, tone = 'text-[var(--color-text-primary)]' }) => (
  <div className="flex justify-between py-1 text-xs">
    <span className="text-[var(--color-text-muted)]">{label}</span>
    <span className={`font-semibold ${tone}`}>{value}</span>
  </div>
)

interface ResultListProps {
  title: string
  items: string[]
}

// ResultList 渲染结果明细列表。
const ResultList: FC<ResultListProps> = ({ title, items }) => (
  <div className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] p-3">
    <h3 className="mb-2 text-xs font-semibold text-[var(--color-text-primary)]">{title}</h3>
    <div className="space-y-1">
      {items.map((item, index) => (
        <p key={`${title}-${index}`} className="rounded bg-[var(--color-surface-dim)] px-2 py-1 text-[10px] text-[var(--color-text-secondary)]">{item}</p>
      ))}
    </div>
  </div>
)

export default SlotMachineGame
