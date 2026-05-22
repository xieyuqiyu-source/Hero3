interface DashboardStat {
  label: string
  value: string
  hint: string
}

interface MetricsGridProps {
  stats: DashboardStat[]
}

export default function MetricsGrid({ stats }: MetricsGridProps) {
  return (
    <section className="metrics-grid" aria-label="运营指标">
      {stats.map((stat) => (
        <article className="metric-card" key={stat.label}>
          <span>{stat.label}</span>
          <strong>{stat.value}</strong>
          <small>{stat.hint}</small>
        </article>
      ))}
    </section>
  )
}
