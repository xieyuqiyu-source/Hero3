import { useCallback, useEffect, useMemo, useState, type FC } from 'react'
import {
  ChevronLeft,
  ChevronRight,
  Loader2,
  Mail,
  MailOpen,
  X,
} from 'lucide-react'
import { gameApi } from '@/api/game'
import { useGameStore } from '@/store/gameStore'
import type { Mail as PlayerMail } from '@/types/game'
import MailDetail from './components/MailDetail'
import { PAGE_SIZE, TYPE_CONFIG, TYPE_OPTIONS, type MailType } from './data'

const MailPage: FC = () => {
  const [activeType, setActiveType] = useState<MailType>('all')
  const [currentPage, setCurrentPage] = useState(1)
  const [selectedMail, setSelectedMail] = useState<PlayerMail | null>(null)
  const [mails, setMails] = useState<PlayerMail[]>([])
  const [totalMails, setTotalMails] = useState(0)
  const [unreadCount, setUnreadCount] = useState(0)
  const [loading, setLoading] = useState(false)
  const [loadError, setLoadError] = useState('')
  const [hasLoaded, setHasLoaded] = useState(false)
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const patchState = useGameStore((s) => s.patchState)

  const filteredMails = useMemo(() => {
    return activeType === 'all' ? mails : mails.filter((mail) => mail.mailType === activeType)
  }, [activeType, mails])

  const totalPages = Math.max(1, Math.ceil(totalMails / PAGE_SIZE))

  const loadMails = useCallback(async (page: number) => {
    if (!activePlayerId) {
      setMails([])
      setTotalMails(0)
      setUnreadCount(0)
      setHasLoaded(true)
      return
    }
    setLoading(true)
    setLoadError('')
    try {
      const result = await gameApi.listMails(activePlayerId, page, PAGE_SIZE)
      setMails(result.mails)
      setTotalMails(result.total)
      setUnreadCount(result.unread)
      patchState({ unreadMailCount: result.unread })
      const nextTotalPages = Math.max(1, Math.ceil(result.total / PAGE_SIZE))
      if (page > nextTotalPages) setCurrentPage(nextTotalPages)
    } catch (error) {
      setMails([])
      setTotalMails(0)
      setLoadError(error instanceof Error ? error.message : '信函加载失败')
    } finally {
      setLoading(false)
      setHasLoaded(true)
    }
  }, [activePlayerId, patchState])

  useEffect(() => {
    void loadMails(currentPage)
  }, [currentPage, loadMails])

  const handleTypeChange = (type: MailType) => {
    setActiveType(type)
    setCurrentPage(1)
  }

  const handleSelectMail = async (mail: PlayerMail) => {
    if (!activePlayerId) return
    setSelectedMail(mail)
    if (!mail.isRead) {
      const nextUnreadCount = Math.max(0, unreadCount - 1)
      setMails((items) => items.map((item) => item.id === mail.id ? { ...item, isRead: true } : item))
      setUnreadCount(nextUnreadCount)
      patchState({ unreadMailCount: nextUnreadCount })
    }
    try {
      const detail = await gameApi.getMail(activePlayerId, mail.id)
      setSelectedMail(detail)
      void loadMails(currentPage)
    } catch {
      setSelectedMail(mail)
    }
  }

  const handleDeleteMail = async (mailId: string) => {
    if (!activePlayerId) return
    await gameApi.deleteMail(activePlayerId, mailId)
    setSelectedMail(null)
    await loadMails(currentPage)
  }

  if (!hasLoaded || (loading && mails.length === 0)) {
    return (
      <div className="flex items-center justify-center py-16">
        <span className="inline-flex items-center gap-2 text-sm text-[var(--color-text-muted)]">
          <Loader2 size={16} className="animate-spin text-[var(--color-accent)]" />
          信函加载中...
        </span>
      </div>
    )
  }

  return (
    <div className="max-w-4xl">
      <section className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] overflow-hidden">
        <div className="flex items-center gap-2 px-4 py-3 border-b border-[var(--color-border)]">
          <Mail size={16} className="text-[var(--color-accent)]" />
          <h1 className="text-sm font-semibold text-[var(--color-text-primary)]">信函</h1>
          {unreadCount > 0 && (
            <span className="ml-auto rounded-full bg-red-500/10 px-2 py-0.5 text-[10px] font-semibold text-red-500">
              {unreadCount} 未读
            </span>
          )}
        </div>

        <div className="px-3 py-3 border-b border-[var(--color-border)]">
          <div className="grid grid-cols-3 gap-2 sm:grid-cols-6 xl:grid-cols-3">
            {TYPE_OPTIONS.map((option) => {
              const active = activeType === option.key
              return (
                <button
                  key={option.key}
                  type="button"
                  onClick={() => handleTypeChange(option.key)}
                  className={`
                    h-8 rounded-xl border px-2 text-[11px] font-medium cursor-pointer transition-colors
                    ${active
                      ? 'border-[var(--color-accent-border)] bg-[var(--color-accent-light)] text-[var(--color-accent)]'
                      : 'border-[var(--color-border)] bg-[var(--color-surface-dim)] text-[var(--color-text-secondary)] hover:border-[var(--color-accent-border)]'
                    }
                  `}
                >
                  {option.label}
                </button>
              )
            })}
          </div>
        </div>

        <div className="space-y-2 p-3">
          {loadError && (
            <div className="rounded-2xl border border-red-200 bg-red-50 px-3 py-3 text-xs text-red-600">
              {loadError}
            </div>
          )}
          {filteredMails.length === 0 && (
            <div className="flex items-center justify-center py-12 text-sm text-[var(--color-text-muted)]">
              暂无信函
            </div>
          )}
          {filteredMails.map((mail, index) => {
            const config = TYPE_CONFIG[mail.mailType]
            const Icon = mail.isRead ? MailOpen : config.icon
            return (
              <button
                key={mail.id}
                type="button"
                onClick={() => void handleSelectMail(mail)}
                className={`
                  w-full rounded-2xl border px-3 py-3 text-left cursor-pointer
                  border-[var(--color-border)] bg-[var(--color-surface-dim)]
                  opacity-0 animate-fade-in-up
                  transition-[border-color,background-color,box-shadow,transform] duration-200
                  hover:-translate-y-0.5 hover:border-[var(--color-accent-border)]
                  hover:shadow-[0_10px_24px_rgba(15,23,42,0.07)]
                `}
                style={{ animationDelay: `${index * 35}ms`, animationFillMode: 'forwards' }}
              >
                <div className="flex items-start gap-2">
                  <span className={`mt-0.5 inline-flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-xl ${config.color}`}>
                    <Icon size={15} />
                  </span>
                  <span className="min-w-0 flex-1">
                    <span className="flex items-center gap-2">
                      <span className="truncate text-sm font-semibold text-[var(--color-text-primary)]">{mail.title}</span>
                      {!mail.isRead && <span className="h-2 w-2 flex-shrink-0 rounded-full bg-red-500" />}
                    </span>
                    <span className="mt-1 flex items-center gap-2 text-[10px] text-[var(--color-text-muted)]">
                      <span>{config.label}</span>
                      <span>{mail.senderName}</span>
                      <span className="ml-auto">{formatMailTime(mail.createdAt)}</span>
                    </span>
                    <span className="mt-2 line-clamp-2 text-xs leading-5 text-[var(--color-text-secondary)]">
                      {mail.content}
                    </span>
                  </span>
                </div>
              </button>
            )
          })}
        </div>

        <div className="flex items-center justify-between gap-3 border-t border-[var(--color-border)] px-3 py-3">
          <button
            type="button"
            onClick={() => setCurrentPage((page) => Math.max(1, page - 1))}
            disabled={currentPage <= 1}
            className="h-9 w-9 inline-flex items-center justify-center rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-secondary)] disabled:opacity-40 disabled:cursor-not-allowed hover:border-[var(--color-accent-border)] cursor-pointer transition-colors"
            aria-label="上一页"
          >
            <ChevronLeft size={16} />
          </button>
          <div className="text-xs text-[var(--color-text-muted)]">
            第 <span className="font-semibold text-[var(--color-text-primary)]">{currentPage}</span> / {totalPages} 页
            <span className="ml-2">共 {totalMails} 封</span>
          </div>
          <button
            type="button"
            onClick={() => setCurrentPage((page) => Math.min(totalPages, page + 1))}
            disabled={currentPage >= totalPages}
            className="h-9 w-9 inline-flex items-center justify-center rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-secondary)] disabled:opacity-40 disabled:cursor-not-allowed hover:border-[var(--color-accent-border)] cursor-pointer transition-colors"
            aria-label="下一页"
          >
            <ChevronRight size={16} />
          </button>
        </div>
      </section>

      {selectedMail && (
        <div className="fixed inset-0 z-[9000] flex items-end justify-center bg-slate-900/25 px-3 py-3 backdrop-blur-[6px] animate-mail-backdrop-in sm:items-center sm:p-4">
          <button
            type="button"
            className="absolute inset-0 cursor-default"
            onClick={() => setSelectedMail(null)}
            aria-label="关闭信函"
          />
          <div className="relative flex max-h-[calc(100dvh-24px)] w-full max-w-2xl flex-col overflow-hidden rounded-3xl border border-[var(--color-border)] bg-[var(--color-surface)] shadow-[0_24px_70px_rgba(15,23,42,0.22)] animate-mail-dialog-in">
            <button
              type="button"
              onClick={() => setSelectedMail(null)}
              className="absolute right-3 top-3 z-10 flex h-8 w-8 items-center justify-center rounded-full bg-[var(--color-surface)]/90 text-[var(--color-text-muted)] hover:bg-[var(--color-accent-light)] hover:text-[var(--color-text-primary)] cursor-pointer transition-colors"
              aria-label="关闭"
            >
              <X size={16} />
            </button>
            <div className="overflow-y-auto scrollbar-none">
              <MailDetail mail={selectedMail} onDelete={() => void handleDeleteMail(selectedMail.id)} />
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

function formatMailTime(dateStr: string): string {
  const date = new Date(dateStr)
  if (Number.isNaN(date.getTime())) return dateStr
  return date.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

export default MailPage
