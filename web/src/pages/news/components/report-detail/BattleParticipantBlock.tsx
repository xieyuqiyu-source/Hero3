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
  buildingDamage?: string
  showUnits?: boolean
  showResources?: boolean
  showGenerals?: boolean
}

const KAI_FONT = '"KaiTi", "STKaiti", "Kaiti SC", serif'

// formatGenerals 将参战武将压缩为一行文本。
function formatGenerals(side: BattleReportSide): string {
  const generals = side.generals ?? []
  if (generals.length === 0) return '-'
  return generals.map((general) => `${general.name || general.id}${general.level ? ` Lv.${general.level}` : ''}`).join('、')
}

// getFutureTextField 读取后续后端可能补上的称号字段，当前没有则统一占位。
function getFutureTextField(side: BattleReportSide, keys: string[]): string {
  const view = side as unknown as Record<string, unknown>
  const value = keys.map((key) => view[key]).find((item) => typeof item === 'string' && item.trim())
  return typeof value === 'string' ? value : '-'
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
function formatRewardFeedback(rewards?: BattleReportRewards, fallback?: string): string {
  const parts: string[] = []
  if ((rewards?.generalExp ?? 0) > 0) parts.push(`武将经验 +${rewards?.generalExp}`)
  if ((rewards?.cityGold ?? 0) > 0) parts.push(`城金 +${rewards?.cityGold}`)
  Object.entries(rewards?.resources ?? {}).forEach(([key, amount]) => {
    if (amount > 0) parts.push(`${RESOURCE_LABELS[key] || key} +${amount.toLocaleString()}`)
  })
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
  buildingDamage = '无',
  showUnits = true,
  showResources = true,
  showGenerals = true,
}) => {
  const hasGenerals = showGenerals && (side.generals ?? []).length > 0
  const titleName = resolveIdentityName(side)
  const factionName = resolveFactionName(side)
  const seal = resultSealView(result)

  return (
    <section className="overflow-hidden rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)]">
      <div className="flex items-center justify-start gap-2 border-b border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-xs font-black">
        <span className="text-amber-500">{title}</span>
        <span className="min-w-0 truncate text-left text-amber-300">{titleName}</span>
      </div>

    <div className="grid grid-cols-[64px_1fr] border-b border-[var(--color-border)] sm:grid-cols-[72px_1fr_1fr_1fr_64px]">
      <div className="row-span-3 flex items-stretch justify-stretch border-r border-[var(--color-border)]">
        <div className={`flex min-h-full w-full items-center justify-center border text-center text-2xl font-black leading-none ${factionToneClass(side.factionLabel || side.faction)}`} style={{ fontFamily: KAI_FONT }}>
          <span className="line-clamp-2 break-all">{factionName}</span>
        </div>
      </div>
      {!hasGenerals && side.role === 'defender' ? (
        <div className="border-b border-[var(--color-border)] px-3 py-2 text-center text-[11px] font-semibold text-[var(--color-text-secondary)] sm:col-span-3">
          -
        </div>
      ) : (
        <>
          <div className="border-b border-[var(--color-border)] px-3 py-2 text-[11px] sm:border-r">
            <span className="text-[var(--color-text-muted)]">将领名称：</span>
            <span className="font-bold text-[var(--color-text-primary)]">{showGenerals ? formatGenerals(side) : '情报未揭示'}</span>
          </div>
          <div className="border-b border-[var(--color-border)] px-3 py-2 text-[11px] sm:border-r">
            <span className="text-[var(--color-text-muted)]">官职：</span>
            <span className="font-semibold text-[var(--color-text-secondary)]">
              {getFutureTextField(side, ['officialTitle', 'officeTitle', 'officialRank', 'office'])}
            </span>
          </div>
          <div className="border-b border-[var(--color-border)] px-3 py-2 text-[11px]">
            <span className="text-[var(--color-text-muted)]">军衔：</span>
            <span className="font-semibold text-[var(--color-text-secondary)]">
              {getFutureTextField(side, ['militaryRank', 'rankTitle', 'rank'])}
            </span>
          </div>
        </>
      )}
      <div className="hidden row-span-3 items-center justify-center border-l border-[var(--color-border)] p-2 sm:flex">
        <div className={`flex h-11 w-11 items-center justify-center rounded-md border text-2xl font-black leading-none ${seal.className}`} style={{ fontFamily: KAI_FONT }}>
          {seal.label}
        </div>
      </div>
      <div className="col-span-1 px-3 py-2 text-[11px] text-[var(--color-text-secondary)] sm:col-span-3">
        -
      </div>
      <div className={`col-span-1 border-t border-[var(--color-border)] px-3 py-2 text-center text-[11px] sm:col-span-3 ${effectTone === 'highlight' ? 'font-bold text-amber-500' : 'text-[var(--color-text-secondary)]'}`}>
        {effectText || '-'}
      </div>
    </div>

    {showUnits ? (
      <UnitLossMatrix title="兵种" units={side.units} />
    ) : (
      <div className="border-t border-[var(--color-border)] px-3 py-4 text-center text-xs font-semibold text-[var(--color-text-muted)]">
        敌方剩余兵力情报未揭示
      </div>
    )}

    <div className="divide-y divide-[var(--color-border)] border-t border-[var(--color-border)] text-[11px]">
      <div className="grid grid-cols-[88px_1fr]">
        <div className="bg-[var(--color-surface-dim)] px-3 py-2 font-bold text-amber-500">掠夺资源</div>
        <div className="px-3 py-2 text-center text-[var(--color-text-secondary)]">{showResources ? formatResources(rewards?.resources || side.resources) : '情报未揭示'}</div>
      </div>
      <div className="grid grid-cols-[88px_1fr]">
        <div className="bg-[var(--color-surface-dim)] px-3 py-2 font-bold text-amber-500">建筑战损</div>
        <div className="px-3 py-2 text-center text-[var(--color-text-secondary)]">{buildingDamage}</div>
      </div>
      <div className="grid grid-cols-[88px_1fr]">
        <div className="bg-[var(--color-surface-dim)] px-3 py-2 font-bold text-amber-500">战损反馈</div>
        <div className="px-3 py-2 text-center text-[var(--color-text-secondary)]">{formatRewardFeedback(rewards, feedback)}</div>
      </div>
      <div className="grid grid-cols-[88px_1fr]">
        <div className="bg-[var(--color-surface-dim)] px-3 py-2 font-bold text-amber-500">宝物掉落</div>
        <div className="px-3 py-2 text-center text-[var(--color-text-secondary)]">{formatDrops(rewards)}</div>
      </div>
    </div>
    </section>
  )
}

export default BattleParticipantBlock
