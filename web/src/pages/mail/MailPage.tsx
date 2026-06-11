import { useCallback, useEffect, useState, type FC } from 'react'
import {
  ChevronLeft,
  ChevronRight,
  Copy,
  Loader2,
  Mail,
  MailOpen,
  Send,
  X,
} from 'lucide-react'
import { gameApi } from '@/api/game'
import { toast } from '@/components/ui'
import { useAccountStore } from '@/store/accountStore'
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
  const [claiming, setClaiming] = useState(false)
  const [sendOpen, setSendOpen] = useState(false)
  const [sendRecipient, setSendRecipient] = useState('')
  const [sendTitle, setSendTitle] = useState('')
  const [sendContent, setSendContent] = useState('')
  const [sending, setSending] = useState(false)
  const [sendMessage, setSendMessage] = useState('')
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const gameState = useGameStore((s) => s.state)
  const patchState = useGameStore((s) => s.patchState)
  const account = useAccountStore((s) => s.account)

  const myMailAddress = gameState?.player.mailCode
    ? `${gameState.player.nickname}#${gameState.player.mailCode}`
    : ''

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
      const result = await gameApi.listMails(activePlayerId, page, PAGE_SIZE, activeType)
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
  }, [activePlayerId, activeType, patchState])

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
      setMails((items) => items.map((item) => item.id === detail.id ? detail : item))
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

  const handleClaimMail = async (mailId: string) => {
    if (!activePlayerId || claiming) return
    setClaiming(true)
    try {
      const result = await gameApi.claimMailAttachments(activePlayerId, mailId)
      setSelectedMail(result.mail)
      setMails((items) => items.map((item) => item.id === mailId ? result.mail : item))
      patchState({ resources: result.resources, cityGold: result.cityGold })
      if (account && result.accountGold !== undefined) {
        useAccountStore.setState({ account: { ...account, gold: result.accountGold } })
      }
    } finally {
      setClaiming(false)
    }
  }

  const handleCopyAddress = async () => {
    if (!myMailAddress) return
    try {
      if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(myMailAddress)
      } else {
        const input = document.createElement('textarea')
        input.value = myMailAddress
        input.style.position = 'fixed'
        input.style.opacity = '0'
        document.body.appendChild(input)
        input.select()
        document.execCommand('copy')
        document.body.removeChild(input)
      }
      toast.success('收信地址已复制')
    } catch {
      toast.error('复制失败，请手动复制')
    }
  }

  const handleSendPlayerMail = async () => {
    if (!activePlayerId || sending) return
    if (!sendRecipient.trim() || !sendTitle.trim() || !sendContent.trim()) {
      setSendMessage('请填写收信地址、标题和正文')
      return
    }
    setSending(true)
    setSendMessage('')
    try {
      await gameApi.sendPlayerMail(activePlayerId, sendRecipient.trim(), sendTitle.trim(), sendContent.trim())
      toast.success('信函已发送')
      setSendRecipient('')
      setSendTitle('')
      setSendContent('')
      setSendOpen(false)
      void loadMails(currentPage)
    } catch (error) {
      setSendMessage(error instanceof Error ? error.message : '发送失败')
    } finally {
      setSending(false)
    }
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
    <div className="w-full min-w-0">
      <section className="w-full min-w-0 overflow-hidden rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)]">
        <div className="flex min-w-0 flex-wrap items-center gap-2 border-b border-[var(--color-border)] px-3 py-3 sm:px-4">
          <Mail size={16} className="text-[var(--color-accent)]" />
          <h1 className="shrink-0 text-sm font-semibold text-[var(--color-text-primary)]">信函</h1>
          {myMailAddress && (
            <button
              type="button"
              onClick={() => void handleCopyAddress()}
              className="inline-flex min-w-0 max-w-full items-center gap-1 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-2 py-1 text-[10px] text-[var(--color-text-muted)] hover:border-[var(--color-accent-border)] cursor-pointer transition-colors"
              title="复制收信地址"
            >
              <Copy size={11} className="shrink-0" />
              <span className="min-w-0 truncate">{myMailAddress}</span>
            </button>
          )}
          <button
            type="button"
            onClick={() => setSendOpen(true)}
            className="inline-flex shrink-0 items-center gap-1 rounded-lg bg-[var(--color-accent)] px-2.5 py-1 text-[10px] font-semibold text-white hover:opacity-90 cursor-pointer transition-opacity"
          >
            <Send size={11} />
            写信
          </button>
          {unreadCount > 0 && (
            <span className="ml-auto shrink-0 rounded-full bg-red-500/10 px-2 py-0.5 text-[10px] font-semibold text-red-500">
              {unreadCount} 未读
            </span>
          )}
        </div>

        <div className="px-3 py-3 border-b border-[var(--color-border)]">
          <div className="flex flex-wrap gap-2">
            {TYPE_OPTIONS.map((option) => {
              const active = activeType === option.key
              return (
                <button
                  key={option.key}
                  type="button"
                  onClick={() => handleTypeChange(option.key)}
                  className={`
                    h-8 rounded-xl border px-3 text-[11px] font-medium cursor-pointer transition-colors whitespace-nowrap
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
          {mails.length === 0 && (
            <div className="flex items-center justify-center py-12 text-sm text-[var(--color-text-muted)]">
              暂无信函
            </div>
          )}
          {mails.map((mail, index) => {
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
                    <span className="flex min-w-0 items-center gap-2">
                      <span className="truncate text-sm font-semibold text-[var(--color-text-primary)]">{mail.title}</span>
                      {!mail.isRead && <span className="h-2 w-2 flex-shrink-0 rounded-full bg-red-500" />}
                    </span>
                    <span className="mt-1 flex min-w-0 items-center gap-2 text-[10px] text-[var(--color-text-muted)]">
                      <span className="shrink-0">{config.label}</span>
                      <span className="min-w-0 truncate">{mail.senderName}</span>
                      <span className="ml-auto shrink-0">{formatMailTime(mail.createdAt)}</span>
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
          <div className="relative flex max-h-[calc(100dvh-24px)] w-[min(100%,42rem)] min-w-0 flex-col overflow-hidden rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] shadow-[0_24px_70px_rgba(15,23,42,0.22)] animate-mail-dialog-in sm:rounded-3xl">
            <button
              type="button"
              onClick={() => setSelectedMail(null)}
              className="absolute right-3 top-3 z-10 flex h-8 w-8 items-center justify-center rounded-full bg-[var(--color-surface)]/90 text-[var(--color-text-muted)] hover:bg-[var(--color-accent-light)] hover:text-[var(--color-text-primary)] cursor-pointer transition-colors"
              aria-label="关闭"
            >
              <X size={16} />
            </button>
            <div className="min-w-0 overflow-y-auto scrollbar-none">
              <MailDetail
                mail={selectedMail}
                claiming={claiming}
                onClaim={() => void handleClaimMail(selectedMail.id)}
                onDelete={() => void handleDeleteMail(selectedMail.id)}
              />
            </div>
          </div>
        </div>
      )}

      {sendOpen && (
        <div className="fixed inset-0 z-[9000] flex items-end justify-center bg-slate-900/25 px-3 py-3 backdrop-blur-[6px] animate-mail-backdrop-in sm:items-center sm:p-4">
          <button
            type="button"
            className="absolute inset-0 cursor-default"
            onClick={() => setSendOpen(false)}
            aria-label="关闭写信"
          />
          <div className="relative w-[min(100%,32rem)] min-w-0 rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-3 shadow-[0_24px_70px_rgba(15,23,42,0.22)] animate-mail-dialog-in sm:rounded-3xl sm:p-4">
            <button
              type="button"
              onClick={() => setSendOpen(false)}
              className="absolute right-3 top-3 flex h-8 w-8 items-center justify-center rounded-full bg-[var(--color-surface-dim)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] cursor-pointer transition-colors"
              aria-label="关闭"
            >
              <X size={16} />
            </button>
            <div className="pr-8">
              <h2 className="text-sm font-bold text-[var(--color-text-primary)]">写信</h2>
              {myMailAddress && (
                <p className="mt-1 min-w-0 truncate text-[10px] text-[var(--color-text-muted)]">我的收信地址：{myMailAddress}</p>
              )}
            </div>
            <div className="mt-4 space-y-3">
              <input
                value={sendRecipient}
                onChange={(event) => setSendRecipient(event.target.value)}
                placeholder="收信地址，例如 玄术#482193"
                className="w-full rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-sm outline-none focus:border-[var(--color-accent-border)]"
              />
              <input
                value={sendTitle}
                onChange={(event) => setSendTitle(event.target.value)}
                maxLength={60}
                placeholder="标题"
                className="w-full rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-sm outline-none focus:border-[var(--color-accent-border)]"
              />
              <textarea
                value={sendContent}
                onChange={(event) => setSendContent(event.target.value)}
                maxLength={5000}
                rows={6}
                placeholder="正文"
                className="w-full resize-none rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-sm outline-none focus:border-[var(--color-accent-border)]"
              />
              {sendMessage && <p className="text-xs text-[var(--color-text-muted)]">{sendMessage}</p>}
              <div className="flex justify-end gap-2">
                <button
                  type="button"
                  onClick={() => setSendOpen(false)}
                  className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-xs text-[var(--color-text-secondary)] hover:border-[var(--color-accent-border)] cursor-pointer transition-colors"
                >
                  取消
                </button>
                <button
                  type="button"
                  onClick={() => void handleSendPlayerMail()}
                  disabled={sending}
                  className="inline-flex items-center gap-1.5 rounded-xl bg-[var(--color-accent)] px-3 py-2 text-xs font-semibold text-white hover:opacity-90 disabled:opacity-50 cursor-pointer transition-opacity"
                >
                  {sending ? <Loader2 size={13} className="animate-spin" /> : <Send size={13} />}
                  发送
                </button>
              </div>
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
