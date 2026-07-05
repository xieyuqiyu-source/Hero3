/* 本文件实现地图据点页，统一承载黄巾起义等据点类玩法。 */
import { type FC } from 'react'
import { Flag, Lock, Tent } from 'lucide-react'
import YellowTurbanTab from './YellowTurbanTab'

interface StrongholdEntry {
  id: string
  title: string
  subtitle: string
  icon: typeof Flag
  disabled?: boolean
}

const FUTURE_STRONGHOLDS: StrongholdEntry[] = [
  { id: 'mountain-bandits', title: '山寨据点', subtitle: '流寇盘踞，等待开放', icon: Flag, disabled: true },
  { id: 'frontier-camp', title: '边境营寨', subtitle: '边防冲突，等待开放', icon: Tent, disabled: true },
]

/** StrongholdTab 渲染据点页和后续据点预留入口。 */
const StrongholdTab: FC = () => (
  <div className="space-y-5">
    <YellowTurbanTab />

    <section className="space-y-3">
      {FUTURE_STRONGHOLDS.map((entry) => (
        <StrongholdRow key={entry.id} entry={entry} />
      ))}
    </section>
  </div>
)

/** StrongholdRow 渲染单个据点入口行。 */
const StrongholdRow: FC<{ entry: StrongholdEntry }> = ({ entry }) => {
  const Icon = entry.icon
  return (
    <div className="flex min-h-[82px] items-center justify-between gap-3 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-4 py-3 opacity-70">
      <div className="flex min-w-0 items-center gap-3">
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] text-[var(--color-accent)]">
          <Icon size={18} />
        </div>
        <div className="min-w-0">
          <h3 className="truncate text-sm font-bold text-[var(--color-text-primary)]">{entry.title}</h3>
          <p className="mt-1 truncate text-xs text-[var(--color-text-secondary)]">{entry.subtitle}</p>
        </div>
      </div>
      <span className="inline-flex shrink-0 items-center gap-1 rounded-lg border border-[var(--color-border)] px-2.5 py-1 text-xs font-semibold text-[var(--color-text-muted)]">
        <Lock size={13} />
        未开放
      </span>
    </div>
  )
}

export default StrongholdTab
