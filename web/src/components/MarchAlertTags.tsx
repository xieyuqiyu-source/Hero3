// 本组件在左侧菜单展示玩家当前相关的行军倒计时提示。
import { RotateCcw, Zap } from 'lucide-react'
import { useCallback, useEffect, useMemo, useRef, useState, type FC } from 'react'
import { gameApi } from '@/api/game'
import { toast } from '@/components/ui'
import { useGameStore } from '@/store/gameStore'
import type { PvpMarch, Reinforcement } from '@/types/game'

const PVP_RECALL_WINDOW_MS = 120_000
const PVP_MAX_ACCELERATE_TIMES = 2

type MarchTag = '出征' | '返程' | '增援' | '被攻击' | '被侦查'

interface MarchAlertItem {
  id: string
  tag: MarchTag
  playerName: string
  endsAt?: string
  fallbackText?: string
  sortAt: number
  priority: number
  pvpMarch?: PvpMarch
}

interface MarchAlertTagsProps {
  limit?: number
}

const ALERT_PRIORITY: Record<MarchTag, number> = {
  被攻击: 0,
  出征: 1,
  返程: 2,
  被侦查: 3,
  增援: 4,
}

// MarchAlertTags 聚合 PVP 行军和增援行军，显示成简短高亮标签。
const MarchAlertTags: FC<MarchAlertTagsProps> = ({ limit = 5 }) => {
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const [pvpMarches, setPvpMarches] = useState<PvpMarch[]>([])
  const [sentReinforcements, setSentReinforcements] = useState<Reinforcement[]>([])
  const [receivedReinforcements, setReceivedReinforcements] = useState<Reinforcement[]>([])
  const [nowMs, setNowMs] = useState(Date.now())
  const [busyMarchId, setBusyMarchId] = useState<string | null>(null)
  const arrivedRefreshIds = useRef<Set<string>>(new Set())

  // loadMarches 拉取当前玩家可见的行军和增援数据。
  const loadMarches = useCallback(async (cancelled?: () => boolean) => {
    if (!activePlayerId) {
      setPvpMarches([])
      setSentReinforcements([])
      setReceivedReinforcements([])
      return
    }
    try {
      const [pvpResult, sentResult, receivedResult] = await Promise.all([
        gameApi.listPvpMarches(activePlayerId),
        gameApi.listSentReinforcements(activePlayerId),
        gameApi.listReceivedReinforcements(activePlayerId),
      ])
      if (cancelled?.()) return
      setPvpMarches(Array.isArray(pvpResult.items) ? pvpResult.items : [])
      setSentReinforcements(Array.isArray(sentResult.items) ? sentResult.items : [])
      setReceivedReinforcements(Array.isArray(receivedResult.items) ? receivedResult.items : [])
    } catch {
      if (cancelled?.()) return
      setPvpMarches([])
      setSentReinforcements([])
      setReceivedReinforcements([])
    }
  }, [activePlayerId])

  useEffect(() => {
    let cancelled = false

    void loadMarches(() => cancelled)
    const refreshTimer = window.setInterval(() => void loadMarches(() => cancelled), 12_000)
    return () => {
      cancelled = true
      window.clearInterval(refreshTimer)
    }
  }, [loadMarches])

  useEffect(() => {
    const tickTimer = window.setInterval(() => setNowMs(Date.now()), 1000)
    return () => window.clearInterval(tickTimer)
  }, [])

  const items = useMemo(() => {
    if (!activePlayerId) return []
    const next: MarchAlertItem[] = []

    pvpMarches.forEach((march) => {
      const item = buildPvpAlertItem(march, activePlayerId)
      if (item) next.push(item)
    })

    sentReinforcements.forEach((reinforcement) => {
      const item = buildSentReinforcementItem(reinforcement)
      if (item) next.push(item)
    })

    receivedReinforcements.forEach((reinforcement) => {
      const item = buildReceivedReinforcementItem(reinforcement)
      if (item) next.push(item)
    })

    return next
      .sort((a, b) => a.priority - b.priority || a.sortAt - b.sortAt)
      .slice(0, limit)
  }, [activePlayerId, limit, pvpMarches, receivedReinforcements, sentReinforcements])

  useEffect(() => {
    arrivedRefreshIds.current.clear()
  }, [activePlayerId])

  useEffect(() => {
    const arrivedItem = items.find((item) => item.endsAt && toTime(item.endsAt) <= nowMs && !arrivedRefreshIds.current.has(item.id))
    if (!arrivedItem) return
    arrivedRefreshIds.current.add(arrivedItem.id)
    void loadMarches()
    const store = useGameStore.getState()
    void store.loadMilitaryView()
    void store.loadGeneralsView()
  }, [items, loadMarches, nowMs])

  // handleAccelerate 使用城金加速自己的 PVP 出征行军。
  const handleAccelerate = async (march: PvpMarch) => {
    if (!activePlayerId || busyMarchId) return
    setBusyMarchId(march.id)
    try {
      const result = await gameApi.acceleratePvpMarch(activePlayerId, march.id)
      setPvpMarches((prev) => prev.map((item) => item.id === result.march.id ? result.march : item))
      useGameStore.getState().patchState({
        ...(typeof result.cityGold === 'number' ? { cityGold: result.cityGold } : {}),
        serverTime: result.serverTime,
      })
      toast.success(`行军已加速，消耗 ${result.cost ?? 0} 城金。`)
      void loadMarches()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '加速失败')
    } finally {
      setBusyMarchId(null)
    }
  }

  // handleRecall 召回自己的 PVP 出征行军。
  const handleRecall = async (march: PvpMarch) => {
    if (!activePlayerId || busyMarchId) return
    setBusyMarchId(march.id)
    try {
      const result = await gameApi.recallPvpMarch(activePlayerId, march.id)
      setPvpMarches((prev) => prev.map((item) => item.id === result.march.id ? result.march : item))
      useGameStore.getState().patchState({
        ...(result.army ? { army: result.army } : {}),
        ...(result.generals ? { generals: result.generals } : {}),
        serverTime: result.serverTime,
      })
      toast.success('行军已召回，返程完成后兵力会归队。')
      void useGameStore.getState().loadMilitaryView()
      void loadMarches()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '召回失败')
    } finally {
      setBusyMarchId(null)
    }
  }

  if (items.length === 0) return null

  return (
    <div className="mb-2.5 space-y-1">
      {items.map((item) => (
        <div
          key={item.id}
          className={`flex h-8 min-w-0 items-center gap-1.5 rounded-lg border px-2 text-[10px] ${alertClass(item.tag).row}`}
        >
          <span className={`shrink-0 rounded px-1.5 py-0.5 font-black ${alertClass(item.tag).tag}`}>{item.tag}</span>
          <span className="min-w-0 flex-1 truncate font-bold text-[var(--color-text-primary)]">{item.playerName || '未知玩家'}</span>
          <span className={`shrink-0 font-mono font-black ${alertClass(item.tag).time}`}>{item.fallbackText ?? formatCountdown(item.endsAt, nowMs)}</span>
          {item.pvpMarch && canAcceleratePvpMarch(item.pvpMarch) && (
            <button
              type="button"
              onClick={() => void handleAccelerate(item.pvpMarch!)}
              disabled={busyMarchId !== null}
              className="flex h-5 w-5 shrink-0 items-center justify-center rounded bg-amber-500/20 text-amber-700 transition-colors hover:bg-amber-500/35 disabled:opacity-50"
              title="加速"
            >
              <Zap size={11} />
            </button>
          )}
          {item.pvpMarch && canRecallPvpMarch(item.pvpMarch, nowMs) && (
            <button
              type="button"
              onClick={() => void handleRecall(item.pvpMarch!)}
              disabled={busyMarchId !== null}
              className="flex h-5 w-5 shrink-0 items-center justify-center rounded bg-slate-500/15 text-slate-600 transition-colors hover:bg-slate-500/25 disabled:opacity-50"
              title="召回"
            >
              <RotateCcw size={11} />
            </button>
          )}
        </div>
      ))}
    </div>
  )
}

