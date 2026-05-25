import { useState, type ReactNode } from 'react'
import { ChevronDown, ChevronRight } from 'lucide-react'

interface CollapsiblePanelProps {
  icon: ReactNode
  title: string
  badge?: string
  defaultOpen?: boolean
  children: ReactNode
}

export default function CollapsiblePanel({ icon, title, badge, defaultOpen = false, children }: CollapsiblePanelProps) {
  const [open, setOpen] = useState(defaultOpen)

  return (
    <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] shadow-[var(--shadow-panel)] overflow-hidden">
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="w-full flex items-center gap-2.5 px-4 py-3 cursor-pointer hover:bg-[var(--color-surface-dim)] transition-colors"
      >
        {icon}
        <h2 className="text-sm font-bold text-[var(--color-text-primary)] flex-1 text-left">{title}</h2>
        {badge && (
          <span className="text-[11px] text-[var(--color-text-muted)] mr-2">{badge}</span>
        )}
        {open
          ? <ChevronDown size={14} className="text-[var(--color-text-muted)]" />
          : <ChevronRight size={14} className="text-[var(--color-text-muted)]" />
        }
      </button>
      {open && <div className="[&>*]:rounded-none [&>*]:border-0 [&>*]:shadow-none">{children}</div>}
    </div>
  )
}
