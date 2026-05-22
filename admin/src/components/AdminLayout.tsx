import type { ReactNode } from 'react'

export type AdminPage = 'overview' | 'accounts' | 'operations' | 'balance' | 'api' | 'audit'

const navItems: Array<{ key: AdminPage; label: string }> = [
  { key: 'overview', label: '总览' },
  { key: 'accounts', label: '玩家' },
  { key: 'operations', label: '操作' },
  { key: 'balance', label: '配置' },
  { key: 'api', label: '接口' },
  { key: 'audit', label: '审计' },
]

const pageTitles: Record<AdminPage, { eyebrow: string; title: string }> = {
  overview: { eyebrow: 'GM Console', title: 'Hero3 管理后台' },
  accounts: { eyebrow: 'Accounts', title: '注册玩家与云存档' },
  operations: { eyebrow: 'Operations', title: '运营操作' },
  balance: { eyebrow: 'Balance', title: '游戏数值配置' },
  api: { eyebrow: 'API Diagnostics', title: '接口文档与诊断' },
  audit: { eyebrow: 'Audit', title: '审计与安全边界' },
}

interface AdminLayoutProps {
  activePage: AdminPage
  children: ReactNode
  environment: string
  loading: boolean
  onNavigate: (page: AdminPage) => void
}

export default function AdminLayout({
  activePage,
  children,
  environment,
  loading,
  onNavigate,
}: AdminLayoutProps) {
  const currentPage = pageTitles[activePage]

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
          {navItems.map((item) => (
            <button
              type="button"
              className={item.key === activePage ? 'active' : ''}
              onClick={() => onNavigate(item.key)}
              key={item.key}
            >
              {item.label}
            </button>
          ))}
        </nav>
      </aside>

      <main className="admin-main">
        <header className="topbar">
          <div>
            <p className="eyebrow">{currentPage.eyebrow}</p>
            <h1>{currentPage.title}</h1>
          </div>
          <div className="operator-chip">
            <span>{loading ? '正在连接' : '当前环境'}</span>
            <strong>{environment}</strong>
          </div>
        </header>

        {children}
      </main>
    </div>
  )
}
