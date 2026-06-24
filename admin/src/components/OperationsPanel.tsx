import { useEffect, useState } from 'react'
import { Package, Zap, Coins, Gift, RefreshCcw, Repeat2 } from 'lucide-react'
import { adminApi } from '@/api/admin'
import PlayerSelector from './PlayerSelector'
import type { ItemDefinition } from '@/types'

export function ResourceToolsPanel() {
  const [playerId, setPlayerId] = useState('')
  const [accountId, setAccountId] = useState('')
  const [goldAmount, setGoldAmount] = useState(10)
  const [items, setItems] = useState<Record<string, ItemDefinition>>({})
  const [itemId, setItemId] = useState('')
  const [itemAmount, setItemAmount] = useState(1)
  const [factionsConfig, setFactionsConfig] = useState<Record<string, any>>({})
  const [playerFaction, setPlayerFaction] = useState('')
  const [currentGeneralId, setCurrentGeneralId] = useState('')
  const [generalId, setGeneralId] = useState('')
  const [message, setMessage] = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    void Promise.all([adminApi.getItemsConfig(), adminApi.getFactionsConfig()])
      .then(([itemResult, factionResult]) => {
        setItems(itemResult)
        setFactionsConfig(factionResult as Record<string, any>)
        const first = Object.keys(itemResult)[0] ?? ''
        setItemId((current) => current || first)
      })
      .catch(() => {
        setItems({})
        setFactionsConfig({})
      })
  }, [])

  const generalOptions = playerFaction
    ? ((factionsConfig[playerFaction]?.generals ?? []) as Array<{ id: string; name: string }>).filter((item) => item.id !== currentGeneralId)
    : []

  const loadPlayerMeta = async (pid: string) => {
    if (!pid) {
      setPlayerFaction('')
      setCurrentGeneralId('')
      setGeneralId('')
      return
    }
    try {
      const state = await adminApi.getPlayerState(pid)
      const faction = state.player.faction
      const current = state.general?.id ?? ''
      setPlayerFaction(faction)
      setCurrentGeneralId(current)
      const generals = ((factionsConfig[faction]?.generals ?? []) as Array<{ id: string; name: string }>)
      setGeneralId(generals.find((item) => item.id !== current)?.id || '')
    } catch {
      setPlayerFaction('')
      setCurrentGeneralId('')
      setGeneralId('')
    }
  }

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

  const handleGrantItem = async () => {
    if (!playerId || !itemId || itemAmount <= 0) return
    const itemName = items[itemId]?.name ?? itemId
    if (!confirm(`确认给玩家 ${playerId} 发放 ${itemName} x${itemAmount}？`)) return
    setLoading(true)
    try {
      await adminApi.grantItem(playerId, itemId, itemAmount)
      showMsg(`✅ 物品发放成功：${itemName} x${itemAmount}`)
    } catch (e: any) {
      showMsg(`❌ 失败: ${e.message}`)
    } finally {
      setLoading(false)
    }
  }

  const handleResetGeneralStats = async () => {
    if (!playerId) return
    if (!confirm(`确认给玩家 ${playerId} 洗点？将消耗该账号 10 金币。`)) return
    setLoading(true)
    try {
      const result = await adminApi.resetGeneralStats(playerId)
      showMsg(`✅ 洗点成功，账户金币余额: ${result.accountGold}`)
    } catch (e: any) {
      showMsg(`❌ 失败: ${e.message}`)
    } finally {
      setLoading(false)
    }
  }

  const handleChangeGeneral = async () => {
    if (!playerId || !generalId) return
    const name = generalOptions.find((item) => item.id === generalId)?.name ?? generalId
    if (!confirm(`确认将玩家 ${playerId} 的将领更换为 ${name}？等级经验保留，四维加点重置。`)) return
    setLoading(true)
    try {
      await adminApi.changeGeneral(playerId, generalId)
      showMsg(`✅ 换将成功：${name}`)
      await loadPlayerMeta(playerId)
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
          onChange={(pid, aid) => { setPlayerId(pid); setAccountId(aid); void loadPlayerMeta(pid) }}
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

        <div className="rounded-xl border border-violet-500/30 bg-violet-500/5 p-3">
          <div className="mb-2 flex items-center gap-2">
            <Gift size={14} className="text-violet-500" />
            <strong className="text-sm text-violet-600">发放物品</strong>
          </div>
          <div className="grid gap-2 sm:grid-cols-[1fr_96px_auto]">
            <select
              value={itemId}
              onChange={(e) => setItemId(e.target.value)}
              className="min-w-0 px-3 py-2 rounded-xl text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)] outline-none focus:border-[var(--color-accent-border)]"
            >
              {Object.entries(items).map(([id, item]) => (
                <option key={id} value={id}>{item.name}</option>
              ))}
            </select>
            <input
              type="number"
              min={1}
              value={itemAmount}
              onChange={(e) => setItemAmount(parseInt(e.target.value) || 0)}
              className="px-3 py-2 rounded-xl text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)] outline-none focus:border-[var(--color-accent-border)]"
            />
            <button
              type="button"
              onClick={handleGrantItem}
              disabled={loading || !playerId || !itemId || itemAmount <= 0}
              className="px-3 py-2 rounded-xl text-xs font-bold text-violet-600 border border-violet-500/30 bg-violet-500/10 hover:bg-violet-500/15 cursor-pointer transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              发放
            </button>
          </div>
        </div>

        <div className="rounded-xl border border-amber-500/30 bg-amber-500/5 p-3">
          <div className="mb-2 flex items-center gap-2">
            <Repeat2 size={14} className="text-amber-500" />
            <strong className="text-sm text-amber-600">将领操作</strong>
            {playerFaction && <span className="ml-auto text-[10px] text-[var(--color-text-muted)]">{playerFaction}</span>}
          </div>
          <div className="grid gap-2 sm:grid-cols-[auto_1fr_auto]">
            <button
              type="button"
              onClick={handleResetGeneralStats}
              disabled={loading || !playerId}
              className="inline-flex items-center justify-center gap-1.5 rounded-xl border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-xs font-bold text-amber-600 transition-colors hover:bg-amber-500/15 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <RefreshCcw size={13} />
              洗点
            </button>
            <select
              value={generalId}
              onChange={(e) => setGeneralId(e.target.value)}
              disabled={loading || !playerId || generalOptions.length === 0}
              className="min-w-0 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-xs text-[var(--color-text-primary)] outline-none disabled:opacity-50"
            >
              {generalOptions.length === 0 ? (
                <option value="">先选择玩家</option>
              ) : (
                generalOptions.map((general) => <option key={general.id} value={general.id}>{general.name}</option>)
              )}
            </select>
            <button
              type="button"
              onClick={handleChangeGeneral}
              disabled={loading || !playerId || !generalId}
              className="inline-flex items-center justify-center gap-1.5 rounded-xl border border-blue-500/30 bg-blue-500/10 px-3 py-2 text-xs font-bold text-blue-600 transition-colors hover:bg-blue-500/15 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <Repeat2 size={13} />
              换将
            </button>
          </div>
          <p className="mt-2 text-[10px] text-[var(--color-text-muted)]">换将限制为玩家当前阵营，保留等级经验并重置四维加点。</p>
        </div>
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
