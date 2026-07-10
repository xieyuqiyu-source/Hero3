/* 本文件按传统战报骨架渲染一个进攻、防守或协防参与方区块。 */
import type { FC } from 'react'
import type { BattleReportRewards, BattleReportSide } from '@/types/game'
import { mergeBattleReportDrops } from '@/utils/reportDrops'
import UnitLossMatrix from './UnitLossMatrix'

const RESOURCE_LABELS: Record<string, string> = { wood: '木材', stone: '石料', iron: '铁矿', food: '粮食' }
const RESOURCE_ORDER = ['wood', 'stone', 'iron', 'food']

interface BattleParticipantBlockProps {
  title: string
  side: BattleReportSide
  rewards?: BattleReportRewards
  feedback?: string
  effectText?: string
  effectTone?: 'normal' | 'highlight'
  result?: 'victory' | 'defeat' | 'draw' | 'none'
  settlement?: 'attacker' | 'defender' | 'none'
  buildingDamage?: string
  showUnits?: boolean
  showResources?: boolean
  showGenerals?: boolean
}

const KAI_FONT = '"KaiTi", "STKaiti", "Kaiti SC", serif'

// formatGenerals 将参战武将压缩为一行文本。
function formatGenerals(side: BattleReportSide): string {
  const general = side.generals?.[0]
  if (!general) return '无参战武将'
  return `${general.name || general.id}${general.level ? ` Lv.${general.level}` : ''}`
}

// getFutureTextField 读取后续后端可能补上的称号字段。
function getFutureTextField(side: BattleReportSide, keys: string[]): string {
  const view = side as unknown as Record<string, unknown>
  const value = keys.map((key) => view[key]).find((item) => typeof item === 'string' && item.trim())
  return typeof value === 'string' ? value : ''
}

// resolveIdentityName 生成标题栏右侧的玩家、NPC 或增援方名称。
function resolveIdentityName(side: BattleReportSide): string {
  return side.playerName || side.targetName || side.cityName || side.targetId || side.playerId || '-'
}

// resolveFactionName 生成左侧徽标里的阵营名称。
function resolveFactionName(side: BattleReportSide): string {
  const value = side.factionLabel || side.faction || '-'
  const key = value.toLowerCase()
  if (key === 'wei' || value === '魏') return '魏'
  if (key === 'shu' || value === '蜀') return '蜀'
  if (key === 'wu' || value === '吴') return '吴'
  return value
}

// factionToneClass 按阵营返回徽标颜色，魏蜀吴和未知阵营保持区分。
function factionToneClass(faction?: string): string {
  const key = (faction || '').toLowerCase()
  if (key.includes('wei') || key.includes('魏')) return 'border-sky-400/50 bg-sky-500/20 text-sky-200 shadow-[inset_0_0_18px_rgba(56,189,248,0.18)]'
  if (key.includes('shu') || key.includes('蜀')) return 'border-emerald-400/50 bg-emerald-500/20 text-emerald-200 shadow-[inset_0_0_18px_rgba(52,211,153,0.18)]'
  if (key.includes('wu') || key.includes('吴')) return 'border-red-400/50 bg-red-500/20 text-red-200 shadow-[inset_0_0_18px_rgba(248,113,113,0.18)]'
  return 'border-amber-400/50 bg-amber-500/20 text-amber-200 shadow-[inset_0_0_18px_rgba(251,191,36,0.18)]'
}

// resultSealView 返回右上角胜败印的文字和颜色。
function resultSealView(result: BattleParticipantBlockProps['result']): { label: string; className: string } {
  if (result === 'victory') return { label: '胜', className: 'border-emerald-400/50 bg-emerald-500/10 text-emerald-400' }
  if (result === 'defeat') return { label: '败', className: 'border-red-400/50 bg-red-500/10 text-red-400' }
  if (result === 'draw') return { label: '平', className: 'border-slate-400/50 bg-slate-500/10 text-slate-300' }
  return { label: '-', className: 'border-[var(--color-border)] text-[var(--color-text-muted)]' }
}

// formatResources 将资源奖励或资源快照压缩为一行文本。
function formatResources(resources?: Record<string, number>): string {
  const text = RESOURCE_ORDER
    .map((key) => `${RESOURCE_LABELS[key]}:${(resources?.[key] ?? 0).toLocaleString()}`)
    .join('  ')
  return text || '无'
}

// formatDrops 将掉落快照压缩为一行文本。
function formatDrops(rewards?: BattleReportRewards): string {
  const drops = mergeBattleReportDrops(rewards?.drops ?? [])
  if (drops.length === 0) return '无'
  return drops.map((drop) => `${drop.name || drop.itemId || '掉落'} x${drop.amount.toLocaleString()}`).join('、')
}

// formatRewardFeedback 生成战后奖励和反馈说明。
function formatRewardFeedback(rewards?: BattleReportRewards, fallback?: string, includeResources = false): string {
  const parts: string[] = []
  if ((rewards?.generalExp ?? 0) > 0) parts.push(`武将经验 +${rewards?.generalExp}`)
  if ((rewards?.cityGold ?? 0) > 0) parts.push(`城金 +${rewards?.cityGold}`)
  if (includeResources) {
    Object.entries(rewards?.resources ?? {}).forEach(([key, amount]) => {
      if (amount > 0) parts.push(`${RESOURCE_LABELS[key] || key} +${amount.toLocaleString()}`)
    })
  }
  return parts.join('，') || fallback || '无'
}