// buildPvpAlertItem 按当前玩家身份生成出征或被攻击提示。
function buildPvpAlertItem(march: PvpMarch, activePlayerId: string): MarchAlertItem | null {
  if (march.attackerPlayerId === activePlayerId) {
    if (march.status !== 'marching' && march.status !== 'returning' && march.status !== 'resolving') return null
    const returning = march.status === 'returning'
    return {
      id: `pvp:${march.id}:attack`,
      tag: returning ? '返程' : '出征',
      playerName: march.defenderName,
      endsAt: returning ? march.returnsAt : march.arrivesAt,
      fallbackText: march.status === 'resolving' ? '结算中' : undefined,
      sortAt: toTime(returning ? march.returnsAt : march.arrivesAt),
      priority: returning ? ALERT_PRIORITY.返程 : ALERT_PRIORITY.出征,
      pvpMarch: march.status === 'marching' ? march : undefined,
    }
  }

  if (march.defenderPlayerId === activePlayerId) {
    if (march.status !== 'marching' && march.status !== 'resolving') return null
    return {
      id: `pvp:${march.id}:defense`,
      tag: '被攻击',
      playerName: march.attackerName,
      endsAt: march.arrivesAt,
      fallbackText: march.status === 'resolving' ? '结算中' : undefined,
      sortAt: toTime(march.arrivesAt),
      priority: ALERT_PRIORITY.被攻击,
    }
  }

  return null
}

