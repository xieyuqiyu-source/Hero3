// 本文件实现天机轮转老虎机小游戏，并接入后端权威结算与天机库存兑换。
import { useCallback, useEffect, useMemo, useRef, useState, type FC } from 'react'
import { History, Loader2, PackageCheck, Play, RotateCw, Sparkles, Trophy, XCircle } from 'lucide-react'
import { gameApi } from '@/api/game'
import { toast } from '@/components/ui'
import { useConfigStore, type SlotSymbolConfig } from '@/store/configStore'
import { useGameStore } from '@/store/gameStore'
import type { ArmyUnit, MiniGameRecord, SlotRoundResult } from '@/types/game'
import { RARITY_CONFIG } from './fishing/fishingConfig'
import type { FishCatch } from './fishing/types'
import { SlotInventoryModal } from './SlotInventoryModal'

const FALLBACK_SLOT_SYMBOLS: SlotSymbolConfig[] = [
  { id: 'bronze_charm', name: '玄铜符', rarity: 'common', weight: 35, multiplier: 5 },
  { id: 'silver_charm', name: '白银符', rarity: 'rare', weight: 25, multiplier: 12 },
  { id: 'gold_charm', name: '赤金符', rarity: 'rare', weight: 18, multiplier: 30 },
  { id: 'jade_seal', name: '玉玺', rarity: 'epic', weight: 12, multiplier: 80 },
  { id: 'tiger_tally', name: '虎符', rarity: 'epic', weight: 7, multiplier: 250 },
  { id: 'heaven_order', name: '天命令', rarity: 'legendary', weight: 3, multiplier: 1000 },
]

const BET_AMOUNTS = [1000, 5000, 10000, 50000, 100000]
const RECORD_PAGE_SIZE = 100
const BULK_REDEEM_ID = '__all__'

type SlotPhase = 'betting' | 'spinning' | 'result'

