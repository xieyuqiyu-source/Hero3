import { resourceActions, systemActions } from '@/data'

export function ResourceToolsPanel() {
  return (
    <article className="panel" id="资源">
      <div className="panel-heading">
        <div>
          <p className="eyebrow">Resource Tools</p>
          <h2>资源调整</h2>
        </div>
      </div>

      <div className="action-list">
        {resourceActions.map((action) => (
          <button type="button" key={action.title} disabled>
            <strong>{action.title}</strong>
            <span>{action.desc}</span>
          </button>
        ))}
      </div>
    </article>
  )
}

export function SystemActionsPanel() {
  return (
    <article className="panel" id="系统">
      <div className="panel-heading">
        <div>
          <p className="eyebrow">System Actions</p>
          <h2>系统操作</h2>
        </div>
      </div>

      <div className="system-actions">
        {systemActions.map((action) => (
          <button type="button" className={action.level} key={action.title} disabled>
            {action.title}
          </button>
        ))}
      </div>
    </article>
  )
}
