/* 本文件实现战报详情页，兼容 NPC、PVP 和副本战斗的标准战报展示。 */
import { type FC, useState } from 'react'
import { ArrowLeft, Share2, Check } from 'lucide-react'
import { useGameStore } from '@/store/gameStore'
import { useConfigStore } from '@/store/configStore'
import type { BattleReport, BattleReportSweepDefender, BattleReportSweepExtra, BattleReportUnit } from '@/types/game'
import { formatTraitOutcomeDetail, getTraitMeta } from '@/utils/traits'
import { sortUnitEntries, sortUnitIds } from '@/utils/unitOrder'
import { gameApi } from '@/api/game'
import { mergeBattleReportDrops } from '@/utils/reportDrops'
import { buildReportShareURL } from '../reportPresentation'

interface BattleReportDetailProps {
  report: BattleReport
  onBack: () => void
}

const RESOURCE_LABELS: Record<string, string> = { wood: '木材', stone: '石料', iron: '铁矿', food: '粮食' }
const RESOURCE_ICONS: Record<string, string> = { wood: '🪵', stone: '🪨', iron: '💎', food: '🌾' }
const RESOURCE_ORDER = ['wood', 'stone', 'iron', 'food']
const TYPE_LABELS: Record<string, string> = { attack: '攻击', plunder: '掠夺', scout: '侦查', reinforce: '增援' }
const DROP_QUALITY_CLASS: Record<string, string> = {
  common: 'text-slate-600 bg-slate-500/10 border-slate-500/20',
  rare: 'text-blue-600 bg-blue-500/10 border-blue-500/20',
  epic: 'text-purple-600 bg-purple-500/10 border-purple-500/20',
  legendary: 'text-amber-600 bg-amber-500/10 border-amber-500/25',
}

// safeMap 兼容旧战报或异常空字段，避免详情页读取 null。
function safeMap(value?: Record<string, number> | null): Record<string, number> {
  return value ?? {}
}

// safeArray 兼容旧战报或异常空字段，避免详情页对 null 调用 map。
function safeArray<T>(value?: T[] | null): T[] {
  return Array.isArray(value) ? value : []
}

// isHiddenReportUnit 判断战报中不展示的非战斗兵种。
function isHiddenReportUnit(unitType: string, units?: Record<string, Record<string, { name?: string; role?: string }>>): boolean {
  for (const factionUnits of Object.values(units ?? {})) {
    const config = factionUnits[unitType]
    if (config?.role === 'transport' || config?.name?.includes('商人')) return true
  }
  return unitType.toLowerCase().includes('merchant')
}

// readSweepExtra 读取扫荡聚合战报的扩展摘要，兼容旧数据没有 extra 的情况。
function readSweepExtra(report: BattleReport): BattleReportSweepExtra | null {
  if (report.battleType !== 'sweep' && report.detail?.battleType !== 'sweep') return null
  const raw = report.detail?.extra?.sweep
  if (!raw || typeof raw !== 'object') return {
    requested: 0,
    success: 0,
    failed: 0,
    stopped: false,
    mode: report.type,
  }
  return raw as BattleReportSweepExtra
}

// unitMapFromSideUnits 把标准战报兵种快照转成旧版表格可直接消费的数量 map。
function unitMapFromSideUnits(sideUnits: BattleReportUnit[], key: 'amountBefore' | 'lost'): Record<string, number> {
  return sideUnits.reduce<Record<string, number>>((acc, unit) => {
    acc[unit.unitType] = unit[key] ?? 0
    return acc
  }, {})
}

