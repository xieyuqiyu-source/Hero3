import { Gamepad2 } from 'lucide-react'

function buildWebUrl() {
  const explicit = import.meta.env.VITE_WEB_URL
  if (explicit) return explicit

  const { protocol, hostname } = window.location
  return `${protocol}//${hostname}:5173/`
}

export default function DevAppSwitcher() {
  if (!import.meta.env.DEV) return null

  return (
    <a
      href={buildWebUrl()}
      className="fixed bottom-[72px] right-4 z-[9999] flex items-center gap-2 rounded-full border border-emerald-400/40 bg-slate-950/90 px-4 py-2 text-xs font-black text-emerald-200 shadow-[0_10px_30px_rgba(15,23,42,0.35)] backdrop-blur transition hover:-translate-y-0.5 hover:border-emerald-300 hover:bg-slate-900 lg:bottom-4"
      title="切换到游戏前端"
    >
      <Gamepad2 size={15} />
      游戏前端
    </a>
  )
}
