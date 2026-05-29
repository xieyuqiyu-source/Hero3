import { useState } from 'react'
import { Package, Zap, Coins, Gift } from 'lucide-react'
import { adminApi } from '@/api/admin'
import PlayerSelector from './PlayerSelector'

export function ResourceToolsPanel() {
  const [playerId, setPlayerId] = useState('')
  const [accountId, setAccountId] = useState('')
  const [goldAmount, setGoldAmount] = useState(10)
  const [message, setMessage] = useState('')
  const [loading, setLoading] = useState(false)

  const showMsg = (msg: string) => {
    setMessage(msg)
    setTimeout(() => setMessage(''), 3000)
  }

  const handleAddAccountGold = async () => {
    if (!accountId || goldAmount <= 0) return
    if (!confirm(`确认给账户 ${accountId} 赠送 ${goldAmount} 金币？`)) return
    setLoading(true)
    try {
      const result = await adminApi.addAccountGold(accountId, goldAmount)
      showMsg(`✅ 赠送成功，账户金币余额: ${result.gold}`)
    } catch (e: any) {
      showMsg(`❌ 失败: ${e.message}`)
    } finally {
      setLoading(false)
    }
  }

  const handleAddCityGold = async () => {
    if (!playerId || goldAmount <= 0) return
    if (!confirm(`确认给玩家 ${playerId} 补发 ${goldAmount} 城金？`)) return
    setLoading(true)
    try {
      await adminApi.addCityGold(playerId, goldAmount)
      showMsg(`✅ 城金补发成功`)
    } catch (e: any) {
      showMsg(`❌ 失败: ${e.message}`)
    } finally {
      setLoading(false)
    }
  }

  const handleQuickResources = async () => {
    if (!playerId) return
    if (!confirm(`确认给玩家 ${playerId} 补发 30000 全资源？`)) return
    setLoading(true)
    try {
      await adminApi.adjustResources(playerId, { wood: 30000, stone: 30000, iron: 30000, food: 30000 })
      showMsg(`✅ 资源补发成功（各 +30000）`)
    } catch (e: any) {
      showMsg(`❌ 失败: ${e.message}`)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] shadow-[var(--shadow-panel)] p-4">
      <div className="flex items-center gap-2 mb-3">
        <Package size={16} className="text-[var(--color-accent)]" />
        <h2 className="text-base font-bold text-[var(--color-text-primary)]">玩家操作</h2>
      </div>

      {/* ID inputs */}
      <div className="grid gap-2 mb-3">
        <input
          type="text"
          value={accountId}
          onChange={(e) => setAccountId(e.target.value)}
          placeholder="账户 ID (acc_xxx)"
          className="px-3 py-2 rounded-xl text-xs border border-[var(--color-border)] bg-[var(--color-surface-dim)] text-[var(--color-text-primary)] placeholder-[var(--color-text-muted)] outline-none focus:border-[var(--color-accent-border)]"
        />
        <PlayerSelector
          value={playerId}
          onChange={(pid, aid) => { setPlayerId(pid); setAccountId(aid) }}
          placeholder="选择玩家存档"
        />
        <div className="flex items-center gap-2">
          <span className="text-[10px] text-[var(--color-text-muted)]">数量</span>
          <input
            type="number"
            value={goldAmount}
            onChange={(e) => setGoldAmount(parseInt(e.target.value) || 0)}
            className="flex-1 px-3 py-2 rounded-xl text-xs border border-[var(--color-border)] bg-[var(--color-surface-dim)] text-[var(--color-text-primary)] outline-none focus:border-[var(--color-accent-border)]"
          />
        </div>
      </div>

      {/* Action buttons */}
      <div className="grid gap-2">
        <button
          type="button"
          onClick={handleAddAccountGold}
          disabled={loading || !accountId || goldAmount <= 0}
          className="flex items-center gap-2 px-3 py-2.5 rounded-xl text-left border border-amber-500/30 bg-amber-500/5 hover:bg-amber-500/10 cursor-pointer transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <Coins size={14} className="text-amber-500" />
          <div>
            <strong className="text-sm text-amber-600">赠送金币</strong>
            <span className="text-[10px] text-[var(--color-text-muted)] ml-2">给账户加充值金币</span>
          </div>
        </button>

        <button
          type="button"
          onClick={handleAddCityGold}
          disabled={loading || !playerId || goldAmount <= 0}
          className="flex items-center gap-2 px-3 py-2.5 rounded-xl text-left border border-blue-500/30 bg-blue-500/5 hover:bg-blue-500/10 cursor-pointer transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <Gift size={14} className="text-blue-500" />
          <div>
            <strong className="text-sm text-blue-600">补发城金</strong>
            <span className="text-[10px] text-[var(--color-text-muted)] ml-2">给玩家存档加城金</span>
          </div>
        </button>

        <button
          type="button"
          onClick={handleQuickResources}
          disabled={loading || !playerId}
          className="flex items-center gap-2 px-3 py-2.5 rounded-xl text-left border border-emerald-500/30 bg-emerald-500/5 hover:bg-emerald-500/10 cursor-pointer transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <Package size={14} className="text-emerald-500" />
          <div>
            <strong className="text-sm text-emerald-600">一键补发资源</strong>
            <span className="text-[10px] text-[var(--color-text-muted)] ml-2">木/石/铁/粮 各 +30000</span>
          </div>
        </button>
      </div>

      {/* Feedback */}
      {message && (
        <div className="mt-3 px-3 py-2 rounded-lg bg-[var(--color-surface-dim)] text-xs text-[var(--color-text-primary)]">
          {message}
        </div>
      )}
    </div>
  )
}

export function SystemActionsPanel() {
  const systemActions = [
    { title: '发布维护公告', level: 'normal' },
    { title: '刷新配置缓存', level: 'normal' },
    { title: '冻结玩家操作', level: 'danger' },
  ]

  return (
    <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] shadow-[var(--shadow-panel)] p-4">
      <div className="flex items-center gap-2 mb-3">
        <Zap size={16} className="text-[var(--color-accent)]" />
        <h2 className="text-base font-bold text-[var(--color-text-primary)]">系统操作</h2>
      </div>
      <div className="grid gap-2">
        {systemActions.map((action) => (
          <button
            key={action.title}
            type="button"
            disabled
            className={`
              px-3 py-2.5 rounded-xl text-left text-sm font-medium
              border cursor-not-allowed opacity-60
              ${action.level === 'danger'
                ? 'border-red-500/20 bg-red-500/5 text-red-600'
                : 'border-[var(--color-border)] bg-[var(--color-surface-dim)] text-[var(--color-text-primary)]'
              }
            `}
          >
            {action.title}
          </button>
        ))}
      </div>
    </div>
  )
}
