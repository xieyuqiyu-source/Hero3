import { type ReactNode } from 'react'
import {
  LayoutDashboard,
  Users,
  Wrench,
  Sliders,
  Coins,
  FileText,
  Mail,
  Shield,
} from 'lucide-react'

export type AdminPage = 'overview' | 'accounts' | 'operations' | 'mails' | 'ledger' | 'balance' | 'api' | 'audit'

const navItems: Array<{ key: AdminPage; label: string; icon: typeof LayoutDashboard }> = [
  { key: 'overview', label: '总览', icon: LayoutDashboard },
  { key: 'accounts', label: '玩家', icon: Users },
  { key: 'operations', label: '操作', icon: Wrench },
  { key: 'mails', label: '信函', icon: Mail },
  { key: 'ledger', label: '流水', icon: Coins },
  { key: 'balance', label: '配置', icon: Sliders },
  { key: 'api', label: '接口', icon: FileText },
  { key: 'audit', label: '审计', icon: Shield },
]

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
  return (
    <div className="flex min-h-dvh">
      {/* Sidebar */}
      <aside className="
        fixed left-3 top-3 z-50 flex flex-col
        h-[calc(100dvh-24px)] w-[240px] rounded-3xl
        bg-[var(--color-surface)] backdrop-blur-[14px]
        border border-[var(--color-border)]
        shadow-[0_18px_44px_rgba(15,23,42,0.12)]
        max-lg:hidden
      ">
        {/* Brand */}
        <div className="flex items-center gap-3 px-5 py-4 border-b border-[var(--color-border)]">
          <div className="
            w-9 h-9 flex items-center justify-center rounded-xl
            bg-gradient-to-br from-amber-500 to-orange-600
            text-white font-black text-sm
            shadow-[0_8px_20px_rgba(245,158,11,0.25)]
          ">
            GM
          </div>
          <div className="flex flex-col">
            <span className="text-sm font-bold text-[var(--color-text-primary)]">Hero3</span>
            <span className="text-[10px] text-[var(--color-text-muted)] tracking-widest">管理后台</span>
          </div>
        </div>

        {/* Nav */}
        <nav className="flex-1 overflow-y-auto px-2.5 py-3 scrollbar-none">
          <div className="grid gap-1">
            {navItems.map((item) => {
              const Icon = item.icon
              const isActive = item.key === activePage
              return (
                <button
                  key={item.key}
                  type="button"
                  onClick={() => onNavigate(item.key)}
                  className={`
                    flex items-center gap-2.5 px-3 py-2.5 rounded-xl
                    text-sm font-medium cursor-pointer
                    transition-all duration-200 border
                    ${isActive
                      ? 'bg-[var(--color-accent-light)] border-[var(--color-accent-border)] text-[var(--color-accent)]'
                      : 'bg-transparent border-transparent text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-dim)] hover:text-[var(--color-text-primary)]'
                    }
                  `}
                >
                  <Icon size={16} />
                  <span>{item.label}</span>
                </button>
              )
            })}
          </div>
        </nav>

        {/* Status */}
        <div className="px-3 py-3 border-t border-[var(--color-border)]">
          <div className="flex items-center gap-2 px-2.5 py-2 rounded-xl bg-[var(--color-surface-dim)] border border-[var(--color-border)]">
            <div className={`w-2 h-2 rounded-full ${loading ? 'bg-amber-400 animate-pulse' : 'bg-emerald-500'}`} />
            <span className="text-[11px] text-[var(--color-text-secondary)]">{environment}</span>
          </div>
        </div>
      </aside>

      {/* Mobile Nav */}
      <nav className="
        fixed bottom-0 left-0 right-0 z-40
        flex items-center justify-around
        h-14 px-2
        bg-[var(--color-surface)]/90 backdrop-blur-md
        border-t border-[var(--color-border)]
        lg:hidden
      ">
        {navItems.slice(0, 5).map((item) => {
          const Icon = item.icon
          const isActive = item.key === activePage
          return (
            <button
              key={item.key}
              type="button"
              onClick={() => onNavigate(item.key)}
              className={`
                flex flex-col items-center gap-0.5 px-2 py-1.5 rounded-lg
                cursor-pointer transition-colors duration-200
                ${isActive ? 'text-[var(--color-accent)]' : 'text-[var(--color-text-muted)]'}
              `}
            >
              <Icon size={18} />
              <span className="text-[10px] font-bold">{item.label}</span>
            </button>
          )
        })}
      </nav>

      {/* Main Content */}
      <main className="flex-1 lg:ml-[264px]">
        <div className="max-w-[1200px] mx-auto px-4 py-6 lg:px-6 lg:py-8 pb-20 lg:pb-8">
          {children}
        </div>
      </main>
    </div>
  )
}
