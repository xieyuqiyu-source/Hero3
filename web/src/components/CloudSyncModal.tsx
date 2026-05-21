import { useState, type FC } from 'react'
import { useNavigate } from 'react-router-dom'
import { Cloud, LogIn, UserPlus, ArrowLeft, Check } from 'lucide-react'
import { Modal } from '@/components/ui'
import { useAccountStore } from '@/store/accountStore'
import { useGameStore } from '@/store/gameStore'
import type { PlayerSummary } from '@/types/game'

type View = 'login' | 'register' | 'saves'

interface CloudSyncModalProps {
  open: boolean
  onClose: () => void
}

const CloudSyncModal: FC<CloudSyncModalProps> = ({ open, onClose }) => {
  const navigate = useNavigate()
  const { account, players, login, register, loadPlayers } = useAccountStore()
  const { setActivePlayer, loadGameState } = useGameStore()

  const [view, setView] = useState<View>(account ? 'saves' : 'login')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      await login(username, password)
      setView('saves')
    } catch {
      setError('登录失败，请检查用户名和密码')
    } finally {
      setLoading(false)
    }
  }

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault()
    if (password !== confirmPassword) {
      setError('两次密码不一致')
      return
    }
    setError('')
    setLoading(true)
    try {
      await register(username, password)
      await loadPlayers()
      setView('saves')
    } catch {
      setError('注册失败，请稍后重试')
    } finally {
      setLoading(false)
    }
  }

  const handleSelectSave = async (player: PlayerSummary) => {
    setActivePlayer(player.id)
    await loadGameState(player.id)
    onClose()
    navigate('/city')
  }

  const handleBack = () => {
    setError('')
    if (view === 'register') setView('login')
  }

  return (
    <Modal open={open} onClose={onClose} title="云同步">
      {view === 'login' && (
        <div className="space-y-4">
          <p className="text-sm text-[var(--color-text-secondary)]">
            登录账号同步游戏存档，多设备畅玩。
          </p>

          {error && (
            <div className="px-3 py-2 rounded-xl bg-red-500/10 border border-red-500/30 text-xs text-red-400">
              {error}
            </div>
          )}

          <form onSubmit={handleLogin} className="space-y-3">
            <div>
              <label className="block text-xs font-medium text-[var(--color-text-secondary)] mb-1">用户名</label>
              <input
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="请输入用户名"
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
              {loading ? '登录中...' : '登录'}
            </button>
          </form>

          <div className="flex items-center gap-3 pt-2">
            <div className="flex-1 h-px bg-[var(--color-border)]" />
            <span className="text-[10px] text-[var(--color-text-muted)]">或</span>
            <div className="flex-1 h-px bg-[var(--color-border)]" />
          </div>

          <button
            type="button"
            onClick={() => { setView('register'); setError('') }}
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

          {error && (
            <div className="px-3 py-2 rounded-xl bg-red-500/10 border border-red-500/30 text-xs text-red-400">
              {error}
            </div>
          )}

          <form onSubmit={handleRegister} className="space-y-3">
            <div>
              <label className="block text-xs font-medium text-[var(--color-text-secondary)] mb-1">用户名</label>
              <input
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="请输入用户名"
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
              {loading ? '注册中...' : '注册'}
            </button>
          </form>
        </div>
      )}

      {view === 'saves' && (
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <p className="text-sm text-[var(--color-text-secondary)]">选择存档继续游戏</p>
            <span className="flex items-center gap-1 text-[10px] text-green-500 font-medium">
              <Cloud size={12} />
              已同步
            </span>
          </div>

          {players.length > 0 ? (
            <div className="space-y-2">
              {players.map((player) => (
                <button
                  key={player.id}
                  type="button"
                  onClick={() => handleSelectSave(player)}
                  className="
                    w-full flex items-center gap-3 px-4 py-3 rounded-xl
                    bg-[var(--color-surface-dim)] border border-[var(--color-border)]
                    hover:border-[var(--color-accent-border)] hover:shadow-[0_4px_12px_rgba(15,23,42,0.06)]
                    cursor-pointer transition-all duration-200 text-left
                  "
                >
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-semibold text-[var(--color-text-primary)]">{player.nickname}</span>
                      <span className="text-[10px] px-1.5 py-0.5 rounded bg-[var(--color-accent-light)] text-[var(--color-accent)] font-medium">
                        {player.faction}
                      </span>
                    </div>
                    <div className="mt-1">
                      <span className="text-[11px] text-[var(--color-text-muted)]">{player.updatedAt}</span>
                    </div>
                  </div>
                  <Check size={16} className="text-[var(--color-text-muted)] flex-shrink-0" />
                </button>
              ))}
            </div>
          ) : (
            <div className="flex items-center justify-center py-6">
              <span className="text-sm text-[var(--color-text-muted)]">暂无云端存档，创建新角色后自动同步</span>
            </div>
          )}

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
