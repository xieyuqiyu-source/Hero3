import { useEffect, useMemo, useState } from 'react'
import { adminApi } from '@/api/admin'
import type { BalanceConfig } from '@/types'

function formatBalance(balance: BalanceConfig) {
  return JSON.stringify(balance, null, 2)
}

export default function BalanceConfigPanel() {
  const [balance, setBalance] = useState<BalanceConfig | null>(null)
  const [draft, setDraft] = useState('')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const buildingCount = useMemo(() => Object.keys(balance?.buildings ?? {}).length, [balance])

  useEffect(() => {
    let cancelled = false
    adminApi.getBalance()
      .then((nextBalance) => {
        if (cancelled) return
        setBalance(nextBalance)
        setDraft(formatBalance(nextBalance))
        setError(null)
      })
      .catch((loadError) => {
        if (cancelled) return
        setError(loadError instanceof Error ? loadError.message : '配置加载失败')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [])

  const handleFormat = () => {
    try {
      const parsed = JSON.parse(draft) as BalanceConfig
      setDraft(formatBalance(parsed))
      setError(null)
    } catch {
      setError('JSON 格式不正确，无法格式化')
    }
  }

  const handleSave = async () => {
    setSaving(true)
    setMessage(null)
    try {
      const parsed = JSON.parse(draft) as BalanceConfig
      const nextBalance = await adminApi.updateBalance(parsed)
      setBalance(nextBalance)
      setDraft(formatBalance(nextBalance))
      setError(null)
      setMessage('数值配置已保存，后续资源结算会读取新配置')
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : '配置保存失败')
    } finally {
      setSaving(false)
    }
  }

  return (
    <article className="panel balance-panel">
      <div className="panel-header">
        <div>
          <p className="eyebrow">Balance Config</p>
          <h2>资源与建筑数值配置</h2>
        </div>
        <span className="panel-count">{buildingCount} 类建筑</span>
      </div>

      <div className="balance-summary">
        {Object.entries(balance?.baseProduction ?? {}).map(([resource, value]) => (
          <div key={resource}>
            <span>{resource}</span>
            <strong>{value}/小时</strong>
          </div>
        ))}
      </div>

      <textarea
        className="config-editor"
        value={draft}
        onChange={(event) => setDraft(event.target.value)}
        spellCheck={false}
        disabled={loading || saving}
      />

      <div className="panel-actions">
        <button type="button" className="ghost-button" onClick={handleFormat} disabled={loading || saving}>
          格式化 JSON
        </button>
        <button type="button" className="danger-button" onClick={() => void handleSave()} disabled={loading || saving}>
          {saving ? '保存中' : '保存配置'}
        </button>
      </div>

      {message && <p className="inline-success">{message}</p>}
      {error && <p className="inline-error">{error}</p>}
    </article>
  )
}
