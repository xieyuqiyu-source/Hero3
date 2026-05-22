import { useState } from 'react'
import PlayerDetailPanel from '@/components/PlayerDetailPanel'
import PlayerStatePanel from '@/components/PlayerStatePanel'
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
  gameState,
  onDeleteAccount,
  onDeletePlayer,
}: AccountsPanelProps) {
  const [selectedPlayerId, setSelectedPlayerId] = useState<string | null>(null)

  return (
    <article className="panel player-panel" id="玩家">
      <div className="panel-heading">
        <div>
          <p className="eyebrow">Accounts</p>
          <h2>注册玩家与存档</h2>
        </div>
        <span className="panel-count">{accounts.length} 个账号</span>
      </div>

      {accounts.length === 0 ? (
        <div className="empty-state">
          <strong>暂无注册账号</strong>
          <span>玩家完成注册并创建存档后会显示在这里。</span>
        </div>
      ) : (
        <div className="account-list">
          {accounts.map((account) => (
            <section className="account-card" key={account.id}>
              <div className="account-card-header">
                <div>
                  <strong>{account.username}</strong>
                  <span>{account.id}</span>
                </div>
                <div className="account-actions">
                  <small>{account.players.length} 个存档</small>
                  <button
                    type="button"
                    className="danger-text-button"
                    disabled={busyTarget !== null}
                    onClick={() => void onDeleteAccount(account)}
                  >
                    {busyTarget === `account:${account.id}` ? '删除中' : '删除账号'}
                  </button>
                </div>
              </div>

              {account.players.length === 0 ? (
                <div className="save-empty">尚未创建游戏存档</div>
              ) : (
                <div className="save-list">
                  {account.players.map((player) => (
                    <div className="save-row" key={player.id}>
                      <div>
                        <strong>{player.nickname}</strong>
                        <span>{player.id}</span>
                      </div>
                      <div className="save-meta">
                        <span>{player.faction}</span>
                        <time>{new Date(player.updatedAt).toLocaleString()}</time>
                        <button
                          type="button"
                          className="ghost-button"
                          onClick={() => setSelectedPlayerId(player.id)}
                        >
                          查看详情
                        </button>
                        <button
                          type="button"
                          className="danger-text-button"
                          disabled={busyTarget !== null}
                          onClick={() => void onDeletePlayer(player)}
                        >
                          {busyTarget === `player:${player.id}` ? '删除中' : '删除存档'}
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </section>
          ))}
        </div>
      )}

      <PlayerStatePanel gameState={gameState} />

      {selectedPlayerId && (
        <PlayerDetailPanel
          playerId={selectedPlayerId}
          onClose={() => setSelectedPlayerId(null)}
        />
      )}
    </article>
  )
}
