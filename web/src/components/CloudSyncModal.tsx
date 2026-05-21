import { useEffect, useState, type FC, type FormEvent } from 'react'
import { Cloud, LogIn, UserPlus, ArrowLeft, Check } from 'lucide-react'
import { gameApi } from '@/api/game'
import { Modal } from '@/components/ui'
import type { AccountSession, PlayerSummary } from '@/types/game'

type View = 'login' | 'register' | 'saves'

interface CloudSyncModalProps {
  open: boolean
  onClose: () => void
  onAccountReady: (account: AccountSession) => void
  onPlayerSelected: (playerId: string) => void
}

const CloudSyncModal: FC<CloudSyncModalProps> = ({ open, onClose, onAccountReady, onPlayerSelected }) => {
  const [account, setAccount] = useState<AccountSession | null>(() => {
    const accountId = localStorage.getItem('hero3_account_id')
    const storedUsername = localStorage.getItem('hero3_account_name')
    return accountId && storedUsername ? { accountId, username: storedUsername } : null
  })
  const [view, setView] = useState<View>(() => (account ? 'saves' : 'login'))
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [players, setPlayers] = useState<PlayerSummary[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!open || !account) return

    gameApi.listAccountPlayers(account.accountId).then((result) => {
      setPlayers(result.players)
    }).catch(() => {
      setPlayers([])
    })
  }, [account, open])

  const loadPlayers = async (accountId: string) => {
    const result = await gameApi.listAccountPlayers(accountId)
    setPlayers(result.players)
  }

  const persistAccount = async (nextAccount: AccountSession) => {
    localStorage.setItem('hero3_account_id', nextAccount.accountId)
    localStorage.setItem('hero3_account_name', nextAccount.username)
    setAccount(nextAccount)
    onAccountReady(nextAccount)
    await loadPlayers(nextAccount.accountId)
    setView('saves')
  }

  const handleLogin = async (e: FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError(null)
    try {
      await persistAccount(await gameApi.loginAccount(username, password))
    } catch (loginError) {
      setError(loginError instanceof Error ? loginError.message : '登录失败')
    } finally {
      setLoading(false)
    }
  }

  const handleRegister = async (e: FormEvent) => {
    e.preventDefault()
    if (password !== confirmPassword) {
      setError('两次输入的密码不一致')
      return
    }

    setLoading(true)
    setError(null)
    try {
      await persistAccount(await gameApi.registerAccount(username, password))
    } catch (registerError) {
      setError(registerError instanceof Error ? registerError.message : '注册失败')
    } finally {
      setLoading(false)
    }
  }

  const handleSelectSave = (playerId: string) => {
    localStorage.setItem('hero3_active_player_id', playerId)
    onPlayerSelected(playerId)
    onClose()
  }

  const handleBack = () => {
    if (view === 'register') {
      setView('login')
    }
  }

  return (
    <Modal open={open} onClose={onClose} title="云同步">
      {view === 'login' && (
        <div className="space-y-4">
          <p className="text-sm text-[var(--color-text-secondary)]">
            登录账号同步游戏存档，多设备畅玩。
          </p>

          <form onSubmit={handleLogin} className="space-y-3">
            <div>
              <label className="block text-xs font-medium text-[var(--color-text-secondary)] mb-1">账号</label>
              <input
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="请输入账号"
                className="
                  w-full px-3 py-2.5 rounded-xl text-sm
                  bg-[var(--color-surface-dim)] border border-[var(--color-border)]
                  text-[var(--color-text-primary)] placeholder-[var(--color-text-muted)]
                  outline-none focus:border-[var(--color-accent-border)]
                  transition-colors duration-200
                "
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-[var(--color-text-secondary)] mb-1">密码</label>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="请输入密码"
                className="
                  w-full px-3 py-2.5 rounded-xl text-sm
                  bg-[var(--color-surface-dim)] border border-[var(--color-border)]
                  text-[var(--color-text-primary)] placeholder-[var(--color-text-muted)]
                  outline-none focus:border-[var(--color-accent-border)]
                  transition-colors duration-200
                "
              />
            </div>

            <button
              type="submit"
              disabled={!username || !password || loading}
              className={`
                w-full flex items-center justify-center gap-2 px-4 py-2.5 rounded-xl text-sm font-semibold
                transition-all duration-200
                ${username && password && !loading
                  ? 'bg-[var(--color-accent)] text-white cursor-pointer hover:-translate-y-0.5'
                  : 'bg-[var(--color-surface-dim)] text-[var(--color-text-muted)] cursor-not-allowed'
                }
              `}
            >
              <LogIn size={14} />
              {loading ? '处理中...' : '登录'}
            </button>
          </form>

          <div className="flex items-center gap-3 pt-2">
            <div className="flex-1 h-px bg-[var(--color-border)]" />
            <span className="text-[10px] text-[var(--color-text-muted)]">或</span>
            <div className="flex-1 h-px bg-[var(--color-border)]" />
          </div>

          <button
            type="button"
            onClick={() => setView('register')}
            className="
              w-full flex items-center justify-center gap-2 px-4 py-2.5 rounded-xl text-sm font-medium
              bg-[var(--color-surface-dim)] border border-[var(--color-border)]
              text-[var(--color-text-secondary)]
              hover:border-[var(--color-text-muted)] hover:text-[var(--color-text-primary)]
              cursor-pointer transition-all duration-200
            "
          >
            <UserPlus size={14} />
            注册新账号
          </button>
        </div>
      )}

      {view === 'register' && (
        <div className="space-y-4">
          <button
            type="button"
            onClick={handleBack}
            className="flex items-center gap-1 text-xs text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] cursor-pointer transition-colors"
          >
            <ArrowLeft size={12} />
            返回登录
          </button>

          <p className="text-sm text-[var(--color-text-secondary)]">
            创建账号，开始云同步之旅。
          </p>

          <form onSubmit={handleRegister} className="space-y-3">
            <div>
              <label className="block text-xs font-medium text-[var(--color-text-secondary)] mb-1">账号</label>
              <input
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="请输入账号"
                className="
                  w-full px-3 py-2.5 rounded-xl text-sm
                  bg-[var(--color-surface-dim)] border border-[var(--color-border)]
                  text-[var(--color-text-primary)] placeholder-[var(--color-text-muted)]
                  outline-none focus:border-[var(--color-accent-border)]
                  transition-colors duration-200
                "
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-[var(--color-text-secondary)] mb-1">密码</label>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="请输入密码"
                className="
                  w-full px-3 py-2.5 rounded-xl text-sm
                  bg-[var(--color-surface-dim)] border border-[var(--color-border)]
                  text-[var(--color-text-primary)] placeholder-[var(--color-text-muted)]
                  outline-none focus:border-[var(--color-accent-border)]
                  transition-colors duration-200
                "
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-[var(--color-text-secondary)] mb-1">确认密码</label>
              <input
                type="password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                placeholder="再次输入密码"
                className="
                  w-full px-3 py-2.5 rounded-xl text-sm
                  bg-[var(--color-surface-dim)] border border-[var(--color-border)]
                  text-[var(--color-text-primary)] placeholder-[var(--color-text-muted)]
                  outline-none focus:border-[var(--color-accent-border)]
                  transition-colors duration-200
                "
              />
            </div>

            <button
              type="submit"
              disabled={!username || !password || password !== confirmPassword || loading}
              className={`
                w-full flex items-center justify-center gap-2 px-4 py-2.5 rounded-xl text-sm font-semibold
                transition-all duration-200
                ${username && password && password === confirmPassword && !loading
                  ? 'bg-[var(--color-accent)] text-white cursor-pointer hover:-translate-y-0.5'
                  : 'bg-[var(--color-surface-dim)] text-[var(--color-text-muted)] cursor-not-allowed'
                }
              `}
            >
              <UserPlus size={14} />
              {loading ? '处理中...' : '注册'}
            </button>
          </form>
        </div>
      )}

      {view === 'saves' && (
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <p className="text-sm text-[var(--color-text-secondary)]">
              {account?.username} 的游戏存档
            </p>
            <span className="flex items-center gap-1 text-[10px] text-green-500 font-medium">
              <Cloud size={12} />
              已同步
            </span>
          </div>

          {error && <p className="text-xs text-red-500">{error}</p>}

          <div className="space-y-2">
            {players.length === 0 && (
              <div className="rounded-xl bg-[var(--color-surface-dim)] px-4 py-3 text-sm text-[var(--color-text-muted)]">
                暂无绑定存档，关闭弹窗后创建新游戏即可自动绑定当前账号。
              </div>
            )}
            {players.map((save) => (
              <button
                key={save.id}
                type="button"
                onClick={() => handleSelectSave(save.id)}
                className="
                  w-full flex items-center gap-3 px-4 py-3 rounded-xl
                  bg-[var(--color-surface-dim)] border border-[var(--color-border)]
                  hover:border-[var(--color-accent-border)] hover:shadow-[0_4px_12px_rgba(15,23,42,0.06)]
                  cursor-pointer transition-all duration-200 text-left
                "
              >
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-semibold text-[var(--color-text-primary)]">{save.nickname}</span>
                    <span className="text-[10px] px-1.5 py-0.5 rounded bg-[var(--color-accent-light)] text-[var(--color-accent)] font-medium">
                      {save.faction}
                    </span>
                  </div>
                  <div className="flex items-center gap-3 mt-1">
                    <span className="text-[11px] text-[var(--color-text-muted)]">{save.id}</span>
                    <span className="text-[11px] text-[var(--color-text-muted)]">{save.updatedAt}</span>
                  </div>
                </div>
                <Check size={16} className="text-[var(--color-text-muted)] flex-shrink-0" />
              </button>
            ))}
          </div>

          <button
            type="button"
            onClick={onClose}
            className="
              w-full px-4 py-2.5 rounded-xl text-sm font-medium
              bg-[var(--color-surface-dim)] border border-[var(--color-border)]
              text-[var(--color-text-secondary)]
              hover:border-[var(--color-text-muted)]
              cursor-pointer transition-all duration-200
            "
          >
            创建新存档
          </button>
        </div>
      )}
    </Modal>
  )
}

export default CloudSyncModal
