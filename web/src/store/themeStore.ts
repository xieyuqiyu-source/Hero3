import { create } from 'zustand'

type ThemeMode = 'system' | 'light' | 'dark'

interface ThemeStore {
  mode: ThemeMode
  setMode: (mode: ThemeMode) => void
}

function getStoredMode(): ThemeMode {
  const stored = localStorage.getItem('hero3-theme')
  if (stored === 'light' || stored === 'dark' || stored === 'system') return stored
  return 'system'
}

function applyTheme(mode: ThemeMode) {
  const root = document.documentElement
  if (mode === 'dark') {
    root.classList.add('dark')
  } else if (mode === 'light') {
    root.classList.remove('dark')
  } else {
    // system
    if (window.matchMedia('(prefers-color-scheme: dark)').matches) {
      root.classList.add('dark')
    } else {
      root.classList.remove('dark')
    }
  }
}

// Initialize on load
const initialMode = getStoredMode()
applyTheme(initialMode)

// Listen for system changes when in system mode
window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
  const current = useThemeStore.getState().mode
  if (current === 'system') {
    applyTheme('system')
  }
})

export const useThemeStore = create<ThemeStore>((set) => ({
  mode: initialMode,
  setMode: (mode) => {
    localStorage.setItem('hero3-theme', mode)
    applyTheme(mode)
    set({ mode })
  },
}))
