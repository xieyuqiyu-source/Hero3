import { useState, type FC, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { Cloud } from 'lucide-react'
import { gameApi } from '@/api/game'
import GeneralCarousel from '@/components/GeneralCarousel'
import CloudSyncModal from '@/components/CloudSyncModal'
import { useGameStore } from '@/store/gameStore'
import type { AccountSession } from '@/types/game'
import heroBg from '@/assets/hero3background.png'

type Faction = 'wei' | 'shu' | 'wu'

interface General {
  id: string
  name: string
  title: string
  faction: string
}

interface FactionData {
  key: Faction
  name: string
  subtitle: string
  color: string
  borderActive: string
  motto: string
  desc: string
  traits: string[]
  generals: General[]
}

const FACTIONS: FactionData[] = [
  {
    key: 'wei',
    name: '魏',
    subtitle: '曹魏',
    color: 'text-blue-400',
    borderActive: 'border-blue-400',
    motto: '宁教我负天下人，休教天下人负我',
    desc: '挟天子以令诸侯，坐拥中原沃土。曹魏以铁腕治军、唯才是举闻名于世，麾下谋臣如雨、猛将如云，兵精粮足，雄踞北方。',
    traits: ['兵力恢复加成', '谋略系武将强化', '资源产出稳定'],
    generals: [
      { id: 'caocao', name: '曹操', title: '魏武帝', faction: 'wei' },
      { id: 'simayi', name: '司马懿', title: '冢虎', faction: 'wei' },
      { id: 'xiahouyuan', name: '夏侯渊', title: '虎步关右', faction: 'wei' },
      { id: 'zhangliao', name: '张辽', title: '威震逍遥津', faction: 'wei' },
      { id: 'xuchu', name: '许褚', title: '虎痴', faction: 'wei' },
      { id: 'guojia', name: '郭嘉', title: '鬼才', faction: 'wei' },
      { id: 'xunyu', name: '荀彧', title: '王佐之才', faction: 'wei' },
      { id: 'dianwei', name: '典韦', title: '古之恶来', faction: 'wei' },
    ],
  },
  {
    key: 'shu',
    name: '蜀',
    subtitle: '蜀汉',
    color: 'text-green-400',
    borderActive: 'border-green-400',
    motto: '勿以恶小而为之，勿以善小而不为',
    desc: '兴复汉室，还于旧都。蜀汉以仁义立国，桃园结义传为佳话，五虎上将威震四方，卧龙凤雏运筹帷幄，虽偏安一隅却志在天下。',
    traits: ['武将忠诚度加成', '步兵系战力强化', '防御建筑加成'],
    generals: [
      { id: 'liubei', name: '刘备', title: '昭烈帝', faction: 'shu' },
      { id: 'guanyu', name: '关羽', title: '武圣', faction: 'shu' },
      { id: 'zhangfei', name: '张飞', title: '万人敌', faction: 'shu' },
      { id: 'zhugeliang', name: '诸葛亮', title: '卧龙', faction: 'shu' },
      { id: 'zhaoyun', name: '赵云', title: '常胜将军', faction: 'shu' },
      { id: 'machao', name: '马超', title: '锦马超', faction: 'shu' },
      { id: 'huangzhong', name: '黄忠', title: '老当益壮', faction: 'shu' },
      { id: 'weiyan', name: '魏延', title: '踏破敌阵', faction: 'shu' },
    ],
  },
  {
    key: 'wu',
    name: '吴',
    subtitle: '东吴',
    color: 'text-red-400',
    borderActive: 'border-red-400',
    motto: '内事不决问张昭，外事不决问周瑜',
    desc: '据江东六郡以观天下，凭长江天险与水战之利鼎足三分。东吴英才辈出，水军无敌，火攻赤壁名垂青史，以少胜多屡建奇功。',
    traits: ['水军战力加成', '弓兵系强化', '贸易收入加成'],
    generals: [
      { id: 'sunquan', name: '孙权', title: '大帝', faction: 'wu' },
      { id: 'zhouyu', name: '周瑜', title: '美周郎', faction: 'wu' },
      { id: 'lvmeng', name: '吕蒙', title: '白衣渡江', faction: 'wu' },
      { id: 'luxun', name: '陆逊', title: '火烧连营', faction: 'wu' },
      { id: 'ganning', name: '甘宁', title: '锦帆贼', faction: 'wu' },
      { id: 'taishici', name: '太史慈', title: '信义笃烈', faction: 'wu' },
      { id: 'huanggai', name: '黄盖', title: '苦肉计', faction: 'wu' },
      { id: 'sunce', name: '孙策', title: '小霸王', faction: 'wu' },
    ],
  },
]

const LoginPage: FC = () => {
  const [nickname, setNickname] = useState('')
  const [faction, setFaction] = useState<Faction | null>(null)
  const [selectedGeneral, setSelectedGeneral] = useState<string | null>(null)
  const [cloudSyncOpen, setCloudSyncOpen] = useState(false)
  const [account, setAccount] = useState<AccountSession | null>(() => {
    const accountId = localStorage.getItem('hero3_account_id')
    const username = localStorage.getItem('hero3_account_name')
    return accountId && username ? { accountId, username } : null
  })
  const [creating, setCreating] = useState(false)
  const [showSyncReminder, setShowSyncReminder] = useState(false)
  const setGameState = useGameStore((store) => store.setState)
  const setActivePlayer = useGameStore((store) => store.setActivePlayer)
  const loadGameState = useGameStore((store) => store.loadGameState)
  const navigate = useNavigate()

  const canSubmit = nickname.trim().length > 0 && faction !== null && selectedGeneral !== null

  const handleFactionClick = (key: Faction) => {
    if (faction === key) return
    setFaction(key)
    // Auto-select first general when switching faction
    const factionData = FACTIONS.find((f) => f.key === key)
    if (factionData) {
      setSelectedGeneral(factionData.generals[0].id)
    }
  }

  const handlePlayerSelected = async (playerId: string) => {
    setActivePlayer(playerId)
    await loadGameState(playerId)
    navigate('/city')
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    if (!canSubmit || !faction) return

    // If not logged in, show sync reminder first
    if (!account && !showSyncReminder) {
      setShowSyncReminder(true)
      return
    }

    setCreating(true)
    try {
      if (account) {
        const result = await gameApi.createPlayer(account.accountId, nickname, faction, selectedGeneral ?? undefined)
        setActivePlayer(result.playerId)
        setGameState(result.state)
      }
      // TODO: handle local-only mode when no account
      navigate('/city')
    } finally {
      setCreating(false)
    }
  }

  const handleSkipSync = () => {
    setShowSyncReminder(false)
    setCreating(true)
    // Enter game without cloud sync
    // TODO: create local player
    navigate('/city')
  }

  return (
    <div className="min-h-dvh flex flex-col relative overflow-hidden">
      {/* Background image - full visible */}
      <div
        className="absolute inset-0 bg-cover bg-center bg-no-repeat"
        style={{ backgroundImage: `url(${heroBg})` }}
      />

      <form onSubmit={handleSubmit} className="relative z-10 flex flex-col h-dvh p-4 sm:p-6 lg:p-8">

        {/* Top: Logo left-aligned */}
        <div className="flex items-center mb-5 flex-shrink-0">
          <h1 className="text-3xl sm:text-4xl font-black tracking-wide text-white drop-shadow-lg">
            <span style={{ fontFamily: "'STKaiti', 'KaiTi', 'SimKai', serif" }}>英雄三国</span>
            <span className="text-xl sm:text-2xl font-bold tracking-tight ml-3 opacity-80">Hero3</span>
          </h1>
          <button
            type="button"
            onClick={() => setCloudSyncOpen(true)}
            className="
              ml-4 flex items-center gap-1.5 px-3 py-1.5 rounded-lg
              bg-green-600 text-white text-xs font-medium
              hover:bg-green-500 hover:-translate-y-0.5
              cursor-pointer transition-all duration-200
              shadow-[0_4px_12px_rgba(22,163,74,0.3)]
            "
          >
            <Cloud size={13} />
            {account ? account.username : '云同步'}
          </button>
        </div>

        {/* Main: Three Faction Columns */}
        <div className="flex-1 flex flex-col sm:grid sm:grid-cols-3 gap-2 sm:gap-4 min-h-0">
          {FACTIONS.map((f) => {
            const isActive = faction === f.key
            // On mobile: collapsed factions show as single line
            const isMobileCollapsed = faction !== null && !isActive

            return (
              <div
                key={f.key}
                onClick={() => handleFactionClick(f.key)}
                className={`
                  flex flex-col rounded-2xl border overflow-hidden cursor-pointer
                  backdrop-blur-sm
                  transition-all duration-400 ease-in-out
                  ${isActive
                    ? `${f.borderActive} bg-[var(--color-surface)]/90 opacity-100 shadow-lg sm:opacity-100 max-sm:flex-1`
                    : 'border-white/10 bg-[var(--color-surface)]/50 opacity-80 hover:opacity-90'
                  }
                  ${isMobileCollapsed ? 'max-sm:flex-none max-sm:h-11' : 'max-sm:flex-1'}
                `}
              >
                {/* Faction Header */}
                <div className={`
                  flex items-center justify-center transition-all duration-400 ease-in-out
                  ${isMobileCollapsed
                    ? 'py-2 sm:py-6 sm:flex-col'
                    : 'py-6 sm:py-8 flex-col'
                  }
                `}>
                  <span
                    className={`
                      font-black transition-all duration-400 ease-in-out
                      ${isActive ? f.color : 'text-[var(--color-text-muted)]'}
                      ${isMobileCollapsed
                        ? 'text-xl sm:text-5xl sm:text-6xl lg:text-7xl'
                        : 'text-5xl sm:text-6xl lg:text-7xl'
                      }
                    `}
                    style={{ fontFamily: "'STKaiti', 'KaiTi', 'SimKai', serif" }}
                  >
                    {f.name}
                  </span>
                  <span className={`
                    transition-all duration-400 ease-in-out
                    ${isActive ? 'text-[var(--color-text-primary)]' : 'text-[var(--color-text-muted)]'}
                    ${isMobileCollapsed
                      ? 'ml-2 text-xs sm:ml-0 sm:mt-2 sm:text-sm'
                      : 'mt-2 text-sm'
                    }
                  `}>
                    {f.subtitle}
                  </span>
                </div>

                {/* Content - hidden on mobile when collapsed */}
                <div className={`
                  flex flex-col flex-1 min-h-0 overflow-hidden
                  transition-all duration-400 ease-in-out
                  ${isMobileCollapsed ? 'max-sm:max-h-0 max-sm:opacity-0' : 'max-sm:max-h-[2000px] max-sm:opacity-100'}
                `}>
                  {/* Motto */}
                  <div className="px-5 pb-3">
                    <p
                      className={`text-center text-xs italic ${isActive ? 'text-[var(--color-text-secondary)]' : 'text-[var(--color-text-muted)]'} transition-colors duration-200`}
                      style={{ fontFamily: "'STKaiti', 'KaiTi', 'SimKai', serif", fontSize: 20 }}
                    >
                      「{f.motto}」
                    </p>
                  </div>

                  {/* Description */}
                  <div className="px-5 pb-4">
                    <p className={`text-sm leading-relaxed ${isActive ? 'text-[var(--color-text-secondary)]' : 'text-[var(--color-text-muted)]'} transition-colors duration-200`}>
                      {f.desc}
                    </p>
                    <div className="flex flex-wrap gap-2 mt-4">
                      {f.traits.map((trait) => (
                        <span
                          key={trait}
                          className={`
                            px-3 py-1 rounded-lg text-xs font-medium
                            ${isActive ? 'bg-[var(--color-accent-light)] text-[var(--color-text-primary)] border border-[var(--color-border)]' : 'bg-[var(--color-surface-dim)]/50 text-[var(--color-text-muted)] border border-[var(--color-border)]'}
                            transition-colors duration-200
                          `}
                        >
                          {trait}
                        </span>
                      ))}
                    </div>
                  </div>

                  {/* Generals - carousel in center area */}
                  <div className="flex-1 flex items-center justify-center px-2 py-2 min-h-[180px]">
                    <div className="w-full h-full">
                      <GeneralCarousel
                        generals={f.generals}
                        selectedId={faction === f.key ? selectedGeneral : null}
                        color={f.color}
                        borderActive={f.borderActive}
                        onSelect={(id) => {
                          setFaction(f.key)
                          setSelectedGeneral(id)
                        }}
                      />
                    </div>
                  </div>

                  {/* Bottom action - only show when this faction is fully selected */}
                <div className={`
                  border-t border-[var(--color-border)] px-4 py-3
                  transition-all duration-300 overflow-hidden
                  ${isActive && selectedGeneral ? 'max-h-40 opacity-100' : 'max-h-0 opacity-0 py-0 border-t-0'}
                `}>
                  {/* Sync reminder */}
                  {showSyncReminder && !account && isActive && (
                    <div className="mb-3 p-3 rounded-xl bg-amber-500/10 border border-amber-500/30">
                      <p className="text-xs text-amber-200 mb-2">是否登录同步云存档？登录后可多设备同步进度。</p>
                      <div className="flex items-center gap-2">
                        <button
                          type="button"
                          onClick={(e) => { e.stopPropagation(); setCloudSyncOpen(true); setShowSyncReminder(false) }}
                          className="px-3 py-1.5 rounded-lg text-xs font-medium bg-green-600 text-white cursor-pointer hover:bg-green-500 transition-colors"
                        >
                          去登录
                        </button>
                        <button
                          type="button"
                          onClick={(e) => { e.stopPropagation(); handleSkipSync() }}
                          className="px-3 py-1.5 rounded-lg text-xs font-medium text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] cursor-pointer transition-colors"
                        >
                          暂不登录，直接进入
                        </button>
                      </div>
                    </div>
                  )}

                  {!showSyncReminder && (
                    <div className="flex items-center gap-3">
                      <input
                        type="text"
                        value={nickname}
                        onChange={(e) => setNickname(e.target.value)}
                        onClick={(e) => e.stopPropagation()}
                        placeholder="输入主公名号"
                        maxLength={12}
                        className="
                          flex-1 px-3 py-2 rounded-xl
                          bg-[var(--color-surface-dim)] border border-[var(--color-border)]
                          text-[var(--color-text-primary)] placeholder-[var(--color-text-muted)]
                          text-sm outline-none
                          focus:border-[var(--color-accent-border)]
                          transition-colors duration-200
                        "
                      />
                      <button
                        type="submit"
                        disabled={!canSubmit || creating}
                        onClick={(e) => e.stopPropagation()}
                        className={`
                          px-4 py-2 rounded-xl text-sm font-semibold whitespace-nowrap
                          transition-all duration-200
                          ${canSubmit && !creating
                            ? 'bg-[var(--color-accent)] text-white hover:-translate-y-0.5 cursor-pointer'
                            : 'bg-[var(--color-surface-dim)] text-[var(--color-text-muted)] cursor-not-allowed'
                          }
                        `}
                      >
                        {creating ? '创建中...' : '征战天下'}
                      </button>
                    </div>
                  )}
                </div>
                </div>
              </div>
            )
          })}
        </div>

        {/* Footer */}
        <div className="text-center pt-4 flex-shrink-0">
          <p className="text-[11px] text-white/40">天下大势，分久必合，合久必分</p>
        </div>
      </form>
      <CloudSyncModal
        open={cloudSyncOpen}
        onClose={() => setCloudSyncOpen(false)}
        onAccountReady={setAccount}
        onPlayerSelected={(playerId) => void handlePlayerSelected(playerId)}
      />
    </div>
  )
}

export default LoginPage
