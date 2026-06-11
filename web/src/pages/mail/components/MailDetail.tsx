import { type FC } from 'react'
import { Bell, Gift, Loader2, Trash2 } from 'lucide-react'
import type { Mail } from '@/types/game'
import { TYPE_CONFIG } from '../data'

const MailDetail: FC<{
  mail: Mail
  onDelete: () => void
  onClaim: () => void
  claiming: boolean
}> = ({ mail, onDelete, onClaim, claiming }) => {
  const config = TYPE_CONFIG[mail.mailType]
  const Icon = config.icon
  const hasAttachment = (mail.attachments?.length ?? 0) > 0

  return (
    <div className="min-w-0">
      <div className="border-b border-[var(--color-border)] px-3 py-4 sm:px-4">
        <div className="flex min-w-0 items-start gap-3">
          <span className={`inline-flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-2xl ${config.color}`}>
            <Icon size={18} />
          </span>
          <div className="min-w-0 flex-1">
            <div className="flex min-w-0 flex-wrap items-center gap-2">
              <span className={`shrink-0 rounded px-1.5 py-0.5 text-[10px] font-bold ${config.color}`}>{config.label}</span>
              {!mail.isRead && <span className="rounded bg-red-500/10 px-1.5 py-0.5 text-[10px] font-bold text-red-500">未读</span>}
              {mail.expiresAt && <span className="min-w-0 break-words text-[10px] text-[var(--color-text-muted)]">有效期至 {formatMailDate(mail.expiresAt)}</span>}
            </div>
            <h2 className="mt-2 break-words text-base font-bold text-[var(--color-text-primary)]">{mail.title}</h2>
            <p className="mt-1 break-words text-xs text-[var(--color-text-muted)]">{mail.senderName} · {new Date(mail.createdAt).toLocaleString('zh-CN')}</p>
          </div>
        </div>
      </div>

      <div className="min-w-0 space-y-4 px-3 py-4 sm:px-4">
        <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-4 py-4">
          <p className="break-words whitespace-pre-line text-sm leading-7 text-[var(--color-text-secondary)]">{mail.content}</p>
        </div>

        <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-4 py-4">
          <div className="flex min-w-0 items-center gap-2">
            <Gift size={15} className="shrink-0 text-[var(--color-accent)]" />
            <h3 className="text-sm font-semibold text-[var(--color-text-primary)]">附件</h3>
            {hasAttachment && (
              <span className={`ml-auto shrink-0 rounded px-1.5 py-0.5 text-[10px] font-bold ${mail.isClaimed ? 'bg-green-500/10 text-green-600' : 'bg-amber-500/10 text-amber-600'}`}>
                {mail.isClaimed ? '已领取' : '待领取'}
              </span>
            )}
          </div>
          {hasAttachment ? (
            <div className="mt-3 grid min-w-0 grid-cols-2 gap-2 sm:grid-cols-4">
              {mail.attachments?.map((item, index) => (
                <div key={`${item.type}-${item.itemId}-${index}`} className="min-w-0 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2">
                  <p className="truncate text-xs font-medium text-[var(--color-text-primary)]">{item.itemId}</p>
                  <p className="mt-1 truncate text-[10px] text-[var(--color-text-muted)]">x{item.amount.toLocaleString()}</p>
                </div>
              ))}
            </div>
          ) : (
            <p className="mt-3 text-xs text-[var(--color-text-muted)]">该信函没有附件。</p>
          )}
        </div>

        <div className="flex flex-wrap justify-end gap-2">
          <button
            type="button"
            onClick={onDelete}
            className="inline-flex items-center gap-1.5 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-xs font-medium text-[var(--color-text-secondary)] hover:border-red-400 hover:text-red-500 cursor-pointer transition-colors"
          >
            <Trash2 size={13} />
            删除
          </button>
          {hasAttachment && !mail.isClaimed && (
            <button
              type="button"
              onClick={onClaim}
              disabled={claiming}
              className="inline-flex items-center gap-1.5 rounded-xl bg-[var(--color-accent)] px-3 py-2 text-xs font-semibold text-white hover:opacity-90 cursor-pointer transition-opacity"
            >
              {claiming ? <Loader2 size={13} className="animate-spin" /> : <Bell size={13} />}
              {claiming ? '领取中' : '领取附件'}
            </button>
          )}
        </div>
      </div>
    </div>
  )
}

function formatMailDate(dateStr: string): string {
  const date = new Date(dateStr)
  if (Number.isNaN(date.getTime())) return dateStr
  return date.toLocaleString('zh-CN')
}

export default MailDetail
