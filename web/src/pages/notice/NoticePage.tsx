/* 玩家公告页面，展示全服公告列表、详情和已读状态。 */

import { useEffect, useMemo, useState, type FC } from 'react'
import { AlertCircle, CalendarClock, Loader2, Megaphone, Pin, RefreshCw } from 'lucide-react'
import { Modal } from '@/components/ui'
import { useAnnouncementStore } from '@/store/announcementStore'
import { useGameStore } from '@/store/gameStore'
import type { Announcement, AnnouncementType } from '@/types/game'

const TYPE_LABELS: Record<AnnouncementType, string> = {
  system: '系统公告',
  maintenance: '维护公告',
  event: '活动公告',
  update: '更新公告',
}

// formatDateTime 格式化公告时间。
function formatDateTime(value?: string) {
  if (!value) return '立即展示'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

const NoticePage: FC = () => {
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const announcements = useAnnouncementStore((s) => s.announcements)
  const unread = useAnnouncementStore((s) => s.unread)
  const loading = useAnnouncementStore((s) => s.loading)
  const error = useAnnouncementStore((s) => s.error)
  const loadedPlayerId = useAnnouncementStore((s) => s.loadedPlayerId)
  const loadAnnouncements = useAnnouncementStore((s) => s.loadAnnouncements)
  const markRead = useAnnouncementStore((s) => s.markRead)
  const [selectedId, setSelectedId] = useState('')
  const [readingId, setReadingId] = useState('')

  const selectedAnnouncement = useMemo(
    () => announcements.find((item) => item.id === selectedId) ?? null,
    [announcements, selectedId],
  )

  useEffect(() => {
    if (activePlayerId && loadedPlayerId !== activePlayerId) {
      void loadAnnouncements(activePlayerId)
    }
  }, [activePlayerId, loadedPlayerId, loadAnnouncements])

  // handleOpenDetail 打开详情并标记已读。
  const handleOpenDetail = async (announcement: Announcement) => {
    setSelectedId(announcement.id)
    if (!activePlayerId || announcement.read) return
    setReadingId(announcement.id)
    try {
      await markRead(activePlayerId, announcement.id)
    } finally {
      setReadingId('')
    }
  }

  // handleRefresh 重新拉取公告列表。
  const handleRefresh = () => {
    if (activePlayerId) void loadAnnouncements(activePlayerId)
  }

  if (loading && announcements.length === 0) {
    return (
      <div className="flex items-center justify-center py-16">
        <span className="inline-flex items-center gap-2 text-sm text-[var(--color-text-muted)]">
          <Loader2 size={16} className="animate-spin text-[var(--color-accent)]" />
          公告加载中...
        </span>
      </div>
    )
  }

  return (
    <div className="w-full min-w-0">
      <section className="w-full min-w-0 overflow-hidden rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)]">
        <div className="flex min-w-0 flex-wrap items-center gap-2 border-b border-[var(--color-border)] px-3 py-3 sm:px-4">
          <Megaphone size={16} className="text-[var(--color-accent)]" />
          <h1 className="shrink-0 text-sm font-semibold text-[var(--color-text-primary)]">公告</h1>
          {unread > 0 && (
            <span className="shrink-0 rounded-full bg-red-500/10 px-2 py-0.5 text-[10px] font-semibold text-red-500">
              {unread} 未读
            </span>
          )}
          <button
            type="button"
            onClick={handleRefresh}
            className="ml-auto inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] text-[var(--color-text-secondary)] hover:border-[var(--color-accent-border)] hover:text-[var(--color-accent)] cursor-pointer transition-colors"
            title="刷新公告"
          >
            <RefreshCw size={14} className={loading ? 'animate-spin' : ''} />
          </button>
        </div>

        {error && (
          <div className="mx-3 mt-3 flex items-start gap-2 rounded-xl border border-red-500/20 bg-red-500/5 px-3 py-2 text-xs text-red-500">
            <AlertCircle size={14} className="mt-0.5 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        {announcements.length === 0 && !error ? (
          <div className="px-4 py-16 text-center text-sm text-[var(--color-text-muted)]">
            暂无公告
          </div>
        ) : (
          <div className="space-y-2 p-3">
            {announcements.map((announcement) => (
              <button
                key={announcement.id}
                type="button"
                onClick={() => void handleOpenDetail(announcement)}
                className="w-full min-w-0 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-3 text-left hover:border-[var(--color-accent-border)] hover:bg-[var(--color-accent-light)] cursor-pointer transition-colors"
              >
                <div className="flex min-w-0 items-start gap-2">
                  <span className={`mt-1 h-2 w-2 shrink-0 rounded-full ${announcement.read ? 'bg-[var(--color-border)]' : 'bg-red-500'}`} />
                  <div className="min-w-0 flex-1">
                    <div className="flex min-w-0 flex-wrap items-center gap-1.5">
                      {announcement.pinned && (
                        <span className="inline-flex shrink-0 items-center gap-1 rounded-full bg-amber-500/10 px-2 py-0.5 text-[10px] font-semibold text-amber-600">
                          <Pin size={10} />
                          置顶
                        </span>
                      )}
                      <span className="shrink-0 rounded-full bg-[var(--color-surface)] px-2 py-0.5 text-[10px] font-medium text-[var(--color-text-secondary)]">
                        {TYPE_LABELS[announcement.type] ?? '系统公告'}
                      </span>
                      <h2 className="min-w-[160px] flex-1 truncate text-sm font-semibold text-[var(--color-text-primary)]">
                        {announcement.title}
                      </h2>
                      {readingId === announcement.id && (
                        <Loader2 size={13} className="shrink-0 animate-spin text-[var(--color-accent)]" />
                      )}
                    </div>
                    <div className="mt-2 flex min-w-0 flex-wrap items-center gap-2 text-[11px] text-[var(--color-text-muted)]">
                      <span className="inline-flex items-center gap-1">
                        <CalendarClock size={12} />
                        {formatDateTime(announcement.startsAt || announcement.createdAt)}
                      </span>
                      <span>{announcement.read ? '已读' : '未读'}</span>
                    </div>
                  </div>
                </div>
              </button>
            ))}
          </div>
        )}
      </section>

      <Modal
        open={Boolean(selectedAnnouncement)}
        onClose={() => setSelectedId('')}
        title={selectedAnnouncement?.title ?? '公告详情'}
        width="max-w-2xl"
      >
        {selectedAnnouncement && (
          <div className="min-w-0 space-y-4">
            <div className="flex flex-wrap items-center gap-2 text-[11px] text-[var(--color-text-muted)]">
              <span className="rounded-full bg-[var(--color-surface-dim)] px-2 py-1">
                {TYPE_LABELS[selectedAnnouncement.type] ?? '系统公告'}
              </span>
              {selectedAnnouncement.pinned && (
                <span className="inline-flex items-center gap-1 rounded-full bg-amber-500/10 px-2 py-1 font-semibold text-amber-600">
                  <Pin size={11} />
                  置顶
                </span>
              )}
              <span>{formatDateTime(selectedAnnouncement.startsAt || selectedAnnouncement.createdAt)}</span>
            </div>
            <div className="whitespace-pre-wrap break-words text-sm leading-7 text-[var(--color-text-primary)]">
              {selectedAnnouncement.content}
            </div>
          </div>
        )}
      </Modal>
    </div>
  )
}

export default NoticePage
