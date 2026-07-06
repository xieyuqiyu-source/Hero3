/* 本文件实现 GM 战报排查只读面板。 */
import { useEffect, useMemo, useState } from 'react'
import { Clipboard, Loader2, Search } from 'lucide-react'
import { adminApi } from '@/api/admin'
import type { AdminBattleEvent, AdminBattleReport, AdminBattleReportParticipant } from '@/types'

const PAGE_SIZE = '20'

// formatTime 展示本地化时间。
function formatTime(value?: string): string {
  if (!value) return ''
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN')
}

// compactJSON 将对象压缩为可读排查文本。
function compactJSON(value: unknown): string {
  if (!value || typeof value !== 'object') return ''
  return JSON.stringify(value, null, 2)
}

// BattleEventReportPanel 展示战斗事件、视角战报和参与方。
export default function BattleEventReportPanel() {
  const [playerId, setPlayerId] = useState('')
  const [eventId, setEventId] = useState('')
  const [sourceType, setSourceType] = useState('')
  const [battleType, setBattleType] = useState('')
  const [result, setResult] = useState('')
  const [events, setEvents] = useState<AdminBattleEvent[]>([])
  const [selected, setSelected] = useState<AdminBattleEvent | null>(null)
  const [reports, setReports] = useState<AdminBattleReport[]>([])
  const [participants, setParticipants] = useState<AdminBattleReportParticipant[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const params = useMemo(() => {
    const next: Record<string, string> = { page: '1', pageSize: PAGE_SIZE }
    if (playerId.trim()) next.playerId = playerId.trim()
    if (eventId.trim()) next.eventId = eventId.trim()
    if (sourceType.trim()) next.sourceType = sourceType.trim()
    if (battleType.trim()) next.battleType = battleType.trim()
    if (result.trim()) next.result = result.trim()
    return next
  }, [battleType, eventId, playerId, result, sourceType])

  // loadEvents 按筛选条件加载事件列表。
  const loadEvents = async () => {
    setLoading(true)
    setError('')
    try {
      const page = await adminApi.getBattleEvents(params)
      setEvents(page.items ?? [])
      if ((page.items ?? []).length > 0) {
        await selectEvent(page.items[0])
      } else {
        setSelected(null)
        setReports([])
        setParticipants([])
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '战报排查加载失败')
    } finally {
      setLoading(false)
    }
  }

  // selectEvent 加载某个事件的多视角报告和参与方。
  const selectEvent = async (event: AdminBattleEvent) => {
    setSelected(event)
    const [reportRes, participantRes] = await Promise.all([
      adminApi.getBattleEventReports(event.id),
      adminApi.getBattleEventParticipants(event.id),
    ])
    setReports(reportRes.reports ?? [])
    setParticipants(participantRes.participants ?? [])
  }

  // copyText 复制排查 ID。
  const copyText = async (value: string) => {
    await navigator.clipboard.writeText(value)
  }

  useEffect(() => {
    loadEvents()
  }, [])

  return (
    <div className="grid gap-4">
      <section className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-end">
          {[
            ['玩家 ID', playerId, setPlayerId],
            ['事件 ID', eventId, setEventId],
            ['来源类型', sourceType, setSourceType],
            ['战斗类型', battleType, setBattleType],
            ['结果', result, setResult],
          ].map(([label, value, setter]) => (
            <label key={label as string} className="grid gap-1 text-xs font-semibold text-[var(--color-text-secondary)]">
              {label as string}
              <input
                value={value as string}
                onChange={(event) => (setter as (next: string) => void)(event.target.value)}
                className="h-9 rounded-xl border border-[var(--color-border)] bg-[var(--color-bg)] px-3 text-sm text-[var(--color-text-primary)] outline-none focus:border-[var(--color-accent)]"
              />
            </label>
          ))}
          <button type="button" onClick={loadEvents} disabled={loading} className="inline-flex h-9 items-center justify-center gap-2 rounded-xl bg-[var(--color-accent)] px-4 text-sm font-bold text-white disabled:opacity-60">
            {loading ? <Loader2 size={14} className="animate-spin" /> : <Search size={14} />}
            查询
          </button>
        </div>
        {error && <div className="mt-3 text-sm font-semibold text-red-600">{error}</div>}
      </section>

      <div className="grid gap-4 xl:grid-cols-[360px_1fr]">
        <section className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-3">
          <h2 className="px-1 text-sm font-black text-[var(--color-text-primary)]">战斗事件</h2>
          <div className="mt-3 grid gap-2">
            {events.map((event) => (
              <button
                key={event.id}
                type="button"
                onClick={() => selectEvent(event)}
                className={`rounded-xl border px-3 py-2 text-left transition-colors ${selected?.id === event.id ? 'border-[var(--color-accent)] bg-[var(--color-accent)]/8' : 'border-[var(--color-border)] bg-[var(--color-bg)] hover:border-[var(--color-accent-border)]'}`}
              >
                <div className="truncate text-xs font-bold text-[var(--color-text-primary)]">{event.id}</div>
                <div className="mt-1 text-[11px] text-[var(--color-text-secondary)]">{event.sourceType} · {event.battleType} · {event.result}</div>
                <div className="mt-1 text-[10px] text-[var(--color-text-muted)]">{formatTime(event.occurredAt)}</div>
              </button>
            ))}
            {events.length === 0 && <div className="px-1 py-8 text-center text-sm text-[var(--color-text-muted)]">暂无事件</div>}
          </div>
        </section>

        <section className="grid gap-4">
          {selected && (
            <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4">
              <div className="flex items-center justify-between gap-2">
                <h2 className="text-sm font-black text-[var(--color-text-primary)]">事件摘要</h2>
                <button type="button" onClick={() => copyText(selected.id)} className="inline-flex items-center gap-1 rounded-lg px-2 py-1 text-xs font-bold text-[var(--color-accent)] hover:bg-[var(--color-accent)]/10">
                  <Clipboard size={13} />
                  复制事件 ID
                </button>
              </div>
              <div className="mt-3 grid gap-2 text-xs text-[var(--color-text-secondary)] md:grid-cols-3">
                <span>来源：{selected.sourceType}</span>
                <span>类型：{selected.battleType}</span>
                <span>结果：{selected.result}</span>
                <span>攻击方：{selected.attackerName || selected.attackerPlayerId || '-'}</span>
                <span>防守方：{selected.defenderName || selected.defenderPlayerId || '-'}</span>
                <span>时间：{formatTime(selected.occurredAt)}</span>
              </div>
              {selected.summary && <pre className="mt-3 max-h-48 overflow-auto rounded-xl bg-[var(--color-bg)] p-3 text-[11px] text-[var(--color-text-secondary)]">{compactJSON(selected.summary)}</pre>}
            </div>
          )}

          <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4">
            <h2 className="text-sm font-black text-[var(--color-text-primary)]">玩家视角战报</h2>
            <div className="mt-3 grid gap-2">
              {reports.map((report) => (
                <div key={report.id} className="rounded-xl border border-[var(--color-border)] bg-[var(--color-bg)] p-3">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <div className="min-w-0">
                      <div className="truncate text-xs font-bold text-[var(--color-text-primary)]">{report.title || report.detail?.title || report.id}</div>
                      <div className="mt-1 text-[11px] text-[var(--color-text-secondary)]">{report.playerId} · {report.viewType} · {report.sourceType} · {report.ownerOutcome || report.detail?.ownerOutcome || report.result}</div>
                    </div>
                    <button type="button" onClick={() => copyText(report.id)} className="text-[11px] font-bold text-[var(--color-accent)]">复制报告 ID</button>
                  </div>
                  {report.detail?.extra && <pre className="mt-2 max-h-44 overflow-auto rounded-lg bg-[var(--color-surface)] p-2 text-[10px] text-[var(--color-text-secondary)]">{compactJSON(report.detail.extra)}</pre>}
                </div>
              ))}
              {reports.length === 0 && <div className="py-6 text-center text-sm text-[var(--color-text-muted)]">未选择事件或无战报</div>}
            </div>
          </div>

          <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4">
            <h2 className="text-sm font-black text-[var(--color-text-primary)]">参与方快照</h2>
            <div className="mt-3 grid gap-2 md:grid-cols-2">
              {participants.map((item) => (
                <div key={item.id} className="rounded-xl border border-[var(--color-border)] bg-[var(--color-bg)] p-3 text-xs text-[var(--color-text-secondary)]">
                  <div className="font-bold text-[var(--color-text-primary)]">{item.role} · {item.nickname || item.playerId || item.id}</div>
                  <div className="mt-1">兵力：{Object.values(item.troopsBefore ?? {}).reduce((sum, amount) => sum + amount, 0).toLocaleString()} / 损失 {Object.values(item.troopsLost ?? {}).reduce((sum, amount) => sum + amount, 0).toLocaleString()}</div>
                  {item.extra && <pre className="mt-2 max-h-32 overflow-auto rounded-lg bg-[var(--color-surface)] p-2 text-[10px]">{compactJSON(item.extra)}</pre>}
                </div>
              ))}
              {participants.length === 0 && <div className="py-6 text-sm text-[var(--color-text-muted)]">暂无参与方快照</div>}
            </div>
          </div>
        </section>
      </div>
    </div>
  )
}