const BattleReportDetail: FC<BattleReportDetailProps> = ({ report, onBack }) => {
  const faction = useGameStore((s) => s.state?.player.faction) || report.playerFaction || ''
  const units = useConfigStore((s) => s.units)
  const factionUnits = units?.[faction] ?? {}
  const [copied, setCopied] = useState(false)

  const isVictory = report.result === 'attacker_victory'
  const isDraw = report.result === 'draw'
  const isDefenseView = report.viewType === 'defense' || report.type === 'defense'
  const selfDisplayName = report.playerName || useGameStore.getState().state?.player.nickname || '主公'
  const targetDisplayName = report.targetName || report.targetId

  const handleShare = async () => {
    let token = report.share?.token || report.detail?.share?.token
    if (!token) {
      const activePlayerId = useGameStore.getState().activePlayerId
      if (activePlayerId) {
        const link = await gameApi.shareReport(activePlayerId, report.id)
        token = link.token
      }
    }
    const url = buildReportShareURL(window.location.origin, report, token)
    navigator.clipboard.writeText(url)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  // 全兵种列表（进攻方）— 用阵营配置获取完整列表
  const allUnitIds = Object.keys(factionUnits).length > 0
    ? Object.keys(factionUnits)
    : Object.keys(report.dispatchedUnits ?? {})
  const visibleAllUnitIds = sortUnitIds(allUnitIds, faction, units ?? undefined).filter((unitType) => !isHiddenReportUnit(unitType, units ?? undefined))

  // 防守方阵营兵种
  const defenderFaction = report.defenderFaction || ''
  const defenderFactionUnits = units?.[defenderFaction] ?? {}
  const defenderAllUnitIds = Object.keys(defenderFactionUnits).length > 0
    ? Object.keys(defenderFactionUnits)
    : Object.keys(report.defenderUnits ?? {})
  const visibleDefenderAllUnitIds = sortUnitIds(defenderAllUnitIds, defenderFaction, units ?? undefined).filter((unitType) => !isHiddenReportUnit(unitType, units ?? undefined))

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

  const formatTroopMap = (troops?: Record<string, number>) => {
    const text = sortUnitEntries(troops, faction, units ?? undefined)
      .filter(([, amount]) => amount > 0)
      .map(([unitType, amount]) => `${getUnitName(unitType)} ${amount.toLocaleString()}`)
      .join('、')
    return text || '无'
  }

  const formatPvpGenerals = (generals?: Array<{ id: string; name?: string; level?: number }>) => {
    const text = (generals ?? [])
      .map((general) => `${general.name || general.id}${general.level ? ` Lv.${general.level}` : ''}`)
      .join('、')
    return text || '无'
  }

  const pvpPointEntries = Object.entries(safeMap(report.pvpPointsDelta)).filter(([, amount]) => amount !== 0)
  const pvpAttackerGenerals = safeArray(report.pvpAttackerGenerals)
  const pvpDefenderGenerals = safeArray(report.pvpDefenderGenerals)
  const pvpReinforcements = safeArray(report.pvpReinforcements)
  const pvpReinforcementLosses = report.pvpReinforcementLosses ?? {}
  const hasPvpGenerals = pvpAttackerGenerals.length > 0 || pvpDefenderGenerals.length > 0
  const dispatchedUnits = safeMap(report.dispatchedUnits)
  const lostUnits = safeMap(report.lostUnits)
  const rewards = safeMap(report.rewards)
  const defenderUnits = safeMap(report.defenderUnits)
  const defenderLostUnits = safeMap(report.defenderLostUnits)
  const capturedUnits = safeMap(report.capturedUnits)
  const capturedToGarrison = safeMap(report.capturedToGarrison)
  const revivedUnits = safeMap(report.revivedUnits)
  const traitTriggered = safeArray(report.traitTriggered)
  const drops = report.drops ?? report.detail?.rewards?.drops ?? []
  const mergedDrops = mergeBattleReportDrops(drops)
  const topDisplayName = isDefenseView ? targetDisplayName : selfDisplayName
  const bottomDisplayName = isDefenseView ? selfDisplayName : targetDisplayName
  const topPower = isDefenseView ? report.enemyPower : report.playerPower
  const bottomPower = isDefenseView ? report.playerPower : report.enemyPower
  const topUnitIds = isDefenseView ? visibleDefenderAllUnitIds : visibleAllUnitIds
  const bottomUnitIds = isDefenseView ? visibleAllUnitIds : visibleDefenderAllUnitIds
  const topUnits = isDefenseView ? defenderUnits : dispatchedUnits
  const topLostUnits = isDefenseView ? defenderLostUnits : lostUnits
  const bottomUnits = isDefenseView ? dispatchedUnits : defenderUnits
  const bottomLostUnits = isDefenseView ? lostUnits : defenderLostUnits
  const bottomRevealed = isDefenseView ? true : report.defenderRevealed
  const topGeneralText = hasPvpGenerals ? formatPvpGenerals(pvpAttackerGenerals) : '无'
  const bottomGeneralText = hasPvpGenerals ? formatPvpGenerals(pvpDefenderGenerals) : '无'
  const reportGeneralExp = report.generalExpGained ?? report.detail?.rewards?.generalExp ?? 0
  const sweepExtra = readSweepExtra(report)
  const isSweepReport = Boolean(sweepExtra)
  const sweepDefenders = safeArray<BattleReportSweepDefender>(sweepExtra?.defenders)
  const generalLevelBefore = report.generalLevelBefore ?? report.detail?.rewards?.generalLevelBefore
  const generalLevelAfter = report.generalLevelAfter ?? report.detail?.rewards?.generalLevelAfter
  const generalExpText = reportGeneralExp > 0
    ? `+${reportGeneralExp}${generalLevelAfter && generalLevelBefore && generalLevelAfter > generalLevelBefore
      ? ` Lv.${generalLevelBefore} → Lv.${generalLevelAfter}`
      : ''}`
    : '—'
  const topGeneralExpText = isDefenseView ? '—' : generalExpText
  const bottomGeneralExpText = isDefenseView ? generalExpText : '—'

  const renderDefenseCard = (params: {
    keyValue: string
    displayName: string
    factionText?: string
    power: number
    unitIds: string[]
    unitsMap: Record<string, number>
    lostMap: Record<string, number>
    revealed: boolean
    resources?: Record<string, number>
    unitName: (unitType: string) => string
  }) => (
    <div key={params.keyValue} className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] overflow-hidden">
      <div className="px-4 py-2 border-b border-[var(--color-border)] bg-blue-500/5">
        <span className="text-xs font-bold text-blue-600">🛡 防守方 — {params.displayName}</span>
        {params.factionText && <span className="text-[10px] text-[var(--color-text-muted)] ml-2">{params.factionText}</span>}
        <span className="text-[10px] text-[var(--color-text-muted)] ml-2">战力 {params.power.toLocaleString()}</span>
      </div>

      <div className="px-4 py-2 border-b border-[var(--color-border)]">
        <div className="flex items-center">
          <span className="text-xs text-[var(--color-text-secondary)] w-12 flex-shrink-0">将领</span>
          <span className="text-[10px] text-[var(--color-text-muted)] flex-1 text-center">{bottomGeneralText}</span>
        </div>
        <div className="flex items-center mt-1">
          <span className="text-xs text-[var(--color-text-secondary)] w-12 flex-shrink-0">经验</span>
          <span className="text-[10px] text-[var(--color-text-muted)] flex-1 text-center">{bottomGeneralExpText}</span>
        </div>
      </div>

      <div className="px-4 py-2 border-b border-[var(--color-border)]">
        {params.revealed && params.unitIds.length > 0 ? (
          <>
            <div className="hidden sm:block overflow-x-auto">
              <table className="w-full text-center text-[10px]">
                <thead>
                  <tr className="text-[var(--color-text-muted)]">
                    <td className="py-1 text-left font-medium">兵种</td>
                    {params.unitIds.map((uid) => (
                      <td key={uid} className="py-1 px-1 min-w-[40px]">
                        <span className="text-[9px]">{params.unitName(uid)}</span>
                      </td>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td className="py-1 text-left font-medium text-[var(--color-text-secondary)]">驻守</td>
                    {params.unitIds.map((uid) => {
                      const count = params.unitsMap[uid] ?? 0
                      return (
                        <td key={uid} className={`py-1 px-1 ${count > 0 ? 'font-bold text-[var(--color-text-primary)]' : 'text-[var(--color-text-muted)]'}`}>
                          {count}
                        </td>
                      )
                    })}
                  </tr>
                  <tr>
                    <td className="py-1 text-left font-medium text-red-500">阵亡</td>
                    {params.unitIds.map((uid) => {
                      const lost = params.lostMap[uid] ?? 0
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
            <div className="sm:hidden space-y-1.5">
              {params.unitIds
                .filter((uid) => (params.unitsMap[uid] ?? 0) > 0)
                .map((uid) => {
                  const count = params.unitsMap[uid] ?? 0
                  const lost = params.lostMap[uid] ?? 0
                  return (
                    <div key={uid} className="flex items-center justify-between px-2 py-1.5 rounded-lg bg-[var(--color-surface-dim)] border border-[var(--color-border)]">
                      <span className="text-[10px] font-medium text-[var(--color-text-primary)]">{params.unitName(uid)}</span>
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

      <div className="px-4 py-2 border-b border-[var(--color-border)]">
        <div className="flex items-center">
          <span className="text-xs font-medium text-[var(--color-text-secondary)] w-16 flex-shrink-0">剩余资源</span>
          <div className="flex flex-wrap items-center justify-center gap-3 flex-1">
            {RESOURCE_ORDER.map((res) => (
              <span key={res} className="inline-flex items-center gap-1 text-[10px] text-[var(--color-text-secondary)]">
                {RESOURCE_ICONS[res]} {RESOURCE_LABELS[res]} {(params.resources?.[res] ?? 0).toLocaleString()}
              </span>
            ))}
          </div>
        </div>
      </div>

      <div className="px-4 py-2 border-b border-[var(--color-border)]">
        <div className="flex items-center">
          <span className="text-xs font-medium text-[var(--color-text-secondary)] w-16 flex-shrink-0">建筑损坏</span>
          <span className="text-[10px] text-[var(--color-text-muted)] flex-1 text-center">无</span>
        </div>
      </div>

      <div className="px-4 py-2 border-b border-[var(--color-border)]">
        <div className="flex items-center">
          <span className="text-xs font-medium text-[var(--color-text-secondary)] w-16 flex-shrink-0">战损反馈</span>
          <span className="text-[10px] text-[var(--color-text-muted)] flex-1 text-center">—</span>
        </div>
      </div>

      <div className="px-4 py-2 border-b border-[var(--color-border)]">
        <div className="flex items-center">
          <span className="text-xs font-medium text-[var(--color-text-secondary)] w-16 flex-shrink-0">宝物掉落</span>
          <span className="text-[10px] text-[var(--color-text-muted)] flex-1 text-center">无</span>
        </div>
      </div>

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
  )

  const renderSweepDefenseCard = (defender: BattleReportSweepDefender) => {
    const sideUnits = safeArray(defender.units)
    const unitIds = sortUnitIds(sideUnits.map((unit) => unit.unitType), defender.faction, units ?? undefined).filter((unitType) => !isHiddenReportUnit(unitType, units ?? undefined))
    const unitsMap = unitMapFromSideUnits(sideUnits, 'amountBefore')
    const lostMap = unitMapFromSideUnits(sideUnits, 'lost')
    const unitNames = new Map(sideUnits.map((unit) => [unit.unitType, unit.unitName || getDefenderUnitName(unit.unitType)]))
    return renderDefenseCard({
      keyValue: defender.targetId,
      displayName: defender.targetName || defender.targetId,
      factionText: defender.factionLabel || defender.faction,
      power: defender.power,
      unitIds,
      unitsMap,
      lostMap,
      revealed: defender.defenderRevealed,
      resources: defender.resources,
      unitName: (unitType) => unitNames.get(unitType) || getDefenderUnitName(unitType),
    })
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
          {isSweepReport ? 'NPC 扫荡' : `${topDisplayName} ${TYPE_LABELS[report.type] ?? '攻击'} ${bottomDisplayName}`}
        </h2>
        <p className="text-[10px] text-[var(--color-text-muted)] mt-1">
          {new Date(report.createdAt).toLocaleString('zh-CN')}
        </p>
      </div>

      {/* 进攻方 */}
      <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] overflow-hidden">
        <div className="px-4 py-2 border-b border-[var(--color-border)] bg-red-500/5">
          <span className="text-xs font-bold text-red-600">⚔ 进攻方</span>
          <span className="text-[10px] text-[var(--color-text-muted)] ml-2">战力 {topPower.toLocaleString()}</span>
        </div>

        {/* 将领 & 经验 */}
        <div className="px-4 py-2 border-b border-[var(--color-border)]">
          <div className="flex items-center">
            <span className="text-xs text-[var(--color-text-secondary)] w-12 flex-shrink-0">将领</span>
            <span className="text-xs font-semibold text-[var(--color-text-primary)] flex-1 text-center">
              {topGeneralText}
            </span>
          </div>
          <div className="flex items-center mt-1">
            <span className="text-xs text-[var(--color-text-secondary)] w-12 flex-shrink-0">经验</span>
            <span className="text-[10px] text-[var(--color-text-muted)] flex-1 text-center">
              {topGeneralExpText}
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
                  {topUnitIds.map((uid) => (
                    <td key={uid} className="py-1 px-1 min-w-[40px]">
                      <span className="text-[9px]">{getUnitName(uid)}</span>
                    </td>
                  ))}
                </tr>
              </thead>
              <tbody>
                <tr>
                  <td className="py-1 text-left font-medium text-[var(--color-text-secondary)]">出动</td>
                  {topUnitIds.map((uid) => {
                    const dispatched = topUnits[uid] ?? 0
                    return (
                      <td key={uid} className={`py-1 px-1 ${dispatched > 0 ? 'font-bold text-[var(--color-text-primary)]' : 'text-[var(--color-text-muted)]'}`}>
                        {dispatched}
                      </td>
                    )
                  })}
                </tr>
                <tr>
                  <td className="py-1 text-left font-medium text-red-500">阵亡</td>
                  {topUnitIds.map((uid) => {
                    const lost = topLostUnits[uid] ?? 0
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
            {topUnitIds
              .filter((uid) => (topUnits[uid] ?? 0) > 0)
              .map((uid) => {
                const dispatched = topUnits[uid] ?? 0
                const lost = topLostUnits[uid] ?? 0
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
            {topUnitIds.filter((uid) => (topUnits[uid] ?? 0) > 0).length === 0 && (
              <span className="text-[10px] text-[var(--color-text-muted)]">无出动兵种</span>
            )}
          </div>
        </div>

        {/* 掠夺资源 */}
        <div className="px-4 py-2 border-b border-[var(--color-border)]">
          <div className="flex items-center">
            <span className="text-xs font-medium text-[var(--color-text-secondary)] w-16 flex-shrink-0">掠夺资源</span>
            <div className="flex flex-wrap items-center justify-center gap-3 flex-1">
              {RESOURCE_ORDER.filter((res) => (rewards[res] ?? 0) > 0).length > 0 ? (
                RESOURCE_ORDER.filter((res) => (rewards[res] ?? 0) > 0).map((res) => (
                  <span key={res} className="inline-flex items-center gap-1 text-[10px] text-amber-500 font-semibold">
                    {RESOURCE_ICONS[res]} {RESOURCE_LABELS[res]} {rewards[res].toLocaleString()}
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
            <div className="flex flex-1 flex-wrap items-center justify-center gap-1.5">
              {mergedDrops.length > 0 ? (
                mergedDrops.map((drop) => (
                  <span
                    key={`${drop.itemId ?? drop.name ?? 'drop'}-${drop.quality ?? ''}`}
                    className={`rounded-lg border px-2 py-1 text-[10px] font-bold ${DROP_QUALITY_CLASS[drop.quality ?? ''] ?? DROP_QUALITY_CLASS.common}`}
                  >
                    {drop.name ?? drop.itemId} ×{drop.amount.toLocaleString()}
                  </span>
                ))
              ) : (
                <span className="text-[10px] text-[var(--color-text-muted)]">无</span>
              )}
            </div>
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
        traitTriggered.length > 0 ||
        Object.keys(capturedUnits).length > 0 ||
        Object.keys(capturedToGarrison).length > 0 ||
        Object.keys(revivedUnits).length > 0
      ) && (
        <div className="rounded-2xl border border-amber-400/40 bg-amber-400/5 overflow-hidden">
          <div className="px-4 py-2 border-b border-amber-400/30 bg-amber-400/10">
            <span className="text-xs font-bold text-amber-600">将领特性结果</span>
          </div>
          <div className="p-4 space-y-3">
            {traitTriggered.length > 0 && (
              <div className="space-y-1.5">
                {traitTriggered.map((traitId) => {
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
                                {formatTraitOutcomeDetail(k, v, { faction, units: units ?? undefined, sortUnitEntries })}
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

            {Object.keys(capturedUnits).length > 0 && (
              <div>
                <div className="text-[11px] font-semibold text-pink-500 mb-1.5">美人计·俘虏归队</div>
                <div className="flex flex-wrap gap-1.5">
                  {sortUnitEntries(capturedUnits, faction, units ?? undefined).filter(([, v]) => v > 0).map(([unitType, count]) => (
                    <span key={unitType} className="text-[10px] px-2 py-1 rounded-lg bg-pink-500/10 text-pink-600 font-medium">
                      {getUnitName(unitType)} +{count}
                    </span>
                  ))}
                </div>
              </div>
            )}

            {Object.keys(capturedToGarrison).length > 0 && (
              <div>
                <div className="text-[11px] font-semibold text-pink-500 mb-1.5">美人计·俘虏驻防</div>
                <div className="flex flex-wrap gap-1.5">
                  {sortUnitEntries(capturedToGarrison, faction, units ?? undefined).filter(([, v]) => v > 0).map(([unitType, count]) => (
                    <span key={unitType} className="text-[10px] px-2 py-1 rounded-lg bg-pink-500/10 text-pink-600 font-medium">
                      {getUnitName(unitType)} +{count}
                    </span>
                  ))}
                </div>
              </div>
            )}

            {Object.keys(revivedUnits).length > 0 && (
              <div>
                <div className="text-[11px] font-semibold text-emerald-500 mb-1.5">仁德·复活归队</div>
                <div className="flex flex-wrap gap-1.5">
                  {sortUnitEntries(revivedUnits, faction, units ?? undefined).filter(([, v]) => v > 0).map(([unitType, count]) => (
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

      {(pvpPointEntries.length > 0 || hasPvpGenerals || pvpReinforcements.length > 0 || Object.keys(pvpReinforcementLosses).length > 0) && (
        <div className="rounded-2xl border border-indigo-400/40 bg-indigo-400/5 overflow-hidden">
          <div className="px-4 py-2 border-b border-indigo-400/30 bg-indigo-400/10">
            <span className="text-xs font-bold text-indigo-600">参战信息</span>
          </div>
          <div className="p-4 space-y-3">
            {pvpPointEntries.length > 0 && (
              <div>
                <div className="text-[11px] font-semibold text-indigo-600 mb-1.5">积分变化</div>
                <div className="flex flex-wrap gap-1.5">
                  {pvpPointEntries.map(([key, amount]) => (
                    <span key={key} className={`text-[10px] px-2 py-1 rounded-lg font-medium ${amount > 0 ? 'bg-emerald-500/10 text-emerald-600' : 'bg-red-500/10 text-red-600'}`}>
                      {key === 'self' ? '我方' : key === 'target' ? '对方' : key} {amount > 0 ? '+' : ''}{amount}
                    </span>
                  ))}
                </div>
              </div>
            )}

            {hasPvpGenerals && (
              <div>
                <div className="text-[11px] font-semibold text-indigo-600 mb-1.5">参战武将</div>
                <div className="grid gap-1.5 sm:grid-cols-2">
                  <div className="rounded-xl border border-indigo-400/20 bg-[var(--color-surface)] px-3 py-2">
                    <div className="text-[10px] font-bold text-indigo-600">攻击方</div>
                    <div className="mt-1 text-[10px] text-[var(--color-text-secondary)]">{formatPvpGenerals(pvpAttackerGenerals)}</div>
                  </div>
                  <div className="rounded-xl border border-indigo-400/20 bg-[var(--color-surface)] px-3 py-2">
                    <div className="text-[10px] font-bold text-indigo-600">防守方</div>
                    <div className="mt-1 text-[10px] text-[var(--color-text-secondary)]">{formatPvpGenerals(pvpDefenderGenerals)}</div>
                  </div>
                </div>
              </div>
            )}

            {pvpReinforcements.length > 0 && (
              <div>
                <div className="text-[11px] font-semibold text-indigo-600 mb-1.5">参战驻防/援军</div>
                <div className="space-y-1.5">
                  {pvpReinforcements.map((item) => (
                    <div key={item.reinforcementId} className="rounded-xl border border-indigo-400/20 bg-[var(--color-surface)] px-3 py-2">
                      <div className="flex items-center gap-2">
                        <span className="text-[10px] font-bold text-indigo-600">{item.sourceTags?.source_type === 'obtained' ? '获得驻防' : '增援'}</span>
                        <span className="text-[10px] text-[var(--color-text-muted)] truncate">{item.fromPlayerId}</span>
                      </div>
                      <div className="mt-1 text-[10px] text-[var(--color-text-secondary)]">{formatTroopMap(item.troops)}</div>
                      {item.generals && item.generals.length > 0 && (
                        <div className="mt-1 text-[10px] text-[var(--color-text-muted)]">
                          武将：{item.generals.map((general) => general.name || general.id).join('、')}
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            )}

            {Object.keys(pvpReinforcementLosses).length > 0 && (
              <div>
                <div className="text-[11px] font-semibold text-indigo-600 mb-1.5">援军损耗</div>
                <div className="space-y-1.5">
                  {Object.entries(pvpReinforcementLosses).map(([reinforcementId, losses]) => (
                    <div key={reinforcementId} className="flex items-center justify-between gap-2 rounded-lg bg-red-500/10 px-2.5 py-1.5">
                      <span className="min-w-0 truncate text-[10px] text-red-600">{reinforcementId}</span>
                      <span className="shrink-0 text-[10px] font-semibold text-red-600">{formatTroopMap(losses)}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {isSweepReport && sweepDefenders.length > 0
        ? sweepDefenders.map(renderSweepDefenseCard)
        : renderDefenseCard({
          keyValue: 'defender',
          displayName: bottomDisplayName,
          power: bottomPower,
          unitIds: bottomUnitIds,
          unitsMap: bottomUnits,
          lostMap: bottomLostUnits,
          revealed: bottomRevealed,
          resources: report.defenderResources,
          unitName: getDefenderUnitName,
        })}
    </div>
  )
}

export default BattleReportDetail
