import { useState, useEffect, type FC } from 'react'
import { useNavigate } from 'react-router-dom'
import { Coins, Crown, Scroll, Users, Castle, KeyRound, Sparkles, Cloud, LogOut, Check, ArrowRightLeft } from 'lucide-react'
import { useAccountStore } from '@/store/accountStore'
import { useGameStore } from '@/store/gameStore'
import { getFactionLabel } from '@/utils/faction'
import { gameApi } from '@/api/game'
import { toast } from '@/components/ui'
import CloudSyncModal from '@/components/CloudSyncModal'
import Section from './components/Section'
import InfoItem from './components/InfoItem'

const AccountPage: FC = () => {
  const navigate = useNavigate()
  const { account, players, logout, loadPlayers } = useAccountStore()
  const { state: gameState, activePlayerId, clearActivePlayer, setActivePlayer, loadGameState } = useGameStore()
  const [cloudSyncOpen, setCloudSyncOpen] = useState(false)
  const [switching, setSwitching] = useState<string | null>(null)

  // 页面加载时拉取存档列表
  useEffect(() => {
    if (account) {
      loadPlayers()
    }
  }, [account, loadPlayers])

  const handleLogout = () => {
    logout()
  }

  const handleSwitchSave = () => {
    clearActivePlayer()
    navigate('/login')
  }

  const handleSelectPlayer = async (playerId: string) => {
    if (playerId === activePlayerId || switching) return
    setSwitching(playerId)
    try {
      setActivePlayer(playerId)
      await loadGameState(playerId)
    } finally {
      setSwitching(null)
    }
  }

  const handleCloudSyncClose = () => {
    setCloudSyncOpen(false)
    loadPlayers()
  }

  return (
    <div className="space-y-5 max-w-3xl">

      {/* 账户信息 */}
      <Section title="账户信息" icon={Coins}>
        <AccountGoldSection />
      </Section>

      {/* 个人概况 */}
      <Section title="个人概况" icon={Crown}>
        <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
          <InfoItem label="昵称" value={gameState?.player.nickname ?? '--'} highlight />
          <InfoItem label="天梯排名" value="等待统计中.." />
          <InfoItem label="国度" value={getFactionLabel(gameState?.player.faction ?? '--')} />
          <InfoItem label="所属联盟" value="--" />
          <InfoItem label="城池数" value="1" />
          <InfoItem label="文明度" value="0" />
        </div>
      </Section>

      {/* 兑换称号 */}
      <Section title="兑换称号" icon={Scroll} badge="当前战功: 0">
        <div className="flex items-center justify-center py-8">
          <span className="text-sm text-[var(--color-text-muted)]">称号系统开发中，敬请期待</span>
        </div>
      </Section>

      {/* 将领信息 */}
      <Section title="将领信息" icon={Users}>
        <div className="flex items-center justify-center py-8">
          <span className="text-sm text-[var(--color-text-muted)]">将领系统开发中，敬请期待</span>
        </div>
      </Section>

      {/* 城池列表（存档列表） */}
      <Section title="城池列表" icon={Castle}>
        {players.length > 0 ? (
          <div className="space-y-2">
            {players.map((p) => {
              const isActive = p.id === activePlayerId
              return (
                <button
                  key={p.id}
                  type="button"
                  onClick={() => handleSelectPlayer(p.id)}
                  disabled={isActive || switching !== null}
                  className={`
                    w-full flex items-center justify-between px-3 py-2.5 rounded-xl
                    border text-left cursor-pointer
                    transition-all duration-200
                    ${isActive
                      ? 'bg-[var(--color-accent-light)] border-[var(--color-accent-border)]'
                      : 'bg-[var(--color-surface-dim)] border-[var(--color-border)] hover:border-[var(--color-accent-border)]'
                    }
                    disabled:cursor-default
                  `}
                >
                  <div className="flex flex-col gap-1">
                    <div className="flex items-center gap-2">
                      {isActive && <Check size={12} className="text-[var(--color-accent)]" />}
                      <span className={`text-sm font-medium ${isActive ? 'text-[var(--color-accent)]' : 'text-[var(--color-text-primary)]'}`}>
                        {p.nickname}
                      </span>
                      <span className="text-[10px] px-1.5 py-0.5 rounded bg-[var(--color-accent-light)] text-[var(--color-accent)] font-medium">
                        {getFactionLabel(p.faction)}
                      </span>
                      {isActive && (
                        <span className="text-[10px] text-[var(--color-accent)] font-medium">当前</span>
                      )}
                    </div>
                    <div className="flex items-center gap-3 text-[10px] text-[var(--color-text-muted)]">
                      <span>兵力 {p.totalArmy.toLocaleString()}</span>
                      <span>文明度 {p.buildingLevel}</span>
                    </div>
                  </div>
                  <div className="flex flex-col items-end gap-1">
                    {switching === p.id && (
                      <span className="text-[10px] text-[var(--color-text-muted)]">切换中...</span>
                    )}
                    <span className="text-[10px] text-[var(--color-text-muted)]">{p.updatedAt}</span>
                  </div>
                </button>
              )
            })}
          </div>
        ) : (
          <div className="flex items-center justify-center py-6">
            <span className="text-sm text-[var(--color-text-muted)]">
              {account ? '暂无云端存档' : '登录后可查看云端存档'}
            </span>
          </div>
        )}
      </Section>

      {/* 卡密输入 */}
      <Section title="卡密输入" icon={KeyRound}>
        <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-3">
          <input
            type="text"
            placeholder="请输入卡密"
            className="
              flex-1 px-3 py-2 rounded-xl text-sm
              bg-[var(--color-surface-dim)] border border-[var(--color-border)]
              text-[var(--color-text-primary)] placeholder-[var(--color-text-muted)]
              outline-none focus:border-[var(--color-accent-border)]
              transition-colors duration-200
            "
          />
          <button
            type="button"
            className="
              px-4 py-2 rounded-xl text-xs font-semibold
              bg-[var(--color-accent)] text-white
              hover:opacity-90 cursor-pointer transition-opacity
            "
          >
            兑换
          </button>
        </div>
        <p className="text-[11px] text-[var(--color-text-muted)] mt-2">输入卡密兑换金币或道具，未来开发内容。</p>
      </Section>

      {/* 传奇兑换 */}
      <Section title="传奇兑换" icon={Sparkles}>
        <div className="flex items-center justify-center py-8">
          <span className="text-sm text-[var(--color-text-muted)]">传奇兑换系统开发中，敬请期待</span>
        </div>
      </Section>

      {/* 云同步 / 账号 */}
      <Section title="云同步" icon={Cloud}>
        {account ? (
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <span className="w-2 h-2 rounded-full bg-green-500" />
                <span className="text-sm text-[var(--color-text-primary)]">{account.username}</span>
                <span className="text-[10px] text-green-500 font-medium">已同步</span>
              </div>
              <button
                type="button"
                onClick={handleLogout}
                className="
                  flex items-center gap-1 px-3 py-1.5 rounded-lg text-xs font-medium
                  text-[var(--color-text-muted)] hover:text-red-500
                  cursor-pointer transition-colors
                "
              >
                <LogOut size={12} />
                退出登录
              </button>
            </div>
            <button
              type="button"
              onClick={handleSwitchSave}
              className="
                w-full px-4 py-2 rounded-xl text-xs font-medium
                bg-[var(--color-surface-dim)] border border-[var(--color-border)]
                text-[var(--color-text-secondary)]
                hover:border-[var(--color-text-muted)] hover:text-[var(--color-text-primary)]
                cursor-pointer transition-all duration-200
              "
            >
              切换存档 / 返回选择界面
            </button>
          </div>
        ) : (
          <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
            <span className="text-sm text-[var(--color-text-secondary)]">未登录，存档仅保存在本地</span>
            <button
              type="button"
              onClick={() => setCloudSyncOpen(true)}
              className="
                flex items-center gap-1.5 px-4 py-1.5 rounded-xl text-xs font-medium
                bg-green-600 text-white
                hover:bg-green-500 cursor-pointer transition-colors
              "
            >
              <Cloud size={13} />
              登录同步
            </button>
          </div>
        )}
      </Section>

      <CloudSyncModal open={cloudSyncOpen} onClose={handleCloudSyncClose} />
    </div>
  )
}

