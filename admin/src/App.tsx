import './App.css'
import {
  auditLogs,
  guardrails,
  resourceActions,
  systemActions,
} from './adminData'
import PlayerStatePanel from './PlayerStatePanel'
import { useAdminDashboard } from './useAdminDashboard'

function App() {
  const { dashboardStats, error, gameState, health, loading } = useAdminDashboard()

  return (
    <div className="admin-shell">
      <aside className="admin-sidebar" aria-label="GM 后台导航">
        <div className="brand-block">
          <span className="brand-mark">H3</span>
          <div>
            <strong>Hero3 GM</strong>
            <span>运营管理台</span>
          </div>
        </div>

        <nav className="nav-list">
          {['总览', '玩家', '资源', '战斗', '公告', '审计'].map((item) => (
            <a href={`#${item}`} key={item}>
              {item}
            </a>
          ))}
        </nav>
      </aside>

      <main className="admin-main">
        <header className="topbar">
          <div>
            <p className="eyebrow">GM Console</p>
            <h1>Hero3 管理后台</h1>
          </div>
          <div className="operator-chip">
            <span>{loading ? '正在连接' : '当前环境'}</span>
            <strong>{health?.environment ?? 'Offline'}</strong>
          </div>
        </header>

        {error && (
          <section className="status-banner">
            后端未连接：{error}
          </section>
        )}

        <section className="metrics-grid" aria-label="运营指标">
          {dashboardStats.map((stat) => (
            <article className="metric-card" key={stat.label}>
              <span>{stat.label}</span>
              <strong>{stat.value}</strong>
              <small>{stat.hint}</small>
            </article>
          ))}
        </section>

        <section className="workspace-grid">
          <article className="panel player-panel" id="玩家">
            <div className="panel-heading">
              <div>
                <p className="eyebrow">Player Lookup</p>
                <h2>玩家检索</h2>
              </div>
              <button type="button">查询</button>
            </div>

            <label className="search-field">
              <span>玩家 ID / 昵称</span>
              <input value={gameState?.player.id ?? 'demo-player'} readOnly />
            </label>

            <PlayerStatePanel gameState={gameState} />
          </article>

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

          <article className="panel" id="审计">
            <div className="panel-heading">
              <div>
                <p className="eyebrow">Audit Log</p>
                <h2>操作审计</h2>
              </div>
            </div>

            <ol className="audit-list">
              {auditLogs.map((log) => (
                <li key={log.time}>
                  <time>{log.time}</time>
                  <span>{log.action}</span>
                </li>
              ))}
            </ol>
          </article>
        </section>

        <section className="guardrail-panel">
          {guardrails.map((item) => (
            <div key={item.title}>
              <strong>{item.title}</strong>
              <span>{item.desc}</span>
            </div>
          ))}
        </section>
      </main>
    </div>
  )
}

export default App
