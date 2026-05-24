interface Stat {
  label: string
  value: string
  hint: string
}

interface MetricsGridProps {
  stats: Stat[]
}

export default function MetricsGrid({ stats }: MetricsGridProps) {
  return (
    <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
      {stats.map((stat) => (
        <div
          key={stat.label}
          className="
            relative overflow-hidden px-4 py-3.5 rounded-2xl
            border border-[var(--color-border)]
            bg-[var(--color-surface)]
            shadow-[var(--shadow-panel)]
          "
        >
          <div className="absolute inset-y-0 left-0 w-[3px] bg-gradient-to-b from-[var(--color-gold)] to-[var(--color-accent)]" />
          <span className="text-xs text-[var(--color-text-secondary)]">{stat.label}</span>
          <strong className="block mt-1.5 text-2xl font-black text-[var(--color-text-primary)] leading-none">
            {stat.value}
          </strong>
          <small className="text-[11px] text-[var(--color-text-muted)] mt-1 block">{stat.hint}</small>
        </div>
      ))}
    </div>
  )
}
