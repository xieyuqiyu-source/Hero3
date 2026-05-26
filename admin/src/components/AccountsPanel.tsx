import { useState } from 'react'
import { Users, Trash2, Eye, Coins, Gift, ChevronDown, ChevronRight } from 'lucide-react'
import PlayerDrawer from '@/components/PlayerDrawer'
import { adminApi } from '@/api/admin'
import type { AccountSummary, PlayerSummary } from '@/types'

interface AccountsPanelProps {
  accounts: AccountSummary[]
  busyTarget: string | null
  onDeleteAccount: (account: AccountSummary) => Promise<void>
  onDeletePlayer: (player: PlayerSummary) => Promise<void>
}

export default function AccountsPanel({
  accounts,
  busyTarget,
  onDeleteAccount,
  onDeletePlayer,
}: AccountsPanelProps) {
  const [selectedPlayerId, setSelectedPlayerId] = useState<string | null>(null)
  const [expandedAccount, setExpandedAccount] = useState<string | null>(null)
  const [goldInput, setGoldInput] = useState<Record<string, number>>({})
  const [cityGoldInput, setCityGoldInput] = useState<Record<string, number>>({})
  const [feedback, setFeedback] = useState('')

  const showFeedback = (msg: string) => {
    setFeedback(msg)
    setTimeout(() => setFeedback(''), 3000)
  }

  const handleAddAccountGold = async (accountId: string) => {
    const amount = goldInput[accountId] || 10
    if (!confirm(`确认给账户赠送 ${amount} 金币？`)) return
    try {
      const result = await adminApi.addAccountGold(accountId, amount)
      showFeedback(`✅ 赠送成功，余额: ${result.gold}`)
    } catch (e: any) {
      showFeedback(`❌ ${e.message}`)
    }
  }

  const handleAddCityGold = async (playerId: string) => {
    const amount = cityGoldInput[playerId] || 100
    if (!confirm(`确认补发 ${amount} 城金？`)) return
    try {
      await adminApi.addCityGold(playerId, amount)
      showFeedback(`✅ 城金补发成功`)
    } catch (e: any) {
      showFeedback(`❌ ${e.message}`)
    }
  }

  return (
    <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] shadow-[var(--shadow-panel)] p-4">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <Users size={16} className="text-[var(--color-accent)]" />
          <h2 className="text-base font-bold text-[var(--color-text-primary)]">注册玩家</h2>
        </div>
        <span className="px-2.5 py-1 rounded-full text-xs font-bold text-amber-700 bg-[var(--color-gold-soft)]">
          {accounts.length} 个账号
        </span>
      </div>

      {/* Feedback */}
      {feedback && (
        <div className="mb-3 px-3 py-2 rounded-lg bg-[var(--color-surface-dim)] text-xs text-[var(--color-text-primary)]">
          {feedback}
        </div>
      )}

      <div className="flex gap-4 max-lg:flex-col">
        <div className="flex-1 min-w-0">
          {accounts.length === 0 ? (
            <div className="py-8 text-center rounded-2xl border border-dashed border-[var(--color-border)] bg-[var(--color-surface-dim)]">
              <p className="text-sm text-[var(--color-text-secondary)]">暂无注册账号</p>
              <p className="text-xs text-[var(--color-text-muted)] mt-1">玩家完成注册后会显示在这里</p>
            </div>
          ) : (
            <div className="grid gap-3">
              {accounts.map((account) => {
                const isExpanded = expandedAccount === account.id
                return (
                  <div key={account.id} className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] overflow-hidden">
                    {/* Account header - clickable to expand */}
                    <div
                      className="flex items-center justify-between p-3 cursor-pointer hover:bg-white/50 dark:hover:bg-white/5 transition-colors"
                      onClick={() => setExpandedAccount(isExpanded ? null : account.id)}
                    >
                      <div className="flex items-center gap-2">
                        {isExpanded ? <ChevronDown size={12} className="text-[var(--color-text-muted)]" /> : <ChevronRight size={12} className="text-[var(--color-text-muted)]" />}
                        <div>
                          <strong className="text-sm text-[var(--color-text-primary)]">{account.username}</strong>
                          <span className="block text-[11px] text-[var(--color-text-muted)]">{account.id}</span>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <span className="text-xs text-[var(--color-accent)] font-bold">{account.players.length} 存档</span>
                        {/* Gold gift button */}
                        <button
                          type="button"
                          onClick={(e) => { e.stopPropagation(); handleAddAccountGold(account.id) }}
                          className="flex items-center gap-1 px-2 py-1 rounded-lg text-[11px] font-bold text-amber-600 bg-amber-500/8 border border-amber-500/20 hover:bg-amber-500/15 cursor-pointer transition-colors"
                          title="赠送金币"
                        >
                          <Coins size={11} />
                          赠金币
                        </button>
                        <button
                          type="button"
                          disabled={busyTarget !== null}
                          onClick={(e) => { e.stopPropagation(); void onDeleteAccount(account) }}
                          className="flex items-center gap-1 px-2 py-1 rounded-lg text-[11px] font-bold text-red-600 bg-red-500/8 border border-red-500/20 hover:bg-red-500/15 cursor-pointer transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                        >
                          <Trash2 size={11} />
                          {busyTarget === `account:${account.id}` ? '删除中' : '删除'}
                        </button>
                      </div>
                    </div>

                    {/* Expanded content */}
                    {isExpanded && (
                      <div className="px-3 pb-3 border-t border-[var(--color-border)]">
                        {/* Gold amount input */}
                        <div className="flex items-center gap-2 py-2">
                          <span className="text-[10px] text-[var(--color-text-muted)]">赠送数量:</span>
                          <input
                            type="number"
                            value={goldInput[account.id] ?? 10}
                            onChange={(e) => setGoldInput({ ...goldInput, [account.id]: parseInt(e.target.value) || 0 })}
                            onClick={(e) => e.stopPropagation()}
                            className="w-20 px-2 py-1 rounded-lg text-xs border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)] outline-none"
                          />
                          <span className="text-[10px] text-[var(--color-text-muted)]">金币</span>
                        </div>

                        {/* Players list */}
                        {account.players.length > 0 && (
                          <div className="grid gap-1.5 mt-1">
                            {account.players.map((player) => (
                              <div
                                key={player.id}
                                className="flex items-center justify-between px-2.5 py-2 rounded-lg bg-white/70 dark:bg-white/5 border border-[var(--color-border)]"
                              >
                                <div>
                                  <strong className="text-xs text-[var(--color-text-primary)]">{player.nickname}</strong>
                                  <span className="ml-2 text-[10px] text-[var(--color-text-muted)]">{player.faction}</span>
                                </div>
                                <div className="flex items-center gap-1.5">
                                  {/* City gold input + button */}
                                  <input
                                    type="number"
                                    value={cityGoldInput[player.id] ?? 100}
                                    onChange={(e) => setCityGoldInput({ ...cityGoldInput, [player.id]: parseInt(e.target.value) || 0 })}
                                    onClick={(e) => e.stopPropagation()}
                                    className="w-14 px-1.5 py-1 rounded text-[10px] border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)] outline-none text-center"
                                  />
                                  <button
                                    type="button"
                                    onClick={(e) => { e.stopPropagation(); handleAddCityGold(player.id) }}
                                    className="flex items-center gap-0.5 px-2 py-1 rounded-lg text-[10px] font-bold text-blue-600 bg-blue-500/8 border border-blue-500/20 hover:bg-blue-500/15 cursor-pointer transition-colors"
                                    title="补发城金"
                                  >
                                    <Gift size={10} />
                                    城金
                                  </button>
                                  <button
                                    type="button"
                                    onClick={(e) => { e.stopPropagation(); setSelectedPlayerId(player.id) }}
                                    className="flex items-center gap-1 px-2 py-1 rounded-lg text-[11px] font-bold text-[var(--color-accent)] bg-[var(--color-accent-light)] border border-[var(--color-accent-border)] hover:bg-[var(--color-accent)]/15 cursor-pointer transition-colors"
                                  >
                                    <Eye size={11} />
                                    详情
                                  </button>
                                  <button
                                    type="button"
                                    disabled={busyTarget !== null}
                                    onClick={(e) => { e.stopPropagation(); void onDeletePlayer(player) }}
                                    className="px-2 py-1 rounded-lg text-[11px] font-bold text-red-600 bg-red-500/8 border border-red-500/20 hover:bg-red-500/15 cursor-pointer transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                                  >
                                    {busyTarget === `player:${player.id}` ? '...' : '删除'}
                                  </button>
                                </div>
                              </div>
                            ))}
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          )}
        </div>

        {selectedPlayerId && (
          <PlayerDrawer
            playerId={selectedPlayerId}
            onClose={() => setSelectedPlayerId(null)}
          />
        )}
      </div>
    </div>
  )
}
