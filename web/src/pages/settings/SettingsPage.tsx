import { type FC } from 'react'
import { Sun, Moon, Monitor, BellOff } from 'lucide-react'
import { useThemeStore } from '@/store/themeStore'
import { useConfirmPreferenceStore } from '@/store/confirmPreferenceStore'
import BattleSimulator from './components/BattleSimulator'

const SettingsPage: FC = () => {
  const { mode, setMode } = useThemeStore()
  const skipConfirmations = useConfirmPreferenceStore((s) => s.skipConfirmations)
  const setSkipConfirmations = useConfirmPreferenceStore((s) => s.setSkipConfirmations)

  const themeOptions = [
    { key: 'light' as const, label: '亮色', icon: Sun },
    { key: 'dark' as const, label: '深色', icon: Moon },
    { key: 'system' as const, label: '跟随系统', icon: Monitor },
  ]

  return (
    <div className="space-y-5 max-w-3xl">
      <BattleSimulator />

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

      {/* 操作确认 */}
      <section className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] overflow-hidden">
        <div className="flex items-center gap-2 px-4 py-3 border-b border-[var(--color-border)]">
          <BellOff size={16} className="text-[var(--color-accent)]" />
          <h2 className="text-sm font-semibold text-[var(--color-text-primary)]">操作确认</h2>
        </div>
        <div className="px-4 py-3">
          <label className="flex items-center justify-between gap-4 cursor-pointer">
            <div>
              <p className="text-sm font-medium text-[var(--color-text-primary)]">跳过二次确认</p>
              <p className="text-xs text-[var(--color-text-muted)] mt-0.5">开启后，城金消费、云存档删除和强力副本提醒会直接执行。</p>
            </div>
            <input
              type="checkbox"
              checked={skipConfirmations}
              onChange={(e) => setSkipConfirmations(e.target.checked)}
              className="w-4 h-4 rounded border-[var(--color-border)] accent-[var(--color-accent)]"
            />
          </label>
        </div>
      </section>
    </div>
  )
}

export default SettingsPage
