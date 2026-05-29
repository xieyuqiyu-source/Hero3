import { useState, useRef, useEffect, type FC } from 'react'
import { X, Award, TrendingUp, Anchor, ChevronDown } from 'lucide-react'
import { useGameStore } from '@/store/gameStore'
import { gameApi } from '@/api/game'

interface FishCatch {
  name: string
  rarity: 'common' | 'rare' | 'epic' | 'legendary'
  reward: string
  rewardAmount: number
  description: string
  emoji: string
}

const FISH_POOL: FishCatch[] = [
  { name: '草鱼', rarity: 'common', reward: '青州军', rewardAmount: 800, description: '常见的淡水鱼，肉质鲜美', emoji: '🐟' },
  { name: '鲤鱼', rarity: 'common', reward: '贪狼营', rewardAmount: 800, description: '跃龙门的吉祥之鱼', emoji: '🐠' },
  { name: '鲈鱼', rarity: 'common', reward: '禁卫甲士', rewardAmount: 600, description: '清蒸最佳的上等食材', emoji: '🐡' },
  { name: '锦鲤', rarity: 'common', reward: '麒麟卫', rewardAmount: 600, description: '色彩斑斓的观赏鱼', emoji: '🎏' },
  { name: '泥鳅', rarity: 'common', reward: '青州军', rewardAmount: 1000, description: '滑不溜秋但营养丰富', emoji: '🪱' },
  { name: '金龙鱼', rarity: 'rare', reward: '骁骑营', rewardAmount: 5000, description: '金光闪闪的珍贵鱼种', emoji: '✨' },
  { name: '银鲨', rarity: 'rare', reward: '西凉铁骑', rewardAmount: 5000, description: '银色鳞片如铠甲般坚硬', emoji: '🦈' },
  { name: '玄武龟', rarity: 'rare', reward: '青龙军', rewardAmount: 8000, description: '传说中玄武的后裔', emoji: '🐢' },
  { name: '雷电鳗', rarity: 'rare', reward: '虎卫', rewardAmount: 6000, description: '体内蕴含雷电之力', emoji: '⚡' },
  { name: '九尾金鲤', rarity: 'rare', reward: '骁骑营', rewardAmount: 10000, description: '九条尾鳍如扇般展开', emoji: '🌊' },
  { name: '虎鲸', rarity: 'epic', reward: '虎豹骑', rewardAmount: 50000, description: '海中霸主，力量惊人', emoji: '🐋' },
  { name: '蛟龙', rarity: 'epic', reward: '南蛮象', rewardAmount: 50000, description: '即将化龙的水中神兽', emoji: '🐲' },
  { name: '凤凰鱼', rarity: 'epic', reward: '木牛流马', rewardAmount: 800, description: '浴火重生的神秘鱼种', emoji: '🔥' },
  { name: '白泽', rarity: 'epic', reward: '虎豹骑', rewardAmount: 80000, description: '通晓万物的上古瑞兽', emoji: '🦄' },
  { name: '鲲鹏', rarity: 'legendary', reward: '汉室宗亲', rewardAmount: 10000, description: '北冥有鱼，其名为鲲', emoji: '🌌' },
  { name: '神龙', rarity: 'legendary', reward: '土族', rewardAmount: 5000, description: '万灵之首，至高无上', emoji: '🐉' },
  { name: '混沌', rarity: 'legendary', reward: '汉室宗亲', rewardAmount: 20000, description: '天地未分之初的原始神兽', emoji: '🌀' },
]

const RARITY_CONFIG = {
  common: { label: '普通', color: 'text-slate-500', bg: 'bg-slate-500/10', border: 'border-slate-500/20', weight: 65, glow: '' },
  rare: { label: '稀有', color: 'text-blue-500', bg: 'bg-blue-500/10', border: 'border-blue-500/20', weight: 22, glow: 'shadow-[0_0_15px_rgba(59,130,246,0.3)]' },
  epic: { label: '史诗', color: 'text-purple-500', bg: 'bg-purple-500/10', border: 'border-purple-500/20', weight: 8, glow: 'shadow-[0_0_25px_rgba(168,85,247,0.4)]' },
  legendary: { label: '传说', color: 'text-amber-500', bg: 'bg-amber-500/10', border: 'border-amber-500/20', weight: 1.5, glow: 'shadow-[0_0_40px_rgba(245,158,11,0.5)]' },
}

