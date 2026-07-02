// 云同步弹窗负责账号登录、注册和云端存档选择。
import { useState, useEffect, type FC } from 'react'
import { useNavigate } from 'react-router-dom'
import { Cloud, LogIn, UserPlus, ArrowLeft, Check, Trash2, LogOut } from 'lucide-react'
import { Modal } from '@/components/ui'
import { useAccountStore } from '@/store/accountStore'
import { useGameStore } from '@/store/gameStore'
import type { PlayerSummary } from '@/types/game'
import { deletionRemainingMs, formatDeletionCountdown, isPlayerDeletionDue, isPlayerDeletionPending } from '@/utils/playerDeletion'

type View = 'login' | 'register' | 'saves'

interface CloudSyncModalProps {
  open: boolean
  onClose: () => void
}

const CloudSyncModal: FC<CloudSyncModalProps> = ({ open, onClose }) => {
  const navigate = useNavigate()
  const { account, players, login, register, logout, loadPlayers, deletePlayer, restorePlayerDeletion } = useAccountStore()
  const { setActivePlayer, loadGameState } = useGameStore()

  const [view, setView] = useState<View>(account ? 'saves' : 'login')
  const [now, setNow] = useState(() => Date.now())
  const [nextDeletionRefreshAt, setNextDeletionRefreshAt] = useState(0)

  useEffect(() => {
    if (open && account) {
      loadPlayers()
    }
  }, [open, account, loadPlayers])

  useEffect(() => {
    if (!open) return
    const timer = window.setInterval(() => setNow(Date.now()), 1000)
    return () => window.clearInterval(timer)
  }, [open])

  useEffect(() => {
    if (!open || players.length === 0) return
    const current = Date.now()
    const due = players.some((player) => isPlayerDeletionDue(player, current))
    if (due) {
      if (current >= nextDeletionRefreshAt) {
        setNextDeletionRefreshAt(current + 5000)
        void loadPlayers()
      }
      return
    }

    const nextRemaining = players
      .filter(isPlayerDeletionPending)
      .map((player) => deletionRemainingMs(player, current))
      .filter((remaining) => remaining > 0)
      .sort((a, b) => a - b)[0]
    if (!Number.isFinite(nextRemaining)) return
    const timer = window.setTimeout(() => {
      void loadPlayers()
    }, nextRemaining + 500)
    return () => window.clearTimeout(timer)
  }, [open, players, loadPlayers, nextDeletionRefreshAt])

  const activeView: View = account ? 'saves' : (view === 'saves' ? 'login' : view)
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
      await loadPlayers()
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
    if (isPlayerDeletionPending(player)) return
    setActivePlayer(player.id)
    await loadGameState(player.id)
    onClose()
    navigate('/city')
  }

  // 退出当前云同步账号并回到登录视图。
  const handleLogout = () => {
    logout()
    setError('')
    setView('login')
  }

  // 申请删除云端存档，进入删除冷静期。
  const handleDeletePlayer = async (e: React.MouseEvent, player: PlayerSummary) => {
    e.stopPropagation()
    if (!confirm(`确定申请删除存档「${player.nickname}」吗？\n\n确认后会进入 1 小时冷静期，期间可以恢复。`)) return
    if (!confirm(`请再次确认：存档「${player.nickname}」将在 1 小时后删除。\n\n冷静期结束后刷新列表会自动删除。`)) return
    try {
      await deletePlayer(player.id)
    } catch {
      // silently fail
    }
  }

  const handleHardDeletePlayer = async (e: React.MouseEvent, player: PlayerSummary) => {
    e.stopPropagation()
    if (!confirm(`冷静期已结束，确定永久删除存档「${player.nickname}」吗？`)) return
    try {
      await deletePlayer(player.id)
    } catch {
      // silently fail
    }
  }

  const handleRestorePlayer = async (e: React.MouseEvent, player: PlayerSummary) => {
    e.stopPropagation()
    try {
      await restorePlayerDeletion(player.id)
    } catch {
      // silently fail
    }
  }

  const handleBack = () => {
    setError('')
    if (activeView === 'register') setView('login')
  }

  const savesFooter = activeView === 'saves' ? (
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
  ) : undefined

  return (
    <Modal open={open} onClose={onClose} title="云同步" footer={savesFooter}>
      {activeView === 'login' && (
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

      {activeView === 'register' && (
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

      {activeView === 'saves' && (
        <div className="space-y-4">
          <div className="flex items-center justify-between gap-3">
            <p className="text-sm text-[var(--color-text-secondary)]">选择存档继续游戏</p>
            <div className="flex items-center gap-2">
              <span className="hidden sm:flex items-center gap-1 text-[10px] text-green-500 font-medium">
                <Cloud size={12} />
                已同步
              </span>
              <button
                type="button"
                onClick={handleLogout}
                className="
                  flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-[10px] font-semibold
                  text-[var(--color-text-muted)] hover:text-red-500 hover:bg-red-500/10
                  cursor-pointer transition-colors
                "
              >
                <LogOut size={12} />
                退出登录
              </button>
            </div>
          </div>

          {players.length > 0 ? (
            <div className="space-y-2">
              {players.map((player) => {
                const pendingDelete = isPlayerDeletionPending(player)
                const remaining = deletionRemainingMs(player, now)
                const deletionDue = isPlayerDeletionDue(player, now)
                return (
                <div
                  key={player.id}
                  onClick={() => handleSelectSave(player)}
                  onKeyDown={(e) => { if (e.key === 'Enter') void handleSelectSave(player) }}
                  role="button"
                  tabIndex={pendingDelete ? -1 : 0}
                  className={`
                    w-full flex items-center gap-3 px-4 py-3 rounded-xl
                    bg-[var(--color-surface-dim)] border border-[var(--color-border)]
                    ${pendingDelete ? 'opacity-80 cursor-default' : 'hover:border-[var(--color-accent-border)] hover:shadow-[0_4px_12px_rgba(15,23,42,0.06)] cursor-pointer'}
                    transition-all duration-200 text-left
                  `}
                >
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-semibold text-[var(--color-text-primary)]">{player.nickname}</span>
                      <span className="text-[10px] px-1.5 py-0.5 rounded bg-[var(--color-accent-light)] text-[var(--color-accent)] font-medium">
                        {player.faction}
                      </span>
                      {pendingDelete && (
                        <span className="text-[10px] px-1.5 py-0.5 rounded bg-red-500/10 text-red-500 font-medium">
                          {deletionDue ? '冷静期已结束' : `删除倒计时 ${formatDeletionCountdown(remaining)}`}
                        </span>
                      )}
                    </div>
                    <div className="mt-1">
                      <span className="text-[11px] text-[var(--color-text-muted)]">{player.updatedAt}</span>
                    </div>
                  </div>
                  {pendingDelete ? (
                    deletionDue ? (
                      <button
                        type="button"
                        onClick={(e) => handleHardDeletePlayer(e, player)}
                        className="px-2.5 py-1.5 rounded-lg flex-shrink-0 text-[10px] font-semibold text-red-600 bg-red-500/10 hover:bg-red-500/20 cursor-pointer transition-colors"
                      >
                        删除
                      </button>
                    ) : (
                    <button
                      type="button"
                      onClick={(e) => handleRestorePlayer(e, player)}
                      className="px-2.5 py-1.5 rounded-lg flex-shrink-0 text-[10px] font-semibold text-emerald-600 bg-emerald-500/10 hover:bg-emerald-500/20 cursor-pointer transition-colors"
                    >
                      恢复
                    </button>
                    )
                  ) : (
                    <div
                      role="button"
                      tabIndex={0}
                      onClick={(e) => handleDeletePlayer(e, player)}
                      onKeyDown={(e) => { if (e.key === 'Enter') handleDeletePlayer(e as unknown as React.MouseEvent, player) }}
                      className="
                        w-7 h-7 flex items-center justify-center rounded-lg flex-shrink-0
                        text-[var(--color-text-muted)] hover:text-red-500 hover:bg-red-500/10
                        transition-colors duration-150
                      "
                      aria-label={`删除存档 ${player.nickname}`}
                    >
                      <Trash2 size={14} />
                    </div>
                  )}
                  {!pendingDelete && <Check size={16} className="text-[var(--color-text-muted)] flex-shrink-0" />}
                </div>
              )})}
            </div>
          ) : (
            <div className="flex items-center justify-center py-6">
              <span className="text-sm text-[var(--color-text-muted)]">暂无云端存档，创建新角色后自动同步</span>
            </div>
          )}
        </div>
      )}
    </Modal>
  )
}

export default CloudSyncModal
