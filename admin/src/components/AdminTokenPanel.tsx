import { useState, useEffect } from 'react'
import { Key, Eye, EyeOff } from 'lucide-react'

/**
 * GM Admin Token 配置面板
 * 用于配置 X-Admin-Token，存储在 localStorage 中
 */
export default function AdminTokenPanel() {
  const [token, setToken] = useState('')
  const [showToken, setShowToken] = useState(false)
  const [savedMsg, setSavedMsg] = useState('')

  useEffect(() => {
    const stored = localStorage.getItem('hero3_admin_token') ?? ''
    setToken(stored)
  }, [])

  const handleSave = () => {
    if (token.trim()) {
      localStorage.setItem('hero3_admin_token', token.trim())
      setSavedMsg('✅ 已保存，刷新页面后生效')
    } else {
      localStorage.removeItem('hero3_admin_token')
      setSavedMsg('✅ 已清除 admin token')
    }
    setTimeout(() => setSavedMsg(''), 3000)
  }

  return (
    <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] shadow-[var(--shadow-panel)] p-4">
      <div className="flex items-center gap-2 mb-3">
        <Key size={16} className="text-amber-500" />
        <h2 className="text-base font-bold text-[var(--color-text-primary)]">Admin Token</h2>
      </div>

      <p className="text-[10px] text-[var(--color-text-muted)] mb-3">
        所有 GM 接口都需要在请求头带上 <code className="px-1 bg-[var(--color-surface-dim)] rounded">X-Admin-Token</code>。
        token 与后端 <code className="px-1 bg-[var(--color-surface-dim)] rounded">HERO3_ADMIN_TOKEN</code> 环境变量保持一致。
      </p>

      <div className="flex gap-2">
        <div className="flex-1 relative">
          <input
            type={showToken ? 'text' : 'password'}
            value={token}
            onChange={(e) => setToken(e.target.value)}
            placeholder="输入 admin token"
            className="w-full px-3 py-2 pr-9 rounded-xl text-xs border border-[var(--color-border)] bg-[var(--color-surface-dim)] text-[var(--color-text-primary)] placeholder-[var(--color-text-muted)] outline-none focus:border-[var(--color-accent-border)]"
          />
          <button
            type="button"
            onClick={() => setShowToken(!showToken)}
            className="absolute right-2 top-1/2 -translate-y-1/2 text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] cursor-pointer"
          >
            {showToken ? <EyeOff size={14} /> : <Eye size={14} />}
          </button>
        </div>
        <button
          type="button"
          onClick={handleSave}
          className="px-4 py-2 rounded-xl text-xs font-medium bg-[var(--color-accent)] text-white hover:opacity-90 cursor-pointer"
        >
          保存
        </button>
      </div>

      {savedMsg && (
        <div className="mt-2 px-3 py-1.5 rounded-lg bg-[var(--color-surface-dim)] text-[11px] text-[var(--color-text-primary)]">
          {savedMsg}
        </div>
      )}
    </div>
  )
}
