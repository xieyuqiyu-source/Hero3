import { useMemo, useState, type FC } from 'react'
import {
  ChevronLeft,
  ChevronRight,
  Mail,
  MailOpen,
  X,
} from 'lucide-react'
import MailDetail from './components/MailDetail'
import { MOCK_MAILS, PAGE_SIZE, TYPE_CONFIG, TYPE_OPTIONS, type MailType, type MockMail } from './data'

const MailPage: FC = () => {
  const [activeType, setActiveType] = useState<MailType>('all')
  const [currentPage, setCurrentPage] = useState(1)
  const [selectedMail, setSelectedMail] = useState<MockMail | null>(null)

  const filteredMails = useMemo(() => {
    const items = activeType === 'all' ? MOCK_MAILS : MOCK_MAILS.filter((mail) => mail.type === activeType)
    return items
  }, [activeType])

  const totalPages = Math.max(1, Math.ceil(filteredMails.length / PAGE_SIZE))
  const pageMails = filteredMails.slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE)
  const unreadCount = MOCK_MAILS.filter((mail) => !mail.read).length

  const handleTypeChange = (type: MailType) => {
    setActiveType(type)
    setCurrentPage(1)
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
          {pageMails.map((mail, index) => {
            const config = TYPE_CONFIG[mail.type]
            const Icon = mail.read ? MailOpen : config.icon
            return (
              <button
                key={mail.id}
                type="button"
                onClick={() => setSelectedMail(mail)}
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
                      {!mail.read && <span className="h-2 w-2 flex-shrink-0 rounded-full bg-red-500" />}
                    </span>
                    <span className="mt-1 flex items-center gap-2 text-[10px] text-[var(--color-text-muted)]">
                      <span>{config.label}</span>
                      <span>{mail.sender}</span>
                      <span className="ml-auto">{mail.createdAt}</span>
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
            <span className="ml-2">共 {filteredMails.length} 封</span>
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
              <MailDetail mail={selectedMail} />
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

export default MailPage
