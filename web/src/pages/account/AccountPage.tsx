import { useState, useEffect, type FC } from 'react'
import { useNavigate } from 'react-router-dom'
import { Coins, Crown, Scroll, Users, Castle, KeyRound, Sparkles, Cloud, LogOut, Check } from 'lucide-react'
import { useAccountStore } from '@/store/accountStore'
import { useGameStore } from '@/store/gameStore'
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
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
          <div className="flex items-center gap-2">
            <span className="text-sm text-[var(--color-text-secondary)]">您目前帐户上有</span>
            <span className="text-lg font-bold text-amber-500">0</span>
            <span className="text-sm text-[var(--color-text-secondary)]">金币</span>
          </div>
          <button
            type="button"
            className="
              px-4 py-1.5 rounded-xl text-xs font-semibold
              bg-amber-500 text-white
              hover:bg-amber-400 hover:-translate-y-0.5
              cursor-pointer transition-all duration-200
              shadow-[0_4px_12px_rgba(245,158,11,0.3)]
            "
          >
            充值金币
          </button>
        </div>
      </Section>

      {/* 个人概况 */}
      <Section title="个人概况" icon={Crown}>
        <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
          <InfoItem label="昵称" value={gameState?.player.nickname ?? '--'} highlight />
          <InfoItem label="天梯排名" value="等待统计中.." />
          <InfoItem label="国度" value={gameState?.player.faction ?? '--'} />
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
                        {p.faction}
                      </span>
                      {isActive && (
                        <span className="text-[10px] text-[var(--color-accent)] font-medium">当前</span>
                      )}
                    </div>
                    <div className="flex items-center gap-3 text-[10px] text-[var(--color-text-muted)]">
                      <span>兵力 {p.totalArmy.toLocaleString()}</span>
                      <span>建筑总等级 {p.buildingLevel}</span>
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

export default AccountPage
