// 本文件实现地图页 NPC 城池列表、手动攻击和一键扫荡交互。
import { useState, useEffect, useCallback, type FC } from 'react'
import { RefreshCw, LoaderCircle, Swords } from 'lucide-react'
import { useGameStore } from '@/store/gameStore'
import { gameApi } from '@/api/game'
import { ApiError } from '@/api/client'
import type { NpcCity, BattleReport, NpcSweepTask } from '@/types/game'
import { toast } from '@/components/ui'
import NpcCityCard from './NpcCityCard'
import AttackPanel from './AttackPanel'
import BattleResultModal from './BattleResultModal'
import ScoutResultModal from './ScoutResultModal'
import { getNpcQuickBattleGeneralIds } from './npcQuickGeneral'

type NpcTier = NpcCity['tier']

const TIER_ORDER: NpcTier[] = ['small', 'medium', 'large', 'golden']
const TIER_LABELS: Record<NpcTier, string> = {
  small: '小型',
  medium: '中型',
  large: '大型',
  golden: '金色',
}
const SWEEP_TASK_POLL_MS = 800
const SWEEP_TASK_POLL_TIMEOUT_MS = 120_000
const MAX_SWEEP_TASK_TARGETS = 50

// NpcCityTab 渲染 NPC 城池页签，并维护一键扫荡的本地状态。
const NpcCityTab: FC = () => {
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const [cities, setCities] = useState<NpcCity[]>([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [sweeping, setSweeping] = useState(false)
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

  const loadCities = useCallback(async () => {
    if (!activePlayerId) return
    try {
      const data = await gameApi.getNpcCities(activePlayerId)
      setCities(data.cities ?? [])
    } finally {
      setLoading(false)
    }
  }, [activePlayerId])

  useEffect(() => {
    loadCities()
  }, [loadCities])

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

  const allSweepTargets = cities
    .filter((city) => selectedTiers[city.tier])
    .sort((a, b) => TIER_ORDER.indexOf(a.tier) - TIER_ORDER.indexOf(b.tier))
  const selectedSweepTargets = allSweepTargets

  // handleSweepTierToggle 切换参与一键扫荡的 NPC 城池等级。
  const handleSweepTierToggle = (tier: NpcTier) => {
    if (sweeping) return
    setSelectedTiers((prev) => ({ ...prev, [tier]: !prev[tier] }))
  }

  // handleSweep 调用后端批量扫荡接口，完成后同步局部玩家状态和聚合战报。
  const handleSweep = async () => {
    if (!activePlayerId || sweeping || selectedSweepTargets.length === 0) return
    if (selectedSweepTargets.length > MAX_SWEEP_TASK_TARGETS) {
      toast.error(`单次最多扫荡 ${MAX_SWEEP_TASK_TARGETS} 座城，请减少勾选等级。`)
      return
    }

    setSelectedCity(null)
    setSweeping(true)
    setLastSweepSummary(null)

    try {
      const npcIds = selectedSweepTargets.map((city) => city.id)
      const generalIds = getNpcQuickBattleGeneralIds(useGameStore.getState().state)
      const task = await gameApi.sweepNpc(activePlayerId, npcIds, 'attack', generalIds)
      const completedTask = await pollSweepTask(activePlayerId, task)
      if (completedTask.status === 'failed' || !completedTask.result) {
        toast.error(completedTask.error || '扫荡失败，请稍后重试')
        return
      }
      const result = completedTask.result
      useGameStore.getState().patchState({
        resources: result.resources,
        resourceProduction: result.resourceProduction,
        resourceSettledAt: result.resourceSettledAt,
        generalTraitProgress: result.generalTraitProgress,
        army: result.army,
        general: result.general,
        generals: result.generals,
        cityGold: result.cityGold,
        npcState: result.npcState,
        serverTime: result.serverTime,
      })
      if (result.npcState?.cities) {
        setCities(result.npcState.cities)
      }
      const totalCityGold = result.battleReport?.overflowCityGold ?? 0
      if (result.battleReport?.id) {
        setBattleReport(result.battleReport)
      }
      setLastSweepSummary({ done: result.done, failed: result.failed, cityGold: totalCityGold })
      await loadCities()
      void useGameStore.getState().loadMilitaryView()
      void useGameStore.getState().loadResourceView()
      if (result.stopped) {
        toast.info(`扫荡已停止：成功 ${result.done} 场，失败 ${result.failed} 场，获得 ${totalCityGold.toLocaleString()} 城金。`)
      } else {
        toast.success(`扫荡完成：成功 ${result.done} 场，失败 ${result.failed} 场，获得 ${totalCityGold.toLocaleString()} 城金。可前往军情查看战报。`)
      }
    } catch (error) {
      if (!(error instanceof ApiError)) {
        toast.error(error instanceof Error ? error.message : '扫荡失败，请稍后重试')
      }
    } finally {
      setSweeping(false)
    }
  }

  const handleAttackComplete = () => {
    setSelectedCity(null)
    loadCities()
    void useGameStore.getState().loadMilitaryView()
    void useGameStore.getState().loadResourceView()
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
        一键操作将派出全部兵力，并自动携带 1 名可用武将。城池等级：<span className="text-slate-500">小型</span> &lt; <span className="text-blue-500">中型</span> &lt; <span className="text-purple-500">大型</span> &lt; <span className="text-amber-500">金色</span>，等级越高资源越多、守军越强。每24小时自动刷新。
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
              按小型到金色顺序逐个发起一键攻击，后台自动结算，单次最多 {MAX_SWEEP_TASK_TARGETS} 城。
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

        {selectedSweepTargets.length > MAX_SWEEP_TASK_TARGETS && (
          <div className="rounded-xl border border-red-400/40 bg-red-500/10 px-3 py-2 text-[10px] text-red-500">
            当前已选择 {selectedSweepTargets.length} 座城，超过单次上限 {MAX_SWEEP_TASK_TARGETS} 座。
          </div>
        )}

        {sweeping && (
          <div className="flex items-center gap-2 rounded-xl border border-red-400/30 bg-red-500/10 px-3 py-2 text-[10px] text-[var(--color-text-muted)]">
            <LoaderCircle size={12} className="animate-spin text-red-500" />
            <span>后台批量结算中，完成后会自动展示结果。</span>
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

// pollSweepTask 轮询后台扫荡任务，直到任务完成或失败。
const pollSweepTask = async (playerId: string, initialTask: NpcSweepTask) => {
  let current = initialTask
  const startedAt = Date.now()
  while (current.status === 'queued' || current.status === 'running') {
    if (Date.now() - startedAt >= SWEEP_TASK_POLL_TIMEOUT_MS) {
      throw new Error('扫荡任务等待超时，请刷新页面后查看任务结果。')
    }
    await new Promise((resolve) => window.setTimeout(resolve, SWEEP_TASK_POLL_MS))
    current = await gameApi.getNpcSweepTask(playerId, current.id)
  }
  return current
}

export default NpcCityTab