// --- Account Gold Section (inline button + expand panel) ---
const AccountGoldSection: FC = () => {
  const [expanded, setExpanded] = useState<'to_city' | 'to_account' | null>(null)
  const [amount, setAmount] = useState(1)
  const [loading, setLoading] = useState(false)
  const account = useAccountStore((s) => s.account)
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const setState = useGameStore((s) => s.setState)
  const gameState = useGameStore((s) => s.state)

  const cityName = gameState?.player.nickname ?? '当前城池'

  const cooldownRemaining = (() => {
    if (!gameState?.lastExchangeAt) return 0
    const last = new Date(gameState.lastExchangeAt).getTime()
    return Math.max(0, Math.ceil(3600 - (Date.now() - last) / 1000))
  })()
  const onCooldown = cooldownRemaining > 0

  const handleToggle = (dir: 'to_city' | 'to_account') => {
    if (expanded === dir) { setExpanded(null); return }
    setExpanded(dir)
    setAmount(dir === 'to_city' ? 1 : 15)
  }

  const handleExchange = async () => {
    if (!account || !activePlayerId || loading || onCooldown || !expanded) return
    setLoading(true)
    try {
      if (expanded === 'to_city') {
        const result = await gameApi.exchangeGold(account.accountId, activePlayerId, amount)
        setState(result.state)
        toast.success(`${amount * 10} 城金已存入「${cityName}」`)
      } else {
        const result = await gameApi.reverseExchangeGold(account.accountId, activePlayerId, amount)
        setState(result.state)
        toast.success(`从「${cityName}」提取 ${Math.floor(amount / 15)} 金币到账户`)
      }
      setExpanded(null)
    } catch { /* global */ } finally { setLoading(false) }
  }

  const formatCooldown = (s: number) => `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`
  const minAmount = expanded === 'to_account' ? 15 : 1
  const preview = expanded === 'to_city'
    ? `→ ${(amount * 10).toLocaleString()} 城金 存入「${cityName}」`
    : expanded === 'to_account'
      ? `→ ${Math.floor(amount / 15)} 金币 到账户（损耗33%）`
      : ''

  return (
    <div>
      {/* Top row: balance + 3 buttons */}
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <span className="text-sm text-[var(--color-text-secondary)]">您目前帐户上有</span>
          <span className="text-lg font-bold text-amber-500">{account?.gold ?? 0}</span>
          <span className="text-sm text-[var(--color-text-secondary)]">金币</span>
        </div>
        <div className="flex items-center gap-2">
          {account && activePlayerId && (
            <>
              <button
                type="button"
                onClick={() => handleToggle('to_city')}
                className={`px-3 py-1.5 rounded-xl text-xs font-semibold cursor-pointer transition-all duration-200 ${expanded === 'to_city' ? 'bg-indigo-500/20 text-indigo-600 border border-indigo-500/40' : 'bg-indigo-500/10 text-indigo-600 border border-indigo-500/30 hover:bg-indigo-500/20'}`}
              >
                金币→城金
              </button>
              <button
                type="button"
                onClick={() => handleToggle('to_account')}
                className={`px-3 py-1.5 rounded-xl text-xs font-semibold cursor-pointer transition-all duration-200 ${expanded === 'to_account' ? 'bg-amber-500/20 text-amber-600 border border-amber-500/40' : 'bg-amber-500/10 text-amber-600 border border-amber-500/30 hover:bg-amber-500/20'}`}
              >
                城金→金币
              </button>
            </>
          )}
          <button
            type="button"
            className="px-3 py-1.5 rounded-xl text-xs font-semibold bg-amber-500 text-white hover:bg-amber-400 cursor-pointer transition-all duration-200 shadow-[0_4px_12px_rgba(245,158,11,0.3)]"
          >
            充值金币
          </button>
        </div>
      </div>

      {/* Expand panel */}
      <div className={`overflow-hidden transition-all duration-200 ease-out ${expanded ? 'max-h-[80px] mt-3 pt-3 border-t border-[var(--color-border)] opacity-100' : 'max-h-0 opacity-0'}`}>
        <div className="flex items-center gap-2 flex-wrap">
          <input
            type="number"
            min={minAmount}
            value={amount}
            onChange={(e) => setAmount(Math.max(minAmount, parseInt(e.target.value) || minAmount))}
            className="w-16 text-center text-xs font-bold bg-[var(--color-surface-dim)] border border-[var(--color-border)] rounded-lg py-1.5 text-[var(--color-text-primary)] outline-none focus:border-[var(--color-accent-border)]"
          />
          <span className="text-[10px] text-[var(--color-text-secondary)]">{preview}</span>
          <span className="text-[9px] text-[var(--color-text-muted)] ml-auto mr-2">冷却1h</span>
          <button
            type="button"
            onClick={handleExchange}
            disabled={loading || onCooldown || amount < minAmount}
            className="px-3 py-1.5 rounded-lg text-[10px] font-bold bg-[var(--color-accent)] text-white hover:opacity-90 cursor-pointer transition-opacity disabled:opacity-50"
          >
            {loading ? '...' : onCooldown ? formatCooldown(cooldownRemaining) : '确认兑换'}
          </button>
        </div>
      </div>
    </div>
  )
}

export default AccountPage
