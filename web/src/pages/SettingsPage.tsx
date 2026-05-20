import { type FC } from 'react'
import { Sun, Moon, Monitor } from 'lucide-react'
import { useThemeStore } from '@/store/themeStore'

const SettingsPage: FC = () => {
  const { mode, setMode } = useThemeStore()

  const themeOptions = [
    { key: 'light' as const, label: '亮色', icon: Sun },
    { key: 'dark' as const, label: '深色', icon: Moon },
    { key: 'system' as const, label: '跟随系统', icon: Monitor },
  ]

  return (
    <div className="space-y-6">
      {/* Theme */}
      <section>
        <h2 className="text-sm font-semibold text-[var(--color-text-primary)] mb-3">外观</h2>
        <div className="grid grid-cols-3 gap-2">
          {themeOptions.map((opt) => {
            const Icon = opt.icon
            const active = mode === opt.key
            return (
              <button
                key={opt.key}
                type="button"
                onClick={() => setMode(opt.key)}
                className={`
                  flex flex-col items-center gap-1.5 px-3 py-4 rounded-2xl border cursor-pointer
                  transition-all duration-200
                  ${active
                    ? 'bg-[var(--color-accent-light)] border-[var(--color-accent-border)] text-[var(--color-accent)]'
                    : 'bg-[var(--color-surface)] border-[var(--color-border)] text-[var(--color-text-secondary)] hover:border-[var(--color-text-muted)]'
                  }
                `}
              >
                <Icon size={20} />
                <span className="text-xs font-medium">{opt.label}</span>
              </button>
            )
          })}
        </div>
      </section>

      {/* Sound - placeholder */}
      <section>
        <h2 className="text-sm font-semibold text-[var(--color-text-primary)] mb-3">音效</h2>
        <div className="px-4 py-3 rounded-2xl bg-[var(--color-surface)] border border-[var(--color-border)]">
          <span className="text-sm text-[var(--color-text-secondary)]">游戏音效（预留）</span>
        </div>
      </section>

      {/* Account - placeholder */}
      <section>
        <h2 className="text-sm font-semibold text-[var(--color-text-primary)] mb-3">账号</h2>
        <div className="px-4 py-3 rounded-2xl bg-[var(--color-surface)] border border-[var(--color-border)]">
          <span className="text-sm text-[var(--color-text-secondary)]">玩家信息（预留）</span>
        </div>
      </section>
    </div>
  )
}

export default SettingsPage
