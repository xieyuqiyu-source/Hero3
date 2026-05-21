import { useState, type FC } from 'react'
import { useNavigate } from 'react-router-dom'
import GeneralCarousel from '@/components/GeneralCarousel'
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

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!canSubmit) return
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
        </div>

        {/* Main: Three Faction Columns */}
        <div className="flex-1 grid grid-cols-1 sm:grid-cols-3 gap-4 min-h-0">
          {FACTIONS.map((f) => {
            const isActive = faction === f.key
            return (
              <div
                key={f.key}
                onClick={() => handleFactionClick(f.key)}
                className={`
                  flex flex-col rounded-2xl border overflow-hidden cursor-pointer
                  backdrop-blur-sm
                  transition-all duration-300
                  ${isActive
                    ? `${f.borderActive} bg-[var(--color-surface)]/90 opacity-100 shadow-lg`
                    : 'border-white/10 bg-[var(--color-surface)]/40 opacity-40 hover:opacity-60'
                  }
                `}
              >
                {/* Faction Header */}
                <div className="flex flex-col items-center justify-center py-6 sm:py-8">
                  <span
                    className={`text-5xl sm:text-6xl lg:text-7xl font-black ${isActive ? f.color : 'text-[var(--color-text-muted)]'} transition-colors duration-200`}
                    style={{ fontFamily: "'STKaiti', 'KaiTi', 'SimKai', serif" }}
                  >
                    {f.name}
                  </span>
                  <span className={`text-sm mt-2 ${isActive ? 'text-[var(--color-text-primary)]' : 'text-[var(--color-text-muted)]'} transition-colors duration-200`}>
                    {f.subtitle}
                  </span>
                </div>

                {/* Motto */}
                <div className="px-5 pb-3">
                  <p
                    className={`text-center text-xs italic ${isActive ? 'text-[var(--color-text-secondary)]' : 'text-[var(--color-text-muted)]'} transition-colors duration-200`}
                    style={{ fontFamily: "'STKaiti', 'KaiTi', 'SimKai', serif" }}
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
                  ${isActive && selectedGeneral ? 'max-h-20 opacity-100' : 'max-h-0 opacity-0 py-0 border-t-0'}
                `}>
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
                      disabled={!canSubmit}
                      onClick={(e) => e.stopPropagation()}
                      className={`
                        px-4 py-2 rounded-xl text-sm font-semibold whitespace-nowrap
                        transition-all duration-200
                        ${canSubmit
                          ? 'bg-[var(--color-accent)] text-white hover:-translate-y-0.5 cursor-pointer'
                          : 'bg-[var(--color-surface-dim)] text-[var(--color-text-muted)] cursor-not-allowed'
                        }
                      `}
                    >
                      征战天下
                    </button>
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
    </div>
  )
}

export default LoginPage
