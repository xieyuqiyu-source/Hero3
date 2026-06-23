import { useEffect, useRef, useState, type FC } from 'react'
import { Award, TrendingUp } from 'lucide-react'
import { gameApi } from '@/api/game'
import { toast } from '@/components/ui'
import { useConfigStore } from '@/store/configStore'
import { useGameStore } from '@/store/gameStore'
import type { MiniGameRecord } from '@/types/game'
import { FishingBaitSelector } from './fishing/FishingBaitSelector'
import { FishingInventoryModal } from './fishing/FishingInventoryModal'
import { FishingPondScene } from './fishing/FishingPondScene'
import { FishingResultModal } from './fishing/FishingResultModal'
import { BAITS, FISH_POOL, RARITY_CONFIG } from './fishing/fishingConfig'
import type { BaitType, Bubble, FishCatch, FishShadow, GamePhase, FishingStats } from './fishing/types'

const FishingGame: FC = () => {
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const gameState = useGameStore((s) => s.state)
  const patchState = useGameStore((s) => s.patchState)
  const units = useConfigStore((s) => s.units)

  const [phase, setPhase] = useState<GamePhase>('idle')
  const [catchResult, setCatchResult] = useState<FishCatch | null>(null)
  const [showResult, setShowResult] = useState(false)
  const [castPower, setCastPower] = useState(0)
  const [selectedBait, setSelectedBait] = useState<BaitType>(BAITS[0])
  const [stats, setStats] = useState<FishingStats>({
    totalCasts: 0, totalCaught: 0, combo: 0, bestCombo: 0, legendaryCount: 0, epicCount: 0,
  })
  const [recentCatches, setRecentCatches] = useState<FishCatch[]>([])
  const [showBaitSelect, setShowBaitSelect] = useState(false)
  const [bubbles, setBubbles] = useState<Bubble[]>([])
  const [fishShadow, setFishShadow] = useState<FishShadow>({ x: 50, visible: false })
  const [biteSpot, setBiteSpot] = useState({ x: 58, y: 58 })
  const [tensionLevel, setTensionLevel] = useState(0)
  const [records, setRecords] = useState<MiniGameRecord[]>([])
  const [recordsLoading, setRecordsLoading] = useState(false)
  const [recordsTotal, setRecordsTotal] = useState(0)
  const [recordsHasMore, setRecordsHasMore] = useState(false)
  const [recordsOffset, setRecordsOffset] = useState(0)
  const [redeemingId, setRedeemingId] = useState('')
  const [showInventory, setShowInventory] = useState(false)

  const biteTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const escapeTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const powerAnimRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const bubbleTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const shadowTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const tensionTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const powerDirectionRef = useRef(1)

  useEffect(() => {
    return () => {
      if (biteTimerRef.current) clearTimeout(biteTimerRef.current)
      if (escapeTimerRef.current) clearTimeout(escapeTimerRef.current)
      if (powerAnimRef.current) clearInterval(powerAnimRef.current)
      if (bubbleTimerRef.current) clearInterval(bubbleTimerRef.current)
      if (shadowTimerRef.current) clearInterval(shadowTimerRef.current)
      if (tensionTimerRef.current) clearInterval(tensionTimerRef.current)
    }
  }, [])

  const loadRecords = async (offset = 0) => {
    if (!activePlayerId) return
    setRecordsLoading(true)
    try {
      const result = await gameApi.listMiniGameRecords(activePlayerId, 100, offset, 'fishing')
      setRecords(result.records)
      setRecordsTotal(result.totalRecords)
      setRecordsHasMore(result.hasMore)
      setRecordsOffset(result.offset)
    } finally {
      setRecordsLoading(false)
    }
  }

  useEffect(() => {
    void loadRecords()
  }, [activePlayerId])

  const isFactionUnit = (unitName: string): boolean => {
    const faction = gameState?.player.faction
    if (!faction || !units?.[faction]) return false
    return Object.values(units[faction]).some(config => config.name === unitName)
  }

  const handleRedeemGroup = async (unitName: string, groupRecords: MiniGameRecord[]) => {
    if (!activePlayerId || redeemingId) return
    const targets = groupRecords.filter(record => record.remainingAmount > 0 && isFactionUnit(record.rewardUnit))
    const totalAmount = targets.reduce((sum, record) => sum + record.remainingAmount, 0)
    if (targets.length === 0 || totalAmount <= 0) {
      toast.error('没有可兑换库存')
      return
    }
    setRedeemingId(unitName)
    try {
      let redeemed = 0
      let latestState = null
      for (const record of targets) {
        const result = await gameApi.redeemMiniGameReward(activePlayerId, record.id, record.remainingAmount)
        redeemed += result.redeemedAmount
        latestState = result.state
        setRecords(prev => prev.map(item => item.id === result.record.id ? result.record : item))
      }
      if (latestState) {
        patchState({ army: latestState.army, serverTime: latestState.serverTime })
      }
      toast.success(`${unitName} ×${redeemed.toLocaleString()} 已加入军队`)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '兑换失败')
    } finally {
      setRedeemingId('')
    }
  }

  const rollFish = (): FishCatch => {
    const weights = {
      common: RARITY_CONFIG.common.weight,
      rare: RARITY_CONFIG.rare.weight * selectedBait.rarityBoost,
      epic: RARITY_CONFIG.epic.weight * selectedBait.rarityBoost,
      legendary: RARITY_CONFIG.legendary.weight * selectedBait.rarityBoost,
    }
    const comboBonus = 1 + Math.floor(stats.combo / 3) * 0.1
    weights.rare *= comboBonus
    weights.epic *= comboBonus
    weights.legendary *= comboBonus

    const totalWeight = Object.values(weights).reduce((sum, weight) => sum + weight, 0)
    let roll = Math.random() * totalWeight
    let selectedRarity: FishCatch['rarity'] = 'common'
    for (const [rarity, weight] of Object.entries(weights)) {
      roll -= weight
      if (roll <= 0) {
        selectedRarity = rarity as FishCatch['rarity']
        break
      }
    }
    const candidates = FISH_POOL.filter(fish => fish.rarity === selectedRarity)
    return candidates[Math.floor(Math.random() * candidates.length)]
  }

  const startBubbles = () => {
    bubbleTimerRef.current = setInterval(() => {
      setBubbles(prev => [...prev.slice(-6), {
        id: Date.now(),
        x: 30 + Math.random() * 40,
        size: 4 + Math.random() * 8,
        delay: Math.random() * 0.5,
      }])
    }, 600)
  }

  const startFishShadow = () => {
    setFishShadow({ x: 50, visible: true })
    shadowTimerRef.current = setInterval(() => {
      setFishShadow(prev => ({
        ...prev,
        x: Math.max(20, Math.min(80, prev.x + (Math.random() - 0.5) * 15)),
      }))
    }, 400)
  }

  const stopEffects = () => {
    if (bubbleTimerRef.current) clearInterval(bubbleTimerRef.current)
    if (shadowTimerRef.current) clearInterval(shadowTimerRef.current)
    if (tensionTimerRef.current) clearInterval(tensionTimerRef.current)
    setBubbles([])
    setFishShadow({ x: 50, visible: false })
    setTensionLevel(0)
  }

  const startCasting = () => {
    setPhase('casting')
    setCastPower(0)
    powerDirectionRef.current = 1
    powerAnimRef.current = setInterval(() => {
      setCastPower(prev => {
        let next = prev + powerDirectionRef.current * 2.5
        if (next >= 100) {
          next = 100
          powerDirectionRef.current = -1
        }
        if (next <= 0) {
          next = 0
          powerDirectionRef.current = 1
        }
        return next
      })
    }, 25)
  }

  const confirmCast = () => {
    if (powerAnimRef.current) clearInterval(powerAnimRef.current)
    setPhase('waiting')
    setStats(s => ({ ...s, totalCasts: s.totalCasts + 1 }))
    setBiteSpot({
      x: 48 + Math.random() * 34,
      y: 47 + Math.random() * 24,
    })
    startBubbles()

    let tension = 0
    tensionTimerRef.current = setInterval(() => {
      tension++
      setTensionLevel(Math.min(tension, 3))
    }, 800)

    setTimeout(() => startFishShadow(), 800 + Math.random() * 1000)

    const baseDelay = 2000 + Math.random() * 3000
    const delay = baseDelay / (selectedBait.rarityBoost * 0.7 + 0.3)
    biteTimerRef.current = setTimeout(() => {
      stopEffects()
      if (Math.random() > selectedBait.biteChance) {
        setPhase('escaped')
        setStats(s => ({ ...s, combo: 0 }))
        return
      }
      setPhase('biting')
      escapeTimerRef.current = setTimeout(() => {
        setPhase('escaped')
        setStats(s => ({ ...s, combo: 0 }))
      }, selectedBait.biteWindowMs)
    }, delay)
  }

  const reel = () => {
    if (phase !== 'biting') return
    if (escapeTimerRef.current) clearTimeout(escapeTimerRef.current)
    setPhase('reeling')

    const sweetSpot = castPower >= selectedBait.sweetStart && castPower <= selectedBait.sweetEnd
    const baseChance = sweetSpot ? 0.92 : 0.55
    const comboChance = Math.min(baseChance + stats.combo * 0.02, 0.98)
    const success = Math.random() < comboChance

    setTimeout(() => {
      if (success) {
        const fish = rollFish()
        setCatchResult(fish)
        setPhase('caught')
        const suspenseDelay = fish.rarity === 'legendary' ? 2000 : fish.rarity === 'epic' ? 1200 : fish.rarity === 'rare' ? 700 : 400
        setTimeout(() => setShowResult(true), suspenseDelay)
        setRecentCatches(prev => [fish, ...prev].slice(0, 10))
        setStats(s => ({
          ...s,
          totalCaught: s.totalCaught + 1,
          combo: s.combo + 1,
          bestCombo: Math.max(s.bestCombo, s.combo + 1),
          legendaryCount: s.legendaryCount + (fish.rarity === 'legendary' ? 1 : 0),
          epicCount: s.epicCount + (fish.rarity === 'epic' ? 1 : 0),
        }))
        if (activePlayerId) {
          gameApi.saveMiniGameRecord(activePlayerId, 'fishing', fish.name, fish.rarity, fish.reward, fish.rewardAmount)
            .then(record => {
              setRecords(prev => [record, ...prev].slice(0, 200))
            })
            .catch(() => {})
        }
      } else {
        setPhase('escaped')
        setStats(s => ({ ...s, combo: 0 }))
      }
    }, 1200)
  }

  const reset = () => {
    setPhase('idle')
    setCatchResult(null)
    setShowResult(false)
    setCastPower(0)
    stopEffects()
  }

  const inventoryRecords = records.filter(record => record.remainingAmount > 0)

  return (
    <div className="mx-auto max-w-3xl">
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="mr-1 flex min-w-0 items-center gap-2">
          <h2 className="shrink-0 text-base font-black tracking-[0.08em] text-[var(--color-text-primary)]">仙池垂钓</h2>
          <span className="hidden text-[10px] text-[var(--color-text-muted)] sm:inline">像素钓场 · 涟漪出现时点击收杆</span>
        </div>

        <div className="flex flex-wrap items-center gap-2 text-[10px] text-[var(--color-text-muted)]">
          <span>抛竿 <b className="text-[var(--color-text-primary)]">{stats.totalCasts}</b></span>
          <span>钓获 <b className="text-[var(--color-text-primary)]">{stats.totalCaught}</b></span>
          <span>最佳 <b className="text-amber-600">{stats.bestCombo}</b></span>
          {stats.legendaryCount > 0 && <span>传说 <b className="text-amber-500">{stats.legendaryCount}</b></span>}
        </div>

        <div className="min-w-0 flex-1" />

        <FishingBaitSelector
          baits={BAITS}
          selectedBait={selectedBait}
          showBaitSelect={showBaitSelect}
          inventoryCount={inventoryRecords.length}
          canChangeBait={phase === 'idle'}
          onToggleBaitSelect={() => setShowBaitSelect(prev => !prev)}
          onSelectBait={(bait) => { setSelectedBait(bait); setShowBaitSelect(false) }}
          onOpenInventory={() => setShowInventory(true)}
        />

        {stats.combo > 0 && (
          <div className="flex items-center gap-1 rounded-md border-2 border-amber-500/30 bg-amber-500/10 px-2 py-0.5">
            <TrendingUp size={10} className="text-amber-500" />
            <span className="text-[10px] font-bold text-amber-600">×{stats.combo}</span>
          </div>
        )}
      </div>

      <FishingPondScene
        phase={phase}
        selectedBait={selectedBait}
        castPower={castPower}
        combo={stats.combo}
        tensionLevel={tensionLevel}
        bubbles={bubbles}
        fishShadow={fishShadow}
        biteSpot={biteSpot}
        catchResult={catchResult}
        showResult={showResult}
        onCastDown={startCasting}
        onCastUp={confirmCast}
        onReel={reel}
        onReset={reset}
      />

      {recentCatches.length > 0 && (
        <div className="mt-3 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2.5">
          <div className="mb-1.5 flex items-center gap-1.5">
            <Award size={10} className="text-[var(--color-accent)]" />
            <span className="text-[10px] font-medium text-[var(--color-text-secondary)]">最近钓获</span>
          </div>
          <div className="flex flex-wrap gap-1">
            {recentCatches.map((fish, i) => {
              const cfg = RARITY_CONFIG[fish.rarity]
              return (
                <span key={`${fish.name}-${i}`} className={`rounded px-1.5 py-0.5 text-[9px] font-medium ${cfg.bg} ${cfg.color}`}>
                  {fish.emoji} {fish.name}
                </span>
              )
            })}
          </div>
        </div>
      )}

      {showInventory && (
        <FishingInventoryModal
          records={records}
          recordsLoading={recordsLoading}
          recordsTotal={recordsTotal}
          recordsHasMore={recordsHasMore}
          recordsOffset={recordsOffset}
          recordsPageSize={100}
          redeemingId={redeemingId}
          isFactionUnit={isFactionUnit}
          onClose={() => setShowInventory(false)}
          onRefresh={() => void loadRecords(0)}
          onPageChange={(nextOffset) => void loadRecords(nextOffset)}
          onRedeemGroup={(unitName, groupRecords) => void handleRedeemGroup(unitName, groupRecords)}
        />
      )}

      {showResult && catchResult && (
        <FishingResultModal fish={catchResult} combo={stats.combo} onClose={reset} />
      )}
    </div>
  )
}

export default FishingGame
