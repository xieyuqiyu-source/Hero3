import { type FC } from 'react'

interface SectionProps {
  title: string
  icon: FC<{ size?: number; className?: string }>
  badge?: string
  children: React.ReactNode
}

const Section: FC<SectionProps> = ({ title, icon: Icon, badge, children }) => (
  <section className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] overflow-hidden">
    <div className="flex items-center justify-between px-4 py-3 border-b border-[var(--color-border)]">
      <div className="flex items-center gap-2">
        <Icon size={16} className="text-[var(--color-accent)]" />
        <h2 className="text-sm font-semibold text-[var(--color-text-primary)]">{title}</h2>
      </div>
      {badge && <span className="text-xs text-[var(--color-text-muted)]">{badge}</span>}
    </div>
    <div className="px-4 py-3">
      {children}
    </div>
  </section>
)

export default Section
