import { type FC } from 'react'
import { Bell, Gift, Trash2 } from 'lucide-react'
import { TYPE_CONFIG, type MockMail } from '../data'

const MailDetail: FC<{ mail: MockMail }> = ({ mail }) => {
  const config = TYPE_CONFIG[mail.type]
  const Icon = config.icon

  return (
    <div>
      <div className="border-b border-[var(--color-border)] px-4 py-4">
        <div className="flex items-start gap-3">
          <span className={`inline-flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-2xl ${config.color}`}>
            <Icon size={18} />
          </span>
          <div className="min-w-0 flex-1">
            <div className="flex flex-wrap items-center gap-2">
              <span className={`rounded px-1.5 py-0.5 text-[10px] font-bold ${config.color}`}>{config.label}</span>
              {!mail.read && <span className="rounded bg-red-500/10 px-1.5 py-0.5 text-[10px] font-bold text-red-500">未读</span>}
              {mail.expiresAt && <span className="text-[10px] text-[var(--color-text-muted)]">有效期至 {mail.expiresAt}</span>}
            </div>
            <h2 className="mt-2 text-base font-bold text-[var(--color-text-primary)]">{mail.title}</h2>
            <p className="mt-1 text-xs text-[var(--color-text-muted)]">{mail.sender} · {mail.createdAt}</p>
          </div>
        </div>
      </div>

      <div className="space-y-4 px-4 py-4">
        <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-4 py-4">
          <p className="whitespace-pre-line text-sm leading-7 text-[var(--color-text-secondary)]">{mail.content}</p>
        </div>

        <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-4 py-4">
          <div className="flex items-center gap-2">
            <Gift size={15} className="text-[var(--color-accent)]" />
            <h3 className="text-sm font-semibold text-[var(--color-text-primary)]">附件</h3>
            {mail.hasAttachment && (
              <span className={`ml-auto rounded px-1.5 py-0.5 text-[10px] font-bold ${mail.claimed ? 'bg-green-500/10 text-green-600' : 'bg-amber-500/10 text-amber-600'}`}>
                {mail.claimed ? '已领取' : '待领取'}
              </span>
            )}
          </div>
          {mail.hasAttachment ? (
            <div className="mt-3 grid grid-cols-2 gap-2 sm:grid-cols-4">
              {['木材', '石料', '城金', '将领经验'].map((item) => (
                <div key={item} className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2">
                  <p className="text-xs font-medium text-[var(--color-text-primary)]">{item}</p>
                  <p className="mt-1 text-[10px] text-[var(--color-text-muted)]">后续接入真实附件</p>
                </div>
              ))}
            </div>
          ) : (
            <p className="mt-3 text-xs text-[var(--color-text-muted)]">该信函没有附件。</p>
          )}
        </div>

        <div className="flex justify-end gap-2">
          <button
            type="button"
            className="inline-flex items-center gap-1.5 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-xs font-medium text-[var(--color-text-secondary)] hover:border-red-400 hover:text-red-500 cursor-pointer transition-colors"
          >
            <Trash2 size={13} />
            删除
          </button>
          {mail.hasAttachment && !mail.claimed && (
            <button
              type="button"
              className="inline-flex items-center gap-1.5 rounded-xl bg-[var(--color-accent)] px-3 py-2 text-xs font-semibold text-white hover:opacity-90 cursor-pointer transition-opacity"
            >
              <Bell size={13} />
              领取附件
            </button>
          )}
        </div>
      </div>
    </div>
  )
}

export default MailDetail
