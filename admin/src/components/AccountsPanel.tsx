import { useState } from 'react'
import { Users, Trash2, Eye } from 'lucide-react'
import PlayerDetailPanel from '@/components/PlayerDetailPanel'
import type { AccountSummary, GameState, PlayerSummary } from '@/types'

interface AccountsPanelProps {
  accounts: AccountSummary[]
  busyTarget: string | null
  gameState: GameState | null
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

      {accounts.length === 0 ? (
        <div className="py-8 text-center rounded-2xl border border-dashed border-[var(--color-border)] bg-[var(--color-surface-dim)]">
          <p className="text-sm text-[var(--color-text-secondary)]">暂无注册账号</p>
          <p className="text-xs text-[var(--color-text-muted)] mt-1">玩家完成注册后会显示在这里</p>
        </div>
      ) : (
        <div className="grid gap-3">
          {accounts.map((account) => (
            <div key={account.id} className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-3">
              <div className="flex items-center justify-between mb-2">
                <div>
                  <strong className="text-sm text-[var(--color-text-primary)]">{account.username}</strong>
                  <span className="block text-[11px] text-[var(--color-text-muted)]">{account.id}</span>
                </div>
                <div className="flex items-center gap-2">
                  <span className="text-xs text-[var(--color-accent)] font-bold">{account.players.length} 存档</span>
                  <button
                    type="button"
                    disabled={busyTarget !== null}
                    onClick={() => void onDeleteAccount(account)}
                    className="
                      flex items-center gap-1 px-2 py-1 rounded-lg text-[11px] font-bold
                      text-red-600 bg-red-500/8 border border-red-500/20
                      hover:bg-red-500/15 cursor-pointer transition-colors
                      disabled:opacity-50 disabled:cursor-not-allowed
                    "
                  >
                    <Trash2 size={11} />
                    {busyTarget === `account:${account.id}` ? '删除中' : '删除'}
                  </button>
                </div>
              </div>

              {account.players.length > 0 && (
                <div className="grid gap-1.5">
                  {account.players.map((player) => (
                    <div key={player.id} className="flex items-center justify-between px-2.5 py-2 rounded-lg bg-white/70 dark:bg-white/5 border border-[var(--color-border)]">
                      <div>
                        <strong className="text-xs text-[var(--color-text-primary)]">{player.nickname}</strong>
                        <span className="ml-2 text-[10px] text-[var(--color-text-muted)]">{player.faction}</span>
                      </div>
                      <div className="flex items-center gap-1.5">
                        <button
                          type="button"
                          onClick={() => setSelectedPlayerId(player.id)}
                          className="
                            flex items-center gap-1 px-2 py-1 rounded-lg text-[11px] font-bold
                            text-[var(--color-accent)] bg-[var(--color-accent-light)] border border-[var(--color-accent-border)]
                            hover:bg-[var(--color-accent)]/15 cursor-pointer transition-colors
                          "
                        >
                          <Eye size={11} />
                          详情
                        </button>
                        <button
                          type="button"
                          disabled={busyTarget !== null}
                          onClick={() => void onDeletePlayer(player)}
                          className="
                            px-2 py-1 rounded-lg text-[11px] font-bold
                            text-red-600 bg-red-500/8 border border-red-500/20
                            hover:bg-red-500/15 cursor-pointer transition-colors
                            disabled:opacity-50 disabled:cursor-not-allowed
                          "
                        >
                          {busyTarget === `player:${player.id}` ? '...' : '删除'}
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {selectedPlayerId && (
        <PlayerDetailPanel
          playerId={selectedPlayerId}
          onClose={() => setSelectedPlayerId(null)}
        />
      )}
    </div>
  )
}
