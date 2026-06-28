/* 本文件实现地图副本页，并接入轮回绝境副本玩法。 */
import { type FC, useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Crown, Flame, Lock, ScrollText, ShieldAlert, Swords, Timer, Trophy } from 'lucide-react'
import { gameApi } from '@/api/game'
import { toast } from '@/components/ui'
import { useConfigStore } from '@/store/configStore'
import { useGameStore } from '@/store/gameStore'
import type { ReincarnationRun, ReincarnationWave, Reward } from '@/types/game'
import reincarnationAbyssBg from '@/assets/dungeons/reincarnation-abyss.webp'
import kingsWarBg from '@/assets/dungeons/kings-war.webp'
import famousGeneralsBg from '@/assets/dungeons/famous-generals.webp'

interface DungeonEntry {
  id: string
  title: string
  subtitle: string
  icon: typeof Swords
  limited?: boolean
  backgroundImage?: string
}

const LIMITED_DUNGEONS: DungeonEntry[] = [
  { id: 'god-demon-battlefield', title: '神魔战场', subtitle: '限时副本，神魔裂土', icon: ShieldAlert, limited: true },
  { id: 'ancient-heaven', title: '远古天庭', subtitle: '限时副本，金阙重开', icon: ScrollText, limited: true },
]

const OTHER_DUNGEONS: DungeonEntry[] = [
  { id: 'kings-war', title: '万王争霸', subtitle: '诸王并起，问鼎天下', icon: Crown, backgroundImage: kingsWarBg },
  { id: 'famous-generals', title: '天下名将', subtitle: '群雄列阵，名将归心', icon: Swords, backgroundImage: famousGeneralsBg },
]

// DungeonTab 渲染副本页与轮回绝境操作面板。
const DungeonTab: FC = () => (
  <div className="space-y-5">
    <ReincarnationAbyssPanel />

    <div className="space-y-4">
      {OTHER_DUNGEONS.map((dungeon) => (
        <DungeonRow key={dungeon.id} dungeon={dungeon} />
      ))}
    </div>

    <div className="pt-2">
      <div className="mb-3 flex items-center gap-2">
        <span className="h-px flex-1 bg-amber-500/25" />
        <span className="rounded-full border border-amber-400/40 bg-amber-400/10 px-3 py-1 text-xs font-bold text-amber-600">限时副本</span>
        <span className="h-px flex-1 bg-amber-500/25" />
      </div>
      <div className="space-y-4">
        {LIMITED_DUNGEONS.map((dungeon) => (
          <DungeonRow key={dungeon.id} dungeon={dungeon} />
        ))}
      </div>
    </div>
  </div>
)

