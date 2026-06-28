/* 本文件实现地图页副本入口占位 UI，提前展示常驻副本和限时副本。 */
import { type FC } from 'react'
import { Crown, Flame, Lock, ScrollText, ShieldAlert, Swords } from 'lucide-react'
import reincarnationAbyssBg from '@/assets/dungeons/reincarnation-abyss.webp'
import kingsWarBg from '@/assets/dungeons/kings-war.webp'
import famousGeneralsBg from '@/assets/dungeons/famous-generals.webp'

interface DungeonEntry {
  id: string
  title: string
  subtitle: string
  icon: typeof Swords
  limited?: boolean
  backgroundImage?: string
}

const DUNGEONS: DungeonEntry[] = [
  { id: 'reincarnation-abyss', title: '轮回绝境', subtitle: '九死轮回，万劫不复', icon: Flame, backgroundImage: reincarnationAbyssBg },
  { id: 'kings-war', title: '万王争霸', subtitle: '诸王并起，问鼎天下', icon: Crown, backgroundImage: kingsWarBg },
  { id: 'famous-generals', title: '天下名将', subtitle: '群雄列阵，名将归心', icon: Swords, backgroundImage: famousGeneralsBg },
]

const LIMITED_DUNGEONS: DungeonEntry[] = [
  { id: 'god-demon-battlefield', title: '神魔战场', subtitle: '限时副本，神魔裂土', icon: ShieldAlert, limited: true },
  { id: 'ancient-heaven', title: '远古天庭', subtitle: '限时副本，金阙重开', icon: ScrollText, limited: true },
]

// DungeonTab 渲染副本列表和限时副本占位。
const DungeonTab: FC = () => (
  <div className="space-y-5">
    <div>
      <h2 className="text-lg font-bold text-[var(--color-text-primary)]">副本</h2>
      <p className="mt-1 text-sm text-[var(--color-text-muted)]">挑战玩法即将开放，当前仅展示入口占位</p>
    </div>

    <div className="space-y-4">
      {DUNGEONS.map((dungeon) => (
        <DungeonRow key={dungeon.id} dungeon={dungeon} />
      ))}
    </div>

    <div className="pt-2">
      <div className="mb-3 flex items-center gap-2">
        <span className="h-px flex-1 bg-amber-500/25" />
        <span className="rounded-full border border-amber-400/40 bg-amber-400/10 px-3 py-1 text-xs font-bold text-amber-600">限时副本</span>
        <span className="h-px flex-1 bg-amber-500/25" />
      </div>
      <div className="space-y-4">
        {LIMITED_DUNGEONS.map((dungeon) => (
          <DungeonRow key={dungeon.id} dungeon={dungeon} />
        ))}
      </div>
    </div>
  </div>
)

