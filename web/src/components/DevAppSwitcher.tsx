import { ShieldCheck } from 'lucide-react'

function buildAdminUrl() {
  const explicit = import.meta.env.VITE_ADMIN_URL
  if (explicit) return explicit

  const { protocol, hostname } = window.location
  return `${protocol}//${hostname}:5174/admin/`
}

export default function DevAppSwitcher() {
  if (!import.meta.env.DEV) return null

  return (
    <a
      href={buildAdminUrl()}
      className="fixed bottom-4 right-4 z-[9999] flex items-center gap-2 rounded-full border border-amber-400/40 bg-slate-950/90 px-4 py-2 text-xs font-black text-amber-200 shadow-[0_10px_30px_rgba(15,23,42,0.35)] backdrop-blur transition hover:-translate-y-0.5 hover:border-amber-300 hover:bg-slate-900"
      title="切换到 GM 后台"
    >
      <ShieldCheck size={15} />
      GM 后台
    </a>
  )
}
