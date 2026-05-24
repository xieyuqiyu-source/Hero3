import { type FC } from 'react'
import { ArrowLeft } from 'lucide-react'
import { useGameStore } from '@/store/gameStore'
import { useConfigStore } from '@/store/configStore'
import type { BattleReport } from '@/types/game'

interface BattleReportDetailProps {
  report: BattleReport
  onBack: () => void
}

const RESOURCE_LABELS: Record<string, string> = { wood: '木材', stone: '石料', iron: '铁矿', food: '粮食' }
const RESOURCE_ICONS: Record<string, string> = { wood: '🪵', stone: '🪨', iron: '💎', food: '🌾' }
const RESOURCE_ORDER = ['wood', 'stone', 'iron', 'food']
const TYPE_LABELS: Record<string, string> = { attack: '攻击', plunder: '掠夺', scout: '侦查', reinforce: '增援' }

const BattleReportDetail: FC<BattleReportDetailProps> = ({ report, onBack }) => {
  const faction = useGameStore((s) => s.state?.player.faction ?? 'wei')
  const general = useGameStore((s) => s.state?.general)
  const units = useConfigStore((s) => s.units)
  const factionUnits = units?.[faction] ?? {}

  const isVictory = report.result === 'attacker_victory'
  const isDraw = report.result === 'draw'
  const targetDisplayName = report.targetName || report.targetId

  // 全兵种列表（进攻方）
  const allUnitIds = Object.keys(factionUnits)

  // 防守方阵营兵种
  const defenderFaction = report.defenderFaction || ''
  const defenderFactionUnits = units?.[defenderFaction] ?? {}
  const defenderAllUnitIds = Object.keys(defenderFactionUnits)

  const getUnitName = (unitType: string): string => {
    return factionUnits[unitType]?.name ?? defenderFactionUnits[unitType]?.name ?? unitType
  }

  const getDefenderUnitName = (unitType: string): string => {
    return defenderFactionUnits[unitType]?.name ?? unitType
  }

  return (
    <div className="space-y-4">
      {/* Back button */}
      <button
        type="button"
        onClick={onBack}
        className="flex items-center gap-1.5 text-xs text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] cursor-pointer transition-colors"
      >
        <ArrowLeft size={14} />
        返回列表
      </button>

      {/* Title */}
      <div className={`text-center py-3 rounded-xl ${isVictory ? 'bg-green-500/10' : isDraw ? 'bg-slate-500/10' : 'bg-red-500/10'}`}>
        <h2 className={`text-base font-bold ${isVictory ? 'text-green-600' : isDraw ? 'text-slate-500' : 'text-red-600'}`}>
          {general?.name ?? '主公'} {TYPE_LABELS[report.type] ?? '攻击'} {targetDisplayName}
        </h2>
        <p className="text-[10px] text-[var(--color-text-muted)] mt-1">
          {new Date(report.createdAt).toLocaleString('zh-CN')}
        </p>
      </div>

      {/* 进攻方 */}
      <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] overflow-hidden">
        <div className="px-4 py-2 border-b border-[var(--color-border)] bg-red-500/5">
          <span className="text-xs font-bold text-red-600">⚔ 进攻方</span>
          <span className="text-[10px] text-[var(--color-text-muted)] ml-2">战力 {report.playerPower.toLocaleString()}</span>
        </div>

        {/* 将领 & 经验 */}
        <div className="px-4 py-2 border-b border-[var(--color-border)]">
          <div className="flex items-center">
            <span className="text-xs text-[var(--color-text-secondary)] w-12 flex-shrink-0">将领</span>
            <span className="text-xs font-semibold text-[var(--color-text-primary)] flex-1 text-center">
              {general?.name ?? '—'} {general ? `Lv.${general.level}` : ''}
            </span>
          </div>
          <div className="flex items-center mt-1">
            <span className="text-xs text-[var(--color-text-secondary)] w-12 flex-shrink-0">经验</span>
            <span className="text-[10px] text-[var(--color-text-muted)] flex-1 text-center">—</span>
          </div>
        </div>

        {/* 兵种表格 */}
        <div className="px-4 py-2 border-b border-[var(--color-border)] overflow-x-auto">
          <table className="w-full text-center text-[10px]">
            <thead>
              <tr className="text-[var(--color-text-muted)]">
                <td className="py-1 text-left font-medium">兵种</td>
                {allUnitIds.map((uid) => (
                  <td key={uid} className="py-1 px-1 min-w-[40px]">
                    <span className="text-[9px]">{getUnitName(uid)}</span>
                  </td>
                ))}
              </tr>
            </thead>
            <tbody>
              <tr>
                <td className="py-1 text-left font-medium text-[var(--color-text-secondary)]">出动</td>
                {allUnitIds.map((uid) => {
                  // 优先用 dispatchedUnits，旧战报 fallback 到 lostUnits
                  const dispatched = report.dispatchedUnits?.[uid] ?? report.lostUnits?.[uid] ?? 0
                  return (
                    <td key={uid} className={`py-1 px-1 ${dispatched > 0 ? 'font-bold text-[var(--color-text-primary)]' : 'text-[var(--color-text-muted)]'}`}>
                      {dispatched}
                    </td>
                  )
                })}
              </tr>
              <tr>
                <td className="py-1 text-left font-medium text-red-500">阵亡</td>
                {allUnitIds.map((uid) => {
                  const lost = report.lostUnits?.[uid] ?? 0
                  return (
                    <td key={uid} className={`py-1 px-1 ${lost > 0 ? 'font-bold text-red-600' : 'text-[var(--color-text-muted)]'}`}>
                      {lost}
                    </td>
                  )
                })}
              </tr>
            </tbody>
          </table>
        </div>

        {/* 掠夺资源 */}
        <div className="px-4 py-2 border-b border-[var(--color-border)]">
          <div className="flex items-center">
            <span className="text-xs font-medium text-[var(--color-text-secondary)] w-16 flex-shrink-0">掠夺资源</span>
            <div className="flex flex-wrap items-center justify-center gap-3 flex-1">
              {RESOURCE_ORDER.filter((res) => (report.rewards?.[res] ?? 0) > 0).length > 0 ? (
                RESOURCE_ORDER.filter((res) => (report.rewards?.[res] ?? 0) > 0).map((res) => (
                  <span key={res} className="inline-flex items-center gap-1 text-[10px] text-amber-500 font-semibold">
                    {RESOURCE_ICONS[res]} {RESOURCE_LABELS[res]} {report.rewards[res].toLocaleString()}
                  </span>
                ))
              ) : (
                <span className="text-[10px] text-[var(--color-text-muted)]">无</span>
              )}
            </div>
          </div>
        </div>

        {/* 建筑损坏 */}
        <div className="px-4 py-2 border-b border-[var(--color-border)]">
          <div className="flex items-center">
            <span className="text-xs font-medium text-[var(--color-text-secondary)] w-16 flex-shrink-0">建筑损坏</span>
            <span className="text-[10px] text-[var(--color-text-muted)] flex-1 text-center">无</span>
          </div>
        </div>

        {/* 战损反馈 */}
        <div className="px-4 py-2 border-b border-[var(--color-border)]">
          <div className="flex items-center">
            <span className="text-xs font-medium text-[var(--color-text-secondary)] w-16 flex-shrink-0">战损反馈</span>
            <span className="text-[10px] text-[var(--color-text-muted)] flex-1 text-center">—</span>
          </div>
        </div>

        {/* 宝物掉落 */}
        <div className="px-4 py-2 border-b border-[var(--color-border)]">
          <div className="flex items-center">
            <span className="text-xs font-medium text-[var(--color-text-secondary)] w-16 flex-shrink-0">宝物掉落</span>
            <span className="text-[10px] text-[var(--color-text-muted)] flex-1 text-center">无</span>
          </div>
        </div>

        {/* 词条加成 & 战术卡 */}
        <div className="px-4 py-2">
          <div className="flex items-center">
            <span className="text-xs font-medium text-[var(--color-text-secondary)] w-16 flex-shrink-0">词条/战术</span>
            <div className="flex flex-wrap items-center justify-center gap-1.5 flex-1">
              <span className="text-[9px] px-2 py-0.5 rounded bg-slate-100 dark:bg-slate-800 text-[var(--color-text-muted)]">词条加成</span>
              <span className="text-[9px] px-2 py-0.5 rounded bg-slate-100 dark:bg-slate-800 text-[var(--color-text-muted)]">战术卡</span>
            </div>
          </div>
        </div>
      </div>

      {/* 防守方 */}
      <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] overflow-hidden">
        <div className="px-4 py-2 border-b border-[var(--color-border)] bg-blue-500/5">
          <span className="text-xs font-bold text-blue-600">🛡 防守方 — {targetDisplayName}</span>
          <span className="text-[10px] text-[var(--color-text-muted)] ml-2">战力 {report.enemyPower.toLocaleString()}</span>
        </div>

        {/* 将领 & 经验 */}
        <div className="px-4 py-2 border-b border-[var(--color-border)]">
          <div className="flex items-center">
            <span className="text-xs text-[var(--color-text-secondary)] w-12 flex-shrink-0">将领</span>
            <span className="text-[10px] text-[var(--color-text-muted)] flex-1 text-center">无</span>
          </div>
          <div className="flex items-center mt-1">
            <span className="text-xs text-[var(--color-text-secondary)] w-12 flex-shrink-0">经验</span>
            <span className="text-[10px] text-[var(--color-text-muted)] flex-1 text-center">—</span>
          </div>
        </div>

        {/* 防守方兵种 */}
        <div className="px-4 py-2 border-b border-[var(--color-border)] overflow-x-auto">
          {report.defenderRevealed && defenderAllUnitIds.length > 0 ? (
            <table className="w-full text-center text-[10px]">
              <thead>
                <tr className="text-[var(--color-text-muted)]">
                  <td className="py-1 text-left font-medium">兵种</td>
                  {defenderAllUnitIds.map((uid) => (
                    <td key={uid} className="py-1 px-1 min-w-[40px]">
                      <span className="text-[9px]">{getDefenderUnitName(uid)}</span>
                    </td>
                  ))}
                </tr>
              </thead>
              <tbody>
                <tr>
                  <td className="py-1 text-left font-medium text-[var(--color-text-secondary)]">驻守</td>
                  {defenderAllUnitIds.map((uid) => {
                    const count = report.defenderUnits?.[uid] ?? 0
                    return (
                      <td key={uid} className={`py-1 px-1 ${count > 0 ? 'font-bold text-[var(--color-text-primary)]' : 'text-[var(--color-text-muted)]'}`}>
                        {count}
                      </td>
                    )
                  })}
                </tr>
                <tr>
                  <td className="py-1 text-left font-medium text-red-500">阵亡</td>
                  {defenderAllUnitIds.map((uid) => {
                    const lost = report.defenderLostUnits?.[uid] ?? 0
                    return (
                      <td key={uid} className={`py-1 px-1 ${lost > 0 ? 'font-bold text-red-600' : 'text-[var(--color-text-muted)]'}`}>
                        {lost}
                      </td>
                    )
                  })}
                </tr>
              </tbody>
            </table>
          ) : (
            <div className="flex items-center justify-center py-2">
              <span className="text-[11px] text-amber-600 font-medium">对方战损低于25%，无法显示对方详细兵力情报</span>
            </div>
          )}
        </div>

        {/* 防守方剩余资源 */}
        <div className="px-4 py-2 border-b border-[var(--color-border)]">
          <div className="flex items-center">
            <span className="text-xs font-medium text-[var(--color-text-secondary)] w-16 flex-shrink-0">剩余资源</span>
            <div className="flex flex-wrap items-center justify-center gap-3 flex-1">
              {RESOURCE_ORDER.map((res) => (
                <span key={res} className="inline-flex items-center gap-1 text-[10px] text-[var(--color-text-secondary)]">
                  {RESOURCE_ICONS[res]} {RESOURCE_LABELS[res]} {(report.defenderResources?.[res] ?? 0).toLocaleString()}
                </span>
              ))}
            </div>
          </div>
        </div>

        {/* 建筑损坏 */}
        <div className="px-4 py-2 border-b border-[var(--color-border)]">
          <div className="flex items-center">
            <span className="text-xs font-medium text-[var(--color-text-secondary)] w-16 flex-shrink-0">建筑损坏</span>
            <span className="text-[10px] text-[var(--color-text-muted)] flex-1 text-center">无</span>
          </div>
        </div>

        {/* 战损反馈 */}
        <div className="px-4 py-2 border-b border-[var(--color-border)]">
          <div className="flex items-center">
            <span className="text-xs font-medium text-[var(--color-text-secondary)] w-16 flex-shrink-0">战损反馈</span>
            <span className="text-[10px] text-[var(--color-text-muted)] flex-1 text-center">—</span>
          </div>
        </div>

        {/* 宝物掉落 */}
        <div className="px-4 py-2 border-b border-[var(--color-border)]">
          <div className="flex items-center">
            <span className="text-xs font-medium text-[var(--color-text-secondary)] w-16 flex-shrink-0">宝物掉落</span>
            <span className="text-[10px] text-[var(--color-text-muted)] flex-1 text-center">无</span>
          </div>
        </div>

        {/* 词条加成 & 战术卡 */}
        <div className="px-4 py-2">
          <div className="flex items-center">
            <span className="text-xs font-medium text-[var(--color-text-secondary)] w-16 flex-shrink-0">词条/战术</span>
            <div className="flex flex-wrap items-center justify-center gap-1.5 flex-1">
              <span className="text-[9px] px-2 py-0.5 rounded bg-slate-100 dark:bg-slate-800 text-[var(--color-text-muted)]">词条加成</span>
              <span className="text-[9px] px-2 py-0.5 rounded bg-slate-100 dark:bg-slate-800 text-[var(--color-text-muted)]">战术卡</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

export default BattleReportDetail