// DungeonRow 渲染单个副本入口行。
const DungeonRow: FC<{ dungeon: DungeonEntry }> = ({ dungeon }) => {
  const Icon = dungeon.icon
  const isLimited = dungeon.limited
  const hasBackground = Boolean(dungeon.backgroundImage)

  return (
    <article
      data-dungeon-id={dungeon.id}
      className={`
        group relative min-h-[154px] overflow-hidden rounded-2xl border p-5 sm:min-h-[180px] sm:p-6
        ${isLimited
          ? 'border-amber-400/45 bg-[linear-gradient(135deg,rgba(120,53,15,0.16),rgba(245,158,11,0.13),rgba(255,255,255,0.04))] shadow-[0_18px_42px_rgba(180,83,9,0.12)]'
          : 'border-[var(--color-border)] bg-[var(--color-surface)] shadow-[0_12px_34px_rgba(15,23,42,0.06)]'
        }
      `}
    >
      {hasBackground && (
        <div
          className="absolute inset-y-0 left-0 right-0 opacity-80 transition-transform duration-500 group-hover:scale-[1.025]"
          style={{
            backgroundImage: `url(${dungeon.backgroundImage})`,
            backgroundPosition: 'center',
            backgroundSize: 'cover',
            clipPath: 'polygon(0 18%, 7% 10%, 16% 16%, 26% 9%, 38% 13%, 50% 7%, 63% 12%, 76% 8%, 90% 15%, 100% 10%, 100% 88%, 92% 82%, 80% 90%, 66% 84%, 54% 93%, 42% 86%, 29% 91%, 18% 84%, 8% 91%, 0 84%)',
            maskImage: 'linear-gradient(90deg, transparent 0%, rgba(0,0,0,0.2) 8%, black 24%, black 72%, rgba(0,0,0,0.32) 84%, transparent 100%), radial-gradient(ellipse at center, black 0%, black 54%, transparent 83%)',
            WebkitMaskImage: 'linear-gradient(90deg, transparent 0%, rgba(0,0,0,0.2) 8%, black 24%, black 72%, rgba(0,0,0,0.32) 84%, transparent 100%), radial-gradient(ellipse at center, black 0%, black 54%, transparent 83%)',
            maskComposite: 'intersect',
            WebkitMaskComposite: 'source-in',
          }}
        />
      )}
      <div className={`absolute inset-0 ${isLimited ? 'bg-[radial-gradient(circle_at_82%_18%,rgba(251,191,36,0.22),transparent_32%)]' : 'bg-[radial-gradient(circle_at_82%_18%,rgba(100,116,139,0.11),transparent_30%)]'}`} />
      <div className={`absolute inset-0 ${hasBackground ? 'bg-[linear-gradient(90deg,rgba(2,6,23,0.88),rgba(2,6,23,0.34)_38%,rgba(2,6,23,0.08)_62%,rgba(2,6,23,0.72))]' : 'bg-[linear-gradient(90deg,rgba(2,6,23,0.18),transparent_42%)] opacity-0 group-hover:opacity-100'} transition-opacity duration-300`} />

      <div className="relative flex h-full flex-col justify-between gap-5 sm:flex-row sm:items-center">
        <div className="flex items-center gap-4">
          <div className={`flex h-14 w-14 shrink-0 items-center justify-center rounded-2xl border sm:h-16 sm:w-16 ${isLimited ? 'border-amber-400/50 bg-amber-400/15 text-amber-500' : 'border-[var(--color-border)] bg-[var(--color-surface-dim)] text-[var(--color-accent)]'}`}>
            <Icon size={28} />
          </div>
          <div>
            <p className={`mb-1 text-xs font-bold tracking-[0.22em] ${isLimited ? 'text-amber-600' : 'text-[var(--color-text-muted)]'}`}>
              {isLimited ? 'LIMITED TIME' : 'DUNGEON'}
            </p>
            <h3
              className={`
                text-[clamp(2rem,4vw,3.8rem)] font-black leading-none tracking-normal
                [font-family:"STKaiti","KaiTi","Songti_SC","SimSun",serif]
                ${isLimited ? 'text-amber-500 drop-shadow-[0_2px_10px_rgba(245,158,11,0.25)]' : 'text-[var(--color-text-primary)]'}
              `}
            >
              {dungeon.title}
            </h3>
            <p className={`mt-3 text-sm font-medium ${isLimited ? 'text-amber-700/80 dark:text-amber-300/80' : 'text-[var(--color-text-secondary)]'}`}>{dungeon.subtitle}</p>
          </div>
        </div>

        <div className={`inline-flex w-fit items-center gap-2 rounded-full border px-4 py-2 text-sm font-bold ${isLimited ? 'border-amber-400/50 bg-amber-400/15 text-amber-600' : 'border-[var(--color-border)] bg-[var(--color-surface-dim)] text-[var(--color-text-muted)]'}`}>
          <Lock size={15} />
          待开放
        </div>
      </div>
    </article>
  )
}

export default DungeonTab
