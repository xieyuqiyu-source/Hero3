import { useState, useEffect, useRef, type FC } from 'react'
import { Clock, LoaderCircle, ChevronDown, ChevronRight } from 'lucide-react'
import { useGameStore } from '@/store/gameStore'
import { useConfigStore } from '@/store/configStore'
import { toast } from '@/components/ui'
import type { RecruitQueue } from '@/types/game'

const EMPTY_QUEUES: RecruitQueue[] = []
const MAX_QUEUE = 5

function getRemainingSeconds(endsAt: string): number {
  return Math.max(0, Math.ceil((new Date(endsAt).getTime() - Date.now()) / 1000))
}

function formatCountdown(totalSeconds: number): string {
  if (totalSeconds <= 0) return '完成中'
  const h = Math.floor(totalSeconds / 3600)
  const m = Math.floor((totalSeconds % 3600) / 60)
  const s = totalSeconds % 60
  if (h > 0) return `${h}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
  return `${m}:${String(s).padStart(2, '0')}`
}

const RecruitQueuePanel: FC = () => {
  const queues = useGameStore((s) => s.state?.recruitQueues ?? EMPTY_QUEUES)
  const faction = useGameStore((s) => s.state?.player.faction ?? 'wei')
  const units = useConfigStore((s) => s.units)
  const [now, setNow] = useState(Date.now())
  const [expanded, setExpanded] = useState(true)
  const getUnitName = (unitId: string): string => {
    const factionUnits = units?.[faction]
    if (!factionUnits) return unitId
    return factionUnits[unitId]?.name ?? unitId
  }

  const [lastRefreshed, setLastRefreshed] = useState<string | null>(null)
  const prevQueueMapRef = useRef<Map<string, string>>(new Map())

  const pendingQueues = queues

  // 记录上一轮队列的 id → unitType 映射，用于完成通知

  // 检测队列项消失（征兵完成），弹通知
  useEffect(() => {
    const currentIds = new Set(pendingQueues.map((q) => q.id))
    const prevMap = prevQueueMapRef.current

    if (prevMap.size > 0) {
      for (const [prevId, unitType] of prevMap) {
        if (!currentIds.has(prevId)) {
          const name = getUnitName(unitType)
          toast.success(`${name}征兵已完成`)
        }
      }
    }

    prevQueueMapRef.current = new Map(pendingQueues.map((q) => [q.id, q.unitType]))
  }, [pendingQueues])

  useEffect(() => {
    if (pendingQueues.length === 0) return
    const timer = setInterval(() => setNow(Date.now()), 1000)
    return () => clearInterval(timer)
  }, [pendingQueues.length])

  useEffect(() => {
    if (pendingQueues.length === 0) return
    const firstDone = pendingQueues.find((q) => getRemainingSeconds(q.endsAt) <= 0)
    if (firstDone && firstDone.id !== lastRefreshed) {
      setLastRefreshed(firstDone.id)
      useGameStore.getState().loadGameState()
    }
  }, [now, pendingQueues, lastRefreshed])

  // 判断某个队列是否正在训练中：
  // 第一个队列始终在训练；后续队列只有在前一个的 endsAt 已过时才开始训练
  const isTraining = (index: number): boolean => {
    if (index === 0) return true
    const prevEndsAt = new Date(pendingQueues[index - 1].endsAt).getTime()
    return Date.now() >= prevEndsAt
  }

  // 填充到 5 个槽位
  const slots = Array.from({ length: MAX_QUEUE }, (_, i) => pendingQueues[i] ?? null)

  return (
    <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] overflow-hidden">
      <button
        type="button"
        onClick={() => setExpanded(!expanded)}
        className="w-full flex items-center gap-2 px-3 py-2 border-b border-[var(--color-border)] cursor-pointer lg:cursor-default"
      >
        {pendingQueues.length > 0 && <LoaderCircle size={13} className="text-[var(--color-accent)] animate-spin" />}
        <span className="text-xs font-semibold text-[var(--color-text-primary)]">征兵队列</span>
        <span className="text-[10px] text-[var(--color-text-muted)] ml-auto">{pendingQueues.length}/{MAX_QUEUE}</span>
        <span className="lg:hidden">
          {expanded ? <ChevronDown size={12} className="text-[var(--color-text-muted)]" /> : <ChevronRight size={12} className="text-[var(--color-text-muted)]" />}
        </span>
      </button>

      <div className={`
        transition-all duration-200 overflow-hidden
        ${expanded ? 'max-h-[500px] opacity-100' : 'max-sm:max-h-0 max-sm:opacity-0 lg:max-h-[500px] lg:opacity-100'}
      `}>
        {/* Desktop: 5-col grid */}
        <div className="hidden lg:grid grid-cols-5 gap-2 px-3 py-2.5">
          {slots.map((queue, i) => (
            <div
              key={queue?.id ?? `empty-${i}`}
              className={`
                flex items-center justify-between px-2.5 py-2 rounded-xl border
                ${queue
                  ? 'border-[var(--color-accent-border)] bg-[var(--color-accent-light)]'
                  : 'border-dashed border-[var(--color-border)] bg-[var(--color-surface-dim)]'
                }
              `}
            >
              {queue ? (
                <>
                  <span className="text-[10px] font-semibold text-[var(--color-text-primary)]">
                    {getUnitName(queue.unitType)}
                  </span>
                  <span className="text-xs font-bold text-[var(--color-accent)]">{queue.amount}</span>
                  {isTraining(i) ? (
                    <span className="flex items-center gap-0.5 text-[9px] font-mono font-bold text-amber-500">
                      <Clock size={8} />
                      {formatCountdown(getRemainingSeconds(queue.endsAt))}
                    </span>
                  ) : (
                    <span className="text-[9px] font-medium text-[var(--color-text-muted)]">队列中</span>
                  )}
                </>
              ) : (
                <span className="text-[10px] text-[var(--color-text-muted)] w-full text-center">空闲</span>
              )}
            </div>
          ))}
        </div>

        {/* Mobile: single column */}
        <div className="lg:hidden px-3 py-2.5 space-y-1.5">
          {slots.map((queue, i) => (
            <div
              key={queue?.id ?? `empty-${i}`}
              className={`
                flex items-center justify-between px-3 py-2 rounded-xl border
                ${queue
                  ? 'border-[var(--color-accent-border)] bg-[var(--color-accent-light)]'
                  : 'border-dashed border-[var(--color-border)] bg-[var(--color-surface-dim)]'
                }
              `}
            >
              {queue ? (
                <>
                  <span className="text-[11px] font-semibold text-[var(--color-text-primary)]">
                    {getUnitName(queue.unitType)}
                  </span>
                  <span className="text-sm font-bold text-[var(--color-accent)]">{queue.amount}</span>
                  {isTraining(i) ? (
                    <span className="flex items-center gap-0.5 text-[10px] font-mono font-bold text-amber-500">
                      <Clock size={9} />
                      {formatCountdown(getRemainingSeconds(queue.endsAt))}
                    </span>
                  ) : (
                    <span className="text-[10px] font-medium text-[var(--color-text-muted)]">队列中</span>
                  )}
                </>
              ) : (
                <span className="text-[10px] text-[var(--color-text-muted)] w-full text-center">空闲</span>
              )}
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

export default RecruitQueuePanel