interface PlayerUnit {
  unitType: string
  name: string
  amount: number
  maxBet: number
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

// slotMaxBet 计算前端展示用押注上限，后端仍是最终权威。
const slotMaxBet = (amount: number, maxBet: number, ratio: number): number => {
  return Math.min(maxBet, Math.floor(amount * ratio))
}

// symbolById 根据图案 ID 获取配置，旧数据缺失时返回兜底符号。
const symbolById = (symbols: SlotSymbolConfig[], id: string): SlotSymbolConfig => {
  return symbols.find(symbol => symbol.id === id) ?? FALLBACK_SLOT_SYMBOLS.find(symbol => symbol.id === id) ?? FALLBACK_SLOT_SYMBOLS[0]
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
  const minBet = slotConfig?.minBet ?? 1000
  const maxBetLimit = slotConfig?.maxBet ?? 1000000
  const maxBetRatio = slotConfig?.maxBetRatio ?? 0.05

  const playerUnits = useMemo<PlayerUnit[]>(() => {
    return (gameState?.army ?? [])
      .filter(unit => unit.amount > 0)
      .map(unit => {
        const cfg = factionUnits[unit.unitType]
        return {
          unitType: unit.unitType,
          name: cfg?.name ?? unit.unitType,
          amount: unit.amount,
          maxBet: slotMaxBet(unit.amount, maxBetLimit, maxBetRatio),
        }
      })
      .filter(unit => {
        const cfg = factionUnits[unit.unitType]
        if (!cfg) return false
        if (cfg.role === 'scout' || cfg.role === 'transport') return false
        return (cfg.stats?.upkeep ?? 0) > 0 && unit.maxBet >= minBet
      })
  }, [factionUnits, gameState?.army, maxBetLimit, maxBetRatio, minBet])

  const [selectedUnit, setSelectedUnit] = useState<PlayerUnit | null>(playerUnits[0] ?? null)
  const [betAmount, setBetAmount] = useState(BET_AMOUNTS[0])
  const [customBet, setCustomBet] = useState('')
  const [phase, setPhase] = useState<SlotPhase>('betting')
  const [reels, setReels] = useState<string[]>(['bronze_charm', 'silver_charm', 'gold_charm'])
  const [lockedReels, setLockedReels] = useState(0)
  const [pendingRound, setPendingRound] = useState<SlotRoundResult | null>(null)
  const [history, setHistory] = useState<SlotRoundResult[]>([])
  const [showHistory, setShowHistory] = useState(false)
  const [stats, setStats] = useState<SlotStats>({ totalGames: 0, wins: 0, biggestWin: 0, totalWon: 0, totalBet: 0 })
  const [records, setRecords] = useState<MiniGameRecord[]>([])
  const [recordsLoading, setRecordsLoading] = useState(false)
  const [recordsTotal, setRecordsTotal] = useState(0)
  const [recordsHasMore, setRecordsHasMore] = useState(false)
  const [recordsOffset, setRecordsOffset] = useState(0)
  const [redeemingId, setRedeemingId] = useState('')
  const [showInventory, setShowInventory] = useState(false)
  const [resolvingRound, setResolvingRound] = useState(false)
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const actualBet = customBet ? parseInt(customBet) || 0 : betAmount
  const selectedMaxBet = selectedUnit?.maxBet ?? 0
  const canBet = Boolean(selectedUnit) && actualBet >= minBet && actualBet <= selectedMaxBet && phase === 'betting' && !resolvingRound
  const inventoryRecords = records.filter(record => record.remainingAmount > 0)

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
    if (latestUnit.amount !== selectedUnit.amount || latestUnit.maxBet !== selectedUnit.maxBet) {
      setSelectedUnit(latestUnit)
    }
  }, [playerUnits, selectedUnit])

  useEffect(() => {
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current)
      if (timeoutRef.current) clearTimeout(timeoutRef.current)
    }
  }, [])

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

  // startSlotRound 请求后端结算并播放最终落点动画。
  const startSlotRound = async () => {
    if (!activePlayerId || !selectedUnit || !canBet) return
    setResolvingRound(true)
    try {
      const round = await gameApi.resolveSlotRound(activePlayerId, selectedUnit.unitType, actualBet)
      if (round.army) patchState({ army: round.army, serverTime: round.serverTime })
      setRecords(prev => [round.record, ...prev].slice(0, 200))
      setRecordsTotal(prev => prev + 1)
      setPendingRound(round)
      setPhase('spinning')
      setLockedReels(0)
      playReelAnimation(round)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '天机轮转结算失败')
      setResolvingRound(false)
    }
  }

  // playReelAnimation 播放转轮动画，并按后端返回图案逐轴停下。
  const playReelAnimation = (round: SlotRoundResult) => {
    let ticks = 0
    intervalRef.current = setInterval(() => {
      ticks += 1
      setReels(prev => prev.map((current, index) => {
        const lockThreshold = 12 + index * 8
        if (ticks >= lockThreshold) return round.symbols[index] ?? current
        return symbols[(ticks + index) % symbols.length]?.id ?? current
      }))
      if (ticks === 12) setLockedReels(1)
      if (ticks === 20) setLockedReels(2)
      if (ticks >= 28) {
        if (intervalRef.current) clearInterval(intervalRef.current)
        setLockedReels(3)
        setReels(round.symbols)
        timeoutRef.current = setTimeout(() => finishRound(round), 700)
      }
    }, 80)
  }

  // finishRound 收尾一局统计和历史记录。
  const finishRound = (round: SlotRoundResult) => {
    setStats(prev => ({
      totalGames: prev.totalGames + 1,
      wins: prev.wins + (round.won ? 1 : 0),
      biggestWin: Math.max(prev.biggestWin, round.winAmount),
      totalWon: prev.totalWon + round.winAmount,
      totalBet: prev.totalBet + round.betAmount,
    }))
    setHistory(prev => [round, ...prev].slice(0, 20))
    setResolvingRound(false)
    setPhase('result')
  }

  // resetRound 返回押注态，等待下一局。
  const resetRound = () => {
    setPhase('betting')
    setPendingRound(null)
    setLockedReels(0)
  }

  if (playerUnits.length === 0 && phase === 'betting') {
    return (
      <div className="max-w-md mx-auto text-center py-12">
        <RotateCw size={40} className="text-[var(--color-text-muted)] mx-auto mb-4" />
        <p className="text-sm text-[var(--color-text-primary)] font-medium">暂无可押注兵种</p>
        <p className="text-xs text-[var(--color-text-muted)] mt-1">至少需要某个战斗兵种达到 20,000 才能参与天机轮转</p>
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
          <p className="mt-0.5 text-[11px] text-[var(--color-text-muted)]">三符同现即可获得押注兵种库存</p>
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => setShowHistory(!showHistory)}
            className="p-1.5 rounded-lg bg-[var(--color-surface)] border border-[var(--color-border)] cursor-pointer hover:border-violet-500/40 transition-colors"
            aria-label="查看天机轮转历史"
          >
            <History size={14} className="text-[var(--color-text-muted)]" />
          </button>
          <button
            type="button"
            onClick={() => setShowInventory(true)}
            className="relative p-1.5 rounded-lg bg-[var(--color-surface)] border border-[var(--color-border)] cursor-pointer hover:border-violet-500/40 transition-colors"
            aria-label="打开天机库存"
          >
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
          <div className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 py-1.5 text-center">
            <p className="text-[10px] text-[var(--color-text-muted)]">总局</p>
            <p className="text-sm font-bold text-[var(--color-text-primary)]">{stats.totalGames}</p>
          </div>
          <div className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 py-1.5 text-center">
            <p className="text-[10px] text-[var(--color-text-muted)]">中奖</p>
            <p className="text-sm font-bold text-emerald-600">{stats.wins}</p>
          </div>
          <div className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 py-1.5 text-center">
            <p className="text-[10px] text-[var(--color-text-muted)]">最大赢</p>
            <p className="text-sm font-bold text-violet-600">{stats.biggestWin > 0 ? `${Math.round(stats.biggestWin / 10000)}万` : '-'}</p>
          </div>
          <div className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 py-1.5 text-center">
            <p className="text-[10px] text-[var(--color-text-muted)]">返奖</p>
            <p className="text-sm font-bold text-amber-600">{stats.totalBet > 0 ? `${Math.round(stats.totalWon / stats.totalBet * 100)}%` : '-'}</p>
          </div>
        </div>
      )}

      {showHistory && history.length > 0 && (
        <div className="mb-4 max-h-[180px] overflow-y-auto rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-3">
          <h3 className="mb-2 text-xs font-semibold text-[var(--color-text-primary)]">最近记录</h3>
          <div className="space-y-1.5">
            {history.map(round => (
              <div key={round.record.id} className="flex items-center justify-between rounded-lg bg-[var(--color-surface-dim)] px-2.5 py-1.5">
                <div className="min-w-0">
                  <p className="truncate text-[10px] font-medium text-[var(--color-text-secondary)]">{round.record.resultName}</p>
                  <p className="truncate text-[9px] text-[var(--color-text-muted)]">押 {round.betUnit} ×{round.betAmount.toLocaleString()}</p>
                </div>
                <span className={`text-[10px] font-bold ${round.won ? 'text-emerald-600' : 'text-red-500'}`}>
                  {round.won ? `+${round.winAmount.toLocaleString()}` : `-${round.betAmount.toLocaleString()}`}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="mb-4 rounded-2xl border border-violet-500/20 bg-[var(--color-surface)] p-4">
        <div className="flex items-center justify-center gap-3">
          {reels.map((symbolID, index) => {
            const symbol = symbolById(symbols, symbolID)
            const rarity = normalizeRarity(symbol.rarity)
            const cfg = RARITY_CONFIG[rarity] ?? RARITY_CONFIG.common
            const locked = lockedReels > index
            return (
              <div
                key={`${index}-${symbolID}`}
                className={`flex h-24 w-24 flex-col items-center justify-center rounded-2xl border-2 bg-[var(--color-surface-dim)] transition-all duration-200 ${locked ? `${cfg.border} ${cfg.glow} scale-105` : 'border-[var(--color-border)]'} ${phase === 'spinning' && !locked ? 'animate-pulse' : ''}`}
              >
                <Sparkles size={22} className={cfg.color} />
                <p className="mt-2 text-xs font-bold text-[var(--color-text-primary)]">{symbol.name}</p>
                <p className={`mt-0.5 text-[10px] font-semibold ${cfg.color}`}>×{symbol.multiplier}</p>
              </div>
            )
          })}
        </div>
      </div>

      {phase === 'betting' && (
        <div className="space-y-4">
          <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4">
            <h3 className="mb-3 text-xs font-semibold text-[var(--color-text-primary)]">选择押注兵种</h3>
            <div className="grid grid-cols-2 gap-2">
              {playerUnits.map(unit => (
                <button
                  key={unit.unitType}
                  type="button"
                  onClick={() => setSelectedUnit(unit)}
                  className={`flex items-center gap-2 rounded-xl border-2 p-2.5 text-left transition-all duration-150 cursor-pointer ${selectedUnit?.unitType === unit.unitType ? 'border-violet-500/40 bg-violet-500/10 shadow-sm' : 'border-transparent bg-[var(--color-surface-dim)] hover:border-[var(--color-border)]'}`}
                >
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-[11px] font-medium text-[var(--color-text-primary)]">{unit.name}</p>
                    <p className="text-[9px] text-[var(--color-text-muted)]">拥有 {unit.amount.toLocaleString()}</p>
                    <p className="text-[9px] text-violet-600">本局上限 {unit.maxBet.toLocaleString()}</p>
                  </div>
                </button>
              ))}
            </div>
          </div>

          <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4">
            <h3 className="mb-3 text-xs font-semibold text-[var(--color-text-primary)]">押注数量</h3>
            <div className="mb-3 grid grid-cols-3 gap-2">
              {BET_AMOUNTS.map(amount => (
                <button
                  key={amount}
                  type="button"
                  onClick={() => { setBetAmount(amount); setCustomBet('') }}
                  disabled={amount < minBet || amount > selectedMaxBet}
                  className={`rounded-xl border-2 px-2 py-2 text-xs font-medium transition-all duration-150 cursor-pointer disabled:cursor-not-allowed disabled:opacity-40 ${!customBet && betAmount === amount ? 'border-violet-500/40 bg-violet-500/10 text-violet-600' : 'border-transparent bg-[var(--color-surface-dim)] text-[var(--color-text-secondary)] hover:border-[var(--color-border)]'}`}
                >
                  {amount >= 10000 ? `${amount / 10000}万` : amount.toLocaleString()}
                </button>
              ))}
            </div>
            <input
              type="number"
              placeholder="自定义数量..."
              value={customBet}
              onChange={(event) => setCustomBet(event.target.value)}
              className="w-full rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-xs text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] focus:border-violet-500/40 focus:outline-none"
            />
            {actualBet > 0 && actualBet < minBet && (
              <p className="mt-1.5 text-[10px] text-red-500">低于最小押注 {minBet.toLocaleString()}</p>
            )}
            {selectedUnit && actualBet > selectedMaxBet && (
              <p className="mt-1.5 text-[10px] text-red-500">超过本局上限 {selectedMaxBet.toLocaleString()}</p>
            )}
          </div>

          <div className="rounded-2xl border border-violet-500/30 bg-violet-500/5 p-4 text-center">
            <p className="text-sm text-[var(--color-text-primary)]">
              押注 <span className="font-bold text-violet-600">{actualBet.toLocaleString()}</span> {selectedUnit?.name ?? ''}
            </p>
            <p className="mt-1 text-xs text-[var(--color-text-muted)]">
              只有三符完全相同才中奖，押注兵力先扣除
            </p>
            <button
              type="button"
              onClick={() => void startSlotRound()}
              disabled={!canBet}
              className="mt-3 inline-flex items-center justify-center gap-2 rounded-xl bg-violet-600 px-8 py-3 text-sm font-bold text-white shadow-lg shadow-violet-500/20 transition-colors hover:bg-violet-700 disabled:cursor-not-allowed disabled:opacity-50 cursor-pointer"
            >
              {resolvingRound ? <Loader2 size={16} className="animate-spin" /> : <Play size={16} />}
              {resolvingRound ? '结算中...' : '启动轮转'}
            </button>
          </div>
        </div>
      )}

      {phase === 'spinning' && (
        <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-5 text-center">
          <p className="text-sm font-medium text-violet-600">{lockedReels < 3 ? '天机轮盘转动中...' : '落定'}</p>
          <p className="mt-1 text-xs text-[var(--color-text-muted)]">押注 {pendingRound?.betUnit} ×{pendingRound?.betAmount.toLocaleString()}</p>
        </div>
      )}

      {phase === 'result' && pendingRound && (
        <SlotResultModal round={pendingRound} onClose={resetRound} />
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

interface SlotResultModalProps {
  round: SlotRoundResult
  onClose: () => void
}

// SlotResultModal 展示一局天机轮转结果。
const SlotResultModal: FC<SlotResultModalProps> = ({ round, onClose }) => {
  const [visible, setVisible] = useState(false)

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
      <div className={`relative w-full max-w-xs overflow-hidden rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] shadow-[0_24px_60px_rgba(15,23,42,0.3)] transition-all duration-200 ${visible ? 'translate-y-0 scale-100 opacity-100' : 'translate-y-4 scale-95 opacity-0'}`}>
        <div className={`px-4 py-4 text-center ${round.won ? 'bg-emerald-500/10' : 'bg-red-500/10'}`}>
          {round.won ? <Trophy size={28} className="mx-auto mb-1 text-emerald-500" /> : <XCircle size={28} className="mx-auto mb-1 text-red-500" />}
          <h2 className={`text-lg font-bold ${round.won ? 'text-emerald-600' : 'text-red-600'}`}>
            {round.won ? '三符同现' : '未中奖'}
          </h2>
          <p className="mt-1 text-[10px] text-[var(--color-text-muted)]">{round.symbolNames.join(' / ')}</p>
        </div>
        <div className="space-y-3 p-4">
          <div className="rounded-xl bg-[var(--color-surface-dim)] p-3">
            <div className="flex justify-between text-xs">
              <span className="text-[var(--color-text-muted)]">押注</span>
              <span className="font-semibold text-[var(--color-text-primary)]">{round.betUnit} ×{round.betAmount.toLocaleString()}</span>
            </div>
            <div className="mt-2 flex justify-between text-xs">
              <span className="text-[var(--color-text-muted)]">倍率</span>
              <span className="font-semibold text-violet-600">×{round.multiplier || 0}</span>
            </div>
            <div className="mt-2 flex justify-between text-xs">
              <span className="text-[var(--color-text-muted)]">库存奖励</span>
              <span className={`font-semibold ${round.won ? 'text-emerald-600' : 'text-red-500'}`}>{round.winAmount.toLocaleString()}</span>
            </div>
          </div>
          <button
            type="button"
            onClick={handleClose}
            className="w-full rounded-xl bg-violet-600 py-2.5 text-sm font-bold text-white transition-colors hover:bg-violet-700 cursor-pointer"
          >
            继续
          </button>
        </div>
      </div>
    </div>
  )
}

export default SlotMachineGame
