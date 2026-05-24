import { Package, Zap } from 'lucide-react'

const resourceActions = [
  { title: '补发资源', desc: '通过玩家详情面板操作' },
  { title: '清理队列', desc: '待队列模型接入后启用' },
  { title: '重置地图', desc: '待地图刷新规则接入后启用' },
]

const systemActions = [
  { title: '发布维护公告', level: 'normal' },
  { title: '刷新配置缓存', level: 'normal' },
  { title: '冻结玩家操作', level: 'danger' },
]

export function ResourceToolsPanel() {
  return (
    <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] shadow-[var(--shadow-panel)] p-4">
      <div className="flex items-center gap-2 mb-3">
        <Package size={16} className="text-[var(--color-accent)]" />
        <h2 className="text-base font-bold text-[var(--color-text-primary)]">资源工具</h2>
      </div>
      <div className="grid gap-2">
        {resourceActions.map((action) => (
          <button
            key={action.title}
            type="button"
            disabled
            className="
              flex flex-col gap-0.5 px-3 py-2.5 rounded-xl text-left
              border border-[var(--color-border)] bg-[var(--color-surface-dim)]
              opacity-60 cursor-not-allowed
            "
          >
            <strong className="text-sm text-[var(--color-text-primary)]">{action.title}</strong>
            <span className="text-[11px] text-[var(--color-text-muted)]">{action.desc}</span>
          </button>
        ))}
      </div>
    </div>
  )
}

export function SystemActionsPanel() {
  return (
    <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] shadow-[var(--shadow-panel)] p-4">
      <div className="flex items-center gap-2 mb-3">
        <Zap size={16} className="text-[var(--color-accent)]" />
        <h2 className="text-base font-bold text-[var(--color-text-primary)]">系统操作</h2>
      </div>
      <div className="grid gap-2">
        {systemActions.map((action) => (
          <button
            key={action.title}
            type="button"
            disabled
            className={`
              px-3 py-2.5 rounded-xl text-left text-sm font-medium
              border cursor-not-allowed opacity-60
              ${action.level === 'danger'
                ? 'border-red-500/20 bg-red-500/5 text-red-600'
                : 'border-[var(--color-border)] bg-[var(--color-surface-dim)] text-[var(--color-text-primary)]'
              }
            `}
          >
            {action.title}
          </button>
        ))}
      </div>
    </div>
  )
}