// buildSentReinforcementItem 生成我派出增援的倒计时提示。
function buildSentReinforcementItem(reinforcement: Reinforcement): MarchAlertItem | null {
  if (reinforcement.status !== 'marching' && reinforcement.status !== 'returning') return null
  const endsAt = reinforcement.status === 'returning' ? reinforcement.expectedReturnedAt : reinforcement.arriveAt
  return {
    id: `reinforcement:sent:${reinforcement.reinforcementId}`,
    tag: '增援',
    playerName: reinforcement.toPlayerName ?? reinforcement.toPlayerId,
    endsAt,
    sortAt: toTime(endsAt),
    priority: ALERT_PRIORITY.增援,
  }
}

// buildReceivedReinforcementItem 生成别人向我增援的到达倒计时提示。
function buildReceivedReinforcementItem(reinforcement: Reinforcement): MarchAlertItem | null {
  if (reinforcement.status !== 'marching') return null
  return {
    id: `reinforcement:received:${reinforcement.reinforcementId}`,
    tag: '增援',
    playerName: reinforcement.fromPlayerName ?? reinforcement.fromPlayerId,
    endsAt: reinforcement.arriveAt,
    sortAt: toTime(reinforcement.arriveAt),
    priority: ALERT_PRIORITY.增援,
  }
}

// alertClass 返回不同倒计时类型的视觉样式。
function alertClass(tag: MarchTag) {
  if (tag === '被攻击') {
    return {
      row: 'border-red-500/55 bg-red-500/15 shadow-[inset_0_0_14px_rgba(239,68,68,0.12)]',
      tag: 'bg-red-500 text-white',
      time: 'text-red-600',
    }
  }
  if (tag === '出征' || tag === '被侦查') {
    return {
      row: 'border-rose-500/45 bg-rose-500/10 shadow-[inset_0_0_14px_rgba(244,63,94,0.12)]',
      tag: 'bg-rose-500 text-white',
      time: 'text-rose-600',
    }
  }
  return {
    row: 'border-emerald-500/35 bg-emerald-500/10 shadow-[inset_0_0_14px_rgba(16,185,129,0.12)]',
    tag: 'bg-emerald-500 text-white',
    time: 'text-emerald-600',
  }
}

// formatCountdown 将结束时间显示为分秒倒计时。
function formatCountdown(value: string | undefined, nowMs: number) {
  if (!value) return '-'
  const targetMs = new Date(value).getTime()
  if (Number.isNaN(targetMs)) return '-'
  const remaining = Math.max(0, Math.ceil((targetMs - nowMs) / 1000))
  const minutes = Math.floor(remaining / 60)
  const seconds = remaining % 60
  return `${minutes}:${seconds.toString().padStart(2, '0')}`
}

// toTime 把可选时间转成排序值，缺失时排到最后。
function toTime(value?: string) {
  if (!value) return Number.MAX_SAFE_INTEGER
  const time = new Date(value).getTime()
  return Number.isNaN(time) ? Number.MAX_SAFE_INTEGER : time
}

// canRecallPvpMarch 判断出征行军是否还在 2 分钟召回窗口。
function canRecallPvpMarch(march: PvpMarch, nowMs: number) {
  if (march.status !== 'marching') return false
  const startedMs = new Date(march.startedAt).getTime()
  if (Number.isNaN(startedMs)) return false
  return nowMs - startedMs <= PVP_RECALL_WINDOW_MS
}

// canAcceleratePvpMarch 判断出征行军是否还能加速。
function canAcceleratePvpMarch(march: PvpMarch) {
  return march.status === 'marching' && (march.acceleratedTimes ?? 0) < PVP_MAX_ACCELERATE_TIMES
}

export default MarchAlertTags
