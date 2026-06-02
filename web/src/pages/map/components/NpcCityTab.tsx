import { useState, useEffect, type FC } from 'react'
import { RefreshCw, LoaderCircle, Swords } from 'lucide-react'
import { useGameStore } from '@/store/gameStore'
import { gameApi } from '@/api/game'
import type { NpcCity, BattleReport } from '@/types/game'
import { toast } from '@/components/ui'
import NpcCityCard from './NpcCityCard'
import AttackPanel from './AttackPanel'
import BattleResultModal from './BattleResultModal'
import ScoutResultModal from './ScoutResultModal'

type NpcTier = NpcCity['tier']

const TIER_ORDER: NpcTier[] = ['small', 'medium', 'large', 'golden']
const TIER_LABELS: Record<NpcTier, string> = {
  small: '小型',
  medium: '中型',
  large: '大型',
  golden: '金色',
}

const NpcCityTab: FC = () => {
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const [cities, setCities] = useState<NpcCity[]>([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [sweeping, setSweeping] = useState(false)
  const [sweepProgress, setSweepProgress] = useState({ done: 0, total: 0, failed: 0, current: '', cityGold: 0 })
  const [lastSweepSummary, setLastSweepSummary] = useState<{ done: number; failed: number; cityGold: number } | null>(null)
  const [selectedTiers, setSelectedTiers] = useState<Record<NpcTier, boolean>>({
    small: true,
    medium: true,
    large: false,
    golden: false,
  })
  const [selectedCity, setSelectedCity] = useState<NpcCity | null>(null)
  const [battleReport, setBattleReport] = useState<BattleReport | null>(null)
  const [scoutReport, setScoutReport] = useState<BattleReport | null>(null)

  const loadCities = async () => {
    if (!activePlayerId) return
    try {
      const data = await gameApi.getNpcCities(activePlayerId)
      setCities(data.cities ?? [])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadCities()
  }, [activePlayerId])

  const handleRefresh = async () => {
    if (!activePlayerId || refreshing || sweeping) return
    setRefreshing(true)
    try {
      const data = await gameApi.refreshNpcCities(activePlayerId)
      setCities(data.cities ?? [])
      setSelectedCity(null)
    } finally {
      setRefreshing(false)
    }
  }

  const selectedSweepTargets = cities
    .filter((city) => selectedTiers[city.tier])
    .sort((a, b) => TIER_ORDER.indexOf(a.tier) - TIER_ORDER.indexOf(b.tier))

  const handleSweepTierToggle = (tier: NpcTier) => {
    if (sweeping) return
    setSelectedTiers((prev) => ({ ...prev, [tier]: !prev[tier] }))
  }

  const buildAllArmyUnits = () => {
    const army = useGameStore.getState().state?.army ?? []
    const units: Record<string, number> = {}
    for (const unit of army) {
      if (unit.amount > 0) units[unit.unitType] = unit.amount
    }
    return units
  }

  const handleSweep = async () => {
    if (!activePlayerId || sweeping || selectedSweepTargets.length === 0) return

    setSelectedCity(null)
    setSweeping(true)
    setLastSweepSummary(null)
    setSweepProgress({ done: 0, total: selectedSweepTargets.length, failed: 0, current: '', cityGold: 0 })

    let done = 0
    let failed = 0
    let totalCityGold = 0

    for (const city of selectedSweepTargets) {
      const units = buildAllArmyUnits()
      if (Object.keys(units).length === 0) {
        toast.info('当前没有可出征兵力，扫荡已停止')
        break
      }

      setSweepProgress({ done, total: selectedSweepTargets.length, failed, current: city.name, cityGold: totalCityGold })
      try {
        const result = await gameApi.attackNpc(activePlayerId, city.id, 'attack', units)
        useGameStore.getState().setState(result.state)
        if (result.state.npcState?.cities) {
          setCities(result.state.npcState.cities)
        }
        totalCityGold += result.battleReport.overflowCityGold ?? 0
        done += 1
      } catch {
        failed += 1
      }
      setSweepProgress({ done, total: selectedSweepTargets.length, failed, current: city.name, cityGold: totalCityGold })
    }

    setSweeping(false)
    setLastSweepSummary({ done, failed, cityGold: totalCityGold })
    await loadCities()
    await useGameStore.getState().loadGameState()
    toast.success(`扫荡完成：成功 ${done} 场，失败 ${failed} 场，获得 ${totalCityGold.toLocaleString()} 城金。可前往军情查看战报。`)
  }

  const handleAttackComplete = () => {
    setSelectedCity(null)
    loadCities()
    useGameStore.getState().loadGameState()
  }

  if (loading) {
    return (
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div className="h-4 w-24 rounded bg-[var(--color-surface-dim)]" />
          <div className="h-8 w-20 rounded-lg bg-[var(--color-surface-dim)]" />
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
          {Array.from({ length: 12 }).map((_, i) => (
            <div
              key={i}
              className="rounded-2xl border border-[var(--color-border)] p-3 h-[88px] backdrop-blur-sm bg-white/40 dark:bg-white/5 animate-pulse"
              style={{ animationDelay: `${i * 80}ms` }}
            />
          ))}
        </div>
        <div className="flex items-center justify-center pt-4">
          <LoaderCircle size={16} className="text-[var(--color-accent)] animate-spin" />
          <span className="text-xs text-[var(--color-text-muted)] ml-2">正在探索周边城池...</span>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {/* Info Banner */}
      <div className="px-3 py-2 rounded-xl bg-[var(--color-surface-dim)] border border-[var(--color-border)] text-[10px] text-[var(--color-text-muted)] leading-relaxed">
        <span className="font-medium text-[var(--color-text-secondary)]">说明：</span>
        一键操作将派出全部兵力。城池等级：<span className="text-slate-500">小型</span> &lt; <span className="text-blue-500">中型</span> &lt; <span className="text-purple-500">大型</span> &lt; <span className="text-amber-500">金色</span>，等级越高资源越多、守军越强。每24小时自动刷新。
      </div>

      {/* Header */}
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <span className="text-xs text-[var(--color-text-muted)]">
          共 {cities.length} 个城池
        </span>
        <button
          type="button"
          onClick={handleRefresh}
          disabled={refreshing}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium bg-[var(--color-surface-dim)] border border-[var(--color-border)] text-[var(--color-text-secondary)] hover:border-[var(--color-accent-border)] cursor-pointer transition-colors disabled:opacity-50"
        >
          <RefreshCw size={12} className={refreshing ? 'animate-spin' : ''} />
          刷新城池
        </button>
      </div>

      {/* Sweep */}
      <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-3 space-y-3">
        <div className="flex items-center justify-between gap-3 flex-wrap">
          <div>
            <div className="text-xs font-bold text-[var(--color-text-primary)]">一键扫荡</div>
            <div className="text-[10px] text-[var(--color-text-muted)] mt-0.5">
              按小型到金色顺序逐个发起一键攻击，详细结果在军情查看。
            </div>
          </div>
          <button
            type="button"
            onClick={handleSweep}
            disabled={sweeping || selectedSweepTargets.length === 0}
            className="flex items-center justify-center gap-1.5 px-4 py-2 rounded-xl text-xs font-bold bg-red-500 text-white hover:bg-red-600 cursor-pointer transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {sweeping ? <LoaderCircle size={13} className="animate-spin" /> : <Swords size={13} />}
            {sweeping ? '扫荡中' : `扫荡 ${selectedSweepTargets.length} 城`}
          </button>
        </div>

        <div className="grid grid-cols-2 sm:grid-cols-4 gap-2">
          {TIER_ORDER.map((tier) => (
            <label
              key={tier}
              className="flex items-center gap-2 px-2.5 py-2 rounded-xl bg-[var(--color-surface-dim)] border border-[var(--color-border)] cursor-pointer"
            >
              <input
                type="checkbox"
                checked={selectedTiers[tier]}
                disabled={sweeping}
                onChange={() => handleSweepTierToggle(tier)}
                className="w-3.5 h-3.5 rounded border-[var(--color-border)] accent-[var(--color-accent)]"
              />
              <span className="text-xs font-medium text-[var(--color-text-secondary)]">{TIER_LABELS[tier]}</span>
            </label>
          ))}
        </div>

        {sweeping && (
          <div className="space-y-1.5">
            <div className="flex items-center justify-between text-[10px] text-[var(--color-text-muted)]">
              <span>{sweepProgress.current ? `正在攻击：${sweepProgress.current}` : '准备出征...'}</span>
              <span>{sweepProgress.done}/{sweepProgress.total}，失败 {sweepProgress.failed}，城金 +{sweepProgress.cityGold.toLocaleString()}</span>
            </div>
            <div className="h-2 rounded-full bg-[var(--color-surface-dim)] overflow-hidden">
              <div
                className="h-full rounded-full bg-red-500 transition-all duration-300"
                style={{
                  width: `${sweepProgress.total > 0 ? Math.round((sweepProgress.done / sweepProgress.total) * 100) : 0}%`,
                }}
              />
            </div>
          </div>
        )}

        {!sweeping && lastSweepSummary && (
          <div className="rounded-xl border border-amber-400/50 bg-amber-400/10 px-3 py-2">
            <div className="flex items-center justify-between gap-3 flex-wrap">
              <span className="text-[11px] font-bold text-amber-600">上次扫荡收益</span>
              <span className="text-[11px] text-[var(--color-text-secondary)]">
                成功 {lastSweepSummary.done} 场，失败 {lastSweepSummary.failed} 场
              </span>
            </div>
            <div className="mt-1 text-lg font-black text-amber-500 leading-none">
              +{lastSweepSummary.cityGold.toLocaleString()} 城金
            </div>
            <div className="mt-1 text-[10px] text-[var(--color-text-muted)]">
              详细战斗结果可前往军情查看；下次扫荡后这里会自动更新。
            </div>
          </div>
        )}
      </div>

      {/* City Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
        {cities.map((city, i) => (
          <div
            key={city.id}
            className="animate-fade-in-up"
            style={{ animationDelay: `${i * 50}ms`, animationFillMode: 'both' }}
          >
            <NpcCityCard
              city={city}
              selected={selectedCity?.id === city.id}
              onClick={() => setSelectedCity(selectedCity?.id === city.id ? null : city)}
              onBattleResult={(report) => { setBattleReport(report); loadCities() }}
              onScoutResult={(report) => { setScoutReport(report); loadCities() }}
            />
          </div>
        ))}
      </div>

      {/* Attack Panel */}
      {selectedCity && (
        <AttackPanel
          city={selectedCity}
          onClose={() => setSelectedCity(null)}
          onComplete={handleAttackComplete}
        />
      )}

      {/* Battle Result from quick actions */}
      {battleReport && (
        <BattleResultModal report={battleReport} onClose={() => setBattleReport(null)} />
      )}

      {/* Scout Result from quick actions */}
      {scoutReport && (
        <ScoutResultModal report={scoutReport} onClose={() => setScoutReport(null)} />
      )}
    </div>
  )
}

export default NpcCityTab
