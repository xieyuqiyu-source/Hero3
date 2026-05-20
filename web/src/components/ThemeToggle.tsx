import { type FC } from 'react'
import { Sun, Moon, Monitor } from 'lucide-react'
import { useThemeStore } from '@/store/themeStore'

const ThemeToggle: FC = () => {
  const { mode, setMode } = useThemeStore()

  const next = () => {
    if (mode === 'light') setMode('dark')
    else if (mode === 'dark') setMode('system')
    else setMode('light')
  }

  const icon = mode === 'light' ? Sun : mode === 'dark' ? Moon : Monitor
  const Icon = icon
  const label = mode === 'light' ? '亮色' : mode === 'dark' ? '深色' : '跟随系统'

  return (
    <button
      type="button"
      onClick={next}
      className="
        flex items-center gap-1.5 px-2.5 py-1.5 rounded-xl
        text-xs text-[var(--color-text-secondary)]
        hover:text-[var(--color-text-primary)]
        hover:bg-[var(--color-accent-light)]
        cursor-pointer transition-all duration-200
      "
      title={`当前: ${label}，点击切换`}
      aria-label={`切换主题，当前: ${label}`}
    >
      <Icon size={14} />
      <span className="hidden sm:inline">{label}</span>
    </button>
  )
}

export default ThemeToggle
