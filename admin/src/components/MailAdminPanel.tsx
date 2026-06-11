import { useState, type FormEvent } from 'react'
import { ChevronLeft, ChevronRight, Mail, Plus, RefreshCw, Send, Trash2 } from 'lucide-react'
import { adminApi } from '@/api/admin'
import type { Mail as MailItem, MailAttachment } from '@/types'

const MAIL_TYPES = [
  { value: 'gm_notice', label: 'GM 通知' },
  { value: 'compensation', label: '补偿' },
  { value: 'reward', label: '奖励' },
  { value: 'event_reward', label: '活动奖励' },
  { value: 'system_notice', label: '系统通知' },
]

const ATTACHMENT_OPTIONS = [
  { type: 'resource', itemId: 'wood', label: '木材' },
  { type: 'resource', itemId: 'stone', label: '石料' },
  { type: 'resource', itemId: 'iron', label: '铁矿' },
  { type: 'resource', itemId: 'food', label: '粮食' },
  { type: 'city_gold', itemId: 'city_gold', label: '城金' },
  { type: 'gold', itemId: 'gold', label: '金币' },
]

export default function MailAdminPanel() {
  const [playerId, setPlayerId] = useState('')
  const [mailType, setMailType] = useState('gm_notice')
  const [title, setTitle] = useState('')
  const [content, setContent] = useState('')
  const [expiresAt, setExpiresAt] = useState('')
  const [attachments, setAttachments] = useState<MailAttachment[]>([])
  const [mails, setMails] = useState<MailItem[]>([])
  const [currentPage, setCurrentPage] = useState(1)
  const [totalMails, setTotalMails] = useState(0)
  const [unreadMails, setUnreadMails] = useState(0)
  const [message, setMessage] = useState('')
  const [loading, setLoading] = useState(false)
  const totalPages = Math.max(1, Math.ceil(totalMails / 10))

  const loadMails = async (page = currentPage, options: { silent?: boolean } = {}) => {
    if (!playerId.trim()) return
    if (!options.silent) setLoading(true)
    setMessage('')
    try {
      const result = await adminApi.getPlayerMails(playerId.trim(), page, 10)
      setMails(result.mails)
      setTotalMails(result.total)
      setUnreadMails(result.unread)
      setCurrentPage(result.page)
      setMessage(`已加载 ${result.total} 封信函，未读 ${result.unread} 封`)
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '加载信函失败')
    } finally {
      if (!options.silent) setLoading(false)
    }
  }

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault()
    if (!playerId.trim() || !title.trim() || !content.trim()) {
      setMessage('请填写 playerId、标题和正文')
      return
    }
    const confirmed = window.confirm(`确认向 ${playerId.trim()} 发送信函「${title.trim()}」？`)
    if (!confirmed) return

    setLoading(true)
    setMessage('')
    try {
      await adminApi.sendMail({
        playerId: playerId.trim(),
        mailType,
        title: title.trim(),
        content: content.trim(),
        attachments: attachments.filter((item) => item.amount > 0),
        expiresAt: expiresAt ? new Date(expiresAt).toISOString() : undefined,
      })
      setTitle('')
      setContent('')
      setExpiresAt('')
      setAttachments([])
      setMessage('信函已发送')
      setCurrentPage(1)
      await loadMails(1, { silent: true })
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '发送失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] overflow-hidden">
      <div className="flex items-center gap-2 px-4 py-3 border-b border-[var(--color-border)]">
        <Mail size={16} className="text-[var(--color-accent)]" />
        <h2 className="text-sm font-semibold text-[var(--color-text-primary)]">信函管理</h2>
        <span className="ml-auto text-[10px] text-[var(--color-text-muted)]">第一版：指定玩家发信</span>
      </div>

      <form onSubmit={handleSubmit} className="grid gap-3 p-4">
        <div className="grid gap-3 md:grid-cols-[1fr_160px]">
          <input
            value={playerId}
            onChange={(event) => setPlayerId(event.target.value)}
            placeholder="玩家 playerId"
            className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-sm outline-none focus:border-[var(--color-accent-border)]"
          />
          <select
            value={mailType}
            onChange={(event) => setMailType(event.target.value)}
            className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-sm outline-none focus:border-[var(--color-accent-border)]"
          >
            {MAIL_TYPES.map((item) => (
              <option key={item.value} value={item.value}>{item.label}</option>
            ))}
          </select>
        </div>
        <input
          value={title}
          onChange={(event) => setTitle(event.target.value)}
          maxLength={60}
          placeholder="标题"
          className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-sm outline-none focus:border-[var(--color-accent-border)]"
        />
        <textarea
          value={content}
          onChange={(event) => setContent(event.target.value)}
          maxLength={5000}
          rows={5}
          placeholder="正文"
          className="resize-none rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-sm outline-none focus:border-[var(--color-accent-border)]"
        />
        <input
          type="datetime-local"
          value={expiresAt}
          onChange={(event) => setExpiresAt(event.target.value)}
          className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-sm outline-none focus:border-[var(--color-accent-border)]"
        />
        <div className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-3">
          <div className="mb-2 flex items-center gap-2">
            <span className="text-xs font-semibold text-[var(--color-text-primary)]">附件</span>
            <button
              type="button"
              onClick={() => setAttachments((items) => [...items, { type: 'resource', itemId: 'wood', amount: 1000 }])}
              className="ml-auto inline-flex items-center gap-1 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 py-1 text-[10px] text-[var(--color-text-secondary)] hover:border-[var(--color-accent-border)] cursor-pointer transition-colors"
            >
              <Plus size={12} />
              添加附件
            </button>
          </div>
          {attachments.length === 0 ? (
            <p className="text-xs text-[var(--color-text-muted)]">无附件。玩家互发信函不支持附件，GM 信函可附带补偿奖励。</p>
          ) : (
            <div className="space-y-2">
              {attachments.map((item, index) => {
                const selected = `${item.type}:${item.itemId}`
                return (
                  <div key={index} className="grid gap-2 md:grid-cols-[1fr_140px_36px]">
                    <select
                      value={selected}
                      onChange={(event) => {
                        const [type, itemId] = event.target.value.split(':')
                        setAttachments((items) => items.map((next, i) => i === index ? { ...next, type, itemId } : next))
                      }}
                      className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 py-2 text-xs outline-none focus:border-[var(--color-accent-border)]"
                    >
                      {ATTACHMENT_OPTIONS.map((option) => (
                        <option key={`${option.type}:${option.itemId}`} value={`${option.type}:${option.itemId}`}>
                          {option.label}
                        </option>
                      ))}
                    </select>
                    <input
                      type="number"
                      min={1}
                      value={item.amount}
                      onChange={(event) => setAttachments((items) => items.map((next, i) => i === index ? { ...next, amount: Number(event.target.value) } : next))}
                      className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 py-2 text-xs outline-none focus:border-[var(--color-accent-border)]"
                    />
                    <button
                      type="button"
                      onClick={() => setAttachments((items) => items.filter((_, i) => i !== index))}
                      className="inline-flex h-9 items-center justify-center rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-muted)] hover:border-red-400 hover:text-red-500 cursor-pointer transition-colors"
                      aria-label="删除附件"
                    >
                      <Trash2 size={13} />
                    </button>
                  </div>
                )
              })}
            </div>
          )}
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <button
            type="submit"
            disabled={loading}
            className="inline-flex items-center gap-1.5 rounded-xl bg-[var(--color-accent)] px-4 py-2 text-xs font-semibold text-white hover:opacity-90 disabled:opacity-50 cursor-pointer transition-opacity"
          >
            <Send size={13} />
            发送信函
          </button>
          <button
            type="button"
            onClick={() => void loadMails(1)}
            disabled={loading || !playerId.trim()}
            className="inline-flex items-center gap-1.5 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-4 py-2 text-xs font-medium text-[var(--color-text-secondary)] hover:border-[var(--color-accent-border)] disabled:opacity-50 cursor-pointer transition-colors"
          >
            <RefreshCw size={13} className={loading ? 'animate-spin' : ''} />
            查看玩家信函
          </button>
          {message && <span className="text-xs text-[var(--color-text-muted)]">{message}</span>}
        </div>
      </form>

      <div className="border-t border-[var(--color-border)] p-4">
        <div className="mb-3 flex flex-wrap items-center gap-2 text-xs text-[var(--color-text-muted)]">
          <span>共 {totalMails} 封</span>
          <span>未读 {unreadMails} 封</span>
          <span className="ml-auto">第 {currentPage} / {totalPages} 页</span>
          <button
            type="button"
            onClick={() => void loadMails(Math.max(1, currentPage - 1))}
            disabled={loading || currentPage <= 1 || !playerId.trim()}
            className="inline-flex h-8 w-8 items-center justify-center rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] text-[var(--color-text-secondary)] hover:border-[var(--color-accent-border)] disabled:opacity-40 cursor-pointer transition-colors"
            aria-label="上一页"
          >
            <ChevronLeft size={14} />
          </button>
          <button
            type="button"
            onClick={() => void loadMails(Math.min(totalPages, currentPage + 1))}
            disabled={loading || currentPage >= totalPages || !playerId.trim()}
            className="inline-flex h-8 w-8 items-center justify-center rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] text-[var(--color-text-secondary)] hover:border-[var(--color-accent-border)] disabled:opacity-40 cursor-pointer transition-colors"
            aria-label="下一页"
          >
            <ChevronRight size={14} />
          </button>
        </div>
        <div className="space-y-2">
          {mails.length === 0 ? (
            <div className="py-6 text-center text-sm text-[var(--color-text-muted)]">暂无信函记录</div>
          ) : mails.map((mail) => (
            <div key={mail.id} className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2">
              <div className="flex items-center gap-2">
                <span className="text-sm font-semibold text-[var(--color-text-primary)]">{mail.title}</span>
                {!mail.isRead && <span className="rounded bg-red-500/10 px-1.5 py-0.5 text-[10px] font-bold text-red-500">未读</span>}
                <span className="ml-auto text-[10px] text-[var(--color-text-muted)]">{new Date(mail.createdAt).toLocaleString('zh-CN')}</span>
              </div>
              <p className="mt-1 line-clamp-2 text-xs text-[var(--color-text-secondary)]">{mail.content}</p>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
