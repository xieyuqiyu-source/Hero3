import { type FC, useState } from 'react'
import { ArrowLeft, Share2, Check } from 'lucide-react'
import { useGameStore } from '@/store/gameStore'
import { useConfigStore } from '@/store/configStore'
import type { BattleReport } from '@/types/game'
import { getTraitMeta } from '@/utils/traits'

interface BattleReportDetailProps {
  report: BattleReport
  onBack: () => void
}

const RESOURCE_LABELS: Record<string, string> = { wood: '木材', stone: '石料', iron: '铁矿', food: '粮食' }
const RESOURCE_ICONS: Record<string, string> = { wood: '🪵', stone: '🪨', iron: '💎', food: '🌾' }
const RESOURCE_ORDER = ['wood', 'stone', 'iron', 'food']
const TYPE_LABELS: Record<string, string> = { attack: '攻击', plunder: '掠夺', scout: '侦查', reinforce: '增援' }

const BattleReportDetail: FC<BattleReportDetailProps> = ({ report, onBack }) => {
  const faction = useGameStore((s) => s.state?.player.faction) || report.playerFaction || ''
  const general = useGameStore((s) => s.state?.general)
  const units = useConfigStore((s) => s.units)
  const factionUnits = units?.[faction] ?? {}
  const [copied, setCopied] = useState(false)

  const isVictory = report.result === 'attacker_victory'
  const isDraw = report.result === 'draw'
  const targetDisplayName = report.targetName || report.targetId

  const handleShare = () => {
    const url = `${window.location.origin}/report/${report.id}`
    navigator.clipboard.writeText(url)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  // 全兵种列表（进攻方）— 用阵营配置获取完整列表
  const allUnitIds = Object.keys(factionUnits).length > 0
    ? Object.keys(factionUnits)
    : Object.keys(report.dispatchedUnits ?? {})

  // 防守方阵营兵种
  const defenderFaction = report.defenderFaction || ''
  const defenderFactionUnits = units?.[defenderFaction] ?? {}
  const defenderAllUnitIds = Object.keys(defenderFactionUnits).length > 0
    ? Object.keys(defenderFactionUnits)
    : Object.keys(report.defenderUnits ?? {})

  const getUnitName = (unitType: string): string => {
    // 尝试从所有阵营配置里找名字
    for (const f of Object.values(units ?? {})) {
      if (f[unitType]?.name) return f[unitType].name
    }
    return unitType
  }

  const getDefenderUnitName = (unitType: string): string => {
    if (defenderFactionUnits[unitType]?.name) return defenderFactionUnits[unitType].name
    for (const f of Object.values(units ?? {})) {
      if (f[unitType]?.name) return f[unitType].name
    }
    return unitType
  }

  const formatOutcomeDetail = (key: string, value: number | string | Record<string, number>): string => {
    const labels: Record<string, string> = {
      totalCaptured: '俘虏',
      totalRevived: '复活',
      extraDamage: '额外伤害',
      damagePercent: '伤害比例',
      foodRatio: '口粮比',
      triggerChance: '触发概率',
      suppressRate: '震慑比例',
      totalSuppressed: '震慑兵力',
      suppressedUnits: '震慑明细',
    }
    const label = labels[key] ?? key
    if (typeof value === 'number') {
      if (key.endsWith('Percent') || key.endsWith('Rate') || key.endsWith('Chance')) {
        return `${label}: ${Math.round(value * 100)}%`
      }
      return `${label}: ${value.toLocaleString()}`
    }
    if (typeof value === 'object' && value !== null) {
      const text = Object.entries(value)
        .filter(([, amount]) => amount > 0)
        .map(([unitType, amount]) => `${getUnitName(unitType)} ${amount.toLocaleString()}`)
        .join('、')
      return `${label}: ${text || '无'}`
    }
    return `${label}: ${value}`
  }

  return (
    <div className="space-y-4">
      {/* Back button + Share */}
      <div className="flex items-center justify-between">
        <button
          type="button"
          onClick={onBack}
          className="flex items-center gap-1.5 text-xs text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] cursor-pointer transition-colors"
        >
          <ArrowLeft size={14} />
          返回列表
        </button>
        <button
          type="button"
          onClick={handleShare}
          className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-[10px] font-medium text-blue-500 bg-blue-500/10 hover:bg-blue-500/20 cursor-pointer transition-colors"
        >
          {copied ? <Check size={12} /> : <Share2 size={12} />}
          {copied ? '已复制' : '分享'}
        </button>
      </div>

      {/* Title */}
      <div className={`text-center py-3 rounded-xl ${isVictory ? 'bg-green-500/10' : isDraw ? 'bg-slate-500/10' : 'bg-red-500/10'}`}>
        <h2 className={`text-base font-bold ${isVictory ? 'text-green-600' : isDraw ? 'text-slate-500' : 'text-red-600'}`}>
          {report.playerName || useGameStore.getState().state?.player.nickname || '主公'} {TYPE_LABELS[report.type] ?? '攻击'} {targetDisplayName}
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
            <span className="text-[10px] text-[var(--color-text-muted)] flex-1 text-center">
              {(report.generalExpGained ?? 0) > 0
                ? `+${report.generalExpGained}${report.generalLevelAfter && report.generalLevelBefore && report.generalLevelAfter > report.generalLevelBefore
                  ? ` Lv.${report.generalLevelBefore} → Lv.${report.generalLevelAfter}`
                  : ''}`
                : '—'}
            </span>
          </div>
        </div>

        {/* 兵种表格 - 桌面横向表格 / 手机竖向列表 */}
        <div className="px-4 py-2 border-b border-[var(--color-border)]">
          {/* Desktop: horizontal table */}
          <div className="hidden sm:block overflow-x-auto">
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
                    const dispatched = report.dispatchedUnits?.[uid] ?? 0
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
          {/* Mobile: vertical list, only show units with dispatched > 0 */}
          <div className="sm:hidden space-y-1.5">
            {allUnitIds
              .filter((uid) => (report.dispatchedUnits?.[uid] ?? 0) > 0)
              .map((uid) => {
                const dispatched = report.dispatchedUnits?.[uid] ?? 0
                const lost = report.lostUnits?.[uid] ?? 0
                return (
                  <div key={uid} className="flex items-center justify-between px-2 py-1.5 rounded-lg bg-[var(--color-surface-dim)] border border-[var(--color-border)]">
                    <span className="text-[10px] font-medium text-[var(--color-text-primary)]">{getUnitName(uid)}</span>
                    <div className="flex items-center gap-3">
                      <span className="text-[10px] text-[var(--color-text-secondary)]">出动 <span className="font-bold">{dispatched}</span></span>
                      {lost > 0 && <span className="text-[10px] text-red-600">阵亡 <span className="font-bold">{lost}</span></span>}
                    </div>
                  </div>
                )
              })}
            {allUnitIds.filter((uid) => (report.dispatchedUnits?.[uid] ?? 0) > 0).length === 0 && (
              <span className="text-[10px] text-[var(--color-text-muted)]">无出动兵种</span>
            )}
          </div>
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

      {/* 将领特性结果 */}
      {(
        (report.traitTriggered && report.traitTriggered.length > 0) ||
        (report.capturedUnits && Object.keys(report.capturedUnits).length > 0) ||
        (report.capturedToGarrison && Object.keys(report.capturedToGarrison).length > 0) ||
        (report.revivedUnits && Object.keys(report.revivedUnits).length > 0)
      ) && (
        <div className="rounded-2xl border border-amber-400/40 bg-amber-400/5 overflow-hidden">
          <div className="px-4 py-2 border-b border-amber-400/30 bg-amber-400/10">
            <span className="text-xs font-bold text-amber-600">将领特性结果</span>
          </div>
          <div className="p-4 space-y-3">
            {report.traitTriggered && report.traitTriggered.length > 0 && (
              <div className="space-y-1.5">
                {report.traitTriggered.map((traitId) => {
                  const meta = getTraitMeta(traitId)
                  const outcome = report.traitOutcomes?.[traitId]
                  return (
                    <div key={traitId} className="flex items-start gap-2 px-2.5 py-2 rounded-xl bg-amber-500/10 border border-amber-500/30">
                      <span className="text-base">{meta.icon}</span>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 flex-wrap">
                          <span className="text-xs font-bold text-amber-600">{meta.name}</span>
                          <span className="text-[10px] text-amber-600/70">{meta.trigger}</span>
                        </div>
                        {outcome?.detail && (
                          <div className="mt-0.5 text-[10px] text-[var(--color-text-secondary)]">
                            {Object.entries(outcome.detail).map(([k, v]) => (
                              <span key={k} className="mr-2">
                                {formatOutcomeDetail(k, v)}
                              </span>
                            ))}
                          </div>
                        )}
                      </div>
                    </div>
                  )
                })}
              </div>
            )}

            {report.capturedUnits && Object.keys(report.capturedUnits).length > 0 && (
              <div>
                <div className="text-[11px] font-semibold text-pink-500 mb-1.5">美人计·俘虏归队</div>
                <div className="flex flex-wrap gap-1.5">
                  {Object.entries(report.capturedUnits).filter(([, v]) => v > 0).map(([unitType, count]) => (
                    <span key={unitType} className="text-[10px] px-2 py-1 rounded-lg bg-pink-500/10 text-pink-600 font-medium">
                      {getUnitName(unitType)} +{count}
                    </span>
                  ))}
                </div>
              </div>
            )}

            {report.capturedToGarrison && Object.keys(report.capturedToGarrison).length > 0 && (
              <div>
                <div className="text-[11px] font-semibold text-pink-500 mb-1.5">美人计·俘虏驻防</div>
                <div className="flex flex-wrap gap-1.5">
                  {Object.entries(report.capturedToGarrison).filter(([, v]) => v > 0).map(([unitType, count]) => (
                    <span key={unitType} className="text-[10px] px-2 py-1 rounded-lg bg-pink-500/10 text-pink-600 font-medium">
                      {getUnitName(unitType)} +{count}
                    </span>
                  ))}
                </div>
              </div>
            )}

            {report.revivedUnits && Object.keys(report.revivedUnits).length > 0 && (
              <div>
                <div className="text-[11px] font-semibold text-emerald-500 mb-1.5">仁德·复活归队</div>
                <div className="flex flex-wrap gap-1.5">
                  {Object.entries(report.revivedUnits).filter(([, v]) => v > 0).map(([unitType, count]) => (
                    <span key={unitType} className="text-[10px] px-2 py-1 rounded-lg bg-emerald-500/10 text-emerald-600 font-medium">
                      {getUnitName(unitType)} +{count}
                    </span>
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>
      )}

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
        <div className="px-4 py-2 border-b border-[var(--color-border)]">
          {report.defenderRevealed && defenderAllUnitIds.length > 0 ? (
            <>
              {/* Desktop: horizontal table */}
              <div className="hidden sm:block overflow-x-auto">
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
              </div>
              {/* Mobile: vertical list, only show units with count > 0 */}
              <div className="sm:hidden space-y-1.5">
                {defenderAllUnitIds
                  .filter((uid) => (report.defenderUnits?.[uid] ?? 0) > 0)
                  .map((uid) => {
                    const count = report.defenderUnits?.[uid] ?? 0
                    const lost = report.defenderLostUnits?.[uid] ?? 0
                    return (
                      <div key={uid} className="flex items-center justify-between px-2 py-1.5 rounded-lg bg-[var(--color-surface-dim)] border border-[var(--color-border)]">
                        <span className="text-[10px] font-medium text-[var(--color-text-primary)]">{getDefenderUnitName(uid)}</span>
                        <div className="flex items-center gap-3">
                          <span className="text-[10px] text-[var(--color-text-secondary)]">驻守 <span className="font-bold">{count}</span></span>
                          {lost > 0 && <span className="text-[10px] text-red-600">阵亡 <span className="font-bold">{lost}</span></span>}
                        </div>
                      </div>
                    )
                  })}
              </div>
            </>
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