// ReincarnationAbyssPanel 渲染轮回绝境的层级、波次、奖励和出兵操作。
const ReincarnationAbyssPanel: FC = () => {
  const navigate = useNavigate()
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const state = useGameStore((s) => s.state)
  const patchState = useGameStore((s) => s.patchState)
  const reincarnation = useConfigStore((s) => s.reincarnation)
  const items = useConfigStore((s) => s.items)
  const units = useConfigStore((s) => s.units)
  const factions = useConfigStore((s) => s.factions)
  const [run, setRun] = useState<ReincarnationRun | null>(null)
  const [selectedLevel, setSelectedLevel] = useState(1)
  const [troops, setTroops] = useState<Record<string, number>>({})
  const [loading, setLoading] = useState(false)
  const [defenseCountdown, setDefenseCountdown] = useState(0)
  const [lastReportId, setLastReportId] = useState('')

  const army = state?.army ?? []
  const currentWave = run?.waves.find((wave) => wave.waveIndex === run.currentWave)
  const availableLevels = reincarnation?.levels.filter((level) => level.enabled) ?? []
  const playerFaction = state?.player.faction ?? ''
  const unitConfig = units?.[playerFaction] ?? {}
  const remainingSeconds = run ? Math.max(0, Math.floor((new Date(run.expiresAt).getTime() - Date.now()) / 1000)) : 0

  useEffect(() => {
    if (!activePlayerId) return
    void loadRun()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activePlayerId])

  useEffect(() => {
    if (availableLevels[0] && selectedLevel === 1) {
      setSelectedLevel(availableLevels[0].level)
    }
  }, [availableLevels, selectedLevel])

  useEffect(() => {
    if (defenseCountdown <= 0) return
    const timer = window.setTimeout(() => setDefenseCountdown((value) => value - 1), 1000)
    return () => window.clearTimeout(timer)
  }, [defenseCountdown])

  useEffect(() => {
    if (defenseCountdown === 0 && currentWave?.waveType === 'defense' && loading) {
      void submitWaveAction()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [defenseCountdown])

  // loadRun 加载玩家当前轮回绝境实例。
  const loadRun = async () => {
    if (!activePlayerId) return
    try {
      const result = await gameApi.getReincarnationRun(activePlayerId)
      setRun(result.run ?? null)
      if (result.army) patchState({ army: result.army, serverTime: result.serverTime })
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '轮回绝境加载失败')
    }
  }

  // startRun 开启所选层级轮回。
  const startRun = async () => {
    if (!activePlayerId) return
    const confirmed = window.confirm('轮回绝境会真实消耗兵力，阵亡兵力不会返还，确认开启？')
    if (!confirmed) return
    setLoading(true)
    try {
      const result = await gameApi.startReincarnation(activePlayerId, selectedLevel)
      setRun(result.run)
      if (result.army) patchState({ army: result.army, serverTime: result.serverTime })
      toast.success('轮回绝境已开启')
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '开启失败')
    } finally {
      setLoading(false)
    }
  }

  // submitWaveAction 提交当前波次的进攻或防守兵力。
  const submitWaveAction = async () => {
    if (!activePlayerId || !currentWave) return
    const selectedTotal = Object.values(troops).reduce((sum, value) => sum + value, 0)
    if (selectedTotal <= 0) {
      toast.error('请选择出兵数量')
      setLoading(false)
      return
    }
    if (selectedTotal > currentWave.troopCap) {
      toast.error('出兵超过本波上限')
      setLoading(false)
      return
    }
    setLoading(true)
    try {
      const clientActionId = `${currentWave.id}_${Date.now()}_${Math.random().toString(16).slice(2)}`
      const result = currentWave.waveType === 'defense'
        ? await gameApi.readyReincarnationDefense(activePlayerId, currentWave.id, troops, clientActionId)
        : await gameApi.attackReincarnationWave(activePlayerId, currentWave.id, troops, clientActionId)
      setRun(result.run)
      setTroops({})
      setLastReportId(result.battleReport?.id ?? '')
      patchState({
        army: result.army,
        inventory: result.inventory,
        inventorySlots: result.inventorySlots,
        serverTime: result.serverTime,
      })
      toast.success(currentWave.waveType === 'defense' ? '防守结算完成' : '进攻结算完成')
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '结算失败')
    } finally {
      setLoading(false)
    }
  }

  // handleDefenseReady 先展示 3 秒倒计时，再提交防守结算。
  const handleDefenseReady = () => {
    if (loading || defenseCountdown > 0) return
    setLoading(true)
    setDefenseCountdown(reincarnation?.defenseCountdownSeconds ?? 3)
  }

  // settleRun 主动结算已结束或超时的轮回。
  const settleRun = async () => {
    if (!activePlayerId) return
    setLoading(true)
    try {
      const result = await gameApi.settleReincarnation(activePlayerId)
      setRun(result.run)
      patchState({
        army: result.army,
        inventory: result.inventory,
        inventorySlots: result.inventorySlots,
        serverTime: result.serverTime,
      })
      toast.success('轮回奖励已结算')
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '结算失败')
    } finally {
      setLoading(false)
    }
  }

  const rewardText = (rewards: Reward[]) => rewards.length === 0
    ? '暂无奖励'
    : rewards.map((reward) => `${items?.[reward.id]?.name ?? reward.id} x${reward.amount.toLocaleString()}`).join('、')

  const waveProgress = run ? Math.round(((run.currentWave - 1) / 18) * 100) : 0

  return (
    <section className="overflow-hidden rounded-2xl border border-violet-500/30 bg-[var(--color-surface)] shadow-[0_18px_44px_rgba(88,28,135,0.12)]">
      <div
        className="relative min-h-[210px] p-5 sm:p-6"
        style={{
          backgroundImage: `linear-gradient(90deg,rgba(2,6,23,0.9),rgba(2,6,23,0.5),rgba(2,6,23,0.12)), url(${reincarnationAbyssBg})`,
          backgroundPosition: 'center',
          backgroundSize: 'cover',
        }}
      >
        <div className="relative flex flex-col gap-5 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <p className="mb-2 text-xs font-bold tracking-[0.22em] text-violet-300">DUNGEON</p>
            <h2 className='text-[clamp(2.3rem,5vw,4.4rem)] font-black leading-none text-white [font-family:"STKaiti","KaiTi","Songti_SC","SimSun",serif]'>
              轮回绝境
            </h2>
            <p className="mt-3 max-w-xl text-sm font-medium text-violet-100/85">18 波攻防轮转，真实兵力损耗，通关或结束后统一结算累计奖励。</p>
          </div>
          <div className="flex flex-wrap gap-2">
            {run ? (
              <>
                <StatusPill icon={Timer} text={run.status === 'running' ? `剩余 ${formatDuration(remainingSeconds)}` : statusLabel(run.status)} />
                <StatusPill icon={Trophy} text={`${run.levelName} · 第 ${run.currentWave}/18 波`} />
              </>
            ) : (
              <StatusPill icon={Flame} text="可开启" />
            )}
          </div>
        </div>
      </div>

      <div className="p-4 sm:p-5">
        {!run ? (
          <div className="grid gap-3 md:grid-cols-[1fr_auto] md:items-end">
            <div>
              <label className="mb-2 block text-xs font-bold text-[var(--color-text-primary)]">选择轮回层级</label>
              <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
                {availableLevels.map((level) => (
                  <button
                    key={level.level}
                    type="button"
                    onClick={() => setSelectedLevel(level.level)}
                    className={`rounded-xl border px-3 py-2 text-left transition ${selectedLevel === level.level ? 'border-violet-500/60 bg-violet-500/12' : 'border-[var(--color-border)] bg-[var(--color-surface-dim)] hover:border-violet-500/30'}`}
                  >
                    <span className="block text-xs font-bold text-[var(--color-text-primary)]">{level.name}</span>
                    <span className="mt-1 block text-[10px] text-[var(--color-text-muted)]">规模 {compactNumber(level.enemyTroopBase)}</span>
                  </button>
                ))}
              </div>
            </div>
            <button
              type="button"
              onClick={() => void startRun()}
              disabled={loading || !activePlayerId}
              className="h-10 rounded-xl bg-violet-600 px-5 text-sm font-bold text-white transition hover:brightness-105 disabled:cursor-not-allowed disabled:opacity-50"
            >
              开启轮回
            </button>
          </div>
        ) : (
          <div className="space-y-4">
            <div className="h-2 overflow-hidden rounded-full bg-black/10 dark:bg-white/10">
              <div className="h-full rounded-full bg-violet-500 transition-all" style={{ width: `${waveProgress}%` }} />
            </div>

            {currentWave ? (
              <WavePanel
                wave={currentWave}
                army={army}
                troops={troops}
                setTroops={setTroops}
                unitConfig={unitConfig}
                factionName={factions?.[currentWave.enemyFaction]?.name ?? currentWave.enemyFaction}
                rewardText={rewardText}
              />
            ) : (
              <p className="text-sm text-[var(--color-text-muted)]">当前轮回已结束，可结算累计奖励。</p>
            )}

            <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2">
              <div className="min-w-0">
                <p className="text-xs font-bold text-[var(--color-text-primary)]">累计奖励</p>
                <p className="mt-1 truncate text-[11px] text-[var(--color-text-secondary)]">{rewardText(run.pendingRewards)}</p>
              </div>
              <div className="flex gap-2">
                {lastReportId && (
                  <button
                    type="button"
                    onClick={() => navigate(`/report/${lastReportId}?from=dungeon`)}
                    className="h-9 rounded-xl border border-[var(--color-border)] px-3 text-xs font-bold text-[var(--color-text-secondary)]"
                  >
                    战报 {lastReportId.slice(-6)}
                  </button>
                )}
                {run.status === 'running' && currentWave?.waveType === 'attack' && (
                  <button type="button" onClick={() => void submitWaveAction()} disabled={loading} className="h-9 rounded-xl bg-violet-600 px-4 text-xs font-bold text-white disabled:opacity-50">
                    进攻
                  </button>
                )}
                {run.status === 'running' && currentWave?.waveType === 'defense' && (
                  <button type="button" onClick={handleDefenseReady} disabled={loading && defenseCountdown === 0} className="h-9 rounded-xl bg-amber-600 px-4 text-xs font-bold text-white disabled:opacity-50">
                    {defenseCountdown > 0 ? `${defenseCountdown}s` : '防御就绪'}
                  </button>
                )}
                {run.status !== 'running' && !run.rewardGrantedAt && (
                  <button type="button" onClick={() => void settleRun()} disabled={loading} className="h-9 rounded-xl bg-green-600 px-4 text-xs font-bold text-white disabled:opacity-50">
                    结算奖励
                  </button>
                )}
              </div>
            </div>
          </div>
        )}
      </div>
    </section>
  )
}