interface BaitType {
  id: string
  name: string
  icon: string
  description: string
  rarityBoost: number
}

const BAITS: BaitType[] = [
  { id: 'worm', name: '蚯蚓', icon: '🪱', description: '普通鱼饵', rarityBoost: 1 },
  { id: 'shrimp', name: '鲜虾', icon: '🦐', description: '稀有+20%', rarityBoost: 1.2 },
  { id: 'golden', name: '金饵', icon: '✨', description: '史诗+50%', rarityBoost: 1.5 },
  { id: 'dragon', name: '龙涎', icon: '🐲', description: '传说翻倍', rarityBoost: 2 },
]

type GamePhase = 'idle' | 'casting' | 'waiting' | 'biting' | 'reeling' | 'caught' | 'escaped'

interface FishingStats {
  totalCasts: number
  totalCaught: number
  combo: number
  bestCombo: number
  legendaryCount: number
  epicCount: number
}

const FishingGame: FC = () => {
  const [phase, setPhase] = useState<GamePhase>('idle')
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const [catchResult, setCatchResult] = useState<FishCatch | null>(null)
  const [showResult, setShowResult] = useState(false)
  const [castPower, setCastPower] = useState(0)
  const [powerDirection, setPowerDirection] = useState(1)
  const [selectedBait, setSelectedBait] = useState<BaitType>(BAITS[0])
  const [stats, setStats] = useState<FishingStats>({
    totalCasts: 0, totalCaught: 0, combo: 0, bestCombo: 0, legendaryCount: 0, epicCount: 0,
  })
  const [recentCatches, setRecentCatches] = useState<FishCatch[]>([])
  const [showBaitSelect, setShowBaitSelect] = useState(false)
  const [bubbles, setBubbles] = useState<{ id: number; x: number; size: number; delay: number }[]>([])
  const [fishShadow, setFishShadow] = useState<{ x: number; visible: boolean }>({ x: 50, visible: false })
  const [tensionLevel, setTensionLevel] = useState(0) // 0-3 for waiting phase visual

  const biteTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const escapeTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const powerAnimRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const bubbleTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const shadowTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const tensionTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)

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

  const rollFish = (): FishCatch => {
    const boost = selectedBait.rarityBoost
    const weights = {
      common: RARITY_CONFIG.common.weight,
      rare: RARITY_CONFIG.rare.weight * boost,
      epic: RARITY_CONFIG.epic.weight * boost,
      legendary: RARITY_CONFIG.legendary.weight * boost,
    }
    const comboBonus = 1 + Math.floor(stats.combo / 3) * 0.1
    weights.rare *= comboBonus
    weights.epic *= comboBonus
    weights.legendary *= comboBonus

    const totalWeight = Object.values(weights).reduce((sum, w) => sum + w, 0)
    let roll = Math.random() * totalWeight
    let selectedRarity: FishCatch['rarity'] = 'common'
    for (const [rarity, weight] of Object.entries(weights)) {
      roll -= weight
      if (roll <= 0) { selectedRarity = rarity as FishCatch['rarity']; break }
    }
    const candidates = FISH_POOL.filter(f => f.rarity === selectedRarity)
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
    setPowerDirection(1)
    // Power charges while holding
    powerAnimRef.current = setInterval(() => {
      setCastPower(prev => {
        let dir = powerDirection
        let next = prev + dir * 2.5
        if (next >= 100) { next = 100; setPowerDirection(-1) }
        if (next <= 0) { next = 0; setPowerDirection(1) }
        return next
      })
    }, 25)
  }

  // Long press: start charging on pointer down
  const handleCastDown = () => {
    if (phase === 'idle') {
      startCasting()
    }
  }

  // Release: confirm cast on pointer up
  const handleCastUp = () => {
    if (phase === 'casting') {
      confirmCast()
    }
  }

  const confirmCast = () => {
    if (powerAnimRef.current) clearInterval(powerAnimRef.current)
    setPhase('waiting')
    setStats(s => ({ ...s, totalCasts: s.totalCasts + 1 }))
    startBubbles()

    // Gradually increase tension
    let tension = 0
    tensionTimerRef.current = setInterval(() => {
      tension++
      setTensionLevel(Math.min(tension, 3))
    }, 800)

    // Show fish shadow approaching
    setTimeout(() => startFishShadow(), 800 + Math.random() * 1000)

    const baseDelay = 2000 + Math.random() * 3000
    const delay = baseDelay / (selectedBait.rarityBoost * 0.7 + 0.3)
    biteTimerRef.current = setTimeout(() => {
      stopEffects()
      setPhase('biting')
      const escapeTime = castPower >= 60 && castPower <= 80 ? 2500 : 1500
      escapeTimerRef.current = setTimeout(() => {
        setPhase('escaped')
        setStats(s => ({ ...s, combo: 0 }))
      }, escapeTime)
    }, delay)
  }

  const reel = () => {
    if (phase !== 'biting') return
    if (escapeTimerRef.current) clearTimeout(escapeTimerRef.current)
    setPhase('reeling')

    const sweetSpot = castPower >= 60 && castPower <= 80
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
        // 上报钓鱼记录到后端
        if (activePlayerId) {
          gameApi.saveMiniGameRecord(activePlayerId, 'fishing', fish.name, fish.rarity, fish.reward, fish.rewardAmount).catch(() => {})
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

  return (
    <div className="max-w-md mx-auto">
      {/* Header */}
      <div className="flex items-center justify-between mb-3">
        <div>
          <h2 className="text-base font-bold text-[var(--color-text-primary)] flex items-center gap-1.5">
            🎣 仙池垂钓
          </h2>
        </div>
        {stats.combo > 0 && (
          <div className="flex items-center gap-1 px-2 py-0.5 rounded-full bg-amber-500/10 border border-amber-500/20">
            <TrendingUp size={10} className="text-amber-500" />
            <span className="text-[10px] font-bold text-amber-600">×{stats.combo}</span>
          </div>
        )}
      </div>

      {/* Mini Stats */}
      <div className="flex gap-3 mb-3 text-[10px] text-[var(--color-text-muted)]">
        <span>抛竿 <b className="text-[var(--color-text-primary)]">{stats.totalCasts}</b></span>
        <span>钓获 <b className="text-[var(--color-text-primary)]">{stats.totalCaught}</b></span>
        <span>最佳连击 <b className="text-amber-600">{stats.bestCombo}</b></span>
        {stats.legendaryCount > 0 && <span>传说 <b className="text-amber-500">{stats.legendaryCount}</b></span>}
      </div>

      {/* Bait - compact */}
      <div className="mb-3">
        <button
          type="button"
          onClick={() => phase === 'idle' && setShowBaitSelect(!showBaitSelect)}
          className="inline-flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg bg-[var(--color-surface)] border border-[var(--color-border)] cursor-pointer hover:border-blue-500/30 transition-colors text-[11px]"
        >
          <Anchor size={10} className="text-blue-500" />
          <span className="text-[var(--color-text-muted)]">鱼饵</span>
          <span className="font-medium text-[var(--color-text-primary)]">{selectedBait.icon} {selectedBait.name}</span>
          <ChevronDown size={10} className="text-[var(--color-text-muted)]" />
        </button>
        {showBaitSelect && (
          <div className="mt-1.5 flex gap-1.5 flex-wrap">
            {BAITS.map((bait) => (
              <button
                key={bait.id}
                type="button"
                onClick={() => { setSelectedBait(bait); setShowBaitSelect(false) }}
                className={`
                  inline-flex items-center gap-1 px-2 py-1 rounded-lg text-[10px] cursor-pointer transition-all
                  ${selectedBait.id === bait.id
                    ? 'bg-blue-500/10 border border-blue-500/30 text-blue-600 font-medium'
                    : 'bg-[var(--color-surface-dim)] border border-transparent text-[var(--color-text-secondary)] hover:border-[var(--color-border)]'
                  }
                `}
              >
                {bait.icon} {bait.name}
                <span className="text-[8px] text-[var(--color-text-muted)]">{bait.description}</span>
              </button>
            ))}
          </div>
        )}
      </div>

      {/* ===== GAME AREA ===== */}
      <div
        className="relative rounded-2xl overflow-hidden min-h-[340px] flex flex-col select-none touch-none"
        onPointerDown={() => {
          if (phase === 'idle') handleCastDown()
          else if (phase === 'biting') reel()
        }}
        onPointerUp={() => {
          if (phase === 'casting') handleCastUp()
        }}
        onPointerCancel={() => {
          if (phase === 'casting') handleCastUp()
        }}
      >
        {/* Sky gradient */}
        <div className="absolute inset-0 bg-gradient-to-b from-sky-900/80 via-sky-800/60 to-blue-900/90" />
        {/* Stars */}
        <div className="absolute top-3 left-[20%] w-1 h-1 rounded-full bg-white/40 animate-pulse" />
        <div className="absolute top-5 left-[60%] w-0.5 h-0.5 rounded-full bg-white/30 animate-pulse" style={{ animationDelay: '0.5s' }} />
        <div className="absolute top-2 left-[80%] w-1 h-1 rounded-full bg-white/20 animate-pulse" style={{ animationDelay: '1s' }} />
        {/* Moon */}
        <div className="absolute top-4 right-6 w-6 h-6 rounded-full bg-amber-100/20 shadow-[0_0_10px_rgba(251,191,36,0.2)]" />

        {/* Water surface line */}
        <div className="absolute top-[35%] left-0 right-0 h-[2px] bg-gradient-to-r from-transparent via-blue-300/30 to-transparent" />

        {/* Water body */}
        <div className="absolute top-[35%] bottom-0 left-0 right-0 bg-gradient-to-b from-blue-800/50 via-blue-900/70 to-slate-900/90" />

        {/* Underwater light rays */}
        <div className="absolute top-[40%] left-[30%] w-[2px] h-[40%] bg-gradient-to-b from-blue-300/15 to-transparent rotate-[5deg]" />
        <div className="absolute top-[38%] left-[55%] w-[2px] h-[35%] bg-gradient-to-b from-blue-300/10 to-transparent rotate-[-3deg]" />

        {/* Bubbles */}
        {bubbles.map(b => (
          <div
            key={b.id}
            className="absolute rounded-full border border-blue-300/30 bg-blue-200/10 animate-float-up"
            style={{
              left: `${b.x}%`,
              bottom: '10%',
              width: b.size,
              height: b.size,
              animationDelay: `${b.delay}s`,
              animationDuration: '2s',
            }}
          />
        ))}

        {/* Fish shadow under water */}
        {fishShadow.visible && (
          <div
            className="absolute top-[55%] transition-all duration-500 ease-in-out"
            style={{ left: `${fishShadow.x}%`, transform: 'translateX(-50%)' }}
          >
            <div className="w-8 h-3 rounded-full bg-slate-300/15 blur-[2px] animate-pulse" />
          </div>
        )}

        {/* Fishing line */}
        {(phase === 'waiting' || phase === 'biting') && (
          <div className="absolute top-[10%] left-[50%] w-[1px] h-[45%] bg-gradient-to-b from-slate-300/50 to-slate-400/30">
            {/* Float/bobber */}
            <div className={`absolute -bottom-1 -left-[3px] w-[7px] h-[7px] rounded-full border border-red-400 ${phase === 'biting' ? 'bg-red-500 animate-bounce' : 'bg-red-400/80'}`} />
          </div>
        )}

        {/* Content overlay */}
        <div className="relative flex-1 flex flex-col items-center justify-center p-6 z-10">

          {/* Idle */}
          {phase === 'idle' && (
            <div className="text-center space-y-5">
              <div className="relative">
                <div className="w-16 h-16 rounded-full bg-blue-500/15 flex items-center justify-center mx-auto border border-blue-400/20 backdrop-blur-sm">
                  <span className="text-2xl">🎣</span>
                </div>
              </div>
              <div>
                <p className="text-[11px] text-blue-200/60 mb-2">
                  {stats.combo > 0 ? `🔥 连击 ${stats.combo}，稀有概率提升中` : '月色正好，适合垂钓'}
                </p>
                <p className="text-xs text-blue-300/80 font-medium animate-pulse">
                  按住屏幕蓄力
                </p>
              </div>
            </div>
          )}

          {/* Casting - Power bar (long press) */}
          {phase === 'casting' && (
            <div className="text-center space-y-4 w-full max-w-[280px]">
              <p className="text-xs text-blue-100/80 font-medium">蓄力中...松手抛竿</p>
              <div className="relative">
                <div className="w-full h-4 rounded-full bg-slate-800/60 border border-slate-600/30 overflow-hidden backdrop-blur-sm">
                  <div
                    className="h-full rounded-full transition-all duration-[25ms]"
                    style={{
                      width: `${castPower}%`,
                      background: castPower >= 60 && castPower <= 80
                        ? 'linear-gradient(90deg, #22c55e, #16a34a)'
                        : castPower > 80
                          ? 'linear-gradient(90deg, #f59e0b, #ef4444)'
                          : 'linear-gradient(90deg, #60a5fa, #3b82f6)',
                    }}
                  />
                </div>
                {/* Sweet spot markers */}
                <div className="absolute top-0 bottom-0 left-[60%] w-[20%] border-x border-green-400/50 pointer-events-none rounded" />
              </div>
              <div className="flex justify-between text-[9px] text-blue-200/40 px-0.5">
                <span>弱</span>
                <span className="text-green-400/70">最佳 60-80%</span>
                <span>强</span>
              </div>
              <p className="text-lg font-bold text-blue-100/90">{castPower}%</p>
            </div>
          )}

          {/* Waiting */}
          {phase === 'waiting' && (
            <div className="text-center space-y-4">
              <div className="relative">
                {/* Tension indicator rings */}
                {tensionLevel >= 1 && <div className="absolute inset-0 w-12 h-12 mx-auto rounded-full border border-blue-400/20 animate-ping" style={{ animationDuration: '2s' }} />}
                {tensionLevel >= 2 && <div className="absolute inset-0 w-12 h-12 mx-auto rounded-full border border-blue-300/30 animate-ping" style={{ animationDuration: '1.5s', animationDelay: '0.3s' }} />}
                <div className="w-12 h-12 rounded-full bg-blue-500/10 flex items-center justify-center mx-auto border border-blue-400/20 backdrop-blur-sm">
                  <span className="text-lg animate-pulse">🫧</span>
                </div>
              </div>
              <div>
                <p className="text-xs text-blue-100/70 font-medium">
                  {tensionLevel === 0 ? '鱼线入水...' : tensionLevel === 1 ? '水面微动...' : tensionLevel === 2 ? '似乎有东西靠近...' : '就快了...'}
                </p>
                <p className="text-[9px] text-blue-200/40 mt-1">{selectedBait.icon} {selectedBait.name}</p>
              </div>
            </div>
          )}

          {/* Biting! - tap anywhere to reel */}
          {phase === 'biting' && (
            <div className="text-center space-y-3">
              {/* Ripple rings */}
              <div className="relative w-20 h-20 mx-auto">
                <div className="absolute inset-0 rounded-full border-2 border-red-400/40 animate-ping" style={{ animationDuration: '1s' }} />
                <div className="absolute inset-2 rounded-full border border-red-400/30 animate-ping" style={{ animationDuration: '1.2s', animationDelay: '0.2s' }} />
                <div className="absolute inset-4 rounded-full border border-red-400/20 animate-ping" style={{ animationDuration: '1.4s', animationDelay: '0.4s' }} />
                <div className="absolute inset-0 rounded-full bg-red-500/15 flex items-center justify-center border-2 border-red-400/50 shadow-[0_0_25px_rgba(239,68,68,0.3)] animate-pulse">
                  <span className="text-2xl animate-bounce">🐟</span>
                </div>
              </div>
              <p className="text-sm font-bold text-red-400 animate-pulse">上钩了！</p>
              <p className="text-[10px] text-red-300/60">点击屏幕收杆</p>
            </div>
          )}

          {/* Reeling */}
          {phase === 'reeling' && (
            <div className="text-center space-y-3">
              <div className="w-14 h-14 rounded-full bg-amber-500/15 flex items-center justify-center mx-auto border border-amber-400/20">
                <span className="text-xl animate-spin" style={{ animationDuration: '0.8s' }}>⭐</span>
              </div>
              <p className="text-xs text-blue-100/70">拉起中...</p>
              <div className="flex justify-center gap-1.5">
                {[0, 1, 2, 3].map(i => (
                  <div key={i} className="w-1.5 h-1.5 rounded-full bg-amber-400/60 animate-bounce" style={{ animationDelay: `${i * 0.12}s` }} />
                ))}
              </div>
            </div>
          )}

          {/* Escaped */}
          {phase === 'escaped' && (
            <div className="text-center space-y-4">
              <div className="w-14 h-14 rounded-full bg-slate-500/10 flex items-center justify-center mx-auto border border-slate-500/20">
                <span className="text-xl opacity-50">💨</span>
              </div>
              <div>
                <p className="text-xs text-blue-100/60">跑掉了...</p>
                {stats.combo > 0 && <p className="text-[9px] text-red-400/70 mt-1">连击中断</p>}
              </div>
              <button
                type="button"
                onClick={reset}
                className="px-6 py-2 rounded-xl bg-blue-500/80 text-white text-xs font-medium hover:bg-blue-500 cursor-pointer transition-all"
              >
                再来
              </button>
            </div>
          )}

          {/* Caught - suspense reveal */}
          {phase === 'caught' && !showResult && catchResult && (
            <div className="text-center space-y-3">
              <div className={`
                w-20 h-20 rounded-full flex items-center justify-center mx-auto transition-all duration-500
                ${catchResult.rarity === 'legendary'
                  ? 'bg-amber-500/20 border-2 border-amber-400/60 shadow-[0_0_40px_rgba(245,158,11,0.5)] animate-pulse'
                  : catchResult.rarity === 'epic'
                  ? 'bg-purple-500/20 border-2 border-purple-400/50 shadow-[0_0_25px_rgba(168,85,247,0.4)] animate-pulse'
                  : catchResult.rarity === 'rare'
                  ? 'bg-blue-500/20 border-2 border-blue-400/40 shadow-[0_0_15px_rgba(59,130,246,0.3)]'
                  : 'bg-slate-500/10 border border-slate-400/20'}
              `}>
                <span className={`text-3xl ${catchResult.rarity === 'legendary' ? 'animate-bounce' : ''}`}>
                  {catchResult.emoji}
                </span>
              </div>
              <p className={`text-sm font-bold animate-pulse
                ${catchResult.rarity === 'legendary' ? 'text-amber-400' :
                  catchResult.rarity === 'epic' ? 'text-purple-400' :
                  catchResult.rarity === 'rare' ? 'text-blue-400' :
                  'text-blue-100/80'}
              `}>
                {catchResult.rarity === 'legendary' ? '✨ 传说之物浮出水面...' :
                 catchResult.rarity === 'epic' ? '💎 史诗灵兽！' :
                 catchResult.rarity === 'rare' ? '🌟 稀有！' :
                 '钓到了'}
              </p>
            </div>
          )}
        </div>
      </div>

      {/* Recent Catches */}
      {recentCatches.length > 0 && (
        <div className="mt-3 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2.5">
          <div className="flex items-center gap-1.5 mb-1.5">
            <Award size={10} className="text-[var(--color-accent)]" />
            <span className="text-[10px] font-medium text-[var(--color-text-secondary)]">最近钓获</span>
          </div>
          <div className="flex flex-wrap gap-1">
            {recentCatches.map((fish, i) => {
              const cfg = RARITY_CONFIG[fish.rarity]
              return (
                <span key={i} className={`text-[9px] px-1.5 py-0.5 rounded ${cfg.bg} ${cfg.color} font-medium`}>
                  {fish.emoji} {fish.name}
                </span>
              )
            })}
          </div>
        </div>
      )}

      {/* Result Modal */}
      {showResult && catchResult && (
        <CatchResultModal fish={catchResult} combo={stats.combo} onClose={reset} />
      )}
    </div>
  )
}

/* ---------- Catch Result Modal ---------- */
interface CatchResultModalProps {
  fish: FishCatch
  combo: number
  onClose: () => void
}

const CatchResultModal: FC<CatchResultModalProps> = ({ fish, combo, onClose }) => {
  const [visible, setVisible] = useState(false)
  const config = RARITY_CONFIG[fish.rarity]

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
        className={`absolute inset-0 transition-opacity duration-300 ${visible ? 'opacity-100' : 'opacity-0'}
          ${fish.rarity === 'legendary' ? 'bg-amber-950/60 backdrop-blur-[6px]' :
            fish.rarity === 'epic' ? 'bg-purple-950/60 backdrop-blur-[5px]' :
            'bg-slate-900/50 backdrop-blur-[4px]'}
        `}
        onClick={handleClose}
      />
      <div className={`
        relative w-full max-w-[300px] rounded-2xl overflow-hidden
        bg-[var(--color-surface)] border
        transition-all duration-300
        ${fish.rarity === 'legendary' ? 'border-amber-500/40 shadow-[0_0_60px_rgba(245,158,11,0.2)]' :
          fish.rarity === 'epic' ? 'border-purple-500/40 shadow-[0_0_40px_rgba(168,85,247,0.15)]' :
          'border-[var(--color-border)] shadow-[0_24px_60px_rgba(15,23,42,0.3)]'}
        ${visible ? 'opacity-100 scale-100 translate-y-0' : 'opacity-0 scale-95 translate-y-4'}
      `}>
        {/* Header */}
        <div className={`px-4 py-5 text-center relative overflow-hidden ${config.bg}`}>
          {fish.rarity === 'legendary' && (
            <>
              <div className="absolute inset-0 bg-gradient-to-r from-amber-500/0 via-amber-500/20 to-amber-500/0 animate-pulse" />
              <div className="absolute inset-0 bg-[radial-gradient(circle,rgba(245,158,11,0.15)_0%,transparent_70%)]" />
            </>
          )}
          {fish.rarity === 'epic' && (
            <div className="absolute inset-0 bg-gradient-to-r from-purple-500/0 via-purple-500/15 to-purple-500/0 animate-pulse" />
          )}
          <span className={`text-4xl block mb-2 relative ${fish.rarity === 'legendary' ? 'animate-bounce' : ''}`}>{fish.emoji}</span>
          <h2 className={`text-base font-bold ${config.color} relative`}>
            {fish.rarity === 'legendary' ? '🐲 传说降临！' :
             fish.rarity === 'epic' ? '💎 史诗钓获！' :
             fish.rarity === 'rare' ? '🌟 稀有鱼种！' :
             '钓到了'}
          </h2>
          {combo > 1 && (
            <p className="text-[10px] text-amber-600 font-medium mt-1 relative">🔥 连击 ×{combo}</p>
          )}
          <button
            type="button"
            onClick={handleClose}
            className="absolute top-3 right-3 p-1 rounded-full hover:bg-white/20 cursor-pointer"
          >
            <X size={14} className="text-[var(--color-text-muted)]" />
          </button>
        </div>

        {/* Body */}
        <div className="px-4 py-4 space-y-3 text-center">
          <div>
            <p className="text-base font-bold text-[var(--color-text-primary)]">{fish.name}</p>
            <span className={`inline-block mt-1 text-[9px] px-2 py-0.5 rounded-full ${config.bg} ${config.color} font-medium border ${config.border}`}>
              {config.label}
            </span>
            <p className="text-[10px] text-[var(--color-text-muted)] mt-2 italic leading-relaxed">"{fish.description}"</p>
          </div>

          <div className={`rounded-xl p-3.5 ${config.bg} border ${config.border} ${config.glow}`}>
            <p className="text-[10px] text-[var(--color-text-muted)] mb-1">可兑换</p>
            <p className={`${fish.rarity === 'legendary' || fish.rarity === 'epic' ? 'text-xl' : 'text-base'} font-bold ${config.color}`}>
              {fish.reward}
            </p>
            <p className={`${fish.rarity === 'legendary' || fish.rarity === 'epic' ? 'text-lg' : 'text-sm'} font-bold ${config.color} mt-0.5`}>
              ×{fish.rewardAmount.toLocaleString()}
            </p>
            {fish.rewardAmount >= 10000 && (
              <p className="text-[10px] text-amber-600 mt-1.5 font-medium">
                {fish.rewardAmount >= 50000 ? '🎉 超级大奖！' : '🎉 丰厚奖励！'}
              </p>
            )}
          </div>

          <p className="text-[9px] text-[var(--color-text-muted)]">* 奖励将在系统对接后发放</p>
        </div>

        {/* Footer */}
        <div className="px-4 py-3 border-t border-[var(--color-border)]">
          <button
            type="button"
            onClick={handleClose}
            className="w-full px-4 py-2.5 rounded-xl text-sm font-bold bg-[var(--color-accent)] text-white hover:opacity-90 cursor-pointer transition-opacity"
          >
            继续垂钓
          </button>
        </div>
      </div>
    </div>
  )
}

export default FishingGame