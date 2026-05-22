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
    <div className="space-y-5 max-w-3xl">
      {/* 外观 */}
      <section className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] overflow-hidden">
        <div className="flex items-center gap-2 px-4 py-3 border-b border-[var(--color-border)]">
          <Sun size={16} className="text-[var(--color-accent)]" />
          <h2 className="text-sm font-semibold text-[var(--color-text-primary)]">外观设置</h2>
        </div>
        <div className="px-4 py-3">
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
                    flex flex-col items-center gap-1.5 px-3 py-3 rounded-xl border cursor-pointer
                    transition-all duration-200
                    ${active
                      ? 'bg-[var(--color-accent-light)] border-[var(--color-accent-border)] text-[var(--color-accent)]'
                      : 'bg-[var(--color-surface-dim)] border-[var(--color-border)] text-[var(--color-text-secondary)] hover:border-[var(--color-text-muted)]'
                    }
                  `}
                >
                  <Icon size={18} />
                  <span className="text-xs font-medium">{opt.label}</span>
                </button>
              )
            })}
          </div>
        </div>
      </section>

      {/* 更多设置预留 */}
      <section className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] overflow-hidden">
        <div className="flex items-center gap-2 px-4 py-3 border-b border-[var(--color-border)]">
          <h2 className="text-sm font-semibold text-[var(--color-text-primary)]">更多设置</h2>
        </div>
        <div className="px-4 py-8 flex items-center justify-center">
          <span className="text-sm text-[var(--color-text-muted)]">更多设置开发中</span>
        </div>
      </section>
    </div>
  )
}

export default SettingsPage