// WavePanel 展示当前波次和出兵输入。
const WavePanel: FC<{
  wave: ReincarnationWave
  army: Array<{ unitType: string; amount: number }>
  troops: Record<string, number>
  setTroops: (value: Record<string, number>) => void
  unitConfig: Record<string, { name: string }>
  factionName: string
  rewardText: (rewards: Reward[]) => string
}> = ({ wave, army, troops, setTroops, unitConfig, factionName, rewardText }) => {
  const selectedTotal = Object.values(troops).reduce((sum, value) => sum + value, 0)
  const enemyTotal = totalMapAmount(wave.enemyTroops)
  const enemyRemaining = totalMapAmount(wave.enemyRemaining)
  const enemyProgress = enemyTotal > 0 ? Math.max(0, Math.min(100, Math.round((enemyRemaining / enemyTotal) * 100))) : 0
  const toggleUnit = (unitType: string, available: number) => {
    const current = troops[unitType] ?? 0
    const next = { ...troops }
    if (current > 0) {
      delete next[unitType]
      setTroops(next)
      return
    }
    const remainCap = Math.max(0, wave.troopCap - selectedTotal)
    const fill = Math.min(available, remainCap)
    if (fill <= 0) return
    next[unitType] = fill
    setTroops(next)
  }

  return (
    <div className="grid gap-4 lg:grid-cols-[1fr_1.1fr]">
      <div className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-3">
        <div className="mb-3 flex items-center justify-between gap-2">
          <div>
            <p className="text-xs font-bold text-[var(--color-text-primary)]">第 {wave.waveIndex} 波 · {wave.waveType === 'attack' ? '进攻波' : '防守波'}</p>
            <p className="mt-1 text-[11px] text-[var(--color-text-muted)]">敌方阵营：{factionName}</p>
          </div>
          <span className="rounded-lg border border-violet-500/25 bg-violet-500/10 px-2 py-1 text-[10px] font-bold text-violet-500">
            上限 {compactNumber(wave.troopCap)}
          </span>
        </div>
        <div className="grid gap-2 sm:grid-cols-2">
          <InfoLine label="己方加成" value={wave.allyBonus.label} tone="green" />
          <InfoLine label="敌方加成" value={wave.enemyBonus.label} tone="amber" />
        </div>
        <div className="mt-3 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] p-3">
          <div className="mb-2 flex items-center justify-between gap-2 text-[11px] font-bold">
            <span className="text-[var(--color-text-primary)]">敌军态势</span>
            <span className="text-rose-500">剩余压力</span>
          </div>
          <div className="h-2 overflow-hidden rounded-full bg-rose-500/10">
            <div className="h-full rounded-full bg-rose-500 transition-all" style={{ width: `${enemyProgress}%` }} />
          </div>
        </div>
        <p className="mt-3 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-[11px] text-[var(--color-text-secondary)]">
          本波奖励：{rewardText(wave.rewardPreview)}
        </p>
      </div>

      <div className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-3">
        <div className="mb-3 flex items-center justify-between">
          <p className="text-xs font-bold text-[var(--color-text-primary)]">{wave.waveType === 'attack' ? '进攻出兵' : '防守配置'}</p>
          <span className="text-[10px] font-semibold text-[var(--color-text-muted)]">已选 {compactNumber(selectedTotal)}</span>
        </div>
        <div className="space-y-2">
          {army.filter((unit) => unit.amount > 0).map((unit) => (
            <button
              key={unit.unitType}
              type="button"
              onClick={() => toggleUnit(unit.unitType, unit.amount)}
              className={`grid w-full grid-cols-[1fr_auto] items-center gap-2 rounded-lg border px-3 py-2 text-left transition ${
                (troops[unit.unitType] ?? 0) > 0
                  ? 'border-violet-500/55 bg-violet-500/12 shadow-[0_0_0_1px_rgba(139,92,246,0.18)]'
                  : 'border-[var(--color-border)] bg-[var(--color-surface)] hover:border-violet-500/35'
              }`}
            >
              <div className="min-w-0">
                <p className="truncate text-xs font-bold text-[var(--color-text-primary)]">{unitConfig[unit.unitType]?.name ?? unit.unitType}</p>
                <p className="text-[10px] text-[var(--color-text-muted)]">拥有 {unit.amount.toLocaleString()}</p>
              </div>
              <span className={`shrink-0 rounded-md px-2 py-1 text-[10px] font-bold ${(troops[unit.unitType] ?? 0) > 0 ? 'bg-violet-500 text-white' : 'bg-[var(--color-surface-dim)] text-[var(--color-text-muted)]'}`}>
                {(troops[unit.unitType] ?? 0) > 0 ? compactNumber(troops[unit.unitType]) : '点选'}
              </span>
            </button>
          ))}
        </div>
        <p className="mt-3 text-[10px] leading-relaxed text-rose-500">真实损耗：阵亡兵力不会返还，请确认后继续。</p>
      </div>
    </div>
  )
}

