import { useState, useEffect, type FC } from 'react'
import { RefreshCw, LoaderCircle } from 'lucide-react'
import { useGameStore } from '@/store/gameStore'
import { gameApi } from '@/api/game'
import type { NpcCity, BattleReport } from '@/types/game'
import NpcCityCard from './NpcCityCard'
import AttackPanel from './AttackPanel'
import BattleResultModal from './BattleResultModal'

const NpcCityTab: FC = () => {
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const [cities, setCities] = useState<NpcCity[]>([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [selectedCity, setSelectedCity] = useState<NpcCity | null>(null)
  const [battleReport, setBattleReport] = useState<BattleReport | null>(null)

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
    if (!activePlayerId || refreshing) return
    setRefreshing(true)
    try {
      const data = await gameApi.refreshNpcCities(activePlayerId)
      setCities(data.cities ?? [])
      setSelectedCity(null)
    } finally {
      setRefreshing(false)
    }
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
      <div className="flex items-center justify-between">
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
    </div>
  )
}

export default NpcCityTab