// BattleParticipantBlock 渲染完整参与方区块：身份、效果、兵种、资源和战损反馈。
const BattleParticipantBlock: FC<BattleParticipantBlockProps> = ({
  title,
  side,
  rewards,
  feedback,
  effectText,
  effectTone = 'normal',
  result = 'none',
  settlement = 'none',
  buildingDamage = '无',
  showUnits = true,
  showResources = true,
  showGenerals = true,
}) => {
  const titleName = resolveIdentityName(side)
  const factionName = resolveFactionName(side)
  const seal = resultSealView(result)
  const officialTitle = getFutureTextField(side, ['officialTitle', 'officeTitle', 'officialRank', 'office'])
  const militaryRank = getFutureTextField(side, ['militaryRank', 'rankTitle', 'rank'])

  return (
    <section className="overflow-hidden rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)]">
      <div className="flex items-center gap-2 border-b border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-xs font-black">
        <span className="shrink-0 text-amber-500">{title}</span>
        <span className="min-w-0 flex-1 truncate text-left text-[var(--color-text-primary)]">{titleName}</span>
        <span className={`flex h-7 w-7 shrink-0 items-center justify-center rounded border text-base font-black leading-none ${seal.className}`} style={{ fontFamily: KAI_FONT }}>
          {seal.label}
        </span>
      </div>

      <div className="grid grid-cols-[64px_minmax(0,1fr)] border-b border-[var(--color-border)] sm:grid-cols-[72px_minmax(0,1fr)]">
        <div className="flex items-stretch justify-stretch border-r border-[var(--color-border)]">
          <div className={`flex min-h-full w-full items-center justify-center border text-center text-2xl font-black leading-none ${factionToneClass(side.factionLabel || side.faction)}`} style={{ fontFamily: KAI_FONT }}>
            <span className="line-clamp-2 break-all">{factionName}</span>
          </div>
        </div>
        <div>
          <div className="grid gap-2 border-b border-[var(--color-border)] px-3 py-2 text-[11px] sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center">
            <div className="min-w-0">
              <span className="text-[var(--color-text-muted)]">将领名称：</span>
              <span className="font-bold text-[var(--color-text-primary)]">{showGenerals ? formatGenerals(side) : '情报未揭示'}</span>
            </div>
            <div className="flex flex-wrap gap-x-4 gap-y-1 text-[var(--color-text-secondary)]">
              {officialTitle && <span>官职：<b className="text-[var(--color-text-primary)]">{officialTitle}</b></span>}
              {militaryRank && <span>军衔：<b className="text-[var(--color-text-primary)]">{militaryRank}</b></span>}
              {side.power > 0 && <span>战力：<b className="text-amber-500">{side.power.toLocaleString()}</b></span>}
            </div>
          </div>
          <div className={`px-3 py-2 text-center text-[11px] ${effectTone === 'highlight' ? 'font-bold text-amber-500' : 'text-[var(--color-text-secondary)]'}`}>
            {effectText || '本场无触发效果'}
          </div>
        </div>
      </div>

      {showUnits ? (
        <UnitLossMatrix title="兵种" units={side.units} />
      ) : (
        <div className="border-t border-[var(--color-border)] px-3 py-4 text-center text-xs font-semibold text-[var(--color-text-muted)]">
          敌方兵力情报未揭示
        </div>
      )}

      {settlement !== 'none' && (
        <div className="divide-y divide-[var(--color-border)] border-t border-[var(--color-border)] text-[11px]">
          {settlement === 'attacker' && (
            <div className="grid grid-cols-[88px_1fr]">
              <div className="bg-[var(--color-surface-dim)] px-3 py-2 font-bold text-amber-500">掠夺资源</div>
              <div className="px-3 py-2 text-center text-[var(--color-text-secondary)]">{showResources ? formatResources(rewards?.resources) : '情报未揭示'}</div>
            </div>
          )}
          {settlement === 'defender' && (
            <div className="grid grid-cols-[88px_1fr]">
              <div className="bg-[var(--color-surface-dim)] px-3 py-2 font-bold text-amber-500">建筑损坏</div>
              <div className="px-3 py-2 text-center text-[var(--color-text-secondary)]">{buildingDamage}</div>
            </div>
          )}
          <div className="grid grid-cols-[88px_1fr]">
            <div className="bg-[var(--color-surface-dim)] px-3 py-2 font-bold text-amber-500">战损反馈</div>
            <div className="px-3 py-2 text-center text-[var(--color-text-secondary)]">{formatRewardFeedback(rewards, feedback, settlement === 'defender')}</div>
          </div>
          <div className="grid grid-cols-[88px_1fr]">
            <div className="bg-[var(--color-surface-dim)] px-3 py-2 font-bold text-amber-500">宝物掉落</div>
            <div className="px-3 py-2 text-center text-[var(--color-text-secondary)]">{formatDrops(rewards)}</div>
          </div>
        </div>
      )}
    </section>
  )
}

export default BattleParticipantBlock