const StatusPill: FC<{ icon: typeof Timer; text: string }> = ({ icon: Icon, text }) => (
  <span className="inline-flex items-center gap-1.5 rounded-full border border-white/25 bg-white/12 px-3 py-1.5 text-xs font-bold text-white backdrop-blur">
    <Icon size={13} />
    {text}
  </span>
)

const InfoLine: FC<{ label: string; value: string; tone: 'green' | 'amber' }> = ({ label, value, tone }) => (
  <div className={`rounded-lg border px-3 py-2 ${tone === 'green' ? 'border-green-500/25 bg-green-500/10 text-green-500' : 'border-amber-500/25 bg-amber-500/10 text-amber-500'}`}>
    <span className="block text-[10px] font-semibold opacity-80">{label}</span>
    <span className="mt-0.5 block truncate text-xs font-bold">{value}</span>
  </div>
)

const totalMapAmount = (values: Record<string, number>) => Object.values(values).reduce((sum, value) => sum + Math.max(0, value), 0)

// DungeonRow 渲染未开放副本入口行。
const DungeonRow: FC<{ dungeon: DungeonEntry }> = ({ dungeon }) => {
  const Icon = dungeon.icon
  const isLimited = dungeon.limited
  const hasBackground = Boolean(dungeon.backgroundImage)

  return (
    <article
      data-dungeon-id={dungeon.id}
      className={`
        group relative min-h-[154px] overflow-hidden rounded-2xl border p-5 sm:min-h-[180px] sm:p-6
        ${isLimited
          ? 'border-amber-400/45 bg-[linear-gradient(135deg,rgba(120,53,15,0.16),rgba(245,158,11,0.13),rgba(255,255,255,0.04))] shadow-[0_18px_42px_rgba(180,83,9,0.12)]'
          : 'border-[var(--color-border)] bg-[var(--color-surface)] shadow-[0_12px_34px_rgba(15,23,42,0.06)]'
        }
      `}
    >
      {hasBackground && (
        <div
          className="absolute inset-y-0 left-0 right-0 opacity-80 transition-transform duration-500 group-hover:scale-[1.025]"
          style={{
            backgroundImage: `url(${dungeon.backgroundImage})`,
            backgroundPosition: 'center',
            backgroundSize: 'cover',
            maskImage: 'linear-gradient(90deg, transparent 0%, rgba(0,0,0,0.2) 8%, black 24%, black 72%, rgba(0,0,0,0.32) 84%, transparent 100%)',
            WebkitMaskImage: 'linear-gradient(90deg, transparent 0%, rgba(0,0,0,0.2) 8%, black 24%, black 72%, rgba(0,0,0,0.32) 84%, transparent 100%)',
          }}
        />
      )}
      <div className={`absolute inset-0 ${hasBackground ? 'bg-[linear-gradient(90deg,rgba(2,6,23,0.88),rgba(2,6,23,0.34)_38%,rgba(2,6,23,0.08)_62%,rgba(2,6,23,0.72))]' : 'bg-[linear-gradient(90deg,rgba(2,6,23,0.18),transparent_42%)] opacity-0 group-hover:opacity-100'} transition-opacity duration-300`} />

      <div className="relative flex h-full flex-col justify-between gap-5 sm:flex-row sm:items-center">
        <div className="flex items-center gap-4">
          <div className={`flex h-14 w-14 shrink-0 items-center justify-center rounded-2xl border sm:h-16 sm:w-16 ${isLimited ? 'border-amber-400/50 bg-amber-400/15 text-amber-500' : 'border-[var(--color-border)] bg-[var(--color-surface-dim)] text-[var(--color-accent)]'}`}>
            <Icon size={28} />
          </div>
          <div>
            <p className={`mb-1 text-xs font-bold tracking-[0.22em] ${isLimited ? 'text-amber-600' : 'text-[var(--color-text-muted)]'}`}>
              {isLimited ? 'LIMITED TIME' : 'DUNGEON'}
            </p>
            <h3 className={`text-[clamp(2rem,4vw,3.8rem)] font-black leading-none tracking-normal [font-family:"STKaiti","KaiTi","Songti_SC","SimSun",serif] ${isLimited ? 'text-amber-500 drop-shadow-[0_2px_10px_rgba(245,158,11,0.25)]' : 'text-[var(--color-text-primary)]'}`}>
              {dungeon.title}
            </h3>
            <p className={`mt-3 text-sm font-medium ${isLimited ? 'text-amber-700/80 dark:text-amber-300/80' : 'text-[var(--color-text-secondary)]'}`}>{dungeon.subtitle}</p>
          </div>
        </div>

        <div className={`inline-flex w-fit items-center gap-2 rounded-full border px-4 py-2 text-sm font-bold ${isLimited ? 'border-amber-400/50 bg-amber-400/15 text-amber-600' : 'border-[var(--color-border)] bg-[var(--color-surface-dim)] text-[var(--color-text-muted)]'}`}>
          <Lock size={15} />
          待开放
        </div>
      </div>
    </article>
  )
}

function compactNumber(value: number) {
  if (value >= 100000000) return `${Math.round(value / 100000000)}亿`
  if (value >= 10000) return `${Math.round(value / 10000)}万`
  return value.toLocaleString()
}

function formatDuration(seconds: number) {
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const s = seconds % 60
  if (h > 0) return `${h}时${m}分`
  if (m > 0) return `${m}分${s}秒`
  return `${s}秒`
}

function statusLabel(status: string) {
  const labels: Record<string, string> = {
    completed: '已通关',
    failed: '已失败',
    expired: '已超时',
    rewarded: '已结算',
  }
  return labels[status] ?? status
}

export default DungeonTab
